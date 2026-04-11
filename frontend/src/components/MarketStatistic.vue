<script setup>
import { onMounted, onUnmounted, ref, computed, watch, nextTick } from 'vue'
import * as echarts from 'echarts'
import { GetTodayMarketStatistic, GetRecentDaysMarketStatistic, FetchAndSaveMarketStatistic } from '../services/wails-bridge.js'
import { NCard, NSpin, NText, NTag, NGrid, NGi, NStatistic, NNumberAnimation, NButton, useMessage } from 'naive-ui'

const props = defineProps({
  darkTheme: {
    type: Boolean,
    default: false
  },
  chartHeight: {
    type: Number,
    default: 300
  }
})

const loading = ref(false)
const todayData = ref([])
const recentData = ref([])
const sentimentGaugeChartRef = ref(null)
const trendChartRef = ref(null)
const distributionChartRef = ref(null)
let sentimentGaugeChart = null
let trendChart = null
let distributionChart = null
const isInitialized = ref(false)
const message = useMessage()

// 加载所有数据
async function loadData() {
  console.log('[MarketStatistic] 开始加载数据...')
  await fetchTodayData()
  await fetchRecentData()
  isInitialized.value = true
  console.log('[MarketStatistic] 数据加载完成')
}

// 手动刷新数据(先获取再加载)
async function handleRefresh() {
  try {
    loading.value = true
    message.loading('正在获取最新市场数据...')
    console.log('[MarketStatistic] 开始获取并保存数据...')
    await FetchAndSaveMarketStatistic()
    message.success('数据获取成功')
    // 重新加载显示
    await loadData()
  } catch (error) {
    console.error('获取市场数据失败:', error)
    message.error('获取数据失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 获取今日数据
async function fetchTodayData() {
  try {
    console.log('[MarketStatistic] 开始获取今日数据...')
    loading.value = true
    const data = await GetTodayMarketStatistic()
    console.log('[MarketStatistic] 今日数据:', data)
    todayData.value = Array.isArray(data) ? data : []
    
    if (todayData.value.length > 0) {
      // 使用最新的数据点
      const latest = todayData.value[todayData.value.length - 1]
      await nextTick()
      renderSentimentGauge(latest)
      renderDistribution(latest)
    }
  } catch (error) {
    console.error('获取今日市场统计数据失败:', error)
  } finally {
    loading.value = false
  }
}

// 获取最近7天数据
async function fetchRecentData() {
  try {
    console.log('[MarketStatistic] 开始获取近期数据...')
    const data = await GetRecentDaysMarketStatistic(7)
    console.log('[MarketStatistic] 近期数据:', data)
    recentData.value = Array.isArray(data) ? data : []
    
    if (recentData.value.length > 0) {
      await nextTick()
      renderTrendChart()
    }
  } catch (error) {
    console.error('获取近期市场统计数据失败:', error)
  }
}

// 渲染情绪仪表盘
function renderSentimentGauge(data) {
  if (!sentimentGaugeChartRef.value) return
  
  if (!sentimentGaugeChart) {
    sentimentGaugeChart = echarts.init(sentimentGaugeChartRef.value)
  }
  
  const upDownRatio = data.upDownRatio || 0
  let gaugeValue = Math.min(upDownRatio / 3 * 100, 100) // 归一化到0-100
  
  const option = {
    series: [{
      type: 'gauge',
      startAngle: 180,
      endAngle: 0,
      min: 0,
      max: 100,
      splitNumber: 5,
      radius: '90%',
      center: ['50%', '70%'],
      axisLine: {
        lineStyle: {
          width: 20,
          color: [
            [0.3, '#ff4d4f'],
            [0.5, '#faad14'],
            [0.7, '#52c41a'],
            [1, '#1890ff']
          ]
        }
      },
      pointer: {
        icon: 'path://M12.8,0.7l12,40.1H0.7L12.8,0.7z',
        length: '60%',
        width: 10,
        offsetCenter: [0, '-20%'],
        itemStyle: {
          color: 'auto'
        }
      },
      axisTick: {
        length: 12,
        lineStyle: {
          color: 'auto',
          width: 2
        }
      },
      splitLine: {
        length: 20,
        lineStyle: {
          color: 'auto',
          width: 5
        }
      },
      axisLabel: {
        color: '#464646',
        fontSize: 12,
        distance: -40,
        formatter: function (value) {
          if (value === 0) return '冰点'
          if (value === 25) return '偏弱'
          if (value === 50) return '中性'
          if (value === 75) return '偏强'
          if (value === 100) return '极强'
          return ''
        }
      },
      title: {
        offsetCenter: [0, '-40%'],
        fontSize: 16
      },
      detail: {
        fontSize: 24,
        offsetCenter: [0, '0%'],
        valueAnimation: true,
        formatter: function (value) {
          return data.sentimentDesc || '未知'
        },
        color: 'auto'
      },
      data: [{
        value: gaugeValue,
        name: '市场情绪'
      }]
    }]
  }
  
  sentimentGaugeChart.setOption(option)
}

// 渲染涨跌分布图
function renderDistribution(data) {
  if (!distributionChartRef.value) return
  
  if (!distributionChart) {
    distributionChart = echarts.init(distributionChartRef.value)
  }
  
  const upCount = data.upCount || 0
  const downCount = data.downCount || 0
  const limitUp = data.limitUp || 0
  const limitDown = data.limitDown || 0
  
  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '10%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: ['上涨', '下跌', '涨停', '跌停'],
      axisLabel: {
        fontSize: 12
      }
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        fontSize: 12
      }
    },
    series: [{
      type: 'bar',
      data: [
        {
          value: upCount,
          itemStyle: { color: '#ff4d4f' }
        },
        {
          value: downCount,
          itemStyle: { color: '#52c41a' }
        },
        {
          value: limitUp,
          itemStyle: { color: '#ff7875' }
        },
        {
          value: limitDown,
          itemStyle: { color: '#95de64' }
        }
      ],
      label: {
        show: true,
        position: 'top',
        fontSize: 12
      },
      barWidth: '50%'
    }]
  }
  
  distributionChart.setOption(option)
}

// 渲染趋势图
function renderTrendChart() {
  if (!trendChartRef.value) return
  
  if (!trendChart) {
    trendChart = echarts.init(trendChartRef.value)
  }
  
  // 按日期和时间排序
  const sortedData = [...recentData.value].sort((a, b) => {
    const dateA = new Date(a.dataDate + ' ' + a.dataTime)
    const dateB = new Date(b.dataDate + ' ' + b.dataTime)
    return dateA - dateB
  })
  
  const times = sortedData.map(item => {
    const date = new Date(item.dataDate + ' ' + item.dataTime)
    return `${date.getMonth() + 1}/${date.getDate()} ${item.dataTime}`
  })
  
  const upRatios = sortedData.map(item => item.upRatio || 0)
  const upDownRatios = sortedData.map(item => item.upDownRatio || 0)
  
  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross'
      }
    },
    legend: {
      data: ['上涨占比(%)', '涨跌比'],
      textStyle: {
        fontSize: 12
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '15%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: times,
      axisLabel: {
        fontSize: 10,
        rotate: 45
      }
    },
    yAxis: [
      {
        type: 'value',
        name: '上涨占比(%)',
        position: 'left',
        axisLabel: {
          fontSize: 11
        }
      },
      {
        type: 'value',
        name: '涨跌比',
        position: 'right',
        axisLabel: {
          fontSize: 11
        }
      }
    ],
    series: [
      {
        name: '上涨占比(%)',
        type: 'line',
        smooth: true,
        data: upRatios,
        itemStyle: { color: '#ff4d4f' },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(255, 77, 79, 0.3)' },
            { offset: 1, color: 'rgba(255, 77, 79, 0.05)' }
          ])
        }
      },
      {
        name: '涨跌比',
        type: 'line',
        smooth: true,
        yAxisIndex: 1,
        data: upDownRatios,
        itemStyle: { color: '#1890ff' }
      }
    ]
  }
  
  trendChart.setOption(option)
}

// 窗口大小改变时重新调整图表
function handleResize() {
  sentimentGaugeChart?.resize()
  trendChart?.resize()
  distributionChart?.resize()
}

onMounted(() => {
  console.log('[MarketStatistic] 组件已挂载')
  loadData()
  
  window.addEventListener('resize', handleResize)
})

// 监听图表容器变化,确保渲染
watch([sentimentGaugeChartRef, distributionChartRef, trendChartRef], () => {
  if (isInitialized.value && todayData.value.length > 0) {
    const latest = todayData.value[todayData.value.length - 1]
    nextTick(() => {
      renderSentimentGauge(latest)
      renderDistribution(latest)
    })
  }
  if (isInitialized.value && recentData.value.length > 0) {
    nextTick(() => {
      renderTrendChart()
    })
  }
})

// 提供手动刷新方法
defineExpose({
  refresh: loadData,
  fetchAndRefresh: handleRefresh
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  sentimentGaugeChart?.dispose()
  trendChart?.dispose()
  distributionChart?.dispose()
})

// 计算最新数据
const latestData = computed(() => {
  return todayData.value.length > 0 ? todayData.value[todayData.value.length - 1] : null
})
</script>

<template>
  <n-card :title="'市场统计数据'" :bordered="false">
    <template #header-extra>
      <n-button size="small" type="primary" @click="handleRefresh" :loading="loading">
        刷新数据
      </n-button>
    </template>
    <n-spin :show="loading">
      <div v-if="latestData" style="margin-bottom: 20px">
        <n-grid :cols="4" :y-gap="10" style="margin-bottom: 20px">
          <n-gi>
            <n-statistic label="上涨家数" :value="latestData.upCount">
              <template #prefix>
                <n-text type="error">↑</n-text>
              </template>
            </n-statistic>
          </n-gi>
          <n-gi>
            <n-statistic label="下跌家数" :value="latestData.downCount">
              <template #prefix>
                <n-text type="success">↓</n-text>
              </template>
            </n-statistic>
          </n-gi>
          <n-gi>
            <n-statistic label="涨停家数" :value="latestData.limitUp">
              <template #suffix>
                <n-tag type="error" size="small">涨停</n-tag>
              </template>
            </n-statistic>
          </n-gi>
          <n-gi>
            <n-statistic label="跌停家数" :value="latestData.limitDown">
              <template #suffix>
                <n-tag type="success" size="small">跌停</n-tag>
              </template>
            </n-statistic>
          </n-gi>
        </n-grid>
        
        <n-grid :cols="2" :x-gap="10" :y-gap="10">
          <n-gi>
            <div ref="sentimentGaugeChartRef" :style="{ height: chartHeight + 'px' }"></div>
          </n-gi>
          <n-gi>
            <div ref="distributionChartRef" :style="{ height: chartHeight + 'px' }"></div>
          </n-gi>
        </n-grid>
        
        <div v-if="recentData.length > 0" style="margin-top: 20px">
          <div ref="trendChartRef" :style="{ height: chartHeight + 'px' }"></div>
        </div>
      </div>
      
      <div v-else style="text-align: center; padding: 40px; color: #999">
        <div style="margin-bottom: 20px">暂无市场统计数据</div>
        <n-button type="primary" @click="handleRefresh" :loading="loading">
          点击获取最新数据
        </n-button>
      </div>
    </n-spin>
  </n-card>
</template>

<style scoped>
</style>
