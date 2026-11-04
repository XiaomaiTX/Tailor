---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-08-05
source_of_truth:
  - backend/internal/
  - frontend/src/
  - api/openapi/openapi.yaml
---

# Module Map

## Scope

本文件映射 Go React Starter 的主要模块、入口点、关键代码路径与上下游边界。  
分层方向见 `docs/standards/dependency-layering.md`。

## Module Table

### 后端

| 模块 | 主要入口 | 关键代码路径 | 上游依赖 | 下游依赖 |
| --- | --- | --- | --- | --- |
| `cmd/api` | `main.go` | `backend/cmd/api` | `config`、`router`、`handler`、`service`、`repository` | 启动 HTTP 服务并优雅退出 |
| `config` | `config.Load` | `backend/internal/config` | 环境变量 | 进程启动、日志级别、CORS |
| `router` | `router.New` | `backend/internal/router` | `generated`、`middleware`、`problem`、`config` | Gin 引擎与路由注册 |
| `middleware` | `RequestID` 等 | `backend/internal/middleware` | `gin`、`problem`、`zap` | 安全头、CORS、访问日志、Recovery |
| `handler` | `handler.API` | `backend/internal/handler` | `generated`、`service`、`problem`、`model` | HTTP 入参与响应映射 |
| `service` | `ExampleService` | `backend/internal/service` | `repository`、`model` | 业务规则、输入归一化 |
| `repository` | `ExampleRepository` | `backend/internal/repository` | `gorm`、`model` | 数据访问与迁移兼容 |
| `model` | `Example` | `backend/internal/model` | GORM struct | 数据库表映射 |
| `problem` | `problem.Write` | `backend/internal/problem` | `gin`、`net/http` | RFC 7807 错误响应 |
| `migrations` | `000001_create_examples` | `backend/migrations` | PostgreSQL | 表结构与索引 |
| `generated`（Go） | `openapi.gen.go` | `backend/internal/api/generated` | `oapi-codegen` 输出 | 严格服务端接口与 DTO |

### 前端

| 模块 | 主要入口 | 关键代码路径 | 上游依赖 | 下游依赖 |
| --- | --- | --- | --- | --- |
| `main.tsx` | `createRoot` | `frontend/src/main.tsx` | React、React Query、Router、ThemeProvider、I18nProvider | SPA 启动 |
| `app/router` | `router` | `frontend/src/app/router.tsx` | React Router | 页面装配 |
| `pages` | `ExamplePage` 等 | `frontend/src/pages` | `@/api/generated/client`、`@/i18n`、`@/components/ui` | 用户视图 |
| `components` | `Button`、`Input`、`ThemeProvider` | `frontend/src/components` | Tailwind、Zustand | 复用 UI 与状态 |
| `stores` | `preference-store` | `frontend/src/stores` | Zustand | 用户偏好持久化 |
| `i18n` | `I18nProvider`、`useI18n` | `frontend/src/i18n` | React Context、`messages/*.ts` | 中英文文案 |
| `api/http` | `customFetch` | `frontend/src/api/http.ts` | fetch | 统一传输层与 Problem 解析 |
| `api/generated`（React） | `client.ts`、`model/*` | `frontend/src/api/generated` | orval 输出、`api/http` | React Query hooks 与类型 |
| `lib/problem` | `problemMessageKey` | `frontend/src/lib/problem.ts` | `ApiProblemError` | 错误文案映射 |

### 工具与编排

| 模块 | 主要入口 | 关键代码路径 | 上游依赖 | 下游依赖 |
| --- | --- | --- | --- | --- |
| OpenAPI | `openapi.yaml` | `api/openapi/openapi.yaml` | Redocly、oapi-codegen、orval | 契约源 |
| Makefile | `verify` 等 | `Makefile` | Go、pnpm、golangci-lint | 默认验证入口 |
| Compose | `docker-compose.example.yaml` | 仓库根 | postgres、redis、migrate、backend、frontend | 容器化编排 |

## Boundaries

- 后端严格分层：`handler -> service -> repository -> model`，反向依赖视为违规
- 前端调用必须经 `@/api/generated/client`；页面不直接拼装 URL
- 任何 generated 目录的改动只能通过重新生成产生，不可手改
- Compose 用 `${VAR}` 占位，所有敏感值来自 `.env`

## Invariants

- 模块划分必须与 `dependency-layering.md` 一致
- 同一职责不要拆成多个影子模块
- 新增模块时同步更新本表与相关 architecture / feature 文档
