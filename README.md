# Go React Starter

Go React Starter 是一个中性、可扩展的 Web 技术基座：Gin、GORM/SQLite、Zap、OpenAPI、React、React Query、Tailwind 与 shadcn/ui 共同提供最小可运行链路。

## 首次启动

```bash
cp .env.example .env
docker compose up --build
```

浏览器访问 `http://localhost:8080`。后端启动时会按 GORM 模型自动创建或补充 SQLite 表结构；数据库文件保存在仓库根目录的 `data/app.db`。

## 本地开发

SQLite 会使用本地文件，无需单独启动数据库服务。`.env` 示例将数据库文件设为 `backend/data/app.db`（后端命令的工作目录是 `backend/`）。首次本地开发先安装前端依赖：

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
make backend-storage-test
make verify
```

`make verify` 是本地默认验证命令，不包含 Docker Compose 烟测；`make backend-storage-test` 使用临时 SQLite 文件，不需要 Docker。若 Compose 无法启动，先确认端口未占用、Docker Desktop 已运行，以及本地数据目录的权限正确。

更多说明见 [文档索引](docs/README.md)。
