package main

import (
	"fmt"
	"strings"
)

func hasFunctionCalls(expr string) bool {
	functions := []string{
		"contains(", "starts-with(", "string-length(", "normalize-space(",
		"substring(", "not(", "not (", "text()", "position()", "last()", "count(",
	}
	
	for _, fn := range functions {
		if strings.Contains(expr, fn) {
			return true
		}
	}
	
	return false
}

func main() {
	fmt.Println("=== ROUTER CLASSIFICATION DEBUG ===")
	fmt.Println()
	
	// Test the failing predicate
	failingPredicate := "substring(text(), string-length(text()) - 3) = 'Text'"
	
	fmt.Printf("Testing predicate: %s\n", failingPredicate)
	fmt.Println()
	
	// Check function detection
	hasFunc := hasFunctionCalls(failingPredicate)
	fmt.Printf("Has function calls? %t\n", hasFunc)
	
	if hasFunc {
		functions := []string{
			"contains(", "starts-with(", "string-length(", "normalize-space(",
			"substring(", "not(", "not (", "text()", "position()", "last()", "count(",
		}
		
		fmt.Println("Functions detected:")
		for _, fn := range functions {
			if strings.Contains(failingPredicate, fn) {
				fmt.Printf("  - %s\n", strings.TrimSuffix(fn, "("))
			}
		}
		
		fmt.Println("→ Should be classified as FunctionType")
		fmt.Println("→ Should route to function handler")
		
		if strings.Contains(failingPredicate, "substring(") {
			fmt.Println("→ Should call applySubstringPredicate")
		}
	}
}