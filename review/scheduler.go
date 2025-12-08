package review

import (
	"context"
	"log"
	"time"
)

// ReviewScheduler 复盘调度器接口
type ReviewScheduler interface {
	Start(ctx context.Context, traderID string) error
	Stop() error
}

// ReviewSchedulerConfig 调度器配置
type ReviewSchedulerConfig struct {
	Interval      time.Duration // 默认6小时
	FirstRunDelay time.Duration // 首次执行延迟
	Enabled       bool          // 是否启用
}

// reviewScheduler 复盘调度器实现
type reviewScheduler struct {
	service ReviewService
	config  ReviewSchedulerConfig
	ticker  *time.Ticker
	done    chan bool
}

// NewReviewScheduler 创建复盘调度器
func NewReviewScheduler(service ReviewService, config ReviewSchedulerConfig) ReviewScheduler {
	if config.Interval <= 0 {
		config.Interval = 6 * time.Hour
	}

	return &reviewScheduler{
		service: service,
		config:  config,
		done:    make(chan bool),
	}
}

// Start 启动调度器
func (s *reviewScheduler) Start(ctx context.Context, traderID string) error {
	if !s.config.Enabled {
		log.Printf("ℹ️ 复盘调度器已禁用")
		return nil
	}

	log.Printf("🕐 启动复盘调度器 (间隔: %v, TraderID: %s)", s.config.Interval, traderID)

	// 创建ticker
	s.ticker = time.NewTicker(s.config.Interval)
	defer s.ticker.Stop()

	// 首次执行延迟
	if s.config.FirstRunDelay > 0 {
		log.Printf("⏳ 首次执行延迟: %v", s.config.FirstRunDelay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.config.FirstRunDelay):
		}
	}

	// 立即执行一次
	go s.runReviewOnce(ctx, traderID)

	// 定时执行
	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 复盘调度器已停止")
			return nil
		case <-s.ticker.C:
			go s.runReviewOnce(ctx, traderID)
		case <-s.done:
			log.Printf("🛑 复盘调度器已停止")
			return nil
		}
	}
}

// Stop 停止调度器
func (s *reviewScheduler) Stop() error {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.done)
	return nil
}

// runReviewOnce 执行一次复盘
func (s *reviewScheduler) runReviewOnce(ctx context.Context, traderID string) {
	// 计算复盘时间范围（过去6小时）
	endTime := time.Now()
	startTime := endTime.Add(-6 * time.Hour)

	log.Printf("🔄 定时触发复盘 (TraderID: %s, 时间范围: %s ~ %s)",
		traderID, startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))

	result, err := s.service.RunReview(ctx, traderID, startTime, endTime)
	if err != nil {
		log.Printf("❌ 复盘执行失败: %v", err)
		return
	}

	if result.Status == "failed" {
		log.Printf("❌ 复盘失败: %s", result.ErrorMessage)
	} else if result.Status == "partial" {
		log.Printf("⚠️ 复盘部分成功: %s", result.ErrorMessage)
	} else {
		log.Printf("✅ 复盘执行成功 (耗时: %v, 报告: %s)", result.Duration, result.ReportPath)
	}
}

