<template>
  <el-tooltip :content="`主题：${themeModeLabel(mode)}（点击切换）`" placement="left">
    <div class="theme-switch-box" @click="switchTheme">
      <el-icon :size="22">
        <Monitor v-if="mode === 'auto'" />
        <Sunny v-else-if="mode === 'light'" />
        <Moon v-else />
      </el-icon>
    </div>
  </el-tooltip>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Monitor, Moon, Sunny } from '@element-plus/icons-vue'
import { getThemeMode, nextThemeMode, setThemeMode, themeModeLabel } from '../utils/theme'
import type { ThemeMode } from '../utils/theme'

const mode = ref<ThemeMode>(getThemeMode())

const switchTheme = () => {
  mode.value = nextThemeMode(mode.value)
  setThemeMode(mode.value)
}
</script>