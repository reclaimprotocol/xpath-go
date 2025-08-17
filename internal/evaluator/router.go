package evaluator

import (
	"strconv"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// PredicateType represents the type of predicate expression
type PredicateType int

const (
	// Boolean expressions (highest precedence)
	BooleanType PredicateType = iota // AND/OR expressions with any complexity

	// Single expressions
	AttributeType     // @attr or @attr='value'
	TextType          // text() or text()='value'
	AxisType          // ancestor::div, following-sibling::p
	FunctionType      // contains(), position(), count(), etc.
	PositionalType    // [1], [2], [10], numeric position predicates
	NestedElementType // span[text()='value'], div[@class]
	SimpleElementType // span, div, a (simple element existence)

	// Fallback
	UnknownType
)

// PredicateClassifier analyzes and classifies predicate expressions
type PredicateClassifier struct {
	expr string
}

// ClassifyPredicate analyzes an expression and returns its type and metadata
func ClassifyPredicate(expr string) (PredicateType, map[string]interface{}) {
	expr = strings.TrimSpace(expr)
	classifier := &PredicateClassifier{expr: expr}
	metadata := make(map[string]interface{})

	// Check for boolean expressions first (highest precedence)
	if classifier.hasBooleanOperators() {
		// All boolean expressions use the unified BooleanType
		parts := classifier.splitBooleanExpression()
		metadata["operator"] = classifier.getBooleanOperator()
		metadata["parts"] = parts
		return BooleanType, metadata
	}

	// Single expression types
	switch {
	case classifier.isAxisExpression():
		metadata["axis"], metadata["nodetest"] = classifier.parseAxisExpression()
		return AxisType, metadata

	case classifier.isAttributeExpression():
		metadata["attribute"], metadata["value"] = classifier.parseAttributeExpression()
		return AttributeType, metadata

	case classifier.isTextExpression():
		metadata["comparison"] = classifier.parseTextExpression()
		return TextType, metadata

	case classifier.isFunctionCall():
		metadata["function"], metadata["args"] = classifier.parseFunctionCall()
		return FunctionType, metadata

	case classifier.isPositionalExpression():
		metadata["position"] = classifier.parsePositionalExpression()
		return PositionalType, metadata

	case classifier.hasNestedPredicates():
		metadata["element"], metadata["predicate"] = classifier.parseNestedElement()
		return NestedElementType, metadata

	case classifier.isSimpleElement():
		metadata["element"] = expr
		return SimpleElementType, metadata

	default:
		return UnknownType, metadata
	}
}

// Router functions for the classifier
func (c *PredicateClassifier) hasBooleanOperators() bool {
	return strings.Contains(c.expr, " and ") || strings.Contains(c.expr, " or ")
}

func (c *PredicateClassifier) hasParentheses() bool {
	return strings.Contains(c.expr, "(") && strings.Contains(c.expr, ")")
}

func (c *PredicateClassifier) hasFunctionCalls() bool {
	functions := []string{
		"contains(", "starts-with(", "string-length(", "normalize-space(",
		"substring(", "not(", "not (", "text()", "position()", "last()", "count(",
	}

	for _, fn := range functions {
		if strings.Contains(c.expr, fn) {
			return true
		}
	}
	return false
}

func (c *PredicateClassifier) hasNestedPredicates() bool {
	// Look for element[predicate] pattern
	return strings.Contains(c.expr, "[") && strings.Contains(c.expr, "]") &&
		!strings.HasPrefix(c.expr, "@") && !strings.HasPrefix(c.expr, "text()")
}

func (c *PredicateClassifier) isAxisExpression() bool {
	return strings.Contains(c.expr, "::")
}

func (c *PredicateClassifier) isAttributeExpression() bool {
	return strings.HasPrefix(c.expr, "@")
}

func (c *PredicateClassifier) isTextExpression() bool {
	return strings.HasPrefix(c.expr, "text()")
}

func (c *PredicateClassifier) isFunctionCall() bool {
	// Check if expression starts with a known function or contains function patterns
	functions := []string{
		"contains(", "starts-with(", "string-length(", "normalize-space(",
		"substring(", "not(", "not (", "position()", "last()", "count(",
	}

	for _, fn := range functions {
		if strings.HasPrefix(c.expr, fn) ||
			(strings.Contains(c.expr, fn) && !c.hasNestedPredicates()) {
			return true
		}
	}
	return false
}

func (c *PredicateClassifier) isPositionalExpression() bool {
	// Check if the expression is a pure numeric value (position predicate)
	expr := strings.TrimSpace(c.expr)
	if expr == "" {
		return false
	}

	// Check if it's a positive integer
	if num, err := strconv.Atoi(expr); err == nil && num > 0 {
		return true
	}

	// Check for last() function
	if expr == "last()" {
		return true
	}

	return false
}

func (c *PredicateClassifier) isSimpleElement() bool {
	// Simple element name without attributes, functions, or operators
	if c.hasBooleanOperators() || c.hasParentheses() || c.hasFunctionCalls() ||
		c.hasNestedPredicates() || c.isAttributeExpression() || c.isTextExpression() ||
		c.isAxisExpression() || c.isPositionalExpression() {
		return false
	}

	// Check if it's a valid element name (alphanumeric + some special chars)
	expr := strings.TrimSpace(c.expr)
	if expr == "" {
		return false
	}

	// Simple heuristic: if it contains no special XPath characters, it's likely an element name
	specialChars := []string{"@", "[", "]", "(", ")", "=", "'", "\"", "::", "/"}
	for _, char := range specialChars {
		if strings.Contains(expr, char) {
			return false
		}
	}

	return true
}

// Parser functions
func (c *PredicateClassifier) splitBooleanExpression() []string {
	// Simple split - in a real implementation, this would handle nested parentheses
	if strings.Contains(c.expr, " and ") {
		return strings.Split(c.expr, " and ")
	}
	if strings.Contains(c.expr, " or ") {
		return strings.Split(c.expr, " or ")
	}
	return []string{c.expr}
}

func (c *PredicateClassifier) getBooleanOperator() string {
	if strings.Contains(c.expr, " and ") {
		return "and"
	}
	if strings.Contains(c.expr, " or ") {
		return "or"
	}
	return ""
}

func (c *PredicateClassifier) parseAxisExpression() (string, string) {
	parts := strings.Split(c.expr, "::")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", c.expr
}

func (c *PredicateClassifier) parseAttributeExpression() (string, string) {
	if strings.Contains(c.expr, "=") {
		parts := strings.SplitN(c.expr, "=", 2)
		attr := strings.TrimPrefix(strings.TrimSpace(parts[0]), "@")
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		return attr, value
	}
	return strings.TrimPrefix(c.expr, "@"), ""
}

func (c *PredicateClassifier) parseTextExpression() string {
	return c.expr // Return full expression for now
}

func (c *PredicateClassifier) parseFunctionCall() (string, []string) {
	// Extract function name and arguments - simplified
	if idx := strings.Index(c.expr, "("); idx != -1 {
		funcName := c.expr[:idx]
		return funcName, []string{} // Simplified - would parse args in real implementation
	}
	return c.expr, []string{}
}

func (c *PredicateClassifier) parsePositionalExpression() interface{} {
	expr := strings.TrimSpace(c.expr)
	
	// Check for last() function
	if expr == "last()" {
		return "last()"
	}
	
	// Parse numeric position
	if num, err := strconv.Atoi(expr); err == nil && num > 0 {
		return num
	}
	
	return expr
}

func (c *PredicateClassifier) parseNestedElement() (string, string) {
	if idx := strings.Index(c.expr, "["); idx != -1 {
		element := strings.TrimSpace(c.expr[:idx])
		predicate := strings.TrimSpace(c.expr[idx+1:])
		predicate = strings.TrimSuffix(predicate, "]")
		return element, predicate
	}
	return c.expr, ""
}

// RoutePredicateExpression routes an expression to the appropriate handler
func (e *Evaluator) RoutePredicateExpression(nodes []*types.Node, expr string) []*types.Node {
	predicateType, metadata := ClassifyPredicate(expr)

	switch predicateType {
	case BooleanType:
		operator := metadata["operator"].(string)
		if operator == "and" {
			return e.applyAndPredicate(nodes, expr)
		} else if operator == "or" {
			return e.applyOrPredicate(nodes, expr)
		}
		// Fallback if operator is neither and nor or
		return nodes

	case AttributeType:
		return e.applyAttributePredicate(nodes, expr)

	case TextType:
		return e.applyTextPredicate(nodes, expr)

	case AxisType:
		var filtered []*types.Node
		for _, node := range nodes {
			if e.evaluateSimpleCondition(node, expr) {
				filtered = append(filtered, node)
			}
		}
		return filtered

	case FunctionType:
		return e.routeFunctionCall(nodes, expr)

	case PositionalType:
		return e.applyPositionalPredicate(nodes, expr)

	case NestedElementType:
		return e.applyNestedPredicate(nodes, expr)

	case SimpleElementType:
		var filtered []*types.Node
		for _, node := range nodes {
			if e.evaluateSimpleCondition(node, expr) {
				filtered = append(filtered, node)
			}
		}
		return filtered

	default:
		// Fallback to old logic
		var filtered []*types.Node
		for _, node := range nodes {
			if e.evaluateSimpleCondition(node, expr) {
				filtered = append(filtered, node)
			}
		}
		return filtered
	}
}

// routeFunctionCall routes function calls to appropriate handlers
func (e *Evaluator) routeFunctionCall(nodes []*types.Node, expr string) []*types.Node {
	switch {
	case strings.HasPrefix(expr, "not(") || strings.HasPrefix(expr, "not ("):
		return e.applyNotPredicate(nodes, expr)
	case strings.Contains(expr, "position()"):
		// Check for modulo operations first
		if strings.Contains(expr, "mod") {
			return e.applyPositionModPredicate(nodes, expr)
		}
		return e.applyPositionPredicate(nodes, expr)
	case strings.Contains(expr, "contains("):
		return e.applyContainsPredicate(nodes, expr)
	case strings.Contains(expr, "starts-with("):
		return e.applyStartsWithPredicate(nodes, expr)
	case strings.Contains(expr, "substring("):
		return e.applySubstringPredicate(nodes, expr)
	case strings.Contains(expr, "string-length("):
		return e.applyStringLengthPredicate(nodes, expr)
	case strings.Contains(expr, "count("):
		return e.applyCountPredicate(nodes, expr)
	case strings.Contains(expr, "normalize-space("):
		return e.applyNormalizeSpacePredicate(nodes, expr)
	default:
		// Fallback
		var filtered []*types.Node
		for _, node := range nodes {
			if e.evaluateSimpleCondition(node, expr) {
				filtered = append(filtered, node)
			}
		}
		return filtered
	}
}
