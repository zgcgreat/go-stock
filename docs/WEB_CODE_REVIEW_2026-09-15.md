# go-stock Web 端代码审查报告

- **审查时间**：2026-09-15
- **审查范围**：`internal/webserver/server.go`、`internal/middleware/*`（4 文件）、`internal/handlers/*`（14 文件）、`internal/migrations/`、`backend/agent/cron_task_api.go`、`backend/data/{settings_api,daily_operation_plan_api,tool_daily_operation_plan}.go`、`frontend/src/services/{api,wails-bridge}.js`
- **审查维度**：认证鉴权 / 用户隔离(IDOR) / 密钥与敏感信息 / SSRF / 注入 / 限流与 DoS / 并发与资源 / 响应契约 / 前端一致性
- **结论**：发现 **3 个 P0、12 个 P1、15 个 P2**。用户隔离（userID 过滤）整体做得扎实，主要风险集中在**敏感信息明文下发**、**SSRF** 与**限流失效**三块。

---

## 一、总体结论（先看这里）

| 维度 | 评级 | 一句话结论 |
|------|------|-----------|
| 用户隔离 / IDOR | 🟢 良好 | 14 个 handler 基本都带 `user_id` 过滤，CronTask / 分组 / 自选股 / 交易记录实现规范 |
| 敏感信息保护 | 🔴 严重 | 密码哈希、AI API Key、飞书/钉钉密钥全部明文下发前端 |
| 认证与会话 | 🟡 一般 | JWT 实现正确，但账号禁用不即时生效、仅登录接口有限流且可被绕过 |
| SSRF | 🔴 严重 | 提示词广场同步接口无任何内网校验；AI 模型接口防护可被重定向绕过 |
| 输入校验 / 注入 | 🟢 良好 | 全部使用 GORM 参数化查询，未发现 SQL 注入 |
| 限流与 DoS | 🟠 较弱 | 仅 login/register 有限流且可绕过；AI 与爬虫接口无任何配额 |
| 并发与资源 | 🟡 一般 | SSE 无超时、定时任务可无限触发 goroutine |
| 响应契约 | 🟠 不一致 | 大量错误用 HTTP 200 + `code:1` 返回 |

---

## 二、P0 — 必须立即修复（3 项）

### P0-1 管理员用户列表接口泄露 bcrypt 密码哈希

**证据**
- `backend/models/user.go:13` — `Password string \`json:"password" gorm:"not null"\`` ← **没有 `json:"-"`**
- `internal/handlers/admin_handler.go:184-194` — `AdminUserList` 直接 `c.JSON(..., gin.H{"list": users})`

**影响**：`GET /api/v1/admin/users` 响应中每个用户都带 `password` 字段（bcrypt 哈希）。任何管理员账号泄露 / 越权访问即可批量拿到全库密码哈希离线爆破。

**修复**
```go
// backend/models/user.go
Password string `json:"-" gorm:"not null"`
```
同时建议 `AdminUserList` 改用专门的 DTO（`AdminUserVO`）而非直接返回 `models.User`，避免以后新增字段再次误泄。

---

### P0-2 AI API Key / TushareToken / 飞书密钥明文下发到浏览器

**证据**
- `backend/data/settings_api.go:75` — `ApiKey string \`json:"apiKey"\``
- `backend/data/settings_api.go:18,24,28` — `TushareToken` / `FeishuSecret` / `FeishuAppSecret` 均无 `json:"-"`
- `internal/handlers/ai_handler.go:270` — `GetAIConfigs` 返回 `settingConfig.AiConfigs`（含 apiKey）
- `internal/handlers/setting_handler.go:286` — `GetAppConfig` 返回整个 `SettingConfig`
- `internal/handlers/setting_handler.go:315` — `UpdateAppConfig` 保存后原样回显 `config`

**影响**：任何登录用户请求 `GET /api/v1/config` 即可拿到自己配置的全部明文密钥，并经 `UpdateAppConfig` 再次回显。密钥一旦进入浏览器内存 / DevTools / 前端错误上报 / 浏览器扩展，即视为已泄露。若 XSS 或恶意第三方脚本存在，等于密钥托管给攻击者。

**修复**
1. 后端增加只读视图模型，密钥字段改为掩码返回（如 `sk-****abcd`），或 `json:"-"` + 单独提供"仅写入"语义：前端传空字符串表示不修改。
2. 若前端确需展示，返回 `hasApiKey: bool` 而非明文。

---

### P0-3 提示词广场同步接口 SSRF（无任何内网校验）

**证据**
- `internal/handlers/plaza_handler.go:192-227` — `SyncPlazaData` 接受客户端传入的 `apiBase`，直接 `data.NewPlazaSyncService(req.ApiBase)` 发起请求
- 对比 `FetchAiModels`（`web_handler.go:784`）有 `isSSRFBlockedHost` 校验，**这里完全没有**

**影响**：任意登录用户可让服务器向 `http://127.0.0.1:*/`、`http://169.254.169.254/`（云元数据）、内网 Redis/未授权服务发起请求；响应错误信息 `"登录提示词广场失败: " + err.Error()` 还会把内网响应特征回显，形成可用的盲/半盲 SSRF 探针。

**修复**
1. 复用 `isSSRFBlockedHost` 对 `apiBase` 做校验；
2. 加白名单（只允许官方广场域名）或改为服务端配置项，禁止客户端传入；
3. 错误信息统一为"同步失败"，不回显 `err.Error()`。

---

## 三、P1 — 高风险（12 项）

### P1-1 gin 未设置 TrustedProxies → 登录限流可被 X-Forwarded-For 伪造绕过
`internal/webserver/server.go:48` 用 `gin.New()`，未调用 `SetTrustedProxies`。gin 默认信任所有代理，`c.ClientIP()` 会优先取 `X-Forwarded-For`，攻击者每次改一个 header 值即可重置限流计数。
**修复**：`ws.router.SetTrustedProxies([]string{"127.0.0.1"})`（或真实网关 IP 段）。

### P1-2 注册接口限流完全失效
`internal/middleware/ratelimit.go:30` 的 `LoginRateLimit` **只在进入时检查计数，从不自增**；自增依赖 handler 调 `RecordLoginFailure`。`Login` 三个失败分支都调了（`auth_handler.go:302,315`），但 **`Register`（`auth_handler.go:220-289`）所有失败分支都没调** → 计数永远为 0 → 限流形同虚设，可无限次注册探测用户名（用户枚举）。
**修复**：`Register` 的"用户名已存在/邮箱已存在"分支补调 `middleware.RecordLoginFailure(c.ClientIP())`；更好的做法是让中间件在响应后根据 status code 自增。

### P1-3 禁用账号后 Token 最长 24 小时仍有效
`internal/middleware/auth_middleware.go:133` 的 `AuthRequired` 只解析 JWT，不查库校验 `is_active` / `role`。管理员"禁用用户"或"降权"后，该用户已签发的 token 在过期前依然可用（管理员降权后原管理员仍能进后台）。
**修复**：`AuthRequired` 中查一次用户（可加 60s 本地缓存），校验 `IsActive` 且角色未被降级；或维护"用户版本号"放进 JWT claims。

### P1-4 `UpdateUserInfoHandler` 会把用户强制禁用（逻辑 Bug）
`internal/handlers/admin_handler.go:479` — `updates["is_active"] = req.IsActive` **无条件写入**。`req.IsActive` 是 `bool`，请求体不传该字段时为零值 `false` → 管理员调用 `PUT /admin/users/:id` 只改一个用户名，就会顺带把该用户禁用。
**修复**：改为 `*bool` 指针，仅非 nil 时写入。

### P1-5 管理员可随意提权为 super_admin，无自我保护
`admin_handler.go:500`（`UpdateUserRoleHandler`）、`460-482`（`UpdateUserInfoHandler`）均允许把任意账号（含自己/其他 super_admin）的角色改成 `super_admin`，也没有"不能修改同级或更高级账号"的约束。`DeleteUserHandler` 有"不能删自己"保护，但角色/密码接口没有。
**修复**：引入角色等级比较，禁止修改 ≥ 自身等级的账号。

### P1-6 除登录注册外全局无限流
`server.go` 中仅 `auth.POST("/register")` / `auth.POST("/login")` 挂了 `LoginRateLimit`。`POST /ai/analyze`、`POST /ai/agent-chat`（消耗 AI token 费用）、`/market/*` 爬虫类接口（消耗外部 API 配额、可能触发封禁）、`POST /cron-task/:id/execute` 均无限制。
**修复**：增加按 userID 的全局令牌桶（AI 类严格，查询类宽松）。

### P1-7 API Key 经 URL query 传递 → 明文进入访问日志
- `internal/handlers/web_handler.go:769-770` — `FetchAiModels` 从 `c.Query("apiKey")` 读取
- `frontend/src/services/wails-bridge.js:1492,1499` — 前端确实以 query 方式发送
- `gin.Logger()`（`server.go:49`）会打印完整请求 URI → **AI API Key 明文落盘日志**
- 同类问题：`plaza_handler.go:200-208` 的 `SyncPlazaData` 注释已承认"不推荐，凭据会进入访问日志"，但仍保留 query 回退

**修复**：全部改为 POST + JSON body；或至少给 gin.Logger 加 URI 脱敏 Formatter。

### P1-8 resty 默认跟随重定向 → AI 接口 SSRF 防护可绕过
`web_handler.go:796-801` 与 `853-858` 的 resty client 未设置 `SetRedirectPolicy`。`isSSRFBlockedHost` 只在首次请求前校验，攻击者用公网域名返回 302 跳转到 `169.254.169.254` 即可绕过。此外存在 TOCTOU/DNS rebinding 窗口（校验时解析一次，resty 实际连接时再解析一次）。
**修复**：
```go
client.SetRedirectPolicy(resty.RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
    if isSSRFBlockedHost(req.URL.Hostname()) { return errors.New("blocked") }
    return nil
}))
```
更彻底的做法：自定义 `http.Transport.DialContext`，在 dial 层对解析出的 IP 做校验（消除 rebinding）。

### P1-9 `CreatePlazaSyncCronTask` 未绑定 UserID → 任意用户创建全局任务
`internal/handlers/plaza_handler.go:300-317` 构造 `models.CronTask` 时**没有设置 `UserID`**（对比 `CreateCronTask` 在 `web_handler.go:462` 有 `task.UserID = userID`）。后果：
1. 任意登录用户可创建 `user_id=0` 的全局任务，污染桌面端/系统级任务空间；
2. `ExistsByTaskType` 是全局检查 → 任一用户创建后，其他所有人都收到 409，形成持久 DoS；
3. 普通用户用 `GetCronTaskList`（带 userID 过滤）看不到自己刚创建的任务，无法管理；
4. 创建后未调用 `CronSchedulerHooks.AddCronTask`（对比 `web_handler.go:471`），任务要等重启才进调度器。
**修复**：`task.UserID = userID`；`ExistsByTaskType` 增加 userID 维度；创建后注册到调度器；接口建议加 AdminRequired。

### P1-10 `GetGroupStockList` 路由参数名不匹配 → 接口 100% 返回 400
- `internal/webserver/server.go:159` — `protected.GET("/groups/:id/stocks", handlers.GetGroupStockList)`
- `internal/handlers/stock_group_handler.go:239` — `groupIDStr := c.Param("groupId")` ← 恒为空串
- 结果：`strconv.ParseUint("")` 必失败，永远返回 `{"code":1,"message":"无效的分组ID"}`

**修复**：改为 `c.Param("id")`（或把路由改成 `:groupId`）。这是确定的功能 Bug，建议立即验证：`GET /api/v1/groups/1/stocks` 必然 400。

### P1-11 `ExecuteCronTaskNow` 无并发/频率控制
`web_handler.go:542` 每次请求都 `go func(){ ExecuteTask }()`，无同一任务去重、无并发上限、无用户级限流。任务类型包含股票分析（调 AI）与新闻抓取（调外部 API），狂点即可打爆 AI 额度 / 触发外部封禁 / 打满 goroutine。
**修复**：加单飞（同任务 ID 执行中直接返回）+ 用户级并发信号量。

### P1-12 AI 工具可跨用户修改他人"每日操作计划"
`backend/data/tool_daily_operation_plan.go:507,514`：
```go
existing, err := NewDailyOperationPlanApi().GetDailyOperationPlanByID(planID)        // 未传 userID
...
if err := NewDailyOperationPlanApi().UpdateDailyOperationPlanStatus(planID, status)  // 未传 userID
```
两个调用都没传 `userID`，`firstUserID` 返回 0 → 走"不过滤"分支。任意登录用户可在 Agent Chat 中诱导 AI 按 ID 枚举并修改其他用户的数据（ID 自增可直接遍历）。
**修复**：从 `ToolContext` 取 userID 透传（需确认 `ToolContext` 是否已带 UserID 字段，当前 `tool_registry.go` 中未见使用）。

### P1-13 删除用户不级联清理（补充）
`admin_handler.go:86` 的 `DeleteUser` 只删 `users` 记录，该用户的自选股/分组/交易记录/AI 配置/会话/定时任务/设置全部残留为孤儿数据（`user_id` 悬挂）。
**修复**：事务内级联清理，或改为软删除 + 保留数据（当前已有 `DeletedAt` 软删字段，硬删反而绕过了它）。

---

## 四、P2 — 中风险（15 项，摘要）

| # | 问题 | 位置 |
|---|------|------|
| 1 | 错误统一返回 HTTP 200 + `code:1`，前端/监控无法用状态码判定失败 | `web_handler.go` cron 全套（509/520/538/566/590/622 等） |
| 2 | 错误信息直接回显 `err.Error()`，泄露 SQL/内部细节（≥20 处） | `web_handler.go:466,538,566,590`；`admin_handler.go:305`；`ai_handler.go:471,506` |
| 3 | `pageSize` 无上限（可传 10^9 拉全表 → 内存 DoS） | `admin_handler.go:155`；`ai_handler.go:145`；`setting_handler.go:40`；`prompt_handler.go:24` |
| 4 | `LIKE` 查询未转义 `%` `_`（可控通配符 → 全表匹配） | `admin_handler.go:68`；`ai_handler.go:158`；`stock_group_handler.go:36` |
| 5 | `AdminRequired` 每请求 5 条 `log.Printf`，且打印 username/role 到日志 | `middleware/admin.go:14,28,41,50`；`role.go:75,78,80` |
| 6 | `RelearnUserProfile` 返回裸字符串 `c.JSON(200, v)`，破坏统一响应格式 | `web_handler.go:2228` |
| 7 | `SaveUserProfileHandler` `io.ReadAll` 无 body 大小限制（对比 `ImportSkills` 有 10MB 限制） | `web_handler.go:2165` |
| 8 | CORS `AllowOrigins` 为空 + `AllowCredentials=true`，需实测确认库行为（若回退 `*` 则构成"通配符+凭据"漏洞） | `server.go:52-60` |
| 9 | `http.Server` 无 `ReadTimeout`/`WriteTimeout`/`MaxHeaderBytes` → Slowloris 风险 | `server.go:436` `router.Run` |
| 10 | `UpdateUserProfile` 不校验 email 唯一性，可把邮箱改成与他人重复 | `auth_handler.go:432-438` |
| 11 | 注册无密码强度/邮箱格式校验，开放注册无开关 | `auth_handler.go:220-264` |
| 12 | 广场同步任务 `params` 用 `fmt.Sprintf` 拼 JSON，username 含引号即破坏结构 | `plaza_handler.go:305` |
| 13 | JWT 24h 无刷新机制；`GO_STOCK_JWT_SECRET` 未配置时随机生成，重启即全体掉线 | `auth_middleware.go:24-36,94` |
| 14 | Logout 只吊销当前 token；改密/禁用/删号均不吊销已签发 token | `auth_handler.go:356` |
| 15 | SSE 无超时/心跳，`flusher.Flush()` 错误被忽略，客户端假死时资源不释放 | `ai_handler.go:97-119,384-426` |

---

## 五、P3 — 低风险/清理项

- `ai_handler.go:645` `marshalEvent` 定义后从未使用（死代码）
- `stock_follow_handler.go:97,107` 使用 `fmt.Printf` 而非项目统一 logger，日志无法落文件/分级
- `web_handler.go:393` `GetSponsorInfo` 只校验 `VipEndAt` 未校验 `VipStartAt`，理论上存在"未生效即享受"窗口
- 前端 JWT 存 `localStorage`（XSS 可窃取）——业界通用权衡，若需强化可改 `httpOnly` Cookie + 后端 CSRF 防护
- `web_handler.go:646-653` `ShareText` / `ShareAnalysis` 是空实现，恒返回"分享成功"（前端若有分享入口会误导用户）

---

## 六、建议修复顺序

```
第 1 批（立刻，半天）
  P0-1  models.User.Password 加 json:"-"
  P0-2  AIConfig.ApiKey / Settings 密钥字段改掩码或 json:"-" + DTO
  P0-3  SyncPlazaData 接入 isSSRFBlockedHost + 错误脱敏
  P1-10 GetGroupStockList 参数名 :id → groupId（一行修复，当前功能全废）

第 2 批（本周）
  P1-1 SetTrustedProxies        P1-2 注册失败计次
  P1-3 AuthRequired 校验 is_active/role
  P1-4 is_active 改 *bool        P1-7 密钥改 POST body
  P1-8 resty 禁重定向 + dial 层校验

第 3 批（下周）
  P1-5 角色等级保护   P1-6 全局限流   P1-9 广场任务 userID
  P1-11 任务执行单飞  P1-12 AI 工具透传 userID  P1-13 级联清理

第 4 批（技术债）
  P2 全量：响应契约统一、pageSize 上限、错误脱敏、日志收敛
```

---

## 七、做得好的地方（避免误伤，勿改）

- **用户隔离实现扎实**：`CronTaskApi.Update/Delete/GetByID/EnableTask`（`backend/agent/cron_task_api.go:46,54,63,136`）统一 `if userID > 0 { Where("user_id = ?") }` 模式，兼顾桌面端 `user_id=0`；
- `stock_group_handler.go` / `stock_follow_handler.go` / `prompt_handler.go` / `fund_handler.go` 的写操作全部先查 `id = ? AND user_id = ?` 再改，无 IDOR；
- `DailyOperationPlanApi` 用 `userID ...uint` 可变参数 + `firstUserID` 兼容桌面端，更新路径同样带 owner 校验（`daily_operation_plan_api.go:110-122`）；
- **密码用 bcrypt，初始管理员不写死默认密码**（`admin_handler.go:108-125` 必须配置 `GO_STOCK_ADMIN_PASSWORD` 才创建）；
- 全量使用 GORM 参数化查询，未发现 SQL 注入；
- `plaza_handler.go:281` 凭据落库前用 `security.EncryptString` 加密，意识到位；
- `web_handler.go:723-765` 自建 `isSSRFBlockedHost` 覆盖 CGNAT / 基准网络 / 链路本地，覆盖面比常见实现更全。
