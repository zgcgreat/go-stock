<script setup>
import {h, onBeforeUnmount, onMounted, ref, computed} from "vue";
import {
  AddPrompt,
  DelPrompt,
  ExportConfig,
  GetConfig,
  GetPromptTemplates,
  SendDingDingMessageByType,
  SendFeishuMessageByType,
  UpdateConfig,
  CheckSponsorCode,
  FetchAiModels,
  FetchAiModelInfo,
  GetContractProxy,
  SetContractProxy
} from "../services/wails-bridge.js";
import {NTag, NTooltip, NIcon, useMessage} from "naive-ui";
import {EventsEmit} from "../services/wails-bridge.js";
import {HelpCircleFilledIcon, HelpIcon} from "tdesign-icons-vue-next";
import {useIsWebMode} from '../composables/useResponsive'

const message = useMessage()
// Web 端不走赞助码：VIP 由本地用户管理（管理员设置 vipLevel）发放，设置页隐藏赞助码入口
const {isWebMode} = useIsWebMode()

// 合约代理（币安 USDT-M / Bitget 美股永续共用一项）：
// 与爬虫 httpProxy 完全独立，且是「服务端出网配置」而非用户偏好——
// 代理值由后端 client 读取，多个用户共享同一份，故走 /contracts/proxy 专用端点，
// 不进 UpdateConfig 的 per-user 整行覆盖路径（那会覆盖其它字段的并发修改）。
const binanceProxy = ref('')

function loadContractProxy() {
  GetContractProxy()
    .then(cfg => { binanceProxy.value = (cfg && cfg.binanceProxy) || '' })
    .catch(() => { /* 读取失败保持空=直连 */ })
}

function onBinanceProxyInput(v) {
  binanceProxy.value = v || ''
  SetContractProxy(binanceProxy.value, '')
    .then(() => message.success('合约代理已保存，留空表示直连'))
    .catch(err => message.error('合约代理保存失败: ' + (err?.response?.data?.message || err?.message || err)))
}


const formRef = ref(null)
const formValue = ref({
  ID: 1,
  tushareToken: '',
  iwencaiApiKey: '',
  emApiKey: '',
  dingPush: {
    enable: false,
    dingRobot: ''
  },
  feishuPush: {
    enable: false,
    feishuRobot: '',
    feishuSecret: ''
  },
  localPush: {
    enable: true,
  },
  updateBasicInfoOnStart: false,
  refreshInterval: 1,
  openAI: {
    enable: false,
    aiConfigs: [], // AI配置列表
    prompt: "",
    questionTemplate: "{{stockName}}分析和总结",
    crawlTimeOut: 30,
    kDays: 30,
    httpProxy:"",
    httpProxyEnabled:false,
  },
  enableDanmu: false,
  browserPath: '',
  enableNews: false,
  darkTheme: true,
  enableFund: false,
  enablePushNews: true,
  enableOnlyPushRedNews: true,
  sponsorCode: "",
  httpProxy:"",
  httpProxyEnabled:false,
  // [007/c0f43ef] 全局代理作用范围：overseas=仅国外站点与数据（默认）/ global=所有请求
  httpProxyMode:"overseas",
  // [007/6562395+f455797] OKX（欧易）交易设置：模拟盘/实盘双凭证、交易开关、
  // 信号自动下单（模拟盘/实盘独立开关+金额）、防爆仓风控、持仓止盈止损
  okxApiKey:"",
  okxSecretKey:"",
  okxPassphrase:"",
  okxLiveApiKey:"",
  okxLiveSecretKey:"",
  okxLivePassphrase:"",
  okxEnableTrade:false,
  okxSimulated:true,
  okxSignalTradeEnable:false,
  okxSignalTradeUSDT:"100",
  okxSignalTradeLive:false,
  okxSignalTradeLiveUSDT:"100",
  okxSignalTradeAmountMode:"fixed",
  okxSignalTradeAmountRatio:"0.25",
  okxSignalTradeMinScore:"",
  okxSignalTradeCooldownMin:"",
  okxRiskMaxPositions:"3",
  okxRiskMaxMarginRatio:"30",
  okxSignalTradeInstType:"SWAP",
  okxSignalTradeLever:"",
  okxSignalTradeKlts:"",
  okxSignalStockEnable:false,
  okxSignalStockUSDT:"100",
  okxTpSlEnable:false,
  okxTakeProfitPct:"",
  okxStopLossPct:"",
  okxTpSlMode:"fixed",
  okxTpSlVolKlt:"15m",
  okxTpSlVolBars:"20",
  okxTpSlTpMult:"1.5",
  okxTpSlSlMult:"1",
  enableAgent: false,
  qgqpBId: '',
  updateChannel: 'release',
  promptPlazaApiBase: '',
})

// 添加一个新的AI配置到列表
let aiConfigIdCounter = Date.now()
function addAiConfig() {
  formValue.value.openAI.aiConfigs.push({
    _uid: ++aiConfigIdCounter, // 唯一标识，用于 Vue key 绑定
    name: '',
    baseUrl: 'https://api.deepseek.com',
    apiKey: '',
    modelName: 'deepseek-reasoner',
    temperature: 0.1,
    maxTokens: 8192,
    timeOut: 6000,
    httpProxy:"",
    httpProxyEnabled:false,
    thinking: true,
  });
}

// 从列表中移除一个AI配置
function removeAiConfig(index) {
  const originalCount = formValue.value.openAI.aiConfigs.length;
  // 使用filter创建新数组确保响应式更新
  formValue.value.openAI.aiConfigs = formValue.value.openAI.aiConfigs.filter((_, i) => i !== index);
}

const updateChannelOptions = [
  { label: 'Release（稳定版）', value: 'release' },
  { label: 'Pre-release（预发布版）', value: 'pre' },
  { label: 'Dev（开发版）', value: 'dev' },
]

// [007] OKX 交易设置辅助选项（与上游一致）
// 仓位比例选项（按可用余额比例下单模式）
const okxAmountRatioOptions = [
  { label: '1/4 仓', value: '0.25' },
  { label: '1/3 仓', value: '0.3333' },
  { label: '1/2 仓', value: '0.5' },
  { label: '全仓', value: '1' },
]

// 杠杆选项（手动下单与信号自动下单共用）
const okxLeverOptions = [
  { label: '账户默认', value: '' },
  { label: '1x', value: '1' },
  { label: '2x', value: '2' },
  { label: '3x', value: '3' },
  { label: '5x', value: '5' },
  { label: '10x', value: '10' },
  { label: '20x', value: '20' },
]

// 信号下单的 K 线周期多选（options 同信号监控面板口径）
const okxKltOptions = [
  { label: '1 分钟', value: '1' },
  { label: '5 分钟', value: '5' },
  { label: '15 分钟', value: '15' },
  { label: '30 分钟', value: '30' },
  { label: '60 分钟', value: '60' },
  { label: '日K', value: '101' },
  { label: '周K', value: '102' },
  { label: '月K', value: '103' },
]

// 交易周期过滤的多选绑定：formValue.okxSignalTradeKlts 为逗号分隔串
const okxSignalTradeKltsSel = computed({
  get: () => (formValue.value.okxSignalTradeKlts || '').split(',').map(s => s.trim()).filter(Boolean),
  set: (v) => { formValue.value.okxSignalTradeKlts = (v || []).join(',') },
})

async function fetchAiModels(aiConfig) {
  if (!aiConfig.baseUrl || !aiConfig.apiKey) {
    message.warning('请先填写接口地址和 apiKey')
    return
  }
  if (aiConfig._loadingModels) {
    return
  }
  aiConfig._loadingModels = true
  try {
    const list = await FetchAiModels(aiConfig.baseUrl, aiConfig.apiKey)
    const options = (list || []).map(id => ({ label: id, value: id }))
    aiConfig._modelOptions = options
    if (!aiConfig.modelName && options.length > 0) {
      aiConfig.modelName = options[0].value
      onModelNameChange(aiConfig, aiConfig.modelName)
    }
    if (!options.length) {
      message.warning('未从接口获取到可用模型，请检查地址和 apiKey')
    }
  } catch (e) {
    console.error('FetchAiModels error', e)
    message.error('获取模型列表失败，请检查接口地址和 apiKey')
  } finally {
    aiConfig._loadingModels = false
  }
}


const promptTemplates = ref([])
const aiConfigExpandedNames = ref([])

const aiPlatformOptions = [
  { label: 'DeepSeek (https://api.deepseek.com)', value: 'https://api.deepseek.com' },
  { label: '硅基流动 (https://api.siliconflow.cn/v1)', value: 'https://api.siliconflow.cn/v1' },
  { label: '智谱AI(GLM) (https://open.bigmodel.cn/api/paas/v4)', value: 'https://open.bigmodel.cn/api/paas/v4' },
  { label: '字节豆包(火山引擎) (https://ark.cn-beijing.volces.com/api/v3)', value: 'https://ark.cn-beijing.volces.com/api/v3' },
  { label: '阿里云百炼 (https://dashscope.aliyuncs.com/compatible-mode/v1)', value: 'https://dashscope.aliyuncs.com/compatible-mode/v1' },
  { label: 'Moonshot(月之暗面) (https://api.moonshot.cn/v1)', value: 'https://api.moonshot.cn/v1' },
  { label: '腾讯混元 (https://api.hunyuan.cloud.tencent.com/v1)', value: 'https://api.hunyuan.cloud.tencent.com/v1' },
  { label: '讯飞星火 (https://spark-api-open.xf-yun.com/v1)', value: 'https://spark-api-open.xf-yun.com/v1' },
  { label: '零一万物 (https://api.lingyiwanwu.com/v1)', value: 'https://api.lingyiwanwu.com/v1' },
  { label: 'MiniMax (https://api.minimax.chat/v1)', value: 'https://api.minimax.chat/v1' },
  { label: '小米MiMo TokenPlan (https://token-plan-cn.xiaomimimo.com/v1)', value: 'https://token-plan-cn.xiaomimimo.com/v1' },
  { label: '小米MiMo (https://api.xiaomimimo.com/v1)', value: 'https://api.xiaomimimo.com/v1' },
  { label: '腾讯云TokenHub (https://tokenhub.tencentmaas.com/v1)', value: 'https://tokenhub.tencentmaas.com/v1' },
  { label: 'OpenAI (https://api.openai.com/v1)', value: 'https://api.openai.com/v1' },
  { label: 'Azure OpenAI (https://YOUR_RESOURCE.openai.azure.com)', value: 'https://YOUR_RESOURCE.openai.azure.com' },
  { label: 'OpenRouter (https://openrouter.ai/api/v1)', value: 'https://openrouter.ai/api/v1' },
  { label:'Ollama (http://localhost:11434/v1)', value: 'http://localhost:11434/v1' },
]

function getPlatformName(baseUrl) {
  if (!baseUrl) return ''
  const platform = aiPlatformOptions.find(opt => opt.value === baseUrl)
  if (platform) {
    const idx = platform.label.indexOf(' (')
    return idx > 0 ? platform.label.substring(0, idx) : platform.label
  }
  return ''
}

function onBaseUrlChange(aiConfig, newBaseUrl) {
  const platformName = getPlatformName(newBaseUrl)
  if (platformName && aiConfig.name && !aiConfig.name.startsWith(platformName)) {
    aiConfig.name = platformName + '-' + aiConfig.name
  } else if (platformName && !aiConfig.name) {
    aiConfig.name = platformName
  }
}

function onModelNameChange(aiConfig, newModelName) {
  if (!newModelName) return
  const platformName = getPlatformName(aiConfig.baseUrl)
  const baseName = platformName || 'AI'
  
  if (!aiConfig.name) {
    aiConfig.name = baseName + '-' + newModelName
  } else if (aiConfig.name === platformName) {
    aiConfig.name = platformName + '-' + newModelName
  } else {
    const parts = aiConfig.name.split('-')
    if (parts.length >= 2 && parts[0] === platformName) {
      parts[parts.length - 1] = newModelName
      aiConfig.name = parts.join('-')
    } else if (!aiConfig.name.endsWith(newModelName)) {
      aiConfig.name = aiConfig.name + '-' + newModelName
    }
  }

  fetchModelInfo(aiConfig, newModelName)
}

async function fetchModelInfo(aiConfig, modelName) {
  if (!modelName || !aiConfig.baseUrl) return
  try {
    const info = await FetchAiModelInfo(aiConfig.baseUrl, aiConfig.apiKey || '', modelName)
    if (info && info.maxTokens > 0) {
      aiConfig.maxTokens = info.maxTokens
      const sourceLabel = info.source === 'api' ? 'API' : '内置数据'
      message.success(`已自动设置 ${modelName} 的 MaxTokens 为 ${info.maxTokens}（来源：${sourceLabel}）`)
    }
  } catch (e) {
    console.error('FetchAiModelInfo error', e)
  }
}

onMounted(() => {
  loadContractProxy()
  GetConfig().then(res => {
    console.log('GetConfig 返回数据:', res)
    formValue.value.ID = res.ID || 1
    formValue.value.tushareToken = res.tushareToken
    formValue.value.iwencaiApiKey = res.iwencaiApiKey || ''
    formValue.value.emApiKey = res.emApiKey || ''
    formValue.value.dingPush = {
      enable: res.dingPushEnable,
      dingRobot: res.dingRobot
    }
    formValue.value.feishuPush = {
      enable: res.feishuPushEnable,
      feishuRobot: res.feishuRobot,
      feishuSecret: res.feishuSecret
    }
    formValue.value.localPush = {
      enable: res.localPushEnable,
    }
    formValue.value.updateBasicInfoOnStart = res.updateBasicInfoOnStart
    formValue.value.refreshInterval = res.refreshInterval
    // 加载AI配置
    formValue.value.openAI = {
      enable: res.openAiEnable,
      aiConfigs: res.aiConfigs || [],
      prompt: res.prompt,
      questionTemplate: res.questionTemplate ? res.questionTemplate : '{{stockName}}分析和总结',
      crawlTimeOut: res.crawlTimeOut,
      kDays: res.kDays,
      httpProxy:"",
      httpProxyEnabled:false,
    }


    formValue.value.enableDanmu = res.enableDanmu
    formValue.value.browserPath = res.browserPath
    formValue.value.enableNews = res.enableNews
    formValue.value.darkTheme = res.darkTheme
    formValue.value.enableFund = res.enableFund
    formValue.value.enablePushNews = res.enablePushNews
    formValue.value.enableOnlyPushRedNews = res.enableOnlyPushRedNews
    formValue.value.sponsorCode = res.sponsorCode
    formValue.value.httpProxy=res.httpProxy;
    formValue.value.httpProxyEnabled=res.httpProxyEnabled;
    // [007] 代理作用范围 + OKX 交易设置回填
    formValue.value.httpProxyMode=res.httpProxyMode || "overseas";
    formValue.value.okxApiKey=res.okxApiKey || "";
    formValue.value.okxSecretKey=res.okxSecretKey || "";
    formValue.value.okxPassphrase=res.okxPassphrase || "";
    formValue.value.okxLiveApiKey=res.okxLiveApiKey || "";
    formValue.value.okxLiveSecretKey=res.okxLiveSecretKey || "";
    formValue.value.okxLivePassphrase=res.okxLivePassphrase || "";
    formValue.value.okxEnableTrade=res.okxEnableTrade === true;
    formValue.value.okxSimulated=res.okxSimulated === true;
    formValue.value.okxSignalTradeEnable=res.okxSignalTradeEnable === true;
    formValue.value.okxSignalTradeUSDT=res.okxSignalTradeUSDT || "100";
    formValue.value.okxSignalTradeLive=res.okxSignalTradeLive === true;
    formValue.value.okxSignalTradeLiveUSDT=res.okxSignalTradeLiveUSDT || "100";
    formValue.value.okxSignalTradeAmountMode=res.okxSignalTradeAmountMode || "fixed";
    formValue.value.okxSignalTradeAmountRatio=res.okxSignalTradeAmountRatio || "0.25";
    formValue.value.okxRiskMaxPositions=res.okxRiskMaxPositions ?? "3";
    formValue.value.okxRiskMaxMarginRatio=res.okxRiskMaxMarginRatio ?? "30";
    formValue.value.okxSignalTradeInstType=res.okxSignalTradeInstType || "SWAP";
    formValue.value.okxSignalTradeLever=res.okxSignalTradeLever || "";
    formValue.value.okxSignalTradeKlts=res.okxSignalTradeKlts || "";
    formValue.value.okxSignalTradeMinScore=res.okxSignalTradeMinScore || "";
    formValue.value.okxSignalTradeCooldownMin=res.okxSignalTradeCooldownMin || "";
    formValue.value.okxSignalStockEnable=res.okxSignalStockEnable === true;
    formValue.value.okxSignalStockUSDT=res.okxSignalStockUSDT || "100";
    formValue.value.okxTpSlEnable=res.okxTpSlEnable === true;
    formValue.value.okxTakeProfitPct=res.okxTakeProfitPct || "";
    formValue.value.okxStopLossPct=res.okxStopLossPct || "";
    formValue.value.okxTpSlMode=res.okxTpSlMode || "fixed";
    formValue.value.okxTpSlVolKlt=res.okxTpSlVolKlt || "15m";
    formValue.value.okxTpSlVolBars=res.okxTpSlVolBars || "20";
    formValue.value.okxTpSlTpMult=res.okxTpSlTpMult || "1.5";
    formValue.value.okxTpSlSlMult=res.okxTpSlSlMult || "1";
    formValue.value.enableAgent = res.enableAgent;
    formValue.value.qgqpBId = res.qgqpBId;
    formValue.value.updateChannel = res.updateChannel || 'release';
    formValue.value.promptPlazaApiBase = res.promptPlazaApiBase || '';

  })

  GetPromptTemplates("", "").then(res => {
    promptTemplates.value = res
  })
})
onBeforeUnmount(() => {
  message.destroyAll()
})

function saveConfig() {
  console.log('开始保存设置', formValue.value);
  // 构建配置时，包含aiConfigs列表
  let config = {
    ID: formValue.value.ID,
    dingPushEnable: formValue.value.dingPush.enable,
    dingRobot: formValue.value.dingPush.dingRobot,
    feishuPushEnable: formValue.value.feishuPush.enable,
    feishuRobot: formValue.value.feishuPush.feishuRobot,
    feishuSecret: formValue.value.feishuPush.feishuSecret,
    localPushEnable: formValue.value.localPush.enable,
    updateBasicInfoOnStart: formValue.value.updateBasicInfoOnStart,
    refreshInterval: formValue.value.refreshInterval,
    openAiEnable: formValue.value.openAI.enable,
    aiConfigs: formValue.value.openAI.aiConfigs,
    // 序列化aiConfigs列表以传递给后端
    tushareToken: formValue.value.tushareToken,
    iwencaiApiKey: formValue.value.iwencaiApiKey,
    emApiKey: formValue.value.emApiKey,
    prompt: formValue.value.openAI.prompt,
    questionTemplate: formValue.value.openAI.questionTemplate,
    crawlTimeOut: formValue.value.openAI.crawlTimeOut,
    kDays: formValue.value.openAI.kDays,
    enableDanmu: formValue.value.enableDanmu,
    browserPath: formValue.value.browserPath,
    enableNews: formValue.value.enableNews,
    darkTheme: formValue.value.darkTheme,
    enableFund: formValue.value.enableFund,
    enablePushNews: formValue.value.enablePushNews,
    enableOnlyPushRedNews: formValue.value.enableOnlyPushRedNews,
    sponsorCode: formValue.value.sponsorCode,
    httpProxy:formValue.value.httpProxy,
    httpProxyEnabled:formValue.value.httpProxyEnabled,
    // [007] 代理作用范围 + OKX 交易设置提交
    httpProxyMode:formValue.value.httpProxyMode || "overseas",
    okxApiKey:formValue.value.okxApiKey,
    okxSecretKey:formValue.value.okxSecretKey,
    okxPassphrase:formValue.value.okxPassphrase,
    okxLiveApiKey:formValue.value.okxLiveApiKey,
    okxLiveSecretKey:formValue.value.okxLiveSecretKey,
    okxLivePassphrase:formValue.value.okxLivePassphrase,
    okxEnableTrade:formValue.value.okxEnableTrade,
    okxSimulated:formValue.value.okxSimulated,
    okxSignalTradeEnable:formValue.value.okxSignalTradeEnable,
    okxSignalTradeUSDT:formValue.value.okxSignalTradeUSDT,
    okxSignalTradeLive:formValue.value.okxSignalTradeLive,
    okxSignalTradeLiveUSDT:formValue.value.okxSignalTradeLiveUSDT,
    okxSignalTradeAmountMode:formValue.value.okxSignalTradeAmountMode || "fixed",
    okxSignalTradeAmountRatio:formValue.value.okxSignalTradeAmountRatio || "0.25",
    okxRiskMaxPositions:formValue.value.okxRiskMaxPositions ?? "3",
    okxRiskMaxMarginRatio:formValue.value.okxRiskMaxMarginRatio ?? "30",
    okxSignalTradeInstType:formValue.value.okxSignalTradeInstType,
    okxSignalTradeLever:formValue.value.okxSignalTradeLever || "",
    okxSignalTradeKlts:formValue.value.okxSignalTradeKlts,
    okxSignalTradeMinScore:formValue.value.okxSignalTradeMinScore || "",
    okxSignalTradeCooldownMin:formValue.value.okxSignalTradeCooldownMin || "",
    okxSignalStockEnable:formValue.value.okxSignalStockEnable,
    okxSignalStockUSDT:formValue.value.okxSignalStockUSDT,
    okxTpSlEnable:formValue.value.okxTpSlEnable,
    okxTakeProfitPct:formValue.value.okxTakeProfitPct,
    okxStopLossPct:formValue.value.okxStopLossPct,
    okxTpSlMode:formValue.value.okxTpSlMode,
    okxTpSlVolKlt:formValue.value.okxTpSlVolKlt,
    okxTpSlVolBars:formValue.value.okxTpSlVolBars,
    okxTpSlTpMult:formValue.value.okxTpSlTpMult,
    okxTpSlSlMult:formValue.value.okxTpSlSlMult,
    enableAgent: formValue.value.enableAgent,
    qgqpBId: formValue.value.qgqpBId,
    updateChannel: formValue.value.updateChannel,
    promptPlazaApiBase: formValue.value.promptPlazaApiBase,
  }

  if (!isWebMode.value && config.sponsorCode) {
    CheckSponsorCode(config.sponsorCode).then(res => {
      if (!res.code) {
        message.warning(res.msg || '赞助码验证失败')
      }
    })
  }

  UpdateConfig(config).then(res => {
    // 桌面端返回 "保存成功！"，Web端返回 "配置更新成功"
    if (res === '保存成功！' || res === '配置更新成功') {
      message.success('保存成功！')
    } else {
      message.error(res)
    }
    EventsEmit("updateSettings", config);
  })
}


function getHeight() {
  return document.documentElement.clientHeight
}

function sendTestNotice() {
  let markdown = "### go-stock test\n" + new Date()
  let msg = '{' +
      '     "msgtype": "markdown",' +
      '     "markdown": {' +
      '         "title":"go-stock' + new Date() + '",' +
      '         "text": "' + markdown + '"' +
      '     },' +
      '      "at": {' +
      '          "isAtAll": true' +
      '      }' +
      ' }'

  SendDingDingMessageByType(msg, "test-" + new Date().getTime(), 1).then(res => {
    message.info(res)
  })
}

function sendFeishuTestNotice() {
  let markdown = "### go-stock 飞书测试\n" + new Date()
  let msg = JSON.stringify({
    msg_type: "interactive",
    card: {
      schema: "2.0",
      header: {
        title: {
          tag: "plain_text",
          content: "go-stock 飞书测试 " + new Date()
        }
      },
      body: {
        elements: [
          {
            tag: "markdown",
            content: '<at id=all></at>\n' + markdown
          }
        ]
      }
    }
  })
  SendFeishuMessageByType(msg, "test-feishu-" + new Date().getTime(), 1).then(res => {
    message.info(res)
  })
}

function exportConfig() {
  ExportConfig().then(res => {
    message.info(res)
  })
}

function importConfig() {
  let input = document.createElement('input');
  input.type = 'file';
  input.accept = '.json';
  input.onchange = (e) => {
    let file = e.target.files[0];
    let reader = new FileReader();
    reader.onload = (e) => {
      let config = JSON.parse(e.target.result);
      formValue.value.ID = config.ID
      formValue.value.tushareToken = config.tushareToken
      formValue.value.iwencaiApiKey = config.iwencaiApiKey || ''
      formValue.value.emApiKey = config.emApiKey || ''
      formValue.value.dingPush = {
        enable: config.dingPushEnable,
        dingRobot: config.dingRobot
      }
      formValue.value.localPush = {
        enable: config.localPushEnable,
      }
      formValue.value.updateBasicInfoOnStart = config.updateBasicInfoOnStart
      formValue.value.refreshInterval = config.refreshInterval
      // 导入AI配置
      formValue.value.openAI = {
        enable: config.openAiEnable,
        aiConfigs: config.aiConfigs || [],
        prompt: config.prompt,
        questionTemplate: config.questionTemplate,
        crawlTimeOut: config.crawlTimeOut,
        kDays: config.kDays
      }
      formValue.value.enableDanmu = config.enableDanmu
      formValue.value.browserPath = config.browserPath
      formValue.value.enableNews = config.enableNews
      formValue.value.darkTheme = config.darkTheme
      formValue.value.enableFund = config.enableFund
      formValue.value.enablePushNews = config.enablePushNews
      formValue.value.enableOnlyPushRedNews = config.enableOnlyPushRedNews
      formValue.value.sponsorCode = config.sponsorCode
      formValue.value.httpProxy=config.httpProxy
      formValue.value.httpProxyEnabled=config.httpProxyEnabled
      // [007] 代理作用范围 + OKX 设置导入
      formValue.value.httpProxyMode=config.httpProxyMode || "overseas"
      formValue.value.okxApiKey=config.okxApiKey || ""
      formValue.value.okxSecretKey=config.okxSecretKey || ""
      formValue.value.okxPassphrase=config.okxPassphrase || ""
      formValue.value.okxLiveApiKey=config.okxLiveApiKey || ""
      formValue.value.okxLiveSecretKey=config.okxLiveSecretKey || ""
      formValue.value.okxLivePassphrase=config.okxLivePassphrase || ""
      formValue.value.okxEnableTrade=config.okxEnableTrade === true
      formValue.value.okxSimulated=config.okxSimulated === true
      formValue.value.okxSignalTradeEnable=config.okxSignalTradeEnable === true
      formValue.value.okxSignalTradeUSDT=config.okxSignalTradeUSDT || "100"
      formValue.value.okxSignalTradeLive=config.okxSignalTradeLive === true
      formValue.value.okxSignalTradeLiveUSDT=config.okxSignalTradeLiveUSDT || "100"
      formValue.value.okxSignalTradeAmountMode=config.okxSignalTradeAmountMode || "fixed"
      formValue.value.okxSignalTradeAmountRatio=config.okxSignalTradeAmountRatio || "0.25"
      formValue.value.okxRiskMaxPositions=config.okxRiskMaxPositions ?? "3"
      formValue.value.okxRiskMaxMarginRatio=config.okxRiskMaxMarginRatio ?? "30"
      formValue.value.okxSignalTradeInstType=config.okxSignalTradeInstType || "SWAP"
      formValue.value.okxSignalTradeLever=config.okxSignalTradeLever || ""
      formValue.value.okxSignalTradeKlts=config.okxSignalTradeKlts || ""
      formValue.value.okxSignalTradeMinScore=config.okxSignalTradeMinScore || ""
      formValue.value.okxSignalTradeCooldownMin=config.okxSignalTradeCooldownMin || ""
      formValue.value.okxSignalStockEnable=config.okxSignalStockEnable === true
      formValue.value.okxSignalStockUSDT=config.okxSignalStockUSDT || "100"
      formValue.value.okxTpSlEnable=config.okxTpSlEnable === true
      formValue.value.okxTakeProfitPct=config.okxTakeProfitPct || ""
      formValue.value.okxStopLossPct=config.okxStopLossPct || ""
      formValue.value.okxTpSlMode=config.okxTpSlMode || "fixed"
      formValue.value.okxTpSlVolKlt=config.okxTpSlVolKlt || "15m"
      formValue.value.okxTpSlVolBars=config.okxTpSlVolBars || "20"
      formValue.value.okxTpSlTpMult=config.okxTpSlTpMult || "1.5"
      formValue.value.okxTpSlSlMult=config.okxTpSlSlMult || "1"
      formValue.value.enableAgent = config.enableAgent
      formValue.value.qgqpBId = config.qgqpBId
      formValue.value.updateChannel = config.updateChannel || 'release'
    };
    reader.readAsText(file);
  };
  input.click();
}


window.onerror = function (event, source, lineno, colno, error) {
  EventsEmit("frontendError", {
    page: "settings.vue",
    message: event,
    source: source,
    lineno: lineno,
    colno: colno,
    error: error ? error.stack : null
  });
  return true;
};

const showManagePromptsModal = ref(false)
const promptTypeOptions = [
  {label: "模型系统Prompt", value: '模型系统Prompt'},
  {label: "模型用户Prompt", value: '模型用户Prompt'},]
const formPromptRef = ref(null)
const formPrompt = ref({
  ID: 0,
  Name: '',
  Content: '',
  Type: '',
})

function managePrompts() {
  formPrompt.value.ID = 0
  showManagePromptsModal.value = true
}

function savePrompt() {
  AddPrompt(formPrompt.value).then(res => {
    message.success(res)
    GetPromptTemplates("", "").then(res => {
      promptTemplates.value = res
    })
    showManagePromptsModal.value = false
  })
}

function editPrompt(prompt) {
  formPrompt.value.ID = prompt.ID
  formPrompt.value.Name = prompt.name
  formPrompt.value.Content = prompt.content
  formPrompt.value.Type = prompt.type
  showManagePromptsModal.value = true
}

function deletePrompt(ID) {
  DelPrompt(ID).then(res => {
    message.success(res)
    GetPromptTemplates("", "").then(res => {
      promptTemplates.value = res
    })
  })
}
</script>

<template>
  <n-flex justify="left" style="text-align: left; --wails-draggable:no-drag">
    <n-form ref="formRef" :label-placement="'left'" :label-align="'left'">
      <n-space vertical size="large">
        <n-card :title="() => h(NTag, { type: 'primary', bordered: false }, () => '基础设置')" size="small">
          <n-grid :cols="24" :x-gap="24" style="text-align: left">
<!--            <n-form-item-gi :span="10" label="Tushare Token：" path="tushareToken">
              <n-input type="text" placeholder="Tushare api token" v-model:value="formValue.tushareToken" clearable/>
            </n-form-item-gi>-->
            <n-form-item-gi :span="4" label="启动时更新基础信息：" path="updateBasicInfoOnStart">
              <n-switch v-model:value="formValue.updateBasicInfoOnStart"/>
            </n-form-item-gi>
            <n-form-item-gi :span="4" label="数据刷新间隔：" path="refreshInterval">
              <n-input-number v-model:value="formValue.refreshInterval" placeholder="请输入数据刷新间隔(秒)">
                <template #suffix>秒</template>
              </n-input-number>
            </n-form-item-gi>
            <n-form-item-gi :span="6" label="暗黑主题：" path="darkTheme">
              <n-switch v-model:value="formValue.darkTheme"/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" label="更新通道：" path="updateChannel">
              <n-select v-model:value="formValue.updateChannel" :options="updateChannelOptions" />
              <n-tooltip placement="top">
                <template #trigger>
                  <n-icon color="#0e7a0d" size="20">
                    <HelpCircleFilledIcon />
                  </n-icon>
                </template>
                <template #default>
                  <n-gradient-text :type="'warning'">
                  <div style="max-width: 400px;text-align: left">
                    更新通道说明：<br>
                    <b>Release（稳定版）</b>：仅接收正式发布版本，稳定性最高<br>
                    <b>Pre-release（预发布版）</b>：包含预发布版本，可提前体验新功能<br>
                    <b>Dev（开发版）</b>：包含所有可用版本，获取最新开发进度
                  </div>
                  </n-gradient-text>
                </template>
              </n-tooltip>
            </n-form-item-gi>
            <n-form-item-gi :span="10" label="浏览器安装路径：" path="browserPath">
              <n-input type="text" placeholder="浏览器安装路径" v-model:value="formValue.browserPath" clearable/>
            </n-form-item-gi>
           <n-form-item-gi :span="3" label="指数基金：" path="enableFund">
              <n-switch v-model:value="formValue.enableFund"/>
            </n-form-item-gi>
            <n-form-item-gi :span="3" label="AI智能体：" path="enableAgent">
              <n-switch v-model:value="formValue.enableAgent"/>
            </n-form-item-gi>
            <n-form-item-gi :span="11" label="东财唯一标识：" path="qgqpBId">
              <n-input type="text" placeholder="东财唯一标识" v-model:value="formValue.qgqpBId" clearable/>
              <n-tooltip placement="top">
                <template #trigger>
                  <n-icon color="#0e7a0d" size="20">
                    <HelpCircleFilledIcon />
                  </n-icon>
                </template>
                <template #default>
                  <n-gradient-text :type="'warning'">
                  <div style="max-width: 400px;text-align: left">
                    获取方法：<br>
                    打开浏览器,访问东财网站，<br>
                    按F12打开开发人员工具-》网络面板，<br>
                    随便点开一个请求，复制请求cookie中qgqp_b_id对应的值。
                  </div>
                  </n-gradient-text>
                </template>
              </n-tooltip>
            </n-form-item-gi>

            <n-form-item-gi :span="11" label="问财API密钥：" path="iwencaiApiKey">
              <n-input type="password" placeholder="同花顺问财开放平台API Key" v-model:value="formValue.iwencaiApiKey" clearable show-password-on="click"/>
              <n-tooltip placement="top">
                <template #trigger>
                  <n-icon color="#0e7a0d" size="20">
                    <HelpCircleFilledIcon />
                  </n-icon>
                </template>
                <template #default>
                  <n-gradient-text :type="'warning'">
                  <div style="max-width: 400px;text-align: left">
                    获取方法：<br>
                    访问同花顺问财开放平台：<br>
                    <a href="https://open.iwencai.com" target="_blank" style="color: #63e2b7">https://www.iwencai.com/skillhub</a><br>
                    注册并登录后，在控制台获取API Key。<br>
                    配置后可使用问财智能选股、行情查询、研报搜索等功能。
                  </div>
                  </n-gradient-text>
                </template>
              </n-tooltip>
            </n-form-item-gi>

            <n-form-item-gi :span="11" label="东财AI密钥：" path="emApiKey">
              <n-input type="password" placeholder="东方财富AI SaaS API Key" v-model:value="formValue.emApiKey" clearable show-password-on="click"/>
              <n-tooltip placement="top">
                <template #trigger>
                  <n-icon color="#0e7a0d" size="20">
                    <HelpCircleFilledIcon />
                  </n-icon>
                </template>
                <template #default>
                  <n-gradient-text :type="'warning'">
                  <div style="max-width: 400px;text-align: left">
                    获取方法：<br>
                    访问东方财富妙想AI平台获取API Key。
                    https://ai.eastmoney.com/mxClaw<br>
                    配置后可使用个股业绩点评功能。
                  </div>
                  </n-gradient-text>
                </template>
              </n-tooltip>
            </n-form-item-gi>

            <n-form-item-gi v-if="!isWebMode" :span="24" path="sponsorCode" class="sponsor-code-item">
              <div class="sponsor-code-box">
                <div class="sponsor-code-label">
                  <n-tag type="warning" size="small" :bordered="false" round>👑 VIP</n-tag>
                  <span class="sponsor-code-label-text">赞助码</span>
                  <span class="sponsor-code-label-sub">填写后激活 VIP 高级功能</span>
                </div>
                <n-input-group>
                  <n-input size="large" :show-count="true" class="sponsor-code-input"
                           placeholder="💎 联系作者QQ或微信获取赞助码，激活VIP功能"
                           v-model:value="formValue.sponsorCode">
                  </n-input>
                  <n-button size="large" type="warning" strong
                            @click="CheckSponsorCode(formValue.sponsorCode).then((res) => {message.warning(res.msg)})">立即验证
                  </n-button>
                  <n-popover trigger="hover" placement="top">
                    <template #trigger>
                      <n-icon color="#f0a020" size="22">
                        <HelpCircleFilledIcon />
                      </n-icon>
                    </template>
                    <n-gradient-text :type="'warning'">
                      <div style="max-width: 400px;text-align: left">
                        赞助码获取方式：<br>
                        联系作者获取赞助码，激活VIP功能<br>
                        享受更多高级功能和优先支持
                      </div>
                    </n-gradient-text>
                  </n-popover>
                </n-input-group>
              </div>
            </n-form-item-gi>

            <n-form-item-gi :span="11" label="提示词广场地址：" path="promptPlazaApiBase">
              <n-input type="text" placeholder="http://go-stock.sparkmemory.top:1918/api" v-model:value="formValue.promptPlazaApiBase" clearable/>
              <n-tooltip placement="top">
                <template #trigger>
                  <n-icon color="#0e7a0d" size="20">
                    <HelpCircleFilledIcon />
                  </n-icon>
                </template>
                <template #default>
                  <n-gradient-text :type="'warning'">
                  <div style="max-width: 400px;text-align: left">
                    提示词广场服务接口地址<br>
                    默认: http://go-stock.sparkmemory.top:1918/api<br>
                    如已部署提示词广场服务，可修改为实际地址
                  </div>
                  </n-gradient-text>
                </template>
              </n-tooltip>
            </n-form-item-gi>
          </n-grid>
        </n-card>

        <n-card :title="() => h(NTag, { type: 'primary', bordered: false }, () => '通知设置')" size="small">
          <n-grid :cols="24" :x-gap="24" style="text-align: left">
            <n-form-item-gi :span="3" label="钉钉推送：" path="dingPush.enable">
              <n-switch v-model:value="formValue.dingPush.enable"/>
            </n-form-item-gi>
            <n-form-item-gi :span="3" label="本地推送：" path="localPush.enable">
              <n-switch v-model:value="formValue.localPush.enable"/>
            </n-form-item-gi>
            <n-form-item-gi :span="3" label="弹幕功能：" path="enableDanmu">
              <n-switch v-model:value="formValue.enableDanmu"/>
            </n-form-item-gi>
            <n-form-item-gi :span="3" label="显示滚动快讯：" path="enableNews">
              <n-switch v-model:value="formValue.enableNews"/>
            </n-form-item-gi>
            <n-form-item-gi :span="3" label="市场资讯提醒：" path="enablePushNews">
              <n-switch v-model:value="formValue.enablePushNews"/>
            </n-form-item-gi>
            <n-form-item-gi v-if="formValue.enablePushNews" :span="4" label="只提醒红字或关注个股的新闻：" path="enableOnlyPushRedNews">
              <n-switch v-model:value="formValue.enableOnlyPushRedNews"/>
            </n-form-item-gi>

            <n-form-item-gi :span="22" v-if="formValue.dingPush.enable" label="钉钉机器人接口地址："
                            path="dingPush.dingRobot">
              <n-input placeholder="请输入钉钉机器人接口地址" v-model:value="formValue.dingPush.dingRobot"/>
              <n-button type="primary" @click="sendTestNotice">发送测试通知</n-button>
            </n-form-item-gi>

            <n-form-item-gi :span="3" label="飞书推送：" path="feishuPush.enable">
              <n-switch v-model:value="formValue.feishuPush.enable"/>
            </n-form-item-gi>
            <n-form-item-gi :span="22" v-if="formValue.feishuPush.enable" label="飞书机器人Webhook："
                            path="feishuPush.feishuRobot">
              <n-input placeholder="请输入飞书机器人Webhook地址" v-model:value="formValue.feishuPush.feishuRobot"/>
            </n-form-item-gi>
            <n-form-item-gi :span="22" v-if="formValue.feishuPush.enable" label="飞书机器人签名密钥："
                            path="feishuPush.feishuSecret">
              <n-input placeholder="请输入飞书机器人签名密钥（选填）" v-model:value="formValue.feishuPush.feishuSecret"/>
              <n-button type="primary" @click="sendFeishuTestNotice">发送飞书测试通知</n-button>
            </n-form-item-gi>

          </n-grid>
        </n-card>

        <n-card :title="() => h(NTag, { type: 'primary', bordered: false }, () => 'AI设置')" size="small">
          <n-grid :cols="24" :x-gap="24" style="text-align: left;">
            <n-form-item-gi :span="24" label="AI诊股：" path="openAI.enable">
              <n-switch v-model:value="formValue.openAI.enable"/>
            </n-form-item-gi>

            <n-form-item-gi :span="6" v-if="formValue.openAI.enable" label="Crawler Timeout(秒)"
                            title="资讯采集超时时间(秒)" path="openAI.crawlTimeOut">
              <n-input-number min="30" step="1" v-model:value="formValue.openAI.crawlTimeOut"/>
            </n-form-item-gi>
            <n-form-item-gi :span="4" v-if="formValue.openAI.enable" title="天数越多消耗tokens越多"
                            label="日K线数据(天)" path="openAI.kDays">
              <n-input-number min="30" step="1" max="60" v-model:value="formValue.openAI.kDays"/>
            </n-form-item-gi>
            <n-form-item-gi :span="2" label="爬虫http代理" path="httpProxyEnabled">
              <n-switch v-model:value="formValue.httpProxyEnabled"/>
            </n-form-item-gi>
            <n-form-item-gi :span="10" v-if="formValue.httpProxyEnabled" title="http代理地址"
                            label="http代理地址" path="httpProxy">
              <n-input type="text" placeholder="爬虫http代理地址" v-model:value="formValue.httpProxy" clearable/>
            </n-form-item-gi>
            <!-- [007/c0f43ef] 全局代理作用范围：overseas=仅国外站点与数据（推荐，国内数据源保持直连） -->
            <n-form-item-gi :span="10" v-if="formValue.httpProxyEnabled"
                            title="选择 http 代理的作用范围：「仅国外站点与数据」保持国内数据源直连（推荐）；「全局所有请求」所有请求均走代理"
                            label="代理作用范围" path="httpProxyMode">
              <n-radio-group v-model:value="formValue.httpProxyMode" name="httpProxyMode">
                <n-radio value="overseas">仅国外站点与数据</n-radio>
                <n-radio value="global">全局所有请求</n-radio>
              </n-radio-group>
            </n-form-item-gi>
            <n-form-item-gi :span="10" title="币圈数据源专用 HTTP 代理，与上方爬虫代理完全独立；覆盖币安合约、OKX、Bitget 及其公告/快讯接口；留空表示直连"
                            label="合约代理(可选)" path="binanceProxy">
              <n-input type="text" placeholder="如 http://127.0.0.1:7890，留空表示直连"
                        :value="binanceProxy" @update:value="onBinanceProxyInput" clearable/>
            </n-form-item-gi>

            <!-- [007/6562395+f455797] OKX（欧易）交易设置 -->
            <n-gi :span="24" v-if="formValue.openAI.enable">
              <n-divider title-placement="left">OKX（欧易）交易</n-divider>
            </n-gi>
            <template v-if="formValue.openAI.enable">
              <n-form-item-gi :span="8" label="模拟盘 API Key" path="okxApiKey">
                <n-input type="password" show-password-on="click" placeholder="OKX 模拟交易环境创建的 API Key" v-model:value="formValue.okxApiKey" clearable/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="模拟盘 Secret Key" path="okxSecretKey">
                <n-input type="password" show-password-on="click" placeholder="模拟盘 Secret Key" v-model:value="formValue.okxSecretKey" clearable/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="模拟盘口令" path="okxPassphrase">
                <n-input type="password" show-password-on="click" placeholder="模拟盘密钥的「口令」（非备注名）" v-model:value="formValue.okxPassphrase" clearable/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="实盘 API Key" path="okxLiveApiKey">
                <n-input type="password" show-password-on="click" placeholder="OKX 真实环境创建的 API Key" v-model:value="formValue.okxLiveApiKey" clearable/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="实盘 Secret Key" path="okxLiveSecretKey">
                <n-input type="password" show-password-on="click" placeholder="实盘 Secret Key" v-model:value="formValue.okxLiveSecretKey" clearable/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="实盘口令" path="okxLivePassphrase">
                <n-input type="password" show-password-on="click" placeholder="实盘密钥的「口令」（非备注名）" v-model:value="formValue.okxLivePassphrase" clearable/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="OKX 交易" path="okxEnableTrade">
                <n-switch v-model:value="formValue.okxEnableTrade"/>
                <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                  开启后 AI 助手可实际下单/撤单；关闭时仅允许查询
                </n-text>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="模拟盘自动下单" path="okxSignalTradeEnable">
                <n-switch v-model:value="formValue.okxSignalTradeEnable"/>
                <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                  加密买卖点信号自动在 OKX 模拟盘下单，不动真实资金
                </n-text>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="模拟盘每笔金额(USDT)" path="okxSignalTradeUSDT">
                <n-input type="text" placeholder="模拟盘每笔信号投入金额，默认 100" v-model:value="formValue.okxSignalTradeUSDT" :disabled="formValue.okxSignalTradeAmountMode === 'ratio'" clearable/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="信号下单品种" path="okxSignalTradeInstType">
                <n-select v-model:value="formValue.okxSignalTradeInstType"
                          :options="[{label:'永续合约 SWAP',value:'SWAP'},{label:'币币现货 SPOT',value:'SPOT'}]"/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="信号下单杠杆" path="okxSignalTradeLever">
                <n-select v-model:value="formValue.okxSignalTradeLever" :options="okxLeverOptions"/>
                <n-text type="warning" depth="3" style="margin-left: 12px; font-size: 12px;">
                  仅 SWAP；每笔下单前按环境设置合约杠杆（全仓），设置失败该笔拒绝下单
                </n-text>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="实盘自动下单" path="okxSignalTradeLive">
                <n-switch v-model:value="formValue.okxSignalTradeLive">
                  <template #checked>启用</template>
                  <template #unchecked>暂停</template>
                </n-switch>
                <n-tag type="error" size="small" style="margin-left: 12px;">实盘 · 真实资金！</n-tag>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="实盘每笔金额(USDT)" path="okxSignalTradeLiveUSDT">
                <n-input type="text" placeholder="实盘每笔信号投入金额，默认 10" v-model:value="formValue.okxSignalTradeLiveUSDT" :disabled="formValue.okxSignalTradeAmountMode === 'ratio'" clearable/>
                <n-text type="error" depth="3" style="margin-left: 12px; font-size: 12px;">
                  实盘金额独立于模拟盘，请按风险承受能力设置
                </n-text>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="下单金额模式" path="okxSignalTradeAmountMode">
                <n-select v-model:value="formValue.okxSignalTradeAmountMode" :options="[{label:'固定金额',value:'fixed'},{label:'按可用余额比例（仓位）',value:'ratio'}]"/>
                <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                  比例模式：每笔信号按下单环境账户可用 USDT × 仓位比例开仓，模拟盘/实盘共用
                </n-text>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="仓位比例" path="okxSignalTradeAmountRatio">
                <n-select v-model:value="formValue.okxSignalTradeAmountRatio" :options="okxAmountRatioOptions" :disabled="formValue.okxSignalTradeAmountMode !== 'ratio'"/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="防爆仓·最大持仓数" path="okxRiskMaxPositions">
                <n-input type="text" placeholder="默认 3；0=不限制" v-model:value="formValue.okxRiskMaxPositions" clearable/>
                <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                  当前环境账户持仓笔数达到上限后拒绝开新仓（不影响已持仓的平仓），防止连续信号累加成超额仓位
                </n-text>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="防爆仓·最大保证金率(%)" path="okxRiskMaxMarginRatio">
                <n-input type="text" placeholder="默认 30；0=不限制" v-model:value="formValue.okxRiskMaxMarginRatio" clearable/>
                <n-text type="warning" depth="3" style="margin-left: 12px; font-size: 12px;">
                  维持保证金÷总权益 ≥ 阈值时拒绝开新仓——这是距强平最直接的指标，建议配合止盈止损使用
                </n-text>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="交易周期过滤" path="okxSignalTradeKlts">
                <n-select v-model:value="okxSignalTradeKltsSel" multiple :options="okxKltOptions" placeholder="留空=全部周期" clearable/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="最低信号分数" path="okxSignalTradeMinScore">
                <n-input type="text" placeholder="如 4，留空=不限制" v-model:value="formValue.okxSignalTradeMinScore" clearable/>
                <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                  买卖点共振路数（满分 9）低于该值不自动下单；建议 4（标准档），过滤低分噪音信号
                </n-text>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="止盈止损冷却(分钟)" path="okxSignalTradeCooldownMin">
                <n-input type="text" placeholder="留空=默认 30 分钟" v-model:value="formValue.okxSignalTradeCooldownMin" clearable/>
                <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                  止盈/止损平仓后，同合约同方向冷却期内不再自动开仓（防震荡市反复止损）；0=关闭
                </n-text>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="股票信号下单" path="okxSignalStockEnable">
                <n-switch v-model:value="formValue.okxSignalStockEnable"/>
                <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                  股票永续信号（美股/港股）独立开关，与加密信号互不影响；当前暂仅模拟盘
                </n-text>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="股票每笔金额(USDT)" path="okxSignalStockUSDT">
                <n-input type="text" placeholder="股票信号每笔投入金额，默认 100" v-model:value="formValue.okxSignalStockUSDT" clearable/>
                <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                  OKX 模拟盘未上架股票永续，实盘 xStocks 类产品可用
                </n-text>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="止盈止损(持仓)" path="okxTpSlEnable">
                <n-switch v-model:value="formValue.okxTpSlEnable"/>
                <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                  每 10 秒检查全部持仓，达到平仓线立即市价平仓
                </n-text>
              </n-form-item-gi>
              <n-form-item-gi :span="8" label="平仓线模式" path="okxTpSlMode">
                <n-select v-model:value="formValue.okxTpSlMode"
                          :options="[{label:'固定收益率(%)',value:'fixed'},{label:'波动率自适应',value:'vol'}]"/>
              </n-form-item-gi>
              <template v-if="formValue.okxTpSlMode==='fixed'">
                <n-form-item-gi :span="8" label="止盈收益率(%)" path="okxTakeProfitPct">
                  <n-input type="text" placeholder="如 5 表示 +5% 平仓；0/留空=不启用" v-model:value="formValue.okxTakeProfitPct" clearable/>
                </n-form-item-gi>
                <n-form-item-gi :span="8" label="止损收益率(%)" path="okxStopLossPct">
                  <n-input type="text" placeholder="如 3 表示 -3% 平仓；0/留空=不启用" v-model:value="formValue.okxStopLossPct" clearable/>
                </n-form-item-gi>
              </template>
              <template v-else>
                <n-form-item-gi :span="8" label="波动率参考周期" path="okxTpSlVolKlt">
                  <n-select v-model:value="formValue.okxTpSlVolKlt"
                            :options="[{label:'15 分钟',value:'15m'},{label:'1 小时',value:'1H'},{label:'日K',value:'1D'}]"/>
                </n-form-item-gi>
                <n-form-item-gi :span="8" label="波动率回看根数" path="okxTpSlVolBars">
                  <n-input type="text" placeholder="默认 20" v-model:value="formValue.okxTpSlVolBars" clearable/>
                </n-form-item-gi>
                <n-form-item-gi :span="8" label="止盈倍数" path="okxTpSlTpMult">
                  <n-input type="text" placeholder="止盈线 = 平均振幅 × 倍数，默认 1.5" v-model:value="formValue.okxTpSlTpMult" clearable/>
                </n-form-item-gi>
                <n-form-item-gi :span="8" label="止损倍数" path="okxTpSlSlMult">
                  <n-input type="text" placeholder="止损线 = 平均振幅 × 倍数，默认 1" v-model:value="formValue.okxTpSlSlMult" clearable/>
                </n-form-item-gi>
              </template>
            </template>


            <n-gi :span="24" v-if="formValue.openAI.enable">
              <n-divider title-placement="left">默认提示词设置</n-divider>
            </n-gi>
            <n-form-item-gi :span="12" v-if="formValue.openAI.enable" label="默认系统提示词" path="openAI.prompt">
              <n-input v-model:value="formValue.openAI.prompt" type="textarea" :show-count="true"
                       placeholder="请输入系统提示词" :autosize="{ minRows: 4, maxRows: 8 }"/>
            </n-form-item-gi>
            <n-form-item-gi :span="12" v-if="formValue.openAI.enable" label="默认个股分析提示词"
                            path="openAI.questionTemplate">
              <n-input v-model:value="formValue.openAI.questionTemplate" type="textarea" :show-count="true"
                       placeholder="请输入个股分析提示词:例如{{stockName}}[{{stockCode}}]分析和总结"
                       :autosize="{ minRows: 4, maxRows: 8 }"/>
            </n-form-item-gi>

            <n-gi :span="24" v-if="formValue.openAI.enable">
              <n-divider title-placement="left">AI模型服务配置</n-divider>
            </n-gi>
            <n-gi :span="24" v-if="formValue.openAI.enable">
              <n-space vertical>
                <n-collapse v-model:expanded-names="aiConfigExpandedNames" accordion>
                  <n-collapse-item v-for="(aiConfig, index) in formValue.openAI.aiConfigs" :key="index" :name="String(index)">
                    <template #header>
                      <n-flex justify="space-between" align="center" style="width: 100%;">
                        <n-text>{{ aiConfig.name || `AI 配置 #${index + 1}` }}</n-text>
                        <n-text depth="3" style="font-size: 12px;">{{ aiConfig.modelName || '未选择模型' }}</n-text>
                      </n-flex>
                    </template>
                    <template #header-extra>
                      <n-button type="error" size="tiny" ghost @click.stop="removeAiConfig(index)" style="margin-right: 8px;">删除</n-button>
                    </template>
                    <n-grid :cols="24" :x-gap="24">
                      <n-form-item-gi :span="24" hidden label="配置ID" :path="`openAI.aiConfigs[${index}].ID`">
                        <n-input type="text" placeholder="配置ID" v-model:value="aiConfig.ID" clearable/>
                      </n-form-item-gi>
                      <n-form-item-gi :span="12" label="配置名称" :path="`openAI.aiConfigs[${index}].name`">
                        <n-input type="text" placeholder="配置名称" v-model:value="aiConfig.name" clearable/>
                      </n-form-item-gi>
                      <n-form-item-gi :span="12" label="接口地址" :path="`openAI.aiConfigs[${index}].baseUrl`">
                        <n-select
                          v-model:value="aiConfig.baseUrl"
                          :options="aiPlatformOptions"
                          filterable
                          tag
                          clearable
                          placeholder="选择或输入AI接口地址"
                          @update:value="(val) => onBaseUrlChange(aiConfig, val)"
                        />
                      </n-form-item-gi>
                      <n-form-item-gi :span="12" label="令牌(apiKey)" :path="`openAI.aiConfigs[${index}].apiKey`">
                        <n-input type="password" placeholder="apiKey" v-model:value="aiConfig.apiKey" clearable
                                 show-password-on="click"/>
                      </n-form-item-gi>
                      <n-form-item-gi :span="8" label="模型名称" :path="`openAI.aiConfigs[${index}].modelName`">
                        <n-select
                          v-model:value="aiConfig.modelName"
                          :options="aiConfig._modelOptions || []"
                          filterable
                          tag
                          :loading="aiConfig._loadingModels"
                          placeholder="点击获取模型列表或手动输入"
                          @click="fetchAiModels(aiConfig)"
                          @update:value="(val) => onModelNameChange(aiConfig, val)"
                        />
                      </n-form-item-gi>
                      <n-form-item-gi :span="5" label="Temperature" :path="`openAI.aiConfigs[${index}].temperature`">
                        <n-input-number placeholder="temperature" v-model:value="aiConfig.temperature" :step="0.1"/>
                      </n-form-item-gi>
                      <n-form-item-gi :span="5" label="MaxTokens" :path="`openAI.aiConfigs[${index}].maxTokens`">
                        <n-input-number placeholder="maxTokens" v-model:value="aiConfig.maxTokens"/>
                      </n-form-item-gi>
                      <n-form-item-gi :span="5" label="Timeout(秒)" :path="`openAI.aiConfigs[${index}].timeOut`">
                        <n-input-number min="60" step="1" placeholder="超时(秒)" v-model:value="aiConfig.timeOut"/>
                      </n-form-item-gi>
                      <n-form-item-gi :span="12" label="深度思考">
                        <n-switch v-model:value="aiConfig.thinking"/>
                        <n-tooltip placement="top">
                          <template #trigger>
                            <n-icon color="#0e7a0d" size="20" style="margin-left: 8px;">
                              <HelpCircleFilledIcon />
                            </n-icon>
                          </template>
                          <template #default>
                            <n-gradient-text :type="'warning'">
                            <div style="max-width: 400px;text-align: left">
                              启用深度思考模式：<br>
                              适用于 DeepSeek-Reasoner、MiMo-V2.5-Pro 等支持推理的模型。<br>
                              如使用普通模型请关闭此选项
                            </div>
                            </n-gradient-text>
                          </template>
                        </n-tooltip>
                      </n-form-item-gi>
                      <n-form-item-gi :span="12" label="http代理" :path="`openAI.aiConfigs[${index}].httpProxyEnabled`">
                        <n-switch v-model:value="aiConfig.httpProxyEnabled"/>
                      </n-form-item-gi>
                      <n-form-item-gi :span="12" v-if="aiConfig.httpProxyEnabled" title="http代理地址" :path="`openAI.aiConfigs[${index}].httpProxy`">
                        <n-input type="text" placeholder="http代理地址" v-model:value="aiConfig.httpProxy" clearable/>
                      </n-form-item-gi>
                    </n-grid>
                  </n-collapse-item>
                </n-collapse>
                <n-button type="primary" dashed @click="addAiConfig" style="width: 100%;">+ 添加AI配置</n-button>
              </n-space>
            </n-gi>

            <n-gi :span="24">
              <n-divider/>
            </n-gi>

            <n-gi :span="24">
              <n-space vertical>
                <n-space justify="center">
<!--                  <n-button type="warning" @click="managePrompts">管理提示词模板</n-button>-->
                  <n-button type="primary" strong @click="saveConfig">保存设置</n-button>
                  <n-button type="info" @click="exportConfig">导出配置</n-button>
                  <n-button type="error" @click="importConfig">导入配置</n-button>
                </n-space>

<!--                <n-flex justify="start" style="margin-top: 10px" v-if="promptTemplates.length > 0">-->
<!--                  <n-tag :bordered="false" type="warning">提示词模板:</n-tag>-->
<!--                  <n-tag size="medium" secondary v-for="prompt in promptTemplates" closable-->
<!--                         @close="deletePrompt(prompt.ID)" @click="editPrompt(prompt)" :title="prompt.content"-->
<!--                         :type="prompt.type === '模型系统Prompt' ? 'success' : 'info'" :bordered="false">{{-->
<!--                      prompt.name-->
<!--                    }}-->
<!--                  </n-tag>-->
<!--                </n-flex>-->
              </n-space>
            </n-gi>

          </n-grid>
        </n-card>
      </n-space>
    </n-form>
  </n-flex>

  <n-modal v-model:show="showManagePromptsModal" closable :mask-closable="false">
    <n-card style="width: 800px; height: 600px; text-align: left" :bordered="false"
            :title="(formPrompt.ID > 0 ? '修改' : '添加') + '提示词'" size="huge" role="dialog" aria-modal="true">
      <n-form ref="formPromptRef" :label-placement="'left'" :label-align="'left'">
        <n-form-item label="名称">
          <n-input v-model:value="formPrompt.Name" placeholder="请输入提示词名称"/>
        </n-form-item>
        <n-form-item label="类型">
          <n-select v-model:value="formPrompt.Type" :options="promptTypeOptions" placeholder="请选择提示词类型"/>
        </n-form-item>
        <n-form-item label="内容">
          <n-input v-model:value="formPrompt.Content" type="textarea" :show-count="true" placeholder="请输入prompt"
                   :autosize="{ minRows: 12, maxRows: 12, }"/>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-flex justify="end">
          <n-button type="primary" @click="savePrompt">保存</n-button>
          <n-button type="warning" @click="showManagePromptsModal = false">取消</n-button>
        </n-flex>
      </template>
    </n-card>
  </n-modal>
</template>

<style scoped>
.cardHeaderClass {
  font-size: 16px;
  font-weight: bold;
  color: red;
}

/* 赞助码醒目样式 */
.sponsor-code-item :deep(.n-form-item-blank) {
  display: block;
  width: 100%;
}

.sponsor-code-box {
  width: 100%;
  padding: 14px 16px;
  border-radius: 10px;
  background: linear-gradient(135deg, rgba(240, 160, 32, 0.10), rgba(240, 160, 32, 0.03));
  border: 1.5px dashed rgba(240, 160, 32, 0.55);
  box-shadow: 0 2px 10px rgba(240, 160, 32, 0.12);
  transition: border-color 0.3s ease, box-shadow 0.3s ease;
}

.sponsor-code-box:hover,
.sponsor-code-box:focus-within {
  border-color: #f0a020;
  border-style: solid;
  box-shadow: 0 2px 14px rgba(240, 160, 32, 0.35);
}

.sponsor-code-label {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.sponsor-code-label-text {
  font-size: 16px;
  font-weight: bold;
  color: #f0a020;
}

.sponsor-code-label-sub {
  font-size: 12px;
  color: rgba(240, 160, 32, 0.75);
}

.sponsor-code-input :deep(.n-input__input-el) {
  font-weight: 600;
  letter-spacing: 1px;
}
</style>