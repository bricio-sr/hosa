# Makefile
.PHONY: generate test build

generate:
	@echo ">> Compilando o Kernel C para o bytecode eBPF..."
	go generate ./internal/bpf/...

test:
	go test ./... -v

build: generate
	go build -o hosa_agent cmd/hosa/main.go