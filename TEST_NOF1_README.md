# NOF1 Prompt 决策测试脚本

## 概述

`test_nof1_decision.go` 是一个独立的测试脚本，用于测试使用 `nof1.txt` prompt 进行交易决策的功能。

## 功能特点

- ✅ 使用 `nof1.txt` 作为主 prompt 模板
- ✅ 自动从 `config.json` 读取 DeepSeek API 密钥和配置
- ✅ 初始化市场监控器获取实时市场数据
- ✅ 构建完整的交易上下文（账户、持仓、候选币种等）
- ✅ 调用 AI 进行决策并显示完整结果

## 使用方法

### 1. 确保配置文件存在

脚本会从 `config.json` 读取以下配置：
- DeepSeek API 密钥（`hyperliquid_deepseek` trader 的 `deepseek_key`）
- 初始余额（`initial_balance`）
- 杠杆配置（`btc_eth_leverage`, `altcoin_leverage`）
- 默认币种列表（`default_coins`）

### 2. 运行测试脚本

```bash
# 方式1：直接运行
go run test_nof1_decision.go

# 方式2：先编译再运行
go build -o test_nof1_decision test_nof1_decision.go
./test_nof1_decision
```

## 输出说明

脚本会输出以下内容：

1. **配置信息**：加载的 DeepSeek API Key 和初始余额
2. **市场数据初始化**：监控器的初始化和数据加载状态
3. **系统提示词 (System Prompt)**：发送给 AI 的完整系统提示词（基于 `nof1.txt`）
4. **用户输入 (User Prompt)**：包含账户状态、市场数据等动态信息
5. **AI思维链 (CoT Trace)**：AI 的推理过程
6. **交易决策 (Decisions)**：结构化的交易决策列表
7. **JSON格式决策**：便于程序处理的 JSON 格式决策

## 示例输出

```
=== NOF1 Prompt 决策测试脚本 ===
✓ 从配置加载: DeepSeek API Key: sk-e7...00f2
✓ 初始余额: 500.00

📊 初始化市场监控器...
⏳ 等待市场数据加载...

🤖 初始化AI客户端...
✓ DeepSeek客户端已配置

📝 构建交易上下文...
📋 设置候选币种...
  - BTCUSDT
  - ETHUSDT
  - SOLUSDT
  - BNBUSDT

🚀 发起AI决策请求（使用 nof1.txt prompt）...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 决策完成！

📄 系统提示词 (System Prompt):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[完整的 nof1.txt prompt 内容]
...

💡 交易决策 (Decisions):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
决策 #1:
  币种: BTCUSDT
  动作: open_long
  杠杆: 5x
  仓位大小: 2500.00 USD
  止损: 98000.00
  止盈: 102000.00
  信心度: 75/100
  风险金额: 500.00 USD
  理由: [AI的决策理由]
...

📊 JSON格式决策:
[
  {
    "symbol": "BTCUSDT",
    "action": "open_long",
    "leverage": 5,
    "position_size_usd": 2500.00,
    "stop_loss": 98000.00,
    "take_profit": 102000.00,
    "confidence": 75,
    "risk_usd": 500.00,
    "reasoning": "..."
  }
]
```

## 配置要求

### config.json 结构

```json
{
  "traders": [
    {
      "id": "hyperliquid_deepseek",
      "deepseek_key": "sk-your-api-key",
      "initial_balance": 500
    }
  ],
  "leverage": {
    "btc_eth_leverage": 5,
    "altcoin_leverage": 5
  },
  "default_coins": [
    "BTCUSDT",
    "ETHUSDT",
    "SOLUSDT",
    "BNBUSDT"
  ]
}
```

## 注意事项

1. **市场数据依赖**：脚本需要从 Binance API 获取市场数据，确保网络连接正常
2. **API 密钥安全**：DeepSeek API Key 会从配置文件读取，请妥善保管
3. **数据加载时间**：脚本会在初始化后等待 3 秒让市场数据加载完成
4. **prompts 文件**：确保 `prompts/nof1.txt` 文件存在
5. **网络超时**：DeepSeek API 调用超时时间为 120 秒，请耐心等待

## 故障排除

### 问题：无法读取配置文件
**解决**：确保 `config.json` 文件在脚本运行目录下

### 问题：市场数据获取失败
**解决**：检查网络连接，确保可以访问 Binance API

### 问题：AI API 调用失败
**解决**：
- 检查 DeepSeek API Key 是否正确
- 检查 API 余额是否充足
- 查看错误日志了解具体原因

### 问题：找不到 nof1.txt prompt
**解决**：确保 `prompts/nof1.txt` 文件存在

## 代码结构

- **配置读取**：从 `config.json` 读取参数
- **市场监控**：初始化 `market.WSMonitor` 获取市场数据
- **AI 客户端**：初始化 `mcp.Client` 配置 DeepSeek
- **上下文构建**：构建 `decision.Context` 包含账户、持仓等信息
- **决策调用**：使用 `decision.GetFullDecisionWithCustomPrompt` 发起决策
- **结果输出**：格式化显示决策结果

## 扩展使用

### 修改候选币种

在代码中修改 `defaultCoins` 变量或从配置文件读取：

```go
defaultCoins := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT"}
```

### 添加持仓信息

如果需要测试已有持仓的情况，可以在构建 `Context` 时添加 `Positions`：

```go
ctx.Positions = []decision.PositionInfo{
    {
        Symbol: "BTCUSDT",
        Side: "long",
        EntryPrice: 100000,
        MarkPrice: 101000,
        // ... 其他字段
    },
}
```

### 使用不同的 prompt 模板

修改 `GetFullDecisionWithCustomPrompt` 的 `templateName` 参数：

```go
// 使用 nof1.txt
fullDecision, err := decision.GetFullDecisionWithCustomPrompt(ctx, mcpClient, "", false, "nof1")

// 使用 default.txt
fullDecision, err := decision.GetFullDecisionWithCustomPrompt(ctx, mcpClient, "", false, "default")
```

## 许可证

与主项目相同。

