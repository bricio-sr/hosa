# Arquitetura do HOSA — Deep Dive

> **Este documento cobre em profundidade a arquitetura interna do HOSA.**
> Para o modelo matemático, veja [`docs/math_model.md`](math_model.md).
> Para a fundamentação teórica, veja o [Whitepaper v2.1](whitepaper.pdf).

---

## Índice

- [Filosofia Arquitetural](#filosofia-arquitetural)
- [A Analogia Biológica](#a-analogia-biológica)
- [Visão Geral do Sistema](#visão-geral-do-sistema)
- [Camada 1 — Kernel Space (eBPF)](#camada-1--kernel-space-ebpf)
  - [Sondas Sensoriais](#sondas-sensoriais)
  - [Atuadores](#atuadores)
  - [Transporte via Ring Buffer](#transporte-via-ring-buffer)
- [Camada 2 — User Space](#camada-2--user-space)
  - [O Sistema Sensorial (`internal/sensor`)](#o-sistema-sensorial-internalsensor)
  - [O Sistema Límbico (`internal/state`)](#o-sistema-límbico-internalstate)
  - [O Córtex Preditivo (`internal/brain`)](#o-córtex-preditivo-internalbrain)
  - [O Arco Reflexo (`internal/motor`)](#o-arco-reflexo-internalmotor)
  - [Comunicação Oportunista](#comunicação-oportunista)
- [Subsistemas Transversais](#subsistemas-transversais)
  - [Propriocepção de Hardware — Fase de Warm-Up](#propriocepção-de-hardware--fase-de-warm-up)
  - [Sistema de Resposta Graduada (Níveis 0–5)](#sistema-de-resposta-graduada-níveis-05)
  - [Habituação — Deriva do Perfil Basal](#habituação--deriva-do-perfil-basal)
  - [Filtro Talâmico — Supressão de Telemetria](#filtro-talâmico--supressão-de-telemetria)
  - [Modos de Quarentena por Ambiente](#modos-de-quarentena-por-ambiente)
- [Fluxo de Dados — Ponta a Ponta](#fluxo-de-dados--ponta-a-ponta)
- [Máquina de Estados](#máquina-de-estados)
- [A Safelist](#a-safelist)
- [Decisões de Design](#decisões-de-design)
- [Características de Performance](#características-de-performance)
- [Postura de Segurança](#postura-de-segurança)
- [Limitações](#limitações)

---

## Filosofia Arquitetural

O HOSA é governado por cinco princípios não-negociáveis:

| Princípio | Descrição |
|---|---|
| **Autonomia Local** | O ciclo completo de detecção e mitigação deve rodar sem conectividade de rede, APIs externas ou intervenção humana. |
| **Zero Dependências Externas de Runtime** | Todas as dependências estão dentro do binário ou dentro do kernel hospedeiro. Comunicação com sistemas externos (orquestradores, dashboards) é *oportunista* — executada quando disponível, nunca requerida. |
| **Footprint Computacional Previsível** | Memória é `O(1)` além da matriz de covariância `O(n²)`. Consumo de CPU é configurável e limitado. O agente não pode se tornar a causa do problema que pretende resolver. |
| **Resposta Graduada** | Mitigação é um espectro, não um interruptor binário. Cada ação é proporcional à severidade da anomalia *e à sua taxa de variação*. |
| **Observabilidade da Decisão** | Cada ação autônoma é registrada com sua justificativa matemática completa: o valor de `D_M`, a derivada, o limiar acionado, as dimensões contribuintes e a ação executada. O agente é totalmente auditável. |

---

## A Analogia Biológica

A arquitetura do HOSA mapeia diretamente para o sistema nervoso humano. Isso não é cosmético — a analogia dirige as decisões de design.

| Sistema Biológico | Componente HOSA | Módulo | Papel |
|---|---|---|---|
| **Sistema nervoso periférico** (receptores sensoriais) | Sondas eBPF | `internal/bpf/sensors.c` | Coleta sinais brutos do kernel |
| **Fibras nervosas aferentes** | eBPF ring buffer | Transporte kernel ↔ user space | Conduz sinais dos sensores ao córtex |
| **Arco reflexo medular** | Motor + lógica de resposta | `internal/motor/` | Executa contenção sem esperar o córtex |
| **Sistema límbico** (memória de curto prazo) | Ring buffer de baseline | `internal/state/memory.go` | Mantém estado recente para cálculo de derivadas |
| **Córtex pré-frontal** (reconhecimento de padrões) | Motor Mahalanobis | `internal/brain/` | Calcula desvio em relação ao perfil basal |
| **Tálamo** (filtragem de sinais) | Supressor de telemetria | Filtro Talâmico | Suprime sinais redundantes em homeostase |
| **Neuroplasticidade** (habituação) | Recalibração do baseline | Decaimento de pesos no Welford | Adapta-se a mudanças legítimas de workload |
| **Sistema nervoso simpático** (resposta a ameaças) | Resposta graduada | Lógica de escalação de nível | Escala a resposta de forma proporcional à ameaça |

O insight central da biologia: **a medula espinhal age antes que o cérebro seja notificado.** A camada motor do HOSA aplica contenção em milissegundos. O orquestrador externo (o "cérebro") é notificado *após* o reflexo — via webhook, não antes.

---

## Visão Geral do Sistema

```
┌─────────────────────────────────────────────────────────────────────┐
│                         KERNEL SPACE (eBPF)                         │
│                                                                     │
│  ┌──────────────────┐  ┌──────────────────┐  ┌───────────────────┐  │
│  │ SONDAS SENSORIAIS│  │ SONDAS SENSORIAIS│  │    ATUADORES      │  │
│  │  (tracepoints)   │  │  (kprobes, PSI)  │  │  (XDP / cgroup    │  │
│  │                  │  │                  │  │   controllers)    │  │
│  │ · sched_*        │  │ · mm_page_alloc  │  │                   │  │
│  │ · net_dev_*      │  │ · psi hooks      │  │ · memory.high     │  │
│  │ · block_rq_*     │  │ · tcp_*          │  │ · cpu.max         │  │
│  └────────┬─────────┘  └────────┬─────────┘  └────────▲──────────┘  │
│           │                     │                      │             │
│           ▼                     ▼                      │             │
│  ┌──────────────────────────────────────────┐          │             │
│  │            eBPF RING BUFFER              │          │             │
│  │         (transporte de baixa latência)   │          │             │
│  └─────────────────────┬────────────────────┘          │             │
│                        │                      ┌────────┴──────────┐  │
│                        │                      │    BPF MAPS       │  │
│                        │                      │  (I/O de comandos)│  │
│                        │                      └───────────────────┘  │
├────────────────────────┼──────────────────────────────┬──────────────┤
│                        │        USER SPACE             │              │
│                        ▼                               │              │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                   SISTEMA SENSORIAL                             │ │
│  │                   internal/sensor/collector.go                  │ │
│  │  · Lê eventos do eBPF ring buffer                               │ │
│  │  · Normaliza dados brutos do kernel no vetor de estado x(t)     │ │
│  └─────────────────────────────┬───────────────────────────────────┘ │
│                                │                                      │
│                                ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                   SISTEMA LÍMBICO                               │ │
│  │                   internal/state/memory.go                      │ │
│  │  · Ring buffer de curto prazo com vetores de estado recentes    │ │
│  │  · Alimenta o cálculo de derivadas no córtex                    │ │
│  └─────────────────────────────┬───────────────────────────────────┘ │
│                                │                                      │
│                                ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                   CÓRTEX PREDITIVO                              │ │
│  │        internal/brain/{matrix, mahalanobis, predictor}.go       │ │
│  │                                                                 │ │
│  │  1. Atualiza μ e Σ incrementalmente  (Welford)                  │ │
│  │  2. Calcula D_M(x(t))                (Mahalanobis)              │ │
│  │  3. Aplica suavização EWMA →  D̄_M(t)                           │ │
│  │  4. Calcula dD̄_M/dt e d²D̄_M/dt²     (Predictor)                │ │
│  │  5. Avalia contra limiares adaptativos                          │ │
│  │  6. Determina nível de resposta (0–5)                           │ │
│  └────────────────────┬────────────────────┬────────────────────────┘ │
│                       │                    │                           │
│                       ▼                    ▼                           │
│  ┌──────────────────────────┐  ┌────────────────────────────────────┐ │
│  │      ARCO REFLEXO        │  │   COMUNICAÇÃO OPORTUNISTA          │ │
│  │  internal/motor/         │  │                                    │ │
│  │                          │  │  · Webhooks para orquestradores    │ │
│  │  · cgroups.go            │  │  · Endpoint compatível Prometheus  │ │
│  │    throttling CPU/mem    │  │  · Log de auditoria estruturado    │ │
│  │  · signals.go            │  │  · /healthz com vetor de estado    │ │
│  │    SIGSTOP / SIGTERM     │  │                                    │ │
│  └──────────────────────────┘  └────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Camada 1 — Kernel Space (eBPF)

Toda coleta de sinais e atuação de baixo nível acontece em kernel space. O HOSA nunca faz polling em `/proc`. Ele se conecta diretamente nos caminhos de execução do kernel.

### Sondas Sensoriais

Localizadas em `internal/bpf/sensors.c`. Compiladas para bytecode eBPF via `clang` e carregadas em runtime pelo loader customizado em `internal/sysbpf/syscall.go`.

**Tracepoints** (interface estável e versionada do kernel):

| Tracepoint | Dado coletado | Dimensão no vetor de estado |
|---|---|---|
| `sched_switch` | Trocas de contexto, profundidade da run queue | `runqueue`, `ctx_switches` |
| `sched_process_exit` | Término inesperado de processos | Detecção de silêncio anômalo |
| `net_dev_xmit` / `netif_receive_skb` | Pacotes por segundo, bytes entrada/saída | `net_tx`, `net_rx` |
| `block_rq_issue` / `block_rq_complete` | IOPS, latência de I/O (delta issue→complete) | `io_throughput`, `io_latency` |

**kprobes** (hooks dinâmicos em funções do kernel, sensíveis à versão):

| kprobe | Dado coletado | Dimensão no vetor de estado |
|---|---|---|
| `mm_page_alloc` | Taxa de alocação de memória | `mem_alloc_rate` |
| `try_charge` (cgroup memory) | Cobrança de memória por cgroup | Atribuição por cgroup |
| `tcp_retransmit_skb` | Taxa de retransmissão TCP | `net_retransmit` |

**Hooks PSI** (Pressure Stall Information — Weiner, 2018):

O HOSA lê PSI diretamente de `/sys/fs/cgroup/.../memory.pressure` e `/proc/pressure/`, em vez de se conectar ao caminho kernel do subsistema PSI. Os dados PSI são consumidos como parte do lote de eventos do ring buffer.

| Métrica PSI | Mapeada para |
|---|---|
| `memory some avg10` | `mem_pressure` |
| `cpu some avg10` | `cpu_pressure` |
| `io some avg10` | `io_pressure` |

### Atuadores

Comandos de atuação são escritos em **BPF maps** pelo córtex em user space e lidos por um pequeno programa eBPF acoplado à camada de cgroup.

**Atuação via cgroups v2** (`internal/motor/cgroups.go`):

O HOSA escreve diretamente nos arquivos de controle do cgroup via Linux VFS — sem biblioteca externa. A implementação vive em `internal/syscgroup/file_edit.go` e usa syscalls brutas `open`/`write`/`close` para latência determinística.

| Arquivo | Efeito | Nível de resposta |
|---|---|---|
| `memory.high` | Aplica backpressure de reclaim agressivo — desacelera alocação sem matar | Nível 2+ |
| `memory.max` | Limite rígido — dispara OOM dentro do cgroup antes do OOM-Killer global | Nível 4 |
| `cpu.max` | Throttling de banda de CPU (quota CFS) | Nível 3+ |
| `cgroup.freeze` | Suspende todos os processos no cgroup (similar a SIGSTOP) | Nível 5 |

**Atuação XDP**:

A política de descarte de pacotes é aplicada no nível do driver via um programa XDP carregado na interface de rede primária do nó. As regras de descarte são atualizadas via BPF maps a partir do user space.

| Ação XDP | Efeito | Nível de resposta |
|---|---|---|
| Descarta novos pacotes SYN | Rejeita novas conexões de entrada, preserva as existentes | Nível 3 |
| Descarta todo tráfego de entrada exceto IPs de healthcheck | Load shedding agressivo | Nível 4 |
| Descarta todo tráfego entrada + saída exceto gerência | Isolamento de rede | Nível 5 |

### Transporte via Ring Buffer

O eBPF ring buffer (`BPF_MAP_TYPE_RINGBUF`) é o único transporte entre kernel e user space. Ele é:

- **Lock-free** — produtor único (kernel), consumidor único (goroutine em user space)
- **Zero-copy** — user space lê diretamente da memória compartilhada, sem cópia kernel→user
- **Ciente de back-pressure** — se o consumidor atrasar, eventos mais antigos são descartados (contador de eventos perdidos exposto via endpoint de métricas)

Latência típica kernel→user: **1–10 μs** em hardware moderno.

---

## Camada 2 — User Space

### O Sistema Sensorial (`internal/sensor`)

**Arquivo:** `internal/sensor/collector.go`

Responsabilidades:
- Consome o eBPF ring buffer em uma goroutine dedicada
- Normaliza contadores brutos do kernel no vetor de estado `x(t) ∈ ℝⁿ`
- Computa deltas por intervalo (ex: *taxa* de trocas de contexto a partir de contador cumulativo)
- Resolve atribuição de cgroup por evento (mapeia `task → caminho do cgroup` para identificação do processo contribuinte)

O collector emite uma struct `StateEvent` no intervalo de amostragem configurado (padrão 100ms em homeostase, adaptativo até 10ms em vigilância). Essa struct é a representação canônica do estado do nó no tempo `t`.

```go
// StateEvent — representação canônica do estado
type StateEvent struct {
    Timestamp   time.Time
    Vector      []float64   // x(t): vetor de estado normalizado
    CgroupMap   map[string][]float64  // breakdown de recursos por cgroup
    SamplingMs  int         // intervalo de amostragem adaptativo atual
}
```

### O Sistema Límbico (`internal/state`)

**Arquivo:** `internal/state/memory.go`

Um ring buffer de tamanho fixo com structs `StateEvent` recentes. Esta é a **memória de curto prazo** do agente — não cresce com o tempo.

- Tamanho: configurável, padrão 1000 amostras (~100s a 100ms de amostragem)
- Usado pelo córtex para: suavização EWMA, cálculo de derivadas, e checagem da condição de habituação

Footprint de memória: `O(window_size × n)` — com `n = 10` dimensões e 1000 amostras, aproximadamente **80KB**. Fixo na inicialização.

### O Córtex Preditivo (`internal/brain`)

O córtex é o núcleo computacional. Implementado em três arquivos:

---

#### `internal/brain/matrix.go` — Gerenciamento da Matriz de Covariância

Implementa o **algoritmo de Welford incremental** para atualizações online de `μ` e `Σ`:

```
Para cada nova amostra x(t):
  n += 1
  delta = x(t) - μ
  μ += delta / n
  delta2 = x(t) - μ
  M += produto_externo(delta, delta2)   // M₂ do Welford
  Σ = M / (n - 1)
```

**Memória:** `O(n²)` — para `n = 10`, `Σ` é uma matriz 10×10 = 800 bytes. Fixo.
**Custo por amostra:** `O(n²)` — alguns microssegundos para `n ≤ 15`.

**Regularização de Tikhonov** é aplicada antes da inversão para tratar matrizes próximas de singular (variáveis colineares):

```
Σ_reg = Σ + λI
```

onde `λ` é autoajustado durante o warm-up com base no número de condição observado de `Σ`.

**Inversão de Cholesky** (`Σ⁻¹`) é recomputada apenas quando `Σ` mudou significativamente (rastreado via delta da norma de Frobenius), não a cada amostra. Isso amortiza o custo `O(n³)` da inversão entre muitas amostras.

---

#### `internal/brain/mahalanobis.go` — Cálculo da Homeostase

Computa a Distância de Mahalanobis atual:

```
D_M(x(t)) = sqrt( (x(t) - μ)ᵀ Σ⁻¹ (x(t) - μ) )
```

Também computa:

**Decomposição de contribuição dimensional** — identifica *quais recursos* estão conduzindo o desvio:

```
d = x(t) - μ
c_j = d_j × (Σ⁻¹ d)_j
```

Os valores `c_j` são registrados junto a cada decisão e incluídos nos payloads de webhook.

**Índice de Direção de Carga (φ)** — determina se o desvio é em direção à sobrecarga ou ociosidade:

```
φ(t) = (1/n) Σ_j  s_j × (d_j / σ_j)
```

onde `s_j ∈ {+1, -1}` é o sinal de carga da variável `j` (pré-configurado na inicialização: +1 para utilização de CPU, memória usada, etc.; -1 para CPU idle, memória livre, etc.).

**Razão de deformação de covariância (ρ)** — mede se a *estrutura* das correlações mudou, não apenas a magnitude:

```
ρ(t) = ‖Σ_recente - Σ_basal‖_F / ‖Σ_basal‖_F
```

`ρ` alto com `D_M` contido é a assinatura de demanda adversarial (Regime +3).

---

#### `internal/brain/predictor.go` — Derivadas e Estimativa de Tempo para Falha

**Suavização EWMA** antes da diferenciação (evita diferenciação numérica mal-posta em dados ruidosos):

```
D̄_M(t) = α × D_M(t) + (1 - α) × D̄_M(t-1)
```

`α` é calibrado por recurso durante o warm-up com base na variância observada do sinal.

**Primeira derivada** (velocidade de afastamento da homeostase):

```
dD̄_M/dt ≈ (D̄_M(t) - D̄_M(t-Δt)) / Δt
```

**Segunda derivada** (aceleração — o sistema está indo *mais rápido* em direção ao colapso?):

```
d²D̄_M/dt² ≈ (dD̄_M/dt(t) - dD̄_M/dt(t-Δt)) / Δt
```

**Estimativa de Tempo para Falha — TTF** (usada na ativação do Nível 4):

Se `dD̄_M/dt > 0` e `D_M` está em trajetória em direção ao limiar `θ₄`:

```
TTF = (θ₄ - D̄_M(t)) / (dD̄_M/dt)
```

O Nível 4 é ativado quando `TTF < T_critico` (configurável, padrão 10s), *mesmo que* `D_M` ainda não tenha atingido `θ₄`. Esta é contenção proativa, não reativa.

### O Arco Reflexo (`internal/motor`)

**Arquivos:** `internal/motor/cgroups.go`, `internal/motor/signals.go`

A camada motor traduz decisões do córtex em ações do kernel. É intencionalmente simples — nenhuma lógica de decisão aqui. O córtex decide; o motor executa.

**`cgroups.go`** — Manipulação direta de arquivos de cgroup via `internal/syscgroup/file_edit.go`. Usa escritas VFS brutas, não `libcgroup` nem dependência externa. Todas as escritas são atômicas (único syscall `write()` no pseudo-arquivo do cgroup).

**`signals.go`** — Sinalização de processos para o Nível 5 (quarentena). Envia `SIGSTOP` para congelar processos não-críticos e `SIGTERM` (com fallback `SIGKILL`) para processos explicitamente identificados como destrutivos. Nunca toca entradas da safelist.

**Atomicidade e ordenação das ações:** Escritas em cgroup e atualizações do mapa XDP são aplicadas em ordem definida para minimizar a janela de inconsistência:

1. Define backpressure `memory.high` (mais suave, primeiro)
2. Atualiza regras de descarte XDP (nível de rede)
3. Define throttle `cpu.max`
4. Define limite rígido `memory.max`
5. `cgroup.freeze` (mais disruptivo, por último)

Rollback em qualquer falha de etapa registra o estado parcial e escala para o próximo nível de resposta.

### Comunicação Oportunista

Toda comunicação externa é não-bloqueante e best-effort. O HOSA **nunca espera** a resposta de um webhook antes de prosseguir com a mitigação.

**Payload de webhook** — emitido no Nível 2+ (configurável):

```json
{
  "severity": "warning",
  "node": "worker-node-07",
  "timestamp": "2026-03-10T14:23:09.000Z",
  "hosa_level": 2,
  "d_m": 4.7,
  "d_m_derivative": 2.1,
  "d_m_acceleration": 0.5,
  "phi": 1.8,
  "rho": 0.12,
  "dominant_dimension": "mem_used",
  "dominant_contribution_pct": 68,
  "dimensional_contributions": {
    "mem_used": 0.68,
    "mem_pressure": 0.19,
    "io_latency": 0.08,
    "cpu_total": 0.05
  },
  "suspected_cgroup": "/kubepods/pod-payment-service-7b4f",
  "action_taken": "memory.high reduzido para 1.6G",
  "action_status": "efetivo",
  "d2dm_dt2": -0.45
}
```

**Endpoint Prometheus** (Fase 2): expõe o vetor de estado normalizado, `D_M` atual, nível de resposta e valores de derivada em formato texto Prometheus em `:9090/metrics`.

**Log de auditoria** — cada decisão é escrita em `/var/log/hosa/decisions.log` em JSON estruturado, independentemente de conectividade de rede. Esta é a fonte primária para análise pós-incidente.

---

## Subsistemas Transversais

### Propriocepção de Hardware — Fase de Warm-Up

Na inicialização, antes de qualquer detecção ou mitigação, o HOSA executa a sequência de **Propriocepção de Hardware**:

```
1. Descoberta de topologia
   └─ Lê /sys/devices/system/node/   → topologia NUMA
   └─ Lê /sys/devices/system/cpu/    → contagem de núcleos, tamanhos de cache
   └─ Lê /proc/meminfo               → memória total/disponível
   └─ Identifica classe de ambiente  → bare metal / VM cloud / Kubernetes / edge

2. Definição do vetor de estado
   └─ Seleciona dimensões com base na topologia
      (nós NUMA → dimensões de memória por nó para topologias complexas)
   └─ Resolve pontos de anexação das sondas eBPF

3. Acumulação basal (padrão: 5 minutos)
   └─ Coleta amostras sem mitigação
   └─ Constrói μ₀ e Σ₀ iniciais via Welford
   └─ Detecta multimodalidade do workload (checagem básica via curtose)

4. Calibração de α (EWMA)
   └─ Por recurso: α = f(variância observada do sinal durante o warm-up)
   └─ Sinais de alta variância → α menor (mais suavização)

5. Cálculo dos limiares adaptativos
   └─ θ₁ = μ_DM + 2σ_DM   (Vigilância)
   └─ θ₂ = μ_DM + 3σ_DM   (Contenção Leve)
   └─ θ₃ = μ_DM + 4σ_DM   (Contenção Ativa)
   └─ θ₄ = μ_DM + 5σ_DM   (Contenção Severa)
```

Janela de vulnerabilidade no cold start: durante o warm-up, o HOSA opera em **modo somente-monitoramento** (sem mitigação). Esta é uma limitação conhecida documentada em [Limitações](#limitações).

---

### Sistema de Resposta Graduada (Níveis 0–5)

O nível de resposta é a saída da avaliação de limiares do córtex. Transições seguem regras estritas — não é possível pular níveis (exceto no caminho de colapso rápido descrito abaixo).

```
                    ┌─── d²D̄_M/dt² > 0 (acelerando)
                    │
              D_M > θ₃ ──────────────────────────────────→ Nível 3
              D_M > θ₂ ─── dD̄_M/dt > 0 ─────────────────→ Nível 2
              D_M > θ₁ ─── ou dD̄_M/dt > 0 sustentado ──→ Nível 1
              D_M < θ₁ ─── dD̄_M/dt ≤ 0 ─────────────────→ Nível 0

              TTF < T_critico ─────────────────────────────→ Nível 4 (bypassa o 3)
              Nível 3/4 falhando + D_M ascendente ─────────→ Nível 5
```

**Histerese** previne oscilação (flapping). De-escalação do nível `N` para `N-1` requer:
- `D_M < θ_{N-1}` sustentado por `T_histerese` (padrão: 60s)
- `dD̄_M/dt < 0` (melhorando ativamente)

**Bypass de colapso rápido:** Se a estimativa de TTF indica colapso em `T_critico` segundos, o HOSA escala diretamente para o Nível 4 independente do nível atual. Isso previne cenários de escalação lenta onde o sistema colapsa antes que a resposta graduada alcance o nível necessário.

Especificação completa dos níveis:

| Nível | Nome | Gatilho | Ações | Reversibilidade |
|---|---|---|---|---|
| **0** | Homeostase | `D_M < θ₁` e `dD̄/dt ≤ 0` | Filtro Talâmico ativo. Telemetria mínima de heartbeat. Otimizações GreenOps (opcional). | — |
| **1** | Vigilância | `D_M > θ₁` ou `dD̄/dt > 0` sustentado | Amostragem: 100ms → 10ms. Entrada no log local. Nenhuma intervenção no sistema. | Auto quando condição cessa |
| **2** | Contenção Leve | `D_M > θ₂` e `dD̄/dt > 0` | `renice` nos processos contribuintes. Backpressure `memory.high` no cgroup ofensor. Webhook (async, não-bloqueante). | Auto + histerese |
| **3** | Contenção Ativa | `D_M > θ₃` e `d²D̄/dt² > 0` | Throttle `cpu.max`. XDP: descarta novos pacotes SYN. Webhook urgente. | Auto + histerese estendida |
| **4** | Contenção Severa | `D_M > θ₄` ou `TTF < T_critico` | Limite rígido `memory.max`. XDP: descarta todo tráfego de entrada exceto healthcheck. `cgroup.freeze` em cgroups não-críticos. | Requer `D_M < θ₃` sustentado |
| **5** | Quarentena Autônoma | Falha de contenção + `D_M` ascendente | Isolamento de rede (modo depende da classe de ambiente). `SIGSTOP` em processos não-críticos. Snapshot completo do estado para log persistente. Tentativa final de webhook. | **Intervenção manual requerida** |

---

### Habituação — Deriva do Perfil Basal

A habituação previne **falsos positivos crônicos** quando o workload muda permanentemente (novo deploy, crescimento orgânico, mudança de configuração).

**Condições de gatilho** (todas devem ser verdadeiras simultaneamente):

```
|dD̄_M/dt| < ε_d               // sistema estabilizou
ρ(t) < ρ_limiar                // estrutura de covariância preservada (sem deformação)
ΔH(t) < ΔH_limiar              // distribuição de syscalls inalterada
PBI(t) < PBI_limiar            // sem indicadores de propagação
D_M(t) < D_M_segurança         // não muito próximo do esgotamento de recursos
t_estável > T_min              // sustentado por pelo menos T_min (padrão: 30 min)
```

**Mecanismo:** Decaimento exponencial dos pesos das amostras no acumulador Welford. Amostras recentes têm mais peso; amostras antigas decaem. `μ` e `Σ` convergem para o novo regime operacional.

**Habituação é bloqueada para:**

| Regime | Razão |
|---|---|
| Regime −3 (Silêncio Anômalo) | Silêncio incoerente com o contexto temporal nunca é "normal" |
| Regime +3 (Adversarial) | `ρ` alto indica deformação estrutural — nunca normalizar |
| Regime +4 (Falha Local) | `D_M` crescendo monotonicamente é uma falha progressiva, não um novo normal |
| Regime +5 (Viral) | Indicadores de propagação são categoricamente nunca habituados |
| Qualquer regime próximo dos limites de recursos | Se `D_M` está acima de `D_M_segurança`, o sistema não tem margem — habituação normalizaria um estado perigoso |

---

### Filtro Talâmico — Supressão de Telemetria

No Regime 0 (homeostase), o HOSA suprime telemetria detalhada para sistemas externos. Apenas um heartbeat periódico mínimo é emitido confirmando `{"status": "alive", "level": 0, "d_m": 1.1}`.

**Por que isso importa:** Em uma frota grande, cada nó emitindo telemetria completa 24/7 gera custo significativo de ingestão mesmo quando nada está errado. O Filtro Talâmico significa que nós HOSA são *silenciosos quando saudáveis* e *barulhentos quando doentes* — exatamente a semântica que você quer.

O nível de detalhe escala com o nível de resposta:
- Nível 0: somente heartbeat (intervalo padrão: 60s)
- Nível 1: heartbeat + `D_M` + dimensão dominante (intervalo: 10s)
- Nível 2+: vetor de estado completo + contribuições dimensionais + ação tomada (intervalo: 1s ou a cada mudança de estado)

---

### Modos de Quarentena por Ambiente

O Nível 5 (Quarentena Autônoma) envolve isolamento de rede. A estratégia varia por classe de ambiente, detectada automaticamente durante a Propriocepção de Hardware.

| Ambiente | Detecção | Estratégia de Rede | Recovery |
|---|---|---|---|
| **Bare metal (IPMI)** | `/sys/class/net/` + módulos kernel `ipmi_*` | Desativa todas as interfaces *exceto* a interface de gerência IPMI/iLO/iDRAC | Manual via console IPMI |
| **VM em cloud (AWS/GCP/Azure)** | DMI/SMBIOS + metadata service `169.254.169.254` | XDP descarta todo tráfego *exceto* metadata service, DHCP e API do orquestrador. Escreve tag `hosa-quarantine=true` via metadata API. | Orquestrador termina/substitui instância. Auto-termination via API do cloud opcional (desabilitada por padrão). |
| **Kubernetes (DaemonSet)** | Namespace `/proc/1/cgroup` + env `KUBERNETES_SERVICE_HOST` | Não pode desativar interfaces do host. Aplica contenção máxima via cgroups + adiciona taint `hosa.io/quarantine=true:NoExecute` ao Node via K8s API → scheduler evacua os pods. | Operador remove o taint após investigação |
| **Edge / IoT (acesso físico)** | Flag `environment: edge-physical` | Desativação completa de interfaces de rede. Preserva logs em flash/eMMC. | Manual por técnico de campo |
| **Edge / IoT (somente remoto)** | Flag `environment: edge-remote` | Desativação de rede + hardware watchdog timer (padrão: 30 min timeout → reboot automático). Pós-reboot: modo conservador por período configurável. | Auto via reboot do watchdog + período de observação |
| **Air-gapped (SCADA/ICS)** | Flag `environment: airgap` | Idêntico a bare metal. Toda comunicação oportunista permanentemente desabilitada. Logs criptografados, coletados por acesso físico autorizado. | Manual com autorização de acesso físico |

**Princípio do padrão conservador:** Em casos ambíguos (ex: VM em cloud privada que não responde ao endpoint padrão do metadata service), o HOSA assume o **modo de quarentena mais conservador** (cloud VM — somente XDP, sem desativação de interface), priorizando recuperabilidade sobre isolamento.

---

## Fluxo de Dados — Ponta a Ponta

Um ciclo completo pelo sistema, do evento do kernel à ação de mitigação:

```
1. EVENTO DO KERNEL
   └─ ex: mm_page_alloc dispara (pico de alocação de memória)
   └─ Sonda eBPF registra: timestamp, PID, cgroup, delta bytes
   └─ Evento escrito no ring buffer

2. RING BUFFER → COLLECTOR  (~1–10 μs)
   └─ collector.go lê lote de eventos do ring buffer
   └─ Agrega no vetor de estado x(t) no intervalo de amostragem
   └─ Resolve atribuição de cgroup (PID → caminho do cgroup)
   └─ Emite StateEvent para o Sistema Límbico

3. SISTEMA LÍMBICO (~10 μs)
   └─ Adiciona StateEvent ao ring buffer de estados recentes
   └─ Fornece janela deslizante ao córtex para cálculo de derivadas

4. ATUALIZAÇÃO DE COVARIÂNCIA  (~50–100 μs)
   └─ matrix.go: atualização Welford de μ e Σ
   └─ Verifica se Σ mudou o suficiente para justificar recomputar Σ⁻¹
   └─ Se sim: inversão de Cholesky (O(n³), ~10 μs para n=10)

5. CÁLCULO DE MAHALANOBIS  (~10 μs)
   └─ D_M = sqrt( (x-μ)ᵀ Σ⁻¹ (x-μ) )
   └─ Computa c_j (contribuições dimensionais)
   └─ Computa φ (direção de carga)
   └─ Computa ρ (deformação de covariância)

6. EWMA + DERIVADAS  (~5 μs)
   └─ D̄_M(t) = α × D_M(t) + (1-α) × D̄_M(t-1)
   └─ dD̄_M/dt  ≈ (D̄_M(t) - D̄_M(t-Δt)) / Δt
   └─ d²D̄_M/dt² ≈ diferença finita de segunda ordem
   └─ Estimativa de TTF se dD̄/dt > 0

7. AVALIAÇÃO DE LIMIARES + DECISÃO DE NÍVEL  (~1 μs)
   └─ Avalia (D_M, dD̄/dt, d²D̄/dt², TTF) contra limiares
   └─ Aplica histerese (sem de-escalação abaixo de T_histerese)
   └─ Checagem da safelist (nunca atingir processos protegidos)
   └─ Determina cgroup alvo (maior contribuinte c_j)
   └─ Determina nível de resposta N

8. ATUAÇÃO  (~100–500 μs, dominado pela escrita VFS no cgroup)
   └─ Escreve nos arquivos de controle do cgroup (memory.high, cpu.max, etc.)
   └─ Atualiza BPF maps para regras XDP (se nível ≥ 3)
   └─ Escreve entrada no log de auditoria (async, não-bloqueante)
   └─ Despacha webhook (async, não-bloqueante)

TOTAL: ~200 μs – 1 ms por ciclo (dominado pelo syscall de escrita no cgroup)
```

---

## Máquina de Estados

O nível de resposta do HOSA segue esta máquina de estados:

```
         ┌──────────────────────────────────────────────────────┐
         │                    CONDIÇÕES                          │
         ├─────────────────┬────────────────────────────────────┤
         │ ESCALAÇÃO       │ DE-ESCALAÇÃO                        │
         │ imediata        │ requer histerese + D_M melhorando   │
         └─────────────────┴────────────────────────────────────┘

    ┌─────────┐
    │    0    │ ◄──────────────────────────────────────────────────┐
    │Homeost. │                                                    │
    └────┬────┘                                                    │
         │ D_M>θ₁ ou dD̄/dt>0                                      │
         ▼                                                         │
    ┌─────────┐                                                    │
    │    1    │ ◄──────────────────────────────────────────────────┤
    │Vigilânc.│                                                    │
    └────┬────┘                                                    │
         │ D_M>θ₂ e dD̄/dt>0                                       │
         ▼                                                         │
    ┌─────────┐                                                    │
    │    2    │ ◄──────────────────────────────────────────────────┤
    │ Cont.   │                                                    │
    │  Leve   │                                                    │
    └────┬────┘                                                    │
         │ D_M>θ₃ e d²D̄/dt²>0                                     │ histerese +
         ▼                                                         │ D_M melhorando
    ┌─────────┐                                                    │
    │    3    │ ◄──────────────────────────────────────────────────┤
    │  Cont.  │                                                    │
    │  Ativa  │                                                    │
    └────┬────┘                                                    │
         │ D_M>θ₄ ou TTF<T_critico                                 │
         ▼                                                         │
    ┌─────────┐                                                    │
    │    4    │ ◄──────────────────────────────────────────────────┘
    │  Cont.  │
    │  Severa │
    └────┬────┘
         │ contenção falhando + D_M ascendente
         ▼
    ┌─────────┐
    │    5    │  ──── RECOVERY MANUAL REQUERIDO ───────────────────
    │Quarent. │
    └─────────┘
```

---

## A Safelist

Certos processos e cgroups **nunca são alvo** do throttling ou sinalização do HOSA, independente de quão alta seja sua contribuição ao consumo de recursos.

**Entradas permanentes da safelist (built-in):**

- O próprio agente HOSA (PID resolvido na inicialização)
- Todas as threads do kernel (`kthreadd` e descendentes — PIDs sem parent em user space)
- Processos no cgroup do HOSA (`/sys/fs/cgroup/hosa/`)

**Entradas auto-detectadas da safelist:**

| Processo | Método de detecção |
|---|---|
| `kubelet` | Presença de `/var/run/kubelet.sock` ou env `KUBERNETES_SERVICE_HOST` |
| `containerd` | Presença de `/run/containerd/containerd.sock` |
| `dockerd` | Presença de `/var/run/docker.sock` |
| `systemd` (PID 1) | Sempre PID 1 |
| `sshd` | Processo escutando na porta 22 (preserva acesso do operador durante quarentena) |

**Entradas da safelist definidas pelo operador:**

```bash
# No arquivo hosa.yaml:
safelist:
  cgroups:
    - /kubepods/besteffort/critical-monitoring
  process_names:
    - vault-agent
    - consul
  pids:
    - 1234  # PID específico (limpo no restart)
```

**Safelist e Nível 5:** Mesmo no Nível 5, entradas da safelist não são congeladas. O objetivo da quarentena é isolar o nó da rede e congelar processos *não-críticos* — a safelist define o que "crítico" significa.

---

## Decisões de Design

| Decisão | Justificativa | Alternativa considerada |
|---|---|---|
| **Distância de Mahalanobis em vez de ML/DL** | Memória constante `O(n²)`, sem GPU, sem pipeline de treinamento, inferência sub-ms, output interpretável. Roda em um Raspberry Pi. | Autoencoders (rejeitado: requer infraestrutura de treinamento, opaco, footprint grande), Isolation Forest (rejeitado: requer janelas de dados, não incremental) |
| **Atualizações incrementais de Welford** | `O(n²)` por amostra com alocação `O(1)`. Nenhuma janela de dados armazenada. Memória previsível. | Cálculo batch de covariância (rejeitado: memória `O(n²×k)`, crescimento ilimitado) |
| **EWMA em vez de diferenciação bruta** | Diferenciação numérica é mal-posta em dados discretos e ruidosos do kernel. EWMA fornece suavização principiada com um único parâmetro ajustável. | Filtro de Kalman (mantido como alternativa futura — ótimo para ruído gaussiano, mas mais complexo de ajustar; previsto para comparação experimental) |
| **Escritas VFS diretas em cgroup em vez de libcgroup** | Elimina uma dependência de runtime. Latência determinística (único syscall write). Sem drift de versão de biblioteca. | libcgroup (rejeitado: adiciona dependência de runtime, potencial incompatibilidade de versão entre distros) |
| **Go em vez de Rust/C para user space** | Pragmático: iteração mais rápida para a fase de pesquisa. Goroutines tornam a camada de comunicação async limpa. O hot path usa padrões zero-allocation (`sync.Pool`, slices pré-alocados). Pausas do GC < 1ms no Go 1.22+. | Rust (mantido como opção futura se benchmarks do GC mostrarem impacto na janela de detecção — a arquitetura permite migração do hot path) |
| **Ring buffer em vez de perf buffer** | `BPF_MAP_TYPE_RINGBUF` é lock-free e suporta registros de tamanho variável. Overhead menor que `BPF_MAP_TYPE_PERF_EVENT_ARRAY`. | perf_event_array (rejeitado: alocação por CPU, baseado em lock, overhead maior) |
| **Complementar, não substituir o monitoramento** | Clareza arquitetural. O HOSA resolve o problema do Intervalo Letal — uma escala de tempo fundamentalmente diferente do que Prometheus/Datadog resolvem. Tentar fazer os dois comprometeria ambos. | Substituição completa de observabilidade (rejeitado: fora de escopo, exigiria armazenamento centralizado, dashboards, alerting — domínio de problema diferente) |

---

## Características de Performance

Todos os números são metas para a implementação da Fase 1. A validação experimental será documentada separadamente.

| Métrica | Meta | Notas |
|---|---|---|
| **Latência de detecção** | < 2s do início da anomalia ao Nível 1 | Com amostragem adaptativa de 10ms |
| **Latência de mitigação** | < 100ms do gatilho do Nível 2 à escrita no cgroup | Dominado pelo syscall de escrita VFS |
| **Overhead de CPU** | < 1% de um único núcleo | Durante homeostase; maior durante ciclos de contenção ativa |
| **Footprint de memória** | < 10MB RSS | Inclui histórico do vetor de estado, matriz de covariância, mapas de programas eBPF |
| **Latência do ring buffer** | 1–10 μs | Evento do kernel → consumidor em user space |
| **Latência do ciclo completo** | < 1ms | Da coleta do evento à decisão de atuação |
| **Overhead do programa eBPF** | < 0,1% de latência adicional de syscall | Overhead por sonda medido via `bpftool` |

**Budget de overhead do HOSA:** O próprio HOSA opera dentro de um cgroup dedicado com `cpu.max = 50000 100000` (50ms por janela de 100ms = máximo 50% de 1 núcleo) e `memory.max = 64M`. Se o agente exceder seu próprio budget, o kernel o contém antes de afetar outros processos. O HOSA pratica o que prega.

---

## Postura de Segurança

O HOSA requer privilégios elevados para executar sua função. As capabilities Linux mínimas necessárias:

| Capability | Necessária para | Níveis de resposta |
|---|---|---|
| `CAP_BPF` | Carregar e executar programas eBPF | Todos os níveis (coleta) |
| `CAP_PERFMON` | Anexar a tracepoints e kprobes | Todos os níveis (coleta) |
| `CAP_NET_ADMIN` | Carregar programas XDP nas interfaces | Níveis 3–5 (atuação) |
| `CAP_SYS_ADMIN` | Escrever nos arquivos de controle do cgroup v2 | Níveis 2–5 (atuação) |
| `CAP_KILL` | Enviar `SIGSTOP`/`SIGTERM` para processos | Somente Nível 5 |

**Princípio de mínimo privilégio no deploy:**
- Níveis 0–1 (somente observação): apenas `CAP_BPF` + `CAP_PERFMON`
- Níveis 0–2 (mitigação leve): adiciona `CAP_SYS_ADMIN`
- Capacidade completa (todos os níveis): adiciona `CAP_NET_ADMIN` + `CAP_KILL`

Operadores podem fazer o deploy do HOSA em modo somente-observação (equivalente aos Níveis 0–1) e gradualmente habilitar capabilities conforme ganham confiança no comportamento do agente.

**O agente não pode ser corrompido pelos workloads que monitora.** O cgroup do HOSA é separado de todos os cgroups monitorados. Seus programas eBPF são verificados pelo verificador do kernel antes de carregar — um programa malformado é rejeitado no momento do carregamento, não em runtime.

---

## Limitações

Documentadas com honestidade. Veja também [Whitepaper §9](whitepaper.pdf#section-9).

**1. Janela de cold start.** Durante o warm-up (padrão 5 minutos), o HOSA não tem baseline e não consegue tomar decisões confiáveis de detecção. O nó fica desprotegido durante essa janela. Mitigação: pré-alimentação do baseline a partir de agregados da frota (feature da Fase 2).

**2. Workloads não-estacionários.** Workloads que variam aleatoriamente em magnitude e timing — sem padrão temporal e sem estabilização — minam a premissa do perfil basal. A eficácia do HOSA é reduzida. Perfis sazonais (Seção 6.6) e habituação (Seção 5.5) endereçam a variabilidade *previsível*; variabilidade *aleatória* é uma limitação reconhecida.

**3. Evasão adversarial.** Um atacante sofisticado que entende a arquitetura do HOSA pode executar um ataque "low-and-slow" que mantém `D_M` e suas derivadas abaixo dos limiares de detecção enquanto sustenta atividade maliciosa. A detecção de deformação de covariância (`ρ`, `ΔH`) eleva significativamente a barra, mas a possibilidade teórica de evasão existe. Análise formal de resistência adversarial é pesquisa futura.

**4. Efeitos colaterais do throttling.** Backpressure de `memory.high` pode aumentar a latência do serviço contido. Throttle de CPU pode causar timeouts em cascata em serviços upstream. A safelist e o direcionamento para processos contribuintes minimizam isso, mas não eliminam.

**5. Somente Linux, kernel ≥ 5.8.** eBPF CO-RE (Compile Once — Run Everywhere) requer kernel 5.8+. Portabilidade para outros sistemas operacionais não está planejada.

**6. Impacto das pausas do GC (Go).** O garbage collector do Go, apesar de sub-milissegundo no 1.22+, é não-determinístico. Se benchmarks do hot path revelarem pausas do GC impactando a latência de detecção sob pressão adversarial de alocação, a migração do hot path para uma linguagem zero-GC (Rust ou C via CGo) está planejada.

---

*Para a formulação matemática completa, veja [`docs/math_model.md`](math_model.md).*
*Para a fundamentação teórica e contexto acadêmico, veja o [Whitepaper v2.1](whitepaper.pdf).*
