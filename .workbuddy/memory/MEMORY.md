# MEMORY.md - go-stock 项目长期记忆

## 项目基本信息
- **项目名**: go-stock
- **定位**: 基于大语言模型的 AI 赋能股票分析工具（桌面应用 + Web端）
- **技术栈**: Go + Wails + Vue 3 + NaiveUI + Vite
- **数据库**: GORM（本地 SQLite）
- **AI 接入**: OpenAI 兼容接口，支持 DeepSeek、Ollama、LMStudio、硅基流动、火山方舟等
- **编译**: `go build -o go-stock-web.exe ./web/` 编译 Web 端（入口 `web/main.go`，**不是 ./cmd/web/**）；根包 `go build ./...` 需 Wails toolchain
- **上游源**: `github.com/ArvinLovegood/go-stock` (origin/dev)
- **分支策略**: `dev-web` 基于 `origin/dev` 二次开发；**功能块移植路线（不 merge、不整体 cherry-pick）**，同步批次见 `.devflow-kit/00X-upstream-web-sync-*/sync-design.md`
- **同步前沿**: `9cf1802`（2026-10-07 完成 005 批次 B0–B5c，8 个 commit 覆盖 18/20；回滚锚点 tag `backup/pre-005-sync`=`fd981fb`）
- **⚠️ 不可整文件取上游版的文件**：`backend/data/stock_data_api.go`（本地 `models.go` 也有 `StockBasic`，整取会 redeclared）——只能手工补改动点
- **Web 认证**: axios 拦截器统一拦截无 token 业务请求；公开路由 `/public/` `/auth/` 不拦截
- **Web 挂载点**: `frontend/src/components/layout/WebLayout.vue`（菜单 + 面板挂载），路由 `frontend/src/router/router.js`
- **⚠️ 禁止触碰 `frontend/wailsjs/`**：Wails 生成物，跨版本 diff 巨大、冲突高发；Web 端只改 `frontend/src/services/wails-bridge.js` 补 HTTP 映射
- **⚠️ 本环境禁用 git stash**（已两次损坏仓库）；临时还原用 python copy2 备份到 gitignored 目录 + `git checkout -- <精确清单>`

## 目录结构
- `backend/data/` — 数据层：行情爬虫、AI 接口、东方财富K线、市场资讯、基金、工具函数集
- `backend/agent/` — AI 智能体：工具调用 Agent，定时任务 CronTaskApi
- `backend/models/` — 数据模型（GORM）
- `backend/db/` — 数据库操作
- `internal/webserver/server.go` — Web 路由注册 + MigrateAllTables
- `internal/handlers/` — Web 端 Gin Handler
- `internal/middleware/` — AuthRequired/AdminRequired/GetUserIDFromContext
- `frontend/src/components/` — 40+ Vue 组件
- `frontend/src/services/wails-bridge.js` — Web 模式 API 桥接

## 核心设计原则
- **桌面端不改动**：Web 端基于桌面端二次开发，桌面端代码尽量不改动
- **用户隔离**：`userID=0` 桌面端不过滤，`userID>0` Web 端过滤；公共数据不按 userID 过滤
- **路由模式**：`/api/v1/public` 无需认证；`/api/v1/` protected 需 AuthRequired；`/api/v1/admin` 需 AdminRequired
- **响应格式**：`gin.H{"code": 0, "message": "success", "data": ...}`
- **SQLite 迁移**：GORM AutoMigrate + `migrateUserIDColumn()` 显式补列
- **⚠️ 「整文件取上游版」会丢 Web 改造**：002 批次 `8478430` 曾整取 `agent_api.go`，
  把 Web 用户隔离（`GetSettingConfigByUserID`）覆盖掉，导致多用户共用 `user_id=0` 全局配置。
  取上游文件前必须先 `git diff <base> HEAD -- <file>` 确认我方是否改过；改过就手工补改动点。
  已修：`agent.ChatRequest` 新增 `UserID` 字段，`newStockAiAgent` 按其走 `GetSettingConfigByUserID`

## 本机环境坑位（2026-10-07 实测）
- **npm/npx 均损坏**（`Cannot find module npm-prefix.js`）。可用：`node "D:/Softwares/nvm4w/nodejs/node_modules/npm/bin/npm-cli.js" install <pkg>`
- **vite build 须绕开批量删除保护**：清空 `frontend/dist`（181 文件 > 50 阈值）被 `node-safe-delete-shim` 拦下。
  用 `node node_modules/vite/bin/vite.js build --outDir <临时目录> --emptyOutDir false`（不能走 npx）
- **`.gitignore` 的 `data/` 屏蔽了 `backend/data/`**：已跟踪文件用 `git add -u backend/data/`；新文件需 `-f`
- **远程 git 操作**须 `GIT_CONFIG_GLOBAL=/dev/null`（credential.helper=helper-selector 会卡死）


## 提示词广场架构（改造后 2026-06-07）
- **查询策略**：直连外部API优先，Web端在外部API失败时降级查本地缓存表
- **本地表**：`plaza_prompts`（PlazaPrompt）、`plaza_questions`（PlazaQuestion）— 仅做备份和降级
- **定时任务类型**：`prompt_plaza_sync`，默认账号 Joy/zgc@202123，定时同步外部数据到本地表
- **Web端降级路由**：`/api/v1/plaza/prompts|questions|categories|sync`（前端只在catch中调用）
- **写操作直连外部**：发布/点赞/收藏/评论/提问/回答等仍前端直连外部API
- **前端改造**：`promptPlaza.vue`和`promptQa.vue`改为 try外部API → catch降级本地API
- **核心文件**：`backend/data/plaza_sync_service.go`、`internal/handlers/plaza_handler.go`
- **降级时提示**：`message.warning('外部服务不可用，已切换到本地缓存数据')`
- **VIP控制本地化**：Web端用本地用户VIP等级（Auth.getUserInfo()）判断权限，不依赖外部广场needVip/vipLevel
  - `isLocalVip` computed：本地VIP有效或管理员
  - VIP遮罩：`v-if="detailModal.data.vipOnly && (isWebMode ? !isLocalVip : detailModal.data.needVip)"`
  - addPromptToTemplate/handleDownload/handleCopyContent：Web端用isLocalVip拦截
  - 顶部用户标签：Web端显示本地VIP状态
  - 发布默认vipOnly：Web端用isLocalVip
  - 列表作者VIP标签：Web端不显示（避免混淆本地/外部VIP）

## 已完成的重大改造
- Web 化改造（2026-04-05）：wails-bridge.js 重写、Web 服务器、SSE 流式接口
- 用户隔离（2026-04-26）：11 表加 user_id，所有 Handler 加用户过滤
- 编译修复（2026-04-06）：多文件 import/类型错误修复
- K线/配置/Agent 用户隔离修复（2026-04-26）
- AI Agent 模型选择/涨停梯队修复（2026-04-26 晚）
- Markdown 渲染统一 MdPreview（2026-06-04）
- Agent Chat 配置用户隔离（2026-06-06）
- 代码审查批量修复：AuthRequired 统一、路由整理、userID 统一（2026-06-06）
- 用户管理菜单条件展示（2026-06-07）
- 上游同步 9 提交 cherry-pick（2026-06-20）：概念资金流向、TDX MAC、板块查询、时区修复
- Web 端 AI 菜单功能对齐（2026-06-21）：agentMode 修复、enableAgent/enableFund 联动、会话持久化、提示词/思考/记忆/分享/导出图片
- 登录页 API 拦截（2026-06-21）：axios 拦截器统一拦截无 token 业务请求
- 设置保存修复（2026-06-21）：enableAgent/enableFund 强制覆盖 bug、保存成功提示红叉
