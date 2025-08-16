package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test how nodeTest predicate matching works
	html := `<html><body>
		<div class="content">
			<p>Target paragraph</p>
		</div>
	</body></html>`
	
	fmt.Println("=== Testing NodeTest Predicate Matching ===")
	fmt.Println("HTML Structure: body > div.content > p")
	fmt.Println()
	
	// Test individual components step by step
	testCases := []string{
		"//div",                      // Should find 1 div
		"//div[@class]",             // Should find 1 div with class attribute
		"//div[@class='content']",   // Should find 1 div with class='content'
		"//p[ancestor::div]",        // Should find 1 p with div ancestor
		"//p[ancestor::div[@class='content']]", // Should find 1 p but currently fails
	}
	
	for i, query := range testCases {
		fmt.Printf("%d. Query: %s\n", i+1, query)
		
		results, err := xpath.Query(query, html)
		if err != nil {
			fmt.Printf("   ERROR: %v\n", err)
		} else {
			fmt.Printf("   Results: %d\n", len(results))
			for j, result := range results {
				fmt.Printf("     %d. %s (tag: %s)\n", j+1, result.TextContent, result.NodeName)
				if result.NodeName == "div" && len(result.Attributes) > 0 {
					fmt.Printf("        Attributes: %v\n", result.Attributes)
				}
			}
		}
		fmt.Println()
	}
}