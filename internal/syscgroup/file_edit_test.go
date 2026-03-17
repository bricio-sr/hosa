package syscgroup

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// simulateCgroup cria uma estrutura de diretório temporária imitando um cgroup v2.
// Retorna o caminho e uma função de cleanup.
func simulateCgroup(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "hosa_cgroup_test_*")
	if err != nil {
		t.Fatalf("falha ao criar cgroup simulado: %v", err)
	}
	return dir, func() { os.RemoveAll(dir) }
}

// TestWriteControl verifica que WriteControl cria o arquivo com o valor correto.
func TestWriteControl(t *testing.T) {
	cgPath, cleanup := simulateCgroup(t)
	defer cleanup()

	if err := WriteControl(cgPath, "memory.high", "1073741824"); err != nil {
		t.Fatalf("WriteControl falhou: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(cgPath, "memory.high"))
	if err != nil {
		t.Fatalf("falha ao ler arquivo escrito: %v", err)
	}
	if string(data) != "1073741824" {
		t.Errorf("conteúdo esperado %q, obtido %q", "1073741824", string(data))
	}
}

// TestReadControl verifica que ReadControl lê e faz trim do valor corretamente.
func TestReadControl(t *testing.T) {
	cgPath, cleanup := simulateCgroup(t)
	defer cleanup()

	// Escreve manualmente com espaço/newline como o kernel faria
	controlFile := filepath.Join(cgPath, "memory.current")
	if err := os.WriteFile(controlFile, []byte("524288000\n"), 0644); err != nil {
		t.Fatalf("falha ao preparar fixture: %v", err)
	}

	val, err := ReadControl(cgPath, "memory.current")
	if err != nil {
		t.Fatalf("ReadControl falhou: %v", err)
	}
	if val != "524288000" {
		t.Errorf("valor esperado %q, obtido %q", "524288000", val)
	}
}

// TestSetMemoryHigh_WithLimit verifica que um valor em bytes é escrito corretamente.
func TestSetMemoryHigh_WithLimit(t *testing.T) {
	cgPath, cleanup := simulateCgroup(t)
	defer cleanup()

	const limit = uint64(1 << 30) // 1 GB
	if err := SetMemoryHigh(cgPath, limit); err != nil {
		t.Fatalf("SetMemoryHigh falhou: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(cgPath, "memory.high"))
	got := string(data)
	expected := strconv.FormatUint(limit, 10)
	if got != expected {
		t.Errorf("esperado %q, obtido %q", expected, got)
	}
}

// TestSetMemoryHigh_RemoveLimit verifica que bytes=0 escreve "max".
func TestSetMemoryHigh_RemoveLimit(t *testing.T) {
	cgPath, cleanup := simulateCgroup(t)
	defer cleanup()

	if err := SetMemoryHigh(cgPath, 0); err != nil {
		t.Fatalf("SetMemoryHigh(0) falhou: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(cgPath, "memory.high"))
	if string(data) != "max" {
		t.Errorf("esperado %q, obtido %q", "max", string(data))
	}
}

// TestGetMemoryCurrent verifica o parsing do uso de memória.
func TestGetMemoryCurrent(t *testing.T) {
	cgPath, cleanup := simulateCgroup(t)
	defer cleanup()

	const expected = uint64(786432000) // ~750 MB
	controlFile := filepath.Join(cgPath, "memory.current")
	os.WriteFile(controlFile, []byte(strconv.FormatUint(expected, 10)+"\n"), 0644)

	got, err := GetMemoryCurrent(cgPath)
	if err != nil {
		t.Fatalf("GetMemoryCurrent falhou: %v", err)
	}
	if got != expected {
		t.Errorf("esperado %d, obtido %d", expected, got)
	}
}

// TestGetMemoryCurrent_InvalidValue verifica que conteúdo não numérico retorna erro.
func TestGetMemoryCurrent_InvalidValue(t *testing.T) {
	cgPath, cleanup := simulateCgroup(t)
	defer cleanup()

	os.WriteFile(filepath.Join(cgPath, "memory.current"), []byte("not_a_number\n"), 0644)

	_, err := GetMemoryCurrent(cgPath)
	if err == nil {
		t.Fatal("esperado erro para valor não numérico, mas nenhum erro retornado")
	}
}

// TestCgroupPath_InvalidPID verifica que um PID inexistente retorna erro.
func TestCgroupPath_InvalidPID(t *testing.T) {
	_, err := CgroupPath(99999999)
	if err == nil {
		t.Fatal("esperado erro para PID inexistente, mas nenhum erro retornado")
	}
}