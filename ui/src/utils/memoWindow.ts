import { ref } from 'vue'

/**
 * 便签窗口是否展开（响应式，刷新即重置）。
 * 由 MemoWindow 维护，HomeView 读取：便签展开时首页搜索框切换为「纯网页搜索」模式——
 * 输入不过滤卡片，回车 / 点搜索按钮用默认搜索引擎做网页搜索，不再打开工具卡片。
 * 与 utils/theme.ts 的模块级状态使用同一惯例。
 */
export const memoExpanded = ref(false)
