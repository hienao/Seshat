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

> 默认 SQLite 数据库路径固定为 `/data/db/seshat.db`（由容器内部固定目录派生）。

2. **启动前端**
```bash
cd frontend
npm install
npm run dev
```

4. **访问应用**
- 前端: http://localhost:5173
- API 文档: http://localhost:8080/swagger/index.html
- 空数据库首次启动会创建一次性 `admin/admin` 引导账户，登录后必须设置正式管理员凭据

### Docker 部署

#### 环境变量

| 变量名 | 必填 | 说明 | 示例 |
|--------|------|------|------|
| `DB_DRIVER` | ❌ | 数据库类型，默认 `sqlite` | `sqlite` / `postgres` |
| `HOST_DATA_DIR` | ❌ | 宿主机数据目录（bind mount 源） | `./runtime/data` |
| `HOST_CACHE_DIR` | ❌ | 宿主机缓存目录（bind mount 源） | `./runtime/cache` |
| `SQLITE_PATH` | ❌ | SQLite 数据库文件路径，开发环境可覆盖 | `/data/db/seshat.db` |
| `DATABASE_URL` | `postgres` 模式必填 | PostgreSQL 连接串 | `postgres://user:pass@host:5432/dbname?sslmode=disable` |
| `JWT_SECRET` | ✅ | JWT 签名密钥（生产环境请使用强随机字符串） | `your-secret-key-at-least-32-chars` |

#### 挂载目录

| 容器目录 | 用途 |
|----------|------|
| `/data/db` | SQLite 数据库文件目录 |
| `/cache/logs/nginx` | Nginx access/error 日志 |
| `/cache/logs/app` | 应用日志预留目录 |
| `/cache/tmp` | 临时文件和运行时缓存 |

其中：
- `SQLite` 路径固定派生为 `/data/db/seshat.db`
- 应用日志目录固定派生为 `/cache/logs/app`
- 接口日志固定启用并保存到 `/cache/logs/app/api-logs.db`
- 接口日志默认保留 30 天，管理员可在“系统管理 → 系统设置”中调整为 1–3650 天，保存后立即生效
- 登录令牌由前端保存到 LocalStorage，并通过 `Authorization: Bearer <token>` 请求头发送
- Webhook Secret 以明文保存在业务数据库中；接入列表和详情接口不返回该字段，仅在创建或轮换时返回一次

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

#### 从 Docker Hub 镜像部署

如果不需要在部署机器上构建镜像，可以使用仓库中的 `docker-compose.dockerhub.yml`，它只会拉取 Docker Hub 镜像，不会执行本地构建：

```bash
# 下载项目配置（或在已有项目目录中执行）
git clone https://github.com/hienao/Seshat.git
cd Seshat

# 创建并编辑生产环境配置
cp .env.example .env
```

至少修改 `.env` 中的以下配置：

```dotenv
JWT_SECRET=请替换为至少32位的随机字符串
```

默认使用 Beta 滚动标签 `hienao/seshat:beta`。启动前可通过 `SESHAT_IMAGE` 指定其他 Docker Hub 镜像或版本：

```bash
# Beta 最新镜像
SESHAT_IMAGE=hienao/seshat:beta \
docker compose -f docker-compose.dockerhub.yml pull
SESHAT_IMAGE=hienao/seshat:beta \
docker compose -f docker-compose.dockerhub.yml up -d

# Release 最新镜像
SESHAT_IMAGE=hienao/seshat:latest \
docker compose -f docker-compose.dockerhub.yml up -d

# 使用不可变版本镜像（推荐用于可追溯部署）
SESHAT_IMAGE=hienao/seshat:beta-v0.0.4 \
docker compose -f docker-compose.dockerhub.yml up -d
```

如果 Docker Hub 用户名或仓库名不同，请将镜像改为 `<用户名>/<仓库名>:<标签>`。首次部署建议先拉取并检查配置：

```bash
docker compose -f docker-compose.dockerhub.yml config
docker compose -f docker-compose.dockerhub.yml up -d
docker compose -f docker-compose.dockerhub.yml ps
```

默认端口为 `80`，SQLite 数据保存在 `./runtime/data`，应用和 Nginx 日志保存在 `./runtime/cache`。如需使用内置 PostgreSQL：

```bash
SESHAT_IMAGE=hienao/seshat:beta \
DB_DRIVER=postgres \
DATABASE_URL="postgres://postgres:postgres@db:5432/basegoapp?sslmode=disable" \
docker compose --profile postgres -f docker-compose.dockerhub.yml up -d
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

- CORS 默认允许任意 Origin，返回 `Access-Control-Allow-Origin: *`，且不允许跨域 Cookie 凭据。
- 登录后令牌保存到浏览器 LocalStorage，前端通过 `Authorization: Bearer <token>` 显式发送。
- LocalStorage 可被页面 JavaScript 读取，因此部署时仍需防范 XSS，并建议配置严格的 Content Security Policy。
- 用户退出、改密或角色变更后，旧 JWT 会立即失效（基于 `token_version` 校验）；退出会同时使该用户当前版本的其他令牌失效。
- 服务固定使用 Gin `release` 模式，`JWT_SECRET` 必须为非默认值且长度至少 32 位。
- 空数据库首次启动会创建受限的一次性 `admin/admin` 账户；该账户只能访问管理员初始化、个人信息和退出接口，设置正式凭据后立即失效。

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
