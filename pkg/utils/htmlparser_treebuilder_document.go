// Document insertion modes: implicit wrappers, head/body/frameset handling,
// and document-end location recovery.
package utils

import (
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func syntheticParagraph(parent *types.Node) *types.Node {
	return &types.Node{
		Type: types.ElementNode, Name: "p", Attributes: make(map[string]string),
		Children: []*types.Node{}, Parent: parent,
	}
}

type explicitDocumentMode uint8

const (
	explicitBeforeHead explicitDocumentMode = iota
	explicitInHead
	explicitAfterHead
	explicitInBody
	explicitAfterBody
)

func syntheticDocumentWrapper(name string, parent *types.Node) *types.Node {
	return &types.Node{
		Type:                types.ElementNode,
		Name:                name,
		NamespaceURI:        htmlElementNamespace,
		Attributes:          make(map[string]string),
		AttributeOrder:      []string{},
		AttributeNamespaces: make(map[string]string),
		AttributeLocalNames: make(map[string]string),
		AttributePrefixes:   make(map[string]string),
		Children:            []*types.Node{},
		Parent:              parent,
	}
}

func appendDocumentChild(parent, child *types.Node) {
	appendChildIncremental(parent, child)
}

func (p *HTMLParser) ensureDocumentHead(html *types.Node) *types.Node {
	if p.documentHead == nil {
		p.documentHead = syntheticDocumentWrapper("head", html)
		appendDocumentChild(html, p.documentHead)
	}
	return p.documentHead
}

func (p *HTMLParser) ensureDocumentBody(html *types.Node) *types.Node {
	if p.documentHead == nil {
		p.ensureDocumentHead(html)
	}
	if p.documentBody == nil {
		p.documentBody = syntheticDocumentWrapper("body", html)
		appendDocumentChild(html, p.documentBody)
	}
	p.headExited = true
	p.bodyContentStarted = true
	return p.documentBody
}

// parseLeadingDocumentWhitespace consumes only data-state character tokens
// whose decoded values are HTML whitespace. It stops before the first decoded
// non-whitespace token, preserving its raw source offset for reprocessing in
// the next insertion mode.
func (p *HTMLParser) parseLeadingDocumentWhitespace(parent *types.Node) *types.Node {
	// A prior insertion-mode pass may already have split an ignored whitespace
	// prefix from the first decoded nonspace reference. Leave that reference
	// untouched until the body-mode text parser consumes it and applies the
	// saved parse5 source boundary.
	if p.documentTextStartOverride >= p.pos {
		return nil
	}
	start, line, column := p.pos, p.line, p.col
	var value strings.Builder
	p.documentTextStartOverride = -1
	for p.pos < len(p.content) {
		if p.peek() == '<' && p.startsMarkupToken() {
			break
		}
		if p.peek() == 0 {
			p.advance()
			continue
		}
		preview := *p
		if decoded, consumed := preview.consumeNumericCharacterReference(); consumed {
			if !isHTMLWhitespaceString(string(decoded)) {
				p.documentTextStartOverride = preview.pos - 1
				p.documentTextStartLine = preview.line
				p.documentTextStartColumn = preview.col - 1
				break
			}
			decoded, _ = p.consumeNumericCharacterReference()
			value.WriteRune(decoded)
			continue
		}
		preview = *p
		if decoded, consumed := preview.consumeNamedCharacterReference(false); consumed {
			if !isHTMLWhitespaceString(decoded) {
				p.documentTextStartOverride = preview.pos - 1
				p.documentTextStartLine = preview.line
				p.documentTextStartColumn = preview.col - 1
				break
			}
			decoded, _ = p.consumeNamedCharacterReference(false)
			value.WriteString(decoded)
			continue
		}
		r, size := p.peekHTMLRune()
		if size == 0 || !isHTMLWhitespaceString(string(r)) {
			break
		}
		value.WriteRune(r)
		p.advanceRune(size)
	}
	if value.Len() == 0 {
		// A non-whitespace character reference at the beginning of the run is
		// parsed normally by the in-body text tokenizer and retains its whole
		// raw source range. The semicolon boundary override is only the parse5
		// convention for a nonspace reference following an ignored whitespace
		// prefix split across insertion modes.
		p.documentTextStartOverride = -1
		return nil
	}
	text := value.String()
	end, endLine, endColumn := p.pos, p.line, p.col
	if p.documentTextStartOverride >= 0 {
		end = p.documentTextStartOverride
		endLine = p.documentTextStartLine
		endColumn = p.documentTextStartColumn
	}
	return &types.Node{
		Type: types.TextNode, Name: "#text", Value: text, TextContent: text, Parent: parent,
		StartPos: start, EndPos: end, StartLine: line, StartColumn: column,
		EndLine: endLine, EndColumn: endColumn,
	}
}

func (p *HTMLParser) consumeDocumentDoctype(parent *types.Node) error {
	start, line, column := p.pos, p.line, p.col
	_, err := p.parseDoctype(parent, start, line, column)
	return err
}

func (p *HTMLParser) sourceCoordinatesAt(offset int) (int, int) {
	preview := HTMLParser{htmlParserState: htmlParserState{inputState: inputState{content: p.content, line: 1, col: 1}}}
	for preview.pos < offset {
		_, size := preview.peekHTMLRune()
		if size == 0 || preview.pos+size > offset {
			break
		}
		preview.advanceRune(size)
	}
	return preview.line, preview.col
}

// parse5/jsdom end an implicitly closed source-backed HEAD at the end of its
// last inserted child content (for example, immediately before a title end
// tag), while the containing HTML continues through EOF. Preserve that
// observable source-location convention for an omitted or discarded </head>.
func (p *HTMLParser) finishDocumentHeadAtLastContent(head *types.Node, text string, contentStart int) {
	end := contentStart
	if len(head.Children) > 0 {
		last := head.Children[len(head.Children)-1]
		if last.Type == types.ElementNode && last.ContentEnd > 0 {
			end = last.ContentEnd
		} else if last.EndPos > 0 {
			end = last.EndPos
		}
	}
	line, column := p.sourceCoordinatesAt(end)
	p.finishElementAt(head, text, contentStart, end, line, column)
}

func (p *HTMLParser) parseHeadNoscriptDisabled(node *types.Node, contentStart int) (*types.Node, error) {
	var text strings.Builder
	acceptedChildBoundary := 0
	acceptedChildLine := 0
	acceptedChildColumn := 0
	for p.pos < len(p.content) {
		if whitespace := p.parseTemplateColumnWhitespace(node); whitespace != nil {
			appendDocumentChild(node, whitespace)
			text.WriteString(whitespace.Value)
			continue
		}
		if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
			child, err := p.parseNode(node)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(node, child)
			continue
		}
		if p.nextTokenIsIgnoredDoctype() {
			if err := p.consumeDocumentDoctype(node); err != nil {
				return nil, err
			}
			continue
		}

		name, closing, tag := p.peekTagName()
		if tag && closing && name == "noscript" {
			end := p.pos
			_, emitted, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if !emitted {
				p.finishImplicitElement(node, text.String(), contentStart)
				return node, nil
			}
			node.TextContent = text.String()
			node.ContentStart = contentStart
			node.ContentEnd = end
			node.EndPos = p.pos
			node.EndLine = p.line
			node.EndColumn = p.col
			return node, nil
		}
		if tag && closing {
			if name == "br" {
				p.finishElementAt(node, text.String(), contentStart, p.pos, p.line, p.col)
				return node, nil
			}
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if tag && !closing && name == "html" && p.currentTagTokenWillEmit() {
			if err := p.mergeDuplicateStartInto(p.documentHTML, "html"); err != nil {
				return nil, err
			}
			continue
		}
		if tag && !closing && (name == "head" || name == "noscript") && p.currentTagTokenWillEmit() {
			p.consumeIgnoredStartTag()
			continue
		}
		if tag && !closing && (name == "basefont" || name == "bgsound" || name == "link" || name == "meta" || name == "noframes" || name == "style") && p.currentTagTokenWillEmit() {
			child, err := p.parseNode(node)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(node, child)
			if child != nil && child.Type == types.ElementNode {
				text.WriteString(child.TextContent)
				acceptedChildBoundary = child.ContentEnd
				if p.isSelfClosingTag(child.Name) || child.ContentEnd >= child.EndPos {
					acceptedChildBoundary = child.StartPos
				}
				acceptedChildLine, acceptedChildColumn = p.sourceCoordinatesAt(acceptedChildBoundary)
			}
			continue
		}
		if tag && !p.currentTagTokenWillEmit() {
			child, err := p.parseNode(node)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(node, child)
			continue
		}

		// Anything else pops noscript and is reprocessed by the surrounding
		// in-head/after-head driver without consuming the triggering token.
		p.finishElementAt(node, text.String(), contentStart, p.pos, p.line, p.col)
		return node, nil
	}
	if acceptedChildBoundary > 0 {
		p.finishElementAt(node, text.String(), contentStart, acceptedChildBoundary, acceptedChildLine, acceptedChildColumn)
		return node, nil
	}
	p.finishImplicitElement(node, text.String(), contentStart)
	return node, nil
}

func (p *HTMLParser) parseExplicitDocumentHead(head *types.Node, contentStart int) (*types.Node, error) {
	p.documentHead = head
	var text strings.Builder
	for p.pos < len(p.content) {
		if whitespace := p.parseLeadingDocumentWhitespace(head); whitespace != nil {
			appendDocumentChild(head, whitespace)
			text.WriteString(whitespace.Value)
			continue
		}
		if p.pos >= len(p.content) {
			break
		}
		if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
			child, err := p.parseNode(head)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(head, child)
			continue
		}
		if p.nextTokenIsIgnoredDoctype() {
			if err := p.consumeDocumentDoctype(head); err != nil {
				return nil, err
			}
			continue
		}
		name, closing, tag := p.peekTagName()
		if tag && closing && name != "head" && name != "body" && name != "html" && name != "br" {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if tag && closing && name == "head" {
			start := p.pos
			_, emitted, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if emitted {
				head.TextContent = text.String()
				head.ContentStart = contentStart
				head.ContentEnd = start
				head.EndPos = p.pos
				head.EndLine = p.line
				head.EndColumn = p.col
			} else {
				p.finishDocumentHeadAtLastContent(head, text.String(), contentStart)
			}
			p.headExited = true
			return head, nil
		}
		if tag && !closing && name == "head" && p.currentTagTokenWillEmit() {
			p.consumeIgnoredStartTag()
			continue
		}
		if tag && !closing && (isHeadOnlyElement(name) || name == "noscript") && name != "head" && p.currentTagTokenWillEmit() {
			child, err := p.parseNode(head)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(head, child)
			if child != nil && child.Type == types.ElementNode {
				text.WriteString(child.TextContent)
			}
			continue
		}
		if tag && !p.currentTagTokenWillEmit() {
			child, err := p.parseNode(head)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(head, child)
			continue
		}
		// Any other emitted token implicitly closes HEAD and is reprocessed by
		// the after-head insertion mode.
		p.finishElementAt(head, text.String(), contentStart, p.pos, p.line, p.col)
		p.headExited = true
		return head, nil
	}
	p.finishDocumentHeadAtLastContent(head, text.String(), contentStart)
	p.headExited = true
	return head, nil
}

func (p *HTMLParser) finalizeExplicitHTMLAtEOF(html *types.Node, contentStart int) *types.Node {
	p.ensureDocumentHead(html)
	if p.documentFrameset == nil {
		p.ensureDocumentBody(html)
	}
	p.markTextDirty(p.documentHead)
	p.refreshNodeTextContent(p.documentHead)
	if p.documentBody != nil {
		p.markTextDirty(p.documentBody)
		p.refreshNodeTextContent(p.documentBody)
	}
	p.markTextDirty(html)
	p.refreshNodeTextContent(html)
	html.ContentStart = contentStart
	if p.documentHTMLBoundary.end.pos > 0 {
		// A descendant recovery path may consume the explicit </html> while
		// keeping virtual descendants open for after-after-body reprocessing.
		// Their later content changes DOM/TextContent, but not the source-backed
		// HTML element's already established token range.
		html.ContentEnd = p.documentHTMLBoundary.contentEnd
		html.EndPos = p.documentHTMLBoundary.end.pos
		html.EndLine = p.documentHTMLBoundary.end.line
		html.EndColumn = p.documentHTMLBoundary.end.column
		return html
	}
	html.ContentEnd = p.pos
	html.EndPos = p.pos
	html.EndLine = p.line
	html.EndColumn = p.col
	return html
}

func (p *HTMLParser) parseExplicitDocumentHTML(html *types.Node, contentStart int) (*types.Node, error) {
	p.documentHTML = html
	mode := explicitBeforeHead
	p.documentMode = mode

	for p.pos < len(p.content) {
		// Publish the mode before dispatching this token. Recovery may recurse
		// into an open descendant, but the wrapper driver remains the owner of
		// the explicit-document insertion mode.
		p.documentMode = mode
		// List/select recovery can consume a body end while still inside a
		// descendant frame and defer following after-body comments until that
		// frame returns. Materialize them as HTML children before routing the
		// next token so their source order is preserved.
		if len(p.deferredHTMLComments) > 0 && p.documentBody != nil {
			comments := p.deferredHTMLComments
			p.deferredHTMLComments = nil
			for _, comment := range comments {
				appendDocumentChild(html, comment)
			}
		}
		// A body/html end token can change the document insertion mode while
		// descendants remain logically open. Before routing the next token by
		// the wrapper mode, reprocess ordinary content through the deepest such
		// descendant. Comments and processing instructions retain their
		// after-body/after-after-body placement and therefore bypass this path.
		if pending := p.pendingDocumentEndTarget(); pending != nil &&
			!p.nextTokenIsComment() && !p.nextTokenIsProcessingInstruction() && !p.nextTokenIsIgnoredDoctype() {
			name, closing, tag := p.peekTagName()
			if tag && !closing && p.paragraphClosingStartTag(name) && p.currentTagTokenWillEmit() {
				pending = p.resolvePendingThroughParagraph()
				parent := pending
				if parent == nil {
					parent = p.ensureDocumentBody(html)
				}
				child, err := p.parseNode(parent)
				if err != nil {
					return nil, err
				}
				if child != nil {
					if pending != nil {
						p.appendToPendingDocumentEnd(pending, child)
					} else {
						appendDocumentChild(parent, child)
					}
				}
				continue
			}
			if tag && closing {
				if name == "body" || name == "html" {
					// The wrapper insertion mode consumes these without popping the
					// pending descendant identities.
				} else if resolved, err := p.resolvePendingThroughName(name, true); err != nil {
					return nil, err
				} else if resolved {
					continue
				} else {
					child, err := p.parseNode(pending)
					if err != nil {
						return nil, err
					}
					if child != nil {
						p.appendToPendingDocumentEnd(pending, child)
					}
					continue
				}
			} else {
				child, err := p.parseNode(pending)
				if err != nil {
					return nil, err
				}
				if child != nil {
					p.appendToPendingDocumentEnd(pending, child)
				}
				continue
			}
		}
		switch mode {
		case explicitBeforeHead:
			if whitespace := p.parseLeadingDocumentWhitespace(html); whitespace != nil {
				// Before-head whitespace character tokens are ignored.
				continue
			}
			if p.pos >= len(p.content) {
				break
			}
			if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(html, child)
				continue
			}
			if p.nextTokenIsIgnoredDoctype() {
				if err := p.consumeDocumentDoctype(html); err != nil {
					return nil, err
				}
				continue
			}
			name, closing, tag := p.peekTagName()
			if tag && closing && name != "head" && name != "body" && name != "html" && name != "br" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				continue
			}
			if tag && closing && name == "head" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				continue
			}
			if tag && !closing && name == "html" && p.currentTagTokenWillEmit() {
				if err := p.mergeDuplicateStartInto(html, "html"); err != nil {
					return nil, err
				}
				continue
			}
			if tag && !closing && name == "head" && p.currentTagTokenWillEmit() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				if child != nil {
					p.documentHead = child
					appendDocumentChild(html, child)
					mode = explicitAfterHead
				}
				continue
			}
			if tag && !closing && (isHeadOnlyElement(name) || name == "noscript") && name != "head" && p.currentTagTokenWillEmit() {
				head := p.ensureDocumentHead(html)
				mode = explicitInHead
				child, err := p.parseNode(head)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(head, child)
				continue
			}
			if tag && !p.currentTagTokenWillEmit() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(html, child)
				continue
			}
			p.ensureDocumentHead(html)
			mode = explicitAfterHead
			continue

		case explicitInHead:
			head := p.ensureDocumentHead(html)
			if whitespace := p.parseLeadingDocumentWhitespace(head); whitespace != nil {
				appendDocumentChild(head, whitespace)
				continue
			}
			if p.pos >= len(p.content) {
				break
			}
			if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
				child, err := p.parseNode(head)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(head, child)
				continue
			}
			if p.nextTokenIsIgnoredDoctype() {
				if err := p.consumeDocumentDoctype(head); err != nil {
					return nil, err
				}
				continue
			}
			name, closing, tag := p.peekTagName()
			if tag && closing && name != "head" && name != "body" && name != "html" && name != "br" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				continue
			}
			if tag && closing && name == "head" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				mode = explicitAfterHead
				p.headExited = true
				continue
			}
			if tag && !closing && name == "head" && p.currentTagTokenWillEmit() {
				p.consumeIgnoredStartTag()
				continue
			}
			if tag && !closing && (isHeadOnlyElement(name) || name == "noscript") && name != "head" && p.currentTagTokenWillEmit() {
				child, err := p.parseNode(head)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(head, child)
				continue
			}
			if tag && !p.currentTagTokenWillEmit() {
				child, err := p.parseNode(head)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(head, child)
				continue
			}
			mode = explicitAfterHead
			p.headExited = true
			continue

		case explicitAfterHead:
			if whitespace := p.parseLeadingDocumentWhitespace(html); whitespace != nil {
				appendDocumentChild(html, whitespace)
				continue
			}
			if p.pos >= len(p.content) {
				break
			}
			if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(html, child)
				continue
			}
			if p.nextTokenIsIgnoredDoctype() {
				if err := p.consumeDocumentDoctype(html); err != nil {
					return nil, err
				}
				continue
			}
			name, closing, tag := p.peekTagName()
			if tag && !closing && name == "html" && p.currentTagTokenWillEmit() {
				if err := p.mergeDuplicateStartInto(html, "html"); err != nil {
					return nil, err
				}
				continue
			}
			if tag && !closing && name == "head" && p.currentTagTokenWillEmit() {
				p.consumeIgnoredStartTag()
				continue
			}
			if tag && !closing && isHeadOnlyElement(name) && name != "head" && p.currentTagTokenWillEmit() {
				head := p.ensureDocumentHead(html)
				// The after-head insertion mode temporarily reuses the retained
				// head pointer for head-only tokens. parse5 extends a source-backed
				// HEAD through any intervening after-head tokens up to the start of
				// the token routed back into HEAD; the routed token itself remains
				// outside that source range. Synthetic heads stay locationless.
				if head.StartLine != 0 {
					head.ContentEnd = p.pos
					head.EndPos = p.pos
					head.EndLine = p.line
					head.EndColumn = p.col
				}
				child, err := p.parseNode(head)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(head, child)
				continue
			}
			if tag && !closing && name == "body" && p.currentTagTokenWillEmit() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				if child != nil {
					// A source BODY that reaches EOF without an emitted end tag has
					// only start-tag source location in parse5/DOM. Its descendants
					// still close at EOF. An explicit </body> ending at EOF retains
					// the full source-backed range.
					if p.pos == len(p.content) && child.StartLine != 0 && html.StartLine == 0 {
						explicitEnd := p.hasEmittedEndTagBetween("body", child.ContentStart, p.pos) ||
							p.hasEmittedEndTagBetween("html", child.ContentStart, p.pos)
						if !explicitEnd {
							child.ContentEnd = child.ContentStart
							child.EndPos = child.ContentStart
							contentStart := p.elementContentStart[child]
							child.EndLine = contentStart.line
							child.EndColumn = contentStart.column
						}
					}
					p.documentBody = child
					appendDocumentChild(html, child)
					p.appendDeferredListComments(html, child)
					mode = explicitAfterBody
				}
				continue
			}
			if tag && !closing && name == "frameset" && p.currentTagTokenWillEmit() && p.documentBody == nil {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				if child != nil {
					p.documentFrameset = child
					appendDocumentChild(html, child)
					mode = explicitAfterBody
				}
				continue
			}
			if tag && !p.currentTagTokenWillEmit() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(html, child)
				continue
			}
			p.ensureDocumentBody(html)
			mode = explicitInBody
			continue

		case explicitInBody:
			body := p.ensureDocumentBody(html)
			name, closing, tag := p.peekTagName()
			if tag && closing && name == "head" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				continue
			}
			if tag && closing && name == "html" {
				mode = explicitAfterBody
				continue
			}
			if tag && closing && name == "body" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				mode = explicitAfterBody
				continue
			}
			if tag && !closing && name == "html" && p.currentTagTokenWillEmit() {
				if err := p.mergeDuplicateStartInto(html, "html"); err != nil {
					return nil, err
				}
				continue
			}
			if tag && !closing && name == "body" && p.currentTagTokenWillEmit() {
				if err := p.mergeDuplicateStartInto(body, "body"); err != nil {
					return nil, err
				}
				continue
			}
			if tag && !closing && name == "head" && p.currentTagTokenWillEmit() {
				p.consumeIgnoredStartTag()
				continue
			}
			if tag && !closing && (name == "frameset" || name == "frame") && p.currentTagTokenWillEmit() {
				p.consumeIgnoredStartTag()
				continue
			}
			if p.nextTokenIsIgnoredDoctype() {
				if err := p.consumeDocumentDoctype(body); err != nil {
					return nil, err
				}
				continue
			}
			if err := p.parseDocumentBodyToken(body); err != nil {
				return nil, err
			}

		case explicitAfterBody:
			if p.pos >= len(p.content) {
				break
			}
			if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(html, child)
				continue
			}
			if p.nextTokenIsIgnoredDoctype() {
				if err := p.consumeDocumentDoctype(html); err != nil {
					return nil, err
				}
				continue
			}
			name, closing, tag := p.peekTagName()
			if tag && closing && name == "html" {
				start := p.pos
				_, emitted, err := p.parseClosingTag()
				if err != nil {
					return nil, err
				}
				if emitted {
					p.ensureDocumentHead(html)
					if p.documentFrameset == nil {
						p.ensureDocumentBody(html)
					}
					p.markTextDirty(p.documentHead)
					p.refreshNodeTextContent(p.documentHead)
					if p.documentBody != nil {
						p.markTextDirty(p.documentBody)
						p.refreshNodeTextContent(p.documentBody)
					}
					p.markTextDirty(html)
					p.refreshNodeTextContent(html)
					html.ContentStart = contentStart
					html.ContentEnd = start
					html.EndPos = p.pos
					html.EndLine = p.line
					html.EndColumn = p.col
					p.documentHTMLBoundary.set(start, p.pos, p.line, p.col)
					p.documentEndRecovery = true
					p.explicitAfterAfterBody = true
					return html, nil
				}
				break
			}
			if tag && !closing && name == "html" && p.currentTagTokenWillEmit() {
				if err := p.mergeDuplicateStartInto(html, "html"); err != nil {
					return nil, err
				}
				continue
			}
			if tag && !closing && name == "body" && p.currentTagTokenWillEmit() {
				if err := p.mergeDuplicateStartInto(p.ensureDocumentBody(html), "body"); err != nil {
					return nil, err
				}
				continue
			}
			if p.documentFrameset != nil {
				if tag && !closing && name == "noframes" && p.currentTagTokenWillEmit() {
					head := p.ensureDocumentHead(html)
					child, err := p.parseNode(head)
					if err != nil {
						return nil, err
					}
					appendDocumentChild(head, child)
					continue
				}
				// The after-frameset insertion mode ignores ordinary tokens.
				// Whitespace is retained at the HTML level; comments/PIs and the
				// closing-html cases were already handled above.
				if whitespace := p.parseLeadingDocumentWhitespace(html); whitespace != nil {
					appendDocumentChild(html, whitespace)
					continue
				}
				if tag {
					if closing {
						if _, _, err := p.parseClosingTag(); err != nil {
							return nil, err
						}
					} else {
						p.consumeIgnoredStartTag()
					}
					continue
				}
				if _, err := p.parseTextNode(html, p.pos, p.line, p.col); err != nil {
					return nil, err
				}
				continue
			}
			// Whitespace and all ordinary tokens are processed using in-body
			// rules; comments/PIs above remain children of HTML.
			mode = explicitInBody
			continue
		}
	}

	return p.finalizeExplicitHTMLAtEOF(html, contentStart), nil
}

// parseDocumentBodyToken processes one token at the top level of a synthetic
// BODY. Source-backed BODY elements normally run the same rules from the
// recursive parseElement loop; a synthetic wrapper has no start tag/frame, so
// the document insertion-mode driver must preserve the identity-aware token
// handling that the former source-root loop provided.
func (p *HTMLParser) parseDocumentBodyToken(body *types.Node) error {
	if len(p.adoptionSplit) > 0 {
		p.appendAllDeferredAdoption(body)
	}
	if name, closing, ok := p.peekTagName(); ok && closing && p.adoptionIgnoredEnds[name] > 0 && p.activeFormattingEntryByName(name) == nil {
		start := p.pos
		if _, _, err := p.parseClosingTag(); err != nil {
			return err
		}
		p.extendPreviousTextRange(body, start)
		p.adoptionIgnoredEnds[name]--
		return nil
	}
	if consumed, err := p.consumeForeignIgnoredEnd(body); consumed || err != nil {
		return err
	}
	if name, closing, ok := p.peekTagName(); ok && closing && p.foreignIgnoredHTMLEnds[name] > 0 {
		start := p.pos
		if _, emitted, err := p.parseClosingTag(); err != nil {
			return err
		} else if !emitted {
			p.extendPreviousTextRange(body, start)
		}
		p.foreignIgnoredHTMLEnds[name]--
		return nil
	}
	if handled, err := p.handleForeignDocumentStart(); handled || err != nil {
		return err
	}
	if handled, err := p.handleFormToken(body, false); handled || err != nil {
		return err
	}
	if name, closing, ok := p.peekTagName(); ok && !closing && isTableStructureStart(name) && p.currentTagTokenWillEmit() {
		p.consumeIgnoredStartTag()
		return nil
	}
	if name, closing, ok := p.peekTagName(); ok && closing && isFormattingElement(name) {
		if isCoreAdoptionSubject(name) {
			if entry := p.activeFormattingEntryByName(name); entry == nil || entry.current == nil {
				start := p.pos
				if _, _, err := p.parseClosingTag(); err != nil {
					return err
				}
				if entry != nil {
					p.removeActiveFormattingEntry(entry)
				}
				p.extendPreviousTextRange(body, start)
				return nil
			}
		}
		if entry := p.activeFormattingEntryByName(name); entry != nil {
			if !p.openFormatting[entry.current] {
				start := p.pos
				if _, _, err := p.parseClosingTag(); err != nil {
					return err
				}
				p.extendPreviousTextRange(body, start)
				p.removeActiveFormattingEntry(entry)
				return nil
			}
			tokenStart := p.pos
			if _, _, err := p.parseClosingTag(); err != nil {
				return err
			}
			p.refreshFormattingChain(entry.current)
			if entry.current.StartLine != 0 {
				entry.current.ContentEnd = tokenStart
				entry.current.EndPos = p.pos
				entry.current.EndLine = p.line
				entry.current.EndColumn = p.col
			}
			p.removeActiveFormattingEntry(entry)
			return nil
		}
	}
	// The in-body "any other end tag" rule ignores a complete end tag when
	// no matching ordinary element is in scope. All identity-aware formatting,
	// form, list, select, and foreign consumers above get first refusal.
	if name, closing, ok := p.peekTagName(); ok && closing && name != "p" && name != "br" &&
		!p.isBogusEndTagOpen() && !p.isMissingEndTagName() && p.currentTagTokenWillEmit() {
		if _, _, err := p.parseClosingTag(); err != nil {
			return err
		}
		return nil
	}
	if action, err := p.handleSpecialFormattingStart(body, "", 0); err != nil {
		return err
	} else if action == adoptionContinue {
		return nil
	}
	processingInstruction := p.nextTokenIsProcessingInstruction()
	insertionParent := body
	if !processingInstruction {
		insertionParent = p.formattingInsertionParent(body, p.nextTokenReconstructsFormatting())
	}
	child, err := p.parseNode(insertionParent)
	if err != nil {
		return err
	}
	if child == nil {
		return nil
	}
	if child.Type == types.TextNode && p.documentTextStartOverride >= 0 {
		child.StartPos = p.documentTextStartOverride
		child.StartLine = p.documentTextStartLine
		child.StartColumn = p.documentTextStartColumn
		p.documentTextStartOverride = -1
	}
	if insertionParent != body {
		p.appendFormattingChild(body, insertionParent, child)
		if !p.isSelfClosingTag(child.Name) {
			return nil
		}
		return nil
	}
	appendDocumentChild(body, child)
	p.appendDeferredTableFoster(body, child)
	p.appendDeferredListComments(body, child)
	p.appendDeferredAdoption(body, child)
	p.appendDeferredFormSiblings(body, child)
	return nil
}

func isTableStructureStart(name string) bool {
	switch name {
	case "caption", "col", "colgroup", "tbody", "td", "tfoot", "th", "thead", "tr":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) hasEmittedEndTagBetween(name string, start, end int) bool {
	if start < 0 {
		start = 0
	}
	if end > len(p.content) {
		end = len(p.content)
	}
	for offset := start; offset+2 < end; {
		relative := strings.Index(p.content[offset:end], "</")
		if relative < 0 {
			return false
		}
		offset += relative
		preview := *p
		preview.pos = offset
		preview.tagPreview = tagPreview{position: -1}
		candidate, closing, ok := preview.peekTagName()
		if ok && closing && candidate == name && preview.currentTagTokenWillEmit() {
			return true
		}
		offset += 2
	}
	return false
}

// parseDocumentFramesetElement implements the bounded in-frameset insertion
// mode needed by whole-document parsing. Ordinary character/start/end tokens
// are ignored; whitespace, comments/PIs, nested framesets, frame, and noframes
// retain their browser insertion parents.
func (p *HTMLParser) parseDocumentFramesetElement(frameset *types.Node, contentStart int) (*types.Node, error) {
	var text strings.Builder
	for p.pos < len(p.content) {
		if whitespace := p.parseLeadingDocumentWhitespace(frameset); whitespace != nil {
			appendDocumentChild(frameset, whitespace)
			text.WriteString(whitespace.Value)
			continue
		}
		if p.pos >= len(p.content) {
			break
		}
		if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
			child, err := p.parseNode(frameset)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(frameset, child)
			continue
		}
		if p.nextTokenIsIgnoredDoctype() {
			if err := p.consumeDocumentDoctype(frameset); err != nil {
				return nil, err
			}
			continue
		}
		name, closing, tag := p.peekTagName()
		if tag && closing && name == "frameset" {
			end := p.pos
			_, emitted, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if emitted {
				p.finishElementAt(frameset, text.String(), contentStart, end, p.line, p.col)
				frameset.EndPos = p.pos
				frameset.EndLine = p.line
				frameset.EndColumn = p.col
				return frameset, nil
			}
			break
		}
		if tag && !closing && (name == "frameset" || name == "frame" || name == "noframes") && p.currentTagTokenWillEmit() {
			child, err := p.parseNode(frameset)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(frameset, child)
			if child != nil && child.Type == types.ElementNode {
				text.WriteString(child.TextContent)
			}
			continue
		}
		if tag {
			if closing {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
			} else {
				p.consumeIgnoredStartTag()
			}
			continue
		}
		if _, err := p.parseTextNode(frameset, p.pos, p.line, p.col); err != nil {
			return nil, err
		}
	}
	p.finishImplicitElement(frameset, text.String(), contentStart)
	return frameset, nil
}

func isHeadOnlyElement(name string) bool {
	switch name {
	case "base", "basefont", "bgsound", "head", "link", "meta", "noframes", "script", "style", "template", "title":
		return true
	default:
		return false
	}
}
