<template>
  <div class="search">
    <div class="search-wraper">
      <el-input
        id="search-bar"
        ref="inputRef"
        class="search-input"
        :model-value="modelValue"
        size="large"
        type="search"
        clearable
        placeholder="按任意键直接开始搜索"
        @update:model-value="onInput"
      />
      <button class="search-btn" type="button" aria-label="搜索" @click="emit('search')">
        <el-icon :size="18"><Search /></el-icon>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { Search } from '@element-plus/icons-vue'
import type { InputInstance } from 'element-plus'

defineProps<{ modelValue: string }>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'search'): void
}>()

const inputRef = ref<InputInstance>()

const onInput = (value: string) => {
  emit('update:modelValue', value ?? '')
}

/** 页面上任意按键都可以直接聚焦搜索框 */
const onKeyDown = (ev: KeyboardEvent) => {
  const reg = /[a-zA-Z0-9]|[\u4e00-\u9fa5]/
  if (ev.code === 'Enter' || reg.test(ev.key)) {
    inputRef.value?.focus()
  }
}

onMounted(() => document.addEventListener('keydown', onKeyDown))
onBeforeUnmount(() => document.removeEventListener('keydown', onKeyDown))

defineExpose({ focus: () => inputRef.value?.focus() })
</script>
