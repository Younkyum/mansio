# Contributing to Mansio

Thanks for taking the time to contribute! This document covers the basics of working on Mansio.

## Code of conduct

Be respectful. Assume good intent. We follow the spirit of the [Contributor Covenant](https://www.contributor-covenant.org/). Harassment, discrimination, or personal attacks aren't tolerated.

## Ways to contribute

- **Bug reports** — open an issue using the bug-report template. Include OS, install mode (native vs. Docker), browser, and steps to reproduce.
- **Feature requests** — open an issue using the feature-request template. Describe the use case before the implementation.
- **Pull requests** — for non-trivial changes, please open an issue first so we can discuss scope.
- **Translations** — the README has Korean and Simplified Chinese versions. Additional locales are welcome (`README.<lang>.md` at the repo root).

## Development setup

Prerequisites: Go 1.22+, Node.js 20+, tmux, git.

```bash
git clone https://github.com/Younkyum/Mansio.git
cd Mansio

# Two terminals
make dev-backend    # Go server on :8080
make dev-frontend   # Vite dev server (proxies API + WebSocket to :8080)
```

Single-binary build:

```bash
make build          # → ./mansio
```

## Tests

Run the full suite before submitting a PR:

```bash
make test           # Go + frontend
make test-go        # Go only
make test-frontend  # Vitest only
```

Add tests for new behavior. Bug fixes should come with a regression test that fails before the fix and passes after.

## Pull request checklist

- [ ] The branch builds (`make build`) and tests pass (`make test`).
- [ ] New behavior has tests (Go or Vitest, whichever applies).
- [ ] Frontend changes were verified in a browser; mobile changes were verified at a narrow viewport.
- [ ] No unrelated formatting churn — keep the diff focused.
- [ ] Commit messages follow the existing style (`feat:`, `fix:`, `docs:`, `chore:` …).

## Reporting security issues

Please **do not** open public issues for security vulnerabilities. See [SECURITY.md](SECURITY.md) for the disclosure process.

## License

By contributing, you agree that your contributions will be licensed under **GPL-3.0-or-later**, the same license as the project. See [LICENSE](LICENSE).
