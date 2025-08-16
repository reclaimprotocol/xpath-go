package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test ancestor axis filtering issues
	html := `<html><body>
		<div class="container">
			<div class="content">
				<article>
					<p class="text">Target paragraph</p>
				</article>
			</div>
		</div>
		<div class="other">
			<article>
				<p class="text">Non-target paragraph</p>
			</article>
		</div>
	</body></html>`
	
	fmt.Println("=== Testing Ancestor Axis Filtering ===")
	fmt.Println("HTML Structure:")
	fmt.Println("- div.container > div.content > article > p.text (should match)")
	fmt.Println("- div.other > article > p.text (should NOT match)")
	fmt.Println()
	
	testCases := []struct {
		query       string
		description string
		expected    int
	}{
		{
			"//p[@class='text']",
			"All p elements with class='text'",
			2,
		},
		{
			"//p[@class='text'][ancestor::div[@class='content']]",
			"p.text elements that have ancestor div.content",
			1, // Only the first one should match
		},
		{
			"//div[@class='container']//article//p[ancestor::div[@class='content']]",
			"Complex nested: p elements under container/article with content ancestor",
			1,
		},
		{
			"//article//p[ancestor::div[@class='container']]",
			"p elements under article with container ancestor",
			1, // Only the first p has div.container as ancestor
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