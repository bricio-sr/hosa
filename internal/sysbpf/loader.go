// loader.go implementa um parser ELF minimal para extrair bytecode eBPF,
// definições de mapas e resolver relocações BPF do .o gerado pelo clang.
//
// Suporta múltiplas seções de programa (um .o com N tracepoints),
// necessário para o sensors.c multivariável do HOSA.
package sysbpf

import (
	"encoding/binary"
	"fmt"
	"os"
)

// BPFObject representa um arquivo .o gerado pelo clang -target bpf, já parseado.
type BPFObject struct {
	// License extraída da seção "license".
	License string

	// MapDefs mapeia nome do mapa → parâmetros de criação.
	MapDefs map[string]MapDef

	// InsnsBySection mapeia nome da seção → bytecode bruto (com relocações pendentes).
	// Ex: "tracepoint/sched/sched_wakeup" → []byte{...}
	InsnsBySection map[string][]byte

	// InsnMapRefsBySection mapeia nome da seção → (índice de instrução → nome do mapa).
	// Usado pelo collector para resolver relocações por seção individualmente.
	InsnMapRefsBySection map[string]map[int]string

	// Insns é o bytecode da primeira seção de programa encontrada.
	// Mantido para compatibilidade com o loader single-program anterior.
	Insns []byte

	// insnMapRefs é o mapa de relocações da primeira seção (compatibilidade).
	insnMapRefs map[int]string
}

// MapDefNames retorna os nomes dos mapas encontrados no ELF — útil para diagnóstico.
func (obj *BPFObject) MapDefNames() []string {
	names := make([]string, 0, len(obj.MapDefs))
	for k := range obj.MapDefs {
		names = append(names, k)
	}
	return names
}

// BPFObjectSlice é um wrapper leve para resolver relocações em uma fatia de bytecode
// extraída de uma seção específica do BPFObject.
type BPFObjectSlice struct {
	Insns       []byte
	InsnMapRefs map[int]string
}

// RelocateInsns resolve as relocações BPF nesta fatia, injetando map_fds.
func (s *BPFObjectSlice) RelocateInsns(fds MapFDs) error {
	for insnIdx, mapName := range s.InsnMapRefs {
		fd, ok := fds[mapName]
		if !ok {
			return fmt.Errorf("RelocateInsns: fd não encontrado para mapa %q", mapName)
		}
		byteOffset := insnIdx * bpfInsnSize
		if byteOffset+bpfInsnSize > len(s.Insns) {
			return fmt.Errorf("RelocateInsns: offset %d fora dos limites", byteOffset)
		}
		dstReg := s.Insns[byteOffset+1] & 0x0f
		s.Insns[byteOffset+1] = dstReg | (bpfPseudoMapFD << 4)
		binary.LittleEndian.PutUint32(s.Insns[byteOffset+4:], uint32(fd))
	}
	return nil
}

// RelocateInsns resolve relocações no BPFObject inteiro (seção única — compatibilidade).
func (obj *BPFObject) RelocateInsns(fds MapFDs) error {
	s := &BPFObjectSlice{Insns: obj.Insns, InsnMapRefs: obj.insnMapRefs}
	if err := s.RelocateInsns(fds); err != nil {
		return err
	}
	obj.Insns = s.Insns
	return nil
}

// MapDef contém os parâmetros necessários para criar um mapa via CreateMap.
type MapDef struct {
	Type       uint32
	KeySize    uint32
	ValueSize  uint32
	MaxEntries uint32
}

// MapFDs mapeia nome do mapa → file descriptor retornado por BPF_MAP_CREATE.
type MapFDs map[string]MapFD

// Constantes ELF
const (
	elfMagic    = "\x7fELF"
	elfClass64  = 2
	elfDataLSB  = 1
	shtProgbits = 1
	shtSymtab   = 2
	shtRel      = 9
	bpfInsnSize = 8
)

// Constantes BPF
const (
	bpfLdImmOpcode = 0x18
	bpfPseudoMapFD = 1
)

type sectionHeader struct {
	nameOff uint32
	shType  uint32
	flags   uint64
	addr    uint64
	offset  uint64
	size    uint64
	link    uint32
	info    uint32
	align   uint64
	entsize uint64
}

type relocation struct {
	offset  uint64
	symIdx  uint32
	relType uint32
}

// LoadObject lê e parseia um arquivo ELF gerado pelo clang -target bpf.
func LoadObject(path string) (*BPFObject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("LoadObject: não foi possível ler %q: %w", path, err)
	}
	return parseELF(data)
}

func parseELF(data []byte) (*BPFObject, error) {
	if len(data) < 64 {
		return nil, fmt.Errorf("parseELF: arquivo muito pequeno (%d bytes)", len(data))
	}
	if string(data[:4]) != elfMagic {
		return nil, fmt.Errorf("parseELF: magic inválido")
	}
	if data[4] != elfClass64 {
		return nil, fmt.Errorf("parseELF: apenas ELF64 suportado")
	}
	if data[5] != elfDataLSB {
		return nil, fmt.Errorf("parseELF: apenas little-endian suportado")
	}

	bo := binary.LittleEndian
	shoff := bo.Uint64(data[40:48])
	shentsize := bo.Uint16(data[58:60])
	shnum := bo.Uint16(data[60:62])
	shstrndx := bo.Uint16(data[62:64])

	sections := make([]sectionHeader, shnum)
	for i := 0; i < int(shnum); i++ {
		base := int(shoff) + i*int(shentsize)
		if base+int(shentsize) > len(data) {
			return nil, fmt.Errorf("parseELF: section header %d fora dos limites", i)
		}
		s := &sections[i]
		s.nameOff = bo.Uint32(data[base:])
		s.shType = bo.Uint32(data[base+4:])
		s.flags = bo.Uint64(data[base+8:])
		s.addr = bo.Uint64(data[base+16:])
		s.offset = bo.Uint64(data[base+24:])
		s.size = bo.Uint64(data[base+32:])
		s.link = bo.Uint32(data[base+40:])
		s.info = bo.Uint32(data[base+44:])
		s.align = bo.Uint64(data[base+48:])
		s.entsize = bo.Uint64(data[base+56:])
	}

	shstrtab := rawSection(data, sections[shstrndx])
	getName := func(off uint32) string {
		end := int(off)
		for end < len(shstrtab) && shstrtab[end] != 0 {
			end++
		}
		return string(shstrtab[off:end])
	}

	obj := &BPFObject{
		License:              "GPL",
		MapDefs:              make(map[string]MapDef),
		InsnsBySection:       make(map[string][]byte),
		InsnMapRefsBySection: make(map[string]map[int]string),
		insnMapRefs:          make(map[int]string),
	}

	// Mapeia índice de seção → nome (para resolver relocações)
	sectionNames := make(map[int]string, shnum)
	var symtabIdx int = -1
	var mapsSectionIdx int = -1

	// Primeira passagem: identifica e coleta todas as seções relevantes
	for i, s := range sections {
		name := getName(s.nameOff)
		sectionNames[i] = name
		raw := rawSection(data, s)

		switch {
		case s.shType == shtProgbits && isProgSection(name):
			insns := make([]byte, len(raw))
			copy(insns, raw)
			obj.InsnsBySection[name] = insns
			obj.InsnMapRefsBySection[name] = make(map[int]string)
			// Mantém a primeira seção em Insns para compatibilidade
			if obj.Insns == nil {
				obj.Insns = insns
			}

		case name == "license":
			if len(raw) > 0 {
				end := len(raw)
				for j, b := range raw {
					if b == 0 {
						end = j
						break
					}
				}
				obj.License = string(raw[:end])
			}

		case name == "maps":
			mapsSectionIdx = i
			parseMapSectionLegacy(raw, bo, obj)

		case s.shType == shtSymtab:
			symtabIdx = i
		}
	}

	if len(obj.InsnsBySection) == 0 {
		return nil, fmt.Errorf("parseELF: nenhuma seção de programa eBPF encontrada")
	}

	// Segunda passagem: resolve relocações por seção de programa
	if symtabIdx >= 0 && mapsSectionIdx >= 0 {
		symtab := rawSection(data, sections[symtabIdx])
		strtab := rawSection(data, sections[sections[symtabIdx].link])

		for _, s := range sections {
			if s.shType != shtRel {
				continue
			}
			targetName := sectionNames[int(s.info)]
			if !isProgSection(targetName) {
				continue
			}
			for _, rel := range parseRels(rawSection(data, s), bo) {
				name := symbolName(symtab, strtab, rel.symIdx, bo)
				if name == "" {
					continue
				}
				insnIdx := int(rel.offset) / bpfInsnSize
				obj.InsnMapRefsBySection[targetName][insnIdx] = name
				// Mantém compatibilidade com a primeira seção
				if obj.InsnsBySection[targetName] != nil &&
					sameBuf(obj.InsnsBySection[targetName], obj.Insns) {
					obj.insnMapRefs[insnIdx] = name
				}
			}
		}
	}

	return obj, nil
}

// sameBuf verifica se dois slices apontam para o mesmo backing array.
func sameBuf(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	return &a[0] == &b[0]
}

func parseMapSectionLegacy(sec []byte, bo binary.ByteOrder, obj *BPFObject) {
	const entrySize = 20
	if len(sec) < entrySize {
		return
	}
	for i := 0; i+entrySize <= len(sec); i += entrySize {
		def := MapDef{
			Type:       bo.Uint32(sec[i:]),
			KeySize:    bo.Uint32(sec[i+4:]),
			ValueSize:  bo.Uint32(sec[i+8:]),
			MaxEntries: bo.Uint32(sec[i+12:]),
		}
		if def.Type == 0 && def.MaxEntries == 0 {
			continue
		}
		name := "hosa_metrics"
		if i > 0 {
			name = fmt.Sprintf("map_%d", i/entrySize)
		}
		obj.MapDefs[name] = def
	}
}

func parseRels(data []byte, bo binary.ByteOrder) []relocation {
	const relSize = 16
	rels := make([]relocation, 0, len(data)/relSize)
	for i := 0; i+relSize <= len(data); i += relSize {
		info := bo.Uint64(data[i+8:])
		rels = append(rels, relocation{
			offset:  bo.Uint64(data[i:]),
			symIdx:  uint32(info >> 32),
			relType: uint32(info & 0xffffffff),
		})
	}
	return rels
}

func symbolName(symtab, strtab []byte, idx uint32, bo binary.ByteOrder) string {
	const symSize = 24
	offset := int(idx) * symSize
	if offset+symSize > len(symtab) {
		return ""
	}
	nameOff := bo.Uint32(symtab[offset:])
	end := int(nameOff)
	for end < len(strtab) && strtab[end] != 0 {
		end++
	}
	return string(strtab[nameOff:end])
}

func rawSection(data []byte, s sectionHeader) []byte {
	if s.size == 0 || s.offset == 0 {
		return nil
	}
	end := s.offset + s.size
	if end > uint64(len(data)) {
		return nil
	}
	return data[s.offset:end]
}

func isProgSection(name string) bool {
	prefixes := []string{
		"tracepoint/", "kprobe/", "kretprobe/",
		"xdp", "tc/", "socket",
		"struct_ops/", "struct_ops.link/", // Fase 2: sched_ext survival scheduler
	}
	for _, p := range prefixes {
		if len(name) >= len(p) && name[:len(p)] == p {
			return true
		}
	}
	return false
}