// utils/auth.js - 认证工具类

const TOKEN_KEY = 'token';
const USER_KEY = 'user_info';
const LOGOUT_EVENT_KEY = '__auth_logout_ts__';

const Auth = {
  getToken() {
    return localStorage.getItem(TOKEN_KEY);
  },

  setToken(token) {
    if (token) {
      localStorage.setItem(TOKEN_KEY, token);
    }
  },

  getAuthHeaders() {
    const token = this.getToken();
    return token ? { Authorization: `Bearer ${token}` } : {};
  },

  clearToken() {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
    // 广播退出事件到同源其他标签页
    localStorage.setItem(LOGOUT_EVENT_KEY, String(Date.now()));
    localStorage.removeItem(LOGOUT_EVENT_KEY);
  },

  isLoggedIn() {
    return !!this.getToken();
  },

  setUserInfo(userInfo) {
    if (userInfo) {
      localStorage.setItem(USER_KEY, JSON.stringify(userInfo));
    }
  },

  getUserInfo() {
    try {
      const raw = localStorage.getItem(USER_KEY);
      return raw ? JSON.parse(raw) : null;
    } catch (e) {
      return null;
    }
  },

  onCrossTabLogout(callback) {
    const handler = (e) => {
      if (e.key === TOKEN_KEY && !e.newValue) {
        callback();
      }
    };
    window.addEventListener('storage', handler);
    return () => window.removeEventListener('storage', handler);
  },
};

export default Auth;
