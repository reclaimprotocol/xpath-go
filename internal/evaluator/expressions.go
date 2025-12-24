package evaluator

import (
	"fmt"
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
	leftVal := b.Left.Evaluate(node, evaluator)

	switch b.Operator {
	case "or":
		if isTruthy(leftVal) {
			return "true"
		}
		rightVal := b.Right.Evaluate(node, evaluator)
		if isTruthy(rightVal) {
			return "true"
		}
		return "false"
	case "and":
		if !isTruthy(leftVal) {
			return "false"
		}
		rightVal := b.Right.Evaluate(node, evaluator)
		if isTruthy(rightVal) {
			return "true"
		}
		return "false"
	}

	return "false"
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
	leftVal := strings.TrimPrefix(c.Left.Evaluate(node, evaluator), NodeSetPrefix)
	rightVal := strings.TrimPrefix(c.Right.Evaluate(node, evaluator), NodeSetPrefix)
	Trace("Compare: '%s' %s '%s'", leftVal, c.Operator, rightVal)

	// Try numeric comparison first
	leftNum, err1 := strconv.ParseFloat(leftVal, 64)
	rightNum, err2 := strconv.ParseFloat(rightVal, 64)

	if err1 == nil && err2 == nil {
		switch c.Operator {
		case "=":
			if leftNum == rightNum {
				return "true"
			}
		case "!=":
			if leftNum != rightNum {
				return "true"
			}
		case "<":
			if leftNum < rightNum {
				return "true"
			}
		case ">":
			if leftNum > rightNum {
				return "true"
			}
		case "<=":
			if leftNum <= rightNum {
				return "true"
			}
		case ">=":
			if leftNum >= rightNum {
				return "true"
			}
		}
		return "false"
	}

	// String comparison
	switch c.Operator {
	case "=":
		if leftVal == rightVal {
			return "true"
		}
	case "!=":
		if leftVal != rightVal {
			return "true"
		}
	}

	return "false"
}

func (c *ComparisonExpression) String() string {
	return fmt.Sprintf("%s %s %s", c.Left.String(), c.Operator, c.Right.String())
}

// ElementExpression represents check for a child element existence
type ElementExpression struct {
	Name string
}

func (e *ElementExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	if e.Name == "." {
		return NodeSetPrefix + node.TextContent
	}

	for _, child := range node.Children {
		if child.Type == types.ElementNode && (e.Name == "*" || child.Name == e.Name) {
			return NodeSetPrefix + child.TextContent
		}
	}
	return ""
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
	// For axis navigation in expressions, it returns true if any node matches
	if evaluator.evaluateAxisExpression(node, fmt.Sprintf("%s::%s", a.Axis, a.NodeTest)) {
		return "true"
	}
	return "false"
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
	var currentNodes []*types.Node
	if p.IsAbsolute {
		// Start from document root
		root := node
		for root.Parent != nil {
			root = root.Parent
		}
		if p.IsDeep {
			currentNodes = evaluator.getDescendantNodes(root, true)
		} else {
			currentNodes = []*types.Node{root}
		}
	} else {
		currentNodes = []*types.Node{node}
	}

	for _, step := range p.Steps {
		var nextNodes []*types.Node
		for _, n := range currentNodes {
			// Handle attribute step
			if strings.HasPrefix(step.Name, "@") {
				attrName := strings.TrimPrefix(step.Name, "@")
				if val, exists := n.Attributes[attrName]; exists {
					return NodeSetPrefix + val
				}
				continue
			}

			// Handle child axis for element step
			candidates := evaluator.getChildNodes(n)

			// Filter by name
			var matchingNodes []*types.Node
			for _, child := range candidates {
				if child.Type == types.ElementNode && (step.Name == "*" || child.Name == step.Name) {
					matchingNodes = append(matchingNodes, child)
				}
			}

			// Apply predicates sequentially
			currentStepNodes := matchingNodes
			for _, predicate := range step.Predicates {
				var filtered []*types.Node
				for i, m := range currentStepNodes {
					evaluator.contextPosition = i + 1
					evaluator.contextSize = len(currentStepNodes)
					if evaluator.evaluateSimpleCondition(m, predicate) {
						filtered = append(filtered, m)
					}
				}
				currentStepNodes = filtered
			}

			nextNodes = append(nextNodes, currentStepNodes...)
		}
		if len(nextNodes) == 0 && !strings.HasPrefix(step.Name, "@") {
			return ""
		}
		currentNodes = nextNodes
	}

	if len(currentNodes) > 0 {
		// Return the string value of the first node with a NodeSet marker
		first := currentNodes[0]
		val := ""
		if first.Type == types.TextNode {
			val = first.Value
		} else {
			val = first.TextContent
		}
		return NodeSetPrefix + val
	}
	return ""
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

func (a *ArithmeticExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	leftVal := strings.TrimPrefix(a.Left.Evaluate(node, evaluator), NodeSetPrefix)
	rightVal := strings.TrimPrefix(a.Right.Evaluate(node, evaluator), NodeSetPrefix)

	left, _ := strconv.ParseFloat(leftVal, 64)
	right, _ := strconv.ParseFloat(rightVal, 64)

	var result float64
	switch a.Operator {
	case "+":
		result = left + right
	case "-":
		result = left - right
	case "*":
		result = left * right
	case "div":
		if right != 0 {
			result = left / right
		}
	case "mod":
		if right != 0 {
			result = float64(int(left) % int(right))
		}
	}

	return strconv.FormatFloat(result, 'f', -1, 64)
}

func (a *ArithmeticExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", a.Left.String(), a.Operator, a.Right.String())
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
