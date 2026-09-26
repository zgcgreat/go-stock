import { ref, onMounted, onBeforeUnmount } from 'vue'

const MOBILE_BREAKPOINT = 768

export function useResponsive() {
  const isMobile = ref(false)

  const checkMobile = () => {
    isMobile.value = window.innerWidth < MOBILE_BREAKPOINT
  }

  onMounted(() => {
    checkMobile()
    window.addEventListener('resize', checkMobile)
  })

  onBeforeUnmount(() => {
    window.removeEventListener('resize', checkMobile)
  })

  return { isMobile }
}

export function useIsWebMode() {
  // 检测是否为 Web 模式（非 Wails 桌面端）。
  // window.go 由 Wails 运行时注入，可能在组件挂载后才就绪；
  // 因此不能只做一次性求值，需要在挂载后复查并短暂轮询，避免永久误判为 Web 模式。
  const detect = () => {
    try {
      if (typeof window === 'undefined') return true
      return !(window.go && window.go.main && window.go.main.App)
    } catch (e) {
      return true
    }
  }

  const isWebMode = ref(detect())
  let timer = null

  onMounted(() => {
    isWebMode.value = detect()
    if (!isWebMode.value) return // 已确认为桌面端，无需轮询
    let tries = 0
    timer = setInterval(() => {
      const next = detect()
      if (next !== isWebMode.value) isWebMode.value = next
      // 检测到桌面端或超过 2s 后停止轮询
      if (!next || ++tries >= 20) {
        clearInterval(timer)
        timer = null
      }
    }, 100)
  })

  onBeforeUnmount(() => {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  })

  return { isWebMode }
}
