// Package sysbpf implementa um wrapper minimal sobre a syscall SYS_BPF do Linux.
// É a camada de baixo nível do HOSA para interação com o subsistema eBPF do kernel,
// sem nenhuma dependência de bibliotecas de terceiros como github.com/cilium/ebpf.
//
// Referência: linux/bpf.h, man 2 bpf
package sysbpf

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Constantes de comandos BPF (bpf_cmd do linux/bpf.h).
const (
	BPF_MAP_CREATE      = 0
	BPF_MAP_LOOKUP_ELEM = 1
	BPF_PROG_LOAD       = 5
)

// Tipos de mapa eBPF (bpf_map_type do linux/bpf.h).
const (
	BPF_MAP_TYPE_ARRAY = 2
)

// Tipos de programa eBPF (bpf_prog_type do linux/bpf.h).
const (
	BPF_PROG_TYPE_TRACEPOINT = 5
)

// MapFD é um file descriptor de um mapa eBPF criado no kernel.
type MapFD int

// ProgFD é um file descriptor de um programa eBPF carregado no kernel.
type ProgFD int

// --- Structs de atributo para a syscall bpf(2) ---
// Cada operação usa um union de atributos. Em Go, representamos cada variante
// como uma struct separada com padding para cobrir o tamanho máximo do union
// (128 bytes conforme linux/bpf.h).

// mapCreateAttr corresponde ao union bpf_attr para BPF_MAP_CREATE.
// Alinhado com a definição do kernel: linux/bpf.h struct { __u32 map_type; ... }
type mapCreateAttr struct {
	mapType    uint32
	keySize    uint32
	valueSize  uint32
	maxEntries uint32
	mapFlags   uint32
	_          [108]byte // padding até 128 bytes
}

// mapLookupAttr corresponde ao union bpf_attr para BPF_MAP_LOOKUP_ELEM.
type mapLookupAttr struct {
	mapFD uint32
	_pad0 [4]byte
	key   uint64
	value uint64
	flags uint64
	_     [96]byte // padding até 128 bytes
}

// progLoadAttr corresponde ao union bpf_attr para BPF_PROG_LOAD.
type progLoadAttr struct {
	progType    uint32
	insnCnt     uint32
	insns       uint64 // ponteiro para array de bpf_insn
	license     uint64 // ponteiro para string de licença
	logLevel    uint32
	logSize     uint32
	logBuf      uint64 // ponteiro para buffer de log do verifier
	kernVersion uint32
	progFlags   uint32
	progName    [16]byte
	_           [60]byte // padding até 128 bytes
}

// bpf é o ponto de entrada para todas as operações — chama a syscall SYS_BPF diretamente.
func bpf(cmd int, attr unsafe.Pointer, size uintptr) (uintptr, error) {
	r, _, errno := unix.Syscall(unix.SYS_BPF, uintptr(cmd), uintptr(attr), size)
	if errno != 0 {
		return 0, fmt.Errorf("syscall bpf(%d): %w", cmd, errno)
	}
	return r, nil
}

// CreateMap cria um mapa eBPF no kernel e retorna seu file descriptor.
//
// mapType: tipo do mapa (ex: BPF_MAP_TYPE_ARRAY)
// keySize: tamanho da chave em bytes (ex: 4 para uint32)
// valueSize: tamanho do valor em bytes (ex: 8 para uint64)
// maxEntries: número máximo de entradas
func CreateMap(mapType uint32, keySize, valueSize, maxEntries uint32) (MapFD, error) {
	attr := mapCreateAttr{
		mapType:    mapType,
		keySize:    keySize,
		valueSize:  valueSize,
		maxEntries: maxEntries,
	}

	fd, err := bpf(BPF_MAP_CREATE, unsafe.Pointer(&attr), unsafe.Sizeof(attr))
	if err != nil {
		return -1, fmt.Errorf("CreateMap: %w", err)
	}

	return MapFD(fd), nil
}

// LookupElem lê um valor do mapa eBPF dado uma chave.
// key e value devem ser ponteiros para variáveis do tipo correto para o mapa.
func LookupElem(fd MapFD, key unsafe.Pointer, value unsafe.Pointer) error {
	attr := mapLookupAttr{
		mapFD: uint32(fd),
		key:   uint64(uintptr(key)),
		value: uint64(uintptr(value)),
	}

	_, err := bpf(BPF_MAP_LOOKUP_ELEM, unsafe.Pointer(&attr), unsafe.Sizeof(attr))
	if err != nil {
		return fmt.Errorf("LookupElem(fd=%d): %w", fd, err)
	}

	return nil
}

// LoadProg carrega um programa eBPF compilado (bytecode ELF já parseado) no kernel.
//
// progType: tipo do programa (ex: BPF_PROG_TYPE_TRACEPOINT)
// insns: slice de instruções eBPF ([]byte com o conteúdo da seção .text do ELF)
// license: string de licença (deve ser "GPL" para tracepoints)
// logBuf: buffer para receber o log do verifier em caso de erro (pode ser nil)
func LoadProg(progType uint32, insns []byte, license string, logBuf []byte) (ProgFD, error) {
	if len(insns) == 0 {
		return -1, fmt.Errorf("LoadProg: bytecode vazio")
	}
	if len(insns)%8 != 0 {
		return -1, fmt.Errorf("LoadProg: bytecode com tamanho inválido (%d bytes, deve ser múltiplo de 8)", len(insns))
	}

	licenseBytes := append([]byte(license), 0) // null-terminated

	attr := progLoadAttr{
		progType: progType,
		insnCnt:  uint32(len(insns) / 8), // número de instruções (cada uma tem 8 bytes)
		insns:    uint64(uintptr(unsafe.Pointer(&insns[0]))),
		license:  uint64(uintptr(unsafe.Pointer(&licenseBytes[0]))),
		logLevel: 0,
	}

	if len(logBuf) > 0 {
		attr.logLevel = 1
		attr.logSize = uint32(len(logBuf))
		attr.logBuf = uint64(uintptr(unsafe.Pointer(&logBuf[0])))
	}

	fd, err := bpf(BPF_PROG_LOAD, unsafe.Pointer(&attr), unsafe.Sizeof(attr))
	if err != nil {
		verifierMsg := ""
		if len(logBuf) > 0 {
			verifierMsg = "\nverifier log:\n" + string(logBuf)
		}
		return -1, fmt.Errorf("LoadProg: %w%s", err, verifierMsg)
	}

	return ProgFD(fd), nil
}

// AttachTracepoint anexa um programa eBPF a um tracepoint do kernel.
// Retorna um file descriptor que deve ser mantido aberto enquanto o programa
// estiver ativo — fechar este fd desanexa o programa automaticamente.
func AttachTracepoint(subsystem, event string, progFD ProgFD) (int, error) {
	// O kernel expõe o ID do tracepoint em dois caminhos possíveis:
	// 1. /sys/kernel/debug/tracing/events/<subsystem>/<event>/id  (debugfs montado)
	// 2. /sys/kernel/tracing/events/<subsystem>/<event>/id        (tracefs direto)
	idPath := fmt.Sprintf("/sys/kernel/debug/tracing/events/%s/%s/id", subsystem, event)
	if _, err := os.Stat(idPath); err != nil {
		// debugfs não montado — tenta o caminho direto do tracefs
		idPath = fmt.Sprintf("/sys/kernel/tracing/events/%s/%s/id", subsystem, event)
	}

	tracepointID, err := readTracepointID(idPath)
	if err != nil {
		return -1, fmt.Errorf("AttachTracepoint: não foi possível ler ID do tracepoint %s/%s.\n"+
			"  Tente: sudo mount -t debugfs none /sys/kernel/debug\n"+
			"  Erro original: %w", subsystem, event, err)
	}

	// perf_event_attr para tracepoint
	// Baseado em: linux/perf_event.h, struct perf_event_attr
	attr := unix.PerfEventAttr{
		Type:        unix.PERF_TYPE_TRACEPOINT,
		Config:      uint64(tracepointID),
		Sample_type: unix.PERF_SAMPLE_RAW,
		Sample:      1,
		Wakeup:      1,
	}

	// Abre o perf event no contexto global (pid=-1, cpu=0 monitora todos os processos)
	efd, err := unix.PerfEventOpen(&attr, -1, 0, -1, unix.PERF_FLAG_FD_CLOEXEC)
	if err != nil {
		return -1, fmt.Errorf("AttachTracepoint: perf_event_open: %w", err)
	}

	// Anexa o programa eBPF ao perf event via ioctl(PERF_EVENT_IOC_SET_BPF)
	if err := unix.IoctlSetInt(efd, unix.PERF_EVENT_IOC_SET_BPF, int(progFD)); err != nil {
		unix.Close(efd)
		return -1, fmt.Errorf("AttachTracepoint: ioctl PERF_EVENT_IOC_SET_BPF: %w", err)
	}

	// Habilita o perf event
	if err := unix.IoctlSetInt(efd, unix.PERF_EVENT_IOC_ENABLE, 0); err != nil {
		unix.Close(efd)
		return -1, fmt.Errorf("AttachTracepoint: ioctl PERF_EVENT_IOC_ENABLE: %w", err)
	}

	return efd, nil
}

// Close fecha um file descriptor retornado por AttachTracepoint, CreateMap ou LoadProg.
// Fechar o fd do perf event desanexa o programa eBPF do kernel automaticamente.
func Close(fd int) error {
	return unix.Close(fd)
}