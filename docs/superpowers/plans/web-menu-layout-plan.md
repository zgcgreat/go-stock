# Implementation Plan: Web 端响应式菜单布局

## Overview

基于 OpenSpec 设计文档，实现 Web 端响应式菜单布局。采用渐进式开发策略，确保每一步都可验证。

## Prerequisites

- [x] OpenSpec 文档已完成
- [x] 项目结构已了解
- [x] Naive UI 组件库已集成

## Implementation Steps

### Step 1: 创建响应式检测 Hook

**文件**: `src/composables/useResponsive.js`

```javascript
import { ref, onMounted, onBeforeUnmount } from 'vue'

const MOBILE_BREAKPOINT = 768

export function useResponsive() {
  const isMobile = ref(false)
  
  const checkMobile = () => {
    isMobile.value = window.innerWidth < MOBILE_BREAKPOINT
  }
  
  onMounted(() => {
    checkMobile()
    window.addEventListener('resize', checkMobile)
  })
  
  onBeforeUnmount(() => {
    window.removeEventListener('resize', checkMobile)
  })
  
  return { isMobile }
}
```

**验证**: 控制台输出 `isMobile` 值，确认响应式检测正常

### Step 2: 创建 WebLayout 主布局组件

**文件**: `src/components/layout/WebLayout.vue`

- 整合桌面端和移动端布局
- 使用 `useResponsive` Hook
- 提供默认插槽

**验证**: 组件渲染正常，无控制台错误

### Step 3: 创建 DesktopMenu 桌面端侧边栏

**文件**: `src/components/layout/DesktopMenu.vue`

- 使用 `n-layout-sider` 组件
- 复用 `menuOptions` 数据
- 实现折叠功能

**验证**: 侧边栏显示正常，菜单项可点击

### Step 4: 创建 MobileTabBar 移动端 Tab 栏

**文件**: `src/components/layout/MobileTabBar.vue`

- 固定底部显示
- 4 个主要入口 + 更多按钮
- 路由切换功能

**验证**: Tab 栏显示正常，点击可切换路由

### Step 5: 创建 MobileDrawerMenu 移动端抽屉

**文件**: `src/components/layout/MobileDrawerMenu.vue`

- 使用 `n-drawer` 组件
- 显示完整菜单
- 点击后关闭

**验证**: 抽屉可打开关闭，菜单项可点击

### Step 6: 集成到 App.vue

**修改**: `src/App.vue`

- 添加 Web 模式判断
- Web 模式使用 WebLayout
- 桌面端模式保持原有代码

**验证**: 
- Web 端显示新布局
- 桌面端保持原有布局

### Step 7: 样式优化

- 添加过渡动画
- 深色主题适配
- 边界情况处理

**验证**: 视觉效果符合设计

### Step 8: 最终验证

- [ ] 前端构建通过
- [ ] Go 后端构建通过
- [ ] 桌面端功能正常
- [ ] 移动端功能正常
- [ ] 响应式切换正常

## Risk Mitigation

| 风险 | 缓解措施 |
|------|----------|
| 影响桌面端功能 | 使用条件渲染，完全隔离 |
| 样式冲突 | 使用 scoped CSS |
| 性能问题 | 使用 CSS transition，避免 JS 动画 |

## Rollback Plan

如果出现问题，可以：
1. 移除 App.vue 中的 WebLayout 条件渲染
2. 删除 `src/components/layout/` 目录
3. 恢复原有布局

## Success Metrics

- 移动端菜单操作流畅
- 桌面端功能不受影响
- 构建无错误
- 用户满意
