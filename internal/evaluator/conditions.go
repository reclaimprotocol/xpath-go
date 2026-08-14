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

// matchesNodeTest checks if a node matches a node test
func (e *Evaluator) matchesNodeTest(node *types.Node, nodeTest string) bool {
	if nodeTest == "*" {
		return node.Type == types.ElementNode
	}
	if nodeTest == "node()" || nodeTest == "node" {
		return true
	}
	if nodeTest == "text()" {
		return node.Type == types.TextNode
	}
	if nodeTest == "comment()" {
		return node.Type == types.CommentNode
	}
	if nodeTest == "processing-instruction()" || strings.HasPrefix(nodeTest, "processing-instruction(") {
		target := ""
		if nodeTest != "processing-instruction()" {
			target = strings.TrimSuffix(strings.TrimPrefix(nodeTest, "processing-instruction("), ")")
			target = strings.Trim(target, `"'`)
		}
		return node.Type == types.ProcessingInstructionNode && (target == "" || node.Name == target)
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
