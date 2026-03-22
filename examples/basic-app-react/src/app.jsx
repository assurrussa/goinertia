import { createInertiaApp } from '@inertiajs/react'
import { createRoot } from 'react-dom/client'
import './style.css'
import { resolvePage } from './resolvePage'

createInertiaApp({
  resolve: resolvePage,
  setup({ el, App, props }) {
    createRoot(el).render(<App {...props} />)
  },
})
