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
