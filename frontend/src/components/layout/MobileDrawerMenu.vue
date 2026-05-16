<template>
  <n-drawer v-model:show="drawerVisible" :width="280" placement="right">
    <n-drawer-content title="菜单" closable>
      <n-menu
        :value="activeKey"
        :options="flatMenuOptions"
        @update:value="handleSelect"
      />
    </n-drawer-content>
  </n-drawer>
</template>

<script setup>
import { computed, watch } from 'vue'
import { useRouter } from 'vue-router'

const props = defineProps({
  show: {
    type: Boolean,
    default: false
  },
  menuOptions: {
    type: Array,
    default: () => []
  },
  activeKey: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:show', 'select'])

const router = useRouter()

const drawerVisible = computed({
  get: () => props.show,
  set: (val) => emit('update:show', val)
})

// 扁平化菜单，方便移动端显示
const flatMenuOptions = computed(() => {
  const result = []

  props.menuOptions.forEach(item => {
    if (item.show === false) return

    // 添加一级菜单
    result.push({
      ...item,
      children: undefined // 先清除子菜单
    })

    // 如果有子菜单，展开添加
    if (item.children && item.children.length > 0) {
      item.children.forEach(child => {
        if (child.show === false) return
        result.push({
          ...child,
          label: `  └ ${child.label}`,
          key: child.key
        })
      })
    }
  })

  return result
})

const handleSelect = (key) => {
  emit('select', key)
}
</script>

<style scoped>
:deep(.n-menu-item) {
  height: 44px;
}
</style>
