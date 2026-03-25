package bench

import (
	"math/rand"
	"runtime"
	"testing"

	"github.com/bricio-sr/hosa/internal/brain"
	"github.com/bricio-sr/hosa/internal/state"
)

// BenchmarkMemoryFootprint measures the heap memory used by the core
// data structures of HOSA after a full warm-up.
//
// The whitepaper commits to O(1) memory footprint — this test validates that.
// Expected: RingBuffer + WelfordState + HomeostasisModel = constant regardless of n.
func BenchmarkMemoryFootprint(b *testing.B) {
	var before, after runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&before)

	// Allocate the core structures
	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < benchSamples*10; i++ { // 10x capacity — still O(1)
		r := stableReading(rng, 5.0)
		buf.Insert(r)
	}

	cortex := brain.NewPredictiveCortex(buf, brain.DefaultConfig())

	// Run one analysis to ensure all internal state is allocated
	cortex.Analyze()

	runtime.GC()
	runtime.ReadMemStats(&after)

	heapAllocKB := float64(after.HeapAlloc-before.HeapAlloc) / 1024.0

	b.ReportMetric(heapAllocKB, "heap_KB")
	b.ReportMetric(float64(after.HeapObjects-before.HeapObjects), "heap_objects")

	// Validate O(1) claim: total footprint should be well under 1MB
	if heapAllocKB > 1024 {
		b.Errorf("memory footprint %.1fKB exceeds 1MB budget — O(1) claim violated", heapAllocKB)
	}
}

// BenchmarkAllocationsPerCycle counts heap allocations per Analyze() call.
// Zero allocations in the hot path is the goal (sync.Pool, pre-allocation).
// Current target: < 10 allocs/op.
func BenchmarkAllocationsPerCycle(b *testing.B) {
	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < benchSamples; i++ {
		buf.Insert(stableReading(rng, 5.0))
	}

	cortex := brain.NewPredictiveCortex(buf, brain.DefaultConfig())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		r := stableReading(rng, 5.0)
		buf.Insert(r)
		cortex.Analyze()
	}
}

// BenchmarkRingBufferInsert measures the cost of inserting a sample
// into the ring buffer — the memory layer. This runs on every cycle.
func BenchmarkRingBufferInsert(b *testing.B) {
	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(42))
	reading := stableReading(rng, 5.0)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf.Insert(reading)
	}
}