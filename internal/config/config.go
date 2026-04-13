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
	Survival  SurvivalConfig // Phase 2: Sympathetic Nervous System
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

// SurvivalConfig controls Phase 2 — The Sympathetic Nervous System.
// These parameters govern the survival scheduler and memory thermodynamics.
type SurvivalConfig struct {
	// Enabled activates Phase 2 physical intervention at LevelSurvival.
	// When false, LevelSurvival falls back to LevelProtection behavior.
	Enabled bool

	// ThresholdSurvival is the D̄_M value that triggers Level 4 (Survival).
	// Must be > ThresholdProtection. Default: 12.0.
	ThresholdSurvival float64

	// SchedExtBPFObject is the path to the compiled sched_ext eBPF object.
	// If empty or sched_ext is unavailable, fallback to cpu.weight starvation.
	SchedExtBPFObject string

	// OffenderCgroupPath is the cgroup v2 path of the process under containment.
	// Receives cpu.weight=CpuWeightStarve and cpuset.cpus isolation.
	OffenderCgroupPath string

	// VitalCgroupPath is the cgroup v2 path for vital/protected processes.
	// Receives cache-warm CPU pinning and memory.swap.max=0.
	VitalCgroupPath string

	// CpuWeightStarve is the cpu.weight assigned to the offender at LevelSurvival.
	// Range: 1–10000. Value 1 = near-starvation within cgroup hierarchy.
	CpuWeightStarve int

	// SwappinessOffender is memory.swappiness for the offender cgroup (0–200).
	// High value forces the kernel to aggressively swap offender pages first.
	SwappinessOffender int

	// SwappinessVital is memory.swappiness for the vital cgroup (0–200).
	// Value 0 prevents vital process pages from being swapped at all.
	SwappinessVital int

	// FragEntropyThreshold is the normalized H_frag value above which
	// preemptive memory compaction is triggered. Range [0, 1]. Default: 0.78.
	FragEntropyThreshold float64

	// CompactionTroughCPUPct is the cpu_run_queue rate below which the system
	// is considered "in a CPU trough" — safe to trigger compaction.
	// Prevents compaction from adding latency during active stress.
	CompactionTroughCPUPct float64
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
		Survival: SurvivalConfig{
			Enabled:                false,
			ThresholdSurvival:      12.0,
			SchedExtBPFObject:      "",
			OffenderCgroupPath:     "/sys/fs/cgroup/hosa",
			VitalCgroupPath:        "/sys/fs/cgroup/hosa/vital",
			CpuWeightStarve:        1,
			SwappinessOffender:     200,
			SwappinessVital:        0,
			FragEntropyThreshold:   0.78,
			CompactionTroughCPUPct: 0.10,
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

	cfg.Survival.Enabled = t.getBool("survival.enabled", cfg.Survival.Enabled)
	cfg.Survival.ThresholdSurvival = t.getFloat("survival.threshold_survival", cfg.Survival.ThresholdSurvival)
	cfg.Survival.SchedExtBPFObject = t.getString("survival.sched_ext_bpf_object", cfg.Survival.SchedExtBPFObject)
	cfg.Survival.OffenderCgroupPath = t.getString("survival.offender_cgroup_path", cfg.Survival.OffenderCgroupPath)
	cfg.Survival.VitalCgroupPath = t.getString("survival.vital_cgroup_path", cfg.Survival.VitalCgroupPath)
	cfg.Survival.CpuWeightStarve = t.getInt("survival.cpu_weight_starve", cfg.Survival.CpuWeightStarve)
	cfg.Survival.SwappinessOffender = t.getInt("survival.swappiness_offender", cfg.Survival.SwappinessOffender)
	cfg.Survival.SwappinessVital = t.getInt("survival.swappiness_vital", cfg.Survival.SwappinessVital)
	cfg.Survival.FragEntropyThreshold = t.getFloat("survival.frag_entropy_threshold", cfg.Survival.FragEntropyThreshold)
	cfg.Survival.CompactionTroughCPUPct = t.getFloat("survival.compaction_trough_cpu_pct", cfg.Survival.CompactionTroughCPUPct)

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

	survivalEnabled   := flag.Bool("survival-enabled", defaults.Survival.Enabled, "Enable Phase 2 survival scheduler and memory thermodynamics")
	thresholdSurvival := flag.Float64("threshold-survival", defaults.Survival.ThresholdSurvival, "D_M threshold for Level 4 (Survival)")
	fragThreshold     := flag.Float64("frag-entropy-threshold", defaults.Survival.FragEntropyThreshold, "Normalized H_frag entropy threshold for preemptive compaction")

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
		case "survival-enabled":
			cfg.Survival.Enabled = *survivalEnabled
		case "threshold-survival":
			cfg.Survival.ThresholdSurvival = *thresholdSurvival
		case "frag-entropy-threshold":
			cfg.Survival.FragEntropyThreshold = *fragThreshold
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

	if cfg.Survival.Enabled {
		sv := cfg.Survival
		if sv.ThresholdSurvival <= d.ThresholdProtection {
			return fmt.Errorf("config: survival.threshold_survival (%.2f) must be > threshold_protection (%.2f)",
				sv.ThresholdSurvival, d.ThresholdProtection)
		}
		if sv.FragEntropyThreshold <= 0 || sv.FragEntropyThreshold > 1 {
			return fmt.Errorf("config: survival.frag_entropy_threshold must be in (0, 1], got %.3f", sv.FragEntropyThreshold)
		}
		if sv.CpuWeightStarve < 1 || sv.CpuWeightStarve > 10000 {
			return fmt.Errorf("config: survival.cpu_weight_starve must be in [1, 10000], got %d", sv.CpuWeightStarve)
		}
	}

	return nil
}

// Summary returns a one-line human-readable summary for boot logging.
func (cfg Config) Summary() string {
	d := cfg.Detection
	s := cfg.Sampling
	phase2 := "disabled"
	if cfg.Survival.Enabled {
		phase2 = fmt.Sprintf("enabled(th=%.1f h_frag=%.2f)", cfg.Survival.ThresholdSurvival, cfg.Survival.FragEntropyThreshold)
	}
	return fmt.Sprintf(
		"thresholds=[%.1f/%.1f/%.1f/%.1f] alpha=%.2f min_samples=%d normal=%dms vigilance=%dms heartbeat=%ds phase2=%s",
		d.ThresholdVigilance, d.ThresholdContainment, d.ThresholdProtection, cfg.Survival.ThresholdSurvival,
		d.AlphaEWMA, d.MinSamples,
		s.NormalIntervalMs, s.VigilanceIntervalMs,
		cfg.Thalamus.HeartbeatIntervalS,
		phase2,
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