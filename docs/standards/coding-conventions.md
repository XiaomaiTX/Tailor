---
status: active
doc_type: standard
owner: engineering
last_reviewed: 2026-11-05
source_of_truth:
  - docs/ai/repo-rules.md
  - docs/standards/dependency-layering.md
  - frontend/components.json
  - frontend/package.json
---

# Coding Conventions

## Scope

本标准定义本仓库的高层工程约束，不替代 Go / TypeScript 语言级 style guide。  
语言级 lint 由 `make backend-lint` 与 `make frontend-lint` 强制执行。

## Naming

- 文件、模块、目录命名保持稳定和可搜索
- 名称按职责命名，而不是按临时结论命名
- 避免 `new-`、`final-`、`latest-`、`temp-` 等容易过期的命名
- Go 包名遵循短小、单数、全小写；TypeScript 模块按业务域或层划分

## Boundaries

- 业务逻辑放在 `service` 层，不放在 handler / router / middleware / page
- handler / 路由层负责入参解析、调用 service、把结果映射成 generated 类型
- 前端业务调用必须走 `@/api/generated/client` 暴露的 React Query hooks
- 公共契约放在 `api/openapi/openapi.yaml`，不在页面、handler 或单个特性内部
- 生成代码（`backend/internal/api/generated/`、`frontend/src/api/generated/`）不可手改

## shadcn/ui Components

- `frontend/src/components/ui/` 中的组件必须由官方 `shadcn` CLI 从 shadcn/ui 官方 registry 添加或更新，并遵循 `frontend/components.json` 的项目配置（当前 style 为 `aria-sera`）。
- 新增组件使用项目安装的 CLI，例如在 `frontend/` 中运行 `pnpm exec shadcn add checkbox`；不得手工新建、复制第三方实现或从其他 registry 安装到此目录。
- 不得手工修改该目录内组件的 Tailwind 类、CSS、尺寸、颜色、间距、排版、边框或其他视觉表现。页面布局和呈现需求应在调用组件的页面或 feature 层组合；基础组件需要更新时，先用 CLI 的 `--diff` 检查，再通过官方 CLI 更新。
- 为满足项目集成需要而增加的非视觉适配（如国际化标签或可访问名称）必须保持最小范围，且不得改写官方视觉样式。
- CLI 生成或更新后，审阅其文件差异，并按前端验证规范运行适用检查。

## Error Handling

- 错误统一通过 `internal/problem` 封装为 RFC 7807 风格 `application/problem+json`
- handler 收到 service 错误后用 `service.ErrValidation` / `service.ErrNotFound` 等哨兵错误映射到对应 Problem
- 不要吞错；不要用模糊的通用异常掩盖上下文
- 外部系统错误保留必要诊断信息，但避免泄露敏感数据
- 前端通过 `ApiProblemError` 携带 Problem 详情，UI 用 `problemMessageKey` 翻译

## Logging And Observability

- 后端使用 Zap；`middleware.AccessLogger` 记录每个请求的方法、路径、状态、延迟与 `request_id`
- 请求 ID 由 `middleware.RequestID` 注入并写回响应头 `X-Request-ID`
- 不在日志中输出密钥、token、密码或完整敏感 payload
- 日志级别受 `LOG_LEVEL` 控制（`debug` / `info` / `warn` / `error`）

## Config

- 配置集中在 `backend/internal/config/config.go` 加载与校验
- 必填环境变量缺失或格式不合法时进程直接退出，并打印明确错误
- 环境变量包括：`API_PORT`、`LOG_LEVEL`、`DATABASE_PATH`、`CORS_ORIGINS`
- 前端构建期变量 `VITE_API_BASE_URL` 在 `vite.config.ts` 通过 `define` 注入
- 不在代码里散落第二套配置读取逻辑

## OpenAPI Generate Flow

1. 修改 `api/openapi/openapi.yaml`
2. 运行 `make openapi-lint`
3. 运行 `make generate`（生成 Go 桩与 React 客户端）
4. 在 handler 中实现新的 `StrictServerInterface` 方法
5. 在前端通过 generated hook 调用
6. 运行 `make generate-check` 确认无未提交漂移

## Frontend Internationalization

- 所有用户可见文案必须走 `frontend/src/i18n/messages/{zh-CN,en-US}.ts`
- `useI18n()` 是消费文案的唯一入口
- 不要新增硬编码文案而不更新词条
- 日志和内部异常消息是否本地化按需决定，但默认中文

## Commit Message

- 标题必须使用英文，建议采用 Conventional Commits：`<type>: <english title>`
- 内容应精简干练，避免冗长罗列与重复表述
- 描述必须先写英文，再写中文
- 英文描述需先有一句简短清晰的变更动机总结（为什么要改），然后用无序列表补充关键改动与影响
- 中文描述需与英文描述语义对应，同样先写改动动机，再列出关键改动与影响
- 不要只写"改了什么"，必须交代"为什么改"和预期收益或要解决的问题
- 避免空泛描述（如 `update code`、`fix issues`），必须体现本次改动范围

示例：

```text
feat: add startup diagnostics panel for faster local troubleshooting
Speed up startup issue triage during local development:
- Show .NET SDK and Windows runtime details at startup
- Surface current config source and initialization status
- Add one-click copy for diagnostics output

This reduces repeated environment checks and shortens debug cycles.

为加快本地排障新增启动诊断面板：
- 启动时展示 .NET SDK 与 Windows 运行时信息
- 展示当前配置来源与初始化状态
- 提供诊断信息一键复制

该改动减少重复环境排查，缩短调试周期。
```

## Prohibitions

禁止：

- 越层调用（handler 直连 GORM、service 反向依赖 router 等）
- 手改 generated 目录
- 影子配置或第二套环境变量读取
- 一次性抽象但没有复用场景
- 复制粘贴业务规则
- 缺乏上下文的 TODO 或注释掉的大段旧代码
- 在前端页面或组件中硬编码用户可见文案
