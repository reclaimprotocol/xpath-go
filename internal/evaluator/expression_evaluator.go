package evaluator

import (
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
	parsedExpr, err := parser.Parse()
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

	result, err := ee.EvaluateExpression(expr, node)
	if err != nil {
		return false, err
	}

	// In XPath, non-empty strings and non-zero numbers are true
	// Our EvaluateExpression returns "true", "false", or a value
	if result == "true" {
		return true, nil
	}
	if result == "false" || result == "" || result == "0" {
		return false, nil
	}

	// If it's a number != 0, it's true
	if num, err := strconv.ParseFloat(result, 64); err == nil {
		return num != 0, nil
	}

	// Non-empty string is true
	return len(result) > 0, nil
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
