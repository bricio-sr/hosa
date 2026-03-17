package brain

import (
	"math"
	"testing"
)

const wEpsilon = 1e-9

func wAlmostEqual(a, b float64) bool {
	return math.Abs(a-b) <= wEpsilon
}

// TestWelford_MeanUnivariate verifica a média incremental para 1 variável.
// Compara com a média batch calculada manualmente.
func TestWelford_MeanUnivariate(t *testing.T) {
	w := NewWelfordState(1)

	samples := []float64{2.0, 4.0, 6.0}
	for _, s := range samples {
		w.Update([]float64{s})
	}

	// Média esperada: (2 + 4 + 6) / 3 = 4.0
	mean := w.Mean()
	if !wAlmostEqual(mean.Get(0, 0), 4.0) {
		t.Errorf("média univariada: esperado 4.0, obtido %.10f", mean.Get(0, 0))
	}
}

// TestWelford_MeanMultivariate verifica a média para 2 variáveis simultaneamente.
func TestWelford_MeanMultivariate(t *testing.T) {
	w := NewWelfordState(2)

	// Mesmas amostras dos testes de linalg para permitir comparação direta
	w.Update([]float64{2.0, 3.0})
	w.Update([]float64{4.0, 7.0})
	w.Update([]float64{6.0, 5.0})

	mean := w.Mean()

	// CPU: (2+4+6)/3 = 4.0, Mem: (3+7+5)/3 = 5.0
	if !wAlmostEqual(mean.Get(0, 0), 4.0) {
		t.Errorf("média CPU: esperado 4.0, obtido %.10f", mean.Get(0, 0))
	}
	if !wAlmostEqual(mean.Get(1, 0), 5.0) {
		t.Errorf("média Mem: esperado 5.0, obtido %.10f", mean.Get(1, 0))
	}
}

// TestWelford_CovarianceMatchesBatch verifica que a covariância Welford é
// numericamente equivalente à covariância batch calculada pelo linalg.
// Este é o teste mais importante — garante que a implementação incremental
// produz o mesmo resultado que o cálculo completo.
func TestWelford_CovarianceMatchesBatch(t *testing.T) {
	w := NewWelfordState(2)

	w.Update([]float64{2.0, 3.0})
	w.Update([]float64{4.0, 7.0})
	w.Update([]float64{6.0, 5.0})

	cov, err := w.Covariance()
	if err != nil {
		t.Fatalf("Covariance() falhou: %v", err)
	}

	// Valores esperados calculados manualmente (mesmos do TestCovarianceMatrix em linalg):
	// Var(CPU) = 4.0, Var(Mem) = 4.0, Cov(CPU,Mem) = 2.0
	expected := [][]float64{
		{4.0, 2.0},
		{2.0, 4.0},
	}

	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			got := cov.Get(i, j)
			exp := expected[i][j]
			if !wAlmostEqual(got, exp) {
				t.Errorf("cov[%d][%d]: esperado %.10f, obtido %.10f", i, j, exp, got)
			}
		}
	}
}

// TestWelford_IncrementalVsBatch verifica que adicionar amostras uma a uma
// produz o mesmo resultado que processar todas de uma vez (comutatividade).
func TestWelford_IncrementalVsBatch(t *testing.T) {
	samples := [][2]float64{
		{1.0, 5.0}, {3.0, 2.0}, {5.0, 8.0},
		{2.0, 4.0}, {7.0, 1.0}, {4.0, 6.0},
	}

	// Batch: popula tudo de uma vez
	wBatch := NewWelfordState(2)
	for _, s := range samples {
		wBatch.Update([]float64{s[0], s[1]})
	}

	// Incremental: popula metade, depois o resto
	wInc := NewWelfordState(2)
	for i := 0; i < 3; i++ {
		wInc.Update([]float64{samples[i][0], samples[i][1]})
	}
	for i := 3; i < 6; i++ {
		wInc.Update([]float64{samples[i][0], samples[i][1]})
	}

	// Médias devem ser idênticas
	mBatch := wBatch.Mean()
	mInc := wInc.Mean()
	for j := 0; j < 2; j++ {
		if !wAlmostEqual(mBatch.Get(j, 0), mInc.Get(j, 0)) {
			t.Errorf("média[%d]: batch=%.10f, incremental=%.10f", j, mBatch.Get(j, 0), mInc.Get(j, 0))
		}
	}

	// Covariâncias devem ser idênticas
	covBatch, _ := wBatch.Covariance()
	covInc, _ := wInc.Covariance()
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			b := covBatch.Get(i, j)
			inc := covInc.Get(i, j)
			if !wAlmostEqual(b, inc) {
				t.Errorf("cov[%d][%d]: batch=%.10f, incremental=%.10f", i, j, b, inc)
			}
		}
	}
}

// TestWelford_IsReady verifica o gate de mínimo de amostras.
func TestWelford_IsReady(t *testing.T) {
	w := NewWelfordState(1)

	if w.IsReady(30) {
		t.Error("IsReady(30) deveria ser false com 0 amostras")
	}

	for i := 0; i < 29; i++ {
		w.Update([]float64{float64(i)})
	}
	if w.IsReady(30) {
		t.Error("IsReady(30) deveria ser false com 29 amostras")
	}

	w.Update([]float64{29.0})
	if !w.IsReady(30) {
		t.Error("IsReady(30) deveria ser true com 30 amostras")
	}
}

// TestWelford_CovarianceRequiresN2 verifica erro com menos de 2 amostras.
func TestWelford_CovarianceRequiresN2(t *testing.T) {
	w := NewWelfordState(2)

	_, err := w.Covariance()
	if err == nil {
		t.Error("Covariance() com n=0 deveria retornar erro")
	}

	w.Update([]float64{1.0, 2.0})
	_, err = w.Covariance()
	if err == nil {
		t.Error("Covariance() com n=1 deveria retornar erro")
	}

	w.Update([]float64{3.0, 4.0})
	_, err = w.Covariance()
	if err != nil {
		t.Errorf("Covariance() com n=2 não deveria retornar erro: %v", err)
	}
}

// TestWelford_StdDev verifica desvio padrão para amostras com variância conhecida.
func TestWelford_StdDev(t *testing.T) {
	w := NewWelfordState(1)

	// Amostras: 2, 4, 6 — variância amostral = 4.0, std = 2.0
	w.Update([]float64{2.0})
	w.Update([]float64{4.0})
	w.Update([]float64{6.0})

	std := w.StdDev()
	if !wAlmostEqual(std[0], 2.0) {
		t.Errorf("StdDev: esperado 2.0, obtido %.10f", std[0])
	}
}

// TestWelford_Habituation simula uma mudança permanente de patamar e verifica
// que o basal converge para o novo valor — o mecanismo de habituação.
func TestWelford_Habituation(t *testing.T) {
	w := NewWelfordState(1)

	// Fase 1: basal baixo (CPU média ~20%)
	for i := 0; i < 100; i++ {
		w.Update([]float64{20.0 + float64(i%3)})
	}

	meanBefore := w.Mean().Get(0, 0)

	// Fase 2: novo patamar permanente (CPU ~60% após upgrade de serviço)
	for i := 0; i < 200; i++ {
		w.Update([]float64{60.0 + float64(i%3)})
	}

	meanAfter := w.Mean().Get(0, 0)

	// Após 200 amostras no novo patamar (vs 100 no anterior),
	// a média deve ter se movido significativamente em direção a 60.
	if meanAfter <= meanBefore+10.0 {
		t.Errorf("habituação: média não convergiu para novo patamar. antes=%.2f, depois=%.2f",
			meanBefore, meanAfter)
	}
	if meanAfter >= 60.0 {
		t.Errorf("habituação: média ultrapassou o novo patamar (%.2f >= 60.0)", meanAfter)
	}
}

// TestWelford_DimensionMismatch verifica erro de dimensão na Update.
func TestWelford_DimensionMismatch(t *testing.T) {
	w := NewWelfordState(2)

	err := w.Update([]float64{1.0}) // 1 elemento, esperado 2
	if err == nil {
		t.Error("Update com dimensão errada deveria retornar erro")
	}
}