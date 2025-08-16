package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test not(a) evaluation specifically
	testCases := []struct {
		name     string
		html     string
		query    string
		expected int
	}{
		{
			name:     "Element without a",
			html:     `<li>text only</li>`,
			query:    "//li[not(a)]",
			expected: 1,
		},
		{
			name:     "Element with a",
			html:     `<li><a href="#">link</a></li>`,
			query:    "//li[not(a)]",
			expected: 0,
		},
		{
			name:     "Element with span but not a",
			html:     `<li><span>text</span></li>`,
			query:    "//li[not(a)]",
			expected: 1,
		},
		{
			name:     "Element with both span and a",
			html:     `<li><span>text</span><a href="#">link</a></li>`,
			query:    "//li[not(a)]",
			expected: 0,
		},
	}

	fmt.Println("=== Testing not(a) evaluation ===")
	for _, test := range testCases {
		fmt.Printf("\n%s:\n", test.name)
		fmt.Printf("HTML: %s\n", test.html)
		fmt.Printf("Query: %s\n", test.query)

		results, err := xpath.Query(test.query, test.html)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
		} else {
			success := len(results) == test.expected
			fmt.Printf("Results: %d (expected %d) %s\n", len(results), test.expected, checkMark(success))
		}
	}

	// Now test the full combination on our problematic case
	fmt.Println("\n=== Testing the combination ===")
	html := `<li><span>Item 1</span></li>`

	// Test individual parts
	spanTest, err := xpath.Query("//li[span]", html)
	if err != nil {
		fmt.Printf("Span test error: %v\n", err)
		return
	}
	fmt.Printf("//li[span]: %d\n", len(spanTest))

	notATest, err := xpath.Query("//li[not(a)]", html)
	if err != nil {
		fmt.Printf("Not(a) test error: %v\n", err)
		return
	}
	fmt.Printf("//li[not(a)]: %d\n", len(notATest))

	// Test combination
	bothTest, err := xpath.Query("//li[span and not(a)]", html)
	if err != nil {
		fmt.Printf("Combination test error: %v\n", err)
		return
	}
	fmt.Printf("//li[span and not(a)]: %d (should be 1)\n", len(bothTest))
}

func checkMark(correct bool) string {
	if correct {
		return "✅"
	}
	return "❌"
}
