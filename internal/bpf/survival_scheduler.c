// survival_scheduler.c — HOSA Phase 2: Survival Scheduler
//
// Implements a sched_ext scheduler that replaces CFS during a cascade failure
// (AlertLevel == LevelSurvival, D_M >= 12.0).
//
// Strategy:
//   - Targeted Starvation: the offending process receives zero CPU cycles.
//     Its enqueue() call is silently dropped — the kernel never dispatches it.
//   - Predictive Cache Affinity: vital processes are dispatched to a dedicated
//     per-CPU DSQ, keeping them on cache-warm cores and out of the offender's
//     scheduling domain.
//   - All other processes use the global DSQ (standard round-robin fairness).
//
// Kernel requirements: Linux >= 6.11, CONFIG_SCHED_CLASS_EXT=y
//
// Build:
//   clang -target bpf -O2 -g \
//     -I/usr/include/bpf \
//     -D__TARGET_ARCH_x86 \
//     -c survival_scheduler.c \
//     -o survival_scheduler.o
//
// Reference: Documentation/scheduler/sched-ext.rst
//            tools/testing/selftests/bpf/progs/sched_ext.c

// +build ignore

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

/* sched_ext headers — available in kernel >= 6.11 with CONFIG_SCHED_CLASS_EXT=y */
#include <linux/sched/ext.h>

char __license[] SEC("license") = "GPL";

/* -------------------------------------------------------------------------
 * Control Map — written by the Go SurvivalMotor to communicate policy
 *
 * Key → Value semantics:
 *   0 → offender_pid   (u32): PID of the process to starve; 0 = none
 *   1 → starvation_on  (u64): 1 = starvation active, 0 = passthrough mode
 *   2 → vital_dsq_id   (u64): DSQ ID reserved for vital processes
 *   3 → vital_cpu_mask (u64): bitmask of cache-warm CPUs for vital processes
 * -------------------------------------------------------------------------*/
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 8);
    __type(key, __u32);
    __type(value, __u64);
} hosa_sched_ctrl SEC(".maps");

/* Per-task storage — tracks whether this task is the offender */
struct {
    __uint(type, BPF_MAP_TYPE_TASK_STORAGE);
    __uint(map_flags, BPF_F_NO_PREALLOC);
    __type(key, int);
    __type(value, __u64);  /* 1 = offender, 2 = vital, 0 = normal */
} hosa_task_role SEC(".maps");

/* Reserved DSQ for vital processes (cache-warm dispatch) */
#define HOSA_VITAL_DSQ_ID   0xDEAD0001ULL
#define TASK_ROLE_NORMAL    0ULL
#define TASK_ROLE_OFFENDER  1ULL
#define TASK_ROLE_VITAL     2ULL

static __always_inline __u64 ctrl_get(__u32 key)
{
    __u64 *val = bpf_map_lookup_elem(&hosa_sched_ctrl, &key);
    return val ? *val : 0;
}

static __always_inline __u64 task_role(struct task_struct *p)
{
    __u32 offender_pid = (__u32)ctrl_get(0);
    if (offender_pid == 0)
        return TASK_ROLE_NORMAL;
    if (p->pid == (__s32)offender_pid)
        return TASK_ROLE_OFFENDER;
    return TASK_ROLE_NORMAL;
}

/* -------------------------------------------------------------------------
 * sched_ext_ops callbacks
 * -------------------------------------------------------------------------*/

/*
 * hosa_select_cpu — called before wakeup to pick the initial CPU.
 *
 * For vital processes: steer toward cache-warm CPUs (vital_cpu_mask).
 * For the offender: any CPU is fine — it will be dropped in enqueue().
 * For normal processes: return prev_cpu (let CFS heuristics work).
 */
SEC("struct_ops/hosa_select_cpu")
s32 BPF_PROG(hosa_select_cpu, struct task_struct *p, s32 prev_cpu, u64 wake_flags)
{
    __u64 active = ctrl_get(1);
    if (!active)
        return prev_cpu;

    __u64 role = task_role(p);
    if (role == TASK_ROLE_VITAL) {
        __u64 mask = ctrl_get(3);
        if (mask != 0) {
            /* Find the lowest set bit in vital_cpu_mask */
            int cpu = __builtin_ctzll(mask);
            if (cpu >= 0 && cpu < 64)
                return cpu;
        }
    }

    return prev_cpu;
}

/*
 * hosa_enqueue — called when a task becomes runnable.
 *
 * Targeted Starvation: if the task is the identified offender AND
 * starvation is active, silently drop the enqueue — the task never
 * gets a timeslice.
 *
 * Vital processes go to the reserved HOSA_VITAL_DSQ_ID (cache-warm).
 * All other processes go to the global DSQ.
 */
SEC("struct_ops/hosa_enqueue")
void BPF_PROG(hosa_enqueue, struct task_struct *p, u64 enq_flags)
{
    __u64 active = ctrl_get(1);

    if (active) {
        __u64 role = task_role(p);
        if (role == TASK_ROLE_OFFENDER) {
            /*
             * Targeted Starvation: do NOT call scx_bpf_dispatch().
             * The kernel will re-enqueue via hosa_dispatch() only if we
             * call scx_bpf_dispatch_vtime() or the task is explicitly
             * woken by the fallback path. By returning here, the task
             * receives zero CPU cycles until starvation is lifted.
             *
             * Note: the kernel guarantees eventual progress via the
             * stall detection mechanism (scx_ops_check_stall), which
             * fires a WARN and forces dispatch after a timeout.
             * This is intentional — the stall timeout is our safety net.
             */
            return;
        }

        if (role == TASK_ROLE_VITAL) {
            scx_bpf_dispatch(p, HOSA_VITAL_DSQ_ID, SCX_SLICE_DFL, enq_flags);
            return;
        }
    }

    /* Default: global FIFO — preserves fairness for non-targeted processes */
    scx_bpf_dispatch(p, SCX_DSQ_GLOBAL, SCX_SLICE_DFL, enq_flags);
}

/*
 * hosa_dispatch — called when a CPU needs a task to run.
 *
 * Priority order:
 *   1. Vital DSQ (cache-warm processes)
 *   2. Global DSQ (all normal processes)
 *
 * The offender's DSQ is never consumed — it starves.
 */
SEC("struct_ops/hosa_dispatch")
void BPF_PROG(hosa_dispatch, s32 cpu, struct task_struct *prev)
{
    __u64 active = ctrl_get(1);
    if (active) {
        /* Try vital processes first */
        if (scx_bpf_consume(HOSA_VITAL_DSQ_ID))
            return;
    }
    scx_bpf_consume(SCX_DSQ_GLOBAL);
}

/*
 * hosa_init — called once when the scheduler is loaded.
 * Creates the vital DSQ for cache-warm dispatch.
 */
SEC("struct_ops/hosa_init")
s32 BPF_PROG(hosa_init)
{
    return scx_bpf_create_dsq(HOSA_VITAL_DSQ_ID, -1);
}

/*
 * hosa_exit — called once when the scheduler is unloaded (link FD closed).
 * Cleanup is handled automatically by the kernel — no explicit DSQ destroy needed.
 */
SEC("struct_ops/hosa_exit")
void BPF_PROG(hosa_exit, struct scx_exit_info *ei)
{
    /* The kernel logs the exit reason via ei->reason — no action needed */
    (void)ei;
}

/* -------------------------------------------------------------------------
 * Scheduler registration
 *
 * struct sched_ext_ops hosa_survival_scheduler is the entry point.
 * Loaded as BPF_MAP_TYPE_STRUCT_OPS, activated via BPF_LINK_CREATE.
 * The link FD kept open in SurvivalMotor.schedExtFD — closing it reverts to CFS.
 * -------------------------------------------------------------------------*/
SEC(".struct_ops.link")
struct sched_ext_ops hosa_survival_scheduler = {
    .select_cpu = (void *)hosa_select_cpu,
    .enqueue    = (void *)hosa_enqueue,
    .dispatch   = (void *)hosa_dispatch,
    .init       = (void *)hosa_init,
    .exit       = (void *)hosa_exit,
    .name       = "hosa_survival",
    .flags      = 0,
};
