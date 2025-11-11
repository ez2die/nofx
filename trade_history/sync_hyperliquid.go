package trade_history

import (
	"context"
	"fmt"
	"log"
	"strconv"
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

	// 设置方向（Dir字段）
	if fill.Dir != "" {
		exchangeFill.Dir = fill.Dir
	} else {
		return exchangeFill, fmt.Errorf("Fill.Dir为空")
	}

	// 设置Side（根据StartPosition判断：正数为long，负数为short）
	// Hyperliquid的Side字段是"A"或"B"（订单类型），不是仓位方向
	// 我们需要根据StartPosition来判断是long还是short
	if fill.StartPosition != "" {
		startPos, err := strconv.ParseFloat(fill.StartPosition, 64)
		if err == nil {
			if startPos > 0 {
				exchangeFill.Side = "long"
			} else if startPos < 0 {
				exchangeFill.Side = "short"
			} else {
				// 如果StartPosition为0，可能是开仓，根据Dir推断
				if fill.Dir == "Open" {
					// 开仓时，如果数量为正，通常是long；如果为负，通常是short
					if fill.Size != "" {
						sz, err := strconv.ParseFloat(fill.Size, 64)
						if err == nil && sz > 0 {
							exchangeFill.Side = "long"
						} else if err == nil && sz < 0 {
							exchangeFill.Side = "short"
						}
					}
				} else {
					// 平仓时，根据平仓前的持仓方向推断
					// 如果ClosedPnl为正，可能是做多盈利；如果为负，可能是做空盈利
					// 但这不够准确，最好保留原Side字段信息用于调试
					exchangeFill.Side = "" // 需要根据历史持仓推断
				}
			}
		}
	}
	
	// 如果仍然无法确定Side，尝试从StartPosition推断
	if exchangeFill.Side == "" && fill.StartPosition != "" {
		startPos, err := strconv.ParseFloat(fill.StartPosition, 64)
		if err == nil && startPos != 0 {
			// 如果持仓为正（long），平仓时价格上升盈利，价格下降亏损
			// 如果持仓为负（short），平仓时价格下降盈利，价格上升亏损
			if startPos > 0 {
				exchangeFill.Side = "long"
			} else {
				exchangeFill.Side = "short"
			}
		}
	}

	// 设置数量（Size字段，json:"sz"）
	if fill.Size != "" {
		sz, err := strconv.ParseFloat(fill.Size, 64)
		if err != nil {
			return exchangeFill, fmt.Errorf("解析数量失败: %w", err)
		}
		exchangeFill.Quantity = sz
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

	// 设置持仓（StartPosition字段）
	if fill.StartPosition != "" {
		startPos, err := strconv.ParseFloat(fill.StartPosition, 64)
		if err == nil {
			exchangeFill.StartPosition = &startPos
		}
	}

	return exchangeFill, nil
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
