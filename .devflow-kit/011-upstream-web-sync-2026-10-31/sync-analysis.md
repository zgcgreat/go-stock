# 011 上游同步（796e70d..d18a0eb，2026-10-31）

上游区间：`796e70d..origin/dev(d18a0eb)`，共 1 个提交。
执行前基线：dev-web == 010 同步完成态（5ac36f5）。

## 上游提交

| 提交 | 内容 | 规模 |
|------|------|------|
| d18a0eb | feat(okx): 信号自动交易新增 Mass Index 反转闸门，K线主图反转窗口阈值分位自适应 | 9 文件 +820/-50 |

## 处理分层

### A. 直接同步（零偏移/新文件，6 文件 hash 全一致）
- backend/data/okx_mass_gate.go（新 +277：1H MI(9,9,25) 反转窗口闸门，进入线=该合约 MI 历史 90% 分位钳制 24~31、离开线低 0.5、样本不足回退 27/26.5，只拦开仓不拦平仓）
- backend/data/okx_mass_gate_test.go（新 +135，3 用例全过）
- backend/data/okx_signal_trade.go（+20 闸门接线）
- frontend/src/components/kline/massZonePrimitive.js（新 +279：主图琥珀色竖带 primitive）
- frontend/src/components/kline/indicators/tips.ts（+2 自适应阈值说明）
- frontend/wailsjs/go/models.ts（+2 Settings 字段）

### B. 手工合并
1. **backend/data/settings_api.go**：Settings +OKXSignalTradeMassGate(bool，Regime 字段后)；
   UpdateConfig + UpdateConfigByUserID 各 +1 键。
2. **frontend/src/components/settings.vue**：MassGate 开关 UI（顺大势门控表单项后）+ 4 处 JS 映射。
3. **frontend/src/components/StockLightweightKlineChart.vue**（+142/-50 上游，本批仅合并 script 功能部分）：
   - import massZonePrimitive + 声明（沿本地 VPVR 同构 primitive 模式：ensure/sync/detach）
   - getMassZoneData（版本缓存，本地 extractOHLCV 返回形状兼容 {times,highs,lows} 解构）
   - syncMassZonePrimitive() 挂进 syncIndicators（syncVolumeProfilePrimitive() 后）
   - 两处固定阈值 27/26.5 → massAdaptiveThresholds 自适应（统计行内判断 + evaluateIndicatorSignals）
   - disposeChart 卸载
   - **跳过：上游指标侧边栏按钮重排**（🔄反转分组重排/按钮移位，纯装饰性；本地保持现有布局，
     侧边栏完整重组留待在案缺口「overlay 体系接线」专项处理）

### C. Web 接线
无——本批无新端点/事件/bridge 函数（闸门在后端信号交易链路内生效，竖带为纯前端绘制）。

### D. 跳过
无桌面根包/生成物改动（上游本批未触及）。

## 验证基线（全部通过）
- A 类 hash 核对：6/6 与 origin/dev 一致
- go build ./backend/... ./web/：绿；go vet 同范围：绿
- go test -run 'TestMass|TestOkxMass'：TestOkxMassIndexConstantRange / TestOkxMassAdaptiveThresholds / TestOkxMassTailState 全 PASS
- vite build：通过（4m36s，tmp-build-011 验证后清理）

## 同步后状态
- 同步前沿：`d18a0eb`
- 提交链：B1（后端+零偏移 6 文件+settings_api）→ B2（settings.vue+图表 Mass 接线）→ 本记录
