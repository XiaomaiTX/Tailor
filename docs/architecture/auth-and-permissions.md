---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-08-05
source_of_truth:
  - backend/internal/router/router.go
  - backend/internal/middleware/
  - api/openapi/openapi.yaml
---

# Auth And Permissions

## Scope

本文件说明当前认证、授权与访问控制边界。  
**当前项目没有身份认证**，所有 API 默认公开。本文件主要任务是明确这一事实，并指出未来加入认证时必须同步的边界。

## Current Model

- 认证方式：**无**
- 角色或权限模型：**无**
- 匿名访问策略：所有 `/api/v1/*` 路由对任何能访问后端的客户端公开
- 高风险操作保护方式：当前不存在需要授权的高风险操作；`DELETE /examples/{id}` 等写操作也无访问控制

## What Is Not Access Control

以下机制只是传输或浏览器层保护，**不能视为访问控制**：

- `middleware.CORS`：仅响应浏览器同源策略；`CORS_ORIGINS` 控制哪些 Origin 可以从浏览器跨域调用，攻击者可绕过
- `middleware.SecurityHeaders`：设置 `X-Content-Type-Options`、`X-Frame-Options`、`Referrer-Policy`、`Content-Security-Policy`，是浏览器内容保护
- `middleware.Recovery` / `RequestID`：运行时稳定性与可观测性，与权限无关

## Enforcement Points

当前不存在服务端权限决策点。OpenAPI 中 `security: []` 显式声明接口无认证。

## Permission Boundaries

- 不存在；任何接入认证后的"谁能读 / 谁能写"边界必须在该改动中新增，并落到本文件

## Invariants

- 权限必须在可信执行边界（服务端中间件或 service 层）强制执行
- UI / 客户端侧权限逻辑只能作为 UX 辅助，不能作为唯一保护
- 涉及权限变化时必须同步更新：
  1. 本文件
  2. `docs/api/conventions.md` 的 Boundary Type 与 trust model
  3. `docs/api/route-index.md` 中受影响接口的"Auth / Trust Boundary"列
  4. 相关 feature 文档（如 `docs/features/current/example-feature.md`）
  5. `docs/standards/security-and-secrets.md` 的"Current Posture"
  6. 行为级测试覆盖权限决策点
