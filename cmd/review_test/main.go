package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"nofx/config"
	"nofx/logger"
	"nofx/review"
	"nofx/review/analyzer"
	"nofx/review/collector"
	"nofx/review/matcher"
	"nofx/review/metrics"
	"nofx/review/reporter"
	"nofx/trade_analytics"
	"nofx/trade_history"
)

// DEXDataProviderAdapter 适配器：将 trade_history.ExchangeFillsProvider 适配到 review.DEXDataProvider
type DEXDataProviderAdapter struct {
	provider trade_history.ExchangeFillsProvider
}

// NewDEXDataProviderAdapter 创建适配器
func NewDEXDataProviderAdapter(provider trade_history.ExchangeFillsProvider) *DEXDataProviderAdapter {
	return &DEXDataProviderAdapter{provider: provider}
}

// GetFillsByTimeRange 实现 review.DEXDataProvider 接口
func (a *DEXDataProviderAdapter) GetFillsByTimeRange(startTime, endTime time.Time) ([]trade_history.ExchangeFill, error) {
	return a.provider.GetFillsByTimeRange(startTime, endTime)
}

// TradeAnalyticsServiceAdapter 适配器：将 trade_analytics.Service 适配到 review.TradeAnalyticsService
type TradeAnalyticsServiceAdapter struct {
	service trade_analytics.Service
}

// NewTradeAnalyticsServiceAdapter 创建适配器
func NewTradeAnalyticsServiceAdapter(service trade_analytics.Service) *TradeAnalyticsServiceAdapter {
	return &TradeAnalyticsServiceAdapter{service: service}
}

// GetAnalytics 实现 review.TradeAnalyticsService 接口
func (a *TradeAnalyticsServiceAdapter) GetAnalytics(ctx context.Context, filter interface{}) (interface{}, error) {
	analyticsFilter, ok := filter.(*trade_analytics.AnalyticsFilter)
	if !ok {
		return nil, fmt.Errorf("filter必须是*trade_analytics.AnalyticsFilter类型")
	}
	return a.service.GetAnalytics(ctx, analyticsFilter)
}

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║              复盘模块测试程序                               ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 解析命令行参数
	if len(os.Args) < 2 {
		fmt.Println("用法: go run cmd/review_test/main.go <trader_id> [db_path]")
		fmt.Println("示例: go run cmd/review_test/main.go hyperliquid_xxx config.db")
		os.Exit(1)
	}

	traderID := os.Args[1]
	dbPath := "config.db"
	if len(os.Args) > 2 {
		dbPath = os.Args[2]
	}

	log.Printf("📋 初始化配置数据库: %s", dbPath)
	database, err := config.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("❌ 初始化数据库失败: %v", err)
	}
	defer database.Close()

	// 获取数据库连接
	db, err := database.GetDB()
	if err != nil {
		log.Fatalf("❌ 获取数据库连接失败: %v", err)
	}

	// 1. 创建 DecisionLogger
	// 优先使用 decision_logs_test（test docker v2 使用），如果不存在则使用 decision_logs
	logDirBase := "decision_logs_test"
	if os.Getenv("USE_DECISION_LOGS") == "true" {
		logDirBase = "decision_logs"
	} else {
		// 检查 decision_logs_test 是否存在，如果不存在且 decision_logs 存在，则使用 decision_logs
		if _, err := os.Stat(logDirBase); os.IsNotExist(err) {
			if _, err := os.Stat("decision_logs"); err == nil {
				logDirBase = "decision_logs"
			}
		}
	}
	logDir := fmt.Sprintf("%s/%s", logDirBase, traderID)
	decisionLogger := logger.NewDecisionLogger(logDir)
	decisionLogAdapter := review.NewDecisionLoggerAdapter(decisionLogger)
	log.Printf("✅ DecisionLogger 已初始化 (目录: %s)", logDir)

	// 2. 创建 TradeHistoryRepository
	tradeHistoryRepo := trade_history.NewRepository(db)
	tradeHistoryReader := tradeHistoryRepo // Repository 实现了 review.TradeHistoryReader 接口
	log.Printf("✅ TradeHistoryRepository 已初始化")

	// 3. 创建 DEXDataProvider（这里使用模拟，实际应该从交易所获取）
	// 注意：实际使用时需要从 trader 配置中获取交易所信息
	// 这里简化处理，使用一个空的适配器（需要用户提供真实的 ExchangeFillsProvider）
	var dexDataProvider review.DEXDataProvider
	if os.Getenv("USE_MOCK_DEX") == "true" {
		// 使用模拟数据（可以在测试时使用）
		dexDataProvider = &MockDEXDataProvider{}
		log.Printf("⚠️ 使用模拟 DEX 数据提供者")
	} else {
		log.Printf("⚠️ 未配置真实的 DEX 数据提供者，复盘将无法获取 DEX 数据")
		log.Printf("   设置环境变量 USE_MOCK_DEX=true 使用模拟数据")
		log.Printf("   或在实际集成时提供真实的 ExchangeFillsProvider")
		dexDataProvider = &MockDEXDataProvider{} // 默认使用模拟
	}

	// 4. 创建 TradeAnalyticsService
	analyticsRepo := trade_analytics.NewRepository(db, tradeHistoryRepo)
	analyticsAnalyzer := trade_analytics.NewAnalyzer(analyticsRepo)
	analyticsPairMatcher := trade_analytics.NewPairMatcher(analyticsRepo)
	analyticsService := trade_analytics.NewService(analyticsRepo, analyticsAnalyzer, analyticsPairMatcher)
	tradeAnalyticsAdapter := NewTradeAnalyticsServiceAdapter(analyticsService)
	log.Printf("✅ TradeAnalyticsService 已初始化")

	// 5. 创建 ReviewRepository
	reviewRepo := review.NewReviewRepository(db)
	log.Printf("✅ ReviewRepository 已初始化")

	// 6. 创建所有子组件
	decisionLogCollector := collector.NewDecisionLogCollector(decisionLogAdapter)
	tradeHistoryCollector := collector.NewTradeHistoryCollector(tradeHistoryReader)
	dexDataCollector := collector.NewDEXDataCollector(dexDataProvider)
	ruleEngine := analyzer.NewRuleEngine()
	performanceAnalyzer := analyzer.NewPerformanceAnalyzer(tradeAnalyticsAdapter)
	dexValidationAnalyzer := analyzer.NewDEXValidationAnalyzer()
	decisionMatcher := matcher.NewDecisionMatcher(10*time.Minute, 0.05, 0.01)
	violationMetricsCalc := metrics.NewViolationMetricsCalculator()
	reporter := reporter.NewMarkdownReporter("data/reviews")

	// 7. 创建复盘服务
	reviewService := review.NewReviewService(
		decisionLogCollector,
		tradeHistoryCollector,
		dexDataCollector,
		ruleEngine,
		performanceAnalyzer,
		dexValidationAnalyzer,
		decisionMatcher,
		violationMetricsCalc,
		reporter,
		reviewRepo,
	)

	log.Printf("✅ 复盘服务已初始化")
	fmt.Println()

	// 检查是否是一次性测试还是持续运行
	if os.Getenv("RUN_ONCE") == "true" {
		// 一次性运行：执行一次复盘
		log.Printf("📊 执行一次性复盘测试 (TraderID: %s)", traderID)

		// 计算时间范围（过去6小时）
		endTime := time.Now()
		startTime := endTime.Add(-6 * time.Hour)

		ctx := context.Background()
		result, err := reviewService.RunReview(ctx, traderID, startTime, endTime)
		if err != nil {
			log.Fatalf("❌ 复盘执行失败: %v", err)
		}

		log.Printf("✅ 复盘执行完成")
		log.Printf("   状态: %s", result.Status)
		log.Printf("   耗时: %v", result.Duration)
		if result.ReportPath != "" {
			log.Printf("   报告路径: %s", result.ReportPath)
		}
		if result.ErrorMessage != "" {
			log.Printf("   错误信息: %s", result.ErrorMessage)
		}
	} else {
		// 持续运行：启动调度器
		log.Printf("📅 启动复盘调度器 (TraderID: %s, 间隔: 6小时)", traderID)

		// 配置调度器
		schedulerConfig := review.ReviewSchedulerConfig{
			Interval:      6 * time.Hour,
			FirstRunDelay: 0, // 立即开始第一次复盘
			Enabled:       true,
		}

		scheduler := review.NewReviewScheduler(reviewService, schedulerConfig)

		// 启动调度器
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go scheduler.Start(ctx, traderID)

		log.Printf("✅ 调度器已启动，等待复盘任务执行...")
		fmt.Println()
		fmt.Println("按 Ctrl+C 停止")
		fmt.Println(strings.Repeat("=", 60))

		// 等待退出信号
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		fmt.Println()
		log.Println("📛 收到退出信号，正在停止调度器...")
		cancel()

		// 等待调度器停止
		time.Sleep(2 * time.Second)
		log.Println("✅ 调度器已停止")
	}

	fmt.Println()
	fmt.Println("👋 测试完成！")
}

// MockDEXDataProvider 模拟的 DEX 数据提供者（用于测试）
type MockDEXDataProvider struct{}

// GetFillsByTimeRange 返回空的 ExchangeFill 列表（模拟）
func (m *MockDEXDataProvider) GetFillsByTimeRange(startTime, endTime time.Time) ([]trade_history.ExchangeFill, error) {
	log.Printf("⚠️ MockDEXDataProvider: 返回空数据 (时间范围: %s ~ %s)",
		startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))
	return []trade_history.ExchangeFill{}, nil
}

