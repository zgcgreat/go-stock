<script setup>
import {computed, h, onBeforeMount, onBeforeUnmount, onMounted,onUnmounted, ref,reactive, watch} from 'vue'
import {GetAIResponseResultList} from "../services/wails-bridge.js";
import {NButton, NEllipsis, NText} from "naive-ui";
import ResearchReport from "./researchReport.vue";
import AiRecommendStocksList from "./aiRecommendStocksList.vue";
import PromptTemplateList from "./promptTemplateList.vue";
import AllStockList from "./allStockList.vue";
import AllStockInfoList from "./allStockInfoList.vue";
import CronTaskManager from "./cron-task-manager.vue";
import TradingRecordManager from "./TradingRecordManager.vue";
import StockChangesMonitor from "./stockChangesMonitor.vue";
import {EventsOff, EventsOn} from "../services/wails-bridge.js";
import {useRoute} from 'vue-router'


const nowTab = ref("AI分析报告")
const route = useRoute()

// 初始化时设置标签页
onBeforeMount(() => {
  // 如果路由中有 name 参数则使用，否则默认为"AI分析报告"
  nowTab.value = route.query.name || "AI分析报告"
})

// 监听路由变化，自动切换标签页
watch(
  () => route.query.name,
  (newName) => {
    if (newName) {
      nowTab.value = newName
    }
  }
)

onBeforeUnmount(() => {
  EventsOff("changeResearchTab")
})

onUnmounted(() => {

});

EventsOn("changeResearchTab", async (msg) => {
  console.log("changeResearchTab", msg)
  updateTab(msg.name)
})
function updateTab(name) {
  nowTab.value = name
}
</script>

<template>
  <n-card>
    <n-tabs type="line" animated @update-value="updateTab" :value="nowTab" style="--wails-draggable:no-drag">
      <n-tab-pane name="AI分析报告">
        <ResearchReport/>
      </n-tab-pane>
      <n-tab-pane name="股票推荐记录">
        <AiRecommendStocksList/>
      </n-tab-pane>
      <n-tab-pane name="异动监控">
        <StockChangesMonitor/>
      </n-tab-pane>
      <n-tab-pane name="提示词模板">
        <PromptTemplateList/>
      </n-tab-pane>
      <n-tab-pane name="股票信息筛选">
        <AllStockList/>
      </n-tab-pane>
      <n-tab-pane name="定时任务">
        <CronTaskManager />
      </n-tab-pane>
      <n-tab-pane name="交易日志">
        <TradingRecordManager />
      </n-tab-pane>
<!--      <n-tab-pane name="全部股票信息">-->
<!--        <AllStockInfoList/>-->
<!--      </n-tab-pane>-->
    </n-tabs>
  </n-card>
</template>

<style scoped>
</style>
