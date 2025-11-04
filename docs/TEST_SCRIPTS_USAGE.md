# 测试脚本使用说明

## 修改内容

已修改 `test_nof1_decision.go` 和 `test_lean_decision.go`，添加了**真实 Hyperliquid 账户数据**支持。

## 使用方法

### 方式 1：使用模拟数据（默认）

快速测试 prompt 格式和兼容性：

```bash
# 测试 nof1.txt prompt
go run test_nof1_decision.go

# 测试 lean.txt prompt
go run test_lean_decision.go
```

**特点：**
- ✅ 快速执行，无需真实账户
- ✅ 适合测试 prompt 格式
- ✅ 持仓为空，账户数据为模拟值

### 方式 2：使用真实 Hyperliquid 数据

测试完整功能，包括持仓管理和真实账户状态：

```bash
# 测试 nof1.txt prompt（使用真实数据）
go run test_nof1_decision.go --real

# 测试 lean.txt prompt（使用真实数据）
go run test_lean_decision.go --real
```

**特点：**
- ✅ 获取真实的账户余额和持仓
- ✅ 测试持仓管理决策（关闭/持有）
- ✅ 测试 circuit breakers（drawdown >25%）
- ✅ 完整的风险评估场景

## 配置要求

### 基本配置（模拟数据模式）

在 `config.json` 中配置：

```json
{
  "traders": [
    {
      "id": "hyperliquid_deepseek",
      "deepseek_key": "your-deepseek-key",
      "initial_balance": 10000
    }
  ],
  "leverage": {
    "btc_eth_leverage": 5,
    "altcoin_leverage": 5
  }
}
```

### 完整配置（真实数据模式）

在 `config.json` 中额外配置 Hyperliquid 账户信息：

```json
{
  "traders": [
    {
      "id": "hyperliquid_deepseek",
      "deepseek_key": "your-deepseek-key",
      "initial_balance": 10000,
      "hyperliquid_private_key": "0x...",
      "hyperliquid_wallet_addr": "0x...",
      "hyperliquid_testnet": false
    }
  ]
}
```

**注意：**
- `hyperliquid_private_key`: Hyperliquid 钱包私钥（hex 格式）
- `hyperliquid_wallet_addr`: Hyperliquid 钱包地址
- `hyperliquid_testnet`: `true` 使用测试网，`false` 使用主网

## 功能对比

| 功能 | 模拟数据模式 | 真实数据模式 |
|------|------------|------------|
| 测试 prompt 格式 | ✅ | ✅ |
| 测试 JSON 输出 | ✅ | ✅ |
| 测试兼容性验证 | ✅ | ✅ |
| 测试持仓管理 | ❌ | ✅ |
| 测试 circuit breakers | ❌ | ✅ |
| 测试账户状态 | ❌ | ✅ |
| 测试风险计算 | ❌ | ✅ |

## 输出示例

### 模拟数据模式输出

```
=== NOF1 Prompt 决策测试脚本 ===
📝 使用模拟数据模式（默认）
💡 提示: 使用 --real 参数可启用真实账户数据
✓ 从配置加载: DeepSeek API Key: sk-...xxxx
✓ 初始余额: 10000.00
```

### 真实数据模式输出

```
=== NOF1 Prompt 决策测试脚本 ===
🔴 使用真实 Hyperliquid 账户数据模式
✓ 从配置加载: DeepSeek API Key: sk-...xxxx
✓ 初始余额: 10000.00
🔄 正在从 Hyperliquid 获取真实账户数据...
✓ Hyperliquid交易器初始化成功 (testnet=false, wallet=0x...)
✓ Hyperliquid 账户: 总净值=12500.00 (钱包12000.00+未实现500.00), 可用=11500.00, 保证金占用=1000.00
✓ 找到 2 个真实持仓
  - BTCUSDT long: 数量=0.1000, 入场价=95000.0000, 当前价=98000.0000, 盈亏=300.00 (3.16%)
  - ETHUSDT short: 数量=0.5000, 入场价=3500.0000, 当前价=3450.0000, 盈亏=25.00 (1.43%)
✓ 交易上下文构建完成: 持仓数=2, 保证金使用率=8.00%
```

## 注意事项

1. **安全提醒**：
   - 私钥和钱包地址信息存储在 `config.json` 中
   - 确保 `config.json` 不会被提交到版本控制系统
   - 建议使用环境变量或加密存储

2. **网络要求**：
   - 真实数据模式需要网络连接
   - 需要能够访问 Hyperliquid API

3. **测试网 vs 主网**：
   - 测试环境建议使用 `hyperliquid_testnet: true`
   - 生产环境使用 `hyperliquid_testnet: false`

4. **Lint 警告**：
   - 两个测试脚本在同一个目录中都有 `main` 函数
   - 这是正常的，因为它们应该单独运行
   - 可以忽略 "main redeclared" 的 lint 警告

## 故障排除

### 错误：未找到 hyperliquid_deepseek 配置

**解决**：确保 `config.json` 中有 `id` 为 `hyperliquid_deepseek` 的 trader 配置

### 错误：使用 --real 模式需要配置 hyperliquid_private_key

**解决**：在 trader 配置中添加 Hyperliquid 账户信息

### 错误：初始化 Hyperliquid 交易器失败

**解决**：
- 检查私钥格式是否正确（hex 格式，以 0x 开头）
- 检查钱包地址是否正确
- 检查网络连接

### 错误：获取账户余额失败

**解决**：
- 检查网络连接
- 检查账户是否存在
- 检查 API 访问权限

