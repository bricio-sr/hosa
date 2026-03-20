package sensor

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCollector_StartMissingSensorsO verifica que Start() falha com mensagem
// clara quando sensors.o não existe OU quando não há permissão de root.
// Em ambientes com o sensors.o presente, o erro esperado é de permissão (não-root).
func TestCollector_StartMissingSensorsO(t *testing.T) {
	c := &Collector{}
	err := c.Start()
	if err == nil {
		t.Fatal("esperado erro quando rodando sem root ou sem sensors.o, mas Start() retornou nil")
	}

	// Aceita qualquer um dos dois cenários:
	// 1. sensors.o não encontrado → mensagem orienta a compilar
	// 2. sensors.o existe mas não há root → erro de syscall/permissão
	errMsg := err.Error()
	validHints := []string{
		"sensors.o", "build-bpf", "não encontrado", // cenário 1
		"permission denied", "operation not permitted", "syscall bpf", // cenário 2
	}
	for _, hint := range validHints {
		if contains(errMsg, hint) {
			return // erro esperado e reconhecível
		}
	}
	t.Errorf("mensagem de erro não é reconhecível: %q", errMsg)
}

// TestFileExists verifica a função auxiliar de checagem de existência de arquivo.
func TestFileExists(t *testing.T) {
	// Arquivo que sabidamente existe
	f, err := os.CreateTemp("", "hosa_sensor_test_*")
	if err != nil {
		t.Fatalf("falha ao criar arquivo temporário: %v", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	if !fileExists(f.Name()) {
		t.Errorf("fileExists(%q) = false, esperado true", f.Name())
	}

	// Arquivo que não existe
	ghost := filepath.Join(os.TempDir(), "hosa_sensor_definitively_not_there.o")
	if fileExists(ghost) {
		t.Errorf("fileExists(%q) = true, esperado false", ghost)
	}
}

// TestCollector_CloseZeroValue verifica que Close() em um Collector não inicializado
// não causa panic — importante para o defer no main.go.
func TestCollector_CloseZeroValue(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Close() em Collector zerado causou panic: %v", r)
		}
	}()

	c := &Collector{}
	c.Close() // não deve panic
}

// TestCollector_ReadMetrics_ZeroValue verifica que ReadMetrics em Collector zerado
// retorna um slice de NumVars zeros sem panic.
func TestCollector_ReadMetrics_ZeroValue(t *testing.T) {
	c := &Collector{}
	result := c.ReadMetrics()
	if len(result) != NumVars {
		t.Errorf("ReadMetrics: esperado slice de %d elementos, obtido %d", NumVars, len(result))
	}
}

// contains é um helper simples para não importar strings só para isso.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}