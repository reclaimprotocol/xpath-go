# API Reference

Complete API documentation for xpath-go library.

## Table of Contents

- [Core Functions](#core-functions)
- [Types](#types)
- [Options](#options)
- [Error Handling](#error-handling)
- [Advanced Features](#advanced-features)
- [Debugging](#debugging)

## Core Functions

### Query

```go
func Query(xpathExpr, content string) ([]Result, error)
```

Evaluates an XPath expression against HTML/XML content with default options.

**Parameters:**
- `xpathExpr`: XPath expression string (e.g., "//div[@class='content']")
- `content`: HTML/XML content to query against

**Returns:**
- `[]Result`: Slice of matching results with location tracking
- `error`: Error if parsing or evaluation fails

**Example:**
```go
results, err := xpath.Query("//div[@id='main']", htmlContent)
if err != nil {
    log.Fatal(err)
}
for _, result := range results {
    fmt.Printf("Found: %s\n", result.TextContent)
}
```

### QueryWithOptions

```go
func QueryWithOptions(xpathExpr, content string, opts Options) ([]Result, error)
```

Evaluates XPath with custom options for advanced control.

**Parameters:**
- `xpathExpr`: XPath expression string
- `content`: HTML/XML content to query
- `opts`: Options struct for customization

**Returns:**
- `[]Result`: Slice of results
- `error`: Error if operation fails

**Example:**
```go
results, err := xpath.QueryWithOptions("//p", html, xpath.Options{
    IncludeLocation: false,
    OutputFormat:    "values",
})
```

### Compile

```go
func Compile(xpathExpr string) (*XPath, error)
```

Pre-compiles an XPath expression for repeated use (performance optimization).

**Parameters:**
- `xpathExpr`: XPath expression to compile

**Returns:**
- `*XPath`: Compiled XPath object
- `error`: Error if compilation fails

**Example:**
```go
compiled, err := xpath.Compile("//div[@class='item']")
if err != nil {
    log.Fatal(err)
}

// Reuse compiled expression
for _, doc := range documents {
    results, err := compiled.Evaluate(doc)
    // Process results...
}
```

## Types

### Result

Represents a single XPath query result with complete metadata.

```go
type Result struct {
    Value         string            `json:"value"`
    NodeName      string            `json:"nodeName"`
    NodeType      int               `json:"nodeType"`
    Attributes    map[string]string `json:"attributes,omitempty"`
    StartLocation int               `json:"startLocation"`
    EndLocation   int               `json:"endLocation"`
    Path          string            `json:"path"`
    TextContent   string            `json:"textContent"`
}
```

**Fields:**
- `Value`: Primary value of the node (varies by OutputFormat)
- `NodeName`: HTML/XML element name (e.g., "div", "span", "a")
- `NodeType`: Node type constant (1=element, 2=attribute, 3=text)
- `Attributes`: Map of element attributes
- `StartLocation`: Character position where element starts in source
- `EndLocation`: Character position where element ends in source
- `Path`: Generated XPath path to this node
- `TextContent`: Combined text content of node and all children

**Node Types:**
- `1`: Element node (div, span, etc.)
- `2`: Attribute node (@class, @id, etc.)
- `3`: Text node (text content)

### XPath

Compiled XPath expression for repeated evaluation.

```go
type XPath struct {
    // Private fields
}
```

**Methods:**

#### Evaluate

```go
func (x *XPath) Evaluate(content string) ([]Result, error)
```

Evaluates the compiled XPath against new content.

**Parameters:**
- `content`: HTML/XML content string

**Returns:**
- `[]Result`: Matching results
- `error`: Evaluation error

#### GetExpression

```go
func (x *XPath) GetExpression() string
```

Returns the original XPath expression string.

**Returns:**
- `string`: Original XPath expression

### Options

Configuration options for XPath evaluation.

```go
type Options struct {
    IncludeLocation bool   `json:"include_location"`
    OutputFormat    string `json:"output_format"`
}
```

**Fields:**
- `IncludeLocation`: Include character position tracking (default: true)
- `OutputFormat`: Result format - "nodes", "values", or "paths" (default: "nodes")

**Output Formats:**
- `"nodes"`: Full node information with metadata (default)
- `"values"`: Only text content/values
- `"paths"`: Only XPath paths to matched nodes

## Error Handling

The library provides detailed error information for different failure scenarios.

### Common Errors

```go
// Empty expression
results, err := xpath.Query("", html)
// Error: "xpath expression cannot be empty"

// Empty content
results, err := xpath.Query("//div", "")
// Error: "content cannot be empty"

// Invalid XPath syntax
results, err := xpath.Query("//div[", html)
// Error: "invalid xpath syntax: unclosed predicate"

// Compilation error
compiled, err := xpath.Compile("invalid//xpath//")
// Error: detailed parsing error with position
```

### Error Types

All errors implement the standard Go `error` interface and provide descriptive messages suitable for logging or user display.

## Advanced Features

### Version Information

```go
// Get library version
version := xpath.Version        // "1.0.0"
apiVersion := xpath.APIVersion  // "v1"

// Get build information
buildInfo := xpath.GetBuildInfo()
fmt.Printf("Version: %s\n", buildInfo.Version)
fmt.Printf("Go Version: %s\n", buildInfo.GoVersion)
fmt.Printf("Platform: %s\n", buildInfo.Platform)
```

### Compatibility Checking

```go
// Check API compatibility
isCompat := xpath.IsCompatible("v1")     // true
isCompat = xpath.IsCompatible("v2")      // false

// Check Go version compatibility
err := xpath.CheckGoVersion()
if err != nil {
    log.Fatal("Go version not supported:", err)
}
```

## Debugging

### Trace Logging

Enable detailed trace logging to understand XPath evaluation steps.

```go
// Enable tracing (outputs to stderr)
xpath.EnableTrace()

// Your XPath operations will now show detailed logs
results, err := xpath.Query("//div[contains(@class, 'complex')]", html)

// Disable tracing
xpath.DisableTrace()
```

**Trace Output Example:**
```
[XPATH-TRACE] Parsing expression: //div[contains(@class, 'complex')]
[XPATH-TRACE] Step 1: axis=descendant-or-self, nodetest=div
[XPATH-TRACE] Predicate: contains(@class, 'complex')
[XPATH-TRACE] Evaluating contains() function
[XPATH-TRACE] Found 3 matching nodes
```

### Performance Monitoring

```go
import "time"

start := time.Now()
results, err := xpath.Query(complexExpression, largeHTML)
duration := time.Since(start)

fmt.Printf("Query took %v, found %d results\n", duration, len(results))
```

## Thread Safety

All functions in the xpath-go library are **thread-safe** and can be used concurrently:

```go
var wg sync.WaitGroup

// Concurrent queries are safe
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        results, err := xpath.Query("//div", htmlContent)
        fmt.Printf("Goroutine %d found %d results\n", id, len(results))
    }(i)
}

wg.Wait()
```

## Best Practices

### Performance Optimization

1. **Use Compile() for repeated queries:**
   ```go
   // Good: Compile once, use many times
   compiled, _ := xpath.Compile("//div[@class='item']")
   for _, doc := range manyDocuments {
       results, _ := compiled.Evaluate(doc)
   }
   
   // Avoid: Parsing same expression repeatedly
   for _, doc := range manyDocuments {
       results, _ := xpath.Query("//div[@class='item']", doc)
   }
   ```

2. **Disable location tracking for performance-critical code:**
   ```go
   results, err := xpath.QueryWithOptions(expr, html, xpath.Options{
       IncludeLocation: false,  // Faster processing
   })
   ```

3. **Use specific XPath expressions:**
   ```go
   // Good: Specific path
   xpath.Query("/html/body/div[@id='main']", html)
   
   // Less efficient: Broad search
   xpath.Query("//*[@id='main']", html)
   ```

### Error Handling

```go
results, err := xpath.Query(userInput, htmlContent)
if err != nil {
    // Log the error with context
    log.Printf("XPath evaluation failed: expression=%q, error=%v", userInput, err)
    
    // Return user-friendly error
    return fmt.Errorf("invalid search expression: %w", err)
}
```

### Memory Management

The library automatically manages memory and doesn't require explicit cleanup. However, for long-running applications:

```go
// For very large documents, consider processing in chunks
const maxDocSize = 10 * 1024 * 1024 // 10MB
if len(htmlContent) > maxDocSize {
    // Split or stream processing
}
```