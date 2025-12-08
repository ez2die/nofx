package review

import (
	"time"

	"nofx/logger"
)

// DecisionLoggerAdapter 将 logger.DecisionLogger 适配为 review.DecisionLogReader
type DecisionLoggerAdapter struct {
	logger *logger.DecisionLogger
}

// NewDecisionLoggerAdapter 创建适配器
func NewDecisionLoggerAdapter(logger *logger.DecisionLogger) *DecisionLoggerAdapter {
	return &DecisionLoggerAdapter{
		logger: logger,
	}
}

// GetRecordsByTimeRange 实现 DecisionLogReader 接口
func (a *DecisionLoggerAdapter) GetRecordsByTimeRange(startTime, endTime time.Time) ([]*DecisionRecord, error) {
	// 调用 logger 的方法
	loggerRecords, err := a.logger.GetRecordsByTimeRange(startTime, endTime)
	if err != nil {
		return nil, err
	}

	// 转换为 review.DecisionRecord
	records := make([]*DecisionRecord, len(loggerRecords))
	for i, lr := range loggerRecords {
		records[i] = convertLoggerDecisionRecord(lr)
	}

	return records, nil
}

// convertLoggerDecisionRecord 将 logger.DecisionRecord 转换为 review.DecisionRecord
func convertLoggerDecisionRecord(src *logger.DecisionRecord) *DecisionRecord {
	if src == nil {
		return nil
	}

	// 转换 Decisions
	decisions := make([]DecisionAction, len(src.Decisions))
	for i, d := range src.Decisions {
		decisions[i] = DecisionAction{
			Action:          d.Action,
			Symbol:          d.Symbol,
			Quantity:        d.Quantity,
			Leverage:        d.Leverage,
			Price:           d.Price,
			OrderID:         d.OrderID,
			Timestamp:       d.Timestamp,
			Success:         d.Success,
			Error:           d.Error,
			IsAutoTriggered: d.IsAutoTriggered,
			WasStopLoss:     d.WasStopLoss,
		}
	}

	// 转换 Positions
	positions := make([]PositionSnapshot, len(src.Positions))
	for i, p := range src.Positions {
		positions[i] = PositionSnapshot{
			Symbol:           p.Symbol,
			Side:             p.Side,
			PositionAmt:      p.PositionAmt,
			EntryPrice:       p.EntryPrice,
			MarkPrice:        p.MarkPrice,
			UnrealizedProfit: p.UnrealizedProfit,
			Leverage:         p.Leverage,
			LiquidationPrice: p.LiquidationPrice,
		}
	}

	return &DecisionRecord{
		Timestamp:      src.Timestamp,
		CycleNumber:    src.CycleNumber,
		SystemPrompt:   src.SystemPrompt,
		InputPrompt:    src.InputPrompt,
		CoTTrace:       src.CoTTrace,
		DecisionJSON:   src.DecisionJSON,
		AccountState: AccountSnapshot{
			TotalBalance:          src.AccountState.TotalBalance,
			AvailableBalance:      src.AccountState.AvailableBalance,
			TotalUnrealizedProfit: src.AccountState.TotalUnrealizedProfit,
			PositionCount:         src.AccountState.PositionCount,
			MarginUsedPct:         src.AccountState.MarginUsedPct,
		},
		Positions:      positions,
		CandidateCoins: src.CandidateCoins,
		Decisions:      decisions,
		ExecutionLog:   src.ExecutionLog,
		Success:        src.Success,
		ErrorMessage:   src.ErrorMessage,
	}
}

