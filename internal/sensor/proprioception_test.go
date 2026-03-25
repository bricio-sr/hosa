package sensor

import (
	"strings"
	"testing"
)

// TestDiscoverTopology_Runs verifica que DiscoverTopology executa sem erro
// no ambiente atual e retorna valores plausíveis.
func TestDiscoverTopology_Runs(t *testing.T) {
	topo, err := DiscoverTopology()
	if err != nil {
		t.Fatalf("DiscoverTopology falhou: %v", err)
	}

	if topo.MemoryTotalBytes == 0 {
		t.Error("MemoryTotalBytes = 0")
	}
	if topo.LogicalCores <= 0 {
		t.Errorf("LogicalCores inválido: %d", topo.LogicalCores)
	}
	if topo.PhysicalCores <= 0 {
		t.Errorf("PhysicalCores inválido: %d", topo.PhysicalCores)
	}
	if topo.PhysicalCores > topo.LogicalCores {
		t.Errorf("PhysicalCores(%d) > LogicalCores(%d) — impossível",
			topo.PhysicalCores, topo.LogicalCores)
	}
	if topo.NUMANodes <= 0 {
		t.Errorf("NUMANodes inválido: %d", topo.NUMANodes)
	}

	t.Logf("Topologia detectada: %s", topo.String())
}

// TestParseCPURange verifica o parsing de ranges de CPU do kernel.
func TestParseCPURange(t *testing.T) {
	cases := []struct {
		input    string
		expected int
	}{
		{"0", 1},
		{"0-3", 4},
		{"0-7", 8},
		{"0,2,4", 3},
		{"0-3,5,7-9", 8}, // 0,1,2,3 + 5 + 7,8,9 = 8
		{"0-1,4-5", 4},
		{"", 0},
	}

	for _, c := range cases {
		got := parseCPURange(c.input)
		if got != c.expected {
			t.Errorf("parseCPURange(%q) = %d, esperado %d", c.input, got, c.expected)
		}
	}
}

// TestParseMemSize verifica a conversão de tamanhos de memória.
func TestParseMemSize(t *testing.T) {
	cases := []struct {
		input    string
		expected uint64
	}{
		{"8192K", 8192 * 1024},
		{"16M", 16 * 1024 * 1024},
		{"1G", 1024 * 1024 * 1024},
		{"512K", 512 * 1024},
		{"0K", 0},
		{"", 0},
	}

	for _, c := range cases {
		got := parseMemSize(c.input)
		if got != c.expected {
			t.Errorf("parseMemSize(%q) = %d, esperado %d", c.input, got, c.expected)
		}
	}
}

// TestTopologyString_Format verifica que String() retorna um formato legível.
func TestTopologyString_Format(t *testing.T) {
	topo := &Topology{
		PhysicalCores:    4,
		LogicalCores:     8,
		NUMANodes:        1,
		MemoryTotalBytes: 8 * 1024 * 1024 * 1024,
		CacheSizeL3Bytes: 8 * 1024 * 1024,
		IsVM:             true,
		HypervisorVendor: "KVM",
	}

	s := topo.String()

	// Verifica que os campos relevantes aparecem no output
	checks := []string{"4", "8", "NUMA=1", "8.0GB", "8MB", "KVM"}
	for _, check := range checks {
		if !strings.Contains(s, check) {
			t.Errorf("String() não contém %q: %q", check, s)
		}
	}
}

// TestCountCPUDirs_ReturnsPositive verifica que countCPUDirs retorna pelo menos 1.
func TestCountCPUDirs_ReturnsPositive(t *testing.T) {
	count := countCPUDirs()
	if count <= 0 {
		t.Errorf("countCPUDirs() = %d, esperado >= 1", count)
	}
	t.Logf("CPUs lógicas via diretórios: %d", count)
}