import { test, expect } from '@playwright/test'

async function state(page) {
  return JSON.parse(await page.locator('#prop-state').textContent())
}

async function openProps(page) {
  await page.goto('/props')
  await expect(page.locator('html')).toHaveAttribute('data-ready', 'true')
  await expect.poll(() => state(page)).toMatchObject({
    cached: 'cached-1', deferredOnce: 'deferredOnce-1', onceDeferred: 'onceDeferred-1',
  })
  return JSON.parse(await page.locator('#app').getAttribute('data-page'))
}

async function clickResponse(page, control, partial = '') {
  const pending = page.waitForResponse(response => {
    const headers = response.request().headers()
    return headers['x-inertia'] === 'true' && (headers['x-inertia-partial-data'] || '') === partial
  })
  await page.locator(`#${control}`).click()
  const response = await pending
  expect(response.status()).toBe(200)
  return { data: await response.json(), headers: response.request().headers() }
}

function remembered(headers) {
  return (headers['x-inertia-except-once-props'] || '').split(',').filter(Boolean)
}

test('once props are omitted on the wire and reused by the real client', async ({ page }) => {
  await openProps(page)
  const { data, headers } = await clickResponse(page, 'propsNext')
  expect(remembered(headers)).toContain('stable-cache')
  expect(data.props).not.toHaveProperty('cached')
  expect(data.onceProps['stable-cache']).toEqual({ prop: 'cached', expiresAt: null })
  await expect.poll(() => state(page)).toMatchObject({ step: 2, cached: 'cached-1' })
})

test('explicit once reloads refresh and except-only refresh is opt-in', async ({ page }) => {
  await openProps(page)
  let result = await clickResponse(page, 'propsReload', 'cached')
  expect(remembered(result.headers)).toContain('stable-cache')
  expect(result.data.props.cached).toBe('cached-2')
  await expect.poll(() => state(page)).toMatchObject({ cached: 'cached-2', refreshable: 'refreshable-1' })

  await openProps(page)
  result = await clickResponse(page, 'propsExcept')
  expect(result.headers['x-inertia-partial-except']).toBe('catalog')
  expect(remembered(result.headers)).toEqual(expect.arrayContaining(['stable-cache', 'refreshable']))
  expect(result.data.props).not.toHaveProperty('cached')
  expect(result.data.props.refreshable).toBe('refreshable-2')
  await expect.poll(() => state(page)).toMatchObject({ step: 2, cached: 'cached-1', refreshable: 'refreshable-2' })
})

test('server fresh option replaces a remembered once value on an ordinary visit', async ({ page }) => {
  await openProps(page)
  const { data, headers } = await clickResponse(page, 'propsFresh')
  expect(remembered(headers)).toContain('stable-cache')
  expect(data.props.cached).toBe('cached-2')
  await expect.poll(() => state(page)).toMatchObject({ step: 2, cached: 'cached-2' })
})

test('the real client excludes expired once keys from reuse', async ({ page }) => {
  const initial = await openProps(page)
  const expiresAt = initial.onceProps.expiring.expiresAt
  expect(typeof expiresAt).toBe('number')
  await page.clock.setFixedTime(expiresAt + 1)
  const { data, headers } = await clickResponse(page, 'propsNext')
  expect(remembered(headers)).not.toContain('expiring')
  expect(remembered(headers)).toContain('stable-cache')
  expect(data.props.expiring).toBe('expiring-2')
  await expect.poll(() => state(page)).toMatchObject({ step: 2, expiring: 'expiring-2', cached: 'cached-1' })
})

test('custom once keys reuse a value under a renamed prop on another component', async ({ page }) => {
  await openProps(page)
  const { data, headers } = await clickResponse(page, 'propsRenamed')
  expect(remembered(headers)).toContain('shared-catalog')
  expect(data.component).toBe('PropsRenamed')
  expect(data.onceProps['shared-catalog']).toEqual({ prop: 'renamedCatalog', expiresAt: null })
  expect(data.props).not.toHaveProperty('renamedCatalog')
  await expect.poll(() => state(page)).toMatchObject({ title: 'Props renamed', renamedCatalog: 'catalog-1' })
  expect(await state(page)).not.toHaveProperty('catalog')
})

test('once composes with deferred and optional props in either wrapper order', async ({ page }) => {
  const deferredRequests = []
  page.on('request', request => {
    const only = (request.headers()['x-inertia-partial-data'] || '').split(',').sort()
    if (only.join(',') === 'deferredOnce,onceDeferred') deferredRequests.push(request.url())
  })
  const initial = await openProps(page)
  expect(initial.deferredProps.composed.sort()).toEqual(['deferredOnce', 'onceDeferred'])
  for (const prop of ['deferredOnce', 'onceDeferred', 'optionalOnce', 'onceOptional']) {
    expect(initial.props).not.toHaveProperty(prop)
    expect(initial.onceProps[prop]).toEqual({ prop, expiresAt: null })
  }
  expect(deferredRequests).toHaveLength(1)
  expect(await state(page)).not.toHaveProperty('optionalOnce')
  expect(await state(page)).not.toHaveProperty('onceOptional')
  const optional = await clickResponse(page, 'propsOptional', 'optionalOnce,onceOptional')
  expect(optional.data.props).toMatchObject({ optionalOnce: 'optionalOnce-1', onceOptional: 'onceOptional-1' })
  await expect.poll(() => state(page)).toMatchObject({ optionalOnce: 'optionalOnce-1', onceOptional: 'onceOptional-1' })
  const reused = await clickResponse(page, 'propsNext')
  expect(reused.data.deferredProps).toBeUndefined()
  for (const prop of ['deferredOnce', 'onceDeferred', 'optionalOnce', 'onceOptional']) {
    expect(remembered(reused.headers)).toContain(prop)
    expect(reused.data.props).not.toHaveProperty(prop)
    expect(reused.data.onceProps[prop]).toEqual({ prop, expiresAt: null })
  }
  await expect.poll(() => state(page)).toMatchObject({
    step: 2, deferredOnce: 'deferredOnce-1', onceDeferred: 'onceDeferred-1',
    optionalOnce: 'optionalOnce-1', onceOptional: 'onceOptional-1',
  })
  expect(deferredRequests).toHaveLength(1)
})

test('nested append/prepend match identities and root reset replaces both arrays', async ({ page }) => {
  await page.goto('/nested')
  await expect(page.locator('html')).toHaveAttribute('data-ready', 'true')
  const merged = await clickResponse(page, 'nestedMore', 'feed')
  expect(merged.data.mergeProps).toEqual(['feed.data'])
  expect(merged.data.prependProps).toEqual(['feed.older'])
  expect(merged.data.matchPropsOn.sort()).toEqual(['feed.data.id', 'feed.older.id'])
  await expect.poll(() => state(page)).toMatchObject({ feed: {
    data: [{ id: 1, label: 'first updated' }, { id: 2, label: 'second' }],
    older: [{ id: 9, label: 'ninth' }, { id: 10, label: 'tenth updated' }],
    page: 2,
  } })
  // Both pinned v2 clients concatenate only + reset without deduplicating.
  // Keep this exact: missing, wrong, or extra partial targets must not match.
  const reset = await clickResponse(page, 'nestedReset', 'feed,feed')
  expect(reset.headers['x-inertia-reset']).toBe('feed')
  expect(reset.data.mergeProps).toBeUndefined()
  expect(reset.data.prependProps).toBeUndefined()
  expect(reset.data.matchPropsOn).toBeUndefined()
  const replacement = {
    data: [{ id: 3, label: 'third' }], older: [{ id: 8, label: 'eighth' }], page: 3,
  }
  expect(reset.data.props.feed).toEqual(replacement)
  await expect.poll(async () => (await state(page)).feed).toEqual(replacement)
})

test('dotted only/except selections return nested objects with real partial replacement', async ({ page }) => {
  await page.goto('/nested')
  await expect(page.locator('html')).toHaveAttribute('data-ready', 'true')
  const selected = await clickResponse(page, 'nestedOnly', 'selection.profile.name')
  expect(selected.data.props.selection).toEqual({ profile: { name: 'name-2' } })
  expect(selected.data.props).not.toHaveProperty('feed')
  // Inertia shallowly replaces this root prop; the server does not synthesize a
  // deep-merge policy for ordinary nested selection.
  await expect.poll(async () => (await state(page)).selection).toEqual({ profile: { name: 'name-2' } })
  const excluded = await clickResponse(page, 'nestedExcept')
  expect(excluded.headers['x-inertia-partial-except']).toBe('selection.profile.email')
  expect(excluded.data.props.selection).toEqual({ profile: { name: 'name-3' }, settings: { theme: 'theme-3' } })
  await expect.poll(async () => (await state(page)).selection).toEqual({ profile: { name: 'name-3' }, settings: { theme: 'theme-3' } })
})
