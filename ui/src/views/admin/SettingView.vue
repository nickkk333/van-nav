<template>
  <div class="setting-page">
    <el-card shadow="never" class="admin-card">
      <template #header>修改用户信息</template>
      <el-form ref="userFormRef" v-loading="loading" :model="userForm" :rules="userRules" label-width="140px">
        <el-form-item label="用户名" prop="name">
          <el-input v-model="userForm.name" placeholder="请输入新用户名" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="userForm.password" type="password" show-password placeholder="请输入新密码" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="requestLoading" @click="handleUpdateUser">提交</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="admin-card">
      <template #header>修改网站信息</template>
      <el-form ref="settingFormRef" v-loading="loading" :model="settingForm" :rules="settingRules" label-width="140px">
        <el-form-item label="网站 logo" prop="favicon">
          <el-tooltip content="输入或上传 logo，仅支持 png 或 svg 格式" placement="top">
            <ImageUploader
              v-model="settingForm.favicon"
              placeholder="请输入网站 logo"
              accept=".png,.svg"
            />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="网站标题" prop="title">
          <el-input v-model="settingForm.title" placeholder="请输入网站标题" />
        </el-form-item>
        <el-form-item label="公信部备案" prop="govRecord">
          <el-input v-model="settingForm.govRecord" placeholder="请输入网站备案信息" />
        </el-form-item>
        <el-form-item label="首页背景图" prop="backgroundImage">
          <el-tooltip
            content="上传或填写图片地址，前台首页会作为全屏背景图；留空则不展示背景图"
            placement="top"
          >
            <ImageUploader
              v-model="settingForm.backgroundImage"
              preview
              placeholder="上传图片或输入图片地址，留空则不展示背景图"
            />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="默认跳转方式" prop="jumpTargetBlank">
          <el-tooltip content="选择点击卡片后默认的跳转方式" placement="top">
            <el-select v-model="settingForm.jumpTargetBlank" style="width: 100%">
              <el-option label="原地跳转" :value="false" />
              <el-option label="新标签页" :value="true" />
            </el-select>
          </el-tooltip>
        </el-form-item>
        <el-form-item label="logo 192x192" prop="logo192">
          <el-tooltip content="192x192 大小的 logo，用于实现可安装的 web 应用" placement="top">
            <ImageUploader v-model="settingForm.logo192" placeholder="192x192 大小的 logo 链接" />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="logo 512x512" prop="logo512">
          <el-tooltip content="512x512 大小的 logo，用于实现可安装的 web 应用" placement="top">
            <ImageUploader v-model="settingForm.logo512" placeholder="512x512 大小的 logo 链接" />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="隐藏管理员后台卡片">
          <el-tooltip content="默认展示，开启后将在前台隐藏管理员卡片" placement="top">
            <el-switch v-model="settingForm.hideAdmin" inline-prompt active-text="开" inactive-text="关" />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="隐藏 Github 按钮">
          <el-tooltip content="默认展示，开启后将在前台隐藏 Github 按钮" placement="top">
            <el-switch v-model="settingForm.hideGithub" inline-prompt active-text="开" inactive-text="关" />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="隐藏跳转方式卡片">
          <el-tooltip content="默认展示，开启后将在前台隐藏跳转方式卡片" placement="top">
            <el-switch
              v-model="settingForm.hideToggleJumpTarget"
              inline-prompt
              active-text="开"
              inactive-text="关"
            />
          </el-tooltip>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="requestLoading" @click="handleUpdateSetting">提交</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="admin-card">
      <template #header>修改网站配置</template>
      <el-form v-loading="loading" :model="siteConfigForm" label-width="140px">
        <el-form-item label="无图模式">
          <el-tooltip content="开启后前台将不展示工具 logo 等图片" placement="top">
            <el-switch v-model="siteConfigForm.noImageMode" inline-prompt active-text="开" inactive-text="关" />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="精简模式">
          <el-tooltip content="开启后卡片只显示标题和 logo，如果同时开启无图模式则只显示标题" placement="top">
            <el-switch v-model="siteConfigForm.compactMode" inline-prompt active-text="开" inactive-text="关" />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="每行显示数量">
          <el-tooltip :content="`首页每行展示的网站数量，可填 1-${MAX_CARDS_PER_ROW}，默认 ${DEFAULT_CARDS_PER_ROW}`" placement="top">
            <el-input-number
              v-model="siteConfigForm.cardsPerRow"
              :min="1"
              :max="MAX_CARDS_PER_ROW"
              :step="1"
              controls-position="right"
            />
          </el-tooltip>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="requestLoading" @click="handleUpdateSiteConfig">提交</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { fetchUpdateSetting, fetchUpdateSiteConfig, fetchUpdateUser, resolveError } from '../../api'
import { useAdminStore } from '../../stores/admin'
import ImageUploader from '../../components/ImageUploader.vue'
import { DEFAULT_CARDS_PER_ROW, MAX_CARDS_PER_ROW } from '../../utils/setting'

const defaultSettingForm = () => ({
  favicon: 'favicon.ico',
  title: 'Van Nav',
  govRecord: '',
  logo192: 'logo192.png',
  logo512: 'logo512.png',
  hideAdmin: false,
  hideGithub: false,
  hideToggleJumpTarget: false,
  jumpTargetBlank: true,
  backgroundImage: '',
})

const adminStore = useAdminStore()
const loading = computed(() => adminStore.loading)

const userFormRef = ref<FormInstance>()
const settingFormRef = ref<FormInstance>()
const requestLoading = ref(false)

const userForm = reactive({ name: '', password: '' })
const settingForm = reactive(defaultSettingForm())
const siteConfigForm = reactive({
  noImageMode: false,
  compactMode: false,
  cardsPerRow: DEFAULT_CARDS_PER_ROW,
})

const userRules: FormRules = {
  name: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

const settingRules: FormRules = {
  favicon: [{ required: true, message: '请输入网站 logo 链接', trigger: 'blur' }],
  title: [{ required: true, message: '请输入网站 title', trigger: 'blur' }],
  jumpTargetBlank: [{ required: true, message: '这是必填项', trigger: 'change' }],
  logo192: [{ required: true, message: '请输入 192x192 大小的 logo 链接', trigger: 'blur' }],
  logo512: [{ required: true, message: '请输入 512x512 大小的 logo 链接', trigger: 'blur' }],
}

const syncForms = () => {
  const store = adminStore.store
  userForm.name = store.user?.name ?? ''
  Object.assign(settingForm, defaultSettingForm(), store.setting ?? {})
  const siteConfig = store.siteConfig
  // 老数据可能没有 cardsPerRow 字段，回落到默认值
  const cardsPerRow = Number(siteConfig?.cardsPerRow)
  Object.assign(siteConfigForm, {
    noImageMode: siteConfig?.noImageMode ?? false,
    compactMode: siteConfig?.compactMode ?? false,
    cardsPerRow: cardsPerRow >= 1 ? cardsPerRow : DEFAULT_CARDS_PER_ROW,
  })
}

watch(() => adminStore.store, syncForms, { deep: true })

const reload = async () => {
  try {
    await adminStore.load(true)
    syncForms()
  } catch (error) {
    ElMessage.error(resolveError(error, '加载数据失败'))
  }
}

const handleUpdateUser = async () => {
  const valid = await userFormRef.value?.validate().catch(() => false)
  if (!valid) {
    return
  }
  requestLoading.value = true
  try {
    await fetchUpdateUser({
      id: adminStore.store.user?.id,
      name: userForm.name,
      password: userForm.password,
    })
    ElMessage.success('修改成功!')
    userForm.password = ''
  } catch (error) {
    ElMessage.warning(resolveError(error, '修改失败'))
  } finally {
    requestLoading.value = false
    await reload()
  }
}

const handleUpdateSetting = async () => {
  const valid = await settingFormRef.value?.validate().catch(() => false)
  if (!valid) {
    return
  }
  requestLoading.value = true
  try {
    await fetchUpdateSetting({ ...settingForm })
    ElMessage.success('修改成功!')
  } catch (error) {
    ElMessage.warning(resolveError(error, '修改失败'))
  } finally {
    requestLoading.value = false
    await reload()
  }
}

const handleUpdateSiteConfig = async () => {
  requestLoading.value = true
  try {
    await fetchUpdateSiteConfig({ ...siteConfigForm })
    ElMessage.success('修改成功!')
  } catch (error) {
    ElMessage.warning(resolveError(error, '修改失败'))
  } finally {
    requestLoading.value = false
    await reload()
  }
}

onMounted(reload)
</script>
