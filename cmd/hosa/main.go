package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bricio-sr/hosa/internal/brain"
	"github.com/bricio-sr/hosa/internal/config"
	"github.com/bricio-sr/hosa/internal/motor"
	"github.com/bricio-sr/hosa/internal/sensor"
	"github.com/bricio-sr/hosa/internal/state"
	"github.com/bricio-sr/hosa/internal/syscgroup"
	"github.com/bricio-sr/hosa/internal/telemetry"
)

const (
	ringBufferCapacity = 300
	numVars            = 4  // must match sensor.NumVars
	logEveryN          = 10
)

// phase2State encapsula os componentes da Fase 2 para passar ao react().
type phase2State struct {
	survivalMotor *motor.SurvivalMotor
	fragMonitor   *sensor.FragmentationMonitor
	lastFragState sensor.FragState
	enabled       bool
}

func main() {
	log.SetFlags(0)

	// Single call: defaults → TOML → CLI flags. One flag.Parse(), no repetition.
	cfg, err := config.LoadWithFlags()
	if err != nil {
		log.Fatalf("HOSA [FATAL] config_load_failed err=%v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("HOSA [FATAL] config_invalid err=%v", err)
	}

	// --- Build runtime intervals from config ---
	normalInterval    := cfg.Sampling.NormalInterval()
	vigilanceInterval := cfg.Sampling.VigilanceInterval()

	// --- Thalamic Filter (log authority) ---
	thalamus := brain.NewThalamicFilter(
		brain.ThalamicConfig{HeartbeatInterval: cfg.Thalamus.HeartbeatInterval()},
		nil,
	)

	// --- Layer 0: Hardware Proprioception ---
	topo, err := sensor.DiscoverTopology()
	if err != nil {
		thalamus.Boot(fmt.Sprintf("topology=unknown err=%v", err))
	} else {
		thalamus.Boot(fmt.Sprintf("topology=%s", topo))
	}
	thalamus.Boot(fmt.Sprintf("config=%s", cfg.Summary()))

	// --- Layer 1: Short-Term Memory ---
	buf := state.NewRingBuffer(ringBufferCapacity, numVars)

	// --- Layer 2: eBPF Sensor ---
	col := &sensor.Collector{}
	if err := col.Start(); err != nil {
		log.Fatalf("HOSA [FATAL] sensor_init_failed err=%v", err)
	}
	defer col.Close()

	// --- Layer 3: Predictive Cortex ---
	cortex := brain.NewPredictiveCortex(buf, brain.PredictorConfig{
		MinSamples:                  cfg.Detection.MinSamples,
		Alpha:                       cfg.Detection.AlphaEWMA,
		ThresholdVigilance:          cfg.Detection.ThresholdVigilance,
		ThresholdContainment:        cfg.Detection.ThresholdContainment,
		ThresholdProtection:         cfg.Detection.ThresholdProtection,
		ThresholdDerivativeEscalate: cfg.Detection.ThresholdDerivativeEscalate,
		ThresholdDerivativeRelax:    cfg.Detection.ThresholdDerivativeRelax,
		HysteresisDown:              cfg.Detection.HysteresisDown,
	})

	// --- Layer 4: Motor ---
	cgPath, err := syscgroup.EnsureHosaCgroupAt(cfg.Motor.CgroupPath)
	if err != nil {
		log.Fatalf("HOSA [FATAL] cgroup_init_failed err=%v", err)
	}
	mot := motor.NewCgroupMotor(cgPath, motor.MotorConfig{
		ContainmentFraction: cfg.Motor.ContainmentFraction,
		ProtectionHighFrac:  cfg.Motor.ProtectionHighFrac,
		ProtectionMaxFrac:   cfg.Motor.ProtectionMaxFrac,
	})

	memTotal, err := readMemTotal()
	if err != nil {
		log.Fatalf("HOSA [FATAL] mem_read_failed err=%v", err)
	}
	thalamus.Boot(fmt.Sprintf("mem_total=%.1fGB status=calibrating", float64(memTotal)/(1<<30)))

	// --- Layer 5: Phase 2 — Sympathetic Nervous System ---
	p2 := &phase2State{enabled: cfg.Survival.Enabled}
	if cfg.Survival.Enabled {
		survMotor, err := motor.NewSurvivalMotor(motor.SurvivalConfig{
			SchedExtBPFObject:  cfg.Survival.SchedExtBPFObject,
			OffenderCgroupPath: cfg.Survival.OffenderCgroupPath,
			VitalCgroupPath:    cfg.Survival.VitalCgroupPath,
			CpuWeightStarve:    uint32(cfg.Survival.CpuWeightStarve),
			SwappinessOffender: cfg.Survival.SwappinessOffender,
			SwappinessVital:    cfg.Survival.SwappinessVital,
		}, topo)
		if err != nil {
			// Não fatal — Fase 2 é opcional
			thalamus.Boot(fmt.Sprintf("phase2=init_failed err=%v", err))
		} else {
			p2.survivalMotor = survMotor
			p2.fragMonitor = sensor.NewFragmentationMonitor(sensor.FragConfig{
				Threshold:          cfg.Survival.FragEntropyThreshold,
				CPUTroughThreshold: cfg.Survival.CompactionTroughCPUPct,
			})
			schedExtStatus := "unavailable"
			if survMotor.SchedExtAvailable() {
				schedExtStatus = "available"
			}
			thalamus.Boot(fmt.Sprintf("phase2=ready sched_ext=%s frag_threshold=%.2f",
				schedExtStatus, cfg.Survival.FragEntropyThreshold))
		}
	}

	// --- Layer 6: Phase 3 — Ecosystem Symbiosis ---
	telState := &telemetry.AtomicState{}
	var telSrv *telemetry.Server
	if cfg.Telemetry.MetricsAddr != "" {
		telSrv = telemetry.NewServer(cfg.Telemetry.MetricsAddr, telState)
		if err := telSrv.Start(); err != nil {
			thalamus.Boot(fmt.Sprintf("phase3=metrics_failed err=%v", err))
		} else {
			thalamus.Boot(fmt.Sprintf("phase3=ready metrics=%s", cfg.Telemetry.MetricsAddr))
		}
	}
	var webhook *telemetry.WebhookClient
	if cfg.Telemetry.WebhookURL != "" {
		webhook = telemetry.NewWebhookClient(cfg.Telemetry.WebhookURL, cfg.Telemetry.WebhookDMDotThreshold)
		thalamus.Boot(fmt.Sprintf("phase3=webhook_enabled url=%s threshold=%.2f",
			cfg.Telemetry.WebhookURL, cfg.Telemetry.WebhookDMDotThreshold))
	}

	// --- Graceful shutdown ---
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	interval := normalInterval
	var tickCount int

	// --- Main Loop: The Reflex Arc + SNS ---
	// Sense → Memorize → Analyze → Filter → React (Phase 1 + Phase 2)
	for {
		select {
		case <-ctx.Done():
			log.Print("HOSA [SHUTDOWN] restoring homeostasis")
			if telSrv != nil {
				shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				_ = telSrv.Stop(shutCtx)
				cancel()
			}
			if p2.survivalMotor != nil {
				p2.survivalMotor.Release()
			}
			mot.Apply(motor.LevelHomeostasis, memTotal)
			return

		case <-time.After(interval):
			tickCount++

			reading := col.ReadMetrics()
			if err := buf.Insert(reading); err != nil {
				log.Printf("HOSA [ERROR] buffer_insert err=%v", err)
				continue
			}

			stress, dmDot, level, err := cortex.Analyze()
			if err != nil {
				log.Printf("HOSA [ERROR] cortex_analyze err=%v", err)
				continue
			}

			// Fase 2: amostrar H_frag a cada ciclo (quando habilitado)
			if p2.fragMonitor != nil {
				fragState, triggered, fragErr := p2.fragMonitor.Sample(reading[0])
				if fragErr == nil {
					p2.lastFragState = fragState
					if triggered {
						log.Printf("HOSA [COMPACTION] h_frag=%.4f compactions_total=%d",
							fragState.HFragNorm, p2.fragMonitor.CompactionCount())
					}
				}
			}

			// Fase 3: atualiza estado compartilhado para /metrics e /healthz
			svCopy := make([]float64, len(reading))
			copy(svCopy, reading[:])
			snap := telemetry.Snapshot{
				DM:        stress,
				DMDot:     dmDot,
				Level:     int(level),
				HFragNorm: p2.lastFragState.HFragNorm,
				StateVec:  svCopy,
				UpdatedAt: time.Now(),
			}
			telState.Set(snap)
			if webhook != nil {
				webhook.Notify(snap)
			}

			thalamus.Observe(level, stress, dmDot)
			interval = react(mot, thalamus, stress, dmDot, level, memTotal,
				tickCount, normalInterval, vigilanceInterval, p2)
		}
	}
}

func react(mot *motor.CgroupMotor, thalamus *brain.ThalamicFilter,
	stress, dmDot float64, level brain.AlertLevel, memTotal uint64,
	tick int, normalInterval, vigilanceInterval time.Duration,
	p2 *phase2State) time.Duration {

	changed, err := mot.Apply(motor.ContainmentLevel(level), memTotal)
	if err != nil {
		log.Printf("HOSA [ERROR] motor_apply err=%v", err)
	}
	if changed && level >= brain.LevelContainment {
		thalamus.NotifyContainment(level, stress, motor.ActionSummary(motor.ContainmentLevel(level), memTotal))
	}

	// Fase 2: engaja ou libera o SurvivalMotor conforme o nível
	if p2.enabled && p2.survivalMotor != nil {
		if level == brain.LevelSurvival && !p2.survivalMotor.Active() {
			if err := p2.survivalMotor.Engage(memTotal); err != nil {
				log.Printf("HOSA [ERROR] survival_engage err=%v", err)
			} else {
				thalamus.NotifySurvival(stress, dmDot, p2.lastFragState.HFragNorm,
					p2.survivalMotor.ActionSummary())
			}
		} else if level < brain.LevelSurvival && p2.survivalMotor.Active() {
			if err := p2.survivalMotor.Release(); err != nil {
				log.Printf("HOSA [ERROR] survival_release err=%v", err)
			}
		}
	}

	switch level {
	case brain.LevelHomeostasis:
		return normalInterval
	case brain.LevelVigilance:
		if dmDot >= 0 || tick%logEveryN == 0 {
			log.Printf("HOSA [VIGILANCE]  level=1 dm=%.4f dm_dot=%+.4f", stress, dmDot)
		}
		return vigilanceInterval
	case brain.LevelContainment:
		log.Printf("HOSA [CONTAINMENT] level=2 dm=%.4f dm_dot=%+.4f", stress, dmDot)
		return vigilanceInterval
	case brain.LevelProtection:
		log.Printf("HOSA [PROTECTION]  level=3 dm=%.4f dm_dot=%+.4f", stress, dmDot)
		return vigilanceInterval
	case brain.LevelSurvival:
		log.Printf("HOSA [SURVIVAL]    level=4 dm=%.4f dm_dot=%+.4f h_frag=%.4f",
			stress, dmDot, p2.lastFragState.HFragNorm)
		return vigilanceInterval
	default:
		return normalInterval
	}
}

func readMemTotal() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	for _, line := range splitLines(string(data)) {
		if len(line) > 9 && line[:9] == "MemTotal:" {
			fields := splitFields(line[9:])
			if len(fields) == 0 {
				continue
			}
			return parseUint(fields[0]) * 1024, nil
		}
	}
	return 0, os.ErrNotExist
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitFields(s string) []string {
	var fields []string
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			if start == -1 {
				start = i
			}
		} else {
			if start != -1 {
				fields = append(fields, s[start:i])
				start = -1
			}
		}
	}
	if start != -1 {
		fields = append(fields, s[start:])
	}
	return fields
}

func parseUint(s string) uint64 {
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + uint64(c-'0')
	}
	return n
}