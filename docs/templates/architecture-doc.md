---
status: template
doc_type: template
owner: engineering
last_reviewed: 2026-08-05
---

# 架构文档模板

> 仅作为模板使用。把它当作当前实现架构文档之前，请替换所有占位内容。模板来源：`docs/standards/doc-writing-standard.md` 与 `docs/architecture/overview.md`。

## 范围

- <在此描述本文件覆盖的子系统或边界>

## 当前实现

- <在此描述已落地的结构、运行形态与依赖>

## 关键入口文件

- <应用入口>
- <模块装配入口>
- <运行时配置入口>

## 不变量

- <示例：所有写操作都经过 application service>
- <示例：所有外部调用都封装在 adapter 层>
