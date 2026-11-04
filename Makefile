.PHONY: openapi-lint generate generate-go generate-react generate-check \
	backend-test backend-storage-test backend-lint backend-build backend-dev \
	frontend-lint frontend-typecheck frontend-test frontend-build frontend-dev dev verify

# 默认读取根目录 .env；命令行变量优先覆盖。
-include .env

OPENAPI := api/openapi/openapi.yaml
OAPI    := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1
AIR     := go run github.com/air-verse/air@v1.65.1
FE      := pnpm --dir frontend

openapi-lint:
	$(FE) exec redocly lint --config ../redocly.yaml ../$(OPENAPI)

generate: generate-go generate-react

generate-go:
	$(OAPI) --config api/openapi/oapi-codegen.yaml $(OPENAPI)

generate-react:
	$(FE) run generate

generate-check: generate
	git diff --exit-code -- backend/internal/api/generated frontend/src/api/generated

backend-test:
	cd backend && go test ./...

backend-storage-test:
	cd backend && go test ./internal/repository

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
