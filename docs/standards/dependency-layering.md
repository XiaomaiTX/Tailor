---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-08-05
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/architecture/module-map.md
---

# Dependency Layering Standard

## Scope

本标准定义 Peer Relay Manager 的依赖方向约束。

## Core Rule

- 依赖只能朝更底层抽象流动
- 上层可以依赖下层抽象，下层不能直接依赖上层实现
- 边界处的转换、适配、序列化应集中在边界层
- OpenAPI 是跨端契约单一来源；生成的代码不可手改

## Web Full-Stack 当前结构

后端（`backend/internal/`）按以下方向分层：

```text
handler -> service -> repository -> model
```

补充约束：

- `handler` 只做 HTTP 入参解析、调用 service、把结果映射为 OpenAPI 生成的响应对象；不包含业务规则
- `service` 承担业务规则、输入归一化与跨 repository 编排
- `repository` 只做 GORM 数据访问；`model.Example` 不依赖更高层
- `model` / schema 不依赖 handler / service
- `router` 负责把 generated strict handler 挂载到 `/api/v1`，并装配中间件
- `middleware` 依赖 `gin.Context` 与 `problem`，不依赖业务 service
- `problem` 是错误响应的唯一封装层；handler 返回错误时必须经过它

前端（`frontend/src/`）按以下方向分层：

```text
page/component -> api/generated -> api/http -> fetch
```

补充约束：

- 页面与组件通过 `@/api/generated/client` 暴露的 React Query hooks 调用接口，不直接拼装 URL
- `api/http.ts` 的 `customFetch` 是统一传输层，负责 `Accept`、`Content-Type`、Problem 解析
- `api/generated/` 下所有文件由 orval 生成，不可手改；如需调整改 OpenAPI 后重新 `make generate`
- 用户可见文案必须走 `i18n/messages/*.ts`，不允许在页面里写硬编码字符串

## Generated Code Boundary

- Go 服务端桩：`backend/internal/api/generated/openapi.gen.go`，由 `oapi-codegen` 生成
- React 客户端：`frontend/src/api/generated/`，由 `orval` 生成
- 改契约时：先改 `api/openapi/openapi.yaml`，再 `make generate`，再补 handler 实现与调用方
- 任何对 generated 目录的手改都会在 `make generate-check` 阶段被识别为漂移

## Common Violations

- handler / 页面组件直接包含业务规则或 SQL
- service 反向依赖 handler 或 router
- repository 暴露 GORM 内部细节给 service
- 前端页面绕过 generated hooks，自己写 fetch URL
- 在 generated 目录里手改类型或客户端

## Enforcement

- 在代码审查中检查依赖方向
- `docs/architecture/module-map.md` 记录模块边界
- `docs/standards/pre-completion-checklist.md` 检查是否引入新越层调用
- `make generate-check` 强制 OpenAPI 与生成产物对齐
