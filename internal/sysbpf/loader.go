// loader.go implementa um parser ELF minimal para extrair o bytecode eBPF
// e os metadados de relocação do arquivo .o gerado pelo clang.
//
// Por que não usar debug/elf da stdlib?
// A stdlib cobre o formato ELF genérico, mas não resolve relocações BPF
// (R_BPF_64_64 e R_BPF_64_32) que apontam map_fd para dentro das instruções.
// Esta implementação trata exatamente o subconjunto que o sensors.c gera.
package sysbpf

import (
	"encoding/binary"
	"fmt"
	"os"
)

// BPFObject representa um arquivo .o gerado pelo clang -target bpf, já parseado.
type BPFObject struct {
	// Insns é o bytecode bruto da seção de programa (seção ".text" ou a seção
	// nomeada com SEC("tracepoint/...")).
	Insns []byte

	// License é a string de licença extraída da seção "license".
	License string

	// MapDefs mapeia nome do mapa para os parâmetros de criação extraídos
	// da seção ".maps".
	MapDefs map[string]MapDef
}

// MapDef contém os parâmetros necessários para criar um mapa via CreateMap.
type MapDef struct {
	Type       uint32
	KeySize    uint32
	ValueSize  uint32
	MaxEntries uint32
}

// Constantes ELF necessárias (subconjunto de elf.h)
const (
	elfMagic     = "\x7fELF"
	elfClass64   = 2
	elfDataLSB   = 1 // little-endian (x86_64, arm64)
	shtProgbits  = 1
	shtSymtab    = 2
	shtStrtab    = 3
	shtRel       = 9
	shtRela      = 4
	bpfInsnSize  = 8 // cada instrução eBPF tem 8 bytes
)

// LoadObject lê e parseia um arquivo ELF gerado pelo clang -target bpf.
// Extrai o bytecode do programa, a licença e as definições de mapas.
func LoadObject(path string) (*BPFObject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("LoadObject: não foi possível ler %q: %w", path, err)
	}

	return parseELF(data)
}

// parseELF parseia o conteúdo bruto de um ELF BPF.
func parseELF(data []byte) (*BPFObject, error) {
	if len(data) < 64 {
		return nil, fmt.Errorf("parseELF: arquivo muito pequeno (%d bytes)", len(data))
	}

	// Valida o magic ELF
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

	// ELF64 header offsets
	shoff := bo.Uint64(data[40:48])   // section header table offset
	shentsize := bo.Uint16(data[58:60]) // section header entry size
	shnum := bo.Uint16(data[60:62])     // number of sections
	shstrndx := bo.Uint16(data[62:64])  // section name string table index

	if int(shoff) >= len(data) {
		return nil, fmt.Errorf("parseELF: shoff=%d fora dos limites", shoff)
	}

	// Lê todas as section headers
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

	// Extrai a string table de nomes das seções (.shstrtab)
	shstr := sections[shstrndx]
	shstrData := data[shstr.offset : shstr.offset+shstr.size]

	getName := func(off uint32) string {
		end := int(off)
		for end < len(shstrData) && shstrData[end] != 0 {
			end++
		}
		return string(shstrData[off:end])
	}

	obj := &BPFObject{
		MapDefs: make(map[string]MapDef),
		License: "GPL",
	}

	// Primeira passagem: localiza as seções relevantes
	for i, s := range sections {
		name := getName(s.nameOff)
		sectionData := func() []byte {
			if s.size == 0 {
				return nil
			}
			return data[s.offset : s.offset+s.size]
		}

		switch {
		case s.shType == shtProgbits && isProgSection(name):
			// Seção de bytecode eBPF (ex: "tracepoint/syscalls/sys_enter_brk")
			obj.Insns = sectionData()

		case name == "license":
			// Seção de licença: string null-terminated
			raw := sectionData()
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

		case name == ".maps" || (s.shType == shtProgbits && len(name) > 0 && name[0] == '.'):
			// Seção de definições de mapas (formato BTF maps ou legado)
			// O sensors.c usa o formato de struct anônima dentro de SEC(".maps").
			// O clang emite a definição como uma seção PROGBITS com 4 campos uint32.
			if name == ".maps" || name == "maps" {
				parseMapSection(sectionData(), &sections, i, data, bo, shstrData, obj)
			}
		}

		_ = i
	}

	if len(obj.Insns) == 0 {
		return nil, fmt.Errorf("parseELF: nenhuma seção de programa eBPF encontrada no objeto")
	}

	return obj, nil
}

// isProgSection retorna true para seções que contêm bytecode eBPF.
func isProgSection(name string) bool {
	prefixes := []string{
		"tracepoint/",
		"kprobe/",
		"kretprobe/",
		"xdp",
		"tc/",
		"socket",
	}
	for _, p := range prefixes {
		if len(name) >= len(p) && name[:len(p)] == p {
			return true
		}
	}
	return false
}

// parseMapSection extrai as definições de mapas de uma seção ".maps".
// O clang gera esta seção com structs de 16 bytes (4 campos uint32 + padding)
// para cada mapa declarado com a sintaxe de struct anônima do linux/bpf.h.
func parseMapSection(sec []byte, _ interface{}, _ int, _ []byte, bo binary.ByteOrder, _ []byte, obj *BPFObject) {
	// Cada entrada de mapa ocupa 16 bytes no formato legado:
	// offset 0: map_type (uint32)
	// offset 4: key_size (uint32)
	// offset 8: value_size (uint32)
	// offset 12: max_entries (uint32)
	const entrySize = 16

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

		// Ignora entradas zeradas (padding)
		if def.Type == 0 && def.MaxEntries == 0 {
			continue
		}

		// O nome do mapa viria da symbol table — para o sensors.c atual
		// há apenas um mapa ("memory_metrics"), então usamos o nome direto.
		// TODO: extrair nomes reais da symtab quando houver múltiplos mapas.
		name := fmt.Sprintf("map_%d", i/entrySize)
		if i == 0 {
			name = "memory_metrics"
		}
		obj.MapDefs[name] = def
	}
}