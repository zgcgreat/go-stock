# 004 · 信号监控 Web 化立项（Web 节拍 + 面板挂载点 + 持久化端点）

> 状态：**已立项待执行**（2026-09-27）｜ 前置：003 §7「剩 5 个信号桩维持暂缓」的正式解锁设计 ｜ 来源：003 收尾后用户指令
>
> ## 结论速览
>
> - **解桩范围实际是 8 个，不是 5 个**：5 个 reject 桩（Save/GetPage/GetStats/Clear/Notify）之外，还有 3 个「web-dead」函数
>   （`IsTradingTime`/`IsHKTradingTime`/`IsUSTradingTime` 在 Web 模式硬编码 `false`）——后者更致命：不报错、静默让
>   `anyMarketOpen()` 恒 false，**即使解锁 5 个桩，一轮扫描也不会执行**。批 1 必须先行。
> - **架构上不移植算法**：信号算法保持前端 TS 单实现（上游哲学，见 §1），Web 化 = 「浏览器里的引擎 + 服务端节拍推送 + 用户隔离的落库端点」，
>   与桌面唯一的本质差异是节拍载体（Wails 事件 → SSE 事件总线）。
> - **多用户隔离是本项目的真工作量所在**：`signal_record` 表无 user_id、`ClearSignalRecords` 是全表删除（多用户下事故级）、
>   唯一索引不含 user_id。模型改造 + 迁移占批 2 大头。
> - **5 批实施**：批1 时间窗提取 → 批2 持久化端点+隔离 → 批3 SSE 事件总线+节拍 → 批4 面板挂载+推送通道 → 批5 文档收尾。

## 1. 背景与现状勘察

### 1.1 上游架构（为什么不能按常规桩解锁思路做）

上游把买卖点信号监控设计成「前端算法 + 后端通道」：

- **算法单实现**：买卖点/TEMA 转折算法只有一份 TypeScript（`kline/calc.ts` 的 `buySellPointsValues`/`temaTurnPointsValues`），
  `backend/models/signal_record.go` 头注释明确「移植到 Go 会造成永久双份维护，并与图上箭头逐根漂移」——
  **本项目同样不得移植算法到 Go**。
- **引擎**：`kline/signalMonitor.ts` 应用级单例（reactive state + 基线 Map），职责链为
  节拍唤醒 → 交易时段判断 → 池×周期任务轮转（并发 4 / 预算 25s / 单轮 120 任务）→ 取 K 线（`GetStockKLineWithFallback`，
  已桥接）→ 算法 → 基线比对出新信号 → 落库 + 提醒（音效/语音/推送）。
- **节拍**：桌面 Go cron 每 60s `runtime.EventsEmit("signalMonitorTick")`（`app.go:958-966` + `app_signal_monitor.go:23`），
  因为窗口最小化时 Chromium 会把前端 setInterval 深度节流。前端另留兜底定时器（`GO_TICK_STALE_MS=150s` 未收到 Go 节拍则自跑）。
- **持久化分工**：信号流水进 SQLite（`signal_record` 表）；**监控池/灵敏度/渠道等设置在 localStorage**（`PERSIST_KEY='signal-monitor-settings'`）。
- **VIP 门槛**：VIP2+，`startSignalMonitor` 无权限时连事件监听都不注册。Web 模式的 `GetEffectiveSponsorVip` 已桥接
  （`/user/profile`，admin=VIP2，wails-bridge.js:961），门槛逻辑直接复用。

### 1.2 依赖面盘点（引擎 import 的 10 个 App 方法在 Web 的现状）

| 依赖 | Web 现状 | 处置 |
|---|---|---|
| `GetStockKLineWithFallback` | ✅ 已桥接（K线图在用） | 无需动 |
| `GetEffectiveSponsorVip` | ✅ 已桥接 | 无需动 |
| `GetStockList` / `GetConfig` | ✅ 已桥接（面板选股/主题用） | 无需动 |
| `SaveSignalRecords` | ❌ reject 桩（:3452） | 批2 解锁 |
| `GetSignalRecordPage` | ❌ reject 桩（:3319） | 批2 解锁 |
| `GetSignalStats` | ❌ reject 桩（:3325） | 批2 解锁 |
| `ClearSignalRecords` | ❌ reject 桩（:3047） | 批2 解锁 |
| `NotifySignal` | ❌ reject 桩（:3386） | 批4 解锁 |
| `IsTradingTime` | ⚠️ **web-dead**：`Promise.resolve(false)`（:2641） | 批1 修复 |
| `IsHKTradingTime` / `IsUSTradingTime` | ⚠️ 同上（:2646/:2651） | 批1 修复 |
| `EventsOn('signalMonitorTick')` | CustomEvent 实现（:1000），**无人派发** → 兜底定时器接管，节拍退化 | 批3 修复 |

注：`signalMonitor.ts` 从 `wailsjs/go/main/App` import，构建层 vite 别名重写到 wails-bridge.js（003 发现的机制），改桥接即改 Web 行为。

### 1.3 面板与挂载现状

- `SignalMonitorPanel.vue`（1032 行）**上游也是孤儿**：grep 全前端无任何 import（003 §7 已记录「上游无人挂载」）。
  组件自带入口与壳：右侧 edge-trigger（fixed，`top: calc(50% + 130px)`，z-index 9997）+ 抽屉（常驻渲染不销毁，
  与 AI 助手抽屉同款），设计意图就是挂布局根级（`signalMonitor.ts:794` 注释「面板由 App.vue 常驻挂载」，实际没挂）。
- naive-ui 技术栈，与 WebLayout 一致（无 agent-chat 那种 TDesign 适配成本）。
- 面板 `onMounted` 调 `startSignalMonitor()`（幂等，started 标志）——挂载点必须常驻，不能随路由销毁重建。

### 1.4 多用户隔离缺口（本项目的核心工作量）

| # | 缺口 | 后果 |
|---|---|---|
| 1 | `signal_record` 表无 `user_id` | 流水跨用户混看，他人信号进入我的流水与统计 |
| 2 | `ClearSignalRecords` 用 `AllowGlobalUpdate` 全表删（`signal_record_api.go:85`） | 任何用户点「清空」删光**所有用户**的流水——事故级 |
| 3 | 唯一索引 `(code,klt,family,kind,bar_time)` 不含 user_id | 多用户监控同一票时，同根 K 线只落一条，第二个用户的信号丢失 |
| 4 | 交易时段逻辑在 package main（`app.go:1392-1640` 的 `tradingDayCache`/`isTradingTime`/`isTradingDay`/`IsHKTradingTime`/`isHKTradingDay`/`checkHKHolidayAPI`/`IsUSTradingTime`/`isUSTradingDay` + `app_common.go:25` `ShanghaiTimezone`），internal 包引用不到 | Web 端点无法复用；`internal/webserver/scheduler.go:321` 已被迫复制了一份无节假日校验的 A 股副本 |
| 5 | `NotifySignal` 的飞书/钉钉走全局配置（`data.NewFeishuAPI()`） | Web 沿用全局配置（与桌面同源），per-user webhook 明确不做（§6） |

隔离惯例沿用既有先例：`userID > 0 则 Where user_id 过滤，0 不过滤（桌面语义）`（`ImportTradingRecords` 已开此例）；
处理器用 `getUserID(c)` 注入（trade_handler.go:17）。

### 1.5 可复用的既有模式

- **fetch 流式 SSE**：`ai_handler.go:86/:342` 两个 SSE 处理器在案——`EventSource` 不能带 Authorization 头，Web 事件通道用 fetch-stream 读。
- **webserver 后台 goroutine + ticker 惯例**：`scheduler.go:150-248`（市场统计 5min / 板块资金 1min / 热词 30min，均带 `isTradingTime` 门控与 stopChan）。
- **web 迁移注册点**：`internal/migrations/web_migrations.go` 的 `migrateUserIDColumn`（幂等 pragma 检查）——signal_record 加列直接进清单；
  唯一索引重建需新写专用函数（drop 旧索引 → 建新索引，见 §3.1）。
- **响应约定**：`{code, data, message}` 包裹（UploadImageHandler 先例）；路由挂 `protected` 组（JWT）。

## 2. 方案设计

### 2.1 Web 节拍：SSE 事件总线（方案 B）

三案对比：

| 方案 | 做法 | 判定 |
|---|---|---|
| A. 纯前端兜底定时器 | 不做任何事（现状退化态，150s 后兜底自跑） | ❌ 标签页隐藏时浏览器深度节流（intensive throttling 1/min，极端时冻结），信号延迟/丢失，违背「窗口最小化也会持续提醒」的产品承诺 |
| **B. SSE 事件总线** | 服务端 60s tick 经 SSE 推给在线连接，前端 `EventsOn('signalMonitorTick')` 收到即扫描 | ✅ **桌面「Go 节拍」的真正等价物**：网络事件回调不受定时器节流影响；且是通用机制——003 已记录 `promptBacktestProgress` 桌面事件同样到不了 Web，一次投入两处受益 |
| C. 服务端移植算法 | Go 侧重算信号 | ❌ 违背上游单实现哲学（与上游同一否决理由，§1.1） |

**B 方案细化**：

- 端点：`GET /api/v1/events/stream`（protected 组，JWT；fetch-stream 读，token 走 Authorization 头不进 URL）。
- 服务端：连接注册表（`sync.Map[connID]chan []byte`）+ hub goroutine 60s ticker 广播 `event: signalMonitorTick\ndata: {}\n\n`；
  写失败/`c.Request.Context().Done()` 即摘除连接。**服务端保持 dumb pipe**——不判交易时段、不看用户配置，
  门控全在引擎侧（与桌面一致：「Go 侧不做业务判断」，`app_signal_monitor.go:21`）。tick 对空连接列表零成本，无需启停。
- 前端桥接：wails-bridge.js 的 web 模式 `EventsOn` 升级为共享事件流订阅器——首个监听注册时建立 fetch-stream 连接
  （断线指数退避 + 抖动重连），按事件名 `dispatchEvent(new CustomEvent('web-event-<name>'))` 落回现有 CustomEvent 通路，
  **下游引擎代码零改动**；最后一个监听注销时断开（引用计数）。`EventsEmit`（客户端→客户端）行为不变。
- 兜底不变：引擎的 `GO_TICK_STALE_MS` 定时器保留——SSE 断线时自动退化到方案 A，节拍可靠性三层兜底
  （SSE 主通道 → 前端定时器 → 手动「立即扫描」按钮）。
- 已知边界：同一用户开多个标签页时各标签页独立扫描，唯一索引保证不重复落库，但**通知可能双发**（记录为已知边界，不做去重协调）。

**产品语义对齐**（写进用户可见文案）：桌面「关闭窗口即停止」↔ Web「关闭标签页即停止」；「窗口最小化仍提醒」↔「标签页切后台仍提醒（SSE 唤醒）」。

### 2.2 交易时段死桩修复：时间窗逻辑提取（批 1，节拍的前置）

- **纯搬运不改写**：`app.go:1392-1640` + `app_common.go:203-238` 的 8 个函数与 2 个变量
  （`tradingDayCache`、`ShanghaiTimezone`、`isTradingTime`、`isTradingDay`、`IsHKTradingTime`、`isHKTradingDay`、
  `checkHKHolidayAPI`、`IsUSTradingTime`、`isUSTradingDay`、`IsTradingDay` 核心逻辑）→ 新文件 `backend/data/market_hours.go`（package data）。
  `checkHKHolidayAPI` 已用 `data.SharedHTTPClient`，搬到 data 包后引用反而变直。
- 桌面 `App.IsTradingTime/IsHKTradingTime/IsUSTradingTime/IsTradingDay` 包装改调 data 包（**行为零变化**，go build 门禁兜底）。
- 新端点 `GET /market/trading-time?market=a|hk|us` → `{code:0, data:{trading:bool}}`（新 `market_handler.go` 或并入 signal_handler）。
- 桥接三个函数 web 分支改调该端点（引擎内本就 `Promise.all` 并行三市场，延迟无感）。
- 顺带收敛：`scheduler.go:321` 的 A 股副本改调 `data.IsTradingTimeNow()`（副本无节假日校验，收敛是净收益；可选，若冲突大则留 TODO）。

### 2.3 持久化端点：5 桩解锁 + 隔离改造（批 2）

**模型改造（唯一动上游模型处）**：

- `models.SignalRecord` 加 `UserID uint`（`gorm:"index"`；json tag 加 `json:"-"` 防客户端伪造，落库时由服务端注入）。
- 唯一索引重建为 `(user_id, code, klt, family, kind, bar_time)`。**迁移注意**：GORM AutoMigrate 不会重建既有唯一索引，
  需专用迁移：`DROP INDEX IF EXISTS idx_signal_unique` → AutoMigrate/`CREATE UNIQUE INDEX idx_signal_unique_user`。
  桌面库同样受益（存量数据 user_id=0，桌面语义不变）。
- `web_migrations.go` 加 `migrateSignalRecordIsolation()`（加列复用 `migrateUserIDColumn` 思路 + 索引重建专用逻辑，全部幂等）。

**服务层**（`signal_record_api.go` 全部方法加 `userID uint` 参数）：

- `SaveSignalRecords(items, userID)`：落库前统一注入 `UserID`（不信任请求体）；`clause.OnConflict` 列加 `user_id`。
- `GetSignalRecordPage(query, userID)`：`userID > 0 → Where("user_id = ?", userID)`。
- `ClearSignalRecords(userID)`：**必须 Where user_id**，消灭全局删除。
- `GetSignalStats(query, userID)`：同上（配对统计口径按用户隔离后天然正确）。
- 桌面调用点（app_signal_monitor.go 4 处）传 0——行为不变。

**端点**（新 `internal/handlers/signal_handler.go`，server.go protected 组注册）：

| 端点 | 方法 | 桩对应 | 说明 |
|---|---|---|---|
| `/signals/records` | POST | `SaveSignalRecords` | body: `[{code,klt,family,kind,time,name,price,score,detail,at}]`，返回 `{code:0}` 或错误信息字符串语义与桌面一致 |
| `/signals/records` | GET | `GetSignalRecordPage` | query 绑定 `models.SignalRecordQuery` |
| `/signals/records` | DELETE | `ClearSignalRecords` | 仅清当前用户 |
| `/signals/stats` | GET | `GetSignalStats` | query 绑定 `models.SignalStatQuery` |
| `/signals/notify` | POST | `NotifySignal` | 见 §2.5（批 4 实现，端点可批 2 先占位） |

权限：全部 JWT（protected 组）。**VIP2 服务端强制列入可选加固**（上游同语义是客户端门控；服务端可用 `middleware/role.go` 的 RoleVIP，
但 web 赞助 VIP 与 role 体系不同源，MVP 不混两套体系，仅记录）。

**设置/监控池持久化决策**：MVP 保持 localStorage（与上游一致、零后端）。理由：引擎只在标签页打开时采集，
池随浏览器是自洽语义；跨设备同步池属新特性，不做（§6）。信号流水照桌面先例进 SQLite（唯一服务端持久化面）。

### 2.4 面板挂载点（批 4）

- `WebLayout.vue` 根级常驻挂载 `<SignalMonitorPanel />`（header 与 router-view 同级、template 末尾）——
  面板自带 edge-trigger 入口与抽屉，**无需新增路由/菜单项**（与「更多」抽屉类页面不同，它是全局悬浮入口）。
- 同时修复了上游自身的孤儿状态；桌面侧若上游将来挂载，merge 时天然对齐（dev-web 侧不改桌面 App.vue）。
- 挂载约束（写进代码注释防回归）：**只准挂 WebLayout 一处**——引擎 state 是模块级单例，多挂会串状态
  （startSignalMonitor 虽幂等，但两个面板实例的 UI 状态各自为政）。
- z-index 已核对：edge-trigger 9997 / 抽屉 9999 / 弹窗 10010，均高于 WebLayout 常规内容，无冲突。
- VIP 无权限路径已在组件内闭环（togglePanel/onToggleMonitor/scanNow 三处 ensureVip），Web 的
  `GetEffectiveSponsorVip` 已桥接，无需改。

### 2.5 NotifySignal 的 Web 语义（批 4）

渠道矩阵（引擎把 `channels` 原样传给后端）：

| 渠道 | 桌面 | Web |
|---|---|---|
| `app` | 系统通知（Wails Notification）+ `newsPush` 事件横幅 | 浏览器 **Notification API**（桥接 web 分支内实现：权限请求 → new Notification；被拒则降级仅音效）；横幅经事件总线推 `web-event-newsPush`（无消费者时无害） |
| `feishu` | `data.NewFeishuAPI().SendToFeishu` | 服务端 `/signals/notify` 转发（全局配置，与桌面同源） |
| `dingding` | `data.NewDingDingAPI().SendToDingDing` | 同上 |

- 音效/语音提醒是纯前端（`alertSound.ts`，Web Audio + speechSynthesis），Web 直接可用——按钮交互手势预热逻辑已内置（pointerdown/keydown 监听）。
- 请求体：`{title, content, plain, channels}`；服务端对 feishu/dingding 做渠道分发，app 渠道服务端忽略（由前端 Notification 承担）。

## 3. 实施批次（每批独立 commit + 三重门禁 `go build ./...` / `go vet ./internal/... ./backend/...` / `npm run build`）

| 批 | 内容 | 交付物 | 验证 |
|---|---|---|---|
| **1** | 时间窗提取 + trading-time 端点 | `backend/data/market_hours.go`（纯搬运）、App 包装改调、`GET /market/trading-time`、桥接 3 函数复活、（可选）scheduler 副本收敛 | curl 三市场真假值（盘中/盘后各测一次）；桌面 App 方法回归（编译 + 冒烟调用） |
| **2** | 持久化 5 端点 + user_id 隔离 | 模型加列+索引迁移、服务层 4 方法加参、`signal_handler.go`、5 桩中 4 个替换（notify 留批 4）、web_migrations 注册 | curl 冒烟：落库→去重幂等→分页→双用户互不可见→清空仅本人→统计配对正确 |
| **3** | SSE 事件总线 + 节拍 | `GET /events/stream` + hub、桥接 EventsOn 升级（fetch-stream + 重连） | 浏览器：标签页切后台/最小化窗口，服务端日志见 tick 推送、引擎 `lastTickAt` 每 60s 前进；断网重连恢复 |
| **4** | 面板挂载 + notify 通道 | WebLayout 挂载、NotifySignal web 分支（Notification API）、`/signals/notify` 服务端转发、最后 1 桩替换 | e2e 全链：登录 VIP2 → edge-trigger 开面板 → 加监控池 → 立即扫描 → 流水落库 → 统计 → 清空；通知权限弹窗与降级路径 |
| **5** | 004 文档同步收尾 | §4 验证矩阵、遗留记录 | — |

批次依赖：批 1 是批 3/4 e2e 的硬前置（否则扫描永不执行）；批 2 与批 1 无依赖可并行（保持线性则按序）；批 3 依赖批 1（验证时需要扫描真的跑）；批 4 依赖 1/2/3。

## 4. 风险登记

| 风险 | 等级 | 缓解 |
|---|---|---|
| 唯一索引迁移在存量库上失败（旧索引存在/结构差异） | 中 | 迁移前 DB 备份（P2 惯例）；`DROP INDEX IF EXISTS` 幂等；迁移函数先 pragma 检查再动 |
| SSE 连接泄漏（异常断开未摘除） | 中 | `c.Request.Context().Done()` 监听 + 写超时 + 发送失败即删；上线后观察连接数 |
| 重连风暴（服务端重启全体重连） | 低 | 指数退避 + 随机抖动；前端兜底定时器保证功能连续 |
| 多标签页通知双发 | 低 | 已知边界，唯一索引兜住落库不重复；不做跨标签页协调 |
| Notification 权限被拒 | 低 | 降级仅音效+横幅，面板提示授权路径 |
| 桌面回归（时间窗搬运/服务层加参） | 中 | 纯搬运不改逻辑；App 包装传 0；三重门禁 + 桌面冒烟点验交易时段判断 |
| 引擎单例被重复挂载 | 低 | 挂载约束写入两处注释（WebLayout + 组件头） |
| VIP 客户端门控被绕过（直调 API） | 低 | 数据按 user_id 隔离，绕过者只能污染自己；服务端 VIP 强制列为可选加固（§2.3） |

## 5. 明确不做（范围控制）

1. **不移植算法到 Go**（上游哲学：单份 TS 实现，防箭头/流水漂移）。
2. **不做 per-user 飞书/钉钉 webhook**（沿用全局配置；per-user 属新特性另立项）。
3. **不做监控池/设置的跨设备同步**（保持 localStorage）。
4. **不做服务端代跑扫描**（无标签页不采集；与桌面「关窗口即停」语义对齐而非背叛）。
5. **不做跨标签页通知去重协调**（唯一索引已兜住数据面，通知面接受双发）。
6. **不做 VIP2 的服务端强制**（MVP 客户端门控与上游同语义；服务端加固单列可选）。

## 6. 与 003 的关系

- 003 §7「剩 5 个信号桩维持暂缓」的解除以本项目为唯一通道；批 5 收尾时回写 003 §7 状态（42 桩 → 42/42）。
- SSE 事件总线同时是 003 记录的「promptBacktestProgress 桌面事件不到 Web」遗留的候选解法（本项目只保证
  `signalMonitorTick` 消费到位，回测进度事件的接入不在本项目范围内）。
