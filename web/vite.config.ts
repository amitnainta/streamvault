import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: '../web/dist',
    emptyOutDir: true,
    // No external chunks — everything bundled for embedding
    rollupOptions: {
      output: {
        manualChunks: undefined,
      },
    },
  },
  server: {
    port: 5174,
    strictPort: true, // fail instead of auto-incrementing to a different port
    proxy: {
      '/api': 'http://localhost:8096',
      '/stream': 'http://localhost:8096',
      '/direct': 'http://localhost:8096',
      '/artwork': 'http://localhost:8096',
      '/ws': { target: 'ws://localhost:8096', ws: true },
    },
  },
})
