package utils

import (
	"fmt"
	stdhtml "html"
	"strings"
	"unicode/utf8"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func (p *HTMLParser) consumeNumericCharacterReference() (rune, bool) {
	if p.pos+2 >= len(p.content) || p.content[p.pos:p.pos+2] != "&#" {
		return 0, false
	}

	position := p.pos + 2
	base := uint32(10)
	if position < len(p.content) && (p.content[position] == 'x' || p.content[position] == 'X') {
		base = 16
		position++
	}
	digitsStart := position
	value := uint32(0)
	overflow := false
	for position < len(p.content) {
		digit, ok := numericReferenceDigit(p.content[position], base)
		if !ok {
			break
		}
		if !overflow {
			if value > (0x10FFFF-digit)/base {
				overflow = true
			} else {
				value = value*base + digit
			}
		}
		position++
	}
	if position == digitsStart {
		return 0, false
	}
	if position < len(p.content) && p.content[position] == ';' {
		position++
	}

	p.advanceRune(position - p.pos)
	if overflow || value == 0 || value >= 0xD800 && value <= 0xDFFF {
		return '\uFFFD', true
	}
	if replacement, ok := numericReferenceControlReplacement[value]; ok {
		value = replacement
	}
	return rune(value), true
}

func numericReferenceDigit(character byte, base uint32) (uint32, bool) {
	switch {
	case character >= '0' && character <= '9':
		return uint32(character - '0'), true
	case base == 16 && character >= 'A' && character <= 'F':
		return uint32(character-'A') + 10, true
	case base == 16 && character >= 'a' && character <= 'f':
		return uint32(character-'a') + 10, true
	default:
		return 0, false
	}
}

var numericReferenceControlReplacement = map[uint32]uint32{
	0x80: 0x20AC,
	0x82: 0x201A,
	0x83: 0x0192,
	0x84: 0x201E,
	0x85: 0x2026,
	0x86: 0x2020,
	0x87: 0x2021,
	0x88: 0x02C6,
	0x89: 0x2030,
	0x8A: 0x0160,
	0x8B: 0x2039,
	0x8C: 0x0152,
	0x8E: 0x017D,
	0x91: 0x2018,
	0x92: 0x2019,
	0x93: 0x201C,
	0x94: 0x201D,
	0x95: 0x2022,
	0x96: 0x2013,
	0x97: 0x2014,
	0x98: 0x02DC,
	0x99: 0x2122,
	0x9A: 0x0161,
	0x9B: 0x203A,
	0x9C: 0x0153,
	0x9E: 0x017E,
	0x9F: 0x0178,
}

// consumeNamedCharacterReference decodes the longest named reference from the
// current ampersand. The standard library supplies the complete WHATWG entity
// table, including legacy semicolonless and two-code-point names. Attribute
// values apply HTML's historical ambiguity exception: a semicolonless match
// followed by an ASCII alphanumeric or '=' remains literal.
func (p *HTMLParser) consumeNamedCharacterReference(attribute bool) (string, bool) {
	if p.pos+1 >= len(p.content) || p.content[p.pos] != '&' || p.content[p.pos+1] == '#' {
		return "", false
	}

	nameEnd := p.pos + 1
	// CounterClockwiseContourIntegral; is the longest WHATWG entity name at
	// 32 bytes. No lookup can succeed beyond this bound, so do not scan an
	// untrusted alphanumeric run quadratically.
	const longestNamedCharacterReference = len("CounterClockwiseContourIntegral;")
	lookupLimit := min(p.pos+1+longestNamedCharacterReference, len(p.content))
	for nameEnd < lookupLimit && isASCIIAlphanumeric(p.content[nameEnd]) {
		nameEnd++
	}
	if nameEnd == p.pos+1 {
		return "", false
	}
	candidateEnd := nameEnd
	if candidateEnd < len(p.content) && p.content[candidateEnd] == ';' {
		candidateEnd++
	}

	// Infer the longest actual table match from UnescapeString while keeping
	// any unmatched suffix out of the consumed range. A longer candidate that
	// merely preserves more suffix has the form previousDecode + rawSuffix;
	// a different result identifies a new, longer entity-table entry.
	matchEnd := 0
	matchValue := ""
	for end := p.pos + 2; end <= candidateEnd; end++ {
		candidate := p.content[p.pos:end]
		decoded := stdhtml.UnescapeString(candidate)
		if decoded == candidate {
			continue
		}
		if matchEnd == 0 || decoded != matchValue+p.content[matchEnd:end] {
			matchEnd = end
			matchValue = decoded
		}
	}
	if matchEnd == 0 {
		return "", false
	}
	matchedSemicolon := p.content[matchEnd-1] == ';'
	if attribute && !matchedSemicolon && matchEnd < len(p.content) && (isASCIIAlphanumeric(p.content[matchEnd]) || p.content[matchEnd] == '=') {
		return "", false
	}

	p.advanceRune(matchEnd - p.pos)
	return matchValue, true
}

func isASCIIAlphanumeric(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9'
}

// startsMarkupToken reports whether '<' enters a markup tokenizer state. A
// less-than sign followed by any other character remains ordinary data text.
func (p *HTMLParser) startsMarkupToken() bool {
	if p.pos >= len(p.content) || p.content[p.pos] != '<' || p.pos+1 >= len(p.content) {
		return false
	}

	next := p.content[p.pos+1]
	// At EOF, the end-tag-open state emits both '<' and '/' as text.
	if next == '/' && p.pos+2 >= len(p.content) {
		return false
	}
	return isASCIIAlpha(next) || next == '!' || next == '?' || next == '/'
}

func (p *HTMLParser) isMissingEndTagName() bool {
	return p.pos+2 < len(p.content) && p.content[p.pos:p.pos+3] == "</>"
}

func (p *HTMLParser) isBogusEndTagOpen() bool {
	if p.pos+2 >= len(p.content) || p.content[p.pos:p.pos+2] != "</" {
		return false
	}
	next := p.content[p.pos+2]
	return next != '>' && !isASCIIAlpha(next)
}

// parseBogusEndTagComment handles an invalid first character after "</".
// The slash and less-than sign are tokenizer syntax and are not comment data.
func (p *HTMLParser) parseBogusEndTagComment(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if !p.isBogusEndTagOpen() {
		return nil, fmt.Errorf("expected invalid end-tag opener at position %d", p.pos)
	}

	p.advanceRune(2) // Skip "</".
	var value strings.Builder
	for p.pos < len(p.content) && p.peek() != '>' {
		r, size := p.peekHTMLRune()
		value.WriteRune(r)
		p.advanceRune(size)
	}
	if p.peek() == '>' {
		p.advance()
	} else {
		p.recoverOpenElementsAtEOF = true
	}

	comment := value.String()
	return &types.Node{
		Type:        types.CommentNode,
		Name:        "#comment",
		Value:       comment,
		TextContent: comment,
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}

// parseBogusComment applies HTML's recovery for an incorrectly opened
// comment. Everything after "<!" through the next '>' (or EOF) becomes the
// comment value while the node range remains tied to the original input.
func (p *HTMLParser) parseBogusComment(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	declaration := strings.HasPrefix(p.content[p.pos:], "<!")
	questionMark := strings.HasPrefix(p.content[p.pos:], "<?")
	if !declaration && !questionMark {
		return nil, fmt.Errorf("expected markup declaration at position %d", p.pos)
	}

	p.advance() // Skip '<'
	if declaration {
		p.advance() // Skip '!'
	}

	var comment strings.Builder
	for p.pos < len(p.content) && p.peek() != '>' {
		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		comment.WriteRune(r)
		p.advanceRune(size)
	}
	if p.peek() == '>' {
		p.advance()
	} else {
		p.recoverOpenElementsAtEOF = true
	}

	value := comment.String()
	return &types.Node{
		Type:        types.CommentNode,
		Name:        "#comment",
		Value:       value,
		TextContent: value,
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}

func (p *HTMLParser) parseProcessingInstruction(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if !strings.HasPrefix(p.content[p.pos:], "<?") {
		return nil, fmt.Errorf("expected processing instruction at position %d", p.pos)
	}
	p.advanceRune(2)
	if p.pos >= len(p.content) {
		p.recoverOpenElementsAtEOF = true
		return nil, nil
	}
	if !isPITargetStart(p.peek()) {
		p.pos, p.line, p.col = startPos, startLine, startCol
		return p.parseBogusComment(parent, startPos, startLine, startCol)
	}
	var target strings.Builder
	for p.pos < len(p.content) && isPITargetContinue(p.peek()) {
		character := p.peek()
		if character >= 'A' && character <= 'Z' {
			character += 'a' - 'A'
		}
		target.WriteByte(character)
		p.advance()
	}
	name := target.String()
	if asciiEqualFold(name, "xml") || asciiEqualFold(name, "xml-stylesheet") || (p.pos < len(p.content) && !isPIDelimiter(p.peek())) {
		p.pos, p.line, p.col = startPos, startLine, startCol
		return p.parseBogusComment(parent, startPos, startLine, startCol)
	}
	for p.pos < len(p.content) && isWhitespace(p.peek()) {
		p.advance()
	}
	var data strings.Builder
	for p.pos < len(p.content) {
		if p.peek() == '>' {
			p.advance()
			return processingInstructionNode(parent, name, data.String(), startPos, startLine, startCol, p.pos, p.line, p.col), nil
		}
		if p.peek() == '?' && p.pos+1 < len(p.content) && p.content[p.pos+1] == '>' {
			p.advanceRune(2)
			return processingInstructionNode(parent, name, data.String(), startPos, startLine, startCol, p.pos, p.line, p.col), nil
		}
		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		data.WriteRune(r)
		p.advanceRune(size)
	}
	p.recoverOpenElementsAtEOF = true
	return nil, nil
}

func isPITargetStart(character byte) bool {
	return character == '_' || character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z'
}

func isPITargetContinue(character byte) bool {
	return isPITargetStart(character) || character == '-' || character == '_' || character >= '0' && character <= '9'
}

func isPIDelimiter(character byte) bool {
	return isWhitespace(character) || character == '?' || character == '>'
}

func processingInstructionNode(parent *types.Node, name, data string, start, startLine, startColumn, end, endLine, endColumn int) *types.Node {
	return &types.Node{Type: types.ProcessingInstructionNode, Name: name, Value: data, TextContent: data, Parent: parent,
		StartPos: start, EndPos: end, StartLine: startLine, StartColumn: startColumn, EndLine: endLine, EndColumn: endColumn}
}

// parseComment parses an HTML comment
func (p *HTMLParser) parseComment(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if !strings.HasPrefix(p.content[p.pos:], "<!--") {
		return nil, fmt.Errorf("expected comment at position %d", p.pos)
	}

	for i := 0; i < 4; i++ {
		p.advance() // Skip "<!--"
	}

	const (
		commentStart = iota
		commentStartDash
		commentData
		commentLessThan
		commentLessThanBang
		commentLessThanBangDash
		commentLessThanBangDashDash
		commentEndDash
		commentEnd
		commentEndBang
	)

	state := commentStart
	var comment strings.Builder
	terminated := false
	for !terminated && p.pos < len(p.content) {
		c, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		reconsume := false

		switch state {
		case commentStart:
			switch c {
			case '-':
				state = commentStartDash
			case '>':
				terminated = true
			default:
				state = commentData
				reconsume = true
			}
		case commentStartDash:
			switch c {
			case '-':
				state = commentEnd
			case '>':
				terminated = true
			default:
				comment.WriteByte('-')
				state = commentData
				reconsume = true
			}
		case commentData:
			switch c {
			case '<':
				comment.WriteRune(c)
				state = commentLessThan
			case '-':
				state = commentEndDash
			default:
				comment.WriteRune(c)
			}
		case commentLessThan:
			switch c {
			case '!':
				comment.WriteRune(c)
				state = commentLessThanBang
			case '<':
				comment.WriteRune(c)
			default:
				state = commentData
				reconsume = true
			}
		case commentLessThanBang:
			if c == '-' {
				state = commentLessThanBangDash
			} else {
				state = commentData
				reconsume = true
			}
		case commentLessThanBangDash:
			if c == '-' {
				state = commentLessThanBangDashDash
			} else {
				state = commentEndDash
				reconsume = true
			}
		case commentLessThanBangDashDash:
			state = commentEnd
			reconsume = true
		case commentEndDash:
			if c == '-' {
				state = commentEnd
			} else {
				comment.WriteByte('-')
				state = commentData
				reconsume = true
			}
		case commentEnd:
			switch c {
			case '>':
				terminated = true
			case '!':
				state = commentEndBang
			case '-':
				comment.WriteByte('-')
			default:
				comment.WriteString("--")
				state = commentData
				reconsume = true
			}
		case commentEndBang:
			switch c {
			case '-':
				comment.WriteString("--!")
				state = commentEndDash
			case '>':
				terminated = true
			default:
				comment.WriteString("--!")
				state = commentData
				reconsume = true
			}
		}

		if !reconsume {
			p.advanceRune(size)
		}
	}

	value := comment.String()

	return &types.Node{
		Type:        types.CommentNode,
		Name:        "#comment",
		Value:       value,
		TextContent: value,
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}

// parseDoctype emits the browser's DocumentType shape while consuming the
// declaration through the first '>' or EOF. The detailed public/system
// identifier states affect quirks mode, which this DOM model does not expose,
// but they do not alter the DocumentType name or source extent.
func (p *HTMLParser) parseDoctype(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if !strings.HasPrefix(strings.ToUpper(p.content[p.pos:]), "<!DOCTYPE") {
		return nil, fmt.Errorf("expected DOCTYPE at position %d", p.pos)
	}

	for i := 0; i < len("<!DOCTYPE"); i++ {
		p.advance()
	}
	p.skipWhitespace()

	var name strings.Builder
	for p.pos < len(p.content) {
		c := p.peek()
		if isWhitespace(c) || c == '>' {
			break
		}
		r, size := p.peekHTMLRune()
		if isASCIIUpper(r) {
			r += 'a' - 'A'
		}
		name.WriteRune(r)
		p.advanceRune(size)
	}

	// Every DOCTYPE state emits at the first greater-than sign. EOF also emits
	// the in-progress token rather than failing the HTML parse.
	for p.pos < len(p.content) && p.peek() != '>' {
		p.advance()
	}
	terminated := p.peek() == '>'
	if terminated {
		p.advance()
	} else {
		// EOF emits the in-progress token. When tree construction ignores that
		// token inside an element, this signal closes the affected open stack.
		p.recoverOpenElementsAtEOF = true
	}
	if parent != nil && parent.Type == types.DocumentNode && p.shouldKeepDoctype(parent) {
		p.quirksMode = !doctypeUsesStandardsTableRules(p.content[startPos:p.pos], name.String())
	}

	return &types.Node{
		Type:        types.DocumentTypeNode,
		Name:        name.String(),
		Value:       "",
		TextContent: "",
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}

func doctypeUsesStandardsTableRules(raw, name string) bool {
	if name != "html" || len(raw) < len("<!doctype html>") {
		return false
	}
	declaration := trimHTMLWhitespace(raw)
	if !strings.HasSuffix(declaration, ">") {
		return false
	}
	inside := trimHTMLWhitespace(declaration[len("<!doctype") : len(declaration)-1])
	if len(inside) < len("html") || !asciiEqualFold(inside[:len("html")], "html") {
		return false
	}
	remainder := trimHTMLWhitespace(inside[len("html"):])
	if remainder == "" {
		return true
	}
	keywordEnd := strings.IndexFunc(remainder, func(r rune) bool { return r <= 0x7f && isWhitespace(byte(r)) })
	if keywordEnd < 0 {
		return false
	}
	keyword, rest := remainder[:keywordEnd], trimHTMLWhitespace(remainder[keywordEnd+1:])
	switch {
	case asciiEqualFold(keyword, "system"):
		systemID, _, ok := consumeQuotedDoctypeIdentifier(rest)
		return ok && !asciiEqualFold(systemID, "http://www.ibm.com/data/dtd/v11/ibmxhtml1-transitional.dtd")
	case asciiEqualFold(keyword, "public"):
		publicID, tail, ok := consumeQuotedDoctypeIdentifier(rest)
		if !ok {
			return false
		}
		tail = trimHTMLWhitespace(tail)
		systemIDPresent := false
		if tail != "" {
			_, tail, ok = consumeQuotedDoctypeIdentifier(tail)
			if !ok || trimHTMLWhitespace(tail) != "" {
				return false
			}
			systemIDPresent = true
		}
		publicID = strings.ToLower(publicID)
		if doctypePublicIDForcesQuirks(publicID) {
			return false
		}
		if strings.HasPrefix(publicID, "-//w3c//dtd xhtml 1.0 frameset//") || strings.HasPrefix(publicID, "-//w3c//dtd xhtml 1.0 transitional//") {
			return true
		}
		if strings.HasPrefix(publicID, "-//w3c//dtd html 4.01 frameset//") || strings.HasPrefix(publicID, "-//w3c//dtd html 4.01 transitional//") {
			return systemIDPresent
		}
		return true
	default:
		return false
	}
}

func trimHTMLWhitespace(value string) string {
	return strings.TrimFunc(value, func(r rune) bool { return r <= 0x7f && isWhitespace(byte(r)) })
}

func consumeQuotedDoctypeIdentifier(value string) (identifier, tail string, ok bool) {
	if len(value) < 2 || (value[0] != '\'' && value[0] != '"') {
		return "", value, false
	}
	if end := strings.IndexByte(value[1:], value[0]); end >= 0 {
		end++
		return value[1:end], value[end+1:], true
	}
	return "", value, false
}

func doctypePublicIDForcesQuirks(publicID string) bool {
	if publicID == "html" || publicID == "-//w3o//dtd w3 html strict 3.0//en//" || publicID == "-/w3c/dtd html 4.0 transitional/en" {
		return true
	}
	for _, prefix := range []string{
		"+//silmaril//dtd html pro v0r11 19970101//", "-//advasoft ltd//dtd html 3.0 aswedit + extensions//",
		"-//as//dtd html 3.0 aswedit + extensions//", "-//ietf//dtd html 2.", "-//ietf//dtd html 3",
		"-//ietf//dtd html level", "-//ietf//dtd html strict level", "-//microsoft//dtd internet explorer 2.0 html", "-//microsoft//dtd internet explorer 3.0 html",
		"-//metrius//dtd metrius presentational//", "-//microsoft//dtd internet explorer 4.0 html", "-//netscape comm. corp.//dtd html",
		"-//netscape comm. corp.//dtd strict html//", "-//o'reilly and associates//dtd html 2.0", "-//o'reilly and associates//dtd html extended 1.0",
		"-//o'reilly and associates//dtd html extended relaxed 1.0", "-//softquad software//dtd hotmetal pro 6.0::19990601::extensions to html 4.0//",
		"-//softquad//dtd hotmetal pro 4.0::19971010::extensions to html 4.0//", "-//spyglass//dtd html 2.0 extended//",
		"-//sq//dtd html 2.0 hotmetal + extensions//", "-//sun microsystems corp.//dtd hotjava html//",
		"-//sun microsystems corp.//dtd hotjava strict html//", "-//w3c//dtd html 3", "-//w3c//dtd html 4.0 frameset//",
		"-//w3c//dtd html 4.0 transitional//", "-//w3c//dtd html experimental 19960712//", "-//w3c//dtd html experimental 970421//", "-//w3c//dtd w3 html//",
		"-//w3o//dtd w3 html 3.0//", "-//webtechs//dtd mozilla html 2.0//", "-//webtechs//dtd mozilla html//",
	} {
		if strings.HasPrefix(publicID, prefix) {
			return true
		}
	}
	return false
}

// shouldKeepDoctype applies the observable part of HTML's insertion modes:
// only the first document-level DOCTYPE before body content is inserted.
// Leading comments do not leave the initial declaration phase.
func (p *HTMLParser) shouldKeepDoctype(parent *types.Node) bool {
	if parent == nil || parent.Type != types.DocumentNode {
		return false
	}
	for _, child := range parent.Children {
		if child.Type != types.CommentNode && child.Type != types.ProcessingInstructionNode {
			return false
		}
	}
	return true
}

func (p *HTMLParser) doctypeWasIgnoredAfterBodyContent(parent *types.Node) bool {
	if parent == nil || parent.Type != types.DocumentNode {
		return false
	}
	for _, child := range parent.Children {
		if child.Type != types.CommentNode && child.Type != types.ProcessingInstructionNode && child.Type != types.DocumentTypeNode {
			return true
		}
	}
	return false
}

type startTagState uint8

const (
	beforeAttributeNameState startTagState = iota
	attributeNameState
	afterAttributeNameState
	beforeAttributeValueState
	attributeValueDoubleQuotedState
	attributeValueSingleQuotedState
	attributeValueUnquotedState
	afterAttributeValueQuotedState
	selfClosingStartTagState
)

// parseStartTagTail implements the HTML tokenizer states from "before
// attribute name" through "self-closing start tag". The returned emitted flag
// is false when EOF discards the incomplete start-tag token.
func (p *HTMLParser) parseStartTagTail(node *types.Node) (selfClosing, emitted bool) {
	state := beforeAttributeNameState
	var attributeName strings.Builder
	var attributeValue strings.Builder

	resetAttribute := func() {
		attributeName.Reset()
		attributeValue.Reset()
	}
	commitAttribute := func() {
		name := attributeName.String()
		if name == "" {
			return
		}
		// The tokenizer removes later duplicate attributes after ASCII-case
		// normalization. Keep both the first value and its document order.
		if _, exists := node.Attributes[name]; !exists {
			node.Attributes[name] = attributeValue.String()
			node.AttributeOrder = append(node.AttributeOrder, name)
		}
	}

	for {
		if p.pos >= len(p.content) {
			return false, false
		}

		switch state {
		case beforeAttributeNameState:
			switch c := p.peek(); {
			case isWhitespace(c):
				p.advance()
			case c == '/':
				p.advance()
				state = selfClosingStartTagState
			case c == '>':
				p.advance()
				return false, true
			default:
				resetAttribute()
				// '=' has special handling in this state: it is the first
				// character of an attribute name, not the value separator.
				if c == '=' {
					attributeName.WriteByte(c)
					p.advance()
				}
				state = attributeNameState
			}

		case attributeNameState:
			switch c := p.peek(); {
			case isWhitespace(c):
				p.advance()
				state = afterAttributeNameState
			case c == '/' || c == '>':
				state = afterAttributeNameState
			case c == '=':
				p.advance()
				state = beforeAttributeValueState
			default:
				r, size := p.peekHTMLRune()
				if isASCIIUpper(r) {
					r += 'a' - 'A'
				}
				attributeName.WriteRune(r)
				p.advanceRune(size)
			}

		case afterAttributeNameState:
			switch c := p.peek(); {
			case isWhitespace(c):
				p.advance()
			case c == '/':
				commitAttribute()
				p.advance()
				state = selfClosingStartTagState
			case c == '=':
				p.advance()
				state = beforeAttributeValueState
			case c == '>':
				commitAttribute()
				p.advance()
				return false, true
			default:
				commitAttribute()
				resetAttribute()
				state = attributeNameState
			}

		case beforeAttributeValueState:
			switch c := p.peek(); {
			case isWhitespace(c):
				p.advance()
			case c == '"':
				p.advance()
				state = attributeValueDoubleQuotedState
			case c == '\'':
				p.advance()
				state = attributeValueSingleQuotedState
			case c == '>':
				commitAttribute()
				p.advance()
				return false, true
			default:
				state = attributeValueUnquotedState
			}

		case attributeValueDoubleQuotedState:
			if p.peek() == '"' {
				commitAttribute()
				p.advance()
				state = afterAttributeValueQuotedState
				continue
			}
			if value, consumed := p.consumeNumericCharacterReference(); consumed {
				attributeValue.WriteRune(value)
				continue
			}
			if value, consumed := p.consumeNamedCharacterReference(true); consumed {
				attributeValue.WriteString(value)
				continue
			}
			r, size := p.peekHTMLRune()
			attributeValue.WriteRune(r)
			p.advanceRune(size)

		case attributeValueSingleQuotedState:
			if p.peek() == '\'' {
				commitAttribute()
				p.advance()
				state = afterAttributeValueQuotedState
				continue
			}
			if value, consumed := p.consumeNumericCharacterReference(); consumed {
				attributeValue.WriteRune(value)
				continue
			}
			if value, consumed := p.consumeNamedCharacterReference(true); consumed {
				attributeValue.WriteString(value)
				continue
			}
			r, size := p.peekHTMLRune()
			attributeValue.WriteRune(r)
			p.advanceRune(size)

		case attributeValueUnquotedState:
			switch c := p.peek(); {
			case isWhitespace(c):
				commitAttribute()
				p.advance()
				state = beforeAttributeNameState
			case c == '>':
				commitAttribute()
				p.advance()
				return false, true
			default:
				if value, consumed := p.consumeNumericCharacterReference(); consumed {
					attributeValue.WriteRune(value)
					continue
				}
				if value, consumed := p.consumeNamedCharacterReference(true); consumed {
					attributeValue.WriteString(value)
					continue
				}
				// Quotes, apostrophes, '<', '=', and '`' are parse errors in
				// this state, but browsers still append them to the value.
				r, size := p.peekHTMLRune()
				attributeValue.WriteRune(r)
				p.advanceRune(size)
			}

		case afterAttributeValueQuotedState:
			switch c := p.peek(); {
			case isWhitespace(c):
				p.advance()
				state = beforeAttributeNameState
			case c == '/':
				p.advance()
				state = selfClosingStartTagState
			case c == '>':
				p.advance()
				return false, true
			default:
				// Missing whitespace: reconsume in before-attribute-name.
				state = beforeAttributeNameState
			}

		case selfClosingStartTagState:
			if p.peek() == '>' {
				p.advance()
				return true, true
			}
			// An unexpected character after '/' cancels the self-closing
			// flag and is reconsumed as the beginning of another attribute.
			state = beforeAttributeNameState
		}
	}
}

// extendPreviousTextRange mirrors parse5's location behavior for an ignored
// start-tag token at EOF. The text value is unchanged; only its original-source
// range covers the discarded bytes.
func (p *HTMLParser) extendPreviousTextRange(parent *types.Node, tagStart int) {
	if parent == nil || len(parent.Children) == 0 {
		return
	}
	previous := parent.Children[len(parent.Children)-1]
	if previous.Type != types.TextNode || previous.EndPos != tagStart {
		return
	}
	previous.EndPos = p.pos
	previous.EndLine = p.line
	previous.EndColumn = p.col
}

// parseTagName consumes the tag-name state. Callers enter this helper only
// after the tag-open state has observed an ASCII alpha character.
func (p *HTMLParser) parseTagName() string {
	var name strings.Builder
	for p.pos < len(p.content) {
		c := p.peek()
		if isWhitespace(c) || c == '/' || c == '>' {
			break
		}
		r, size := p.peekHTMLRune()
		if isASCIIUpper(r) {
			r += 'a' - 'A'
		}
		name.WriteRune(r)
		p.advanceRune(size)
	}
	return name.String()
}

// parseClosingTag tokenizes a syntactically valid end-tag opener. Attributes
// and a trailing solidus are consumed by the same states as a start tag but
// discarded. emitted is false when EOF discards the incomplete token.
func (p *HTMLParser) parseClosingTag() (name string, emitted bool, err error) {
	startPos := p.pos
	if p.pos+1 >= len(p.content) || p.content[p.pos:p.pos+2] != "</" {
		return "", false, fmt.Errorf("expected closing tag at position %d", p.pos)
	}

	p.advanceRune(2) // Skip "</".
	name = p.parseTagName()
	if name == "" {
		return "", false, fmt.Errorf("expected closing tag name at position %d", p.pos)
	}

	discardedAttributes := &types.Node{Attributes: make(map[string]string)}
	_, emitted = p.parseStartTagTail(discardedAttributes)
	if !emitted && p.pos < len(p.content) {
		return "", false, fmt.Errorf("failed to parse closing tag at position %d", startPos)
	}
	if !emitted {
		p.recoverOpenElementsAtEOF = true
	}
	return name, emitted, nil
}

// Helper functions
func (p *HTMLParser) peek() byte {
	if p.pos >= len(p.content) {
		return 0
	}
	return p.content[p.pos]
}

func (p *HTMLParser) advance() {
	if p.pos < len(p.content) {
		c := p.content[p.pos]
		if p.utf8Continuation > 0 {
			p.utf8Continuation--
			p.pos++
			return
		}
		switch c {
		case '\r':
			p.line++
			p.col = 1
			p.lastWasCR = true
		case '\n':
			if !p.lastWasCR {
				p.line++
				p.col = 1
			}
			p.lastWasCR = false
		default:
			if c >= utf8.RuneSelf {
				r, size := utf8.DecodeRuneInString(p.content[p.pos:])
				if size > 1 {
					p.utf8Continuation = size - 1
				}
				if r > 0xFFFF {
					p.col += 2
				} else {
					p.col++
				}
			} else {
				p.col++
			}
			p.lastWasCR = false
		}
		p.pos++
	}
}

// peekHTMLRune applies the input-stream preprocessing used by HTML parsers
// while retaining the number of bytes occupied in the original source.
func (p *HTMLParser) peekHTMLRune() (rune, int) {
	if p.pos >= len(p.content) {
		return 0, 0
	}
	if p.content[p.pos] == '\r' {
		if p.pos+1 < len(p.content) && p.content[p.pos+1] == '\n' {
			return '\n', 2
		}
		return '\n', 1
	}
	if p.content[p.pos] == 0 {
		return '\uFFFD', 1
	}
	return utf8.DecodeRuneInString(p.content[p.pos:])
}

// advanceRune advances the position by the given number of bytes (for a UTF-8 rune)
func (p *HTMLParser) advanceRune(size int) {
	for i := 0; i < size && p.pos < len(p.content); i++ {
		p.advance()
	}
}

func (p *HTMLParser) skipWhitespace() {
	for p.pos < len(p.content) {
		c := p.peek()
		if !isWhitespace(c) {
			break
		}
		p.advance()
	}
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func isHTMLWhitespaceString(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		switch character {
		case ' ', '\t', '\n', '\r', '\f':
		default:
			return false
		}
	}
	return true
}

func isASCIIAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isASCIIUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func (p *HTMLParser) isSelfClosingTag(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "frame", "hr", "img", "input", "keygen", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

// isRawTextElement checks if an element should have its content parsed as raw text
func (p *HTMLParser) isRawTextElement(name string) bool {
	switch name {
	case "script", "style", "textarea", "title", "xmp", "iframe", "noembed", "noframes":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) parsePlaintextContent() textStateContent {
	result := textStateContent{start: p.pos, startLine: p.line, startColumn: p.col}
	var content strings.Builder
	for p.pos < len(p.content) {
		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		content.WriteRune(r)
		p.advanceRune(size)
	}
	result.content = content.String()
	result.end = p.pos
	result.endLine, result.endColumn = p.line, p.col
	p.recoverOpenElementsAtEOF = true
	return result
}

type textStateContent struct {
	content                string
	start, end             int
	startLine, startColumn int
	endLine, endColumn     int
}

// parseRawTextContentWithPos parses RCDATA, RAWTEXT, and the simple script-data
// path while retaining the source boundaries of the emitted character tokens.
func (p *HTMLParser) parseRawTextContentWithPos(tagName string) (textStateContent, error) {
	if tagName == "script" {
		return p.parseScriptDataContent()
	}
	var content strings.Builder
	rcdata := tagName == "title" || tagName == "textarea"
	result := textStateContent{start: p.pos, startLine: p.line, startColumn: p.col}
	stripTextareaLF := tagName == "textarea"
	strippedStart := result.start
	strippedStartLine, strippedStartColumn := result.startLine, result.startColumn

	appendRune := func(value rune) {
		if content.Len() == 0 && value == '\n' && result.start > strippedStart {
			// parse5 includes both original newline tokens in the source range
			// when stripping the first still leaves a leading newline.
			result.start = strippedStart
			result.startLine, result.startColumn = strippedStartLine, strippedStartColumn
		}
		content.WriteRune(value)
	}
	appendString := func(value string) {
		if value == "" {
			return
		}
		first, size := utf8.DecodeRuneInString(value)
		appendRune(first)
		content.WriteString(value[size:])
	}
	consumeTextareaInitialLF := func(value string) (string, bool) {
		if !stripTextareaLF {
			return value, false
		}
		stripTextareaLF = false
		if strings.HasPrefix(value, "\n") {
			result.start = p.pos
			result.startLine, result.startColumn = p.line, p.col
			return strings.TrimPrefix(value, "\n"), true
		}
		return value, false
	}

	for p.pos < len(p.content) {
		if p.isAppropriateTextStateEndTag(tagName) {
			contentEnd := p.pos
			contentEndLine, contentEndColumn := p.line, p.col
			_, emitted, err := p.parseClosingTag()
			if err != nil {
				return textStateContent{}, err
			}
			if emitted {
				result.content = content.String()
				result.end = contentEnd
				result.endLine, result.endColumn = contentEndLine, contentEndColumn
				return result, nil
			}
		}
		if rcdata {
			if value, consumed := p.consumeNumericCharacterReference(); consumed {
				decoded, stripped := consumeTextareaInitialLF(string(value))
				if stripped && decoded == "" {
					continue
				}
				appendString(decoded)
				continue
			}
			if value, consumed := p.consumeNamedCharacterReference(false); consumed {
				decoded, stripped := consumeTextareaInitialLF(value)
				if stripped && decoded == "" {
					continue
				}
				appendString(decoded)
				continue
			}
		}
		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		p.advanceRune(size)
		decoded, stripped := consumeTextareaInitialLF(string(r))
		if stripped && decoded == "" {
			continue
		}
		appendString(decoded)
	}

	// EOF emits the accumulated character tokens and closes the open element.
	result.content = content.String()
	result.end = p.pos
	result.endLine, result.endColumn = p.line, p.col
	p.recoverOpenElementsAtEOF = true
	return result, nil
}

type scriptTextState uint8

const (
	scriptDataState scriptTextState = iota
	scriptEscapedState
	scriptEscapedDashState
	scriptEscapedDashDashState
	scriptDoubleEscapedState
	scriptDoubleEscapedDashState
	scriptDoubleEscapedDashDashState
)

func isScriptDoubleEscapedState(state scriptTextState) bool {
	switch state {
	case scriptDoubleEscapedState, scriptDoubleEscapedDashState, scriptDoubleEscapedDashDashState:
		return true
	default:
		return false
	}
}

// parseScriptDataContent implements the observable script-data escaped and
// double-escaped transitions. Script execution is outside this parser's scope;
// this scanner only determines character data and the appropriate closing tag.
func (p *HTMLParser) parseScriptDataContent() (textStateContent, error) {
	result := textStateContent{start: p.pos, startLine: p.line, startColumn: p.col}
	var content strings.Builder
	state := scriptDataState

	consumeThrough := func(end int) {
		for p.pos < end {
			r, size := p.peekHTMLRune()
			if size == 0 {
				break
			}
			content.WriteRune(r)
			p.advanceRune(size)
		}
	}

	for p.pos < len(p.content) {
		if !isScriptDoubleEscapedState(state) && p.isAppropriateTextStateEndTag("script") {
			contentEnd := p.pos
			contentEndLine, contentEndColumn := p.line, p.col
			_, emitted, err := p.parseClosingTag()
			if err != nil {
				return textStateContent{}, err
			}
			if emitted {
				result.content = content.String()
				result.end = contentEnd
				result.endLine, result.endColumn = contentEndLine, contentEndColumn
				return result, nil
			}
		}

		switch state {
		case scriptDataState:
			if strings.HasPrefix(p.content[p.pos:], "<!--") {
				consumeThrough(p.pos + len("<!--"))
				state = scriptEscapedDashDashState
				continue
			}
		case scriptEscapedState:
			if end, ok := p.scriptKeywordTransitionEnd("<script"); ok {
				consumeThrough(end)
				state = scriptDoubleEscapedState
				continue
			}
			if p.peek() == '-' {
				content.WriteByte('-')
				p.advance()
				state = scriptEscapedDashState
				continue
			}
		case scriptEscapedDashState:
			if p.peek() == '-' {
				content.WriteByte('-')
				p.advance()
				state = scriptEscapedDashDashState
				continue
			}
			state = scriptEscapedState
			continue
		case scriptEscapedDashDashState:
			switch p.peek() {
			case '>':
				content.WriteByte('>')
				p.advance()
				state = scriptDataState
				continue
			case '-':
				content.WriteByte('-')
				p.advance()
				continue
			default:
				state = scriptEscapedState
				continue
			}
		case scriptDoubleEscapedState:
			if end, ok := p.scriptKeywordTransitionEnd("</script"); ok {
				consumeThrough(end)
				state = scriptEscapedState
				continue
			}
			if p.peek() == '-' {
				content.WriteByte('-')
				p.advance()
				state = scriptDoubleEscapedDashState
				continue
			}
		case scriptDoubleEscapedDashState:
			if p.peek() == '-' {
				content.WriteByte('-')
				p.advance()
				state = scriptDoubleEscapedDashDashState
				continue
			}
			state = scriptDoubleEscapedState
			continue
		case scriptDoubleEscapedDashDashState:
			switch p.peek() {
			case '>':
				content.WriteByte('>')
				p.advance()
				state = scriptDataState
				continue
			case '-':
				content.WriteByte('-')
				p.advance()
				continue
			default:
				state = scriptDoubleEscapedState
				continue
			}
		}

		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		content.WriteRune(r)
		p.advanceRune(size)
	}

	result.content = content.String()
	result.end = p.pos
	result.endLine, result.endColumn = p.line, p.col
	p.recoverOpenElementsAtEOF = true
	return result, nil
}

func (p *HTMLParser) scriptKeywordTransitionEnd(keyword string) (int, bool) {
	if p.pos+len(keyword) >= len(p.content) {
		return 0, false
	}
	for index := 0; index < len(keyword); index++ {
		left := p.content[p.pos+index]
		right := keyword[index]
		if left >= 'A' && left <= 'Z' {
			left += 'a' - 'A'
		}
		if left != right {
			return 0, false
		}
	}
	delimiter := p.content[p.pos+len(keyword)]
	if !isWhitespace(delimiter) && delimiter != '/' && delimiter != '>' {
		return 0, false
	}
	return p.pos + len(keyword) + 1, true
}

func (p *HTMLParser) isAppropriateTextStateEndTag(tagName string) bool {
	prefixLength := 2 + len(tagName)
	if p.pos+prefixLength >= len(p.content) || p.content[p.pos:p.pos+2] != "</" {
		return false
	}
	for index := 0; index < len(tagName); index++ {
		character := p.content[p.pos+2+index]
		if character >= 'A' && character <= 'Z' {
			character += 'a' - 'A'
		}
		if character != tagName[index] {
			return false
		}
	}
	delimiter := p.content[p.pos+prefixLength]
	return isWhitespace(delimiter) || delimiter == '/' || delimiter == '>'
}
