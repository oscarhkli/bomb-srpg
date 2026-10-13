import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  outputDir: './test-results',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI
    ? [['github'], ['html', { open: 'never', outputFolder: 'playwright-report' }]]
    : 'list',
  use: {
    baseURL: 'http://localhost:8080',
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'firefox', use: { ...devices['Desktop Firefox'] } },
    {
      name: 'firefox-canvas',
      use: {
        ...devices['Desktop Firefox'],
        launchOptions: { firefoxUserPrefs: { 'webgl.disabled': true } },
      },
    },
    { name: 'webkit', use: { ...devices['Desktop Safari'] } },
  ],
  webServer: {
    command: './bin/srpg-web',
    cwd: '../..',
    url: 'http://localhost:8080/api/health',
    reuseExistingServer: !process.env.CI,
  },
});
