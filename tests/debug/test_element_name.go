package main

import (
	"fmt"
	"strings"
)

// isLetter checks if a byte is a letter
func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// isDigit checks if a byte is a digit
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// isSimpleElementName checks if a condition is a simple element name (like span, a, div)
func isSimpleElementName(condition string) bool {
	// Must be a simple identifier without spaces, operators, or special characters
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return false
	}
	
	// Check for disqualifying characters that indicate it's not a simple element name
	if strings.ContainsAny(condition, "@()[]='\"<>!&|+ ") {
		return false
	}
	
	// Must start with a letter and contain only letters, numbers, hyphens, and underscores
	if len(condition) == 0 || (!isLetter(condition[0]) && condition[0] != '_') {
		return false
	}
	
	for _, char := range condition {
		if !isLetter(byte(char)) && !isDigit(byte(char)) && char != '-' && char != '_' && char != ':' {
			return false
		}
	}
	
	return true
}

func main() {
	testCases := []string{"span", "a", "div", "@id", "text()", "contains(text(), 'test')", "span and not(a)"}
	
	for _, test := range testCases {
		fmt.Printf("'%s' -> isSimpleElementName: %t\n", test, isSimpleElementName(test))
	}
}