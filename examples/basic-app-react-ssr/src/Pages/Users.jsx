import { Link } from '@inertiajs/react'

const sortOptions = [
  { value: 'name', label: 'Name ↑' },
  { value: 'name_desc', label: 'Name ↓' },
  { value: 'id_desc', label: 'ID ↓' },
  { value: 'role', label: 'Role' },
]

export default function Users({ title, users = [], sort, page, totalPages, prevPage, nextPage }) {
  return (
    <div>
      <h1>{title}</h1>

      <div className="controls">
        <span className="label">Sort:</span>
        {sortOptions.map((option) => (
          <Link
            key={option.value}
            href={`/users?sort=${option.value}`}
            className={`chip${sort === option.value ? ' active' : ''}`}
          >
            {option.label}
          </Link>
        ))}
      </div>

      <ul className="user-list">
        {users.map((user) => (
          <li key={user.id} className="user-item">
            <div className="user-name">{user.name}</div>
            <div className="user-meta">
              #{user.id} · {user.role}
            </div>
          </li>
        ))}
      </ul>

      <div className="pager">
        {prevPage ? (
          <Link href={`/users?sort=${sort}&page=${prevPage}`} className="btn secondary">
            Prev
          </Link>
        ) : null}
        {nextPage ? (
          <Link
            href={`/users?sort=${sort}&page=${nextPage}`}
            className="btn"
            preserveScroll
            only={['users', 'page', 'prevPage', 'nextPage', 'totalPages']}
          >
            Load more
          </Link>
        ) : null}
      </div>

      <p className="muted">
        Page {page} of {totalPages}
      </p>
    </div>
  )
}
