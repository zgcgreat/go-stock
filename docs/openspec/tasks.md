# Tasks: Web 端响应式菜单布局

## Phase 1: 基础设施

### Task 1.1: 创建布局组件目录结构
- 创建 `src/components/layout/` 目录
- 创建 `WebLayout.vue` 空组件

### Task 1.2: 实现响应式检测 Hook
- 创建 `src/composables/useResponsive.js`
- 实现 `isMobile` 响应式变量
- 实现屏幕宽度监听

## Phase 2: 桌面端侧边栏

### Task 2.1: 创建 DesktopMenu.vue
- 使用 `n-layout-sider` 组件
- 复用 `menuOptions` 数据
- 实现垂直菜单模式

### Task 2.2: 实现折叠功能
- 添加折叠按钮
- 实现折叠状态管理
- 添加折叠动画

### Task 2.3: 样式调整
- 设置侧边栏宽度
- 设置折叠后宽度
- 内容区域 margin 调整

## Phase 3: 移动端 Tab 栏

### Task 3.1: 创建 MobileTabBar.vue
- 使用 `n-tab-bar` 或自定义组件
- 显示 4 个主要入口 + 更多按钮
- 实现路由切换

### Task 3.2: Tab 栏样式
- 固定在底部
- 设置高度 56px
- 图标和文字样式

## Phase 4: 移动端抽屉菜单

### Task 4.1: 创建 MobileDrawerMenu.vue
- 使用 `n-drawer` 组件
- 从右侧滑出
- 显示完整菜单

### Task 4.2: 菜单项渲染
- 扁平化菜单结构
- 实现点击跳转
- 点击后关闭抽屉

## Phase 5: 集成

### Task 5.1: 创建 WebLayout.vue
- 整合 DesktopMenu、MobileTabBar、MobileDrawerMenu
- 根据响应式状态切换布局
- 插槽传递内容

### Task 5.2: 修改 App.vue
- 添加 `isWebMode` 条件判断
- Web 模式使用 WebLayout
- 桌面端模式保持原有代码

### Task 5.3: 样式优化
- 添加过渡动画
- 处理边界情况
- 适配深色主题

## Phase 6: 测试与验证

### Task 6.1: 功能测试
- 桌面端菜单功能
- 移动端 Tab 功能
- 抽屉菜单功能
- 响应式切换

### Task 6.2: 构建验证
- 前端构建通过
- Go 后端构建通过
- 运行时无错误

## Task Dependencies

```
1.1 -> 1.2 -> 5.1
2.1 -> 2.2 -> 2.3 -> 5.1
3.1 -> 3.2 -> 5.1
4.1 -> 4.2 -> 5.1
5.1 -> 5.2 -> 5.3 -> 6.1 -> 6.2
```

## Estimated Effort

| Phase | 任务数 | 预估时间 |
|-------|--------|----------|
| Phase 1 | 2 | 30min |
| Phase 2 | 3 | 1h |
| Phase 3 | 2 | 45min |
| Phase 4 | 2 | 45min |
| Phase 5 | 3 | 1h |
| Phase 6 | 2 | 30min |
| **Total** | **14** | **4.5h** |
