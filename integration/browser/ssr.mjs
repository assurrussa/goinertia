import { createServer } from 'node:http'
import { render } from 'fixture-framework'
import { parsePage } from 'fixture-json'
const port = Number(process.env.SSR_PORT || 18985)
createServer(async (req, res) => {
  if (req.url === '/health') { res.end('ok'); return }
  if (req.method !== 'POST' || req.url !== '/render') { res.writeHead(404).end(); return }
  try {
    const chunks = []
    let size = 0
    for await (const chunk of req) {
      size += chunk.length
      if (size > 1_000_000) throw new Error('SSR fixture body limit')
      chunks.push(chunk)
    }
    const page = parsePage(Buffer.concat(chunks).toString('utf8'))
    if (page.component === 'SSRFailure') throw new Error('Synthetic SSR failure')
    const rendered = await render(page)
    res.setHeader('Content-Type', 'application/json')
    res.end(JSON.stringify(rendered))
  } catch (error) { console.error(error); res.writeHead(500).end('SSR fixture error') }
}).listen(port, '127.0.0.1')
