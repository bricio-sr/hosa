.PHONY: generate generate-phase2 test build bench bench-quick bench-phase2 bench-phase2-quick build-bpf build-phase2

BPF_CLANG ?= clang
BPF_CFLAGS := -O2 -g -Wall -target bpf

# --- Phase 1: eBPF sensor probes ---
internal/bpf/sensors.o: internal/bpf/sensors.c
	@echo ">> Compiling eBPF (sensors.o)..."
	$(BPF_CLANG) $(BPF_CFLAGS) -c $< -o $@

build-bpf: internal/bpf/sensors.o

generate: internal/bpf/sensors.o
	@echo ">> Running go generate (Phase 1: sensors)..."
	go generate ./internal/bpf/...
	@echo ">> eBPF bytecode ready"

# --- Phase 2: sched_ext survival scheduler ---
# Requires: Linux >= 6.11 with CONFIG_SCHED_CLASS_EXT=y
# Requires: clang >= 16 with BPF target support
internal/bpf/survival_scheduler.o: internal/bpf/survival_scheduler.c
	@echo ">> Compiling sched_ext survival scheduler (Phase 2)..."
	@echo "   Requires: Linux >= 6.11 with CONFIG_SCHED_CLASS_EXT=y"
	$(BPF_CLANG) $(BPF_CFLAGS) \
		-I/usr/include \
		-I/usr/include/bpf \
		-D__TARGET_ARCH_x86 \
		-Wno-unused-value -Wno-pointer-sign \
		-c $< -o $@
	@echo "   Output: $@"

generate-phase2: internal/bpf/survival_scheduler.o

# --- Core Commands ---
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