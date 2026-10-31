<template>
  <div class="pv-editor" :class="{ 'is-dragging': dragOver }">
    <div
      ref="bodyRef"
      class="pv-editor-body"
      contenteditable="true"
      role="textbox"
      aria-multiline="true"
      :aria-label="placeholder"
      @input="sync"
      @paste="onPaste"
      @click="onClick"
      @dragover.prevent="dragOver = true"
      @dragleave="dragOver = false"
      @drop.prevent="onDrop"
    ></div>
    <div v-if="!hasContent" class="pv-editor-placeholder">{{ placeholder }}</div>
    <div v-if="uploading" class="pv-editor-uploading">上传中…</div>
    <input ref="inputRef" class="pv-editor-input" type="file" multiple @change="onPicked" />
    <!-- 点击图片放大：浮层显示原图（自带缩放 / 旋转） -->
    <el-image-viewer v-if="previewUrl" :url-list="[previewUrl]" teleported hide-on-click-modal @close="previewUrl = ''" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { fetchUploadMemoFile, resolveError } from '../../api'
import type { MemoFile } from '../../types'

/**
 * 便签富文本编辑区：contenteditable + 附件上传 / 图片粘贴。
 * - v-model 绑定的是 HTML 字符串（meta.mode = 'rich'）
 * - 上传的文件：图片统一高度内联显示（两侧不留空、不占整行，点击放大看原图），其它文件显示「文件图标 + 文件名」卡片
 */
const props = withDefaults(
  defineProps<{
    /** HTML 内容 */
    modelValue: string
    /** 空内容时的提示文案 */
    placeholder?: string
  }>(),
  {
    placeholder: '记录灵感、账号备注、常用文本……支持上传文件与粘贴图片，输入后自动保存',
  }
)

const emit = defineEmits<{ (e: 'update:modelValue', value: string): void }>()

const bodyRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const dragOver = ref(false)
const hasContent = ref(false)
const previewUrl = ref('')

/** 编辑器里最后的光标位置：点外部「上传文件」按钮后，内容要插回原来的位置 */
let savedRange: Range | null = null

const isBlank = (el: HTMLElement) => !(el.textContent || '').trim() && !el.querySelector('img, a')

/** 把编辑器内容同步给外部（v-model） */
const sync = () => {
  const el = bodyRef.value
  if (!el) return
  hasContent.value = !isBlank(el)
  emit('update:modelValue', el.innerHTML)
}

/** 外部内容变化（加载 / 恢复历史版本 / 清空）时写回编辑器 */
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

/** 记住光标位置（只在选区落在编辑区内时更新，点外部按钮不会把它清掉） */
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

/** 在光标处插入节点，并把光标移到插入内容之后 */
const insertNodes = (nodes: Node[]) => {
  const el = bodyRef.value
  if (!el || !nodes.length) return
  const range = resolveRange(el)
  range.deleteContents()
  const frag = document.createDocumentFragment()
  nodes.forEach((node) => frag.appendChild(node))
  range.insertNode(frag)
  // 光标落在插入内容之后，继续输入不会被插到前面
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

/** 直接丢掉（含内容）的标签：脚本、页面级标签、可交互控件 */
const DROP_TAGS = new Set([
  'SCRIPT', 'STYLE', 'IFRAME', 'OBJECT', 'EMBED', 'LINK', 'META', 'BASE',
  'SVG', 'MATH', 'FORM', 'INPUT', 'BUTTON', 'SELECT', 'TEXTAREA', 'VIDEO',
  'AUDIO', 'CANVAS', 'NOSCRIPT', 'TEMPLATE',
])

/** 允许保留的标签，其余标签拆掉、只留文字 */
const KEEP_TAGS = new Set([
  'DIV', 'P', 'BR', 'SPAN', 'B', 'STRONG', 'I', 'EM', 'U', 'S', 'STRIKE', 'DEL',
  'A', 'IMG', 'UL', 'OL', 'LI', 'H1', 'H2', 'H3', 'H4', 'BLOCKQUOTE', 'CODE',
  'PRE', 'HR',
])

/** 允许保留的属性（其余一律去掉，避免事件属性 / 内联脚本） */
const KEEP_ATTR: Record<string, string[]> = {
  A: ['href', 'title', 'target', 'download', 'class', 'contenteditable'],
  IMG: ['src', 'alt', 'class'],
  SPAN: ['class'],
  DIV: ['class'],
}

const isSafeUrl = (raw: string): boolean => {
  const url = (raw || '').trim()
  if (!url) return false
  // 站内相对地址（便签附件就是 /api/memoFile/xxx）
  if (url.startsWith('/')) return true
  // 外链图片粘贴进来时保留原地址，data: 一律不要（既撑大数据库，也可能是伪协议）
  return /^https?:\/\//i.test(url) || /^mailto:/i.test(url)
}

/** 只保留自己生成的类名（pv-*），避免粘贴进来的样式串到便签上 */
const isSafeClass = (value: string) => value.split(/\s+/).filter(Boolean).every((name) => name.startsWith('pv-'))

/** 清洗粘贴进来的 HTML：去脚本 / 去事件属性 / 去伪协议 */
const sanitizeHtml = (html: string): string => {
  const doc = new DOMParser().parseFromString(html, 'text/html')
  doc.body.querySelectorAll(Array.from(DROP_TAGS).join(',')).forEach((el) => el.remove())
  // 先收集再拆标签：拆掉外层后内层元素依然在列表里，一次遍历即可
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

/** 附件卡片的图标文字：取扩展名，最多 5 个字符 */
const extLabel = (name: string) => {
  const matched = /\.([A-Za-z0-9]{1,5})$/.exec(name || '')
  return matched ? matched[1].toUpperCase() : 'FILE'
}

/** 图片节点：不写死宽高，显示尺寸由 CSS 统一高度、宽度按比例（不占整行，可与文字/其它图片同行），点击可看原图 */
const imageNodes = (file: MemoFile): Node[] => {
  const img = document.createElement('img')
  img.src = file.url
  img.alt = file.original || '图片'
  img.loading = 'lazy'
  return [img]
}

/** 附件节点：文件图标 + 文件名，点击在新窗口打开（非图片由后端按下载返回） */
const fileNodes = (file: MemoFile): Node[] => {
  const link = document.createElement('a')
  link.className = 'pv-file'
  link.href = file.url
  link.title = `${file.original}（${formatSize(file.size)}）`
  link.setAttribute('target', '_blank')
  link.setAttribute('rel', 'noreferrer')
  // 让整张卡片像一个小整体，光标不会插进图标与文件名中间
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

/** 打开系统文件选择框（便签工具栏的「上传文件」按钮调用） */
const pickFiles = () => {
  rememberRange()
  inputRef.value?.click()
}

const onPicked = () => uploadFiles(Array.from(inputRef.value?.files || []))

/** 拖拽落点位置（不同浏览器接口不一样） */
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
  // 剪贴板里是文件（截图 / 复制的图片 / 复制的文件）：直接上传
  const files = Array.from(data.files || [])
  if (files.length) {
    e.preventDefault()
    await uploadFiles(files)
    return
  }
  // 复制的网页片段：清洗后按原样式插入
  const html = data.getData('text/html')
  if (html) {
    e.preventDefault()
    insertHtml(sanitizeHtml(html))
    return
  }
  // 纯文本：自己插入，保证换行原样保留（不依赖浏览器的默认粘贴行为）
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
  // 点图片：浮层显示原图
  if (target.tagName === 'IMG') {
    const src = (target as HTMLImageElement).src
    if (src) previewUrl.value = src
    return
  }
  // 点附件卡片：新窗口打开 / 下载（在编辑区里链接默认点不动）
  const link = target.closest('a')
  if (link && link.classList.contains('pv-file')) {
    e.preventDefault()
    const href = link.getAttribute('href')
    if (href) window.open(href, '_blank', 'noopener')
  }
}

onMounted(() => {
  const el = bodyRef.value
  if (el) {
    el.innerHTML = props.modelValue || ''
    hasContent.value = !isBlank(el)
  }
  document.addEventListener('selectionchange', rememberRange)
})

onUnmounted(() => document.removeEventListener('selectionchange', rememberRange))

defineExpose({ pickFiles })
</script>
