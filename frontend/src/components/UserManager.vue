<script setup>
import { ref, h, onMounted } from 'vue'
import { useMessage, NButton, NSwitch, NTag, NText, NInput, NForm, NFormItem, NModal, NSpace, NDatePicker, NSelect, NInputGroup } from 'naive-ui'
import apiService from '../services/api.js'

const message = useMessage()
const loading = ref(false)
const userList = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const searchKeyword = ref('')
const searchRole = ref(null)

// 模态框状态
const showCreateModal = ref(false)
const showResetPasswordModal = ref(false)
const showChangeRoleModal = ref(false)
const showEditVipModal = ref(false)
const showEditUserModal = ref(false)
const selectedUser = ref(null)

// 表单数据
const createForm = ref({
  username: '',
  email: '',
  password: '',
  display_name: '',
  is_active: true,
  role: 'user' // user, vip, admin, super_admin
})

const editUserForm = ref({
  username: '',
  email: '',
  display_name: '',
  is_active: true,
  role: 'user'
})

const resetPasswordForm = ref({
  newPassword: ''
})

const changeRoleForm = ref({
  role: 'user'
})

const editVipForm = ref({
  vip_level: 0,
  vip_start_at: '',
  vip_end_at: ''
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
    key: 'role',
    width: 120,
    render(row) {
      const roleMap = {
        'super_admin': { type: 'error', text: '超级管理员' },
        'admin': { type: 'warning', text: '管理员' },
        'vip': { type: 'success', text: 'VIP' },
        'user': { type: 'info', text: '普通用户' }
      }
      const roleInfo = roleMap[row.role] || { type: 'default', text: row.role }
      return h(NTag, { 
        type: roleInfo.type,
        size: 'small'
      }, { default: () => roleInfo.text })
    }
  },
  {
    title: 'VIP等级',
    key: 'vip_level',
    width: 100,
    render(row) {
      if (!row.vip_level || row.vip_level === 0) {
        return h(NTag, { type: 'default', size: 'small' }, { default: () => '-' })
      }
      return h(NTag, { type: 'warning', size: 'small' }, { default: () => `VIP${row.vip_level}` })
    }
  },
  {
    title: 'VIP有效期',
    key: 'vip_period',
    width: 200,
    render(row) {
      if (!row.vip_level || row.vip_level === 0) {
        return h(NText, { depth: 3 }, { default: () => '-' })
      }
      const start = row.vip_start_at ? new Date(row.vip_start_at).toLocaleDateString('zh-CN') : '-'
      const end = row.vip_end_at ? new Date(row.vip_end_at).toLocaleDateString('zh-CN') : '-'
      const now = new Date()
      const endDate = row.vip_end_at ? new Date(row.vip_end_at) : null
      const isExpired = endDate && endDate < now
      return h(NText, { type: isExpired ? 'error' : 'success' }, { default: () => `${start} ~ ${end}` })
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
    width: 400,
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
            onClick: () => openEditUserModal(row)
          }, { default: () => '编辑' }),
          h(NButton, {
            size: 'small',
            type: 'info',
            onClick: () => openEditVipModal(row)
          }, { default: () => 'VIP' }),
          h(NButton, {
            size: 'small',
            type: 'primary',
            onClick: () => openChangeRoleModal(row)
          }, { default: () => '角色' }),
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
    const params = {
      page: page.value,
      pageSize: pageSize.value
    }
    if (searchKeyword.value) {
      params.keyword = searchKeyword.value
    }
    if (searchRole.value) {
      params.role = searchRole.value
    }
    console.log('正在请求 /admin/users 接口，参数:', params)
    const response = await apiService.client.get('/admin/users', { params })

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

// 搜索
function handleSearch() {
  page.value = 1
  loadUsers()
}

// 重置筛选
function handleReset() {
  searchKeyword.value = ''
  searchRole.value = null
  page.value = 1
  loadUsers()
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

// 打开编辑用户模态框
function openEditUserModal(user) {
  selectedUser.value = user
  editUserForm.value = {
    username: user.username,
    email: user.email,
    display_name: user.display_name || '',
    is_active: user.is_active,
    role: user.role || 'user'
  }
  showEditUserModal.value = true
}

// 更新用户信息
async function handleUpdateUser() {
  if (!selectedUser.value) return

  try {
    const response = await apiService.client.put(`/admin/users/${selectedUser.value.id}`, {
      username: editUserForm.value.username,
      email: editUserForm.value.email,
      display_name: editUserForm.value.display_name,
      is_active: editUserForm.value.is_active,
      role: editUserForm.value.role
    })
    
    if (response.data.code === 0) {
      message.success('用户信息更新成功')
      showEditUserModal.value = false
      loadUsers()
    } else {
      message.error(response.data.message || '更新失败')
    }
  } catch (error) {
    console.error('Update user error:', error)
    message.error('更新用户信息失败：' + (error.response?.data?.message || error.message))
  }
}

// 打开修改角色模态框
function openChangeRoleModal(user) {
  selectedUser.value = user
  changeRoleForm.value.role = user.role || 'user'
  showChangeRoleModal.value = true
}

// 打开编辑VIP模态框
function openEditVipModal(user) {
  selectedUser.value = user
  editVipForm.value = {
    vip_level: user.vip_level || 0,
    vip_start_at: user.vip_start_at ? new Date(user.vip_start_at).getTime() : null,
    vip_end_at: user.vip_end_at ? new Date(user.vip_end_at).getTime() : null
  }
  showEditVipModal.value = true
}

// 更新用户VIP信息
async function handleUpdateVip() {
  if (!selectedUser.value) return

  // 转换时间戳为后端格式
  const formatTimestamp = (ts) => {
    if (!ts) return ''
    const d = new Date(ts)
    const year = d.getFullYear()
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    const hour = String(d.getHours()).padStart(2, '0')
    const minute = String(d.getMinutes()).padStart(2, '0')
    const second = String(d.getSeconds()).padStart(2, '0')
    return `${year}-${month}-${day} ${hour}:${minute}:${second}`
  }

  try {
    const response = await apiService.client.put(`/admin/users/${selectedUser.value.id}/vip`, {
      vip_level: editVipForm.value.vip_level,
      vip_start_at: editVipForm.value.vip_start_at ? formatTimestamp(editVipForm.value.vip_start_at) : '',
      vip_end_at: editVipForm.value.vip_end_at ? formatTimestamp(editVipForm.value.vip_end_at) : ''
    })
    
    if (response.data.code === 0) {
      message.success('VIP信息更新成功')
      showEditVipModal.value = false
      loadUsers()
    } else {
      message.error(response.data.message || '更新失败')
    }
  } catch (error) {
    console.error('Update VIP error:', error)
    message.error('更新VIP信息失败：' + (error.response?.data?.message || error.message))
  }
}

// 修改角色
async function handleChangeRole() {
  if (!selectedUser.value) return

  try {
    const response = await apiService.client.put(`/admin/users/${selectedUser.value.id}/role`, {
      role: changeRoleForm.value.role
    })
    
    if (response.data.code === 0) {
      message.success('角色修改成功')
      showChangeRoleModal.value = false
      loadUsers()
    } else {
      message.error(response.data.message || '修改失败')
    }
  } catch (error) {
    console.error('Change role error:', error)
    message.error('修改角色失败：' + (error.response?.data?.message || error.message))
  }
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
    role: 'user'
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

      <!-- 搜索筛选区域 -->
      <div class="search-bar">
        <n-space align="center">
          <n-input-group>
            <n-input v-model:value="searchKeyword" placeholder="搜索用户名/邮箱" clearable style="width: 200px" @keyup.enter="handleSearch" />
          </n-input-group>
          <n-select
            v-model:value="searchRole"
            placeholder="选择角色"
            clearable
            style="width: 140px"
            :options="[
              { label: '普通用户', value: 'user' },
              { label: 'VIP', value: 'vip' },
              { label: '管理员', value: 'admin' },
              { label: '超级管理员', value: 'super_admin' }
            ]"
          />
          <n-button type="primary" @click="handleSearch">搜索</n-button>
          <n-button @click="handleReset">重置</n-button>
        </n-space>
      </div>

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
        <n-form-item label="角色">
          <n-select 
            v-model:value="createForm.role" 
            :options="[
              { label: '普通用户', value: 'user' },
              { label: 'VIP', value: 'vip' },
              { label: '管理员', value: 'admin' },
              { label: '超级管理员', value: 'super_admin' }
            ]"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreateModal = false">取消</n-button>
          <n-button type="primary" @click="handleCreateUser">创建</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 编辑用户模态框 -->
    <n-modal v-model:show="showEditUserModal" preset="card" title="编辑用户信息" style="width: 500px">
      <n-form :model="editUserForm" label-placement="left" label-width="100">
        <n-form-item label="用户名">
          <n-input v-model:value="editUserForm.username" placeholder="3-20个字符" />
        </n-form-item>
        <n-form-item label="邮箱">
          <n-input v-model:value="editUserForm.email" type="email" placeholder="请输入邮箱" />
        </n-form-item>
        <n-form-item label="显示名称">
          <n-input v-model:value="editUserForm.display_name" placeholder="可选" />
        </n-form-item>
        <n-form-item label="状态">
          <n-switch v-model:value="editUserForm.is_active">
            <template #checked>激活</template>
            <template #unchecked>停用</template>
          </n-switch>
        </n-form-item>
        <n-form-item label="角色">
          <n-select 
            v-model:value="editUserForm.role" 
            :options="[
              { label: '普通用户', value: 'user' },
              { label: 'VIP', value: 'vip' },
              { label: '管理员', value: 'admin' },
              { label: '超级管理员', value: 'super_admin' }
            ]"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEditUserModal = false">取消</n-button>
          <n-button type="primary" @click="handleUpdateUser">保存</n-button>
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

    <!-- 修改角色模态框 -->
    <n-modal v-model:show="showChangeRoleModal" preset="card" title="修改用户角色" style="width: 450px">
      <n-form :model="changeRoleForm" label-placement="left" label-width="100">
        <n-form-item label="用户名">
          <n-text strong>{{ selectedUser?.username }}</n-text>
        </n-form-item>
        <n-form-item label="当前角色">
          <n-tag :type="selectedUser?.role === 'super_admin' ? 'error' : selectedUser?.role === 'admin' ? 'warning' : selectedUser?.role === 'vip' ? 'success' : 'info'">
            {{ selectedUser?.role === 'super_admin' ? '超级管理员' : selectedUser?.role === 'admin' ? '管理员' : selectedUser?.role === 'vip' ? 'VIP' : '普通用户' }}
          </n-tag>
        </n-form-item>
        <n-form-item label="新角色" required>
          <n-select 
            v-model:value="changeRoleForm.role" 
            :options="[
              { label: '普通用户', value: 'user' },
              { label: 'VIP', value: 'vip' },
              { label: '管理员', value: 'admin' },
              { label: '超级管理员', value: 'super_admin' }
            ]"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showChangeRoleModal = false">取消</n-button>
          <n-button type="primary" @click="handleChangeRole">确认</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 编辑VIP模态框 -->
    <n-modal v-model:show="showEditVipModal" preset="card" title="编辑用户VIP信息" style="width: 500px">
      <n-form :model="editVipForm" label-placement="left" label-width="100">
        <n-form-item label="用户名">
          <n-text strong>{{ selectedUser?.username }}</n-text>
        </n-form-item>
        <n-form-item label="VIP等级">
          <n-select 
            v-model:value="editVipForm.vip_level"
            :options="[
              { label: '普通用户', value: 0 },
              { label: 'VIP 1级', value: 1 },
              { label: 'VIP 2级', value: 2 },
              { label: 'VIP 3级', value: 3 },
              { label: 'VIP 4级', value: 4 }
            ]"
          />
        </n-form-item>
        <n-form-item label="开始时间">
          <n-date-picker v-model:value="editVipForm.vip_start_at" type="datetime" clearable style="width: 100%" />
        </n-form-item>
        <n-form-item label="到期时间">
          <n-date-picker v-model:value="editVipForm.vip_end_at" type="datetime" clearable style="width: 100%" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEditVipModal = false">取消</n-button>
          <n-button type="primary" @click="handleUpdateVip">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.user-manager {
  padding: 20px;
}
.search-bar {
  margin-bottom: 16px;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 6px;
}
</style>
