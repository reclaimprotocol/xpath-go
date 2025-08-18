package evaluator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// ExpressionEvaluator handles complex XPath expressions with function chaining
type ExpressionEvaluator struct {
	evaluator *Evaluator
}

// NewExpressionEvaluator creates a new expression evaluator
func NewExpressionEvaluator(evaluator *Evaluator) *ExpressionEvaluator {
	return &ExpressionEvaluator{
		evaluator: evaluator,
	}
}

// EvaluateExpression evaluates a complex XPath expression
func (ee *ExpressionEvaluator) EvaluateExpression(expr string, node *types.Node) (string, error) {
	Trace("ExpressionEvaluator.EvaluateExpression: '%s'", expr)

	parser := NewFunctionParser(expr)
	parsedExpr, err := parser.ParseExpression()
	if err != nil {
		Trace("Expression parsing failed: %v", err)
		return "", err
	}

	result := parsedExpr.Evaluate(node, ee.evaluator)
	Trace("Expression result: '%s'", result)
	return result, nil
}

// EvaluateComparison evaluates a comparison expression like "expr > 5"
func (ee *ExpressionEvaluator) EvaluateComparison(expr string, node *types.Node) (bool, error) {
	Trace("ExpressionEvaluator.EvaluateComparison: '%s'", expr)

	// Handle boolean operators first
	if strings.Contains(expr, " and ") {
		return ee.evaluateAndExpression(expr, node)
	}
	if strings.Contains(expr, " or ") {
		return ee.evaluateOrExpression(expr, node)
	}

	// Find comparison operator
	operators := []string{">=", "<=", "!=", "=", ">", "<"}
	var operator string
	var leftExpr, rightExpr string

	for _, op := range operators {
		if idx := strings.Index(expr, op); idx != -1 {
			operator = op
			leftExpr = strings.TrimSpace(expr[:idx])
			rightExpr = strings.TrimSpace(expr[idx+len(op):])
			break
		}
	}

	if operator == "" {
		// No comparison operator, treat as boolean expression
		result, err := ee.EvaluateExpression(expr, node)
		if err != nil {
			return false, err
		}
		// Non-empty string or non-zero number is true
		if result == "" || result == "0" || result == "false" {
			return false, nil
		}
		return true, nil
	}

	// Evaluate left and right sides
	leftResult, err := ee.EvaluateExpression(leftExpr, node)
	if err != nil {
		Trace("Left expression evaluation failed: %v", err)
		return false, err
	}

	rightResult, err := ee.EvaluateExpression(rightExpr, node)
	if err != nil {
		Trace("Right expression evaluation failed: %v", err)
		return false, err
	}

	Trace("Comparison: '%s' %s '%s'", leftResult, operator, rightResult)

	// Perform comparison
	result := ee.performComparison(leftResult, operator, rightResult)
	Trace("Comparison result: %v", result)
	return result, nil
}

// performComparison performs the actual comparison
func (ee *ExpressionEvaluator) performComparison(left, operator, right string) bool {
	// Try numeric comparison first
	leftNum, leftErr := strconv.ParseFloat(left, 64)
	rightNum, rightErr := strconv.ParseFloat(right, 64)

	if leftErr == nil && rightErr == nil {
		// Numeric comparison
		switch operator {
		case ">":
			return leftNum > rightNum
		case ">=":
			return leftNum >= rightNum
		case "<":
			return leftNum < rightNum
		case "<=":
			return leftNum <= rightNum
		case "=":
			return leftNum == rightNum
		case "!=":
			return leftNum != rightNum
		}
	}

	// String comparison
	switch operator {
	case "=":
		return left == right
	case "!=":
		return left != right
	case ">":
		return left > right
	case ">=":
		return left >= right
	case "<":
		return left < right
	case "<=":
		return left <= right
	}

	return false
}

// IsComplexFunctionExpression checks if an expression contains function calls
func IsComplexFunctionExpression(expr string) bool {
	// Look for function patterns
	functions := []string{
		"string-length(", "normalize-space(", "substring(", "contains(",
		"starts-with(", "count(", "position(", "last(", "text(",
	}

	for _, fn := range functions {
		if strings.Contains(expr, fn) {
			return true
		}
	}

	return false
}

// evaluateAndExpression handles "and" boolean expressions
func (ee *ExpressionEvaluator) evaluateAndExpression(expr string, node *types.Node) (bool, error) {
	parts := strings.Split(expr, " and ")
	if len(parts) < 2 {
		return false, fmt.Errorf("invalid and expression")
	}

	for _, part := range parts {
		result, err := ee.EvaluateComparison(strings.TrimSpace(part), node)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil // Short-circuit
		}
	}
	return true, nil
}

// evaluateOrExpression handles "or" boolean expressions
func (ee *ExpressionEvaluator) evaluateOrExpression(expr string, node *types.Node) (bool, error) {
	parts := strings.Split(expr, " or ")
	if len(parts) < 2 {
		return false, fmt.Errorf("invalid or expression")
	}

	for _, part := range parts {
		result, err := ee.EvaluateComparison(strings.TrimSpace(part), node)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil // Short-circuit
		}
	}
	return false, nil
}

// HasArithmeticOperations checks if expression has arithmetic
func HasArithmeticOperations(expr string) bool {
	operators := []string{" + ", " - ", " * ", " / ", " div ", " mod "}
	for _, op := range operators {
		if strings.Contains(expr, op) {
			return true
		}
	}
	return false
}
