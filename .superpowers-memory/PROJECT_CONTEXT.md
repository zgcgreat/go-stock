# Project Context

Use this file for stable project knowledge that should survive across sessions.

## Project Summary

- **What this project does**: go-stock 是一个基于大语言模型的 AI 赋能股票分析工具，支持 A股、港股、美股的实时行情监控、AI 分析、技术指标分析等功能
- **Who uses it**: 个人投资者、股票分析爱好者、量化研究初学者
- **Main goals**: 提供便捷的股票数据查看和 AI 辅助分析功能，降低投资分析门槛

## Architecture Notes

- **Core modules**:
  - `frontend/` - Vue 3 + Naive UI 前端界面
  - `internal/` - Go 后端服务（Gin 框架）
  - `backend/` - 数据层和业务逻辑
  - `wailsjs/` - Wails 桌面端桥接
- **Important boundaries**:
  - 桌面端使用 Wails 框架，Web 端通过 `wails-bridge.js` 适配
  - 用户数据隔离通过 `user_id` 字段实现
  - SQLite 数据库存储
- **Key data flows**:
  - 前端 → wails-bridge.js → 后端 API / Wails Go 函数
  - 外部数据源：Tushare、东方财富、新浪财经等

## Working Agreements

- **Coding conventions**:
  - Go 后端使用 Gin 框架，遵循 RESTful API 设计
  - 前端使用 Vue 3 Composition API + Naive UI 组件库
  - 数据库操作使用 GORM
- **Testing expectations**: 暂无自动化测试，手动测试为主
- **Deployment notes**:
  - 桌面端：Wails 编译为可执行文件
  - Web 端：Docker 容器部署，支持低内存 VPS

## Known Constraints

- **Technical constraints**:
  - Windows 10+ 为主要开发环境
  - 部分外部 API（如 TradingView）在中国需要代理访问
  - Web 端不支持桌面端特有的窗口操作功能
- **Product constraints**:
  - AI 分析结果仅供学习研究，不构成投资建议
  - 港股数据有延迟
- **Operational constraints**:
  - 单用户模式（桌面端）vs 多用户模式（Web 端）

## Durable Facts

### Fact: 双模式架构
- id: context-2026-05-16-dual-mode
- type: durable_fact
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: code analysis

Statement: 项目支持桌面端（Wails）和 Web 端两种运行模式，通过 `wails-bridge.js` 统一接口调用。

Why it matters: 所有前端功能需要同时考虑两种模式的兼容性。

### Fact: 用户数据隔离
- id: context-2026-05-16-user-isolation
- type: durable_fact
- status: active
- confidence: verified
- last_updated: 2026-05-16
- source: database schema

Statement: Web 端支持多用户，所有用户相关数据表都有 `user_id` 字段进行隔离。

Why it matters: API 设计和数据处理需要始终考虑用户上下文。
