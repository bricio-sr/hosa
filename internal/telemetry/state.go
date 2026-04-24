package telemetry

import (
	"sync"
	"time"
)

// Snapshot is a point-in-time view of the agent's stress state.
// Written by the main loop, read by HTTP handlers and the webhook client.
type Snapshot struct {
	DM        float64   // smoothed Mahalanobis distance D̄_M
	DMDot     float64   // dD̄_M/dt — rate of change per second
	Level     int       // brain.AlertLevel cast to int (0=homeostasis … 4=survival)
	HFragNorm float64   // normalized memory fragmentation entropy [0, 1]; 0 if Phase 2 disabled
	StateVec  []float64 // raw sensor readings (copy of the last ring-buffer row)
	UpdatedAt time.Time
}

// AtomicState is a concurrency-safe holder for the current Snapshot.
// Single writer (main loop), multiple readers (HTTP handlers, webhook).
type AtomicState struct {
	mu sync.RWMutex
	s  Snapshot
}

func (a *AtomicState) Set(s Snapshot) {
	a.mu.Lock()
	a.s = s
	a.mu.Unlock()
}

func (a *AtomicState) Get() Snapshot {
	a.mu.RLock()
	s := a.s
	a.mu.RUnlock()
	return s
}
