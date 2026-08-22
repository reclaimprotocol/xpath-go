package evaluator

import "github.com/reclaimprotocol/xpath-go/pkg/types"

// predicates.go - XPath predicate application functions
// Functions that filter node sets based on predicates

// applyPositionContextPredicateExpression evaluates an already parsed
// predicate with the correct XPath position/size context.
func (e *Evaluator) applyPositionContextPredicateExpression(nodes []*types.Node, expression Expression) []*types.Node {
	// For position() expressions, we need the full context including total count
	var filtered []*types.Node
	// Save old context
	oldPos := e.contextPosition
	oldSize := e.contextSize
	e.contextSize = len(nodes)

	for position, node := range nodes {
		// Position is 1-indexed in XPath
		e.contextPosition = position + 1
		result := evaluateXPathValue(expression, node, e)
		if (result.kind == numberXPathValue && result.number == float64(e.contextPosition)) ||
			(result.kind != numberXPathValue && result.toBoolean()) {
			filtered = append(filtered, node)
		}
	}

	// Restore context
	e.contextPosition = oldPos
	e.contextSize = oldSize

	return filtered
}

// Helper functions for predicate evaluation
