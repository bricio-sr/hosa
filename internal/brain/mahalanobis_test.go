package brain

import (
	"math"
	"testing"

	"github.com/bricio-sr/hosa/internal/linalg"
)

// epsilon para tolerância de ponto flutuante
const epsilon = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= epsilon
}

func TestCalculateStress(t *testing.T) {
	// Vamos simular um servidor monitorando 2 variáveis: CPU e Memória (Matriz 2x1)

	// 1. Vetor Médio (mu): O que é o "normal" do servidor
	// Digamos que o normal seja 2.0 de CPU e 3.0 de RAM
	mu := linalg.NewMatrix(2, 1)
	mu.Set(0, 0, 2.0)
	mu.Set(1, 0, 3.0)

	// 2. Matriz de Covariância Inversa (Sigma^-1)
	// Para facilitar o teste, usaremos a Matriz Identidade 2x2.
	// Isso transforma Mahalanobis na Distância Euclidiana clássica.
	invCov := linalg.NewMatrix(2, 2)
	invCov.Set(0, 0, 1.0)
	invCov.Set(0, 1, 0.0)
	invCov.Set(1, 0, 0.0)
	invCov.Set(1, 1, 1.0)

	// Inicializa o Córtex
	model := NewHomeostasisModel(mu, invCov)

	// 3. Vetor Atual (X): O estado do servidor AGORA
	// Digamos que subiu para 4.0 de CPU e 5.0 de RAM
	current := linalg.NewMatrix(2, 1)
	current.Set(0, 0, 4.0)
	current.Set(1, 0, 5.0)

	// Cálculo manual esperado:
	// diff = [4.0 - 2.0, 5.0 - 3.0]^T = [2.0, 2.0]^T
	// Como a covariância é identidade, o stress^2 = (2.0)^2 + (2.0)^2 = 4.0 + 4.0 = 8.0
	// Distância = sqrt(8.0)
	expectedStress := math.Sqrt(8.0)

	stress, err := model.CalculateStress(current)
	if err != nil {
		t.Fatalf("Erro inesperado ao calcular estresse: %v", err)
	}

	if !almostEqual(stress, expectedStress) {
		t.Errorf("Estresse calculado incorreto. Esperado %f, obtido %f", expectedStress, stress)
	}
}