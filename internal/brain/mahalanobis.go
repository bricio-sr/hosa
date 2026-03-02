package brain

import (
	"errors"
	"math"

	"github.com/bricio-sr/hosa/internal/linalg"
)

// HomeostasisModel guarda a "memória genética" do servidor.
// Ele sabe o que é o normal (MeanVector) e como os recursos interagem (InverseCovariance).
type HomeostasisModel struct {
	MeanVector        *linalg.Matrix // mu: matriz coluna (ex: 3x1 para CPU, Mem, I/O)
	InverseCovariance *linalg.Matrix // sigma^-1: matriz quadrada (ex: 3x3)
}

// NewHomeostasisModel inicializa o modelo de estresse.
func NewHomeostasisModel(mean *linalg.Matrix, invCov *linalg.Matrix) *HomeostasisModel {
	return &HomeostasisModel{
		MeanVector:        mean,
		InverseCovariance: invCov,
	}
}

// CalculateStress é o cálculo da Distância de Mahalanobis.
// Retorna um float64 representando o quão "doente" o servidor está neste milissegundo.
func (h *HomeostasisModel) CalculateStress(current *linalg.Matrix) (float64, error) {
	// Passo 1: Calcula o desvio da média (X - mu)
	diff, err := current.Sub(h.MeanVector)
	if err != nil {
		return 0, errors.New("falha ao calcular desvio: dimensões do vetor atual não batem com a média histórica")
	}

	// Passo 2: Transpõe o vetor de desvio (X - mu)^T
	diffT := diff.Transpose()

	// Passo 3: Multiplica pela matriz de covariância inversa: (X - mu)^T * Sigma^-1
	step3, err := diffT.Mul(h.InverseCovariance)
	if err != nil {
		return 0, err
	}

	// Passo 4: Multiplica pelo desvio original: [(X - mu)^T * Sigma^-1] * (X - mu)
	step4, err := step3.Mul(diff)
	if err != nil {
		return 0, err
	}

	// O resultado final no step4 é uma matriz 1x1. Vamos extrair esse único valor.
	stressSquared := step4.Get(0, 0)

	// Prevenção de instabilidade de ponto flutuante (float64 precision)
	// Às vezes, um zero matemático vira -0.000000000001, e raiz de negativo dá NaN.
	if stressSquared < 0 {
		stressSquared = 0
	}

	// Passo 5: Tira a raiz quadrada para finalizar a Distância de Mahalanobis
	return math.Sqrt(stressSquared), nil
}