// Package brain includes the Thalamic Filter — the single log authority of HOSA.
//
// All output goes through here. Nothing else logs directly to stdout/stderr.
// In homeostasis: one heartbeat line every 30s, silence otherwise.
// In anomaly: one structured line per significant event.
//
// Log format:
//
//	HOSA [TAG]  key=value key=value ...
//
// Tags: BOOT, SENSOR, HEARTBEAT, ANOMALY, ESCALATION, CONTAINMENT, RECOVERY, HOMEOSTASIS, PROTECTION
//
// Reference: HOSA whitepaper, Section 6.3 — Regime 0, Thalamic Filter.
package brain

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ThalamicConfig defines the filter behaviour.
type ThalamicConfig struct {
	// HeartbeatInterval is the period between heartbeats in homeostasis.
	HeartbeatInterval time.Duration
}

// DefaultThalamicConfig returns the recommended default.
func DefaultThalamicConfig() ThalamicConfig {
	return ThalamicConfig{
		HeartbeatInterval: 30 * time.Second,
	}
}

// TelemetryEvent is the structured event emitted by the filter.
type TelemetryEvent struct {
	Timestamp   time.Time
	Type        TelemetryEventType
	Level       AlertLevel
	StressDM    float64
	StressDMDot float64
	Message     string
}

// TelemetryEventType classifies emitted events.
type TelemetryEventType int

const (
	EventHeartbeat         TelemetryEventType = iota
	EventAnomalyDetected
	EventLevelChange
	EventContainmentApplied
	EventHomeostasisRestored
	EventProtectionApplied
)

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
	case EventProtectionApplied:
		return "protection_applied"
	default:
		return "unknown"
	}
}

// ThalamicFilter is the single log authority of HOSA.
type ThalamicFilter struct {
	config        ThalamicConfig
	mu            sync.Mutex
	currentLevel  AlertLevel
	prevLevel     AlertLevel
	lastHeartbeat time.Time
	handler       func(TelemetryEvent)
}

// NewThalamicFilter initializes the filter.
// handler=nil uses the default structured log handler.
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

// Boot emits the initial startup lines (topology, sensor, status).
// Called once during initialization — replaces all ad-hoc log.Printf in main.
func (tf *ThalamicFilter) Boot(fields string) {
	tf.emit(TelemetryEvent{
		Timestamp: time.Now(),
		Type:      EventHeartbeat, // reuse type, tag overridden by Boot handler
		Message:   "[BOOT]	   " + fields,
	})
}

// Sensor emits the sensor initialization summary.
func (tf *ThalamicFilter) Sensor(fields string) {
	tf.emit(TelemetryEvent{
		Timestamp: time.Now(),
		Message:   "SENSOR " + fields,
	})
}

// Observe is called every cortex cycle. Decides what to emit.
func (tf *ThalamicFilter) Observe(level AlertLevel, dm, dmDot float64) {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	now := time.Now()
	tf.prevLevel = tf.currentLevel
	tf.currentLevel = level

	switch {
	case level == LevelHomeostasis && tf.prevLevel == LevelHomeostasis:
		if now.Sub(tf.lastHeartbeat) >= tf.config.HeartbeatInterval {
			tf.emit(TelemetryEvent{
				Timestamp: now,
				Type:      EventHeartbeat,
				Level:     level,
				StressDM:  dm,
				Message:   fmt.Sprintf("[HEARTBEAT]   level=0 dm=%.4f", dm),
			})
			tf.lastHeartbeat = now
		}

	case level > LevelHomeostasis && tf.prevLevel == LevelHomeostasis:
		tf.emit(TelemetryEvent{
			Timestamp:   now,
			Type:        EventAnomalyDetected,
			Level:       level,
			StressDM:    dm,
			StressDMDot: dmDot,
			Message: fmt.Sprintf("[ANOMALY]     level=%d dm=%.4f dm_dot=%+.4f",
				level, dm, dmDot),
		})

	case level > tf.prevLevel:
		tf.emit(TelemetryEvent{
			Timestamp:   now,
			Type:        EventLevelChange,
			Level:       level,
			StressDM:    dm,
			StressDMDot: dmDot,
			Message: fmt.Sprintf("[ESCALATION]  level=%d→%d dm=%.4f dm_dot=%+.4f",
				tf.prevLevel, level, dm, dmDot),
		})

	case level < tf.prevLevel && level > LevelHomeostasis:
		tf.emit(TelemetryEvent{
			Timestamp:   now,
			Type:        EventLevelChange,
			Level:       level,
			StressDM:    dm,
			StressDMDot: dmDot,
			Message: fmt.Sprintf("[RECOVERY]    level=%d→%d dm=%.4f dm_dot=%+.4f",
				tf.prevLevel, level, dm, dmDot),
		})

	case level == LevelHomeostasis && tf.prevLevel > LevelHomeostasis:
		tf.emit(TelemetryEvent{
			Timestamp: now,
			Type:      EventHomeostasisRestored,
			Level:     level,
			StressDM:  dm,
			Message:   fmt.Sprintf("[HOMEOSTASIS] dm=%.4f", dm),
		})
		tf.lastHeartbeat = now
	}
}

// NotifyContainment is called by the motor when a containment action is applied.
func (tf *ThalamicFilter) NotifyContainment(level AlertLevel, dm float64, action string) {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	tag := "[CONTAINMENT]"
	evtType := EventContainmentApplied
	if level == LevelProtection {
		tag = "[PROTECTION] "
		evtType = EventProtectionApplied
	}

	tf.emit(TelemetryEvent{
		Timestamp: time.Now(),
		Type:      evtType,
		Level:     level,
		StressDM:  dm,
		Message:   fmt.Sprintf("%s level=%d dm=%.4f action=%s", tag, level, dm, action),
	})
}

// CurrentLevel returns the last observed level.
func (tf *ThalamicFilter) CurrentLevel() AlertLevel {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	return tf.currentLevel
}

func (tf *ThalamicFilter) emit(evt TelemetryEvent) {
	tf.handler(evt)
}

// defaultLogHandler is the production handler — single structured line per event.
func defaultLogHandler(evt TelemetryEvent) {
	log.Printf("HOSA %s", evt.Message)
}