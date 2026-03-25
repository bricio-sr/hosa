// Package bench contains the Phase 1 benchmark suite for the HOSA dissertation.
//
// This file measures the latency of the full reflex arc cycle:
//   Analyze() → classify() → Apply() (detection → decision → actuation)
//
// Methodology: Go's testing.B framework with explicit timer control.
// All allocations are pre-warmed to avoid measuring GC startup.
// Results are reported as p50/p99/p999 from a sorted latency distribution.
//
// Reference: HOSA whitepaper, Section 7 — Language Trade-offs,
// "benchmarks measuring p50/p99 latency and jitter".
package bench

import (
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/bricio-sr/hosa/internal/brain"
	"github.com/bricio-sr/hosa/internal/state"
)

const (
	benchVars    = 4   // must match sensor.NumVars
	benchSamples = 300 // warm buffer to enable cortex analysis
)

// newWarmCortex returns a PredictiveCortex with a fully populated buffer,
// ready to analyze without cold-start delays.
func newWarmCortex(b *testing.B) *brain.PredictiveCortex {
	b.Helper()
	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(42))

	// Populate with stable baseline samples (low variance, realistic values)
	for i := 0; i < benchSamples; i++ {
		reading := make([]float64, benchVars)
		for j := range reading {
			reading[j] = 100.0 + rng.NormFloat64()*5.0
		}
		buf.Insert(reading)
	}

	return brain.NewPredictiveCortex(buf, brain.DefaultConfig())
}

// BenchmarkAnalyzeCycle measures the end-to-end latency of one Analyze() call.
// This is the hot path that runs every 100ms (vigilance) or 1s (homeostasis).
func BenchmarkAnalyzeCycle(b *testing.B) {
	cortex := newWarmCortex(b)
	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(99))

	// Pre-warm to avoid measuring first-call allocation
	for i := 0; i < benchSamples; i++ {
		r := make([]float64, benchVars)
		for j := range r {
			r[j] = 100.0 + rng.NormFloat64()*5.0
		}
		buf.Insert(r)
	}
	cortex2 := brain.NewPredictiveCortex(buf, brain.DefaultConfig())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cortex2.Analyze()
		_ = cortex
	}
}

// BenchmarkAnalyzeLatencyDistribution collects N individual latency samples
// and reports p50, p99, p999. Unlike BenchmarkAnalyzeCycle which reports
// average, this reveals tail latency — critical for the dissertation argument
// about GC pauses.
func BenchmarkAnalyzeLatencyDistribution(b *testing.B) {
	cortex := newWarmCortex(b)
	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(7))

	for i := 0; i < benchSamples; i++ {
		r := make([]float64, benchVars)
		for j := range r {
			r[j] = 100.0 + rng.NormFloat64()*5.0
		}
		buf.Insert(r)
	}
	cortex2 := brain.NewPredictiveCortex(buf, brain.DefaultConfig())

	const N = 10_000
	latencies := make([]time.Duration, N)

	b.ResetTimer()

	for i := 0; i < N; i++ {
		start := time.Now()
		cortex2.Analyze()
		latencies[i] = time.Since(start)
		_ = cortex
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	p50 := latencies[N*50/100]
	p99 := latencies[N*99/100]
	p999 := latencies[N*999/1000]

	b.ReportMetric(float64(p50.Nanoseconds()), "p50_ns")
	b.ReportMetric(float64(p99.Nanoseconds()), "p99_ns")
	b.ReportMetric(float64(p999.Nanoseconds()), "p999_ns")
}

// BenchmarkWelfordUpdate measures the cost of a single Welford state update.
// This is the incremental covariance update — O(p²) per sample.
func BenchmarkWelfordUpdate(b *testing.B) {
	w := brain.NewWelfordState(benchVars)
	rng := rand.New(rand.NewSource(1))
	reading := make([]float64, benchVars)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for j := range reading {
			reading[j] = rng.NormFloat64()
		}
		w.Update(reading)
	}
}

// BenchmarkMahalanobisCalculation measures the D_M computation cost alone,
// isolating it from Welford and buffer operations.
func BenchmarkMahalanobisCalculation(b *testing.B) {
	// Build a stable model with known baseline
	w := brain.NewWelfordState(benchVars)
	rng := rand.New(rand.NewSource(2))

	for i := 0; i < 1000; i++ {
		r := make([]float64, benchVars)
		for j := range r {
			r[j] = 50.0 + rng.NormFloat64()*3.0
		}
		w.Update(r)
	}

	mean := w.Mean()
	cov, _ := w.Covariance()
	invCov, _ := cov.Inverse()
	model := brain.NewHomeostasisModel(mean, invCov)

	// Current observation slightly off baseline
	current := makeSampleMatrix(benchVars, 55.0)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		model.CalculateStress(current)
	}
}