package sysbpf

import (
	"os"
	"testing"
)

// TestReadTracepointID_InvalidPath verifica que um caminho inexistente retorna erro.
func TestReadTracepointID_InvalidPath(t *testing.T) {
	_, err := readTracepointID("/tmp/hosa_test_nonexistent_tracepoint_id")
	if err == nil {
		t.Fatal("esperado erro para caminho inexistente, mas nenhum erro retornado")
	}
}

// TestReadTracepointID_ValidContent verifica que o parsing do ID funciona corretamente.
func TestReadTracepointID_ValidContent(t *testing.T) {
	// Cria um arquivo temporário simulando o conteúdo do debugfs
	f, err := os.CreateTemp("", "hosa_tp_id_*")
	if err != nil {
		t.Fatalf("falha ao criar arquivo temporário: %v", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString("42\n"); err != nil {
		t.Fatalf("falha ao escrever no arquivo temporário: %v", err)
	}
	f.Close()

	id, err := readTracepointID(f.Name())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if id != 42 {
		t.Errorf("ID esperado: 42, obtido: %d", id)
	}
}

// TestReadTracepointID_InvalidContent verifica que conteúdo não numérico retorna erro.
func TestReadTracepointID_InvalidContent(t *testing.T) {
	f, err := os.CreateTemp("", "hosa_tp_id_*")
	if err != nil {
		t.Fatalf("falha ao criar arquivo temporário: %v", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString("nao_e_numero\n"); err != nil {
		t.Fatalf("falha ao escrever no arquivo temporário: %v", err)
	}
	f.Close()

	_, err = readTracepointID(f.Name())
	if err == nil {
		t.Fatal("esperado erro para conteúdo não numérico, mas nenhum erro retornado")
	}
}

// TestLoadObject_InvalidPath verifica que LoadObject retorna erro para arquivo inexistente.
func TestLoadObject_InvalidPath(t *testing.T) {
	_, err := LoadObject("/tmp/hosa_test_nonexistent.o")
	if err == nil {
		t.Fatal("esperado erro para arquivo inexistente, mas nenhum erro retornado")
	}
}

// TestLoadObject_InvalidMagic verifica que um arquivo com magic ELF inválido é rejeitado.
func TestLoadObject_InvalidMagic(t *testing.T) {
	f, err := os.CreateTemp("", "hosa_elf_*")
	if err != nil {
		t.Fatalf("falha ao criar arquivo temporário: %v", err)
	}
	defer os.Remove(f.Name())

	// Escreve 64 bytes com magic inválido
	payload := make([]byte, 64)
	copy(payload, []byte("NOTELF"))
	if _, err := f.Write(payload); err != nil {
		t.Fatalf("falha ao escrever: %v", err)
	}
	f.Close()

	_, err = LoadObject(f.Name())
	if err == nil {
		t.Fatal("esperado erro para magic ELF inválido, mas nenhum erro retornado")
	}
}

// TestIsProgSection verifica o reconhecimento de seções de bytecode eBPF.
func TestIsProgSection(t *testing.T) {
	cases := []struct {
		name     string
		expected bool
	}{
		{"tracepoint/syscalls/sys_enter_brk", true},
		{"kprobe/do_sys_open", true},
		{"xdp_prog", true},
		{"tc/ingress", true},
		{".maps", false},
		{"license", false},
		{".text", false},
		{"", false},
	}

	for _, c := range cases {
		got := isProgSection(c.name)
		if got != c.expected {
			t.Errorf("isProgSection(%q) = %v, esperado %v", c.name, got, c.expected)
		}
	}
}