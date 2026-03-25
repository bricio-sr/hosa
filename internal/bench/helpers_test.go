package bench

import (
	"time"

	"github.com/bricio-sr/hosa/internal/linalg"
)

// Index aliases matching sensor.NumVars layout — used in fault injection.
// Must stay in sync with sensor/collector.go constants.
const (
	idxCPURunQueue   = 0
	idxMemBrkCalls   = 1
	idxMemPageFaults = 2
	idxIOBlockOps    = 3
)

// normalInterval is the homeostasis sampling interval — used in ms_to_detect metric.
const normalInterval = time.Second

// makeSampleMatrix creates a (vars x 1) column matrix with a constant value.
func makeSampleMatrix(vars int, value float64) *linalg.Matrix {
	m := linalg.NewMatrix(vars, 1)
	for j := 0; j < vars; j++ {
		m.Set(j, 0, value)
	}
	return m
}