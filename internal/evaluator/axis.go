package evaluator

import "github.com/reclaimprotocol/xpath-go/pkg/types"

// axis.go - XPath axis navigation functions
// Functions for traversing the document tree along different axes

// getChildNodes returns direct child nodes of the given node
func (e *Evaluator) getChildNodes(node *types.Node) []*types.Node {
	return node.Children
}

// getDescendantNodes returns all descendant nodes, optionally including self
func (e *Evaluator) getDescendantNodes(node *types.Node, includeSelf bool) []*types.Node {
	var nodes []*types.Node

	if includeSelf {
		nodes = append(nodes, node)
	}

	for _, child := range node.Children {
		nodes = append(nodes, child)
		nodes = append(nodes, e.getDescendantNodes(child, false)...)
	}

	return nodes
}

// getAncestorNodes returns all ancestor nodes, optionally including self
func (e *Evaluator) getAncestorNodes(node *types.Node, includeSelf bool) []*types.Node {
	var nodes []*types.Node

	// First collect ancestors (excluding self)
	current := node.Parent
	for current != nil {
		nodes = append(nodes, current)
		current = current.Parent
	}

	// Reverse to get document order (top-down)
	for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}

	// Add self at the end if needed (document order)
	if includeSelf {
		nodes = append(nodes, node)
	}

	return nodes
}

// getFollowingSiblings returns all following sibling nodes
func (e *Evaluator) getFollowingSiblings(node *types.Node) []*types.Node {
	var siblings []*types.Node

	if node.Parent == nil {
		return siblings
	}

	found := false
	for _, sibling := range node.Parent.Children {
		if found {
			siblings = append(siblings, sibling)
		} else if sibling == node {
			found = true
		}
	}

	return siblings
}

// getPrecedingSiblings returns all preceding sibling nodes
func (e *Evaluator) getPrecedingSiblings(node *types.Node) []*types.Node {
	var siblings []*types.Node

	if node.Parent == nil {
		return siblings
	}

	for _, sibling := range node.Parent.Children {
		if sibling == node {
			break
		}
		siblings = append(siblings, sibling)
	}

	return siblings
}

// getAttributeNodes returns all attribute nodes for the given element
func (e *Evaluator) getAttributeNodes(node *types.Node) []*types.Node {
	var attrNodes []*types.Node

	for attrName, attrValue := range node.Attributes {
		attrNode := &types.Node{
			Name:        attrName,
			Type:        types.AttributeNode,
			TextContent: attrValue,
			Parent:      node,
		}
		attrNodes = append(attrNodes, attrNode)
	}

	return attrNodes
}

// getAllNodes returns all nodes in the document tree
func (e *Evaluator) getAllNodes(root *types.Node) []*types.Node {
	var nodes []*types.Node

	nodes = append(nodes, root)
	for _, child := range root.Children {
		nodes = append(nodes, e.getAllNodes(child)...)
	}

	return nodes
}

// hasAncestor checks if a node has an ancestor of the specified type
func (e *Evaluator) hasAncestor(node *types.Node, ancestorType string) bool {
	current := node.Parent
	for current != nil {
		if current.Name == ancestorType {
			return true
		}
		current = current.Parent
	}
	return false
}
