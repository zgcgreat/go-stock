import { ref, computed } from 'vue'
import { GetConfig, GetTelegraphList } from '../services/wails-bridge.js'
import Auth from '../utils/auth'

// ===== 快讯轮播数据源（桌面端「telegraph」Wails 事件的 Web 等价物）=====
// 桌面端：Go cron 60s 直抓财联社 → runtime.EventsEmit("telegraph") → App.vue EventsOn 消费。
// Web 端：无 Wails 事件通道，改为前端轮询既有接口 GET /telegraph（读本地 telegraph_list 表，
// 由 webserver scheduler 交易时段每 30 分钟抓取入库）。60s 轮询频率与桌面端刷新节奏一致。
//
// 模块级单例：WebAppShell 与 TelegraphMarquee 共享同一份状态，避免重复拉取。

const enableNews = ref(false)
const telegraph = ref([])
const visible = computed(() => enableNews.value && telegraph.value.length > 0)

let timer = null
let started = false

function fetchTelegraph() {
  // 登录页/令牌失效时跳过，避免 401 刷屏（与 WebLayout.loadMenuVisibility 同款守卫）
  if (!Auth.getToken()) return
  GetTelegraphList('')
    .then((news) => {
      // 接口返回 Telegraph 模型数组，轮播条只展示内容文本（与桌面端事件载荷 []string 对齐）
      telegraph.value = (Array.isArray(news) ? news : [])
        .map((item) => (typeof item === 'string' ? item : item?.content || ''))
        .filter((content) => content)
    })
    .catch(() => {})
}

function loadConfig() {
  const token = Auth.getToken()
  if (!token) return
  GetConfig()
    .then((config) => {
      enableNews.value = !!config?.enableNews
      if (enableNews.value) {
        fetchTelegraph()
      }
    })
    .catch(() => {})
}

// start 幂等：同一页面生命周期内重复调用（如组件重建）不会叠加定时器
function start() {
  if (started) return
  started = true
  loadConfig()
  timer = setInterval(fetchTelegraph, 60 * 1000)
  // 登录成功/登出时重载配置（WebAppShell 不随路由重建，需事件驱动刷新）
  window.addEventListener('user-info-updated', loadConfig)
}

function stop() {
  started = false
  if (timer) {
    clearInterval(timer)
    timer = null
  }
  window.removeEventListener('user-info-updated', loadConfig)
  // 复位状态，避免登出后残留上一账号的数据
  enableNews.value = false
  telegraph.value = []
}

export function useTelegraph() {
  return { enableNews, telegraph, visible, start, stop }
}
