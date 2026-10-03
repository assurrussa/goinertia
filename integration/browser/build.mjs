import { build } from 'esbuild'
import { mkdir, writeFile } from 'node:fs/promises'

const version = process.env.INERTIA_VERSION || '2.3.28'
const framework = process.env.FRAMEWORK || 'vue'
if (!['2.3.18', '2.3.28'].includes(version) || !['vue', 'react'].includes(framework)) {
  throw new Error('Supported matrix: Inertia 2.3.18/2.3.28 × Vue/React')
}
const client = version === '2.3.18'
  ? `inertia-${framework === 'vue' ? 'vue' : 'react'}-v218`
  : `@inertiajs/${framework === 'vue' ? 'vue3' : 'react'}`
const common = {
  bundle: true,
  alias: { 'inertia-client': client },
  define: { 'process.env.NODE_ENV': '"development"', __VUE_OPTIONS_API__: 'true', __VUE_PROD_DEVTOOLS__: 'false', __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: 'true' },
  logLevel: 'info',
}
await mkdir('dist', { recursive: true })
await build({ ...common, entryPoints: [`${framework}.mjs`], outfile: 'dist/app.js', platform: 'browser', format: 'esm' })
await build({ ...common, entryPoints: ['ssr.mjs'], outfile: 'dist/ssr.mjs', platform: 'node', format: 'esm',
  alias: { ...common.alias, 'fixture-framework': `./${framework}.mjs` },
  packages: 'external' })
await writeFile('dist/matrix.json', JSON.stringify({ version, framework }) + '\n')
