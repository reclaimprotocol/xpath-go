package main

import (
	"fmt"
	"log"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><head><title>Test</title><meta charset='utf-8'/></head><body><main><h1>Title</h1><p>Content</p></main></body></html>`
	
	fmt.Println("=== TESTING DOCUMENT STRUCTURE VALIDATION ===")
	fmt.Println()
	
	// Test the failing expression
	xpathExpr := "/html[head/title and head/meta[@charset] and body/main]"
	fmt.Printf("XPath: %s\n", xpathExpr)
	
	result, err := xpath.Query(xpathExpr, htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Go results: %d\n", len(result))
	
	// Test individual parts
	fmt.Println("\nTesting individual conditions:")
	
	result1, err := xpath.Query("/html[head/title]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("html with head/title: %d results\n", len(result1))
	
	result2, err := xpath.Query("/html[head/meta[@charset]]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("html with head/meta[@charset]: %d results\n", len(result2))
	
	result3, err := xpath.Query("/html[body/main]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("html with body/main: %d results\n", len(result3))
	
	// Test two-part combinations
	fmt.Println("\nTesting two-part combinations:")
	
	result4, err := xpath.Query("/html[head/title and head/meta[@charset]]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("head/title AND head/meta[@charset]: %d results\n", len(result4))
	
	result5, err := xpath.Query("/html[head/meta[@charset] and body/main]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("head/meta[@charset] AND body/main: %d results\n", len(result5))
	
	fmt.Println("\nExpected: All parts should return 1, full expression should return 1")
}