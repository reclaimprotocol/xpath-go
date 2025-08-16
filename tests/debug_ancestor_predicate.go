package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test ancestor axis with predicates
	html := `<html><body>
		<div class="content">
			<p>Target paragraph</p>
		</div>
		<div class="other">
			<p>Non-target paragraph</p>
		</div>
	</body></html>`

	fmt.Println("=== Testing Ancestor Axis with Predicates ===")
	fmt.Println("HTML Structure:")
	fmt.Println("- div.content > p (Target paragraph)")
	fmt.Println("- div.other > p (Non-target paragraph)")
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
			"p elements with any div ancestor",
			2, // Both have div ancestors
		},
		{
			"//div[@class='content']",
			"Divs with class='content'",
			1, // Verify the div exists
		},
		{
			"//p[ancestor::div[@class='content']]",
			"p elements with div.content ancestor",
			1, // Only 'Target paragraph' should match
		},
		{
			"//p[ancestor::div[@class='other']]",
			"p elements with div.other ancestor",
			1, // Only 'Non-target paragraph' should match
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
