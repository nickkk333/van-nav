<template>
  <el-popover placement="bottom-end" :width="230" trigger="click">
    <template #reference>
      <el-button :icon="Setting" :loading="store.saving">列设置</el-button>
    </template>
    <div class="col-setting">
      <div class="col-setting-head">
        <span class="col-setting-title">显示列</span>
        <el-button link type="primary" size="small" @click="restoreDefault">恢复默认</el-button>
      </div>
      <el-checkbox-group v-model="checked" class="col-setting-list" @change="onChange">
        <el-checkbox v-for="col in columns" :key="col.key" :value="col.key" class="col-setting-item">
          {{ col.label }}
        </el-checkbox>
      </el-checkbox-group>
      <div class="col-setting-tip">配置保存在服务端，多个设备端自动同步</div>
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Setting } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useTableColumnsStore } from '../stores/tableColumns'
import type { TableColumnOption, TableKey } from '../stores/tableColumns'

const props = defineProps<{
  /** 表格标识：决定读写哪一组隐藏列 */
  tableKey: TableKey
  /** 该表格可配置的所有列 */
  columns: TableColumnOption[]
}>()

const store = useTableColumnsStore()
/** 当前勾选（=显示）的列 key */
const checked = ref<string[]>([])

const visibleKeys = () =>
  props.columns.filter((col) => store.isVisible(props.tableKey, col.key)).map((col) => col.key)

/** 与 store 同步：其它设备改了配置后，勾选状态也要跟着变 */
const syncChecked = () => {
  const next = visibleKeys()
  if (next.length !== checked.value.length || next.some((key, i) => key !== checked.value[i])) {
    checked.value = next
  }
}
watch(() => store.hidden, syncChecked, { deep: true, immediate: true })

let applying = false
const apply = async (visible: string[]) => {
  if (applying) return
  applying = true
  const hidden = props.columns.filter((col) => !visible.includes(col.key)).map((col) => col.key)
  try {
    await store.setHidden(props.tableKey, hidden)
  } catch {
    ElMessage.error('列配置保存失败，请重试')
    syncChecked()
  } finally {
    applying = false
  }
}

const onChange = () => apply(checked.value)
const restoreDefault = () => apply(props.columns.map((col) => col.key))
</script>

<!-- 不加 scoped：popover 内容会 Teleport 到 body，scoped 样式作用不到 -->
<style>
.col-setting-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.col-setting-title { font-size: 13px; font-weight: 600; }
.col-setting-list { display: flex; flex-direction: column; gap: 2px; max-height: 320px; overflow: auto; }
.col-setting-item { height: 26px; margin-right: 0; }
.col-setting-tip { margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--border-color); font-size: 12px; color: var(--text-secondary); }
</style>
