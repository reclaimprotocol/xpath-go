package utils

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// NewHTMLParser creates a parser that reports malformed HTML/XML.
func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

// SetScriptingEnabled configures whole-document noscript parsing. When true,
// noscript contents use RAWTEXT; when false, markup is parsed using the
// scripting-disabled body and in-head-noscript rules.
func (p *HTMLParser) SetScriptingEnabled(enabled bool) {
	p.scriptingEnabled = enabled
}

// Parse parses HTML/XML content into a node tree with location information
func (p *HTMLParser) Parse(content string) (*types.Node, error) {
	if isBinaryInput(content) {
		return nil, fmt.Errorf("binary input is not supported")
	}

	p.htmlParserState = newHTMLParserState(content)

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
	p.documentRoot = root

	// Parse child nodes
	for p.pos < len(content) {
		if len(p.adoptionSplit) > 0 {
			p.appendAllDeferredAdoption(root)
		}
		pendingTarget := p.pendingDocumentEndTarget()
		pendingName, pendingClosing, pendingTag := p.peekTagName()
		blockAfterDocumentEnd := pendingTarget != nil && pendingTag && !pendingClosing && p.paragraphClosingStartTag(pendingName) && p.currentTagTokenWillEmit()
		if blockAfterDocumentEnd {
			pendingTarget = p.resolvePendingThroughParagraph()
		} else if pendingTarget != nil && pendingTag && !pendingClosing && pendingName == "table" && p.quirksMode && p.currentTagTokenWillEmit() {
			for _, pending := range p.nodesClosedAtDocumentEnd {
				if pending.Name == "p" {
					pendingTarget = pending
					break
				}
			}
		}
		if pendingTarget != nil {
			if name, closing, ok := p.peekTagName(); ok && closing {
				if resolved, err := p.resolvePendingThroughName(name, true); err != nil {
					return nil, err
				} else if resolved {
					// Resolving a still-open descendant after </html> re-enters
					// in-body processing. Subsequent comments therefore belong to
					// BODY, not the after-after-body Document position.
					if p.explicitAfterAfterBody && name != "html" && name != "body" {
						p.explicitAfterAfterBody = false
					}
					continue
				}
			}
			if pendingTarget == nil && len(p.nodesClosedAtDocumentEnd) > 0 {
				pendingTarget = p.pendingDocumentEndTarget()
			}
			if pendingTarget == nil && pendingTag && pendingClosing && (pendingName == "p" || pendingName == "button") {
				p.skipWhitespace()
				if p.pos < len(content) {
					if name, closing, ok := p.peekTagName(); !ok || !closing || name != pendingName {
						pendingTarget = p.pendingDocumentEndTarget()
					}
				}
			}
		}
		if !p.preserveNextRootWhitespace && pendingTarget == nil && !p.documentEndRecovery && len(p.activeFormatting) == 0 {
			p.skipWhitespace()
		}
		p.preserveNextRootWhitespace = false
		if p.pos >= len(content) {
			break
		}
		// Every whole-document parse uses the browser document insertion
		// modes.  Keep preamble comments, processing instructions, and the
		// accepted doctype as Document children, but create a locationless
		// HTML element before reprocessing the first token that exits the
		// before-html mode.  This must happen before parsing that token: doing
		// a final tree wrap would lose the real insertion parent used by table
		// foster parenting, active formatting, forms, lists, and foreign
		// content.
		if p.beforeHTMLMode && p.documentHTML == nil {
			if whitespace := p.parseLeadingDocumentWhitespace(root); whitespace != nil {
				// Character tokens that are whitespace are ignored in the
				// before-html insertion mode (including decoded references).
				continue
			}
			if p.pos >= len(content) {
				break
			}
			name, closing, tag := p.peekTagName()
			qualifyingHTML := tag && !closing && name == "html" && p.currentTagTokenWillEmit()
			preambleToken := p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() || p.nextTokenIsIgnoredDoctype() || p.isMissingEndTagName()
			ignoredEnd := tag && closing && name != "head" && name != "body" && name != "html" && name != "br"
			if !qualifyingHTML && !preambleToken && !ignoredEnd {
				html := syntheticDocumentWrapper("html", root)
				p.documentHTML = html
				p.explicitDocumentSkeleton = true
				p.beforeHTMLMode = false
				parsed, err := p.parseExplicitDocumentHTML(html, 0)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(root, parsed)
				continue
			}
		}
		// In the before-html insertion mode, unrelated end tags are parse
		// errors that are ignored and do not force creation of the document
		// element. Keeping this state intact lets a later emitted <html> opt
		// into the explicit-document skeleton path.
		if p.beforeHTMLMode {
			if name, closing, ok := p.peekTagName(); ok && closing && name != "head" && name != "body" && name != "html" && name != "br" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				continue
			}
		}
		if name, closing, ok := p.peekTagName(); ok && closing && p.adoptionIgnoredEnds[name] > 0 && p.activeFormattingEntryByName(name) == nil {
			start := p.pos
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			p.extendPreviousTextRange(root, start)
			p.adoptionIgnoredEnds[name]--
			continue
		}
		if name, closing, ok := p.peekTagName(); ok && closing && p.foreignIgnoredEnds[name] > 0 {
			start := p.pos
			if _, emitted, err := p.parseClosingTag(); err != nil {
				return nil, err
			} else if !emitted {
				p.extendPreviousTextRange(root, start)
				p.recoverOpenElementsAtEOF = true
			}
			p.retireForeignIgnoredEnd(name)
			continue
		}
		if name, closing, ok := p.peekTagName(); ok && closing && p.foreignIgnoredHTMLEnds[name] > 0 {
			start := p.pos
			if _, emitted, err := p.parseClosingTag(); err != nil {
				return nil, err
			} else if !emitted {
				p.extendPreviousTextRange(root, start)
			}
			p.foreignIgnoredHTMLEnds[name]--
			continue
		}
		if handled, err := p.handleForeignDocumentStart(); handled || err != nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		if handled, err := p.handleFormToken(root, false); handled || err != nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		if name, closing, ok := p.peekTagName(); ok && closing && isFormattingElement(name) {
			if isCoreAdoptionSubject(name) {
				if entry := p.activeFormattingEntryByName(name); entry == nil || entry.current == nil {
					start := p.pos
					if _, _, err := p.parseClosingTag(); err != nil {
						return nil, err
					}
					if entry != nil {
						p.removeActiveFormattingEntry(entry)
					}
					p.extendPreviousTextRange(root, start)
					continue
				}
			}
			if entry := p.activeFormattingEntryByName(name); entry != nil {
				if !p.openFormatting[entry.current] {
					start := p.pos
					if _, _, err := p.parseClosingTag(); err != nil {
						return nil, err
					}
					p.extendPreviousTextRange(root, start)
					p.removeActiveFormattingEntry(entry)
					continue
				}
				tokenStart := p.pos
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				p.refreshFormattingChain(entry.current)
				entry.current.ContentEnd = tokenStart
				entry.current.EndPos = p.pos
				entry.current.EndLine = p.line
				entry.current.EndColumn = p.col
				p.removeActiveFormattingEntry(entry)
				continue
			}
		}
		// At the document/body insertion level, an absent generic end tag is a
		// parse error that the HTML tree builder ignores. Paragraph and br ends
		// have dedicated in-body recovery below; all formatting/form/list/select
		// subjects have already had their identity-aware handlers above.
		if name, closing, ok := p.peekTagName(); ok && closing && name != "p" && name != "br" &&
			!p.isBogusEndTagOpen() && !p.isMissingEndTagName() && p.currentTagTokenWillEmit() {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		parent := root
		processingInstruction := p.nextTokenIsProcessingInstruction()
		commentToken := p.nextTokenIsComment()
		attachToPending := pendingTarget != nil && !p.nextTokenIsComment() && !processingInstruction
		attachToBody := p.documentEndRecovery && pendingTarget == nil && p.documentBody != nil &&
			(!p.explicitAfterAfterBody || (!commentToken && !processingInstruction))
		if attachToBody && p.explicitAfterAfterBody && !commentToken && !processingInstruction {
			p.explicitAfterAfterBody = false
		}
		if attachToPending {
			parent = pendingTarget
		} else if attachToBody {
			parent = p.documentBody
		}
		if action, err := p.handleSpecialFormattingStart(parent, "", 0); err != nil {
			return nil, err
		} else if action == adoptionContinue {
			continue
		}
		insertionParent := parent
		if !processingInstruction {
			insertionParent = p.formattingInsertionParent(parent, p.nextTokenReconstructsFormatting())
		}
		node, err := p.parseNode(insertionParent)
		if err != nil {
			return nil, err
		}
		if node != nil {
			if insertionParent != parent {
				p.appendFormattingChild(parent, insertionParent, node)
				if p.isSelfClosingTag(node.Name) {
					continue
				}
				// Non-void child parsing already consumed the following content.
				continue
			}
			if attachToPending {
				p.appendToPendingDocumentEnd(pendingTarget, node)
				continue
			}
			if attachToBody {
				p.appendToDocumentBody(node)
				continue
			}
			if node.Type == types.TextNode && !isHTMLWhitespaceString(node.Value) {
				p.bodyContentStarted = true
				p.beforeHTMLMode = false
			}
			if node.Type == types.ElementNode && (node.Name != "html" || !p.explicitDocumentSkeleton) {
				p.beforeHTMLMode = false
			}
			// The initial insertion phase ignores whitespace character tokens.
			// Test the decoded DOM value as well as the source bytes so numeric
			// references such as "&#32;" behave like literal leading spaces.
			if node.Type == types.TextNode && !p.doctypeWasIgnoredAfterBodyContent(root) && isHTMLWhitespaceString(node.Value) {
				continue
			}
			if node.Type == types.TextNode && len(root.Children) > 0 && root.Children[len(root.Children)-1].Type == types.TextNode {
				mergeTextNodes(root.Children[len(root.Children)-1], node)
			} else {
				root.Children = append(root.Children, node)
			}
			p.appendDeferredTableFoster(root, node)
			p.appendDeferredListComments(root, node)
			p.appendDeferredAdoption(root, node)
			p.appendDeferredFormSiblings(root, node)
		}
	}
	// EOF in the before-html insertion mode still creates the complete
	// locationless document skeleton. This covers empty, whitespace-only,
	// comment-only, and doctype-only documents.
	if p.documentHTML == nil {
		html := syntheticDocumentWrapper("html", root)
		p.documentHTML = html
		p.explicitDocumentSkeleton = true
		p.beforeHTMLMode = false
		parsed, err := p.parseExplicitDocumentHTML(html, 0)
		if err != nil {
			return nil, err
		}
		appendDocumentChild(root, parsed)
	}

	// Calculate final position
	if len(p.adoptionSplit) > 0 {
		p.appendAllDeferredAdoption(root)
	}
	p.refreshOpenFormattingText()
	for _, child := range root.Children {
		if child.Type == types.ElementNode {
			p.finalizeTextContent(child)
		}
	}
	removeMisplacedDoctypes(root)
	for _, node := range p.nodesClosedAtDocumentEnd {
		node.EndPos = p.pos
		node.ContentEnd = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
	}
	normalizeSyntheticLocations(root)
	root.EndLine = p.line
	root.EndColumn = p.col
	root.SourceLength = len(content)

	return root, nil
}

func normalizeSyntheticLocations(node *types.Node) {
	for _, child := range node.Children {
		if child.Type != types.ElementNode {
			continue
		}
		if child.NamespaceURI == "" {
			child.NamespaceURI = htmlElementNamespace
		}
		normalizeSyntheticLocations(child)
		if child.StartLine == 0 {
			child.StartPos, child.ContentStart, child.ContentEnd, child.EndPos = 0, 0, 0, 0
			child.StartColumn, child.EndLine, child.EndColumn = 0, 0, 0
		}
	}
}

func removeMisplacedDoctypes(root *types.Node) {
	seenDoctype, seenElement := false, false
	children := root.Children[:0]
	for _, child := range root.Children {
		switch child.Type {
		case types.DocumentTypeNode:
			if seenDoctype || seenElement {
				continue
			}
			seenDoctype = true
		case types.ElementNode, types.TextNode:
			seenElement = true
		}
		children = append(children, child)
	}
	root.Children = children
}

func (p *HTMLParser) nextTokenIsComment() bool {
	if p.pos >= len(p.content) || p.peek() != '<' {
		return false
	}
	remaining := p.content[p.pos:]
	return strings.HasPrefix(remaining, "<!--") || strings.HasPrefix(remaining, "<!") ||
		p.isBogusEndTagOpen()
}

func (p *HTMLParser) nextTokenIsProcessingInstruction() bool {
	return p.pos+1 < len(p.content) && p.content[p.pos] == '<' && p.content[p.pos+1] == '?'
}

func (p *HTMLParser) pendingDocumentEndTarget() *types.Node {
	if len(p.nodesClosedAtDocumentEnd) == 0 {
		return nil
	}
	return p.nodesClosedAtDocumentEnd[0]
}

func (p *HTMLParser) pendingDocumentEndBlockStarts() bool {
	name, closing, ok := p.peekTagName()
	return ok && !closing && p.paragraphClosingStartTag(name) && p.currentTagTokenWillEmit()
}

func (p *HTMLParser) resolvePendingDocumentEndNodes() {
	for _, node := range p.nodesClosedAtDocumentEnd {
		node.EndPos = p.pos
		node.ContentEnd = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
	}
	p.nodesClosedAtDocumentEnd = nil
}

func (p *HTMLParser) resolvePendingThroughParagraph() *types.Node {
	for index, node := range p.nodesClosedAtDocumentEnd {
		if node.Name == "p" {
			p.resolvePendingPrefix(index + 1)
			return p.pendingDocumentEndTarget()
		}
	}
	p.resolvePendingDocumentEndNodes()
	return nil
}

func (p *HTMLParser) resolvePendingThroughName(name string, consume bool) (bool, error) {
	for index, node := range p.nodesClosedAtDocumentEnd {
		if node.Name != name {
			continue
		}
		if consume {
			tokenStart := p.pos
			tokenStartLine, tokenStartColumn := p.line, p.col
			_, emitted, err := p.parseClosingTag()
			if err != nil {
				return false, err
			}
			if !emitted {
				return false, nil
			}
			for pendingIndex, pending := range p.nodesClosedAtDocumentEnd[:index+1] {
				end := tokenStart
				if pendingIndex == index {
					end = p.pos
				}
				pending.EndPos = end
				pending.ContentEnd = tokenStart
				if pendingIndex == index {
					pending.EndLine = p.line
					pending.EndColumn = p.col
				} else {
					pending.EndLine = tokenStartLine
					pending.EndColumn = tokenStartColumn
				}
			}
			p.nodesClosedAtDocumentEnd = p.nodesClosedAtDocumentEnd[index+1:]
			return true, nil
		}
		p.resolvePendingPrefix(index + 1)
		return true, nil
	}
	return false, nil
}

func (p *HTMLParser) resolvePendingPrefix(count int) {
	if count > len(p.nodesClosedAtDocumentEnd) {
		count = len(p.nodesClosedAtDocumentEnd)
	}
	for _, node := range p.nodesClosedAtDocumentEnd[:count] {
		node.EndPos = p.pos
		node.ContentEnd = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
	}
	p.nodesClosedAtDocumentEnd = p.nodesClosedAtDocumentEnd[count:]
}

func (p *HTMLParser) appendToPendingDocumentEnd(target, node *types.Node) {
	if target == nil {
		return
	}
	node.Parent = target
	if node.Type == types.TextNode && len(target.Children) > 0 && target.Children[len(target.Children)-1].Type == types.TextNode {
		mergeTextNodes(target.Children[len(target.Children)-1], node)
	} else {
		target.Children = append(target.Children, node)
	}
	value := node.TextContent
	switch node.Type {
	case types.TextNode:
		value = node.Value
	case types.CommentNode, types.ProcessingInstructionNode:
		value = ""
	}
	for _, pending := range p.nodesClosedAtDocumentEnd {
		p.markTextDirty(pending)
		pending.EndPos = p.pos
		pending.ContentEnd = p.pos
		pending.EndLine = p.line
		pending.EndColumn = p.col
	}
	p.addDocumentTextContent(value)
}

func (p *HTMLParser) addDocumentTextContent(value string) {
	if value == "" {
		return
	}
	if p.documentBody != nil {
		p.markTextDirty(p.documentBody)
	}
	if p.documentHTML != nil {
		p.markTextDirty(p.documentHTML)
	}
}

func (p *HTMLParser) appendToDocumentBody(node *types.Node) {
	if p.documentBody == nil {
		return
	}
	node.Parent = p.documentBody
	p.documentBody.Children = append(p.documentBody.Children, node)
	p.markTextDirty(p.documentBody)
	value := node.TextContent
	switch node.Type {
	case types.TextNode:
		value = node.Value
	case types.CommentNode, types.ProcessingInstructionNode:
		value = ""
	}
	p.addDocumentTextContent(value)
}

func isBinaryInput(content string) bool {
	if strings.HasPrefix(content, "\x1f\x8b") || !utf8.ValidString(content) {
		return true
	}

	for i := 0; i < len(content); i++ {
		c := content[i]
		if c != 0 && ((c < ' ' && c != '\t' && c != '\n' && c != '\r' && c != '\f') || c == 0x7f) {
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

	if p.peek() == '<' && p.startsMarkupToken() {
		if p.isBogusEndTagOpen() {
			return p.parseBogusEndTagComment(parent, startPos, startLine, startCol)
		}
		if p.isMissingEndTagName() {
			p.advanceRune(3) // Ignore "</>".
			if p.doctypeWasIgnoredAfterBodyContent(parent) {
				p.preserveNextRootWhitespace = true
			}
			return nil, nil
		}
		if recovered, consumed, err := p.consumeVoidElementClosingTag(parent); consumed || err != nil {
			return recovered, err
		}
		if p.pos+2 < len(p.content) && p.content[p.pos:p.pos+2] == "</" && isASCIIAlpha(p.content[p.pos+2]) {
			tokenStart := p.pos
			name, emitted, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if !emitted {
				p.extendPreviousTextRange(parent, tokenStart)
				return nil, nil
			}
			// The "in body" tree builder ignores a script end tag when
			// there is no script element in scope. This commonly follows a
			// correctly closed script after escaped script-data recovery.
			if name == "script" {
				return nil, nil
			}
			if name == "li" || name == "dd" || name == "dt" {
				return nil, nil
			}
			if name == "select" && (p.activeSelect == nil || p.selectBlockedByTable) {
				return nil, nil
			}
			if name == "p" {
				if parent.Type == types.DocumentNode && !p.bodyContentStarted {
					return nil, nil
				}
				return syntheticParagraph(parent), nil
			}
			// Any other end tag whose name has no matching element in scope is a
			// parse error whose token is ignored (HTML5 "in body" generic end-tag
			// rule, which the "in table" anything-else and "in template" any-other
			// end-tag entries route into). This parser reaches this branch only
			// when no element with that name is open in the current recursion
			// chain, and any matching ancestor above a scope boundary (table, td,
			// th, caption, template, select) is not in scope, so ignoring always
			// matches the spec's "not in scope" branch.
			p.extendPreviousTextRange(parent, tokenStart)
			return nil, nil
		}
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
			if strings.HasPrefix(p.content[p.pos:], "<!--") {
				return p.parseComment(parent, startPos, startLine, startCol)
			}
			if strings.HasPrefix(strings.ToUpper(p.content[p.pos:]), "<!DOCTYPE") {
				doctype, err := p.parseDoctype(parent, startPos, startLine, startCol)
				if err != nil {
					return nil, err
				}
				if p.shouldKeepDoctype(parent) {
					return doctype, nil
				}
				if p.doctypeWasIgnoredAfterBodyContent(parent) {
					p.preserveNextRootWhitespace = true
				}
				return nil, nil
			}
			return p.parseBogusComment(parent, startPos, startLine, startCol)
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

	// The tag-open state only creates a start tag when the first character is
	// ASCII alpha. Other less-than sequences are emitted by parseTextNode.
	tagName := p.parseTagName()
	if tagName == "" {
		return nil, fmt.Errorf("expected tag name at position %d", p.pos)
	}
	// The in-body tree builder reparses the legacy image start tag as img.
	if tagName == "image" {
		tagName = "img"
	}

	node := &types.Node{
		Type:           types.ElementNode,
		Name:           tagName,
		NamespaceURI:   htmlElementNamespace,
		Attributes:     make(map[string]string),
		AttributeOrder: []string{},
		Children:       []*types.Node{},
		Parent:         parent,
		StartPos:       startPos,
		StartLine:      startLine,
		StartColumn:    startCol,
	}
	if owner := p.templateOwner[parent]; owner != nil {
		p.templateOwner[node] = owner
	}

	selfClosing, emitted := p.parseStartTagTail(node)
	if !emitted {
		// EOF in any start-tag state discards the incomplete token. parse5 and
		// browsers associate its original bytes with an immediately preceding
		// text node even though those bytes do not contribute to its value.
		p.extendPreviousTextRange(parent, startPos)
		return nil, nil
	}
	node.ContentStart = p.pos
	if selfClosing && p.foreignBreakoutRecovery && !p.isSelfClosingTag(node.Name) {
		// HTML ignores the self-closing flag on an ordinary element. parse5's
		// recovered content boundary begins at the flag's solidus.
		node.ContentStart = p.pos - 2
	}
	switch node.Name {
	case "svg":
		node.NamespaceURI = svgElementNamespace
		adjustSVGAttributes(node)
		initializeAttributeMetadata(node, true)
	case "math":
		node.NamespaceURI = mathElementNamespace
		adjustMathMLAttributes(node)
		initializeAttributeMetadata(node, true)
	default:
		initializeAttributeMetadata(node, false)
	}
	if node.NamespaceURI == htmlElementNamespace && p.foreignIgnoredEnds[node.Name] > 0 {
		// A real HTML element with the same name supersedes an omitted stale
		// foreign close. Its eventual end tag belongs to this new identity.
		p.retireForeignIgnoredEnd(node.Name)
	}
	if node.NamespaceURI == htmlElementNamespace && p.foreignBreakoutRecovery && p.foreignBreakoutBoundary == nil && !p.isSelfClosingTag(node.Name) {
		p.foreignBreakoutBoundary = node
		defer func() {
			if p.foreignBreakoutBoundary == node {
				p.foreignBreakoutBoundary = nil
				p.foreignBreakoutRecovery = p.foreignIgnoredEndCount > 0
			}
		}()
	}
	qualifyingDocumentHTML := node.Name == "html" && parent == p.documentRoot && p.beforeHTMLMode && p.documentHTML == nil
	if qualifyingDocumentHTML {
		p.explicitDocumentSkeleton = true
		p.beforeHTMLMode = false
	}
	if node.Name == "form" {
		previousForm := p.activeForm
		insideTemplate := p.templateOwnerForNode(parent) != nil
		p.activeForm = node
		p.openForms[node] = true
		p.formEOFBoundaries[node] = true
		if insideTemplate {
			defer func() { p.activeForm = previousForm }()
		}
	}
	p.elementContentStart[node] = sourcePoint{line: p.line, column: p.col}
	if isFormattingElement(node.Name) {
		p.addActiveFormatting(node)
		p.setFormattingOpen(node, true)
		defer p.setFormattingOpen(node, false)
	}
	restoreElementContext := p.enterElementContext(node)
	defer restoreElementContext()
	if !isFormattingElement(node.Name) {
		defer p.closeFormattingOnElementReturn(node)
	}
	if qualifyingDocumentHTML {
		return p.parseExplicitDocumentHTML(node, node.ContentStart)
	}
	if p.explicitDocumentSkeleton && node.Name == "head" && node.Parent == p.documentHTML {
		return p.parseExplicitDocumentHead(node, node.ContentStart)
	}
	if node.Name == "template" && node.NamespaceURI == htmlElementNamespace {
		return p.parseTemplateElement(node, node.ContentStart)
	}
	if node.Name == "noscript" && !p.scriptingEnabled && node.Parent == p.documentHead {
		return p.parseHeadNoscriptDisabled(node, node.ContentStart)
	}

	// In HTML, a self-closing flag on an ordinary element is acknowledged as
	// a parse error but does not close the element. Void elements remain empty.
	if p.isSelfClosingTag(node.Name) {
		node.EndPos = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
		// For self-closing tags, content start and end are the same (no inner content)
		node.ContentStart = p.pos
		node.ContentEnd = p.pos
		for _, entry := range p.activeFormatting {
			current := entry.current
			if entry.marker || current == nil || !p.openFormatting[current] || !nodeWithin(parent, current) {
				continue
			}
			current.TextContent += node.TextContent
			current.ContentEnd = p.pos
			current.EndPos = p.pos
			current.EndLine = p.line
			current.EndColumn = p.col
		}
		return node, nil
	}
	if node.NamespaceURI == svgElementNamespace || node.NamespaceURI == mathElementNamespace {
		if selfClosing {
			p.finishForeignSelfClosing(node)
			return node, nil
		}
		if p.foreignHTMLDepth > 0 {
			previousNames := p.activeSVGNames
			p.activeSVGNames = make(map[string]*types.Node)
			defer func() { p.activeSVGNames = previousNames }()
		}
		return p.parseSVGElement(node, p.pos)
	}

	// Mark the start of inner content (after opening tag)
	contentStartPos := node.ContentStart
	if node.Name == "frameset" && p.explicitDocumentSkeleton && p.documentBody == nil &&
		(parent == p.documentHTML || (parent != nil && parent.Name == "frameset")) {
		return p.parseDocumentFramesetElement(node, contentStartPos)
	}
	if node.Name == "table" {
		return p.parseTableElement(node, contentStartPos)
	}

	// Handle raw text elements like script, style, textarea, title
	if p.isRawTextElement(node.Name) || (node.Name == "noscript" && p.scriptingEnabled) {
		textState, err := p.parseRawTextContentWithPos(node.Name)
		if err != nil {
			return nil, err
		}
		textContent := textState.content

		// Create a single text node for the raw content
		if textContent != "" {
			textNode := &types.Node{
				Type:        types.TextNode,
				Name:        "#text",
				Value:       textContent,
				TextContent: textContent,
				Parent:      node,
				StartPos:    textState.start,
				EndPos:      textState.end,
				StartLine:   textState.startLine,
				StartColumn: textState.startColumn,
				EndLine:     textState.endLine,
				EndColumn:   textState.endColumn,
			}
			node.Children = append(node.Children, textNode)
		}

		node.TextContent = textContent
		node.ContentStart = contentStartPos
		node.ContentEnd = textState.end
		node.EndPos = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
		return node, nil
	}

	if node.Name == "plaintext" {
		textState := p.parsePlaintextContent()
		if textState.content != "" {
			textParent := p.formattingInsertionParent(node, true)
			textNode := &types.Node{
				Type: types.TextNode, Name: "#text", Value: textState.content, TextContent: textState.content,
				Parent: textParent, StartPos: textState.start, EndPos: textState.end,
				StartLine: textState.startLine, StartColumn: textState.startColumn,
				EndLine: textState.endLine, EndColumn: textState.endColumn,
			}
			if textParent != node {
				p.appendFormattingChild(node, textParent, textNode)
			} else {
				textParent.Children = append(textParent.Children, textNode)
			}
		}
		node.TextContent = textState.content
		node.ContentStart = contentStartPos
		node.ContentEnd = textState.end
		node.EndPos = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
		return node, nil
	}

	// Parse child nodes normally for other elements
	var textContent strings.Builder
	suppressInitialPreformattedLF := node.NamespaceURI == htmlElementNamespace && (node.Name == "pre" || node.Name == "listing")
	preformattedRangeStart := -1
	preformattedRangeLine, preformattedRangeColumn := 0, 0

	for p.pos < len(p.content) {
		if p.pos >= len(p.content) {
			break
		}
		if preformattedRangeStart >= 0 && !p.preformattedRangeStartContinues() {
			preformattedRangeStart = -1
		}
		if suppressInitialPreformattedLF {
			suppressed, _, start, line, column := p.consumeInitialPreformattedLF(&suppressInitialPreformattedLF)
			if suppressed {
				// Keep this tentative boundary across tokenizer input that emits no
				// token (notably exact </>). The next loop either applies it to a
				// leading LF character token or retires it on any emitted token.
				preformattedRangeStart = start
				preformattedRangeLine, preformattedRangeColumn = line, column
				continue
			}
		}
		if consumed, err := p.consumePendingAdoptionEnd(node); consumed || err != nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		if p.currentTemplateEndCloses(node) {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		if p.currentTemplateTableTransitionWillEmit(node) {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		if name, closing, ok := p.peekTagName(); ok && closing {
			if owner, blocked := p.matchingForeignAncestorWithinIntegration(node, name); owner != nil {
				if blocked {
					start := p.pos
					if _, emitted, err := p.parseClosingTag(); err != nil {
						return nil, err
					} else if !emitted {
						p.extendPreviousTextRange(node, start)
					}
					continue
				}
				p.finishImplicitElement(node, textContent.String(), contentStartPos)
				return node, nil
			}
			if name != node.Name && p.foreignIgnoredEnds[name] > 0 && nearestOpenElementByName(node, name) == nil {
				start := p.pos
				if _, emitted, err := p.parseClosingTag(); err != nil {
					return nil, err
				} else if !emitted {
					p.extendPreviousTextRange(node, start)
					p.recoverOpenElementsAtEOF = true
				}
				p.retireForeignIgnoredEnd(name)
				continue
			}
			if name != node.Name && p.foreignIgnoredHTMLEnds[name] > 0 && nearestOpenElementByName(node, name) == nil {
				start := p.pos
				if _, emitted, err := p.parseClosingTag(); err != nil {
					return nil, err
				} else if !emitted {
					p.extendPreviousTextRange(node, start)
				}
				p.foreignIgnoredHTMLEnds[name]--
				continue
			}
			if p.foreignBreakoutRecovery && name != node.Name && nearestOpenElementByName(node.Parent, name) != nil && !isHTMLSpecialElement(node.Name) {
				p.foreignIgnoredHTMLEnds[node.Name]++
				p.finishImplicitElement(node, textContent.String(), contentStartPos)
				return node, nil
			}
			if p.foreignIgnoredEndCount > 0 && name != node.Name && foreignRecoveryGenericEndTarget(node, name) != nil {
				p.foreignIgnoredHTMLEnds[node.Name]++
				p.finishImplicitElement(node, textContent.String(), contentStartPos)
				return node, nil
			}
		}
		if p.explicitDocumentSkeleton {
			if name, closing, ok := p.peekTagName(); ok && !closing && p.currentTagTokenWillEmit() {
				switch name {
				case "html":
					if err := p.mergeDuplicateStartInto(p.documentHTML, "html"); err != nil {
						return nil, err
					}
					continue
				case "body":
					if p.documentBody != nil && p.activeSelect == nil {
						if err := p.mergeDuplicateStartInto(p.documentBody, "body"); err != nil {
							return nil, err
						}
						continue
					}
				case "head":
					if p.documentHead != nil {
						p.consumeIgnoredStartTag()
						continue
					}
				}
			}
		}
		if p.templateOwnerForNode(node) != nil {
			if name, closing, tag := p.peekTagName(); tag && !closing && (name == "frame" || name == "frameset") && p.currentTagTokenWillEmit() {
				p.consumeIgnoredStartTag()
				continue
			}
		}
		if handled, err := p.handleForeignDocumentStart(); handled || err != nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		if name, closing, tag := p.peekTagName(); tag && closing && name == "form" && node != p.activeForm && isImpliedEndTagName(node.Name) && p.activeForm != nil && formattingInOrdinaryScope(p.activeForm, node) {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		if handled, err := p.handleFormToken(node, p.tableFosterDepth.active()); handled || err != nil {
			if err != nil {
				return nil, err
			}
			if p.closedForms[node] {
				delete(p.formEOFBoundaries, node)
				p.markTextDirty(node)
				p.refreshNodeTextContent(node)
				return node, nil
			}
			continue
		}
		if (p.tableFosterDepth.active() && p.currentFosterTransitionWillEmit()) || (p.tableCellDepth.active() && p.currentCellTransitionWillEmit()) {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		if p.tableFosterDepth.active() && p.currentFosterIgnoredEndTag() {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if p.tableCellDepth.active() && p.currentCellIgnoredEndTag() {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if p.tableCaptionDepth.active() && p.currentCaptionIgnoredEndTag() {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if p.currentNestedButtonStart(node) {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		if p.tableCaptionDepth.active() && p.currentCaptionTransitionWillEmit() {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		// The in-table rules insert an input whose parsed type is exactly
		// ASCII-case-insensitive "hidden" at the current node and immediately
		// pop it. This must happen before normal in-body reconstruction/foster
		// parenting, including while an ordinary descendant is being parsed.
		if p.tableFosterDepth.active() {
			hiddenInput, consumed, err := p.parseHiddenInputInTable(node)
			if err != nil {
				return nil, err
			}
			if consumed {
				if hiddenInput != nil {
					p.appendUniqueChild(node, hiddenInput)
				}
				continue
			}
		}
		if p.listDocumentEndMode != listDocumentEndNone && (p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction()) {
			commentParent := p.documentHTML
			if p.listDocumentEndMode == listDocumentEndHTML {
				commentParent = p.documentRoot
			}
			comment, err := p.parseNode(commentParent)
			if err != nil {
				return nil, err
			}
			if comment != nil {
				comment.Parent = commentParent
				if p.listDocumentEndMode == listDocumentEndHTML {
					p.deferredRootComments = append(p.deferredRootComments, comment)
				} else {
					p.deferredHTMLComments = append(p.deferredHTMLComments, comment)
				}
			}
			continue
		}
		if p.listDocumentEndMode != listDocumentEndNone && p.nextTokenIsIgnoredDoctype() {
			start, line, column := p.pos, p.line, p.col
			if _, err := p.parseDoctype(node, start, line, column); err != nil {
				return nil, err
			}
			if p.pos == start {
				return nil, fmt.Errorf("failed to consume doctype at position %d", start)
			}
			continue
		}
		if p.listDocumentEndMode != listDocumentEndNone && p.nextTokenIsHTMLStart() {
			if err := p.mergeDuplicateHTMLStart(); err != nil {
				return nil, err
			}
			continue
		}
		if p.currentListDocumentEndTag() {
			needsOpenRecovery := p.hasActiveListItem() || len(p.activeFormatting) > 0
			tokenStart := p.pos
			tokenLine, tokenColumn := p.line, p.col
			name, _, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if name == "body" && p.documentBody != nil {
				p.listDocumentEndMode = listDocumentEndBody
				p.documentBodyBoundary.set(tokenStart, p.pos, p.line, p.col)
			}
			if name == "html" && p.documentHTML != nil {
				p.listDocumentEndMode = listDocumentEndHTML
				p.documentHTMLBoundary.set(tokenStart, p.pos, p.line, p.col)
				if p.documentBody != nil && p.documentBodyBoundary.end.pos == 0 {
					p.documentBodyBoundary.set(tokenStart, tokenStart, tokenLine, tokenColumn)
				}
			}
			if len(p.activeFormatting) > 0 && !p.formattingDocumentEnd {
				p.formattingDocumentEnd = true
				p.formattingDocumentEndPrior = p.recoverOpenElementsAtEOF
			}
			if needsOpenRecovery {
				p.recoverOpenElementsAtEOF = true
			}
			continue
		}
		if p.listDocumentEndMode != listDocumentEndNone && p.currentTokenReentersListBody() {
			p.listDocumentEndMode = listDocumentEndNone
		}
		if p.activeSelect != nil {
			name, closing, tag := p.peekTagName()
			// The select insertion mode ignores noscript tags in both scripting
			// configurations; their contents are still processed as select tokens.
			if tag && name == "noscript" && p.currentTagTokenWillEmit() {
				if closing {
					if _, _, err := p.parseClosingTag(); err != nil {
						return nil, err
					}
				} else {
					p.consumeIgnoredStartTag()
				}
				continue
			}
			if tag && !closing && (name == "html" || name == "body" || name == "head") && p.currentTagTokenWillEmit() {
				tokenStart, tokenLine, tokenColumn := p.pos, p.line, p.col
				var err error
				switch name {
				case "html":
					err = p.mergeDuplicateStartInto(p.documentHTML, "html")
				case "body":
					p.consumeIgnoredStartTag()
				case "head":
					p.consumeIgnoredStartTag()
				}
				if err != nil {
					return nil, err
				}
				p.preserveIgnoredTextRange(node, tokenStart, tokenLine, tokenColumn)
				continue
			}
			// Bare table-structure starts are parse errors in a customizable
			// select and their wrappers are ignored. Specialized table/cell and
			// template table-derived transitions have already unwound above; in
			// ordinary in-body select parsing, retain only the following content.
			if tag && !closing && isTableStructureStart(name) && p.currentTagTokenWillEmit() {
				p.consumeIgnoredStartTag()
				continue
			}
		}
		// Active select identities are dynamically scoped to the recursive open
		// path, so a cached target is necessarily an ancestor of this frame.
		// Avoid re-walking that path in every descendant during one unwind.
		if target, action := p.selectTreeUnwindTarget(node); target != nil {
			if node == target {
				tokenStart := p.pos
				tokenLine, tokenColumn := p.line, p.col
				switch action {
				case selectUnwindEnd:
					_, emitted, err := p.parseClosingTag()
					if err != nil {
						return nil, err
					}
					if !emitted {
						p.extendPreviousTextRange(node, tokenStart)
						p.recoverOpenElementsAtEOF = true
						p.finishImplicitElement(node, textContent.String(), contentStartPos)
						p.clearSelectUnwindCache()
						return node, nil
					}
					node.TextContent = textContent.String()
					node.ContentStart = contentStartPos
					node.ContentEnd = tokenStart
					node.EndPos = p.pos
					node.EndLine = p.line
					node.EndColumn = p.col
					p.clearSelectUnwindCache()
					return node, nil
				case selectUnwindNestedSelect:
					if !p.consumeIgnoredStartTag() {
						p.finishImplicitElement(node, textContent.String(), contentStartPos)
						p.clearSelectUnwindCache()
						return node, nil
					}
					p.finishElementAt(node, textContent.String(), contentStartPos, tokenStart, tokenLine, tokenColumn)
					p.clearSelectUnwindCache()
					return node, nil
				}
				p.clearSelectUnwindCache()
			}
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		if p.currentIgnoredSelectEndTag(node) {
			tokenStart := p.pos
			tokenLine, tokenColumn := p.line, p.col
			ignoredName, _, _ := p.peekTagName()
			if _, emitted, err := p.parseClosingTag(); err != nil {
				return nil, err
			} else if !emitted {
				p.extendPreviousTextRange(node, tokenStart)
			} else {
				p.preserveIgnoredTextRange(node, tokenStart, tokenLine, tokenColumn)
				var outside *types.Node
				if p.activeSelect != nil {
					outside = nearestOpenElementByName(p.activeSelect.Parent, ignoredName)
				}
				if outside != nil {
					p.recoverOpenElementsAtEOF = true
					p.selectDocumentEndRecovery = p.activeSelect
				}
			}
			continue
		}
		if target, explicit := p.listItemUnwindTarget(node); target != nil {
			if currentIsTarget := node == target; currentIsTarget && explicit {
				tokenStart := p.pos
				name, emitted, err := p.parseClosingTag()
				if err != nil {
					return nil, err
				}
				if !emitted || name != target.Name {
					p.extendPreviousTextRange(node, tokenStart)
					p.recoverOpenElementsAtEOF = true
					p.finishImplicitElement(node, textContent.String(), contentStartPos)
					return node, nil
				}
				node.TextContent = textContent.String()
				node.ContentStart = contentStartPos
				node.ContentEnd = tokenStart
				node.EndPos = p.pos
				node.EndLine = p.line
				node.EndColumn = p.col
				if target.Parent != nil && (target.Parent.Name == "ul" || target.Parent.Name == "ol" || target.Parent.Name == "dl") {
					p.recoverOpenElementsAtEOF = true
					p.listEOFRecoveryBoundary = target.Parent
				}
				return node, nil
			}
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			if node == target {
				p.clearListUnwindCache()
			}
			return node, nil
		}
		if p.currentIgnoredListEndTag(node) {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if name, closing, ok := p.peekTagName(); ok && closing && p.adoptionIgnoredEnds[name] > 0 && p.activeFormattingEntryByName(name) == nil && nearestOpenElementByName(node, name) == nil {
			start := p.pos
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			p.extendPreviousTextRange(node, start)
			p.adoptionIgnoredEnds[name]--
			continue
		}
		if node.Name == "html" && p.documentEndRecovery && p.nextTokenIsProcessingInstruction() {
			instruction, err := p.parseNode(p.documentRoot)
			if err != nil {
				return nil, err
			}
			if instruction != nil {
				instruction.Parent = p.documentRoot
				p.deferredRootComments = append(p.deferredRootComments, instruction)
			}
			continue
		}
		if node.Name == "html" && p.documentEndRecovery && p.pendingDocumentEndTarget() == nil && p.documentBody != nil {
			name, closing, tag := p.peekTagName()
			if !tag || !closing || (name != "html" && name != "body") {
				insertionParent := p.formattingInsertionParent(p.documentBody, p.nextTokenReconstructsFormatting())
				child, err := p.parseNode(insertionParent)
				if err != nil {
					return nil, err
				}
				if child != nil {
					if insertionParent != p.documentBody {
						p.appendFormattingChild(p.documentBody, insertionParent, child)
						p.markTextDirty(p.documentBody)
						p.refreshNodeTextContent(p.documentBody)
						p.markTextDirty(p.documentHTML)
						p.refreshNodeTextContent(p.documentHTML)
					} else {
						p.appendToDocumentBody(child)
					}
					textContent.WriteString(child.TextContent)
				}
				continue
			}
		}
		if node.Name == "html" && p.pendingDocumentEndTarget() != nil {
			if p.pendingDocumentEndBlockStarts() {
				parent := p.resolvePendingThroughParagraph()
				if parent == nil {
					parent = p.documentBody
				}
				child, err := p.parseNode(parent)
				if err != nil {
					return nil, err
				}
				if child != nil {
					if p.pendingDocumentEndTarget() != nil {
						p.appendToPendingDocumentEnd(p.pendingDocumentEndTarget(), child)
					} else {
						p.appendToDocumentBody(child)
					}
					textContent.WriteString(child.TextContent)
				}
				continue
			}
			if name, closing, ok := p.peekTagName(); ok && !closing && name == "table" && p.quirksMode && p.currentTagTokenWillEmit() {
				var paragraph *types.Node
				for _, pending := range p.nodesClosedAtDocumentEnd {
					if pending.Name == "p" {
						paragraph = pending
						break
					}
				}
				if paragraph != nil {
					child, err := p.parseNode(paragraph)
					if err != nil {
						return nil, err
					}
					if child != nil {
						p.appendToPendingDocumentEnd(paragraph, child)
						textContent.WriteString(child.TextContent)
					}
					continue
				}
			}
			if name, closing, ok := p.peekTagName(); ok && closing {
				if resolved, err := p.resolvePendingThroughName(name, true); err != nil {
					return nil, err
				} else if resolved {
					continue
				}
			}
			if p.peek() != '<' || !p.startsMarkupToken() {
				child, err := p.parseNode(p.pendingDocumentEndTarget())
				if err != nil {
					return nil, err
				}
				if child != nil {
					p.appendToPendingDocumentEnd(p.pendingDocumentEndTarget(), child)
					textContent.WriteString(child.TextContent)
				}
				continue
			}
		}
		if p.paragraphAncestorClose != "" {
			name, closing, ok := p.peekTagName()
			if ok && closing && name == p.paragraphAncestorClose {
				if node.Name != name {
					if (name == "html" || name == "body") && node.Name != "html" && node.Name != "body" {
						p.nodesClosedAtDocumentEnd = append(p.nodesClosedAtDocumentEnd, node)
						p.documentEndRecovery = true
					}
					p.finishImplicitElement(node, textContent.String(), contentStartPos)
					return node, nil
				}
				p.paragraphAncestorClose = ""
			}
		}

		// HTML's tree-construction rules implicitly close the current table row
		// when another row/section begins or the containing table/section ends.
		// Leave the triggering tag unconsumed so the parent can process it.
		if p.shouldImplicitlyCloseTableRow(node.Name) {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}

		// In the "in body" insertion mode, block-like start tags close a p
		// in button scope before the triggering token is processed. Descendant
		// inline frames unwind the same way, leaving the token for the p's
		// parent. An ancestor end tag and an explicit p end tag do likewise.
		if p.shouldUnwindForParagraph(node) {
			name, closing, _ := p.peekTagName()
			if closing && (name == "html" || name == "body") && node.Name != "html" && node.Name != "body" {
				p.nodesClosedAtDocumentEnd = append(p.nodesClosedAtDocumentEnd, node)
				p.documentEndRecovery = true
			}
			if node == p.activeParagraph {
				if closing && name != "p" {
					p.paragraphAncestorClose = name
				}
			}
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		// Body/html end tags change the document insertion mode without
		// discarding ordinary descendants that are still logically open. Keep
		// those frames in the pending document-end stack so after-body content
		// is reprocessed through them. Specialized paragraph/list/select paths
		// above get first refusal because they also maintain their own identity
		// and source-range state.
		if p.explicitDocumentSkeleton {
			if name, closing, ok := p.peekTagName(); ok && closing && (name == "body" || name == "html") {
				insideBody := p.documentBody != nil && node != p.documentBody && nodeWithin(node, p.documentBody)
				insideHTML := p.documentHTML != nil && node != p.documentHTML && nodeWithin(node, p.documentHTML)
				if (name == "body" && insideBody) || (name == "html" && insideHTML) {
					if name != "html" || node != p.documentBody {
						seen := slices.Contains(p.nodesClosedAtDocumentEnd, node)
						if !seen {
							p.nodesClosedAtDocumentEnd = append(p.nodesClosedAtDocumentEnd, node)
						}
						p.documentEndRecovery = true
					}
					p.finishImplicitElement(node, textContent.String(), contentStartPos)
					return node, nil
				}
			}
		}
		if action, err := p.handleSpecialFormattingStart(node, textContent.String(), contentStartPos); err != nil {
			return nil, err
		} else {
			switch action {
			case adoptionContinue:
				continue
			case adoptionReturn:
				return node, nil
			}
		}
		if name, closing, ok := p.peekTagName(); ok && closing && isCoreAdoptionSubject(name) {
			action, err := p.handleCoreAdoption(node, name, textContent.String(), contentStartPos, true)
			if err != nil {
				return nil, err
			}
			switch action {
			case adoptionContinue:
				continue
			case adoptionReturn:
				return node, nil
			}
		}
		if name, closing, ok := p.peekTagName(); ok && closing && isFormattingElement(name) {
			if entry := p.activeFormattingEntryByName(name); entry != nil && entry.current != nil && entry.current != node && nodeWithin(entry.current, node) {
				tokenStart := p.pos
				if _, emitted, err := p.parseClosingTag(); err != nil {
					return nil, err
				} else if !emitted {
					p.extendPreviousTextRange(entry.current, tokenStart)
				}
				p.refreshFormattingChain(entry.current)
				entry.current.ContentEnd = tokenStart
				entry.current.EndPos = p.pos
				entry.current.EndLine = p.line
				entry.current.EndColumn = p.col
				p.removeActiveFormatting(entry.current)
				p.recoverOpenElementsAtEOF = true
				continue
			}
		}
		if isFormattingElement(node.Name) {
			name, closing, ok := p.peekTagName()
			if ok && closing && name == node.Name {
				if active := p.formattingEntryForNode(node); active != nil && active.current != node {
					tokenStart := p.pos
					if _, emitted, err := p.parseClosingTag(); err != nil {
						return nil, err
					} else if !emitted {
						p.extendPreviousTextRange(node, tokenStart)
					}
					p.removeActiveFormatting(node)
					p.finishImplicitElement(node, textContent.String(), contentStartPos)
					return node, nil
				}
			}
			if ok && closing && name != node.Name && isParagraphClosingAncestorEndTag(name) && nearestOpenElementByName(node.Parent, name) != nil {
				p.finishImplicitElement(node, textContent.String(), contentStartPos)
				return node, nil
			}
		}

		// Complete in-body generic/special end-tag recovery before falling into
		// the recursive parser's strict matched-end path. Special block/list
		// ends pop every intervening node when their target is in ordinary scope;
		// any-other ends pop only through non-special nodes. If no eligible target
		// exists, the emitted token is ignored. Leaving a target token unconsumed
		// lets each recursive frame unwind naturally and the matching ancestor
		// consume it with the correct source range.
		if name, closing, ok := p.peekTagName(); ok && closing && name != node.Name && name != "p" && name != "br" && name != "script" &&
			!p.isBogusEndTagOpen() && !p.isMissingEndTagName() && p.foreignHTMLDepth == 0 &&
			(p.activeSelect == nil || !nodeIsDescendantOrSelf(node, p.activeSelect)) &&
			(!isHeadingName(node.Name) || !isHeadingName(name)) && p.currentTagTokenWillEmit() {
			if p.activeParagraph != nil {
				if _, isAncestor := p.activeParagraphAncestors[name]; isAncestor && !isParagraphClosingAncestorEndTag(name) {
					// The established paragraph recovery path below records scoped
					// EOF state for this ignored ancestor end.
					goto strictMatchedEndPath
				}
			}
			target := p.cachedGenericEndTarget(node, name)
			if target != nil {
				p.finishImplicitElement(node, textContent.String(), contentStartPos)
				return node, nil
			}
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}

	strictMatchedEndPath:
		// Check for closing tag
		if p.peek() == '<' && p.pos+1 < len(p.content) && p.content[p.pos+1] == '/' && p.startsMarkupToken() {
			if p.isBogusEndTagOpen() {
				commentStart := p.pos
				commentLine := p.line
				commentColumn := p.col
				comment, err := p.parseBogusEndTagComment(node, commentStart, commentLine, commentColumn)
				if err != nil {
					return nil, err
				}
				node.Children = append(node.Children, comment)
				continue
			}
			if p.isMissingEndTagName() {
				p.advanceRune(3) // Ignore "</>" and merge surrounding text.
				continue
			}
			if recovered, consumed, err := p.consumeVoidElementClosingTag(node); consumed || err != nil {
				if err != nil {
					return nil, err
				}
				if recovered != nil {
					node.Children = append(node.Children, recovered)
					textContent.WriteString(recovered.TextContent)
				}
				continue
			}

			// Mark content end position before closing tag
			contentEndPos := p.pos
			contentEndLine, contentEndColumn := p.line, p.col
			closingTag, emitted, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if !emitted {
				p.extendPreviousTextRange(node, contentEndPos)
				node.TextContent = textContent.String()
				node.ContentStart = contentStartPos
				node.ContentEnd = contentEndPos
				node.EndPos = p.pos
				node.EndLine = p.line
				node.EndColumn = p.col
				return node, nil
			}
			if closingTag == node.Name {
				p.retireForeignIgnoredEnd(closingTag)
				if !isFormattingElement(node.Name) {
					p.finishFormattingWithin(node, contentEndPos, contentEndLine, contentEndColumn)
				}
				node.TextContent = textContent.String()
				node.ContentStart = contentStartPos
				node.ContentEnd = contentEndPos
				node.EndPos = p.pos
				node.EndLine = p.line
				node.EndColumn = p.col
				if p.listEOFRecoveryBoundary == node {
					p.listEOFRecoveryBoundary = nil
					p.recoverOpenElementsAtEOF = false
				}
				if p.selectDocumentEndRecovery != nil && nodeIsDescendantOrSelf(p.selectDocumentEndRecovery, node) {
					p.selectDocumentEndRecovery = nil
					p.recoverOpenElementsAtEOF = false
				}
				if isFormattingElement(node.Name) {
					p.removeActiveFormatting(node)
				}
				if p.adoptionEOFBoundary == node {
					p.adoptionEOFBoundary = nil
					p.recoverOpenElementsAtEOF = false
				}
				return node, nil
			}
			if isHeadingName(node.Name) && isHeadingName(closingTag) {
				p.finishElementAt(node, textContent.String(), contentStartPos, contentEndPos, contentEndLine, contentEndColumn)
				return node, nil
			}
			// An end tag for script is ignored when no script element is in
			// scope, including while parsing an explicit body or another
			// ordinary ancestor.
			if closingTag == "script" {
				continue
			}
			if closingTag == "p" && p.paragraphInButtonScope(node) == nil {
				if p.headDepth > 0 && !p.headExited {
					continue
				}
				if node.Name == "html" && !p.bodyContentStarted {
					continue
				}
				node.Children = append(node.Children, syntheticParagraph(node))
				continue
			}
			// A table is an ordinary-scope boundary for customizable selects.
			// An end-select token inside that table cannot reach an outer select
			// and is ignored by the in-body generic end-tag algorithm.
			if closingTag == "select" && p.selectBlockedByTable {
				continue
			}
			if p.listItemBlocksGenericAncestorEnd(node, closingTag) {
				continue
			}
			if p.listItemIgnoresAbsentAncestorEnd(node, closingTag) {
				continue
			}
			// A customizable select is an ordinary-scope boundary. End tags for
			// elements outside it are ignored while the select remains open.
			if p.activeSelect != nil && nodeIsDescendantOrSelf(node, p.activeSelect) {
				outside := nearestOpenElementByName(p.activeSelect.Parent, closingTag)
				if outside != nil {
					p.preserveIgnoredTextRange(node, contentEndPos, contentEndLine, contentEndColumn)
					p.recoverOpenElementsAtEOF = true
					p.selectDocumentEndRecovery = p.activeSelect
					continue
				}
			}
			if p.activeParagraph != nil {
				if _, isAncestor := p.activeParagraphAncestors[closingTag]; isAncestor && !isParagraphClosingAncestorEndTag(closingTag) {
					p.recoverOpenElementsAtEOF = true
					continue
				}
			}
			if p.foreignHTMLDepth > 0 {
				// HTML descendants of an SVG integration point use the ordinary
				// in-body generic end-tag algorithm, but the integration owner is a
				// special barrier. An absent or outside matching name is ignored.
				continue
			}
			return nil, fmt.Errorf("expected closing tag </%s>, got </%s> at position %d", node.Name, closingTag, contentEndPos)
		}

		formattingContainer := node
		if pending := p.adoptionPendingEnds[node]; len(pending) > 0 {
			formattingContainer = pending[0]
		}
		if p.documentEndRecovery && node == p.documentHTML && p.documentBody != nil {
			formattingContainer = p.documentBody
		}
		reconstructFormatting := p.nextTokenReconstructsFormatting()
		insertionParent := formattingContainer
		if !p.nextTokenIsProcessingInstruction() {
			insertionParent = p.formattingInsertionParent(formattingContainer, reconstructFormatting)
		}
		child, err := p.parseNode(insertionParent)
		if err != nil {
			return nil, err
		}
		if preformattedRangeStart >= 0 && child != nil {
			if child.Type == types.TextNode && strings.HasPrefix(child.Value, "\n") {
				child.StartPos = preformattedRangeStart
				child.StartLine = preformattedRangeLine
				child.StartColumn = preformattedRangeColumn
			}
			preformattedRangeStart = -1
		}
		if child != nil {
			_, splitCurrent := p.adoptionSplit[node]
			detachedChild := p.hasAdoptionFlag(child, adoptionDetached)
			if insertionParent != node {
				p.appendFormattingChild(node, insertionParent, child)
				if child.Type != types.CommentNode && child.Type != types.ProcessingInstructionNode {
					textContent.WriteString(child.TextContent)
				}
			}
			if child.Type == types.TextNode && !isHTMLWhitespaceString(child.Value) {
				p.bodyContentStarted = true
				if p.headDepth > 0 {
					p.headExited = true
				}
			}
			if detachedChild {
				p.clearAdoptionFlag(child, adoptionDetached)
			} else if insertionParent != node {
				// The child is already attached through reconstructed formatting
				// wrappers; only the ancestor's aggregate text remains to update.
			} else if child.Type == types.TextNode && len(node.Children) > 0 && node.Children[len(node.Children)-1].Type == types.TextNode {
				mergeTextNodes(node.Children[len(node.Children)-1], child)
			} else {
				node.Children = append(node.Children, child)
			}
			fosteredText := ""
			if !detachedChild {
				fosteredText = p.appendDeferredTableFoster(node, child)
				p.appendDeferredListComments(node, child)
				p.appendDeferredAdoption(node, child)
				p.appendDeferredFormSiblings(node, child)
			}
			if insertionParent == node && !detachedChild {
				switch child.Type {
				case types.TextNode:
					textContent.WriteString(child.Value)
				case types.ElementNode:
					// Fostered table content precedes the table in the DOM even
					// though its source token occurred after the table start tag.
					textContent.WriteString(fosteredText)
					textContent.WriteString(child.TextContent)
				}
			}
			if splitCurrent {
				node.TextContent = textContent.String()
				return node, nil
			}
			if p.hasAdoptionFlag(node, adoptionClosed) {
				p.clearAdoptionFlag(node, adoptionClosed)
				p.markTextDirty(node)
				p.refreshNodeTextContent(node)
				return node, nil
			}
			if p.closedForms[node] {
				delete(p.formEOFBoundaries, node)
				p.markTextDirty(node)
				p.refreshNodeTextContent(node)
				return node, nil
			}
		}
		if p.hasAdoptionFlag(node, adoptionReturnAtSplit) {
			p.clearAdoptionFlag(node, adoptionReturnAtSplit)
			delete(p.formattingText, node)
			p.markTextDirty(node)
			p.refreshNodeTextContent(node)
			return node, nil
		}
	}
	p.finishPendingAdoptionAtEOF(node)
	documentBoundaryClosed := (node == p.documentBody && p.documentBodyBoundary.end.pos > 0) ||
		(node == p.documentHTML && (p.documentHTMLBoundary.end.pos > 0 || p.documentBodyBoundary.end.pos > 0))
	formRecovery := false
	for boundary := range p.formEOFBoundaries {
		if nodeIsDescendantOrSelf(node, boundary) {
			formRecovery = true
			break
		}
	}
	foreignRecovery := p.foreignHTMLDepth > 0 || (p.foreignBreakoutBoundary != nil && nodeIsDescendantOrSelf(node, p.foreignBreakoutBoundary))
	// parseElement is only entered through the document-mode driver once a
	// source-backed or synthetic HTML element exists. Therefore every open
	// ordinary frame belongs to this document, and EOF recovery is O(1). A
	// recursive nodeWithin check here makes a depth-N EOF path quadratic.
	explicitDocumentEOF := p.explicitDocumentSkeleton && p.documentHTML != nil
	if p.recoverOpenElementsAtEOF || explicitDocumentEOF || foreignRecovery || documentBoundaryClosed || formRecovery || p.openForms[node] || p.paragraphInButtonScope(node) != nil || p.hasActiveListItem() || p.hasActiveSelectTree() || len(p.activeFormatting) > 0 {
		if len(p.activeFormatting) > 0 {
			p.recoverOpenElementsAtEOF = true
		}
		if p.hasActiveSelectTree() {
			p.recoverOpenElementsAtEOF = true
		}
		if p.hasActiveListItem() {
			p.recoverOpenElementsAtEOF = true
		}
		p.finishImplicitElement(node, textContent.String(), contentStartPos)
		return node, nil
	}

	return nil, fmt.Errorf("unterminated element <%s> at position %d", node.Name, startPos)
}

func (p *HTMLParser) hasActiveSelectTree() bool {
	return p.activeSelect != nil || p.activeOptionEnd != nil || p.activeOptgroupEnd != nil
}

func (p *HTMLParser) enterElementContext(node *types.Node) func() {
	p.pushOpenElement(node)
	formattingMarker := isFormattingMarkerElement(node.Name)
	previousFormattingFamilies := p.formattingFamilies
	previousFormattingByName := p.formattingByName
	previousFormattingMarkerStart := p.formattingMarkerStart
	if formattingMarker {
		p.activeFormatting = append(p.activeFormatting, &formattingEntry{marker: true})
		p.formattingMarkerStart = len(p.activeFormatting)
		p.formattingFamilies = make(map[string][]*formattingEntry)
		p.formattingByName = make(map[string][]*formattingEntry)
	}
	previousParagraph := p.activeParagraph
	previousAncestors := p.activeParagraphAncestors
	previousButton := p.activeButton
	previousHeadDepth := p.headDepth
	previousLIStart := p.activeLIStart
	previousDefinitionStart := p.activeDefinitionStart
	previousLIEnd := p.activeLIEnd
	previousDDEnd := p.activeDDEnd
	previousDTEnd := p.activeDTEnd
	previousSelect := p.activeSelect
	previousOptionStart := p.activeOptionStart
	previousOptgroupStart := p.activeOptgroupStart
	previousOptionEnd := p.activeOptionEnd
	previousOptgroupEnd := p.activeOptgroupEnd
	previousSelectImpliedEnd := p.activeSelectImpliedEnd
	previousSelectOptionImplied := p.activeSelectOptionImplied
	previousSelectNames := p.activeSelectNames
	previousSelectAllNames := p.activeSelectAllNames
	previousSelectBlockedByTable := p.selectBlockedByTable
	var previousSelectNameNode *types.Node
	previousSelectNamePresent := false
	var previousSelectAllNameNode *types.Node
	previousSelectAllNamePresent := false
	selectGenericScopeBoundary := false

	if node.Name == "head" {
		p.headDepth++
	} else if node.Name == "body" {
		p.documentBody = node
		p.headExited = true
		p.bodyContentStarted = true
	} else if node.Name != "html" && p.headDepth > 0 && !p.headExited && !isHeadOnlyElement(node.Name) && (node.Name != "noscript" || node.Parent != p.documentHead) {
		p.headExited = true
		p.bodyContentStarted = true
	} else if node.Name != "html" && p.headDepth == 0 && !isHeadOnlyElement(node.Name) {
		p.bodyContentStarted = true
	}
	if node.Name == "html" {
		p.documentHTML = node
	}

	if isListStartScanStopper(node.Name) {
		p.activeLIStart = nil
		p.activeDefinitionStart = nil
	}
	if isOrdinaryScopeBoundary(node.Name) {
		p.activeDDEnd = nil
		p.activeDTEnd = nil
	}
	if isListItemScopeBoundary(node.Name) {
		p.activeLIEnd = nil
	}
	// Customizable selects otherwise use the normal in-body rules. Keep two
	// views of option/group state: start-tag implied-end scans stop at the
	// first non-implied element, while matching end tags stop at an HTML
	// special element. Select itself is an ordinary-scope boundary.
	if !isImpliedEndTagName(node.Name) && node.Name != "select" {
		p.activeOptionStart = nil
		p.activeOptgroupStart = nil
		p.activeSelectImpliedEnd = nil
		p.activeSelectOptionImplied = nil
	}
	if isHTMLSpecialElement(node.Name) && node.Name != "option" && node.Name != "optgroup" && node.Name != "search" {
		p.activeOptionEnd = nil
		p.activeOptgroupEnd = nil
	}
	if isOrdinaryScopeBoundary(node.Name) {
		p.activeSelect = nil
	}
	if node.Name == "table" && previousSelect != nil {
		p.selectBlockedByTable = true
	}
	if p.activeSelect != nil && isImpliedEndTagName(node.Name) {
		p.activeSelectImpliedEnd = node
		if node.Name != "optgroup" {
			p.activeSelectOptionImplied = node
		}
	}
	switch node.Name {
	case "select":
		p.activeSelect = node
		p.activeOptionStart = nil
		p.activeOptgroupStart = nil
		p.activeOptionEnd = nil
		p.activeOptgroupEnd = nil
		p.activeSelectImpliedEnd = nil
		p.activeSelectOptionImplied = nil
		p.activeSelectNames = map[string]*types.Node{"select": node}
		p.activeSelectAllNames = map[string]*types.Node{"select": node}
	case "option":
		p.activeOptionStart = node
		p.activeOptionEnd = node
	case "optgroup":
		p.activeOptgroupStart = node
		p.activeOptgroupEnd = node
	}
	if p.activeSelect != nil && node.Name != "select" {
		selectGenericScopeBoundary = isHTMLSpecialElement(node.Name) && node.Name != "search"
		if selectGenericScopeBoundary {
			p.activeSelectNames = make(map[string]*types.Node)
		}
		if p.activeSelectNames == nil {
			p.activeSelectNames = make(map[string]*types.Node)
		}
		previousSelectNameNode, previousSelectNamePresent = p.activeSelectNames[node.Name]
		p.activeSelectNames[node.Name] = node
		if p.activeSelectAllNames == nil {
			p.activeSelectAllNames = make(map[string]*types.Node)
		}
		previousSelectAllNameNode, previousSelectAllNamePresent = p.activeSelectAllNames[node.Name]
		p.activeSelectAllNames[node.Name] = node
	}
	switch node.Name {
	case "li":
		p.activeLIStart = node
		p.activeLIEnd = node
	case "dd":
		p.activeDefinitionStart = node
		p.activeDDEnd = node
	case "dt":
		p.activeDefinitionStart = node
		p.activeDTEnd = node
	}

	if isParagraphButtonScopeBoundary(node.Name) {
		p.activeParagraph = nil
		p.activeParagraphAncestors = nil
		p.activeButton = nil
	}
	if node.Name == "button" && node.NamespaceURI == htmlElementNamespace {
		p.activeButton = node
	}
	if node.Name == "p" {
		p.activeParagraph = node
		p.activeParagraphAncestors = make(map[string]struct{})
		for ancestor := node.Parent; ancestor != nil; ancestor = ancestor.Parent {
			p.activeParagraphAncestors[ancestor.Name] = struct{}{}
			if isParagraphAncestorScopeBoundary(ancestor.Name) {
				break
			}
		}
	}

	return func() {
		defer p.popOpenElement(node)
		if formattingMarker {
			p.clearActiveFormattingToMarker()
			p.formattingFamilies = previousFormattingFamilies
			p.formattingByName = previousFormattingByName
			p.formattingMarkerStart = previousFormattingMarkerStart
		}
		if node.Name == "html" {
			p.bodyContentStarted = true
		}
		p.activeParagraph = previousParagraph
		p.activeParagraphAncestors = previousAncestors
		p.activeButton = previousButton
		p.headDepth = previousHeadDepth
		p.activeLIStart = previousLIStart
		p.activeDefinitionStart = previousDefinitionStart
		p.activeLIEnd = previousLIEnd
		p.activeDDEnd = previousDDEnd
		p.activeDTEnd = previousDTEnd
		p.activeSelect = previousSelect
		p.activeOptionStart = previousOptionStart
		p.activeOptgroupStart = previousOptgroupStart
		p.activeOptionEnd = previousOptionEnd
		p.activeOptgroupEnd = previousOptgroupEnd
		p.activeSelectImpliedEnd = previousSelectImpliedEnd
		p.activeSelectOptionImplied = previousSelectOptionImplied
		if p.activeSelectNames != nil && node.Name != "select" && !selectGenericScopeBoundary {
			if previousSelectNamePresent {
				p.activeSelectNames[node.Name] = previousSelectNameNode
			} else {
				delete(p.activeSelectNames, node.Name)
			}
		}
		if p.activeSelectAllNames != nil && node.Name != "select" {
			if previousSelectAllNamePresent {
				p.activeSelectAllNames[node.Name] = previousSelectAllNameNode
			} else {
				delete(p.activeSelectAllNames, node.Name)
			}
		}
		p.activeSelectNames = previousSelectNames
		p.activeSelectAllNames = previousSelectAllNames
		p.selectBlockedByTable = previousSelectBlockedByTable
	}
}
