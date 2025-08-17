package evaluator

import (
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// utils.go - Utility and helper functions
// Common utility functions used across the evaluator

// removeDuplicates removes duplicate nodes from a slice while preserving order
func (e *Evaluator) removeDuplicates(nodes []*types.Node) []*types.Node {
	seen := make(map[*types.Node]bool)
	result := make([]*types.Node, 0, len(nodes))

	for _, node := range nodes {
		if !seen[node] {
			seen[node] = true
			result = append(result, node)
		}
	}

	return result
}

// isChildPathExpression checks if a condition represents a child path expression
func (e *Evaluator) isChildPathExpression(condition string) bool {
	// Check for path-like expressions: head/title, head/meta[@charset]
	if !strings.Contains(condition, "/") {
		return false
	}

	// Split by '/' and validate each part
	parts := strings.Split(condition, "/")
	if len(parts) < 2 {
		return false
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if part looks like element name (with optional predicate)
		if strings.Contains(part, "[") {
			// Extract element name before predicate
			idx := strings.Index(part, "[")
			elementName := strings.TrimSpace(part[:idx])
			if !e.isValidElementName(elementName) {
				return false
			}
		} else {
			if !e.isValidElementName(part) {
				return false
			}
		}
	}

	return true
}

// isValidElementName checks if a string is a valid XML element name
func (e *Evaluator) isValidElementName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return false
	}

	// Special case for wildcard
	if name == "*" {
		return true
	}

	// Simple validation: letters, digits, hyphens, underscores, colons
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == ':') {
			return false
		}
	}

	return true
}

// isSimpleElementName checks if a condition is a simple element name
func (e *Evaluator) isSimpleElementName(condition string) bool {
	condition = strings.TrimSpace(condition)

	// Should not contain operators or special characters
	if strings.ContainsAny(condition, "=!<>@()[]/:") {
		return false
	}

	return e.isValidElementName(condition)
}

// evaluateChildPath evaluates child path expressions like head/title
func (e *Evaluator) evaluateChildPath(node *types.Node, path string) bool {
	parts := strings.Split(path, "/")
	currentNodes := []*types.Node{node}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var nextNodes []*types.Node

		// Check if part has predicate
		if strings.Contains(part, "[") {
			idx := strings.Index(part, "[")
			elementName := strings.TrimSpace(part[:idx])
			predicate := strings.TrimSpace(part[idx+1:])

			// Remove closing bracket
			predicate = strings.TrimSuffix(predicate, "]")

			// Find children matching element name and predicate
			for _, currentNode := range currentNodes {
				for _, child := range currentNode.Children {
					if child.Name == elementName {
						if e.evaluateSimpleCondition(child, predicate) {
							nextNodes = append(nextNodes, child)
						}
					}
				}
			}
		} else {
			// Simple element name
			for _, currentNode := range currentNodes {
				for _, child := range currentNode.Children {
					if child.Name == part {
						nextNodes = append(nextNodes, child)
					}
				}
			}
		}

		currentNodes = nextNodes
		if len(currentNodes) == 0 {
			return false
		}
	}

	return len(currentNodes) > 0
}

// hasChildElement checks if a node has a child element with the given name
func (e *Evaluator) hasChildElement(node *types.Node, elementName string) bool {
	// Handle wildcard: * matches any child element
	if elementName == "*" {
		for _, child := range node.Children {
			if child.Type == types.ElementNode {
				return true
			}
		}
		return false
	}

	// Handle specific element name
	for _, child := range node.Children {
		if child.Name == elementName {
			return true
		}
	}
	return false
}

// Helper functions for character validation
