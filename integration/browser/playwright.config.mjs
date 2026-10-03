import { defineConfig } from '@playwright/test'
const combinations = ['fiber', 'nethttp'].flatMap((adapter, index) => [false, true].map((ssr, offset) => ({
  adapter, ssr, port: 19000 + index * 2 + offset,
})))
export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: { browserName: 'chromium', trace: 'retain-on-failure' },
  projects: combinations.map(({ adapter, ssr, port }) => ({
    name: `${adapter}-${ssr ? 'ssr' : 'csr'}`,
    use: { baseURL: `http://127.0.0.1:${port}` },
    metadata: { adapter, ssr },
  })),
  webServer: [
    { command: 'node dist/ssr.mjs', url: 'http://127.0.0.1:18985/health', reuseExistingServer: false },
    ...combinations.map(({ adapter, ssr, port }) => ({
      command: `./dist/browser-server -adapter ${adapter} -port ${port} -assets dist${ssr ? ' -ssr-url http://127.0.0.1:18985/render' : ''}`,
      url: `http://127.0.0.1:${port}/health`, reuseExistingServer: false,
    })),
  ],
})
