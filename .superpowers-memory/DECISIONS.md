# Decisions

Use this file for important project or workflow decisions that should remain visible across sessions.

## Entry Template

```md
### Decision: <short-title>
- id: decision-YYYY-MM-DD-<slug>
- type: decision
- status: active
- confidence: verified
- last_updated: YYYY-MM-DD
- source:
- owner:
- review_after:

Reason:

Alternatives considered:

Impact:
```

### Decision: Web API 用户隔离
- id: decision-2026-05-16-user-isolation
- type: architecture_decision
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: Web 端开发需求
- owner: developer
- review_after: 2026-06-16

Reason: Web 端是多用户系统，需要确保数据隔离，所有用户相关 API 都需要通过 middleware 获取用户 ID 并过滤数据

Alternatives considered: 无用户隔离（单用户模式）

Impact: 所有涉及用户数据的 handler 都需要从 context 获取 userID

### Decision: wails-bridge.js 作为适配层
- id: decision-2026-05-16-wails-bridge
- type: architecture_decision
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: 双模式适配需求
- owner: developer
- review_after: 2026-06-16

Reason: wails-bridge.js 作为桌面端和 Web 端的统一接口层，检测运行模式并调用对应实现，保持前端代码统一

Alternatives considered: 分别开发两套前端代码

Impact: 所有跨平台功能调用都需要通过 wails-bridge.js

### Decision: Docker 部署方案
- id: decision-2026-05-16-docker-deploy
- type: process_decision
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: VPS 部署需求
- owner: developer
- review_after: 2026-06-16

Reason: VPS 内存不足（1C2G），采用本地构建镜像推送方案

Alternatives considered: VPS 直接构建、增加 swap

Impact: 部署流程需要本地构建 + 推送镜像 + VPS 拉取运行

## Notes

- Put only decisions that still matter to future sessions.
- Move outdated decisions to `status: superseded` instead of deleting history blindly.
- Reference code, docs, tests, or session notes when possible.
- Durable entries should always include `id`, `status`, `confidence`, `source`, `last_updated`, and `review_after`.
- Do not mark an entry `confidence: verified` unless `source` points to real evidence.