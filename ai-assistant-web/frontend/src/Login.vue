<template>
  <div class="login-container">
    <n-card title="go-stock AI 助手登录" class="login-card">
      <n-form :model="formValue" :rules="rules" ref="formRef">
        <n-form-item label="用户名" path="username">
          <n-input v-model:value="formValue.username" placeholder="请输入用户名" />
        </n-form-item>
        <n-form-item label="密码" path="password">
          <n-input v-model:value="formValue.password" type="password" placeholder="请输入密码" @keyup.enter="handleLogin" />
        </n-form-item>
        <n-button type="primary" block :loading="loading" @click="handleLogin">
          登录
        </n-button>
      </n-form>
      <div class="register-link">
        还没有账号？<n-button text type="primary" @click="showRegister = true">立即注册</n-button>
      </div>
    </n-card>

    <!-- 注册对话框 -->
    <n-modal v-model:show="showRegister" preset="dialog" title="用户注册">
      <n-form :model="registerForm" :rules="registerRules" ref="registerFormRef">
        <n-form-item label="用户名" path="username">
          <n-input v-model:value="registerForm.username" placeholder="请输入用户名" />
        </n-form-item>
        <n-form-item label="邮箱" path="email">
          <n-input v-model:value="registerForm.email" placeholder="请输入邮箱（可选）" />
        </n-form-item>
        <n-form-item label="显示名称" path="displayName">
          <n-input v-model:value="registerForm.displayName" placeholder="请输入显示名称（可选）" />
        </n-form-item>
        <n-form-item label="密码" path="password">
          <n-input v-model:value="registerForm.password" type="password" placeholder="请输入密码" />
        </n-form-item>
        <n-form-item label="确认密码" path="confirmPassword">
          <n-input v-model:value="registerForm.confirmPassword" type="password" placeholder="请再次输入密码" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-button @click="showRegister = false">取消</n-button>
        <n-button type="primary" :loading="registerLoading" @click="handleRegister">注册</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue';
import { useMessage } from 'naive-ui';
import { useRouter } from 'vue-router';

const message = useMessage();
const router = useRouter();

const formRef = ref();
const registerFormRef = ref();
const loading = ref(false);
const registerLoading = ref(false);
const showRegister = ref(false);

const formValue = reactive({
  username: '',
  password: '',
});

const registerForm = reactive({
  username: '',
  email: '',
  displayName: '',
  password: '',
  confirmPassword: '',
});

const rules = {
  username: { required: true, message: '请输入用户名', trigger: 'blur' },
  password: { required: true, message: '请输入密码', trigger: 'blur' },
};

const registerRules = {
  username: { required: true, message: '请输入用户名', trigger: 'blur' },
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少为6位', trigger: 'blur' },
  ],
  confirmPassword: {
    required: true,
    validator: (rule: any, value: string) => {
      if (value !== registerForm.password) {
        return new Error('两次输入的密码不一致');
      }
      return true;
    },
    trigger: 'blur',
  },
};

async function handleLogin() {
  try {
    await formRef.value?.validate();
    loading.value = true;

    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: formValue.username,
        password: formValue.password,
      }),
    });

    const data = await res.json();

    if (!res.ok) {
      message.error(data.message || '登录失败');
      return;
    }

    // 保存token到localStorage
    localStorage.setItem('token', data.token);
    localStorage.setItem('userId', String(data.userId));
    localStorage.setItem('username', data.username);
    localStorage.setItem('role', data.role);

    message.success('登录成功');
    
    // 跳转到主页
    setTimeout(() => {
      window.location.href = '/';
    }, 500);
  } catch (error: any) {
    message.error(error.message || '登录失败');
  } finally {
    loading.value = false;
  }
}

async function handleRegister() {
  try {
    await registerFormRef.value?.validate();
    registerLoading.value = true;

    const res = await fetch('/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: registerForm.username,
        email: registerForm.email,
        displayName: registerForm.displayName,
        password: registerForm.password,
      }),
    });

    const data = await res.json();

    if (!res.ok) {
      message.error(data.message || '注册失败');
      return;
    }

    message.success('注册成功，请登录');
    showRegister.value = false;
    
    // 清空注册表单
    registerForm.username = '';
    registerForm.email = '';
    registerForm.displayName = '';
    registerForm.password = '';
    registerForm.confirmPassword = '';
  } catch (error: any) {
    message.error(error.message || '注册失败');
  } finally {
    registerLoading.value = false;
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
  padding: 20px;
}

.register-link {
  margin-top: 16px;
  text-align: center;
}
</style>
