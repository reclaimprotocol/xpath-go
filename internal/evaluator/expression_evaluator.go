package evaluator

import (
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

	result := evaluateXPathValue(parsedExpr, node, ee.evaluator).legacyString()
	Trace("Expression result: '%s'", result)
	return result, nil
}

// EvaluateValue evaluates an expression without erasing its XPath type. This
// is used by predicates, where a numeric result means "the context position"
// rather than merely a truthy non-zero value.
func (ee *ExpressionEvaluator) EvaluateValue(expr string, node *types.Node) (xpathValue, error) {
	parser := NewFunctionParser(expr)
	parsedExpr, err := parser.Parse()
	if err != nil {
		return xpathValue{kind: invalidXPathValue}, err
	}
	return evaluateXPathValue(parsedExpr, node, ee.evaluator), nil
}

// EvaluateComparison evaluates a comparison expression like "expr > 5"
func (ee *ExpressionEvaluator) EvaluateComparison(expr string, node *types.Node) (bool, error) {
	Trace("ExpressionEvaluator.EvaluateComparison: '%s'", expr)

	result, err := ee.EvaluateValue(expr, node)
	if err != nil {
		return false, err
	}

	if result.kind == numberXPathValue {
		return result.number == float64(ee.evaluator.contextPosition), nil
	}
	return result.toBoolean(), nil
}

// IsComplexFunctionExpression checks if an expression contains function calls
func IsComplexFunctionExpression(expr string) bool {
	// Look for function patterns
	functions := []string{
		"string-length(", "normalize-space(", "substring(", "contains(",
		"starts-with(", "count(", "position(", "last(", "text(",
		"string(", "number(", "boolean(",
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
