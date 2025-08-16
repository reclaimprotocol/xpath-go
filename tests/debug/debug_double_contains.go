package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Simplified test to isolate double contains issue
	html := `<html><body>
		<div class="primary active large">Item 1</div>
		<div class="primary inactive">Item 2</div>
		<div class="secondary active">Item 3</div>
	</body></html>`
	
	fmt.Println("=== Double Contains Test ===")
	fmt.Println("HTML:")
	fmt.Println("- Item 1: class=\"primary active large\"")
	fmt.Println("- Item 2: class=\"primary inactive\"") 
	fmt.Println("- Item 3: class=\"secondary active\"")
	fmt.Println()
	
	tests := []struct {
		query    string
		expected int
		name     string
	}{
		{"//div[contains(@class, 'primary')]", 2, "Has 'primary'"},
		{"//div[contains(@class, 'active')]", 2, "Has 'active'"},
		{"//div[contains(@class, 'primary') and contains(@class, 'active')]", 1, "Has BOTH 'primary' AND 'active'"},
	}
	
	for _, test := range tests {
		results, err := xpath.Query(test.query, html)
		if err != nil {
			fmt.Printf("❌ %s: ERROR - %v\n", test.name, err)
			continue
		}
		
		status := "✅"
		if len(results) != test.expected {
			status = "❌"
		}
		
		fmt.Printf("%s %s: %d results (expected %d)\n", status, test.name, len(results), test.expected)
		
		if len(results) != test.expected {
			fmt.Println("  Found:")
			for i, result := range results {
				fmt.Printf("    %d. %s\n", i+1, result.TextContent)
			}
		}
	}
	
	fmt.Println()
	fmt.Println("The issue: Item 2 ('primary inactive') should NOT match")
	fmt.Println("the query [contains(@class, 'primary') and contains(@class, 'active')]")
	fmt.Println("because it doesn't have 'active' class.")
}