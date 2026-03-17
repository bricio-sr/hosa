package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bricio-sr/hosa/internal/brain"
	"github.com/bricio-sr/hosa/internal/motor"
	"github.com/bricio-sr/hosa/internal/sensor"
	"github.com/bricio-sr/hosa/internal/state"
	"github.com/bricio-sr/hosa/internal/syscgroup"
)

const (
	// ringBufferCapacity é quantas amostras de histórico o HOSA mantém em memória.
	// 300 amostras @ 1s/amostra = 5 minutos de histórico para aprender o basal.
	ringBufferCapacity = 300

	// numVars é a dimensão do vetor de métricas monitoradas.
	// Por ora: [brk_count] (1 variável). Expandir conforme sensor evolui.
	numVars = 1

	// normalInterval é a frequência de coleta em homeostase.
	normalInterval = 1 * time.Second

	// vigilanceInterval é a frequência de coleta em Nível 1+ (Vigilância/Contenção/Proteção).
	vigilanceInterval = 100 * time.Millisecond
)

func main() {
	log.Println("HOSA: Homeostasis Operating System Agent — iniciando...")

	// --- Camada 1: Memória de Curto Prazo (Sistema Límbico) ---
	buf := state.NewRingBuffer(ringBufferCapacity, numVars)

	// --- Camada 2: Sensor eBPF (Sistema Nervoso Periférico) ---
	col := &sensor.Collector{}
	if err := col.Start(); err != nil {
		log.Fatalf("HOSA: falha ao inicializar sensor eBPF: %v", err)
	}
	defer col.Close()

	// --- Camada 3: Córtex Preditivo (Cérebro) ---
	cortex := brain.NewPredictiveCortex(buf, brain.DefaultConfig())

	// --- Camada 4: Motor (Sistema Motor) ---
	// Garante que o cgroup /sys/fs/cgroup/hosa existe e inicializa o motor.
	cgPath, err := syscgroup.EnsureHosaCgroup()
	if err != nil {
		log.Fatalf("HOSA: falha ao inicializar cgroup: %v", err)
	}
	mot := motor.NewCgroupMotor(cgPath)

	// Lê o total de memória disponível uma vez na inicialização.
	// Este valor é usado como referência para calcular os limites proporcionais.
	memTotal, err := readMemTotal()
	if err != nil {
		log.Fatalf("HOSA: falha ao ler memória total do host: %v", err)
	}
	log.Printf("HOSA: memória total do host: %d bytes (%.1f GB)", memTotal, float64(memTotal)/(1<<30))

	// --- Shutdown gracioso ---
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("HOSA: sistema online. Aguardando amostras para calibrar o basal...")

	interval := normalInterval

	// --- Loop Principal: O Arco Reflexo ---
	// Sensor → Memória → Córtex → Motor
	for {
		select {
		case <-ctx.Done():
			log.Println("HOSA: sinal de encerramento recebido. Restaurando homeostase e desligando...")
			// Garante que os limites de contenção são removidos antes de sair.
			if err := mot.Apply(motor.LevelHomeostasis, memTotal); err != nil {
				log.Printf("HOSA: erro ao restaurar homeostase no shutdown: %v", err)
			}
			return

		case <-time.After(interval):
			// Passo 1 — SENTIR
			reading := []float64{col.ReadMetrics()}

			// Passo 2 — MEMORIZAR
			if err := buf.Insert(reading); err != nil {
				log.Printf("HOSA: erro ao inserir no buffer: %v", err)
				continue
			}

			// Passo 3 — ANALISAR
			stress, dmDot, level, err := cortex.Analyze()
			if err != nil {
				log.Printf("HOSA: erro na análise do córtex: %v", err)
				continue
			}

			// Passo 4 — REAGIR
			interval = react(mot, stress, dmDot, level, memTotal)
		}
	}
}

// react aciona o motor e retorna o intervalo de próxima amostragem.
func react(mot *motor.CgroupMotor, stress, dmDot float64, level brain.AlertLevel, memTotal uint64) time.Duration {
	containLevel := motor.ContainmentLevel(level)

	if err := mot.Apply(containLevel, memTotal); err != nil {
		log.Printf("HOSA: erro ao aplicar contenção (nível=%d): %v", level, err)
	}

	switch level {
	case brain.LevelHomeostasis:
		return normalInterval

	case brain.LevelVigilance:
		log.Printf("HOSA [VIGILÂNCIA]  D_M=%.4f dD_M/dt=%.4f — monitoramento intensificado", stress, dmDot)
		return vigilanceInterval

	case brain.LevelContainment:
		log.Printf("HOSA [CONTENÇÃO]   D_M=%.4f dD_M/dt=%.4f — cgroups aplicados", stress, dmDot)
		return vigilanceInterval

	case brain.LevelProtection:
		log.Printf("HOSA [PROTEÇÃO]    D_M=%.4f dD_M/dt=%.4f — contenção máxima aplicada", stress, dmDot)
		return vigilanceInterval

	default:
		return normalInterval
	}
}

// readMemTotal lê a memória total do host em bytes a partir de /proc/meminfo.
// Usa apenas stdlib — sem dependências externas.
func readMemTotal() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	// Formato da linha: "MemTotal:       16384000 kB"
	for _, line := range splitLines(string(data)) {
		if len(line) > 9 && line[:9] == "MemTotal:" {
			var kb uint64
			// Extrai o número da linha manualmente — sem fmt.Sscanf para não importar fmt
			fields := splitFields(line[9:])
			if len(fields) == 0 {
				continue
			}
			kb = parseUint(fields[0])
			return kb * 1024, nil // converte kB → bytes
		}
	}

	return 0, os.ErrNotExist
}

// splitLines divide uma string por '\n' sem alocar um slice de strings desnecessário.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	// Inclui o trecho final quando a string não termina com '\n'
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// splitFields divide uma string por espaços, ignorando múltiplos espaços consecutivos.
func splitFields(s string) []string {
	var fields []string
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			if start == -1 {
				start = i
			}
		} else {
			if start != -1 {
				fields = append(fields, s[start:i])
				start = -1
			}
		}
	}
	if start != -1 {
		fields = append(fields, s[start:])
	}
	return fields
}

// parseUint converte uma string decimal para uint64 sem usar strconv,
// retornando 0 para qualquer entrada inválida.
func parseUint(s string) uint64 {
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + uint64(c-'0')
	}
	return n
}