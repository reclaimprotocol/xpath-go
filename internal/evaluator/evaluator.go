package evaluator

import (
	"fmt"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
	"github.com/reclaimprotocol/xpath-go/pkg/utils"
)

// Evaluator handles XPath expression evaluation
type Evaluator struct {
	htmlParser       *utils.HTMLParser
	contextPosition  int
	contextSize      int
	axisOrder        []*types.Node
	axisIndex        map[*types.Node]int
	axisSubtreeEnd   map[*types.Node]int
	axisMatchIndexes map[string][]int
	treeOrder        map[*types.Node]int
	attributeBase    map[*types.Node]int
	attributeRanks   map[*types.Node]map[string]int
	// attributeNodes retains generated DOM attribute nodes for the lifetime of
	// one document evaluation. Attribute nodes are virtual (the parser stores
	// attributes on their owning element), but repeated axis/path expressions
	// must observe stable node identity and should not rematerialize them.
	attributeNodes map[*types.Node][]*types.Node
}

// NewEvaluator creates a new XPath evaluator
func NewEvaluator() *Evaluator {
	return &Evaluator{
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
	program, err := Compile(xpathExpr)
	if err != nil {
		return nil, err
	}
	return e.EvaluateProgramWithDocument(program, content, prepareDocument)
}

// EvaluateProgramWithDocument evaluates a precompiled, immutable XPath
// program. The Evaluator remains per-call mutable because it owns the parsed
// document and axis/order indexes.
func (e *Evaluator) EvaluateProgramWithDocument(program *Program, content string, prepareDocument func(*types.Node)) ([]types.Node, error) {
	if program == nil || program.root == nil {
		return nil, fmt.Errorf("XPath program is nil")
	}

	// Parse HTML/XML content
	// Evaluators may be reused by internal callers, so document-scoped caches
	// must never leak between evaluations.
	e.attributeNodes = make(map[*types.Node][]*types.Node)
	documentNode, err := e.htmlParser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("HTML parsing failed: %w", err)
	}
	if prepareDocument != nil {
		prepareDocument(documentNode)
	}
	e.prepareAxisOrder(documentNode)

	// Evaluate the immutable typed tree directly.  In particular, this does not
	// call a parser: compiled XPath values are safe to share between goroutines.
	value := evaluateXPathValue(program.root, documentNode, e)
	if value.kind != nodeSetXPathValue {
		return nil, fmt.Errorf("XPath program did not select a node set")
	}
	nodes := e.removeDuplicates(value.nodes)
	e.sortNodePointersByTreeOrder(nodes, documentNode)
	results := make([]types.Node, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		result := *node
		result.Origin = node
		results = append(results, result)
	}
	return results, nil
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

// applyPredicate filters nodes using the typed expression carried by the
// execution AST. No public compatibility struct or source text reaches this
// hot path.
func (e *Evaluator) applyPredicate(nodes []*types.Node, expression Expression, contextNode *types.Node) []*types.Node {
	Trace("applyPredicate called with nodes=%d", len(nodes))

	// Numeric predicates select their one-indexed context position. Keep this
	// fast path typed so it does not depend on reparsing predicate source.
	if number, ok := expression.(*NumberExpression); ok {
		position := int(number.Value)
		if number.Value == float64(position) && position > 0 && position <= len(nodes) {
			return []*types.Node{nodes[position-1]}
		}
		return []*types.Node{}
	}
	if function, ok := expression.(*FunctionExpression); ok && function.Function.Name == "last" && len(function.Function.Arguments) == 0 {
		if len(nodes) > 0 {
			return []*types.Node{nodes[len(nodes)-1]}
		}
		return []*types.Node{}
	}

	return e.applyPositionContextPredicateExpression(nodes, expression)
}

// isSimpleElementName checks if a condition is a simple element name (like span, a, div)
