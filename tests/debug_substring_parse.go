package main

import (
	"fmt"
	"strings"
)

// Simplified parseSubstringArgs to test parsing
func parseSubstringArgs(argsStr string) []string {
	fmt.Printf("DEBUG: Parsing args: '%s'\n", argsStr)
	
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
			fmt.Printf("DEBUG: Found arg: '%s'\n", strings.TrimSpace(current))
			current = ""
		} else {
			current += string(c)
		}
	}
	
	if current != "" {
		args = append(args, strings.TrimSpace(current))
		fmt.Printf("DEBUG: Final arg: '%s'\n", strings.TrimSpace(current))
	}
	
	fmt.Printf("DEBUG: Total args: %d\n", len(args))
	return args
}

func main() {
	// Test the argument parsing for our failing case
	argsStr := "text(), string-length(text()) - 3"
	fmt.Println("Testing argument parsing for substring:")
	args := parseSubstringArgs(argsStr)
	
	fmt.Printf("\nParsed %d arguments:\n", len(args))
	for i, arg := range args {
		fmt.Printf("  args[%d] = '%s'\n", i, arg)
	}
	
	// Test the condition we're checking
	if len(args) > 1 {
		fmt.Printf("\nChecking if args[1] contains 'string-length(text())':\n")
		fmt.Printf("  args[1] = '%s'\n", args[1])
		fmt.Printf("  Contains 'string-length(text())'? %v\n", strings.Contains(args[1], "string-length(text())"))
	}
}