---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-04-14
source_of_truth:
  - docs/README.md
  - docs/ai/repo-rules.md
---

# Documentation Governance Standard

## Scope

本标准管理仓库级 canonical 文档，包括：

- 根目录代理入口文件
- `docs/` 下的所有规范性文档
- 需要被 AI 和人类共同消费的当前事实文档

## Core Rules

- 每份文档只能有一个主要职责
- 每类事实只能有一个 canonical source
- 当前实现、工程规则、未来提案必须分离
- 不建立第二套并行文档树
- 根 `README.md` 可以做 onboarding，但不是工程规则源
- 子目录 `README.md` 只能做局部说明，不能重新定义仓库级规则

## Canonical Source 原则

当一类信息需要被多个文档引用时，必须指定唯一主源，其他文档只引用，不复制维护。

常见映射示例：

- 仓库级规则 -> `docs/ai/repo-rules.md`
- 测试命令和覆盖要求 -> `docs/standards/testing-and-verification.md`
- 完成门禁 -> `docs/standards/pre-completion-checklist.md`
- 架构边界 -> `docs/architecture/*.md`
- 当前功能行为 -> `docs/features/current/*.md`

## Front Matter

所有 canonical 文档必须包含：

```yaml
status:
doc_type:
owner:
last_reviewed:
source_of_truth:
```

推荐补充：

- `related_docs`
- `supersedes`

状态约定：

- `active`
- `draft`
- `deprecated`
- `template`

## Audience Convention

- 面向 AI 的文档优先写边界、约束、入口和行为
- 面向人类的文档优先写流程、命令和排障方法
- 如果一份文档同时服务两类受众，也必须先确定主要受众，避免内容同时向两个方向扩张

## When To Create A New Document

应该新建文档的场景：

- 新功能模块已经足够稳定，值得独立维护
- 一条工程标准会被多个模块重复引用
- 一个未实现提案需要持续讨论

不应该新建文档的场景：

- 只是重复已有规则
- 只是换个角度重写同一套接口
- 只是记录临时讨论结论
- 只是创建另一个 README 去竞争解释权

## Anti-Patterns

避免：

- 在多个 README 中复制同一套规则
- 在 feature 文档中重写标准文档里的通用约束
- 在 architecture 文档里写未实现设计
- 在 draft 文档里用当前实现口吻写内容
- 在模板文件里放具体项目事实

