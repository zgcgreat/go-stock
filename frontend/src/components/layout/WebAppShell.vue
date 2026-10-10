<template>
  <RouterView v-if="isAuthPage" />

  <WebLayout
    v-else
    :active-key="activeKey"
    @update:active-key="emit('update:activeKey', $event)"
  >
    <!-- 快讯轮播条：对齐桌面端顶部消息轮播（enableNews 开启且有数据时显示） -->
    <TelegraphMarquee />

    <n-spin :show="loading">
      <template #description>
        {{ loadingMsg }}
      </template>
      <n-scrollbar class="web-main-scrollbar" :class="{ 'with-telegraph': telegraphVisible }">
        <n-skeleton v-if="loading" height="calc(100dvh - 44px)" />
        <RouterView />
      </n-scrollbar>
    </n-spin>
  </WebLayout>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import WebLayout from './WebLayout.vue'
import TelegraphMarquee from './TelegraphMarquee.vue'
import { useTelegraph } from '../../composables/useTelegraph'

defineProps({
  activeKey: {
    type: String,
    default: 'stock'
  },
  loading: {
    type: Boolean,
    default: false
  },
  loadingMsg: {
    type: String,
    default: '加载数据中...'
  }
})

const emit = defineEmits(['update:activeKey'])
const route = useRoute()

// Web 模式下登录/注册页面不显示导航。
const isAuthPage = computed(() => route.name === 'login' || route.name === 'register')

// 快讯轮播：页面生命周期内启动 60s 轮询，卸载时停止（useTelegraph 内部幂等）
const { visible: telegraphVisible, start: startTelegraph, stop: stopTelegraph } = useTelegraph()

onMounted(startTelegraph)
onBeforeUnmount(stopTelegraph)
</script>

<style scoped>
.web-main-scrollbar {
  height: calc(100dvh - 44px);
}

/* 轮播条显示时压缩内容区高度（44px 头部 + 34px 轮播条），避免双层滚动 */
.web-main-scrollbar.with-telegraph {
  height: calc(100dvh - 78px);
}

@supports not (height: 100dvh) {
  .web-main-scrollbar {
    height: calc(100vh - 44px);
  }

  .web-main-scrollbar.with-telegraph {
    height: calc(100vh - 78px);
  }
}
</style>
