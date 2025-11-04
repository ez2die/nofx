# 域名部署指南 - 无需端口号访问

本指南说明如何配置域名，使前端可以通过域名直接访问而无需输入端口号。

## 📋 前置要求

1. 已绑定域名并完成DNS解析（A记录指向服务器IP）
2. 服务器已安装Docker和Docker Compose
3. 服务器80端口（HTTP）或443端口（HTTPS）未被占用

## 🚀 快速配置步骤

### 步骤1: 修改域名配置

编辑 `nginx/nginx.conf`，将 `server_name` 替换为您的实际域名：

```nginx
server {
    listen 80;
    server_name yourdomain.com www.yourdomain.com;  # 替换为您的域名
    # ... 其他配置
}
```

### 步骤2: 修改端口映射（已完成）

`docker-compose.yml` 已经配置为使用80端口：

```yaml
ports:
  - "${NOFX_FRONTEND_PORT:-80}:80"
```

### 步骤3: 重启服务

```bash
# 停止现有服务
docker-compose down

# 重新构建并启动
docker-compose up -d --build
```

### 步骤4: 验证访问

现在可以通过以下方式访问：
- `http://yourdomain.com` （无需端口号）

---

## 🔒 HTTPS 配置（可选但推荐）

### 使用 Let's Encrypt 免费证书

#### 方式1: 在主机上安装证书（推荐）

1. **安装 certbot**

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install certbot

# CentOS/RHEL
sudo yum install certbot
```

2. **获取证书**

```bash
sudo certbot certonly --standalone -d yourdomain.com -d www.yourdomain.com
```

证书将保存在：`/etc/letsencrypt/live/yourdomain.com/`

3. **修改 docker-compose.yml**

将证书目录挂载到容器中：

```yaml
nofx-frontend:
  # ... 其他配置
  volumes:
    - ./nginx/nginx.conf:/etc/nginx/conf.d/default.conf:ro
    - /etc/letsencrypt:/etc/letsencrypt:ro  # 挂载证书目录
```

4. **使用 HTTPS 配置**

```bash
# 复制HTTPS配置示例
cp nginx/nginx.conf.https.example nginx/nginx.conf
```

5. **编辑 nginx.conf**，更新域名和证书路径：

```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com www.yourdomain.com;
    
    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    # ... 其他配置
}
```

6. **修改 docker-compose.yml 端口映射**

```yaml
ports:
  - "80:80"    # HTTP redirect
  - "443:443"  # HTTPS
```

7. **重启服务**

```bash
docker-compose down
docker-compose up -d --build
```

#### 方式2: 使用 Docker 容器自动获取证书

使用 `certbot/certbot` 镜像，在容器内自动获取和更新证书。

---

## 🔧 环境变量配置

可以通过环境变量覆盖默认端口：

```bash
# 使用自定义端口（如果需要）
export NOFX_FRONTEND_PORT=8080
docker-compose up -d
```

---

## 📝 常见问题

### Q1: 80端口被占用怎么办？

**方案A**: 查找并停止占用80端口的服务

```bash
# 查看占用80端口的进程
sudo lsof -i :80
sudo netstat -tulpn | grep :80

# 停止占用服务（根据实际情况操作）
sudo systemctl stop nginx  # 或其他服务
```

**方案B**: 使用其他端口（如8080），然后通过外部Nginx反向代理

```yaml
ports:
  - "8080:80"  # 容器内仍用80，映射到主机8080
```

然后在主机上安装Nginx，配置反向代理：

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    
    location / {
        proxy_pass http://localhost:8080;
        # ... 其他代理配置
    }
}
```

### Q2: 如何配置多个域名？

在 `nginx.conf` 中列出所有域名：

```nginx
server_name domain1.com www.domain1.com domain2.com www.domain2.com;
```

### Q3: 证书如何自动续期？

Let's Encrypt 证书有效期90天，需要定期续期。

**手动续期**：
```bash
sudo certbot renew
```

**自动续期**（推荐）：
```bash
# 添加到 crontab
sudo crontab -e
# 添加以下行（每天凌晨2点检查续期）
0 2 * * * certbot renew --quiet && docker-compose restart nofx-frontend
```

### Q4: 访问时出现 502 Bad Gateway？

检查后端服务是否正常运行：

```bash
# 查看容器状态
docker-compose ps

# 查看后端日志
docker-compose logs nofx

# 检查后端健康状态
curl http://localhost:8080/api/health
```

---

## ✅ 验证清单

配置完成后，请验证：

- [ ] 域名DNS解析正确（`ping yourdomain.com` 返回服务器IP）
- [ ] 80端口未被占用
- [ ] Nginx配置中的 `server_name` 已更新为实际域名
- [ ] Docker容器正常运行（`docker-compose ps`）
- [ ] 可以通过 `http://yourdomain.com` 访问（无需端口号）
- [ ] API请求正常（`http://yourdomain.com/api/health`）
- [ ] （如配置HTTPS）可以通过 `https://yourdomain.com` 访问

---

## 📚 相关文档

- [Nginx 官方文档](https://nginx.org/en/docs/)
- [Let's Encrypt 文档](https://letsencrypt.org/docs/)
- [Docker Compose 文档](https://docs.docker.com/compose/)

