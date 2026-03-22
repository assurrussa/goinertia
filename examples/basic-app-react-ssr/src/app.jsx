import { createInertiaApp } from '@inertiajs/react'
import { hydrateRoot } from 'react-dom/client'
import './style.css'
import { resolvePage } from './resolvePage'

createInertiaApp({
  resolve: resolvePage,
  setup({ el, App, props }) {
    hydrateRoot(el, <App {...props} />)
  },
})
