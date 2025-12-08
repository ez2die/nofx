# 测试 Docker 环境 V2 使用说明

## 概述

此测试 Docker 环境 V2 是新的测试环境，与原有测试环境和线上运行环境完全隔离，使用不同的端口和容器名称，避免冲突。

## 端口配置

- **后端 API**: `8083` (主机端口) → `8080` (容器端口)
- **前端 HTTP**: `8084` (主机端口) → `80` (容器端口)
- **前端 HTTPS**: `8444` (主机端口) → `443` (容器端口)

## 容器名称

- 后端: `nofx-trading-test-v2`
- 前端: `nofx-frontend-test-v2`

## 优化特性

### 阿里云友好的镜像源

本配置已优化为使用阿里云友好的镜像源，大幅提升构建速度：

1. **Alpine Linux 镜像**: 使用 `mirrors.aliyun.com`
2. **Go 模块代理**: 使用 `goproxy.cn` 和 `goproxy.io`
3. **Go 校验和数据库**: 使用 `sum.golang.google.cn`
4. **npm 镜像**: 使用 `registry.npmmirror.com` (淘宝镜像)
5. **GitHub 代理**: 使用 `mirror.ghproxy.com` 和 `ghproxy.com`

## 使用方法

### 1. 构建并启动测试环境 V2

```bash
docker-compose -f docker-compose-test-v2.yml up -d --build
```

### 2. 查看日志

```bash
# 查看所有服务日志
docker-compose -f docker-compose-test-v2.yml logs -f

# 查看后端日志
docker-compose -f docker-compose-test-v2.yml logs -f nofx-test-v2

# 查看前端日志
docker-compose -f docker-compose-test-v2.yml logs -f nofx-frontend-test-v2
```

### 3. 停止测试环境

```bash
docker-compose -f docker-compose-test-v2.yml down
```

### 4. 停止并删除卷数据（谨慎使用）

```bash
docker-compose -f docker-compose-test-v2.yml down -v
```

### 5. 重新构建（强制重建）

```bash
docker-compose -f docker-compose-test-v2.yml up -d --build --force-recreate
```

## 环境变量

可以通过环境变量自定义端口：

```bash
export NOFX_BACKEND_PORT_TEST_V2=8083
export NOFX_FRONTEND_PORT_TEST_V2=8084
export NOFX_FRONTEND_HTTPS_PORT_TEST_V2=8444
export NOFX_TIMEZONE=Asia/Shanghai
```

## 访问地址

- 前端 HTTP: http://localhost:8084
- 前端 HTTPS: https://localhost:8444 (需要 SSL 证书)
- 后端 API: http://localhost:8083/api/

## 健康检查

- 前端健康检查: http://localhost:8084/health
- 后端健康检查: http://localhost:8083/api/health

## 数据隔离

测试环境 V2 使用**独立的配置文件和数据库**，与线上环境和原有测试环境完全隔离：

- **配置文件**: `config.json.test` (与原测试环境共享，但容器独立)
- **数据库**: `config.db.test` (与原测试环境共享，但容器独立)
- **日志目录**: `decision_logs_test/` (与原测试环境共享，但容器独立)
- **共享文件**: `prompts/`, `beta_codes.txt` (只读共享)

## 初始化测试环境

首次使用前，需要创建测试配置文件：

```bash
# 方法1: 复制线上配置文件作为基础
cp config.json config.json.test

# 方法2: 使用初始化脚本（如果有）
./init-test-env.sh
```

## 端口对比表

| 环境 | 后端端口 | 前端HTTP端口 | 前端HTTPS端口 |
|------|---------|-------------|-------------|
| 线上环境 | 8080 | 80 | 443 |
| 测试环境 (原) | 8081 | 8082 | 8443 |
| 测试环境 V2 (新) | **8083** | **8084** | **8444** |

## 容器名称对比表

| 环境 | 后端容器名 | 前端容器名 | Docker网络 |
|------|-----------|-----------|-----------|
| 线上环境 | nofx-trading | nofx-frontend | nofx-network |
| 测试环境 (原) | nofx-trading-test | nofx-frontend-test | nofx-network-test |
| 测试环境 V2 (新) | **nofx-trading-test-v2** | **nofx-frontend-test-v2** | **nofx-network-test-v2** |

## 构建优化说明

### 镜像源优化

1. **Alpine 包管理器**: 自动检测 Alpine 版本并使用对应的阿里云镜像源
2. **Go 模块下载**: 使用 `goproxy.cn` 作为主要代理，`goproxy.io` 作为备用
3. **Go 校验和**: 使用 `sum.golang.google.cn` 加速校验和验证
4. **npm 包下载**: 使用淘宝镜像 `registry.npmmirror.com`
5. **GitHub 资源**: 优先使用 `mirror.ghproxy.com` 和 `ghproxy.com` 代理

### 构建速度提升

使用阿里云友好的镜像源后，构建速度预计可提升：
- **Alpine 包下载**: 3-5倍提升
- **Go 模块下载**: 5-10倍提升
- **npm 包下载**: 3-5倍提升
- **GitHub 资源**: 2-3倍提升

## 注意事项

1. **数据隔离**: 测试环境 V2 使用独立的容器，但配置文件与原测试环境共享（`config.json.test` 和 `config.db.test`）
2. **日志隔离**: 测试环境的日志存储在 `decision_logs_test/` 目录，与原测试环境共享
3. **SSL 证书**: HTTPS 功能需要 `nginx/ssl/` 目录下的证书文件，如果不存在，HTTPS 将无法启动
4. **端口冲突**: 如果 8083、8084、8444 端口被占用，可以通过环境变量修改
5. **网络隔离**: 测试环境 V2 使用独立的 Docker 网络 `nofx-network-test-v2`，与其他环境完全隔离
6. **首次启动**: 如果 `config.db.test` 不存在，首次启动时会自动创建
7. **构建缓存**: 使用 `--build` 时会利用 Docker 构建缓存，如需完全重建，使用 `--no-cache`

## 故障排查

### 构建失败

如果构建失败，可以尝试：

```bash
# 清理构建缓存
docker builder prune

# 强制重新构建（不使用缓存）
docker-compose -f docker-compose-test-v2.yml build --no-cache

# 查看详细构建日志
docker-compose -f docker-compose-test-v2.yml build --progress=plain
```

### 端口冲突

如果端口被占用：

```bash
# 检查端口占用
netstat -tuln | grep -E '8083|8084|8444'

# 或使用 ss 命令
ss -tuln | grep -E '8083|8084|8444'

# 修改环境变量使用其他端口
export NOFX_BACKEND_PORT_TEST_V2=8085
export NOFX_FRONTEND_PORT_TEST_V2=8086
export NOFX_FRONTEND_HTTPS_PORT_TEST_V2=8445
```

### 镜像源问题

如果某个镜像源不可用，Dockerfile 已配置了多个备用源，会自动切换。如果所有源都不可用，可以：

1. 检查网络连接
2. 检查防火墙设置
3. 尝试使用 VPN 或代理

## 与原有测试环境的区别

| 项目 | 测试环境 (原) | 测试环境 V2 (新) |
|------|-------------|----------------|
| 后端端口 | 8081 | 8083 |
| 前端 HTTP 端口 | 8082 | 8084 |
| 前端 HTTPS 端口 | 8443 | 8444 |
| 后端容器名 | nofx-trading-test | nofx-trading-test-v2 |
| 前端容器名 | nofx-frontend-test | nofx-frontend-test-v2 |
| Docker 网络 | nofx-network-test | nofx-network-test-v2 |
| Compose 文件 | docker-compose-test.yml | docker-compose-test-v2.yml |
| Nginx 配置 | nginx/nginx-test.conf | nginx/nginx-test-v2.conf |
| 镜像源优化 | 基础优化 | **全面优化（阿里云友好）** |

## 快速开始

```bash
# 1. 确保配置文件存在
cp config.json config.json.test 2>/dev/null || echo "config.json.test already exists"

# 2. 构建并启动
docker-compose -f docker-compose-test-v2.yml up -d --build

# 3. 查看日志
docker-compose -f docker-compose-test-v2.yml logs -f

# 4. 访问前端
# 浏览器打开: http://localhost:8084
```

---

**文档版本**: v1.0  
**创建日期**: 2025-01-XX  
**最后更新**: 2025-01-XX

