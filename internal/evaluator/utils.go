package evaluator

import (
	"sort"

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

// sortNodePointersByTreeOrder exposes XPath node sets in DOM document order.
// Source offsets cannot be used here because foster parenting and adoption
// agency recovery can deliberately place nodes before nodes that appeared
// earlier in the input.
func (e *Evaluator) sortNodePointersByTreeOrder(nodes []*types.Node, document *types.Node) {
	order, attributeBase, attributeRanks := e.treeOrder, e.attributeBase, e.attributeRanks
	if order == nil {
		order, attributeBase, attributeRanks = treeOrderMaps(document)
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return treeOrderForNode(nodes[i], order, attributeBase, attributeRanks) < treeOrderForNode(nodes[j], order, attributeBase, attributeRanks)
	})
}

func treeOrderMaps(document *types.Node) (map[*types.Node]int, map[*types.Node]int, map[*types.Node]map[string]int) {
	order := make(map[*types.Node]int)
	attributeBase := make(map[*types.Node]int)
	attributeRanks := make(map[*types.Node]map[string]int)
	next := 0
	var visit func(*types.Node)
	visit = func(node *types.Node) {
		if node == nil {
			return
		}
		order[node] = next
		next++
		attributeBase[node] = next
		if len(node.AttributeOrder) > 0 {
			ranks := make(map[string]int, len(node.AttributeOrder))
			for index, name := range node.AttributeOrder {
				ranks[name] = index
			}
			attributeRanks[node] = ranks
		}
		next += len(node.Attributes)
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(document)
	return order, attributeBase, attributeRanks
}

func treeOrderForNode(node *types.Node, order, attributeBase map[*types.Node]int, attributeRanks map[*types.Node]map[string]int) int {
	if node == nil {
		return int(^uint(0) >> 1)
	}
	identity := node
	if node.Origin != nil {
		identity = node.Origin
	}
	if node.Type == types.AttributeNode {
		rank, ok := attributeRanks[node.Parent][node.Name]
		if !ok {
			rank = int(^uint(0) >> 1)
		}
		return attributeBase[node.Parent] + rank
	}
	if value, ok := order[identity]; ok {
		return value
	}
	return int(^uint(0) >> 1)
}
