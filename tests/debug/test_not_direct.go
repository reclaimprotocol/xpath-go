package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test if the issue is in the single-node slice handling
	html := `<li><span>Item 1</span></li>`

	fmt.Println("=== Testing applyNotPredicate behavior ===")
	fmt.Printf("HTML: %s\n", html)
	fmt.Println()

	// First, let's verify the direct queries work
	fmt.Println("1. Direct verification:")

	// Get the li node
	liNodes, err := xpath.Query("//li", html)
	if err != nil {
		fmt.Printf("ERROR getting li: %v\n", err)
		return
	}
	fmt.Printf("Found %d li nodes\n", len(liNodes))

	// Test direct not(a) predicate
	notANodes, err := xpath.Query("//li[not(a)]", html)
	if err != nil {
		fmt.Printf("ERROR with not(a): %v\n", err)
		return
	}
	fmt.Printf("Direct //li[not(a)]: %d results ✅\n", len(notANodes))

	fmt.Println("\n2. Understanding the problem:")
	fmt.Println("The issue might be that when evaluateSimpleCondition calls")
	fmt.Println("applyNotPredicate with a single-node slice, the context is different")
	fmt.Println("than when applyNotPredicate is called directly from applyPredicate.")

	fmt.Println("\n3. Testing context differences:")

	// Test some edge cases that might reveal the issue
	edgeCases := []string{
		"//li[@nonexistent and not(a)]", // Attribute + not() (should be 0)
		"//li[text() and not(a)]",       // Function + not() (should be 0)
		"//li[position()=1 and not(a)]", // Position + not() (should be 1)
	}

	for _, query := range edgeCases {
		fmt.Printf("Query: %s → ", query)
		results, err := xpath.Query(query, html)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
		} else {
			fmt.Printf("%d results\n", len(results))
		}
	}

	fmt.Println("\n4. Key insight:")
	fmt.Println("If position()=1 and not(a) works but span and not(a) doesn't,")
	fmt.Println("then the issue is specifically with element existence in complex boolean context.")
}
