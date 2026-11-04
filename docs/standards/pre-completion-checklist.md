---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-08-05
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/standards/testing-and-verification.md
---

# Pre-Completion Checklist

## Scope

在任务标记完成前使用本清单。只跳过明显不适用项。

## Core Rules

- 这是统一的完成门禁
- 测试命令和覆盖规则以 `docs/standards/testing-and-verification.md` 为准
- 如有跳过项，必须说明原因
- 任意代码变更后，必须完成“文档影响检查”和“测试影响检查”并记录结论；未记录结论视为未通过门禁

## Mandatory Doc/Test Impact Check (All Code Changes)

- [ ] 已评估并处理文档影响（`docs/features/current/`、`docs/api/`、`docs/architecture/`、`docs/guides/`）
- [ ] 已评估并处理测试影响（新增/更新测试，或记录可审计的豁免原因）
- [ ] 交付说明中已记录两项结论（更新了什么，或为何无需更新）

## Documentation-Only Change

- [ ] 作用范围清晰，没有夹带无关变更
- [ ] front matter 完整且状态正确
- [ ] 交叉引用和路径没有过期
- [ ] 涉及当前实现时已经对照过代码
- [ ] 已完成测试影响检查；如未新增/更新测试，已记录原因

## Backend / Core Logic Change

- [ ] 运行适用的 build / test / lint
- [ ] 没有引入新的越层依赖
- [ ] 业务逻辑仍位于业务层
- [ ] 如为 bug fix，已补充或更新回归测试
- [ ] 如行为变化，已更新相关 feature 或 architecture 文档
- [ ] 已在交付说明中记录“文档影响结论”和“测试影响结论”

## Frontend / UI Change

- [ ] 运行适用的 build / test / lint / typecheck
- [ ] 没有把复杂业务逻辑塞进 UI 层
- [ ] 用户可见文案遵守国际化约定
- [ ] 如行为变化，已更新相关 feature 文档
- [ ] 已新增/更新对应测试；若无更新，已记录合理原因
- [ ] 已在交付说明中记录“文档影响结论”和“测试影响结论”

## Cross-Contract Change

- [ ] 生产方契约已更新
- [ ] 调用封装层已更新
- [ ] 共享类型或 schema 已更新
- [ ] 消费方已更新
- [ ] `docs/api/` 已同步
- [ ] 相关 feature 或 architecture 文档已同步
- [ ] 已新增/更新契约相关测试；若无更新，已记录合理原因
- [ ] 已在交付说明中记录“文档影响结论”和“测试影响结论”

## Permission / Security Change

- [ ] 已完成 Cross-Contract Change 所需检查
- [ ] 权限边界已经在生产侧强制执行
- [ ] 敏感数据处理符合 `security-and-secrets.md`
- [ ] 相关权限 / 安全文档已更新

## New Feature Or Module

- [ ] 已定义模块入口与职责
- [ ] 已添加 `docs/features/current/` 文档或更新现有文档
- [ ] 已补充接口 / 架构文档
- [ ] 已添加至少一层合理的行为级验证，除非明确说明例外
- [ ] 已在交付说明中记录“文档影响结论”和“测试影响结论”

## If A Test Is Skipped

必须明确写出：

1. 跳过了什么测试
2. 为什么跳过
3. 后续应在哪一层或哪个位置补上
