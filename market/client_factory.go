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
				log.Printf("⚠️  Hyperliquid客户端初始化失败: %v", err)
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

