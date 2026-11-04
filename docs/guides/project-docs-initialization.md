---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-04-14
source_of_truth:
  - docs/README.md
  - docs/ai/repo-rules.md
  - docs/standards/documentation-governance.md
---

# Project Docs Initialization Guide

## Purpose

本指南说明如何把这套初始化包复用到新的 LLM 介入项目中。

## 适用项目

适用于：

- Web 全栈
- C# Windows 客户端
- Node.js CLI
- 服务端
- 自动化工具或内部平台

只要项目满足以下条件，就适合使用：

- AI 会直接参与实现、审查、调试或文档维护
- 需要单一权威入口
- 希望把规则、当前事实和未来规划分开维护

## Must Fill First

新项目落地时，优先填写：

1. `docs/ai/repo-rules.md`
2. `docs/README.md`
3. `docs/standards/testing-and-verification.md`
4. `docs/standards/pre-completion-checklist.md`
5. `docs/architecture/overview.md`
6. `docs/architecture/module-map.md`

这六类文件决定：

- AI 会先读什么
- 什么文档算权威
- 默认验证命令是什么
- 当前系统怎么被理解

## Can Fill Later

以下内容可在项目逐步稳定后补齐：

- `docs/features/current/*.md`
- `docs/api/route-index.md`
- `docs/guides/release-or-deploy.md`
- `docs/specs/draft/*.md`
- `docs/reference/*`

## How To Replace Placeholders

替换占位内容时，优先补这些事实：

- 项目名称和目标
- 技术栈
- 目录结构
- 核心模块
- 认证和权限模型
- 外部依赖
- 默认验证命令
- 当前稳定接口

保留这些结构不变：

- `AGENTS.md -> docs/ai/repo-rules.md`
- authority order
- docs 分层
- front matter 约定
- current / draft / template 分离

## Tech Stack Migration

迁移到不同技术栈时，通常只需要改：

- `docs/standards/dependency-layering.md`
- `docs/standards/testing-and-verification.md`
- `docs/architecture/overview.md`
- `docs/architecture/runtime-and-startup.md`
- `docs/api/*`

一般不需要改：

- 文档治理结构
- authority order
- 完成门禁的基本结构

## When To Promote A Draft

满足以下条件时，可以把 draft 升级为 active canonical 文档：

- 设计已落地
- 行为已经稳定
- 调用边界已明确
- 后续维护者需要把它当作当前事实阅读

升级方式：

1. 将文件从 `docs/specs/draft/` 移到正确目录
2. 改 `status`
3. 改 `doc_type`
4. 更新 `last_reviewed`
5. 清理旧引用

## Suggested Rollout

1. 复制整个初始化包
2. 先填仓库级规则和测试门禁
3. 再填架构和接口
4. 最后按模块补 feature 文档
5. 形成“行为变更必须带文档更新”的工作习惯

