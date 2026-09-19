<template>
  <el-card shadow="never" class="admin-card">
    <template #header>
      <div class="card-header">
        <div class="card-header-left">
          <span class="card-header-title">{{ `当前共 ${catelogs.length} 条` }}</span>
        </div>
        <div class="card-header-right">
          <el-button type="primary" :icon="Plus" @click="openAdd">添加</el-button>
          <el-button :icon="Refresh" @click="reload">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table v-loading="loading" :data="catelogs" row-key="id">
      <el-table-column prop="id" label="序号" width="90" />
      <el-table-column prop="name" label="名称" min-width="180" />
      <el-table-column label="排序" width="140">
        <template #header>
          <span class="column-with-tip">
            排序
            <el-tooltip content="升序，按数字从小到大排序" placement="top">
              <el-icon><QuestionFilled /></el-icon>
            </el-tooltip>
          </span>
        </template>
        <template #default="{ row }">{{ row.sort }}</template>
      </el-table-column>
      <el-table-column label="隐藏" width="140">
        <template #header>
          <span class="column-with-tip">
            隐藏
            <el-tooltip content="开启后只有登录后才会展示该分类下的工具" placement="top">
              <el-icon><QuestionFilled /></el-icon>
            </el-tooltip>
          </span>
        </template>
        <template #default="{ row }">{{ row.hide ? '是' : '否' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">修改</el-button>
          <el-popconfirm :title="`确定要删除分类 ${row.name} 吗？`" @confirm="handleDelete(row.id)">
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
        <el-tooltip content="升序，按数字从小到大排序" placement="top">
          <el-input-number v-model="addForm.sort" :min="0" />
        </el-tooltip>
      </el-form-item>
      <el-form-item label="隐藏">
        <el-tooltip content="开启后只有登录后才会展示该分类" placement="top">
          <el-switch v-model="addForm.hide" inline-prompt active-text="开" inactive-text="关" />
        </el-tooltip>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="showAdd = false">取消</el-button>
      <el-button type="primary" :loading="requestLoading" @click="handleCreate">确定</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="showEdit" title="修改分类" width="480px" destroy-on-close>
    <el-form ref="editFormRef" :model="editForm" :rules="rules" label-width="80px">
      <el-form-item label="序号">
        <el-input v-model="editForm.id" disabled />
      </el-form-item>
      <el-form-item label="名称" prop="name">
        <el-input v-model="editForm.name" placeholder="请输入分类名称" />
      </el-form-item>
      <el-form-item label="排序" prop="sort">
        <el-input-number v-model="editForm.sort" :min="0" />
      </el-form-item>
      <el-form-item label="隐藏">
        <el-switch v-model="editForm.hide" inline-prompt active-text="开" inactive-text="关" />
      </el-form-item>
</el-form>
    <template #footer>
      <el-button @click="showEdit = false">取消</el-button>
      <el-button type="primary" :loading="requestLoading" @click="handleUpdate">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, QuestionFilled, Refresh } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import { fetchAddCateLog, fetchDeleteCatelog, fetchUpdateCateLog, resolveError } from '../../api'
import { useAdminStore } from '../../stores/admin'
import type { Catelog } from '../../types'

interface CatelogForm {
  id?: number
  name: string
  sort: number
  hide: boolean
}

const createEmptyForm = (): CatelogForm => ({ name: '', sort: 1, hide: false })

const adminStore = useAdminStore()
const loading = computed(() => adminStore.loading)
const catelogs = computed<Catelog[]>(() => adminStore.store.catelogs ?? [])

const addFormRef = ref<FormInstance>()
const editFormRef = ref<FormInstance>()
const addForm = reactive<CatelogForm>(createEmptyForm())
const editForm = reactive<CatelogForm>(createEmptyForm())
const showAdd = ref(false)
const showEdit = ref(false)
const requestLoading = ref(false)

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

const openAdd = () => {
  Object.assign(addForm, createEmptyForm())
  showAdd.value = true
}

const openEdit = (row: Catelog) => {
  Object.assign(editForm, createEmptyForm(), row)
  showEdit.value = true
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

const handleUpdate = async () => {
  const valid = await editFormRef.value?.validate().catch(() => false)
  if (!valid) {
    return
  }
  requestLoading.value = true
  try {
    await fetchUpdateCateLog({ ...editForm })
    ElMessage.success('更新成功!')
    showEdit.value = false
  } catch (error) {
    ElMessage.warning(resolveError(error, '更新失败'))
  } finally {
    requestLoading.value = false
    await reload()
  }
}

const handleDelete = async (id: number) => {
  try {
    await fetchDeleteCatelog(id)
    ElMessage.success('删除分类成功!')
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
