# Docker Test V2 代码更新指南

## 概述

Docker Test V2 使用**多阶段构建**，代码在构建时编译成二进制文件打包进镜像。因此，**代码修改后必须重新构建镜像才能生效**。

## 快速更新步骤

### 方法 1：使用 docker-compose（推荐）

```bash
# 停止并删除现有容器
docker-compose -f docker-compose-test-v2.yml down

# 重新构建并启动（使用构建缓存，速度较快）
docker-compose -f docker-compose-test-v2.yml up -d --build

# 或者强制重建（不使用缓存，确保完全重新编译）
docker-compose -f docker-compose-test-v2.yml up -d --build --force-recreate --no-cache
```

### 方法 2：使用快速脚本

```bash
# 使用提供的快速启动脚本（会自动构建）
./test-docker-v2.sh
```

### 方法 3：仅重新构建后端服务（更快）

如果只修改了后端代码，可以只重新构建后端服务：

```bash
# 停止后端容器
docker-compose -f docker-compose-test-v2.yml stop nofx-test-v2

# 重新构建后端镜像
docker-compose -f docker-compose-test-v2.yml build --no-cache nofx-test-v2

# 启动后端容器
docker-compose -f docker-compose-test-v2.yml up -d nofx-test-v2
```

## 详细步骤说明

### 1. 确认代码修改已保存

```bash
# 检查修改的文件
git status

# 或查看具体修改
git diff trade_analytics/models.go
git diff trade_analytics/analyzer.go
git diff decision/engine.go
```

### 2. 停止现有容器

```bash
docker-compose -f docker-compose-test-v2.yml down
```

### 3. 清理旧镜像（可选，但推荐）

```bash
# 删除旧的测试镜像（释放空间）
docker rmi $(docker images | grep nofx.*test-v2 | awk '{print $3}') 2>/dev/null || true

# 或清理所有未使用的镜像
docker image prune -f
```

### 4. 重新构建并启动

```bash
# 标准重建（使用缓存）
docker-compose -f docker-compose-test-v2.yml up -d --build

# 完全重建（不使用缓存，确保代码完全更新）
docker-compose -f docker-compose-test-v2.yml build --no-cache nofx-test-v2
docker-compose -f docker-compose-test-v2.yml up -d
```

### 5. 验证更新

```bash
# 查看容器状态
docker-compose -f docker-compose-test-v2.yml ps

# 查看后端日志（确认新代码已加载）
docker-compose -f docker-compose-test-v2.yml logs -f nofx-test-v2

# 检查健康状态
curl http://localhost:8083/api/health
```

## 构建选项说明

### `--build`
- **作用**: 重新构建镜像
- **缓存**: 使用 Docker 构建缓存（如果代码未变化，会复用缓存层）
- **速度**: 较快（如果只有少量文件变化）

### `--no-cache`
- **作用**: 不使用构建缓存，完全重新构建
- **适用场景**: 
  - 确保代码完全更新
  - 依赖项发生变化
  - 构建缓存可能有问题时
- **速度**: 较慢（需要重新下载依赖和编译）

### `--force-recreate`
- **作用**: 强制重新创建容器（即使配置未变化）
- **适用场景**: 确保使用最新的镜像

## 验证代码更新是否生效

### 方法 1：查看日志

```bash
# 查看启动日志，确认新代码已编译
docker-compose -f docker-compose-test-v2.yml logs nofx-test-v2 | grep -i "启动\|start\|build"

# 查看运行时日志，确认新功能已生效
docker-compose -f docker-compose-test-v2.yml logs -f nofx-test-v2
```

### 方法 2：测试新功能

对于本次的熔断机制时间戳功能，可以：

1. **触发连续亏损**：确保有 2 次连续亏损的交易记录
2. **查看决策日志**：检查 prompt 中是否显示剩余暂停时间
3. **查看调试日志**：确认 `LastLossTimestamp` 字段已正确设置

```bash
# 查看决策日志中的熔断机制状态
grep -r "熔断机制状态" decision_logs_test/

# 查看调试日志
docker-compose -f docker-compose-test-v2.yml logs nofx-test-v2 | grep "LastLossTimestamp\|剩余暂停时间"
```

### 方法 3：检查编译时间

```bash
# 查看镜像构建时间（确认是新构建的）
docker images | grep nofx.*test-v2

# 查看容器创建时间
docker inspect nofx-trading-test-v2 | grep Created
```

## 常见问题

### Q1: 修改代码后，容器还在运行，需要重启吗？

**A**: 是的，必须重新构建镜像。因为代码已经编译成二进制文件，运行时不会重新编译。

### Q2: 只修改了 Go 代码，需要重新构建前端吗？

**A**: 不需要。前端和后端是独立的服务，只修改后端代码时，只需重新构建后端服务：

```bash
docker-compose -f docker-compose-test-v2.yml build --no-cache nofx-test-v2
docker-compose -f docker-compose-test-v2.yml up -d nofx-test-v2
```

### Q3: 构建很慢，有什么优化方法？

**A**: 
1. **使用构建缓存**（默认）：只重新编译变化的文件
2. **只构建后端**：如果只修改了后端代码
3. **使用国内镜像源**：Dockerfile 已配置阿里云镜像源
4. **并行构建**：Docker 会自动并行构建多个阶段

### Q4: 如何确认代码已更新？

**A**: 
1. 查看镜像构建时间
2. 查看容器日志中的调试信息
3. 测试新功能是否生效
4. 检查编译产物（二进制文件）的修改时间

### Q5: 构建失败怎么办？

**A**: 
```bash
# 查看详细构建日志
docker-compose -f docker-compose-test-v2.yml build --progress=plain nofx-test-v2

# 清理构建缓存后重试
docker builder prune
docker-compose -f docker-compose-test-v2.yml build --no-cache nofx-test-v2
```

## 本次修改的更新步骤

针对本次的熔断机制时间戳功能修改，执行以下步骤：

```bash
# 1. 确认代码已修改
git status
git diff trade_analytics/models.go trade_analytics/analyzer.go decision/engine.go

# 2. 停止现有容器
docker-compose -f docker-compose-test-v2.yml down

# 3. 重新构建后端（不使用缓存，确保完全更新）
docker-compose -f docker-compose-test-v2.yml build --no-cache nofx-test-v2

# 4. 启动容器
docker-compose -f docker-compose-test-v2.yml up -d

# 5. 查看日志确认启动成功
docker-compose -f docker-compose-test-v2.yml logs -f nofx-test-v2

# 6. 验证新功能（等待触发连续亏损后检查）
# 查看决策日志中的熔断机制状态
tail -f decision_logs_test/*/decision_*.json | grep -A 5 "熔断机制状态"
```

## 快速命令总结

```bash
# 完整重建（推荐）
docker-compose -f docker-compose-test-v2.yml down
docker-compose -f docker-compose-test-v2.yml up -d --build --force-recreate

# 仅重建后端（更快）
docker-compose -f docker-compose-test-v2.yml build --no-cache nofx-test-v2
docker-compose -f docker-compose-test-v2.yml up -d nofx-test-v2

# 查看日志
docker-compose -f docker-compose-test-v2.yml logs -f nofx-test-v2

# 检查状态
docker-compose -f docker-compose-test-v2.yml ps
```

---

**文档版本**: v1.0  
**创建日期**: 2025-01-18  
**最后更新**: 2025-01-18

