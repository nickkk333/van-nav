<template>
  <el-card shadow="never" class="admin-card">
    <template #header>
      <div class="card-header">
        <div class="card-header-left">
          <span class="card-header-title">{{ `当前共 ${catelogs.length} 条` }}</span>
          <span class="card-header-tip">点击单元格即可直接修改，失焦/回车后自动保存；拖动左侧手柄可排序</span>
        </div>
        <div class="card-header-right">
          <el-button type="primary" :icon="Plus" @click="openAdd">添加</el-button>
          <el-button :icon="Refresh" @click="reload">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table
      ref="tableRef"
      v-loading="loading"
      :data="sortedCatelogs"
      :row-class-name="rowClassName"
      row-key="id"
    >
      <el-table-column width="92" align="center">
        <template #header>
          <span class="column-with-tip">
            排序
            <el-tooltip content="拖动左侧手柄排序，序号从 1 开始依次递增" placement="top">
              <el-icon><QuestionFilled /></el-icon>
            </el-tooltip>
          </span>
        </template>
        <template #default="{ row }">
          <div class="sort-cell">
            <el-icon class="drag-handle"><Rank /></el-icon>
            <span class="sort-value">{{ row.sort ?? 0 }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="名称" min-width="180">
        <template #default="{ row }">
          <el-input v-model="row.name" placeholder="请输入分类名称" @change="saveRow(row)" />
        </template>
      </el-table-column>
      <el-table-column width="76" align="center">
        <template #header>
          <span class="column-with-tip">
            隐藏
            <el-tooltip content="开启后只有登录后才会展示该分类下的工具，且该分类下的工具会一起被隐藏" placement="top">
              <el-icon><QuestionFilled /></el-icon>
            </el-tooltip>
          </span>
        </template>
        <template #default="{ row }">
          <el-switch v-model="row.hide" inline-prompt active-text="开" inactive-text="关" @change="saveRow(row)" />
        </template>
      </el-table-column>
      <el-table-column width="76" align="center">
        <template #header>
          <span class="column-with-tip">
            默认
            <el-tooltip
              content="开启后该分类下的工具都会展示在主页默认栏；切换后会同步设置该分类下所有工具的默认状态"
              placement="top"
            >
              <el-icon><QuestionFilled /></el-icon>
            </el-tooltip>
          </span>
        </template>
        <template #default="{ row }">
          <el-switch v-model="row.default" inline-prompt active-text="开" inactive-text="关" @change="saveRow(row)" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <span v-if="isSaving(row)" class="cell-status">保存中…</span>
          <template v-else-if="isDirty(row)">
            <el-button link type="primary" @click="saveRow(row)">保存</el-button>
            <el-button link type="info" @click="revertRow(row)">撤销</el-button>
          </template>
          <el-popconfirm
            :title="`确定要删除分类 ${row.name} 及其下的所有工具吗？`"
            @confirm="handleDelete(row.id)"
          >
            <template #reference>
              <el-button link type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="showAdd" title="新建分类" width="480px" destroy-on-close>
    <el-form ref="addFormRef" :model="addForm" :rules="rules" label-width="80px">
      <el-form-item label="名称" prop="name">
        <el-input v-model="addForm.name" placeholder="请输入分类名称" />
      </el-form-item>
      <el-form-item label="排序" prop="sort">
        <el-tooltip
          content="-1（默认）排到最后；0 或留空排到最前；正数插入到该序号位置；保存后所有排序值会重排为 1、2、3…"
          placement="top"
        >
          <el-input-number v-model="addForm.sort" :min="-1" :step="1" controls-position="right" />
        </el-tooltip>
      </el-form-item>
      <el-form-item label="隐藏">
        <el-tooltip content="开启后只有登录后才会展示该分类下的工具，同时该分类下的工具都会被隐藏" placement="top">
          <el-switch v-model="addForm.hide" inline-prompt active-text="开" inactive-text="关" />
        </el-tooltip>
      </el-form-item>
      <el-form-item label="默认">
        <el-tooltip content="开启后该分类下的工具都会展示在主页默认栏" placement="top">
          <el-switch v-model="addForm.default" inline-prompt active-text="开" inactive-text="关" />
        </el-tooltip>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="showAdd = false">取消</el-button>
      <el-button type="primary" :loading="requestLoading" @click="handleCreate">确定</el-button>
    </template>
  </el-dialog>

</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, QuestionFilled, Rank, Refresh } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import {
  fetchAddCateLog,
  fetchDeleteCatelog,
  fetchUpdateCateLog,
  fetchUpdateCatelogsSort,
  resolveError,
} from '../../api'
import { useAdminStore } from '../../stores/admin'
import { useTableSortable } from '../../composables/useTableSortable'
import type { Catelog } from '../../types'

interface CatelogForm {
  name: string
  sort: number
  hide: boolean
  default: boolean
}

/** 新建分类：排序值默认 -1（排到最后）、默认开关默认打开 */
const createEmptyForm = (): CatelogForm => ({ name: '', sort: -1, hide: false, default: true })

const adminStore = useAdminStore()
const loading = computed(() => adminStore.loading)
const catelogs = computed<Catelog[]>(() => adminStore.store.catelogs ?? [])
/** 表格数据：按排序值升序，拖动后立即按新顺序渲染 */
const sortedCatelogs = computed<Catelog[]>(() =>
  [...catelogs.value].sort((a, b) => (a.sort ?? 0) - (b.sort ?? 0))
)

const tableRef = ref<any>()
const addFormRef = ref<FormInstance>()
const addForm = reactive<CatelogForm>(createEmptyForm())
const showAdd = ref(false)
const requestLoading = ref(false)

/** 每行「服务端已保存」的快照，用于判断是否被改动 / 撤销 */
const snapshotMap = reactive<Record<number, Catelog>>({})
/** 每行是否正在保存 */
const savingMap = reactive<Record<number, boolean>>({})

const rules: FormRules = {
  name: [{ required: true, message: '请输入分类名称', trigger: 'blur' }],
  sort: [{ required: true, message: '请输入排序', trigger: 'change' }],
}

const reload = async () => {
  try {
    await adminStore.load(true)
  } catch (error) {
    ElMessage.error(resolveError(error, '加载数据失败'))
  }
}

// ==================== 拖拽排序 ====================

/** 按给定顺序把排序值重写成「从 1 开始依次递增」，返回给接口的数据 */
const buildSortUpdates = (list: Catelog[]) => {
  const updates: { id: number; sort: number }[] = []
  list.forEach((item, index) => {
    const sort = index + 1
    item.sort = sort
    // 同步快照里的排序值，避免排序后被标成「未保存」
    const snapshot = snapshotMap[item.id]
    if (snapshot) {
      snapshot.sort = sort
    }
    updates.push({ id: item.id, sort })
  })
  return updates
}

const persistSort = async (movedId: number, targetId: number) => {
  const list = [...sortedCatelogs.value]
  const from = list.findIndex((item) => item.id === movedId)
  const to = list.findIndex((item) => item.id === targetId)
  if (from < 0 || to < 0) {
    return
  }
  const [moved] = list.splice(from, 1)
  list.splice(to, 0, moved)
  // 先写回本地，表格立即按新顺序刷新；排序值从 1 开始依次递增
  const updates = buildSortUpdates(list)
  try {
    await fetchUpdateCatelogsSort(updates)
    ElMessage({ message: '排序已更新', type: 'success', grouping: true, duration: 1200 })
  } catch (error) {
    ElMessage.error(resolveError(error, '排序更新失败'))
    // 失败时以服务端数据为准
    await reload()
  }
}

useTableSortable({
  tableRef,
  rows: sortedCatelogs,
  onEnd: (oldIndex, newIndex) => {
    const moved = sortedCatelogs.value[oldIndex]
    const target = sortedCatelogs.value[newIndex]
    if (!moved || !target) {
      return
    }
    persistSort(moved.id, target.id)
  },
})

// ==================== 行内编辑（点击单元格直接修改） ====================

/** 取出分类的可编辑字段（用于快照比对） */
const pickRow = (row: Catelog): Catelog => ({
  id: row.id,
  name: row.name,
  sort: row.sort,
  hide: row.hide,
  default: row.default,
})

/** 数据（重新）加载后重建快照 */
const seedSnapshots = () => {
  Object.keys(snapshotMap).forEach((key) => delete snapshotMap[Number(key)])
  catelogs.value.forEach((row) => {
    snapshotMap[row.id] = pickRow(row)
  })
}

watch(catelogs, seedSnapshots, { immediate: true })

const isDirty = (row: Catelog) => {
  const snap = snapshotMap[row.id]
  if (!snap) {
    return false
  }
  return (
    row.name !== snap.name ||
    (row.sort ?? 0) !== (snap.sort ?? 0) ||
    Boolean(row.hide) !== Boolean(snap.hide) ||
    Boolean(row.default) !== Boolean(snap.default)
  )
}

const isSaving = (row: Catelog) => savingMap[row.id] === true

const rowClassName = ({ row }: { row: Catelog }) => (isDirty(row) ? 'row-dirty' : '')

/** 行内保存前的校验 */
const validateRow = (row: Catelog) => {
  if (!row.name || !String(row.name).trim()) {
    return '分类名称不能为空'
  }
  return ''
}

/** 该分类下的全部工具（含被隐藏的工具） */
const toolsOf = (catelog: string) => (adminStore.store.tools ?? []).filter((tool) => tool.catelog === catelog)

/** 保存单行（输入框失焦/回车、隐藏/默认开关变化时自动触发） */
const saveRow = async (row: Catelog) => {
  if (isSaving(row)) {
    return
  }
  const errorMessage = validateRow(row)
  if (errorMessage) {
    ElMessage.warning(errorMessage)
    return
  }
  const snapshot = snapshotMap[row.id]
  const oldName = snapshot?.name ?? row.name
  const oldHide = Boolean(snapshot?.hide)
  const oldDefault = Boolean(snapshot?.default)
  const newName = String(row.name).trim()
  const newHide = Boolean(row.hide)
  const newDefault = Boolean(row.default)
  savingMap[row.id] = true
  try {
    const res = await fetchUpdateCateLog({ ...pickRow(row), name: newName })
    if (res.success === false) {
      ElMessage.warning(res.errorMessage || '更新失败')
      return
    }
    row.name = newName
    // 改名 / 隐藏 / 默认变化会同步影响该分类下的工具，这里同步本地数据，避免整表重新加载
    if (oldName !== newName) {
      toolsOf(oldName).forEach((tool) => {
        tool.catelog = newName
      })
    }
    if (oldHide !== newHide) {
      toolsOf(newName).forEach((tool) => {
        tool.hide = newHide
      })
    }
    if (oldDefault !== newDefault) {
      toolsOf(newName).forEach((tool) => {
        tool.default = newDefault
      })
    }
    snapshotMap[row.id] = pickRow(row)
    ElMessage({ message: '已保存', type: 'success', grouping: true, duration: 1500 })
  } catch (error) {
    ElMessage.warning(resolveError(error, '更新失败'))
  } finally {
    savingMap[row.id] = false
  }
}

/** 撤销该行未保存的修改 */
const revertRow = (row: Catelog) => {
  const snap = snapshotMap[row.id]
  if (!snap) {
    return
  }
  Object.assign(row, pickRow(snap))
  ElMessage.info('已撤销未保存的修改')
}

const openAdd = () => {
  Object.assign(addForm, createEmptyForm())
  showAdd.value = true
}

const handleCreate = async () => {
  const valid = await addFormRef.value?.validate().catch(() => false)
  if (!valid) {
    return
  }
  requestLoading.value = true
  try {
    await fetchAddCateLog({ ...addForm })
    ElMessage.success('添加成功!')
    showAdd.value = false
  } catch (error) {
    ElMessage.warning(resolveError(error, '添加失败'))
  } finally {
    requestLoading.value = false
    await reload()
  }
}

const handleDelete = async (id: number) => {
  try {
    await fetchDeleteCatelog(id)
    ElMessage.success('删除分类成功，该分类下的工具已一并删除!')
  } catch (error) {
    ElMessage.warning(resolveError(error, '删除分类失败'))
  } finally {
    await reload()
  }
}

onMounted(() => {
  adminStore.load().catch((error) => ElMessage.error(resolveError(error, '加载数据失败')))
})
</script>
