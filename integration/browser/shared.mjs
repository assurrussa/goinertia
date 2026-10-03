// Shared fixture controls use the actual client router and real transport.
export function controls(router) {
  return {
    home: () => router.visit('/'),
    second: () => router.visit('/second'),
    form: () => router.visit('/form'),
    more: () => router.get('/feed?page=2', {}, { only: ['items'], preserveState: true }),
    reset: () => router.get('/feed?page=3', {}, { only: ['items'], reset: ['items'], preserveState: true }),
    native: () => router.visit('/native', { onFlash: (flash) => { window.__onFlash = flash } }),
  }
}
export function installObservations(router) {
  window.__flashEvents = []
  window.__onFlash = null
  router.on('flash', (event) => window.__flashEvents.push(event.detail.flash))
}
