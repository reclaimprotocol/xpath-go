package evaluator

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// FunctionCall represents a parsed function call
type FunctionCall struct {
	Name      string
	Arguments []Expression
	StartPos  int
	EndPos    int
}

// Expression represents any XPath expression component
type Expression interface {
	Evaluate(node *types.Node, evaluator *Evaluator) string
	String() string
}

// LiteralExpression represents a string or numeric literal
type LiteralExpression struct {
	Value string
}

func (l *LiteralExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return l.Value
}

func (l *LiteralExpression) String() string {
	return fmt.Sprintf("'%s'", l.Value)
}

// TextExpression represents text() function
type TextExpression struct{}

func (t *TextExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return node.TextContent
}

func (t *TextExpression) String() string {
	return "text()"
}

// DotExpression represents . operator (context node)
type DotExpression struct{}

func (d *DotExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	return node.TextContent
}

func (d *DotExpression) String() string {
	return "."
}

// AttributeExpression represents @attribute access
type AttributeExpression struct {
	Name string
}

func (a *AttributeExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	if value, exists := node.Attributes[a.Name]; exists {
		// For boolean attributes like 'disabled', 'checked', 'selected', etc.
		// If the attribute exists, it should be considered true regardless of value
		booleanAttrs := map[string]bool{
			"disabled": true, "checked": true, "selected": true, "readonly": true,
			"multiple": true, "autofocus": true, "autoplay": true, "controls": true,
			"defer": true, "hidden": true, "loop": true, "required": true,
		}

		if booleanAttrs[a.Name] {
			// For boolean attributes, return "true" if present
			return "true"
		}

		// For regular attributes, return the actual value
		return value
	}
	return ""
}

func (a *AttributeExpression) String() string {
	return fmt.Sprintf("@%s", a.Name)
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

// ArithmeticExpression represents arithmetic operations
type ArithmeticExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (a *ArithmeticExpression) Evaluate(node *types.Node, evaluator *Evaluator) string {
	leftVal := a.Left.Evaluate(node, evaluator)
	rightVal := a.Right.Evaluate(node, evaluator)

	left, err1 := strconv.ParseFloat(leftVal, 64)
	right, err2 := strconv.ParseFloat(rightVal, 64)

	if err1 != nil || err2 != nil {
		return ""
	}

	var result float64
	switch a.Operator {
	case "+":
		result = left + right
	case "-":
		result = left - right
	case "*":
		result = left * right
	case "/", "div":
		if right == 0 {
			return ""
		}
		result = left / right
	case "mod":
		if right == 0 {
			return ""
		}
		result = float64(int(left) % int(right))
	default:
		return ""
	}

	// Return clean integer if possible
	if result == float64(int(result)) {
		return strconv.Itoa(int(result))
	}
	return strconv.FormatFloat(result, 'f', -1, 64)
}

func (a *ArithmeticExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", a.Left.String(), a.Operator, a.Right.String())
}

// FunctionParser handles parsing of complex function expressions
type FunctionParser struct {
	input string
	pos   int
}

// NewFunctionParser creates a new function parser
func NewFunctionParser(input string) *FunctionParser {
	return &FunctionParser{
		input: strings.TrimSpace(input),
		pos:   0,
	}
}

// ParseExpression parses a complete expression
func (p *FunctionParser) ParseExpression() (Expression, error) {
	return p.parseComparison()
}

// parseComparison handles comparison operators (>, <, =, etc.)
func (p *FunctionParser) parseComparison() (Expression, error) {
	left, err := p.parseArithmetic()
	if err != nil {
		return nil, err
	}

	p.skipWhitespace()

	// Check for comparison operators
	if p.pos < len(p.input) {
		if p.peek() == '>' || p.peek() == '<' || p.peek() == '=' {
			// This is a comparison, not a standalone expression
			// Return the left side for function evaluation
			return left, nil
		}
	}

	return left, nil
}

// parseArithmetic handles arithmetic operators (+, -, *, /, mod)
func (p *FunctionParser) parseArithmetic() (Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}

		var operator string
		if p.peek() == '+' || p.peek() == '-' || p.peek() == '*' || p.peek() == '/' {
			operator = string(p.peek())
			p.pos++
		} else if p.match("div") {
			operator = "div"
		} else if p.match("mod") {
			operator = "mod"
		} else {
			break
		}

		p.skipWhitespace()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}

		left = &ArithmeticExpression{
			Left:     left,
			Operator: operator,
			Right:    right,
		}
	}

	return left, nil
}

// parsePrimary handles primary expressions (functions, literals, etc.)
func (p *FunctionParser) parsePrimary() (Expression, error) {
	p.skipWhitespace()

	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	// Handle parentheses
	if p.peek() == '(' {
		p.pos++ // consume '('
		expr, err := p.parseArithmetic()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.pos >= len(p.input) || p.peek() != ')' {
			return nil, fmt.Errorf("expected closing parenthesis")
		}
		p.pos++ // consume ')'
		return expr, nil
	}

	// Handle function calls
	if p.isFunction() {
		return p.parseFunction()
	}

	// Handle text()
	if p.match("text()") {
		return &TextExpression{}, nil
	}

	// Handle dot (.) operator
	if p.peek() == '.' {
		p.pos++
		return &DotExpression{}, nil
	}

	// Handle attributes (@attr)
	if p.peek() == '@' {
		return p.parseAttribute()
	}

	// Handle string literals
	if p.peek() == '\'' || p.peek() == '"' {
		return p.parseStringLiteral()
	}

	// Handle numeric literals
	if unicode.IsDigit(rune(p.peek())) {
		return p.parseNumericLiteral()
	}

	return nil, fmt.Errorf("unexpected character: %c at position %d", p.peek(), p.pos)
}

// isFunction checks if current position starts a function call
func (p *FunctionParser) isFunction() bool {
	// Look ahead to find function name followed by '('
	saved := p.pos
	defer func() { p.pos = saved }()

	// Read function name
	start := p.pos
	for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || p.input[p.pos] == '-') {
		p.pos++
	}

	if p.pos == start {
		return false
	}

	p.skipWhitespace()
	return p.pos < len(p.input) && p.peek() == '('
}

// parseFunction parses a function call
func (p *FunctionParser) parseFunction() (Expression, error) {
	// Read function name
	start := p.pos
	for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || p.input[p.pos] == '-') {
		p.pos++
	}

	if p.pos == start {
		return nil, fmt.Errorf("expected function name")
	}

	funcName := p.input[start:p.pos]

	p.skipWhitespace()
	if p.pos >= len(p.input) || p.peek() != '(' {
		return nil, fmt.Errorf("expected opening parenthesis after function name")
	}

	p.pos++ // consume '('

	var args []Expression

	// Parse arguments
	p.skipWhitespace()
	if p.pos < len(p.input) && p.peek() != ')' {
		for {
			arg, err := p.parseArithmetic()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			p.skipWhitespace()
			if p.pos >= len(p.input) {
				return nil, fmt.Errorf("expected closing parenthesis")
			}

			if p.peek() == ')' {
				break
			}

			if p.peek() != ',' {
				return nil, fmt.Errorf("expected comma or closing parenthesis")
			}

			p.pos++ // consume ','
			p.skipWhitespace()
		}
	}

	if p.pos >= len(p.input) || p.peek() != ')' {
		return nil, fmt.Errorf("expected closing parenthesis")
	}

	p.pos++ // consume ')'

	functionCall := &FunctionCall{
		Name:      funcName,
		Arguments: args,
		StartPos:  start,
		EndPos:    p.pos,
	}

	return &FunctionExpression{Function: functionCall}, nil
}

// parseAttribute parses @attribute
func (p *FunctionParser) parseAttribute() (Expression, error) {
	if p.peek() != '@' {
		return nil, fmt.Errorf("expected @ for attribute")
	}

	p.pos++ // consume '@'
	start := p.pos

	for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '-' || p.input[p.pos] == '_') {
		p.pos++
	}

	if p.pos == start {
		return nil, fmt.Errorf("expected attribute name after @")
	}

	attrName := p.input[start:p.pos]
	return &AttributeExpression{Name: attrName}, nil
}

// parseStringLiteral parses 'string' or "string"
func (p *FunctionParser) parseStringLiteral() (Expression, error) {
	quote := p.peek()
	if quote != '\'' && quote != '"' {
		return nil, fmt.Errorf("expected string quote")
	}

	p.pos++ // consume opening quote
	start := p.pos

	for p.pos < len(p.input) && p.peek() != quote {
		p.pos++
	}

	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unterminated string literal")
	}

	value := p.input[start:p.pos]
	p.pos++ // consume closing quote

	return &LiteralExpression{Value: value}, nil
}

// parseNumericLiteral parses numeric values
func (p *FunctionParser) parseNumericLiteral() (Expression, error) {
	start := p.pos

	for p.pos < len(p.input) && (unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '.') {
		p.pos++
	}

	if p.pos == start {
		return nil, fmt.Errorf("expected numeric literal")
	}

	value := p.input[start:p.pos]
	return &LiteralExpression{Value: value}, nil
}

// Helper functions
func (p *FunctionParser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *FunctionParser) match(s string) bool {
	if p.pos+len(s) > len(p.input) {
		return false
	}

	if p.input[p.pos:p.pos+len(s)] == s {
		// Check that it's not part of a larger word
		if p.pos+len(s) < len(p.input) {
			next := p.input[p.pos+len(s)]
			if unicode.IsLetter(rune(next)) || unicode.IsDigit(rune(next)) {
				return false
			}
		}

		p.pos += len(s)
		return true
	}
	return false
}

func (p *FunctionParser) skipWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

// evaluateFunction evaluates a function call using the new parser
func (e *Evaluator) evaluateFunction(fn *FunctionCall, node *types.Node) string {
	Trace("evaluateFunction: %s with %d arguments", fn.Name, len(fn.Arguments))

	switch fn.Name {
	case "string-length":
		if len(fn.Arguments) != 1 {
			return "0"
		}
		text := fn.Arguments[0].Evaluate(node, e)
		result := strconv.Itoa(len(text))
		Trace("string-length('%s') = %s", text, result)
		return result

	case "normalize-space":
		if len(fn.Arguments) != 1 {
			return ""
		}
		text := fn.Arguments[0].Evaluate(node, e)
		// Normalize whitespace: trim and collapse multiple spaces
		normalized := strings.Join(strings.Fields(text), " ")
		Trace("normalize-space('%s') = '%s'", text, normalized)
		return normalized

	case "substring":
		if len(fn.Arguments) < 2 || len(fn.Arguments) > 3 {
			return ""
		}

		text := fn.Arguments[0].Evaluate(node, e)
		startStr := fn.Arguments[1].Evaluate(node, e)

		start, err := strconv.Atoi(startStr)
		if err != nil {
			return ""
		}

		// XPath is 1-based, Go is 0-based
		start = start - 1
		if start < 0 {
			start = 0
		}
		if start >= len(text) {
			return ""
		}

		if len(fn.Arguments) == 2 {
			// substring(string, start) - from start to end
			result := text[start:]
			Trace("substring('%s', %d) = '%s'", text, start+1, result)
			return result
		} else {
			// substring(string, start, length)
			lengthStr := fn.Arguments[2].Evaluate(node, e)
			length, err := strconv.Atoi(lengthStr)
			if err != nil || length <= 0 {
				return ""
			}

			end := start + length
			if end > len(text) {
				end = len(text)
			}

			result := text[start:end]
			Trace("substring('%s', %d, %d) = '%s'", text, start+1, length, result)
			return result
		}

	case "contains":
		if len(fn.Arguments) != 2 {
			return "false"
		}
		text := fn.Arguments[0].Evaluate(node, e)
		search := fn.Arguments[1].Evaluate(node, e)
		if strings.Contains(text, search) {
			return "true"
		}
		return "false"

	case "starts-with":
		if len(fn.Arguments) != 2 {
			return "false"
		}
		text := fn.Arguments[0].Evaluate(node, e)
		prefix := fn.Arguments[1].Evaluate(node, e)
		if strings.HasPrefix(text, prefix) {
			return "true"
		}
		return "false"

	case "text":
		return node.TextContent

	case "true":
		return "true"

	case "false":
		return "false"

	default:
		Trace("unknown function: %s", fn.Name)
		return ""
	}
}
