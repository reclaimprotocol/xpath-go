package main

import (
	"fmt"
	"strings"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Based on the failing test indices from the previous conversation:
	// 35, 38, 48, 54, 59, 62, 66, 67, 71, 72
	// We've fixed: 59 (Empty vs non-empty), 62 (Position), 66 (Class), 67 (Document), 71 (Substring OR), 72 (Substring edge)
	// Still need to test: 35, 38, 48, 54

	testCases := []struct {
		name  string
		html  string
		xpath string
		desc  string
	}{
		{
			name:  "Empty elements (Test 35)",
			html:  `<html><body><div></div><div>   </div><div><br/></div><div>Content</div></body></html>`,
			xpath: `//div[not(text()) or normalize-space(.)='']`,
			desc:  "Should find divs that are empty or have only whitespace",
		},
		{
			name:  "Ancestor-or-self axis (Test 38)",
			html:  `<html><body><div class="container"><section><p id="target">Text</p></section></div></body></html>`,
			xpath: `//p[@id='target']/ancestor-or-self::*[@class]`,
			desc:  "Should find ancestors (including self) with class attribute",
		},
		{
			name:  "String functions combination (Test 48)",
			html:  `<div><span title="Hello World">A</span><span title="Test">B</span><span title="Hello">C</span></div>`,
			xpath: `//span[contains(@title, 'Hello') and string-length(@title) > 5]`,
			desc:  "Should find spans with 'Hello' in title AND title length > 5",
		},
		{
			name:  "Complex table navigation (Test 54)",
			html:  `<table><tr><td class="header">Name</td><td class="header">Age</td></tr><tr><td>John</td><td>25</td></tr><tr><td>Jane</td><td>30</td></tr></table>`,
			xpath: `//tr[position()>1]/td[position()=1]/following-sibling::td[number(.)>25]`,
			desc:  "Should find data rows where the age (second cell) is > 25",
		},
	}

	fmt.Println("🔍 TESTING PENDING XPATH ISSUES WITH TRACE MODE")
	fmt.Println(strings.Repeat("=", 60))

	// Enable trace mode for detailed debugging
	xpath.EnableTrace()

	for i, tc := range testCases {
		fmt.Printf("\n%d. %s\n", i+1, tc.name)
		fmt.Printf("   %s\n", tc.desc)
		fmt.Printf("   XPath: %s\n", tc.xpath)
		fmt.Printf("   HTML: %s\n\n", tc.html)
		
		fmt.Println("--- TRACE OUTPUT ---")
		results, err := xpath.Query(tc.xpath, tc.html)
		
		fmt.Printf("\n--- RESULTS ---\n")
		if err != nil {
			fmt.Printf("❌ ERROR: %v\n", err)
		} else {
			fmt.Printf("Found %d matches:\n", len(results))
			for j, result := range results {
				fmt.Printf("  %d. <%s", j+1, result.NodeName)
				if len(result.Attributes) > 0 {
					for attr, val := range result.Attributes {
						fmt.Printf(" %s='%s'", attr, val)
					}
				}
				fmt.Printf(">")
				if result.TextContent != "" {
					fmt.Printf(": '%s'", result.TextContent)
				}
				fmt.Println()
			}
		}
		
		fmt.Println("\n" + strings.Repeat("-", 60))
	}

	xpath.DisableTrace()
	fmt.Println("\n🔍 Trace analysis complete. Review the output above to identify issues.")
}