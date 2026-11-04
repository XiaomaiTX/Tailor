---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-08-05
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/guides/testing-guide.md
---

# Testing And Verification Standard

## Scope

本标准定义 Go React Starter 仓库的验证命令、覆盖要求、允许例外和最小完成验证。  
具体测试选择与分层策略见 `docs/guides/testing-guide.md`。

## Required Tool Versions

工具版本以仓库事实为准；本机已安装版本可能低于以下目标，使用前用对应的版本管理器对齐。

| Tool | Version | Notes |
| --- | --- | --- |
| Go | 1.27.3 | 见 `backend/go.mod`；后端构建与测试用 |
| Node.js | 24.x | `.nvmrc` 维护大版本；前端镜像锁定 `node:24.12.0-alpine3.23` |
| pnpm | 12.17.0 | 见 `frontend/package.json` 的 `packageManager` |
| golangci-lint | 当前稳定版 | 后端 lint；本地缺失时 `make backend-lint` 会失败 |
| ESLint | 10.x | 前端 lint，由 `pnpm --dir frontend lint` 触发 |
| Vitest | 4.x | 前端测试运行器，由 `pnpm --dir frontend test` 触发 |
| Redocly CLI | 2.x | OpenAPI 校验，由 `make openapi-lint` 触发 |
| Air | v1.65.1 | 后端热重载，仅本地开发用；由 `make backend-dev` 经 `go run` 自动拉取，无需安装 |
| Docker | 任意稳定版 | 仅在运行 `make backend-integration` 或 Compose 烟测时需要 |

## Default Commands

仓库根 `Makefile` 是命令入口。常用目标：

### Backend / Core

- `make backend-build`：编译 `backend/cmd/api`
- `make backend-test`：运行 `go test ./...`（不含集成测试）
- `make backend-lint`：运行 `golangci-lint run ./...`
- `make backend-integration`：运行 `go test -tags integration ./internal/repository`，需要 Docker 拉起 Testcontainers PostgreSQL
- `make backend-dev`：通过 Air 启动后端热重载

### Frontend / UI

- `make frontend-build`：`pnpm --dir frontend build`（含 `tsc -b` 类型检查）
- `make frontend-test`：`pnpm --dir frontend test`（Vitest）
- `make frontend-lint`：`pnpm --dir frontend lint`（ESLint）
- `make frontend-typecheck`：`pnpm --dir frontend exec tsc -b`

### OpenAPI / 生成一致性

- `make openapi-lint`：用 Redocly 校验 `api/openapi/openapi.yaml`
- `make generate`：重新生成 Go 服务端桩与 React 客户端
- `make generate-check`：重新生成并断言 `backend/internal/api/generated` 与 `frontend/src/api/generated` 没有未提交漂移

### 数据库迁移

迁移命令直接调用 `golang-migrate`，`DATABASE_URL` 是必填环境变量：

- `make migrate-up`
- `make migrate-down-one`
- `make migrate-version`
- `make migrate-force VERSION=1`

### 总入口

- `make verify`：依次执行 `openapi-lint` → `generate-check` → `backend-lint` → `backend-test` → `backend-build` → `frontend-lint` → `frontend-typecheck` → `frontend-test` → `frontend-build`
- `make backend-integration` 不在 `verify` 内，仅在具备 Docker 时按需运行

## Rules

- `build`、`lint`、`typecheck` 不能替代行为级测试
- 新功能应在合理可测的前提下增加自动化覆盖
- bug fix 应在合理可测的前提下增加回归测试
- 契约变化至少验证边界两侧：变更 OpenAPI 后必须运行 `make generate-check`，并视情况补 service / repository / handler / 前端调用方测试
- 测试应锁定真实变更逻辑，不要在测试中重写生产逻辑
- 记录的命令必须可直接运行；`make verify` 是默认本地验证入口

## Test Choice

- 纯业务规则优先在 `service` 层测试（如 `backend/internal/service/example_test.go`）
- 数据访问与 SQL/迁移行为优先在 `repository` 层用 Testcontainers 集成测试（`*_integration_test.go`，需 `//go:build integration`）
- 路由、中间件、Problem 转换在 `router` / `handler` / `middleware` 层用 `httptest` 测试
- 前端纯辅助逻辑用轻量单元测试；UI 行为用 `@testing-library/react`（如 `frontend/src/pages/example-page.test.tsx`）
- 不要默认选择最重的测试层

## Allowed Exceptions

仅在明确说明原因时允许省略测试，例如：

- 纯文档修改
- 纯格式调整
- 明确行为不变的重命名
- Testcontainers / Docker 不可用时跳过 `make backend-integration`，并在交付说明中记录
- 外部依赖使本地稳定测试暂时不可行

## Minimum Verification

- 文档修改：检查交叉引用、front matter、状态字段、UTF-8 无 BOM
- 核心逻辑修改：运行对应 `make backend-test` 与 `make backend-lint`
- UI 修改：运行 `make frontend-test`、`make frontend-lint`、`make frontend-typecheck`，并在相关时补测试
- 契约修改：运行 `make openapi-lint` 与 `make generate-check`，并验证 service / handler / 前端调用方
- 权限 / 安全修改：补行为级验证并复查 `docs/architecture/auth-and-permissions.md`

## Completion Check

- 是否运行了 `make verify`，或在子集中运行了对应命令
- 是否添加了与变更匹配的自动化覆盖
- 如果跳过测试，是否明确记录原因
- 如果新增了文档化命令，是否验证过命令可运行
