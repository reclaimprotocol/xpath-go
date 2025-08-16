package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Compare the two different paths for not(a) evaluation
	html := `<li><span>Item 1</span></li>`
	
	fmt.Println("=== Comparing not(a) Evaluation Paths ===")
	fmt.Printf("HTML: %s\n", html)
	fmt.Println()
	
	// Path 1: Direct not() predicate → applyNotPredicate
	fmt.Println("Path 1: Direct not() predicate")
	fmt.Println("Query: //li[not(a)]")
	fmt.Println("Route: //li → applyPredicate → applyNotPredicate")
	
	directResults, err := xpath.Query("//li[not(a)]", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("Results: %d ✅\n", len(directResults))
	}
	
	// Path 2: Complex boolean → evaluateComplexBooleanExpression → evaluateSimpleCondition
	fmt.Println("\nPath 2: Complex boolean with not()")
	fmt.Println("Query: //li[span and not(a)]")
	fmt.Println("Route: //li → applyPredicate → applyComplexBooleanPredicate → evaluateComplexBooleanExpression")
	fmt.Println("  → evaluateSimpleCondition('span') AND evaluateSimpleCondition('not(a)')")
	
	complexResults, err := xpath.Query("//li[span and not(a)]", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("Results: %d ❌\n", len(complexResults))
	}
	
	// The key insight: evaluateSimpleCondition('not(a)') must be returning false
	// when it should return true
	
	fmt.Println("\n=== Analysis ===")
	fmt.Println("The issue is that evaluateSimpleCondition('not(a)') returns different")
	fmt.Println("results when called from complex boolean context vs direct context.")
	fmt.Println()
	fmt.Println("In evaluateSimpleCondition, 'not(a)' triggers:")
	fmt.Println("  if strings.HasPrefix(condition, \"not(\") && strings.HasSuffix(condition, \")\") {")
	fmt.Println("    return len(e.applyNotPredicate([]*types.Node{node}, condition)) > 0")
	fmt.Println("  }")
	fmt.Println()
	fmt.Println("This creates a single-node slice and calls applyNotPredicate.")
	fmt.Println("If applyNotPredicate works correctly, this should work too.")
	fmt.Println("But apparently it doesn't...")
	
	// Test a simpler case to isolate the issue further
	fmt.Println("\n=== Testing simpler combinations ===")
	
	simpleTests := []string{
		"//li[span]",              // Simple element (should work)
		"//li[not(a)]",            // Simple not() (should work)  
		"//li[span and span]",     // Element AND element (should work)
		"//li[not(a) and not(a)]", // Not() AND not() (complex - test if not() is the issue)
	}
	
	for _, query := range simpleTests {
		fmt.Printf("Query: %s → ", query)
		results, err := xpath.Query(query, html)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
		} else {
			fmt.Printf("%d results\n", len(results))
		}
	}
}