package linalg

import (
	"errors"
	"math"
)

// Mul multiplica a matriz M pela matriz B (M * B).
func (m *Matrix) Mul(b *Matrix) (*Matrix, error) {
	if m.Cols != b.Rows {
		return nil, errors.New("dimensões incompativeis: Cols de A deve ser igual a Rows de B")
	}

	result := NewMatrix(m.Rows, b.Cols)
	for i := 0; i < m.Rows; i++ {
		for j := 0; j < b.Cols; j++ {
			var sum float64
			for k := 0; k < m.Cols; k++ {
				sum += m.Get(i, k) * b.Get(k, j)
			}
			result.Set(i, j, sum)
		}
	}
	return result, nil
}

// Inverse calcula a matriz inversa (M^-1) usando Eliminação de Gauss-Jordan com pivoteamento.
// Isso é o coração para acharmos a Matriz de Covariância Inversa depois.
func (m *Matrix) Inverse() (*Matrix, error) {
	if m.Rows != m.Cols {
		return nil, errors.New("a matriz precisa ser quadrada para ser invertida")
	}

	n := m.Rows
	// Cria uma matriz aumentada [M | I]
	aug := NewMatrix(n, 2*n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			aug.Set(i, j, m.Get(i, j))
		}
		aug.Set(i, n+i, 1.0) // Matriz Identidade do lado direito
	}

	// Eliminação de Gauss-Jordan
	for i := 0; i < n; i++ {
		// Pivoteamento parcial (acha o maior valor na coluna para evitar divisão por zero ou instabilidade)
		pivot := i
		maxVal := math.Abs(aug.Get(i, i))
		for k := i + 1; k < n; k++ {
			if val := math.Abs(aug.Get(k, i)); val > maxVal {
				maxVal = val
				pivot = k
			}
		}

		if maxVal == 0 {
			return nil, errors.New("matriz singular, não possui inversa")
		}

		// Troca as linhas se o pivô não for o atual
		if pivot != i {
			for j := 0; j < 2*n; j++ {
				temp := aug.Get(i, j)
				aug.Set(i, j, aug.Get(pivot, j))
				aug.Set(pivot, j, temp)
			}
		}

		// Normaliza a linha do pivô (divide tudo pelo valor do pivô para ele virar 1)
		pivotVal := aug.Get(i, i)
		for j := 0; j < 2*n; j++ {
			aug.Set(i, j, aug.Get(i, j)/pivotVal)
		}

		// Zera o resto da coluna
		for k := 0; k < n; k++ {
			if k != i {
				factor := aug.Get(k, i)
				for j := 0; j < 2*n; j++ {
					aug.Set(k, j, aug.Get(k, j)-(factor*aug.Get(i, j)))
				}
			}
		}
	}

	// Extrai a matriz inversa do lado direito da aumentada
	inv := NewMatrix(n, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			inv.Set(i, j, aug.Get(i, n+j))
		}
	}

	return inv, nil
}