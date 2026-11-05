---
status: active
doc_type: agent-rules
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - AGENTS.md
  - docs/README.md
  - docs/standards/documentation-governance.md
---

# Repository Rules

## Scope

本文件定义仓库范围内的 AI 工作规则、权威顺序、上下文边界和完成要求。  
详细标准、架构事实、接口契约、功能行为和操作指南分别维护在 `docs/` 的对应目录。

## Authority Order

当规则冲突时，按以下顺序判定：

1. 本文件
2. `docs/standards/*.md`
3. `docs/architecture/*.md`
4. `docs/api/*.md`
5. `docs/features/current/*.md`
6. `docs/guides/*.md`
7. `docs/specs/draft/*.md`
8. `docs/templates/*.md`

## Context Boundaries

权威上下文仅限于已提交或当前会话中明确提供的项目资产：

- code
- docs
- schemas
- config
- migrations
- tests
- checked-in generated types

默认不把以下内容视为权威来源，除非用户在当前会话中明确提供：

- Slack
- 飞书 / Google Docs
- 口头决定
- issue 评论中的未落地结论
- 私聊记录

## Code-vs-Docs Rule

代码是当前实现。

如果代码与文档冲突，按以下顺序处理：

1. 检查相关代码路径和调用链
2. 判断文档是否过期，或代码是否偏离了预期
3. 如果当前行为是有意设计，更新过期文档
4. 只有在用户明确要求或文档明显代表目标行为时，才修改代码去匹配文档

## Project Intent

本仓库实现 Peer Relay Manager：基于单个 Headscale 实例管理 Peer Relay 注册、申请审批与可视化 Policy Grants。工程链路仍遵循“OpenAPI 契约 + 分层 Go 后端 + 生成的 React 客户端 + React SPA”。

当前项目事实：

- 技术栈：Go 1.27.3（Gin、GORM、SQLite、Zap）、React 19（Vite、React Query、Tailwind、shadcn 风格组件）、OpenAPI 3.0.3
- 目录结构：`api/openapi`、`backend/{cmd,internal}`、`frontend/src`、`docs`、Compose 示例
- 核心模块：`config`、`router`、`handler`、`service`、`repository`、`model`、`middleware`、`problem`、`headscale`、`policy`
- 身份边界：Tailor 使用外部 OIDC 提供方完成登录，并将唯一匹配的已验证邮箱映射到当前 Headscale 用户；业务 API 使用 SQLite 服务端会话，健康探针公开
- 接口边界：`/api/v1` JSON HTTP API；详见 `docs/api/conventions.md` 与 `docs/api/route-index.md`
- Policy 写入默认关闭；仅在确认 Headscale 数据库存储 Policy 模式后配置启用
- 验证命令：`make verify`；详见 `docs/standards/testing-and-verification.md`

## Non-Negotiable Rules

1. **单一权威入口**
   - 代理入口文件只委托给 `docs/ai/repo-rules.md`
   - 不建立第二套仓库级规则入口

2. **当前事实、规则、规划分离**
   - current facts 不写进 draft
   - future proposals 不写进 architecture 或 feature current

3. **契约必须同步**
   - 跨边界接口变化必须同步更新实现、调用方、类型和相关文档

4. **测试策略集中**
   - 测试命令、覆盖策略、允许例外统一维护在 `docs/standards/testing-and-verification.md`

5. **完成门禁集中**
   - 完成前统一参考 `docs/standards/pre-completion-checklist.md`

6. **文档随行为变更**
   - 行为、路由、权限、运行方式、验证方式变化时，同步更新相关文档

7. **代码变更后强制执行文档与测试更新检查（MUST）**
   - 任意代码变更（新增功能、修复、删改、重构）完成后，必须执行“文档影响检查”和“测试影响检查”
   - 若无需更新文档或测试，也必须在交付说明中记录“不更新原因”
   - 未完成该检查并记录结论，不得视为任务完成

8. **优先既有模式**
   - 引入新抽象前先检查项目已有模式

9. **AI 只基于权威上下文工作**
   - 不把推测、草案或历史讨论当成当前事实

10. **限制范围**
   - 不把无关重构混入功能开发或 bug 修复

11. **安全边界不可弱化**
   - 密钥、权限、脱敏、外部系统凭据必须遵守 `docs/standards/security-and-secrets.md`

## Documentation Routing

不熟悉项目时，先读：

- `docs/README.md`
- `docs/ai/agent-onboarding.md`
- `docs/standards/documentation-governance.md`

之后按任务继续路由：

- 改规则 -> `docs/standards/`
- 改架构边界 -> `docs/architecture/`
- 改接口 -> `docs/api/`
- 改功能行为 -> `docs/features/current/`
- 改人类流程 -> `docs/guides/`
- 做未实现方案 -> `docs/specs/draft/`

## Architecture and Contract Rules

### Layering

必须在项目中明确定义依赖方向，并维护在 `docs/standards/dependency-layering.md`。

默认示例：

- Web：view -> api -> service -> repository -> model
- Desktop：ui -> application -> domain -> infrastructure
- CLI：command -> application -> domain -> adapter

示例结构可替换，但方向性约束不能丢。

### Contract Change Order

接口或边界变更时，按以下顺序同步：

1. 生产方契约或返回结构
2. 相关业务逻辑
3. 调用封装层
4. 共享类型或 schema
5. 消费方
6. 文档索引与行为文档

不要允许 contract drift 长期存在。

## Completion Routing

完成任务前：

- 使用 `docs/standards/pre-completion-checklist.md` 作为完成门禁
- 使用 `docs/standards/testing-and-verification.md` 作为验证命令与覆盖要求来源

## Guardrails

### When Blocked or Looping

如果停滞不前：

1. 回到相关规则和架构文档重新确认边界
2. 再次检查当前实现
3. 更换方法
4. 如果仍被阻塞，再明确说明阻塞点

### Anti-Drift Rules

- 不引入与现有规范冲突的新模式
- 不建立影子文档树
- 不复制已经 canonical 的事实到多个地方维护
- 不留下无上下文 TODO、占位实现或注释掉的大段旧代码

### Divergence Rule

如果多数文件遵循同一模式而少数文件偏离，默认认为偏离需要明确理由，而不是直接复制偏离写法。

## Quick Start

新代理进入项目时：

1. 读 `AGENTS.md`
2. 读本文件
3. 读 `docs/README.md`
4. 读 `docs/ai/agent-onboarding.md`
5. 按任务类型路由到对应文档
