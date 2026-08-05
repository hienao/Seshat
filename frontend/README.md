# Seshat Frontend

React 19 + Vite + Appica UI 前端。服务端状态由 TanStack Query 管理，当前用户由 Zustand 管理，路由使用 React Router。

## 开发

```bash
npm install
npm run dev
```

开发服务器默认监听 `http://localhost:5173`，并将 `/api` 与 `/swagger` 代理到 `http://localhost:8080`。

## 常用命令

```bash
npm run build         # TypeScript 检查并构建 dist
npm run test          # 运行 Vitest
npm run api:generate  # 根据 backend/docs/swagger.json 更新 API 客户端
npm run preview       # 预览生产构建
```

## 目录

```text
src/
├── api/          # 请求封装、业务 API 与 Swagger 生成客户端
├── app/          # 全局 Provider 与认证启动逻辑
├── components/   # 通用组件、布局和数据表格
├── features/     # 按业务组织的 hooks
├── pages/        # 路由页面
├── router/       # 路由与权限守卫
└── stores/       # Zustand 客户端状态
```

认证令牌由后端写入 HttpOnly Cookie，前端不把令牌或用户会话持久化到 LocalStorage。
