import { createServer } from 'node:http'
import { render } from 'fixture-framework'
const port = Number(process.env.SSR_PORT || 18985)
createServer(async (req, res) => {
  if (req.url === '/health') { res.end('ok'); return }
  if (req.method !== 'POST' || req.url !== '/render') { res.writeHead(404).end(); return }
  try {
    let body = ''
    for await (const chunk of req) {
      body += chunk
      if (body.length > 1_000_000) throw new Error('SSR fixture body limit')
    }
    const rendered = await render(JSON.parse(body))
    res.setHeader('Content-Type', 'application/json')
    res.end(JSON.stringify(rendered))
  } catch (error) { console.error(error); res.writeHead(500).end('SSR fixture error') }
}).listen(port, '127.0.0.1')
