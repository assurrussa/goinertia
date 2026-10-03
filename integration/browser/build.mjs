import { build } from 'esbuild'
import { mkdir, writeFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'

const version = process.env.INERTIA_VERSION || '2.3.28'
const framework = process.env.FRAMEWORK || 'vue'
if (!['2.3.18', '2.3.28', '3.8.0'].includes(version) || !['vue', 'react'].includes(framework)) {
  throw new Error('Supported matrix: Inertia 2.3.18/2.3.28/3.8.0 × Vue/React')
}
const v3 = version.startsWith('3.')
const resolve = v3 ? (await import('./v3/resolve.mjs')).resolve : (name) => import.meta.resolve(name)
const profile = { resolve: (name) => fileURLToPath(resolve(name)) }
const client = version === '2.3.18'
  ? `inertia-${framework === 'vue' ? 'vue' : 'react'}-v218`
  : `@inertiajs/${framework === 'vue' ? 'vue3' : 'react'}`
// Absolute package entrypoints keep React 19 in the v3 profile and React 18 in
// both v2 profiles, including SSR. No peer overrides or runtime substitution.
const alias = { 'inertia-client': profile.resolve(client),
  'fixture-json': v3 ? profile.resolve('@inertiajs/core/json') : new URL('./json-v2.mjs', import.meta.url).pathname,
}
for (const name of ['react', 'react-dom/client', 'react-dom/server', 'vue', '@vue/server-renderer']) {
  alias[name] = profile.resolve(name)
}
const common = {
  bundle: true,
  alias,
  define: {
    'process.env.NODE_ENV': '"development"', __VUE_OPTIONS_API__: 'true',
    __VUE_PROD_DEVTOOLS__: 'false', __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: 'true',
    __INERTIA_V3__: JSON.stringify(v3),
  },
  logLevel: 'info',
}
await mkdir('dist', { recursive: true })
await build({ ...common, alias: { ...alias,
  'react-dom/server': profile.resolve('react-dom/server.browser'),
  'vue': profile.resolve('vue/dist/vue.runtime.esm-bundler.js'),
  '@vue/server-renderer': profile.resolve('@vue/server-renderer/dist/server-renderer.esm-bundler.js'),
}, entryPoints: [`${framework}.mjs`], outfile: 'dist/app.js', platform: 'browser', format: 'esm' })
const ssrOptions = { ...common, platform: 'node', format: 'esm',
  alias: { ...alias, 'fixture-framework': `./${framework}.mjs` },
  banner: { js: "import { createRequire } from 'node:module'; const require = createRequire(import.meta.url);" },
}
await build({ ...ssrOptions, entryPoints: ['ssr.mjs'], outfile: 'dist/ssr.mjs' })
await build({ ...ssrOptions, entryPoints: [`${framework}.mjs`], outfile: 'dist/render.mjs' })
await writeFile('dist/matrix.json', JSON.stringify({ version, framework, protocol: v3 ? 3 : 2 }) + '\n')
