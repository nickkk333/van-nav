<template>
  <div class="pv-memo">
    <div class="pv-memo-bar">
      <span class="pv-save" :class="saveState">{{ saveText }}</span>
      <div class="pv-memo-actions">
        <el-button size="small" text @click="editorRef?.pickFiles()">上传文件</el-button>
        <el-button size="small" text @click="showVersions">历史版本</el-button>
        <el-button size="small" type="primary" :loading="saving" @click="save(true)">保存</el-button>
      </div>
    </div>
    <MemoEditor ref="editorRef" v-model="content" />
    <el-dialog v-model="verOpen" title="历史版本" width="420px" append-to-body>
      <div v-if="!versions.length" class="pv-empty">暂无历史版本</div>
      <div v-for="v in versions" :key="v.id" class="pv-ver">
        <div class="pv-ver-main"><b>{{ fmtTime(v.createdAt) }}</b><span>{{ preview(v) }}</span></div>
        <el-button size="small" text type="primary" @click="restore(v)">恢复</el-button>
        <el-button size="small" text type="danger" @click="delVer(v)">删除</el-button>
      </div>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  fetchDeleteMemoVersion,
  fetchGetMemo,
  fetchGetMemoVersions,
  fetchRestoreMemoVersion,
  fetchSaveMemo,
  resolveError,
} from '../../api'
import type { Memo, MemoVersion } from '../../types'
import MemoEditor from './MemoEditor.vue'

/** 便签存的是 HTML（mode=rich）；老数据是纯文本（mode=simple），加载时转成等价段落 */
const content = ref('')
const serverContent = ref('')
const saving = ref(false)
const versions = ref<MemoVersion[]>([])
const verOpen = ref(false)
const editorRef = ref<InstanceType<typeof MemoEditor> | null>(null)
let timer: number | null = null

const dirty = computed(() => content.value !== serverContent.value)
const saveState = computed(() => (saving.value ? 'saving' : dirty.value ? 'dirty' : 'saved'))
const saveText = computed(() => (saving.value ? '保存中…' : dirty.value ? '有未保存修改' : '已同步'))

const escapeHtml = (text: string) => text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
/** 纯文本段落化：每行一个 <div>，空行用 <br> 占位（与 contenteditable 的行为一致） */
const plainToHtml = (text: string) =>
  text
    .split('\n')
    .map((line) => `<div>${line ? escapeHtml(line) : '<br>'}</div>`)
    .join('')
/** 便签（或历史版本）转成编辑器里的 HTML */
const toEditorHtml = (m: Pick<Memo, 'content' | 'mode'>) => (m.mode === 'rich' ? m.content || '' : plainToHtml(m.content || ''))
/** 富文本取纯文本，用于历史版本列表的摘要 */
const htmlToText = (html: string) => {
  const box = document.createElement('div')
  box.innerHTML = html
  return (box.textContent || '').replace(/\s+/g, ' ').trim()
}

const load = async () => {
  try {
    const m = await fetchGetMemo()
    content.value = toEditorHtml(m)
    serverContent.value = content.value
  } catch (e) {
    console.error(e)
  }
}

const save = async (tip = false) => {
  if (saving.value) return
  if (timer) {
    window.clearTimeout(timer)
    timer = null
  }
  if (!dirty.value) {
    if (tip) ElMessage.success('已是最新')
    return
  }
  saving.value = true
  try {
    const m = await fetchSaveMemo({ content: content.value, mode: 'rich' })
    serverContent.value = toEditorHtml(m)
  } catch (e) {
    ElMessage.error(resolveError(e, '保存失败'))
  } finally {
    saving.value = false
  }
}

/** 输入后防抖自动保存 */
const scheduleSave = () => {
  if (timer) window.clearTimeout(timer)
  timer = window.setTimeout(() => save(false), 1200)
}
watch(content, scheduleSave)

const showVersions = async () => {
  try {
    versions.value = await fetchGetMemoVersions()
  } catch (e) {
    ElMessage.error(resolveError(e, '加载失败'))
  }
  verOpen.value = true
}

const preview = (v: MemoVersion) => htmlToText(toEditorHtml(v)).slice(0, 40) || '(空)'
const fmtTime = (t: number) => new Date(t).toLocaleString()

const restore = async (v: MemoVersion) => {
  try {
    await ElMessageBox.confirm('恢复该版本？当前内容会先归档，可撤销。', '恢复版本', { type: 'warning' })
  } catch {
    return
  }
  try {
    const m = await fetchRestoreMemoVersion(v.id)
    content.value = toEditorHtml(m)
    serverContent.value = content.value
    verOpen.value = false
    ElMessage.success('已恢复')
  } catch (e) {
    ElMessage.error(resolveError(e, '恢复失败'))
  }
}

const delVer = async (v: MemoVersion) => {
  try {
    await fetchDeleteMemoVersion(v.id)
    versions.value = versions.value.filter((x) => x.id !== v.id)
  } catch (e) {
    ElMessage.error(resolveError(e, '删除失败'))
  }
}

const reload = () => load()
onMounted(load)
onUnmounted(() => {
  if (timer) window.clearTimeout(timer)
})
defineExpose({ reload })
</script>
