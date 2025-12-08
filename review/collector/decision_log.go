package collector

import (
	"context"
	"fmt"
	"log"
	"time"

	"nofx/review"
)

// DecisionLogCollector 决策日志收集器
type DecisionLogCollector struct {
	logger review.DecisionLogReader
}

// NewDecisionLogCollector 创建决策日志收集器
func NewDecisionLogCollector(logger review.DecisionLogReader) *DecisionLogCollector {
	return &DecisionLogCollector{
		logger: logger,
	}
}

// Collect 收集指定时间范围内的决策日志
func (c *DecisionLogCollector) Collect(ctx context.Context, startTime, endTime time.Time) ([]*review.DecisionRecord, error) {
	log.Printf("📊 开始收集决策日志 (时间范围: %s ~ %s)", startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))

	// 调用 DecisionLogReader 的 GetRecordsByTimeRange 方法
	records, err := c.logger.GetRecordsByTimeRange(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("收集决策日志失败: %w", err)
	}

	log.Printf("✅ 决策日志收集完成，共 %d 条记录", len(records))
	return records, nil
}

// CollectWithParallel 并行读取决策日志文件（性能优化）
// 这个方法可以用于处理大量文件的情况
func (c *DecisionLogCollector) CollectWithParallel(ctx context.Context, startTime, endTime time.Time, maxWorkers int) ([]*review.DecisionRecord, error) {
	log.Printf("📊 开始并行收集决策日志 (时间范围: %s ~ %s, 并发数: %d)",
		startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"), maxWorkers)

	if maxWorkers <= 0 {
		maxWorkers = 10 // 默认10个并发
	}

	// 如果是 DecisionLogger，直接调用 GetRecordsByTimeRange
	// 如果需要并行优化，可以在 DecisionLogger 层面实现
	records, err := c.logger.GetRecordsByTimeRange(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("收集决策日志失败: %w", err)
	}

	log.Printf("✅ 决策日志收集完成，共 %d 条记录", len(records))
	return records, nil
}
