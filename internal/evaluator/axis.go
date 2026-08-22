package evaluator

import (
	"sort"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// prepareAxisOrder builds the XPath document order once per evaluation. HTML
// recovery can move nodes away from their source offsets, so following and
// preceding axes must use the recovered DOM tree rather than byte positions.
func (e *Evaluator) prepareAxisOrder(document *types.Node) {
	e.axisOrder = e.axisOrder[:0]
	e.axisIndex = make(map[*types.Node]int)
	e.axisSubtreeEnd = make(map[*types.Node]int)
	e.axisMatchIndexes = make(map[string][]int)
	var visit func(*types.Node)
	visit = func(node *types.Node) {
		if node == nil {
			return
		}
		e.axisIndex[node] = len(e.axisOrder)
		e.axisOrder = append(e.axisOrder, node)
		for _, child := range node.Children {
			visit(child)
		}
		e.axisSubtreeEnd[node] = len(e.axisOrder)
	}
	visit(document)
	// DOM ordering also defines output order for all regular node sets. Cache
	// the metadata alongside the axis preorder so repeated sort operations do
	// not traverse the whole recovered tree again.
	e.treeOrder, e.attributeBase, e.attributeRanks = treeOrderMaps(document)
}

func (e *Evaluator) axisIndexesMatching(nodeTest string) []int {
	if indexes, ok := e.axisMatchIndexes[nodeTest]; ok {
		return indexes
	}
	indexes := make([]int, 0)
	for index, node := range e.axisOrder {
		if axisNodeMatches(node, nodeTest) {
			indexes = append(indexes, index)
		}
	}
	e.axisMatchIndexes[nodeTest] = indexes
	return indexes
}

func axisNodeMatches(node *types.Node, nodeTest string) bool {
	switch {
	case nodeTest == "*":
		return node.Type == types.ElementNode
	case nodeTest == "node()":
		return true
	case nodeTest == "text()":
		return node.Type == types.TextNode
	case nodeTest == "comment()":
		return node.Type == types.CommentNode
	case nodeTest == "processing-instruction()" || strings.HasPrefix(nodeTest, "processing-instruction("):
		target := ""
		if nodeTest != "processing-instruction()" {
			target = strings.TrimSuffix(strings.TrimPrefix(nodeTest, "processing-instruction("), ")")
			target = strings.Trim(target, `"'`)
		}
		return node.Type == types.ProcessingInstructionNode && (target == "" || node.Name == target)
	default:
		return node.Type == types.ElementNode && strings.EqualFold(node.Name, nodeTest)
	}
}

func (e *Evaluator) positionalAxisNode(node *types.Node, axis types.XPathAxis, nodeTest string, position int, last bool) *types.Node {
	if node == nil {
		return nil
	}
	identity := axisTreeIdentity(node)
	indexes := e.axisIndexesMatching(nodeTest)
	if len(indexes) == 0 {
		return nil
	}

	if axis == types.AxisFollowing {
		start := 0
		if node.Type == types.AttributeNode {
			identity = axisTreeIdentity(node.Parent)
			ownerIndex, exists := e.axisIndex[identity]
			if !exists {
				return nil
			}
			start = ownerIndex + 1
		} else {
			var exists bool
			start, exists = e.axisSubtreeEnd[identity]
			if !exists {
				return nil
			}
		}
		first := sort.SearchInts(indexes, start)
		if first == len(indexes) {
			return nil
		}
		selected := first + position - 1
		if last {
			selected = len(indexes) - 1
		}
		if selected < first || selected >= len(indexes) {
			return nil
		}
		return e.axisOrder[indexes[selected]]
	}

	if node.Type == types.AttributeNode {
		identity = axisTreeIdentity(node.Parent)
	}
	contextIndex, exists := e.axisIndex[identity]
	if !exists {
		return nil
	}
	ancestors := make(map[*types.Node]struct{})
	ancestor := identity.Parent
	if node.Type == types.AttributeNode {
		ancestor = identity
	}
	for current := ancestor; current != nil; current = current.Parent {
		ancestors[current] = struct{}{}
	}
	limit := sort.SearchInts(indexes, contextIndex)
	if last {
		for i := 0; i < limit; i++ {
			candidate := e.axisOrder[indexes[i]]
			if _, excluded := ancestors[candidate]; !excluded {
				return candidate
			}
		}
		return nil
	}
	seen := 0
	for i := limit - 1; i >= 0; i-- {
		candidate := e.axisOrder[indexes[i]]
		if _, excluded := ancestors[candidate]; excluded {
			continue
		}
		seen++
		if seen == position {
			return candidate
		}
	}
	return nil
}

func axisTreeIdentity(node *types.Node) *types.Node {
	if node != nil && node.Origin != nil {
		return node.Origin
	}
	return node
}

// getFollowingNodes returns nodes after the context subtree in document order.
// Attribute nodes are ordered after their owner and before its children, so an
// attribute's following axis begins with the owner's first child.
func (e *Evaluator) getFollowingNodes(node *types.Node) []*types.Node {
	if node == nil {
		return nil
	}
	identity := axisTreeIdentity(node)
	start := 0
	if node.Type == types.AttributeNode {
		identity = axisTreeIdentity(node.Parent)
		index, ok := e.axisIndex[identity]
		if !ok {
			return nil
		}
		start = index + 1
	} else {
		var ok bool
		start, ok = e.axisSubtreeEnd[identity]
		if !ok {
			return nil
		}
	}
	if start >= len(e.axisOrder) {
		return nil
	}
	return e.axisOrder[start:]
}

// getPrecedingNodes returns the reverse axis order (nearest first), excluding
// ancestors and all generated attribute/namespace nodes.
func (e *Evaluator) getPrecedingNodes(node *types.Node) []*types.Node {
	if node == nil {
		return nil
	}
	identity := axisTreeIdentity(node)
	ancestor := identity.Parent
	if node.Type == types.AttributeNode {
		identity = axisTreeIdentity(node.Parent)
		ancestor = identity
	}
	index, ok := e.axisIndex[identity]
	if !ok || index == 0 {
		return nil
	}
	ancestors := make(map[*types.Node]struct{})
	for current := ancestor; current != nil; current = current.Parent {
		ancestors[current] = struct{}{}
	}
	result := make([]*types.Node, 0, index)
	for i := index - 1; i >= 0; i-- {
		candidate := e.axisOrder[i]
		if _, excluded := ancestors[candidate]; !excluded {
			result = append(result, candidate)
		}
	}
	return result
}

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

	// Collect ancestors in closest-to-farthest order first
	current := node.Parent
	for current != nil {
		nodes = append(nodes, current)
		current = current.Parent
	}

	if includeSelf {
		// For ancestor-or-self, order should be document order (root to self)
		// Reverse the ancestor list and add self at the end
		for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
			nodes[i], nodes[j] = nodes[j], nodes[i]
		}
		nodes = append(nodes, node)
	}
	// For ancestor axis (without self), keep closest-to-farthest order for position() predicates

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

	index := -1
	for i, sibling := range node.Parent.Children {
		if sibling == node {
			index = i
			break
		}
	}

	// preceding-sibling is a reverse axis. Predicates therefore see the
	// nearest sibling as position 1, even though final node-set results are
	// returned in document order.
	for i := index - 1; i >= 0; i-- {
		siblings = append(siblings, node.Parent.Children[i])
	}

	return siblings
}

// getAttributeNodes returns all attribute nodes for the given element
func (e *Evaluator) getAttributeNodes(node *types.Node) []*types.Node {
	if node == nil {
		return nil
	}
	if e.attributeNodes == nil {
		e.attributeNodes = make(map[*types.Node][]*types.Node)
	}
	if cached, ok := e.attributeNodes[node]; ok {
		return cached
	}

	var attrNodes []*types.Node

	// Use AttributeOrder if available (preserves document order)
	if len(node.AttributeOrder) > 0 {
		// Create attribute nodes in document order
		for _, attrName := range node.AttributeOrder {
			if attrValue, exists := node.Attributes[attrName]; exists {
				attrNode := &types.Node{
					Name:         attrName,
					Type:         types.AttributeNode,
					NamespaceURI: node.AttributeNamespaces[attrName],
					LocalName:    attributeLocalName(node, attrName),
					Prefix:       node.AttributePrefixes[attrName],
					TextContent:  attrValue,
					Value:        attrValue,
					Parent:       node,
				}
				attrNodes = append(attrNodes, attrNode)
			}
		}
	} else {
		// Fallback to sorted order for consistency when AttributeOrder is not available
		var attrNames []string
		for attrName := range node.Attributes {
			attrNames = append(attrNames, attrName)
		}
		sort.Strings(attrNames)

		// Create attribute nodes in sorted order
		for _, attrName := range attrNames {
			attrValue := node.Attributes[attrName]
			attrNode := &types.Node{
				Name:         attrName,
				Type:         types.AttributeNode,
				NamespaceURI: node.AttributeNamespaces[attrName],
				LocalName:    attributeLocalName(node, attrName),
				Prefix:       node.AttributePrefixes[attrName],
				TextContent:  attrValue,
				Value:        attrValue,
				Parent:       node,
			}
			attrNodes = append(attrNodes, attrNode)
		}
	}

	e.attributeNodes[node] = attrNodes
	return attrNodes
}

func attributeLocalName(owner *types.Node, qualified string) string {
	if local := owner.AttributeLocalNames[qualified]; local != "" {
		return local
	}
	return qualified
}
