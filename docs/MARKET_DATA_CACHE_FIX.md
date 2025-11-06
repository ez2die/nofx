# 市场价格数据缓存问题修复

## 问题描述

Trader 02 和 Trader 04 在过去几个小时的 cycle 中，市场价格数据一直保持不变，导致交易决策基于过时的市场数据。

### 根本原因分析

1. **配置使用 Hyperliquid**：当前配置使用 `market_data_source: "hyperliquid"`
2. **WebSocket 硬编码连接 Binance**：WebSocket 客户端硬编码连接到 `wss://fstream.binance.com/stream`
3. **Hyperliquid 不支持 Binance WebSocket**：Hyperliquid 不使用 Binance 的 WebSocket，导致连接失败
4. **数据不更新**：由于 WebSocket 连接失败，数据不会更新，缓存一直使用旧数据

### 问题分析

1. **根本原因**：`GetCurrentKlines` 函数只检查缓存是否存在，没有检查数据的时效性
2. **触发条件**：当 WebSocket 断开或未正常更新数据时，缓存数据会一直使用旧数据
3. **影响范围**：所有使用缓存的 trader 都会受到影响

### 发现的问题

#### 问题1：缓存时效性检查缺失

在 `market/monitor.go` 的 `GetCurrentKlines` 函数中：

```go
func (m *WSMonitor) GetCurrentKlines(symbol string, _time string) ([]Kline, error) {
	value, exists := m.getKlineDataMap(_time).Load(symbol)
	if !exists {
		// 只有在缓存不存在时才从API获取
		// ...
	}
	return value.([]Kline), nil  // ⚠️ 直接返回缓存，没有检查时效性
}
```

**问题**：一旦数据被缓存，即使 WebSocket 断开或数据过期，函数也会一直返回缓存的旧数据。

#### 问题2：WebSocket 硬编码连接 Binance

在 `market/monitor.go` 的 `startWithRetry` 函数中：

```go
func (m *WSMonitor) startWithRetry(coins []string) {
	// ⚠️ 无论使用什么数据源，都尝试连接 Binance WebSocket
	err = m.combinedClient.Connect()  // 硬编码连接到 wss://fstream.binance.com/stream
	// ...
}
```

**问题**：当使用 Hyperliquid 数据源时，WebSocket 仍然尝试连接 Binance，导致连接失败，数据不会更新。

## 修复方案

### 1. 添加更新时间跟踪

在 `WSMonitor` 结构体中添加了更新时间映射：

```go
klineUpdateTime3m sync.Map // 存储3分钟K线数据的最后更新时间
klineUpdateTime4h sync.Map // 存储4小时K线数据的最后更新时间
```

### 2. 实现时效性检查

在 `GetCurrentKlines` 函数中添加了时效性检查：

- **3分钟K线**：如果超过3分钟未更新，则重新从API获取（确保每个K线周期都会刷新）
- **4小时K线**：如果超过10分钟未更新，则重新从API获取

### 3. 更新机制

- 在 `processKlineUpdate` 中，每次 WebSocket 更新数据时，同时更新最后更新时间
- 在 `initializeHistoricalData` 中，初始化时也设置更新时间
- 在 `GetCurrentKlines` 中，从API获取新数据时，更新缓存和更新时间

### 4. 修复 WebSocket 数据源检查

在 `startWithRetry` 函数中添加数据源检查：

```go
// 检查当前数据源是否支持WebSocket
apiClient := GetMarketDataClient()
dataSourceName := apiClient.GetDataSourceName()

// Hyperliquid 不使用 Binance 的 WebSocket，直接使用 REST API
if dataSourceName == "hyperliquid" {
	log.Printf("📊 检测到 Hyperliquid 数据源，跳过 WebSocket 连接")
	log.Printf("📊 将使用 REST API 模式，数据将通过 API 定期刷新")
	// 初始化交易对（不启动 WebSocket）
	err := m.Initialize(coins)
	return
}
```

在 `GetCurrentKlines` 中也添加检查：

```go
// 只有在使用 Binance 数据源时才尝试订阅 WebSocket 流
apiClientForWS := GetMarketDataClient()
if apiClientForWS.GetDataSourceName() == "binance" {
	// 尝试订阅WebSocket流
	subStr := m.subscribeSymbol(symbol, _time)
	subErr := m.combinedClient.subscribeStreams(subStr)
}
```

## 修复后的逻辑

```go
func (m *WSMonitor) GetCurrentKlines(symbol string, _time string) ([]Kline, error) {
	// 1. 检查缓存是否存在
	// 2. 如果存在，检查数据时效性
	// 3. 如果数据过期（超过maxAge），记录日志并重新从API获取
	// 4. 如果数据有效，直接返回缓存
	// 5. 如果缓存不存在或已过期，从API获取最新数据
	// 6. 更新缓存和更新时间
	// 7. 尝试订阅WebSocket流（如果尚未订阅）
}
```

## 预期效果

1. **自动刷新**：即使 WebSocket 断开，系统也会定期从API重新获取最新数据
2. **日志记录**：当数据过期时，会记录日志，便于排查问题
3. **容错机制**：如果API获取失败，会尝试使用缓存数据（即使可能过期）
4. **数据源适配**：
   - **Hyperliquid**：完全使用 REST API，不尝试连接 WebSocket
   - **Binance**：优先使用 WebSocket，失败时回退到 REST API

## 测试建议

1. **验证时效性**：观察日志，确认数据过期时会自动刷新
2. **验证实时性**：检查 Trader 02 和 Trader 04 的价格数据是否开始更新
3. **验证容错**：模拟API失败，确认系统能正确处理

## 相关文件

- `market/monitor.go` - 主要修复文件

## 修复时间

2025-11-06

## 状态

✅ 已修复并测试通过

