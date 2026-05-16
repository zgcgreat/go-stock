# Docker 部署指南

## 方案一：本地构建推送（推荐）

适用于 VPS 内存不足的情况，在本地构建镜像后推送到镜像仓库。

### 1. 注册 Docker Hub 账号

- 访问 https://hub.docker.com/signup 注册账号
- 登录后右上角显示的就是你的用户名（例如 `joy123`）

> **注意**：Docker Hub 免费账号的镜像是**公开的**，所有人都能看到和拉取。如需私有镜像，需要付费订阅或使用其他私有仓库。

### 2. 本地构建并推送

假设你的 Docker Hub 用户名是 `joy123`：

```bash
# 登录 Docker Hub
docker login

# 构建镜像（替换 joy123 为你的用户名）
docker build -t joy123/go-stock-web:latest .

# 推送到 Docker Hub
docker push joy123/go-stock-web:latest
```

### 3. VPS 拉取运行

```bash
# 创建目录
mkdir -p /opt/go-stock && cd /opt/go-stock

# 创建 docker-compose.yml（替换镜像名）
cat > docker-compose.yml << 'EOF'
version: '3.8'
services:
  go-stock-web:
    image: joy123/go-stock-web:latest
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

## 方案二：镜像文件传输（无需公开）

如果不想公开镜像，可以导出镜像文件直接传输到 VPS。

### 1. 本地导出镜像

```bash
# 构建镜像
docker build -t go-stock-web:latest .

# 导出为压缩文件
docker save go-stock-web:latest | gzip > go-stock-web.tar.gz
```

### 2. 上传到 VPS

```bash
# 使用 scp 上传
scp go-stock-web.tar.gz user@your-vps-ip:/opt/
```

### 3. VPS 加载并运行

```bash
# 加载镜像
cd /opt
docker load < go-stock-web.tar.gz

# 创建目录
mkdir -p /opt/go-stock && cd /opt/go-stock

# 创建 docker-compose.yml
cat > docker-compose.yml << 'EOF'
version: '3.8'
services:
  go-stock-web:
    image: go-stock-web:latest
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

## 方案三：VPS 直接构建

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

## 方案四：使用 GitHub Actions 自动构建

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

## 方案五：使用私有镜像仓库

### 阿里云容器镜像服务

```bash
# 登录阿里云镜像仓库
docker login --username=your-email registry.cn-hangzhou.aliyuncs.com

# 构建并打标签
docker build -t registry.cn-hangzhou.aliyuncs.com/your-namespace/go-stock-web:latest .

# 推送
docker push registry.cn-hangzhou.aliyuncs.com/your-namespace/go-stock-web:latest
```

### GitHub Container Registry

```bash
# 登录 GitHub
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# 构建并推送
docker build -t ghcr.io/your-username/go-stock-web:latest .
docker push ghcr.io/your-username/go-stock-web:latest
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
