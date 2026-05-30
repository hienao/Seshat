# BaseGoApp

基于 Nuxt 3 + Go Gin 的全栈模板工程。

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | Nuxt 3 (SSG) + @nuxt/ui |
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

> 默认 SQLite 数据库路径为 `/data/db/basegoapp.db`。本地开发如需自定义路径，可设置 `SQLITE_PATH=./data/db/basegoapp.db`。

2. **启动前端**
```bash
cd frontend
npm install
npm run dev
```

4. **访问应用**
- 前端: http://localhost:3000
- API 文档: http://localhost:8080/swagger/index.html
- 默认用户: admin / admin

### Docker 部署

#### 环境变量

| 变量名 | 必填 | 说明 | 示例 |
|--------|------|------|------|
| `DB_DRIVER` | ❌ | 数据库类型，默认 `sqlite` | `sqlite` / `postgres` |
| `SQLITE_PATH` | ❌ | SQLite 文件路径，默认 `/data/db/basegoapp.db` | `/data/db/basegoapp.db` |
| `DATA_DIR` | ❌ | 业务持久数据目录，默认 `/data` | `/data` |
| `CACHE_DIR` | ❌ | 日志/缓存目录，默认 `/cache` | `/cache` |
| `LOG_DIR` | ❌ | 应用日志目录，默认 `/cache/logs/app` | `/cache/logs/app` |
| `DATABASE_URL` | `postgres` 模式必填 | PostgreSQL 连接串 | `postgres://user:pass@host:5432/dbname?sslmode=disable` |
| `JWT_SECRET` | ✅ | JWT 签名密钥（生产环境请使用强随机字符串） | `your-secret-key-at-least-32-chars` |
| `GIN_MODE` | ❌ | Gin 运行模式，默认 `debug` | `release` |
| `SERVER_PORT` | ❌ | 后端 API 端口，默认 `8080` | `8080` |

#### 挂载目录

| 容器目录 | 用途 |
|----------|------|
| `/data/db` | SQLite 数据库文件目录 |
| `/cache/logs/nginx` | Nginx access/error 日志 |
| `/cache/logs/app` | 应用日志预留目录 |
| `/cache/tmp` | 临时文件和运行时缓存 |

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

## 项目结构

```
basegoapp/
├── frontend/          # Nuxt 前端
│   ├── app/
│   │   ├── pages/     # 页面
│   │   ├── composables/ # 组合式函数
│   │   └── middleware/  # 路由中间件
│   └── nuxt.config.ts
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

## 更换项目名

只需修改以下文件：
1. `backend/go.mod` - module 名称
2. `frontend/package.json` - name 字段
3. `deploy/Dockerfile` - 镜像标签（如需要）
