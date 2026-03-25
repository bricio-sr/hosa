// Package sensor inclui a propriocepção de hardware do HOSA —
// a capacidade do agente de descobrir automaticamente a topologia
// do nó em que está rodando, sem dependências externas.
//
// Referência: whitepaper HOSA, Seção 5.3 — Fase de Warm-Up e Calibração Proprioceptiva.
package sensor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Topology contém a descrição do hardware descoberta durante o warm-up.
// Usada pelo PredictiveCortex para calibrar limiares e pelo Thalamic Filter
// para decisões de supressão de telemetria.
type Topology struct {
	// PhysicalCores é o número de núcleos físicos (sem hyperthreading).
	PhysicalCores int

	// LogicalCores é o número de CPUs lógicas (com hyperthreading).
	LogicalCores int

	// NUMANodes é o número de nós NUMA. 1 = sistema UMA (comum em VMs).
	NUMANodes int

	// MemoryTotalBytes é a memória RAM total do host em bytes.
	MemoryTotalBytes uint64

	// CacheSizeL3Bytes é o tamanho do cache L3 em bytes. 0 se não detectado.
	CacheSizeL3Bytes uint64

	// IsVM indica se o sistema parece ser uma máquina virtual.
	// Detectado via DMI/hypervisor CPUID em /sys/hypervisor/ e /proc/cpuinfo.
	IsVM bool

	// HypervisorVendor é o nome do hypervisor quando IsVM=true. Vazio em bare metal.
	HypervisorVendor string
}

// String retorna um resumo legível da topologia — usado nos logs de inicialização.
func (t *Topology) String() string {
	vm := "bare metal"
	if t.IsVM {
		vm = fmt.Sprintf("VM (%s)", t.HypervisorVendor)
	}
	return fmt.Sprintf(
		"cores=%d/%d(físico/lógico) NUMA=%d mem=%.1fGB L3=%dMB env=%s",
		t.PhysicalCores, t.LogicalCores,
		t.NUMANodes,
		float64(t.MemoryTotalBytes)/(1<<30),
		t.CacheSizeL3Bytes/(1<<20),
		vm,
	)
}

// DiscoverTopology lê a topologia de hardware do nó via sysfs e /proc.
// Não requer privilégios de root — todos os caminhos são legíveis por qualquer usuário.
func DiscoverTopology() (*Topology, error) {
	t := &Topology{}
	var err error

	// 1. Memória total via /proc/meminfo
	t.MemoryTotalBytes, err = readMemTotalBytes()
	if err != nil {
		return nil, fmt.Errorf("proprioception: falha ao ler memória total: %w", err)
	}

	// 2. CPUs lógicas via /sys/devices/system/cpu/present
	t.LogicalCores, err = readCPUCount("/sys/devices/system/cpu/present")
	if err != nil {
		// Fallback: conta diretamente os diretórios cpu0, cpu1, ...
		t.LogicalCores = countCPUDirs()
	}

	// 3. Núcleos físicos via core_id (elimina hyperthreads duplicados)
	t.PhysicalCores = readPhysicalCores()
	if t.PhysicalCores == 0 {
		t.PhysicalCores = t.LogicalCores // fallback conservador
	}

	// 4. Nós NUMA via /sys/devices/system/node/
	t.NUMANodes = countNUMANodes()
	if t.NUMANodes == 0 {
		t.NUMANodes = 1 // UMA: trata como 1 nó
	}

	// 5. Cache L3 via /sys/devices/system/cpu/cpu0/cache/
	t.CacheSizeL3Bytes = readL3CacheSize()

	// 6. Detecção de VM via /sys/hypervisor/ e /proc/cpuinfo
	t.IsVM, t.HypervisorVendor = detectVM()

	return t, nil
}

// readMemTotalBytes lê a memória total de /proc/meminfo em bytes.
func readMemTotalBytes() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, err
		}
		return kb * 1024, nil
	}
	return 0, fmt.Errorf("MemTotal não encontrado em /proc/meminfo")
}

// readCPUCount lê um arquivo de range de CPU do kernel (ex: "0-7" ou "0,2,4-6")
// e retorna a contagem total de CPUs representadas.
func readCPUCount(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return parseCPURange(strings.TrimSpace(string(data))), nil
}

// parseCPURange parseia um range de CPUs no formato "0-3,5,7-9" e retorna a contagem.
func parseCPURange(s string) int {
	count := 0
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			if len(bounds) != 2 {
				continue
			}
			lo, err1 := strconv.Atoi(bounds[0])
			hi, err2 := strconv.Atoi(bounds[1])
			if err1 != nil || err2 != nil || hi < lo {
				continue
			}
			count += hi - lo + 1
		} else {
			if _, err := strconv.Atoi(part); err == nil {
				count++
			}
		}
	}
	return count
}

// countCPUDirs conta os diretórios cpu0, cpu1, ... em /sys/devices/system/cpu/.
func countCPUDirs() int {
	entries, err := os.ReadDir("/sys/devices/system/cpu")
	if err != nil {
		return 1
	}
	count := 0
	for _, e := range entries {
		name := e.Name()
		if len(name) > 3 && name[:3] == "cpu" {
			if _, err := strconv.Atoi(name[3:]); err == nil {
				count++
			}
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

// readPhysicalCores conta núcleos físicos únicos via core_id de cada CPU lógica.
// Elimina hyperthreads duplicados (mesmo socket + mesmo core_id = mesmo núcleo físico).
func readPhysicalCores() int {
	type coreKey struct{ socket, core int }
	seen := make(map[coreKey]struct{})

	entries, err := os.ReadDir("/sys/devices/system/cpu")
	if err != nil {
		return 0
	}

	for _, e := range entries {
		name := e.Name()
		if len(name) <= 3 || name[:3] != "cpu" {
			continue
		}
		if _, err := strconv.Atoi(name[3:]); err != nil {
			continue
		}

		cpuBase := filepath.Join("/sys/devices/system/cpu", name, "topology")

		coreID := readIntFile(filepath.Join(cpuBase, "core_id"))
		socketID := readIntFile(filepath.Join(cpuBase, "physical_package_id"))

		seen[coreKey{socketID, coreID}] = struct{}{}
	}

	return len(seen)
}

// countNUMANodes conta os diretórios node0, node1, ... em /sys/devices/system/node/.
func countNUMANodes() int {
	entries, err := os.ReadDir("/sys/devices/system/node")
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		name := e.Name()
		if len(name) > 4 && name[:4] == "node" {
			if _, err := strconv.Atoi(name[4:]); err == nil {
				count++
			}
		}
	}
	return count
}

// readL3CacheSize lê o tamanho do cache L3 do primeiro núcleo disponível.
// Percorre os índices de cache até encontrar o nível 3 (unified).
func readL3CacheSize() uint64 {
	cacheBase := "/sys/devices/system/cpu/cpu0/cache"
	entries, err := os.ReadDir(cacheBase)
	if err != nil {
		return 0
	}

	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "index") {
			continue
		}
		indexPath := filepath.Join(cacheBase, e.Name())

		level := readIntFile(filepath.Join(indexPath, "level"))
		if level != 3 {
			continue
		}

		// Lê o tamanho: pode estar em K, M, G
		sizeRaw, err := os.ReadFile(filepath.Join(indexPath, "size"))
		if err != nil {
			continue
		}
		return parseMemSize(strings.TrimSpace(string(sizeRaw)))
	}
	return 0
}

// parseMemSize converte strings como "8192K", "16M", "1G" para bytes.
func parseMemSize(s string) uint64 {
	if len(s) == 0 {
		return 0
	}
	suffix := s[len(s)-1]
	numStr := s
	var multiplier uint64 = 1

	switch suffix {
	case 'K', 'k':
		multiplier = 1024
		numStr = s[:len(s)-1]
	case 'M', 'm':
		multiplier = 1024 * 1024
		numStr = s[:len(s)-1]
	case 'G', 'g':
		multiplier = 1024 * 1024 * 1024
		numStr = s[:len(s)-1]
	}

	n, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return 0
	}
	return n * multiplier
}

// detectVM tenta detectar se o sistema está rodando em uma máquina virtual.
// Retorna (true, vendor) se detectado, (false, "") em bare metal.
func detectVM() (bool, string) {
	// 1. /sys/hypervisor/type (Xen)
	if data, err := os.ReadFile("/sys/hypervisor/type"); err == nil {
		vendor := strings.TrimSpace(string(data))
		if vendor != "" && vendor != "none" {
			return true, vendor
		}
	}

	// 2. /proc/cpuinfo — "hypervisor" flag e "vendor" de VM
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		content := string(data)
		if strings.Contains(content, "hypervisor") {
			// Tenta extrair o vendor do hypervisor
			for _, line := range strings.Split(content, "\n") {
				if strings.HasPrefix(line, "vendor_id") {
					if strings.Contains(line, "KVMKVMKVM") {
						return true, "KVM"
					}
					if strings.Contains(line, "Microsoft Hv") {
						return true, "Hyper-V"
					}
					if strings.Contains(line, "VMwareVMware") {
						return true, "VMware"
					}
				}
			}
			return true, "unknown"
		}
	}

	// 3. DMI via /sys/class/dmi/id/
	dmiFiles := map[string]string{
		"/sys/class/dmi/id/sys_vendor":     "",
		"/sys/class/dmi/id/product_name":   "",
		"/sys/class/dmi/id/board_vendor":   "",
	}
	vmKeywords := []string{"vmware", "virtualbox", "kvm", "qemu", "xen", "microsoft", "amazon", "google"}

	for path := range dmiFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lower := strings.ToLower(strings.TrimSpace(string(data)))
		for _, kw := range vmKeywords {
			if strings.Contains(lower, kw) {
				return true, strings.TrimSpace(string(data))
			}
		}
	}

	return false, ""
}

// readIntFile lê um inteiro de um arquivo sysfs. Retorna -1 em caso de erro.
func readIntFile(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return -1
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return -1
	}
	return n
}