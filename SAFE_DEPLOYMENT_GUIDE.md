# 安全部署指南 - 降低服务暂停风险

## 🎯 部署目标

- **零停机部署**：部署过程中服务不中断
- **快速回滚**：部署失败时能快速回退
- **数据安全**：部署过程中数据不丢失
- **风险可控**：分阶段部署，逐步验证

## 📋 部署前准备

### 1. 备份关键数据

```bash
# 1. 备份数据库
cp config.db config.db.backup.$(date +%Y%m%d_%H%M%S)

# 2. 备份配置文件
cp config.json config.json.backup.$(date +%Y%m%d_%H%M%S)

# 3. 备份日志（如果需要）
tar -czf decision_logs_backup_$(date +%Y%m%d_%H%M%S).tar.gz decision_logs/

# 4. 备份二进制文件（如果使用非Docker部署）
cp nofx nofx.backup.$(date +%Y%m%d_%H%M%S)
```

### 2. 验证当前服务状态

```bash
# 检查服务健康状态
curl -f http://localhost:8080/api/health || echo "服务未响应"

# 检查容器状态
docker compose ps

# 检查日志是否有错误
docker compose logs --tail=50 backend | grep -i error
```

### 3. 检查代码修改

```bash
# 查看修改的文件
git diff --stat

# 查看具体修改内容
git diff trader/auto_trader.go logger/decision_logger.go

# 确认没有未提交的重要修改
git status
```

## 🚀 安全部署策略

### 策略1：蓝绿部署（推荐）

**原理**：同时运行两个版本（蓝/绿），切换流量

**优点**：
- ✅ 零停机部署
- ✅ 快速回滚（切换流量即可）
- ✅ 可以并行验证新版本

**实现步骤**：

```bash
# 1. 构建新版本镜像（不启动）
docker compose -f docker-compose-v2.yml build --no-cache

# 2. 启动新版本容器（使用不同端口）
docker compose -f docker-compose-v2.yml up -d \
  --scale nofx-v2=2 \
  -p nofx-v2-backup:8081

# 3. 验证新版本健康
curl -f http://localhost:8081/api/health

# 4. 验证新版本功能（测试API）
curl -f http://localhost:8081/api/traders

# 5. 如果验证通过，切换流量
# 5.1 停止旧版本
docker compose stop nofx-v2

# 5.2 启动新版本（使用原端口）
docker compose -f docker-compose-v2.yml up -d

# 5.3 验证新版本运行正常
curl -f http://localhost:8080/api/health

# 6. 如果验证失败，立即回滚
docker compose restart  # 重启旧版本
```

### 策略2：滚动更新（Docker Compose）

**原理**：逐步替换容器，保持服务可用

**优点**：
- ✅ 平滑过渡
- ✅ 资源占用少
- ✅ 适合单个实例

**实现步骤**：

```bash
# 1. 构建新版本镜像
docker compose -f docker-compose-v2.yml build --no-cache

# 2. 停止旧容器（优雅停止）
docker compose stop nofx-v2

# 3. 启动新容器（使用新镜像）
docker compose -f docker-compose-v2.yml up -d nofx-v2

# 4. 等待健康检查通过（最多60秒）
timeout=60
elapsed=0
while [ $elapsed -lt $timeout ]; do
    if curl -f http://localhost:8080/api/health > /dev/null 2>&1; then
        echo "✅ 新版本健康检查通过"
        break
    fi
    sleep 2
    elapsed=$((elapsed + 2))
    echo "等待健康检查... ${elapsed}/${timeout}秒"
done

# 5. 如果健康检查失败，立即回滚
if [ $elapsed -ge $timeout ]; then
    echo "❌ 健康检查失败，回滚到旧版本"
    docker compose restart nofx-v2
    exit 1
fi

# 6. 验证功能
curl -f http://localhost:8080/api/traders
```

### 策略3：分阶段部署（推荐用于生产）

**原理**：分多个阶段部署，每阶段验证通过后再继续

**阶段1：仅后端部署**
```bash
# 1. 仅部署后端（前端继续使用旧版本）
docker compose -f docker-compose-v2.yml up -d --build nofx-v2

# 2. 验证后端健康
curl -f http://localhost:8080/api/health

# 3. 验证后端API
curl -f http://localhost:8080/api/traders

# 4. 观察日志（5分钟）
docker compose logs -f nofx-v2 --tail=100

# 5. 如果一切正常，继续下一阶段
```

**阶段2：前端部署**
```bash
# 1. 部署前端
docker compose -f docker-compose-v2.yml up -d --build nofx-frontend-v2

# 2. 验证前端访问
curl -f http://localhost/

# 3. 验证前端功能
# 打开浏览器测试关键功能
```

## 🔄 回滚机制

### 快速回滚脚本

```bash
#!/bin/bash
# rollback.sh - 快速回滚脚本

echo "🔄 开始回滚..."

# 1. 停止当前版本
docker compose -f docker-compose-v2.yml stop

# 2. 使用旧镜像（如果有）
# docker compose -f docker-compose-v2.yml pull <old-image>

# 3. 或者使用旧版本代码
git checkout <previous-commit>
docker compose -f docker-compose-v2.yml build --no-cache
docker compose -f docker-compose-v2.yml up -d

# 4. 验证回滚成功
sleep 5
if curl -f http://localhost:8080/api/health > /dev/null 2>&1; then
    echo "✅ 回滚成功"
else
    echo "❌ 回滚失败，需要手动检查"
    exit 1
fi
```

### 回滚检查清单

- [ ] 停止新版本容器
- [ ] 恢复数据库备份（如果需要）
- [ ] 恢复配置文件（如果需要）
- [ ] 启动旧版本容器
- [ ] 验证服务健康
- [ ] 验证功能正常
- [ ] 通知团队回滚完成

## ✅ 部署验证清单

### 部署后验证

```bash
# 1. 健康检查
curl -f http://localhost:8080/api/health

# 2. API可用性
curl -f http://localhost:8080/api/traders

# 3. 容器状态
docker compose ps

# 4. 日志检查（无错误）
docker compose logs --tail=100 backend | grep -i error

# 5. 资源使用（无异常）
docker stats --no-stream

# 6. 功能验证
# - 访问前端页面
# - 测试登录功能
# - 测试trader列表
# - 测试AI Learning页面
# - 验证PL计算是否正确
```

### 功能验证清单

- [ ] 前端页面可以访问
- [ ] 后端API响应正常
- [ ] 用户登录功能正常
- [ ] Trader列表显示正常
- [ ] AI Learning页面数据正确
- [ ] PL计算正确（quantity不为0）
- [ ] 交易决策正常执行
- [ ] 日志记录正常

## 📊 监控和告警

### 部署后监控（前30分钟）

```bash
# 1. 持续监控日志
docker compose logs -f backend --tail=50

# 2. 监控资源使用
watch -n 5 'docker stats --no-stream'

# 3. 监控健康检查
watch -n 10 'curl -s http://localhost:8080/api/health | jq'

# 4. 监控错误日志
watch -n 10 'docker compose logs --tail=20 backend | grep -i error'
```

### 告警指标

- ⚠️ 健康检查失败超过3次
- ⚠️ 错误日志持续出现
- ⚠️ 内存使用超过80%
- ⚠️ CPU使用超过90%
- ⚠️ API响应时间超过5秒
- ⚠️ 交易决策失败率增加

## 🛡️ 风险控制措施

### 1. 分时段部署

**建议部署时间**：
- ✅ 交易低峰期（如凌晨2-4点）
- ✅ 周末或节假日
- ❌ 交易高峰期
- ❌ 重要交易进行中

### 2. 灰度发布

**步骤**：
1. 先部署到测试环境
2. 验证通过后部署到生产环境
3. 先部署单个trader
4. 观察24小时后再部署其他trader

### 3. 功能开关

**如果代码支持**：
- 可以通过配置开关控制新功能
- 出现问题可以立即关闭新功能
- 不影响旧功能继续运行

### 4. 数据备份

**部署前必须备份**：
- 数据库文件（config.db）
- 配置文件（config.json）
- 日志文件（decision_logs/）
- 二进制文件（如果使用非Docker部署）

## 📝 部署脚本

### 安全部署脚本（完整版）

```bash
#!/bin/bash
# safe_deploy.sh - 安全部署脚本

set -e  # 遇到错误立即退出

# 配置
BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"
COMPOSE_FILE="docker-compose-v2.yml"
HEALTH_CHECK_URL="http://localhost:8080/api/health"
TIMEOUT=60

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 1. 备份
print_info "步骤1: 备份关键数据..."
mkdir -p "$BACKUP_DIR"
cp config.db "$BACKUP_DIR/" 2>/dev/null || print_warn "数据库文件不存在，跳过"
cp config.json "$BACKUP_DIR/" 2>/dev/null || print_warn "配置文件不存在，跳过"
tar -czf "$BACKUP_DIR/decision_logs.tar.gz" decision_logs/ 2>/dev/null || print_warn "日志目录不存在，跳过"
print_info "✅ 备份完成: $BACKUP_DIR"

# 2. 验证当前服务
print_info "步骤2: 验证当前服务状态..."
if curl -f "$HEALTH_CHECK_URL" > /dev/null 2>&1; then
    print_info "✅ 当前服务正常"
else
    print_error "❌ 当前服务异常，请先修复"
    exit 1
fi

# 3. 检查代码修改
print_info "步骤3: 检查代码修改..."
if git diff --quiet; then
    print_info "✅ 没有未提交的修改"
else
    print_warn "⚠️  有未提交的修改，请确认是否继续"
    read -p "继续部署? (y/n): " confirm
    if [ "$confirm" != "y" ]; then
        exit 1
    fi
fi

# 4. 构建新镜像
print_info "步骤4: 构建新版本镜像..."
docker compose -f "$COMPOSE_FILE" build --no-cache nofx-v2
print_info "✅ 镜像构建完成"

# 5. 停止旧容器
print_info "步骤5: 停止旧版本容器..."
docker compose -f "$COMPOSE_FILE" stop nofx-v2
print_info "✅ 旧容器已停止"

# 6. 启动新容器
print_info "步骤6: 启动新版本容器..."
docker compose -f "$COMPOSE_FILE" up -d nofx-v2
print_info "✅ 新容器已启动"

# 7. 等待健康检查
print_info "步骤7: 等待健康检查通过（最多${TIMEOUT}秒）..."
elapsed=0
while [ $elapsed -lt $TIMEOUT ]; do
    if curl -f "$HEALTH_CHECK_URL" > /dev/null 2>&1; then
        print_info "✅ 健康检查通过（耗时${elapsed}秒）"
        break
    fi
    sleep 2
    elapsed=$((elapsed + 2))
    echo -n "."
done
echo ""

# 8. 验证健康检查
if [ $elapsed -ge $TIMEOUT ]; then
    print_error "❌ 健康检查失败（超过${TIMEOUT}秒）"
    print_info "🔄 开始回滚..."
    docker compose -f "$COMPOSE_FILE" restart nofx-v2
    print_error "已回滚到旧版本，请检查问题"
    exit 1
fi

# 9. 功能验证
print_info "步骤8: 验证功能..."
if curl -f "http://localhost:8080/api/traders" > /dev/null 2>&1; then
    print_info "✅ API功能正常"
else
    print_error "❌ API功能异常，开始回滚..."
    docker compose -f "$COMPOSE_FILE" restart nofx-v2
    exit 1
fi

# 10. 日志检查
print_info "步骤9: 检查错误日志..."
if docker compose -f "$COMPOSE_FILE" logs --tail=50 nofx-v2 | grep -i error > /dev/null; then
    print_warn "⚠️  发现错误日志，请检查"
    docker compose -f "$COMPOSE_FILE" logs --tail=50 nofx-v2 | grep -i error
else
    print_info "✅ 未发现错误日志"
fi

# 11. 部署完成
print_info "✅ 部署完成！"
print_info "备份位置: $BACKUP_DIR"
print_info "查看日志: docker compose -f $COMPOSE_FILE logs -f nofx-v2"
print_info "回滚脚本: ./rollback.sh"
```

## 🎯 推荐部署流程

### 生产环境推荐流程

```bash
# 1. 准备阶段（30分钟）
./safe_deploy.sh prepare  # 备份、验证、检查

# 2. 部署阶段（5分钟）
./safe_deploy.sh deploy   # 构建、部署、验证

# 3. 验证阶段（30分钟）
# - 自动验证：健康检查、API测试
# - 手动验证：前端功能、PL计算、交易决策

# 4. 监控阶段（24小时）
# - 持续监控日志
# - 监控资源使用
# - 监控错误率
```

### 快速部署（低风险修改）

对于本次修改（修复quantity记录bug），属于**低风险修改**：

```bash
# 1. 备份（1分钟）
mkdir -p backups/$(date +%Y%m%d_%H%M%S)
cp config.db backups/$(date +%Y%m%d_%H%M%S)/

# 2. 构建和部署（2分钟）
docker compose -f docker-compose-v2.yml build --no-cache nofx-v2
docker compose -f docker-compose-v2.yml up -d --no-deps nofx-v2

# 3. 验证（1分钟）
sleep 10
curl -f http://localhost:8080/api/health && echo "✅ 部署成功"
```

## ⚠️ 注意事项

### 1. 数据库兼容性

- ✅ 本次修改不涉及数据库结构变更
- ✅ 无需数据库迁移
- ✅ 旧数据可以正常使用

### 2. 配置兼容性

- ✅ 本次修改不涉及配置格式变更
- ✅ 现有配置可以继续使用

### 3. 向后兼容性

- ✅ 代码有fallback机制，旧数据也能处理
- ✅ 新版本可以处理旧版本的日志格式

### 4. 性能影响

- ⚠️ 增加了一次API调用（GetPositions）
- ⚠️ 延迟增加约几十到几百毫秒
- ✅ 不影响交易执行

## 📊 部署后检查

### 立即检查（部署后5分钟内）

```bash
# 1. 健康检查
curl -f http://localhost:8080/api/health

# 2. 查看日志
docker compose -f docker-compose-v2.yml logs --tail=50 nofx-v2

# 3. 检查是否有panic或错误
docker compose -f docker-compose-v2.yml logs nofx-v2 | grep -i "panic\|fatal\|error"

# 4. 验证PL计算修复
# 等待下一个周期，检查日志中的quantity是否已记录
```

### 持续监控（部署后24小时）

```bash
# 1. 监控日志
docker compose -f docker-compose-v2.yml logs -f nofx-v2

# 2. 监控资源
docker stats --no-stream

# 3. 监控健康检查
watch -n 30 'curl -s http://localhost:8080/api/health | jq'
```

## 🚨 故障处理

### 如果部署失败

1. **立即回滚**：
   ```bash
   docker compose -f docker-compose-v2.yml restart nofx-v2
   ```

2. **检查日志**：
   ```bash
   docker compose -f docker-compose-v2.yml logs --tail=100 nofx-v2
   ```

3. **恢复备份**（如果需要）：
   ```bash
   cp backups/YYYYMMDD_HHMMSS/config.db config.db
   ```

### 如果功能异常

1. **检查日志**：查看是否有错误
2. **验证配置**：检查配置文件是否正确
3. **验证数据**：检查数据库是否正常
4. **回滚**：如果无法快速修复，立即回滚

## 📝 部署记录

建议记录每次部署：
- 部署时间
- 部署版本/commit
- 修改内容
- 部署结果
- 验证结果
- 问题记录

## 总结

### ✅ 本次修改特点

- **低风险**：修复bug，不涉及架构变更
- **向后兼容**：有fallback机制
- **无数据迁移**：无需数据库变更
- **快速回滚**：可以快速回退

### 🎯 推荐部署方式

**快速部署**（适合本次修改）：
```bash
# 1. 备份
cp config.db config.db.backup.$(date +%Y%m%d_%H%M%S)

# 2. 构建和部署
docker compose -f docker-compose-v2.yml up -d --build nofx-v2

# 3. 验证
sleep 10 && curl -f http://localhost:8080/api/health
```

**完整部署**（适合重大更新）：
```bash
./safe_deploy.sh
```

