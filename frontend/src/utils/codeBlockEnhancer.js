/**
 * 代码块增强工具
 * 功能：行号显示、语言标签、一键复制、折叠功能
 */

/**
 * 增强代码块显示
 * @param {HTMLElement} container - 包含代码块的容器元素
 * @param {Object} options - 配置选项
 * @param {number} options.collapseThreshold - 折叠阈值（行数），默认 8
 * @param {boolean} options.addLineNumbers - 是否添加行号，默认 true
 * @param {boolean} options.addCopyButton - 是否添加复制按钮，默认 true
 * @param {boolean} options.addLanguageTag - 是否添加语言标签，默认 true
 */
export function enhanceCodeBlocks(container, options = {}) {
  const {
    collapseThreshold = 8,
    addLineNumbers = true,
    addCopyButton = true,
    addLanguageTag = true
  } = options

  const codeBlocks = container.querySelectorAll('.md-editor-code-block')

  codeBlocks.forEach(block => {
    // 跳过已处理的代码块
    if (block.dataset.enhanced === 'true') return

    const codeEl = block.querySelector('code')
    if (!codeEl) return

    // 获取语言信息
    const lang = extractLanguage(codeEl)
    const text = codeEl.textContent || ''
    const lineCount = text.split('\n').length

    // 标记为已处理
    block.dataset.enhanced = 'true'

    // 添加语言标签
    if (addLanguageTag && lang) {
      addLangTag(block, lang)
    }

    // 添加复制按钮
    if (addCopyButton) {
      addCopyBtn(block, text)
    }

    // 添加行号
    if (addLineNumbers) {
      addLineNumbersToBlock(block, codeEl, lineCount)
    }

    // 折叠功能（JSON 或超过阈值的代码块）
    const shouldCollapse = lang === 'json' || lineCount > collapseThreshold
    if (shouldCollapse) {
      addCollapseBtn(block)
    }
  })
}

/**
 * 从 code 元素提取语言信息
 */
function extractLanguage(codeEl) {
  const className = codeEl.className || ''
  // 匹配 language-* 或 hljs-* 格式
  const match = className.match(/(?:language-|hljs)(\w+)/)
  if (match) return match[1]

  // 尝试从父元素获取
  const pre = codeEl.parentElement
  if (pre && pre.className) {
    const preMatch = pre.className.match(/(?:language-)(\w+)/)
    if (preMatch) return preMatch[1]
  }

  return ''
}

/**
 * 添加语言标签
 */
function addLangTag(block, lang) {
  const tag = document.createElement('span')
  tag.className = 'code-lang-tag'
  tag.textContent = lang.toUpperCase()
  block.appendChild(tag)
}

/**
 * 添加复制按钮
 */
function addCopyBtn(block, text) {
  const btn = document.createElement('button')
  btn.className = 'code-copy-btn'
  btn.type = 'button'
  btn.title = '复制代码'
  btn.innerHTML = `
    <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
      <path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
    </svg>
  `

  btn.addEventListener('click', async (e) => {
    e.stopPropagation()
    try {
      await copyToClipboard(text)
      showCopySuccess(btn)
    } catch (err) {
      showCopyError(btn)
    }
  })

  block.appendChild(btn)
}

/**
 * 复制文本到剪贴板
 */
async function copyToClipboard(text) {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    await navigator.clipboard.writeText(text)
  } else {
    // 降级方案
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
  }
}

/**
 * 显示复制成功状态
 */
function showCopySuccess(btn) {
  btn.classList.add('copy-success')
  btn.innerHTML = `
    <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
    </svg>
  `
  setTimeout(() => {
    btn.classList.remove('copy-success')
    btn.innerHTML = `
      <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
        <path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
      </svg>
    `
  }, 2000)
}

/**
 * 显示复制失败状态
 */
function showCopyError(btn) {
  btn.classList.add('copy-error')
  btn.title = '复制失败'
  setTimeout(() => {
    btn.classList.remove('copy-error')
    btn.title = '复制代码'
  }, 2000)
}

/**
 * 添加行号
 */
function addLineNumbersToBlock(block, codeEl, lineCount) {
  // 使用 CSS counter 方式，更高效
  const pre = codeEl.parentElement
  if (!pre) return

  pre.classList.add('code-with-line-numbers')
  block.dataset.lineCount = lineCount

  // 为超过一定行数的代码块添加行号显示
  if (lineCount > 3) {
    block.classList.add('show-line-numbers')
  }
}

/**
 * 添加折叠按钮
 */
function addCollapseBtn(block) {
  // 检查是否已有折叠按钮
  if (block.querySelector('.code-collapse-btn')) return

  block.classList.add('code-collapsed')

  const btn = document.createElement('span')
  btn.className = 'code-collapse-btn'
  btn.textContent = '展开'
  btn.addEventListener('click', (e) => {
    e.stopPropagation()
    const collapsed = block.classList.toggle('code-collapsed')
    btn.textContent = collapsed ? '展开' : '收起'
  })

  block.appendChild(btn)
}

/**
 * 获取代码主题
 * @param {boolean} isDark - 是否深色模式
 * @returns {string} - 主题名称
 */
export function getCodeTheme(isDark) {
  return isDark ? 'atom-one-dark' : 'atom-one-light'
}

/**
 * CSS 样式（需要添加到组件或全局样式中）
 */
export const codeBlockEnhanceStyles = `
/* 代码块增强样式 */

/* 语言标签 */
.code-lang-tag {
  position: absolute;
  top: 8px;
  left: 12px;
  z-index: 2;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 500;
  color: var(--n-text-color-3);
  background: var(--n-color-hover);
  border-radius: 4px;
  opacity: 0.8;
  pointer-events: none;
}

/* 复制按钮 */
.code-copy-btn {
  position: absolute;
  top: 8px;
  right: 12px;
  z-index: 2;
  padding: 4px 8px;
  font-size: 12px;
  color: var(--n-text-color-3);
  background: var(--n-color-hover);
  border: 1px solid var(--n-border-color);
  border-radius: 4px;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.2s, background 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.md-editor-code-block:hover .code-copy-btn {
  opacity: 1;
}

.code-copy-btn:hover {
  background: var(--n-primary-color-suppl);
  border-color: var(--n-primary-color);
  color: var(--n-primary-color);
}

.code-copy-btn.copy-success {
  color: #67c23a;
  border-color: #67c23a;
}

.code-copy-btn.copy-error {
  color: #f56c6c;
  border-color: #f56c6c;
}

/* 行号样式 */
.md-editor-code-block.show-line-numbers pre.code-with-line-numbers {
  padding-left: 3em;
  position: relative;
}

.md-editor-code-block.show-line-numbers pre.code-with-line-numbers::before {
  content: '';
  position: absolute;
  left: 2.5em;
  top: 0;
  bottom: 0;
  width: 1px;
  background: var(--n-border-color);
}

/* 折叠按钮 */
.code-collapse-btn {
  position: absolute;
  bottom: 8px;
  right: 12px;
  z-index: 2;
  padding: 2px 8px;
  font-size: 11px;
  color: var(--n-text-color-3);
  background: var(--n-color-hover);
  border: 1px solid var(--n-border-color);
  border-radius: 4px;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.2s;
}

.md-editor-code-block:hover .code-collapse-btn {
  opacity: 1;
}

/* 折叠状态 */
.md-editor-code-block.code-collapsed pre {
  max-height: 120px;
  overflow: hidden;
}

.md-editor-code-block.code-collapsed::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 50px;
  background: linear-gradient(transparent, var(--n-color));
  pointer-events: none;
}

/* 深色模式适配 */
[theme-mode="dark"] .code-lang-tag,
[theme-mode="dark"] .code-copy-btn,
[theme-mode="dark"] .code-collapse-btn {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.15);
}
`