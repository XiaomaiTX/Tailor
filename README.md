# Tailor

**Peer Relay management for Headscale-managed Tailscale networks**

Tailor 为 Headscale 管理的 Tailscale 网络提供 Peer Relay 注册、申请审批和设备授权管理。它同步设备与 Headscale Policy，展示设备到 Relay 的授权关系，并在更新 Tailor 管理的 Relay Grants 时保留 Policy 中的其他内容。

## 启动前

部署需要准备：

- 单个 Headscale 实例的 REST API 地址和 API Key
- 一个支持 OIDC 的身份提供方，并为 Tailor 注册独立的 Client ID、Client Secret 和回调地址
- SQLite 可写目录
- 如需提交 Policy，Headscale 必须使用 database Policy 模式；写入功能默认关闭

不要把 Headscale API Key 或 OIDC Client Secret 放进浏览器配置。Tailor 使用已验证的 OIDC 邮箱唯一匹配 Headscale 用户邮箱；Headscale Policy Groups 仍控制平台角色。

## Compose 部署示例

```bash
cp .env.example .env
# 编辑 .env：填写 Headscale 凭据、OIDC Issuer/Client 参数和部署配置
docker compose --env-file .env -f docker-compose.example.yaml up --build
```

Compose 提供 backend 与前端 Nginx。请在 IdP 中将 Tailor 配置为独立 OIDC Client，登记 `OIDC_REDIRECT_URL`，并确保浏览器通过 HTTPS 访问 Tailor。

## 本地开发

安装 [Go 1.27.3](backend/go.mod)、可用 C 编译器、Node.js 24.x 和 pnpm，然后按 [本地开发指南](docs/guides/local-development.md) 启动。需要 Headscale 访问和有效 OIDC 配置后，业务控制台才能正常使用。

## 常用命令

```bash
make openapi-lint
make generate
make backend-build
make frontend-build
make verify
```

## 文档

- [Peer Relay 管理功能说明](docs/features/current/peer-relay-manager.md)
- [文档索引](docs/README.md)
- [接口路由索引](docs/api/route-index.md)
- [架构与权限边界](docs/architecture/overview.md)
- [发布与部署](docs/guides/release-or-deploy.md)
