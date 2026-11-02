<template>
  <div class="pv-editor">
    <div class="pv-editor-blocks">
      <MemoBlock
        v-for="(b, i) in blocks"
        :key="b.id"
        :ref="(el) => setBlockRef(b.id, el)"
        v-model="b.html"
        :height="b.height"
        :color-index="i"
        :active="b.id === activeId"
        :removable="blocks.length > 1"
        @update:height="(h) => (b.height = h)"
        @focus="activeId = b.id"
        @remove="removeBlock(b.id)"
      />
    </div>
    <button type="button" class="pv-add-block" title="添加块" aria-label="添加块" @click="addBlock">
      <el-icon :size="14"><Plus /></el-icon><span>添加块</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import MemoBlock from './MemoBlock.vue'

/**
 * 便签块编辑器：便签由多个「块」组成，每个块是独立的富文本区域（带圆角矩形、可拉伸高度）。
 * - v-model 绑定的是整篇 HTML（兼容后端 content 字符串），内部按 <div class="pv-block"> 切块
 * - 「上传文件」按钮由父级工具栏触发 pickFiles()，插入到当前聚焦的块
 */
interface Block {
  id: string
  html: string
  height: number | null
}

const props = defineProps<{
  /** 整篇便签 HTML */
  modelValue: string
}>()

const emit = defineEmits<{ (e: 'update:modelValue', value: string): void }>()

const blocks = ref<Block[]>([])
const activeId = ref('')
const blockRefs = new Map<string, InstanceType<typeof MemoBlock>>()

let seq = 0
const genId = () => `b${Date.now().toString(36)}-${(seq++).toString(36)}`

const setBlockRef = (id: string, el: unknown) => {
  if (el) blockRefs.set(id, el as InstanceType<typeof MemoBlock>)
  else blockRefs.delete(id)
}

/** 块列表 → 整篇 HTML */
const serialize = () =>
  blocks.value
    .map((b) => {
      const h = b.height ? ` data-h="${Math.round(b.height)}"` : ''
      return `<div class="pv-block"${h}>${b.html || ''}</div>`
    })
    .join('')

/** 整篇 HTML → 块列表（兼容老数据：没有块结构时整篇作为一个块） */
const parseBlocks = (html: string): Block[] => {
  const doc = new DOMParser().parseFromString(html || '', 'text/html')
  const blockEls = Array.from(doc.body.children).filter((el) => el.classList.contains('pv-block'))
  if (!blockEls.length) {
    return [{ id: genId(), html: html || '', height: null }]
  }
  return blockEls.map((el) => ({
    id: genId(),
    html: el.innerHTML,
    height: parseFloat(el.getAttribute('data-h') || '') || null,
  }))
}

/** 内容变化后把整篇 HTML 同步给父级 */
const emitHtml = () => emit('update:modelValue', serialize())

// 块内容 / 高度变化时同步到 v-model
watch(
  blocks,
  () => emitHtml(),
  { deep: true }
)

// 外部整篇内容变化（加载 / 恢复历史版本）时重新切块；自己 emit 的忽略，避免光标丢失
watch(
  () => props.modelValue,
  (value) => {
    if (serialize() === (value || '')) return
    blocks.value = parseBlocks(value || '')
    if (!blocks.value.length) blocks.value = [{ id: genId(), html: '', height: null }]
    activeId.value = blocks.value[0]?.id || ''
  }
)

/** 添加一个空块并聚焦 */
const addBlock = async () => {
  const block: Block = { id: genId(), html: '', height: null }
  blocks.value.push(block)
  activeId.value = block.id
  await nextTick()
  blockRefs.get(block.id)?.focus()
}

/** 删除块：删除后至少保留一个块 */
const removeBlock = (id: string) => {
  if (blocks.value.length <= 1) return
  const idx = blocks.value.findIndex((b) => b.id === id)
  if (idx < 0) return
  blocks.value.splice(idx, 1)
  if (activeId.value === id) {
    activeId.value = (blocks.value[idx] || blocks.value[idx - 1])?.id || ''
  }
  emitHtml()
}

/** 工具栏「上传文件」：上传到当前聚焦的块 */
const pickFiles = () => {
  blockRefs.get(activeId.value)?.pickFiles()
}

// 初始化
blocks.value = parseBlocks(props.modelValue || '')
if (!blocks.value.length) blocks.value = [{ id: genId(), html: '', height: null }]
activeId.value = blocks.value[0]?.id || ''

defineExpose({ pickFiles })
</script>
