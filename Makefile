# Comandos de desenvolvimento da La fabrique. `make help` lista os alvos.
SHELL := /bin/bash
.DEFAULT_GOAL := help

-include .env
export

DB_ADMIN_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@127.0.0.1:$(POSTGRES_PORT)/postgres?sslmode=disable
DB_APP_URL   ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@127.0.0.1:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

.PHONY: help bootstrap up down logs ps up-prod down-prod ps-prod logs-prod migrate migrate-status seed create-admin \
        fmt lint test test-integration test-web build typecheck check-secrets db-backup db-restore sync-skills

help: ## Lista os comandos
	@awk 'BEGIN{FS=":.*## "} /^[a-zA-Z_-]+:.*## /{printf "  \033[36m%-18s\033[0m %s\n",$$1,$$2}' $(MAKEFILE_LIST)

bootstrap: ## Prepara o ambiente local (.env, dependências Go e web)
	@test -f .env || (cp .env.example .env && echo ".env criado a partir de .env.example")
	go mod download
	cd apps/web && npm ci

# --- Docker (dev) -----------------------------------------------------------
up: ## Sobe o stack de desenvolvimento (hot reload) em Docker
	scripts/compose.sh dev up -d --build
down: ## Derruba o stack de desenvolvimento (mantém volumes)
	scripts/compose.sh dev down
ps: ## Estado dos serviços (dev)
	scripts/compose.sh dev ps -a
logs: ## Logs seguidos (dev). Ex.: make logs S=api
	scripts/compose.sh dev logs -f --tail 100 $(S)

# --- Docker (VPS/produção). Use ENV_FILE=caminho para outro arquivo de ambiente
up-prod: ## Sobe o stack de produção (imagens finais, só o web publicado)
	scripts/compose.sh prod up -d --build
down-prod: ## Derruba o stack de produção (mantém volumes)
	scripts/compose.sh prod down
ps-prod: ## Estado dos serviços (produção)
	scripts/compose.sh prod ps -a
logs-prod: ## Logs seguidos (produção). Ex.: make logs-prod S=api
	scripts/compose.sh prod logs -f --tail 100 $(S)

# --- Banco ------------------------------------------------------------------
migrate: ## Aplica migrations pendentes no banco de dev (via host)
	DATABASE_URL='$(DB_APP_URL)' go run ./cmd/migrate up
migrate-status: ## Mostra o estado das migrations
	DATABASE_URL='$(DB_APP_URL)' go run ./cmd/migrate status
seed: ## Carrega projetos fictícios de dev (somente APP_ENV=development)
	DATABASE_URL='$(DB_APP_URL)' APP_ENV=development go run ./cmd/migrate seed-dev
create-admin: ## Cria a conta única do administrador (pergunta usuário e senha; o 2FA é cadastrado no 1º login)
	@read -p "Usuário: " u; read -s -p "Senha (mín. 12 caracteres): " p; echo; \
	  ADMIN_USERNAME="$$u" ADMIN_PASSWORD="$$p" DATABASE_URL='$(DB_APP_URL)' go run ./cmd/migrate create-admin
db-backup: ## Backup do banco (MODE=dev|prod)
	scripts/db-backup.sh $(or $(MODE),dev)
db-restore: ## Restaura em banco isolado. Ex.: make db-restore F=backups/db-XXXX.dump
	scripts/db-restore.sh $(F) $(or $(MODE),dev)

# --- Qualidade --------------------------------------------------------------
fmt: ## Formata Go e web
	gofmt -w cmd internal db tests
	cd apps/web && npx biome format --write .
lint: ## gofmt, go vet e Biome (sem alterar arquivos)
	@test -z "$$(gofmt -l cmd internal db tests)" || (echo "gofmt pendente:"; gofmt -l cmd internal db tests; exit 1)
	go vet ./...
	cd apps/web && npx biome check .
typecheck: ## Tipos do front
	cd apps/web && npx tsc -b
test: ## Testes unitários e de contrato (Go + web); não exige banco
	go test -count=1 ./...
	cd apps/web && npx vitest run
test-web: ## Somente testes do front
	cd apps/web && npx vitest run
test-integration: ## Testes com PostgreSQL real (exige `make up` rodando)
	@test -n "$(POSTGRES_PASSWORD)" || (echo "defina POSTGRES_* no .env (make bootstrap)"; exit 1)
	TEST_DATABASE_URL='$(DB_ADMIN_URL)' go test -count=1 ./tests/integration/...
build: ## Compila binários Go (./bin) e o front (apps/web/dist)
	mkdir -p bin && go build -trimpath -o bin/ ./cmd/...
	cd apps/web && npm run build
check-secrets: ## Procura padrões de segredo em arquivos rastreados
	scripts/check-secrets.sh

sync-skills: ## Copia skills/ para .claude/skills, .agents/skills e .agent/skills
	node scripts/sincronizar-skills.mjs --apply
