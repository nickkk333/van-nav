<template>
  <el-tooltip
    :content="`${homeModeLabel(homeMode)}模式（点击切换到${homeModeLabel(nextHomeMode(homeMode))}）`"
    placement="left"
  >
    <div class="theme-switch-box" @click="switchMode">
      <el-icon :size="22">
        <Search v-if="homeMode === 'search'" />
        <Grid v-else />
      </el-icon>
    </div>
  </el-tooltip>
</template>

<script setup lang="ts">
import { Grid, Search } from '@element-plus/icons-vue'
import { homeMode, homeModeLabel, nextHomeMode, setHomeMode } from '../utils/theme'

/** 右下角按钮只切换首页显示模式；自动 / 浅色 / 深色配色在管理页「系统配置」里设置 */
const switchMode = () => setHomeMode(nextHomeMode(homeMode.value))
</script>
