// Template insertion modes and document-location finalization.
package utils

import (
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func (p *HTMLParser) currentTemplate() *types.Node {
	if len(p.activeTemplates) == 0 {
		return nil
	}
	return p.activeTemplates[len(p.activeTemplates)-1]
}

// currentTemplateEndCloses reports whether the next token closes the template
// whose inert fragment owns the current recursive frame. The token remains
// unconsumed so frames can unwind to the template driver one at a time.
func (p *HTMLParser) currentTemplateEndCloses(node *types.Node) bool {
	owner := p.currentTemplate()
	if owner == nil || p.templateOwnerForNode(node) != owner {
		return false
	}
	name, closing, ok := p.peekTagName()
	return ok && closing && name == "template" && p.currentTagTokenWillEmit()
}

// currentTemplateTableTransitionWillEmit reports a structural start token
// that must be reprocessed by the persistent table-derived insertion mode of
// the innermost template. Generic descendant frames must unwind before the
// template driver can apply that mode; otherwise elements such as div/select
// incorrectly swallow rows and cells.
func (p *HTMLParser) currentTemplateTableTransitionWillEmit(node *types.Node) bool {
	if len(p.templateInsertionModes) == 0 || p.templateOwnerForNode(node) != p.currentTemplate() {
		return false
	}
	mode := p.templateInsertionModes[len(p.templateInsertionModes)-1]
	if mode != templateModeTable && mode != templateModeTableBody && mode != templateModeRow {
		return false
	}
	name, closing, ok := p.peekTagName()
	return ok && !closing && isTableStructuralName(name) && p.currentTagTokenWillEmit()
}

func (p *HTMLParser) templateOwnerForNode(node *types.Node) *types.Node {
	if node == nil {
		return nil
	}
	if owner, known := p.templateOwner[node]; known {
		return owner
	}
	path := make([]*types.Node, 0, 4)
	var owner *types.Node
	for ancestor := node; ancestor != nil; ancestor = ancestor.Parent {
		if cached, known := p.templateOwner[ancestor]; known {
			owner = cached
			break
		}
		path = append(path, ancestor)
	}
	for _, visited := range path {
		p.templateOwner[visited] = owner
	}
	return owner
}

func newTemplateContent(host *types.Node) *types.Node {
	return &types.Node{
		Type:     types.DocumentFragmentNode,
		Name:     "#document-fragment",
		Children: []*types.Node{},
	}
}

func (p *HTMLParser) appendTemplateChild(fragment, child *types.Node) {
	appendChildIncremental(fragment, child)
}

type templateInsertionMode uint8

const (
	templateModeInitial templateInsertionMode = iota
	templateModeBody
	templateModeTable
	templateModeColumnGroup
	templateModeTableBody
	templateModeRow
)

// parseTemplateElement parses into an inert DocumentFragment. The template is
// a leaf in the document tree; recursive parser scope is tracked separately in
// templateOwner because DOM Parent intentionally stops at the fragment.
func (p *HTMLParser) parseTemplateElement(template *types.Node, contentStart int) (*types.Node, error) {
	fragment := newTemplateContent(template)
	template.TemplateContent = fragment
	p.templateOwner[fragment] = template
	p.activeTemplates = append(p.activeTemplates, template)
	p.templateInsertionModes = append(p.templateInsertionModes, templateModeInitial)
	defer func() {
		delete(p.templateOwner, fragment)
		p.activeTemplates = p.activeTemplates[:len(p.activeTemplates)-1]
		p.templateInsertionModes = p.templateInsertionModes[:len(p.templateInsertionModes)-1]
	}()

	// The form element pointer is scoped outside template contents. Forms in the
	// inert subtree may use ordinary pointer recovery without replacing or
	// clearing the pointer belonging to the surrounding document tree.
	outerForm := p.activeForm
	p.activeForm = nil
	defer func() { p.activeForm = outerForm }()
	mode := templateModeInitial
	setMode := func(next templateInsertionMode) {
		mode = next
		p.templateInsertionModes[len(p.templateInsertionModes)-1] = next
	}
	lastBoundary := template.StartPos
	lastBoundaryLine, lastBoundaryColumn := template.StartLine, template.StartColumn
	updateIgnoredStartBoundary := func() {
		lastBoundary = p.pos
		lastBoundaryLine, lastBoundaryColumn = p.line, p.col
	}

	for p.pos < len(p.content) {
		if p.templateForeignTransition {
			p.templateForeignTransition = false
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			child, err := p.parseNode(fragment)
			if err != nil {
				return nil, err
			}
			p.appendTemplateChild(fragment, child)
			continue
		}
		if tag && closing && name == "template" {
			contentEnd := p.pos
			if _, emitted, err := p.parseClosingTag(); err != nil {
				return nil, err
			} else if !emitted {
				p.extendPreviousTextRange(fragment, contentEnd)
				break
			}
			template.ContentStart = contentStart
			template.ContentEnd = contentEnd
			template.EndPos = p.pos
			template.EndLine = p.line
			template.EndColumn = p.col
			p.refreshOpenFormattingText()
			p.markTextDirty(fragment)
			p.refreshNodeTextContent(fragment)
			return template, nil
		}
		if tag && closing && name == "table" {
			// A table outside the template is not in the template's table scope.
			// The in-template rule ignores this end tag.
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if mode == templateModeInitial && tag && !closing && (name == "html" || name == "head" || name == "body") && p.currentTagTokenWillEmit() {
			setMode(templateModeBody)
			continue
		}
		if tag && !closing && p.currentTagTokenWillEmit() {
			switch name {
			case "html":
				updateIgnoredStartBoundary()
				p.consumeIgnoredStartTag()
				continue
			case "head", "body":
				updateIgnoredStartBoundary()
				p.consumeIgnoredStartTag()
				continue
			}
		}
		if tag && closing && name != "p" && name != "br" {
			// With every content descendant already unwound, an end tag seen by
			// the template driver has no matching element in template scope.
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if tag && !closing && (name == "frame" || name == "frameset") && p.currentTagTokenWillEmit() {
			updateIgnoredStartBoundary()
			p.consumeIgnoredStartTag()
			continue
		}

		// The stack of template insertion modes is persistent. Once a token
		// switches away from "in template", later tokens are processed using that
		// mode until a table transition changes it again; dispatching every root
		// token independently produces the wrong wrappers and ignored-token rules.
		if mode == templateModeInitial {
			if !tag || p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() || p.nextTokenIsIgnoredDoctype() || (tag && !closing && isHeadOnlyElement(name) && name != "head") {
				// Character/comment/doctype and in-head tokens do not replace the
				// current template insertion mode.
			} else if closing {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				continue
			} else {
				switch name {
				case "caption", "colgroup", "tbody", "tfoot", "thead":
					setMode(templateModeTable)
				case "col":
					setMode(templateModeColumnGroup)
				case "tr":
					setMode(templateModeTableBody)
				case "td", "th":
					setMode(templateModeRow)
				default:
					setMode(templateModeBody)
				}
				continue
			}
		}

		// A form start is processed using the in-body rule only after the
		// template insertion-mode stack has switched to in body. In table and
		// table-derived template modes, the in-table rule ignores a form start
		// while a template is open. Form end tags still use the ordinary pointer
		// recovery rule in every mode.
		if name == "form" && (closing || mode == templateModeBody) {
			if handled, err := p.handleFormToken(fragment, false); handled || err != nil {
				if err != nil {
					return nil, err
				}
				continue
			}
		} else if name == "form" && tag && !closing && p.currentTagTokenWillEmit() {
			updateIgnoredStartBoundary()
			p.consumeIgnoredStartTag()
			continue
		}

		if mode == templateModeColumnGroup && (!tag || closing || (name != "col" && name != "template")) {
			switch {
			case p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction():
				child, err := p.parseNode(fragment)
				if err != nil {
					return nil, err
				}
				p.appendTemplateChild(fragment, child)
			case p.nextTokenIsIgnoredDoctype():
				if _, err := p.parseNode(fragment); err != nil {
					return nil, err
				}
			case tag && !closing:
				updateIgnoredStartBoundary()
				p.consumeIgnoredStartTag()
			case tag && closing:
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
			default:
				if whitespace := p.parseTemplateColumnWhitespace(fragment); whitespace != nil {
					p.appendTemplateChild(fragment, whitespace)
				} else if _, err := p.parseTextNode(fragment, p.pos, p.line, p.col); err != nil {
					return nil, err
				}
			}
			continue
		}

		if tag && !closing && ((mode == templateModeTableBody && (name == "caption" || name == "col" || name == "colgroup" || name == "tbody" || name == "tfoot" || name == "thead")) ||
			(mode == templateModeRow && (name == "caption" || name == "col" || name == "colgroup" || name == "tbody" || name == "tfoot" || name == "thead" || name == "tr"))) {
			updateIgnoredStartBoundary()
			p.consumeIgnoredStartTag()
			continue
		}
		if tag && !closing && name == "table" && mode != templateModeInitial && mode != templateModeBody {
			updateIgnoredStartBoundary()
			p.consumeIgnoredStartTag()
			continue
		}

		if mode == templateModeBody && tag && ((closing && isTableStructuralName(name)) || (!closing && isTableStructureStart(name))) {
			if closing {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
			} else {
				updateIgnoredStartBoundary()
				p.consumeIgnoredStartTag()
			}
			continue
		}
		var child *types.Node
		var err error
		ordinary := false
		if tag && !closing {
			switch mode {
			case templateModeTable:
				switch name {
				case "caption":
					child, err = p.parseTableLooseContainer(fragment, "caption", map[string]bool{"caption": true, "col": true, "colgroup": true, "tbody": true, "tfoot": true, "thead": true, "tr": true})
				case "colgroup":
					child, err = p.parseTableColgroup(fragment, false)
				case "col":
					child, err = p.parseTableColgroup(fragment, true)
				case "tbody", "thead", "tfoot":
					child, err = p.parseTableSection(fragment, false)
				case "tr", "td", "th":
					child, err = p.parseTableSection(fragment, true)
				default:
					ordinary = true
				}
			case templateModeColumnGroup:
				child, err = p.parseNode(fragment)
			case templateModeTableBody:
				switch name {
				case "tr":
					child, err = p.parseTableRow(fragment, false)
				case "td", "th":
					child, err = p.parseTableRow(fragment, true)
				default:
					ordinary = true
				}
			case templateModeRow:
				switch name {
				case "td", "th":
					child, err = p.parseTableCell(fragment)
				default:
					ordinary = true
				}
			default:
				ordinary = true
			}
		} else {
			ordinary = true
		}
		if ordinary {
			insertionParent := p.formattingInsertionParent(fragment, p.nextTokenReconstructsFormatting())
			child, err = p.parseNode(insertionParent)
			if err == nil && child != nil && insertionParent != fragment {
				p.appendFormattingChild(fragment, insertionParent, child)
				child = nil
			}
		}
		if err != nil {
			return nil, err
		}
		p.appendTemplateChild(fragment, child)
		if boundary, line, column, ok := p.templateElementBoundary(child); ok {
			lastBoundary, lastBoundaryLine, lastBoundaryColumn = boundary, line, column
		}
	}

	p.refreshOpenFormattingText()
	p.markTextDirty(fragment)
	p.refreshNodeTextContent(fragment)
	// parse5 ends an unclosed template at the current token boundary. Preserve
	// the normal source-backed host location while keeping the fragment itself
	// locationless.
	if boundary, ok := p.templateEOFBoundary[template]; ok {
		template.ContentStart = contentStart
		template.ContentEnd = boundary.pos
		template.EndPos = boundary.pos
		template.EndLine = boundary.line
		template.EndColumn = boundary.column
	} else {
		template.ContentStart = contentStart
		template.ContentEnd = lastBoundary
		template.EndPos = lastBoundary
		template.EndLine = lastBoundaryLine
		template.EndColumn = lastBoundaryColumn
	}
	return template, nil
}

func (p *HTMLParser) templateElementBoundary(node *types.Node) (int, int, int, bool) {
	if node == nil || node.Type != types.ElementNode {
		return 0, 0, 0, false
	}
	if node.StartLine != 0 {
		boundary := node.ContentEnd
		if p.isSelfClosingTag(node.Name) || node.ContentEnd >= node.EndPos {
			boundary = node.StartPos
		}
		line, column := p.coordinatesFromNodeStart(node, boundary)
		return boundary, line, column, true
	}
	for index := len(node.Children) - 1; index >= 0; index-- {
		if boundary, line, column, ok := p.templateElementBoundary(node.Children[index]); ok {
			return boundary, line, column, true
		}
	}
	return 0, 0, 0, false
}

func (p *HTMLParser) parseTemplateColumnWhitespace(parent *types.Node) *types.Node {
	start, line, column := p.pos, p.line, p.col
	var value strings.Builder
	for p.pos < len(p.content) {
		if p.peek() == '<' && p.startsMarkupToken() {
			break
		}
		preview := *p
		if decoded, consumed := preview.consumeNumericCharacterReference(); consumed {
			if !isHTMLWhitespaceString(string(decoded)) {
				break
			}
			decoded, _ = p.consumeNumericCharacterReference()
			value.WriteRune(decoded)
			continue
		}
		preview = *p
		if decoded, consumed := preview.consumeNamedCharacterReference(false); consumed {
			if !isHTMLWhitespaceString(decoded) {
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
		return nil
	}
	text := value.String()
	return &types.Node{
		Type: types.TextNode, Name: "#text", Value: text, TextContent: text, Parent: parent,
		StartPos: start, EndPos: p.pos, StartLine: line, StartColumn: column,
		EndLine: p.line, EndColumn: p.col,
	}
}

// templatePseudoTableAnythingElse identifies a token that a synthetic
// table-body/row frame cannot own because there is no real table element in
// scope. Return it to the template driver so adjusted foster insertion and
// active-formatting reconstruction happen at the inert fragment root.
func (p *HTMLParser) templatePseudoTableAnythingElse(node *types.Node) bool {
	if p.activeTable != nil || p.templateOwnerForNode(node) == nil || p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() || p.nextTokenIsIgnoredDoctype() {
		return false
	}
	if name, closing, tag := p.peekTagName(); tag {
		return closing || (name != "template" && name != "style" && name != "script" && !isTableStructuralName(name))
	}
	preview := *p
	parent := &types.Node{Type: types.ElementNode, Name: "#preview", Children: []*types.Node{}}
	text, err := preview.parseTextNode(parent, preview.pos, preview.line, preview.col)
	return err == nil && text != nil && !isHTMLWhitespaceString(text.Value)
}

func (p *HTMLParser) consumeTemplatePseudoAbsentEnd(node *types.Node) (bool, error) {
	if p.activeTable != nil || p.templateOwnerForNode(node) == nil {
		return false, nil
	}
	name, closing, tag := p.peekTagName()
	if !tag || !closing || name == "template" || name == "p" || name == "br" || name == "form" || name == "script" || isTableStructuralName(name) || isCoreAdoptionSubject(name) || !p.currentTagTokenWillEmit() {
		return false, nil
	}
	_, _, err := p.parseClosingTag()
	return true, err
}

func (p *HTMLParser) coordinatesFromNodeStart(node *types.Node, target int) (int, int) {
	preview := HTMLParser{
		htmlParserState: htmlParserState{inputState: inputState{
			content: p.content,
			pos:     node.StartPos,
			line:    node.StartLine,
			col:     node.StartColumn,
		}},
	}
	for preview.pos < target {
		_, size := preview.peekHTMLRune()
		if size == 0 || preview.pos+size > target {
			break
		}
		preview.advanceRune(size)
	}
	return preview.line, preview.col
}

func (p *HTMLParser) finishImplicitElement(node *types.Node, textContent string, contentStart int) {
	node.TextContent = textContent
	if node.StartLine == 0 {
		return
	}
	if owner := p.templateOwnerForNode(node); owner != nil && p.pos == len(p.content) {
		boundary, ok := p.templateEOFBoundary[owner]
		if !ok {
			boundary = sourcePoint{pos: node.StartPos, line: node.StartLine, column: node.StartColumn}
			p.templateEOFBoundary[owner] = boundary
		}
		node.ContentStart = contentStart
		node.ContentEnd = boundary.pos
		node.EndPos = boundary.pos
		node.EndLine = boundary.line
		node.EndColumn = boundary.column
		return
	}
	if node == p.documentBody && p.documentBodyBoundary.end.pos > 0 {
		node.ContentStart = contentStart
		node.ContentEnd = p.documentBodyBoundary.contentEnd
		node.EndPos = p.documentBodyBoundary.end.pos
		node.EndLine = p.documentBodyBoundary.end.line
		node.EndColumn = p.documentBodyBoundary.end.column
		return
	}
	if node == p.documentHTML && p.documentHTMLBoundary.end.pos > 0 {
		node.ContentStart = contentStart
		node.ContentEnd = p.documentHTMLBoundary.contentEnd
		node.EndPos = p.documentHTMLBoundary.end.pos
		node.EndLine = p.documentHTMLBoundary.end.line
		node.EndColumn = p.documentHTMLBoundary.end.column
		return
	}
	node.ContentStart = contentStart
	node.ContentEnd = p.pos
	node.EndPos = p.pos
	node.EndLine = p.line
	node.EndColumn = p.col
}

func (p *HTMLParser) finishElementAt(node *types.Node, textContent string, contentStart, end, line, column int) {
	node.TextContent = textContent
	if node.StartLine == 0 {
		return
	}
	node.ContentStart = contentStart
	node.ContentEnd = end
	node.EndPos = end
	node.EndLine = line
	node.EndColumn = column
}
