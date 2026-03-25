// Package brain inclui o Filtro Talâmico do HOSA — a camada que decide
// o que é emitido para sistemas externos em função do nível de alerta atual.
//
// Analogia biológica: o tálamo cerebral filtra e prioriza sinais sensoriais
// antes de encaminhá-los ao córtex. Em homeostase, a maioria dos sinais é
// suprimida. Em emergência, todos os canais são abertos.
//
// Referência: whitepaper HOSA, Seção 6.3 — Regime 0, Filtro Talâmico.
package brain

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ThalamicConfig define o comportamento do Filtro Talâmico.
type ThalamicConfig struct {
	// HeartbeatInterval é o período entre heartbeats em homeostase.
	// Whitepaper: "heartbeat mínimo periódico confirmando que o nó está vivo".
	HeartbeatInterval time.Duration

	// SuppressHomeostasisLogs controla se logs detalhados são suprimidos
	// quando o sistema está em homeostase. true = suprime (comportamento padrão).
	SuppressHomeostasisLogs bool
}

// DefaultThalamicConfig retorna a configuração padrão do filtro.
func DefaultThalamicConfig() ThalamicConfig {
	return ThalamicConfig{
		HeartbeatInterval:       30 * time.Second,
		SuppressHomeostasisLogs: true,
	}
}

// TelemetryEvent representa um evento de telemetria emitido pelo HOSA.
// Em homeostase, apenas eventos do tipo Heartbeat são emitidos.
// Em anomalia, eventos DetailedStress são emitidos a cada ciclo.
type TelemetryEvent struct {
	// Timestamp do evento.
	Timestamp time.Time

	// Type é o tipo do evento.
	Type TelemetryEventType

	// Level é o nível de alerta atual.
	Level AlertLevel

	// StressDM é a Distância de Mahalanobis suavizada (D̄_M).
	StressDM float64

	// StressDMDot é a derivada temporal dD̄_M/dt.
	StressDMDot float64

	// Message é uma descrição legível do evento.
	Message string
}

// TelemetryEventType classifica os eventos emitidos pelo filtro.
type TelemetryEventType int

const (
	// EventHeartbeat é emitido periodicamente em homeostase.
	// Confirma que o nó está vivo e saudável — informação mínima suficiente.
	EventHeartbeat TelemetryEventType = iota

	// EventAnomalyDetected é emitido quando o nível sobe acima de Homeostase.
	// Marca o início de um episódio de estresse.
	EventAnomalyDetected

	// EventLevelChange é emitido em cada transição de nível (escalada ou descida).
	EventLevelChange

	// EventContainmentApplied é emitido quando o motor aplica uma ação de contenção.
	EventContainmentApplied

	// EventHomeostasisRestored é emitido quando o sistema retorna à homeostase.
	EventHomeostasisRestored
)

// String retorna o nome legível do tipo de evento.
func (t TelemetryEventType) String() string {
	switch t {
	case EventHeartbeat:
		return "heartbeat"
	case EventAnomalyDetected:
		return "anomaly_detected"
	case EventLevelChange:
		return "level_change"
	case EventContainmentApplied:
		return "containment_applied"
	case EventHomeostasisRestored:
		return "homeostasis_restored"
	default:
		return "unknown"
	}
}

// ThalamicFilter é o guardião da telemetria do HOSA.
// Ele decide o que emitir para sistemas externos com base no nível de alerta.
type ThalamicFilter struct {
	config       ThalamicConfig
	mu           sync.Mutex
	currentLevel AlertLevel
	prevLevel    AlertLevel
	lastHeartbeat time.Time

	// handler é chamado para cada evento que passa pelo filtro.
	// Permite integrar com qualquer sistema de saída (log, webhook, etc).
	handler func(TelemetryEvent)
}

// NewThalamicFilter inicializa o filtro com a configuração e um handler de saída.
// O handler padrão (nil) usa o logger padrão do Go.
func NewThalamicFilter(cfg ThalamicConfig, handler func(TelemetryEvent)) *ThalamicFilter {
	if handler == nil {
		handler = defaultLogHandler
	}
	return &ThalamicFilter{
		config:        cfg,
		currentLevel:  LevelHomeostasis,
		prevLevel:     LevelHomeostasis,
		lastHeartbeat: time.Now(),
		handler:       handler,
	}
}

// Observe é o método principal do filtro. Chamado a cada ciclo do Córtex.
// Decide o que emitir com base no nível atual e no estado anterior.
func (tf *ThalamicFilter) Observe(level AlertLevel, dm, dmDot float64) {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	now := time.Now()
	tf.prevLevel = tf.currentLevel
	tf.currentLevel = level

	switch {
	case level == LevelHomeostasis && tf.prevLevel == LevelHomeostasis:
		// Homeostase estável — emite apenas heartbeat periódico.
		if now.Sub(tf.lastHeartbeat) >= tf.config.HeartbeatInterval {
			tf.emit(TelemetryEvent{
				Timestamp: now,
				Type:      EventHeartbeat,
				Level:     level,
				StressDM:  dm,
				Message:   fmt.Sprintf("nó saudável — D̄_M=%.4f", dm),
			})
			tf.lastHeartbeat = now
		}

	case level > LevelHomeostasis && tf.prevLevel == LevelHomeostasis:
		// Transição homeostase → anomalia: abre o canal completo.
		tf.emit(TelemetryEvent{
			Timestamp:   now,
			Type:        EventAnomalyDetected,
			Level:       level,
			StressDM:    dm,
			StressDMDot: dmDot,
			Message: fmt.Sprintf("anomalia detectada — nível=%d D̄_M=%.4f dD̄_M/dt=%.4f",
				level, dm, dmDot),
		})

	case level != tf.prevLevel && level > LevelHomeostasis:
		// Mudança de nível durante episódio de estresse.
		tf.emit(TelemetryEvent{
			Timestamp:   now,
			Type:        EventLevelChange,
			Level:       level,
			StressDM:    dm,
			StressDMDot: dmDot,
			Message: fmt.Sprintf("mudança de nível %d→%d — D̄_M=%.4f dD̄_M/dt=%.4f",
				tf.prevLevel, level, dm, dmDot),
		})

	case level == LevelHomeostasis && tf.prevLevel > LevelHomeostasis:
		// Retorno à homeostase: emite evento de recuperação e reinicia heartbeat.
		tf.emit(TelemetryEvent{
			Timestamp: now,
			Type:      EventHomeostasisRestored,
			Level:     level,
			StressDM:  dm,
			Message:   fmt.Sprintf("homeostase restaurada — D̄_M=%.4f", dm),
		})
		tf.lastHeartbeat = now
	}
}

// NotifyContainment é chamado pelo motor quando uma ação de contenção é aplicada.
// Sempre emite — contenção é uma ação auditável independente do nível.
func (tf *ThalamicFilter) NotifyContainment(level AlertLevel, dm float64, action string) {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	tf.emit(TelemetryEvent{
		Timestamp: time.Now(),
		Type:      EventContainmentApplied,
		Level:     level,
		StressDM:  dm,
		Message:   fmt.Sprintf("contenção aplicada — nível=%d D̄_M=%.4f ação=%s", level, dm, action),
	})
}

// CurrentLevel retorna o último nível observado pelo filtro.
func (tf *ThalamicFilter) CurrentLevel() AlertLevel {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	return tf.currentLevel
}

// emit chama o handler com o evento. Deve ser chamado com o mutex já adquirido.
func (tf *ThalamicFilter) emit(evt TelemetryEvent) {
	tf.handler(evt)
}

// defaultLogHandler é o handler padrão — emite via log padrão do Go.
func defaultLogHandler(evt TelemetryEvent) {
	prefix := "HOSA [TÁLAMO]"
	switch evt.Type {
	case EventHeartbeat:
		log.Printf("%s heartbeat: %s", prefix, evt.Message)
	case EventAnomalyDetected:
		log.Printf("%s ANOMALIA: %s", prefix, evt.Message)
	case EventLevelChange:
		log.Printf("%s nível alterado: %s", prefix, evt.Message)
	case EventContainmentApplied:
		log.Printf("%s contenção: %s", prefix, evt.Message)
	case EventHomeostasisRestored:
		log.Printf("%s RECUPERADO: %s", prefix, evt.Message)
	}
}