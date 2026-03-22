import React, { useState } from 'https://esm.sh/react@18.3.1'
import { createRoot } from 'https://esm.sh/react-dom@18.3.1/client?deps=react@18.3.1'
import {
  createInertiaApp,
  Deferred,
  Link,
  usePage,
} from 'https://esm.sh/@inertiajs/react@2.0.11?deps=react@18.3.1,react-dom@18.3.1'
import htm from 'https://esm.sh/htm@3.1.1'

const html = htm.bind(React.createElement)

function Layout({ children }) {
  const page = usePage()
  const menu = page.props.menu ?? []
  const flash = page.props.flash ?? {}

  return html`
    <div className="app-container">
      <nav>
        ${menu.map(
          (item) => html`
            <${Link}
              key=${item.href}
              href=${item.href}
              className=${page.url === item.href ? 'active' : undefined}
            >
              ${item.label}
            <//>
          `,
        )}
      </nav>

      ${flash.success ? html`<div className="flash flash-success">${flash.success}</div>` : null}
      ${flash.error ? html`<div className="flash flash-error">${flash.error}</div>` : null}

      <main className="card">${children}</main>
    </div>
  `
}

function Home({ title, plan, heavy = [] }) {
  return html`
    <div>
      <h1>${title}</h1>
      <p>Welcome to the Home page. This version uses a React client.</p>
      <a href="/undefined-page">Example link to a non-existent page</a>

      <div className="panel">
        <h2>Once prop</h2>
        <p className="muted">
          Plan: <strong>${plan}</strong>
        </p>
      </div>

      <div className="panel">
        <h2>Deferred prop</h2>
        <${Deferred}
          data="heavy"
          fallback=${html`<div className="muted">Loading heavy data...</div>`}
        >
          <ul className="list">
            ${heavy.map((item) => html`<li key=${item}>${item}</li>`)}
          </ul>
        <//>
      </div>
    </div>
  `
}

const sortOptions = [
  { value: 'name', label: 'Name ↑' },
  { value: 'name_desc', label: 'Name ↓' },
  { value: 'id_desc', label: 'ID ↓' },
  { value: 'role', label: 'Role' },
]

function Users({ title, users = [], sort, page, totalPages, prevPage, nextPage }) {
  return html`
    <div>
      <h1>${title}</h1>

      <div className="controls">
        <span className="label">Sort:</span>
        ${sortOptions.map(
          (option) => html`
            <${Link}
              key=${option.value}
              href=${`/users?sort=${option.value}`}
              className=${`chip${sort === option.value ? ' active' : ''}`}
            >
              ${option.label}
            <//>
          `,
        )}
      </div>

      <ul className="user-list">
        ${users.map(
          (user) => html`
            <li key=${user.id} className="user-item">
              <div className="user-name">${user.name}</div>
              <div className="user-meta">#${user.id} · ${user.role}</div>
            </li>
          `,
        )}
      </ul>

      <div className="pager">
        ${prevPage
          ? html`
              <${Link} href=${`/users?sort=${sort}&page=${prevPage}`} className="btn secondary">
                Prev
              <//>
            `
          : null}

        ${nextPage
          ? html`
              <${Link}
                href=${`/users?sort=${sort}&page=${nextPage}`}
                className="btn"
                preserveScroll=${true}
                only=${['users', 'page', 'prevPage', 'nextPage', 'totalPages']}
              >
                Load more
              <//>
            `
          : null}
      </div>

      <p className="muted">Page ${page} of ${totalPages}</p>
    </div>
  `
}

function Settings({ title, diagnostics }) {
  const [form, setForm] = useState({ name: '', email: '' })
  const [errors, setErrors] = useState({})
  const [status, setStatus] = useState('')

  const handleChange = (event) => {
    const { name, value } = event.target
    setForm((current) => ({ ...current, [name]: value }))
  }

  const validateForm = async (event) => {
    event.preventDefault()
    setStatus('Validating...')
    setErrors({})

    const payload = new URLSearchParams()
    payload.set('name', form.name)
    payload.set('email', form.email)

    try {
      const response = await fetch('/users/create', {
        method: 'POST',
        headers: {
          Precognition: 'true',
          'Precognition-Validate-Only': 'name,email',
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: payload.toString(),
      })

      if (response.status === 422) {
        const data = await response.json()
        setErrors(data.errors ?? {})
        setStatus('Fix the errors above')
        return
      }

      if (!response.ok) {
        throw new Error(`Unexpected status ${response.status}`)
      }

      setStatus('Looks good')
    } catch (_error) {
      setStatus('Request failed')
    }
  }

  return html`
    <div>
      <h1>${title}</h1>
      <p>Settings page content.</p>

      <div className="panel">
        <h2>Optional prop</h2>
        <p className="muted">Diagnostics are loaded only when requested.</p>

        ${diagnostics
          ? html`
              <div className="diagnostics">
                <pre>${JSON.stringify(diagnostics, null, 2)}</pre>
              </div>
            `
          : html`
              <${Link} href="/settings" className="btn" preserveScroll=${true} only=${['diagnostics']}>
                Load diagnostics
              <//>
            `}
      </div>

      <div className="panel">
        <h2>Precognition form</h2>
        <p className="muted">Validation-only request using Precognition headers.</p>

        <form className="form-grid" onSubmit=${validateForm}>
          <label className="field">
            Name
            <input
              name="name"
              value=${form.name}
              onInput=${handleChange}
              type="text"
              className="input"
            />
            ${errors.name ? html`<span className="error">${errors.name[0]}</span>` : null}
          </label>

          <label className="field">
            Email
            <input
              name="email"
              value=${form.email}
              onInput=${handleChange}
              type="email"
              className="input"
            />
            ${errors.email ? html`<span className="error">${errors.email[0]}</span>` : null}
          </label>

          <div className="actions">
            <button type="submit" className="btn">Validate only</button>
            ${status ? html`<span className="status">${status}</span>` : null}
          </div>
        </form>
      </div>
    </div>
  `
}

function ErrorPage({ status, message }) {
  return html`
    <div className="error-page">
      <div className="error-code">${status}</div>
      <p className="error-message">${message || 'Something went wrong'}</p>

      <${Link} href="/" className="btn">Go Home<//>
    </div>
  `
}

const pages = {
  Home,
  Users,
  Settings,
  Error: ErrorPage,
}

for (const component of Object.values(pages)) {
  if (component.layout === undefined) {
    component.layout = (page) => html`<${Layout}>${page}<//>`
  }
}

createInertiaApp({
  resolve: (name) => {
    const page = pages[name]
    if (!page) {
      throw new Error(`Page ${name} not found!`)
    }

    return page
  },
  setup({ el, App, props }) {
    createRoot(el).render(html`<${App} ...${props} />`)
  },
})
