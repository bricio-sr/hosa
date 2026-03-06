// +build ignore

#include <linux/bpf.h>
#include <linux/ptrace.h>
#include <bpf/bpf_helpers.h>

// Licença obrigatória. Sem isso o Kernel recusa injetar o código.
// Como discutimos, a camada C TEM que ser GPL.
char __license[] SEC("license") = "GPL";

// Estrutura do Mapa eBPF: Um array de 1 posição para a gente somar as métricas de forma atômica
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);   // Key 0 (Nossa única linha de dados temporária)
    __type(value, __u64); // Value: Contador total de bytes alocados
    __uint(max_entries, 1);
} memory_metrics SEC(".maps");

// O Programa eBPF que monitora a Syscall 'sys_brk' (alocação de memória do heap)
SEC("tracepoint/syscalls/sys_enter_brk")
int trace_sys_brk(struct trace_event_raw_sys_enter *ctx) {
    __u32 key = 0; // Posição fixa no nosso array
    __u64 *val;
    
    // Procura a chave no mapa eBPF
    val = bpf_map_lookup_elem(&memory_metrics, &key);
    if (val) {
        // Usa uma operação atômica pra somar. Como o kernel inteiro pode estar alocando memória
        // ao mesmo tempo, a soma atômica evita race conditions.
        __sync_fetch_and_add(val, 1); 
    }

    return 0; // Retorna 0 pro kernel seguir a vida dele
}