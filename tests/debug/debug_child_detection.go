package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test very simple cases
	tests := []struct{
		name string
		html string
		query string
		expected int
	}{
		{
			name: "Empty div",
			html: `<div></div>`,
			query: "//div[not(*)]",
			expected: 1,
		},
		{
			name: "Div with span",
			html: `<div><span></span></div>`,
			query: "//div[not(*)]",
			expected: 0,
		},
		{
			name: "Div with span check",
			html: `<div><span></span></div>`,
			query: "//div[span]",
			expected: 1,
		},
		{
			name: "Two divs - one empty, one with span",
			html: `<div></div><div><span></span></div>`,
			query: "//div[not(*)]",
			expected: 1,
		},
	}
	
	for _, test := range tests {
		fmt.Printf("Test: %s\n", test.name)
		fmt.Printf("HTML: %s\n", test.html)
		fmt.Printf("Query: %s\n", test.query)
		
		results, err := xpath.Query(test.query, test.html)
		if err != nil {
			fmt.Printf("  ERROR: %v\n", err)
		} else {
			fmt.Printf("  Expected: %d, Got: %d, Pass: %v\n", test.expected, len(results), len(results) == test.expected)
		}
		fmt.Println()
	}
}