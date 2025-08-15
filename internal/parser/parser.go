package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Parser handles XPath expression parsing
type Parser struct {
	expression string
	tokens     []Token
	position   int
}

// Token represents a parsed XPath token
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// TokenType represents the type of XPath token
type TokenType int

const (
	TokenAxis TokenType = iota
	TokenNodeTest
	TokenPredicate
	TokenOperator
	TokenFunction
	TokenLiteral
	TokenNumber
	TokenVariable
	TokenSlash
	TokenDoubleSlash
	TokenDot
	TokenDoubleDot
	TokenAt
	TokenLeftBracket
	TokenRightBracket
	TokenLeftParen
	TokenRightParen
	TokenComma
	TokenPipe
	TokenEOF
)

// NewParser creates a new XPath parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses an XPath expression into a structured representation
func (p *Parser) Parse(expression string) (*types.ParsedXPath, error) {
	p.expression = strings.TrimSpace(expression)
	if p.expression == "" {
		return nil, fmt.Errorf("empty XPath expression")
	}

	// Tokenize the expression
	if err := p.tokenize(); err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	// Parse tokens into steps
	parsed, err := p.parseExpression()
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	return parsed, nil
}

// tokenize breaks down the XPath expression into tokens
func (p *Parser) tokenize() error {
	p.tokens = []Token{}
	p.position = 0

	expr := p.expression
	pos := 0

	for pos < len(expr) {
		// Skip whitespace
		if isWhitespace(expr[pos]) {
			pos++
			continue
		}

		// Double slash
		if pos < len(expr)-1 && expr[pos:pos+2] == "//" {
			p.tokens = append(p.tokens, Token{TokenDoubleSlash, "//", pos})
			pos += 2
			continue
		}

		// Single slash
		if expr[pos] == '/' {
			p.tokens = append(p.tokens, Token{TokenSlash, "/", pos})
			pos++
			continue
		}

		// Double dot
		if pos < len(expr)-1 && expr[pos:pos+2] == ".." {
			p.tokens = append(p.tokens, Token{TokenDoubleDot, "..", pos})
			pos += 2
			continue
		}

		// Single dot
		if expr[pos] == '.' {
			p.tokens = append(p.tokens, Token{TokenDot, ".", pos})
			pos++
			continue
		}

		// At symbol
		if expr[pos] == '@' {
			p.tokens = append(p.tokens, Token{TokenAt, "@", pos})
			pos++
			continue
		}

		// Brackets
		if expr[pos] == '[' {
			p.tokens = append(p.tokens, Token{TokenLeftBracket, "[", pos})
			pos++
			continue
		}
		if expr[pos] == ']' {
			p.tokens = append(p.tokens, Token{TokenRightBracket, "]", pos})
			pos++
			continue
		}

		// Parentheses
		if expr[pos] == '(' {
			p.tokens = append(p.tokens, Token{TokenLeftParen, "(", pos})
			pos++
			continue
		}
		if expr[pos] == ')' {
			p.tokens = append(p.tokens, Token{TokenRightParen, ")", pos})
			pos++
			continue
		}

		// Pipe for union
		if expr[pos] == '|' {
			p.tokens = append(p.tokens, Token{TokenPipe, "|", pos})
			pos++
			continue
		}

		// String literals
		if expr[pos] == '"' || expr[pos] == '\'' {
			quote := expr[pos]
			start := pos
			pos++
			for pos < len(expr) && expr[pos] != quote {
				pos++
			}
			if pos >= len(expr) {
				return fmt.Errorf("unterminated string literal at position %d", start)
			}
			pos++ // Skip closing quote
			p.tokens = append(p.tokens, Token{TokenLiteral, expr[start:pos], start})
			continue
		}

		// Numbers
		if isDigit(expr[pos]) {
			start := pos
			for pos < len(expr) && (isDigit(expr[pos]) || expr[pos] == '.') {
				pos++
			}
			p.tokens = append(p.tokens, Token{TokenNumber, expr[start:pos], start})
			continue
		}

		// Operators
		if op := p.parseOperator(expr, pos); op != "" {
			p.tokens = append(p.tokens, Token{TokenOperator, op, pos})
			pos += len(op)
			continue
		}

		// Names (axes, node tests, functions)
		if isNameStart(expr[pos]) {
			start := pos
			for pos < len(expr) && isNameChar(expr[pos]) {
				pos++
			}
			name := expr[start:pos]

			// Check if it's an axis
			if pos < len(expr) && expr[pos] == ':' && pos+1 < len(expr) && expr[pos+1] == ':' {
				p.tokens = append(p.tokens, Token{TokenAxis, name + "::", start})
				pos += 2
				continue
			}

			// Check if it's a function
			if pos < len(expr) && expr[pos] == '(' {
				p.tokens = append(p.tokens, Token{TokenFunction, name, start})
				continue
			}

			// Default to node test
			p.tokens = append(p.tokens, Token{TokenNodeTest, name, start})
			continue
		}

		return fmt.Errorf("unexpected character '%c' at position %d", expr[pos], pos)
	}

	p.tokens = append(p.tokens, Token{TokenEOF, "", len(expr)})
	return nil
}

// parseOperator attempts to parse an operator at the given position
func (p *Parser) parseOperator(expr string, pos int) string {
	operators := []string{
		"!=", "<=", ">=", "and", "or", "not", "div", "mod",
		"=", "<", ">", "+", "-", "*",
	}

	for _, op := range operators {
		if pos+len(op) <= len(expr) && expr[pos:pos+len(op)] == op {
			// For word operators, ensure they're not part of a larger identifier
			if isLetter(op[0]) {
				if (pos > 0 && isNameChar(expr[pos-1])) ||
					(pos+len(op) < len(expr) && isNameChar(expr[pos+len(op)])) {
					continue
				}
			}
			return op
		}
	}
	return ""
}

// parseExpression parses the tokenized expression into steps
func (p *Parser) parseExpression() (*types.ParsedXPath, error) {
	p.position = 0
	
	parsed := &types.ParsedXPath{
		Steps:      []types.XPathStep{},
		IsAbsolute: false,
	}

	// Check if expression starts with / (absolute path)
	if p.currentToken().Type == TokenSlash || p.currentToken().Type == TokenDoubleSlash {
		parsed.IsAbsolute = true
		if p.currentToken().Type == TokenDoubleSlash {
			// Add descendant-or-self step for //
			parsed.Steps = append(parsed.Steps, types.XPathStep{
				Axis:     types.AxisDescendantOrSelf,
				NodeTest: "node",
			})
		}
		p.advance()
	}

	// Parse location steps
	for p.currentToken().Type != TokenEOF {
		step, err := p.parseLocationStep()
		if err != nil {
			return nil, err
		}
		parsed.Steps = append(parsed.Steps, *step)

		// Check for path separator
		if p.currentToken().Type == TokenSlash {
			p.advance()
		} else if p.currentToken().Type == TokenDoubleSlash {
			p.advance()
			// Add descendant-or-self step
			parsed.Steps = append(parsed.Steps, types.XPathStep{
				Axis:     types.AxisDescendantOrSelf,
				NodeTest: "node",
			})
		} else if p.currentToken().Type != TokenEOF {
			break // Could be union or other operator
		}
	}

	return parsed, nil
}

// parseLocationStep parses a single location step
func (p *Parser) parseLocationStep() (*types.XPathStep, error) {
	step := &types.XPathStep{
		Axis:       types.AxisChild, // Default axis
		NodeTest:   "",
		Predicates: []types.XPathPredicate{},
	}

	// Parse axis
	if p.currentToken().Type == TokenAxis {
		axisName := strings.TrimSuffix(p.currentToken().Value, "::")
		step.Axis = types.XPathAxis(axisName)
		p.advance()
	} else if p.currentToken().Type == TokenAt {
		step.Axis = types.AxisAttribute
		p.advance()
	} else if p.currentToken().Type == TokenDot {
		step.Axis = types.AxisSelf
		step.NodeTest = "node"
		p.advance()
		return step, nil
	} else if p.currentToken().Type == TokenDoubleDot {
		step.Axis = types.AxisParent
		step.NodeTest = "node"
		p.advance()
		return step, nil
	}

	// Parse node test
	if p.currentToken().Type == TokenNodeTest {
		step.NodeTest = p.currentToken().Value
		p.advance()
	} else if p.currentToken().Type == TokenFunction {
		step.NodeTest = p.currentToken().Value + "()"
		p.advance()
		if p.currentToken().Type == TokenLeftParen {
			p.advance()
			if p.currentToken().Type == TokenRightParen {
				p.advance()
			}
		}
	} else if step.Axis == types.AxisAttribute {
		// For attribute axis, the node test might be missing (select all attributes)
		step.NodeTest = "*"
	} else {
		return nil, fmt.Errorf("expected node test at position %d", p.currentToken().Pos)
	}

	// Parse predicates
	for p.currentToken().Type == TokenLeftBracket {
		predicate, err := p.parsePredicate()
		if err != nil {
			return nil, err
		}
		step.Predicates = append(step.Predicates, *predicate)
	}

	return step, nil
}

// parsePredicate parses a predicate expression
func (p *Parser) parsePredicate() (*types.XPathPredicate, error) {
	if p.currentToken().Type != TokenLeftBracket {
		return nil, fmt.Errorf("expected '[' at position %d", p.currentToken().Pos)
	}
	p.advance()

	// Collect all tokens until closing bracket
	start := p.position
	depth := 1
	
	for depth > 0 && p.currentToken().Type != TokenEOF {
		if p.currentToken().Type == TokenLeftBracket {
			depth++
		} else if p.currentToken().Type == TokenRightBracket {
			depth--
		}
		if depth > 0 {
			p.advance()
		}
	}

	if p.currentToken().Type != TokenRightBracket {
		return nil, fmt.Errorf("unterminated predicate")
	}

	// Extract predicate expression
	predicateTokens := p.tokens[start:p.position]
	expression := p.tokensToString(predicateTokens)
	
	p.advance() // Skip closing bracket

	return &types.XPathPredicate{
		Expression: expression,
		Parsed:     nil, // TODO: Parse predicate expression
	}, nil
}

// tokensToString converts tokens back to string representation
func (p *Parser) tokensToString(tokens []Token) string {
	var parts []string
	for _, token := range tokens {
		parts = append(parts, token.Value)
	}
	return strings.Join(parts, "")
}

// currentToken returns the current token
func (p *Parser) currentToken() Token {
	if p.position >= len(p.tokens) {
		return Token{TokenEOF, "", 0}
	}
	return p.tokens[p.position]
}

// advance moves to the next token
func (p *Parser) advance() {
	if p.position < len(p.tokens) {
		p.position++
	}
}

// Helper functions for character classification
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNameStart(c byte) bool {
	return isLetter(c) || c == '_'
}

func isNameChar(c byte) bool {
	return isNameStart(c) || isDigit(c) || c == '-' || c == '.'
}