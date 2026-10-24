<script setup>
import {h, defineComponent, onBeforeUnmount, onMounted, ref, computed} from "vue";
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

// HelpTip 表单项 label 的问号图标 + 悬停提示（lines 支持字符串或多行数组，每行一个 div）（上游 2872d65）
const HelpTip = defineComponent({
  props: { lines: { type: [String, Array], required: true } },
  setup(props) {
    return () => h(NTooltip, { trigger: 'hover', placement: 'top-start' }, {
      trigger: () => h(NIcon, {
        size: 14,
        color: '#8a8a8a',
        style: 'cursor:help;vertical-align:-2px;margin-left:3px;'
      }, () => h(HelpCircleFilledIcon)),
      default: () => h('div', { style: 'max-width:380px;text-align:left;line-height:1.7;font-size:12px;' },
        (Array.isArray(props.lines) ? props.lines : [props.lines]).map(l => h('div', l)))
    })
  }
})

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
  deepSeekSearchApiKey: '',
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
  okxSignalTradeLiveUSDT:"10",
  okxSignalTradeAmountMode:"fixed",
  okxSignalTradeAmountRatio:"0.25",
  okxSignalTradeMinScore:"",
  okxSignalTradeCooldownMin:"",
  okxSignalMinTpPct:"",
  okxSignalMaxTpPct:"",
  okxRiskMaxPositions:"3",
  okxRiskMaxMarginRatio:"30",
  okxSignalTradeInstType:"SWAP",
  okxSignalTradeLever:"",
  okxSignalTradeKlts:"",
  okxSignalTradeDirMode:"both",
  okxSignalTradeRegime:false,
  okxSignalTradeMaxPerDay:"",
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
    formValue.value.deepSeekSearchApiKey = res.deepSeekSearchApiKey || ''
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
    formValue.value.okxSignalTradeLiveUSDT=res.okxSignalTradeLiveUSDT || "10";
    formValue.value.okxSignalTradeAmountMode=res.okxSignalTradeAmountMode || "fixed";
    formValue.value.okxSignalTradeAmountRatio=res.okxSignalTradeAmountRatio || "0.25";
    formValue.value.okxRiskMaxPositions=res.okxRiskMaxPositions ?? "3";
    formValue.value.okxRiskMaxMarginRatio=res.okxRiskMaxMarginRatio ?? "30";
    formValue.value.okxSignalTradeInstType=res.okxSignalTradeInstType || "SWAP";
    formValue.value.okxSignalTradeLever=res.okxSignalTradeLever || "";
    formValue.value.okxSignalTradeKlts=res.okxSignalTradeKlts || "";
    formValue.value.okxSignalTradeMinScore=res.okxSignalTradeMinScore || "";
    formValue.value.okxSignalTradeCooldownMin=res.okxSignalTradeCooldownMin || "";
    formValue.value.okxSignalMinTpPct=res.okxSignalMinTpPct || "";
    formValue.value.okxSignalMaxTpPct=res.okxSignalMaxTpPct || "";
    formValue.value.okxSignalTradeDirMode=res.okxSignalTradeDirMode || "both";
    formValue.value.okxSignalTradeRegime=res.okxSignalTradeRegime === true;
    formValue.value.okxSignalTradeMaxPerDay=res.okxSignalTradeMaxPerDay || "";
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

/**
 * 信号自动交易参数模板。（上游 02a3691/2872d65）
 * 每个模板是一组「交易周期 / 信号分数门槛 / 开仓方向 / 顺大势门控 / 单日开仓上限 / 冷却 / 止盈止损」的预设，
 * 按自己的交易风格选一组，点「应用模板」填入后仍可对单个参数微调。
 */
const SIGNAL_PRESETS = {
  h1_flat: {
    label: '① 1h 双向 + 顺大势 + 冷却 120 分钟',
    klts: '60', ms: '6', dir: 'both', regime: true, mpd: '2', cd: '120', tpsl: false,
    desc: '交易最稀疏的一组：只做 60 分钟信号、只做高共振（≥6 路）、顺大势过滤、单日最多 2 笔、冷却 120 分钟，适合想尽量少动手的人。',
  },
  h1_long: {
    label: '② 1h 趋势跟随（只做多 + 顺大势）',
    klts: '60', ms: '6', dir: 'long', regime: true, mpd: '2', cd: '120', tpsl: false,
    desc: '只吃上涨段：只做 60 分钟多头信号，空头行情里会长期不出手（既是特点也是取舍）。',
  },
  h1_guard: {
    label: '③ 1h 双向 + 顺大势 + 只止损保护',
    klts: '60', ms: '6', dir: 'both', regime: true, mpd: '2', cd: '120',
    tpsl: true, tpMode: 'vol', volBars: '14', slMult: '8', tpMult: '0',
    desc: '在①的基础上开启止盈止损：用较宽的波动率止损（8 倍振幅）限制单笔亏损，不设固定止盈（账户浮盈 ≥20% 后仍有移动止盈兜底）。',
  },
  m15_sel: {
    label: '④ 15m 精选（7 路共振 + 只多 + 顺大势）',
    klts: '15', ms: '7', dir: 'long', regime: true, mpd: '2', cd: '60', tpsl: false,
    desc: '15 分钟周期的稳健写法：门槛提到 7 路共振、只做多、冷却 60 分钟，出手更少更挑。',
  },
  m5_fast: {
    label: '⑤ 5m 高频（仅适合想练盘感的人）',
    klts: '5', ms: '6', dir: 'long', regime: true, mpd: '2', cd: '60', tpsl: false,
    desc: '5 分钟周期、交易最频繁；信号密集、手续费累积快，仅建议想看盘感的人使用。',
  },
  legacy15: {
    label: '⑥ 15m 双向（上一版默认）',
    klts: '15', ms: '6', dir: 'both', regime: true, mpd: '2', cd: '60', tpsl: false,
    desc: '保留备查的一组：15 分钟双向、6 路共振，比 ④ 门槛低、出手更多。',
  },
  stock15: {
    label: '⑦ 股票类合约（SOXL / NVDA）15m 双向',
    klts: '15', ms: '6', dir: 'both', regime: true, mpd: '0', cd: '60', tpsl: false,
    desc: '面向股票类永续：必须配合「顺大势（标的自身 1H）」使用，用 BTC 当趋势代理会反向；仅对股票类标的适用。',
  },
}
const presetKey = ref('h1_flat')
const signalPresetOptions = Object.entries(SIGNAL_PRESETS).map(([k, v]) => ({ label: v.label, value: k }))
const curPreset = computed(() => SIGNAL_PRESETS[presetKey.value] || null)

/** 一键应用参数模板：只改自动交易相关字段，不碰密钥与下单金额 */
function applySignalPreset(key) {
  if (key === 'off') {
    formValue.value.okxSignalTradeKlts = ''
    message.info('已清除周期过滤（全部周期都会自动下单），记得保存')
    return
  }
  const p = SIGNAL_PRESETS[key]
  if (!p) return
  formValue.value.okxSignalTradeKlts = p.klts
  formValue.value.okxSignalTradeMinScore = p.ms
  formValue.value.okxSignalTradeDirMode = p.dir
  formValue.value.okxSignalTradeRegime = p.regime
  formValue.value.okxSignalTradeMaxPerDay = p.mpd
  formValue.value.okxSignalTradeCooldownMin = p.cd
  formValue.value.okxSignalTradeLever = '1'
  formValue.value.okxTpSlEnable = !!p.tpsl
  if (p.tpsl) {
    formValue.value.okxTpSlMode = p.tpMode || 'vol'
    // TPSL 参考周期用 OKX bar 口径（1H 而非 60m）：非法的 bar 会导致振幅取 0、vol 模式监控静默跳过
    formValue.value.okxTpSlVolKlt = p.klts === '60' ? '1H' : '15m'
    formValue.value.okxTpSlVolBars = p.volBars || '14'
    formValue.value.okxTpSlSlMult = p.slMult || '8'
    formValue.value.okxTpSlTpMult = p.tpMult || '1.5'
  }
  message.success(`已填入「${p.label}」，点击保存生效`)
}

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
    deepSeekSearchApiKey: formValue.value.deepSeekSearchApiKey,
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
    okxSignalMinTpPct:formValue.value.okxSignalMinTpPct || "",
    okxSignalMaxTpPct:formValue.value.okxSignalMaxTpPct || "",
    okxSignalTradeDirMode:formValue.value.okxSignalTradeDirMode || "both",
    okxSignalTradeRegime:formValue.value.okxSignalTradeRegime === true,
    okxSignalTradeMaxPerDay:formValue.value.okxSignalTradeMaxPerDay || "",
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
      formValue.value.deepSeekSearchApiKey = config.deepSeekSearchApiKey || ''
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
      formValue.value.okxSignalTradeLiveUSDT=config.okxSignalTradeLiveUSDT || "10"
      formValue.value.okxSignalTradeAmountMode=config.okxSignalTradeAmountMode || "fixed"
      formValue.value.okxSignalTradeAmountRatio=config.okxSignalTradeAmountRatio || "0.25"
      formValue.value.okxRiskMaxPositions=config.okxRiskMaxPositions ?? "3"
      formValue.value.okxRiskMaxMarginRatio=config.okxRiskMaxMarginRatio ?? "30"
      formValue.value.okxSignalTradeInstType=config.okxSignalTradeInstType || "SWAP"
      formValue.value.okxSignalTradeLever=config.okxSignalTradeLever || ""
      formValue.value.okxSignalTradeKlts=config.okxSignalTradeKlts || ""
      formValue.value.okxSignalTradeMinScore=config.okxSignalTradeMinScore || ""
      formValue.value.okxSignalTradeCooldownMin=config.okxSignalTradeCooldownMin || ""
      formValue.value.okxSignalMinTpPct=config.okxSignalMinTpPct || ""
      formValue.value.okxSignalMaxTpPct=config.okxSignalMaxTpPct || ""
      formValue.value.okxSignalTradeDirMode=config.okxSignalTradeDirMode || "both"
      formValue.value.okxSignalTradeRegime=config.okxSignalTradeRegime === true
      formValue.value.okxSignalTradeMaxPerDay=config.okxSignalTradeMaxPerDay || ""
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

            <n-form-item-gi :span="11" label="DeepSeek搜索密钥：" path="deepSeekSearchApiKey">
              <n-input type="password" placeholder="DeepSeek API Key（api.deepseek.com）" v-model:value="formValue.deepSeekSearchApiKey" clearable show-password-on="click"/>
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
                    访问 DeepSeek 开放平台创建 API Key：<br>
                    <a href="https://platform.deepseek.com" target="_blank" style="color: #63e2b7">https://platform.deepseek.com</a><br>
                    配置后 AI 助手可联网搜索实时资讯、抓取网页正文（WebSearch/WebFetch 工具）。<br>
                    按成功请求次数计费（无结果不计费），仅 AI 需要时才会调用。
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
              <n-input type="text" placeholder="留空=使用后端默认地址（由后端配置下发）" v-model:value="formValue.promptPlazaApiBase" clearable/>
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
                    留空 = 使用后端默认地址（默认域名 + 回退列表由后端配置统一下发，不再写死）<br>
                    如已自建提示词广场服务，可修改为实际地址
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
            <!-- [008/2872d65+02a3691] OKX 交易设置：上游 HelpTip 化 + 参数模板 + 方向模式/顺大势/单日限笔数 -->
            <template v-if="formValue.openAI.enable">
            <n-gi :span="24" v-if="formValue.openAI.enable">
              <n-space vertical :size="6" style="padding: 0 0 14px 0;">
                <n-space align="center" :size="8">
                  <n-text depth="3" style="font-size: 12px;">参数模板：</n-text>
                  <n-select size="small" style="width: 320px" v-model:value="presetKey" :options="signalPresetOptions"/>
                  <n-button size="small" type="primary" secondary @click="applySignalPreset(presetKey)">应用模板</n-button>
                  <n-button size="small" tertiary @click="applySignalPreset('off')">关闭周期过滤</n-button>
                </n-space>
                <n-text v-if="curPreset" depth="3" style="font-size: 12px; line-height: 1.6;">
                  {{ curPreset.desc }}
                </n-text>
              </n-space>
            </n-gi>
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
            <n-form-item-gi :span="8" path="okxEnableTrade">
              <template #label>OKX 交易<HelpTip :lines="['AI 助手交易权限总开关：开启后 AI 助手才可实际下单/撤单（如对话中说「用 OKX 市价买入 0.01 BTC」），关闭时仅可查询行情、持仓、账户等只读操作。', '与下方「模拟盘/实盘自动下单」无关：信号自动交易只受各自开关控制，不受此开关限制。']"/></template>
              <n-switch v-model:value="formValue.okxEnableTrade"/>
              <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                开启后 AI 助手可实际下单/撤单；关闭时仅允许查询
              </n-text>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeEnable">
              <template #label>模拟盘自动下单<HelpTip :lines="['信号→自动交易链路：信号监控「OKX 模拟盘池」上报买卖点信号 → 命中加密代码（ok:/bn:/bt: 前缀）时按市价自动在 OKX 模拟盘下单，不动真实资金。', '防重复开仓四道闸：① 同一信号同一分钟去重；② 同向已有持仓不加仓；③ 止盈/止损平仓后同向冷却；④ 合约不可交易（51001）熔断 6 小时。', '下单结果落流水：success=已成交 / failed=失败(含原因) / skipped=正常跳过(如金额低于最小下单量)。']"/></template>
              <n-switch v-model:value="formValue.okxSignalTradeEnable"/>
              <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                加密买卖点信号自动在 OKX 模拟盘下单，不动真实资金
              </n-text>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeUSDT">
              <template #label>模拟盘每笔金额(USDT)<HelpTip :lines="['名义价值口径（不乘杠杆）：SWAP 按张数换算（张数 = 金额 ÷ 现价 ÷ 每张面值），SPOT 按基础币数量换算（数量 = 金额 ÷ 现价）；20x 杠杆下 SWAP 实际占用保证金 ≈ 名义 ÷ 20，杠杆只影响保证金占用、不影响仓位大小。', '金额低于该合约最小下单量时记 skipped 跳过、不视为失败（每合约半小时最多记一条流水）。', '仅「固定金额」模式生效；比例模式下此项禁用，按可用余额 × 仓位比例开仓。']"/></template>
              <n-input type="text" placeholder="模拟盘每笔信号投入金额（名义价值，不乘杠杆），默认 100" v-model:value="formValue.okxSignalTradeUSDT" :disabled="formValue.okxSignalTradeAmountMode === 'ratio'" clearable/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeInstType">
              <template #label>信号下单品种<HelpTip :lines="['永续合约 SWAP：按合约面值换算张数，支持杠杆与交易所侧附带止盈止损。', '币币现货 SPOT：按币种数量下单，无杠杆，最小下单量以交易所规则为准。']"/></template>
              <n-select v-model:value="formValue.okxSignalTradeInstType"
                        :options="[{label:'永续合约 SWAP',value:'SWAP'},{label:'币币现货 SPOT',value:'SPOT'}]"/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeLever">
              <template #label>信号下单杠杆<HelpTip :lines="['仅 SWAP 有效：下单前按环境（模拟盘/实盘）设置该合约杠杆（全仓模式），设置失败该笔拒绝下单；超过合约最大杠杆时自动收敛到上限。', '杠杆只影响保证金占用（保证金 = 名义 ÷ 杠杆），不影响仓位大小：数量始终按名义价值金额换算。', '留空 = 不显式设置、沿用账户当前杠杆；此时交易所侧附带止盈止损可能不下发，仅由软件侧监控兜底，建议显式设置。']"/></template>
              <n-select v-model:value="formValue.okxSignalTradeLever" :options="okxLeverOptions"/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeLive">
              <template #label>实盘自动下单<HelpTip :lines="['信号监控「OKX 实盘池」上报的买卖点信号自动在 OKX 实盘下单，动用真实资金，请充分评估风险后开启。', '与「模拟盘自动下单」相互独立：同一信号两个池都有时，两个环境各下一笔、各落一条流水。', '模拟盘与实盘是两个独立账户（密钥各自独立），反向平仓、止盈止损也按环境各查各的持仓。']"/></template>
              <n-switch v-model:value="formValue.okxSignalTradeLive">
                <template #checked>启用</template>
                <template #unchecked>暂停</template>
              </n-switch>
              <n-tag type="error" size="small" style="margin-left: 12px;">实盘 · 真实资金！</n-tag>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeLiveUSDT">
              <template #label>实盘每笔金额(USDT)<HelpTip :lines="['名义价值口径（不乘杠杆），与模拟盘金额完全独立，请按风险承受能力设置。', '比例模式下此项禁用：实盘按实盘账户可用余额 × 仓位比例开仓，模拟盘按模拟盘余额计算，互不干扰。', '金额低于合约最小下单量时记 skipped 跳过（每合约半小时最多记一条，不刷流水）。']"/></template>
              <n-input type="text" placeholder="实盘每笔信号投入金额（名义价值，不乘杠杆），默认 10" v-model:value="formValue.okxSignalTradeLiveUSDT" :disabled="formValue.okxSignalTradeAmountMode === 'ratio'" clearable/>
              <n-text type="error" depth="3" style="margin-left: 12px; font-size: 12px;">
                实盘金额独立于模拟盘，请按风险承受能力设置
              </n-text>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeAmountMode">
              <template #label>下单金额模式<HelpTip :lines="['固定金额（默认）：每笔信号按上方固定金额开仓。', '按可用余额比例（仓位）：每笔按下单环境账户可用 USDT × 仓位比例作为名义价值开仓（不乘杠杆）；模拟盘/实盘共用该模式，各自按各自余额计算。', '比例模式下每笔金额随余额实时浮动：盈利后金额变大、亏损后变小；余额查询失败或为 0 时该笔落失败流水。']"/></template>
              <n-select v-model:value="formValue.okxSignalTradeAmountMode" :options="[{label:'固定金额',value:'fixed'},{label:'按可用余额比例（仓位）',value:'ratio'}]"/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeAmountRatio">
              <template #label>仓位比例<HelpTip :lines="['仅比例模式生效：每笔名义价值 = 下单环境账户可用 USDT × 该比例。', '0.25 = 1/4 仓、0.3333 = 1/3 仓、0.5 = 半仓、1 = 全仓；名义价值口径不随杠杆放大。', '注意：全仓 + 高杠杆下连续信号会快速叠加仓位（20x 下每笔保证金只占名义 1/20，余额几乎不消耗），建议配合「最大持仓数」与「最大保证金率」使用。']"/></template>
              <n-select v-model:value="formValue.okxSignalTradeAmountRatio" :options="okxAmountRatioOptions" :disabled="formValue.okxSignalTradeAmountMode !== 'ratio'"/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxRiskMaxPositions">
              <template #label>防爆仓·最大持仓数<HelpTip :lines="['当前环境账户非零持仓笔数 ≥ 上限时拒绝开新仓（平仓、止盈止损不受影响）。', '防止连续信号在几分钟内累加成超额仓位；默认 3，0 = 不限制。', '模拟盘与实盘各自计数、互不干扰。']"/></template>
              <n-input type="text" placeholder="默认 3；0=不限制" v-model:value="formValue.okxRiskMaxPositions" clearable/>
              <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                当前环境账户持仓笔数达到上限后拒绝开新仓（不影响的平仓），防止连续信号累加成超额仓位
              </n-text>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxRiskMaxMarginRatio">
              <template #label>防爆仓·最大保证金率(%)<HelpTip :lines="['维持保证金合计 ÷ 总权益 ≥ 该阈值(%)时拒绝开新仓——这是距离强制平仓最直接的指标，越接近 100% 越危险。', '默认 30，0 = 不限制；建议配合止盈止损一起使用。', '模拟盘与实盘各自按各自账户计算。']"/></template>
              <n-input type="text" placeholder="默认 30；0=不限制" v-model:value="formValue.okxRiskMaxMarginRatio" clearable/>
              <n-text type="warning" depth="3" style="margin-left: 12px; font-size: 12px;">
                维持保证金÷总权益 ≥ 阈值时拒绝开新仓——这是距强平最直接的指标，建议配合止盈止损使用
              </n-text>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeKlts">
              <template #label>交易周期过滤<HelpTip :lines="['只允许勾选周期的买卖点信号自动下单：监控池信号按 K 线周期（1m/5m/15m/30m/60m 等）分级，这里决定哪些周期有资格触发下单。', '建议只勾选「60分钟」：周期越短信号越密集、噪声越大，短线信号反复进出会让手续费快速吞噬本金。', '留空 = 不过滤，全部周期的信号都会下单（不推荐）。']"/></template>
              <n-select v-model:value="okxSignalTradeKltsSel" multiple :options="okxKltOptions" placeholder="留空=不过滤（全部周期都会下单）" clearable/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeMinScore">
              <template #label>最低信号分数<HelpTip :lines="['买卖点信号为多路指标共振产生（满分 14 分）：分数 = 参与共振的路数，路数越多信号越可靠。', '低于该分数的信号不自动下单；建议设 8~10 过滤低分噪声（14 路制 ≈ 57%~71%，相当于旧 10 路制 6~7 的严格度），留空 = 不限制（所有信号都下单）。', '注意：信号集扩路后满分变大，若此前设过 6~7，建议同步上调以免门槛变松。']"/></template>
              <n-input type="text" placeholder="如 8，留空=不限制" v-model:value="formValue.okxSignalTradeMinScore" clearable/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeCooldownMin">
              <template #label>止盈止损冷却(分钟)<HelpTip :lines="['止盈/止损或反向平仓后，同合约同方向在该冷却时长内不再自动开仓，防止震荡行情中被反复止损。', '反向换向不受冷却影响（平多后仍可立即开空）；默认 30 分钟，0 = 关闭冷却。']"/></template>
              <n-input type="text" placeholder="留空=默认 30 分钟" v-model:value="formValue.okxSignalTradeCooldownMin" clearable/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalMinTpPct">
              <template #label>开单最低止盈(%)<HelpTip :lines="['开仓前按当前止盈止损配置估算本笔预期止盈收益率（保证金口径，与止盈线同口径），低于该值不开单。', '固定收益率模式 = 止盈收益率本身；波动率自适应 = 平均振幅 × 止盈倍数 × 生效杠杆（杠杆未知按 10x 估算）。', '低波动合约算出的止盈线过近，扣掉双边手续费后无利可图，可用此门槛过滤这类信号。', '止盈止损未开启或未配置止盈时无法估算、不拦截；留空 = 不限制。']"/></template>
              <n-input type="text" placeholder="如 5 表示预期止盈不足 +5% 不开单，留空=不限制" v-model:value="formValue.okxSignalMinTpPct" clearable/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalMaxTpPct">
              <template #label>开单最大止盈(%)<HelpTip :lines="['与「开单最低止盈」同口径、同开仓门槛：预期止盈收益率高于该值不开单。', '波动率自适应模式下振幅过大意味着止损线同样过远、单笔最大回撤远超常规，可用于过滤极端行情/妖币合约。', '预期止盈恰好落在上下限之间才开单；止盈止损未开启或未配置止盈时无法估算、不拦截。留空 = 不限制。']"/></template>
              <n-input type="text" placeholder="如 50 表示预期止盈超过 +50% 不开单，留空=不限制" v-model:value="formValue.okxSignalMaxTpPct" clearable/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeDirMode">
              <template #label>开仓方向<HelpTip :lines="['多空都做：买入信号开多、卖出信号开空。', '只做多：仅执行买入信号开多；卖出信号只用于平掉多头持仓，不开空。', '只做空：仅执行卖出信号开空；买入信号只用于平掉空头持仓，不开多。']"/></template>
              <n-select v-model:value="formValue.okxSignalTradeDirMode"
                        :options="[{label:'多空都做',value:'both'},{label:'只做多（卖出信号仅平多）',value:'long'},{label:'只做空（买入信号仅平空）',value:'short'}]"/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeRegime">
              <template #label>顺大势门控（标的自身 1H）<HelpTip :lines="['按标的自身 1 小时线趋势过滤逆势单：多头趋势只开多、空头趋势只开空、多空过渡区不限制。', '用于减少下跌段中错误做多、上涨段中错误做空这类逆势亏损单。', '趋势判定基于标的自身 1H 均线体系，与其他标的无关。']"/></template>
              <n-switch v-model:value="formValue.okxSignalTradeRegime"/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalTradeMaxPerDay">
              <template #label>单合约单日开仓上限<HelpTip :lines="['同一合约单个自然日（UTC+8）累计开仓笔数达到上限后，当日不再对该合约自动开仓。', '避免同一波行情被反复进出、手续费不断累积；短周期信号尤其需要限制。', '留空 = 不限制。']"/></template>
              <n-input type="text" placeholder="如 2，留空=不限制" v-model:value="formValue.okxSignalTradeMaxPerDay" clearable/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalStockEnable">
              <template #label>股票信号下单<HelpTip :lines="['股票永续信号（名称带「美股/港股永续」标记）的独立开关，与加密货币信号互不影响。', '当前暂仅模拟盘语义：OKX 模拟盘未上架股票永续合约，实盘 xStocks 类产品可用。']"/></template>
              <n-switch v-model:value="formValue.okxSignalStockEnable"/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxSignalStockUSDT">
              <template #label>股票每笔金额(USDT)<HelpTip :lines="['股票永续信号每笔投入金额（名义价值口径）。', 'OKX 模拟盘未上架股票永续，实盘 xStocks 类产品可用；金额独立于加密信号设置。']"/></template>
              <n-input type="text" placeholder="股票信号每笔投入金额，默认 100" v-model:value="formValue.okxSignalStockUSDT" clearable/>
              <n-text depth="3" style="margin-left: 12px; font-size: 12px;">
                OKX 模拟盘未上架股票永续，实盘 xStocks 类产品可用
              </n-text>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxTpSlEnable">
              <template #label>止盈止损(持仓)<HelpTip :lines="['软件侧兜底监控：后台每 10 秒分别拉取模拟盘与实盘两个账户的全部持仓，任一持仓达到平仓线立即市价 reduceOnly 平仓（只减仓、不加仓）。', '与开仓时附带的交易所侧条件单（止盈止损触发价）互补：软件侧算的是账户收益率口径，交易所侧算的是价格口径，双方谁先到线谁先平，先触发的一侧平掉后软件会自动撤销另一侧的孤儿条件单。', '平仓失败自动指数退避重试（60s→2min→…封顶 30min）；强烈建议开启，关闭后仓位无自动兜底、风险敞口显著放大。']"/></template>
              <n-switch v-model:value="formValue.okxTpSlEnable"/>
            </n-form-item-gi>
            <n-form-item-gi :span="8" path="okxTpSlMode">
              <template #label>平仓线模式<HelpTip :lines="['固定收益率：用 OKX 保证金收益率 uplRatio（含杠杆、多空同号）与固定百分比比较，所有合约共用同一组阈值。', '波动率自适应（推荐）：按各合约近 N 根参考周期 K 线的平均振幅（(high-low)/open）动态缩放平仓线，波动大的合约自动放宽、波动小的自动收紧，避免固定百分比对不同合约失配。', '两种模式的阈值参数两环境（模拟盘/实盘）共用。']"/></template>
              <n-select v-model:value="formValue.okxTpSlMode"
                        :options="[{label:'固定收益率(%)',value:'fixed'},{label:'波动率自适应',value:'vol'}]"/>
            </n-form-item-gi>
            <template v-if="formValue.okxTpSlMode==='fixed'">
              <n-form-item-gi :span="8" path="okxTakeProfitPct">
                <template #label>止盈收益率(%)<HelpTip :lines="['保证金口径（含杠杆）：如 5 表示持仓浮盈达 +5% 平仓（20x 杠杆下约等于价格涨 0.25%）。', '多空同号：空头持仓按浮盈为正同样适用；0 或留空 = 不启用固定止盈。']"/></template>
                <n-input type="text" placeholder="如 5 表示 +5% 平仓；0/留空=不启用" v-model:value="formValue.okxTakeProfitPct" clearable/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" path="okxStopLossPct">
                <template #label>止损收益率(%)<HelpTip :lines="['保证金口径（含杠杆）：如 3 表示持仓浮亏达 -3% 平仓（20x 杠杆下约等于价格反向波动 0.15%）。', '注意高杠杆下固定止损百分比会非常贴近成本区，容易在正常波动中被扫出场；可改用「波动率自适应」模式。', '0 或留空 = 不启用固定止损（风险自负）。']"/></template>
                <n-input type="text" placeholder="如 3 表示 -3% 平仓；0/留空=不启用" v-model:value="formValue.okxStopLossPct" clearable/>
              </n-form-item-gi>
            </template>
            <template v-else>
              <n-form-item-gi :span="8" path="okxTpSlVolKlt">
                <template #label>波动率参考周期<HelpTip :lines="['取该周期的近 N 根 K 线计算平均振幅，作为平仓线缩放基准。', '实际生效不低于 15 分钟：5m 及更短周期振幅噪声大、阈值抖动剧烈且常得出过窄的止损线；1H 以上又过于迟钝，15m 是折中。', '默认 15m。']"/></template>
                <n-select v-model:value="formValue.okxTpSlVolKlt"
                          :options="[{label:'1 分钟',value:'1m'},{label:'5 分钟',value:'5m'},{label:'15 分钟',value:'15m'},{label:'30 分钟',value:'30m'},{label:'1 小时',value:'1H'},{label:'4 小时',value:'4H'},{label:'1 天',value:'1D'}]"/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" path="okxTpSlVolBars">
                <template #label>回看K线根数<HelpTip :lines="['参与平均振幅计算的参考周期 K 线根数，默认 20；实际按 N+1 根取数（含当前尚未收线的一根），振幅数据缓存 5 分钟。', '根数越多振幅越平滑、平仓线越稳定，过少则阈值抖动、平仓线忽宽忽窄。']"/></template>
                <n-input type="text" placeholder="默认 20" v-model:value="formValue.okxTpSlVolBars" clearable/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" path="okxTpSlTpMult">
                <template #label>止盈倍数<HelpTip :lines="['止盈线 = 平均振幅 × 止盈倍数（价格口径；账户口径阈值 = 该值 × 持仓杠杆）。', '默认搭配移动止盈：账户浮盈 ≥ 20% 后开始跟踪峰值、从峰值回吐一半即平仓（价格门槛 = 20% ÷ 实际杠杆），避免利润过早离场。', '填 0 = 关闭固定止盈，只保留移动止盈与止损。']"/></template>
                <n-input type="text" placeholder="默认 1.5；填 0 = 关闭固定止盈" v-model:value="formValue.okxTpSlTpMult" clearable/>
              </n-form-item-gi>
              <n-form-item-gi :span="8" path="okxTpSlSlMult">
                <template #label>止损倍数<HelpTip :lines="['止损线 = 平均振幅 × 止损倍数（价格口径；账户口径 = 该值 × 杠杆）。', '倍数越小止损线越窄、越容易被盘中噪声扫出场：实测 5m×20 根得出的止损线常落进手续费+滑点成本区，导致止损笔数远超止盈。建议放大到 ≥8 倍平均振幅，给波动留足空间。', '账户口径下限 10%：价格止损线最低 = 10% ÷ 持仓实际杠杆，保证不同杠杆下账户最大亏损一致（10x → 价格 1%、1x → 价格 10%）。']"/></template>
                <n-input type="text" placeholder="默认 1；建议 ≥8（约 8 倍平均振幅）" v-model:value="formValue.okxTpSlSlMult" clearable/>
              </n-form-item-gi>
            </template>
            <n-form-item-gi :span="24">
              <n-gradient-text type="info">
                <div style="font-size: 12px; line-height: 1.6;">
                  配置说明：<br>
                  1. 在 OKX「交易所-API」创建 API 密钥，权限勾选「读取」与「交易」，并将本机出口 IP 加入白名单（IP 不在白名单时私有接口会报错）<br>
                  2. Passphrase 为创建密钥时自行设置的「口令」，不是密钥备注名<br>
                  3. 支持 SPOT 现货与 SWAP 永续合约，AI 助手中说「查一下我的 OKX 持仓」「用 OKX 市价买入 0.01 BTC」即可调用<br>
                  4. 模拟盘与实盘的 API 密钥是两套：模拟盘密钥在 OKX「模拟交易」环境单独创建；
                     实盘密钥在正式环境创建并绑定 IP 白名单。手动交易环境由「合约行情」页的
                     「模拟盘 / 实盘」双页签显式决定（各自固定用对应密钥，互不影响），
                     信号自动交易由「模拟盘自动下单 / 实盘自动下单」两个独立开关控制<br>
                  5. 「模拟盘自动下单」与「实盘自动下单」相互独立，可同时启用或各自暂停；
                     监控池也按环境分成两套（信号监控页签「OKX 模拟盘池 / 实盘池」，
                     交易面板「模拟盘 / 实盘」页签各自编辑对应池）：
                     模拟盘池的信号只在模拟盘下单、实盘池只在实盘下单（真实资金，请谨慎！）；
                     同一加密买卖点信号（buysell 族，ok:/bn:/bt: 代码）放入两池时两个环境各下一笔（金额各自独立）；
                     成交记录见信号监控面板「自动交易」记录，并按环境（模拟盘/实盘）标注；
                     「股票信号下单」为股票永续信号（Name 带美股/港股永续标记）的独立开关与金额，暂仅模拟盘<br>
                  6. 「止盈止损(持仓)」：固定模式按保证金收益率 uplRatio（含杠杆）与固定百分比比较；
                     波动率自适应模式按各合约近 N 根参考周期 K 线的平均振幅动态计算平仓线
                     （止盈线=振幅×止盈倍数、止损线=振幅×止损倍数，均以账户口径=价格口径收益×实际杠杆比较；
                     止损下限恒为账户 10%，价格止损线=10%÷实际杠杆，不同杠杆下账户最大亏损一致），
                     账户浮盈达 20% 后进入移动止盈跟踪、从峰值回吐一半即平仓（价格门槛=20%÷实际杠杆）；
                     手动开仓与信号自动开的仓都会被监控（模拟盘与实盘两个账户独立检查）；
                     平仓事件记录在应用日志（关键词「止盈止损·模拟盘 / 止盈止损·实盘」）
                </div>
              </n-gradient-text>
            </n-form-item-gi>
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