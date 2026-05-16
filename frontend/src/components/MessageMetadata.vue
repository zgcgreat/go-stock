<template>
  <div class="message-metadata" v-if="hasMetadata">
    <span v-if="modelName" class="metadata-item model-name" :title="'模型: ' + modelName">
      <NIcon :component="CubeOutline" size="12" />
      <span class="metadata-value">{{ truncateModelName(modelName) }}</span>
    </span>
    <span v-if="totalTokens > 0" class="metadata-item tokens" :title="'Token: ' + formatNumber(totalTokens)">
      <NIcon :component="BarChartOutline" size="12" />
      <span class="metadata-value">{{ formatTokens(totalTokens) }}</span>
    </span>
    <span v-if="durationMs > 0" class="metadata-item duration" :title="'耗时: ' + formatDuration(durationMs)">
      <NIcon :component="TimeOutline" size="12" />
      <span class="metadata-value">{{ formatDuration(durationMs) }}</span>
    </span>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { NIcon } from 'naive-ui'
import { CubeOutline, BarChartOutline, TimeOutline } from '@vicons/ionicons5'

const props = defineProps({
  modelName: {
    type: String,
    default: ''
  },
  promptTokens: {
    type: Number,
    default: 0
  },
  completionTokens: {
    type: Number,
    default: 0
  },
  totalTokens: {
    type: Number,
    default: 0
  },
  durationMs: {
    type: Number,
    default: 0
  }
})

const hasMetadata = computed(() => {
  return props.modelName || props.totalTokens > 0 || props.durationMs > 0
})

function truncateModelName(name) {
  if (!name) return ''
  // 截断过长的模型名称
  if (name.length > 20) {
    return name.substring(0, 18) + '...'
  }
  return name
}

function formatTokens(tokens) {
  if (tokens >= 1000) {
    return (tokens / 1000).toFixed(1) + 'k'
  }
  return tokens.toString()
}

function formatNumber(num) {
  return num.toLocaleString()
}

function formatDuration(ms) {
  if (ms < 1000) {
    return ms + 'ms'
  }
  return (ms / 1000).toFixed(1) + 's'
}
</script>

<style scoped>
.message-metadata {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  color: var(--n-text-color-3);
  flex-wrap: wrap;
}

.metadata-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 6px;
  background: var(--n-color-hover);
  border-radius: 4px;
  white-space: nowrap;
}

.metadata-item .n-icon {
  opacity: 0.7;
}

.metadata-value {
  font-weight: 500;
}

.model-name {
  color: var(--n-primary-color);
}

.tokens {
  color: #67c23a;
}

.duration {
  color: #e6a23c;
}
</style>
