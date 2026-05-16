# Docker 部署指南

## 方案一：本地构建推送（推荐）

适用于 VPS 内存不足的情况，在本地构建镜像后推送到 Docker Hub。

### 1. 本地构建镜像

```bash
# 构建镜像
docker build -t your-dockerhub-username/go-stock-web:latest .

# 登录 Docker Hub
docker login

# 推送镜像
docker push your-dockerhub-username/go-stock-web:latest
```

### 2. VPS 拉取运行

```bash
# 创建目录
mkdir -p /opt/go-stock && cd /opt/go-stock

# 创建 docker-compose.yml
cat > docker-compose.yml << 'EOF'
version: '3.8'
services:
  go-stock-web:
    image: your-dockerhub-username/go-stock-web:latest
    container_name: go-stock-web
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./logs:/app/logs
    environment:
      - TZ=Asia/Shanghai
EOF

# 启动服务
docker-compose up -d
```

---

## 方案二：VPS 直接构建

如果 VPS 内存 >= 2G，可以尝试直接构建。

### 1. 增加 Swap（重要）

```bash
# 创建 2G swap 文件
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# 永久生效
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

### 2. 克隆代码并构建

```bash
# 克隆代码
git clone https://github.com/your-repo/go-stock.git
cd go-stock

# 构建并启动
docker-compose up -d --build
```

---

## 方案三：使用 GitHub Actions 自动构建

创建 `.github/workflows/docker.yml`：

```yaml
name: Build and Push Docker Image

on:
  push:
    branches: [main, dev-web]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Docker Hub
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Build and Push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ${{ secrets.DOCKERHUB_USERNAME }}/go-stock-web:latest
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

---

## 常用命令

```bash
# 查看日志
docker-compose logs -f

# 重启服务
docker-compose restart

# 停止服务
docker-compose down

# 更新镜像
docker-compose pull && docker-compose up -d
```

## 访问应用

启动后访问: `http://your-vps-ip:8080`
