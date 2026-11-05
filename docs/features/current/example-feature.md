---
status: active
doc_type: feature
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - backend/cmd/api/main.go
  - backend/internal/handler/api.go
  - backend/internal/service/example.go
  - backend/internal/model/example.go
  - api/openapi/openapi.yaml
---

# Example 资源链路

> 这是保留的技术基座 CRUD API，用于演示 OpenAPI → Go 分层 → React 生成客户端链路。当前控制台主流程是 Peer Relay Manager；该示例页面不再作为主导航入口。

## Module Purpose

- 提供可被替换的中性资源 CRUD 链路
- 验证 OpenAPI 契约 → Go 生成桩 → React Query hooks 的代码生成与类型一致
- 验证健康探针、Problem 错误响应、CORS、安全头和 SQLite 表结构初始化

## Entry Points

### User Entry

- API operations are retained under `/api/v1/examples` for the generated-handler example.
- The application does not currently link the starter Example page from the Peer Relay console.

### System Entry

- `GET /api/v1/health/live`、`GET /api/v1/health/ready`：进程存活与依赖就绪探针，与 Example 业务共用同一组中间件与 Problem 错误处理
- Compose 启动时 `backend` healthcheck 调用 `/health/live`

## Permission Boundaries

- Example API routes pass through the same Tailor session middleware as other business APIs.
- Only health probes are public; see [auth-and-permissions.md](../architecture/auth-and-permissions.md).
- 详见 [docs/architecture/auth-and-permissions.md](../architecture/auth-and-permissions.md)

## Key Invariants

- Example 列表分页：`page ≥ 1`、`1 ≤ pageSize ≤ 100`，排序固定为 `created_at DESC, id DESC`
- `name` 长度 1–120（按 UTF-8 字符计数，使用 `[]rune`）；`description` 可空，最长 2000 字符
- SQLite 表结构由 `model.Example` 定义并在后端启动时自动初始化；API 字段变化需同步 OpenAPI schema 与生成产物
- 错误响应必须经 `internal/problem` 输出为 `application/problem+json`；handler / service 不直接拼装错误 JSON
- React generated client artifacts remain as the example of generated API calls; the Example page is not wired into the main console.

## Primary Code Files

- 契约：[api/openapi/openapi.yaml](../../api/openapi/openapi.yaml)
- 后端：
  - [backend/internal/handler/api.go](../../backend/internal/handler/api.go)
  - [backend/internal/service/example.go](../../backend/internal/service/example.go)
  - [backend/internal/repository/example.go](../../backend/internal/repository/example.go)
  - [backend/internal/model/example.go](../../backend/internal/model/example.go)
- 前端：
  - [frontend/src/api/generated/client.ts](../../frontend/src/api/generated/client.ts)
  - [frontend/src/api/http.ts](../../frontend/src/api/http.ts)
  - [frontend/src/lib/problem.ts](../../frontend/src/lib/problem.ts)
