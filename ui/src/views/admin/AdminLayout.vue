<template>
  <el-container class="admin-layout">
    <el-header class="admin-header">
      <div class="admin-header-title">VanNav 管理系统</div>
      <div class="admin-header-actions">
        <el-button link :icon="HomeFilled" @click="router.push('/')">返回主页</el-button>
        <el-button link :icon="SwitchButton" @click="handleLogout">退出登录</el-button>
      </div>
    </el-header>
    <div class="admin-body">
      <div class="admin-aside" :style="{ width: collapsed ? '64px' : '200px' }">
        <el-menu
          class="admin-menu"
          :default-active="activeMenu"
          :collapse="collapsed"
          :collapse-transition="false"
          router
        >
          <el-menu-item v-for="item in menus" :key="item.path" :index="item.path">
            <el-icon><component :is="item.icon" /></el-icon>
            <template #title>{{ item.label }}</template>
          </el-menu-item>
        </el-menu>
        <el-button class="collapse-btn" circle size="small" @click="toggleCollapse">
          <el-icon>
            <component :is="collapsed ? DArrowRight : DArrowLeft" />
          </el-icon>
        </el-button>
      </div>
      <div class="admin-main">
        <router-view />
      </div>
    </div>
  </el-container>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Box,
  Collection,
  DArrowLeft,
  DArrowRight,
  HomeFilled,
  Key,
  Search,
  Setting,
  SwitchButton,
} from '@element-plus/icons-vue'
import { useAdminStore } from '../../stores/admin'

const COLLAPSE_KEY = 'admin-sidebar-collapsed'

const menus = [
  { key: 'tools', path: '/admin/tools', label: '工具管理', icon: Box },
  { key: 'categories', path: '/admin/categories', label: '分类管理', icon: Collection },
  { key: 'search-engines', path: '/admin/search-engines', label: '搜索引擎管理', icon: Search },
  { key: 'api-token', path: '/admin/api-token', label: 'API Token', icon: Key },
  { key: 'settings', path: '/admin/settings', label: '系统设置', icon: Setting },
]

const route = useRoute()
const router = useRouter()
const adminStore = useAdminStore()

const collapsed = ref(localStorage.getItem(COLLAPSE_KEY) === 'true')
const activeMenu = computed(() => route.path)

const toggleCollapse = () => {
  collapsed.value = !collapsed.value
  localStorage.setItem(COLLAPSE_KEY, String(collapsed.value))
}

const handleLogout = () => {
  localStorage.removeItem('_token')
  adminStore.reset()
  router.replace('/')
}
</script>
