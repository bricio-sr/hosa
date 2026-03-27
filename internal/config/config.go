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
type AlphaPerProbeConfig struct {
	CPURunQueue   float64
	MemBrkCalls   float64
	MemPageFaults float64
	IOBlockOps    float64
}

// SamplingConfig controls collection frequency.
type SamplingConfig struct {
	NormalIntervalMs    int
	VigilanceIntervalMs int
}

// MotorConfig controls the containment actuator.
type MotorConfig struct {
	CgroupPath          string
	ContainmentFraction float64
	ProtectionHighFrac  float64
	ProtectionMaxFrac   float64
}

// ThalamicConfig controls telemetry emission.
type ThalamicConfig struct {
	HeartbeatIntervalS int
}

func (t ThalamicConfig) HeartbeatInterval() time.Duration {
	return time.Duration(t.HeartbeatIntervalS) * time.Second
}

func (s SamplingConfig) NormalInterval() time.Duration {
	return time.Duration(s.NormalIntervalMs) * time.Millisecond
}

func (s SamplingConfig) VigilanceInterval() time.Duration {
	return time.Duration(s.VigilanceIntervalMs) * time.Millisecond
}

// Default returns the recommended default configuration.
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
				CPURunQueue:   0.3,
				MemBrkCalls:   0.2,
				MemPageFaults: 0.2,
				IOBlockOps:    0.15,
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

// Load reads the TOML config file and merges values over the defaults.
// Returns default config if the file does not exist.
func Load(path string) (Config, error) {
	cfg := Default()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	t, err := parseTOML(path)
	if err != nil {
		return cfg, fmt.Errorf("config.Load: %w", err)
	}

	cfg.Detection.ThresholdVigilance = t.getFloat("detection.threshold_vigilance", cfg.Detection.ThresholdVigilance)
	cfg.Detection.ThresholdContainment = t.getFloat("detection.threshold_containment", cfg.Detection.ThresholdContainment)
	cfg.Detection.ThresholdProtection = t.getFloat("detection.threshold_protection", cfg.Detection.ThresholdProtection)
	cfg.Detection.AlphaEWMA = t.getFloat("detection.alpha_ewma", cfg.Detection.AlphaEWMA)
	cfg.Detection.MinSamples = t.getInt("detection.min_samples", cfg.Detection.MinSamples)
	cfg.Detection.ThresholdDerivativeEscalate = t.getFloat("detection.threshold_derivative_escalate", cfg.Detection.ThresholdDerivativeEscalate)
	cfg.Detection.ThresholdDerivativeRelax = t.getFloat("detection.threshold_derivative_relax", cfg.Detection.ThresholdDerivativeRelax)
	cfg.Detection.HysteresisDown = t.getInt("detection.hysteresis_down", cfg.Detection.HysteresisDown)

	cfg.Detection.AlphaPerProbe.CPURunQueue = t.getFloat("detection.alpha_per_probe.cpu_run_queue", cfg.Detection.AlphaPerProbe.CPURunQueue)
	cfg.Detection.AlphaPerProbe.MemBrkCalls = t.getFloat("detection.alpha_per_probe.mem_brk_calls", cfg.Detection.AlphaPerProbe.MemBrkCalls)
	cfg.Detection.AlphaPerProbe.MemPageFaults = t.getFloat("detection.alpha_per_probe.mem_page_faults", cfg.Detection.AlphaPerProbe.MemPageFaults)
	cfg.Detection.AlphaPerProbe.IOBlockOps = t.getFloat("detection.alpha_per_probe.io_block_ops", cfg.Detection.AlphaPerProbe.IOBlockOps)

	cfg.Sampling.NormalIntervalMs = t.getInt("sampling.normal_interval_ms", cfg.Sampling.NormalIntervalMs)
	cfg.Sampling.VigilanceIntervalMs = t.getInt("sampling.vigilance_interval_ms", cfg.Sampling.VigilanceIntervalMs)

	cfg.Motor.CgroupPath = t.getString("motor.cgroup_path", cfg.Motor.CgroupPath)
	cfg.Motor.ContainmentFraction = t.getFloat("motor.containment_fraction", cfg.Motor.ContainmentFraction)
	cfg.Motor.ProtectionHighFrac = t.getFloat("motor.protection_high_frac", cfg.Motor.ProtectionHighFrac)
	cfg.Motor.ProtectionMaxFrac = t.getFloat("motor.protection_max_frac", cfg.Motor.ProtectionMaxFrac)

	cfg.Thalamus.HeartbeatIntervalS = t.getInt("thalamus.heartbeat_interval_s", cfg.Thalamus.HeartbeatIntervalS)

	return cfg, nil
}

// LoadWithFlags is the single entry point for configuration in main().
// It handles the full precedence chain in one call:
//
//  1. Start with hardcoded defaults
//  2. Register --config flag and all tunable flags (with defaults from step 1)
//  3. Parse CLI flags once
//  4. If --config points to a non-default path, reload the TOML and re-apply
//     CLI flags on top (so CLI always wins over file)
//
// Usage in main():
//
//	cfg, err := config.LoadWithFlags()
func LoadWithFlags() (Config, error) {
	defaults := Default()

	// Register --config first so it's available in flag.Parse().
	configPath := flag.String("config", DefaultConfigPath, "path to TOML config file")

	// Register all tunable flags with hardcoded defaults.
	// After flag.Parse() these will hold either the default or whatever the
	// user passed on the CLI.
	thresholdVigilance          := flag.Float64("threshold-vigilance", defaults.Detection.ThresholdVigilance, "D_M threshold for Level 1 (Vigilance)")
	thresholdContainment        := flag.Float64("threshold-containment", defaults.Detection.ThresholdContainment, "D_M threshold for Level 2 (Containment)")
	thresholdProtection         := flag.Float64("threshold-protection", defaults.Detection.ThresholdProtection, "D_M threshold for Level 3 (Protection)")
	alpha                       := flag.Float64("alpha", defaults.Detection.AlphaEWMA, "EWMA smoothing factor (0 < α ≤ 1)")
	minSamples                  := flag.Int("min-samples", defaults.Detection.MinSamples, "Warm-up samples before analysis is enabled")
	hysteresis                  := flag.Int("hysteresis", defaults.Detection.HysteresisDown, "Cycles below threshold before descending a level")
	normalIntervalMs            := flag.Int("normal-interval-ms", defaults.Sampling.NormalIntervalMs, "Sampling interval in homeostasis (ms)")
	vigilanceIntervalMs         := flag.Int("vigilance-interval-ms", defaults.Sampling.VigilanceIntervalMs, "Sampling interval during anomaly (ms)")
	cgroupPath                  := flag.String("cgroup-path", defaults.Motor.CgroupPath, "cgroup v2 path managed by HOSA")
	heartbeatIntervalS          := flag.Int("heartbeat-interval-s", defaults.Thalamus.HeartbeatIntervalS, "Heartbeat interval in homeostasis (s)")

	// Single flag.Parse() for the entire program.
	flag.Parse()

	// Load TOML (uses defaults if file doesn't exist).
	cfg, err := Load(*configPath)
	if err != nil {
		return cfg, err
	}

	// Apply CLI flags on top of TOML values — but only for flags the user
	// explicitly passed. We detect this by comparing against the hardcoded
	// default: if a flag value differs from the hardcoded default, the user
	// set it on the CLI and it wins.
	//
	// Note: this means a CLI flag that happens to equal the hardcoded default
	// won't override a different TOML value — which is the correct behavior
	// (you can't distinguish "user passed the default" from "not passed").
	// For that edge case, users should edit the TOML file directly.
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "threshold-vigilance":
			cfg.Detection.ThresholdVigilance = *thresholdVigilance
		case "threshold-containment":
			cfg.Detection.ThresholdContainment = *thresholdContainment
		case "threshold-protection":
			cfg.Detection.ThresholdProtection = *thresholdProtection
		case "alpha":
			cfg.Detection.AlphaEWMA = *alpha
		case "min-samples":
			cfg.Detection.MinSamples = *minSamples
		case "hysteresis":
			cfg.Detection.HysteresisDown = *hysteresis
		case "normal-interval-ms":
			cfg.Sampling.NormalIntervalMs = *normalIntervalMs
		case "vigilance-interval-ms":
			cfg.Sampling.VigilanceIntervalMs = *vigilanceIntervalMs
		case "cgroup-path":
			cfg.Motor.CgroupPath = *cgroupPath
		case "heartbeat-interval-s":
			cfg.Thalamus.HeartbeatIntervalS = *heartbeatIntervalS
		}
	})

	return cfg, nil
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