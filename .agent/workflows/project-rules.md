---
description: Seshat 项目技术规范和约束规则
---

# Seshat 项目规则

Seshat 是一个支持多 App、多消息类型展示的 Webhook 消息管理工具。

## 技术栈

### 前端

- React 19 + TypeScript + Vite
- Appica UI + Tailwind CSS 4
- TanStack Query / Table + Zustand
- React Router 路由与权限守卫

### 后端

- Go 1.25+ + Gin
- GORM
- Swaggo/swag API 文档
- JWT Bearer Token 认证

### 数据库

- 默认 SQLite：`/data/db/seshat.db`
- 可选 PostgreSQL：通过 `DATABASE_URL` 配置
- 接口日志和业务日志：`/cache/logs/app/api-logs.db`，分表存储并共用保留周期

## 主要目录

```text
seshat/
├── frontend/          # React 前端
│   └── src/
│       ├── api/       # API 封装与生成类型
│       ├── components/# 通用组件和布局
│       ├── pages/     # 业务页面
│       ├── router/    # 路由与权限守卫
│       └── stores/    # 客户端状态
├── backend/           # Go 后端
│   ├── internal/      # handler/service/repository/model/middleware
│   ├── pkg/           # 数据库和响应工具
│   └── docs/          # Swagger 生成文件
└── deploy/            # Docker 与 Nginx 配置
```

## 业务路由

| 路径 | 页面 | 权限 |
|------|------|------|
| `/` | 产品首页 / 登录后概览 | 公开 |
| `/events` | Webhook 消息流 | 登录用户 |
| `/events/:id` | 消息详情 | 登录用户 |
| `/integrations` | 接入实例 | 登录用户 |
| `/profile` | 个人中心 | 登录用户 |
| `/admin` | 系统管理 | 管理员 |
| `/admin/logs` | 接口日志 | 管理员 |
| `/admin/application-logs` | 分级业务日志 | 管理员 |

## 业务日志

- 手动业务日志统一使用 `internal/logging` 的 `Debug`、`Info`、`Warn`、`Error` 方法。
- 标准库 `log.Printf` 只用于控制台输出，不进入业务日志页面。
- 结构化字段禁止写入 Secret、Token、Password 等敏感信息；日志组件会按字段名自动脱敏作为兜底。

## 验证方式

```bash
cd backend && go test ./...
cd ../frontend && npm run test && npm run build
```

修改 `.github/workflows/` 后还需执行：

```bash
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/docker-publish.yml
```

## 部署约定

- 单一 Docker 镜像同时包含 Go 服务、前端静态文件和 Nginx。
- Docker Hub 镜像：`hienao6/seshat`。
- 支持 `linux/amd64` 和 `linux/arm64`。
- 数据与缓存必须通过 `/data`、`/cache` 持久化。
