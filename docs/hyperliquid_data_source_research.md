# Hyperliquid 数据源研究

## 测试结果

### ✅ API 可访问性
- **主API端点**: `https://api.hyperliquid.xyz/info` - **可以访问**
- **测试结果**: 成功返回 HTTP 200
- **网络延迟**: 正常（相比Binance的完全超时）

### ✅ 已验证可用的API端点

1. **Meta信息** (type: "meta")
   - 返回所有交易对的元信息（币种列表、杠杆、精度等）
   - **状态**: ✅ 可用

2. **所有币种中间价** (type: "allMids")
   - 返回所有币种的当前中间价格
   - **格式**: `{"BTC": "价格字符串", "ETH": "价格字符串", ...}`
   - **状态**: ✅ 可用
   - **代码已有实现**: `trader/hyperliquid_trader.go` 中的 `GetMarketPrice()` 方法

### ✅ 已验证可用的API端点

1. **K线数据** (type: "candleSnapshot")
   - **请求格式**: `{"type":"candleSnapshot","req":{"coin":"BTC","interval":"1h","startTime":毫秒时间戳,"endTime":毫秒时间戳}}`
   - **状态**: ✅ **可用且已验证**
   - **支持的间隔**: 3m, 15m, 1h, 4h 等
   - **返回格式**:
     ```json
     [{
       "t": 1762149600000,  // 开始时间（毫秒）
       "T": 1762153199999,  // 结束时间（毫秒）
       "s": "BTC",          // 币种
       "i": "1h",           // 间隔
       "o": "107566.0",     // 开盘价
       "c": "107446.0",     // 收盘价
       "h": "107670.0",     // 最高价
       "l": "106969.0",     // 最低价
       "v": "2279.1684",    // 成交量
       "n": 26794           // 交易次数
     }]
     ```
   - **数据格式对比**:
     - Hyperliquid: 使用 `startTime` 和 `endTime`（毫秒时间戳）
     - Binance: 使用 `limit`（数量限制）
   - **测试结果**: ✅ 成功获取3分钟和4小时K线数据

2. **Funding Rate** (type: "fundingHistory")
   - **状态**: ⚠️ 待测试（需要使用正确的请求格式）
   - **需求**: 需要获取当前和历史资金费率
   - **备注**: 需要查找正确的API格式

3. **Open Interest (OI)** (type: "openInterest")
   - **状态**: ⚠️ 待测试（需要使用正确的请求格式）
   - **需求**: 需要获取持仓量数据
   - **备注**: 可能需要使用不同的请求格式

## 代码现状

### 已有的 Hyperliquid 集成
- **交易器实现**: `trader/hyperliquid_trader.go`
- **依赖库**: `github.com/sonirico/go-hyperliquid`
- **已实现功能**:
  - ✅ 账户余额查询
  - ✅ 持仓查询
  - ✅ 下单/撤单
  - ✅ 获取当前价格 (`AllMids`)

### 当前市场数据获取方式
- **位置**: `market/api_client.go`
- **当前实现**: 完全依赖 Binance API (`https://fapi.binance.com`)
- **需要的功能**:
  1. 获取交易对列表 (`GetExchangeInfo`)
  2. 获取K线数据 (`GetKlines`) - **关键**
  3. 获取当前价格 (`GetCurrentPrice`)
  4. 获取OI数据 (`getOpenInterestData`)
  5. 获取Funding Rate (`getFundingRate`)

## 替代方案设计

### 方案1: 完全使用 Hyperliquid API（如果支持K线）
**优点**:
- 单一数据源，数据一致性高
- 如果用户交易在Hyperliquid，数据和交易在同一平台

**缺点**:
- 需要验证是否支持历史K线
- Hyperliquid的币种列表可能比Binance少
- 需要适配不同的数据格式

### 方案2: 混合数据源（Hyperliquid + 其他）
**设计**:
- 价格数据: 使用 Hyperliquid `AllMids`（已可用）
- K线数据: 需要找到替代源（如果Hyperliquid不支持）
- OI和Funding: 使用Hyperliquid（如果支持）

### 方案3: 改进缓存机制
**设计**:
- 当Binance API失败时，使用历史缓存数据
- 改进 `GetCurrentKlines` 方法，在失败时返回缓存数据而不是错误
- 仅在需要最新价格时使用 Hyperliquid API

## 结论

### ✅ 当前完全可用的功能
1. **价格数据**: 可以使用 Hyperliquid `AllMids` API 获取所有币种的当前价格 ✅
2. **交易对列表**: 可以使用 `Meta` API 获取所有交易对信息 ✅
3. **K线数据**: ✅ **已验证可用！**
   - 支持获取历史K线数据（candleSnapshot）
   - 支持多种时间间隔（3m, 15m, 1h, 4h等）
   - 需要使用 `startTime` 和 `endTime`（毫秒时间戳）
   - 数据格式完整（开盘价、收盘价、最高价、最低价、成交量、交易次数）
4. **网络访问**: Hyperliquid API 可以正常访问（相比Binance的完全无法连接）✅

### ⚠️ 待验证的功能
1. **OI数据**: 需要测试正确的API格式
2. **Funding Rate**: 需要测试正确的API格式

### 💡 推荐方案

#### 方案1: 使用 Hyperliquid 替代 Binance（推荐）✅
- **价格数据**: 使用 Hyperliquid `AllMids`（已可用，实时性好）✅
- **K线数据**: 使用 Hyperliquid `candleSnapshot`（已验证可用）✅
- **OI和Funding**: 设置为可选，失败时使用默认值（已有容错机制）
- **交易对列表**: 使用 Hyperliquid `Meta` API ✅

**优点**:
- **完全替代Binance API**：Hyperliquid API可以正常访问
- **数据完整性**：支持价格和K线数据
- **实时性好**：数据更新及时
- **无需代理**：可以直接访问

**实现要点**:
1. 创建 `market/hyperliquid_client.go` 实现与 `market/api_client.go` 相同的接口
2. 数据格式转换：将Hyperliquid格式转换为系统内部Kline格式
3. 时间处理：将Binance的`limit`方式改为Hyperliquid的`startTime/endTime`方式
4. 符号转换：Hyperliquid使用"BTC"，系统使用"BTCUSDT"

#### 方案2: 混合数据源（作为备用）
- **主要数据源**: Hyperliquid（已完全可用）
- **备用数据源**: 改进缓存机制，使用历史缓存数据（当Hyperliquid API失败时）
- **优点**: 提供数据源冗余，提高系统可靠性

#### 方案3: 使用其他数据源
- 考虑使用 CoinGecko API 获取价格和K线数据
- 或其他第三方数据聚合服务
- 但需要考虑API限制和成本

## 下一步行动

### 立即实施（已验证可行）✅
1. **实现 Hyperliquid 数据客户端**
   - 创建 `market/hyperliquid_client.go`
   - 实现 `GetExchangeInfo()`：使用 `Meta` API
   - 实现 `GetKlines()`：使用 `candleSnapshot` API
   - 实现 `GetCurrentPrice()`：使用 `AllMids` API
   - 数据格式转换：Hyperliquid格式 → 系统内部Kline格式

2. **实现数据源切换机制**
   - 在 `market/api_client.go` 中添加数据源选择
   - 默认使用 Hyperliquid（可访问）
   - 保留 Binance 作为备用（如果未来可用）

3. **符号转换和映射**
   - Hyperliquid使用"BTC"，系统使用"BTCUSDT"
   - 实现符号转换函数
   - 处理交易对名称映射

### 中期（完善功能）
1. **验证 OI 和 Funding Rate**
   - 测试 Hyperliquid API 的 OI 数据获取方式
   - 测试 Funding Rate 数据获取方式
   - 如果可用，实现相应功能

2. **性能优化**
   - 批量获取K线数据（减少API调用）
   - 实现本地缓存机制
   - 优化时间范围计算

### 长期（架构优化）
1. **多数据源支持**
   - 设计数据源接口抽象
   - 支持动态切换数据源（Hyperliquid/Binance/其他）
   - 实现数据源健康检查和故障转移
   - 支持数据源优先级配置

## API 文档参考

需要查找 Hyperliquid 官方 API 文档：
- 官方文档: https://hyperliquid.gitbook.io/hyperliquid-docs
- API端点: `https://api.hyperliquid.xyz/info`

## 测试命令

```bash
# 测试 Meta 信息
curl -s "https://api.hyperliquid.xyz/info" -X POST -H "Content-Type: application/json" -d '{"type":"meta"}' | jq '.universe[0:5]'

# 测试价格数据
curl -s "https://api.hyperliquid.xyz/info" -X POST -H "Content-Type: application/json" -d '{"type":"allMids"}' | jq '. | keys | length'

# 测试K线数据（3分钟，最近1小时）
current_time=$(date +%s) && start_time=$((current_time - 3600)) && end_time=$current_time
curl -s "https://api.hyperliquid.xyz/info" -X POST -H "Content-Type: application/json" \
  -d "{\"type\":\"candleSnapshot\",\"req\":{\"coin\":\"BTC\",\"interval\":\"3m\",\"startTime\":${start_time}000,\"endTime\":${end_time}000}}" | jq '.'

# 测试K线数据（4小时，最近24小时）
current_time=$(date +%s) && start_time=$((current_time - 86400)) && end_time=$current_time
curl -s "https://api.hyperliquid.xyz/info" -X POST -H "Content-Type: application/json" \
  -d "{\"type\":\"candleSnapshot\",\"req\":{\"coin\":\"BTC\",\"interval\":\"4h\",\"startTime\":${start_time}000,\"endTime\":${end_time}000}}" | jq '.'
```

## 数据格式对比

### Hyperliquid K线数据格式
```json
{
  "t": 1762149600000,  // 开始时间（毫秒）
  "T": 1762153199999,  // 结束时间（毫秒）
  "s": "BTC",          // 币种
  "i": "1h",           // 间隔
  "o": "107566.0",     // 开盘价（字符串）
  "c": "107446.0",     // 收盘价（字符串）
  "h": "107670.0",     // 最高价（字符串）
  "l": "106969.0",     // 最低价（字符串）
  "v": "2279.1684",    // 成交量（字符串）
  "n": 26794           // 交易次数（整数）
}
```

### 系统内部Kline格式
```go
type Kline struct {
    OpenTime            int64   // 开始时间（毫秒）
    Open                float64 // 开盘价
    High                float64 // 最高价
    Low                 float64 // 最低价
    Close               float64 // 收盘价
    Volume              float64 // 成交量
    CloseTime           int64   // 结束时间（毫秒）
    QuoteVolume         float64 // 成交额
    Trades              int     // 交易次数
    TakerBuyBaseVolume  float64 // 主动买入量
    TakerBuyQuoteVolume float64 // 主动买入额
}
```

### 转换映射
- `t` → `OpenTime`
- `T` → `CloseTime`
- `o` (字符串) → `Open` (float64)
- `h` (字符串) → `High` (float64)
- `l` (字符串) → `Low` (float64)
- `c` (字符串) → `Close` (float64)
- `v` (字符串) → `Volume` (float64)
- `n` (整数) → `Trades` (整数)
- `QuoteVolume` = `Volume * Close` (需要计算)
- `TakerBuyBaseVolume` = 0 (Hyperliquid不提供)
- `TakerBuyQuoteVolume` = 0 (Hyperliquid不提供)

## 实现建议

### 1. 创建 Hyperliquid 客户端
```go
// market/hyperliquid_client.go
package market

type HyperliquidClient struct {
    baseURL string
    client  *http.Client
}

func NewHyperliquidClient() *HyperliquidClient {
    return &HyperliquidClient{
        baseURL: "https://api.hyperliquid.xyz",
        client: &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *HyperliquidClient) GetKlines(coin, interval string, startTime, endTime int64) ([]Kline, error) {
    // 实现candleSnapshot API调用
    // 转换数据格式
}
```

### 2. 时间范围计算
```go
// 获取最近100根3分钟K线
now := time.Now().Unix()
startTime := now - (100 * 3 * 60) // 100根 * 3分钟 * 60秒
endTime := now
```

### 3. 符号转换
```go
// Hyperliquid使用"BTC"，系统使用"BTCUSDT"
func normalizeSymbol(symbol string) string {
    if strings.HasSuffix(symbol, "USDT") {
        return strings.TrimSuffix(symbol, "USDT")
    }
    return symbol
}
```

