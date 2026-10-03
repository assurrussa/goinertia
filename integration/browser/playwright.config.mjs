import { defineConfig } from '@playwright/test'
import { readFileSync } from 'node:fs'
const matrix = JSON.parse(readFileSync(new URL('./dist/matrix.json', import.meta.url), 'utf8'))
const combinations = ['fiber', 'nethttp'].flatMap((adapter, index) => [false, true].map((ssr, offset) => ({
  adapter, ssr, port: 19000 + index * 2 + offset,
})))
export default defineConfig({
  testDir: './tests',
  testIgnore: matrix.protocol === 2 ? '**/v3.spec.mjs' : undefined,
  outputDir: `test-results/${matrix.version}-${matrix.framework}`,
  timeout: 30_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list'], ['html', { open: 'never', outputFolder: `playwright-report/${matrix.version}-${matrix.framework}` }]],
  use: { browserName: 'chromium', trace: 'retain-on-failure',
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE
      ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE } : {},
  },
  projects: combinations.map(({ adapter, ssr, port }) => ({
    name: `${adapter}-${ssr ? 'ssr' : 'csr'}`,
    use: { baseURL: `http://127.0.0.1:${port}` },
    metadata: { adapter, ssr, ...matrix },
  })),
  webServer: [
    { command: 'node dist/ssr.mjs', url: 'http://127.0.0.1:18985/health', reuseExistingServer: false },
    ...combinations.map(({ adapter, ssr, port }) => ({
      command: `./dist/browser-server -adapter ${adapter} -port ${port} -assets dist -protocol ${matrix.protocol}${ssr ? ' -ssr-url http://127.0.0.1:18985/render' : ''}`,
      url: `http://127.0.0.1:${port}/health`, reuseExistingServer: false,
    })),
  ],
})
