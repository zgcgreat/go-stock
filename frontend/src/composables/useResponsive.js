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
  // 检测是否为 Web 模式（非 Wails 桌面端）
  const isWebMode = ref(!window.go || !window.go.main || !window.go.main.App)
  return { isWebMode }
}
