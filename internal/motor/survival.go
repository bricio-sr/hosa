// Package motor — SurvivalMotor: intervenção física da Fase 2.
//
// SurvivalMotor implementa o "Sistema Nervoso Simpático" do HOSA.
// Quando LevelSurvival é atingido (D_M ≥ 12.0), o motor aplica
// intervenção física além do throttling de cgroups da Fase 1:
//
// Tier 1 — sched_ext (Linux ≥ 6.11, CONFIG_SCHED_CLASS_EXT=y):
//   - Substitui o CFS pelo escalonador de sobrevivência via BPF STRUCT_OPS
//   - Targeted Starvation: processo ofensor recebe 0 ciclos de CPU
//   - Predictive Cache Affinity: processos vitais pinados em CPUs com L3 quente
//
// Tier 2 — fallback (qualquer kernel com cgroup v2):
//   - cpu.weight=1 no cgroup do ofensor (near-starvation no CFS)
//   - cpuset.cpus: ofensor isolado em CPUs não-compartilhadas com vitais
//   - memory.swappiness=200 no cgroup do ofensor (swap agressivo)
//   - memory.swappiness=0 + memory.swap.max=0 nos cgroups vitais (sem swap)
//
// O Tier 1 é detectado em runtime via probeSchedExt(). Se indisponível,
// o Tier 2 é aplicado automaticamente sem log de erro — é a degradação esperada.
//
// Referência: whitepaper HOSA, Seção 7.1 — Escalonador de Sobrevivência, Fase 2.
package motor

import (
	"fmt"
	"os"
	"strings"

	"github.com/bricio-sr/hosa/internal/sensor"
	"github.com/bricio-sr/hosa/internal/syscgroup"
	"github.com/bricio-sr/hosa/internal/sysbpf"
)

// SurvivalConfig agrupa os parâmetros do SurvivalMotor.
// Mapeado diretamente de config.SurvivalConfig.
type SurvivalConfig struct {
	// SchedExtBPFObject é o caminho para o objeto eBPF compilado do sched_ext.
	// Vazio = usar apenas o fallback (Tier 2).
	SchedExtBPFObject string

	// OffenderCgroupPath é o cgroup v2 do processo sob contenção.
	OffenderCgroupPath string

	// VitalCgroupPath é o cgroup v2 dos processos vitais protegidos.
	VitalCgroupPath string

	// CpuWeightStarve é o cpu.weight do ofensor em LevelSurvival (padrão: 1).
	CpuWeightStarve uint32

	// SwappinessOffender é o memory.swappiness do ofensor (0–200, padrão: 200).
	SwappinessOffender int

	// SwappinessVital é o memory.swappiness dos vitais (0–200, padrão: 0).
	SwappinessVital int
}

// SurvivalMotor executa a intervenção física da Fase 2.
type SurvivalMotor struct {
	cfg          SurvivalConfig
	topo         *sensor.Topology
	schedExtFD   int  // BPF link FD do sched_ext; -1 se não carregado
	active       bool // true enquanto LevelSurvival estiver ativo
	schedExtAvail bool // true se o kernel suporta sched_ext
}

// NewSurvivalMotor cria e inicializa o SurvivalMotor.
// Detecta suporte ao sched_ext em runtime — sem falhar se indisponível.
func NewSurvivalMotor(cfg SurvivalConfig, topo *sensor.Topology) (*SurvivalMotor, error) {
	sm := &SurvivalMotor{
		cfg:           cfg,
		topo:          topo,
		schedExtFD:    -1,
		schedExtAvail: probeSchedExt(),
	}

	// Garante que o cgroup vital exista
	if cfg.VitalCgroupPath != "" {
		if err := os.MkdirAll(cfg.VitalCgroupPath, 0755); err != nil {
			// Não fatal — cgroup pode já existir ou ser gerenciado externamente
			_ = err
		}
	}

	return sm, nil
}

// SchedExtAvailable retorna true se o kernel tem suporte a sched_ext.
func (sm *SurvivalMotor) SchedExtAvailable() bool {
	return sm.schedExtAvail
}

// Engage ativa a intervenção física de LevelSurvival.
// Idempotente — chamar múltiplas vezes não duplica ações.
func (sm *SurvivalMotor) Engage(memTotal uint64) error {
	if sm.active {
		return nil
	}

	var errs []string

	// --- Tier 2: fallback via cgroups (sempre disponível) ---
	if err := sm.applyFallback(); err != nil {
		errs = append(errs, fmt.Sprintf("fallback: %v", err))
	}

	// --- Tier 1: sched_ext survival scheduler (Linux ≥ 6.11) ---
	if sm.schedExtAvail && sm.cfg.SchedExtBPFObject != "" {
		if err := sm.loadSchedExt(); err != nil {
			// Não fatal — fallback já está ativo
			errs = append(errs, fmt.Sprintf("sched_ext: %v", err))
		}
	}

	sm.active = true

	if len(errs) > 0 {
		return fmt.Errorf("SurvivalMotor.Engage: %s", strings.Join(errs, "; "))
	}
	return nil
}

// Release desativa a intervenção física, restaurando o estado pré-Survival.
// Idempotente — seguro de chamar mesmo se Engage nunca foi chamado.
func (sm *SurvivalMotor) Release() error {
	if !sm.active {
		return nil
	}

	var errs []string

	// Desanexa o sched_ext (fechar o link FD restaura o CFS)
	if sm.schedExtFD >= 0 {
		if err := sysbpf.Close(sm.schedExtFD); err != nil {
			errs = append(errs, fmt.Sprintf("sched_ext release: %v", err))
		}
		sm.schedExtFD = -1
	}

	// Restaura cpu.weight padrão do ofensor
	if sm.cfg.OffenderCgroupPath != "" {
		if err := ResetCgroupCPUWeight(sm.cfg.OffenderCgroupPath); err != nil {
			errs = append(errs, fmt.Sprintf("cpu.weight reset: %v", err))
		}
		// Restaura swappiness padrão (60 = Linux default)
		if err := SetCgroupSwappiness(sm.cfg.OffenderCgroupPath, 60); err != nil {
			errs = append(errs, fmt.Sprintf("swappiness offender reset: %v", err))
		}
	}

	// Restaura swap.max dos vitais
	if sm.cfg.VitalCgroupPath != "" {
		if err := SetCgroupSwapMax(sm.cfg.VitalCgroupPath, -1); err != nil {
			errs = append(errs, fmt.Sprintf("swap.max vital reset: %v", err))
		}
		if err := SetCgroupSwappiness(sm.cfg.VitalCgroupPath, 60); err != nil {
			errs = append(errs, fmt.Sprintf("swappiness vital reset: %v", err))
		}
	}

	sm.active = false

	if len(errs) > 0 {
		return fmt.Errorf("SurvivalMotor.Release: %s", strings.Join(errs, "; "))
	}
	return nil
}

// Active retorna true se a intervenção de sobrevivência está ativa.
func (sm *SurvivalMotor) Active() bool {
	return sm.active
}

// ActionSummary retorna uma descrição das ações ativas para telemetria.
func (sm *SurvivalMotor) ActionSummary() string {
	tier := "tier2(cpu.weight=1+swappiness)"
	if sm.schedExtFD >= 0 {
		tier = "tier1(sched_ext)+tier2"
	}
	return tier
}

// applyFallback aplica as ações do Tier 2 via cgroup v2.
func (sm *SurvivalMotor) applyFallback() error {
	var errs []string

	if sm.cfg.OffenderCgroupPath != "" {
		// 1. CPU near-starvation via cpu.weight=1
		if err := SetCgroupCPUWeight(sm.cfg.OffenderCgroupPath, sm.cfg.CpuWeightStarve); err != nil {
			errs = append(errs, err.Error())
		}

		// 2. Pressão de swap agressiva no ofensor
		if err := SetCgroupSwappiness(sm.cfg.OffenderCgroupPath, sm.cfg.SwappinessOffender); err != nil {
			errs = append(errs, err.Error())
		}

		// 3. Pinagem de CPU: restringe ofensor a CPUs não usadas pelos vitais
		if sm.topo != nil && len(sm.topo.CacheGroups) > 1 {
			offenderCPUs := sm.selectOffenderCPUs()
			if len(offenderCPUs) > 0 {
				if err := SetCgroupCPUSet(sm.cfg.OffenderCgroupPath, offenderCPUs); err != nil {
					errs = append(errs, err.Error())
				}
			}
		}
	}

	if sm.cfg.VitalCgroupPath != "" {
		// 4. Page Table Isolation: vitais não podem ser swapados
		if err := SetCgroupSwapMax(sm.cfg.VitalCgroupPath, 0); err != nil {
			errs = append(errs, err.Error())
		}
		if err := SetCgroupSwappiness(sm.cfg.VitalCgroupPath, sm.cfg.SwappinessVital); err != nil {
			errs = append(errs, err.Error())
		}

		// 5. Predictive Cache Affinity: vitais pinados em CPUs com L3 quente
		if sm.topo != nil && len(sm.topo.CacheGroups) > 0 {
			vitalCPUs := sm.selectCacheWarmCPUs()
			if len(vitalCPUs) > 0 {
				if err := SetCgroupCPUSet(sm.cfg.VitalCgroupPath, vitalCPUs); err != nil {
					errs = append(errs, err.Error())
				}
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("applyFallback: %s", strings.Join(errs, "; "))
	}
	return nil
}

// loadSchedExt carrega o programa sched_ext eBPF e cria o link de ativação.
// O link FD é mantido em sm.schedExtFD — fechá-lo restaura o CFS automaticamente.
//
// Fluxo: abrir BTF vmlinux → encontrar type ID de sched_ext_ops → criar mapa STRUCT_OPS
//        → carregar progs → preencher mapa → BPF_LINK_CREATE → escalonador ativo.
func (sm *SurvivalMotor) loadSchedExt() error {
	obj, err := sysbpf.LoadObject(sm.cfg.SchedExtBPFObject)
	if err != nil {
		return fmt.Errorf("loadSchedExt: carregar objeto BPF: %w", err)
	}

	// 1. Cria o mapa de controle compartilhado (hosa_sched_ctrl)
	ctrlMap, err := sysbpf.CreateMap(sysbpf.BPF_MAP_TYPE_ARRAY, 4, 8, 8)
	if err != nil {
		return fmt.Errorf("loadSchedExt: criar mapa de controle: %w", err)
	}
	defer func() {
		if sm.schedExtFD < 0 {
			sysbpf.Close(int(ctrlMap))
		}
	}()

	// 2. Realoca referências do mapa no bytecode (por seção)
	fds := sysbpf.MapFDs{"hosa_sched_ctrl": ctrlMap}
	if err := obj.RelocateInsns(fds); err != nil {
		return fmt.Errorf("loadSchedExt: realocar referências: %w", err)
	}

	// 3. Carrega os programas struct_ops
	logBuf := make([]byte, 65536)
	var lastProgFD sysbpf.ProgFD = -1
	progCount := 0
	for section, insns := range obj.InsnsBySection {
		if !strings.HasPrefix(section, "struct_ops/") {
			continue
		}
		fd, err := sysbpf.LoadStructOpsProg(insns, obj.License, logBuf)
		if err != nil {
			return fmt.Errorf("loadSchedExt: carregar prog %q: %w", section, err)
		}
		lastProgFD = fd
		progCount++
	}

	if progCount == 0 {
		return fmt.Errorf("loadSchedExt: nenhuma seção struct_ops/ encontrada em %q", sm.cfg.SchedExtBPFObject)
	}

	// 4. Abre o BTF do kernel e localiza o type ID do sched_ext_ops
	btfFD, err := sysbpf.OpenVMLinuxBTF()
	if err != nil {
		return fmt.Errorf("loadSchedExt: abrir BTF vmlinux: %w", err)
	}
	defer sysbpf.Close(btfFD)

	typeID, err := sysbpf.FindSchedExtOpsTypeID()
	if err != nil {
		return fmt.Errorf("loadSchedExt: localizar sched_ext_ops no BTF: %w", err)
	}

	// 5. Cria o mapa BPF_MAP_TYPE_STRUCT_OPS e vincula ao kernel
	opsMap, err := sysbpf.CreateStructOpsMap(btfFD, typeID)
	if err != nil {
		if lastProgFD >= 0 {
			sysbpf.Close(int(lastProgFD))
		}
		return fmt.Errorf("loadSchedExt: criar mapa struct_ops: %w", err)
	}

	linkFD, err := sysbpf.LinkStructOps(opsMap)
	sysbpf.Close(int(opsMap)) // mapa pode ser fechado após o link estar ativo
	if lastProgFD >= 0 {
		sysbpf.Close(int(lastProgFD))
	}
	if err != nil {
		return fmt.Errorf("loadSchedExt: ativar link struct_ops: %w", err)
	}

	sm.schedExtFD = linkFD
	return nil
}

// selectCacheWarmCPUs retorna o grupo de CPUs com cache L3 mais quente
// para pinagem dos processos vitais. Usa o primeiro grupo de cache da topologia
// como heurística conservadora (grupo 0 = socket 0 = geralmente o socket primário).
func (sm *SurvivalMotor) selectCacheWarmCPUs() CPUSet {
	if len(sm.topo.CacheGroups) == 0 {
		return nil
	}
	return CPUSet(sm.topo.CacheGroups[0])
}

// selectOffenderCPUs retorna as CPUs que NÃO são usadas pelos vitais,
// para isolar o processo ofensor das CPUs com L3 quente.
func (sm *SurvivalMotor) selectOffenderCPUs() CPUSet {
	if len(sm.topo.CacheGroups) < 2 {
		return nil // Sistema single-socket: sem isolamento de cache disponível
	}
	// Em sistemas multi-socket: ofensor vai para o segundo grupo de cache
	return CPUSet(sm.topo.CacheGroups[1])
}

// probeSchedExt verifica se o kernel atual tem suporte a sched_ext.
// Testa a existência de /sys/kernel/sched_ext/ — criado pelo kernel quando
// CONFIG_SCHED_CLASS_EXT=y está habilitado.
func probeSchedExt() bool {
	_, err := os.Stat("/sys/kernel/sched_ext")
	return err == nil
}

// readSysFile lê o conteúdo de um arquivo sysfs como string.
// Usado internamente em cpuset.go e survival.go para evitar import circular.
func readSysFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// EnsureVitalCgroup garante que o cgroup vital existe e tem os controladores necessários.
// Deve ser chamado durante a inicialização, antes de Engage.
func EnsureVitalCgroup(vitalPath string) error {
	if err := os.MkdirAll(vitalPath, 0755); err != nil {
		return fmt.Errorf("EnsureVitalCgroup: %w", err)
	}

	// Verifica se o controlador cpu está disponível
	parent := vitalPath[:strings.LastIndex(vitalPath, "/")]
	controllers, err := syscgroup.ReadControl(parent, "cgroup.controllers")
	if err != nil {
		return nil // Não fatal — pode não ter acesso de leitura
	}

	// Habilita controladores necessários no pai se ainda não estiverem ativos
	needed := []string{"cpu", "cpuset", "memory"}
	subtree, _ := syscgroup.ReadControl(parent, "cgroup.subtree_control")
	for _, ctrl := range needed {
		if strings.Contains(controllers, ctrl) && !strings.Contains(subtree, ctrl) {
			_ = syscgroup.WriteControl(parent, "cgroup.subtree_control", "+"+ctrl)
		}
	}

	return nil
}

// SurvivalActionSummary retorna uma string descritiva das ações do LevelSurvival
// para uso em telemetria, dado o estado atual do motor.
func SurvivalActionSummary(cfg SurvivalConfig, topo *sensor.Topology, schedExtActive bool) string {
	parts := []string{
		fmt.Sprintf("cpu.weight=%d", cfg.CpuWeightStarve),
		fmt.Sprintf("swappiness_offender=%d", cfg.SwappinessOffender),
		"swap.max_vital=0",
	}

	if topo != nil && len(topo.CacheGroups) > 1 {
		parts = append(parts, "cache_affinity=multi_socket")
	}

	if schedExtActive {
		parts = append(parts, "sched_ext=active")
	} else {
		parts = append(parts, "sched_ext=unavailable(fallback)")
	}

	return strings.Join(parts, " ")
}

