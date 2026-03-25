package brain

import (
	"testing"
	"time"
)

// collectEvents é um helper que captura eventos emitidos pelo filtro em uma slice.
func collectEvents(cfg ThalamicConfig) (*ThalamicFilter, *[]TelemetryEvent) {
	events := &[]TelemetryEvent{}
	f := NewThalamicFilter(cfg, func(e TelemetryEvent) {
		*events = append(*events, e)
	})
	return f, events
}

// TestThalamicFilter_HeartbeatInHomeostasis verifica que o filtro emite
// heartbeat em homeostase após o intervalo configurado.
func TestThalamicFilter_HeartbeatInHomeostasis(t *testing.T) {
	cfg := ThalamicConfig{HeartbeatInterval: 10 * time.Millisecond}
	f, events := collectEvents(cfg)

	// Força o lastHeartbeat para o passado
	f.lastHeartbeat = time.Now().Add(-20 * time.Millisecond)

	f.Observe(LevelHomeostasis, 1.0, 0.0)

	if len(*events) != 1 {
		t.Fatalf("esperado 1 evento (heartbeat), obtido %d", len(*events))
	}
	if (*events)[0].Type != EventHeartbeat {
		t.Errorf("tipo esperado EventHeartbeat, obtido %v", (*events)[0].Type)
	}
}

// TestThalamicFilter_SuppressesHomeostasisLogs verifica que nenhum evento é
// emitido em homeostase antes do intervalo de heartbeat expirar.
func TestThalamicFilter_SuppressesHomeostasisLogs(t *testing.T) {
	cfg := ThalamicConfig{HeartbeatInterval: 1 * time.Hour} // intervalo longo
	f, events := collectEvents(cfg)

	// Múltiplos ciclos em homeostase — nenhum deve gerar evento
	for i := 0; i < 10; i++ {
		f.Observe(LevelHomeostasis, 1.0+float64(i)*0.01, 0.0)
	}

	if len(*events) != 0 {
		t.Errorf("esperado 0 eventos em homeostase (heartbeat suprimido), obtido %d", len(*events))
	}
}

// TestThalamicFilter_AnomalyDetectedOnEscalation verifica que a transição
// homeostase → vigilância emite EventAnomalyDetected imediatamente.
func TestThalamicFilter_AnomalyDetectedOnEscalation(t *testing.T) {
	cfg := ThalamicConfig{HeartbeatInterval: 1 * time.Hour}
	f, events := collectEvents(cfg)

	f.Observe(LevelVigilance, 3.8, 1.2)

	if len(*events) != 1 {
		t.Fatalf("esperado 1 evento (anomalia), obtido %d", len(*events))
	}
	if (*events)[0].Type != EventAnomalyDetected {
		t.Errorf("tipo esperado EventAnomalyDetected, obtido %v", (*events)[0].Type)
	}
	if (*events)[0].Level != LevelVigilance {
		t.Errorf("nível esperado LevelVigilance, obtido %d", (*events)[0].Level)
	}
}

// TestThalamicFilter_LevelChangeEvent verifica emissão em transições de nível
// durante um episódio de estresse (vigilância → contenção).
func TestThalamicFilter_LevelChangeEvent(t *testing.T) {
	cfg := ThalamicConfig{HeartbeatInterval: 1 * time.Hour}
	f, events := collectEvents(cfg)

	f.Observe(LevelVigilance, 3.8, 1.2)   // anomaly_detected
	f.Observe(LevelContainment, 5.6, 2.1)  // level_change

	if len(*events) != 2 {
		t.Fatalf("esperado 2 eventos, obtido %d", len(*events))
	}
	if (*events)[1].Type != EventLevelChange {
		t.Errorf("segundo evento: esperado EventLevelChange, obtido %v", (*events)[1].Type)
	}
}

// TestThalamicFilter_HomeostasisRestoredEvent verifica emissão quando o sistema
// retorna à homeostase após um episódio de estresse.
func TestThalamicFilter_HomeostasisRestoredEvent(t *testing.T) {
	cfg := ThalamicConfig{HeartbeatInterval: 1 * time.Hour}
	f, events := collectEvents(cfg)

	f.Observe(LevelVigilance, 3.8, 1.2)    // anomaly_detected
	f.Observe(LevelHomeostasis, 2.1, -0.5) // homeostasis_restored

	if len(*events) != 2 {
		t.Fatalf("esperado 2 eventos, obtido %d", len(*events))
	}
	if (*events)[1].Type != EventHomeostasisRestored {
		t.Errorf("segundo evento: esperado EventHomeostasisRestored, obtido %v", (*events)[1].Type)
	}
}

// TestThalamicFilter_NotifyContainment verifica que NotifyContainment sempre emite.
func TestThalamicFilter_NotifyContainment(t *testing.T) {
	cfg := ThalamicConfig{HeartbeatInterval: 1 * time.Hour}
	f, events := collectEvents(cfg)

	f.NotifyContainment(LevelContainment, 5.6, "memory.high=6GB")

	if len(*events) != 1 {
		t.Fatalf("esperado 1 evento de contenção, obtido %d", len(*events))
	}
	if (*events)[0].Type != EventContainmentApplied {
		t.Errorf("tipo esperado EventContainmentApplied, obtido %v", (*events)[0].Type)
	}
}

// TestThalamicFilter_NoLevelChangeOnSameLevel verifica que ciclos no mesmo nível
// (acima de homeostase) não geram eventos repetidos de level_change.
func TestThalamicFilter_NoLevelChangeOnSameLevel(t *testing.T) {
	cfg := ThalamicConfig{HeartbeatInterval: 1 * time.Hour}
	f, events := collectEvents(cfg)

	f.Observe(LevelVigilance, 3.8, 1.2) // anomaly_detected
	f.Observe(LevelVigilance, 3.6, 0.8) // mesmo nível — sem novo evento
	f.Observe(LevelVigilance, 3.4, 0.6) // mesmo nível — sem novo evento

	if len(*events) != 1 {
		t.Errorf("esperado apenas 1 evento (anomaly_detected), obtido %d", len(*events))
	}
}

// TestTelemetryEventType_String verifica os nomes dos tipos de evento.
func TestTelemetryEventType_String(t *testing.T) {
	cases := []struct {
		t        TelemetryEventType
		expected string
	}{
		{EventHeartbeat, "heartbeat"},
		{EventAnomalyDetected, "anomaly_detected"},
		{EventLevelChange, "level_change"},
		{EventContainmentApplied, "containment_applied"},
		{EventHomeostasisRestored, "homeostasis_restored"},
		{EventProtectionApplied, "protection_applied"},
	}
	for _, c := range cases {
		if got := c.t.String(); got != c.expected {
			t.Errorf("TelemetryEventType(%d).String() = %q, esperado %q", c.t, got, c.expected)
		}
	}
}