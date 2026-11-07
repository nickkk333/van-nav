<template>
  <Teleport to="body">
    <!-- 展开状态：打开即自动扩充，覆盖页面「除搜索框」以外的区域 -->
    <section
      v-if="visible"
      ref="winRef"
      class="pv-memo-win"
      :class="{ dragging }"
      :style="winStyle"
      aria-label="便签"
    >
      <header class="pv-memo-win-head" @pointerdown="startDrag">
        <span class="pv-memo-win-title">便签</span>
        <div class="pv-memo-win-actions">
          <button type="button" class="pv-memo-win-btn" title="关闭" aria-label="关闭" @click="close"><el-icon><Close /></el-icon></button>
        </div>
      </header>
      <div class="pv-memo-win-body"><MemoPanel ref="memoRef" /></div>
      <span class="pv-memo-win-rz" title="拖动调整大小" @pointerdown="startResize"></span>
    </section>
  </Teleport>
</template>
<script setup lang="ts">
import { computed, nextTick, onUnmounted, reactive, ref, watch } from 'vue'
import { Close } from '@element-plus/icons-vue'
import MemoPanel from './MemoPanel.vue'
import { memoExpanded } from '../../utils/memoWindow'

// ==================== 便签浮动窗口（打开自动覆盖除搜索框外的页面区域） ====================
/** 窗口与视口边缘的间距 */
const EDGE = 8
/** 窗口顶部与搜索框底部的间距 */
const TOP_GAP = 12
/** 找不到搜索框时，顶部预留的兜底距离 */
const FALLBACK_TOP = 96
/** 手动调整大小的下限 */
const W_MIN = 280
const H_MIN = 260

const visible = ref(false)
const dragging = ref(false)
const memoRef = ref<InstanceType<typeof MemoPanel> | null>(null)
const winRef = ref<HTMLElement | null>(null)
const size = reactive({ w: 0, h: 0 })
const pos = reactive({ x: 0, y: 0 })

const clamp = (v: number, min: number, max: number) => Math.min(Math.max(v, min), Math.max(min, max))

const winStyle = computed(() => ({
  left: `${pos.x}px`,
  top: `${pos.y}px`,
  width: `${size.w}px`,
  height: `${size.h}px`,
}))

/** 尺寸收进视口 */
const clampSize = () => {
  const maxW = Math.max(1, window.innerWidth - EDGE * 2)
  const maxH = Math.max(1, window.innerHeight - EDGE * 2)
  size.w = clamp(size.w, 1, maxW)
  size.h = clamp(size.h, 1, maxH)
}
/** 窗口整体收进视口 */
const clampPos = () => {
  pos.x = clamp(pos.x, 0, Math.max(0, window.innerWidth - size.w))
  pos.y = clamp(pos.y, 0, Math.max(0, window.innerHeight - size.h))
}
/**
 * 打开时自动扩充：覆盖页面「除搜索框」以外的区域。
 * 以首页搜索框的实际底部为起点（适配不同视口 / 主题 / 「仅显示搜索框」模式），
 * 横向铺满、纵向延伸到视口底部；不再贴边吸附成图标
 */
const coverBelowSearch = () => {
  const vw = window.innerWidth
  const vh = window.innerHeight
  const search =
    document.querySelector<HTMLElement>('.topbar .search-wraper') ??
    document.querySelector<HTMLElement>('.topbar .search')
  const rect = search?.getBoundingClientRect()
  const baseTop = rect && rect.bottom > 0 ? rect.bottom : FALLBACK_TOP
  const top = Math.max(0, Math.min(Math.round(baseTop + TOP_GAP), vh - 80))
  size.w = Math.max(1, vw - EDGE * 2)
  size.h = Math.max(1, vh - top - EDGE)
  pos.x = EDGE
  pos.y = top
}

const reloadMemo = async () => {
  await nextTick()
  memoRef.value?.reload?.()
}
/** 打开：自动扩充到覆盖除搜索框外的页面区域，并拉取最新便签 */
const open = async () => {
  // 首页的搜索框与便签窗口是否显示无关，先算好尺寸再显示，避免 0 尺寸闪一帧
  coverBelowSearch()
  visible.value = true
  // reloadMemo 内部会等面板挂载完成
  await reloadMemo()
}
const close = () => {
  // 关闭会销毁面板，先把防抖期内尚未提交的修改落盘，避免丢失
  void memoRef.value?.save?.()
  visible.value = false
}
/** 浮窗入口的点击行为：开着就关，否则展开 */
const toggle = () => {
  if (visible.value) close()
  else void open()
}

// ==================== 点击便签外区域自动收起（不再吸附到边缘） ====================
const onOutsidePointerDown = (ev: PointerEvent) => {
  if (!visible.value) return
  // 只响应主键 / 触摸：右键是菜单操作（卡片右键「设置」、浏览器原生菜单），
  // 不能顺手把便签关掉，否则同一次右键会让页面状态突变、卡片右键菜单丢失
  if (ev.pointerType === 'mouse' && ev.button !== 0) return
  const target = ev.target as HTMLElement | null
  if (!target) return
  if (winRef.value?.contains(target)) return
  // 便签入口按钮交给自身点击逻辑处理
  if (target.closest('.pv-dock')) return
  // 搜索框区域是便签覆盖页面后特意保留的：点击它不关闭便签，可继续做网页搜索
  if (target.closest('.topbar')) return
  // 历史版本弹窗、图片预览等 Teleport 到 body 的浮层，点击不算「便签外」
  if (target.closest('.el-overlay') || target.closest('.el-image-viewer__wrapper')) return
  close()
}
watch(visible, (isOpen) => {
  memoExpanded.value = isOpen
  if (isOpen) document.addEventListener('pointerdown', onOutsidePointerDown)
  else document.removeEventListener('pointerdown', onOutsidePointerDown)
})

// ==================== 拖动移动（仅本次会话临时调整，不再贴边吸附） ====================
let dragOrigin = { x: 0, y: 0, px: 0, py: 0 }
const onDrag = (ev: PointerEvent) => {
  pos.x = dragOrigin.px + (ev.clientX - dragOrigin.x)
  pos.y = dragOrigin.py + (ev.clientY - dragOrigin.y)
  clampPos()
}
const stopDrag = () => {
  dragging.value = false
  document.body.style.userSelect = ''
  window.removeEventListener('pointermove', onDrag)
  window.removeEventListener('pointerup', stopDrag)
  window.removeEventListener('pointercancel', stopDrag)
}
const startDrag = (ev: PointerEvent) => {
  if (ev.pointerType === 'mouse' && ev.button !== 0) return
  // 标题栏里的按钮不触发拖动
  if ((ev.target as HTMLElement | null)?.closest('button, a, input, textarea')) return
  ev.preventDefault()
  dragging.value = true
  dragOrigin = { x: ev.clientX, y: ev.clientY, px: pos.x, py: pos.y }
  document.body.style.userSelect = 'none'
  window.addEventListener('pointermove', onDrag)
  window.addEventListener('pointerup', stopDrag)
  window.addEventListener('pointercancel', stopDrag)
}

// ==================== 右下角拖动调整大小 ====================
const startResize = (ev: PointerEvent) => {
  if (ev.pointerType === 'mouse' && ev.button !== 0) return
  ev.preventDefault()
  const start = { x: ev.clientX, y: ev.clientY, w: size.w, h: size.h }
  document.body.style.cursor = 'nwse-resize'
  document.body.style.userSelect = 'none'
  const onMove = (e: PointerEvent) => {
    const maxW = Math.max(W_MIN, window.innerWidth - EDGE * 2)
    const maxH = Math.max(H_MIN, window.innerHeight - EDGE * 2)
    size.w = clamp(start.w + (e.clientX - start.x), Math.min(W_MIN, maxW), maxW)
    size.h = clamp(start.h + (e.clientY - start.y), Math.min(H_MIN, maxH), maxH)
    clampPos()
  }
  const stop = () => {
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', stop)
    window.removeEventListener('pointercancel', stop)
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', stop)
  window.addEventListener('pointercancel', stop)
}

// ==================== 视口变化：展开时重新覆盖，保证始终贴合页面 ====================
const onViewportResize = () => {
  if (!visible.value) return
  coverBelowSearch()
  clampSize()
  clampPos()
}
window.addEventListener('resize', onViewportResize)

onUnmounted(() => {
  memoExpanded.value = false
  window.removeEventListener('resize', onViewportResize)
  document.removeEventListener('pointerdown', onOutsidePointerDown)
  window.removeEventListener('pointermove', onDrag)
  window.removeEventListener('pointerup', stopDrag)
  window.removeEventListener('pointercancel', stopDrag)
  document.body.style.userSelect = ''
  document.body.style.cursor = ''
})

defineExpose({ open, close, toggle })
</script>
