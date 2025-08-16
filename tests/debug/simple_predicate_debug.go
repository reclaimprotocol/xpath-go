package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Ultra simple HTML to make sure we trigger the predicate logic
	html := `<li><span>Item 1</span></li>`

	fmt.Println("=== Simple Predicate Test ===")
	fmt.Printf("HTML: %s\n", html)
	fmt.Println()

	// Test the exact failing query
	fmt.Println("Testing: //li[span and not(a)]")
	results, err := xpath.Query("//li[span and not(a)]", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("Results: %d\n", len(results))
	}
}
