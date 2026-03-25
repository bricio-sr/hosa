package config

import (
	"os"
	"testing"
)

// TestParseTOML_Basic verifica o parsing de um TOML simples com seções e tipos.
func TestParseTOML_Basic(t *testing.T) {
	toml := `
# Comment line
[detection]
threshold_vigilance = 3.5
min_samples = 30
alpha_ewma = 0.2

[detection.alpha_per_probe]
cpu_run_queue = 0.30

[motor]
cgroup_path = "/sys/fs/cgroup/hosa"
`
	tbl, err := parseTOMLBytes([]byte(toml))
	if err != nil {
		t.Fatalf("parseTOML failed: %v", err)
	}

	cases := []struct {
		key      string
		expected string
	}{
		{"detection.threshold_vigilance", "3.5"},
		{"detection.min_samples", "30"},
		{"detection.alpha_ewma", "0.2"},
		{"detection.alpha_per_probe.cpu_run_queue", "0.30"},
		{"motor.cgroup_path", "/sys/fs/cgroup/hosa"},
	}

	for _, c := range cases {
		got, ok := tbl[c.key]
		if !ok {
			t.Errorf("key %q not found", c.key)
			continue
		}
		if got != c.expected {
			t.Errorf("key %q: expected %q, got %q", c.key, c.expected, got)
		}
	}
}

// TestParseTOML_InlineComment verifica que comentários inline são removidos.
func TestParseTOML_InlineComment(t *testing.T) {
	toml := `
[detection]
alpha_ewma = 0.2  # conservative for noisy environments
min_samples = 30  # 30 seconds at 1s interval
`
	tbl, err := parseTOMLBytes([]byte(toml))
	if err != nil {
		t.Fatalf("parseTOML failed: %v", err)
	}

	if got := tbl["detection.alpha_ewma"]; got != "0.2" {
		t.Errorf("inline comment not stripped: got %q", got)
	}
	if got := tbl["detection.min_samples"]; got != "30" {
		t.Errorf("inline comment not stripped: got %q", got)
	}
}

// TestDefault_IsValid verifica que a configuração padrão passa na validação.
func TestDefault_IsValid(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Default config failed validation: %v", err)
	}
}

// TestLoad_NoFile verifica que Load retorna defaults quando o arquivo não existe.
func TestLoad_NoFile(t *testing.T) {
	cfg, err := Load("/tmp/hosa_definitely_not_there.toml")
	if err != nil {
		t.Fatalf("Load with missing file should not error: %v", err)
	}
	def := Default()
	if cfg.Detection.ThresholdVigilance != def.Detection.ThresholdVigilance {
		t.Error("missing file should return defaults")
	}
}

// TestLoad_OverridesDefaults verifica que valores do arquivo sobrescrevem os defaults.
func TestLoad_OverridesDefaults(t *testing.T) {
	f, err := os.CreateTemp("", "hosa_config_test_*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	f.WriteString(`
[detection]
threshold_vigilance = 4.0
alpha_ewma = 0.4
min_samples = 60

[sampling]
normal_interval_ms = 2000
`)
	f.Close()

	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Detection.ThresholdVigilance != 4.0 {
		t.Errorf("threshold_vigilance: expected 4.0, got %.2f", cfg.Detection.ThresholdVigilance)
	}
	if cfg.Detection.AlphaEWMA != 0.4 {
		t.Errorf("alpha_ewma: expected 0.4, got %.2f", cfg.Detection.AlphaEWMA)
	}
	if cfg.Detection.MinSamples != 60 {
		t.Errorf("min_samples: expected 60, got %d", cfg.Detection.MinSamples)
	}
	if cfg.Sampling.NormalIntervalMs != 2000 {
		t.Errorf("normal_interval_ms: expected 2000, got %d", cfg.Sampling.NormalIntervalMs)
	}

	// Unspecified values should remain as defaults
	def := Default()
	if cfg.Detection.ThresholdContainment != def.Detection.ThresholdContainment {
		t.Error("unspecified threshold_containment should remain default")
	}
}

// TestValidate_ThresholdOrdering verifica que limiares fora de ordem são rejeitados.
func TestValidate_ThresholdOrdering(t *testing.T) {
	cfg := Default()
	cfg.Detection.ThresholdContainment = cfg.Detection.ThresholdVigilance - 0.1

	if err := cfg.Validate(); err == nil {
		t.Error("expected error when containment <= vigilance, got nil")
	}
}

// TestValidate_AlphaRange verifica que α fora do intervalo válido é rejeitado.
func TestValidate_AlphaRange(t *testing.T) {
	cfg := Default()
	cfg.Detection.AlphaEWMA = 1.5

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for alpha > 1, got nil")
	}

	cfg.Detection.AlphaEWMA = 0.0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for alpha = 0, got nil")
	}
}

// TestSummary_Format verifica que Summary() retorna uma string não-vazia.
func TestSummary_Format(t *testing.T) {
	cfg := Default()
	s := cfg.Summary()
	if s == "" {
		t.Error("Summary() returned empty string")
	}
	t.Logf("Config summary: %s", s)
}

// TestLoad_DefaultTOMLFile verifica que o hosa.toml padrão é parseable e válido.
func TestLoad_DefaultTOMLFile(t *testing.T) {
	cfg, err := Load("hosa.toml")
	if err != nil {
		t.Fatalf("Load of default hosa.toml failed: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Default hosa.toml failed validation: %v", err)
	}
	t.Logf("Loaded config: %s", cfg.Summary())
}