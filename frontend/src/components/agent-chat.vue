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
        <t-chat-reasoning v-if="item.role === 'assistant'"  expand-icon-placement="right">
          <t-chat-loading v-if="isStreamLoad" text="思考中..." />
          <t-chat-content v-if="item.reasoning.length > 0" :content="item.reasoning" />
        </t-chat-reasoning>
        <div v-if="item.role === 'assistant' && item.jsonMarkdown" class="agent-json-md">
          <div class="agent-json-md-header" @click="toggleJsonMd(index)">
            <svg :class="['agent-json-md-arrow', { 'agent-json-md-arrow-expanded': jsonMdExpandedMap[index] }]" viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><path d="M9 6l6 6-6 6z"/></svg>
            <span class="agent-json-md-title">📊 分析报告</span>
          </div>
          <div v-show="jsonMdExpandedMap[index]" class="agent-json-md-content">
            <t-chat-content :content="item.jsonMarkdown" />
          </div>
        </div>
        <t-chat-content v-if="item.content.length > 0" :content="item.content" />
        <div v-if="item.role === 'user' && item.images && item.images.length" class="msg-image-grid">
          <NImage
            v-for="(img, i) in item.images"
            :key="'msg-img-' + index + '-' + i"
            :src="img"
            class="msg-image-thumb"
            object-fit="cover"
          />
        </div>
      </template>
      <template #actions="{ item, index }">
        <t-chat-action
            :content="item.content"
            :operation-btn="['copy']"
            @operation="handleOperation"
        />
        <span v-if="item.role === 'assistant' && !item.feedback" class="feedback-btns">
          <t-button size="small" variant="text" title="这个回答有用" @click="submitFeedback(item, 1)">👍</t-button>
          <t-button size="small" variant="text" title="这个回答没用" @click="openFeedbackDialog(item)">👎</t-button>
        </span>
        <span v-else-if="item.role === 'assistant' && item.feedback" class="feedback-done">{{ item.feedback === 1 ? '👍' : '👎' }}</span>
      </template>
      <template #footer>
<!--        <t-chat-input :stop-disabled="isStreamLoad" @send="inputEnter" @stop="onStop"> </t-chat-input>-->
          <div v-if="pendingImages.length" class="pending-images">
            <div v-for="(img, i) in pendingImages" :key="i" class="pending-image-item">
              <NImage :src="img.preview" class="pending-image-thumb" object-fit="cover" />
              <div v-if="img.uploading" class="pending-image-uploading" title="正在上传图床">
                <NSpin size="small" />
              </div>
              <NButton
                quaternary
                circle
                size="tiny"
                class="pending-image-remove"
                title="移除图片"
                @click="removePendingImage(i)"
              >
                <template #icon>
                  <NIcon :component="CloseOutline" size="14" />
                </template>
              </NButton>
            </div>
          </div>
          <t-chat-sender
              ref="chatSenderRef"
              v-model="inputValue"
              class="chat-sender"
              :textarea-props="{
                placeholder: '请输入消息...',
                onPaste: onPasteImage,
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
              <NFlex>
                <NPopover
                    v-if="currentConfigSupportsVision"
                    trigger="click"
                    placement="top-start"
                    :show-arrow="true"
                    to="body"
                >
                  <template #trigger>
                    <NButton
                        quaternary
                        size="small"
                        class="chat-footer-img-btn"
                        title="添加图片（支持上传 / 链接 / 粘贴）"
                        :disabled="isStreamLoad"
                    >
                      <template #icon>
                        <NIcon :component="ImageOutline" />
                      </template>
                    </NButton>
                  </template>
                  <div class="image-add-popover">
                    <NButton size="small" dashed block @click="triggerImageSelect">
                      上传本地图片
                    </NButton>
                    <div class="image-url-row">
                      <NInput
                          v-model:value="imageUrlInput"
                          size="small"
                          placeholder="或输入图片链接 https://..."
                          clearable
                      />
                      <NButton size="small" type="primary" ghost @click="addImageUrl">
                        添加
                      </NButton>
                    </div>
                    <div class="image-add-tip">支持 JPEG/PNG/GIF/WebP，单张 ≤ 8MB，最多 {{ MAX_IMAGE_COUNT }} 张；也可直接在输入框粘贴图片。本地图默认自动托管到免费图床转为外链发送（图床为公共免费服务，请勿上传敏感图片），失败时回退 base64 直传</div>
                  </div>
                </NPopover>
                <NButton
                    v-else
                    quaternary
                    size="small"
                    class="chat-footer-img-btn"
                    title="当前模型未开启视觉理解，可在「AI模型服务配置」中开启"
                    disabled
                >
                  <template #icon>
                    <NIcon :component="ImageOutline" />
                  </template>
                </NButton>
                <NSelect
                    v-model:value="selectValue"
                    :options="selectOptions"
                    label-field="name" value-field="ID"
                    size="tiny"
                    style="width: 200px;"
                />
                <NSelect
                    v-model:value="agentMode"
                    :options="agentModeOptions"
                    size="tiny"
                    style="width: 120px;"
                />
              </NFlex>
            </template>
          </t-chat-sender>

          <input
              ref="imageFileInputRef"
              type="file"
              accept="image/jpeg,image/png,image/gif,image/webp"
              multiple
              style="display: none"
              @change="onImageFilesSelected"
          />
      </template>
    </t-chat>
    <t-button v-show="isShowToBottom" variant="text" class="bottomBtn" @click="backBottom">
      <div class="to-bottom">
        <ArrowDownIcon />
      </div>
    </t-button>

    <!-- 👎 反馈理由弹窗：采集纠正原因，供画像学习"需规避项/偏好格式" -->
    <n-modal
      v-model:show="feedbackDialogShow"
      preset="dialog"
      title="这个回答哪里不行？"
      positive-text="提交反馈"
      negative-text="跳过"
      @positive-click="confirmFeedbackSubmit"
      @negative-click="submitFeedbackSkip"
      @close="submitFeedbackSkip"
    >
      <div style="display:flex; flex-direction:column; gap:8px; text-align:left;">
        <n-select
          v-model:value="feedbackReasonPreset"
          :options="feedbackReasonOptions"
          placeholder="选择主要问题（可选）"
          clearable
          size="small"
        />
        <n-input
          v-model:value="feedbackReasonText"
          type="textarea"
          placeholder="补充说明（可选，例如：没结合我的持仓成本）"
          :rows="2"
          maxlength="200"
          show-count
        />
      </div>
    </n-modal>
  </div>
</template>
<script setup lang="ts">
import {ref, reactive, computed, watch, onMounted, h, onBeforeUnmount, onBeforeMount} from 'vue';
import {ArrowDownIcon, CheckCircleIcon, SystemSumIcon} from 'tdesign-icons-vue-next';
import {ImageOutline, CloseOutline} from '@vicons/ionicons5';
const fetchCancel = ref(null);
const loading = ref(false);

const inputValue = ref('');
// 流式数据加载中
const isStreamLoad = ref(false);
let formatTimer = null

const chatRef = ref(null);
const isShowToBottom = ref(false);

const icon = ref('https://raw.githubusercontent.com/ArvinLovegood/go-stock/master/build/appicon.png');
import {darkTheme, NFlex, NImage,NSelect, NButton, NIcon, NInput, NPopover, NSpin} from "naive-ui";
import {ChatWithAgent, GetAiConfigs, GetConfig, GetSponsorInfo, GetVersionInfo, SubmitAgentFeedback, UploadImageToImageBed} from "../../wailsjs/go/main/App";
import {models} from '../../wailsjs/go/models';
import {EventsOff, EventsOn} from '../../wailsjs/runtime'
import 'tdesign-vue-next/es/style/index.css';


const allowToolTip = ref(true);
const chatSenderRef = ref(null);

// ===== 视觉理解（图片输入）：仅所选 AI 配置开启「视觉理解」时可用 =====
// 默认外部 URL 图片模式：本地图先托管到免费图床（img.scdn.io）转成外链 URL 再发送，
// 请求体小且多轮对话不膨胀；图床上传失败时自动回退 base64 直传。
const MAX_IMAGE_COUNT = 10
const MAX_IMAGE_SIZE = 8 * 1024 * 1024 // 单张 8MB
const pendingImages = ref([]) // 待发送图片：[{ url: 发送用 URL, preview: 预览图, uploading: 是否上传图床中 }]
const imageUrlInput = ref('')
const imageFileInputRef = ref(null)

function warn(msg) {
  if (window.$message) window.$message.warning(msg)
}

function triggerImageSelect() {
  imageFileInputRef.value?.click()
}

function onImageFilesSelected(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = '' // 允许重复选择同一文件
  files.forEach(f => addImageFromLocal(f))
}

function readFileAsDataURL(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result)
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

async function addImageFromLocal(file) {
  if (!file.type || !file.type.startsWith('image/')) {
    warn('仅支持图片文件')
    return
  }
  if (file.size > MAX_IMAGE_SIZE) {
    warn(`图片 ${file.name} 超过 8MB 限制`)
    return
  }
  if (pendingImages.value.length >= MAX_IMAGE_COUNT) {
    warn(`最多添加 ${MAX_IMAGE_COUNT} 张图片`)
    return
  }
  let dataUrl
  try {
    dataUrl = await readFileAsDataURL(file)
  } catch (_) {
    warn('图片读取失败，请重试')
    return
  }
  if (typeof dataUrl !== 'string' || !dataUrl.startsWith('data:image/')) {
    warn('图片读取失败，请重试')
    return
  }
  // 先以本地 base64 作为预览占位，异步上传图床换取外链（reactive 保证上传状态变化触发视图更新）
  const item = reactive({ url: '', preview: dataUrl, uploading: true })
  pendingImages.value.push(item)
  try {
    const url = await UploadImageToImageBed(dataUrl, file.name || 'image.png')
    item.url = url
    item.preview = url // 外链可直接预览
    item.uploading = false
  } catch (e) {
    // 图床不可用时回退 base64 直传（OpenAI 兼容 image_url 同样支持 data URL）
    item.url = dataUrl
    item.uploading = false
    warn('图床上传失败，本图将改用 base64 直传（' + (e?.message ?? e) + '）')
  }
}

function addImageUrl() {
  const url = imageUrlInput.value.trim()
  if (!url) return
  if (!/^https?:\/\//i.test(url)) {
    warn('请输入 http(s) 图片链接')
    return
  }
  if (pendingImages.value.length >= MAX_IMAGE_COUNT) {
    warn(`最多添加 ${MAX_IMAGE_COUNT} 张图片`)
    return
  }
  pendingImages.value.push({ url, preview: url, uploading: false })
  imageUrlInput.value = ''
}

function removePendingImage(index) {
  pendingImages.value.splice(index, 1)
}

// 粘贴图片：直接加入待发送列表（输入框粘贴文本不受影响）
function onPasteImage(e) {
  if (!currentConfigSupportsVision.value) return
  const files = Array.from(e.clipboardData?.files || [])
  const images = files.filter(f => f.type && f.type.startsWith('image/'))
  if (images.length) {
    e.preventDefault()
    images.forEach(f => addImageFromLocal(f))
  }
}
const selectOptions = ref([]);
const selectValue = ref("default");
const agentMode = ref('deepagents')
const agentModeOptions = [
  { label: '🤖 自动', value: 'auto' },
  { label: '⚡ 快速', value: 'react' },
  { label: '🧠 规划', value: 'plan_execute' },
  { label: '🔬 DeepAgents', value: 'deepagents' },
]
const jsonMdExpandedMap = ref({})

// ===== 视觉理解：所选模型是否开启（依赖 selectOptions/selectValue，须在其后声明避免 TDZ）=====
const currentConfigSupportsVision = computed(() => {
  const cfg = (selectOptions.value || []).find(c => Number(c.ID) === Number(selectValue.value))
  return !!cfg?.supportVision
})

const hasUploadingImage = computed(() => pendingImages.value.some(i => i.uploading))

// 切换到不支持视觉的模型时，清空待发送图片避免误发
watch(currentConfigSupportsVision, (v) => {
  if (!v && pendingImages.value.length) {
    pendingImages.value = []
    warn('已切换到未开启视觉理解的模型，待发送图片已清空')
  }
})

function toggleJsonMd(index) {
  jsonMdExpandedMap.value = {
    ...jsonMdExpandedMap.value,
    [index]: !jsonMdExpandedMap.value[index]
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
  formatTimer = setInterval(() => {
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
  }, 1500)
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
  const match = line.match(/(#{1,6}\s+\S)/)
  if (!match) return line
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

const handleAgentMessage = (data) => {
  console.log(data)
  if(data['role']==="assistant"){
    loading.value = false;
    const lastItem = chatList.value[0];
    if (data['reasoning_content']){
      const rc = data['reasoning_content']
      if (rc.startsWith('[STEP]')) {
        const stepText = rc.replace(/^\[STEP\]/, '').trim()
        if (stepText) {
          if (!lastItem.steps) lastItem.steps = []
          const parsed = parseStepText(stepText)
          lastItem.steps.push(...parsed)
        }
      } else {
        lastItem.rawReasoning = (lastItem.rawReasoning || '') + rc
        lastItem.reasoning = lastItem.rawReasoning
      }
    }
    if (data['content']){
      lastItem.rawContent = (lastItem.rawContent || '') + data['content']
      lastItem.content = lastItem.rawContent
    }
    if(data['tool_calls']){
      for (const tool of  data['tool_calls']) {
          console.log(tool.id, tool.type, tool.function.name, tool.function.arguments);
        lastItem.reasoning += "\n```"+tool.function.name+"\n" +
            "参数："+ (tool.function.arguments?tool.function.arguments:"无")+
            "\n```\n";
      }
    }
  }
  if(data['response_meta']&&data['response_meta'].finish_reason==="stop"){
    isStreamLoad.value = false;
    loading.value = false;
    stopFormatTimer()
    const lastItem = chatList.value[0];
    if (lastItem) {
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
  }
}

onBeforeUnmount(() => {
  EventsOff("agent-message", handleAgentMessage)
})

onBeforeMount(() => {
  // 每次挂载前都重新注册事件监听
  EventsOn("agent-message", handleAgentMessage)
  GetAiConfigs().then(res=>{
    console.log(res)
    // 视觉理解模型加 [视觉] 标记（与桌面端一致）
    selectOptions.value = (res || []).map(c => ({...c, name: (c.name ?? '') + (c.supportVision ? ' [视觉]' : '')}))
    if (res && res.length > 0) {
      selectValue.value = res[0].ID
    }
  })
})

onMounted(() => {
  //chatRef.value.scrollToBottom();

  GetConfig().then((res) => {
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
  console.log('handleOperation', type, options);
};

// 提交对某条回答的反馈（👍 直接提交；👎 先弹理由框，可跳过）
const submitFeedback = function (item, rating, reason = '') {
  if (!item) return;
  const fb = models.AgentFeedback.createFrom({
    sessionId: '',
    question: item.question || '',
    response: item.rawContent || item.content || '',
    rating: rating,
    reason: reason,
    mode: agentMode.value === 'auto' ? '' : agentMode.value,
  });
  SubmitAgentFeedback(fb).then(() => {
    item.feedback = rating;
    window.$message && window.$message.success(rating === 1 ? '感谢反馈，我会继续优化' : '已收到，我会改进');
  }).catch((e) => {
    console.error('submit feedback error', e);
  });
};

// ---- 👎 理由弹窗 ----
const feedbackDialogShow = ref(false);
const feedbackReasonPreset = ref(null);
const feedbackReasonText = ref('');
const feedbackTargetItem = ref(null);
// 预设理由选项：与 user-profile.vue 的 classifyFeedbackReason 分类对应，便于画像学习归类
const feedbackReasonOptions = [
  {label: '数据不准 / 过时', value: '数据不准'},
  {label: '逻辑推理有误', value: '逻辑有误'},
  {label: '太啰嗦 / 格式不佳', value: '太啰嗦'},
  {label: '风险提示不合我的风格', value: '风险偏好不符'},
  {label: '没结合我的持仓 / 关注', value: '没结合我的持仓'},
];

function openFeedbackDialog(item) {
  if (!item) return;
  feedbackTargetItem.value = item;
  feedbackReasonPreset.value = null;
  feedbackReasonText.value = '';
  feedbackDialogShow.value = true;
}

// 拼接预设 + 自由文本
function buildFeedbackReason() {
  return [feedbackReasonPreset.value, feedbackReasonText.value.trim()]
    .filter(Boolean).join('；')
}

function confirmFeedbackSubmit() {
  const item = feedbackTargetItem.value;
  feedbackDialogShow.value = false;
  if (item) submitFeedback(item, -1, buildFeedbackReason());
}

// 跳过：不填理由直接提交 👎
function submitFeedbackSkip() {
  const item = feedbackTargetItem.value;
  feedbackDialogShow.value = false;
  if (item) submitFeedback(item, -1, '');
}
// 倒序渲染
const chatList = ref([
  // {
  //   content: `模型由<span>hunyuan</span>变为<span>GPT4</span>`,
  //   role: 'model-change',
  //   reasoning: '',
  // },
  {
    avatar: h(NImage, { src: icon.value, height: '48px', width: '48px'}),
    name: 'Go-Stock AI',
    datetime: '',
    reasoning: '',
    content: '我是您的AI赋能股票分析助手,您可以问我任何关于股票投资方面的问题。',
    role: 'assistant',
    duration: 10,
  },
  {
    avatar: 'https://tdesign.gtimg.com/site/avatar.jpg',
    name: '宇宙无敌大韭菜',
    datetime: '',
    content: '介绍下自己？',
    role: 'user',
    reasoning: '',
  },
]);

const onStop = function () {
  if (fetchCancel.value) {
    fetchCancel.value.controller.close();
    loading.value = false;
    isStreamLoad.value = false;
  }
  stopFormatTimer()
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
  if (hasUploadingImage.value) {
    warn('图片正在上传图床，请稍候再发送')
    return
  }
  let text = inputValue.value ? inputValue.value.trim() : '';
  const images = pendingImages.value.map(i => i.url).filter(Boolean);
  if (!text && images.length === 0) return;
  // 视觉图片仅发送给开启了「视觉理解」的模型
  if (images.length > 0 && !currentConfigSupportsVision.value) {
    warn('当前模型未开启视觉理解，无法发送图片。请在「AI模型服务配置」中开启该选项，或删除图片后重试')
    return
  }
  // 纯图片提问时补充默认文本
  if (!text) text = '请分析这些图片';
  const params = {
    avatar: 'https://tdesign.gtimg.com/site/avatar.jpg',
    name: '宇宙无敌大韭菜',
    datetime: new Date().toDateString(),
    content: text,
    role: 'user',
  };
  if (images.length) params.images = images;
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
    question: text,
    feedback: 0,
    role: 'assistant',
  };
  chatList.value.unshift(params2);
  loading.value = true;
  isStreamLoad.value = true;
  startFormatTimer()
  jsonMdExpandedMap.value = { ...jsonMdExpandedMap.value, [0]: true }
  inputValue.value = '';
  pendingImages.value = [];
  ChatWithAgent(text,selectValue.value,0,false,0,false,agentMode.value === 'auto' ? '' : agentMode.value,'','', images.length ? JSON.stringify(images) : '')
};
</script>
<style lang="less">
/* ===== 视觉理解（图片输入）样式，与桌面端 FloatingAgentAssistant 对齐 ===== */
/* 图片按钮 */
.chat-footer-img-btn {
  flex-shrink: 0;
}
/* 待发送图片预览条 */
.pending-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 8px 8px 8px;
}
.pending-image-item {
  position: relative;
}
.pending-image-thumb {
  width: 64px;
  height: 64px;
  border-radius: 8px;
  overflow: hidden;
}
.pending-image-uploading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.35);
  border-radius: 8px;
}
.pending-image-remove {
  position: absolute;
  top: -7px;
  right: -7px;
  z-index: 1;
}
/* 图片添加弹层 */
.image-add-popover {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 260px;
}
.image-url-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
.image-url-row .n-input {
  flex: 1;
}
.image-add-tip {
  font-size: 12px;
  color: var(--n-text-color-3);
  line-height: 1.5;
}
/* 用户消息气泡内图片 */
.msg-image-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-start;
  margin-top: 6px;
  margin-bottom: 6px;
}
.msg-image-thumb {
  width: 120px;
  height: 120px;
  border-radius: 8px;
  overflow: hidden;
}
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
  .feedback-btns {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    margin-left: 4px;
    vertical-align: middle;
  }
  .feedback-done {
    margin-left: 6px;
    opacity: 0.7;
    vertical-align: middle;
  }
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

</style>