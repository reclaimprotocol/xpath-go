# Contributing to xpath-go

Thanks for contributing. By participating, you agree to follow the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Before opening an issue

- **Security vulnerabilities:** follow the private reporting guidance in
  [SECURITY.md](SECURITY.md). Do not open a public issue.
- **Usage questions and expected behavior:** first consult the
  [README](README.md), [API reference](docs/API.md), and
  [compatibility scope](docs/COMPATIBILITY.md).
- **Bugs and feature requests:** use the matching GitHub issue form so that
  maintainers receive the reproduction details and motivation needed to triage
  the request.

## Development setup

The module declares Go 1.21. The browser-compatibility checks also require
Node.js 20 or later.

```sh
git clone https://github.com/reclaimprotocol/xpath-go.git
cd xpath-go
go mod download
make test
```

Read [docs/TESTING.md](docs/TESTING.md) before making a change. In particular,
run the narrowest relevant test while iterating, then run `make test-all`
before requesting review. Compatibility, race, scaling, fuzz, and benchmark
checks are separate because they have different runtime and machine
requirements.

## Pull requests

Keep pull requests focused and describe:

- the problem being solved and any behavior change;
- the tests run, including skipped checks and why;
- documentation or compatibility-scope changes; and
- any backwards-compatibility, performance, or security considerations.

Add or update tests for changed behavior. Keep public documentation accurate;
update `CHANGELOG.md` for user-visible changes that should appear in the next
release. Do not commit generated build output, local benchmark profiles, or
dependency directories.

## Commit messages

This repository follows the organization’s Conventional Commit convention:

```text
type(optional-scope): concise imperative summary
```

Typical types are `feat`, `fix`, `docs`, `test`, `perf`, `refactor`, and
`chore`. Use the body to explain context when the change is not self-evident,
and use a `BREAKING CHANGE:` footer for incompatible changes.

## Maintainer review

Maintainers may request changes for correctness, compatibility, API design,
test coverage, documentation, release impact, or project scope. A pull request
being open does not guarantee it will be merged or released.
