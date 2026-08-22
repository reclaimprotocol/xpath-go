# Testing

The repository exposes one testing interface from the project root. Each target
has a distinct purpose, so normal development does not need to run the slowest
checks on every edit.

| Command | Purpose | Typical use |
|---|---|---|
| `make test` | All Go unit and integration tests | Normal edit/test loop |
| `make test-quality` | Formatting, vet, lint when installed, and diff checks | Before committing |
| `make test-race` | Full Go suite with the race detector | Concurrency or state changes |
| `make test-scaling` | Three runs of scaling/performance invariants | Parser/evaluator performance changes |
| `make test-compat` | Complete jsdom, conversion, charset, and edge-case oracles | Behavior or compatibility changes |
| `make test-fuzz` | Unified XPath parser fuzz target | Parser grammar changes |
| `make test-bench` | Checked-in Go benchmarks with allocation reporting | Performance comparison |
| `make test-all` | Quality, unit, race, scaling, and compatibility checks | Pre-merge verification |

`make test-compat` installs Node dependencies with `npm ci` only when they are
missing. The compatibility runner normally writes its JSON report; the wrapper
restores the pre-run file on success or failure so local verification does not
dirty a tracked report.

## Focused Go tests

Use ordinary Go package and `-run` selection when iterating on one area:

```sh
go test ./internal/evaluator -run TestUnion
go test ./pkg/utils -run TestParseTable
go test ./tests/integration -run TestQueryTemplate
```

Scaling tests are separated from benchmarks. Scaling tests contain assertions
about growth and belong in pre-merge verification; benchmarks report machine-
specific measurements and should be compared with `benchstat` on an otherwise
idle machine.

The fuzz duration is caller-controlled:

```sh
FUZZTIME=30s make test-fuzz
```

## CI layers

Pull requests run formatting/vet/lint, the supported Go-version unit matrix, one
race job on the primary Go version, and the browser compatibility oracle.
Benchmarks are not pass/fail correctness gates and therefore remain an explicit
local or scheduled measurement.
