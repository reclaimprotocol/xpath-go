package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test the exact case from comprehensive test
	html := `<html><body><ul><li><span>Item 1</span><!-- comment --></li><li>Item 2</li><li><a href='#'>Item 3</a><span>Extra</span></li></ul></body></html>`
	query := `//li[span and not(a)]`

	fmt.Printf("=== Exact Comprehensive Test Case ===\n")
	fmt.Printf("HTML structure:\n")
	fmt.Printf("  <li><span>Item 1</span><!-- comment --></li>    (should match: has span, no a)\n")
	fmt.Printf("  <li>Item 2</li>                                (should NOT match: no span)\n")
	fmt.Printf("  <li><a href='#'>Item 3</a><span>Extra</span></li> (should NOT match: has both span and a)\n")
	fmt.Printf("\nQuery: %s\n", query)
	fmt.Printf("Expected: 1 result (first li)\n\n")

	// Test the actual query
	results, err := xpath.Query(query, html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	fmt.Printf("Actual results: %d\n", len(results))

	if len(results) == 1 {
		fmt.Println("✅ SUCCESS!")
	} else {
		fmt.Println("❌ FAILED")

		// Debug the components
		fmt.Println("\n=== Component Analysis ===")

		// Count total elements
		allLi, _ := xpath.Query("//li", html)
		fmt.Printf("Total li elements: %d\n", len(allLi))

		allSpan, _ := xpath.Query("//span", html)
		fmt.Printf("Total span elements: %d\n", len(allSpan))

		allA, _ := xpath.Query("//a", html)
		fmt.Printf("Total a elements: %d\n", len(allA))

		// Test components
		liWithSpan, _ := xpath.Query("//li[span]", html)
		fmt.Printf("Li with span: %d\n", len(liWithSpan))

		liWithoutA, _ := xpath.Query("//li[not(a)]", html)
		fmt.Printf("Li without a: %d\n", len(liWithoutA))

		// Manual intersection check
		fmt.Printf("Expected intersection: 1 (first li has span AND no a)\n")
	}
}
