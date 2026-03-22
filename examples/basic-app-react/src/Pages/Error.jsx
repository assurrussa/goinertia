import { Link } from '@inertiajs/react'

export default function ErrorPage({ status, message }) {
  return (
    <div className="error-page">
      <div className="error-code">{status}</div>
      <p className="error-message">{message || 'Something went wrong'}</p>

      <Link href="/" className="btn">
        Go Home
      </Link>
    </div>
  )
}
