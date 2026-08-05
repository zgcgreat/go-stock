# 2026-08-02 — sync upstream to web (dev-web) 接线完成

## 完成内容
- 修复 staged 移植 WIP 的全部编译错误：FollowedStock/Group/GroupStock 字段恢复、tool_group_concept_manage GroupID、app.go/app_common.go userID=0 参数、settings_api.go 误删行、data_tools_wrapper AgentMeta 移植、iwencai 网关搜索、测试文件清理
- Web 接线：每日操作计划（user_id 隔离 + 迁移）、TDX 分时/成交、AI 配置更新（UpdateAiConfigsByUserID）、概念 CRUD（新 handler 文件）
- 前端导航：router.js 加 /daily-operation-plans 与 /ai-config-manager；WebLayout.vue 加「计划」页签（CalendarOutline）
- ai-config-manager.vue import 改走 wails-bridge（web 回退）
- 验证：go build/vet 全绿；npm run build 通过；bridge node --check 通过

## 缺口确认（未移植分析）
- TEMA/planEndDate/A股指数/MCP headers/交易盈亏 均已移植，无重大剩余
- FetchAiModelInfo 仍是 stub（无后端路由），已知缺口，非核心接线范围

## 待办
- git 提交（跳过 frontend/wailsjs 自动生成文件）
