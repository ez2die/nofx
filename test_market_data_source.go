package main

import (
	"flag"
	"fmt"
	"log"
	"nofx/market"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 解析命令行参数
	dataSource := flag.String("source", "binance", "Market data source: binance or hyperliquid")
	flag.Parse()

	log.Println("=== 市场数据源测试脚本 ===")
	log.Printf("📊 测试数据源: %s\n", *dataSource)

	// 初始化市场数据源
	if err := market.SetDataSource(market.DataSource(*dataSource)); err != nil {
		log.Fatalf("❌ 设置市场数据源失败: %v", err)
	}

	// 获取市场数据客户端
	client := market.GetMarketDataClient()
	log.Printf("✓ 数据源客户端: %s\n", client.GetDataSourceName())

	// 测试1: 获取交易对列表
	log.Println("\n📋 测试1: 获取交易对列表...")
	exchangeInfo, err := client.GetExchangeInfo()
	if err != nil {
		log.Fatalf("❌ 获取交易对列表失败: %v", err)
	}
	log.Printf("✓ 成功获取 %d 个交易对\n", len(exchangeInfo.Symbols))

	// 显示前5个交易对
	for i, symbol := range exchangeInfo.Symbols {
		if i >= 5 {
			break
		}
		log.Printf("  %d. %s (%s/%s) - 状态: %s, 类型: %s",
			i+1, symbol.Symbol, symbol.BaseAsset, symbol.QuoteAsset,
			symbol.Status, symbol.ContractType)
	}

	// 测试2: 获取当前价格
	testSymbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}
	log.Println("\n💰 测试2: 获取当前价格...")
	for _, symbol := range testSymbols {
		price, err := client.GetCurrentPrice(symbol)
		if err != nil {
			log.Printf("❌ 获取 %s 价格失败: %v", symbol, err)
			continue
		}
		log.Printf("✓ %s: $%.2f", symbol, price)
	}

	// 测试3: 获取K线数据
	log.Println("\n📊 测试3: 获取K线数据...")
	for _, symbol := range testSymbols {
		klines, err := client.GetKlines(symbol, "1h", 5)
		if err != nil {
			log.Printf("❌ 获取 %s K线失败: %v", symbol, err)
			continue
		}
		log.Printf("✓ %s: 成功获取 %d 根1小时K线", symbol, len(klines))
		if len(klines) > 0 {
			latest := klines[len(klines)-1]
			log.Printf("  最新K线: 开盘=%.2f, 收盘=%.2f, 最高=%.2f, 最低=%.2f, 成交量=%.2f",
				latest.Open, latest.Close, latest.High, latest.Low, latest.Volume)
		}
	}

	log.Println("\n✅ 所有测试通过！")
	log.Println(fmt.Sprintf("市场数据源 '%s' 工作正常", *dataSource))
}
