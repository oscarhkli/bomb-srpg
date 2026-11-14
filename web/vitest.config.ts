import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['src/test/setup.ts'],
    include: ['src/**/*.test.ts', 'eslint-rules/**/*.test.js'],
    // Phaser mocks need this
    server: {
      deps: {
        inline: ['phaser']
      }
    },
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'json-summary', 'json'],
      reportsDirectory: './coverage',
      exclude: [
        'src/main.ts',
        'src/counter.ts',
        'src/test/**',
        'src/assets/**',
        'src/types/**',
        'src/style.css',
      ]
    }
  }
})
