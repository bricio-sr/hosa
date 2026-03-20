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
	bpfObjectPath   = "internal/bpf/sensors.o"
	mapName         = "hosa_metrics"
	verifierLogSize = 64 * 1024

	// NumVars é a dimensão do vetor de estado — deve ser sincronizado com sensors.c.
	NumVars = 4

	// Índices das variáveis no vetor — espelham os defines do sensors.c.
	IdxCPURunQueue   = 0
	IdxMemBrkCalls   = 1
	IdxMemPageFaults = 2
	IdxIOBlockOps    = 3
)

// probe descreve um programa eBPF e seu ponto de anexo no kernel.
type probe struct {
	subsystem string
	event     string
}

// probes lista os 4 programas que o sensors.c expõe, na ordem dos índices acima.
var probes = []probe{
	{subsystem: "sched", event: "sched_wakeup"},
	{subsystem: "syscalls", event: "sys_enter_brk"},
	{subsystem: "exceptions", event: "page_fault_kernel"},
	{subsystem: "block", event: "block_rq_issue"},
}

// Collector é o sensor eBPF do HOSA.
// Carrega os 4 programas do sensors.o, compartilham um único mapa de NUM_VARS entradas.
type Collector struct {
	mapFD    sysbpf.MapFD   // mapa compartilhado entre todos os programas
	progFDs  []sysbpf.ProgFD // um fd por programa carregado
	eventFDs []int           // um fd por perf event anexado
}

// Start inicializa os 4 sensores eBPF:
//  1. Parseia o ELF do sensors.o
//  2. Cria o mapa hosa_metrics (4 entradas) no kernel
//  3. Para cada seção de programa no ELF:
//     a. Resolve relocações (injeta map_fd)
//     b. BPF_PROG_LOAD
//     c. perf_event_open + ioctl attach
func (c *Collector) Start() error {
	objPath, err := resolveObjectPath(bpfObjectPath)
	if err != nil {
		return fmt.Errorf("sensor.Start: %w", err)
	}

	obj, err := sysbpf.LoadObject(objPath)
	if err != nil {
		return fmt.Errorf("sensor.Start: falha ao parsear ELF %q: %w", objPath, err)
	}

	// Passo 2 — Cria o mapa compartilhado
	mapDef, ok := obj.MapDefs[mapName]
	if !ok {
		return fmt.Errorf("sensor.Start: mapa %q não encontrado no ELF (encontrados: %v)", mapName, obj.MapDefNames())
	}

	c.mapFD, err = sysbpf.CreateMap(mapDef.Type, mapDef.KeySize, mapDef.ValueSize, mapDef.MaxEntries)
	if err != nil {
		return fmt.Errorf("sensor.Start: falha ao criar mapa eBPF: %w", err)
	}

	// Passos 3a-3c — Para cada programa no ELF
	for i, p := range probes {
		progInsns, ok := obj.InsnsBySection[fmt.Sprintf("tracepoint/%s/%s", p.subsystem, p.event)]
		if !ok {
			// Probe não encontrada no ELF — pula silenciosamente (kernel pode não ter o tracepoint)
			log.Printf("HOSA Sensor: probe %s/%s não encontrada no ELF — pulando", p.subsystem, p.event)
			continue
		}

		// 3a — Resolve relocações para esta seção
		insnsCopy := make([]byte, len(progInsns))
		copy(insnsCopy, progInsns)
		objCopy := &sysbpf.BPFObjectSlice{Insns: insnsCopy, InsnMapRefs: obj.InsnMapRefsBySection[fmt.Sprintf("tracepoint/%s/%s", p.subsystem, p.event)]}
		if err = objCopy.RelocateInsns(sysbpf.MapFDs{mapName: c.mapFD}); err != nil {
			c.closeAll()
			return fmt.Errorf("sensor.Start: probe %d (%s/%s) relocação falhou: %w", i, p.subsystem, p.event, err)
		}

		// 3b — BPF_PROG_LOAD
		verifierLog := make([]byte, verifierLogSize)
		progFD, err := sysbpf.LoadProg(sysbpf.BPF_PROG_TYPE_TRACEPOINT, objCopy.Insns, obj.License, verifierLog)
		if err != nil {
			c.closeAll()
			return fmt.Errorf("sensor.Start: probe %s/%s falhou no verifier: %w", p.subsystem, p.event, err)
		}
		c.progFDs = append(c.progFDs, progFD)

		// 3c — Attach
		eventFD, err := sysbpf.AttachTracepoint(p.subsystem, p.event, progFD)
		if err != nil {
			c.closeAll()
			return fmt.Errorf("sensor.Start: falha ao anexar probe %s/%s: %w", p.subsystem, p.event, err)
		}
		c.eventFDs = append(c.eventFDs, eventFD)

		log.Printf("HOSA Sensor: probe ativa — %s/%s", p.subsystem, p.event)
	}

	if len(c.eventFDs) == 0 {
		c.closeAll()
		return fmt.Errorf("sensor.Start: nenhuma probe foi anexada com sucesso")
	}

	log.Printf("HOSA Sensor: %d/%d probes ativas (mapFD=%d)", len(c.eventFDs), len(probes), c.mapFD)
	return nil
}

// ReadMetrics lê o vetor de estado completo do mapa eBPF.
// Retorna []float64 de tamanho NumVars com os contadores acumulados.
// Índices: [cpu_run_queue, mem_brk_calls, mem_page_faults, io_block_ops]
func (c *Collector) ReadMetrics() []float64 {
	result := make([]float64, NumVars)

	for i := 0; i < NumVars; i++ {
		var key uint32 = uint32(i)
		var value uint64

		if err := sysbpf.LookupElem(c.mapFD, unsafe.Pointer(&key), unsafe.Pointer(&value)); err != nil {
			log.Printf("HOSA Sensor: erro ao ler índice %d do mapa: %v", i, err)
			continue
		}
		result[i] = float64(value)
	}

	return result
}

// Close desanexa todos os programas eBPF e libera os file descriptors.
func (c *Collector) Close() {
	c.closeAll()
}

func (c *Collector) closeAll() {
	for _, fd := range c.eventFDs {
		if fd > 0 {
			sysbpf.Close(fd)
		}
	}
	for _, fd := range c.progFDs {
		if int(fd) > 0 {
			unix.Close(int(fd))
		}
	}
	if int(c.mapFD) > 0 {
		unix.Close(int(c.mapFD))
	}
	c.eventFDs = nil
	c.progFDs = nil
	c.mapFD = 0
}

func resolveObjectPath(relPath string) (string, error) {
	_, callerFile, _, ok := runtime.Caller(1)
	if ok {
		repoRoot := filepath.Join(filepath.Dir(callerFile), "..", "..")
		candidate := filepath.Join(repoRoot, relPath)
		if fileExists(candidate) {
			return filepath.Abs(candidate)
		}
	}
	if fileExists(relPath) {
		return filepath.Abs(relPath)
	}
	return "", fmt.Errorf("sensors.o não encontrado em %q — execute 'make build-bpf' primeiro", relPath)
}

func fileExists(path string) bool {
	var st unix.Stat_t
	return unix.Stat(path, &st) == nil
}