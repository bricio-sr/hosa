window.BENCHMARK_DATA = {
  "lastUpdate": 1774624794756,
  "repoUrl": "https://github.com/bricio-sr/hosa",
  "entries": {
    "Benchmark": [
      {
        "commit": {
          "author": {
            "email": "f@bricio.dev.br",
            "name": "Fabricio Amorim",
            "username": "bricio-sr"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "628c449dc2a809e25d838c9d3aea003cd9d85ced",
          "message": "fix(actions): Change benchmark path",
          "timestamp": "2026-03-27T12:18:52-03:00",
          "tree_id": "d91e7056983d1d24e622e6a5e8d29ea33e1f76c2",
          "url": "https://github.com/bricio-sr/hosa/commit/628c449dc2a809e25d838c9d3aea003cd9d85ced"
        },
        "date": 1774624794147,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkAnalyzeCycle",
            "value": 3266,
            "unit": "ns/op\t   10224 B/op\t       6 allocs/op",
            "extra": "1823062 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - ns/op",
            "value": 3266,
            "unit": "ns/op",
            "extra": "1823062 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - B/op",
            "value": 10224,
            "unit": "B/op",
            "extra": "1823062 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "1823062 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution",
            "value": 0.03363,
            "unit": "ns/op\t      2436 p50_ns\t     82962 p999_ns\t     18726 p99_ns\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - ns/op",
            "value": 0.03363,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p50_ns",
            "value": 2436,
            "unit": "p50_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p999_ns",
            "value": 82962,
            "unit": "p999_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p99_ns",
            "value": 18726,
            "unit": "p99_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkWelfordUpdate",
            "value": 57.43,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkWelfordUpdate - ns/op",
            "value": 57.43,
            "unit": "ns/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkWelfordUpdate - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkWelfordUpdate - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation",
            "value": 143.7,
            "unit": "ns/op\t     136 B/op\t       4 allocs/op",
            "extra": "41865556 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - ns/op",
            "value": 143.7,
            "unit": "ns/op",
            "extra": "41865556 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - B/op",
            "value": 136,
            "unit": "B/op",
            "extra": "41865556 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "41865556 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate",
            "value": 0.03864,
            "unit": "ns/op\t      1821 false_positives\t        18.21 fpr_%\t     10000 total_cycles\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate - ns/op",
            "value": 0.03864,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate - false_positives",
            "value": 1821,
            "unit": "false_positives",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate - fpr_%",
            "value": 18.21,
            "unit": "fpr_%",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate - total_cycles",
            "value": 10000,
            "unit": "total_cycles",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_MemoryLeak",
            "value": 0.008303,
            "unit": "ns/op\t         1.000 cycles_to_detect\t        50.00 leak_rate_per_cycle\t      1000 ms_to_detect\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_MemoryLeak - ns/op",
            "value": 0.008303,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_MemoryLeak - cycles_to_detect",
            "value": 1,
            "unit": "cycles_to_detect",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_MemoryLeak - leak_rate_per_cycle",
            "value": 50,
            "unit": "leak_rate_per_cycle",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_MemoryLeak - ms_to_detect",
            "value": 1000,
            "unit": "ms_to_detect",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_MemoryLeak - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_MemoryLeak - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_CPUBurn",
            "value": 0.007308,
            "unit": "ns/op\t       200.0 cycles_after_fault\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_CPUBurn - ns/op",
            "value": 0.007308,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_CPUBurn - cycles_after_fault",
            "value": 200,
            "unit": "cycles_after_fault",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_CPUBurn - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_CPUBurn - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle",
            "value": 3848,
            "unit": "ns/op\t   10600 B/op\t      12 allocs/op",
            "extra": "1519159 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - ns/op",
            "value": 3848,
            "unit": "ns/op",
            "extra": "1519159 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - B/op",
            "value": 10600,
            "unit": "B/op",
            "extra": "1519159 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "1519159 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert",
            "value": 32.72,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "183485660 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - ns/op",
            "value": 32.72,
            "unit": "ns/op",
            "extra": "183485660 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "183485660 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "183485660 times\n4 procs"
          }
        ]
      }
    ]
  }
}