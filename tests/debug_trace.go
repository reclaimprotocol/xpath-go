package main

import (
	"fmt"
	"strings"
)

// Simulate the exact applyAndPredicate logic
func applyAndPredicate(nodes []map[string]string, expr string) []map[string]string {
	fmt.Printf("=== applyAndPredicate called with expr: '%s' ===\n", expr)

	parts := strings.Split(expr, " and ")
	fmt.Printf("Split parts: %v\n", parts)

	if len(parts) != 2 {
		fmt.Printf("Not exactly 2 parts, returning original nodes\n")
		return nodes
	}

	var filtered []map[string]string

	// Apply both conditions to each node
	for i, node := range nodes {
		fmt.Printf("\n--- Testing node %d: %v ---\n", i, node)

		firstCondition := strings.TrimSpace(parts[0])
		secondCondition := strings.TrimSpace(parts[1])

		fmt.Printf("First condition: '%s'\n", firstCondition)
		fmt.Printf("Second condition: '%s'\n", secondCondition)

		// Check first condition
		firstMatches := evaluateSimpleCondition(node, firstCondition)
		fmt.Printf("First matches: %v\n", firstMatches)
		if !firstMatches {
			fmt.Printf("First condition failed, skipping node\n")
			continue
		}

		// Check second condition
		secondMatches := evaluateSimpleCondition(node, secondCondition)
		fmt.Printf("Second matches: %v\n", secondMatches)
		if secondMatches {
			fmt.Printf("Both conditions match! Adding node\n")
			filtered = append(filtered, node)
		} else {
			fmt.Printf("Second condition failed\n")
		}
	}

	fmt.Printf("\nFiltered result: %d nodes\n", len(filtered))
	return filtered
}

func evaluateSimpleCondition(attributes map[string]string, condition string) bool {
	condition = strings.TrimSpace(condition)
	fmt.Printf("  evaluateSimpleCondition: '%s'\n", condition)

	// Attribute existence: @id
	if strings.HasPrefix(condition, "@") && !strings.Contains(condition, "=") {
		attrName := strings.TrimPrefix(condition, "@")
		fmt.Printf("  Checking attribute existence: '%s'\n", attrName)
		_, exists := attributes[attrName]
		fmt.Printf("  Exists: %v\n", exists)
		return exists
	}

	// Attribute value comparison: @id='test'
	if strings.HasPrefix(condition, "@") && strings.Contains(condition, "=") {
		parts := strings.SplitN(condition, "=", 2)
		if len(parts) != 2 {
			fmt.Printf("  Invalid attribute comparison format\n")
			return false
		}
		attrName := strings.TrimPrefix(strings.TrimSpace(parts[0]), "@")
		expectedValue := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		fmt.Printf("  Checking attribute value: '%s' == '%s'\n", attrName, expectedValue)

		if value, exists := attributes[attrName]; exists {
			fmt.Printf("  Actual value: '%s', matches: %v\n", value, value == expectedValue)
			return value == expectedValue
		}
		fmt.Printf("  Attribute doesn't exist\n")
		return false
	}

	fmt.Printf("  Condition not recognized\n")
	return false
}

func main() {
	// Test data
	nodes := []map[string]string{
		{"id": "test", "class": "highlight"}, // Should match
		{"id": "other"},                      // Should not match
		{"class": "highlight"},               // Should not match
	}

	fmt.Println("Testing AND predicate logic")
	fmt.Printf("Nodes: %v\n", nodes)

	result := applyAndPredicate(nodes, "@id and @class")
	fmt.Printf("\nFinal result: %v\n", result)
}
