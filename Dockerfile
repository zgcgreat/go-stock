# ============================================
# 多阶段构建 Dockerfile
# 适用于 1C2G VPS 部署
# 前端需在本地预先构建：cd frontend && npm install && npm run build
# ============================================

# 阶段1: 构建后端
FROM golang:1.26-alpine AS backend-builder

# 安装必要工具
RUN apk add --no-cache git ca-certificates gcc musl-dev

WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 设置 Go 代理（国内网络优化）
ENV GOPROXY=https://goproxy.cn,direct

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 复制本地已构建的前端产物
COPY frontend/dist ./frontend/dist

# 构建后端，限制并发减少内存占用
ENV GOMAXPROCS=1
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o go-stock-web ./web/

# ============================================
# 阶段2: 运行镜像
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
