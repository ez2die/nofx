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

	// GetOpenInterest 获取Open Interest数据（持仓量）
	// 返回持仓量和平均值，如果数据源不支持则返回错误
	GetOpenInterest(symbol string) (*OIData, error)

	// GetDataSourceName 返回数据源名称（用于日志和调试）
	GetDataSourceName() string
}
