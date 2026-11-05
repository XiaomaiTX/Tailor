---
status: active
doc_type: index
owner: engineering
last_reviewed: 2026-11-05
source_of_truth:
  - AGENTS.md
  - docs/ai/repo-rules.md
  - docs/standards/documentation-governance.md
---

# 文档索引

## 目的

`docs/` 是 Tailor 的 canonical 文档树，用来把仓库规则、当前架构、接口契约、功能行为、人类操作指南、未来规划和模板明确分层，避免 AI 与人类从多个入口读取互相冲突的信息。

根目录 `README.md` 是 onboarding 入口，但不是工程规则源。  
代理入口文件如 `AGENTS.md` / `CLAUDE.md` 只做规则委托，不单独承载规则正文。

## 项目定位

**Peer Relay management for Headscale-managed Tailscale networks**

Tailor 连接单个 Headscale 实例，同步 Tailscale 设备与 Policy，并提供 Peer Relay 注册、申请审批和设备授权关系的查看与编辑。Tailor 管理 Policy 中的 Relay Grants；它不会启动设备上的 Relay 服务，也不显示实时流量路径。技术栈为 Gin、GORM/SQLite、Zap、OpenAPI、React、React Query、Tailwind 与 shadcn 风格组件。`/examples` 是保留的生成式 CRUD 示例 API，不是主业务能力。

## Authority Order

当多个文档描述同一件事且存在冲突时，权威顺序如下：

1. `docs/ai/repo-rules.md`
2. `docs/standards/*.md`
3. `docs/architecture/*.md`
4. `docs/api/*.md`
5. `docs/features/current/*.md`
6. `docs/guides/*.md`
7. `docs/specs/draft/*.md`
8. `docs/templates/*.md`

补充规则：

- `docs/specs/draft/*.md` 只表示提案或未实现设计，不能覆盖当前实现
- `docs/templates/*.md` 是模板，不描述当前项目事实
- 代码是当前实现；如果代码和文档冲突，按 `docs/ai/repo-rules.md` 中的 code-vs-docs 规则处理

## 受众分层

### AI 优先文档

这些文档主要服务 AI 代理和自动化工具：

- `docs/ai/`
- `docs/standards/`
- `docs/architecture/`
- `docs/api/`
- `docs/features/current/`

### 人类优先文档

这些文档主要服务开发者与维护者：

- `docs/guides/`
- 根目录 `README.md`

### 通用文档

这些文档服务两类受众，但必须明确状态和用途：

- `docs/reference/`
- `docs/specs/draft/`
- `docs/templates/`

## 目录职责

| 路径 | 类型 | 作用 |
| --- | --- | --- |
| `docs/ai/` | agent rules / guide | 仓库级 AI 规则、AI 上手顺序、任务路由 |
| `docs/standards/` | standard | 开发规范、文档治理、测试与完成门禁 |
| `docs/architecture/` | architecture | 当前系统结构、模块图、运行方式、数据与契约边界 |
| `docs/api/` | api | API / command / IPC 契约约定与索引 |
| `docs/features/current/` | feature | 当前已落地功能行为说明 |
| `docs/features/archived/` | archive | 已归档功能说明 |
| `docs/guides/` | guide | 面向人类开发者的操作指南 |
| `docs/specs/draft/` | draft | 未实现设计、提案和开放问题 |
| `docs/specs/archive/` | archive | 已废弃或已完成的历史提案 |
| `docs/reference/` | reference | 非权威参考资料 |
| `docs/templates/` | template | 新建文档模板 |

## 推荐阅读顺序

### 对 AI 代理

1. `AGENTS.md` 或其他代理入口文件
2. `docs/ai/repo-rules.md`
3. 本文档
4. `docs/ai/agent-onboarding.md`
5. 按任务类型继续读取相关 `standards`、`architecture`、`api`、`features`

### 对人类开发者

1. 根目录 `README.md`
2. 本文档
3. `docs/standards/documentation-governance.md`
4. `docs/ai/repo-rules.md`
5. 任务对应的架构、接口、功能和操作指南

新接手 Peer Relay 功能时，继续阅读 `docs/features/current/peer-relay-manager.md`、`docs/architecture/auth-and-permissions.md` 和 `docs/guides/release-or-deploy.md`。

## 文档放置规则

- 仓库级规则放在 `docs/ai/` 或 `docs/standards/`
- 当前架构事实放在 `docs/architecture/`
- 接口契约放在 `docs/api/`
- 当前功能行为放在 `docs/features/current/`
- 面向人类的步骤式指南放在 `docs/guides/`
- 未实现设计只放在 `docs/specs/draft/`
- 模板只放在 `docs/templates/`

禁止：

- 在多个目录重复维护同一类规则
- 用局部 `README.md` 重新定义仓库级规则
- 在 current 文档里混入未实现设计
- 在 draft 里写成当前实现口吻

## Front Matter 约定

所有 canonical 文档必须使用 YAML front matter，至少包含：

```yaml
status:
doc_type:
owner:
last_reviewed:
source_of_truth:
```

推荐 `doc_type`：

- `agent-rules`
- `agent-guide`
- `index`
- `standard`
- `architecture`
- `api`
- `feature`
- `guide`
- `reference`
- `draft`
- `template`

补充约定：

- 模板文档必须使用 `status: template`
- 提案文档必须使用 `status: draft`
- 状态或作用变化时必须更新 `last_reviewed`
