---
status: active
doc_type: architecture
owner: engineering
last_reviewed: 2026-08-05
source_of_truth:
  - backend/cmd/api/main.go
  - backend/internal/config/config.go
  - backend/.air.toml
  - frontend/vite.config.ts
  - docker-compose.example.yaml
---

# Runtime And Startup

## Scope

描述 Go React Starter 如何启动、装配依赖、加载配置，以及开发与容器两种运行模式的差异。

## Startup Flow

后端（`backend/cmd/api/main.go`）按以下顺序启动：

1. `config.Load()` 读取并校验环境变量，缺失或非法直接 `os.Exit(1)`
2. `newLogger(cfg.LogLevel)` 装配 Zap logger
3. `gorm.Open(postgres.Open(cfg.DatabaseURL))` 建立 PostgreSQL 连接池
4. `redis.NewClient(redisOptions)` 建立 Redis 客户端
5. 装配 `repository → service → handler`，并把 `sqlDatabase.PingContext` 与 `redisClient.Ping` 注入健康探针
6. `router.New(api, cfg, logger)` 构建 Gin 引擎与中间件链
7. `http.Server` 监听 `:$APIPort`，`ReadHeaderTimeout` 为 5s
8. 主 goroutine 等待 `SIGINT` / `SIGTERM`，收到信号后用 10s 超时优雅关闭

前端开发模式（Vite）由 `frontend/src/main.tsx` 启动：

1. 创建 `QueryClient`（默认 `retry: 1`）
2. 用 `ThemeProvider`、`I18nProvider`、`RouterProvider` 包装 `App`
3. `vite.config.ts` 把 `__API_BASE_URL__` 通过 `define` 注入；`/api` 请求被代理到 `http://localhost:8080`

容器模式（`docker-compose.example.yaml`）启动顺序：

1. `postgres` 与 `redis` 健康检查通过
2. `migrate` 服务执行 `migrate -path /migrations -database "$DATABASE_URL" up`，成功后退出
3. `backend` 在 `migrate` 成功后启动，`/api/v1/health/live` 通过后视为健康
4. `frontend` 在 `backend` 健康后启动，Nginx 监听 `$FRONTEND_PORT`，`/api/` 反向代理到 `backend:8080`

## Configuration Loading

- 配置来源：环境变量（本地）或 Compose 注入（容器）
- 必填变量：`DATABASE_URL`、`REDIS_URL`、`CORS_ORIGINS`
- 可选变量与默认值：`API_PORT`（默认 8080）、`LOG_LEVEL`（默认 `info`）
- 协议白名单：`DATABASE_URL` 只接受 `postgres` / `postgresql`，`REDIS_URL` 只接受 `redis` / `rediss`
- 校验失败行为：进程直接退出，stderr 打印明确错误信息
- 前端构建期变量：`VITE_API_BASE_URL` 在容器构建时通过 `ARG` 注入，默认 `/api/v1`

## Runtime Modes

- **local-dev（后端）**：`make backend-dev` 通过 Air 热重载，`backend/.air.toml` 指定 `env_files = ["../.env"]`
- **local-dev（前端）**：`pnpm --dir frontend dev`，Vite 代理 `/api` 到本地后端
- **container**：Compose 启动 postgres、redis、migrate、backend、frontend；Nginx 做反向代理
- **test**：`go test ./...`（后端单元 / 路由）、`go test -tags integration ./internal/repository`（Testcontainers）、`pnpm --dir frontend test`（Vitest + jsdom）

## External Dependencies

- PostgreSQL 17（容器镜像 `postgres:17.6-alpine3.22`）；本地可用独立实例
- Redis 7（容器镜像 `redis:7.4.2-alpine`，开启 `requirepass`）；当前仅作为就绪探针依赖，未持久化业务数据
- Testcontainers PostgreSQL（仅集成测试，需要 Docker）

## Health Probes

- `GET /api/v1/health/live`：进程存活即返回 `200 { "status": "ok" }`
- `GET /api/v1/health/ready`：在 2s 超时内对 PostgreSQL 与 Redis 各执行一次 `Ping`，两者都成功才返回 `200`；任一失败返回 `500 application/problem+json`

## Notes

- 部署流程细节写入 `docs/guides/release-or-deploy.md`
- 启动失败排查写入 `docs/guides/debugging-guide.md`
