<template>
  <a
    class="card-box"
    :href="isToggleCard ? undefined : tool.url"
    :target="target"
    rel="noreferrer"
    @click="emit('click', tool)"
  >
    <span v-if="showNumIndex" class="card-index">{{ index + 1 }}</span>
    <div class="card-content" :class="{ 'compact-mode': compactMode }">
      <div v-if="!noImageMode" class="card-left">
        <div v-if="imageError" class="card-image-error">🖼️</div>
        <template v-else>
          <span v-if="showLoading" class="card-loading-spinner" />
          <img
            :src="imageSrc"
            :alt="tool.name"
            loading="lazy"
            :style="{ opacity: imageLoaded ? 1 : 0.1 }"
            @load="onImageLoad"
            @error="onImageError"
          />
        </template>
      </div>
      <div class="card-right">
        <div class="card-right-top">
          <span class="card-right-title" :title="tool.name">{{ tool.name }}</span>
          <span v-if="!compactMode" class="card-tag" :title="displayCatelog(tool.catelog)">
            {{ displayCatelog(tool.catelog) }}
          </span>
        </div>
        <div v-if="!compactMode" class="card-right-bottom" :title="tool.desc">{{ tool.desc }}</div>
      </div>
    </div>
  </a>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { displayCatelog, getLogoUrl } from '../utils/check'
import { getJumpTarget } from '../utils/setting'
import type { Tool } from '../types'

const props = defineProps<{
  tool: Tool
  index: number
  isSearching: boolean
  noImageMode?: boolean
  compactMode?: boolean
}>()

const emit = defineEmits<{ (e: 'click', tool: Tool): void }>()

const imageLoaded = ref(false)
const imageError = ref(false)
const showLoading = ref(true)
let timer: ReturnType<typeof setTimeout> | null = null

const isToggleCard = computed(() => props.tool.url === 'toggleJumpTarget')
const imageSrc = computed(() => (isToggleCard.value ? props.tool.logo : getLogoUrl(props.tool.logo)))
const target = computed(() => (getJumpTarget() === 'blank' ? '_blank' : '_self'))
const showNumIndex = computed(() => props.index < 10 && props.isSearching)

const resetImageState = () => {
  imageLoaded.value = false
  imageError.value = false
  showLoading.value = true
  if (timer) {
    clearTimeout(timer)
  }
  // 10 秒超时保护
  timer = setTimeout(() => {
    showLoading.value = false
  }, 10000)
}

const onImageLoad = () => {
  imageLoaded.value = true
  showLoading.value = false
}

const onImageError = () => {
  imageError.value = true
  showLoading.value = false
}

watch(imageSrc, resetImageState, { immediate: true })

onBeforeUnmount(() => {
  if (timer) {
    clearTimeout(timer)
  }
})
</script>
