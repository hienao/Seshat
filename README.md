# Seshat

基于 React 19 + Go Gin 的全栈模板工程。

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | React 19 + Vite + Appica UI + Tailwind CSS 4 |
| 后端 | Go + Gin + GORM |
| 数据库 | SQLite（默认）/ PostgreSQL（可选） |
| 认证 | JWT |
| 部署 | Docker + Nginx |

## 快速开始

### 本地开发

1. **启动后端**
```bash
cd backend
go run main.go
```

> 默认 SQLite 数据库路径固定为 `/data/db/basegoapp.db`（由容器内部固定目录派生）。

2. **启动前端**
```bash
cd frontend
npm install
npm run dev
```

4. **访问应用**
- 前端: http://localhost:5173
- API 文档: http://localhost:8080/swagger/index.html
- 首次启动管理员由 `DEFAULT_ADMIN_USERNAME` / `DEFAULT_ADMIN_PASSWORD` 初始化

### Docker 部署

#### 环境变量

| 变量名 | 必填 | 说明 | 示例 |
|--------|------|------|------|
| `DB_DRIVER` | ❌ | 数据库类型，默认 `sqlite` | `sqlite` / `postgres` |
| `HOST_DATA_DIR` | ❌ | 宿主机数据目录（bind mount 源） | `./runtime/data` |
| `HOST_CACHE_DIR` | ❌ | 宿主机缓存目录（bind mount 源） | `./runtime/cache` |
| `SQLITE_PATH` | ❌ | SQLite 数据库文件路径，开发环境可覆盖 | `/data/db/basegoapp.db` |
| `DATABASE_URL` | `postgres` 模式必填 | PostgreSQL 连接串 | `postgres://user:pass@host:5432/dbname?sslmode=disable` |
| `JWT_SECRET` | ✅ | JWT 签名密钥（生产环境请使用强随机字符串） | `your-secret-key-at-least-32-chars` |
| `WEBHOOK_ENCRYPTION_KEY` | ✅（生产） | 加密保存 Webhook Secret 的独立密钥 | `random-key-at-least-32-chars` |
| `API_LOG_ENABLED` | ❌ | 是否启用后端接口日志 | `true` |
| `API_LOG_PATH` | ❌ | 独立接口日志 SQLite 文件 | `/cache/logs/app/api-logs.db` |
| `API_LOG_RETENTION_DAYS` | ❌ | 接口日志保留天数 | `30` |
| `CORS_ALLOWED_ORIGINS` | ❌ | CORS 白名单（逗号分隔） | `http://localhost,http://127.0.0.1,http://localhost:5173,http://127.0.0.1:5173` |
| `AUTH_COOKIE_NAME` | ❌ | 认证 Cookie 名称 | `auth_token` |
| `AUTH_COOKIE_SECURE` | ❌ | 是否仅 HTTPS 发送 Cookie | `false` |
| `DEFAULT_ADMIN_USERNAME` | ⚠️ 首次启动建议配置 | 数据库空时初始化管理员用户名 | `admin` |
| `DEFAULT_ADMIN_PASSWORD` | ⚠️ 首次启动建议配置 | 数据库空时初始化管理员密码（至少 12 位） | `ChangeMe123456` |
| `GIN_MODE` | ❌ | Gin 运行模式，默认 `debug` | `release` |

#### 挂载目录

| 容器目录 | 用途 |
|----------|------|
| `/data/db` | SQLite 数据库文件目录 |
| `/cache/logs/nginx` | Nginx access/error 日志 |
| `/cache/logs/app` | 应用日志预留目录 |
| `/cache/tmp` | 临时文件和运行时缓存 |

其中：
- `SQLite` 路径固定派生为 `/data/db/basegoapp.db`
- 应用日志目录固定派生为 `/cache/logs/app`

#### DATABASE_URL 格式

仅当 `DB_DRIVER=postgres` 时需要配置。

```
postgres://用户名:密码@主机:端口/数据库名?sslmode=disable
```

示例：
- 本地开发：`postgres://postgres:postgres@localhost:5432/basegoapp?sslmode=disable`
- Docker 网络：`postgres://postgres:postgres@db:5432/basegoapp?sslmode=disable`
- 云数据库：`postgres://admin:password@rds.example.com:5432/basegoapp?sslmode=require`

#### 启动命令

**方式一：直接运行镜像**
```bash
# 构建镜像
docker build -f deploy/Dockerfile -t basegoapp .

# 运行（替换为实际的数据库连接信息）
docker run -d \
  -p 80:80 \
  -v basegoapp_data:/data \
  -v basegoapp_cache:/cache \
  -e JWT_SECRET="your-secret-key-at-least-32-chars" \
  -e GIN_MODE="release" \
  basegoapp
```

**方式二：使用 docker-compose（推荐）**
```bash
# 复制环境变量配置
cp .env.example .env

# 编辑 .env 文件，配置数据库连接等信息
# 然后启动
docker-compose -f docker-compose.prod.yml up -d
```

如需使用内置 PostgreSQL 服务：
```bash
DB_DRIVER=postgres \
DATABASE_URL="postgres://postgres:postgres@db:5432/basegoapp?sslmode=disable" \
docker-compose --profile postgres up -d
```

### 一键部署脚本（macOS/Linux）

```bash
# 开发环境部署
./deploy.sh

# 生产环境部署
./deploy.sh --prod

# 强制无缓存重建
./deploy.sh --force
```

## Docker Hub 自动构建

GitHub Actions 在 Pull Request 真正合并后触发 Docker 镜像发布，直接 push 到目标分支不会触发：

Beta 与 Release 使用相互独立的版本文件，格式都必须为 `v主版本.次版本.修订版本`，例如 `v0.0.1`：

- `VERSION_BETA`：仅控制合并到 `beta` 分支时发布的版本。
- `VERSION_RELEASE`：仅控制合并到 `main` 分支时发布的版本。

| 合并目标 | 版本来源 | 文件内容为 `v0.0.1` 时的不可变标签 | 滚动标签 |
|----------|----------|--------------------------------------|----------|
| `beta` | `VERSION_BETA` | `beta-v0.0.1` | `beta` |
| `main` | `VERSION_RELEASE` | `v0.0.1` | `release`、`latest` |

工作流在构建前使用 Docker Hub 查询不可变版本标签；如果该标签已存在，则跳过构建和推送，避免重复发布同一合并提交。

需要在 GitHub 仓库中配置：

| 类型 | 名称 | 说明 |
|------|------|------|
| Secret | `DOCKERHUB_USERNAME` | Docker Hub 用户名 |
| Secret | `DOCKERHUB_TOKEN` | 具有目标仓库读写权限的 Access Token |
| Variable（可选） | `DOCKERHUB_REPOSITORY` | Docker Hub 仓库名，默认 `seshat` |

默认发布地址为 `DOCKERHUB_USERNAME/seshat`。工作流仅构建 `linux/amd64` 镜像，并通过 GitHub Actions Cache 复用对应分支的构建缓存。

## 安全基线说明

- CORS 默认使用白名单模式，`Origin` 不在 `CORS_ALLOWED_ORIGINS` 中会被拒绝。
- 登录后令牌写入 HttpOnly Cookie，前端不再依赖 LocalStorage 保存 token。
- 用户改密或角色变更后，旧 JWT 会立即失效（基于 `token_version` 校验）。
- 生产模式（`GIN_MODE=release`）下，`JWT_SECRET` 必须为非默认值且长度至少 32 位。
- 当数据库为空时，必须通过 `DEFAULT_ADMIN_USERNAME` 和 `DEFAULT_ADMIN_PASSWORD` 初始化管理员；弱口令 `admin/admin` 被禁止。

## 项目结构

```
basegoapp/
├── frontend/          # React + Vite 前端
│   ├── src/
│   │   ├── api/       # API 封装与生成客户端
│   │   ├── components/# Appica UI 业务组件
│   │   ├── pages/     # 路由页面
│   │   ├── router/    # 路由与权限守卫
│   │   └── stores/    # Zustand 状态
│   └── vite.config.ts
├── backend/           # Go 后端
│   ├── internal/      # 内部模块
│   ├── pkg/           # 公共包
│   └── docs/          # Swagger 文档
├── deploy/            # 部署配置
└── docker-compose.yml
```

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/register | 用户注册 |
| POST | /api/auth/login | 用户登录 |
| POST | /api/auth/logout | 用户退出 |
| GET | /api/user/profile | 获取用户信息 |
| PUT | /api/user/password | 修改密码 |
| GET | /api/webhooks/apps | 获取可用 App 类型 |
| GET/POST | /api/webhooks/integrations | 查询/创建 Webhook 接入实例 |
| GET | /api/webhooks/events | 查询收到的 Webhook 消息 |
| GET | /api/webhooks/events/:id | 查看消息详情和原始内容 |
| POST | /hooks/v1/:endpointKey | 外部 App Webhook 接收入口（无需登录） |
| GET | /api/admin/logs | 查询后端接口日志（管理员） |
| GET | /api/admin/logs/:id | 查看接口日志详情（管理员） |
| GET | /api/admin/logs/export | 导出接口日志 CSV/JSONL（管理员） |
| POST | /api/admin/logs/clear | 按条件清空接口日志（管理员） |

## 更换项目名

只需修改以下文件：
1. `backend/go.mod` - module 名称
2. `frontend/package.json` - name 字段
3. `deploy/Dockerfile` - 镜像标签（如需要）
