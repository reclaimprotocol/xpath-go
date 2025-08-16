package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test with html/body wrapper
	tests := []struct {
		name     string
		html     string
		query    string
		expected int
	}{
		{
			name:     "Simple divs with wrapper",
			html:     `<html><body><div></div><div><span></span></div></body></html>`,
			query:    "//div[not(*)]",
			expected: 1,
		},
		{
			name:     "Div with span check with wrapper",
			html:     `<html><body><div></div><div><span></span></div></body></html>`,
			query:    "//div[span]",
			expected: 1,
		},
		{
			name:     "All divs with wrapper",
			html:     `<html><body><div></div><div><span></span></div></body></html>`,
			query:    "//div",
			expected: 2,
		},
		{
			name:     "All spans with wrapper",
			html:     `<html><body><div></div><div><span></span></div></body></html>`,
			query:    "//span",
			expected: 1,
		},
	}

	for _, test := range tests {
		fmt.Printf("Test: %s\n", test.name)
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
