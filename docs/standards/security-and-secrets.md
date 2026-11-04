---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/architecture/auth-and-permissions.md
  - docs/architecture/runtime-and-startup.md
---

# Security And Secrets Standard

## Scope

本标准定义 Go React Starter 仓库的密钥、凭据、敏感数据和外部系统访问边界。  
当前项目没有身份认证，所有 API 默认公开——这不代表安全边界不重要，而是要求任何接入认证或外部系统时都先回到本文件。

## Current Posture

- **无身份认证**：当前没有用户系统、会话、Token、角色或租户模型
- **接口默认公开**：`/api/v1` 下所有路由对任何能访问后端的客户端开放
- **CORS 不是访问控制**：`middleware.CORS` 只是浏览器侧同源限制，攻击者可以绕过；真正的访问控制必须出现在服务端
- **安全响应头**：`middleware.SecurityHeaders` 设置 `X-Content-Type-Options`、`X-Frame-Options`、`Referrer-Policy`、`Content-Security-Policy`，属于传输与浏览器保护，不构成授权

详见 `docs/architecture/auth-and-permissions.md`。

## Core Rules

- 密钥、token、密码、证书私钥不得提交到仓库
- `.env`、`data/`、本地 TLS 证书等必须出现在 `.gitignore`
- `.env.example` 只能包含占位值，例如 `change-me`，不得出现真实可用的凭据
- SQLite 文件目录只授予应用所需的读写权限；未来接入第三方服务时，其凭据遵循最小权限原则

## Sensitive Variables

当前由 `.env` 注入的环境变量属于敏感配置，禁止外泄到日志、错误响应或前端构建产物：

- `CORS_ORIGINS`
- 任何未来加入的第三方凭据

Docker Compose 通过 `${VAR}` 从 `.env` 注入；Compose 文件本身只出现变量名，不出现明文值。

## Sensitive Data Handling

敏感信息包括但不限于：

- API keys、access tokens、refresh tokens
- 本地 SQLite 数据文件
- 私有证书
- 用户身份信息（未来加入时）
- 商业或客户敏感数据

处理要求：

- 存储时最小化
- 展示时脱敏
- 传输时使用安全通道
- 调试时避免完整 dump
- 错误响应通过 `internal/problem` 输出，不附带堆栈或内部主机名

## AI Tool Boundaries

- 不把生产密钥直接暴露给 AI
- 不把大批量敏感数据作为提示上下文
- 如果必须提供样本，优先使用脱敏或合成数据
- 使用 AI 生成的代码或文档时，重新检查是否泄露环境变量名、内部主机名或敏感流程

## Config And Secret Placement

权威位置：

- 凭据来源：本地 `.env`，由用户从 `.env.example` 复制
- 后端读取入口：`backend/internal/config/config.go`
- Compose 注入点：`docker-compose.example.yaml` 的 `${VAR}` 占位
- 必须加入 `.gitignore` 的路径：`.env`、`data/`、`backend/tmp/`

## When Adding Auth

未来加入任何形式的身份认证或外部凭据时，必须同步：

1. 在 `backend/internal/middleware/` 或独立包中实现服务端强制执行
2. 更新 `docs/architecture/auth-and-permissions.md`
3. 更新 `docs/api/conventions.md` 的 trust model 段
4. 评估是否需要更新 `route-index.md` 的公开边界列
5. 补行为级测试覆盖权限决策点
6. 复查 `.gitignore` 与 `.env.example`

## Review Checklist

- 是否新增了敏感配置路径
- 是否引入了日志泄露风险
- `.env.example` 是否只包含占位值
- 是否引入默认凭据或示例密钥
- 是否更新了相关运行与部署文档
