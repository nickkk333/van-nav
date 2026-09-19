<template>
  <div class="tag-selector span-3">
    <div class="tag-selector-wrapper">
      <span
        v-for="tag in tags"
        :key="`${tag}-select-tag`"
        class="select-tag"
        :class="{ 'select-tag-active': currTag === tag }"
        @click="emit('change', tag)"
      >
        {{ displayCatelog(tag) }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { displayCatelog } from '../utils/check'

defineProps<{
  tags: string[]
  currTag: string
}>()

const emit = defineEmits<{ (e: 'change', tag: string): void }>()
</script>
