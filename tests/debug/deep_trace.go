package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
	"strings"
)

// Replicate the exact logic to trace execution
func simulateComplexBooleanEvaluation(expr string, html string) {
	fmt.Printf("=== Deep Trace: %s ===\n", expr)

	// Step 1: Check routing
	hasAnd := strings.Contains(expr, " and ")
	hasOr := strings.Contains(expr, " or ")
	hasParens := strings.Contains(expr, "(")
	hasFunctions := containsFunctionCall(expr)

	fmt.Printf("Routing analysis:\n")
	fmt.Printf("  Has AND: %v\n", hasAnd)
	fmt.Printf("  Has OR: %v\n", hasOr)
	fmt.Printf("  Has parens: %v\n", hasParens)
	fmt.Printf("  Has functions: %v\n", hasFunctions)

	isComplex := (hasAnd || hasOr) && (hasParens || hasFunctions)
	fmt.Printf("  → Uses complex boolean: %v\n", isComplex)

	if !isComplex {
		fmt.Printf("  → Should use simple AND instead\n")
		return
	}

	// Step 2: Split boolean expression
	op, left, right := findMainBooleanOperator(expr)
	fmt.Printf("\nBoolean splitting:\n")
	fmt.Printf("  Operator: '%s'\n", op)
	fmt.Printf("  Left: '%s'\n", left)
	fmt.Printf("  Right: '%s'\n", right)

	if op == "" {
		fmt.Printf("  → No operator found, would evaluate as simple condition\n")
		return
	}

	// Step 3: Test individual evaluations
	fmt.Printf("\nTesting individual conditions:\n")

	// Test left condition
	leftQuery := fmt.Sprintf("//li[%s]", left)
	leftResults, err := xpath.Query(leftQuery, html)
	if err != nil {
		fmt.Printf("  Left ERROR: %v\n", err)
	} else {
		fmt.Printf("  Left '%s': %d results\n", left, len(leftResults))
	}

	// Test right condition
	rightQuery := fmt.Sprintf("//li[%s]", right)
	rightResults, err := xpath.Query(rightQuery, html)
	if err != nil {
		fmt.Printf("  Right ERROR: %v\n", err)
	} else {
		fmt.Printf("  Right '%s': %d results\n", right, len(rightResults))
	}

	// Step 4: Test the combination
	fmt.Printf("\nTesting combination:\n")
	combinedQuery := fmt.Sprintf("//li[%s]", expr)
	combinedResults, err := xpath.Query(combinedQuery, html)
	if err != nil {
		fmt.Printf("  Combined ERROR: %v\n", err)
	} else {
		fmt.Printf("  Combined '%s': %d results\n", expr, len(combinedResults))
	}

	// Step 5: Analysis
	expectedCombined := 0
	if len(leftResults) > 0 && len(rightResults) > 0 && op == "and" {
		expectedCombined = 1 // Should find intersection
	}

	fmt.Printf("\nExpected logic:\n")
	fmt.Printf("  Left results > 0: %v\n", len(leftResults) > 0)
	fmt.Printf("  Right results > 0: %v\n", len(rightResults) > 0)
	fmt.Printf("  Expected combined (AND): %d\n", expectedCombined)
	fmt.Printf("  Actual combined: %d\n", len(combinedResults))

	if len(combinedResults) == expectedCombined {
		fmt.Printf("  ✅ Logic is working correctly\n")
	} else {
		fmt.Printf("  ❌ Logic is broken\n")
	}
}

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

func findMainBooleanOperator(expr string) (string, string, string) {
	parenDepth := 0

	// Look for 'and' operator outside parentheses
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
	html := `<li><span>Item 1</span></li>`

	// Test both working and failing cases
	expressions := []string{
		"span and div",    // Working: simple AND
		"span and not(a)", // Failing: complex AND with not()
		"@id and @class",  // Working: attribute AND
	}

	for _, expr := range expressions {
		simulateComplexBooleanEvaluation(expr, html)
		fmt.Println(strings.Repeat("=", 60))
	}
}
