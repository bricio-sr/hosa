# HOSA — Homeostasis Operating System Agent

## Whitepaper & Architectural Manifesto

**Author:** Fabricio Roney de Amorim
**Document Version:** 2.2 — Architectural Expansion
**Creation Date:** March 9, 2026
**Revision Date:** March 30, 2026
**Academic Context:** Foundational intent for Master's dissertation — Unicamp (IMECC)
**Status:** Vision and Theoretical Foundation Document

**Integrity Record:**
- Reference repository: https://github.com/bricio-sr/hosa

**Version History:**
| Version | Date | Description |
|---|---|---|
| 1.0 | 03/09/2026 | First whitepaper version. Initial concept. |
| 2.0 | 03/10/2026 | Critical revision: bipolar taxonomy, supplementary metrics, expanded graduated response. |
| 2.1 | 03/10/2026 | Objection hardening: robustness under non-normality, ICP calibration, environment-based quarantine, narrative walkthrough, FAQ. |
| 2.2 | 03/30/2026 | Architectural expansion: Phase 2 (Sympathetic Nervous System); Phases 2–7 renumbered to 3–9; Phase 8 (Causal Kernel); eSRE moved to Phase 9; DTrace/eBPF lineage added; eBPF interaction corrected to reflect zero third-party dependencies. |


---

## Abstract

This document presents HOSA (Homeostasis Operating System Agent), a bio-inspired software architecture for autonomous resilience in Linux operating systems. HOSA proposes replacing the dominant exogenous telemetry model with a model of **Endogenous Resilience**, in which each computational node possesses autonomous capability for multivariable detection and local real-time mitigation, regardless of network connectivity.

Anomaly detection is performed through multivariable statistical analysis based on the Mahalanobis Distance and its temporal rate of change, with signal collection via eBPF in the Linux Kernel Space. Mitigation is executed in progressive layers: in Phase 1, through deterministic manipulation of Cgroups v2 and XDP; in Phase 2, through direct intervention in the process scheduler (via `sched_ext`) and the virtual memory subsystem, replacing the kernel's general-purpose algorithms with deterministic survival policies during the Lethal Interval. Subsequent phases extend the system with semantic, swarm, and causal reasoning capabilities.

HOSA does not replace orchestrators or global monitoring systems. It complements them by operating in the temporal interval where those systems are structurally incapable of acting: the milliseconds between the onset of a collapse and the arrival of the first metric at the external control plane.

**Keywords:** Endogenous Resilience, Autonomic Computing, eBPF, sched_ext, Multivariable Anomaly Detection, Mahalanobis Distance, Causal Inference, do-calculus, Bio-Inspired Systems, Edge Computing, SRE.

---

## 1. Introduction and Problem Statement

### 1.1. The Dominant Model and Its Structural Limitations

Systems reliability engineering (SRE) has consolidated over the last decade around a paradigm this work terms **Exogenous Telemetry**: a model in which local agents collect metrics, transmit them via network to central analysis servers, and await mitigation instructions derived from that remote analysis.

This paradigm, supported by widely adopted tools such as Prometheus (Prometheus Authors, 2012), Datadog, Grafana, and orchestrators such as Kubernetes (Burns et al., 2016), operates under assumptions that become progressively fragile as computational infrastructure expands into IoT, Edge Computing, telecommunications, and industrial embedded systems scenarios.

The structural fragility of the exogenous model manifests in two dimensions:

**The Latency of Awareness.** The operational cycle follows a discrete sequence: periodic collection (polling with typical intervals of 10 to 60 seconds), network transmission, TSDB storage, evaluation against static thresholds, and alert dispatch. Each step introduces cumulative latency. The central system makes decisions based on a statistically stale snapshot of the remote node. In fast-collapse scenarios — DDoS attacks, aggressive memory leaks, instantaneous load spikes — mitigation arrives too late.

**The Connectivity Fragility.** The exogenous model assumes continuous and reliable connectivity. This premise is routinely violated in Edge Computing scenarios (intermittent connectivity), during DDoS attacks that saturate the monitored node's outbound bandwidth, and in industrial infrastructures with segmented networks. When the network fails, the node simultaneously loses its ability to report and to receive mitigation instructions.

### 1.2. The Physics of Collapse: The Lethal Interval

The collapse of a computational node is not gradual; it is an exponential cascade. When physical memory is exhausted, the Linux Kernel activates the OOM-Killer, abruptly terminating processes based on scoring heuristics, corrupting in-flight transactions, and generating immediate unavailability. `systemd-oomd` (Poettering, 2020) and PSI (Weiner, 2018) represent attempts to address this gap, but operate with limited scope: PSI provides pressure metrics without autonomous mitigation; `systemd-oomd` acts with static policies that do not consider multivariable resource correlation.

The temporal interval between the onset of lethal stress and the arrival of the first usable metric at the external monitoring system constitutes the **Lethal Interval** — the window where systems die without the external observer even being aware of the problem.

#### Figure 1 — Temporal Visualization of the Lethal Interval

```
COLLAPSE TIMELINE — MEMORY LEAK AT 50MB/s

Time  │ Node State             │ HOSA (Endogenous)      │ Prometheus (Exogenous)
──────┼────────────────────────┼────────────────────────┼─────────────────────────
 t=0  │ Leak starts  mem: 61%  │ D_M=1.1  Level 0       │ Last scrape 8s ago: OK
 t=1  │ mem: 64%  PSI: 18%    │ ⚡ D_M=2.8 → Level 1    │ (no scrape)
 t=2  │ mem: 68%  swap activ. │ ⚡ D_M=4.7 → Level 2    │ (no scrape)
      │                        │ memory.high → 1.6G     │
 t=4  │ mem: 72% (contained)  │ dD̄/dt decelerating     │ Scrape! 1.47G → OK (!)
 t=8  │ mem: 74% (plateau)    │ ✓ STABILIZED           │ (no scrape)
──────┼────────────────────────┼────────────────────────┼─────────────────────────
 t=40 │ ☠ OOM-Kill (no HOSA)  │ ✓ System contained     │ Scrape. Detects restart
 t=100│ ☠ 502 errors          │ ✓ Rollback complete    │ ⚠ ALERT FIRED (60s late)
```

### 1.3. The Central Thesis

> *Orchestrators and centralized monitoring systems are essential instruments for capacity planning, load balancing, and long-term infrastructure governance. However, they are structurally — not accidentally — too slow to guarantee the survival of a node in real time. If collapse occurs in the interval between exogenous perception and action, the immediate decision-making capability must reside in the node itself.*

HOSA proposes **complementing** central monitoring with a layer of local intelligence that operates autonomously during the Lethal Interval.

---

## 2. Conceptual Genesis: The Biological Metaphor as a Design Tool

### 2.1. The Reflex Arc as an Architectural Pattern

The HOSA architecture was conceived from the observation of the **spinal reflex arc**: when a human organism touches a harmful surface, the nociceptive signal does not travel to the cerebral cortex. Instead, the spinal cord executes a reflex muscle contraction in sub-milliseconds. Only after the reflex is executed is the cortex notified (Bear, Connors & Paradiso, 2015).

This pattern — **immediate local action followed by contextual notification to the command center** — is the operational model of HOSA.

Phase 2 deepens this metaphor: just as the **sympathetic nervous system** triggers deeper physiological responses under acute stress — redistributing blood flow to vital muscles, constricting peripheral vessels — HOSA Phase 2 **physically redistributes processor time and memory topology** in favor of survival processes, altering the kernel's own rules during the Lethal Interval.

It is important to delimit the scope of this metaphor: it is used as a **heuristic tool for architectural design**, not as a claim of functional equivalence between biological and computational systems.

### 2.2. Precedents in the Literature

IBM's Autonomic Computing manifesto (Horn, 2001) articulated four desirable properties — self-configuration, self-optimization, self-healing, and self-protection — but remained at the level of strategic vision, without providing the low-level instrumentation for sub-millisecond latency.

Forrest, Hofmeyr & Somayaji (1997) on computational immunology established the theoretical foundations of the "self" versus "non-self" distinction in computational systems. HOSA absorbs this principle into its behavioral screening layer.

What distinguishes HOSA is the **operational synthesis**: combination of continuous multivariable detection with kernel-space actuation via contemporary mechanisms (eBPF, `sched_ext`, Cgroups v2, XDP) that did not exist when those works were published.

---

## 3. Related Work and Positioning

### 3.1. Native Linux Kernel Mechanisms

| Mechanism | Function | Limitation HOSA Addresses |
|---|---|---|
| **PSI** — Weiner, 2018 | Exposes CPU, memory, and I/O pressure metrics as stall percentage. | Passive sensor only; unidimensional; no mitigation capability. |
| **systemd-oomd** — Poettering, 2020 | Kills entire cgroups when memory PSI exceeds threshold. | Static one-dimensional threshold; binary action (nothing or kill); no graduated responses. |
| **OOM-Killer** | Kernel mechanism of last resort to free memory. | Reactive and destructive; simplified heuristics frequently eliminate critical processes. |
| **cgroups v2** — Heo, 2015 | Resource control interface per process group. | Actuator mechanism without associated decision intelligence. HOSA uses it as Phase 1 motor. |
| **sched_ext** — Torvalds et al., 2024 | Framework for replacing the process scheduler via eBPF programs loaded at runtime. | Extension mechanism without embedded policy. HOSA uses it as Phase 2 motor for the Survival Scheduler. |
| **Buddy Allocator / Compaction** | Physical page allocator; compaction moves pages to defragment. | Compaction occurs reactively under pressure, causing Compaction Stalls invisible to traditional metrics. |

### 3.2. Dynamic Programmable Observability: DTrace and the eBPF Lineage

The trajectory culminating in eBPF has an intellectual genealogy that cannot be omitted from an honest literature review. The origin of this lineage is **DTrace**, developed by Bryan Cantrill, Mike Shapiro, and Adam Leventhal at Sun Microsystems and introduced in Solaris 10 in 2004 (Cantrill et al., 2004).

DTrace established principles that defined the field of dynamic observability in production:

**Zero-cost-when-unused principle.** Probes have strictly zero CPU cost when not enabled — instrumentation code is replaced by NOP instructions at compile time. This represented a break from prior models of conditional compilation or ptrace-based interception, both with permanent cost incompatible with production use without performance degradation.

**Kernel safety principle.** DTrace's central verifier guarantees that D programs cannot compromise system stability: formally verified before execution for absence of infinite loops, invalid memory access, and destructive side effects. The Linux eBPF verifier inherits and formalizes this same principle. It is this principle that makes viable the execution of dynamic code in Ring 0 — a bug in an eBPF program results in rejection by the verifier, not a kernel panic.

**Probe language as a programming interface.** System instrumentation must be programmable at runtime by a high-level language, without kernel recompilation or system restart — observability as code, not configuration.

**Convergence with Linux.** SystemTap (Red Hat, 2005) was the first attempt at dynamic programmable observability on Linux. The perf subsystem and uprobes/kprobes generalized instrumentation of arbitrary kernel and user-space points. DTrace for Linux (Oracle) attempted a direct port. All these initiatives converge, in modern practice, on eBPF — native to the kernel, universally adopted from kernel 4.x onwards.

The relevance for HOSA: the safety principle inherited from DTrace makes viable Phase 2's Survival Scheduler in Ring 0. The DTrace → eBPF lineage also establishes the **policy-mechanism separation** pattern that HOSA exploits: the kernel provides mechanisms (`sched_ext`, eBPF maps, ring buffers, XDP hooks), and HOSA defines the policies (Survival Scheduler, preemptive defragmentation, causal do-calculus).

### 3.3. Observability Ecosystem Tools

| Tool/Project | Function | HOSA Differentiation |
|---|---|---|
| **Prometheus + Alertmanager** | Pull-based metric collection, TSDB storage, rule-based alerts. | Classic exogenous model. Scrape interval: 15–60s. Minimum alert latency: >1 minute. No actuation. |
| **Sysdig Falco** — Sysdig, 2016 | Runtime anomalous behavior detection via eBPF, security-focused. | Detects security policy violations; does not monitor resource health; no autonomous mitigation. |
| **Cilium Tetragon** — Isovalent, 2022 | Security policy enforcement in kernel space via eBPF. | Static operator-defined rules; no statistical anomaly model; no graduated responses. |
| **Pixie (px.dev)** — New Relic | Continuous observability via eBPF without code instrumentation. | Collection and visualization system only; no autonomous actuation layer. |
| **BCC / bpftrace** — Gregg, 2019 | eBPF-based performance analysis and debugging tools for interactive use. | Diagnostic tools for human operators; practical realization of the DTrace heritage in Linux, but without the agency layer that HOSA adds. |
| **Facebook FBAR** — Tang et al., 2020 | Automated remediation at scale in Meta's datacenters. | Centralized remediation system with network dependency and proprietary infrastructure; not a local autonomous agent. |

### 3.4. The Identified Gap

No existing tool in the ecosystem combines, in a single local agent:

1. **Continuous multivariable detection** (CPU, memory, I/O, network, disk latency correlation in a unified statistical space);
2. **Rate-of-change analysis** (temporal derivative detecting acceleration toward collapse, not just current state);
3. **Physical scheduler control** (replacing CFS/EEVDF fairness policy with deterministic survival policy via `sched_ext`);
4. **Thermodynamic memory control** (preemptive defragmentation based on physical page topology entropy analysis);
5. **Pre-action causal reasoning** (dynamic IPC DAG construction and counterfactual evaluation via do-calculus before executing interventions);
6. **Total independence from external infrastructure** for its primary survival function.

HOSA positions itself at this intersection.

---

## 4. Mathematical Foundations

### 4.1. System State Representation

HOSA models the instantaneous state of a node as a vector $\vec{x}(t) \in \mathbb{R}^n$, where each component represents a system resource variable:

$$\vec{x}(t) = \begin{bmatrix} x_1(t) \\ x_2(t) \\ \vdots \\ x_n(t) \end{bmatrix}$$

In the reference implementation, the variables include (but are not limited to):
- CPU utilization (aggregate and per-core)
- Memory pressure (utilization, swap, PSI)
- Disk I/O throughput and latency
- Network packet rate (inbound/outbound)
- Scheduler queue depth (run queue depth)
- Page fault rate
- Context switch counters
- Memory fragmentation entropy $H_{frag}$ (introduced in Phase 2 — Section 9.2)

### 4.2. The Mahalanobis Distance as an Anomaly Detector

Static one-dimensional threshold-based anomaly detection (e.g., "CPU > 90%") suffers from a fundamental limitation: it ignores the **correlation structure** between variables. High CPU with low I/O and stable network may represent legitimate intensive processing. High CPU with growing memory pressure, I/O stall, and rising network latency represents imminent collapse. The static threshold does not distinguish these scenarios.

The Mahalanobis Distance (Mahalanobis, 1936) addresses this limitation by measuring the distance of an observation $\vec{x}$ relative to the multivariable distribution defined by the mean vector $\vec{\mu}$ and the Covariance Matrix $\Sigma$:

$$D_M(\vec{x}) = \sqrt{(\vec{x} - \vec{\mu})^T \Sigma^{-1} (\vec{x} - \vec{\mu})}$$

The Covariance Matrix $\Sigma$ captures the correlations between all variables. Its inverse $\Sigma^{-1}$ weights the dimensions according to their variance and interdependence. For a comprehensive review of outlier detection methods, see Aggarwal (2017).

### 4.3. The Temporal Derivative and the Problem of Numerical Stability

HOSA does not act on the instantaneous value of $D_M$, but on its **temporal rate of change** — the speed and acceleration with which the system departs from homeostasis.

The first derivative $\frac{dD_M}{dt}$ indicates the speed of departure. The second derivative $\frac{d^2D_M}{dt^2}$ indicates acceleration.

**Recognized problem: instability of numerical differentiation in discrete, noisy data.** Numerical differentiation is an ill-posed problem in the Hadamard sense. Without treatment, the second derivative of noisy kernel time series oscillates violently, generating false positives.

**Adopted solution:** HOSA implements an **Exponentially Weighted Moving Average (EWMA)** with a decay factor $\alpha$ calibrated per resource before derivative computation:

$$\bar{D}_M(t) = \alpha \cdot D_M(t) + (1 - \alpha) \cdot \bar{D}_M(t-1)$$

The factor $\alpha$ controls the fundamental trade-off between **responsiveness** (high values preserve rapid variations but retain noise) and **stability** (low values smooth the signal but introduce detection latency).

**Alternative under investigation:** The one-dimensional Kalman Filter provides optimal state estimation in the presence of Gaussian noise. The comparative analysis EWMA vs. Kalman will be presented in the dissertation's experimental phase.

### 4.4. Incremental Update of the Covariance Matrix

HOSA uses the **generalized Welford algorithm** (Welford, 1962) for incremental online updates of $\Sigma$ and $\vec{\mu}$. Each new sample $\vec{x}(t)$ updates $\Sigma$ in $O(n^2)$ with constant allocation ($O(1)$), regardless of the number of accumulated samples.

### 4.5. Inversion of the Covariance Matrix

For moderate dimensionality ($n \leq 10$), direct inversion via Cholesky decomposition is computationally feasible and numerically stable. For higher dimensionality, HOSA can resort to the incremental inverse update via the Sherman-Morrison-Woodbury formula.

**Degeneracy:** HOSA applies **Tikhonov regularization** ($\Sigma_{reg} = \Sigma + \lambda I$, with small $\lambda$) to ensure invertibility in collinearity scenarios.

### 4.6. Robustness of the Mahalanobis Distance under Normality Violations

#### 4.6.1. Nature of Expected Violations

Three classes of violation are empirically prevalent in operating system metrics:

| Violation Class | Example in Kernel Metrics | Impact on $D_M$ |
|---|---|---|
| **Heavy-tailed** | Disk I/O latency: most operations complete in microseconds, but outliers occur more frequently than predicted by the normal distribution. | $D_M$ underestimates the frequency of legitimate extreme values, potentially generating false positives in tail events. |
| **Skewness** | CPU utilization: distribution often concentrated near 0% (idle) or near 100% (saturated). | $\vec{\mu}$ and $\Sigma$ may not adequately represent the actual distribution, displacing the detector. |
| **Multimodality** | Systems alternating between two distinct operational regimes. | $\vec{\mu}$ calculated as arithmetic mean sits between the two modes. $D_M$ classifies the normal behavior of both modes as anomalous. |

#### 4.6.2. Evidence of Robustness in the Literature

- **Gnanadesikan & Kettenring (1972)** demonstrated that covariance-based estimators maintain discriminative capability under non-normal elliptical distributions, preserving the **relative ordering** of anomalies.
- **Penny (1996)** confirmed graceful degradation under various non-Gaussian distributions.
- **Hubert, Debruyne & Rousseeuw (2018)** demonstrated preserved detection efficacy under contamination of up to 25% of samples by outliers.

HOSA operates primarily on the **rate of change** of $D_M$ (derivatives), not on its absolute value. Even if the absolute value loses its exact probabilistic interpretation under non-normality, the derivatives remain valid indicators of **acceleration toward collapse**.

#### 4.6.3. Mitigation Strategy: Robust Estimation

**Level 1 — Regularization (default).** Tikhonov regularization already applied mitigates sensitivity to outliers.

**Level 2 — Robust estimation (conditional activation).** When HOSA detects severe normality violations via Mardia's multivariate kurtosis (Mardia, 1970):

$$\kappa_M = \frac{1}{N} \sum_{i=1}^{N} \left[(\vec{x}_i - \vec{\mu})^T \Sigma^{-1} (\vec{x}_i - \vec{\mu})\right]^2$$

compared with the expected value under normality $\kappa_{expected} = n(n+2)$, the agent can replace the estimators of $\vec{\mu}$ and $\Sigma$ with the **Minimum Covariance Determinant (MCD)** (Rousseeuw, 1984), implemented via the FAST-MCD algorithm (Rousseeuw & Van Driessen, 1999).

**Footprint impact:** Incremental MCD requires storing a recent sample window (typically 100–500 samples), occupying approximately 40KB for $n = 10$ — negligible in any operational context.

#### 4.6.4. Multimodality and Interaction with Seasonal Profiles

The multimodality problem is partially addressed by the temporal-context-indexed baseline profiles mechanism (Section 6.6). When multimodality is not temporally segregable, the approach requires extension to **Mixture of Gaussians** with streaming Expectation-Maximization (Engel & Heinen, 2010) — documented as a future research direction.

#### 4.6.5. Empirical Validation Plan

Experimental validation will include:
1. Real kernel metric data collection (minimum 72 continuous hours per scenario);
2. Multivariate normality tests: Mardia's kurtosis, Henze-Zirkler test (Henze & Zirkler, 1990), QQ-plot inspection;
3. Comparative benchmarking of TPR and FPR under classical estimation, robust estimation (MCD), and pre-transformed Mahalanobis;
4. Computational footprint impact analysis of each alternative.

---

## 5. Engineering Architecture

### 5.1. Architectural Principles

HOSA's design is governed by five non-negotiable principles:

| # | Principle | Description |
|---|---|---|
| 1 | **Local Autonomy** | HOSA must execute its complete detection and mitigation cycle without dependency on network, external APIs, or human intervention for its primary function. |
| 2 | **Zero External Runtime Dependencies** | The agent does not depend on external services (TSDB, message brokers, cloud APIs) to operate. All dependencies are internal to the binary or to the host operating system kernel. Communication with external systems is **opportunistic**: performed when available, never required. |
| 3 | **Predictable Computational Footprint** | HOSA's CPU and memory consumption must be constant and predictable ($O(1)$ in memory, configurable and bounded CPU percentage). The agent cannot become the cause of the problem it intends to solve. |
| 4 | **Graduated Response** | Mitigation is not binary. HOSA implements a spectrum of responses proportional to the anomaly's severity and rate of change, from light priority adjustment to complete network isolation. |
| 5 | **Decision Observability** | Every autonomous action by HOSA is recorded locally with mathematical justification (values of $D_M$, derivative, triggered threshold, executed action). The agent is **auditable**. |

### 5.2. Execution Model: The Perceptive-Motor Cycle

HOSA operates in a continuous cycle with three functional layers, inspired by the biological separation between the sensory system, nervous system, and motor system:

```
┌─────────────────────────────────────────────────────────────┐
│                    KERNEL SPACE (eBPF)                      │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │   Sensory    │  │   Sensory    │  │    Actuators      │  │
│  │   Probes     │  │   Probes     │  │   (XDP / cgroup   │  │
│  │ (tracepoints │  │ (kprobes,    │  │  / sched_ext /    │  │
│  │  scheduler,  │  │  PSI hooks,  │  │  mm compaction)   │  │
│  │  mm, net)    │  │  mm/vmstat)  │  │                   │  │
│  └──────┬───────┘  └──────┬───────┘  └────────▲──────────┘  │
│         │                 │                   │             │
│         ▼                 ▼                   │             │
│  ┌──────────────────────────────┐             │             │
│  │     eBPF Ring Buffer         │             │             │
│  │  (events to user space)      │             │             │
│  └──────────────┬───────────────┘             │             │
│                 │                             │             │
├─────────────────┼─────────────────────────────┼─────────────┤
│                 │    USER SPACE               │             │
│                 ▼                             │             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              MATHEMATICAL ENGINE (Go)                 │  │
│  │                                                       │  │
│  │  1. Receives events from ring buffer                  │  │
│  │  2. Updates state vector x(t) + H_frag               │  │
│  │  3. Updates μ and Σ incrementally (Welford)           │  │
│  │  4. Calculates D_M(x(t))                              │  │
│  │  5. Applies EWMA → D̄_M(t)                             │  │
│  │  6. Calculates dD̄_M/dt and d²D̄_M/dt²                  │  │
│  │  7. Evaluates against adaptive thresholds             │  │
│  │  8. Determines response level (0-5)                   │  │
│  │  9. Selects actuation regime (Phase 1 or Phase 2)     │  │
│  │ 10. Sends actuation command via BPF maps              │  │
│  │                                                       │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │          OPPORTUNISTIC COMMUNICATION (Go)             │  │
│  │  - Webhooks to orchestrators (when available)         │  │
│  │  - Metrics exposure (local endpoint)                  │  │
│  │  - Structured local log (audit)                       │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**Architectural note on kernel↔user space transition.** The transition uses the eBPF ring buffer and BPF maps, with typical latency on the order of **microseconds** (1–10μs on modern hardware). The correct terminology is **"zero external runtime dependencies"**: HOSA does not depend on processes, services, or external infrastructure beyond the agent binary and the host kernel.

### 5.3. Warm-Up Phase and Proprioceptive Calibration

Upon startup, HOSA executes the **Hardware Proprioception** sequence:

1. **Topology discovery:** Via reading of `/sys/devices/system/node/` and `/sys/devices/system/cpu/`, identifies NUMA topology, core counts, L1/L2/L3 cache sizes, L3 sharing map per core pair, and memory configuration per NUMA zone.
2. **State vector definition:** Determines which variables to include in $\vec{x}(t)$ and their eBPF sources.
3. **Baseline accumulation:** During a configurable period (default: 5 minutes), collects samples without executing mitigation, building initial $\vec{\mu}_0$ and $\Sigma_0$ via incremental Welford.
4. **$\alpha$ calibration (EWMA):** The smoothing factor is calibrated per resource based on observed signal variance during warm-up.
5. **Adaptive threshold definition:** $D_M$ thresholds for each response level are calculated as multiples of the standard deviation observed in the baseline regime.
6. **`sched_ext` support detection:** HOSA checks whether the running kernel supports `sched_ext` (Linux ≥ 6.11 with `CONFIG_SCHED_CLASS_EXT=y`). If supported, the Survival Scheduler BPF program is pre-compiled and held in standby for immediate activation upon reaching Level 3 response.

### 5.4. Graduated Response System

HOSA implements **six response levels** (0–5):

| Level | Activation Condition | Action | Reversibility |
|---|---|---|---|
| **0 — Homeostasis** | $D_M < \theta_1$ and $\frac{dD_M}{dt} \leq 0$ | None. Suppresses redundant telemetry (sends minimal heartbeat). | N/A |
| **1 — Vigilance** | $D_M > \theta_1$ or $\frac{dD_M}{dt} > 0$ sustained | Local logging. Increased sampling frequency. No intervention. | Automatic |
| **2 — Light Containment** | $D_M > \theta_2$ and $\frac{dD_M}{dt} > 0$ | Renice of non-essential processes via cgroups. Opportunistic webhook. | Automatic |
| **3 — Active Containment** | $D_M > \theta_3$ and $\frac{d^2D_M}{dt^2} > 0$ | CPU/memory throttling in cgroups. Partial XDP load shedding. **Survival Scheduler activation via `sched_ext`**. **Preemptive memory defragmentation** if $H_{frag}$ below warning threshold. Urgent webhook. | Automatic with hysteresis |
| **4 — Severe Containment** | $D_M > \theta_4$ or TTF < T_critical | Aggressive throttling. XDP blocks all inbound except healthcheck. **Targeted Starvation** of the causal process. **Page Table Isolation** for the invasive process. | Requires sustained $D_M$ reduction |
| **5 — Autonomous Quarantine** | Containment failure + $D_M$ in uncontrolled ascent. | Network isolation. Non-essential processes frozen (SIGSTOP). Detailed log written to persistent storage. Final webhook attempt. | **Manual** |

### 5.4.1. Quarantine Modes by Environment Class

| Environment Class | Automatic Detection | Quarantine Strategy | Recovery Mechanism |
|---|---|---|---|
| **Bare metal with IPMI/iLO/iDRAC** | IPMI interface detection via `/sys/class/net/` and `ipmi_*` kernel modules. | Deactivation of **all** network interfaces **except** out-of-band management interface. | Manual via IPMI console. |
| **Cloud VM (AWS, GCP, Azure)** | DMI/SMBIOS detection, presence of metadata service (169.254.169.254). | **Does not deactivate network interfaces.** XDP drops all traffic except metadata service, DHCP, and orchestrator API. Signals quarantine state via native cloud provider mechanism. | External orchestrator terminates and replaces the instance. |
| **Kubernetes (DaemonSet)** | Container detection via `/proc/1/cgroup` namespace and `KUBERNETES_SERVICE_HOST` env. | Applies maximum cgroup containment. Updates Node via Kubernetes API with taint `hosa.io/quarantine=true:NoExecute`. | Operator removes taint after investigation. |
| **Edge/IoT (physical access)** | Explicit configuration (`environment: edge-physical`). | Complete network interface deactivation. Logs preserved in persistent local storage. | Manual by field technician. |
| **Edge/IoT (remote only)** | Explicit configuration (`environment: edge-remote`). | Network deactivation + hardware watchdog timer (default: 30 min timeout → auto-reboot). Post-reboot: conservative mode for configurable period. | Automatic via watchdog reboot + observation period. |
| **Air-gapped (SCADA/ICS)** | Explicit configuration (`environment: airgap`). | Identical to bare metal, with all opportunistic communication permanently disabled. | Manual with authorized physical access. |

**Principle of conservative default:** In ambiguous cases, HOSA assumes the **most conservative quarantine mode** (cloud VM — XDP-only, no interface deactivation), prioritizing recoverability over isolation.

**Note on containers and privilege.** Levels 0–2 require only `CAP_BPF` and read access to `/sys/`. Levels 3–4 additionally require `CAP_SYS_ADMIN` for cgroup manipulation. Level 5 requires `CAP_NET_ADMIN` for XDP manipulation and, in Kubernetes mode, cluster API access for taint application.

### 5.5. Habituation: Adaptation to the New Baseline

HOSA implements a **habituation** mechanism inspired by neuroplasticity via **exponential decay of weights** in the Welford algorithm. The mechanism is conditioned on a set of preconditions (see Section 6.12) that block habituation in pathological regimes.

### 5.6. Selectivity Policy: The Throttling Problem

Throttling of processes via cgroups introduces secondary risks (cascade timeouts, transaction deadlocks, critical component starvation). HOSA addresses these through a **protection list** (safelist): kernel processes, the HOSA agent itself, and orchestration agents (kubelet, containerd, dockerd). Throttling is preferentially applied to the **largest contributors** to the anomaly, determined by the dimensional contribution decomposition $c_j$.

### 5.7. Scenario Walkthrough: Memory Leak in Payment Microservice

This section presents an end-to-end scenario illustrating HOSA's perceptive-motor cycle in operation. All numerical values are representative and based on behavior observed in production systems.

#### Context

- **Node:** VM `worker-node-07` in Kubernetes cluster, 8 vCPUs, 16GB RAM.
- **Workload:** 12 pods, including `payment-service-7b4f` (Java payment microservice, 2GB memory allocated via cgroup).
- **Exogenous monitoring:** Prometheus with 15-second scrape interval, Alertmanager rule: `container_memory_usage_bytes > 1.8GB for 1m`.
- **HOSA:** Operating in homeostasis (Level 0) for 6 hours. Baseline calibrated. 8-dimensional state vector.

#### Timeline

**t = 0s — Memory Leak Start**

`payment-service-7b4f` starts allocating objects not collected by Java's GC. Leak rate: ~50MB/s.

```
Vector x(t):
  cpu_total:     47%    (baseline: 45% ± 8%)
  mem_used:      61%    (baseline: 58% ± 5%)
  mem_pressure:  12%    (baseline: 10% ± 4%)
  io_throughput: 340 IOPS  io_latency: 2.1ms
  net_rx: 1,200 req/s  net_tx: 1,180 resp/s
  runqueue: 3.2

D_M = 1.1  (θ₁=3.0, θ₂=5.0, θ₃=7.0, θ₄=9.0)
Level: 0 (Homeostasis)
```

**t = 1s — HOSA detects initial deviation**

```
  mem_used: 64%  mem_pressure: 18%
  D_M = 2.8  φ = +0.9
  dD̄_M/dt = +1.6/s  d²D̄_M/dt² = +1.6/s²
  Level: 0→1 (Vigilance)
```
Sampling increased 100ms → 10ms. No system intervention. No webhook.

**t = 2s — Escalation to Light Containment**

```
  mem_used: 68%  mem_pressure: 29%
  cpu_total: 52%  io_latency: 3.8ms
  D_M = 4.7  φ = +1.8
  dD̄_M/dt = +2.1/s  d²D̄_M/dt² = +0.5/s²
  ρ(t) = 0.31  (CPU↔memory correlation altered — GC activity)
  Level: 1→2 (Light Containment)
```

Dimensional decomposition: `mem_used` contributes 68% of $D_M^2$. Contributing cgroup identified: `/kubepods/pod-payment-service-7b4f/` (+102MB in last second). **Action:** `memory.high` reduced from `2G` to `1.6G`. Opportunistic webhook fired with full dimensional context.

**t = 4s — Containment holding, derivative decelerating**

```
  D_M = 5.9  dD̄_M/dt = +1.2/s (decelerating from +2.1)
  d²D̄_M/dt² = -0.45/s²  (NEGATIVE — containment working)
  Level: 2 (maintained)
```

`kubelet`, `containerd`, and other pods **not affected** — on safelist.

**t = 8s — Stabilization by containment**

```
  mem_used: 74% (stable)  D_M = 6.2
  dD̄_M/dt ≈ 0  d²D̄_M/dt² ≈ 0  Level: 2 (maintained)
```

System contained at a degraded but functional plateau. Transactions preserved.

**t = 35s — Operator receives HOSA webhook**

```json
{
  "severity": "warning",  "node": "worker-node-07",
  "hosa_level": 2,  "d_m": 4.7,  "d_m_derivative": 2.1,
  "dominant_dimension": "mem_used",  "dominant_contribution_pct": 68,
  "suspected_cgroup": "/kubepods/pod-payment-service-7b4f",
  "action_taken": "memory.high reduced to 1.6G",
  "action_status": "effective (d2DM/dt2 < 0)"
}
```

**t = 60s — Counterfactual without HOSA**

Without HOSA: ~3GB allocated by t=60s, exceeding the 2GB cgroup limit. OOM-Kill at t≈40s. All in-flight payment transactions aborted without graceful shutdown. CrashLoopBackOff. Prometheus alert at t≈100s — **60 seconds after the first crash.**

#### Temporal Synthesis

HOSA transformed a **destructive crash with transaction loss** scenario into **controlled degradation with functionality preservation**. Detection time: 1 second (vs. >60s exogenous). Mitigation preserved transaction integrity. Operator received actionable information with complete dimensional context.

---

## 6. Taxonomy of Operational Regimes and HOSA's Behavioral Classification

### 6.1. The Demand Classification Problem

The effectiveness of an anomaly detection system depends fundamentally on its ability to **distinguish between legitimate variation and pathological deterioration**. A detector that treats every deviation as a threat generates operational fatigue from false positives. An overly tolerant detector allows sophisticated attacks to operate below the perception threshold.

The challenge is compounded by the fact that radically different scenarios can produce superficially similar signatures. CPU at 85% may mean: a normal day for a video rendering server; a predictable Black Friday seasonal spike; the first milliseconds of a volumetric DDoS attack; or a silent cryptominer consuming idle cycles. The isolated metric is identical.

Equally critically, the taxonomy must recognize that anomaly is not exclusively a phenomenon of **excess**. CPU at 2% on a server that should be processing a thousand requests per second is not homeostasis — it is **anomalous silence**, with financial, energetic, and security implications that anomaly detection literature has historically ignored.

---

### 6.2. The Continuous Bipolar Spectrum: Taxonomy Architecture

#### 6.2.1. Organizing Principle

HOSA's taxonomy models operational regimes as a **continuous numeric spectrum centered on homeostasis**:

```
    Under-demand                   Over-demand / Anomaly
    ◄──────────────────────┤├───────────────────────────────────►

    −3      −2      −1      0      +1     +2     +3     +4     +5
    │       │       │       │       │      │      │      │      │
 Anomalous Struct- Legit-  Homeo- Baseline Season- Adver- Local  Viral
 Silence  ural   imate   stasis  Shift  ality  sarial Fail.  Prop.
         Idle    Idle
```

#### 6.2.2. Design Rationale

**Conceptual symmetry.** Biological homeostasis is bidirectional: hypothermia and hyperthermia are both pathologies. HOSA treats under-demand and over-demand as symmetric deviations from the baseline profile.

**Numerical continuity.** The regime integer index reflects a natural ordering of severity in each semi-axis.

**Uniformity of the mathematical framework.** The same primary metric ($D_M$) and Load Direction Index ($\phi$) position any observed state in the spectrum.

#### 6.2.3. Directionality: Extending the Mahalanobis Distance

The Mahalanobis Distance is inherently **non-directional**. To position the state in the bipolar spectrum, HOSA defines the **Load Direction Index ($\phi$)**:

$$\phi(t) = \frac{1}{n} \sum_{j=1}^{n} s_j \cdot \frac{d_j(t)}{\sigma_j}$$

where $d_j(t) = x_j(t) - \mu_j$, $\sigma_j = \sqrt{\Sigma_{jj}}$, and $s_j \in \{+1, -1\}$ is the **load sign** of variable $j$ (+1 if an increase indicates higher load: CPU utilization, memory used, network throughput; −1 if an increase indicates lower load: CPU idle, free memory).

| Value of $\phi(t)$ | Meaning | Semi-axis |
|---|---|---|
| $\phi \approx 0$ | System near baseline | Regime 0 |
| $\phi > 0$ | Deviation toward **overload** | Positive semi-axis (+1 to +5) |
| $\phi < 0$ | Deviation toward **idleness** | Negative semi-axis (−1 to −3) |

---

### 6.3. Regime 0 — Operational Homeostasis

**Definition:** The normal steady state of the node under its typical workload.

**Mathematical signature:** $D_M$ low and stable; $\phi$ oscillates around zero; $\frac{d\bar{D}_M}{dt}$ oscillates around zero; $\Sigma$ stable.

**HOSA behavior:** Level 0. **Thalamic Filter active**: only a minimal heartbeat is emitted, confirming that the node is alive and in homeostasis. Baseline continuously refined via Welford.

---

### 6.4. Negative Semi-Axis: Under-Demand Regimes (−1, −2, −3)

#### 6.4.1. Rationale for Inclusion

The entirety of the anomaly detection literature concentrates on **anomaly by excess**. By focusing exclusively on positive anomaly, the industry systematically ignores a phenomenon with equally significant financial, energetic, and security implications: **anomaly by deficit**.

A server that should be processing a thousand requests per second and is processing zero is not in homeostasis. It is in **anomalous silence**. That silence has a cost: the machine continues consuming electricity, occupying rack space, depreciating hardware, and generating licensing costs — all without producing value.

---

#### 6.4.2. Regime −1 — Legitimate Idleness

**Definition:** Demand reduction compatible with the temporal or operational context — coherent with the baseline profile of the corresponding time window (e.g., nighttime on a corporate web server; weekend on an ERP; scheduled upstream maintenance).

**Mathematical signature:**

| Indicator | Behavior |
|---|---|
| $D_M(t)$ | Elevated relative to global baseline, but **low** relative to the temporal window's baseline profile. |
| $\phi(t)$ | Moderately negative. |
| $\frac{d\bar{D}_M}{dt}$ | Approximately zero or smooth transition. |
| $\rho(t)$ | **Low** — correlation structure preserved. Less network → less CPU → less I/O, proportionally. |
| Temporal context | **Coherent** — period corresponds to historically low-activity window. |

**HOSA behavior:** Level 0. Thalamic Filter maximally active. FinOps signaling: HOSA records underutilization metrics locally and exposes an **idleness report** quantifying accumulated idle hours, estimated cost, and downscale window recommendation. GreenOps energy optimization: CPU frequency reduction via scaling governor; network interface polling reduction; HOSA's own sampling interval increase. All optimizations are **instantly reversible** when $\phi(t)$ starts rising.

---

#### 6.4.3. Regime −2 — Structural Idleness

**Definition:** The node is **permanently** oversized relative to actual demand — no time window in which resources are fully utilized.

**Dedicated metric: Excess Provisioning Index (EPI)**

$$EPI = 1 - \frac{\max_{i \in \text{windows}} \|\vec{\mu}_i\|_{load}}{\vec{C}_{max}}$$

An EPI close to 1 indicates severe oversizing.

**HOSA behavior:** Level 0. Critical FinOps signaling: **oversizing report** containing calculated EPI, per-resource utilization vs. capacity, right-sizing suggestion compatible with maximum observed load, projected annual savings estimate. The HOSA does not autonomously decide to shut down or resize the node — it **provides the mathematical evidence** for the human or orchestrator to make an informed decision. Habituation permitted with persistent FinOps signaling.

---

#### 6.4.4. Regime −3 — Anomalous Silence

**Definition:** Abrupt or gradual drop in activity **incompatible** with the expected temporal context (e.g., traffic redirected by DNS hijacking; silent load balancer failure; application process killed without restart; attack that brought down the service before installing payload).

**Mathematical signature:**

| Indicator | Behavior |
|---|---|
| $D_M(t)$ | **Abrupt elevation** (even though load dropped — deviation from baseline is large). |
| $\phi(t)$ | **Strongly negative**, with rapid transition. |
| $\frac{d\bar{D}_M}{dt}$ | **Abrupt positive peak**. |
| $\frac{d\phi}{dt}$ | **Abrupt** — rapid transition to negative. |
| Temporal context | **Incoherent** — drop occurs at a time that should be active. |

**HOSA behavior:** Level 1 (Vigilance) to 3 (Active Containment). Active investigation: process verification (are expected application processes still running?); network verification (are interfaces operational?); upstream verification (health check reverse if upstream endpoints are known). High-priority webhook: "Node X reports activity significantly below expected for temporal context."

**The paradox of silence as an alarm:** Traditional monitors report "all healthy" when a server stops receiving traffic (low CPU, free memory, calm network). HOSA, by modeling the expected baseline profile, detects that the silence itself is anomalous.

**Interaction with habituation:** **Blocked.** HOSA never habituates to silence incoherent with the temporal context.

---

#### 6.4.5. Consolidated Mathematical Signature — Negative Semi-Axis

| Indicator | Regime −1 (Legitimate) | Regime −2 (Structural) | Regime −3 (Anomalous) |
|---|---|---|---|
| $D_M(t)$ vs. global baseline | Moderate | Chronically low | High (abrupt) |
| $D_M(t)$ vs. temporal profile | **Low** (coherent) | Low in all windows | **High** (incoherent) |
| $\phi(t)$ | Moderately negative | Persistently negative | **Strongly negative** |
| $\frac{d\phi}{dt}$ | Gradual | ≈ 0 (stable) | **Abrupt** |
| $\rho(t)$ | Low | Low | Variable |
| Temporal coherence | **Yes** | Irrelevant | **No** |
| $EPI$ | Variable | **Close to 1** | Irrelevant |

#### 6.4.6. Theoretical Contribution of Sub-Demand Detection

The inclusion of the negative semi-axis enables three practical contributions absent from existing local agents:

**1. FinOps grounded in endogenous evidence.** Second-level granularity evidence of underutilization, including multivariable correlation and temporal context, enabling higher-precision right-sizing recommendations.

**2. GreenOps as a consequence of homeostasis.** Energy optimization is the **natural response of the agent to the under-demand regime** — exactly as biological metabolism reduces energy consumption at rest. Homeostasis is bidirectional.

**3. Operational blackout detection as a security capability.** Anomalous Silence is a genuine security scenario that traditional resource health monitors are structurally incapable of detecting — all capacity metrics are "healthy" when the server stops receiving work.

---

### 6.5. Regime +1 — High Baseline Demand (Permanent Plateau Shift)

**Definition:** A **persistent and unreversed** elevation in resource consumption caused by legitimate workload changes (e.g., new version deployment, additional microservice migration, organic user base growth).

**Key discriminant:** The derivative converging to zero while $D_M$ remains elevated. Differentiates from an ongoing attack, where the derivative remains positive or accelerates.

**Covariance deformation ratio:**

$$\rho(t) = \frac{\|\Sigma_{recent} - \Sigma_{baseline}\|_F}{\|\Sigma_{baseline}\|_F}$$

Low $\rho$ with high $D_M$ indicates plateau shift with structure preservation (Regime +1). High $\rho$ indicates **covariance structure deformation** (potentially Regime +3 or +4).

**Interaction with habituation:** This regime is the **primary use case for habituation.** When stability and covariance preservation criteria are satisfied, HOSA recalibrates $\vec{\mu}$ and $\Sigma$ to reflect the new operational regime.

**Safeguard against premature habituation:** Habituation is **not triggered** if the stabilization occurs near physical resource limits (e.g., memory > 90%), or if the SLM (Phase 4) identifies compromise indicators simultaneous to the elevation.

---

### 6.6. Regime +2 — Seasonal High Demand (Predictable Periodicity)

**Definition:** Demand variations following recurring temporal patterns (e.g., daily access peaks between 09:00–11:00 in corporate applications; nighttime traffic drops; weekly peaks; monthly seasonality; annual seasonality like Black Friday).

**Solution: Time-Context-Indexed Baseline Profiles (Digital Circadian Rhythm)**

HOSA implements **temporal baseline segmentation**, maintaining **N baseline profiles** indexed by time window:

$$\mathcal{B} = \{(\vec{\mu}_i, \Sigma_i, w_i) \mid i = 1, 2, \ldots, N\}$$

Segmentation granularity is determined automatically via **autocorrelation analysis** of the $D_M$ time series. If periodicity is detected (e.g., 24h lag peak), $\mathcal{B}$ is automatically segmented into corresponding windows; each segment accumulates its own baseline via independent Welford. The $D_M$ calculation at each instant $t$ uses the baseline profile corresponding to the current time window:

$$D_M(t) = \sqrt{(\vec{x}(t) - \vec{\mu}_{i(t)})^T \Sigma_{i(t)}^{-1} (\vec{x}(t) - \vec{\mu}_{i(t)})}$$

**Cyclic encoding** of temporal variables avoids discontinuities (23h→0h):

$$x_{hour,sin}(t) = \sin\left(\frac{2\pi \cdot hour(t)}{24}\right), \quad x_{hour,cos}(t) = \cos\left(\frac{2\pi \cdot hour(t)}{24}\right)$$

**Interaction with habituation:** Habituation occurs **within each temporal segment**, not globally.

---

### 6.7. Regime +3 — Disguised High Demand (Adversarial Demand)

**Definition:** Resource consumption caused by malicious activity that **deliberately mimics legitimate demand patterns** to evade detection (e.g., Layer 7 DDoS; parasitic cryptomining; Low-and-Slow data exfiltration; resource exhaustion attacks).

**Central thesis:** Even when individual **magnitudes** are kept within normal range, malicious activity produces **deformation in the covariance structure** that legitimate demand does not produce.

**Second-Level Metrics — Structural Deformation Detection:**

**a) Shannon entropy of the syscall profile:**

$$H(S, t) = -\sum_{i=1}^{k} p_i(t) \log_2 p_i(t), \quad \Delta H(t) = |H(S, t) - H_{baseline}|$$

**b) Work Efficiency Index (WEI):**

$$WEI(t) = \frac{\text{application throughput}(t)}{\text{computational resource consumption}(t)}$$

Cryptomining and parasitic processing consume CPU/memory without producing application throughput, causing **WEI decline** even when no individual metric is in alert range.

**c) Kernel/User Context Ratio:**

$$R_{ku}(t) = \frac{\text{CPU in kernel mode}(t)}{\text{CPU in user mode}(t)}$$

Network attacks produce disproportionate increases in kernel space time.

**Interaction with habituation:** Habituation is **blocked** when the deformation ratio $\rho(t)$ exceeds the threshold. The condition for habituation is extended:

$$\text{Habituation allowed} \iff \left(\frac{d\bar{D}_M}{dt} \approx 0\right) \wedge \left(\rho(t) < \rho_{threshold}\right) \wedge \left(\Delta H(t) < \Delta H_{threshold}\right)$$

---

### 6.8. Regime +4 — Non-Viral Anomaly (Localized Failure)

**Definition:** Resource deterioration caused by failure or pathology **confined to the local node**, without a propagation component (e.g., memory leak; disk degradation; file descriptor accumulation; fork bomb; deadlock; CPU thermal degradation).

**Dimensional Contribution Decomposition:**

Given the deviation vector $\vec{d} = \vec{x}(t) - \vec{\mu}$ and $D_M^2 = \vec{d}^T \Sigma^{-1} \vec{d}$, the contribution of the $j$-th dimension is:

$$c_j = d_j \cdot (\Sigma^{-1} \vec{d})_j$$

The dimensions with highest $c_j$ are the **dominant contributors** of the anomaly. This allows HOSA to direct throttling to the most consuming processes, log the **mathematical reason** for each decision, and provide dimensional context for causal diagnosis (Phase 4).

**Interaction with habituation:** **Blocked when the derivative remains sustainedly positive.** Monotonically growing anomalies are not "new normals" — they are progressive failures.

---

### 6.9. Regime +5 — Viral Anomaly (Propagation and Contagion)

**Definition:** Malicious activity or cascade failure with a **propagation component between nodes** (e.g., worms and malware with lateral propagation; post-compromise lateral movement; microservice failure cascade; compromised node used as internal DDoS amplifier).

**Formal metric: Propagation Behavior Index (PBI)**

$$PBI(t) = w_1 \cdot \hat{C}_{out}(t) + w_2 \cdot \hat{H}_{dest}(t) + w_3 \cdot \hat{F}_{anom}(t) + w_4 \cdot \hat{\rho}_{D_M \leftrightarrow net_{out}}(t)$$

where $\hat{C}_{out}(t)$ is the normalized rate of new outbound connections; $\hat{H}_{dest}(t)$ is the normalized entropy of destination IPs; $\hat{F}_{anom}(t)$ is the normalized rate of anomalous forks/execs; $\hat{\rho}_{D_M \leftrightarrow net_{out}}(t)$ is the correlation between $D_M$ and outbound traffic.

**Weight Calibration Strategy:**

**Stage 1 — Uniform Initialization.** $w_i = \frac{1}{4}$ for $i \in \{1, 2, 3, 4\}$ — conservative, unbiased prior.

**Stage 2 — Calibration via AUC-ROC maximization** over controlled attack scenarios with known ground truth:

$$\vec{w}^* = \arg\max_{\vec{w}} \text{AUC-ROC}\left(\{PBI^{(j)}(\vec{w}), y^{(j)}\}_{j=1}^{M}\right)$$

**Stage 3 — Leave-one-out cross-validation and publication** of final $\vec{w}^*$ values as reference parameters.

**HOSA behavior:** PBI low + $D_M$ high → local containment, no network isolation. PBI high + $D_M$ high → **network isolation** prioritized (Level 4-5). PBI high + $D_M$ moderate → selective containment + outbound connection restriction via XDP.

**Interaction with habituation:** **Categorically blocked** when $PBI > PBI_{threshold}$.

---

### 6.10. Exogenous Contextual Signals as Supplementary State Vector Dimensions

The most fundamental contextual signal — requiring no external dependency — is **time**. Cyclic encoding avoids discontinuities. In Edge Computing and industrial IoT scenarios, environmental signals (ambient temperature, humidity, supply voltage, vibration) can be incorporated into $\vec{x}(t)$. The Covariance Matrix automatically captures correlations between environmental conditions and resource metrics, allowing HOSA to **discount** performance variations caused by physical environmental factors.

**Design principle: graceful degradation.** If no environmental sensors are available, HOSA operates without those dimensions. Their presence **improves** classification; their absence **does not prevent** functioning.

Operator-configurable contextual signals include: event calendar (preemptive threshold relaxation for planned peaks), workload profile (relative weight calibration per resource type), geographic zone (for Phase 5+ swarm context), and client time zones (temporal segmentation refinement).

---

### 6.11. Synthesis: Integrated Classification Matrix

| Regime | $D_M$ | $\frac{dD_M}{dt}$ | $\frac{d^2D_M}{dt^2}$ | $\phi(t)$ | $\rho(t)$ | $\Delta H$ | $PBI$ | Classification |
|---|---|---|---|---|---|---|---|---|
| **−3** | High (abrupt) | Peak | Variable | **Strongly negative** | Variable | Variable | Variable | **Anomalous Silence** → Investigation |
| **−2** | Chronically low | ≈ 0 | ≈ 0 | **Persistently negative** | Low | Low | Low | **Oversizing** → FinOps |
| **−1** | Low (vs. temporal) | ≈ 0 or smooth | ≈ 0 | **Negative** | Low | Low | Low | **Legitimate Idleness** → FinOps/GreenOps |
| **0** | Low | ≈ 0 | ≈ 0 | ≈ 0 | Low | Low | Low | **Homeostasis** |
| **+1** | High, stable | ≈ 0 (after transient) | ≈ 0 | **Positive** | Low | Low | Low | **Plateau shift** → Habituation |
| **+2** | Oscillates | Oscillates | Oscillates | **Oscillates** | Low | Low | Low | **Seasonality** → Temporal profiles |
| **+3** | Any | Any | Any | Positive | **High** | **High** | Variable | **Adversarial** → Containment |
| **+4** | Growing | Sustained positive | Variable | Positive | Variable | Low | **Low** | **Localized failure** → Graduated containment |
| **+5** | Variable | Variable | Variable | Variable | Variable | Variable | **High** | **Propagation** → Network isolation |

**Note on ambiguous classification:** HOSA adopts the **precautionary principle**: classifies temporarily as the highest severity regime compatible with observed data. The audit log records the ambiguity and indicators that led to the decision.

**Note on cross-axis transitions:** Regime −3 (Anomalous Silence) can transition to the positive semi-axis when investigation reveals compromise indicators (high PBI, anomalous processes). The state is then directly reclassified as Regime +5 (Viral Propagation). Traversal of zero without stopping at homeostasis is recorded as a high-priority event.

---

### 6.12. Habituation Mechanism Implications: Consolidated Rules

**Necessary preconditions (all must be satisfied simultaneously):**

$$\text{Habituation} \iff \begin{cases} \left|\frac{d\bar{D}_M}{dt}\right| < \epsilon_d & \text{(stabilization)} \\ \rho(t) < \rho_{threshold} & \text{(covariance preserved)} \\ \Delta H(t) < \Delta H_{threshold} & \text{(stable syscalls)} \\ PBI(t) < PBI_{threshold} & \text{(no propagation)} \\ D_M(t) < D_{M,safety} & \text{(safe plateau)} \\ t_{stable} > T_{min} & \text{(sustained stabilization)} \\ \text{temporal coherence of } \phi(t) & \text{(if } \phi < 0\text{, coherent with seasonal profile)} \end{cases}$$

| Regime | Habituation |
|---|---|
| **−3 — Anomalous Silence** | **Blocked** |
| **−2 — Structural Idleness** | Permitted (with persistent FinOps signaling) |
| **−1 — Legitimate Idleness** | Incorporated into seasonal profiles |
| **0 — Homeostasis** | N/A (is the baseline) |
| **+1 — Plateau shift** | **Permitted** if preconditions satisfied |
| **+2 — Seasonality** | Intra-segment |
| **+3 — Adversarial** | **Blocked** |
| **+4 — Localized failure** | **Blocked** while derivative is positive |
| **+5 — Viral/Propagation** | **Categorically blocked** |

**Visual pattern:** Habituation is permitted in the central spectrum regimes (−2 to +2), where deviations are legitimate or structural. It is blocked at the extremes (−3, +3 to +5), where deviations are pathological or adversarial.

---

### 6.13. Summary of Supplementary Metrics

| Metric | Symbol | Definition | Section |
|---|---|---|---|
| Load Direction Index | $\phi(t)$ | Normalized weighted projection of deviation onto load axis — indicates direction (overload vs. idleness) | 6.2.3 |
| Excess Provisioning Index | $EPI$ | Ratio between provisioned capacity and maximum historical utilization | 6.4.3 |
| Covariance Deformation Ratio | $\rho(t)$ | Frobenius norm of the difference between recent and baseline covariance, normalized | 6.5 |
| Shannon entropy of syscall profile | $H(S, t)$ and $\Delta H(t)$ | Measure of diversity and change in the system call distribution | 6.7 |
| Work Efficiency Index | $WEI(t)$ | Ratio application throughput / resource consumption | 6.7 |
| Kernel/User Ratio | $R_{ku}(t)$ | Proportion of CPU time in kernel space vs. user space | 6.7 |
| Propagation Behavior Index | $PBI(t)$ | Weighted combination of viral activity indicators | 6.9 |
| Dimensional Contribution | $c_j$ | Decomposition of $D_M^2$ by state vector dimension | 6.8 |
| $D_M$ Autocorrelation | $ACF_{D_M}(\tau)$ | Autocorrelation function for periodicity detection | 6.6 |
| Memory Fragmentation Entropy | $H_{frag}(t)$ | Measure of disorder in the distribution of free physical pages by order and NUMA zone | 9.2 |

---

### 6.14. Theoretical Contribution of the Taxonomy

The taxonomy formalizes two distinctions frequently treated ad hoc in operational practice:

**1. Not every deviation from the baseline is an anomaly, and not every anomaly is a threat.** The bipolar spectral organization enables proportional responses: central regimes (−2 to +2) are treated with adaptation and optimization; extreme regimes (−3, +3 to +5) are treated with containment and isolation.

**2. Anomaly by deficit is as significant as anomaly by excess.** The spectrum symmetry around Regime 0 establishes that HOSA implements genuine homeostasis — bidirectional equilibrium — not just overload protection.

---

## 7. Language Choice: Trade-off Analysis

| Criterion | Go | Rust | C |
|---|---|---|---|
| **GC latency** | GC with sub-ms pauses (Go 1.22+), but non-deterministic. Mitigable with `sync.Pool`, pre-allocation, and `GOGC` tuning. | No GC. Deterministic latency. | No GC. Deterministic latency. |
| **eBPF interaction** | `internal/sysbpf` — proprietary wrapper over `SYS_BPF` via `golang.org/x/sys/unix`, with no third-party dependencies. Implements only the subset required by HOSA: `BPF_MAP_CREATE`, `BPF_MAP_LOOKUP_ELEM`, `BPF_PROG_LOAD`, and attach via `perf_event_open`. | `aya-rs` (active library, smaller ecosystem) or equivalent proprietary wrapper. | `libbpf` (kernel upstream reference) — more complete, but requires manual memory management. |
| **Development speed** | High. Fast compilation. Native concurrency (goroutines). | Medium. Borrow checker requires discipline. Slow compilation. | Low. Manual memory management. |
| **Memory safety** | Guaranteed by runtime. | Guaranteed by compiler (no runtime). | Programmer's responsibility. |
| **Academic adequacy** | Readable code, facilitates reproducibility. | Readable code with learning curve. | Prone to subtle bugs. |

**Note on package independence.** HOSA deliberately adopts the policy of **zero third-party dependencies for its primary function**. The only external dependency is `golang.org/x/sys`, an official Go project package with indefinite maintenance guarantee while Linux exists as a target. Everything else — ELF parser (`internal/sysbpf/loader.go`), BPF syscall wrapper (`internal/sysbpf/syscall.go`), cgroup manipulation (`internal/syscgroup`), and linear algebra (`internal/linalg`) — is code native to the repository. This decision eliminates the risk of dependency obsolescence, simplifies security auditing, and ensures portability to Linux distributions without access to external package registries (SCADA, air-gapped, embedded scenarios).

**Provisional decision:** Go for the mathematical engine and control plane, with the hot path computation implemented with minimal allocation (slice pre-allocation, `sync.Pool`, `GOGC=off` during critical cycles). The rationale is pragmatic: for a master's dissertation scope, Go's iteration speed allows greater focus on validating the mathematical thesis.

**Validation commitment:** The dissertation will include comparative benchmarks of the hot path measuring p50/p99 latency and jitter, with explicit discussion of whether observed GC pauses impact the detection window in real collapse scenarios. If GC pauses prove problematic in benchmarks, migration of the hot path to C (via CGo or auxiliary process) will be documented as future work.

---

## 8. Roadmap: Executable Horizon and Long-Term Vision

### 8.1. Executable Horizon (Dissertation Scope and Immediate Continuity)

#### Phase 1: Foundation — The Mathematical Engine and the Reflex Arc (v1.0)

**Scope:** Complete implementation of the perceptive-motor cycle with mitigation via cgroups v2 and XDP.

**Deliverables:**
- eBPF probes for state vector collection (CPU, memory, I/O, network, scheduler) via tracepoints and kprobes — implemented via `internal/sysbpf` (zero third-party dependencies)
- Mathematical engine with incremental Welford, Mahalanobis, EWMA, and derivatives
- Hardware proprioception (warm-up with automatic calibration)
- Graduated response system (Levels 0–4) based exclusively on logical cgroup and XDP manipulation
- Thalamic Filter: redundant telemetry suppression in homeostasis (minimal heartbeat)
- Benchmark of complete cycle latency (detection → decision → actuation)

**Experimental validation:**
- Controlled fault injection: gradual memory leak, fork bomb, CPU burn, network flood
- Quantitative comparison: HOSA detection and mitigation time vs. Prometheus+Alertmanager vs. systemd-oomd
- Sensitivity analysis of parameter $\alpha$ (EWMA) and adaptive thresholds
- Agent overhead measurement (CPU, memory, added system latency)

---

#### Phase 2: The Sympathetic Nervous System — Physical and Thermodynamic Intervention (v2.0)

**Scope:** Transition from passive mitigation (logical limits via cgroups) to active mitigation, altering the **physical rules of processor scheduling and physical memory topology** at runtime via eBPF. The goal is to eliminate reactive containment and replace the kernel's standard algorithms — which prioritize fairness and general purpose — with **deterministic survival algorithms** during the Lethal Interval.

The biological metaphor governing this phase is precise: while Phase 1 acts as the spinal reflex arc (fast, localized response), Phase 2 acts as the **sympathetic nervous system** under acute stress — actively redistributing vital resource flow to survival organs, depriving peripherals, and altering the organism's own metabolic rules. An animal in flight does not distribute blood flow equally between all organs; it diverts it from digestive processes to skeletal muscles. HOSA Phase 2 performs the computational equivalent of this physiological redistribution.

##### 8.2.1. CPU Deliverables: The Survival Scheduler via `sched_ext`

The standard operating system uses the CFS (Completely Fair Scheduler) or, in more recent kernels (≥ 6.6), EEVDF (Earliest Eligible Virtual Deadline First). The fundamental architectural assumption of these algorithms is **fairness** in CPU time division: no process of equal priority should be deprived of clock cycles indefinitely.

However, during a cascade failure, computational fairness is **mathematical suicide**. Guaranteeing that the memory-leaking process — the *causal* process of the crisis — receives its fair share of the processor while the *critical* database competes for the same cycles is not equity; it is active propagation of the collapse. CFS, by design, does not distinguish between a pathological process and a vital process.

`sched_ext` (Extensible Scheduler Class), introduced in kernel 6.11 as a stable feature, provides the mechanism that resolves this paradox. It allows dynamically replacing — without system restart — the scheduling algorithm with an eBPF program loaded at runtime.

**Targeted Starvation:**

When the mathematical engine reaches Level 3 response ($D_M > \theta_3$ with positive acceleration), the HOSA Survival Scheduler is activated via `sched_ext`. The scheduler eBPF program implements the following policy:

The process or cgroup identified as the anomaly causer — determined by the dimensional contribution decomposition $c_j$ combined with per-cgroup consumption delta attribution via memory and CPU tracepoints — is physically removed from the main dispatch queue (`SCX_DSQ_GLOBAL`) and inserted into a very-low-priority scheduling quarantine structure, with minimal timeslice (`SCX_SLICE_DFL / 16`). Operationally, it receives **zero clock cycles** unless *all* available cores are simultaneously in absolute idle state.

This is a fundamental technical distinction from throttling via cgroups `cpu.max` (Phase 1): cgroup throttling operates in the *bandwidth* domain — the process receives a guaranteed time fraction but in a larger window. Targeted Starvation operates in the *dispatch* domain — the process simply is not scheduled. Under high CPU utilization (precisely the situations where HOSA Phase 2 activates), this distinction determines whether the database has guaranteed processor access or competes with the invasive process at every time quantum.

The audit log records, for each targeted starvation decision: the affected process PID, cgroup, dimensional contribution $c_j$ that justified the decision, activation timestamp, and $D_M$ at the time of the decision.

**Predictive Cache Affinity (Zero Preemption for Vital Processes):**

Context switching during a crisis introduces a frequently overlooked cost: the **destruction of L1 and L2 processor cache content**. When the kernel preempts process A to execute process B on the same physical core, the cache lines that process A had loaded are progressively replaced by process B's memory footprint. When A resumes execution, it finds a cold cache — subsequent operations result in cache misses with latencies of tens to hundreds of nanoseconds per access, compared to 1–4 ns for an L1 hit.

The HOSA Survival Scheduler implements **Predictive Cache Affinity** for processes marked as vital (those on the safelist or explicitly classified as critical via configuration):

The `sched_ext` program identifies vital processes and associates them with a subset of dedicated physical cores — typically the cores with highest L3 cache locality relative to the process's critical data structures (determined during warm-up via NUMA and cache profile analysis, reading `/sys/devices/system/cpu/cpu*/cache/index*/shared_cpu_map`). The eBPF scheduler emits affinity instructions that **prohibit preemption on that core subset** by any process outside the vital process list. The critical process, once dispatched to its dedicated core, maintains the core until its timeslice expires or the process voluntarily blocks (I/O, synchronization). Kernel interrupts (softirq, hardirq) are permitted, but preemption by scheduling of other processes is blocked.

The practical effect is that the dedicated cores operate as **soft real-time cores** for vital processes during the Lethal Interval — without requiring the complexity of a full real-time system (PREEMPT_RT) and without permanent costs (the policy is automatically reverted when HOSA returns to Level 0).

**Kernel requirements for Phase 2 CPU:**
- Linux ≥ 6.11 with `CONFIG_SCHED_CLASS_EXT=y`
- `CAP_SYS_ADMIN` for loading the `sched_ext` program via `bpf(BPF_PROG_LOAD)`
- Checked during Hardware Proprioception; if not available, HOSA operates exclusively with cgroups (Phase 1) without Phase 1 degradation

##### 8.2.2. RAM Deliverables: Thermodynamics and Memory Topology Entropy

The Linux memory manager uses the **Buddy Allocator**, which organizes physical memory in power-of-two-sized blocks. With time and heavy use, physical memory **fragments**: contiguous blocks of free pages become scarce as allocations and frees of different sizes create irregular gaps in the physical address space.

When a process needs a large contiguous block and the system is fragmented, the kernel triggers **Compaction Stall**: the kernel sweeps the physical address space moving memory pages to create sufficiently large contiguous blocks, during which userspace experiences unpredictable latencies of tens to hundreds of milliseconds. This phenomenon generates **fatal latency invisible to traditional metrics** — no CPU, total memory, or disk I/O metric captures it directly. The system appears healthy to Prometheus while critical processes stall waiting for memory allocations.

Memory fragmentation is a **thermodynamic phenomenon** in the precise sense: the entropy of the allocation system increases monotonically with use, and the only way to reverse it is through active compaction work.

**Multivariable Fragmentation Entropy Calculation:**

HOSA instruments virtual memory subsystem tracepoints:
- `mm_compaction_begin` / `mm_compaction_end`: detects when the kernel starts compaction and its duration
- `mm_page_alloc_extfrag`: emitted when an allocation causes external fragmentation
- `mm_page_alloc_zone_locked`: indicates contention in the memory allocator by zone
- Periodic reading of `/proc/buddyinfo`: current distribution of free blocks by order and NUMA zone

From this data, HOSA calculates the **Fragmentation Entropy** $H_{frag}(t)$:

$$H_{frag}(t) = -\sum_{o=0}^{O_{max}} \sum_{z \in Z} p_{o,z}(t) \log_2 p_{o,z}(t)$$

where $p_{o,z}(t)$ is the normalized proportion of free blocks of order $o$ in NUMA zone $z$ at instant $t$, and $O_{max}$ is the maximum Buddy Allocator order (typically 10, corresponding to 4MB blocks).

In a system with ideally defragmented memory, high-order blocks are available and $H_{frag}$ is high (high distribution entropy = many possible combinations = healthy system). In a severely fragmented system, $H_{frag}$ converges to low values — the available allocation space has collapsed to the smallest possible granularities.

This metric is incorporated into the state vector $\vec{x}(t)$ as dimension $x_{frag}(t)$ with a negative load sign ($s_{frag} = -1$: a *drop* in $H_{frag}$ indicates *greater* stress). The Welford algorithm already implemented in Phase 1 is **reused without modification** — the generality of the matrix framework naturally absorbs the addition of new variables to the state vector.

**Preemptive Defragmentation:**

In response to $H_{frag}$ crossing a warning threshold — calibrated during warm-up as $\mu_{frag} - 2\sigma_{frag}$ — HOSA injects **micro-dosed background compaction directives** via writes to `/proc/sys/vm/compact_memory` or, with finer granularity, via `MADV_COLLAPSE` invocation on specific memory regions.

The "micro-dosing" is fundamental: instead of requesting global system compaction (which generates the Compaction Stall we want to avoid), HOSA schedules small page reorganization operations at natural low-CPU pressure intervals — the troughs between processing bursts identified by the Survival Scheduler. The Compaction Stall is avoided because the compaction work is distributed over time, never accumulating to the point where a synchronous blocking operation becomes necessary.

**Page Table Isolation:**

In acute memory leak cases — identified by $x_{mem\_used}$ dominating the $c_j$ decomposition with sustained positive derivative — HOSA alters the **allocation rules for the invasive process**, implementing the equivalent of a geographic memory isolation:

The invasive process is forced to consume pages exclusively from **more distant NUMA memory zones** (higher access latency) or from **already compressed pages** (via `zswap`) before receiving swap paging. Simultaneously, HOSA instructs the kernel to mark as `MADV_PAGEOUT` the invasive process's pages that have remained inactive for more than a configurable time window, aggressively pushing them to the swap area (swap) and freeing fast, contiguous RAM for healthy processes.

This approach contrasts with the OOM-Killer (which *destroys* the process) and with `memory.high` throttling from Phase 1 (which *pressures* the process via backpressure): Page Table Isolation *degrades* the quality of memory available to the invasive process while **preserving access quality** for critical processes.

**Kernel requirements for Phase 2 RAM:**
- Linux ≥ 5.8 (support for `vmstat` tracepoints via eBPF — already required by Phase 1)
- `CAP_SYS_ADMIN` for writing to `/proc/sys/vm/compact_memory`
- `CAP_SYS_PTRACE` for emitting `MADV_PAGEOUT` to processes of other UIDs

**Relationship with Phase 1:** Phase 2 does not replace Phase 1; it extends it. `memory.high` throttling (Phase 1) remains active and effective for most containment scenarios. Phase 2 adds a *deeper* intervention layer for cases where Phase 1's logical mitigation is insufficient — when the problem is not just *how much* memory a process uses, but *how* that usage destroys the physical topology of memory available to other processes.

**Phase 2 experimental validation:**
- Compaction Stall benchmark: controlled fragmentation injection via test allocator; comparison of stall frequency and duration with and without HOSA preemptive defragmentation
- Cache efficiency measurement: L1/L2 cache hit rate of the critical process during containment with and without Predictive Cache Affinity
- Phase 2 overhead: CPU and memory consumed by `sched_ext` programs and HOSA compaction operations vs. baseline

---

### 8.2. Long-Term Vision (Doctoral Scope and Future Research)

#### Phase 3: Ecosystem Symbiosis (v3.0)

**Scope:** Opportunistic integration with orchestrators and monitoring systems.

**Deliverables:**
- Webhooks for K8s HPA/KEDA: preemptive scale-up triggering based on $D_M$ derivative
- HOSA metrics exposure in Prometheus-compatible format
- Enriched `/healthz` endpoint: normalized state vector instead of binary healthy/unhealthy
- Digital Endocrine System: long-term "fatiguability" metrics (thermal wear, SSD write cycles) exposed as labels for the Kubernetes scheduler

#### Phase 4: Local Semantic Triage (v4.0)

**Scope:** Introduction of post-containment causal analysis.

**Deliverables:**
- Small Language Model (SLM) running locally, activated **only** after Level 3+ containment for probable root cause diagnosis
- Model operating **air-gapped** (without internet connection)
- Memory T-Cells: attack pattern signatures stored in eBPF Bloom Filter for nanosecond blocking on recurrence
- Autonomous Quarantine (Level 5) complete with all environment-class modes (per Section 5.4.1)
- Neural Habituation: automatic baseline recalibration when workload changes are classified as benign by the SLM

**Note on footprint:** The SLM is a **conditional** component, activated only on nodes with sufficient resources (minimum recommended: 4GB available RAM). On resource-constrained devices (IoT, low-capacity Edge), Phase 4 is not deployed.

#### Phase 5: Swarm Intelligence (v5.0) — *Future Research*

**Research hypothesis:** HOSA-equipped nodes can establish local consensus on cluster state via lightweight P2P communication, reducing control plane dependency for collective health decisions.

**Recognized technical challenges:** Consensus distribution is a problem with decades of research (Lamport, 1998; Ongaro & Ousterhout, 2014). The proposal is not to reinvent Paxos/Raft, but to investigate whether the limited scope of the decision (collective anomaly confirmation) permits lighter protocols. Anticipatory seasonality (allostasis) would allow pre-positioning resources before predicted peaks.

#### Phase 6: Federated Learning and Collective Immunity (v6.0) — *Future Research*

**Research hypothesis:** Mathematical weight updates (not sensitive data) shared between HOSA instances can create collective immunity against emerging attack patterns.

**Recognized technical challenges:** Federated learning convergence in heterogeneous environments (Li et al., 2020); resistance to model poisoning attacks; differential privacy (Dwork & Roth, 2014).

#### Phase 7: Hardware Offload (v7.0) — *Future Research*

**Research hypothesis:** Migration of the perceptive-motor cycle to dedicated hardware (SmartNIC/DPU) eliminates CPU competition with node applications and allows operation in low-power states.

**Recognized technical challenges:** SmartNICs and DPUs are specialized hardware with significant cost, potentially contradicting the ubiquitous hardware premise. SmartNIC programming (P4, offloaded eBPF) has computational complexity limitations.

---

#### Phase 8: The Causal Kernel — Causal Inference and do-Calculus in Ring 0 (v8.0) — *Future Research*

If Phases 1 to 7 of HOSA built the reflex arc, the sympathetic nervous system, the immune system, and the distributed neural network, Phase 8 is the moment when the operating system develops the **Prefrontal Cortex**. It is the absolute transition from a system that *reacts* to mathematical symptoms to a system that *reasons* about the laws of cause and effect before acting.

##### 8.8.1. The Ladder of Causality Applied to Silicon

The current SRE and systems monitoring live paralyzed on the first rung of Pearl's mathematics. Phase 8's goal is to force the operating system to climb to the third rung in real time.

**Rung 1: Association (Blind Statistics)**

The question: *What is happening?*

The current state of the industry operates here. Prometheus sees that CPU rose and latency rose. They are correlated. The problem with correlation is that it is **symmetric** (A correlates with B, therefore B correlates with A), but causality is **asymmetric** (A causes B, but B does not cause A). The Linux OOM Killer operates here: it kills the process using the most memory at the moment, often eliminating the victim and leaving intact the process causing the memory leak. HOSA's own Phases 1 and 2 operate on this rung — the Mahalanobis Distance quantifies the *deviation*, but does not reason about *why* the deviation occurred or *who* caused it.

**Rung 2: Intervention (HOSA Phases 1–4 Action)**

The question: *What happens if I do X?*

HOSA already operates here from Phase 1. It perceives the anomaly and actively intervenes — applies cgroup limits, isolates the network, activates the Survival Scheduler. Phase 4 (SLM) elevates sophistication with semantic post-containment diagnosis. But even with the SLM, HOSA still operates on correlations and classifications — it does not build an explicit model of the causal relationships between system processes.

**Rung 3: Counterfactuals (Mechanical Imagination)**

The question: *What if I had acted differently?*

This is the abyss that Phase 8 crosses. The system evaluates a hypothetical scenario that never physically occurred. The question the Causal Kernel formulates, in microseconds, before any action:

> *"If I throttle the CPU of Process A through the mathematical operation do(limit\_A), does the causal model predict that the database stabilizes in 300 milliseconds? Or if I redirect external network traffic before limiting A, does the order of operations produce a better result?"*

This ability to simulate the future before acting is what distinguishes a system that reacts from a system that decides.

##### 8.8.2. The Causal Kernel Architecture: Directed Acyclic Graphs in Ring 0

For Linux to be able to execute a counterfactual, it needs to build and maintain in memory a **Directed Acyclic Graph** (DAG) dynamic of the system state, representing the causal relationships between processes.

In Phase 8, eBPF programs track the **IPC causal flow** (Inter-Process Communication), building the causal topology in real time:

**Genesis (causal graph root):** The tracepoints `sched:sched_process_fork`, `sched:sched_process_exec`, and the syscall `clone()` are instrumented to record the process parentage tree.

**Communication (graph edges):** The syscalls `sendmsg()`, `recvmsg()`, pipe operations (`write()`/`read()` on pipe file descriptors), and shared memory regions (`shmget()`, `mmap()` with `MAP_SHARED`) are instrumented via kprobes. If Microservice A (PID 100) writes to a UNIX socket and Microservice B (PID 200) reads from that socket, HOSA creates a **directional edge** in the DAG: $A \rightarrow B$. The edge carries metadata: average data volume transferred, communication frequency, and the variance of these metrics (calculated via incremental Welford).

**Resource Consumption (node weights):** The tracepoints `mm:mm_page_fault_user` and allocation calls intercepted via kprobes in `__alloc_pages()` and `do_mmap()` associate physical resource consumption with each DAG node.

Formally, let $G = (V, E)$ be the causal DAG, where $V = \{P_1, P_2, \ldots, P_k\}$ is the set of active processes and $E = \{(P_i, P_j, w_{ij})\}$ is the set of directional edges with weight $w_{ij}$ representing the IPC volume and frequency from $P_i$ to $P_j$. HOSA maintains $G$ in a `BPF_MAP_TYPE_HASH` eBPF Map for nodes (indexed by PID) and `BPF_MAP_TYPE_ARRAY` for the sparse adjacency structure, updated incrementally at each detected IPC event, with an exponential decay mechanism for edges that receive no recent traffic.

##### 8.8.3. The do-Calculus Compiled into the Kernel

The heart of Phase 8 is the implementation of Pearl's **$do(\cdot)$ operator** within HOSA's decision engine.

The operator $do(X = x)$ represents an *intervention* — the surgical modification of a variable in the causal model, regardless of its natural causes. The distinction between $P(Y | X = x)$ (conditional probability — observation) and $P(Y | do(X = x))$ (interventional probability — action) is the mathematical foundation that separates correlation from causality.

**Concrete example (the Section 5.7 walkthrough scenario, elevated to Rung 3):**

- **Observed symptom:** Process B (database, PID 200) is exhausting RAM. $c_j$ indicates `mem_used` contributes 68% of $D_M^2$.
- **Causal Graph:** $\text{External Traffic} \rightarrow \text{PID 100 (API)} \rightarrow \text{PID 200 (DB)}$. Edge weight $A \rightarrow B$ shows growing IPC volume.
- **Counterfactual 1:** $do(\text{kill}_{B})$ → global service goes down. **Catastrophic failure.** Cost: ∞.
- **Counterfactual 2:** $do(\text{limit\_memory}_{B})$ via `memory.high` → pressure relieved, but A's IPC traffic still flows. **Temporary mitigation.** Cost: moderate.
- **Counterfactual 3:** $do(\text{reduce\_bandwidth}_{A})$ via XDP on PID 100 → edge $A \rightarrow B$ carries less load, relieving pressure on B at the causal root. **Causal mitigation.** Cost: low.
- **Action:** HOSA throttles Process A's network, saving Process B.

The operating system just **punished the instigator, not the executor**.

The mathematical formalization follows Pearl's **identifiability algorithm** (Pearl, 2009, Ch. 3):

$$P(Y | do(X = x)) = \sum_{z} P(Y | X = x, Z = z) \cdot P(Z = z)$$

where $Z$ are the parents of $X$ in the DAG. In HOSA's context, $Y$ is the system health state (function of $D_M$), $X$ is the candidate containment action, and $Z$ is the state of upstream processes in the DAG.

##### 8.8.4. The Engineering Abyss: Real Implementation Challenges

**The eBPF Verifier Limit:**

The Linux verifier rejects any eBPF program containing *unbounded loops*. The problem is that graph traversal — DFS or BFS algorithms — requires, by definition, an iteration whose number of steps depends on the graph size at runtime, not on a constant known at compile time.

The proposed solution is **limited causal depth with unrolled graph structure**: instead of a generic traversal loop, the eBPF program implements a fixed number $N$ of pre-compiled traversal steps, where $N$ is the maximum causal chain depth considered significant (initial proposal: $N = 8$ hops, covering most real microservice architectures). The verifier sees a program with a compile-time-defined number of instructions ($N \times \text{cost\_per\_hop}$) and accepts it.

**Space Complexity:**

HOSA proposes a **Sparse Adjacency Matrix** implemented over `BPF_MAP_TYPE_HASH` with composite key $(PID_i, PID_j)$, limited to a configurable maximum size (proposal: 10,000 active edges). For systems with many ephemeral processes, **Directional Bloom Filters** can replace edges with low $w_{ij}$, maintaining edge presence with minimal space cost.

**Distributed Causality (Integration with Phase 5):**

Within the scope of standalone Phase 8 (intra-node), the DAG covers only local processes. Long-term, if Process A is on Server 1 and Process B is on Server 2, the DAG needs to cross the physical network. The research proposal is the introduction of a **Causal Trace ID** — a causality identifier embedded in HTTP/gRPC headers as a distributed tracing extension compatible with OpenTelemetry — that allows the HOSA on Server 2 to join its local graph with the graph received from Server 1, building a distributed DAG without centralized coordination.

##### 8.8.5. Scientific Impact and State-of-the-Art Positioning

The completion of Phase 8 represents an original contribution to the intersection of two fields that rarely directly dialogue: **causal inference theory** (Pearl, 2009; Peters, Janzing & Schölkopf, 2017) and **operating systems engineering**.

Causal inference has been extensively applied in economics, epidemiology, and more recently in machine learning. Its application to operating systems — where "observation" is kernel metrics collected in microseconds and "intervention" is resource control operations in Ring 0 — is, as far as this work's literature review identified, absent from published literature.

HOSA Phase 8 is not just an engineering extension — it is a proposal for a **new reasoning primitive for operating systems**: the capability of an OS to reason about the causal consequences of its own actions before executing them. The architectural beauty of Phase 8 is that it discards nothing built in previous phases. Phase 1's data collection provides signals for the DAG. Phase 2's Survival Scheduler provides the actuator that the do-calculus commands. Phase 4's SLM can interpret the causal graph in natural language for the operator.

---

#### Phase 9: eSRE — Methodological Formalization (v9.0) — *Future Research*

**Goal:** Consolidation of HOSA principles into an open methodology called **eSRE (Endogenous Site Reliability Engineering)**, documenting the "Laws of Cellular Survival" as recommended practices for resilient system design.

**Dependency:** Adoption and empirical validation across diverse production environments, spanning Phases 1 to 8. This is a dissemination and methodological systematization objective, not an engineering component.

---

## 9. Known Limitations and Work Boundaries

1. **Distribution assumption.** The Mahalanobis Distance implicitly assumes an approximately ellipsoidal baseline profile. Workloads with multimodal distributions may violate this assumption. The dissertation will investigate detector robustness under non-Gaussian distributions.

2. **Cold start.** During warm-up (first minutes after initialization), the agent has no reliable baseline. HOSA operates in conservative mode (logging only, no mitigation), constituting a vulnerability window.

3. **Adversarial evasion.** A sophisticated attacker understanding HOSA's architecture could execute a "low-and-slow" attack keeping $D_M$ and derivatives below detection thresholds. Covariance deformation detection ($\rho$, $\Delta H$) raises the bar significantly, but the theoretical evasion possibility exists.

4. **Throttling side effects.** Throttling may introduce cascade timeouts, transaction deadlocks, or critical component starvation. The safelist and contributing-process targeting minimize this.

5. **OS scope.** HOSA is designed exclusively for Linux kernel (≥ 5.8 for Phases 1 and 2 RAM; ≥ 6.11 with `CONFIG_SCHED_CLASS_EXT=y` for Phase 2 CPU). No portability to other kernels is planned.

6. **NUMA interaction and hardware heterogeneity.** Complex NUMA topologies may exhibit localized pressure patterns that the aggregated state vector does not capture.

7. **eBPF verifier and graph traversal.** The eBPF verifier's unbounded loop restriction imposes an $N$-hop limit on causal DAG traversal (Phase 8). The choice of $N$ is a trade-off between causal coverage and verifier approval.

8. **`sched_ext` and cgroup interaction.** The interaction between the Survival Scheduler via `sched_ext` (Phase 2) and `cpu.max` limits via cgroups (Phase 1) requires careful validation to ensure the two mechanisms do not produce unexpected behaviors when simultaneously active.

---

## 10. Anticipated Questions and Answers

**Q1: "Why not use Machine Learning / Deep Learning instead of Mahalanobis Distance?"**

HOSA must operate on any hardware running Linux ≥ 5.8, including IoT devices with 512MB RAM and no GPU. Autoencoders and LSTMs require training infrastructure, runtime with significant footprint, and stored data windows. The Mahalanobis Distance with incremental Welford offers: online calibration without a separate training phase; fixed $O(n^2)$ memory footprint (< 2KB for $n \leq 15$); constant time calculation per sample (~microseconds for $n = 10$); and **interpretable** results (dimensional contribution $c_j$). The question is not "which technique is more sophisticated?" — it is "which technique detects anomalies with sub-millisecond latency, constant memory, without GPU, on a Raspberry Pi?"

**Q2: "Isn't this just a HIDS with a different name?"**

| Dimension | HIDS (e.g., OSSEC, Wazuh) | HOSA |
|---|---|---|
| **Primary focus** | Security — intrusion detection | Operational survival — node homeostasis |
| **Detection model** | Known attack signatures (model of "known bad") | Baseline profile deviation (model of "known good") |
| **Monitored variables** | Logs, file integrity, suspicious syscalls | Resource metrics and their multivariable correlations |
| **Action** | Alert. Point blocking. | Autonomous graduated mitigation: throttling, load shedding, quarantine |
| **Sub-demand detection** | No | Yes — Regimes −1, −2, −3 |
| **Network dependency** | Typically requires central server | Total autonomy for primary function |

**Q3: "Why not contribute multivariable detection to `systemd-oomd`?"**

Three structural incompatibilities: (1) `systemd-oomd` monitors only memory PSI; HOSA monitors $n$ correlated variables. (2) `systemd-oomd` has one action: kill the entire cgroup. HOSA implements 6 graduated response levels. (3) `systemd-oomd` is coupled to the systemd ecosystem; HOSA operates on any Linux environment without init system dependency.

**Q4: "Can the resilience agent itself become the cause of the problem?"**

HOSA addresses this through: dedicated cgroup v2 with strict CPU and memory limits (it practices what it preaches); safelist including itself; automatically reversible Levels 0–4; escalation hysteresis; dry-run mode; static compilation without dynamic dependencies. Phase 2 additionally: the `sched_ext` program unconditionally respects real-time policy processes (`SCHED_FIFO`/`SCHED_RR`); memory compaction operations are micro-dosed to prevent secondary Compaction Stalls.

**Q5: "What is the difference between HOSA and Meta FBAR?"**

| Dimension | FBAR | HOSA |
|---|---|---|
| **Architecture** | Centralized | Distributed/local |
| **Network dependency** | Total | None for primary function |
| **Decision latency** | Seconds to minutes | Milliseconds |
| **Action scope** | Broad (drain nodes, restart, redirect traffic) | Restricted to local node |
| **Availability** | Proprietary | Open-source, any Linux ≥ 5.8 |

FBAR and HOSA are complementary: in a datacenter with both, HOSA stabilizes the node in initial milliseconds while FBAR deliberates and executes systemic remediation.

**Q6: "The Mahalanobis Distance is a 1936 technique. Isn't it obsolete?"**

Linear algebra and differential calculus are from the 18th century. We continue using them because they are correct. HOSA extends Mahalanobis with: incremental Welford update (constant footprint); temporal derivative analysis (sensitivity to dynamics, not just state); regularization for numerical robustness; supplementary metrics for regime classification ($\rho$, $\Delta H$, $WEI$, $R_{ku}$, $PBI$). Mahalanobis is the **foundation**, not the totality of the detection system.

**Q7: "How does HOSA behave in systems with highly variable load?"**

Seasonal profiles (Section 6.6) address temporally predictable variability. Habituation (Section 5.5) addresses permanent plateau shifts. Derivative tolerance handles fast stabilizing spikes. Genuinely random workloads without temporal pattern are a recognized limitation in Section 9.

**Q8: "Does Phase 2 (sched_ext) interfere with scheduling guarantees for critical system processes?"**

The HOSA Survival Scheduler respects processes with real-time policies (`SCHED_FIFO`/`SCHED_RR`) unconditionally — these receive absolute priority. Only processes with `SCHED_NORMAL` or `SCHED_BATCH` policies are managed by Targeted Starvation. Kernel processes (kthreadd, ksoftirqd, kworkers) are on the safelist by default. Cache Affinity mode for vital processes is additive — it guarantees priority access to dedicated cores without removing processing from real-time processes.

**Q9: "Is the Phase 8 causal DAG subject to loops? Processes can have bidirectional communication."**

Yes. Bidirectional communication between processes would produce cycles in the raw communication graph, violating the DAG definition. HOSA resolves this through two strategies: (1) **temporal orientation** — the edge $A \rightarrow B$ is only created when A *initiates* the communication; in short time windows, most communications have a causally dominant direction identifiable by temporal precedence; (2) **cycle detection and treatment** — the DAG traversal algorithm maintains a set of visited nodes and aborts traversal if an already-visited node is found, treating the cycle as a feedback loop without attempting causal resolution — the dominant causal contributor is determined by edge weights ($w_{ij}$), not traversal direction.

---

## 11. Expected Contributions

1. **Formalization of the Endogenous Resilience concept** as a paradigm complementary to exogenous observability, with precise definition of the operational limits of each approach.

2. **Real-time multivariable anomaly detection model** based on Mahalanobis with incremental update and rate-of-change analysis, validated against real and synthetic collapse scenarios.

3. **Physical mitigation architecture via `sched_ext` and thermodynamic memory control**: first documented proposal for dynamic replacement of the Linux process scheduler as a node survival mechanism, combined with preemptive defragmentation based on entropy analysis of physical page topology.

4. **Reference architecture for autonomous mitigation agents with kernel-space actuation**, documenting design trade-offs (latency vs. stability, autonomy vs. mitigation risk).

5. **Quantitative comparative analysis** of detection and mitigation time between the endogenous model (HOSA) and the exogenous model (Prometheus + Alertmanager + orchestrator).

6. **Graduated response framework** for autonomous mitigation, with explicit documentation of risks and protection mechanisms (safelist, hysteresis, quarantine vs. destruction).

7. **First causal inference framework applied to real-time kernel decisions**: implementation of Pearl's $do(\cdot)$ operator over dynamic IPC DAGs maintained in eBPF Maps, with counterfactual evaluation of mitigation interventions before their execution.

---

## 12. References

Aggarwal, C. C. (2017). *Outlier Analysis* (2nd ed.). Springer.

Bear, M. F., Connors, B. W., & Paradiso, M. A. (2015). *Neuroscience: Exploring the Brain* (4th ed.). Wolters Kluwer.

Beyer, B., Jones, C., Petoff, J., & Murphy, N. R. (2016). *Site Reliability Engineering: How Google Runs Production Systems*. O'Reilly Media.

Brewer, E. A. (2000). Towards robust distributed systems. *Proceedings of the 19th Annual ACM Symposium on Principles of Distributed Computing (PODC)*.

Burns, B., Grant, B., Oppenheimer, D., Brewer, E., & Wilkes, J. (2016). Borg, Omega, and Kubernetes. *ACM Queue*, 14(1), 70–93.

Cantrill, B., Shapiro, M. W., & Leventhal, A. H. (2004). Dynamic Instrumentation of Production Systems. *Proceedings of the USENIX Annual Technical Conference (ATC)*, 15–28.

Chandola, V., Banerjee, A., & Kumar, V. (2009). Anomaly Detection: A Survey. *ACM Computing Surveys*, 41(3), Article 15.

Dwork, C., & Roth, A. (2014). The Algorithmic Foundations of Differential Privacy. *Foundations and Trends in Theoretical Computer Science*, 9(3–4), 211–407.

Engel, P. M., & Heinen, M. R. (2010). Incremental Learning of Multivariate Gaussian Mixture Models. *Proceedings of the Brazilian Symposium on Artificial Intelligence (SBIA)*.

Forrest, S., Hofmeyr, S. A., & Somayaji, A. (1997). Computer immunology. *Communications of the ACM*, 40(10), 88–96.

Gnanadesikan, R., & Kettenring, J. R. (1972). Robust Estimates, Residuals, and Outlier Detection with Multiresponse Data. *Biometrics*, 28(1), 81–124.

Gregg, B. (2019). *BPF Performance Tools: Linux System and Application Observability*. Addison-Wesley Professional.

Hellerstein, J. L., Diao, Y., Parekh, S., & Tilbury, D. M. (2004). *Feedback Control of Computing Systems*. John Wiley & Sons.

Henze, N., & Zirkler, B. (1990). A Class of Invariant Consistent Tests for Multivariate Normality. *Communications in Statistics — Theory and Methods*, 19(10), 3595–3617.

Heo, T. (2015). Control Group v2. *Linux Kernel Documentation*. https://www.kernel.org/doc/Documentation/cgroup-v2.txt

Horn, P. (2001). Autonomic Computing: IBM's Perspective on the State of Information Technology. *IBM Corporation*.

Hubert, M., Debruyne, M., & Rousseeuw, P. J. (2018). Minimum Covariance Determinant and Extensions. *WIREs Computational Statistics*, 10(3), e1421.

Isovalent. (2022). Tetragon: eBPF-based Security Observability and Runtime Enforcement. https://tetragon.io/

Lamport, L. (1998). The Part-Time Parliament. *ACM Transactions on Computer Systems*, 16(2), 133–169.

Li, T., Sahu, A. K., Talwalkar, A., & Smith, V. (2020). Federated Learning: Challenges, Methods, and Future Directions. *IEEE Signal Processing Magazine*, 37(3), 50–60.

Mahalanobis, P. C. (1936). On the generalized distance in statistics. *Proceedings of the National Institute of Sciences of India*, 2(1), 49–55.

Mardia, K. V. (1970). Measures of Multivariate Skewness and Kurtosis with Applications. *Biometrika*, 57(3), 519–530.

Ongaro, D., & Ousterhout, J. (2014). In Search of an Understandable Consensus Algorithm. *Proceedings of the USENIX Annual Technical Conference (ATC)*.

Pearl, J. (2009). *Causality: Models, Reasoning, and Inference* (2nd ed.). Cambridge University Press.

Pearl, J., & Mackenzie, D. (2018). *The Book of Why: The New Science of Cause and Effect*. Basic Books.

Penny, K. I. (1996). Appropriate Critical Values When Testing for a Single Multivariate Outlier by Using the Mahalanobis Distance. *Journal of the Royal Statistical Society: Series C*, 45(1), 73–81.

Peters, J., Janzing, D., & Schölkopf, B. (2017). *Elements of Causal Inference: Foundations and Learning Algorithms*. MIT Press.

Poettering, L. (2020). systemd-oomd: A userspace out-of-memory (OOM) killer. *systemd Documentation*. https://www.freedesktop.org/software/systemd/man/systemd-oomd.service.html

Prometheus Authors. (2012). Prometheus: Monitoring System and Time Series Database. Cloud Native Computing Foundation. https://prometheus.io/

Rousseeuw, P. J. (1984). Least Median of Squares Regression. *Journal of the American Statistical Association*, 79(388), 871–880.

Rousseeuw, P. J., & Van Driessen, K. (1999). A Fast Algorithm for the Minimum Covariance Determinant Estimator. *Technometrics*, 41(3), 212–223.

Scholz, D., Raumer, D., Emmerich, P., Kurber, A., Lessman, K., & Carle, G. (2018). Performance Implications of Packet Filtering with Linux eBPF. *Proceedings of the IEEE/IFIP Network Operations and Management Symposium (NOMS)*.

Sysdig. (2016). Falco: Cloud-Native Runtime Security. https://falco.org/

Tang, C., et al. (2020). FBAR: Facebook's Automated Remediation System. *Proceedings of the ACM Symposium on Cloud Computing (SoCC)*.

Torvalds, L., et al. (2024). sched_ext: Extensible Scheduler Class. *Linux Kernel 6.11 Release Notes and Documentation*. https://www.kernel.org/doc/html/latest/scheduler/sched-ext.html

Vieira, M. A., Castanho, M. S., Pacífico, R. D. G., Santos, E. R. S., Júnior, E. P. M. C., & Vieira, L. F. M. (2020). Fast Packet Processing with eBPF and XDP: Concepts, Code, Challenges, and Applications. *ACM Computing Surveys*, 53(1), Article 16.

Weiner, J. (2018). PSI — Pressure Stall Information. *Linux Kernel Documentation*. https://www.kernel.org/doc/html/latest/accounting/psi.html

Welford, B. P. (1962). Note on a Method for Calculating Corrected Sums of Squares and Products. *Technometrics*, 4(3), 419–420.

---

*End of Whitepaper — Version 2.2*