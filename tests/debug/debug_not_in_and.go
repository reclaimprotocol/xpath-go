package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Create a minimal test case to debug the specific issue
	html := `<li><span>Item 1</span></li>`
	
	fmt.Println("=== Debugging not() in AND context ===")
	fmt.Printf("HTML: %s\n", html)
	fmt.Println()
	
	// Test all the individual components that should work
	tests := []struct{
		query string
		expected int
		description string
	}{
		{"//li", 1, "Basic li selection"},
		{"//li[span]", 1, "Li with span child"},
		{"//li[not(a)]", 1, "Li without a child"},
		{"//li[span and not(a)]", 1, "Li with span AND without a"},
		
		// Test some variations
		{"//li[@id and @class]", 0, "Li with both id and class (should fail)"},
		{"//li[text() and not(a)]", 0, "Li with text and not a (should fail - no direct text)"},
	}
	
	for i, test := range tests {
		fmt.Printf("%d. %s\n", i+1, test.description)
		fmt.Printf("   Query: %s\n", test.query)
		
		results, err := xpath.Query(test.query, html)
		if err != nil {
			fmt.Printf("   ERROR: %v\n", err)
		} else {
			success := len(results) == test.expected
			status := "❌"
			if success {
				status = "✅"
			}
			fmt.Printf("   Results: %d (expected %d) %s\n", len(results), test.expected, status)
		}
		fmt.Println()
	}
	
	// Test with a case that has both span and a to make sure the logic is sound
	fmt.Println("=== Testing with span AND a ===")
	htmlWithBoth := `<li><span>Item</span><a href="#">Link</a></li>`
	
	bothTests := []struct{
		query string
		expected int
		description string
	}{
		{"//li[span]", 1, "Should find li with span"},
		{"//li[not(a)]", 0, "Should NOT find li (has a)"},
		{"//li[span and not(a)]", 0, "Should NOT find li (has both span and a)"},
	}
	
	for i, test := range bothTests {
		fmt.Printf("%d. %s\n", i+1, test.description)
		fmt.Printf("   HTML: %s\n", htmlWithBoth)
		fmt.Printf("   Query: %s\n", test.query)
		
		results, err := xpath.Query(test.query, htmlWithBoth)
		if err != nil {
			fmt.Printf("   ERROR: %v\n", err)
		} else {
			success := len(results) == test.expected
			status := "❌"
			if success {
				status = "✅"
			}
			fmt.Printf("   Results: %d (expected %d) %s\n", len(results), test.expected, status)
		}
		fmt.Println()
	}
}