package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
	"github.com/reclaimprotocol/xpath-go/pkg/types"
	"strings"
)

// Replicate the exact evaluateSimpleCondition logic to trace it
func debugEvaluateSimpleCondition(node *types.Node, condition string) bool {
	fmt.Printf("  → debugEvaluateSimpleCondition called with condition: '%s'\n", condition)
	condition = strings.TrimSpace(condition)
	fmt.Printf("  → After trimming: '%s'\n", condition)

	// Not function: not(*)
	if strings.HasPrefix(condition, "not(") && strings.HasSuffix(condition, ")") {
		fmt.Printf("  → Detected not() function\n")

		// Extract the condition inside not()
		var innerCondition string
		if strings.HasPrefix(condition, "not(") && strings.HasSuffix(condition, ")") {
			innerCondition = condition[4 : len(condition)-1] // Remove "not(" and ")"
			fmt.Printf("  → Extracted inner condition (method 1): '%s'\n", innerCondition)
		} else if strings.HasPrefix(condition, "not (") && strings.HasSuffix(condition, ")") {
			innerCondition = condition[5 : len(condition)-1] // Remove "not (" and ")"
			fmt.Printf("  → Extracted inner condition (method 2): '%s'\n", innerCondition)
		} else {
			fmt.Printf("  → Failed to extract inner condition\n")
			return false
		}
		innerCondition = strings.TrimSpace(innerCondition)
		fmt.Printf("  → Inner condition after trimming: '%s'\n", innerCondition)

		// For our test case, innerCondition should be "a"
		if innerCondition != "a" {
			fmt.Printf("  → ERROR: Expected inner condition 'a', got '%s'\n", innerCondition)
			return false
		}

		// Check if node has any child elements of type "a"
		fmt.Printf("  → Checking if node has child elements of type 'a'\n")
		fmt.Printf("  → Node has %d children\n", len(node.Children))

		for i, child := range node.Children {
			fmt.Printf("  → Child %d: Type=%d, Name='%s'\n", i, child.Type, child.Name)
			if child.Type == types.ElementNode && strings.ToLower(child.Name) == "a" {
				fmt.Printf("  → Found 'a' element! Returning false (because this is not())\n")
				return false
			}
		}

		fmt.Printf("  → No 'a' elements found. Returning true (because this is not())\n")
		return true
	}

	// Child element existence: span, a, div, etc.
	if isSimpleElementName(condition) {
		fmt.Printf("  → Detected simple element name: '%s'\n", condition)
		return hasChildElement(node, condition)
	}

	fmt.Printf("  → Condition not recognized, returning false\n")
	return false
}

func isSimpleElementName(name string) bool {
	// Simple validation for element names
	if len(name) == 0 {
		return false
	}

	// Check if it contains any special characters that would indicate it's not a simple element name
	if strings.Contains(name, "@") || strings.Contains(name, "(") || strings.Contains(name, ")") ||
		strings.Contains(name, "=") || strings.Contains(name, "[") || strings.Contains(name, "]") ||
		strings.Contains(name, "/") || strings.Contains(name, ":") {
		return false
	}

	return true
}

func hasChildElement(node *types.Node, elementName string) bool {
	fmt.Printf("  → hasChildElement: looking for '%s' in %d children\n", elementName, len(node.Children))
	for i, child := range node.Children {
		fmt.Printf("  → Child %d: Type=%d, Name='%s'\n", i, child.Type, child.Name)
		if child.Type == types.ElementNode && strings.ToLower(child.Name) == strings.ToLower(elementName) {
			fmt.Printf("  → Found element '%s'! Returning true\n", elementName)
			return true
		}
	}
	fmt.Printf("  → Element '%s' not found. Returning false\n", elementName)
	return false
}

func main() {
	html := `<li><span>Item 1</span></li>`

	fmt.Println("=== Debug Simple Condition Evaluation ===")
	fmt.Printf("HTML: %s\n", html)
	fmt.Println()

	// Get the li node
	liNodes, err := xpath.Query("//li", html)
	if err != nil || len(liNodes) == 0 {
		fmt.Printf("ERROR: Could not get li node: %v\n", err)
		return
	}

	node := liNodes[0]
	fmt.Printf("Li node: Type=%d, Name='%s', Children=%d\n", node.Type, node.Name, len(node.Children))
	fmt.Println()

	// Test the individual conditions
	fmt.Println("1. Testing condition 'span':")
	spanResult := debugEvaluateSimpleCondition(node, "span")
	fmt.Printf("   Result: %v\n\n", spanResult)

	fmt.Println("2. Testing condition 'not(a)':")
	notAResult := debugEvaluateSimpleCondition(node, "not(a)")
	fmt.Printf("   Result: %v\n\n", notAResult)

	fmt.Printf("=== Expected Combined Result ===\n")
	fmt.Printf("span (%v) AND not(a) (%v) = %v\n", spanResult, notAResult, spanResult && notAResult)

	// Now test the actual xpath to see if it matches our expectation
	fmt.Println("\n=== Actual XPath Test ===")
	actualResults, err := xpath.Query("//li[span and not(a)]", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("Actual //li[span and not(a)] results: %d\n", len(actualResults))
		if len(actualResults) == 1 && spanResult && notAResult {
			fmt.Println("✅ Results match expectations!")
		} else {
			fmt.Println("❌ Results do NOT match expectations!")
		}
	}
}
