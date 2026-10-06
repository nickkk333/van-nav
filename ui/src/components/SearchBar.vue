<template>
  <div class="search">
    <div class="search-wraper" ref="wraperRef">
      <el-input
        id="search-bar"
        ref="inputRef"
        class="search-input"
        :model-value="modelValue"
        size="large"
        type="search"
        clearable
        :placeholder="placeholder"
        @update:model-value="onInput"
        @focus="focused = true"
        @blur="focused = false"
        @keydown="onKeydown"
      />
      <div
        v-if="completion && focused"
        ref="ghostRef"
        class="search-completion"
        aria-hidden="true"
      ><span class="search-completion-typed">{{ modelValue }}</span><span class="search-completion-suffix">{{ completion }}</span></div>
      <button class="search-btn" type="button" aria-label="搜索" @click="emit('search')">
        <el-icon :size="18"><Search /></el-icon>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Search } from '@element-plus/icons-vue'
import type { InputInstance } from 'element-plus'

const props = withDefaults(defineProps<{ modelValue: string; placeholder?: string; completion?: string }>(), {
  // 默认提示；首页会换成必应壁纸的图片描述（见 HomeView）
  placeholder: '按任意键直接开始搜索',
  completion: '',
})
const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'search'): void
}>()

const inputRef = ref<InputInstance>()
const wraperRef = ref<HTMLElement>()
const ghostRef = ref<HTMLElement>()
const focused = ref(false)

const onInput = (value: string) => {
  emit('update:modelValue', value ?? '')
}

// Tab 键把未补全的域名补全，如输入 git -> 补全 hub.com（得到 github.com）
const onKeydown = (ev: KeyboardEvent) => {
  if (ev.key === 'Tab' && props.completion) {
    ev.preventDefault()
    emit('update:modelValue', (props.modelValue ?? '') + props.completion)
  }
}

/** 让 ghost 补全层与输入框文字精确对齐：拷贝输入框的位置与字体度量 */
const syncGhost = () => {
  const ghost = ghostRef.value
  const wrap = wraperRef.value
  const inputEl = inputRef.value?.$el?.querySelector('input') as HTMLInputElement | null
  if (!ghost || !inputEl || !wrap) return
  const inputRect = inputEl.getBoundingClientRect()
  const wrapRect = wrap.getBoundingClientRect()
  const cs = window.getComputedStyle(inputEl)
  ghost.style.left = `${inputRect.left - wrapRect.left}px`
  ghost.style.top = `${inputRect.top - wrapRect.top}px`
  ghost.style.width = `${inputRect.width}px`
  ghost.style.height = `${inputRect.height}px`
  ghost.style.font = cs.font
  ghost.style.letterSpacing = cs.letterSpacing
  ghost.style.textIndent = cs.textIndent
}

let raf = 0
const scheduleSync = () => {
  cancelAnimationFrame(raf)
  raf = requestAnimationFrame(syncGhost)
}
watch(() => [props.modelValue, props.completion, focused.value], scheduleSync)

/** 页面上任意按键都可以直接聚焦搜索框 */
const onGlobalKeyDown = (ev: KeyboardEvent) => {
  const reg = /[a-zA-Z0-9]|[\u4e00-\u9fa5]/
  if (ev.code === 'Enter' || reg.test(ev.key)) {
    inputRef.value?.focus()
  }
}

onMounted(() => {
  document.addEventListener('keydown', onGlobalKeyDown)
  window.addEventListener('resize', scheduleSync)
  scheduleSync()
})
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onGlobalKeyDown)
  window.removeEventListener('resize', scheduleSync)
  cancelAnimationFrame(raf)
})

defineExpose({ focus: () => inputRef.value?.focus() })
</script>
