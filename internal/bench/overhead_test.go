package bench

import (
	"math/rand"
	"runtime"
	"testing"

	"github.com/bricio-sr/hosa/internal/brain"
	"github.com/bricio-sr/hosa/internal/state"
)

// BenchmarkMemoryFootprint measures the steady-state heap used by core
// HOSA structures after a full warm-up cycle.
//
// Uses HeapInuse after GC (stable measure) rather than delta (unreliable
// when GC runs between ReadMemStats calls).
func BenchmarkMemoryFootprint(b *testing.B) {
	// Allocate and warm up the core structures
	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < benchSamples*10; i++ {
		buf.Insert(stableReading(rng, 5.0))
	}

	cortex := brain.NewPredictiveCortex(buf, brain.DefaultConfig())
	cortex.Analyze() // ensure all internal state is allocated

	// Force GC and measure stable heap
	runtime.GC()
	runtime.GC() // twice to collect any finalizers

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	heapInuseKB := float64(ms.HeapInuse) / 1024.0
	heapAllocKB := float64(ms.HeapAlloc) / 1024.0

	b.ReportMetric(heapAllocKB, "heap_alloc_KB")
	b.ReportMetric(heapInuseKB, "heap_inuse_KB")
	b.ReportMetric(float64(ms.HeapObjects), "heap_objects")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// no-op — this benchmark measures allocation, not throughput
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