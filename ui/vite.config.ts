import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
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
    chunkSizeWarningLimit: 2000,
  },
})