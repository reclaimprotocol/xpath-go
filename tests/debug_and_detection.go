package main

import (
	"fmt"
	"strings"
)

func main() {
	testCases := []string{
		"ancestor:: div",
		"ancestor:: div [@class='content']",
		"@id and @class",
		"@id='test' and @class='content'",
	}

	for _, expr := range testCases {
		fmt.Printf("Expression: '%s'\n", expr)
		fmt.Printf("  Contains ' and ': %v\n", strings.Contains(expr, " and "))
		fmt.Printf("  Contains ' or ': %v\n", strings.Contains(expr, " or "))
		fmt.Println()
	}
}
