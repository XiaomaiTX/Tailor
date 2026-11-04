---
status: active
doc_type: api
owner: engineering
last_reviewed: 2026-08-05
source_of_truth:
  - api/openapi/openapi.yaml
  - backend/internal/router/router.go
  - docs/architecture/data-and-contracts.md
---

# API Conventions

## Scope

本文件描述 Go React Starter 的对外 HTTP API 通用约定。  
具体接口清单见 `route-index.md`。

## Boundary Type

- interface type: HTTP / JSON
- base path: `/api/v1`
- authentication / trust model: **无认证**，所有接口公开（详见 `docs/architecture/auth-and-permissions.md`）
- OpenAPI 权威源：`api/openapi/openapi.yaml`

## Request And Response Notes

- 请求体与响应体默认 `application/json`；错误响应统一为 `application/problem+json`
- 输入解析由 `oapi-codegen` 生成的 strict handler 负责；解析失败时 router 的 `ErrorHandler` 写出 `validation` Problem
- service 层用哨兵错误（`ErrValidation`、`ErrNotFound`）表达业务异常，handler 负责映射到对应 Problem
- Problem 字段：`type`（URN，形如 `urn:go-react-starter:problem:<kind>`）、`title`、`status`、可选 `detail`、可选 `instance`（自动填充为请求路径）
- 列表接口分页契约：`page`（≥1，默认 1）、`pageSize`（1–100，默认 20）、响应 `{ page, pageSize, list, total }`，排序 `created_at DESC, id DESC`
- 时间字段使用 ISO 8601（`date-time`）

## Versioning

- 当前只存在 `/api/v1`
- 新增破坏性变更应引入 `/api/v2` 并同时维护旧版本，直到调用方迁移完成
- 非破坏性变更在原路径下进行，无需新建版本

## Synchronization Requirements

接口变化时必须按以下顺序同步：

1. 更新 `api/openapi/openapi.yaml`（生产方契约）
2. 运行 `make openapi-lint` 与 `make generate`（重新生成 Go 桩与 React 客户端）
3. 更新 handler / service / repository 实现（共享 schema 或类型）
4. 更新前端调用方（消费方）
5. 更新 `route-index.md` 与相关 feature 文档（文档索引与行为文档）
6. `make generate-check` 确认无未提交漂移

## Notes

- 任何新增接口都需要在 `route-index.md` 中登记
- 不要在 handler 或前端页面里复制 Problem 结构，统一使用 `internal/problem` 与生成类型
