# Contributing to goinertia

First off, thank you for considering contributing to `goinertia`! It's people like you that make goinertia such a great tool.

## Getting Started

1.  Fork the repository on GitHub.
2.  Clone your fork locally.
3.  Create a new branch for your feature or bug fix.

## Development Workflow

We use a `Makefile` to simplify common development tasks. Here are the most useful commands:

### Prerequisites

Ensure you have the following installed:
- [Go](https://go.dev/) (latest version recommended)
- [golangci-lint](https://golangci-lint.run/)
- [gofumpt](https://github.com/mvdan/gofumpt)
- [gci](https://github.com/daixiang0/gci)

### Commands

- **Run all checks (generate, format, lint, test)**:
  ```bash
  make check
  ```
  This is the default goal. Run this before submitting a PR to ensure everything is in order.

- **Run tests**:
  ```bash
  make test
  ```

- **Run tests with race detector**:
  ```bash
  make test-race
  ```

- **Format code**:
  ```bash
  make fmt
  ```
  This will apply `go fmt`, `gofumpt`, and `gci` to format imports and code style.

- **Lint code**:
  ```bash
  make lint
  ```
  Uses `golangci-lint` to check for code quality issues.

- **Run the basic example**:
  ```bash
  make run-example-base
  ```
  Or with a custom port:
  ```bash
  make run-example-base PORT=8383
  ```

## Pull Requests

1.  Ensure your code passes all checks (`make` or `make check`).
2.  Commit your changes with clear, descriptive messages. We prefer [Conventional Commits](https://www.conventionalcommits.org/).
3.  Push your branch to your fork.
4.  Submit a pull request to the `master` branch.

## License

By contributing, you agree that your contributions will be licensed under its MIT License.
