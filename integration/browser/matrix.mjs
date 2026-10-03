// One bounded invocation; no reruns, retries, watchers or external services.
import { spawnSync } from 'node:child_process'
import process from 'node:process'
const profile = process.argv[2] || 'all'
const versions = profile === 'v2' ? ['2.3.18', '2.3.28']
  : profile === 'v3' ? ['3.8.0'] : profile === 'all' ? ['2.3.18', '2.3.28', '3.8.0'] : null
if (!versions) throw new Error('Usage: npm run matrix -- [all|v2|v3]')
for (const version of versions) {
  for (const framework of ['vue', 'react']) {
    const env = { ...process.env, INERTIA_VERSION: version, FRAMEWORK: framework }
    console.log(`\n=== Inertia ${version} / ${framework}: native Fiber + net/http, CSR + SSR ===`)
    for (const args of [['run', 'build'], ['run', 'verify:ssr'], ['test']]) {
      const result = spawnSync(process.platform === 'win32' ? 'npm.cmd' : 'npm', args, { env, stdio: 'inherit' })
      if (result.error) throw result.error
      if (result.status !== 0) process.exit(result.status ?? 1)
    }
  }
}
console.log('Complete requested browser matrix passed with retries disabled.')
