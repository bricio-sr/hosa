package bench

import (
	"math/rand"
	"runtime"
	"testing"

	"github.com/bricio-sr/hosa/internal/brain"
	"github.com/bricio-sr/hosa/internal/state"
)

// TestMemoryFootprint measures the heap delta caused by HOSA core structures.
// Not a throughput benchmark — measures allocation once and reports via t.Logf.
// Run with: go test -v -run TestMemoryFootprint ./internal/bench/
func TestMemoryFootprint(t *testing.T) {
	runtime.GC()
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	buf := state.NewRingBuffer(benchSamples, benchVars)
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < benchSamples*10; i++ {
		buf.Insert(stableReading(rng, 5.0))
	}
	cortex := brain.NewPredictiveCortex(buf, brain.DefaultConfig())
	cortex.Analyze()

	runtime.GC()
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	var deltaBytes uint64
	if after.HeapAlloc > before.HeapAlloc {
		deltaBytes = after.HeapAlloc - before.HeapAlloc
	}

	t.Logf("HOSA memory footprint after warm-up:")
	t.Logf("  delta_heap    = %d KB", deltaBytes/1024)
	t.Logf("  total_heap    = %d KB", after.HeapAlloc/1024)
	t.Logf("  heap_inuse    = %d KB", after.HeapInuse/1024)
	t.Logf("  heap_objects  = %d", after.HeapObjects)

	// O(1) claim: total footprint must be under 2MB regardless of sample count
	const maxBytes = 2 * 1024 * 1024
	if after.HeapAlloc > maxBytes {
		t.Errorf("heap_alloc %d KB exceeds 2MB budget — O(1) claim violated",
			after.HeapAlloc/1024)
	}

	runtime.KeepAlive(cortex)
	runtime.KeepAlive(buf)
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