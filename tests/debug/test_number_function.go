package main

import (
	"fmt"
	"strings"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test the specific failing case with number() function
	html := `<table><tr><td class="header">Name</td><td class="header">Age</td></tr><tr><td>John</td><td>25</td></tr><tr><td>Jane</td><td>30</td></tr></table>`
	
	fmt.Println("Testing number() function compatibility:")
	fmt.Println(strings.Repeat("=", 50))
	
	// Enable trace for debugging
	xpath.EnableTrace()
	
	testCases := []struct {
		name  string
		xpath string
		desc  string
	}{
		{
			name:  "Basic number comparison",
			xpath: `//td[number(.) > 25]`,
			desc:  "Should find td cells with numeric value > 25",
		},
		{
			name:  "Complex table navigation with number",
			xpath: `//tr[position()>1]/td[position()=1]/following-sibling::td[number(.)>25]`,
			desc:  "Original failing case - find ages > 25 in data rows",
		},
		{
			name:  "Number equal comparison",
			xpath: `//td[number(.) = 30]`,
			desc:  "Should find td with value exactly 30",
		},
		{
			name:  "Number with text content",
			xpath: `//td[number(text()) >= 25]`,
			desc:  "Should find td with text value >= 25",
		},
	}
	
	for i, tc := range testCases {
		fmt.Printf("\n%d. %s\n", i+1, tc.name)
		fmt.Printf("   %s\n", tc.desc)
		fmt.Printf("   XPath: %s\n\n", tc.xpath)
		
		results, err := xpath.Query(tc.xpath, html)
		if err != nil {
			fmt.Printf("   ❌ ERROR: %v\n", err)
		} else {
			fmt.Printf("   Found %d matches:", len(results))
			for j, result := range results {
				fmt.Printf(" [%d: '%s']", j+1, result.TextContent)
			}
			fmt.Println()
		}
		fmt.Println()
	}
	
	xpath.DisableTrace()
	
	fmt.Println("\nExpected results:")
	fmt.Println("1. Basic: Should find '30' (only value > 25)")
	fmt.Println("2. Complex: Should find '30' from Jane's row")
	fmt.Println("3. Equal: Should find '30'")
	fmt.Println("4. Text: Should find '25' and '30'")
}