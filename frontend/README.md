# 前端 SPA

本目录是 Go React Starter 的 React 19 + Vite 单页应用。React Router 负责浏览器路由，React Query 通过 Orval 生成的客户端访问 OpenAPI 契约，Zustand 保存语言和主题偏好。

## 常用命令

```bash
pnpm install --frozen-lockfile
pnpm dev
pnpm lint
pnpm exec tsc -b
pnpm test
pnpm build
pnpm run generate
```

开发服务器会把 `/api` 代理到 `http://localhost:8080`。可用 `VITE_API_BASE_URL` 覆盖构建期 API 基址，默认值为 `/api/v1`。

## 组件与接口约束

- `src/components/ui/` 与 `src/components/hooks/` 只放置 shadcn CLI 安装的官方 registry 组件；使用 `pnpm dlx shadcn@latest add <component>` 安装。
- 业务页面只能经 `src/api/generated/` 的 Orval React Query hooks 调用接口；不得手写重复的请求函数或模型。
- `src/api/generated/` 由 `pnpm run generate` 更新，禁止手工编辑。
