# Performance Benchmarks

Performance guidance for xpath-go. The repository contains reproducible
benchmarks; it intentionally does not publish fixed timing or allocation
numbers because those vary by Go version, architecture, CPU, input size, and
background load.

## Table of Contents

- [Benchmark Results](#benchmark-results)
- [Performance Tips](#performance-tips)
- [Profiling and Debugging](#profiling-and-debugging)
- [Benchmark Your Own Use Cases](#benchmark-your-own-use-cases)

## Benchmark Results

Run the checked-in suites from the repository root:

```bash
go test -run '^$' -bench . -benchmem ./...
```

`-run '^$'` skips ordinary tests, while `-benchmem` reports allocations. Use
`-count` and `-benchtime` when comparing changes, for example:

```bash
go test -run '^$' -bench 'Benchmark(Query|CompiledEvaluate)$' \
  -benchmem -count=5 .
go test -run '^$' -bench 'BenchmarkHTMLParser' -benchmem -count=5 ./pkg/utils
```

The XPath package currently includes:

- `BenchmarkQuery`: one-shot XPath parsing, HTML parsing, evaluation, and
  public result conversion.
- `BenchmarkCompiledEvaluate`: repeated evaluation after `Compile`.
- `BenchmarkPredicateHeavyEvaluation`: nested predicates and typed
  conversions.
- `BenchmarkResultPathConstruction`: result conversion with many repeated
  sibling names.
- `BenchmarkResultPathScaling`: result-path conversion at 64, 256, and 1024
  selected nodes.

The evaluator package also includes `BenchmarkAttributeNodeMaterialization`,
which compares repeated cached attribute-axis access with fresh materialization.
On the audit machine (Apple M2 Pro, Go benchmark defaults), the cached case
reported 0 B/op and 0 allocs/op; use the benchmark on your target workload
before treating those numbers as portable.

The HTML parser package includes `BenchmarkHTMLParserPlainText`,
`BenchmarkHTMLParserMalformedFormattingRecovery`, and
`BenchmarkHTMLParserTableRecovery`.

The benchmark output from the target machine is the authoritative source for
current timings and allocations. Do not infer a universal speedup from one
machine's output.

### Compiled vs. non-compiled usage

```go
// One-shot: parses the expression on each call.
for i := 0; i < 1000; i++ {
    results, err := xpath.Query("//div[@class='item']", html)
    _ = results
    _ = err
}

// Reusable: compile once, then evaluate many documents.
compiled, err := xpath.Compile("//div[@class='item']")
if err != nil {
    log.Fatal(err)
}
for i := 0; i < 1000; i++ {
    results, err := compiled.Evaluate(html)
    _ = results
    _ = err
}
```

Compilation is intended to validate and retain the expression for reuse. It
does not eliminate HTML parsing or evaluation work. Compare
`BenchmarkQuery` and `BenchmarkCompiledEvaluate` with identical inputs when
quantifying the effect of compilation.

## Performance Tips

### 1. Reuse compiled expressions

```go
compiled, err := xpath.Compile("//div[@class='item']")
if err != nil {
    return err
}
for _, doc := range documents {
    results, err := compiled.Evaluate(doc)
    if err != nil {
        return err
    }
    _ = results
}
```

This avoids reparsing the same expression. The benefit depends on the
expression and the relative cost of parsing versus document processing.

### 2. Prefer selective expressions

Specific paths and predicates can reduce traversal and candidate nodes:

```go
xpath.Query("//div[@id='target']", html)
xpath.Query("//div[@class='container']/span/a", html)
```

Broad searches and deeply nested descendant paths may require more tree
traversal. Measure representative documents before changing selectors solely
for performance.

### 3. Choose the output you need

When callers do not need node metadata, use a value-oriented output format:

```go
results, err := xpath.QueryWithOptions(expr, html, xpath.Options{
    OutputFormat: "values",
})
```

`IncludeLocation: false` is intended to omit source-location metadata. It may
reduce work, but the size of any improvement is workload-dependent and must be
measured. Code that needs `StartLocation`, `EndLocation`, `ContentStart`, or
`ContentEnd` should leave it enabled.

### 4. Reuse parsed documents when the application allows it

If an application evaluates many expressions against the same HTML, avoid
reparsing the source between expressions where the public API or an application
cache makes that possible. Account for document lifetime and memory usage when
doing so.

### 5. Process independent documents concurrently

Compiled expressions can be shared by concurrent evaluations when the API
contract and current implementation guarantee that use. Keep document inputs
and result handling per call, and verify concurrency-sensitive changes with:

```bash
go test -race ./...
```

## Profiling and Debugging

Use Go's benchmark and profiling tools for evidence:

```bash
go test -run '^$' -bench . -benchmem -cpuprofile cpu.prof -memprofile mem.prof .
go tool pprof cpu.prof
go tool pprof mem.prof
```

Profile the smallest representative benchmark first. Look at both CPU samples
and allocation counts; an optimization that improves one can regress the
other.

## Benchmark Your Own Use Cases

Add a benchmark next to the package it exercises:

```go
func BenchmarkYourUseCase(b *testing.B) {
    html := loadTestHTML()
    compiled, err := xpath.Compile("//your/xpath/here")
    if err != nil {
        b.Fatal(err)
    }

    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        results, err := compiled.Evaluate(html)
        if err != nil {
            b.Fatal(err)
        }
        if len(results) == 0 {
            b.Fatal("expected at least one result")
        }
    }
}
```

Keep benchmark inputs stable, report the Go version and machine when sharing
results, and compare multiple runs. Benchmark results are measurements, not
compatibility guarantees; refer to [`COMPATIBILITY.md`](COMPATIBILITY.md) for
the validated XPath and HTML scope.
