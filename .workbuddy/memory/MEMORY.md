# MEMORY.md - go-stock 项目长期记忆

## 项目基本信息
- **项目名**: go-stock
- **定位**: 基于大语言模型的 AI 赋能股票分析工具（桌面应用）
- **技术栈**: Go + Wails + Vue 3 + NaiveUI + Vite
- **数据库**: GORM（本地 SQLite）
- **AI 接入**: OpenAI 兼容接口，支持 DeepSeek、Ollama、LMStudio、硅基流动、火山方舟等

## 目录结构
- `backend/data/` — 数据层：行情爬虫、AI 接口、东方财富K线、市场资讯、基金、工具函数集（tool_*.go）
- `backend/agent/` — AI 智能体：工具调用 Agent，带 chat memory
- `backend/models/` — 数据模型（GORM）
- `backend/db/` — 数据库操作
- `frontend/src/components/` — 40+ Vue 组件（行情卡片、K线图、AI分析、市场资讯等）

## 当前开发状态（2026-04-05）
- 所在分支：dev
- Web 化改造已基本完成（2026-04-05 下午），具体内容：
  - frontend/src/services/wails-bridge.js：完全重写，消除重复函数，AI 流式接口（NewChatStream/ChatWithAgent/SummaryStockNews）已实现 SSE 接入
  - frontend/src/utils/auth.js：新建，修复 websocket.js 引用
  - internal/webserver/server.go：Web 服务器核心（从 web_server.go 迁移）
  - cmd/web/main.go：Web 服务器独立入口，go build -o go-stock-web.exe ./cmd/web
  - internal/handlers/web_handler.go：补全 SendDingDingMessage/FetchAiModels/CheckFrequentTrading 真实实现
- Web 认证改造（2026-04-05 傍晚）：
  - 新增 AuthOptional 中间件，Web 端无需登录即可使用所有功能（匿名用户 ID=0）
  - 前端多处 .map/.filter 增加 Array.isArray 防御性处理
- 剩余低优先级：定时任务 Web 端 stub、ai-assistant-web 未集成

## Web 端用户隔离修复（2026-04-26）
- `stock_follow_handler.go`：`UnfollowStock`/`GetFollowList`/`SetCostPriceAndVolume`/`SetAlarmChangePercent`/`SetStockSort` 加用户验证
- `stock_handler.go`：`GetStockRealTime` 持仓填充加用户过滤
- `data.TradingRecord`/`trading_service.go`/`trade_handler.go`：交易记录加用户隔离
- `models.CronTask`/`cron_task_api.go`/`web_handler.go`/`app.go`：定时任务加用户隔离
- `AiRecommendStocksQuery`/`AiRecommendStocksService`/`ai_handler.go`/`app_common.go`：AI 推荐加用户隔离
- `AiAssistantSession`/`ai_assistant_api.go`/`ai_handler.go`/`app.go`/`ai-assistant-web/server.go`：AI 会话加用户隔离
- `AIAnalyzeRequest`/`DeleteAIResponse`（ai_handler.go）：AI 分析结果加用户隔离
- 设计原则：`userID=0` 桌面端不过滤，`userID>0` Web 端过滤；所有修改均为扩展式，不改桌面端逻辑
- 全部编译通过 ✅；`StockChangeHistory` 加用户隔离未完成（低优先级）

## Web 端 Bug 修复记录（2026-04-05 晚）
- **Unfollow 400 修复**：前端 DELETE /stocks/unfollow 从 params 改为 JSON body {stockCode}
- **K线 404 修复**：protected 路由组补注册 GET /stocks/:code/kline
- **K线空数据崩溃修复**：KLineChart.vue 增加空数据防御性检查
- **设置保存 400 修复**：新增 /config GET/POST 路由（对应 Wails 的 GetConfig/UpdateConfig），前端 GetConfig/UpdateConfig 从 /settings 改走 /config
- **设置保存"成功但报错"修复**：UpdateConfig 返回 "保存成功！" 被当作非空错误处理，改为忽略返回值（UpdateConfig 返回的是提示文字不是错误标识）
- **提示词模板不显示修复**：GetPromptTemplateList 改为直接返回完整分页对象（res.data?.data），不用 extractApiData（它会自动提取 list 导致丢失 total/totalPages）
- **AI 分析 400 修复**：SummaryStockNews Web 模式下请求体字段映射错误（arg1/arg2 误当 stockCode/stockName），按 Wails 参数签名正确映射 question/aiConfigId/sysPromptId；同时将硬编码事件名改为动态 eventName
- **NewChatStream 400 修复**：JS 函数参数签名与 Go 端不一致（userPromptId/aiConfigId 顺序错误导致 bool 传到 aiConfigId），修正为匹配 Go 端 (stock, stockCode, question, aiConfigId, sysPromptId, enableTools, think)，body 中 stockCode/stockName 字段映射也做了修正，加 Number() 类型转换
- **AI 流式内容不显示修复**：NewChatStream 和 SummaryStockNews 的 SSE processLine 直接发纯文本 raw 字符串，但前端事件监听期望 { content, chatId, question } 对象格式。修复为 EventsEmit(eventName, { content: raw }) 包装对象后发送

## 编译修复（2026-04-06 下午）
`go build ./cmd/web/...` 编译成功，`go run ./cmd/web/main.go` 启动正常（http://0.0.0.0:8080）
- stock_service.go：修复 getHistoryData（NewEastMoneyKLineApi 传 config，改用 GetDayKLine，正确字段映射）；清理未使用 import；补加 context import
- trading_service.go：清理未使用 import（sync/strutil），修复 price 未使用变量
- ai_assistant_service.go、settings_service.go、market_service.go：清理未使用 import
- backend/agent/tools/data_tools_wrapper.go：修复两处 \\n\\t 字面量错误（展开为多行代码）
- backend/agent/tools/bk_dict_tool.go：删除未使用 freecache import
- internal/handlers/admin_handler.go：修复 GetUserList/UpdateUserStatus 调用多传 db.Dao 参数
- internal/handlers/trade_handler.go：完全重写，用 data.TradingRecord + TradingService 替代不存在的 models.StockTradeRecord
- internal/handlers/web_handler.go：GetTradingRecordStatistics 改用 TradingService；添加 services import
- internal/webserver/server.go：AutoMigrate 删除不存在的 models.StockTradeRecord/StockPool/AIRecommendStocksHistory/AIRecommendStocksSummary
- app.go：添加 go-stock/backend/services import
- 注意：go build ./...（根包）仍报错（BuildKey/Version/PanicHandler 未定义），这是 Wails 桌面端问题，需 Wails toolchain，属正常现象

## K线接口修复（2026-04-26 傍晚）
- **问题**：`GET /api/v1/stocks/600522.SH/kline` 返回空数据 `{"code":0,"data":[]}`
- **根因**：
  1. 腾讯日K接口（`web.ifzq.gtimg.cn`）只支持港股代码，不支持A股格式
  2. 东方财富接口调用时 config 可能有问题，导致空数据
  3. 前端用 `limit=800` 参数但 handler 只识别 `days`
- **修复**：`GetStockKLine` 完全复用桌面端 `FetchKLineWithFallback` 函数，自动多数据源切换；同时支持 `days` 和 `limit` 参数
- **编译**：`go build ./web/...` ✅

## Web端 /config 用户隔离修复（2026-04-26 傍晚）
- **问题**：`/config` 接口没有用户隔离，所有用户共享同一配置
- **根因**：`Settings` 和 `AIConfig` 表没有 `user_id` 字段
- **修复**：
  - `Settings`/`AIConfig` 结构体添加 `UserID` 字段
  - 新增 `GetSettingConfigByUserID(userID uint)` 和 `UpdateConfigByUserID(userID uint, s *SettingConfig)`
  - 修改 `GetAppConfig`/`UpdateAppConfig`/`AITradeAnalyze` 使用用户隔离配置
- **编译**：`go build ./web/...` ✅

## SQLite user_id 列迁移修复（2026-04-26 傍晚）
- **问题**：GORM AutoMigrate 对 SQLite 添加列支持不完善，首次部署时 `user_id` 列缺失
- **修复**：`MigrateAllTables` 中添加 `migrateUserIDColumn()` 函数，显式检查并添加缺失列
- **涉及的表**（共11个用户隔离表）：
  - settings, ai_config, followed_stock, trading_records
  - cron_tasks, ai_assistant_sessions, ai_recommend_stocks
  - followed_fund, stock_groups, group_stock_info, ai_response_result
- **编译**：`go build ./web/...` ✅

## AI Agent 模型选择为空修复（2026-04-26 晚）
- **问题1**：新用户首次登录时，AI Agent 助手选择模型为空
- **修复1**：`GetSettingConfigByUserID` 增加逻辑，当用户无配置时从系统默认配置（user_id=0）复制模板
- **问题2**：`/public/ai/configs` 路由没有 AuthOptional 中间件，无法获取当前用户 ID
- **修复2**：
  - public 组添加 `middleware.AuthOptional()` 中间件
  - `GetAIConfigs` 使用 `middleware.GetUserIDFromContext(c)` 获取用户 ID
- **涨停梯队修复**：
  - 新增 `GetUplimitHot` Web Handler（复用 `data.NewMarketNewsApi().GetUplimitHot()`）
  - 注册路由 `GET /market/uplimit-hot`
  - 前端 `wails-bridge.js` Web 模式调用后端 API
- **编译**：`go build ./web/...` ✅

## agent-chat.vue Markdown渲染修复（2026-06-04）
- **问题**：桌面端 Agent 聊天组件使用 `t-chat-content`（marked库），但 style.css 的 Markdown 增强样式全部写给 `.md-editor-preview`，导致表格/标题/分隔线等样式全部失效
- **修复**：将3处 `<t-chat-content>` 替换为 `<MdPreview>`（md-editor-v3），与 FloatingAgentAssistant.vue 和 ai-assistant-web/App.vue 统一
- **同步修复**：`formatMarkdown` 中 `---##` 拆分问题（hrHeadingMatch/hrBlockMatch）、`splitInlineHeading` 中 hrMatch 修复
- **新增**：darkThemeRef/mdTheme/codeTheme/onMdHtmlChanged、MdPreview样式穿透、代码块增强样式
- **涉及提交**：e07e198(style.css增强)、acd3406(style.css优化) — CSS选择器与 t-chat-content 不匹配

## Web端 Agent Chat 配置查询用户隔离修复（2026-06-06）
- **问题**：Web端 `AgentChat` 和 `AITradeAnalyze` 调用 `agent.ChatWithContext` 时，Agent 内部用 `GetSettingConfig()` 查 `user_id=0` 全局配置，登录用户找不到自己的AI配置
- **修复**：
  - `agent_api.go`：签名不变（桌面端不受影响），通过 `optsOverride[2]` 传递 userID 字符串
  - `newStockAiAgent`：从 optsOverride[2] 解析 userID，`userID>0` 用 `GetSettingConfigByUserID(userID)`，`userID=0` 用 `GetSettingConfig()`
  - `ChatWithContext` 内部第二次配置查询也改为同样逻辑
  - `ai_handler.go`：`AgentChat` 和 `AITradeAnalyze` 调用时传入 `"", "", userIDStr`
- **编译**：`go build ./web/...` ✅
