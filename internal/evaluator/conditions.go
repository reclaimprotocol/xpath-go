package evaluator

import (
	"strconv"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// conditions.go - XPath condition evaluation logic
// Functions that evaluate individual conditions against nodes

// evaluateComplexBooleanExpression evaluates complex boolean expressions with and/or
func (e *Evaluator) evaluateComplexBooleanExpression(expr string, node *types.Node) bool {
	expr = strings.TrimSpace(expr)

	// Find the main boolean operator outside parentheses
	mainOperator, leftExpr, rightExpr := e.findMainBooleanOperator(expr)

	if mainOperator == "" {
		// No main operator found, evaluate as simple expression
		return e.evaluateSimpleCondition(node, expr)
	}

	leftResult := false
	rightResult := false

	// Evaluate left expression
	if strings.HasPrefix(leftExpr, "(") && strings.HasSuffix(leftExpr, ")") {
		// Remove parentheses and evaluate the inner expression
		innerExpr := leftExpr[1 : len(leftExpr)-1]
		leftResult = e.evaluateComplexBooleanExpression(innerExpr, node)
	} else if strings.Contains(leftExpr, " and ") || strings.Contains(leftExpr, " or ") {
		// Left expression contains boolean operators, evaluate recursively
		leftResult = e.evaluateComplexBooleanExpression(leftExpr, node)
	} else {
		leftResult = e.evaluateSimpleCondition(node, leftExpr)
	}

	// Evaluate right expression
	if strings.HasPrefix(rightExpr, "(") && strings.HasSuffix(rightExpr, ")") {
		// Remove parentheses and evaluate the inner expression
		innerExpr := rightExpr[1 : len(rightExpr)-1]
		rightResult = e.evaluateComplexBooleanExpression(innerExpr, node)
	} else if strings.Contains(rightExpr, " and ") || strings.Contains(rightExpr, " or ") {
		// Right expression contains boolean operators, evaluate recursively
		rightResult = e.evaluateComplexBooleanExpression(rightExpr, node)
	} else {
		rightResult = e.evaluateSimpleCondition(node, rightExpr)
	}

	// Apply the operator
	switch mainOperator {
	case "and":
		return leftResult && rightResult
	case "or":
		return leftResult || rightResult
	default:
		return false
	}
}

// findMainBooleanOperator finds the main AND/OR operator outside parentheses
func (e *Evaluator) findMainBooleanOperator(expr string) (string, string, string) {
	parenDepth := 0

	// Look for 'and' operator outside parentheses (AND has higher precedence)
	for i := 0; i < len(expr)-4; i++ {
		c := expr[i]
		if c == '(' {
			parenDepth++
		} else if c == ')' {
			parenDepth--
		} else if parenDepth == 0 && expr[i:i+5] == " and " {
			left := strings.TrimSpace(expr[:i])
			right := strings.TrimSpace(expr[i+5:])
			return "and", left, right
		}
	}

	// Reset and look for 'or' operator
	parenDepth = 0
	for i := 0; i < len(expr)-3; i++ {
		c := expr[i]
		if c == '(' {
			parenDepth++
		} else if c == ')' {
			parenDepth--
		} else if parenDepth == 0 && expr[i:i+4] == " or " {
			left := strings.TrimSpace(expr[:i])
			right := strings.TrimSpace(expr[i+4:])
			return "or", left, right
		}
	}

	return "", "", ""
}

// evaluateSimpleCondition evaluates a simple condition against a single node
func (e *Evaluator) evaluateSimpleCondition(node *types.Node, condition string) bool {
	condition = strings.TrimSpace(condition)

	// Simple condition evaluation

	// Attribute existence: @id (handle spaced tokens like "@ id")
	if strings.HasPrefix(condition, "@") && !strings.Contains(condition, "=") {
		attrName := strings.TrimPrefix(condition, "@")
		attrName = strings.TrimSpace(attrName) // Handle "@ id" -> "id"
		_, exists := node.Attributes[attrName]
		return exists
	}

	// Attribute value comparison: @id='value' or @id = 'value'
	if strings.HasPrefix(condition, "@") && (strings.Contains(condition, "=") || strings.Contains(condition, "!=")) {
		if strings.Contains(condition, "!=") {
			parts := strings.SplitN(condition, "!=", 2)
			if len(parts) == 2 {
				attrName := strings.TrimSpace(strings.TrimPrefix(parts[0], "@"))
				expectedValue := strings.Trim(strings.TrimSpace(parts[1]), "'\"")

				if actualValue, exists := node.Attributes[attrName]; exists {
					return actualValue != expectedValue
				}
				return true // Attribute doesn't exist, so it's != any value
			}
		} else {
			parts := strings.SplitN(condition, "=", 2)
			if len(parts) == 2 {
				attrName := strings.TrimSpace(strings.TrimPrefix(parts[0], "@"))
				expectedValue := strings.Trim(strings.TrimSpace(parts[1]), "'\"")

				if actualValue, exists := node.Attributes[attrName]; exists {
					return actualValue == expectedValue
				}
				return false
			}
		}
	}

	// Node test: node() - returns true if there are any child nodes
	if condition == "node()" {
		// node() matches any child node (element, text, comment, etc.)
		// Check if node has any children or any non-empty text content
		if len(node.Children) > 0 {
			return true
		}
		// Also check for text content (text nodes)
		if strings.TrimSpace(node.TextContent) != "" {
			return true
		}
		return false
	}

	// Text content existence: text()
	if condition == "text()" {
		return strings.TrimSpace(node.TextContent) != ""
	}

	// Text content comparison: text()='value'
	if strings.Contains(condition, "text()") && strings.Contains(condition, "=") {
		if strings.Contains(condition, "!=") {
			parts := strings.SplitN(condition, "!=", 2)
			if len(parts) == 2 {
				expectedValue := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				return node.TextContent != expectedValue
			}
		} else {
			parts := strings.SplitN(condition, "=", 2)
			if len(parts) == 2 {
				expectedValue := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				return node.TextContent == expectedValue
			}
		}
	}

	// Position function: position()=2
	if strings.Contains(condition, "position()") && strings.Contains(condition, "=") {
		// Position evaluation is context-dependent and should be handled by caller
		return false
	}

	// Last function: last()
	if condition == "last()" {
		// Last evaluation is context-dependent and should be handled by caller
		return false
	}

	// Contains function: contains(text(), 'value') or contains(@attr, 'value')
	if strings.Contains(condition, "contains(") {
		return e.evaluateContainsExpression(condition, node)
	}

	// Starts-with function: starts-with(text(), 'value')
	if strings.Contains(condition, "starts-with(") {
		return e.evaluateStartsWithExpression(condition, node)
	}

	// String-length function: string-length(text())>10
	if strings.Contains(condition, "string-length(") {
		return e.evaluateStringLengthExpression(condition, node)
	}

	// Count function: count(li)=3
	if strings.Contains(condition, "count(") {
		return e.evaluateCountExpression(condition, node)
	}

	// Normalize-space function: normalize-space(text()) = 'value'
	if strings.Contains(condition, "normalize-space(") {
		return e.evaluateNormalizeSpaceExpression(condition, node)
	}

	// Substring function: substring(text(), 1, 3) = 'Fir'
	if strings.Contains(condition, "substring(") {
		return e.evaluateSubstringExpression(condition, node)
	}

	// Not function: not(condition)
	if strings.HasPrefix(condition, "not(") || strings.HasPrefix(condition, "not (") {
		return e.evaluateNotExpression(condition, node)
	}

	// Axis expressions: parent::div, ancestor::table, etc.
	if strings.Contains(condition, "::") {
		return e.evaluateAxisExpression(node, condition)
	}

	// Node test with predicate: span[@class='loading']
	if strings.Contains(condition, "[") && strings.Contains(condition, "]") {
		return e.evaluateNestedElementCondition(node, condition)
	}

	// Child path expressions: head/title, head/meta[@charset]
	if e.isChildPathExpression(condition) {
		return e.evaluateChildPath(node, condition)
	}

	// Simple element existence
	if e.isSimpleElementName(condition) {
		return e.hasChildElement(node, condition)
	}

	// Default: try to match as element name
	return node.Name == condition
}

// evaluateNestedElementCondition evaluates nested element conditions like span[@class='loading']
func (e *Evaluator) evaluateNestedElementCondition(node *types.Node, condition string) bool {
	idx := strings.Index(condition, "[")
	if idx == -1 {
		return false
	}

	elementName := strings.TrimSpace(condition[:idx])
	predicate := strings.TrimSpace(condition[idx+1:])

	// Remove closing bracket
	if strings.HasSuffix(predicate, "]") {
		predicate = predicate[:len(predicate)-1]
	}

	// Check if any child matches the element name and predicate
	for _, child := range node.Children {
		if child.Name == elementName {
			if e.evaluateSimpleCondition(child, predicate) {
				return true
			}
		}
	}

	return false
}

// evaluateAxisExpression evaluates axis expressions like parent::div, ancestor::table
func (e *Evaluator) evaluateAxisExpression(node *types.Node, axisExpr string) bool {
	parts := strings.Split(axisExpr, "::")
	if len(parts) != 2 {
		return false
	}

	axis := strings.TrimSpace(parts[0])
	nodeTest := strings.TrimSpace(parts[1])

	switch axis {
	case "parent":
		if node.Parent != nil {
			return e.matchesNodeTest(node.Parent, nodeTest)
		}
		return false

	case "ancestor":
		current := node.Parent
		for current != nil {
			if e.matchesNodeTest(current, nodeTest) {
				return true
			}
			current = current.Parent
		}
		return false

	case "ancestor-or-self":
		// Check self first
		if e.matchesNodeTest(node, nodeTest) {
			return true
		}
		// Then check ancestors
		current := node.Parent
		for current != nil {
			if e.matchesNodeTest(current, nodeTest) {
				return true
			}
			current = current.Parent
		}
		return false

	case "following-sibling":
		if node.Parent == nil {
			return false
		}
		found := false
		for _, sibling := range node.Parent.Children {
			if found && e.matchesNodeTest(sibling, nodeTest) {
				return true
			}
			if sibling == node {
				found = true
			}
		}
		return false

	case "preceding-sibling":
		if node.Parent == nil {
			return false
		}
		for _, sibling := range node.Parent.Children {
			if sibling == node {
				break
			}
			if e.matchesNodeTest(sibling, nodeTest) {
				return true
			}
		}
		return false

	case "self":
		return e.matchesNodeTest(node, nodeTest)

	default:
		return false
	}
}

// matchesNodeTest checks if a node matches a node test
func (e *Evaluator) matchesNodeTest(node *types.Node, nodeTest string) bool {
	if nodeTest == "*" {
		return node.Type == types.ElementNode
	}

	if strings.Contains(nodeTest, "[") {
		return e.matchesNodeTestWithPredicate(node, nodeTest)
	}

	return node.Name == nodeTest
}

// matchesNodeTestWithPredicate checks if a node matches a node test with predicate
func (e *Evaluator) matchesNodeTestWithPredicate(node *types.Node, nodeTest string) bool {
	idx := strings.Index(nodeTest, "[")
	if idx == -1 {
		return node.Name == nodeTest
	}

	elementName := strings.TrimSpace(nodeTest[:idx])
	predicate := strings.TrimSpace(nodeTest[idx+1:])

	// Remove closing bracket
	if strings.HasSuffix(predicate, "]") {
		predicate = predicate[:len(predicate)-1]
	}

	// Check element name match
	if elementName != "*" && node.Name != elementName {
		return false
	}

	// Evaluate predicate
	return e.evaluateSimpleCondition(node, predicate)
}

// evaluateContainsExpression evaluates contains() function expressions
func (e *Evaluator) evaluateContainsExpression(expr string, node *types.Node) bool {
	// Parse contains(text(), 'value') or contains(@attr, 'value')
	start := strings.Index(expr, "contains(")
	if start == -1 {
		return false
	}

	// Find matching closing parenthesis
	depth := 0
	var end int
	for i := start + 9; i < len(expr); i++ {
		if expr[i] == '(' {
			depth++
		} else if expr[i] == ')' {
			if depth == 0 {
				end = i
				break
			}
			depth--
		}
	}

	if end == 0 {
		return false
	}

	// Extract arguments
	args := expr[start+9 : end]
	parts := strings.Split(args, ",")
	if len(parts) != 2 {
		return false
	}

	source := strings.TrimSpace(parts[0])
	searchText := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

	var textToSearch string
	if source == "text()" {
		textToSearch = node.TextContent
	} else if strings.HasPrefix(source, "@") {
		attrName := strings.TrimPrefix(source, "@")
		if value, exists := node.Attributes[attrName]; exists {
			textToSearch = value
		}
	}

	return strings.Contains(textToSearch, searchText)
}

// evaluateStartsWithExpression evaluates starts-with() function expressions
func (e *Evaluator) evaluateStartsWithExpression(expr string, node *types.Node) bool {
	// Parse starts-with(text(), 'prefix') or starts-with(@attr, 'prefix')
	start := strings.Index(expr, "starts-with(")
	if start == -1 {
		return false
	}

	// Find matching closing parenthesis
	depth := 0
	var end int
	for i := start + 12; i < len(expr); i++ {
		if expr[i] == '(' {
			depth++
		} else if expr[i] == ')' {
			if depth == 0 {
				end = i
				break
			}
			depth--
		}
	}

	if end == 0 {
		return false
	}

	// Extract arguments
	args := expr[start+12 : end]
	parts := strings.Split(args, ",")
	if len(parts) != 2 {
		return false
	}

	source := strings.TrimSpace(parts[0])
	prefix := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

	var textToCheck string
	if source == "text()" {
		textToCheck = node.TextContent
	} else if strings.HasPrefix(source, "@") {
		attrName := strings.TrimPrefix(source, "@")
		if value, exists := node.Attributes[attrName]; exists {
			textToCheck = value
		}
	}

	return strings.HasPrefix(textToCheck, prefix)
}

// evaluateStringLengthExpression evaluates string-length() function expressions
func (e *Evaluator) evaluateStringLengthExpression(expr string, node *types.Node) bool {
	// Parse string-length(text())>10 or string-length(@attr)>5
	start := strings.Index(expr, "string-length(")
	if start == -1 {
		return false
	}

	// Find matching closing parenthesis
	depth := 0
	var end int
	for i := start + 14; i < len(expr); i++ {
		if expr[i] == '(' {
			depth++
		} else if expr[i] == ')' {
			if depth == 0 {
				end = i
				break
			}
			depth--
		}
	}

	if end == 0 {
		return false
	}

	// Get comparison part
	comparison := strings.TrimSpace(expr[end+1:])
	if comparison == "" {
		return false
	}

	// Extract source
	args := expr[start+14 : end]
	source := strings.TrimSpace(args)

	var textToMeasure string
	if source == "text()" {
		textToMeasure = node.TextContent
	} else if strings.HasPrefix(source, "@") {
		attrName := strings.TrimPrefix(source, "@")
		if value, exists := node.Attributes[attrName]; exists {
			textToMeasure = value
		}
	} else if strings.HasPrefix(source, "normalize-space(") {
		// Handle nested normalize-space function call
		textToMeasure = e.evaluateNormalizeSpaceFunction(source, node)
	}

	actualLength := len(textToMeasure)

	// Parse comparison
	if strings.HasPrefix(comparison, ">") {
		if targetLength, err := strconv.Atoi(strings.TrimSpace(comparison[1:])); err == nil {
			return actualLength > targetLength
		}
	} else if strings.HasPrefix(comparison, "<") {
		if targetLength, err := strconv.Atoi(strings.TrimSpace(comparison[1:])); err == nil {
			return actualLength < targetLength
		}
	} else if strings.HasPrefix(comparison, "=") {
		if targetLength, err := strconv.Atoi(strings.TrimSpace(comparison[1:])); err == nil {
			return actualLength == targetLength
		}
	}

	return false
}

// evaluateNormalizeSpaceExpression evaluates normalize-space() function expressions
func (e *Evaluator) evaluateNormalizeSpaceExpression(expr string, node *types.Node) bool {
	// Parse normalize-space(text()) = 'value'
	start := strings.Index(expr, "normalize-space(")
	if start == -1 {
		return false
	}

	// Find matching closing parenthesis
	depth := 0
	var end int
	for i := start + 16; i < len(expr); i++ {
		if expr[i] == '(' {
			depth++
		} else if expr[i] == ')' {
			if depth == 0 {
				end = i
				break
			}
			depth--
		}
	}

	if end == 0 {
		return false
	}

	// Get comparison part
	comparison := strings.TrimSpace(expr[end+1:])

	// Extract and normalize text
	normalizedText := e.evaluateNormalizeSpaceFunction(expr[start:end+1], node)

	// Parse comparison
	if strings.HasPrefix(comparison, " = ") || strings.HasPrefix(comparison, "=") {
		var expectedValue string
		if strings.HasPrefix(comparison, " = ") {
			expectedValue = strings.Trim(strings.TrimSpace(comparison[3:]), "'\"")
		} else {
			expectedValue = strings.Trim(strings.TrimSpace(comparison[1:]), "'\"")
		}
		return normalizedText == expectedValue
	}

	return false
}

// evaluateCountExpression evaluates count() function expressions
func (e *Evaluator) evaluateCountExpression(expr string, node *types.Node) bool {
	// Parse count(li)=3 or count(*)>2
	start := strings.Index(expr, "count(")
	if start == -1 {
		return false
	}

	// Find matching closing parenthesis
	depth := 0
	var end int
	for i := start + 6; i < len(expr); i++ {
		if expr[i] == '(' {
			depth++
		} else if expr[i] == ')' {
			if depth == 0 {
				end = i
				break
			}
			depth--
		}
	}

	if end == 0 {
		return false
	}

	// Get selector and comparison
	selector := strings.TrimSpace(expr[start+6 : end])
	comparison := strings.TrimSpace(expr[end+1:])

	// Count matching children
	actualCount := e.countChildElements(node, selector)

	// Parse comparison
	if strings.HasPrefix(comparison, "=") {
		if targetCount, err := strconv.Atoi(strings.TrimSpace(comparison[1:])); err == nil {
			return actualCount == targetCount
		}
	} else if strings.HasPrefix(comparison, ">") {
		if targetCount, err := strconv.Atoi(strings.TrimSpace(comparison[1:])); err == nil {
			return actualCount > targetCount
		}
	} else if strings.HasPrefix(comparison, "<") {
		if targetCount, err := strconv.Atoi(strings.TrimSpace(comparison[1:])); err == nil {
			return actualCount < targetCount
		}
	}

	return false
}

// evaluateNotExpression evaluates not() function expressions
func (e *Evaluator) evaluateNotExpression(expr string, node *types.Node) bool {
	// Parse not(condition) or not (condition)
	var start int
	if strings.HasPrefix(expr, "not(") {
		start = 4
	} else if strings.HasPrefix(expr, "not (") {
		start = 5
	} else {
		return false
	}

	// Find matching closing parenthesis
	depth := 0
	var end int
	for i := start; i < len(expr); i++ {
		if expr[i] == '(' {
			depth++
		} else if expr[i] == ')' {
			if depth == 0 {
				end = i
				break
			}
			depth--
		}
	}

	if end == 0 || end <= start {
		return false
	}

	// Extract inner condition
	innerCondition := strings.TrimSpace(expr[start:end])

	// Evaluate inner condition and return negation
	return !e.evaluateSimpleCondition(node, innerCondition)
}

// evaluateAndExpression evaluates expressions with 'and' operator
func (e *Evaluator) evaluateAndExpression(expr string, node *types.Node) bool {
	parts := strings.Split(expr, " and ")
	if len(parts) != 2 {
		return false
	}

	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	return e.evaluateSimpleCondition(node, left) && e.evaluateSimpleCondition(node, right)
}

// evaluateOrExpression evaluates expressions with 'or' operator
func (e *Evaluator) evaluateOrExpression(expr string, node *types.Node) bool {
	parts := strings.Split(expr, " or ")
	if len(parts) != 2 {
		return false
	}

	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	return e.evaluateSimpleCondition(node, left) || e.evaluateSimpleCondition(node, right)
}

// evaluatePositionExpression evaluates position-based expressions
func (e *Evaluator) evaluatePositionExpression(expr string, position int) bool {
	// Handle position() = n, position() != n, etc.
	if strings.Contains(expr, "position()") {
		if strings.Contains(expr, " = ") {
			parts := strings.Split(expr, " = ")
			if len(parts) == 2 {
				if targetPos, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					return position == targetPos
				}
			}
		} else if strings.Contains(expr, "=") {
			parts := strings.Split(expr, "=")
			if len(parts) == 2 {
				if targetPos, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					return position == targetPos
				}
			}
		}
		// Add more position operators as needed
	}

	// Handle numeric position directly
	if pos, err := strconv.Atoi(strings.TrimSpace(expr)); err == nil {
		return position == pos
	}

	return false
}

// evaluatePositionModExpression evaluates position modulo expressions
func (e *Evaluator) evaluatePositionModExpression(expr string, position int) bool {
	// Handle position() mod n = 0
	if strings.Contains(expr, "mod") && strings.Contains(expr, "position()") {
		// Parse "position() mod N = X"
		parts := strings.Split(expr, "=")
		if len(parts) != 2 {
			return false
		}

		leftSide := strings.TrimSpace(parts[0])
		rightSide := strings.TrimSpace(parts[1])

		// Extract mod operands from "position() mod N"
		modParts := strings.Split(leftSide, "mod")
		if len(modParts) != 2 {
			return false
		}

		divisor, err1 := strconv.Atoi(strings.TrimSpace(modParts[1]))
		remainder, err2 := strconv.Atoi(rightSide)

		if err1 != nil || err2 != nil {
			return false
		}

		return position%divisor == remainder
	}

	return false
}

// evaluatePositionComparison evaluates position comparison expressions
func (e *Evaluator) evaluatePositionComparison(expr string, position int) bool {
	// Handle position() > n, position() < n, etc.
	if strings.Contains(expr, "position()") {
		if strings.Contains(expr, " > ") {
			parts := strings.Split(expr, " > ")
			if len(parts) == 2 {
				if targetPos, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					return position > targetPos
				}
			}
		} else if strings.Contains(expr, ">") {
			parts := strings.Split(expr, ">")
			if len(parts) == 2 {
				if targetPos, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					return position > targetPos
				}
			}
		}
		if strings.Contains(expr, " < ") {
			parts := strings.Split(expr, " < ")
			if len(parts) == 2 {
				if targetPos, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					return position < targetPos
				}
			}
		} else if strings.Contains(expr, "<") {
			parts := strings.Split(expr, "<")
			if len(parts) == 2 {
				if targetPos, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					return position < targetPos
				}
			}
		}
		// Add more comparison operators as needed
	}

	return false
}
