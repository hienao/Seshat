---
description: BaseGoApp 项目技术规范和约束规则
---

# BaseGoApp 项目规则

本文档定义了 BaseGoApp 模板工程的技术规范，所有开发操作需遵循以下约束。

## 技术栈约束

### 前端
- **框架**: Nuxt 3 (SSG 静态生成模式)
- **UI**: @nuxt/ui (Nuxt 官方组件库)
- **构建命令**: `npm run generate`
- **输出目录**: `.output/public/`

### 后端
- **语言**: Go 1.21+
- **框架**: Gin
- **ORM**: GORM
- **API 文档**: Swaggo/swag (从注释自动生成)
- **认证**: JWT (Token 存储于 LocalStorage)

### 数据库
- **类型**: PostgreSQL
- **连接**: 通过环境变量 `DATABASE_URL` 配置

---

## 代码结构

```
basegoapp/
├── frontend/          # Nuxt 前端项目
│   ├── pages/         # 页面路由
│   ├── components/    # 可复用组件
│   ├── composables/   # 组合式函数
│   └── middleware/    # 路由中间件
├── backend/           # Go 后端项目
│   ├── internal/      # 内部业务代码
│   │   ├── handler/   # HTTP 处理器
│   │   ├── service/   # 业务逻辑
│   │   ├── repository/# 数据访问
│   │   ├── model/     # 数据模型
│   │   └── middleware/# Gin 中间件
│   ├── pkg/           # 可复用包
│   └── docs/          # Swagger 文档
└── deploy/            # 部署配置
```

---

## API 路由规则

### 前端路由
| 路径 | 页面 | 需认证 |
|------|------|--------|
| `/` | 首页 | ❌ |
| `/login` | 登录 | ❌ |
| `/register` | 注册 | ❌ |
| `/profile` | 个人中心 | ✅ |

### 后端 API
| 方法 | 路径 | 说明 | 需认证 |
|------|------|------|--------|
| POST | `/api/auth/register` | 用户注册 | ❌ |
| POST | `/api/auth/login` | 用户登录 | ❌ |
| POST | `/api/auth/logout` | 用户退出 | ✅ |
| GET | `/api/user/profile` | 获取信息 | ✅ |
| PUT | `/api/user/password` | 修改密码 | ✅ |
| GET | `/swagger/*` | API 文档 | ❌ |

---

## 测试方式

### 后端测试
```bash
cd backend
go test ./...
```

### 前端构建验证
```bash
cd frontend
npm run generate
```

### 完整环境测试
```bash
docker-compose up -d
# 访问 http://localhost 测试功能
# 访问 http://localhost/swagger/ 查看 API 文档
```

---

## 部署规则

### Nginx 路由
- `/` → 静态文件 (Nuxt SSG)
- `/api/*` → Gin API (:8080)
- `/swagger/*` → Swagger 文档

### 环境变量

| 变量名 | 必填 | 说明 | 默认值 |
|--------|------|------|--------|
| `DATABASE_URL` | ✅ | PostgreSQL 连接串 | - |
| `JWT_SECRET` | ✅ | JWT 签名密钥 | - |
| `GIN_MODE` | ❌ | Gin 运行模式 | `debug` |
| `SERVER_PORT` | ❌ | 后端 API 端口 | `8080` |

#### DATABASE_URL 格式

```
postgres://用户名:密码@主机:端口/数据库名?sslmode=disable
```

**示例：**
- 本地：`postgres://postgres:postgres@localhost:5432/basegoapp?sslmode=disable`
- Docker 网络：`postgres://postgres:postgres@db:5432/basegoapp?sslmode=disable`
- 云数据库：`postgres://admin:password@rds.example.com:5432/basegoapp?sslmode=require`

#### Docker 启动示例

```bash
docker run -d \
  -p 80:80 \
  -e DATABASE_URL="postgres://user:pass@host:5432/dbname?sslmode=disable" \
  -e JWT_SECRET="your-secret-key-at-least-32-chars" \
  -e GIN_MODE="release" \
  basegoapp
```

---

## 命名约束

工程名 `basegoapp` 仅在以下位置声明：
1. `backend/go.mod` - module 名称
2. `frontend/package.json` - name 字段
3. `deploy/Dockerfile` - 镜像标签

修改工程名时，只需更新以上三处即可。
