# User Profile

Use this file for durable user-facing preferences that are helpful across sessions but are not project facts.

## Entry Template

```md
### User Preference: <short-title>
- id: user-YYYY-MM-DD-<slug>
- type: user_preference
- status: active
- confidence: verified
- last_updated: YYYY-MM-DD
- source:
- owner:
- review_after:

Preference:

Why it matters:

How to apply it:
```

### User Preference: 编译产物不提交
- id: user-2026-05-16-no-dist-commit
- type: user_preference
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: 用户明确要求

Preference: 不提交 frontend/dist 编译产物到 Git

Why it matters: 减少仓库体积，避免不必要的冲突

How to apply it: 提交时只添加源代码文件，忽略 dist 目录变更

### User Preference: 尽量不改动原始项目代码
- id: user-2026-05-16-minimal-changes
- type: user_preference
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: 用户明确要求

Preference: Web 端修改尽量不影响原始桌面端代码

Why it matters: 保持桌面端功能稳定，便于后续合并上游更新

How to apply it: 优先修改 Web 端特有文件（如 wails-bridge.js），必要时再修改共享文件

## Notes

- Keep this separate from `PROJECT_CONTEXT.md` so user preferences do not pollute project facts.
- Prefer durable interaction preferences over one-off requests.
- If a preference changes, update the entry or mark it `superseded`.
