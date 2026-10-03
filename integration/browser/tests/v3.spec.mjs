import { test, expect } from '@playwright/test'
import { initialPage, matrix, propState } from './support.mjs'

test.skip(matrix.protocol !== 3, 'v3 wire contract uses its isolated official client profile')

async function ready(page, url = '/second') {
  await page.goto(url)
  await expect(page.locator('html')).toHaveAttribute('data-ready', 'true')
}
async function visit(page, url, options = {}) {
  await page.evaluate(({ url, options }) => window.__router.visit(url, options), { url, options })
}
const metadata = async (page) => JSON.parse(await page.locator('#page-meta').textContent())

async function expectManagedTitle(page, title) {
  // Playwright 1.58's text matcher intentionally skips every element in <head>.
  // Check the actual node property and document title without losing ownership
  // or deduplication assertions.
  await expect(page.locator('head title')).toHaveCount(1)
  await expect(page.locator('head title[data-inertia]')).toHaveCount(1)
  await expect(page.locator('head title[data-inertia]')).toHaveJSProperty('textContent', title)
  await expect(page).toHaveTitle(title)
}

test('script bootstrap safely round-trips script terminators, Unicode and JSON characters', async ({ page, request }, info) => {
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  const html = await (await request.get('/v3/unsafe')).text()
  expect(html).toContain('type="application/json"')
  expect(html).toContain('data-page="app"')
  expect(html).not.toContain('<script>window.__injected = true</script>')
  expect(html).not.toMatch(/<div[^>]+data-page=/)
  await ready(page, '/v3/unsafe')
  await expectManagedTitle(page, 'Safe bootstrap')
  expect((await propState(page)).unsafe).toBe('</script><script>window.__injected = true</script>&"雪\u2028\u2029')
  expect(await page.evaluate(() => window.__injected)).toBeUndefined()
  await expect(page.locator('html')).toHaveAttribute('data-mode', info.project.metadata.ssr ? 'hydrated' : 'mounted')
  expect(errors).toEqual([])
})

test('409 X-Inertia-Redirect makes a fresh GET and preserves fragments without document reload', async ({ page }) => {
  await ready(page, '/form')
  const marker = await page.evaluate(() => { window.__documentMarker = Math.random(); return window.__documentMarker })
  const responsePromise = page.waitForResponse(response => response.url().includes('/v3/redirect'))
  await visit(page, '/v3/redirect')
  const response = await responsePromise
  expect(response.status()).toBe(409)
  expect(response.headers()['x-inertia-redirect']).toBe('/second#destination')
  expect(response.headers()['x-inertia']).toBeUndefined()
  await expect(page).toHaveURL(/\/second#destination$/)
  await expect(page.locator('#title')).toHaveText('Second')
  await expectManagedTitle(page, 'Second')
  expect(await page.evaluate(() => window.__documentMarker)).toBe(marker)
  await page.goBack()
  await expect(page).toHaveURL(/\/form$/)
  await expect(page.locator('#title')).toHaveText('Form')
})

test('newer navigation cancels an older visit and cancelAll leaves the current page intact', async ({ page }) => {
  await ready(page)
  let release
  const gate = new Promise(resolve => { release = resolve })
  await page.route('**/v3/slow', async route => {
    await gate
    await route.abort('aborted')
  })
  try {
    const older = page.waitForRequest(request => request.url().includes('/v3/slow'))
    await page.evaluate(() => window.__router.visit('/v3/slow', { onCancel: () => { window.__cancelled++ } }))
    await older
    await visit(page, '/form')
    await expect(page.locator('#title')).toHaveText('Form')
    await expect.poll(() => page.evaluate(() => window.__cancelled)).toBe(1)
    const pending = page.waitForRequest(request => request.url().includes('/v3/slow'))
    await page.evaluate(() => window.__router.visit('/v3/slow', { async: true, onCancel: () => { window.__cancelled++ } }))
    await pending
    await page.evaluate(() => window.__router.cancelAll())
    await expect.poll(() => page.evaluate(() => window.__cancelled)).toBe(2)
    // The public cancellation callbacks run before the intercepted requests
    // are released, proving cancellation came from the router operations.
    await visit(page, '/second')
    await expect(page.locator('#title')).toHaveText('Second')
  } finally {
    release()
    await page.unrouteAll({ behavior: 'wait' })
  }
})

test('error pages retain HTTP status, shared props and visit exception callbacks', async ({ page, request }) => {
  for (const status of [404, 500]) {
    const response = await request.get(`/v3/status?code=${status}`, { headers: { 'X-Inertia': 'true', 'X-Inertia-Version': 'browser-v1' } })
    expect(response.status()).toBe(status)
    expect(response.headers()['x-inertia']).toBe('true')
    expect((await response.json()).props).toMatchObject({ status, sharedMarker: 'shared-value' })
    await ready(page)
    await page.evaluate(status => window.__router.visit(`/v3/status?code=${status}`, {
      onHttpException: response => { window.__exceptionStatus = response.status },
    }), status)
    await expect(page.locator('#title')).toHaveText('Exception page')
    expect(await page.evaluate(() => window.__exceptionStatus)).toBe(status)
    expect(await propState(page)).toMatchObject({ status, sharedMarker: 'shared-value' })
  }
})

test('recursive wrappers announce dotted deferred metadata, merge and rescue then recover', async ({ page }) => {
  const rescued = page.waitForResponse(response => response.request().headers()['x-inertia-partial-data'] === 'panel.rescued')
  await ready(page, '/v3/recursive')
  const initial = await initialPage(page)
  expect(initial.deferredProps.nested).toEqual(['panel.heavy'])
  expect(initial.deferredProps.rescue).toEqual(['panel.rescued'])
  expect(initial.onceProps['nested-cache']).toEqual({ prop: 'panel.cached', expiresAt: null })
  expect(initial.props.panel).not.toHaveProperty('heavy')
  expect(initial.props.panel).not.toHaveProperty('optional')
  expect((await (await rescued).json()).rescuedProps).toEqual(['panel.rescued'])
  await expect.poll(() => metadata(page)).toMatchObject({ rescuedProps: ['panel.rescued'] })
  await expect.poll(() => propState(page)).toMatchObject({ panel: { heavy: 'heavy-1', cached: 'cache-1', items: [1] } })
  await page.evaluate(() => window.__router.get('/v3/recursive?step=2', {}, { only: ['panel.items'], preserveState: true }))
  await expect.poll(() => propState(page)).toMatchObject({ panel: { items: [1, 2], label: 'panel-1', cached: 'cache-1', heavy: 'heavy-1' } })
  await page.evaluate(() => window.__router.get('/v3/recursive?recover=1', {}, { only: ['panel.rescued', 'panel.optional'], preserveState: true }))
  await expect.poll(() => propState(page)).toMatchObject({ panel: { rescued: 'recovered', optional: 'optional-1', items: [1, 2] } })
  await expect.poll(() => metadata(page)).toMatchObject({ rescuedProps: [] })
})

test('combined only and except intersect, and recursive once is reused on full visits', async ({ page }) => {
  await ready(page, '/v3/recursive?recover=1')
  await expect.poll(() => propState(page)).toMatchObject({ panel: { heavy: 'heavy-1', rescued: 'recovered' } })
  const full = page.waitForResponse(response => response.url().includes('step=2') && !response.request().headers()['x-inertia-partial-data'])
  await visit(page, '/v3/recursive?step=2&recover=1')
  const data = await (await full).json()
  expect(data.props.panel).not.toHaveProperty('cached')
  await expect.poll(() => propState(page)).toMatchObject({ panel: { label: 'panel-2', cached: 'cache-1', heavy: 'heavy-2' } })
  const partial = page.waitForResponse(response => response.request().headers()['x-inertia-partial-except'] === 'panel.label')
  await page.evaluate(() => window.__router.reload({ data: { step: 3 }, only: ['panel.label', 'panel.cached'], except: ['panel.label'] }))
  const selected = await (await partial).json()
  expect(selected.props.panel).toEqual({ cached: 'cache-3' })
  await expect.poll(() => propState(page)).toMatchObject({ panel: { label: 'panel-2', cached: 'cache-3' } })
})

test('foreground asset conflicts reload while background conflicts only notify', async ({ page }) => {
  await ready(page)
  await page.evaluate(() => new Promise(resolve => window.__router.replace({ version: 'obsolete', onFinish: () => resolve() })))
  await expect.poll(() => metadata(page)).toMatchObject({ version: 'obsolete' })
  const marker = await page.evaluate(() => { window.__documentMarker = Math.random(); return window.__documentMarker })
  const background = page.waitForResponse(response => response.status() === 409)
  await page.evaluate(() => window.__router.reload())
  const response = await background
  expect(response.headers()['x-inertia-version']).toBe('browser-v1')
  expect(response.headers()['x-inertia-location']).toMatch(/\/second$/)
  await expect.poll(() => page.evaluate(() => window.__locations.length)).toBe(1)
  expect(await page.evaluate(() => window.__documentMarker)).toBe(marker)
  expect(await page.evaluate(() => window.__locations[0].versionChange)).toBe(true)
  await expect(page.locator('#title')).toHaveText('Second')
  await visit(page, '/form')
  await expect(page.locator('#title')).toHaveText('Form')
  expect(await page.evaluate(() => window.__documentMarker)).toBeUndefined()
})

test('history encryption hides page contents and encrypted Back restoration works', async ({ page }) => {
  await ready(page, '/v3/encrypted')
  await expect(page.locator('#title')).toHaveText('Encrypted secret')
  const marker = await page.evaluate(() => { window.__historyMarker = Math.random(); return window.__historyMarker })
  await expect.poll(() => page.evaluate(() => history.state?.page instanceof ArrayBuffer)).toBe(true)
  expect(await page.evaluate(() => JSON.stringify(history.state))).not.toContain('Encrypted secret')
  await visit(page, '/second')
  await expect(page.locator('#title')).toHaveText('Second')
  await page.goBack()
  await expect(page.locator('#title')).toHaveText('Encrypted secret')
  expect(await page.evaluate(() => window.__historyMarker)).toBe(marker)
  await visit(page, '/v3/clear-history')
  await expect(page.locator('#title')).toHaveText('History cleared')
  expect((await metadata(page)).clearHistory).toBe(true)
})

test('SSR failure is reported then boots CSR, and per-request SSR suppression is isolated', async ({ page, request }, info) => {
  test.skip(!info.project.metadata.ssr, 'requires a configured SSR renderer')
  const before = (await (await request.get('/v3/ssr-reports')).json()).count
  await ready(page, '/v3/ssr-failure')
  await expect(page.locator('#title')).toHaveText('SSR fallback')
  await expect(page.locator('html')).toHaveAttribute('data-mode', 'mounted')
  expect((await (await request.get('/v3/ssr-reports')).json()).count).toBeGreaterThan(before)
  await ready(page, '/v3/no-ssr')
  await expect(page.locator('html')).toHaveAttribute('data-mode', 'mounted')
  await ready(page, '/second')
  await expect(page.locator('html')).toHaveAttribute('data-mode', 'hydrated')
})

test('opt-in large integers survive official parsing, rendering and hydration exactly', async ({ page }) => {
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  page.on('console', message => { if (/hydration.*mismatch|hydration failed/i.test(message.text())) errors.push(message.text()) })
  await ready(page, '/v3/bigint')
  await expect(page.locator('#bigint-type')).toHaveText('bigint')
  expect((await propState(page)).large.id).toBe('9007199254740993n')
  const initial = await initialPage(page)
  expect(initial.preserveBigIntegers).toBe(true)
  expect(initial.props.large.id).toEqual({ $bigint: '9007199254740993' })
  expect(errors).toEqual([])
})

test('shared metadata supplies an instant visit before its server response', async ({ page }) => {
  await ready(page)
  expect((await initialPage(page)).sharedProps).toContain('sharedMarker')
  let release
  const gate = new Promise(resolve => { release = resolve })
  await page.route('**/v3/slow', async route => {
    await gate
    await route.continue()
  })
  const response = page.waitForResponse(result => result.url().endsWith('/v3/slow'))
  try {
    await page.evaluate(() => window.__router.visit('/v3/slow', { component: 'Slow' }))
    await expect.poll(() => propState(page)).toEqual({ sharedMarker: 'shared-value', errors: {} })
  } finally {
    release()
  }
  await response
  await expect(page.locator('#title')).toHaveText('Slow')
})

test('all validation messages survive a named error bag and redirect session round trip', async ({ page }) => {
  await ready(page, '/form')
  await page.evaluate(() => window.__router.post('/v3/array-errors', {}, {
    errorBag: 'signup', onError: errors => { window.__bagErrors = errors },
  }))
  await expect.poll(() => page.evaluate(() => window.__bagErrors)).toEqual({ name: ['Required', 'Must be unique'] })
  expect((await propState(page)).errors).toEqual({ signup: { name: ['Required', 'Must be unique'] } })
  await page.reload()
  await expect.poll(() => propState(page)).toMatchObject({ errors: {} })
})

test('custom scroll data paths append, prepend and reset with real client merging', async ({ page }) => {
  await ready(page, '/v3/scroll')
  const append = page.waitForResponse(response => response.request().headers()['x-inertia-partial-data'] === 'results')
  await page.evaluate(() => window.__router.get('/v3/scroll?step=2', {}, { only: ['results'], preserveState: true }))
  const appended = await (await append).json()
  expect(appended.mergeProps).toEqual(['results.records'])
  expect(appended.scrollProps.results).toMatchObject({ pageName: 'step', currentPage: 2 })
  await expect.poll(() => propState(page)).toMatchObject({ results: { records: [1, 2] } })
  await page.evaluate(() => window.__router.get('/v3/scroll?step=3', {}, {
    only: ['results'], preserveState: true, headers: { 'X-Inertia-Infinite-Scroll-Merge-Intent': 'prepend' },
  }))
  await expect.poll(() => propState(page)).toMatchObject({ results: { records: [3, 1, 2] } })
  const reset = page.waitForResponse(response => response.request().headers()['x-inertia-reset'] === 'results')
  await page.evaluate(() => window.__router.get('/v3/scroll?step=4', {}, { only: ['results'], reset: ['results'], preserveState: true }))
  const replacement = await (await reset).json()
  expect(replacement.mergeProps).toBeUndefined()
  expect(replacement.scrollProps.results.reset).toBe(true)
  await expect.poll(() => propState(page)).toMatchObject({ results: { records: [4] } })
})

test('explicit preserveFragment is emitted and honored by the real client', async ({ page }) => {
  await ready(page)
  const pending = page.waitForResponse(response => response.url().includes('/v3/preserved') && response.headers()['x-inertia'] === 'true')
  await visit(page, '/v3/fragment#destination')
  expect((await (await pending).json()).preserveFragment).toBe(true)
  await expect(page).toHaveURL(/\/v3\/preserved#destination$/)
  await expect(page.locator('#title')).toHaveText('Exception page')
})

test('array wrappers omit unresolved indexes and Deferred observes dotted data correctly', async ({ page }) => {
  await ready(page, '/v3/array')
  const initial = await initialPage(page)
  expect(initial.props.array).toEqual({ 1: 'visible-1' })
  expect(initial.deferredProps.array).toEqual(['array.0'])
  expect(initial.deferredProps['array-rescue']).toEqual(['array.3'])
  await expect(page.locator('#array-value')).toHaveText('deferred-array-value')
  await expect(page.locator('#array-fallback')).toHaveCount(0)
  await expect.poll(() => metadata(page)).toMatchObject({ rescuedProps: ['array.3'] })
  expect((await propState(page)).array).not.toHaveProperty('3')
  await page.evaluate(() => window.__router.reload({ only: ['array.1'], data: { step: 2 } }))
  await expect.poll(() => propState(page)).toMatchObject({ array: { 0: 'deferred-array-value', 1: 'visible-2' } })
  await page.evaluate(() => window.__router.reload({ only: ['array.2'] }))
  await expect.poll(() => propState(page)).toMatchObject({ array: {
    0: 'deferred-array-value', 1: 'visible-2', 2: 'optional-array-value',
  } })
  await expect(page.locator('#array-value')).toHaveText('deferred-array-value')
})
