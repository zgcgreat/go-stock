<template>
  <div v-if="authLoading" class="auth-loading">
    <NSpin size="large" />
    <div class="auth-loading-text">正在验证身份...</div>
  </div>
  <div v-else-if="!isAuthenticated" class="auth-required">
    <div class="auth-title">需要登录</div>
    <p class="auth-desc">请先登录以使用 go-stock AI 助手</p>
    <NButton type="primary" @click="goToLogin">前往登录</NButton>
  </div>
  <div v-else-if="!vipGateOk" class="vip-gate vip-gate-denied">
    <div class="vip-gate-title">权限不足</div>
    <p class="vip-gate-desc">{{ vipGateMessage }}</p>
    <p class="vip-gate-hint">请联系管理员将您的账户升级为VIP或更高权限。</p>
  </div>
  <div v-else class="page">
        <div class="header">
          <div class="title">go-stock AI 助手（Web）</div>
          <div class="motto">「{{ currentMotto }}」</div>
          <div class="toolbar">
            <NButton size="small" type="primary" class="new-chat-btn" @click="startNewChat">
              新会话
            </NButton>
            <NButton
              v-if="messages.length > DEFAULT_VISIBLE_COUNT"
              quaternary
              size="small"
              class="history-toggle-btn"
              @click="showMoreHistory"
            >
              {{ expandAll ? '收起' : '展开更多历史' }}{{ expandAll ? '' : '（共 ' + hiddenCount + ' 条）' }}
            </NButton>
          </div>
        </div>

        <NCard size="small" :bordered="false" class="chat-card">
          <NScrollbar ref="scrollRef" class="chat-scroll">
            <div class="msg-list">
              <div v-for="(m, idx) in displayedMessages" :key="fromIndex + idx" :class="['msg', m.role]">
                <div v-if="m.role === 'assistant'" class="msg-avatar ai-avatar">
                  <NAvatar round size="small" color="#6d5dfc">
                    <NIcon :component="SparklesOutline" />
                  </NAvatar>
                </div>
                <div class="bubble" :id="'bubble-' + (fromIndex + idx)">
                  <template v-if="m.role === 'assistant'">
                    <div v-if="needBubbleCollapse(m) && !isBubbleExpanded(fromIndex + idx)" class="msg-content-collapsed">
                      {{ getBubblePreview(m) }}
                    </div>
                    <div v-else>
                        <details v-if="m.reasoning" class="reasoning-details">
                        <summary class="reasoning-summary">
                          <svg class="reasoning-icon" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2a7 7 0 0 1 7 7c0 2.38-1.19 4.47-3 5.74V17a1 1 0 0 1-1 1h-6a1 1 0 0 1-1-1v-2.26C6.19 13.47 5 11.38 5 9a7 7 0 0 1 7-7z"/><line x1="9" y1="21" x2="15" y2="21"/></svg>
                          思考过程
                        </summary>
                        <MdPreview
                          :model-value="m.reasoning"
                          :theme="'light'"
                          :code-theme="codeTheme"
                          :editor-id="'reasoning-' + (fromIndex + idx)"
                          :style="{ textAlign: 'left' }"
                          class="msg-markdown reasoning-md"
                        />
                      </details>
                      <details v-if="m.jsonMarkdown" class="json-md-details">
                        <summary class="json-md-summary">
                          <svg class="reasoning-icon" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
                          JSON 分析报告
                        </summary>
                        <MdPreview
                          :model-value="m.jsonMarkdown"
                          :theme="'light'"
                          :code-theme="codeTheme"
                          :editor-id="'json-md-' + (fromIndex + idx)"
                          :style="{ textAlign: 'left' }"
                          class="msg-markdown"
                          @onHtmlChanged="onMdHtmlChanged"
                        />
                      </details>
                      <MdPreview
                        v-if="m.content"
                        :model-value="m.content"
                        :theme="'light'"
                        :code-theme="codeTheme"
                        :editor-id="'msg-' + (fromIndex + idx)"
                        :style="{ textAlign: 'left' }"
                        class="msg-markdown"
                        @onHtmlChanged="onMdHtmlChanged"
                      />
                    </div>
                  </template>
                  <div v-else-if="m.content" class="user-text">{{ m.content }}</div>
                  <div class="meta">
                    <span>{{ m.time }}</span>
                    <span v-if="m.role === 'assistant'" class="meta-actions">
                      <NButton
                        size="tiny"
                        quaternary
                        class="msg-img-btn"
                        :loading="saveImageLoading === fromIndex + idx"
                        @click="saveBubbleAsImage(fromIndex + idx)"
                        :disabled="!m.content"
                      >
                        保存为图片
                      </NButton>
                      <NButton
                        v-if="needBubbleCollapse(m)"
                        quaternary
                        size="tiny"
                        class="msg-expand-btn"
                        @click="toggleBubble(fromIndex + idx)"
                      >
                        {{ isBubbleExpanded(fromIndex + idx) ? '收起' : '展开' }}
                      </NButton>
                      <NButton size="tiny" quaternary @click="copyText(m.content)">复制</NButton>
                      <NButton size="tiny" quaternary :loading="shareLoading" @click="shareOne(m.content)">分享</NButton>
                    </span>
                  </div>
                </div>
                <div v-if="m.role === 'user'" class="msg-avatar user-avatar">
                  <NAvatar round size="small" color="#3b82f6">
                    <NIcon :component="PersonCircleOutline" />
                  </NAvatar>
                </div>
              </div>
              <div v-if="isStreaming && !lastAssistantHasContent" class="streaming">
                <NSpin size="small" />
                <span>思考中...</span>
              </div>
            </div>
          </NScrollbar>

          <div class="footer">
            <div class="footer-toolbar">
              <NSelect
                v-model:value="aiConfigId"
                size="small"
                filterable
                placeholder="选择模型"
                :options="aiConfigOptions"
                class="footer-select"
              />
              <NSelect
                v-model:value="sysPromptId"
                size="small"
                clearable
                placeholder="系统提示词"
                :options="sysPromptOptions"
                class="footer-select"
              />
              <NSelect
                v-model:value="userPromptId"
                size="small"
                clearable
                placeholder="用户提示词"
                :options="userPromptOptions"
                class="footer-select"
                @update:value="onUserPromptChange"
              />
              <div class="toggle-stack">
                <div class="tool-item">
                  <span class="tool-label">思考模式</span>
                  <NSwitch v-model:value="thinking" size="small" />
                </div>
                <div class="tool-item">
                  <span class="tool-label">记忆模式</span>
                  <NSwitch v-model:value="memoryMode" size="small" />
                  <NSelect
                    v-if="memoryMode"
                    v-model:value="memoryCount"
                    size="small"
                    :options="memoryCountOptions"
                    class="footer-memory-select"
                  />
                </div>
              </div>
            </div>

            <NInput
              v-model:value="inputValue"
              type="textarea"
              :autosize="{ minRows: 2, maxRows: 4 }"
              placeholder="输入消息，回车发送（Shift+Enter 换行）"
              :disabled="isStreaming"
              @keydown.enter.exact.prevent="send"
              @keydown.enter.shift.stop
            />
            <div class="footer-actions">
              <NButton v-if="isStreaming" type="warning" quaternary @click="abort">中断</NButton>
              <NButton type="primary" :disabled="!canSend || isStreaming" :loading="isStreaming" @click="send">
                发送
              </NButton>
              <NButton quaternary @click="save">保存会话</NButton>
              <NButton quaternary :loading="shareLoading" @click="shareLast">分享最后一条</NButton>
            </div>
          </div>
        </NCard>
      </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { MdPreview } from "md-editor-v3";
import { NAvatar, NButton, NCard, NIcon, NInput, NScrollbar, NSelect, NSpin, NSwitch, useMessage } from "naive-ui";
import html2canvas from "html2canvas";
import { PersonCircleOutline, SparklesOutline } from "@vicons/ionicons5";
import type { SelectOption } from "naive-ui";
import {
  getAiConfigs,
  getPrompts,
  getSession,
  getVipStatus,
  saveSession,
  shareText,
  type PromptTemplate,
  type SessionMessage,
} from "./api";

const message = useMessage();

const DEFAULT_VISIBLE_COUNT = 20;
const COLLAPSE_CHAR_LIMIT = 200;
const STORAGE_KEY_MODEL_ID = "ai-assistant-last-model-id";

const scrollRef = ref<InstanceType<typeof NScrollbar> | null>(null);

const messages = ref<SessionMessage[]>([]);
const visibleCount = ref(DEFAULT_VISIBLE_COUNT);
const expandedBubbles = ref<Record<number, boolean>>({});
const inputValue = ref("");
const isStreaming = ref(false);
const shareLoading = ref(false);
const controller = ref<AbortController | null>(null);
const saveImageLoading = ref<number | null>(null);
let formatTimer: ReturnType<typeof setInterval> | null = null;

const authLoading = ref(true);
const isAuthenticated = ref(false);
const investmentMottos = [
  "投资有风险，入市需谨慎",
  "别人贪婪我恐惧，别人恐惧我贪婪",
  "不要把所有鸡蛋放在一个篮子里",
  "时间是优秀企业的朋友",
  "买股票就是买公司",
  "市场短期是投票机，长期是称重机",
  "保住本金是投资的第一要务",
  "在别人恐慌时贪婪，在别人贪婪时恐慌",
  "风险来自于你不知道自己在做什么",
  "价格是你付出的，价值是你得到的",
  "投资最重要的品质是耐心",
  "机会总是留给有准备的人",
  "知行合一，方能致远",
  "顺势而为，逆势而思",
  "投资是一场马拉松，不是百米冲刺",
  "独立思考是投资成功的关键",
  "市场永远在波动，但价值终将回归",
  "控制风险比追求收益更重要",
  "学习是最好的投资",
]
const currentMotto = ref(investmentMottos[Math.floor(Math.random() * investmentMottos.length)])
let mottoTimer: ReturnType<typeof setInterval> | null = null

function refreshMotto() {
  currentMotto.value = investmentMottos[Math.floor(Math.random() * investmentMottos.length)]
}

const vipGateLoading = ref(true);
const vipGateOk = ref(false);
const vipGateMessage = ref(
  "您的账户没有VIP权限。"
);

const aiConfigId = ref<number | null>(null);
const aiConfigOptions = ref<SelectOption[]>([]);

// 监听模型选择变化，保存到 localStorage
watch(aiConfigId, (newId) => {
  if (newId != null) {
    localStorage.setItem(STORAGE_KEY_MODEL_ID, String(newId));
  }
});

const sysPromptId = ref<number | null>(null);
const sysPromptOptions = ref<SelectOption[]>([]);
const sysPromptTemplates = ref<PromptTemplate[]>([]);

const userPromptId = ref<number | null>(null);
const userPromptOptions = ref<SelectOption[]>([]);
const userPromptTemplates = ref<PromptTemplate[]>([]);

const thinking = ref(true);
const memoryMode = ref(true);
const memoryCount = ref(5);
const memoryCountOptions: SelectOption[] = [
  { label: "5 条", value: 5 },
  { label: "10 条", value: 10 },
  { label: "20 条", value: 20 },
  { label: "30 条", value: 30 },
  { label: "50 条", value: 50 },
];

const canSend = computed(() => inputValue.value.trim().length > 0);
const lastAssistantHasContent = computed(() => {
  const last = messages.value[messages.value.length - 1];
  return last?.role === 'assistant' && !!last.content?.trim();
});

const codeTheme = computed(() => 'atom-one-light');

function onMdHtmlChanged() {
  nextTick(() => {
    document.querySelectorAll('.msg-markdown').forEach(container => {
      enhanceCodeBlocks(container as HTMLElement, {
        collapseThreshold: 8,
        addLineNumbers: true,
        addCopyButton: true,
        addLanguageTag: true
      })
    })
  })
}

const fromIndex = computed(() => Math.max(0, messages.value.length - visibleCount.value));
const hiddenCount = computed(() => Math.max(0, messages.value.length - visibleCount.value));
const expandAll = computed(() => visibleCount.value >= messages.value.length);
const displayedMessages = computed(() => {
  const from = fromIndex.value;
  return messages.value.slice(from);
});

function getBubbleFullText(msg: SessionMessage) {
  const r = (msg.reasoning || "").trim();
  const c = (msg.content || "").trim();
  return r ? r + "\n" + c : c;
}
function needBubbleCollapse(msg: SessionMessage) {
  return getBubbleFullText(msg).length > COLLAPSE_CHAR_LIMIT;
}
function getBubblePreview(msg: SessionMessage) {
  const full = getBubbleFullText(msg);
  return full.length <= COLLAPSE_CHAR_LIMIT ? full : full.slice(0, COLLAPSE_CHAR_LIMIT) + "...";
}
function isBubbleExpanded(index: number) {
  return !!expandedBubbles.value[index];
}
function toggleBubble(index: number) {
  expandedBubbles.value = { ...expandedBubbles.value, [index]: !expandedBubbles.value[index] };
}

function showMoreHistory() {
  if (expandAll.value) {
    visibleCount.value = DEFAULT_VISIBLE_COUNT;
  } else {
    visibleCount.value = messages.value.length;
  }
  scrollToBottom();
}

async function saveBubbleAsImage(msgIndex: number) {
  const el = document.getElementById("bubble-" + msgIndex);
  if (!el) return;
  saveImageLoading.value = msgIndex;
  try {
    const bg = window.getComputedStyle(el).backgroundColor;
    const backgroundColor = bg && bg !== "rgba(0, 0, 0, 0)" && bg !== "transparent" ? bg : null;

    const canvas = await html2canvas(el, {
      backgroundColor: backgroundColor ?? undefined,
      scale: window.devicePixelRatio ? Math.max(2, window.devicePixelRatio) : 2,
      useCORS: true,
      allowTaint: true,
    });

    const link = document.createElement("a");
    const ts = new Date().toISOString().slice(0, 19).replace(/:/g, "-");
    link.download = `go-stock_ai_${ts}_msg_${msgIndex}.png`;
    link.href = canvas.toDataURL("image/png");
    link.click();
    message.success("图片已保存");
  } catch (e: any) {
    message.error("保存图片失败：" + (e?.message ?? e));
  } finally {
    saveImageLoading.value = null;
  }
}

function nowText() {
  return new Date().toLocaleString();
}

function scrollToBottom() {
  nextTick(() => {
    scrollRef.value?.scrollTo({ top: 999999 });
  });
}

function startNewChat() {
  if (isStreaming.value) {
    message.warning("当前有回答正在生成，请先中断或等待完成");
    return;
  }
  messages.value = [
    {
      role: "assistant",
      content: "你好，我是 go-stock AI 助手（Web 版）。",
      reasoning: "",
      time: nowText(),
    },
  ];
  visibleCount.value = DEFAULT_VISIBLE_COUNT;
  expandedBubbles.value = { 0: true };
  scrollToBottom();
}

async function loadInit() {
  const cfgs = await getAiConfigs();
  aiConfigOptions.value = cfgs.map((c) => ({
    label: `${c.name}${c.modelName ? " [" + c.modelName + "]" : ""}`,
    value: c.id,
  }));
  if (aiConfigId.value == null && aiConfigOptions.value.length) {
    // 优先使用 localStorage 中保存的上一次模型 ID，否则使用第一个模型
    const lastModelId = localStorage.getItem(STORAGE_KEY_MODEL_ID);
    const foundId = lastModelId ? Number(lastModelId) : Number(aiConfigOptions.value[0].value);
    // 检查该 ID 是否仍然可用，不可用则回退到第一个模型
    const isValid = aiConfigOptions.value.some((opt) => opt.value === foundId);
    aiConfigId.value = isValid ? foundId : Number(aiConfigOptions.value[0].value);
  }

  const prompts = await getPrompts();
  sysPromptTemplates.value = prompts.filter((p) => p.type === "模型系统Prompt");
  sysPromptOptions.value = sysPromptTemplates.value
    .map((p) => ({
      label: p.name || "",
      value: Number(p.ID ?? p.id),
    }))
    .filter((x) => Number.isFinite(Number(x.value)));

  userPromptTemplates.value = prompts.filter((p) => p.type === "模型用户Prompt");
  userPromptOptions.value = userPromptTemplates.value
    .map((p) => ({
      label: p.name || "",
      value: Number(p.ID ?? p.id),
    }))
    .filter((x) => Number.isFinite(Number(x.value)));

  const session = await getSession();
  if (Array.isArray(session) && session.length > 0) {
    messages.value = session.map((m) => ({
      role: m.role,
      content: m.content || "",
      reasoning: m.reasoning || "",
      time: m.time || "",
    }));
    visibleCount.value = DEFAULT_VISIBLE_COUNT;
    // 默认：只展开最后一条已存在的助手回复，其余按需折叠
    const lastAssistantIndex = (() => {
      for (let i = messages.value.length - 1; i >= 0; i--) {
        if (messages.value[i]?.role === "assistant" && (messages.value[i]?.content || "").trim()) return i;
      }
      return -1;
    })();
    expandedBubbles.value = lastAssistantIndex >= 0 ? { [lastAssistantIndex]: true } : {};
  } else {
    startNewChat();
  }
  scrollToBottom();
}

function onUserPromptChange(id: number | null) {
  if (!id) return;
  const t = userPromptTemplates.value.find((x) => Number(x.ID ?? x.id) === id);
  if (t?.content) inputValue.value = t.content;
}

async function save() {
  await saveSession(messages.value);
  message.success("会话已保存");
}

function abort() {
  controller.value?.abort();
  controller.value = null;
  isStreaming.value = false;
  message.info("已中断本次回答");
}

function buildHistoryJSON(): string {
  if (!memoryMode.value) return "";
  const n = Math.max(1, Number(memoryCount.value) || 5);
  const history = messages.value.slice(-n).map((m) => ({
    role: m.role,
    content: m.content || "",
    reasoning: m.reasoning || "",
  }));
  return JSON.stringify(history);
}

async function send() {
  if (!canSend.value || isStreaming.value) return;
  if (aiConfigId.value == null) {
    message.warning("请先选择模型");
    return;
  }
  const question = inputValue.value.trim();
  inputValue.value = "";

  messages.value.push({ role: "user", content: question, reasoning: "", time: nowText() });
  const assistantIndex = messages.value.length;
  messages.value.push({ role: "assistant", content: "", reasoning: "", time: nowText() });
  expandedBubbles.value = { ...expandedBubbles.value, [assistantIndex]: true };
  scrollToBottom();

  isStreaming.value = true;
  controller.value = new AbortController();

  // 流式原始内容，定时格式化后赋值给 content
  let rawContent = "";
  let rawReasoning = "";
  formatTimer = setInterval(() => {
    const assistant = messages.value[assistantIndex];
    if (!assistant || assistant.role !== "assistant") return;
    if (rawContent) {
      const { content: formatted, jsonMarkdown } = formatMarkdown(rawContent);
      assistant.content = formatted;
      assistant.jsonMarkdown = jsonMarkdown;
    }
    if (rawReasoning) {
      const { content: formattedReasoning } = formatMarkdown(rawReasoning);
      assistant.reasoning = formattedReasoning;
    }
    scrollToBottom();
  }, 1500);

  try {
    const res = await fetch("/api/chat/agent-chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      signal: controller.value.signal,
      body: JSON.stringify({
        question,
        aiConfigId: aiConfigId.value,
        sysPromptId: sysPromptId.value ?? 0,
        thinking: thinking.value,
        enableTools: true,
        historyJSON: buildHistoryJSON(),
      }),
    });
    if (!res.ok || !res.body) throw new Error(await res.text());

    const reader = res.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const parts = buffer.split("\n\n");
      buffer = parts.pop() || "";

      for (const part of parts) {
        const lines = part.split("\n");
        let eventType = "message";
        for (const line of lines) {
          if (line.startsWith("event:")) eventType = line.slice(6).trim();
          if (!line.startsWith("data:")) continue;
          const dataText = line.slice(5).trim();
          if (eventType === "done" || dataText === "[DONE]") continue;

          try {
            const msg = JSON.parse(dataText) as any;
            if (msg?.reasoning_content) rawReasoning += msg.reasoning_content;
            if (msg?.content) rawContent += msg.content;
            if (Array.isArray(msg?.tool_calls)) {
              for (const t of msg.tool_calls) {
                rawContent += `\n[工具调用] ${t.function?.name || "unknown"}: ${t.function?.arguments || ""}\n`;
              }
            }
          } catch {
            // ignore parse error
          }
        }
        scrollToBottom();
      }
    }
  } catch (e: any) {
    if (e?.name !== "AbortError") message.error(`发送失败：${e?.message ?? e}`);
  } finally {
    // 停止格式化定时器，最终格式化一次
    if (formatTimer) { clearInterval(formatTimer); formatTimer = null; }
    const assistant = messages.value[assistantIndex];
    if (assistant && assistant.role === "assistant") {
      if (rawContent) {
        const { content: formatted, jsonMarkdown } = formatMarkdown(rawContent);
        assistant.content = formatted;
        assistant.jsonMarkdown = jsonMarkdown;
      }
      if (rawReasoning) {
        const { content: formattedReasoning } = formatMarkdown(rawReasoning);
        assistant.reasoning = formattedReasoning;
      }
    }
    isStreaming.value = false;
    controller.value = null;
    await saveSession(messages.value).catch(() => {});
    scrollToBottom();
  }
}

async function copyText(text: string) {
  const t = (text || "").trim();
  if (!t) return;
  try {
    await navigator.clipboard.writeText(t);
    message.success("已复制");
  } catch {
    message.warning("复制失败，请手动选择文本");
  }
}

async function shareOne(text: string) {
  const t = (text || "").trim();
  if (!t) return;
  shareLoading.value = true;
  try {
    const msg = await shareText(t, "AI助手");
    message.success(msg);
  } catch (e: any) {
    message.error(e?.message ?? "分享失败");
  } finally {
    shareLoading.value = false;
  }
}

async function shareLast() {
  for (let i = messages.value.length - 1; i >= 0; i--) {
    const m = messages.value[i];
    if (m.role === "assistant" && (m.content || "").trim()) {
      await shareOne(m.content);
      return;
    }
  }
  message.warning("暂无可分享内容");
}

onMounted(async () => {
  // 检查用户是否登录
  const token = localStorage.getItem('token');
  if (!token) {
    authLoading.value = false;
    isAuthenticated.value = false;
    return;
  }

  isAuthenticated.value = true;
  
  // 验证VIP状态
  vipGateLoading.value = true;
  try {
    const st = await getVipStatus();
    vipGateOk.value = !!st.ok;
    if (!st.ok && st.message) vipGateMessage.value = st.message;
  } catch (e: any) {
    vipGateOk.value = false;
    vipGateMessage.value = "无法连接校验接口：" + String(e?.message ?? e);
  } finally {
    vipGateLoading.value = false;
    authLoading.value = false;
  }
  
  if (!vipGateOk.value) return;
  
  loadInit().catch((e) => {
    message.error(String(e?.message ?? e));
    startNewChat();
  });
  mottoTimer = setInterval(refreshMotto, 30000);
});

onBeforeUnmount(() => {
  if (mottoTimer) {
    clearInterval(mottoTimer);
    mottoTimer = null;
  }
  if (formatTimer) {
    clearInterval(formatTimer);
    formatTimer = null;
  }
});

function goToLogin() {
  window.location.href = '/login';
}

// === Markdown 格式化工具函数 ===
function isBlockElement(line: string): boolean {
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

function splitInlineHeading(line: string): string {
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

function hasMarkdownContent(str: string): boolean {
  if (!str || typeof str !== 'string') return false
  return /(^|\n)\s*#{1,6}\s/.test(str) ||
    /(^|\n)\s*\|/.test(str) ||
    /(^|\n)\s*---/.test(str) ||
    /(^|\n)\s*[-*+]\s/.test(str) ||
    /(^|\n)\s*>\s/.test(str) ||
    /(^|\n)\s*```/.test(str)
}

function extractMarkdownFromJson(obj: any): string | null {
  if (typeof obj === 'string') return obj
  if (Array.isArray(obj)) {
    return obj.map((item: any) => typeof item === 'string' ? item : JSON.stringify(item, null, 2)).join('\n\n')
  }
  if (typeof obj === 'object' && obj !== null) {
    for (const key of ['response', 'content', 'text', 'result', 'answer', 'message', 'output']) {
      if (obj[key] != null) {
        const val = obj[key]
        if (typeof val === 'string' && hasMarkdownContent(val)) return val
        if (typeof val === 'object') {
          const extracted: string | null = extractMarkdownFromJson(val)
          if (extracted) return extracted
        }
      }
    }
    const values = Object.values(obj).filter((v: any) => typeof v === 'string' && hasMarkdownContent(v)) as string[]
    if (values.length > 0) return values.join('\n\n')
  }
  return null
}

function findJsonEnd(content: string, start: number): number {
  let depth = 0, bracketDepth = 0, inStr = false, escape = false
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

function extractJsonMarkdown(content: string): { content: string; jsonMarkdown: string } {
  if (!content) return { content: '', jsonMarkdown: '' }
  const cleaned: string[] = []
  const jsonParts: string[] = []
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
    if (inCodeBlock) { cleaned.push(content[i]); i++; continue }
    if (content[i] === '{') {
      const end = findJsonEnd(content, i)
      if (end > i) {
        const jsonStr = content.substring(i, end + 1)
        try {
          const obj = JSON.parse(jsonStr)
          const md = extractMarkdownFromJson(obj)
          if (md) { jsonParts.push(md) } else { cleaned.push('\n\n```json\n' + jsonStr + '\n```\n\n') }
          i = end + 1
          continue
        } catch {}
      }
    }
    cleaned.push(content[i])
    i++
  }
  return { content: cleaned.join(''), jsonMarkdown: jsonParts.join('\n\n---\n\n') }
}

function formatMarkdown(content: string): { content: string; jsonMarkdown: string } {
  if (!content) return { content: '', jsonMarkdown: '' }
  const { content: cleaned, jsonMarkdown } = extractJsonMarkdown(content)
  let inCodeBlock = false
  const lines = cleaned.split('\n')
  const result: string[] = []

  for (let i = 0; i < lines.length; i++) {
    let line = lines[i]
    const trimmed = line.replace(/^[\t ]+/, '')
    if (trimmed.startsWith('```')) {
      inCodeBlock = !inCodeBlock
      if (!inCodeBlock) { result.push(trimmed); continue }
    }
    if (inCodeBlock) { result.push(line); continue }
    if (trimmed !== line && trimmed !== '') { line = trimmed }

    // 处理 ---##标题 或 ---#标题 等水平线紧贴标题的情况
    const hrHeadingMatch = line.match(/^(---+|\*\*\*+|___+)(#{1,6}\s+.*)$/)
    if (hrHeadingMatch) {
      // 先确保水平线前有空行
      if (result.length > 0) {
        const prev = result[result.length - 1]
        if (prev !== '' && !isBlockElement(prev.replace(/^[\t ]+/, ''))) {
          result.push('')
        }
      }
      result.push(hrHeadingMatch[1])  // 水平线
      result.push('')                  // 空行
      result.push(hrHeadingMatch[2])   // 标题
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
      result.push(hrBlockMatch[1])  // 水平线
      result.push('')               // 空行
      result.push(line.substring(hrBlockMatch[1].length))  // 剩余内容
      continue
    }

    if (i > 0 && isBlockElement(trimmed)) {
      const prev = result.length > 0 ? result[result.length - 1] : ''
      if (prev !== '' && !isBlockElement(prev.replace(/^[\t ]+/, ''))) { result.push('') }
    }
    line = splitInlineHeading(line)
    result.push(line)
  }
  return { content: result.join('\n'), jsonMarkdown }
}

// === 代码块增强（内联，与桌面端 codeBlockEnhancer 一致）===
function extractCodeLanguage(codeEl: HTMLElement): string {
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

function enhanceCodeBlocks(container: HTMLElement, options: { collapseThreshold?: number; addLineNumbers?: boolean; addCopyButton?: boolean; addLanguageTag?: boolean } = {}): void {
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

    const lang = extractCodeLanguage(codeEl)
    const text = codeEl.textContent || ''
    const lineCount = text.split('\n').length

    block.dataset.enhanced = 'true'

    if (addLanguageTag && lang) {
      const tag = document.createElement('span')
      tag.className = 'code-lang-tag'
      tag.textContent = lang.toUpperCase()
      block.appendChild(tag)
    }

    if (addCopyButton) {
      const btn = document.createElement('button')
      btn.className = 'code-copy-btn'
      btn.type = 'button'
      btn.title = '复制代码'
      btn.innerHTML = '<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/></svg>'
      btn.addEventListener('click', async (e: Event) => {
        e.stopPropagation()
        try {
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
          btn.classList.add('copy-success')
          btn.innerHTML = '<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/></svg>'
          setTimeout(() => {
            btn.classList.remove('copy-success')
            btn.innerHTML = '<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/></svg>'
          }, 2000)
        } catch {
          btn.classList.add('copy-error')
          btn.title = '复制失败'
          setTimeout(() => { btn.classList.remove('copy-error'); btn.title = '复制代码' }, 2000)
        }
      })
      block.appendChild(btn)
    }

    if (addLineNumbers) {
      const pre = codeEl.parentElement
      if (pre) {
        pre.classList.add('code-with-line-numbers')
        block.dataset.lineCount = String(lineCount)
        if (lineCount > 3) block.classList.add('show-line-numbers')
      }
    }

    const shouldCollapse = lang === 'json' || lineCount > collapseThreshold
    if (shouldCollapse) {
      if (!block.querySelector('.code-collapse-btn')) {
        block.classList.add('code-collapsed')
        const cbtn = document.createElement('span')
        cbtn.className = 'code-collapse-btn'
        cbtn.textContent = '展开'
        cbtn.addEventListener('click', (e: Event) => {
          e.stopPropagation()
          const collapsed = block.classList.toggle('code-collapsed')
          cbtn.textContent = collapsed ? '展开' : '收起'
        })
        block.appendChild(cbtn)
      }
    }
  })
}
</script>

<style scoped>
.auth-loading {
  min-height: 60vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 32px 20px;
  text-align: center;
}
.auth-loading-text {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.55);
}
.auth-required {
  min-height: 60vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 32px 20px;
  text-align: center;
}
.auth-title {
  font-size: 20px;
  font-weight: 700;
  color: #111827;
}
.auth-desc {
  margin: 0;
  font-size: 15px;
  line-height: 1.6;
  color: #374151;
}
.vip-gate {
  min-height: 60vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 32px 20px;
  text-align: center;
}
.vip-gate-text {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.55);
}
.vip-gate-denied {
  max-width: 520px;
  margin: 0 auto;
}
.vip-gate-title {
  font-size: 20px;
  font-weight: 700;
  color: #111827;
}
.vip-gate-desc {
  margin: 0;
  font-size: 15px;
  line-height: 1.6;
  color: #374151;
}
.vip-gate-hint {
  margin: 0;
  font-size: 13px;
  line-height: 1.55;
  color: #6b7280;
}
.page {
  max-width: 980px;
  margin: 0 auto;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.title {
  font-weight: 700;
  font-size: 18px;
  white-space: nowrap;
}
.motto {
  flex: 1;
  text-align: center;
  font-size: 13px;
  color: #6b7280;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  opacity: 0.85;
  transition: opacity 0.4s;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}
.tool-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.tool-label {
  font-size: 12px;
  opacity: 0.8;
}
.chat-card {
  border-radius: 12px;
}
.chat-scroll {
  height: calc(100vh - 220px);
  min-height: 420px;
}
.msg-list {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.message-list-expand {
  display: flex;
  justify-content: center;
  padding: 6px 0;
}
.msg {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.msg.user {
  justify-content: flex-end;
}
.msg-avatar {
  flex: 0 0 auto;
  margin-top: 2px;
}
.ai-avatar :deep(.n-avatar) {
  box-shadow: 0 4px 10px rgba(109, 93, 252, 0.35);
}
.user-avatar :deep(.n-avatar) {
  box-shadow: 0 4px 10px rgba(59, 130, 246, 0.35);
}
.bubble {
  width: min(860px, 100%);
  border-radius: 12px;
  padding: 10px 12px;
  background: #fff;
  border: 1px solid rgba(0, 0, 0, 0.08);
}
.msg.user .bubble {
  background: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
  color: #f8fafc;
}
.reasoning-details {
  margin-bottom: 8px;
  border: 1px solid rgba(109, 93, 252, 0.2);
  border-radius: 8px;
  overflow: hidden;
}
.reasoning-summary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 500;
  color: #6d5dfc;
  background: linear-gradient(135deg, rgba(109, 93, 252, 0.08) 0%, rgba(109, 93, 252, 0.03) 100%);
  cursor: pointer;
  user-select: none;
  list-style: none;
}
.reasoning-summary::-webkit-details-marker {
  display: none;
}
.reasoning-summary::before {
  content: '▶';
  font-size: 10px;
  transition: transform 0.2s;
}
.reasoning-details[open] .reasoning-summary::before {
  transform: rotate(90deg);
}
.reasoning-icon {
  flex-shrink: 0;
}
.reasoning-md {
  padding: 8px 12px;
  font-size: 13px;
  line-height: 1.6;
  color: #6b7280;
  background: rgba(109, 93, 252, 0.02);
  border-top: 1px dashed rgba(109, 93, 252, 0.15);
}
.reasoning-md :deep(.md-editor-preview-wrapper) {
  padding: 0;
}
.reasoning-md :deep(.md-editor-preview) {
  font-size: 13px;
  color: #6b7280;
}
.json-md-details {
  margin-bottom: 8px;
  border: 1px solid rgba(59, 130, 246, 0.2);
  border-radius: 8px;
  overflow: hidden;
}
.json-md-summary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 500;
  color: #3b82f6;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.08) 0%, rgba(59, 130, 246, 0.03) 100%);
  cursor: pointer;
  user-select: none;
  list-style: none;
}
.json-md-summary::-webkit-details-marker {
  display: none;
}
.json-md-summary::before {
  content: '▶';
  font-size: 10px;
  transition: transform 0.2s;
}
.json-md-details[open] .json-md-summary::before {
  transform: rotate(90deg);
}
.msg-content-collapsed {
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.5;
  opacity: 0.9;
}
.bubble-actions {
  margin-top: 8px;
  display: flex;
  justify-content: flex-end;
}
.msg-expand-btn {
  font-size: 12px;
}
.msg.user .meta {
  color: rgba(248, 250, 252, 0.92);
}
.msg.user .meta :deep(.n-button) {
  color: #f8fafc;
}
.user-text {
  white-space: pre-wrap;
  word-break: break-word;
  color: #f8fafc;
  line-height: 1.6;
}
.meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-top: 8px;
  font-size: 11px;
  opacity: 0.75;
}
.meta-actions {
  display: inline-flex;
  gap: 6px;
}
.footer {
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.footer-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-bottom: 4px;
}
.toggle-stack {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 0 0 auto;
}
.toggle-stack .tool-item {
  width: max-content;
}
.footer-select {
  min-width: 140px;
  flex: 1 1 140px;
  max-width: 200px;
}
.footer-memory-select {
  width: 90px;
}
.footer-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  flex-wrap: wrap;
}
.streaming {
  display: flex;
  align-items: center;
  gap: 8px;
  opacity: 0.8;
  font-size: 12px;
}
.new-chat-btn {
  font-weight: 700;
}
.history-toggle-btn {
  white-space: nowrap;
}
.msg-expand-btn {
  white-space: nowrap;
}

/* 响应式适配 */
@media (max-width: 768px) {
  .page {
    padding: 10px;
  }
  .header {
    flex-wrap: wrap;
    gap: 8px;
  }
  .motto {
    order: 3;
    width: 100%;
    text-align: left;
  }
  .toolbar {
    width: 100%;
    justify-content: flex-start;
  }
  .bubble {
    width: calc(100% - 48px);
    max-width: calc(100vw - 80px);
  }
  .footer-select {
    min-width: 120px;
    flex: 1 1 120px;
  }
  .chat-scroll {
    height: calc(100vh - 280px);
    min-height: 300px;
  }
}
</style>

<style>
/* 代码块增强全局样式 */
.msg-markdown .md-editor-code-block {
  position: relative;
}

.msg-markdown .code-lang-tag {
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

.msg-markdown .code-copy-btn {
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

.msg-markdown .md-editor-code-block:hover .code-copy-btn {
  opacity: 1;
}

.msg-markdown .code-copy-btn:hover {
  background: #3b82f6;
  border-color: #3b82f6;
  color: #fff;
}

.msg-markdown .code-copy-btn.copy-success {
  color: #67c23a;
  border-color: #67c23a;
}

.msg-markdown .code-copy-btn.copy-error {
  color: #f56c6c;
  border-color: #f56c6c;
}

.msg-markdown .code-collapse-btn {
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

.msg-markdown .md-editor-code-block:hover .code-collapse-btn {
  opacity: 1;
}

.msg-markdown .md-editor-code-block.code-collapsed pre {
  max-height: 120px;
  overflow: hidden;
}

.msg-markdown .md-editor-code-block.code-collapsed::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 50px;
  background: linear-gradient(transparent, #fff);
  pointer-events: none;
}
</style>