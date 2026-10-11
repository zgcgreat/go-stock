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
	"go-stock/internal/migrations"
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

	// 注册定时任务调度器回调到 handlers（避免 handlers ↔ webserver 循环导入）
	handlers.CronSchedulerHooks.AddCronTask = GetScheduler().AddCronTask
	handlers.CronSchedulerHooks.RemoveCronTask = GetScheduler().RemoveCronTask

	return server
}

func (ws *WebServer) initRouter() {
	ws.router = gin.New()

	// 可信代理：默认不信任 X-Forwarded-For（ClientIP 取真实远端地址，避免伪造绕过限流）。
	// 部署在 Nginx / 反代之后时，用 GO_STOCK_TRUSTED_PROXIES 指定代理网段，例如 "10.0.0.0/8,172.16.0.0/12"。
	if raw := os.Getenv("GO_STOCK_TRUSTED_PROXIES"); raw != "" {
		parts := strings.Split(raw, ",")
		trusted := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				trusted = append(trusted, p)
			}
		}
		if err := ws.router.SetTrustedProxies(trusted); err != nil {
			log.Printf("Warning: invalid GO_STOCK_TRUSTED_PROXIES: %v", err)
		}
	} else {
		_ = ws.router.SetTrustedProxies(nil)
	}

	ws.router.Use(gin.Logger())
	ws.router.Use(gin.Recovery())

	// 生产环境通过 GO_STOCK_CORS_ORIGINS 配置可信前端源（逗号分隔），禁止使用通配符 + 凭据组合。
	// 白名单为空时跳过 CORS 中间件：默认同源部署（前端由本服务托管）无需 CORS，
	// 且 gin-contrib/cors 在 AllowAllOrigins=false 且 AllowOrigins 为空时会 panic 拒绝启动。
	if corsOrigins := parseCORSOrigins(os.Getenv("GO_STOCK_CORS_ORIGINS")); len(corsOrigins) > 0 {
		ws.router.Use(cors.New(cors.Config{
			AllowOrigins:     corsOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "User-Agent"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	ws.router.Use(middleware.DatabaseMiddleware(db.Dao))

	ws.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now()})
	})

	v1 := ws.router.Group("/api/v1")
	{
		// 公开路由：无需认证即可访问
		public := v1.Group("/public")
		{
			public.GET("/health", handlers.HealthCheck)
		}

		auth := v1.Group("/auth")
		{
			auth.POST("/register", middleware.LoginRateLimit(), handlers.Register)
			auth.POST("/login", middleware.LoginRateLimit(), handlers.Login)
			auth.POST("/logout", middleware.AuthRequired(), handlers.Logout)
		}

		// 受保护路由：所有业务接口都需要登录
		protected := v1.Group("/")
		protected.Use(middleware.AuthRequired())
		{
			protected.GET("/user/profile", handlers.GetUserProfile)
			protected.PUT("/user/profile", handlers.UpdateUserProfile)

			protected.GET("/stocks/basic", handlers.GetStockBasics)
			protected.GET("/stocks/realtime", handlers.GetStockRealTime)
			protected.GET("/stocks/price-info", handlers.GetStockPriceInfo)
			protected.GET("/stocks/:code", handlers.GetStockByCode)
			protected.GET("/stocks/:code/kline", handlers.GetStockKLine)
			protected.GET("/stocks/:code/kline/page", handlers.GetStockKLinePage)
			protected.GET("/stocks/:code/minute", handlers.GetStockRealTimePrice)
			protected.GET("/stocks/search", handlers.SearchStocks)

			protected.POST("/ai/analyze", handlers.AITradeAnalyze)
			protected.POST("/ai/agent-chat", handlers.AgentChat)
			protected.POST("/ai/upload-image", handlers.UploadImageHandler)
			protected.GET("/ai/configs", handlers.GetAIConfigs)
			protected.PUT("/ai/configs", handlers.UpdateAiConfigs)
			protected.GET("/ai/responses", handlers.GetAIResponses)
			protected.DELETE("/ai/responses/:id", handlers.DeleteAIResponse)

			// AI推荐股票
			protected.GET("/ai/recommend-stocks", handlers.GetAIRecommendStocksList)
			protected.GET("/ai/recommend-stocks/today-stats", handlers.GetAIRecommendStocksTodayStatsHandler)
			protected.DELETE("/ai/recommend-stocks/:id", handlers.DeleteAIRecommendStock)
			protected.PUT("/ai/recommend-stocks/alert", handlers.UpdateAIRecommendStockAlert)

			// AI助手会话管理
			protected.GET("/ai/assistant/session", handlers.GetAiAssistantSessionHandler)
			protected.POST("/ai/assistant/session", handlers.SaveAiAssistantSessionHandler)
			protected.GET("/ai/assistant/sessions", handlers.ListAiAssistantSessionsHandler)
			protected.DELETE("/ai/assistant/session/:sessionId", handlers.DeleteAiAssistantSessionHandler)

			protected.GET("/prompts/templates", handlers.GetPromptTemplates)
			protected.POST("/prompts/templates", handlers.CreatePromptTemplate)
			protected.PUT("/prompts/templates/:id", handlers.UpdatePromptTemplate)
			protected.DELETE("/prompts/templates/:id", handlers.DeletePromptTemplate)

			// 提示词广场（本地缓存查询）
			protected.GET("/plaza/prompts", handlers.GetPlazaPrompts)
			protected.GET("/plaza/prompts/:extId", handlers.GetPlazaPromptDetail)
			protected.GET("/plaza/categories", handlers.GetPlazaCategories)
			protected.GET("/plaza/questions", handlers.GetPlazaQuestions)
			protected.GET("/plaza/questions/:extId", handlers.GetPlazaQuestionDetail)
			protected.POST("/plaza/sync", handlers.SyncPlazaData)
			protected.GET("/plaza/sync/status", handlers.GetPlazaSyncStatus)
			protected.POST("/plaza/sync/cron-task", handlers.CreatePlazaSyncCronTask)
			// 服务端代理外部广场 API（解决 Web 模式混合内容与 CORS 问题）
			protected.POST("/plaza/proxy", handlers.PlazaProxy)

			protected.GET("/trades", handlers.GetTradeRecords)
			protected.POST("/trades", handlers.CreateTradeRecord)
			protected.PUT("/trades/:id", handlers.UpdateTradeRecord)
			protected.DELETE("/trades/:id", handlers.DeleteTradeRecord)
			protected.POST("/trades/import", handlers.ImportTradingRecordsHandler)
			protected.GET("/trades/template", handlers.ExportTradingRecordTemplateHandler)
			protected.POST("/export/table-xlsx", handlers.ExportTableXLSXHandler)

			// 004 批1：交易时段查询（信号监控引擎 anyMarketOpen 的 Web 数据源）
			protected.GET("/market/trading-time", handlers.GetMarketTradingTimeHandler)

			// 004 批2：信号流水持久化（用户隔离；VIP2 门槛与上游同语义留在客户端）
			protected.POST("/signals/records", handlers.SaveSignalRecordsHandler)
			protected.GET("/signals/records", handlers.GetSignalRecordPageHandler)
			protected.DELETE("/signals/records", handlers.ClearSignalRecordsHandler)
			protected.GET("/signals/stats", handlers.GetSignalStatsHandler)

			// 004 批3：SSE 事件总线（signalMonitorTick 节拍的 Web 载体）
			protected.GET("/events/stream", handlers.EventsStreamHandler)

			// 004 批4：信号提醒推送（feishu/dingding 服务端转发；app 渠道由前端 Notification 承担）
			protected.POST("/signals/notify", handlers.NotifySignalHandler)

			// 005 批5a：合约行情（币安 USDT-M 永续 / Bitget 美股永续）
			// 桌面端走 app.go 的 Wails 绑定，Web 端在 wails-bridge.js 补 HTTP 回退映射
			protected.GET("/contracts/proxy", handlers.GetContractProxyHandler)
			protected.POST("/contracts/proxy", handlers.SetContractProxyHandler)
			protected.GET("/contracts/binance/symbols", handlers.GetBinanceFuturesSymbolsHandler)
			protected.GET("/contracts/binance/ticker", handlers.GetBinanceFuturesTickerHandler)
			protected.GET("/contracts/binance/premium", handlers.GetBinanceFuturesPremiumHandler)
			protected.GET("/contracts/binance/derivatives", handlers.GetBinanceFuturesDerivativesHandler)
			protected.GET("/contracts/binance/funding-rate", handlers.GetBinanceFundingRateHistoryHandler)
			protected.GET("/contracts/bitget/symbols", handlers.GetBitgetFuturesSymbolsHandler)
			protected.GET("/contracts/bitget/ticker", handlers.GetBitgetFuturesTickerHandler)
			protected.GET("/contracts/bitget/derivatives", handlers.GetBitgetFuturesDerivativesHandler)
			protected.GET("/contracts/bitget/funding-rate", handlers.GetBitgetFundingRateHistoryHandler)

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

			protected.GET("/groups", handlers.GetGroupList)
			protected.GET("/groups/all-stocks", handlers.GetAllGroupStocks)
			protected.POST("/groups", handlers.AddGroup)
			protected.PUT("/groups/:id", handlers.UpdateGroup)
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
			protected.GET("/funds/follow/paged", handlers.GetFollowedFundPaged)
			protected.GET("/funds/search", handlers.SearchFundCodesWeb)
			protected.GET("/funds/:code/kline", handlers.GetFundKLineWeb)
			protected.GET("/funds/:code/history", handlers.GetFundHistoryNetValueWeb)
			protected.GET("/funds/:code/holdings", handlers.GetFundTop10HoldingsWeb)

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
			protected.GET("/market/bk-constituents", handlers.GetBKConstituentStocks)
			protected.GET("/market/lhb-daily-summary", handlers.GetLhbDailySummary)

			// 政策新闻（部委政策模块）。列表/部门为全局共享只读数据；重点部门为全局配置，写需管理员。
			protected.GET("/news/policy", handlers.GetPolicyNews)
			protected.GET("/news/policy/stored", handlers.GetStoredPolicyNews)
			protected.GET("/news/policy/all", handlers.GetAllDeptPolicyNews)
			protected.GET("/news/policy/departments", handlers.GetGovDepartments)
			protected.GET("/news/policy/key-departments", handlers.GetKeyDepartments)
			policyAdmin := protected.Group("/news/policy")
			policyAdmin.Use(middleware.AdminRequired())
			policyAdmin.POST("/key-departments", handlers.SaveKeyDepartments)

			// 游资席位名录（全局共享配置：读开放、写/重置/远程刷新需管理员，远程刷新带 SSRF 防护）
			protected.GET("/settings/hot-money-seats", handlers.GetHotMoneySeats)
			hotMoneyAdmin := protected.Group("/settings/hot-money-seats")
			hotMoneyAdmin.Use(middleware.AdminRequired())
			hotMoneyAdmin.POST("", handlers.SaveHotMoneySeats)
			hotMoneyAdmin.POST("/reset", handlers.ResetHotMoneySeats)
			hotMoneyAdmin.POST("/refresh", handlers.RefreshHotMoneySeats)

			// 复盘/晨报（全局共享 AI 生成内容：读开放；生成消耗 AI 额度、删除不可逆 → 需管理员）
			protected.GET("/agent/daily-review", handlers.GetDailyReviewByDateHandler)
			protected.GET("/agent/daily-reviews", handlers.GetDailyReviewListHandler)
			protected.GET("/agent/morning-strategy", handlers.GetMorningStrategyByDateHandler)
			protected.GET("/agent/morning-strategies", handlers.GetMorningStrategyListHandler)
			reviewAdmin := protected.Group("/agent")
			reviewAdmin.Use(middleware.AdminRequired())
			reviewAdmin.POST("/daily-review/generate", handlers.GenerateDailyReviewHandler)
			reviewAdmin.POST("/daily-review/delete", handlers.DeleteDailyReviewHandler)
			reviewAdmin.POST("/morning-strategy/generate", handlers.GenerateMorningStrategyHandler)
			reviewAdmin.POST("/morning-strategy/delete", handlers.DeleteMorningStrategyHandler)

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

			// 概念标签（独立于分组）。全局共享数据：读操作对用户开放，写操作需管理员，防止任意用户篡改他人概念。
			protected.GET("/concepts", handlers.GetConceptList)
			protected.GET("/concepts/stocks/all", handlers.GetAllStockConcepts)
			protected.GET("/concepts/stocks/by-code", handlers.GetStockConceptsByStockCode)
			conceptAdmin := protected.Group("/concepts")
			conceptAdmin.Use(middleware.AdminRequired())
			{
				conceptAdmin.POST("", handlers.AddConcept)
				conceptAdmin.PUT("/:id", handlers.UpdateConcept)
				conceptAdmin.DELETE("/:id", handlers.RemoveConcept)
				conceptAdmin.POST("/stocks", handlers.AddStockConcept)
				conceptAdmin.DELETE("/stocks", handlers.RemoveStockConcept)
			}

			// 每日操作计划
			protected.GET("/daily-operation-plans", handlers.GetDailyOperationPlanList)
			protected.GET("/daily-operation-plans/:id", handlers.GetDailyOperationPlanByID)
			protected.POST("/daily-operation-plans", handlers.SaveDailyOperationPlan)
			protected.DELETE("/daily-operation-plans/:id", handlers.DeleteDailyOperationPlan)
			protected.PUT("/daily-operation-plans/:id/status", handlers.UpdateDailyOperationPlanStatus)
			protected.PUT("/daily-operation-plans/:id/alert", handlers.UpdateDailyOperationPlanAlert)

			// TDX 分时/成交数据
			protected.GET("/tdx/minute", handlers.GetTdxMinuteTimeData)
			protected.GET("/tdx/transactions", handlers.GetAllTdxTransactionData)

			// 技能管理（全局共享配置，写操作需管理员）
			protected.GET("/skills", handlers.GetSkillList)
			protected.GET("/skills/all", handlers.GetAllSkills)
			protected.GET("/skills/export", handlers.ExportAllSkills)
			protected.GET("/skills/:id", handlers.GetSkillByID)
			protected.GET("/skills/:id/export", handlers.ExportSkillByID)
			skillAdmin := protected.Group("/skills")
			skillAdmin.Use(middleware.AdminRequired())
			{
				skillAdmin.POST("", handlers.CreateSkill)
				skillAdmin.POST("/import", handlers.ImportSkills)
				skillAdmin.PUT("/:id", handlers.UpdateSkill)
				skillAdmin.DELETE("/:id", handlers.DeleteSkill)
				skillAdmin.POST("/:id/enable", handlers.EnableSkill)
			}

			// MCP 服务器管理（全局共享配置，写操作需管理员）
			protected.GET("/mcp/servers", handlers.GetMCPServerList)
			protected.GET("/mcp/servers/:id", handlers.GetMCPServerByID)
			protected.GET("/mcp/servers/:id/tools", handlers.GetMCPToolsByServerID)
			protected.GET("/mcp/tools", handlers.GetAllMCPTools)
			mcpAdmin := protected.Group("/mcp")
			mcpAdmin.Use(middleware.AdminRequired())
			{
				mcpAdmin.POST("/servers", handlers.CreateMCPServer)
				mcpAdmin.PUT("/servers/:id", handlers.UpdateMCPServer)
				mcpAdmin.DELETE("/servers/:id", handlers.DeleteMCPServer)
				mcpAdmin.POST("/servers/:id/enable", handlers.EnableMCPServer)
				mcpAdmin.POST("/servers/:id/test", handlers.TestMCPServer)
				mcpAdmin.POST("/servers/:id/oauth", handlers.StartMCPOAuthHandler)
			}

			// 定时任务
			protected.GET("/cron-task/types", handlers.GetCronTaskTypes)
			protected.POST("/cron-task", handlers.CreateCronTask)
			protected.GET("/cron-task", handlers.GetCronTaskList)
			protected.GET("/cron-task/:id", handlers.GetCronTaskByID)
			protected.GET("/cron-task/search", handlers.SearchCronTasks)
			protected.GET("/cron-task/validate", handlers.ValidateCronExpr)
			protected.GET("/cron-task/calculate-next-runs", handlers.CalculateNextRunTimes)
			protected.GET("/cron-task/calculate-next-run", handlers.CalculateNextRunTime)
			protected.POST("/cron-task/:id/execute", handlers.ExecuteCronTaskNow)
			protected.PUT("/cron-task/:id", handlers.UpdateCronTask)
			protected.DELETE("/cron-task/:id", handlers.DeleteCronTask)
			protected.POST("/cron-task/:id/enable", handlers.EnableCronTask)

			// 自定义选股策略
			protected.GET("/custom-strategies", handlers.GetCustomStrategies)
			protected.GET("/custom-strategies/list", handlers.GetCustomStrategyList)
			protected.POST("/custom-strategies", handlers.SaveCustomStrategy)
			protected.DELETE("/custom-strategies/:id", handlers.DeleteCustomStrategy)

			// 分享
			protected.POST("/share/text", handlers.ShareText)
			protected.POST("/share/analysis", handlers.ShareAnalysis)

			// 钉钉
			protected.POST("/dingding/message", handlers.SendDingDingMessage)

			// 飞书消息推送（用户隔离，与桌面端 SendFeishuMessageByType 对齐）
			protected.POST("/feishu/message", handlers.SendFeishuMessage)

			// 赞助
			protected.GET("/sponsor/info", handlers.GetSponsorInfo)
			protected.POST("/sponsor/check", handlers.CheckSponsorCode)

			// 指标选股
			protected.GET("/market/hot-strategy", handlers.GetHotStrategy)
			protected.GET("/stocks/indicator-search", handlers.IndicatorSearchStock)

			// 其他
			protected.GET("/market/em-dict/:code", handlers.GetEMDictCode)
			protected.GET("/market/sentiment", handlers.AnalyzeSentiment)
			protected.GET("/market/hot-words", handlers.GetHotWords)
			protected.GET("/ai/models", handlers.FetchAiModels)
			protected.GET("/ai/model-info", handlers.FetchAiModelInfo)
			protected.GET("/trades/check-frequent", handlers.CheckFrequentTrading)

			// 市场统计
			protected.POST("/market/statistic/fetch", handlers.FetchAndSaveMarketStatistic)
			protected.GET("/market/statistic/today", handlers.GetTodayMarketStatistic)
			protected.GET("/market/statistic/recent", handlers.GetRecentDaysMarketStatistic)
			protected.GET("/market/statistic/by-date", handlers.GetMarketStatisticByDate)

			// 板块资金流向
			protected.POST("/market/bk-fund-flow/fetch", handlers.FetchAndSaveBKFundFlow)
			protected.GET("/market/bk-fund-flow/top", handlers.GetBKFundFlowTopListByDate)
			protected.GET("/market/bk-fund-flow/top-latest", handlers.GetBKFundFlowTopList)
			protected.GET("/market/bk-fund-flow/list", handlers.GetBKFundFlowListByDate)
			protected.GET("/market/bk-fund-flow/history", handlers.GetBKFundFlowList)
			protected.GET("/market/bk-fund-flow/codes", handlers.GetAllBKCodes)

			// 概念资金流向
			protected.GET("/market/concept-fund-flow/codes", handlers.GetAllConceptCodes)
			protected.GET("/market/concept-fund-flow/list", handlers.GetConceptFundFlowListByDate)
			protected.GET("/market/concept-fund-flow/top", handlers.GetConceptFundFlowTopListByDate)
			protected.POST("/market/concept-fund-flow/fetch", handlers.FetchAndSaveConceptFundFlow)

			// 概念板块详情/K线/成分股/实时行情
			protected.GET("/market/concept-detail", handlers.GetConceptDetail)
			protected.GET("/market/concept-kline", handlers.GetConceptKLine)
			protected.GET("/market/concept-realhead", handlers.GetConceptRealHead)
			protected.GET("/market/concept-stocks", handlers.GetConceptStocks)
			protected.GET("/market/concept-plates", handlers.GetAllConceptPlates)
			protected.GET("/market/concept-code-by-name", handlers.FindConceptCodeByName)

			// 每日炒作题材
			protected.GET("/market/concept-events", handlers.GetConceptEventList)

			// 融资融券排名
			protected.GET("/market/rzrq-rank", handlers.GetRzrqRank)

			// 股指期货持仓
			protected.GET("/market/futures-position", handlers.GetFuturesPositionTrend)

			// 知识库
			protected.GET("/knowledge-base/list", handlers.ListKnowledgeBases)
			protected.POST("/knowledge-base/create", handlers.CreateKnowledgeBase)
			protected.POST("/knowledge-base/delete", handlers.DeleteKnowledgeBase)
			protected.GET("/knowledge-base/info", handlers.GetKnowledgeBaseInfo)
			protected.POST("/knowledge-base/add-doc", handlers.AddKBDocument)
			protected.POST("/knowledge-base/search", handlers.SearchKnowledgeBaseHandler)
			protected.GET("/knowledge-base/documents", handlers.ListKBDocuments)
			protected.POST("/knowledge-base/delete-doc", handlers.DeleteKBDocument)
			protected.GET("/knowledge-base/vectorizing-status", handlers.GetKBVectorizingStatus)
			protected.GET("/knowledge-base/vectorizing-statuses", handlers.GetAllKBVectorizingStatuses)
			protected.POST("/knowledge-base/search-all", handlers.SearchAllKnowledge)
			protected.GET("/knowledge-base/documents-paged", handlers.ListKBDocumentsPagedHandler)
			protected.GET("/knowledge-base/long-term-memory-info", handlers.GetLongTermMemoryInfoHandler)
			protected.POST("/knowledge-base/search-long-term-memory", handlers.SearchLongTermMemoryHandler)
			protected.POST("/knowledge-base/chat-qa", handlers.ChatWithAgentKBQAHandler)
			protected.GET("/settings/long-term-memory-ai-config-id", handlers.GetLongTermMemoryAiConfigIdHandler)
			protected.POST("/settings/long-term-memory-ai-config-id", handlers.SetLongTermMemoryAiConfigIdHandler)

			// 用户画像
			protected.GET("/user-profile", handlers.GetUserProfileHandler)
			protected.POST("/user-profile/save", handlers.SaveUserProfileHandler)
			protected.GET("/user-profile/enabled", handlers.GetUserProfileEnabled)
			protected.POST("/user-profile/enabled", handlers.SetUserProfileEnabled)
			protected.GET("/user-profile/updated-at", handlers.GetUserProfileUpdatedAt)
			protected.POST("/user-profile/relearn", handlers.RelearnUserProfile)
			protected.POST("/user-profile/clear", handlers.ClearUserProfile)
			protected.GET("/user-profile/feedback-stats", handlers.GetAgentFeedbackStats)
			protected.GET("/user-profile/feedback-list", handlers.ListAgentFeedbackPaged)
			protected.POST("/user-profile/feedback", handlers.SubmitAgentFeedbackHandler)
			protected.GET("/user-profile/learn-ai-config", handlers.GetProfileLearnAiConfigIdHandler)
			protected.POST("/user-profile/learn-ai-config", handlers.SetProfileLearnAiConfigIdHandler)
			protected.GET("/agent/backtest/template-detail", handlers.GetPromptTemplateBacktestDetailHandler)

			// 提示词回测 + 推荐回测（读开放；创建/执行/删除消耗算力/不可逆 → 需管理员）
			protected.GET("/agent/prompt-backtest/tasks", handlers.GetPromptBacktestTaskListHandler)
			protected.GET("/agent/prompt-backtest/tasks/:id", handlers.GetPromptBacktestTaskDetailHandler)
			protected.GET("/agent/prompt-backtest/picks", handlers.GetPromptBacktestPicksHandler)
			protected.GET("/agent/backtest", handlers.ListRecommendBacktestHandler)
			protected.GET("/agent/backtest/by-prompt", handlers.ListRecommendBacktestByPromptHandler)
			protected.GET("/agent/backtest/by-template", handlers.ListRecommendBacktestByTemplateHandler)
			protected.GET("/agent/backtest/by-skill", handlers.ListRecommendBacktestBySkillHandler)
			protected.GET("/agent/backtest/stats", handlers.GetRecommendBacktestStatsHandler)
			backtestAdmin := protected.Group("/agent")
			backtestAdmin.Use(middleware.AdminRequired())
			backtestAdmin.POST("/prompt-backtest/tasks", handlers.CreatePromptBacktestTaskHandler)
			backtestAdmin.POST("/prompt-backtest/tasks/:id/delete", handlers.DeletePromptBacktestTaskHandler)
			backtestAdmin.POST("/backtest/run", handlers.RunRecommendBacktestHandler)

			// 异动监控
			protected.GET("/stock-changes", handlers.GetStockChanges)
			protected.GET("/stock-changes/all", handlers.GetAllStockChangesWithPagingHandler)
			protected.GET("/stock-changes/history", handlers.GetStockChangeHistoryHandler)
			protected.POST("/stock-changes/save", handlers.SaveStockChangesToHistoryHandler)
			protected.GET("/stock-changes/daily-stats", handlers.GetDailyChangeStats)
			protected.GET("/stock-changes/type-stats", handlers.GetChangeTypeDailyStats)
			protected.GET("/stock-changes/dimension-stats", handlers.GetDailyDimensionStatsHandler)
			protected.GET("/stock-changes/type-stats-by-date", handlers.GetTypeStatsByDateHandler)

			// 用户手册
			protected.GET("/manual", handlers.GetUserManual)

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
		&data.Concept{},
		&data.ConceptStock{},
		&data.StockInfo{},
		&data.Settings{},
		&data.AIConfig{},
		&data.TradingRecord{},
		&data.IndexBasic{},
		&data.FundBasic{},
		&data.FollowedFund{},
		&models.DailyOperationPlan{},
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
		&models.BKDict{},
		&models.AiRecommendStocks{},
		&models.AllStockInfo{},
		&models.CronTask{},
		&models.VersionInfo{},
		&models.StockChangeHistory{},
		&models.MarketStatistic{},
		&models.BKFundFlow{},
		&models.ConceptFundFlow{},
		&models.PlazaPrompt{},
		&models.PlazaQuestion{},
		&models.CustomStrategy{},
		&models.MCPServer{},
		&models.MCPServerTool{},
		&models.Skill{},
		&handlers.UserSetting{},
		&models.User{},
		&models.SignalRecord{}, // 004 批2：Web 新库此前不建信号表
		&models.MorningStrategy{}, // 盘前策略表：Web 此前未建表，手动生成/查询会报 no such table
		&models.DailyReview{}, // 复盘报告表：Web 此前未建表，手动生成/查询会报 no such table
	)

	// Web 端 SQLite 兼容迁移：用户隔离列与历史 settings 缺列补丁。
	migrations.ApplyWebCompatibilityMigrations()
}

// parseCORSOrigins 将逗号分隔的源字符串解析为去空白后的列表。
// 未配置时返回空列表（仅允许同源访问，跨域前端需显式配置 GO_STOCK_CORS_ORIGINS）。
func parseCORSOrigins(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
