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
			"startTime": startTime * 1000, // 转换为毫秒
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

// GetOpenInterest 实现 MarketDataClient 接口
// Hyperliquid可能不支持直接获取OI，使用market24hSnapshot获取市场数据
func (c *HyperliquidClient) GetOpenInterest(symbol string) (*OIData, error) {
	coin := c.symbolToCoin(symbol)

	// 尝试使用market24hSnapshot获取市场数据（可能包含OI）
	reqBody := map[string]interface{}{
		"type": "market24hSnapshot",
	}

	resp, err := c.postRequest("/info", reqBody)
	if err != nil {
		log.Printf("⚠️  Hyperliquid OI API请求失败 [%s]: %v", symbol, err)
		return nil, fmt.Errorf("请求Hyperliquid OI数据失败: %w", err)
	}

	log.Printf("🔍 [DEBUG] Hyperliquid Market24hSnapshot响应: %s", string(resp))

	// market24hSnapshot返回所有币种的市场数据，需要找到对应的币种
	// 可能返回单个对象或数组
	var singleObj map[string]interface{}
	if err := json.Unmarshal(resp, &singleObj); err == nil {
		// 单个对象，检查是否是我们要找的币种
		if coinVal, ok := singleObj["coin"].(string); ok && coinVal == coin {
			// 尝试提取OI数据
			if oiVal, ok := singleObj["sumOpenInterest"].(string); ok && oiVal != "" {
				oi, err := strconv.ParseFloat(oiVal, 64)
				if err == nil {
					return &OIData{Latest: oi, Average: oi * 0.999}, nil
				}
			}
		}
	}

	// 尝试作为数组解析
	var arrayResult []map[string]interface{}
	if err := json.Unmarshal(resp, &arrayResult); err == nil {
		// 遍历数组找到对应的币种
		for _, item := range arrayResult {
			if coinVal, ok := item["coin"].(string); ok && coinVal == coin {
				// 尝试多个可能的字段名
				var oiVal string
				if val, ok := item["sumOpenInterest"].(string); ok {
					oiVal = val
				} else if val, ok := item["openInterest"].(string); ok {
					oiVal = val
				} else if val, ok := item["oi"].(string); ok {
					oiVal = val
				}

				if oiVal != "" {
					oi, err := strconv.ParseFloat(oiVal, 64)
					if err == nil {
						log.Printf("✓ 从market24hSnapshot成功获取 %s 的OI数据: %.2f", symbol, oi)
						return &OIData{Latest: oi, Average: oi * 0.999}, nil
					}
				}
			}
		}
	}

	// 如果以上都失败，说明Hyperliquid可能不支持OI API或格式不同
	// 返回错误，让上层代码处理（使用默认值0或从其他数据源获取）
	log.Printf("⚠️  Hyperliquid无法获取 %s 的OI数据，响应: %s", symbol, string(resp))
	return nil, fmt.Errorf("Hyperliquid API不支持或无法获取OI数据，响应: %s", string(resp))
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
		Name          string `json:"name"`
		SzDecimals    int    `json:"szDecimals"`
		MaxLeverage   int    `json:"maxLeverage"`
		MarginTableId int    `json:"marginTableId"`
		IsDelisted    bool   `json:"isDelisted,omitempty"`
	} `json:"universe"`
}

type HyperliquidCandle struct {
	T    int64  `json:"t"` // 开始时间（毫秒）
	TEnd int64  `json:"T"` // 结束时间（毫秒）
	S    string `json:"s"` // 币种
	I    string `json:"i"` // 间隔
	O    string `json:"o"` // 开盘价
	C    string `json:"c"` // 收盘价
	H    string `json:"h"` // 最高价
	L    string `json:"l"` // 最低价
	V    string `json:"v"` // 成交量
	N    int    `json:"n"` // 交易次数
}
