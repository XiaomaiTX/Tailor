.PHONY: openapi-lint generate generate-go generate-react generate-check \
	migrate-up migrate-down-one migrate-version migrate-force \
	backend-test backend-integration backend-lint backend-build backend-dev \
	frontend-lint frontend-typecheck frontend-test frontend-build frontend-dev dev verify

# 默认读取根目录 .env；命令行变量优先覆盖。
-include .env

OPENAPI := api/openapi/openapi.yaml
OAPI    := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1
MIGRATE := go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1
AIR     := go run github.com/air-verse/air@v1.65.1
FE      := pnpm --dir frontend
# 惰性变量：仅 migrate-* 目标展开时校验
DB_URL  = $(if $(strip $(DATABASE_URL)),$(DATABASE_URL),$(error 需要 DATABASE_URL：写入 .env 或命令行传入))

openapi-lint:
	$(FE) exec redocly lint --config ../redocly.yaml ../$(OPENAPI)

generate: generate-go generate-react

generate-go:
	$(OAPI) --config api/openapi/oapi-codegen.yaml $(OPENAPI)

generate-react:
	$(FE) run generate

generate-check: generate
	git diff --exit-code -- backend/internal/api/generated frontend/src/api/generated

# PostgreSQL schema 迁移入口
migrate-up:
	@$(MIGRATE) -path backend/migrations -database "$(DB_URL)" up

migrate-down-one:
	@$(MIGRATE) -path backend/migrations -database "$(DB_URL)" down 1

migrate-version:
	@$(MIGRATE) -path backend/migrations -database "$(DB_URL)" version

migrate-force:
	@$(MIGRATE) -path backend/migrations -database "$(DB_URL)" force $(VERSION)

backend-test:
	cd backend && go test ./...

# 需要本机 Docker（Testcontainers 拉起 PostgreSQL）
backend-integration:
	cd backend && go test -tags integration ./internal/repository

backend-lint:
	cd backend && golangci-lint run ./...

backend-build:
	cd backend && go build ./cmd/api

backend-dev:
	cd backend && $(AIR) -c .air.toml

frontend-lint:
	$(FE) lint

frontend-typecheck:
	$(FE) exec tsc -b

frontend-test:
	$(FE) test

frontend-build:
	$(FE) build

frontend-dev:
	$(FE) dev

dev:
	@$(MAKE) --jobs=2 backend-dev frontend-dev

verify: openapi-lint generate-check backend-lint backend-test backend-build frontend-lint frontend-typecheck frontend-test frontend-build
