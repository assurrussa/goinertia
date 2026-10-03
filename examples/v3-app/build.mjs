import { build } from 'esbuild'
const framework = process.env.FRAMEWORK || 'vue'
if (!['vue', 'react'].includes(framework)) throw new Error('FRAMEWORK must be vue or react')
const common = {
  bundle: true,
  format: 'esm',
  define: { 'process.env.NODE_ENV': '"production"', __VUE_OPTIONS_API__: 'true', __VUE_PROD_DEVTOOLS__: 'false' },
  alias: { 'example-app': `./src/${framework}.mjs` },
  logLevel: 'info',
}
await build({ ...common, platform: 'browser', entryPoints: [`src/${framework}.mjs`], outfile: 'dist/app.js', minify: true })
await build({ ...common, platform: 'node', entryPoints: ['src/ssr.mjs'], outfile: 'dist/ssr.mjs',
  banner: { js: "import { createRequire } from 'node:module'; const require = createRequire(import.meta.url);" } })
console.log(`Built official Inertia 3.8.0 ${framework} client and SSR renderer`)
