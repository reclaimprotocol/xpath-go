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

// SourceSpan identifies a byte range in the XPath source. Parser diagnostics
// use these offsets directly; retained spans also let future API layers point
// at a typed path/function node without reparsing its text.
type SourceSpan struct {
	Start int
	End   int
}

// BooleanExpression represents 'and'/'or' logic
type BooleanExpression struct {
	Left     Expression
	Operator string
	Right    Expression
	Span     SourceSpan
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
	Span     SourceSpan
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
	Span SourceSpan
}

func (e *ElementExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluateXPathValue(e, node, evaluator).legacyString()
}

func (e *ElementExpression) String() string {
	return e.Name
}

// AxisExpression represents an axis navigation in an expression
type AxisExpression struct {
	Axis       string
	NodeTest   string
	Predicates []Expression
	Following  []PathStep
	Span       SourceSpan
}

func (a *AxisExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluateXPathValue(a, node, evaluator).legacyString()
}

func (a *AxisExpression) String() string {
	var value strings.Builder
	fmt.Fprintf(&value, "%s::%s", a.Axis, a.NodeTest)
	for _, predicate := range a.Predicates {
		value.WriteString("[" + predicate.String() + "]")
	}
	for _, step := range a.Following {
		if step.Descendant {
			value.WriteString("//")
		} else {
			value.WriteString("/")
		}
		value.WriteString(pathStepString(step))
	}
	return value.String()
}

// PathStep represents a single step in a path
type PathStep struct {
	Name string
	Span SourceSpan
	// Axis is empty for the child axis. Keeping it on every path step means
	// explicit axes can occur at any depth of a parsed expression rather than
	// being represented by a special first-step expression.
	Axis       string
	Predicates []Expression
	// Descendant is the `//` separator immediately before this step. Keeping
	// it on the step preserves the compact predicate-expression AST while
	// making relative paths such as .//span unambiguous.
	Descendant bool
}

// PathExpression represents a path like head/title or */span
type PathExpression struct {
	Steps      []PathStep
	IsAbsolute bool
	IsDeep     bool // true if starts with //
	Span       SourceSpan
}

// FilterExpression applies predicates to the complete node-set produced by
// Base before evaluating an optional relative path. This is distinct from a
// predicate on a location step: (//item)[1] selects the first item in the
// complete result, while //item[1] selects the first item for each step
// context.
type FilterExpression struct {
	Base       Expression
	Predicates []Expression
	Following  []PathStep
	Span       SourceSpan
}

func (f *FilterExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluateXPathValue(f, node, evaluator).legacyString()
}

func (f *FilterExpression) String() string {
	value := "(" + f.Base.String() + ")"
	for _, predicate := range f.Predicates {
		value += "[" + predicate.String() + "]"
	}
	for _, step := range f.Following {
		if step.Descendant {
			value += "//"
		} else {
			value += "/"
		}
		value += pathStepString(step)
	}
	return value
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
	for index, step := range p.Steps {
		if index > 0 {
			if step.Descendant {
				parts = append(parts, "//")
			} else {
				parts = append(parts, "/")
			}
		}
		parts = append(parts, pathStepString(step))
	}
	return prefix + strings.Join(parts, "")
}

func pathStepString(step PathStep) string {
	value := step.Name
	if step.Axis == "attribute" {
		value = "@" + value
	} else if step.Axis != "" && step.Axis != "child" {
		value = step.Axis + "::" + value
	}
	for _, predicate := range step.Predicates {
		value += "[" + predicate.String() + "]"
	}
	return value
}

// FunctionCall is a typed function-call AST node. Source offsets are retained
// for diagnostics produced at compile time.
type FunctionCall struct {
	Name      string
	Arguments []Expression
	StartPos  int
	EndPos    int
	Span      SourceSpan
}

// FunctionExpression represents a function call
type FunctionExpression struct {
	Function *FunctionCall
	Span     SourceSpan
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
	Span  SourceSpan
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
	Span  SourceSpan
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
	Span SourceSpan
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
	Span     SourceSpan
}

// UnaryExpression represents XPath's recursive unary minus expression.
type UnaryExpression struct {
	Operand Expression
	Span    SourceSpan
}

func (u *UnaryExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return evaluateXPathValue(u, node, evaluator).legacyString()
}

func (u *UnaryExpression) String() string { return "-" + u.Operand.String() }

// UnionExpression retains all members of both node sets. Conversions such as
// string(A | B) use the first member in recovered DOM document order.
type UnionExpression struct {
	Operands []Expression
	Span     SourceSpan
}

type nodeSetIdentity struct {
	node      *types.Node
	owner     *types.Node
	namespace string
	localName string
	attribute bool
}

func nodeSetIdentityFor(candidate *types.Node) nodeSetIdentity {
	identity := nodeSetIdentity{node: candidate}
	if candidate == nil {
		return identity
	}
	if candidate.Type == types.AttributeNode {
		localName := candidate.LocalName
		if localName == "" {
			localName = candidate.Name
		}
		return nodeSetIdentity{owner: candidate.Parent, namespace: candidate.NamespaceURI, localName: localName, attribute: true}
	}
	if candidate.Origin != nil {
		identity.node = candidate.Origin
	}
	return identity
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
		if expr.Name == ".." {
			if node.Parent == nil {
				return nodeSetValue()
			}
			return nodeSetValue(node.Parent)
		}
		nodes := make([]*types.Node, 0)
		for _, child := range node.Children {
			if child.Type == types.ElementNode && (expr.Name == "*" || child.Name == expr.Name) {
				nodes = append(nodes, child)
			}
		}
		return nodeSetValue(nodes...)

	case *AxisExpression:
		nodes := evaluator.evaluateAxisNodesWithPredicates(node, expr.Axis, expr.NodeTest, expr.Predicates)
		if len(expr.Following) == 0 {
			return nodeSetValue(nodes...)
		}
		return nodeSetValue(evaluatePathSteps(expr.Following, nodes, evaluator)...)

	case *PathExpression:
		return nodeSetValue(evaluatePathNodes(expr, node, evaluator)...)

	case *FilterExpression:
		base := evaluateXPathValue(expr.Base, node, evaluator)
		if base.kind != nodeSetXPathValue {
			return xpathValue{kind: invalidXPathValue}
		}
		nodes := base.nodes
		for _, predicate := range expr.Predicates {
			nodes = evaluator.applyPredicate(nodes, predicate, node)
		}
		if len(expr.Following) > 0 {
			nodes = evaluatePathSteps(expr.Following, nodes, evaluator)
		}
		return nodeSetValue(nodes...)

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
		seen := make(map[nodeSetIdentity]struct{})
		for _, operand := range expr.Operands {
			value := evaluateXPathValue(operand, node, evaluator)
			if value.kind != nodeSetXPathValue {
				return xpathValue{kind: invalidXPathValue}
			}
			for _, candidate := range value.nodes {
				identity := nodeSetIdentityFor(candidate)
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

// evaluateAxisNodesWithPredicates evaluates a pre-parsed axis expression.
// Axis predicates appear inside already compiled outer predicates, so carrying
// their typed AST here avoids reparsing them for every outer context node.
func (evaluator *Evaluator) evaluateAxisNodesWithPredicates(node *types.Node, axis, nodeTest string, predicates []Expression) []*types.Node {
	// Following and preceding are global reverse/document-order axes. Selecting
	// [N] or [last()] through the precomputed matching-index cache avoids
	// materializing every later/earlier node for each context node.
	if (axis == "following" || axis == "preceding") && len(predicates) > 0 {
		position, last, positional := typedAxisPosition(predicates[0])
		if positional {
			candidate := evaluator.positionalAxisNode(node, types.XPathAxis(axis), nodeTest, position, last)
			if candidate == nil {
				return nil
			}
			result := []*types.Node{candidate}
			for _, predicate := range predicates[1:] {
				result = evaluator.applyPredicate(result, predicate, node)
			}
			return result
		}
	}
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
	// applyNodeTest carries the axis-sensitive wildcard rule: * means element
	// on ordinary axes and attribute on the attribute axis.
	result := evaluator.applyNodeTest(candidates, nodeTest)
	for _, predicate := range predicates {
		result = evaluator.applyPredicate(result, predicate, node)
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

func typedAxisPosition(expression Expression) (position int, last bool, ok bool) {
	if number, ok := expression.(*NumberExpression); ok {
		position = int(number.Value)
		return position, false, number.Value == float64(position) && position > 0
	}
	if function, ok := expression.(*FunctionExpression); ok && function.Function.Name == "last" && len(function.Function.Arguments) == 0 {
		return 0, true, true
	}
	return 0, false, false
}

func evaluatePathNodes(path *PathExpression, node *types.Node, evaluator *Evaluator) []*types.Node {
	oldPosition, oldSize := evaluator.contextPosition, evaluator.contextSize
	defer func() {
		evaluator.contextPosition, evaluator.contextSize = oldPosition, oldSize
	}()
	var currentNodes []*types.Node
	steps := path.Steps
	if path.IsAbsolute {
		root := node
		for root.Parent != nil {
			root = root.Parent
		}
		currentNodes = []*types.Node{root}
		// // is shorthand for /descendant-or-self::node()/ rather than
		// starting a child step from every descendant. Marking the first step
		// preserves that distinction for node tests and explicit axes alike.
		if path.IsDeep && len(steps) > 0 {
			steps = append([]PathStep(nil), steps...)
			steps[0].Descendant = true
		}
	} else {
		currentNodes = []*types.Node{node}
	}

	return evaluatePathSteps(steps, currentNodes, evaluator)
}

// evaluatePathSteps evaluates already parsed path steps from a caller-provided
// node set. Axis expressions use this after their typed node test, which keeps
// an explicit axis followed by a location path out of string-based parsing.
func evaluatePathSteps(steps []PathStep, currentNodes []*types.Node, evaluator *Evaluator) []*types.Node {
	for _, step := range steps {
		var nextNodes []*types.Node
		stepContexts := currentNodes
		if step.Descendant {
			stepContexts = make([]*types.Node, 0)
			for _, current := range currentNodes {
				stepContexts = append(stepContexts, evaluator.getDescendantNodes(current, true)...)
			}
		}
		for _, current := range stepContexts {
			if step.Axis != "" && step.Axis != "child" {
				matching := evaluator.evaluateAxisNodesWithPredicates(current, step.Axis, step.Name, step.Predicates)
				nextNodes = append(nextNodes, matching...)
				continue
			}
			if step.Name == "." {
				matching := []*types.Node{current}
				for _, predicate := range step.Predicates {
					matching = evaluator.applyPredicate(matching, predicate, current)
				}
				nextNodes = append(nextNodes, matching...)
				continue
			}
			if step.Name == ".." {
				matching := make([]*types.Node, 0, 1)
				if current.Parent != nil {
					matching = append(matching, current.Parent)
				}
				for _, predicate := range step.Predicates {
					matching = evaluator.applyPredicate(matching, predicate, current)
				}
				nextNodes = append(nextNodes, matching...)
				continue
			}
			if after, ok := strings.CutPrefix(step.Name, "@"); ok {
				name := after
				matching := make([]*types.Node, 0)
				for _, attribute := range evaluator.getAttributeNodes(current) {
					if name == "*" || attribute.Name == name {
						matching = append(matching, attribute)
					}
				}
				for _, predicate := range step.Predicates {
					matching = evaluator.applyPredicate(matching, predicate, current)
				}
				nextNodes = append(nextNodes, matching...)
				continue
			}
			matching := make([]*types.Node, 0)
			for _, child := range evaluator.getChildNodes(current) {
				if evaluator.matchesNodeTest(child, step.Name) {
					matching = append(matching, child)
				}
			}
			for _, predicate := range step.Predicates {
				matching = evaluator.applyPredicate(matching, predicate, current)
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
