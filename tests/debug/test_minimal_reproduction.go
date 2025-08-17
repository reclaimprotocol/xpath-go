package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Exact test case from the failing suite
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`
	xpath_expr := `//div[normalize-space(text())='' and not(*)]`
	
	fmt.Printf("Testing: %s\n", xpath_expr)
	fmt.Printf("HTML: %s\n", html)
	fmt.Println()
	
	results, err := xpath.Query(xpath_expr, html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	
	fmt.Printf("Go Results: %d matches\n", len(results))
	for i, result := range results {
		fmt.Printf("  %d. Text: '%s' (len=%d)\n", i+1, result.TextContent, len(result.TextContent))
	}
	
	fmt.Println("\nExpected (from JavaScript): 2 matches")
	fmt.Println("  1. Empty div: ''")
	fmt.Println("  2. Space-only div: ' '")
	
	// Test individual parts
	fmt.Println("\nTesting individual conditions:")
	
	results1, _ := xpath.Query("//div[normalize-space(text())='']", html)
	fmt.Printf("  normalize-space(text())='': %d matches\n", len(results1))
	
	results2, _ := xpath.Query("//div[not(*)]", html)
	fmt.Printf("  not(*): %d matches\n", len(results2))
	
	// Expected intersection
	fmt.Println("\nShould be intersection of both conditions")
}