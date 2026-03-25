// Package motor implementa o sistema de resposta graduada do HOSA —
// o "arco reflexo" que age sobre os processos monitorados quando o
// Córtex Preditivo detecta estresse.
//
// Os 4 níveis de contenção mapeiam diretamente para os AlertLevels do brain:
//
//	Nível 0 (Homeostase)  → sem ação, limites removidos
//	Nível 1 (Vigilância)  → sem ação motora, apenas monitoramento intensificado
//	Nível 2 (Contenção)   → memory.high reduzido para throttling progressivo
//	Nível 3 (Proteção)    → memory.max aplicado + sinal ao processo
//
// Referência: whitepaper HOSA, Seção 5 — Arco Reflexo e Resposta Graduada.
package motor

import (
	"fmt"

	"github.com/bricio-sr/hosa/internal/syscgroup"
)

// ContainmentLevel represents the containment levels applied by the motor.
// Mirrors brain.AlertLevel — defined here to avoid circular dependency.
type ContainmentLevel int

const (
	LevelHomeostasis ContainmentLevel = 0
	LevelVigilance   ContainmentLevel = 1
	LevelContainment ContainmentLevel = 2
	LevelProtection  ContainmentLevel = 3
)

const (
	fractionContainment    = 0.75
	fractionProtectionHigh = 0.50
	fractionProtectionMax  = 0.90
)

// CgroupMotor applies containment actions on a cgroup v2 path.
type CgroupMotor struct {
	cgPath    string
	lastLevel ContainmentLevel
}

// NewCgroupMotor initializes the motor for an existing cgroup.
func NewCgroupMotor(cgPath string) *CgroupMotor {
	return &CgroupMotor{cgPath: cgPath}
}

// Apply executes the action for the given containment level.
// Returns (changed bool, error) — changed=true when an action was taken.
// Only acts when the level changes; no-ops on repeated same-level calls.
func (m *CgroupMotor) Apply(level ContainmentLevel, memTotalBytes uint64) (bool, error) {
	if level == m.lastLevel {
		return false, nil
	}
	m.lastLevel = level

	switch level {
	case LevelHomeostasis, LevelVigilance:
		return true, m.release()
	case LevelContainment:
		return true, m.contain(memTotalBytes)
	case LevelProtection:
		return true, m.protect(memTotalBytes)
	default:
		return false, fmt.Errorf("CgroupMotor.Apply: unknown level %d", level)
	}
}

// ActionSummary returns a human-readable summary of the action for the given level.
func ActionSummary(level ContainmentLevel, memTotalBytes uint64) string {
	switch level {
	case LevelHomeostasis, LevelVigilance:
		return "limits_removed"
	case LevelContainment:
		high := uint64(float64(memTotalBytes) * fractionContainment)
		return fmt.Sprintf("memory.high=%dMB", high/(1<<20))
	case LevelProtection:
		high := uint64(float64(memTotalBytes) * fractionProtectionHigh)
		max := uint64(float64(memTotalBytes) * fractionProtectionMax)
		return fmt.Sprintf("memory.high=%dMB memory.max=%dMB", high/(1<<20), max/(1<<20))
	default:
		return "unknown"
	}
}

func (m *CgroupMotor) release() error {
	if err := syscgroup.SetMemoryHigh(m.cgPath, 0); err != nil {
		return fmt.Errorf("motor.release: %w", err)
	}
	if err := syscgroup.SetMemoryMax(m.cgPath, 0); err != nil {
		return fmt.Errorf("motor.release: %w", err)
	}
	return nil
}

func (m *CgroupMotor) contain(memTotalBytes uint64) error {
	highLimit := uint64(float64(memTotalBytes) * fractionContainment)
	if err := syscgroup.SetMemoryHigh(m.cgPath, highLimit); err != nil {
		return fmt.Errorf("motor.contain: %w", err)
	}
	return nil
}

func (m *CgroupMotor) protect(memTotalBytes uint64) error {
	highLimit := uint64(float64(memTotalBytes) * fractionProtectionHigh)
	maxLimit := uint64(float64(memTotalBytes) * fractionProtectionMax)
	if err := syscgroup.SetMemoryHigh(m.cgPath, highLimit); err != nil {
		return fmt.Errorf("motor.protect: %w", err)
	}
	if err := syscgroup.SetMemoryMax(m.cgPath, maxLimit); err != nil {
		return fmt.Errorf("motor.protect: %w", err)
	}
	return nil
}

// CurrentMemory returns the current memory usage of the monitored cgroup.
func (m *CgroupMotor) CurrentMemory() (uint64, error) {
	return syscgroup.GetMemoryCurrent(m.cgPath)
}