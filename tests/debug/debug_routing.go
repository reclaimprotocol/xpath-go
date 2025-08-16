package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
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

func analyzeRouting(expr string) {
	fmt.Printf("=== Analyzing routing for: %s ===\n", expr)

	hasAnd := strings.Contains(expr, " and ")
	hasOr := strings.Contains(expr, " or ")
	hasParens := strings.Contains(expr, "(")
	hasFunctions := containsFunctionCall(expr)

	fmt.Printf("Contains ' and ': %v\n", hasAnd)
	fmt.Printf("Contains ' or ': %v\n", hasOr)
	fmt.Printf("Contains '(': %v\n", hasParens)
	fmt.Printf("Contains functions: %v\n", hasFunctions)

	isComplex := (hasAnd || hasOr) && (hasParens || hasFunctions)
	isSimpleAnd := hasAnd && !isComplex

	fmt.Printf("Should use Complex Boolean: %v\n", isComplex)
	fmt.Printf("Should use Simple AND: %v\n", isSimpleAnd)
	fmt.Println()
}

func main() {
	// Analyze different expressions to see routing
	expressions := []string{
		"span and not(a)",       // Our failing case
		"@id and @class",        // Working case
		"span and div",          // Simple element AND (no functions)
		"text() and not(a)",     // Function AND
		"(@id='test') and span", // Complex case
	}

	for _, expr := range expressions {
		analyzeRouting(expr)
	}

	// Now test if simple element AND (without functions) works
	fmt.Println("=== Testing simple element AND ===")
	html := `<li><span>text</span><div>content</div></li>`

	testCases := []struct {
		query       string
		expected    int
		description string
	}{
		{"//li[span]", 1, "Li with span"},
		{"//li[div]", 1, "Li with div"},
		{"//li[span and div]", 1, "Li with span AND div (no functions)"},
		{"//li[span and not(a)]", 1, "Li with span AND not(a) (has function)"},
	}

	for _, test := range testCases {
		fmt.Printf("%s: ", test.description)
		results, err := xpath.Query(test.query, html)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
		} else {
			success := len(results) == test.expected
			status := "✅"
			if !success {
				status = "❌"
			}
			fmt.Printf("%d (expected %d) %s\n", len(results), test.expected, status)
		}
	}
}
