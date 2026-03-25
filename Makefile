.PHONY: generate test build bench bench-quick

generate:
	@echo ">> Compiling eBPF kernel bytecode..."
	go generate ./internal/bpf/...

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