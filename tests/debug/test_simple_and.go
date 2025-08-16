package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test simple element AND that should definitely work
	html := `<li><span>text</span><div>content</div></li>`
	
	fmt.Println("=== Testing Simple Element AND ===")
	fmt.Printf("HTML: %s\n", html)
	fmt.Println()
	
	// These should all work
	tests := []struct{
		query string
		expected int
		shouldWork bool
	}{
		{"//li", 1, true},                    // Basic
		{"//li[span]", 1, true},              // Single element
		{"//li[div]", 1, true},               // Single element  
		{"//li[span and div]", 1, true},      // Simple AND (no functions)
		{"//li[not(a)]", 1, true},            // Single not()
		{"//li[span and not(a)]", 1, true},   // Complex AND with not()
	}
	
	for i, test := range tests {
		fmt.Printf("%d. Query: %s\n", i+1, test.query)
		
		results, err := xpath.Query(test.query, html)
		if err != nil {
			fmt.Printf("   ERROR: %v\n", err)
		} else {
			success := len(results) == test.expected
			status := "✅"
			if !success && test.shouldWork {
				status = "❌"
			} else if success && !test.shouldWork {
				status = "❌"
			}
			fmt.Printf("   Results: %d (expected %d) %s\n", len(results), test.expected, status)
		}
		fmt.Println()
	}
}