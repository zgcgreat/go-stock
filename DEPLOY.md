# Docker 部署指南

> ⚠️ **重要前提**：项目 `go.mod` 要求 Go >= 1.26，前端 Vite 构建需要约 2GB+ 内存。
> 低内存 VPS/NAS（< 4G）建议在本地构建前端产物后 scp 传输，VPS 上只构建 Go 后端。

## 方案一：本地构建推送（推荐）

适用于 VPS 内存不足的情况，在本地构建镜像后推送到镜像仓库。

### 1. 注册 Docker Hub 账号

- 访问 https://hub.docker.com/signup 注册账号
- 登录后右上角显示的就是你的用户名（例如 `joy123`）

> **注意**：Docker Hub 免费账号的镜像是**公开的**，所有人都能看到和拉取。如需私有镜像，需要付费订阅或使用其他私有仓库。

### 2. 本地构建并推送

假设你的 Docker Hub 用户名是 `joy123`：

```bash
# 1. 先在本地构建前端产物（Vite 构建需 2GB+ 内存，本地 Windows 轻松完成）
cd frontend && npm install && npm run build && cd ..

# 2. 登录 Docker Hub
docker login

# 3. 构建镜像（替换 joy123 为你的用户名）
docker build -t joy123/go-stock-web:latest .

# 4. 推送到 Docker Hub
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
      - "5173:8080"
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

如果不想公开镜像，可以导出镜像文件直接传输到 VPS。**推荐低内存 VPS 使用此方案**。

### 1. 本地构建前端产物

```bash
# 在本地 Windows/Mac 上构建（Vite 构建需约 2GB 内存，本地轻松完成）
cd frontend
npm install
npm run build
cd ..

# 可选：打包前端产物便于传输
tar -czf frontend-dist.tar.gz -C frontend/dist .
```

### 2. 本地构建并导出镜像

```bash
# 构建镜像（需要本地有 Docker 环境）
docker build -t go-stock-web:latest .

# 导出为压缩文件
docker save go-stock-web:latest | gzip > go-stock-web.tar.gz
```

### 3. 上传到 VPS

```bash
# 使用 scp 上传
scp go-stock-web.tar.gz user@your-vps-ip:/opt/
```


### 4. VPS 加载并运行

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
      - "5173:8080"
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

## 🐳 Docker 部署更新

Docker 部署后，代码更新时无需重新走完整部署流程，按以下步骤更新即可：

### 方式一：本地构建 → scp 传输 → VPS 加载（推荐，低内存 VPS）

```bash
# 1. 拉取最新代码
git pull origin dev-web

# 2. 在本地构建前端产物（Vite 构建需 2GB+ 内存，低内存 VPS 会 OOM）
cd frontend
npm install
npm run build
cd ..

# 3. 本地构建镜像（需要本地 Docker 环境）
docker build -t go-stock-web:latest .

# 4. 导出为压缩文件
docker save go-stock-web:latest | gzip > go-stock-web.tar.gz

# 5. 传输到 VPS
scp go-stock-web.tar.gz user@your-vps-ip:/opt/

# 6. VPS 加载并重启
docker load < /opt/go-stock-web.tar.gz
cd /opt/go-stock
docker-compose down
docker-compose up -d

# 7. 清理
rm /opt/go-stock-web.tar.gz
docker image prune -f
```

> 💡 数据持久化通过 volume 映射（`./data:/app/data`、`./logs:/app/logs`），更新镜像不影响已有数据。

### 方式二：VPS 上仅构建后端（前端产物从本地 scp）

适用于本地没有 Docker 环境，但 VPS 有 Docker 的情况：

```bash
# 1. 在本地构建前端产物并打包
cd frontend && npm install && npm run build && cd ..
tar -czf frontend-dist.tar.gz -C frontend/dist .

# 2. 传输前端产物和源码到 VPS
scp frontend-dist.tar.gz user@your-vps-ip:/tmp/
# 源码通过 git pull 在 VPS 上拉取

# 3. VPS 上解压前端产物
cd ~/go-stock/frontend
rm -rf dist
mkdir dist
tar -xzf /tmp/frontend-dist.tar.gz -C dist/

# 4. VPS 上构建 Docker 镜像（仅 Go 后端，内存占用小）
cd ~/go-stock
docker build -t go-stock-web:latest .

# 5. 重启服务
cd /opt/go-stock
docker-compose down
docker-compose up -d

# 6. 清理
rm /tmp/frontend-dist.tar.gz
```

---

## 方案三：VPS 直接构建

如果 VPS 内存 >= 4G，可以尝试在 VPS 上全量构建。

> ⚠️ **注意**：Vite 前端构建需约 2GB+ 内存，Go 后端构建也需内存。1C2G 的 VPS 即使加了 swap 也可能不稳定，建议使用方案一或方案二。

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

# 构建前端（限制 Node 内存）
cd frontend
npm install
NODE_OPTIONS="--max-old-space-size=1536" npm run build
cd ..

# 构建并启动
docker build -t go-stock-web:latest .
docker-compose up -d
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

# 更新镜像（方案一/五）
docker-compose pull && docker-compose up -d
```

## 访问应用

启动后访问: `http://your-vps-ip:5173`

> 💡 容器内部应用监听 8080 端口，通过 Docker 端口映射到宿主机 5173 端口。如需更改宿主机端口，修改 `docker-compose.yml` 中 `ports` 的左侧数字即可。