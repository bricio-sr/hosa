package state

import (
	"errors"
	"sync"

	"github.com/bricio-sr/hosa/internal/linalg"
)

// RingBuffer atua como a memória de curto prazo (Sistema Límbico) do HOSA.
// Armazena as últimas N leituras de forma contígua e thread-safe, com zero alocação dinâmica.
type RingBuffer struct {
	data     *linalg.Matrix // A matriz que guarda as amostras
	head     int            // O ponteiro de onde inserir a próxima leitura
	count    int            // Quantos itens válidos temos no buffer
	capacity int            // Tamanho máximo do buffer (número de linhas)
	vars     int            // Número de variáveis monitoradas (CPU, RAM, I/O)
	mu       sync.RWMutex   // Proteção para concorrência (eBPF escreve, Córtex lê)
}

// NewRingBuffer inicializa o buffer circular.
func NewRingBuffer(capacity, vars int) *RingBuffer {
	return &RingBuffer{
		data:     linalg.NewMatrix(capacity, vars),
		capacity: capacity,
		vars:     vars,
		head:     0,
		count:    0,
	}
}

// Insert adiciona uma nova leitura sensorial (ex: [uso_cpu, uso_ram]) ao buffer.
func (rb *RingBuffer) Insert(reading []float64) error {
	if len(reading) != rb.vars {
		return errors.New("dimensão da leitura incompatível com o buffer")
	}

	rb.mu.Lock()
	defer rb.mu.Unlock()

	// Escreve os dados na linha apontada pelo 'head'
	for j := 0; j < rb.vars; j++ {
		rb.data.Set(rb.head, j, reading[j])
	}

	// Move o ponteiro de forma circular
	rb.head = (rb.head + 1) % rb.capacity

	// Incrementa o contador até atingir a capacidade máxima
	if rb.count < rb.capacity {
		rb.count++
	}

	return nil
}

// Snapshot retorna uma cópia contígua das amostras válidas para o Córtex (Matemática) processar.
// Retorna uma matriz N x Vars, onde N é a quantidade de leituras válidas (rb.count).
func (rb *RingBuffer) Snapshot() *linalg.Matrix {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	snap := linalg.NewMatrix(rb.count, rb.vars)
	
	// Copiamos os dados válidos. 
	// Obs: Para covariância, a ordem cronológica exata das linhas não altera o resultado da matriz,
	// então podemos simplesmente copiar as primeiras 'count' linhas da memória subjacente.
	for i := 0; i < rb.count; i++ {
		for j := 0; j < rb.vars; j++ {
			snap.Set(i, j, rb.data.Get(i, j))
		}
	}

	return snap
}

// IsReady verifica se temos o mínimo de amostras para fazer cálculos estatísticos (N > 1)
func (rb *RingBuffer) IsReady() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count > 1
}