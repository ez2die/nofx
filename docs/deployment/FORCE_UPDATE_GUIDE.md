# 强制使用新代码和镜像的操作指南

## 操作步骤

### 1. 停止并删除容器
```bash
docker compose down nofx
# 或
docker compose stop nofx
docker compose rm -f nofx
```

### 2. 强制重新编译（不使用缓存）
```bash
docker compose build --no-cache nofx
```

### 3. 启动新容器
```bash
docker compose up -d nofx
```

### 4. 验证新代码
```bash
# 检查可执行文件时间戳
docker exec nofx-trading ls -lh /app/nofx

# 检查镜像ID
docker images nofx-nofx --format "{{.ID}} {{.CreatedAt}}"

# 检查容器使用的镜像
docker inspect nofx-trading --format='{{.Image}}'
```

## 验证结果

### 当前状态（02:54编译）
- ✅ **新可执行文件时间**：`Nov  4 02:54`（最新）
- ✅ **新镜像ID**：`sha256:130964295ca4344e9df27e9e58bdff5ff6c7efaa1de68d7d84fd23b16de197df`
- ✅ **容器已启动**：test trader已启动
- ✅ **后台日志监控**：已启动，持续跟踪

## 关键命令总结

### 完整流程
```bash
# 1. 停止容器
docker compose down nofx

# 2. 强制重新编译（不使用缓存）
docker compose build --no-cache nofx

# 3. 启动新容器
docker compose up -d nofx

# 4. 验证
docker exec nofx-trading ls -lh /app/nofx
docker logs nofx-trading -f | grep "决策周期\|跳过OI过滤\|成功获取"
```

### 快速重启（保留缓存）
```bash
docker compose restart nofx
```

### 强制重建（清除所有缓存）
```bash
docker compose build --no-cache --pull nofx
docker compose up -d --force-recreate nofx
```

## 预期效果

使用新代码后，应该看到：
1. ✅ `ℹ️  {symbol} OI数据为0（数据源可能不支持），跳过OI过滤，允许进入决策`
2. ✅ `✓ 成功获取 {symbol} 市场数据`
3. ✅ `✓ 成功获取 {count}/{total} 个币种的市场数据`
4. ✅ AI调用和执行日志

## 注意事项

1. **--no-cache**：确保所有层都重新构建，不使用任何缓存
2. **等待决策周期**：新代码需要等待下一个决策周期（约3分钟）才能看到效果
3. **日志验证**：持续监控日志，确认新代码是否生效

