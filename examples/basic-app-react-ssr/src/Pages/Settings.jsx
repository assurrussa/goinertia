import { Link } from '@inertiajs/react'
import axios from 'axios'
import { useState } from 'react'

export default function Settings({ title, diagnostics }) {
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
      await axios.post('/users/create', payload, {
        headers: {
          Precognition: 'true',
          'Precognition-Validate-Only': 'name,email',
          'Content-Type': 'application/x-www-form-urlencoded',
        },
      })
      setStatus('Looks good')
    } catch (error) {
      if (error.response?.status === 422) {
        setErrors(error.response.data?.errors ?? {})
        setStatus('Fix the errors above')
        return
      }

      setStatus('Request failed')
    }
  }

  return (
    <div>
      <h1>{title}</h1>
      <p>Settings page content.</p>

      <div className="panel">
        <h2>Optional prop</h2>
        <p className="muted">Diagnostics are loaded only when requested.</p>

        {diagnostics ? (
          <div className="diagnostics">
            <pre>{JSON.stringify(diagnostics, null, 2)}</pre>
          </div>
        ) : (
          <Link href="/settings" className="btn" preserveScroll only={['diagnostics']}>
            Load diagnostics
          </Link>
        )}
      </div>

      <div className="panel">
        <h2>Precognition form</h2>
        <p className="muted">Validation-only request using Precognition headers.</p>

        <form className="form-grid" onSubmit={validateForm}>
          <label className="field">
            Name
            <input
              name="name"
              value={form.name}
              onChange={handleChange}
              type="text"
              className="input"
            />
            {errors.name ? <span className="error">{errors.name[0]}</span> : null}
          </label>

          <label className="field">
            Email
            <input
              name="email"
              value={form.email}
              onChange={handleChange}
              type="email"
              className="input"
            />
            {errors.email ? <span className="error">{errors.email[0]}</span> : null}
          </label>

          <div className="actions">
            <button type="submit" className="btn">
              Validate only
            </button>
            {status ? <span className="status">{status}</span> : null}
          </div>
        </form>
      </div>
    </div>
  )
}
