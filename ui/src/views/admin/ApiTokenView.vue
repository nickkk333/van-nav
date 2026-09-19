<template>
  <el-card shadow="never" class="admin-card">
    <template #header>
      <div class="card-header">
        <div class="card-header-left">
          <span class="card-header-title">{{ `当前共 ${tokens.length} 条` }}</span>
        </div>
        <div class="card-header-right">
          <el-button type="primary" :icon="Plus" @click="openAdd">添加</el-button>
          <el-button :icon="Refresh" @click="reload">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table v-loading="loading" :data="tokens" row-key="id">
      <el-table-column prop="id" label="序号" width="90" />
      <el-table-column prop="name" label="名称" width="180" />
      <el-table-column label="值" min-width="320">
        <template #default="{ row }">
          <div class="token-value">
            <span class="token-text">{{ row.value }}</span>
            <el-tooltip content="复制" placement="top">
              <el-button link type="primary" :icon="CopyDocument" @click="copyToken(row.value)" />
            </el-tooltip>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-popconfirm :title="`确定要删除 Token ${row.name} 吗？`" @confirm="handleDelete(row.id)">
            <template #reference>
              <el-button link type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="showAdd" title="新建 Token" width="480px" destroy-on-close>
    <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" placeholder="请输入 API Token 名称" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="showAdd = false">取消</el-button>
      <el-button type="primary" :loading="requestLoading" @click="handleCreate">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { CopyDocument, Plus, Refresh } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import { fetchAddApiToken, fetchDeleteApiToken, resolveError } from '../../api'
import { useAdminStore } from '../../stores/admin'
import type { Token } from '../../types'

const adminStore = useAdminStore()
const loading = computed(() => adminStore.loading)
const tokens = computed<Token[]>(() => adminStore.store.tokens ?? [])

const formRef = ref<FormInstance>()
const showAdd = ref(false)
const requestLoading = ref(false)
const form = reactive({ name: '' })

const rules: FormRules = {
  name: [{ required: true, message: '请输入 API Token 名称', trigger: 'blur' }],
}

const reload = async () => {
  try {
    await adminStore.load(true)
  } catch (error) {
    ElMessage.error(resolveError(error, '加载数据失败'))
  }
}

const openAdd = () => {
  form.name = ''
  showAdd.value = true
}

const handleCreate = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) {
    return
  }
  requestLoading.value = true
  try {
    await fetchAddApiToken({ name: form.name })
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
    await fetchDeleteApiToken(id)
    ElMessage.success('删除成功!')
  } catch (error) {
    ElMessage.warning(resolveError(error, '删除失败'))
  } finally {
    await reload()
  }
}

const copyToken = async (value: string) => {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(value)
    } else {
      const input = document.createElement('input')
      input.value = value
      document.body.appendChild(input)
      input.select()
      document.execCommand('copy')
      document.body.removeChild(input)
    }
    ElMessage.success('已复制到剪贴板')
  } catch (error) {
    ElMessage.warning('复制失败，请手动复制')
  }
}

onMounted(() => {
  adminStore.load().catch((error) => ElMessage.error(resolveError(error, '加载数据失败')))
})
</script>