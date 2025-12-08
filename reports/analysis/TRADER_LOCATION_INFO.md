# Test Docker V2 中正在运行的 Trader 信息

## 📊 正在运行的 Trader

### Trader ID
```
hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728
```

### Trader 名称
```
alids-5-leanopt-100
```

### 交易所配置 (DEX)
- **交易所类型**: Hyperliquid (DEX)
- **钱包地址**: `0x37ac0816b8fdc3b0b65ff1691299e981d5799616`
- **测试网**: 否 (testnet = 0)
- **Exchange ID**: `hyperliquid`

---

## 📁 日志位置

### 容器内路径
```
/app/decision_logs/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728/
```

### 主机路径
```
./decision_logs_test/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728/
```

### Docker Volume 映射
根据 `docker-compose-test-v2.yml`:
```yaml
volumes:
  - ./decision_logs_test:/app/decision_logs
```

### 访问日志的方法

#### 1. 从主机访问
```bash
# 查看日志目录
ls -lh decision_logs_test/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728/

# 查看最新日志
ls -lt decision_logs_test/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728/*.json | head -5

# 查看日志数量
find decision_logs_test/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728 -name "*.json" | wc -l
```

#### 2. 从容器内访问
```bash
# 进入容器
docker exec -it nofx-trading-test-v2 sh

# 查看日志目录
ls -lh /app/decision_logs/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728/

# 查看最新日志
ls -lt /app/decision_logs/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728/*.json | head -5
```

---

## 🔧 DEX 配置详情

### Hyperliquid 配置

#### 钱包信息
- **钱包地址**: `0x37ac0816b8fdc3b0b65ff1691299e981d5799616`
- **网络**: 主网 (testnet = 0)

#### 从数据库查询完整配置
```bash
sqlite3 config.db.test "SELECT * FROM exchanges WHERE id = 'hyperliquid';"
```

#### 在复盘模块中使用

要创建真实的 DEX 数据提供者（而不是模拟），需要：

1. **获取 ExchangeFillsProvider**
   ```go
   // 需要从 trader 配置中获取 Hyperliquid 客户端
   // 参考 trade_history/sync_hyperliquid.go
   ```

2. **在测试主程序中使用**
   ```go
   // 从数据库获取 trader 配置
   // 创建 Hyperliquid 客户端
   // 创建 HyperliquidFillsProvider
   // 创建 DEXDataProviderAdapter
   ```

---

## 📋 完整信息查询命令

### 查询运行中的 Trader
```bash
sqlite3 config.db.test "SELECT t.id, t.name, t.exchange_id, e.type, e.hyperliquid_wallet_addr FROM traders t JOIN exchanges e ON t.exchange_id = e.id WHERE t.is_running = 1;"
```

### 查询容器状态
```bash
docker ps --filter "name=nofx-trading-test-v2"
```

### 查看容器日志（最新）
```bash
docker logs nofx-trading-test-v2 --tail 50
```

### 查看容器中的日志文件
```bash
docker exec nofx-trading-test-v2 sh -c "ls -lt /app/decision_logs/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728/*.json | head -5"
```

---

## 🎯 复盘模块测试

### 使用真实的 Trader ID 测试复盘
```bash
RUN_ONCE=true USE_MOCK_DEX=true go run cmd/review_test/main.go \
  "hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728" \
  config.db.test
```

### 日志路径配置
复盘模块会从以下路径读取决策日志：
```
decision_logs_test/hyperliquid_84ea7a13-7bd4-4401-b4ab-ad63580f7b89_deepseek_1763366728/
```

---

## 📝 注意事项

1. **数据库**: Test Docker V2 使用 `config.db.test`
2. **日志目录**: Test Docker V2 使用 `decision_logs_test/`
3. **容器名**: `nofx-trading-test-v2`
4. **容器内路径**: `/app/decision_logs/`
5. **主机路径**: `./decision_logs_test/`

---

**最后更新**: 2025-11-21
**容器状态**: ✅ Running (Up 47 hours)

