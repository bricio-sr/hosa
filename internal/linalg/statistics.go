package linalg

import "errors"

// MeanVector calcula o vetor de médias (matriz coluna) a partir de uma matriz de amostras.
// As linhas de 'samples' são as observações no tempo, e as colunas são as variáveis (CPU, RAM, I/O).
func MeanVector(samples *Matrix) *Matrix {
	vars := samples.Cols
	n := samples.Rows

	mean := NewMatrix(vars, 1) // Vetor coluna
	for j := 0; j < vars; j++ {
		var sum float64
		for i := 0; i < n; i++ {
			sum += samples.Get(i, j)
		}
		mean.Set(j, 0, sum/float64(n))
	}
	return mean
}

// CovarianceMatrix calcula a Matriz de Covariância a partir das amostras.
func CovarianceMatrix(samples *Matrix) (*Matrix, error) {
	n := samples.Rows
	if n <= 1 {
		return nil, errors.New("são necessárias pelo menos 2 amostras para calcular covariância")
	}

	mean := MeanVector(samples)
	vars := samples.Cols

	// Passo 1: Matriz Centralizada (D_c) = Amostras - Média
	centered := NewMatrix(n, vars)
	for i := 0; i < n; i++ {
		for j := 0; j < vars; j++ {
			centered.Set(i, j, samples.Get(i, j)-mean.Get(j, 0))
		}
	}

	// Passo 2: D_c^T (Transposta da Centralizada)
	centeredT := centered.Transpose()

	// Passo 3: Multiplica (D_c^T * D_c)
	covSum, err := centeredT.Mul(centered)
	if err != nil {
		return nil, err
	}

	// Passo 4: Divide por (N - 1) para covariância amostral
	cov := NewMatrix(vars, vars)
	divisor := float64(n - 1)
	for i := 0; i < vars; i++ {
		for j := 0; j < vars; j++ {
			cov.Set(i, j, covSum.Get(i, j)/divisor)
		}
	}

	return cov, nil
}