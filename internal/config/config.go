package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

const (
	// DefaultConfigPath is the standard Linux system config location.
	// Follows /etc/<package>/<package>.conf convention for daemon packages.
	DefaultConfigPath = "/etc/hosa/hosa.toml"
)

// Config holds all tunable parameters of the HOSA agent.
// Precedence: hardcoded defaults → TOML file → CLI flags.
type Config struct {
	Detection DetectionConfig
	Sampling  SamplingConfig
	Motor     MotorConfig
	Thalamus  ThalamicConfig
}

// DetectionConfig controls the Predictive Cortex behavior.
type DetectionConfig struct {
	// ThresholdVigilance is the D̄_M value that triggers Level 1 (Vigilance).
	// Calibrated for p=4 state vector (chi-squared(4) expectation ≈ 2.0).
	ThresholdVigilance float64

	// ThresholdContainment triggers Level 2 (Containment) — cgroups applied.
	ThresholdContainment float64

	// ThresholdProtection triggers Level 3 (Protection) — maximum containment.
	ThresholdProtection float64

	// AlphaEWMA is the global EWMA smoothing factor (0 < α ≤ 1).
	// Higher = more responsive but noisier. Lower = smoother but slower to detect.
	AlphaEWMA float64

	// AlphaPerProbe overrides AlphaEWMA per sensor probe.
	// CPU probes benefit from higher alpha (sudden spikes);
	// memory probes benefit from lower alpha (gradual leaks).
	AlphaPerProbe AlphaPerProbeConfig

	// MinSamples is the warm-up period before the cortex enables analysis.
	MinSamples int

	// ThresholdDerivativeEscalate: if dD̄_M/dt exceeds this, escalate one extra level.
	ThresholdDerivativeEscalate float64

	// ThresholdDerivativeRelax: below this, hysteresis counter advances toward descent.
	ThresholdDerivativeRelax float64

	// HysteresisDown is how many consecutive cycles below threshold before descending.
	HysteresisDown int
}

// AlphaPerProbeConfig allows per-probe EWMA tuning.
// CPU benefits from higher alpha (reacts to sudden spikes).
// Memory/IO benefits from lower alpha (smooths out gradual trends).
type AlphaPerProbeConfig struct {
	CPURunQueue   float64
	MemBrkCalls   float64
	MemPageFaults float64
	IOBlockOps    float64
}

// SamplingConfig controls collection frequency.
type SamplingConfig struct {
	// NormalIntervalMs is the collection interval in homeostasis (milliseconds).
	NormalIntervalMs int

	// VigilanceIntervalMs is the collection interval during anomaly (milliseconds).
	VigilanceIntervalMs int
}

// MotorConfig controls the containment actuator.
type MotorConfig struct {
	// CgroupPath is the cgroup v2 directory managed by HOSA.
	CgroupPath string

	// ContainmentFraction is the memory.high limit as a fraction of total RAM (Level 2).
	ContainmentFraction float64

	// ProtectionHighFrac is the memory.high limit as a fraction of total RAM (Level 3).
	ProtectionHighFrac float64

	// ProtectionMaxFrac is the memory.max limit as a fraction of total RAM (Level 3).
	ProtectionMaxFrac float64
}

// ThalamicConfig controls telemetry emission.
type ThalamicConfig struct {
	// HeartbeatIntervalS is the period between heartbeats in homeostasis (seconds).
	HeartbeatIntervalS int
}

// HeartbeatInterval returns the heartbeat interval as a time.Duration.
func (t ThalamicConfig) HeartbeatInterval() time.Duration {
	return time.Duration(t.HeartbeatIntervalS) * time.Second
}

// NormalInterval returns the normal sampling interval as a time.Duration.
func (s SamplingConfig) NormalInterval() time.Duration {
	return time.Duration(s.NormalIntervalMs) * time.Millisecond
}

// VigilanceInterval returns the vigilance sampling interval as a time.Duration.
func (s SamplingConfig) VigilanceInterval() time.Duration {
	return time.Duration(s.VigilanceIntervalMs) * time.Millisecond
}

// Default returns the recommended default configuration.
// These values are calibrated for a p=4 state vector on a typical Linux server.
// Operators should override via /etc/hosa/hosa.toml for their specific workload.
func Default() Config {
	return Config{
		Detection: DetectionConfig{
			ThresholdVigilance:          3.5,
			ThresholdContainment:        5.5,
			ThresholdProtection:         8.0,
			AlphaEWMA:                   0.2,
			MinSamples:                  30,
			ThresholdDerivativeEscalate: 2.0,
			ThresholdDerivativeRelax:    0.5,
			HysteresisDown:              5,
			AlphaPerProbe: AlphaPerProbeConfig{
				CPURunQueue:   0.3,  // CPU spikes are sudden — more responsive
				MemBrkCalls:   0.2,  // Memory leaks are gradual — smoother
				MemPageFaults: 0.2,
				IOBlockOps:    0.15, // I/O is the noisiest — most smoothing
			},
		},
		Sampling: SamplingConfig{
			NormalIntervalMs:    1000,
			VigilanceIntervalMs: 100,
		},
		Motor: MotorConfig{
			CgroupPath:          "/sys/fs/cgroup/hosa",
			ContainmentFraction: 0.75,
			ProtectionHighFrac:  0.50,
			ProtectionMaxFrac:   0.90,
		},
		Thalamus: ThalamicConfig{
			HeartbeatIntervalS: 30,
		},
	}
}

// Load reads the TOML config file and returns a Config with file values
// merged over the defaults. Missing keys keep their default values.
// Returns default config (with a warning) if the file does not exist.
func Load(path string) (Config, error) {
	cfg := Default()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// No config file is not an error — use defaults silently.
		return cfg, nil
	}

	t, err := parseTOML(path)
	if err != nil {
		return cfg, fmt.Errorf("config.Load: %w", err)
	}

	// Detection
	cfg.Detection.ThresholdVigilance = t.getFloat("detection.threshold_vigilance", cfg.Detection.ThresholdVigilance)
	cfg.Detection.ThresholdContainment = t.getFloat("detection.threshold_containment", cfg.Detection.ThresholdContainment)
	cfg.Detection.ThresholdProtection = t.getFloat("detection.threshold_protection", cfg.Detection.ThresholdProtection)
	cfg.Detection.AlphaEWMA = t.getFloat("detection.alpha_ewma", cfg.Detection.AlphaEWMA)
	cfg.Detection.MinSamples = t.getInt("detection.min_samples", cfg.Detection.MinSamples)
	cfg.Detection.ThresholdDerivativeEscalate = t.getFloat("detection.threshold_derivative_escalate", cfg.Detection.ThresholdDerivativeEscalate)
	cfg.Detection.ThresholdDerivativeRelax = t.getFloat("detection.threshold_derivative_relax", cfg.Detection.ThresholdDerivativeRelax)
	cfg.Detection.HysteresisDown = t.getInt("detection.hysteresis_down", cfg.Detection.HysteresisDown)

	// Per-probe alpha
	cfg.Detection.AlphaPerProbe.CPURunQueue = t.getFloat("detection.alpha_per_probe.cpu_run_queue", cfg.Detection.AlphaPerProbe.CPURunQueue)
	cfg.Detection.AlphaPerProbe.MemBrkCalls = t.getFloat("detection.alpha_per_probe.mem_brk_calls", cfg.Detection.AlphaPerProbe.MemBrkCalls)
	cfg.Detection.AlphaPerProbe.MemPageFaults = t.getFloat("detection.alpha_per_probe.mem_page_faults", cfg.Detection.AlphaPerProbe.MemPageFaults)
	cfg.Detection.AlphaPerProbe.IOBlockOps = t.getFloat("detection.alpha_per_probe.io_block_ops", cfg.Detection.AlphaPerProbe.IOBlockOps)

	// Sampling
	cfg.Sampling.NormalIntervalMs = t.getInt("sampling.normal_interval_ms", cfg.Sampling.NormalIntervalMs)
	cfg.Sampling.VigilanceIntervalMs = t.getInt("sampling.vigilance_interval_ms", cfg.Sampling.VigilanceIntervalMs)

	// Motor
	cfg.Motor.CgroupPath = t.getString("motor.cgroup_path", cfg.Motor.CgroupPath)
	cfg.Motor.ContainmentFraction = t.getFloat("motor.containment_fraction", cfg.Motor.ContainmentFraction)
	cfg.Motor.ProtectionHighFrac = t.getFloat("motor.protection_high_frac", cfg.Motor.ProtectionHighFrac)
	cfg.Motor.ProtectionMaxFrac = t.getFloat("motor.protection_max_frac", cfg.Motor.ProtectionMaxFrac)

	// Thalamus
	cfg.Thalamus.HeartbeatIntervalS = t.getInt("thalamus.heartbeat_interval_s", cfg.Thalamus.HeartbeatIntervalS)

	return cfg, nil
}

// ApplyCLIFlags registers CLI flags and merges them over cfg after parsing.
// Call this after Load() — flags always win.
//
// Usage:
//
//	cfg, _ := config.Load(path)
//	cfg = cfg.ApplyCLIFlags()
func (cfg Config) ApplyCLIFlags() Config {
	// Detection
	flagFloat("threshold-vigilance", &cfg.Detection.ThresholdVigilance, "D_M threshold for Level 1 (Vigilance)")
	flagFloat("threshold-containment", &cfg.Detection.ThresholdContainment, "D_M threshold for Level 2 (Containment)")
	flagFloat("threshold-protection", &cfg.Detection.ThresholdProtection, "D_M threshold for Level 3 (Protection)")
	flagFloat("alpha", &cfg.Detection.AlphaEWMA, "EWMA smoothing factor (0 < α ≤ 1)")
	flagInt("min-samples", &cfg.Detection.MinSamples, "Warm-up samples before analysis is enabled")
	flagInt("hysteresis", &cfg.Detection.HysteresisDown, "Cycles below threshold before descending a level")

	// Sampling
	flagInt("normal-interval-ms", &cfg.Sampling.NormalIntervalMs, "Sampling interval in homeostasis (ms)")
	flagInt("vigilance-interval-ms", &cfg.Sampling.VigilanceIntervalMs, "Sampling interval during anomaly (ms)")

	// Motor
	flagString("cgroup-path", &cfg.Motor.CgroupPath, "cgroup v2 path managed by HOSA")

	// Thalamus
	flagInt("heartbeat-interval-s", &cfg.Thalamus.HeartbeatIntervalS, "Heartbeat interval in homeostasis (s)")

	flag.Parse()
	return cfg
}

// Validate checks that the configuration is internally consistent.
func (cfg Config) Validate() error {
	d := cfg.Detection

	if d.ThresholdVigilance <= 0 {
		return fmt.Errorf("config: detection.threshold_vigilance must be > 0")
	}
	if d.ThresholdContainment <= d.ThresholdVigilance {
		return fmt.Errorf("config: threshold_containment (%.2f) must be > threshold_vigilance (%.2f)",
			d.ThresholdContainment, d.ThresholdVigilance)
	}
	if d.ThresholdProtection <= d.ThresholdContainment {
		return fmt.Errorf("config: threshold_protection (%.2f) must be > threshold_containment (%.2f)",
			d.ThresholdProtection, d.ThresholdContainment)
	}
	if d.AlphaEWMA <= 0 || d.AlphaEWMA > 1 {
		return fmt.Errorf("config: alpha_ewma must be in (0, 1], got %.3f", d.AlphaEWMA)
	}
	if d.MinSamples < 2 {
		return fmt.Errorf("config: min_samples must be >= 2, got %d", d.MinSamples)
	}

	s := cfg.Sampling
	if s.NormalIntervalMs <= 0 {
		return fmt.Errorf("config: sampling.normal_interval_ms must be > 0")
	}
	if s.VigilanceIntervalMs <= 0 || s.VigilanceIntervalMs > s.NormalIntervalMs {
		return fmt.Errorf("config: vigilance_interval_ms must be > 0 and <= normal_interval_ms")
	}

	m := cfg.Motor
	if m.ContainmentFraction <= 0 || m.ContainmentFraction >= 1 {
		return fmt.Errorf("config: motor.containment_fraction must be in (0, 1)")
	}
	if m.ProtectionHighFrac >= m.ContainmentFraction {
		return fmt.Errorf("config: protection_high_frac must be < containment_fraction")
	}

	return nil
}

// Summary returns a one-line human-readable summary for boot logging.
func (cfg Config) Summary() string {
	d := cfg.Detection
	s := cfg.Sampling
	return fmt.Sprintf(
		"thresholds=[%.1f/%.1f/%.1f] alpha=%.2f min_samples=%d normal=%dms vigilance=%dms heartbeat=%ds",
		d.ThresholdVigilance, d.ThresholdContainment, d.ThresholdProtection,
		d.AlphaEWMA, d.MinSamples,
		s.NormalIntervalMs, s.VigilanceIntervalMs,
		cfg.Thalamus.HeartbeatIntervalS,
	)
}

// flagFloat registers a float64 flag that overrides the current value if set.
func flagFloat(name string, dest *float64, usage string) {
	flag.Float64Var(dest, name, *dest, usage)
}

// flagInt registers an int flag that overrides the current value if set.
func flagInt(name string, dest *int, usage string) {
	flag.IntVar(dest, name, *dest, usage)
}

// flagString registers a string flag that overrides the current value if set.
func flagString(name string, dest *string, usage string) {
	flag.StringVar(dest, name, *dest, usage)
}