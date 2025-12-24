package evaluator

import (
	"strconv"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// conditions.go - XPath condition evaluation logic
// Functions that evaluate individual conditions against nodes

// evaluateSimpleCondition evaluates a simple condition against a single node
func (e *Evaluator) evaluateSimpleCondition(node *types.Node, condition string) bool {
	Trace("evaluateSimpleCondition: '%s' on node '%s'", condition, node.Name)
	// Handle numeric positional predicate (e.g. [1])
	if pos, err := strconv.Atoi(strings.TrimSpace(condition)); err == nil {
		return e.contextPosition == pos
	}

	ee := NewExpressionEvaluator(e)
	result, err := ee.EvaluateComparison(condition, node)
	if err != nil {
		Trace("evaluateSimpleCondition error: %v", err)
		return false
	}
	return result
}

// Legacy simple condition evaluation - kept for compatibility

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
		// Check if nodeTest has a positional predicate
		if strings.Contains(nodeTest, "[") && e.hasPositionalPredicate(nodeTest) {
			return e.evaluateAncestorWithPosition(node, nodeTest)
		}

		// Standard ancestor evaluation - any matching ancestor
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
	predicate = strings.TrimSuffix(predicate, "]")

	Trace("matchesNodeTestWithPredicate: nodeTest='%s', elementName='%s', predicate='%s', node='%s'",
		nodeTest, elementName, predicate, node.Name)

	// Check element name match
	if elementName != "*" && node.Name != elementName {
		Trace("elementName mismatch: expected='%s', actual='%s'", elementName, node.Name)
		return false
	}

	// Handle positional predicates differently - they need context of sibling nodes
	if e.isPositionalPredicate(predicate) {
		Trace("detected positional predicate '%s' - this requires axis context evaluation", predicate)
		// For positional predicates in axis context, we can't evaluate them per-node
		// They need to be evaluated with the full context of matching nodes
		// Return true for element name match, position filtering happens at axis level
		return true
	}

	// Evaluate non-positional predicate
	result := e.evaluateSimpleCondition(node, predicate)
	Trace("predicate evaluation: '%s' -> %t", predicate, result)
	return result
}

// isPositionalPredicate checks if a predicate is a position-based predicate
func (e *Evaluator) isPositionalPredicate(predicate string) bool {
	predicate = strings.TrimSpace(predicate)

	// Check for numeric position
	if _, err := strconv.Atoi(predicate); err == nil {
		return true
	}

	// Check for last() function
	if predicate == "last()" {
		return true
	}

	// Check for position() function calls
	if strings.Contains(predicate, "position()") {
		return true
	}

	return false
}

// hasPositionalPredicate checks if a nodeTest contains a positional predicate
func (e *Evaluator) hasPositionalPredicate(nodeTest string) bool {
	idx := strings.Index(nodeTest, "[")
	if idx == -1 {
		return false
	}

	predicate := strings.TrimSpace(nodeTest[idx+1:])
	predicate = strings.TrimSuffix(predicate, "]")

	return e.isPositionalPredicate(predicate)
}

// evaluateAncestorWithPosition evaluates ancestor axis with positional predicates
func (e *Evaluator) evaluateAncestorWithPosition(node *types.Node, nodeTest string) bool {
	// Parse nodeTest to extract element name and predicate
	idx := strings.Index(nodeTest, "[")
	if idx == -1 {
		return false
	}

	elementName := strings.TrimSpace(nodeTest[:idx])
	predicate := strings.TrimSpace(nodeTest[idx+1:])
	predicate = strings.TrimSuffix(predicate, "]")

	Trace("evaluateAncestorWithPosition: elementName='%s', predicate='%s'", elementName, predicate)

	// Collect all ancestor nodes that match the element name
	// For ancestor axis with position predicates, order is from closest to farthest
	var matchingAncestors []*types.Node
	current := node.Parent
	for current != nil {
		if elementName == "*" || current.Name == elementName {
			matchingAncestors = append(matchingAncestors, current)
		}
		current = current.Parent
	}

	Trace("found %d matching ancestors for element '%s'", len(matchingAncestors), elementName)
	for i, ancestor := range matchingAncestors {
		Trace("  ancestor[%d]: %s", i+1, ancestor.Name)
	}

	// Apply positional predicate to the collection
	filtered := e.applyPositionalPredicate(matchingAncestors, predicate)

	Trace("after position filtering: %d nodes remain", len(filtered))

	// Return true if any nodes remain after position filtering
	return len(filtered) > 0
}
