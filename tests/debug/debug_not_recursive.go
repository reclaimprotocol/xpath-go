package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test individual elements to understand the recursive not() issue
	html := `<html><body>
		<div class='primary active large'>Item 1</div>
		<div class='secondary active'>Item 2</div>
		<div class='primary inactive'>Item 3</div>
	</body></html>`
	
	fmt.Println("=== Debugging Recursive not() Issue ===")
	fmt.Println()
	
	// Test each individual element with both positive and negative contains
	elements := []string{
		"Item 1: class='primary active large'",
		"Item 2: class='secondary active'", 
		"Item 3: class='primary inactive'",
	}
	
	queries := []string{
		"contains(@class, 'inactive')",
		"not(contains(@class, 'inactive'))",
	}
	
	for i, elem := range elements {
		fmt.Printf("Testing %s:\n", elem)
		
		for _, query := range queries {
			fullQuery := fmt.Sprintf("//div[position()=%d][%s]", i+1, query)
			results, err := xpath.Query(fullQuery, html)
			if err != nil {
				fmt.Printf("  %s: ERROR - %v\n", query, err)
			} else {
				hasResult := len(results) > 0
				fmt.Printf("  %s: %v\n", query, hasResult)
			}
		}
		fmt.Println()
	}
	
	fmt.Println("Expected behavior:")
	fmt.Println("- Item 1: contains(class, 'inactive') = false, not(contains(class, 'inactive')) = true")
	fmt.Println("- Item 2: contains(class, 'inactive') = false, not(contains(class, 'inactive')) = true") 
	fmt.Println("- Item 3: contains(class, 'inactive') = true, not(contains(class, 'inactive')) = false")
}