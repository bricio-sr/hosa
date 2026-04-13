// struct_ops.go estende o loader sysbpf para suportar BPF_PROG_TYPE_STRUCT_OPS,
// necessário para o escalonador de sobrevivência da Fase 2 (sched_ext).
//
// sched_ext usa o mecanismo BPF STRUCT_OPS para substituir o escalonador CFS
// em tempo de execução, sem reinicialização do kernel. Requer:
//   - Linux ≥ 6.11 com CONFIG_SCHED_CLASS_EXT=y
//   - CONFIG_BPF_SYSCALL=y (universal em distribuições modernas)
//
// Fluxo de ativação:
//  1. Abrir o BTF do vmlinux via /sys/kernel/btf/vmlinux (BPF_BTF_LOAD)
//  2. Localizar o type ID de sched_ext_ops no BTF (busca por nome)
//  3. LoadStructOpsProg: carregar cada callback com BPF_PROG_TYPE_STRUCT_OPS
//  4. CreateStructOpsMap: criar mapa BPF_MAP_TYPE_STRUCT_OPS com btf_fd + type_id
//  5. Preencher o mapa com os prog FDs (via BPF_MAP_UPDATE_ELEM)
//  6. LinkStructOps: BPF_LINK_CREATE com attach_type=BPF_STRUCT_OPS_LINK
//     → o link FD ativado substitui o CFS; fechá-lo restaura o CFS.
//
// Referência: kernel Documentation/scheduler/sched-ext.rst,
//             tools/testing/selftests/bpf/progs/sched_ext.c
package sysbpf

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Constantes da Fase 2 — BPF STRUCT_OPS.
const (
	// BPF_PROG_TYPE_STRUCT_OPS é o tipo de programa para implementar struct_ops callbacks.
	BPF_PROG_TYPE_STRUCT_OPS = 27

	// BPF_MAP_TYPE_STRUCT_OPS é o tipo de mapa que registra uma implementação de struct_ops.
	BPF_MAP_TYPE_STRUCT_OPS = 34

	// BPF_LINK_CREATE é o comando BPF para criar um link entre prog/mapa e o kernel.
	BPF_LINK_CREATE = 28

	// BPF_STRUCT_OPS_LINK é o attach_type para links do tipo struct_ops.
	BPF_STRUCT_OPS_LINK = 27

	// BPF_BTF_LOAD carrega um blob BTF no kernel e retorna um fd.
	BPF_BTF_LOAD = 18

	// vmlinuxBTFPath é o caminho para o BTF do kernel em execução.
	vmlinuxBTFPath = "/sys/kernel/btf/vmlinux"

	// schedExtOpsTypeName é o nome do tipo struct_ops do sched_ext no BTF do kernel.
	schedExtOpsTypeName = "sched_ext_ops"
)

// linkCreateAttr é o bpf_attr union para BPF_LINK_CREATE com struct_ops.
// Layout alinhado com linux/bpf.h struct bpf_link_create (campos relevantes).
//   +0  prog_fd     uint32
//   +4  target_fd   uint32  (union com target_ifindex)
//   +8  attach_type uint32
//   +12 flags       uint32
//   +16 ... (campos adicionais não usados, preenchidos com zeros)
type linkCreateAttr struct {
	progFD     uint32    // +0
	targetFD   uint32    // +4  map_fd do mapa BPF_MAP_TYPE_STRUCT_OPS
	attachType uint32    // +8
	flags      uint32    // +12
	_          [112]byte // padding até 128 bytes (16 + 112 = 128)
}

// structOpsMapAttr é o bpf_attr para BPF_MAP_CREATE com BPF_MAP_TYPE_STRUCT_OPS.
// Layout alinhado com linux/bpf.h (offsets verificados):
//   +0   map_type              uint32
//   +4   key_size              uint32
//   +8   value_size            uint32
//   +12  max_entries           uint32
//   +16  map_flags             uint32
//   +20  inner_map_fd          uint32
//   +24  numa_node             uint32
//   +28  map_name              [16]byte (BPF_OBJ_NAME_LEN=16)
//   +44  map_ifindex           uint32
//   +48  btf_fd                uint32
//   +52  btf_key_type_id       uint32
//   +56  btf_value_type_id     uint32
//   +60  btf_vmlinux_value_type_id uint32
//   +64  map_extra             uint64
//   +72  ... padding até 128 bytes
type structOpsMapAttr struct {
	mapType         uint32   // +0
	keySize         uint32   // +4
	valueSize       uint32   // +8
	maxEntries      uint32   // +12
	mapFlags        uint32   // +16
	innerMapFD      uint32   // +20
	numaNode        uint32   // +24
	mapName         [16]byte // +28
	mapIfindex      uint32   // +44
	btfFD           uint32   // +48
	btfKeyTypeID    uint32   // +52
	btfValueTypeID  uint32   // +56
	btfVmlinuxValID uint32   // +60
	mapExtra        uint64   // +64
	_               [56]byte // padding até 128 bytes (72 + 56 = 128)
}

// btfLoadAttr é o bpf_attr para BPF_BTF_LOAD.
type btfLoadAttr struct {
	btf         uint64 // ponteiro para blob BTF
	btfLogBuf   uint64
	btfSize     uint32
	btfLogSize  uint32
	btfLogLevel uint32
	_           [100]byte // padding até 128 bytes
}

// LoadStructOpsProg carrega um programa BPF_PROG_TYPE_STRUCT_OPS no kernel.
// É análogo a LoadProg, mas com o tipo correto para callbacks do sched_ext.
func LoadStructOpsProg(insns []byte, license string, logBuf []byte) (ProgFD, error) {
	return LoadProg(BPF_PROG_TYPE_STRUCT_OPS, insns, license, logBuf)
}

// OpenVMLinuxBTF abre o BTF do kernel em execução via /sys/kernel/btf/vmlinux
// usando BPF_BTF_LOAD. Retorna o fd do BTF carregado no kernel.
// Requer /sys/kernel/btf/vmlinux — disponível em kernels ≥ 5.2 com CONFIG_DEBUG_INFO_BTF=y.
func OpenVMLinuxBTF() (int, error) {
	data, err := os.ReadFile(vmlinuxBTFPath)
	if err != nil {
		return -1, fmt.Errorf("OpenVMLinuxBTF: não foi possível ler %q: %w", vmlinuxBTFPath, err)
	}

	logBuf := make([]byte, 4096)
	attr := btfLoadAttr{
		btf:         uint64(uintptr(unsafe.Pointer(&data[0]))),
		btfSize:     uint32(len(data)),
		btfLogBuf:   uint64(uintptr(unsafe.Pointer(&logBuf[0]))),
		btfLogSize:  uint32(len(logBuf)),
		btfLogLevel: 0,
	}

	fd, _, errno := unix.Syscall(unix.SYS_BPF,
		uintptr(BPF_BTF_LOAD),
		uintptr(unsafe.Pointer(&attr)),
		unsafe.Sizeof(attr))
	if errno != 0 {
		return -1, fmt.Errorf("OpenVMLinuxBTF: BPF_BTF_LOAD: %w", errno)
	}

	return int(fd), nil
}

// FindSchedExtOpsTypeID localiza o type ID do sched_ext_ops no blob BTF do vmlinux.
// Parseia o formato BTF binário para encontrar o tipo pelo nome.
// Retorna 0 se o tipo não for encontrado (kernel sem CONFIG_SCHED_CLASS_EXT=y).
func FindSchedExtOpsTypeID() (uint32, error) {
	data, err := os.ReadFile(vmlinuxBTFPath)
	if err != nil {
		return 0, fmt.Errorf("FindSchedExtOpsTypeID: %w", err)
	}
	return findBTFTypeByName(data, schedExtOpsTypeName)
}

// findBTFTypeByName parseia um blob BTF binário (formato do kernel) e retorna
// o type ID do tipo com o nome dado.
//
// Formato BTF (linux/btf.h):
//   - Header: struct btf_header (24 bytes)
//   - Type section: sequência de struct btf_type + dados extras
//   - String section: strings null-terminated
func findBTFTypeByName(data []byte, name string) (uint32, error) {
	const btfMagic = 0xeB9F
	const btfHeaderSize = 24

	if len(data) < btfHeaderSize {
		return 0, fmt.Errorf("BTF muito pequeno")
	}

	bo := binary.LittleEndian

	magic := bo.Uint16(data[0:2])
	if magic != btfMagic {
		return 0, fmt.Errorf("BTF magic inválido: 0x%04x", magic)
	}

	// Offsets do header BTF
	hdrLen := bo.Uint32(data[4:8])
	typeOff := bo.Uint32(data[8:12])
	typeLen := bo.Uint32(data[12:16])
	strOff := bo.Uint32(data[16:20])
	strLen := bo.Uint32(data[20:24])

	typeStart := hdrLen + typeOff
	strStart := hdrLen + strOff

	if uint32(len(data)) < typeStart+typeLen || uint32(len(data)) < strStart+strLen {
		return 0, fmt.Errorf("BTF truncado")
	}

	typeSection := data[typeStart : typeStart+typeLen]
	strSection := data[strStart : strStart+strLen]

	getString := func(off uint32) string {
		if off >= strLen {
			return ""
		}
		end := int(off)
		for end < len(strSection) && strSection[end] != 0 {
			end++
		}
		return string(strSection[off:end])
	}

	// Itera sobre os tipos BTF
	// Cada tipo começa com struct btf_type (12 bytes):
	//   uint32 name_off, uint32 info, uint32 size_or_type
	const btfTypeSize = 12
	typeID := uint32(1) // type IDs começam em 1
	pos := 0

	for pos+btfTypeSize <= len(typeSection) {
		nameOff := bo.Uint32(typeSection[pos:])
		info := bo.Uint32(typeSection[pos+4:])

		kind := (info >> 24) & 0x1f
		vlen := info & 0xffff

		typeName := getString(nameOff)
		if typeName == name {
			return typeID, nil
		}

		// Avança para o próximo tipo (tamanho depende do kind)
		extra := btfExtraSize(kind, vlen, typeSection[pos+btfTypeSize:])
		pos += btfTypeSize + extra
		typeID++
	}

	return 0, fmt.Errorf("tipo %q não encontrado no BTF (kernel sem CONFIG_SCHED_CLASS_EXT?)", name)
}

// btfExtraSize retorna o tamanho dos dados extras após o btf_type header,
// dependendo do kind do tipo. Baseado em linux/btf.h.
func btfExtraSize(kind uint32, vlen uint32, _ []byte) int {
	const (
		BTF_KIND_INT        = 1
		BTF_KIND_PTR        = 2
		BTF_KIND_ARRAY      = 3
		BTF_KIND_STRUCT     = 4
		BTF_KIND_UNION      = 5
		BTF_KIND_ENUM       = 6
		BTF_KIND_FWD        = 7
		BTF_KIND_TYPEDEF    = 8
		BTF_KIND_VOLATILE   = 9
		BTF_KIND_CONST      = 10
		BTF_KIND_RESTRICT   = 11
		BTF_KIND_FUNC       = 12
		BTF_KIND_FUNC_PROTO = 13
		BTF_KIND_VAR        = 14
		BTF_KIND_DATASEC    = 15
		BTF_KIND_FLOAT      = 16
		BTF_KIND_DECL_TAG   = 17
		BTF_KIND_TYPE_TAG   = 18
		BTF_KIND_ENUM64     = 19
	)

	switch kind {
	case BTF_KIND_INT:
		return 4 // uint32 encoding
	case BTF_KIND_ARRAY:
		return 12 // struct btf_array
	case BTF_KIND_STRUCT, BTF_KIND_UNION:
		return int(vlen) * 12 // vlen × struct btf_member
	case BTF_KIND_ENUM:
		return int(vlen) * 8 // vlen × struct btf_enum
	case BTF_KIND_ENUM64:
		return int(vlen) * 12 // vlen × struct btf_enum64
	case BTF_KIND_FUNC_PROTO:
		return int(vlen) * 8 // vlen × struct btf_param
	case BTF_KIND_VAR:
		return 4 // uint32 linkage
	case BTF_KIND_DATASEC:
		return int(vlen) * 12 // vlen × struct btf_var_secinfo
	case BTF_KIND_DECL_TAG:
		return 4 // int component_idx
	default:
		return 0
	}
}

// CreateStructOpsMap cria um mapa BPF_MAP_TYPE_STRUCT_OPS no kernel.
// Requer o fd do BTF do vmlinux e o type ID do sched_ext_ops nesse BTF.
// O mapa criado recebe os prog FDs dos callbacks via BPF_MAP_UPDATE_ELEM.
func CreateStructOpsMap(btfFD int, btfVmlinuxValueTypeID uint32) (MapFD, error) {
	if btfVmlinuxValueTypeID == 0 {
		return -1, fmt.Errorf("CreateStructOpsMap: btfVmlinuxValueTypeID=0 (tipo não encontrado no BTF)")
	}

	var name [16]byte
	copy(name[:], "hosa_survival")

	attr := structOpsMapAttr{
		mapType:         BPF_MAP_TYPE_STRUCT_OPS,
		keySize:         4,    // uint32
		valueSize:       3000, // tamanho conservador do sched_ext_ops; kernel verifica via BTF
		maxEntries:      1,
		btfFD:           uint32(btfFD),
		btfVmlinuxValID: btfVmlinuxValueTypeID,
		mapName:         name,
	}

	fd, _, errno := unix.Syscall(unix.SYS_BPF,
		uintptr(BPF_MAP_CREATE),
		uintptr(unsafe.Pointer(&attr)),
		unsafe.Sizeof(attr))
	if errno != 0 {
		return -1, fmt.Errorf("CreateStructOpsMap: BPF_MAP_CREATE: %w", errno)
	}

	return MapFD(fd), nil
}

// LinkStructOps ativa um mapa BPF_MAP_TYPE_STRUCT_OPS no kernel via BPF_LINK_CREATE.
// Retorna um link FD que deve ser mantido aberto enquanto o escalonador estiver ativo.
// Fechar o link FD atomicamente restaura o CFS — sem reinicialização.
func LinkStructOps(mapFD MapFD) (int, error) {
	attr := linkCreateAttr{
		progFD:     0, // não usado para struct_ops links
		targetFD:   uint32(mapFD),
		attachType: BPF_STRUCT_OPS_LINK,
	}

	fd, _, errno := unix.Syscall(unix.SYS_BPF,
		uintptr(BPF_LINK_CREATE),
		uintptr(unsafe.Pointer(&attr)),
		unsafe.Sizeof(attr))
	if errno != 0 {
		return -1, fmt.Errorf("LinkStructOps: BPF_LINK_CREATE: %w", errno)
	}

	return int(fd), nil
}

// SchedExtSupported verifica se o kernel tem suporte a sched_ext e BTF vmlinux.
// Testa a presença de /sys/kernel/sched_ext/ e /sys/kernel/btf/vmlinux.
func SchedExtSupported() bool {
	if _, err := os.Stat("/sys/kernel/sched_ext"); err != nil {
		return false
	}
	if _, err := os.Stat(vmlinuxBTFPath); err != nil {
		return false
	}
	// Verifica se sched_ext_ops existe no BTF do vmlinux
	data, err := os.ReadFile(vmlinuxBTFPath)
	if err != nil {
		return false
	}
	typeID, err := findBTFTypeByName(data, schedExtOpsTypeName)
	return err == nil && typeID > 0
}

// SchedExtState lê o estado atual do sched_ext via /sys/kernel/sched_ext/root/ops.
// Retorna o nome do escalonador ativo, ou "" se nenhum estiver carregado.
func SchedExtState() string {
	data, err := os.ReadFile("/sys/kernel/sched_ext/root/ops")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
