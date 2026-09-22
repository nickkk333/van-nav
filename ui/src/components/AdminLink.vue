<template>
  <el-tooltip content="管理后台" placement="left">
    <router-link class="admin-link-box" to="/admin" aria-label="管理后台">
      <el-icon :size="24">
        <Setting />
      </el-icon>
    </router-link>
  </el-tooltip>
</template>

<script setup lang="ts">
import { Setting } from '@element-plus/icons-vue'
// 右下角管理后台入口（位于 Github 按钮下方）
</script>
