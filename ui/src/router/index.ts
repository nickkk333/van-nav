import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    component: () => import('../views/HomeView.vue'),
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/LoginView.vue'),
  },
  {
    path: '/admin',
    component: () => import('../views/admin/AdminLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        redirect: '/admin/tools',
      },
      {
        path: 'tools',
        name: 'admin-tools',
        component: () => import('../views/admin/ToolsView.vue'),
      },
      {
        path: 'categories',
        name: 'admin-categories',
        component: () => import('../views/admin/CatelogView.vue'),
      },
      {
        path: 'search-engines',
        name: 'admin-search-engines',
        component: () => import('../views/admin/SearchEngineView.vue'),
      },
      {
        path: 'api-token',
        name: 'admin-api-token',
        component: () => import('../views/admin/ApiTokenView.vue'),
      },
      {
        path: 'settings',
        name: 'admin-settings',
        component: () => import('../views/admin/SettingView.vue'),
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/',
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 后台页面需要登录态
router.beforeEach((to) => {
  if (to.meta.requiresAuth && !localStorage.getItem('_token')) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  return true
})

export default router