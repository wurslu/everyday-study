# 使用官方 Go 镜像作为构建环境
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装构建依赖（修复 SQLite 编译问题）
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    sqlite-dev \
    build-base

# 设置 CGO 环境变量
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用（修复编译参数）
RUN go build -a -ldflags '-linkmode external -extldflags "-static"' -o main .

# 使用轻量级的 alpine 镜像作为运行环境
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata

# 创建非root用户
RUN addgroup -g 1001 -S appuser && \
    adduser -S appuser -u 1001 -G appuser

# 设置工作目录
WORKDIR /app

# 创建数据目录
RUN mkdir -p /app/data && chown -R appuser:appuser /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 设置文件权限
RUN chown appuser:appuser /app/main && chmod +x /app/main

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 91

# 设置环境变量
ENV DATABASE_PATH=/app/data/learning.db
ENV ENVIRONMENT=production
ENV PORT=91

# 运行应用
CMD ["./main"]