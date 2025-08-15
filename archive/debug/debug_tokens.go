package main

import (
	"fmt"
)

// Copied from parser to debug tokenization
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

type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

func main() {
	expr := "//div"
	tokens := tokenize(expr)
	
	fmt.Printf("Tokenizing: %s\n", expr)
	for i, token := range tokens {
		fmt.Printf("Token %d: Type=%s, Value='%s', Pos=%d\n", 
			i, getTokenTypeName(token.Type), token.Value, token.Pos)
	}
}

func tokenize(expr string) []Token {
	var tokens []Token
	pos := 0

	for pos < len(expr) {
		// Skip whitespace
		if isWhitespace(expr[pos]) {
			pos++
			continue
		}

		// Double slash
		if pos < len(expr)-1 && expr[pos:pos+2] == "//" {
			tokens = append(tokens, Token{TokenDoubleSlash, "//", pos})
			pos += 2
			continue
		}

		// Single slash
		if expr[pos] == '/' {
			tokens = append(tokens, Token{TokenSlash, "/", pos})
			pos++
			continue
		}

		// Names (axes, node tests, functions)
		if isNameStart(expr[pos]) {
			start := pos
			for pos < len(expr) && isNameChar(expr[pos]) {
				pos++
			}
			name := expr[start:pos]

			// Default to node test
			tokens = append(tokens, Token{TokenNodeTest, name, start})
			continue
		}

		fmt.Printf("ERROR: unexpected character '%c' at position %d\n", expr[pos], pos)
		break
	}

	tokens = append(tokens, Token{TokenEOF, "", len(expr)})
	return tokens
}

func getTokenTypeName(t TokenType) string {
	switch t {
	case TokenAxis: return "TokenAxis"
	case TokenNodeTest: return "TokenNodeTest"
	case TokenSlash: return "TokenSlash"
	case TokenDoubleSlash: return "TokenDoubleSlash"
	case TokenEOF: return "TokenEOF"
	default: return "Unknown"
	}
}

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