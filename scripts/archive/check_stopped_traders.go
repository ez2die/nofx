package main

import (
	"fmt"
	"log"
	"nofx/config"
	"nofx/manager"
	"os"
	"text/tabwriter"
)

func main() {
	// 获取数据库路径
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "config.db"
	}

	// 连接数据库
	database, err := config.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}
	defer database.Close()

	// 创建TraderManager
	traderManager := manager.NewTraderManager()

	// 加载所有trader到内存(以检查哪些已加载)
	err = traderManager.LoadTradersFromDatabase(database)
	if err != nil {
		log.Printf("⚠️  加载trader到内存失败(可能部分已加载): %v", err)
	}

	// 获取所有用户
	userIDs, err := database.GetAllUsers()
	if err != nil {
		log.Fatalf("❌ 获取用户列表失败: %v", err)
	}

	fmt.Printf("\n🔍 检查所有已配置但未运行的trader...\n\n")

	var allTraders []*config.TraderRecord
	var stoppedTraders []*config.TraderRecord
	var loadedButNotRunning []*config.TraderRecord
	var notLoaded []*config.TraderRecord

	// 获取所有用户的trader
	for _, userID := range userIDs {
		traders, err := database.GetTraders(userID)
		if err != nil {
			log.Printf("⚠️  获取用户 %s 的交易员失败: %v", userID, err)
			continue
		}
		allTraders = append(allTraders, traders...)
	}

	fmt.Printf("📊 总配置数: %d 个trader\n", len(allTraders))

	// 检查每个trader的状态
	for _, trader := range allTraders {
		// 检查数据库中的运行状态
		isRunningInDB := trader.IsRunning

		// 检查内存中的运行状态
		var isRunningInMemory bool
		var isLoadedInMemory bool

		if at, err := traderManager.GetTrader(trader.ID); err == nil {
			isLoadedInMemory = true
			status := at.GetStatus()
			if running, ok := status["is_running"].(bool); ok {
				isRunningInMemory = running
			}
		}

		// 判断是否未运行
		if !isRunningInDB || !isRunningInMemory {
			stoppedTraders = append(stoppedTraders, trader)

			if isLoadedInMemory && !isRunningInMemory {
				// 已加载到内存但未运行
				loadedButNotRunning = append(loadedButNotRunning, trader)
			} else if !isLoadedInMemory {
				// 未加载到内存
				notLoaded = append(notLoaded, trader)
			}
		}
	}

	// 使用tabwriter格式化输出
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	if len(stoppedTraders) == 0 {
		fmt.Printf("\n✅ 所有trader都在运行中!\n\n")
		return
	}

	fmt.Printf("\n📋 已配置但未运行的trader: %d 个\n\n", len(stoppedTraders))

	// 打印表头
	fmt.Fprintf(w, "TRADER ID\tTRADER NAME\tUSER ID\tAI MODEL\tEXCHANGE\tDB状态\t内存状态\t已加载\n")
	fmt.Fprintf(w, "---------\t----------\t-------\t--------\t--------\t------\t--------\t------\n")

	// 打印每个未运行的trader
	for _, trader := range stoppedTraders {
		// 检查内存状态
		isLoaded := false
		isRunningInMemory := false
		if at, err := traderManager.GetTrader(trader.ID); err == nil {
			isLoaded = true
			status := at.GetStatus()
			if running, ok := status["is_running"].(bool); ok {
				isRunningInMemory = running
			}
		}

		dbStatus := "❌ 未运行"
		if trader.IsRunning {
			dbStatus = "✅ 运行中"
		}

		memStatus := "❌ 未运行"
		if isRunningInMemory {
			memStatus = "✅ 运行中"
		}

		loadedStatus := "❌ 否"
		if isLoaded {
			loadedStatus = "✅ 是"
		}

		// 提取AI model provider
		aiModelID := trader.AIModelID
		if idx := len(aiModelID) - 1; idx >= 0 && aiModelID[idx] == '_' {
			// 如果以_结尾,去掉
			aiModelID = aiModelID[:idx]
		}
		parts := []rune(aiModelID)
		if len(parts) > 0 && parts[len(parts)-1] == '_' {
			aiModelID = string(parts[:len(parts)-1])
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			trader.ID,
			trader.Name,
			trader.UserID,
			aiModelID,
			trader.ExchangeID,
			dbStatus,
			memStatus,
			loadedStatus,
		)
	}

	w.Flush()

	// 打印统计信息
	fmt.Printf("\n📊 统计信息:\n")
	fmt.Printf("  - 总配置数: %d 个\n", len(allTraders))
	fmt.Printf("  - 未运行总数: %d 个\n", len(stoppedTraders))
	fmt.Printf("  - 已加载但未运行: %d 个\n", len(loadedButNotRunning))
	fmt.Printf("  - 未加载到内存: %d 个\n", len(notLoaded))
	fmt.Printf("\n")
}
