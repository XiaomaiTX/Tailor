---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-04-14
source_of_truth:
  - docs/README.md
  - docs/standards/documentation-governance.md
---

# Doc Writing Standard

## Scope

本标准定义规范性文档的写法、最小结构和更新时机。

## Core Writing Rules

- 先确定文档类型，再开始写内容
- 先写事实和边界，再写解释
- 面向 AI 的文档保持紧凑，避免重复代码中显而易见的信息
- 面向人类的 guide 应强调步骤、前置条件、结果和排障

## Document Types

- `standard`：规则、禁止项、例外、检查项
- `architecture`：当前实现结构、入口文件、运行边界
- `api`：接口契约、认证、返回格式、同步要求
- `feature`：当前功能目的、入口、权限边界、关键不变量
- `guide`：面向人的操作步骤
- `draft`：提案、未实现设计、开放问题
- `template`：供复制的新文档模板

## Minimum Structure

### standard

- Scope
- Core Rules
- Exceptions
- Enforcement or Checklist

### architecture

- Scope
- Current Implementation
- Key Entry Files
- Invariants

### api

- Scope
- Authentication / Boundary
- Interface Index
- Request / Response Notes
- Synchronization Requirements

### feature

- Module Purpose
- Entry Points
- Permission Boundaries
- Key Invariants
- Primary Code Files

### guide

- Purpose
- Preconditions
- Steps
- Expected Outcome
- Troubleshooting

### draft

- Background
- Current Status
- Proposal
- Open Questions
- Explicit Non-Implementation Statement

## What Not To Repeat In Feature Docs

- 通用测试命令
- 仓库级完成门禁
- 整个权限矩阵的完整重复版本
- 已经 canonical 的 API 约定
- 明显可从代码直接推导的大量实现细节

## Update Triggers

以下变化应更新文档：

- 功能行为变化
- 接口变化
- 权限或安全边界变化
- 启动与运行方式变化
- 测试或验证方式变化
- 新增可复用工程标准

