// loader.go implementa um parser ELF minimal para extrair o bytecode eBPF,
// as definições de mapas e resolver as relocações BPF (R_BPF_64_64) do
// arquivo .o gerado pelo clang -target bpf.
//
// O passo crítico que o bpf2go faz e que precisamos fazer à mão:
// após criar os mapas no kernel (BPF_MAP_CREATE), seus file descriptors
// precisam ser injetados nas instruções BPF_LD_IMM64 antes de carregar
// o programa. Sem isso, o verifier rejeita com "expected=map_ptr".
//
// Referências:
//   - linux/bpf.h: BPF_LD_IMM64, BPF_PSEUDO_MAP_FD
//   - ELF64 spec: seção de relocações SHT_REL (tipo 9)
package sysbpf

import (
	"encoding/binary"
	"fmt"
	"os"
)

// BPFObject representa um arquivo .o gerado pelo clang -target bpf, já parseado.
type BPFObject struct {
	// Insns é o bytecode bruto com relocações pendentes.
	// Chame RelocateInsns(fds) antes de passar para LoadProg.
	Insns []byte

	// License é a string de licença extraída da seção "license".
	License string

	// MapDefs mapeia nome do mapa → parâmetros de criação.
	MapDefs map[string]MapDef

	// insnMapRefs mapeia índice de instrução → nome do mapa.
	// Preenchido pelo parseador de relocações, consumido por RelocateInsns.
	insnMapRefs map[int]string
}

// MapDef contém os parâmetros necessários para criar um mapa via CreateMap.
type MapDef struct {
	Type       uint32
	KeySize    uint32
	ValueSize  uint32
	MaxEntries uint32
}

// MapFDs mapeia nome do mapa → file descriptor retornado por BPF_MAP_CREATE.
// Passado para RelocateInsns após criar os mapas no kernel.
type MapFDs map[string]MapFD

// RelocateInsns resolve as relocações BPF injetando os map_fds reais nas
// instruções BPF_LD_IMM64 que referenciam mapas.
// Deve ser chamado após BPF_MAP_CREATE e antes de BPF_PROG_LOAD.
func (obj *BPFObject) RelocateInsns(fds MapFDs) error {
	for insnIdx, mapName := range obj.insnMapRefs {
		fd, ok := fds[mapName]
		if !ok {
			return fmt.Errorf("RelocateInsns: fd não encontrado para mapa %q", mapName)
		}

		byteOffset := insnIdx * bpfInsnSize
		if byteOffset+bpfInsnSize > len(obj.Insns) {
			return fmt.Errorf("RelocateInsns: offset %d fora dos limites do bytecode", byteOffset)
		}

		// Instrução BPF_LD_IMM64 (slot 0 de 2):
		// bytes [0]   = opcode (0x18)
		// bytes [1]   = dst_reg | src_reg (src = BPF_PSEUDO_MAP_FD = 1)
		// bytes [2:4] = off (0)
		// bytes [4:8] = imm_lo = map_fd  ← aqui injetamos o fd
		binary.LittleEndian.PutUint32(obj.Insns[byteOffset+4:], uint32(fd))

		// Slot 1 (bytes [8:16]): opcode=0, imm_hi=0 — já zerado pelo clang
	}
	return nil
}

// Constantes ELF (subconjunto de elf.h)
const (
	elfMagic    = "\x7fELF"
	elfClass64  = 2
	elfDataLSB  = 1
	shtProgbits = 1
	shtSymtab   = 2
	shtRel      = 9
	bpfInsnSize = 8
)

// Constantes BPF para identificar instruções de referência a mapas
const (
	bpfLdImmOpcode = 0x18 // BPF_LD | BPF_IMM | BPF_DW
	bpfPseudoMapFD = 1    // src_reg quando imm é um map_fd
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
		return nil, fmt.Errorf("parseELF: apenas ELF64 suportado (class=%d)", data[4])
	}
	if data[5] != elfDataLSB {
		return nil, fmt.Errorf("parseELF: apenas little-endian suportado")
	}

	bo := binary.LittleEndian
	shoff := bo.Uint64(data[40:48])
	shentsize := bo.Uint16(data[58:60])
	shnum := bo.Uint16(data[60:62])
	shstrndx := bo.Uint16(data[62:64])

	if int(shoff) >= len(data) {
		return nil, fmt.Errorf("parseELF: shoff=%d fora dos limites", shoff)
	}

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
		MapDefs:     make(map[string]MapDef),
		License:     "GPL",
		insnMapRefs: make(map[int]string),
	}

	var progSectionIdx int = -1
	var symtabIdx int = -1
	var mapsSectionIdx int = -1

	// Primeira passagem: coleta seções
	for i, s := range sections {
		name := getName(s.nameOff)
		raw := rawSection(data, s)

		switch {
		case s.shType == shtProgbits && isProgSection(name):
			obj.Insns = make([]byte, len(raw))
			copy(obj.Insns, raw)
			progSectionIdx = i

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

	if len(obj.Insns) == 0 {
		return nil, fmt.Errorf("parseELF: nenhuma seção de programa eBPF encontrada")
	}

	// Segunda passagem: resolve relocações
	if symtabIdx >= 0 && mapsSectionIdx >= 0 && progSectionIdx >= 0 {
		symtab := rawSection(data, sections[symtabIdx])
		strtab := rawSection(data, sections[sections[symtabIdx].link])

		for _, s := range sections {
			if s.shType != shtRel || int(s.info) != progSectionIdx {
				continue
			}
			for _, rel := range parseRels(rawSection(data, s), bo) {
				name := symbolName(symtab, strtab, rel.symIdx, bo)
				if name == "" {
					continue
				}
				// Mapeia o índice da instrução (em slots de 8 bytes) ao mapa
				insnIdx := int(rel.offset) / bpfInsnSize
				obj.insnMapRefs[insnIdx] = name
			}
		}
	}

	return obj, nil
}

// parseMapSectionLegacy lê mapas no formato legado (struct bpf_map_def de 20 bytes).
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
		name := "memory_metrics"
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
	prefixes := []string{"tracepoint/", "kprobe/", "kretprobe/", "xdp", "tc/", "socket"}
	for _, p := range prefixes {
		if len(name) >= len(p) && name[:len(p)] == p {
			return true
		}
	}
	return false
}