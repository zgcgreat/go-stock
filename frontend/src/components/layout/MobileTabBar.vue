<template>
  <div class="mobile-tab-bar">
    <div
      v-for="tab in mainTabs"
      :key="tab.key"
      class="tab-item"
      :class="{ active: activeKey === tab.key || activeKey.startsWith(tab.key + '-') || isSubActive(tab.key) }"
      @click="handleTabClick(tab)"
    >
      <n-icon :size="22">
        <component :is="tab.icon" />
      </n-icon>
      <span class="tab-label">{{ tab.label }}</span>
    </div>
    <div class="tab-item" @click="emit('open-drawer')">
      <n-icon :size="22">
        <MenuOutline />
      </n-icon>
      <span class="tab-label">更多</span>
    </div>
  </div>
</template>

<script setup>
import { h } from 'vue'
import { useRouter } from 'vue-router'
import {
  StarOutline,
  NewspaperOutline,
  FlaskOutline,
  SettingsOutline,
  MenuOutline
} from '@vicons/ionicons5'

const props = defineProps({
  activeKey: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['select', 'open-drawer'])

const router = useRouter()

const mainTabs = [
  { key: 'stock', icon: StarOutline, label: '自选', route: { name: 'stock' } },
  { key: 'market', icon: NewspaperOutline, label: '行情', route: { name: 'market' } },
  { key: 'research', icon: FlaskOutline, label: '研究', route: { name: 'research' } },
  { key: 'settings', icon: SettingsOutline, label: '设置', route: { name: 'settings' } },
]

const isSubActive = (key) => {
  // 检查是否是子菜单激活状态
  if (key === 'stock' && ['fund'].includes(props.activeKey)) return false
  if (key === 'market' && props.activeKey.startsWith('market')) return true
  if (key === 'research' && (props.activeKey.startsWith('research') || ['agent'].includes(props.activeKey))) return true
  return false
}

const handleTabClick = (tab) => {
  emit('select', tab.key)
  router.push(tab.route)
}
</script>

<style scoped>
.mobile-tab-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: 56px;
  background: var(--n-color);
  border-top: 1px solid var(--n-border-color);
  display: flex;
  justify-content: space-around;
  align-items: center;
  z-index: 100;
}

.tab-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  height: 100%;
  cursor: pointer;
  transition: all 0.2s ease;
  color: var(--n-text-color-3);
}

.tab-item.active {
  color: var(--n-primary-color);
}

.tab-item:active {
  background: var(--n-color-hover);
}

.tab-label {
  font-size: 12px;
  margin-top: 2px;
}
</style>
