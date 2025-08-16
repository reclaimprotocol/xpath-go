package main

import (
	"fmt"
	"strings"
)

// Simulate the AND logic to debug
func debugAndLogic(expr string) {
	fmt.Printf("=== Debugging AND logic for: %s ===\n", expr)

	// Simulate applyAndPredicate splitting
	parts := strings.Split(expr, " and ")
	if len(parts) != 2 {
		fmt.Printf("ERROR: Expected 2 parts, got %d\n", len(parts))
		return
	}

	firstCondition := strings.TrimSpace(parts[0])
	secondCondition := strings.TrimSpace(parts[1])

	fmt.Printf("First condition: '%s'\n", firstCondition)
	fmt.Printf("Second condition: '%s'\n", secondCondition)

	// Check what type of conditions these are
	fmt.Printf("First condition analysis:\n")
	analyzeCondition(firstCondition)

	fmt.Printf("Second condition analysis:\n")
	analyzeCondition(secondCondition)
}

func analyzeCondition(condition string) {
	fmt.Printf("  Condition: '%s'\n", condition)
	fmt.Printf("  Starts with @: %v\n", strings.HasPrefix(condition, "@"))
	fmt.Printf("  Starts with text(): %v\n", strings.HasPrefix(condition, "text()"))
	fmt.Printf("  Starts with not(: %v\n", strings.HasPrefix(condition, "not("))
	fmt.Printf("  Contains (: %v\n", strings.Contains(condition, "("))
	fmt.Printf("  Is simple element name: %v\n", isSimpleElementName(condition))
	fmt.Println()
}

func isSimpleElementName(condition string) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return false
	}

	// Check for disqualifying characters
	if strings.ContainsAny(condition, "@()[]='\"<>!&|+ ") {
		return false
	}

	// Must start with a letter
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

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func main() {
	debugAndLogic("span and not(a)")
	debugAndLogic("@id and @class")
}
