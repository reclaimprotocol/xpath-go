package main

import (
	"fmt"
)

func main() {
	expr := "normalize-space(text())='' and not (*)"
	
	fmt.Printf("Analyzing string: '%s' (len=%d)\n", expr, len(expr))
	fmt.Println("Character by character:")
	
	for i, char := range expr {
		if char == '\'' {
			fmt.Printf("  [%d]: '%c' (QUOTE)\n", i, char)
		} else if char == ' ' {
			fmt.Printf("  [%d]: ' ' (SPACE)\n", i)
		} else {
			fmt.Printf("  [%d]: '%c'\n", i, char)
		}
	}
	
	// Count quotes
	quoteCount := 0
	for _, char := range expr {
		if char == '\'' {
			quoteCount++
		}
	}
	
	fmt.Printf("\nTotal quotes: %d\n", quoteCount)
	fmt.Println("Expected for normalize-space(text())='': 2 quotes")
	
	// Expected string analysis
	expected := "normalize-space(text())=''"
	fmt.Printf("\nExpected left part: '%s' (len=%d)\n", expected, len(expected))
	expectedQuotes := 0
	for _, char := range expected {
		if char == '\'' {
			expectedQuotes++
		}
	}
	fmt.Printf("Expected quotes in left part: %d\n", expectedQuotes)
}