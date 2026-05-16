# ============================================
# 多阶段构建 Dockerfile
# 适用于 1C2G VPS 部署
# ============================================

# 阶段1: 构建前端
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# 先复制依赖文件，利用 Docker 缓存
COPY frontend/package*.json ./

# 安装依赖，限制内存使用
RUN npm install --registry=https://registry.npmmirror.com

# 复制前端源码
COPY frontend/ ./

# 构建前端，限制 Node 内存
ENV NODE_OPTIONS="--max-old-space-size=1024"
RUN npm run build

# ============================================
# 阶段2: 构建后端
FROM golang:1.23-alpine AS backend-builder

# 安装必要工具
RUN apk add --no-cache git ca-certificates gcc musl-dev

WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 复制前端构建产物
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# 构建后端，限制并发减少内存占用
ENV GOMAXPROCS=1
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o go-stock-web ./web/

# ============================================
# 阶段3: 运行镜像
FROM alpine:latest

# 安装必要工具
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=backend-builder /app/go-stock-web .

# 创建必要的目录
RUN mkdir -p logs data

EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./go-stock-web"]
