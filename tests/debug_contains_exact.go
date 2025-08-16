package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test exact contains behavior
	html := `<html><body>
		<div class='primary active large'>Item 1</div>
		<div class='secondary active'>Item 2</div>
		<div class='primary inactive'>Item 3</div>
	</body></html>`
	
	fmt.Println("=== Testing contains() function exact behavior ===")
	
	testCases := []struct {
		query       string
		description string
		expected    []string
	}{
		{
			"//div[contains(@class, 'active')]",
			"Should find divs containing 'active' (substring match)",
			[]string{"Item 1", "Item 2", "Item 3"}, // Item 3 matches because 'active' is in 'inactive'
		},
		{
			"//div[contains(@class, 'inactive')]",
			"Should find divs containing 'inactive'",
			[]string{"Item 3"},
		},
		{
			"//div[contains(@class, 'primary')]",
			"Should find divs containing 'primary'",
			[]string{"Item 1", "Item 3"},
		},
	}
	
	for i, test := range testCases {
		fmt.Printf("\n%d. %s\n", i+1, test.description)
		fmt.Printf("Query: %s\n", test.query)
		fmt.Printf("Expected: %v\n", test.expected)
		
		results, err := xpath.Query(test.query, html)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}
		
		fmt.Printf("Go results: %d\n", len(results))
		for j, result := range results {
			fmt.Printf("  %d. %s\n", j+1, result.TextContent)
		}
		
		// Check if correct
		if len(results) == len(test.expected) {
			fmt.Println("✅ Correct count")
		} else {
			fmt.Printf("❌ WRONG count - expected %d, got %d\n", len(test.expected), len(results))
			
			// Print each element's class attribute for debugging
			fmt.Println("\nDEBUG: Element classes:")
			allDivs, _ := xpath.Query("//div", html)
			for k, div := range allDivs {
				fmt.Printf("  Item %d: class='%s'\n", k+1, div.Attributes["class"])
			}
		}
	}
}