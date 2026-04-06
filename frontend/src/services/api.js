// services/api.js
import axios from 'axios';

// 创建一个 axios 实例
const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器 - 用于添加认证令牌
apiClient.interceptors.request.use(
  (config) => {
    // 从localStorage获取token并添加到请求头
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器 - 用于统一处理错误和认证过期
apiClient.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    // 处理401未授权错误（排除登录/注册接口本身）
    if (error.response && error.response.status === 401) {
      const url = error.config?.url || '';
      if (!url.includes('/auth/login') && !url.includes('/auth/register')) {
        localStorage.removeItem('token');
        window.location.href = '/login';
      }
    }

    console.error('API Error:', error);
    return Promise.reject(error);
  }
);

// API 服务类
export class ApiService {
  constructor() {
    this.client = apiClient;
  }

  // 认证相关 API
  async register(userData) {
    try {
      const response = await this.client.post('/auth/register', userData);
      if (response.data.token) {
        localStorage.setItem('token', response.data.token);
      }
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async login(credentials) {
    try {
      const response = await this.client.post('/auth/login', credentials);
      if (response.data.token) {
        localStorage.setItem('token', response.data.token);
      }
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async logout() {
    try {
      const response = await this.client.post('/auth/logout');
      localStorage.removeItem('token');
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async getUserProfile() {
    try {
      const response = await this.client.get('/user/profile');
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async updateUserProfile(profileData) {
    try {
      const response = await this.client.put('/user/profile', profileData);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  // 股票相关 API
  async getStockBasics(params = {}) {
    try {
      const response = await this.client.get('/stocks/basic', { params });
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async getStockByCode(code) {
    try {
      const response = await this.client.get(`/stocks/${code}`);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async getStockKLine(code, params = {}) {
    try {
      const response = await this.client.get(`/stocks/${code}/kline`, { params });
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async getStockMinuteData(code) {
    try {
      const response = await this.client.get(`/stocks/${code}/minute`);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async searchStocks(query, params = {}) {
    try {
      params.q = query; // 使用查询参数
      const response = await this.client.get('/stocks/basic', { params });
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  // AI 相关 API
  async aiTradeAnalyze(request) {
    try {
      const response = await this.client.post('/ai/analyze', request);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async getAIResponses(params = {}) {
    try {
      const response = await this.client.get('/ai/responses', { params });
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async deleteAIResponse(id) {
    try {
      const response = await this.client.delete(`/ai/responses/${id}`);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  // 提示模板 API
  async getPromptTemplates(params = {}) {
    try {
      const response = await this.client.get('/prompts/templates', { params });
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async createPromptTemplate(templateData) {
    try {
      const response = await this.client.post('/prompts/templates', templateData);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async updatePromptTemplate(id, templateData) {
    try {
      const response = await this.client.put(`/prompts/templates/${id}`, templateData);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async deletePromptTemplate(id) {
    try {
      const response = await this.client.delete(`/prompts/templates/${id}`);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  // 交易记录 API
  async getTradeRecords(params = {}) {
    try {
      const response = await this.client.get('/trades', { params });
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async createTradeRecord(tradeData) {
    try {
      const response = await this.client.post('/trades', tradeData);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async updateTradeRecord(id, tradeData) {
    try {
      const response = await this.client.put(`/trades/${id}`, tradeData);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async deleteTradeRecord(id) {
    try {
      const response = await this.client.delete(`/trades/${id}`);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  // 设置 API
  async getUserSettings(params = {}) {
    try {
      const response = await this.client.get('/settings', { params });
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async setUserSetting(settingData) {
    try {
      const response = await this.client.post('/settings', settingData);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async updateSetting(key, settingData) {
    try {
      const response = await this.client.put(`/settings/${key}`, settingData);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  async deleteSetting(key) {
    try {
      const response = await this.client.delete(`/settings/${key}`);
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  // 健康检查
  async healthCheck() {
    try {
      const response = await this.client.get('/public/health');
      return response.data;
    } catch (error) {
      throw this.handleError(error);
    }
  }

  // 错误处理辅助方法
  handleError(error) {
    if (error.response) {
      // 服务器响应了错误状态码
      const errorMessage = error.response.data?.message || error.response.data?.error || `服务器错误: ${error.response.status}`;
      console.error(`API Error ${error.response.status}:`, errorMessage);
      return new Error(errorMessage);
    } else if (error.request) {
      // 请求已发出但没有收到响应
      console.error('网络错误:', error.message);
      return new Error('网络连接失败，请检查网络连接');
    } else {
      // 其他错误
      console.error('请求错误:', error.message);
      return new Error(error.message || '请求失败');
    }
  }
}

// 创建全局实例
const apiService = new ApiService();

export default apiService;