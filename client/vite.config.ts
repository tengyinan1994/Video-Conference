import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    // 0.0.0.0：监听所有网卡，可用 http://10.64.3.83:5173 访问
    // 注意：命令行若写了 --host 127.0.0.1 会覆盖这里，只会显示 Local
    host: '0.0.0.0',
    port: 5173,
    strictPort: true,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8000',
        changeOrigin: true,
      },
      // Cursor 内置预览会拦直连 :7880 的 WebSocket；开发期同源代理 LiveKit 信令
      '/rtc': {
        target: 'http://127.0.0.1:7880',
        changeOrigin: true,
        ws: true,
      },
    },
  },
})
