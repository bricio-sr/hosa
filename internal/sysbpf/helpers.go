package sysbpf

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// readTracepointID lê o ID numérico de um tracepoint a partir do debugfs.
// O kernel expõe este arquivo em:
// /sys/kernel/debug/tracing/events/<subsystem>/<event>/id
//
// Este ID é necessário para o perf_event_open(2) ao criar um perf event
// do tipo PERF_TYPE_TRACEPOINT.
func readTracepointID(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("readTracepointID: erro ao ler %q: %w", path, err)
	}

	idStr := strings.TrimSpace(string(data))
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("readTracepointID: valor inválido em %q: %q", path, idStr)
	}

	return id, nil
}