// wails-bridge.js - 桥接 Wails 和 Web API，使应用同时支持桌面模式和浏览器模式

import apiService from './api.js';

// 检测是否在 Wails 环境中
function isWailsMode() {
  return typeof window !== 'undefined' && window.go && window.go.main && window.go.main.App;
}

// 获取认证 token
function getAuthHeaders() {
  // 优先使用 Auth 工具类获取 token
  const token = localStorage.getItem('token');  // 现在仍然可以从 localStorage 获取
  return token ? { Authorization: `Bearer ${token}` } : {};
}

// 辅助函数：从 Web API 响应中提取数据
function extractApiData(response) {
  if (response.data) {
    // 标准格式 {code: 0, data: [...]}
    if (response.data.code === 0) {
      // 检查是否是分页格式 {code: 0, data: {list: [...]}}
      if (response.data.data && typeof response.data.data === 'object' && response.data.data.list !== undefined) {
        return response.data.data.list;
      }
      return response.data.data;
    }
    // 直接返回数据
    return response.data;
  }
  return response;
}

// 辅助函数：从 Web API 响应中提取单个对象
function extractApiSingle(response) {
  if (response.data) {
    if (response.data.code === 0) {
      return response.data.data;
    }
    return response.data;
  }
  return response;
}

/**
 * SSE 工具函数：建立 SSE 连接，通过 EventsEmit 将数据分发到前端事件系统
 * @param {string} url         - SSE 接口路径（相对于 baseURL）
 * @param {object} body        - POST body
 * @param {string} eventName   - 对应的 Wails 事件名（通过 EventsEmit 触发）
 * @param {Function} doneMsg   - 生成"完成"时的消息对象，默认 'DONE'
 */
function connectSSE(url, body, eventName, doneMsg = 'DONE') {
  const baseURL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
  const token = localStorage.getItem('token');
  const fullUrl = baseURL.startsWith('http') ? `${baseURL}${url}` : `${window.location.origin}${baseURL}${url}`;

  fetch(fullUrl, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(body),
  }).then(response => {
    if (!response.ok) {
      EventsEmit(eventName, { error: `HTTP ${response.status}` });
      return;
    }
    const reader = response.body.getReader();
    const decoder = new TextDecoder('utf-8');
    let buffer = '';

    function processLine(line) {
      if (!line.trim()) return;
      if (line.startsWith('event:')) return; // 跳过 event 行，由 data 行驱动
      if (line.startsWith('data:')) {
        const raw = line.slice(5).trim();
        // 完成信号
        if (raw.startsWith('{')) {
          try {
            const parsed = JSON.parse(raw);
            if (parsed.chatId !== undefined) {
              // 收到 done 信号
              EventsEmit(eventName, doneMsg);
              return;
            }
          } catch (e) { /* ignore */ }
        }
        if (raw) {
          EventsEmit(eventName, { content: raw });
        }
      }
    }

    function read() {
      reader.read().then(({ done, value }) => {
        if (done) {
          // 流结束
          if (buffer.trim()) processLine(buffer);
          EventsEmit(eventName, doneMsg);
          return;
        }
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop(); // 最后一行可能不完整，留到下次
        lines.forEach(processLine);
        read();
      }).catch(err => {
        console.error('SSE read error:', err);
        EventsEmit(eventName, doneMsg);
      });
    }
    read();
  }).catch(err => {
    console.error('SSE connect error:', err);
    EventsEmit(eventName, doneMsg);
  });
}

// ========== 股票相关 ==========

export function GetGroupList() {
  if (isWailsMode()) {
    return window.go.main.App.GetGroupList();
  }
  return apiService.client.get('/groups', { headers: getAuthHeaders() })
    .then(extractApiData)
    .catch(err => {
      console.warn('GetGroupList web fallback failed:', err.message);
      return [];
    });
}

export function GetStockList(keyword) {
  if (isWailsMode()) {
    return window.go.main.App.GetStockList(keyword || '');
  }
  if (!keyword) {
    return Promise.resolve([]);
  }
  const params = { keyword };
  return apiService.client.get('/stocks/search', { params, headers: getAuthHeaders() })
    .then(res => {
      if (res.data && res.data.data) {
        return res.data.data
      }
      return []
    })
    .catch(err => {
      console.warn('GetStockList web fallback failed:', err.message);
      return [];
    });
}

export function GetFollowList(groupId) {
  if (isWailsMode()) {
    return window.go.main.App.GetFollowList(groupId || 0);
  }
  const params = groupId ? { groupId } : {};
  return apiService.client.get('/stocks/follow/list', { params, headers: getAuthHeaders() })
    .then(extractApiData)
    .catch(err => {
      console.warn('GetFollowList web fallback failed:', err.message);
      return [];
    });
}

export function Follow(code) {
  if (isWailsMode()) {
    return window.go.main.App.Follow(code);
  }
  return apiService.client.post('/stocks/follow', { stockCode: code }, { headers: getAuthHeaders() })
    .then(res => res.data?.message || res.data?.data || '关注成功')
    .catch(err => err.message || '关注失败');
}

export function UnFollow(code) {
  if (isWailsMode()) {
    return window.go.main.App.UnFollow(code);
  }
  return apiService.client.delete('/stocks/unfollow', {
    data: { stockCode: code },
    headers: getAuthHeaders()
  })
    .then(res => {
      // 根据 API 响应结构适配返回值
      if (res.data?.code === 0 || res.data?.success) {
        return res.data?.message || res.data?.data || '取消关注成功';
      } else {
        // 如果后端返回了具体错误信息，直接使用
        const errorMsg = res.data?.message || res.data?.error || '取消关注失败';
        console.error('取消关注失败:', errorMsg);
        throw new Error(errorMsg);
      }
    })
    .catch(err => {
      // 提取更详细的错误信息
      if (err.response) {
        // 服务器返回了错误响应
        if (err.response.data?.message) {
          console.error('API错误:', err.response.data.message);
          throw new Error(err.response.data.message);
        } else if (err.response.data?.error) {
          console.error('API错误:', err.response.data.error);
          throw new Error(err.response.data.error);
        } else {
          const errorMsg = `取消关注失败: ${err.response.status} ${err.response.statusText}`;
          console.error('API错误:', errorMsg);
          throw new Error(errorMsg);
        }
      } else if (err.request) {
        // 请求已发出但没有收到响应
        const errorMsg = '网络错误，无法连接到服务器';
        console.error('网络错误:', errorMsg);
        throw new Error(errorMsg);
      } else {
        // 其他错误
        const errorMsg = err.message || '取消关注失败';
        console.error('请求错误:', errorMsg);
        throw new Error(errorMsg);
      }
    });
}

export function Greet(code) {
  if (isWailsMode()) {
    return window.go.main.App.Greet(code);
  }
  // Web 模式下获取股票实时数据 - API 期望 codes 参数
  return apiService.client.get('/stocks/realtime', { params: { codes: code }, headers: getAuthHeaders() })
    .then(res => {
      if (res.data?.code === 0 && Array.isArray(res.data.data) && res.data.data.length > 0) {
        return res.data.data[0];
      }
      return {};
    })
    .catch(err => {
      console.warn('Greet web fallback failed:', err.message);
      return {};
    });
}

export function GetConfig() {
  if (isWailsMode()) {
    return window.go.main.App.GetConfig();
  }
  return apiService.client.get('/config', { headers: getAuthHeaders() })
    .then(res => res.data?.data)
    .catch(err => {
      console.warn('GetConfig web fallback failed:', err.message);
      return {};
    });
}

export function GetPromptTemplates(type, keyword) {
  if (isWailsMode()) {
    return window.go.main.App.GetPromptTemplates(type || '', keyword || '');
  }
  const params = {};
  if (type) params.type = type;
  if (keyword) params.keyword = keyword;
  return apiService.client.get('/prompts/templates', { params, headers: getAuthHeaders() })
    .then(extractApiData)
    .catch(err => {
      console.warn('GetPromptTemplates web fallback failed:', err.message);
      return [];
    });
}

export function GetAiConfigs() {
  if (isWailsMode()) {
    return window.go.main.App.GetAiConfigs();
  }
  return apiService.client.get('/public/ai/configs', { headers: getAuthHeaders() })
    .then(extractApiData)
    .catch(err => {
      console.warn('GetAiConfigs web fallback failed:', err.message);
      return [];
    });
}

export function GetStockKLine(code, name, days) {
  if (isWailsMode()) {
    return window.go.main.App.GetStockKLine(code, name, days);
  }
  return apiService.client.get(`/stocks/${code}/kline`, { params: { name, days }, headers: getAuthHeaders() })
    .then(extractApiData)
    .catch(err => {
      console.warn('GetStockKLine web fallback failed:', err.message);
      return [];
    });
}

export function GetStockMinutePriceLineData(code, name) {
  if (isWailsMode()) {
    return window.go.main.App.GetStockMinutePriceLineData(code, name);
  }
  // Web 模式下暂不支持分时数据
  return Promise.resolve({ priceData: [] });
}

export function SetCostPriceAndVolume(code, price, volume) {
  if (isWailsMode()) {
    return window.go.main.App.SetCostPriceAndVolume(code, price, volume);
  }
  return apiService.client.post('/stocks/cost', { code, price, volume }, { headers: getAuthHeaders() })
    .then(res => res.data?.message || res.data?.data || '设置成功')
    .catch(err => err.message || '设置失败');
}

export function SetAlarmChangePercent(alarm, alarmPrice, code) {
  if (isWailsMode()) {
    return window.go.main.App.SetAlarmChangePercent(alarm, alarmPrice, code);
  }
  return apiService.client.post('/stocks/alarm', { alarm, alarmPrice, code }, { headers: getAuthHeaders() })
    .then(res => res.data?.message || res.data?.data || '设置成功')
    .catch(err => err.message || '设置失败');
}

export function SetStockSort(sort, code) {
  if (isWailsMode()) {
    return window.go.main.App.SetStockSort(sort, code);
  }
  return apiService.client.post('/stocks/sort', { sort, code }, { headers: getAuthHeaders() })
    .then(res => res.data?.message || res.data?.data || '设置成功')
    .catch(err => err.message || '设置失败');
}

export function SetStockAICron(cron, code) {
  if (isWailsMode()) {
    return window.go.main.App.SetStockAICron(cron, code);
  }
  return Promise.resolve('设置成功');
}

export function SetTradingPrice(code, entryPrice, takeProfitPrice, stopLossPrice, costPrice) {
  if (isWailsMode()) {
    return window.go.main.App.SetTradingPrice(code, entryPrice, takeProfitPrice, stopLossPrice, costPrice);
  }
  return Promise.resolve('设置成功');
}

export function AddGroup(name) {
  if (isWailsMode()) {
    return window.go.main.App.AddGroup(name);
  }
  return apiService.client.post('/groups', { name }, { headers: getAuthHeaders() })
    .then(res => res.data?.message || res.data?.data || '添加成功')
    .catch(err => err.message || '添加失败');
}

export function RemoveGroup(id) {
  if (isWailsMode()) {
    return window.go.main.App.RemoveGroup(id);
  }
  return Promise.resolve('删除成功');
}

export function AddStockGroup(groupId, stockCode) {
  if (isWailsMode()) {
    return window.go.main.App.AddStockGroup(groupId, stockCode);
  }
  return apiService.client.post(`/groups/${groupId}/stocks`, { stockCode }, { headers: getAuthHeaders() })
    .then(res => res.data?.message || res.data?.data || '添加成功')
    .catch(err => err.message || '添加失败');
}

export function RemoveStockGroup(groupId, stockCode, index) {
  if (isWailsMode()) {
    return window.go.main.App.RemoveStockGroup(groupId, stockCode, index);
  }
  return Promise.resolve('删除成功');
}

export function UpdateGroupSort(id, newSort) {
  if (isWailsMode()) {
    return window.go.main.App.UpdateGroupSort(id, newSort);
  }
  return apiService.client.put(`/groups/${id}/sort`, { newSort }, { headers: getAuthHeaders() })
    .then(res => res.data?.code === 0)
    .catch(() => false);
}

export function InitializeGroupSort() {
  if (isWailsMode()) {
    return window.go.main.App.InitializeGroupSort();
  }
  return Promise.resolve(true);
}

export function GetGroupStockList(groupId) {
  if (isWailsMode()) {
    return window.go.main.App.GetGroupStockList(groupId);
  }
  return apiService.client.get(`/groups/${groupId}/stocks`, { headers: getAuthHeaders() })
    .then(extractApiData)
    .catch(err => {
      console.warn('GetGroupStockList web fallback failed:', err.message);
      return [];
    });
}

// ========== AI 相关 ==========

/**
 * NewChatStream - 个股 AI 分析（流式）
 * Web 模式下：使用 SSE 连接 /api/v1/ai/analyze，通过 EventsEmit('newChatStream') 分发事件
 * 消息格式与 Wails 桌面模式保持一致：
 *   - { chatId: string }  → 首条消息，通知 chatId
 *   - { content: string } → 流式文字内容
 *   - 'DONE'             → 完成信号
 */
export function NewChatStream(stock, stockCode, question, aiConfigId, sysPromptId, enableTools, think) {
  if (isWailsMode()) {
    return window.go.main.App.NewChatStream(stock, stockCode, question, aiConfigId, sysPromptId, enableTools, think);
  }
  // Web 模式：建立 SSE 连接，事件通过 EventsEmit('newChatStream') 分发
  const baseURL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
  const token = localStorage.getItem('token');
  const origin = window.location.origin;
  const fullUrl = baseURL.startsWith('http') ? `${baseURL}/ai/analyze` : `${origin}${baseURL}/ai/analyze`;

  fetch(fullUrl, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({
      stockCode: stockCode,
      stockName: stock,
      question: question || '',
      aiConfigId: Number(aiConfigId) || 0,
      sysPromptId: sysPromptId != null ? Number(sysPromptId) : null,
    }),
  }).then(response => {
    if (!response.ok) {
      EventsEmit('newChatStream', 'DONE');
      return;
    }
    const reader = response.body.getReader();
    const decoder = new TextDecoder('utf-8');
    let buffer = '';
    let eventType = '';

    function processLine(line) {
      if (!line.trim()) {
        eventType = '';
        return;
      }
      if (line.startsWith('event:')) {
        eventType = line.slice(6).trim();
        return;
      }
      if (line.startsWith('data:')) {
        const raw = line.slice(5).trim();
        if (eventType === 'chat_id') {
          // 收到 chatId
          EventsEmit('newChatStream', { chatId: raw });
        } else if (eventType === 'done' || (eventType === '' && raw.startsWith('{'))) {
          // 完成信号
          EventsEmit('newChatStream', 'DONE');
        } else if (raw) {
          // 普通文字片段（包装为对象，与 Wails 桌面端消息格式一致）
          EventsEmit('newChatStream', { content: raw });
        }
      }
    }

    function read() {
      reader.read().then(({ done, value }) => {
        if (done) {
          if (buffer.trim()) processLine(buffer);
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
        EventsEmit('newChatStream', 'DONE');
      });
    }
    read();
  }).catch(err => {
    console.error('NewChatStream SSE connect error:', err);
    EventsEmit('newChatStream', 'DONE');
  });

  return Promise.resolve();
}

export function SaveAIResponseResult(code, name, result, chatId, question, aiConfigId) {
  if (isWailsMode()) {
    return window.go.main.App.SaveAIResponseResult(code, name, result, chatId, question, aiConfigId);
  }
  // Web 模式下 AI 结果已由后端自动保存，前端无需重复保存
  return Promise.resolve(true);
}

export function GetAIResponseResult(id) {
  if (isWailsMode()) {
    return window.go.main.App.GetAIResponseResult(id);
  }
  return apiService.client.get('/ai/responses', { params: { stockCode: id }, headers: getAuthHeaders() })
    .then(res => {
      const d = res.data?.data;
      if (d && d.list && d.list.length > 0) return d.list[0];
      if (Array.isArray(d) && d.length > 0) return d[0];
      return null;
    })
    .catch(() => null);
}

export function GetAIResponseResultList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetAIResponseResultList(arg1);
  return apiService.client.get('/ai/responses', { params: arg1, headers: getAuthHeaders() })
    .then(res => {
      const d = res.data?.data;
      if (d && d.list) return d;
      if (Array.isArray(d)) return { list: d, total: d.length };
      return { list: [], total: 0 };
    })
    .catch(() => ({ list: [], total: 0 }));
}

export function SaveAsMarkdown(filename, content) {
  if (isWailsMode()) {
    return window.go.main.App.SaveAsMarkdown(filename, content);
  }
  // Web 模式下下载 markdown 文件
  const blob = new Blob([content], { type: 'text/markdown' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename || 'export.md';
  a.click();
  URL.revokeObjectURL(url);
  return Promise.resolve(true);
}

export function SaveImage(filename, imageData) {
  if (isWailsMode()) {
    return window.go.main.App.SaveImage(filename, imageData);
  }
  return Promise.resolve(true);
}

export function SaveWordFile(filename, content) {
  if (isWailsMode()) {
    return window.go.main.App.SaveWordFile(filename, content);
  }
  return Promise.resolve(true);
}

export function ShareAnalysis(code, name) {
  if (isWailsMode()) {
    return window.go.main.App.ShareAnalysis(code, name);
  }
  return apiService.client.post('/share/analysis', { text: code, title: name }, { headers: getAuthHeaders() })
    .then(res => res.data?.data || '')
    .catch(() => '');
}

// ========== 通知相关 ==========

export function SendDingDingMessageByType(msg, code, type) {
  if (isWailsMode()) {
    return window.go.main.App.SendDingDingMessageByType(msg, code, type);
  }
  return Promise.resolve('发送成功');
}

export function SendDingDingMessage(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.SendDingDingMessage(arg1, arg2);
  return apiService.client.post('/dingding/message', { message: arg1, stockCode: arg2 }, { headers: getAuthHeaders() })
    .then(res => res.data?.data || '发送成功')
    .catch(() => '发送成功');
}

// ========== 版本信息 ==========

export function GetVersionInfo() {
  if (isWailsMode()) {
    return window.go.main.App.GetVersionInfo();
  }
  return Promise.resolve({ icon: '', version: 'web' });
}

export function GetEffectiveSponsorVip() {
  if (isWailsMode()) {
    return window.go.main.App.GetEffectiveSponsorVip();
  }
  return Promise.resolve({ active: false, vipLevel: 0 });
}

export function OpenURL(url) {
  if (isWailsMode()) {
    return window.go.main.App.OpenURL(url);
  }
  window.open(url, '_blank');
  return Promise.resolve(true);
}

// ========== 事件相关 (Web 模式下使用 CustomEvent) ==========

export function EventsOn(eventName, callback) {
  if (isWailsMode()) {
    return window.runtime.EventsOn(eventName, callback);
  }
  // Web 模式下使用 CustomEvent
  const handler = (e) => callback(e.detail);
  window.addEventListener('web-event-' + eventName, handler);
  return () => window.removeEventListener('web-event-' + eventName, handler);
}

export function EventsOff(eventName) {
  if (isWailsMode()) {
    return window.runtime.EventsOff(eventName);
  }
  // Web 模式下不需要处理（每次 EventsOn 返回的 cleanup 函数负责移除）
}

export function EventsEmit(eventName, data) {
  if (isWailsMode()) {
    return window.runtime.EventsEmit(eventName, data);
  }
  // Web 模式下使用 CustomEvent
  window.dispatchEvent(new CustomEvent('web-event-' + eventName, { detail: data }));
}

// ========== 窗口相关 ==========

export function WindowFullscreen() {
  if (isWailsMode()) {
    return window.runtime.WindowFullscreen();
  }
  document.documentElement.requestFullscreen?.();
}

export function WindowUnfullscreen() {
  if (isWailsMode()) {
    return window.runtime.WindowUnfullscreen();
  }
  document.exitFullscreen?.();
}

export function WindowReload() {
  if (isWailsMode()) {
    return window.runtime.WindowReload();
  }
  window.location.reload();
}

export function Environment() {
  if (isWailsMode()) {
    return window.runtime.Environment();
  }
  return Promise.resolve({ platform: 'web', buildType: 'web' });
}

// ========== 额外函数 (按字母顺序) ==========

export function AbortChatWithAgent() {
  if (isWailsMode()) return window.go.main.App.AbortChatWithAgent();
  // Web 模式下通过标志位中断，在 ChatWithAgent 中处理
  window._abortAgentStream = true;
  return Promise.resolve();
}

export function AbortSummaryStockNews() {
  if (isWailsMode()) return window.go.main.App.AbortSummaryStockNews();
  window._abortNewsStream = true;
  return Promise.resolve();
}

export function AddPrompt(arg1) {
  if (isWailsMode()) return window.go.main.App.AddPrompt(arg1);
  return Promise.resolve('添加成功');
}

export function AddPromptTemplate(arg1) {
  if (isWailsMode()) return window.go.main.App.AddPromptTemplate(arg1);
  return apiService.client.post('/prompts/templates', arg1, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '添加成功')
    .catch(err => err.message || '添加失败');
}

export function AddTradingRecord(arg1) {
  if (isWailsMode()) return window.go.main.App.AddTradingRecord(arg1);
  return apiService.client.post('/trades', arg1, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '添加成功')
    .catch(err => err.message || '添加失败');
}

export function AnalyzeSentimentWithFreqWeight(arg1) {
  if (isWailsMode()) return window.go.main.App.AnalyzeSentimentWithFreqWeight(arg1);
  return apiService.client.get('/market/sentiment', { params: { text: arg1 }, headers: getAuthHeaders() })
    .then(res => {
      const d = res.data?.data || {};
      return {
        frequencies: [],
        result: { Score: d.score || 0 },
        score: d.score || 0,
        category: d.category || 0,
        PositiveCount: 0,
        NegativeCount: 0,
      };
    })
    .catch(() => ({ frequencies: [], result: { Score: 0 }, score: 0, category: 0, PositiveCount: 0, NegativeCount: 0 }));
}

export function CalculateNextRunTime(arg1) {
  if (isWailsMode()) return window.go.main.App.CalculateNextRunTime(arg1);
  return Promise.resolve('');
}

export function CalculateNextRunTimes(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.CalculateNextRunTimes(arg1, arg2);
  return Promise.resolve([]);
}

/**
 * ChatWithAgent - Agent 模式流式对话
 * Web 模式下：使用 SSE 连接 /api/v1/public/ai/agent-chat（无需认证）
 * 事件通过 EventsEmit('agent-message') 分发，格式与桌面模式一致
 */
export function ChatWithAgent(arg1, arg2, arg3, arg4, arg5, arg6) {
  if (isWailsMode()) return window.go.main.App.ChatWithAgent(arg1, arg2, arg3, arg4, arg5, arg6);

  // 重置中断标志
  window._abortAgentStream = false;

  const baseURL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
  const token = localStorage.getItem('token');
  const origin = window.location.origin;
  const fullUrl = baseURL.startsWith('http')
    ? `${baseURL}/public/ai/agent-chat`
    : `${origin}${baseURL}/public/ai/agent-chat`;

  fetch(fullUrl, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({
      question: arg1,
      aiConfigId: arg2 || 0,
      sysPromptId: arg3 || null,
      memoryMode: arg4,
      memoryCount: arg5 || 10,
      thinking: arg6 || false,
    }),
  }).then(response => {
    if (!response.ok) {
      EventsEmit('agent-message', { content: 'agent-DONE' });
      return;
    }
    const reader = response.body.getReader();
    const decoder = new TextDecoder('utf-8');
    let buffer = '';
    let eventType = '';

    function processLine(line) {
      if (window._abortAgentStream) return;
      if (!line.trim()) { eventType = ''; return; }
      if (line.startsWith('event:')) { eventType = line.slice(6).trim(); return; }
      if (line.startsWith('data:')) {
        const raw = line.slice(5).trim();
        try {
          const parsedData = JSON.parse(raw);
          // 如果解析成功，检查是否为完成信号
          if (parsedData && (parsedData.done || parsedData.finish_reason === 'stop' || (parsedData.chatId && eventType === 'done'))) {
            EventsEmit('agent-message', { content: 'agent-DONE' });
          } else {
            // 对于普通内容，也发送给前端进行处理
            EventsEmit('agent-message', parsedData);
          }
        } catch (e) {
          // 如果不是有效的JSON字符串
          if (eventType === 'done' || raw === '{}') {
            EventsEmit('agent-message', { content: 'agent-DONE' });
          } else if (raw) {
            // 直接发送原始内容，前端的onAgentMessage会处理role
            EventsEmit('agent-message', { content: raw });
          }
        }
      }
    }

    function read() {
      if (window._abortAgentStream) {
        reader.cancel();
        EventsEmit('agent-message', { content: 'agent-DONE' });
        return;
      }
      reader.read().then(({ done, value }) => {
        if (done) {
          if (buffer.trim()) processLine(buffer);
          EventsEmit('agent-message', { content: 'agent-DONE' });
          return;
        }
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop();
        lines.forEach(processLine);
        read();
      }).catch(err => {
        console.error('ChatWithAgent SSE read error:', err);
        EventsEmit('agent-message', { content: 'agent-DONE' });
      });
    }
    read();
  }).catch(err => {
    console.error('ChatWithAgent SSE connect error:', err);
    EventsEmit('agent-message', { content: 'agent-DONE' });
  });

  return Promise.resolve();
}

export function CheckFrequentTrading(arg1) {
  if (isWailsMode()) return window.go.main.App.CheckFrequentTrading(arg1);
  return apiService.client.get('/trades/check-frequent', { params: arg1, headers: getAuthHeaders() })
    .then(res => res.data?.data || false)
    .catch(() => false);
}

export function CheckSponsorCode(arg1) {
  if (isWailsMode()) return window.go.main.App.CheckSponsorCode(arg1);
  return Promise.resolve({ valid: false });
}

export function CheckUpdate(arg1) {
  if (isWailsMode()) return window.go.main.App.CheckUpdate(arg1);
  return Promise.resolve({ hasUpdate: false });
}

export function ClsCalendar() {
  if (isWailsMode()) return window.go.main.App.ClsCalendar();
  return apiService.client.get('/calendar/cls', { headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function CreateCronTask(arg1) {
  if (isWailsMode()) return window.go.main.App.CreateCronTask(arg1);
  return Promise.resolve(true);
}

export function DeleteAIResponseResult(arg1) {
  if (isWailsMode()) return window.go.main.App.DeleteAIResponseResult(arg1);
  return apiService.client.delete(`/ai/responses/${arg1}`, { headers: getAuthHeaders() })
    .then(() => true)
    .catch(() => false);
}

export function DeleteCronTask(arg1) {
  if (isWailsMode()) return window.go.main.App.DeleteCronTask(arg1);
  return Promise.resolve(true);
}

export function DeletePromptTemplate(arg1) {
  if (isWailsMode()) return window.go.main.App.DeletePromptTemplate(arg1);
  return apiService.client.delete(`/prompts/templates/${arg1}`, { headers: getAuthHeaders() })
    .then(() => true)
    .catch(() => false);
}

export function DeleteTradingRecord(arg1) {
  if (isWailsMode()) return window.go.main.App.DeleteTradingRecord(arg1);
  return apiService.client.delete(`/trades/${arg1}`, { headers: getAuthHeaders() })
    .then(() => true)
    .catch(() => false);
}

export function DelPrompt(arg1) {
  if (isWailsMode()) return window.go.main.App.DelPrompt(arg1);
  return Promise.resolve('删除成功');
}

export function EMDictCode(arg1) {
  if (isWailsMode()) return window.go.main.App.EMDictCode(arg1);
  return apiService.client.get(`/market/em-dict/${arg1}`, { headers: getAuthHeaders() })
    .then(res => res.data?.data || arg1)
    .catch(() => arg1);
}

export function EnableCronTask(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.EnableCronTask(arg1, arg2);
  return Promise.resolve(true);
}

export function ExecuteCronTaskNow(arg1) {
  if (isWailsMode()) return window.go.main.App.ExecuteCronTaskNow(arg1);
  return Promise.resolve(true);
}

export function ExportConfig() {
  if (isWailsMode()) return window.go.main.App.ExportConfig();
  return Promise.resolve('');
}

export function FetchAiModels(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.FetchAiModels(arg1, arg2);
  return apiService.client.get('/ai/models', { params: { baseUrl: arg1, apiKey: arg2 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function FollowFund(arg1) {
  if (isWailsMode()) return window.go.main.App.FollowFund(arg1);
  return apiService.client.post('/funds/follow', { code: arg1 }, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '关注成功')
    .catch(err => err.message || '关注失败');
}

export function GlobalStockIndexes() {
  if (isWailsMode()) return window.go.main.App.GlobalStockIndexes();
  return apiService.client.get('/market/global-indexes', { headers: getAuthHeaders() })
    .then(res => res.data?.data || {})
    .catch(() => ({}));
}

export function GetAiAssistantSession(arg1) {
  if (isWailsMode()) return window.go.main.App.GetAiAssistantSession(arg1);
  return Promise.resolve([]);
}

export function GetAiRecommendStocksList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetAiRecommendStocksList(arg1);
  // Web 模式：调用后端 API
  return apiService.client.get('/ai/recommend-stocks', { params: arg1, headers: getAuthHeaders() })
    .then(res => {
      const d = res.data?.data;
      if (d && d.list) return d;
      if (Array.isArray(d)) return { list: d, total: d.length };
      return { list: [], total: 0 };
    })
    .catch(() => ({ list: [], total: 0 }));
}

export function DeleteAiRecommendStocks(id) {
  if (isWailsMode()) return window.go.main.App.DeleteAiRecommendStocks(id);
  // Web 模式：调用后端 API
  return apiService.client.delete(`/ai/recommend-stocks/${id}`, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '删除成功')
    .catch(err => err.message || '删除失败');
}

export function UpdateAiRecommendStocksAlert(id, enableAlert) {
  if (isWailsMode()) return window.go.main.App.UpdateAiRecommendStocksAlert(id, enableAlert);
  // Web 模式：调用后端 API
  return apiService.client.put('/ai/recommend-stocks/alert', { id, enableAlert }, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '更新成功')
    .catch(err => err.message || '更新失败');
}

export function GetAllConcepts() {
  if (isWailsMode()) return window.go.main.App.GetAllConcepts();
  return Promise.resolve([]);
}

export function GetAllIndustries() {
  if (isWailsMode()) return window.go.main.App.GetAllIndustries();
  return Promise.resolve([]);
}

export function GetAllMarkets() {
  if (isWailsMode()) return window.go.main.App.GetAllMarkets();
  return Promise.resolve([]);
}

export function GetStockChanges(arg1, arg2, arg3) {
  if (isWailsMode()) return window.go.main.App.GetStockChanges(arg1, arg2, arg3);
  // Web 模式：调用后端 API
  const changeTypes = Array.isArray(arg1) ? arg1 : [];
  const pageIndex = arg2 || 0;
  const pageSize = arg3 || 50;
  
  const params = {
    pageIndex,
    pageSize,
  };
  if (changeTypes.length > 0) {
    params.changeTypes = changeTypes.join(',');
  }
  
  return apiService.client.get('/stock-changes', { params, headers: getAuthHeaders() })
    .then(res => {
      if (res.data) {
        return { data: res.data.data || [], totalCount: res.data.totalCount || 0 }
      }
      return { data: [], totalCount: 0 }
    })
    .catch(() => ({ data: [], totalCount: 0 }));
}

export function GetAllStockChangesWithPaging(arg1) {
  if (isWailsMode()) return window.go.main.App.GetAllStockChangesWithPaging(arg1);
  // Web 模式：调用后端 API
  const pageSize = arg1 || 500;
  return apiService.client.get('/stock-changes/all', { params: { pageSize }, headers: getAuthHeaders() })
    .then(res => res.data?.data || { data: [], totalCount: 0 })
    .catch(() => ({ data: [], totalCount: 0 }));
}

export function GetStockChangeHistory(arg1) {
  if (isWailsMode()) return window.go.main.App.GetStockChangeHistory(arg1);
  // Web 模式：调用后端 API
  return apiService.client.get('/stock-changes/history', { params: arg1, headers: getAuthHeaders() })
    .then(res => {
      const d = res.data?.data;
      if (d && d.list) return d;
      if (Array.isArray(d)) return { list: d, total: d.length };
      return { list: [], total: 0 };
    })
    .catch(() => ({ list: [], total: 0 }));
}

export function GetAllStockInfoById(arg1) {
  if (isWailsMode()) return window.go.main.App.GetAllStockInfoById(arg1);
  return Promise.resolve(null);
}

export function GetAllStockInfoList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetAllStockInfoList(arg1);
  return apiService.client.get('/stocks/all-info/list', { params: { page: arg1?.page || 1, pageSize: arg1?.pageSize || 50 }, headers: getAuthHeaders() })
    .then(res => res.data?.data?.list || [])
    .catch(() => []);
}

export function GetAllStocks(arg1, arg2, arg3, arg4) {
  if (isWailsMode()) return window.go.main.App.GetAllStocks(arg1, arg2, arg3, arg4);
  return Promise.resolve([]);
}

export function GetCronTaskByID(arg1) {
  if (isWailsMode()) return window.go.main.App.GetCronTaskByID(arg1);
  return Promise.resolve(null);
}

export function GetCronTaskList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetCronTaskList(arg1);
  return Promise.resolve([]);
}

export function GetCronTaskTypes() {
  if (isWailsMode()) return window.go.main.App.GetCronTaskTypes();
  return apiService.client.get('/cron-task/types', { headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetFollowedFund() {
  if (isWailsMode()) return window.go.main.App.GetFollowedFund();
  return apiService.client.get('/funds/follow/list', { headers: getAuthHeaders() })
    .then(extractApiData)
    .catch(() => []);
}

export function GetfundList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetfundList(arg1);
  return apiService.client.get('/funds', { params: arg1, headers: getAuthHeaders() })
    .then(extractApiData)
    .catch(() => []);
}

export function GetHotStrategy() {
  if (isWailsMode()) return window.go.main.App.GetHotStrategy();
  return apiService.client.get('/market/hot-strategy', { headers: getAuthHeaders() })
    .then(res => res.data)
    .catch(() => ({}));
}

export function GetIndustryRank(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.GetIndustryRank(arg1, arg2);
  return apiService.client.get('/market/industry-rank', { params: { sort: arg1, cnt: arg2 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetIndustryMoneyRankSina(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.GetIndustryMoneyRankSina(arg1, arg2);
  return apiService.client.get('/market/industry-money-rank', { params: { fenlei: arg1, sort: arg2 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetMoneyRankSina(arg1) {
  if (isWailsMode()) return window.go.main.App.GetMoneyRankSina(arg1);
  return apiService.client.get('/market/money-rank', { params: { sort: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetStockMoneyTrendByDay(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.GetStockMoneyTrendByDay(arg1, arg2);
  return apiService.client.get(`/market/stocks/${arg1}/money-trend`, { params: { days: arg2 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetPromptTemplateList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetPromptTemplateList(arg1);
  return apiService.client.get('/prompts/templates', { params: arg1, headers: getAuthHeaders() })
    .then(res => res.data?.data || { list: [], total: 0, totalPages: 0 })
    .catch(() => ({ list: [], total: 0, totalPages: 0 }));
}

export function GetSponsorInfo() {
  if (isWailsMode()) return window.go.main.App.GetSponsorInfo();
  return apiService.client.get('/sponsor/info', { headers: getAuthHeaders() })
    .then(res => res.data?.data || { sponsor: false })
    .catch(() => ({ sponsor: false }));
}

export function GetStockEastMoneyKLine(arg1, arg2, arg3, arg4) {
  if (isWailsMode()) return window.go.main.App.GetStockEastMoneyKLine(arg1, arg2, arg3, arg4);
  return apiService.client.get(`/public/stocks/${arg1}/kline`, { params: { klt: arg2, days: arg4 }, headers: getAuthHeaders() })
    .then(res => res.data?.data?.kline || [])
    .catch(() => []);
}

export function GetStockEastMoneyKLinePage(arg1, arg2, arg3, arg4, arg5) {
  if (isWailsMode()) return window.go.main.App.GetStockEastMoneyKLinePage(arg1, arg2, arg3, arg4, arg5);
  return apiService.client.get(`/public/stocks/${arg1}/kline`, { params: { klt: arg2, days: arg4, page: arg5 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || { kline: [], total: 0 })
    .catch(() => ({ kline: [], total: 0 }));
}

export function GetStockRealTimePrice(arg1) {
  if (isWailsMode()) return window.go.main.App.GetStockRealTimePrice(arg1);
  return apiService.client.get('/stocks/realtime', { params: { codes: arg1 }, headers: getAuthHeaders() })
    .then(res => {
      if (res.data?.code === 0 && Array.isArray(res.data.data) && res.data.data.length > 0) {
        return res.data.data[0];
      }
      return {};
    })
    .catch(() => ({}));
}

export function GetTelegraphList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetTelegraphList(arg1);
  return apiService.client.get('/telegraph', { params: { source: arg1 }, headers: getAuthHeaders() })
    .then(extractApiData)
    .catch(() => []);
}

export function GetTradingRecordList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetTradingRecordList(arg1);
  return apiService.client.get('/trades', { params: arg1, headers: getAuthHeaders() })
    .then(extractApiData)
    .catch(() => []);
}

export function GetTradingRecordStatistics() {
  if (isWailsMode()) return window.go.main.App.GetTradingRecordStatistics();
  return apiService.client.get('/trades/statistics', { headers: getAuthHeaders() })
    .then(res => res.data?.data || {})
    .catch(() => ({}));
}

export function GlobalStockIndexesReadable() {
  if (isWailsMode()) return window.go.main.App.GlobalStockIndexesReadable();
  return Promise.resolve([]);
}

export function HotStock(arg1) {
  if (isWailsMode()) return window.go.main.App.HotStock(arg1);
  return apiService.client.get('/market/hot-stock', { params: { marketType: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function HotEvent(arg1) {
  if (isWailsMode()) return window.go.main.App.HotEvent(arg1);
  return apiService.client.get('/market/hot-event', { params: { size: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function HotTopic(arg1) {
  if (isWailsMode()) return window.go.main.App.HotTopic(arg1);
  return apiService.client.get('/market/hot-topic', { params: { size: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function InvestCalendarTimeLine(arg1) {
  if (isWailsMode()) return window.go.main.App.InvestCalendarTimeLine(arg1);
  return apiService.client.get('/calendar/invest', { params: { yearMonth: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function IndustryResearchReport(arg1) {
  if (isWailsMode()) return window.go.main.App.IndustryResearchReport(arg1);
  return apiService.client.get('/research/industry-report', { params: { industryCode: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function LongTigerRank(arg1) {
  if (isWailsMode()) return window.go.main.App.LongTigerRank(arg1);
  return apiService.client.get('/market/long-tiger', { params: { date: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function ReFleshTelegraphList(arg1) {
  if (isWailsMode()) return window.go.main.App.ReFleshTelegraphList(arg1);
  return apiService.client.get('/telegraph/refresh', { params: { source: arg1 }, headers: getAuthHeaders() })
    .then(extractApiData)
    .catch(() => []);
}

export function SaveAiAssistantSession(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.SaveAiAssistantSession(arg1, arg2);
  return Promise.resolve(true);
}

export function SaveStockChangesToHistory(arg1) {
  if (isWailsMode()) return window.go.main.App.SaveStockChangesToHistory(arg1);
  return Promise.resolve(true);
}

export function SearchCronTasks(arg1) {
  if (isWailsMode()) return window.go.main.App.SearchCronTasks(arg1);
  return Promise.resolve([]);
}

export function SearchStock(arg1) {
  if (isWailsMode()) return window.go.main.App.SearchStock(arg1);
  return apiService.client.get('/stocks/indicator-search', { params: { keyword: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data)
    .catch(() => ({}));
}

export function ShareText(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.ShareText(arg1, arg2);
  return apiService.client.post('/share/text', { text: arg1, title: arg2 }, { headers: getAuthHeaders() })
    .then(res => res.data?.data || '')
    .catch(() => '');
}

export function StockNotice(arg1) {
  if (isWailsMode()) return window.go.main.App.StockNotice(arg1);
  return apiService.client.get('/research/stock-notice', { params: { stockCode: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function StockResearchReport(arg1) {
  if (isWailsMode()) return window.go.main.App.StockResearchReport(arg1);
  return apiService.client.get('/research/stock-report', { params: { stockCode: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

/**
 * SummaryStockNews - 市场资讯 AI 总结（流式）
 * Web 模式下复用 /api/v1/ai/analyze 接口，将股票新闻摘要问题传入
 * 事件通过 EventsEmit('summary-news') 分发
 */
export function SummaryStockNews(arg1, arg2, arg3, arg4, arg5, arg6, arg7) {
  if (isWailsMode()) return window.go.main.App.SummaryStockNews(arg1, arg2, arg3, arg4, arg5, arg6, arg7);

  // Wails 参数签名: SummaryStockNews(question, aiConfigId, sysPromptId, enableTools, think, eventName, historyJSON)
  const question = arg1 || '';
  const aiConfigId = Number(arg2) || 0;
  const sysPromptId = arg3 != null ? Number(arg3) : null;
  const eventName = arg6 || 'summary-news';

  window._abortNewsStream = false;

  const baseURL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
  const token = localStorage.getItem('token');
  const origin = window.location.origin;
  const fullUrl = baseURL.startsWith('http')
    ? `${baseURL}/ai/analyze`
    : `${origin}${baseURL}/ai/analyze`;

  fetch(fullUrl, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({
      stockCode: '',
      stockName: '',
      question: question,
      aiConfigId: aiConfigId,
      sysPromptId: sysPromptId,
    }),
  }).then(response => {
    if (!response.ok) {
      EventsEmit(eventName, 'DONE');
      return;
    }
    const reader = response.body.getReader();
    const decoder = new TextDecoder('utf-8');
    let buffer = '';
    let eventType = '';

    function processLine(line) {
      if (window._abortNewsStream) return;
      if (!line.trim()) { eventType = ''; return; }
      if (line.startsWith('event:')) { eventType = line.slice(6).trim(); return; }
      if (line.startsWith('data:')) {
        const raw = line.slice(5).trim();
        if (eventType === 'chat_id') {
          // 忽略 chatId 事件
        } else if (eventType === 'done') {
          EventsEmit(eventName, 'DONE');
        } else if (raw) {
          // 普通文字片段（包装为对象，与 Wails 桌面端消息格式一致）
          EventsEmit(eventName, { content: raw });
        }
      }
    }

    function read() {
      if (window._abortNewsStream) {
        reader.cancel();
        EventsEmit(eventName, 'DONE');
        return;
      }
      reader.read().then(({ done, value }) => {
        if (done) {
          if (buffer.trim()) processLine(buffer);
          EventsEmit(eventName, 'DONE');
          return;
        }
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop();
        lines.forEach(processLine);
        read();
      }).catch(err => {
        console.error('SummaryStockNews SSE read error:', err);
        EventsEmit(eventName, 'DONE');
      });
    }
    read();
  }).catch(err => {
    console.error('SummaryStockNews SSE connect error:', err);
    EventsEmit(eventName, 'DONE');
  });

  return Promise.resolve();
}

export function UnFollowFund(arg1) {
  if (isWailsMode()) return window.go.main.App.UnFollowFund(arg1);
  return apiService.client.delete('/funds/unfollow', { params: { code: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.message || '取消关注成功')
    .catch(err => err.message || '取消关注失败');
}

export function UpdateConfig(arg1) {
  if (isWailsMode()) return window.go.main.App.UpdateConfig(arg1);
  return apiService.client.post('/config', arg1, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '更新成功')
    .catch(err => err.message || '更新失败');
}

export function UpdateCronTask(arg1) {
  if (isWailsMode()) return window.go.main.App.UpdateCronTask(arg1);
  return Promise.resolve(true);
}

export function UpdatePromptTemplate(arg1) {
  if (isWailsMode()) return window.go.main.App.UpdatePromptTemplate(arg1);
  return apiService.client.put(`/prompts/templates/${arg1.ID || arg1}`, arg1, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '更新成功')
    .catch(err => err.message || '更新失败');
}

export function UpdateTradingRecord(arg1) {
  if (isWailsMode()) return window.go.main.App.UpdateTradingRecord(arg1);
  return apiService.client.put(`/trades/${arg1.ID || arg1}`, arg1, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '更新成功')
    .catch(err => err.message || '更新失败');
}

export function ValidateCronExpr(arg1) {
  if (isWailsMode()) return window.go.main.App.ValidateCronExpr(arg1);
  return Promise.resolve({ valid: true });
}

export function BrowserOpenURL(url) {
  if (isWailsMode()) return window.runtime.BrowserOpenURL(url);
  window.open(url, '_blank');
}
