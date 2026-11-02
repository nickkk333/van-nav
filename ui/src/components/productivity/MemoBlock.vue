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
    <span class="pv-block-resize" title="拖动调整高度" @pointerdown="startResize"></span>
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
  (e: 'update:height', value: number): void
  (e: 'focus'): void
  (e: 'remove'): void
}>()

/** 块最小高度（px）：一行文本 + 内边距 */
const MIN_H = 44

const rootRef = ref<HTMLElement | null>(null)
const bodyRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const dragOver = ref(false)
const hasContent = ref(false)
const previewUrl = ref('')
const uploading = ref(false)

/** 编辑器里最后的光标位置：点「上传文件」按钮后，内容要插回原来的位置 */
let savedRange: Range | null = null

const blockStyle = computed(() => (props.height ? { height: `${props.height}px` } : {}))

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

// ==================== 高度拉伸 ====================

const startResize = (ev: PointerEvent) => {
  if (ev.pointerType === 'mouse' && ev.button !== 0) return
  ev.preventDefault()
  const el = rootRef.value
  if (!el) return
  const startY = ev.clientY
  const startH = el.offsetHeight
  document.body.style.cursor = 'row-resize'
  document.body.style.userSelect = 'none'
  const onMove = (e: PointerEvent) => {
    emit('update:height', Math.max(MIN_H, startH + (e.clientY - startY)))
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

const focus = () => bodyRef.value?.focus()

onMounted(() => {
  const el = bodyRef.value
  if (el) {
    el.innerHTML = props.modelValue || ''
    hasContent.value = !isBlank(el)
  }
  document.addEventListener('selectionchange', rememberRange)
})

onUnmounted(() => document.removeEventListener('selectionchange', rememberRange))

defineExpose({ pickFiles, focus })
</script>
