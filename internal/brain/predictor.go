package brain

import (
	"time"

	"github.com/bricio-sr/hosa/internal/linalg"
	"github.com/bricio-sr/hosa/internal/state"
)

// AlertLevel representa os 5 níveis de resposta do HOSA,
// inspirados no sistema nervoso humano:
//   - Fase 1 (arco reflexo): Homeostase → Vigilância → Contenção → Proteção
//   - Fase 2 (sistema nervoso simpático): Sobrevivência
type AlertLevel int

const (
	LevelHomeostasis AlertLevel = iota
	LevelVigilance
	LevelContainment
	LevelProtection
	LevelSurvival // Fase 2: intervenção física — escalonador de sobrevivência + termodinâmica de memória
)

// Limiares de D_M para escalonamento de nível.
// Com p=4 variáveis, o D_M esperado em homeostase segue χ²(4):
// média ≈ 2.0, std ≈ 2.0. Limiares calibrados empiricamente para
// separar ruído normal de estresse real com 4 probes.
const (
	ThresholdVigilance   = 3.5
	ThresholdContainment = 5.5
	ThresholdProtection  = 8.0
	ThresholdSurvival    = 12.0 // Fase 2: cascata iminente — substituição física do escalonador
)

// Limiares de dD̄_M/dt — sobre o D_M já suavizado pelo EWMA.
const (
	ThresholdDerivativeEscalate = 2.0
	ThresholdDerivativeRelax    = 0.5
)

// hysteresisDown define quantos ciclos consecutivos abaixo do limiar são
// necessários para desescalar um nível. Evita flapping em bordas de limiar.
const hysteresisDown = 5

// PredictorConfig defines the parameters of the Predictive Cortex.
type PredictorConfig struct {
	MinSamples                  int
	Alpha                       float64
	ThresholdVigilance          float64
	ThresholdContainment        float64
	ThresholdProtection         float64
	ThresholdSurvival           float64 // Fase 2: limiar para escalonador de sobrevivência
	ThresholdDerivativeEscalate float64
	ThresholdDerivativeRelax    float64
	HysteresisDown              int
}

// DefaultConfig returns the recommended default configuration.
func DefaultConfig() PredictorConfig {
	return PredictorConfig{
		MinSamples:                  30,
		Alpha:                       0.2,
		ThresholdVigilance:          ThresholdVigilance,
		ThresholdContainment:        ThresholdContainment,
		ThresholdProtection:         ThresholdProtection,
		ThresholdSurvival:           ThresholdSurvival,
		ThresholdDerivativeEscalate: ThresholdDerivativeEscalate,
		ThresholdDerivativeRelax:    ThresholdDerivativeRelax,
		HysteresisDown:              hysteresisDown,
	}
}

// StressReading é o resultado de um ciclo de análise do Córtex.
type StressReading struct {
	// DM é a Distância de Mahalanobis — magnitude do desvio do basal.
	DM float64

	// DMDot é dD_M/dt — taxa de variação do estresse por segundo.
	// Positivo = estresse crescendo, negativo = sistema se recuperando.
	DMDot float64

	// Level é o AlertLevel resultante após aplicar magnitude + derivada + histerese.
	Level AlertLevel

	// Timestamp é o instante desta leitura — usado para calcular dt.
	Timestamp time.Time
}

// PredictiveCortex é o "cérebro" do HOSA.
// Além de calcular D_M, aplica EWMA para suavização, rastreia dD̄_M/dt
// e aplica histerese para evitar oscilação entre níveis.
//
// Pipeline de sinais:
//   D_M(t) bruto → EWMA → D̄_M(t) suavizado → dD̄_M/dt → classify()
type PredictiveCortex struct {
	buffer  *state.RingBuffer
	config  PredictorConfig
	welford *WelfordState

	// EWMA: D̄_M(t) = α·D_M(t) + (1-α)·D̄_M(t-1)
	ewmaValue    float64 // valor suavizado atual
	ewmaReady    bool    // false até a primeira amostra válida

	// Estado interno entre ciclos
	prev         *StressReading
	currentLevel AlertLevel
	belowCount   int
}

// NewPredictiveCortex inicializa o Córtex com um buffer e configuração.
func NewPredictiveCortex(buf *state.RingBuffer, cfg PredictorConfig) *PredictiveCortex {
	snap := buf.Snapshot()
	vars := snap.Cols
	if vars == 0 {
		vars = 1
	}

	alpha := cfg.Alpha
	if alpha <= 0 || alpha > 1 {
		alpha = DefaultConfig().Alpha
	}
	cfg.Alpha = alpha

	return &PredictiveCortex{
		buffer:       buf,
		config:       cfg,
		welford:      NewWelfordState(vars),
		currentLevel: LevelHomeostasis,
	}
}

// Analyze é o método principal do Córtex.
// Incorpora a leitura mais recente ao WelfordState e retorna:
// stress (D_M), dmDot (dD_M/dt) e o AlertLevel resultante.
func (pc *PredictiveCortex) Analyze() (stress float64, dmDot float64, level AlertLevel, err error) {
	if !pc.buffer.IsReady() {
		return 0, 0, LevelHomeostasis, nil
	}

	snap := pc.buffer.Snapshot()
	if snap.Rows < 2 {
		return 0, 0, LevelHomeostasis, nil
	}

	// --- Atualiza o basal incremental com a leitura mais recente ---
	lastRow := snap.Rows - 1
	reading := make([]float64, snap.Cols)
	for j := 0; j < snap.Cols; j++ {
		reading[j] = snap.Get(lastRow, j)
	}
	if err = pc.welford.Update(reading); err != nil {
		return 0, 0, LevelHomeostasis, err
	}

	// Aguarda o mínimo de amostras para análise confiável
	if !pc.welford.IsReady(pc.config.MinSamples) {
		return 0, 0, LevelHomeostasis, nil
	}

	// --- Obtém o basal atual do WelfordState (O(p²), não O(n·p²)) ---
	mean := pc.welford.Mean()

	cov, err := pc.welford.Covariance()
	if err != nil {
		return 0, 0, LevelHomeostasis, err
	}

	invCov, err := cov.Inverse()
	if err != nil {
		// Matriz singular: variáveis sem variância ainda (cold-start).
		return 0, 0, LevelHomeostasis, nil
	}

	model := NewHomeostasisModel(mean, invCov)

	// --- Avalia o estado atual contra o basal ---
	current := linalg.NewMatrix(snap.Cols, 1)
	for j := 0; j < snap.Cols; j++ {
		current.Set(j, 0, reading[j])
	}

	dm, err := model.CalculateStress(current)
	if err != nil {
		return 0, 0, LevelHomeostasis, err
	}

	now := time.Now()

	// --- Aplica EWMA: D̄_M(t) = α·D_M(t) + (1-α)·D̄_M(t-1) ---
	// Na primeira amostra válida, inicializa o EWMA com o valor bruto
	// para evitar o transitório de arranque.
	if !pc.ewmaReady {
		pc.ewmaValue = dm
		pc.ewmaReady = true
	} else {
		pc.ewmaValue = pc.config.Alpha*dm + (1-pc.config.Alpha)*pc.ewmaValue
	}
	smoothedDM := pc.ewmaValue

	// --- Calcula dD̄_M/dt sobre o sinal suavizado ---
	dmDot = pc.calcDerivative(smoothedDM, now)

	// --- Classifica com magnitude suavizada + derivada + histerese ---
	level = pc.classify(smoothedDM, dmDot)

	// Persiste usando o D_M suavizado como referência para o próximo ciclo
	pc.prev = &StressReading{DM: smoothedDM, DMDot: dmDot, Level: level, Timestamp: now}

	return smoothedDM, dmDot, level, nil
}

// calcDerivative calcula dD_M/dt usando diferença finita entre o ciclo atual e o anterior.
// Retorna 0 no primeiro ciclo (sem referência anterior).
func (pc *PredictiveCortex) calcDerivative(dm float64, now time.Time) float64 {
	if pc.prev == nil {
		return 0
	}

	dt := now.Sub(pc.prev.Timestamp).Seconds()
	if dt <= 0 {
		return 0
	}

	return (dm - pc.prev.DM) / dt
}

// classify determina o AlertLevel combinando magnitude (D_M), derivada (dD_M/dt)
// e histerese. Implementa as regras do whitepaper Seção 5.3:
//
//  1. Calcula o nível base pela magnitude de D_M.
//  2. Se dD_M/dt indica aceleração, escala um nível acima.
//  3. Aplica histerese na descida: só desce após hysteresisDown ciclos consecutivos abaixo.
func (pc *PredictiveCortex) classify(dm, dmDot float64) AlertLevel {
	cfg := pc.config

	// Resolve thresholds — use config values if set, fall back to package constants
	thVigilance := cfg.ThresholdVigilance
	if thVigilance <= 0 {
		thVigilance = ThresholdVigilance
	}
	thContainment := cfg.ThresholdContainment
	if thContainment <= 0 {
		thContainment = ThresholdContainment
	}
	thProtection := cfg.ThresholdProtection
	if thProtection <= 0 {
		thProtection = ThresholdProtection
	}
	thSurvival := cfg.ThresholdSurvival
	if thSurvival <= 0 {
		thSurvival = ThresholdSurvival
	}
	thDerivEscalate := cfg.ThresholdDerivativeEscalate
	if thDerivEscalate <= 0 {
		thDerivEscalate = ThresholdDerivativeEscalate
	}
	thDerivRelax := cfg.ThresholdDerivativeRelax
	if thDerivRelax <= 0 {
		thDerivRelax = ThresholdDerivativeRelax
	}
	hystDown := cfg.HysteresisDown
	if hystDown <= 0 {
		hystDown = hysteresisDown
	}

	// Step 1: level by magnitude
	var baseLevel AlertLevel
	switch {
	case dm >= thSurvival:
		baseLevel = LevelSurvival
	case dm >= thProtection:
		baseLevel = LevelProtection
	case dm >= thContainment:
		baseLevel = LevelContainment
	case dm >= thVigilance:
		baseLevel = LevelVigilance
	default:
		baseLevel = LevelHomeostasis
	}

	// Step 2: derivative escalation — applies up to LevelSurvival
	candidateLevel := baseLevel
	if dmDot > thDerivEscalate && baseLevel < LevelSurvival {
		candidateLevel = baseLevel + 1
	}

	// Step 3: hysteresis on descent
	if candidateLevel < pc.currentLevel {
		if dmDot < thDerivRelax {
			pc.belowCount++
		} else {
			pc.belowCount = 0
		}
		if pc.belowCount >= hystDown {
			pc.belowCount = 0
			pc.currentLevel = candidateLevel
		}
	} else {
		pc.belowCount = 0
		pc.currentLevel = candidateLevel
	}

	return pc.currentLevel
}

// classifyByMagnitude mapeia D_M para AlertLevel pelos limiares do whitepaper.
func classifyByMagnitude(dm float64) AlertLevel {
	switch {
	case dm >= ThresholdSurvival:
		return LevelSurvival
	case dm >= ThresholdProtection:
		return LevelProtection
	case dm >= ThresholdContainment:
		return LevelContainment
	case dm >= ThresholdVigilance:
		return LevelVigilance
	default:
		return LevelHomeostasis
	}
}