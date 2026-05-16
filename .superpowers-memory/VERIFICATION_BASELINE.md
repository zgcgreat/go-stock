# Verification Baseline

Use this file for the verification commands and evidence standards that the team considers trustworthy for this project.

## Entry Template

```md
### Verification Rule: <short-title>
- id: verification-YYYY-MM-DD-<slug>
- type: verification_rule
- status: active
- confidence: verified
- last_updated: YYYY-MM-DD
- source:
- owner:
- review_after:

Command or method:

What it validates:

What it does not validate:

Evidence expected:
```

### Verification Rule: Go 后端编译
- id: verification-2026-05-16-go-build
- type: verification_rule
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: 开发流程
- owner: developer
- review_after: 2026-06-16

Command or method: `go build -o /dev/null ./web/...`

What it validates: Go 后端代码语法正确，无编译错误

What it does not validate: 运行时行为、API 功能正确性

Evidence expected: 命令无错误输出

### Verification Rule: 前端构建
- id: verification-2026-05-16-frontend-build
- type: verification_rule
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: 开发流程
- owner: developer
- review_after: 2026-06-16

Command or method: `cd frontend && npm run build`

What it validates: Vue 前端代码可正常构建

What it does not validate: UI 渲染正确性、功能完整性

Evidence expected: 构建成功，生成 dist 目录

### Verification Rule: Web 服务健康检查
- id: verification-2026-05-16-health-check
- type: verification_rule
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: API 设计
- owner: developer
- review_after: 2026-06-16

Command or method: `curl http://localhost:8080/health`

What it validates: Web 服务正常运行

What it does not validate: 各功能模块正确性

Evidence expected: 返回 `{"status": "ok", "time": ...}`

## Notes

- Prefer commands that are reproducible and already used successfully by the team.
- Record known blind spots so future sessions do not overclaim confidence.
- Durable entries should always include `id`, `status`, `confidence`, `source`, `last_updated`, and `review_after`.
- If the rule is `verified`, the `source` should point to a successful command, log, test, or documented evidence.