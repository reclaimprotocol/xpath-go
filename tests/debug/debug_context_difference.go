package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test the exact same node evaluation in different contexts
	html := `<li><span>Item 1</span></li>`

	fmt.Println("=== Testing evaluation context differences ===")
	fmt.Printf("HTML: %s\n", html)
	fmt.Println()

	// Test the three different routes that should all give the same result
	tests := []struct {
		query         string
		description   string
		expectedRoute string
	}{
		{
			"//li[not(a)]",
			"Direct not(a) - simple predicate route",
			"applyNotPredicate",
		},
		{
			"//li[span and not(a)]",
			"Complex boolean route with not(a)",
			"applyComplexBooleanPredicate → evaluateSimpleCondition",
		},
		{
			"//li[@id and not(a)]",
			"Different complex case for comparison",
			"applyComplexBooleanPredicate → evaluateSimpleCondition",
		},
	}

	for i, test := range tests {
		fmt.Printf("%d. %s\n", i+1, test.description)
		fmt.Printf("   Route: %s\n", test.expectedRoute)
		fmt.Printf("   Query: %s\n", test.query)

		results, err := xpath.Query(test.query, html)
		if err != nil {
			fmt.Printf("   ERROR: %v\n", err)
		} else {
			fmt.Printf("   Results: %d\n", len(results))
		}
		fmt.Println()
	}

	// Now test a case that we know works to see the difference
	fmt.Println("=== Testing known working cases ===")

	workingTests := []struct {
		query       string
		description string
	}{
		{"//li[@id and @class]", "Attribute AND (working)"},
		{"//li[span and div]", "Simple element AND (working)"},
		{"//li[contains(text(), 'Item') and not(a)]", "Function AND not() (test)"},
	}

	for i, test := range workingTests {
		fmt.Printf("%d. %s\n", i+1, test.description)
		fmt.Printf("   Query: %s\n", test.query)

		results, err := xpath.Query(test.query, html)
		if err != nil {
			fmt.Printf("   ERROR: %v\n", err)
		} else {
			fmt.Printf("   Results: %d\n", len(results))
		}
		fmt.Println()
	}
}
