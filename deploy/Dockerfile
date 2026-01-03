# 阶段 1: 构建前端
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run generate

# 阶段 2: 构建后端
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app/backend
# 安装依赖
RUN apk add --no-cache git
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# 阶段 3: 最终镜像
FROM alpine:latest
RUN apk --no-cache add ca-certificates nginx

WORKDIR /app

# 复制后端二进制文件
COPY --from=backend-builder /app/backend/main .

# 复制前端静态文件
COPY --from=frontend-builder /app/frontend/.output/public /usr/share/nginx/html

# 复制 Nginx 配置
COPY deploy/nginx.conf /etc/nginx/nginx.conf

# 创建启动脚本
RUN echo '#!/bin/sh' > /app/start.sh && \
    echo 'nginx' >> /app/start.sh && \
    echo './main' >> /app/start.sh && \
    chmod +x /app/start.sh

EXPOSE 80

CMD ["/app/start.sh"]
