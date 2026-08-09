# 阶段 1: 前端产物与目标 CPU 架构无关，固定在构建机原生架构执行，避免 QEMU 下重复运行 npm。
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci \
    --registry=https://registry.npmjs.org \
    --replace-registry-host=always \
    --fetch-retries=5 \
    --fetch-retry-maxtimeout=120000
COPY frontend/ ./
RUN npm run build

# 阶段 2: 构建后端
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app/backend
# 安装依赖
RUN sed -i 's|https://dl-cdn.alpinelinux.org/alpine|https://mirrors.ustc.edu.cn/alpine|g' /etc/apk/repositories && \
    apk update && \
    apk add --no-cache git gcc musl-dev
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# 构建
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# 阶段 3: 最终镜像
FROM alpine:latest
ENV TZ=UTC
RUN sed -i 's|https://dl-cdn.alpinelinux.org/alpine|https://mirrors.ustc.edu.cn/alpine|g' /etc/apk/repositories && \
    apk update && \
    apk --no-cache add ca-certificates nginx tzdata

WORKDIR /app

# 复制后端二进制文件
COPY --from=backend-builder /app/backend/main .

# 复制前端静态文件
COPY --from=frontend-builder /app/frontend/dist /usr/share/nginx/html

# 复制 Nginx 配置
COPY deploy/nginx.conf /etc/nginx/nginx.conf

# 创建启动脚本
RUN mkdir -p /data/db /cache/logs/nginx /cache/logs/app /cache/tmp && \
    echo '#!/bin/sh' > /app/start.sh && \
    echo 'set -e' >> /app/start.sh && \
    echo 'mkdir -p /data/db /cache/logs/nginx /cache/logs/app /cache/tmp' >> /app/start.sh && \
    echo 'nginx' >> /app/start.sh && \
    echo 'exec ./main' >> /app/start.sh && \
    chmod +x /app/start.sh

EXPOSE 3112

CMD ["/app/start.sh"]
