<template>
  <el-card shadow="never" class="admin-card">
    <template #header>
      <div class="card-header">
        <div class="card-header-left">
          <span class="card-header-title">{{ `当前共 ${allTools.length} 条` }}</span>
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
      row-key="id"
      @selection-change="onSelectionChange"
    >
      <el-table-column type="selection" width="46" />
      <el-table-column label="排序" width="60" align="center">
        <template #default>
          <el-icon class="drag-handle"><Rank /></el-icon>
        </template>
      </el-table-column>
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column label="名称" min-width="170">
        <template #default="{ row }">
          <div class="tool-name-cell">
            <el-image v-if="row.logo" class="tool-logo" :src="getLogoUrl(row.logo)" fit="cover" lazy>
              <template #error>
                <div class="tool-logo-error">🖼️</div>
              </template>
            </el-image>
            <span class="tool-name-text">{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="分类" width="120">
        <template #default="{ row }">{{ displayCatelog(row.catelog) }}</template>
      </el-table-column>
      <el-table-column prop="url" label="网址" min-width="220" show-overflow-tooltip />
      <el-table-column label="隐藏" width="80">
        <template #default="{ row }">{{ row.hide ? '是' : '否' }}</template>
      </el-table-column>
      <el-table-column label="默认" width="80">
        <template #default="{ row }">{{ row.default ? '是' : '否' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">修改</el-button>
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
      <el-form-item label="名称" prop="name">
        <el-input v-model="addForm.name" placeholder="请输入工具名称" />
      </el-form-item>
      <el-form-item label="网址" prop="url">
        <el-input v-model="addForm.url" placeholder="请输入完整 URL（以 http:// 或 https:// 开头）" />
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
        <el-input v-model="addForm.desc" placeholder="请输入描述" />
      </el-form-item>
      <el-form-item label="排序" prop="sort">
        <el-tooltip content="升序，按数字从小到大排序" placement="top">
          <el-input-number v-model="addForm.sort" :min="0" :step="1" controls-position="right" />
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

  <el-dialog v-model="showEdit" title="修改工具" width="560px" destroy-on-close>
    <el-form ref="editFormRef" :model="editForm" :rules="toolRules" label-width="90px">
      <el-form-item label="序号">
        <el-input v-model="editForm.id" disabled />
      </el-form-item>
      <el-form-item label="名称" prop="name">
        <el-input v-model="editForm.name" placeholder="请输入工具名称" />
      </el-form-item>
      <el-form-item label="网址" prop="url">
        <el-input v-model="editForm.url" placeholder="请输入 url" />
      </el-form-item>
      <el-form-item label="logo 网址" prop="logo">
        <el-input v-model="editForm.logo" placeholder="请输入 logo url，为空则自动获取" />
      </el-form-item>
      <el-form-item label="分类" prop="catelog">
        <el-select v-model="editForm.catelog" placeholder="请选择分类" style="width: 100%">
          <el-option v-for="opt in catelogOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
        </el-select>
      </el-form-item>
      <el-form-item label="描述" prop="desc">
        <el-input v-model="editForm.desc" placeholder="请输入描述" />
      </el-form-item>
      <el-form-item label="排序" prop="sort">
        <el-input-number v-model="editForm.sort" :min="0" :step="1" controls-position="right" />
      </el-form-item>
      <el-form-item label="隐藏">
        <el-switch v-model="editForm.hide" inline-prompt active-text="开" inactive-text="关" />
      </el-form-item>
      <el-form-item label="默认">
        <el-switch v-model="editForm.default" inline-prompt active-text="开" inactive-text="关" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="showEdit = false">取消</el-button>
      <el-button type="primary" :loading="requestLoading" @click="handleUpdate">确定</el-button>
    </template>
  </el-dialog>

</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Download, Plus, Rank, Refresh, Upload } from '@element-plus/icons-vue'
import type { FormInstance, FormRules, UploadRawFile } from 'element-plus'
import {
  fetchAddTool,
  fetchDeleteTool,
  fetchExportTools,
  fetchImportTools,
  fetchUpdateTool,
  fetchUpdateToolsSort,
  resolveError,
} from '../../api'
import { useAdminStore } from '../../stores/admin'
import { useTableSortable } from '../../composables/useTableSortable'
import { multiSearch } from '../../utils/match'
import { displayCatelog, getLogoUrl } from '../../utils/check'
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
  sort: 1,
  hide: false,
  default: false,
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
const editFormRef = ref<FormInstance>()
const addForm = reactive<ToolForm>(createEmptyForm())
const editForm = reactive<ToolForm>(createEmptyForm())

const searchString = ref('')
const catelogName = ref('')
const selectedRows = ref<Tool[]>([])
const requestLoading = ref(false)
const showAdd = ref(false)
const showEdit = ref(false)
const page = ref(1)
const pageSize = ref(10)

const toolRules: FormRules = {
  name: [{ required: true, message: '请填写名称', trigger: 'blur' }],
  url: [
    { required: true, message: '请填写网址', trigger: 'blur' },
    { pattern: /^https?:\/\//, message: '网址必须以 http:// 或 https:// 开头', trigger: 'blur' },
  ],
  catelog: [{ required: true, message: '请选择分类', trigger: 'change' }],
  desc: [{ required: true, message: '请填写描述', trigger: 'blur' }],
  sort: [{ required: true, message: '请填写排序', trigger: 'change' }],
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

const persistSort = async (movedId: number, targetId: number) => {
  const list = [...sortedTools.value]
  const from = list.findIndex((item) => item.id === movedId)
  const to = list.findIndex((item) => item.id === targetId)
  if (from < 0 || to < 0) {
    return
  }
  const [moved] = list.splice(from, 1)
  list.splice(to, 0, moved)
  const updates = list.map((item, index) => ({ id: item.id, sort: index + 1 }))
  try {
    await fetchUpdateToolsSort(updates)
    ElMessage.success('排序更新成功')
  } catch (error) {
    ElMessage.error(resolveError(error, '排序更新失败'))
  } finally {
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

// ==================== 增删改 ====================

const openAdd = () => {
  Object.assign(addForm, createEmptyForm())
  showAdd.value = true
}

const openEdit = (row: Tool) => {
  Object.assign(editForm, createEmptyForm(), row)
  showEdit.value = true
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

const handleUpdate = async () => {
  if (!(await validateForm(editFormRef.value))) {
    return
  }
  requestLoading.value = true
  try {
    const res = await fetchUpdateTool({ ...editForm })
    if (res.success === false) {
      ElMessage.warning(res.errorMessage || '更新失败')
      return
    }
    ElMessage.success('更新成功! Logo 将在 3 秒后刷新并加载！')
    showEdit.value = false
    await reload()
    setTimeout(reload, 3000)
  } catch (error) {
    ElMessage.warning(resolveError(error, '更新失败'))
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
