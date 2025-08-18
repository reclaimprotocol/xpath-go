package evaluator

import (
	"strconv"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// predicates.go - XPath predicate application functions
// Functions that filter node sets based on predicates

// applyUnifiedPredicate handles all predicate expressions using the unified parser
func (e *Evaluator) applyUnifiedPredicate(nodes []*types.Node, expr string) []*types.Node {
	Trace("applyUnifiedPredicate: expr='%s', input nodes=%d", expr, len(nodes))

	// Handle concat() function expressions FIRST
	if strings.Contains(expr, "concat(") {
		var filtered []*types.Node
		for _, node := range nodes {
			if e.evaluateAtomicCondition(node, expr) {
				filtered = append(filtered, node)
			}
		}
		return filtered
	}

	// Handle simple positional predicates first (performance optimization)
	if pos, err := strconv.Atoi(strings.TrimSpace(expr)); err == nil {
		if pos > 0 && pos <= len(nodes) {
			return []*types.Node{nodes[pos-1]}
		}
		return []*types.Node{}
	}

	// Handle last() function
	if strings.TrimSpace(expr) == "last()" {
		if len(nodes) > 0 {
			return []*types.Node{nodes[len(nodes)-1]}
		}
		return []*types.Node{}
	}

	// Handle position() expressions (need context awareness)
	if strings.Contains(expr, "position()") {
		return e.applyPositionContextPredicate(nodes, expr)
	}

	// Handle modulo expressions (arithmetic with position context)
	if strings.Contains(expr, " mod ") {
		return e.applyArithmeticContextPredicate(nodes, expr)
	}

	// Handle axis expressions like ancestor::div[10] (check this BEFORE nested elements)
	// But only for simple axis expressions, not complex ones with functions or operators
	if e.isSimpleAxisExpression(expr) {
		return e.applyAxisExpressionPredicate(nodes, expr)
	}

	// Handle nested element predicates like span[text()='Second']
	// But exclude child path expressions and boolean expressions which should be handled by the general evaluator
	if e.isNestedElementPredicate(expr) && !e.isChildPathExpression(expr) &&
		!strings.Contains(expr, " and ") && !strings.Contains(expr, " or ") {
		return e.applyNestedElementPredicate(nodes, expr)
	}

	// Handle node() function
	if strings.Contains(expr, "node()") {
		return e.applyNodeFunctionPredicate(nodes, expr)
	}

	// Handle multiple attribute predicates (chained)
	if e.isMultipleAttributePredicate(expr) {
		return e.applyChainedAttributePredicate(nodes, expr)
	}

	// Handle true() function (should return all nodes)
	if strings.TrimSpace(expr) == "true()" {
		return nodes
	}

	// Handle false() function (should return no nodes)
	if strings.TrimSpace(expr) == "false()" {
		return []*types.Node{}
	}

	// For all other expressions, use the general condition evaluator
	var filtered []*types.Node
	for _, node := range nodes {
		if e.evaluateSimpleCondition(node, expr) {
			filtered = append(filtered, node)
		}
	}

	Trace("applyUnifiedPredicate: expr='%s', filtered nodes=%d", expr, len(filtered))
	return filtered
}

// applyPositionContextPredicate handles position-aware predicates with proper context
func (e *Evaluator) applyPositionContextPredicate(nodes []*types.Node, expr string) []*types.Node {
	// For position() expressions, we need the full context including total count
	var filtered []*types.Node

	for position, node := range nodes {
		// Position is 1-indexed in XPath
		if e.evaluateArithmeticExpressionWithContext(expr, position+1, len(nodes)) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyArithmeticContextPredicate handles arithmetic expressions with position context
func (e *Evaluator) applyArithmeticContextPredicate(nodes []*types.Node, expr string) []*types.Node {
	// For mod operations and other arithmetic, we need position context
	var filtered []*types.Node

	for position, node := range nodes {
		// Position is 1-indexed in XPath
		if e.evaluateArithmeticExpressionWithContext(expr, position+1, len(nodes)) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyNestedElementPredicate handles nested element predicates like li[span[text()='Second']]
func (e *Evaluator) applyNestedElementPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateNestedElementCondition(node, expr) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyNodeFunctionPredicate handles node() function predicates
func (e *Evaluator) applyNodeFunctionPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateNodeFunctionCondition(node, expr) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyAxisExpressionPredicate handles axis expressions like ancestor::div[10]
func (e *Evaluator) applyAxisExpressionPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateAxisExpressionCondition(node, expr) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyChainedAttributePredicate handles multiple attribute predicates
func (e *Evaluator) applyChainedAttributePredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateChainedAttributeCondition(node, expr) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// isNestedElementPredicate checks if the expression is a nested element predicate
func (e *Evaluator) isNestedElementPredicate(expr string) bool {
	// Look for patterns like element[condition]
	// Must contain brackets and element name at start
	if !strings.Contains(expr, "[") || !strings.Contains(expr, "]") {
		return false
	}

	// Should not be position-based or function-based
	return !strings.Contains(expr, "position()") &&
		!strings.Contains(expr, "ancestor::") &&
		!strings.Contains(expr, "following-sibling::") &&
		!strings.Contains(expr, "preceding-sibling::") &&
		!strings.Contains(expr, "descendant::")
}

// isSimpleAxisExpression checks if the expression is a simple axis expression
func (e *Evaluator) isSimpleAxisExpression(expr string) bool {
	// Check for axis operators
	hasAxis := strings.Contains(expr, "ancestor::") ||
		strings.Contains(expr, "following-sibling::") ||
		strings.Contains(expr, "preceding-sibling::") ||
		strings.Contains(expr, "descendant::") ||
		strings.Contains(expr, "parent::") ||
		strings.Contains(expr, "child::")

	if !hasAxis {
		return false
	}

	// Ensure it's simple (no complex functions or operators inside)
	// Allow position brackets like [10] but not complex expressions
	if strings.Contains(expr, "position()") ||
		strings.Contains(expr, "text()") ||
		strings.Contains(expr, "contains(") ||
		strings.Contains(expr, "starts-with(") ||
		strings.Contains(expr, " and ") ||
		strings.Contains(expr, " or ") {
		return false
	}

	return true
}

// isMultipleAttributePredicate checks if the expression has multiple chained attribute predicates
func (e *Evaluator) isMultipleAttributePredicate(expr string) bool {
	// Don't treat concat expressions as multiple attribute predicates
	if strings.Contains(expr, "concat(") {
		return false
	}

	// Count [@...] patterns
	attributeCount := 0
	for i := 0; i < len(expr)-1; i++ {
		if expr[i] == '[' && i+1 < len(expr) && expr[i+1] == '@' {
			attributeCount++
		}
	}

	return attributeCount >= 2
}

// Helper functions for predicate evaluation
