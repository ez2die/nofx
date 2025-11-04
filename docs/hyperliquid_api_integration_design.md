# Hyperliquid API 集成详细设计文档

## 1. 设计目标

1. **可替代 Binance API 获取市场数据**：完整实现 Hyperliquid API 客户端，支持所有必需的市场数据功能
2. **不影响当前数据消费端代码**：通过接口抽象，确保现有代码无需修改即可切换数据源
3. **支持参数化配置选择数据源**：通过配置文件动态选择使用 Binance 或 Hyperliquid 作为数据源

## 2. 架构设计

### 2.1 接口抽象

定义统一的市场数据客户端接口，所有数据源实现该接口：

```go
// market/client_interface.go
package market

// MarketDataClient 市场数据客户端接口
// 所有数据源（Binance、Hyperliquid等）必须实现此接口
type MarketDataClient interface {
    // GetExchangeInfo 获取交易所信息（交易对列表）
    GetExchangeInfo() (*ExchangeInfo, error)
    
    // GetKlines 获取K线数据
    // symbol: 交易对符号（如 "BTCUSDT"）
    // interval: 时间间隔（如 "3m", "4h"）
    // limit: 获取数量（Binance使用），对于Hyperliquid会转换为时间范围
    GetKlines(symbol, interval string, limit int) ([]Kline, error)
    
    // GetCurrentPrice 获取当前价格
    GetCurrentPrice(symbol string) (float64, error)
    
    // GetDataSourceName 返回数据源名称（用于日志和调试）
    GetDataSourceName() string
}
```

### 2.2 实现类

#### 2.2.1 Binance 客户端（现有代码重构）

将现有的 `APIClient` 重构为 `BinanceClient`，实现 `MarketDataClient` 接口：

```go
// market/binance_client.go
package market

type BinanceClient struct {
    baseURL string
    client  *http.Client
}

func NewBinanceClient() *BinanceClient {
    return &BinanceClient{
        baseURL: "https://fapi.binance.com",
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (c *BinanceClient) GetDataSourceName() string {
    return "binance"
}

// 实现 MarketDataClient 接口的所有方法
// （现有的 GetExchangeInfo、GetKlines、GetCurrentPrice 代码保持不变）
```

#### 2.2.2 Hyperliquid 客户端（新增）

创建新的 `HyperliquidClient`，实现 `MarketDataClient` 接口：

```go
// market/hyperliquid_client.go
package market

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "strconv"
    "strings"
    "time"
)

type HyperliquidClient struct {
    baseURL string
    client  *http.Client
    // 币种名称映射表（从Meta API获取）
    coinMap map[string]string // symbol -> coin name, e.g., "BTCUSDT" -> "BTC"
}

func NewHyperliquidClient() (*HyperliquidClient, error) {
    client := &HyperliquidClient{
        baseURL: "https://api.hyperliquid.xyz",
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
        coinMap: make(map[string]string),
    }
    
    // 初始化币种映射表
    if err := client.initializeCoinMap(); err != nil {
        log.Printf("⚠️  初始化Hyperliquid币种映射失败: %v", err)
        // 不返回错误，允许延迟初始化
    }
    
    return client, nil
}

func (c *HyperliquidClient) GetDataSourceName() string {
    return "hyperliquid"
}

// 初始化币种映射表
func (c *HyperliquidClient) initializeCoinMap() error {
    meta, err := c.getMeta()
    if err != nil {
        return err
    }
    
    for _, coin := range meta.Universe {
        symbol := coin.Name + "USDT" // Hyperliquid使用"BTC"，我们转换为"BTCUSDT"
        c.coinMap[symbol] = coin.Name
    }
    
    log.Printf("✓ Hyperliquid币种映射表已初始化，共 %d 个币种", len(c.coinMap))
    return nil
}

// 将系统symbol转换为Hyperliquid coin名称
// "BTCUSDT" -> "BTC"
func (c *HyperliquidClient) symbolToCoin(symbol string) string {
    symbol = strings.ToUpper(symbol)
    
    // 如果映射表未初始化，尝试直接提取币种名称
    if len(c.coinMap) == 0 {
        if strings.HasSuffix(symbol, "USDT") {
            return strings.TrimSuffix(symbol, "USDT")
        }
        return symbol
    }
    
    // 从映射表查找
    if coin, ok := c.coinMap[symbol]; ok {
        return coin
    }
    
    // 如果找不到，尝试直接提取
    if strings.HasSuffix(symbol, "USDT") {
        return strings.TrimSuffix(symbol, "USDT")
    }
    
    return symbol
}

// getMeta 获取Meta信息（内部方法）
func (c *HyperliquidClient) getMeta() (*HyperliquidMeta, error) {
    reqBody := map[string]interface{}{
        "type": "meta",
    }
    
    resp, err := c.postRequest("/info", reqBody)
    if err != nil {
        return nil, err
    }
    
    var meta HyperliquidMeta
    if err := json.Unmarshal(resp, &meta); err != nil {
        return nil, fmt.Errorf("解析Meta信息失败: %w", err)
    }
    
    return &meta, nil
}

// GetExchangeInfo 实现 MarketDataClient 接口
func (c *HyperliquidClient) GetExchangeInfo() (*ExchangeInfo, error) {
    meta, err := c.getMeta()
    if err != nil {
        return nil, err
    }
    
    // 转换Hyperliquid格式到ExchangeInfo格式
    exchangeInfo := &ExchangeInfo{
        Symbols: make([]SymbolInfo, 0),
    }
    
    for _, coin := range meta.Universe {
        symbol := coin.Name + "USDT"
        exchangeInfo.Symbols = append(exchangeInfo.Symbols, SymbolInfo{
            Symbol:            symbol,
            Status:            "TRADING", // Hyperliquid没有状态字段，假设都是交易中
            BaseAsset:         coin.Name,
            QuoteAsset:        "USDT",
            ContractType:      "PERPETUAL",
            PricePrecision:    8, // 默认精度
            QuantityPrecision: coin.SzDecimals,
        })
        
        // 更新币种映射表
        c.coinMap[symbol] = coin.Name
    }
    
    return exchangeInfo, nil
}

// GetKlines 实现 MarketDataClient 接口
func (c *HyperliquidClient) GetKlines(symbol, interval string, limit int) ([]Kline, error) {
    coin := c.symbolToCoin(symbol)
    
    // 将limit转换为时间范围
    // 计算所需的时间范围
    now := time.Now().Unix()
    startTime, endTime := c.calculateTimeRange(interval, limit, now)
    
    // 调用Hyperliquid API
    reqBody := map[string]interface{}{
        "type": "candleSnapshot",
        "req": map[string]interface{}{
            "coin":      coin,
            "interval":  interval,
            "startTime": startTime * 1000,  // 转换为毫秒
            "endTime":   endTime * 1000,
        },
    }
    
    resp, err := c.postRequest("/info", reqBody)
    if err != nil {
        return nil, err
    }
    
    var hlCandles []HyperliquidCandle
    if err := json.Unmarshal(resp, &hlCandles); err != nil {
        return nil, fmt.Errorf("解析K线数据失败: %w", err)
    }
    
    // 转换为系统内部Kline格式
    klines := make([]Kline, 0, len(hlCandles))
    for _, hlCandle := range hlCandles {
        kline, err := c.convertCandleToKline(hlCandle)
        if err != nil {
            log.Printf("⚠️  转换K线数据失败: %v", err)
            continue
        }
        klines = append(klines, kline)
    }
    
    // 限制返回数量（如果Hyperliquid返回的数据多于limit）
    if len(klines) > limit {
        klines = klines[len(klines)-limit:]
    }
    
    return klines, nil
}

// calculateTimeRange 根据interval和limit计算时间范围
func (c *HyperliquidClient) calculateTimeRange(interval string, limit int, now int64) (startTime, endTime int64) {
    endTime = now
    
    // 解析interval并计算开始时间
    var secondsPerCandle int64
    
    switch interval {
    case "1m":
        secondsPerCandle = 60
    case "3m":
        secondsPerCandle = 3 * 60
    case "5m":
        secondsPerCandle = 5 * 60
    case "15m":
        secondsPerCandle = 15 * 60
    case "1h":
        secondsPerCandle = 60 * 60
    case "4h":
        secondsPerCandle = 4 * 60 * 60
    case "1d":
        secondsPerCandle = 24 * 60 * 60
    default:
        // 默认使用1小时
        secondsPerCandle = 60 * 60
    }
    
    startTime = now - (int64(limit) * secondsPerCandle)
    
    return startTime, endTime
}

// convertCandleToKline 将Hyperliquid格式转换为系统内部Kline格式
func (c *HyperliquidClient) convertCandleToKline(hlCandle HyperliquidCandle) (Kline, error) {
    var kline Kline
    
    kline.OpenTime = hlCandle.T
    kline.CloseTime = hlCandle.TEnd
    
    var err error
    kline.Open, err = strconv.ParseFloat(hlCandle.O, 64)
    if err != nil {
        return kline, fmt.Errorf("解析开盘价失败: %w", err)
    }
    
    kline.High, err = strconv.ParseFloat(hlCandle.H, 64)
    if err != nil {
        return kline, fmt.Errorf("解析最高价失败: %w", err)
    }
    
    kline.Low, err = strconv.ParseFloat(hlCandle.L, 64)
    if err != nil {
        return kline, fmt.Errorf("解析最低价失败: %w", err)
    }
    
    kline.Close, err = strconv.ParseFloat(hlCandle.C, 64)
    if err != nil {
        return kline, fmt.Errorf("解析收盘价失败: %w", err)
    }
    
    kline.Volume, err = strconv.ParseFloat(hlCandle.V, 64)
    if err != nil {
        return kline, fmt.Errorf("解析成交量失败: %w", err)
    }
    
    kline.Trades = hlCandle.N
    
    // Hyperliquid不提供这些字段，使用计算值或默认值
    kline.QuoteVolume = kline.Volume * kline.Close
    kline.TakerBuyBaseVolume = 0  // Hyperliquid不提供
    kline.TakerBuyQuoteVolume = 0 // Hyperliquid不提供
    
    return kline, nil
}

// GetCurrentPrice 实现 MarketDataClient 接口
func (c *HyperliquidClient) GetCurrentPrice(symbol string) (float64, error) {
    coin := c.symbolToCoin(symbol)
    
    reqBody := map[string]interface{}{
        "type": "allMids",
    }
    
    resp, err := c.postRequest("/info", reqBody)
    if err != nil {
        return 0, err
    }
    
    var allMids map[string]string
    if err := json.Unmarshal(resp, &allMids); err != nil {
        return 0, fmt.Errorf("解析价格数据失败: %w", err)
    }
    
    priceStr, ok := allMids[coin]
    if !ok {
        return 0, fmt.Errorf("未找到币种 %s 的价格", coin)
    }
    
    price, err := strconv.ParseFloat(priceStr, 64)
    if err != nil {
        return 0, fmt.Errorf("解析价格失败: %w", err)
    }
    
    return price, nil
}

// postRequest 执行POST请求（内部辅助方法）
func (c *HyperliquidClient) postRequest(endpoint string, reqBody map[string]interface{}) ([]byte, error) {
    url := c.baseURL + endpoint
    
    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        return nil, fmt.Errorf("序列化请求失败: %w", err)
    }
    
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("创建请求失败: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("请求失败: %w", err)
    }
    defer resp.Body.Close()
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("读取响应失败: %w", err)
    }
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API返回错误 (status %d): %s", resp.StatusCode, string(body))
    }
    
    return body, nil
}

// Hyperliquid数据结构
type HyperliquidMeta struct {
    Universe []struct {
        Name         string `json:"name"`
        SzDecimals   int    `json:"szDecimals"`
        MaxLeverage  int    `json:"maxLeverage"`
        MarginTableId int  `json:"marginTableId"`
        IsDelisted   bool   `json:"isDelisted,omitempty"`
    } `json:"universe"`
}

type HyperliquidCandle struct {
    T     int64  `json:"t"`     // 开始时间（毫秒）
    TEnd  int64  `json:"T"`     // 结束时间（毫秒）
    S     string `json:"s"`    // 币种
    I     string `json:"i"`    // 间隔
    O     string `json:"o"`    // 开盘价
    C     string `json:"c"`    // 收盘价
    H     string `json:"h"`    // 最高价
    L     string `json:"l"`    // 最低价
    V     string `json:"v"`    // 成交量
    N     int    `json:"n"`    // 交易次数
}
```

### 2.3 工厂函数

创建工厂函数，根据配置选择创建哪个客户端：

```go
// market/client_factory.go
package market

import (
    "fmt"
    "log"
    "sync"
)

// DataSource 数据源类型
type DataSource string

const (
    DataSourceBinance     DataSource = "binance"
    DataSourceHyperliquid DataSource = "hyperliquid"
)

var (
    defaultClient MarketDataClient
    clientOnce    sync.Once
)

// SetDataSource 设置数据源（在应用启动时调用）
func SetDataSource(source DataSource) error {
    var err error
    clientOnce.Do(func() {
        switch source {
        case DataSourceBinance:
            log.Printf("📊 使用 Binance 作为市场数据源")
            defaultClient = NewBinanceClient()
        case DataSourceHyperliquid:
            log.Printf("📊 使用 Hyperliquid 作为市场数据源")
            var hlClient *HyperliquidClient
            hlClient, err = NewHyperliquidClient()
            if err != nil {
                log.Printf("⚠️  Hyperliquid客户端初始化失败，尝试继续: %v", err)
            }
            defaultClient = hlClient
        default:
            err = fmt.Errorf("不支持的数据源: %s", source)
        }
    })
    return err
}

// GetMarketDataClient 获取市场数据客户端（单例）
func GetMarketDataClient() MarketDataClient {
    if defaultClient == nil {
        // 默认使用Binance（向后兼容）
        log.Printf("⚠️  数据源未设置，默认使用 Binance")
        defaultClient = NewBinanceClient()
    }
    return defaultClient
}

// NewAPIClient 保持向后兼容的工厂函数（已废弃，但保留以避免破坏现有代码）
// Deprecated: 使用 GetMarketDataClient() 代替
func NewAPIClient() MarketDataClient {
    return GetMarketDataClient()
}
```

### 2.4 配置支持

在 `config.json` 中添加数据源配置：

```json
{
  "admin_mode": false,
  "beta_mode": false,
  "market_data_source": "hyperliquid",  // 新增：可选 "binance" 或 "hyperliquid"
  "leverage": {
    "btc_eth_leverage": 5,
    "altcoin_leverage": 5
  },
  ...
}
```

在应用启动时读取配置并设置数据源：

```go
// 在主程序启动时（main.go 或初始化函数中）
config, err := loadConfig("config.json")
if err != nil {
    log.Fatalf("加载配置失败: %v", err)
}

// 设置市场数据源
dataSource := market.DataSource(config.MarketDataSource)
if dataSource == "" {
    dataSource = market.DataSourceBinance // 默认使用Binance
}

if err := market.SetDataSource(dataSource); err != nil {
    log.Fatalf("设置数据源失败: %v", err)
}
```

### 2.5 现有代码修改（最小化改动）

#### 修改 1: `market/monitor.go`

```go
// 修改前：
apiClient := NewAPIClient()

// 修改后：
apiClient := GetMarketDataClient() // 使用统一的接口
```

所有其他代码保持不变，因为 `GetMarketDataClient()` 返回的接口与原来 `NewAPIClient()` 返回的结构体具有相同的方法签名。

#### 修改 2: `market/data.go`

```go
// 修改前（如果有直接使用）：
// 保持不变，因为 GetCurrentKlines 内部会调用 GetMarketDataClient()
```

#### 修改 3: 配置加载

在配置结构体中添加字段：

```go
// config/config.go
type Config struct {
    AdminMode        bool    `json:"admin_mode"`
    BetaMode         bool    `json:"beta_mode"`
    MarketDataSource string  `json:"market_data_source"` // 新增
    Leverage         LeverageConfig `json:"leverage"`
    ...
}
```

## 3. 实现步骤

### 步骤 1: 创建接口定义文件
- 创建 `market/client_interface.go`
- 定义 `MarketDataClient` 接口

### 步骤 2: 重构 Binance 客户端
- 将 `market/api_client.go` 重命名为 `market/binance_client.go`
- 让 `BinanceClient` 实现 `MarketDataClient` 接口
- 保持所有现有方法不变

### 步骤 3: 创建 Hyperliquid 客户端
- 创建 `market/hyperliquid_client.go`
- 实现 `MarketDataClient` 接口的所有方法
- 实现符号转换和时间范围计算

### 步骤 4: 创建工厂函数
- 创建 `market/client_factory.go`
- 实现 `SetDataSource` 和 `GetMarketDataClient`
- 保持 `NewAPIClient` 向后兼容

### 步骤 5: 修改现有代码
- 在 `market/monitor.go` 中使用 `GetMarketDataClient()`
- 在配置加载时调用 `SetDataSource()`

### 步骤 6: 更新配置
- 在 `config.json` 中添加 `market_data_source` 字段
- 更新配置结构体

## 4. 测试计划

### 4.1 单元测试
- 测试 HyperliquidClient 的所有方法
- 测试数据格式转换
- 测试符号转换逻辑
- 测试时间范围计算

### 4.2 集成测试
- 测试使用 Hyperliquid 作为数据源时的完整流程
- 测试数据源切换
- 测试向后兼容性

### 4.3 功能测试
- 验证 K 线数据获取正确性
- 验证价格数据获取正确性
- 验证交易对列表获取正确性
- 验证与现有消费端代码的兼容性

## 5. 错误处理

### 5.1 Hyperliquid API 错误
- 网络错误：记录日志，可配置重试
- API 错误：记录详细错误信息
- 数据格式错误：记录警告，跳过该条数据

### 5.2 数据源切换错误
- 如果 Hyperliquid 初始化失败，记录警告但不中断程序
- 可以考虑自动降级到 Binance（如果可用）

### 5.3 符号映射错误
- 如果找不到币种映射，记录警告
- 尝试自动提取币种名称（从 "BTCUSDT" 提取 "BTC"）

## 6. 性能考虑

### 6.1 API 调用优化
- Hyperliquid 使用时间范围查询，可能需要调整查询策略
- 缓存 Meta 信息，避免重复获取币种列表

### 6.2 内存优化
- 币种映射表使用 map，查找效率 O(1)
- K 线数据转换时避免不必要的内存分配

## 7. 日志和监控

### 7.1 日志输出
- 数据源选择日志：显示当前使用的数据源
- API 调用日志：记录每次 API 调用的结果
- 错误日志：详细记录所有错误信息

### 7.2 监控指标
- API 调用成功率
- API 响应时间
- 数据转换错误次数

## 8. 向后兼容性

### 8.1 API 兼容性
- 保持 `NewAPIClient()` 函数存在（标记为废弃）
- 所有方法签名保持不变
- 返回的数据格式保持一致

### 8.2 配置兼容性
- `market_data_source` 字段可选，默认使用 Binance
- 如果未配置，自动使用 Binance（保持现有行为）

## 9. 未来扩展

### 9.1 多数据源支持
- 支持同时使用多个数据源
- 实现数据源故障转移
- 实现数据源优先级配置

### 9.2 数据源健康检查
- 定期检查数据源可用性
- 自动切换数据源
- 记录数据源切换历史

### 9.3 其他数据源
- 支持添加更多数据源（如 CoinGecko、CoinMarketCap）
- 通过配置动态添加数据源

## 10. 文件清单

### 新增文件
1. `market/client_interface.go` - 接口定义
2. `market/hyperliquid_client.go` - Hyperliquid 客户端实现
3. `market/client_factory.go` - 工厂函数
4. `docs/hyperliquid_api_integration_design.md` - 本文档

### 修改文件
1. `market/api_client.go` → `market/binance_client.go` - 重构为 Binance 客户端
2. `market/monitor.go` - 使用统一的客户端接口
3. `config.json` - 添加数据源配置
4. `config/config.go` - 添加配置字段（如果存在）
5. `main.go` 或初始化函数 - 读取配置并设置数据源

## 11. 风险评估

### 11.1 技术风险
- **数据格式差异**：Hyperliquid 和 Binance 的数据格式不同，需要仔细转换
  - **缓解措施**：充分的单元测试和集成测试

- **符号映射错误**：币种名称不一致可能导致数据获取失败
  - **缓解措施**：自动符号转换，详细的错误日志

- **时间范围计算错误**：limit 转换为时间范围可能出现偏差
  - **缓解措施**：仔细测试时间计算逻辑，考虑边界情况

### 11.2 业务风险
- **数据源不可用**：如果 Hyperliquid API 不可用，可能导致系统无法运行
  - **缓解措施**：保留 Binance 作为备用，实现数据源切换

- **数据质量差异**：不同数据源的数据可能有细微差异
  - **缓解措施**：对比测试，确保数据质量可接受

## 12. 实施时间表

### 阶段 1: 基础实现（2-3天）
- 创建接口定义
- 实现 Hyperliquid 客户端基础功能
- 重构 Binance 客户端

### 阶段 2: 集成和测试（2-3天）
- 实现工厂函数
- 修改现有代码
- 单元测试和集成测试

### 阶段 3: 优化和文档（1-2天）
- 性能优化
- 错误处理完善
- 文档更新

### 总计：5-8天

## 13. 总结

本设计通过接口抽象实现了数据源的解耦，使得：
1. **完全替代 Binance API**：Hyperliquid 客户端提供完整功能
2. **零影响现有代码**：通过接口和向后兼容的工厂函数，现有代码无需修改
3. **灵活配置**：通过配置文件轻松切换数据源，支持运行时配置

该设计具有良好的可扩展性，未来可以轻松添加更多数据源支持。

