// Package bench — Fase 2 benchmark suite.
//
// Valida as propriedades de desempenho do Sistema Nervoso Simpático do HOSA:
//
//  1. BenchmarkHFragComputation: custo de ReadFragState() a cada ciclo
//     Target: < 50µs p99 (leitura de /proc/buddyinfo + cálculo de entropia)
//
//  2. BenchmarkHFragEntropyCalculation: custo isolado do cálculo de entropia
//     sobre dados sintéticos (sem I/O) — valida que o algoritmo é O(11) por zona.
//     Target: < 1µs (dominado por log2, não por I/O)
//
//  3. BenchmarkFragmentationMonitorSample: custo total do ciclo do FragmentationMonitor
//     incluindo leitura de /proc/buddyinfo, parsing e cálculo de entropia.
//     Target: < 100µs p999
//
//  4. BenchmarkCPUSetString: custo de formatação do CPUSet para escrita em cpuset.cpus
//     Target: < 500ns (chamada rara, mas validar que não aloca em excesso)
//
//  5. BenchmarkSurvivalDecisionCycle: latência do ciclo completo de detecção e
//     decisão de sobrevivência (Cortex.Analyze → classify → LevelSurvival).
//     Verifica que a nova lógica não aumenta a latência de análise da Fase 1.
//     Target: latência da Fase 1 + < 1µs overhead
//
// Referência: whitepaper HOSA, Seção 8 — Fase 2 Benchmarks.
package bench

import (
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/bricio-sr/hosa/internal/brain"
	"github.com/bricio-sr/hosa/internal/motor"
	"github.com/bricio-sr/hosa/internal/sensor"
	"github.com/bricio-sr/hosa/internal/state"
)

// --- Benchmark 1: H_frag lendo /proc/buddyinfo real ---

// BenchmarkHFragComputation mede o custo de uma leitura completa de /proc/buddyinfo
// + parsing + cálculo de entropia. Esta é a operação mais custosa da Fase 2
// em cada ciclo do loop principal.
func BenchmarkHFragComputation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = sensor.ReadFragState()
	}
}

// BenchmarkHFragComputationLatencyDistribution coleta N amostras individuais
// e reporta p50/p99/p999 da latência de ReadFragState().
func BenchmarkHFragComputationLatencyDistribution(b *testing.B) {
	const N = 1000
	latencies := make([]time.Duration, N)

	b.ResetTimer()

	for i := 0; i < N; i++ {
		start := time.Now()
		_, _ = sensor.ReadFragState()
		latencies[i] = time.Since(start)
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	p50 := latencies[N*50/100]
	p99 := latencies[N*99/100]
	p999 := latencies[N*999/1000]

	b.ReportMetric(float64(p50.Nanoseconds()), "p50_ns")
	b.ReportMetric(float64(p99.Nanoseconds()), "p99_ns")
	b.ReportMetric(float64(p999.Nanoseconds()), "p999_ns")
}

// --- Benchmark 2: Cálculo de entropia isolado (sem I/O) ---

// BenchmarkHFragEntropyCalculation mede o custo do algoritmo de entropia puro,
// usando dados sintéticos fixos. Isola o custo computacional (log2, divisões)
// do custo de I/O (leitura de /proc/buddyinfo).
func BenchmarkHFragEntropyCalculation(b *testing.B) {
	// Simula uma zona Normal típica com distribuição de páginas realista
	// (maior concentração em ordens 2-4, poucas pages em ordens altas)
	syntheticCounts := [sensor.BuddyMaxOrder]uint64{
		6259, 9145, 4312, 2871, 1204, 589, 124, 42, 12, 3, 1,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = sensor.ZoneEntropyBench(syntheticCounts)
	}
}

// --- Benchmark 3: FragmentationMonitor.Sample() completo ---

// BenchmarkFragmentationMonitorSample mede o custo de um ciclo completo do
// FragmentationMonitor incluindo decisão de compactação (threshold não atingido
// para evitar efeitos colaterais no sistema durante o benchmark).
func BenchmarkFragmentationMonitorSample(b *testing.B) {
	monitor := sensor.NewFragmentationMonitor(sensor.FragConfig{
		Threshold:          0.99, // limiar irreal alto — não dispara compactação durante o bench
		CPUTroughThreshold: 0.0,  // nunca em trough
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = monitor.Sample(100.0) // cpu_queue_rate alto = não em trough
	}
}

// --- Benchmark 4: CPUSet.String() ---

// BenchmarkCPUSetString mede o custo de formatação do CPUSet para escrita
// em cpuset.cpus. Testa tanto ranges contíguos (0-7) quanto fragmentados (0,2,4,6,8).
func BenchmarkCPUSetString(b *testing.B) {
	contiguous := motor.CPUSet{0, 1, 2, 3, 4, 5, 6, 7}
	fragmented := motor.CPUSet{0, 2, 4, 6, 8, 10, 12, 14}

	b.Run("contiguous", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = contiguous.String()
		}
	})

	b.Run("fragmented", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = fragmented.String()
		}
	})
}

// --- Benchmark 5: Ciclo de decisão Fase 1 + Fase 2 ---

// BenchmarkSurvivalDecisionCycle mede a latência do ciclo completo de análise
// incluindo o novo threshold LevelSurvival. Compara com BenchmarkAnalyzeCycle
// da Fase 1 para verificar que o overhead da Fase 2 na classificação é < 1µs.
func BenchmarkSurvivalDecisionCycle(b *testing.B) {
	// Cortex com configuração da Fase 2 (inclui ThresholdSurvival)
	cfg := brain.DefaultConfig()
	cfg.ThresholdSurvival = brain.ThresholdSurvival

	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(42))

	// Warm com baseline estável
	for i := 0; i < benchSamples; i++ {
		reading := make([]float64, benchVars)
		for j := range reading {
			reading[j] = 100.0 + rng.NormFloat64()*5.0
		}
		buf.Insert(reading)
	}

	cortex := brain.NewPredictiveCortex(buf, cfg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cortex.Analyze()
	}
}

// BenchmarkSurvivalDecisionLatencyDistribution coleta p50/p99/p999 do ciclo
// de análise com o threshold de Sobrevivência habilitado.
func BenchmarkSurvivalDecisionLatencyDistribution(b *testing.B) {
	cfg := brain.DefaultConfig()
	cfg.ThresholdSurvival = brain.ThresholdSurvival

	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(99))

	for i := 0; i < benchSamples; i++ {
		reading := make([]float64, benchVars)
		for j := range reading {
			reading[j] = 100.0 + rng.NormFloat64()*5.0
		}
		buf.Insert(reading)
	}

	cortex := brain.NewPredictiveCortex(buf, cfg)

	const N = 10_000
	latencies := make([]time.Duration, N)

	b.ResetTimer()

	for i := 0; i < N; i++ {
		start := time.Now()
		cortex.Analyze()
		latencies[i] = time.Since(start)
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	p50 := latencies[N*50/100]
	p99 := latencies[N*99/100]
	p999 := latencies[N*999/1000]

	b.ReportMetric(float64(p50.Nanoseconds()), "p50_ns")
	b.ReportMetric(float64(p99.Nanoseconds()), "p99_ns")
	b.ReportMetric(float64(p999.Nanoseconds()), "p999_ns")
}

// --- Benchmark 6: Detecção de LevelSurvival ---

// BenchmarkDetectionRate_SurvivalCascade mede quantos ciclos o Córtex leva
// para detectar uma cascata de falha que deve acionar LevelSurvival.
// Injeta um spike de 20× na leitura após a calibração e conta ciclos até D_M ≥ 12.0.
func BenchmarkDetectionRate_SurvivalCascade(b *testing.B) {
	cfg := brain.DefaultConfig()
	cfg.ThresholdSurvival = brain.ThresholdSurvival

	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(42))

	// Calibração: baseline estável
	for i := 0; i < benchSamples; i++ {
		reading := make([]float64, benchVars)
		for j := range reading {
			reading[j] = 100.0 + rng.NormFloat64()*5.0
		}
		buf.Insert(reading)
	}

	b.ResetTimer()

	for run := 0; run < b.N; run++ {
		// Reinicia buffer e cortex para este run
		freshBuf := state.NewRingBuffer(benchSamples, benchVars)
		for i := 0; i < benchSamples; i++ {
			r := make([]float64, benchVars)
			for j := range r {
				r[j] = 100.0 + rng.NormFloat64()*5.0
			}
			freshBuf.Insert(r)
		}
		cortex := brain.NewPredictiveCortex(freshBuf, cfg)

		// Injeta cascata: 20× o basal em todas as dimensões
		cyclesUntilSurvival := 0
		maxCycles := 200
		for i := 0; i < maxCycles; i++ {
			spike := make([]float64, benchVars)
			for j := range spike {
				spike[j] = 2000.0 + rng.NormFloat64()*10.0 // 20× basal
			}
			freshBuf.Insert(spike)

			_, _, level, _ := cortex.Analyze()
			if level == brain.LevelSurvival {
				cyclesUntilSurvival = i + 1
				break
			}
		}

		b.ReportMetric(float64(cyclesUntilSurvival), "cycles_to_survival")
	}
}

// --- Benchmark 7: H_frag com zonas sintéticas diversas ---

// BenchmarkHFragEntropyProfiles valida o comportamento da entropia em
// perfis de fragmentação extremos: totalmente fragmentado vs. totalmente consolidado.
func BenchmarkHFragEntropyProfiles(b *testing.B) {
	// Perfil 1: totalmente fragmentado — tudo em order-0 (páginas de 4KB)
	allOrder0 := [sensor.BuddyMaxOrder]uint64{
		100000, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}

	// Perfil 2: totalmente consolidado — tudo em order-10 (blocos de 4MB)
	allOrder10 := [sensor.BuddyMaxOrder]uint64{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 100,
	}

	// Perfil 3: distribuição uniforme — máxima entropia
	uniform := [sensor.BuddyMaxOrder]uint64{
		1000, 500, 250, 125, 62, 31, 15, 8, 4, 2, 1,
	}

	b.Run("all_order0_fragmented", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = sensor.ZoneEntropyBench(allOrder0)
		}
	})

	b.Run("all_order10_consolidated", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = sensor.ZoneEntropyBench(allOrder10)
		}
	})

	b.Run("uniform_max_entropy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = sensor.ZoneEntropyBench(uniform)
		}
	})
}
