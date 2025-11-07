package trade_history

import (
	"context"
	"log"
	"time"
)

// SyncService 同步服务
type SyncService struct {
	service  Service
	interval time.Duration
}

// NewSyncService 创建同步服务
func NewSyncService(service Service, interval time.Duration) *SyncService {
	return &SyncService{
		service:  service,
		interval: interval,
	}
}

// Start 启动定期同步
func (s *SyncService) Start(ctx context.Context, traderID string, provider ExchangeFillsProvider) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Printf("🔄 交易历史同步服务已启动（trader_id=%s, interval=%v）", traderID, s.interval)

	// 立即执行一次同步
	if err := s.service.SyncFromExchange(ctx, traderID, provider); err != nil {
		log.Printf("⚠️ 初始同步失败: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 交易历史同步服务已停止（trader_id=%s）", traderID)
			return
		case <-ticker.C:
			if err := s.service.SyncFromExchange(ctx, traderID, provider); err != nil {
				log.Printf("⚠️ 定期同步失败: %v", err)
			}
		}
	}
}

// SyncOnce 执行一次同步
func (s *SyncService) SyncOnce(ctx context.Context, traderID string, provider ExchangeFillsProvider) error {
	return s.service.SyncFromExchange(ctx, traderID, provider)
}

