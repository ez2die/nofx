# LEAN Prompt 决策测试脚本 v2 使用说明

## 📋 概述

`test_lean_decision_v2.go` 是适配新系统结构的测试脚本，使用数据库配置而不是配置文件。

## 🔄 主要变化（相比 v1）

### v1 (旧版本)
- 从 `config.json` 读取配置
- 手动解析 JSON 配置

### v2 (新版本)
- ✅ 从数据库 (`config.db`) 读取配置
- ✅ 使用 `config.Database` API
- ✅ 支持从数据库加载交易员配置
- ✅ 支持从数据库加载AI模型和交易所配置

## 🚀 使用方法

### 基本用法

```bash
cd /root/nofx
export PATH=$PATH:/usr/local/go/bin
go run test_lean_decision_v2.go
```

### 命令行参数

```bash
go run test_lean_decision_v2.go [选项]
```

**选项：**

- `--real`: 使用真实 Hyperliquid 账户数据（需要配置）
- `--source`: 市场数据源 (`binance` 或 `hyperliquid`)，默认: `binance`
- `--user`: 用户ID，默认: `default`
- `--trader`: 交易员ID（可选，如果指定则从数据库加载该交易员配置）
- `--db`: 数据库路径，默认: `config.db`

### 示例

#### 1. 使用默认配置（模拟数据）

```bash
go run test_lean_decision_v2.go
```

#### 2. 使用指定交易员配置

```bash
go run test_lean_decision_v2.go --trader hyperliquid_deepseek_xxx --user default
```

#### 3. 使用真实 Hyperliquid 账户数据

```bash
go run test_lean_decision_v2.go --real --source hyperliquid --trader hyperliquid_deepseek_xxx
```

#### 4. 使用自定义数据库路径

```bash
go run test_lean_decision_v2.go --db /path/to/config.db
```

## 📝 配置要求

### 数据库配置

脚本需要从数据库读取以下配置：

1. **AI模型配置** (`ai_models` 表)
   - 至少需要一个启用的 DeepSeek 配置
   - 需要配置 `api_key`

2. **交易所配置** (`exchanges` 表) - 如果使用 `--real` 模式
   - 需要配置 Hyperliquid 交易所
   - 需要配置 `hyperliquid_wallet_addr`
   - 需要配置 `secret_key`（作为 private key）

3. **系统配置** (`system_config` 表)
   - `market_data_source`: 市场数据源
   - `btc_eth_leverage`: BTC/ETH杠杆倍数
   - `altcoin_leverage`: 山寨币杠杆倍数
   - `default_coins`: 默认币种列表（JSON格式）

4. **交易员配置** (`traders` 表) - 如果使用 `--trader` 参数
   - 交易员ID对应的配置
   - 关联的AI模型和交易所配置

### 配置方式

通过 Web 界面配置：
1. 访问 `http://localhost:8080`
2. 配置 AI 模型（添加 DeepSeek API Key）
3. 配置交易所（添加 Hyperliquid 配置）
4. 创建交易员（可选）

或通过数据库直接配置（需要了解数据库结构）。

## 🔍 功能说明

### 1. 配置加载流程

1. **初始化数据库**
   ```go
   database, err := config.NewDatabase("config.db")
   ```

2. **读取系统配置**
   - 市场数据源
   - 杠杆配置
   - 默认币种列表

3. **加载交易员配置**（如果指定了 `--trader`）
   - 从数据库加载交易员完整配置
   - 包括关联的AI模型和交易所配置

4. **加载AI模型配置**（如果没有指定交易员）
   - 从数据库查找第一个启用的 DeepSeek 配置

### 2. 市场数据源初始化

```go
market.SetDataSource(market.DataSource(dataSource))
```

支持的数据源：
- `binance`: Binance 市场数据
- `hyperliquid`: Hyperliquid 市场数据

### 3. 真实账户数据模式 (`--real`)

当使用 `--real` 参数时：
1. 从数据库读取 Hyperliquid 交易所配置
2. 初始化 Hyperliquid 交易器
3. 获取真实账户余额和持仓
4. 构建包含真实数据的交易上下文

### 4. 决策请求

使用 `lean.txt` 模板调用决策引擎：
```go
fullDecision, err := decision.GetFullDecisionWithCustomPrompt(
    ctx, mcpClient, "", false, "lean"
)
```

## 📊 输出说明

脚本会输出：

1. **系统提示词** - 完整的系统提示（包含 lean.txt 模板）
2. **用户输入** - 构建的交易上下文
3. **AI思维链** - AI的思考过程
4. **交易决策** - 具体的决策列表
5. **JSON格式决策** - 便于程序处理
6. **兼容性验证总结** - 验证决策是否符合 lean.txt 要求

## ✅ 验证规则

脚本会验证以下内容：

1. **Action 有效性**
   - 必须是: `open_long`, `open_short`, `close_long`, `close_short`, `hold`, `wait`

2. **信心度** (开仓时)
   - 必须 ≥ 70（lean.txt 要求）

3. **杠杆**
   - 不能超过配置的上限
   - BTC/ETH 使用 `btc_eth_leverage`
   - 其他币种使用 `altcoin_leverage`

4. **风险回报比**
   - 必须 ≥ 3:1（lean.txt 要求）

5. **风险金额**
   - 建议在 5-8% 账户余额范围内

## 🔧 故障排除

### 问题 1: 找不到 DeepSeek 配置

**错误**: `❌ 未找到启用的DeepSeek配置`

**解决**:
1. 通过 Web 界面配置 AI 模型
2. 确保 DeepSeek 配置已启用
3. 确保 API Key 已配置

### 问题 2: 数据库不存在

**错误**: `❌ 初始化数据库失败`

**解决**:
1. 确保 `config.db` 文件存在
2. 或运行主程序一次，会自动创建数据库

### 问题 3: 真实账户数据获取失败

**错误**: `❌ 获取账户余额失败`

**解决**:
1. 检查 Hyperliquid 交易所配置是否正确
2. 检查 `hyperliquid_wallet_addr` 和 `secret_key` 是否配置
3. 检查网络连接

### 问题 4: 市场数据源初始化失败

**错误**: `❌ 设置市场数据源失败`

**解决**:
1. 检查数据源名称是否正确（`binance` 或 `hyperliquid`）
2. 检查网络连接

## 📝 与旧版本对比

| 特性 | v1 (旧) | v2 (新) |
|------|---------|---------|
| 配置来源 | config.json | config.db 数据库 |
| 配置API | 手动解析JSON | config.Database API |
| 交易员配置 | 从JSON读取 | 从数据库加载 |
| AI模型配置 | 从JSON读取 | 从数据库加载 |
| 交易所配置 | 从JSON读取 | 从数据库加载 |
| 系统配置 | 从JSON读取 | 从数据库读取 |
| 用户支持 | 单一用户 | 多用户支持 |

## 🎯 使用场景

1. **测试 lean.txt prompt**
   - 验证 lean.txt 模板是否正常工作
   - 验证决策是否符合 lean.txt 要求

2. **调试决策引擎**
   - 查看完整的系统提示词和用户输入
   - 查看 AI 的思维链分析

3. **验证数据库配置**
   - 验证从数据库加载的配置是否正确
   - 测试交易员配置是否完整

4. **真实账户测试**
   - 使用真实 Hyperliquid 账户数据测试
   - 验证决策在真实环境下的表现

## 📚 相关文件

- `test_lean_decision.go` - 旧版本（使用 config.json）
- `prompts/lean.txt` - LEAN 提示词模板
- `config.db` - 配置数据库
- `decision/engine.go` - 决策引擎

---

**创建时间**: 2025-01-04  
**版本**: v2.0  
**适配系统**: 新数据库驱动架构

