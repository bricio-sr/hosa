package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Server exposes /metrics (Prometheus text) and /healthz (JSON) over HTTP.
// It is non-blocking: Start() launches a goroutine; Stop() shuts it down gracefully.
type Server struct {
	addr  string
	state *AtomicState
	srv   *http.Server
}

// NewServer creates a Server bound to addr that reads from state.
func NewServer(addr string, state *AtomicState) *Server {
	s := &Server{addr: addr, state: state}

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.NotFound(w, r)
	})

	s.srv = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	return s
}

// Start binds the listener and serves in a background goroutine.
// Returns an error immediately if the address cannot be bound.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("telemetry.Server: listen %s: %w", s.addr, err)
	}
	go func() { _ = s.srv.Serve(ln) }()
	return nil
}

// Stop gracefully shuts down the HTTP server, waiting at most for ctx.
func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

// handleMetrics writes a Prometheus text-format response.
// No external client library — manual format per the Prometheus exposition spec.
func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	snap := s.state.Get()
	var b strings.Builder

	writeMetric(&b, "hosa_dm_stress", "gauge",
		"Smoothed Mahalanobis distance (D̄_M)",
		snap.DM)
	writeMetric(&b, "hosa_dm_dot", "gauge",
		"Rate of change of D̄_M per second (dD̄_M/dt)",
		snap.DMDot)
	writeMetric(&b, "hosa_alert_level", "gauge",
		"Current alert level (0=homeostasis … 4=survival)",
		float64(snap.Level))
	writeMetric(&b, "hosa_h_frag_norm", "gauge",
		"Normalized memory fragmentation entropy H_frag [0,1]",
		snap.HFragNorm)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, b.String())
}

func writeMetric(b *strings.Builder, name, typ, help string, value float64) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s %s\n", name, typ)
	fmt.Fprintf(b, "%s %.4f\n", name, value)
}

// healthzResponse is the JSON payload for /healthz.
type healthzResponse struct {
	Status      string    `json:"status"`
	Level       int       `json:"level"`
	DM          float64   `json:"dm"`
	DMDot       float64   `json:"dm_dot"`
	HFragNorm   float64   `json:"h_frag_norm"`
	StateVector []float64 `json:"state_vector"`
	Timestamp   time.Time `json:"timestamp"`
}

var levelNames = [5]string{"homeostasis", "vigilance", "containment", "protection", "survival"}

func levelName(l int) string {
	if l >= 0 && l < len(levelNames) {
		return levelNames[l]
	}
	return "unknown"
}

// handleHealthz writes a JSON snapshot of the current agent state.
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	snap := s.state.Get()
	resp := healthzResponse{
		Status:      levelName(snap.Level),
		Level:       snap.Level,
		DM:          snap.DM,
		DMDot:       snap.DMDot,
		HFragNorm:   snap.HFragNorm,
		StateVector: snap.StateVec,
		Timestamp:   snap.UpdatedAt,
	}
	if resp.Timestamp.IsZero() {
		resp.Timestamp = time.Now()
	}
	if resp.StateVector == nil {
		resp.StateVector = []float64{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
