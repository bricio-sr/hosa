// Package sensor implementa a camada de coleta de métricas do HOSA via eBPF.
// Esta implementação não utiliza bibliotecas de terceiros — toda a interação
// com o kernel é feita através do pacote internal/sysbpf.
package sensor

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"unsafe"

	"github.com/bricio-sr/hosa/internal/sysbpf"
	"golang.org/x/sys/unix"
)

const (
	// bpfObjectPath é o caminho para o bytecode compilado pelo clang.
	// Gerado via: clang -O2 -target bpf -c internal/bpf/sensors.c -o internal/bpf/sensors.o
	bpfObjectPath = "internal/bpf/sensors.o"

	// mapName é o nome do mapa eBPF definido em sensors.c.
	mapName = "memory_metrics"

	// tracepointSubsystem e tracepointEvent identificam o ponto de captura no kernel.
	tracepointSubsystem = "syscalls"
	tracepointEvent     = "sys_enter_brk"

	// verifierLogSize é o tamanho do buffer para o log do verifier eBPF.
	// Usado apenas em caso de erro no carregamento do programa.
	verifierLogSize = 64 * 1024 // 64 KB
)

// Collector é o sensor eBPF do HOSA.
// Ele carrega o programa sensors.o no kernel, o anexa ao tracepoint
// sys_enter_brk e disponibiliza a leitura das métricas coletadas.
type Collector struct {
	mapFD   sysbpf.MapFD  // file descriptor do mapa eBPF (compartilhado kernel↔userspace)
	progFD  sysbpf.ProgFD // file descriptor do programa eBPF carregado
	eventFD int            // file descriptor do perf event (mantê-lo aberto = programa ativo)
}

// Start inicializa o sensor eBPF em 4 passos:
//  1. Parseia o ELF do sensors.o
//  2. Cria o mapa no kernel (BPF_MAP_CREATE)
//  3. Carrega o programa no kernel (BPF_PROG_LOAD)
//  4. Anexa ao tracepoint sys_enter_brk (perf_event_open + ioctl)
func (c *Collector) Start() error {
	objPath, err := resolveObjectPath(bpfObjectPath)
	if err != nil {
		return fmt.Errorf("sensor.Start: %w", err)
	}

	// Passo 1 — Parseia o ELF
	obj, err := sysbpf.LoadObject(objPath)
	if err != nil {
		return fmt.Errorf("sensor.Start: falha ao parsear ELF %q: %w", objPath, err)
	}

	// Passo 2 — Cria o mapa no kernel
	mapDef, ok := obj.MapDefs[mapName]
	if !ok {
		return fmt.Errorf("sensor.Start: mapa %q não encontrado no ELF", mapName)
	}

	c.mapFD, err = sysbpf.CreateMap(mapDef.Type, mapDef.KeySize, mapDef.ValueSize, mapDef.MaxEntries)
	if err != nil {
		return fmt.Errorf("sensor.Start: falha ao criar mapa eBPF: %w", err)
	}

	// Passo 3 — Carrega o programa no kernel
	// O buffer de log captura a saída do verifier em caso de rejeição.
	verifierLog := make([]byte, verifierLogSize)

	c.progFD, err = sysbpf.LoadProg(sysbpf.BPF_PROG_TYPE_TRACEPOINT, obj.Insns, obj.License, verifierLog)
	if err != nil {
		_ = unix.Close(int(c.mapFD))
		return fmt.Errorf("sensor.Start: falha ao carregar programa eBPF: %w", err)
	}

	// Passo 4 — Anexa ao tracepoint
	c.eventFD, err = sysbpf.AttachTracepoint(tracepointSubsystem, tracepointEvent, c.progFD)
	if err != nil {
		_ = unix.Close(int(c.progFD))
		_ = unix.Close(int(c.mapFD))
		return fmt.Errorf("sensor.Start: falha ao anexar ao tracepoint %s/%s: %w",
			tracepointSubsystem, tracepointEvent, err)
	}

	log.Printf("HOSA Sensor: capturando %s/%s via eBPF (mapFD=%d, progFD=%d)",
		tracepointSubsystem, tracepointEvent, c.mapFD, c.progFD)

	return nil
}

// ReadMetrics lê o contador acumulado de chamadas sys_brk do mapa eBPF.
// O valor é incrementado atomicamente pelo programa em kernel space a cada brk().
// Retorna 0 em caso de erro de leitura.
func (c *Collector) ReadMetrics() float64 {
	var key uint32 = 0
	var value uint64

	if err := sysbpf.LookupElem(c.mapFD, unsafe.Pointer(&key), unsafe.Pointer(&value)); err != nil {
		log.Printf("HOSA Sensor: erro ao ler mapa eBPF: %v", err)
		return 0
	}

	return float64(value)
}

// Close desanexa o programa eBPF e libera todos os file descriptors.
// Fechar o eventFD é suficiente para desativar o programa no kernel —
// o kernel remove o link quando nenhum fd referencia o perf event.
func (c *Collector) Close() {
	if c.eventFD > 0 {
		if err := sysbpf.Close(c.eventFD); err != nil {
			log.Printf("HOSA Sensor: erro ao fechar eventFD: %v", err)
		}
	}
	if c.progFD > 0 {
		if err := unix.Close(int(c.progFD)); err != nil {
			log.Printf("HOSA Sensor: erro ao fechar progFD: %v", err)
		}
	}
	if c.mapFD > 0 {
		if err := unix.Close(int(c.mapFD)); err != nil {
			log.Printf("HOSA Sensor: erro ao fechar mapFD: %v", err)
		}
	}
}

// resolveObjectPath encontra o sensors.o relativo à raiz do repositório.
func resolveObjectPath(relPath string) (string, error) {
	// Em desenvolvimento (go run ./cmd/hosa), usa o arquivo fonte como âncora
	// para subir até a raiz do repositório.
	_, callerFile, _, ok := runtime.Caller(1)
	if ok {
		repoRoot := filepath.Join(filepath.Dir(callerFile), "..", "..")
		candidate := filepath.Join(repoRoot, relPath)
		if fileExists(candidate) {
			return filepath.Abs(candidate)
		}
	}

	// Fallback: relativo ao diretório de trabalho atual
	if fileExists(relPath) {
		return filepath.Abs(relPath)
	}

	return "", fmt.Errorf("sensors.o não encontrado em %q — execute 'make build-bpf' primeiro", relPath)
}

func fileExists(path string) bool {
	var st unix.Stat_t
	return unix.Stat(path, &st) == nil
}