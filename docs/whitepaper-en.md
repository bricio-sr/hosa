# HOSA — Homeostasis Operating System Agent

## Whitepaper & Architectural Manifesto

**Author:** Fabricio Roney de Amorim
**Document Version:** 2.1 — Critical Revision
**Creation Date:** March 9, 2026
**Revision Date:** March 10, 2026
**Academic Context:** Foundational intent for Master's dissertation — Unicamp (IMECC)
**Status:** Vision and Theoretical Foundation Document

**Integrity Record:**
- Reference repository: https://github.com/bricio-sr/hosa

**Version History:**
| Version | Date | Description |
|---|---|---|
| 1.0 | 03/09/2026 | First version of the whitepaper. Initial concept. |
| 2.0 | 03/10/2026 | Critical revision: bipolar taxonomy, supplementary metrics, expanded graduated response. |
| 2.1 | 03/10/2026 | Objection hardening: robustness under non-normality, ICP calibration, environment-based quarantine, narrative walkthrough, FAQ. |

---

## Abstract

This document presents HOSA (Homeostasis Operating System Agent), a bio-inspired software architecture for autonomous resilience in Linux operating systems. HOSA proposes replacing the dominant exogenous telemetry model — in which anomaly detection and failure mitigation depend on external central servers — with a model of **Endogenous Resilience**, in which each computational node possesses autonomous capability for multivariable detection and local real-time mitigation, regardless of network connectivity.

Anomaly detection is performed through multivariable statistical analysis based on the Mahalanobis Distance and its temporal rate of change, with signal collection via eBPF (Extended Berkeley Packet Filter) in the Linux Kernel Space. Mitigation is executed through deterministic manipulation of Cgroups v2 and XDP (eXpress Data Path), implementing a graduated response system inspired by the reflex arc of the human nervous system.

HOSA does not replace orchestrators or global monitoring systems. It complements them by operating in the temporal interval where those systems are structurally incapable of acting: the milliseconds between the onset of a collapse and the arrival of the first metric at the external control plane.

**Keywords:** Endogenous Resilience, Autonomic Computing, eBPF, Multivariable Anomaly Detection, Mahalanobis Distance, Bio-Inspired Systems, Edge Computing, SRE.

---

## 1. Introduction and Problem Statement

### 1.1. The Dominant Model and Its Structural Limitations

Systems reliability engineering (Site Reliability Engineering — SRE) has consolidated over the last decade around a paradigm this work terms **Exogenous Telemetry**: a model in which local agents collect metrics, transmit them via network to central analysis servers, and await mitigation instructions derived from that remote analysis.

This paradigm, supported by widely adopted tools such as Prometheus (Prometheus Authors, 2012), Datadog, Grafana, and orchestrators such as Kubernetes (Burns et al., 2016), operates under assumptions that become progressively fragile as computational infrastructure expands into Internet of Things (IoT), Edge Computing, telecommunications, and industrial embedded systems scenarios.

The structural fragility of the exogenous model manifests in two dimensions:

**The Latency of Awareness.** The operational cycle of exogenous monitoring follows a discrete sequence: periodic collection (polling/pulling with typical intervals of 10 to 60 seconds), network transmission, storage in a time-series database (TSDB), evaluation against static thresholds (e.g., "CPU > 90% for 1 minute"), and alert dispatch. Each step introduces cumulative latency. The central system makes decisions based on a statistically stale snapshot of the remote node. In fast-collapse scenarios — denial-of-service attacks, aggressive memory leaks, instantaneous load spikes — mitigation arrives too late.

**The Connectivity Fragility.** The exogenous model assumes continuous and reliable connectivity between the monitored node and the control plane. This premise is routinely violated in Edge Computing scenarios (field devices with intermittent connectivity), during DDoS attacks that saturate the monitored node's outbound bandwidth, and in industrial infrastructures with networks segmented by security requirements. When the network fails, the node simultaneously loses its ability to report and to receive mitigation instructions, operating in complete operational blindness.

### 1.2. The Physics of Collapse: The Lethal Interval

The collapse of a computational node is not a gradual, linear process; it is an exponential cascade. When physical memory is exhausted, the Linux Kernel activates the OOM-Killer (Out-Of-Memory Killer), abruptly terminating processes based on scoring heuristics, corrupting in-flight transactions, and generating immediate unavailability. The `systemd-oomd` mechanism (Poettering, 2020) and the kernel's PSI (Pressure Stall Information) subsystem (Weiner, 2018) represent attempts by the Linux ecosystem itself to address this gap, but operate with limited scope: PSI provides pressure metrics without autonomous mitigation capability, and `systemd-oomd` acts with static policies that do not consider multivariable resource correlation.

The temporal interval between the onset of lethal stress and the arrival of the first usable metric at the external monitoring system constitutes what this work terms the **Lethal Interval** — the window where systems die without the external observer even being aware of the problem.

#### Figure 1 — Temporal Visualization of the Lethal Interval: HOSA vs. Exogenous Model

```
COLLAPSE TIMELINE — MEMORY LEAK AT 50MB/s

Time     │ Node State             │ HOSA (Endogenous)      │ Prometheus+Alertmanager
(sec)    │                        │                        │ (Exogenous)
─────────┼────────────────────────┼────────────────────────┼─────────────────────────
         │                        │                        │
  t=0    │ Leak starts            │ D_M=1.1 (homeostasis)  │ Last scrape 8s ago
         │ mem: 61%               │ Level 0                │ Data shows: "healthy"
         │                        │                        │
  t=1    │ mem: 64%               │ ⚡ D_M=2.8 DETECTS      │
         │ PSI: 18%               │ Level 0→1 (Vigilance)  │ (no scrape)
         │                        │ Sampling: 100ms→10ms   │
         │                        │                        │
  t=2    │ mem: 68%               │ ⚡ D_M=4.7 CONTAINS     │
         │ PSI: 29%               │ Level 1→2 (Containment)│ (no scrape)
         │ swap activating        │ memory.high → 1.6G     │
         │                        │ Webhook fired          │
         │                        │                        │
  t=4    │ mem: 72%               │ dD̄/dt decelerating     │ Scrape! Collects mem=1.47G
         │ (contained by HOSA)    │ Containment effective  │ Rule: >1.8G for 1m
         │                        │ Maintains Level 2      │ Result: OK (!)
         │                        │                        │
  t=8    │ mem: 74%               │ ✓ STABILIZED           │
         │ (containment plateau)  │ System degraded        │ (no scrape)
         │                        │ but functional         │
         │                        │                        │
  t=15   │ mem: 74%               │ ✓ Maintains containment│ Scrape. mem=1.52G
         │                        │                        │ Result: OK (!)
         │                        │                        │
  t=30   │ mem: 75%               │ ✓ Maintains containment│ Scrape. mem=1.55G
         │                        │ Operator received      │ Result: OK (!)
         │                        │ webhook, investigating  │
         │                        │                        │
─────────┼────────────────────────┼────────────────────────┼─────────────────────────
         │                        │                        │
         │   COUNTERFACTUAL       │   WITH HOSA            │   WITHOUT HOSA
         │   SCENARIO             │                        │
         │   (without HOSA)       │                        │
─────────┼────────────────────────┼────────────────────────┼─────────────────────────
         │                        │                        │
  t=40   │ ☠ OOM-Kill             │ ✓ System contained     │ Scrape. Detects restart.
         │ payment-service dead   │ Transactions preserved │ Still no alert
         │ Transactions corrupted │                        │ (for 1m not satisfied)
         │                        │                        │
  t=80   │ ☠ 2nd crash            │ ✓ System contained     │ Scrape. CrashLoopBackOff
         │ CrashLoopBackOff       │ Operator performing    │ detected.
         │                        │ rollback               │
         │                        │                        │
  t=100  │ ☠ Clients with 502     │ ✓ Rollback complete    │ ⚠ ALERT FIRED
         │ since t=40             │ System recovered       │ (60s after 1st crash)
         │                        │                        │
─────────┴────────────────────────┴────────────────────────┴─────────────────────────

         ├──── 2s ────┤
         │HOSA acted  │
         │   here     │
         │            │
         │            ├─────────────────────────── 98s ───────────────────────────┤
         │            │           Prometheus acted here                           │
         │            │           (100x slower)                                   │
```

### 1.3. The Central Thesis

The thesis underlying HOSA can be stated as:

> *Orchestrators and centralized monitoring systems are essential instruments for capacity planning, load balancing, and long-term infrastructure governance. However, they are structurally — not accidentally — too slow to guarantee the survival of a node in real time. If collapse occurs in the interval between exogenous perception and action, the immediate decision-making capability must reside in the node itself.*

HOSA does not propose the elimination of central monitoring. It proposes **complementing** that monitoring with a layer of local intelligence that operates autonomously during the Lethal Interval, stabilizing the node until the global system can take control of the situation.

---

## 2. Conceptual Genesis: The Biological Metaphor as a Design Tool

### 2.1. The Reflex Arc as an Architectural Pattern

The HOSA architecture was conceived from the observation of a fundamental biological pattern: the **spinal reflex arc**.

When a human organism touches a surface at a harmful temperature, the nociceptive signal does not travel the full path to the cerebral cortex (the "central orchestrator") for contextual processing and conscious deliberation. The latency of that long pathway — hundreds of milliseconds — would result in tissue injury. Instead, the signal travels a short arc to the spinal cord, which executes a reflex muscle contraction in sub-milliseconds, withdrawing the limb from the source of damage. Only after the reflex is executed is the cortex notified for contextual processing and memory formation (Bear, Connors & Paradiso, 2015).

This pattern — **immediate local action followed by contextual notification to the command center** — is precisely the operational model of HOSA.

It is important to delimit the scope of this metaphor: it is used as a **heuristic tool for architectural design**, not as a claim of functional equivalence between biological and computational systems. Biology informs the decision structure (where to process, where to act, when to escalate), but the implementation is purely mathematical and systems-engineering.

### 2.2. Precedents in the Literature: Autonomic Computing and Computational Immunology

The aspiration for self-regulating computational systems is not novel. IBM's Autonomic Computing manifesto (Horn, 2001) articulated four desirable properties — self-configuration, self-optimization, self-healing, and self-protection — but remained predominantly at the level of strategic vision, without providing the low-level instrumentation to achieve them with sub-millisecond latency.

The work of Forrest, Hofmeyr & Somayaji (1997) on computational immunology established the theoretical foundations of the "self" versus "non-self" distinction in computational systems, proposing that anomalous processes can be identified by deviations in system call (syscall) sequences. HOSA absorbs this principle into its behavioral screening layer.

What distinguishes HOSA from these predecessors is the **operational synthesis**: the combination of continuous multivariable detection (not signature-based) with kernel-space actuation via contemporary mechanisms (eBPF, Cgroups v2, XDP) that did not exist when those works were published. HOSA is, in this sense, the contemporary engineering response to a need that the literature identified two decades ago.

---

## 3. Related Work and Positioning

A responsible academic contribution requires explicit confrontation with the works and technologies operating in the same problem space. This section maps the existing ecosystem and articulates the specific gap that HOSA fills.

### 3.1. Native Linux Kernel Mechanisms

| Mechanism | Function | Limitation HOSA Addresses |
|---|---|---|
| **PSI (Pressure Stall Information)** — Weiner, 2018 | Exposes CPU, memory, and I/O pressure metrics as percentage of stall time. | PSI is a **passive sensor**: it quantifies pressure but does not execute mitigation. Additionally, PSI is a one-dimensional metric per resource — it does not simultaneously correlate CPU, memory, I/O, and network. HOSA uses PSI as one of the inputs to its multivariable state vector, but complements it with cross-covariance analysis and rate of change. |
| **systemd-oomd** — Poettering, 2020 | Daemon that monitors memory PSI and kills entire cgroups when pressure exceeds a threshold. | Operates with **static one-dimensional thresholds** (memory pressure only). Does not consider correlation with other resources. Does not offer graduated responses — the action is binary: nothing or kill. |
| **OOM-Killer** | Kernel mechanism of last resort to free memory. | **Reactive and destructive**: activates only after total memory exhaustion, and uses simplified heuristics (oom_score) that frequently eliminate critical processes. |
| **cgroups v2** — Heo, 2015 | Resource control interface per group of processes. | Is an **actuator mechanism** without associated decision intelligence. Requires something external to decide which limits to apply and when. HOSA uses cgroups v2 as its motor system. |

### 3.2. Observability Ecosystem Tools

| Tool/Project | Function | HOSA Differentiation |
|---|---|---|
| **Prometheus + Alertmanager** | Metric collection via pull, TSDB storage, rule-based alerts. | Classic exogenous model. Default scrape interval: 15–60s. Minimum alert latency: typically >1 minute. No actuation capability. |
| **Sysdig Falco** — Sysdig, 2016 | Runtime anomalous behavior detection via eBPF, focused on security. | Falco detects security policy violations (suspicious syscalls), but **does not monitor resource health** (CPU, memory, I/O) and **does not execute autonomous mitigation**. Its focus is alerting, not acting. |
| **Cilium Tetragon** — Isovalent, 2022 | Security policy enforcement in kernel space via eBPF. | Tetragon allows defining blocking policies (e.g., "block process that opens /etc/shadow"), but operates on **static operator-defined rules**. It has no statistical anomaly model, does not calculate state derivatives, and does not implement graduated responses based on severity. |
| **Pixie (px.dev)** — New Relic | Continuous observability via eBPF without code instrumentation. | Pixie is a **collection and visualization** system — it has no autonomous actuation layer. |
| **Facebook FBAR** — Tang et al., 2020 | Automated remediation at scale in Meta's datacenters. | FBAR operates as a **centralized remediation system** with network dependency and proprietary infrastructure. It is not a local autonomous agent. |

### 3.3. The Identified Gap

No existing tool in the ecosystem combines, in a single local agent:

1. **Continuous multivariable detection** (correlation between CPU, memory, I/O, network, and disk latency in a unified statistical space);
2. **Rate-of-change analysis** (temporal derivative of the state vector, detecting acceleration toward collapse rather than just current state);
3. **Autonomous graduated local actuation** (from selective throttling to network isolation, without network dependency or human intervention);
4. **Total independence from external infrastructure** for its primary survival function.

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

### 4.2. The Mahalanobis Distance as an Anomaly Detector

Static one-dimensional threshold-based anomaly detection (e.g., "CPU > 90%") suffers from a fundamental limitation: it ignores the **correlation structure** between variables. High CPU with low I/O and stable network may represent legitimate intensive processing. High CPU with growing memory pressure, I/O stall, and rising network latency represents imminent collapse. The static threshold does not distinguish these scenarios.

The Mahalanobis Distance (Mahalanobis, 1936) addresses this limitation by measuring the distance of an observation $\vec{x}$ relative to the multivariable distribution defined by the mean vector $\vec{\mu}$ and the Covariance Matrix $\Sigma$:

$$D_M(\vec{x}) = \sqrt{(\vec{x} - \vec{\mu})^T \Sigma^{-1} (\vec{x} - \vec{\mu})}$$

The Covariance Matrix $\Sigma$ captures the correlations between all variables. Its inverse $\Sigma^{-1}$ weights the dimensions according to their variance and interdependence. A vector $\vec{x}(t)$ that departs from the baseline profile in correlated dimensions in an unusual manner produces a high $D_M$, even if no individual variable has exceeded an absolute threshold.

For a comprehensive review of outlier detection methods, see Aggarwal (2017).

### 4.3. The Temporal Derivative and the Problem of Numerical Stability

HOSA does not act on the instantaneous value of $D_M$, but on its **temporal rate of change** — the speed and acceleration with which the system departs from homeostasis.

The first derivative $\frac{dD_M}{dt}$ indicates the speed of departure. The second derivative $\frac{d^2D_M}{dt^2}$ indicates acceleration — whether the system is accelerating toward collapse or decelerating.

**Recognized problem: instability of numerical differentiation in discrete, noisy data.** Numerical differentiation is an ill-posed problem in the Hadamard sense: small perturbations in input data produce large variations in the calculated derivative. The second derivative amplifies this effect quadratically. Without treatment, the second derivative of noisy kernel time series oscillates violently, generating false positives.

**Adopted solution:** HOSA implements an **Exponentially Weighted Moving Average (EWMA)** with a decay factor $\alpha$ calibrated per resource before derivative computation:

$$\bar{D}_M(t) = \alpha \cdot D_M(t) + (1 - \alpha) \cdot \bar{D}_M(t-1)$$

The factor $\alpha$ controls the fundamental trade-off between **responsiveness** (high $\alpha$ values preserve rapid variations but retain noise) and **stability** (low $\alpha$ values smooth the signal but introduce detection latency).

Calibration of $\alpha$ is performed during the agent's **warm-up** phase (Section 5.2), and constitutes one of the architecture's critical parameters. The technical documentation (separate document) will present the sensitivity analysis of $\alpha$ against synthetic and real collapse datasets, quantifying the latency vs. false positive rate trade-off.

**Alternative under investigation:** The one-dimensional Kalman Filter provides optimal state estimation in the presence of Gaussian noise, with the advantage of dynamically adapting to observed variance. The comparative analysis EWMA vs. Kalman will be presented in the dissertation's experimental phase.

### 4.4. Incremental Update of the Covariance Matrix

Batch calculation of the Covariance Matrix ($\Sigma$) over accumulated data windows is computationally expensive ($O(n^2 \cdot k)$ for $n$ variables and $k$ samples) and introduces memory allocation proportional to window size.

HOSA uses the **generalized Welford algorithm** (Welford, 1962) for incremental online updates of $\Sigma$ and $\vec{\mu}$. Each new sample $\vec{x}(t)$ updates $\Sigma$ in $O(n^2)$ with constant allocation ($O(1)$), regardless of the number of accumulated samples. This eliminates the need to store data windows and ensures predictable memory footprint.

### 4.5. Inversion of the Covariance Matrix

The Mahalanobis Distance requires $\Sigma^{-1}$. For moderate dimensionality ($n \leq 10$), direct inversion via Cholesky decomposition is computationally feasible and numerically stable (the Covariance Matrix is positive semi-definite by construction). For higher dimensionality, HOSA can resort to incremental inverse update via the Sherman-Morrison-Woodbury formula, avoiding recalculation of the full inversion for each sample.

**Degeneracy:** In systems with highly collinear variables (e.g., `cpu_user` and `cpu_total`), $\Sigma$ may become singular or ill-conditioned. HOSA applies **Tikhonov regularization** ($\Sigma_{reg} = \Sigma + \lambda I$, with small $\lambda$) to ensure invertibility.

### 4.6. Robustness of the Mahalanobis Distance under Normality Violations

The Mahalanobis Distance, as formulated in Section 4.2, implicitly assumes that the system's baseline profile follows an approximately ellipsoidal (multivariate normal) distribution. This assumption merits explicit analysis, as kernel metrics in real systems frequently exhibit characteristics that violate it.

#### 4.6.1. Nature of Expected Violations

Three classes of violation are empirically prevalent in operating system metrics:

| Violation Class | Example in Kernel Metrics | Impact on $D_M$ |
|---|---|---|
| **Heavy-tailed** | Disk I/O latency: most operations complete in microseconds, but outliers of hundreds of milliseconds occur more frequently than predicted by the normal distribution. | $D_M$ underestimates the frequency of legitimate extreme values, potentially generating false positives in tail events. |
| **Skewness** | CPU utilization: distribution often concentrated near 0% (idle system) or near 100% (saturated system), with skewness depending on the operational regime. | $\vec{\mu}$ and $\Sigma$ may not adequately represent the center and dispersion of the actual distribution, displacing the detector. |
| **Multimodality** | Systems that alternate between two distinct operational regimes (e.g., batch server processing jobs every hour — alternation between total idleness and full load). | $\vec{\mu}$ calculated as arithmetic mean is located **between** the two modes, where few real samples exist. $D_M$ classifies the normal behavior of both modes as anomalous. |

#### 4.6.2. Evidence of Robustness in the Literature

The robustness of the Mahalanobis Distance as an outlier detector under moderate normality violations is documented in the literature:

- **Gnanadesikan & Kettenring (1972)** demonstrated that covariance-based estimators maintain discriminative capability under non-normal elliptical distributions (e.g., multivariate $t$ distributions), losing the exact probabilistic interpretation (the relationship with the $\chi^2$ distribution does not hold) but preserving the **relative ordering** of anomalies — more anomalous observations continue to produce higher $D_M$ values.

- **Penny (1996)** analyzed the performance of $D_M$ as a classification criterion under various non-Gaussian distributions, confirming graceful degradation: the error rate increases under severe violations, but the detector does not collapse abruptly.

- **Hubert, Debruyne & Rousseeuw (2018)** demonstrated that the combination of $D_M$ with robust estimators of location and dispersion preserves detection efficacy even under contamination of up to 25% of samples by outliers.

HOSA operates primarily on the **rate of change** of $D_M$ (derivatives), not on its absolute value. Even if the absolute value of $D_M$ loses its exact probabilistic interpretation under non-normality, the derivatives $\frac{dD_M}{dt}$ and $\frac{d^2D_M}{dt^2}$ remain valid indicators of **acceleration toward collapse**, since they reflect the temporal dynamics of the deviation, not its probabilistic magnitude.

#### 4.6.3. Mitigation Strategy: Robust Estimation

To address severe violations when detected, HOSA implements a two-level strategy:

**Level 1 — Regularization (default).** The Tikhonov regularization already applied ($\Sigma_{reg} = \Sigma + \lambda I$) partially mitigates sensitivity to outliers by stabilizing the inversion of the covariance matrix, functioning as a form of shrinkage that approximates $\Sigma^{-1}$ to the identity.

**Level 2 — Robust estimation (conditional activation).** When HOSA detects that the baseline distribution severely violates normality — operationalized via continuous monitoring of Mardia's multivariate kurtosis (Mardia, 1970):

$$\kappa_M = \frac{1}{N} \sum_{i=1}^{N} \left[(\vec{x}_i - \vec{\mu})^T \Sigma^{-1} (\vec{x}_i - \vec{\mu})\right]^2$$

compared with the expected value under normality $\kappa_{expected} = n(n+2)$ (where $n$ is the dimensionality) — the agent can replace the estimators of $\vec{\mu}$ and $\Sigma$ with the **Minimum Covariance Determinant (MCD)** (Rousseeuw, 1984).

The MCD estimates location and dispersion using the subset of $h$ observations (out of $N$ total, with $h \approx \lceil N/2 \rceil$) whose covariance matrix has the smallest determinant, effectively discarding the influence of the $N-h$ most extreme outliers in estimating the baseline parameters. The incremental implementation of MCD via the FAST-MCD algorithm (Rousseeuw & Van Driessen, 1999) is computationally feasible for the expected dimensionality of the state vector ($n \leq 15$).

**Impact on footprint:** Incremental MCD requires storing a window of recent samples (typically 100–500 samples) to recalculate the optimal subset, partially violating the $O(1)$ memory principle of Welford. The trade-off is explicit: statistical robustness against predictable footprint. MCD activation is **conditional** — it only occurs when $\kappa_M$ diverges significantly from $\kappa_{expected}$, indicating that the observed distribution requires robust treatment. In practice, a 500-sample window with $n = 10$ variables occupies ~40KB of memory — negligible in any operational context.

#### 4.6.4. Multimodality and Interaction with Seasonal Profiles

The multimodality problem — the most severe violation for $D_M$ — is partially addressed by the mechanism of baseline profiles indexed by temporal context (Section 6.6). When multimodality is caused by predictable temporal alternation between regimes (e.g., day/night, weekday/weekend), each temporal segment accumulates its own **unimodal** baseline profile, eliminating multimodality at the root.

When multimodality is not temporally segregable (e.g., system that alternates between modes randomly), the approach requires extension to **Mixture of Gaussians** with Expectation-Maximization (EM) estimation adapted for streaming (Engel & Heinen, 2010). This extension is documented as a future research direction, as it introduces computational complexity ($O(k \cdot n^2)$ per sample, where $k$ is the number of modes) and the model selection problem (determination of $k$).

#### 4.6.5. Empirical Validation Plan

Experimental validation (documented separately in the experimental plan) will include:

1. **Real data collection** of kernel metrics in production systems (minimum 72 continuous hours per scenario);
2. **Multivariate normality tests**: Mardia's kurtosis, Henze-Zirkler test (Henze & Zirkler, 1990), and visual inspection via multivariate QQ-plots;
3. **Comparative benchmarking** of detection rate (True Positive Rate) and false positive rate (False Positive Rate) under:
   - Classical estimation (Welford, sample $\vec{\mu}$ and $\Sigma$);
   - Robust estimation (MCD);
   - Mahalanobis with prior transformation (e.g., multivariate Box-Cox for skewness reduction);
4. **Computational footprint impact analysis** of each alternative.

---

## 5. Engineering Architecture

### 5.1. Architectural Principles

HOSA's design is governed by five non-negotiable principles:

| # | Principle | Description |
|---|---|---|
| 1 | **Local Autonomy** | HOSA must execute its complete detection and mitigation cycle without dependency on network, external APIs, or human intervention for its primary function. |
| 2 | **Zero External Runtime Dependencies** | The agent does not depend on external services (TSDB, message brokers, cloud APIs) to operate. All dependencies are internal to the binary or to the host operating system kernel. Communication with external systems (orchestrators, dashboards) is **opportunistic**: performed when available, but never required. |
| 3 | **Predictable Computational Footprint** | HOSA's CPU and memory consumption must be constant and predictable ($O(1)$ in memory, configurable and limited CPU percentage). The agent cannot become the cause of the problem it intends to solve. |
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
│  │ (tracepoints │  │ (kprobes,    │  │    controllers)   │  │
│  │  scheduler,  │  │  PSI hooks)  │  │                   │  │
│  │  mm, net)    │  │              │  │                   │  │
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
│  │              MATHEMATICAL ENGINE (Go/Rust)            │  │
│  │                                                       │  │
│  │  1. Receives events from ring buffer                  │  │
│  │  2. Updates vector x(t)                               │  │
│  │  3. Updates μ and Σ incrementally (Welford)           │  │
│  │  4. Calculates D_M(x(t))                              │  │
│  │  5. Applies EWMA → D̄_M(t)                             │  │
│  │  6. Calculates dD̄_M/dt and d²D̄_M/dt²                  │  │
│  │  7. Evaluates against adaptive thresholds             │  │
│  │  8. Determines response level (0-5)                   │  │
│  │  9. Sends actuation command via BPF maps              │  │
│  │                                                       │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │          OPPORTUNISTIC COMMUNICATION (Go)             │  │
│  │                                                       │  │
│  │  - Webhooks to orchestrators (when available)         │  │
│  │  - Metrics exposure (local endpoint)                  │  │
│  │  - Structured local log (audit)                       │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**Architectural note on kernel↔user space transition.** HOSA's execution model involves transition between kernel space (eBPF collection and actuation) and user space (mathematical computation). This transition uses the eBPF ring buffer and BPF maps mechanism, with typical latency on the order of **microseconds** (1–10μs on modern hardware). The correct terminology is **"zero external runtime dependencies"**: HOSA does not depend on processes, services, or external infrastructure beyond the agent binary and the host kernel. The kernel↔user transition is internal to the agent.

### 5.3. Warm-Up Phase and Proprioceptive Calibration

Upon startup, HOSA executes a calibration phase called **Hardware Proprioception**:

1. **Topology discovery:** Via reading of `/sys/devices/system/node/` and `/sys/devices/system/cpu/`, the agent identifies NUMA topology, number of physical and logical cores, L1/L2/L3 cache sizes, and memory configuration.

2. **State vector definition:** Based on the topology, HOSA determines which variables to include in the vector $\vec{x}(t)$ and their respective eBPF sources.

3. **Baseline accumulation:** During a configurable period (default: 5 minutes), the agent collects samples without executing mitigation, accumulating initial $\vec{\mu}_0$ and $\Sigma_0$ via incremental Welford. This is the node's **baseline profile**.

4. **$\alpha$ calibration (EWMA):** The smoothing factor is calibrated for each resource based on the variance observed during warm-up.

5. **Adaptive threshold definition:** The $D_M$ thresholds for each response level are calculated as multiples of the standard deviation observed in the baseline regime (e.g., Level 1 = 2σ, Level 3 = 4σ).

After warm-up, $\vec{\mu}$ and $\Sigma$ continue to be updated incrementally, allowing the baseline profile to evolve with legitimate workload changes (Section 5.5, Habituation).

### 5.4. Graduated Response System

HOSA implements **six response levels** (0–5), each with specific actions proportional to the anomaly's severity:

| Level | Activation Condition | Action | Reversibility |
|---|---|---|---|
| **0 — Homeostasis** | $D_M < \theta_1$ and $\frac{dD_M}{dt} \leq 0$ | None. Suppresses redundant telemetry (sends minimal heartbeat). | N/A |
| **1 — Vigilance** | $D_M > \theta_1$ or $\frac{dD_M}{dt} > 0$ sustained | Local logging. Increased sampling frequency. No intervention. | Automatic (return to L0 when condition ceases). |
| **2 — Light Containment** | $D_M > \theta_2$ and $\frac{dD_M}{dt} > 0$ | Renice of non-essential processes via cgroups. Notification via webhook (opportunistic). | Automatic (gradual relaxation of renice). |
| **3 — Active Containment** | $D_M > \theta_3$ and $\frac{d^2D_M}{dt^2} > 0$ (positive acceleration) | CPU/memory throttling in cgroups of identified contributing processes. Partial load shedding via XDP (dropping new connection packets, preserving existing ones). Urgent webhook to orchestrator. | Automatic with hysteresis (relaxation when $D_M < \theta_2$ for sustained period). |
| **4 — Severe Containment** | $D_M > \theta_4$ or convergence velocity indicates exhaustion in < T seconds | Aggressive throttling. XDP blocks all inbound traffic except orchestrator healthcheck. Freeze of non-critical cgroups. | Requires sustained reduction of $D_M$ below $\theta_3$ for extended period. |
| **5 — Autonomous Quarantine** | Containment failure at previous levels. $D_M$ in uncontrolled ascent despite active mitigations. | **Network isolation**: programmatic deactivation of network interfaces (except management/IPMI interface, if present). Non-essential processes frozen (SIGSTOP). Detailed log written to persistent storage. Node signals "quarantined" state in final possible webhook. | **Manual**: requires administrative intervention to restore the node. |

### 5.4.1. Quarantine Modes by Environment Class

Autonomous quarantine (Level 5) involves network isolation of the compromised node. The feasibility and strategy of that isolation varies fundamentally depending on the infrastructure class. HOSA implements **differentiated quarantine modes**, selected automatically during the Hardware Proprioception phase (Section 5.3) or explicitly configured by the operator.

| Environment Class | Automatic Detection | Quarantine Strategy | Recovery Mechanism |
|---|---|---|---|
| **Bare metal with IPMI/iLO/iDRAC** | Detection of IPMI interface via `/sys/class/net/` and presence of `ipmi_*` modules in kernel. | Programmatic deactivation of **all** network interfaces **except** the out-of-band management interface (IPMI/iLO/iDRAC). Node remains accessible via management console for diagnosis and restoration. | Manual via IPMI console. Operator inspects HOSA logs, diagnoses root cause, restores interfaces and restarts services. |
| **VM in public cloud (AWS, GCP, Azure)** | Detection via DMI/SMBIOS (`dmidecode`), presence of metadata service (169.254.169.254), and hypervisor identification via `/sys/hypervisor/` or CPUID. | **Does not deactivate network interfaces.** Instead: (1) XDP applies total drop on all inbound and outbound traffic **except**: traffic to the cloud provider's metadata service (169.254.169.254), DHCP traffic (IP lease maintenance), and traffic to the orchestrator API endpoint (if configured). (2) HOSA signals quarantine state via **native cloud provider mechanism** when available: writing a tag/label on the instance via metadata service (e.g., `hosa-quarantine=true`), publishing to an SNS/Pub-Sub topic (if pre-configured credentials), or updating the healthcheck endpoint to return HTTP 503 with a JSON body detailing the state. (3) The external orchestrator (Kubernetes, ASG, etc.) is responsible for the terminate/replace decision. | External orchestrator terminates the instance and provisions a replacement. If the orchestrator does not act within a configurable time (default: 5 minutes), HOSA can execute **self-termination** via the cloud provider API (when IAM credentials with `ec2:TerminateInstances` or equivalent permission are available). Self-termination is **disabled by default** and requires explicit activation in configuration. |
| **Kubernetes (pod/container)** | Detection of running in container via presence of `/proc/1/cgroup` with cgroup namespace, environment variables `KUBERNETES_SERVICE_HOST`, or presence of service account mounted in `/var/run/secrets/kubernetes.io/`. | HOSA operating as a DaemonSet **does not isolate the node** (does not have permission to deactivate host interfaces). Instead: (1) Applies maximum containment via cgroups on identified contributing pods. (2) Updates Node status via Kubernetes API with **taint** `hosa.io/quarantine=true:NoExecute` and **condition** `HOSAQuarantine=True`, causing automatic pod evacuation by the scheduler. (3) Issues Event in the affected pod's namespace with type `Warning` and reason `HOSAQuarantine`. | Operator or automation removes the taint after investigation. The node returns to the scheduling pool. |
| **Edge/IoT with physical access** | Explicit configuration by operator (flag `environment: edge-physical`). | Complete deactivation of network interfaces. Device operates in isolated mode until physical intervention. Logs are preserved in persistent local storage (flash/eMMC). If the device has status LED or display, HOSA signals quarantine state visually. | Manual. Field technician accesses device, collects logs, diagnoses, and restores. |
| **Edge/IoT without physical access** | Explicit configuration by operator (flag `environment: edge-remote`). | **Quarantine with watchdog timer.** (1) Deactivation of network interfaces. (2) Activation of hardware watchdog timer (via `/dev/watchdog`) with configurable timeout (default: 30 minutes). (3) If no remote intervention occurs before the timeout (impossible, since network is deactivated), the watchdog reboots the device, which returns to the pre-quarantine state with persistent flag `quarantine_recovery=true`. (4) On restarting with this flag, HOSA enters conservative mode (logging only for a configurable period) to allow remote diagnosis before resuming autonomous mitigation. | Automatic via watchdog reboot, with post-recovery observation period. Remote operator can access during the observation period for diagnosis. |
| **Air-gapped environment (classified networks, SCADA/ICS)** | Explicit configuration by operator (flag `environment: airgap`). | Identical to bare metal, with the addition that **all opportunistic communication is permanently deactivated** (no webhooks, no endpoint exposure). HOSA operates in purely endogenous mode. Logs are written exclusively to encrypted local storage and collected periodically by authorized staff with physical access. | Manual via authorized physical access with operator-defined security procedure. |

**Design principle: automatic detection with manual override.** HOSA attempts to automatically detect the environment class and select the appropriate quarantine mode. The operator can override this detection via explicit configuration. In case of ambiguity (e.g., VM in private cloud that does not respond to the standard metadata service), HOSA assumes the **most conservative** mode (public cloud — does not deactivate interfaces), prioritizing recoverability over isolation.

**Note on containers and privilege.** When HOSA operates as a container (DaemonSet in Kubernetes), its access to host cgroups and network interfaces depends on specific Linux capabilities (`CAP_SYS_ADMIN`, `CAP_NET_ADMIN`, `CAP_BPF`). The deployment documentation will specify the minimum set of capabilities required for each response level, following the principle of least privilege. Levels 0-2 require only `CAP_BPF` and read access to `/sys/`. Levels 3-4 additionally require `CAP_SYS_ADMIN` for cgroup manipulation. Level 5 requires `CAP_NET_ADMIN` for XDP manipulation and, in Kubernetes mode, cluster API access for taint application.

### 5.5. Habituation: Adaptation to the New Baseline

A recurring problem in anomaly detection systems is **chronic false positives**: when the legitimate workload permanently changes (e.g., deployment of a new version that consumes more memory), the detector continues flagging anomalies indefinitely.

HOSA implements a **habituation** mechanism inspired by neuroplasticity:

1. If $D_M$ remains stably elevated (derivative near zero) for a configurable period without any real failure occurring (no OOM, no timeout, no process crash);
2. HOSA recalculates $\vec{\mu}$ and $\Sigma$ with increasing weight on recent samples, effectively shifting the baseline profile to the new operational regime.

This mechanism is implemented via **exponential decay of weights** in the Welford algorithm, assigning less influence to old samples and allowing $\Sigma$ to reflect the contemporary covariance of the system.

### 5.6. Selectivity Policy: The Throttling Problem

Throttling of processes via cgroups, while an effective mitigation against resource exhaustion, introduces secondary risks that must be explicitly addressed:

- **Cascade timeouts:** A throttled HTTP backend may cause upstream connection buildup, propagating the degradation.
- **Transaction deadlocks:** A throttled process during a database transaction may hold locks for an indeterminate time.
- **Critical component starvation:** If Kubernetes' kubelet is throttled, the node is marked `NotReady` and all pods are evacuated, potentially causing more damage than the original problem.

HOSA addresses these risks through a **protection list** (safelist) of processes and cgroups that are never targeted for throttling:

- Kernel processes (kthreadd, ksoftirqd, etc.)
- The HOSA agent itself
- Orchestration agents (kubelet, containerd, dockerd) when detected
- Processes explicitly marked by the operator via configuration or cgroup label

Throttling is preferentially applied to processes identified as **largest contributors** to the anomaly, determined by decomposition of the vector $\vec{x}(t)$ — the processes whose resource consumption most contributes to the dimensions where $D_M$ diverges from the baseline.

### 5.7. Scenario Walkthrough: Memory Leak in Payment Microservice

This section presents an end-to-end scenario illustrating HOSA's perceptive-motor cycle in operation, contrasting it with the behavior of an exogenous monitoring system operating simultaneously. Numerical values are representative and based on behavior observed in production systems; formal experimental validation is documented separately.

#### Context

- **Node:** VM `worker-node-07` in a Kubernetes cluster, 8 vCPUs, 16GB RAM.
- **Workload:** 12 pods, including `payment-service-7b4f` (Java payment processing microservice, 2GB of memory allocated via cgroup).
- **Exogenous monitoring:** Prometheus with 15-second scrape interval, Alertmanager with rule: `container_memory_usage_bytes > 1.8GB for 1m`.
- **HOSA:** Operating in homeostasis regime (Level 0) for 6 hours. Baseline profile calibrated. State vector with 8 dimensions.

#### Timeline

**t = 0s (14:23:07.000) — Memory Leak Start**

The `payment-service-7b4f` starts allocating objects not collected by Java's GC (circular reference in session cache). Leak rate: ~50MB/s.

System state at this instant:
```
Vector x(t):
  cpu_total:     47%    (baseline: 45% ± 8%)
  mem_used:      61%    (baseline: 58% ± 5%)
  mem_pressure:  12%    (baseline: 10% ± 4%)  [PSI some avg10]
  io_throughput: 340 IOPS (baseline: 320 ± 60 IOPS)
  io_latency:    2.1ms  (baseline: 1.9 ± 0.8ms)
  net_rx:        1,200 req/s (baseline: 1,150 ± 200 req/s)
  net_tx:        1,180 resp/s (baseline: 1,130 ± 190 resp/s)
  runqueue:      3.2    (baseline: 2.8 ± 1.5)

D_M = 1.1    (θ₁ = 3.0, θ₂ = 5.0, θ₃ = 7.0, θ₄ = 9.0)
φ = +0.3
dD̄_M/dt ≈ 0
Response Level: 0 (Homeostasis)
```

Prometheus collected last metric 8 seconds ago. Next scrape in 7 seconds.

**t = 1s (14:23:08.000) — HOSA detects initial deviation**

```
  mem_used:      64%    (+3pp in 1s — 2σ above expected for Δt=1s)
  mem_pressure:  18%    (+6pp — rapid transition)

D_M = 2.8
φ = +0.9
dD̄_M/dt = +1.6/s   (positive — accelerating departure)
d²D̄_M/dt² = +1.6/s² (positive acceleration — not decelerating)

Response Level: 0→1 (Vigilance)
```

**HOSA actions:**
- Sampling frequency increased from 100ms to 10ms.
- Local log: `[VIGILANCE] D_M=2.8 dDM/dt=+1.6 dominant_dim=mem_used(c_j=0.72) mem_pressure(c_j=0.21)`
- No system intervention. No webhook (not urgent).

**Prometheus:** No metric collected in this interval. Unaware of the event.

**t = 2s (14:23:09.000) — Escalation to Light Containment**

```
  mem_used:      68%    (+7pp accumulated)
  mem_pressure:  29%    (PSI rising rapidly)
  cpu_total:     52%    (Java GC activated — CPU rises due to memory pressure)
  io_latency:    3.8ms  (swap starting to be used — I/O latency rises)

D_M = 4.7
φ = +1.8
dD̄_M/dt = +2.1/s   (accelerating)
d²D̄_M/dt² = +0.5/s² (sustained positive acceleration)

ρ(t) = 0.31  (CPU↔memory correlation altered — CPU rising due to GC,
              not legitimate load. mem↔io_latency correlation emerging
              where it did not exist before → swap active)

Response Level: 1→2 (Light Containment)
```

**HOSA actions:**
- Dimensional decomposition: `mem_used` contributes 68% of $D_M^2$, `mem_pressure` contributes 19%, `io_latency` contributes 8%.
- Contributing cgroup identification: `/sys/fs/cgroup/kubepods/pod-payment-service-7b4f/` is the cgroup with the largest `memory.current` delta in the last second (+102MB).
- **Containment action:** `memory.high` of the `payment-service-7b4f` cgroup reduced from `2G` (original pod configuration) to `1.6G`. This instructs the kernel to apply memory backpressure (aggressive reclaim) to the container, decelerating the allocation rate without killing the process.
- Opportunistic webhook fired: `POST /api/v1/alerts` with severity `warning`, payload containing state vector and dimensional contribution.

**Prometheus:** Next scrape in 5 seconds. Prometheus will still display metrics from the previous scrape (t=-8s), which showed a healthy system.

**t = 4s (14:23:11.000) — Containment holding, derivative decelerating**

```
  mem_used:      72%    (still rising, but rate reduced by memory.high containment)
  mem_pressure:  34%    (rising, but decelerating — active reclaim)

D_M = 5.9
φ = +2.1
dD̄_M/dt = +1.2/s   (DECELERATING — from +2.1/s to +1.2/s)
d²D̄_M/dt² = -0.45/s² (NEGATIVE — containment is working)

Response Level: 2 (maintains Light Containment — d²D̄_M/dt² negative indicates
                   mitigation is effective. No unnecessary escalation.)
```

**HOSA actions:**
- Log: `[CONTAINMENT-HOLDING] D_M=5.9 dDM/dt=+1.2(decreasing) d2DM/dt2=-0.45 action=memory.high_effective target=payment-service-7b4f`
- Maintains `memory.high` at 1.6G. Monitors whether deceleration continues.
- `kubelet`, `containerd`, and other pod processes **are not affected** — they are on the safelist.

**Prometheus:** Executes scrape at this instant (t=4s). Collects `container_memory_usage_bytes{pod="payment-service-7b4f"} = 1.47GB`. Stores in TSDB. Alert rule: "container_memory_usage > 1.8GB for 1m" — **condition not satisfied** (memory is at 1.47GB thanks to HOSA's containment, and the `for 1m` condition requires sustaining for 60 seconds).

**t = 8s (14:23:15.000) — Stabilization by containment**

```
  mem_used:      74%    (stabilizing — reclaim equaling allocation rate)
  mem_pressure:  36%    (stable)
  cpu_total:     58%    (Java GC working continuously)

D_M = 6.2
φ = +2.2
dD̄_M/dt = +0.15/s   (near zero — system stabilizing at new plateau)
d²D̄_M/dt² ≈ 0        (zero acceleration — neither worsening nor improving)

Response Level: 2 (maintained)
```

**HOSA bought time.** The system is contained at a degraded but functional plateau. The `payment-service` is slow (memory backpressure causes higher latency), but did not crash. In-flight transactions were not corrupted. No process was killed.

**Prometheus:** Second scrape since the leak started. Collects `container_memory_usage_bytes = 1.52GB`. Alert condition: 1.52GB < 1.8GB, and `for 1m` not elapsed. **No alert.**

**t = 19s (14:23:26.000) — Prometheus collects third sample**

Collects `container_memory_usage_bytes = 1.55GB`. Alert condition: memory below threshold. **No alert.** HOSA's containment is preventing the metric collected by Prometheus from reaching the configured threshold.

**t = 35s (14:23:42.000) — Operator receives HOSA webhook**

The human operator or automation system receives the webhook sent by HOSA at t=2s (delivery may have variable latency depending on webhook infrastructure). The payload contains:

```json
{
  "severity": "warning",
  "node": "worker-node-07",
  "timestamp": "2026-03-04T22:23:09.000Z",
  "hosa_level": 2,
  "d_m": 4.7,
  "d_m_derivative": 2.1,
  "dominant_dimension": "mem_used",
  "dominant_contribution_pct": 68,
  "suspected_cgroup": "/kubepods/pod-payment-service-7b4f",
  "action_taken": "memory.high reduced to 1.6G",
  "action_status": "effective (d2DM/dt2 < 0)"
}
```

The operator can now take informed action: investigate the memory leak in `payment-service`, roll back the deployment, or scale horizontally. The information arrived with **dimensional context** (which resource, which process, which action was taken, whether it is working) — not as a generic binary alert.

**t = 60s (14:24:07.000) — Counterfactual scenario without HOSA**

If HOSA were not operating:
- At 50MB/s, the container would have allocated ~3GB in 60 seconds, exceeding the cgroup 2GB limit.
- The kernel would have triggered the OOM-Killer against the Java process of `payment-service` at t ≈ 40s.
- All in-flight payment transactions would have been aborted without graceful shutdown.
- The kubelet would have restarted the pod (CrashLoopBackOff), but the memory leak would persist, causing crash cycles every ~40 seconds.
- Prometheus would finally emit an alert (condition `for 1m` satisfied) at t ≈ 100s — **60 seconds after the first crash.**
- Customers would have experienced 502/504 errors in financial transactions throughout that period.

#### Temporal Synthesis

```
t=0s      t=1s      t=2s      t=4s       t=8s       t=15s      t=40s     t=100s
 │         │         │         │          │          │          │          │
 │ Leak    │ HOSA    │ HOSA    │ HOSA     │ HOSA     │Prometheus│ NO HOSA: │Prometheus
 │ starts  │ detects │ contains│ confirms │stabilizes│1st scrape│ OOM-Kill │ alert
 │         │ (L1)    │ (L2)    │ efficacy │ system   │ post-leak│ (crash)  │ (late)
 │         │         │         │          │          │          │          │
 ├─────────┴─────────┤         │          │          │          │          │
 │  LETHAL INTERVAL  │         │          │          │          │          │
 │  (2 seconds)      │         │          │          │          │          │
 │  HOSA acted here  │         │          │          │          │          │
 └───────────────────┘         │          │          │          │          │
                               │          │          │          │          │
                    ┌──────────┴──────────┴──────────┤          │          │
                    │  HOSA maintains active         │          │          │
                    │  containment                   │          │          │
                    │  System degraded but ALIVE     │          │          │
                    └────────────────────────────────┘          │          │
                                                                │          │
                                                      ┌─────────┴──────────┤
                                                      │ NO HOSA: cascade   │
                                                      │ OOM → crash → 502  │
                                                      │ → CrashLoopBackOff │
                                                      │ → late alert       │
                                                      └────────────────────┘
```

HOSA transformed a **destructive crash with transaction loss** scenario into a **controlled degradation with functionality preservation** scenario. Detection time was 1 second (vs. >60 seconds for the exogenous model). Mitigation preserved the integrity of in-flight transactions. The operator received actionable information with complete dimensional context.

This is the Lethal Interval in operation — and the demonstration of why immediate decision-making capability must reside in the node itself.

---

## 6. Taxonomy of Operational Regimes and HOSA's Behavioral Classification

### 6.1. The Demand Classification Problem

The effectiveness of an anomaly detection system depends fundamentally on its ability to **distinguish between legitimate variation and pathological deterioration**. A detector that treats every deviation as a threat generates operational fatigue from false positives. An overly tolerant detector allows sophisticated attacks to operate below the perception threshold.

The challenge is compounded by the fact that, from a purely metric perspective, radically different scenarios can produce superficially similar signatures. CPU at 85% may mean:

- A normal day of operation for a video rendering server;
- A predictable seasonal Black Friday spike for an e-commerce platform;
- The first milliseconds of a volumetric DDoS attack;
- A silent cryptominer consuming idle cycles.

The isolated metric is identical. What differentiates these scenarios is the **multivariable structure of the stress** — how variables correlate with each other — and the **temporal dynamics** — how that correlation evolves over time. This is precisely the distinction that the Mahalanobis Distance and its derivatives allow capturing.

Equally critically, the taxonomy must recognize that anomaly is not exclusively a phenomenon of **excess**. CPU at 2% on a server that should be processing a thousand requests per second is not homeostasis — it is **anomalous silence**, with financial, energetic, and security implications that anomaly detection literature has historically ignored.

This section formalizes a **taxonomy of operational regimes** organized as a continuous bipolar spectrum, detailing for each regime: its operational definition, its mathematical signature in Mahalanobis space, HOSA's expected behavior, and the interaction with the habituation mechanism.

---

### 6.2. The Continuous Bipolar Spectrum: Taxonomy Architecture

#### 6.2.1. Organizing Principle

HOSA's taxonomy models operational regimes as a **continuous numeric spectrum centered on homeostasis**, where:

- The **sign** of the index indicates the **direction** of deviation relative to the baseline profile;
- The **magnitude** of the index indicates the **severity** of the deviation.

Regime 0 (homeostasis) constitutes the central reference point. Negative deviations represent **under-demand** states (the node operates below expectations). Positive deviations represent **over-demand or anomaly** states (the node operates above expectations or under pathological conditions).

```
    Under-demand                   Over-demand / Anomaly
    ◄──────────────────────┤├───────────────────────────────────►

    −3      −2      −1      0      +1     +2     +3     +4     +5
    │       │       │       │       │      │      │      │      │
 Anomalous Struc-  Legit-  Homeo- Baseline Seaso- Adver- Local  Viral
 Silence  tural   imate   stasis  Shift   nality  sarial Fail.  Prop.
         Idle     Idle
    │                       │                                   │
    └── severity ──────────►│◄──────────── severity ────────────┘
        increasing          │              increasing
        (deficit)         BASELINE       (excess/pathology)
```

#### 6.2.2. Design Rationale

This spectral organization resolves three problems that ad hoc taxonomies introduce:

**Conceptual symmetry.** Biological homeostasis is inherently bidirectional: hypothermia and hyperthermia are both pathologies, with baseline temperature as the central reference. Similarly, HOSA treats under-demand and over-demand as symmetric deviations relative to the baseline profile, not as ontologically distinct categories.

**Numerical continuity.** The regime's integer index reflects a natural ordering of severity in each semi-axis. Transitions between adjacent regimes are smooth and auditable (e.g., from −1 to −2 when legitimate idleness reveals itself as structural; from +3 to +4 when adversarial activity causes localized failure).

**Uniformity of the mathematical framework.** The same primary metric ($D_M$) and the same Load Direction Index ($\phi$) position any observed state in the spectrum. The sign of $\phi$ determines the semi-axis; the combination of $D_M$, its derivatives, and supplementary metrics determines the position within the semi-axis.

#### 6.2.3. Directionality: Extending the Mahalanobis Distance

The Mahalanobis Distance, being a distance metric, is inherently **non-directional** — it measures how far the state has departed from the baseline, but does not indicate whether the deviation is "upward" (overload) or "downward" (idleness). To position the state in the bipolar spectrum, HOSA requires an indicator of **direction of deviation** in the multivariable space.

**Load Direction Index ($\phi$):**

Given the deviation vector $\vec{d}(t) = \vec{x}(t) - \vec{\mu}$, we define the Load Direction Index as the normalized weighted projection of the deviation onto the load axis:

$$\phi(t) = \frac{1}{n} \sum_{j=1}^{n} s_j \cdot \frac{d_j(t)}{\sigma_j}$$

where:
- $d_j(t) = x_j(t) - \mu_j$ is the deviation of the $j$-th variable relative to its baseline mean;
- $\sigma_j = \sqrt{\Sigma_{jj}}$ is the baseline standard deviation of the $j$-th variable;
- $s_j \in \{+1, -1\}$ is the **load sign** of the variable: $+1$ if an increase indicates higher load (CPU utilization, memory usage, network throughput), $-1$ if an increase indicates lower load (CPU idle, free memory);
- $n$ is the dimensionality of the state vector.

**Interpretation:**

| Value of $\phi(t)$ | Meaning | Semi-axis |
|---|---|---|
| $\phi \approx 0$ | System near baseline | Regime 0 |
| $\phi > 0$ | Deviation in direction of **overload** | Positive semi-axis (+1 to +5) |
| $\phi < 0$ | Deviation in direction of **idleness** | Negative semi-axis (−1 to −3) |

---

### 6.3. Regime 0 — Operational Homeostasis

**Definition:** The normal steady state of the node under its typical workload. Resource variables fluctuate within a predictable range, reflecting the ordinary activity of hosted applications.

**Mathematical signature:**

| Indicator | Behavior |
|---|---|
| $D_M(t)$ | Low and stable, fluctuating near the origin of the normalized space. Typically $D_M < \theta_1$. |
| $\phi(t)$ | Oscillates around zero. No sustained directional trend. |
| $\frac{d\bar{D}_M}{dt}$ | Oscillates around zero. No sustained directional trend. |
| $\frac{d^2\bar{D}_M}{dt^2}$ | Low-amplitude stationary noise. |
| Matrix $\Sigma$ | Stable. Correlations between variables are consistent over time. |

**HOSA behavior:**

- **Response Level:** 0 (Homeostasis).
- **Thalamic Filter active:** HOSA suppresses sending of detailed telemetry to external systems. Only a minimal heartbeat is emitted periodically, confirming that the node is alive and in homeostasis. This drastically reduces data ingestion cost (FinOps).
- **Baseline update:** $\vec{\mu}$ and $\Sigma$ continue to be updated incrementally via Welford, continuously refining the baseline profile.

---

### 6.4. Negative Semi-Axis: Under-Demand Regimes (−1, −2, −3)

#### 6.4.1. Rationale for Inclusion

The entirety of the anomaly detection literature in computational systems concentrates on **anomaly by excess**: resource consumption above expected, traffic above forecast, latency above acceptable. This asymmetry reflects an understandable operational bias — it is excess that brings down services. However, by focusing exclusively on positive anomaly, the industry systematically ignores a phenomenon with equally significant financial, energetic, and security implications: **anomaly by deficit**.

A server that should be processing a thousand requests per second and is processing zero is not in homeostasis. It is in **anomalous silence**. That silence has a cost: the machine continues consuming electricity, occupying rack space, depreciating hardware, and generating licensing costs — all without producing value.

More critically, anomalous silence can be a **symptom of compromise**. A server whose traffic has suddenly disappeared may indicate DNS hijacking, BGP redirection, upstream failure invisible to the local node, or an attacker who shut down the application process to replace it.

If HOSA aspires to implement genuine homeostasis — not merely overload protection — it must detect and classify deviations in **both directions** of the baseline profile. The negative semi-axis of the spectrum formalizes this capability.

---

#### 6.4.2. Regime −1 — Legitimate Idleness

**Definition:** Demand reduction compatible with the temporal or operational context. Resource consumption is below the global baseline profile, but is **coherent** with the baseline profile of the corresponding time window. Examples:

- Nighttime on a corporate web application server;
- Weekend on an ERP server;
- Scheduled upstream maintenance.

**Mathematical signature:**

| Indicator | Behavior |
|---|---|
| $D_M(t)$ | Elevated relative to the **global** baseline, but **low** relative to the baseline profile of the corresponding time window (if seasonal profiles are already calibrated — Section 6.7). |
| $\phi(t)$ | Moderately negative. |
| $\frac{d\bar{D}_M}{dt}$ | Approximately zero or with smooth transition (demand drop was gradual and predictable). |
| $\rho(t)$ | **Low** — correlation structure is preserved. Resources decrease proportionally, maintaining the same relationships. Less network → less CPU → less I/O, in the same proportion as the baseline profile. |
| Temporal context | **Coherent** — the period corresponds to a historically low-activity window. |

**HOSA behavior:**

| Aspect | Action |
|---|---|
| **Response Level** | 0 (Homeostasis) — idleness is expected. |
| **Thalamic Filter** | **Maximally active.** If the node is idle and healthy, telemetry is suppressed to absolute minimum — periodic heartbeat confirming "alive, idle, healthy" status. |
| **FinOps Signaling** | HOSA records underutilization metrics locally and, when connectivity is available, exposes an **idleness report** that quantifies: accumulated idle hours, estimated cost of keeping the node active during that period (if configured with instance cost-per-hour data), and downscale window recommendation. |
| **GreenOps — Energy Optimization** | HOSA can instruct the kernel to apply local energy optimizations that do not affect quick-return capacity: CPU frequency reduction via scaling governor; network interface polling frequency reduction for idle interfaces; HOSA's own sampling interval increase. |

**Reversibility:** All energy optimizations are **instantly reversible**. If $\phi(t)$ starts rising (traffic returning), HOSA restores frequencies and sampling intervals before load reaches the baseline profile, ensuring the node is at full capacity when demand arrives.

---

#### 6.4.3. Regime −2 — Structural Idleness

**Definition:** The node is **permanently** oversized relative to actual demand. There is no time window in which its resources are fully utilized. Examples:

- Instance provisioned based on incorrect capacity estimation;
- Legacy server that has lost operational relevance but was not decommissioned;
- Infrastructure provisioned for projected peaks that never materialized.

**Dedicated metric: Excess Provisioning Index (EPI)**

$$EPI = 1 - \frac{\max_{i \in \text{windows}} \|\vec{\mu}_i\|_{load}}{\vec{C}_{max}}$$

where $\vec{C}_{max}$ is the maximum hardware capacity vector (total CPU, total memory, etc.) and $\|\vec{\mu}_i\|_{load}$ is the weighted norm of the baseline profile $i$'s mean vector, projected onto load dimensions.

An EPI close to 1 indicates that, even during the highest-activity periods, the node utilizes a minimal fraction of its capacity — strong indicator of oversizing.

**HOSA behavior:**

| Aspect | Action |
|---|---|
| **Response Level** | 0 (no immediate operational risk). |
| **FinOps Signaling (critical)** | This is the highest financial impact subclass. HOSA generates an **oversizing report** containing: calculated EPI with historical data; vector of maximum used capacity vs. provisioned capacity, per resource; suggestion of a smaller instance compatible with the observed maximum load (when configured with cloud provider instance catalog); projected annual savings estimate. |
| **Orchestrator Exposure** | When integrated with Kubernetes (Phase 2), HOSA can expose the node with a taint or annotation indicating `hosa.io/structurally-idle=true`, allowing the cluster autoscaler to consider the node as a decommissioning candidate. |

**Interaction with FinOps and HOSA's Philosophy:** HOSA does not autonomously make the decision to shut down or resize the node — this would exceed its local action scope and could violate availability contracts. It **provides the mathematical evidence** for the human or orchestrator to make an informed decision. Financial and energy savings are a **consequence** of precise detection, not a direct action of the agent.

---

#### 6.4.4. Regime −3 — Anomalous Silence

**Definition:** Abrupt or gradual drop in activity **incompatible** with the expected temporal context. The node should be active and is not. Examples:

- Traffic redirected by DNS hijacking;
- Silent failure of load balancer that stopped sending requests;
- Application process killed without restart;
- Attack that brought down the service before installing payload.

**HOSA behavior:**

| Aspect | Action |
|---|---|
| **Response Level** | **1 (Vigilance) to 3 (Active Containment)**, depending on the speed and magnitude of the drop, and the presence of compromise indicators. |
| **Active investigation** | HOSA executes supplementary checks when anomalous silence is detected: process verification (are expected application processes still running?); network verification (are network interfaces operational? Is there outbound connectivity?); upstream verification (if HOSA knows upstream endpoints, it can execute a reverse health check). |

**The paradox of silence as an alarm:** Anomalous Silence is, counterintuitively, one of HOSA's most valuable scenarios. Traditional monitors are designed to alert about excess. When a server stops receiving traffic and all metrics are "green" (low CPU, free memory, calm network), the traditional monitor reports: "all healthy." HOSA, by modeling the expected baseline profile and not just capacity limits, detects that the silence is anomalous and signals: "this node should be active and is not."

**Interaction with habituation:** **Blocked.** HOSA does not habituate to silence incoherent with the temporal context.

---

#### 6.4.5. Consolidated Mathematical Signature — Negative Semi-Axis

| Indicator | Regime −1 (Legitimate) | Regime −2 (Structural) | Regime −3 (Anomalous) |
|---|---|---|---|
| $D_M(t)$ vs. global baseline | Moderate | Chronically low | High (abrupt) |
| $D_M(t)$ vs. temporal profile | **Low** (coherent) | Low in all windows | **High** (incoherent) |
| $\phi(t)$ | Moderately negative | Persistently negative | **Strongly negative** |
| $\frac{d\phi}{dt}$ | Gradual | ≈ 0 (stable) | **Abrupt** |
| $\rho(t)$ | Low (correlations preserved) | Low | Variable (possibly high) |
| Temporal coherence | **Yes** | Irrelevant (always idle) | **No** |
| $EPI$ | Variable | **Close to 1** | Irrelevant |

#### 6.4.6. Theoretical Contribution of Sub-Demand Detection

The inclusion of the negative semi-axis in HOSA's spectrum introduces a **conceptual symmetry** absent in anomaly detection literature for computational systems. Anomaly is redefined as **significant deviation from the baseline profile in any direction** — not just toward excess.

This symmetry enables three practical contributions that, as far as this work's literature review identifies, are not addressed by any existing local agent in an integrated manner:

**1. FinOps grounded in endogenous evidence.** Cloud cost optimization tools (AWS Cost Explorer, GCP Recommender, Kubecost) operate on billing data and metrics aggregated at hourly or daily intervals. HOSA offers underutilization evidence at second-level granularity, including multivariable correlation and temporal context, enabling right-sizing recommendations with greater precision and statistical confidence.

**2. GreenOps as a consequence of homeostasis.** Energy optimization is not implemented as a separate module, but as the **natural response of the agent to the under-demand regime** — exactly as biological metabolism reduces energy consumption at rest. CPU frequency reduction, sampling interval increase, and telemetry reduction are actions of the same graduated response system that applies throttling under overload. Homeostasis is bidirectional.

**3. "Operational blackout" detection as a security capability.** Anomalous Silence (Regime −3) is a genuine security scenario that traditional resource health monitors are structurally incapable of detecting — precisely because all capacity metrics are "healthy" when the server stops receiving work. Detection requires a model of "what should be happening" (contextualized baseline profile), not just "what is dangerous" (capacity thresholds).

---

### 6.5. Regime +1 — High Baseline Demand (Permanent Plateau Shift)

**Definition:** A **persistent and unreversed** elevation in resource consumption, caused by legitimate changes in workload nature. Examples:

- Deployment of new application version with higher memory consumption;
- Migration of an additional microservice to the same node;
- Organic growth of user base;
- Kernel or runtime update that alters the consumption profile.

**The critical discriminant:** The derivative converging to zero while $D_M$ remains elevated is the fundamental signature of this regime. It differentiates itself from an ongoing attack, where the derivative remains positive or accelerates. Additionally, the **preservation of covariance structure** is a strong legitimacy indicator: legitimate workload changes typically maintain the consumption proportions between resources, while pathologies introduce anomalous correlations.

The covariance matrix deformation ratio $\rho(t)$ is calculated as:

$$\rho(t) = \frac{\|\Sigma_{recent} - \Sigma_{baseline}\|_F}{\|\Sigma_{baseline}\|_F}$$

A low $\rho$ with high $D_M$ indicates a plateau shift with structure preservation (Regime +1). A high $\rho$ indicates **covariance structure deformation** (potentially Regime +3 or +4).

**Interaction with habituation:** This regime is the **primary use case for habituation.** When stability and covariance preservation criteria are satisfied, HOSA recalibrates $\vec{\mu}$ and $\Sigma$ to reflect the new operational regime. The new plateau becomes the baseline.

---

### 6.6. Regime +2 — Seasonal High Demand (Predictable Periodicity)

**Definition:** Demand variations that follow recurring temporal patterns, determined by predictable usage cycles. Examples:

- Daily access peaks between 09:00 and 11:00 in corporate applications;
- Nighttime traffic drop;
- Weekly peaks (Monday in ERPs, Friday in e-commerce);
- Monthly seasonality (accounting close, payroll);
- Annual seasonality (Black Friday, marketing campaigns).

**Solution: Baseline Profiles Indexed by Temporal Context (Digital Circadian Rhythm)**

HOSA implements a **temporal baseline segmentation** mechanism. Instead of maintaining a single pair ($\vec{\mu}$, $\Sigma$), the agent maintains **N baseline profiles** indexed by time window:

$$\mathcal{B} = \{(\vec{\mu}_i, \Sigma_i, w_i) \mid i = 1, 2, \ldots, N\}$$

Segmentation granularity is determined automatically during the first weeks of operation through **autocorrelation analysis** of the $D_M$ time series. If periodicity is detected (e.g., 24h peak), $\mathcal{B}$ is automatically segmented into corresponding windows; each segment accumulates its own baseline profile via independent Welford.

**Practical implication:** The 09:00 Monday peak is compared with the "Monday 08:00–12:00" baseline profile, not the "Sunday 03:00" profile. This eliminates seasonal false positives without sacrificing sensitivity.

---

### 6.7. Regime +3 — Disguised High Demand (Adversarial Demand)

**Definition:** Resource consumption caused by malicious activity that **deliberately mimics legitimate demand patterns** to evade detection. This is the highest adversarial sophistication category and includes:

- **Application layer DDoS (Layer 7):** Syntactically valid HTTP requests generated by botnets that simulate human browsing;
- **Parasitic cryptomining:** Processes consuming CPU at levels calculated to stay below alert thresholds;
- **Slow data exfiltration (Low-and-Slow):** Low-volume but continuous network transfers, diluted over hours;
- **Resource exhaustion attacks:** Gradual opening of file handles, socket connections, or threads until reaching OS limits.

**The central problem:** The sophisticated adversary knows detection thresholds (or assumes they exist) and deliberately operates below them. Against a one-dimensional threshold detector, this is trivial: just keep each individual metric below the threshold. Against the Mahalanobis Distance, evasion is significantly more difficult — but not impossible.

**Central thesis of this classification:** Even when individual **magnitudes** are kept within normal range, malicious activity produces **deformation in the covariance structure** that legitimate demand does not produce. This occurs because malicious activity consumes resources **disproportionately** relative to the machine's legitimate work profile.

**Second-Level Metrics — Structural Deformation Detection:**

- **Shannon entropy of syscall profile:** $H(S, t) = -\sum_{i=1}^{k} p_i(t) \log_2 p_i(t)$. A significant change in $H$ without a corresponding change in application metrics (throughput, response latency) is indicative of anomalous activity.

- **Work Efficiency Index (WEI):** $WEI(t) = \frac{\text{application throughput}(t)}{\text{computational resource consumption}(t)}$. Cryptomining and parasitic processing consume CPU/memory without producing application throughput, causing **WEI decline** even when no individual metric is in alert range.

- **Kernel/User Context Ratio:** $R_{ku}(t) = \frac{\text{CPU in kernel mode}(t)}{\text{CPU in user mode}(t)}$. Network attacks and network malware produce disproportionate increases in kernel space time.

**Interaction with habituation:** Habituation is **blocked** when the deformation ratio $\rho(t)$ exceeds the deformation threshold. HOSA **does not habituate to activity that deforms the covariance structure**, even if stable in magnitude. This prevents a persistent attacker from "training" the detector to accept their presence as normality.

---

### 6.8. Regime +4 — Non-Viral Anomaly (Localized Failure)

**Definition:** Resource deterioration caused by failure or pathology **confined to the local node**, without a propagation component to other systems. Examples:

- Memory leak in an application process;
- Disk degradation (defective sectors, growing I/O latency);
- Application bug causing accumulation of file descriptors or threads;
- Accidental or intentional fork bomb;
- Application deadlock causing request accumulation in queue;
- CPU thermal degradation (thermal throttling by hardware).

**Dimensional Contribution Decomposition:**

To diagnose **which resources** are causing the deviation, HOSA decomposes $D_M$ into per-dimension contributions. Given the deviation vector $\vec{d} = \vec{x}(t) - \vec{\mu}$ and the Mahalanobis metric $D_M^2 = \vec{d}^T \Sigma^{-1} \vec{d}$, the contribution of the $j$-th dimension is:

$$c_j = d_j \cdot (\Sigma^{-1} \vec{d})_j$$

The dimensions with the highest $c_j$ are the **dominant contributors** of the anomaly. This allows HOSA to:

1. Direct throttling actions to the processes most consuming the contributing resource;
2. Record in the log the **mathematical reason** for the decision (auditability);
3. When SLM is available (Phase 3), provide dimensional context for causal diagnosis.

**Interaction with habituation:** Habituation is **blocked when the derivative remains sustainedly positive.** Monotonically growing anomalies are not "new normals" — they are progressive failures.

---

### 6.9. Regime +5 — Viral Anomaly (Propagation and Contagion)

**Definition:** Malicious activity or cascade failure with a **propagation component between nodes**, where the affected node attempts to compromise, overload, or infect other systems on the network. Examples:

- Worms and malware with lateral propagation capability;
- Post-compromise lateral movement (attacker pivot);
- Microservice failure cascade (a degraded service causes backpressure in upstream dependents);
- Compromised node used as base for internal DDoS (amplification).

**Formal metric: Propagation Behavior Index (PBI)**

$$PBI(t) = w_1 \cdot \hat{C}_{out}(t) + w_2 \cdot \hat{H}_{dest}(t) + w_3 \cdot \hat{F}_{anom}(t) + w_4 \cdot \hat{\rho}_{D_M \leftrightarrow net_{out}}(t)$$

where:
- $\hat{C}_{out}(t)$: normalized rate of new outbound connections (against baseline profile);
- $\hat{H}_{dest}(t)$: normalized entropy of destination IPs;
- $\hat{F}_{anom}(t)$: normalized rate of anomalous forks/execs;
- $\hat{\rho}_{D_M \leftrightarrow net_{out}}(t)$: correlation between $D_M$ and outbound traffic (positive = viral indicator);
- $w_i$: empirically calibrated weights.

**Weight Calibration Strategy:**

**Stage 1 — Uniform Initialization.** In the absence of empirical data, weights are initialized uniformly: $w_i = \frac{1}{4}$ for $i \in \{1, 2, 3, 4\}$.

**Stage 2 — Calibration via Sensitivity Analysis.** During the experimental phase, HOSA is subjected to controlled attack scenarios with known ground truth (confirmed presence/absence of propagation). Weights are then calibrated via **AUC-ROC maximization**:

$$\vec{w}^* = \arg\max_{\vec{w}} \text{AUC-ROC}\left(\{PBI^{(j)}(\vec{w}), y^{(j)}\}_{j=1}^{M}\right)$$

**Stage 3 — Cross-Validation and Weight Publication.** Calibrated weights are validated via leave-one-out cross-validation. Final values of $\vec{w}^*$, along with the confidence interval for each weight and the resulting AUC-ROC, are published as reference parameters of the implementation.

**Interaction with habituation:** Habituation is **categorically blocked** when $PBI > PBI_{threshold}$. HOSA never habituates to propagation patterns.

---

### 6.10. Exogenous Contextual Signals as Supplementary Dimension of the State Vector

Anomaly detection based exclusively on endogenous resource metrics (CPU, memory, I/O, network) is powerful, but can be enriched with **contextual signals** that inform HOSA about the *expected reason* for demand variations.

The most fundamental contextual signal — requiring no external dependency — is **time**. Cyclic encoding is used to avoid discontinuities (23h→0h):

$$x_{hour,sin}(t) = \sin\left(\frac{2\pi \cdot hour(t)}{24}\right), \quad x_{hour,cos}(t) = \cos\left(\frac{2\pi \cdot hour(t)}{24}\right)$$

In Edge Computing and industrial IoT scenarios, environmental signals (ambient temperature, humidity, supply voltage, vibration/acceleration) can also be incorporated into the state vector. The Covariance Matrix automatically captures correlations between environmental conditions and resource metrics, allowing HOSA to **discount** performance variations caused by physical environmental factors rather than software pathologies.

**Design principle: graceful degradation.** If no environmental sensors are available, HOSA operates without those dimensions. The presence of sensors **improves** classification; their absence **does not prevent** functioning.

Certain contextual signals can be provided by the operator as **configuration metadata** loaded at HOSA deployment:

| Signal | Use |
|---|---|
| **Event calendar** | Allows HOSA to **preemptively relax thresholds** during expected high-demand events. |
| **Workload profile** | Allows calibration of relative weights in the state vector (database server has dominant I/O profile; web server has dominant network profile). |
| **Geographic zone** | Used in Phase 4+ to contextualize swarm communication between nodes. |
| **Client time zones** | Refines temporal segmentation of the baseline profile when users are in different time zones from the server. |

---

### 6.11. Synthesis: Integrated Classification Matrix

| Regime | $D_M$ | $\frac{dD_M}{dt}$ | $\frac{d^2D_M}{dt^2}$ | $\phi(t)$ | $\rho(t)$ | $\Delta H$ | $PBI$ | Classification |
|---|---|---|---|---|---|---|---|---|
| **−3** | High (abrupt) | Peak | Variable | **Strongly negative** | Variable | Variable | Variable | **Anomalous Silence** → Investigation |
| **−2** | Chronically low | ≈ 0 | ≈ 0 | **Persistently negative** | Low | Low | Low | **Oversizing** → FinOps |
| **−1** | Low (vs. temporal profile) | ≈ 0 or smooth | ≈ 0 | **Negative** | Low | Low | Low | **Legitimate Idleness** → FinOps/GreenOps |
| **0** | Low | ≈ 0 | ≈ 0 | ≈ 0 | Low | Low | Low | **Homeostasis** |
| **+1** | High, stable | ≈ 0 (after transient) | ≈ 0 | **Positive** | Low | Low | Low | **Plateau shift** → Habituation |
| **+2** | Oscillates | Oscillates | Oscillates | **Oscillates** | Low | Low | Low | **Seasonality** → Temporal profiles |
| **+3** | Any | Any | Any | Positive | **High** | **High** | Variable | **Adversarial** → Containment |
| **+4** | Growing | Sustained positive | Variable | Positive | Variable | Low | **Low** | **Localized failure** → Graduated containment |
| **+5** | Variable | Variable | Variable | Variable | Variable | Variable | **High** | **Propagation** → Network isolation |

---

### 6.12. Habituation Mechanism Implications: Consolidated Rules

**Necessary preconditions (all must be satisfied simultaneously):**

$$\text{Habituation} \iff \begin{cases} \left|\frac{d\bar{D}_M}{dt}\right| < \epsilon_d & \text{(stabilization)} \\ \rho(t) < \rho_{threshold} & \text{(covariance preserved)} \\ \Delta H(t) < \Delta H_{threshold} & \text{(stable syscalls)} \\ PBI(t) < PBI_{threshold} & \text{(no propagation)} \\ D_M(t) < D_{M,safety} & \text{(safe plateau)} \\ t_{stable} > T_{min} & \text{(sustained stabilization)} \\ \text{temporal coherence of } \phi(t) & \text{(if } \phi < 0 \text{, coherent with seasonal profile)} \end{cases}$$

**Regimes and habituation across the spectrum:**

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

**Visual pattern:** Habituation is permitted in the central spectrum regimes (−2 to +2), where deviations are legitimate or structural. It is blocked at the extremes (−3, +3 to +5), where deviations are pathological or adversarial. This symmetry reflects the principle that HOSA adapts to legitimate variation but refuses to normalize pathology.

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

---

## 7. Language Choice: Trade-off Analysis

| Criterion | Go | Rust | C |
|---|---|---|---|
| **GC latency** | GC with sub-ms pauses (Go 1.22+), but non-deterministic. Mitigable with `sync.Pool`, pre-allocation, and `GOGC` tuning. | No GC. Deterministic latency. | No GC. Deterministic latency. |
| **eBPF ecosystem** | `cilium/ebpf` (mature and active library). | `aya-rs` (active library, smaller ecosystem). | `libbpf` (kernel upstream reference). |
| **Development speed** | High. Fast compilation. Native concurrency (goroutines). | Medium. Borrow checker requires discipline. Slow compilation. | Low. Manual memory management. |
| **Memory safety** | Guaranteed by runtime. | Guaranteed by compiler (no runtime). | Programmer's responsibility. |
| **Academic adequacy** | Readable code, facilitates reproducibility. | Readable code with learning curve. | Prone to subtle bugs. |

**Provisional decision:** Go for the mathematical engine and control plane, with the hot path computation implemented with minimal allocation (slice pre-allocation, `sync.Pool`, `GOGC=off` during critical cycles). The rationale is pragmatic: for a master's dissertation scope, Go's iteration speed allows greater focus on validating the mathematical thesis.

**Validation commitment:** The dissertation will include comparative benchmarks of the hot path measuring p50/p99 latency and jitter, with explicit discussion of whether observed GC pauses impact the detection window in real collapse scenarios.

---

## 8. Roadmap: Executable Horizon and Long-Term Vision

### 8.1. Executable Horizon (Dissertation Scope and Immediate Continuity)

#### Phase 1: Foundation — The Mathematical Engine and the Reflex Arc (v1.0)

**Scope:** Complete implementation of the perceptive-motor cycle.

**Deliverables:**
- eBPF probes for state vector collection (CPU, memory, I/O, network, scheduler) via tracepoints and kprobes
- Mathematical engine with incremental Welford, Mahalanobis, EWMA, and derivatives
- Hardware proprioception (warm-up with automatic calibration)
- Graduated response system (Levels 0–4)
- Thalamic Filter: redundant telemetry suppression in homeostasis (minimal heartbeat)
- Benchmark of complete cycle latency (detection → decision → actuation)

**Experimental validation:**
- Controlled fault injection: gradual memory leak, fork bomb, CPU burn, network flood
- Quantitative comparison: HOSA detection and mitigation time vs. Prometheus+Alertmanager vs. systemd-oomd
- Sensitivity analysis of parameter $\alpha$ (EWMA) and adaptive thresholds
- Measurement of agent overhead (CPU, memory, added latency to the system)

#### Phase 2: Ecosystem Symbiosis (v2.0)

**Scope:** Opportunistic integration with orchestrators and monitoring systems.

**Deliverables:**
- Webhooks for K8s HPA/KEDA: preemptive scale-up triggering based on $D_M$ derivative
- HOSA metrics exposure in Prometheus-compatible format (for integration with existing dashboards)
- Enriched `/healthz` endpoint: instead of binary (healthy/unhealthy), returns normalized state vector
- Digital Endocrine System: long-term "fatiguability" metrics (thermal wear, SSD write cycles) exposed as labels for the Kubernetes scheduler

#### Phase 3: Local Semantic Triage (v3.0)

**Scope:** Introduction of post-containment causal analysis.

**Deliverables:**
- Small Language Model (SLM) running locally, activated **only** after Level 3+ containment to diagnose probable root cause
- Model operating **air-gapped** (without internet connection)
- Memory T-Cells: attack pattern signatures stored in eBPF Bloom Filter for nanosecond blocking in case of recurrence
- Autonomous Quarantine (Level 5): controlled network isolation
- Neural Habituation: automatic recalibration of baseline profile when workload changes are classified as benign by the SLM

**Note on footprint:** The SLM is a **conditional** component, activated only on nodes with sufficient resources (minimum recommended: 4GB available RAM). On resource-constrained devices (IoT, low-capacity Edge), Phase 3 is not deployed, and HOSA operates exclusively with the mathematical engine from Phases 1-2.

### 8.2. Long-Term Vision (Doctoral Scope and Future Research)

#### Phase 4: Swarm Intelligence (v4.0) — *Future Research*

**Research hypothesis:** HOSA-equipped nodes can establish local consensus on cluster state via lightweight P2P communication, reducing control plane dependency for collective health decisions.

#### Phase 5: Federated Learning and Collective Immunity (v5.0) — *Future Research*

**Research hypothesis:** Mathematical weight updates (not sensitive data) shared between HOSA instances can create collective immunity against emerging attack patterns.

**Recognized technical challenges:**
- Federated learning convergence in heterogeneous environments (Li et al., 2020)
- Resistance to model poisoning attacks
- Differential privacy (Dwork & Roth, 2014)

#### Phase 6: Hardware Offload (v6.0) — *Future Research*

**Research hypothesis:** Migration of the perceptive-motor cycle to dedicated hardware (SmartNIC/DPU) eliminates CPU competition with node applications and allows operation in low-power states.

#### Phase 7: eSRE — Methodological Formalization (v7.0) — *Future Research*

**Goal:** Consolidation of HOSA principles into an open methodology called **eSRE (Endogenous Site Reliability Engineering)**, documenting the "Laws of Cellular Survival" as recommended practices for resilient system design.

---

## 9. Known Limitations and Work Boundaries

1. **Distribution assumption.** The Mahalanobis Distance implicitly assumes that the baseline profile follows an approximately ellipsoidal (multivariate normal) distribution. Workloads with multimodal distributions may violate this assumption. The dissertation will investigate detector robustness under non-Gaussian distributions.

2. **Cold start.** During the warm-up phase (first minutes after initialization), the agent does not have sufficient baseline profile for reliable detection. In this interval, HOSA operates in conservative mode (logging only, no mitigation), constituting a vulnerability window.

3. **Adversarial evasion.** An attacker with knowledge of HOSA's architecture could, in theory, execute a "low-and-slow" attack that keeps $D_M$ and its derivatives below detection thresholds. Evasion resistance analysis is a future research topic (Phase 5).

4. **Throttling costs.** Throttling may introduce side effects, as detailed in Section 5.6. The effectiveness of the safelist mechanism and target process selection will be validated experimentally.

5. **OS scope.** HOSA is designed exclusively for the Linux kernel (≥ 5.8, with eBPF CO-RE support). Portability to other kernels is not a goal.

6. **NUMA interaction and hardware heterogeneity.** Systems with complex NUMA topology may exhibit localized pressure patterns that the aggregated state vector does not capture.

---

## 10. Anticipated Questions and Answers

**Q1: "Why not use Machine Learning / Deep Learning instead of Mahalanobis Distance? Autoencoders, LSTMs, and Isolation Forests are more sophisticated for anomaly detection."**

The choice of Mahalanobis Distance is not due to ignorance of more complex techniques — it is due to **appropriateness for the agent's operational requirements**.

HOSA must operate on any hardware running Linux ≥ 5.8, including IoT devices with 512MB RAM and no GPU. Autoencoders and LSTMs require: (a) training infrastructure; (b) inference runtime with significant footprint; (c) stored data windows for inference.

The Mahalanobis Distance with incremental Welford offers: (a) online calibration without a separate training phase; (b) fixed memory footprint $O(n^2)$ (for $n \leq 15$, this is < 2KB); (c) constant time calculation per sample ($O(n^2)$, ~microseconds for $n = 10$).

The question is not "which technique is more sophisticated?" — it is "which technique detects anomalies with sub-millisecond latency, constant memory, without GPU, on a Raspberry Pi?" The answer excludes deep learning and favors robust classical statistics.

Additionally, Mahalanobis Distance produces **interpretable** results: the operator can inspect which dimensions contribute to the deviation ($c_j$), understand the decision, and audit the agent's behavior. Deep learning models are opaque by construction, hindering auditability — a non-negotiable requirement for an agent that executes autonomous mitigation.

---

**Q2: "Isn't this just a HIDS (Host Intrusion Detection System) with a different name?"**

No. The distinction is structural, not cosmetic.

| Dimension | HIDS (e.g., OSSEC, Wazuh) | HOSA |
|---|---|---|
| **Primary focus** | Security — intrusion detection | Operational survival — node homeostasis maintenance |
| **Detection model** | Rules and known attack signatures (model of "known bad") | Baseline profile deviation (model of "known good") |
| **Monitored variables** | Logs, file integrity, suspicious syscalls | Resource metrics (CPU, memory, I/O, network) and their multivariable correlations |
| **Action** | Alert. Point blocking (e.g., fail2ban). | Autonomous graduated mitigation: cgroup throttling, XDP load shedding, quarantine |
| **Sub-demand detection** | No — HIDS is not interested in idle servers | Yes — Regimes −1, −2, −3 detect structural idleness and anomalous silence |
| **Network dependency** | Typically requires central server | Total autonomy for primary function |

HOSA can detect *consequences* of attacks (covariance deformation, Regime +3), but is not designed to replace specialized security tools. It complements HIDS the same way it complements Prometheus: operating in a different layer (resource health vs. security), on a different time horizon (milliseconds vs. minutes), with a different objective (keep the node alive vs. identify the attacker).

---

**Q3: "Why not contribute multivariable detection to `systemd-oomd` instead of creating a completely new agent?"**

`systemd-oomd`'s architecture is fundamentally incompatible with the model proposed by HOSA for three structural reasons:

1. **Monitoring scope.** `systemd-oomd` monitors exclusively **memory pressure** (PSI memory). HOSA monitors $n$ correlated variables (CPU, memory, I/O, network, scheduler). Adding multivariability to `oomd` would mean transforming it into something it was not designed to be.

2. **Action model.** `systemd-oomd` has one action: kill the entire cgroup. HOSA implements 6 levels of graduated response, including selective throttling, partial load shedding, and quarantine.

3. **systemd coupling.** `systemd-oomd` is a component of the systemd ecosystem. HOSA is designed as an autonomous agent without dependency on a specific init system, operating on any Linux environment (including containers with minimal init and embedded systems without systemd).

HOSA is not an OOM protection tool — it is a **systemic homeostasis agent**. The scope is categorically broader.

---

**Q4: "Can the resilience agent itself become the cause of the problem? What prevents HOSA from causing a crash?"**

This is a legitimate and central design concern. HOSA addresses it through multiple mechanisms:

1. **Controlled and self-limited footprint.** HOSA itself operates within a dedicated cgroup v2 with strict CPU and memory limits. If the agent exceeds its own limits, the kernel contains it before it affects the system. HOSA practices what it preaches.

2. **Safelist that includes itself.** HOSA is the first item in the safelist of processes protected against throttling.

3. **Reversible mitigation principle.** Response Levels 0-4 are automatically reversible. No destructive action (process kill, interface deactivation) is executed below Level 5.

4. **Escalation hysteresis.** Transition between levels requires sustaining activation conditions for minimum periods, preventing oscillation (flapping) between states.

5. **Dry-run mode.** The agent can be executed in pure observation mode (logging and decision calculation without executing actions), allowing decision quality validation before enabling actuation.

6. **Deterministic compilation.** The binary is statically compiled without dynamic dependencies.

---

**Q5: "What is the difference between HOSA and projects like Meta FBAR (Facebook Auto-Remediation)?"**

| Dimension | FBAR | HOSA |
|---|---|---|
| **Architecture** | Centralized. Remediation decisions made by central servers with global cluster view. | Distributed/local. Each node decides autonomously. |
| **Network dependency** | Total. FBAR requires continuous communication with Meta's observability infrastructure. | None for primary function. |
| **Decision latency** | Seconds to minutes. | Milliseconds (complete local cycle). |
| **Action scope** | Broad: can drain nodes, restart services, redirect traffic, scale clusters. | Restricted to local node: throttling, load shedding, quarantine. |
| **Availability** | Proprietary (Meta's internal infrastructure). | Open-source, portable to any Linux ≥ 5.8. |
| **Edge/IoT adequacy** | None (designed for hyperscale datacenters). | Designed to operate in any environment, including devices with intermittent connectivity. |

FBAR is Meta's answer for remediation at scale — an intelligent orchestrator of infrastructure actions. HOSA is a local survival reflex. They are complementary: in a datacenter equipped with both FBAR and HOSA, HOSA would stabilize the node in the initial milliseconds while FBAR deliberates and executes systemic remediation.

---

**Q6: "The Mahalanobis Distance is a 1936 technique. Isn't it obsolete?"**

Linear algebra and calculus are from the 18th century. We continue using them because they are correct.

The Mahalanobis Distance remains the standard metric for multivariate outlier detection in industrial statistics (quality control), medical diagnosis (physiological signal anomaly detection), and aerospace engineering (structural health monitoring). The reason is that its properties — correlation sensitivity, interpretability, and predictable computational cost — have not been surpassed by more recent techniques in the scenarios where those properties are requirements.

HOSA does not apply Mahalanobis naively. It extends it with: (a) incremental Welford update (constant footprint); (b) temporal derivative analysis (sensitivity to dynamics, not just state); (c) regularization for numerical robustness; (d) supplementary metrics for regime classification (covariance deformation, syscall entropy, PBI). Mahalanobis is the **foundation**, not the totality of the detection system.

---

**Q7: "How does HOSA behave in systems with highly variable load (e.g., serverless, Lambda functions, sporadic batch workloads)?"**

Highly variable workloads represent a legitimate challenge for any baseline-based detector, and HOSA addresses it in layers:

1. **Seasonal profiles (Section 6.6):** If variability is temporally predictable (batch jobs at fixed times, seasonal peaks), time-window-indexed profiles capture legitimate variability.

2. **Habituation (Section 5.5):** If variability is a permanent plateau change, the habituation mechanism recalibrates the baseline.

3. **Derivative tolerance:** HOSA scales responses based on the **acceleration** of the deviation, not just magnitude. A fast spike that stabilizes (like a batch job activation) produces transiently high derivative followed by stabilization — HOSA may briefly reach Level 1 (Vigilance) during the transient, but will not escalate to containment if acceleration ceases.

4. **Genuinely problematic scenario:** Workloads that vary **randomly** in magnitude and timing, without temporal pattern, without stabilization, and without predictable correlation between variables. For these scenarios, the fundamental premise of a "baseline profile" is weak, and HOSA's effectiveness is reduced. The limitations documentation (Section 9) recognizes this scenario. Investigation of detection models for non-stationary workloads without a baseline profile is a future research topic.

---

## 11. Expected Contributions

This work proposes the following contributions to the state of the art:

1. **Formalization of the Endogenous Resilience concept** as a paradigm complementary to exogenous observability, with precise definition of the operational limits of each approach.

2. **Real-time multivariate anomaly detection model** based on Mahalanobis with incremental update and rate-of-change analysis, validated against real and synthetic collapse scenarios.

3. **Reference architecture** for autonomous mitigation agents with kernel-space actuation, documenting design trade-offs (latency vs. stability, autonomy vs. mitigation risk).

4. **Quantitative comparative analysis** of detection and mitigation time between the endogenous model (HOSA) and the exogenous model (Prometheus + Alertmanager + orchestrator), contributing empirical data to a debate that has been predominantly theoretical.

5. **Graduated response framework** for autonomous mitigation, with explicit documentation of risks and protection mechanisms (safelist, hysteresis, quarantine vs. destruction).

---

## 12. References

Aggarwal, C. C. (2017). *Outlier Analysis* (2nd ed.). Springer.

Bear, M. F., Connors, B. W., & Paradiso, M. A. (2015). *Neuroscience: Exploring the Brain* (4th ed.). Wolters Kluwer.

Beyer, B., Jones, C., Petoff, J., & Murphy, N. R. (2016). *Site Reliability Engineering: How Google Runs Production Systems*. O'Reilly Media.

Brewer, E. A. (2000). Towards robust distributed systems. *Proceedings of the 19th Annual ACM Symposium on Principles of Distributed Computing (PODC)*.

Burns, B., Grant, B., Oppenheimer, D., Brewer, E., & Wilkes, J. (2016). Borg, Omega, and Kubernetes. *ACM Queue*, 14(1), 70–93.

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

Isovalent. (2022). Tetragon: eBPF-based Security Observability and Runtime Enforcement. Isovalent Open Source. https://tetragon.io/

Lamport, L. (1998). The Part-Time Parliament. *ACM Transactions on Computer Systems*, 16(2), 133–169.

Li, T., Sahu, A. K., Talwalkar, A., & Smith, V. (2020). Federated Learning: Challenges, Methods, and Future Directions. *IEEE Signal Processing Magazine*, 37(3), 50–60.

Mahalanobis, P. C. (1936). On the generalized distance in statistics. *Proceedings of the National Institute of Sciences of India*, 2(1), 49–55.

Mardia, K. V. (1970). Measures of Multivariate Skewness and Kurtosis with Applications. *Biometrika*, 57(3), 519–530.

Ongaro, D., & Ousterhout, J. (2014). In Search of an Understandable Consensus Algorithm. *Proceedings of the USENIX Annual Technical Conference (ATC)*.

Penny, K. I. (1996). Appropriate Critical Values When Testing for a Single Multivariate Outlier by Using the Mahalanobis Distance. *Journal of the Royal Statistical Society: Series C*, 45(1), 73–81.

Poettering, L. (2020). systemd-oomd: A userspace out-of-memory (OOM) killer. *systemd Documentation*. https://www.freedesktop.org/software/systemd/man/systemd-oomd.service.html

Prometheus Authors. (2012). Prometheus: Monitoring System and Time Series Database. Cloud Native Computing Foundation. https://prometheus.io/

Rousseeuw, P. J. (1984). Least Median of Squares Regression. *Journal of the American Statistical Association*, 79(388), 871–880.

Rousseeuw, P. J., & Van Driessen, K. (1999). A Fast Algorithm for the Minimum Covariance Determinant Estimator. *Technometrics*, 41(3), 212–223.

Scholz, D., Raumer, D., Emmerich, P., Kurber, A., Lessman, K., & Carle, G. (2018). Performance Implications of Packet Filtering with Linux eBPF. *Proceedings of the IEEE/IFIP Network Operations and Management Symposium (NOMS)*.

Sysdig. (2016). Falco: Cloud-Native Runtime Security. *Sysdig Open Source*. https://falco.org/

Tang, C., et al. (2020). FBAR: Facebook's Automated Remediation System. *Proceedings of the ACM Symposium on Cloud Computing (SoCC)*.

Vieira, M. A., Castanho, M. S., Pacífico, R. D. G., Santos, E. R. S., Júnior, E. P. M. C., & Vieira, L. F. M. (2020). Fast Packet Processing with eBPF and XDP: Concepts, Code, Challenges, and Applications. *ACM Computing Surveys*, 53(1), Article 16.

Weiner, J. (2018). PSI — Pressure Stall Information. *Linux Kernel Documentation*. https://www.kernel.org/doc/html/latest/accounting/psi.html

Welford, B. P. (1962). Note on a Method for Calculating Corrected Sums of Squares and Products. *Technometrics*, 4(3), 419–420.

---

*End of Whitepaper — Version 2.1*
