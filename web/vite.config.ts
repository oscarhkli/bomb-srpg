import { defineConfig } from 'vite'

export default defineConfig({
  // Phaser 4 uses eval() for shaders - breaks HMR if pre-bundled
  optimizeDeps: {
    exclude: ['phaser']
  },
  // Phaser 4 targets modern browsers, ESNext output
  build: {
    target: 'esnext'
  },
  // Dev server proxy to Go backend (avoids CORS)
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  },
  // Path aliases for clean imports
  resolve: {
    alias: {
      '@': '/src',
      '@engine': '/src/engine',
      '@rendering': '/src/rendering',
      '@input': '/src/input',
      '@ui': '/src/ui',
      '@types': '/src/types',
      '@scenes': '/src/scenes'
    }
  }
})
