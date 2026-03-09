

# HOSA — Homeostasis Operating System Agent

## Whitepaper & Manifesto Arquitetural

**Autor:** Fabrício Roney de Amorim
**Versão do Documento:** 2.0 — Revisão Crítica
**Contexto Acadêmico:** Fundamentação para dissertação de Mestrado — Unicamp
**Status:** Documento de Visão e Fundamentação Teórica

---

## Resumo

Este documento apresenta o HOSA (Homeostasis Operating System Agent), uma arquitetura de software bio-inspirada para resiliência autônoma de sistemas operacionais Linux. O HOSA propõe a substituição do modelo dominante de telemetria exógena — no qual a detecção de anomalias e a mitigação de falhas dependem de servidores centrais externos — por um modelo de **Resiliência Endógena**, no qual cada nó computacional possui capacidade autônoma de detecção multivariável e mitigação local em tempo real, independentemente de conectividade de rede.

A detecção de anomalias é realizada através de análise estatística multivariável baseada na Distância de Mahalanobis e sua taxa de variação temporal, com coleta de sinais via eBPF (Extended Berkeley Packet Filter) no Kernel Space do Linux. A mitigação é executada através de manipulação determinística de Cgroups v2 e XDP (eXpress Data Path), implementando um sistema de respostas graduadas inspirado no arco reflexo do sistema nervoso humano.

O HOSA não substitui orquestradores ou sistemas de monitoramento global. Ele os complementa ao operar no intervalo temporal em que esses sistemas são estruturalmente incapazes de atuar: os milissegundos entre o início de um colapso e a chegada da primeira métrica ao control plane externo.

**Palavras-chave:** Resiliência Endógena, Computação Autonômica, eBPF, Detecção de Anomalias Multivariável, Distância de Mahalanobis, Sistemas Bio-Inspirados, Edge Computing, SRE.

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

É importante delimitar o escopo desta metáfora: ela é utilizada como **ferramenta heurística de design arquitetural**, não como reivindicação de equivalência funcional entre sistemas biológicos e computacionais. A biologia informa a estrutura de decisão (onde processar, onde atuar, quando escalar), mas a implementação é puramente matemática e de engenharia de sistemas.

### 2.2. Precedentes na Literatura: Computação Autonômica e Imunologia Computacional

A aspiração por sistemas computacionais auto-reguláveis não é inédita. O manifesto de Computação Autonômica da IBM (Horn, 2001) articulou quatro propriedades desejáveis — auto-configuração, auto-otimização, auto-cura e auto-proteção — mas permaneceu predominantemente no nível de visão estratégica, sem fornecer a instrumentação de baixo nível para viabilizá-las com latência sub-milissegundo.

O trabalho de Forrest, Hofmeyr & Somayaji (1997) sobre imunologia computacional estabeleceu os fundamentos teóricos da distinção entre "self" e "non-self" em sistemas computacionais, propondo que processos anômalos podem ser identificados por desvios em sequências de chamadas de sistema (syscalls). O HOSA absorve este princípio na sua camada de triagem comportamental.

O que diferencia o HOSA destes precedentes é a **síntese operacional**: a combinação de detecção multivariável contínua (não baseada em assinaturas) com atuação em kernel space via mecanismos contemporâneos (eBPF, Cgroups v2, XDP) que não existiam quando esses trabalhos foram publicados. O HOSA é, neste sentido, a resposta de engenharia contemporânea a uma necessidade que a literatura identificou há duas décadas.

---

## 3. Trabalhos Relacionados e Posicionamento

Uma contribuição acadêmica responsável exige o confronto explícito com os trabalhos e tecnologias que operam no mesmo espaço de problema. Esta seção mapeia o ecossistema existente e articula a lacuna específica que o HOSA preenche.

### 3.1. Mecanismos Nativos do Kernel Linux

| Mecanismo | Função | Limitação que o HOSA Endereça |
|---|---|---|
| **PSI (Pressure Stall Information)** — Weiner, 2018 | Expõe métricas de pressão de CPU, memória e I/O como porcentagem de tempo em stall. | PSI é um **sensor passivo**: ele quantifica a pressão, mas não executa mitigação. Adicionalmente, PSI é uma métrica unidimensional por recurso — ele não correlaciona CPU, memória, I/O e rede simultaneamente. O HOSA utiliza PSI como uma das entradas do seu vetor de estado multivariável, mas complementa-o com análise de covariância cruzada e taxa de variação. |
| **systemd-oomd** — Poettering, 2020 | Daemon que monitora PSI de memória e mata cgroups inteiros quando pressão excede limiar. | Opera com **limiares estáticos unidimensionais** (pressão de memória apenas). Não considera correlação com outros recursos. Não oferece respostas graduadas — a ação é binária: nada ou kill. |
| **OOM-Killer** | Mecanismo de última instância do kernel para liberar memória. | **Reativo e destrutivo**: ativa-se apenas após o esgotamento total da memória, e usa heurísticas simplificadas (oom_score) que frequentemente eliminam processos críticos. |
| **cgroups v2** — Heo, 2015 | Interface de controle de recursos por grupo de processos. | É um **mecanismo atuador** sem inteligência de decisão associada. Requer que algo externo decida quais limites aplicar e quando. O HOSA utiliza cgroups v2 como seu sistema motor. |

### 3.2. Ferramentas do Ecossistema de Observabilidade

| Ferramenta/Projeto | Função | Diferenciação do HOSA |
|---|---|---|
| **Prometheus + Alertmanager** | Coleta de métricas via pull, armazenamento em TSDB, alertas baseados em regras. | Modelo exógeno clássico. Intervalo de scrape padrão: 15–60s. Latência mínima de alerta: tipicamente >1 minuto. Sem capacidade de atuação. |
| **Sysdig Falco** — Sysdig, 2016 | Detecção de comportamento anômalo em runtime via eBPF, focado em segurança. | Falco detecta violações de política de segurança (syscalls suspeitas), mas **não monitora saúde de recursos** (CPU, memória, I/O) e **não executa mitigação autônoma**. Seu foco é alertar, não atuar. |
| **Cilium Tetragon** — Isovalent, 2022 | Enforcement de políticas de segurança em kernel space via eBPF. | Tetragon permite definir políticas de bloqueio (e.g., "bloquear processo que abrir /etc/shadow"), mas opera sobre **regras estáticas definidas pelo operador**. Não possui modelo estatístico de anomalia, não calcula derivadas de estado, e não implementa respostas graduadas baseadas em severidade. |
| **Pixie (px.dev)** — New Relic | Observabilidade contínua via eBPF sem instrumentação de código. | Pixie é um sistema de **coleta e visualização** — não possui camada de atuação autônoma. |
| **Facebook FBAR** — Tang et al., 2020 | Remediação automática em escala nos datacenters do Meta. | FBAR opera como **sistema centralizado de remediação** com dependência de rede e infraestrutura proprietária. Não é um agente local autônomo. |

### 3.3. A Lacuna Identificada

Nenhuma ferramenta existente no ecossistema combina, em um único agente local:

1. **Detecção multivariável contínua** (correlação entre CPU, memória, I/O, rede e latência de disco em espaço estatístico unificado);
2. **Análise de taxa de variação** (derivada temporal do vetor de estado, detectando aceleração em direção ao colapso e não apenas estado presente);
3. **Atuação local autônoma graduada** (desde throttling seletivo até isolamento de rede, sem dependência de rede ou intervenção humana);
4. **Independência total de infraestrutura externa** para sua função primária de sobrevivência.

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

### 4.2. A Distância de Mahalanobis como Detector de Anomalia

A detecção de anomalia baseada em limiar unidimensional estático (e.g., "CPU > 90%") sofre de uma limitação fundamental: ela ignora a **estrutura de correlação** entre as variáveis. CPU alta com I/O baixo e rede estável pode representar processamento intensivo legítimo. CPU alta com memory pressure crescente, I/O em stall e latência de rede subindo representa colapso iminente. O limiar estático não distingue esses cenários.

A Distância de Mahalanobis (Mahalanobis, 1936) endereça esta limitação ao medir a distância de uma observação $\vec{x}$ em relação à distribuição multivariável definida pelo vetor de médias $\vec{\mu}$ e pela Matriz de Covariância $\Sigma$:

$$D_M(\vec{x}) = \sqrt{(\vec{x} - \vec{\mu})^T \Sigma^{-1} (\vec{x} - \vec{\mu})}$$

A Matriz de Covariância $\Sigma$ captura as correlações entre todas as variáveis. A sua inversa $\Sigma^{-1}$ pondera as dimensões de acordo com sua variância e interdependência. Um vetor $\vec{x}(t)$ que se afasta do perfil basal em dimensões correlacionadas de maneira não-usual produz um $D_M$ elevado, mesmo que nenhuma variável individual tenha excedido um limiar absoluto.

### 4.3. A Derivada Temporal e o Problema da Estabilidade Numérica

O HOSA não atua sobre o valor instantâneo de $D_M$, mas sobre a sua **taxa de variação temporal** — a velocidade e a aceleração com que o sistema se afasta da homeostase.

A primeira derivada $\frac{dD_M}{dt}$ indica a velocidade de afastamento. A segunda derivada $\frac{d^2D_M}{dt^2}$ indica a aceleração — se o sistema está acelerando em direção ao colapso ou desacelerando.

**Problema reconhecido: instabilidade da diferenciação numérica em dados discretos e ruidosos.** A diferenciação numérica é um problema mal-posto (ill-posed) no sentido de Hadamard: pequenas perturbações nos dados de entrada produzem grandes variações na derivada calculada. A segunda derivada amplifica este efeito quadraticamente. Sem tratamento, a segunda derivada de séries temporais ruidosas de kernel oscila violentamente, gerando falsos positivos.

**Solução adotada:** O HOSA implementa uma **Média Móvel Exponencialmente Ponderada (EWMA)** com fator de decaimento $\alpha$ calibrado por recurso antes do cálculo da derivada:

$$\bar{D}_M(t) = \alpha \cdot D_M(t) + (1 - \alpha) \cdot \bar{D}_M(t-1)$$

O fator $\alpha$ controla o trade-off fundamental entre **responsividade** (valores altos de $\alpha$ preservam variações rápidas, mas mantêm ruído) e **estabilidade** (valores baixos de $\alpha$ suavizam o sinal, mas introduzem latência de detecção).

A calibração de $\alpha$ é realizada durante a fase de **warm-up** do agente (Seção 5.2), e constitui um dos parâmetros críticos da arquitetura. A documentação técnica (documento separado) apresentará a análise de sensibilidade de $\alpha$ contra datasets sintéticos e reais de colapso, quantificando o trade-off latência vs. taxa de falsos positivos.

**Alternativa sob investigação:** O Filtro de Kalman unidimensional oferece estimativa ótima do estado em presença de ruído gaussiano, com a vantagem de adaptar-se dinamicamente à variância observada. A análise comparativa EWMA vs. Kalman será apresentada na fase experimental da dissertação.

### 4.4. Atualização Incremental da Matriz de Covariância

O cálculo batch da Matriz de Covariância ($\Sigma$) sobre janelas de dados acumulados é computacionalmente custoso ($O(n^2 \cdot k)$ para $n$ variáveis e $k$ amostras) e introduz alocação de memória proporcional ao tamanho da janela.

O HOSA utiliza o **algoritmo de Welford generalizado** (Welford, 1962) para atualização incremental online de $\Sigma$ e $\vec{\mu}$. Cada nova amostra $\vec{x}(t)$ atualiza $\Sigma$ em $O(n^2)$ com alocação constante ($O(1)$), independentemente do número de amostras acumuladas. Isso elimina a necessidade de armazenar janelas de dados e garante footprint de memória previsível.

### 4.5. Inversão da Matriz de Covariância

A Distância de Mahalanobis requer $\Sigma^{-1}$. Para dimensionalidade moderada ($n \leq 10$), a inversão direta via decomposição de Cholesky é computacionalmente viável e numericamente estável (a Matriz de Covariância é positiva semidefinida por construção). Para dimensionalidade maior, o HOSA pode recorrer à atualização incremental da inversa via fórmula de Sherman-Morrison-Woodbury, evitando recalcular a inversão completa a cada amostra.

**Degenerescência:** Em sistemas com variáveis altamente colineares (e.g., `cpu_user` e `cpu_total`), $\Sigma$ pode tornar-se singular ou mal-condicionada. O HOSA aplica **regularização de Tikhonov** ($\Sigma_{reg} = \Sigma + \lambda I$, com $\lambda$ pequeno) para garantir invertibilidade.

---

## 5. Arquitetura de Engenharia

### 5.1. Princípios Arquiteturais

O design do HOSA é governado por cinco princípios não-negociáveis:

| # | Princípio | Descrição |
|---|---|---|
| 1 | **Autonomia Local** | O HOSA deve executar seu ciclo completo de detecção e mitigação sem dependência de rede, APIs externas ou intervenção humana para sua função primária. |
| 2 | **Zero Dependências Externas de Runtime** | O agente não depende de serviços externos (TSDB, message brokers, cloud APIs) para operar. Todas as dependências são internas ao binário ou ao kernel do sistema operacional hospedeiro. A comunicação com sistemas externos (orquestradores, dashboards) é **oportunista**: realizada quando disponível, mas nunca requerida. |
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
│  │ (tracepoints │  │ (kprobes,    │  │    controllers)   │  │
│  │  scheduler,  │  │  PSI hooks)  │  │                   │  │
│  │  mm, net)    │  │              │  │                   │  │
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
│  │  2. Atualiza vetor x(t)                               │  │
│  │  3. Atualiza μ e Σ incrementalmente (Welford)         │  │
│  │  4. Calcula D_M(x(t))                                 │  │
│  │  5. Aplica EWMA → D̄_M(t)                              │  │
│  │  6. Calcula dD̄_M/dt e d²D̄_M/dt²                       │  │
│  │  7. Avalia contra limiares adaptativos                │  │
│  │  8. Determina nível de resposta (0-5)                 │  │
│  │  9. Envia comando de atuação via BPF maps             │  │
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

**Nota arquitetural sobre transição kernel↔user space.** O modelo de execução do HOSA envolve transição entre kernel space (coleta eBPF e atuação) e user space (cálculo matemático). Esta transição utiliza o mecanismo de ring buffer do eBPF e BPF maps, com latência típica na ordem de **microsegundos** (1–10μs em hardware moderno). O documento anterior descrevia incorretamente este modelo como "zero context switch". A terminologia correta é **"zero dependências externas de runtime"**: o HOSA não depende de processos, serviços ou infraestrutura externa ao binário do agente e ao kernel hospedeiro. A transição kernel↔user é interna ao agente.

### 5.3. Fase de Warm-Up e Calibração Proprioceptiva

Ao iniciar, o HOSA executa uma fase de calibração denominada **Propriocepção de Hardware**:

1. **Descoberta topológica:** Via leitura de `/sys/devices/system/node/` e `/sys/devices/system/cpu/`, o agente identifica a topologia NUMA, número de núcleos físicos e lógicos, tamanhos de cache L1/L2/L3, e configuração de memória.

2. **Definição do vetor de estado:** Com base na topologia, o HOSA determina quais variáveis incluir no vetor $\vec{x}(t)$ e suas respectivas fontes eBPF.

3. **Acumulação basal:** Durante um período configurável (padrão: 5 minutos), o agente coleta amostras sem executar mitigação, acumulando $\vec{\mu}_0$ e $\Sigma_0$ iniciais via Welford incremental. Este é o **perfil basal** do nó.

4. **Calibração de $\alpha$ (EWMA):** O fator de suavização é calibrado para cada recurso com base na variância observada durante o warm-up.

5. **Definição dos limiares adaptativos:** Os limiares de $D_M$ para cada nível de resposta são calculados como múltiplos do desvio padrão observado no regime basal (e.g., Nível 1 = 2σ, Nível 3 = 4σ).

Após o warm-up, $\vec{\mu}$ e $\Sigma$ continuam sendo atualizados incrementalmente, permitindo que o perfil basal evolua com mudanças legítimas no workload (seção 5.5, Habituação).

### 5.4. Sistema de Resposta Graduada

O HOSA implementa **seis níveis de resposta** (0–5), cada um com ações específicas e proporcionais à severidade da anomalia:

| Nível | Condição de Ativação | Ação | Reversibilidade |
|---|---|---|---|
| **0 — Homeostase** | $D_M < \theta_1$ e $\frac{dD_M}{dt} \leq 0$ | Nenhuma. Suprime telemetria redundante (envia heartbeat mínimo). | N/A |
| **1 — Vigilância** | $D_M > \theta_1$ ou $\frac{dD_M}{dt} > 0$ sustentado | Logging local. Aumento da frequência de amostragem. Nenhuma intervenção. | Automática (retorno a N0 quando condição cessa). |
| **2 — Contenção Leve** | $D_M > \theta_2$ e $\frac{dD_M}{dt} > 0$ | Renice de processos não-essenciais via cgroups. Notificação via webhook (oportunista). | Automática (relaxamento gradual de renice). |
| **3 — Contenção Ativa** | $D_M > \theta_3$ e $\frac{d^2D_M}{dt^2} > 0$ (aceleração positiva) | Throttling de CPU/memória em cgroups de processos identificados como contribuintes. Load shedding parcial via XDP (descarte de pacotes de conexões novas, preservando as existentes). Webhook urgente para orquestrador. | Automática com histerese (relaxamento quando $D_M < \theta_2$ por período sustentado). |
| **4 — Contenção Severa** | $D_M > \theta_4$ ou velocidade de convergência indica esgotamento em < T segundos | Throttling agressivo. XDP bloqueia todo tráfego de entrada exceto healthcheck do orquestrador. Freeze de cgroups não-críticos. | Requer redução sustentada de $D_M$ abaixo de $\theta_3$ por período estendido. |
| **5 — Quarentena Autônoma** | Falha de contenção nos níveis anteriores. $D_M$ em ascensão descontrolada apesar de mitigações ativas. | **Isolamento de rede**: desativação programática de interfaces de rede (exceto interface de gerência/IPMI, se presente). Processos não-essenciais congelados (SIGSTOP). Log detalhado gravado em armazenamento persistente. Nó sinaliza estado "quarentenado" no último webhook possível. | **Manual**: requer intervenção administrativa para restaurar o nó. |

**Nota sobre o Nível 5 (substituição do conceito anterior de "Apoptose").** A versão anterior deste documento propunha um kernel panic intencional como mecanismo de defesa extrema. Esta abordagem foi revisada por apresentar riscos inaceitáveis de corrupção de dados (escritas parciais em disco, journals incompletos) e violação de requisitos de integridade em ambientes regulados. A arquitetura revisada substitui o kernel panic por **quarentena controlada**: o nó é isolado da rede para impedir propagação de ameaça, mas seus processos são congelados de forma ordenada — não destruídos — preservando a possibilidade de análise forense e recuperação de dados. O sistema operacional permanece ativo com funcionalidade mínima.

A decisão de quarentena é **autônoma** (não requer intervenção humana para ser *ativada*), consistente com o princípio de independência operacional. A *restauração* do nó, contudo, é manual, funcionando como um análogo do "fusível" elétrico: a proteção é automática, mas o reset exige inspeção humana.

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

Esta seção formaliza uma **taxonomia de regimes operacionais** que o HOSA deve identificar e classificar, detalhando para cada regime: sua definição operacional, sua assinatura matemática no espaço de Mahalanobis, o comportamento esperado do HOSA, e a interação com o mecanismo de habituação.

---

### 6.2. Regime 0 — Demanda Basal (Homeostase Operacional)

**Definição:** O estado estacionário normal do nó sob sua carga de trabalho típica. As variáveis de recurso flutuam dentro de uma faixa previsível, refletindo a atividade ordinária das aplicações hospedadas.

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | Baixo e estável, flutuando próximo à origem do espaço normalizado. Tipicamente $D_M < \theta_1$. |
| $\frac{d\bar{D}_M}{dt}$ | Oscila em torno de zero. Sem tendência direcional sustentada. |
| $\frac{d^2\bar{D}_M}{dt^2}$ | Ruído estacionário de baixa amplitude. |
| Matriz $\Sigma$ | Estável. As correlações entre variáveis são consistentes ao longo do tempo. |

**Comportamento do HOSA:**

- **Nível de Resposta:** 0 (Homeostase).
- **Filtro Talâmico ativo:** O HOSA suprime o envio de telemetria detalhada para sistemas externos. Apenas um heartbeat mínimo é emitido periodicamente, confirmando que o nó está vivo e em homeostase. Isso reduz drasticamente o custo de ingestão de dados (FinOps).
- **Atualização basal:** $\vec{\mu}$ e $\Sigma$ continuam sendo atualizados incrementalmente via Welford, refinando continuamente o perfil basal.

**Interação com habituação:** Este é o **regime de referência**. Todas as detecções de anomalia são medidas como desvios em relação a este estado. A qualidade da detecção depende diretamente da representatividade estatística da fase de warm-up e da acumulação contínua neste regime.

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

Esta seção formaliza uma **taxonomia de regimes operacionais** organizada como um espectro contínuo bipolar, detalhando para cada regime: sua definição operacional, sua assinatura matemática no espaço de Mahalanobis, o comportamento esperado do HOSA, e a interação com o mecanismo de habituação.

---

### 6.2. O Espectro Bipolar Contínuo: Arquitetura da Taxonomia

#### 6.2.1. Princípio Organizador

A taxonomia do HOSA modela os regimes operacionais como um **espectro numérico contínuo centrado na homeostase**, onde:

- O **sinal** do índice indica a **direção** do desvio em relação ao perfil basal;
- A **magnitude** do índice indica a **severidade** do desvio.

O Regime 0 (homeostase) constitui o ponto de referência central. Desvios negativos representam estados de **sub-demanda** (o nó opera abaixo do esperado). Desvios positivos representam estados de **sobre-demanda ou anomalia** (o nó opera acima do esperado ou sob condições patológicas).

```
    Sub-demanda                    Sobre-demanda / Anomalia
    ◄──────────────────────┤├──────────────────────────────►

    −3      −2      −1      0      +1     +2     +3     +4     +5
    │       │       │       │       │      │      │      │      │
 Silêncio Ociosi-  Ociosi- Homeo- Mudança Sazo-  Adver- Falha  Propa-
 Anômalo  dade    dade    stase  Patamar nali-  sarial Local  gação
         Estru-  Legí-                   dade                 Viral
         tural   tima
    │                       │                                   │
    └── severidade ────────►│◄──────────── severidade ──────────┘
        crescente           │              crescente
        (déficit)        BASAL          (excesso/patologia)
```

#### 6.2.2. Justificativa do Design

Esta organização espectral resolve três problemas que taxonomias ad hoc introduzem:

**Simetria conceitual.** A homeostase biológica é inerentemente bidirecional: hipotermia e hipertermia são ambas patologias, com a temperatura basal como referência central. Da mesma forma, o HOSA trata sub-demanda e sobre-demanda como desvios simétricos em relação ao perfil basal, não como categorias ontologicamente distintas.

**Continuidade numérica.** O índice inteiro do regime reflete uma ordenação natural de severidade em cada semi-eixo. Transições entre regimes adjacentes são suaves e auditáveis (e.g., de −1 para −2 quando ociosidade legítima se revela estrutural; de +3 para +4 quando atividade adversarial causa falha localizada).

**Uniformidade do framework matemático.** A mesma métrica primária ($D_M$) e o mesmo Índice de Direção de Carga ($\phi$) posicionam qualquer estado observado no espectro. O sinal de $\phi$ determina o semi-eixo; a combinação de $D_M$, suas derivadas e métricas suplementares determina a posição dentro do semi-eixo.

#### 6.2.3. Direcionalidade: Estendendo a Distância de Mahalanobis

A Distância de Mahalanobis, por ser uma métrica de distância, é inerentemente **não-direcional** — ela mede o quanto o estado se afastou do basal, mas não indica se o desvio é "para cima" (sobrecarga) ou "para baixo" (ociosidade). Para posicionar o estado no espectro bipolar, o HOSA necessita de um indicador de **direção do desvio** no espaço multivariável.

**Índice de Direção de Carga ($\phi$):**

Dado o vetor de desvio $\vec{d}(t) = \vec{x}(t) - \vec{\mu}$, definimos o Índice de Direção de Carga como a projeção normalizada ponderada do desvio sobre o eixo de carga:

$$\phi(t) = \frac{1}{n} \sum_{j=1}^{n} s_j \cdot \frac{d_j(t)}{\sigma_j}$$

onde:
- $d_j(t) = x_j(t) - \mu_j$ é o desvio da $j$-ésima variável em relação à sua média basal;
- $\sigma_j = \sqrt{\Sigma_{jj}}$ é o desvio padrão basal da $j$-ésima variável;
- $s_j \in \{+1, -1\}$ é o **sinal de carga** da variável: $+1$ se um aumento indica maior carga (CPU utilization, memory usage, network throughput), $-1$ se um aumento indica menor carga (CPU idle, free memory);
- $n$ é a dimensionalidade do vetor de estado.

**Interpretação:**

| Valor de $\phi(t)$ | Significado | Semi-eixo |
|---|---|---|
| $\phi \approx 0$ | Sistema próximo ao basal | Regime 0 |
| $\phi > 0$ | Desvio na direção de **sobrecarga** | Semi-eixo positivo (+1 a +5) |
| $\phi < 0$ | Desvio na direção de **ociosidade** | Semi-eixo negativo (−1 a −3) |

O $\phi$ complementa o $D_M$: enquanto $D_M$ mede a **magnitude** do desvio, $\phi$ indica a **direção**. Juntos, eles posicionam o estado do sistema em um espaço bidimensional (magnitude × direção) que mapeia diretamente para o espectro:

```
                        D_M alto
                           │
                           │
     Regime −3             │          Regimes +3 a +5
   (Silêncio               │        (Adversarial /
    Anômalo)               │         Falha / Viral)
                           │
    ←─── φ < 0 ────────────┼──────────── φ > 0 ───→
                           │
     Regimes −1, −2        │          Regimes +1, +2
   (Ociosidade             │        (Mudança patamar /
    leve/estrutural)       │          Sazonalidade)
                           │
                        D_M baixo
```

---

### 6.3. Regime 0 — Homeostase Operacional

**Definição:** O estado estacionário normal do nó sob sua carga de trabalho típica. As variáveis de recurso flutuam dentro de uma faixa previsível, refletindo a atividade ordinária das aplicações hospedadas.

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | Baixo e estável, flutuando próximo à origem do espaço normalizado. Tipicamente $D_M < \theta_1$. |
| $\phi(t)$ | Oscila em torno de zero. Sem tendência direcional sustentada. |
| $\frac{d\bar{D}_M}{dt}$ | Oscila em torno de zero. Sem tendência direcional sustentada. |
| $\frac{d^2\bar{D}_M}{dt^2}$ | Ruído estacionário de baixa amplitude. |
| Matriz $\Sigma$ | Estável. As correlações entre variáveis são consistentes ao longo do tempo. |

**Comportamento do HOSA:**

- **Nível de Resposta:** 0 (Homeostase).
- **Filtro Talâmico ativo:** O HOSA suprime o envio de telemetria detalhada para sistemas externos. Apenas um heartbeat mínimo é emitido periodicamente, confirmando que o nó está vivo e em homeostase. Isso reduz drasticamente o custo de ingestão de dados (FinOps).
- **Atualização basal:** $\vec{\mu}$ e $\Sigma$ continuam sendo atualizados incrementalmente via Welford, refinando continuamente o perfil basal.

**Interação com habituação:** Este é o **regime de referência**. Todas as detecções de anomalia são medidas como desvios em relação a este estado. A qualidade da detecção depende diretamente da representatividade estatística da fase de warm-up e da acumulação contínua neste regime.

---

### 6.4. Semi-Eixo Negativo: Regimes de Sub-Demanda (−1, −2, −3)

#### 6.4.1. Justificativa da Inclusão

A totalidade da literatura de detecção de anomalias em sistemas computacionais concentra-se na **anomalia por excesso**: consumo de recursos acima do esperado, tráfego acima do previsto, latência acima do aceitável. Essa assimetria reflete um viés operacional compreensível — é o excesso que derruba serviços. Porém, ao focar exclusivamente na anomalia positiva, a indústria ignora sistematicamente um fenômeno com implicações financeiras, energéticas e de segurança igualmente significativas: a **anomalia por déficit**.

Um servidor que deveria estar processando mil requisições por segundo e está processando zero não está em homeostase. Está em **silêncio anômalo**. Esse silêncio tem custo: a máquina continua consumindo energia elétrica, ocupando espaço em rack, depreciando hardware e gerando custos de licenciamento — tudo sem produzir valor.

Mais criticamente, o silêncio anômalo pode ser **sintoma de comprometimento**. Um servidor cujo tráfego desapareceu subitamente pode indicar sequestro de DNS, redirecionamento BGP, falha de upstream invisível ao nó local, ou um atacante que derrubou o processo de aplicação para substituí-lo.

Se o HOSA aspira implementar homeostase genuína — e não apenas proteção contra sobrecarga — ele deve detectar e classificar desvios em **ambas as direções** do perfil basal. O semi-eixo negativo do espectro formaliza esta capacidade.

---

#### 6.4.2. Regime −1 — Ociosidade Legítima

**Definição:** Redução de demanda compatível com o contexto temporal ou operacional. O consumo de recursos está abaixo do perfil basal global, mas é **coerente** com o perfil basal da janela temporal correspondente. Exemplos:

- Madrugada em servidor de aplicação web corporativa;
- Fim de semana em servidor de ERP;
- Manutenção programada de upstream.

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | Elevado em relação ao basal **global**, mas **baixo** em relação ao perfil basal da janela temporal correspondente (se perfis sazonais já estiverem calibrados — Seção 6.7). |
| $\phi(t)$ | Negativo moderado. |
| $\frac{d\bar{D}_M}{dt}$ | Aproximadamente zero ou com transição suave (a queda de demanda foi gradual e previsível). |
| $\rho(t)$ | **Baixo** — a estrutura de correlação é preservada. Os recursos diminuem proporcionalmente, mantendo as mesmas relações entre si. Menos rede → menos CPU → menos I/O, na mesma proporção do perfil basal. |
| Contexto temporal | **Coerente** — o período corresponde a uma janela historicamente de baixa atividade. |

**Comportamento do HOSA:**

| Aspecto | Ação |
|---|---|
| **Nível de Resposta** | 0 (Homeostase) — a ociosidade é esperada. |
| **Filtro Talâmico** | **Maximamente ativo.** Se o nó está ocioso e saudável, a telemetria é suprimida ao mínimo absoluto — heartbeat periódico confirmando status "vivo, ocioso, saudável". |
| **Sinalização FinOps** | O HOSA registra localmente as métricas de subutilização e, quando conectividade está disponível, expõe um **relatório de ociosidade** que quantifica: horas de ociosidade acumuladas, custo estimado de manter o nó ativo durante esse período (se configurado com dados de custo por hora da instância), e recomendação de janela de downscale. |
| **GreenOps — Otimização Energética** | O HOSA pode instruir o kernel a aplicar otimizações de energia locais que não afetam a capacidade de retorno rápido à operação: |
| | • Redução de frequência de CPU via scaling governor (`schedutil` → perfil conservativo) através de escrita em `/sys/devices/system/cpu/cpufreq/` |
| | • Redução da frequência de polling de interfaces de rede ociosas via `ethtool` adaptive coalescing |
| | • Aumento do intervalo de amostragem do próprio HOSA (se o sistema está ocioso, amostrar a cada 500ms em vez de 10ms reduz o consumo do próprio agente) |

**Reversibilidade:** Todas as otimizações energéticas são **instantaneamente reversíveis**. Se $\phi(t)$ começa a subir (tráfego retornando), o HOSA restaura frequências e intervalos de amostragem antes que a carga atinja o perfil basal, garantindo que o nó esteja em plena capacidade quando a demanda chegar.

**Interação com habituação:** Se perfis sazonais estão calibrados, o Regime −1 já faz parte do basal da janela temporal correspondente — não é uma anomalia, é a normalidade daquele segmento temporal. Se não estão calibrados, o Regime −1 contribui para a acumulação dos perfis sazonais.

---

#### 6.4.3. Regime −2 — Ociosidade Estrutural

**Definição:** O nó está **permanentemente** superdimensionado em relação à demanda real. Não há janela temporal em que seus recursos sejam plenamente utilizados. Exemplos:

- Instância provisionada com base em estimativa de capacidade incorreta;
- Servidor legado que perdeu relevância operacional mas não foi decommissionado;
- Infraestrutura provisionada para picos projetados que nunca se materializaram.

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | Baixo de forma **crônica**. O sistema raramente se afasta da região de baixo consumo. |
| $\phi(t)$ | Negativo de forma **persistente** — em todas as janelas temporais, inclusive aquelas que deveriam ser de pico. |
| $\frac{d\bar{D}_M}{dt}$ | Aproximadamente zero (estável no patamar baixo). |
| Perfis sazonais | **Ausência de variação significativa** entre janelas de pico e vale — o nó é ocioso independentemente do horário. |
| Utilização máxima histórica | Mesmo nos picos de demanda sazonal, a utilização máxima permanece uma fração pequena da capacidade provisionada. |

**Métrica dedicada: Índice de Provisionamento Excedente (IPE)**

$$IPE = 1 - \frac{\max_{i \in \text{janelas}} \|\vec{\mu}_i\|_{carga}}{\vec{C}_{max}}$$

onde $\vec{C}_{max}$ é o vetor de capacidade máxima do hardware (total de CPU, total de memória, etc.) e $\|\vec{\mu}_i\|_{carga}$ é a norma ponderada do vetor de médias do perfil basal $i$, projetada sobre as dimensões de carga.

Um $IPE$ próximo de 1 indica que, mesmo nos períodos de maior atividade, o nó utiliza uma fração mínima de sua capacidade — forte indicador de superdimensionamento.

**Comportamento do HOSA:**

| Aspecto | Ação |
|---|---|
| **Nível de Resposta** | 0 (não há risco operacional imediato). |
| **Sinalização FinOps (crítica)** | Esta é a subclasse de maior impacto financeiro. O HOSA gera um **relatório de superdimensionamento** contendo: |
| | • $IPE$ calculado com dados históricos |
| | • Vetor de capacidade máxima utilizada vs. capacidade provisionada, por recurso |
| | • Sugestão de instância de menor porte compatível com a carga máxima observada (quando configurado com catálogo de instâncias do cloud provider) |
| | • Estimativa de economia anual projetada |
| **Exposição para Orquestrador** | Quando integrado a Kubernetes (Fase 2), o HOSA pode expor o nó com uma taint ou annotation indicando `hosa.io/structurally-idle=true`, permitindo que o cluster autoscaler considere o nó como candidato a descomissionamento. |
| **GreenOps** | Idêntico ao Regime −1, com a adição de que a **persistência** do estado ocioso é registrada como evidência para decisão de decommissioning. |

**Interação com FinOps e a Filosofia do HOSA:** O HOSA não toma a decisão de desligar ou redimensionar o nó autonomamente — isso excederia seu escopo de atuação local e poderia violar contratos de disponibilidade. Ele **fornece a evidência matemática** para que o humano ou o orquestrador tome a decisão informada. A economia financeira e energética é uma **consequência** da detecção precisa, não uma ação direta do agente.

**Interação com habituação:** **Permitida com ressalva.** O HOSA pode recalibrar o perfil basal para refletir a realidade operacional do nó (o basal real é baixo). Contudo, a **sinalização FinOps permanece ativa** — o HOSA se habitua à baixa demanda para fins de detecção de anomalia, mas continua reportando o superdimensionamento como informação de governança. A habituação elimina falsos positivos, mas não silencia a evidência de desperdício.

---

#### 6.4.4. Regime −3 — Silêncio Anômalo

**Definição:** Queda abrupta ou gradual de atividade **incompatível** com o contexto temporal esperado. O nó deveria estar ativo e não está. Exemplos:

- Tráfego redirecionado por sequestro de DNS;
- Falha silenciosa de load balancer que parou de enviar requisições;
- Processo de aplicação morto sem restart;
- Ataque que derrubou o serviço antes de instalar payload.

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | **Elevação abrupta** (mesmo sendo uma queda de carga — o desvio em relação ao basal é grande). |
| $\phi(t)$ | **Fortemente negativo**, com transição rápida. |
| $\frac{d\bar{D}_M}{dt}$ | **Pico positivo abrupto** (o $D_M$ está subindo rapidamente, embora o consumo esteja descendo). |
| $\frac{d\phi}{dt}$ | **Abrupto** — transição rápida para negativo. |
| $\rho(t)$ | **Potencialmente alto** — a queda pode ser desproporcional entre recursos. Se o tráfego de rede caiu a zero mas CPU permanece elevada, a estrutura de correlação está deformada (algo consome CPU sem receber requisições — processos residuais, malware, ou aplicação em loop infinito). |
| Contexto temporal | **Incoerente** — a queda ocorre em horário que deveria ser de atividade normal ou alta. |
| Processos ativos | Verificação de que os processos de aplicação esperados ainda estão em execução (via eBPF tracepoints em `sched_process_exit`). |

**Comportamento do HOSA:**

| Aspecto | Ação |
|---|---|
| **Nível de Resposta** | **1 (Vigilância) a 3 (Contenção Ativa)**, dependendo da velocidade e magnitude da queda, e da presença de indicadores de comprometimento. |
| **Investigação ativa** | O HOSA executa verificações suplementares quando detecta silêncio anômalo: |
| | • **Verificação de processos:** Os processos de aplicação esperados (configuráveis via safelist) ainda estão em execução? Houve `sched_process_exit` inesperado nos últimos segundos? |
| | • **Verificação de rede:** As interfaces de rede estão operacionais? Há conectividade de saída? (via tentativa de envio de webhook — se falhar, pode indicar isolamento de rede externo) |
| | • **Verificação de upstream:** Se o HOSA conhece os endpoints de upstream (load balancer, API gateway — configurável), pode executar health check reverso para verificar se o upstream está vivo e encaminhando tráfego. |
| **Sinalização urgente** | Webhook de prioridade alta para orquestrador e equipe operacional: "Nó X reporta atividade significativamente abaixo do esperado para o contexto temporal. Investigação recomendada." |
| **Correlação com ICP** | Se o silêncio anômalo é acompanhado de $ICP$ elevado (aumento de conexões de saída, processos anômalos), o cenário é reclassificado como potencial comprometimento (Regime +5 — Viral), com escalação correspondente. |

**O discriminante crítico entre Regime −1 e Regime −3:** A coerência temporal. Se o sistema opera com perfis basais sazonais (Seção 6.7), o HOSA compara a atividade observada com o perfil esperado para aquela janela temporal específica. Uma queda de atividade às 03:00 é coerente com o perfil de madrugada (Regime −1). Uma queda de atividade às 10:00 de uma terça-feira, quando o perfil prevê pico, é **incoerente** (Regime −3).

Quando perfis sazonais ainda não estão calibrados (primeiras semanas de operação), o HOSA utiliza o critério de **velocidade da transição**: uma queda abrupta ($|d\phi/dt|$ alto) é classificada provisoriamente como Regime −3, enquanto uma queda gradual é tratada como potencialmente legítima.

**O paradoxo do silêncio como alarme:** O Silêncio Anômalo é, contraintuitivamente, um dos cenários mais valiosos do HOSA. Monitores tradicionais são projetados para alertar sobre excesso. Quando um servidor para de receber tráfego e todas as métricas estão "verdes" (CPU baixa, memória livre, rede calma), o monitor tradicional reporta: "tudo saudável." O HOSA, por modelar o perfil basal esperado e não apenas os limites de capacidade, detecta que o silêncio é anômalo e sinaliza: "este nó deveria estar ativo e não está."

**Interação com habituação:** **Bloqueada.** O HOSA não se habitua a silêncio incoerente com o contexto temporal.

---

#### 6.4.5. Assinatura Matemática Consolidada — Semi-Eixo Negativo

| Indicador | Regime −1 (Legítima) | Regime −2 (Estrutural) | Regime −3 (Anômala) |
|---|---|---|---|
| $D_M(t)$ vs. basal global | Moderado | Baixo crônico | Alto (abrupto) |
| $D_M(t)$ vs. perfil temporal | **Baixo** (coerente) | Baixo em todas as janelas | **Alto** (incoerente) |
| $\phi(t)$ | Negativo moderado | Negativo persistente | **Fortemente negativo** |
| $\frac{d\phi}{dt}$ | Gradual | ≈ 0 (estável) | **Abrupto** |
| $\rho(t)$ | Baixo (correlações preservadas) | Baixo | Variável (possivelmente alto) |
| Coerência temporal | **Sim** | Irrelevante (sempre ocioso) | **Não** |
| $IPE$ | Variável | **Próximo de 1** | Irrelevante |

#### 6.4.6. Contribuição Teórica da Detecção de Sub-Demanda

A inclusão do semi-eixo negativo no espectro do HOSA introduz uma **simetria conceitual** ausente na literatura de detecção de anomalias para sistemas computacionais. A anomalia é redefinida como **desvio significativo do perfil basal em qualquer direção** — não apenas em direção ao excesso.

Esta simetria habilita três contribuições práticas que, até onde a revisão bibliográfica deste trabalho identifica, não são endereçadas por nenhum agente local existente de forma integrada:

**1. FinOps fundamentado em evidência endógena.** Ferramentas de otimização de custo em nuvem (AWS Cost Explorer, GCP Recommender, Kubecost) operam sobre dados de billing e métricas agregadas em intervalos de horas ou dias. O HOSA oferece evidência de subutilização com granularidade de segundos, incluindo correlação multivariável e contexto temporal, permitindo recomendações de right-sizing com maior precisão e confiança estatística.

**2. GreenOps como consequência da homeostase.** A otimização energética não é implementada como um módulo separado, mas como a **resposta natural do agente ao regime de sub-demanda** — exatamente como o metabolismo biológico reduz o consumo energético em repouso. A redução de frequência de CPU, o aumento de intervalos de amostragem e a redução de telemetria são ações do mesmo sistema de resposta graduada que aplica throttling em sobrecarga. A homeostase é bidirecional.

**3. Detecção de "blackout operacional" como capacidade de segurança.** O Silêncio Anômalo (Regime −3) é um cenário de segurança genuíno que monitores tradicionais de saúde de recurso são estruturalmente incapazes de detectar — precisamente porque todas as métricas de capacidade estão "saudáveis" quando o servidor para de receber trabalho. A detecção requer um modelo de "o que deveria estar acontecendo" (perfil basal contextualizado), não apenas "o que é perigoso" (limiares de capacidade).

---

### 6.5. Regime +1 — Alta Demanda Basal (Mudança Permanente de Patamar)

**Definição:** Uma elevação **persistente e não-revertida** no consumo de recursos, causada por mudanças legítimas na natureza do workload. Exemplos:

- Deploy de nova versão da aplicação com maior consumo de memória;
- Migração de microserviço adicional para o mesmo nó;
- Crescimento orgânico da base de usuários;
- Atualização de kernel ou runtime que altera o perfil de consumo.

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | **Elevação abrupta seguida de estabilização** em um novo platô. $D_M$ permanece acima de $\theta_1$ mas **constante**. |
| $\phi(t)$ | **Positivo**, estável após transitório. |
| $\frac{d\bar{D}_M}{dt}$ | Pico transitório no momento da mudança, seguido de **convergência para zero**. |
| $\frac{d^2\bar{D}_M}{dt^2}$ | Pico negativo após o transitório (desaceleração), seguido de **estabilização em zero**. |
| Matriz $\Sigma$ | **Preservação da estrutura de correlação.** As proporções entre variáveis permanecem similares — a "forma" da elipsoide de covariância é escalada, não deformada. CPU e memória continuam correlacionadas na mesma direção, apenas em magnitude maior. |

**O discriminante crítico:** A derivada convergindo a zero enquanto $D_M$ permanece elevado é a assinatura fundamental deste regime. Diferencia-se de um ataque em curso, onde a derivada permanece positiva ou acelera. Adicionalmente, a **preservação da estrutura de covariância** é um forte indicador de legitimidade: mudanças legítimas de workload tipicamente mantêm as proporções de consumo entre recursos, enquanto patologias introduzem correlações anômalas.

Para formalizar: o HOSA calcula a **razão de deformação** da matriz de covariância comparando a estrutura recente com a basal:

$$\rho(t) = \frac{\|\Sigma_{recente} - \Sigma_{basal}\|_F}{\|\Sigma_{basal}\|_F}$$

onde $\|\cdot\|_F$ denota a norma de Frobenius. Um $\rho$ baixo com $D_M$ alto indica mudança de patamar com preservação de estrutura (Regime +1). Um $\rho$ alto indica **deformação da estrutura de correlação** (potencialmente Regime +3 ou +4).

**Comportamento do HOSA:**

- **Fase transitória** (primeiros minutos): O HOSA pode atingir brevemente o Nível 1 ou 2 de resposta durante o pico transitório, aplicando vigilância ou contenção leve enquanto avalia a dinâmica.
- **Fase de confirmação:** Quando $\frac{d\bar{D}_M}{dt} \approx 0$ por um período configurável e $\rho(t) < \rho_{limiar}$, o HOSA classifica a situação como **mudança de patamar legítima** e aciona o mecanismo de habituação.

**Interação com habituação:** Este regime é o **caso de uso primário da habituação.** Quando os critérios de estabilidade e preservação de covariância são satisfeitos, o HOSA recalibra $\vec{\mu}$ e $\Sigma$ para refletir o novo regime operacional. O novo patamar torna-se o basal.

**Salvaguarda contra habituação prematura:** A habituação **não é acionada** se:
- A estabilização ocorre em patamar próximo ao limite de segurança física do recurso (e.g., memória > 90%). Estabilizar a 92% de memória não é um "novo normal" seguro — é um sistema no limite que perdeu margem de manobra.
- O SLM (Fase 3, quando disponível) identifica indicadores de comprometimento (processos desconhecidos, syscalls anômalas) simultâneos à elevação. Neste caso, a elevação pode ser Regime +3 (demanda disfarçada), não Regime +1.

---

### 6.6. Regime +2 — Alta Demanda Sazonal (Periodicidade Previsível)

**Definição:** Variações de demanda que seguem padrões temporais recorrentes, determinadas por ciclos previsíveis de uso. Exemplos:

- Pico de acessos diários entre 09:00 e 11:00 em aplicações corporativas;
- Queda de tráfego na madrugada;
- Picos semanais (segunda-feira em ERPs, sexta-feira em e-commerce);
- Sazonalidade mensal (fechamento contábil, folha de pagamento);
- Sazonalidade anual (Black Friday, campanhas de marketing).

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | **Oscilação periódica** com amplitude e frequência previsíveis. O $D_M$ sobe e desce seguindo o mesmo padrão em intervalos regulares. |
| $\phi(t)$ | **Oscila** entre positivo (picos) e negativo (vales), com periodicidade correspondente. |
| $\frac{d\bar{D}_M}{dt}$ | **Oscilação periódica correspondente**, com sinais alternados previsíveis. |
| Autocorrelação de $D_M$ | **Picos significativos** em lags correspondentes ao período sazonal (e.g., lag de 24h para ciclo diário, lag de 168h para ciclo semanal). |
| Matriz $\Sigma$ | **Variação periódica da covariância** — a estrutura de correlação pode mudar de forma previsível entre horários de pico e vale (e.g., durante o pico, a correlação CPU-rede aumenta porque mais requisições chegam). |

**O desafio fundamental:** Se o HOSA opera apenas com um perfil basal único ($\vec{\mu}$, $\Sigma$), os picos sazonais serão classificados como anomalias repetidamente. A habituação simples também falha, pois o sistema oscila entre patamares — habituar-se ao pico significaria perder sensibilidade durante o vale, e vice-versa.

**Solução: Perfis Basais Indexados por Contexto Temporal (Ritmo Circadiano Digital)**

O HOSA implementa um mecanismo de **segmentação temporal do perfil basal**. Em vez de manter um único par ($\vec{\mu}$, $\Sigma$), o agente mantém **N perfis basais** indexados por janela temporal:

$$\mathcal{B} = \{(\vec{\mu}_i, \Sigma_i, w_i) \mid i = 1, 2, \ldots, N\}$$

onde $w_i$ representa a janela temporal associada ao perfil $i$ (e.g., "segunda a sexta, 08:00–12:00" ou "sábado, 00:00–08:00").

A granularidade de segmentação é determinada automaticamente durante as primeiras semanas de operação através de **análise de autocorrelação** da série temporal de $D_M$:

1. O HOSA acumula a série $D_M(t)$ por um período mínimo de observação (configurável, padrão: 7 dias para detectar ciclo semanal);
2. Calcula a **função de autocorrelação (ACF)** da série;
3. Identifica os **lags com picos de autocorrelação significativos** (acima de um limiar de significância estatística);
4. Se detectar periodicidade (e.g., pico em lag de 24h), segmenta $\mathcal{B}$ automaticamente em janelas correspondentes;
5. Cada segmento acumula seu próprio perfil basal via Welford independente.

A partir da segmentação, o cálculo de $D_M$ em cada instante $t$ utiliza o perfil basal correspondente à janela temporal corrente:

$$D_M(t) = \sqrt{(\vec{x}(t) - \vec{\mu}_{i(t)})^T \Sigma_{i(t)}^{-1} (\vec{x}(t) - \vec{\mu}_{i(t)})}$$

onde $i(t)$ é o índice do perfil basal ativo para o instante $t$.

**Implicação prática:** O pico das 09:00 de segunda-feira é comparado com o perfil basal de "segunda-feira 08:00–12:00", não com o perfil de "domingo 03:00". Isso elimina falsos positivos sazonais sem sacrificar sensibilidade.

**Nota:** A segmentação temporal também beneficia o semi-eixo negativo. A fase de vale (e.g., madrugada) acumula seu próprio perfil basal, permitindo que o Regime −1 (ociosidade legítima) seja reconhecido como normalidade do segmento correspondente, e que o Regime −3 (silêncio anômalo) seja detectado precisamente quando a atividade cai abaixo do esperado para *aquela* janela.

**Comportamento do HOSA:**

- **Primeiras semanas (antes da segmentação):** Opera com perfil basal único. Pode gerar falsos positivos durante picos e vales sazonais. Aceita-se esta limitação como custo de cold start, mitigada pelo sistema de resposta graduada (o HOSA não executa mitigação agressiva em Nível 1).
- **Após segmentação:** Opera com perfis contextuais. Falsos positivos sazonais são eliminados.
- **Alostase antecipatória (Fase 4+):** Quando o enxame acumula dados suficientes de sazonalidade, o HOSA pode pré-posicionar recursos ou relaxar limiares **antes** do pico previsto, em lógica feed-forward.

**Interação com habituação:** A habituação ocorre **dentro de cada segmento temporal**, não globalmente. Se o pico das 09:00 aumenta permanentemente de magnitude (mais usuários orgânicos), o perfil de "08:00–12:00" é recalibrado, sem afetar o perfil da madrugada.

---

### 6.7. Regime +3 — Alta Demanda Disfarçada (Demanda Adversarial)

**Definição:** Consumo de recursos causado por atividade maliciosa que **deliberadamente mimetiza padrões de demanda legítima** para evadir detecção. Esta é a categoria de maior sofisticação adversarial e inclui:

- **DDoS de camada de aplicação (Layer 7):** Requisições HTTP sintaticamente válidas geradas por botnets que simulam navegação humana;
- **Cryptomining parasitário:** Processos que consomem CPU em níveis calculados para permanecer abaixo de limiares de alerta;
- **Exfiltração lenta de dados (Low-and-Slow):** Transferências de rede em volume baixo mas contínuo, diluídas ao longo de horas;
- **Ataques de esgotamento de recursos (Resource Exhaustion):** Abertura gradual de handles de arquivo, conexões de socket ou threads até atingir limites do sistema operacional.

**O problema central:** O adversário sofisticado conhece os limiares de detecção (ou assume que eles existem) e opera deliberadamente abaixo deles. Em um detector unidimensional de threshold, isso é trivial: basta manter cada métrica individual abaixo do limiar. Contra a Distância de Mahalanobis, a evasão é significativamente mais difícil — mas não impossível.

**Assinatura matemática — o que diferencia demanda disfarçada de demanda legítima:**

A tese central desta classificação é que, mesmo quando as **magnitudes** individuais são mantidas em faixa normal, a atividade maliciosa produz **deformação na estrutura de covariância** que a demanda legítima não produz. Isso ocorre porque a atividade maliciosa consome recursos de forma **desproporcional** em relação ao perfil de trabalho legítimo da máquina.

| Indicador | Demanda Legítima | Demanda Disfarçada |
|---|---|---|
| $D_M(t)$ | Pode estar em faixa normal ou elevada | Pode estar em faixa normal (evasão por magnitude) |
| $\phi(t)$ | Positivo, proporcional à carga | Positivo, mas com inconsistências entre dimensões |
| Razão de deformação $\rho(t)$ | Baixa — correlações preservadas | **Elevada** — correlações alteradas |
| Perfil de correlação CPU↔Rede | Proporcionais (mais rede → mais CPU de aplicação) | **Desproporcionais** (mais rede mas CPU de *kernel* aumenta, não CPU de *aplicação*; ou CPU aumenta sem aumento proporcional de I/O de disco para uma aplicação que deveria fazer leituras) |
| Entropia de syscalls | Estável — distribuição de chamadas de sistema consistente com o perfil da aplicação | **Alterada** — surgimento de syscalls atípicas (e.g., aumento de `connect()`, `sendto()` para exfiltração; aumento de `mmap()` para cryptomining) |
| Distribuição de fontes de conexão | Consistente com a base de usuários | **Concentração ou padrão atípico** (muitas conexões de poucos ASNs, ou distribuição uniforme artificial) |
| Razão trabalho/recurso | Proporcional (mais CPU → mais transações completadas) | **Desproporcional** (mais CPU → **sem** aumento correspondente em throughput de aplicação) |

**Métricas de Segundo Nível — Detecção de Deformação Estrutural:**

Além do $D_M$ e suas derivadas, o HOSA calcula indicadores específicos para detecção de deformação:

**a) Entropia de Shannon do perfil de syscalls:**

$$H(S, t) = -\sum_{i=1}^{k} p_i(t) \log_2 p_i(t)$$

onde $p_i(t)$ é a proporção da $i$-ésima syscall no intervalo $t$. Uma mudança significativa em $H$ sem mudança correspondente em métricas de aplicação (throughput, latência de resposta) é indicativa de atividade anômala.

O HOSA mantém um perfil basal de $H_{basal}$ e monitora:

$$\Delta H(t) = |H(S, t) - H_{basal}|$$

Um $\Delta H$ elevado é adicionado como **dimensão suplementar** ao vetor de estado $\vec{x}(t)$, influenciando diretamente o cálculo de $D_M$.

**b) Índice de Eficiência de Trabalho (Work Efficiency Index — WEI):**

$$WEI(t) = \frac{\text{throughput de aplicação}(t)}{\text{consumo de recurso computacional}(t)}$$

Em operação legítima, $WEI$ flutua em torno de uma média estável. Cryptomining e processamento parasitário consomem CPU/memória sem produzir throughput de aplicação, causando **queda do WEI** mesmo que nenhuma métrica individual esteja em faixa de alarme.

**c) Razão de Contexto Kernel/User:**

$$R_{ku}(t) = \frac{\text{CPU em modo kernel}(t)}{\text{CPU em modo user}(t)}$$

Ataques de rede (DDoS, scanning) e malware de rede produzem aumento de tempo em kernel space (processamento de pacotes, syscalls de rede) desproporcional ao tempo em user space. Flutuações significativas em $R_{ku}$ sem correlação com mudanças no workload legítimo são indicativas de atividade parasitária.

**Comportamento do HOSA:**

- **Detecção primária:** O HOSA não depende de assinaturas de ataques conhecidos (model of "known bad"). Opera sobre o princípio de **deformação estrutural do perfil basal** (model of "known good"). Qualquer atividade — conhecida ou inédita — que deforma a estrutura de covariância do sistema em relação ao perfil legítimo é detectada.
- **Classificação de confiança:** O HOSA não emite diagnósticos binários ("ataque" vs. "não-ataque"). Ele calcula um **vetor de indicadores** ($D_M$, $\rho$, $\Delta H$, $WEI$, $R_{ku}$) que, combinados, posicionam o estado atual em um espectro de confiança. A decisão de mitigação segue o sistema de resposta graduada (Seção 5.4).
- **Escalabilidade de resposta:** Se a deformação estrutural é detectada sem risco iminente de colapso (e.g., cryptominer consumindo 15% de CPU), o HOSA pode operar em Nível 1 (vigilância) ou Nível 2 (contenção leve), registrando evidências sem ação destrutiva. Se a deformação acelera, escala para níveis superiores.

**Interação com habituação:** A habituação é **bloqueada** quando a razão de deformação $\rho(t)$ excede o limiar de deformação. O HOSA **não se habitua a atividade que deforma a estrutura de covariância**, mesmo que estável em magnitude. Isso impede que um atacante persistente "treine" o detector a aceitar sua presença como normalidade.

Formalmente, a condição para habituação é ampliada:

$$\text{Habituação permitida} \iff \left(\frac{d\bar{D}_M}{dt} \approx 0\right) \wedge \left(\rho(t) < \rho_{limiar}\right) \wedge \left(\Delta H(t) < \Delta H_{limiar}\right)$$

Todas as três condições devem ser satisfeitas simultaneamente.

**Limitação reconhecida:** Um adversário que compreenda a arquitetura do HOSA e consiga executar atividade maliciosa que **preserva perfeitamente a estrutura de covariância e a distribuição de syscalls** evadirá a detecção. A análise de resistência adversarial formal (security games, limites teóricos de evasão) é um tema de pesquisa futura.

---

### 6.8. Regime +4 — Anomalia Não-Viral (Falha Localizada)

**Definição:** Deterioração de recursos causada por falha ou patologia **confinada ao nó local**, sem componente de propagação para outros sistemas. Exemplos:

- Memory leak em processo de aplicação;
- Degradação de disco (setores defeituosos, latência crescente de I/O);
- Bug de aplicação causando acumulação de file descriptors ou threads;
- Fork bomb acidental ou intencional;
- Deadlock de aplicação causando acumulação de requisições na fila;
- Degradação térmica de CPU (thermal throttling por hardware).

**Assinatura matemática:**

| Indicador | Comportamento |
|---|---|
| $D_M(t)$ | Elevação progressiva ou abrupta (dependendo da velocidade da falha). |
| $\phi(t)$ | **Positivo**, crescente. |
| $\frac{d\bar{D}_M}{dt}$ | **Positiva sustentada.** A anomalia não reverte espontaneamente. |
| $\frac{d^2\bar{D}_M}{dt^2}$ | Variável. Memory leak: $\approx 0$ (crescimento linear). Fork bomb: **positiva e crescente** (crescimento exponencial). |
| $\rho(t)$ | Pode ser alta (a falha altera as correlações — e.g., I/O stall causa CPU iowait desproporcional) ou baixa (memory leak uniforme que escala proporcionalmente). |
| $ICP(t)$ | **Baixo** — ausência de indicadores de propagação para outros nós. |
| Localização dimensional | **Uma ou duas dimensões dominantes** no desvio. O $D_M$ é "puxado" predominantemente por memória (leak), I/O (disco degradado), ou processos (fork bomb). |

**Decomposição de Contribuição Dimensional:**

Para diagnosticar **quais recursos** estão causando o desvio, o HOSA decompõe o $D_M$ em contribuições por dimensão. Dado o vetor de desvio $\vec{d} = \vec{x}(t) - \vec{\mu}$ e a métrica de Mahalanobis $D_M^2 = \vec{d}^T \Sigma^{-1} \vec{d}$, a contribuição da $j$-ésima dimensão é:

$$c_j = d_j \cdot (\Sigma^{-1} \vec{d})_j$$

As dimensões com maiores $c_j$ são os **contribuintes dominantes** da anomalia. Isso permite ao HOSA:

1. Direcionar ações de throttling para os processos que mais consomem o recurso contribuinte;
2. Registrar no log a **razão matemática** da decisão (auditabilidade);
3. Quando o SLM está disponível (Fase 3), fornecer contexto dimensional para diagnóstico causal.

**Comportamento do HOSA:**

- Escala o nível de resposta conforme a velocidade de convergência ao colapso;
- Identifica processos contribuintes via correlação entre consumo por cgroup e a dimensão dominante do desvio;
- Aplica throttling seletivo nos processos contribuintes, respeitando a safelist;
- Emite webhook com vetor de estado e contribuições dimensionais para permitir ação do orquestrador.

**Interação com habituação:** A habituação é **bloqueada quando a derivada permanece positiva sustentada**. Anomalias que crescem monotonicamente não são "novos normais" — são falhas progressivas. A habituação só é considerada após estabilização e confirmação de que o sistema permanece funcional no novo patamar (transição para Regime +1).

---

### 6.9. Regime +5 — Anomalia Viral (Propagação e Contágio)

**Definição:** Atividade maliciosa ou falha em cascata com componente de **propagação entre nós**, onde o nó afetado tenta comprometer, sobrecarregar ou infectar outros sistemas na rede. Exemplos:

- Worms e malware com capacidade de propagação lateral;
- Movimento lateral pós-comprometimento (pivot de atacante);
- Cascata de falhas em microserviços (um serviço degradado causa backpressure em dependentes upstream);
- Nó comprometido usado como base para DDoS interno (amplificação).

**Assinatura matemática — indicadores locais de comportamento viral:**

Mesmo operando apenas com dados locais (sem comunicação com outros nós), o HOSA pode detectar **indicadores de comportamento de propagação** no próprio nó:

| Indicador Local | Significado |
|---|---|
| **Explosão de conexões de saída** | Aumento abrupto de `connect()` para múltiplos IPs/portas que o nó normalmente não contata. Indicativo de scanning ou propagação. |
| **Diversidade de destinos de rede** | Entropia alta na distribuição de IPs de destino, inconsistente com o perfil de comunicação legítimo do nó (que tipicamente se comunica com um conjunto limitado e estável de peers). |
| **Processos gerando processos anômalos** | Árvore de processos com padrão de fork/exec atípico — processos de aplicação gerando shells (`/bin/sh`, `/bin/bash`), downloads (`curl`, `wget`), ou processos com nomes randomizados. |
| **Correlação temporal anomalia↔rede de saída** | A anomalia local coincide temporalmente com aumento de tráfego de saída. Em falhas não-virais (memory leak, disk degradation), o tráfego de saída tipicamente **diminui** (o nó falha em responder). Em propagação, o tráfego de saída **aumenta** paradoxalmente. |

**Métrica formal: Índice de Comportamento de Propagação (ICP)**

$$ICP(t) = w_1 \cdot \hat{C}_{out}(t) + w_2 \cdot \hat{H}_{dest}(t) + w_3 \cdot \hat{F}_{anom}(t) + w_4 \cdot \hat{\rho}_{D_M \leftrightarrow net_{out}}(t)$$

onde:
- $\hat{C}_{out}(t)$: taxa normalizada de novas conexões de saída (contra o perfil basal);
- $\hat{H}_{dest}(t)$: entropia normalizada dos IPs de destino;
- $\hat{F}_{anom}(t)$: taxa normalizada de forks/execs anômalos;
- $\hat{\rho}_{D_M \leftrightarrow net_{out}}(t)$: correlação entre $D_M$ e tráfego de saída (positiva = indicativo viral);
- $w_i$: pesos calibrados empiricamente.

O ICP é incorporado como dimensão suplementar ao vetor de estado, influenciando o cálculo de $D_M$ e, crucialmente, afetando a decisão entre **contenção** (Nível 3-4, preserva conectividade) e **quarentena** (Nível 5, isola o nó da rede).

**Comportamento do HOSA:**

- **ICP baixo + $D_M$ alto:** Anomalia não-viral (Regime +4). HOSA aplica contenção local sem isolamento de rede.
- **ICP alto + $D_M$ alto:** Forte indicação de propagação. HOSA prioriza **isolamento de rede** (Nível 4-5) para proteger o cluster, mesmo antes de esgotar as opções de contenção local.
- **ICP alto + $D_M$ moderado:** Propagação precoce — o nó está tentando se propagar mas ainda não está sob estresse severo (e.g., cryptominer com módulo de worm). HOSA aplica contenção seletiva dos processos contribuintes e restrição de conexões de saída via XDP.

**Relação com Fase 4 (Enxame):** No escopo das Fases 1-3, a detecção viral é baseada exclusivamente em indicadores locais. Na Fase 4, a confirmação cruzada entre nós (Quorum Sensing) permite validar se a anomalia detectada localmente é corroborada por anomalias simultâneas em outros nós, aumentando a confiança da classificação.

**Interação com habituação:** Habituação é **categoricamente bloqueada** quando $ICP > ICP_{limiar}$. O HOSA nunca se habitua a padrões de propagação.

---

### 6.10. Sinais Contextuais Exógenos como Dimensão Suplementar do Vetor de Estado

A detecção de anomalias baseada exclusivamente em métricas endógenas de recursos (CPU, memória, I/O, rede) é poderosa, mas pode ser enriquecida com **sinais contextuais** que informam o HOSA sobre o *porquê* esperado de variações de demanda. Esses sinais não violam o princípio de autonomia local, desde que sejam tratados como **dados de configuração carregados localmente**, não como dependências de runtime.

#### 6.10.1. Contexto Temporal (Endógeno)

O sinal contextual mais fundamental — e que não requer nenhuma dependência externa — é o **tempo**. O relógio do sistema fornece:

| Sinal | Formato | Uso |
|---|---|---|
| Hora do dia | Inteiro 0–23 ou codificação cíclica | Indexação do perfil basal sazonal (Seção 6.6) |
| Dia da semana | Inteiro 0–6 ou codificação cíclica | Distinção entre perfil weekday e weekend |
| Dia do mês | Inteiro 1–31 | Detecção de sazonalidade mensal (fechamento contábil) |
| Época do ano | Derivável da data | Sazonalidade anual (não detectável por autocorrelação em janelas curtas; requer configuração explícita ou acumulação de dados de longo prazo) |

**Codificação cíclica:** Para evitar descontinuidades (23h→0h, domingo→segunda), variáveis temporais são codificadas em componentes senoidais:

$$x_{hora,sin}(t) = \sin\left(\frac{2\pi \cdot hora(t)}{24}\right), \quad x_{hora,cos}(t) = \cos\left(\frac{2\pi \cdot hora(t)}{24}\right)$$

Essa codificação preserva a proximidade entre 23h e 0h no espaço numérico e pode ser incluída como dimensões do vetor de estado $\vec{x}(t)$, permitindo que a Matriz de Covariância capture correlações entre recursos e posição no ciclo temporal.

#### 6.10.2. Contexto Ambiental (IoT e Edge)

Em cenários de Edge Computing e IoT industrial, o nó computacional pode estar sujeito a condições ambientais que afetam diretamente o comportamento do hardware e do workload:

| Sinal | Fonte | Impacto |
|---|---|---|
| **Temperatura ambiente** | Sensores locais (I²C, GPIO) | Temperaturas extremas causam thermal throttling de CPU, degradação de baterias, e alteração de performance de armazenamento SSD. Um aumento de CPU load que correlaciona com aumento de temperatura ambiente é provavelmente thermal throttling, não ataque. |
| **Umidade** | Sensores locais | Em ambientes industriais, umidade alta correlaciona com falhas de conectividade em interfaces de rede sem fio. |
| **Tensão de alimentação** | Sensores de power management (ACPI, PMBus) | Flutuações de tensão afetam estabilidade de clock e podem causar comportamento errático de hardware que mimetiza anomalia de software. |
| **Vibração/aceleração** | Acelerômetros (comum em IoT industrial) | Vibrações mecânicas podem causar erros de leitura em discos rotativos (HDD) e desconexões intermitentes de cabos. |

Quando disponíveis, esses sinais são incorporados ao vetor de estado $\vec{x}(t)$ como dimensões suplementares. A Matriz de Covariância captura automaticamente a correlação entre condições ambientais e métricas de recurso, permitindo ao HOSA **descontar** variações de performance que são causadas por fatores físicos do ambiente e não por patologias de software.

**Princípio de design: degradação graciosa.** Se nenhum sensor ambiental estiver disponível, o HOSA opera sem essas dimensões. O vetor de estado é menor, a detecção é funcional mas menos contextualizada. A presença de sensores **melhora** a classificação; sua ausência **não impede** o funcionamento.

#### 6.10.3. Contexto Operacional (Carregado por Configuração)

Certos sinais contextuais não podem ser derivados de sensores, mas podem ser fornecidos pelo operador como **metadados de configuração** carregados no momento do deploy do HOSA:

| Sinal | Formato | Uso |
|---|---|---|
| **Calendário de eventos** | Lista de datas/horários com labels (e.g., "black_friday", "maintenance_window") | Permite ao HOSA **relaxar limiares preemptivamente** durante eventos esperados de alta demanda. Reduz falsos positivos durante picos planejados. |
| **Perfil de workload** | Label descritivo (e.g., "web_server", "database", "batch_processing", "iot_gateway") | Permite calibração de pesos relativos no vetor de estado. Um servidor de banco de dados tem perfil de I/O dominante; um web server tem perfil de rede dominante. |
| **Zona geográfica** | Label (e.g., "us-east-1", "factory-floor-B", "offshore-rig-3") | Utilizada em Fase 4+ para contextualizar comunicação entre nós do enxame. Não afeta o funcionamento local nas Fases 1-3, mas é armazenada para uso futuro. |
| **Fusos horários de clientes** | Lista de fusos horários predominantes da base de usuários | Refina a segmentação temporal do perfil basal quando os usuários estão em fusos horários diferentes do servidor. |

**Esses sinais são estritamente opcionais.** O HOSA funciona integralmente sem nenhum deles. Eles representam uma camada de **otimização de precisão**, não uma dependência funcional.

---

### 6.11. Síntese: Matriz de Classificação Integrada

A tabela abaixo sintetiza os indicadores discriminantes para cada regime ao longo do espectro, fornecendo o mapa de decisão utilizado pelo HOSA:

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

**Nota sobre classificação ambígua:** Em cenários onde os indicadores não apontam inequivocamente para um único regime, o HOSA adota o **princípio da precaução**: classifica temporariamente como o regime de maior severidade (maior $|índice|$) compatível com os dados observados e aplica a resposta correspondente. A classificação é revisada continuamente à medida que novos dados acumulam evidência. O log de auditoria registra a ambiguidade e os indicadores que levaram à decisão, garantindo transparência para análise posterior.

**Nota sobre transições entre semi-eixos:** O Regime −3 (Silêncio Anômalo) pode transicionar para o semi-eixo positivo quando a investigação revela que o silêncio é acompanhado de indicadores de comprometimento ($ICP$ elevado, processos anômalos). Neste caso, o estado é reclassificado diretamente como Regime +5 (Propagação Viral). A travessia do ponto zero sem parada em homeostase é registrada como evento de alta prioridade.

---

### 6.12. Implicações para o Mecanismo de Habituação: Regras Consolidadas

A interação entre regimes e habituação é suficientemente crítica para merecer formalização explícita das regras que governam quando a recalibração do perfil basal é permitida, bloqueada ou condicional.

**Pré-condições necessárias (todas devem ser satisfeitas simultaneamente):**

$$\text{Habituação} \iff \begin{cases} \left|\frac{d\bar{D}_M}{dt}\right| < \epsilon_d & \text{(estabilização)} \\ \rho(t) < \rho_{limiar} & \text{(covariância preservada)} \\ \Delta H(t) < \Delta H_{limiar} & \text{(syscalls estáveis)} \\ ICP(t) < ICP_{limiar} & \text{(sem propagação)} \\ D_M(t) < D_{M,segurança} & \text{(patamar seguro)} \\ t_{estável} > T_{min} & \text{(estabilização sustentada)} \\ \text{coerência temporal de } \phi(t) & \text{(se } \phi < 0 \text{, coerente com perfil sazonal)} \end{cases}$$

onde:
- $\epsilon_d$ é o limiar de quase-estacionariedade da derivada;
- $T_{min}$ é o tempo mínimo de estabilização antes de aceitar habituação (padrão: 30 minutos);
- $D_{M,segurança}$ é o limiar máximo de $D_M$ que é considerado operacionalmente seguro para habituação (impede habituação a estados próximos ao colapso).

**Regimes e habituação ao longo do espectro:**

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

**Padrão visual:** A habituação é permitida nos regimes centrais do espectro (−2 a +2), onde os desvios são legítimos ou estruturais. É bloqueada nas extremidades (−3, +3 a +5), onde os desvios são patológicos ou adversariais. Esta simetria reflete o princípio de que o HOSA se adapta à variação legítima mas recusa normalizar a patologia.

---

### 6.13. Resumo das Métricas Suplementares

Esta seção introduziu métricas além da Distância de Mahalanobis e suas derivadas que são necessárias para a classificação completa de regimes e o posicionamento no espectro. Para referência consolidada:

| Métrica | Símbolo | Definição | Seção |
|---|---|---|---|
| Índice de Direção de Carga | $\phi(t)$ | Projeção normalizada ponderada do desvio sobre o eixo de carga — indica se o desvio é em direção a sobrecarga ($\phi > 0$) ou ociosidade ($\phi < 0$). Determina o semi-eixo do espectro. | 6.2.3 |
| Índice de Provisionamento Excedente | $IPE$ | Razão entre capacidade provisionada e utilização máxima histórica — quantifica superdimensionamento | 6.4.3 |
| Razão de Deformação de Covariância | $\rho(t)$ | Norma de Frobenius da diferença entre covariância recente e basal, normalizada | 6.5 |
| Entropia de Shannon do perfil de syscalls | $H(S, t)$ e $\Delta H(t)$ | Medida de diversidade e mudança na distribuição de chamadas de sistema | 6.7 |
| Índice de Eficiência de Trabalho | $WEI(t)$ | Razão throughput de aplicação / consumo de recurso | 6.7 |
| Razão Kernel/User | $R_{ku}(t)$ | Proporção de tempo de CPU em kernel space vs. user space | 6.7 |
| Índice de Comportamento de Propagação | $ICP(t)$ | Combinação ponderada de indicadores de atividade viral | 6.9 |
| Contribuição Dimensional | $c_j$ | Decomposição de $D_M^2$ por dimensão do vetor de estado | 6.8 |
| Autocorrelação de $D_M$ | $ACF_{D_M}(\tau)$ | Função de autocorrelação para detecção de periodicidade | 6.6 |

Todas estas métricas são calculadas em user space pelo motor matemático. As métricas que dependem de contadores de kernel (syscalls, context switches, conexões de rede) são coletadas via sondas eBPF existentes. Não há dependência de ferramentas ou bibliotecas externas ao agente.

---

### 6.14. Contribuição Teórica da Taxonomia

A taxonomia de regimes operacionais proposta nesta seção contribui para a literatura de detecção de anomalias em sistemas computacionais ao formalizar duas distinções frequentemente tratadas de forma ad hoc na prática operacional:

**1. Nem todo desvio do basal é uma anomalia, e nem toda anomalia é uma ameaça.** A organização espectral bipolar permite respostas proporcionais à posição no espectro: regimes centrais (−2 a +2) são tratados com adaptação e otimização; regimes extremos (−3, +3 a +5) são tratados com contenção e isolamento.

**2. A anomalia por déficit é tão significativa quanto a anomalia por excesso.** A simetria do espectro em torno do Regime 0 estabelece que o HOSA implementa homeostase genuína — equilíbrio bidirecional — e não apenas proteção contra sobrecarga. O semi-eixo negativo habilita FinOps endógeno, GreenOps como consequência natural e detecção de blackout operacional, capacidades ausentes em agentes locais existentes.

A combinação de análise de magnitude ($D_M$), direção ($\phi$), dinâmica temporal (derivadas), integridade estrutural ($\rho$), perfil comportamental ($\Delta H$, $WEI$, $R_{ku}$) e intenção de propagação ($ICP$) fornece um framework de classificação que permite respostas proporcionais e informadas, reduzindo simultaneamente falsos positivos (habituação quando apropriado) e falsos negativos (bloqueio de habituação quando indicadores de deformação ou propagação estão presentes).

A validação experimental desta taxonomia — incluindo a calibração dos limiares $\rho_{limiar}$, $\Delta H_{limiar}$, $ICP_{limiar}$ e $D_{M,segurança}$ contra datasets reais de produção e cenários de ataque controlados — é um dos objetivos centrais da fase experimental da dissertação.

---

## 7. Escolha de Linguagem: Análise de Trade-offs

A escolha de linguagem de implementação para o motor matemático (user space) é uma decisão arquitetural com implicações mensuráveis em latência, previsibilidade e velocidade de desenvolvimento.

| Critério | Go | Rust | C |
|---|---|---|---|
| **Latência de GC** | GC com pausas sub-ms (Go 1.22+), mas não-determinísticas. Mitigável com `sync.Pool`, pré-alocação e tuning de `GOGC`. | Sem GC. Latência determinística. | Sem GC. Latência determinística. |
| **Ecossistema eBPF** | `cilium/ebpf` (biblioteca madura e ativa). | `aya-rs` (biblioteca ativa, ecossistema menor). | `libbpf` (referência upstream do kernel). |
| **Velocidade de desenvolvimento** | Alta. Compilação rápida. Concorrência nativa (goroutines). | Média. Borrow checker exige disciplina. Compilação lenta. | Baixa. Gerência manual de memória. |
| **Segurança de memória** | Garantida pelo runtime. | Garantida pelo compilador (sem runtime). | Responsabilidade do programador. |
| **Adequação acadêmica** | Código legível, facilita reprodutibilidade. | Código legível com curva de aprendizado. | Propenso a bugs sutis. |

**Decisão provisória:** Go para o motor matemático e plano de controle, com o hot path de cálculo implementado com alocação mínima (pré-alocação de slices, `sync.Pool`, `GOGC=off` durante ciclos críticos). A justificativa é pragmática: para o escopo de uma dissertação de mestrado, a velocidade de iteração do Go permite maior foco na validação da tese matemática.

**Compromisso de validação:** A dissertação incluirá benchmarks comparativos do hot path (cálculo de $D_M$, derivadas, decisão) medindo latência p50/p99 e jitter, com discussão explícita sobre se as pausas de GC observadas impactam a janela de detecção em cenários de colapso real. Se as pausas de GC se mostrarem problemáticas nos benchmarks, a migração do hot path para C (via CGo ou processo auxiliar) será documentada como trabalho futuro.

---

## 8. Roadmap: Horizonte Executável e Visão de Longo Prazo

### 8.1. Horizonte Executável (Escopo da Dissertação e Continuidade Imediata)

#### Fase 1: Fundação — O Motor Matemático e o Arco Reflexo (v1.0)

**Escopo:** Implementação completa do ciclo perceptivo-motor.

**Entregas:**
- Sondas eBPF para coleta de vetor de estado (CPU, memória, I/O, rede, scheduler) via tracepoints e kprobes
- Motor matemático com Welford incremental, Mahalanobis, EWMA e derivadas
- Propriocepção de hardware (warm-up com calibração automática)
- Sistema de resposta graduada (Níveis 0–4)
- Filtro Talâmico: supressão de telemetria redundante em homeostase (heartbeat mínimo)
- Benchmark de latência do ciclo completo (detecção → decisão → atuação)

**Validação experimental:**
- Injeção de falhas controladas: memory leak gradual, fork bomb, CPU burn, flood de rede
- Comparação quantitativa: tempo de detecção e mitigação do HOSA vs. Prometheus+Alertmanager vs. systemd-oomd
- Análise de sensibilidade do parâmetro $\alpha$ (EWMA) e dos limiares adaptativos
- Medição de overhead do agente (CPU, memória, latência adicionada ao sistema)

#### Fase 2: Simbiose com Ecossistema (v2.0)

**Escopo:** Integração oportunista com orquestradores e sistemas de monitoramento.

**Entregas:**
- Webhooks para K8s HPA/KEDA: disparo de scale-up preemptivo baseado na derivada de $D_M$
- Exposição de métricas HOSA em formato compatível com Prometheus (para integração com dashboards existentes)
- Endpoint de `/healthz` enriquecido: ao invés de binário (healthy/unhealthy), retorna vetor de estado normalizado
- Sistema Endócrino Digital: métricas de "fadigabilidade" de longo prazo (desgaste térmico, ciclos de escrita em SSD) expostas como labels para o scheduler do Kubernetes

#### Fase 3: Triagem Semântica Local (v3.0)

**Escopo:** Introdução de análise causal pós-contenção.

**Entregas:**
- Small Language Model (SLM) executando localmente, ativado **apenas** após contenção de Nível 3+ para diagnosticar causa raiz provável
- Modelo operando **air-gapped** (sem conexão à internet)
- Células T de Memória: assinaturas de padrões de ataque armazenadas em Bloom Filter eBPF para bloqueio em nanossegundos em caso de recorrência
- Quarentena Autônoma (Nível 5): isolamento de rede controlado (substituindo o conceito anterior de kernel panic)
- Habituação Neural: recalibração automática do perfil basal quando mudanças de workload são classificadas como benignas pelo SLM

**Nota sobre footprint:** O SLM é um componente **condicional**, ativado apenas em nós com recursos suficientes (mínimo recomendado: 4GB RAM disponível). Em dispositivos com recursos limitados (IoT, Edge de baixa capacidade), a Fase 3 não é implantada, e o HOSA opera exclusivamente com o motor matemático das Fases 1-2. A arquitetura é projetada para degradação graciosa: funcionalidade completa em hardware capaz, funcionalidade essencial em hardware limitado.

### 8.2. Visão de Longo Prazo (Escopo de Doutorado e Pesquisa Futura)

As fases a seguir representam direções de pesquisa que dependem da validação empírica das Fases 1-3 e de avanços no estado da arte de seus respectivos campos. São documentadas aqui como **horizonte de investigação**, não como compromissos de engenharia.

#### Fase 4: Inteligência de Enxame (v4.0) — *Pesquisa Futura*

**Hipótese de pesquisa:** Nós equipados com HOSA podem estabelecer consenso local sobre o estado do cluster via comunicação P2P leve, reduzindo a dependência do control plane para decisões de saúde coletiva.

**Desafios técnicos reconhecidos:**
- Consenso distribuído é um problema com décadas de pesquisa (Lamport, 1998; Ongaro & Ousterhout, 2014). A proposta não é reinventar Paxos/Raft, mas investigar se o escopo limitado da decisão (confirmação coletiva de anomalia, não consenso de estado geral) permite protocolos mais leves.
- Sazonalidade aprendida (alostase): aplicação antecipatória de recursos baseada em padrões temporais observados.

#### Fase 5: Aprendizado Federado e Imunidade Coletiva (v5.0) — *Pesquisa Futura*

**Hipótese de pesquisa:** Atualizações de pesos matemáticos (não dados sensíveis) compartilhadas entre instâncias HOSA podem criar imunidade coletiva contra padrões de ataque emergentes.

**Desafios técnicos reconhecidos:**
- Convergência de aprendizado federado em ambientes heterogêneos (Li et al., 2020)
- Resistência a model poisoning attacks
- Privacidade diferencial (Dwork & Roth, 2014)

#### Fase 6: Offload para Hardware Dedicado (v6.0) — *Pesquisa Futura*

**Hipótese de pesquisa:** A migração do ciclo perceptivo-motor para hardware dedicado (SmartNIC/DPU) elimina a competição por CPU com as aplicações do nó e permite operação em estados de baixo consumo energético.

**Desafios técnicos reconhecidos:**
- SmartNICs e DPUs são hardware especializado com custo significativo, potencialmente contradizendo a premissa de ubiquidade de hardware.
- A programação de SmartNICs (P4, eBPF offloaded) possui limitações de complexidade computacional.

#### Fase 7: eSRE — Formalização Metodológica (v7.0) — *Pesquisa Futura*

**Objetivo:** Consolidação dos princípios do HOSA em uma metodologia aberta denominada **eSRE (Endogenous Site Reliability Engineering)**, documentando as "Leis de Sobrevivência Celular" como práticas recomendadas para design de sistemas resilientes.

**Dependência:** Adoção e validação empírica em ambientes de produção diversos. Este é um objetivo de disseminação, não de engenharia.

---

## 9. Limitações Conhecidas e Fronteiras do Trabalho

A honestidade intelectual exige a documentação explícita das limitações conhecidas:

1. **Pressuposto de distribuição.** A Distância de Mahalanobis assume implicitamente que o perfil basal segue uma distribuição aproximadamente elipsoidal (multivariável normal). Workloads com distribuições multimodais (e.g., sistema que alterna entre dois regimes operacionais distintos) podem violar este pressuposto. A dissertação investigará a robustez do detector sob distribuições não-gaussianas e, se necessário, avaliará alternativas como Minimum Covariance Determinant (MCD) ou detecção de anomalias baseada em Local Outlier Factor (LOF).

2. **Cold start.** Durante a fase de warm-up (primeiros minutos após inicialização), o agente não possui perfil basal suficiente para detecção confiável. Neste intervalo, o HOSA opera em modo conservador (apenas logging, sem mitigação), constituindo uma janela de vulnerabilidade.

3. **Evasão adversária.** Um atacante com conhecimento da arquitetura do HOSA poderia, teoricamente, executar um ataque "low-and-slow" que mantém $D_M$ e suas derivadas abaixo dos limiares de detecção. A análise de resistência a evasão adversária é um tema de pesquisa futura (Fase 5).

4. **Custos do throttling.** Conforme detalhado na Seção 5.6, o throttling pode introduzir efeitos colaterais. A eficácia do mecanismo de safelist e da seleção de processos-alvo será validada experimentalmente.

5. **Escopo do sistema operacional.** O HOSA é projetado exclusivamente para o kernel Linux (≥ 5.8, com suporte a eBPF CO-RE). Portabilidade para outros kernels não é um objetivo.

6. **Interação com NUMA e heterogeneidade de hardware.** Sistemas com topologia NUMA complexa (múltiplos sockets, memória heterogênea) podem exibir padrões de pressão localizados que o vetor de estado agregado não captura. A granularidade per-NUMA-node do vetor de estado será investigada.

---

## 10. Contribuições Esperadas

Este trabalho propõe as seguintes contribuições ao estado da arte:

1. **Formalização do conceito de Resiliência Endógena** como paradigma complementar à observabilidade exógena, com definição precisa dos limites operacionais de cada abordagem.

2. **Modelo de detecção de anomalias multivariável em tempo real** baseado em Mahalanobis com atualização incremental e análise de taxa de variação, validado contra cenários de colapso reais e sintéticos.

3. **Arquitetura de referência** para agentes de mitigação autônoma com atuação em kernel space, documentando os trade-offs de design (latência vs. estabilidade, autonomia vs. risco de mitigação).

4. **Análise comparativa quantitativa** do tempo de detecção e mitigação entre o modelo endógeno (HOSA) e o modelo exógeno (Prometheus + Alertmanager + orquestrador), contribuindo dados empíricos para um debate que tem sido predominantemente teórico.

5. **Framework de resposta graduada** para mitigação autônoma, com documentação explícita de riscos e mecanismos de proteção (safelist, histerese, quarentena vs. destruição).

---

## 11. Referências Bibliográficas

Bear, M. F., Connors, B. W., & Paradiso, M. A. (2015). *Neuroscience: Exploring the Brain* (4th ed.). Wolters Kluwer.

Beyer, B., Jones, C., Petoff, J., & Murphy, N. R. (2016). *Site Reliability Engineering: How Google Runs Production Systems*. O'Reilly Media.

Brewer, E. A. (2000). Towards robust distributed systems. *Proceedings of the 19th Annual ACM Symposium on Principles of Distributed Computing (PODC)*.

Burns, B., Grant, B., Oppenheimer, D., Brewer, E., & Wilkes, J. (2016). Borg, Omega, and Kubernetes. *ACM Queue*, 14(1), 70–93.

Dwork, C., & Roth, A. (2014). The Algorithmic Foundations of Differential Privacy. *Foundations and Trends in Theoretical Computer Science*, 9(3–4), 211–407.

Forrest, S., Hofmeyr, S. A., & Somayaji, A. (1997). Computer immunology. *Communications of the ACM*, 40(10), 88–96.

Gregg, B. (2019). *BPF Performance Tools: Linux System and Application Observability*. Addison-Wesley Professional.

Hellerstein, J. L., Diao, Y., Parekh, S., & Tilbury, D. M. (2004). *Feedback Control of Computing Systems*. John Wiley & Sons.

Heo, T. (2015). Control Group v2. *Linux Kernel Documentation*. https://www.kernel.org/doc/Documentation/cgroup-v2.txt

Horn, P. (2001). Autonomic Computing: IBM's Perspective on the State of Information Technology. *IBM Corporation*.

Lamport, L. (1998). The Part-Time Parliament. *ACM Transactions on Computer Systems*, 16(2), 133–169.

Li, T., Sahu, A. K., Talwalkar, A., & Smith, V. (2020). Federated Learning: Challenges, Methods, and Future Directions. *IEEE Signal Processing Magazine*, 37(3), 50–60.

Mahalanobis, P. C. (1936). On the generalized distance in statistics. *Proceedings of the National Institute of Sciences of India*, 2(1), 49–55.

Ongaro, D., & Ousterhout, J. (2014). In Search of an Understandable Consensus Algorithm. *USENIX Annual Technical Conference (ATC)*.

Tang, C., et al. (2020). FBAR: Facebook's Automated Remediation System. *Proceedings of the ACM Symposium on Cloud Computing (SoCC)*.

Weiner, J. (2018). PSI - Pressure Stall Information. *Linux Kernel Documentation*. https://www.kernel.org/doc/html/latest/accounting/psi.html

Welford, B. P. (1962). Note on a Method for Calculating Corrected Sums of Squares and Products. *Technometrics*, 4(3), 419–420.

---

*Fim do Whitepaper — Versão 2.0*