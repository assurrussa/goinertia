# Configuration Options

`goinertia.New` and `goinertia.NewWithValidation` accept a variable number of `Option` functions to configure the
instance.

## Core Options

| Option                               | Description                                                                                          |
|--------------------------------------|------------------------------------------------------------------------------------------------------|
| `WithFS(fs fs.FS)`                   | Sets the filesystem for loading templates. Usually `os.DirFS("views")` or `embed.FS`.                |
| `WithPublicFS(fs fs.ReadFileFS)`     | Sets the filesystem for reading public assets (e.g., for `hot` file check).                          |
| `WithRootTemplate(path string)`      | Sets the path to the root layout template. Default: `app.gohtml`.                                    |
| `WithRootErrorTemplate(path string)` | Sets the path to the error page template. Default: `error.gohtml`.                                   |
| `WithAssetVersion(version string)`   | Sets the asset version string to force client-side reloads when assets change.                       |
| `WithDevMode()`                      | Enables development mode: disables template caching and checks for Vite `hot` file on every request. |

## Data & Context

| Option                                         | Description                                                                          |
|------------------------------------------------|--------------------------------------------------------------------------------------|
| `WithSharedProps(props map[string]any)`        | Adds global props accessible to all Inertia pages (e.g., user info, flash messages). |
| `WithSharedViewData(data map[string]any)`      | Adds data available to the root template (Go template), but not passed to JS.        |
| `WithSetSharedFuncMap(funcs template.FuncMap)` | Adds custom functions to the Go template engine (e.g., `asset`, `url`).              |
| `WithSessionStore(store SessionStore)`         | Configures the session store for Flash messages and validation errors.               |

## Security & CSRF

| Option                           | Description                                                         |
|----------------------------------|---------------------------------------------------------------------|
| `WithCSRFTokenProvider(fn)`      | Sets a function to retrieve the CSRF token from the context.        |
| `WithCSRFTokenCheckProvider(fn)` | Sets a function to validate the CSRF token on requests.             |
| `WithCSRFPropName(name string)`  | Customizes the prop name for the CSRF token. Default: `csrf_token`. |

## Error Handling

| Option                              | Description                                                                                   |
|-------------------------------------|-----------------------------------------------------------------------------------------------|
| `WithLogger(logger Logger)`         | Sets a custom logger. Default is a no-op logger.                                              |
| `WithCanExposeDetails(fn)`          | Callback to determine if detailed error messages should be shown (e.g., based on admin role). |
| `WithCustomErrorGettingHandler(fn)` | Customizes how errors are extracted/processed.                                                |
| `WithCustomErrorDetailsHandler(fn)` | Customizes how error details are formatted for the response.                                  |

## SSR

| Option                            | Description                                                                    |
|-----------------------------------|--------------------------------------------------------------------------------|
| `WithSSRConfig(config SSRConfig)` | Enables and configures Server-Side Rendering. See [SSR Documentation](ssr.md). |

