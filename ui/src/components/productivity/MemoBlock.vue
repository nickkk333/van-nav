<template>
  <div ref="rootRef" class="pv-block" :class="[`pv-block-c${colorIndex % 3}`, { 'is-active': active }]" :style="blockStyle">
    <div
      ref="bodyRef"
      class="pv-block-body"
      contenteditable="true"
      role="textbox"
      aria-multiline="true"
      :aria-label="placeholder"
      @input="sync"
      @focus="emit('focus')"
      @paste="onPaste"
      @click="onClick"
      @dragover.prevent="dragOver = true"
      @dragleave="dragOver = false"
      @drop.prevent="onDrop"
    ></div>
    <div v-if="!hasContent" class="pv-block-placeholder">{{ placeholder }}</div>
    <button v-if="removable" type="button" class="pv-block-remove" title="删除块" aria-label="删除块" @mousedown.prevent @click="emit('remove')">
      <el-icon :size="12"><Close /></el-icon>
    </button>
    <span
      ref="resizeRef"
      class="pv-block-resize"
      :class="{ 'is-resizing': resizing, 'is-collapsed': collapsed }"
      role="separator"
      aria-orientation="horizontal"
      title="拖动调整高度，双击折叠为一行 / 展开"
      aria-label="拖动调整块高度，双击折叠为一行或展开"
      @pointerdown="startResize"
    ></span>
    <input ref="inputRef" class="pv-block-input" type="file" multiple @change="onPicked" />
    <!-- 点击图片放大：浮层显示原图（自带缩放 / 旋转） -->
    <el-image-viewer v-if="previewUrl" :url-list="[previewUrl]" teleported hide-on-click-modal @close="previewUrl = ''" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { Close } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { fetchUploadMemoFile, resolveError } from '../../api'
import type { MemoFile } from '../../types'

/**
 * 便签的单个「块」：带圆角的矩形区域，宽度撑满便签、高度可拖动拉伸。
 * - 块内是独立的富文本编辑区，可输入文字、粘贴图片/文件、拖入文件
 * - v-model 绑定块内 HTML；height 为 null 时高度随内容自适应，拖动底部把手后固定高度
 * - 双击底部把手在「缩为一行」与「展开（恢复原高度 / 自适应）」之间切换
 */
const props = withDefaults(
  defineProps<{
    /** 块内 HTML 内容 */
    modelValue: string
    /** 块高度（px），null 表示自适应内容 */
    height: number | null
    /** 是否为当前聚焦块 */
    active?: boolean
    /** 是否显示删除按钮（至少保留一个块时最后一个不显示） */
    removable?: boolean
    /** 空内容时的提示文案 */
    placeholder?: string
    /** 颜色索引（0/1/2 循环），决定块的单色填充 */
    colorIndex?: number
  }>(),
  {
    height: null,
    active: false,
    removable: true,
    placeholder: '输入内容，或拖入 / 粘贴文件、图片',
    colorIndex: 0,
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'update:height', value: number | null): void
  (e: 'focus'): void
  (e: 'remove'): void
}>()

/** 块最小高度（px）：一行文本 + 内边距，拖动下限的兜底值 */
const MIN_H = 44

const rootRef = ref<HTMLElement | null>(null)
const bodyRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const resizeRef = ref<HTMLElement | null>(null)
const resizing = ref(false)
const dragOver = ref(false)
const hasContent = ref(false)
const previewUrl = ref('')
const uploading = ref(false)

/** 编辑器里最后的光标位置：点「上传文件」按钮后，内容要插回原来的位置 */
let savedRange: Range | null = null

const blockStyle = computed(() => (props.height ? { height: `${props.height}px` } : {}))

/**
 * 「刚好显示一行」的块高度：按正文实际行高与内边距计算。
 * 触屏下底部把手更高、正文 padding-bottom 更大，直接写死 44px 会裁掉那一行文字。
 */
const oneLineHeight = (): number => {
  const body = bodyRef.value
  if (!body) return MIN_H
  const cs = window.getComputedStyle(body)
  const line = parseFloat(cs.lineHeight) || 22.4
  const pad = (parseFloat(cs.paddingTop) || 0) + (parseFloat(cs.paddingBottom) || 0)
  // 2px 是 .pv-block 上下边框；块的 height 是 border-box，也要算进去
  return Math.max(MIN_H, Math.ceil(line + pad + 2))
}

/** 当前是否处于「缩为一行」状态（null 表示自适应内容，不算折叠） */
const collapsed = computed(() => props.height !== null && props.height <= oneLineHeight() + 1)

/** 折叠前记住的高度：null 表示原本自适应，展开时恢复自适应 */
let expandedHeight: number | null = null

const toggleHeight = () => {
  const oneLine = oneLineHeight()
  if (collapsed.value) {
    emit('update:height', expandedHeight && expandedHeight > oneLine + 1 ? expandedHeight : null)
  } else {
    expandedHeight = props.height
    emit('update:height', oneLine)
  }
}

const isBlank = (el: HTMLElement) => !(el.textContent || '').trim() && !el.querySelector('img, a')

const sync = () => {
  const el = bodyRef.value
  if (!el) return
  hasContent.value = !isBlank(el)
  emit('update:modelValue', el.innerHTML)
}

watch(
  () => props.modelValue,
  (value) => {
    const el = bodyRef.value
    if (!el) return
    // 内容一致就不动 DOM，否则会丢掉正在输入的光标位置
    if (el.innerHTML === (value || '')) return
    el.innerHTML = value || ''
    hasContent.value = !isBlank(el)
  }
)

const rememberRange = () => {
  const el = bodyRef.value
  const sel = window.getSelection()
  if (!el || !sel || sel.rangeCount === 0) return
  const range = sel.getRangeAt(0)
  if (el.contains(range.commonAncestorContainer)) {
    savedRange = range.cloneRange()
  }
}

/** 取当前用于插入的位置：优先记住的光标位置，其次实时选区，最后追加到末尾 */
const resolveRange = (el: HTMLElement): Range => {
  if (savedRange && el.contains(savedRange.commonAncestorContainer)) {
    return savedRange
  }
  const sel = window.getSelection()
  if (sel && sel.rangeCount > 0) {
    const range = sel.getRangeAt(0)
    if (el.contains(range.commonAncestorContainer)) return range
  }
  const range = document.createRange()
  range.selectNodeContents(el)
  range.collapse(false)
  return range
}

const insertNodes = (nodes: Node[]) => {
  const el = bodyRef.value
  if (!el || !nodes.length) return
  const range = resolveRange(el)
  range.deleteContents()
  const frag = document.createDocumentFragment()
  nodes.forEach((node) => frag.appendChild(node))
  range.insertNode(frag)
  const after = document.createRange()
  after.setStartAfter(nodes[nodes.length - 1])
  after.collapse(true)
  const sel = window.getSelection()
  sel?.removeAllRanges()
  sel?.addRange(after)
  savedRange = after.cloneRange()
  sync()
}

const insertHtml = (html: string) => {
  const tpl = document.createElement('template')
  tpl.innerHTML = html
  insertNodes(Array.from(tpl.content.childNodes))
}

// ==================== 粘贴内容清洗 ====================

const DROP_TAGS = new Set([
  'SCRIPT', 'STYLE', 'IFRAME', 'OBJECT', 'EMBED', 'LINK', 'META', 'BASE',
  'SVG', 'MATH', 'FORM', 'INPUT', 'BUTTON', 'SELECT', 'TEXTAREA', 'VIDEO',
  'AUDIO', 'CANVAS', 'NOSCRIPT', 'TEMPLATE',
])

const KEEP_TAGS = new Set([
  'DIV', 'P', 'BR', 'SPAN', 'B', 'STRONG', 'I', 'EM', 'U', 'S', 'STRIKE', 'DEL',
  'A', 'IMG', 'UL', 'OL', 'LI', 'H1', 'H2', 'H3', 'H4', 'BLOCKQUOTE', 'CODE',
  'PRE', 'HR',
])

const KEEP_ATTR: Record<string, string[]> = {
  A: ['href', 'title', 'target', 'download', 'class', 'contenteditable'],
  IMG: ['src', 'alt', 'class'],
  SPAN: ['class'],
  DIV: ['class'],
}

const isSafeUrl = (raw: string): boolean => {
  const url = (raw || '').trim()
  if (!url) return false
  if (url.startsWith('/')) return true
  return /^https?:\/\//i.test(url) || /^mailto:/i.test(url)
}

const isSafeClass = (value: string) => value.split(/\s+/).filter(Boolean).every((name) => name.startsWith('pv-'))

const sanitizeHtml = (html: string): string => {
  const doc = new DOMParser().parseFromString(html, 'text/html')
  doc.body.querySelectorAll(Array.from(DROP_TAGS).join(',')).forEach((el) => el.remove())
  Array.from(doc.body.querySelectorAll('*'))
    .filter((el) => !KEEP_TAGS.has(el.tagName.toUpperCase()))
    .forEach((el) => el.replaceWith(...Array.from(el.childNodes)))
  doc.body.querySelectorAll('*').forEach((el) => {
    const allowed = KEEP_ATTR[el.tagName.toUpperCase()] || []
    Array.from(el.attributes).forEach((attr) => {
      const name = attr.name.toLowerCase()
      const ok =
        allowed.includes(name) &&
        (name !== 'href' && name !== 'src' ? true : isSafeUrl(attr.value)) &&
        (name !== 'class' ? true : isSafeClass(attr.value))
      if (!ok) el.removeAttribute(attr.name)
    })
  })
  return doc.body.innerHTML
}

// ==================== 文件上传 ====================

const formatSize = (size: number) => (size >= 1024 * 1024 ? `${(size / 1024 / 1024).toFixed(1)} MB` : `${Math.max(1, Math.round(size / 1024))} KB`)

const extLabel = (name: string) => {
  const matched = /\.([A-Za-z0-9]{1,5})$/.exec(name || '')
  return matched ? matched[1].toUpperCase() : 'FILE'
}

const imageNodes = (file: MemoFile): Node[] => {
  const img = document.createElement('img')
  img.src = file.url
  img.alt = file.original || '图片'
  img.loading = 'lazy'
  return [img]
}

const fileNodes = (file: MemoFile): Node[] => {
  const link = document.createElement('a')
  link.className = 'pv-file'
  link.href = file.url
  link.title = `${file.original}（${formatSize(file.size)}）`
  link.setAttribute('target', '_blank')
  link.setAttribute('rel', 'noreferrer')
  link.setAttribute('contenteditable', 'false')
  const icon = document.createElement('span')
  icon.className = 'pv-file-icon'
  icon.textContent = extLabel(file.original)
  const name = document.createElement('span')
  name.className = 'pv-file-name'
  name.textContent = file.original || '附件'
  link.append(icon, name)
  const br = document.createElement('br')
  return [link, br]
}

const uploadFiles = async (files: File[]) => {
  if (!files.length || uploading.value) return
  uploading.value = true
  try {
    for (const file of files) {
      try {
        const uploaded = await fetchUploadMemoFile(file)
        insertNodes(uploaded.isImage ? imageNodes(uploaded) : fileNodes(uploaded))
      } catch (e) {
        ElMessage.error(resolveError(e, `上传「${file.name}」失败`))
      }
    }
  } finally {
    uploading.value = false
    if (inputRef.value) inputRef.value.value = ''
  }
}

const pickFiles = () => {
  rememberRange()
  inputRef.value?.click()
}

const onPicked = () => uploadFiles(Array.from(inputRef.value?.files || []))

const rangeFromPoint = (x: number, y: number): Range | null => {
  const doc = document as Document & {
    caretRangeFromPoint?: (x: number, y: number) => Range | null
    caretPositionFromPoint?: (x: number, y: number) => { offsetNode: Node; offset: number } | null
  }
  if (typeof doc.caretRangeFromPoint === 'function') {
    return doc.caretRangeFromPoint(x, y) || null
  }
  if (typeof doc.caretPositionFromPoint === 'function') {
    const pos = doc.caretPositionFromPoint(x, y)
    if (!pos) return null
    const range = document.createRange()
    range.setStart(pos.offsetNode, pos.offset)
    range.collapse(true)
    return range
  }
  return null
}

const onDrop = async (e: DragEvent) => {
  dragOver.value = false
  const files = Array.from(e.dataTransfer?.files || [])
  if (!files.length) return
  const el = bodyRef.value
  if (el) {
    const range = rangeFromPoint(e.clientX, e.clientY)
    if (range && el.contains(range.commonAncestorContainer)) savedRange = range
  }
  await uploadFiles(files)
}

const onPaste = async (e: ClipboardEvent) => {
  const data = e.clipboardData
  if (!data) return
  const files = Array.from(data.files || [])
  if (files.length) {
    e.preventDefault()
    await uploadFiles(files)
    return
  }
  const html = data.getData('text/html')
  if (html) {
    e.preventDefault()
    insertHtml(sanitizeHtml(html))
    return
  }
  const text = data.getData('text/plain')
  if (text) {
    e.preventDefault()
    insertNodes([document.createTextNode(text)])
  }
}

// ==================== 交互 ====================

const onClick = (e: MouseEvent) => {
  const target = e.target as HTMLElement | null
  if (!target) return
  if (target.tagName === 'IMG') {
    const src = (target as HTMLImageElement).src
    if (src) previewUrl.value = src
    return
  }
  const link = target.closest('a')
  if (link && link.classList.contains('pv-file')) {
    e.preventDefault()
    const href = link.getAttribute('href')
    if (href) window.open(href, '_blank', 'noopener')
  }
}

// ==================== 高度拉伸 / 双击折叠 ====================

/** 收尾函数句柄：块被删除时也能清掉全局监听与 body 样式 */
let endResize: (() => void) | null = null

/**
 * 双击判定：只有两次「短按且没有发生拖动」才算双击，避免把连续两次微调高度误判成折叠。
 * 用 pointerdown 自行判定而不是 dblclick —— pointerdown 阻止默认后 dblclick 不会再触发，触屏也支持。
 */
const DOUBLE_TAP_MS = 320
const DOUBLE_TAP_SLOP = 12
let lastTapAt = 0
let lastTapX = 0
let lastTapY = 0
let lastTapWasPlain = false

const startResize = (ev: PointerEvent) => {
  if (ev.pointerType === 'mouse' && ev.button !== 0) return
  const el = rootRef.value
  const handle = resizeRef.value
  if (!el || !handle || resizing.value) return
  ev.preventDefault()
  const now = Date.now()
  const isDoubleTap =
    lastTapWasPlain &&
    now - lastTapAt < DOUBLE_TAP_MS &&
    Math.abs(ev.clientX - lastTapX) < DOUBLE_TAP_SLOP &&
    Math.abs(ev.clientY - lastTapY) < DOUBLE_TAP_SLOP
  lastTapWasPlain = false
  if (isDoubleTap) {
    lastTapAt = 0
    toggleHeight()
    return
  }
  const pointerId = ev.pointerId
  const startY = ev.clientY
  const startH = el.offsetHeight
  const minH = oneLineHeight()
  let moved = false
  resizing.value = true
  document.body.style.cursor = 'row-resize'
  document.body.style.userSelect = 'none'
  // 指针捕获：触摸时手指滑出细把手、或滚动容器想接管手势，事件仍然持续发到把手上
  try {
    handle.setPointerCapture(pointerId)
  } catch {
    // 捕获不可用时仍由 window 监听兜底
  }
  const onMove = (e: PointerEvent) => {
    if (e.pointerId !== pointerId) return
    if (Math.abs(e.clientY - startY) > 3) moved = true
    emit('update:height', Math.max(minH, startH + (e.clientY - startY)))
  }
  const stop = (e: PointerEvent) => {
    if (e.pointerId !== pointerId) return
    // 只有正常抬起且全程没拖动才算一次「短按」；取消 / 捕获丢失的坐标不可靠，直接作废
    if (e.type === 'pointerup') {
      lastTapAt = Date.now()
      lastTapX = e.clientX
      lastTapY = e.clientY
      lastTapWasPlain = !moved
    } else {
      lastTapWasPlain = false
    }
    endResize?.()
  }
  endResize = () => {
    resizing.value = false
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', stop)
    window.removeEventListener('pointercancel', stop)
    window.removeEventListener('lostpointercapture', stop)
    try {
      if (handle.hasPointerCapture(pointerId)) handle.releasePointerCapture(pointerId)
    } catch {
      // 指针已失效或捕获已自动释放
    }
    endResize = null
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', stop)
  window.addEventListener('pointercancel', stop)
  window.addEventListener('lostpointercapture', stop)
}

const focus = () => bodyRef.value?.focus()

onMounted(() => {
  const el = bodyRef.value
  if (el) {
    el.innerHTML = props.modelValue || ''
    hasContent.value = !isBlank(el)
  }
  document.addEventListener('selectionchange', rememberRange)
})

onUnmounted(() => {
  endResize?.()
  document.removeEventListener('selectionchange', rememberRange)
})

defineExpose({ pickFiles, focus })
</script>
