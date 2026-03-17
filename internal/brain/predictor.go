package brain

import (
	"time"

	"github.com/bricio-sr/hosa/internal/linalg"
	"github.com/bricio-sr/hosa/internal/state"
)

// AlertLevel representa os 4 níveis de resposta do HOSA,
// inspirados no arco reflexo do sistema nervoso humano.
type AlertLevel int

const (
	LevelHomeostasis AlertLevel = iota
	LevelVigilance
	LevelContainment
	LevelProtection
)

// Limiares de D_M para escalonamento de nível (whitepaper Seção 5.2).
const (
	ThresholdVigilance   = 2.5
	ThresholdContainment = 4.5
	ThresholdProtection  = 7.0
)

// Limiares de dD_M/dt — taxa de variação do estresse por segundo.
// Um valor alto aqui significa que o sistema está se deteriorando rapidamente,
// mesmo que D_M ainda não tenha cruzado o limiar de magnitude.
const (
	// ThresholdDerivativeEscalate: se dD_M/dt > este valor, sobe um nível extra.
	// Calibrado para detectar memory leaks agressivos (~50MB/s conforme whitepaper Fig. 1).
	ThresholdDerivativeEscalate = 1.5

	// ThresholdDerivativeRelax: derivada abaixo deste valor permite desescalada.
	// Histerese: mais baixo que o de escalada para evitar oscilação.
	ThresholdDerivativeRelax = 0.3
)

// hysteresisDown define quantos ciclos consecutivos abaixo do limiar são
// necessários para desescalar um nível. Evita flapping em bordas de limiar.
const hysteresisDown = 3

// PredictorConfig define os parâmetros do Córtex Preditivo.
type PredictorConfig struct {
	// MinSamples é o mínimo de amostras no RingBuffer antes de habilitar predições.
	MinSamples int
}

// DefaultConfig retorna a configuração padrão recomendada pelo whitepaper.
func DefaultConfig() PredictorConfig {
	return PredictorConfig{
		MinSamples: 30,
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
// Além de calcular D_M, rastreia sua derivada temporal dD_M/dt e aplica
// histerese para evitar oscilação entre níveis.
type PredictiveCortex struct {
	buffer *state.RingBuffer
	config PredictorConfig

	// Estado interno entre ciclos
	prev          *StressReading // última leitura válida
	currentLevel  AlertLevel     // nível atual (com histerese)
	belowCount    int            // ciclos consecutivos abaixo do limiar atual
}

// NewPredictiveCortex inicializa o Córtex com um buffer e configuração.
func NewPredictiveCortex(buf *state.RingBuffer, cfg PredictorConfig) *PredictiveCortex {
	return &PredictiveCortex{
		buffer:       buf,
		config:       cfg,
		currentLevel: LevelHomeostasis,
	}
}

// Analyze é o método principal do Córtex.
// Retorna stress (D_M), dmDot (dD_M/dt) e o AlertLevel resultante.
func (pc *PredictiveCortex) Analyze() (stress float64, dmDot float64, level AlertLevel, err error) {
	if !pc.buffer.IsReady() {
		return 0, 0, LevelHomeostasis, nil
	}

	snap := pc.buffer.Snapshot()
	if snap.Rows < pc.config.MinSamples {
		return 0, 0, LevelHomeostasis, nil
	}

	// --- Aprende o basal ---
	mean := linalg.MeanVector(snap)

	cov, err := linalg.CovarianceMatrix(snap)
	if err != nil {
		return 0, 0, LevelHomeostasis, err
	}

	invCov, err := cov.Inverse()
	if err != nil {
		// Matriz singular: cold-start ou variáveis sem variância ainda.
		return 0, 0, LevelHomeostasis, nil
	}

	model := NewHomeostasisModel(mean, invCov)

	// --- Avalia o estado atual (última linha do snapshot) ---
	vars := snap.Cols
	current := linalg.NewMatrix(vars, 1)
	lastRow := snap.Rows - 1
	for j := 0; j < vars; j++ {
		current.Set(j, 0, snap.Get(lastRow, j))
	}

	dm, err := model.CalculateStress(current)
	if err != nil {
		return 0, 0, LevelHomeostasis, err
	}

	now := time.Now()

	// --- Calcula dD_M/dt ---
	dmDot = pc.calcDerivative(dm, now)

	// --- Classifica com magnitude + derivada + histerese ---
	level = pc.classify(dm, dmDot)

	// Persiste a leitura para o próximo ciclo
	pc.prev = &StressReading{DM: dm, DMDot: dmDot, Level: level, Timestamp: now}

	return dm, dmDot, level, nil
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
	// Passo 1: nível pela magnitude
	baseLevel := classifyByMagnitude(dm)

	// Passo 2: escalada por derivada — aceleração alta sobe um nível
	candidateLevel := baseLevel
	if dmDot > ThresholdDerivativeEscalate && baseLevel < LevelProtection {
		candidateLevel = baseLevel + 1
	}

	// Passo 3: histerese na descida
	if candidateLevel < pc.currentLevel {
		// Sistema melhorando — só desce após N ciclos consecutivos
		if dmDot < ThresholdDerivativeRelax {
			pc.belowCount++
		} else {
			pc.belowCount = 0 // derivada ainda alta: reseta o contador
		}

		if pc.belowCount >= hysteresisDown {
			pc.belowCount = 0
			pc.currentLevel = candidateLevel
		}
		// Enquanto não atingir hysteresisDown, mantém o nível atual
	} else {
		// Sistema estável ou piorando — aplica imediatamente, sem atraso
		pc.belowCount = 0
		pc.currentLevel = candidateLevel
	}

	return pc.currentLevel
}

// classifyByMagnitude mapeia D_M para AlertLevel pelos limiares do whitepaper.
func classifyByMagnitude(dm float64) AlertLevel {
	switch {
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