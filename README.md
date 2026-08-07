# 📋 Casbin Authorization Platform

[![Tech Stack](https://img.shields.io/badge/Stack-Go%201.25%20%7C%20Casbin%20v2%20%7C%20Gin%20%7C%20PostgreSQL%20%7C%20Redis-blue)](#)
[![Status](https://img.shields.io/badge/Status-Active-brightgreen)](#)
[![Security](https://img.shields.io/badge/Security-RBAC%20%26%20ABAC-orange)](#)

O **Casbin Authorization Platform** é um motor centralizado de autorização e controle de acesso (IAM) de nível enterprise projetado em **Go (v1.25.0)** utilizando **Apache Casbin v2**, **Go Gin**, **PostgreSQL (GORM Adapter)** e **Redis (Watcher)**. O sistema resolve a complexidade de gerenciar permissões em ecossistemas de microserviços distribuídos e multi-tenant, desacoplando completamente as políticas de segurança do código de aplicação.

A plataforma implementa dois modelos complementares e poderosos de autorização: controle de acesso baseado em papéis por domínios/inquilinos (**Domain-Based Multi-Tenancy RBAC**) e controle de acesso baseado em atributos dinâmicos (**Dynamic Attribute ABAC**). A persistência é gerenciada em tempo real por adaptadores relacionais e a sincronização de políticas entre réplicas ativas é realizada por canais pub/sub distribuídos via Redis, garantindo latência de consulta sub-milissegundo para os interceptores de segurança.

---

## 📐 1. Arquitetura & Componentes Internos

### Mapeamento da Estrutura de Pastas
```text
casbin-authz-platform/
├── config/                   # Arquivos de modelo declarativos do Casbin
│   ├── model_rbac_tenants.conf # Definição do modelo RBAC multi-tenant (g, g2, g3)
│   └── model_abac.conf       # Definição de atributos dinâmicos para ABAC
├── middleware/               # Interceptores de requisições e segurança
│   └── interceptor.go        # Middleware Gin para avaliação de políticas em tempo real
├── policy/                   # Gerenciamento dinâmico de regras e políticas
│   └── syncer.go             # Redis Watcher para propagação instantânea de regras
├── tests/                    # Testes automatizados do motor
│   └── enforcer_test.go      # Teste de carga e validação lógica de regras
├── Dockerfile                # Arquivo de containerização do motor Go
├── go.mod                    # Módulo Go e dependências estáveis
└── go.sum
```

### Diagrama de Fluxo de Decisão de Autorização
```text
  Client Request (HTTP REST)
            │
            ▼
┌───────────────────────────────┐
│     Gin API / Gateway         │
│  - Extract: User, Tenant, URI │
└───────────┬───────────────────┘
            │
            ▼ (Execute Middleware Interceptor)
┌───────────────────────────────┐
│    Casbin Enforcer Engine     │ ◄─── Synced via Redis Watcher
└───────────┬───────────────────┘
            │
            ├─► Evaluate Dynamic ABAC? ──► [Match Resource Attributes]
            │
            └─► Evaluate Tenant RBAC?  ──► [Match Role Hierarchy]
            │
            ▼
┌───────────────────────────────┐
│     Enforcer Decision         │
└─────┬───────────────────┬─────┘
      │                   │
   [Allow]             [Deny]
      │                   │
      ▼                   ▼
  Status 200 OK      Status 403 Forbidden
```

### Tabela de Tecnologias & Dependências

| Componente | Tecnologia | Versão Standard | Descrição / Papel no Ecossistema |
|------------|------------|-----------------|----------------------------------|
| **Linguagem** | Go | 1.25.0 | Linguagem estaticamente tipada de altíssima performance |
| **Engine de Autorização** | Apache Casbin v2 | ^2.x | Motor avaliador de regras declarativas estruturadas |
| **Framework Web** | Go Gin | ^1.9.0 | Roteador e servidor HTTP ultrarrápido |
| **Persistência de Políticas**| GORM Casbin Adapter| ^3.x | Sincronizador persistente de políticas em banco relacional |
| **Watcher de Políticas** | Redis Watcher v2 | ^2.x | Mecanismo distribuído pub/sub para sincronização de réplicas |
| **Banco de Dados** | PostgreSQL / SQLite | v15+ (Fallback SQLite)| Banco relacional para repositório oficial de regras de acesso |

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

---

## 🛠️ 4. Instruções de Execução e Uso

### Executando em Produção com Docker Compose
Para subir a plataforma de autorização centralizada de forma integrada ao Redis e Postgres corporativos, utilize:
```bash
docker-compose up -d --build
```

### Exemplos Práticos de Chamada de API

#### 1. Validar se usuário tem permissão para ler um recurso (Check Policy)
```bash
curl -X POST http://localhost:8080/api/v1/enforce \
     -H "Content-Type: application/json" \
     -d '{
       "sub": "alice",
       "dom": "tenant-1",
       "obj": "data-report",
       "act": "read"
     }'
```

**Resposta (Permissão Concedida):**
```json
{
  "allowed": true,
  "reason": "Matched policy: alice, tenant-1, data-report, read"
}
```

#### 2. Adicionar Nova Regra RBAC para Domínio Dinamicamente
```bash
curl -X POST http://localhost:8080/api/v1/policies \
     -H "Content-Type: application/json" \
     -d '{
       "sub": "bob",
       "dom": "tenant-2",
       "obj": "finance-dashboard",
       "act": "write"
     }'
```

---

## 🧪 5. End-to-End Testing & Qualidade

O repositório possui uma robusta suite de testes que valida toda a matriz de permissões RBAC por tenant e as condições de contorno dinâmicas do ABAC.

### Executando Testes Automatizados
```bash
go test -v ./...
```

Para verificar o tempo de avaliação das regras (benchmarking de performance do enforcer):
```bash
go test -bench=. ./...
```

---

## 🛡️ 6. Aspectos de Segurança & Governança (AppSec)

A governança do Casbin garante auditoria estrita em tempo de execução:

1. **Multi-Tenancy Isolado (Domain Constraint):**
   A configuração do modelo `model_rbac_tenants.conf` exige o parâmetro `dom` (domínio/inquilino) em todas as validações estruturais. Isso garante que a herança de regras de um papel (ex: `admin`) nunca ultrapasse a barreira lógica da organização do usuário, blindando o sistema contra vazamento cruzado de privilégios.

2. **Watcher de Sincronização Segura:**
   A integração do `redis-watcher` garante consistência eventual forte em menos de 10 milissegundos para todas as réplicas ativas do cluster, impedindo que políticas revogadas continuem ativas em servidores paralelos.

3. **Auditoria de Decisões:**
   O motor do enforcer está configurado para registrar logs detalhados de auditoria a cada negação de acesso (`DENY`), facilitando a investigação de potenciais varreduras ou tentativas de elevação de privilégios maliciosas.

---

## 💡 7. Troubleshooting & Solução de Problemas Comuns

| Sintoma de Erro | Causa Provável | Ação Corretiva / Resolução |
|-----------------|----------------|----------------------------|
| `Redis connection timeout` | O watcher não conseguiu se conectar ao cluster Redis Pub/Sub | Revise as credenciais na variável `REDIS_URL`. O motor tentará reconexões automáticas exponencialmente. |
| `Enforce returning false unexpectedly` | Inconsistência de casing nos nomes dos papéis ou ausência do domínio nas regras criadas | Garanta que o domínio/tenant foi incluído na chamada da função `Enforce` e verifique se as regras respeitam a sintaxe especificada no arquivo `.conf`. |
| `Database lock errors on concurrent writes`| Uso do driver padrão SQLite sob concorrência extrema de criação de regras | Altere o driver para `postgres` definindo `DB_DRIVER=postgres` para habilitar transações simultâneas seguras gerenciadas pelo PostgreSQL. |
