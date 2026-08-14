package evaluator

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

const (
	NodeSetPrefix = "\x01"
)

// Expression represents an evaluatable XPath expression
type Expression interface {
	Evaluate(node *types.Node, evaluator *Evaluator) string
	String() string
}

// BooleanExpression represents 'and'/'or' logic
type BooleanExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (b *BooleanExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluateXPathValue(b, node, evaluator).legacyString()
}

func (b *BooleanExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left.String(), b.Operator, b.Right.String())
}

// ComparisonExpression represents =, !=, <, >, <=, >=
type ComparisonExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (c *ComparisonExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluateXPathValue(c, node, evaluator).legacyString()
}

func (c *ComparisonExpression) String() string {
	return fmt.Sprintf("%s %s %s", c.Left.String(), c.Operator, c.Right.String())
}

// ElementExpression represents check for a child element existence
type ElementExpression struct {
	Name string
}

func (e *ElementExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluateXPathValue(e, node, evaluator).legacyString()
}

func (e *ElementExpression) String() string {
	return e.Name
}

// AxisExpression represents an axis navigation in an expression
type AxisExpression struct {
	Axis     string
	NodeTest string
}

func (a *AxisExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluateXPathValue(a, node, evaluator).legacyString()
}

func (a *AxisExpression) String() string {
	return fmt.Sprintf("%s::%s", a.Axis, a.NodeTest)
}

// PathStep represents a single step in a path
type PathStep struct {
	Name       string
	Predicates []string
}

// PathExpression represents a path like head/title or */span
type PathExpression struct {
	Steps      []PathStep
	IsAbsolute bool
	IsDeep     bool // true if starts with //
}

func (p *PathExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluateXPathValue(p, node, evaluator).legacyString()
}

func (p *PathExpression) String() string {
	var parts []string
	prefix := ""
	if p.IsAbsolute {
		prefix = "/"
		if p.IsDeep {
			prefix = "//"
		}
	}
	for _, step := range p.Steps {
		s := step.Name
		for _, pred := range step.Predicates {
			s += "[" + pred + "]"
		}
		parts = append(parts, s)
	}
	return prefix + strings.Join(parts, "/")
}

// FunctionExpression represents a function call
type FunctionExpression struct {
	Function *FunctionCall
}

func (f *FunctionExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluator.evaluateFunction(f.Function, node)
}

func (f *FunctionExpression) String() string {
	var args []string
	for _, arg := range f.Function.Arguments {
		args = append(args, arg.String())
	}
	return fmt.Sprintf("%s(%s)", f.Function.Name, strings.Join(args, ", "))
}

// LiteralExpression represents a string literal
type LiteralExpression struct {
	Value string
}

func (l *LiteralExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return l.Value
}

func (l *LiteralExpression) String() string {
	return fmt.Sprintf("'%s'", l.Value)
}

// NumberExpression represents a numeric constant
type NumberExpression struct {
	Value float64
}

func (n *NumberExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return strconv.FormatFloat(n.Value, 'f', -1, 64)
}

func (n *NumberExpression) String() string {
	return strconv.FormatFloat(n.Value, 'f', -1, 64)
}

// AttributeExpression represents @attribute
type AttributeExpression struct {
	Name string
}

func (a *AttributeExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	if val, exists := node.Attributes[a.Name]; exists {
		return NodeSetPrefix + val
	}
	return ""
}

func (a *AttributeExpression) String() string {
	return "@" + a.Name
}

// ArithmeticExpression handles +, -, *, div, mod
type ArithmeticExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

// UnaryExpression represents XPath's recursive unary minus expression.
type UnaryExpression struct {
	Operand Expression
}

func (u *UnaryExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluateXPathValue(u, node, evaluator).legacyString()
}

func (u *UnaryExpression) String() string { return "-" + u.Operand.String() }

// UnionExpression retains all members of both node sets. Conversions such as
// string(A | B) use the first member in recovered DOM document order.
type UnionExpression struct {
	Operands []Expression
}

func (u *UnionExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluateXPathValue(u, node, evaluator).legacyString()
}

func (u *UnionExpression) String() string {
	parts := make([]string, 0, len(u.Operands))
	for _, operand := range u.Operands {
		parts = append(parts, operand.String())
	}
	return strings.Join(parts, " | ")
}

func (a *ArithmeticExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluateXPathValue(a, node, evaluator).legacyString()
}

func (a *ArithmeticExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", a.Left.String(), a.Operator, a.Right.String())
}

func evaluateXPathValue(expression Expression, node *types.Node, evaluator *Evaluator) xpathValue {
	switch expr := expression.(type) {
	case *BooleanExpression:
		left := evaluateXPathValue(expr.Left, node, evaluator)
		if expr.Operator == "or" {
			if left.toBoolean() {
				return booleanValue(true)
			}
			return booleanValue(evaluateXPathValue(expr.Right, node, evaluator).toBoolean())
		}
		if expr.Operator == "and" {
			if !left.toBoolean() {
				return booleanValue(false)
			}
			return booleanValue(evaluateXPathValue(expr.Right, node, evaluator).toBoolean())
		}
		return booleanValue(false)

	case *ComparisonExpression:
		left := evaluateXPathValue(expr.Left, node, evaluator)
		right := evaluateXPathValue(expr.Right, node, evaluator)
		return booleanValue(compareTypedXPathValues(left, expr.Operator, right))

	case *ElementExpression:
		if expr.Name == "." {
			return nodeSetValue(node)
		}
		nodes := make([]*types.Node, 0)
		for _, child := range node.Children {
			if child.Type == types.ElementNode && (expr.Name == "*" || child.Name == expr.Name) {
				nodes = append(nodes, child)
			}
		}
		return nodeSetValue(nodes...)

	case *AxisExpression:
		return nodeSetValue(evaluator.evaluateAxisNodes(node, expr.Axis, expr.NodeTest)...)

	case *PathExpression:
		return nodeSetValue(evaluatePathNodes(expr, node, evaluator)...)

	case *FunctionExpression:
		return evaluator.evaluateFunctionValue(expr.Function, node)

	case *LiteralExpression:
		return stringValue(expr.Value)

	case *NumberExpression:
		return numberValue(expr.Value)

	case *AttributeExpression:
		attributes := evaluator.getAttributeNodes(node)
		nodes := make([]*types.Node, 0, len(attributes))
		for _, attribute := range attributes {
			if expr.Name == "*" || attribute.Name == expr.Name {
				nodes = append(nodes, attribute)
			}
		}
		return nodeSetValue(nodes...)

	case *ArithmeticExpression:
		left := evaluateXPathValue(expr.Left, node, evaluator).toNumber()
		right := evaluateXPathValue(expr.Right, node, evaluator).toNumber()
		result := math.NaN()
		switch expr.Operator {
		case "+":
			result = left + right
		case "-":
			result = left - right
		case "*":
			result = left * right
		case "div":
			result = left / right
		case "mod":
			result = math.Mod(left, right)
		}
		return numberValue(result)

	case *UnaryExpression:
		return numberValue(-evaluateXPathValue(expr.Operand, node, evaluator).toNumber())

	case *UnionExpression:
		var nodes []*types.Node
		type unionIdentity struct {
			node      *types.Node
			owner     *types.Node
			namespace string
			localName string
			attribute bool
		}
		seen := make(map[unionIdentity]struct{})
		for _, operand := range expr.Operands {
			value := evaluateXPathValue(operand, node, evaluator)
			if value.kind != nodeSetXPathValue {
				return xpathValue{kind: invalidXPathValue}
			}
			for _, candidate := range value.nodes {
				identity := unionIdentity{node: candidate}
				if candidate.Type == types.AttributeNode {
					localName := candidate.LocalName
					if localName == "" {
						localName = candidate.Name
					}
					identity = unionIdentity{owner: candidate.Parent, namespace: candidate.NamespaceURI, localName: localName, attribute: true}
				} else if candidate.Origin != nil {
					identity.node = candidate.Origin
				}
				if _, exists := seen[identity]; exists {
					continue
				}
				seen[identity] = struct{}{}
				nodes = append(nodes, candidate)
			}
		}
		root := node
		for root.Parent != nil {
			root = root.Parent
		}
		evaluator.sortNodePointersByTreeOrder(nodes, root)
		return nodeSetValue(nodes...)
	}
	return xpathValue{kind: invalidXPathValue}
}

func (evaluator *Evaluator) evaluateAxisNodes(node *types.Node, axis, nodeTest string) []*types.Node {
	var candidates []*types.Node
	switch axis {
	case "parent":
		if node.Parent != nil {
			candidates = []*types.Node{node.Parent}
		}
	case "ancestor":
		candidates = evaluator.getAncestorNodes(node, false)
	case "ancestor-or-self":
		candidates = evaluator.getAncestorNodes(node, true)
		for left, right := 0, len(candidates)-1; left < right; left, right = left+1, right-1 {
			candidates[left], candidates[right] = candidates[right], candidates[left]
		}
	case "following-sibling":
		candidates = evaluator.getFollowingSiblings(node)
	case "preceding-sibling":
		candidates = evaluator.getPrecedingSiblings(node)
	case "following", "preceding":
		root := node
		if node.Type == types.AttributeNode && node.Parent != nil {
			root = node.Parent
		}
		for root.Parent != nil {
			root = root.Parent
		}
		if len(evaluator.axisOrder) == 0 || evaluator.axisIndex[axisTreeIdentity(root)] == 0 && evaluator.axisOrder[0] != axisTreeIdentity(root) {
			evaluator.prepareAxisOrder(root)
		}
		if axis == "following" {
			candidates = evaluator.getFollowingNodes(node)
		} else {
			candidates = evaluator.getPrecedingNodes(node)
		}
	case "self":
		candidates = []*types.Node{node}
	case "child":
		candidates = evaluator.getChildNodes(node)
	case "descendant":
		candidates = evaluator.getDescendantNodes(node, false)
	case "descendant-or-self":
		candidates = evaluator.getDescendantNodes(node, true)
	case "attribute":
		candidates = evaluator.getAttributeNodes(node)
	}
	baseTest, predicates := splitAxisNodeTest(nodeTest)
	result := make([]*types.Node, 0, len(candidates))
	for _, candidate := range candidates {
		if evaluator.matchesNodeTest(candidate, baseTest) {
			result = append(result, candidate)
		}
	}
	for _, predicate := range predicates {
		result = evaluator.applyUnifiedPredicate(result, predicate)
	}
	root := node
	if node.Type == types.AttributeNode && node.Parent != nil {
		root = node.Parent
	}
	for root.Parent != nil {
		root = root.Parent
	}
	evaluator.sortNodePointersByTreeOrder(result, root)
	return result
}

func splitAxisNodeTest(nodeTest string) (string, []string) {
	start := strings.IndexByte(nodeTest, '[')
	if start < 0 {
		return nodeTest, nil
	}
	base := strings.TrimSpace(nodeTest[:start])
	var predicates []string
	for position := start; position < len(nodeTest); {
		if nodeTest[position] != '[' {
			position++
			continue
		}
		contentStart, depth := position+1, 1
		position++
		for position < len(nodeTest) && depth > 0 {
			switch nodeTest[position] {
			case '[':
				depth++
			case ']':
				depth--
			}
			position++
		}
		if depth == 0 {
			predicates = append(predicates, nodeTest[contentStart:position-1])
		}
	}
	return base, predicates
}

func evaluatePathNodes(path *PathExpression, node *types.Node, evaluator *Evaluator) []*types.Node {
	oldPosition, oldSize := evaluator.contextPosition, evaluator.contextSize
	defer func() {
		evaluator.contextPosition, evaluator.contextSize = oldPosition, oldSize
	}()
	var currentNodes []*types.Node
	if path.IsAbsolute {
		root := node
		for root.Parent != nil {
			root = root.Parent
		}
		if path.IsDeep {
			currentNodes = evaluator.getDescendantNodes(root, true)
		} else {
			currentNodes = []*types.Node{root}
		}
	} else {
		currentNodes = []*types.Node{node}
	}

	for _, step := range path.Steps {
		var nextNodes []*types.Node
		for _, current := range currentNodes {
			if strings.HasPrefix(step.Name, "@") {
				name := strings.TrimPrefix(step.Name, "@")
				for _, attribute := range evaluator.getAttributeNodes(current) {
					if name == "*" || attribute.Name == name {
						nextNodes = append(nextNodes, attribute)
					}
				}
				continue
			}
			matching := make([]*types.Node, 0)
			for _, child := range evaluator.getChildNodes(current) {
				if child.Type == types.ElementNode && (step.Name == "*" || child.Name == step.Name) {
					matching = append(matching, child)
				}
			}
			for _, predicate := range step.Predicates {
				filtered := make([]*types.Node, 0, len(matching))
				for index, candidate := range matching {
					evaluator.contextPosition = index + 1
					evaluator.contextSize = len(matching)
					if evaluator.evaluateSimpleCondition(candidate, predicate) {
						filtered = append(filtered, candidate)
					}
				}
				matching = filtered
			}
			nextNodes = append(nextNodes, matching...)
		}
		currentNodes = nextNodes
		if len(currentNodes) == 0 {
			break
		}
	}
	return currentNodes
}

// isTruthy implements XPath truthiness rules
func isTruthy(val string) bool {
	if strings.HasPrefix(val, NodeSetPrefix) {
		return true // Node-set is true if not empty
	}
	if val == "true" {
		return true
	}
	if val == "false" || val == "" || val == "0" {
		return false
	}
	if num, err := strconv.ParseFloat(val, 64); err == nil {
		return num != 0
	}
	return len(val) > 0
}
