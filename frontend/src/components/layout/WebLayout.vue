<template>
  <div class="web-layout">
    <!-- 顶部导航栏：只有一级Tab -->
    <header class="web-header">
      <div class="header-inner">
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

        <!-- 用户信息 + 退出登录 -->
        <div class="user-area" v-if="isWebMode">
          <n-dropdown :options="userMenuOptions" @select="handleUserMenuSelect">
            <n-button quaternary size="small" class="user-btn">
              <template #icon>
                <n-icon><PersonCircleOutline /></n-icon>
              </template>
              <span class="user-name">{{ displayName }}</span>
            </n-button>
          </n-dropdown>
        </div>
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
import { ref, computed, watch, onMounted } from 'vue'
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
  PeopleOutline,
  LogOutOutline,
  PersonCircleOutline,
} from '@vicons/ionicons5'
import { Robot } from '@vicons/fa'
import { useResponsive } from '../../composables/useResponsive'
import { useIsWebMode } from '../../composables/useResponsive'
import Auth from '../../utils/auth'
import apiService from '../../services/api.js'
import { useMessage, useDialog } from 'naive-ui'

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
const { isWebMode } = useIsWebMode()
const message = useMessage()
const dialog = useDialog()
const showMoreDrawer = ref(false)

// 退出登录
function handleLogout() {
  dialog.warning({
    title: '退出登录',
    content: '确定要退出登录吗？',
    positiveText: '确定退出',
    negativeText: '取消',
    onPositiveClick: () => {
      Auth.clearToken()
      window.dispatchEvent(new Event('user-info-updated'))
      message.success('已退出登录')
      router.push({ name: 'login' })
    }
  })
}

// 用户信息
const userInfo = computed(() => Auth.getUserInfo())
const displayName = computed(() => userInfo.value?.username || '用户')

// 响应式用户角色：登录后 setUserInfo 触发更新
const userRole = ref('')
const isAdmin = computed(() => {
  const role = userRole.value
  return role === 'admin' || role === 'super_admin'
})

// 初始化 + 监听 localStorage 变化（跨 Tab / 登录后刷新）
function refreshUserRole() {
  const userInfo = Auth.getUserInfo()
  userRole.value = userInfo?.role || ''
}

// 如果 localStorage 没有 user_info 但有 token，主动获取用户信息
async function ensureUserInfo() {
  if (Auth.getUserInfo()) {
    refreshUserRole()
    return
  }
  if (!Auth.getToken()) return
  try {
    const res = await apiService.client.get('/user/profile')
    const user = res.data || {}
    Auth.setUserInfo({
      userId: user.id,
      username: user.username,
      role: user.role,
      vipLevel: user.vipLevel,
      vipStartAt: user.vipStartAt,
      vipEndAt: user.vipEndAt,
    })
    refreshUserRole()
  } catch (e) {
    console.warn('[WebLayout] 获取用户信息失败:', e)
  }
}
ensureUserInfo()

// 监听 storage 事件（其他 Tab 登录/退出时触发）
onMounted(() => {
  window.addEventListener('storage', refreshUserRole)
  // 自定义事件：同 Tab 内 Login.vue 登录成功后触发
  window.addEventListener('user-info-updated', refreshUserRole)
})

// 主Tab定义：只负责一级页面切换，子页面内部自己管Tab
const mainTabs = [
  { key: 'stock', label: '自选', icon: StarOutline, route: { name: 'stock' } },
  { key: 'market', label: '行情', icon: NewspaperOutline, route: { name: 'market' } },
  { key: 'klineAnalysis', label: 'K线', icon: StatsChartOutline, route: { name: 'klineAnalysis' } },
  { key: 'fund', label: '基金', icon: SparklesOutline, route: { name: 'fund' } },
  { key: 'agent', label: 'AI', icon: Robot, route: { name: 'agent' } },
  { key: 'research', label: '研究', icon: FlaskOutline, route: { name: 'research' } },
  { key: 'settings', label: '设置', icon: SettingsOutline, route: { name: 'settings' } },
  { key: 'admin', label: '用户管理', icon: PeopleOutline, route: { name: 'userManagement' }, adminOnly: true },
]

// 移动端只显示4个核心Tab
const mobileVisibleCount = 4

const visibleMainTabs = computed(() => {
  const tabs = mainTabs.filter(tab => !tab.adminOnly || isAdmin.value)
  if (isMobile.value) {
    return tabs.slice(0, mobileVisibleCount)
  }
  return tabs
})

const overflowTabs = computed(() => {
  if (isMobile.value) {
    const tabs = mainTabs.filter(tab => !tab.adminOnly || isAdmin.value)
    return tabs.slice(mobileVisibleCount)
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
  if (key === 'admin' || key === 'userManagement') return 'admin'
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

// 用户下拉菜单
const userMenuOptions = computed(() => {
  const options = [
    { label: displayName.value, key: 'info', disabled: true },
    { type: 'divider', key: 'd1' },
    { label: '退出登录', key: 'logout', icon: () => h(NIcon, null, { default: () => h(LogOutOutline) }) },
  ]
  return options
})

function handleUserMenuSelect(key) {
  if (key === 'logout') {
    handleLogout()
  }
}
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

.user-area {
  flex-shrink: 0;
  margin-left: 8px;
}

.user-btn {
  display: flex;
  align-items: center;
  gap: 4px;
}

.user-name {
  font-size: 13px;
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>