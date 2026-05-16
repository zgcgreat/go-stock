import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite';
import Components from 'unplugin-vue-components/vite';
import { TDesignResolver } from '@tdesign-vue-next/auto-import-resolver';
import path from 'path'

export default defineConfig({
  plugins: [
    vue(),
    AutoImport({
      resolvers: [TDesignResolver({
        library: 'chat'
      })],
    }),
    Components({
      resolvers: [TDesignResolver({
        library: 'chat'
      })],
    }),
  ],
  resolve: {
    alias: {
      '../wailsjs/go/main/App': path.resolve(__dirname, 'src/utils/wails-bridge.js'),
      '../../wailsjs/go/main/App': path.resolve(__dirname, 'src/utils/wails-bridge.js'),
      '../wailsjs/runtime': path.resolve(__dirname, 'src/utils/wails-bridge.js'),
      '../../wailsjs/runtime': path.resolve(__dirname, 'src/utils/wails-bridge.js'),
    }
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      }
    }
  },
  build: {
    // 减少构建内存占用
    chunkSizeWarningLimit: 1500,
    rollupOptions: {
      output: {
        // 分包策略，减少单个 chunk 大小
        manualChunks: {
          'vendor': ['vue', 'vue-router'],
          'tdesign': ['tdesign-vue-next'],
        }
      }
    }
  }
})
