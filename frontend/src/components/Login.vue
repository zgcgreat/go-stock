<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <h1>go-stock</h1>
        <p>AI赋能股票分析</p>
      </div>
      <n-form ref="formRef" :model="formData" :rules="rules" label-placement="left" label-width="0" require-mark-placement="right-hanging">
        <n-form-item path="username">
          <n-input v-model:value="formData.username" placeholder="用户名" size="large" @keyup.enter="handleLogin">
            <template #prefix>
              <n-icon :component="PersonOutline" />
            </template>
          </n-input>
        </n-form-item>
        <n-form-item path="password">
          <n-input v-model:value="formData.password" type="password" placeholder="密码" size="large" show-password-on="click" @keyup.enter="handleLogin">
            <template #prefix>
              <n-icon :component="LockClosedOutline" />
            </template>
          </n-input>
        </n-form-item>
        <n-button type="primary" size="large" block :loading="loading" @click="handleLogin">
          登录
        </n-button>
      </n-form>
      <div class="login-footer">
        <n-text depth="3">还没有账号？</n-text>
        <n-button text type="primary" @click="router.push('/register')">
          立即注册
        </n-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { PersonOutline, LockClosedOutline } from '@vicons/ionicons5'
import { NIcon, useMessage } from 'naive-ui'
import apiService from '../services/api.js'
import Auth from '../utils/auth.js'

const router = useRouter()
const message = useMessage()
const formRef = ref(null)
const loading = ref(false)

const formData = ref({
  username: '',
  password: ''
})

const rules = {
  username: { required: true, message: '请输入用户名', trigger: 'blur' },
  password: { required: true, message: '请输入密码', trigger: 'blur' }
}

const handleLogin = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }

  loading.value = true
  try {
    const res = await apiService.login(formData.value)
    if (res.token) {
      localStorage.setItem('token', res.token)
      // 保存用户信息（含 role），用于权限判断
      Auth.setUserInfo({
        userId: res.userId,
        username: res.username,
        role: res.role,
        vipLevel: res.vipLevel,
        vipStartAt: res.vipStartAt,
        vipEndAt: res.vipEndAt,
      })
      // 通知同 Tab 其他组件刷新用户信息
      window.dispatchEvent(new Event('user-info-updated'))
      message.success('登录成功')
      // 路由守卫可能带了 redirect 参数
      const redirect = router.currentRoute.value.query.redirect || '/'
      router.push(redirect)
    }
  } catch (err) {
    message.error(err.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
.login-card {
  width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}
.login-header {
  text-align: center;
  margin-bottom: 32px;
}
.login-header h1 {
  font-size: 28px;
  color: #333;
  margin: 0;
}
.login-header p {
  color: #666;
  margin-top: 8px;
}
.login-footer {
  text-align: center;
  margin-top: 20px;
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
}
</style>
