package brain

import (
	"github.com/bricio-sr/hosa/internal/linalg"
	"github.com/bricio-sr/hosa/internal/state"
)

// AlertLevel representa os 4 níveis de resposta do HOSA,
// inspirados no arco reflexo do sistema nervoso humano.
type AlertLevel int

const (
	// LevelHomeostasis (0): O sistema está saudável. Nenhuma ação necessária.
	LevelHomeostasis AlertLevel = iota
	// LevelVigilance (1): Desvio detectado. Aumenta a frequência de amostragem.
	LevelVigilance
	// LevelContainment (2): Estresse confirmado. Aciona cgroups para conter o processo.
	LevelContainment
	// LevelProtection (3): Risco de colapso iminente. Ações drásticas de preservação do host.
	LevelProtection
)

// Thresholds são os limites de Distância de Mahalanobis para escalonamento de nível.
// Baseados no whitepaper: Seção 5.2 (Calibração do ICP).
const (
	ThresholdVigilance   = 2.5
	ThresholdContainment = 4.5
	ThresholdProtection  = 7.0
)

// PredictorConfig define os parâmetros do Córtex Preditivo.
type PredictorConfig struct {
	// MinSamples é o mínimo de amostras no RingBuffer antes de habilitar predições.
	// Abaixo disso, a covariância não é computável.
	MinSamples int
}

// DefaultConfig retorna a configuração padrão recomendada pelo whitepaper.
func DefaultConfig() PredictorConfig {
	return PredictorConfig{
		MinSamples: 30,
	}
}

// Predictivecortex é o "cérebro" do HOSA.
// Ele consome o RingBuffer (memória de curto prazo), calcula o estresse
// via Mahalanobis e decide o nível de alerta correspondente.
type PredictiveCortex struct {
	buffer *state.RingBuffer
	config PredictorConfig
}

// NewPredictiveCortex inicializa o Córtex com um buffer e configuração.
func NewPredictiveCortex(buf *state.RingBuffer, cfg PredictorConfig) *PredictiveCortex {
	return &PredictiveCortex{
		buffer: buf,
		config: cfg,
	}
}

// Analyze é o método principal do Córtex.
// Ele tira um snapshot do buffer, computa a média e covariância do histórico,
// e calcula o estresse do vetor mais recente em relação ao basal aprendido.
// Retorna o score de estresse e o AlertLevel correspondente.
func (pc *PredictiveCortex) Analyze() (stress float64, level AlertLevel, err error) {
	// Ainda não temos amostras suficientes para uma análise estatística confiável.
	if !pc.buffer.IsReady() {
		return 0, LevelHomeostasis, nil
	}

	snap := pc.buffer.Snapshot()

	// Precisamos de pelo menos MinSamples para que a covariância seja representativa.
	if snap.Rows < pc.config.MinSamples {
		return 0, LevelHomeostasis, nil
	}

	// --- Aprende o basal a partir do histórico completo do snapshot ---

	// Calcula o vetor de médias (mu): o que é "normal" para este servidor
	mean := linalg.MeanVector(snap)

	// Calcula a Matriz de Covariância amostral
	cov, err := linalg.CovarianceMatrix(snap)
	if err != nil {
		return 0, LevelHomeostasis, err
	}

	// Inverte a covariância para o cálculo de Mahalanobis
	invCov, err := cov.Inverse()
	if err != nil {
		// Matriz singular: variáveis perfeitamente correlacionadas ou sem variância.
		// Isso pode acontecer no cold-start. Retorna homeostase e aguarda mais dados.
		return 0, LevelHomeostasis, nil
	}

	// Monta o modelo de homeostase com o basal aprendido
	model := NewHomeostasisModel(mean, invCov)

	// --- Avalia o estado ATUAL (última linha do snapshot) ---

	vars := snap.Cols
	current := linalg.NewMatrix(vars, 1) // Vetor coluna
	lastRow := snap.Rows - 1
	for j := 0; j < vars; j++ {
		current.Set(j, 0, snap.Get(lastRow, j))
	}

	// Calcula a Distância de Mahalanobis: o "termômetro de estresse"
	stress, err = model.CalculateStress(current)
	if err != nil {
		return 0, LevelHomeostasis, err
	}

	level = classifyStress(stress)
	return stress, level, nil
}

// classifyStress mapeia um score de Mahalanobis para um AlertLevel.
// Os limiares são os definidos no whitepaper (Seção 5.2).
func classifyStress(d float64) AlertLevel {
	switch {
	case d >= ThresholdProtection:
		return LevelProtection
	case d >= ThresholdContainment:
		return LevelContainment
	case d >= ThresholdVigilance:
		return LevelVigilance
	default:
		return LevelHomeostasis
	}
}