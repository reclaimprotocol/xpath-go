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

// NewHTMLParser creates a new HTML parser
func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

// Parse parses HTML/XML content into a node tree with location information
func (p *HTMLParser) Parse(content string) (*types.Node, error) {
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
		return nil, nil // This is a closing tag, handled by parent
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
			value = p.parseAttributeValue()
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
		textContent, contentEnd := p.parseRawTextContentWithPos(node.Name)

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
	contentEndPos := contentStartPos // Default to start if no content

	for p.pos < len(p.content) {
		if p.pos >= len(p.content) {
			break
		}

		// Check for closing tag
		if p.peek() == '<' && p.pos+1 < len(p.content) && p.content[p.pos+1] == '/' {
			// Mark content end position before closing tag
			contentEndPos = p.pos
			closingTag := p.parseClosingTag()
			if strings.EqualFold(closingTag, node.Name) {
				break
			}
			// If it's not our closing tag, treat as text
			textContent += "</" + closingTag + ">"
			continue
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

	node.TextContent = textContent
	node.ContentStart = contentStartPos
	node.ContentEnd = contentEndPos
	node.EndPos = p.pos
	node.EndLine = p.line
	node.EndColumn = p.col

	return node, nil
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
	for p.pos < len(p.content)-2 {
		if p.content[p.pos:p.pos+3] == "-->" {
			p.pos += 3
			break
		}
		comment += string(p.content[p.pos])
		p.advance()
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
	if p.peek() == '>' {
		p.advance() // Skip '>'
	}

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
	for p.pos < len(p.content)-1 {
		if p.content[p.pos:p.pos+2] == "?>" {
			p.pos += 2
			break
		}
		instruction += string(p.content[p.pos])
		p.advance()
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

// parseClosingTag parses a closing tag and returns the tag name
func (p *HTMLParser) parseClosingTag() string {
	if p.content[p.pos:p.pos+2] != "</" {
		return ""
	}

	p.pos += 2 // Skip "</"
	name := p.parseName()

	// Skip to '>'
	for p.pos < len(p.content) && p.peek() != '>' {
		p.advance()
	}
	if p.peek() == '>' {
		p.advance()
	}

	return name
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

// parseAttributeValue parses an attribute value
func (p *HTMLParser) parseAttributeValue() string {
	p.skipWhitespace()

	if p.peek() == '"' || p.peek() == '\'' {
		quote := p.peek()
		p.advance() // Skip opening quote

		value := ""
		for p.pos < len(p.content) && p.peek() != quote {
			value += string(p.peek())
			p.advance()
		}
		if p.peek() == quote {
			p.advance() // Skip closing quote
		}
		return value
	}

	// Unquoted value
	value := ""
	for p.pos < len(p.content) {
		c := p.peek()
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '>' || c == '/' {
			break
		}
		value += string(c)
		p.advance()
	}
	return value
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
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		p.advance()
	}
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
func (p *HTMLParser) parseRawTextContentWithPos(tagName string) (string, int) {
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
						return content, contentEndPos
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

	return content, p.pos
}
