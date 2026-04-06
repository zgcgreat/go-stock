// utils/auth.js - 认证工具类

const TOKEN_KEY = 'token';
const USER_KEY = 'user_info';

const Auth = {
  /**
   * 获取 token
   * @returns {string|null}
   */
  getToken() {
    return localStorage.getItem(TOKEN_KEY);
  },

  /**
   * 保存 token
   * @param {string} token
   */
  setToken(token) {
    if (token) {
      localStorage.setItem(TOKEN_KEY, token);
    }
  },

  /**
   * 获取授权头
   * @returns {object}
   */
  getAuthHeaders() {
    const token = this.getToken();
    return token ? { Authorization: `Bearer ${token}` } : {};
  },

  /**
   * 清除 token（退出登录）
   */
  clearToken() {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
  },

  /**
   * 判断是否已登录
   * @returns {boolean}
   */
  isLoggedIn() {
    return !!this.getToken();
  },

  /**
   * 保存用户信息
   * @param {object} userInfo
   */
  setUserInfo(userInfo) {
    if (userInfo) {
      localStorage.setItem(USER_KEY, JSON.stringify(userInfo));
    }
  },

  /**
   * 获取用户信息
   * @returns {object|null}
   */
  getUserInfo() {
    try {
      const raw = localStorage.getItem(USER_KEY);
      return raw ? JSON.parse(raw) : null;
    } catch (e) {
      return null;
    }
  },

  /**
   * 获取 Authorization Header
   * @returns {object}
   */
  getAuthHeaders() {
    const token = this.getToken();
    return token ? { Authorization: `Bearer ${token}` } : {};
  },
};

export default Auth;
