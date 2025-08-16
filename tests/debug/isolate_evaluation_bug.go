package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test the exact failing scenario with minimal HTML
	html := `<li><span>Item 1</span></li>`

	fmt.Println("=== Isolating the Complex Boolean Bug ===")
	fmt.Printf("HTML: %s\n", html)
	fmt.Println()

	// Get the single li node to test evaluateSimpleCondition behavior
	liNodes, err := xpath.Query("//li", html)
	if err != nil {
		fmt.Printf("ERROR getting li: %v\n", err)
		return
	}

	if len(liNodes) == 0 {
		fmt.Println("ERROR: No li nodes found")
		return
	}

	fmt.Printf("Found %d li node(s)\n", len(liNodes))
	fmt.Println()

	// Test individual components that should work
	fmt.Println("=== Component Tests ===")

	tests := []struct {
		query    string
		expected int
		name     string
	}{
		{"//li[span]", 1, "Has span child"},
		{"//li[not(a)]", 1, "Does not have a child"},
		{"//li[span and not(a)]", 1, "Has span AND not a (FAILING)"},
	}

	for _, test := range tests {
		results, err := xpath.Query(test.query, html)
		if err != nil {
			fmt.Printf("  %s: ERROR - %v\n", test.name, err)
		} else {
			status := "✅"
			if len(results) != test.expected {
				status = "❌"
			}
			fmt.Printf("  %s: %d results (expected %d) %s\n", test.name, len(results), test.expected, status)
		}
	}

	fmt.Println()
	fmt.Println("=== Key Insight ===")
	fmt.Println("Both individual conditions work, but the combination fails.")
	fmt.Println("This suggests the issue is in evaluateComplexBooleanExpression,")
	fmt.Println("specifically in how it calls evaluateSimpleCondition.")
	fmt.Println()
	fmt.Println("The flow should be:")
	fmt.Println("1. //li[span and not(a)] → applyPredicate")
	fmt.Println("2. expr = 'span and not(a)' → applyComplexBooleanPredicate")
	fmt.Println("3. evaluateComplexBooleanExpression(expr='span and not(a)', node)")
	fmt.Println("4. Split: left='span', right='not(a)', op='and'")
	fmt.Println("5. leftResult = evaluateSimpleCondition(node, 'span') → should be true")
	fmt.Println("6. rightResult = evaluateSimpleCondition(node, 'not(a)') → should be true")
	fmt.Println("7. return leftResult && rightResult → should be true")
	fmt.Println()
	fmt.Println("But step 6 must be returning false instead of true.")
}
