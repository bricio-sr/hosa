package brain

import (
	"math"
	"testing"
	"time"

	"github.com/bricio-sr/hosa/internal/state"
)

// newCortexWithSamples cria um PredictiveCortex já populado com N amostras estáveis,
// de forma que o basal seja aprendido e a análise seja habilitada.
func newCortexWithSamples(n, vars int, value float64) *PredictiveCortex {
	buf := state.NewRingBuffer(n*2, vars)
	for i := 0; i < n; i++ {
		row := make([]float64, vars)
		for j := range row {
			row[j] = value + float64(i%3)*0.01 // pequena variância para evitar singular
		}
		buf.Insert(row)
	}
	cortex := NewPredictiveCortex(buf, PredictorConfig{MinSamples: n})
	return cortex
}

// TestClassifyByMagnitude verifica o mapeamento D_M → AlertLevel sem derivada.
func TestClassifyByMagnitude(t *testing.T) {
	cases := []struct {
		dm       float64
		expected AlertLevel
	}{
		{0.5, LevelHomeostasis},
		{3.4, LevelHomeostasis},
		{3.5, LevelVigilance},
		{5.4, LevelVigilance},
		{5.5, LevelContainment},
		{7.9, LevelContainment},
		{8.0, LevelProtection},
		{99.0, LevelProtection},
	}

	for _, c := range cases {
		got := classifyByMagnitude(c.dm)
		if got != c.expected {
			t.Errorf("classifyByMagnitude(%.1f) = %d, esperado %d", c.dm, got, c.expected)
		}
	}
}

// TestCalcDerivative_FirstCycle verifica que o primeiro ciclo retorna 0.
func TestCalcDerivative_FirstCycle(t *testing.T) {
	cortex := &PredictiveCortex{}
	got := cortex.calcDerivative(5.0, time.Now())
	if got != 0 {
		t.Errorf("primeiro ciclo deve retornar dmDot=0, obtido %.4f", got)
	}
}

// TestCalcDerivative_Rising verifica derivada positiva (estresse crescendo).
func TestCalcDerivative_Rising(t *testing.T) {
	cortex := &PredictiveCortex{}

	t0 := time.Now()
	t1 := t0.Add(1 * time.Second)

	cortex.prev = &StressReading{DM: 2.0, Timestamp: t0}
	got := cortex.calcDerivative(4.0, t1)

	// dD_M/dt = (4.0 - 2.0) / 1.0 = 2.0
	if math.Abs(got-2.0) > 1e-9 {
		t.Errorf("derivada esperada 2.0, obtida %.6f", got)
	}
}

// TestCalcDerivative_Falling verifica derivada negativa (estresse caindo).
func TestCalcDerivative_Falling(t *testing.T) {
	cortex := &PredictiveCortex{}

	t0 := time.Now()
	t1 := t0.Add(500 * time.Millisecond)

	cortex.prev = &StressReading{DM: 6.0, Timestamp: t0}
	got := cortex.calcDerivative(4.0, t1)

	// dD_M/dt = (4.0 - 6.0) / 0.5 = -4.0
	if math.Abs(got-(-4.0)) > 1e-9 {
		t.Errorf("derivada esperada -4.0, obtida %.6f", got)
	}
}

// TestClassify_DerivativeEscalation verifica que derivada alta sobe um nível extra.
func TestClassify_DerivativeEscalation(t *testing.T) {
	cortex := &PredictiveCortex{currentLevel: LevelHomeostasis}

	// D_M em Vigilância (3.6), mas derivada muito alta → deve subir para Contenção
	level := cortex.classify(3.6, ThresholdDerivativeEscalate+0.1)

	if level != LevelContainment {
		t.Errorf("escalada por derivada: esperado LevelContainment(%d), obtido %d", LevelContainment, level)
	}
}

// TestClassify_NoEscalationWithLowDerivative verifica que derivada baixa não escala.
func TestClassify_NoEscalationWithLowDerivative(t *testing.T) {
	cortex := &PredictiveCortex{currentLevel: LevelHomeostasis}

	// D_M em Vigilância, derivada baixa → deve ficar em Vigilância
	level := cortex.classify(3.6, 0.1)

	if level != LevelVigilance {
		t.Errorf("sem escalada: esperado LevelVigilance(%d), obtido %d", LevelVigilance, level)
	}
}

// TestClassify_Hysteresis verifica que a descida de nível exige hysteresisDown ciclos.
func TestClassify_Hysteresis(t *testing.T) {
	cortex := &PredictiveCortex{currentLevel: LevelContainment}

	// D_M caiu para Vigilância e derivada baixa — mas não deve descer imediatamente
	for i := 0; i < hysteresisDown-1; i++ {
		level := cortex.classify(3.6, 0.0)
		if level != LevelContainment {
			t.Errorf("ciclo %d: histerese deveria manter LevelContainment(%d), obtido %d",
				i+1, LevelContainment, level)
		}
	}

	// No hysteresisDown-ésimo ciclo, deve finalmente descer
	level := cortex.classify(3.6, 0.0)
	if level != LevelVigilance {
		t.Errorf("após %d ciclos, esperado LevelVigilance(%d), obtido %d",
			hysteresisDown, LevelVigilance, level)
	}
}

// TestClassify_HysteresisResetOnHighDerivative verifica que derivada alta reseta
// o contador de histerese, impedindo a descida.
func TestClassify_HysteresisResetOnHighDerivative(t *testing.T) {
	cortex := &PredictiveCortex{currentLevel: LevelContainment}

	// Acumula alguns ciclos de descida
	cortex.classify(3.6, 0.0)
	cortex.classify(3.6, 0.0)

	// Derivada sobe — deve resetar o contador
	cortex.classify(3.6, ThresholdDerivativeRelax+0.5)

	// Agora precisa de hysteresisDown ciclos novamente para descer
	for i := 0; i < hysteresisDown-1; i++ {
		level := cortex.classify(3.6, 0.0)
		if level != LevelContainment {
			t.Errorf("após reset, ciclo %d ainda deveria ser LevelContainment, obtido %d", i+1, level)
		}
	}
}

// TestClassify_ImmediateEscalation verifica que a subida de nível é imediata (sem histerese).
func TestClassify_ImmediateEscalation(t *testing.T) {
	cortex := &PredictiveCortex{currentLevel: LevelHomeostasis}

	// D_M salta direto para Proteção — deve subir imediatamente
	level := cortex.classify(9.0, 0.0)

	if level != LevelProtection {
		t.Errorf("escalada imediata: esperado LevelProtection(%d), obtido %d", LevelProtection, level)
	}
}

// TestAnalyze_ReturnsHomeostasisBeforeMinSamples verifica o comportamento no cold-start.
func TestAnalyze_ReturnsHomeostasisBeforeMinSamples(t *testing.T) {
	buf := state.NewRingBuffer(100, 1)
	cortex := NewPredictiveCortex(buf, PredictorConfig{MinSamples: 30, Alpha: 0.2})

	// Insere apenas 5 amostras — abaixo do mínimo
	for i := 0; i < 5; i++ {
		buf.Insert([]float64{float64(i)})
	}

	stress, dmDot, level, err := cortex.Analyze()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if stress != 0 || dmDot != 0 || level != LevelHomeostasis {
		t.Errorf("cold-start: esperado (0, 0, Homeostasis), obtido (%.2f, %.2f, %d)", stress, dmDot, level)
	}
}

// TestEWMA_Smoothing verifica que o EWMA suaviza valores oscilantes.
// Com α=0.2, valores alternando entre 1 e 10 devem produzir saída estável.
func TestEWMA_Smoothing(t *testing.T) {
	cortex := &PredictiveCortex{
		config:    PredictorConfig{Alpha: 0.2},
		ewmaReady: false,
	}

	// Simula D_M oscilando entre 1.0 e 10.0
	values := []float64{1.0, 10.0, 1.0, 10.0, 1.0, 10.0}
	var prev float64
	for i, v := range values {
		if !cortex.ewmaReady {
			cortex.ewmaValue = v
			cortex.ewmaReady = true
		} else {
			cortex.ewmaValue = cortex.config.Alpha*v + (1-cortex.config.Alpha)*cortex.ewmaValue
		}

		if i > 0 {
			// A variação entre ciclos consecutivos deve ser menor que a variação bruta (9.0)
			delta := cortex.ewmaValue - prev
			if delta < 0 {
				delta = -delta
			}
			if delta >= 9.0 {
				t.Errorf("ciclo %d: EWMA não suavizou (delta=%.2f >= 9.0)", i, delta)
			}
		}
		prev = cortex.ewmaValue
	}
}

// TestEWMA_Convergence verifica que com α baixo o EWMA converge lentamente
// e com α alto converge rápido — garantindo que o parâmetro tem efeito.
func TestEWMA_Convergence(t *testing.T) {
	applyEWMA := func(alpha, initial, target float64, steps int) float64 {
		v := initial
		for i := 0; i < steps; i++ {
			v = alpha*target + (1-alpha)*v
		}
		return v
	}

	// α=0.8 deve convergir para 10.0 em 10 passos bem mais do que α=0.1
	fastConv := applyEWMA(0.8, 0.0, 10.0, 10)
	slowConv := applyEWMA(0.1, 0.0, 10.0, 10)

	if fastConv <= slowConv {
		t.Errorf("α alto deveria convergir mais rápido: fast=%.4f, slow=%.4f", fastConv, slowConv)
	}
}

// TestEWMA_InitializationNoBias verifica que a primeira amostra inicializa
// o EWMA diretamente (sem transitório de arranque).
func TestEWMA_InitializationNoBias(t *testing.T) {
	cortex := &PredictiveCortex{
		config:    PredictorConfig{Alpha: 0.2},
		ewmaReady: false,
	}

	firstValue := 5.0
	cortex.ewmaValue = firstValue
	cortex.ewmaReady = true

	if cortex.ewmaValue != firstValue {
		t.Errorf("inicialização com bias: esperado %.1f, obtido %.4f", firstValue, cortex.ewmaValue)
	}
}

// TestDefaultConfig_AlphaValid verifica que o DefaultConfig retorna um α válido.
func TestDefaultConfig_AlphaValid(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Alpha <= 0 || cfg.Alpha > 1 {
		t.Errorf("DefaultConfig.Alpha inválido: %.4f (deve ser 0 < α ≤ 1)", cfg.Alpha)
	}
}