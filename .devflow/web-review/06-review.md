# go-stock Web 端代码审查报告（Phase 6 双层对抗审查）

> 审查范围：Web 后端（internal/webserver、internal/handlers、internal/middleware、internal/migrations）+ 前端 web 模式适配（wails-bridge.js、axios 拦截、SSE、提示词广场、VIP 本地化）
> 模式：Strict（跨多模块）｜日期：2026-09-15
> 结论：用户隔离主体扎实，但存在 2 个 P0（密钥/初始管理员）、4 个 P1（VIP 越权、概念未隔离、SSE 401、skillPlaza 失效）及若干中低风险。

---

## 一、审查维度（自定 8 切面）
1. 架构分层 & Web/桌面端隔离
2. 认证授权（AuthRequired/AdminRequired、token、越权）
3. 用户隔离（user_id 全链路过滤、公共数据不串号）
4. 前端 web 模式适配（桥接、拦截、SSE、VIP 本地化、降级）
5. 健壮性与错误处理（nil/panic/吞错/并发）
6. 安全（注入、密钥、OWASP、SSRF）
7. 数据与并发（定时任务、连接、资源泄漏）
8. 配置与一致性（设置保存、路由模式）

---

## 二、问题清单（按严重度）

### 🔴 P0 — 阻塞级（必须修复后才能公网部署）

| # | 位置 | 问题 | 影响 | 修复 |
|---|------|------|------|------|
| 1 | `internal/middleware/auth_middleware.go:16` | **JWT 密钥硬编码** `JWTSecret = "go-stock-jwt-secret-key-2026-change-in-production"`，且在源码/二进制中明文 | 任何人可离线伪造任意用户（含管理员）token，完全绕过认证 | 改为从环境变量/配置读取；部署前轮换；禁止提交真实密钥 |
| 2 | `internal/handlers/admin_handler.go:104-143` | **初始管理员 `admin/admin123` 自动创建**，且用 `println` 把账号密码打到 stdout | 空库部署后任何人可用 admin/admin123 登录接管；密码泄露到日志 | 初始密码随机生成或首次登录强制改密；删除 `println`；生产默认关闭自动建管 |

### 🟠 P1 — 高（应本迭代修复）

| # | 位置 | 问题 | 影响 | 修复 |
|---|------|------|------|------|
| 3 | `internal/handlers/plaza_handler.go:60-85` + 前端 `promptPlaza.vue:1179-1194` | **VIP 内容服务端未鉴权**：`GetPlazaPromptDetail` 无条件返回完整 `content`；前端仅用底部 120px 渐变遮罩“软挡”，正文已在 DOM | 非 VIP 用户直接 `GET /api/v1/plaza/prompts/:id` 即可拿到 VIP 全文/复制，付费墙形同虚设（越权读） | 服务端按当前用户 VIP 校验 `vipOnly`，无权限则不返回 `content`；前端遮罩仅作 UX |
| 4 | `internal/handlers/stock_concept_handler.go`（全文） | **概念标签未做用户隔离**：`AddConcept` 不写 `user_id`；`UpdateConcept/RemoveConcept/AddStockConcept` 按 id/code 操作且无归属校验；`concepts` 表也未在迁移列表（`web_migrations.go:14-27`） | 多用户下概念是全局共享资源，任一登录用户可改/删他人概念，与同为用户数据的「分组」(已隔离) 行为不一致 | 若概念应属用户私有：加 `user_id`+归属校验；若设计为共享：显式说明并至少加 `AdminRequired` 防误删 |
| 5 | 前端 `frontend/src/services/wails-bridge.js`（`NewChatStream`/`ChatWithAgent`/`SummaryStockNews`） | **SSE 流式 401 不处理**：`!response.ok` 仅 emit error 后 return，不清 token、不跳登录；错误/abort 分支未清理 `sseConnections`，连接泄漏 | token 过期时用户滞留聊天界面无引导；失败连接残留 | 401 时复用 api.js 逻辑（清 token→dispatch user-info-updated→跳 login）；错误/abort 分支统一 `abortSSEByKey` |
| 6 | 前端 `skillPlaza.vue`（`localApiBase` 缺失，默认 `http://go-stock.sparkmemory.top:1918/api`） | **skillPlaza 在 web 模式完全失效**：无 `/api/v1/plaza` 降级、硬编码外部 HTTP | HTTPS 下被混合内容拦截 + CORS 失败，技能广场列表/详情/分享全挂 | web 模式优先走本地 `/api/v1/plaza`，外部仅作可选增强；或经 `GetConfig` 注入 base |

### 🟡 P2 — 中（排期修复）

| # | 位置 | 问题 | 影响 | 修复 |
|---|------|------|------|------|
| 7 | `internal/webserver/server.go:52-59` | **CORS `AllowOrigins:["*"]` + `AllowCredentials:true`**：gin-cors 会反射任意 Origin | 任意站点可读 API 响应（令牌在 Header 下风险有限，但属错误配置） | 显式白名单 `AllowOrigins`，关闭通配+凭据组合 |
| 8 | `internal/handlers/auth_handler.go:353-357` | **登出无实效**：Logout 直接返回成功，token 无黑名单/jti，有效期 24h | 令牌被盗用 24h 内可用，无法服务端吊销 | 引入 jti+黑名单或短期 token+刷新 |
| 9 | `server.go:77-79`（/auth/login、/auth/register） | **无限流/锁容**：登录无失败锁定；注册返回 `USERNAME_EXISTS/EMAIL_EXISTS` | 暴力破解 + 用户枚举 | 登录加限流/锁容；注册统一返回模糊错误 |
| 10 | `internal/handlers/plaza_handler.go:163-174,253` | **广场凭据明文**：`SyncPlazaData` 密码走 query string（进访问日志）；`CreatePlazaSyncCronTask` 把 `username/password` 明文写进 `cron_tasks.params` | 凭据泄露（日志、DB） | 改 POST body 传参；DB 落库加密 |
| 11 | `internal/handlers/web_handler.go:1276,1204-1210` | **GET 产生写副作用**：`GetAllStockChangesWithPagingHandler` 落库；`GetHotWords` 无 24h 新闻时触发爬取 | 爬虫/预取即可触发重 DB 写与对外请求 | 改为 POST 或加防重/开关 |
| 12 | `internal/handlers/web_handler.go:732-745,796-837` | **SSRF 防护不全**：仅按 hostname 前缀挡内网，未解析域名 IP（DNS 重绑定），漏 `100.64.0.0/10`、`198.18.0.0/15` 等 | 通过域名指向内网可探测内部服务 | 解析 IP 后校验，覆盖全部保留段 |
| 13 | `server.go` skills/MCP 路由（无 `AdminRequired`、无 `user_id`） | **技能/MCP 服务器任意用户可改**：`CreateSkill/UpdateSkill/DeleteSkill`、`CreateMCPServer/.../DeleteMCPServer` 全局共享且无归属校验 | 任一经登录用户可删他人全局技能/MCP | 明确共享则加 `AdminRequired`；用户级则加 `user_id` |
| 14 | `internal/handlers/web_handler.go:1650-1668` | **ImportSkills 无上传大小限制**（`io.ReadAll` 整文件读） | 大文件内存耗尽（DoS） | 用 `http.MaxBytesReader` 限流 |
| 15 | `internal/handlers/fix_admin.go`（仍注册于 `server.go:381`） | **FixAdminRole 后门工具**虽 admin 网关，注释“生产应删除”但仍在线 | 误操作/遗留风险 | 生产构建移除该路由 |

### 🟢 P3 — 低（技术债，可延后）

| # | 位置 | 问题 |
|---|------|------|
| 16 | `internal/middleware/vip_http.go`（整文件）、`wails_compat.go` | **死代码**：`RequireVipRole/RequireAdminRole`（http.Handler 型）未接入 gin 路由；`WailsCompatMiddleware` 未在 `server.go` 注册，且 `handleAppMethods` 仅改写 path 后 `Abort()`，从不重分发 → `/wails/*` 返回 200 空体 |
| 17 | `admin_handler.go:110/125/139`、`stock_group_handler.go:20` | **生产代码残留 debug `println`/`fmt.Printf`**，泄露 userID/username 到 stdout |
| 18 | `web_handler.go:38` | `GetTelegraphList` 对 `data` 层可能返回的 nil 指针做 `len(*news)`，**有 panic 风险**（gin.Recovery 兜底为 500） |
| 19 | `web_handler.go:192-197` | `GetEMDictCode` 直接回传 path 参数忽略输入，空实现端点 |
| 20 | `web_handler.go:305`、`ai_handler.go:288` | `UpdateAppConfig/UpdateAiConfigs` 用 `!= "保存成功！"` 字符串判错，脆弱 |
| 21 | 前端 `useResponsive.js:26` | Web 模式检测一次性求值（`!window.go...`），若 `window.go` 异步就绪会永久误判为 web |
| 22 | `web_handler.go:645-652` | `ShareText/ShareAnalysis` 桩函数恒返回成功，误导 |

---

## 三、做得好的地方（保留）
- ✅ **用户隔离主体扎实**：trades（`trade_handler.go`）、groups（`stock_group_handler.go` 全程 `user_id`+归属校验）、follow、settings、AI 响应/助手会话、每日计划、提示词模板（`user_id=? OR is_public=?`）、定时任务均按 user_id 过滤并校验归属后再写。
- ✅ **密码 bcrypt 哈希**，登录/注册错误模糊化（`INVALID_CREDENTIALS`）。
- ✅ **全部 DB 访问走 GORM 参数化**，handler 层无字符串拼接 SQL（迁移里表名是常量，非外部输入，无注入）。
- ✅ **AuthRequired/AdminRequired 中间件**模式统一，管理员接口集中在 `/admin` 子树。
- ✅ **SSRF 已有雏形**（FetchAiModels）、前端广场降级逻辑本身正确（仅 catch 时回退、有提示、无双发）。

---

## 四、优先级行动建议
1. **立即（P0）**：环境变量化 JWT 密钥 + 轮换；初始管理员密码随机化/去除 `println`。
2. **本迭代（P1）**：服务端 VIP 校验；概念用户隔离或加 admin 网关；SSE 401 与连接清理；skillPlaza 走本地 API。
3. **下个版本（P2）**：CORS 白名单；token 吊销；登录限流；广场凭据加密；GET 去写副作用；SSRF 补全；技能/MCP 权限；上传限流。
4. **技术债（P3）**：删死代码/debug 日志/桩端点；nil 守卫；统一错误判定；统一 web 模式检测。

---

---

## 五、修复记录（全部问题已处理）

> 修复日期：2026-09-15 ｜ 验证：`go build ./internal/... ./backend/...` 通过（EXIT=0）；
> 前端改动文件通过 `node --check` 语法校验（wails-bridge.js、useResponsive.js、skillPlaza.vue script 块）。
> 注：构建需把 GOCACHE/GOTMPDIR 指向剩余空间充足的磁盘（本机 C 盘仅剩 ~61M 会失败）。

### 🔴 P0
| # | 修复 | 文件 |
|---|------|------|
| 1 | JWT 密钥改为 `GO_STOCK_JWT_SECRET` 环境变量；未设置时随机生成并打 WARN（重启后旧 token 失效），源码内不再有硬编码密钥 | `internal/middleware/auth_middleware.go` |
| 2 | 初始管理员改为需配置 `GO_STOCK_ADMIN_PASSWORD` 才创建；删除默认 `admin123` 与全部 `println`（改用 logger，不再泄露账号/用户数到 stdout） | `internal/handlers/admin_handler.go` |

### 🟠 P1
| # | 修复 | 文件 |
|---|------|------|
| 3 | **VIP 服务端鉴权**：`GetPlazaPromptDetail` 对 `vipOnly/needVip` 内容校验本地 VIP（含管理员），非 VIP 不下发 `content` | `internal/handlers/plaza_handler.go` |
| 4 | **概念写操作收敛**：概念为共享数据，读保持开放，`POST/PUT/DELETE` 全部加 `AdminRequired`，杜绝任意用户篡改他人概念 | `internal/webserver/server.go` |
| 5 | **SSE 401 + 连接清理**：新增 `handleSSEHttpError`（401 清 token→dispatch→跳登录）与 `cleanupSSE`，4 处流式函数（`connectSSE`/`NewChatStream`/`ChatWithAgent`/`SummaryStockNews`）错误与中断分支统一清理 | `frontend/src/services/wails-bridge.js` |
| 6 | **skillPlaza 可用性**：新增同源代理 `POST /api/v1/plaza/proxy`（带 SSRF 校验），Web 模式不再直连外部 HTTP；`skillPlaza.vue` 增加本地技能降级与 HTTPS 自适应 apiBase | `plaza_handler.go`/`server.go`/`wails-bridge.js`/`skillPlaza.vue` |

### 🟡 P2
| # | 修复 | 文件 |
|---|------|------|
| 7 | CORS 改为 `GO_STOCK_CORS_ORIGINS` 白名单，移除 `*`+凭据组合 | `server.go` |
| 8 | **登出真实生效**：token 增加 `jti`，`Logout` 将 jti 加入黑名单，`ParseToken` 校验吊销 | `auth_middleware.go`/`auth_handler.go` |
| 9 | 登录/注册加 IP 限流（8 次/15 分钟）；注册错误统一为「用户名或邮箱已被注册」，消除用户枚举 | `middleware/ratelimit.go`/`server.go`/`auth_handler.go` |
| 10 | 广场凭据：`SyncPlazaData` 改 JSON body 传参（向后兼容 query）；定时任务密码经 AES-GCM 加密落库，执行时解密；**删除 cron 里硬编码的默认账号 Joy/zgc@202123** | `plaza_handler.go`/`backend/security/crypto.go`(新)/`backend/agent/cron_task_api.go` |
| 11 | GET 写副作用：`/stock-changes/all` 落库需 `?save=1`；`GetHotWords` 爬取需 `?refresh=1` | `web_handler.go` |
| 12 | SSRF：新增 `isSSRFBlockedHost`（解析真实 IP，覆盖 10/8、172.16/12、192.168/16、100.64/10、198.18/15、169.254/16、环回/链路本地/IPv6 私有），替换原前缀匹配 | `web_handler.go` |
| 13 | 技能与 MCP 的写操作（创建/更新/删除/启用/导入/测试）加 `AdminRequired`，读操作保持开放 | `server.go` |
| 14 | `ImportSkills` 限制请求体 10MB（`http.MaxBytesReader`） | `web_handler.go` |
| 15 | 移除 `FixAdminRole` 路由并删除 `fix_admin.go` | `server.go`/`fix_admin.go`(删除) |

### 🟢 P3
| # | 修复 | 文件 |
|---|------|------|
| 16 | 删除死代码 `vip_http.go`、`wails_compat.go` | 已删除 |
| 17 | 移除生产 debug 输出（`admin_handler.go` println、`stock_group_handler.go` fmt.Printf） | 对应文件 |
| 18 | `GetTelegraphList`/`RefreshTelegraphList` 增加 nil 守卫，消除 panic 风险 | `web_handler.go` |
| 19 | ⚠️ `GetEMDictCode` 占位端点**保留**（前端依赖其契约，改为真实实现需产品确认） | `web_handler.go` |
| 20 | 配置保存改为 `error` 返回，去掉 `!= "保存成功！"` 魔法字符串（`UpdateConfigByUserID`/`UpdateAiConfigsByUserID` 签名变更） | `backend/data/settings_api.go`/`setting_handler.go`/`ai_handler.go` |
| 21 | Web 模式检测改为挂载后复查 + 短暂轮询，避免 `window.go` 异步就绪导致永久误判 | `frontend/src/composables/useResponsive.js` |
| 22 | ⚠️ `ShareText`/`ShareAnalysis` 占位**保留**（无后端分享服务，改契约会破坏前端分享交互） | `web_handler.go` |

### 补充修复（第三轮审查发现，本次一并处理）
| 问题 | 修复 | 文件 |
|------|------|------|
| **P0** `User.Password` 带 `json:"password"` → 管理员用户列表泄露 bcrypt 哈希 | 改为 `json:"-"`（GORM 用 gorm 标签，读写不受影响） | `backend/models/user.go` |
| **P0** `SyncPlazaData` 用客户端传入 apiBase 请求外部，无 SSRF 校验 | 新增协议与 `isSSRFBlockedHost` 校验 | `plaza_handler.go` |
| **P1** `/groups/:id/stocks` 与 `c.Param("groupId")` 不匹配 → 接口恒 400 | 改为 `c.Param("id")` | `stock_group_handler.go` |
| **P1** 只改用户名也会因 `updates["is_active"]=req.IsActive` 误禁用用户 | `IsActive` 改 `*bool`，仅显式传参才更新 | `admin_handler.go` |
| **P1** 注册失败不计数 → 注册限流形同虚设 | 冲突分支调用 `RecordLoginFailure` | `auth_handler.go` |
| **P1** 广场同步任务未设 `UserID` → 建出 user_id=0 全局任务 | 创建时写入当前 userID | `plaza_handler.go` |
| **P1** resty 跟随 302 可绕过 `isSSRFBlockedHost` | 全部加 `resty.NoRedirectPolicy()` | `web_handler.go`/`plaza_handler.go` |
| **P1** 可信代理未配置 → `X-Forwarded-For` 可伪造绕过限流 | 新增 `GO_STOCK_TRUSTED_PROXIES`；未配置则 `SetTrustedProxies(nil)` 不信任代理头 | `server.go` |

> 说明：`AIConfig.ApiKey` / `Settings.TushareToken/FeishuSecret/FeishuAppSecret` 仍会随 `/config` 返回。
> 这些是**当前用户自己的**配置（已按 user_id 隔离），且设置页需要回显已保存的密钥；
> 若直接置 `json:"-"` 会导致「读取→编辑→保存」把密钥清空（数据丢失）。
> 建议后续改为返回脱敏值 + 单独的输入字段，本次未改以避免回归。

### 部署前必须配置的环境变量
```
GO_STOCK_JWT_SECRET=<32字节以上随机串>       # 未设置则每次启动随机，重启后全部登录态失效
GO_STOCK_ADMIN_PASSWORD=<初始管理员密码>      # 空库首次启动创建管理员所必需
GO_STOCK_SECRET_KEY=<32字节>                 # 广场凭据加密密钥（未设置用内置开发密钥，不安全）
GO_STOCK_CORS_ORIGINS=https://your-domain     # 跨域前端源，逗号分隔；未设置则仅同源
GO_STOCK_TRUSTED_PROXIES=10.0.0.0/8          # 仅当部署在反代/Nginx 后时配置，否则留空
```

[COMPLIANCE]
  Phase: 6 (Review) → 修复已执行
  模式: Strict
  门控状态: P0 已解除；P1 全部处理；#19/#22 为占位端点，改契约需产品确认（已标注）
  产出记录: .devflow/web-review/06-review.md
  验证: go build ./internal/... ./backend/... → EXIT=0；node --check 前端改动文件通过
  构建提示: 需 GOCACHE/GOTMPDIR 指向空间充足磁盘（C 盘已满）
  强制检查点: R0 R1 R2 R4 R5 R6 R7
