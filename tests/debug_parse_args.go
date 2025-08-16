package main

import (
	"fmt"
	"reflect"
	"strings"
)

// Copy of parseSubstringArgs logic to debug
func parseSubstringArgs(argsStr string) []string {
	var args []string
	current := ""
	inQuotes := false
	quoteChar := byte(0)
	parenDepth := 0

	for i := 0; i < len(argsStr); i++ {
		c := argsStr[i]

		if !inQuotes && (c == '\'' || c == '"') {
			inQuotes = true
			quoteChar = c
			current += string(c)
		} else if inQuotes && c == quoteChar {
			inQuotes = false
			quoteChar = 0
			current += string(c)
		} else if !inQuotes && c == '(' {
			parenDepth++
			current += string(c)
		} else if !inQuotes && c == ')' {
			parenDepth--
			current += string(c)
		} else if !inQuotes && c == ',' && parenDepth == 0 {
			args = append(args, strings.TrimSpace(current))
			current = ""
		} else {
			current += string(c)
		}
	}

	if current != "" {
		args = append(args, strings.TrimSpace(current))
	}

	return args
}

func main() {
	fmt.Println("=== DEBUGGING ARGUMENT PARSING ===")
	fmt.Println()

	// Test the exact expression that's failing
	testExpr := "text(), string-length(text()) - 3"

	fmt.Printf("Input: %s\n", testExpr)

	args := parseSubstringArgs(testExpr)

	fmt.Printf("Parsed args: %s\n", reflect.ValueOf(args))
	fmt.Printf("Number of args: %d\n", len(args))

	for i, arg := range args {
		fmt.Printf("  Arg %d: '%s'\n", i, arg)
	}

	fmt.Println()
	fmt.Println("Expected:")
	fmt.Println("  Arg 0: 'text()'")
	fmt.Println("  Arg 1: 'string-length(text()) - 3'")

	if len(args) >= 2 {
		fmt.Printf("\nArg 1 analysis: '%s'\n", args[1])
		fmt.Printf("Contains 'string-length(text())': %t\n", strings.Contains(args[1], "string-length(text())"))
		fmt.Printf("Contains ' - ': %t\n", strings.Contains(args[1], " - "))

		if strings.Contains(args[1], " - ") {
			parts := strings.Split(args[1], " - ")
			fmt.Printf("Split by ' - ': %v\n", parts)
		}
	}
}
