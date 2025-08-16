package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test the predicate evaluation logic more directly
	htmlCases := []struct {
		name     string
		html     string
		query    string
		expected int
	}{
		{
			name:     "Li with span",
			html:     `<li><span>test</span></li>`,
			query:    "//li[span]",
			expected: 1,
		},
		{
			name:     "Li without span",
			html:     `<li>just text</li>`,
			query:    "//li[span]",
			expected: 0,
		},
		{
			name:     "Li with a",
			html:     `<li><a href="#">test</a></li>`,
			query:    "//li[a]",
			expected: 1,
		},
		{
			name:     "Li without a",
			html:     `<li>just text</li>`,
			query:    "//li[a]",
			expected: 0,
		},
		{
			name:     "Li with not(a) - should match",
			html:     `<li>just text</li>`,
			query:    "//li[not(a)]",
			expected: 1,
		},
		{
			name:     "Li with not(a) - should not match",
			html:     `<li><a href="#">test</a></li>`,
			query:    "//li[not(a)]",
			expected: 0,
		},
	}

	for _, test := range htmlCases {
		fmt.Printf("=== %s ===\n", test.name)
		fmt.Printf("HTML: %s\n", test.html)
		fmt.Printf("Query: %s\n", test.query)

		results, err := xpath.Query(test.query, test.html)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
		} else {
			success := len(results) == test.expected
			fmt.Printf("Results: %d (expected %d) %s\n", len(results), test.expected, checkMark(success))
		}
		fmt.Println()
	}
}

func checkMark(correct bool) string {
	if correct {
		return "✅"
	}
	return "❌"
}
