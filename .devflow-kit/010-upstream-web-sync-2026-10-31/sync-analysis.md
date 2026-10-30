# 010 上游同步（8bdd55e..796e70d，2026-10-31）

上游区间：`8bdd55e..origin/dev(796e70d)`，共 5 个有效提交（2026-10-24 ~ 10-30）。
执行前基线：dev-web == 009 同步完成态（022c1e3）。

## 上游 5 提交清单

| 提交 | 内容 | 规模 |
|------|------|------|
| 029c68b | feat(okx): 趋势跟随自动调池与全市场永续趋势扫描 | okx_signal_trade +okxTrendScanner 前后端 |
| 2a83f0b | docs: 2026.10 新功能介绍与 OKX 自动交易设置参数详解 | 2 md |
| 59cfc64 | feat(okx): 满仓换仓腾槽与反手开仓分数门槛 | okx_signal_trade + settings 8 字段 |
| 9f7b0f6 | feat: 全市场信号扫描、信号行情列与定制版构建 | signal_algo(+1008)/market_signal_scanner(+380)/custom_build 门控 |
| 796e70d | feat(kline): 统一 K 线分页逻辑并增强图表标题显示 | sina/tdx 分页游标 + 图表 MA250/标题行情 |

合计 43 文件 +4732/-150。本批为历史最大同步批次（全市场扫描 + 趋势跟随调池两大功能）。

## 处理分层

### A. 直接同步（本地零偏移，checkout origin/dev，24 文件 hash 逐一核对全一致）

- 后端：binance_futures_api / bitget_futures_client / okx_signal_regime / okx_signal_trade(+408 方向模式增强+满仓换仓+反手门槛) /
  okx_tpsl_monitor / okx_trade_api(GetOKXCandles +afterMs 游标) / tdx_kline(GetMACKLineDataBefore) / tools.go /
  market_signal_scanner(新+380) / signal_algo(新+1008 全市场买卖点共振算法) / custom_build_on-off 门控对(data+agent/tools) /
  models/signal_trade_record / 根 custom_build_on.go
- 前端：OKXTradingPanel(+363 趋势调池 UI) / SignalMonitorPanel(全市场页签+方向过滤) / aiRecommendStocksList(实时行情列) /
  MarketSignalScanPanel(新+435) / kline/calc.ts(+50 okxUptrendScan) / kline/signalMonitor.ts(+308 趋势跟随集成+tick 看门狗) /
  kline/okxTrendScanner.ts(新+103) / wailsjs/go/models.ts(+80)
- docs：2026.10新功能介绍与使用说明.md / OKX自动交易设置参数详解.md

### B. 手工合并（本地有用户隔离/Web 重写偏离）

1. **backend/data/settings_api.go**：Settings +8 字段（OKXAutoPoolEnable/Live bool、OKXSignalReverseMinScore、
   OKXRiskReservedSlots/HighScore/SwapGap/SwapLossPct/SwapMaxPerDay）；MinScore 注释 9→14；
   UpdateConfig +8 键；**UpdateConfigByUserID 同步补齐 8 字段映射**。
2. **backend/models/signal_record.go**：SignalRecordQuery +Kind 字段（保留 UserID 偏离）。
3. **backend/data/signal_record_api.go**：GetSignalRecordPage +kind 方向过滤（buy/sell，脏值当全部）。
4. **backend/agent/tools/data_tools_wrapper.go**：币安工具注册包 `if !CustomBuild` 门控（本地本就未注册 OKX 工具，仅门控币安）。
5. **backend/data/sina_kline_api.go**：end 分页标记（paged）、fetchFromOKX/MAC +end 参数、分页请求到东财为止早退、
   fetchFromMAC 按 end 走 GetMACKLineDataBefore。保留本地海外指数腾讯降级偏离。
6. **frontend/src/components/settings.vue**：8 新设置项 UI（自动调池双开关在实盘 tag 后、防爆仓 5 项在保证金率后、
   反手门槛在最低信号分数后）+ 冷却 HelpTip 补反手提示 + 4 处 JS 映射（init/readback/submit/import，?? → || 对齐上游）。
   **isCustomBuild 门控包裹与合约行情/币圈代理 custom 门控跳过**——Web 无定制版概念恒 false（沿 007 决策），
   本地保留 openAI.enable 包裹的 Web 偏离结构。
7. **frontend/src/components/kline-analysis.vue**：合约清单加载重构为 loadContractList() + GetVersionInfo custom 门控。
8. **frontend/src/components/uplimitLadder.vue**：checkout 上游版（最新价/涨跌幅列 + 批量行情 quoteMap），
   仅回填本地 import 风格（services/wails-bridge.js 直引）；本地 37 行偏离为纯格式（声明位置），不保留。
9. **frontend/src/components/StockLightweightKlineChart.vue**：MA250 均线（ind/remove/create/setData/dispose 五处）+
   titleQuote 标题行情 computed + 十字线条报价 + 分页修复（barsBeforeVisibleRange 逻辑索引兜底、
   轮询 immediate=true 跳过防抖、loadOlderHistory catch 落日志）。标题 div 用内联样式（本地无 lw-kline-toolbar-name 类）。

### C. Web 端接线（internal/ + services/wails-bridge.js）

1. **internal/handlers/signal_handler.go** +3 端点：
   - POST /signals/market-scan/start（body=MarketSignalScanRequest；回调桥接 BroadcastEvent：
     progress 节流 300ms → SSE `marketSignalScanProgress`，命中 → `marketSignalScanHit`；
     桌面 app.go runtime.EventsEmit 的 Web 等价实现）
   - POST /signals/market-scan/stop；GET /signals/market-scan/status
2. **internal/handlers/stock_handler.go**：GET /stocks/realtime-quotes?codes=（复刻 app.go GetStocksRealtimeQuotes：
   去重保序、A1P/B1P 停牌回退、涨跌幅 3 位舍入——本地无 mathutil 用除法直算）
3. **internal/handlers/okx_handler.go**：GetOKXCandleListHandler 透传 query.afterMs 游标（GetOKXCandles 新第 4 参）
4. **internal/webserver/server.go**：注册 4 条路由
5. **frontend/src/services/wails-bridge.js**：+4 函数（StartMarketSignalScan/StopMarketSignalScan/
   GetMarketSignalScanStatus/GetStocksRealtimeQuotes），wails 分支直调 App、Web 分支走 REST

### D. 跳过（桌面根包 / 生成物，与 006-009 惯例一致）

- app.go(+92: GetStocksRealtimeQuotes/扫描回调 EventsEmit 桥)、app_okx.go(本地无此文件，-tags !custom 门控)、
  app_okx_custom.go(新，-tags custom)、app_signal_monitor.go、main.go——桌面根包历史性编译不过；
  其中共享逻辑（扫描回调→SSE、批量行情）已在 internal/ 等价实现（C-1/C-2）
- frontend/src/App.vue（+6：SignalMonitorPanel 的 isCustomBuild 门控注释变化——桌面菜单）
- frontend/wailsjs/go/main/App.js / App.d.ts——Wails 生成物，Web 走 bridge（models.ts 例外已随 A 类同步）

## 验证基线（全部通过）

- A 类 hash 核对：24/24 与 origin/dev 一致
- go build ./backend/... ./web ./internal/... ./ai-assistant-web/...：全绿
- go vet 同范围：全绿
- go test backend/data（TestOkxSignal*/TestOKXCalcSize/TestOKXSignalBarTimeWindows 等）+ agent/tools：全绿
  （本批上游未附 signal_algo/market_signal_scanner 测试文件）
- vite build：通过（4m42s，outDir tmp-build-010，验证后清理）

## 环境事故留档

- 010 执行中发现 **Go 模块缓存解压损坏**（gobwas/ws 缺 dialer_tls_go18.go 等 6 文件、resty/jwt 同批损坏）：
  症状为 `go build` 通过但 `go vet`/`go test` 报第三方包 undefined 符号（差异：损坏包仅测试链路引入）。
  修复：定向删除 GOMODCACHE 对应模块目录 + download 缓存后 `go mod download` 重取；
  若重取后仍失败需 `go clean -cache`（构建缓存残留坏元数据会让 go list 看不见已恢复的文件）。

## 同步后状态

- 同步前沿：`796e70d`
- 提交链：bad247d(B1 后端) → 38355e0(B2 接线) → 7073f9e(B3 前端纯量) → d13f649(B4 手工合并) → 本记录
- 新增 Web 能力：全市场信号扫描（面板内嵌 SignalMonitorPanel 页签）、OKX 趋势跟随自动调池（双重总闸）、
  批量实时行情端点、K 线向左拖动分页（MAC 深取 + OKX after 游标）
