# HOSA Architecture — Deep Dive

> **This document covers the internal architecture of HOSA in depth.**
> For the mathematical model, see [`docs/math_model.md`](math_model.md).
> For the theoretical foundation, see the [Whitepaper v2.1](whitepaper.pdf).

---

## Table of Contents

- [Architectural Philosophy](#architectural-philosophy)
- [The Biological Analogy](#the-biological-analogy)
- [System Overview](#system-overview)
- [Layer 1 — Kernel Space (eBPF)](#layer-1--kernel-space-ebpf)
  - [Sensory Probes](#sensory-probes)
  - [Actuators](#actuators)
  - [Ring Buffer Transport](#ring-buffer-transport)
- [Layer 2 — User Space](#layer-2--user-space)
  - [The Sensory System (`internal/sensor`)](#the-sensory-system-internalsensor)
  - [The Limbic System (`internal/state`)](#the-limbic-system-internalstate)
  - [The Predictive Cortex (`internal/brain`)](#the-predictive-cortex-internalbrain)
  - [The Reflex Arc (`internal/motor`)](#the-reflex-arc-internalmotor)
  - [Opportunistic Communication](#opportunistic-communication)
- [Cross-Cutting Subsystems](#cross-cutting-subsystems)
  - [Hardware Proprioception — Warm-Up Phase](#hardware-proprioception--warm-up-phase)
  - [Graduated Response System (Levels 0–5)](#graduated-response-system-levels-05)
  - [Habituation — Baseline Drift](#habituation--baseline-drift)
  - [Thalamic Filter — Telemetry Suppression](#thalamic-filter--telemetry-suppression)
  - [Quarantine Modes by Environment](#quarantine-modes-by-environment)
- [Data Flow — End to End](#data-flow--end-to-end)
- [State Machine](#state-machine)
- [The Safelist](#the-safelist)
- [Key Design Decisions](#key-design-decisions)
- [Performance Characteristics](#performance-characteristics)
- [Security Posture](#security-posture)
- [Limitations](#limitations)

---

## Architectural Philosophy

HOSA is governed by five non-negotiable principles:

| Principle | Description |
|---|---|
| **Local Autonomy** | The complete detection and mitigation cycle must run without network connectivity, external APIs, or human intervention. |
| **Zero External Runtime Dependencies** | All dependencies are either inside the binary or inside the host kernel. Communication with external systems (orchestrators, dashboards) is *opportunistic* — performed when available, never required. |
| **Predictable Computational Footprint** | Memory is `O(1)` beyond the covariance matrix `O(n²)`. CPU consumption is configurable and bounded. The agent cannot become the cause of the problem it intends to solve. |
| **Graduated Response** | Mitigation is a spectrum, not a binary switch. Every action is proportional to the anomaly's severity *and its rate of change*. |
| **Decision Observability** | Every autonomous action is logged with its complete mathematical justification: the `D_M` value, derivative, threshold crossed, contributing dimensions, and action taken. The agent is fully auditable. |

---

## The Biological Analogy

HOSA's architecture maps directly to the human nervous system. This is not cosmetic — the analogy drives the design decisions.

| Biological System | HOSA Component | Module | Role |
|---|---|---|---|
| **Peripheral nervous system** (sensory receptors) | eBPF probes | `internal/bpf/sensors.c` | Collect raw signals from the kernel |
| **Afferent nerve fibers** | eBPF ring buffer | Kernel ↔ user space transport | Carry signals from sensors to cortex |
| **Spinal cord reflex arc** | Motor + response logic | `internal/motor/` | Execute containment without waiting for cortex |
| **Limbic system** (short-term memory) | Ring buffer baseline | `internal/state/memory.go` | Maintain recent state for derivative computation |
| **Prefrontal cortex** (pattern recognition) | Mahalanobis engine | `internal/brain/` | Compute deviation from baseline profile |
| **Thalamus** (signal gating) | Telemetry suppressor | Thalamic Filter | Suppress redundant signals in homeostasis |
| **Neuroplasticity** (habituation) | Baseline recalibration | Welford weight decay | Adapt to legitimate workload shifts |
| **Sympathetic nervous system** (threat response) | Graduated response | Level escalation logic | Escalate response proportional to threat |

The key insight from biology: **the spinal cord acts before the brain is notified.** HOSA's motor layer applies containment in milliseconds. The external orchestrator (the "brain") is notified *after* the reflex — via webhook, not before.

---

## System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         KERNEL SPACE (eBPF)                         │
│                                                                     │
│  ┌──────────────────┐  ┌──────────────────┐  ┌───────────────────┐  │
│  │  SENSORY PROBES  │  │  SENSORY PROBES  │  │    ACTUATORS      │  │
│  │  (tracepoints)   │  │  (kprobes, PSI)  │  │  (XDP / cgroup    │  │
│  │                  │  │                  │  │   controllers)    │  │
│  │ · sched_*        │  │ · mm_page_alloc  │  │                   │  │
│  │ · net_dev_*      │  │ · psi hooks      │  │ · memory.high     │  │
│  │ · block_rq_*     │  │ · tcp_*          │  │ · cpu.max         │  │
│  └────────┬─────────┘  └────────┬─────────┘  └────────▲──────────┘  │
│           │                     │                      │             │
│           ▼                     ▼                      │             │
│  ┌──────────────────────────────────────────┐          │             │
│  │            eBPF RING BUFFER              │          │             │
│  │         (low-latency transport)          │          │             │
│  └─────────────────────┬────────────────────┘          │             │
│                        │                      ┌────────┴──────────┐  │
│                        │                      │    BPF MAPS       │  │
│                        │                      │  (command I/O)    │  │
│                        │                      └───────────────────┘  │
├────────────────────────┼──────────────────────────────┬──────────────┤
│                        │        USER SPACE             │              │
│                        ▼                               │              │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                   SENSORY SYSTEM                                │ │
│  │                   internal/sensor/collector.go                  │ │
│  │  · Reads eBPF ring buffer events                                │ │
│  │  · Normalizes raw kernel data into the state vector x(t)        │ │
│  └─────────────────────────────┬───────────────────────────────────┘ │
│                                │                                      │
│                                ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                   LIMBIC SYSTEM                                 │ │
│  │                   internal/state/memory.go                      │ │
│  │  · Short-term ring buffer of recent state vectors               │ │
│  │  · Feeds derivatives computation in the cortex                  │ │
│  └─────────────────────────────┬───────────────────────────────────┘ │
│                                │                                      │
│                                ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                   PREDICTIVE CORTEX                             │ │
│  │        internal/brain/{matrix, mahalanobis, predictor}.go       │ │
│  │                                                                 │ │
│  │  1. Update μ and Σ incrementally  (Welford)                     │ │
│  │  2. Calculate D_M(x(t))           (Mahalanobis)                 │ │
│  │  3. Apply EWMA smoothing →  D̄_M(t)                              │ │
│  │  4. Calculate dD̄_M/dt and d²D̄_M/dt²  (Predictor)               │ │
│  │  5. Evaluate against adaptive thresholds                        │ │
│  │  6. Determine response level (0–5)                              │ │
│  └────────────────────┬────────────────────┬────────────────────────┘ │
│                       │                    │                           │
│                       ▼                    ▼                           │
│  ┌──────────────────────────┐  ┌────────────────────────────────────┐ │
│  │      REFLEX ARC          │  │   OPPORTUNISTIC COMMUNICATION      │ │
│  │  internal/motor/         │  │                                    │ │
│  │                          │  │  · Webhooks to orchestrators       │ │
│  │  · cgroups.go            │  │  · Prometheus-compatible endpoint  │ │
│  │    CPU/mem throttling    │  │  · Structured audit log            │ │
│  │  · signals.go            │  │  · /healthz enriched state vector  │ │
│  │    SIGSTOP / SIGTERM     │  │                                    │ │
│  └──────────────────────────┘  └────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Layer 1 — Kernel Space (eBPF)

All signal collection and low-level actuation happens in kernel space. HOSA never polls `/proc`. It hooks directly into kernel execution paths.

### Sensory Probes

Located in `internal/bpf/sensors.c`. Compiled to eBPF bytecode via `clang` and loaded at runtime via the custom loader in `internal/sysbpf/syscall.go`.

**Tracepoints** (stable, versioned kernel interface):

| Tracepoint | Data collected | State vector dimension |
|---|---|---|
| `sched_switch` | Context switches, run queue depth | `runqueue`, `ctx_switches` |
| `sched_process_exit` | Unexpected process terminations | Anomalous silence detection |
| `net_dev_xmit` / `netif_receive_skb` | Packets per second, bytes in/out | `net_tx`, `net_rx` |
| `block_rq_issue` / `block_rq_complete` | IOPS, I/O latency (issue→complete delta) | `io_throughput`, `io_latency` |

**kprobes** (dynamic kernel function hooks, kernel-version-aware):

| kprobe | Data collected | State vector dimension |
|---|---|---|
| `mm_page_alloc` | Memory allocation rate | `mem_alloc_rate` |
| `try_charge` (cgroup memory) | Per-cgroup memory charge | Per-cgroup attribution |
| `tcp_retransmit_skb` | TCP retransmission rate | `net_retransmit` |

**PSI hooks** (Pressure Stall Information — Weiner, 2018):

HOSA reads PSI directly from `/sys/fs/cgroup/.../memory.pressure` and `/proc/pressure/` rather than hooking into the PSI subsystem's kernel path. PSI data is consumed as part of the ring buffer event batch.

| PSI metric | Mapped to |
|---|---|
| `memory some avg10` | `mem_pressure` |
| `cpu some avg10` | `cpu_pressure` |
| `io some avg10` | `io_pressure` |

### Actuators

Actuation commands are written to **BPF maps** by the user-space cortex and read by a small eBPF program attached to the cgroup layer.

**cgroups v2 actuation** (`internal/motor/cgroups.go`):

HOSA writes directly to cgroup control files via the Linux VFS — no external library. The implementation lives in `internal/syscgroup/file_edit.go` and uses raw `open`/`write`/`close` syscalls for deterministic latency.

| File | Effect | Response level |
|---|---|---|
| `memory.high` | Applies aggressive reclaim backpressure — slows allocation without killing | Level 2+ |
| `memory.max` | Hard limit — triggers OOM within the cgroup before kernel-wide OOM-Killer | Level 4 |
| `cpu.max` | CPU bandwidth throttling (CFS quota) | Level 3+ |
| `cgroup.freeze` | Suspends all processes in the cgroup (SIGSTOP-like) | Level 5 |

**XDP actuation**:

Packet drop policy is enforced at the driver level via an XDP program loaded onto the node's primary network interface. Drop rules are updated via BPF maps from user space.

| XDP action | Effect | Response level |
|---|---|---|
| Drop new SYN packets | Reject new inbound connections, preserve existing | Level 3 |
| Drop all inbound except healthcheck IPs | Hard load shedding | Level 4 |
| Drop all inbound + outbound except management | Network isolation | Level 5 |

### Ring Buffer Transport

The eBPF ring buffer (`BPF_MAP_TYPE_RINGBUF`) is the sole transport between kernel and user space. It is:

- **Lock-free** — single-producer (kernel), single-consumer (user-space goroutine)
- **Zero-copy** — user space reads directly from shared memory, no kernel→user copy
- **Back-pressure aware** — if the consumer falls behind, older events are dropped (events lost counter exposed via metrics endpoint)

Typical kernel→user latency: **1–10 μs** on modern hardware.

---

## Layer 2 — User Space

### The Sensory System (`internal/sensor`)

**File:** `internal/sensor/collector.go`

Responsibilities:
- Consumes the eBPF ring buffer in a dedicated goroutine
- Normalizes raw kernel counters into the state vector `x(t) ∈ ℝⁿ`
- Computes per-interval deltas (e.g., context switch *rate* from cumulative counter)
- Resolves per-event cgroup attribution (maps `task → cgroup path` for contributing process identification)

The collector outputs a `StateEvent` struct at the configured sampling interval (default 100ms in homeostasis, adaptive down to 10ms in vigilance). This struct is the canonical representation of the node's state at time `t`.

```go
// StateEvent — canonical state representation
type StateEvent struct {
    Timestamp   time.Time
    Vector      []float64   // x(t): normalized state vector
    CgroupMap   map[string][]float64  // per-cgroup resource breakdown
    SamplingMs  int         // current adaptive sampling interval
}
```

### The Limbic System (`internal/state`)

**File:** `internal/state/memory.go`

A fixed-size ring buffer of recent `StateEvent` structs. This is the agent's **short-term memory** — it does not grow with time.

- Size: configurable, default 1000 samples (~100s at 100ms sampling)
- Used by the cortex for: EWMA smoothing, derivative computation, and the habituation condition check

Memory footprint: `O(window_size × n)` — with `n = 10` dimensions and 1000 samples, this is approximately **80KB**. Fixed at startup.

### The Predictive Cortex (`internal/brain`)

The cortex is the computational core. It is implemented across three files:

---

#### `internal/brain/matrix.go` — Covariance Matrix Management

Implements the **incremental Welford algorithm** for online updates of `μ` and `Σ`:

```
On each new sample x(t):
  n += 1
  delta = x(t) - μ
  μ += delta / n
  delta2 = x(t) - μ
  M += outer_product(delta, delta2)   // Welford's M₂
  Σ = M / (n - 1)
```

**Memory:** `O(n²)` — for `n = 10`, `Σ` is a 10×10 matrix = 800 bytes. Fixed.
**Per-sample cost:** `O(n²)` — approximately a few microseconds for `n ≤ 15`.

**Tikhonov regularization** is applied before inversion to handle near-singular matrices (collinear variables):

```
Σ_reg = Σ + λI
```

where `λ` is auto-tuned during warm-up based on the observed condition number of `Σ`.

**Cholesky inversion** (`Σ⁻¹`) is recomputed only when `Σ` has changed significantly (tracked via Frobenius norm delta), not on every sample. This amortizes the `O(n³)` inversion cost across many samples.

---

#### `internal/brain/mahalanobis.go` — Homeostasis Calculation

Computes the current Mahalanobis Distance:

```
D_M(x(t)) = sqrt( (x(t) - μ)ᵀ Σ⁻¹ (x(t) - μ) )
```

Also computes:

**Dimensional contribution decomposition** — identifies *which resources* are driving the deviation:

```
d = x(t) - μ
c_j = d_j × (Σ⁻¹ d)_j
```

The `c_j` values are logged alongside every decision and included in webhook payloads.

**Load Direction Index (φ)** — determines whether the deviation is toward overload or idleness:

```
φ(t) = (1/n) Σ_j  s_j × (d_j / σ_j)
```

where `s_j ∈ {+1, -1}` is the load sign of variable `j` (pre-configured at startup: +1 for CPU util, memory used, etc.; -1 for CPU idle, free memory, etc.).

**Covariance deformation ratio (ρ)** — measures whether the *structure* of correlations has changed, not just the magnitude:

```
ρ(t) = ‖Σ_recent - Σ_baseline‖_F / ‖Σ_baseline‖_F
```

High `ρ` with contained `D_M` is the adversarial demand signature (Regime +3).

---

#### `internal/brain/predictor.go` — Derivatives and Time-to-Failure

**EWMA smoothing** before differentiation (avoids ill-posed numerical differentiation on noisy data):

```
D̄_M(t) = α × D_M(t) + (1 - α) × D̄_M(t-1)
```

`α` is calibrated per-resource during warm-up based on observed signal variance.

**First derivative** (velocity of departure from homeostasis):

```
dD̄_M/dt ≈ (D̄_M(t) - D̄_M(t-Δt)) / Δt
```

**Second derivative** (acceleration — is the system heading *faster* toward collapse?):

```
d²D̄_M/dt² ≈ (dD̄_M/dt(t) - dD̄_M/dt(t-Δt)) / Δt
```

**Time-to-failure estimation** (used in Level 4 activation):

If `dD̄_M/dt > 0` and `D_M` is on a trajectory toward threshold `θ₄`:

```
TTF = (θ₄ - D̄_M(t)) / (dD̄_M/dt)
```

Level 4 activates when `TTF < T_critical` (configurable, default 10s), *even if* `D_M` has not yet reached `θ₄`. This is proactive containment, not reactive.

### The Reflex Arc (`internal/motor`)

**Files:** `internal/motor/cgroups.go`, `internal/motor/signals.go`

The motor layer translates cortex decisions into kernel actions. It is intentionally simple — no decision logic here. The cortex decides; the motor executes.

**`cgroups.go`** — Direct cgroup file manipulation via `internal/syscgroup/file_edit.go`. Uses raw VFS writes, not `libcgroup` or any external dependency. All writes are atomic (single `write()` syscall to the cgroup pseudo-file).

**`signals.go`** — Process signaling for Level 5 (quarantine). Sends `SIGSTOP` to freeze non-critical processes and `SIGTERM` (with `SIGKILL` fallback) for processes explicitly identified as destructive. Never touches safelist entries.

**Action atomicity and ordering:** Cgroup writes and XDP map updates are applied in a defined order to minimize the window of inconsistency:

1. Set `memory.high` backpressure (gentlest, first)
2. Update XDP drop rules (network-level)
3. Set `cpu.max` throttle
4. Set `memory.max` hard limit
5. `cgroup.freeze` (most disruptive, last)

Rollback on any step failure logs the partial state and escalates to the next response level.

### Opportunistic Communication

All external communication is non-blocking and best-effort. HOSA **never waits** for a webhook response before proceeding with mitigation.

**Webhook payload** — emitted at Level 2+ (configurable):

```json
{
  "severity": "warning",
  "node": "worker-node-07",
  "timestamp": "2026-03-10T14:23:09.000Z",
  "hosa_level": 2,
  "d_m": 4.7,
  "d_m_derivative": 2.1,
  "d_m_acceleration": 0.5,
  "phi": 1.8,
  "rho": 0.12,
  "dominant_dimension": "mem_used",
  "dominant_contribution_pct": 68,
  "dimensional_contributions": {
    "mem_used": 0.68,
    "mem_pressure": 0.19,
    "io_latency": 0.08,
    "cpu_total": 0.05
  },
  "suspected_cgroup": "/kubepods/pod-payment-service-7b4f",
  "action_taken": "memory.high reduced to 1.6G",
  "action_status": "effective",
  "d2dm_dt2": -0.45
}
```

**Prometheus endpoint** (Phase 2): exposes the normalized state vector, current `D_M`, response level, and derivative values in Prometheus text format at `:9090/metrics`.

**Audit log** — every decision is written to `/var/log/hosa/decisions.log` in structured JSON, regardless of network connectivity. This is the primary source for post-incident analysis.

---

## Cross-Cutting Subsystems

### Hardware Proprioception — Warm-Up Phase

At startup, before any detection or mitigation, HOSA executes the **Hardware Proprioception** sequence:

```
1. Topology discovery
   └─ Read /sys/devices/system/node/   → NUMA topology
   └─ Read /sys/devices/system/cpu/    → core count, cache sizes
   └─ Read /proc/meminfo               → total/available memory
   └─ Identify environment class       → bare metal / cloud VM / Kubernetes / edge

2. State vector definition
   └─ Select dimensions based on topology
      (NUMA nodes → per-node memory dimensions for complex topologies)
   └─ Resolve eBPF probe attachment points

3. Baseline accumulation (default: 5 minutes)
   └─ Collect samples without mitigation
   └─ Build initial μ₀ and Σ₀ via Welford
   └─ Detect workload multimodality (basic check via kurtosis)

4. α calibration (EWMA)
   └─ Per-resource: α = f(observed signal variance during warm-up)
   └─ High-variance signals → lower α (more smoothing)

5. Adaptive threshold computation
   └─ θ₁ = μ_DM + 2σ_DM   (Vigilance)
   └─ θ₂ = μ_DM + 3σ_DM   (Soft Containment)
   └─ θ₃ = μ_DM + 4σ_DM   (Active Containment)
   └─ θ₄ = μ_DM + 5σ_DM   (Severe Containment)
```

Cold start vulnerability window: during warm-up, HOSA operates in **monitoring-only mode** (no mitigation). This is a known limitation documented in [Limitations](#limitations).

---

### Graduated Response System (Levels 0–5)

The response level is the output of the cortex's threshold evaluation. Transitions follow strict rules — you cannot skip levels (except in the fast-collapse path described below).

```
                    ┌─── d²D̄_M/dt² > 0 (accelerating)
                    │
              D_M > θ₃ ──────────────────────────────────→ Level 3
              D_M > θ₂ ─── dD̄_M/dt > 0 ─────────────────→ Level 2
              D_M > θ₁ ─── or dD̄_M/dt > 0 sustained ───→ Level 1
              D_M < θ₁ ─── dD̄_M/dt ≤ 0 ─────────────────→ Level 0

              TTF < T_critical ────────────────────────────→ Level 4 (bypasses 3)
              Level 3/4 failing + D_M ascending ──────────→ Level 5
```

**Hysteresis** prevents oscillation (flapping). De-escalation from level `N` to `N-1` requires:
- `D_M < θ_{N-1}` sustained for `T_hysteresis` (default: 60s)
- `dD̄_M/dt < 0` (actively improving)

**Fast-collapse bypass:** If TTF estimation indicates collapse within `T_critical` seconds, HOSA escalates directly to Level 4 regardless of current level. This prevents slow-escalation scenarios where the system collapses before graduated response can reach the necessary level.

Full level specification:

| Level | Name | Trigger | Actions | Reversibility |
|---|---|---|---|---|
| **0** | Homeostasis | `D_M < θ₁` and `dD̄/dt ≤ 0` | Thalamic filter active. Minimal heartbeat telemetry. GreenOps optimizations (optional). | — |
| **1** | Vigilance | `D_M > θ₁` or `dD̄/dt > 0` sustained | Sampling: 100ms → 10ms. Local log entry. No system intervention. | Auto when condition clears |
| **2** | Soft Containment | `D_M > θ₂` and `dD̄/dt > 0` | `renice` contributing processes. `memory.high` backpressure on offending cgroup. Webhook (async, non-blocking). | Auto + hysteresis |
| **3** | Active Containment | `D_M > θ₃` and `d²D̄/dt² > 0` | `cpu.max` throttle. XDP: drop new SYN packets. Urgent webhook. | Auto + extended hysteresis |
| **4** | Severe Containment | `D_M > θ₄` or `TTF < T_critical` | `memory.max` hard limit. XDP: drop all inbound except healthcheck. `cgroup.freeze` on non-critical cgroups. | Requires sustained `D_M < θ₃` |
| **5** | Autonomous Quarantine | Containment failure + ascending `D_M` | Network isolation (mode depends on environment class). `SIGSTOP` non-critical processes. Full state snapshot to persistent log. Final webhook attempt. | **Manual intervention required** |

---

### Habituation — Baseline Drift

Habituation prevents **chronic false positives** when the workload permanently shifts (new deployment, organic growth, configuration change).

**Trigger conditions** (all must be true simultaneously):

```
|dD̄_M/dt| < ε_d               // system has stabilized
ρ(t) < ρ_threshold             // covariance structure preserved (no deformation)
ΔH(t) < ΔH_threshold           // syscall distribution unchanged
PBI(t) < PBI_threshold         // no propagation indicators
D_M(t) < D_M_safety            // not too close to resource exhaustion
t_stable > T_min               // sustained for at least T_min (default: 30 min)
```

**Mechanism:** Exponential decay of sample weights in the Welford accumulator. Recent samples carry more weight; old samples decay. `μ` and `Σ` shift toward the new operational regime.

**Habituation is blocked for:**

| Regime | Reason |
|---|---|
| Regime −3 (Anomalous Silence) | Silence incoherent with temporal context is never "normal" |
| Regime +3 (Adversarial) | High `ρ` indicates structural deformation — never normalize |
| Regime +4 (Local Failure) | Monotonically growing `D_M` is a progressive failure, not a new normal |
| Regime +5 (Viral) | Propagation indicators are categorically never habituated |
| Any regime near resource limits | If `D_M` is above `D_M_safety`, the system has no margin — habituation would normalize a dangerous state |

---

### Thalamic Filter — Telemetry Suppression

In Regime 0 (homeostasis), HOSA suppresses detailed telemetry to external systems. Only a minimal periodic heartbeat is emitted confirming `{"status": "alive", "level": 0, "d_m": 1.1}`.

**Why this matters:** In a large fleet, every node emitting full telemetry 24/7 generates significant ingestion cost even when nothing is wrong. The Thalamic Filter means HOSA nodes are *silent when healthy* and *loud when sick* — exactly the semantics you want.

Detail level escalates with response level:
- Level 0: heartbeat only (default interval: 60s)
- Level 1: heartbeat + `D_M` + dominant dimension (interval: 10s)
- Level 2+: full state vector + dimensional contributions + action taken (interval: 1s or on every state change)

---

### Quarantine Modes by Environment

Level 5 (Autonomous Quarantine) involves network isolation. The strategy varies by environment class, detected automatically during Hardware Proprioception.

| Environment | Detection | Network Strategy | Recovery |
|---|---|---|---|
| **Bare metal (IPMI)** | `/sys/class/net/` + `ipmi_*` kernel modules | Deactivate all interfaces *except* IPMI/iLO/iDRAC management interface | Manual via IPMI console |
| **Cloud VM (AWS/GCP/Azure)** | DMI/SMBIOS + metadata service `169.254.169.254` | XDP drops all traffic *except* metadata service, DHCP, and orchestrator API. Writes `hosa-quarantine=true` tag via metadata API. | Orchestrator terminates/replaces instance. Optional self-termination via cloud API (disabled by default). |
| **Kubernetes (DaemonSet)** | `/proc/1/cgroup` namespace + `KUBERNETES_SERVICE_HOST` env | Cannot deactivate host interfaces. Applies max cgroup containment + adds `hosa.io/quarantine=true:NoExecute` taint to Node via K8s API → scheduler evacuates pods. | Operator removes taint after investigation |
| **Edge / IoT (physical access)** | Flag `environment: edge-physical` | Full network interface deactivation. Preserves logs to flash/eMMC. | Manual by field technician |
| **Edge / IoT (remote only)** | Flag `environment: edge-remote` | Network deactivation + hardware watchdog timer (default: 30 min timeout → auto-reboot). Post-reboot: conservative mode for configurable period. | Auto via watchdog reboot + observation period |
| **Air-gapped (SCADA/ICS)** | Flag `environment: airgap` | Identical to bare metal. All opportunistic communication permanently disabled. Logs encrypted, collected by authorized physical access only. | Manual with physical access authorization |

**Principle of conservative default:** In ambiguous cases (e.g., private cloud VM that doesn't respond to the standard metadata service endpoint), HOSA assumes the **most conservative quarantine mode** (cloud VM — XDP-only, no interface deactivation), prioritizing recoverability over isolation.

---

## Data Flow — End to End

A single cycle through the system, from kernel event to mitigation action:

```
1. KERNEL EVENT
   └─ e.g., mm_page_alloc fires (memory allocation spike)
   └─ eBPF probe records: timestamp, PID, cgroup, delta bytes
   └─ Event written to ring buffer

2. RING BUFFER → COLLECTOR  (~1–10 μs)
   └─ collector.go reads event batch from ring buffer
   └─ Aggregates into state vector x(t) at sampling interval
   └─ Resolves cgroup attribution (PID → cgroup path)
   └─ Emits StateEvent to Limbic System

3. LIMBIC SYSTEM (~10 μs)
   └─ Appends StateEvent to ring buffer of recent states
   └─ Provides sliding window to cortex for derivative computation

4. COVARIANCE UPDATE  (~50–100 μs)
   └─ matrix.go: Welford update of μ and Σ
   └─ Checks if Σ has changed enough to warrant Σ⁻¹ recomputation
   └─ If yes: Cholesky inversion (O(n³), ~10 μs for n=10)

5. MAHALANOBIS COMPUTATION  (~10 μs)
   └─ D_M = sqrt( (x-μ)ᵀ Σ⁻¹ (x-μ) )
   └─ Compute c_j (dimensional contributions)
   └─ Compute φ (load direction)
   └─ Compute ρ (covariance deformation)

6. EWMA + DERIVATIVES  (~5 μs)
   └─ D̄_M(t) = α × D_M(t) + (1-α) × D̄_M(t-1)
   └─ dD̄_M/dt  ≈ (D̄_M(t) - D̄_M(t-Δt)) / Δt
   └─ d²D̄_M/dt² ≈ second-order finite difference
   └─ TTF estimation if dD̄/dt > 0

7. THRESHOLD EVALUATION + LEVEL DECISION  (~1 μs)
   └─ Evaluate (D_M, dD̄/dt, d²D̄/dt², TTF) against thresholds
   └─ Apply hysteresis (no de-escalation below T_hysteresis)
   └─ Safelist check (never target protected processes)
   └─ Determine target cgroup (highest c_j contributor)
   └─ Determine response level N

8. ACTUATION  (~100–500 μs, dominated by cgroup VFS write)
   └─ Write to cgroup control files (memory.high, cpu.max, etc.)
   └─ Update BPF maps for XDP rules (if level ≥ 3)
   └─ Write audit log entry (async, non-blocking)
   └─ Dispatch webhook (async, non-blocking)

TOTAL: ~200 μs – 1 ms per cycle (dominated by cgroup write syscall)
```

---

## State Machine

HOSA's response level follows this state machine:

```
         ┌──────────────────────────────────────────────────────┐
         │                    CONDITIONS                         │
         ├─────────────────┬────────────────────────────────────┤
         │ ESCALATION      │ DE-ESCALATION                       │
         │ immediate       │ requires hysteresis + improving D_M │
         └─────────────────┴────────────────────────────────────┘

    ┌─────────┐
    │    0    │ ◄──────────────────────────────────────────────────┐
    │Homeost. │                                                    │
    └────┬────┘                                                    │
         │ D_M>θ₁ or dD̄/dt>0                                      │
         ▼                                                         │
    ┌─────────┐                                                    │
    │    1    │ ◄──────────────────────────────────────────────────┤
    │Vigilance│                                                    │
    └────┬────┘                                                    │
         │ D_M>θ₂ and dD̄/dt>0                                     │
         ▼                                                         │
    ┌─────────┐                                                    │
    │    2    │ ◄──────────────────────────────────────────────────┤
    │ Soft    │                                                    │
    │Contain. │                                                    │
    └────┬────┘                                                    │
         │ D_M>θ₃ and d²D̄/dt²>0                                   │ hysteresis +
         ▼                                                         │ D_M improving
    ┌─────────┐                                                    │
    │    3    │ ◄──────────────────────────────────────────────────┤
    │ Active  │                                                    │
    │Contain. │                                                    │
    └────┬────┘                                                    │
         │ D_M>θ₄ or TTF<T_critical                               │
         ▼                                                         │
    ┌─────────┐                                                    │
    │    4    │ ◄──────────────────────────────────────────────────┘
    │ Severe  │
    │Contain. │
    └────┬────┘
         │ containment failing + D_M ascending
         ▼
    ┌─────────┐
    │    5    │  ──── MANUAL RECOVERY REQUIRED ────────────────────
    │Quarant. │
    └─────────┘
```

---

## The Safelist

Certain processes and cgroups are **never targeted** by HOSA's throttling or signaling, regardless of how high their resource contribution is.

**Permanent safelist entries (built-in):**

- The HOSA agent itself (PID resolved at startup)
- All kernel threads (`kthreadd` and descendants — PIDs with no user-space parent)
- Processes in the HOSA cgroup (`/sys/fs/cgroup/hosa/`)

**Auto-detected safelist entries:**

| Process | Detection method |
|---|---|
| `kubelet` | Presence of `/var/run/kubelet.sock` or `KUBERNETES_SERVICE_HOST` env |
| `containerd` | Presence of `/run/containerd/containerd.sock` |
| `dockerd` | Presence of `/var/run/docker.sock` |
| `systemd` (PID 1) | Always PID 1 |
| `sshd` | Port 22 listening process (preserves operator access during quarantine) |

**Operator-defined safelist entries:**

```bash
# In hosa.yaml config:
safelist:
  cgroups:
    - /kubepods/besteffort/critical-monitoring
  process_names:
    - vault-agent
    - consul
  pids:
    - 1234  # specific PID (cleared on restart)
```

**Safelist and Level 5:** Even at Level 5, safelist entries are not frozen. The goal of quarantine is to isolate the node from the network and freeze *non-critical* processes — the safelist defines what "critical" means.

---

## Key Design Decisions

| Decision | Rationale | Alternative considered |
|---|---|---|
| **Mahalanobis Distance over ML/DL** | `O(n²)` constant memory, no GPU, no training pipeline, sub-ms inference, interpretable output. Runs on a Raspberry Pi. | Autoencoders (rejected: requires training infrastructure, opaque, large footprint), Isolation Forest (rejected: requires data windows, not incremental) |
| **Welford incremental updates** | `O(n²)` per sample with `O(1)` allocation. No data windows stored. Predictable memory. | Batch covariance computation (rejected: `O(n²×k)` memory, unbounded growth) |
| **EWMA over raw differentiation** | Numerical differentiation is ill-posed on noisy, discrete kernel data. EWMA provides a principled smoothing with a single tunable parameter. | Kalman filter (kept as future alternative — optimal for Gaussian noise but more complex to tune; planned for experimental comparison) |
| **Direct VFS cgroup writes over libcgroup** | Eliminates a runtime dependency. Deterministic latency (single write syscall). No library version drift. | libcgroup (rejected: adds runtime dependency, potential version mismatch across distros) |
| **Go over Rust/C for user space** | Pragmatic: faster iteration for research phase. Goroutines make the async communication layer clean. Hot path uses zero-allocation patterns (`sync.Pool`, pre-allocated slices). GC pauses < 1ms on Go 1.22+. | Rust (kept as future option if GC benchmarks show impact on detection window — architecture allows hot-path migration) |
| **Ring buffer over perf buffer** | `BPF_MAP_TYPE_RINGBUF` is lock-free and supports variable-length records. Lower overhead than `BPF_MAP_TYPE_PERF_EVENT_ARRAY`. | perf_event_array (rejected: per-CPU allocation, lock-based, higher overhead) |
| **Complement, not replace monitoring** | Architectural clarity. HOSA solves the Lethal Interval problem — a fundamentally different timescale than what Prometheus/Datadog solve. Trying to do both would compromise both. | Full observability replacement (rejected: out of scope, would require centralized storage, dashboards, alerting — different problem domain) |

---

## Performance Characteristics

All figures are targets for the Phase 1 implementation. Experimental validation will be documented separately.

| Metric | Target | Notes |
|---|---|---|
| **Detection latency** | < 2s from anomaly onset to Level 1 | At 10ms adaptive sampling |
| **Mitigation latency** | < 100ms from Level 2 trigger to cgroup write | Dominated by VFS write syscall |
| **CPU overhead** | < 1% of a single core | During homeostasis; higher during active containment cycles |
| **Memory footprint** | < 10MB RSS | Includes state vector history, covariance matrix, eBPF program maps |
| **Ring buffer latency** | 1–10 μs | Kernel event → user space consumer |
| **Full cycle latency** | < 1ms | Event collection through actuation decision |
| **eBPF program overhead** | < 0.1% additional syscall latency | Per-probe overhead measured via `bpftool` |

**HOSA overhead budget:** HOSA itself operates inside a dedicated cgroup with `cpu.max = 50000 100000` (50ms per 100ms window = max 50% of 1 core) and `memory.max = 64M`. If the agent exceeds its own budget, the kernel contains it before it affects other processes. HOSA practices what it preaches.

---

## Security Posture

HOSA requires elevated privileges to perform its function. The minimum required Linux capabilities:

| Capability | Required for | Response levels |
|---|---|---|
| `CAP_BPF` | Loading and running eBPF programs | All levels (collection) |
| `CAP_PERFMON` | Attaching to tracepoints and kprobes | All levels (collection) |
| `CAP_NET_ADMIN` | Loading XDP programs onto interfaces | Levels 3–5 (actuation) |
| `CAP_SYS_ADMIN` | Writing to cgroup v2 control files | Levels 2–5 (actuation) |
| `CAP_KILL` | Sending `SIGSTOP`/`SIGTERM` to processes | Level 5 only |

**Principle of least privilege in deployment:**
- Levels 0–1 (observation only): `CAP_BPF` + `CAP_PERFMON` only
- Levels 0–2 (soft mitigation): add `CAP_SYS_ADMIN`
- Full capability (all levels): add `CAP_NET_ADMIN` + `CAP_KILL`

Operators can deploy HOSA in observation-only mode (equivalent to Levels 0–1) and gradually enable capabilities as they gain confidence in the agent's behavior.

**The agent cannot be corrupted by the workloads it monitors.** HOSA's cgroup is separate from all monitored cgroups. Its eBPF programs are verified by the kernel verifier before loading — a malformed program is rejected at load time, not at runtime.

---

## Limitations

These are documented honestly. See also [Whitepaper §9](whitepaper.pdf#section-9).

**1. Cold start window.** During warm-up (default 5 minutes), HOSA has no baseline and cannot make reliable detection decisions. The node is unprotected during this window. Mitigation: pre-seeding baseline from fleet aggregates (Phase 2 feature).

**2. Non-stationary workloads.** Workloads that vary randomly in magnitude and timing — without temporal pattern and without stabilization — undermine the baseline profile assumption. HOSA's effectiveness is reduced. Seasonal profiles (Section 6.6) and habituation (Section 5.5) address *predictable* variability; *random* variability is a recognized limitation.

**3. Adversarial evasion.** A sophisticated attacker who understands HOSA's architecture can execute a "low-and-slow" attack that keeps `D_M` and its derivatives below detection thresholds while maintaining malicious activity below the detection floor. Covariance deformation detection (`ρ`, `ΔH`) raises the bar significantly, but the theoretical evasion possibility exists. Formal adversarial resistance analysis is future research.

**4. Throttling side effects.** `memory.high` backpressure may increase latency of the contained service. CPU throttle may cause cascade timeouts in upstream services. The safelist and contributing-process targeting minimize this, but do not eliminate it.

**5. Linux-only, kernel ≥ 5.8.** eBPF CO-RE (Compile Once — Run Everywhere) requires kernel 5.8+. No portability to other operating systems is planned.

**6. GC pause impact (Go).** Go's garbage collector, while sub-millisecond on 1.22+, is non-deterministic. If hot-path benchmarks reveal GC pauses impacting detection latency under adversarial allocation pressure, migration of the hot path to a zero-GC language (Rust or C via CGo) is planned.

---

*For the complete mathematical formulation, see [`docs/math_model.md`](math_model.md).*
*For the theoretical foundation and academic context, see the [Whitepaper v2.1](whitepaper.pdf).*
