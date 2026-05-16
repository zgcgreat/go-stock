# Design: Web 端响应式菜单布局

## Overview

本设计采用响应式布局，根据屏幕宽度自动切换菜单样式：
- **桌面端（>= 768px）**：左侧可折叠侧边栏
- **移动端（< 768px）**：底部 Tab 栏 + 更多菜单抽屉

## Architecture

### 组件结构

```
App.vue
├── WebLayout.vue (新增 - 仅 Web 端使用)
│   ├── DesktopMenu.vue (新增 - 桌面端侧边栏)
│   ├── MobileTabBar.vue (新增 - 移动端底部 Tab)
│   └── MobileDrawerMenu.vue (新增 - 移动端抽屉菜单)
└── 原有内容
```

### 响应式检测

使用 Vue 响应式 API + CSS 媒体查询：

```javascript
// 使用 window.matchMedia 检测屏幕宽度
const isMobile = ref(false)
const checkMobile = () => {
  isMobile.value = window.innerWidth < 768
}
```

## Component Design

### 1. WebLayout.vue

主布局组件，根据 `isMobile` 切换布局：

```vue
<template>
  <div class="web-layout">
    <!-- 桌面端：左侧菜单 + 右侧内容 -->
    <template v-if="!isMobile">
      <DesktopMenu :collapsed="menuCollapsed" />
      <main class="desktop-content">
        <slot />
      </main>
    </template>
    
    <!-- 移动端：内容 + 底部 Tab -->
    <template v-else>
      <main class="mobile-content">
        <slot />
      </main>
      <MobileTabBar @open-drawer="showDrawer = true" />
      <MobileDrawerMenu v-model:show="showDrawer" />
    </template>
  </div>
</template>
```

### 2. DesktopMenu.vue

桌面端侧边栏，使用 Naive UI 的 `n-layout-sider`：

- 固定在左侧
- 支持折叠/展开（图标模式）
- 显示完整菜单结构
- 宽度：展开 200px，折叠 64px

### 3. MobileTabBar.vue

移动端底部 Tab 栏，显示 4-5 个主要入口：

| Tab | 图标 | 路由 |
|-----|------|------|
| 自选 | StarOutline | /stock |
| 行情 | NewspaperOutline | /market |
| 研究 | FlaskOutline | /research |
| 设置 | SettingsOutline | /settings |
| 更多 | MenuOutline | 打开抽屉 |

### 4. MobileDrawerMenu.vue

移动端抽屉菜单，从右侧滑出：

- 显示所有菜单项（包括子菜单）
- 点击后自动关闭
- 支持分组显示

## Implementation Strategy

### 方案选择：最小改动原则

为了"尽量不改动原始项目代码"，采用以下策略：

1. **不修改 App.vue 的菜单逻辑**
2. **创建 Web 端专用的布局组件**
3. **通过条件渲染切换布局**

### 具体实现

在 App.vue 中检测 Web 模式，使用不同的布局：

```vue
<template>
  <!-- Web 模式：使用新布局 -->
  <WebLayout v-if="isWebMode">
    <!-- 原有内容区域 -->
  </WebLayout>
  
  <!-- 桌面端模式：保持原有布局 -->
  <template v-else>
    <!-- 原有代码 -->
  </template>
</template>
```

## Menu Data Structure

复用现有的 `menuOptions` 数据，但需要适配：

### 桌面端侧边栏

直接使用 `menuOptions`，`mode="vertical"`

### 移动端 Tab 栏

提取主要入口：

```javascript
const mainTabs = [
  { key: 'stock', icon: StarOutline, label: '自选' },
  { key: 'market', icon: NewspaperOutline, label: '行情' },
  { key: 'research', icon: FlaskOutline, label: '研究' },
  { key: 'settings', icon: SettingsOutline, label: '设置' },
]
```

### 移动端抽屉

使用完整 `menuOptions`，扁平化显示

## CSS Design

```css
/* 桌面端布局 */
.web-layout.desktop {
  display: flex;
  flex-direction: row;
}

.desktop-content {
  flex: 1;
  margin-left: 200px; /* 侧边栏宽度 */
  transition: margin-left 0.3s;
}

.desktop-content.collapsed {
  margin-left: 64px;
}

/* 移动端布局 */
.web-layout.mobile {
  display: flex;
  flex-direction: column;
}

.mobile-content {
  flex: 1;
  padding-bottom: 56px; /* Tab 栏高度 */
}
```

## Interaction Design

### 桌面端

- 点击菜单项：切换路由
- 点击折叠按钮：折叠/展开侧边栏
- 鼠标悬停：显示子菜单

### 移动端

- 点击 Tab：切换路由
- 点击"更多"：打开抽屉菜单
- 抽屉内点击：切换路由并关闭抽屉
- 点击遮罩：关闭抽屉

## Accessibility

- 键盘导航支持
- ARIA 标签
- 焦点管理

## Performance Considerations

- 使用 CSS transition 而非 JS 动画
- 抽屉菜单懒加载
- 响应式检测使用防抖
