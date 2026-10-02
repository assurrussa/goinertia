import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { DeferredReplay, ResponseReplay, scrollResetReplay, setCurrent } from './client-v2.3.18.ts'

const fixtures = JSON.parse(readFileSync(process.argv[2], 'utf8'))
for (const fixture of fixtures) {
  const initial = fixture.initial
  const scheduler = new DeferredReplay()
  scheduler.apply(initial.deferredProps)
  assert.equal(scheduler.requests.length, 2, `${fixture.adapter}: expected two initial groups`)
  for (const request of [...scheduler.requests]) {
    const response = fixture.deferred[request.only[0]]
    scheduler.apply(response.deferredProps)
  }
  assert.equal(scheduler.requests.length, 2, `${fixture.adapter}: deferred reload scheduled another group`)

  const response = new ResponseReplay()
  setCurrent(structuredClone(initial))
  response.apply(fixture.excluded)
  assert.deepEqual(fixture.excluded.props.profile, { name: 'Alice' }, `${fixture.adapter}: excluded deep merge erased profile`)

  setCurrent(structuredClone(initial))
  response.apply(fixture.append)
  assert.deepEqual(fixture.append.props.posts.data, [1, 2], `${fixture.adapter}: scroll append lost page one`)
  setCurrent(structuredClone(initial))
  response.apply(fixture.prepend)
  assert.deepEqual(fixture.prepend.props.posts.data, [2, 1], `${fixture.adapter}: scroll prepend lost page one`)

  setCurrent(structuredClone(initial))
  response.apply(fixture.reset)
  assert.deepEqual(fixture.reset.props.posts.data, [2], `${fixture.adapter}: reset still merged old results`)
  const reset = scrollResetReplay(fixture.reset)
  assert.equal(reset.resets, 1, `${fixture.adapter}: scroll reset callback was skipped`)
  assert.equal(reset.state.requestCount, 0)
  assert.equal(reset.state.lastLoadedPage, 2)
  console.log(`${fixture.adapter}: grouped deferred, excluded deep merge, paginator append/prepend/reset PASS`)
}
