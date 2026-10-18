<template>
  <Teleport to="body">
    <!-- 收起状态：吸附在屏幕边缘的便签图标，点击重新展开 -->
    <button
      v-if="visible && collapsed"
      type="button"
      class="pv-memo-icon"
      :class="`pv-memo-icon-${side}`"
      :style="iconStyle"
      title="打开便签"
      aria-label="打开便签"
      @click="expand"
    >
      <el-icon :size="18"><EditPen /></el-icon>
    </button>
    <!-- 展开状态：可拖动、可调整大小的浮动便签窗口 -->
    <section
      v-else-if="visible"
      class="pv-memo-win"
      :class="{ dragging }"
      :style="winStyle"
      aria-label="便签"
    >
      <header class="pv-memo-win-head" @pointerdown="startDrag">
        <span class="pv-memo-win-title">便签</span>
        <div class="pv-memo-win-actions">
          <button type="button" class="pv-memo-win-btn" title="收起到边缘" aria-label="收起到边缘" @click="minimize"><el-icon><Minus /></el-icon></button>
          <button type="button" class="pv-memo-win-btn" title="关闭" aria-label="关闭" @click="close"><el-icon><Close /></el-icon></button>
        </div>
      </header>
      <div class="pv-memo-win-body"><MemoPanel ref="memoRef" /></div>
      <span class="pv-memo-win-rz" title="拖动调整大小" @pointerdown="startResize"></span>
    </section>
  </Teleport>
</template>
<script setup lang="ts">
import { computed, nextTick, onUnmounted, reactive, ref } from 'vue'
import { Close, EditPen, Minus } from '@element-plus/icons-vue'
import MemoPanel from './MemoPanel.vue'

// ==================== 便签浮动窗口（拖动移动、贴边吸附成图标、点击展开） ====================
type Side = 'left' | 'right' | 'top' | 'bottom'
/** 位置 / 尺寸 / 吸附边 / 收起状态存 localStorage，刷新后回到原位 */
const STORE_KEY = 'pv-memo-window'
const W_DEFAULT = 380
const H_DEFAULT = 420
const W_MIN = 280
const H_MIN = 260
/** 拖到离边缘多近就吸附（px） */
const SNAP = 28
/** 收起成图标后与屏幕上下/左右角的最小间距，避免图标缩到角落点不到 */
const ICON_GAP = 60

const visible = ref(false)
const collapsed = ref(false)
const dragging = ref(false)
const memoRef = ref<InstanceType<typeof MemoPanel> | null>(null)
const size = reactive({ w: W_DEFAULT, h: H_DEFAULT })
const pos = reactive({ x: 0, y: 0 })
const side = ref<Side>('right')
/** 视口尺寸变化的计数器：让依赖窗口大小的计算属性（图标位置等）重新求值 */
const vp = ref(0)

const clamp = (v: number, min: number, max: number) => Math.min(Math.max(v, min), Math.max(min, max))

const winStyle = computed(() => ({
  left: `${pos.x}px`,
  top: `${pos.y}px`,
  width: `${size.w}px`,
  height: `${size.h}px`,
}))
/** 图标居中贴在被吸附的那条边上 */
const iconStyle = computed(() => {
  void vp.value
  if (side.value === 'left' || side.value === 'right') {
    return { top: `${clamp(pos.y + size.h / 2, ICON_GAP, window.innerHeight - ICON_GAP)}px` }
  }
  return { left: `${clamp(pos.x + size.w / 2, ICON_GAP, window.innerWidth - ICON_GAP)}px` }
})

/** 尺寸收进视口 */
const clampSize = () => {
  const maxW = Math.max(W_MIN, window.innerWidth - 24)
  const maxH = Math.max(H_MIN, window.innerHeight - 24)
  size.w = clamp(size.w, Math.min(W_MIN, maxW), maxW)
  size.h = clamp(size.h, Math.min(H_MIN, maxH), maxH)
}
/** 窗口整体收进视口 */
const clampPos = () => {
  pos.x = clamp(pos.x, 0, Math.max(0, window.innerWidth - size.w))
  pos.y = clamp(pos.y, 0, Math.max(0, window.innerHeight - size.h))
}
/** 默认位置：贴着右下角浮窗入口上方 */
const defaultPos = () => {
  pos.x = Math.max(12, window.innerWidth - size.w - 76)
  pos.y = Math.max(12, window.innerHeight - size.h - 76)
}
/** 当前离哪条边最近 */
const nearestSide = () => {
  const dist: Record<Side, number> = {
    left: pos.x,
    right: window.innerWidth - (pos.x + size.w),
    top: pos.y,
    bottom: window.innerHeight - (pos.y + size.h),
  }
  let best: Side = 'right'
  ;(Object.keys(dist) as Side[]).forEach((k) => {
    if (dist[k] < dist[best]) best = k
  })
  return { side: best, dist: dist[best] }
}
/** 把窗口贴到指定边（不改收起状态） */
const snapTo = (s: Side) => {
  side.value = s
  if (s === 'left') pos.x = 0
  else if (s === 'right') pos.x = Math.max(0, window.innerWidth - size.w)
  else if (s === 'top') pos.y = 0
  else pos.y = Math.max(0, window.innerHeight - size.h)
}

const persist = () => {
  try {
    window.localStorage.setItem(
      STORE_KEY,
      JSON.stringify({ x: pos.x, y: pos.y, w: size.w, h: size.h, side: side.value, collapsed: collapsed.value }),
    )
  } catch {
    /* 隐私模式等存不了：忽略 */
  }
}
/** 读取上次的窗口状态；有记录返回 true */
const restore = () => {
  let saved: Record<string, unknown> | null = null
  try {
    saved = JSON.parse(window.localStorage.getItem(STORE_KEY) || 'null')
  } catch {
    saved = null
  }
  if (!saved || typeof saved !== 'object') return false
  if (typeof saved.w === 'number') size.w = saved.w
  if (typeof saved.h === 'number') size.h = saved.h
  if (typeof saved.x === 'number') pos.x = saved.x
  if (typeof saved.y === 'number') pos.y = saved.y
  if (saved.side === 'left' || saved.side === 'right' || saved.side === 'top' || saved.side === 'bottom') side.value = saved.side
  collapsed.value = saved.collapsed === true
  clampSize()
  clampPos()
  return true
}
if (!restore()) defaultPos()

const reloadMemo = async () => {
  await nextTick()
  memoRef.value?.reload?.()
}
/** 打开：展开成完整窗口并拉取最新便签 */
const open = async () => {
  visible.value = true
  collapsed.value = false
  clampSize()
  clampPos()
  persist()
  await reloadMemo()
}
const close = () => {
  visible.value = false
  persist()
}
/** 点击图标展开 */
const expand = async () => {
  collapsed.value = false
  persist()
  await reloadMemo()
}
/** 收起到最近的一条边，变成图标 */
const minimize = () => {
  snapTo(nearestSide().side)
  collapsed.value = true
  persist()
}
/** 收起成图标时，也把窗口坐标贴到该边，展开后位置才一致 */
const collapseTo = (s: Side) => {
  snapTo(s)
  collapsed.value = true
  persist()
}
/** 浮窗入口的点击行为：开着就关，否则展开（图标状态也顺带展开） */
const toggle = () => {
  if (visible.value && !collapsed.value) close()
  else void open()
}

// ==================== 拖动移动 + 贴边吸附 ====================
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
  const near = nearestSide()
  side.value = near.side
  // 松手时贴着边缘：吸附并收起成图标；否则停在原地，两种结果都记下来
  if (near.dist <= SNAP) collapseTo(near.side)
  else persist()
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
    const maxW = Math.max(W_MIN, window.innerWidth - 24)
    const maxH = Math.max(H_MIN, window.innerHeight - 24)
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
    persist()
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', stop)
  window.addEventListener('pointercancel', stop)
}

// ==================== 视口变化：尺寸与位置收回可视范围 ====================
const onViewportResize = () => {
  vp.value++
  clampSize()
  clampPos()
}
window.addEventListener('resize', onViewportResize)

onUnmounted(() => {
  window.removeEventListener('resize', onViewportResize)
  window.removeEventListener('pointermove', onDrag)
  window.removeEventListener('pointerup', stopDrag)
  window.removeEventListener('pointercancel', stopDrag)
  document.body.style.userSelect = ''
  document.body.style.cursor = ''
})

defineExpose({ open, close, toggle })
</script>
