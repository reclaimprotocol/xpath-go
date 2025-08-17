package evaluator

import (
	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// predicates.go - XPath predicate application functions
// Functions that filter node sets based on predicates

// applyAttributePredicate filters nodes based on attribute predicates
func (e *Evaluator) applyAttributePredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateSimpleCondition(node, expr) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyTextPredicate filters nodes based on text content predicates
func (e *Evaluator) applyTextPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateSimpleCondition(node, expr) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyPositionPredicate filters nodes based on position predicates
func (e *Evaluator) applyPositionPredicate(nodes []*types.Node, expr string) []*types.Node {
	// Handle position() = n, position() > n, etc.
	var filtered []*types.Node

	for i, node := range nodes {
		position := i + 1 // XPath positions are 1-based

		if e.evaluatePositionExpressionWithContext(expr, position, len(nodes)) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyPositionModPredicate filters nodes based on modulo position predicates
func (e *Evaluator) applyPositionModPredicate(nodes []*types.Node, expr string) []*types.Node {
	// Handle position() mod n = 0, etc.
	var filtered []*types.Node

	for i, node := range nodes {
		position := i + 1 // XPath positions are 1-based

		if e.evaluatePositionModExpression(expr, position) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyContainsPredicate filters nodes based on contains() function
func (e *Evaluator) applyContainsPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateContainsExpression(expr, node) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyStartsWithPredicate filters nodes based on starts-with() function
func (e *Evaluator) applyStartsWithPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateStartsWithExpression(expr, node) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyStringLengthPredicate filters nodes based on string-length() function
func (e *Evaluator) applyStringLengthPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateStringLengthExpression(expr, node) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyNormalizeSpacePredicate filters nodes based on normalize-space() function
func (e *Evaluator) applyNormalizeSpacePredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateNormalizeSpaceExpression(expr, node) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyCountPredicate filters nodes based on count() function
func (e *Evaluator) applyCountPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateCountExpression(expr, node) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applySubstringPredicate handles substring() function predicates
func (e *Evaluator) applySubstringPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateSubstringExpression(expr, node) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyNotPredicate handles not() function predicates
func (e *Evaluator) applyNotPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateNotExpression(expr, node) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyAndPredicate handles expressions with 'and' operator
func (e *Evaluator) applyAndPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateAndExpression(expr, node) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyOrPredicate handles expressions with 'or' operator
func (e *Evaluator) applyOrPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateOrExpression(expr, node) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyNestedPredicate handles nested element predicates like div[span[@class='test']]
func (e *Evaluator) applyNestedPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	for _, node := range nodes {
		if e.evaluateNestedElementCondition(node, expr) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// Helper functions for predicate evaluation
