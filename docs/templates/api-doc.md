---
status: template
doc_type: template
owner: engineering
last_reviewed: 2026-08-05
---

# API 文档模板

> 仅作为模板使用。把它当作 canonical 接口文档之前，请替换所有占位内容。模板来源：`docs/standards/doc-writing-standard.md` 与 `docs/api/conventions.md`。

## 范围

- <在此描述本文件覆盖的接口集合>

## 边界类型

- 接口类型：<HTTP / CLI / IPC / 事件 / 插件>
- 基础路径或命名空间：<base path / namespace / command root>
- 认证或信任模型：<anonymous / token / mTLS / 用户角色>

## 接口表

| 类型 | 标识符 | 用途 | 边界 | 消费方 |
| --- | --- | --- | --- | --- |
| <http\|command\|ipc\|event> | <路径或命令名> | <用途> | <公开或受限> | <调用方> |

## 请求 / 响应说明

- 输入编码：<JSON / binary / form>
- 输出组织：<分页结构 / 流式 / 单对象>
- 错误表示：<RFC 7807 problem+json / 自定义错误码>

## 同步要求

- 更新生产方契约
- 更新共享 schema 或类型
- 更新调用封装层
- 更新消费方
- 更新文档索引与相关 feature 文档
