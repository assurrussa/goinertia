import { createSSRApp, h } from 'vue'
import { createInertiaApp } from '@inertiajs/vue3'
import '../src/style.css'
import Layout from './Layout.vue'

const pages = import.meta.glob('./Pages/**/*.vue', { eager: true })

createInertiaApp({
  resolve: (name) => {
    const page = pages[`./Pages/${name}.vue`]
    
    if (!page) {
      throw new Error(`Page ${name} not found!`)
    }

    const mod = page.default
    
    // Set default layout if not present
    if (mod.layout === undefined) {
       mod.layout = Layout
    }
    
    return mod
  },
  setup({ el, App, props, plugin }) {
    createSSRApp({ render: () => h(App, props) })
      .use(plugin)
      .mount(el)
  },
})