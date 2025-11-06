package market

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type WSMonitor struct {
	wsClient       *WSClient
	combinedClient *CombinedStreamsClient
	symbols        []string
	featuresMap    sync.Map
	alertsChan     chan Alert
	klineDataMap3m sync.Map // 存储每个交易对的K线历史数据
	klineDataMap4h sync.Map // 存储每个交易对的K线历史数据
	klineUpdateTime3m sync.Map // 存储3分钟K线数据的最后更新时间 (symbol -> time.Time)
	klineUpdateTime4h sync.Map // 存储4小时K线数据的最后更新时间 (symbol -> time.Time)
	tickerDataMap  sync.Map // 存储每个交易对的ticker数据
	batchSize      int
	filterSymbols  sync.Map // 使用sync.Map来存储需要监控的币种和其状态
	symbolStats    sync.Map // 存储币种统计信息
	FilterSymbol   []string //经过筛选的币种
}
type SymbolStats struct {
	LastActiveTime   time.Time
	AlertCount       int
	VolumeSpikeCount int
	LastAlertTime    time.Time
	Score            float64 // 综合评分
}

var WSMonitorCli *WSMonitor
var subKlineTime = []string{"3m", "4h"} // 管理订阅流的K线周期

func NewWSMonitor(batchSize int) *WSMonitor {
	WSMonitorCli = &WSMonitor{
		wsClient:       NewWSClient(),
		combinedClient: NewCombinedStreamsClient(batchSize),
		alertsChan:     make(chan Alert, 1000),
		batchSize:      batchSize,
	}
	return WSMonitorCli
}

func (m *WSMonitor) Initialize(coins []string) error {
	log.Println("初始化WebSocket监控器...")
	// 获取交易对信息
	apiClient := GetMarketDataClient()
	// 如果不指定交易对，则使用market市场的所有交易对币种
	if len(coins) == 0 {
		exchangeInfo, err := apiClient.GetExchangeInfo()
		if err != nil {
			return err
		}
		// 筛选永续合约交易对 --仅测试时使用
		//exchangeInfo.Symbols = exchangeInfo.Symbols[0:2]
		for _, symbol := range exchangeInfo.Symbols {
			if symbol.Status == "TRADING" && symbol.ContractType == "PERPETUAL" && strings.ToUpper(symbol.Symbol[len(symbol.Symbol)-4:]) == "USDT" {
				m.symbols = append(m.symbols, symbol.Symbol)
				m.filterSymbols.Store(symbol.Symbol, true)
			}
		}
	} else {
		m.symbols = coins
	}

	log.Printf("找到 %d 个交易对", len(m.symbols))
	// 初始化历史数据
	if err := m.initializeHistoricalData(); err != nil {
		log.Printf("初始化历史数据失败: %v", err)
	}

	return nil
}

func (m *WSMonitor) initializeHistoricalData() error {
	apiClient := GetMarketDataClient()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // 限制并发数

	for _, symbol := range m.symbols {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(s string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			// 获取历史K线数据
			klines, err := apiClient.GetKlines(s, "3m", 100)
			if err != nil {
				log.Printf("获取 %s 历史数据失败: %v", s, err)
				return
			}
			if len(klines) > 0 {
				m.klineDataMap3m.Store(s, klines)
				m.klineUpdateTime3m.Store(s, time.Now())
				log.Printf("已加载 %s 的历史K线数据-3m: %d 条", s, len(klines))
			}
			// 获取历史K线数据
			klines4h, err := apiClient.GetKlines(s, "4h", 100)
			if err != nil {
				log.Printf("获取 %s 历史数据失败: %v", s, err)
				return
			}
			if len(klines4h) > 0 {
				m.klineDataMap4h.Store(s, klines4h)
				m.klineUpdateTime4h.Store(s, time.Now())
				log.Printf("已加载 %s 的历史K线数据-4h: %d 条", s, len(klines4h))
			}
		}(symbol)
	}

	wg.Wait()
	return nil
}

func (m *WSMonitor) Start(coins []string) {
	go m.startWithRetry(coins)
}

func (m *WSMonitor) startWithRetry(coins []string) {
	// 检查当前数据源是否支持WebSocket
	apiClient := GetMarketDataClient()
	dataSourceName := apiClient.GetDataSourceName()
	
	// Hyperliquid 不使用 Binance 的 WebSocket，直接使用 REST API
	if dataSourceName == "hyperliquid" {
		log.Printf("📊 检测到 Hyperliquid 数据源，跳过 WebSocket 连接（Hyperliquid 不使用 Binance WebSocket）")
		log.Printf("📊 将使用 REST API 模式，数据将通过 API 定期刷新")
		// 初始化交易对（不启动 WebSocket）
		err := m.Initialize(coins)
		if err != nil {
			log.Printf("❌ 初始化币种失败: %v", err)
		}
		return
	}
	
	log.Printf("启动WebSocket实时监控...")
	// 初始化交易对
	err := m.Initialize(coins)
	if err != nil {
		log.Printf("❌ 初始化币种失败: %v (将使用REST API回退)", err)
		return
	}

	// 重试逻辑：最多重试10次，每次间隔30秒
	maxRetries := 10
	retryInterval := 30 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = m.combinedClient.Connect()
		if err != nil {
			log.Printf("⚠️  批量订阅流连接失败 (尝试 %d/%d): %v", attempt, maxRetries, err)
			if attempt < maxRetries {
				log.Printf("   将在 %.0f 秒后重试...", retryInterval.Seconds())
				time.Sleep(retryInterval)
				continue
			} else {
				log.Printf("❌ WebSocket连接彻底失败，将使用REST API回退模式")
				return
			}
		}

		// 订阅所有交易对
		err = m.subscribeAll()
		if err != nil {
			log.Printf("⚠️  订阅币种交易对失败 (尝试 %d/%d): %v", attempt, maxRetries, err)
			if attempt < maxRetries {
				log.Printf("   将在 %.0f 秒后重试...", retryInterval.Seconds())
				time.Sleep(retryInterval)
				continue
			} else {
				log.Printf("❌ WebSocket订阅彻底失败，将使用REST API回退模式")
				return
			}
		}

		// 成功连接和订阅
		log.Printf("✅ WebSocket实时监控启动成功")
		return
	}
}

// subscribeSymbol 注册监听
func (m *WSMonitor) subscribeSymbol(symbol, st string) []string {
	var streams []string
	stream := fmt.Sprintf("%s@kline_%s", strings.ToLower(symbol), st)
	ch := m.combinedClient.AddSubscriber(stream, 100)
	streams = append(streams, stream)
	go m.handleKlineData(symbol, ch, st)

	return streams
}
func (m *WSMonitor) subscribeAll() error {
	// 执行批量订阅
	log.Println("开始订阅所有交易对...")
	for _, symbol := range m.symbols {
		for _, st := range subKlineTime {
			m.subscribeSymbol(symbol, st)
		}
	}
	for _, st := range subKlineTime {
		err := m.combinedClient.BatchSubscribeKlines(m.symbols, st)
		if err != nil {
			log.Printf("❌ 订阅%s K线失败: %v", st, err)
			return err
		}
	}
	log.Println("所有交易对订阅完成")
	return nil
}

func (m *WSMonitor) handleKlineData(symbol string, ch <-chan []byte, _time string) {
	for data := range ch {
		var klineData KlineWSData
		if err := json.Unmarshal(data, &klineData); err != nil {
			log.Printf("解析Kline数据失败: %v", err)
			continue
		}
		m.processKlineUpdate(symbol, klineData, _time)
	}
}

func (m *WSMonitor) getKlineDataMap(_time string) *sync.Map {
	var klineDataMap *sync.Map
	if _time == "3m" {
		klineDataMap = &m.klineDataMap3m
	} else if _time == "4h" {
		klineDataMap = &m.klineDataMap4h
	} else {
		klineDataMap = &sync.Map{}
	}
	return klineDataMap
}

func (m *WSMonitor) getKlineUpdateTimeMap(_time string) *sync.Map {
	var updateTimeMap *sync.Map
	if _time == "3m" {
		updateTimeMap = &m.klineUpdateTime3m
	} else if _time == "4h" {
		updateTimeMap = &m.klineUpdateTime4h
	} else {
		updateTimeMap = &sync.Map{}
	}
	return updateTimeMap
}
func (m *WSMonitor) processKlineUpdate(symbol string, wsData KlineWSData, _time string) {
	// 转换WebSocket数据为Kline结构
	kline := Kline{
		OpenTime:  wsData.Kline.StartTime,
		CloseTime: wsData.Kline.CloseTime,
		Trades:    wsData.Kline.NumberOfTrades,
	}
	kline.Open, _ = parseFloat(wsData.Kline.OpenPrice)
	kline.High, _ = parseFloat(wsData.Kline.HighPrice)
	kline.Low, _ = parseFloat(wsData.Kline.LowPrice)
	kline.Close, _ = parseFloat(wsData.Kline.ClosePrice)
	kline.Volume, _ = parseFloat(wsData.Kline.Volume)
	kline.High, _ = parseFloat(wsData.Kline.HighPrice)
	kline.QuoteVolume, _ = parseFloat(wsData.Kline.QuoteVolume)
	kline.TakerBuyBaseVolume, _ = parseFloat(wsData.Kline.TakerBuyBaseVolume)
	kline.TakerBuyQuoteVolume, _ = parseFloat(wsData.Kline.TakerBuyQuoteVolume)
	// 更新K线数据
	var klineDataMap = m.getKlineDataMap(_time)
	value, exists := klineDataMap.Load(symbol)
	var klines []Kline
	if exists {
		klines = value.([]Kline)

		// 检查是否是新的K线
		if len(klines) > 0 && klines[len(klines)-1].OpenTime == kline.OpenTime {
			// 更新当前K线
			klines[len(klines)-1] = kline
		} else {
			// 添加新K线
			klines = append(klines, kline)

			// 保持数据长度
			if len(klines) > 100 {
				klines = klines[1:]
			}
		}
	} else {
		klines = []Kline{kline}
	}

	klineDataMap.Store(symbol, klines)
	
	// 更新该symbol的最后更新时间
	updateTimeMap := m.getKlineUpdateTimeMap(_time)
	updateTimeMap.Store(symbol, time.Now())
}

func (m *WSMonitor) GetCurrentKlines(symbol string, _time string) ([]Kline, error) {
	symbol = strings.ToUpper(symbol)
	klineDataMap := m.getKlineDataMap(_time)
	updateTimeMap := m.getKlineUpdateTimeMap(_time)
	
	// 检查缓存是否存在
	value, exists := klineDataMap.Load(symbol)
	if exists {
		// 检查缓存数据的时效性
		lastUpdate, updateExists := updateTimeMap.Load(symbol)
		
		// 如果存在更新时间，检查数据是否过期
		// 对于3分钟K线，如果超过3分钟未更新，则重新获取（确保每个K线周期都会刷新）
		// 对于4小时K线，如果超过10分钟未更新，则重新获取
		var maxAge time.Duration
		if _time == "3m" {
			maxAge = 3 * time.Minute
		} else if _time == "4h" {
			maxAge = 10 * time.Minute
		} else {
			maxAge = 5 * time.Minute // 默认5分钟
		}
		
		if updateExists {
			lastUpdateTime := lastUpdate.(time.Time)
			if time.Since(lastUpdateTime) < maxAge {
				// 缓存数据仍然有效，直接返回
				return value.([]Kline), nil
			} else {
				// 缓存数据过期，记录日志并重新获取
				log.Printf("⚠️  %s 的 %s K线数据已过期（最后更新: %v，已过期 %v），重新从API获取", symbol, _time, lastUpdateTime, time.Since(lastUpdateTime))
			}
		} else {
			// 没有更新时间记录，可能是旧数据，重新获取
			log.Printf("⚠️  %s 的 %s K线数据缺少更新时间记录，重新从API获取", symbol, _time)
		}
	}
	
	// 缓存不存在或已过期，从API获取最新数据
	apiClient := GetMarketDataClient()
	klines, err := apiClient.GetKlines(symbol, _time, 100)
	if err != nil {
		// 如果API获取失败，尝试返回缓存数据（即使可能过期）
		if exists {
			log.Printf("⚠️  从API获取 %s 的 %s K线数据失败: %v，使用可能过期的缓存数据", symbol, _time, err)
			return value.([]Kline), nil
		}
		return nil, fmt.Errorf("获取%v分钟K线失败: %v", _time, err)
	}
	
	// 更新缓存
	klineDataMap.Store(symbol, klines)
	updateTimeMap.Store(symbol, time.Now())
	log.Printf("✓ 成功从API获取并更新 %s 的 %s K线数据（%d 条）", symbol, _time, len(klines))
	
	// 只有在使用 Binance 数据源时才尝试订阅 WebSocket 流
	apiClientForWS := GetMarketDataClient()
	if apiClientForWS.GetDataSourceName() == "binance" {
		// 尝试订阅WebSocket流（如果尚未订阅）
		subStr := m.subscribeSymbol(symbol, _time)
		subErr := m.combinedClient.subscribeStreams(subStr)
		if subErr != nil {
			// 订阅失败不影响数据获取，只记录警告
			log.Printf("⚠️  动态订阅%v分钟K线流失败: %v (将继续使用API数据)", _time, subErr)
		}
	}
	
	return klines, nil
}

func (m *WSMonitor) Close() {
	m.wsClient.Close()
	close(m.alertsChan)
}
