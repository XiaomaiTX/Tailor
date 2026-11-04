---
status: active
doc_type: api
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - api/openapi/openapi.yaml
  - backend/internal/router/router.go
  - backend/internal/handler/api.go
---

# Route Index

## Scope

本文件登记当前稳定的 HTTP 接口。所有接口都在 `/api/v1` 下，所有接口当前公开（无认证）。

## Interface Table

| Type | Identifier | Purpose | Auth / Trust Boundary | Consumer |
| --- | --- | --- | --- | --- |
| HTTP GET | `/api/v1/health/live` | 进程存活探针 | 公开；Compose `backend` healthcheck 调用 | Compose、Nginx、运维监控 |
| HTTP GET | `/api/v1/health/ready` | SQLite 数据库就绪探针 | 公开；返回 `200` 或 `500 problem+json` | Compose、运维监控 |
| HTTP GET | `/api/v1/examples` | 分页列出 Example | 公开 | 前端 `useListExamples` |
| HTTP POST | `/api/v1/examples` | 创建 Example | 公开 | 前端 `useCreateExample` |
| HTTP GET | `/api/v1/examples/{id}` | 获取单个 Example | 公开 | 前端 `useGetExample` |
| HTTP PUT | `/api/v1/examples/{id}` | 更新 Example | 公开 | 前端 `useUpdateExample` |
| HTTP DELETE | `/api/v1/examples/{id}` | 删除 Example，响应 `204` | 公开 | 前端 `useDeleteExample` |

## Details

- 列表参数：`page`（int32，默认 1）、`pageSize`（int32，默认 20，上限 100）
- 列表响应：`{ page, pageSize, list: Example[], total: int64 }`
- 写接口入参：`ExampleInput { name: string(1–120), description?: string(≤2000) }`
- 错误响应：`400 validation`（请求不合法）、`404 not-found`（资源不存在，仅 GET/PUT/DELETE 单个）、`500 internal`（服务内部错误）
- 所有错误响应 Content-Type 为 `application/problem+json`，包含 `type`、`title`、`status`、可选 `detail`、`instance`

## How To Maintain

- 新增稳定入口时在本表登记，并同步更新 `api/openapi/openapi.yaml` 与相关 feature 文档
- 边界或权限变化时同步更新本表与 `docs/architecture/auth-and-permissions.md`
- 不在这里复制完整请求响应示例；详细约定写在 `conventions.md` 或具体 feature 文档
