package linalg

import (
	"math"
	"testing"
)

// epsilon é a nossa tolerância para erros de precisão de ponto flutuante.
const epsilon = 1e-9

// almostEqual verifica se dois float64 são praticamente iguais.
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= epsilon
}

func TestMatrixMultiplication(t *testing.T) {
	// Matriz A (2x3)
	a := NewMatrix(2, 3)
	a.Data = []float64{
		1, 2, 3,
		4, 5, 6,
	}

	// Matriz B (3x2)
	b := NewMatrix(3, 2)
	b.Data = []float64{
		7, 8,
		9, 1,
		2, 3,
	}

	// Resultado Esperado (2x2)
	// [ (1*7 + 2*9 + 3*2), (1*8 + 2*1 + 3*3) ] == [31, 19]
	// [ (4*7 + 5*9 + 6*2), (4*8 + 5*1 + 6*3) ] == [85, 55]

	result, err := a.Mul(b)
	if err != nil {
		t.Fatalf("Erro inesperado na multiplicação: %v", err)
	}

	expected := []float64{31, 19, 85, 55}

	for i := 0; i < len(expected); i++ {
		if !almostEqual(result.Data[i], expected[i]) {
			t.Errorf("Erro no índice %d: esperado %f, obtido %f", i, expected[i], result.Data[i])
		}
	}
}

func TestMatrixInverse(t *testing.T) {
	// Matriz M quadrada (3x3)
	m := NewMatrix(3, 3)
	m.Data = []float64{
		2, -1, 0,
		-1, 2, -1,
		0, -1, 2,
	}

	inv, err := m.Inverse()
	if err != nil {
		t.Fatalf("Erro inesperado ao inverter a matriz: %v", err)
	}

	// A prova real: M * M^-1 DEVE ser igual à Matriz Identidade
	identity, err := m.Mul(inv)
	if err != nil {
		t.Fatalf("Erro ao multiplicar pela inversa: %v", err)
	}

	// Verifica se a diagonal principal é 1 e o resto é 0
	for i := 0; i < identity.Rows; i++ {
		for j := 0; j < identity.Cols; j++ {
			val := identity.Get(i, j)
			if i == j {
				// Diagonal principal deve ser 1
				if !almostEqual(val, 1.0) {
					t.Errorf("Falha na diagonal da Identidade [%d,%d]: esperado 1.0, obtido %f", i, j, val)
				}
			} else {
				// Resto deve ser 0
				if !almostEqual(val, 0.0) {
					t.Errorf("Falha fora da diagonal [%d,%d]: esperado 0.0, obtido %f", i, j, val)
				}
			}
		}
	}
}

func TestMatrixSingular(t *testing.T) {
	// Matriz com uma linha dependente (determinante = 0)
	m := NewMatrix(2, 2)
	m.Data = []float64{
		1, 2,
		2, 4,
	}

	_, err := m.Inverse()
	if err == nil {
		t.Errorf("Era esperado um erro ao tentar inverter uma matriz singular, mas passou")
	}
}