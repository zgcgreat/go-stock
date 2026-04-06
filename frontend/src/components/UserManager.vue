<script setup>
import { ref, h, onMounted } from 'vue'
import { useMessage, NButton, NSwitch, NTag, NText, NInput, NForm, NFormItem, NModal, NSpace } from 'naive-ui'
import apiService from '../services/api.js'

const message = useMessage()
const loading = ref(false)
const userList = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

// 模态框状态
const showCreateModal = ref(false)
const showResetPasswordModal = ref(false)
const selectedUser = ref(null)

// 表单数据
const createForm = ref({
  username: '',
  email: '',
  password: '',
  display_name: '',
  is_active: true,
  is_admin: false
})

const resetPasswordForm = ref({
  newPassword: ''
})

// 表格列定义
const columns = [
  {
    title: 'ID',
    key: 'id',
    width: 80
  },
  {
    title: '用户名',
    key: 'username',
    width: 150
  },
  {
    title: '显示名称',
    key: 'display_name',
    width: 150,
    render(row) {
      return h(NText, { type: 'info' }, { default: () => row.display_name || '-' })
    }
  },
  {
    title: '邮箱',
    key: 'email',
    width: 200
  },
  {
    title: '状态',
    key: 'is_active',
    width: 100,
    render(row) {
      return h(NTag, { 
        type: row.is_active ? 'success' : 'error',
        size: 'small'
      }, { default: () => row.is_active ? '激活' : '停用' })
    }
  },
  {
    title: '角色',
    key: 'is_admin',
    width: 100,
    render(row) {
      return h(NTag, { 
        type: row.is_admin ? 'warning' : 'info',
        size: 'small'
      }, { default: () => row.is_admin ? '管理员' : '普通用户' })
    }
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 180,
    render(row) {
      return new Date(row.created_at).toLocaleString('zh-CN')
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 250,
    fixed: 'right',
    render(row) {
      return h(NSpace, {}, {
        default: () => [
          h(NSwitch, {
            value: row.is_active,
            onUpdateValue: (value) => handleToggleStatus(row.id, value),
            size: 'small'
          }),
          h(NButton, {
            size: 'small',
            onClick: () => openResetPasswordModal(row)
          }, { default: () => '重置密码' }),
          h(NButton, {
            size: 'small',
            type: 'error',
            onClick: () => handleDeleteUser(row.id)
          }, { default: () => '删除' })
        ]
      })
    }
  }
]

// 加载用户列表
async function loadUsers() {
  loading.value = true
  try {
    console.log('正在请求 /admin/users 接口，参数:', {
      page: page.value,
      pageSize: pageSize.value
    })
    const response = await apiService.client.get('/admin/users', {
      params: {
        page: page.value,
        pageSize: pageSize.value
      }
    })

    console.log('收到响应:', response.data)

    if (response.data.code === 0) {
      userList.value = response.data.data.list
      total.value = response.data.data.total
      console.log('用户列表加载成功，用户数量:', userList.value.length)
    } else {
      message.error(response.data.message || '获取用户列表失败')
      console.error('接口返回错误:', response.data.message)
    }
  } catch (error) {
    console.error('Load users error:', error)
    if (error.response) {
      console.error('错误详情:', error.response.status, error.response.data)
    }
    message.error('获取用户列表失败：' + (error.response?.data?.message || error.message || error))
  } finally {
    loading.value = false
  }
}

// 切换用户状态
async function handleToggleStatus(userId, isActive) {
  try {
    const response = await apiService.client.put('/admin/users/status', {
      userId,
      active: isActive
    })
    
    if (response.data.code === 0) {
      message.success('用户状态更新成功')
      loadUsers()
    } else {
      message.error(response.data.message || '更新失败')
    }
  } catch (error) {
    console.error('Update status error:', error)
    message.error('更新用户状态失败：' + (error.response?.data?.message || error.message))
  }
}

// 打开重置密码模态框
function openResetPasswordModal(user) {
  selectedUser.value = user
  resetPasswordForm.value.newPassword = ''
  showResetPasswordModal.value = true
}

// 重置密码
async function handleResetPassword() {
  if (!resetPasswordForm.value.newPassword || resetPasswordForm.value.newPassword.length < 6) {
    message.warning('密码长度至少6位')
    return
  }

  try {
    const response = await apiService.client.put('/admin/users/password', {
      userId: selectedUser.value.id,
      newPassword: resetPasswordForm.value.newPassword
    })
    
    if (response.data.code === 0) {
      message.success('密码重置成功')
      showResetPasswordModal.value = false
    } else {
      message.error(response.data.message || '重置失败')
    }
  } catch (error) {
    console.error('Reset password error:', error)
    message.error('重置密码失败：' + (error.response?.data?.message || error.message))
  }
}

// 删除用户
async function handleDeleteUser(userId) {
  if (!confirm('确定要删除该用户吗？此操作不可恢复！')) {
    return
  }

  try {
    const response = await apiService.client.delete(`/admin/users/${userId}`)
    
    if (response.data.code === 0) {
      message.success('用户删除成功')
      loadUsers()
    } else {
      message.error(response.data.message || '删除失败')
    }
  } catch (error) {
    console.error('Delete user error:', error)
    message.error('删除用户失败：' + (error.response?.data?.message || error.message))
  }
}

// 创建用户
async function handleCreateUser() {
  if (!createForm.value.username || !createForm.value.email || !createForm.value.password) {
    message.warning('请填写必填项')
    return
  }

  if (createForm.value.password.length < 6) {
    message.warning('密码长度至少6位')
    return
  }

  try {
    const response = await apiService.client.post('/admin/users', createForm.value)
    
    if (response.data.code === 0) {
      message.success('用户创建成功')
      showCreateModal.value = false
      resetCreateForm()
      loadUsers()
    } else {
      message.error(response.data.message || '创建失败')
    }
  } catch (error) {
    console.error('Create user error:', error)
    message.error('创建用户失败：' + (error.response?.data?.message || error.message))
  }
}

// 重置创建表单
function resetCreateForm() {
  createForm.value = {
    username: '',
    email: '',
    password: '',
    display_name: '',
    is_active: true,
    is_admin: false
  }
}

onMounted(() => {
  loadUsers()
})
</script>

<template>
  <div class="user-manager">
    <n-card title="用户管理">
      <template #header-extra>
        <n-button type="primary" @click="showCreateModal = true">
          创建用户
        </n-button>
      </template>

      <n-data-table
        :columns="columns"
        :data="userList"
        :loading="loading"
        :pagination="{
          page: page,
          pageSize: pageSize,
          itemCount: total,
          showSizePicker: true,
          pageSizes: [10, 20, 50],
          onUpdatePage: (p) => { page = p; loadUsers() },
          onUpdatePageSize: (ps) => { pageSize = ps; page = 1; loadUsers() }
        }"
        :bordered="false"
      />
    </n-card>

    <!-- 创建用户模态框 -->
    <n-modal v-model:show="showCreateModal" preset="card" title="创建新用户" style="width: 500px">
      <n-form :model="createForm" label-placement="left" label-width="100">
        <n-form-item label="用户名" required>
          <n-input v-model:value="createForm.username" placeholder="3-20个字符" />
        </n-form-item>
        <n-form-item label="邮箱" required>
          <n-input v-model:value="createForm.email" type="email" placeholder="请输入邮箱" />
        </n-form-item>
        <n-form-item label="密码" required>
          <n-input v-model:value="createForm.password" type="password" placeholder="至少6位" />
        </n-form-item>
        <n-form-item label="显示名称">
          <n-input v-model:value="createForm.display_name" placeholder="可选" />
        </n-form-item>
        <n-form-item label="状态">
          <n-switch v-model:value="createForm.is_active">
            <template #checked>激活</template>
            <template #unchecked>停用</template>
          </n-switch>
        </n-form-item>
        <n-form-item label="管理员">
          <n-switch v-model:value="createForm.is_admin">
            <template #checked>是</template>
            <template #unchecked>否</template>
          </n-switch>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreateModal = false">取消</n-button>
          <n-button type="primary" @click="handleCreateUser">创建</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 重置密码模态框 -->
    <n-modal v-model:show="showResetPasswordModal" preset="card" title="重置密码" style="width: 400px">
      <n-form :model="resetPasswordForm" label-placement="left" label-width="100">
        <n-form-item label="新密码" required>
          <n-input v-model:value="resetPasswordForm.newPassword" type="password" placeholder="至少6位" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showResetPasswordModal = false">取消</n-button>
          <n-button type="primary" @click="handleResetPassword">确认</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.user-manager {
  padding: 20px;
}
</style>
