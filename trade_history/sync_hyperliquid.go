package trade_history

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

// HyperliquidFillsProvider Hyperliquid成交记录提供者
type HyperliquidFillsProvider struct {
	exchange   *hyperliquid.Exchange
	ctx        context.Context
	walletAddr string
}

// NewHyperliquidFillsProvider 创建Hyperliquid成交记录提供者
func NewHyperliquidFillsProvider(exchange *hyperliquid.Exchange, ctx context.Context, walletAddr string) *HyperliquidFillsProvider {
	return &HyperliquidFillsProvider{
		exchange:   exchange,
		ctx:        ctx,
		walletAddr: walletAddr,
	}
}

// convertSymbolToHyperliquid 将系统symbol转换为Hyperliquid coin名称
// "BTCUSDT" -> "BTC"
func convertSymbolToHyperliquid(symbol string) string {
	// 移除 "USDT" 后缀
	if len(symbol) > 4 && symbol[len(symbol)-4:] == "USDT" {
		return symbol[:len(symbol)-4]
	}
	return symbol
}

// convertHyperliquidFillToExchangeFill 将Hyperliquid Fill转换为ExchangeFill
func convertHyperliquidFillToExchangeFill(fill hyperliquid.Fill) (ExchangeFill, error) {
	exchangeFill := ExchangeFill{}

	// 设置币种（从Fill.Coin转换为symbol格式）
	if fill.Coin != "" {
		exchangeFill.Symbol = fill.Coin + "USDT"
	} else {
		return exchangeFill, fmt.Errorf("Fill.Coin为空")
	}

	// 设置方向（Dir字段），标准化为"Open"/"Close"
	if fill.Dir != "" {
		normalizedDir, rawDir := normalizeExchangeDir(fill.Dir)
		exchangeFill.RawDir = rawDir
		if normalizedDir == "" {
			log.Printf("⚠️ 未知的Hyperliquid dir值: %q，保留原始值", fill.Dir)
			exchangeFill.Dir = rawDir
		} else {
			exchangeFill.Dir = normalizedDir
		}
	} else {
		return exchangeFill, fmt.Errorf("Fill.Dir为空")
	}

	dirSideHint := inferSideFromDir(fill.Dir)
	if dirSideHint != "" {
		exchangeFill.Side = dirSideHint
	}

	// 设置Side（根据StartPosition判断：正数为long，负数为short）
	// Hyperliquid的Side字段是"A"或"B"（订单类型），不是仓位方向
	// 我们需要根据StartPosition来判断是long还是short
	var startPos float64
	var startPosValid bool
	if fill.StartPosition != "" {
		if sp, err := strconv.ParseFloat(fill.StartPosition, 64); err == nil {
			startPos = sp
			startPosValid = true
		}
	}

	// 设置数量（Size字段，json:"sz"）
	var rawSize float64
	var sizeValid bool
	if fill.Size != "" {
		sz, err := strconv.ParseFloat(fill.Size, 64)
		if err != nil {
			return exchangeFill, fmt.Errorf("解析数量失败: %w", err)
		}
		rawSize = sz
		sizeValid = true
		exchangeFill.Quantity = math.Abs(sz)
		signed := sz
		exchangeFill.SignedQty = &signed
	}

	// 当StartPosition不为0时，以其符号为准
	if startPosValid {
		if startPos > 0 {
			exchangeFill.Side = "long"
		} else if startPos < 0 {
			exchangeFill.Side = "short"
		}
		sp := startPos
		exchangeFill.StartPosition = &sp
	}

	// StartPosition为0时，优先使用dir中的方向提示
	if exchangeFill.Side == "" && dirSideHint != "" {
		exchangeFill.Side = dirSideHint
	}

	// 如果依然无法确定方向，尝试使用成交数量的正负号
	if exchangeFill.Side == "" && sizeValid {
		if rawSize < 0 {
			exchangeFill.Side = "short"
		} else if rawSize > 0 {
			exchangeFill.Side = "long"
		}
	}

	// 如果仍然无法判断，记录警告并默认long（避免空值）
	if exchangeFill.Side == "" {
		log.Printf("⚠️ 无法根据成交信息判断方向，默认long (dir=%q start_position=%q size=%q)", fill.Dir, fill.StartPosition, fill.Size)
		exchangeFill.Side = "long"
	}

	// 设置价格（Price字段，json:"px"）
	if fill.Price != "" {
		px, err := strconv.ParseFloat(fill.Price, 64)
		if err != nil {
			return exchangeFill, fmt.Errorf("解析价格失败: %w", err)
		}
		exchangeFill.Price = px
	}

	// 设置时间戳（Time字段）
	if fill.Time > 0 {
		exchangeFill.TimestampMs = fill.Time
		exchangeFill.Timestamp = time.Unix(fill.Time/1000, (fill.Time%1000)*1000000)
	} else {
		exchangeFill.Timestamp = time.Now()
		exchangeFill.TimestampMs = time.Now().UnixMilli()
	}

	// 设置订单ID（Oid字段）
	if fill.Oid > 0 {
		exchangeFill.ExchangeOid = fill.Oid
		exchangeFill.OrderID = fmt.Sprintf("%d", fill.Oid)
	}

	// 设置交易ID（Tid字段）
	if fill.Tid > 0 {
		exchangeFill.ExchangeTid = fill.Tid
	}

	// 设置交易哈希（Hash字段）
	if fill.Hash != "" {
		exchangeFill.ExchangeHash = fill.Hash
	}

	// 保存交易所原始Side（A/B，用于调试）
	if fill.Side != "" {
		exchangeFill.ExchangeSide = fill.Side
	}

	// 设置盈亏（ClosedPnl字段，仅平仓时有效）
	if fill.ClosedPnl != "" {
		closedPnl, err := strconv.ParseFloat(fill.ClosedPnl, 64)
		if err == nil {
			exchangeFill.ClosedPnl = &closedPnl
		}
	}

	// 设置手续费（Fee字段）
	if fill.Fee != "" {
		fee, err := strconv.ParseFloat(fill.Fee, 64)
		if err == nil {
			exchangeFill.Fee = fee
		}
	}

	// 设置手续费币种（FeeToken字段）
	if fill.FeeToken != "" {
		exchangeFill.FeeToken = fill.FeeToken
	}

	// 设置BuilderFee
	if fill.BuilderFee != "" {
		if builderFee, err := strconv.ParseFloat(fill.BuilderFee, 64); err == nil {
			exchangeFill.BuilderFee = &builderFee
		}
	}

	return exchangeFill, nil
}

// normalizeExchangeDir 将交易所返回的dir字符串标准化为"Open"或"Close"
// 部分交易所会返回"Open Long"、"Close Short"等字符串，或者大小写不一致
func normalizeExchangeDir(dir string) (normalized string, raw string) {
	raw = strings.TrimSpace(dir)
	if raw == "" {
		return "", ""
	}

	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "open"):
		return "Open", raw
	case strings.HasPrefix(lower, "close"):
		return "Close", raw
	case strings.HasPrefix(lower, "reduce"):
		// Hyperliquid可能返回"Reduce Long/Short"，本质属于平仓
		return "Close", raw
	case strings.HasPrefix(lower, "increase"):
		// 视为开仓
		return "Open", raw
	default:
		return "", raw
	}
}

// inferSideFromDir 根据原始的dir字符串推断仓位方向
func inferSideFromDir(dir string) string {
	lower := strings.ToLower(strings.TrimSpace(dir))
	switch {
	case strings.Contains(lower, "short"):
		return "short"
	case strings.Contains(lower, "long"):
		return "long"
	default:
		return ""
	}
}

// GetRecentFills 获取最近的成交记录
func (p *HyperliquidFillsProvider) GetRecentFills(limit int) ([]ExchangeFill, error) {
	// 调用 exchange.Info().UserFills()
	fills, err := p.exchange.Info().UserFills(p.ctx, p.walletAddr)
	if err != nil {
		return nil, fmt.Errorf("获取UserFills失败: %w", err)
	}

	// 转换为统一格式
	var exchangeFills []ExchangeFill
	count := 0

	// UserFills返回的是从旧到新的顺序，需要从后往前取最近的limit条
	startIdx := len(fills) - limit
	if startIdx < 0 {
		startIdx = 0
	}

	for i := len(fills) - 1; i >= startIdx && count < limit; i-- {
		fill := fills[i]
		exchangeFill, err := convertHyperliquidFillToExchangeFill(fill)
		if err != nil {
			log.Printf("⚠️ 转换Hyperliquid Fill失败: %v", err)
			continue
		}
		exchangeFills = append(exchangeFills, exchangeFill)
		count++
	}

	// 反转顺序，使最新的在前
	for i, j := 0, len(exchangeFills)-1; i < j; i, j = i+1, j-1 {
		exchangeFills[i], exchangeFills[j] = exchangeFills[j], exchangeFills[i]
	}

	return exchangeFills, nil
}

// GetFillsByTimeRange 按时间范围获取成交记录
func (p *HyperliquidFillsProvider) GetFillsByTimeRange(startTime, endTime time.Time) ([]ExchangeFill, error) {
	// 调用 exchange.Info().UserFills()
	fills, err := p.exchange.Info().UserFills(p.ctx, p.walletAddr)
	if err != nil {
		return nil, fmt.Errorf("获取UserFills失败: %w", err)
	}

	// 转换为统一格式并过滤时间范围
	var exchangeFills []ExchangeFill
	startTimeMs := startTime.UnixMilli()
	endTimeMs := endTime.UnixMilli()

	for _, fill := range fills {
		// 检查时间范围
		if fill.Time > 0 {
			if fill.Time < startTimeMs || fill.Time > endTimeMs {
				continue
			}
		}

		exchangeFill, err := convertHyperliquidFillToExchangeFill(fill)
		if err != nil {
			log.Printf("⚠️ 转换Hyperliquid Fill失败: %v", err)
			continue
		}
		exchangeFills = append(exchangeFills, exchangeFill)
	}

	return exchangeFills, nil
}
