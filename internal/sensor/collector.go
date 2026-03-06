package sensor

import (
	"log"
	"time"

	"github.com/bricio-sr/hosa/internal/bpf"
	"github.com/cilium/ebpf/link"
)

type Collector struct {
	objs bpf.bpfObjects
	kp   link.Link
}

// Start inicializa o sensor eBPF no Kernel
func (c *Collector) Start() error {
	// 1. Carrega o bytecode eBPF no Kernel
	var objs bpf.bpfObjects
	if err := bpf.LoadBpfObjects(&objs, nil); err != nil {
		return err
	}
	c.objs = objs

	// 2. Pendura o nosso programa no Tracepoint do sys_brk
	// É como ligar um eletrodo no nervo do Kernel
	l, err := link.Tracepoint("syscalls", "sys_enter_brk", objs.TraceSysBrk, nil)
	if err != nil {
		return err
	}
	c.kp = l

	log.Println("HOSA Sensor: Capturando alocações de memória via eBPF...")
	return nil
}

// ReadMetrics lê o valor acumulado no mapa eBPF
func (c *Collector) ReadMetrics() float64 {
	var key uint32 = 0
	var value uint64

	// Lê direto do mapa compartilhado com o Kernel
	err := c.objs.MemoryMetrics.Lookup(key, &value)
	if err != nil {
		log.Printf("Erro ao ler mapa eBPF: %v", err)
		return 0
	}

	return float64(value)
}

func (c *Collector) Close() {
	c.kp.Close()
	c.objs.Close()
}