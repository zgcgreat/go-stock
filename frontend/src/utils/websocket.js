// utils/websocket.js
import Auth from './auth';

class WebSocketService {
  constructor(url) {
    this.url = url;
    this.ws = null;
    this.baseReconnectInterval = 1000; // 初始1秒，指数退避增长
    this.maxReconnectInterval = 30000; // 最大30秒
    this.reconnectJitter = 1000;       // 随机抖动 ±500ms
    this.maxReconnectAttempts = 0;     // 0 = 无限重连（由应用层策略控制）
    this.reconnectAttempts = 0;
    this.eventHandlers = {};
    this.isManuallyClosed = false;
  }

  // 连接 WebSocket
  connect() {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return;
    }

    // 在URL中包含认证token
    const token = Auth.getToken();
    let wsUrl = this.url;

    if (token) {
      // 如果WebSocket服务支持认证token，可以通过URL参数传递
      // 或者在连接建立后发送认证消息
      wsUrl = this.url;
    }

    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = (event) => {
      this.reconnectAttempts = 0; // 重置重连计数
      console.log('WebSocket connected');

      // 如果有token，发送认证消息
      const token = Auth.getToken();
      if (token) {
        try {
          this.sendMessage({
            type: 'AUTH',
            token: token
          });
        } catch (e) {
          console.error('发送认证消息失败:', e);
        }
      }

      this._triggerEvent('open', event);
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);

        // 处理认证响应
        if (data.type === 'AUTH_RESPONSE') {
          if (!data.success) {
            console.error('WebSocket认证失败:', data.message);
            // 认证失败，可能是token过期
            if (data.code === 'TOKEN_EXPIRED' || data.code === 'UNAUTHORIZED') {
              Auth.clearToken();
              this._triggerEvent('auth_expired', data);
              // 关闭当前连接，触发 onclose → 自动重连（带新 token）
              this.isManuallyClosed = false;
              if (this.ws) {
                this.ws.close();
              }
            }
          } else {
            console.log('WebSocket认证成功');
          }
          return;
        }

        console.log('Received WebSocket message:', data);
        this._triggerEvent('message', data);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', event.data, error);
        // 包装为标准格式，避免监听器因类型不匹配崩溃
        this._triggerEvent('message', {
          type: 'PARSE_ERROR',
          raw: event.data,
          error: error.message
        });
      }
    };

    this.ws.onclose = (event) => {
      console.log('WebSocket closed:', event.code, event.reason);

      if (!this.isManuallyClosed) {
        this.reconnectAttempts++;
        // 指数退避 + 随机抖动
        const backoff = Math.min(
          this.baseReconnectInterval * Math.pow(2, this.reconnectAttempts - 1),
          this.maxReconnectInterval
        );
        const jitter = Math.random() * this.reconnectJitter - this.reconnectJitter / 2;
        const delay = Math.max(backoff + jitter, 500);
        console.log(`Attempting to reconnect (attempt ${this.reconnectAttempts}, delay ${Math.round(delay)}ms)...`);
        setTimeout(() => this.connect(), delay);
      } else {
        this._triggerEvent('close', event);
      }
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);

      // 检查是否是认证错误
      if (error.message && (error.message.includes('401') || error.message.includes('unauthorized'))) {
        console.log('WebSocket连接遇到认证错误');
        Auth.clearToken();
        this._triggerEvent('auth_error', error);
      }

      this._triggerEvent('error', error);
    };
  }

  // 发送消息
  sendMessage(message) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket is not connected, message dropped:', message);
    }
  }

  // 监听事件
  on(event, handler) {
    if (!this.eventHandlers[event]) {
      this.eventHandlers[event] = [];
    }
    this.eventHandlers[event].push(handler);
  }

  // 触发事件
  _triggerEvent(event, data) {
    if (this.eventHandlers[event]) {
      this.eventHandlers[event].forEach(handler => {
        try {
          handler(data);
        } catch (error) {
          console.error(`Error in ${event} handler:`, error);
        }
      });
    }
  }

  // 手动关闭连接
  close() {
    this.isManuallyClosed = true;
    if (this.ws) {
      this.ws.close();
    }
  }

  // 获取当前连接状态
  getStatus() {
    if (!this.ws) return WebSocket.CLOSED;
    return this.ws.readyState;
  }

  // 检查是否已连接
  isConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN;
  }
}

export default WebSocketService;