package state

import (
	"math"
	"sync"
	"testing"
)

// epsilon para tolerância
const epsilon = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= epsilon
}

func TestRingBuffer_WrapAround(t *testing.T) {
	// Cria um buffer que guarda no máximo 3 leituras, de 2 variáveis (ex: CPU, RAM)
	rb := NewRingBuffer(3, 2)

	// Inserindo 4 leituras (vai forçar o buffer a dar a volta e sobrescrever a primeira)
	rb.Insert([]float64{1.0, 1.0}) // Será sobrescrito
	rb.Insert([]float64{2.0, 2.0})
	rb.Insert([]float64{3.0, 3.0})
	rb.Insert([]float64{4.0, 4.0}) // Sobrescreve a posição 0

	snap := rb.Snapshot()

	// A capacidade máxima é 3, então o snapshot tem que ter 3 linhas
	if snap.Rows != 3 {
		t.Fatalf("Esperado 3 linhas no snapshot, obtido %d", snap.Rows)
	}

	// Como a posição 0 foi sobrescrita pela 4ª leitura, a linha 0 deve ser {4.0, 4.0}
	if !almostEqual(snap.Get(0, 0), 4.0) || !almostEqual(snap.Get(0, 1), 4.0) {
		t.Errorf("Wrap around falhou. Esperado 4.0 na posição 0, obtido %f", snap.Get(0, 0))
	}
	// A linha 1 deve ter continuado intacta com a 2ª leitura {2.0, 2.0}
	if !almostEqual(snap.Get(1, 0), 2.0) || !almostEqual(snap.Get(1, 1), 2.0) {
		t.Errorf("Dados corrompidos na linha 1. Esperado 2.0, obtido %f", snap.Get(1, 0))
	}
}

func TestRingBuffer_Concurrency(t *testing.T) {
	// Teste de stress: escrevendo e lendo ao mesmo tempo com goroutines
	rb := NewRingBuffer(100, 2)
	var wg sync.WaitGroup

	// 10 goroutines simulando o eBPF inserindo dados freneticamente
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				_ = rb.Insert([]float64{float64(j), float64(j)})
			}
		}()
	}

	// 5 goroutines simulando o Córtex Preditivo tirando snapshots simultâneos
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				snap := rb.Snapshot()
				_ = snap // Só para garantir que a leitura não crashe o sistema
			}
		}()
	}

	wg.Wait()
	// Se o código chegar aqui sem dar "panic: data race" ou falha de mutex, a arquitetura é a prova de balas.
}