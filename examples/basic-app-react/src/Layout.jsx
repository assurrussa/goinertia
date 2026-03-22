import { Link, usePage } from '@inertiajs/react'

export default function Layout({ children }) {
  const page = usePage()
  const menu = page.props.menu ?? []
  const flash = page.props.flash ?? {}

  return (
    <div className="app-container">
      <nav>
        {menu.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className={page.url === item.href ? 'active' : undefined}
          >
            {item.label}
          </Link>
        ))}
      </nav>

      {flash.success && <div className="flash flash-success">{flash.success}</div>}
      {flash.error && <div className="flash flash-error">{flash.error}</div>}

      <main className="card">{children}</main>
    </div>
  )
}
