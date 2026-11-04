---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - backend/cmd/api/main.go
  - backend/internal/router/router.go
  - frontend/src/main.tsx
  - api/openapi/openapi.yaml
---

# Architecture Overview

## Scope

本文件描述 Go React Starter 的整体结构，只记录已实现事实。  
未实现设计写入 `docs/specs/draft/`。

## Project Intent

Go React Starter 是一个中性、可扩展的 Web 技术基座。Go API 与 React SPA 共同提供最小可运行链路，OpenAPI 是跨端契约的单一来源。`/examples` 是可被替换的端到端示例资源，不是业务能力。

## Current Implementation

- 项目类型：Web 全栈基座（Go API + React SPA）
- 后端运行形态：单进程 HTTP 服务，监听 `:$API_PORT`（默认 8080）
- 前端运行形态：Vite 开发服务器（本地）或 Nginx 静态资源 + 反向代理（容器）
- 主要技术栈：
  - Go 1.27.3、Gin、GORM、SQLite、Zap
  - React 19、Vite、React Query、React Router、Tailwind、shadcn 风格组件、Zustand
  - OpenAPI 3.0.3、oapi-codegen（Go 服务端）、orval（React 客户端）
- 主要模块：`config`、`router`、`handler`、`service`、`repository`、`model`、`middleware`、`problem`
- 容器化：`docker-compose.example.yaml` 启动 backend、frontend；SQLite 文件通过共享数据卷持久化

## Subsystems

- **后端 API**：`backend/internal/` 下的分层 Go 服务
- **数据库表结构**：由 `backend/internal/model/` 中的 GORM 模型定义，后端启动时自动创建或补充
- **前端 SPA**：`frontend/src/`，通过生成的 React Query hooks 调用后端
- **OpenAPI 契约**：`api/openapi/openapi.yaml`，是跨端类型与客户端的唯一来源
- **Compose 编排**：`docker-compose.example.yaml`，串联以上组件并定义启动顺序与健康检查

## Key Entry Files

- 后端入口：[backend/cmd/api/main.go](../../backend/cmd/api/main.go)
- 路由装配：[backend/internal/router/router.go](../../backend/internal/router/router.go)
- 配置加载：[backend/internal/config/config.go](../../backend/internal/config/config.go)
- 前端入口：[frontend/src/main.tsx](../../frontend/src/main.tsx)
- 前端路由：[frontend/src/app/router.tsx](../../frontend/src/app/router.tsx)
- 契约源：[api/openapi/openapi.yaml](../../api/openapi/openapi.yaml)
- Compose 示例：[docker-compose.example.yaml](../../docker-compose.example.yaml)

## Invariants

- OpenAPI 是契约单一来源；任何接口变化先改 `api/openapi/openapi.yaml`，再 `make generate`
- 生成目录（`backend/internal/api/generated/`、`frontend/src/api/generated/`）不可手改
- 业务规则只放在 `service` 层；handler / page / repository 不承担业务规则
- 错误响应必须经过 `internal/problem`，统一为 RFC 7807 风格 `application/problem+json`
- 当前没有身份认证；任何接入认证的改动必须同步更新 `auth-and-permissions.md` 与 `api/conventions.md`

## Notes

- 若整体结构变化，同步更新 `module-map.md` 和 `runtime-and-startup.md`
- 接口契约变化按 `docs/ai/repo-rules.md` 的 Contract Change Order 处理
