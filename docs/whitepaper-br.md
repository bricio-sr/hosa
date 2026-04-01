# HOSA — Homeostasis Operating System Agent

## Whitepaper & Manifesto Arquitetural

**Autor:** Fabricio Roney de Amorim
**Versão do Documento:** 2.2 — Expansão Arquitetural
**Data de Criação:** 9 de março de 2026
**Data desta Revisão:** 30 de março de 2026
**Contexto Acadêmico:** Fundamentação de intenção para dissertação de Mestrado — Unicamp (IMECC)
**Status:** Documento de Visão e Fundamentação Teórica

**Registro de Integridade:**
- Repositório de referência: https://github.com/bricio-sr/hosa

**Histórico de Versões:**
| Versão | Data | Descrição |
|---|---|---|
| 1.0 | 09/03/2026 | Primeira versão do whitepaper. Conceito inicial. |
| 2.0 | 10/03/2026 | Revisão crítica: taxonomia bipolar, métricas suplementares, resposta graduada expandida. |
| 2.1 | 10/03/2026 | Blindagem de objeções: robustez sob não-normalidade, calibração de ICP, quarentena por ambiente, walkthrough narrativo, FAQ. |
| 2.2 | 30/03/2026 | Expansão arquitetural: inserção da Fase 2 (Sistema Nervoso Simpático — escalonamento físico e termodinâmica de memória via `sched_ext` e eBPF); renumeração das fases 2–7 para 3–9; inserção da Fase 8 (Kernel Causal — inferência causal, do-calculus e DAG em Ring 0); formalização metodológica deslocada para Fase 9; adição de DTrace/linhagem eBPF nos Trabalhos Relacionados. |

---

## Resumo

Este documento apresenta o HOSA (Homeostasis Operating System Agent), uma arquitetura de software bio-inspirada para resiliência autônoma de sistemas operacionais Linux. O HOSA propõe a substituição do modelo dominante de telemetria exógena — no qual a detecção de anomalias e a mitigação de falhas dependem de servidores centrais externos — por um modelo de **Resiliência Endógena**, no qual cada nó computacional possui capacidade autônoma de detecção multivariável e mitigação local em tempo real, independentemente de conectividade de rede.

A detecção de anomalias é realizada através de análise estatística multivariável baseada na Distância de Mahalanobis e sua taxa de variação temporal, com coleta de sinais via eBPF (Extended Berkeley Packet Filter) no Kernel Space do Linux. A mitigação é executada em camadas progressivas: na Fase 1, através de manipulação determinística de Cgroups v2 e XDP (eXpress Data Path); na Fase 2, através de intervenção direta no escalonador de processos (via `sched_ext`) e no subsistema de memória virtual, substituindo os algoritmos de propósito geral do kernel por políticas de sobrevivência determinística durante o Intervalo Letal. As fases subsequentes estendem o sistema com capacidades semânticas, de enxame e de raciocínio causal.

O HOSA não substitui orquestradores ou sistemas de monitoramento global. Ele os complementa ao operar no intervalo temporal em que esses sistemas são estruturalmente incapazes de atuar: os milissegundos entre o início de um colapso e a chegada da primeira métrica ao control plane externo.

**Palavras-chave:** Resiliência Endógena, Computação Autonômica, eBPF, sched_ext, Detecção de Anomalias Multivariável, Distância de Mahalanobis, Inferência Causal, do-calculus, Sistemas Bio-Inspirados, Edge Computing, SRE.

---

## 1. Introdução e Declaração do Problema

### 1.1. O Modelo Dominante e Suas Limitações Estruturais

A engenharia de confiabilidade de sistemas computacionais (Site Reliability Engineering — SRE) consolidou-se na última década ao redor de um paradigma que este trabalho denomina **Telemetria Exógena**: um modelo no qual agentes locais coletam métricas, transmitem-nas via rede a servidores centrais de análise, e aguardam instruções de mitigação derivadas dessa análise remota.

Este paradigma, sustentado por ferramentas amplamente adotadas como Prometheus (Prometheus Authors, 2012), Datadog, Grafana e orquestradores como Kubernetes (Burns et al., 2016), opera sob pressupostos que se tornam progressivamente frágeis à medida que a infraestrutura computacional se expande para cenários de Internet das Coisas (IoT), Edge Computing, telecomunicações e sistemas embarcados industriais.

A fragilidade estrutural do modelo exógeno manifesta-se em duas dimensões:

**A Latência de Consciência.** O ciclo operacional do monitoramento exógeno segue uma sequência discreta: coleta periódica (polling/pulling com intervalos típicos de 10 a 60 segundos), transmissão via rede, armazenamento em banco de séries temporais (TSDB), avaliação contra limiares estáticos (e.g., "CPU > 90% por 1 minuto"), e disparo de alerta. Cada etapa introduz latência acumulativa. O sistema central toma decisões com base em um retrato estatisticamente defasado do nó remoto. Em cenários de colapso rápido — ataques de negação de serviço, memory leaks agressivos, picos instantâneos de carga —, a mitigação chega tarde.

**A Fragilidade de Conexão.** O modelo exógeno assume conectividade contínua e confiável entre o nó monitorado e o control plane. Esta premissa é violada rotineiramente em cenários de Edge Computing (dispositivos em campo com conectividade intermitente), durante ataques DDoS que saturam a banda de saída do próprio nó monitorado, e em infraestruturas industriais com redes segmentadas por requisitos de segurança. Quando a rede falha, o nó perde simultaneamente sua capacidade de reportar e de receber instruções de mitigação, operando em completa cegueira operacional.

### 1.2. A Física do Colapso: O Intervalo Letal

O colapso de um nó computacional não é um processo gradual e linear; é uma cascata exponencial. Quando a memória física se esgota, o Kernel Linux aciona o OOM-Killer (Out-Of-Memory Killer), encerrando processos abruptamente com base em heurísticas de pontuação, corrompendo transações em andamento e gerando indisponibilidade imediata. O mecanismo `systemd-oomd` (Poettering, 2020) e o subsistema PSI (Pressure Stall Information) do kernel (Weiner, 2018) representam tentativas do próprio ecossistema Linux de endereçar esta lacuna, mas operam com escopo limitado: PSI fornece métricas de pressão sem capacidade de mitigação autônoma, e `systemd-oomd` atua com políticas estáticas que não consideram correlação multivariável entre recursos.

O intervalo temporal entre o início do estresse letal e a chegada da primeira métrica utilizável ao sistema de monitoramento externo constitui o que este trabalho denomina **Intervalo Letal** — a janela onde sistemas morrem sem que o observador externo tenha sequer consciência do problema.

#### Figura 1 — Visualização Temporal do Intervalo Letal: HOSA vs. Modelo Exógeno

```
LINHA DO TEMPO DE UM COLAPSO — MEMORY LEAK A 50MB/s

Tempo    │ Estado do Nó           │ HOSA (Endógeno)        │ Prometheus+Alertmanager
(seg)    │                        │                        │ (Exógeno)
─────────┼────────────────────────┼────────────────────────┼─────────────────────────
         │                        │                        │
  t=0    │ Leak inicia            │ D_M=1.1 (homeostase)   │ Último scrape há 8s
         │ mem: 61%               │ Nível 0                │ Dados mostram: "saudável"
         │                        │                        │
  t=1    │ mem: 64%               │ ⚡ D_M=2.8 DETECTA      │
         │ PSI: 18%               │ Nível 0→1 (Vigilância) │ (sem scrape)
         │                        │ Amostragem: 100ms→10ms │
         │                        │                        │
  t=2    │ mem: 68%               │ ⚡ D_M=4.7 CONTÉM       │
         │ PSI: 29%               │ Nível 1→2 (Contenção)  │ (sem scrape)
         │ swap ativando          │ memory.high → 1.6G     │
         │                        │ Webhook disparado      │
         │                        │                        │
  t=4    │ mem: 72%               │ dD̄/dt desacelerando    │ Scrape! Coleta mem=1.47G
         │ (contido pelo HOSA)    │ Contenção eficaz       │ Regra: >1.8G for 1m
         │                        │ Mantém Nível 2         │ Resultado: OK (!)
         │                        │                        │
  t=8    │ mem: 74%               │ ✓ ESTABILIZADO         │
         │ (platô de contenção)   │ Sistema degradado      │ (sem scrape)
         │                        │ mas funcional          │
         │                        │                        │
  t=15   │ mem: 74%               │ ✓ Mantém contenção     │ Scrape. mem=1.52G
         │                        │                        │ Resultado: OK (!)
         │                        │                        │
  t=30   │ mem: 75%               │ ✓ Mantém contenção     │ Scrape. mem=1.55G
         │                        │ Operador recebeu       │ Resultado: OK (!)
         │                        │ webhook, investiga     │
         │                        │                        │
─────────┼────────────────────────┼────────────────────────┼─────────────────────────
         │                        │                        │
         │     CENÁRIO            │     COM HOSA           │     SEM HOSA
         │  CONTRAFACTUAL         │                        │
         │  (sem HOSA)            │                        │
─────────┼────────────────────────┼────────────────────────┼─────────────────────────
         │                        │                        │
  t=40   │ ☠ OOM-Kill             │ ✓ Sistema contido      │ Scrape. Detecta restart.
         │ payment-service morto  │ Transações preservadas │ Ainda sem alerta
         │ Transações corrompidas │                        │ (for 1m não satisfeito)
         │                        │                        │
  t=80   │ ☠ 2° crash             │ ✓ Sistema contido      │ Scrape. CrashLoopBackOff
         │ CrashLoopBackOff       │ Operador fazendo       │ detectado.
         │                        │ rollback               │
         │                        │                        │
  t=100  │ ☠ Clientes com 502     │ ✓ Rollback concluído   │ ⚠ ALERTA DISPARADO
         │ desde t=40             │ Sistema recuperado     │ (60s após 1° crash)
         │                        │                        │
─────────┴────────────────────────┴────────────────────────┴─────────────────────────

         ├──── 2s ────┤
         │HOSA atuou  │
         │   aqui     │
         │            │
         │            ├─────────────────────────── 98s ───────────────────────────┤
         │            │           Prometheus atuou aqui                           │
         │            │           (100x mais lento)                               │
```

### 1.3. A Tese Central

A tese que fundamenta o HOSA pode ser enunciada como:

> *Orquestradores e sistemas de monitoramento centralizados são instrumentos essenciais para planejamento de capacidade, balanceamento de carga e governança de infraestrutura a longo prazo. Contudo, eles são estruturalmente — e não acidentalmente — lentos demais para garantir a sobrevivência de um nó em tempo real. Se o colapso ocorre no intervalo entre a percepção e a ação exógena, a capacidade de decisão imediata deve residir no próprio nó.*

O HOSA não propõe a eliminação do monitoramento central. Propõe a **complementação** desse monitoramento com uma camada de inteligência local que opera de forma autônoma durante o Intervalo Letal, estabilizando o nó até que o sistema global possa assumir o controle da situação.

---

## 2. Gênese Conceitual: A Metáfora Biológica como Ferramenta de Design

### 2.1. O Arco Reflexo como Padrão Arquitetural

A arquitetura do HOSA foi concebida a partir da observação de um padrão biológico fundamental: o **arco reflexo medular**.

Quando um organismo humano toca uma superfície em temperatura nociva, o sinal nociceptivo não percorre o trajeto completo até o córtex cerebral (o "orquestrador central") para processamento contextual e deliberação consciente. A latência dessa via longa — centenas de milissegundos — resultaria em lesão tecidual. Em vez disso, o sinal percorre um arco curto até a medula espinhal, que executa uma contração muscular reflexa em sub-milissegundos, retirando o membro da fonte de dano. Apenas após a execução do reflexo, o córtex é notificado para processamento contextual e formação de memória (Bear, Connors & Paradiso, 2015).

Este padrão — **ação local imediata seguida de notificação contextual ao centro de comando** — é precisamente o modelo operacional do HOSA.

A Fase 2 do HOSA aprofunda esta metáfora: assim como o **sistema nervoso simpático** (distinto do reflexo medular) desencadeia respostas fisiológicas mais profundas sob stress agudo — acelerando a frequência cardíaca, redistribuindo o fluxo sanguíneo para os músculos vitais, contraindo vasos periféricos não-essenciais —, o HOSA Fase 2 não apenas limita logicamente os recursos, mas **redistribui fisicamente o tempo de processador e a topologia de memória** em favor dos processos de sobrevivência, alterando as próprias regras do kernel durante o Intervalo Letal.

É importante delimitar o escopo desta metáfora: ela é utilizada como **ferramenta heurística de design arquitetural**, não como reivindicação de equivalência funcional entre sistemas biológicos e computacionais. A biologia informa a estrutura de decisão (onde processar, onde atuar, quando escalar), mas a implementação é puramente matemática e de engenharia de sistemas.

### 2.2. Precedentes na Literatura: Computação Autonômica e Imunologia Computacional

A aspiração por sistemas computacionais auto-reguláveis não é inédita. O manifesto de Computação Autonômica da IBM (Horn, 2001) articulou quatro propriedades desejáveis — auto-configuração, auto-otimização, auto-cura e auto-proteção — mas permaneceu predominantemente no nível de visão estratégica, sem fornecer a instrumentação de baixo nível para viabilizá-las com latência sub-milissegundo.

O trabalho de Forrest, Hofmeyr & Somayaji (1997) sobre imunologia computacional estabeleceu os fundamentos teóricos da distinção entre "self" e "non-self" em sistemas computacionais, propondo que processos anômalos podem ser identificados por desvios em sequências de chamadas de sistema (syscalls). O HOSA absorve este princípio na sua camada de triagem comportamental.

O que diferencia o HOSA destes precedentes é a **síntese operacional**: a combinação de detecção multivariável contínua (não baseada em assinaturas) com atuação em kernel space via mecanismos contemporâneos (eBPF, `sched_ext`, Cgroups v2, XDP) que não existiam quando esses trabalhos foram publicados. O HOSA é, neste sentido, a resposta de engenharia contemporânea a uma necessidade que a literatura identificou há duas décadas.

---

## 3. Trabalhos Relacionados e Posicionamento

Uma contribuição acadêmica responsável exige o confronto explícito com os trabalhos e tecnologias que operam no mesmo espaço de problema. Esta seção mapeia o ecossistema existente e articula a lacuna específica que o HOSA preenche.

### 3.1. Mecanismos Nativos do Kernel Linux

| Mecanismo | Função | Limitação que o HOSA Endereça |
|---|---|---|
| **PSI (Pressure Stall Information)** — Weiner, 2018 | Expõe métricas de pressão de CPU, memória e I/O como porcentagem de tempo em stall. | PSI é um **sensor passivo**: ele quantifica a pressão, mas não executa mitigação. Adicionalmente, PSI é uma métrica unidimensional por recurso — ele não correlaciona CPU, memória, I/O e rede simultaneamente. O HOSA utiliza PSI como uma das entradas do seu vetor de estado multivariável, mas complementa-o com análise de covariância cruzada e taxa de variação. |
| **systemd-oomd** — Poettering, 2020 | Daemon que monitora PSI de memória e mata cgroups inteiros quando pressão excede limiar. | Opera com **limiares estáticos unidimensionais** (pressão de memória apenas). Não considera correlação com outros recursos. Não oferece respostas graduadas — a ação é binária: nada ou kill. |
| **OOM-Killer** | Mecanismo de última instância do kernel para liberar memória. | **Reativo e destrutivo**: ativa-se apenas após o esgotamento total da memória, e usa heurísticas simplificadas (oom_score) que frequentemente eliminam processos críticos. |
| **cgroups v2** — Heo, 2015 | Interface de controle de recursos por grupo de processos. | É um **mecanismo atuador** sem inteligência de decisão associada. Requer que algo externo decida quais limites aplicar e quando. O HOSA utiliza cgroups v2 como seu sistema motor na Fase 1. |
| **sched_ext** — Torvalds et al., 2024 | Framework para substituição do escalonador de processos via programas eBPF carregáveis dinamicamente. | É um **mecanismo de extensão** sem política embutida. Requer que algo defina os critérios de escalonamento. O HOSA utiliza `sched_ext` como o motor da Fase 2 para implementar o Escalonador de Sobrevivência. |
| **Buddy Allocator / Compaction** | Alocador de páginas físicas do kernel; compactação move páginas para desfragmentar. | A compactação ocorre **reativamente** sob pressão, causando Compaction Stalls — paralisações de userspace invisíveis às métricas tradicionais. O HOSA antecipa a fragmentação via análise de entropia e realiza desfragmentação preemptiva. |

### 3.2. Observabilidade Dinâmica Programável: DTrace e a Linhagem do eBPF

A trajetória que culmina no eBPF — o substrato tecnológico central do HOSA — possui uma genealogia intelectual que não pode ser omitida de uma revisão honesta da literatura. O ponto de origem desta linhagem é o **DTrace**, desenvolvido por Bryan Cantrill, Mike Shapiro e Adam Leventhal na Sun Microsystems e introduzido no Solaris 10 em 2004 (Cantrill et al., 2004).

DTrace estabeleceu um conjunto de princípios que definiram o campo da observabilidade dinâmica em produção:

**O princípio do custo zero em desuso.** DTrace introduziu a noção de que probes (sondas) instrumentadas no código do sistema operativo têm custo de CPU estritamente nulo quando não estão habilitadas — o código de instrumentação é substituído por instruções NOP em tempo de compilação. Apenas quando uma sonda é ativada dinamicamente o overhead materializa-se. Este princípio, aparentemente óbvio em retrospecto, representou uma ruptura com o modelo anterior de instrumentação por compilação condicional ou por intercepção via ptrace — ambos com custo permanente e incompatíveis com uso em produção sem degradação de performance.

**O princípio da segurança do kernel.** O verificador (verifier) central do DTrace garante que programas D (a linguagem de sondas do DTrace) não podem comprometer a estabilidade do sistema: eles são verificados formalmente antes de execução para ausência de loops infinitos, acesso a memória inválida e efeitos colaterais destrutivos. O verificador eBPF do Linux herda e formaliza este mesmo princípio, aplicando análise de fluxo de controle e verificação de tipos ao código BPF bytecode. É este princípio que torna viável a execução de código dinâmico em Ring 0 — a garantia de que um bug no programa eBPF resulta em rejeição pelo verificador, não em kernel panic.

**A linguagem de sondas como interface de programação.** DTrace introduziu a ideia de que a instrumentação do sistema deve ser programmable em tempo de execução por uma linguagem de alto nível (D, no caso do DTrace), sem recompilação do kernel ou reinicialização do sistema. Esta ideia — observabilidade como código, não como configuração — é o fundamento conceitual das sondas eBPF e dos tracepoints que o HOSA emprega.

**A convergência com Linux.** A influência do DTrace no ecossistema Linux manifesta-se em múltiplas camadas. O **SystemTap** (Red Hat, 2005) foi a primeira tentativa de trazer observabilidade dinâmica programável ao Linux, utilizando um compilador que traduz scripts para módulos de kernel — mas com custo de desenvolvimento elevado e riscos de estabilidade. O subsistema **perf** e os **uprobes/kprobes** generalizaram a instrumentação de pontos arbitrários do kernel e de aplicações em espaço de usuário. O projeto **DTrace for Linux** (Oracle) tentou um port direto. Todas estas iniciativas convergem, na prática moderna, para o eBPF: a plataforma que realizou no Linux o que DTrace realizou no Solaris, com a vantagem de ser nativa ao kernel e de ter alcançado adoção universal a partir do kernel 4.x.

A relevância desta genealogia para o HOSA não é meramente histórica. O princípio de segurança do verificador herdado do DTrace é a garantia fundamental que torna viável a Fase 2 do HOSA: instalar um escalonador de sobrevivência via `sched_ext` em Ring 0 e instrumentar os tracepoints do subsistema de memória virtual sem risco de corrupção do sistema. Adicionalmente, a linhagem DTrace → eBPF estabelece o padrão arquitetural de **separação entre política e mecanismo** que o HOSA explora sistematicamente: o kernel provê os mecanismos (`sched_ext`, eBPF maps, ring buffers, XDP hooks), e o HOSA define as políticas (Escalonador de Sobrevivência, desfragmentação preemptiva, do-calculus causal). Esta separação é a razão pela qual o HOSA pode substituir o escalonador padrão durante o Intervalo Letal sem modificar o código-fonte do kernel.

### 3.3. Ferramentas do Ecossistema de Observabilidade

| Ferramenta/Projeto | Função | Diferenciação do HOSA |
|---|---|---|
| **Prometheus + Alertmanager** | Coleta de métricas via pull, armazenamento em TSDB, alertas baseados em regras. | Modelo exógeno clássico. Intervalo de scrape padrão: 15–60s. Latência mínima de alerta: tipicamente >1 minuto. Sem capacidade de atuação. |
| **Sysdig Falco** — Sysdig, 2016 | Detecção de comportamento anômalo em runtime via eBPF, focado em segurança. | Falco detecta violações de política de segurança (syscalls suspeitas), mas **não monitora saúde de recursos** (CPU, memória, I/O) e **não executa mitigação autônoma**. Seu foco é alertar, não atuar. |
| **Cilium Tetragon** — Isovalent, 2022 | Enforcement de políticas de segurança em kernel space via eBPF. | Tetragon permite definir políticas de bloqueio (e.g., "bloquear processo que abrir /etc/shadow"), mas opera sobre **regras estáticas definidas pelo operador**. Não possui modelo estatístico de anomalia, não calcula derivadas de estado, e não implementa respostas graduadas baseadas em severidade. |
| **Pixie (px.dev)** — New Relic | Observabilidade contínua via eBPF sem instrumentação de código. | Pixie é um sistema de **coleta e visualização** — não possui camada de atuação autônoma. |
| **BCC / bpftrace** — Gregg, 2019 | Ferramentas de análise de performance e depuração via eBPF para uso interativo. | Ferramentas de diagnóstico interativo para operadores humanos. Não possuem capacidade de decisão ou mitigação autônoma. São a realização prática da herança DTrace no ecossistema Linux, mas sem a camada de agência que o HOSA adiciona. |
| **Facebook FBAR** — Tang et al., 2020 | Remediação automática em escala nos datacenters do Meta. | FBAR opera como **sistema centralizado de remediação** com dependência de rede e infraestrutura proprietária. Não é um agente local autônomo. |

### 3.4. A Lacuna Identificada

Nenhuma ferramenta existente no ecossistema combina, em um único agente local:

1. **Detecção multivariável contínua** (correlação entre CPU, memória, I/O, rede e latência de disco em espaço estatístico unificado);
2. **Análise de taxa de variação** (derivada temporal do vetor de estado, detectando aceleração em direção ao colapso e não apenas estado presente);
3. **Atuação local autônoma graduada** (desde throttling seletivo até isolamento de rede, sem dependência de rede ou intervenção humana);
4. **Controle físico do escalonador** (substituição da política de justiça do CFS por política de sobrevivência determinística via `sched_ext`);
5. **Controle termodinâmico da memória** (desfragmentação preemptiva baseada em análise de entropia de topologia de páginas físicas);
6. **Raciocínio causal pré-ação** (construção de DAG dinâmico de IPC e avaliação contrafactual via do-calculus antes de executar intervenções);
7. **Independência total de infraestrutura externa** para sua função primária de sobrevivência.

O HOSA posiciona-se nesta intersecção.


---

## 4. Fundamentação Matemática

### 4.1. Representação do Estado do Sistema

O HOSA modela o estado instantâneo de um nó como um vetor $\vec{x}(t) \in \mathbb{R}^n$, onde cada componente representa uma variável de recurso do sistema:

$$\vec{x}(t) = \begin{bmatrix} x_1(t) \\ x_2(t) \\ \vdots \\ x_n(t) \end{bmatrix}$$

Na implementação de referência, as variáveis incluem (mas não se limitam a):
- Utilização de CPU (agregada e por núcleo)
- Pressão de memória (utilização, swap, PSI)
- Throughput e latência de I/O de disco
- Taxa de pacotes de rede (entrada/saída)
- Profundidade de filas de scheduler (run queue depth)
- Taxa de page faults
- Contadores de context switches
- Entropia de fragmentação de memória física $H_{frag}$ (introduzida na Fase 2 — Seção 9.2)

### 4.2. A Distância de Mahalanobis como Detector de Anomalia

A detecção de anomalia baseada em limiar unidimensional estático (e.g., "CPU > 90%") sofre de uma limitação fundamental: ela ignora a **estrutura de correlação** entre as variáveis. CPU alta com I/O baixo e rede estável pode representar processamento intensivo legítimo. CPU alta com memory pressure crescente, I/O em stall e latência de rede subindo representa colapso iminente. O limiar estático não distingue esses cenários.

A Distância de Mahalanobis (Mahalanobis, 1936) endereça esta limitação ao medir a distância de uma observação $\vec{x}$ em relação à distribuição multivariável definida pelo vetor de médias $\vec{\mu}$ e pela Matriz de Covariância $\Sigma$:

$$D_M(\vec{x}) = \sqrt{(\vec{x} - \vec{\mu})^T \Sigma^{-1} (\vec{x} - \vec{\mu})}$$

A Matriz de Covariância $\Sigma$ captura as correlações entre todas as variáveis. A sua inversa $\Sigma^{-1}$ pondera as dimensões de acordo com sua variância e interdependência. Um vetor $\vec{x}(t)$ que se afasta do perfil basal em dimensões correlacionadas de maneira não-usual produz um $D_M$ elevado, mesmo que nenhuma variável individual tenha excedido um limiar absoluto.

Para uma revisão abrangente de métodos de detecção de outliers, ver Aggarwal (2017).

### 4.3. A Derivada Temporal e o Problema da Estabilidade Numérica

O HOSA não atua sobre o valor instantâneo de $D_M$, mas sobre a sua **taxa de variação temporal** — a velocidade e a aceleração com que o sistema se afasta da homeostase.

A primeira derivada $\frac{dD_M}{dt}$ indica a velocidade de afastamento. A segunda derivada $\frac{d^2D_M}{dt^2}$ indica a aceleração — se o sistema está acelerando em direção ao colapso ou desacelerando.

**Problema reconhecido: instabilidade da diferenciação numérica em dados discretos e ruidosos.** A diferenciação numérica é um problema mal-posto (ill-posed) no sentido de Hadamard: pequenas perturbações nos dados de entrada produzem grandes variações na derivada calculada. A segunda derivada amplifica este efeito quadraticamente. Sem tratamento, a segunda derivada de séries temporais ruidosas de kernel oscila violentamente, gerando falsos positivos.

**Solução adotada:** O HOSA implementa uma **Média Móvel Exponencialmente Ponderada (EWMA)** com fator de decaimento $\alpha$ calibrado por recurso antes do cálculo da derivada:

$$\bar{D}_M(t) = \alpha \cdot D_M(t) + (1 - \alpha) \cdot \bar{D}_M(t-1)$$

O fator $\alpha$ controla o trade-off fundamental entre **responsividade** (valores altos de $\alpha$ preservam variações rápidas, mas mantêm ruído) e **estabilidade** (valores baixos de $\alpha$ suavizam o sinal, mas introduzem latência de detecção).

A calibração de $\alpha$ é realizada durante a fase de **warm-up** do agente (Seção 5.3), e constitui um dos parâmetros críticos da arquitetura. A documentação técnica apresentará a análise de sensibilidade de $\alpha$ contra datasets sintéticos e reais de colapso, quantificando o trade-off latência vs. taxa de falsos positivos.

**Alternativa sob investigação:** O Filtro de Kalman unidimensional oferece estimativa ótima do estado em presença de ruído gaussiano, com a vantagem de adaptar-se dinamicamente à variância observada. A análise comparativa EWMA vs. Kalman será apresentada na fase experimental da dissertação.

### 4.4. Atualização Incremental da Matriz de Covariância

O cálculo batch da Matriz de Covariância ($\Sigma$) sobre janelas de dados acumulados é computacionalmente custoso ($O(n^2 \cdot k)$ para $n$ variáveis e $k$ amostras) e introduz alocação de memória proporcional ao tamanho da janela.

O HOSA utiliza o **algoritmo de Welford generalizado** (Welford, 1962) para atualização incremental online de $\Sigma$ e $\vec{\mu}$. Cada nova amostra $\vec{x}(t)$ atualiza $\Sigma$ em $O(n^2)$ com alocação constante ($O(1)$), independentemente do número de amostras acumuladas. Isso elimina a necessidade de armazenar janelas de dados e garante footprint de memória previsível.

### 4.5. Inversão da Matriz de Covariância

A Distância de Mahalanobis requer $\Sigma^{-1}$. Para dimensionalidade moderada ($n \leq 10$), a inversão direta via decomposição de Cholesky é computacionalmente viável e numericamente estável (a Matriz de Covariância é positiva semidefinida por construção). Para dimensionalidade maior, o HOSA pode recorrer à atualização incremental da inversa via fórmula de Sherman-Morrison-Woodbury, evitando recalcular a inversão completa a cada amostra.

**Degenerescência:** Em sistemas com variáveis altamente colineares (e.g., `cpu_user` e `cpu_total`), $\Sigma$ pode tornar-se singular ou mal-condicionada. O HOSA aplica **regularização de Tikhonov** ($\Sigma_{reg} = \Sigma + \lambda I$, com $\lambda$ pequeno) para garantir invertibilidade.

### 4.6. Robustez da Distância de Mahalanobis sob Violações de Normalidade

A Distância de Mahalanobis, conforme formulada na Seção 4.2, assume implicitamente que o perfil basal do sistema segue uma distribuição aproximadamente elipsoidal (multivariável normal). Esta suposição merece análise explícita, pois métricas de kernel em sistemas reais frequentemente exibem características que a violam.

#### 4.6.1. Natureza das Violações Esperadas

Três classes de violação são empiricamente prevalentes em métricas de sistemas operacionais:

| Classe de Violação | Exemplo em Métricas de Kernel | Impacto na $D_M$ |
|---|---|---|
| **Caudas pesadas (heavy-tailed)** | Latência de I/O de disco: a maioria das operações completa em microsegundos, mas outliers de centenas de milissegundos ocorrem com frequência maior que a prevista pela distribuição normal. | $D_M$ subestima a frequência de valores extremos legítimos, potencialmente gerando falsos positivos em eventos de cauda. |
| **Assimetria (skewness)** | Utilização de CPU: distribuição frequentemente concentrada próximo de 0% (sistema ocioso) ou próximo de 100% (sistema saturado), com assimetria dependente do regime operacional. | $\vec{\mu}$ e $\Sigma$ podem não representar adequadamente o centro e a dispersão da distribuição real, deslocando o detector. |
| **Multimodalidade** | Sistemas que alternam entre dois regimes operacionais distintos (e.g., servidor batch que processa jobs a cada hora). | $\vec{\mu}$ calculado como média aritmética localiza-se **entre** os dois modos, onde poucas amostras reais existem. $D_M$ classifica o comportamento normal de ambos os modos como anômalo. |

#### 4.6.2. Evidência de Robustez na Literatura

A robustez da Distância de Mahalanobis como detector de outliers sob violações moderadas de normalidade é documentada na literatura:

- **Gnanadesikan & Kettenring (1972)** demonstraram que estimadores baseados em covariância mantêm capacidade discriminativa sob distribuições elípticas não-normais, perdendo a interpretação probabilística exata mas preservando a **ordenação relativa** de anomalias.
- **Penny (1996)** analisou a performance de $D_M$ como critério de classificação sob diversas distribuições não-gaussianas, confirmando degradação graciosa.
- **Hubert, Debruyne & Rousseeuw (2018)** demonstraram que a combinação de $D_M$ com estimadores robustos de localização e dispersão preserva a eficácia de detecção mesmo sob contaminação de até 25% das amostras por outliers.

O HOSA opera primariamente sobre a **taxa de variação** de $D_M$ (derivadas), não sobre seu valor absoluto. Mesmo que o valor absoluto de $D_M$ perca a interpretação probabilística exata sob não-normalidade, as derivadas $\frac{dD_M}{dt}$ e $\frac{d^2D_M}{dt^2}$ permanecem indicadores válidos de **aceleração em direção ao colapso**, pois refletem a dinâmica temporal do desvio, não sua magnitude probabilística.

#### 4.6.3. Estratégia de Mitigação: Estimação Robusta

Para endereçar violações severas quando detectadas, o HOSA implementa uma estratégia de dois níveis:

**Nível 1 — Regularização (padrão).** A regularização de Tikhonov já aplicada ($\Sigma_{reg} = \Sigma + \lambda I$) mitiga parcialmente a sensibilidade a outliers ao estabilizar a inversão da matriz de covariância, funcionando como uma forma de shrinkage que aproxima $\Sigma^{-1}$ da identidade.

**Nível 2 — Estimação robusta (ativação condicional).** Quando o HOSA detecta que a distribuição basal viola severamente a normalidade — operacionalizado via monitoramento contínuo da curtose multivariada de Mardia (Mardia, 1970):

$$\kappa_M = \frac{1}{N} \sum_{i=1}^{N} \left[(\vec{x}_i - \vec{\mu})^T \Sigma^{-1} (\vec{x}_i - \vec{\mu})\right]^2$$

comparada com o valor esperado sob normalidade $\kappa_{esperado} = n(n+2)$ (onde $n$ é a dimensionalidade) — o agente pode substituir os estimadores de $\vec{\mu}$ e $\Sigma$ pelo **Minimum Covariance Determinant (MCD)** (Rousseeuw, 1984).

O MCD estima localização e dispersão utilizando o subconjunto de $h$ observações (de $N$ totais, com $h \approx \lceil N/2 \rceil$) cuja matriz de covariância tem o menor determinante, efetivamente descartando a influência dos $N-h$ outliers mais extremos na estimação dos parâmetros basais. A implementação incremental do MCD via algoritmo FAST-MCD (Rousseeuw & Van Driessen, 1999) é computacionalmente viável para a dimensionalidade esperada do vetor de estado ($n \leq 15$).

**Impacto no footprint:** O MCD incremental requer armazenamento de uma janela de amostras recentes (tipicamente 100–500 amostras) para recalcular o subconjunto ótimo, violando parcialmente o princípio de memória $O(1)$ do Welford. O trade-off é explícito: robustez estatística contra footprint previsível. A ativação do MCD é **condicional** — só ocorre quando $\kappa_M$ diverge significativamente de $\kappa_{esperado}$. Na prática, a janela de 500 amostras com $n = 10$ variáveis ocupa $\sim$40KB de memória — negligível em qualquer contexto operacional.

#### 4.6.4. Multimodalidade e Interação com Perfis Sazonais

O problema de multimodalidade — a violação mais severa para $D_M$ — é parcialmente endereçado pelo mecanismo de perfis basais indexados por contexto temporal (Seção 6.6). Quando a multimodalidade é causada por alternância temporal previsível entre regimes, cada segmento temporal acumula seu próprio perfil basal **unimodal**, eliminando a multimodalidade na raiz.

Quando a multimodalidade não é temporalmente segregável, a abordagem requer extensão para **Mixture of Gaussians** com estimação via Expectation-Maximization (EM) adaptado para streaming (Engel & Heinen, 2010). Esta extensão é documentada como direção de pesquisa futura, pois introduz complexidade computacional ($O(k \cdot n^2)$ por amostra, onde $k$ é o número de modos) e o problema de seleção de modelo (determinação de $k$).

#### 4.6.5. Plano de Validação Empírica

A validação experimental incluirá:

1. **Coleta de dados reais** de métricas de kernel em sistemas de produção (mínimo 72 horas contínuas por cenário);
2. **Testes de normalidade multivariada**: curtose de Mardia, teste de Henze-Zirkler (Henze & Zirkler, 1990), e inspeção visual via QQ-plots multivariados;
3. **Benchmarking comparativo** da taxa de detecção (True Positive Rate) e taxa de falsos positivos (False Positive Rate) sob estimação clássica (Welford), estimação robusta (MCD), e Mahalanobis com transformação prévia (e.g., Box-Cox multivariada);
4. **Análise de impacto no footprint computacional** de cada alternativa.

---

## 5. Arquitetura de Engenharia

### 5.1. Princípios Arquiteturais

O design do HOSA é governado por cinco princípios não-negociáveis:

| # | Princípio | Descrição |
|---|---|---|
| 1 | **Autonomia Local** | O HOSA deve executar seu ciclo completo de detecção e mitigação sem dependência de rede, APIs externas ou intervenção humana para sua função primária. |
| 2 | **Zero Dependências Externas de Runtime** | O agente não depende de serviços externos (TSDB, message brokers, cloud APIs) para operar. Todas as dependências são internas ao binário ou ao kernel do sistema operacional hospedeiro. A comunicação com sistemas externos é **oportunista**: realizada quando disponível, mas nunca requerida. |
| 3 | **Footprint Computacional Previsível** | O consumo de CPU e memória do HOSA deve ser constante e previsível ($O(1)$ em memória, percentual de CPU configurável e limitado). O agente não pode tornar-se causa do problema que pretende resolver. |
| 4 | **Resposta Graduada** | A mitigação não é binária. O HOSA implementa um espectro de respostas proporcionais à severidade e à taxa de variação da anomalia, desde ajuste leve de prioridades até isolamento de rede completo. |
| 5 | **Observabilidade da Decisão** | Toda ação autônoma do HOSA é registrada localmente com justificativa matemática (valores de $D_M$, derivada, limiar acionado, ação executada). O agente é **auditável**. |

### 5.2. Modelo de Execução: O Ciclo Perceptivo-Motor

O HOSA opera em um ciclo contínuo com três camadas funcionais, inspiradas na separação biológica entre sistema sensorial, sistema nervoso e sistema motor:

```
┌─────────────────────────────────────────────────────────────┐
│                    KERNEL SPACE (eBPF)                      │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │   Sondas     │  │   Sondas     │  │    Atuadores      │  │
│  │  Sensoriais  │  │  Sensoriais  │  │   (XDP / cgroup   │  │
│  │ (tracepoints │  │ (kprobes,    │  │    / sched_ext /  │  │
│  │  scheduler,  │  │  PSI hooks,  │  │    mm compaction) │  │
│  │  mm, net)    │  │  mm/vmstat)  │  │                   │  │
│  └──────┬───────┘  └──────┬───────┘  └────────▲──────────┘  │
│         │                 │                   │             │
│         ▼                 ▼                   │             │
│  ┌──────────────────────────────┐             │             │
│  │     eBPF Ring Buffer         │             │             │
│  │  (eventos para user space)   │             │             │
│  └──────────────┬───────────────┘             │             │
│                 │                             │             │
├─────────────────┼─────────────────────────────┼─────────────┤
│                 │    USER SPACE               │             │
│                 ▼                             │             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              MOTOR MATEMÁTICO (Go/Rust)               │  │
│  │                                                       │  │
│  │  1. Recebe eventos do ring buffer                     │  │
│  │  2. Atualiza vetor x(t) + H_frag                     │  │
│  │  3. Atualiza μ e Σ incrementalmente (Welford)         │  │
│  │  4. Calcula D_M(x(t))                                 │  │
│  │  5. Aplica EWMA → D̄_M(t)                              │  │
│  │  6. Calcula dD̄_M/dt e d²D̄_M/dt²                       │  │
│  │  7. Avalia contra limiares adaptativos                │  │
│  │  8. Determina nível de resposta (0-5)                 │  │
│  │  9. Seleciona regime de atuação (Fase 1 ou Fase 2)    │  │
│  │ 10. Envia comando de atuação via BPF maps             │  │
│  │                                                       │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │          COMUNICAÇÃO OPORTUNISTA (Go)                 │  │
│  │                                                       │  │
│  │  - Webhooks para orquestradores (quando disponível)   │  │
│  │  - Exposição de métricas (endpoint local)             │  │
│  │  - Log estruturado local (auditoria)                  │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**Nota arquitetural sobre transição kernel↔user space.** O modelo de execução do HOSA envolve transição entre kernel space (coleta eBPF e atuação) e user space (cálculo matemático). Esta transição utiliza o mecanismo de ring buffer do eBPF e BPF maps, com latência típica na ordem de **microsegundos** (1–10μs em hardware moderno). A terminologia correta é **"zero dependências externas de runtime"**: o HOSA não depende de processos, serviços ou infraestrutura externa ao binário do agente e ao kernel hospedeiro. A transição kernel↔user é interna ao agente.

### 5.3. Fase de Warm-Up e Calibração Proprioceptiva

Ao iniciar, o HOSA executa uma fase de calibração denominada **Propriocepção de Hardware**:

1. **Descoberta topológica:** Via leitura de `/sys/devices/system/node/` e `/sys/devices/system/cpu/`, o agente identifica a topologia NUMA, número de núcleos físicos e lógicos, tamanhos de cache L1/L2/L3, mapa de compartilhamento de L3 por par de núcleos, e configuração de memória por zona NUMA.

2. **Definição do vetor de estado:** Com base na topologia, o HOSA determina quais variáveis incluir no vetor $\vec{x}(t)$ e suas respectivas fontes eBPF. Em sistemas sem suporte a `sched_ext`, a dimensão $H_{frag}$ ainda é coletada; apenas o Escalonador de Sobrevivência não é ativado.

3. **Acumulação basal:** Durante um período configurável (padrão: 5 minutos), o agente coleta amostras sem executar mitigação, acumulando $\vec{\mu}_0$ e $\Sigma_0$ iniciais via Welford incremental. Este é o **perfil basal** do nó.

4. **Calibração de $\alpha$ (EWMA):** O fator de suavização é calibrado para cada recurso com base na variância observada durante o warm-up.

5. **Definição dos limiares adaptativos:** Os limiares de $D_M$ para cada nível de resposta são calculados como múltiplos do desvio padrão observado no regime basal (e.g., Nível 1 = 2σ, Nível 3 = 4σ).

6. **Detecção de suporte a `sched_ext`:** O HOSA verifica a presença de `CONFIG_SCHED_CLASS_EXT=y` no kernel em execução (Linux ≥ 6.11). Se suportado, o programa BPF do Escalonador de Sobrevivência é pré-compilado e mantido em standby para ativação imediata ao atingir Nível 3 de resposta.

Após o warm-up, $\vec{\mu}$ e $\Sigma$ continuam sendo atualizados incrementalmente, permitindo que o perfil basal evolua com mudanças legítimas no workload (Seção 5.5, Habituação).

### 5.4. Sistema de Resposta Graduada

O HOSA implementa **seis níveis de resposta** (0–5), cada um com ações específicas e proporcionais à severidade da anomalia:

| Nível | Condição de Ativação | Ação | Reversibilidade |
|---|---|---|---|
| **0 — Homeostase** | $D_M < \theta_1$ e $\frac{dD_M}{dt} \leq 0$ | Nenhuma. Suprime telemetria redundante (envia heartbeat mínimo). | N/A |
| **1 — Vigilância** | $D_M > \theta_1$ ou $\frac{dD_M}{dt} > 0$ sustentado | Logging local. Aumento da frequência de amostragem. Nenhuma intervenção. | Automática (retorno a N0 quando condição cessa). |
| **2 — Contenção Leve** | $D_M > \theta_2$ e $\frac{dD_M}{dt} > 0$ | Renice de processos não-essenciais via cgroups. Notificação via webhook (oportunista). | Automática (relaxamento gradual de renice). |
| **3 — Contenção Ativa** | $D_M > \theta_3$ e $\frac{d^2D_M}{dt^2} > 0$ (aceleração positiva) | Throttling de CPU/memória em cgroups de processos identificados como contribuintes. Load shedding parcial via XDP (descarte de pacotes de conexões novas, preservando as existentes). **Ativação do Escalonador de Sobrevivência via `sched_ext`**. **Desfragmentação preemptiva de memória** se $H_{frag}$ abaixo do limiar de aviso. Webhook urgente para orquestrador. | Automática com histerese (relaxamento quando $D_M < \theta_2$ por período sustentado). |
| **4 — Contenção Severa** | $D_M > \theta_4$ ou velocidade de convergência indica esgotamento em < T segundos | Throttling agressivo. XDP bloqueia todo tráfego de entrada exceto healthcheck do orquestrador. Freeze de cgroups não-críticos. **Inanição Direcionada** do processo causador. **Isolamento de Tabela de Páginas** para processo invasor. | Requer redução sustentada de $D_M$ abaixo de $\theta_3$ por período estendido. |
| **5 — Quarentena Autônoma** | Falha de contenção nos níveis anteriores. $D_M$ em ascensão descontrolada apesar de mitigações ativas. | **Isolamento de rede**: desativação programática de interfaces de rede (exceto interface de gerência/IPMI, se presente). Processos não-essenciais congelados (SIGSTOP). Log detalhado gravado em armazenamento persistente. Nó sinaliza estado "quarentenado" no último webhook possível. | **Manual**: requer intervenção administrativa para restaurar o nó. |

### 5.4.1. Modos de Quarentena por Classe de Ambiente

A quarentena autônoma (Nível 5) envolve isolamento de rede do nó comprometido. A viabilidade e a estratégia desse isolamento variam fundamentalmente conforme a classe de infraestrutura. O HOSA implementa **modos de quarentena diferenciados**, selecionados automaticamente durante a fase de Propriocepção de Hardware (Seção 5.3) ou configurados explicitamente pelo operador.

| Classe de Ambiente | Detecção Automática | Estratégia de Quarentena | Mecanismo de Recovery |
|---|---|---|---|
| **Bare metal com IPMI/iLO/iDRAC** | Detecção de interface IPMI via `/sys/class/net/` e presença de módulos `ipmi_*` no kernel. | Desativação programática de **todas** as interfaces de rede **exceto** a interface de gerência out-of-band (IPMI/iLO/iDRAC). O nó permanece acessível via console de gerência para diagnóstico e restauração. | Manual via console IPMI. Operador inspeciona logs do HOSA, diagnostica causa raiz, restaura interfaces e reinicia serviços. |
| **VM em cloud pública (AWS, GCP, Azure)** | Detecção via DMI/SMBIOS (`dmidecode`), presença de metadata service (169.254.169.254), e identificação do hypervisor via `/sys/hypervisor/` ou CPUID. | **Não desativa interfaces de rede.** Em vez disso: (1) XDP aplica drop total em todo tráfego de entrada e saída **exceto**: tráfego para o metadata service do cloud provider (169.254.169.254), tráfego DHCP, e tráfego para o endpoint de API do orquestrador (se configurado). (2) HOSA sinaliza estado de quarentena via mecanismo nativo do cloud provider quando disponível: escrita de tag/label na instância via metadata service, publicação em tópico SNS/Pub-Sub (se credenciais pré-configuradas), ou atualização de healthcheck endpoint para retornar HTTP 503 com corpo JSON detalhando o estado. (3) O orquestrador externo é responsável pela decisão de terminate/replace. | O orquestrador externo termina a instância e provisiona substituição. Se o orquestrador não atuar em tempo configurável (padrão: 5 minutos), o HOSA pode executar **self-termination** via API do cloud provider (quando credenciais IAM disponíveis). Self-termination é **desativada por padrão** e requer ativação explícita na configuração. |
| **Kubernetes (pod/container)** | Detecção de execução em container via presença de `/proc/1/cgroup` com namespace de cgroup, variáveis de ambiente `KUBERNETES_SERVICE_HOST`, ou presença de service account montado em `/var/run/secrets/kubernetes.io/`. | O HOSA operando como DaemonSet **não isola o nó** (não tem permissão para desativar interfaces do host). Em vez disso: (1) Aplica contenção máxima via cgroups nos pods identificados como contribuintes. (2) Atualiza o status do Node via Kubernetes API com **taint** `hosa.io/quarantine=true:NoExecute` e **condition** `HOSAQuarantine=True`, causando evacuação automática dos pods pelo scheduler. (3) Emite Event no namespace do pod afetado com tipo `Warning` e razão `HOSAQuarantine`. | Operador ou automação remove a taint após investigação. O node retorna ao pool de scheduling. |
| **Edge/IoT com acesso físico** | Configuração explícita pelo operador (flag `environment: edge-physical`). | Desativação completa de interfaces de rede. O dispositivo opera em modo isolado até intervenção física. Logs são preservados em armazenamento local persistente (flash/eMMC). Se o dispositivo possui LED de status ou display, o HOSA sinaliza estado de quarentena visualmente. | Manual. Técnico de campo acessa o dispositivo, coleta logs, diagnostica e restaura. |
| **Edge/IoT sem acesso físico** | Configuração explícita pelo operador (flag `environment: edge-remote`). | **Quarentena com watchdog timer.** (1) Desativação de interfaces de rede. (2) Ativação de hardware watchdog timer (via `/dev/watchdog`) com timeout configurável (padrão: 30 minutos). (3) Se nenhuma intervenção remota ocorre antes do timeout, o watchdog reinicia o dispositivo, que retorna ao estado pré-quarentena com flag persistente `quarantine_recovery=true`. (4) Ao reiniciar com essa flag, o HOSA entra em modo conservador (apenas logging por período configurável) para permitir diagnóstico remoto antes de retomar mitigação autônoma. | Automática via watchdog reboot, com período de observação pós-recovery. |
| **Ambiente air-gapped (redes classificadas, SCADA/ICS)** | Configuração explícita pelo operador (flag `environment: airgap`). | Idêntico a bare metal, com a adição de que **toda comunicação oportunista é desativada permanentemente** (nenhum webhook, nenhuma exposição de endpoint). O HOSA opera em modo puramente endógeno. Logs são escritos exclusivamente em armazenamento local criptografado e coletados periodicamente por equipe com acesso físico autorizado. | Manual via acesso físico autorizado com procedimento de segurança definido pelo operador. |

**Princípio de design: detecção automática com override manual.** O HOSA tenta detectar automaticamente a classe de ambiente e selecionar o modo de quarentena apropriado. Em caso de ambiguidade, o HOSA assume o modo **mais conservador** (cloud pública — não desativa interfaces), priorizando recuperabilidade sobre isolamento.

**Nota sobre containers e privilégio.** Quando o HOSA opera como container (DaemonSet em Kubernetes), seu acesso a cgroups do host e interfaces de rede depende de capabilities Linux específicas (`CAP_SYS_ADMIN`, `CAP_NET_ADMIN`, `CAP_BPF`). Níveis 0-2 requerem apenas `CAP_BPF` e acesso de leitura a `/sys/`. Níveis 3-4 requerem adicionalmente `CAP_SYS_ADMIN` para manipulação de cgroups. Nível 5 requer `CAP_NET_ADMIN` para manipulação de XDP e, no modo Kubernetes, acesso à API do cluster para aplicação de taints.

### 5.5. Habituação: Adaptação ao Novo Basal

Um problema recorrente em sistemas de detecção de anomalias é o **falso positivo crônico**: quando o workload legítimo muda permanentemente (e.g., deploy de nova versão que consome mais memória), o detector continua sinalizando anomalia indefinidamente.

O HOSA implementa um mecanismo de **habituação** inspirado na neuroplasticidade:

1. Se $D_M$ permanece elevado de forma estável (derivada próxima de zero) por um período configurável sem que nenhuma falha real ocorra (sem OOM, sem timeout, sem crash de processo);
2. O HOSA recalcula $\vec{\mu}$ e $\Sigma$ com peso crescente nas amostras recentes, efetivamente deslocando o perfil basal para o novo regime operacional.

Este mecanismo é implementado via **decaimento exponencial dos pesos** no algoritmo de Welford, atribuindo menor influência a amostras antigas e permitindo que $\Sigma$ reflita a covariância contemporânea do sistema.

### 5.6. Política de Seletividade: O Problema do Throttling

O throttling de processos via cgroups, embora seja uma mitigação eficaz contra esgotamento de recursos, introduz riscos secundários que devem ser explicitamente endereçados:

- **Timeouts em cascata:** Um backend HTTP throttled pode causar acúmulo de conexões upstream, propagando a degradação.
- **Deadlocks de transação:** Um processo throttled durante uma transação de banco de dados pode segurar locks por tempo indeterminado.
- **Starvation de componentes críticos:** Se o kubelet do Kubernetes for throttled, o nó é marcado como `NotReady` e todos os pods são evacuados, potencialmente causando mais dano que o problema original.

O HOSA endereça estes riscos através de uma **lista de proteção** (safelist) de processos e cgroups que nunca são alvo de throttling:

- Processos do kernel (kthreadd, ksoftirqd, etc.)
- O próprio agente HOSA
- Agentes de orquestração (kubelet, containerd, dockerd) quando detectados
- Processos explicitamente marcados pelo operador via configuração ou label de cgroup

O throttling é aplicado preferencialmente aos processos identificados como **maiores contribuintes** para a anomalia, determinados pela decomposição do vetor $\vec{x}(t)$ — os processos cujo consumo de recursos mais contribui para as dimensões onde $D_M$ diverge do basal.

### 5.7. Cenário Walkthrough: Memory Leak em Microsserviço de Pagamento

Esta seção apresenta um cenário end-to-end que ilustra o ciclo perceptivo-motor do HOSA em operação, contrastando-o com o comportamento de um sistema de monitoramento exógeno operando simultaneamente. Os valores numéricos são representativos e baseados em comportamento observado em sistemas de produção; a validação experimental formal é documentada separadamente.

#### Contexto

- **Nó:** VM `worker-node-07` em cluster Kubernetes, 8 vCPUs, 16GB RAM.
- **Workload:** 12 pods, incluindo `payment-service-7b4f` (microsserviço de processamento de pagamentos, Java, 2GB de memória alocada via cgroup).
- **Monitoramento exógeno:** Prometheus com scrape interval de 15 segundos, Alertmanager com regra: `container_memory_usage_bytes > 1.8GB for 1m`.
- **HOSA:** Operando em regime de homeostase (Nível 0) há 6 horas. Perfil basal calibrado. Vetor de estado com 8 dimensões.

#### Timeline

**t = 0s (14:23:07.000) — Início do Memory Leak**

O `payment-service-7b4f` inicia alocação de objetos não-coletados pelo GC do Java (referência circular em cache de sessões). Taxa de vazamento: ~50MB/s.

Estado do sistema neste instante:
```
Vetor x(t):
  cpu_total:     47%    (basal: 45% ± 8%)
  mem_used:      61%    (basal: 58% ± 5%)
  mem_pressure:  12%    (basal: 10% ± 4%)  [PSI some avg10]
  io_throughput: 340 IOPS (basal: 320 ± 60 IOPS)
  io_latency:    2.1ms  (basal: 1.9 ± 0.8ms)
  net_rx:        1,200 req/s (basal: 1,150 ± 200 req/s)
  net_tx:        1,180 resp/s (basal: 1,130 ± 190 resp/s)
  runqueue:      3.2    (basal: 2.8 ± 1.5)

D_M = 1.1    (θ₁ = 3.0, θ₂ = 5.0, θ₃ = 7.0, θ₄ = 9.0)
φ = +0.3
dD̄_M/dt ≈ 0
Nível de Resposta: 0 (Homeostase)
```

Prometheus coletou última métrica há 8 segundos. Próximo scrape em 7 segundos.

**t = 1s (14:23:08.000) — HOSA detecta desvio inicial**

```
  mem_used:      64%    (+3pp em 1s — 2σ acima do esperado para Δt=1s)
  mem_pressure:  18%    (+6pp — transição rápida)

D_M = 2.8
φ = +0.9
dD̄_M/dt = +1.6/s   (positiva — afastamento acelerado)
d²D̄_M/dt² = +1.6/s² (aceleração positiva — não é desaceleração)

Nível de Resposta: 0→1 (Vigilância)
```

**Ações do HOSA:**
- Frequência de amostragem aumentada de 100ms para 10ms.
- Log local: `[VIGILANCE] D_M=2.8 dDM/dt=+1.6 dominant_dim=mem_used(c_j=0.72) mem_pressure(c_j=0.21)`
- Nenhuma intervenção no sistema. Nenhum webhook (não é urgente).

**Prometheus:** Não coletou nenhuma métrica neste intervalo. Não tem consciência do evento.

**t = 2s (14:23:09.000) — Escalação para Contenção Leve**

```
  mem_used:      68%    (+7pp acumulado)
  mem_pressure:  29%    (PSI em ascensão rápida)
  cpu_total:     52%    (GC do Java ativado — CPU sobe por pressão de memória)
  io_latency:    3.8ms  (swap começando a ser utilizado — latência de I/O sobe)

D_M = 4.7
φ = +1.8
dD̄_M/dt = +2.1/s   (acelerando)
d²D̄_M/dt² = +0.5/s² (aceleração positiva sustentada)

ρ(t) = 0.31  (correlação CPU↔memória alterada — CPU subindo por causa de GC,
              não por carga legítima. Correlação mem↔io_latency emergindo
              onde antes não existia → swap ativo)

Nível de Resposta: 1→2 (Contenção Leve)
```

**Ações do HOSA:**
- Decomposição dimensional: `mem_used` contribui 68% de $D_M^2$, `mem_pressure` contribui 19%, `io_latency` contribui 8%.
- Identificação do cgroup contribuinte: `/sys/fs/cgroup/kubepods/pod-payment-service-7b4f/` é o cgroup com maior delta de `memory.current` no último segundo (+102MB).
- **Ação de contenção:** `memory.high` do cgroup do `payment-service-7b4f` reduzido de `2G` para `1.6G`. Isso instrui o kernel a aplicar backpressure de memória (reclaim agressivo) no container, desacelerando a taxa de alocação sem matar o processo.
- Webhook oportunista disparado: `POST /api/v1/alerts` com severidade `warning`, payload contendo vetor de estado e contribuição dimensional.

**Prometheus:** Próximo scrape em 5 segundos. Prometheus ainda exibirá as métricas do scrape anterior (t=-8s), que mostravam sistema saudável.

**t = 4s (14:23:11.000) — Contenção segurando, derivada desacelerando**

```
  mem_used:      72%    (ainda subindo, mas taxa reduzida pela contenção de memory.high)
  mem_pressure:  34%    (subindo, mas desacelerando — reclaim ativo)

D_M = 5.9
φ = +2.1
dD̄_M/dt = +1.2/s   (DESACELERANDO — de +2.1/s para +1.2/s)
d²D̄_M/dt² = -0.45/s² (NEGATIVA — a contenção está funcionando)

Nível de Resposta: 2 (mantém Contenção Leve — d²D̄_M/dt² negativa indica que
                       a mitigação está sendo eficaz. Não escalar desnecessariamente.)
```

**Ações do HOSA:**
- Log: `[CONTAINMENT-HOLDING] D_M=5.9 dDM/dt=+1.2(decreasing) d2DM/dt2=-0.45 action=memory.high_effective target=payment-service-7b4f`
- Mantém `memory.high` em 1.6G. Monitora se a desaceleração continua.
- Processos do `kubelet`, `containerd` e outros pods **não são afetados** — estão na safelist.

**Prometheus:** Executa scrape neste instante (t=4s). Coleta `container_memory_usage_bytes{pod="payment-service-7b4f"} = 1.47GB`. Regra de alerta: "container_memory_usage > 1.8GB for 1m" — **condição não satisfeita** (memória está a 1.47GB graças à contenção do HOSA, e a condição `for 1m` exige sustentação por 60 segundos).

**t = 8s (14:23:15.000) — Estabilização pela contenção**

```
  mem_used:      74%    (estabilizando — reclaim igualando taxa de alocação)
  mem_pressure:  36%    (estável)
  cpu_total:     58%    (GC do Java trabalhando continuamente)

D_M = 6.2
φ = +2.2
dD̄_M/dt = +0.15/s   (quase zero — sistema estabilizando no novo patamar)
d²D̄_M/dt² ≈ 0        (aceleração nula — nem piorando nem melhorando)

Nível de Resposta: 2 (mantém)
```

**O HOSA comprou tempo.** O sistema está contido em um patamar degradado mas funcional. O `payment-service` está lento (backpressure de memória causa latência maior), mas não crashou. Transações em andamento não foram corrompidas. Nenhum processo foi morto.

**t = 35s (14:23:42.000) — Operador recebe webhook do HOSA**

O operador humano ou o sistema de automação recebe o webhook enviado pelo HOSA no t=2s. O payload contém:

```json
{
  "severity": "warning",
  "node": "worker-node-07",
  "timestamp": "2026-03-04T22:23:09.000Z",
  "hosa_level": 2,
  "d_m": 4.7,
  "d_m_derivative": 2.1,
  "dominant_dimension": "mem_used",
  "dominant_contribution_pct": 68,
  "suspected_cgroup": "/kubepods/pod-payment-service-7b4f",
  "action_taken": "memory.high reduced to 1.6G",
  "action_status": "effective (d2DM/dt2 < 0)"
}
```

O operador pode agora tomar ação informada: investigar o memory leak no `payment-service`, fazer rollback do deploy, ou escalar horizontalmente. A informação chegou com **contexto dimensional** (qual recurso, qual processo, qual ação foi tomada, se está funcionando) — não como um alerta binário genérico.

**t = 60s (14:24:07.000) — Cenário contrafactual sem HOSA**

Se o HOSA não estivesse operando:
- A 50MB/s, o container teria alocado ~3GB em 60 segundos, excedendo o limit de 2GB do cgroup.
- O kernel teria acionado o OOM-Killer contra o processo Java do `payment-service` em t ≈ 40s.
- Todas as transações de pagamento em andamento teriam sido abortadas sem graceful shutdown.
- O kubelet teria reiniciado o pod (CrashLoopBackOff), mas o memory leak persistiria.
- O Prometheus finalmente emitiria alerta em t ≈ 100s — **60 segundos após o primeiro crash.**
- Clientes teriam experimentado erros 502/504 em transações financeiras durante todo esse período.

#### Síntese Temporal

```
t=0s      t=1s      t=2s      t=4s       t=8s       t=15s      t=40s     t=100s
 │         │         │         │          │          │          │          │
 │ Leak    │ HOSA    │ HOSA    │ HOSA     │ HOSA     │Prometheus│ SEM HOSA:│Prometheus
 │ inicia  │ detecta │ contém  │ confirma │estabiliza│1° scrape │ OOM-Kill │ alerta
 │         │ (N1)    │ (N2)    │ eficácia │ sistema  │ pós-leak │ (crash)  │ (tarde)
 │         │         │         │          │          │          │          │
 ├─────────┴─────────┤         │          │          │          │          │
 │  INTERVALO LETAL  │         │          │          │          │          │
 │  (2 segundos)     │         │          │          │          │          │
 │  HOSA atuou aqui  │         │          │          │          │          │
 └───────────────────┘         │          │          │          │          │
                               │          │          │          │          │
                    ┌──────────┴──────────┴──────────┤          │          │
                    │  HOSA mantém contenção ativa   │          │          │
                    │  Sistema degradado mas VIVO    │          │          │
                    └────────────────────────────────┘          │          │
                                                                │          │
                                                      ┌─────────┴──────────┤
                                                      │ SEM HOSA: cascata  │
                                                      │ OOM → crash → 502  │
                                                      │ → CrashLoopBackOff │
                                                      │ → alerta tardio    │
                                                      └────────────────────┘
```

O HOSA transformou um cenário de **crash destrutivo com perda de transações** em um cenário de **degradação controlada com preservação de funcionalidade**. O tempo de detecção foi de 1 segundo (vs. >60 segundos do modelo exógeno). A mitigação preservou a integridade das transações em andamento. O operador recebeu informação acionável com contexto dimensional completo.

Este é o Intervalo Letal em operação — e a demonstração de por que a capacidade de decisão imediata deve residir no próprio nó.


---

## 6. Taxonomia de Regimes Operacionais e Classificação Comportamental do HOSA

### 6.1. O Problema da Classificação de Demanda

A eficácia de um sistema de detecção de anomalias depende fundamentalmente da sua capacidade de **distinguir entre variação legítima e deterioração patológica**. Um detector que trata todo desvio como ameaça gera fadiga operacional por falsos positivos. Um detector tolerante demais permite que ataques sofisticados operem abaixo do limiar de percepção.

O desafio é agravado pelo fato de que, do ponto de vista estritamente métrico, cenários radicalmente distintos podem produzir assinaturas superficialmente semelhantes. CPU a 85% pode significar:

- Um dia normal de operação para um servidor de renderização de vídeo;
- Um pico sazonal previsível de Black Friday em um e-commerce;
- Os primeiros milissegundos de um ataque DDoS volumétrico;
- Um cryptominer silencioso consumindo ciclos ociosos.

A métrica isolada é idêntica. O que diferencia esses cenários é a **estrutura multivariável do estresse** — como as variáveis se correlacionam entre si — e a **dinâmica temporal** — como essa correlação evolui ao longo do tempo. É precisamente essa distinção que a Distância de Mahalanobis e suas derivadas permitem capturar.

De forma igualmente crítica, a taxonomia deve reconhecer que a anomalia não é exclusivamente um fenômeno de **excesso**. CPU a 2% em um servidor que deveria estar processando mil requisições por segundo não é homeostase — é **silêncio anômalo**, com implicações financeiras, energéticas e de segurança que a literatura de detecção de anomalias historicamente ignora.

Esta seção formaliza uma **taxonomia de regimes operacionais** organizada como um espectro contínuo bipolar.

---

### 6.2. O Espectro Bipolar Contínuo: Arquitetura da Taxonomia

#### 6.2.1. Princípio Organizador

A taxonomia do HOSA modela os regimes operacionais como um **espectro numérico contínuo centrado na homeostase**, onde:

- O **sinal** do índice indica a **direção** do desvio em relação ao perfil basal;
- A **magnitude** do índice indica a **severidade** do desvio.

```
    Sub-demanda                    Sobre-demanda / Anomalia
    ◄──────────────────────┤├───────────────────────────────────►

    −3      −2      −1      0      +1     +2     +3     +4     +5
    │       │       │       │       │      │      │      │      │
 Silêncio Ociosi-  Ociosi- Homeo- Mudança Sazo-  Adver- Falha  Propa-
 Anômalo  dade    dade    stase  Patamar nali-  sarial Local  gação
         Estru-  Legí-                   dade                 Viral
         tural   tima
```

#### 6.2.2. Justificativa do Design

Esta organização espectral resolve três problemas que taxonomias ad hoc introduzem:

**Simetria conceitual.** A homeostase biológica é inerentemente bidirecional: hipotermia e hipertermia são ambas patologias, com a temperatura basal como referência central. Da mesma forma, o HOSA trata sub-demanda e sobre-demanda como desvios simétricos em relação ao perfil basal.

**Continuidade numérica.** O índice inteiro do regime reflete uma ordenação natural de severidade em cada semi-eixo. Transições entre regimes adjacentes são suaves e auditáveis.

**Uniformidade do framework matemático.** A mesma métrica primária ($D_M$) e o mesmo Índice de Direção de Carga ($\phi$) posicionam qualquer estado observado no espectro.

#### 6.2.3. Direcionalidade: Estendendo a Distância de Mahalanobis

A Distância de Mahalanobis, por ser uma métrica de distância, é inerentemente **não-direcional**. Para posicionar o estado no espectro bipolar, o HOSA define o **Índice de Direção de Carga ($\phi$)**:

Dado o vetor de desvio $\vec{d}(t) = \vec{x}(t) - \vec{\mu}$:

$$\phi(t) = \frac{1}{n} \sum_{j=1}^{n} s_j \cdot \frac{d_j(t)}{\sigma_j}$$

onde:
- $d_j(t) = x_j(t) - \mu_j$ é o desvio da $j$-ésima variável em relação à sua média basal;
- $\sigma_j = \sqrt{\Sigma_{jj}}$ é o desvio padrão basal da $j$-ésima variável;
- $s_j \in \{+1, -1\}$ é o **sinal de carga** da variável: $+1$ se um aumento indica maior carga (CPU utilization, memory usage, network throughput), $-1$ se um aumento indica menor carga (CPU idle, free memory);
- $n$ é a dimensionalidade do vetor de estado.

| Valor de $\phi(t)$ | Significado | Semi-eixo |
|---|---|---|
| $\phi \approx 0$ | Sistema próximo ao basal | Regime 0 |
| $\phi > 0$ | Desvio na direção de **sobrecarga** | Semi-eixo positivo (+1 a +5) |
| $\phi < 0$ | Desvio na direção de **ociosidade** | Semi-eixo negativo (−1 a −3) |

---

### 6.3. Regime 0 — Homeostase Operacional

**Definição:** O estado estacionário normal do nó sob sua carga de trabalho típica.

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | Baixo e estável. Tipicamente $D_M < \theta_1$. |
| $\phi(t)$ | Oscila em torno de zero. Sem tendência direcional sustentada. |
| $\frac{d\bar{D}_M}{dt}$ | Oscila em torno de zero. |
| $\frac{d^2\bar{D}_M}{dt^2}$ | Ruído estacionário de baixa amplitude. |
| Matriz $\Sigma$ | Estável. As correlações entre variáveis são consistentes ao longo do tempo. |

**Comportamento do HOSA:**

- **Nível de Resposta:** 0 (Homeostase).
- **Filtro Talâmico ativo:** O HOSA suprime o envio de telemetria detalhada para sistemas externos. Apenas um heartbeat mínimo é emitido periodicamente, confirmando que o nó está vivo e em homeostase. Isso reduz drasticamente o custo de ingestão de dados (FinOps).
- **Atualização basal:** $\vec{\mu}$ e $\Sigma$ continuam sendo atualizados incrementalmente via Welford.

---

### 6.4. Semi-Eixo Negativo: Regimes de Sub-Demanda (−1, −2, −3)

#### 6.4.1. Justificativa da Inclusão

A totalidade da literatura de detecção de anomalias em sistemas computacionais concentra-se na **anomalia por excesso**. Ao focar exclusivamente na anomalia positiva, a indústria ignora sistematicamente um fenômeno com implicações financeiras, energéticas e de segurança igualmente significativas: a **anomalia por déficit**.

Um servidor que deveria estar processando mil requisições por segundo e está processando zero não está em homeostase. Está em **silêncio anômalo**. Esse silêncio tem custo: a máquina continua consumindo energia elétrica, ocupando espaço em rack, depreciando hardware e gerando custos de licenciamento — tudo sem produzir valor.

Se o HOSA aspira implementar homeostase genuína — e não apenas proteção contra sobrecarga — ele deve detectar e classificar desvios em **ambas as direções** do perfil basal.

---

#### 6.4.2. Regime −1 — Ociosidade Legítima

**Definição:** Redução de demanda compatível com o contexto temporal ou operacional. O consumo de recursos está abaixo do perfil basal global, mas é **coerente** com o perfil basal da janela temporal correspondente (e.g., madrugada em servidor web corporativo; fim de semana em ERP; manutenção programada de upstream).

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | Elevado em relação ao basal global, mas **baixo** em relação ao perfil basal da janela temporal correspondente. |
| $\phi(t)$ | Negativo moderado. |
| $\frac{d\bar{D}_M}{dt}$ | Aproximadamente zero ou com transição suave. |
| $\rho(t)$ | **Baixo** — correlações preservadas. Os recursos diminuem proporcionalmente. |
| Contexto temporal | **Coerente** — o período corresponde a uma janela historicamente de baixa atividade. |

**Comportamento do HOSA:**

| Aspecto | Ação |
|---|---|
| **Nível de Resposta** | 0 (Homeostase) — a ociosidade é esperada. |
| **Filtro Talâmico** | **Maximamente ativo.** Telemetria suprimida ao mínimo absoluto — heartbeat periódico. |
| **Sinalização FinOps** | O HOSA registra localmente as métricas de subutilização e, quando conectividade está disponível, expõe um **relatório de ociosidade** que quantifica horas de ociosidade acumuladas, custo estimado de manter o nó ativo, e recomendação de janela de downscale. |
| **GreenOps — Otimização Energética** | Redução de frequência de CPU via scaling governor (`schedutil` → perfil conservativo) através de escrita em `/sys/devices/system/cpu/cpufreq/`. Redução da frequência de polling de interfaces de rede ociosas via `ethtool` adaptive coalescing. Aumento do intervalo de amostragem do próprio HOSA. |

**Reversibilidade:** Todas as otimizações energéticas são **instantaneamente reversíveis**. Se $\phi(t)$ começa a subir (tráfego retornando), o HOSA restaura frequências e intervalos de amostragem antes que a carga atinja o perfil basal.

---

#### 6.4.3. Regime −2 — Ociosidade Estrutural

**Definição:** O nó está **permanentemente** superdimensionado em relação à demanda real. Não há janela temporal em que seus recursos sejam plenamente utilizados (e.g., instância provisionada com base em estimativa de capacidade incorreta; servidor legado que perdeu relevância operacional).

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | Baixo de forma **crônica**. O sistema raramente se afasta da região de baixo consumo. |
| $\phi(t)$ | Negativo de forma **persistente** — em todas as janelas temporais, inclusive aquelas que deveriam ser de pico. |
| $\frac{d\bar{D}_M}{dt}$ | Aproximadamente zero (estável no patamar baixo). |
| Perfis sazonais | **Ausência de variação significativa** entre janelas de pico e vale. |

**Métrica dedicada: Índice de Provisionamento Excedente (IPE)**

$$IPE = 1 - \frac{\max_{i \in \text{janelas}} \|\vec{\mu}_i\|_{carga}}{\vec{C}_{max}}$$

onde $\vec{C}_{max}$ é o vetor de capacidade máxima do hardware e $\|\vec{\mu}_i\|_{carga}$ é a norma ponderada do vetor de médias do perfil basal $i$, projetada sobre as dimensões de carga.

Um $IPE$ próximo de 1 indica superdimensionamento severo.

**Comportamento do HOSA:**

| Aspecto | Ação |
|---|---|
| **Nível de Resposta** | 0 (não há risco operacional imediato). |
| **Sinalização FinOps (crítica)** | Relatório de superdimensionamento contendo: $IPE$ calculado com dados históricos; vetor de capacidade máxima utilizada vs. capacidade provisionada; sugestão de instância de menor porte compatível com a carga máxima observada; estimativa de economia anual projetada. |
| **Exposição para Orquestrador** | Quando integrado a Kubernetes, o HOSA pode expor o nó com annotation `hosa.io/structurally-idle=true`, permitindo que o cluster autoscaler considere o nó como candidato a descomissionamento. |
| **GreenOps** | Idêntico ao Regime −1, com a adição de que a **persistência** do estado ocioso é registrada como evidência para decisão de decommissioning. |

**Interação com habituação:** Permitida (com sinalização FinOps persistente). O HOSA se habitua à baixa demanda para fins de detecção de anomalia, mas **continua reportando o superdimensionamento** como informação de governança.

---

#### 6.4.4. Regime −3 — Silêncio Anômalo

**Definição:** Queda abrupta ou gradual de atividade **incompatível** com o contexto temporal esperado (e.g., tráfego redirecionado por sequestro de DNS; falha silenciosa de load balancer; processo de aplicação morto sem restart; ataque que derrubou o serviço antes de instalar payload).

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | **Elevação abrupta** (mesmo sendo uma queda de carga — o desvio em relação ao basal é grande). |
| $\phi(t)$ | **Fortemente negativo**, com transição rápida. |
| $\frac{d\bar{D}_M}{dt}$ | **Pico positivo abrupto** (o $D_M$ está subindo rapidamente). |
| $\frac{d\phi}{dt}$ | **Abrupto** — transição rápida para negativo. |
| $\rho(t)$ | **Potencialmente alto** — a queda pode ser desproporcional entre recursos. |
| Contexto temporal | **Incoerente** — a queda ocorre em horário que deveria ser de atividade normal ou alta. |

**Comportamento do HOSA:**

| Aspecto | Ação |
|---|---|
| **Nível de Resposta** | **1 (Vigilância) a 3 (Contenção Ativa)**, dependendo da velocidade e magnitude da queda. |
| **Investigação ativa** | Verificação de processos (os processos de aplicação esperados ainda estão em execução?). Verificação de rede (as interfaces de rede estão operacionais?). Verificação de upstream (se o HOSA conhece os endpoints de upstream, pode executar health check reverso). |
| **Sinalização urgente** | Webhook de prioridade alta: "Nó X reporta atividade significativamente abaixo do esperado para o contexto temporal. Investigação recomendada." |
| **Correlação com ICP** | Se o silêncio anômalo é acompanhado de $ICP$ elevado, o cenário é reclassificado como potencial comprometimento (Regime +5), com escalação correspondente. |

**O discriminante crítico entre Regime −1 e Regime −3:** A coerência temporal. Uma queda de atividade às 03:00 é coerente com o perfil de madrugada (Regime −1). Uma queda de atividade às 10:00 de uma terça-feira, quando o perfil prevê pico, é **incoerente** (Regime −3).

**O paradoxo do silêncio como alarme:** O Silêncio Anômalo é, contraintuitivamente, um dos cenários mais valiosos do HOSA. Quando um servidor para de receber tráfego e todas as métricas estão "verdes" (CPU baixa, memória livre, rede calma), o monitor tradicional reporta: "tudo saudável." O HOSA, por modelar o perfil basal esperado e não apenas os limites de capacidade, detecta que o silêncio é anômalo e sinaliza.

**Interação com habituação:** **Bloqueada.** O HOSA não se habitua a silêncio incoerente com o contexto temporal.

---

#### 6.4.5. Assinatura Matemática Consolidada — Semi-Eixo Negativo

| Indicador | Regime −1 (Legítima) | Regime −2 (Estrutural) | Regime −3 (Anômala) |
|---|---|---|---|
| $D_M(t)$ vs. basal global | Moderado | Baixo crônico | Alto (abrupto) |
| $D_M(t)$ vs. perfil temporal | **Baixo** (coerente) | Baixo em todas as janelas | **Alto** (incoerente) |
| $\phi(t)$ | Negativo moderado | Negativo persistente | **Fortemente negativo** |
| $\frac{d\phi}{dt}$ | Gradual | ≈ 0 (estável) | **Abrupto** |
| $\rho(t)$ | Baixo | Baixo | Variável |
| Coerência temporal | **Sim** | Irrelevante | **Não** |
| $IPE$ | Variável | **Próximo de 1** | Irrelevante |

---

### 6.5. Regime +1 — Alta Demanda Basal (Mudança Permanente de Patamar)

**Definição:** Uma elevação **persistente e não-revertida** no consumo de recursos, causada por mudanças legítimas na natureza do workload (e.g., deploy de nova versão com maior consumo de memória; migração de microsserviço adicional; crescimento orgânico da base de usuários).

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | **Elevação abrupta seguida de estabilização** em um novo platô. $D_M$ permanece acima de $\theta_1$ mas **constante**. |
| $\phi(t)$ | **Positivo**, estável após transitório. |
| $\frac{d\bar{D}_M}{dt}$ | Pico transitório no momento da mudança, seguido de **convergência para zero**. |
| $\frac{d^2\bar{D}_M}{dt^2}$ | Pico negativo após o transitório (desaceleração), seguido de **estabilização em zero**. |
| Matriz $\Sigma$ | **Preservação da estrutura de correlação.** A "forma" da elipsoide de covariância é escalada, não deformada. |

**Razão de deformação da matriz de covariância:**

$$\rho(t) = \frac{\|\Sigma_{recente} - \Sigma_{basal}\|_F}{\|\Sigma_{basal}\|_F}$$

onde $\|\cdot\|_F$ denota a norma de Frobenius. Um $\rho$ baixo com $D_M$ alto indica mudança de patamar com preservação de estrutura (Regime +1). Um $\rho$ alto indica **deformação da estrutura de correlação** (potencialmente Regime +3 ou +4).

**Interação com habituação:** Este regime é o **caso de uso primário da habituação.** Quando os critérios de estabilidade e preservação de covariância são satisfeitos, o HOSA recalibra $\vec{\mu}$ e $\Sigma$ para refletir o novo regime operacional.

**Salvaguarda contra habituação prematura:** A habituação **não é acionada** se a estabilização ocorre em patamar próximo ao limite de segurança física do recurso (e.g., memória > 90%), ou se o SLM (Fase 4) identifica indicadores de comprometimento simultâneos à elevação.

---

### 6.6. Regime +2 — Alta Demanda Sazonal (Periodicidade Previsível)

**Definição:** Variações de demanda que seguem padrões temporais recorrentes (e.g., pico de acessos diários entre 09:00–11:00; queda de tráfego na madrugada; picos semanais ou anuais como Black Friday).

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | **Oscilação periódica** com amplitude e frequência previsíveis. |
| $\phi(t)$ | **Oscila** entre positivo (picos) e negativo (vales), com periodicidade correspondente. |
| $\frac{d\bar{D}_M}{dt}$ | **Oscilação periódica correspondente**. |
| Autocorrelação de $D_M$ | **Picos significativos** em lags correspondentes ao período sazonal. |

**Solução: Perfis Basais Indexados por Contexto Temporal (Ritmo Circadiano Digital)**

O HOSA implementa um mecanismo de **segmentação temporal do perfil basal**. Em vez de manter um único par ($\vec{\mu}$, $\Sigma$), o agente mantém **N perfis basais** indexados por janela temporal:

$$\mathcal{B} = \{(\vec{\mu}_i, \Sigma_i, w_i) \mid i = 1, 2, \ldots, N\}$$

A granularidade de segmentação é determinada automaticamente durante as primeiras semanas de operação através de **análise de autocorrelação** da série temporal de $D_M$:

1. O HOSA acumula a série $D_M(t)$ por um período mínimo de observação (padrão: 7 dias);
2. Calcula a **função de autocorrelação (ACF)** da série;
3. Identifica os **lags com picos de autocorrelação significativos**;
4. Se detectar periodicidade (e.g., pico em lag de 24h), segmenta $\mathcal{B}$ automaticamente em janelas correspondentes;
5. Cada segmento acumula seu próprio perfil basal via Welford independente.

A partir da segmentação, o cálculo de $D_M$ em cada instante $t$ utiliza o perfil basal correspondente à janela temporal corrente:

$$D_M(t) = \sqrt{(\vec{x}(t) - \vec{\mu}_{i(t)})^T \Sigma_{i(t)}^{-1} (\vec{x}(t) - \vec{\mu}_{i(t)})}$$

**Codificação cíclica de variáveis temporais:** Para evitar descontinuidades (23h→0h, domingo→segunda), variáveis temporais são codificadas em componentes senoidais:

$$x_{hora,sin}(t) = \sin\left(\frac{2\pi \cdot hora(t)}{24}\right), \quad x_{hora,cos}(t) = \cos\left(\frac{2\pi \cdot hora(t)}{24}\right)$$

**Interação com habituação:** A habituação ocorre **dentro de cada segmento temporal**, não globalmente.

---

### 6.7. Regime +3 — Alta Demanda Disfarçada (Demanda Adversarial)

**Definição:** Consumo de recursos causado por atividade maliciosa que **deliberadamente mimetiza padrões de demanda legítima** para evadir detecção (e.g., DDoS de camada de aplicação Layer 7; cryptomining parasitário; exfiltração lenta de dados Low-and-Slow; ataques de esgotamento de recursos).

**Assinatura matemática — o que diferencia demanda disfarçada de demanda legítima:**

A tese central desta classificação é que, mesmo quando as **magnitudes** individuais são mantidas em faixa normal, a atividade maliciosa produz **deformação na estrutura de covariância** que a demanda legítima não produz.

| Indicador | Demanda Legítima | Demanda Disfarçada |
|---|---|---|
| $D_M(t)$ | Pode estar em faixa normal ou elevada | Pode estar em faixa normal (evasão por magnitude) |
| Razão de deformação $\rho(t)$ | Baixa — correlações preservadas | **Elevada** — correlações alteradas |
| Perfil de correlação CPU↔Rede | Proporcionais | **Desproporcionais** |
| Entropia de syscalls | Estável | **Alterada** |
| Razão trabalho/recurso | Proporcional | **Desproporcional** |

**Métricas de Segundo Nível — Detecção de Deformação Estrutural:**

**a) Entropia de Shannon do perfil de syscalls:**

$$H(S, t) = -\sum_{i=1}^{k} p_i(t) \log_2 p_i(t)$$

onde $p_i(t)$ é a proporção da $i$-ésima syscall no intervalo $t$. O HOSA mantém um perfil basal de $H_{basal}$ e monitora:

$$\Delta H(t) = |H(S, t) - H_{basal}|$$

**b) Índice de Eficiência de Trabalho (Work Efficiency Index — WEI):**

$$WEI(t) = \frac{\text{throughput de aplicação}(t)}{\text{consumo de recurso computacional}(t)}$$

Em operação legítima, $WEI$ flutua em torno de uma média estável. Cryptomining e processamento parasitário consomem CPU/memória sem produzir throughput de aplicação, causando **queda do WEI**.

**c) Razão de Contexto Kernel/User:**

$$R_{ku}(t) = \frac{\text{CPU em modo kernel}(t)}{\text{CPU em modo user}(t)}$$

Ataques de rede (DDoS, scanning) e malware de rede produzem aumento de tempo em kernel space desproporcional ao tempo em user space.

**Interação com habituação:** A habituação é **bloqueada** quando a razão de deformação $\rho(t)$ excede o limiar de deformação. O HOSA **não se habitua a atividade que deforma a estrutura de covariância**.

Formalmente:

$$\text{Habituação permitida} \iff \left(\frac{d\bar{D}_M}{dt} \approx 0\right) \wedge \left(\rho(t) < \rho_{limiar}\right) \wedge \left(\Delta H(t) < \Delta H_{limiar}\right)$$

---

### 6.8. Regime +4 — Anomalia Não-Viral (Falha Localizada)

**Definição:** Deterioração de recursos causada por falha ou patologia **confinada ao nó local**, sem componente de propagação (e.g., memory leak em processo de aplicação; degradação de disco; bug de aplicação acumulando file descriptors; fork bomb; deadlock; degradação térmica de CPU).

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | Elevação progressiva ou abrupta. |
| $\phi(t)$ | **Positivo**, crescente. |
| $\frac{d\bar{D}_M}{dt}$ | **Positiva sustentada.** A anomalia não reverte espontaneamente. |
| $\frac{d^2\bar{D}_M}{dt^2}$ | Memory leak: $\approx 0$ (crescimento linear). Fork bomb: **positiva e crescente** (crescimento exponencial). |
| $ICP(t)$ | **Baixo** — ausência de indicadores de propagação para outros nós. |

**Decomposição de Contribuição Dimensional:**

Para diagnosticar **quais recursos** estão causando o desvio, o HOSA decompõe o $D_M$ em contribuições por dimensão. Dado o vetor de desvio $\vec{d} = \vec{x}(t) - \vec{\mu}$ e a métrica de Mahalanobis $D_M^2 = \vec{d}^T \Sigma^{-1} \vec{d}$, a contribuição da $j$-ésima dimensão é:

$$c_j = d_j \cdot (\Sigma^{-1} \vec{d})_j$$

As dimensões com maiores $c_j$ são os **contribuintes dominantes** da anomalia.

**Interação com habituação:** A habituação é **bloqueada quando a derivada permanece positiva sustentada**. Anomalias que crescem monotonicamente não são "novos normais" — são falhas progressivas.

---

### 6.9. Regime +5 — Anomalia Viral (Propagação e Contágio)

**Definição:** Atividade maliciosa ou falha em cascata com componente de **propagação entre nós**, onde o nó afetado tenta comprometer, sobrecarregar ou infectar outros sistemas na rede (e.g., worms e malware com capacidade de propagação lateral; movimento lateral pós-comprometimento; cascata de falhas em microsserviços; nó comprometido usado para DDoS interno).

**Métrica formal: Índice de Comportamento de Propagação (ICP)**

$$ICP(t) = w_1 \cdot \hat{C}_{out}(t) + w_2 \cdot \hat{H}_{dest}(t) + w_3 \cdot \hat{F}_{anom}(t) + w_4 \cdot \hat{\rho}_{D_M \leftrightarrow net_{out}}(t)$$

onde:
- $\hat{C}_{out}(t)$: taxa normalizada de novas conexões de saída (contra o perfil basal);
- $\hat{H}_{dest}(t)$: entropia normalizada dos IPs de destino;
- $\hat{F}_{anom}(t)$: taxa normalizada de forks/execs anômalos;
- $\hat{\rho}_{D_M \leftrightarrow net_{out}}(t)$: correlação entre $D_M$ e tráfego de saída (positiva = indicativo viral);
- $w_i$: pesos calibrados empiricamente.

**Estratégia de Calibração dos Pesos $w_i$:**

**Estágio 1 — Inicialização Uniforme.** Na ausência de dados empíricos, os pesos são inicializados uniformemente: $w_i = \frac{1}{4}$ para $i \in \{1, 2, 3, 4\}$. Esta escolha expressa a premissa agnóstica de que, a priori, nenhum indicador parcial é mais informativo que os demais.

**Estágio 2 — Calibração por Análise de Sensibilidade.** Durante a fase experimental, o HOSA é submetido a cenários de ataque controlados com ground truth conhecido. Os pesos são calibrados via **maximização da AUC-ROC**:

$$\vec{w}^* = \arg\max_{\vec{w}} \text{AUC-ROC}\left(\{ICP^{(j)}(\vec{w}), y^{(j)}\}_{j=1}^{M}\right)$$

sujeito às restrições $w_i \geq 0$ e $\sum_i w_i = 1$.

**Estágio 3 — Validação Cruzada e Publicação dos Pesos.** Os pesos calibrados são validados via leave-one-out cross-validation. Os valores finais de $\vec{w}^*$ são publicados como parâmetros de referência.

**Comportamento do HOSA:**

- **ICP baixo + $D_M$ alto:** Anomalia não-viral (Regime +4). HOSA aplica contenção local sem isolamento de rede.
- **ICP alto + $D_M$ alto:** Forte indicação de propagação. HOSA prioriza **isolamento de rede** (Nível 4-5) para proteger o cluster.
- **ICP alto + $D_M$ moderado:** Propagação precoce. HOSA aplica contenção seletiva e restrição de conexões de saída via XDP.

**Interação com habituação:** Habituação é **categoricamente bloqueada** quando $ICP > ICP_{limiar}$.

---

### 6.10. Sinais Contextuais Exógenos como Dimensão Suplementar do Vetor de Estado

#### 6.10.1. Contexto Temporal (Endógeno)

| Sinal | Formato | Uso |
|---|---|---|
| Hora do dia | Inteiro 0–23 ou codificação cíclica | Indexação do perfil basal sazonal |
| Dia da semana | Inteiro 0–6 ou codificação cíclica | Distinção entre perfil weekday e weekend |
| Dia do mês | Inteiro 1–31 | Detecção de sazonalidade mensal |
| Época do ano | Derivável da data | Sazonalidade anual |

#### 6.10.2. Contexto Ambiental (IoT e Edge)

| Sinal | Fonte | Impacto |
|---|---|---|
| **Temperatura ambiente** | Sensores locais (I²C, GPIO) | Temperaturas extremas causam thermal throttling de CPU. Um aumento de CPU load que correlaciona com aumento de temperatura ambiente é provavelmente thermal throttling, não ataque. |
| **Umidade** | Sensores locais | Em ambientes industriais, umidade alta correlaciona com falhas de conectividade em interfaces de rede sem fio. |
| **Tensão de alimentação** | Sensores de power management (ACPI, PMBus) | Flutuações de tensão afetam estabilidade de clock. |
| **Vibração/aceleração** | Acelerômetros | Vibrações mecânicas podem causar erros de leitura em discos rotativos e desconexões intermitentes de cabos. |

#### 6.10.3. Contexto Operacional (Carregado por Configuração)

| Sinal | Formato | Uso |
|---|---|---|
| **Calendário de eventos** | Lista de datas/horários com labels | Permite ao HOSA **relaxar limiares preemptivamente** durante eventos esperados de alta demanda. |
| **Perfil de workload** | Label descritivo (e.g., "web_server", "database") | Permite calibração de pesos relativos no vetor de estado. |
| **Zona geográfica** | Label (e.g., "us-east-1", "factory-floor-B") | Utilizada em Fase 5+ para contextualizar comunicação entre nós do enxame. |
| **Fusos horários de clientes** | Lista de fusos horários predominantes | Refina a segmentação temporal quando os usuários estão em fusos horários diferentes do servidor. |

---

### 6.11. Síntese: Matriz de Classificação Integrada

| Regime | $D_M$ | $\frac{dD_M}{dt}$ | $\frac{d^2D_M}{dt^2}$ | $\phi(t)$ | $\rho(t)$ | $\Delta H$ | $ICP$ | Classificação |
|---|---|---|---|---|---|---|---|---|
| **−3** | Alto (abrupto) | Pico | Variável | **Fortemente negativo** | Variável | Variável | Variável | **Silêncio Anômalo** → Investigação |
| **−2** | Baixo crônico | ≈ 0 | ≈ 0 | **Negativo persistente** | Baixo | Baixo | Baixo | **Superdimensionamento** → FinOps |
| **−1** | Baixo (vs. perfil temporal) | ≈ 0 ou transição suave | ≈ 0 | **Negativo** | Baixo | Baixo | Baixo | **Ociosidade Legítima** → FinOps/GreenOps |
| **0** | Baixo | ≈ 0 | ≈ 0 | ≈ 0 | Baixo | Baixo | Baixo | **Homeostase** |
| **+1** | Alto, estável | ≈ 0 (após transitório) | ≈ 0 | **Positivo** | Baixo | Baixo | Baixo | **Mudança de patamar** → Habituação |
| **+2** | Oscila | Oscila | Oscila | **Oscila** | Baixo | Baixo | Baixo | **Sazonalidade** → Perfis temporais |
| **+3** | Qualquer | Qualquer | Qualquer | Positivo | **Alto** | **Alto** | Variável | **Adversarial** → Contenção |
| **+4** | Crescente | Positiva sustentada | Variável | Positivo | Variável | Baixo | **Baixo** | **Falha localizada** → Contenção graduada |
| **+5** | Variável | Variável | Variável | Variável | Variável | Variável | **Alto** | **Propagação** → Isolamento de rede |

**Nota sobre classificação ambígua:** Em cenários onde os indicadores não apontam inequivocamente para um único regime, o HOSA adota o **princípio da precaução**: classifica temporariamente como o regime de maior severidade compatível com os dados observados. O log de auditoria registra a ambiguidade e os indicadores que levaram à decisão.

**Nota sobre transições entre semi-eixos:** O Regime −3 (Silêncio Anômalo) pode transicionar para o semi-eixo positivo quando a investigação revela que o silêncio é acompanhado de indicadores de comprometimento ($ICP$ elevado, processos anômalos). Neste caso, o estado é reclassificado diretamente como Regime +5. A travessia do ponto zero sem parada em homeostase é registrada como evento de alta prioridade.

---

### 6.12. Implicações para o Mecanismo de Habituação: Regras Consolidadas

**Pré-condições necessárias (todas devem ser satisfeitas simultaneamente):**

$$\text{Habituação} \iff \begin{cases} \left|\frac{d\bar{D}_M}{dt}\right| < \epsilon_d & \text{(estabilização)} \\ \rho(t) < \rho_{limiar} & \text{(covariância preservada)} \\ \Delta H(t) < \Delta H_{limiar} & \text{(syscalls estáveis)} \\ ICP(t) < ICP_{limiar} & \text{(sem propagação)} \\ D_M(t) < D_{M,segurança} & \text{(patamar seguro)} \\ t_{estável} > T_{min} & \text{(estabilização sustentada)} \\ \text{coerência temporal de } \phi(t) & \text{(se } \phi < 0 \text{, coerente com perfil sazonal)} \end{cases}$$

| Regime | Habituação |
|---|---|
| **−3 — Silêncio Anômalo** | **Bloqueada** |
| **−2 — Ociosidade Estrutural** | Permitida (com sinalização FinOps persistente) |
| **−1 — Ociosidade Legítima** | Incorporada nos perfis sazonais |
| **0 — Homeostase** | N/A (é o basal) |
| **+1 — Mudança de patamar** | **Permitida** se pré-condições satisfeitas |
| **+2 — Sazonalidade** | Intra-segmento |
| **+3 — Adversarial** | **Bloqueada** |
| **+4 — Falha localizada** | **Bloqueada** enquanto derivada positiva |
| **+5 — Viral/Propagação** | **Categoricamente bloqueada** |

---

### 6.13. Resumo das Métricas Suplementares

| Métrica | Símbolo | Definição | Seção |
|---|---|---|---|
| Índice de Direção de Carga | $\phi(t)$ | Projeção normalizada ponderada do desvio sobre o eixo de carga | 6.2.3 |
| Índice de Provisionamento Excedente | $IPE$ | Razão entre capacidade provisionada e utilização máxima histórica | 6.4.3 |
| Razão de Deformação de Covariância | $\rho(t)$ | Norma de Frobenius da diferença entre covariância recente e basal, normalizada | 6.5 |
| Entropia de Shannon do perfil de syscalls | $H(S, t)$ e $\Delta H(t)$ | Medida de diversidade e mudança na distribuição de chamadas de sistema | 6.7 |
| Índice de Eficiência de Trabalho | $WEI(t)$ | Razão throughput de aplicação / consumo de recurso | 6.7 |
| Razão Kernel/User | $R_{ku}(t)$ | Proporção de tempo de CPU em kernel space vs. user space | 6.7 |
| Índice de Comportamento de Propagação | $ICP(t)$ | Combinação ponderada de indicadores de atividade viral | 6.9 |
| Contribuição Dimensional | $c_j$ | Decomposição de $D_M^2$ por dimensão do vetor de estado | 6.8 |
| Autocorrelação de $D_M$ | $ACF_{D_M}(\tau)$ | Função de autocorrelação para detecção de periodicidade | 6.6 |
| Entropia de Fragmentação de Memória | $H_{frag}(t)$ | Medida da desordem na distribuição de páginas físicas livres por ordem e zona NUMA | 9.2 |

---

### 6.14. Contribuição Teórica da Taxonomia

A taxonomia de regimes operacionais proposta nesta seção contribui para a literatura de detecção de anomalias em sistemas computacionais ao formalizar duas distinções frequentemente tratadas de forma ad hoc na prática operacional:

**1. Nem todo desvio do basal é uma anomalia, e nem toda anomalia é uma ameaça.** A organização espectral bipolar permite respostas proporcionais à posição no espectro: regimes centrais (−2 a +2) são tratados com adaptação e otimização; regimes extremos (−3, +3 a +5) são tratados com contenção e isolamento.

**2. A anomalia por déficit é tão significativa quanto a anomalia por excesso.** A simetria do espectro em torno do Regime 0 estabelece que o HOSA implementa homeostase genuína — equilíbrio bidirecional — e não apenas proteção contra sobrecarga. O semi-eixo negativo habilita FinOps endógeno, GreenOps como consequência natural e detecção de blackout operacional, capacidades ausentes em agentes locais existentes.


---

## 7. Escolha de Linguagem: Análise de Trade-offs

A escolha de linguagem de implementação para o motor matemático (user space) é uma decisão arquitetural com implicações mensuráveis em latência, previsibilidade e velocidade de desenvolvimento.

| Critério | Go | Rust | C |
|---|---|---|---|
| **Latência de GC** | GC com pausas sub-ms (Go 1.22+), mas não-determinísticas. Mitigável com `sync.Pool`, pré-alocação e tuning de `GOGC`. | Sem GC. Latência determinística. | Sem GC. Latência determinística. |
| **Interação com eBPF** | `internal/sysbpf` — wrapper próprio sobre `SYS_BPF` via `golang.org/x/sys/unix`, sem dependências de terceiros. Implementa apenas o subconjunto necessário ao HOSA: `BPF_MAP_CREATE`, `BPF_MAP_LOOKUP_ELEM`, `BPF_PROG_LOAD` e attach via `perf_event_open`. | `aya-rs` (biblioteca ativa, ecossistema menor) ou wrapper próprio equivalente. | `libbpf` (referência upstream do kernel) — mais completa, porém exige gerência manual de memória. |
| **Velocidade de desenvolvimento** | Alta. Compilação rápida. Concorrência nativa (goroutines). | Média. Borrow checker exige disciplina. Compilação lenta. | Baixa. Gerência manual de memória. |
| **Segurança de memória** | Garantida pelo runtime. | Garantida pelo compilador (sem runtime). | Responsabilidade do programador. |
| **Adequação acadêmica** | Código legível, facilita reprodutibilidade. | Código legível com curva de aprendizado. | Propenso a bugs sutis. |

**Nota sobre independência de pacotes.** O HOSA adota deliberadamente a política de **zero dependências de terceiros para sua função primária**. A única dependência externa é `golang.org/x/sys`, pacote oficial do projeto Go com garantia de manutenção indefinida enquanto o Linux existir como alvo. Todo o restante — parser ELF (`internal/sysbpf/loader.go`), wrapper de syscall BPF (`internal/sysbpf/syscall.go`), manipulação de cgroups (`internal/syscgroup`) e álgebra linear (`internal/linalg`) — é código próprio do repositório. Esta decisão elimina o risco de desuso de dependências, simplifica a auditoria de segurança e garante portabilidade para distribuições Linux sem acesso a registros de pacotes externos (cenários SCADA, air-gapped, embarcados).

**Decisão provisória:** Go para o motor matemático e plano de controle, com o hot path de cálculo implementado com alocação mínima (pré-alocação de slices, `sync.Pool`, `GOGC=off` durante ciclos críticos). A justificativa é pragmática: para o escopo de uma dissertação de mestrado, a velocidade de iteração do Go permite maior foco na validação da tese matemática.

**Compromisso de validação:** A dissertação incluirá benchmarks comparativos do hot path (cálculo de $D_M$, derivadas, decisão) medindo latência p50/p99 e jitter, com discussão explícita sobre se as pausas de GC observadas impactam a janela de detecção em cenários de colapso real. Se as pausas de GC se mostrarem problemáticas nos benchmarks, a migração do hot path para C (via CGo ou processo auxiliar) será documentada como trabalho futuro.

---

## 8. Roadmap: Horizonte Executável e Visão de Longo Prazo

### 8.1. Horizonte Executável (Escopo da Dissertação e Continuidade Imediata)

#### Fase 1: Fundação — O Motor Matemático e o Arco Reflexo (v1.0)

**Escopo:** Implementação completa do ciclo perceptivo-motor com mitigação via cgroups v2 e XDP.

**Entregas:**
- Sondas eBPF para coleta de vetor de estado (CPU, memória, I/O, rede, scheduler) via tracepoints e kprobes
- Motor matemático com Welford incremental, Mahalanobis, EWMA e derivadas
- Propriocepção de hardware (warm-up com calibração automática)
- Sistema de resposta graduada (Níveis 0–4) baseado exclusivamente em manipulação lógica de cgroups e XDP
- Filtro Talâmico: supressão de telemetria redundante em homeostase (heartbeat mínimo)
- Benchmark de latência do ciclo completo (detecção → decisão → atuação)

**Validação experimental:**
- Injeção de falhas controladas: memory leak gradual, fork bomb, CPU burn, flood de rede
- Comparação quantitativa: tempo de detecção e mitigação do HOSA vs. Prometheus+Alertmanager vs. systemd-oomd
- Análise de sensibilidade do parâmetro $\alpha$ (EWMA) e dos limiares adaptativos
- Medição de overhead do agente (CPU, memória, latência adicionada ao sistema)

---

#### Fase 2: O Sistema Nervoso Simpático — Intervenção Física e Termodinâmica (v2.0)

**Escopo:** Transição da mitigação passiva (limites lógicos via cgroups) para a mitigação ativa, alterando as **regras físicas de escalonamento de processador e de topologia de memória física** em tempo de execução via eBPF. O objetivo é eliminar a contenção reativa e substituir os algoritmos padrão do kernel — que priorizam justiça e propósito geral — por algoritmos de **sobrevivência determinística** durante o Intervalo Letal.

A metáfora biológica que governa esta fase é precisa: enquanto a Fase 1 age como o arco reflexo medular (resposta rápida e localizada), a Fase 2 age como o **sistema nervoso simpático** sob stress agudo — redistribuindo ativamente o fluxo de recursos vitais para os órgãos de sobrevivência, privando os periféricos, e alterando as próprias regras metabólicas do organismo. Um animal em fuga não distribui igualmente o fluxo sanguíneo entre todos os órgãos; ele desvia-o dos processos digestivos para os músculos esqueléticos. O HOSA Fase 2 realiza o equivalente computacional desta redistribuição fisiológica.

##### 8.2.1. Entregas de CPU: O Escalonador de Sobrevivência via `sched_ext`

O sistema operativo padrão utiliza o escalonador CFS (Completely Fair Scheduler) ou, em kernels mais recentes (≥ 6.6), o EEVDF (Earliest Eligible Virtual Deadline First). O pressuposto arquitetural fundamental destes algoritmos é a **justiça** na divisão do tempo de CPU: nenhum processo de mesma prioridade deve ser privado de ciclos de processador indefinidamente; todos avançam.

Contudo, durante uma falha em cascata, a justiça computacional é um **suicídio matemático**. Garantir que o processo com memory leak — o processo *causador* da crise — receba sua fatia justa do processador enquanto o banco de dados *crítico* compete pelos mesmos ciclos não é equidade; é a propagação ativa do colapso. O CFS, por design, não distingue entre um processo patológico e um processo vital: ambos são entidades a serem servidas com igualdade.

O `sched_ext` (Extensible Scheduler Class), introduzido no kernel 6.11 como funcionalidade estável, fornece o mecanismo que resolve este paradoxo. Ele permite substituir dinamicamente — sem reinicialização do sistema — o algoritmo de escalonamento por um programa eBPF carregado em tempo de execução.

**Inanição Direcionada (Targeted Starvation):**

Quando o motor matemático atinge o Nível 3 de resposta ($D_M > \theta_3$ com aceleração positiva), o Escalonador de Sobrevivência do HOSA é ativado via `sched_ext`. O programa eBPF do escalonador implementa a seguinte política:

O processo ou cgroup identificado como causador da anomalia — determinado pela decomposição de contribuição dimensional $c_j$ (Seção 6.8) combinada com atribuição de delta de consumo por cgroup via tracepoints — é fisicamente removido da fila principal de despacho (`SCX_DSQ_GLOBAL`) e inserido numa estrutura de quarentena de escalonamento de baixíssima prioridade, com timeslice mínimo (`SCX_SLICE_DFL / 16`). Operacionalmente, ele recebe **zero ciclos de relógio** a menos que *todos* os núcleos disponíveis estejam simultaneamente em estado ocioso absoluto.

Esta é uma distinção técnica fundamental em relação ao throttling via `cpu.max` dos cgroups (Fase 1): o throttling por cgroup opera no domínio do *bandwidth* — o processo recebe uma fração de tempo garantida, mas numa janela maior. A Inanição Direcionada opera no domínio do *despacho* — o processo simplesmente não é escalonado. Em situações de alta utilização de CPU (precisamente as situações em que o HOSA Fase 2 é ativado), essa distinção determina se o banco de dados tem acesso garantido ao processador ou se compete com o processo invasor a cada quantum de tempo.

O log de auditoria do HOSA registra, para cada decisão de inanição direcionada: o PID do processo afetado, o cgroup, a contribuição dimensional $c_j$ que justificou a decisão, o instante de ativação e o $D_M$ no momento da decisão.

**Afinidade de Cache Preditiva (Zero Preempção para Processos Vitais):**

A troca de contexto (context switch) durante uma crise introduz um custo frequentemente ignorado: a **destruição do conteúdo de cache L1 e L2** do processador. Quando o kernel preempta o processo A para executar o processo B no mesmo núcleo físico, as linhas de cache que o processo A havia carregado são progressivamente substituídas pelo footprint de memória do processo B. Quando A retoma a execução, ele encontra o cache frio — suas operações seguintes resultam em cache misses com latências de dezenas a centenas de nanossegundos por acesso, em contraste com 1–4 ns de um hit de L1.

O Escalonador de Sobrevivência do HOSA implementa **Afinidade de Cache Preditiva** para processos marcados como vitais (aqueles na safelist ou explicitamente classificados como críticos via configuração):

O programa `sched_ext` identifica os processos vitais e os associa a um subconjunto de núcleos físicos dedicados — tipicamente os núcleos com maior localidade de cache L3 em relação às estruturas de dados críticas do processo (determinado durante o warm-up via análise de perfil de NUMA e cache, lendo `/sys/devices/system/cpu/cpu*/cache/index*/shared_cpu_map`). O escalonador eBPF emite instruções de afinidade que **proíbem preempção nesse subconjunto de núcleos** por qualquer processo fora da lista de processos vitais. O processo crítico, uma vez despachado para seu núcleo dedicado, mantém o núcleo até que sua fatia de tempo expire ou que o processo bloqueia voluntariamente (I/O, sincronização). Interrupções do kernel (softirq, hardirq) são permitidas, mas preempção por escalonamento de outros processos é bloqueada.

O efeito prático é que os núcleos dedicados operam como **núcleos de tempo real suave** para os processos vitais durante o Intervalo Letal — sem requerer a complexidade de um sistema de tempo real completo (PREEMPT_RT) e sem custos permanentes (a política é revertida automaticamente quando o HOSA retorna ao Nível 0).

**Requisitos de kernel para a Fase 2 de CPU:**

- Linux ≥ 6.11 com `CONFIG_SCHED_CLASS_EXT=y`
- Capacidade `CAP_SYS_ADMIN` para carregamento do programa `sched_ext` via `bpf(BPF_PROG_LOAD)`
- Verificação durante Propriocepção de Hardware; se não disponível, o HOSA opera exclusivamente com cgroups (Fase 1) sem degradação da Fase 1

##### 8.2.2. Entregas de RAM: Termodinâmica e Entropia de Topologia de Memória

O gestor de memória do Linux utiliza o **Alocador Parceiro** (Buddy Allocator), que organiza a memória física em blocos de potências de dois de tamanho crescente. Com o tempo e o uso intenso, a memória física **fragmenta-se**: blocos contíguos de páginas livres tornam-se raros à medida que alocações e liberações de diferentes tamanhos criam lacunas irregulares no espaço de endereçamento físico.

Quando um processo exige um bloco contíguo grande e o sistema está fragmentado, o kernel aciona a **Paralisação por Compactação** (Compaction Stall): o kernel varre o espaço de endereçamento físico movendo páginas de memória de lugar para criar blocos contíguos suficientemente grandes, durante o qual o userspace experimenta latências imprevisíveis de dezenas a centenas de milissegundos. Este fenômeno gera **latência fatal invisível às métricas tradicionais** — nenhuma métrica de CPU, memória total ou I/O de disco a captura diretamente. O sistema parece saudável ao Prometheus enquanto processos críticos travam esperando alocações de memória.

A fragmentação de memória é um fenômeno **termodinâmico** no sentido preciso do termo: a entropia do sistema de alocação aumenta monotonicamente com o uso, e a única forma de revertê-la é através de trabalho ativo de compactação. O Buddy Allocator sem gerenciamento proativo age como um sistema físico isolado: a desordem aumenta até que a "temperatura" (pressão de alocação) force uma compactação destrutiva.

**Cálculo de Entropia Multivariada de Fragmentação:**

O HOSA instrumenta os tracepoints do subsistema de Memória Virtual do kernel:

- `mm_compaction_begin` / `mm_compaction_end`: detecta quando o kernel inicia compactação e sua duração
- `mm_page_alloc_extfrag`: evento emitido quando uma alocação causa fragmentação externa
- `mm_page_alloc_zone_locked`: indica contenção no alocador por zona de memória
- Leitura periódica de `/proc/buddyinfo`: distribuição atual de blocos livres por ordem e por zona NUMA

A partir desses dados, o HOSA calcula a **Entropia de Fragmentação** $H_{frag}(t)$:

$$H_{frag}(t) = -\sum_{o=0}^{O_{max}} \sum_{z \in Z} p_{o,z}(t) \log_2 p_{o,z}(t)$$

onde $p_{o,z}(t)$ é a proporção normalizada de blocos livres de ordem $o$ na zona NUMA $z$ no instante $t$, e $O_{max}$ é a ordem máxima do Buddy Allocator (tipicamente 10, correspondendo a blocos de 4MB).

Em um sistema com memória idealmente desfragmentada, blocos de ordem alta estão disponíveis e $H_{frag}$ é alta (alta entropia de distribuição = muitas combinações possíveis = sistema saudável). Em um sistema severamente fragmentado, apenas blocos de ordem 0 e 1 estão disponíveis, e $H_{frag}$ converge para valores baixos — o espaço de alocação disponível colapsou para as menores granularidades possíveis.

Esta métrica é incorporada ao vetor de estado $\vec{x}(t)$ como dimensão $x_{frag}(t)$ com sinal de carga negativo ($s_{frag} = -1$: uma *queda* de $H_{frag}$ indica *maior* stress). O algoritmo de Welford já implementado na Fase 1 é **reutilizado sem modificação** para manter a média e covariância desta nova dimensão — a generalidade do framework matricial absorve naturalmente a adição de novas variáveis ao vetor de estado.

**Desfragmentação Preemptiva:**

Em resposta à $H_{frag}$ cruzando um limiar de aviso — calibrado durante o warm-up como $\mu_{frag} - 2\sigma_{frag}$, indicando que a fragmentação está se afastando do perfil basal em direção ao perigoso — o HOSA injeta **diretrizes de compactação micro-dosadas em segundo plano** via escrita no arquivo de controle `/proc/sys/vm/compact_memory` ou, com granularidade mais fina, via invocação de `MADV_COLLAPSE` em regiões de memória específicas.

A "micro-dosagem" é fundamental: em vez de solicitar compactação global do sistema (que gera a Compaction Stall que queremos evitar), o HOSA agenda pequenas operações de reorganização de páginas nos intervalos naturais de baixa pressão de CPU — os vales entre bursts de processamento identificados pelo Escalonador de Sobrevivência.

O efeito é análogo a uma arrumação preventiva de casa: em vez de esperar o caos acumular e então realizar uma limpeza geral disruptiva, o sistema mantém a ordem de forma contínua e não-disruptiva durante os momentos de menor atividade. A Compaction Stall é evitada porque o trabalho de compactação é distribuído no tempo, nunca acumulando até o ponto em que uma operação síncrona e bloqueante se torna necessária.

**Isolamento de Tabela de Páginas (Page Table Isolation):**

Em caso de vazamento de memória agudo — identificado pela dimensão $x_{mem\_used}$ dominando a decomposição $c_j$ com derivada positiva sustentada —, o HOSA altera as **regras de alocação do processo invasor**, implementando o equivalente de um isolamento geográfico de memória:

O processo invasor é forçado a consumir páginas exclusivamente de **zonas de memória NUMA mais distantes** (maior latência de acesso) ou de **páginas já submetidas a compressão** (via `zswap`) antes de receberem paginação para swap. Simultaneamente, o HOSA instrui o kernel a marcar como `MADV_PAGEOUT` as páginas do processo invasor que tenham permanecido inativas por mais que uma janela de tempo configurável, empurrando-as agressivamente para a área de troca (swap) e liberando a RAM rápida e contígua para os processos saudáveis.

Esta abordagem contrasta com o OOM-Killer (que *destrói* o processo) e com o throttling por `memory.high` da Fase 1 (que *pressiona* o processo via backpressure): o Isolamento de Tabela de Páginas *degrada* a qualidade da memória disponível para o processo invasor enquanto **preserva a qualidade de acesso** para os processos críticos. O processo invasor ainda recebe memória — ele não é morto —, mas progressivamente pior memória.

**Requisitos de kernel para a Fase 2 de RAM:**

- Linux ≥ 5.8 (suporte a `vmstat` tracepoints via eBPF — já exigido pela Fase 1)
- Capacidade `CAP_SYS_ADMIN` para escrita em `/proc/sys/vm/compact_memory`
- Capacidade `CAP_SYS_PTRACE` para emissão de `MADV_PAGEOUT` em processos de outros UIDs

**Relação com a Fase 1:** A Fase 2 não substitui a Fase 1; ela a estende. O throttling via `memory.high` (Fase 1) continua ativo e eficaz para a maioria dos cenários de contenção. A Fase 2 adiciona uma camada de intervenção *mais profunda* para os casos em que a mitigação lógica da Fase 1 não é suficiente — quando o problema não é apenas *quanto* de memória um processo usa, mas *como* esse uso destrói a topologia física da memória disponível para outros processos.

**Validação experimental da Fase 2:**

- Benchmark de Compaction Stall: injeção controlada de fragmentação via alocador de teste; comparação da frequência e duração de stalls com e sem desfragmentação preemptiva
- Medição de cache efficiency: taxa de L1/L2 cache hits do processo crítico durante contenção com e sem Afinidade de Cache Preditiva
- Overhead da Fase 2: CPU e memória consumidos pelos programas `sched_ext` e pelas operações de compactação vs. baseline

---

### 8.2. Horizonte de Longo Prazo (Escopo de Doutorado e Pesquisa Futura)

As fases a seguir representam direções de pesquisa que dependem da validação empírica das Fases 1-2 e de avanços no estado da arte de seus respectivos campos.

#### Fase 3: Simbiose com Ecossistema (v3.0)

**Escopo:** Integração oportunista com orquestradores e sistemas de monitoramento.

**Entregas:**
- Webhooks para K8s HPA/KEDA: disparo de scale-up preemptivo baseado na derivada de $D_M$
- Exposição de métricas HOSA em formato compatível com Prometheus (para integração com dashboards existentes)
- Endpoint de `/healthz` enriquecido: ao invés de binário (healthy/unhealthy), retorna vetor de estado normalizado
- Sistema Endócrino Digital: métricas de "fadigabilidade" de longo prazo (desgaste térmico, ciclos de escrita em SSD) expostas como labels para o scheduler do Kubernetes

#### Fase 4: Triagem Semântica Local (v4.0)

**Escopo:** Introdução de análise causal pós-contenção.

**Entregas:**
- Small Language Model (SLM) executando localmente, ativado **apenas** após contenção de Nível 3+ para diagnosticar causa raiz provável
- Modelo operando **air-gapped** (sem conexão à internet)
- Células T de Memória: assinaturas de padrões de ataque armazenadas em Bloom Filter eBPF para bloqueio em nanossegundos em caso de recorrência
- Quarentena Autônoma (Nível 5) completa com todos os modos por classe de ambiente (conforme Seção 5.4.1)
- Habituação Neural: recalibração automática do perfil basal quando mudanças de workload são classificadas como benignas pelo SLM

**Nota sobre footprint:** O SLM é um componente **condicional**, ativado apenas em nós com recursos suficientes (mínimo recomendado: 4GB RAM disponível). Em dispositivos com recursos limitados (IoT, Edge de baixa capacidade), a Fase 4 não é implantada, e o HOSA opera exclusivamente com o motor matemático das Fases 1-2.

#### Fase 5: Inteligência de Enxame (v5.0) — *Pesquisa Futura*

**Hipótese de pesquisa:** Nós equipados com HOSA podem estabelecer consenso local sobre o estado do cluster via comunicação P2P leve, reduzindo a dependência do control plane para decisões de saúde coletiva.

**Desafios técnicos reconhecidos:** Consenso distribuído é um problema com décadas de pesquisa (Lamport, 1998; Ongaro & Ousterhout, 2014). A proposta não é reinventar Paxos/Raft, mas investigar se o escopo limitado da decisão (confirmação coletiva de anomalia, não consenso de estado geral) permite protocolos mais leves. A sazonalidade aprendida (alostase antecipatória) permitiria pré-posicionar recursos antes de picos previstos.

#### Fase 6: Aprendizado Federado e Imunidade Coletiva (v6.0) — *Pesquisa Futura*

**Hipótese de pesquisa:** Atualizações de pesos matemáticos (não dados sensíveis) compartilhadas entre instâncias HOSA podem criar imunidade coletiva contra padrões de ataque emergentes.

**Desafios técnicos reconhecidos:** Convergência de aprendizado federado em ambientes heterogêneos (Li et al., 2020); resistência a model poisoning attacks; privacidade diferencial (Dwork & Roth, 2014).

#### Fase 7: Offload para Hardware Dedicado (v7.0) — *Pesquisa Futura*

**Hipótese de pesquisa:** A migração do ciclo perceptivo-motor para hardware dedicado (SmartNIC/DPU) elimina a competição por CPU com as aplicações do nó e permite operação em estados de baixo consumo energético.

**Desafios técnicos reconhecidos:** SmartNICs e DPUs são hardware especializado com custo significativo, potencialmente contradizendo a premissa de ubiquidade de hardware. A programação de SmartNICs (P4, eBPF offloaded) possui limitações de complexidade computacional.

---

#### Fase 8: O Kernel Causal — Inferência Causal e do-Calculus em Ring 0 (v8.0) — *Pesquisa Futura*

Se as Fases 1 a 7 do HOSA construíram o arco reflexo, o sistema nervoso simpático, o sistema imunitário e a rede neural distribuída, a Fase 8 é o momento em que o sistema operativo desenvolve o **Córtex Pré-Frontal**. É a transição absoluta entre um sistema que *reage* a sintomas matemáticos para um sistema que *raciocina* sobre as leis de causa e efeito antes de agir.

##### 8.8.1. A Escada da Causalidade Aplicada ao Silício

A atual engenharia de confiabilidade (SRE) e a monitorização de sistemas vivem paralisadas no primeiro degrau da matemática de Pearl. O objetivo da Fase 8 é forçar o sistema operativo a escalar para o terceiro degrau em tempo real.

**Degrau 1: Associação (A Estatística Cega)**

A Pergunta: *O que está a acontecer?*

O estado atual da indústria opera aqui. O Prometheus vê que a CPU subiu e a latência também. Estão correlacionados. O problema da correlação é que ela é **simétrica** (A correlaciona-se com B, logo B correlaciona-se com A), mas a causalidade é **assimétrica** (A causa B, mas B não causa A). O OOM Killer do Linux opera aqui: ele mata o processo que está a usar mais memória no momento, muitas vezes eliminando a vítima e deixando intacto o processo causador do vazamento. As próprias Fases 1 e 2 do HOSA operam neste degrau — a Distância de Mahalanobis quantifica o *desvio*, mas não raciocina sobre *por que* o desvio ocorreu ou *quem* o causou.

**Degrau 2: Intervenção (A Ação HOSA Fases 1–4)**

A Pergunta: *O que acontece se eu fizer X?*

O HOSA já atua aqui desde a Fase 1. Ele percebe a anomalia e intervém ativamente — aplica limites de cgroups, isola a rede, ativa o Escalonador de Sobrevivência. A Fase 4 (SLM) eleva a sofisticação, permitindo diagnóstico semântico pós-contenção. Mas mesmo com o SLM, o HOSA ainda opera sobre correlações e classificações — ele não constrói um modelo explícito das relações causais entre os processos do sistema.

**Degrau 3: Contrafactuais (A Imaginação Mecânica)**

A Pergunta: *E se eu tivesse agido de forma diferente?*

Este é o abismo que a Fase 8 atravessa. O sistema avalia um cenário hipotético que nunca aconteceu fisicamente. A pergunta que o Kernel Causal formula, em microssegundos, antes de qualquer ação:

> *"Se eu estrangular a CPU do Processo A através da operação matemática do(limitar\_A), o modelo causal prevê que a base de dados estabiliza em 300 milissegundos? Ou se eu redirecionar o tráfego de rede externo antes de limitar A, a ordem de operações produz um resultado melhor?"*

Esta capacidade de simular o futuro antes de agir é o que diferencia um sistema que reage de um sistema que decide.

##### 8.8.2. A Arquitetura do Kernel Causal: Grafos Acíclicos Dirigidos em Ring 0

Para que o Linux consiga executar um contrafactual, ele precisa construir e manter em memória um **Grafo Acíclico Dirigido** (DAG — Directed Acyclic Graph) dinâmico do estado do sistema, representando as relações causais entre processos.

Na Fase 8, os programas eBPF rastreiam o **fluxo de causalidade IPC** (Inter-Process Communication), construindo a topologia causal em tempo real:

**Génese (raiz do grafo causal):** Os tracepoints `sched:sched_process_fork`, `sched:sched_process_exec` e a syscall `clone()` são instrumentados para registar a árvore de parentesco de processos. O HOSA sabe exatamente qual processo pai gerou qual processo filho.

**Comunicação (arestas do grafo):** As syscalls `sendmsg()`, `recvmsg()`, operações em pipes (`write()`/`read()` sobre file descriptors de pipe), e regiões de memória partilhada (`shmget()`, `mmap()` com `MAP_SHARED`) são instrumentadas via kprobes. Se o Microsserviço A (PID 100) escreve num socket UNIX e o Microsserviço B (PID 200) lê desse socket, o HOSA cria uma **aresta direcional** no DAG: $A \rightarrow B$. A aresta carrega metadados: volume médio de dados transferidos, frequência de comunicação, e a variância dessas métricas (calculada via Welford incremental).

**Consumo de Recursos (pesos dos nós):** Os tracepoints `mm:mm_page_fault_user` e as chamadas de alocação interceptadas via kprobes em `__alloc_pages()` e `do_mmap()` associam o consumo de recursos físicos a cada nó do DAG.

O resultado é uma **topologia em tempo real**: o HOSA sabe que a anomalia de latência no Processo B (base de dados) não é um evento isolado, mas um nó descendente do tráfego gerado pelo Processo A (API), que por sua vez é descendente do tráfego de rede externo.

Formalmente, seja $G = (V, E)$ o DAG causal, onde $V = \{P_1, P_2, \ldots, P_k\}$ é o conjunto de processos ativos e $E = \{(P_i, P_j, w_{ij})\}$ é o conjunto de arestas direcionadas com peso $w_{ij}$ representando o volume e frequência de IPC de $P_i$ para $P_j$. O HOSA mantém $G$ em um eBPF Map do tipo `BPF_MAP_TYPE_HASH` para os nós (indexado por PID) e `BPF_MAP_TYPE_ARRAY` para a estrutura de adjacência esparsa. O grafo é atualizado incrementalmente a cada evento de IPC detectado, com um mecanismo de decaimento exponencial para arestas que não recebem tráfego recente.

##### 8.8.3. O Do-Calculus Compilado no Kernel

O coração da Fase 8 é a implementação do **operador $do(\cdot)$** de Pearl dentro do motor de decisão do HOSA.

O operador $do(X = x)$ representa uma *intervenção* — a modificação cirúrgica de uma variável no modelo causal, independentemente de suas causas naturais. A distinção entre $P(Y | X = x)$ (probabilidade condicional — observação) e $P(Y | do(X = x))$ (probabilidade interventional — ação) é o fundamento matemático que separa correlação de causalidade.

Quando o motor matemático de Mahalanobis sinaliza que o nó entrou no "Intervalo Letal", o **Córtex Causal** avalia uma inferência sobre o DAG antes de agir:

**Exemplo concreto:**

- **Sintoma observado:** Processo B (base de dados, PID 200) está a esgotar a RAM. $c_j$ indica que `mem_used` contribui 68% de $D_M^2$.
- **Grafo Causal:** $\text{Tráfego Externo} \rightarrow \text{PID 100 (API)} \rightarrow \text{PID 200 (BD)}$. O peso da aresta $A \rightarrow B$ mostra volume de IPC crescente.
- **Avaliação Contrafactual 1:** $do(\text{matar}_{B})$ → serviço global cai. **Falha catastrófica.** Custo: ∞.
- **Avaliação Contrafactual 2:** $do(\text{limitar\_memória}_{B})$ via `memory.high` → pressão aliviada, mas tráfego de IPC de A ainda flui. **Mitigação temporária.** Custo: moderado.
- **Avaliação Contrafactual 3:** $do(\text{reduzir\_banda}_{A})$ via XDP em PID 100 → a aresta $A \rightarrow B$ transporta menos carga, aliviando a pressão em B na raiz causal. **Mitigação causal.** Custo: baixo.
- **Atuação:** O HOSA estrangula a rede do Processo A, salvando o Processo B.

O sistema operativo acabou de **punir o mandante do crime, não o executor**.

A formalização matemática segue o **algoritmo de identificabilidade de Pearl** (Pearl, 2009, Cap. 3):

$$P(Y | do(X = x)) = \sum_{z} P(Y | X = x, Z = z) \cdot P(Z = z)$$

onde $Z$ são os pais de $X$ no DAG. No contexto do HOSA, $Y$ é o estado de saúde do sistema (função de $D_M$), $X$ é a ação de contenção candidata, e $Z$ é o estado dos processos upstream no DAG.

##### 8.8.4. O Abismo da Engenharia: Desafios Reais de Implementação

**O Limite do Verificador eBPF:**

O verificador do Linux rejeita qualquer programa eBPF que contenha ciclos não limitados (*unbounded loops*). O problema é que percorrer um grafo — os algoritmos de Graph Traversal como DFS ou BFS — requer, por definição, uma iteração cujo número de passos depende do tamanho do grafo em tempo de execução, não de uma constante conhecida em tempo de compilação.

A solução proposta é a **profundidade causal limitada com estrutura de grafo desdobrada** (*unrolled*): em vez de um loop genérico de traversal, o programa eBPF implementa um número fixo $N$ de passos de traversal pré-compilados, onde $N$ é a profundidade máxima de cadeia causal considerada significativa (proposta inicial: $N = 8$ saltos, cobrindo a maioria das arquiteturas de microsserviços reais). O verificador vê um programa com número de instruções definido em tempo de compilação ($N \times \text{custo\_por\_salto}$) e aceita-o.

**Complexidade de Espaço:**

Manter um DAG causal de milhares de threads comunicando em tempo real consome memória. O HOSA propõe uma **Matriz de Adjacência Esparsa** implementada sobre `BPF_MAP_TYPE_HASH` com chave composta $(PID_i, PID_j)$, limitada a um tamanho máximo configurável (proposta: 10.000 arestas ativas). Arestas não atualizadas por um período de decaimento são removidas do mapa. Para sistemas com muitos processos efêmeros, **Bloom Filters Direcionais** podem substituir arestas com baixo $w_{ij}$, mantendo a presença da aresta com custo de espaço mínimo.

**Causalidade Distribuída (Integração com a Fase 5):**

No escopo da Fase 8 isolada (intra-nó), o DAG cobre apenas os processos locais. A longo prazo, se o Processo A está no Servidor 1 e o Processo B está no Servidor 2, o DAG precisa de atravessar a rede física. A proposta de pesquisa é a introdução de um **Causal Trace ID** — um identificador de causalidade embutido em cabeçalhos HTTP/gRPC como extensão de tracing distribuído compatível com OpenTelemetry — que permite ao HOSA no Servidor 2 unir o seu grafo local com o grafo recebido do Servidor 1, construindo um DAG distribuído sem coordenação centralizada.

##### 8.8.5. Impacto Científico e Posicionamento no Estado da Arte

A conclusão da Fase 8 representa uma contribuição original para a interseção de dois campos que raramente dialogam diretamente: a **teoria da inferência causal** (Pearl, 2009; Peters, Janzing & Schölkopf, 2017) e a **engenharia de sistemas operativos**.

A inferência causal tem sido aplicada extensivamente em economia, epidemiologia e, mais recentemente, em aprendizado de máquina. A sua aplicação a sistemas operativos — onde a "observação" são métricas de kernel coletadas em microssegundos e a "intervenção" são operações de controle de recursos em Ring 0 — é, até onde a revisão bibliográfica deste trabalho identificou, ausente da literatura publicada.

O HOSA Fase 8 não é apenas uma extensão de engenharia — é uma proposta de **novo primitivo de raciocínio para sistemas operativos**: a capacidade de um SO raciocinar sobre as consequências causais de suas próprias ações antes de executá-las. A beleza arquitetural da Fase 8 é que ela não deita fora nada do que as Fases anteriores construíram. A coleta de dados da Fase 1 fornece os sinais para o DAG. O Escalonador de Sobrevivência da Fase 2 fornece o atuador que o do-calculus comanda. O SLM da Fase 4 pode interpretar o grafo causal em linguagem natural para o operador.

---

#### Fase 9: eSRE — Formalização Metodológica (v9.0) — *Pesquisa Futura*

**Objetivo:** Consolidação dos princípios do HOSA em uma metodologia aberta denominada **eSRE (Endogenous Site Reliability Engineering)**, documentando as "Leis de Sobrevivência Celular" como práticas recomendadas para design de sistemas resilientes.

**Dependência:** Adoção e validação empírica em ambientes de produção diversos, abrangendo as Fases 1 a 8. Este é um objetivo de disseminação e sistematização metodológica, não de engenharia de novo componente.

---

## 9. Limitações Conhecidas e Fronteiras do Trabalho

A honestidade intelectual exige a documentação explícita das limitações conhecidas:

1. **Pressuposto de distribuição.** A Distância de Mahalanobis assume implicitamente que o perfil basal segue uma distribuição aproximadamente elipsoidal. Workloads com distribuições multimodais podem violar este pressuposto. A dissertação investigará a robustez do detector sob distribuições não-gaussianas e, se necessário, avaliará alternativas como MCD ou Local Outlier Factor (LOF).

2. **Cold start.** Durante a fase de warm-up (primeiros minutos após inicialização), o agente não possui perfil basal suficiente para detecção confiável. Neste intervalo, o HOSA opera em modo conservador (apenas logging, sem mitigação), constituindo uma janela de vulnerabilidade.

3. **Evasão adversária.** Um atacante com conhecimento da arquitetura do HOSA poderia, teoricamente, executar um ataque "low-and-slow" que mantém $D_M$ e suas derivadas abaixo dos limiares de detecção. A análise de resistência a evasão adversária é um tema de pesquisa futura (Fase 6).

4. **Custos do throttling.** Conforme detalhado na Seção 5.6, o throttling pode introduzir efeitos colaterais. A eficácia do mecanismo de safelist e da seleção de processos-alvo será validada experimentalmente.

5. **Escopo do sistema operacional.** O HOSA é projetado exclusivamente para o kernel Linux (≥ 5.8 para Fases 1 e 2 de RAM; ≥ 6.11 com `CONFIG_SCHED_CLASS_EXT=y` para a Fase 2 de CPU). Portabilidade para outros kernels não é um objetivo.

6. **Interação com NUMA e heterogeneidade de hardware.** Sistemas com topologia NUMA complexa (múltiplos sockets, memória heterogênea) podem exibir padrões de pressão localizados que o vetor de estado agregado não captura. A granularidade per-NUMA-node do vetor de estado será investigada.

7. **Verificador eBPF e traversal de grafo.** A restrição de loops não-limitados do verificador eBPF impõe um limite de $N$ saltos no traversal do DAG causal (Fase 8). A escolha de $N$ é um trade-off entre cobertura causal e aprovação pelo verificador, a ser calibrado empiricamente.

8. **`sched_ext` e interação com cgroups.** A interação entre o Escalonador de Sobrevivência via `sched_ext` (Fase 2) e os limites de `cpu.max` via cgroups (Fase 1) requer validação cuidadosa para garantir que os dois mecanismos não produzam comportamentos inesperados quando ambos estão ativos simultaneamente.

---

## 10. Perguntas Antecipadas e Respostas

**P1: "Por que não usar Machine Learning / Deep Learning em vez da Distância de Mahalanobis? Autoencoders, LSTMs e Isolation Forests são mais sofisticados para detecção de anomalias."**

A escolha da Distância de Mahalanobis não é por desconhecimento de técnicas mais complexas — é por **adequação aos requisitos operacionais** do agente.

O HOSA deve operar em qualquer hardware que execute Linux ≥ 5.8, incluindo dispositivos IoT com 512MB de RAM e sem GPU. Autoencoders e LSTMs exigem: (a) infraestrutura de treinamento; (b) runtime de inferência com footprint significativo (TensorFlow Lite ou ONNX Runtime adicionam 10–50MB ao binário); (c) janelas de dados armazenadas para inferência.

A Distância de Mahalanobis com Welford incremental oferece: (a) calibração online sem fase de treinamento separada; (b) footprint de memória $O(n^2)$ fixo (para $n \leq 15$, isso é < 2KB); (c) cálculo em tempo constante por amostra ($O(n^2)$, ~microsegundos para $n = 10$).

Adicionalmente, a Distância de Mahalanobis produz resultado **interpretável**: o operador pode inspecionar quais dimensões contribuem para o desvio ($c_j$), entender a decisão, e auditar o comportamento do agente. Modelos de deep learning são opacos por construção, dificultando auditabilidade — um requisito não-negociável para um agente que executa mitigação autônoma.

---

**P2: "Isso não é apenas um HIDS (Host Intrusion Detection System) com nome diferente?"**

Não. A distinção é estrutural, não cosmética.

| Dimensão | HIDS (e.g., OSSEC, Wazuh) | HOSA |
|---|---|---|
| **Foco primário** | Segurança — detecção de intrusão | Sobrevivência operacional — manutenção da homeostase do nó |
| **Modelo de detecção** | Baseado em regras e assinaturas de ataques conhecidos | Baseado em desvio do perfil basal (model of "known good") |
| **Variáveis monitoradas** | Logs, integridade de arquivos, syscalls suspeitas | Métricas de recursos e suas correlações multivariáveis |
| **Ação** | Alerta. Bloqueio pontual. | Mitigação graduada autônoma: throttling, load shedding, quarentena |
| **Detecção de sub-demanda** | Não | Sim — os Regimes −1, −2, −3 detectam ociosidade estrutural e silêncio anômalo |
| **Dependência de rede** | Tipicamente requer servidor central | Autonomia total para função primária |

O HOSA pode detectar *consequências* de ataques (deformação da covariância, Regime +3), mas não é projetado para substituir ferramentas de segurança especializadas. Ele complementa HIDS da mesma forma que complementa Prometheus: operando em uma camada diferente, em um horizonte temporal diferente, com um objetivo diferente.

---

**P3: "Por que não contribuir com detecção multivariável para o `systemd-oomd` em vez de criar um agente completamente novo?"**

A arquitetura do `systemd-oomd` é fundamentalmente incompatível com o modelo proposto pelo HOSA, por três razões estruturais:

1. **Escopo de monitoramento.** `systemd-oomd` monitora exclusivamente **pressão de memória** (PSI memory). O HOSA monitora $n$ variáveis correlacionadas. Adicionar multivariabilidade ao `oomd` significaria transformá-lo em algo que ele não foi projetado para ser.

2. **Modelo de ação.** `systemd-oomd` tem uma ação: matar o cgroup inteiro. O HOSA implementa 6 níveis de resposta graduada, incluindo throttling seletivo, load shedding parcial e quarentena. Integrar respostas graduadas ao `oomd` exigiria reescrever sua premissa arquitetural.

3. **Acoplamento a systemd.** `systemd-oomd` é um componente do ecossistema systemd. O HOSA é projetado como agente autônomo sem dependência de init system específico, operando em qualquer ambiente Linux.

---

**P4: "O agente de resiliência pode se tornar a causa do problema? O que impede o HOSA de causar um crash?"**

Esta é uma preocupação legítima e central ao design. O HOSA endereça-a através de múltiplos mecanismos:

1. **Footprint controlado e auto-limitado.** O próprio HOSA opera dentro de um cgroup v2 dedicado com limites rígidos de CPU e memória.
2. **Safelist que inclui a si mesmo.** O HOSA é o primeiro item na safelist de processos protegidos contra throttling.
3. **Princípio de mitigação reversível.** Os Níveis 0-4 de resposta são automaticamente reversíveis. Nenhuma ação destrutiva é executada abaixo do Nível 5.
4. **Histerese na escalação.** A transição entre níveis requer sustentação das condições de ativação por períodos mínimos.
5. **Modo dry-run.** O agente pode ser executado em modo de observação pura (logging e cálculo de decisões sem execução de ações).
6. **Compilação determinística.** O binário é compilado estaticamente sem dependências dinâmicas.

Adicionalmente, na Fase 2: o programa `sched_ext` do HOSA tem prioridade de escalonamento garantida acima de todos os processos que ele gerencia; as operações de compactação de memória são micro-dosadas para não causar Compaction Stalls secundárias; o Escalonador de Sobrevivência respeita incondicionalmente processos com políticas de tempo real (`SCHED_FIFO`/`SCHED_RR`).

---

**P5: "Qual a diferença entre o HOSA e projetos como o Meta FBAR (Facebook Auto-Remediation)?"**

| Dimensão | FBAR | HOSA |
|---|---|---|
| **Arquitetura** | Centralizada. Decisões tomadas por servidores centrais com visão global do cluster. | Distribuída/local. Cada nó decide autonomamente. |
| **Dependência de rede** | Total. | Nenhuma para função primária. |
| **Latência de decisão** | Segundos a minutos. | Milissegundos. |
| **Escopo de ação** | Amplo: pode drenar nós, reiniciar serviços, redirecionar tráfego. | Restrito ao nó local. |
| **Disponibilidade** | Proprietário. | Open-source, portável para qualquer Linux ≥ 5.8. |
| **Adequação a Edge/IoT** | Nenhuma. | Projetado para operar em qualquer ambiente. |

O FBAR é a resposta do Meta para remediação em escala — um orquestrador inteligente de ações de infraestrutura. O HOSA é um reflexo local de sobrevivência. São complementares: em um datacenter equipado com FBAR e HOSA, o HOSA estabilizaria o nó nos milissegundos iniciais enquanto o FBAR delibera e executa a remediação sistêmica.

---

**P6: "A Distância de Mahalanobis é uma técnica de 1936. Não é obsoleta?"**

A álgebra linear e o cálculo diferencial são do século XVIII. Continuamos usando-os porque são corretos.

A Distância de Mahalanobis permanece a métrica padrão para detecção de outliers multivariados em estatística industrial (controle de qualidade), diagnóstico médico e engenharia aeroespacial. A razão é que suas propriedades — sensibilidade a correlação, interpretabilidade, e custo computacional previsível — não foram superadas por técnicas mais recentes nos cenários onde essas propriedades são requisitos.

O HOSA não aplica Mahalanobis ingenuamente. Ele a estende com: (a) atualização incremental via Welford; (b) análise de derivadas temporais; (c) regularização para robustez numérica; (d) métricas suplementares para classificação de regime. A Mahalanobis é o **fundamento**, não a totalidade do sistema de detecção.

---

**P7: "Como o HOSA se comporta em sistemas com carga altamente variável (e.g., serverless, funções Lambda, workloads batch esporádicos)?"**

Workloads altamente variáveis representam um desafio legítimo para qualquer detector baseado em perfil basal, e o HOSA endereça-o em camadas:

1. **Perfis sazonais (Seção 6.6):** Se a variabilidade é temporalmente previsível, os perfis indexados por janela temporal capturam a variabilidade legítima.
2. **Habituação (Seção 5.5):** Se a variabilidade é uma mudança permanente de patamar, o mecanismo de habituação recalibra o basal.
3. **Tolerância da derivada:** O HOSA escala respostas com base na **aceleração** do desvio, não apenas na magnitude. Um pico rápido que se estabiliza produz derivada transitoriamente alta seguida de estabilização.
4. **Cenário genuinamente problemático:** Workloads que variam **aleatoriamente** em magnitude e timing, sem padrão temporal. Para estes cenários, a premissa fundamental de "perfil basal" é fraca, e a eficácia do HOSA é reduzida. A investigação de modelos de detecção para workloads não-estacionários sem perfil basal é um tema de pesquisa futura.

---

**P8: "A Fase 2 (sched_ext) não interfere com garantias de escalonamento de processos críticos do sistema?"**

O `sched_ext` opera como uma classe de escalonamento adicional. O HOSA configura o programa `sched_ext` para respeitar processos com políticas de tempo real (`SCHED_FIFO`/`SCHED_RR`): estes recebem prioridade incondicional. Somente processos com política `SCHED_NORMAL` ou `SCHED_BATCH` são gerenciados pela Inanição Direcionada. Processos do kernel (kthreadd, ksoftirqd, kworkers) estão na safelist por padrão e nunca são alvo de inanição. O modo de Afinidade de Cache para processos vitais é aditivo — ele garante acesso prioritário a núcleos dedicados sem remover processamento de processos de tempo real.

---

**P9: "O DAG causal da Fase 8 está sujeito a loops? Processos podem ter comunicação bidirecional."**

Sim. A comunicação bidirecional entre processos produziria ciclos no grafo de comunicação bruto, violando a definição de DAG. O HOSA resolve isto através de duas estratégias:

Primeiro, **orientação temporal**: a aresta $A \rightarrow B$ é criada apenas quando A *inicia* a comunicação. A resposta de B para A é registrada como aresta $B \rightarrow A$ com timestamp posterior. Em janelas de tempo curtas, a maioria das comunicações tem direção causal dominante identificável pela precedência temporal.

Segundo, **detecção e tratamento de ciclos**: o algoritmo de traversal do DAG mantém um conjunto de nós visitados e aborta a traversal se um nó já visitado for encontrado, tratando o ciclo como um loop de feedback sem tentar resolvê-lo causalmente — o contribuinte causal dominante é determinado pelos pesos das arestas ($w_{ij}$), não pela direção da traversal.

---

## 11. Contribuições Esperadas

Este trabalho propõe as seguintes contribuições ao estado da arte:

1. **Formalização do conceito de Resiliência Endógena** como paradigma complementar à observabilidade exógena, com definição precisa dos limites operacionais de cada abordagem.

2. **Modelo de detecção de anomalias multivariável em tempo real** baseado em Mahalanobis com atualização incremental e análise de taxa de variação, validado contra cenários de colapso reais e sintéticos.

3. **Arquitetura de mitigação física via `sched_ext` e controle termodinâmico de memória**: primeira proposta documentada de substituição dinâmica do escalonador de processos do Linux como mecanismo de sobrevivência de nó, combinada com desfragmentação preemptiva baseada em análise de entropia de topologia de páginas físicas.

4. **Arquitetura de referência para agentes de mitigação autônoma com atuação em kernel space**, documentando os trade-offs de design (latência vs. estabilidade, autonomia vs. risco de mitigação).

5. **Análise comparativa quantitativa** do tempo de detecção e mitigação entre o modelo endógeno (HOSA) e o modelo exógeno (Prometheus + Alertmanager + orquestrador), contribuindo dados empíricos para um debate que tem sido predominantemente teórico.

6. **Framework de resposta graduada** para mitigação autônoma, com documentação explícita de riscos e mecanismos de proteção (safelist, histerese, quarentena vs. destruição).

7. **Primeiro framework de inferência causal aplicado a decisões de kernel em tempo real**: implementação do operador $do(\cdot)$ de Pearl sobre DAGs dinâmicos de IPC mantidos em eBPF Maps, com avaliação contrafactual de intervenções de mitigação antes da sua execução.

---

## 12. Referências Bibliográficas

Aggarwal, C. C. (2017). *Outlier Analysis* (2nd ed.). Springer.

Bear, M. F., Connors, B. W., & Paradiso, M. A. (2015). *Neuroscience: Exploring the Brain* (4th ed.). Wolters Kluwer.

Beyer, B., Jones, C., Petoff, J., & Murphy, N. R. (2016). *Site Reliability Engineering: How Google Runs Production Systems*. O'Reilly Media.

Brewer, E. A. (2000). Towards robust distributed systems. *Proceedings of the 19th Annual ACM Symposium on Principles of Distributed Computing (PODC)*.

Burns, B., Grant, B., Oppenheimer, D., Brewer, E., & Wilkes, J. (2016). Borg, Omega, and Kubernetes. *ACM Queue*, 14(1), 70–93.

Cantrill, B., Shapiro, M. W., & Leventhal, A. H. (2004). Dynamic Instrumentation of Production Systems. *Proceedings of the USENIX Annual Technical Conference (ATC)*, 15–28.

Chandola, V., Banerjee, A., & Kumar, V. (2009). Anomaly Detection: A Survey. *ACM Computing Surveys*, 41(3), Article 15.

Dwork, C., & Roth, A. (2014). The Algorithmic Foundations of Differential Privacy. *Foundations and Trends in Theoretical Computer Science*, 9(3–4), 211–407.

Engel, P. M., & Heinen, M. R. (2010). Incremental Learning of Multivariate Gaussian Mixture Models. *Proceedings of the Brazilian Symposium on Artificial Intelligence (SBIA)*.

Forrest, S., Hofmeyr, S. A., & Somayaji, A. (1997). Computer immunology. *Communications of the ACM*, 40(10), 88–96.

Gnanadesikan, R., & Kettenring, J. R. (1972). Robust Estimates, Residuals, and Outlier Detection with Multiresponse Data. *Biometrics*, 28(1), 81–124.

Gregg, B. (2019). *BPF Performance Tools: Linux System and Application Observability*. Addison-Wesley Professional.

Hellerstein, J. L., Diao, Y., Parekh, S., & Tilbury, D. M. (2004). *Feedback Control of Computing Systems*. John Wiley & Sons.

Henze, N., & Zirkler, B. (1990). A Class of Invariant Consistent Tests for Multivariate Normality. *Communications in Statistics — Theory and Methods*, 19(10), 3595–3617.

Heo, T. (2015). Control Group v2. *Linux Kernel Documentation*. https://www.kernel.org/doc/Documentation/cgroup-v2.txt

Horn, P. (2001). Autonomic Computing: IBM's Perspective on the State of Information Technology. *IBM Corporation*.

Hubert, M., Debruyne, M., & Rousseeuw, P. J. (2018). Minimum Covariance Determinant and Extensions. *WIREs Computational Statistics*, 10(3), e1421.

Isovalent. (2022). Tetragon: eBPF-based Security Observability and Runtime Enforcement. Isovalent Open Source. https://tetragon.io/

Lamport, L. (1998). The Part-Time Parliament. *ACM Transactions on Computer Systems*, 16(2), 133–169.

Li, T., Sahu, A. K., Talwalkar, A., & Smith, V. (2020). Federated Learning: Challenges, Methods, and Future Directions. *IEEE Signal Processing Magazine*, 37(3), 50–60.

Mahalanobis, P. C. (1936). On the generalized distance in statistics. *Proceedings of the National Institute of Sciences of India*, 2(1), 49–55.

Mardia, K. V. (1970). Measures of Multivariate Skewness and Kurtosis with Applications. *Biometrika*, 57(3), 519–530.

Ongaro, D., & Ousterhout, J. (2014). In Search of an Understandable Consensus Algorithm. *Proceedings of the USENIX Annual Technical Conference (ATC)*.

Pearl, J. (2009). *Causality: Models, Reasoning, and Inference* (2nd ed.). Cambridge University Press.

Pearl, J., & Mackenzie, D. (2018). *The Book of Why: The New Science of Cause and Effect*. Basic Books.

Penny, K. I. (1996). Appropriate Critical Values When Testing for a Single Multivariate Outlier by Using the Mahalanobis Distance. *Journal of the Royal Statistical Society: Series C*, 45(1), 73–81.

Peters, J., Janzing, D., & Schölkopf, B. (2017). *Elements of Causal Inference: Foundations and Learning Algorithms*. MIT Press.

Poettering, L. (2020). systemd-oomd: A userspace out-of-memory (OOM) killer. *systemd Documentation*. https://www.freedesktop.org/software/systemd/man/systemd-oomd.service.html

Prometheus Authors. (2012). Prometheus: Monitoring System and Time Series Database. Cloud Native Computing Foundation. https://prometheus.io/

Rousseeuw, P. J. (1984). Least Median of Squares Regression. *Journal of the American Statistical Association*, 79(388), 871–880.

Rousseeuw, P. J., & Van Driessen, K. (1999). A Fast Algorithm for the Minimum Covariance Determinant Estimator. *Technometrics*, 41(3), 212–223.

Scholz, D., Raumer, D., Emmerich, P., Kurber, A., Lessman, K., & Carle, G. (2018). Performance Implications of Packet Filtering with Linux eBPF. *Proceedings of the IEEE/IFIP Network Operations and Management Symposium (NOMS)*.

Sysdig. (2016). Falco: Cloud-Native Runtime Security. *Sysdig Open Source*. https://falco.org/

Tang, C., et al. (2020). FBAR: Facebook's Automated Remediation System. *Proceedings of the ACM Symposium on Cloud Computing (SoCC)*.

Torvalds, L., et al. (2024). sched_ext: Extensible Scheduler Class. *Linux Kernel 6.11 Release Notes and Documentation*. https://www.kernel.org/doc/html/latest/scheduler/sched-ext.html

Vieira, M. A., Castanho, M. S., Pacífico, R. D. G., Santos, E. R. S., Júnior, E. P. M. C., & Vieira, L. F. M. (2020). Fast Packet Processing with eBPF and XDP: Concepts, Code, Challenges, and Applications. *ACM Computing Surveys*, 53(1), Article 16.

Weiner, J. (2018). PSI — Pressure Stall Information. *Linux Kernel Documentation*. https://www.kernel.org/doc/html/latest/accounting/psi.html

Welford, B. P. (1962). Note on a Method for Calculating Corrected Sums of Squares and Products. *Technometrics*, 4(3), 419–420.

---

*Fim do Whitepaper — Versão 2.2*