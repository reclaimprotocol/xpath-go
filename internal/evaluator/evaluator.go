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
	parser           *parser.Parser
	htmlParser       *utils.HTMLParser
	contextPosition  int
	contextSize      int
	axisOrder        []*types.Node
	axisIndex        map[*types.Node]int
	axisSubtreeEnd   map[*types.Node]int
	axisMatchIndexes map[string][]int
}

// NewEvaluator creates a new XPath evaluator
func NewEvaluator() *Evaluator {
	return &Evaluator{
		parser:     parser.NewParser(),
		htmlParser: utils.NewHTMLParser(),
	}
}

// SetScriptingEnabled configures whole-document noscript parsing for the next
// and subsequent evaluations performed by this evaluator.
func (e *Evaluator) SetScriptingEnabled(enabled bool) {
	e.htmlParser.SetScriptingEnabled(enabled)
}

// Evaluate evaluates an XPath expression against HTML/XML content
func (e *Evaluator) Evaluate(xpathExpr, content string) ([]types.Node, error) {
	return e.EvaluateWithDocument(xpathExpr, content, nil)
}

// EvaluateWithDocument evaluates an XPath expression after parsing content and
// before evaluating it. prepareDocument is intended for callers that maintain
// a source view separate from the parser input (for example, a decoded
// character-set view) and need to adjust source locations on the parsed tree.
// It must not change the DOM structure or text used for XPath evaluation.
func (e *Evaluator) EvaluateWithDocument(xpathExpr, content string, prepareDocument func(*types.Node)) ([]types.Node, error) {
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
	if prepareDocument != nil {
		prepareDocument(documentNode)
	}
	e.prepareAxisOrder(documentNode)

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
				// Attribute nodes do not have independent source offsets. Include
				// their owning element identity so equal-named attributes on two
				// selected elements are not collapsed by a union expression.
				key := fmt.Sprintf("%p:%d:%s:%d:%d", result.Origin, result.Type, result.Name, result.StartPos, result.EndPos)
				if result.Type == types.AttributeNode {
					key = fmt.Sprintf("attr:%p:%s:%s", result.Parent, result.NamespaceURI, result.LocalName)
				}
				if !seenNodes[key] {
					seenNodes[key] = true
					allResults = append(allResults, result)
				}
			}
		}

		// Sort results by document order (StartPos) for JavaScript compatibility
		e.sortNodesByDocumentOrder(allResults, document)

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
	e.sortNodePointersByTreeOrder(currentNodes, document)
	var results []types.Node
	for _, node := range currentNodes {
		if node != nil {
			result := *node
			result.Origin = node
			results = append(results, result)
		}
	}

	return results, nil
}

// evaluateStep evaluates a single XPath step
func (e *Evaluator) evaluateStep(step types.XPathStep, contextNode *types.Node) []*types.Node {
	if (step.Axis == types.AxisFollowing || step.Axis == types.AxisPreceding) && len(step.Predicates) > 0 {
		if position, last, ok := simpleAxisPosition(step.Predicates[0].Expression); ok {
			candidate := e.positionalAxisNode(contextNode, step.Axis, step.NodeTest, position, last)
			if candidate == nil {
				return nil
			}
			filtered := []*types.Node{candidate}
			for _, predicate := range step.Predicates[1:] {
				filtered = e.applyPredicate(filtered, predicate, contextNode)
			}
			return filtered
		}
	}

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
	case types.AxisFollowing:
		candidates = e.getFollowingNodes(contextNode)
	case types.AxisPreceding:
		candidates = e.getPrecedingNodes(contextNode)
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

	// Reverse axes use reverse document order while evaluating predicates,
	// but XPath node-set results are exposed in document order.
	if step.Axis == types.AxisPrecedingSibling || step.Axis == types.AxisPreceding {
		for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
			filtered[i], filtered[j] = filtered[j], filtered[i]
		}
	}

	return filtered
}

func simpleAxisPosition(expression string) (position int, last bool, ok bool) {
	expression = strings.TrimSpace(expression)
	if expression == "last()" {
		return 0, true, true
	}
	position, err := strconv.Atoi(expression)
	if err != nil || position < 1 {
		return 0, false, false
	}
	return position, false, true
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

	if nodeTest == "comment()" {
		var comments []*types.Node
		for _, node := range nodes {
			if node.Type == types.CommentNode {
				comments = append(comments, node)
			}
		}
		return comments
	}

	if nodeTest == "processing-instruction()" || strings.HasPrefix(nodeTest, "processing-instruction(") {
		target := ""
		if nodeTest != "processing-instruction()" {
			target = strings.TrimSuffix(strings.TrimPrefix(nodeTest, "processing-instruction("), ")")
			target = strings.Trim(target, `"'`)
		}
		var instructions []*types.Node
		for _, node := range nodes {
			if node.Type == types.ProcessingInstructionNode && (target == "" || node.Name == target) {
				instructions = append(instructions, node)
			}
		}
		return instructions
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

// sortNodesByDocumentOrder sorts nodes by their document position
func (e *Evaluator) sortNodesByDocumentOrder(nodes []types.Node, document *types.Node) {
	order, attributeBase, attributeRanks := treeOrderMaps(document)
	sort.SliceStable(nodes, func(i, j int) bool {
		return treeOrderForNode(&nodes[i], order, attributeBase, attributeRanks) < treeOrderForNode(&nodes[j], order, attributeBase, attributeRanks)
	})
}

// isSimpleElementName checks if a condition is a simple element name (like span, a, div)
