package webserver

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"go-stock/backend/data"
	"go-stock/backend/db"
	"go-stock/backend/models"
	"go-stock/internal/handlers"
	"go-stock/internal/middleware"
)

// WebAssets 嵌入的前端静态资源（由 cmd/web/main.go 注入）
// 如果为 nil，则使用磁盘路径 StaticDir
var WebAssets *embed.FS

// StaticDir 静态资源磁盘路径（WebAssets 为空时使用）
var StaticDir = "frontend/dist"

type WebServer struct {
	router *gin.Engine
	port   string
}

func NewWebServer(port string) *WebServer {
	server := &WebServer{port: port}
	gin.SetMode(gin.ReleaseMode)
	server.initRouter()
	return server
}

func (ws *WebServer) initRouter() {
	ws.router = gin.New()
	ws.router.Use(gin.Logger())
	ws.router.Use(gin.Recovery())

	ws.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "User-Agent"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	ws.router.Use(middleware.DatabaseMiddleware(db.Dao))

	ws.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now()})
	})

	v1 := ws.router.Group("/api/v1")
	{
		public := v1.Group("/public")
		public.Use(middleware.AuthOptional()) // 支持可选认证，获取用户ID
		{
			public.GET("/health", handlers.HealthCheck)
			public.GET("/stocks/search", handlers.SearchStocks)
			public.GET("/stocks/realtime", handlers.GetStockRealTime)
			public.POST("/ai/agent-chat", handlers.AgentChat)
			public.GET("/stocks/:code/kline", handlers.GetStockKLine)
			public.GET("/ai/configs", handlers.GetAIConfigs)
		}

		auth := v1.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
			auth.POST("/logout", middleware.AuthRequired(), handlers.Logout)
		}

		protected := v1.Group("/")
		protected.Use(middleware.AuthOptional())
		{
			protected.GET("/user/profile", handlers.GetUserProfile)
			protected.PUT("/user/profile", handlers.UpdateUserProfile)

			protected.GET("/stocks/basic", handlers.GetStockBasics)
			protected.GET("/stocks/realtime", handlers.GetStockRealTime)
			protected.GET("/stocks/:code", handlers.GetStockByCode)
			protected.GET("/stocks/:code/kline", handlers.GetStockKLine)
			protected.GET("/stocks/:code/minute", handlers.GetStockRealTimePrice)

			protected.POST("/ai/analyze", handlers.AITradeAnalyze)
			protected.GET("/ai/responses", handlers.GetAIResponses)
			protected.DELETE("/ai/responses/:id", handlers.DeleteAIResponse)

			// AI推荐股票
			protected.GET("/ai/recommend-stocks", handlers.GetAIRecommendStocksList)
			protected.DELETE("/ai/recommend-stocks/:id", handlers.DeleteAIRecommendStock)
			protected.PUT("/ai/recommend-stocks/alert", handlers.UpdateAIRecommendStockAlert)

			// AI助手会话管理
			protected.GET("/ai/assistant/session", handlers.GetAiAssistantSessionHandler)
			protected.POST("/ai/assistant/session", handlers.SaveAiAssistantSessionHandler)

			protected.GET("/prompts/templates", handlers.GetPromptTemplates)
			protected.POST("/prompts/templates", handlers.CreatePromptTemplate)
			protected.PUT("/prompts/templates/:id", handlers.UpdatePromptTemplate)
			protected.DELETE("/prompts/templates/:id", handlers.DeletePromptTemplate)

			protected.GET("/trades", handlers.GetTradeRecords)
			protected.POST("/trades", handlers.CreateTradeRecord)
			protected.PUT("/trades/:id", handlers.UpdateTradeRecord)
			protected.DELETE("/trades/:id", handlers.DeleteTradeRecord)

			protected.GET("/settings", handlers.GetUserSettings)
			protected.POST("/settings", handlers.SetUserSetting)
			protected.PUT("/settings/:key", handlers.UpdateUserSetting)
			protected.DELETE("/settings/:key", handlers.DeleteUserSetting)

			// 应用配置（对应 Wails 的 GetConfig / UpdateConfig）
			protected.GET("/config", handlers.GetAppConfig)
			protected.POST("/config", handlers.UpdateAppConfig)

			protected.POST("/stocks/follow", handlers.FollowStock)
			protected.DELETE("/stocks/unfollow", handlers.UnfollowStock)
			protected.GET("/stocks/follow/list", handlers.GetFollowList)
			protected.POST("/stocks/cost", handlers.SetCostPriceAndVolume)
			protected.POST("/stocks/alarm", handlers.SetAlarmChangePercent)
			protected.POST("/stocks/sort", handlers.SetStockSort)
			protected.GET("/stocks/search", handlers.SearchStocks)

			protected.GET("/groups", handlers.GetGroupList)
			protected.POST("/groups", handlers.AddGroup)
			protected.DELETE("/groups/:id", handlers.RemoveGroup)
			protected.PUT("/groups/:id/sort", handlers.UpdateGroupSort)
			protected.GET("/groups/:id/stocks", handlers.GetGroupStockList)
			protected.POST("/groups/:id/stocks", handlers.AddStockGroup)
			protected.DELETE("/groups/:id/stocks", handlers.RemoveStockGroup)

			protected.GET("/funds", handlers.GetFundList)
			protected.GET("/funds/ranking", handlers.GetFundRanking)
			protected.POST("/funds/follow", handlers.FollowFund)
			protected.DELETE("/funds/unfollow", handlers.UnFollowFund)
			protected.GET("/funds/follow/list", handlers.GetFollowedFund)

			// 新闻/电报
			protected.GET("/telegraph", handlers.GetTelegraphList)
			protected.GET("/telegraph/refresh", handlers.RefreshTelegraphList)

			// 市场数据
			protected.GET("/market/global-indexes", handlers.GlobalStockIndexes)
			protected.GET("/market/industry-rank", handlers.GetIndustryRank)
			protected.GET("/market/industry-money-rank", handlers.GetIndustryMoneyRankSina)
			protected.GET("/market/money-rank", handlers.GetMoneyRankSina)
			protected.GET("/market/stocks/:code/money-trend", handlers.GetStockMoneyTrendByDay)
			protected.GET("/market/long-tiger", handlers.GetLongTigerRank)
			protected.GET("/market/hot-stock", handlers.GetHotStock)
			protected.GET("/market/hot-event", handlers.GetHotEvent)
			protected.GET("/market/hot-topic", handlers.GetHotTopic)
			protected.GET("/market/uplimit-hot", handlers.GetUplimitHot)
			protected.GET("/market/change-rank", handlers.GetChangeRank)

			// 研报/公告
			protected.GET("/research/stock-report", handlers.GetStockResearchReport)
			protected.GET("/research/stock-notice", handlers.GetStockNotice)
			protected.GET("/research/industry-report", handlers.GetIndustryResearchReport)

			// 日历
			protected.GET("/calendar/invest", handlers.GetInvestCalendar)
			protected.GET("/calendar/cls", handlers.GetClsCalendar)

			// 交易统计
			protected.GET("/trades/statistics", handlers.GetTradingRecordStatistics)

			// 股票信息
			protected.GET("/stocks/all-info/list", handlers.GetAllStockInfoList)
			protected.GET("/stocks/all", handlers.GetAllStocks)
			protected.GET("/stocks/markets", handlers.GetAllMarkets)
			protected.GET("/stocks/industries", handlers.GetAllIndustries)
			protected.GET("/stocks/concepts", handlers.GetAllConcepts)

			// 技能管理
			protected.GET("/skills", handlers.GetSkillList)
			protected.GET("/skills/all", handlers.GetAllSkills)
			protected.GET("/skills/:id", handlers.GetSkillByID)

			// 定时任务
			protected.GET("/cron-task/types", handlers.GetCronTaskTypes)
			protected.POST("/cron-task", handlers.CreateCronTask)
			protected.GET("/cron-task", handlers.GetCronTaskList)
			protected.GET("/cron-task/:id", handlers.GetCronTaskByID)
			protected.GET("/cron-task/search", handlers.SearchCronTasks)
			protected.GET("/cron-task/validate", handlers.ValidateCronExpr)
			protected.POST("/cron-task/:id/execute", handlers.ExecuteCronTaskNow)
			protected.PUT("/cron-task/:id", handlers.UpdateCronTask)
			protected.DELETE("/cron-task/:id", handlers.DeleteCronTask)
			protected.POST("/cron-task/:id/enable", handlers.EnableCronTask)

			// 分享
			protected.POST("/share/text", handlers.ShareText)
			protected.POST("/share/analysis", handlers.ShareAnalysis)

			// 钉钉
			protected.POST("/dingding/message", handlers.SendDingDingMessage)

			// 赞助
			protected.GET("/sponsor/info", handlers.GetSponsorInfo)

			// 指标选股
			protected.GET("/market/hot-strategy", handlers.GetHotStrategy)
			protected.GET("/stocks/indicator-search", handlers.IndicatorSearchStock)

			// 其他
			protected.GET("/market/em-dict/:code", handlers.GetEMDictCode)
			protected.GET("/market/sentiment", handlers.AnalyzeSentiment)
			protected.GET("/market/hot-words", handlers.GetHotWords)
			protected.GET("/ai/models", handlers.FetchAiModels)
			protected.GET("/trades/check-frequent", handlers.CheckFrequentTrading)

			// 市场统计
			protected.POST("/market/statistic/fetch", handlers.FetchAndSaveMarketStatistic)
			protected.GET("/market/statistic/today", handlers.GetTodayMarketStatistic)
			protected.GET("/market/statistic/recent", handlers.GetRecentDaysMarketStatistic)
			protected.GET("/market/statistic/by-date", handlers.GetMarketStatisticByDate)

			// 异动监控
			protected.GET("/stock-changes", handlers.GetStockChanges)
			protected.GET("/stock-changes/all", handlers.GetAllStockChangesWithPagingHandler)
			protected.GET("/stock-changes/history", handlers.GetStockChangeHistoryHandler)
			protected.POST("/stock-changes/save", handlers.SaveStockChangesToHistoryHandler)
			protected.GET("/stock-changes/daily-stats", handlers.GetDailyChangeStats)
			protected.GET("/stock-changes/type-stats", handlers.GetChangeTypeDailyStats)

			// 用户管理（仅管理员）
			admin := protected.Group("/admin")
			admin.Use(middleware.AdminRequired()) // 应用管理员权限中间件
			{
				admin.GET("/users", handlers.AdminUserList)
				admin.PUT("/users/status", handlers.UpdateUserActiveStatus)
				admin.POST("/users", handlers.CreateUser)
				admin.PUT("/users/:id", handlers.UpdateUserInfoHandler)
				admin.DELETE("/users/:id", handlers.DeleteUserHandler)
				admin.PUT("/users/password", handlers.ResetUserPasswordHandler)
				admin.PUT("/users/:id/role", handlers.UpdateUserRoleHandler)
				admin.PUT("/users/:id/vip", handlers.UpdateUserVipInfoHandler)
			}

			// 临时工具：修复 admin 用户角色（生产环境应删除）
			protected.POST("/tools/fix-admin-role", handlers.FixAdminRole)
		}
	}

	// 静态资源（嵌入 frontend/dist 或磁盘路径）
	if WebAssets != nil {
		distFS, err := fs.Sub(WebAssets, "frontend/dist")
		if err == nil {
			ws.router.NoRoute(func(c *gin.Context) {
				path := strings.TrimPrefix(c.Request.URL.Path, "/")
				if path == "" {
					path = "index.html"
				}
				c.FileFromFS(path, http.FS(distFS))
			})
		} else {
			log.Printf("Warning: could not embed frontend/dist: %v", err)
		}
	} else if StaticDir != "" {
		// 使用磁盘路径
		if _, err := os.Stat(StaticDir); err == nil {
			ws.router.NoRoute(func(c *gin.Context) {
				path := c.Request.URL.Path
				if path == "/" || path == "" {
					path = "/index.html"
				}
				// 尝试文件存在
				filePath := strings.TrimPrefix(path, "/")
				diskPath := StaticDir + "/" + filePath
				if _, err := os.Stat(diskPath); os.IsNotExist(err) {
					// SPA 回退 index.html
					c.File(StaticDir + "/index.html")
					return
				}
				c.File(diskPath)
			})
		} else {
			log.Printf("Warning: static dir %s not found, frontend will not be served", StaticDir)
		}
	}
}

func (ws *WebServer) Start() error {
	log.Printf("Starting web server on port %s", ws.port)
	return ws.router.Run(":" + ws.port)
}

func (ws *WebServer) GetRouter() *gin.Engine {
	return ws.router
}

// MigrateAllTables 执行所有数据库表迁移
func MigrateAllTables() {
	db.Dao.AutoMigrate(
		&data.Group{},
		&data.GroupStock{},
		&data.FollowedStock{},
		&data.StockBasic{},
		&data.StockInfo{},
		&data.Settings{},
		&data.AIConfig{},
		&data.TradingRecord{},
		&data.IndexBasic{},
		&data.FundBasic{},
		&data.FollowedFund{},
		&models.PromptTemplate{},
		&models.AiAssistantSession{},
		&models.AIResponseResult{},
		&models.Tags{},
		&models.Telegraph{},
		&models.TelegraphTags{},
		&models.StockInfoHK{},
		&models.StockInfoUS{},
		&models.GlobalStockIndex{},
		&models.LongTigerRankData{},
		&models.OldSettings{},
		&models.BKDict{},
		&models.AiRecommendStocks{},
		&models.AllStockInfo{},
		&models.CronTask{},
		&models.VersionInfo{},
		&models.StockChangeHistory{},
		&models.MarketStatistic{},
		&handlers.UserSetting{},
		&models.User{},
	)

	// 显式添加 user_id 列（SQLite AutoMigrate 可能不完整）
	// 所有带用户隔离的表：首次部署时自动添加缺失列
	userIDTables := []string{
		"settings",              // data.Settings
		"ai_config",             // data.AIConfig
		"followed_stock",        // data.FollowedStock
		"trading_records",       // data.TradingRecord
		"cron_tasks",           // models.CronTask
		"ai_assistant_sessions", // models.AiAssistantSession
		"ai_recommend_stocks",   // models.AiRecommendStocks
		"followed_fund",         // data.FollowedFund
		"stock_groups",          // data.Group
		"group_stock_info",      // data.GroupStock
		"ai_response_result",    // models.AIResponseResult
	}
	for _, table := range userIDTables {
		migrateUserIDColumn(table)
	}
}

// migrateUserIDColumn 确保表中存在 user_id 列
func migrateUserIDColumn(tableName string) {
	// 检查列是否存在
	var count int64
	db.Dao.Raw("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", tableName, "user_id").Scan(&count)
	if count == 0 {
		// 列不存在，添加它
		db.Dao.Exec("ALTER TABLE " + tableName + " ADD COLUMN user_id INTEGER DEFAULT 0")
		log.Printf("Migrated: Added user_id column to %s", tableName)
	}
}
