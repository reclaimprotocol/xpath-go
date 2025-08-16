package main

import (
	"fmt"
	"strings"
)

func findMainBooleanOperator(expr string) (string, string, string) {
	parenDepth := 0

	// Look for 'and' operator outside parentheses (AND has higher precedence)
	for i := 0; i < len(expr); i++ {
		if expr[i] == '(' {
			parenDepth++
		} else if expr[i] == ')' {
			parenDepth--
		} else if parenDepth == 0 && i+5 <= len(expr) && expr[i:i+5] == " and " {
			leftExpr := strings.TrimSpace(expr[:i])
			rightExpr := strings.TrimSpace(expr[i+5:])
			return "and", leftExpr, rightExpr
		}
	}

	// Look for 'or' operator outside parentheses
	parenDepth = 0
	for i := 0; i < len(expr); i++ {
		if expr[i] == '(' {
			parenDepth++
		} else if expr[i] == ')' {
			parenDepth--
		} else if parenDepth == 0 && i+4 <= len(expr) && expr[i:i+4] == " or " {
			leftExpr := strings.TrimSpace(expr[:i])
			rightExpr := strings.TrimSpace(expr[i+4:])
			return "or", leftExpr, rightExpr
		}
	}

	return "", "", ""
}

func main() {
	expr := "normalize-space(text())='' and not(*)"
	op, left, right := findMainBooleanOperator(expr)
	fmt.Printf("Expression: %s\n", expr)
	fmt.Printf("Operator: '%s'\n", op)
	fmt.Printf("Left: '%s'\n", left)
	fmt.Printf("Right: '%s'\n", right)
}
