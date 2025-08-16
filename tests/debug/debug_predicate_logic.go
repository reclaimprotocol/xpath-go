package main

import (
	"fmt"
	"strings"
)

func containsFunctionCall(expr string) bool {
	functionNames := []string{
		"contains(", "starts-with(", "string-length(", "normalize-space(",
		"substring(", "not(", "text()", "position()", "last()", "count(",
	}

	for _, fn := range functionNames {
		if strings.Contains(expr, fn) {
			return true
		}
	}

	return false
}

func main() {
	expr := "normalize-space(text())='' and not(*)"

	fmt.Printf("Expression: %s\n", expr)
	fmt.Printf("Contains ' and ': %v\n", strings.Contains(expr, " and "))
	fmt.Printf("Contains ' or ': %v\n", strings.Contains(expr, " or "))
	fmt.Printf("Contains '(': %v\n", strings.Contains(expr, "("))
	fmt.Printf("Contains function call: %v\n", containsFunctionCall(expr))

	// Check the condition for complex boolean
	hasBoolean := strings.Contains(expr, " and ") || strings.Contains(expr, " or ")
	hasComplexity := strings.Contains(expr, "(") || containsFunctionCall(expr)
	shouldUseComplex := hasBoolean && hasComplexity

	fmt.Printf("Should use complex boolean: %v\n", shouldUseComplex)
}
