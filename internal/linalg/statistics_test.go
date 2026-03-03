package linalg

import (
	"testing"
)

func TestMeanVector(t *testing.T) {
	// Vamos simular 3 leituras (linhas) de 2 métricas (CPU e Memória - colunas)
	samples := NewMatrix(3, 2)
	samples.Set(0, 0, 2.0) // Leitura 1: CPU
	samples.Set(0, 1, 3.0) // Leitura 1: Mem
	samples.Set(1, 0, 4.0) // Leitura 2: CPU
	samples.Set(1, 1, 7.0) // Leitura 2: Mem
	samples.Set(2, 0, 6.0) // Leitura 3: CPU
	samples.Set(2, 1, 5.0) // Leitura 3: Mem

	// Média esperada:
	// CPU: (2 + 4 + 6) / 3 = 4.0
	// Mem: (3 + 7 + 5) / 3 = 5.0
	mean := MeanVector(samples)

	if !almostEqual(mean.Get(0, 0), 4.0) {
		t.Errorf("Média da CPU errada. Esperado 4.0, obtido %f", mean.Get(0, 0))
	}
	if !almostEqual(mean.Get(1, 0), 5.0) {
		t.Errorf("Média da Memória errada. Esperado 5.0, obtido %f", mean.Get(1, 0))
	}
}

func TestCovarianceMatrix(t *testing.T) {
	// Usando as mesmas amostras do teste acima
	samples := NewMatrix(3, 2)
	samples.Set(0, 0, 2.0)
	samples.Set(0, 1, 3.0)
	samples.Set(1, 0, 4.0)
	samples.Set(1, 1, 7.0)
	samples.Set(2, 0, 6.0)
	samples.Set(2, 1, 5.0)

	// O cálculo manual da matriz de covariância (N=3):
	// Matriz Centralizada (Amostra - Média):
	// [-2.0, -2.0]
	// [ 0.0,  2.0]
	// [ 2.0,  0.0]
	//
	// Covariância = (Transposta * Centralizada) / (N - 1)
	// Var(CPU) = ((-2)^2 + 0^2 + 2^2) / 2 = 8 / 2 = 4.0
	// Var(Mem) = ((-2)^2 + 2^2 + 0^2) / 2 = 8 / 2 = 4.0
	// Cov(CPU, Mem) = ((-2*-2) + (0*2) + (2*0)) / 2 = 4 / 2 = 2.0

	cov, err := CovarianceMatrix(samples)
	if err != nil {
		t.Fatalf("Erro inesperado: %v", err)
	}

	expected := []float64{
		4.0, 2.0,
		2.0, 4.0,
	}

	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			val := cov.Get(i, j)
			exp := expected[i*2+j]
			if !almostEqual(val, exp) {
				t.Errorf("Erro na Covariância [%d,%d]: esperado %f, obtido %f", i, j, exp, val)
			}
		}
	}
}