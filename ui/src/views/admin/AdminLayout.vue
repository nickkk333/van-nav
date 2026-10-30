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
      <div
        ref="asideRef"
        class="admin-aside"
        :style="isMobile ? undefined : { width: collapsed ? '64px' : '200px' }"
      >
        <!-- 手机上侧边栏改为顶部横向分类栏（可左右滑动），内容区占满整宽 -->
        <el-menu
          class="admin-menu"
          :default-active="activeMenu"
          :mode="isMobile ? 'horizontal' : 'vertical'"
          :collapse="!isMobile && collapsed"
          :collapse-transition="false"
          :ellipsis="false"
          router
        >
          <el-menu-item v-for="item in menus" :key="item.path" :index="item.path">
            <el-icon><component :is="item.icon" /></el-icon>
            <template #title>{{ item.label }}</template>
          </el-menu-item>
        </el-menu>
        <el-button v-if="!isMobile" class="collapse-btn" circle size="small" @click="toggleCollapse">
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
import { computed, nextTick, ref, watch } from 'vue'
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
import { useIsMobile } from '../../composables/useIsMobile'

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
/** 手机布局：侧边栏改为顶部横向导航，且不支持折叠 */
const isMobile = useIsMobile()

const collapsed = ref(localStorage.getItem(COLLAPSE_KEY) === 'true')
const activeMenu = computed(() => route.path)

/** 手机上的横向导航：把当前菜单项滚动到可见区域（靠后的菜单项默认会被挡在屏幕外） */
const asideRef = ref<HTMLElement>()
const scrollActiveMenuIntoView = () => {
  const menu = asideRef.value?.querySelector<HTMLElement>('.admin-menu')
  const active = menu?.querySelector<HTMLElement>('.el-menu-item.is-active')
  if (!menu || !active) {
    return
  }
  const menuRect = menu.getBoundingClientRect()
  const activeRect = active.getBoundingClientRect()
  // 让当前项居中显示：差值加上已滚动的距离，就是目标 scrollLeft
  const delta = activeRect.left - menuRect.left - (menuRect.width - activeRect.width) / 2
  menu.scrollTo({ left: Math.max(0, menu.scrollLeft + delta), behavior: 'smooth' })
}

watch(
  [() => route.path, isMobile],
  async () => {
    if (!isMobile.value) {
      return
    }
    await nextTick()
    // el-menu 的选中态是内部 watch 更新的，稍等一帧再滚动，避免还拿不到 is-active
    window.setTimeout(scrollActiveMenuIntoView, 80)
  },
  { immediate: true }
)

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
