package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bricio-sr/hosa/internal/brain"
	"github.com/bricio-sr/hosa/internal/sensor"
	"github.com/bricio-sr/hosa/internal/state"
)

// Parâmetros do loop principal.
const (
	// ringBufferCapacity é quantas amostras de histórico o HOSA mantém em memória.
	// 300 amostras @ 1s/amostra = 5 minutos de histórico para aprender o basal.
	ringBufferCapacity = 300

	// numVars é a dimensão do vetor de métricas monitoradas.
	// Por ora: [brk_count] (1 variável). Expandir conforme sensor evolui.
	numVars = 1

	// normalInterval é a frequência de coleta em homeostase.
	normalInterval = 1 * time.Second

	// vigilanceInterval é a frequência de coleta em Nível 1 (Vigilância).
	// O HOSA aumenta a amostragem ao detectar o primeiro desvio — como pupilas dilatando.
	vigilanceInterval = 100 * time.Millisecond
)

func main() {
	log.Println("HOSA: Homeostasis Operating System Agent — iniciando...")

	// --- Inicializa as camadas ---

	// Camada 1: Memória de Curto Prazo (Sistema Límbico)
	// O RingBuffer armazena o histórico recente de métricas de forma thread-safe.
	buf := state.NewRingBuffer(ringBufferCapacity, numVars)

	// Camada 2: Sensor eBPF (Sistema Nervoso Periférico)
	// O Collector pendura um programa no Kernel para capturar eventos de alocação.
	col := &sensor.Collector{}
	if err := col.Start(); err != nil {
		log.Fatalf("HOSA: falha ao inicializar sensor eBPF: %v", err)
	}
	defer col.Close()

	// Camada 3: Córtex Preditivo (Cérebro)
	// O PredictiveCortex analisa o buffer e retorna o nível de alerta.
	cortex := brain.NewPredictiveCortex(buf, brain.DefaultConfig())

	// --- Configura o shutdown gracioso via SIGINT / SIGTERM ---
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("HOSA: sistema online. Aguardando amostras para calibrar o basal...")

	interval := normalInterval

	// --- Loop Principal: O Arco Reflexo ---
	// Sensor → Memória → Córtex → Motor (a ser implementado)
	for {
		select {
		case <-ctx.Done():
			log.Println("HOSA: sinal de encerramento recebido. Desligando...")
			return

		case <-time.After(interval):
			// Passo 1 — SENTIR: coleta as métricas atuais via eBPF
			reading := []float64{col.ReadMetrics()}

			// Passo 2 — MEMORIZAR: insere no buffer circular
			if err := buf.Insert(reading); err != nil {
				log.Printf("HOSA: erro ao inserir no buffer: %v", err)
				continue
			}

			// Passo 3 — ANALISAR: o Córtex avalia o estado atual vs. basal
			stress, level, err := cortex.Analyze()
			if err != nil {
				log.Printf("HOSA: erro na análise do córtex: %v", err)
				continue
			}

			// Passo 4 — REAGIR: ajusta comportamento conforme o nível de alerta
			interval = react(stress, level)
		}
	}
}

// react loga o estado atual e retorna o intervalo de próxima amostragem.
// Esta função será expandida para acionar o Motor (cgroups/XDP) nos próximos commits.
func react(stress float64, level brain.AlertLevel) time.Duration {
	switch level {
	case brain.LevelHomeostasis:
		// Sistema saudável. Amostragem normal.
		return normalInterval

	case brain.LevelVigilance:
		// Desvio detectado. Aumenta a frequência de amostragem para rastrear a progressão.
		log.Printf("HOSA [VIGILÂNCIA] stress=%.4f — monitoramento intensificado", stress)
		return vigilanceInterval

	case brain.LevelContainment:
		// Estresse confirmado. TODO: acionar motor/cgroups.go para conter o processo.
		log.Printf("HOSA [CONTENÇÃO]  stress=%.4f — contenção via cgroups (não implementado)", stress)
		return vigilanceInterval

	case brain.LevelProtection:
		// Risco de colapso iminente. TODO: acionar motor/signals.go para ações drásticas.
		log.Printf("HOSA [PROTEÇÃO]   stress=%.4f — proteção do host (não implementado)", stress)
		return vigilanceInterval

	default:
		return normalInterval
	}
}