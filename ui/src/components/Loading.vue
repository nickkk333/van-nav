<template>
  <div class="loading-wrapper">
    <el-icon class="is-loading" :size="32">
      <Loading />
    </el-icon>
  </div>
</template>

<script setup lang="ts">
import { Loading } from '@element-plus/icons-vue'
</script>