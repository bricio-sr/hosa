// Package syscgroup implementa a manipulação de cgroups v2 via escrita direta
// nos arquivos de controle expostos pelo kernel em /sys/fs/cgroup/.
// Nenhuma biblioteca de terceiros é utilizada — cgroups v2 é uma interface
// de arquivos de texto, e a stdlib do Go é suficiente para operá-la.
//
// Referência: linux/cgroup-v2.txt, man 7 cgroups
package syscgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// cgroupRoot é o ponto de montagem padrão do cgroup v2 no Linux moderno.
	cgroupRoot = "/sys/fs/cgroup"

	// hosaCgroup é o cgroup dedicado ao HOSA para isolar os processos monitorados.
	hosaCgroup = "hosa"
)

// CgroupPath retorna o caminho absoluto do cgroup de um processo pelo seu PID.
// O kernel expõe isso em /proc/<pid>/cgroup — a linha com "0::" indica o v2.
func CgroupPath(pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return "", fmt.Errorf("CgroupPath(pid=%d): %w", pid, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		// Formato cgroup v2: "0::<caminho>"
		if strings.HasPrefix(line, "0::") {
			cgPath := strings.TrimPrefix(line, "0::")
			cgPath = strings.TrimSpace(cgPath)
			return filepath.Join(cgroupRoot, cgPath), nil
		}
	}

	return "", fmt.Errorf("CgroupPath(pid=%d): cgroup v2 não encontrado (sistema usa v1?)", pid)
}

// WriteControl escreve um valor em um arquivo de controle do cgroup.
// É a operação primitiva — todos os controles de cgroup v2 são feitos assim.
//
// Exemplo: WriteControl("/sys/fs/cgroup/hosa", "memory.high", "1073741824")
func WriteControl(cgPath, filename, value string) error {
	controlFile := filepath.Join(cgPath, filename)

	if err := os.WriteFile(controlFile, []byte(value), 0644); err != nil {
		return fmt.Errorf("WriteControl(%q, %q=%q): %w", cgPath, filename, value, err)
	}

	return nil
}

// ReadControl lê o valor atual de um arquivo de controle do cgroup.
func ReadControl(cgPath, filename string) (string, error) {
	controlFile := filepath.Join(cgPath, filename)

	data, err := os.ReadFile(controlFile)
	if err != nil {
		return "", fmt.Errorf("ReadControl(%q, %q): %w", cgPath, filename, err)
	}

	return strings.TrimSpace(string(data)), nil
}

// SetMemoryHigh define o limite soft de memória (memory.high) de um cgroup.
// Quando um processo ultrapassa este limite, o kernel aplica throttling agressivo
// antes de recorrer ao OOM-Killer — é a alavanca principal de contenção do HOSA.
//
// bytes=0 remove o limite (escreve "max").
func SetMemoryHigh(cgPath string, bytes uint64) error {
	value := "max"
	if bytes > 0 {
		value = strconv.FormatUint(bytes, 10)
	}

	if err := WriteControl(cgPath, "memory.high", value); err != nil {
		return fmt.Errorf("SetMemoryHigh: %w", err)
	}

	return nil
}

// SetMemoryMax define o limite hard de memória (memory.max) de um cgroup.
// Processos que ultrapassam este limite são mortos pelo OOM-Killer do cgroup,
// não pelo OOM-Killer global — o impacto fica isolado.
//
// bytes=0 remove o limite (escreve "max").
func SetMemoryMax(cgPath string, bytes uint64) error {
	value := "max"
	if bytes > 0 {
		value = strconv.FormatUint(bytes, 10)
	}

	if err := WriteControl(cgPath, "memory.max", value); err != nil {
		return fmt.Errorf("SetMemoryMax: %w", err)
	}

	return nil
}

// GetMemoryCurrent retorna o uso atual de memória de um cgroup em bytes.
// Lê o arquivo memory.current — atualizado pelo kernel a cada página alocada.
func GetMemoryCurrent(cgPath string) (uint64, error) {
	val, err := ReadControl(cgPath, "memory.current")
	if err != nil {
		return 0, fmt.Errorf("GetMemoryCurrent: %w", err)
	}

	bytes, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("GetMemoryCurrent: valor inválido %q: %w", val, err)
	}

	return bytes, nil
}

// EnsureHosaCgroup garante que o cgroup /sys/fs/cgroup/hosa existe.
// Cgroups v2 são criados simplesmente com mkdir — o kernel cria os arquivos
// de controle automaticamente.
func EnsureHosaCgroup() (string, error) {
	path := filepath.Join(cgroupRoot, hosaCgroup)

	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("EnsureHosaCgroup: falha ao criar %q: %w", path, err)
	}

	return path, nil
}

// MoveProcess move um processo (pelo PID) para o cgroup do HOSA,
// escrevendo o PID no arquivo cgroup.procs.
// A partir deste momento, o processo passa a ser gerenciado pelo cgroup hosa.
func MoveProcess(cgPath string, pid int) error {
	if err := WriteControl(cgPath, "cgroup.procs", strconv.Itoa(pid)); err != nil {
		return fmt.Errorf("MoveProcess(pid=%d): %w", pid, err)
	}

	return nil
}