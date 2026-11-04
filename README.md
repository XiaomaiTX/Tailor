# Go React Starter

Go React Starter 是一个中性、可扩展的 Web 技术基座：Gin、GORM/PostgreSQL、Redis、Zap、OpenAPI、React、React Query、Tailwind 与 shadcn/ui 共同提供最小可运行链路。

## 首次启动

```bash
cp .env.example .env
docker compose up --build
```

浏览器访问 `http://localhost:8080`。首次启动时 `migrate` 服务会对空数据库执行首条 migration，成功后才启动后端；应用本身不会建表或变更 schema。

## 本地开发

准备一个独立 PostgreSQL、Redis，并将 `.env` 中的 `DATABASE_URL`、`REDIS_URL` 改为本机可连接的地址。首次本地开发先安装前端依赖：

```powershell
cd frontend
pnpm install --frozen-lockfile
```

后端使用 [Air](https://github.com/air-verse/air) 实现热重载，首次使用需安装：

```bash
go install github.com/air-verse/air@v1.65.1
```

安装后确认 `$(go env GOPATH)/bin` 已加入 `PATH`，再用 `air -v` 验证。然后在两个终端分别启动服务：

```powershell
# 终端一：后端
make backend-dev
```

```powershell
# 终端二：前端
cd frontend
pnpm dev
```

修改 Go 源码后 Air 会自动重新编译并重启后端；`.env` 变更后重启 Air 即可重新读取。前端由 Vite 提供热更新，与后端互不影响。

## 常用命令

```bash
make openapi-lint
make generate
make generate-check
make migrate-up
make migrate-down-one
make migrate-version
make migrate-force VERSION=1
make backend-integration
make verify
```

`DATABASE_URL` 是迁移命令的必填环境变量。`make verify` 是本地默认验证命令，不包含 Docker Compose 烟测；`make backend-integration` 仅在具备 Docker 时可选执行。若 Compose 无法启动，先确认端口未占用、Docker Desktop 已运行，以及本项目本地数据目录的权限正确。

更多说明见 [文档索引](docs/README.md)。
