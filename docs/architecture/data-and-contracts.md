---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-08-05
source_of_truth:
  - backend/migrations/000001_create_examples.up.sql
  - backend/internal/model/example.go
  - api/openapi/openapi.yaml
  - backend/internal/problem/problem.go
---

# Data And Contracts

## Scope

本文件说明 Go React Starter 的数据存储、schema、共享契约与外部接口边界。  
只记录当前已存在的事实。

## Data Stores

- **PostgreSQL**：唯一持久化数据存储
  - 表：`examples`（`id BIGSERIAL`、`name VARCHAR(120) NOT NULL`、`description TEXT NULL`、`created_at` / `updated_at TIMESTAMPTZ`）
  - 索引：`idx_examples_created_at_id (created_at DESC, id DESC)`，支撑分页按创建时间倒序
  - GORM 映射：`backend/internal/model/example.go`
- **Redis**：当前仅用于 `/health/ready` 的就绪探针，不持久化业务数据

## Schema Migrations

- 迁移目录：`backend/migrations`
- 工具：`golang-migrate v4.19.1`
- 当前迁移：`000001_create_examples.up.sql` / `.down.sql`
- 执行入口：
  - 容器：`migrate` 服务（`docker-compose.example.yaml`）
  - 本地 / CI：`make migrate-up`、`make migrate-down-one`、`make migrate-version`、`make migrate-force VERSION=...`
- 必填变量：`DATABASE_URL`
- 应用本身不建表、不变更 schema；迁移失败会阻塞 `backend` 启动

## Shared Contracts

- **OpenAPI**：`api/openapi/openapi.yaml` 是跨端契约的唯一来源
- **Go 服务端桩**：`backend/internal/api/generated/openapi.gen.go`，由 `oapi-codegen v2.5.1` 生成
- **React 客户端**：`frontend/src/api/generated/`，由 `orval` 生成（含 React Query hooks 与 model 类型）
- **错误契约**：`internal/problem` 输出 RFC 7807 风格 `application/problem+json`，包含 `type`、`title`、`status`、`detail?`、`instance?` 字段

## Pagination Contract

- 输入：`page`（int32，≥1，默认 1）、`pageSize`（int32，1–100，默认 20）
- 输出：`ExampleList { page, pageSize, list[], total }`
- 排序：`created_at DESC, id DESC`，确保分页稳定

## External Interfaces

- HTTP API：`/api/v1`，详见 `docs/api/route-index.md`
- 无 SDK、无文件格式契约、无 OS 集成（除标准文件与网络栈）

## Compatibility Notes

- 当前没有需要兼容字段或兼容模式的旧契约；首次契约变更时再补本节
- 迁移若引入破坏性变更（删列、改类型），必须新增迁移文件并提供 down 迁移

## Invariants

- 跨边界类型变更必须从 `api/openapi/openapi.yaml` 起步，再 `make generate`，再补 handler / service / 调用方
- 同一契约不在多个路径分别维护
- `examples` 表与 `model.Example` 字段一一对应，新增字段必须同步迁移、模型、OpenAPI schema 与生成产物
