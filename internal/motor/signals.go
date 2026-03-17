package motor

import (
	"fmt"
	"log"
	"os"
	"syscall"
)

// SendSIGSTOP suspende um processo temporariamente (SIGSTOP não pode ser ignorado).
// Usado como último recurso antes de SIGKILL — congela o processo sem perder estado,
// dando tempo para o operador intervir ou para o cgroup OOM-Killer agir de forma
// mais controlada.
func SendSIGSTOP(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("SendSIGSTOP(pid=%d): processo não encontrado: %w", pid, err)
	}

	if err := proc.Signal(syscall.SIGSTOP); err != nil {
		return fmt.Errorf("SendSIGSTOP(pid=%d): %w", pid, err)
	}

	log.Printf("HOSA Motor [PROTEÇÃO]: SIGSTOP enviado ao pid=%d", pid)
	return nil
}

// SendSIGCONT retoma um processo previamente suspenso por SIGSTOP.
// Chamado quando o nível de estresse cai abaixo do limiar de proteção.
func SendSIGCONT(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("SendSIGCONT(pid=%d): processo não encontrado: %w", pid, err)
	}

	if err := proc.Signal(syscall.SIGCONT); err != nil {
		return fmt.Errorf("SendSIGCONT(pid=%d): %w", pid, err)
	}

	log.Printf("HOSA Motor [RECUPERAÇÃO]: SIGCONT enviado ao pid=%d", pid)
	return nil
}