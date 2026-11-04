---
status: template
doc_type: template
owner: engineering
last_reviewed: 2026-08-05
---

# 功能文档模板

> 仅作为模板使用。把它当作当前功能文档之前，请替换所有占位内容。模板来源：`docs/standards/doc-writing-standard.md` 与 `docs/features/README.md`。

## 模块目的

- <在此描述该功能解决什么问题，面向谁>

## 入口

### 用户入口

- <页面 / 路由 / 菜单 / 命令 / 快捷方式>

### 系统入口

- <服务 / 调度任务 / 事件 / Webhook / Job>

## 权限边界

- <谁可读>
- <谁可写>
- <哪些动作需要更高权限>

## 关键不变量

- <关键行为规则>
- <关键数据约束>
- <不允许被破坏的边界>

## 主要代码文件

- <入口或控制器文件>
- <业务或领域文件>
