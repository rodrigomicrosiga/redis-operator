# 🚀 Redis Sentinel Operator

## 📌 Introdução
O **Redis Sentinel Operator** é um controlador Kubernetes customizado (desenvolvido em Go com Kubebuilder) focado em orquestrar a implantação, configuração e resiliência de ambientes Redis em modo de Alta Disponibilidade (HA). 

Diferente de aplicações Stateless, bancos de dados em memória exigem tratamento rigoroso de rede e identidade. Este Operator atua como um SysAdmin automatizado, garantindo o provisionamento de recursos *Stateful*, injeção de configurações de replicação e a inicialização de um quórum de vigilância (Sentinel) capaz de realizar failover automático.

---

## 🧠 Decisão Arquitetural: Sentinel vs. Cluster

O desenvolvimento deste Operator passou por uma análise fundamental sobre a topologia do Redis. Existem duas estratégias primárias para escalar o Redis no Kubernetes:

### Opção A: Redis Sentinel (Alta Disponibilidade Clássica)
* **Arquitetura:** Topologia `Master-Replica` com processos `Sentinel` atuando como monitores de quórum.
* **Escrita/Leitura:** 100% das gravações vão para o Master. As leituras podem ser distribuídas entre as réplicas.
* **Failover:** Se o Master falha, o quórum Sentinel elege e promove uma réplica.
* **Casos de Uso:** Datasets que cabem na memória de um único nó, mas exigem alta disponibilidade (HA) contra falhas de infraestrutura.

### Opção B: Redis Cluster (Sharding e Escala Horizontal)
* **Arquitetura:** Múltiplos nós `Master`, cada um responsável por um subconjunto dos dados (*Hash Slots*), com suas respectivas réplicas.
* **Escrita/Leitura:** O tráfego de gravação é distribuído por todos os Masters da malha. O failover é nativo, sem necessidade de processos Sentinels adicionais.
* **Casos de Uso:** Datasets massivos que excedem a capacidade de RAM de um único nó e exigem alto throughput de gravação (Writes).

Neste projeto, optei por implementar o **Redis Sentinel**, pois o foco principal é dominar a orquestração de processos *Stateful* inter-relacionados no Kubernetes. A construção do Sentinel exige a gestão precisa de identidades fixas (StatefulSets), redes sem balanceamento (Headless Services) e sincronismo na injeção de configurações (`redis.conf` e `sentinel.conf`) para garantir a descoberta dos nós.

---

## 🏗️ Arquitetura (Fluxo de Orquestração)

```mermaid
graph TD
    User([Engenheiro SRE]) -->|Aplica CRD| K8s(Kubernetes API)
    K8s --> Operator{Redis Operator}
    
    Operator -->|Cria StatefulSet| Master[(Redis Master<br/>redis-0)]
    Operator -->|Cria StatefulSet| Rep1[(Redis Replica<br/>redis-1)]
    Operator -->|Cria StatefulSet| Rep2[(Redis Replica<br/>redis-2)]
    
    Rep1 -.->|Sincroniza Dados| Master
    Rep2 -.->|Sincroniza Dados| Master
    
    Operator -->|Cria Deployment| Sent1[Sentinel 1]
    Operator -->|Cria Deployment| Sent2[Sentinel 2]
    Operator -->|Cria Deployment| Sent3[Sentinel 3]
    
    Sent1 -.->|Monitora Quórum| Master
    Sent2 -.->|Monitora Quórum| Master
    Sent3 -.->|Monitora Quórum| Master
```
## 📖 Diário de Desenvolvimento

<details open>
<summary><strong>Clique para expandir o registro das atividades</strong></summary>

* **[08/06/2026] - Setup Inicial e Decisão Arquitetural:**
  * Inicialização do módulo Go e scaffolding do Kubebuilder (`cloud104.io/v1alpha1`).
  * Estudo comparativo arquitetural: Redis Sentinel vs. Redis Cluster.
  * Escolha da topologia Sentinel focada no aprofundamento do tratamento de `StatefulSets` e *Leader Election* manual.
  * Elaboração do manifesto do projeto (ADR) e mapeamento visual em Mermaid.

* **[08/06/2026] - Construção do Núcleo Stateful e Inteligência de Failover:**
  * Criei o design pattern `Factory` (`factory.go`) para isolar a lógica de geração dos recursos físicos.
  * Implementei o `Headless Service` para garantir a identidade DNS fixa de cada nó do banco.
  * Desenvolvi a lógica dos Vigias (Sentinels) via `Deployment`, calculando o quórum de votos dinamicamente com base nas especificações do CRD.
  * Orquestrei o loop completo de reconciliação no controlador para atualizar o status do CRD para `Ready` de forma automatizada.

* **[08/06/2026] - Resiliência POSIX e Validação em Ambiente Local (Tilt):**
  * **Troubleshooting:** Identifiquei e corrigi um comportamento específico do `grep` do BusyBox/Alpine que forçava o nó Master a entrar em `CrashLoopBackOff`.
  * **Solução:** Substituí a validação do hostname por uma estrutura `case` nativa POSIX Shell, tornando o script de inicialização do container 100% agnóstico e imune a variações de ferramentas do sistema operacional.
  * Corrigi o mapeamento do arquivo de configuração do Redis para o diretório correto de montagem do volume (`/config/redis.conf`).
  * Validei com sucesso a subida simultânea e ordenada de toda a infraestrutura: 1 Master, 2 Replicas e 3 Sentinels em perfeita harmonia.

</details>