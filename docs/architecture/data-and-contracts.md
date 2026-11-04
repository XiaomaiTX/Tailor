---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - backend/internal/model/example.go
  - api/openapi/openapi.yaml
  - backend/internal/problem/problem.go
---

# Data And Contracts

## Scope

本文件说明 Go React Starter 的数据存储、schema、共享契约与外部接口边界。  
只记录当前已存在的事实。

## Data Stores

- **SQLite**：唯一持久化数据存储，数据库文件由 `DATABASE_PATH` 指定
  - 表：`examples`（`id INTEGER PRIMARY KEY AUTOINCREMENT`、`name VARCHAR(120) NOT NULL`、`description TEXT NULL`、`created_at` / `updated_at DATETIME`）
  - 索引：`idx_examples_created_at_id (created_at DESC, id DESC)`，支撑分页按创建时间倒序
  - GORM 模型和索引定义：`backend/internal/model/example.go`

## Schema Setup

- 后端启动时调用 GORM `AutoMigrate`，按 `model.Example` 创建或补充表结构与索引
- 启动流程不依赖独立迁移容器、迁移命令或迁移版本表
- GORM `AutoMigrate` 不会删除模型中已移除的列；需要破坏性变更或数据转换时再显式实现相应 SQL 操作

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

## Invariants

- 跨边界类型变更必须从 `api/openapi/openapi.yaml` 起步，再 `make generate`，再补 handler / service / 调用方
- 同一契约不在多个路径分别维护
- `examples` 表结构由 `model.Example` 驱动；新增 API 字段仍须同步 OpenAPI schema 与生成产物
