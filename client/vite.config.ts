import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import basicSsl from '@vitejs/plugin-basic-ssl'
import { fileURLToPath, URL } from 'node:url'

// `tauri dev` 会注入 TAURI_ENV_*；壳内用 http://localhost（安全上下文），勿开自签 https
const isTauri = !!process.env.TAURI_ENV_PLATFORM
const host = process.env.TAURI_DEV_HOST

export default defineConfig({
  plugins: [
    vue(),
    // 仅纯网页开发：内网 IP 必须 https 才有摄像头/麦/投屏
    ...(!isTauri ? [basicSsl()] : []),
  ],
  clearScreen: false,
  envPrefix: ['VITE_', 'TAURI_ENV_*'],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    // 0.0.0.0：网页可用内网 IP；Tauri 仍连 localhost:5173
    host: host || (isTauri ? 'localhost' : '0.0.0.0'),
    port: 5173,
    strictPort: true,
    hmr: host
      ? {
          protocol: 'ws',
          host,
          port: 1421,
        }
      : undefined,
    watch: {
      ignored: ['**/src-tauri/**'],
    },
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
  build: isTauri
    ? {
        // Tauri：Windows 用 Chromium，macOS/Linux 用 WebKit
        target: process.env.TAURI_ENV_PLATFORM === 'windows' ? 'chrome105' : 'safari13',
        minify: !process.env.TAURI_ENV_DEBUG,
        sourcemap: !!process.env.TAURI_ENV_DEBUG,
      }
    : {},
})
