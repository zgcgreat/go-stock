<template>
  <RouterView v-if="isAuthPage" />

  <WebLayout
    v-else
    :active-key="activeKey"
    @update:active-key="emit('update:activeKey', $event)"
  >
    <n-spin :show="loading">
      <template #description>
        {{ loadingMsg }}
      </template>
      <n-scrollbar style="height: calc(100vh - 80px);">
        <n-skeleton v-if="loading" height="calc(100vh)" />
        <RouterView />
      </n-scrollbar>
    </n-spin>
  </WebLayout>
</template>

<script setup>
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import WebLayout from './WebLayout.vue'

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
</script>
