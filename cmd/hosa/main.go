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
	"github.com/bricio-sr/hosa/internal/motor"
	"github.com/bricio-sr/hosa/internal/sensor"
	"github.com/bricio-sr/hosa/internal/state"
	"github.com/bricio-sr/hosa/internal/syscgroup"
)

const (
	ringBufferCapacity = 300
	numVars            = 4 // must match sensor.NumVars
	normalInterval     = 1 * time.Second
	vigilanceInterval  = 100 * time.Millisecond
	logEveryN          = 10
)

func main() {
	// Strip the default log prefix — thalamus controls all formatting.
	log.SetFlags(0)

	thalamus := brain.NewThalamicFilter(brain.DefaultThalamicConfig(), nil)

	// --- Layer 0: Hardware Proprioception ---
	topo, err := sensor.DiscoverTopology()
	if err != nil {
		thalamus.Boot(fmt.Sprintf("topology=unknown err=%v", err))
	} else {
		thalamus.Boot(fmt.Sprintf("topology=%s", topo))
	}

	// --- Layer 1: Short-Term Memory ---
	buf := state.NewRingBuffer(ringBufferCapacity, numVars)

	// --- Layer 2: eBPF Sensor ---
	col := &sensor.Collector{}
	if err := col.Start(); err != nil {
		log.Fatalf("HOSA [FATAL] sensor_init_failed err=%v", err)
	}
	defer col.Close()

	// --- Layer 3: Predictive Cortex ---
	cortex := brain.NewPredictiveCortex(buf, brain.DefaultConfig())

	// --- Layer 4: Motor ---
	cgPath, err := syscgroup.EnsureHosaCgroup()
	if err != nil {
		log.Fatalf("HOSA [FATAL] cgroup_init_failed err=%v", err)
	}
	mot := motor.NewCgroupMotor(cgPath)

	memTotal, err := readMemTotal()
	if err != nil {
		log.Fatalf("HOSA [FATAL] mem_read_failed err=%v", err)
	}

	thalamus.Boot(fmt.Sprintf("mem_total=%.1fGB status=calibrating", float64(memTotal)/(1<<30)))

	// --- Graceful shutdown ---
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	interval := normalInterval
	var tickCount int

	// --- Main Loop: The Reflex Arc ---
	// Sense → Memorize → Analyze → Filter → React
	for {
		select {
		case <-ctx.Done():
			log.Print("HOSA [SHUTDOWN]    restoring homeostasis")
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

			thalamus.Observe(level, stress, dmDot)

			interval = react(mot, thalamus, stress, dmDot, level, memTotal, tickCount)
		}
	}
}

func react(mot *motor.CgroupMotor, thalamus *brain.ThalamicFilter, stress, dmDot float64, level brain.AlertLevel, memTotal uint64, tick int) time.Duration {
	containLevel := motor.ContainmentLevel(level)

	changed, err := mot.Apply(containLevel, memTotal)
	if err != nil {
		log.Printf("HOSA [ERROR] motor_apply err=%v", err)
	}

	// Notify thalamus only when the motor actually acted.
	if changed && level >= brain.LevelContainment {
		action := motor.ActionSummary(containLevel, memTotal)
		thalamus.NotifyContainment(level, stress, action)
	}

	switch level {
	case brain.LevelHomeostasis:
		return normalInterval

	case brain.LevelVigilance:
		// Log only on escalation or periodically during descent — reduce noise.
		if dmDot >= 0 || tick%logEveryN == 0 {
			log.Printf("HOSA [VIGILANCE]   level=1 dm=%.4f dm_dot=%+.4f", stress, dmDot)
		}
		return vigilanceInterval

	case brain.LevelContainment:
		log.Printf("HOSA [CONTAINMENT] level=2 dm=%.4f dm_dot=%+.4f", stress, dmDot)
		return vigilanceInterval

	case brain.LevelProtection:
		log.Printf("HOSA [PROTECTION]  level=3 dm=%.4f dm_dot=%+.4f", stress, dmDot)
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