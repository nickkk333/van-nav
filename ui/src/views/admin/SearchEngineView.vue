<template>
  <el-card shadow="never" class="admin-card">
    <template #header>
      <div class="card-header">
        <div class="card-header-left">
          <span class="card-header-title">{{ `当前共 ${engines.length} 条` }}</span>
        </div>
        <div class="card-header-right">
          <el-button type="primary" :icon="Plus" @click="openAdd">添加搜索引擎</el-button>
          <el-button :icon="Refresh" @click="loadEngines">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table ref="tableRef" v-loading="loading" :data="engines" row-key="id">
      <el-table-column label="排序" width="70" align="center">
        <template #default>
          <el-icon class="drag-handle"><Rank /></el-icon>
        </template>
      </el-table-column>
      <el-table-column label="Logo" width="80">
        <template #default="{ row }">
          <el-image class="engine-logo" :src="logoUrl(row.logo)" fit="contain">
            <template #error>
              <div class="tool-logo-error">️</div>
            </template>
          </el-image>
        </template>
      </el-table-column>
      <el-table-column prop="name" label="名称" width="140" />
      <el-table-column prop="baseUrl" label="基础 URL" min-width="220" show-overflow-tooltip />
      <el-table-column prop="queryParam" label="查询参数" width="120" />
      <el-table-column label="启用" width="100">
        <template #default="{ row }">
          <el-switch
            v-model="row.enabled"
            inline-prompt
            active-text="开"
            inactive-text="关"
            @change="(val: boolean) => handleToggleEnabled(row, val)"
          />
        </template>
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
  </el-card>

  <el-dialog v-model="showDialog" :title="editing ? '编辑搜索引擎' : '添加搜索引擎'" width="540px" destroy-on-close>
    <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" placeholder="例如：百度" />
      </el-form-item>
      <el-form-item label="基础 URL" prop="baseUrl">
        <el-input v-model="form.baseUrl" placeholder="例如：https://www.baidu.com/s" />
      </el-form-item>
      <el-form-item label="查询参数" prop="queryParam">
        <el-input v-model="form.queryParam" placeholder="例如：wd" />
      </el-form-item>
      <el-form-item label="Logo" prop="logo">
        <el-input v-model="form.logo" placeholder="例如：baidu.ico 或 https://example.com/logo.png" />
      </el-form-item>
      <el-form-item label="启用">
        <el-switch v-model="form.enabled" inline-prompt active-text="开" inactive-text="关" />
      </el-form-item>
    </el-form>
<template #footer>
      <el-button @click="showDialog = false">取消</el-button>
      <el-button type="primary" :loading="requestLoading" @click="handleSubmit">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Rank, Refresh } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import {
  fetchAddSearchEngine,
  fetchDeleteSearchEngine,
  fetchGetAllSearchEngines,
  fetchUpdateSearchEngine,
  fetchUpdateSearchEnginesSort,
  resolveError,
} from '../../api'
import { useTableSortable } from '../../composables/useTableSortable'
import { clearSearchEngineCache } from '../../utils/searchEngine'
import type { SearchEngine } from '../../types'

interface EngineForm {
  id?: number
  name: string
  baseUrl: string
  queryParam: string
  logo: string
  enabled: boolean
}

const createEmptyForm = (): EngineForm => ({
  name: '',
  baseUrl: '',
  queryParam: '',
  logo: '',
  enabled: true,
})

const engines = ref<SearchEngine[]>([])
const loading = ref(false)
const requestLoading = ref(false)
const showDialog = ref(false)
const editing = ref(false)
const tableRef = ref<any>()
const formRef = ref<FormInstance>()
const form = reactive<EngineForm>(createEmptyForm())

const rules: FormRules = {
  name: [{ required: true, message: '请输入搜索引擎名称', trigger: 'blur' }],
  baseUrl: [
    { required: true, message: '请输入基础 URL', trigger: 'blur' },
    { pattern: /^https?:\/\//, message: '基础 URL 必须以 http:// 或 https:// 开头', trigger: 'blur' },
  ],
  queryParam: [{ required: true, message: '请输入查询参数', trigger: 'blur' }],
  logo: [
    { required: true, message: '请输入 Logo 文件名或网址', trigger: 'blur' },
    {
      validator: (_rule, value: string, callback) => {
        if (!value) {
          return callback()
        }
        const urlPattern = /^https?:\/\/.+/i
        const filePattern = /\.(ico|png|jpg|jpeg|gif|svg|webp)$/i
        if (urlPattern.test(value) || filePattern.test(value)) {
          return callback()
        }
        callback(new Error('请输入有效的网址(http/https)或图标文件名(.ico/.png 等)'))
      },
      trigger: 'blur',
    },
  ],
}

const logoUrl = (logo: string) => (logo?.startsWith('http') ? logo : `/api/img?url=${logo}`)

const loadEngines = async () => {
  loading.value = true
  try {
    engines.value = await fetchGetAllSearchEngines()
  } catch (error) {
    ElMessage.error(resolveError(error, '加载搜索引擎失败'))
  } finally {
    loading.value = false
  }
}

// ==================== 拖拽排序 ====================

const persistSort = async (movedId: number, targetId: number) => {
  const list = [...engines.value]
  const from = list.findIndex((item) => item.id === movedId)
  const to = list.findIndex((item) => item.id === targetId)
  if (from < 0 || to < 0) {
    return
  }
  const [moved] = list.splice(from, 1)
  list.splice(to, 0, moved)
  engines.value = list.map((item, index) => ({ ...item, sort: index + 1 }))
  try {
    await fetchUpdateSearchEnginesSort(list.map((item, index) => ({ id: item.id, sort: index + 1 })))
    clearSearchEngineCache()
    ElMessage.success('排序已更新')
  } catch (error) {
    ElMessage.error(resolveError(error, '排序更新失败'))
    await loadEngines()
  }
}

useTableSortable({
  tableRef,
  rows: engines,
  onEnd: (oldIndex, newIndex) => {
    const moved = engines.value[oldIndex]
    const target = engines.value[newIndex]
    if (!moved || !target) {
      return
    }
    persistSort(moved.id, target.id)
  },
})

// ==================== 增删改 ====================

const openAdd = () => {
  editing.value = false
  Object.assign(form, createEmptyForm())
  showDialog.value = true
}

const openEdit = (row: SearchEngine) => {
  editing.value = true
  Object.assign(form, createEmptyForm(), row)
  showDialog.value = true
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) {
    return
  }
  requestLoading.value = true
  try {
    if (editing.value) {
      await fetchUpdateSearchEngine({ ...form })
      ElMessage.success('更新成功!')
    } else {
      await fetchAddSearchEngine({ ...form })
      ElMessage.success('添加成功!')
    }
    clearSearchEngineCache()
    showDialog.value = false
    await loadEngines()
  } catch (error) {
    ElMessage.warning(resolveError(error, '保存失败'))
  } finally {
    requestLoading.value = false
  }
}

const handleToggleEnabled = async (row: SearchEngine, enabled: boolean) => {
  try {
    await fetchUpdateSearchEngine({ ...row, enabled })
    clearSearchEngineCache()
    ElMessage.success(enabled ? '已启用' : '已停用')
  } catch (error) {
    row.enabled = !enabled
    ElMessage.warning(resolveError(error, '操作失败'))
  }
}

const handleDelete = async (id: number) => {
  try {
    await fetchDeleteSearchEngine(id)
    clearSearchEngineCache()
    ElMessage.success('删除成功!')
  } catch (error) {
    ElMessage.warning(resolveError(error, '删除失败'))
  } finally {
    await loadEngines()
  }
}

onMounted(loadEngines)
</script>
