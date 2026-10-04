import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    // Element Plus 按需引入：模板里的 el-* 组件、v-loading 等指令、
    // 代码里的 ElMessage/ElMessageBox 等 API 都改为用到才打包；
    // 之前全量引入 index.css（363KB）+ 全量注册组件，是 element 分包 926KB 的主因
    AutoImport({
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
  ],
  server: {
    port: 2333,
    host: true,
    proxy: {
      '/api': {
        target: 'http://localhost:6412',
        changeOrigin: true,
      },
      '/manifest.json': {
        target: 'http://localhost:6412',
        changeOrigin: true,
      },
    },
  },
  build: {
    // 直接输出到项目根目录的 public 目录，供后端 go:embed 使用
    outDir: '../public',
    emptyOutDir: true,
    chunkSizeWarningLimit: 600,
    // Vite 8 底层是 Rolldown：对象形式的 manualChunks 已不再支持，
    // 改用 output.codeSplitting.groups 做分包（首屏只留 vue + 首页逻辑，
    // element-plus / pinyin-pro 字典各自独立分包，利用浏览器并行加载与长期缓存）
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            { name: 'vue', test: /node_modules[\\/](vue|vue-router|pinia)[\\/]/ },
            { name: 'element', test: /node_modules[\\/]element-plus|node_modules[\\/]@element-plus[\\/]/ },
            { name: 'pinyin', test: /node_modules[\\/]pinyin-pro[\\/]/ },
          ],
        },
      },
    },
  },
})