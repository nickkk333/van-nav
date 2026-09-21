<template>
  <el-card shadow="never" class="admin-card">
    <template #header>
      <div class="card-header">
        <div class="card-header-left">
          <span class="card-header-title">{{ `当前共 ${allTools.length} 条` }}</span>
          <span class="card-header-tip">点击单元格即可直接修改，失焦/回车后自动保存</span>
          <template v-if="selectedRows.length">
            <el-popconfirm title="确定删除这些吗？" @confirm="handleBulkDelete">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
            <el-popconfirm title="确定重置这些的图标吗？（会自动获取网站默认的）" @confirm="handleBulkResetLogo">
              <template #reference>
                <el-button link type="primary">重置默认图标</el-button>
              </template>
            </el-popconfirm>
            <el-popconfirm title="确定重新缓存这些的图标吗？（会自动获取图标缓存到数据库）" @confirm="handleBulkCacheLogo">
              <template #reference>
                <el-button link type="primary">重置缓存图标</el-button>
              </template>
            </el-popconfirm>
          </template>
        </div>
        <div class="card-header-right">
          <el-select v-model="catelogName" class="filter-select" placeholder="分类筛选" clearable>
            <el-option v-for="opt in catelogOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
          <el-input v-model="searchString" class="filter-input" placeholder="搜索名称/描述" clearable />
          <el-button type="primary" :icon="Plus" @click="openAdd">添加</el-button>
          <el-button :icon="Refresh" @click="reload">刷新</el-button>
          <el-upload accept=".json" :show-file-list="false" :before-upload="handleImportFile">
            <el-button :icon="Upload">导入</el-button>
          </el-upload>
          <el-button :icon="Download" @click="handleExport">导出</el-button>
        </div>
      </div>
    </template>

    <el-table
      ref="tableRef"
      v-loading="loading"
      :data="pagedData"
      :row-class-name="rowClassName"
      row-key="id"
      @selection-change="onSelectionChange"
    >
      <el-table-column type="selection" width="46" />
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
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column label="名称" min-width="160">
        <template #default="{ row }">
          <div class="tool-name-cell">
            <el-image v-if="row.logo" class="tool-logo" :src="getLogoUrl(row.logo)" fit="cover" lazy>
              <template #error>
                <div class="tool-logo-error">🖼️</div>
              </template>
            </el-image>
            <el-input v-model="row.name" placeholder="请输入名称" @change="saveRow(row)" />
          </div>
        </template>
      </el-table-column>
      <el-table-column label="分类" width="130">
        <template #default="{ row }">
          <el-select
            v-model="row.catelog"
            placeholder="未分类"
            clearable
            style="width: 100%"
            @change="saveRow(row)"
          >
            <el-option v-for="opt in catelogOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="网址" min-width="190">
        <template #default="{ row }">
          <el-input v-model="row.url" placeholder="https://" @change="saveRow(row)" />
        </template>
      </el-table-column>
      <el-table-column label="描述" min-width="150">
        <template #default="{ row }">
          <el-input v-model="row.desc" placeholder="请输入描述" @change="saveRow(row)" />
        </template>
      </el-table-column>
      <el-table-column label="logo 网址" min-width="170">
        <template #header>
          <span class="column-with-tip">
            logo 网址
            <el-tooltip content="为空则保存后自动获取网站图标" placement="top">
              <el-icon><QuestionFilled /></el-icon>
            </el-tooltip>
          </span>
        </template>
        <template #default="{ row }">
          <el-input v-model="row.logo" placeholder="留空则自动获取" @change="saveRow(row)" />
        </template>
      </el-table-column>
      <el-table-column width="76" align="center">
        <template #header>
          <span class="column-with-tip">
            隐藏
            <el-tooltip content="开启后只有登录后才会展示该工具" placement="top">
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
            <el-tooltip content="开启后在默认页展示" placement="top">
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
          <el-popconfirm :title="`确定要删除 ${row.name} 吗？`" @confirm="handleDelete(row.id)">
            <template #reference>
              <el-button link type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-model:current-page="page"
      v-model:page-size="pageSize"
      class="table-pagination"
      background
      :page-sizes="[10, 20, 50, 100]"
      :total="filteredTools.length"
      layout="total, sizes, prev, pager, next, jumper"
    />
  </el-card>

  <el-dialog v-model="showAdd" title="新建工具" width="560px" destroy-on-close>
    <el-form ref="addFormRef" :model="addForm" :rules="toolRules" label-width="90px">
      <el-form-item label="网址" prop="url">
        <div class="url-field">
          <el-input
            v-model="addForm.url"
            placeholder="请输入完整 URL（以 http:// 或 https:// 开头）"
            @change="autoFillFromUrl()"
          >
            <template #suffix>
              <el-icon v-if="urlInfoLoading" class="is-loading"><Loading /></el-icon>
            </template>
          </el-input>
          <div class="form-tip">
            <span>输入网址后会自动获取名称、描述和图标</span>
            <el-button
              link
              type="primary"
              :disabled="!addForm.url"
              :loading="urlInfoLoading"
              @click="autoFillFromUrl()"
            >
              重新获取
            </el-button>
          </div>
        </div>
      </el-form-item>
      <el-form-item label="名称" prop="name">
        <el-input v-model="addForm.name" placeholder="请输入工具名称" />
      </el-form-item>
      <el-form-item label="logo 网址" prop="logo">
        <el-input v-model="addForm.logo" placeholder="请输入 logo url，为空则自动获取" />
      </el-form-item>
      <el-form-item label="分类" prop="catelog">
        <el-select v-model="addForm.catelog" placeholder="请选择分类" style="width: 100%">
          <el-option v-for="opt in catelogOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
        </el-select>
      </el-form-item>
      <el-form-item label="描述" prop="desc">
        <el-input v-model="addForm.desc" placeholder="选填，可自动获取" />
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
        <el-tooltip content="开启后只有登录后才会展示该工具" placement="top">
          <el-switch v-model="addForm.hide" inline-prompt active-text="开" inactive-text="关" />
        </el-tooltip>
      </el-form-item>
      <el-form-item label="默认">
        <el-tooltip content="开启后在默认页展示" placement="top">
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
import { Download, Loading, Plus, QuestionFilled, Rank, Refresh, Upload } from '@element-plus/icons-vue'
import type { FormInstance, FormRules, UploadRawFile } from 'element-plus'
import {
  fetchAddTool,
  fetchDeleteTool,
  fetchExportTools,
  fetchGetUrlInfo,
  fetchImportTools,
  fetchUpdateTool,
  fetchUpdateToolsSort,
  resolveError,
} from '../../api'
import { useAdminStore } from '../../stores/admin'
import { useTableSortable } from '../../composables/useTableSortable'
import { multiSearch } from '../../utils/match'
import { getLogoUrl } from '../../utils/check'
import type { Tool } from '../../types'

interface ToolForm {
  id?: number
  name: string
  url: string
  logo: string
  catelog: string
  desc: string
  sort: number
  hide: boolean
  default: boolean
}

const createEmptyForm = (): ToolForm => ({
  name: '',
  url: '',
  logo: '',
  catelog: '',
  desc: '',
  // -1 表示新增后自动排到最后；0 或留空表示排到最前；正数表示插入到该序号位置
  sort: -1,
  hide: false,
  // 新工具默认展示在默认页
  default: true,
})

const adminStore = useAdminStore()
const store = computed(() => adminStore.store)
const loading = computed(() => adminStore.loading)
const allTools = computed<Tool[]>(() => store.value.tools ?? [])
const catelogOptions = computed(() =>
  (store.value.catelogs ?? []).map((item) => ({ label: item.name, value: item.name }))
)

const tableRef = ref()
const addFormRef = ref<FormInstance>()
const addForm = reactive<ToolForm>(createEmptyForm())

const searchString = ref('')
const catelogName = ref('')
const selectedRows = ref<Tool[]>([])
const requestLoading = ref(false)
const showAdd = ref(false)
const page = ref(1)
const pageSize = ref(10)
/** 添加工具时是否正在抓取网址信息 */
const urlInfoLoading = ref(false)

/** 每行「服务端已保存」的快照，用于判断是否被改动 / 撤销 */
const snapshotMap = reactive<Record<number, Tool>>({})
/** 每行是否正在保存 */
const savingMap = reactive<Record<number, boolean>>({})

const toolRules: FormRules = {
  name: [{ required: true, message: '请填写名称', trigger: 'blur' }],
  url: [
    { required: true, message: '请填写网址', trigger: 'blur' },
    { pattern: /^https?:\/\//, message: '网址必须以 http:// 或 https:// 开头', trigger: 'blur' },
  ],
  catelog: [{ required: true, message: '请选择分类', trigger: 'change' }],
}

const sortedTools = computed(() => [...allTools.value].sort((a, b) => (a.sort ?? 0) - (b.sort ?? 0)))

const filteredTools = computed(() =>
  sortedTools.value.filter((item) => {
    const matchSearch =
      !searchString.value.trim() ||
      multiSearch(item.name, searchString.value) ||
      multiSearch(item.desc, searchString.value)
    const matchCatelog = !catelogName.value || multiSearch(item.catelog, catelogName.value)
    return matchSearch && matchCatelog
  })
)

const pagedData = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredTools.value.slice(start, start + pageSize.value)
})

watch([searchString, catelogName, pageSize], () => {
  page.value = 1
})

const loadData = async (force = false) => {
  try {
    await adminStore.load(force)
  } catch (error) {
    ElMessage.error(resolveError(error, '加载数据失败'))
  }
}

const reload = () => loadData(true)

// ==================== 拖拽排序 ====================

/** 按给定顺序把排序值重写成「从 1 开始依次递增」，并同步本地快照（返回给接口的数据） */
const buildSortUpdates = (list: Tool[]) => {
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
  const list = [...sortedTools.value]
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
    await fetchUpdateToolsSort(updates)
    ElMessage({ message: '排序已更新', type: 'success', grouping: true, duration: 1200 })
  } catch (error) {
    ElMessage.error(resolveError(error, '排序更新失败'))
    // 失败时以服务端数据为准
    await reload()
  }
}

useTableSortable({
  tableRef,
  rows: pagedData,
  onEnd: (oldIndex, newIndex) => {
    const moved = pagedData.value[oldIndex]
    const target = pagedData.value[newIndex]
    if (!moved || !target) {
      return
    }
    persistSort(moved.id, target.id)
  },
})

// ==================== 行内编辑（点击单元格直接修改） ====================

/** 取出一行的可编辑字段（用于快照比对） */
const pickRow = (row: Tool): Tool => ({
  id: row.id,
  name: row.name,
  url: row.url,
  logo: row.logo,
  catelog: row.catelog,
  desc: row.desc,
  sort: row.sort,
  hide: row.hide,
  default: row.default,
})

/** 数据（重新）加载后重建快照 */
const seedSnapshots = () => {
  Object.keys(snapshotMap).forEach((key) => delete snapshotMap[Number(key)])
  allTools.value.forEach((row) => {
    snapshotMap[row.id] = pickRow(row)
  })
}

watch(allTools, seedSnapshots, { immediate: true })

const isDirty = (row: Tool) => {
  const snap = snapshotMap[row.id]
  if (!snap) {
    return false
  }
  return (
    row.name !== snap.name ||
    row.url !== snap.url ||
    row.logo !== snap.logo ||
    row.catelog !== snap.catelog ||
    row.desc !== snap.desc ||
    row.sort !== snap.sort ||
    Boolean(row.hide) !== Boolean(snap.hide) ||
    Boolean(row.default) !== Boolean(snap.default)
  )
}

const isSaving = (row: Tool) => savingMap[row.id] === true

const rowClassName = ({ row }: { row: Tool }) => (isDirty(row) ? 'row-dirty' : '')

/** 行内保存前的校验（分类、描述允许留空，避免历史数据无法保存） */
const validateRow = (row: Tool) => {
  if (!row.name || !String(row.name).trim()) {
    return '名称不能为空'
  }
  if (!row.url || !String(row.url).trim()) {
    return '网址不能为空'
  }
  if (!/^https?:\/\//.test(String(row.url).trim())) {
    return '网址必须以 http:// 或 https:// 开头'
  }
  return ''
}

/** 保存单行（输入框失焦/回车、选择器与开关变化时自动触发） */
const saveRow = async (row: Tool) => {
  if (isSaving(row)) {
    return
  }
  const errorMessage = validateRow(row)
  if (errorMessage) {
    ElMessage.warning(errorMessage)
    return
  }
  savingMap[row.id] = true
  try {
    const res = await fetchUpdateTool({
      ...pickRow(row),
      name: String(row.name).trim(),
      url: String(row.url).trim(),
      desc: String(row.desc ?? '').trim(),
    })
    if (res.success === false) {
      ElMessage.warning(res.errorMessage || '更新失败')
      return
    }
    snapshotMap[row.id] = pickRow(row)
    ElMessage({ message: '已保存', type: 'success', grouping: true, duration: 1500 })
    // logo 为空时后端会去抓取图标，稍后刷新一次拿到新图标
    if (!row.logo) {
      setTimeout(() => {
        if (!isDirty(row)) {
          reload()
        }
      }, 3000)
    }
  } catch (error) {
    ElMessage.warning(resolveError(error, '更新失败'))
  } finally {
    savingMap[row.id] = false
  }
}

/** 撤销该行未保存的修改 */
const revertRow = (row: Tool) => {
  const snap = snapshotMap[row.id]
  if (!snap) {
    return
  }
  Object.assign(row, pickRow(snap))
  ElMessage.info('已撤销未保存的修改')
}

// ==================== 增删改 ====================

const openAdd = () => {
  Object.assign(addForm, createEmptyForm())
  showAdd.value = true
}

/**
 * 根据填写的网址抓取信息：先清空名称/描述/logo 网址，再填入抓取到的内容
 * 名称直接用接口返回的 title
 */
const autoFillFromUrl = async () => {
  const target = addForm.url.trim()
  if (!target) {
    return
  }
  if (!/^https?:\/\//.test(target)) {
    ElMessage.warning('网址必须以 http:// 或 https:// 开头')
    return
  }
  // 网址变了就先把上一次抓来的内容全部清掉，避免残留旧信息
  addForm.name = ''
  addForm.desc = ''
  addForm.logo = ''
  urlInfoLoading.value = true
  try {
    const info = (await fetchGetUrlInfo(target)) ?? { name: '', title: '', description: '', logo: '' }
    const filled: string[] = []
    // 名称直接用接口返回的 title
    const name = (info.title || '').trim()
    if (name) {
      addForm.name = name
      filled.push('名称')
    }
    const desc = (info.description || '').trim()
    if (desc) {
      addForm.desc = desc
      filled.push('描述')
    }
    const logo = (info.logo || '').trim()
    if (logo) {
      addForm.logo = logo
      filled.push('图标')
    }
    if (filled.length) {
      ElMessage.success(`已自动填充：${filled.join('、')}`)
    } else {
      ElMessage.warning('没能读取到网站信息，请手动填写')
    }
  } catch (error) {
    ElMessage.warning(resolveError(error, '获取网址信息失败'))
  } finally {
    urlInfoLoading.value = false
  }
}

const validateForm = async (formRef?: FormInstance) => {
  if (!formRef) {
    return false
  }
  return Boolean(await formRef.validate().catch(() => false))
}

const handleCreate = async () => {
  if (!(await validateForm(addFormRef.value))) {
    return
  }
  requestLoading.value = true
  try {
    // 排序落点（-1/负数排到最后、0 或留空排到最前、正数插入到该序号）与全表排序值重排
    // 统一由后端 /admin/tool 处理，前端只负责提交表单
    const res = await fetchAddTool({ ...addForm })
    if (res.success === false) {
      ElMessage.warning(res.errorMessage || '添加失败')
      return
    }
    ElMessage.success('添加成功! Logo 将在 3 秒后刷新并加载！')
    showAdd.value = false
    await reload()
    setTimeout(reload, 3000)
  } catch (error) {
    ElMessage.warning(resolveError(error, '添加失败'))
  } finally {
    requestLoading.value = false
  }
}

const handleDelete = async (id: number) => {
  try {
    await fetchDeleteTool(id)
    ElMessage.success('删除成功!')
  } catch (error) {
    ElMessage.warning(resolveError(error, '删除失败'))
  } finally {
    await reload()
  }
}

const onSelectionChange = (rows: Tool[]) => {
  selectedRows.value = rows
}

const handleBulkDelete = async () => {
  for (const each of selectedRows.value) {
    try {
      await fetchDeleteTool(each.id)
    } catch (error) {
      console.error(error)
    }
  }
  ElMessage.success('删除成功!')
  await reload()
}

const handleBulkResetLogo = async () => {
  for (const each of selectedRows.value) {
    try {
      await fetchUpdateTool({ ...each, logo: '' })
    } catch (error) {
      console.error(error)
    }
  }
  ElMessage.success('重置成功!')
  await reload()
}

const handleBulkCacheLogo = async () => {
  for (const each of selectedRows.value) {
    try {
      await fetchUpdateTool(each)
    } catch (error) {
      console.error(error)
    }
  }
  ElMessage.success('缓存成功!')
  await reload()
}

// ==================== 导入导出 ====================

const handleImportFile = (file: UploadRawFile) => {
  const reader = new FileReader()
  reader.readAsText(file)
  reader.onload = async (result) => {
    const content = result?.target?.result
    if (!content) {
      return
    }
    try {
      await fetchImportTools(JSON.parse(content as string))
      ElMessage.success('导入成功!')
    } catch (error) {
      ElMessage.warning(resolveError(error, '导入失败'))
    } finally {
      await reload()
    }
  }
  return false
}

const handleExport = async () => {
  try {
    const data = await fetchExportTools()
    const blob = new Blob([JSON.stringify(data)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'tools.json'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    ElMessage.success('导出成功！')
  } catch (error) {
    ElMessage.warning(resolveError(error, '导出失败'))
  }
}

onMounted(() => {
  loadData()
})
</script>
