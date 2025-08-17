package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test normalize-space behavior specifically
	tests := []struct{
		html string
		xpath string
		expected bool
	}{
		{`<div> </div>`, `//div[normalize-space(text())='']`, true},
		{`<div></div>`, `//div[normalize-space(text())='']`, true},
		{`<div>content</div>`, `//div[normalize-space(text())='']`, false},
		{`<div>  space  </div>`, `//div[normalize-space(text())='space']`, true},
	}
	
	fmt.Println("Testing normalize-space function directly:")
	
	for i, test := range tests {
		fmt.Printf("\nTest %d:\n", i+1)
		fmt.Printf("  HTML: %s\n", test.html)
		fmt.Printf("  XPath: %s\n", test.xpath)
		
		results, err := xpath.Query(test.xpath, test.html)
		actual := len(results) > 0
		
		if err != nil {
			fmt.Printf("  ERROR: %v\n", err)
		} else {
			fmt.Printf("  Expected: %v, Actual: %v", test.expected, actual)
			if actual == test.expected {
				fmt.Printf(" ✓\n")
			} else {
				fmt.Printf(" ✗\n")
			}
		}
	}
}