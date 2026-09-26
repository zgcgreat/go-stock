# go-stock Web 端代码审查报告

> 审查日期：2026-09-15
> 审查范围：`internal/webserver`、`internal/handlers`、`internal/middleware`、`frontend/src/services`（Web 模式桥接）、`frontend/src/utils/auth.js`、`frontend/src/services/api.js`
> 审查方法：devflow-kit 健康检查（Health Check）多维代码审查，逐文件实证 + grep 量化隔离覆盖

## 一、审查维度（自定）

| 维度 | 说明 |
|------|------|
| ① 认证与路由守卫 | JWT 签发/校验、中间件、路由分组鉴权覆盖 |
| ② 授权与权限 | 管理员/VIP 门禁、敏感接口的调用者限制 |
| ③ 用户数据隔离 | 各 handler 是否按 `user_id` 过滤私有数据 |
| ④ 凭据与密钥安全 | 密钥硬编码、明文存储、token 存储位置 |
| ⑤ 接口健壮性与响应契约 | 错误处理、分页、SSE 流式、重复实现 |
| ⑥ 前端 Web 模式桥接 | 双模式降级、token 注入、SSE、跨标签登出 |

## 二、总体结论

**路由鉴权覆盖完整、用户隔离覆盖面广（11+ 类私有数据均按 user_id 过滤），密码使用 bcrypt 正确哈希——这是良好的基础。** 但存在 **1 个严重（Critical）密钥硬编码问题** 和若干中高危授权/凭据处理缺陷，集中在「广场同步」与「CORS/JWT 配置」两处，建议优先修复。

## 三、问题清单（按严重度）

### 🔴 Critical

**C1. JWT 签名密钥硬编码在源码中**
- 位置：`internal/middleware/auth_middleware.go:16`
- 代码：`JWTSecret = "go-stock-jwt-secret-key-2026-change-in-production"`
- 影响：任何拿到二进制/源码的人都能离线伪造任意用户（含管理员 `super_admin`）的 token，绕过全部登录与鉴权。配合开放注册与 24h 有效期，危害极大。
- 建议：改为从环境变量/配置文件读取（如 `os.Getenv("JWT_SECRET")`），启动时缺失则拒绝启动；并提供首次运行随机生成并持久化的机制。

### 🟠 High

**H1. 提示词广场同步接口未加管理员门禁 + 明文存储凭据**
- 位置：`internal/handlers/plaza_handler.go`
  - `SyncPlazaData`（`:163`，路由 `POST /plaza/sync`）——仅 `AuthRequired`，任意登录用户可用任意账号触发同步，覆盖共享缓存表。
  - `CreatePlazaSyncCronTask`（`:218`，路由 `POST /plaza/sync/cron-task`）——同样仅 `AuthRequired`，且 `Params: fmt.Sprintf(\`{"username":"%s","password":"%s"}\`, ...)`（`:253`）**将密码明文写入 `cron_tasks` 表**。
- 影响：
  1. 越权：普通用户可创建/修改系统级同步任务；
  2. 凭据泄露：广场账号密码以明文落地数据库，DBA 或任意能读库的人可直接获取；
  3. 设计冲突：记忆中明确「同步应使用默认账号 Joy/zgc@202123 的定时任务」，不应暴露给用户自选凭据。
- 建议：
  - 这两个接口移入 `admin` 分组（`AdminRequired()`）；
  - 凭据改由服务端配置/密钥管理提供，禁止从请求体接收并明文落库；
  - 若必须落库，至少做加密存储。

**H2. CORS 通配源 + 凭证共存（反模式）**
- 位置：`internal/webserver/server.go:52-59`
- 代码：`AllowOrigins: []string{"*"}` 同时 `AllowCredentials: true`
- 影响：gin-contrib/cors 在检测到「通配源 + 凭证」组合时要么启动期报错，要么运行时反射请求 Origin——等于对任意源开放带凭证的跨域请求，存在 CSRF / 跨站数据读取风险。
- 建议：生产环境显式白名单化前端部署源（如 `[]string{"https://your.domain"}`）；同源部署时可直接移除 CORS 中间件（Web 静态资源与 API 同源无需 CORS）。

### 🟡 Medium

**M1. VIP 内容仅在客户端判断，服务端未强制**
- 位置：`internal/handlers/plaza_handler.go` `GetPlazaPrompts`/`GetPlazaPromptDetail`（`:17`/`:60`）
- 现象：前端 `promptPlaza.vue` 用本地 `isLocalVip` 遮罩 VIP 内容，但服务端 `GetPlazaPromptDetail(extId)` 不校验调用者 VIP 等级，`vipOnly` 也仅是客户端下发的过滤参数（可省略）。
- 影响：非 VIP 用户直接调 API 即可获取 VIP 专属提示词原文。
- 建议：若 VIP 内容需真正受限，服务端按 `user.VipLevel/VipEndAt` 强制过滤 `vipOnly` 记录；若仅为 UI 便利，则在文档/接口注释明确「非安全边界」。

**M2. Token 存于 localStorage（XSS 可窃取）**
- 位置：`frontend/src/utils/auth.js:14`、`frontend/src/services/wails-bridge.js:18`、SSE 函数（`:116`/`:687`/`:1282`）
- 影响：SPA 中任何 XSS 都能读取 `localStorage.token` 冒用身份；叠加 C1（密钥硬编码）服务端侧也可被伪造，鉴权面整体脆弱。
- 建议：短期可在关键写操作接口增加二次校验；长期建议改用 httpOnly+SameSite Cookie 或短时 token + 刷新机制。

**M3. 孤儿/重复认证实现，行为不一致**
- 位置：`internal/handlers/auth_handler.go`
  - `HandleLogin`(`:44`)、`HandleRegister`(`:111`)、`HandleGetUserProfile`(`:167`) 为 `net/http` 写法；
  - `server.go` 实际接线的是 Gin 版 `Login`/`Register`/`GetUserProfile`。
- 现象：两套实现并存。net/http 版 `HandleRegister` **不校验邮箱唯一、注册成功不返回 token**；Gin 版两者都做。VIP 检查逻辑也分处 `middleware/vip_http.go`（`http.Handler`）与 `middleware/admin.go`/`role.go`（Gin）两套。
- 影响：维护隐患 + 潜在行为分叉。
- 建议：删除 net/http 版 `Handle*` 孤儿函数，统一走 Gin handler；`vip_http.go` 的 `RequireVipRole/RequireAdminRole` 若未在路由中接线则一并清理。

**M4. 开放注册（自托管场景需评估）**
- 位置：`internal/handlers/auth_handler.go:220` 路由 `POST /auth/register`（仅 `AuthRequired`？实际在 `/auth` 组，无需登录即可注册）
- 影响：互联网暴露部署时，任何人可自助注册并访问共享资源/触发爬取类接口。
- 建议：内网/私有部署可保留；公网部署应加注册开关/邀请码/管理员审批。

### 🟢 Low

**L1. 生产环境调试日志过多**
- 位置：`middleware/admin.go`、`middleware/role.go`、`middleware/vip_http.go` 大量 `log.Printf("[AdminRequired] userID=...")`、`[IsAdmin] ...`
- 影响：每次鉴权打印用户 ID/角色，日志噪声且轻微信息泄露。
- 建议：降级为 `Debug` 级别或移除。

**L2. 跨标签登出事件为空操作**
- 位置：`frontend/src/utils/auth.js:23-29` `clearToken` 中 `setItem` 后立即 `removeItem(LOGOUT_EVENT_KEY)`，该 key 的 storage 事件永不被触发；真正生效的是 `onCrossTabLogout` 监听 `token` key 的移除（可行）。
- 建议：删除无用的 `LOGOUT_EVENT_KEY` set/remove 片段，避免误导。

**L3. 前端桥接残留调试输出**
- 位置：`wails-bridge.js` 中 `GetFollowList`(`:253`)、`GetAllStocks`(`:1904`) 等 `console.log` 原始响应。
- 建议：移除非必要 `console.log`，保留 `console.warn/error` 降级日志。

**L4. SSE 通用辅助 `connectSSE` 提前结束风险**
- 位置：`wails-bridge.js:158-162` —— 当流式 JSON 含 `chatId` 字段即 `EventsEmit(doneMsg)` 并 return，可能误判为完成。`NewChatStream`/`ChatWithAgent` 已各自实现独立 SSE，故影响有限。
- 建议：统一 SSE 解析逻辑，或明确 `connectSSE` 的适用边界。

**L5. 概念标签为全局共享（设计确认项）**
- 位置：`backend/data/stock_concept_api.go:16` `Concept` 模型无 `user_id`；`stock_concept_handler.go` 全部写操作无用户隔离。
- 说明：符合「公共数据不隔离」原则，但意味着多用户下任一人 `RemoveConcept`/`UpdateConcept` 会影响全局标签。若概念应属用户私有，则需补 `user_id` 隔离；若共享，建议在 UI 提示「共享标签，删除影响所有用户」。

## 四、已确认的正面项（非问题）

- ✅ 路由分组鉴权完整：`protected` 全组 `AuthRequired()`，`admin` 子组 `AdminRequired()`，仅 `/health`、`/auth/login`、`/auth/register` 未鉴权（符合预期）。
- ✅ 用户隔离覆盖面广：`web_handler/auth/setting/ai/daily/fund/group/follow/prompt/stock/trade` 等 handler 均通过 `GetUserIDFromContext`/`getUserID()` 取用户 ID 过滤私有数据（grep 量化确认）。
- ✅ 密码使用 `bcrypt.GenerateFromPassword` + `CompareHashAndPassword`，无明文存储。
- ✅ 前端 Web 模式双栈降级结构清晰（`isWailsMode()` 分支 + axios 拦截器统一注入 Bearer）。
- ✅ SSE 专用实现（`NewChatStream`/`ChatWithAgent`）具备 abort、done、错误分支处理。

## 五、修复优先级建议

1. **立即**：C1（JWT 密钥外置）、H1（广场同步接口加 AdminRequired + 去明文密码）。
2. **上线前**：H2（CORS 白名单）、M1（VIP 服务端强制或明确非安全边界）、M2（token 存储策略）。
3. **清理期**：M3（删孤儿 net/http handler）、M4（注册策略）、L1–L5（日志/事件/SSE 细节）。

---
*本报告基于静态代码审查，未执行运行时渗透测试。建议对 C1/H1 做针对性复测。*
