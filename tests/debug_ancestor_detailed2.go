package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test ancestor axis with predicates - detailed debugging
	html := `<html><body>
		<div class="content">
			<p>Target paragraph</p>
		</div>
	</body></html>`

	fmt.Println("=== Detailed Ancestor Axis Debugging ===")
	fmt.Println("HTML Structure: body > div.content > p")
	fmt.Println()

	// Test step by step to isolate the issue
	testCases := []struct {
		query       string
		description string
	}{
		{
			"//div[@class='content']",
			"Direct selection of div.content",
		},
		{
			"//p",
			"Direct selection of p",
		},
		{
			"//p[ancestor::*]",
			"p with any ancestor",
		},
		{
			"//p[ancestor::div]",
			"p with div ancestor",
		},
		{
			"//p[ancestor::body]",
			"p with body ancestor",
		},
		{
			"//p[ancestor::div[@class='content']]",
			"p with div.content ancestor (FAILING)",
		},
	}

	for i, test := range testCases {
		fmt.Printf("%d. %s\n", i+1, test.description)
		fmt.Printf("   Query: %s\n", test.query)

		results, err := xpath.Query(test.query, html)
		if err != nil {
			fmt.Printf("   ERROR: %v\n", err)
		} else {
			fmt.Printf("   Results: %d\n", len(results))
			for j, result := range results {
				fmt.Printf("     %d. %s\n", j+1, result.TextContent)
			}
		}
		fmt.Println()
	}

	// Let's also manually test what a div.content looks like
	fmt.Println("=== Manual verification ===")
	divResults, _ := xpath.Query("//div", html)
	for i, div := range divResults {
		fmt.Printf("Div %d: name='%s', class='%s'\n", i+1, div.NodeName, div.Attributes["class"])
	}
}
