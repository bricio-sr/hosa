window.BENCHMARK_DATA = {
  "lastUpdate": 1774623845279,
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
          "id": "c88fd13d09ea4b82bcf3ff0120c1327817e56cfb",
          "message": "fix(action): Change gobench to go on tools",
          "timestamp": "2026-03-27T11:21:06-03:00",
          "tree_id": "58ec57e28a80240a4bfa6e4bf87fdbe892a4d1e6",
          "url": "https://github.com/bricio-sr/hosa/commit/c88fd13d09ea4b82bcf3ff0120c1327817e56cfb"
        },
        "date": 1774621323390,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkAnalyzeCycle",
            "value": 3370,
            "unit": "ns/op\t   10224 B/op\t       6 allocs/op",
            "extra": "1762064 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - ns/op",
            "value": 3370,
            "unit": "ns/op",
            "extra": "1762064 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - B/op",
            "value": 10224,
            "unit": "B/op",
            "extra": "1762064 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "1762064 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution",
            "value": 0.03533,
            "unit": "ns/op\t      2355 p50_ns\t     89928 p999_ns\t     23095 p99_ns\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - ns/op",
            "value": 0.03533,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p50_ns",
            "value": 2355,
            "unit": "p50_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p999_ns",
            "value": 89928,
            "unit": "p999_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p99_ns",
            "value": 23095,
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
            "value": 57.61,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "100000000 times\n4 procs"
          },
          {
            "name": "BenchmarkWelfordUpdate - ns/op",
            "value": 57.61,
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
            "value": 148.4,
            "unit": "ns/op\t     136 B/op\t       4 allocs/op",
            "extra": "39690416 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - ns/op",
            "value": 148.4,
            "unit": "ns/op",
            "extra": "39690416 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - B/op",
            "value": 136,
            "unit": "B/op",
            "extra": "39690416 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "39690416 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate",
            "value": 0.03875,
            "unit": "ns/op\t      1821 false_positives\t        18.21 fpr_%\t     10000 total_cycles\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate - ns/op",
            "value": 0.03875,
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
            "value": 0.00811,
            "unit": "ns/op\t         1.000 cycles_to_detect\t        50.00 leak_rate_per_cycle\t      1000 ms_to_detect\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_MemoryLeak - ns/op",
            "value": 0.00811,
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
            "value": 0.008104,
            "unit": "ns/op\t       200.0 cycles_after_fault\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_CPUBurn - ns/op",
            "value": 0.008104,
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
            "value": 4034,
            "unit": "ns/op\t   10600 B/op\t      12 allocs/op",
            "extra": "1472721 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - ns/op",
            "value": 4034,
            "unit": "ns/op",
            "extra": "1472721 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - B/op",
            "value": 10600,
            "unit": "B/op",
            "extra": "1472721 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "1472721 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert",
            "value": 16.06,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "374264187 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - ns/op",
            "value": 16.06,
            "unit": "ns/op",
            "extra": "374264187 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "374264187 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "374264187 times\n4 procs"
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
          "id": "c88fd13d09ea4b82bcf3ff0120c1327817e56cfb",
          "message": "fix(action): Change gobench to go on tools",
          "timestamp": "2026-03-27T11:21:06-03:00",
          "tree_id": "58ec57e28a80240a4bfa6e4bf87fdbe892a4d1e6",
          "url": "https://github.com/bricio-sr/hosa/commit/c88fd13d09ea4b82bcf3ff0120c1327817e56cfb"
        },
        "date": 1774623844651,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkAnalyzeCycle",
            "value": 3229,
            "unit": "ns/op\t   10224 B/op\t       6 allocs/op",
            "extra": "1907110 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - ns/op",
            "value": 3229,
            "unit": "ns/op",
            "extra": "1907110 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - B/op",
            "value": 10224,
            "unit": "B/op",
            "extra": "1907110 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeCycle - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "1907110 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution",
            "value": 0.03422,
            "unit": "ns/op\t      2323 p50_ns\t     82925 p999_ns\t     25348 p99_ns\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - ns/op",
            "value": 0.03422,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p50_ns",
            "value": 2323,
            "unit": "p50_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p999_ns",
            "value": 82925,
            "unit": "p999_ns",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkAnalyzeLatencyDistribution - p99_ns",
            "value": 25348,
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
            "value": 60.92,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "99717676 times\n4 procs"
          },
          {
            "name": "BenchmarkWelfordUpdate - ns/op",
            "value": 60.92,
            "unit": "ns/op",
            "extra": "99717676 times\n4 procs"
          },
          {
            "name": "BenchmarkWelfordUpdate - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "99717676 times\n4 procs"
          },
          {
            "name": "BenchmarkWelfordUpdate - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "99717676 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation",
            "value": 140.2,
            "unit": "ns/op\t     136 B/op\t       4 allocs/op",
            "extra": "42196520 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - ns/op",
            "value": 140.2,
            "unit": "ns/op",
            "extra": "42196520 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - B/op",
            "value": 136,
            "unit": "B/op",
            "extra": "42196520 times\n4 procs"
          },
          {
            "name": "BenchmarkMahalanobisCalculation - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "42196520 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate",
            "value": 0.03752,
            "unit": "ns/op\t      1821 false_positives\t        18.21 fpr_%\t     10000 total_cycles\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkFalsePositiveRate - ns/op",
            "value": 0.03752,
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
            "value": 0.007397,
            "unit": "ns/op\t         1.000 cycles_to_detect\t        50.00 leak_rate_per_cycle\t      1000 ms_to_detect\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_MemoryLeak - ns/op",
            "value": 0.007397,
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
            "value": 0.008051,
            "unit": "ns/op\t       200.0 cycles_after_fault\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDetectionRate_CPUBurn - ns/op",
            "value": 0.008051,
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
            "value": 3787,
            "unit": "ns/op\t   10600 B/op\t      12 allocs/op",
            "extra": "1569144 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - ns/op",
            "value": 3787,
            "unit": "ns/op",
            "extra": "1569144 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - B/op",
            "value": 10600,
            "unit": "B/op",
            "extra": "1569144 times\n4 procs"
          },
          {
            "name": "BenchmarkAllocationsPerCycle - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "1569144 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert",
            "value": 17.75,
            "unit": "ns/op\t       0 B/op\t       0 allocs/op",
            "extra": "338526811 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - ns/op",
            "value": 17.75,
            "unit": "ns/op",
            "extra": "338526811 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "338526811 times\n4 procs"
          },
          {
            "name": "BenchmarkRingBufferInsert - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "338526811 times\n4 procs"
          }
        ]
      }
    ]
  }
}