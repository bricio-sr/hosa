package sensor

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCollector_StartMissingSensorsO verifica que Start() falha com mensagem
// clara quando o sensors.o não existe, evitando erros crípticos de syscall.
func TestCollector_StartMissingSensorsO(t *testing.T) {
	c := &Collector{}
	err := c.Start()
	if err == nil {
		t.Fatal("esperado erro quando sensors.o não existe, mas Start() retornou nil")
	}

	// A mensagem deve orientar o desenvolvedor a compilar o bytecode
	errMsg := err.Error()
	foundHint := false
	hints := []string{"sensors.o", "build-bpf", "não encontrado"}
	for _, hint := range hints {
		if contains(errMsg, hint) {
			foundHint = true
			break
		}
	}
	if !foundHint {
		t.Errorf("mensagem de erro não orienta o desenvolvedor: %q", errMsg)
	}
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