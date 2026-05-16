# Spec: Web 端响应式菜单布局

## Feature Overview

为 Web 端实现响应式菜单布局，桌面端使用左侧侧边栏，移动端使用底部 Tab 栏 + 抽屉菜单。

## Functional Requirements

### FR-1: 响应式布局检测

- FR-1.1: 系统应检测屏幕宽度
- FR-1.2: 宽度 >= 768px 时使用桌面端布局
- FR-1.3: 宽度 < 768px 时使用移动端布局
- FR-1.4: 布局切换应平滑无闪烁

### FR-2: 桌面端侧边栏

- FR-2.1: 侧边栏固定在左侧
- FR-2.2: 默认展开，宽度 200px
- FR-2.3: 支持折叠，折叠后宽度 64px（仅显示图标）
- FR-2.4: 显示完整菜单结构，包括子菜单
- FR-2.5: 点击菜单项切换路由
- FR-2.6: 当前路由对应的菜单项高亮显示

### FR-3: 移动端 Tab 栏

- FR-3.1: Tab 栏固定在底部
- FR-3.2: 显示 4 个主要入口 + 1 个"更多"按钮
- FR-3.3: Tab 栏高度 56px
- FR-3.4: 点击 Tab 切换路由
- FR-3.5: 当前路由对应的 Tab 高亮显示

### FR-4: 移动端抽屉菜单

- FR-4.1: 点击"更多"按钮打开抽屉
- FR-4.2: 抽屉从右侧滑出
- FR-4.3: 显示所有菜单项
- FR-4.4: 点击菜单项切换路由并关闭抽屉
- FR-4.5: 点击遮罩层关闭抽屉

### FR-5: Web/桌面端模式隔离

- FR-5.1: 仅 Web 端使用新布局
- FR-5.2: 桌面端（Wails）保持原有布局
- FR-5.3: 通过 `isWebMode` 变量区分

## Non-Functional Requirements

### NFR-1: 性能

- NFR-1.1: 布局切换动画不超过 300ms
- NFR-1.2: 首次渲染不增加明显延迟

### NFR-2: 兼容性

- NFR-2.1: 支持 Chrome、Firefox、Safari、Edge 最新版本
- NFR-2.2: 支持 iOS Safari、Android Chrome

### NFR-3: 可维护性

- NFR-3.1: 新组件与现有代码解耦
- NFR-3.2: 复用现有 `menuOptions` 数据

## Technical Specifications

### TS-1: 新增文件

| 文件 | 说明 |
|------|------|
| `src/components/layout/WebLayout.vue` | Web 端主布局组件 |
| `src/components/layout/DesktopMenu.vue` | 桌面端侧边栏 |
| `src/components/layout/MobileTabBar.vue` | 移动端 Tab 栏 |
| `src/components/layout/MobileDrawerMenu.vue` | 移动端抽屉菜单 |

### TS-2: 修改文件

| 文件 | 修改内容 |
|------|----------|
| `src/App.vue` | 添加 Web 端布局条件渲染 |

### TS-3: 响应式断点

```javascript
const MOBILE_BREAKPOINT = 768
```

### TS-4: CSS 变量

```css
:root {
  --sidebar-width: 200px;
  --sidebar-collapsed-width: 64px;
  --tabbar-height: 56px;
}
```

## Test Cases

### TC-1: 桌面端布局

- TC-1.1: 宽度 1024px 时显示侧边栏
- TC-1.2: 点击折叠按钮，侧边栏折叠
- TC-1.3: 点击菜单项，路由切换

### TC-2: 移动端布局

- TC-2.1: 宽度 375px 时显示 Tab 栏
- TC-2.2: 点击 Tab，路由切换
- TC-2.3: 点击"更多"，抽屉打开
- TC-2.4: 抽屉内点击菜单项，路由切换且抽屉关闭

### TC-3: 响应式切换

- TC-3.1: 从 1024px 缩小到 375px，布局切换
- TC-3.2: 从 375px 放大到 1024px，布局切换

### TC-4: 桌面端模式

- TC-4.1: Wails 模式下保持原有布局

## Acceptance Criteria

1. 所有功能需求测试通过
2. 构建无错误
3. 桌面端功能不受影响
4. 移动端操作流畅
