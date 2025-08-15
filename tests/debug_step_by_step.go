package main

import (
	"fmt"
	"strings"
)

// Simulate the evaluateSimpleCondition logic
func evaluateSimpleCondition(attributes map[string]string, condition string) bool {
	condition = strings.TrimSpace(condition)
	fmt.Printf("  Testing condition: '%s'\n", condition)

	// Attribute existence: @id
	if strings.HasPrefix(condition, "@") && !strings.Contains(condition, "=") {
		attrName := strings.TrimPrefix(condition, "@")
		fmt.Printf("  Attribute existence check for: '%s'\n", attrName)
		_, exists := attributes[attrName]
		fmt.Printf("  Result: %v\n", exists)
		return exists
	}

	fmt.Printf("  Condition not recognized\n")
	return false
}

func main() {
	// Test node attributes
	attributes := map[string]string{
		"id":    "test",
		"class": "active",
	}
	
	fmt.Printf("Node attributes: %+v\n", attributes)
	fmt.Println()
	
	// Test individual conditions
	fmt.Println("=== Testing @id ===")
	result1 := evaluateSimpleCondition(attributes, "@id")
	fmt.Printf("Final result: %v\n", result1)
	fmt.Println()
	
	fmt.Println("=== Testing @class ===")
	result2 := evaluateSimpleCondition(attributes, "@class")
	fmt.Printf("Final result: %v\n", result2)
	fmt.Println()
	
	// Test AND logic
	fmt.Println("=== Testing AND logic ===")
	expr := "@id and @class"
	parts := strings.Split(expr, " and ")
	fmt.Printf("Split into: %v\n", parts)
	
	firstCondition := strings.TrimSpace(parts[0])
	secondCondition := strings.TrimSpace(parts[1])
	
	fmt.Println("--- First condition ---")
	first := evaluateSimpleCondition(attributes, firstCondition)
	
	fmt.Println("--- Second condition ---")
	second := evaluateSimpleCondition(attributes, secondCondition)
	
	fmt.Printf("AND result: %v && %v = %v\n", first, second, first && second)
}