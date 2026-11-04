---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-08-05
source_of_truth:
  - frontend/src/pages/example-page.tsx
  - backend/cmd/api/main.go
  - backend/internal/handler/api.go
  - backend/internal/service/example.go
  - backend/internal/model/example.go
  - api/openapi/openapi.yaml
---

# Example 资源链路

> 这是一个技术基座示例资源链路，用于验证 OpenAPI → Go 分层 → React 调用方的端到端工作方式。它不是业务能力，第一个真实业务模块落地时应在本目录新增独立 feature 文档。

## Module Purpose

- 提供可被替换的中性资源 CRUD 链路
- 验证 OpenAPI 契约 → Go 生成桩 → React Query hooks 的代码生成与类型一致
- 验证健康探针、Problem 错误响应、CORS、安全头和 SQLite 表结构初始化

## Entry Points

### User Entry

- 浏览器路由 `/examples` → [frontend/src/pages/example-page.tsx](../../frontend/src/pages/example-page.tsx)
- 列表查询、新建、编辑、删除都在该页面完成

### System Entry

- `GET /api/v1/health/live`、`GET /api/v1/health/ready`：进程存活与依赖就绪探针，与 Example 业务共用同一组中间件与 Problem 错误处理
- Compose 启动时 `backend` healthcheck 调用 `/health/live`

## Permission Boundaries

- 当前无身份认证；所有接口对任何能访问后端的客户端公开
- 写操作（POST / PUT / DELETE）也没有访问控制
- 详见 [docs/architecture/auth-and-permissions.md](../architecture/auth-and-permissions.md)

## Key Invariants

- Example 列表分页：`page ≥ 1`、`1 ≤ pageSize ≤ 100`，排序固定为 `created_at DESC, id DESC`
- `name` 长度 1–120（按 UTF-8 字符计数，使用 `[]rune`）；`description` 可空，最长 2000 字符
- SQLite 表结构由 `model.Example` 定义并在后端启动时自动初始化；API 字段变化需同步 OpenAPI schema 与生成产物
- 错误响应必须经 `internal/problem` 输出为 `application/problem+json`；handler / service 不直接拼装错误 JSON
- 前端调用必须走 `@/api/generated/client` 暴露的 React Query hooks，写操作完成后通过 `queryClient.invalidateQueries` 让列表与单条缓存失效

## Primary Code Files

- 契约：[api/openapi/openapi.yaml](../../api/openapi/openapi.yaml)
- 后端：
  - [backend/internal/handler/api.go](../../backend/internal/handler/api.go)
  - [backend/internal/service/example.go](../../backend/internal/service/example.go)
  - [backend/internal/repository/example.go](../../backend/internal/repository/example.go)
  - [backend/internal/model/example.go](../../backend/internal/model/example.go)
- 前端：
  - [frontend/src/pages/example-page.tsx](../../frontend/src/pages/example-page.tsx)
  - [frontend/src/api/generated/client.ts](../../frontend/src/api/generated/client.ts)
  - [frontend/src/api/http.ts](../../frontend/src/api/http.ts)
  - [frontend/src/lib/problem.ts](../../frontend/src/lib/problem.ts)
