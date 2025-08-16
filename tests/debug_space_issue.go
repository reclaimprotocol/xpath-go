package main

import (
	"fmt"
	"strings"
)

func parseNodeTest(nodeTest string) {
	fmt.Printf("Original nodeTest: '%s'\n", nodeTest)

	// Trim spaces first
	nodeTest = strings.TrimSpace(nodeTest)
	fmt.Printf("After TrimSpace: '%s'\n", nodeTest)

	// Check for brackets
	if strings.Contains(nodeTest, "[") && strings.Contains(nodeTest, "]") {
		fmt.Printf("Contains brackets - should call matchesNodeTestWithPredicate\n")

		bracketStart := strings.Index(nodeTest, "[")
		elementName := strings.TrimSpace(nodeTest[:bracketStart])
		predicateExpr := nodeTest[bracketStart:]

		fmt.Printf("elementName: '%s'\n", elementName)
		fmt.Printf("predicateExpr: '%s'\n", predicateExpr)

		if !strings.HasPrefix(predicateExpr, "[") || !strings.HasSuffix(predicateExpr, "]") {
			fmt.Printf("ERROR: Invalid predicate format\n")
			return
		}

		predicateContent := predicateExpr[1 : len(predicateExpr)-1] // Remove [ and ]
		fmt.Printf("predicateContent: '%s'\n", predicateContent)

		if strings.HasPrefix(predicateContent, "@") {
			fmt.Printf("Should use attribute predicate method\n")
		} else {
			fmt.Printf("Should use generic predicate method\n")
		}
	} else {
		fmt.Printf("No brackets - simple element name\n")
	}
}

func main() {
	fmt.Println("=== Testing nodeTest parsing with spaces ===")

	// Test cases based on our debug output
	testCases := []string{
		"div",                     // From step 4
		" div [@class='content']", // From step 5 (note the leading space)
		"div[@class='content']",   // What it should be
	}

	for i, test := range testCases {
		fmt.Printf("\n%d. Testing: '%s'\n", i+1, test)
		parseNodeTest(test)
	}
}
