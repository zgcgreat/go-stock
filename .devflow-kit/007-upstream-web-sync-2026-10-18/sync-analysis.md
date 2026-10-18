# 007 上游同步（809b5f9..88bbea5，2026-10-18）

上游区间：`809b5f9..88bbea5`，共 24 个有效提交（v2026.10.11.1 ~ v2026.10.18.1，2026-10-11 ~ 10-18）。
执行前基线：dev-web == 006 同步完成态（15f5760）。

## 同步决策（用户确认）

1. **OKX 交易套件完整接线**——后端全取 + Web REST 端点 + 前端面板挂载 + 桥接层。
2. **悬浮助手整体采用上游重构**（098c333）——本地 eb5fdd2 平行实现被上游等价替代。
3. **桌面根包跳过**（app.go/app_okx.go/app_signal_monitor.go/main.go，与 006 惯例一致）。

## 提交清单

| 提交 | 内容 |
|------|------|
| f4ca251 | A/B0-B3：agent 核心+代理重构+OKX 数据层+信号 v2 后端+Web 接线+前端三实例引擎+桥接层 |
| 34f16eb | B4：悬浮助手/agent-chat 整体上游版+多会话协议接线+skill_handler 平移+settings OKX 区块 |
| 683f417 | B5：go 1.27.2 + x/net v0.60.0 |

## 处理分层

### A. 直接同步（本地零偏离，checkout origin/dev）
- agent.go（内置工具常驻恢复 6ae2c03 + MCP 并行加载/失败退避 1cf1a85）、builtin_models.go（2026-10 模型表 5f83b40）、
  chat_model_factory.go、mcp_tools_cache.go、agent_context.go、builtin_models_test 等
- tools：common.go(stripHTMLForAI)、holiday_tool.go（年度表缓存 73d4dcf）、tool_groups.go（OKX 组）、market_news_tool.go
- holiday_cache.go、httpclient.go（HttpProxyMode 代理作用范围 c0f43ef + CreateProxiedHTTPClient 家族）、
  market_news_api.go、download_progress.go、image_hosting_api.go、proxy_acceleration.go（节点源自建反代 88bbea5）
- OKX 数据层 9 文件全量（6562395/fdfab0a/6c57d67/54e0910）+ 全部测试
- 前端新文件：OKX 三面板、agent/ 5 个 composables（format/MessageGroup/streamParse/useAgentSessions/useAgentStream）
- signalMonitor.ts（三实例引擎 f455797+MA144 54e0910）、SignalMonitorPanel.vue（三页签+交易流水）、
  aiRecommendStocksList.vue（信号快照 54e0910）
- FloatingAgentAssistant.vue、agent-chat.vue 整体上游版

### B. 手工合并（本地有用户隔离偏离，保留 userID 变体）
- settings_api.go：OKX 30 字段+HttpProxyMode+SignalScanIntervalSec+PromptPlazaApiBases 主备回退；
  UpdateConfigByUserID 同步补全字段映射
- signal_record_api.go：Envs 环境合并去重+applySourceFilter+加密空单统计；userID 全链路保留
  （唯一索引/ON CONFLICT/预查询过滤/GetPage/Stats/Clear）
- ai_assistant_api.go：重命名/置顶/标题搜索，全部带 userID 隔离
- openai_tools.go：上游 SSE 内嵌错误帧+max_completion_tokens 降级重试全套；SaveAIResponseResult 保留 userID 变长参数
- models.go：AiAssistantSession Title/Pinned（保留 UserID）
- sina_kline_api.go：ok: 前缀 OKX K线短路（保留本地 23be366 海外指数腾讯修复）
- ai-assistant-web/server.go：广场上传主备回退（84ce588）——保留 GetAiAssistantSession(0) 隔离签名

### C. Web 端接线（internal/ 新增）
- okx_handler.go：15 个 REST 端点（trading-overview/leverage/candles/tickers/ticker/rankings/wallet/
  positions/orders/fills/下单/撤单/平仓/account-config/exchange-announcements）
- signal_handler.go：source 参数、交易流水分页/清空、扫描间隔 GET/PUT
- event_handler.go：节拍循环支持热重启（RestartSignalTickInterval）
- skill_handler.go：buildSkillContext 家族平移（Web AgentChat 的技能选择）+ /skills/filesystem
- ai_handler.go AgentChat：多会话协议（skillDirName、SSE sessionId、agent-DONE 哨兵）；
  会话 rename/pinned 端点
- scheduler.go：StartOKXTPSLMonitor 守护
- wails-bridge.js：+40 函数（OKX 全家/信号 v2/助手新绑定/ExportSessionMarkdown/SendDesktopNotification）；
  ChatWithAgent 对齐上游 10 参签名，SSE 分发全路径带 sessionId
- market.vue：总结中断按钮+cancelledOldPending 状态机（41ab84b）；about.vue 支付域名（84ce588）
- settings.vue：OKX 设置区块 + 代理作用范围（点补，保留 Web 版表单结构）
- AllContracts.vue：OKX 页签（import 改桥接层）

### D. 跳过
- 桌面根包（app.go sponsorMu 竞态修复、app_okx.go、app_signal_monitor.go、main.go 崩溃转储）——根包历史性编译不过，与 006 惯例一致；
  其中含共享逻辑的改动已在 internal/ 等价实现（buildSkillContext/TPSL/扫描间隔）
- isCustomBuild 门控（edf9c02）——Web 无编译渠道概念，恒 false，无操作意义
- App.vue 上游改动——仅合约菜单 custom 门控，Web 端菜单结构完全不同
- grpc v1.83.1——indirect，上游全量升级副产物

## 已知测试口径（非回归）
- TestOKXLivePositionCheck/TestOKXLiveProbe：实盘探测测试，无实盘凭证时 FAIL 为设计行为（上游同款）
- TestOKXCalcSizeByAmount SPOT 断言 "0.002" vs 实现 FormatFloat 5 位输出 "0.00200"——上游自身
  测试与实现不匹配（上游跑同样挂），保持原样不偏离

## 验证
- go build ./web ./internal ./backend ./ai-assistant-web：全绿
- go vet 同范围：全绿
- go test backend/agent、backend/agent/tools：全绿
- go test backend/data（sponsor/openai/holiday/OKXSim/Signal）：全绿
- vite build：通过

## ִ�к󲹼ǣ�2026-10-18 ����

- **�¹����޸�**��f4ca251 �ύʱ��`.gitignore` �� 111 �� `data/` ����Ĭ�̵��� backend/data/ ����
  ȫ�� 14 ������ Go �ļ���OKX ���ݲ�/�ڼ��ջ���/���ԣ������Ѹ����ļ����⡣���� git add -f ��
  52bcf7c ���ύ������ HEAD ȫ���ؽ���֤ͨ����**��ѵ���� backend/data/ �����ļ����� git add -f��
  ��Ѹù�������Ϊ '/data/'������Ŀ¼������������һ��������**
- �����ύ����f4ca251 �� 34f16eb �� 683f417 �� 12acea4���ĵ����� 52bcf7c�����ύ�޸���
- ������ߣ�Ŀ���ļ� HEAD �� origin/dev hash ��һ�� 13/13��go build��web/internal/backend/
  ai-assistant-web����go vet ͬ��Χ��agent+agent-tools+data ������ԡ�vite build ȫ�̡�
