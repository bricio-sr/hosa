package bench

import (
	"math/rand"
	"runtime"
	"testing"

	"github.com/bricio-sr/hosa/internal/brain"
	"github.com/bricio-sr/hosa/internal/state"
)

// BenchmarkMemoryFootprint measures the steady-state heap after HOSA warm-up.
// Reports heap_alloc_KB and heap_objects as custom metrics.
func BenchmarkMemoryFootprint(b *testing.B) {
	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < benchSamples*10; i++ {
		buf.Insert(stableReading(rng, 5.0))
	}

	cortex := brain.NewPredictiveCortex(buf, brain.DefaultConfig())
	cortex.Analyze()

	runtime.GC()
	runtime.GC()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	heapAllocKB := float64(ms.HeapAlloc) / 1024.0
	heapInuseKB := float64(ms.HeapInuse) / 1024.0

	b.ReportMetric(heapAllocKB, "heap_alloc_KB")
	b.ReportMetric(heapInuseKB, "heap_inuse_KB")
	b.ReportMetric(float64(ms.HeapObjects), "heap_objects")

	// Keep b.N loop to satisfy the benchmark framework
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runtime.KeepAlive(cortex)
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