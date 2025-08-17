package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test case with space-only div that was failing
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`
	xpathExpr := `//div[normalize-space(text())='' and not(*)]`
	
	fmt.Println("=== Testing with trace mode ===")
	xpath.EnableTrace()
	
	fmt.Printf("XPath: %s\n", xpathExpr)
	fmt.Printf("HTML: %s\n\n", html)
	
	results, err := xpath.Query(xpathExpr, html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	
	xpath.DisableTrace()
	
	fmt.Printf("\n=== Results ===\n")
	fmt.Printf("Found %d matches:\n", len(results))
	for i, result := range results {
		fmt.Printf("  %d. Node: <%s>, Text: '%s' (len=%d)\n", 
			i+1, result.NodeName, result.TextContent, len(result.TextContent))
	}
	
	fmt.Println("\nExpected: 2 matches (empty div and space-only div)")
	
	// Test individual conditions to verify they work
	fmt.Println("\n=== Individual condition tests ===")
	
	results1, _ := xpath.Query("//div[normalize-space(text())='']", html)
	fmt.Printf("normalize-space(text())='': %d matches\n", len(results1))
	
	results2, _ := xpath.Query("//div[not(*)]", html)
	fmt.Printf("not(*): %d matches\n", len(results2))
	
	if len(results) == 2 {
		fmt.Println("\n✅ SUCCESS: Fixed the compound boolean expression evaluation!")
	} else {
		fmt.Printf("\n❌ FAILURE: Expected 2 matches, got %d\n", len(results))
	}
}