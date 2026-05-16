/**
 * 打字机效果工具
 * 用于思考过程的逐字显示效果
 */

/**
 * 创建打字机效果
 * @param {Object} options - 配置选项
 * @param {number} options.speed - 每个字符的显示间隔（毫秒），默认 30
 * @param {number} options.pauseOnPunctuation - 标点符号处的额外暂停（毫秒），默认 100
 * @param {string[]} options.punctuationChars - 需要暂停的标点符号列表
 * @returns {Object} - 打字机控制器
 */
export function createTypewriter(options = {}) {
  const {
    speed = 30,
    pauseOnPunctuation = 100,
    punctuationChars = ['.', ',', '!', '?', ';', ':', '。', '，', '！', '？', '；', '：', '\n']
  } = options

  let isRunning = false
  let currentIndex = 0
  let fullText = ''
  let timerId = null
  let onUpdate = null
  let onComplete = null

  /**
   * 开始打字机效果
   * @param {string} text - 要显示的完整文本
   * @param {Function} updateCallback - 每次更新时调用的回调，参数为当前显示的文本
   * @param {Function} completeCallback - 完成时调用的回调
   */
  function start(text, updateCallback, completeCallback) {
    if (isRunning) {
      stop()
    }

    fullText = text
    currentIndex = 0
    onUpdate = updateCallback
    onComplete = completeCallback
    isRunning = true

    // 立即显示第一个字符
    if (fullText.length > 0) {
      tick()
    } else {
      complete()
    }
  }

  /**
   * 处理下一个字符
   */
  function tick() {
    if (!isRunning || currentIndex >= fullText.length) {
      complete()
      return
    }

    currentIndex++
    const currentText = fullText.substring(0, currentIndex)

    if (onUpdate) {
      onUpdate(currentText)
    }

    // 检查是否是标点符号，需要额外暂停
    const lastChar = fullText[currentIndex - 1]
    const delay = punctuationChars.includes(lastChar) ? pauseOnPunctuation : speed

    timerId = setTimeout(tick, delay)
  }

  /**
   * 完成打字机效果
   */
  function complete() {
    isRunning = false
    if (timerId) {
      clearTimeout(timerId)
      timerId = null
    }
    if (onComplete) {
      onComplete(fullText)
    }
  }

  /**
   * 停止打字机效果（立即显示全部文本）
   */
  function stop() {
    isRunning = false
    if (timerId) {
      clearTimeout(timerId)
      timerId = null
    }
    // 立即显示全部文本
    if (onUpdate && fullText) {
      onUpdate(fullText)
    }
    if (onComplete) {
      onComplete(fullText)
    }
  }

  /**
   * 中断打字机效果（不显示剩余文本）
   */
  function abort() {
    isRunning = false
    if (timerId) {
      clearTimeout(timerId)
      timerId = null
    }
  }

  /**
   * 添加更多文本（用于流式输入）
   * @param {string} additionalText - 要添加的文本
   */
  function append(additionalText) {
    const previousLength = fullText.length
    fullText += additionalText

    // 如果当前正在运行，继续处理新字符
    if (isRunning) {
      // 已经在 tick 循环中，会自动处理新字符
    } else {
      // 如果已停止，立即显示新文本
      if (onUpdate) {
        onUpdate(fullText)
      }
    }
  }

  /**
   * 获取当前状态
   */
  function getStatus() {
    return {
      isRunning,
      currentIndex,
      totalLength: fullText.length,
      progress: fullText.length > 0 ? currentIndex / fullText.length : 0
    }
  }

  return {
    start,
    stop,
    abort,
    append,
    getStatus
  }
}

/**
 * 流式文本动画处理器
 * 用于处理 SSE 流式输出的实时动画效果
 */
export function createStreamingAnimator(options = {}) {
  const {
    chunkDelay = 50, // 每个数据块的显示延迟
    maxQueueSize = 100 // 最大队列大小
  } = options

  let queue = []
  let isProcessing = false
  let currentText = ''
  let onUpdate = null
  let timerId = null

  /**
   * 初始化
   * @param {Function} updateCallback - 更新回调
   */
  function init(updateCallback) {
    onUpdate = updateCallback
    currentText = ''
    queue = []
    isProcessing = false
  }

  /**
   * 添加文本块
   * @param {string} text - 文本块
   */
  function addChunk(text) {
    if (!text) return

    // 防止队列过大
    if (queue.length >= maxQueueSize) {
      // 直接合并到当前文本
      currentText += queue.join('')
      queue = []
      if (onUpdate) {
        onUpdate(currentText)
      }
    }

    queue.push(text)

    if (!isProcessing) {
      processQueue()
    }
  }

  /**
   * 处理队列
   */
  function processQueue() {
    if (queue.length === 0) {
      isProcessing = false
      return
    }

    isProcessing = true
    const chunk = queue.shift()
    currentText += chunk

    if (onUpdate) {
      onUpdate(currentText)
    }

    timerId = setTimeout(processQueue, chunkDelay)
  }

  /**
   * 完成处理
   */
  function complete() {
    if (timerId) {
      clearTimeout(timerId)
      timerId = null
    }

    // 处理剩余队列
    if (queue.length > 0) {
      currentText += queue.join('')
      queue = []
    }

    isProcessing = false
    if (onUpdate) {
      onUpdate(currentText)
    }

    return currentText
  }

  /**
   * 中断处理
   */
  function abort() {
    if (timerId) {
      clearTimeout(timerId)
      timerId = null
    }
    isProcessing = false
    queue = []
    return currentText
  }

  /**
   * 获取当前文本
   */
  function getText() {
    return currentText
  }

  /**
   * 清空
   */
  function clear() {
    abort()
    currentText = ''
    queue = []
  }

  return {
    init,
    addChunk,
    complete,
    abort,
    getText,
    clear
  }
}

/**
 * CSS 样式（用于思考过程的动画效果）
 */
export const thinkingProcessStyles = `
/* 思考过程动画样式 */

/* 流式输出时的边框发光效果 */
.msg-reasoning-wrapper.streaming {
  border-color: var(--n-primary-color);
  box-shadow: 0 0 8px rgba(var(--n-primary-color-rgb), 0.3);
  animation: border-glow 1.5s ease-in-out infinite;
}

@keyframes border-glow {
  0%, 100% {
    box-shadow: 0 0 4px rgba(var(--n-primary-color-rgb), 0.2);
  }
  50% {
    box-shadow: 0 0 12px rgba(var(--n-primary-color-rgb), 0.4);
  }
}

/* 打字机光标效果 */
.typewriter-cursor::after {
  content: '|';
  animation: cursor-blink 0.8s infinite;
  color: var(--n-primary-color);
  font-weight: bold;
}

@keyframes cursor-blink {
  0%, 50% {
    opacity: 1;
  }
  51%, 100% {
    opacity: 0;
  }
}

/* 步骤进度条 */
.msg-steps-progress {
  height: 4px;
  background: var(--n-border-color);
  border-radius: 2px;
  overflow: hidden;
  margin: 8px 0;
}

.msg-steps-progress-bar {
  height: 100%;
  background: linear-gradient(90deg, var(--n-primary-color), #67c23a);
  transition: width 0.3s ease;
}

/* 步骤完成动画 */
.msg-step-dot.step-done {
  animation: step-done-pop 0.3s ease;
}

@keyframes step-done-pop {
  0% {
    transform: scale(0.8);
  }
  50% {
    transform: scale(1.2);
  }
  100% {
    transform: scale(1);
  }
}

/* 平滑高度过渡 */
.msg-reasoning-content,
.msg-steps-content {
  transition: max-height 0.3s ease, opacity 0.2s ease;
}
`