package trade_history

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"time"
)

// Service 交易历史服务接口
type Service interface {
	// GetTradeHistory 获取交易历史
	GetTradeHistory(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, int, error)

	// GetTradeStatistics 获取交易统计
	GetTradeStatistics(ctx context.Context, traderID string, startTime, endTime *time.Time) (*TradeStatistics, error)

	// GetTradePerformance 获取交易表现
	GetTradePerformance(ctx context.Context, traderID string, options *PerformanceOptions) (*TradePerformance, error)

	// SyncFromExchange 从交易所同步交易记录
	SyncFromExchange(ctx context.Context, traderID string, provider ExchangeFillsProvider) error
}

// service 实现
type service struct {
	repo Repository
}

const (
	DefaultRecentTradesLimit = 20
	DefaultSharpeWindow      = 30
	DefaultLookbackDays      = 30
)

// NewService 创建Service实例
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// GetTradeHistory 获取交易历史
func (s *service) GetTradeHistory(ctx context.Context, filter *TradeRecordFilter) ([]*TradeRecord, int, error) {
	// 查询记录
	records, err := s.repo.FindByFilter(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("查询交易历史失败: %w", err)
	}

	// 查询总数
	total, err := s.repo.CountByFilter(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("统计交易历史失败: %w", err)
	}

	return records, total, nil
}

// GetTradeStatistics 获取交易统计
func (s *service) GetTradeStatistics(ctx context.Context, traderID string, startTime, endTime *time.Time) (*TradeStatistics, error) {
	stats, err := s.repo.GetStatistics(ctx, traderID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("获取交易统计失败: %w", err)
	}

	return stats, nil
}

// GetTradePerformance 获取交易表现
func (s *service) GetTradePerformance(ctx context.Context, traderID string, options *PerformanceOptions) (*TradePerformance, error) {
	if traderID == "" {
		return nil, errors.New("traderID 不能为空")
	}

	opts := s.applyDefaultOptions(options)

	stats, err := s.repo.GetStatistics(ctx, traderID, opts.StartTime, opts.EndTime)
	if err != nil {
		return nil, fmt.Errorf("获取交易统计失败: %w", err)
	}

	perf := &TradePerformance{
		TradeStatistics: *stats,
		RecentTrades:    []*TradeOutcome{},
		SymbolStats:     make(map[string]*SymbolPerformance),
		GeneratedAt:     time.Now(),
	}

	if stats.TotalTrades == 0 {
		return perf, nil
	}

	recent, err := s.repo.GetRecentTrades(ctx, traderID, opts.RecentLimit, opts.StartTime, opts.EndTime)
	if err != nil {
		return nil, fmt.Errorf("获取最近交易失败: %w", err)
	}
	perf.RecentTrades = recent

	symbolStats, err := s.repo.GetSymbolStats(ctx, traderID, opts.StartTime, opts.EndTime)
	if err != nil {
		return nil, fmt.Errorf("获取币种表现失败: %w", err)
	}
	perf.SymbolStats = s.buildSymbolStatMap(symbolStats)
	perf.BestSymbol, perf.WorstSymbol = pickBestAndWorst(perf.SymbolStats)

	pnlSeries, err := s.repo.GetPnLSeries(ctx, traderID, opts.StartTime, opts.EndTime, opts.SharpeWindow)
	if err != nil {
		return nil, fmt.Errorf("获取PnL序列失败: %w", err)
	}
	perf.SharpeRatio = calculateSharpeRatio(pnlSeries)

	return perf, nil
}

// SyncFromExchange 从交易所同步交易记录
func (s *service) SyncFromExchange(ctx context.Context, traderID string, provider ExchangeFillsProvider) error {
	const recentLimit = 200
	const cursorSkew = 2 * time.Minute

	lastCursor, err := s.repo.GetSyncState(ctx, traderID)
	if err != nil {
		return err
	}

	var fills []ExchangeFill
	now := time.Now()

	if lastCursor > 0 {
		start := time.UnixMilli(lastCursor + 1)
		end := now.Add(cursorSkew)
		fills, err = provider.GetFillsByTimeRange(start, end)
		if err != nil {
			log.Printf("⚠️ 按时间范围获取成交记录失败（trader_id=%s）：%v，尝试回退到最近记录", traderID, err)
			fills = nil
		}
	}

	if len(fills) == 0 {
		fills, err = provider.GetRecentFills(recentLimit)
		if err != nil {
			return fmt.Errorf("获取交易所成交记录失败: %w", err)
		}
	}

	if len(fills) == 0 {
		log.Printf("ℹ️ 未找到新的成交记录（trader_id=%s）", traderID)
		return nil
	}

	// 转换为TradeRecord并批量保存
	var records []*TradeRecord
	var maxTimestampMs int64

	for _, fill := range fills {
		candidateTimestampMs := fill.TimestampMs

		if candidateTimestampMs > 0 && candidateTimestampMs <= lastCursor {
			// 已同步过的数据跳过
			continue
		}

		// 转换ExchangeFill到TradeRecord
		record := s.convertFillToRecord(traderID, &fill)

		if candidateTimestampMs == 0 && record.ExchangeTimestampMs != nil {
			candidateTimestampMs = *record.ExchangeTimestampMs
		}
		if candidateTimestampMs == 0 && !record.Timestamp.IsZero() {
			candidateTimestampMs = record.Timestamp.UnixMilli()
		}

		// 检查是否已存在
		var exchangeOid, exchangeTid *int64
		var exchangeHash *string

		if fill.ExchangeOid > 0 {
			exchangeOid = &fill.ExchangeOid
		}
		if fill.ExchangeTid > 0 {
			exchangeTid = &fill.ExchangeTid
		}
		if fill.ExchangeHash != "" {
			exchangeHash = &fill.ExchangeHash
		}

		exists, err := s.repo.ExistsByExchangeID(ctx, exchangeOid, exchangeTid, exchangeHash)
		if err != nil {
			log.Printf("⚠️ 警告：检查交易记录是否存在失败: %v", err)
			continue
		}
		if exists {
			// 已存在，跳过
			if candidateTimestampMs > maxTimestampMs {
				maxTimestampMs = candidateTimestampMs
			}
			continue
		}

		records = append(records, record)

		if candidateTimestampMs > maxTimestampMs {
			maxTimestampMs = candidateTimestampMs
		}
	}

	if len(records) == 0 {
		log.Printf("ℹ️ 所有成交记录已存在（trader_id=%s）", traderID)
		if maxTimestampMs > 0 {
			if err := s.repo.UpsertSyncState(ctx, traderID, maxTimestampMs); err != nil {
				return err
			}
		}
		return nil
	}

	// 批量保存
	if err := s.repo.SaveBatch(ctx, records); err != nil {
		return fmt.Errorf("批量保存交易记录失败: %w", err)
	}

	log.Printf("✅ 同步完成：保存了 %d 条新交易记录（trader_id=%s）", len(records), traderID)

	if maxTimestampMs > 0 {
		if err := s.repo.UpsertSyncState(ctx, traderID, maxTimestampMs); err != nil {
			return err
		}
	}

	return nil
}

// convertFillToRecord 将ExchangeFill转换为TradeRecord
func (s *service) convertFillToRecord(traderID string, fill *ExchangeFill) *TradeRecord {
	record := &TradeRecord{
		TraderID:       traderID,
		Symbol:         fill.Symbol,
		Side:           fill.Side,
		Quantity:       fill.Quantity,
		ExecutionPrice: fill.Price,
		Fee:            fill.Fee,
		Timestamp:      fill.Timestamp,
		PnL:            fill.ClosedPnl,
		SignedQuantity: fill.SignedQty,
		StartPosition:  fill.StartPosition,
		BuilderFee:     fill.BuilderFee,
		RawDir:         fill.RawDir,
	}

	if fill.FeeToken != "" {
		feeToken := fill.FeeToken
		record.FeeToken = &feeToken
	}

	if fill.ExchangeSide != "" {
		exchangeSide := fill.ExchangeSide
		record.ExchangeSide = &exchangeSide
	}

	// 设置Action（确保Side是"long"或"short"）
	// 如果Side为空或不是预期的值，尝试根据其他信息推断
	side := fill.Side
	if side != "long" && side != "short" {
		// 如果Side不是标准值，根据StartPosition推断
		if fill.StartPosition != nil {
			if *fill.StartPosition > 0 {
				side = "long"
			} else if *fill.StartPosition < 0 {
				side = "short"
			}
		}
		// 如果仍然无法确定，使用默认值（但应该避免这种情况）
		if side != "long" && side != "short" {
			log.Printf("⚠️ 警告：无法确定Side，使用默认值long (trader_id=%s, symbol=%s, dir=%s)", traderID, fill.Symbol, fill.Dir)
			side = "long" // 默认值
		}
	}
	record.Side = side

	// 设置Action
	dir := fill.Dir
	rawDir := fill.RawDir

	if normalized, raw := normalizeExchangeDir(fill.Dir); normalized != "" {
		dir = normalized
		if rawDir == "" {
			rawDir = raw
		}
	} else if rawDir == "" {
		rawDir = fill.Dir
	}

	switch dir {
	case "Open":
		if side == "long" {
			record.Action = "open_long"
		} else {
			record.Action = "open_short"
		}
	case "Close":
		if side == "long" {
			record.Action = "close_long"
		} else {
			record.Action = "close_short"
		}
	default:
		log.Printf("⚠️ 无法识别的成交方向 dir=%q (trader_id=%s, symbol=%s)，默认视为平仓", rawDir, traderID, fill.Symbol)
		if side == "long" {
			record.Action = "close_long"
		} else {
			record.Action = "close_short"
		}
	}
	record.RawDir = rawDir

	// 设置交易所时间戳
	if fill.TimestampMs > 0 {
		record.ExchangeTimestampMs = &fill.TimestampMs
		exchangeTime := time.Unix(fill.TimestampMs/1000, (fill.TimestampMs%1000)*1000000)
		record.ExchangeTimestamp = &exchangeTime
		record.Timestamp = exchangeTime
	}

	// 设置交易所订单信息
	if fill.ExchangeOid > 0 {
		oidStr := fmt.Sprintf("%d", fill.ExchangeOid)
		record.ExchangeOrderID = &oidStr
	}
	if fill.ExchangeTid > 0 {
		tidStr := fmt.Sprintf("%d", fill.ExchangeTid)
		record.ExchangeTradeID = &tidStr
	}
	if fill.ExchangeHash != "" {
		record.ExchangeHash = &fill.ExchangeHash
	}

	return record
}

func (s *service) applyDefaultOptions(options *PerformanceOptions) *PerformanceOptions {
	opts := &PerformanceOptions{}
	if options != nil {
		*opts = *options
	}

	now := time.Now()
	if opts.EndTime == nil {
		end := now
		opts.EndTime = &end
	}
	if opts.StartTime == nil {
		start := opts.EndTime.Add(-DefaultLookbackDays * 24 * time.Hour)
		opts.StartTime = &start
	}
	if opts.RecentLimit <= 0 {
		opts.RecentLimit = DefaultRecentTradesLimit
	}
	if opts.SharpeWindow <= 0 {
		opts.SharpeWindow = DefaultSharpeWindow
	}

	return opts
}

func (s *service) buildSymbolStatMap(stats []*SymbolPerformance) map[string]*SymbolPerformance {
	result := make(map[string]*SymbolPerformance, len(stats))
	for _, stat := range stats {
		if stat.TotalTrades > 0 {
			stat.WinRate = float64(stat.WinningTrades) / float64(stat.TotalTrades) * 100
		}
		result[stat.Symbol] = stat
	}
	return result
}

func pickBestAndWorst(stats map[string]*SymbolPerformance) (string, string) {
	var (
		bestSymbol, worstSymbol string
		bestStat, worstStat     *SymbolPerformance
	)

	for symbol, stat := range stats {
		if stat == nil {
			continue
		}
		if bestStat == nil || betterThan(stat, bestStat) {
			bestStat = stat
			bestSymbol = symbol
		}
		if worstStat == nil || worseThan(stat, worstStat) {
			worstStat = stat
			worstSymbol = symbol
		}
	}

	return bestSymbol, worstSymbol
}

func betterThan(a, b *SymbolPerformance) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	if a.TotalPnL != b.TotalPnL {
		return a.TotalPnL > b.TotalPnL
	}
	if a.WinRate != b.WinRate {
		return a.WinRate > b.WinRate
	}
	return a.TotalTrades > b.TotalTrades
}

func worseThan(a, b *SymbolPerformance) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	if a.TotalPnL != b.TotalPnL {
		return a.TotalPnL < b.TotalPnL
	}
	if a.WinRate != b.WinRate {
		return a.WinRate < b.WinRate
	}
	return a.TotalTrades > b.TotalTrades
}

func calculateSharpeRatio(points []*PnLPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	// reverse to chronological order (oldest first)
	sort.SliceStable(points, func(i, j int) bool {
		return points[i].Timestamp.Before(points[j].Timestamp)
	})

	var (
		mean      float64
		m2        float64
		n         int
		firstTime time.Time
		lastTime  time.Time
	)

	for idx, point := range points {
		if idx == 0 {
			firstTime = point.Timestamp
		}
		if idx == len(points)-1 {
			lastTime = point.Timestamp
		}

		returnVal := normalizeReturn(point)
		n++
		delta := returnVal - mean
		mean += delta / float64(n)
		m2 += delta * (returnVal - mean)
	}

	if n < 2 {
		return 0
	}

	variance := m2 / float64(n-1)
	if variance <= 0 {
		return 0
	}
	std := math.Sqrt(variance)
	if std == 0 {
		return 0
	}

	duration := lastTime.Sub(firstTime).Hours() / 24
	if duration < 1 {
		duration = 1
	}

	periodsPerYear := 365.0 / duration
	if periodsPerYear <= 0 {
		return 0
	}

	return (mean / std) * math.Sqrt(periodsPerYear)
}

func normalizeReturn(point *PnLPoint) float64 {
	if point == nil {
		return 0
	}

	var qty float64
	if point.SignedQuantity != nil {
		qty = math.Abs(*point.SignedQuantity)
	}
	if qty == 0 {
		qty = math.Abs(point.Quantity)
	}

	notional := qty * point.ExecutionPrice
	if notional <= 0 {
		return point.PnL
	}

	return point.PnL / notional
}
