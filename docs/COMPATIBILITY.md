# XPath-Go compatibility status

## Scope

XPath-Go targets practical XPath 1.0 evaluation over browser-recovered HTML while
preserving byte ranges into the original UTF-8 input. Compatibility claims in
this document are limited to the checked-in Go tests and the jsdom comparator
corpus; they are not a claim of complete XPath 1.0 or HTML Living Standard
conformance.

## Implemented compatibility batches

### XPath evaluation

- Added typed XPath values for node sets, strings, numbers, and booleans, with
  XPath conversion, comparison, predicate, arithmetic, `NaN`, and document-order
  behavior. This includes existential node-set comparisons and first-node
  conversion of node sets and unions.
- Completed `following::` and `preceding::` traversal, including reverse-axis
  predicate positions, attribute contexts, de-duplication, and stable document
  ordering.
- Added comment and processing-instruction node tests and exposed document type,
  namespace URI, local name, and prefix metadata where applicable.
- Corrected union identity and ordering for attributes and mixed node kinds,
  including namespaced SVG and MathML attributes.
- Expanded function and expression handling used by the corpus, including
  `string`, `number`, `boolean`, `true`, `false`, `concat`, `name`,
  `local-name`, `namespace-uri`, string functions, positional functions, and
  arithmetic/boolean precedence.

### Browser-style HTML parsing

- Added browser-compatible document skeleton construction and EOF recovery for
  implicit and explicit `html`, `head`, and `body` trees.
- Expanded tokenizer recovery for malformed start/end tags, comments, doctypes,
  CDATA-like markup, processing instructions, NUL/CRLF input, adjacent and
  duplicate attributes, and named/numeric character references.
- Completed covered text-state behavior for RCDATA, raw text, escaped and
  double-escaped script data, plaintext, and the initial newline rule for `pre`,
  `listing`, and `textarea`.
- Added covered tree-builder behavior for paragraphs, lists, headings, buttons,
  forms, formatting reconstruction and adoption-agency cases, tables and foster
  parenting, select modes, hidden inputs, and template content fragments.
- Added covered SVG and MathML foreign-content parsing, integration points,
  breakout behavior, adjusted element/attribute names, and namespace metadata.
- Preserved source fidelity through recovery: public locations remain byte
  offsets into the original UTF-8 input, including reconstructed or synthetic
  browser trees.

### Validation infrastructure

- Expanded the browser comparator from 120 to 878 cases (37 original and 841
  extended), covering parser recovery, XPath behavior, namespaces, node kinds,
  locations, and full-element/content-only output.
- Added an independent 18-case jsdom oracle for typed XPath conversions and
  comparisons.
- Updated comparator normalization to translate jsdom's UTF-16 locations to
  UTF-8 byte offsets and to compare namespace/local-name/prefix identity.
- The comparator now builds its Go driver once per run and cleans it up on exit,
  avoiding per-case compilation noise during the expanded suite.

## Validation evidence

Fresh validation was run on **2026-08-14** from the repository root unless a
command says otherwise.

| Check | Result |
|---|---|
| Legacy tracked tests | PASS: all 18 top-level tests present at `HEAD` (8 public API and 10 parser tests, including their subtests) passed against the accumulated implementation |
| `go test -count=1 ./...` | PASS: all packages |
| `go test -race -count=1 ./...` | PASS: all packages on the final rerun; see timing note below |
| `npm test` in `tests/` | PASS: 878/878 matched jsdom; 37/37 original and 841/841 extended |
| `npm run test:typed-conversions` in `tests/` | PASS: 18/18 browser-oracle cases |
| `go vet ./...` | PASS |
| `golangci-lint run` | PASS: 0 issues (golangci-lint 2.3.1) |
| `gofmt -l` over all Go files | PASS: no files reported |
| `git diff --check` | PASS |

The final full race-instrumented rerun passed. During the independent audit, an
earlier race run reported no data race but one performance-ratio assertion
(`TestParseTableAdoptionScaling`) measured 10.0x for a 4x input and failed its
timing threshold. That test passed three consecutive non-race runs, and the race
suite excluding only that timing assertion also passed. This is recorded as an
instrumentation-sensitive benchmark flake, not a functional or data-race
failure.

The generated report is
[`tests/comprehensive_compatibility_report.json`](../tests/comprehensive_compatibility_report.json).

## Intentional boundaries and deferred gaps

- **Corpus-bounded claim:** 100% means 878/878 in the checked-in comparator. New
  browser fixtures can still expose unsupported recovery or XPath behavior.
- **XPath 1.0 surface not yet complete:** variable references, a namespace
  resolver/namespace axis, and unimplemented core functions such as `id`,
  `lang`, `sum`, `substring-before`, `substring-after`, `translate`, `floor`,
  `ceiling`, and `round` remain outside the validated subset.
- **XPath 2.0 and later are not targets:** sequences, richer type systems,
  FLWOR/conditional expressions, and XPath 2.0+ functions are deferred.
- **HTML rather than general XML:** parsing and recovery intentionally follow an
  HTML/browser model; this is not a validating or namespace-aware general XML
  parser.
- **Scripting-enabled parsing is excluded from this compatibility claim:** the
  public/default parser behavior is scripting-disabled. The comparator is not
  used as an authoritative oracle for scripting-sensitive `noscript` cases,
  and no further scripting-enabled support is planned in this compatibility
  effort.
- **Locations are UTF-8 byte offsets by design:** callers expecting JavaScript
  UTF-16 code-unit positions must convert them. Synthetic nodes may have no
  direct source range.

These boundaries should be treated as scope constraints when adding fixtures or
evaluating production expressions; passing the current suite does not override
them.
