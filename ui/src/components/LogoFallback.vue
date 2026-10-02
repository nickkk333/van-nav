<template>
  <span class="logo-fallback">{{ initial }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  /** 工具/搜索引擎名称，取第一个字符作为占位图标 */
  name?: string
}>()

/** 图标加载失败时的占位字符：中文取首字、英文取首字母大写（用 Array.from 保证 emoji 等字符不被截断） */
const initial = computed(() => {
  const name = (props.name || '').trim()
  const first = Array.from(name)[0] || '?'
  return /[a-z]/.test(first) ? first.toUpperCase() : first
})
</script>
