# XPath Examples

Comprehensive examples demonstrating all features of xpath-go library.

## Table of Contents

- [Basic Examples](#basic-examples)
- [Advanced Selections](#advanced-selections)
- [Real-World Use Cases](#real-world-use-cases)
- [Performance Examples](#performance-examples)
- [Error Handling Examples](#error-handling-examples)
- [Location Tracking Examples](#location-tracking-examples)
- [ContentsOnly Examples](#contentsonly-examples)

## Basic Examples

### Simple Element Selection

```go
package main

import (
    "fmt"
    "log"
    "github.com/reclaimprotocol/xpath-go"
)

func main() {
    html := `
    <html>
    <body>
        <div id="header">Header Content</div>
        <div class="content">Main Content</div>
        <div class="sidebar">Sidebar Content</div>
    </body>
    </html>`

    // Select all div elements
    results, err := xpath.Query("//div", html)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d div elements:\n", len(results))
    for i, result := range results {
        fmt.Printf("  [%d] %s: %s\n", i+1, result.NodeName, result.TextContent)
    }
}
```

### Attribute-Based Selection

```go
func attributeExamples() {
    html := `
    <div id="main" class="container active">
        <p class="text primary">Paragraph 1</p>
        <p class="text">Paragraph 2</p>
        <a href="/home" title="Home Page">Home</a>
        <a href="/about">About</a>
    </div>`

    examples := []string{
        "//div[@id='main']",           // Element with specific ID
        "//p[@class='text primary']", // Element with specific class
        "//a[@href]",                  // Elements with href attribute
        "//a[@href and @title]",      // Elements with both attributes
        "//*[@class]",                 // Any element with class attribute
    }

    for _, expr := range examples {
        results, _ := xpath.Query(expr, html)
        fmt.Printf("%-30s -> %d results\n", expr, len(results))
    }
}
```

## Advanced Selections

### Complex Predicates and Boolean Logic

```go
func complexPredicates() {
    html := `
    <article>
        <header>
            <h1 class="title">Article Title</h1>
            <div class="meta">
                <span class="author">John Doe</span>
                <span class="date">2024-01-15</span>
            </div>
        </header>
        <section class="content">
            <p class="intro">Introduction paragraph.</p>
            <p>Regular paragraph 1.</p>
            <p>Regular paragraph 2.</p>
            <p class="highlight">Important paragraph.</p>
        </section>
        <footer>
            <div class="tags">
                <span class="tag active">Tech</span>
                <span class="tag">Programming</span>
                <span class="tag active">Go</span>
            </div>
        </footer>
    </article>`

    // Boolean OR conditions
    results, _ := xpath.Query("//span[@class='author' or @class='date']", html)
    fmt.Printf("Author OR Date spans: %d\n", len(results))

    // Boolean AND conditions
    results, _ = xpath.Query("//span[@class and contains(@class, 'tag')]", html)
    fmt.Printf("Tagged spans: %d\n", len(results))

    // NOT conditions
    results, _ = xpath.Query("//p[not(@class)]", html)
    fmt.Printf("Paragraphs without class: %d\n", len(results))

    // Complex nested conditions
    results, _ = xpath.Query("//section[p[@class='intro'] and p[@class='highlight']]", html)
    fmt.Printf("Sections with intro AND highlight: %d\n", len(results))

    // Multiple conditions with position
    results, _ = xpath.Query("//p[@class='intro' or position()>2]", html)
    fmt.Printf("Intro OR paragraphs after 2nd: %d\n", len(results))
}
```

### Axes Navigation

```go
func axesNavigation() {
    html := `
    <div class="container">
        <header>
            <h1>Title</h1>
            <nav>
                <a href="/">Home</a>
                <a href="/about">About</a>
            </nav>
        </header>
        <main>
            <article>
                <h2>Article Title</h2>
                <p>First paragraph</p>
                <p>Second paragraph</p>
            </article>
            <aside>
                <h3>Related</h3>
                <ul>
                    <li>Item 1</li>
                    <li>Item 2</li>
                </ul>
            </aside>
        </main>
        <footer>
            <p>Copyright 2024</p>
        </footer>
    </div>`

    examples := map[string]string{
        // Parent navigation
        "//p/parent::article":                    "Parent article of paragraphs",
        "//li/ancestor::main":                    "Main ancestor of list items",
        
        // Sibling navigation
        "//h2/following-sibling::p":              "Paragraphs after h2",
        "//aside/preceding-sibling::article":     "Articles before aside",
        
        // Descendant navigation
        "//header/descendant::a":                 "Links inside header",
        "//main/descendant-or-self::*[@class]":  "Elements with class in main",
        
        // Self axis
        "//article/self::article":               "Article selecting itself",
    }

    for expr, desc := range examples {
        results, _ := xpath.Query(expr, html)
        fmt.Printf("%-45s: %d results - %s\n", expr, len(results), desc)
    }
}
```

### Position and Counting Functions

```go
func positionAndCounting() {
    html := `
    <div class="lists">
        <ul class="nav">
            <li>Home</li>
            <li>About</li>
            <li>Contact</li>
        </ul>
        <ol class="steps">
            <li>Step 1</li>
            <li>Step 2</li>
            <li>Step 3</li>
            <li>Step 4</li>
        </ol>
        <ul class="tags">
            <li>Tag A</li>
            <li>Tag B</li>
        </ul>
    </div>`

    examples := []struct {
        xpath string
        desc  string
    }{
        {"//li[1]", "First list item in each list"},
        {"//li[last()]", "Last list item in each list"},
        {"//li[position()=2]", "Second list item in each list"},
        {"//li[position()>2]", "List items after the second"},
        {"//li[position() mod 2 = 0]", "Even-positioned list items"},
        {"//ul[count(li)=3]", "Lists with exactly 3 items"},
        {"//ul[count(li)>2]", "Lists with more than 2 items"},
        {"(//li)[3]", "Third list item globally"},
        {"(//li)[last()]", "Last list item globally"},
    }

    for _, example := range examples {
        results, _ := xpath.Query(example.xpath, html)
        fmt.Printf("%-30s: %d results - %s\n", example.xpath, len(results), example.desc)
    }
}
```

## Real-World Use Cases

### Web Scraping Example

```go
func webScrapingExample() {
    // Simulate a product page HTML
    html := `
    <html>
    <body>
        <div class="product-page">
            <div class="breadcrumb">
                <a href="/">Home</a> > 
                <a href="/electronics">Electronics</a> > 
                <span>Laptop</span>
            </div>
            <div class="product-info">
                <h1 class="product-title">Gaming Laptop Pro</h1>
                <div class="price">
                    <span class="current-price">$1,299.99</span>
                    <span class="original-price">$1,499.99</span>
                </div>
                <div class="rating">
                    <span class="stars">★★★★☆</span>
                    <span class="review-count">(247 reviews)</span>
                </div>
                <div class="specifications">
                    <div class="spec">
                        <span class="label">CPU:</span>
                        <span class="value">Intel i7-12700H</span>
                    </div>
                    <div class="spec">
                        <span class="label">RAM:</span>
                        <span class="value">32GB DDR4</span>
                    </div>
                    <div class="spec">
                        <span class="label">Storage:</span>
                        <span class="value">1TB NVMe SSD</span>
                    </div>
                </div>
                <div class="availability">
                    <span class="status in-stock">In Stock</span>
                </div>
            </div>
        </div>
    </body>
    </html>`

    // Extract product information
    type Product struct {
        Title         string
        CurrentPrice  string
        OriginalPrice string
        Rating        string
        ReviewCount   string
        Specifications map[string]string
        InStock       bool
    }

    product := Product{
        Specifications: make(map[string]string),
    }

    // Extract title
    if results, err := xpath.Query("//h1[@class='product-title']", html); err == nil && len(results) > 0 {
        product.Title = results[0].TextContent
    }

    // Extract prices
    if results, err := xpath.Query("//span[@class='current-price']", html); err == nil && len(results) > 0 {
        product.CurrentPrice = results[0].TextContent
    }
    if results, err := xpath.Query("//span[@class='original-price']", html); err == nil && len(results) > 0 {
        product.OriginalPrice = results[0].TextContent
    }

    // Extract rating and reviews
    if results, err := xpath.Query("//span[@class='stars']", html); err == nil && len(results) > 0 {
        product.Rating = results[0].TextContent
    }
    if results, err := xpath.Query("//span[@class='review-count']", html); err == nil && len(results) > 0 {
        product.ReviewCount = results[0].TextContent
    }

    // Extract specifications
    if labels, err := xpath.Query("//div[@class='spec']/span[@class='label']", html); err == nil {
        if values, err := xpath.Query("//div[@class='spec']/span[@class='value']", html); err == nil {
            for i := 0; i < len(labels) && i < len(values); i++ {
                key := strings.TrimSuffix(labels[i].TextContent, ":")
                product.Specifications[key] = values[i].TextContent
            }
        }
    }

    // Check availability
    if results, err := xpath.Query("//span[contains(@class, 'in-stock')]", html); err == nil {
        product.InStock = len(results) > 0
    }

    // Display extracted data
    fmt.Printf("Product: %s\n", product.Title)
    fmt.Printf("Price: %s (was %s)\n", product.CurrentPrice, product.OriginalPrice)
    fmt.Printf("Rating: %s %s\n", product.Rating, product.ReviewCount)
    fmt.Printf("In Stock: %v\n", product.InStock)
    fmt.Println("Specifications:")
    for key, value := range product.Specifications {
        fmt.Printf("  %s: %s\n", key, value)
    }
}
```

### Form Validation Example

```go
func formValidationExample() {
    html := `
    <form id="registration" class="user-form">
        <div class="field">
            <label for="username">Username:</label>
            <input type="text" id="username" name="username" required>
            <span class="error" style="display:none">Username is required</span>
        </div>
        <div class="field">
            <label for="email">Email:</label>
            <input type="email" id="email" name="email" required>
            <span class="error" style="display:none">Valid email is required</span>
        </div>
        <div class="field">
            <label for="password">Password:</label>
            <input type="password" id="password" name="password" required minlength="8">
            <span class="error" style="display:none">Password must be at least 8 characters</span>
        </div>
        <div class="field">
            <label for="confirm">Confirm Password:</label>
            <input type="password" id="confirm" name="confirm" required>
            <span class="error" style="display:none">Passwords must match</span>
        </div>
        <div class="field">
            <input type="checkbox" id="terms" name="terms" required>
            <label for="terms">I agree to the terms and conditions</label>
            <span class="error" style="display:none">You must accept the terms</span>
        </div>
        <button type="submit">Register</button>
    </form>`

    // Validate form structure
    checks := []struct {
        xpath string
        desc  string
    }{
        {"//form[@id='registration']", "Form with registration ID exists"},
        {"//input[@required]", "Required input fields"},
        {"//input[@type='email']", "Email input field"},
        {"//input[@type='password']", "Password input fields"},
        {"//input[@type='checkbox']", "Checkbox fields"},
        {"//label[@for]", "Labels with 'for' attributes"},
        {"//span[@class='error']", "Error message containers"},
        {"//button[@type='submit']", "Submit button"},
    }

    fmt.Println("Form Validation Results:")
    for _, check := range checks {
        results, _ := xpath.Query(check.xpath, html)
        status := "✅"
        if len(results) == 0 {
            status = "❌"
        }
        fmt.Printf("%s %-40s: %d found - %s\n", status, check.xpath, len(results), check.desc)
    }

    // Check accessibility features
    fmt.Println("\nAccessibility Checks:")
    accessibilityChecks := []struct {
        xpath string
        desc  string
    }{
        {"//input[@required and not(@aria-required)]", "Required fields missing aria-required"},
        {"//input[not(@id) and following-sibling::label]", "Inputs without IDs but with labels"},
        {"//label[not(@for)]", "Labels without 'for' attributes"},
    }

    for _, check := range accessibilityChecks {
        results, _ := xpath.Query(check.xpath, html)
        status := "✅"
        if len(results) > 0 {
            status = "⚠️"
        }
        fmt.Printf("%s %-50s: %d issues - %s\n", status, check.xpath, len(results), check.desc)
    }
}
```

## Performance Examples

### Compiled XPath for High Performance

```go
func performanceExample() {
    // Large HTML document simulation
    largeHTML := generateLargeHTML(10000) // 10k elements

    // Method 1: Regular queries (slower for repeated use)
    start := time.Now()
    for i := 0; i < 1000; i++ {
        xpath.Query("//div[@class='item']", largeHTML)
    }
    regularTime := time.Since(start)

    // Method 2: Compiled XPath (faster for repeated use)
    compiled, _ := xpath.Compile("//div[@class='item']")
    start = time.Now()
    for i := 0; i < 1000; i++ {
        compiled.Evaluate(largeHTML)
    }
    compiledTime := time.Since(start)

    fmt.Printf("Regular queries: %v\n", regularTime)
    fmt.Printf("Compiled queries: %v\n", compiledTime)
    fmt.Printf("Speedup: %.2fx\n", float64(regularTime.Nanoseconds())/float64(compiledTime.Nanoseconds()))
}

func generateLargeHTML(numElements int) string {
    var html strings.Builder
    html.WriteString("<html><body>")
    
    for i := 0; i < numElements; i++ {
        html.WriteString(fmt.Sprintf(`<div class="item" id="item-%d">Content %d</div>`, i, i))
    }
    
    html.WriteString("</body></html>")
    return html.String()
}
```

### Concurrent Processing

```go
func concurrentExample() {
    documents := []string{
        generateDocument("doc1"),
        generateDocument("doc2"),
        generateDocument("doc3"),
        // ... more documents
    }

    // Compile XPath once
    compiled, _ := xpath.Compile("//div[@class='important']")

    // Process documents concurrently
    var wg sync.WaitGroup
    results := make([][]xpath.Result, len(documents))

    for i, doc := range documents {
        wg.Add(1)
        go func(index int, document string) {
            defer wg.Done()
            
            docResults, err := compiled.Evaluate(document)
            if err != nil {
                log.Printf("Error processing document %d: %v", index, err)
                return
            }
            
            results[index] = docResults
            fmt.Printf("Document %d: found %d results\n", index, len(docResults))
        }(i, doc)
    }

    wg.Wait()
    
    // Aggregate results
    totalResults := 0
    for _, docResults := range results {
        totalResults += len(docResults)
    }
    
    fmt.Printf("Total results across all documents: %d\n", totalResults)
}
```

## Error Handling Examples

### Robust Error Handling

```go
func robustErrorHandling() {
    testCases := []struct {
        xpath   string
        html    string
        desc    string
    }{
        {"", "<div>test</div>", "Empty XPath"},
        {"//div", "", "Empty HTML"},
        {"//div[", "<div>test</div>", "Malformed XPath"},
        {"//div[@id='test'", "<div>test</div>", "Unclosed attribute"},
        {"//div[@id='nonexistent']", "<div id='test'>content</div>", "Valid XPath, no matches"},
        {"//div[@id='test']", "<div id='test'>content</div>", "Valid case"},
    }

    for i, tc := range testCases {
        fmt.Printf("\n[%d] Testing: %s\n", i+1, tc.desc)
        fmt.Printf("    XPath: %q\n", tc.xpath)
        
        results, err := xpath.Query(tc.xpath, tc.html)
        
        if err != nil {
            fmt.Printf("    ❌ Error: %v\n", err)
            
            // Handle specific error types
            switch {
            case strings.Contains(err.Error(), "empty"):
                fmt.Printf("    💡 Suggestion: Provide non-empty input\n")
            case strings.Contains(err.Error(), "syntax"):
                fmt.Printf("    💡 Suggestion: Check XPath syntax\n")
            default:
                fmt.Printf("    💡 Suggestion: Review input format\n")
            }
        } else {
            fmt.Printf("    ✅ Success: Found %d results\n", len(results))
            for j, result := range results {
                fmt.Printf("       [%d] %s: %q\n", j+1, result.NodeName, result.TextContent)
            }
        }
    }
}
```

## Location Tracking Examples

### Precise Character Positioning

```go
func locationTrackingExample() {
    html := `<html>
<head><title>Test Page</title></head>
<body>
    <div id="header" class="main">
        <h1>Welcome</h1>
        <nav>
            <a href="/home">Home</a>
            <a href="/about">About</a>
        </nav>
    </div>
    <div id="content">
        <p>This is a paragraph with <strong>bold text</strong> inside.</p>
        <p>Another paragraph here.</p>
    </div>
</body>
</html>`

    results, _ := xpath.Query("//div[@id]", html)
    
    fmt.Println("Location Tracking Results:")
    fmt.Println("=" + strings.Repeat("=", 70))
    
    for i, result := range results {
        fmt.Printf("[%d] Element: <%s id='%s'>\n", 
            i+1, result.NodeName, result.Attributes["id"])
        fmt.Printf("    Text Content: %q\n", 
            truncateString(result.TextContent, 50))
        fmt.Printf("    Character Range: %d-%d\n", 
            result.StartLocation, result.EndLocation)
        fmt.Printf("    XPath: %s\n", result.Path)
        
        // Extract the actual HTML segment
        if result.StartLocation < len(html) && result.EndLocation <= len(html) {
            segment := html[result.StartLocation:result.EndLocation]
            fmt.Printf("    HTML Segment: %q\n", truncateString(segment, 60))
        }
        fmt.Println()
    }
}

func truncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen-3] + "..."
}
```

### Building a Source Code Highlighter

```go
func sourceHighlighterExample() {
    html := `<div class="code-block">
    <pre><code>
function greet(name) {
    console.log("Hello, " + name + "!");
}
    </code></pre>
</div>`

    // Find code blocks with location information
    results, _ := xpath.Query("//code", html)
    
    for _, result := range results {
        fmt.Printf("Found code block at position %d-%d:\n", 
            result.StartLocation, result.EndLocation)
        
        // You could use this information to:
        // 1. Apply syntax highlighting
        // 2. Add line numbers
        // 3. Create clickable regions
        // 4. Generate source maps
        
        codeContent := strings.TrimSpace(result.TextContent)
        lines := strings.Split(codeContent, "\n")
        
        fmt.Println("Code content with line numbers:")
        for i, line := range lines {
            fmt.Printf("%3d: %s\n", i+1, line)
        }
    }
}
```

## ContentsOnly Examples

### Basic Content Extraction

```go
func contentsOnlyBasic() {
    html := `<div class="container">
        <h1>Page Title</h1>
        <p>This is a <strong>paragraph</strong> with nested elements.</p>
        <div class="highlight">Important content here</div>
    </div>`

    // Full element extraction (default)
    fmt.Println("=== Full Element Mode ===")
    results, _ := xpath.QueryWithOptions("//p", html, xpath.Options{
        ContentsOnly: false,
    })
    
    for _, result := range results {
        content := html[result.StartLocation:result.EndLocation]
        fmt.Printf("Full element: %s\n", content)
        // Output: <p>This is a <strong>paragraph</strong> with nested elements.</p>
    }

    // Content-only extraction
    fmt.Println("\n=== Content-Only Mode ===")
    results, _ = xpath.QueryWithOptions("//p", html, xpath.Options{
        ContentsOnly: true,
    })
    
    for _, result := range results {
        content := html[result.StartLocation:result.EndLocation]
        fmt.Printf("Content only: %s\n", content)
        // Output: This is a <strong>paragraph</strong> with nested elements.
    }

    // Fine-grained control (always available)
    fmt.Println("\n=== Fine-Grained Control ===")
    results, _ = xpath.Query("//div[@class='highlight']", html)
    for _, result := range results {
        fullElement := html[result.StartLocation:result.EndLocation]
        innerContent := html[result.ContentStart:result.ContentEnd]
        fmt.Printf("Full element: %s\n", fullElement)
        fmt.Printf("Inner content: %s\n", innerContent)
    }
}
```

### Text Processing Use Case

```go
func textProcessingExample() {
    // Blog post HTML
    html := `<article class="blog-post">
        <header>
            <h1>Understanding XPath in Go</h1>
            <div class="meta">
                <span class="author">By John Doe</span>
                <span class="date">January 15, 2024</span>
            </div>
        </header>
        <div class="content">
            <p>XPath is a powerful query language for <strong>XML and HTML</strong> documents.</p>
            <p>With xpath-go, you can <em>easily extract</em> data from web pages.</p>
            <h2>Key Features</h2>
            <ul>
                <li>High compatibility with web standards</li>
                <li>Precise location tracking</li>
                <li>Dual extraction modes</li>
            </ul>
            <blockquote>
                "XPath makes web scraping much easier and more reliable."
            </blockquote>
        </div>
    </article>`

    // Extract clean text for content analysis
    fmt.Println("=== Blog Post Text Analysis ===")
    
    // Extract title (content-only)
    titleResults, _ := xpath.QueryWithOptions("//h1", html, xpath.Options{
        ContentsOnly: true,
    })
    title := html[titleResults[0].StartLocation:titleResults[0].EndLocation]
    
    // Extract paragraphs (content-only) 
    paragraphResults, _ := xpath.QueryWithOptions("//p", html, xpath.Options{
        ContentsOnly: true,
    })
    
    // Extract list items (content-only)
    listResults, _ := xpath.QueryWithOptions("//li", html, xpath.Options{
        ContentsOnly: true,
    })
    
    // Extract quotes (content-only)
    quoteResults, _ := xpath.QueryWithOptions("//blockquote", html, xpath.Options{
        ContentsOnly: true,
    })

    // Display extracted content
    fmt.Printf("Title: %s\n", title)
    fmt.Printf("Word count: %d\n", len(strings.Fields(title)))
    
    fmt.Println("\nParagraphs:")
    for i, result := range paragraphResults {
        content := html[result.StartLocation:result.EndLocation]
        fmt.Printf("  [%d] %s (words: %d)\n", i+1, content, len(strings.Fields(content)))
    }
    
    fmt.Println("\nKey Points:")
    for i, result := range listResults {
        content := html[result.StartLocation:result.EndLocation]
        fmt.Printf("  • %s\n", content)
    }
    
    fmt.Println("\nQuotes:")
    for _, result := range quoteResults {
        content := strings.TrimSpace(html[result.StartLocation:result.EndLocation])
        fmt.Printf("  \"%s\"\n", content)
    }
    
    // Calculate total word count
    allText := []string{title}
    for _, result := range paragraphResults {
        allText = append(allText, html[result.StartLocation:result.EndLocation])
    }
    
    totalWords := 0
    for _, text := range allText {
        totalWords += len(strings.Fields(text))
    }
    fmt.Printf("\nTotal word count: %d\n", totalWords)
}
```

### Data Extraction Comparison

```go
func dataExtractionComparison() {
    // E-commerce product HTML
    html := `<div class="product-card">
        <div class="product-image">
            <img src="laptop.jpg" alt="Gaming Laptop">
        </div>
        <div class="product-details">
            <h3 class="product-name">Gaming Laptop Pro Max</h3>
            <div class="price-info">
                <span class="current-price">$1,299.99</span>
                <span class="original-price">$1,599.99</span>
                <span class="discount">19% off</span>
            </div>
            <div class="specifications">
                <div class="spec-item">
                    <span class="spec-label">CPU:</span>
                    <span class="spec-value">Intel Core i7-12700H</span>
                </div>
                <div class="spec-item">
                    <span class="spec-label">RAM:</span>
                    <span class="spec-value">32GB DDR4</span>
                </div>
                <div class="spec-item">
                    <span class="spec-label">Storage:</span>
                    <span class="spec-value">1TB NVMe SSD</span>
                </div>
            </div>
        </div>
    </div>`

    fmt.Println("=== Data Extraction: Full vs Content-Only ===")
    
    // Compare extraction modes for product specifications
    fmt.Println("\n--- Specification Labels ---")
    
    // Full element mode
    fullResults, _ := xpath.QueryWithOptions("//span[@class='spec-label']", html, xpath.Options{
        ContentsOnly: false,
    })
    fmt.Println("Full element mode:")
    for _, result := range fullResults {
        content := html[result.StartLocation:result.EndLocation]
        fmt.Printf("  %s\n", content)
    }
    
    // Content-only mode  
    contentResults, _ := xpath.QueryWithOptions("//span[@class='spec-label']", html, xpath.Options{
        ContentsOnly: true,
    })
    fmt.Println("\nContent-only mode:")
    for _, result := range contentResults {
        content := html[result.StartLocation:result.EndLocation]
        fmt.Printf("  %s\n", content)
    }
    
    // Practical application: Build structured data
    fmt.Println("\n--- Building Structured Data ---")
    
    type ProductSpec struct {
        Label string
        Value string
    }
    
    var specifications []ProductSpec
    
    // Extract labels and values (content-only for clean data)
    labelResults, _ := xpath.QueryWithOptions("//span[@class='spec-label']", html, xpath.Options{
        ContentsOnly: true,
    })
    valueResults, _ := xpath.QueryWithOptions("//span[@class='spec-value']", html, xpath.Options{
        ContentsOnly: true,
    })
    
    for i := 0; i < len(labelResults) && i < len(valueResults); i++ {
        label := strings.TrimSuffix(html[labelResults[i].StartLocation:labelResults[i].EndLocation], ":")
        value := html[valueResults[i].StartLocation:valueResults[i].EndLocation]
        
        specifications = append(specifications, ProductSpec{
            Label: label,
            Value: value,
        })
    }
    
    fmt.Println("Structured specifications:")
    for _, spec := range specifications {
        fmt.Printf("  %s: %s\n", spec.Label, spec.Value)
    }
}
```

### HTML Processing vs Text Processing

```go
func htmlVsTextProcessing() {
    html := `<div class="email-template">
        <div class="header">
            <h1>Welcome to <span class="brand">TechCorp</span>!</h1>
        </div>
        <div class="body">
            <p>Dear <span class="placeholder">{{NAME}}</span>,</p>
            <p>Thank you for joining our <strong>premium</strong> service.</p>
            <p>Your account benefits include:</p>
            <ul>
                <li>24/7 support</li>
                <li>Priority access to new features</li>
                <li>Advanced analytics dashboard</li>
            </ul>
            <div class="cta">
                <a href="{{LOGIN_URL}}" class="button">Get Started</a>
            </div>
        </div>
        <div class="footer">
            <p>Best regards,<br>The TechCorp Team</p>
        </div>
    </div>`

    fmt.Println("=== HTML Processing Use Case ===")
    // Extract complete HTML structure for email template processing
    templateResults, _ := xpath.QueryWithOptions("//div[@class='email-template']", html, xpath.Options{
        ContentsOnly: false,
    })
    
    if len(templateResults) > 0 {
        fullTemplate := html[templateResults[0].StartLocation:templateResults[0].EndLocation]
        fmt.Println("Complete HTML template for processing:")
        fmt.Printf("Length: %d characters\n", len(fullTemplate))
        fmt.Printf("Contains placeholders: %t\n", strings.Contains(fullTemplate, "{{"))
        // This full HTML can be processed for template substitution, styling, etc.
    }
    
    fmt.Println("\n=== Text Processing Use Case ===")
    // Extract clean text content for analysis
    paragraphs, _ := xpath.QueryWithOptions("//p", html, xpath.Options{
        ContentsOnly: true,
    })
    
    listItems, _ := xpath.QueryWithOptions("//li", html, xpath.Options{
        ContentsOnly: true,
    })
    
    fmt.Println("Clean text content for analysis:")
    
    fmt.Println("\nParagraphs:")
    for i, result := range paragraphs {
        content := html[result.StartLocation:result.EndLocation]
        fmt.Printf("  [%d] %s\n", i+1, content)
    }
    
    fmt.Println("\nBenefit list:")
    for i, result := range listItems {
        content := html[result.StartLocation:result.EndLocation]
        fmt.Printf("  • %s\n", content)
    }
    
    // Text analysis example
    fmt.Println("\n--- Text Analysis ---")
    allTextContent := []string{}
    
    for _, result := range paragraphs {
        content := html[result.StartLocation:result.EndLocation]
        allTextContent = append(allTextContent, content)
    }
    for _, result := range listItems {
        content := html[result.StartLocation:result.EndLocation]  
        allTextContent = append(allTextContent, content)
    }
    
    combinedText := strings.Join(allTextContent, " ")
    wordCount := len(strings.Fields(combinedText))
    placeholderCount := strings.Count(combinedText, "{{")
    
    fmt.Printf("Total words: %d\n", wordCount)
    fmt.Printf("Placeholders found: %d\n", placeholderCount)
    fmt.Printf("Average words per section: %.1f\n", float64(wordCount)/float64(len(allTextContent)))
}
```

### Edge Cases and Special Elements

```go
func edgeCasesExample() {
    html := `<div class="test-cases">
        <div class="empty"></div>
        <div class="whitespace-only">   
        
        </div>
        <div class="self-closing">
            <img src="test.jpg" alt="Test"/>
            <br/>
            <hr/>
        </div>
        <div class="nested">
            <p>Outer <span>inner <em>deep</em></span> text</p>
        </div>
        <script type="text/javascript">
            function test() {
                console.log('Hello World');
            }
        </script>
        <style>
            .test { color: red; }
        </style>
    </div>`

    fmt.Println("=== Edge Cases: ContentsOnly Behavior ===")
    
    testCases := []struct {
        xpath string
        desc  string
    }{
        {"//div[@class='empty']", "Empty elements"},
        {"//div[@class='whitespace-only']", "Whitespace-only elements"},
        {"//img", "Self-closing elements"},
        {"//br", "Void elements"},
        {"//div[@class='nested']//span", "Nested elements"},
        {"//script", "Raw text elements"},
        {"//style", "Style elements"},
    }
    
    for _, tc := range testCases {
        fmt.Printf("\n--- %s ---\n", tc.desc)
        fmt.Printf("XPath: %s\n", tc.xpath)
        
        // Test both modes
        fullResults, _ := xpath.QueryWithOptions(tc.xpath, html, xpath.Options{
            ContentsOnly: false,
        })
        
        contentResults, _ := xpath.QueryWithOptions(tc.xpath, html, xpath.Options{
            ContentsOnly: true,
        })
        
        if len(fullResults) > 0 && len(contentResults) > 0 {
            fullContent := html[fullResults[0].StartLocation:fullResults[0].EndLocation]
            contentOnly := html[contentResults[0].StartLocation:contentResults[0].EndLocation]
            
            fmt.Printf("Full: %q\n", truncateString(fullContent, 60))
            fmt.Printf("Content: %q\n", truncateString(contentOnly, 60))
            
            // Show position difference
            fmt.Printf("Full range: %d-%d (length: %d)\n", 
                fullResults[0].StartLocation, fullResults[0].EndLocation,
                fullResults[0].EndLocation-fullResults[0].StartLocation)
            fmt.Printf("Content range: %d-%d (length: %d)\n", 
                contentResults[0].StartLocation, contentResults[0].EndLocation,
                contentResults[0].EndLocation-contentResults[0].StartLocation)
        }
    }
}

func truncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen-3] + "..."
}
```

This comprehensive examples file demonstrates the full capabilities of the xpath-go library, from basic selections to advanced real-world use cases. Each example is complete and runnable, showing both the power and flexibility of the library, including the new dual extraction modes with the `contentsOnly` option.