import { createApp } from 'vue'
import { createPinia } from 'pinia'

import 'element-plus/theme-chalk/dark/css-vars.css'
// ElMessage / ElMessageBox 是在脚本里直接 import 调用的，不是模板里的 <el-*> 组件，
// 按需引入插件不会自动注入它们的样式，不显式引入的话提示条和确认弹窗没有排版
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/message-box/style/css'
import './styles/index.css'

import App from './App.vue'
import router from './router'
import { initTheme } from './utils/theme'

const app = createApp(App)

// 注意：不要再写 import ElementPlus from 'element-plus' + app.use(ElementPlus，
// 那会让 element-plus 全量进包（之前 element 分包 926KB 的根因）。
// 模板里的 el-* 组件、v-loading 指令、ElMessage/ElMessageBox 等 API 由
// vite.config.ts 的 AutoImport + Components 插件按需引入；
// 也不要全局注册 @element-plus/icons-vue（import * 进主包约几百 KB，各组件已显式 import 图标）。
// 中文文案靠 App.vue 的 <el-config-provider :locale="zhCn"> 生效，分页/表格等即为中文。

app.use(createPinia())
app.use(router)

initTheme()
app.mount('#app')