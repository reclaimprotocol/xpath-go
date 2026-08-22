# xpath-go

**Browser-oriented XPath library for Go with precise source-byte locations**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/reclaimprotocol/xpath-go)](https://goreportcard.com/report/github.com/reclaimprotocol/xpath-go)
[![Release](https://img.shields.io/github/v/release/reclaimprotocol/xpath-go)](https://github.com/reclaimprotocol/xpath-go/releases)

## Features

- **Browser compatibility** - Validated against jsdom across 888 checked-in cases
- **Precise location tracking** - Raw-byte positioning in the original HTML response
- **Response charset decoding** - WHATWG labels with raw-byte location preservation
- **Dual extraction modes** - Extract full elements or content-only with the `ContentsOnly` option
- **Performance-oriented** - Reusable compiled expressions and allocation-aware evaluation
- **Reference-validated** - Go characterization tests plus browser, charset, and conversion oracles
- **Minimal dependencies** - Uses Go's maintained `x/text` encoding tables
- **Developer friendly** - Rich debugging support with trace logging

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/reclaimprotocol/xpath-go"
)

func main() {
    html := `<html><body><div id="content" class="main">Hello World</div></body></html>`
    
    // Simple query
    results, err := xpath.Query("//div[@id='content']", html)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, result := range results {
        fmt.Printf("Found: %s\n", result.TextContent)
        fmt.Printf("Location: %d-%d\n", result.StartLocation, result.EndLocation)
        fmt.Printf("Path: %s\n", result.Path)
    }
}
```

## Installation

```bash
go get github.com/reclaimprotocol/xpath-go
```

## XPath Support

### Axes (12 of 13 XPath 1.0 axes)
- `child::`, `parent::`, `ancestor::`, `descendant::`
- `following::`, `preceding::`, `following-sibling::`, `preceding-sibling::`
- `attribute::`, `self::`
- `descendant-or-self::`, `ancestor-or-self::`

xpath-go supports 12 of the 13 XPath 1.0 axes. The `namespace::` axis and a
namespace-prefix resolver are not currently implemented. See
[`docs/COMPATIBILITY.md`](docs/COMPATIBILITY.md) for the complete supported
subset and known gaps.

### Functions (validated subset)
- **Node Functions**: `text()`, `node()`, `position()`, `last()`, `count()`
- **String Functions**: `string()`, `normalize-space()`, `starts-with()`, `contains()`, `substring()`
- **Boolean Functions**: `boolean()`, `not()`
- **Number Functions**: `number()`, `string-length()`

### Operators (validated subset)
- **Comparison**: `=`, `!=`, `<`, `>`, `<=`, `>=`
- **Logical**: `and`, `or`, `not()`
- **Arithmetic**: `+`, `-`, `*`, `div`, `mod`
- **Union**: `|` (pipe operator)

### Predicates (validated subset)
- Attribute predicates: `[@id='test']`, `[@class and @id]`
- Position predicates: `[1]`, `[last()]`, `[position()>2]`
- Content predicates: `[text()='value']`, `[contains(text(), 'substring')]`
- Complex boolean expressions: `[@id='a' or @class='b'] and [position()=1]`

## Compatibility Summary

**XPath 1.0 subset with browser-oriented HTML recovery**

| Feature Category | Support | Details |
|------------------|---------|---------|
| Basic Selection | Full | Element, attribute, wildcard selection |
| Attribute Queries | Full | Attribute existence, value matching, complex conditions |
| Text Functions | Full | text(), contains(), starts-with(), normalize-space() |
| Position Functions | Full | position(), last(), numeric positions |
| Axes Navigation | 12 of 13 axes | The `namespace::` axis and namespace-prefix resolver are not implemented |
| Complex Predicates | Full | Boolean logic, nested predicates, unions |
| String Functions | Full | substring(), string-length() with edge cases |

## Important Compatibility Notes

### HTML Entity Handling

XPath-Go decodes named and numeric character references in text and attributes,
matching browser DOM behavior. Result locations and full-node `Value` slices
still refer to the original source bytes.

```html
<!-- Source HTML -->
<p>Text with &amp; &lt; &gt; characters</p>
```

```go
source := `<p>Text with &amp; &lt; &gt; characters</p>`
results, _ := xpath.Query("//p[contains(text(), '&')]", source)
fmt.Println(results[0].TextContent) // Text with & < > characters
fmt.Println(results[0].Value)       // original <p>...</p> source
```

See the **[HTML Entity Handling guide](docs/HTML_ENTITY_HANDLING.md)** for details and best practices.

### Unicode Position Tracking

**XPath-Go uses byte-based position tracking** for performance and Go ecosystem compatibility:

```html
<!-- Source HTML -->
<p>Hello 世界</p>
```

**Why byte-based positioning:**
- **Go idiomatic** - Aligns with Go's string handling and byte slice operations
- **Performance** - No Unicode code point counting overhead during parsing
- **Memory efficient** - Direct byte offset calculations
- **Deterministic** - Consistent across all platforms and Go versions

**Working with Unicode positions:**
```go
// Method 1: Use byte positions directly (recommended for Go)
html := `<p>Hello 世界</p>`
results, _ := xpath.Query("//p", html)
content := html[results[0].StartLocation:results[0].EndLocation]

// Method 2: Convert to character positions if needed
import "unicode/utf8"
func ByteToCharPos(s string, bytePos int) int {
    return utf8.RuneCountInString(s[:bytePos])
}
```

### Compatibility Considerations

While XPath-Go aims for high compatibility with web standards, there are some intentional design choices:

- **HTML character references**: XPath sees decoded DOM text; source extraction retains the original bytes
- **Unicode Position Tracking**: Uses byte offsets for Go ecosystem compatibility  
- **Performance Optimizations**: Some complex expressions may have subtle evaluation differences

For complete compatibility details, see [docs/COMPATIBILITY.md](docs/COMPATIBILITY.md).

## Advanced Usage

### Location Tracking

Get precise source-byte positions for matched nodes (location tracking is
enabled by default):

```go
results, _ := xpath.Query("//div[@class='content']", htmlContent)
for _, result := range results {
    fmt.Printf("Element: <%s>\n", result.NodeName)
    fmt.Printf("Text: %s\n", result.TextContent) 
    fmt.Printf("Byte Range: %d-%d\n", result.StartLocation, result.EndLocation)
    fmt.Printf("XPath: %s\n", result.Path)
    fmt.Printf("Attributes: %+v\n", result.Attributes)
}
```

### Compiled XPath (Performance)

For repeated queries, compile once and reuse:

```go
// Compile once
compiled, err := xpath.Compile("//div[@class='item'][position()>1]")
if err != nil {
    log.Fatal(err)
}

// Reuse for multiple documents; benchmark your workload to measure the benefit
for _, htmlDoc := range documents {
    results, err := compiled.Evaluate(htmlDoc)
    if err != nil {
        log.Printf("Error: %v", err)
        continue
    }
    // Process results...
}
```

### Custom Options

Control output format and extraction mode:

```go
results, err := xpath.QueryWithOptions("//p", html, xpath.Options{
    IncludeLocation: true,
    OutputFormat:    "values", // "nodes", "values", "paths"
    ContentsOnly:    false,    // Extract full elements (default)
})

// Extract only inner content between tags
results, err := xpath.QueryWithOptions("//div", html, xpath.Options{
    ContentsOnly: true,  // Extract content-only: <div>content</div> → "content"
})
```

### Debug Tracing

```go
xpath.EnableTrace()
defer xpath.DisableTrace()

results, err := xpath.Query("//div[contains(@class, 'complex')]//p[last()]", html)
// Detailed evaluation steps logged to stderr
```

## Examples

### Basic Selections

```go
// Element selection
xpath.Query("//div", html)                    // All div elements
xpath.Query("/html/body/div", html)           // Specific path
xpath.Query("//div[@id='main']", html)        // Div with specific ID

// Attribute selection  
xpath.Query("//div/@class", html)             // Class attributes
xpath.Query("//*[@href]", html)               // Elements with href
xpath.Query("//a[@href and @title]", html)   // Links with both attributes
```

### Text and Content

```go
// Text content
xpath.Query("//p[text()='Hello']", html)           // Exact text match
xpath.Query("//div[contains(text(), 'world')]", html) // Text contains
xpath.Query("//span[normalize-space(text())='Clean']", html) // Normalized text

// Position-based
xpath.Query("//li[1]", html)                    // First list item
xpath.Query("//tr[last()]", html)               // Last table row  
xpath.Query("//div[position()>2]", html)        // Divs after second
```

### Complex Predicates

```go
// Boolean logic
xpath.Query("//div[@id='a' or @class='b']", html)           // OR condition
xpath.Query("//p[@class and text()]", html)                 // AND condition
xpath.Query("//div[not(@class)]", html)                     // NOT condition

// Nested conditions
xpath.Query("//ul[li[@class='active']]", html)              // UL containing active LI
xpath.Query("//div[@class='container']//p[position()=2]", html) // Second P in container

// Complex expressions
xpath.Query("//article[.//h1 and count(.//p)>2]", html)     // Articles with H1 and 3+ paragraphs
```

### Axes Navigation

```go
// Family relationships
xpath.Query("//h2/following-sibling::p", html)        // P elements after H2
xpath.Query("//span/parent::div[@class='box']", html)  // Parent div with class
xpath.Query("//td/ancestor::table[@id='data']", html)  // Ancestor table with ID

// Advanced navigation
xpath.Query("//div[@id='start']/descendant-or-self::*[@class]", html) // Descendants with class
xpath.Query("//li[3]/preceding-sibling::li", html)                    // Previous siblings
```

### Dual Extraction Modes

Extract either full elements or just their inner content:

```go
html := `<div class="box">Hello <span>World</span>!</div>`

// Full element extraction (default)
results, _ := xpath.QueryWithOptions("//div", html, xpath.Options{
    ContentsOnly: false,
})
// StartLocation/EndLocation: <div class="box">Hello <span>World</span>!</div>

// Content-only extraction  
results, _ := xpath.QueryWithOptions("//div", html, xpath.Options{
    ContentsOnly: true,
})
// StartLocation/EndLocation: Hello <span>World</span>!

// Fine-grained control when IncludeLocation is enabled
fmt.Printf("Full element: %s\n", html[result.StartLocation:result.EndLocation])
fmt.Printf("Inner content: %s\n", html[result.ContentStart:result.ContentEnd])
```

**Use Cases:**
- **Full elements** (`ContentsOnly: false`): HTML processing, DOM manipulation, complete element extraction
- **Content only** (`ContentsOnly: true`): Text processing, content analysis, clean text extraction without tags

## Performance

Designed for production use:

- **Reusable compiled expressions** for repeated queries
- **Byte-oriented source locations** when location metadata is requested
- **Concurrent evaluation** with per-call state

Performance depends on document size, expression shape, output format, and
whether location metadata is requested. Run the checked-in benchmarks for your
workload instead of relying on a fixed speed or allocation claim.

```go
// Compile once, use many times
compiled, _ := xpath.Compile("//div[@class='item'][position()>1]")
results, _ := compiled.Evaluate(html)
```

## API Reference

### Core Functions

```go
// Basic query evaluation
func Query(xpathExpr, content string) ([]Result, error)

// Query with custom options
func QueryWithOptions(xpathExpr, content string, opts Options) ([]Result, error)

// Byte-oriented response-body evaluation
func QueryBytes(xpathExpr string, content []byte) ([]Result, error)
func QueryBytesWithOptions(xpathExpr string, content []byte, opts Options) ([]Result, error)

// Compile XPath for reuse
func Compile(xpathExpr string) (*XPath, error)

// Enable/disable debug tracing
func EnableTrace()
func DisableTrace()
```

### Result Structure

```go
type Result struct {
    Value         string            // Node value or text content
    NodeName      string            // Element name (div, span, etc.)
    NodeType      int               // Node type (1=element, 2=attribute, 3=text)
    Attributes    map[string]string // Element attributes
    StartLocation int               // Raw-byte start position (full element or content-only)
    EndLocation   int               // Raw-byte end position (full element or content-only)
    ContentStart  int               // Start of inner content (after opening tag)
    ContentEnd    int               // End of inner content (before closing tag)
    Path          string            // Generated XPath path
    TextContent   string            // Text content of node and children
}
```

### Options

```go
type Options struct {
    IncludeLocation  bool   // Include source positions; zero-value Options disables them
    OutputFormat     string // "nodes", "values", "paths" (default: "nodes")
    ContentsOnly     bool   // Extract only inner content between tags (default: false)
    ScriptingEnabled bool   // Parse noscript as RAWTEXT
    Charset          string // Explicit WHATWG response charset; empty = tolerant UTF-8
}
```

Charset decoding affects the logical DOM used by XPath, not the source
coordinate system. Text and attributes are decoded Unicode, while result
locations continue to index the original response bytes.

**ContentsOnly Mode:**
- `false` (default): Extract full elements including tags: `<div>content</div>`
- `true`: Extract only inner content: `content`

With location tracking enabled, both modes maintain source-byte positions.
With `ContentsOnly: true`, `StartLocation`/`EndLocation` point to the content
boundaries, while `ContentStart`/`ContentEnd` identify the inner content.
Set `IncludeLocation: false` when coordinates are not needed; location fields
are then unavailable.

`Query`, `QueryBytes`, and `(*XPath).Evaluate` enable location tracking in
their convenience defaults. The explicit `*WithOptions` APIs use the supplied
value as-is, so `Options{}` leaves location fields at zero.

## Development

### Testing

```bash
# Fast Go unit and integration suite
make test

# Full pre-merge verification
make test-all

# Individual slower layers
make test-race
make test-scaling
make test-compat
make test-fuzz
make test-bench
```

See [Testing](docs/TESTING.md) for the purpose and expected use of each layer.

## Contributing

We welcome contributions! 

```bash
# Clone and setup
git clone https://github.com/reclaimprotocol/xpath-go.git
cd xpath-go && go mod download

# Run tests
make test-all
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with high compatibility goals for [jsdom](https://github.com/jsdom/jsdom) and web standards
- Inspired by the [W3C XPath 1.0 Specification](https://www.w3.org/TR/xpath/)
- Thanks to the Go community for excellent tooling and libraries

---

Compatibility claims are bounded by the checked-in Go and browser-oracle test
suites. See [Compatibility](docs/COMPATIBILITY.md) for the validated surface and
known differences.
