# Lazy props

```go
func Dashboard(c fiber.Ctx, inertia *goinertia.Inertia) error {
    inertia.WithLazyProp(c, "analytics", func(ctx context.Context) (any, error) {
        return calculateAnalytics(), nil
    })

    inertia.WithLazyProp(c, "reports", func(ctx context.Context) (any, error) {
        return generateReports(), nil
    })

    return inertia.Render(c, "Dashboard", map[string]any{
        "title": "Dashboard",
    })
}
```

## Deferred props

Deferred props are not included in the initial response. They are loaded later via a partial reload.

```go
return inertia.Render(c, "Dashboard", map[string]any{
    "heavy": goinertia.Defer(goinertia.LazyProp{
        Key: "heavy",
        Fn: func(ctx context.Context) (any, error) {
            return loadHeavyData(), nil
        },
    }),
})
```

## Optional props

Optional props are omitted on full visits and resolved on a matching partial
reload when selected by only or not excluded by except. Deferred props follow
the same selection policy; partial responses do not announce deferred groups.
This corrects the earlier except-only omission. See
[protocol coverage](protocol-coverage.md) for the complete contract.

```go
return inertia.Render(c, "Dashboard", map[string]any{
    "metrics": goinertia.Optional(goinertia.LazyProp{
        Key: "metrics",
        Fn: func(ctx context.Context) (any, error) {
            return loadMetrics(), nil
        },
    }),
})
```

## Once props

Once props are sent once and then skipped on subsequent requests unless explicitly refreshed.

```go
return inertia.Render(c, "Dashboard", map[string]any{
    "plans": goinertia.Once("basic", goinertia.WithOnceKey("plans_v1")),
})
```

## Merge props

Merge props are merged during partial reloads (append or prepend for lists).

```go
return inertia.Render(c, "Users", map[string]any{
    "users":   goinertia.Merge(users),
    "newest":  goinertia.Prepend(newUsers),
    "profile": goinertia.DeepMerge(profile),
})
```

## Scroll props

Scroll props add pagination metadata for infinite scroll.

Merge, scroll and once metadata are filtered with their selected props. Partial
responses do not announce new deferred requests. A reset returns the selected
merge value without an append/prepend/deep label; scroll metadata emits `reset: true`. A map paginator containing a `data`
member merges that path; a bare array still merges at the root prop.

```go
return inertia.Render(c, "Feed", map[string]any{
    "posts": goinertia.Scroll(posts, goinertia.ScrollPropConfig{
        PageName:    "page",
        NextPage:    3,
        CurrentPage: 2,
    }),
})
```

Nested lazy callbacks in map[string]any and []any containers use separate cache
keys for literal keys, nested map/array paths and flash/shared/context/request
layers. Context and request callbacks at the same path are independent; request
values retain precedence. Input containers are copied only when a callback
changes a child, and are never mutated by resolution.

## Nested partial selection (v2 follow-up)

Dotted `only` and `except` paths traverse `map[string]any` values before
resolving callbacks. For example, `only: ['auth.user.name']` returns only that
leaf under its original parent maps. Selecting a parent includes its entire
subtree; a missing `only` path omits the prop. Excluding a missing descendant is
a no-op. Arrays, structs and typed maps are opaque: numeric array paths are not
supported. Required ancestor callbacks run to reveal their maps, but excluded
sibling callbacks never run. Filtering copies maps and never changes shared
input or cached callback results.

A token matches both the exact literal dotted key and the corresponding nested
path, at every map level. Thus `auth.user` selects a literal `"auth.user"` prop
and `auth["user"]` when both exist. This preserves the existing literal-key API.
As before, a nonempty `except` header takes precedence over `only`; a mismatched
component ignores both. Reserved props and `Always` values bypass selection.

## Nested merge targets

```go
core.MergeAt(map[string]any{
    "data": users,
    "older": olderUsers,
    "total": total,
}, core.AppendAt("data", "id"), core.PrependAt("older", "id"))
```

`MergeAt` adds independent append/prepend targets without changing `MergeProp`
or the existing `Merge`, `Prepend` and `DeepMerge` constructors. Paths are
relative to the root prop and traverse JSON maps; an empty path targets the
root. The optional matching field is relative to each array item. On a prop
named `users`, the example emits `users.data`, `users.older`, and matching paths
`users.data.id`, `users.older.id`. Untargeted fields are replaced by the client.
No implicit root merge label is added. Configure each target once.

Metadata is emitted only for target paths present in successfully resolved,
selected data. A `MergeAt` value replaces lower-precedence data and wrapper metadata;
if its callback fails, that root is omitted instead of reusing a lower-layer
value. This is the new API’s policy; legacy root merge behavior is unchanged. A reset of the root prop suppresses all its wrapper merge and
matching labels. `Defer(MergeAt(...))` and `MergeAt(Defer(...), ...)` are supported.
Arrays may be typed; path traversal itself does not use reflection.

## Once freshness and composition

`Once(value, WithOnceFresh())` sends a remembered value again; pass
`WithOnceFresh(false)` to turn it off. Freshness never bypasses selection,
optional omission or deferred loading. `WithOnceKey` and `WithOnceExpiresAt`
continue to configure client identity and expiry (Unix milliseconds).

Legacy behavior is preserved: an explicit `only` selection refreshes a remembered
Once prop, while an except-only request does not. Opt into the official v2
reference adapter's selected-partial refresh policy with
`Once(value, WithOnceRefreshOnPartial())`. This refreshes selected Once values
on both only and except requests. A matching component alone still does not
make a request partial: this adapter requires a nonempty only/except filter.
The existing Except-Once header policy also remains unchanged: if supplied on
a non-Inertia HTML request, it still participates in remembered-value omission.
The pinned PHP reference restricts that omission to Inertia JSON requests.

One Once wrapper can compose in either order with `Defer`, `Optional`, `Merge`,
`Prepend`, `DeepMerge` and `MergeAt` in a top-level wrapper chain. This is not
recursive wrapper interpretation inside map/array values. Initial deferred or
optional values remain omitted, with Once registration retained. Remembered
Once+deferred values do not announce another deferred request; freshness
re-enables the announcement. Once metadata intentionally remains when a cached
value is omitted, unlike merge metadata, which requires resolved data.

Sources: [v2 merging](https://inertiajs.com/docs/v2/data-props/merging-props),
[v2 Once](https://inertiajs.com/docs/v2/data-props/once-props), and
[pinned server reference](https://github.com/inertiajs/inertia-laravel/blob/67666f22924515f44173dfa8bd4a608580519999/src/Response.php).
