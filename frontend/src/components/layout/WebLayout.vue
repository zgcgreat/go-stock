<template>
  <div class="web-layout">
    <!-- 顶部导航栏：只有一级Tab -->
    <header class="web-header">
      <div class="header-inner">
        <!-- Logo + 市场状态 -->
        <div class="brand">
          <span class="brand-text">go-stock</span>
          <span class="brand-status" v-if="marketStatus">{{ marketStatus }}</span>
        </div>

        <!-- 主 Tab 导航 -->
        <n-tabs
          :value="activeMainTab"
          type="line"
          size="small"
          animated
          class="main-tabs"
          @update:value="handleMainTabChange"
        >
          <n-tab v-for="tab in visibleMainTabs" :key="tab.key" :name="tab.key">
            <n-icon :size="16" style="margin-right: 4px; vertical-align: middle;">
              <component :is="tab.icon" />
            </n-icon>
            <span style="vertical-align: middle;">{{ tab.label }}</span>
          </n-tab>
        </n-tabs>

        <!-- 移动端：更多菜单按钮 -->
        <n-button
          v-if="isMobile && overflowTabs.length > 0"
          quaternary
          size="small"
          @click="showMoreDrawer = true"
          class="more-btn"
        >
          <template #icon>
            <n-icon><MenuOutline /></n-icon>
          </template>
        </n-button>
      </div>
    </header>

    <!-- 内容区域：各页面自带内部Tab切换 -->
    <main class="web-content">
      <slot />
    </main>

    <!-- 移动端更多菜单 Drawer -->
    <n-drawer v-model:show="showMoreDrawer" :width="260" placement="right" v-if="isMobile">
      <n-drawer-content title="更多功能" closable>
        <n-menu
          :value="activeKey"
          :options="overflowMenuOptions"
          @update:value="handleOverflowSelect"
        />
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { h } from 'vue'
import { NIcon } from 'naive-ui'
import {
  StarOutline,
  NewspaperOutline,
  StatsChartOutline,
  SparklesOutline,
  FlaskOutline,
  SettingsOutline,
  MenuOutline,
} from '@vicons/ionicons5'
import { Robot } from '@vicons/fa'
import { useResponsive } from '../../composables/useResponsive'

const props = defineProps({
  menuOptions: {
    type: Array,
    default: () => []
  },
  activeKey: {
    type: String,
    default: ''
  },
  marketStatus: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['select', 'update:activeKey'])

const router = useRouter()
const { isMobile } = useResponsive()
const showMoreDrawer = ref(false)

// 主Tab定义：只负责一级页面切换，子页面内部自己管Tab
const mainTabs = [
  { key: 'stock', label: '自选', icon: StarOutline, route: { name: 'stock' } },
  { key: 'market', label: '行情', icon: NewspaperOutline, route: { name: 'market' } },
  { key: 'klineAnalysis', label: 'K线', icon: StatsChartOutline, route: { name: 'klineAnalysis' } },
  { key: 'fund', label: '基金', icon: SparklesOutline, route: { name: 'fund' } },
  { key: 'agent', label: 'AI', icon: Robot, route: { name: 'agent' } },
  { key: 'research', label: '研究', icon: FlaskOutline, route: { name: 'research' } },
  { key: 'settings', label: '设置', icon: SettingsOutline, route: { name: 'settings' } },
]

// 移动端只显示4个核心Tab
const mobileVisibleCount = 4

const visibleMainTabs = computed(() => {
  if (isMobile.value) {
    return mainTabs.slice(0, mobileVisibleCount)
  }
  return mainTabs
})

const overflowTabs = computed(() => {
  if (isMobile.value) {
    return mainTabs.slice(mobileVisibleCount)
  }
  return []
})

// overflow 菜单选项（给移动端 Drawer 用）
const overflowMenuOptions = computed(() => {
  return overflowTabs.value.map(tab => ({
    label: tab.label,
    key: tab.key,
    icon: () => h(NIcon, null, { default: () => h(tab.icon) }),
  }))
})

// 确定当前激活的主Tab
const activeMainTab = computed(() => {
  const key = props.activeKey
  if (mainTabs.find(t => t.key === key)) return key
  if (!isNaN(key) && key !== '') return 'stock'
  if (key.startsWith('market')) return 'market'
  if (key.startsWith('research') || ['uplimitLadder', 'promptPlaza', 'promptQa',
      'stockChanges', 'mcpServers', 'skills'].includes(key)) return 'research'
  if (['fundFollow', 'fundRanking'].includes(key)) return 'fund'
  return 'stock'
})

// 主Tab切换
const handleMainTabChange = (key) => {
  const tab = mainTabs.find(t => t.key === key)
  if (tab) {
    router.push(tab.route)
    emit('update:activeKey', key)
    emit('select', key)
  }
}

// 移动端 overflow 菜单选中
const handleOverflowSelect = (key) => {
  showMoreDrawer.value = false
  const tab = mainTabs.find(t => t.key === key)
  if (tab) {
    router.push(tab.route)
    emit('update:activeKey', key)
    emit('select', key)
  }
}

watch(isMobile, () => {
  showMoreDrawer.value = false
})
</script>

<style scoped>
.web-layout {
  width: 100%;
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.web-header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: var(--n-color, #fff);
  border-bottom: 1px solid var(--n-border-color, #e0e0e6);
  flex-shrink: 0;
}

.header-inner {
  display: flex;
  align-items: center;
  padding: 0 12px;
  height: 44px;
  gap: 8px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  margin-right: 4px;
}

.brand-text {
  font-size: 15px;
  font-weight: 600;
  color: var(--n-text-color);
  letter-spacing: -0.3px;
}

.brand-status {
  font-size: 11px;
  color: var(--n-text-color-3);
  white-space: nowrap;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--n-color-hover, #f5f5f5);
}

.main-tabs {
  flex: 1;
  min-width: 0;
}

.main-tabs :deep(.n-tabs-nav) {
  justify-content: flex-start;
}

.main-tabs :deep(.n-tab) {
  padding: 6px 10px;
  font-size: 14px;
}

.more-btn {
  flex-shrink: 0;
}

.web-content {
  flex: 1;
  overflow: auto;
  min-height: 0;
}
</style>