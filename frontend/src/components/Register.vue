<template>
  <div class="register-container">
    <div class="register-card">
      <div class="register-header">
        <h1>go-stock</h1>
        <p>注册新账号</p>
      </div>
      <n-form ref="formRef" :model="formData" :rules="rules" label-placement="left" label-width="0" require-mark-placement="right-hanging">
        <n-form-item path="username">
          <n-input v-model:value="formData.username" placeholder="用户名（3-20字符）" size="large">
            <template #prefix>
              <n-icon :component="PersonOutline" />
            </template>
          </n-input>
        </n-form-item>
        <n-form-item path="email">
          <n-input v-model:value="formData.email" placeholder="邮箱" size="large">
            <template #prefix>
              <n-icon :component="MailOutline" />
            </template>
          </n-input>
        </n-form-item>
        <n-form-item path="password">
          <n-input v-model:value="formData.password" type="password" placeholder="密码（至少6位）" size="large" show-password-on="click">
            <template #prefix>
              <n-icon :component="LockClosedOutline" />
            </template>
          </n-input>
        </n-form-item>
        <n-form-item path="confirmPassword">
          <n-input v-model:value="formData.confirmPassword" type="password" placeholder="确认密码" size="large" show-password-on="click">
            <template #prefix>
              <n-icon :component="LockClosedOutline" />
            </template>
          </n-input>
        </n-form-item>
        <n-button type="primary" size="large" block :loading="loading" @click="handleRegister">
          注册
        </n-button>
      </n-form>
      <div class="register-footer">
        <n-text depth="3">已有账号？</n-text>
        <n-button text type="primary" @click="router.push('/login')">
          立即登录
        </n-button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { PersonOutline, LockClosedOutline, MailOutline } from '@vicons/ionicons5'
import { NIcon, useMessage } from 'naive-ui'
import apiService from '../services/api.js'

const router = useRouter()
const message = useMessage()
const formRef = ref(null)
const loading = ref(false)

const formData = ref({
  username: '',
  email: '',
  password: '',
  confirmPassword: ''
})

const validatePasswordSame = (rule, value) => {
  return value === formData.value.password
}

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '用户名长度3-20字符', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效邮箱', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少6位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: validatePasswordSame, message: '两次密码不一致', trigger: 'blur' }
  ]
}

const handleRegister = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }

  loading.value = true
  try {
    const { confirmPassword, ...registerData } = formData.value
    const res = await apiService.register(registerData)
    if (res.token) {
      // 使用统一的 Auth 工具类来处理 token 存储
      localStorage.setItem('token', res.token)
      message.success('注册成功')
      router.push('/')
    }
  } catch (err) {
    message.error(err.message || '注册失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.register-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
.register-card {
  width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}
.register-header {
  text-align: center;
  margin-bottom: 32px;
}
.register-header h1 {
  font-size: 28px;
  color: #333;
  margin: 0;
}
.register-header p {
  color: #666;
  margin-top: 8px;
}
.register-footer {
  text-align: center;
  margin-top: 20px;
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
}
</style>
