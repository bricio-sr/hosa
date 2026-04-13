<<<<<<< HEAD
.PHONY: generate test build bench bench-quick build-bpf

BPF_CLANG ?= clang
BPF_CFLAGS := -O2 -g -Wall -target bpf

internal/bpf/sensors.o: internal/bpf/sensors.c
	@echo ">> Compiling eBPF (sensors.o)..."
	$(BPF_CLANG) $(BPF_CFLAGS) -c $< -o $@

generate: internal/bpf/sensors.o
	@echo ">> eBPF bytecode ready"

build-bpf: internal/bpf/sensors.o
=======
.PHONY: generate generate-phase2 test build bench bench-quick bench-phase2 bench-phase2-quick

# --- Phase 1: eBPF sensor probes ---
generate:
	@echo ">> Compiling eBPF kernel bytecode (Phase 1: sensors)..."
	go generate ./internal/bpf/...
>>>>>>> 023836c (feat(main): Phase 2 Alpha)

# --- Phase 2: sched_ext survival scheduler ---
# Requires: Linux >= 6.11 with CONFIG_SCHED_CLASS_EXT=y
# Requires: clang >= 16 with BPF target support
generate-phase2:
	@echo ">> Compiling sched_ext survival scheduler (Phase 2)..."
	@echo "   Requires: Linux >= 6.11 with CONFIG_SCHED_CLASS_EXT=y"
	clang -target bpf -O2 -g \
		-I/usr/include \
		-I/usr/include/bpf \
		-D__TARGET_ARCH_x86 \
		-Wall -Wno-unused-value -Wno-pointer-sign \
		-c internal/bpf/survival_scheduler.c \
		-o internal/bpf/survival_scheduler.o
	@echo "   Output: internal/bpf/survival_scheduler.o"

test:
	go test ./... -v

build: generate
	go build -o hosa_agent cmd/hosa/main.go

# Build with Phase 2 enabled (compiles sched_ext scheduler first)
build-phase2: generate generate-phase2
	go build -o hosa_agent cmd/hosa/main.go

# --- Phase 1 benchmarks ---
bench:
	go test -v -run=^$$ -bench=. -benchmem -benchtime=5s \
		./internal/bench/... \
		| tee bench_results.txt
	@echo ""
	@echo "Results saved to bench_results.txt"

bench-quick:
	go test -v -run=^$$ -bench=. -benchmem -benchtime=1s \
		./internal/bench/...

# --- Phase 2 benchmarks ---
# Measures: H_frag computation cost, survival decision latency, CPUSet formatting
bench-phase2:
	go test -v -run=^$$ \
		-bench='BenchmarkHFrag|BenchmarkFrag|BenchmarkCPUSet|BenchmarkSurvival|BenchmarkDetection' \
		-benchmem -benchtime=5s \
		./internal/bench/... \
		| tee bench_phase2_results.txt
	@echo ""
	@echo "Phase 2 results saved to bench_phase2_results.txt"

bench-phase2-quick:
	go test -v -run=^$$ \
		-bench='BenchmarkHFrag|BenchmarkFrag|BenchmarkCPUSet|BenchmarkSurvival|BenchmarkDetection' \
		-benchmem -benchtime=1s \
		./internal/bench/...