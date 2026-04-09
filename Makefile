.PHONY: generate test build bench bench-quick build-bpf

BPF_CLANG ?= clang
BPF_CFLAGS := -O2 -g -Wall -target bpf

internal/bpf/sensors.o: internal/bpf/sensors.c
	@echo ">> Compiling eBPF (sensors.o)..."
	$(BPF_CLANG) $(BPF_CFLAGS) -c $< -o $@

generate: internal/bpf/sensors.o
	@echo ">> eBPF bytecode ready"

build-bpf: internal/bpf/sensors.o

test:
	go test ./... -v

build: generate
	go build -o hosa_agent cmd/hosa/main.go

bench:
	go test -v -run=^$$ -bench=. -benchmem -benchtime=5s \
		./internal/bench/... \
		| tee bench_results.txt
	@echo ""
	@echo "Results saved to bench_results.txt"

bench-quick:
	go test -v -run=^$$ -bench=. -benchmem -benchtime=1s \
		./internal/bench/...