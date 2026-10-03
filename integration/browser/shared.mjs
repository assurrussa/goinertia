// Shared fixture controls use the actual client router and real transport.
export function controls(router) {
  return {
    home: () => router.visit('/'),
    second: () => router.visit('/second'),
    form: () => router.visit('/form'),
    more: () => router.get('/feed?page=2', {}, { only: ['items'], preserveState: true }),
    reset: () => router.get('/feed?page=3', {}, { only: ['items'], reset: ['items'], preserveState: true }),
    propsNext: () => router.visit('/props?step=2'),
    propsReload: () => router.reload({ only: ['cached'], data: { step: 2 } }),
    propsExcept: () => router.reload({ except: ['catalog'], data: { step: 2 } }),
    propsFresh: () => router.visit('/props?step=2&fresh=1'),
    propsRenamed: () => router.visit('/props-renamed?step=2'),
    propsOptional: () => router.reload({ only: ['optionalOnce', 'onceOptional'] }),
    nestedMore: () => router.get('/nested?step=2', {}, { only: ['feed'], preserveState: true }),
    nestedReset: () => router.get('/nested?step=3', {}, { only: ['feed'], reset: ['feed'], preserveState: true }),
    nestedOnly: () => router.get('/nested?step=2', {}, { only: ['selection.profile.name'], preserveState: true }),
    nestedExcept: () => router.get('/nested?step=3', {}, { except: ['selection.profile.email'], preserveState: true }),
    native: () => router.visit('/native', { onFlash: (flash) => { window.__onFlash = flash } }),
  }
}
export function installObservations(router) {
  window.__router = router
  window.__cancelled = 0
  window.__locations = []
  window.__flashEvents = []
  window.__onFlash = null
  if (__INERTIA_V3__) router.on('location', (event) => window.__locations.push({ url: String(event.detail.url), versionChange: event.detail.versionChange }))
  router.on('flash', (event) => window.__flashEvents.push(event.detail.flash))
}

// JSON display never feeds the client: the public client parser is responsible
// for preserving BigInts. This view merely makes their runtime type observable.
export const displayJSON = (value) => JSON.stringify(value, (_key, item) =>
  typeof item === 'bigint' ? `${item}n` : item)
export const pageMetadata = (page) => displayJSON({
  rescuedProps: page.rescuedProps, sharedProps: page.sharedProps,
  clearHistory: page.clearHistory, encryptHistory: page.encryptHistory,
  preserveBigIntegers: page.preserveBigIntegers, version: page.version,
})
