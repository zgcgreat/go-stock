<template>
  <div style="padding: 16px">
    <n-card title="游资席位名录维护" size="small" style="max-width: 720px">
      <n-space vertical size="large">
        <n-text depth="3" style="font-size: 12px; display: block">
          维护游资花名与营业部席位映射（支持远程名录 JSON 同步）。数据供龙虎榜页按游资/机构聚合展示使用；
          修改即时生效并持久化到 data/hot_money_seats.json。
        </n-text>
        <n-space>
          <n-button type="primary" @click="openManager">打开名录维护面板</n-button>
        </n-space>
      </n-space>
    </n-card>
    <HotMoneySeatsManager ref="managerRef" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import HotMoneySeatsManager from './HotMoneySeatsManager.vue'

// [Web 移植] 上游该组件为抽屉形态（n-drawer，内部 show ref + defineExpose），
// 桌面端由设置页触发打开；Web 端无宿主页面，故以此薄包装页承载：
// 进入页面自动展开抽屉，用户关闭后可点击按钮重新打开。
const managerRef = ref()

function openManager() {
  if (managerRef.value) managerRef.value.show = true
}

onMounted(() => {
  openManager()
})
</script>
