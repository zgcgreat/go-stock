<template>
  <div class="web-layout" :class="{ 'is-mobile': isMobile, 'is-desktop': !isMobile }">
    <!-- 桌面端布局：左侧菜单 + 右侧内容 -->
    <template v-if="!isMobile">
      <DesktopMenu
        :collapsed="menuCollapsed"
        :menu-options="menuOptions"
        :active-key="activeKey"
        @update:collapsed="menuCollapsed = $event"
        @select="handleSelect"
      />
      <main class="web-content desktop-content" :class="{ 'menu-collapsed': menuCollapsed }">
        <slot />
      </main>
    </template>

    <!-- 移动端布局：内容 + 底部 Tab -->
    <template v-else>
      <main class="web-content mobile-content">
        <slot />
      </main>
      <MobileTabBar
        :active-key="activeKey"
        @select="handleSelect"
        @open-drawer="showDrawer = true"
      />
      <MobileDrawerMenu
        v-model:show="showDrawer"
        :menu-options="menuOptions"
        :active-key="activeKey"
        @select="handleDrawerSelect"
      />
    </template>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useResponsive } from '../../composables/useResponsive'
import DesktopMenu from './DesktopMenu.vue'
import MobileTabBar from './MobileTabBar.vue'
import MobileDrawerMenu from './MobileDrawerMenu.vue'

const props = defineProps({
  menuOptions: {
    type: Array,
    default: () => []
  },
  activeKey: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['select'])

const router = useRouter()
const { isMobile } = useResponsive()
const menuCollapsed = ref(false)
const showDrawer = ref(false)

const handleSelect = (key) => {
  emit('select', key)
}

const handleDrawerSelect = (key) => {
  showDrawer.value = false
  emit('select', key)
}
</script>

<style scoped>
.web-layout {
  width: 100%;
  height: 100vh;
  display: flex;
  overflow: hidden;
}

.web-layout.is-desktop {
  flex-direction: row;
}

.web-layout.is-mobile {
  flex-direction: column;
}

.web-content {
  flex: 1;
  overflow: hidden;
  transition: margin-left 0.3s ease;
}

.desktop-content {
  margin-left: 200px;
}

.desktop-content.menu-collapsed {
  margin-left: 64px;
}

.mobile-content {
  margin-left: 0;
  padding-bottom: 56px;
}
</style>
