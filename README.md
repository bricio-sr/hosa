# HOSA
Homeostasis Operating System Agent - eBPF powered autonomous resilience

## 📂 Project Structure

```text
hosa/
├── cmd/
│   └── hosa/
│       └── main.go           # The entry point. Where the magic happens; initializes the agent.
├── internal/                 # Private agent code (cannot be imported by external packages)
│   ├── sysbpf/               
│   │   └── syscall.go        # Custom eBPF loader via native syscalls.
│   ├── linalg/               
│   │   └── matrix.go         # Custom Linear Algebra library (Matrices, Inversion, Covariance).
│   ├── syscgroup/            
│   │   └── file_edit.go      # Direct file manipulation within the Linux VFS.
│   ├── bpf/                  
│   │   ├── sensors.c         # Pure C eBPF code to be injected into the Kernel.
│   │   └── bpf_bpfeb.go      # Auto-generated files by Cilium/ebpf for Go-to-C communication.
│   ├── sensor/               # The "Sensory System"
│   │   └── collector.go      # Reads eBPF maps and structures raw data (Memory, CPU, I/O).
│   ├── brain/                # The "Predictive Cortex" (The Math)
│   │   ├── matrix.go         # Covariance Matrix manipulation.
│   │   ├── mahalanobis.go    # Homeostasis calculation (Mahalanobis Distance).
│   │   └── predictor.go      # Calculates the derivative and estimates Time of Failure (Tf).
│   ├── motor/                # The "Reflex Arc" (Actuators)
│   │   ├── cgroups.go        # Logic for PID throttling via Cgroups v2.
│   │   └── signals.go        # Logic for sending SIGTERM/SIGKILL if necessary.
│   └── state/                # The "Limbic System"
│       └── memory.go         # Stores short-term history in memory (buffer/ring) for mathematical basis.
├── docs/                     # Detailed documentation
│   ├── architecture.md       # Explanation of the Autonomous Nervous System inspiration.
│   └── math_model.md         # Documentation of mathematical formulas for the Thesis/Master's project.
├── go.mod
├── go.sum
└── Makefile                  # To compile eBPF C code and Go with a single command (e.g., make build).