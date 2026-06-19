# go-stock Web 端 Phase 6 代码审查 - P0 问题修复方案

> **日期**: 2026-06-13  
> **审查模式**: Standard  
> **Verdict**: ❌ NEEDS WORK（8个P0问题必须修复）

---

## 目录

1. [WebSocket AUTH_RESPONSE 不断线重连](#p0-1-websocket-auth_response-失败后不断线重连)
2. [非 JSON 消息类型崩溃](#p0-2-非-json-websocket-消息导致类型崩溃)
3. [extractApiData null 返回](#p0-3-extractapidata-null-返回导致-map-崩溃)
4. [SSE 并发流误中断](#p0-4-sse-并发流共用全局-abort-标志)
5. [SSE 中断伪装 DONE](#p0-5-sse-流中断伪装-done-信号)
6. [ECharts 实例泄漏](#p0-6-echarts-实例泄漏)
7. [SSE reader releaseLock 缺失](#p0-7-sse-reader-releaselock-缺失)
8. [agent-chat 重复赋值](#p0-8-agent-chatvue-handleagentmessage-重复赋值)

---

## P0-1: WebSocket AUTH_RESPONSE 失败后不断线重连

### 文件：`frontend/src/utils/websocket.js`

### 当前代码（line 60-72）：
```javascript
if (data.type === 'AUTH_RESPONSE') {
  if (!data.success) {
    console.error('WebSocket认证失败:', data.message);
    if (data.code === 'TOKEN_EXPIRED' || data.code === 'UNAUTHORIZED') {
      Auth.clearToken();
      this._triggerEvent('auth_expired', data);
    }
  } else {
    console.log('WebSocket认证成功');
  }
  return;
}
```

### 修复后：
```javascript
if (data.type === 'AUTH_RESPONSE') {
  if (!data.success) {
    console.error('WebSocket认证失败:', data.message);
    if (data.code === 'TOKEN_EXPIRED' || data.code === 'UNAUTHORIZED') {
      Auth.clearToken();
      this._triggerEvent('auth_expired', data);
      // ===== 修复：触发登出事件后断线并触发重连 =====
      this.isManuallyClosed = false;
      this.ws.close();  // 关闭当前连接，触发 onclose → 自动重连
      return;
    }
  } else {
    console.log('WebSocket认证成功');
    this.authRetry = false;
  }
  return;
}
```

### 同时需要修复重连策略（line 8-9）：
```javascript
// 当前：
this.reconnectInterval = 5000; // 固定5秒
this.maxReconnectAttempts = 5;  // 最大5次

// 修复后（指数退避 + 随机抖动）：
this.baseReconnectInterval = 1000;  // 初始1秒
this.maxReconnectInterval = 30000;  // 最大30秒
this.reconnectJitter = 1000;        // 随机抖动±500ms
this.maxReconnectAttempts = 0;       // 0 = 无限重连（由上层策略控制）

// 在 onclose 中（line 86-89）重连逻辑改为：
const backoff = Math.min(
  this.baseReconnectInterval * Math.pow(2, this.reconnectAttempts),
  this.maxReconnectInterval
);
const jitter = Math.random() * this.reconnectJitter - this.reconnectJitter / 2;
const delay = Math.max(backoff + jitter, 500); // 最小500ms
setTimeout(() => this.connect(), delay);
```

---

## P0-2: 非 JSON WebSocket 消息导致类型崩溃

### 文件：`frontend/src/utils/websocket.js`

### 当前代码（line 77-79）：
```javascript
} catch (error) {
  console.error('Failed to parse WebSocket message:', event.data, error);
  this._triggerEvent('message', event.data);  // ❌ 原始字符串
}
```

### 修复后：
```javascript
} catch (error) {
  console.error('Failed to parse WebSocket message:', event.data, error);
  // ===== 修复：统一包装为标准格式 =====
  this._triggerEvent('message', { 
    type: 'PARSE_ERROR', 
    raw: event.data,
    error: error.message 
  });
}
```

---

## P0-3: extractApiData null 返回导致 `.map()` 崩溃

### 文件：`frontend/src/services/wails-bridge.js`

### 当前代码（line 23-37）：
```javascript
function extractApiData(response) {
  if (response.data) {
    if (response.data.code === 0) {
      if (response.data.data && typeof response.data.data === 'object' && response.data.data.list !== undefined) {
        return response.data.data.list;
      }
      return response.data.data;  // ❌ 可能返回 null/undefined
    }
    return response.data;
  }
  return response;
}
```

### 修复后：
```javascript
function extractApiData(response) {
  if (response.data) {
    if (response.data.code === 0) {
      // 分页格式 {code: 0, data: {list: [...]}}
      if (response.data.data && typeof response.data.data === 'object' && response.data.data.list !== undefined) {
        return response.data.data.list;
      }
      // ===== 修复：null/undefined 保护 =====
      return response.data.data ?? [];
    }
    return response.data;
  }
  return Array.isArray(response) ? response : (response ?? []);
}
```

### 同步修复 extractApiSingle（line 40-48）：
```javascript
function extractApiSingle(response) {
  if (response.data) {
    if (response.data.code === 0) {
      return response.data.data ?? null;  // single 返回 null 是合法情况
    }
    return response.data;
  }
  return response;
}
```

---

## P0-4: SSE 并发流共用全局 abort 标志

### 文件：`frontend/src/services/wails-bridge.js`

### 问题描述：
`connectSSE()`、`NewChatStream()`、`SummaryStockNews()` 使用 `window._abortXxxStream` 全局标志。
当用户在 agent-chat 中连续输入两条消息，或快速点击不同股票的 AI 分析时，后请求会错误地 `abort` 先请求。

### 修复方案（以 NewChatStream 为例，line 589-670）：

创建一个 SSE 管理器来追踪活跃的 SSE 连接：

```javascript
// 在文件顶部添加 SSE 连接管理器
const sseConnections = new Map();
let sseConnectionId = 0;

function abortSSE(key) {
  const conn = sseConnections.get(key);
  if (conn) {
    conn.aborted = true;
    conn.reader?.cancel();
    sseConnections.delete(key);
  }
}

function createSSEConnection(key) {
  abortSSE(key); // 先取消旧的同 key 连接
  const conn = { aborted: false, reader: null, controller: null };
  sseConnections.set(key, conn);
  return conn;
}
```

然后在 `NewChatStream` 中使用：
```javascript
export function NewChatStream(stock, stockCode, question, aiConfigId, sysPromptId, enableTools, think) {
  if (isWailsMode()) {
    return window.go.main.App.NewChatStream(stock, stockCode, question, aiConfigId, sysPromptId, enableTools, think);
  }
  
  // ===== 修复：创建唯一连接标识，自动取消上次同键连接 =====
  const connKey = `newChatStream_${stockCode}_${Date.now()}`;
  const conn = createSSEConnection(`newChatStream`);
  
  const baseURL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
  const token = localStorage.getItem('token');
  const origin = window.location.origin;
  const fullUrl = baseURL.startsWith('http') ? `${baseURL}/ai/analyze` : `${origin}${baseURL}/ai/analyze`;

  // 使用 AbortController 替代全局标志
  const controller = new AbortController();
  conn.controller = controller;

  fetch(fullUrl, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({...}),
    signal: controller.signal,  // ===== 新增 =====
  }).then(response => {
    // ... 原有 SSE 处理逻辑
    const reader = response.body.getReader();
    conn.reader = reader;  // ===== 新增：保存 reader 以便取消 =====
    
    // 在 read() 函数中每次循环检查 conn.aborted：
    function read() {
      if (conn.aborted) {  // ===== 新增：取消保护 =====
        reader.releaseLock();
        return;
      }
      reader.read().then(...).catch(...);
    }
    // ...
  });
}
```

---

## P0-5: SSE 流中断伪装 DONE 信号

### 文件：`frontend/src/services/wails-bridge.js`

### 修改点：connectSSE (line 79-145)、NewChatStream (line 589-670)、SummaryStockNews 等所有 SSE 函数

### 当前代码（line 135-137）：
```javascript
}).catch(err => {
  console.error('SSE read error:', err);
  EventsEmit(eventName, doneMsg);  // ❌ 发送与正常完成相同的 'DONE'
});
```

### 修复后：
```javascript
}).catch(err => {
  console.error('SSE read error:', err);
  // ===== 修复：区分异常中断和正常完成 =====
  EventsEmit(eventName, { content: 'DONE', truncated: true, error: err.message });
});
```

### 同步修复 NewChatStream（line 658-661）：
```javascript
}).catch(err => {
  console.error('NewChatStream SSE read error:', err);
  EventsEmit('newChatStream', { content: 'DONE', error: err.message });  // 修复
});
```

### 同步修复 connectSSE HTTP 错误（line 92-94）：
```javascript
if (!response.ok) {
  // ===== 修复：提取错误体信息 =====
  response.text().then(body => {
    let errorDetail = `HTTP ${response.status}`;
    try {
      const parsed = JSON.parse(body);
      errorDetail = parsed.message || parsed.error || errorDetail;
    } catch(e) {}
    EventsEmit(eventName, { error: errorDetail, status: response.status });
  }).catch(() => {
    EventsEmit(eventName, { error: `HTTP ${response.status}` });
  });
  return;
}
```

### 前端 stock.vue 监听端需要适配：
```javascript
// stock.vue 中监听 newChatStream 的位置（约 line 200-300）
// 在收到 DONE 信号时增加 checks：
function onStreamDone(data) {
  if (data.error) {
    // 异常截断，显示警告
    data.airesult += '\n\n> ⚠️ AI分析被中断（' + data.error + '），结果可能不完整';
  }
  if (data.truncated) {
    data.airesult += '\n\n> ⚠️ 响应被截断，结果可能不完整';
  }
  data.loading = false;
  data.analysisStatus = "";
  message.destroyAll();
}
```

---

## P0-6: ECharts 实例泄漏

### 文件：`frontend/src/components/stock.vue`

### 问题位置：
- `showFsChart()` (line 1104) — 分时图
- `handleKLine()` (line 1371) — K线图
- `handleFeishi()` (line 1345) — 定时刷新分时图（每10秒）

### 修复方案：

```vue
<script setup>
// ===== 在文件顶部添加 =====
const fsChartInstance = ref(null);   // 分时图 ECharts 实例
const kLineChartInstance = ref(null); // K 线图 ECharts 实例
</script>
```

#### showFsChart 修复（line 1104）：
```javascript
function showFsChart(code, name) {
  data.name = name;
  data.code = code;
  
  // ===== 修复：销毁旧实例 =====
  if (fsChartInstance.value) {
    fsChartInstance.value.dispose();
    fsChartInstance.value = null;
  }
  
  const chart = echarts.init(kLineChartRef2.value);
  fsChartInstance.value = chart;  // ===== 保存引用 =====
  
  GetStockMinutePriceLineData(code, name).then(result => {
    // ... 原有 option 配置不变 ...
    chart.setOption(option);
  }).catch(err => {
    console.error('分时图数据获取失败:', err);
    message.error('分时图数据获取失败');
  });
}
```

#### handleKLine 修复（line 1368-1371）：
```javascript
function handleKLine() {
  GetStockKLine(data.code, data.name, 365).then(result => {
    // ===== 修复：销毁旧实例 =====
    if (kLineChartInstance.value) {
      kLineChartInstance.value.dispose();
      kLineChartInstance.value = null;
    }
    
    const chart = echarts.init(kLineChartRef.value);
    kLineChartInstance.value = chart;  // ===== 保存引用 =====
    
    // ... 原有 option 配置不变 ...
    chart.setOption(option);
  }).catch(err => {
    console.error('K线数据获取失败:', err);
    message.error('K线数据获取失败');
  });
}
```

#### onBeforeUnmount 补充清理（line 721-752）：
```javascript
onBeforeUnmount(() => {
  // ... 原有清理逻辑 ...
  
  // ===== 新增：释放 ECharts 实例 =====
  if (fsChartInstance.value) {
    fsChartInstance.value.dispose();
    fsChartInstance.value = null;
  }
  if (kLineChartInstance.value) {
    kLineChartInstance.value.dispose();
    kLineChartInstance.value = null;
  }
});
```

---

## P0-7: SSE reader releaseLock 缺失

### 文件：`frontend/src/services/wails-bridge.js`

### 问题位置：
- connectSSE (line 122-138) — reader.read().catch()
- NewChatStream (line 646-661) — reader.read().catch()

### 修复 connectSSE（line 122-138）：

```javascript
const reader = response.body.getReader();

function read() {
  reader.read().then(({ done, value }) => {
    if (done) {
      if (buffer.trim()) processLine(buffer);
      reader.releaseLock();  // ===== 新增 =====
      EventsEmit(eventName, doneMsg);
      return;
    }
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop();
    lines.forEach(processLine);
    read();
  }).catch(err => {
    console.error('SSE read error:', err);
    // ===== 修复：释放 lock =====
    try { reader.releaseLock(); } catch(e) { /* lock already released */ }
    EventsEmit(eventName, { content: doneMsg, truncated: true, error: err.message });
  });
}
```

### 同步修复 NewChatStream（line 646-661）：

```javascript
function read() {
  reader.read().then(({ done, value }) => {
    if (done) {
      if (buffer.trim()) processLine(buffer);
      reader.releaseLock();  // ===== 新增 =====
      EventsEmit('newChatStream', 'DONE');
      return;
    }
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop();
    lines.forEach(processLine);
    read();
  }).catch(err => {
    console.error('NewChatStream SSE read error:', err);
    try { reader.releaseLock(); } catch(e) { /* lock already released */ }
    EventsEmit('newChatStream', { content: 'DONE', error: err.message });
  });
}
```

---

## P0-8: agent-chat.vue handleAgentMessage 重复赋值

### 文件：`frontend/src/components/agent-chat.vue`

### 当前代码（line 477-495）：
```javascript
const handleAgentMessage = (data) => {
  // 处理错误消息
  if (data && data['error']) {
    isStreamLoad.value = false;
    loading.value = false;
    stopFormatTimer();
    const lastItemIndex = chatList.value.findIndex(item => item.role === 'assistant');
    if (lastItemIndex !== -1) {
      const lastItem = chatList.value[lastItemIndex];
      const updatedItem = { ...lastItem };
      updatedItem.content = `❌ Agent 调用失败：${data['error']}`;
      updatedItem.rawContent = updatedItem.content;
      chatList.value[lastItemIndex] = updatedItem;  // ← line 491
    }
    // ===== BUG: line 494 在 if 块外，lastItemIndex 可能为 -1 =====
    chatList.value[lastItemIndex] = updatedItem;     // ← line 494 ❌
    return;
  }
  // ...
};
```

### 修复后：
```javascript
const handleAgentMessage = (data) => {
  // 处理错误消息
  if (data && data['error']) {
    isStreamLoad.value = false;
    loading.value = false;
    stopFormatTimer();
    const lastItemIndex = chatList.value.findIndex(item => item.role === 'assistant');
    if (lastItemIndex !== -1) {
      const chatListClone = [...chatList.value];
      const updatedItem = { ...chatListClone[lastItemIndex] };
      updatedItem.content = `❌ Agent 调用失败：${data['error']}`;
      updatedItem.rawContent = updatedItem.content;
      chatListClone[lastItemIndex] = updatedItem;
      chatList.value = chatListClone;  // ===== 统一用一次赋值 =====
    }
    return;
  }
  // ... 后续正常处理逻辑 ...
};
```

---

## 修复汇总时间估计

| 优先级 | 数量 | 估算时间 |
|--------|------|----------|
| P0 | 8个 | 4-6小时 |
| P1 | 12个 | 6-8小时 |
| P2 | 10个 | 4-6小时 |
| P3 | 8个 | 2-3小时 |
| **合计** | **38个** | **16-23小时** |

> 建议：先修复 P0 的 8 个关键问题（约 4-6 小时），然后审查 P1 的高优先级问题分批修复。
