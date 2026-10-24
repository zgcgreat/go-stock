# 009 上游同步（2872d65..8bdd55e，2026-10-24）

上游区间：`2872d65..origin/dev(8bdd55e)`，共 4 个提交（2026-10-24 当天）。
执行前基线：dev-web == 008 同步完成态（48baa24）。
**特殊情况**：上次会话已完成绝大部分移植（工作区暂存 11 文件 + 未暂存 5 文件手工合并），
本批任务 = 覆盖完整性核验 → 分层提交 → 验证 → 欠账登记。

## 上游 4 提交清单

| 提交 | 内容 | 规模 |
|------|------|------|
| cdf4a07 | fix(update): 应用内更新「立即重启」不生效——旧进程退出前 WAL checkpoint 收尾 | 7 文件 +70 |
| 35e53bd | feat(okx): 信号自动交易新增「开单最低止盈收益率」门槛 | 5 文件 +102 |
| 2a69dd7 | feat(okx): 开单止盈门槛新增上限档；信号扫描周期新增 1 秒档位 | 7 文件 +85 |
| 8bdd55e | feat(kline): 买卖点共振 9 路扩至 14 路——新增趋势组并补量价/趋势组 | 10 文件 +195 |

合计 22 文件 +443/-69。go.mod 无变化。

## 覆盖核对结论（22/22 全部有归宿）

### A. 直接同步（工作树 == origin/dev，0 diff，已暂存）

- backend/data/okx_signal_min_tp_test.go（新 68，MinTp/MaxTp 估算测试）
- backend/data/okx_signal_trade.go（+68，开单前止盈收益估算门槛 min/max）
- backend/db/db.go（+33，Close(timeout) WAL checkpoint 优雅关闭 = cdf4a07 数据层）
- frontend/src/components/SignalMonitorPanel.vue（4，满分 9→14 文案）
- frontend/src/components/aiRecommendStocksList.vue（21，14 路信号评分口径）
- frontend/src/components/kline/calc.ts（+126，Donchian/MFI/CMF/Supertrend/BBI 五组扩路至 14 路）
- frontend/src/components/kline/constants.ts（+13，档位 3/4/5→4/6/7）
- frontend/src/components/kline/indicators/tips.ts（2）
- frontend/src/components/kline/signalMonitor.ts（+69，14 路评分 + 1 秒档节流）
- frontend/src/components/kline/temaTurnPrimitive.js（4）
- frontend/wailsjs/go/models.ts（+4，MinTpPct/MaxTpPct 字段）

### B. 手工合并（本地有用户隔离/Web 偏离，未暂存）

1. **backend/data/settings_api.go**（上游 +19）：
   - Settings struct +2 字段（OKXSignalMinTpPct / OKXSignalMaxTpPct，含注释）
   - GetSignalScanIntervalSec 下限 5→1（注释同步：1 秒档仅作唤醒节拍）
   - UpdateConfig map +2 键；**本地独有 UpdateConfigByUserID 同步 +2 字段映射**（用户隔离铁律）
2. **frontend/src/components/settings.vue**（上游 +20）：
   - formValue 回读/保存/导入 4 处 + min/max TpPct 字段
   - 2 个新表单项（开单最低/最大止盈 %，HelpTip 全文照上游）
   - 「最低信号分数」HelpTip 更新为 14 路口径（满分 14、建议 8~10、旧 6~7 需上调提醒）
3. **backend/models/signal_record.go**（2，Family 注释补 hull——8bdd55e）
4. **internal/handlers/okx_handler.go**（9，Web 接线）：SetSignalScanIntervalHandler 校验下限 5→1
5. **web/main.go**（+6，**我方 Web 适配 cdf4a07**，上游无此文件）：
   优雅关停路径补 db.Close(20s) + logger.Sync()——os.Exit(0) 跳过 defer，
   否则强杀遗留巨型 WAL 会让下次启动在 db.Init 全量恢复时长时间无响应

### C. 桌面专用，不移植（Web 构建范围 ./web ./internal ./backend ./ai-assistant-web 不含）

- app.go（+3）、main.go（+5）、update_helper_{darwin,linux,windows}.go（+28）：
  单实例锁重启链路，桌面 Wails 入口专用；Web 端更新=重新部署，无应用内更新交互
- app_signal_monitor.go（6）：桌面端 cron 扫描节拍，Web 端对应物是 scheduler + okx_handler（已改）
- frontend/src/App.vue（6）：重启确认按钮 loading 态——Web 版 App.vue 无 RestartApp 对话框（grep 证实），不适用

### D. 欠账登记（不在本批范围，另开批次）

**StockLightweightKlineChart.vue（上游 8bdd55e +11 行无法套用）**：
Web 版图表组件从未接入买卖点共振体系——上游图表有 7 个绘图原语
（Measure / Wave / VolumeProfile / TDSequential / Divergence / BuySell / TemaTurn），
Web 版只接了 VolumeProfile + Wave 两个。buySellPrimitive.js / temaTurnPrimitive.js 等
原语文件已在库（b316fdb 移植），但组件无接线（buySellMinScore 设置项、
syncBuySellPrimitive、getBuySellBars 数据管线全缺）。
因此：calc 层 14 路计算本批已同步（后台信号监控/自动交易全量生效），
但 Web 图表上看不到买卖点/TemaTurn 标注、也无阈值设置 UI。
**另开批次**：一次性补齐 5 原语接线（自上游 7f7b446 引入买卖点起的组件层演化）。

## 提交拆分计划（沿 008 惯例）

| 提交 | 内容 |
|------|------|
| B1 | 后端：okx_signal_trade + min_tp_test + db.go + settings_api（手工）+ signal_record |
| B2 | Web 接线：okx_handler.go（1s 下限）+ web/main.go（WAL 优雅关停） |
| B3 | 前端直接同步 8 文件（0 diff 组） |
| B4 | 前端手工合并：settings.vue |
| docs | 本文档 |
| chore | 26 个与同步无关的 AI 元数据删除（.claude/.devflow-kit 旧文档/.superpowers-memory/.forgeos/CLAUDE.md/CODE_OF_CONDUCT.md），用户确认单独提交 |

⚠️ 工作区另有 4 个未跟踪项（SOUL.md / memory/ / tools/gen-sponsor-code/ / ai-assistant-web data/）不属于本批，不动。

## Phase 2a 变体记录（轻量）

本次无方案分歧：分层移植是 001~008 验证过的既定路线（不 merge、不整 cherry-pick、
手工保用户隔离），唯一争议点（K线组件缺口）已由用户裁决=登记欠账另开批次。
备选「整取上游图表组件」被否决：会覆盖 Web 重写版（bridge 重接线/轮询架构/008 isOKXCode 适配）。

## 验证基线（沿 007/008）

- go build ./web ./internal ./backend ./ai-assistant-web
- go vet 同范围
- go test backend/agent backend/agent/tools + backend/data 定向（TestOKXSignalMinTp* 新用例）
- vite build（临时 outDir，绕开批量删除保护）
- A 类 11 文件 HEAD vs origin/dev hash 逐一致核对

## 执行后补记（2026-10-24 晚）

### 实际提交链

| 提交 | 内容 |
|------|------|
| 84f2ff9 | B1：后端 5 文件（okx_signal_trade + min_tp_test + db.go Close + settings_api 手工含 UpdateConfigByUserID + signal_record 注释） |
| 50efafb | B2：Web 接线（okx_handler 扫描下限 1s + web/main.go 优雅关停 WAL checkpoint） |
| adb8416 | B3：前端直接同步 8 文件（14 路共振 calc/监控/面板，0 偏移） |
| 026f53f | B4：settings.vue 手工合并（min/max 止盈表单 + 14 路口径文案） |
| 1cc2b04 | docs：本文档 |
| bd81bc5 | chore：26 个废弃 AI 元数据删除（用户确认，独立提交） |

### 验证基线（全部通过）

- A 类 hash 一致核对：11/11（HEAD vs origin/dev）
- go build / go vet ./web ./internal ./backend ./ai-assistant-web 全绿
- go test backend/agent（78s）+ backend/agent/tools 全绿
- backend/data 定向：TestOkxSignalTpYieldPctFixed/Vol/OutOfRange 3 新用例 + TestOKXCalcSizeByAmount/TestOkxAlignPx/TestOKXSignalBarTimeWindows 全过
- vite build 5m15s 通过（仅 chunk 体积警告，与上游同步无关）

### 计划外发现：Go 模块缓存批量损坏（与同步无关的环境事故）

- 今天 12:13 前后模块缓存被外部清理破坏（与 26 个元数据文件被删同时段）：
  `go mod verify` 报 9 个模块 "dir has been modified"，另有 mimetype v1.4.3 漏网
  （提取目录只剩 testdata/，verify 因时机未报）。
- 症状：`undefined: translation` / `no required module provides package` 等编译错误。
- **修复口诀**：删损坏模块目录 → `go mod download <mods>` → **必须再 `go clean -cache`**
  （构建缓存会重放加载期失败，模块修好后不清构建缓存错误依旧）。
- 同时段 26 个 git 跟踪的元数据文件被删，属同一次外部清理，已按用户指示单独提交为 bd81bc5。
