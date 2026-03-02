# HOSA
Homeostasis Operating System Agent - eBPF powered autonomous resilience

## 📂 Project Structure

```text
hosa/
├── cmd/hosa/              # Entry point: Initializes the agent
├── internal/              # Private agent logic
│   ├── sysbpf/            # Native eBPF loader (syscalls)
│   ├── linalg/            # Linear Algebra library (Matrices, Covariance)
│   ├── bpf/               # eBPF C code and Go bindings
│   ├── sensor/            # "Sensory System": Data collection (CPU, RAM, I/O)
│   ├── brain/             # "Predictive Cortex": Mahalanobis & Math models
│   ├── motor/             # "Reflex Arc": Actuators (Cgroups, Signals)
│   └── state/             # "Limbic System": Short-term memory buffer
├── docs/                  # Architecture and Math documentation
└── Makefile               # Build automation