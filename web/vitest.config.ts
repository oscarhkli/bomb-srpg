import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['src/test/setup.ts'],
    include: ['src/**/*.test.ts'],
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
        'src/main.ts',    // Phaser bootstrap — requires real DOM + WebGL, not jsdom
        'src/counter.ts', // Vite scaffold leftover
        'src/test/**',    // test infrastructure
        'src/assets/**',  // static files
        'src/types/**',   // pure TS interfaces, no runtime logic
        'src/style.css',
      ]
    }
  }
})
