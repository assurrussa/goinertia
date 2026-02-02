import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'public',
    emptyOutDir: false,
    rollupOptions: {
      input: 'src/app.js',
      output: {
        entryFileNames: 'app.js',
        assetFileNames: '[name].[ext]',
        chunkFileNames: '[name].js'
      }
    }
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      'vue': 'vue/dist/vue.esm-bundler.js',
    },
  },
})
