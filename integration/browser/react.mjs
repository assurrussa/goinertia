import React, { useEffect } from 'react'
import { createRoot, hydrateRoot } from 'react-dom/client'
import { renderToString } from 'react-dom/server'
import { createInertiaApp, router, useForm, usePage } from 'inertia-client'
import { controls, installObservations } from './shared.mjs'
const h = React.createElement
function View() {
  const page = usePage()
  const form = useForm({ name: '' })
  const actions = controls(router)
  useEffect(() => { document.documentElement.dataset.ready = 'true' }, [])
  return h('main', null,
    h('h1', { id: 'title' }, page.props.title),
    h('p', { id: 'adapter' }, page.props.adapter),
    h('p', { id: 'heavy' }, page.props.heavy || 'loading'),
    h('p', { id: 'items' }, JSON.stringify(page.props.items || [])),
    h('p', { id: 'legacy-flash' }, page.props.flash?.success || ''),
    h('p', { id: 'native-flash' }, page.flash?.message || ''),
    ...Object.entries(actions).map(([name, action]) => h('button', { key: name, id: name, type: 'button', onClick: action }, name)),
    h('form', { onSubmit: (e) => { e.preventDefault(); form.post('/submit', { onFlash: (flash) => { window.__onFlash = flash } }) } },
      h('label', { htmlFor: 'name' }, 'Name'),
      h('input', { id: 'name', name: 'name', value: form.data.name, onChange: (e) => form.setData('name', e.target.value) }),
      h('p', { id: 'error' }, form.errors.name || ''),
      h('button', { id: 'submit', type: 'submit', disabled: form.processing }, 'Submit')))
}
export async function render(page) {
  return createInertiaApp({ page, render: renderToString, resolve: () => View,
    setup: ({ App, props }) => h(App, props) })
}
if (typeof window !== 'undefined') {
  installObservations(router)
  createInertiaApp({ resolve: () => View, setup({ el, App, props }) {
    const hydrated = el.hasChildNodes()
    if (hydrated) hydrateRoot(el, h(App, props))
    else createRoot(el).render(h(App, props))
    document.documentElement.dataset.mode = hydrated ? 'hydrated' : 'mounted'
  } })
}
