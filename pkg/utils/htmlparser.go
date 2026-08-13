package utils

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// HTMLParser parses HTML/XML with location tracking
type HTMLParser struct {
	content string
	pos     int
	line    int
	col     int
}

// NewHTMLParser creates a parser that reports malformed HTML/XML.
func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

// Parse parses HTML/XML content into a node tree with location information
func (p *HTMLParser) Parse(content string) (*types.Node, error) {
	if isBinaryInput(content) {
		return nil, fmt.Errorf("binary input is not supported")
	}

	p.content = content
	p.pos = 0
	p.line = 1
	p.col = 1

	// Create document root
	root := &types.Node{
		Type:        types.DocumentNode,
		Name:        "#document",
		Children:    []*types.Node{},
		StartPos:    0,
		EndPos:      len(content),
		StartLine:   1,
		StartColumn: 1,
	}

	// Parse child nodes
	for p.pos < len(content) {
		p.skipWhitespace()
		if p.pos >= len(content) {
			break
		}

		node, err := p.parseNode(root)
		if err != nil {
			return nil, err
		}
		if node != nil {
			root.Children = append(root.Children, node)
		}
	}

	// Calculate final position
	root.EndLine = p.line
	root.EndColumn = p.col
	root.SourceLength = len(content)

	return root, nil
}

func isBinaryInput(content string) bool {
	if strings.HasPrefix(content, "\x1f\x8b") || !utf8.ValidString(content) {
		return true
	}

	for i := 0; i < len(content); i++ {
		c := content[i]
		if (c < ' ' && c != '\t' && c != '\n' && c != '\r' && c != '\f') || c == 0x7f {
			return true
		}
	}
	return false
}

// parseNode parses a single node (element, text, comment, etc.)
func (p *HTMLParser) parseNode(parent *types.Node) (*types.Node, error) {
	startPos := p.pos
	startLine := p.line
	startCol := p.col

	if p.peek() == '<' {
		return p.parseElement(parent, startPos, startLine, startCol)
	}

	// Parse text node
	return p.parseTextNode(parent, startPos, startLine, startCol)
}

// parseElement parses an HTML/XML element
func (p *HTMLParser) parseElement(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if p.peek() != '<' {
		return nil, fmt.Errorf("expected '<' at position %d", p.pos)
	}

	// Check for special elements
	if p.pos+1 < len(p.content) {
		if p.content[p.pos+1] == '!' {
			// Check if it's a DOCTYPE declaration
			if strings.HasPrefix(strings.ToUpper(p.content[p.pos:]), "<!DOCTYPE") {
				return p.parseDoctype(parent, startPos, startLine, startCol)
			}
			// Otherwise it's a comment
			return p.parseComment(parent, startPos, startLine, startCol)
		}
		if p.content[p.pos+1] == '?' {
			return p.parseProcessingInstruction(parent, startPos, startLine, startCol)
		}
	}

	p.advance() // Skip '<'

	// Check for closing tag
	if p.peek() == '/' {
		return nil, fmt.Errorf("unexpected closing tag at position %d", startPos)
	}

	// Parse tag name
	tagName := p.parseName()
	if tagName == "" {
		return nil, fmt.Errorf("expected tag name at position %d", p.pos)
	}

	node := &types.Node{
		Type:           types.ElementNode,
		Name:           strings.ToLower(tagName),
		Attributes:     make(map[string]string),
		AttributeOrder: []string{},
		Children:       []*types.Node{},
		Parent:         parent,
		StartPos:       startPos,
		StartLine:      startLine,
		StartColumn:    startCol,
	}

	// Parse attributes
	for p.pos < len(p.content) && p.peek() != '>' && p.peek() != '/' {
		// Browsers tolerate a missing space after a quoted attribute value, for
		// example id="target"class="primary". Parsing directly from the original
		// input keeps every node's byte positions unchanged.
		p.skipWhitespace()
		if p.peek() == '>' || p.peek() == '/' {
			break
		}

		name := p.parseName()
		if name == "" {
			break
		}

		value := ""
		p.skipWhitespace()
		if p.peek() == '=' {
			p.advance() // Skip '='
			p.skipWhitespace()
			var err error
			value, err = p.parseAttributeValue()
			if err != nil {
				return nil, err
			}
		}

		lowerName := strings.ToLower(name)
		node.Attributes[lowerName] = value
		node.AttributeOrder = append(node.AttributeOrder, lowerName)
	}

	// Check for self-closing tag
	selfClosing := false
	if p.peek() == '/' {
		selfClosing = true
		p.advance()
	}

	if p.peek() != '>' {
		return nil, fmt.Errorf("expected '>' at position %d", p.pos)
	}
	p.advance() // Skip '>'

	if selfClosing || p.isSelfClosingTag(node.Name) {
		node.EndPos = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
		// For self-closing tags, content start and end are the same (no inner content)
		node.ContentStart = p.pos
		node.ContentEnd = p.pos
		return node, nil
	}

	// Mark the start of inner content (after opening tag)
	contentStartPos := p.pos

	// Handle raw text elements like script, style, textarea, title
	if p.isRawTextElement(node.Name) {
		textContent, contentEnd, err := p.parseRawTextContentWithPos(node.Name)
		if err != nil {
			return nil, err
		}

		// Create a single text node for the raw content
		if textContent != "" {
			textNode := &types.Node{
				Type:        types.TextNode,
				Name:        "#text",
				Value:       textContent,
				TextContent: textContent,
				Parent:      node,
				StartPos:    contentStartPos,
				EndPos:      contentEnd,
				StartLine:   p.line,
				StartColumn: p.col,
				EndLine:     p.line,
				EndColumn:   p.col,
			}
			node.Children = append(node.Children, textNode)
		}

		node.TextContent = textContent
		node.ContentStart = contentStartPos
		node.ContentEnd = contentEnd
		node.EndPos = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
		return node, nil
	}

	// Parse child nodes normally for other elements
	textContent := ""

	for p.pos < len(p.content) {
		if p.pos >= len(p.content) {
			break
		}

		// HTML's tree-construction rules implicitly close the current table row
		// when another row/section begins or the containing table/section ends.
		// Leave the triggering tag unconsumed so the parent can process it.
		if p.shouldImplicitlyCloseTableRow(node.Name) {
			node.TextContent = textContent
			node.ContentStart = contentStartPos
			node.ContentEnd = p.pos
			node.EndPos = p.pos
			node.EndLine = p.line
			node.EndColumn = p.col
			return node, nil
		}

		// Check for closing tag
		if p.peek() == '<' && p.pos+1 < len(p.content) && p.content[p.pos+1] == '/' {
			// Mark content end position before closing tag
			contentEndPos := p.pos
			closingTag, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if strings.EqualFold(closingTag, node.Name) {
				node.TextContent = textContent
				node.ContentStart = contentStartPos
				node.ContentEnd = contentEndPos
				node.EndPos = p.pos
				node.EndLine = p.line
				node.EndColumn = p.col
				return node, nil
			}
			return nil, fmt.Errorf("expected closing tag </%s>, got </%s> at position %d", node.Name, closingTag, contentEndPos)
		}

		child, err := p.parseNode(node)
		if err != nil {
			return nil, err
		}
		if child != nil {
			switch child.Type {
			case types.TextNode:
				textContent += child.Value
			case types.ElementNode:
				// Recursively collect text content from element children
				textContent += child.TextContent
			}
			node.Children = append(node.Children, child)
		}
	}

	return nil, fmt.Errorf("unterminated element <%s> at position %d", node.Name, startPos)
}

// shouldImplicitlyCloseTableRow reports whether the token at the current
// position closes an open tr according to HTML's "in row" insertion mode.
func (p *HTMLParser) shouldImplicitlyCloseTableRow(currentName string) bool {
	if currentName != "tr" {
		return false
	}

	tagName, closing, ok := p.peekTagName()
	if !ok {
		return false
	}

	if closing {
		switch tagName {
		case "table", "tbody", "tfoot", "thead":
			return true
		default:
			return false
		}
	}

	switch tagName {
	case "caption", "col", "colgroup", "tbody", "tfoot", "thead", "tr":
		return true
	default:
		return false
	}
}

// peekTagName inspects an opening or closing tag without advancing the parser.
func (p *HTMLParser) peekTagName() (name string, closing bool, ok bool) {
	if p.pos >= len(p.content) || p.content[p.pos] != '<' {
		return "", false, false
	}

	position := p.pos + 1
	if position < len(p.content) && p.content[position] == '/' {
		closing = true
		position++
	}
	start := position
	for position < len(p.content) && isNameChar(p.content[position]) {
		position++
	}
	if position == start {
		return "", false, false
	}

	return strings.ToLower(p.content[start:position]), closing, true
}

// parseTextNode parses a text node
func (p *HTMLParser) parseTextNode(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	text := ""

	for p.pos < len(p.content) && p.peek() != '<' {
		// Handle UTF-8 correctly by reading the full character
		r, size := p.peekRune()
		if r == 0 {
			break
		}
		text += string(r)
		p.advanceRune(size)
	}

	if text == "" {
		return nil, nil
	}

	// Preserve all text nodes including whitespace-only text for XPath compatibility
	// Don't skip whitespace-only text nodes as they are significant for XPath expressions

	return &types.Node{
		Type:        types.TextNode,
		Name:        "#text",
		Value:       text, // Preserve original text with whitespace
		TextContent: text, // Preserve original text with whitespace
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}

// parseComment parses an HTML comment
func (p *HTMLParser) parseComment(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if !strings.HasPrefix(p.content[p.pos:], "<!--") {
		return nil, fmt.Errorf("expected comment at position %d", p.pos)
	}

	p.pos += 4 // Skip "<!--"

	comment := ""
	terminated := false
	for p.pos < len(p.content)-2 {
		if p.content[p.pos:p.pos+3] == "-->" {
			p.pos += 3
			terminated = true
			break
		}
		comment += string(p.content[p.pos])
		p.advance()
	}
	if !terminated {
		return nil, fmt.Errorf("unterminated comment at position %d", startPos)
	}

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

// parseDoctype parses a DOCTYPE declaration
func (p *HTMLParser) parseDoctype(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if !strings.HasPrefix(strings.ToUpper(p.content[p.pos:]), "<!DOCTYPE") {
		return nil, fmt.Errorf("expected DOCTYPE at position %d", p.pos)
	}

	// Find the end of the DOCTYPE declaration
	startDoctype := p.pos
	for p.pos < len(p.content) && p.peek() != '>' {
		p.advance()
	}
	if p.peek() != '>' {
		return nil, fmt.Errorf("unterminated DOCTYPE at position %d", startPos)
	}
	p.advance() // Skip '>'

	doctypeText := p.content[startDoctype:p.pos]

	return &types.Node{
		Type:        types.DocumentTypeNode,
		Name:        "#doctype",
		Value:       doctypeText,
		TextContent: doctypeText,
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}

// parseProcessingInstruction parses a processing instruction
func (p *HTMLParser) parseProcessingInstruction(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if p.content[p.pos:p.pos+2] != "<?" {
		return nil, fmt.Errorf("expected processing instruction at position %d", p.pos)
	}

	p.pos += 2 // Skip "<?"

	instruction := ""
	terminated := false
	for p.pos < len(p.content)-1 {
		if p.content[p.pos:p.pos+2] == "?>" {
			p.pos += 2
			terminated = true
			break
		}
		instruction += string(p.content[p.pos])
		p.advance()
	}
	if !terminated {
		return nil, fmt.Errorf("unterminated processing instruction at position %d", startPos)
	}

	return &types.Node{
		Type:        types.ProcessingInstructionNode,
		Name:        "#processing-instruction",
		Value:       instruction,
		TextContent: instruction,
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}

// parseClosingTag parses a closing tag and returns the tag name.
func (p *HTMLParser) parseClosingTag() (string, error) {
	startPos := p.pos
	if p.pos+1 >= len(p.content) || p.content[p.pos:p.pos+2] != "</" {
		return "", fmt.Errorf("expected closing tag at position %d", p.pos)
	}

	p.pos += 2 // Skip "</"
	name := p.parseName()
	if name == "" {
		return "", fmt.Errorf("expected closing tag name at position %d", p.pos)
	}
	p.skipWhitespace()
	if p.peek() != '>' {
		return "", fmt.Errorf("expected '>' for closing tag at position %d", startPos)
	}
	p.advance()

	return name, nil
}

// parseName parses an element or attribute name
func (p *HTMLParser) parseName() string {
	name := ""
	for p.pos < len(p.content) {
		c := p.peek()
		if !isNameChar(c) {
			break
		}
		name += string(c)
		p.advance()
	}
	return name
}

// parseAttributeValue parses an attribute value.
func (p *HTMLParser) parseAttributeValue() (string, error) {
	p.skipWhitespace()
	if p.pos >= len(p.content) || p.peek() == '>' || p.peek() == '/' {
		return "", fmt.Errorf("expected attribute value at position %d", p.pos)
	}

	if p.peek() == '"' || p.peek() == '\'' {
		quote := p.peek()
		p.advance() // Skip opening quote

		value := ""
		for p.pos < len(p.content) && p.peek() != quote {
			value += string(p.peek())
			p.advance()
		}
		if p.peek() != quote {
			return "", fmt.Errorf("unterminated quoted attribute value at position %d", p.pos)
		}
		p.advance() // Skip closing quote
		return value, nil
	}

	// Unquoted value
	value := ""
	for p.pos < len(p.content) {
		c := p.peek()
		if isWhitespace(c) || c == '>' || c == '/' {
			break
		}
		if c == '=' || c == '<' || c == '"' || c == '\'' || c == '`' {
			return "", fmt.Errorf("invalid character %q in unquoted attribute value at position %d", c, p.pos)
		}
		value += string(c)
		p.advance()
	}
	if value == "" {
		return "", fmt.Errorf("expected attribute value at position %d", p.pos)
	}
	return value, nil
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
		if p.content[p.pos] == '\n' {
			p.line++
			p.col = 1
		} else {
			p.col++
		}
		p.pos++
	}
}

// peekRune returns the UTF-8 rune at the current position and its byte size
func (p *HTMLParser) peekRune() (rune, int) {
	if p.pos >= len(p.content) {
		return 0, 0
	}
	return utf8.DecodeRuneInString(p.content[p.pos:])
}

// advanceRune advances the position by the given number of bytes (for a UTF-8 rune)
func (p *HTMLParser) advanceRune(size int) {
	for i := 0; i < size && p.pos < len(p.content); i++ {
		if p.content[p.pos] == '\n' {
			p.line++
			p.col = 1
		} else {
			p.col++
		}
		p.pos++
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

func isNameChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '_' || c == ':'
}

func (p *HTMLParser) isSelfClosingTag(name string) bool {
	selfClosingTags := map[string]bool{
		"area": true, "base": true, "br": true, "col": true,
		"embed": true, "hr": true, "img": true, "input": true,
		"link": true, "meta": true, "param": true, "source": true,
		"track": true, "wbr": true,
	}
	return selfClosingTags[name]
}

// isRawTextElement checks if an element should have its content parsed as raw text
func (p *HTMLParser) isRawTextElement(name string) bool {
	rawTextElements := map[string]bool{
		"script":   true,
		"style":    true,
		"textarea": true,
		"title":    true,
	}
	return rawTextElements[name]
}

// parseRawTextContentWithPos parses the raw text content and returns both content and end position
func (p *HTMLParser) parseRawTextContentWithPos(tagName string) (string, int, error) {
	content := ""
	closingTag := "</" + strings.ToLower(tagName)

	for p.pos < len(p.content) {
		// Look for the closing tag
		if p.pos+len(closingTag) <= len(p.content) {
			// Check if we found the closing tag (case-insensitive)
			potentialClosing := strings.ToLower(p.content[p.pos : p.pos+len(closingTag)])
			if potentialClosing == closingTag {
				// Check that the next character is either '>' or whitespace
				nextPos := p.pos + len(closingTag)
				if nextPos < len(p.content) {
					nextChar := p.content[nextPos]
					if nextChar == '>' || nextChar == ' ' || nextChar == '\t' || nextChar == '\n' || nextChar == '\r' {
						// Found the closing tag, capture the content end position
						contentEndPos := p.pos
						// Skip to the end of the closing tag
						for p.pos < len(p.content) && p.peek() != '>' {
							p.advance()
						}
						if p.peek() == '>' {
							p.advance() // Skip '>'
						}
						return content, contentEndPos, nil
					}
				}
			}
		}

		// Add character to content and advance
		r, size := p.peekRune()
		if r == 0 {
			break
		}
		content += string(r)
		p.advanceRune(size)
	}

	return "", p.pos, fmt.Errorf("unterminated raw text element <%s>", tagName)
}
