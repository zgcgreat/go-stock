<template>
  <n-layout-sider
    bordered
    collapse-mode="width"
    :collapsed-width="64"
    :width="200"
    :collapsed="collapsed"
    show-trigger
    @collapse="emit('update:collapsed', true)"
    @expand="emit('update:collapsed', false)"
    class="desktop-menu-sider"
  >
    <n-menu
      :value="activeKey"
      :options="filteredMenuOptions"
      :collapsed="collapsed"
      :collapsed-width="64"
      :collapsed-icon-size="22"
      @update:value="handleSelect"
    />
  </n-layout-sider>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  collapsed: {
    type: Boolean,
    default: false
  },
  menuOptions: {
    type: Array,
    default: () => []
  },
  activeKey: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:collapsed', 'select'])

// 过滤掉 show: false 的菜单项
const filteredMenuOptions = computed(() => {
  return props.menuOptions.filter(item => item.show !== false)
})

const handleSelect = (key) => {
  emit('select', key)
}
</script>

<style scoped>
.desktop-menu-sider {
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  z-index: 100;
  background: var(--n-color);
}

.desktop-menu-sider :deep(.n-menu) {
  font-size: 14px;
}
</style>
