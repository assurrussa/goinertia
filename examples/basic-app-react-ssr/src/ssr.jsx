import { createInertiaApp } from '@inertiajs/react'
import createServer from '@inertiajs/react/server'
import { renderToString } from 'react-dom/server'
import { resolvePage } from './resolvePage'

createServer((page) =>
  createInertiaApp({
    page,
    render: renderToString,
    resolve: resolvePage,
    setup: ({ App, props }) => <App {...props} />,
  }),
)
