<template>
  <!-- 快讯轮播条：对齐桌面端 App.vue 顶部的 n-marquee（telegraph 事件 → n-tag warning 滚动）。
           数据由 useTelegraph 单例提供（Web 模式轮询 /telegraph 接口）。 -->
  <div v-if="visible" class="telegraph-bar">
    <n-marquee :speed="100" class="telegraph-marquee">
      <n-tag type="warning" v-for="(item, idx) in telegraph" :key="idx" style="margin-right: 10px">
        {{ item }}
      </n-tag>
    </n-marquee>
  </div>
</template>

<script setup>
import { useTelegraph } from '../../composables/useTelegraph'

const { telegraph, visible } = useTelegraph()
</script>

<style scoped>
/* 固定高度行：供 WebAppShell 计算内容区滚动条高度（44px 头部 + 34px 轮播条） */
.telegraph-bar {
  height: 34px;
  display: flex;
  align-items: center;
  overflow: hidden;
  flex-shrink: 0;
  border-bottom: 1px solid var(--n-border-color, #efeff5);
  background: var(--n-color, #fff);
}

.telegraph-marquee {
  width: 100%;
}
</style>
