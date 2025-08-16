package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test each li individually with simpler HTML
	tests := []struct {
		name     string
		html     string
		expected map[string]bool // span, not(a), combined
	}{
		{
			name:     "Li with span only",
			html:     `<ul><li><span>Item 1</span></li></ul>`,
			expected: map[string]bool{"span": true, "not(a)": true, "combined": true},
		},
		{
			name:     "Li with text only",
			html:     `<ul><li>Item 2</li></ul>`,
			expected: map[string]bool{"span": false, "not(a)": true, "combined": false},
		},
		{
			name:     "Li with a and span",
			html:     `<ul><li><a href='#'>Item 3</a><span>Extra</span></li></ul>`,
			expected: map[string]bool{"span": true, "not(a)": false, "combined": false},
		},
	}

	for _, test := range tests {
		fmt.Printf("=== %s ===\n", test.name)
		fmt.Printf("HTML: %s\n", test.html)

		// Test span
		spanResults, err := xpath.Query("//li[span]", test.html)
		if err != nil {
			fmt.Printf("Span test ERROR: %v\n", err)
		} else {
			hasSpan := len(spanResults) > 0
			fmt.Printf("Has span: %v (expected %v) %s\n", hasSpan, test.expected["span"], checkMark(hasSpan == test.expected["span"]))
		}

		// Test not(a)
		notAResults, err := xpath.Query("//li[not(a)]", test.html)
		if err != nil {
			fmt.Printf("Not(a) test ERROR: %v\n", err)
		} else {
			matchesNotA := len(notAResults) > 0
			fmt.Printf("Matches not(a): %v (expected %v) %s\n", matchesNotA, test.expected["not(a)"], checkMark(matchesNotA == test.expected["not(a)"]))
		}

		// Test combined
		combinedResults, err := xpath.Query("//li[span and not(a)]", test.html)
		if err != nil {
			fmt.Printf("Combined test ERROR: %v\n", err)
		} else {
			matchesCombined := len(combinedResults) > 0
			fmt.Printf("Combined: %v (expected %v) %s\n", matchesCombined, test.expected["combined"], checkMark(matchesCombined == test.expected["combined"]))
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
