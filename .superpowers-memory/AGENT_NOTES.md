# Agent Notes

Use this file for durable agent-side reminders about execution quality, repo-specific handling, or recurring operational pitfalls.

## Entry Template

```md
### Agent Note: <short-title>
- id: agent-YYYY-MM-DD-<slug>
- type: agent_note
- status: active
- confidence: verified
- last_updated: YYYY-MM-DD
- source:
- owner:
- review_after:

Reminder:

Why it matters:

How to apply it:
```

### Agent Note: Web 端适配优先
- id: agent-2026-05-16-web-first
- type: agent_note
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: 用户需求
- owner: agent
- review_after: 2026-06-16

Reminder: 当前主要工作是 Web 端功能完善，需要确保不破坏桌面端功能

Why it matters: 项目同时支持桌面端和 Web 端，需要保持兼容

How to apply it: 修改共享代码时，检查两种模式下的行为

### Agent Note: 用户数据隔离检查
- id: agent-2026-05-16-user-check
- type: agent_note
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: 架构设计
- owner: agent
- review_after: 2026-06-16

Reminder: 所有涉及用户数据的 API 都需要从 context 获取 userID

Why it matters: Web 端是多用户系统，数据隔离是安全基础

How to apply it: 检查 handler 是否使用 `middleware.GetUserIDFromContext(c)`

## Notes

- Keep this separate from project facts and user preferences.
- Use this file for repeatable execution reminders, not transient scratch notes.
- If a note is no longer relevant, mark it `superseded` instead of silently removing the history.