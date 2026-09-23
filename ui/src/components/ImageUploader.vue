<template>
  <div class="image-uploader">
    <div class="image-uploader-row">
      <el-input v-model="urlValue" :placeholder="placeholder" clearable />
      <el-upload
        :show-file-list="false"
        :accept="accept"
        :disabled="uploading"
        :http-request="handleUpload"
      >
        <el-button :loading="uploading">上传图片</el-button>
      </el-upload>
      <el-button v-if="urlValue" @click="urlValue = ''">清除</el-button>
    </div>
    <div v-if="preview && urlValue" class="image-uploader-preview">
      <img :src="urlValue" alt="预览" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { UploadRequestOptions } from 'element-plus'
import { fetchUploadImage, resolveError } from '../api'

/** 单张图片的大小上限，与后端 service.MaxUploadSize 保持一致 */
const MAX_UPLOAD_SIZE = 5 * 1024 * 1024

const props = withDefaults(
  defineProps<{
    /** 图片地址，支持直接填写外链或使用上传得到的地址 */
    modelValue: string
    placeholder?: string
    /** 文件选择框的 accept，默认放行常用图片格式 */
    accept?: string
    /** 是否展示预览图（背景图这类大图建议开启） */
    preview?: boolean
  }>(),
  {
    placeholder: '请输入图片地址',
    accept: 'image/png,image/jpeg,image/webp,image/gif,image/svg+xml,image/x-icon',
    preview: false,
  }
)

const emit = defineEmits<{ (e: 'update:modelValue', value: string): void }>()

const uploading = ref(false)

const urlValue = computed({
  get: () => props.modelValue,
  set: (value: string) => emit('update:modelValue', value),
})

/** 覆盖 el-upload 的默认上传行为，走后端接口并回填地址 */
const handleUpload = async (options: UploadRequestOptions) => {
  const file = options.file
  if (file.size > MAX_UPLOAD_SIZE) {
    ElMessage.warning(`图片大小不能超过 ${MAX_UPLOAD_SIZE / 1024 / 1024} MB`)
    return
  }
  uploading.value = true
  try {
    const url = await fetchUploadImage(file)
    urlValue.value = url
    ElMessage.success('上传成功，请点击提交保存')
  } catch (error) {
    ElMessage.warning(resolveError(error, '上传失败'))
  } finally {
    uploading.value = false
  }
}
</script>
