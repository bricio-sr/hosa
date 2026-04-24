package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// WebhookClient fires a fire-and-forget HTTP POST when stress is escalating.
// It sends at most one notification per escalation edge (level increase), so
// repeated high-stress ticks do not flood the target endpoint.
type WebhookClient struct {
	url       string
	threshold float64 // DMDot threshold to fire (from config)
	client    *http.Client
	mu        sync.Mutex
	lastLevel int // last level we fired for; -1 = never fired
}

// NewWebhookClient returns a client that fires when DMDot > thresholdDMDot
// and the alert level has increased since the last notification.
func NewWebhookClient(url string, thresholdDMDot float64) *WebhookClient {
	return &WebhookClient{
		url:       url,
		threshold: thresholdDMDot,
		client:    &http.Client{Timeout: 5 * time.Second},
		lastLevel: -1,
	}
}

// webhookPayload is the JSON body sent to the K8s HPA/KEDA endpoint.
type webhookPayload struct {
	Event     string    `json:"event"`
	DM        float64   `json:"dm"`
	DMDot     float64   `json:"dm_dot"`
	Level     int       `json:"level"`
	Node      string    `json:"node"`
	Timestamp time.Time `json:"timestamp"`
}

// Notify fires a POST if the current snapshot represents a new escalation edge.
// It is non-blocking: the HTTP call happens in a separate goroutine.
func (w *WebhookClient) Notify(snap Snapshot) {
	if snap.DMDot <= w.threshold {
		return
	}

	w.mu.Lock()
	if snap.Level <= w.lastLevel {
		w.mu.Unlock()
		return
	}
	w.lastLevel = snap.Level
	w.mu.Unlock()

	go w.send(snap)
}

func (w *WebhookClient) send(snap Snapshot) {
	host, _ := os.Hostname()
	payload := webhookPayload{
		Event:     "stress_escalating",
		DM:        snap.DM,
		DMDot:     snap.DMDot,
		Level:     snap.Level,
		Node:      host,
		Timestamp: snap.UpdatedAt,
	}
	if payload.Timestamp.IsZero() {
		payload.Timestamp = time.Now()
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	resp, err := w.client.Post(w.url, "application/json", bytes.NewReader(body))
	if err != nil {
		// fire-and-forget — log nothing; caller has structured logging
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = fmt.Sprintf("webhook: unexpected status %d", resp.StatusCode)
	}
}
