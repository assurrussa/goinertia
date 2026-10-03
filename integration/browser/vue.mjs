import { createSSRApp, createApp, h, onMounted } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createInertiaApp, Deferred, Head, router, useForm, usePage } from 'inertia-client'
import { controls, installObservations, displayJSON, pageMetadata } from './shared.mjs'

const View = {
  setup() {
    const page = usePage()
    const form = useForm({ name: '' })
    const actions = controls(router)
    onMounted(() => { document.documentElement.dataset.ready = 'true' })
    return () => h('main', [
      h(Head, { title: page.props.title }),
      h('h1', { id: 'title' }, page.props.title),
      h('p', { id: 'adapter' }, page.props.adapter),
      h('p', { id: 'heavy' }, page.props.heavy || 'loading'),
      h('p', { id: 'items' }, JSON.stringify(page.props.items || [])),
      h('pre', { id: 'prop-state' }, displayJSON(page.props)),
      h('pre', { id: 'page-meta' }, pageMetadata(page)),
      h('p', { id: 'bigint-type' }, typeof page.props.large?.id),
      h('div', { id: 'destination' }, 'Fragment destination'),
      page.props.arrayFixture ? h(Deferred, { data: 'array.0' }, {
        default: () => h('p', { id: 'array-value' }, page.props.array?.[0] || 'missing'),
        fallback: () => h('p', { id: 'array-fallback' }, 'Loading array'),
      }) : null,
      h('p', { id: 'legacy-flash' }, page.props.flash?.success || ''),
      h('p', { id: 'native-flash' }, page.flash?.message || ''),
      ...Object.entries(actions).map(([name, action]) => h('button', { id: name, type: 'button', onClick: action }, name)),
      h('form', { onSubmit: (e) => { e.preventDefault(); form.post('/submit', { onFlash: (flash) => { window.__onFlash = flash } }) } }, [
        h('label', { for: 'name' }, 'Name'),
        h('input', { id: 'name', name: 'name', value: form.name, onInput: (e) => { form.name = e.target.value } }),
        h('p', { id: 'error' }, form.errors.name || ''),
        h('button', { id: 'submit', type: 'submit', disabled: form.processing }, 'Submit'),
      ]),
    ])
  },
}
export async function render(page) {
  return createInertiaApp({ page, render: renderToString, resolve: () => View,
    setup: ({ App, props, plugin }) => createSSRApp({ render: () => h(App, props) }).use(plugin) })
}
if (typeof window !== 'undefined') {
  installObservations(router)
  createInertiaApp({ resolve: () => View, setup({ el, App, props, plugin }) {
    const hydrated = el.hasChildNodes()
    const app = hydrated ? createSSRApp : createApp
    app({ render: () => h(App, props) }).use(plugin).mount(el)
    document.documentElement.dataset.mode = hydrated ? 'hydrated' : 'mounted'
  } })
}
