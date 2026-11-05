---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - docs/standards/testing-and-verification.md
  - docs/standards/pre-completion-checklist.md
  - Makefile
---

# Testing Guide

## Purpose

帮助开发者快速决定测试放在 Peer Relay Manager 的哪一层，并给出最小应补的验证。

## Preconditions

- 已阅读 `docs/standards/testing-and-verification.md`
- 已明确变更的根因所在层
- 已准备 Go 1.27.3、pnpm 12.17.0；Compose 烟测需要 Docker

## Layered Strategy

| 根因层 | 选择 | 现有参考 |
| --- | --- | --- |
| 业务规则 / 输入归一化 | `service` 层单元测试 | `backend/internal/service/example_test.go` |
| SQL / 迁移 / 数据访问 | `repository` 层临时 SQLite 文件测试 | `backend/internal/repository/example_sqlite_test.go` |
| HTTP 入参解析、路由、Problem 映射 | `router` / `handler` 单元测试（`httptest`） | `backend/internal/router/router_test.go`、`backend/internal/handler/api_test.go` |
| 配置加载 / Problem 输出 | `config`、`problem` 单元测试 | `backend/internal/config/config_test.go`、`backend/internal/problem/problem_test.go` |
| 前端纯辅助逻辑 | 轻量单元测试 | `frontend/src/api/http.test.ts`、`frontend/src/i18n/i18n.test.tsx` |
| 前端 UI 行为 | `@testing-library/react` + jsdom | `frontend/src/pages/example-page.test.tsx`、`frontend/src/pages/not-found-page.test.tsx` |
| 用户偏好 / 状态 | Zustand store 单元测试 | `frontend/src/stores/preference-store.test.ts` |

## Steps

1. 先判断根因在业务层、数据层、契约层还是 UI 辅助逻辑层
2. 优先在最接近根因的层补自动化测试
3. 再补构建、类型检查、lint 等基础验证
4. 如果无法补测试，记录原因和未来补位位置

## Default Commands

```bash
make backend-test
make backend-storage-test # 临时 SQLite 文件，无需 Docker
make backend-lint
make backend-build

make frontend-test
make frontend-lint
make frontend-typecheck
make frontend-build

make openapi-lint
make generate-check
make verify                # 一键本地验证（不含 backend-storage-test）
```

## Expected Outcome

- 使用最轻但足够稳定的测试层保护真实风险点
- `make verify` 在本地通过；如包含契约变更，`make generate-check` 干净

## Troubleshooting

- 不要默认上最重的 e2e
- 不要只跑 `make backend-build` 就当作已完成行为验证
- `make backend-storage-test` 使用临时数据库文件，不依赖外部服务或 Docker
- Vitest 默认 `environment: 'jsdom'`，新增依赖 DOM API 的测试时确认 setup 已包含
