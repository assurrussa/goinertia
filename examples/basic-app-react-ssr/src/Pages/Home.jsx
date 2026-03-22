import { Deferred } from '@inertiajs/react'

export default function Home({ title, plan, heavy = [] }) {
  return (
    <div>
      <h1>{title}</h1>
      <p>Welcome to the Home page. This SSR version uses a React client.</p>
      <a href="/undefined-page">Example link to a non-existent page</a>

      <div className="panel">
        <h2>Once prop</h2>
        <p className="muted">
          Plan: <strong>{plan}</strong>
        </p>
      </div>

      <div className="panel">
        <h2>Deferred prop</h2>
        <Deferred data="heavy" fallback={<div className="muted">Loading heavy data...</div>}>
          <ul className="list">
            {heavy.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </Deferred>
      </div>
    </div>
  )
}
