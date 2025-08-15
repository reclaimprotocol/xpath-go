package evaluator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/reclaimprotocol/xpath-go/internal/parser"
	"github.com/reclaimprotocol/xpath-go/pkg/types"
	"github.com/reclaimprotocol/xpath-go/pkg/utils"
)

// Evaluator handles XPath expression evaluation
type Evaluator struct {
	parser     *parser.Parser
	htmlParser *utils.HTMLParser
}

// NewEvaluator creates a new XPath evaluator
func NewEvaluator() *Evaluator {
	return &Evaluator{
		parser:     parser.NewParser(),
		htmlParser: utils.NewHTMLParser(),
	}
}

// Evaluate evaluates an XPath expression against HTML/XML content
func (e *Evaluator) Evaluate(xpathExpr, content string) ([]types.Node, error) {
	// Parse XPath expression
	parsedXPath, err := e.parser.Parse(xpathExpr)
	if err != nil {
		return nil, fmt.Errorf("XPath parsing failed: %w", err)
	}

	// Parse HTML/XML content
	documentNode, err := e.htmlParser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("HTML parsing failed: %w", err)
	}

	// Evaluate XPath against document
	return e.evaluateSteps(parsedXPath, documentNode)
}

// evaluateSteps evaluates XPath steps against the document
func (e *Evaluator) evaluateSteps(xpath *types.ParsedXPath, document *types.Node) ([]types.Node, error) {
	var currentNodes []*types.Node

	if xpath.IsAbsolute {
		currentNodes = []*types.Node{document}
	} else {
		// For relative paths, start with all nodes
		currentNodes = e.getAllNodes(document)
	}

	// Apply each step
	for _, step := range xpath.Steps {
		nextNodes := []*types.Node{}
		
		for _, node := range currentNodes {
			stepResults := e.evaluateStep(step, node)
			nextNodes = append(nextNodes, stepResults...)
		}
		
		currentNodes = e.removeDuplicates(nextNodes)
	}

	// Convert to result format
	var results []types.Node
	for _, node := range currentNodes {
		if node != nil {
			results = append(results, *node)
		}
	}

	return results, nil
}

// evaluateStep evaluates a single XPath step
func (e *Evaluator) evaluateStep(step types.XPathStep, contextNode *types.Node) []*types.Node {
	var candidates []*types.Node

	// Apply axis to get candidate nodes
	switch step.Axis {
	case types.AxisChild:
		candidates = e.getChildNodes(contextNode)
	case types.AxisDescendant:
		candidates = e.getDescendantNodes(contextNode, false)
	case types.AxisDescendantOrSelf:
		candidates = e.getDescendantNodes(contextNode, true)
	case types.AxisParent:
		if contextNode.Parent != nil {
			candidates = []*types.Node{contextNode.Parent}
		}
	case types.AxisAncestor:
		candidates = e.getAncestorNodes(contextNode, false)
	case types.AxisAncestorOrSelf:
		candidates = e.getAncestorNodes(contextNode, true)
	case types.AxisFollowingSibling:
		candidates = e.getFollowingSiblings(contextNode)
	case types.AxisPrecedingSibling:
		candidates = e.getPrecedingSiblings(contextNode)
	case types.AxisAttribute:
		candidates = e.getAttributeNodes(contextNode)
	case types.AxisSelf:
		candidates = []*types.Node{contextNode}
	default:
		return []*types.Node{}
	}

	// Apply node test
	filtered := e.applyNodeTest(candidates, step.NodeTest)

	// Apply predicates
	for _, predicate := range step.Predicates {
		filtered = e.applyPredicate(filtered, predicate, contextNode)
	}

	return filtered
}

// getChildNodes returns immediate children of a node
func (e *Evaluator) getChildNodes(node *types.Node) []*types.Node {
	return node.Children
}

// getDescendantNodes returns all descendant nodes
func (e *Evaluator) getDescendantNodes(node *types.Node, includeSelf bool) []*types.Node {
	var descendants []*types.Node
	
	if includeSelf {
		descendants = append(descendants, node)
	}

	for _, child := range node.Children {
		descendants = append(descendants, child)
		descendants = append(descendants, e.getDescendantNodes(child, false)...)
	}

	return descendants
}

// getAncestorNodes returns all ancestor nodes
func (e *Evaluator) getAncestorNodes(node *types.Node, includeSelf bool) []*types.Node {
	var ancestors []*types.Node
	
	if includeSelf {
		ancestors = append(ancestors, node)
	}

	current := node.Parent
	for current != nil {
		ancestors = append(ancestors, current)
		current = current.Parent
	}

	return ancestors
}

// getFollowingSiblings returns following sibling nodes
func (e *Evaluator) getFollowingSiblings(node *types.Node) []*types.Node {
	if node.Parent == nil {
		return []*types.Node{}
	}

	var siblings []*types.Node
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

// getPrecedingSiblings returns preceding sibling nodes
func (e *Evaluator) getPrecedingSiblings(node *types.Node) []*types.Node {
	if node.Parent == nil {
		return []*types.Node{}
	}

	var siblings []*types.Node

	for _, sibling := range node.Parent.Children {
		if sibling == node {
			break
		}
		siblings = append(siblings, sibling)
	}

	return siblings
}

// getAttributeNodes returns attribute nodes (simulated as nodes)
func (e *Evaluator) getAttributeNodes(node *types.Node) []*types.Node {
	var attributes []*types.Node

	for name, value := range node.Attributes {
		attrNode := &types.Node{
			Type:        types.AttributeNode,
			Name:        name,
			Value:       value,
			TextContent: value,
			Parent:      node,
			StartPos:    node.StartPos, // Approximate
			EndPos:      node.StartPos, // Approximate
		}
		attributes = append(attributes, attrNode)
	}

	return attributes
}

// applyNodeTest filters nodes based on node test
func (e *Evaluator) applyNodeTest(nodes []*types.Node, nodeTest string) []*types.Node {
	if nodeTest == "*" {
		// Match all element nodes (not text or other node types)
		var elementNodes []*types.Node
		for _, node := range nodes {
			if node.Type == types.ElementNode {
				elementNodes = append(elementNodes, node)
			}
		}
		return elementNodes
	}

	if nodeTest == "node()" {
		return nodes // Match all nodes
	}

	if nodeTest == "text()" {
		var textNodes []*types.Node
		for _, node := range nodes {
			if node.Type == types.TextNode {
				textNodes = append(textNodes, node)
			}
		}
		return textNodes
	}

	// Element name test
	var filtered []*types.Node
	for _, node := range nodes {
		if node.Type == types.ElementNode && strings.ToLower(node.Name) == strings.ToLower(nodeTest) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyPredicate filters nodes based on predicate
func (e *Evaluator) applyPredicate(nodes []*types.Node, predicate types.XPathPredicate, contextNode *types.Node) []*types.Node {
	expr := strings.TrimSpace(predicate.Expression)
	
	// Handle positional predicates like [1], [2], [last()]
	if pos, err := strconv.Atoi(expr); err == nil {
		if pos > 0 && pos <= len(nodes) {
			return []*types.Node{nodes[pos-1]}
		}
		return []*types.Node{}
	}

	if expr == "last()" {
		if len(nodes) > 0 {
			return []*types.Node{nodes[len(nodes)-1]}
		}
		return []*types.Node{}
	}

	// Handle 'and' logic like [@id and @class] - Check BEFORE simple attribute predicates
	if strings.Contains(expr, " and ") {
		return e.applyAndPredicate(nodes, expr)
	}

	// Handle 'or' logic like [@class='red' or @class='blue'] - Check BEFORE simple attribute predicates
	if strings.Contains(expr, " or ") {
		return e.applyOrPredicate(nodes, expr)
	}

	// Handle attribute predicates like [@id='test']
	if strings.HasPrefix(expr, "@") {
		return e.applyAttributePredicate(nodes, expr)
	}

	// Handle text predicates like [text()='content']
	if strings.HasPrefix(expr, "text()") {
		return e.applyTextPredicate(nodes, expr)
	}

	// Handle function predicates like [position()=2]
	if strings.Contains(expr, "position()") {
		return e.applyPositionPredicate(nodes, expr)
	}

	// Handle contains() function like [contains(text(), 'substring')]
	if strings.Contains(expr, "contains(") {
		return e.applyContainsPredicate(nodes, expr)
	}

	// Handle starts-with() function like [starts-with(@href, 'https')]
	if strings.Contains(expr, "starts-with(") {
		return e.applyStartsWithPredicate(nodes, expr)
	}

	// Default: return all nodes (predicate not implemented)
	return nodes
}

// applyAttributePredicate handles attribute-based predicates
func (e *Evaluator) applyAttributePredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	// Simple attribute existence: [@id]
	if !strings.Contains(expr, "=") {
		attrName := strings.TrimPrefix(expr, "@")
		for _, node := range nodes {
			if _, exists := node.Attributes[attrName]; exists {
				filtered = append(filtered, node)
			}
		}
		return filtered
	}

	// Attribute value comparison: [@id='test']
	parts := strings.SplitN(expr, "=", 2)
	if len(parts) != 2 {
		return nodes
	}

	attrName := strings.TrimPrefix(strings.TrimSpace(parts[0]), "@")
	expectedValue := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

	for _, node := range nodes {
		if value, exists := node.Attributes[attrName]; exists && value == expectedValue {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyTextPredicate handles text-based predicates
func (e *Evaluator) applyTextPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	// text()='content'
	if strings.Contains(expr, "=") {
		parts := strings.SplitN(expr, "=", 2)
		if len(parts) != 2 {
			return nodes
		}
		expectedText := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

		for _, node := range nodes {
			if node.TextContent == expectedText {
				filtered = append(filtered, node)
			}
		}
		return filtered
	}

	// text() (nodes with text content)
	for _, node := range nodes {
		if strings.TrimSpace(node.TextContent) != "" {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyPositionPredicate handles position-based predicates
func (e *Evaluator) applyPositionPredicate(nodes []*types.Node, expr string) []*types.Node {
	// position()=N
	if strings.Contains(expr, "=") {
		parts := strings.SplitN(expr, "=", 2)
		if len(parts) != 2 {
			return nodes
		}
		
		if pos, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			if pos > 0 && pos <= len(nodes) {
				return []*types.Node{nodes[pos-1]}
			}
		}
	}

	return nodes
}

// getAllNodes returns all nodes in the document tree
func (e *Evaluator) getAllNodes(root *types.Node) []*types.Node {
	var allNodes []*types.Node
	
	allNodes = append(allNodes, root)
	for _, child := range root.Children {
		allNodes = append(allNodes, e.getAllNodes(child)...)
	}

	return allNodes
}

// removeDuplicates removes duplicate nodes from the slice
func (e *Evaluator) removeDuplicates(nodes []*types.Node) []*types.Node {
	seen := make(map[*types.Node]bool)
	var unique []*types.Node

	for _, node := range nodes {
		if !seen[node] {
			seen[node] = true
			unique = append(unique, node)
		}
	}

	return unique
}

// applyContainsPredicate handles contains() function predicates
func (e *Evaluator) applyContainsPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	// Parse contains(text(), 'substring') or contains(@attr, 'substring')
	start := strings.Index(expr, "contains(")
	if start == -1 {
		return nodes
	}

	// Find the matching closing parenthesis
	depth := 0
	var end int
	for i := start + 9; i < len(expr); i++ {
		if expr[i] == '(' {
			depth++
		} else if expr[i] == ')' {
			if depth == 0 {
				end = i
				break
			}
			depth--
		}
	}

	if end == 0 {
		return nodes
	}

	// Extract arguments: "text(), 'substring'"
	args := expr[start+9 : end]
	parts := strings.Split(args, ",")
	if len(parts) != 2 {
		return nodes
	}

	source := strings.TrimSpace(parts[0])
	searchText := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

	for _, node := range nodes {
		var textToSearch string

		if source == "text()" {
			textToSearch = node.TextContent
		} else if strings.HasPrefix(source, "@") {
			attrName := strings.TrimPrefix(source, "@")
			if value, exists := node.Attributes[attrName]; exists {
				textToSearch = value
			}
		}

		if strings.Contains(textToSearch, searchText) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyStartsWithPredicate handles starts-with() function predicates
func (e *Evaluator) applyStartsWithPredicate(nodes []*types.Node, expr string) []*types.Node {
	var filtered []*types.Node

	// Parse starts-with(@attr, 'prefix') or starts-with(text(), 'prefix')
	start := strings.Index(expr, "starts-with(")
	if start == -1 {
		return nodes
	}

	// Find the matching closing parenthesis
	depth := 0
	var end int
	for i := start + 12; i < len(expr); i++ {
		if expr[i] == '(' {
			depth++
		} else if expr[i] == ')' {
			if depth == 0 {
				end = i
				break
			}
			depth--
		}
	}

	if end == 0 {
		return nodes
	}

	// Extract arguments
	args := expr[start+12 : end]
	parts := strings.Split(args, ",")
	if len(parts) != 2 {
		return nodes
	}

	source := strings.TrimSpace(parts[0])
	prefix := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

	for _, node := range nodes {
		var textToCheck string

		if source == "text()" {
			textToCheck = node.TextContent
		} else if strings.HasPrefix(source, "@") {
			attrName := strings.TrimPrefix(source, "@")
			if value, exists := node.Attributes[attrName]; exists {
				textToCheck = value
			}
		}

		if strings.HasPrefix(textToCheck, prefix) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyAndPredicate handles 'and' logic predicates
func (e *Evaluator) applyAndPredicate(nodes []*types.Node, expr string) []*types.Node {
	parts := strings.Split(expr, " and ")
	if len(parts) != 2 {
		return nodes
	}

	var filtered []*types.Node

	// Apply both conditions to each node
	for _, node := range nodes {
		firstCondition := strings.TrimSpace(parts[0])
		secondCondition := strings.TrimSpace(parts[1])

		// Check first condition
		firstMatches := e.evaluateSimpleCondition(node, firstCondition)
		if !firstMatches {
			continue
		}

		// Check second condition
		secondMatches := e.evaluateSimpleCondition(node, secondCondition)
		if secondMatches {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyOrPredicate handles 'or' logic predicates
func (e *Evaluator) applyOrPredicate(nodes []*types.Node, expr string) []*types.Node {
	parts := strings.Split(expr, " or ")
	if len(parts) != 2 {
		return nodes
	}

	var filtered []*types.Node
	seen := make(map[*types.Node]bool)

	firstCondition := strings.TrimSpace(parts[0])
	secondCondition := strings.TrimSpace(parts[1])

	// Check both conditions for each node
	for _, node := range nodes {
		if seen[node] {
			continue
		}

		// Check first condition
		if e.evaluateSimpleCondition(node, firstCondition) {
			seen[node] = true
			filtered = append(filtered, node)
			continue
		}

		// Check second condition
		if e.evaluateSimpleCondition(node, secondCondition) {
			seen[node] = true
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// evaluateSimpleCondition evaluates a simple condition against a single node
func (e *Evaluator) evaluateSimpleCondition(node *types.Node, condition string) bool {
	condition = strings.TrimSpace(condition)

	// Attribute existence: @id (handle spaced tokens like "@ id")
	if strings.HasPrefix(condition, "@") && !strings.Contains(condition, "=") {
		attrName := strings.TrimPrefix(condition, "@")
		attrName = strings.TrimSpace(attrName) // Handle "@ id" -> "id"
		_, exists := node.Attributes[attrName]
		return exists
	}

	// Attribute value comparison: @id="test" (handle spaced tokens)
	if strings.HasPrefix(condition, "@") && strings.Contains(condition, "=") {
		parts := strings.SplitN(condition, "=", 2)
		if len(parts) != 2 {
			return false
		}
		attrName := strings.TrimPrefix(strings.TrimSpace(parts[0]), "@")
		attrName = strings.TrimSpace(attrName) // Handle "@ id" -> "id"
		expectedValue := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		
		if value, exists := node.Attributes[attrName]; exists {
			return value == expectedValue
		}
		return false
	}

	// Text content conditions: text()="content"
	if strings.HasPrefix(condition, "text()") && strings.Contains(condition, "=") {
		parts := strings.SplitN(condition, "=", 2)
		if len(parts) != 2 {
			return false
		}
		expectedText := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		return node.TextContent == expectedText
	}

	// Text content existence: text()
	if condition == "text()" {
		return strings.TrimSpace(node.TextContent) != ""
	}

	// Contains function: contains(text(), "substring")
	if strings.Contains(condition, "contains(") {
		return len(e.applyContainsPredicate([]*types.Node{node}, condition)) > 0
	}

	// Starts-with function: starts-with(@attr, "prefix")
	if strings.Contains(condition, "starts-with(") {
		return len(e.applyStartsWithPredicate([]*types.Node{node}, condition)) > 0
	}

	// Default: condition not recognized
	return false
}
