// utils/websocket.js
import Auth from './auth';

class WebSocketService {
  constructor(url) {
    this.url = url;
    this.ws = null;
    this.reconnectInterval = 5000; // 5秒重连间隔
    this.maxReconnectAttempts = 5; // 最大重连次数
    this.reconnectAttempts = 0;
    this.eventHandlers = {};
    this.isManuallyClosed = false;
    this.authRetry = false; // 防止无限重试认证
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
          this.authRetry = false; // 重置认证重试标志
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
              // 不要在WebSocket中直接重定向，而是触发一个事件通知应用
              this._triggerEvent('auth_expired', data);
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
        this._triggerEvent('message', event.data);
      }
    };

    this.ws.onclose = (event) => {
      console.log('WebSocket closed:', event.code, event.reason);

      if (!this.isManuallyClosed && this.reconnectAttempts < this.maxReconnectAttempts) {
        this.reconnectAttempts++;
        console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
        setTimeout(() => this.connect(), this.reconnectInterval);
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
      console.error('WebSocket is not connected');
      throw new Error('WebSocket is not connected');
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