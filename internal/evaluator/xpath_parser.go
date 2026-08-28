package evaluator

// This file is the single grammar used by compiled XPath programs.  It is
// deliberately independent of the legacy parser package: a Program owns the
// resulting expression tree and evaluation never turns source text back into
// syntax.

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type xpathTokenKind uint8

const (
	xpathEOF xpathTokenKind = iota
	xpathName
	xpathString
	xpathNumber
	xpathSlash
	xpathDoubleSlash
	xpathDot
	xpathDoubleDot
	xpathAt
	xpathStar
	xpathLBracket
	xpathRBracket
	xpathLParen
	xpathRParen
	xpathComma
	xpathPipe
	xpathDoubleColon
	xpathOperator
)

type xpathToken struct {
	kind       xpathTokenKind
	text       string
	start, end int // byte offsets in the original expression
}

type xpathLexer struct {
	source string
	pos    int
}

func lexXPath(source string) ([]xpathToken, error) {
	l := xpathLexer{source: source}
	var tokens []xpathToken
	for {
		for l.pos < len(source) {
			r, n := utf8.DecodeRuneInString(source[l.pos:])
			if !unicode.IsSpace(r) {
				break
			}
			l.pos += n
		}
		if l.pos == len(source) {
			return append(tokens, xpathToken{kind: xpathEOF, start: l.pos, end: l.pos}), nil
		}
		start := l.pos
		var r rune
		var n int
		one := func(kind xpathTokenKind, text string) {
			tokens = append(tokens, xpathToken{kind: kind, text: text, start: start, end: l.pos})
		}
		if strings.HasPrefix(source[l.pos:], "//") {
			l.pos += 2
			one(xpathDoubleSlash, "//")
			continue
		}
		if strings.HasPrefix(source[l.pos:], "..") {
			l.pos += 2
			one(xpathDoubleDot, "..")
			continue
		}
		if strings.HasPrefix(source[l.pos:], "::") {
			l.pos += 2
			one(xpathDoubleColon, "::")
			continue
		}
		if strings.HasPrefix(source[l.pos:], "!=") || strings.HasPrefix(source[l.pos:], "<=") || strings.HasPrefix(source[l.pos:], ">=") {
			l.pos += 2
			one(xpathOperator, source[start:l.pos])
			continue
		}
		ch := source[l.pos]
		switch ch {
		case '/':
			l.pos++
			one(xpathSlash, "/")
			continue
		case '.':
			if l.pos+1 < len(source) && isASCIIDigit(source[l.pos+1]) {
				goto number
			}
			l.pos++
			one(xpathDot, ".")
			continue
		case '@':
			l.pos++
			one(xpathAt, "@")
			continue
		case '*':
			l.pos++
			one(xpathStar, "*")
			continue
		case '[':
			l.pos++
			one(xpathLBracket, "[")
			continue
		case ']':
			l.pos++
			one(xpathRBracket, "]")
			continue
		case '(':
			l.pos++
			one(xpathLParen, "(")
			continue
		case ')':
			l.pos++
			one(xpathRParen, ")")
			continue
		case ',':
			l.pos++
			one(xpathComma, ",")
			continue
		case '|':
			l.pos++
			one(xpathPipe, "|")
			continue
		case '=', '<', '>', '+', '-':
			l.pos++
			one(xpathOperator, source[start:l.pos])
			continue
		case '\'', '"':
			quote := ch
			l.pos++
			valueStart := l.pos
			for l.pos < len(source) && source[l.pos] != quote {
				_, n := utf8.DecodeRuneInString(source[l.pos:])
				l.pos += n
			}
			if l.pos == len(source) {
				return nil, xpathSyntaxError(source, start, "unterminated string literal")
			}
			value := source[valueStart:l.pos]
			l.pos++
			tokens = append(tokens, xpathToken{kind: xpathString, text: value, start: start, end: l.pos})
			continue
		}
		if isASCIIDigit(ch) {
			goto number
		}
		r, n = utf8.DecodeRuneInString(source[l.pos:])
		if isXPathNameStart(r) {
			l.pos += n
			for l.pos < len(source) {
				r, n = utf8.DecodeRuneInString(source[l.pos:])
				if !isXPathNameChar(r) || (r == ':' && strings.HasPrefix(source[l.pos:], "::")) {
					break
				}
				l.pos += n
			}
			name := source[start:l.pos]
			tokens = append(tokens, xpathToken{kind: xpathName, text: name, start: start, end: l.pos})
			continue
		}
		return nil, xpathSyntaxError(source, start, fmt.Sprintf("unexpected character %q", r))

	number:
		l.pos = start
		dots := 0
		for l.pos < len(source) && (isASCIIDigit(source[l.pos]) || source[l.pos] == '.') {
			if source[l.pos] == '.' {
				dots++
				if dots > 1 {
					return nil, xpathSyntaxError(source, start, "invalid numeric literal")
				}
			}
			l.pos++
		}
		if source[start:l.pos] == "." {
			return nil, xpathSyntaxError(source, start, "invalid numeric literal")
		}
		one(xpathNumber, source[start:l.pos])
	}
}

func xpathSyntaxError(source string, pos int, message string) error {
	return fmt.Errorf("%s at byte %d", message, pos)
}

func isASCIIDigit(c byte) bool     { return c >= '0' && c <= '9' }
func isXPathNameStart(r rune) bool { return r == '_' || unicode.IsLetter(r) }
func isXPathNameChar(r rune) bool {
	return isXPathNameStart(r) || unicode.IsDigit(r) || r == '-' || r == '.' || r == ':'
}

// XPathParser parses complete expressions and predicate expressions with the
// same precedence rules.  The type is intentionally package-private: callers
// compile source to an immutable Program rather than retaining parser state.
type XPathParser struct {
	source string
	tokens []xpathToken
	pos    int
}

func parseXPath(source string) (Expression, error) {
	if strings.TrimSpace(source) == "" {
		return nil, xpathSyntaxError(source, 0, "empty XPath expression")
	}
	tokens, err := lexXPath(source)
	if err != nil {
		return nil, err
	}
	p := &XPathParser{source: source, tokens: tokens}
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.current().kind != xpathEOF {
		return nil, p.errorf(p.current(), "unexpected token %q", p.current().text)
	}
	return expr, nil
}

func (p *XPathParser) current() xpathToken {
	if p.pos >= len(p.tokens) {
		return xpathToken{kind: xpathEOF, start: len(p.source), end: len(p.source)}
	}
	return p.tokens[p.pos]
}
func (p *XPathParser) next() xpathToken {
	t := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return t
}
func (p *XPathParser) accept(kind xpathTokenKind, text string) bool {
	t := p.current()
	if t.kind == kind && (text == "" || t.text == text) {
		p.next()
		return true
	}
	return false
}
func (p *XPathParser) expect(kind xpathTokenKind, text string, what string) (xpathToken, error) {
	t := p.current()
	if t.kind != kind || text != "" && t.text != text {
		return t, p.errorf(t, "expected %s", what)
	}
	p.next()
	return t, nil
}
func (p *XPathParser) errorf(t xpathToken, format string, args ...any) error {
	return xpathSyntaxError(p.source, t.start, fmt.Sprintf(format, args...))
}

func (p *XPathParser) parseOr() (Expression, error) { return p.parseBinary(p.parseAnd, []string{"or"}) }
func (p *XPathParser) parseAnd() (Expression, error) {
	return p.parseBinary(p.parseEquality, []string{"and"})
}
func (p *XPathParser) parseEquality() (Expression, error) {
	return p.parseBinary(p.parseRelational, []string{"=", "!="})
}
func (p *XPathParser) parseRelational() (Expression, error) {
	return p.parseBinary(p.parseAdditive, []string{"<", ">", "<=", ">="})
}
func (p *XPathParser) parseAdditive() (Expression, error) {
	return p.parseBinary(p.parseMultiplicative, []string{"+", "-"})
}
func (p *XPathParser) parseMultiplicative() (Expression, error) {
	return p.parseBinary(p.parseUnary, []string{"*", "div", "mod"})
}
func (p *XPathParser) parseBinary(next func() (Expression, error), allowed []string) (Expression, error) {
	left, err := next()
	if err != nil {
		return nil, err
	}
	for {
		t := p.current()
		if t.kind == xpathStar && oneOf("*", allowed) {
			t.text = "*"
		} else if (t.kind != xpathOperator && t.kind != xpathName) || !oneOf(t.text, allowed) {
			return left, nil
		}
		p.next()
		right, err := next()
		if err != nil {
			return nil, err
		}
		span := SourceSpan{Start: expressionSpan(left).Start, End: expressionSpan(right).End}
		switch t.text {
		case "and", "or":
			left = &BooleanExpression{Left: left, Operator: t.text, Right: right, Span: span}
		case "+", "-", "*", "div", "mod":
			left = &ArithmeticExpression{Left: left, Operator: t.text, Right: right, Span: span}
		default:
			left = &ComparisonExpression{Left: left, Operator: t.text, Right: right, Span: span}
		}
	}
}
func oneOf(value string, allowed []string) bool {
	return slices.Contains(allowed, value)
}
func (p *XPathParser) parseUnary() (Expression, error) {
	if p.current().kind == xpathOperator && p.current().text == "-" {
		minus := p.next()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpression{Operand: operand, Span: SourceSpan{Start: minus.start, End: expressionSpan(operand).End}}, nil
	}
	return p.parseUnion()
}
func (p *XPathParser) parseUnion() (Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	var operands []Expression
	for p.accept(xpathPipe, "") {
		if operands == nil {
			operands = append(operands, left)
		}
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		operands = append(operands, right)
	}
	if operands != nil {
		return &UnionExpression{Operands: operands, Span: SourceSpan{Start: expressionSpan(operands[0]).Start, End: expressionSpan(operands[len(operands)-1]).End}}, nil
	}
	return left, nil
}

func (p *XPathParser) parsePrimary() (Expression, error) {
	t := p.current()
	var expression Expression
	var filterStart int
	switch t.kind {
	case xpathString:
		p.next()
		expression = &LiteralExpression{Value: t.text, Span: SourceSpan{Start: t.start, End: t.end}}
	case xpathNumber:
		p.next()
		value, err := strconv.ParseFloat(t.text, 64)
		if err != nil && !strings.Contains(err.Error(), "value out of range") {
			return nil, p.errorf(t, "invalid numeric literal %q", t.text)
		}
		expression = &NumberExpression{Value: value, Span: SourceSpan{Start: t.start, End: t.end}}
	case xpathLParen:
		filterStart = t.start
		p.next()
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(xpathRParen, "", "')'"); err != nil {
			return nil, err
		}
		expression = expr
	case xpathName:
		if p.looksLikeFunction() && !isNodeTestName(t.text) {
			var err error
			expression, err = p.parseFunction()
			if err != nil {
				return nil, err
			}
		}
	}
	if expression == nil && p.canStartLocationPath() {
		return p.parseLocationPath()
	}
	if expression == nil {
		return nil, p.errorf(t, "expected XPath expression")
	}
	return p.parseFilterSuffix(expression, filterStart)
}

// parseFilterSuffix implements XPath 1.0 FilterExpr and the FilterExpr path
// forms. In particular, it accepts (//item)[1]/text() without folding [1]
// into the final step of //item, which would change its meaning.
func (p *XPathParser) parseFilterSuffix(base Expression, start int) (Expression, error) {
	var predicates []Expression
	for p.accept(xpathLBracket, "") {
		if p.current().kind == xpathRBracket {
			return nil, p.errorf(p.current(), "empty predicate")
		}
		predicate, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(xpathRBracket, "", "']'"); err != nil {
			return nil, err
		}
		predicates = append(predicates, predicate)
	}

	var following []PathStep
	for p.current().kind == xpathSlash || p.current().kind == xpathDoubleSlash {
		deep := p.next().kind == xpathDoubleSlash
		if !p.canStartLocationPath() {
			return nil, p.errorf(p.current(), "expected location step")
		}
		step, err := p.parseLocationStep()
		if err != nil {
			return nil, err
		}
		step.Descendant = deep
		following = append(following, step)
	}

	if len(predicates) == 0 && len(following) == 0 {
		return base, nil
	}
	if start == 0 && expressionSpan(base).Start != 0 {
		start = expressionSpan(base).Start
	}
	return &FilterExpression{
		Base: base, Predicates: predicates, Following: following,
		Span: SourceSpan{Start: start, End: p.current().start},
	}, nil
}

func (p *XPathParser) looksLikeFunction() bool {
	return p.current().kind == xpathName && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].kind == xpathLParen
}
func isNodeTestName(name string) bool {
	return name == "node" || name == "text" || name == "comment" || name == "processing-instruction"
}
func (p *XPathParser) canStartLocationPath() bool {
	switch p.current().kind {
	case xpathSlash, xpathDoubleSlash, xpathDot, xpathDoubleDot, xpathAt, xpathStar:
		return true
	case xpathName:
		return !p.looksLikeFunction() || isNodeTestName(p.current().text)
	default:
		return false
	}
}

func (p *XPathParser) parseLocationPath() (Expression, error) {
	pathStart := p.current().start
	path := &PathExpression{Span: SourceSpan{Start: pathStart}}
	if p.accept(xpathSlash, "") {
		path.IsAbsolute = true
		if p.current().kind == xpathEOF || p.current().kind == xpathPipe || p.current().kind == xpathRParen || p.current().kind == xpathRBracket {
			path.Span.End = p.current().start
			return path, nil
		}
	} else if p.accept(xpathDoubleSlash, "") {
		path.IsAbsolute, path.IsDeep = true, true
		if !p.canStartLocationPath() {
			return nil, p.errorf(p.current(), "expected location step after '//'")
		}
	}
	step, err := p.parseLocationStep()
	if err != nil {
		return nil, err
	}
	path.Steps = append(path.Steps, step)
	for p.current().kind == xpathSlash || p.current().kind == xpathDoubleSlash {
		deep := p.next().kind == xpathDoubleSlash
		if !p.canStartLocationPath() {
			return nil, p.errorf(p.current(), "expected location step")
		}
		step, err = p.parseLocationStep()
		if err != nil {
			return nil, err
		}
		step.Descendant = deep
		path.Steps = append(path.Steps, step)
	}
	path.Span.End = p.current().start
	// Keep the compact AxisExpression form for an expression whose first step
	// is explicit. Later explicit axes remain normal PathSteps, which supports
	// arbitrary mixed-axis tails without a second grammar.
	if !path.IsAbsolute && !path.IsDeep && len(path.Steps) > 0 && path.Steps[0].Axis != "" && path.Steps[0].Axis != "attribute" {
		first := path.Steps[0]
		return &AxisExpression{Axis: first.Axis, NodeTest: first.Name, Predicates: first.Predicates, Following: path.Steps[1:], Span: path.Span}, nil
	}
	return path, nil
}

func (p *XPathParser) parseLocationStep() (step PathStep, err error) {
	step = PathStep{Span: SourceSpan{Start: p.current().start}}
	defer func() { step.Span.End = p.current().start }()
	if p.accept(xpathDot, "") {
		step.Name = "."
	} else if p.accept(xpathDoubleDot, "") {
		step.Name = ".."
	} else {
		if p.current().kind == xpathName && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].kind == xpathDoubleColon {
			axis := p.next()
			p.next()
			if !isSupportedExpressionAxis(axis.text) {
				return step, p.errorf(axis, "unsupported axis %q", axis.text)
			}
			step.Axis = axis.text
		}
		if p.accept(xpathAt, "") {
			if step.Axis != "" {
				return step, p.errorf(p.current(), "attribute shorthand cannot follow an explicit axis")
			}
			step.Axis = "attribute"
		}
		t := p.current()
		if p.accept(xpathStar, "") {
			step.Name = "*"
		} else if t.kind == xpathName {
			p.next()
			step.Name = t.text
			if p.accept(xpathLParen, "") {
				if !isNodeTestName(t.text) {
					return step, p.errorf(t, "function %q is not a node test in a location step", t.text)
				}
				if t.text == "processing-instruction" && p.current().kind == xpathString {
					target := p.next()
					step.Name += "('" + target.text + "')"
				}
				if _, err := p.expect(xpathRParen, "", "')'"); err != nil {
					return step, err
				}
				if t.text != "processing-instruction" && strings.Contains(step.Name, "(") {
					return step, p.errorf(t, "node test %q accepts no arguments", t.text)
				}
				if !strings.Contains(step.Name, "(") {
					step.Name += "()"
				}
			}
		} else {
			return step, p.errorf(t, "expected node test")
		}
	}
	for p.accept(xpathLBracket, "") {
		if p.current().kind == xpathRBracket {
			return step, p.errorf(p.current(), "empty predicate")
		}
		predicate, err := p.parseOr()
		if err != nil {
			return step, err
		}
		if _, err := p.expect(xpathRBracket, "", "']'"); err != nil {
			return step, err
		}
		step.Predicates = append(step.Predicates, predicate)
	}
	return step, nil
}

func (p *XPathParser) parseFunction() (Expression, error) {
	name := p.next()
	if _, err := p.expect(xpathLParen, "", "'('"); err != nil {
		return nil, err
	}
	args := make([]Expression, 0)
	if !p.accept(xpathRParen, "") {
		for {
			argument, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			args = append(args, argument)
			if p.accept(xpathRParen, "") {
				break
			}
			if _, err := p.expect(xpathComma, "", "',' or ')'"); err != nil {
				return nil, err
			}
		}
	}
	if err := validateFunctionArity(name.text, len(args)); err != nil {
		return nil, p.errorf(name, "%v", err)
	}
	span := SourceSpan{Start: name.start, End: p.current().start}
	return &FunctionExpression{Function: &FunctionCall{Name: name.text, Arguments: args, StartPos: name.start, EndPos: span.End, Span: span}, Span: span}, nil
}

func expressionSpan(expression Expression) SourceSpan {
	switch value := expression.(type) {
	case *BooleanExpression:
		return value.Span
	case *ComparisonExpression:
		return value.Span
	case *ElementExpression:
		return value.Span
	case *AxisExpression:
		return value.Span
	case *PathExpression:
		return value.Span
	case *FilterExpression:
		return value.Span
	case *FunctionExpression:
		return value.Span
	case *LiteralExpression:
		return value.Span
	case *NumberExpression:
		return value.Span
	case *AttributeExpression:
		return value.Span
	case *ArithmeticExpression:
		return value.Span
	case *UnaryExpression:
		return value.Span
	case *UnionExpression:
		return value.Span
	default:
		return SourceSpan{}
	}
}

// validateFunctionArity is shared by all contexts (top-level paths,
// predicates, and nested function arguments), so unsupported functions fail
// at Compile rather than when a candidate node is evaluated.
func validateFunctionArity(name string, arguments int) error {
	switch name {
	case "string", "number":
		if arguments > 1 {
			return fmt.Errorf("%s() expects zero or one argument, got %d", name, arguments)
		}
	case "boolean", "not", "count":
		if arguments != 1 {
			return fmt.Errorf("%s() expects exactly one argument, got %d", name, arguments)
		}
	case "true", "false", "text", "node", "comment", "position", "last":
		if arguments != 0 {
			return fmt.Errorf("%s() expects no arguments, got %d", name, arguments)
		}
	case "string-length", "normalize-space", "local-name", "name", "namespace-uri":
		if arguments > 1 {
			return fmt.Errorf("%s() expects zero or one argument, got %d", name, arguments)
		}
	case "contains", "starts-with":
		if arguments != 2 {
			return fmt.Errorf("%s() expects exactly two arguments, got %d", name, arguments)
		}
	case "substring":
		if arguments < 2 || arguments > 3 {
			return fmt.Errorf("substring() expects two or three arguments, got %d", arguments)
		}
	case "concat":
		if arguments < 2 {
			return fmt.Errorf("concat() expects at least two arguments, got %d", arguments)
		}
	default:
		return fmt.Errorf("unsupported function %q", name)
	}
	return nil
}

func isSupportedExpressionAxis(axis string) bool {
	switch axis {
	case "child", "descendant", "parent", "ancestor", "following-sibling",
		"preceding-sibling", "following", "preceding", "attribute", "self",
		"descendant-or-self", "ancestor-or-self":
		return true
	default:
		return false
	}
}
