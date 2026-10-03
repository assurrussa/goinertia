// Fast non-browser evidence only. Browser hydration remains a separate gate.
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { render } from './dist/render.mjs'
const matrix = JSON.parse(readFileSync(new URL('./dist/matrix.json', import.meta.url), 'utf8'))
const v3 = matrix.protocol === 3
const payload = {
  component: 'Smoke', url: '/', version: 'browser-v1',
  props: { errors: {}, title: 'SSR smoke', adapter: 'smoke', unsafe: '</script>雪',
    ...(v3 ? { large: { id: 9007199254740993n } } : {}),
  },
  ...(v3 ? { sharedProps: [], preserveBigIntegers: true } : {}),
}
const rendered = await render(payload)
assert.ok(rendered.body.includes('<h1 id="title">SSR smoke</h1>'))
assert.ok(rendered.body.includes(v3 ? 'type="application/json"' : 'data-page='))
assert.ok(!rendered.body.includes('"unsafe":"</script>'))
assert.ok(Array.isArray(rendered.head))
assert.ok(rendered.head.some(tag => tag.includes("SSR smoke") && tag.includes(v3 ? "data-inertia" : "inertia")))
console.log(`SSR smoke passed: official Inertia ${matrix.version}, ${matrix.framework} (not a browser/hydration pass)`)
