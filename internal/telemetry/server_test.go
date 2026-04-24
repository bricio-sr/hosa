package telemetry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// newTestServer returns a Server wired to an httptest.Server for in-process testing.
// The caller is responsible for closing the httptest.Server.
func newTestServer(state *AtomicState) (*Server, *httptest.Server) {
	s := NewServer(":0", state)
	ts := httptest.NewServer(s.srv.Handler)
	return s, ts
}

// TestMetricsFormat verifies that /metrics emits valid Prometheus text format
// with the expected metric names and values.
func TestMetricsFormat(t *testing.T) {
	state := &AtomicState{}
	state.Set(Snapshot{
		DM:        2.34,
		DMDot:     0.12,
		Level:     1,
		HFragNorm: 0.45,
		UpdatedAt: time.Now(),
	})

	_, ts := newTestServer(state)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("unexpected Content-Type: %q", ct)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	body := string(bodyBytes)

	mustContain := []string{
		"# HELP hosa_dm_stress",
		"# TYPE hosa_dm_stress gauge",
		"hosa_dm_stress 2.3400",
		"# HELP hosa_dm_dot",
		"# TYPE hosa_dm_dot gauge",
		"hosa_dm_dot 0.1200",
		"# HELP hosa_alert_level",
		"# TYPE hosa_alert_level gauge",
		"hosa_alert_level 1.0000",
		"# HELP hosa_h_frag_norm",
		"# TYPE hosa_h_frag_norm gauge",
		"hosa_h_frag_norm 0.4500",
	}
	for _, want := range mustContain {
		if !strings.Contains(body, want) {
			t.Errorf("/metrics missing %q\nFull body:\n%s", want, body)
		}
	}
}

// TestHealthzJSON verifies that /healthz returns valid JSON with the expected fields.
func TestHealthzJSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	state := &AtomicState{}
	state.Set(Snapshot{
		DM:        5.10,
		DMDot:     -0.30,
		Level:     2,
		HFragNorm: 0.60,
		StateVec:  []float64{0.01, 0.02, 0.00, 0.03},
		UpdatedAt: now,
	})

	_, ts := newTestServer(state)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("unexpected Content-Type: %q", ct)
	}

	var payload healthzResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}

	if payload.Status != "containment" {
		t.Errorf("status: got %q, want %q", payload.Status, "containment")
	}
	if payload.Level != 2 {
		t.Errorf("level: got %d, want 2", payload.Level)
	}
	if fmt.Sprintf("%.2f", payload.DM) != "5.10" {
		t.Errorf("dm: got %.4f, want 5.10", payload.DM)
	}
	if fmt.Sprintf("%.2f", payload.DMDot) != "-0.30" {
		t.Errorf("dm_dot: got %.4f, want -0.30", payload.DMDot)
	}
	if fmt.Sprintf("%.2f", payload.HFragNorm) != "0.60" {
		t.Errorf("h_frag_norm: got %.4f, want 0.60", payload.HFragNorm)
	}
	if len(payload.StateVector) != 4 {
		t.Errorf("state_vector len: got %d, want 4", len(payload.StateVector))
	}
}

// TestHealthzEmptyStateVector ensures state_vector is [] not null when StateVec is nil.
func TestHealthzEmptyStateVector(t *testing.T) {
	state := &AtomicState{}
	state.Set(Snapshot{Level: 0, UpdatedAt: time.Now()})

	_, ts := newTestServer(state)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	sv, ok := raw["state_vector"]
	if !ok {
		t.Fatal("state_vector missing from response")
	}
	if string(sv) == "null" {
		t.Error("state_vector must be [] not null when StateVec is nil")
	}
}

// TestUnknownRoute404 ensures unknown paths return 404.
func TestUnknownRoute404(t *testing.T) {
	state := &AtomicState{}
	_, ts := newTestServer(state)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/unknown")
	if err != nil {
		t.Fatalf("GET /unknown: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// TestConcurrentReads exercises concurrent /metrics and /healthz reads against
// a state that is being updated concurrently. Run with -race.
func TestConcurrentReads(t *testing.T) {
	state := &AtomicState{}
	state.Set(Snapshot{DM: 1.0, Level: 0, UpdatedAt: time.Now()})

	_, ts := newTestServer(state)
	defer ts.Close()

	var wg sync.WaitGroup
	const goroutines = 20

	// writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			state.Set(Snapshot{DM: float64(i), Level: i % 5, UpdatedAt: time.Now()})
		}
	}()

	// readers
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			path := "/metrics"
			if i%2 == 0 {
				path = "/healthz"
			}
			resp, err := http.Get(ts.URL + path)
			if err != nil {
				t.Errorf("GET %s: %v", path, err)
				return
			}
			resp.Body.Close()
		}(i)
	}

	wg.Wait()
}

// TestLevelNames checks that all five known levels produce the expected strings.
func TestLevelNames(t *testing.T) {
	cases := []struct {
		level int
		want  string
	}{
		{0, "homeostasis"},
		{1, "vigilance"},
		{2, "containment"},
		{3, "protection"},
		{4, "survival"},
		{99, "unknown"},
	}
	for _, c := range cases {
		if got := levelName(c.level); got != c.want {
			t.Errorf("levelName(%d) = %q, want %q", c.level, got, c.want)
		}
	}
}

// TestWebhookFires checks that Notify fires a POST on a new escalation edge.
func TestWebhookFires(t *testing.T) {
	fired := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fired <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wh := NewWebhookClient(srv.URL, 2.0)
	snap := Snapshot{DM: 5.0, DMDot: 3.0, Level: 2, UpdatedAt: time.Now()}
	wh.Notify(snap)

	select {
	case <-fired:
	case <-time.After(2 * time.Second):
		t.Fatal("webhook POST not received within 2s")
	}
}

// TestWebhookNoFireBelowThreshold checks that Notify does not fire when DMDot is low.
func TestWebhookNoFireBelowThreshold(t *testing.T) {
	fired := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fired <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wh := NewWebhookClient(srv.URL, 2.0)
	// DMDot = 1.0 which is <= threshold 2.0 — should NOT fire
	snap := Snapshot{DM: 5.0, DMDot: 1.0, Level: 2, UpdatedAt: time.Now()}
	wh.Notify(snap)

	select {
	case <-fired:
		t.Fatal("webhook should not have fired below threshold")
	case <-time.After(200 * time.Millisecond):
	}
}

// TestWebhookNoDuplicateFire checks that the same level does not fire twice.
func TestWebhookNoDuplicateFire(t *testing.T) {
	count := 0
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wh := NewWebhookClient(srv.URL, 2.0)
	snap := Snapshot{DM: 5.0, DMDot: 3.0, Level: 2, UpdatedAt: time.Now()}
	wh.Notify(snap)
	wh.Notify(snap) // duplicate — same level, should not fire again

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Errorf("expected 1 webhook POST, got %d", count)
	}
}
