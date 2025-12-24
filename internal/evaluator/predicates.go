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

	// Handle expressions that need context (position, last, arithmetic)
	if strings.Contains(expr, "position()") || strings.Contains(expr, "last()") ||
		strings.Contains(expr, " mod ") || strings.Contains(expr, " div ") ||
		strings.Contains(expr, "+") || strings.Contains(expr, "-") ||
		strings.Contains(expr, "*") || strings.Contains(expr, "/") {
		return e.applyPositionContextPredicate(nodes, expr)
	}

	// For all other expressions, use the general ExpressionEvaluator
	var filtered []*types.Node
	ee := NewExpressionEvaluator(e)

	for _, node := range nodes {
		// General evaluation
		result, err := ee.EvaluateComparison(expr, node)
		if err == nil && result {
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
	ee := NewExpressionEvaluator(e)

	// Save old context
	oldPos := e.contextPosition
	oldSize := e.contextSize
	e.contextSize = len(nodes)

	for position, node := range nodes {
		// Position is 1-indexed in XPath
		e.contextPosition = position + 1
		result, err := ee.EvaluateComparison(expr, node)
		if err == nil && result {
			filtered = append(filtered, node)
		}
	}

	// Restore context
	e.contextPosition = oldPos
	e.contextSize = oldSize

	return filtered
}

// Helper functions for predicate evaluation
