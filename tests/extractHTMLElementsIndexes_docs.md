# extractHTMLElementsIndexes Function Documentation

## Overview

The `extractHTMLElementsIndexes` function extracts the byte/character offsets of HTML elements matched by an XPath expression. It supports two extraction modes: full element extraction and content-only extraction.

## Function Signature

```javascript
function extractHTMLElementsIndexes(
    html: string,
    xpathExpression: string,
    contentsOnly: boolean
): { start: number, end: number }[]
```

## Parameters

### `html: string`
The HTML document as a string to search within.

### `xpathExpression: string`
The XPath expression used to select elements from the HTML document.

### `contentsOnly: boolean`
Controls the extraction behavior:
- `false`: Returns offsets for the complete element including opening and closing tags
- `true`: Returns offsets for only the inner content between the tags

## Return Value

Returns an array of objects with `start` and `end` properties representing byte offsets in the original HTML string.

```typescript
Array<{ start: number, end: number }>
```

## Behavior Details

### When `contentsOnly = false` (Full Element Mode)
- Returns the complete element including opening tag, content, and closing tag
- Uses `nodeLocation.startOffset` and `nodeLocation.endOffset`
- Useful for element replacement or full element extraction

**Example:**
```html
HTML: <div class="content">Hello World</div>
XPath: //div
Result: { start: 0, end: 39 } → '<div class="content">Hello World</div>'
```

### When `contentsOnly = true` (Content Only Mode)
- Returns only the inner content between opening and closing tags
- Uses `nodeLocation.startTag.endOffset` and `nodeLocation.endTag.startOffset`
- Useful for text extraction or content replacement

**Example:**
```html
HTML: <div class="content">Hello World</div>
XPath: //div
Result: { start: 20, end: 31 } → 'Hello World'
```

## Special Cases

### Empty Elements
```html
HTML: <div></div>
contentsOnly: true → { start: 5, end: 5 } → '' (empty string)
contentsOnly: false → { start: 0, end: 11 } → '<div></div>'
```

### Self-Closing Elements
```html
HTML: <img src="test.jpg" />
contentsOnly: true → { start: 20, end: 20 } → '' (empty string)
contentsOnly: false → { start: 0, end: 20 } → '<img src="test.jpg" />'
```

### Nested Elements
```html
HTML: <div>Outer <span>Inner</span> Text</div>
XPath: //div
contentsOnly: true → 'Outer <span>Inner</span> Text'
contentsOnly: false → '<div>Outer <span>Inner</span> Text</div>'
```

## Error Handling

The function throws errors in the following cases:

### No Elements Found
```javascript
throw new Error(`Failed to find XPath: "${xpathExpression}"`)
```

### Node Location Not Available
```javascript
throw new Error(`Failed to find XPath node location: "${xpathExpression}"`)
```

## Use Cases

### Content-Only Mode (`contentsOnly: true`)
- **Text extraction**: Getting the actual text content of elements
- **Content replacement**: Replacing only the inner content while preserving tags
- **Text analysis**: Analyzing the textual content without markup
- **Template systems**: Extracting content for template processing

### Full Element Mode (`contentsOnly: false`)
- **Element replacement**: Replacing entire elements including tags
- **HTML manipulation**: Moving or duplicating complete elements
- **Markup analysis**: Analyzing both structure and content
- **Element extraction**: Extracting complete HTML fragments

## Implementation Notes

The function uses JSDOM with `includeNodeLocations: true` to get precise byte offsets. The location tracking provides:

- `nodeLocation.startOffset`: Start of the opening tag
- `nodeLocation.endOffset`: End of the closing tag
- `nodeLocation.startTag.endOffset`: End of the opening tag (after '>')
- `nodeLocation.endTag.startOffset`: Start of the closing tag (before '</')

## Dependencies

- **jsdom**: For HTML parsing and XPath evaluation
- **Node.js**: Runtime environment

## Compatibility

This implementation is designed to be compatible with similar functionality in other languages/platforms, particularly Go implementations of XPath processors.

## Performance Considerations

- JSDOM parsing has overhead for large HTML documents
- XPath evaluation performance depends on expression complexity
- Memory usage scales with document size and number of matched elements

## Example Usage

```javascript
const { JSDOM } = require('jsdom');

// Basic usage - extract full elements
const fullElements = extractHTMLElementsIndexes(
    '<html><body><p>Hello</p><p>World</p></body></html>',
    '//p',
    false
);
// Returns: [{ start: 12, end: 24 }, { start: 24, end: 36 }]

// Content-only usage - extract just text
const contentOnly = extractHTMLElementsIndexes(
    '<html><body><p>Hello</p><p>World</p></body></html>',
    '//p', 
    true
);
// Returns: [{ start: 15, end: 20 }, { start: 27, end: 32 }]

// Extract content from original HTML
const html = '<html><body><p>Hello</p><p>World</p></body></html>';
const results = extractHTMLElementsIndexes(html, '//p', true);
results.forEach((result, i) => {
    const content = html.substring(result.start, result.end);
    console.log(`Element ${i}: "${content}"`);
});
// Output: 
// Element 0: "Hello"
// Element 1: "World"
```