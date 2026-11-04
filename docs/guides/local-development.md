---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - docs/architecture/runtime-and-startup.md
  - docs/standards/testing-and-verification.md
  - Makefile
  - docker-compose.example.yaml
---

# Local Development Guide

## Purpose

描述在本地（非容器）环境下如何启动 Go React Starter、开发后端与前端，以及如何最小化验证。

## Preconditions

- Go 1.27.3（`go version` 验证）
- 可用的 C 编译器（SQLite Go 驱动通过 CGO 构建）
- Node.js 24.x（`.nvmrc` 指定大版本；本机若已安装 v22.x，请用版本管理器切换）
- pnpm 12.17.0（`frontend/package.json` 的 `packageManager` 字段）
- Air v1.65.1（后端热重载；由 `make backend-dev` 经 `go run` 自动拉取，无需手动安装）
- SQLite 使用本地文件，无需数据库服务
- Windows / PowerShell 用户：确保 Go 与 Node 工具链在 `PATH` 中

## Steps

### 1. 准备环境变量

```powershell
cp .env.example .env
```

把 `.env` 中的 `DATABASE_PATH` 改为所需 SQLite 文件路径，并按需调整 `CORS_ORIGINS`。
`CORS_ORIGINS` 不能为空，至少包含前端开发地址，例如 `http://localhost:5173,http://localhost:8080`。

### 2. 安装前端依赖

```powershell
cd frontend
pnpm install --frozen-lockfile
```

### 3. 启动后端（热重载）

```powershell
make backend-dev
```

`make backend-dev` 经 `go run` 拉起 Air（首次运行需外网拉取模块）。Air 通过 `backend/.air.toml` 加载 `../.env`，并把构建产物放在 `backend/tmp/`。修改 Go 源码后 Air 会自动重新编译并重启后端；`.env` 变更需重启 Air。

### 4. 启动前端（热更新）

```powershell
cd frontend
pnpm dev
```

Vite 监听 `http://localhost:5173`，并把 `/api` 代理到 `http://localhost:8080`。

## Expected Outcome

- 后端监听 `:$API_PORT`（默认 8080）
- `GET http://localhost:8080/api/v1/health/live` 返回 `200 { "status": "ok" }`
- `GET http://localhost:8080/api/v1/health/ready` 返回 `200`，前提是 SQLite 数据库可用
- 浏览器访问 `http://localhost:5173/examples` 可以加载 Example 页面并完成 CRUD

## Troubleshooting

- **后端启动报 `DATABASE_PATH 不能为空` / `CORS_ORIGINS 不能为空`**：检查 `.env` 是否存在并完成配置；Air 是否从 `backend/` 目录启动以加载 `../.env`
- **`LOG_LEVEL 必须为 debug、info、warn 或 error`**：`.env` 中的 `LOG_LEVEL` 写错或留空
- **`/health/ready` 返回 500**：检查 SQLite 文件路径和目录权限
- **SQLite 表结构初始化失败**：检查 `DATABASE_PATH` 是否指向可写文件，以及其父目录是否可创建
- **前端 Vite 代理 404**：确认后端已启动在 8080；如改过 `API_PORT`，需要同步修改 `frontend/vite.config.ts` 的 `proxy`
- **CORS 报错**：浏览器 Origin 不在 `CORS_ORIGINS` 列表里
- **首次 `make backend-dev` 卡住或失败**：`go run` 需要外网拉取 `air` 模块，确认网络可达后重试
