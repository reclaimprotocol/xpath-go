package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test the class list manipulation case
	html := `<html><body>
		<div class="primary active large">Item 1</div>
		<div class="primary inactive">Item 2</div>
		<div class="secondary active">Item 3</div>
		<div class="primary active inactive">Item 4</div>
	</body></html>`

	query := `//div[contains(@class, 'primary') and contains(@class, 'active') and not(contains(@class, 'inactive'))]`

	fmt.Println("=== Class List Manipulation Test ===")
	fmt.Printf("Query: %s\n", query)
	fmt.Println()
	fmt.Println("Expected: Should match only 'Item 1' (has primary AND active, but NOT inactive)")
	fmt.Println("- Item 1: primary active large ✅ (has primary, has active, no inactive)")
	fmt.Println("- Item 2: primary inactive ❌ (has primary, no active)")
	fmt.Println("- Item 3: secondary active ❌ (no primary)")
	fmt.Println("- Item 4: primary active inactive ❌ (has primary, has active, but HAS inactive)")
	fmt.Println()

	// Test the query
	results, err := xpath.Query(query, html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	fmt.Printf("Actual results: %d\n", len(results))

	if len(results) == 1 {
		fmt.Println("✅ SUCCESS!")
		fmt.Printf("Found: %s\n", results[0].TextContent)
	} else {
		fmt.Println("❌ FAILED")
		for i, result := range results {
			fmt.Printf("  Result %d: %s\n", i+1, result.TextContent)
		}

		// Debug the individual components
		fmt.Println("\n=== Component Analysis ===")

		components := []string{
			"//div[contains(@class, 'primary')]",
			"//div[contains(@class, 'active')]",
			"//div[not(contains(@class, 'inactive'))]",
			"//div[contains(@class, 'primary') and contains(@class, 'active')]",
		}

		for _, comp := range components {
			compResults, err := xpath.Query(comp, html)
			if err != nil {
				fmt.Printf("  %s: ERROR - %v\n", comp, err)
			} else {
				fmt.Printf("  %s: %d results\n", comp, len(compResults))
			}
		}
	}
}
