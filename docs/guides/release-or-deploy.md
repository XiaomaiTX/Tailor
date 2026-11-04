---
status: active
doc_type: guide
owner: engineering
last_reviewed: 2026-11-04
source_of_truth:
  - docker-compose.example.yaml
  - backend/Dockerfile
  - frontend/Dockerfile
  - Makefile
---

# Release Or Deploy Guide

## Purpose

说明 Go React Starter 当前的发布与部署方式。

## Scope

当前仓库只提供 Docker Compose 一键交付路径。**没有 CI/CD、镜像仓库、生产平台运行手册**；未来需求进入 `docs/specs/draft/`。

## Preconditions

- 已通过 `docs/standards/pre-completion-checklist.md`
- 已确认本地或部署机能运行 Docker Compose
- 已准备 `.env`（从 `.env.example` 复制并按部署环境调整配置）

## Steps

### 1. 准备环境变量

```bash
cp .env.example .env
# 按部署环境调整 .env 中的配置
```

### 2. 构建并启动

```bash
docker compose --env-file .env -f docker-compose.example.yaml up --build
```

启动顺序由 Compose 编排：

1. `backend` 启动时按 GORM 模型初始化 SQLite 表结构，`/api/v1/health/live` 通过后视为健康
2. `frontend` 等待 `backend` 健康后启动，Nginx 监听 `$FRONTEND_PORT`

### 3. 注入运行时配置

所有运行时配置通过 `.env` 注入到 Compose：

- `API_PORT`、`LOG_LEVEL`、`CORS_ORIGINS`
- `DATABASE_PATH`（容器内固定为 `/data/app.db`）
- `VITE_API_BASE_URL`（前端构建期注入）
- `FRONTEND_PORT`（Nginx 对外端口）

### 4. 验证发布结果

- `curl http://localhost:$FRONTEND_PORT/` 应返回前端首屏
- `curl http://localhost:$FRONTEND_PORT/api/v1/health/live` 应返回 `200 { "status": "ok" }`
- `curl http://localhost:$FRONTEND_PORT/api/v1/health/ready` 在 SQLite 数据库可用时返回 `200`

## Expected Outcome

- 浏览器访问 `http://localhost:$FRONTEND_PORT` 看到前端
- `/api/v1/examples` 可被访问；`/examples` 页面可完成 CRUD

## Troubleshooting

- **SQLite 表结构初始化失败导致 backend 不启动**：检查容器日志、数据库文件路径及 `/data` 卷的读写权限
- **端口占用**：检查 `FRONTEND_PORT`、`API_PORT` 是否被占用
- **healthcheck 不通过**：`backend` 容器日志中查 Zap 输出与 `request_id`
- **CORS 拒绝**：浏览器访问域名未列入 `CORS_ORIGINS`
- **回滚**：当前没有镜像仓库或自动回滚；可通过 `docker compose down`、修复 `.env` / 数据库文件，再次 `up --build` 完成

## Notes

- 本地开发流程见 `docs/guides/local-development.md`
- 安全与凭据边界见 `docs/standards/security-and-secrets.md`
