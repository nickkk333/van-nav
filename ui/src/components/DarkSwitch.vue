<template>
  <el-tooltip :content="`${themeModeLabel(themeMode)}（点击切换）`" placement="left">
    <div class="theme-switch-box" @click="switchTheme">
      <el-icon :size="22">
        <Monitor v-if="themeMode === 'auto'" />
        <Sunny v-else-if="themeMode === 'light'" />
        <Moon v-else-if="themeMode === 'dark'" />
        <Search v-else />
      </el-icon>
    </div>
  </el-tooltip>
</template>

<script setup lang="ts">
import { Monitor, Moon, Search, Sunny } from '@element-plus/icons-vue'
import { nextThemeMode, setThemeMode, themeMode, themeModeLabel } from '../utils/theme'

const switchTheme = () => {
  setThemeMode(nextThemeMode(themeMode.value))
}
</script>
