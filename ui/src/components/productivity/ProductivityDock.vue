<template>
  <div v-if="showMemo">
    <div class="pv-dock">
      <el-tooltip v-if="showMemo" content="便签" placement="left"><button type="button" class="pv-dock-btn" aria-label="便签" @click="openMemo"><el-icon :size="20"><EditPen /></el-icon></button></el-tooltip>
    </div>
    <!-- 便签：独立浮动窗口，可拖动移动、贴边吸附成图标、点击展开 -->
    <MemoWindow v-if="showMemo" ref="memoWinRef" />
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
import { EditPen } from '@element-plus/icons-vue'
import MemoWindow from './MemoWindow.vue'
// 首页默认显示便签，后台设置里可以关掉（开启「隐藏」后这里对应的入口不渲染）
const props = defineProps<{ showMemo?: boolean }>()
const showMemo = computed(() => props.showMemo !== false)
const memoWinRef = ref<InstanceType<typeof MemoWindow> | null>(null)
// 便签使用独立浮动窗口：点击入口切换展开 / 关闭
const openMemo = () => memoWinRef.value?.toggle()
</script>
