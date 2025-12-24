package evaluator

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// FunctionCall represents a parsed function call
type FunctionCall struct {
	Name      string
	Arguments []Expression
	StartPos  int
	EndPos    int
}

// FunctionParser handles parsing of complex function expressions
type FunctionParser struct {
	input string
	pos   int
}

func NewFunctionParser(input string) *FunctionParser {
	return &FunctionParser{input: input}
}

func (p *FunctionParser) Parse() (Expression, error) {
	p.skipWhitespace()
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.pos < len(p.input) {
		return nil, fmt.Errorf("unexpected character at position %d: %c", p.pos, p.input[p.pos])
	}
	return expr, nil
}

func (p *FunctionParser) parseOr() (Expression, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.match("or") {
			right, err := p.parseAnd()
			if err != nil {
				return nil, err
			}
			left = &BooleanExpression{Left: left, Operator: "or", Right: right}
		} else {
			break
		}
	}
	return left, nil
}

func (p *FunctionParser) parseAnd() (Expression, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.match("and") {
			right, err := p.parseComparison()
			if err != nil {
				return nil, err
			}
			left = &BooleanExpression{Left: left, Operator: "and", Right: right}
		} else {
			break
		}
	}
	return left, nil
}

func (p *FunctionParser) parseComparison() (Expression, error) {
	left, err := p.parseArithmetic()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		op := ""
		if p.match("=") {
			op = "="
		} else if p.match("!=") {
			op = "!="
		} else if p.match("<=") {
			op = "<="
		} else if p.match(">=") {
			op = ">="
		} else if p.match("<") {
			op = "<"
		} else if p.match(">") {
			op = ">"
		}

		if op != "" {
			right, err := p.parseArithmetic()
			if err != nil {
				return nil, err
			}
			left = &ComparisonExpression{Left: left, Operator: op, Right: right}
		} else {
			break
		}
	}
	return left, nil
}

func (p *FunctionParser) parseArithmetic() (Expression, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		op := ""
		if p.match("+") {
			op = "+"
		} else if p.match("-") {
			op = "-"
		}

		if op != "" {
			right, err := p.parseTerm()
			if err != nil {
				return nil, err
			}
			left = &ArithmeticExpression{Left: left, Operator: op, Right: right}
		} else {
			break
		}
	}
	return left, nil
}

func (p *FunctionParser) parseTerm() (Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		op := ""
		if p.match("*") {
			op = "*"
		} else if p.match("div") {
			op = "div"
		} else if p.match("mod") {
			op = "mod"
		}

		if op != "" {
			right, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			left = &ArithmeticExpression{Left: left, Operator: op, Right: right}
		} else {
			break
		}
	}
	return left, nil
}

func (p *FunctionParser) parsePrimary() (Expression, error) {
	p.skipWhitespace()
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	// Handle parentheses
	if p.peek() == '(' {
		p.pos++ // consume '('
		expr, err := p.parseOr()
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

	// Handle string literals
	if p.peek() == '\'' || p.peek() == '"' {
		return p.parseStringLiteral()
	}

	// Handle numeric literals
	if unicode.IsDigit(rune(p.peek())) || (p.peek() == '.' && p.pos+1 < len(p.input) && unicode.IsDigit(rune(p.input[p.pos+1]))) {
		return p.parseNumber()
	}

	// Handle attributes
	if p.peek() == '@' {
		return p.parseAttribute()
	}

	// Handle element names (for boolean checks like [span])
	if unicode.IsLetter(rune(p.peek())) || p.peek() == '*' || p.peek() == '/' || p.peek() == '.' {
		return p.parseStepExpression()
	}

	return nil, fmt.Errorf("unexpected character: %c at position %d", p.peek(), p.pos)
}

// parseStepExpression parses a name that could be an element, axis, or a path
func (p *FunctionParser) parseStepExpression() (Expression, error) {
	p.skipWhitespace()

	isAbsolute := false
	isDeep := false
	if p.peek() == '/' {
		isAbsolute = true
		p.pos++
		if p.peek() == '/' {
			isDeep = true
			p.pos++
		}
	}
	p.skipWhitespace()

	start := p.pos
	var firstStepName string

	if p.pos < len(p.input) && p.peek() == '*' {
		p.pos++
		firstStepName = "*"
	} else if p.pos < len(p.input) && p.peek() == '@' {
		p.pos++
		startAttr := p.pos
		for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '-' || p.input[p.pos] == '_') {
			p.pos++
		}
		firstStepName = "@" + p.input[startAttr:p.pos]
	} else if p.pos < len(p.input) && p.peek() == '.' {
		p.pos++
		firstStepName = "."
	} else {
		for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '-' || p.input[p.pos] == '_') {
			p.pos++
		}
		firstStepName = p.input[start:p.pos]
	}

	p.skipWhitespace()

	// 1. Check for axis prefix (::)
	if p.pos+1 < len(p.input) && p.input[p.pos:p.pos+2] == "::" {
		p.pos += 2
		p.skipWhitespace()
		nodeTestExpr, err := p.parseStepExpression()
		if err != nil {
			return nil, err
		}
		return &AxisExpression{Axis: firstStepName, NodeTest: nodeTestExpr.String()}, nil
	}

	// 2. Check for function call
	if p.peek() == '(' {
		p.pos = start
		return p.parseFunction()
	}

	// 3. Optional predicates (can be multiple)
	var firstStepPredicates []string
	for p.peek() == '[' {
		p.pos++ // consume '['
		pStart := p.pos
		depth := 1
		for p.pos < len(p.input) && depth > 0 {
			switch p.input[p.pos] {
			case '[':
				depth++
			case ']':
				depth--
			}
			p.pos++
		}
		firstStepPredicates = append(firstStepPredicates, p.input[pStart:p.pos-1])
		p.skipWhitespace()
	}

	// 4. Check for path (/) or Finish
	if p.peek() == '/' || isAbsolute || len(firstStepPredicates) > 0 {
		steps := []PathStep{}
		if firstStepName != "" || len(firstStepPredicates) > 0 {
			steps = append(steps, PathStep{Name: firstStepName, Predicates: firstStepPredicates})
		}

		for p.peek() == '/' {
			p.pos++ // consume '/'
			p.skipWhitespace()
			sStart := p.pos
			var sName string
			if p.peek() == '*' {
				p.pos++
				sName = "*"
			} else if p.peek() == '@' {
				p.pos++
				saStart := p.pos
				for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '-' || p.input[p.pos] == '_') {
					p.pos++
				}
				sName = "@" + p.input[saStart:p.pos]
			} else {
				for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '-' || p.input[p.pos] == '_') {
					p.pos++
				}
				sName = p.input[sStart:p.pos]
			}
			p.skipWhitespace()

			var sPreds []string
			for p.peek() == '[' {
				p.pos++
				psStart := p.pos
				depth := 1
				for p.pos < len(p.input) && depth > 0 {
					switch p.input[p.pos] {
					case '[':
						depth++
					case ']':
						depth--
					}
					p.pos++
				}
				sPreds = append(sPreds, p.input[psStart:p.pos-1])
				p.skipWhitespace()
			}
			steps = append(steps, PathStep{Name: sName, Predicates: sPreds})
		}
		return &PathExpression{Steps: steps, IsAbsolute: isAbsolute, IsDeep: isDeep}, nil
	}

	if firstStepName != "" && strings.HasPrefix(firstStepName, "@") {
		return &AttributeExpression{Name: strings.TrimPrefix(firstStepName, "@")}, nil
	}

	return &ElementExpression{Name: firstStepName}, nil
}

// parseFunction parses a function call like contains(@class, 'active')
func (p *FunctionParser) parseFunction() (Expression, error) {
	startPos := p.pos
	start := p.pos
	for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '-' || p.input[p.pos] == '_') {
		p.pos++
	}

	name := p.input[start:p.pos]
	if name == "" {
		return nil, fmt.Errorf("expected function name")
	}

	p.skipWhitespace()
	if p.pos >= len(p.input) || p.peek() != '(' {
		return nil, fmt.Errorf("expected '(' after function name")
	}
	p.pos++ // consume '('

	var args []Expression
	for {
		p.skipWhitespace()
		if p.peek() == ')' {
			p.pos++ // consume ')'
			break
		}

		arg, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		p.skipWhitespace()
		if p.peek() == ',' {
			p.pos++
		} else if p.peek() == ')' {
			p.pos++
			break
		} else {
			return nil, fmt.Errorf("expected ',' or ')' in function arguments")
		}
	}

	return &FunctionExpression{
		Function: &FunctionCall{
			Name:      name,
			Arguments: args,
			StartPos:  startPos,
			EndPos:    p.pos,
		},
	}, nil
}

// parseNumber parses a numeric literal
func (p *FunctionParser) parseNumber() (Expression, error) {
	start := p.pos
	for p.pos < len(p.input) && (unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '.') {
		p.pos++
	}

	val, err := strconv.ParseFloat(p.input[start:p.pos], 64)
	if err != nil {
		return nil, err
	}

	return &NumberExpression{Value: val}, nil
}

// parseAttribute parses an attribute selector like @class
func (p *FunctionParser) parseAttribute() (Expression, error) {
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
		return nil, fmt.Errorf("unexpected end of input in string literal")
	}

	val := p.input[start:p.pos]
	p.pos++ // consume closing quote
	return &LiteralExpression{Value: val}, nil
}

func (p *FunctionParser) peek() byte {
	if p.pos < len(p.input) {
		return p.input[p.pos]
	}
	return 0
}

func (p *FunctionParser) skipWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *FunctionParser) match(s string) bool {
	if p.pos+len(s) > len(p.input) {
		return false
	}

	if p.input[p.pos:p.pos+len(s)] == s {
		// Check that it's not part of a larger word (only for word operators)
		isWord := true
		for _, char := range s {
			if !unicode.IsLetter(rune(char)) {
				isWord = false
				break
			}
		}

		if isWord && p.pos+len(s) < len(p.input) {
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
