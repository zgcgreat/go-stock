# 003 · dev-web 一次性对齐上游基底（B 路线）立项

> 状态：**已执行完成（merge commit `8478430`，2026-09-26）**，运行时冒烟测试待测试环境执行 ｜ 创建：2026-09-15 ｜ 前置：002 评估报告、C 路线已执行（见 §6）
>
> ## ✅ 执行结果速览（2026-09-26）
>
> - **合并提交**：`8478430` = merge origin/dev (`c26304f`) into dev-web
> - **回滚锚点**：tag `web-before-rebase-20260926`（= `abbda6a`）
> - **门禁**：`go build ./...` ✅ ｜ `go vet ./internal/... ./backend/...` ✅ ｜ `vite build` ✅
> - **实际取侧策略**（比 §5 作战地图更精细，基于执行中的发现）：
>   - 上游胜：backend/agent、backend/data 大部、backend/db、app.go、app_common.go、main.go、ai-assistant-web、go.mod/go.sum、wailsjs 绑定、9 个直连 wailsjs 的前端组件、kline/calc.ts+constants.ts
>   - dev-web 胜：internal/**（上游不触碰）、26 个桥接组件、router.js、App.vue、wails-bridge.js、settings_api.go、cron_task_api.go（用户隔离）
>   - 上游版+重放 dev-web 适配：models.go（双方模型并集+去重，上游也已收敛到 UserID 隔离方案）、ai_recommend_stocks_api.go（上游+142 行 + 隔离重放）、dingding/feishu_api（上游 + WithConfig 方法）、settings_api.go（上游 9 个新字段）、cron_task_api.go（补回 ExistsByName）、app.go 等调用点（userID=0 桌面语义）
>   - vite 别名发现：`wailsjs/*` import 在构建层本就重写到桥接（8 个直连组件的 web 兼容机制）；补了 `runtime/runtime` 尾缀别名
>   - wails-bridge.js 追加 42 个上游新方法桩（wails 转发 + web 模式明确报错），对应政策/龙虎榜/游资/复盘/回测/信号等二期功能
> - **遗留**：§8 的运行时冒烟清单必须在测试环境过一遍后才可发布；42 个桩函数的 Web 端点 = 二期解锁工作（§7 清单不变）

## 1. 背景与目标

`dev-web` 与上游 `origin/dev` 自分叉点 `ef9797b`（2026-06-09）以来是**结构性分叉**：
B1–B5 期间以「改造式移植」落地（新 SHA、无共同历史），agent 层被重构、部分 `backend/data`
文件被删、Phase-6 安全审查又叠加了一层本地安全改造。逐提交 cherry-pick 已被证明
每提交撞冲突（002 报告 §4）。

**目标**：把 `dev-web` 一次性对齐到上游最新基底 `origin/dev`（当前 `c26304f`，执行时先重新
fetch），使后续同步退化为「普通 cherry-pick / merge」，消除结构性冲突泥潭。

## 2. 前置条件（缺一不可）

| # | 条件 | 原因 |
|---|---|---|
| P1 | **可运行的 Web 测试环境**（浏览器可访问，能登录、看行情、聊天） | 本开发环境只能编译验证，B 路线冲突量大，纯编译门禁不够 |
| P2 | 数据库备份（或用一次性测试库） | 合并会带入上游模型改动（新表/新列/迁移逻辑） |
| P3 | 磁盘 ≥10GB 空闲（E: 已满足，GOCACHE=/e/gocache） | go build 全量编译 |
| P4 | 预留整块时间，单次会话内完成冲突季 | 半途状态最危险 |

## 3. 策略选择

| 方案 | 做法 | 优点 | 缺点 | 建议 |
|---|---|---|---|---|
| **B-merge** | 在 dev-web 上 `git merge origin/dev` 一次，集中解冲突 | 保留 dev-web 全部历史与既有 commit；可分多次会话处理（merge 中断可用 `git merge --continue` 续） | merge commit 巨大 | **推荐** |
| B-rebase | `git rebase --onto origin/dev ef9797b dev-web` 逐个重放 dev-web 提交 | 历史线性，之后每个 dev-web 提交都能干净对应 | dev-web 提交数量多（含改造式移植的大提交），每个都要解一遍，且重写历史影响远端 | 不推荐 |
| B-fresh | 从 origin/dev 拉新分支，把 dev-web 的「web 适配层」作为补丁集重新套用 | 结果最干净 | 等于重做 web 化，工作量最大 | 仅当 merge 失败兜底 |

## 4. 必须保留的 dev-web 资产清单（冲突时以本清单为准）

1. **wails-bridge.js 模式**：所有前端组件 import 走 `../services/wails-bridge.js`，
   上游是直连 `../../wailsjs/go/main/App`——合并后必须保持桥接 import；
   新增 App 方法需在桥接补「wails 直连 + Web 走 `/api/v1/*` 同源代理」双路。
2. **Phase-6 安全层**（`9e04ce6`）：JWT secret 环境变量、VIP 服务端校验、CORS 白名单、
   token 撤销、登录/注册限流、plaza AES-GCM、SSRF 加固、AdminRequired 写门槛——
   上游若回退这些文件（internal/handlers、middleware、server.go），以上游代码为底、
   把 Phase-6 diff 重新叠加，**不得直接取上游侧**。
3. **路由/菜单**：`router.js` + `WebLayout.vue`（上游用 App.vue 桌面菜单）。
4. **数据隔离**：web 端新表必须带 `user_id`。
5. **已删除文件**：`backend/data/gov_policy_lib_api.go`、`policy_news_api.go` 等
   dev-web 已删文件，merge 出现 modify/delete 时**默认保持删除**（policy 功能如需，
   在 B 完成后作为独立特性重评，见 §5）。

## 5. 冲突分区作战地图（按 002 扫描结果）

| 区域 | 预期冲突 | 解法要点 |
|---|---|---|
| `backend/agent/*`、`backend/agent/tools/*` | agent 层重构 vs 上游演进 | 以上游为底，把 dev-web 的 web 化钩子（web 判定、桥接函数）重新叠加 |
| `internal/handlers/*`、`internal/middleware/*`、`internal/webserver/server.go` | 上游无 Phase-6 | 上游为底 + 重放 Phase-6 diff（§4.2） |
| `backend/data/{gov_policy_lib,policy_news}_api.go` | modify/delete | 保持删除 |
| `frontend/src/components/{skillPlaza,agent-chat,promptPlaza*}.vue` | plaza 系列 Phase-6+上游混改 | 逐 hunk 手工合并，先取上游再叠 Phase-6 |
| `frontend/wailsjs/**` | 绑定生成物 | 取上游生成内容 + 桥接侧补齐 web 代理 |
| `app.go`/`main.go` 桌面入口 | dev-web 弱化桌面 | 保留 dev-web 侧为主，仅吸收上游新增 App 方法签名 |

## 6. C 路线执行结果（本轮已完成，B 前无需重做）

| 提交 | 内容 | 结果 |
|---|---|---|
| `9e04ce6` | Phase-6 安全基线 | ✅ 已提交，编译通过 |
| `de9a8d5` | c26304f 提示词分享走后端代理 | ✅ vite build 通过 |
| `abbda6a` | a0d088a 关于页卡片式重构+赞助码样式 | ✅ vite build 通过（App.vue 游资菜单入口剔除：market 无对应 tab） |
| 12e1140 | K线六项指标 | ⏸ 与未移植的绘图功能纠缠（blk 混合 30KB），留 B 后处理 |
| 51eae6d | 板块/概念点查成分股 | ⏸ 全栈纵向（Go 方法+绑定+桥接+web_handler 端点），留 B 后处理 |

## 7. B 完成后的解锁清单（按价值排序）

> **执行状态（2026-09-26，`c407d6f` → `50464b0`，四批）**：已解锁 36/42 桩——成分股/政策新闻/龙虎榜汇总/游资名录/复盘/晨报/
> 画像学习模型/agent 反馈/回测详情/MCP OAuth/**提示词回测（5）/推荐回测（6）**。新增 7 个页面路由与菜单
> （政策/龙虎/游资-admin/复盘/晨报/回测/回测统计）。RunRecommendBacktest 为异步（回放可能超 axios 30s 超时）；
> promptBacktestProgress 桌面事件不到 Web（任务状态靠列表刷新）。
> **剩余 6 个桩 → 更新（2026-09-27，`ddd513e`）**：GetAiRecommendStocksTodayStats 已接入（此前误判组件未路由——
> 实际挂在 研究→股票推荐记录 Tab，桩返回 undefined 导致统计卡恒显 0；现按用户隔离接入端点，**37/42**）。
> **非桩项推进（2026-09-27，`0f5ddea`）**：自选量比列 ✅（上游 verbatim，数据经桥接 `...stockInfo` 透传自动到达）；
> 龙虎榜 K 线弹窗 ✅（`0fb87dc` 在 B-merge 冲突中丢失的部分，逐段摘樱桃：名称列 hover popover 改点击按钮 +
> toKlineCode 代码转换 + n-modal 内嵌 KLineChart；浏览器验证蜡烛图/MA 图例/成交量柱渲染正常）。
> **非桩项收尾（2026-09-27，`ea3b218` + `bb5c73e`）**：交易记录导入/真 Excel 导出 ✅——上游 070aff4/9609b00/ce58d15
> 的后端解析与 excelize 生成逻辑已随 B-merge 进入，本轮补齐 Web 通道：三端点（multipart 导入/模板下载/表格导出）
> + 桥接双模（Wails 原生对话框 vs Web 上传/blob 下载）+ 三组件入口（研究→交易日志导入指引弹窗、指标/形态选股导出按钮）。
> `ImportTradingRecords` 加 userID 参数完成用户隔离（去重查询与落库均按用户过滤，桌面传 0 行为不变）。
> 顺带修复第 6 个缺陷：导入记录时间显示 +1 天（`bb5c73e`，见 §8.4）。**非桩项清单至此全部完成。**
> **视觉理解补齐（2026-09-27，`9ea949e`）**：上游 `956fd75`+V 批的后端视觉管线已随 B-merge 进入
> （buildVisionImageParts 按 provider 分支：OpenAI 系 URL 直传 / Claude data URL 拆 raw base64 / Gemini、Ollama 下载转 base64），
> 本轮补齐 Web 通道：`POST /ai/upload-image` 图床上传端点、SSE 请求体透传 images、agent-chat 图片 UI 全套
> （粘贴/URL/本地上传、上传中拦截、视觉模型 gating、气泡缩略图，TDesign 适配）。顺带修复存量缺陷：Web 处理器
> optsOverride 位序错位（userIDStr 落在 [2] 被 agent 拼进 sysPrompt；sessionId 未传致全局 "default" 会话跨用户
> 串话风险）→ 位序对齐签名、sessionId 启用会话记忆（见 §8.5）。
> **剩 5 个信号桩维持暂缓**：SignalMonitorPanel 上游也无人挂载，且信号引擎依赖 Wails 运行时事件（Go cron 60s 节拍），
> Web 接入属功能开发（需 Web 节拍 + 面板挂载点 + 持久化端点整体设计），非桥接对齐。**→ 已立项 004**：
> `.devflow-kit/004-signal-monitor-web/README.md`（2026-09-27，Web 节拍用 SSE 事件总线 + WebLayout 挂载 +
> user_id 隔离持久化端点，5 批实施；实际解桩范围 8 个——另含 IsTradingTime 三兄弟 web-dead 修复）。
> **→ 004 已完成（2026-09-28，`36e1361`/`3e3f223`/`15845d1`/`a2bae48`）：本清单 42/42 桩全部达成。**
> 信号引擎三重节拍落地（SSE 主通道→前端定时器兜底→手动扫描）；e2e 抓到 vite alias 三层 runtime 缺口并修复
> （详见 004 §3.4）。本清单至此整体结项。
> **冒烟已通过（2026-09-27，见 §8.1）**：API 19/19 端点 + 浏览器级 7 页点验；抓到并修复 3 个真缺陷（`bf78100`、`5a5d945`）。

1. `51eae6d` + `0fb87dc`：板块/概念点查成分股 ✅（成分股部分）、自选量比列 ✅ + 龙虎榜 K 线弹窗 ✅（`0f5ddea`）
2. `12e1140`：K线成交量分布/神奇九转/BBI/背离/涨跌停线（与上游绘图版自然对齐）✅ 随 B-merge 进入
3. `f0c0d2d`+L 批：龙虎榜席位 ✅、游资名录 ✅、复盘 ✅、晨报 ✅、交易记录导入/真Excel导出 ✅（`ea3b218`）
4. `956fd75`+V：视觉理解（需给 agent-chat.vue 自建图片输入 UI，上游只有桌面版）✅（`9ea949e`，见 §8.5）
5. R 回测（8 提交）+ S 信号监控（3 提交）+ A 注册验证码（1）
6. 23 个独立后端纯逻辑提交（agent 自我进化/反馈学习/模型元数据/VIP 门槛等）
7. `7f7b446`/`dae729d`/`9609b00`/`4917b98`/`4632e5c`/`284c492`/`167170b`/`46513bf`/`070aff4`/`ce58d15`

## 8. 执行步骤（B-merge）

```bash
# 0) 前置检查
git fetch origin dev
git tag web-before-rebase-$(date +%Y%m%d)          # 回滚锚点
go build ./internal/... ./backend/...              # 起点 green
cd frontend && npm run build                       # 起点 green

# 1) 发起合并（勿用 --squash，保留可续性）
git merge origin/dev --no-ff -m "WIP: merge upstream dev for realignment"

# 2) 逐文件解冲突（按 §4 资产清单 + §5 作战地图）
git status | grep -E '^(UU|DU|UD|AA)'
# DU/UD（删改冲突）：gov_policy_lib_api.go、policy_news_api.go 等 → git rm
# 其余按分区解法，每解完一个区域： go build ./... && cd frontend && npm run build

# 3) 全量门禁
go vet ./internal/... ./backend/...
go build ./... && cd frontend && npm run build

# 4) 运行时冒烟（测试环境，P1 前提）——逐项打勾：
#   [ ] 登录/登出/注册   [ ] 自选股增删   [ ] K线图加载+切换指标
#   [ ] AI 对话         [ ] 提示词广场   [ ] 设置页保存
#   [ ] 融资融券/龙虎榜  [ ] VIP 状态显示

# 5) 收尾
git merge --continue   # 或 git commit 收尾 merge
```

**回滚**：任一步不可救 → `git reset --hard web-before-rebase-*`（merge 未 commit 前）
或 `git revert -m 1 <merge-commit>`（已 commit 后）。

### 8.1 运行时冒烟执行结果（2026-09-27）

**隔离环境**：`go build -o /e/smoke-web/go-stock-web.exe ./web`；`GO_STOCK_WEB_PORT=18080`、
`GO_STOCK_DB_PATH=/e/smoke-web/smoke.db`（全新库=验证完整迁移链）、`GO_STOCK_STATIC_DIR=frontend/dist`、
`GO_STOCK_JWT_SECRET`、`GO_STOCK_ADMIN_PASSWORD`（空库仅能由此建管）。

**API 层**（curl + Bearer token）：
- 19/19 个新 GET 端点全部 200（成分股/政策×N/龙虎/游资/复盘/晨报/画像/回测列表/回测统计等）
- 管理写端点业务校验正确：空游资列表拒绝、回测模板不存在拒绝、远程刷新 SSRF 防护在位

**浏览器层**（agent-browser：登录 → 遍历 7 新页 → 截图 + console 采集）：
- 登录跳转正常；政策（82 部门+3 条列表）/龙虎（空态优雅）/游资/复盘/晨报/回测/回测统计 全部渲染通过
- console 仅存上游 vue3-danmaku 弃用警告，无新增错误

**抓到并修复的缺陷**：
| # | 缺陷 | 修复 |
|---|---|---|
| 1 | 全新部署启动即 panic（gin-contrib/cors 空白名单冲突）——同源部署无需 CORS | 仅在解析出非空白名单时挂中间件 → `bf78100` |
| 2 | `/hot-money-seats` 整页空白：HotMoneySeatsManager 是 n-drawer（内部 show+defineExpose），上游桌面也无宿主=孤儿组件，直挂路由只渲染空注释 | 薄包装页 HotMoneySeatsPage.vue（进页自动展开+关闭后可重开）→ `5a5d945` |
| 3 | 一级 Tab 达 17 个后 <1560px 视口溢出且 n-tabs 无法滚动，右侧入口（回测统计/设置/用户管理）不可达 | <1560px CSS 隐藏 Tab 图标；<1250px 复用移动端"更多"抽屉机制 → `5a5d945` |

**判定为测试假象（非缺陷）**：
- `data/key_departments.json` 乱码——冒烟时 curl 中文经 Git Bash 控制台 GBK 转码所致，端点行为正确（存什么回什么），已用 UTF-8 修复本地文件
- `data/hot_money_seats.json` 被改写——启动时远程名录自动拉取（设计行为），已 `git checkout` 还原不入库

**已知边界（遗留）**：~~全新库 `POST /user-profile/learn-ai-config` 返回 500~~ **已修复（`05ca3cc`，见 §8.3）**。
§8 步骤 4 清单中 K线/AI 对话/提示词广场等旧页面回归在第二轮覆盖（见 §8.2）。

### 8.3 遗留缺陷修复：画像学习设置 500 + 用户隔离缺口（2026-09-27）

`SetProfileLearnAiConfigId` 对空表裸 `First()` → 全新库 `record not found` 500；且读写均走桌面全局语义
（无 user_id 过滤），Web 多用户下可能写到系统行或他人行。修复（`05ca3cc`）：
- 全新库自动建行：桌面路径建最小默认行；Web 路径先按 `GetSettingConfigByUserID` 的复制语义从系统默认行
  （user_id=0）复制落库，表全空则建最小行
- 新增 `Get/SetProfileLearnAiConfigIdByUserID`，Web handler 切换为按登录用户隔离读写
- 全新库实测：GET 200(data=0) → POST 200 → GET 200(data=5)，落行 `user_id=1, value=5`

### 8.2 旧页面回归结果（2026-09-27 第二轮浏览器冒烟）

按 §8 步骤 4 清单逐项验证（同一隔离环境）：

| 清单项 | 结果 | 备注 |
|---|---|---|
| 登录/登出 | ✅ | 登出有确认弹窗；token 清除、跳转登录页 |
| 未授权访问（§9 专项） | ✅ | 无 token 前端弹回 `#/login?redirect=…`；后端 API 返回 401 |
| 自选股增删 | ✅ | 关注→行情实时数据（1237.00/-1.138%，红涨绿跌正确）→取消关注（DELETE 200）全链路 |
| K线图加载+切换指标 | ✅* | 周K/日K 渲染、43 项指标信号汇总、区间切换正常。*首挂默认指数（000001.SH）偶发不自动加载，选股后一切正常（上游遗留，低危） |
| AI 对话 | ✅** | 页面/发送/流式管道正常；**抓到 bug**：新库无 AI 配置时 selectValue="default"（字符串），Wails 自动转型掩盖，Web 端 JSON 严格反序列化 400 → 桥接层 `toIntOrZero()` 统一加固 4 处 aiConfigId → `75ceee2`。修复后错误变为正确的业务提示「请先配置AI模型」 |
| 提示词广场 | ✅ | 内嵌 Tab/筛选/发布入口正常，新库空态 |
| 设置页保存 | ✅ | `POST /api/v1/config` 200；VIP 赞助码区（§清单 VIP 状态）渲染正常 |
| 融资融券 | — | 无独立页面（行情内聚合），未单独覆盖 |

**结论**：§8 步骤 4 冒烟清单全部打勾（融资融券除外，无独立入口）。三轮冒烟共发现 5 个缺陷，全部修复提交
（`bf78100`、`5a5d945`、`75ceee2`、`05ca3cc`）。剩余遗留：K线首挂默认指数偶发不加载（上游）。
非桩项：自选量比列 ✅ + 龙虎榜 K 线弹窗 ✅（`0f5ddea`）+ 交易记录导入/真 Excel 导出 ✅（`ea3b218`，见 §8.4）
+ AI 视觉理解 ✅（`9ea949e`，见 §8.5）。

### 8.4 交易记录导入/真 Excel 导出（2026-09-27 第四轮，非桩项收尾）

**范围**：上游 `070aff4`（模板+导入指引）、`9609b00`（真 Excel 导出）、`ce58d15`（东方财富适配）的后端核心已随 B-merge
进入（`export_table_api.go`、`parseTradingImportFile`、excelize v2.11.0 均在），本轮补齐 Web 通道（`ea3b218`）：

- 后端三端点：`POST /trades/import`（multipart 上传落临时文件，复用桌面全套解析/去重；解析器按 PK 魔数嗅探不依赖扩展名）、
  `GET /trades/template`（xlsx 模板附件流，RFC 5987 文件名）、`POST /export/table-xlsx`（通用表格导出，支持二级表头）
- 用户隔离加固：`ImportTradingRecords(filePath, userID)`——去重查询按 user_id 过滤、落库写入 user_id；桌面调用传 0 行为不变
- 桥接双模：导入（Wails 原生文件框 vs Web 隐藏 file input + FormData）、导出（Wails 保存框 vs Web blob 下载）；
  导出 `isWailsMode`；新增 `contentDispositionFilename`（filename* 优先）
- 组件入口：研究→交易日志「导入记录」按钮 + 导入指引弹窗（券商导出指引/模板下载/选择文件）；指标选股、形态选股「导出」按钮
  （形态选股按当前筛选拉全量 5569 条后导出）

**验证（curl + 浏览器级）**：

| 项 | 结果 |
|---|---|
| 模板下载 | ✅ 200，PK 魔数，`filename*=UTF-8''交易记录导入模板.xlsx` |
| 表格导出 | ✅ 200，PK 魔数，中文文件名正确（首次乱码为 Git Bash 控制台转码假象，UTF-8 文件载荷复测排除） |
| 导入（curl） | ✅ 2/2 成功；重复导入幂等（0 导入 2 跳过）；落库 user_id=1、代码归一化 sh600519/sz300750 |
| 导入（浏览器） | ✅ 指引弹窗→选择文件导入→隐藏 file input→multipart→落库 sh600036，列表自动刷新 |
| 模板下载（浏览器） | ✅ toast「模板已保存：交易记录导入模板.xlsx」 |
| 形态选股导出（浏览器） | ✅ 全量拉取→POST /export/table-xlsx→toast「已导出：形态选股_20260927.xlsx」 |
| 指标选股导出按钮 | ✅ 渲染正确，无数据时禁用（未跑 AI 选股，端点已经形态选股链路验证） |

**第 6 个缺陷（`bb5c73e`）**：导入记录时间显示 +1 天——`formatRowTradingTime` 把带 +08:00 的 RFC3339 截断后误加
`' UTC'` 再二次 +8h 补偿，UTC+8 浏览器下双重转换 +16h。修复：带偏移直接解析（尊重原偏移），无偏移（历史数据）按 +08:00 兜底。

**测试环境注意（存量，非本轮引入）**：`go test ./backend/data/` 在本机因 db 包初始化 `log.Fatalf`（sqlite 打不开默认路径，
error 14）整包失败，stash 对照确认与本次改动无关；相关逻辑验证由 smoke 实测兜底。

**冒烟环境**：`/e/smoke-web/`（go-stock-web.exe 重建、server3.log、token3.txt、test-import.txt、dl-template.xlsx、
dl-table2.xlsx、export-payload.json）；admin 密码已重置为 smoke123（原密码遗忘，bcrypt 直改库）。

### 8.5 AI 视觉理解 Web 通道（2026-09-27 第五轮，非桩项收尾）

**范围**：上游 `956fd75`（桌面版 agent-chat 图片输入 + 图床上传）+ V 批（后端视觉管线）的后端已随 B-merge 进入：
imagesJSON → parseImagesJSON → buildVisionImageParts 按 provider 分支（OpenAI 系 URL 直传 / Claude data URL 拆
raw base64+MIMEType / Gemini、Ollama 下载转 base64）→ UserInputMultiContent（Content 必须清空，SDK 互斥）。
本轮补齐 Web 通道（`9ea949e`）：

- 后端 `ai_handler.go`：请求体绑定 `images []string` → JSON 序列化为 imagesJSON 透传 agent；新增
  `POST /ai/upload-image`（AdminRequired 保护内，base64 → data.UploadImageToImageBed → 外链 URL）
- 桥接 `wails-bridge.js`：`ChatWithAgent` Web 模式解析第 10 参 imagesJSON 入 SSE body；`UploadImageToImageBed`
  双模（Wails 调桌面绑定 vs Web POST /ai/upload-image）
- 前端 agent-chat.vue（TDesign 适配）：粘贴/URL/本地上传三种图片入口、pending 预览条（hasUploadingImage 上传中
  拦截发送）、视觉模型 gating（config.supportVision，下拉名尾 [视觉] 标注）、纯图默认文本「请分析这些图片」、
  用户气泡 msg-image-grid 缩略图、上限 10 张 / 单张 8MB、图床失败回退 base64 直传

**顺带修复存量缺陷（optsOverride 位序）**：Web 处理器曾把 userIDStr 放在 optsOverride[2]（resumeContextOverride 位），
agent 侧 `sysPrompt += resumeContextOverride` 会把用户 ID 拼进系统提示词；且 sessionId 未传导致所有人共用全局
"default" 会话（跨用户上下文串话风险）。修复：位序对齐 ChatWithContext 签名（[0]sysPrompt [1]sessionID
[2]resumeContext [3]skillQuestionBlock [4]imagesJSON [5]skillDirName），sessionId 改传 req.SessionID 启用会话记忆。

**验证（e2e，smoke 环境 18080 + agent-browser）**：

| 项 | 结果 |
|---|---|
| 图床 curl | ✅ 1x1 红 PNG multipart POST → 返回真实外链（请求发 img.scdn.io，实测返回 img.cdn1.vip 域名） |
| URL 添加图片 | ✅ chip 入 pending 条；Enter 后 window.fetch 钩子捕获请求体 `{question, images:[外链], aiConfigId:1, sessionId:""}`（XHR 钩子抓不到——桥接用 fetch） |
| 服务端 vision 路由 | ✅ 日志三连：createChatModel vision 路由 → `vision: 下发 1 张图片（config=测试视觉模型, model=deepseek-vision-test）` → OpenAI 兼容组件 |
| 假 key 边界 | ✅ sk-fake-smoke → 401（预期，验证链路到上游 SDK 调用层即止） |
| 气泡渲染 | ✅ 用户消息 msg-image-grid 缩略图正常渲染 |
| 发送路径 | ✅ t-chat-sender 的 v-model 不被原生 setter 同步，e2e 用 textarea focus + KeyboardEvent('keydown',{key:'Enter'}) 触发 send |

**环境注意**：smoke2.db 补插 AI 配置（settings open_ai_enable=1 + ai_config 测试视觉模型 support_vision=1）；
agent-chat.vue 存在 Vue TDZ 陷阱——currentConfigSupportsVision / hasUploadingImage 等 computed 必须声明在
selectOptions/selectValue 之后，否则整页白屏 ReferenceError。


## 9. 风险登记

| 风险 | 等级 | 缓解 |
|---|---|---|
| Phase-6 安全层在合并中被冲掉 | 高 | §4.2 铁律 + 冒烟清单含「未授权访问被拒」专项验证 |
| 上游模型/迁移与 web user_id 方案冲突 | 中 | 合并后 diff 检查 `backend/models` + 新表迁移路径 |
| wailsjs 生成物覆盖桥接 | 中 | 合并后全局 grep `from '../../wailsjs/go/main/App'` 应仅存于桥接文件内部 |
| 运行时回归编译查不出 | 高 | P1 测试环境为硬前置 |
