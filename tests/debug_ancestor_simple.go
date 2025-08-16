package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test very simple ancestor axis
	html := `<html><body>
		<div>
			<p>Target paragraph</p>
		</div>
		<section>
			<p>Other paragraph</p>
		</section>
	</body></html>`

	fmt.Println("=== Testing Simple Ancestor Axis ===")
	fmt.Println("HTML Structure:")
	fmt.Println("- div > p (Target paragraph)")
	fmt.Println("- section > p (Other paragraph)")
	fmt.Println()

	testCases := []struct {
		query       string
		description string
		expected    int
	}{
		{
			"//p",
			"All p elements",
			2,
		},
		{
			"//p[ancestor::div]",
			"p elements with div ancestor",
			1, // Only 'Target paragraph' has div ancestor
		},
		{
			"//p[ancestor::section]",
			"p elements with section ancestor",
			1, // Only 'Other paragraph' has section ancestor
		},
		{
			"//p[ancestor::body]",
			"p elements with body ancestor",
			2, // Both have body ancestor
		},
	}

	for i, test := range testCases {
		fmt.Printf("%d. %s\n", i+1, test.description)
		fmt.Printf("   Query: %s\n", test.query)
		fmt.Printf("   Expected: %d results\n", test.expected)

		results, err := xpath.Query(test.query, html)
		if err != nil {
			fmt.Printf("   ERROR: %v\n", err)
		} else {
			fmt.Printf("   Go results: %d\n", len(results))
			for j, result := range results {
				fmt.Printf("     %d. %s\n", j+1, result.TextContent)
			}

			if len(results) == test.expected {
				fmt.Println("   ✅ PASS")
			} else {
				fmt.Println("   ❌ FAIL")
			}
		}
		fmt.Println()
	}
}
