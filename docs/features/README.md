---
status: active
doc_type: index
owner: engineering
last_reviewed: 2026-08-05
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/standards/doc-writing-standard.md
---

# Features Index

## Purpose

`docs/features/` 用来描述 Go React Starter 当前已落地行为，而不是技术实现细节或未来规划。

`/examples` 是可被替换的端到端技术链路示例资源，被记录在本目录下作为基座示例。它不是业务能力；真实业务模块落地时应在本目录新增独立 feature 文档，而不是复用示例文档。

## Structure

- `current/`：当前有效的功能文档
- `archived/`：已归档、被替代或仅供历史参考的功能文档

## Current Features

- [Example 资源链路](current/example-feature.md)：技术基座示例，覆盖 OpenAPI → handler → service → repository → React 调用方的端到端 CRUD 与健康探针

## Rules

- 当前行为只写在 `current/`
- 被替代或失效的文档移入 `archived/`
- 未实现设计不要放到 `features/`，而应放到 `docs/specs/draft/`
- 第一个业务模块落地时，应在本目录新增独立文档，并把"示例资源"明确区分为基座示例
