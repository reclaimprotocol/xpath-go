package main

import (
	"fmt"
	"strings"
)

// Copied from the evaluator
func splitBooleanExpression(expr string, operator string) (string, string, bool) {
	inQuote := false
	var quoteChar byte
	parenDepth := 0
	
	for i := 0; i <= len(expr)-len(operator); i++ {
		char := expr[i]
		
		// Handle quote state
		if (char == '\'' || char == '"') && (i == 0 || expr[i-1] != '\\') {
			if !inQuote {
				inQuote = true
				quoteChar = char
			} else if char == quoteChar {
				inQuote = false
			}
		}
		
		// Handle parentheses depth (when not in quotes)
		if !inQuote {
			if char == '(' {
				parenDepth++
			} else if char == ')' {
				parenDepth--
			}
			
			// Check for operator at this position (when not in quotes and at paren depth 0)
			if parenDepth == 0 && i+len(operator) <= len(expr) {
				if expr[i:i+len(operator)] == operator {
					left := strings.TrimSpace(expr[:i])
					right := strings.TrimSpace(expr[i+len(operator):])
					return left, right, true
				}
			}
		}
	}
	
	return "", "", false
}

func main() {
	expr := "normalize-space(text())='' and not (*)"
	
	fmt.Printf("Testing splitBooleanExpression with: '%s' (len=%d)\n", expr, len(expr))
	
	left, right, found := splitBooleanExpression(expr, " and ")
	
	fmt.Printf("Found: %v\n", found)
	if found {
		fmt.Printf("Left: '%s' (len=%d)\n", left, len(left))
		fmt.Printf("Right: '%s' (len=%d)\n", right, len(right))
	}
	
	// Also test the naive split for comparison
	parts := strings.Split(expr, " and ")
	fmt.Printf("\nNaive split produces:\n")
	for i, part := range parts {
		fmt.Printf("Part %d: '%s' (len=%d)\n", i, strings.TrimSpace(part), len(strings.TrimSpace(part)))
	}
}