# 008 上游同步（88bbea5..2872d65，2026-10-24）

上游区间：`88bbea5..origin/dev(2872d65)`，共 6 个有效提交（v2026.10.18.2 ~ v2026.10.22.1，2026-10-18 ~ 10-22）。
执行前基线：dev-web == 007 同步完成态（fa6eca5）。

## 上游 6 提交清单

| 提交 | 内容 | 规模 |
|------|------|------|
| d0e3350 | feat(okx): 开仓附带交易所侧TPSL条件单并清理孤儿单；AI推荐股页定时扫描推送 | 6 文件 +478 |
| 02a3691 | feat(okx): 信号交易方向模式/大势门控/单日限笔数；广场默认域名改自建反代 | 8 文件 +1025 |
| 3a92553 | refactor(plaza): 前端移除广场与支付页域名硬编码，统一由后端配置下发 | 6 文件 +18 |
| 2a961a8 | feat(agent): 集成 DeepSeek 联网搜索/网页抓取 API | 7 文件 +510 |
| f49a730 | feat(okx): 信号金额改名义价值口径；应用内更新一键重启 | 19 文件 +398 |
| 2872d65 | feat(okx): K线买卖点/TPSL标注、涨跌颜色习惯切换、列表滚动分页 | 10 文件 +723 |

合计 39 文件 +3124/-334。go.mod 无变化，lightweight-charts 双方均 ^5.2.0。

## 处理分层

### A. 直接同步（本地零偏移或纯新增，checkout origin/dev / 直接 add）

**后端 Go（11 文件）**——OKX 数据层全部零偏移（007 全量接收的延续）：
- backend/agent/agent.go（+5，web_search 工具注册）
- backend/agent/tools/tool_groups.go（+4）
- backend/agent/web_search_tool.go（新 251，DeepSeek 联网搜索/抓取工具）
- backend/data/deepseek_search_api.go（新 210）⚠️ **git add -f**
- backend/data/okx_signal_regime.go（新 131，BTC 1H 大势门控）⚠️ **git add -f**
- backend/data/okx_signal_trade.go（+250，方向模式/限笔数/名义价值/TPSL 附单）
- backend/data/okx_signal_trade_test.go（新 30）⚠️ **git add -f**
- backend/data/okx_sim_flow_test.go（+74）
- backend/data/okx_tpsl_attach_test.go（新 84）⚠️ **git add -f**
- backend/data/okx_tpsl_monitor.go（+374，交易所侧 TPSL 附单与孤儿单清理）
- backend/data/okx_trade_api.go（+79，GetOkxInstruments 等）
- backend/data/okx_trade_api_test.go（+6）
- backend/data/proxy_acceleration.go（+3）

**前端（3 文件）**——vite alias 自动把 `wailsjs/go/main/App` 重定向到 utils/wails-bridge.js：
- frontend/src/components/OKXTradingPanel.vue（+367，K线 createSeriesMarkers 买卖点/TPSL 标注、名义价值口径；新 import GetOKXPositionTPSLEnv）
- frontend/src/components/aiRecommendStocksList.vue（+86，信号自动扫描定时推送 signalAuto/signalAutoMin）
- frontend/src/components/kline/signalMonitor.ts（+33，扫描间隔 5/10s 档位、信号流水滚动分页 loadMoreSignals/append）

**文档**：docs/5m-15m-auto-trade-backtest.md（新 462）

### B. 手工合并（本地有用户隔离/Web 重写偏离）

1. **backend/data/settings_api.go**（上游 +64，本地 529 行偏离 = userID 隔离 + 007 OKX 字段）：
   - Settings struct +4 字段：OKXSignalTradeDirMode / OKXSignalTradeRegime / OKXSignalTradeMaxPerDay / DeepSeekSearchApiKey
   - DefaultPromptPlazaApiBase 值改 `https://proxy.sparkmemory.top/api`；Fallback 常量重构为 `PromptPlazaApiFallbacks` 列表；PromptPlazaApiBases() 重排为「默认地址优先 + primary 追加 + fallbacks 兜底 + lo.Uniq」（替换本地 007 版主键回退实现）
   - GetSignalScanIntervalSec 下限 30→5（注释同步）
   - UpdateConfig map +4 键（dir_mode/regime/max_per_day/deepseek_search_api_key）
   - **本地独有 UpdateConfigByUserID 必须同步补齐这 4 个字段映射**
2. **frontend/src/components/settings.vue**（上游 +279，本地 1074 行偏离 = Web 表单结构）：
   - SIGNAL_PRESETS 7 组回测预设（h1_flat 等）+ presetKey/applyPreset
   - 方向模式/大势门控/单日限笔数 3 个设置项 UI；「每笔金额」label 换名义价值口径 HelpTip
   - DeepSeek 搜索密钥输入项（含 platform.deepseek.com 链接）
   - promptPlazaApiBase 初始值/回读默认 → 空串；placeholder 同步
   - HelpTip defineComponent 局部组件引入
3. **frontend/src/components/SignalMonitorPanel.vue**（上游 +242，本地仅 1 行 import 偏离）：
   策略 = checkout origin/dev 版本 + 恢复本地 import 行，并把 GetOkxInstruments 加入 bridge import。
4. **frontend/src/components/kline-analysis.vue**（上游 +51，本地 361 偏离 = bridge 重接线）：
   ok: 前缀三件套——isContractCode/resolveContractName/toEastMoneyCode 加 ok: 分支、联想扁平化匹配（-SWAP 去连字符）、GetOkxInstruments 加载合约清单进 contractList。
5. **frontend/src/components/StockLightweightKlineChart.vue**（上游 +31，本地 2402 偏离 = Web 重写版）：
   isOKXCode computed + 复权恒 none + 轮询 5s + 无涨跌停 + OKX 代理不可达提示，逐点合入 Web 版。
6. **frontend/src/components/about.vue**（上游 +10，本地 146 偏离 = 007 支付域名）：
   payUrl = GetConfig().promptPlazaApiBase 去 /api 拼 /pay，v-if 门控渲染，替换本地硬编码 stock.sparkmemory.top/pay 两处。
7. **广场 4 页域名去硬编码**（上游 3a92553，各 2 行）：
   promptPlaza.vue / promptQa.vue / skillPlaza.vue / promptTemplateList.vue 的 apiBase 初始值 → 空串。
   注意：Web 版这 4 处还是更旧的 `http://go-stock.sparkmemory.top:1918/api` 系硬编码，本批顺带清掉。
8. **frontend/wailsjs/go/models.ts**（上游 +48，本地 151 偏离）：纯增量追加 OKXPositionTPSLView / OKXInstrument 等类型类（该文件被 agent-chat 等 Web 组件直接 import，vite 直译）。
9. **.gitignore**：合入上游新增（.workbuddy / frontend/tmp-tune / /go-stock / data/settings_backup_*.json），保留本地全部。
   **治理项（007 遗留教训）：`data/` → `/data/`**——根锚定后不再吞 backend/data/ 新文件，本批起 4 个新文件无需 -f；logs/dist 保持不动（未出过事故）。

### C. Web 端接线（internal/ + services/wails-bridge.js）

1. **internal/handlers/okx_handler.go** 新增 2 端点 + 1 处放宽：
   - GET /okx/instruments → data.GetOkxInstruments()（SignalMonitorPanel/kline-analysis 合约联想用）
   - GET /okx/positions/tpsl?simulated= → data.GetOKXPositionTPSLEnv（OKXTradingPanel TPSL 标注用）
   - SetSignalScanIntervalHandler 校验 `sec < 30` → `sec < 5`（对应 settings_api 放宽；event_handler RestartSignalTickInterval 无独立校验，构建后验证）
   - webserver 路由注册 2 条
2. **frontend/src/services/wails-bridge.js** +2 函数：GetOkxInstruments / GetOKXPositionTPSLEnv（bridge 现有 24 个 OKX 函数，缺这 2 个）。

### D. 跳过（桌面根包 main / 桌面专属功能，与 006/007 惯例一致）

- app.go（+52：CheckUpdate 的 newsPush→updateApplied 事件、buildRestartArgs）/ main.go（+18：-wait-parent 等待旧进程退出）/ app_okx.go（+20，GetOkxInstruments/TPSL Env 绑定壳）/ app_signal_monitor.go（+8，扫描下限 30→5 注释与校验）/ update_helper_darwin/linux/windows.go——**应用内更新一键重启整链路**，Web 无自更新与单实例锁概念；其中 GetOkxInstruments/GetOKXPositionTPSLEnv 共享逻辑在 backend/data 已随 A 类进来，internal 接线（C 类）即等价实现。扫描间隔放宽的等价校验在 C 类 okx_handler 完成。
- frontend/src/App.vue（+29：updateApplied 事件对话框 + RestartApp）——Web 版无自动更新安装流程，newsPush 通用监听不受影响。
- frontend/wailsjs/go/main/App.js / App.d.ts——wails 生成物，Web 走 bridge；仅 RestartApp 绑定为桌面专属。models.ts 例外已列入 B-8。

## 提交批次（建议）

| 批次 | 内容 | 验证 |
|------|------|------|
| B1 | A 类后端 13 文件 + docs + .gitignore 治理 | go build/vet 全绿 + data 定向测试 |
| B2 | A 类前端 3 文件 + SignalMonitorPanel(checkout+import 修正) + models.ts | vite build |
| B3 | 手工合并：settings_api.go + UpdateConfigByUserID + settings.vue + 广场 4 页 + about.vue + kline-analysis + StockLightweightKlineChart | go build + vite build |
| B4 | C 类接线：okx_handler 2 端点 + 校验放宽 + webserver 路由 + bridge 2 函数 | go build/vet + vite build + smoke |

## 已知风险与回归口径

- TSPL/交易所侧条件单逻辑全在 A 类文件，风险低；名义价值口径改了下单金额换算（okx_signal_trade.go），TestOKXCalcSizeByAmount 旧断言上游已同步改，测试直接跟上游。
- 广场默认域名切 `proxy.sparkmemory.top` 自建反代——若该域名不可达，Fallbacks 仍含 go-stock/stock.sparkmemory.top 兜底，行为不劣于现状。
- settings.vue 手工合并量最大（279 上游行 vs Web 表单结构差异），按功能块逐块合入（预设组/三设置项/DeepSeek/广场空串化）。
- .gitignore 治理仅动 `data/` 一行；若改后 `git status` 出现根 data/ 未忽略噪音，说明运行时目录存在，验证时确认。

## 验证基线（沿 007）

- go build ./web ./internal ./backend ./ai-assistant-web
- go vet 同范围
- go test backend/agent backend/agent/tools + backend/data 定向（OKXSim/Signal/TPSL attach 新测试）
- vite build
- 目标文件 HEAD 与 origin/dev hash 逐一致核对（A 类清单）

## 执行后补记（2026-10-24）

### 实际提交链（批次顺序有调整：C 类接线提前到前端纯量之前，避免 bridge 缺函数导致中间态构建失败）

| 提交 | 内容 |
|------|------|
| ef1c583 | B1：后端 13 文件直接同步 + settings_api.go 手工合并（含 UpdateConfigByUserID 补 4 字段）+ .gitignore 治理 + wails_events 补提交 + 回测文档 |
| 3146566 | B2（=方案 C 类提前）：okx_handler 2 新端点（/okx/instruments、/okx/positions/tpsl）+ scan-interval 5s 放宽 + webserver 路由 + bridge 2 函数 |
| d466ead | B3：前端直接同步 5 文件（OKXTradingPanel/SignalMonitorPanel/aiRecommendStocksList/signalMonitor.ts/models.ts） |
| 07bfad8 | B4：前端手工合并 8 文件（settings.vue OKX 区整块换上游版/广场 4 页域名下发/about.vue payUrl/kline-analysis 合约联想块/StockLightweightKlineChart ok: 支持） |

### 执行中的计划外发现与处置

1. **wails_events.go/test 从未入库**（007 事故漏网）：已提交代码 openai_stream.go 等 3 处调用 SafeEventsEmit，
   仅存在于未跟踪文件中——fresh clone 无法编译。随 B1 补提交修复。
2. **kline-analysis.vue Web 重写版缺失整个合约联想体系**（bn:/bt: 也没接过）：上游 ok: 建立在其上，
   本批整块移植（合约清单加载 + 前缀识别 + 联想匹配），顺带补齐历史欠账。
3. **SignalMonitorPanel.vue 无需手工改 import**：vite alias 已把 ../../wailsjs/go/main/App 重定向到
   utils/wails-bridge.js（re-export services），checkout 上游版即可。
4. **广场 4 页 Web 版分化处理**：skillPlaza/promptTemplateList 走 PromptPlazaRequest（后端代理处理空串）
   → 完全对齐上游改空串；promptPlaza/promptQa 是裸 fetch 重写版，空串会在首屏竞态时抛 URL 异常
   → 增加 resolvedApiBase() 兜底（默认反代 https://proxy.sparkmemory.top/api）。
5. **StockLightweightKlineChart 轮询间隔不合并**：Web 版由父组件 realtimeIntervalMs prop 控制（默认 60s），
   与上游按代码类型计算 CONTRACT_POLL_INTERVAL_MS 的架构不同；只合并 isOKXCode 系列 5 处。
6. **settings.vue 保留两处 Web 偏离**：OKX 区 template v-if 包裹（上游逐项 v-if/无包裹）；广场地址输入框
   保持可编辑（上游 disabled 固定值），placeholder 与帮助文案去硬编码。

### 验证基线（全部通过）

- hash 一致核对：A 类 19 文件 HEAD vs origin/dev 全部一致（19/19）
- go build/vet ./web/... ./internal/... ./backend/... ./ai-assistant-web/... 全绿
- go test backend/agent + backend/agent/tools 全绿（含 DeepSeek 工具注册）
- data 定向测试：新上游用例（TestOkxAlignPx/ComputeTPSL×4、TestOKXSignalBarTimeWindows、TestOKXCalcSizeByAmount 等）全过；
  仅 007 已知口径失败（TestOKXLiveProbe/PositionCheck 实盘探测无凭证、TestWebSearchApi* chromedp 无本机 Chrome）
- vite build 通过两次（B3 后 5m34s、B4 后 1m09s）

### 已知测试口径（非回归，沿 007）

- TestOKXLivePositionCheck / TestOKXLiveProbe：实盘探测，无凭证 FAIL 为设计行为
- TestWebSearchApi_SearchToMarkdown/Json：chromedp 需本机 Chrome（C:\Program Files (x86)\...），环境缺失失败

### 遗留（下批处理）

- stockhotmap.vue 的 :16667 嵌入地址硬编码——上游未改，不在本批范围
- frontend/wailsjs/go/main/App.js / App.d.ts 仍为旧生成物（Web 不消费，RestartApp 等桌面绑定无需同步）
