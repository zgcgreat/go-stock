<script setup lang="ts">
import {onBeforeMount, ref} from 'vue'
import {LongTigerRank,BrowserOpenURL} from "../services/wails-bridge.js";
import {ArrowDownOutline} from "@vicons/ionicons5";
import _ from "lodash";
import KLineChart from "./KLineChart.vue";
import MoneyTrend from "./moneyTrend.vue";
import {NButton, NText, useMessage} from "naive-ui";
const message = useMessage()

const lhbList=  ref([])
const EXPLANATIONs = ref([])

const today = new Date();
const year = today.getFullYear();
const month = String(today.getMonth() + 1).padStart(2, '0'); // 月份从0开始，需要+1
const day = String(today.getDate()).padStart(2, '0');

// 常见格式：YYYY-MM-DD
const formattedDate = `${year}-${month}-${day}`;

const SearchForm=  ref({
  dateValue:  formattedDate,
  EXPLANATION:null,
})

onBeforeMount(() => {
  longTiger(formattedDate);
})
function longTiger_old(date) {
  if(date) {
    SearchForm.value.dateValue = date
  }
  let loading1=message.loading("正在获取龙虎榜数据...",{
    duration: 0,
  })
  LongTigerRank(date).then(res => {
    lhbList.value = res
    loading1.destroy()
    if (res.length === 0) {
      message.info("暂无数据,请切换日期")
    }
    EXPLANATIONs.value=_.uniqBy(_.map(lhbList.value,function (item){
      return {
        label: item['EXPLANATION'],
        value: item['EXPLANATION'],
      }
    }),'label');
  })
}

function longTiger(date) {
  if (date) {
    SearchForm.value.dateValue = date;
  }

  let loading1 = message.loading("正在获取龙虎榜数据...", {
    duration: 0,
  });

  const fetchDate = (currentDate, retryCount = 0) => {
    if (retryCount > 7) { // 防止无限循环，最多尝试7次
      lhbList.value = [];
      EXPLANATIONs.value = [];
      loading1.destroy();
      message.info("暂无历史数据");
      return;
    }

    LongTigerRank(currentDate).then(res => {
      if (res.length === 0) {
        const previousDate = new Date(currentDate);
        previousDate.setDate(previousDate.getDate() - 1);

        const year = previousDate.getFullYear();
        const month = String(previousDate.getMonth() + 1).padStart(2, '0');
        const day = String(previousDate.getDate()).padStart(2, '0');
        const prevFormattedDate = `${year}-${month}-${day}`;

        message.info(`当前日期 ${currentDate} 暂无数据，尝试查询前一日：${prevFormattedDate}`);

        SearchForm.value.dateValue = prevFormattedDate;
        fetchDate(prevFormattedDate, retryCount + 1); // 递归调用
      } else {
        lhbList.value = res;
        loading1.destroy();
        EXPLANATIONs.value = _.uniqBy(_.map(lhbList.value, function (item) {
          return {
            label: item['EXPLANATION'],
            value: item['EXPLANATION'],
          };
        }), 'label');
      }
    }).catch(err => {
      loading1.destroy();
      message.error("获取数据失败，请重试");
      console.error(err);
    });
  };

  fetchDate(date || formattedDate);
}

function handleEXPLANATION(value, option){
  SearchForm.value.EXPLANATION = value
  if(value){
    LongTigerRank(SearchForm.value.dateValue).then(res => {
      lhbList.value=_.filter(res, function(o) { return o['EXPLANATION']===value; });
      if (res.length === 0) {
        message.info("暂无数据,请切换日期")
      }
    })
  }else{
    longTiger(SearchForm.value.dateValue)
  }
}

// 个股K线弹窗（点击股票名称打开；对齐上游 0fb87dc：悬停弹层改点击弹窗，避免整页 hover 渲染 K 线）
const klineModal = ref({
  show: false,
  code: '',
  name: '',
})

function toKlineCode(secucode: string) {
  // 东财 SECUCODE 形如 "002241.SZ" → "sz002241"
  const parts = String(secucode || '').split('.')
  if (parts.length < 2) return String(secucode || '').toLowerCase()
  return (parts[1] + parts[0]).toLowerCase()
}

function showKline(item) {
  klineModal.value = {
    show: true,
    code: toKlineCode(item.SECUCODE),
    name: item.SECURITY_NAME_ABBR,
  }
}
</script>

<template>
  <n-form :model="SearchForm" >
    <n-grid :cols="24" :x-gap="24">
      <n-form-item-gi  :span="4" label="日期" path="dateValue" label-placement="left">
        <n-date-picker   v-model:formatted-value="SearchForm.dateValue"
                         value-format="yyyy-MM-dd"  type="date"  :on-update:value="(v,v2)=>longTiger(v2)"/>

      </n-form-item-gi>
      <n-form-item-gi :span="8" label="上榜原因" path="EXPLANATION" label-placement="left">
        <n-select  clearable placeholder="上榜原因过滤" v-model:value="SearchForm.EXPLANATION" :options="EXPLANATIONs" :on-update:value="handleEXPLANATION"/>
      </n-form-item-gi>
      <n-form-item-gi :span="10" label=""  label-placement="left">
        <n-text type="error">*当天的龙虎榜数据通常在收盘结束后一小时左右更新</n-text>
      </n-form-item-gi>
    </n-grid>
  </n-form>
  <n-table :single-line="false" striped>
    <n-thead>
      <n-tr>
        <n-th>代码</n-th>
        <!--                <n-th width="90px">日期</n-th>-->
        <n-th width="60px">名称</n-th>
        <n-th>收盘价</n-th>
        <n-th width="60px">涨跌幅</n-th>
        <n-th>龙虎榜净买额(万)</n-th>
        <n-th>龙虎榜买入额(万)</n-th>
        <n-th>龙虎榜卖出额(万)</n-th>
        <n-th>龙虎榜成交额(万)</n-th>
        <!--                <n-th>市场总成交额(万)</n-th>-->
        <!--                <n-th>净买额占总成交比</n-th>-->
        <!--                <n-th>成交额占总成交比</n-th>-->
        <n-th width="60px"  data-field="TURNOVERRATE">换手率<n-icon :component="ArrowDownOutline" /></n-th>
        <n-th>流通市值(亿)</n-th>
        <n-th>上榜原因</n-th>
        <!--                <n-th>解读</n-th>-->
      </n-tr>
    </n-thead>
    <n-tbody>
      <n-tr v-for="(item, index) in lhbList" :key="index">
        <n-td>
          <n-tag :bordered=false type="info">{{ item.SECUCODE.split('.')[1].toLowerCase()+item.SECUCODE.split('.')[0] }}</n-tag>
        </n-td>
        <!--                <n-td>
                          {{item.TRADE_DATE.substring(0,10)}}
                        </n-td>-->
        <n-td>
          <!--                  <n-text :type="item.CHANGE_RATE>0?'error':'success'">{{ item.SECURITY_NAME_ABBR }}</n-text>-->
          <n-button tag="a" text :type="item.CHANGE_RATE>0?'error':'success'" :bordered=false
                    title="点击查看K线" @click="showKline(item)">{{ item.SECURITY_NAME_ABBR }}</n-button>
        </n-td>
        <n-td>
          <n-text :type="item.CHANGE_RATE>0?'error':'success'">{{ item.CLOSE_PRICE }}</n-text>
        </n-td>
        <n-td>
          <n-text :type="item.CHANGE_RATE>0?'error':'success'">{{ (item.CHANGE_RATE).toFixed(2) }}%</n-text>
        </n-td>
        <n-td>
          <!--                  <n-text :type="item.BILLBOARD_NET_AMT>0?'error':'success'">{{ (item.BILLBOARD_NET_AMT/10000).toFixed(2) }}</n-text>-->


          <n-popover trigger="hover" placement="right">
            <template #trigger>
              <n-button tag="a"  text :type="item.BILLBOARD_NET_AMT>0?'error':'success'" :bordered=false >{{ (item.BILLBOARD_NET_AMT/10000).toFixed(2) }}</n-button>
            </template>
            <money-trend :code="item.SECUCODE.split('.')[1].toLowerCase()+item.SECUCODE.split('.')[0]" :name="item.SECURITY_NAME_ABBR" :days="360" :dark-theme="true" :chart-height="500" style="width: 800px"></money-trend>
          </n-popover>

        </n-td>
        <n-td>
          <n-text :type="'error'">{{ (item.BILLBOARD_BUY_AMT/10000).toFixed(2) }}</n-text>
        </n-td>
        <n-td>
          <n-text :type="'success'">{{ (item.BILLBOARD_SELL_AMT/10000).toFixed(2) }}</n-text>
        </n-td>
        <n-td>
          <n-text :type="'info'">{{ (item.BILLBOARD_DEAL_AMT/10000).toFixed(2) }}</n-text>
        </n-td>
        <!--                <n-td>-->
        <!--                  <n-text :type="'info'">{{ (item.ACCUM_AMOUNT/10000).toFixed(2) }}</n-text>-->
        <!--                </n-td>-->
        <!--                <n-td>-->
        <!--                  <n-text :type="item.DEAL_NET_RATIO>0?'error':'success'">{{ (item.DEAL_NET_RATIO).toFixed(2) }}%</n-text>-->
        <!--                </n-td>-->
        <!--                <n-td>-->
        <!--                  <n-text :type="'info'">{{ (item.DEAL_AMOUNT_RATIO).toFixed(2) }}%</n-text>-->
        <!--                </n-td>-->
        <n-td>
          <n-text :type="'info'">{{ (item.TURNOVERRATE).toFixed(2) }}%</n-text>
        </n-td>
        <n-td>
          <n-text :type="'info'">{{ (item.FREE_MARKET_CAP/100000000).toFixed(2) }}</n-text>
        </n-td>
        <n-td>
          <n-text :type="'info'">{{ item.EXPLANATION }}</n-text>
        </n-td>
        <!--                <n-td>
                          <n-text :type="item.CHANGE_RATE>0?'error':'success'">{{ item.EXPLAIN }}</n-text>
                        </n-td>-->
      </n-tr>
    </n-tbody>
  </n-table>

  <!-- 个股K线弹窗（点击股票名称打开） -->
  <n-modal v-model:show="klineModal.show" preset="card"
           :title="klineModal.name + '（' + klineModal.code + '）K线'"
           style="width: 920px; max-width: 95vw">
    <KLineChart :key="klineModal.code" style="width: 100%" :code="klineModal.code"
                :stock-name="klineModal.name" :chart-height="500" :dark-theme="true"/>
  </n-modal>
</template>

<style scoped>

</style>