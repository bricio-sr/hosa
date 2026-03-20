// +build ignore

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

char __license[] SEC("license") = "GPL";

// Índices do vetor de estado — devem ser sincronizados com sensor/collector.go
#define IDX_CPU_RUN_QUEUE   0
#define IDX_MEM_BRK_CALLS   1
#define IDX_MEM_PAGE_FAULTS 2
#define IDX_IO_BLOCK_OPS    3
#define NUM_VARS            4

// Formato legado de mapa (sem BTF) — compatível com o nosso parser ELF.
struct bpf_map_def {
    unsigned int type;
    unsigned int key_size;
    unsigned int value_size;
    unsigned int max_entries;
    unsigned int map_flags;
};

// Um array de NUM_VARS posições. Cada posição é um contador atômico uint64.
// O layout é: [cpu_run_queue, mem_brk_calls, mem_page_faults, io_block_ops]
struct bpf_map_def SEC("maps") hosa_metrics = {
    .type        = BPF_MAP_TYPE_ARRAY,
    .key_size    = sizeof(__u32),
    .value_size  = sizeof(__u64),
    .max_entries = NUM_VARS,
};

// --- Probe 1: CPU run queue depth ---
// Dispara a cada wakeup de processo — proxy direto de pressão de scheduler.
// Quanto mais processos esperando CPU, maior o run_queue.
SEC("tracepoint/sched/sched_wakeup")
int probe_sched_wakeup(void *ctx) {
    __u32 key = IDX_CPU_RUN_QUEUE;
    __u64 *val = bpf_map_lookup_elem(&hosa_metrics, &key);
    if (val)
        __sync_fetch_and_add(val, 1);
    return 0;
}

// --- Probe 2: Alocações de heap (sys_brk) ---
// Mantido da fase anterior. Conta chamadas brk() — proxy de alocação de heap.
SEC("tracepoint/syscalls/sys_enter_brk")
int probe_sys_brk(void *ctx) {
    __u32 key = IDX_MEM_BRK_CALLS;
    __u64 *val = bpf_map_lookup_elem(&hosa_metrics, &key);
    if (val)
        __sync_fetch_and_add(val, 1);
    return 0;
}

// --- Probe 3: Page faults ---
// Dispara a cada page fault do kernel. Page faults altos indicam pressão real
// de memória — swap ativo, working set maior que RAM disponível.
SEC("tracepoint/exceptions/page_fault_kernel")
int probe_page_fault(void *ctx) {
    __u32 key = IDX_MEM_PAGE_FAULTS;
    __u64 *val = bpf_map_lookup_elem(&hosa_metrics, &key);
    if (val)
        __sync_fetch_and_add(val, 1);
    return 0;
}

// --- Probe 4: Operações de bloco emitidas ---
// Dispara a cada request de I/O de bloco enviado ao device driver.
// I/O alto junto com page_faults altos = swap ativo (assinatura de memory leak).
SEC("tracepoint/block/block_rq_issue")
int probe_block_rq(void *ctx) {
    __u32 key = IDX_IO_BLOCK_OPS;
    __u64 *val = bpf_map_lookup_elem(&hosa_metrics, &key);
    if (val)
        __sync_fetch_and_add(val, 1);
    return 0;
}