<template>
  <div class="chat-box">
    <t-chat
        ref="chatRef"
        :clear-history="chatList.length > 0 && !isStreamLoad"
        :data="chatList"
        :text-loading="loading"
        :is-stream-load="isStreamLoad"
        style="height: 100%"
        @scroll="handleChatScroll"
        @clear="clearConfirm"
    >
      <!-- eslint-disable vue/no-unused-vars -->
      <template #content="{ item, index }">
        <div v-if="item.role === 'assistant' && item.steps && item.steps.length > 0" class="agent-steps">
          <div class="agent-steps-header">📋 执行步骤 <span class="agent-steps-badge">{{ item.steps.length }}</span></div>
          <div class="agent-steps-list">
            <div v-for="(step, si) in item.steps" :key="si" class="agent-step-item">
              <div class="agent-step-dot" :class="getStepDotClass(step)"></div>
              <span class="agent-step-text">{{ step }}</span>
            </div>
          </div>
        </div>
        <!-- 思考过程区域：添加思考指示器 -->
        <div v-if="item.role === 'assistant'" class="reasoning-wrapper">
          <div class="reasoning-header" @click="toggleReasoning(index)">
            <div class="reasoning-indicator" :class="{ 'thinking': isStreamLoad && !item.reasoning }">
              <svg class="thinking-icon" viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
              </svg>
              <span class="thinking-text">{{ isStreamLoad && !item.reasoning ? '思考中...' : '思考过程' }}</span>
              <div v-if="isStreamLoad && !item.reasoning" class="thinking-dots">
                <span class="dot"></span>
                <span class="dot"></span>
                <span class="dot"></span>
              </div>
            </div>
            <svg :class="['reasoning-arrow', { 'expanded': reasoningExpandedMap[index] }]" viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
              <path d="M7 10l5 5 5-5z"/>
            </svg>
          </div>
          <div v-show="reasoningExpandedMap[index] || (item.reasoning && item.reasoning.length > 0)" class="reasoning-content">
            <t-chat-loading v-if="isStreamLoad && (!item.reasoning || item.reasoning.length === 0)" text="" />
            <MdPreview
              v-if="item.reasoning && item.reasoning.length > 0"
              :model-value="item.reasoning"
              :theme="mdTheme"
              :code-theme="codeTheme"
              :editor-id="'agent-reasoning-' + index"
              :style="{ textAlign: 'left' }"
              class="msg-markdown"
              @onHtmlChanged="onMdHtmlChanged"
            />
          </div>
        </div>
        <div v-if="item.role === 'assistant' && item.jsonMarkdown" class="agent-json-md">
          <div class="agent-json-md-header" @click="toggleJsonMd(index)">
            <svg :class="['agent-json-md-arrow', { 'agent-json-md-arrow-expanded': jsonMdExpandedMap[index] }]" viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><path d="M9 6l6 6-6 6z"/></svg>
            <span class="agent-json-md-title">📊 分析报告</span>
          </div>
          <div v-show="jsonMdExpandedMap[index]" class="agent-json-md-content">
            <MdPreview
              :model-value="item.jsonMarkdown"
              :theme="mdTheme"
              :code-theme="codeTheme"
              :editor-id="'agent-json-md-' + index"
              :style="{ textAlign: 'left' }"
              class="msg-markdown"
              @onHtmlChanged="onMdHtmlChanged"
            />
          </div>
        </div>
        <MdPreview
          v-if="item.content.length > 0"
          :model-value="item.content"
          :theme="mdTheme"
          :code-theme="codeTheme"
          :editor-id="'agent-msg-' + index"
          :style="{ textAlign: 'left' }"
          class="msg-markdown"
          @onHtmlChanged="onMdHtmlChanged"
        />
      </template>
      <template #actions="{ item, index }">
        <t-chat-action
            :content="item.content"
            :operation-btn="['copy']"
            @operation="handleOperation"
        />
        <NButton v-if="item.role === 'assistant' && item.content && item.content.trim()" quaternary size="tiny" :loading="exportImageKey === String(index)" title="导出为图片" @click="exportAiReplyImage(index)" style="margin-left:4px;font-size:12px;">
          导出图
        </NButton>
      </template>
      <template #footer>
<!--        <t-chat-input :stop-disabled="isStreamLoad" @send="inputEnter" @stop="onStop"> </t-chat-input>-->
          <t-chat-sender
              ref="chatSenderRef"
              v-model="inputValue"
              class="chat-sender"
              :textarea-props="{
                placeholder: '请输入消息...',
              }"
              :loading="loading"
              :stop-disabled="isStreamLoad"
              @send="inputEnter"
              @stop="onStop"
          >
            <template #suffix>
              <!-- 监听键盘回车发送事件需要在sender组件监听 -->
              <t-button theme="default" variant="text" size="large" class="btn" @click="inputEnter"> 发送 </t-button>
            </template>
            <template #prefix>
              <NFlex :wrap="false" align="center" :size="4">
                <NSelect
                    v-model:value="selectValue"
                    :options="selectOptions"
                    label-field="name" value-field="ID"
                    size="tiny"
                    style="width: 180px;"
                    placeholder="选择模型"
                />
                <NSelect
                    v-model:value="sysPromptId"
                    :options="sysPromptOptions"
                    size="tiny"
                    clearable
                    style="width: 130px;"
                    placeholder="系统提示词"
                />
                <NSelect
                    v-model:value="userPromptId"
                    :options="userPromptOptions"
                    size="tiny"
                    clearable
                    style="width: 130px;"
                    placeholder="用户提示词"
                    @update:value="onUserPromptChange"
                />
                <NSelect
                    v-model:value="agentMode"
                    :options="agentModeOptions"
                    size="tiny"
                    style="width: 100px;"
                />
                <NSelect
                    v-model:value="selectedSkillIds"
                    :options="skillOptions"
                    size="tiny"
                    multiple
                    clearable
                    style="width: 130px;"
                    placeholder="技能"
                    max-tag-count="1"
                />
                <div style="display:flex;align-items:center;gap:2px;white-space:nowrap;">
                  <span style="font-size:12px;color:var(--td-text-color-secondary);">思考</span>
                  <NSwitch v-model:value="thinkingMode" size="small" />
                </div>
                <div style="display:flex;align-items:center;gap:2px;white-space:nowrap;">
                  <span style="font-size:12px;color:var(--td-text-color-secondary);">记忆</span>
                  <NSwitch v-model:value="memoryMode" size="small" />
                </div>
                <NSelect
                    v-if="memoryMode"
                    v-model:value="memoryCount"
                    :options="memoryCountOptions"
                    size="tiny"
                    style="width: 70px;"
                />
                <NButton size="tiny" quaternary :loading="shareLoading" @click="shareAiToCommunity" title="分享到社区">
                  分享
                </NButton>
              </NFlex>
              <Transition name="hint-fade">
                <div v-if="hintVisible" style="font-size:12px;color:#854F0B;padding:2px 8px;">{{ hintText }}</div>
              </Transition>
            </template>
          </t-chat-sender>

      </template>
    </t-chat>
    <t-button v-show="isShowToBottom" variant="text" class="bottomBtn" @click="backBottom">
      <div class="to-bottom">
        <ArrowDownIcon />
      </div>
    </t-button>
  </div>
</template>
<script setup lang="ts">
import {ref, onMounted, h, onBeforeUnmount, onBeforeMount, nextTick, computed, watch} from 'vue';
import {ArrowDownIcon, CheckCircleIcon, SystemSumIcon} from 'tdesign-icons-vue-next';
import { MdPreview } from 'md-editor-v3';
import 'md-editor-v3/lib/preview.css';
import { enhanceCodeBlocks, getCodeTheme } from '../utils/codeBlockEnhancer.js';
const loading = ref(false);

const inputValue = ref('');
// 流式数据加载中
const isStreamLoad = ref(false);
let formatTimer = null

const chatRef = ref(null);
const isShowToBottom = ref(false);

const icon = ref('https://raw.githubusercontent.com/ArvinLovegood/go-stock/master/build/appicon.png');
import {darkTheme, NFlex, NImage, NSelect, NSwitch, NButton, NIcon, useMessage} from "naive-ui";
import html2canvas from 'html2canvas';
import {ChatWithAgent, GetAiConfigs, GetConfig, GetSponsorInfo, GetVersionInfo,EventsOff, EventsOn, SaveAiAssistantSession, GetAiAssistantSession, AbortChatWithAgent, GetPromptTemplates, ShareText, SaveImage, GetAllSkills} from "../services/wails-bridge.js";
import 'tdesign-vue-next/es/style/index.css';


const allowToolTip = ref(true);
const chatSenderRef = ref(null);
const selectOptions = ref([]);
const selectValue = ref("default");
const agentMode = ref('auto')
const agentModeOptions = [
  { label: '🤖 自动', value: 'auto' },
  { label: '⚡ 快速', value: 'react' },
  { label: '🧠 规划', value: 'plan_execute' },
]
const jsonMdExpandedMap = ref({})
const reasoningExpandedMap = ref({})

// 提示词模板
const sysPromptTemplates = ref([])
const sysPromptOptions = computed(() => sysPromptTemplates.value.map(t => ({ label: t.name ?? '', value: t.ID ?? t.id })))
const sysPromptId = ref(null)
const userPromptTemplates = ref([])
const userPromptOptions = computed(() => userPromptTemplates.value.map(t => ({ label: t.name ?? '', value: t.ID ?? t.id })))
const userPromptId = ref(null)

// 思考模式 / 记忆模式
const thinkingMode = ref(true)
const memoryMode = ref(false)
const memoryCount = ref(1)
const memoryCountOptions = [
  { label: '1条', value: 1 }, { label: '2条', value: 2 },
  { label: '3条', value: 3 }, { label: '5条', value: 5 },
  { label: '10条', value: 10 },
]

// 技能选择
const skillList = ref([])
const skillOptions = computed(() => skillList.value.map(s => ({ label: s.name ?? '', value: s.id ?? s.ID })))
const selectedSkillIds = ref([])
function onSkillChange(ids) {
  // 保留顺序，只保留已选且仍有效的
  selectedSkillIds.value = ids
}

// 模型智能切换提示
const hintVisible = ref(false)
const hintText = ref('')
let hintTimer = null
function showHint(text) {
  hintText.value = text
  hintVisible.value = true
  if (hintTimer) clearTimeout(hintTimer)
  hintTimer = setTimeout(() => { hintVisible.value = false }, 3000)
}

watch(agentMode, (val) => {
  if (val === 'react') showHint('⚡ 快速模式推荐使用DeepSeek最新版')
  else if (val === 'plan_execute') showHint('🧠 规划模式推荐使用GLM最新版')
})

watch(selectValue, (val) => {
  const opts = selectOptions.value
  if (!opts?.length) return
  const found = opts.find(o => o.ID === val || o.id === val)
  if (!found) return
  const label = ((found.name ?? '') + ' ' + (found.modelName ?? '')).toLowerCase()
  if (label.includes('deepseek-chat')) {
    agentMode.value = 'plan_execute'
    thinkingMode.value = false
    showHint('deepseek-chat 已使用规划模式并关闭思考模式')
  } else if (label.includes('deepseek')) {
    showHint('⚡ DeepSeek模型推荐使用快速模式')
  } else if (label.includes('glm')) {
    showHint('🧠 GLM模型推荐使用规划模式')
  }
})

function onUserPromptChange(id) {
  if (!id) return
  const t = userPromptTemplates.value.find(x => (x.ID ?? x.id) === id)
  if (t?.content) inputValue.value = t.content
}

// 分享 / 导出
const shareLoading = ref(false)
const exportImageKey = ref('')
const message = useMessage()

function shareAiToCommunity() {
  // 获取最后一条 assistant 消息内容
  let text = ''
  for (let i = 0; i < chatList.value.length; i++) {
    if (chatList.value[i].role === 'assistant' && chatList.value[i].content?.trim()) {
      text = chatList.value[i].content.trim()
    }
  }
  if (!text) { message.warning('暂无可分享的内容'); return }
  if (shareLoading.value) return
  shareLoading.value = true
  ShareText(text, 'go-stock AI Agent助手')
    .then(msg => message.success(msg || '分享成功'))
    .catch(err => message.error('分享失败: ' + (err?.message ?? err)))
    .finally(() => { shareLoading.value = false })
}

async function exportAiReplyImage(index) {
  const key = String(index)
  if (exportImageKey.value) return
  exportImageKey.value = key
  await nextTick()
  try {
    const editorId = 'agent-msg-' + index
    const target = document.getElementById(`${editorId}-preview-wrapper`) ||
      document.getElementById(`${editorId}-preview`) || null
    if (!target) {
      message.warning('未找到预览区域，请展开回答后重试')
      return
    }
    // 临时移除 overflow 限制，确保完整截图
    const savedStyles = []
    let el = target.parentElement
    while (el && el !== document.body) {
      const style = getComputedStyle(el)
      if (style.overflow === 'hidden' || style.overflowY === 'hidden' || style.overflowY === 'auto' || style.overflowY === 'scroll') {
        savedStyles.push({ el, overflow: el.style.overflow, overflowY: el.style.overflowY, height: el.style.height, maxHeight: el.style.maxHeight })
        el.style.overflow = 'visible'
        el.style.overflowY = 'visible'
        el.style.height = 'auto'
        el.style.maxHeight = 'none'
      }
      el = el.parentElement
    }
    const savedTargetStyle = { height: target.style.height, maxHeight: target.style.maxHeight, overflow: target.style.overflow, overflowY: target.style.overflowY }
    target.style.height = 'auto'
    target.style.maxHeight = 'none'
    target.style.overflow = 'visible'

    const canvas = await html2canvas(target, {
      useCORS: true,
      scale: 2,
      allowTaint: true,
      logging: false,
      backgroundColor: darkThemeRef.value ? '#1e1e1e' : '#ffffff'
    })
    // 恢复样式
    target.style.height = savedTargetStyle.height
    target.style.maxHeight = savedTargetStyle.maxHeight
    target.style.overflow = savedTargetStyle.overflow
    target.style.overflowY = savedTargetStyle.overflowY
    savedStyles.forEach(({ el: e, overflow, overflowY, height, maxHeight }) => {
      e.style.overflow = overflow
      e.style.overflowY = overflowY
      e.style.height = height
      e.style.maxHeight = maxHeight
    })
    const dataUrl = canvas.toDataURL('image/png')
    const base64 = dataUrl.replace(/^data:image\/png;base64,/, '')
    const safeTime = new Date().toISOString().slice(0, 19).replace(/[:.]/g, '-')
    const result = await SaveImage(`go-stock-agent-${safeTime}`, base64)
    if (result && !result.includes('异常') && !result.includes('无法')) {
      message.success('已导出为 PNG 图片：' + result)
    } else {
      message.info(result || '导出取消')
    }
  } catch (e) {
    message.error('导出图片失败: ' + (e?.message ?? e))
  } finally {
    exportImageKey.value = ''
  }
}

// 会话持久化：sessionId 用于保存/恢复对话
const sessionId = ref('agent-' + Date.now() + '-' + Math.random().toString(36).slice(2, 8))

// Markdown 渲染主题配置
const darkThemeRef = ref(false)
const mdTheme = computed(() => darkThemeRef.value ? 'dark' : 'light')
const codeTheme = computed(() => getCodeTheme(darkThemeRef.value))

function onMdHtmlChanged() {
  nextTick(() => {
    document.querySelectorAll('.msg-markdown').forEach(container => {
      enhanceCodeBlocks(container, {
        collapseThreshold: 8,
        addLineNumbers: true,
        addCopyButton: true,
        addLanguageTag: true
      })
    })
  })
}

function toggleJsonMd(index) {
  jsonMdExpandedMap.value = {
    ...jsonMdExpandedMap.value,
    [index]: !jsonMdExpandedMap.value[index]
  }
}

function toggleReasoning(index) {
  reasoningExpandedMap.value = {
    ...reasoningExpandedMap.value,
    [index]: !reasoningExpandedMap.value[index]
  }
}

// 定义事件处理函数，方便在挂载和卸载时管理
function getStepDotClass(step) {
  if (step.includes('✅')) return 'step-done'
  if (step.includes('🔧')) return 'step-tool'
  if (step.includes('⚡') || step.includes('🧠') || step.includes('📋') || step.includes('🔄')) return 'step-active'
  return ''
}

function startFormatTimer() {
  stopFormatTimer()
  // 缩短格式化间隔，提升流式显示效果
  formatTimer = setInterval(() => {
    const lastItem = chatList.value[0]
    if (lastItem && lastItem.role === 'assistant') {
      if (lastItem.rawContent) {
        const fmt = formatMarkdown(lastItem.rawContent)
        // 强制触发响应式更新
        lastItem.content = fmt.content
        if (fmt.jsonMarkdown) lastItem.jsonMarkdown = fmt.jsonMarkdown
      }
      if (lastItem.rawReasoning) {
        const fmt = formatMarkdown(lastItem.rawReasoning)
        lastItem.reasoning = fmt.content
      }
    }
  }, 300) // 从 1500ms 缩短到 300ms
}

function stopFormatTimer() {
  if (formatTimer) {
    clearInterval(formatTimer)
    formatTimer = null
  }
}

function formatMarkdown(content) {
  if (!content) return { content: '', jsonMarkdown: '' }

  const { content: cleaned, jsonMarkdown } = wrapInlineJson(content)

  let inCodeBlock = false
  const lines = cleaned.split('\n')
  const result = []

  for (let i = 0; i < lines.length; i++) {
    let line = lines[i]
    const trimmed = line.replace(/^[\t ]+/, '')

    if (trimmed.startsWith('```')) {
      inCodeBlock = !inCodeBlock
      if (!inCodeBlock) {
        result.push(trimmed)
        continue
      }
    }

    if (inCodeBlock) {
      result.push(line)
      continue
    }

    if (trimmed !== line && trimmed !== '') {
      line = trimmed
    }

    // 处理 ---##标题 或 ---#标题 等水平线紧贴标题的情况
    const hrHeadingMatch = line.match(/^(---+|\*\*\*+|___+)(#{1,6}\s+.*)$/)
    if (hrHeadingMatch) {
      if (result.length > 0) {
        const prev = result[result.length - 1]
        if (prev !== '' && !isBlockElement(prev.replace(/^[\t ]+/, ''))) {
          result.push('')
        }
      }
      result.push(hrHeadingMatch[1])
      result.push('')
      result.push(hrHeadingMatch[2])
      continue
    }

    // 处理 ---### 或 ---** 等水平线紧贴其他块级元素的情况
    const hrBlockMatch = line.match(/^(---+|\*\*\*+|___+)(#{1,6}\s+|[-*+]\s+|\|\s*|>\s+|```)/)
    if (hrBlockMatch) {
      if (result.length > 0) {
        const prev = result[result.length - 1]
        if (prev !== '' && !isBlockElement(prev.replace(/^[\t ]+/, ''))) {
          result.push('')
        }
      }
      result.push(hrBlockMatch[1])
      result.push('')
      result.push(line.substring(hrBlockMatch[1].length))
      continue
    }

    if (i > 0 && isBlockElement(trimmed)) {
      const prev = result.length > 0 ? result[result.length - 1] : ''
      if (prev !== '' && !isBlockElement(prev.replace(/^[\t ]+/, ''))) {
        result.push('')
      }
    }

    line = splitInlineHeading(line)

    result.push(line)
  }

  return {
    content: result.join('\n'),
    jsonMarkdown
  }
}

function hasMarkdownContent(str) {
  if (!str || typeof str !== 'string') return false
  return /(^|\n)\s*#{1,6}\s/.test(str) ||
    /(^|\n)\s*\|/.test(str) ||
    /(^|\n)\s*---/.test(str) ||
    /(^|\n)\s*[-*+]\s/.test(str) ||
    /(^|\n)\s*>\s/.test(str) ||
    /(^|\n)\s*```/.test(str)
}

function extractMarkdownFromJson(obj) {
  if (typeof obj === 'string') return obj
  if (Array.isArray(obj)) {
    const items = obj.map(item => typeof item === 'string' ? item : JSON.stringify(item, null, 2))
    return items.join('\n\n')
  }
  if (typeof obj === 'object' && obj !== null) {
    for (const key of ['response', 'content', 'text', 'result', 'answer', 'message', 'output']) {
      if (obj[key] != null) {
        const val = obj[key]
        if (typeof val === 'string' && hasMarkdownContent(val)) return val
        if (typeof val === 'object') {
          const extracted = extractMarkdownFromJson(val)
          if (extracted) return extracted
        }
      }
    }
    const values = Object.values(obj).filter(v => typeof v === 'string' && hasMarkdownContent(v))
    if (values.length > 0) return values.join('\n\n')
    const strValues = Object.values(obj).filter(v => typeof v === 'string')
    if (strValues.length > 0) return strValues.join('\n\n')
  }
  return null
}

function wrapInlineJson(content) {
  if (!content) return { content: '', jsonMarkdown: '' }
  const cleaned = []
  const jsonParts = []
  let i = 0
  const len = content.length
  let inCodeBlock = false

  while (i < len) {
    if (content.substring(i, i + 3) === '```') {
      inCodeBlock = !inCodeBlock
      cleaned.push('```')
      i += 3
      continue
    }

    if (inCodeBlock) {
      cleaned.push(content[i])
      i++
      continue
    }

    if (content[i] === '{') {
      const end = findJsonEnd(content, i)
      if (end > i) {
        const jsonStr = content.substring(i, end + 1)
        try {
          const obj = JSON.parse(jsonStr)
          const md = extractMarkdownFromJson(obj)
          if (md) {
            jsonParts.push(md)
          } else {
            cleaned.push('\n\n```json\n' + jsonStr + '\n```\n\n')
          }
          i = end + 1
          continue
        } catch {}
      }
    }
    cleaned.push(content[i])
    i++
  }

  return {
    content: cleaned.join(''),
    jsonMarkdown: jsonParts.join('\n\n---\n\n')
  }
}

function findJsonEnd(content, start) {
  let depth = 0
  let bracketDepth = 0
  let inStr = false
  let escape = false
  for (let i = start; i < content.length; i++) {
    const ch = content[i]
    if (escape) { escape = false; continue }
    if (ch === '\\' && inStr) { escape = true; continue }
    if (ch === '"') { inStr = !inStr; continue }
    if (inStr) continue
    if (ch === '[') bracketDepth++
    else if (ch === ']') bracketDepth--
    else if (ch === '{') depth++
    else if (ch === '}') {
      depth--
      if (depth === 0 && bracketDepth === 0) return i
    }
  }
  return -1
}

function splitInlineHeading(line) {
  // 处理文字---（水平线紧贴在文字后面）
  const hrMatch = line.match(/^(.+?)(---+|\*\*\*+|___+)$/)
  if (hrMatch && hrMatch[1].trim() !== '') {
    return hrMatch[1] + '\n\n' + hrMatch[2]
  }
  // 处理文字##标题（标题紧贴在文字后面）
  const match = line.match(/(#{1,6}\s+\S)/)
  if (!match || match.index === undefined) return line
  const idx = match.index
  if (idx === 0) return line
  const prefix = line.substring(0, idx)
  if (prefix.trim() === '') return line
  return prefix + '\n\n' + line.substring(idx)
}

function isBlockElement(line) {
  if (!line || line.length === 0) return false
  if (line[0] === '#') return true
  if (line.startsWith('- ') || line.startsWith('* ') || line.startsWith('+ ')) return true
  if (line.startsWith('```')) return true
  if (line.startsWith('> ')) return true
  if (line.length >= 2 && line[0] >= '1' && line[0] <= '9' && line[1] === '.') return true
  if (line.startsWith('---') || line.startsWith('***') || line.startsWith('___')) return true
  if (line.startsWith('|')) return true
  return false
}

function parseStepText(text) {
  if (!text) return [text]
  const trimmed = text.trim()
  if (!trimmed.startsWith('{') && !trimmed.startsWith('[')) return [text]
  try {
    const obj = JSON.parse(trimmed)
    if (Array.isArray(obj)) {
      return obj.map((item, i) => `${i + 1}. ${typeof item === 'string' ? item : JSON.stringify(item)}`)
    }
    if (typeof obj === 'object' && obj !== null) {
      const steps = obj.steps || obj.step || obj.plan || obj.items || obj.list
      if (Array.isArray(steps)) {
        return steps.map((item, i) => `${i + 1}. ${typeof item === 'string' ? item : JSON.stringify(item)}`)
      }
      const entries = Object.entries(obj)
      if (entries.length > 0) {
        return entries.map(([k, v]) => `${k}: ${typeof v === 'string' ? v : JSON.stringify(v)}`)
      }
    }
    return [text]
  } catch {
    return [text]
  }
}

// 保存当前会话到后端
function saveCurrentSession() {
  if (chatList.value.length <= 1) return // 只有欢迎消息不保存
  const messages = chatList.value.slice().reverse().map(item => {
    const msg = {
      role: item.role,
      content: item.rawContent || item.content || '',
      time: item.datetime || new Date().toLocaleString(),
    }
    if (item.role === 'assistant' && (item.rawReasoning || item.reasoning)) {
      msg.reasoning = item.rawReasoning || item.reasoning || ''
    }
    return msg
  })
  SaveAiAssistantSession(sessionId.value, messages).catch(() => {})
}

const handleAgentMessage = (data) => {
  // 处理错误消息
  if (data && data['error']) {
    isStreamLoad.value = false;
    loading.value = false;
    stopFormatTimer()
    const lastItemIndex = chatList.value.findIndex(item => item.role === 'assistant')
    if (lastItemIndex !== -1) {
      const chatListClone = [...chatList.value]
      const lastItem = chatListClone[lastItemIndex]
      const updatedItem = { ...lastItem }
      updatedItem.content = '❌ Agent 调用失败：' + String(data['error'] || '未知错误').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      updatedItem.rawContent = updatedItem.content
      chatListClone[lastItemIndex] = updatedItem
      chatList.value = chatListClone
    }
    return
  }

  if(data['role']==="assistant"){
    loading.value = false;
    // 使用更可靠的方式获取和更新最后一个 assistant 消息
    const lastItemIndex = chatList.value.findIndex(item => item.role === 'assistant')
    if (lastItemIndex === -1) return

    // 创建新的对象来触发响应式更新
    const lastItem = chatList.value[lastItemIndex]
    const updatedItem = { ...lastItem }

    if (data['reasoning_content']){
      const rc = data['reasoning_content']
      if (rc.startsWith('[STEP]')) {
        const stepText = rc.replace(/^\[STEP\]/, '').trim()
        if (stepText) {
          if (!updatedItem.steps) updatedItem.steps = []
          const parsed = parseStepText(stepText)
          updatedItem.steps = [...updatedItem.steps, ...parsed]
        }
      } else {
        updatedItem.rawReasoning = (updatedItem.rawReasoning || '') + rc
        updatedItem.reasoning = updatedItem.rawReasoning
      }
    }
    if (data['content']){
      updatedItem.rawContent = (updatedItem.rawContent || '') + data['content']
      updatedItem.content = updatedItem.rawContent
    }
    if(data['tool_calls']){
      for (const tool of data['tool_calls']) {
        console.log(tool.id, tool.type, tool.function.name, tool.function.arguments);
        updatedItem.reasoning = updatedItem.reasoning + "\n```"+tool.function.name+"\n" +
            "参数："+ (tool.function.arguments?tool.function.arguments:"无")+
            "\n```\n";
      }
    }

    // 替换整个对象以触发响应式更新
    chatList.value[lastItemIndex] = updatedItem

    // 流式更新时自动滚动到底部
    nextTick(() => {
      if (chatRef.value) {
        chatRef.value.scrollToBottom({ behavior: 'auto' })
      }
    })
  }
  if(data['response_meta']&&data['response_meta'].finish_reason==="stop"){
    isStreamLoad.value = false;
    loading.value = false;
    stopFormatTimer()
    const lastItemIndex = chatList.value.findIndex(item => item.role === 'assistant')
    if (lastItemIndex !== -1) {
      const lastItem = chatList.value[lastItemIndex]
      const updatedItem = { ...lastItem }
      if (lastItem.rawContent) {
        const fmt = formatMarkdown(lastItem.rawContent)
        updatedItem.content = fmt.content
        if (fmt.jsonMarkdown) updatedItem.jsonMarkdown = fmt.jsonMarkdown
      }
      if (lastItem.rawReasoning) {
        const fmt = formatMarkdown(lastItem.rawReasoning)
        updatedItem.reasoning = fmt.content
      }
      chatList.value[lastItemIndex] = updatedItem
    }
    // 对话结束后自动保存会话
    saveCurrentSession()
  }
  // SSE done 信号也保存
  if(data['content'] === 'agent-DONE'){
    isStreamLoad.value = false
    loading.value = false
    stopFormatTimer()
  }
}

// 保存 EventsOn 返回的 cleanup 函数，用于 Web 模式下正确移除监听器
let cleanupAgentMessage = null

onBeforeUnmount(() => {
  // 优先使用 cleanup 函数移除监听（Web 模式需要，桌面模式 EventsOff 也可用）
  if (cleanupAgentMessage) {
    cleanupAgentMessage()
    cleanupAgentMessage = null
  } else {
    EventsOff("agent-message", handleAgentMessage)
  }
  // 清理流式格式化定时器，防止内存泄漏
  stopFormatTimer()

  // 持久化会话：保存当前对话到后端
  if (chatList.value.length > 0) {
    const messages = chatList.value.slice().reverse().map(item => {
      const msg = {
        role: item.role,
        content: item.rawContent || item.content || '',
        time: item.datetime || new Date().toLocaleString(),
      }
      if (item.role === 'assistant' && (item.rawReasoning || item.reasoning)) {
        msg.reasoning = item.rawReasoning || item.reasoning || ''
      }
      return msg
    })
    SaveAiAssistantSession(sessionId.value, messages).catch(() => {})
  }
})

onBeforeMount(() => {
  // 每次挂载前都重新注册事件监听，保存 cleanup 函数
  cleanupAgentMessage = EventsOn("agent-message", handleAgentMessage)
  GetAiConfigs().then(res=>{
    selectOptions.value = res
    selectValue.value = res[0].ID
  })

  // 加载提示词模板
  GetPromptTemplates('', '').then(res => {
    const list = Array.isArray(res) ? res : []
    sysPromptTemplates.value = list.filter(t => t.type === '模型系统Prompt')
    userPromptTemplates.value = list.filter(t => t.type === '模型用户Prompt')
  }).catch(() => {})

  // 加载已启用技能列表
  GetAllSkills().then(skills => {
    skillList.value = (skills || []).filter(s => s.enable)
  }).catch(() => {})

  // 恢复上一次会话
  GetAiAssistantSession(sessionId.value).then(session => {
    if (session && session.messages && session.messages.length > 0) {
      sessionId.value = session.sessionId || sessionId.value
      const restored = session.messages.map(msg => {
        if (msg.role === 'assistant') {
          return {
            avatar: h(NImage, { src: icon.value, height: '48px', width: '48px'}),
            name: 'Go-Stock AI',
            datetime: msg.time || '',
            content: msg.content || '',
            rawContent: msg.content || '',
            reasoning: msg.reasoning || '',
            rawReasoning: msg.reasoning || '',
            jsonMarkdown: '',
            role: 'assistant',
          }
        }
        return {
          avatar: 'https://tdesign.gtimg.com/site/avatar.jpg',
          name: '宇宙无敌大韭菜',
          datetime: msg.time || '',
          content: msg.content || '',
          role: 'user',
        }
      })
      chatList.value = restored.reverse()
    }
  }).catch(() => {})
})

onMounted(() => {
  //chatRef.value.scrollToBottom();

  GetConfig().then((res) => {
    darkThemeRef.value = !!res.darkTheme
    if (res.darkTheme) {
      document.documentElement.setAttribute("theme-mode", "dark");
    } else {
      document.documentElement.removeAttribute("theme-mode");    }
  })


  GetVersionInfo().then((res) => {
    icon.value = res.icon;
  });

});

// 滚动到底部
const backBottom = () => {
  chatRef.value.scrollToBottom({
    behavior: 'smooth',
  });
};
// 是否显示回到底部按钮
const handleChatScroll = function ({ e }) {
  const scrollTop = e.target.scrollTop;
  isShowToBottom.value = scrollTop < 0;
};
// 清空消息
const clearConfirm = function () {
  chatList.value = [];
};
const handleOperation = function (type, options) {
  // 由 t-chat-action 组件处理，暂无额外逻辑
};
// 倒序渲染
const chatList = ref([
  {
    avatar: h(NImage, { src: icon.value, height: '48px', width: '48px'}),
    name: 'Go-Stock AI',
    datetime: '',
    reasoning: '',
    content: '我是您的AI赋能股票分析助手,您可以问我任何关于股票投资方面的问题。',
    role: 'assistant',
    duration: 10,
  },
]);

const onStop = function () {
  stopFormatTimer()
  AbortChatWithAgent() // 中断后端 Agent 流，停止消耗 API token
  const lastItem = chatList.value[0]
  if (lastItem && lastItem.role === 'assistant') {
    if (lastItem.rawContent) {
      const fmt = formatMarkdown(lastItem.rawContent)
      lastItem.content = fmt.content
      if (fmt.jsonMarkdown) lastItem.jsonMarkdown = fmt.jsonMarkdown
    }
    if (lastItem.rawReasoning) {
      const fmt = formatMarkdown(lastItem.rawReasoning)
      lastItem.reasoning = fmt.content
    }
  }
};

const inputEnter = function () {
  if (isStreamLoad.value) {
    return;
  }
  if (!inputValue.value) return;
  const params = {
    avatar: 'https://tdesign.gtimg.com/site/avatar.jpg',
    name: '宇宙无敌大韭菜',
    datetime: new Date().toDateString(),
    content: inputValue.value,
    role: 'user',
  };
  chatList.value.unshift(params);
  // 空消息占位
  const params2 = {
    avatar:  h(NImage, { src: icon.value, height: '48px', width: '48px'}),
    name: 'Go-Stock AI',
    datetime: new Date().toDateString(),
    content: '',
    rawContent: '',
    reasoning: '',
    rawReasoning: '',
    jsonMarkdown: '',
    role: 'assistant',
  };
  chatList.value.unshift(params2);
  loading.value = true;
  isStreamLoad.value = true;
  startFormatTimer()
  jsonMdExpandedMap.value = { ...jsonMdExpandedMap.value, [0]: true }
  ChatWithAgent(inputValue.value, selectValue.value, sysPromptId.value, memoryMode.value, memoryCount.value, thinkingMode.value, agentMode.value === 'auto' ? '' : agentMode.value, selectedSkillIds.value)
};
</script>
<style lang="less">
/* 应用滚动条样式 */
::-webkit-scrollbar-thumb {
  background-color: var(--td-scrollbar-color);
}
::-webkit-scrollbar-thumb:horizontal:hover {
  background-color: var(--td-scrollbar-hover-color);
}
::-webkit-scrollbar-track {
  background-color: var(--td-scroll-track-color);
}
.chat-box {
  position: relative;
  height: 100%;
  margin: 5px 10px 5px 10px;
  text-align: left;
  .bottomBtn {
    position: absolute;
    left: 50%;
    margin-left: -20px;
    bottom: 210px;
    padding: 0;
    border: 0;
    width: 40px;
    height: 40px;
    border-radius: 50%;
    box-shadow: 0px 8px 10px -5px rgba(0, 0, 0, 0.08), 0px 16px 24px 2px rgba(0, 0, 0, 0.04),
    0px 6px 30px 5px rgba(0, 0, 0, 0.05);
  }
  .to-bottom {
    width: 40px;
    height: 40px;
    border: 1px solid #dcdcdc;
    box-sizing: border-box;
    background: var(--td-bg-color-container);
    border-radius: 50%;
    font-size: 24px;
    line-height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    .t-icon {
      font-size: 24px;
    }
  }
}

.model-select {
  display: flex;
  align-items: center;
  .t-select {
    width: 112px;
    height: 32px;
    margin-right: 8px;
    .t-input {
      border-radius: 32px;
      padding: 0 15px;
    }
  }
  .check-box {
    width: 112px;
    height: 32px;
    border-radius: 32px;
    border: 0;
    background: #e7e7e7;
    color: rgba(0, 0, 0, 0.9);
    box-sizing: border-box;
    flex: 0 0 auto;
    .t-button__text {
      display: flex;
      align-items: center;
      justify-content: center;
      span {
        margin-left: 4px;
      }
    }
  }
  .check-box.is-active {
    border: 1px solid #d9e1ff;
    background: #f2f3ff;
    color: var(--td-brand-color);
  }
}


.chat-sender {
  .btn {
    color: var(--td-text-color-disabled);
    border: none;
    &:hover {
      color: var(--td-brand-color-hover);
      border: none;
      background: none;
    }
  }
  .btn.t-button {
    height: var(--td-comp-size-m);
    padding: 0;
  }
  .model-select {
    display: flex;
    align-items: center;
    .t-select {
      width: 112px;
      height: var(--td-comp-size-m);
      margin-right: var(--td-comp-margin-s);
      .t-input {
        border-radius: 32px;
        padding: 0 15px;
      }
      .t-input.t-is-focused {
        box-shadow: none;
      }
    }
    .check-box {
      width: 112px;
      height: var(--td-comp-size-m);
      border-radius: 32px;
      border: 0;
      background: var(--td-bg-color-component);
      color: var(--td-text-color-primary);
      box-sizing: border-box;
      flex: 0 0 auto;
      .t-button__text {
        display: flex;
        align-items: center;
        justify-content: center;
        span {
          margin-left: var(--td-comp-margin-xs);
        }
      }
    }
    .check-box.is-active {
      border: 1px solid var(--td-brand-color-focus);
      background: var(--td-brand-color-light);
      color: var(--td-text-color-brand);
    }
  }
}

.agent-steps {
  margin-bottom: 8px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
  overflow: hidden;
  background: var(--td-bg-color-container-hover);
}
.agent-steps-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  font-size: 13px;
  font-weight: 500;
  color: var(--td-text-color-secondary);
  background: linear-gradient(135deg, rgba(56, 173, 169, 0.06) 0%, rgba(46, 139, 87, 0.06) 100%);
  border-bottom: 1px solid var(--td-component-border);
}
.agent-steps-badge {
  font-size: 11px;
  background: var(--td-brand-color);
  color: #fff;
  border-radius: 10px;
  padding: 0 6px;
  line-height: 18px;
  min-width: 18px;
  text-align: center;
}
.agent-steps-list {
  padding: 8px 10px 8px 14px;
  max-height: 250px;
  overflow-y: auto;
}
.agent-step-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 3px 0;
  position: relative;
  font-size: 12px;
  color: var(--td-text-color-secondary);
  line-height: 1.5;
}
.agent-step-item:not(:last-child)::before {
  content: '';
  position: absolute;
  left: 4px;
  top: 16px;
  bottom: -3px;
  width: 1px;
  background: var(--td-component-border);
}
.agent-step-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--td-text-color-disabled);
  flex-shrink: 0;
  margin-top: 4px;
  position: relative;
  z-index: 1;
}
.agent-step-dot.step-active {
  background: #e6a23c;
  box-shadow: 0 0 4px rgba(230, 162, 60, 0.4);
}
.agent-step-dot.step-tool {
  background: #409eff;
  box-shadow: 0 0 4px rgba(64, 158, 255, 0.4);
}
.agent-step-dot.step-done {
  background: #67c23a;
  box-shadow: 0 0 4px rgba(103, 194, 58, 0.4);
}
.agent-step-text {
  flex: 1;
  min-width: 0;
  word-break: break-all;
  text-align: left;
}

.agent-json-md {
  margin-bottom: 8px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
  overflow: hidden;
  background: var(--td-bg-color-container-hover);
}
.agent-json-md-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  cursor: pointer;
  user-select: none;
  font-size: 13px;
  font-weight: 500;
  color: var(--td-text-color-secondary);
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.06) 0%, rgba(5, 150, 105, 0.06) 100%);
  border-bottom: 1px solid var(--td-component-border);
  transition: background 0.2s;
}
.agent-json-md-header:hover {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.12) 0%, rgba(5, 150, 105, 0.12) 100%);
}
.agent-json-md-arrow {
  transition: transform 0.2s;
  flex-shrink: 0;
}
.agent-json-md-arrow-expanded {
  transform: rotate(90deg);
}
.agent-json-md-title {
  font-size: 13px;
  font-weight: 500;
}
.agent-json-md-content {
  padding: 10px;
  max-height: 500px;
  overflow-y: auto;
  text-align: left;
}

/* 思考过程样式 */
.reasoning-wrapper {
  margin-bottom: 8px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
  overflow: hidden;
  background: var(--td-bg-color-container-hover);
}

.reasoning-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  cursor: pointer;
  user-select: none;
  font-size: 13px;
  background: linear-gradient(135deg, rgba(147, 51, 234, 0.06) 0%, rgba(79, 70, 229, 0.06) 100%);
  border-bottom: 1px solid var(--td-component-border);
  transition: background 0.2s;
}

.reasoning-header:hover {
  background: linear-gradient(135deg, rgba(147, 51, 234, 0.12) 0%, rgba(79, 70, 229, 0.12) 100%);
}

.reasoning-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
}

.reasoning-indicator.thinking {
  animation: pulse 1.5s ease-in-out infinite;
}

.thinking-icon {
  color: var(--td-text-color-secondary);
  flex-shrink: 0;
}

.reasoning-indicator.thinking .thinking-icon {
  color: #8b5cf6;
  animation: spin 1s linear infinite;
}

.thinking-text {
  font-size: 13px;
  font-weight: 500;
  color: var(--td-text-color-secondary);
}

.reasoning-indicator.thinking .thinking-text {
  color: #8b5cf6;
}

/* 思考中的点点动画 */
.thinking-dots {
  display: flex;
  gap: 3px;
  margin-left: 4px;
}

.thinking-dots .dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: #8b5cf6;
  animation: dotPulse 1.4s ease-in-out infinite;
}

.thinking-dots .dot:nth-child(1) {
  animation-delay: 0s;
}

.thinking-dots .dot:nth-child(2) {
  animation-delay: 0.2s;
}

.thinking-dots .dot:nth-child(3) {
  animation-delay: 0.4s;
}

.reasoning-arrow {
  color: var(--td-text-color-secondary);
  transition: transform 0.2s;
  flex-shrink: 0;
}

.reasoning-arrow.expanded {
  transform: rotate(180deg);
}

.reasoning-content {
  padding: 10px 12px;
  max-height: 400px;
  overflow-y: auto;
  text-align: left;
  font-size: 13px;
  color: var(--td-text-color-secondary);
}

/* MdPreview 样式穿透 */
.reasoning-content :deep(.md-editor-preview-wrapper),
.agent-json-md-content :deep(.md-editor-preview-wrapper) {
  padding: 0;
}

.reasoning-content :deep(.md-editor-preview),
.agent-json-md-content :deep(.md-editor-preview) {
  font-size: 13px;
  color: var(--td-text-color-secondary);
  background: transparent !important;
}

/* 代码块增强全局样式 */
.msg-markdown :deep(.md-editor-code-block) {
  position: relative;
}

.msg-markdown :deep(.code-lang-tag) {
  position: absolute;
  top: 8px;
  left: 12px;
  z-index: 2;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 500;
  color: #666;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 4px;
  opacity: 0.8;
  pointer-events: none;
}

.msg-markdown :deep(.code-copy-btn) {
  position: absolute;
  top: 8px;
  right: 12px;
  z-index: 2;
  padding: 4px 8px;
  font-size: 12px;
  color: #666;
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid #e0e0e0;
  border-radius: 4px;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.2s, background 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.msg-markdown :deep(.md-editor-code-block:hover .code-copy-btn) {
  opacity: 1;
}

.msg-markdown :deep(.code-copy-btn:hover) {
  background: #3b82f6;
  border-color: #3b82f6;
  color: #fff;
}

.msg-markdown :deep(.code-copy-btn.copy-success) {
  color: #67c23a;
  border-color: #67c23a;
}

.msg-markdown :deep(.code-copy-btn.copy-error) {
  color: #f56c6c;
  border-color: #f56c6c;
}

.msg-markdown :deep(.code-collapse-btn) {
  position: absolute;
  bottom: 8px;
  right: 12px;
  z-index: 2;
  padding: 2px 8px;
  font-size: 11px;
  color: #666;
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid #e0e0e0;
  border-radius: 4px;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.2s;
}

.msg-markdown :deep(.md-editor-code-block:hover .code-collapse-btn) {
  opacity: 1;
}

.msg-markdown :deep(.md-editor-code-block.code-collapsed pre) {
  max-height: 120px;
  overflow: hidden;
}

.msg-markdown :deep(.md-editor-code-block.code-collapsed::after) {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 50px;
  background: linear-gradient(transparent, #fff);
  pointer-events: none;
}

/* 动画定义 */
@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes dotPulse {
  0%, 80%, 100% {
    transform: scale(0.6);
    opacity: 0.5;
  }
  40% {
    transform: scale(1);
    opacity: 1;
  }
}

</style>