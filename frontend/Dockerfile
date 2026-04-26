FROM nginx:alpine

# 复制本地构建产物（需先在本地运行 npm run build）
COPY dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]