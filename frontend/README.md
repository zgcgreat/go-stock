# go-stock Web 部署指南

## 概述

go-stock Web 版支持前后端分离部署，前端为 Vue3 + NaiveUI 构建的单页应用，后端为 Go 服务。

## 环境要求

- Node.js >= 18
- Go >= 1.21
- Docker (可选)

---

## 本地开发启动

### 1. 前端启动

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

访问 http://localhost:5173

### 2. 后端启动

```bash
# 方式一：二进制
./go-stock-web.exe

# 方式二：Go运行
go run .
或者
go run web/main.go
# 方式三：Docker
docker run -d -p 8080:8080 go-stock-backend
```

后端运行在 http://localhost:8080

### 3. 配置 API 代理

开发时前端默认访问 `/api/v1`，已配置代理到后端。

如需修改，编辑 `frontend/vite.config.ts`：

```typescript
proxy: {
  '/api/v1': {
    target: 'http://localhost:8080',
    changeOrigin: true,
  },
}
```

---

## Docker 部署

### 方案一：前后端分离

```bash
# 构建前端镜像
docker build -t go-stock-frontend ./frontend

# 构建后端镜像
docker build -t go-stock-backend .

# 启动后端
docker run -d -p 8080:8080 go-stock-backend

# 启动前端
docker run -d -p 80:80 go-stock-frontend
```

访问：
- 前端：http://localhost
- 后端：http://localhost:8080

### 方案二：使用 docker-compose

```bash
# 一键部署
docker-compose -f docker-compose.full.yml up -d

# 查看日志
docker-compose -f docker-compose.full.yml logs -f

# 停止
docker-compose -f docker-compose.full.yml down
```

### 方案三：现有项目部署

项目根目录已有 Dockerfile 和 docker-compose.yml：

```bash
# 构建并启动
docker-compose up -d

# 访问 http://localhost:8080
```

---

## Nginx 部署（生产推荐）

### 1. 构建前端

```bash
cd frontend
npm install
npm run build
```

### 2. Nginx 配置

```nginx
server {
    listen 80;
    server_name your-domain.com;

    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/v1 {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/v1/ai {
        proxy_pass http://backend:8080;
        proxy_buffering off;
        proxy_cache off;
    }
}
```

### 3. Docker Compose 编排

```yaml
version: '3.8'
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./dist:/usr/share/nginx/html
      - ./nginx.conf:/etc/nginx/conf.d/default.conf
    depends_on:
      - backend

  backend:
    image: go-stock-backend
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data