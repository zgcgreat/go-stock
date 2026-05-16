# Known Failures

Use this file for repeated failure modes, environment pitfalls, process traps, and recurring misjudgments.

## Entry Template

```md
### Failure Pattern: <short-title>
- id: failure-YYYY-MM-DD-<slug>
- type: failure_pattern
- status: active
- confidence: verified
- last_updated: YYYY-MM-DD
- source:
- owner:
- review_after:

Trigger:

Symptom:

Likely cause:

How to detect:

Mitigation:
```

### Failure Pattern: TradingView API 被墙
- id: failure-2026-05-16-tradingview-blocked
- type: failure_pattern
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: 外媒新闻功能
- owner: developer
- review_after: 2026-06-16

Trigger: 访问 TradingView 新闻 API

Symptom: 外媒新闻无法加载，请求超时

Likely cause: TradingView 在中国被屏蔽

How to detect: 检查 API 请求返回错误或超时

Mitigation: 配置 HTTP 代理（127.0.0.1:10808）

### Failure Pattern: VPS 内存不足构建失败
- id: failure-2026-05-16-vps-oom
- type: failure_pattern
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: Docker 构建
- owner: developer
- review_after: 2026-06-16

Trigger: 在 1C2G VPS 上执行 docker build

Symptom: 构建过程中 OOM killed

Likely cause: 前端构建（npm run build）内存占用过高

How to detect: 检查 dmesg 或 docker logs

Mitigation: 本地构建镜像推送，或增加 swap

## Notes

- Prefer repeated or high-impact failures over one-off mistakes.
- If a failure is fully obsolete, mark it as `superseded` and explain why.
- Link to verification evidence when possible.
- Durable entries should always include `id`, `status`, `confidence`, `source`, `last_updated`, and `review_after`.
- Use `review_after` to force periodic re-checks of environment-sensitive failure patterns.