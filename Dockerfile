# 使用官方Go镜像作为构建环境
FROM golang:1.26-alpine AS builder

# 安装必要工具
RUN apk add --no-cache git ca-certificates

# 设置工作目录
WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o go-stock-web ./web

# 使用轻量级基础镜像运行应用
FROM alpine:latest

# 安装ca-certificates以支持HTTPS请求
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/go-stock-web .

# 创建必要的目录
RUN mkdir -p logs
RUN mkdir -p data

EXPOSE 8080

CMD ["./go-stock-web"]