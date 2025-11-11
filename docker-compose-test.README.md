# 测试 Docker 环境使用说明

## 概述

此测试 Docker 环境与线上运行的环境完全隔离，使用不同的端口和容器名称，避免冲突。

## 端口配置

- **后端 API**: `8081` (主机端口) → `8080` (容器端口)
- **前端 HTTP**: `8082` (主机端口) → `80` (容器端口)
- **前端 HTTPS**: `8443` (主机端口) → `443` (容器端口)

## 容器名称

- 后端: `nofx-trading-test`
- 前端: `nofx-frontend-test`

## 使用方法

### 1. 构建并启动测试环境

```bash
docker-compose -f docker-compose-test.yml up -d --build
```

### 2. 查看日志

```bash
# 查看所有服务日志
docker-compose -f docker-compose-test.yml logs -f

# 查看后端日志
docker-compose -f docker-compose-test.yml logs -f nofx-test

# 查看前端日志
docker-compose -f docker-compose-test.yml logs -f nofx-frontend-test
```

### 3. 停止测试环境

```bash
docker-compose -f docker-compose-test.yml down
```

### 4. 停止并删除卷数据（谨慎使用）

```bash
docker-compose -f docker-compose-test.yml down -v
```

## 环境变量

可以通过环境变量自定义端口：

```bash
export NOFX_BACKEND_PORT_TEST=8081
export NOFX_FRONTEND_PORT_TEST=8082
export NOFX_FRONTEND_HTTPS_PORT_TEST=8443
export NOFX_TIMEZONE=Asia/Shanghai
```

## 访问地址

- 前端 HTTP: http://localhost:8082
- 前端 HTTPS: https://localhost:8443 (需要 SSL 证书)
- 后端 API: http://localhost:8081/api/

## 健康检查

- 前端健康检查: http://localhost:8082/health
- 后端健康检查: http://localhost:8081/api/health

## 数据隔离

测试环境使用**独立的配置文件和数据库**，与线上环境完全隔离：

- **配置文件**: `config.json.test` (测试环境专用)
- **数据库**: `config.db.test` (测试环境专用)
- **日志目录**: `decision_logs_test/` (测试环境专用)
- **共享文件**: `prompts/`, `beta_codes.txt` (只读共享)

## 初始化测试环境

首次使用前，需要创建测试配置文件：

```bash
# 方法1: 复制线上配置文件作为基础
cp config.json config.json.test

# 方法2: 使用初始化脚本（如果有）
./init-test-env.sh
```

## 注意事项

1. **数据隔离**: 测试环境使用独立的配置文件（`config.json.test`）和数据库（`config.db.test`），与线上环境完全隔离
2. **日志隔离**: 测试环境的日志存储在 `decision_logs_test/` 目录，不会与线上日志混淆
3. **SSL 证书**: HTTPS 功能需要 `nginx/ssl/` 目录下的证书文件，如果不存在，HTTPS 将无法启动
4. **端口冲突**: 如果 8081、8082、8443 端口被占用，可以通过环境变量修改
5. **网络隔离**: 测试环境使用独立的 Docker 网络 `nofx-network-test`，与线上网络完全隔离
6. **首次启动**: 如果 `config.db.test` 不存在，首次启动时会自动创建

## 与线上环境的区别

| 项目 | 线上环境 | 测试环境 |
|------|---------|---------|
| 后端端口 | 8080 | 8081 |
| 前端 HTTP 端口 | 80 | 8082 |
| 前端 HTTPS 端口 | 443 | 8443 |
| 后端容器名 | nofx-trading-v2 | nofx-trading-test |
| 前端容器名 | nofx-frontend-v2 | nofx-frontend-test |
| Docker 网络 | nofx-network-v2 | nofx-network-test |
| Compose 文件 | docker-compose-v2.yml | docker-compose-test.yml |
| Nginx 配置 | nginx/nginx-v2.conf | nginx/nginx-test.conf |
| 配置文件 | config.json | config.json.test |
| 数据库文件 | config.db | config.db.test |
| 日志目录 | decision_logs/ | decision_logs_test/ |

