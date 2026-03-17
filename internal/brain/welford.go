package brain

import (
	"errors"
	"math"

	"github.com/bricio-sr/hosa/internal/linalg"
)

// WelfordState mantém o estado incremental para cálculo online de média
// e covariância multivariável usando o algoritmo de Welford (1962).
//
// A cada nova amostra, a atualização custa O(p²) — onde p é o número de
// variáveis — independente de quantas amostras já foram processadas.
// Isso contrasta com o cálculo batch que custa O(n·p²) por ciclo.
//
// Referência: Welford, B.P. (1962). "Note on a method for calculating
// corrected sums of squares and products." Technometrics, 4(3), 419-420.
//
// Para a covariância multivariável online, usamos a generalização de
// Welford descrita em Chan et al. (1979) e West (1979):
//
//	delta  = x - mean_prev
//	mean  += delta / n
//	delta2 = x - mean_new
//	M2    += outer(delta, delta2)   ← soma externa dos dois deltas
//	cov    = M2 / (n - 1)           ← covariância amostral
type WelfordState struct {
	n    int              // número de amostras processadas
	p    int              // número de variáveis (dimensão)
	mean []float64        // vetor de médias incrementais (p × 1)
	m2   []float64        // matriz de somas de produtos cruzados (p × p, row-major)
}

// NewWelfordState inicializa o estado para p variáveis.
func NewWelfordState(p int) *WelfordState {
	return &WelfordState{
		p:    p,
		mean: make([]float64, p),
		m2:   make([]float64, p*p),
	}
}

// Update incorpora uma nova observação ao estado incremental.
// reading deve ter exatamente p elementos.
func (w *WelfordState) Update(reading []float64) error {
	if len(reading) != w.p {
		return errors.New("WelfordState.Update: dimensão da leitura incompatível")
	}

	w.n++
	n := float64(w.n)

	// delta1[j] = x[j] - mean_prev[j]
	delta1 := make([]float64, w.p)
	for j := 0; j < w.p; j++ {
		delta1[j] = reading[j] - w.mean[j]
	}

	// Atualiza a média: mean += delta1 / n
	for j := 0; j < w.p; j++ {
		w.mean[j] += delta1[j] / n
	}

	// delta2[j] = x[j] - mean_new[j]
	delta2 := make([]float64, w.p)
	for j := 0; j < w.p; j++ {
		delta2[j] = reading[j] - w.mean[j]
	}

	// M2[i][j] += delta1[i] * delta2[j]   (produto externo dos dois deltas)
	for i := 0; i < w.p; i++ {
		for j := 0; j < w.p; j++ {
			w.m2[i*w.p+j] += delta1[i] * delta2[j]
		}
	}

	return nil
}

// Count retorna o número de amostras processadas.
func (w *WelfordState) Count() int {
	return w.n
}

// Mean retorna o vetor de médias atual como matriz coluna (p × 1).
func (w *WelfordState) Mean() *linalg.Matrix {
	m := linalg.NewMatrix(w.p, 1)
	for j := 0; j < w.p; j++ {
		m.Set(j, 0, w.mean[j])
	}
	return m
}

// Covariance retorna a matriz de covariância amostral atual (p × p).
// Requer n >= 2; retorna erro caso contrário.
func (w *WelfordState) Covariance() (*linalg.Matrix, error) {
	if w.n < 2 {
		return nil, errors.New("WelfordState.Covariance: necessário n >= 2")
	}

	divisor := float64(w.n - 1)
	cov := linalg.NewMatrix(w.p, w.p)
	for i := 0; i < w.p; i++ {
		for j := 0; j < w.p; j++ {
			cov.Set(i, j, w.m2[i*w.p+j]/divisor)
		}
	}
	return cov, nil
}

// IsReady retorna true quando há amostras suficientes para uma análise confiável.
func (w *WelfordState) IsReady(minSamples int) bool {
	return w.n >= minSamples
}

// StdDev retorna o desvio padrão de cada variável (raiz da diagonal da covariância).
// Útil para monitorar se o basal está sendo aprendido corretamente.
func (w *WelfordState) StdDev() []float64 {
	if w.n < 2 {
		return make([]float64, w.p)
	}
	divisor := float64(w.n - 1)
	std := make([]float64, w.p)
	for j := 0; j < w.p; j++ {
		variance := w.m2[j*w.p+j] / divisor
		if variance > 0 {
			std[j] = math.Sqrt(variance)
		}
	}
	return std
}