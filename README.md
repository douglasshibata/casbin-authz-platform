# 📋 Casbin Authorization Platform

[![Tech Stack](https://img.shields.io/badge/Stack-Go%201.25%20%7C%20Casbin%20v2%20%7C%20Gin%20%7C%20PostgreSQL%20%7C%20Redis-blue)](#)
[![Status](https://img.shields.io/badge/Status-Active-brightgreen)](#)
[![Security](https://img.shields.io/badge/Security-RBAC%20%26%20ABAC-orange)](#)

O **Casbin Authorization Platform** é um motor centralizado de autorização e controle de acesso (IAM) de nível enterprise projetado em **Go** utilizando **Apache Casbin v2 (SyncedEnforcer)**, **Go Gin**, **PostgreSQL (GORM Adapter)** e **Redis (Watcher)**. O sistema resolve a complexidade de gerenciar permissões em ecossistemas de microserviços distribuídos e multi-tenant, desacoplando completamente as políticas de segurança do código de aplicação.

A plataforma implementa dois modelos complementares de autorização: controle de acesso baseado em papéis por domínios/inquilinos (**Domain-Based Multi-Tenancy RBAC**) com agrupamento de papéis (`g`) e recursos hierárquicos (`g2`), e controle de acesso baseado em atributos dinâmicos (**Dynamic Attribute ABAC**). A persistência é gerenciada em tempo real por adaptadores relacionais e a sincronização de políticas entre réplicas ativas é realizada por canais pub/sub distribuídos via Redis com heartbeat periódico de salvaguarda, garantindo latência de consulta sub-milissegundo para os interceptores de segurança.

---

## 📐 1. Arquitetura & Componentes Internos

### Árvore de Diretórios Reestruturada (Standard Go Project Layout)
```text
casbin-authz-platform/
├── cmd/
│   └── api/
│       └── main.go                     # Entrypoint HTTP, lifecycle e graceful shutdown
├── config/
│   ├── config.go                       # Parsing de envs (cleanenv/viper)
│   ├── model_rbac_tenants.conf         # Modelo PERM RBAC com domínios
│   └── model_abac.conf                 # Modelo PERM ABAC
├── internal/
│   ├── domain/
│   │   ├── authz.go                    # Entidades e interfaces do domínio
│   │   └── errors.go                   # Erros tipados de autorização
│   ├── engine/
│   │   ├── enforcer.go                 # SyncedEnforcer, migrations e hooks
│   │   └── watcher.go                  # Redis watcher com auto-reconnect
│   ├── handler/
│   │   └── http/
│   │       ├── authz_handler.go        # Endpoints /enforce, /batch-enforce
│   │       └── policy_handler.go       # Endpoints CRUD de regras com RBAC
│   └── middleware/
│       ├── casbin_mw.go                # Middleware de interceptação Gin
│       └── tracer.go                   # Propagação de trace_id/request_id
├── pkg/
│   └── logger/
│       └── zap.go                      # Structured logger (Uber Zap)
├── deployments/
│   ├── docker-compose.yml              # Cluster com App, Postgres e Redis
│   └── Dockerfile                      # Multi-stage distroless build
├── tests/
│   ├── benchmark_test.go               # Testes de stress e alocação de memória
│   ├── integration_test.go             # Testes e2e com Postgres/Redis via containers
│   └── enforcer_test.go                # Testes unitários de RBAC e ABAC
├── go.mod
└── go.sum
```

### Resumo das Melhorias Implementadas
| Vulnerabilidade / Gargalo Anterior | Solução Implementada | Impacto Técnico |
|---|---|---|
| Race Conditions em Mutações | Substituição pelo casbin.SyncedEnforcer com RWMutex nativo | Segurança garantida em ambientes de alta concorrência multithreaded |
| Split-Brain em Falhas do Redis | Heartbeat de recarga preventiva periódico (startPeriodicSync) | Consistência recuperada mesmo em quedas temporárias de rede pub/sub |
| Sobrecarga de I/O em Telas Ricas | Endpoint de /batch-enforce | Redução de N requisições HTTP para uma única chamada atômica |
| Superfície de Ataque da Imagem | Migração para imagem Google Distroless multi-stage | Imagem final < 25MB, sem compilador, pacotes ou shell vulneráveis |
| Falta de Padronização de Pastas | Alinhamento com o Standard Go Project Layout (cmd/, internal/) | Isolamento estrito de pacotes e APIs não exportadas para o ecossistema |

---

## 🚀 2. Guia de Instalação & Pré-requisitos

### Pré-requisitos de Sistema
Para rodar a plataforma de autorização centralizada, certifique-se de possuir:
* **Go SDK (versão 1.24+ ou v1.25 recomendado)**
* **Redis server** instalado e rodando na porta `6379`
* **PostgreSQL database** (opcional, fallback para SQLite embutido ativado por padrão se ausente)

### Passo a Passo para Instalação Local

1. **Clonar o Repositório e Navegar:**
   ```bash
   git clone <URL_DO_REPOSITORIO>
   cd casbin-authz-platform
   ```

2. **Instalar Dependências do Módulo Go:**
   ```bash
   go mod download
   ```

3. **Executar em Modo Standalone:**
   ```bash
   go run cmd/api/main.go
   ```

---

## ⚙️ 3. Configuração & Variáveis de Ambiente

As variáveis de ambiente ajustam dinamicamente a persistência e a velocidade de propagação das atualizações de política.

### Tabela de Parâmetros

| Nome da Variável | Descrição | Valor Padrão | Obrigatória? | Sensível? |
|------------------|-----------|--------------|--------------|-----------|
| `DB_DRIVER` | Driver de banco para persistência das políticas (`postgres`/`sqlite`) | `sqlite` | Não | Não |
| `DATABASE_URL` | String de conexão para o banco de dados oficial das políticas | `policies.db` | Não | Sim |
| `REDIS_URL` | Endpoint do Redis para o watcher de atualizações | `localhost:6379` | Sim | Não |
| `SERVER_PORT` | Porta HTTP onde os endpoints administrativos escutam | `8080` | Não | Não |
| `CASBIN_MODEL_PATH` | Caminho para o modelo PERM | `config/model_rbac_tenants.conf` | Não | Não |
| `LOG_LEVEL` | Nível de log do Uber Zap (`debug`, `info`, `warn`, `error`) | `info` | Não | Não |

---

## 🛠️ 4. Instruções de Execução e Uso

### Executando em Produção com Docker Compose
Para subir a plataforma de autorização centralizada de forma integrada ao Redis e Postgres corporativos, utilize:
```bash
docker-compose -f deployments/docker-compose.yml up -d --build
```

### Exemplos Práticos de Chamada de API

#### 1. Validar se usuário tem permissão para ler um recurso (Check Policy)
```bash
curl -X POST http://localhost:8080/api/v1/enforce \
     -H "Content-Type: application/json" \
     -d '{
       "sub": "alice",
       "dom": "tenant-1",
       "obj": "/api/v1/resource",
       "act": "GET"
     }'
```

#### 2. Validação em Lote (High-Throughput Batch Enforce)
```bash
curl -X POST http://localhost:8080/api/v1/batch-enforce \
     -H "Content-Type: application/json" \
     -d '{
       "checks": [
         {"sub": "alice", "dom": "tenant-1", "obj": "/api/v1/resource", "act": "GET"},
         {"sub": "bob", "dom": "tenant-1", "obj": "/api/v1/resource", "act": "GET"}
       ]
     }'
```

#### 3. Adicionar Nova Regra RBAC para Domínio Dinamicamente
```bash
curl -X POST http://localhost:8080/api/v1/policies \
     -H "Content-Type: application/json" \
     -d '{
       "sub": "bob",
       "dom": "tenant-2",
       "obj": "/finance-dashboard",
       "act": "write"
     }'
```

---

## 🧪 5. End-to-End Testing & Qualidade

```bash
go test -v ./...
go test -bench=. ./tests/...
```
