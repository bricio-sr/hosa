// Package sensor — fragmentação de memória física.
//
// FragmentationMonitor monitora a fragmentação de páginas do kernel via
// /proc/buddyinfo e calcula a entropia de Shannon H_frag sobre a distribuição
// de páginas livres por ordem.
//
// Formulação de H_frag:
//
//	Para cada zona (DMA, DMA32, Normal):
//	  pages[o] = count[o] * 2^o           (páginas base representadas por cada order)
//	  p[o]     = pages[o] / total_pages   (fração ponderada por massa de memória)
//	  H_frag   = -Σ(p[o] * log2(p[o]))   (entropia de Shannon, bits)
//
// H_frag_norm = H_frag / log2(BuddyMaxOrder)  →  [0, 1]
//
// Interpretação:
//   - H_frag_norm → 0: toda memória concentrada em poucas orders — pouco fragmentada
//   - H_frag_norm → 1: memória distribuída uniformemente entre todas as orders — muito fragmentada
//
// Referência: whitepaper HOSA, Seção 7.2 — Termodinâmica de Memória, Fase 2.
package sensor

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

const (
	buddyInfoPath = "/proc/buddyinfo"
	hFragMax      = 3.4594 // log2(11) ≈ 3.4594 bits — entropia máxima teórica
	compactionPath = "/proc/sys/vm/compact_memory"

	// BuddyMaxOrder é o número de ordens do buddy allocator (0..10).
	// Exportado para uso nos benchmarks da Fase 2.
	BuddyMaxOrder = 11 // ordens 0..10 (página de 4K até 4M)
)

// FragState é o snapshot de fragmentação física da memória.
type FragState struct {
	// HFragNorm é a entropia de Shannon normalizada sobre buddyinfo, ponderada
	// por massa de memória (order-10 pesa 1024× mais que order-0).
	// Range: [0.0, 1.0]. Valores > FragEntropyThreshold indicam risco de compaction stall.
	HFragNorm float64

	// ZoneEntropies armazena H_frag por zona (ex: "Normal", "DMA32", "DMA").
	ZoneEntropies map[string]float64

	// LargestFreeOrderNormal é o maior order com páginas livres na zona Normal.
	// Medida direta do headroom de alocação contígua. 0 = zona Normal não detectada.
	LargestFreeOrderNormal int

	// TotalFreePagesMB é o total de memória livre em MB, somando todas as zonas.
	TotalFreePagesMB float64
}

// buddyZone representa uma linha do /proc/buddyinfo.
type buddyZone struct {
	node   int
	zone   string
	counts [BuddyMaxOrder]uint64
}

// FragConfig configura o FragmentationMonitor.
type FragConfig struct {
	// Threshold é o H_frag_norm acima do qual a compactação preemptiva é disparada.
	// Range (0, 1]. Default: 0.78.
	Threshold float64

	// CPUTroughThreshold é o nível de cpu_run_queue abaixo do qual o sistema
	// está em "CPU trough" — seguro para disparar compactação sem impacto visível.
	CPUTroughThreshold float64
}

// FragmentationMonitor lê /proc/buddyinfo e dispara compactação preemptiva
// durante calhas de CPU quando H_frag_norm excede o limiar configurado.
type FragmentationMonitor struct {
	cfg         FragConfig
	compactions uint64 // total de compactações disparadas — exposto para benchmarks
}

// NewFragmentationMonitor cria um monitor com a configuração fornecida.
func NewFragmentationMonitor(cfg FragConfig) *FragmentationMonitor {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 0.78
	}
	return &FragmentationMonitor{cfg: cfg}
}

// Sample lê /proc/buddyinfo e retorna o estado de fragmentação atual.
// cpuQueueRate é a métrica cpu_run_queue do ciclo atual — usada como proxy
// de carga de CPU para decidir se é seguro disparar compactação.
//
// Retorna (FragState, triggered bool, error).
// triggered=true indica que compactação preemptiva foi disparada neste ciclo.
func (fm *FragmentationMonitor) Sample(cpuQueueRate float64) (FragState, bool, error) {
	zones, err := readBuddyInfo()
	if err != nil {
		return FragState{}, false, fmt.Errorf("FragmentationMonitor.Sample: %w", err)
	}

	state := computeFragState(zones)

	triggered := false
	inTrough := cpuQueueRate <= fm.cfg.CPUTroughThreshold
	if state.HFragNorm > fm.cfg.Threshold && inTrough {
		if err := fm.triggerCompaction(); err == nil {
			fm.compactions++
			triggered = true
		}
	}

	return state, triggered, nil
}

// CompactionCount retorna o total de compactações preemptivas disparadas.
func (fm *FragmentationMonitor) CompactionCount() uint64 {
	return fm.compactions
}

// ReadFragState é a versão exportada de readBuddyInfo+computeFragState,
// exposta para uso direto nos benchmarks da Fase 2.
func ReadFragState() (FragState, error) {
	zones, err := readBuddyInfo()
	if err != nil {
		return FragState{}, err
	}
	return computeFragState(zones), nil
}

// ZoneEntropyBench é a versão exportada de zoneEntropy para benchmarks.
// Permite medir o custo isolado do cálculo de entropia sem overhead de I/O.
func ZoneEntropyBench(counts [BuddyMaxOrder]uint64) (float64, uint64) {
	return zoneEntropy(counts)
}

// readBuddyInfo parseia /proc/buddyinfo e retorna todas as zonas.
func readBuddyInfo() ([]buddyZone, error) {
	data, err := os.ReadFile(buddyInfoPath)
	if err != nil {
		return nil, fmt.Errorf("readBuddyInfo: %w", err)
	}

	var zones []buddyZone
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		z, err := parseBuddyLine(line)
		if err != nil {
			continue // linha mal-formada — pula sem falhar
		}
		zones = append(zones, z)
	}

	if len(zones) == 0 {
		return nil, fmt.Errorf("readBuddyInfo: nenhuma zona encontrada em %s", buddyInfoPath)
	}

	return zones, nil
}

// parseBuddyLine parseia uma linha no formato:
//
//	"Node 0, zone   Normal   6259   9145   4312 ..."
func parseBuddyLine(line string) (buddyZone, error) {
	// Formato: "Node N, zone  ZONENAME  c0 c1 c2 ... c10"
	nodeIdx := strings.Index(line, "Node ")
	zoneIdx := strings.Index(line, "zone ")
	if nodeIdx < 0 || zoneIdx < 0 || zoneIdx <= nodeIdx {
		return buddyZone{}, fmt.Errorf("formato inválido: %q", line)
	}

	// Extrai node ID
	nodeStr := line[nodeIdx+5:]
	commaIdx := strings.Index(nodeStr, ",")
	if commaIdx < 0 {
		return buddyZone{}, fmt.Errorf("vírgula não encontrada após node: %q", line)
	}
	nodeID, err := strconv.Atoi(strings.TrimSpace(nodeStr[:commaIdx]))
	if err != nil {
		return buddyZone{}, fmt.Errorf("node ID inválido: %w", err)
	}

	// Extrai nome da zona e contagens
	rest := strings.TrimSpace(line[zoneIdx+5:])
	fields := strings.Fields(rest)
	if len(fields) < BuddyMaxOrder+1 {
		return buddyZone{}, fmt.Errorf("campos insuficientes (%d < %d): %q", len(fields), BuddyMaxOrder+1, line)
	}

	zoneName := fields[0]
	var counts [BuddyMaxOrder]uint64
	for i := 0; i < BuddyMaxOrder; i++ {
		n, err := strconv.ParseUint(fields[i+1], 10, 64)
		if err != nil {
			return buddyZone{}, fmt.Errorf("contagem inválida na order %d: %w", i, err)
		}
		counts[i] = n
	}

	return buddyZone{node: nodeID, zone: zoneName, counts: counts}, nil
}

// computeFragState calcula H_frag e métricas derivadas a partir das zonas.
func computeFragState(zones []buddyZone) FragState {
	state := FragState{
		ZoneEntropies:          make(map[string]float64, len(zones)),
		LargestFreeOrderNormal: 0,
	}

	// Agrega a entropia ponderada pelo total de páginas de cada zona
	totalPagesAll := uint64(0)
	weightedEntropySum := 0.0

	for _, z := range zones {
		h, totalPages := zoneEntropy(z.counts)
		state.ZoneEntropies[z.zone] = h / hFragMax // normaliza por zona

		state.TotalFreePagesMB += float64(totalPages) * 4096 / (1 << 20)

		weightedEntropySum += h * float64(totalPages)
		totalPagesAll += totalPages

		// Maior order com páginas livres na zona Normal
		if z.zone == "Normal" {
			for o := BuddyMaxOrder - 1; o >= 0; o-- {
				if z.counts[o] > 0 {
					if o > state.LargestFreeOrderNormal {
						state.LargestFreeOrderNormal = o
					}
					break
				}
			}
		}
	}

	if totalPagesAll > 0 {
		// Entropia global ponderada por massa de memória de cada zona
		state.HFragNorm = (weightedEntropySum / float64(totalPagesAll)) / hFragMax
	}

	return state
}

// zoneEntropy calcula a entropia de Shannon para uma zona do buddy allocator.
// Pondera cada order pelo número de páginas base que representa (count * 2^order),
// de forma que order-10 (blocos de 4MB) pesa 1024× mais que order-0 (páginas de 4KB).
// Retorna (entropia em bits, total de páginas base representadas).
func zoneEntropy(counts [BuddyMaxOrder]uint64) (float64, uint64) {
	// Calcula o total ponderado por massa de memória
	var totalPages uint64
	for o := 0; o < BuddyMaxOrder; o++ {
		totalPages += counts[o] * (1 << uint(o))
	}

	if totalPages == 0 {
		return 0, 0
	}

	fTotal := float64(totalPages)
	h := 0.0
	for o := 0; o < BuddyMaxOrder; o++ {
		if counts[o] == 0 {
			continue
		}
		pages := float64(counts[o]) * float64(uint64(1)<<uint(o))
		p := pages / fTotal
		h -= p * math.Log2(p)
	}

	return h, totalPages
}

// triggerCompaction escreve em /proc/sys/vm/compact_memory para disparar
// uma passagem de compactação de memória no kernel. A operação é não-bloqueante
// do ponto de vista do HOSA — o kernel realiza a compactação de forma assíncrona.
// Requer CAP_SYS_ADMIN.
func (fm *FragmentationMonitor) triggerCompaction() error {
	return os.WriteFile(compactionPath, []byte("1"), 0200)
}
