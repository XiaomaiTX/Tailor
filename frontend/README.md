# 前端 SPA

本目录是 Tailor 的 React 19 + Vite 单页应用。React Router 负责浏览器路由，React Query 通过 Orval 生成的客户端访问 OpenAPI 契约，Zustand 保存语言和主题偏好。

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

## 页面路由

React Router 在 `src/app/router.tsx` 编排控制台页面：`/` 概览、`/editor` 授权编辑、`/relays` Relay 目录、`/requests` 申请、`/policy` Policy 历史、`/settings` 设置，以及 `/login` 登录。直接访问和刷新页面会保留当前路由；`/policy` 与 `/settings` 限管理员访问，`/requests` 限申请人或审核人访问。生产 Nginx 使用 `index.html` 回退支持深层链接。

## 组件与接口约束

- `src/components/ui/` 与 `src/components/hooks/` 只放置 shadcn CLI 安装的官方 registry 组件；使用 `pnpm dlx shadcn@latest add <component>` 安装。
- 业务页面只能经 `src/api/generated/` 的 Orval React Query hooks 调用接口；不得手写重复的请求函数或模型。
- `src/api/generated/` 由 `pnpm run generate` 更新，禁止手工编辑。
