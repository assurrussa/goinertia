import { createServer } from 'node:http'
import { parsePage } from '@inertiajs/core/json'
import { render } from 'example-app'
// A small explicit HTTP bridge works with either Go adapter; no Laravel or
// Vite development middleware is required. Keep this endpoint private.
const port = Number(process.env.SSR_PORT || 13714)
createServer(async (request, response) => {
  if (request.url === '/health') { response.end('ok'); return }
  if (request.method !== 'POST' || request.url !== '/render') { response.writeHead(404).end(); return }
  try {
    const chunks = []
    let size = 0
    for await (const chunk of request) {
      size += chunk.length
      if (size > 1_000_000) { response.writeHead(413).end(); return }
      chunks.push(chunk)
    }
    const page = parsePage(Buffer.concat(chunks).toString('utf8'))
    response.setHeader('Content-Type', 'application/json')
    response.end(JSON.stringify(await render(page)))
  } catch (error) {
    console.error('SSR render failed:', error)
    response.writeHead(500).end('SSR render failed')
  }
}).listen(port, '127.0.0.1', () => console.log(`SSR listening on http://127.0.0.1:${port}/render`))
