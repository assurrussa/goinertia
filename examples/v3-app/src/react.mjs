import React from 'react'
import { createRoot, hydrateRoot } from 'react-dom/client'
import { renderToString } from 'react-dom/server'
import { createInertiaApp, Link, usePage } from '@inertiajs/react'
const h = React.createElement
function Page() {
  const page = usePage()
  return h('main', null,
    h('h1', null, page.props.title),
    h('p', null, `Inertia v3 · React 19 · ${page.props.adapter}`),
    h('nav', null, h(Link, { href: '/' }, 'Home'), ' · ', h(Link, { href: '/about' }, 'About'), ' · ', h(Link, { href: '/without-ssr' }, 'CSR only')),
    h('p', null, page.props.details?.message || 'Loading deferred details…'),
    h('p', null, page.flash?.notice || ''),
    h('p', null, page.rescuedProps?.length ? 'An optional dependency failed; the rest of the page is available.' : ''))
}
export const render = (page) => createInertiaApp({ page, render: renderToString, resolve: () => Page,
  setup: ({ App, props }) => h(App, props) })
if (typeof window !== 'undefined') {
  createInertiaApp({ resolve: () => Page, setup: ({ el, App, props }) => {
    if (el.hasChildNodes()) hydrateRoot(el, h(App, props))
    else createRoot(el).render(h(App, props))
  } })
}
