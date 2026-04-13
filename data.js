window.BENCHMARK_DATA = {
  "lastUpdate": 1776083963690,
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
      },
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
          "id": "2013d371f899c847b6f216c92a15b031a66edd59",
          "message": "feat(docs): Whitepaper v2.2",
          "timestamp": "2026-04-01T09:52:37-03:00",
          "tree_id": "108e843d82b4f97061c204afb92cb2e61581d7f1",
          "url": "https://github.com/bricio-sr/hosa/commit/2013d371f899c847b6f216c92a15b031a66edd59"
        },
        "date": 1775048010970,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkAnalyzeCycle",
            "value": 3423,
            "unit": "ns/op\t   10224 B/op\t       6 allocs/op",
            "extra": "1779700 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - ns/op",
            "value": 3423,
            "unit": "ns/op",
            "extra": "1779700 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - B/op",
            "value": 10224,
            "unit": "B/op",
            "extra": "1779700 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "1779700 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution",
            "value": 0.03544,
            "unit": "ns/op\t      2384 p50_ns\t     72595 p999_ns\t     25157 p99_ns\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - ns/op",
            "value": 0.03544,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p50_ns",
            "value": 2384,
            "unit": "p50_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p999_ns",
            "value": 72595,
            "unit": "p999_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p99_ns",
            "value": 25157,
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
            "value": 57.84,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkWelfordUpdate - ns/op",
            "value": 57.84,
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
            "value": 148.6,
            "unit": "ns/op\t     136 B/op\t       4 allocs/op",
            "extra": "39876280 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - ns/op",
            "value": 148.6,
            "unit": "ns/op",
            "extra": "39876280 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - B/op",
            "value": 136,
            "unit": "B/op",
            "extra": "39876280 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "39876280 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate",
            "value": 0.04141,
            "unit": "ns/op\t      1821 false_positives\t        18.21 fpr_%\t     10000 total_cycles\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate - ns/op",
            "value": 0.04141,
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
            "value": 0.008656,
            "unit": "ns/op\t         1.000 cycles_to_detect\t        50.00 leak_rate_per_cycle\t      1000 ms_to_detect\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_MemoryLeak - ns/op",
            "value": 0.008656,
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
            "value": 0.007584,
            "unit": "ns/op\t       200.0 cycles_after_fault\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_CPUBurn - ns/op",
            "value": 0.007584,
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
            "value": 4097,
            "unit": "ns/op\t   10600 B/op\t      12 allocs/op",
            "extra": "1462198 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - ns/op",
            "value": 4097,
            "unit": "ns/op",
            "extra": "1462198 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - B/op",
            "value": 10600,
            "unit": "B/op",
            "extra": "1462198 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "1462198 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert",
            "value": 16.56,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "362584306 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - ns/op",
            "value": 16.56,
            "unit": "ns/op",
            "extra": "362584306 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "362584306 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "362584306 times\n4 procs"
          }
        ]
      },
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
          "id": "c6a934fa4823657607fd9aacaaf1a978e983b61d",
          "message": "Merge pull request #2 from GiovanniYuriMita/main\n\nfix(build): compile eBPF object in Makefile; tidy go.mod for toolchain",
          "timestamp": "2026-04-09T20:08:24-03:00",
          "tree_id": "2866b9bb12c93c66e1193a21cb9a728aa77f905c",
          "url": "https://github.com/bricio-sr/hosa/commit/c6a934fa4823657607fd9aacaaf1a978e983b61d"
        },
        "date": 1775776169699,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkAnalyzeCycle",
            "value": 4354,
            "unit": "ns/op\t   10224 B/op\t       6 allocs/op",
            "extra": "1411830 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - ns/op",
            "value": 4354,
            "unit": "ns/op",
            "extra": "1411830 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - B/op",
            "value": 10224,
            "unit": "B/op",
            "extra": "1411830 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "1411830 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution",
            "value": 0.03975,
            "unit": "ns/op\t      2385 p50_ns\t     82594 p999_ns\t     28584 p99_ns\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - ns/op",
            "value": 0.03975,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p50_ns",
            "value": 2385,
            "unit": "p50_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p999_ns",
            "value": 82594,
            "unit": "p999_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p99_ns",
            "value": 28584,
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
            "value": 57.78,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkWelfordUpdate - ns/op",
            "value": 57.78,
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
            "value": 152.6,
            "unit": "ns/op\t     136 B/op\t       4 allocs/op",
            "extra": "39335444 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - ns/op",
            "value": 152.6,
            "unit": "ns/op",
            "extra": "39335444 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - B/op",
            "value": 136,
            "unit": "B/op",
            "extra": "39335444 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "39335444 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate",
            "value": 0.05214,
            "unit": "ns/op\t      1821 false_positives\t        18.21 fpr_%\t     10000 total_cycles\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate - ns/op",
            "value": 0.05214,
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
            "value": 0.01073,
            "unit": "ns/op\t         1.000 cycles_to_detect\t        50.00 leak_rate_per_cycle\t      1000 ms_to_detect\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_MemoryLeak - ns/op",
            "value": 0.01073,
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
            "value": 0.01008,
            "unit": "ns/op\t       200.0 cycles_after_fault\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_CPUBurn - ns/op",
            "value": 0.01008,
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
            "value": 5266,
            "unit": "ns/op\t   10599 B/op\t      12 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - ns/op",
            "value": 5266,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - B/op",
            "value": 10599,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert",
            "value": 16.22,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "372917745 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - ns/op",
            "value": 16.22,
            "unit": "ns/op",
            "extra": "372917745 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "372917745 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "372917745 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "f@bricio.dev.br",
            "name": "Fabricio Amorim",
            "username": "bricio-sr"
          },
          "committer": {
            "email": "f@bricio.dev.br",
            "name": "Fabricio Amorim",
            "username": "bricio-sr"
          },
          "distinct": true,
          "id": "63383fe3e8dcb7ea5338391e908e9388142c2901",
          "message": "chore(actions): Repairing makefile to run bench on actions.",
          "timestamp": "2026-04-13T09:36:53-03:00",
          "tree_id": "72115e9b8d766ddcfd4ee39c31ed0d277f604866",
          "url": "https://github.com/bricio-sr/hosa/commit/63383fe3e8dcb7ea5338391e908e9388142c2901"
        },
        "date": 1776083962701,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkAnalyzeCycle",
            "value": 3398,
            "unit": "ns/op\t   10224 B/op\t       6 allocs/op",
            "extra": "1739704 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - ns/op",
            "value": 3398,
            "unit": "ns/op",
            "extra": "1739704 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - B/op",
            "value": 10224,
            "unit": "B/op",
            "extra": "1739704 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "1739704 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution",
            "value": 0.0357,
            "unit": "ns/op\t      2405 p50_ns\t     83787 p999_ns\t     24517 p99_ns\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - ns/op",
            "value": 0.0357,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p50_ns",
            "value": 2405,
            "unit": "p50_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p999_ns",
            "value": 83787,
            "unit": "p999_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p99_ns",
            "value": 24517,
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
            "value": 59.89,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkWelfordUpdate - ns/op",
            "value": 59.89,
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
            "value": 157.3,
            "unit": "ns/op\t     136 B/op\t       4 allocs/op",
            "extra": "37998529 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - ns/op",
            "value": 157.3,
            "unit": "ns/op",
            "extra": "37998529 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - B/op",
            "value": 136,
            "unit": "B/op",
            "extra": "37998529 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "37998529 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate",
            "value": 0.04108,
            "unit": "ns/op\t      1821 false_positives\t        18.21 fpr_%\t     10000 total_cycles\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate - ns/op",
            "value": 0.04108,
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
            "value": 0.008234,
            "unit": "ns/op\t         1.000 cycles_to_detect\t        50.00 leak_rate_per_cycle\t      1000 ms_to_detect\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_MemoryLeak - ns/op",
            "value": 0.008234,
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
            "value": 0.00826,
            "unit": "ns/op\t       200.0 cycles_after_fault\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_CPUBurn - ns/op",
            "value": 0.00826,
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
            "value": 4368,
            "unit": "ns/op\t   10600 B/op\t      12 allocs/op",
            "extra": "1419469 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - ns/op",
            "value": 4368,
            "unit": "ns/op",
            "extra": "1419469 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - B/op",
            "value": 10600,
            "unit": "B/op",
            "extra": "1419469 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "1419469 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert",
            "value": 16.11,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "372102609 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - ns/op",
            "value": 16.11,
            "unit": "ns/op",
            "extra": "372102609 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "372102609 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "372102609 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragComputation",
            "value": 22509,
            "unit": "ns/op\t    3736 B/op\t      16 allocs/op",
            "extra": "352903 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragComputation - ns/op",
            "value": 22509,
            "unit": "ns/op",
            "extra": "352903 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragComputation - B/op",
            "value": 3736,
            "unit": "B/op",
            "extra": "352903 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragComputation - allocs/op",
            "value": 16,
            "unit": "allocs/op",
            "extra": "352903 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragComputationLatencyDistribution",
            "value": 0.02067,
            "unit": "ns/op\t     18936 p50_ns\t    163947 p999_ns\t     42269 p99_ns\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragComputationLatencyDistribution - ns/op",
            "value": 0.02067,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragComputationLatencyDistribution - p50_ns",
            "value": 18936,
            "unit": "p50_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragComputationLatencyDistribution - p999_ns",
            "value": 163947,
            "unit": "p999_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragComputationLatencyDistribution - p99_ns",
            "value": 42269,
            "unit": "p99_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragComputationLatencyDistribution - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragComputationLatencyDistribution - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyCalculation",
            "value": 242.9,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "24682605 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyCalculation - ns/op",
            "value": 242.9,
            "unit": "ns/op",
            "extra": "24682605 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyCalculation - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "24682605 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyCalculation - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "24682605 times\n4 procs"
          },
          {
            "name": "BenchmarkFragmentationMonitorSample",
            "value": 22627,
            "unit": "ns/op\t    3736 B/op\t      16 allocs/op",
            "extra": "337114 times\n4 procs"
          },
          {
            "name": "BenchmarkFragmentationMonitorSample - ns/op",
            "value": 22627,
            "unit": "ns/op",
            "extra": "337114 times\n4 procs"
          },
          {
            "name": "BenchmarkFragmentationMonitorSample - B/op",
            "value": 3736,
            "unit": "B/op",
            "extra": "337114 times\n4 procs"
          },
          {
            "name": "BenchmarkFragmentationMonitorSample - allocs/op",
            "value": 16,
            "unit": "allocs/op",
            "extra": "337114 times\n4 procs"
          },
          {
            "name": "BenchmarkCPUSetString/contiguous",
            "value": 85.37,
            "unit": "ns/op\t      72 B/op\t       2 allocs/op",
            "extra": "69418501 times\n4 procs"
          },
          {
            "name": "BenchmarkCPUSetString/contiguous - ns/op",
            "value": 85.37,
            "unit": "ns/op",
            "extra": "69418501 times\n4 procs"
          },
          {
            "name": "BenchmarkCPUSetString/contiguous - B/op",
            "value": 72,
            "unit": "B/op",
            "extra": "69418501 times\n4 procs"
          },
          {
            "name": "BenchmarkCPUSetString/contiguous - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "69418501 times\n4 procs"
          },
          {
            "name": "BenchmarkCPUSetString/fragmented",
            "value": 200.1,
            "unit": "ns/op\t     120 B/op\t       4 allocs/op",
            "extra": "29748894 times\n4 procs"
          },
          {
            "name": "BenchmarkCPUSetString/fragmented - ns/op",
            "value": 200.1,
            "unit": "ns/op",
            "extra": "29748894 times\n4 procs"
          },
          {
            "name": "BenchmarkCPUSetString/fragmented - B/op",
            "value": 120,
            "unit": "B/op",
            "extra": "29748894 times\n4 procs"
          },
          {
            "name": "BenchmarkCPUSetString/fragmented - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "29748894 times\n4 procs"
          },
          {
            "name": "BenchmarkSurvivalDecisionCycle",
            "value": 3524,
            "unit": "ns/op\t   10224 B/op\t       6 allocs/op",
            "extra": "1696357 times\n4 procs"
          },
          {
            "name": "BenchmarkSurvivalDecisionCycle - ns/op",
            "value": 3524,
            "unit": "ns/op",
            "extra": "1696357 times\n4 procs"
          },
          {
            "name": "BenchmarkSurvivalDecisionCycle - B/op",
            "value": 10224,
            "unit": "B/op",
            "extra": "1696357 times\n4 procs"
          },
          {
            "name": "BenchmarkSurvivalDecisionCycle - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "1696357 times\n4 procs"
          },
          {
            "name": "BenchmarkSurvivalDecisionLatencyDistribution",
            "value": 0.03713,
            "unit": "ns/op\t      2414 p50_ns\t     89137 p999_ns\t     28784 p99_ns\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSurvivalDecisionLatencyDistribution - ns/op",
            "value": 0.03713,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSurvivalDecisionLatencyDistribution - p50_ns",
            "value": 2414,
            "unit": "p50_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSurvivalDecisionLatencyDistribution - p999_ns",
            "value": 89137,
            "unit": "p999_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSurvivalDecisionLatencyDistribution - p99_ns",
            "value": 28784,
            "unit": "p99_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSurvivalDecisionLatencyDistribution - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkSurvivalDecisionLatencyDistribution - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_SurvivalCascade",
            "value": 702936,
            "unit": "ns/op\t         0 cycles_to_survival\t 2051796 B/op\t    1093 allocs/op",
            "extra": "8228 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_SurvivalCascade - ns/op",
            "value": 702936,
            "unit": "ns/op",
            "extra": "8228 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_SurvivalCascade - cycles_to_survival",
            "value": 0,
            "unit": "cycles_to_survival",
            "extra": "8228 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_SurvivalCascade - B/op",
            "value": 2051796,
            "unit": "B/op",
            "extra": "8228 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_SurvivalCascade - allocs/op",
            "value": 1093,
            "unit": "allocs/op",
            "extra": "8228 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyProfiles/all_order0_fragmented",
            "value": 24.5,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "244970841 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyProfiles/all_order0_fragmented - ns/op",
            "value": 24.5,
            "unit": "ns/op",
            "extra": "244970841 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyProfiles/all_order0_fragmented - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "244970841 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyProfiles/all_order0_fragmented - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "244970841 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyProfiles/all_order10_consolidated",
            "value": 25.07,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "239122460 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyProfiles/all_order10_consolidated - ns/op",
            "value": 25.07,
            "unit": "ns/op",
            "extra": "239122460 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyProfiles/all_order10_consolidated - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "239122460 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyProfiles/all_order10_consolidated - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "239122460 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyProfiles/uniform_max_entropy",
            "value": 244.2,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "24605080 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyProfiles/uniform_max_entropy - ns/op",
            "value": 244.2,
            "unit": "ns/op",
            "extra": "24605080 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyProfiles/uniform_max_entropy - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "24605080 times\n4 procs"
          },
          {
            "name": "BenchmarkHFragEntropyProfiles/uniform_max_entropy - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "24605080 times\n4 procs"
          }
        ]
      }
    ]
  }
}