package bench

import (
	"math/rand"
	"testing"

	"github.com/bricio-sr/hosa/internal/brain"
	"github.com/bricio-sr/hosa/internal/state"
)
// BenchmarkFalsePositiveRate simulates a healthy system (no injected fault)
// and counts how many cycles the cortex escalates above homeostasis.
//
// A healthy system should stay at LevelHomeostasis almost always.
// Any escalation in this scenario is a false positive.
//
// This benchmark does NOT use b.N — it runs a fixed number of cycles
// and reports the FPR as a custom metric.
func BenchmarkFalsePositiveRate(b *testing.B) {
	const cycles = 10_000
	const variance = 8.0 // realistic sched_wakeup variance

	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(42))

	// Warm-up phase: populate baseline
	for i := 0; i < benchSamples; i++ {
		r := stableReading(rng, variance)
		buf.Insert(r)
	}

	cortex := brain.NewPredictiveCortex(buf, brain.DefaultConfig())

	var falsePositives int

	b.ResetTimer()

	for i := 0; i < cycles; i++ {
		// Insert a new stable reading
		r := stableReading(rng, variance)
		buf.Insert(r)

		_, _, level, _ := cortex.Analyze()
		if level > brain.LevelHomeostasis {
			falsePositives++
		}
	}

	fpr := float64(falsePositives) / float64(cycles) * 100.0
	b.ReportMetric(fpr, "fpr_%")
	b.ReportMetric(float64(falsePositives), "false_positives")
	b.ReportMetric(float64(cycles), "total_cycles")
}

// BenchmarkDetectionRate_MemoryLeak simulates a gradual memory leak
// and measures how quickly the cortex escalates to Vigilance or above.
//
// Design: basal learned on STABLE data only. Leak injected AFTER calibration
// to prevent Welford from absorbing the leak into the baseline.
func BenchmarkDetectionRate_MemoryLeak(b *testing.B) {
	const stablePhase    = benchSamples * 2 // 600 stable cycles to build clean baseline
	const leakPhase      = 2_000
	const leakRatePerCycle = 50.0 // units/cycle — aggressive leak per whitepaper Fig.1

	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(42))

	// Phase 1: build clean, stable baseline
	for i := 0; i < stablePhase; i++ {
		buf.Insert(stableReading(rng, 5.0))
	}

	cortex := brain.NewPredictiveCortex(buf, brain.DefaultConfig())

	// Drain cold-start with stable data (MinSamples cycles)
	for i := 0; i < benchSamples; i++ {
		buf.Insert(stableReading(rng, 5.0))
		cortex.Analyze()
	}

	// Phase 2: inject leak
	var detectionCycle int = -1
	baselineBrk := 100.0

	b.ResetTimer()

	for i := 0; i < leakPhase; i++ {
		r := stableReading(rng, 5.0)
		r[idxMemBrkCalls] = baselineBrk + float64(i)*leakRatePerCycle
		buf.Insert(r)

		_, _, level, _ := cortex.Analyze()
		if level >= brain.LevelVigilance && detectionCycle == -1 {
			detectionCycle = i
		}
	}

	if detectionCycle == -1 {
		detectionCycle = leakPhase // not detected within window
	}

	b.ReportMetric(float64(detectionCycle), "cycles_to_detect")
	b.ReportMetric(float64(detectionCycle)*float64(normalInterval.Milliseconds()), "ms_to_detect")
	b.ReportMetric(leakRatePerCycle, "leak_rate_per_cycle")
}

// BenchmarkDetectionRate_CPUBurn simulates a sudden CPU spike
// (sched_wakeup rate jumps 10x) and measures detection latency.
func BenchmarkDetectionRate_CPUBurn(b *testing.B) {
	const cycles = 2_000
	const burnMultiplier = 10.0

	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < benchSamples; i++ {
		buf.Insert(stableReading(rng, 5.0))
	}

	cortex := brain.NewPredictiveCortex(buf, brain.DefaultConfig())

	var detectionCycle int = -1

	b.ResetTimer()

	for i := 0; i < cycles; i++ {
		r := stableReading(rng, 5.0)
		// At cycle 100: inject CPU burn
		if i >= 100 {
			r[idxCPURunQueue] *= burnMultiplier
		}
		buf.Insert(r)

		_, _, level, _ := cortex.Analyze()
		if level >= brain.LevelVigilance && detectionCycle == -1 && i >= 100 {
			detectionCycle = i - 100 // cycles after fault injection
		}
	}

	if detectionCycle == -1 {
		detectionCycle = cycles
	}

	b.ReportMetric(float64(detectionCycle), "cycles_after_fault")
}

// stableReading generates a stable multivariate reading with Gaussian noise.
func stableReading(rng *rand.Rand, variance float64) []float64 {
	r := make([]float64, benchVars)
	baselines := []float64{200.0, 100.0, 50.0, 30.0} // cpu, mem_brk, page_fault, block_io
	for j := range r {
		r[j] = baselines[j] + rng.NormFloat64()*variance
		if r[j] < 0 {
			r[j] = 0
		}
	}
	return r
}