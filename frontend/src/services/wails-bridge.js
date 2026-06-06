// wails-bridge.js - 桥接 Wails 和 Web API，使应用同时支持桌面模式和浏览器模式

import apiService from './api.js';

// 检测是否在 Wails 环境中
function isWailsMode() {
  try {
    return !!(typeof window !== 'undefined' && window.go && window.go.main && window.go.main.App);
  } catch (e) {
    return false;
  }
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
    .then(res => {
      console.log('[GetFollowList] Raw response:', res);
      console.log('[GetFollowList] Response data:', res.data);
      const extracted = extractApiData(res);
      console.log('[GetFollowList] Extracted data:', extracted);
      return extracted;
    })
    .catch(err => {
      console.error('[GetFollowList] Error:', err);
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
  console.log('[Greet] Requesting realtime data for code:', code);
  // Web端：调用实时行情API，获取单只股票数据
  return apiService.client.get('/stocks/realtime', { params: { codes: code }, headers: getAuthHeaders() })
    .then(res => {
      console.log('[Greet] Raw response:', res);
      console.log('[Greet] Response data:', res.data);
      
      if (res.data && res.data.code === 0 && res.data.data) {
        // 后端返回的是数组格式 [{...}]
        const stockList = Array.isArray(res.data.data) ? res.data.data : [];
        console.log('[Greet] Stock list:', stockList);
        
        if (stockList.length > 0) {
          const stockInfo = stockList[0];
          console.log('[Greet] First stock info:', stockInfo);
          
          // 转换为前端期望的格式（与桌面端保持一致）
          // 注意：后端StockInfo结构体使用中文JSON标签
          const result = {
            '股票代码': stockInfo['股票代码'] || code,
            '股票名称': stockInfo['股票名称'] || '',
            '当前价格': stockInfo['当前价格'] || 0,
            '今日最高价': stockInfo['今日最高价'] || 0,
            '今日最低价': stockInfo['今日最低价'] || 0,
            '今日开盘价': stockInfo['今日开盘价'] || 0,
            '昨日收盘价': stockInfo['昨日收盘价'] || 0,
            '上次当前价格': stockInfo['上次当前价格'] || 0,
            changePercent: stockInfo['changePercent'] || 0,
            '涨跌值': stockInfo['changePrice'] || 0,
            '成交量': stockInfo['成交的股票数'] || 0,
            '成交额': stockInfo['成交金额'] || 0,
            '买一报价': stockInfo['买一报价'] || 0,
            '卖一报价': stockInfo['卖一报价'] || 0,
            '买一申报': stockInfo['买一申报'] || 0,
            '卖一申报': stockInfo['卖一申报'] || 0,
            '买二报价': stockInfo['买二报价'] || 0,
            '卖二报价': stockInfo['卖二报价'] || 0,
            '买三报价': stockInfo['买三报价'] || 0,
            '卖三报价': stockInfo['卖三报价'] || 0,
            '买四报价': stockInfo['买四报价'] || 0,
            '卖四报价': stockInfo['卖四报价'] || 0,
            '买五报价': stockInfo['买五报价'] || 0,
            '卖五报价': stockInfo['卖五报价'] || 0,
            sort: stockInfo['sort'] || 999,
            costPrice: stockInfo['costPrice'] || 0,
            volume: stockInfo['costVolume'] || 0,
            profit: stockInfo['profit'] || 0,
            profitAmount: stockInfo['profitAmount'] || 0,
            profitAmountToday: stockInfo['profitAmountToday'] || 0,
            highRate: stockInfo['highRate'] || 0,
            lowRate: stockInfo['lowRate'] || 0,
            date: stockInfo['日期'] || '',
            time: stockInfo['时间'] || '',
            ...stockInfo // 保留其他字段
          };
          console.log('[Greet] Converted result:', result);
          return result;
        }
      }
      console.warn('[Greet] No valid data found, returning empty object');
      return {};
    })
    .catch(err => {
      console.error('[Greet] Error:', err);
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
  return apiService.client.get('/ai/configs', { headers: getAuthHeaders() })
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

export function GetStockKLineWithFallback(stockCode, stockName, klt, limit) {
  if (isWailsMode()) {
    return window.go.main.App.GetStockKLineWithFallback(stockCode, stockName, klt, limit);
  }
  // Web 模式：调用 K 线 API，返回带 source 的结果
  return apiService.client.get(`/stocks/${stockCode}/kline`, {
    params: { name: stockName, klt, limit },
    headers: getAuthHeaders()
  })
    .then(res => {
      const data = res.data?.data || [];
      return { data, source: 'web' };
    })
    .catch(err => {
      console.warn('GetStockKLineWithFallback web fallback failed:', err.message);
      return { data: [], source: 'web' };
    });
}

export function GetStockKLinePageWithFallback(stockCode, stockName, klt, limit, end) {
  if (isWailsMode()) {
    return window.go.main.App.GetStockKLinePageWithFallback(stockCode, stockName, klt, limit, end);
  }
  // Web 模式：分页获取 K 线数据
  return apiService.client.get(`/stocks/${stockCode}/kline/page`, {
    params: { name: stockName, klt, limit, end },
    headers: getAuthHeaders()
  })
    .then(res => {
      const data = res.data?.data || [];
      return { data, source: 'web' };
    })
    .catch(err => {
      console.warn('GetStockKLinePageWithFallback web fallback failed:', err.message);
      return { data: [], source: 'web' };
    });
}

export function GetStockMinutePriceLineData(code, name) {
  if (isWailsMode()) {
    return window.go.main.App.GetStockMinutePriceLineData(code, name);
  }
  let stockCode = code;
  if (code.startsWith('sh') || code.startsWith('sz')) {
    stockCode = code;
  } else if (code.startsWith('hk')) {
    stockCode = 'hk' + code.replace('hk', '');
  } else if (code.startsWith('gb_')) {
    stockCode = code.replace('gb_', '').toUpperCase() + '.OQ';
  } else {
    stockCode = 'sh' + code;
  }
  return fetch(`https://web.ifzq.gtimg.cn/appstock/app/minute/query?code=${stockCode}`)
    .then(res => res.json())
    .then(res => {
      const data = res.data;
      if (!data) return { priceData: [] };
      const stockData = data[stockCode];
      if (!stockData || !stockData.data || !stockData.data.data) return { priceData: [] };
      const minuteData = stockData.data.data;
      const date = stockData.date || stockData.qh_date || stockData.data.qh_date || stockData.data.date || '';
      const priceData = minuteData.map(item => {
        const parts = item.split(' ');
        return {
          time: parts[0],
          price: parseFloat(parts[1]) || 0,
          volume: parseFloat(parts[2]) || 0,
          amount: parseFloat(parts[3]) || 0
        };
      });
      return { priceData, date: date, stockName: name, stockCode: code };
    })
    .catch(err => {
      console.error('GetStockMinutePriceLineData error:', err);
      return { priceData: [] };
    });
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

export function AddGroup(group) {
  if (isWailsMode()) {
    return window.go.main.App.AddGroup(group);
  }
  return apiService.client.post('/groups', { name: group.name, sort: group.sort, desc: group.desc }, { headers: getAuthHeaders() })
    .then(res => res.data?.message || res.data?.data || '添加成功')
    .catch(err => err.message || '添加失败');
}

export function RemoveGroup(id) {
  if (isWailsMode()) {
    return window.go.main.App.RemoveGroup(id);
  }
  return apiService.client.delete(`/groups/${id}`, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '删除成功')
    .catch(err => err.message || '删除失败');
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
  return apiService.client.delete(`/groups/${groupId}/stocks?stockCode=${encodeURIComponent(stockCode)}`, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '移除成功')
    .catch(err => err.message || '移除失败');
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

export function GetUserManual() {
  if (isWailsMode()) {
    return window.go.main.App.GetUserManual();
  }
  // Web 模式：从后端 API 获取用户手册
  return apiService.client.get('/manual', { headers: getAuthHeaders() })
    .then(res => res.data?.data || '')
    .catch(err => {
      console.warn('GetUserManual web fallback failed:', err.message);
      return '';
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
  // Web端：调用 /user/profile 接口获取用户信息和VIP等级
  return apiService.client.get('/user/profile', { headers: getAuthHeaders() })
    .then(res => {
      const user = res.data || {};
      const role = user.role || 'user';
      // 管理员/超级管理员直接返回VIP2
      if (role === 'admin' || role === 'super_admin') {
        return { active: true, vipLevel: 2 };
      }
      // 普通用户：根据vipLevel和vipEndAt判断
      const vipLevel = Number(user.vipLevel ?? 0);
      const vipEndAt = user.vipEndAt || '';
      let active = false;
      if (vipLevel > 0 && vipEndAt) {
        const endDate = new Date(vipEndAt);
        active = endDate > new Date();
      }
      return { active, vipLevel };
    })
    .catch(err => {
      console.error('[GetEffectiveSponsorVip] Error:', err);
      return { active: false, vipLevel: 0 };
    });
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

export function WindowSetTitle(title) {
  if (isWailsMode()) {
    return window.runtime.WindowSetTitle(title);
  }
  // Web 模式下设置文档标题
  document.title = title;
}

export function Quit() {
  if (isWailsMode()) {
    return window.runtime.Quit();
  }
  // Web 模式下无法直接退出，提示用户关闭浏览器窗口
  if (confirm('确定要退出应用吗？')) {
    window.close();
  }
}

export function Hide() {
  if (isWailsMode()) {
    return window.runtime.Hide();
  }
  // Web 模式下隐藏应用（最小化到系统托盘不可用）
  console.log('Web 模式下不支持隐藏功能');
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
  return apiService.client.post('/trades', arg1)
    .then(res => res.data?.message || '添加成功')
    .catch(err => err.message || '添加失败');
}

export function AnalyzeSentimentWithFreqWeight(arg1) {
  if (isWailsMode()) return window.go.main.App.AnalyzeSentimentWithFreqWeight(arg1);
  // Web 模式：直接调 /market/hot-words
  // 后端 GetHotWords 会自动判断 DB 是否有数据，无数据时主动触发爬取
  return apiService.client.get('/market/hot-words', { headers: getAuthHeaders() })
    .then(res => {
      const apiData = res.data || {};
      const d = (apiData.code === 0 ? apiData.data : apiData) || {};
      const resultData = (d.result && typeof d.result.Score === 'number')
        ? d.result
        : { Score: 0, Category: 0, PositiveCount: 0, NegativeCount: 0, Description: '' };
      return {
        frequencies: Array.isArray(d.frequencies) ? d.frequencies : [],
        result: resultData,
      };
    })
    .catch(() => {
      console.error('AnalyzeSentimentWithFreqWeight failed');
      return { frequencies: [], result: { Score: 0 } };
    });
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
 * Web 模式下：使用 SSE 连接 /api/v1/ai/agent-chat
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
    ? `${baseURL}/ai/agent-chat`
    : `${origin}${baseURL}/ai/agent-chat`;

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
  return apiService.client.get('/trades/check-frequent', { params: { stockCode: arg1 } })
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
  return apiService.client.post('/cron-task', arg1, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '创建成功')
    .catch(err => err.message || '创建失败');
}

export function DeleteAIResponseResult(arg1) {
  if (isWailsMode()) return window.go.main.App.DeleteAIResponseResult(arg1);
  return apiService.client.delete(`/ai/responses/${arg1}`, { headers: getAuthHeaders() })
    .then(() => true)
    .catch(() => false);
}

export function DeleteCronTask(arg1) {
  if (isWailsMode()) return window.go.main.App.DeleteCronTask(arg1);
  return apiService.client.delete(`/cron-task/${arg1}`, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '删除成功')
    .catch(err => err.message || '删除失败');
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
  return apiService.client.post(`/cron-task/${arg1}/enable`, { enable: arg2 }, { headers: getAuthHeaders() })
    .then(res => res.data?.message || (arg2 ? '已启用' : '已暂停'))
    .catch(err => err.message || '操作失败');
}

export function ExecuteCronTaskNow(arg1) {
  if (isWailsMode()) return window.go.main.App.ExecuteCronTaskNow(arg1);
  return apiService.client.post(`/cron-task/${arg1}/execute`, {}, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '执行成功')
    .catch(err => err.message || '执行失败');
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

export function FetchAiModelInfo(arg1, arg2, arg3) {
  if (isWailsMode()) return window.go.main.App.FetchAiModelInfo(arg1, arg2, arg3);
  // Web fallback: backend currently exposes model list only, so return a safe default.
  return Promise.resolve({
    modelName: arg3 || '',
    maxTokens: 0,
    source: 'fallback',
  });
}

export function FetchAndSaveMarketStatistic() {
  if (isWailsMode()) return window.go.main.App.FetchAndSaveMarketStatistic();
  return apiService.client.post('/market/statistic/fetch', {}, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '获取成功')
    .catch(err => err.message || '获取失败');
}

export function GetTodayMarketStatistic() {
  if (isWailsMode()) return window.go.main.App.GetTodayMarketStatistic();
  return apiService.client.get('/market/statistic/today', { headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetRecentDaysMarketStatistic(arg1) {
  if (isWailsMode()) return window.go.main.App.GetRecentDaysMarketStatistic(arg1);
  const days = arg1 || 7;
  return apiService.client.get('/market/statistic/recent', { params: { days }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetDailyChangeStats(arg1) {
  if (isWailsMode()) return window.go.main.App.GetDailyChangeStats(arg1);
  return apiService.client.get('/stock-changes/daily-stats', { params: { days: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetChangeTypeDailyStats(arg1) {
  if (isWailsMode()) return window.go.main.App.GetChangeTypeDailyStats(arg1);
  return apiService.client.get('/stock-changes/type-stats', { params: { days: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetDailyDimensionStats(arg1, arg2, arg3) {
  if (isWailsMode()) return window.go.main.App.GetDailyDimensionStats(arg1, arg2, arg3);
  return Promise.resolve([]);
}

export function GetTypeStatsByDate(arg1) {
  if (isWailsMode()) return window.go.main.App.GetTypeStatsByDate(arg1);
  return Promise.resolve([]);
}

export function GetChangeRank(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.GetChangeRank(arg1, arg2);
  return apiService.client.get('/market/change-rank', { params: { days: arg1, topN: arg2 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || { topStocks: [], topIndustries: [], topConcepts: [] })
    .catch(() => ({ topStocks: [], topIndustries: [], topConcepts: [] }));
}

export function GetMarketStatisticByDate(arg1) {
  if (isWailsMode()) return window.go.main.App.GetMarketStatisticByDate(arg1);
  return apiService.client.get('/market/statistic/by-date', { params: { date: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function CreateMCPServer(arg1) {
  if (isWailsMode()) return window.go.main.App.CreateMCPServer(arg1);
  return Promise.resolve({ id: 0 });
}

export function UpdateMCPServer(arg1) {
  if (isWailsMode()) return window.go.main.App.UpdateMCPServer(arg1);
  return Promise.resolve();
}

export function DeleteMCPServer(arg1) {
  if (isWailsMode()) return window.go.main.App.DeleteMCPServer(arg1);
  return Promise.resolve();
}

export function GetMCPServerByID(arg1) {
  if (isWailsMode()) return window.go.main.App.GetMCPServerByID(arg1);
  return Promise.resolve(null);
}

export function GetMCPServerList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetMCPServerList(arg1);
  return Promise.resolve({
    list: [],
    total: 0,
    page: 1,
    pageSize: arg1?.pageSize || 10,
  });
}

export function EnableMCPServer(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.EnableMCPServer(arg1, arg2);
  return Promise.resolve();
}

export function TestMCPServer(arg1) {
  if (isWailsMode()) return window.go.main.App.TestMCPServer(arg1);
  return Promise.resolve({
    success: false,
    message: 'Web mode does not support MCP server test yet',
  });
}

export function GetMCPToolsByServerID(arg1) {
  if (isWailsMode()) return window.go.main.App.GetMCPToolsByServerID(arg1);
  return Promise.resolve([]);
}

export function GetAllMCPTools() {
  if (isWailsMode()) return window.go.main.App.GetAllMCPTools();
  return Promise.resolve([]);
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
  // Web 模式：调用后端 API
  const sessionId = arg1 || '';
  return apiService.client.get('/ai/assistant/session', { 
    params: sessionId ? { sessionId } : {},
    headers: getAuthHeaders() 
  })
    .then(res => res.data?.data || { messages: [], sessionId: '' })
    .catch(() => ({ messages: [], sessionId: '' }));
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
  return apiService.client.get('/stocks/concepts', { headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetAllIndustries() {
  if (isWailsMode()) return window.go.main.App.GetAllIndustries();
  return apiService.client.get('/stocks/industries', { headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetAllMarkets() {
  if (isWailsMode()) return window.go.main.App.GetAllMarkets();
  return apiService.client.get('/stocks/markets', { headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
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
    .then(res => res.data?.data || { list: [], total: 0 })
    .catch(() => ({ list: [], total: 0 }));
}

export function GetAllStocks(arg1, arg2, arg3, arg4) {
  if (isWailsMode()) return window.go.main.App.GetAllStocks(arg1, arg2, arg3, arg4);
  const params = {
    page: arg1 || 1,
    pageSize: arg2 || 50,
    keyword: arg3 || '',
  };
  // 将技术指标对象转换为查询参数
  if (arg4) {
    Object.keys(arg4).forEach(key => {
      if (arg4[key] === true) {
        params[key] = 'true';
      } else if (typeof arg4[key] === 'number' && arg4[key] > 0) {
        params[key] = arg4[key].toString();
      }
    });
  }
  return apiService.client.get('/stocks/all', { params, headers: getAuthHeaders() })
    .then(res => {
      console.log('GetAllStocks web response:', res.data)
      return res.data?.data || { result: { data: [], count: 0 } }
    })
    .catch(() => ({ result: { data: [], count: 0 } }));
}

export function GetCronTaskByID(arg1) {
  if (isWailsMode()) return window.go.main.App.GetCronTaskByID(arg1);
  return apiService.client.get(`/cron-task/${arg1}`, { headers: getAuthHeaders() })
    .then(res => res.data?.data || null)
    .catch(() => null);
}

export function GetCronTaskList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetCronTaskList(arg1);
  const params = {
    page: arg1?.page || 1,
    pageSize: arg1?.pageSize || 10,
  };
  if (arg1?.name) params.name = arg1.name;
  if (arg1?.taskType) params.taskType = arg1.taskType;
  if (arg1?.status && arg1.status !== '') params.status = arg1.status;
  return apiService.client.get('/cron-task', { params, headers: getAuthHeaders() })
    .then(res => res.data?.data || { data: [], total: 0 })
    .catch(() => ({ data: [], total: 0 }));
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
  // arg1=code, arg2=stockName, arg3=klt, arg4=limit
  return apiService.client.get(`/stocks/${arg1}/kline`, { params: { klt: arg3, days: arg4 }, headers: getAuthHeaders() })
    .then(res => res.data?.data?.kline || [])
    .catch(() => []);
}

export function GetStockEastMoneyKLinePage(arg1, arg2, arg3, arg4, arg5) {
  if (isWailsMode()) return window.go.main.App.GetStockEastMoneyKLinePage(arg1, arg2, arg3, arg4, arg5);
  // arg1=code, arg2=stockName, arg3=klt, arg4=limit, arg5=end
  return apiService.client.get(`/stocks/${arg1}/kline`, { params: { klt: arg3, days: arg4, page: arg5 }, headers: getAuthHeaders() })
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
  // Web 模式：调用后端 API
  return apiService.client.get('/trades', { params: arg1, headers: getAuthHeaders() })
    .then(res => {
      const d = res.data?.data;
      if (d && d.list) return d;
      if (Array.isArray(d)) return { list: d, total: d.length };
      return { list: [], total: 0 };
    })
    .catch(() => ({ list: [], total: 0 }));
}

export function GetTradingRecordStatistics() {
  if (isWailsMode()) return window.go.main.App.GetTradingRecordStatistics();
  return apiService.client.get('/trades/statistics')
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
  // Web 模式：调用后端 API
  const sessionId = arg1;
  const messages = arg2;
  return apiService.client.post('/ai/assistant/session', 
    { sessionId, messages }, 
    { headers: getAuthHeaders() }
  )
    .then(res => res.data?.message || '保存成功')
    .catch(err => err.message || '保存失败');
}

export function SaveStockChangesToHistory(arg1) {
  if (isWailsMode()) return window.go.main.App.SaveStockChangesToHistory(arg1);
  const changeTypes = Array.isArray(arg1) ? arg1 : [];
  const params = {};
  if (changeTypes.length > 0) {
    params.changeTypes = changeTypes.join(',');
  }
  return apiService.client.post('/stock-changes/save', {}, { params, headers: getAuthHeaders() })
    .then(res => res.data?.message || '保存成功')
    .catch(err => err.message || '保存失败');
}

export function SearchCronTasks(arg1) {
  if (isWailsMode()) return window.go.main.App.SearchCronTasks(arg1);
  return apiService.client.get('/cron-task/search', { params: { keyword: arg1 || '' }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
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
  return apiService.client.put(`/cron-task/${arg1.ID}`, arg1, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '更新成功')
    .catch(err => err.message || '更新失败');
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
  return apiService.client.get('/cron-task/validate', { params: { expr: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.valid ? 'Cron 表达式有效' : res.data?.message || '无效')
    .catch(err => err.message || '验证失败');
}

export function BrowserOpenURL(url) {
  if (isWailsMode()) return window.runtime.BrowserOpenURL(url);
  window.open(url, '_blank');
}

export function AddAllStockInfo(arg1) {
  if (isWailsMode()) return window.go.main.App.AddAllStockInfo(arg1);
  return Promise.resolve(false);
}

export function AddCronTask(arg1) {
  if (isWailsMode()) return window.go.main.App.AddCronTask(arg1);
  return Promise.resolve('Web mode not supported');
}

export function AnalyzeSentiment(arg1) {
  if (isWailsMode()) return window.go.main.App.AnalyzeSentiment(arg1);
  return Promise.resolve(null);
}

export function BatchDeleteAIResponseResult(arg1) {
  if (isWailsMode()) return window.go.main.App.BatchDeleteAIResponseResult(arg1);
  return Promise.resolve('Web mode not supported');
}

export function BatchDeleteAllStockInfo(arg1) {
  if (isWailsMode()) return window.go.main.App.BatchDeleteAllStockInfo(arg1);
  return Promise.resolve('Web mode not supported');
}

export function CheckStockBaseInfo() {
  if (isWailsMode()) return window.go.main.App.CheckStockBaseInfo();
  return Promise.resolve(false);
}

export function CreateSkill(arg1) {
  if (isWailsMode()) return window.go.main.App.CreateSkill(arg1);
  return Promise.resolve({ id: 0 });
}

export function DeleteAllStockInfo(arg1) {
  if (isWailsMode()) return window.go.main.App.DeleteAllStockInfo(arg1);
  return Promise.resolve(false);
}

export function DeleteSkill(arg1) {
  if (isWailsMode()) return window.go.main.App.DeleteSkill(arg1);
  return Promise.resolve();
}

export function DeleteStockChangeHistory() {
  if (isWailsMode()) return window.go.main.App.DeleteStockChangeHistory();
  return Promise.resolve();
}

export function EnableSkill(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.EnableSkill(arg1, arg2);
  return Promise.resolve();
}

export function GetAllSkills() {
  if (isWailsMode()) return window.go.main.App.GetAllSkills();
  return apiService.client.get('/skills/all', { headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetLatestTradingDay() {
  if (isWailsMode()) return window.go.main.App.GetLatestTradingDay();
  return Promise.resolve('');
}

export function IsTradingDay(date) {
  if (isWailsMode()) return window.go.main.App.IsTradingDay(date);
  return Promise.resolve(false);
}

export function GetSkillByID(arg1) {
  if (isWailsMode()) return window.go.main.App.GetSkillByID(arg1);
  return apiService.client.get(`/skills/${arg1}`, { headers: getAuthHeaders() })
    .then(res => res.data?.data || null)
    .catch(() => null);
}

export function GetSkillList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetSkillList(arg1);
  const params = {
    page: arg1?.page || 1,
    pageSize: arg1?.pageSize || 10,
    name: arg1?.name || '',
    status: arg1?.status || '',
  };
  return apiService.client.get('/skills', { params, headers: getAuthHeaders() })
    .then(res => res.data?.data || { list: [], total: 0 })
    .catch(() => ({ list: [], total: 0 }));
}

export function GetStockCommonKLine(arg1, arg2, arg3) {
  if (isWailsMode()) return window.go.main.App.GetStockCommonKLine(arg1, arg2, arg3);
  return Promise.resolve([]);
}

export function GetTimezone() {
  if (isWailsMode()) return window.go.main.App.GetTimezone();
  return Promise.resolve('Asia/Shanghai');
}

export function GetTradingRecordById(arg1) {
  if (isWailsMode()) return window.go.main.App.GetTradingRecordById(arg1);
  return Promise.resolve(null);
}

export function GetUplimitHot(date, limit = 20) {
  if (isWailsMode()) return window.go.main.App.GetUplimitHot(date, limit);
  return apiService.client.get('/market/uplimit-hot', { params: { date: date || '', limit }, headers: getAuthHeaders() })
    .then(res => res.data)
    .catch(() => null);
}

export function InitCronTasks() {
  if (isWailsMode()) return window.go.main.App.InitCronTasks();
  return Promise.resolve();
}

export function IsTradingTime() {
  if (isWailsMode()) return window.go.main.App.IsTradingTime();
  return Promise.resolve(false);
}

export function IsHKTradingTime() {
  if (isWailsMode()) return window.go.main.App.IsHKTradingTime();
  return Promise.resolve(false);
}

export function IsUSTradingTime() {
  if (isWailsMode()) return window.go.main.App.IsUSTradingTime();
  return Promise.resolve(false);
}

export function NewsPush(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.NewsPush(arg1, arg2);
  return Promise.resolve();
}

export function UpdateSkill(arg1) {
  if (isWailsMode()) return window.go.main.App.UpdateSkill(arg1);
  return Promise.resolve();
}

// ========== 桌面端专用功能（Web 端 no-op） ==========

export function RestartAsAdmin() {
  if (isWailsMode()) return window.go.main.App.RestartAsAdmin();
  // Web 模式下不支持以管理员身份重启
  console.warn('Web 模式下不支持 RestartAsAdmin');
  return Promise.resolve();
}

// ========== 基金相关新增 ==========

export function GetFollowedFundPaged(arg1, arg2, arg3) {
  if (isWailsMode()) return window.go.main.App.GetFollowedFundPaged(arg1, arg2, arg3);
  const params = {
    page: arg1 || 1,
    pageSize: arg2 || 20,
    keyword: arg3 || '',
  };
  return apiService.client.get('/funds/follow/paged', { params, headers: getAuthHeaders() })
    .then(res => res.data?.data || { list: [], total: 0 })
    .catch(() => ({ list: [], total: 0 }));
}

export function SearchFundCodes(arg1) {
  if (isWailsMode()) return window.go.main.App.SearchFundCodes(arg1);
  return apiService.client.get('/funds/search', { params: { keyword: arg1 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetFundKLine(arg1, arg2, arg3) {
  if (isWailsMode()) return window.go.main.App.GetFundKLine(arg1, arg2, arg3);
  return apiService.client.get(`/funds/${arg1}/kline`, { params: { klt: arg2, limit: arg3 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || null)
    .catch(() => null);
}

export function GetFundRanking(arg1, arg2, arg3, arg4, arg5, arg6) {
  if (isWailsMode()) return window.go.main.App.GetFundRanking(arg1, arg2, arg3, arg4, arg5, arg6);
  const params = {
    marketType: arg1 || 'kf',
    fundType: arg2 || 'all',
    sortField: arg3 || 'jnzf',
    sortOrder: arg4 || 'desc',
    pageIndex: arg5 || 1,
    pageSize: arg6 || 50,
  };
  return apiService.client.get('/funds/ranking', { params, headers: getAuthHeaders() })
    .then(res => res.data?.data || { items: [], totalCount: 0 })
    .catch(() => ({ items: [], totalCount: 0 }));
}

export function GetFundHistoryNetValue(arg1, arg2, arg3, arg4, arg5) {
  if (isWailsMode()) return window.go.main.App.GetFundHistoryNetValue(arg1, arg2, arg3, arg4, arg5);
  const params = {
    pageIndex: arg2 || 1,
    pageSize: arg3 || 20,
    startDate: arg4 || '',
    endDate: arg5 || '',
  };
  return apiService.client.get(`/funds/${arg1}/history`, { params, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetFundTop10Holdings(arg1) {
  if (isWailsMode()) return window.go.main.App.GetFundTop10Holdings(arg1);
  return apiService.client.get(`/funds/${arg1}/holdings`, { headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

// ========== 自定义策略相关 ==========

export function GetAllCustomStrategies() {
  if (isWailsMode()) return window.go.main.App.GetAllCustomStrategies();
  return apiService.client.get('/custom-strategies', { headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetCustomStrategyList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetCustomStrategyList(arg1);
  const params = {
    page: arg1?.page || 1,
    pageSize: arg1?.pageSize || 10,
  };
  if (arg1?.name) params.name = arg1.name;
  return apiService.client.get('/custom-strategies/list', { params, headers: getAuthHeaders() })
    .then(res => res.data?.data || { list: [], total: 0 })
    .catch(() => ({ list: [], total: 0 }));
}

export function SaveCustomStrategy(arg1) {
  if (isWailsMode()) return window.go.main.App.SaveCustomStrategy(arg1);
  return apiService.client.post('/custom-strategies', arg1, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '保存成功')
    .catch(err => err.message || '保存失败');
}

export function DeleteCustomStrategy(arg1) {
  if (isWailsMode()) return window.go.main.App.DeleteCustomStrategy(arg1);
  return apiService.client.delete(`/custom-strategies/${arg1}`, { headers: getAuthHeaders() })
    .then(res => res.data?.message || '删除成功')
    .catch(err => err.message || '删除失败');
}

// ========== 筹码分布相关 ==========

export function GetChipDistribution(arg1, arg2, arg3, arg4) {
  if (isWailsMode()) return window.go.main.App.GetChipDistribution(arg1, arg2, arg3, arg4);
  return apiService.client.get(`/stocks/${arg1}/chip-distribution`, { params: { klt: arg2, limit: arg3, priceRange: arg4 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || {})
    .catch(() => ({}));
}

// ========== 通达信数据相关 ==========

export function GetTdxCallAuction(arg1, arg2, arg3) {
  if (isWailsMode()) return window.go.main.App.GetTdxCallAuction(arg1, arg2, arg3);
  return apiService.client.get(`/stocks/${arg1}/tdx/call-auction`, { params: { arg2, arg3 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetTdxCompanyCategoryContent(arg1, arg2) {
  if (isWailsMode()) return window.go.main.App.GetTdxCompanyCategoryContent(arg1, arg2);
  return apiService.client.get(`/stocks/${arg1}/tdx/category-content`, { params: { category: arg2 }, headers: getAuthHeaders() })
    .then(res => res.data?.data || '')
    .catch(() => '');
}

export function GetTdxCompanyCategoryList(arg1) {
  if (isWailsMode()) return window.go.main.App.GetTdxCompanyCategoryList(arg1);
  return apiService.client.get(`/stocks/${arg1}/tdx/categories`, { headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

export function GetTdxCompanyInfo(arg1) {
  if (isWailsMode()) return window.go.main.App.GetTdxCompanyInfo(arg1);
  return apiService.client.get(`/stocks/${arg1}/tdx/company-info`, { headers: getAuthHeaders() })
    .then(res => res.data?.data || {})
    .catch(() => ({}));
}

export function GetTdxFinanceInfo(arg1) {
  if (isWailsMode()) return window.go.main.App.GetTdxFinanceInfo(arg1);
  return apiService.client.get(`/stocks/${arg1}/tdx/finance-info`, { headers: getAuthHeaders() })
    .then(res => res.data?.data || {})
    .catch(() => ({}));
}

export function GetTdxXDXRInfo(arg1) {
  if (isWailsMode()) return window.go.main.App.GetTdxXDXRInfo(arg1);
  return apiService.client.get(`/stocks/${arg1}/tdx/xdxr-info`, { headers: getAuthHeaders() })
    .then(res => res.data?.data || [])
    .catch(() => []);
}

// ========== 桌面端托盘功能（Web 端 no-op） ==========

export function HideToTray() {
  if (isWailsMode()) return window.go.main.App.HideToTray();
  console.warn('Web 模式下不支持 HideToTray');
  return Promise.resolve();
}

export function ShowFromTray() {
  if (isWailsMode()) return window.go.main.App.ShowFromTray();
  console.warn('Web 模式下不支持 ShowFromTray');
  return Promise.resolve();
}

// ========== 设备绑定相关 ==========

export function GetMachineId() {
  if (isWailsMode()) return window.go.main.App.GetMachineId();
  return apiService.client.get('/device/machine-id', { headers: getAuthHeaders() })
    .then(res => res.data?.data || '')
    .catch(() => '');
}

export function CheckDeviceBinding(arg1) {
  if (isWailsMode()) return window.go.main.App.CheckDeviceBinding(arg1);
  return apiService.client.post('/device/check-binding', { machineId: arg1 }, { headers: getAuthHeaders() })
    .then(res => res.data?.data || { bound: false })
    .catch(() => ({ bound: false }));
}

export function QuitApp() {
  if (isWailsMode()) return window.go.main.App.QuitApp();
  // Web 模式下无法直接退出
  if (confirm('确定要退出应用吗？')) {
    window.close();
  }
  return Promise.resolve();
}
