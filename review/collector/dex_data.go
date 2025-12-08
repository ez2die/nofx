package collector

import (
	"context"
	"fmt"
	"log"
	"time"

	"nofx/review"
	"nofx/trade_history"
)

// DEXDataCollector DEX数据收集器
type DEXDataCollector struct {
	provider    review.DEXDataProvider
	maxRetries  int
	retryIntervals []time.Duration
}

// NewDEXDataCollector 创建DEX数据收集器
func NewDEXDataCollector(provider review.DEXDataProvider) *DEXDataCollector {
	return &DEXDataCollector{
		provider:    provider,
		maxRetries:  3,
		retryIntervals: []time.Duration{
			1 * time.Second,
			2 * time.Second,
			4 * time.Second,
		},
	}
}

// Collect 收集指定时间范围内的DEX交易数据
// 失败策略：DEX数据拉取失败时，直接报错停止本轮复盘（不降级）
func (c *DEXDataCollector) Collect(ctx context.Context, startTime, endTime time.Time) ([]trade_history.ExchangeFill, error) {
	log.Printf("📊 开始收集DEX数据 (时间范围: %s ~ %s)", startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))

	var lastErr error

	// 重试机制：指数退避
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			// 等待重试间隔
			waitTime := c.retryIntervals[attempt-1]
			if attempt-1 < len(c.retryIntervals) {
				log.Printf("⏳ DEX数据拉取失败，%v 后进行第 %d 次重试...", waitTime, attempt+1)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(waitTime):
				}
			}
		}

		// 调用 DEXDataProvider 的 GetFillsByTimeRange 方法
		fills, err := c.provider.GetFillsByTimeRange(startTime, endTime)
		if err == nil {
			log.Printf("✅ DEX数据收集完成，共 %d 条记录", len(fills))
			return fills, nil
		}

		lastErr = err
		log.Printf("⚠️ DEX数据拉取失败（尝试 %d/%d）: %v", attempt+1, c.maxRetries, err)

		// 检查是否是致命错误（不应该重试）
		if isFatalError(err) {
			return nil, fmt.Errorf("DEX数据拉取遇到致命错误: %w", err)
		}
	}

	// 所有重试都失败
	return nil, fmt.Errorf("DEX数据拉取失败（重试%d次）: %w", c.maxRetries, lastErr)
}

// isFatalError 判断是否是致命错误（不应该重试）
func isFatalError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 认证错误、权限错误等不应该重试
	fatalErrors := []string{
		"unauthorized",
		"forbidden",
		"authentication",
		"invalid credentials",
		"permission denied",
	}

	for _, fatalErr := range fatalErrors {
		// 这里使用简单的字符串匹配，实际可以更精确
		if len(errStr) > 0 && (errStr == fatalErr || len(fatalErr) <= len(errStr) && errStr[:len(fatalErr)] == fatalErr) {
			return true
		}
	}

	return false
}

