# 📘 Tutorial de Operação e Instalação: Casbin Authorization Platform

Guia passo a passo para configuração, inicialização, execução de testes e resolução de problemas no projeto **Casbin Authorization Platform** (`casbin-authz-platform`).

---

## 🛠️ 1. Visão Geral e Arquitetura

O **Casbin Authorization Platform** é um motor centralizado de autorização e controle de acesso (IAM) de nível enterprise projetado em **Go** utilizando **Apache Casbin v2 (SyncedEnforcer)**, **Go Gin**, **PostgreSQL (GORM Adapter)** e **Redis (Watcher)**. O sistema resolve a complexidade de gerenciar permissões em ecossistemas de microserviços distribuídos e multi-tenant, desacoplando completamente as políticas de segurança do código de aplicação.

- **Tecnologias Principais:** Go, Casbin v2 SyncedEnforcer, Gin, GORM, Redis Watcher, Uber Zap.
- **Diretório do Módulo:** `casbin-authz-platform`

---

## 📋 2. Pré-requisitos do Sistema

Antes de iniciar a configuração local, certifique-se de ter os seguintes componentes instalados em seu ambiente de desenvolvimento:

- **Git:** Para versionamento.
- **Go SDK (1.24+ / 1.25 recomendado)**
- **Docker Engine & Docker Compose:** Recomendado para execução do cluster com Postgres e Redis.

---

## 🚀 3. Passo a Passo de Setup e Instalação

### Passo 1: Acessar a Pasta do Projeto
```bash
cd casbin-authz-platform
```

### Passo 2: Instalação de Dependências
```bash
go mod download
```

---

## ⚙️ 4. Configuração do Ambiente

Variáveis suportadas:

```ini
SERVER_PORT=8080
DB_DRIVER=sqlite
DATABASE_URL=policies.db
REDIS_URL=localhost:6379
CASBIN_MODEL_PATH=config/model_rbac_tenants.conf
LOG_LEVEL=info
```

---

## 💻 5. Guia de Execução

### Execução Local
```bash
go run cmd/api/main.go
```

### Execução em Cluster Distribuído
```bash
docker-compose -f deployments/docker-compose.yml up -d --build
```

---

## 🧪 6. Testes e Validação

Para validar o funcionamento do módulo e garantir que não há regressões, execute a suíte de testes:

```bash
go test -v ./...
go test -bench=. ./tests/...
```

---

## 🔍 7. Troubleshooting e Resolução de Problemas

1. **Redis Watcher Inativo / Desconectado:**
   - O enforcer opera em modo isolado com salvaguarda de recarga periódica de políticas a cada 60 segundos (`startPeriodicSync`).
2. **Concorrência e Escritas Simultâneas:**
   - Em produção com réplicas concorrentes, utilize `DB_DRIVER=postgres` para transações relacionais atômicas e serializadas.
3. **Falhas de Permissão no Interceptor:**
   - Verifique os headers `X-User-ID` e `X-Tenant-ID` (ou contexto Gin `user_id` e `tenant_id`) e consulte o log de auditoria emitido pelo Uber Zap associado ao `trace_id`.
