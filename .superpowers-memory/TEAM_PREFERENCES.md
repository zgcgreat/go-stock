# Team Preferences

Use this file for durable collaboration preferences, communication boundaries, and team-specific working agreements.

## Entry Template

```md
### Preference: <short-title>
- id: preference-YYYY-MM-DD-<slug>
- type: team_preference
- status: active
- confidence: verified
- last_updated: YYYY-MM-DD
- source:
- owner:
- review_after:

Preference:

Why it exists:

How to apply it:
```

### Preference: 中文交流
- id: preference-2026-05-16-chinese
- type: team_preference
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: 项目背景
- owner: developer
- review_after: 2026-06-16

Preference: 代码注释、提交信息、文档使用中文

Why it exists: 项目面向中文用户，团队使用中文交流

How to apply it: 所有面向用户的文本使用中文，代码注释使用中文

### Preference: 渐进式开发
- id: preference-2026-05-16-incremental
- type: team_preference
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: 开发风格
- owner: developer
- review_after: 2026-06-16

Preference: 小步提交，频繁验证

Why it exists: 便于问题定位和回滚

How to apply it: 每个功能点单独提交，提交前验证构建通过

## Notes

- Keep this file for durable preferences, not session-specific requests.
- If a preference changes, update the existing entry or mark it `superseded`.
- Durable entries should always include `id`, `status`, `confidence`, `source`, `last_updated`, and `review_after`.