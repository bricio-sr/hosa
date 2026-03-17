// +build ignore

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

// Licença obrigatória. Sem isso o Kernel recusa injetar o código.
// A camada C TEM que ser GPL para usar as funções avançadas do eBPF.
char __license[] SEC("license") = "GPL";

// Estrutura clássica de mapas eBPF (Necessária já que não estamos usando BTF/Cilium)
// O seu parser Go vai ler exatamente esse bloco de memória no ELF
struct bpf_map_def {
    unsigned int type;
    unsigned int key_size;
    unsigned int value_size;
    unsigned int max_entries;
    unsigned int map_flags;
};

// Declaração do mapa na seção "maps" (sem o ponto). 
// Um array de 1 posição para somarmos as métricas de forma atômica.
struct bpf_map_def SEC("maps") memory_metrics = {
    .type = BPF_MAP_TYPE_ARRAY,
    .key_size = sizeof(__u32),
    .value_size = sizeof(__u64),
    .max_entries = 1,
};

// O Programa eBPF que monitora a Syscall 'sys_brk' (alocação de memória do heap)
SEC("tracepoint/syscalls/sys_enter_brk")
int trace_sys_brk(void *ctx) { // void *ctx elimina o warning de struct não declarada
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