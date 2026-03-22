import Layout from './Layout'

const pages = import.meta.glob('./Pages/**/*.jsx', { eager: true })

export function resolvePage(name) {
  const page = pages[`./Pages/${name}.jsx`]
  if (!page) {
    throw new Error(`Page ${name} not found!`)
  }

  const component = page.default
  if (component.layout === undefined) {
    component.layout = (pageNode) => <Layout>{pageNode}</Layout>
  }

  return component
}
