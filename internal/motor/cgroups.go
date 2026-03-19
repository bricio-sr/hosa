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
	"log"

	"github.com/bricio-sr/hosa/internal/syscgroup"
)

// ContainmentLevel representa os níveis de contenção aplicáveis pelo motor.
// Espelha brain.AlertLevel — definido aqui para evitar dependência circular.
type ContainmentLevel int

const (
	LevelHomeostasis  ContainmentLevel = 0
	LevelVigilance    ContainmentLevel = 1
	LevelContainment  ContainmentLevel = 2
	LevelProtection   ContainmentLevel = 3
)

// MemoryFractions define os fatores aplicados sobre o limite total de memória
// do cgroup para cada nível de contenção.
// Ex: se o cgroup tem 4 GB e o nível é Contenção, memory.high = 4 GB * 0.75 = 3 GB.
const (
	// fractionContainment reduz memory.high para 75% — throttling gradual.
	fractionContainment = 0.75

	// fractionProtection reduz memory.high para 50% e aplica memory.max a 90%.
	fractionProtectionHigh = 0.50
	fractionProtectionMax  = 0.90
)

// CgroupMotor aplica ações de contenção em um cgroup v2 específico.
type CgroupMotor struct {
	cgPath   string           // caminho do cgroup monitorado
	lastLevel ContainmentLevel // último nível aplicado — evita logs repetidos
}

// NewCgroupMotor inicializa o motor para um cgroup já existente.
func NewCgroupMotor(cgPath string) *CgroupMotor {
	return &CgroupMotor{cgPath: cgPath}
}

// Apply executa a ação correspondente ao nível de contenção recebido.
// É o ponto de entrada chamado pelo loop principal (main.go → react()).
//
// memTotalBytes é o limite total de memória do cgroup (lido de memory.max antes
// de qualquer contenção, ou do total de RAM do host como fallback).
func (m *CgroupMotor) Apply(level ContainmentLevel, memTotalBytes uint64) error {
	// Só aplica e loga quando o nível muda — evita ruído a cada tick.
	if level == m.lastLevel {
		return nil
	}
	m.lastLevel = level

	switch level {

	case LevelHomeostasis, LevelVigilance:
		// Sem ação motora — remove qualquer contenção residual de ciclos anteriores.
		return m.release()

	case LevelContainment:
		return m.contain(memTotalBytes)

	case LevelProtection:
		return m.protect(memTotalBytes)

	default:
		return fmt.Errorf("CgroupMotor.Apply: nível desconhecido %d", level)
	}
}

// release remove os limites de memória, restaurando a homeostase do cgroup.
// Chamado quando o estresse cai abaixo do limiar de vigilância (histerese).
func (m *CgroupMotor) release() error {
	if err := syscgroup.SetMemoryHigh(m.cgPath, 0); err != nil {
		return fmt.Errorf("motor.release: %w", err)
	}
	if err := syscgroup.SetMemoryMax(m.cgPath, 0); err != nil {
		return fmt.Errorf("motor.release: %w", err)
	}

	log.Printf("HOSA Motor [HOMEOSTASE]: limites de memória removidos (%s)", m.cgPath)
	return nil
}

// contain aplica throttling suave via memory.high.
// O processo não é morto — o kernel aplica pressão de alocação progressiva,
// forçando o processo a liberar memória voluntariamente.
func (m *CgroupMotor) contain(memTotalBytes uint64) error {
	highLimit := uint64(float64(memTotalBytes) * fractionContainment)

	if err := syscgroup.SetMemoryHigh(m.cgPath, highLimit); err != nil {
		return fmt.Errorf("motor.contain: %w", err)
	}

	log.Printf("HOSA Motor [CONTENÇÃO]: memory.high = %d bytes (%.0f%% de %d) (%s)",
		highLimit, fractionContainment*100, memTotalBytes, m.cgPath)

	return nil
}

// protect aplica contenção severa: reduz memory.high a 50% e sela memory.max a 90%.
// O processo pode ser morto pelo OOM-Killer do cgroup se ultrapassar memory.max,
// mas o impacto fica isolado — o host é preservado.
func (m *CgroupMotor) protect(memTotalBytes uint64) error {
	highLimit := uint64(float64(memTotalBytes) * fractionProtectionHigh)
	maxLimit := uint64(float64(memTotalBytes) * fractionProtectionMax)

	if err := syscgroup.SetMemoryHigh(m.cgPath, highLimit); err != nil {
		return fmt.Errorf("motor.protect (memory.high): %w", err)
	}
	if err := syscgroup.SetMemoryMax(m.cgPath, maxLimit); err != nil {
		return fmt.Errorf("motor.protect (memory.max): %w", err)
	}

	log.Printf("HOSA Motor [PROTEÇÃO]: memory.high=%d memory.max=%d (%s)",
		highLimit, maxLimit, m.cgPath)

	return nil
}

// CurrentMemory retorna o uso de memória atual do cgroup monitorado.
// Conveniência para o loop principal ler o estado sem importar syscgroup diretamente.
func (m *CgroupMotor) CurrentMemory() (uint64, error) {
	return syscgroup.GetMemoryCurrent(m.cgPath)
}