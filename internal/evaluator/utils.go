package evaluator

import (
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
