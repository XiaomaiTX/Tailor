---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - docs/architecture/runtime-and-startup.md
  - docs/standards/security-and-secrets.md
  - Makefile
---

# Debugging Guide

## Purpose

给开发者和 AI 一个统一的排查顺序，避免在 Go React Starter 上盲目试错。

## Preconditions

- 已确认目标问题的模块和入口（参考 `docs/architecture/module-map.md`）
- 已读取相关 architecture / feature / api 文档

## Steps

1. 明确问题属于：配置 / 启动、契约 / 生成、运行时 / 业务、外部依赖、前端代理 / CORS、Problem 响应
2. 回到对应入口文件和边界层检查真实调用链
3. 对照相关 feature 或 API 文档判断是代码漂移还是文档过期
4. 运行最小化验证命令（见下文）
5. 若涉及外部依赖，先确认依赖可达性与配置

## Common Paths

### 配置校验失败

- 现象：`config.Load()` 直接退出，stderr 打印中文错误
- 检查：`.env` 是否存在；`DATABASE_PATH` 是否指向可写 SQLite 文件；`CORS_ORIGINS` 是否为空；`LOG_LEVEL` 是否为 `debug` / `info` / `warn` / `error`

### 数据库未就绪

- 现象：`/health/ready` 返回 `500 application/problem+json`
- 检查：SQLite 文件路径和父目录权限；本地 Air 是否正确加载 `.env`

### SQLite 表结构初始化失败

- 现象：后端启动日志报告 SQLite 表结构初始化失败
- 检查：`DATABASE_PATH` 是否指向可写文件；父目录是否存在或可创建；GORM 模型定义是否与当前 SQLite 表兼容

### OpenAPI 与生成代码漂移

- 现象：`make generate-check` 失败，提示 `backend/internal/api/generated` 或 `frontend/src/api/generated` 有未提交差异
- 检查：是否手改了生成目录；是否改了 `openapi.yaml` 但忘了 `make generate`
- 修复：`make generate` 重新生成，提交生成产物

### Vite / Nginx 代理问题

- 本地开发：`frontend/vite.config.ts` 把 `/api` 代理到 `http://localhost:8080`，改过后端 `API_PORT` 需要同步
- 容器：`frontend/nginx.conf` 把 `/api/` 反向代理到 `backend:8080`；`backend` healthcheck 通过后 `frontend` 才启动

### CORS 报错

- 现象：浏览器控制台出现 CORS 错误
- 检查：浏览器 Origin 是否在 `CORS_ORIGINS` 列表；后端响应是否带 `Vary: Origin`；预检请求是否被 `middleware.CORS` 正确处理

### Problem Details

- 现象：客户端收到 `application/problem+json`
- 检查 `type` 字段：
  - `urn:go-react-starter:problem:validation` → 请求参数或 body 不合法
  - `urn:go-react-starter:problem:not-found` → 资源不存在
  - `urn:go-react-starter:problem:internal` → 服务内部错误，查 Zap 日志中的 `request_id`
- 前端 `lib/problem.ts` 用 `problemMessageKey` 把 Problem 翻成 i18n 文案

## Expected Outcome

- 能定位到根因所在层，而不是只看到表面症状
- 留下与 `request_id` 关联的日志证据

## Troubleshooting

- 如果多处逻辑重复，先找 canonical 路径（`problem`、`config`、`generated`）
- 如果文档与代码冲突，按 `docs/ai/repo-rules.md` 的 code-vs-docs rule 处理
