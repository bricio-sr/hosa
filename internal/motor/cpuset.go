// Package motor — controles de CPU e memória para a Fase 2.
//
// cpuset.go implementa os controles de afinidade de CPU e pressão de memória
// via cgroup v2, usados pelo SurvivalMotor para Targeted Starvation e
// Predictive Cache Affinity.
//
// Controles implementados:
//   - cpu.weight    : peso de CPU no CFS (1 = near-starvation, 10000 = máximo)
//   - cpuset.cpus   : conjunto de CPUs permitidas para o cgroup
//   - memory.swappiness : agressividade de swap por cgroup (0–200)
//   - memory.swap.max   : limite máximo de uso de swap ("0" = sem swap)
//
// Referência: kernel docs/admin-guide/cgroup-v2.rst
package motor

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/bricio-sr/hosa/internal/syscgroup"
)

// CPUSet é um slice ordenado de IDs de CPUs lógicas.
// Representa o conjunto de CPUs permitidas para um cgroup.
type CPUSet []int

// String converte um CPUSet para o formato aceito pelo kernel: "0-3,5,7".
// CPUs consecutivas são representadas como ranges; isoladas, individualmente.
func (cs CPUSet) String() string {
	if len(cs) == 0 {
		return ""
	}

	sorted := make([]int, len(cs))
	copy(sorted, cs)
	sort.Ints(sorted)

	var sb strings.Builder
	start := sorted[0]
	end := sorted[0]

	flush := func() {
		if sb.Len() > 0 {
			sb.WriteByte(',')
		}
		if start == end {
			sb.WriteString(strconv.Itoa(start))
		} else {
			sb.WriteString(strconv.Itoa(start))
			sb.WriteByte('-')
			sb.WriteString(strconv.Itoa(end))
		}
	}

	for _, cpu := range sorted[1:] {
		if cpu == end+1 {
			end = cpu
		} else {
			flush()
			start = cpu
			end = cpu
		}
	}
	flush()

	return sb.String()
}

// SetCgroupCPUSet escreve o arquivo cpuset.cpus de um cgroup, restringindo
// as CPUs que os processos nesse cgroup podem usar.
//
// cpus vazio ("") remove a restrição — o cgroup herda o cpuset do pai.
// Requer que o controlador "cpuset" esteja habilitado na hierarquia do cgroup.
func SetCgroupCPUSet(cgPath string, cpus CPUSet) error {
	value := cpus.String()
	if value == "" {
		value = "0-" + strconv.Itoa(maxCPUID())
	}
	if err := syscgroup.WriteControl(cgPath, "cpuset.cpus", value); err != nil {
		return fmt.Errorf("SetCgroupCPUSet(%q, %q): %w", cgPath, value, err)
	}
	return nil
}

// SetCgroupCPUWeight escreve o arquivo cpu.weight de um cgroup.
// Range: 1–10000. Valor 1 = near-starvation (processo recebe ~0.01% de CPU
// quando outros processos competem). Valor 10000 = máxima prioridade relativa.
//
// Nota: cpu.weight=1 não é starvation absoluto — o CFS garante progresso mínimo.
// Para starvation real (0 ciclos), é necessário sched_ext (Fase 2, Tier 1).
func SetCgroupCPUWeight(cgPath string, weight uint32) error {
	if weight < 1 {
		weight = 1
	}
	if weight > 10000 {
		weight = 10000
	}
	if err := syscgroup.WriteControl(cgPath, "cpu.weight", strconv.FormatUint(uint64(weight), 10)); err != nil {
		return fmt.Errorf("SetCgroupCPUWeight(%q, %d): %w", cgPath, weight, err)
	}
	return nil
}

// SetCgroupSwappiness escreve o arquivo memory.swappiness de um cgroup.
// Range: 0–200.
//   - 0   = nunca fazer swap de páginas deste cgroup (processos vitais)
//   - 200 = swap agressivo, priorizado antes de outros cgroups (processo ofensor)
//
// Requer kernel ≥ 5.10 com suporte a memory.swappiness por cgroup v2.
func SetCgroupSwappiness(cgPath string, value int) error {
	if value < 0 {
		value = 0
	}
	if value > 200 {
		value = 200
	}
	if err := syscgroup.WriteControl(cgPath, "memory.swappiness", strconv.Itoa(value)); err != nil {
		return fmt.Errorf("SetCgroupSwappiness(%q, %d): %w", cgPath, value, err)
	}
	return nil
}

// SetCgroupSwapMax define o limite máximo de swap para um cgroup.
//   - bytes=0  → escreve "0" (sem swap permitido — proteção de processos vitais)
//   - bytes=-1 → escreve "max" (sem limite — comportamento padrão)
func SetCgroupSwapMax(cgPath string, bytes int64) error {
	value := "max"
	if bytes == 0 {
		value = "0"
	} else if bytes > 0 {
		value = strconv.FormatInt(bytes, 10)
	}
	if err := syscgroup.WriteControl(cgPath, "memory.swap.max", value); err != nil {
		return fmt.Errorf("SetCgroupSwapMax(%q, %s): %w", cgPath, value, err)
	}
	return nil
}

// ResetCgroupCPUWeight restaura o cpu.weight ao valor padrão (100).
func ResetCgroupCPUWeight(cgPath string) error {
	return SetCgroupCPUWeight(cgPath, 100)
}

// maxCPUID retorna o maior ID de CPU lógica do sistema via /sys/devices/system/cpu/present.
// Usado para construir o cpuset "0-N" que representa "todas as CPUs".
func maxCPUID() int {
	data, err := os.ReadFile("/sys/devices/system/cpu/present")
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(data))
	// Formato: "0-7" ou "0,1,2,3"
	if idx := strings.LastIndex(s, "-"); idx >= 0 {
		n, err := strconv.Atoi(s[idx+1:])
		if err == nil {
			return n
		}
	}
	if idx := strings.LastIndex(s, ","); idx >= 0 {
		n, err := strconv.Atoi(s[idx+1:])
		if err == nil {
			return n
		}
	}
	return 0
}
