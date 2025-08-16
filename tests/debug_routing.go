package main

import (
	"fmt"
	"strings"
)

func containsFunctionCall(expr string) bool {
	functionNames := []string{
		"contains(", "starts-with(", "string-length(", "normalize-space(",
		"substring(", "not(", "not (", "text()", "position()", "last()", "count(",
	}
	
	for _, fn := range functionNames {
		if strings.Contains(expr, fn) {
			return true
		}
	}
	return false
}

func main() {
	expr := "contains(@class, 'primary') and contains(@class, 'active') and not(contains(@class, 'inactive'))"
	
	fmt.Printf("Expression: %s\n", expr)
	fmt.Printf("Contains ' and ': %t\n", strings.Contains(expr, " and "))
	fmt.Printf("Contains ' or ': %t\n", strings.Contains(expr, " or "))
	fmt.Printf("Contains '(': %t\n", strings.Contains(expr, "("))
	fmt.Printf("containsFunctionCall: %t\n", containsFunctionCall(expr))
	
	// Test routing condition
	complexCondition := (strings.Contains(expr, " and ") || strings.Contains(expr, " or ")) && 
	                   (strings.Contains(expr, "(") || containsFunctionCall(expr))
	fmt.Printf("Should route to complex boolean: %t\n", complexCondition)
	
	// Test simple AND condition
	simpleAndCondition := strings.Contains(expr, " and ")
	fmt.Printf("Should route to simple AND: %t\n", simpleAndCondition)
	
	// Which takes precedence?
	fmt.Printf("\nRouting decision:\n")
	if complexCondition {
		fmt.Println("→ Routes to applyComplexBooleanPredicate")
	} else if simpleAndCondition {
		fmt.Println("→ Routes to applyAndPredicate")
	} else {
		fmt.Println("→ Routes to other handler")
	}
}