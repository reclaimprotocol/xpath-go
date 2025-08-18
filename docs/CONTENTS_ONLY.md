# ContentsOnly Extraction Mode

Complete guide to the `contentsOnly` option for dual-mode element extraction in xpath-go.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [API Reference](#api-reference)
- [Examples](#examples)
- [Use Cases](#use-cases)
- [Performance Considerations](#performance-considerations)
- [Edge Cases](#edge-cases)
- [Migration Guide](#migration-guide)

## Overview

The `contentsOnly` option provides dual extraction modes for XPath queries:

- **Full Element Mode** (`contentsOnly: false`, default): Extract complete HTML elements including tags
- **Content-Only Mode** (`contentsOnly: true`): Extract only the inner content between opening and closing tags

Both modes maintain precise position tracking and provide fine-grained control over extraction boundaries.

### Visual Example

```html
<div class="box" id="main">Hello <span>World</span>!</div>
```

| Mode | ContentsOnly | Result | StartLocation/EndLocation |
|------|--------------|--------|---------------------------|
| Full Element | `false` | `<div class="box" id="main">Hello <span>World</span>!</div>` | Full element boundaries |
| Content-Only | `true` | `Hello <span>World</span>!` | Content boundaries only |

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/reclaimprotocol/xpath-go"
)

func main() {
    html := `<article><h1>Title</h1><p>Content here</p></article>`
    
    // Full element extraction (default)
    results, _ := xpath.QueryWithOptions("//p", html, xpath.Options{
        ContentsOnly: false,
    })
    fmt.Println("Full:", html[results[0].StartLocation:results[0].EndLocation])
    // Output: <p>Content here</p>
    
    // Content-only extraction
    results, _ = xpath.QueryWithOptions("//p", html, xpath.Options{
        ContentsOnly: true,
    })
    fmt.Println("Content:", html[results[0].StartLocation:results[0].EndLocation])
    // Output: Content here
}
```

## How It Works

### Position Mapping

The `contentsOnly` option controls how `StartLocation` and `EndLocation` are mapped:

```go
type Result struct {
    StartLocation int // Mapped based on ContentsOnly option
    EndLocation   int // Mapped based on ContentsOnly option
    ContentStart  int // Always points to content start (after opening tag)
    ContentEnd    int // Always points to content end (before closing tag)
}
```

**Default Mode** (`contentsOnly: false`):
```go
StartLocation = node.StartPos     // Start of opening tag
EndLocation   = node.EndPos       // End of closing tag
```

**Content-Only Mode** (`contentsOnly: true`):
```go
StartLocation = node.ContentStart // After opening tag
EndLocation   = node.ContentEnd   // Before closing tag
```

### HTML Parser Integration

The HTML parser tracks four position boundaries for each element:

1. **StartPos**: Beginning of opening tag (`<div>`)
2. **ContentStart**: After opening tag ends (`<div>|content`)
3. **ContentEnd**: Before closing tag begins (`content|</div>`)
4. **EndPos**: End of closing tag (`</div>`)

## API Reference

### Options.ContentsOnly

```go
type Options struct {
    IncludeLocation bool
    OutputFormat    string
    ContentsOnly    bool  // New field
}
```

**Type**: `bool`  
**Default**: `false`  
**Description**: Controls extraction mode for StartLocation/EndLocation mapping

### Result Fields

All Result structs include both element and content position information:

```go
type Result struct {
    // ... other fields
    StartLocation int `json:"startLocation"`     // Mode-dependent
    EndLocation   int `json:"endLocation"`       // Mode-dependent  
    ContentStart  int `json:"contentStart"`      // Always available
    ContentEnd    int `json:"contentEnd"`        // Always available
}
```

## Examples

### Basic Usage

```go
html := `<div id="test">Hello <em>World</em>!</div>`

// Query with different modes
fullResults, _ := xpath.QueryWithOptions("//div", html, xpath.Options{
    ContentsOnly: false,
})

contentResults, _ := xpath.QueryWithOptions("//div", html, xpath.Options{
    ContentsOnly: true,
})

// Compare results
fmt.Printf("Full element: '%s'\n", 
    html[fullResults[0].StartLocation:fullResults[0].EndLocation])
// Output: <div id="test">Hello <em>World</em>!</div>

fmt.Printf("Content only: '%s'\n", 
    html[contentResults[0].StartLocation:contentResults[0].EndLocation])
// Output: Hello <em>World</em>!
```

### Multiple Elements

```go
html := `<ul>
    <li>First item</li>
    <li>Second item</li>
    <li>Third item</li>
</ul>`

// Extract all list items (content-only)
results, _ := xpath.QueryWithOptions("//li", html, xpath.Options{
    ContentsOnly: true,
})

for i, result := range results {
    content := html[result.StartLocation:result.EndLocation]
    fmt.Printf("Item %d: '%s'\n", i+1, content)
}
// Output:
// Item 1: 'First item'
// Item 2: 'Second item'  
// Item 3: 'Third item'
```

### Nested Elements

```go
html := `<article>
    <header>
        <h1>Main Title</h1>
        <p class="subtitle">Subtitle text</p>
    </header>
    <section>
        <p>Paragraph with <strong>bold</strong> text.</p>
    </section>
</article>`

// Extract nested content
results, _ := xpath.QueryWithOptions("//header//p", html, xpath.Options{
    ContentsOnly: true,
})

fmt.Printf("Subtitle: '%s'\n", 
    html[results[0].StartLocation:results[0].EndLocation])
// Output: Subtitle text
```

### Fine-Grained Control

```go
html := `<div class="box">Content with <span>nested</span> elements</div>`

results, _ := xpath.Query("//div", html)
result := results[0]

// All position information available
fmt.Printf("Full element: '%s'\n", 
    html[result.StartLocation:result.EndLocation])
fmt.Printf("Content only: '%s'\n", 
    html[result.ContentStart:result.ContentEnd])

// Same as:
contentResults, _ := xpath.QueryWithOptions("//div", html, xpath.Options{
    ContentsOnly: true,
})
fmt.Printf("Content (via option): '%s'\n", 
    html[contentResults[0].StartLocation:contentResults[0].EndLocation])
```

### Self-Closing Tags

```go
html := `<div>Before <img src="test.jpg" alt="Test"/> After</div>`

results, _ := xpath.QueryWithOptions("//img", html, xpath.Options{
    ContentsOnly: true,
})

// Self-closing tags have ContentStart == ContentEnd
result := results[0]
fmt.Printf("ContentStart: %d, ContentEnd: %d\n", 
    result.ContentStart, result.ContentEnd)
// Output: ContentStart: 34, ContentEnd: 34 (no inner content)
```

### Raw Text Elements

```go
html := `<script type="text/javascript">
function test() {
    console.log('Hello World');
}
</script>`

results, _ := xpath.QueryWithOptions("//script", html, xpath.Options{
    ContentsOnly: true,
})

content := html[results[0].StartLocation:results[0].EndLocation]
fmt.Printf("Script content:\n%s\n", content)
// Output: Raw JavaScript code without <script> tags
```

## Use Cases

### 1. HTML Processing and DOM Manipulation

**Full Element Mode** is ideal when you need complete HTML elements:

```go
// Extract complete components for HTML manipulation
results, _ := xpath.QueryWithOptions("//div[@class='component']", html, xpath.Options{
    ContentsOnly: false,
})

for _, result := range results {
    fullHTML := html[result.StartLocation:result.EndLocation]
    // Process complete HTML element
    processHTMLComponent(fullHTML)
}
```

### 2. Text Processing and Content Analysis

**Content-Only Mode** is perfect for text analysis without HTML noise:

```go
// Extract clean text for NLP processing
results, _ := xpath.QueryWithOptions("//p | //h1 | //h2", html, xpath.Options{
    ContentsOnly: true,
})

var textContent []string
for _, result := range results {
    content := html[result.StartLocation:result.EndLocation]
    textContent = append(textContent, content)
}

// Process clean text (no HTML tags)
analyzeSentiment(textContent)
```

### 3. Search Indexing

```go
// Build search index from content without HTML markup
results, _ := xpath.QueryWithOptions("//article//p", html, xpath.Options{
    ContentsOnly: true,
    OutputFormat: "values",
})

for _, result := range results {
    indexContent(result.Value) // Clean text without HTML
}
```

### 4. Data Extraction and Web Scraping

```go
// Extract structured data
type ArticleData struct {
    Title   string
    Content string
    Tags    []string
}

// Extract title (content-only)
titleResults, _ := xpath.QueryWithOptions("//h1", html, xpath.Options{
    ContentsOnly: true,
})

// Extract content (content-only) 
contentResults, _ := xpath.QueryWithOptions("//div[@class='content']//p", html, xpath.Options{
    ContentsOnly: true,
})

article := ArticleData{
    Title:   html[titleResults[0].StartLocation:titleResults[0].EndLocation],
    Content: html[contentResults[0].StartLocation:contentResults[0].EndLocation],
}
```

### 5. Template Processing

```go
// Extract template placeholders
results, _ := xpath.QueryWithOptions("//span[@class='placeholder']", html, xpath.Options{
    ContentsOnly: true,
})

for _, result := range results {
    placeholder := html[result.StartLocation:result.EndLocation]
    // Process template variable: {{variable_name}}
    processTemplate(placeholder)
}
```

## Performance Considerations

### Memory Usage

Both extraction modes use the same amount of memory - all position boundaries are always tracked. The `contentsOnly` option only affects which boundaries are mapped to `StartLocation`/`EndLocation`.

### Processing Speed

- **No significant performance difference** between modes
- Position mapping is a simple integer assignment
- HTML parsing time remains the same
- Consider disabling `IncludeLocation` for performance-critical scenarios where positions aren't needed

### Compilation Benefits

```go
// Compile once for repeated use with different modes
compiled, _ := xpath.Compile("//div[@class='item']")

// Use with different extraction modes
fullResults, _ := compiled.EvaluateWithOptions(html, xpath.Options{
    ContentsOnly: false,
})

contentResults, _ := compiled.EvaluateWithOptions(html, xpath.Options{
    ContentsOnly: true,
})
```

## Edge Cases

### Empty Elements

```go
html := `<div></div><p></p><span>   </span>`

results, _ := xpath.QueryWithOptions("//div", html, xpath.Options{
    ContentsOnly: true,
})

// Empty elements have StartLocation == EndLocation
result := results[0]
fmt.Printf("Empty content length: %d\n", 
    result.EndLocation - result.StartLocation)
// Output: 0
```

### Whitespace-Only Content

```go
html := `<p>   
    
</p>`

results, _ := xpath.QueryWithOptions("//p", html, xpath.Options{
    ContentsOnly: true,
})

content := html[results[0].StartLocation:results[0].EndLocation]
fmt.Printf("Whitespace content: '%s'\n", content)
// Output: Whitespace preserved exactly as in source
```

### Deeply Nested Elements

```go
html := `<div><span><em><strong>Deep content</strong></em></span></div>`

// Extract content at different nesting levels
levels := []string{"//div", "//span", "//em", "//strong"}

for _, xpath := range levels {
    results, _ := xpath.QueryWithOptions(xpath, html, xpath.Options{
        ContentsOnly: true,
    })
    content := html[results[0].StartLocation:results[0].EndLocation]
    fmt.Printf("%s: '%s'\n", xpath, content)
}
// Output:
// //div: 'Deep content' (with nested tags)
// //span: 'Deep content' (with nested tags)  
// //em: 'Deep content' (with nested tags)
// //strong: 'Deep content' (clean text)
```

### Mixed Content Types

```go
html := `<div>
    Text before
    <span>Nested element</span>
    Text after
    <br/>
    Final text
</div>`

results, _ := xpath.QueryWithOptions("//div", html, xpath.Options{
    ContentsOnly: true,
})

content := html[results[0].StartLocation:results[0].EndLocation]
// Content includes text nodes, elements, and self-closing tags
```

## Migration Guide

### From Full Element Extraction

If you're currently using the default behavior and want to extract content-only:

**Before:**
```go
results, _ := xpath.Query("//p", html)
// Results include full <p>...</p> elements
```

**After:**
```go
results, _ := xpath.QueryWithOptions("//p", html, xpath.Options{
    ContentsOnly: true,
})
// Results include only inner content
```

### Gradual Migration

You can migrate incrementally by using both position sets:

```go
results, _ := xpath.Query("//div", html) // Default behavior

for _, result := range results {
    // Old way: full element
    fullElement := html[result.StartLocation:result.EndLocation]
    
    // New way: content only  
    contentOnly := html[result.ContentStart:result.ContentEnd]
    
    // Choose based on your needs
    processContent(contentOnly) // Migrate to this
}
```

### Backward Compatibility

The `contentsOnly` option is fully backward compatible:

- Default value is `false` (preserves existing behavior)
- All existing code continues to work unchanged
- New fields (`ContentStart`/`ContentEnd`) are always available
- No breaking changes to existing APIs

## Best Practices

### 1. Choose the Right Mode for Your Use Case

- **HTML manipulation**: Use `ContentsOnly: false`
- **Text analysis**: Use `ContentsOnly: true`  
- **Data extraction**: Usually `ContentsOnly: true`
- **Template processing**: Usually `ContentsOnly: true`

### 2. Combine with Output Formats

```go
// Get clean text values for content analysis
results, _ := xpath.QueryWithOptions("//p", html, xpath.Options{
    ContentsOnly: true,
    OutputFormat: "values",
})
// result.Value contains clean text content
```

### 3. Use Fine-Grained Control When Needed

```go
results, _ := xpath.Query("//div", html)
for _, result := range results {
    // Always available regardless of ContentsOnly setting
    if needsFullElement {
        content = html[result.StartLocation:result.EndLocation]
    } else {
        content = html[result.ContentStart:result.ContentEnd]  
    }
}
```

### 4. Performance Optimization

```go
// Disable position tracking for performance-critical code
results, _ := xpath.QueryWithOptions("//p", html, xpath.Options{
    IncludeLocation: false,  // Faster processing
    ContentsOnly: true,      // Still controls Value extraction
})
```

---

**Next Steps:**
- See [API.md](API.md) for complete API reference
- Check [EXAMPLES.md](EXAMPLES.md) for more usage examples
- Review [COMPATIBILITY.md](COMPATIBILITY.md) for browser compatibility notes