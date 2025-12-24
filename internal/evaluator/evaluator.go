package evaluator

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/reclaimprotocol/xpath-go/internal/parser"
	"github.com/reclaimprotocol/xpath-go/pkg/types"
	"github.com/reclaimprotocol/xpath-go/pkg/utils"
)

// Evaluator handles XPath expression evaluation
type Evaluator struct {
	parser          *parser.Parser
	htmlParser      *utils.HTMLParser
	contextPosition int
	contextSize     int
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
	// Handle union expressions
	if len(xpath.Union) > 0 {
		var allResults []types.Node
		seenNodes := make(map[string]bool) // Use unique key for deduplication

		for _, unionExpr := range xpath.Union {
			results, err := e.evaluateSteps(unionExpr, document)
			if err != nil {
				return nil, err
			}

			// Add results while avoiding duplicates
			for _, result := range results {
				// Create unique key based on node properties
				key := fmt.Sprintf("%d:%s:%d:%d", result.Type, result.Name, result.StartPos, result.EndPos)
				if !seenNodes[key] {
					seenNodes[key] = true
					allResults = append(allResults, result)
				}
			}
		}

		// Sort results by document order (StartPos) for JavaScript compatibility
		e.sortNodesByDocumentOrder(allResults)

		return allResults, nil
	}

	// Handle regular (non-union) expressions
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

// applyNodeTest filters nodes based on node test
func (e *Evaluator) applyNodeTest(nodes []*types.Node, nodeTest string) []*types.Node {
	if nodeTest == "*" {
		// For attribute axis, match all attribute nodes
		// For other axes, match all element nodes
		var matchedNodes []*types.Node
		for _, node := range nodes {
			if node.Type == types.AttributeNode || node.Type == types.ElementNode {
				matchedNodes = append(matchedNodes, node)
			}
		}
		return matchedNodes
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

	// For attribute nodes, match by attribute name
	// For element nodes, match by element name
	var filtered []*types.Node
	for _, node := range nodes {
		if node.Type == types.AttributeNode && strings.EqualFold(node.Name, nodeTest) {
			filtered = append(filtered, node)
		} else if node.Type == types.ElementNode && strings.EqualFold(node.Name, nodeTest) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// applyPredicate filters nodes based on predicate
func (e *Evaluator) applyPredicate(nodes []*types.Node, predicate types.XPathPredicate, contextNode *types.Node) []*types.Node {
	expr := strings.TrimSpace(predicate.Expression)
	Trace("applyPredicate called with expr='%s', nodes=%d", expr, len(nodes))

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

	// Use the robust router to classify and route the expression
	return e.RoutePredicateExpression(nodes, expr)
}

// applyPositionalPredicate handles numeric position predicates like [1], [2], [last()]
func (e *Evaluator) applyPositionalPredicate(nodes []*types.Node, expr string) []*types.Node {
	expr = strings.TrimSpace(expr)

	// Handle numeric positions like [1], [2], [10]
	if pos, err := strconv.Atoi(expr); err == nil {
		if pos > 0 && pos <= len(nodes) {
			return []*types.Node{nodes[pos-1]}
		}
		return []*types.Node{}
	}

	// Handle last() function
	if expr == "last()" {
		if len(nodes) > 0 {
			return []*types.Node{nodes[len(nodes)-1]}
		}
		return []*types.Node{}
	}

	// Fallback
	return []*types.Node{}
}

// sortNodesByDocumentOrder sorts nodes by their document position
func (e *Evaluator) sortNodesByDocumentOrder(nodes []types.Node) {
	sort.Slice(nodes, func(i, j int) bool {
		// Sort by start position (document order)
		if nodes[i].StartPos != nodes[j].StartPos {
			return nodes[i].StartPos < nodes[j].StartPos
		}
		// If start positions are equal, sort by end position
		return nodes[i].EndPos < nodes[j].EndPos
	})
}

// isSimpleElementName checks if a condition is a simple element name (like span, a, div)
