# 部署检查清单

## 📋 部署前检查

### ✅ 必须完成的步骤

- [ ] **1. 备份数据**
  ```bash
  ./safe_deploy.sh  # 自动备份
  # 或手动备份
  mkdir -p backups/$(date +%Y%m%d_%H%M%S)
  cp config.db backups/$(date +%Y%m%d_%H%M%S)/
  cp config.json backups/$(date +%Y%m%d_%H%M%S)/
  ```

- [ ] **2. 验证当前服务**
  ```bash
  curl -f http://localhost:8080/api/health
  docker compose ps
  ```

- [ ] **3. 检查代码修改**
  ```bash
  git diff --stat
  git status
  ```

- [ ] **4. 确认部署时间**
  - ✅ 推荐：交易低峰期（凌晨2-4点）
  - ✅ 推荐：周末或节假日
  - ❌ 避免：交易高峰期
  - ❌ 避免：重要交易进行中

### ⚠️ 建议完成的步骤

- [ ] **5. 测试环境验证**（如果有）
  ```bash
  # 在测试环境先部署验证
  ```

- [ ] **6. 通知团队**
  - 通知部署时间
  - 通知预计影响
  - 通知回滚计划

## 🚀 部署步骤

### 快速部署（低风险修改）

**适合本次修改**（修复quantity记录bug）：

```bash
# 1. 使用安全部署脚本（推荐）
./safe_deploy.sh

# 或手动部署
# 1. 备份
mkdir -p backups/$(date +%Y%m%d_%H%M%S)
cp config.db backups/$(date +%Y%m%d_%H%M%S)/

# 2. 构建和部署
docker compose -f docker-compose-v2.yml build --no-cache nofx-v2
docker compose -f docker-compose-v2.yml stop nofx-v2
docker compose -f docker-compose-v2.yml up -d nofx-v2

# 3. 验证
sleep 10
curl -f http://localhost:8080/api/health && echo "✅ 部署成功"
```

### 完整部署（高风险修改）

**适合重大更新**：

```bash
# 使用安全部署脚本（自动完成所有步骤）
./safe_deploy.sh
```

## ✅ 部署后验证

### 立即验证（部署后5分钟内）

- [ ] **1. 健康检查**
  ```bash
  curl -f http://localhost:8080/api/health
  ```

- [ ] **2. API可用性**
  ```bash
  curl -f http://localhost:8080/api/traders
  ```

- [ ] **3. 容器状态**
  ```bash
  docker compose -f docker-compose-v2.yml ps
  ```

- [ ] **4. 错误日志**
  ```bash
  docker compose -f docker-compose-v2.yml logs --tail=50 nofx-v2 | grep -i "panic\|fatal\|error"
  ```

- [ ] **5. 功能验证**
  - [ ] 访问前端页面
  - [ ] 测试登录功能
  - [ ] 测试trader列表
  - [ ] 测试AI Learning页面
  - [ ] 验证PL计算是否正确

### 持续监控（部署后30分钟）

- [ ] **1. 监控日志**
  ```bash
  docker compose -f docker-compose-v2.yml logs -f nofx-v2
  ```

- [ ] **2. 监控资源**
  ```bash
  docker stats --no-stream
  ```

- [ ] **3. 监控健康检查**
  ```bash
  watch -n 10 'curl -s http://localhost:8080/api/health | jq'
  ```

## 🚨 故障处理

### 如果部署失败

- [ ] **1. 立即回滚**
  ```bash
  ./safe_deploy.sh --rollback
  # 或手动回滚
  docker compose -f docker-compose-v2.yml restart nofx-v2
  ```

- [ ] **2. 检查日志**
  ```bash
  docker compose -f docker-compose-v2.yml logs --tail=100 nofx-v2
  ```

- [ ] **3. 恢复备份**（如果需要）
  ```bash
  cp backups/YYYYMMDD_HHMMSS/config.db config.db
  cp backups/YYYYMMDD_HHMMSS/config.json config.json
  ```

### 如果功能异常

- [ ] **1. 检查日志**
  - [ ] 查看是否有panic或fatal错误
  - [ ] 查看是否有类型断言错误
  - [ ] 查看是否有API调用失败

- [ ] **2. 验证配置**
  - [ ] 检查config.json是否正确
  - [ ] 检查环境变量是否正确

- [ ] **3. 验证数据**
  - [ ] 检查数据库是否正常
  - [ ] 检查日志文件是否正常

- [ ] **4. 回滚**
  ```bash
  ./safe_deploy.sh --rollback
  ```

## 📊 部署记录

### 记录内容

- [ ] 部署时间
- [ ] 部署版本/commit
- [ ] 修改内容
- [ ] 部署结果
- [ ] 验证结果
- [ ] 问题记录

### 示例记录

```markdown
## 部署记录 - 2025-11-04 16:30

**部署时间**: 2025-11-04 16:30:00
**部署版本**: commit abc123
**修改内容**: 
- 修复平仓时quantity未记录的问题
- 添加类型安全检查
- 添加错误日志

**部署结果**: ✅ 成功
**验证结果**: ✅ 通过
**问题记录**: 无
```

## 🎯 本次部署建议

### 部署方式

**推荐：快速部署**（低风险修改）

```bash
# 使用安全部署脚本
./safe_deploy.sh
```

### 部署时间

**推荐**：
- ✅ 交易低峰期（凌晨2-4点）
- ✅ 周末或节假日
- ✅ 非重要交易时段

### 验证重点

- ✅ 健康检查通过
- ✅ API功能正常
- ✅ 日志无panic或fatal错误
- ✅ PL计算正确（quantity不为0）

## 📝 总结

### 部署风险等级

- **本次修改风险**: 🟢 低（修复bug，向后兼容）
- **影响范围**: 🟢 小（仅影响记录，不影响交易）
- **回滚难度**: 🟢 低（快速回滚）

### 部署建议

- ✅ **可以使用快速部署**
- ✅ **建议使用安全部署脚本**
- ✅ **部署后持续监控30分钟**
- ✅ **准备回滚方案**

