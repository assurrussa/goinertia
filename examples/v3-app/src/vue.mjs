import { createApp, createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createInertiaApp, Link, usePage } from '@inertiajs/vue3'
const Page = { setup() {
  const page = usePage()
  return () => h('main', [
    h('h1', page.props.title),
    h('p', `Inertia v3 · Vue 3 · ${page.props.adapter}`),
    h('nav', [h(Link, { href: '/' }, () => 'Home'), ' · ', h(Link, { href: '/about' }, () => 'About'), ' · ', h(Link, { href: '/without-ssr' }, () => 'CSR only')]),
    h('p', page.props.details?.message || 'Loading deferred details…'),
    h('p', page.flash?.notice || ''),
    h('p', page.rescuedProps?.length ? 'An optional dependency failed; the rest of the page is available.' : ''),
  ])
} }
export const render = (page) => createInertiaApp({ page, render: renderToString, resolve: () => Page,
  setup: ({ App, props, plugin }) => createSSRApp({ render: () => h(App, props) }).use(plugin) })
if (typeof window !== 'undefined') {
  createInertiaApp({ resolve: () => Page, setup: ({ el, App, props, plugin }) => {
    const app = el.hasChildNodes() ? createSSRApp : createApp
    app({ render: () => h(App, props) }).use(plugin).mount(el)
  } })
}
