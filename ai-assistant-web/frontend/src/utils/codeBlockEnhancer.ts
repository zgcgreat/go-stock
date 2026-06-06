/**
 * 代码块增强工具
 * 功能：行号显示、语言标签、一键复制、折叠功能
 */

interface EnhanceOptions {
  collapseThreshold?: number;
  addLineNumbers?: boolean;
  addCopyButton?: boolean;
  addLanguageTag?: boolean;
}

/**
 * 增强代码块显示
 */
export function enhanceCodeBlocks(container: HTMLElement, options: EnhanceOptions = {}): void {
  const {
    collapseThreshold = 8,
    addLineNumbers = true,
    addCopyButton = true,
    addLanguageTag = true
  } = options

  const codeBlocks = container.querySelectorAll<HTMLElement>('.md-editor-code-block')

  codeBlocks.forEach(block => {
    if (block.dataset.enhanced === 'true') return

    const codeEl = block.querySelector<HTMLElement>('code')
    if (!codeEl) return

    const lang = extractLanguage(codeEl)
    const text = codeEl.textContent || ''
    const lineCount = text.split('\n').length

    block.dataset.enhanced = 'true'

    if (addLanguageTag && lang) {
      addLangTag(block, lang)
    }

    if (addCopyButton) {
      addCopyBtn(block, text)
    }

    if (addLineNumbers) {
      addLineNumbersToBlock(block, codeEl, lineCount)
    }

    const shouldCollapse = lang === 'json' || lineCount > collapseThreshold
    if (shouldCollapse) {
      addCollapseBtn(block)
    }
  })
}

function extractLanguage(codeEl: HTMLElement): string {
  const className = codeEl.className || ''
  const match = className.match(/(?:language-|hljs)(\w+)/)
  if (match) return match[1]

  const pre = codeEl.parentElement
  if (pre && pre.className) {
    const preMatch = pre.className.match(/(?:language-)(\w+)/)
    if (preMatch) return preMatch[1]
  }

  return ''
}

function addLangTag(block: HTMLElement, lang: string): void {
  const tag = document.createElement('span')
  tag.className = 'code-lang-tag'
  tag.textContent = lang.toUpperCase()
  block.appendChild(tag)
}

function addCopyBtn(block: HTMLElement, text: string): void {
  const btn = document.createElement('button')
  btn.className = 'code-copy-btn'
  btn.type = 'button'
  btn.title = '复制代码'
  btn.innerHTML = `
    <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
      <path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
    </svg>
  `

  btn.addEventListener('click', async (e: Event) => {
    e.stopPropagation()
    try {
      await copyToClipboard(text)
      showCopySuccess(btn)
    } catch {
      showCopyError(btn)
    }
  })

  block.appendChild(btn)
}

async function copyToClipboard(text: string): Promise<void> {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    await navigator.clipboard.writeText(text)
  } else {
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

function showCopySuccess(btn: HTMLButtonElement): void {
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

function showCopyError(btn: HTMLButtonElement): void {
  btn.classList.add('copy-error')
  btn.title = '复制失败'
  setTimeout(() => {
    btn.classList.remove('copy-error')
    btn.title = '复制代码'
  }, 2000)
}

function addLineNumbersToBlock(block: HTMLElement, codeEl: HTMLElement, lineCount: number): void {
  const pre = codeEl.parentElement
  if (!pre) return

  pre.classList.add('code-with-line-numbers')
  block.dataset.lineCount = String(lineCount)

  if (lineCount > 3) {
    block.classList.add('show-line-numbers')
  }
}

function addCollapseBtn(block: HTMLElement): void {
  if (block.querySelector('.code-collapse-btn')) return

  block.classList.add('code-collapsed')

  const btn = document.createElement('span')
  btn.className = 'code-collapse-btn'
  btn.textContent = '展开'
  btn.addEventListener('click', (e: Event) => {
    e.stopPropagation()
    const collapsed = block.classList.toggle('code-collapsed')
    btn.textContent = collapsed ? '展开' : '收起'
  })

  block.appendChild(btn)
}

/**
 * 获取代码主题
 */
export function getCodeTheme(isDark: boolean): string {
  return isDark ? 'atom-one-dark' : 'atom-one-light'
}
