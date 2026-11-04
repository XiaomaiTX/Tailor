---
status: active
doc_type: agent-guide
owner: engineering
last_reviewed: 2026-08-05
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/README.md
---

# AI Agent Onboarding

## Purpose

给 AI 代理提供最短的启动路径和任务路由。  
仓库级规则以 `docs/ai/repo-rules.md` 为准。

## Startup

1. 读取代理入口文件，如 `AGENTS.md`
2. 读取 `docs/ai/repo-rules.md`
3. 读取 `docs/README.md`
4. 判断本次任务属于哪一类
5. 只读取与该任务直接相关的文档，再开始修改代码或文档

## Change Routing

### 规则、流程或治理调整

先读：

1. `docs/standards/documentation-governance.md`
2. 目标标准文档
3. `docs/standards/pre-completion-checklist.md`

### 架构、运行时、模块边界调整

先读：

1. `docs/architecture/overview.md`
2. `docs/architecture/module-map.md`
3. `docs/architecture/runtime-and-startup.md`
4. `docs/standards/dependency-layering.md`

### 接口、命令、事件或 IPC 调整

先读：

1. `docs/api/conventions.md`
2. `docs/api/route-index.md`
3. `docs/architecture/data-and-contracts.md`
4. 相关 feature 文档

### 功能行为调整

先读：

1. `docs/features/README.md`
2. 相关 `docs/features/current/*.md`
3. 必要的架构与接口文档

### 调试、验证或发布问题

先读：

1. `docs/standards/testing-and-verification.md`
2. `docs/standards/pre-completion-checklist.md`
3. `docs/guides/debugging-guide.md`
4. `docs/guides/testing-guide.md`

## Conflict Resolution

当多个来源不一致时：

1. 按 `docs/ai/repo-rules.md` 中的 authority order 判定
2. 如果代码和文档冲突，按 code-vs-docs rule 处理
3. 不把 `docs/specs/draft/` 或 `docs/templates/` 当成当前事实

## Documentation Updates

以下变化通常需要同步改文档：

- 行为变化 -> 更新相关 feature 文档
- 接口变化 -> 更新 `docs/api/`
- 架构边界变化 -> 更新 `docs/architecture/`
- 测试策略变化 -> 更新 `docs/standards/testing-and-verification.md`
- 完成门禁变化 -> 更新 `docs/standards/pre-completion-checklist.md`
- 仓库级规则变化 -> 更新 `docs/ai/repo-rules.md`

任务收尾前必须输出“文档影响结论”和“测试影响结论”：

- 文档影响结论：更新了哪些文档；若未更新，写明无需更新原因
- 测试影响结论：新增/更新了哪些测试；若未更新，写明无需更新原因

## Verification

完成前统一使用：

- `docs/standards/testing-and-verification.md`
- `docs/standards/pre-completion-checklist.md`

交付说明必须包含以下统一格式（缺失则视为未通过 pre-completion gate）：

```text
Documentation Impact:
- Updated: <files/sections>
  或
- No Update Needed: <reason>

Test Impact:
- Updated: <tests/cases>
  或
- No Update Needed: <reason>
```

## Notes

- `docs/templates/` 是模板，不描述当前事实
- `docs/specs/draft/` 是提案，不等于已实现
- 面向 AI 的文档应尽量简洁，避免重复代码中可以直接推导的信息
