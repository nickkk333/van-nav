<template>
  <el-config-provider :locale="zhCn">
    <router-view />
  </el-config-provider>
</template>

<script setup lang="ts">
import zhCn from 'element-plus/es/locale/lang/zh-cn'
// 根组件：只负责渲染路由 + 注入 Element Plus 中文语言包
// （按需引入下 app.use(ElementPlus) 已删除，语言包改由这里的 ConfigProvider 生效）
</script>
