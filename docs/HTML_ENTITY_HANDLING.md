# HTML Entity Handling in XPath-Go

## Overview

XPath-Go preserves HTML entities in their original encoded form, which differs from JavaScript's DOM behavior that automatically decodes entities. This document explains this design choice and provides guidance for working with HTML entities.

## Behavior Comparison

### JavaScript DOM Behavior
```html
<!-- Source HTML -->
<p>Text with &amp; &lt; &gt; &quot; characters</p>
```

```javascript
// JavaScript automatically decodes entities
element.textContent; // "Text with & < > " characters"

// XPath query matches decoded content
document.evaluate('//p[contains(text(), "&")]', document, null, 0, null);
// ✅ Matches: finds the <p> element
```

### XPath-Go Behavior
```html
<!-- Source HTML -->
<p>Text with &amp; &lt; &gt; &quot; characters</p>
```

```go
// XPath-Go preserves original entity encoding
results, _ := xpath.Query("//p", html)
results[0].TextContent // "Text with &amp; &lt; &gt; &quot; characters"

// XPath query works with encoded content
xpath.Query("//p[contains(text(), '&')]", html)
// ❌ No match: looks for "&" but content has "&amp;"

xpath.Query("//p[contains(text(), '&amp;')]", html)  
// ✅ Matches: finds the encoded entity
```

## Why XPath-Go Preserves Entities

### 1. Fidelity to Source Content
XPath-Go maintains exactly what was written in the HTML source:
```html
<p>Use &amp;amp; to display &amp; in HTML</p>
```

- **JavaScript**: `"Use &amp; to display & in HTML"` (information loss!)
- **XPath-Go**: `"Use &amp;amp; to display &amp; in HTML"` (preserves intent)

### 2. Predictable Behavior
What you see in the HTML source is exactly what you get in text content:
```html
<div title="&quot;Hello&quot;">&lt;script&gt;alert()&lt;/script&gt;</div>
```

- **JavaScript**: Attribute = `"Hello"`, Text = `<script>alert()</script>` (potential security issues)
- **XPath-Go**: Attribute = `&quot;Hello&quot;`, Text = `&lt;script&gt;alert()&lt;/script&gt;` (safe and predictable)

### 3. No Information Loss
You can always decode entities when needed, but you can't recover the original encoding:
```go
// You can decode when needed
decoded := html.UnescapeString(textContent)

// But you can't go back from decoded to original encoding
// without losing information about what was originally encoded
```

### 4. Consistency Across Languages
Many HTML parsers in other languages also preserve entities:
- Python's `lxml`: Preserves entities by default
- Java's JSoup: Configurable, can preserve entities
- XPath-Go: Preserves entities (consistent with this approach)

## Working with HTML Entities

### Method 1: Query with Encoded Entities
```go
// Query for the encoded form
results, _ := xpath.Query("//p[contains(text(), '&amp;')]", html)
```

### Method 2: Decode After Extraction
```go
import "html"

results, _ := xpath.Query("//p", html)
for _, result := range results {
    decoded := html.UnescapeString(result.TextContent)
    if strings.Contains(decoded, "&") {
        // Process the decoded content
    }
}
```

### Method 3: Pre-process HTML (Advanced)
```go
import "html"

// Decode entities in HTML before parsing (use with caution)
decodedHTML := html.UnescapeString(originalHTML)
results, _ := xpath.Query("//p[contains(text(), '&')]", decodedHTML)
```

⚠️ **Warning**: Pre-processing can cause issues with:
- Nested entities (`&amp;amp;` becomes `&amp;` not `&`)
- Malformed HTML structure
- Security implications

## Common HTML Entities Reference

| Entity | Character | XPath-Go Text Content | JavaScript textContent |
|--------|-----------|----------------------|----------------------|
| `&amp;` | `&` | `&amp;` | `&` |
| `&lt;` | `<` | `&lt;` | `<` |
| `&gt;` | `>` | `&gt;` | `>` |
| `&quot;` | `"` | `&quot;` | `"` |
| `&apos;` | `'` | `&apos;` | `'` |
| `&#39;` | `'` | `&#39;` | `'` |
| `&nbsp;` | ` ` (non-breaking space) | `&nbsp;` | ` ` |

## Best Practices

### 1. Be Explicit in Queries
```go
// ✅ Good: Explicit about entity encoding
xpath.Query("//p[contains(text(), '&amp;')]", html)

// ❌ Avoid: Assuming entities are decoded
xpath.Query("//p[contains(text(), '&')]", html)
```

### 2. Normalize When Comparing
```go
func normalizeText(text string) string {
    return html.UnescapeString(text)
}

// Compare normalized versions
if normalizeText(result.TextContent) == normalizeText(expectedText) {
    // Match found
}
```

### 3. Document Entity Expectations
```go
// Document whether your functions expect encoded or decoded content
func FindElementByText(xpath, text string, encoded bool) []*types.Node {
    if !encoded {
        text = html.EscapeString(text)
    }
    query := fmt.Sprintf("//element[contains(text(), '%s')]", text)
    results, _ := xpath.Query(query, html)
    return results
}
```

## Testing Considerations

When writing tests that involve HTML entities:

```go
func TestHTMLEntities(t *testing.T) {
    html := `<p>Text with &amp; &lt; &gt; characters</p>`
    
    // Test with encoded entities (XPath-Go behavior)
    results, _ := xpath.Query("//p[contains(text(), '&amp;')]", html)
    assert.Len(t, results, 1)
    
    // Test decoded comparison
    decoded := html.UnescapeString(results[0].TextContent)
    assert.Contains(t, decoded, "&")
}
```

## Migration from JavaScript XPath

If migrating from JavaScript XPath code:

```javascript
// JavaScript code
document.evaluate('//p[contains(text(), "&")]', document, null, 0, null)
```

```go
// XPath-Go equivalent
xpath.Query("//p[contains(text(), '&amp;')]", html)

// Or with decoding
results, _ := xpath.Query("//p", html)
for _, result := range results {
    if strings.Contains(html.UnescapeString(result.TextContent), "&") {
        // Found match
    }
}
```

## Summary

XPath-Go's entity preservation is a **feature, not a bug**. It provides:
- ✅ **Fidelity**: Preserves original HTML content exactly
- ✅ **Predictability**: Consistent behavior across all content
- ✅ **Flexibility**: You can decode when needed
- ✅ **Security**: Prevents entity-related parsing issues

When working with XPath-Go, always consider whether your content contains HTML entities and adjust your queries accordingly.