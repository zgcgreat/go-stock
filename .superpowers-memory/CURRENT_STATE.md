# Current State

Use this file for the latest working context that the next session should read first.

## Active Focus

- **Current task**: 上游 `origin/dev` 新功能已同步到 web 端（dev-web），核心功能接线全部完成，待提交
- **Why it matters now**: 完成「完整接线核心功能」选项（每日操作计划/TDX/AI配置/概念CRUD 路由+handler+bridge+导航）

## Latest Decisions

- **Decision**: Web 接线范围按用户选定的「完整接线核心功能」执行；飞书机器人明确不接（需长驻进程）
- **Decision**: 数据层用 variadic `userID ...uint` + `firstUserID()` 兼容桌面/agent/web 调用方；handler 用 `middleware.GetUserIDFromContext`
- **Decision**: `GetStockRealTimePrice` 的 web 回退改为 `/stocks/price-info` 归一化形状（DailyOperationPlan/TradingRecord 依赖）
- **Decision**: ai-config-manager.vue 的 import 已从 wailsjs 直连改为走 wails-bridge（web 回退可用）
- **Decision**: `FetchAiModelInfo` 已补后端路由 `/ai/model-info`（逻辑对齐桌面端，含内置模型 token 对照表）

## Open Questions

- **Question**: 无阻塞问题
- **Blocker or follow-up**: 待提交

## Next Recommended Step

- **Next step**: git 提交本轮改动
- **Suggested owner**: 用户确认后提交
