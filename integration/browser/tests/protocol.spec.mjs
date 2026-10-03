import { test, expect } from '@playwright/test'

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    window.__bootID = Math.random()
    const capture = new MutationObserver(() => {
      const heading = document.querySelector('#title')
      if (heading && !window.__firstHeading) window.__firstHeading = heading
    })
    capture.observe(document, { childList: true, subtree: true })
  })
})

test('initial HTML bootstraps and real SSR hydrates without replacing markup', async ({ page, request }, testInfo) => {
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  page.on('console', message => { if (message.type() === 'error' || /hydration.*mismatch/i.test(message.text())) errors.push(message.text()) })
  const html = await (await request.get('/')).text()
  if (testInfo.project.metadata.ssr) expect(html).toContain('<h1 id="title">Home</h1>')
  else expect(html).not.toContain('<h1 id="title">Home</h1>')
  await page.goto('/')
  await expect(page.locator('html')).toHaveAttribute('data-ready', 'true')
  await expect(page.locator('#title')).toHaveText('Home')
  await expect(page.locator('#adapter')).toHaveText(testInfo.project.metadata.adapter)
  await expect(page.locator('html')).toHaveAttribute('data-mode', testInfo.project.metadata.ssr ? 'hydrated' : 'mounted')
  if (testInfo.project.metadata.ssr) {
    expect(await page.evaluate(() => window.__firstHeading === document.querySelector('#title'))).toBe(true)
  }
  await expect(page.locator('#heavy')).toHaveText('deferred-ready')
  expect(errors).toEqual([])
})

test('client transitions preserve the document and Back/Forward restore pages', async ({ page }) => {
  await page.goto('/')
  await expect(page.locator('#heavy')).toHaveText('deferred-ready')
  const boot = await page.evaluate(() => window.__bootID)
  await page.locator('#second').click()
  await expect(page.locator('#title')).toHaveText('Second')
  await expect(page).toHaveURL(/\/second$/)
  expect(await page.evaluate(() => window.__bootID)).toBe(boot)
  await page.goBack()
  await expect(page.locator('#title')).toHaveText('Home')
  await expect(page.locator('#heavy')).toHaveText('deferred-ready')
  await page.goForward()
  await expect(page.locator('#title')).toHaveText('Second')
  expect(await page.evaluate(() => window.__bootID)).toBe(boot)
})

test('deferred requests resolve once and partial merging/reset use real client logic', async ({ page }) => {
  const deferred = []
  page.on('request', request => {
    if (request.headers()['x-inertia-partial-data'] === 'heavy') deferred.push(request.url())
  })
  await page.goto('/')
  await expect(page.locator('#heavy')).toHaveText('deferred-ready')
  expect(deferred).toHaveLength(1)
  await expect(page.locator('#items')).toHaveText('[1]')
  await page.locator('#more').click()
  await expect(page.locator('#items')).toHaveText('[1,2]')
  await expect(page.locator('#heavy')).toHaveText('deferred-ready')
  await page.locator('#reset').click()
  await expect(page.locator('#items')).toHaveText('[3]')
  expect(deferred).toHaveLength(1)
})

test('real useForm handles validation, redirects, native callbacks and legacy flash', async ({ page }) => {
  await page.goto('/form')
  await page.locator('#submit').click()
  await expect(page.locator('#error')).toHaveText('Required')
  await page.getByLabel('Name', { exact: true }).fill('Browser user')
  await page.locator('#submit').click()
  await expect(page.locator('#error')).toHaveText('')
  await expect(page.locator('#legacy-flash')).toHaveText('Saved')
  await expect(page.locator('#native-flash')).toHaveText('Native saved')
  expect(await page.evaluate(() => window.__onFlash)).toEqual({ message: 'Native saved' })
  expect(await page.evaluate(() => window.__flashEvents.filter(flash => flash.message === 'Native saved').length)).toBe(1)
  expect(await page.evaluate(() => JSON.stringify(history.state))).not.toContain('Native saved')
  expect(await page.evaluate(() => JSON.stringify(history.state))).toContain('Saved')
  await page.locator('#second').click()
  await expect(page.locator('#title')).toHaveText('Second')
  await page.goBack()
  await expect(page.locator('#title')).toHaveText('Form')
  await expect(page.locator('#legacy-flash')).toHaveText('Saved')
  await expect(page.locator('#native-flash')).toHaveText('')
  expect(await page.evaluate(() => window.__flashEvents.filter(flash => flash.message === 'Native saved').length)).toBe(1)
  await page.reload()
  await expect(page.locator('#legacy-flash')).toHaveText('')
  await expect(page.locator('#native-flash')).toHaveText('')
})

test('direct native flash fires onFlash and is not resurrected by history', async ({ page }) => {
  await page.goto('/second')
  await page.locator('#native').click()
  await expect(page.locator('#native-flash')).toHaveText('Native saved')
  expect(await page.evaluate(() => window.__onFlash)).toEqual({ message: 'Native saved' })
  await page.locator('#second').click()
  await expect(page.locator('#title')).toHaveText('Second')
  await page.goBack()
  await expect(page.locator('#native-flash')).toHaveText('')
})

test('initial native flash survives SSR/CSR bootstrap but not Back restoration', async ({ page }) => {
  await page.goto('/native')
  await expect(page.locator('#native-flash')).toHaveText('Native saved')
  await expect(page.locator('html')).toHaveAttribute('data-ready', 'true')
  expect(await page.evaluate(() => JSON.stringify(history.state))).not.toContain('Native saved')
  await page.locator('#second').click()
  await expect(page.locator('#title')).toHaveText('Second')
  await page.goBack()
  await expect(page.locator('#native-flash')).toHaveText('')
})
