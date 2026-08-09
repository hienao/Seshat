# Seshat

用于统一接收、展示和排查多种 App Webhook 消息的轻量管理工具。

当前内置通用 Webhook、GitHub、Jellyfin 和 Emby App 类型。Jellyfin 与 Emby 的媒体库、播放、认证、用户、系统和插件事件会转换为共享的结构化卡片；未知类型统一匹配该 App 的默认消息类型并展示原始内容。每个接入实例都可以按消息类型选择是否通过 Webhook、Telegram、Apprise、邮箱、Server酱、Bark、DingTalk、Feishu、WhatsApp 或 WxPusher 推送，所有推送开关初始均为关闭。

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
| `SESHAT_IMAGE` | ❌ | Docker Hub 部署使用的镜像，默认 Beta 滚动标签 | `hienao6/seshat:beta` |
| `SQLITE_PATH` | ❌ | SQLite 数据库文件路径，开发环境可覆盖 | `/data/db/seshat.db` |
| `DATABASE_URL` | `postgres` 模式必填 | PostgreSQL 连接串 | `postgres://user:pass@host:5432/dbname?sslmode=disable` |
| `JWT_SECRET` | ✅ | JWT 签名密钥（生产环境请使用强随机字符串） | `your-secret-key-at-least-32-chars` |
| `TZ` | ❌ | 容器日志和应用本地时间，默认 `UTC` | `UTC` |
| `SESHAT_UPDATE_FEED_BASE_URL` | ❌ | Beta/Release 固定静态更新源，可替换为 Pages 自定义域名 | `https://seshatapp.pages.dev/updates/v1` |

#### 挂载目录

| 容器目录 | 用途 |
|----------|------|
| `/data/db` | SQLite 数据库文件目录 |
| `/cache/logs/nginx` | Nginx access/error 日志 |
| `/cache/logs/app` | 接口日志和业务日志数据库目录 |
| `/cache/tmp` | 临时文件和运行时缓存 |

其中：
- `SQLite` 路径固定派生为 `/data/db/seshat.db`
- 日志目录固定派生为 `/cache/logs/app`
- 接口日志和业务日志固定启用并保存到 `/cache/logs/app/api-logs.db` 的独立数据表中
- 接口日志、业务日志和外部媒体资料缓存默认保留 7 天，管理员可在“系统管理 → 系统设置”中调整为 1–30 天，保存后立即生效
- Jellyfin/Emby 会展示剧集层级、客户端、设备、简介和视频流信息；可选配置 TMDB API 密钥，按 Provider ID 补充并缓存海报和简介
- 登录令牌由前端保存到 LocalStorage，并通过 `Authorization: Bearer <token>` 请求头发送
- Webhook Secret 以明文保存在业务数据库中；接入列表不批量返回该字段，实例所属用户可在接入卡片中按需查看，查看接口禁止缓存
- Emby 不支持自定义 Webhook 请求头，因此使用随机接入地址作为凭据；其他 App 按接入说明使用 Secret 请求头或签名
- 推送渠道凭据以明文保存在业务数据库中，但 API 不返回其内容；Webhook、Apprise 和其他可配置 URL 支持 HTTP/HTTPS 与私有网络目标，同时阻止链路本地、组播、未指定地址和已知云元数据地址
- 管理员可在系统设置中保存一个 HTTP/HTTPS 代理；每个推送渠道独立决定是否使用，默认不使用代理

#### DATABASE_URL 格式

仅当 `DB_DRIVER=postgres` 时需要配置。

```
postgres://用户名:密码@主机:端口/数据库名?sslmode=disable
```

示例：
- 本地开发：`postgres://postgres:postgres@localhost:5432/seshat?sslmode=disable`
- Docker 网络：`postgres://postgres:postgres@db:5432/seshat?sslmode=disable`
- 云数据库：`postgres://admin:password@rds.example.com:5432/seshat?sslmode=require`

#### 启动命令

**方式一：直接运行镜像**
```bash
# 构建镜像
docker build -f deploy/Dockerfile -t seshat .

# 运行（替换为实际的数据库连接信息）
docker run -d \
  -p 3112:3112 \
  -v seshat_data:/data \
  -v seshat_cache:/cache \
  -e JWT_SECRET="your-secret-key-at-least-32-chars" \
  -e TZ="UTC" \
  seshat
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
DATABASE_URL="postgres://postgres:postgres@db:5432/seshat?sslmode=disable" \
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

默认使用 Beta 滚动标签 `hienao6/seshat:beta`。启动前可通过 `SESHAT_IMAGE` 指定其他 Docker Hub 镜像或版本：

```bash
# Beta 最新镜像
SESHAT_IMAGE=hienao6/seshat:beta \
docker compose -f docker-compose.dockerhub.yml pull
SESHAT_IMAGE=hienao6/seshat:beta \
docker compose -f docker-compose.dockerhub.yml up -d

# 使用不可变版本镜像（推荐用于可追溯部署）
SESHAT_IMAGE=hienao6/seshat:beta-v0.0.4 \
docker compose -f docker-compose.dockerhub.yml up -d
```

当前尚未发布正式 Release，因此 `latest`、`release` 和 `v0.0.1` 标签暂不可用；首次正式版发布成功后再使用对应标签。

新发布的镜像同时支持 `linux/amd64` 和 `linux/arm64`。历史标签 `beta-v0.0.4` 仅包含 `linux/amd64`，ARM64 设备需要使用更新版本；如果设备已配置 x86 模拟，也可临时使用 `DOCKER_DEFAULT_PLATFORM=linux/amd64` 运行旧镜像。

如果 Docker Hub 用户名或仓库名不同，请将镜像改为 `<用户名>/<仓库名>:<标签>`。首次部署建议先拉取并检查配置：

```bash
docker compose -f docker-compose.dockerhub.yml config
docker compose -f docker-compose.dockerhub.yml up -d
docker compose -f docker-compose.dockerhub.yml ps
```

容器内外统一使用 Web 端口 `3112`，默认访问地址为 `http://localhost:3112`。SQLite 数据保存在 `./runtime/data`，应用和 Nginx 日志保存在 `./runtime/cache`。如需使用内置 PostgreSQL：

```bash
SESHAT_IMAGE=hienao6/seshat:beta \
DB_DRIVER=postgres \
DATABASE_URL="postgres://postgres:postgres@db:5432/seshat?sslmode=disable" \
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
- `release-notes/beta/vX.Y.Z.json`：Beta 版本对应的中英文更新记录。
- `release-notes/release/vX.Y.Z.json`：Release/Hotfix 对应的中英文更新记录。

| 合并目标 | 版本来源 | 文件内容为 `v0.0.1` 时的不可变标签 | 滚动标签 |
|----------|----------|--------------------------------------|----------|
| `beta` | `VERSION_BETA` | `beta-v0.0.1` | `beta` |
| `main` | `VERSION_RELEASE` | `v0.0.1` | `release`、`latest` |

工作流在构建前校验目标版本的双语更新记录，并使用 Docker Hub 查询不可变版本标签；如果该标签已存在，则跳过构建和推送，避免重复发布同一合并提交。新镜像构建并通过多架构验证后，工作流会创建同名 GitHub Release；启用 Cloudflare Pages 发布后，还会重建双语项目主页、更新记录页，以及 `/updates/v1/beta.json` 和 `/updates/v1/release.json` 两个固定静态更新源。Beta 与 Release 的更新目录和检查结果完全隔离。

运行镜像会把版本、渠道、提交和构建时间写入后端二进制。管理员登录后可以在左侧 Seshat 名称下查看当前版本及更新状态；更新检查只请求当前渠道对应的 Pages 静态 JSON，不再调用 GitHub Releases API。配置系统 HTTP 代理后，更新请求仍会自动通过该代理发送。需要使用自定义 Pages 域名时，可通过 `SESHAT_UPDATE_FEED_BASE_URL` 覆盖默认地址。

需要在 GitHub 仓库中配置：

| 类型 | 名称 | 说明 |
|------|------|------|
| Secret | `DOCKERHUB_USERNAME` | Docker Hub 用户名，本仓库应配置为 `hienao6` |
| Secret | `DOCKERHUB_TOKEN` | 具有目标仓库读写权限的 Access Token |
| Variable（可选） | `DOCKERHUB_REPOSITORY` | Docker Hub 仓库名，默认 `seshat` |
| Secret | `CLOUDFLARE_API_TOKEN` | 具有 Cloudflare Pages Edit 权限的 API Token |
| Secret | `CLOUDFLARE_ACCOUNT_ID` | Cloudflare Account ID |
| Variable | `CLOUDFLARE_PAGES_ENABLED` | 设置为 `true` 后启用 Pages 发布；未设置时跳过且不影响镜像发版 |
| Variable（可选） | `CLOUDFLARE_PAGES_PROJECT` | Pages 项目名，默认 `seshatapp`；项目需预先创建 |

当前发布地址为 `hienao6/seshat`（由 `DOCKERHUB_USERNAME/seshat` 组合生成）。工作流使用原生 AMD64 与 ARM64 GitHub 托管 Runner 并行构建 `linux/amd64` 和 `linux/arm64` 镜像，各平台按 digest 推送后再统一生成多架构标签；两个平台分别通过 GitHub Actions Cache 复用对应分支的构建缓存。

## 安全基线说明

- CORS 默认允许任意 Origin，返回 `Access-Control-Allow-Origin: *`，且不允许跨域 Cookie 凭据。
- 登录后令牌保存到浏览器 LocalStorage，前端通过 `Authorization: Bearer <token>` 显式发送。
- LocalStorage 可被页面 JavaScript 读取，因此部署时仍需防范 XSS，并建议配置严格的 Content Security Policy。
- 用户退出、改密或角色变更后，旧 JWT 会立即失效（基于 `token_version` 校验）；退出会同时使该用户当前版本的其他令牌失效。
- 服务固定使用 Gin `release` 模式，`JWT_SECRET` 必须为非默认值且长度至少 32 位。
- 空数据库首次启动会创建受限的一次性 `admin/admin` 账户；该账户只能访问管理员初始化、个人信息和退出接口，设置正式凭据后立即失效。

## 项目结构

```
Seshat/
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
| GET/PUT | /api/webhooks/integrations/:id/notification-settings | 查询/更新实例的渠道绑定和消息类型开关 |
| GET | /api/webhooks/events | 查询收到的 Webhook 消息 |
| GET | /api/webhooks/events/:id | 查看消息详情和原始内容 |
| GET | /api/webhooks/events/:id/notification-status | 查看消息的推送状态 |
| GET | /api/public/events/:token | 通过随机访问标识查看标准化消息（无需登录） |
| GET/POST | /api/notification-channels | 查询/创建推送渠道 |
| PUT/DELETE | /api/notification-channels/:id | 更新/删除推送渠道 |
| POST | /api/notification-channels/:id/test | 发送测试通知 |
| GET | /api/notifications/deliveries | 查询最近推送记录 |
| POST | /api/notifications/deliveries/:id/retry | 手动重试最终失败的推送 |
| POST | /hooks/v1/:endpointKey | 外部 App Webhook 接收入口（无需登录） |
| GET | /api/admin/logs | 查询后端接口日志（管理员） |
| GET | /api/admin/logs/:id | 查看接口日志详情（管理员） |
| GET | /api/admin/logs/export | 导出接口日志 CSV/JSONL（管理员） |
| POST | /api/admin/logs/clear | 按条件清空接口日志（管理员） |
| GET | /api/admin/application-logs | 按等级、来源和关键字查询业务日志（管理员） |
| GET | /api/admin/application-logs/:id | 查看业务日志详情（管理员） |
| GET | /api/admin/application-logs/export | 导出业务日志 CSV/JSONL（管理员） |
| POST | /api/admin/application-logs/clear | 按条件清空业务日志（管理员） |

启用消息推送前，请在“系统管理 → 系统设置”中配置用户可访问的“对外访问地址”，例如 `https://seshat.example.com`。每条实际推送的消息末尾会附加不可猜测的公开详情链接；该页面只返回标准化展示，不包含原始 Webhook 内容、接入实例、推送状态或日志信息。该链接本身具有访问权限，请勿公开转发。

## 记录业务日志

后端代码通过 `internal/logging` 提供的等级方法记录业务日志。日志会同时输出到容器控制台和“系统管理 → 业务日志”页面：

```go
logging.Debug("webhook", "开始解析消息", logging.Fields{"request_id": requestID})
logging.Info("webhook", "消息处理完成", logging.Fields{"event_id": eventID})
logging.Warn("webhook", "消息签名无效", logging.Fields{"app_code": appCode})
logging.Error("database", "保存消息失败", logging.Fields{"error": err})
```

支持 `DEBUG`、`INFO`、`WARN`、`ERROR` 四个等级。结构化字段名包含 `secret`、`token`、`password`、`cookie`、`authorization`、`signature` 或 `api-key` 时，值会在控制台和日志数据库中自动替换为 `[REDACTED]`。标准库 `log.Printf` 仍只输出到容器控制台，不会写入业务日志页面。
