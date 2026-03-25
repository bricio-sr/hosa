package motor

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// newTestMotor cria um CgroupMotor apontando para um diretório temporário
// que simula a estrutura de um cgroup v2.
func newTestMotor(t *testing.T) (*CgroupMotor, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "hosa_motor_test_*")
	if err != nil {
		t.Fatalf("falha ao criar cgroup simulado: %v", err)
	}
	m := NewCgroupMotor(dir)
	// Força lastLevel para um valor sentinela para que o primeiro Apply sempre execute.
	m.lastLevel = -1
	return m, func() { os.RemoveAll(dir) }
}

func readFile(t *testing.T, cgPath, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(cgPath, name))
	if err != nil {
		t.Fatalf("falha ao ler %q: %v", name, err)
	}
	return string(data)
}

const totalMem = uint64(4 * 1024 * 1024 * 1024) // 4 GB para os testes

func TestApply_Homeostasis_RemovesLimits(t *testing.T) {
	m, cleanup := newTestMotor(t)
	defer cleanup()

	os.WriteFile(filepath.Join(m.cgPath, "memory.high"), []byte("1000000"), 0644)
	os.WriteFile(filepath.Join(m.cgPath, "memory.max"), []byte("2000000"), 0644)

	if _, err := m.Apply(LevelHomeostasis, totalMem); err != nil {
		t.Fatalf("Apply(Homeostasis) falhou: %v", err)
	}

	if got := readFile(t, m.cgPath, "memory.high"); got != "max" {
		t.Errorf("memory.high: esperado 'max', obtido %q", got)
	}
	if got := readFile(t, m.cgPath, "memory.max"); got != "max" {
		t.Errorf("memory.max: esperado 'max', obtido %q", got)
	}
}

func TestApply_Vigilance_RemovesLimits(t *testing.T) {
	m, cleanup := newTestMotor(t)
	defer cleanup()

	if _, err := m.Apply(LevelVigilance, totalMem); err != nil {
		t.Fatalf("Apply(Vigilance) falhou: %v", err)
	}

	if got := readFile(t, m.cgPath, "memory.high"); got != "max" {
		t.Errorf("memory.high: esperado 'max', obtido %q", got)
	}
}

func TestApply_Containment_SetsMemoryHigh(t *testing.T) {
	m, cleanup := newTestMotor(t)
	defer cleanup()

	if _, err := m.Apply(LevelContainment, totalMem); err != nil {
		t.Fatalf("Apply(Containment) falhou: %v", err)
	}

	expected := uint64(float64(totalMem) * fractionContainment)
	got := readFile(t, m.cgPath, "memory.high")

	if got != strconv.FormatUint(expected, 10) {
		t.Errorf("memory.high: esperado %d, obtido %q", expected, got)
	}
}

func TestApply_Protection_SetsBothLimits(t *testing.T) {
	m, cleanup := newTestMotor(t)
	defer cleanup()

	if _, err := m.Apply(LevelProtection, totalMem); err != nil {
		t.Fatalf("Apply(Protection) falhou: %v", err)
	}

	total := float64(totalMem)
	expectedHigh := uint64(total * fractionProtectionHigh)
	expectedMax := uint64(total * fractionProtectionMax)

	gotHigh := readFile(t, m.cgPath, "memory.high")
	gotMax := readFile(t, m.cgPath, "memory.max")

	if gotHigh != strconv.FormatUint(expectedHigh, 10) {
		t.Errorf("memory.high: esperado %d, obtido %q", expectedHigh, gotHigh)
	}
	if gotMax != strconv.FormatUint(expectedMax, 10) {
		t.Errorf("memory.max: esperado %d, obtido %q", expectedMax, gotMax)
	}
}

func TestApply_UnknownLevel_ReturnsError(t *testing.T) {
	m, cleanup := newTestMotor(t)
	defer cleanup()

	if _, err := m.Apply(ContainmentLevel(99), totalMem); err == nil {
		t.Fatal("esperado erro para nível desconhecido, mas nenhum erro retornado")
	}
}

func TestApply_ContainmentIsStricterThanVigilance(t *testing.T) {
	m, cleanup := newTestMotor(t)
	defer cleanup()

	m.Apply(LevelContainment, totalMem)
	highStr := readFile(t, m.cgPath, "memory.high")
	high, _ := strconv.ParseUint(highStr, 10, 64)

	if high >= totalMem {
		t.Errorf("Contenção deveria limitar memória, mas memory.high=%d >= totalMem=%d", high, totalMem)
	}
}

func TestApply_ProtectionIsStricterThanContainment(t *testing.T) {
	m, cleanup := newTestMotor(t)
	defer cleanup()

	m.Apply(LevelContainment, totalMem)
	containHighStr := readFile(t, m.cgPath, "memory.high")
	containHigh, _ := strconv.ParseUint(containHighStr, 10, 64)

	m.lastLevel = -1 // reset para forçar re-aplicação
	m.Apply(LevelProtection, totalMem)
	protectHighStr := readFile(t, m.cgPath, "memory.high")
	protectHigh, _ := strconv.ParseUint(protectHighStr, 10, 64)

	if protectHigh >= containHigh {
		t.Errorf("Proteção deveria ser mais restritiva que Contenção: protect=%d >= contain=%d",
			protectHigh, containHigh)
	}
}