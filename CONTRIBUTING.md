# Contributing to Orbit

Thank you for taking the time to contribute! Orbit is an open project and all contributions are welcome — bug reports, feature requests, documentation improvements, and code.

## Getting Started

1. **Fork** the repository on GitHub
2. **Clone** your fork: `git clone https://github.com/<your-username>/orbit.git`
3. **Create a branch**: `git checkout -b feat/my-feature`
4. Make your changes
5. **Run checks**: `make check` (fmt + vet + lint + tests)
6. **Commit** with a descriptive message (see below)
7. **Push** and open a **Pull Request**

## Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short description>

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`

Examples:

```
feat(deploy): add blue-green deploy strategy
fix(docker): handle container rename race condition
docs(readme): add remote nodes section
```

## Code Style

- Run `gofmt -s -w .` before committing (or use `make fmt`)
- All public functions must have godoc comments
- Errors must be wrapped with context: `fmt.Errorf("operation: %w", err)`
- Use the `errs` package for user-facing errors that need remediation hints
- Avoid global state — pass dependencies explicitly

## Testing

- Unit tests live alongside the code in `_test.go` files
- Integration tests (requiring Docker) are tagged `//go:build integration`
- Run unit tests: `make test-short`
- Run all tests (requires Docker): `make test`

## Reporting Issues

Please use the GitHub issue templates:

- 🐛 **Bug report** — include OS, Go version, Docker version, and steps to reproduce
- 💡 **Feature request** — describe the use case, not just the solution

## Code of Conduct

Be kind, be respectful. See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
