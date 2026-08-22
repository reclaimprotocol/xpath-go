// Table insertion modes, foster parenting, and ordinary text emission live
// together because table recovery reprocesses tokenizer output directly.
package utils

import (
	"fmt"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func (p *HTMLParser) currentTagTokenWillEmit() (emits bool) {
	if p.pos >= len(p.content) || p.peek() != '<' {
		return false
	}
	p.peekTagName()
	if p.tagPreview.emissionKnown {
		return p.tagPreview.emits
	}
	defer func() {
		p.tagPreview.emits = emits
		p.tagPreview.emissionKnown = true
	}()
	position := p.pos + 1
	if position < len(p.content) && p.content[position] == '/' {
		position++
	}
	for position < len(p.content) && !isWhitespace(p.content[position]) && p.content[position] != '/' && p.content[position] != '>' {
		position++
	}

	state := beforeAttributeNameState
	for position < len(p.content) {
		c := p.content[position]
		switch state {
		case beforeAttributeNameState:
			switch {
			case isWhitespace(c):
				position++
			case c == '/':
				position++
				state = selfClosingStartTagState
			case c == '>':
				return true
			default:
				if c == '=' {
					position++
				}
				state = attributeNameState
			}
		case attributeNameState:
			switch {
			case isWhitespace(c):
				position++
				state = afterAttributeNameState
			case c == '/' || c == '>':
				state = afterAttributeNameState
			case c == '=':
				position++
				state = beforeAttributeValueState
			default:
				position++
			}
		case afterAttributeNameState:
			switch {
			case isWhitespace(c):
				position++
			case c == '/':
				position++
				state = selfClosingStartTagState
			case c == '=':
				position++
				state = beforeAttributeValueState
			case c == '>':
				return true
			default:
				state = attributeNameState
			}
		case beforeAttributeValueState:
			switch {
			case isWhitespace(c):
				position++
			case c == '"':
				position++
				state = attributeValueDoubleQuotedState
			case c == '\'':
				position++
				state = attributeValueSingleQuotedState
			case c == '>':
				return true
			default:
				state = attributeValueUnquotedState
			}
		case attributeValueDoubleQuotedState:
			position++
			if c == '"' {
				state = afterAttributeValueQuotedState
			}
		case attributeValueSingleQuotedState:
			position++
			if c == '\'' {
				state = afterAttributeValueQuotedState
			}
		case attributeValueUnquotedState:
			switch {
			case isWhitespace(c):
				position++
				state = beforeAttributeNameState
			case c == '>':
				return true
			default:
				position++
			}
		case afterAttributeValueQuotedState:
			switch {
			case isWhitespace(c):
				position++
				state = beforeAttributeNameState
			case c == '/':
				position++
				state = selfClosingStartTagState
			case c == '>':
				return true
			default:
				state = beforeAttributeNameState
			}
		case selfClosingStartTagState:
			if c == '>' {
				return true
			}
			state = beforeAttributeNameState
		}
	}
	return false
}

func (p *HTMLParser) parseTableElement(table *types.Node, contentStart int) (*types.Node, error) {
	previousTable := p.activeTable
	p.activeTable = table
	defer func() { p.activeTable = previousTable }()
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(table) {
			p.finishTableImplicit(table, contentStart)
			return table, nil
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			if _, err := p.consumeDiscardedTableToken(table); err != nil {
				return nil, err
			}
			continue
		}
		if tag && closing && name == "table" {
			if err := p.closeTableNode(table); err != nil {
				return nil, err
			}
			table.ContentStart = contentStart
			return table, nil
		}
		if tag && closing && (name == "body" || name == "html") {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if tag && !closing {
			switch name {
			case "template":
				child, err := p.parseNode(table)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(table, child)
				continue
			case "table":
				p.finishTableImplicit(table, contentStart)
				return table, nil
			case "caption":
				caption, err := p.parseTableLooseContainer(table, "caption", map[string]bool{"caption": true, "col": true, "colgroup": true, "tbody": true, "tfoot": true, "thead": true, "tr": true})
				if err != nil {
					return nil, err
				}
				p.appendTableChild(table, caption)
				continue
			case "col":
				colgroup, err := p.parseTableColgroup(table, true)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(table, colgroup)
				continue
			case "colgroup":
				colgroup, err := p.parseTableColgroup(table, false)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(table, colgroup)
				continue
			case "tbody", "thead", "tfoot":
				section, err := p.parseTableSection(table, false)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(table, section)
				continue
			case "tr", "td", "th":
				section, err := p.parseTableSection(table, true)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(table, section)
				continue
			}
		}
		if tag && !closing && (name == "style" || name == "script") {
			child, err := p.parseNode(table)
			if err != nil {
				return nil, err
			}
			p.appendTableChild(table, child)
			continue
		}
		if tag && closing && isTableStructuralName(name) {
			_, _, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			continue
		}
		child, err := p.parseTableFosterNode(table)
		if err != nil {
			return nil, err
		}
		if child == nil {
			continue
		}
		if child.Type == types.CommentNode || child.Type == types.ProcessingInstructionNode || (child.Type == types.TextNode && isHTMLWhitespaceString(child.Value)) {
			p.appendTableChild(table, child)
		} else {
			p.fosterTableChild(table, child)
		}
	}
	p.recoverOpenElementsAtEOF = true
	p.finishTableImplicit(table, contentStart)
	return table, nil
}

func (p *HTMLParser) parseTableSection(table *types.Node, synthetic bool) (*types.Node, error) {
	var section *types.Node
	if synthetic {
		section = syntheticTableNode("tbody", table)
	} else {
		var err error
		section, err = p.parseTableOpenElement(table)
		if err != nil {
			return nil, err
		}
	}
	if err := p.parseRowsIntoSection(section); err != nil {
		return nil, err
	}
	return section, nil
}

func (p *HTMLParser) parseRowsIntoSection(section *types.Node) error {
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(section) {
			p.finishTableImplicit(section, section.ContentStart)
			return nil
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			if _, err := p.consumeDiscardedTableToken(section); err != nil {
				return err
			}
			continue
		}
		if tag {
			if closing {
				if name == section.Name {
					return p.closeTableNode(section)
				}
				if name == "table" || name == "tbody" || name == "thead" || name == "tfoot" {
					if name == "table" || name == section.Name {
						p.finishTableImplicit(section, section.ContentStart)
						return nil
					}
					_, _, err := p.parseClosingTag()
					if err != nil {
						return err
					}
					continue
				}
				if name == "body" || name == "caption" || name == "col" || name == "colgroup" || name == "html" || name == "tr" || name == "td" || name == "th" {
					_, _, err := p.parseClosingTag()
					if err != nil {
						return err
					}
					continue
				}
			} else {
				switch name {
				case "template":
					child, err := p.parseNode(section)
					if err != nil {
						return err
					}
					p.appendTableChild(section, child)
					continue
				case "tr":
					row, err := p.parseTableRow(section, false)
					if err != nil {
						return err
					}
					p.appendTableChild(section, row)
					continue
				case "td", "th":
					row, err := p.parseTableRow(section, true)
					if err != nil {
						return err
					}
					p.appendTableChild(section, row)
					continue
				case "tbody", "thead", "tfoot", "caption", "col", "colgroup", "table":
					p.finishTableImplicit(section, section.ContentStart)
					return nil
				}
			}
		}
		if consumed, err := p.consumeTemplatePseudoAbsentEnd(section); consumed || err != nil {
			if err != nil {
				return err
			}
			continue
		}
		if p.templatePseudoTableAnythingElse(section) {
			p.finishTableImplicit(section, section.ContentStart)
			return nil
		}
		child, err := p.parseTableFosterNodeInMode(section, section.Name, false)
		if err != nil {
			return err
		}
		if child != nil {
			if child.Type == types.CommentNode || child.Type == types.ProcessingInstructionNode || (child.Type == types.TextNode && isHTMLWhitespaceString(child.Value)) {
				p.appendTableChild(section, child)
			} else {
				p.fosterTableChild(p.activeTable, child)
			}
		}
	}
	p.recoverOpenElementsAtEOF = true
	p.finishTableImplicit(section, section.ContentStart)
	return nil
}

func (p *HTMLParser) parseTableRow(section *types.Node, synthetic bool) (*types.Node, error) {
	var row *types.Node
	if synthetic {
		row = syntheticTableNode("tr", section)
	} else {
		var err error
		row, err = p.parseTableOpenElement(section)
		if err != nil {
			return nil, err
		}
	}
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(row) {
			p.finishTableImplicit(row, row.ContentStart)
			return row, nil
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			if _, err := p.consumeDiscardedTableToken(row); err != nil {
				return nil, err
			}
			continue
		}
		if tag {
			if !closing && name == "template" {
				child, err := p.parseNode(row)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(row, child)
				continue
			}
			if closing && name == "tr" {
				return row, p.closeTableNode(row)
			}
			if closing && (name == "body" || name == "caption" || name == "col" || name == "colgroup" || name == "html" || name == "td" || name == "th" || ((name == "tbody" || name == "thead" || name == "tfoot") && name != section.Name)) {
				_, _, err := p.parseClosingTag()
				if err != nil {
					return nil, err
				}
				continue
			}
			if (closing && (name == "table" || name == section.Name)) || (!closing && (name == "tr" || name == "tbody" || name == "thead" || name == "tfoot" || name == "caption" || name == "col" || name == "colgroup" || name == "table")) {
				p.finishTableImplicit(row, row.ContentStart)
				return row, nil
			}
			if !closing && (name == "td" || name == "th") {
				cell, err := p.parseTableCell(row)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(row, cell)
				continue
			}
		}
		if consumed, err := p.consumeTemplatePseudoAbsentEnd(row); consumed || err != nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		if p.templatePseudoTableAnythingElse(row) {
			p.finishTableImplicit(row, row.ContentStart)
			return row, nil
		}
		sectionName := ""
		if section != nil {
			sectionName = section.Name
		}
		child, err := p.parseTableFosterNodeInMode(row, sectionName, true)
		if err != nil {
			return nil, err
		}
		if child != nil {
			if child.Type == types.CommentNode || child.Type == types.ProcessingInstructionNode || (child.Type == types.TextNode && isHTMLWhitespaceString(child.Value)) {
				p.appendTableChild(row, child)
			} else {
				p.fosterTableChild(p.activeTable, child)
			}
		}
	}
	p.recoverOpenElementsAtEOF = true
	p.finishTableImplicit(row, row.ContentStart)
	return row, nil
}

func (p *HTMLParser) parseTableCell(row *types.Node) (*types.Node, error) {
	cell, err := p.parseTableOpenElement(row)
	if err != nil {
		return nil, err
	}
	previousFormattingFamilies := p.formattingFamilies
	previousFormattingByName := p.formattingByName
	previousFormattingMarkerStart := p.formattingMarkerStart
	p.activeFormatting = append(p.activeFormatting, &formattingEntry{marker: true})
	p.formattingMarkerStart = len(p.activeFormatting)
	p.formattingFamilies = make(map[string][]*formattingEntry)
	p.formattingByName = make(map[string][]*formattingEntry)
	defer func() {
		p.clearActiveFormattingToMarker()
		p.formattingFamilies = previousFormattingFamilies
		p.formattingByName = previousFormattingByName
		p.formattingMarkerStart = previousFormattingMarkerStart
	}()
	defer p.closeFormattingOnElementReturn(cell)
	sectionName := ""
	if row.Parent != nil {
		sectionName = row.Parent.Name
	}
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(cell) {
			p.finishTableImplicit(cell, cell.ContentStart)
			return cell, nil
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			if _, err := p.consumeDiscardedTableToken(cell); err != nil {
				return nil, err
			}
			continue
		}
		if consumed, consumeErr := p.consumeForeignIgnoredEnd(cell); consumed || consumeErr != nil {
			if consumeErr != nil {
				return nil, consumeErr
			}
			continue
		}
		if handled, formErr := p.handleFormToken(cell, false); handled || formErr != nil {
			if formErr != nil {
				return nil, formErr
			}
			continue
		}
		if tag {
			if closing && name == cell.Name {
				return cell, p.closeTableNode(cell)
			}
			if closing && (name == "td" || name == "th") {
				_, _, ignoreErr := p.parseClosingTag()
				if ignoreErr != nil {
					return nil, ignoreErr
				}
				continue
			}
			if closing && (name == "tbody" || name == "thead" || name == "tfoot") && name != sectionName {
				_, _, ignoreErr := p.parseClosingTag()
				if ignoreErr != nil {
					return nil, ignoreErr
				}
				continue
			}
			if closing && (name == "body" || name == "caption" || name == "col" || name == "colgroup" || name == "html") {
				_, _, ignoreErr := p.parseClosingTag()
				if ignoreErr != nil {
					return nil, ignoreErr
				}
				continue
			}
			if closing && name == "tr" && (cell.Parent == nil || cell.Parent.Name != "tr") {
				if _, _, ignoreErr := p.parseClosingTag(); ignoreErr != nil {
					return nil, ignoreErr
				}
				continue
			}
			if closing && name == "table" && p.activeTable == nil {
				if _, _, ignoreErr := p.parseClosingTag(); ignoreErr != nil {
					return nil, ignoreErr
				}
				continue
			}
			if (closing && (name == "table" || name == sectionName || name == "tr")) || (!closing && (name == "td" || name == "th" || name == "tr" || name == "tbody" || name == "thead" || name == "tfoot" || name == "caption" || name == "col" || name == "colgroup")) {
				p.finishTableImplicit(cell, cell.ContentStart)
				return cell, nil
			}
		}
		if consumed, consumeErr := p.consumeTemplatePseudoAbsentEnd(cell); consumed || consumeErr != nil {
			if consumeErr != nil {
				return nil, consumeErr
			}
			continue
		}
		if name, closing, tag := p.peekTagName(); tag && closing && isCoreAdoptionSubject(name) && p.activeFormattingEntryByName(name) == nil {
			start := p.pos
			if _, _, closeErr := p.parseClosingTag(); closeErr != nil {
				return nil, closeErr
			}
			p.extendPreviousTextRange(cell, start)
			continue
		}
		if name, closing, tag := p.peekTagName(); tag && closing && isCoreAdoptionSubject(name) {
			action, adoptionErr := p.handleCoreAdoption(cell, name, cell.TextContent, cell.ContentStart, true)
			if adoptionErr != nil {
				return nil, adoptionErr
			}
			if action != adoptionNone {
				continue
			}
		}
		previousCell, previousSection := p.tableCellName, p.tableCellSection
		p.tableCellDepth++
		p.tableCellName, p.tableCellSection = cell.Name, sectionName
		insertionParent := p.formattingInsertionParent(cell, p.nextTokenReconstructsFormatting())
		child, childErr := p.parseNode(insertionParent)
		p.tableCellDepth--
		p.tableCellName, p.tableCellSection = previousCell, previousSection
		if childErr != nil {
			return nil, childErr
		}
		if child != nil {
			if insertionParent != cell {
				p.appendFormattingChild(cell, insertionParent, child)
				p.markTextDirty(cell)
				p.refreshNodeTextContent(cell)
			} else {
				p.appendTableChild(cell, child)
			}
			p.appendDeferredAdoption(cell, child)
		}
	}
	p.recoverOpenElementsAtEOF = true
	p.finishTableImplicit(cell, cell.ContentStart)
	return cell, nil
}

func (p *HTMLParser) parseTableOpenElement(parent *types.Node) (*types.Node, error) {
	start, line, column := p.pos, p.line, p.col
	if p.peek() != '<' {
		return nil, fmt.Errorf("expected table tag at position %d", p.pos)
	}
	p.advance()
	name := p.parseTagName()
	node := &types.Node{Type: types.ElementNode, Name: name, NamespaceURI: htmlElementNamespace, Attributes: make(map[string]string), AttributeOrder: []string{}, Children: []*types.Node{}, Parent: parent, StartPos: start, StartLine: line, StartColumn: column}
	if owner := p.templateOwner[parent]; owner != nil {
		p.templateOwner[node] = owner
	}
	_, emitted := p.parseStartTagTail(node)
	if !emitted {
		return nil, fmt.Errorf("incomplete table tag at position %d", start)
	}
	node.ContentStart = p.pos
	return node, nil
}

func isTableStructuralName(name string) bool {
	switch name {
	case "table", "caption", "colgroup", "col", "tbody", "tfoot", "thead", "tr", "td", "th":
		return true
	default:
		return false
	}
}

func isCaptionIgnoredEndTag(name string) bool {
	switch name {
	case "body", "col", "colgroup", "html", "tbody", "td", "tfoot", "th", "thead", "tr":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) currentFosterTransitionWillEmit() bool {
	name, closing, ok := p.peekTagName()
	if !ok || !p.currentTagTokenWillEmit() {
		return false
	}
	if !closing {
		switch name {
		case "table", "caption", "col", "colgroup", "tbody", "tfoot", "thead", "tr", "td", "th":
			return true
		}
	}
	if !closing {
		return false
	}
	if name == "table" {
		return true
	}
	if p.tableFosterRow && name == "tr" {
		return true
	}
	return p.tableFosterSection != "" && name == p.tableFosterSection
}

func (p *HTMLParser) currentFosterIgnoredEndTag() bool {
	name, closing, ok := p.peekTagName()
	if !ok || !closing || !p.currentTagTokenWillEmit() || name == "table" || (p.tableFosterRow && name == "tr") || (p.tableFosterSection != "" && name == p.tableFosterSection) {
		return false
	}
	return isTableStructuralName(name) || name == "body" || name == "html" || name == "noscript"
}

func (p *HTMLParser) currentCellTransitionWillEmit() bool {
	name, closing, ok := p.peekTagName()
	if !ok || !p.currentTagTokenWillEmit() {
		return false
	}
	if !closing {
		return isTableStructuralName(name) && name != "table"
	}
	return name == "table" || name == "tr" || name == p.tableCellName || (p.tableCellSection != "" && name == p.tableCellSection)
}

func (p *HTMLParser) currentCellIgnoredEndTag() bool {
	name, closing, ok := p.peekTagName()
	if !ok || !closing || !p.currentTagTokenWillEmit() || name == "table" || name == "tr" || name == p.tableCellName || (p.tableCellSection != "" && name == p.tableCellSection) {
		return false
	}
	return isTableStructuralName(name) || name == "body" || name == "html"
}

func (p *HTMLParser) currentCaptionTransitionWillEmit() bool {
	name, closing, ok := p.peekTagName()
	if !ok || !p.currentTagTokenWillEmit() {
		return false
	}
	if closing {
		return name == "caption" || name == "table"
	}
	switch name {
	case "caption", "col", "colgroup", "tbody", "tfoot", "thead", "tr", "td", "th":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) currentCaptionIgnoredEndTag() bool {
	name, closing, ok := p.peekTagName()
	return ok && closing && p.currentTagTokenWillEmit() && isCaptionIgnoredEndTag(name)
}

func (p *HTMLParser) consumeDiscardedTableToken(parent *types.Node) (*types.Node, error) {
	tokenStart := p.pos
	child, err := p.parseNode(parent)
	if err != nil {
		return nil, err
	}
	if child == nil {
		p.extendPreviousTextRange(parent, tokenStart)
		p.recoverOpenElementsAtEOF = true
	}
	return child, nil
}

func (p *HTMLParser) parseTableFosterNode(parent *types.Node) (*types.Node, error) {
	return p.parseTableFosterNodeInMode(parent, "", false)
}

// handleFormToken implements the whole-document form element pointer. The
// pointer and logical-open state are deliberately separate from DOM Parent:
// a non-current form end removes the pointed form from the open-elements stack
// while its still-open descendants remain DOM children of that form.
func (p *HTMLParser) handleFormToken(current *types.Node, inTable bool) (bool, error) {
	name, closing, tag := p.peekTagName()
	if !tag || name != "form" || !p.currentTagTokenWillEmit() {
		return false, nil
	}

	if !closing {
		if p.templateOwnerForNode(current) != nil {
			mode := templateModeInitial
			if len(p.templateInsertionModes) > 0 {
				mode = p.templateInsertionModes[len(p.templateInsertionModes)-1]
			}
			if inTable || mode != templateModeBody {
				p.consumeIgnoredStartTag()
				return true, nil
			}
			// The form pointer is ignored for an in-body form start while a
			// template is open. parseElement dynamically restores the surrounding
			// pointer when each template-local form frame returns.
			return false, nil
		}
		if p.activeForm != nil {
			p.consumeIgnoredStartTag()
			return true, nil
		}
		if inTable {
			form, err := p.parseTableOpenElement(current)
			if err != nil {
				return true, err
			}
			if form != nil {
				// The in-table rule inserts and immediately pops the source-backed
				// form. Keep the library's parse5-compatible start-only range.
				form.ContentStart = form.StartPos
				form.ContentEnd = form.StartPos
				form.EndPos = form.StartPos
				form.EndLine = form.StartLine
				form.EndColumn = form.StartColumn
				p.appendUniqueChild(current, form)
				p.activeForm = form
				p.openForms[form] = false
			}
			return true, nil
		}
		return false, nil
	}

	tokenStart := p.pos
	pointed := p.activeForm
	p.activeForm = nil // The algorithm clears the pointer before scope checks.
	if _, emitted, err := p.parseClosingTag(); err != nil {
		return true, err
	} else if !emitted {
		p.extendPreviousTextRange(current, tokenStart)
		return true, nil
	}
	if pointed == nil || !p.openForms[pointed] || !formattingInOrdinaryScope(pointed, current) {
		return true, nil
	}
	p.openForms[pointed] = false
	pointed.ContentEnd = tokenStart
	pointed.EndPos = p.pos
	pointed.EndLine = p.line
	pointed.EndColumn = p.col
	if pointed != current {
		p.closedForms[pointed] = true
		return true, nil
	}
	p.closedForms[pointed] = true
	return true, nil
}

func (p *HTMLParser) parseTableFosterNodeInMode(parent *types.Node, section string, row bool) (*types.Node, error) {
	previousSection, previousRow := p.tableFosterSection, p.tableFosterRow
	p.tableFosterDepth++
	p.tableFosterSection, p.tableFosterRow = section, row
	defer func() {
		p.tableFosterDepth--
		p.tableFosterSection, p.tableFosterRow = previousSection, previousRow
	}()
	effectiveCurrent := parent
	if open := p.lastOpenFormattingNode(); open != nil && p.formattingContainer[open] == parent {
		effectiveCurrent = open
	}
	// With scripting disabled, an in-table noscript start is fostered and its
	// frame closes when table structure is reprocessed. Its later source end
	// tag is therefore absent from table scope and is ignored.
	if name, closing, ok := p.peekTagName(); ok && closing && name == "noscript" && !p.scriptingEnabled && p.currentTagTokenWillEmit() {
		if _, _, err := p.parseClosingTag(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if handled, err := p.handleFormToken(effectiveCurrent, true); handled || err != nil {
		return nil, err
	}
	if hiddenInput, consumed, err := p.parseHiddenInputInTable(effectiveCurrent); err != nil {
		return nil, err
	} else if consumed {
		if hiddenInput != nil {
			if effectiveCurrent == parent {
				p.appendTableChild(parent, hiddenInput)
			} else {
				p.appendFormattingChild(parent, effectiveCurrent, hiddenInput)
			}
		}
		return nil, nil
	}
	if action, err := p.handleSpecialFormattingStart(effectiveCurrent, effectiveCurrent.TextContent, effectiveCurrent.ContentStart); err != nil {
		return nil, err
	} else if action != adoptionNone {
		return nil, nil
	}
	if name, closing, ok := p.peekTagName(); ok && closing && isCoreAdoptionSubject(name) {
		action, err := p.handleCoreAdoption(effectiveCurrent, name, effectiveCurrent.TextContent, effectiveCurrent.ContentStart, true)
		if err != nil {
			return nil, err
		}
		if action != adoptionNone {
			return nil, nil
		}
	}
	insertionParent := p.formattingInsertionParent(parent, p.nextTableTokenReconstructsFormatting(parent))
	if insertionParent == parent {
		return p.parseNode(parent)
	}
	outerIndex := len(parent.Children) - 1
	outer := parent.Children[outerIndex]
	parent.Children = parent.Children[:outerIndex]
	child, err := p.parseNode(insertionParent)
	if err != nil {
		return nil, err
	}
	if child != nil {
		p.appendFormattingChild(parent, insertionParent, child)
		outer.TextContent = child.TextContent
		outer.ContentEnd = child.EndPos
		outer.EndPos = child.EndPos
		outer.EndLine = child.EndLine
		outer.EndColumn = child.EndColumn
	}
	return outer, nil
}

// parseHiddenInputInTable implements the in-table hidden-input exception.
// It previews the fully tokenized attributes so character references and the
// tokenizer's first-duplicate-wins rule are honored, then consumes the real
// void element only when its type value is exactly ASCII-folded "hidden".
func (p *HTMLParser) parseHiddenInputInTable(parent *types.Node) (*types.Node, bool, error) {
	name, closing, tag := p.peekTagName()
	if !tag || closing || name != "input" || !p.currentTagTokenWillEmit() {
		return nil, false, nil
	}

	preview := *p
	preview.advance() // '<'
	if preview.parseTagName() != "input" {
		return nil, false, nil
	}
	attributes := &types.Node{Attributes: make(map[string]string), AttributeOrder: []string{}}
	if _, emitted := preview.parseStartTagTail(attributes); !emitted || !asciiEqualFold(attributes.Attributes["type"], "hidden") {
		return nil, false, nil
	}

	input, err := p.parseNode(parent)
	if err != nil {
		return nil, true, err
	}
	return input, true, nil
}

func asciiEqualFold(value, expected string) bool {
	if len(value) != len(expected) {
		return false
	}
	for index := 0; index < len(value); index++ {
		left := value[index]
		if left >= 'A' && left <= 'Z' {
			left += 'a' - 'A'
		}
		if left != expected[index] {
			return false
		}
	}
	return true
}

func asciiLower(value string) string {
	var lowered strings.Builder
	lowered.Grow(len(value))
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character >= 'A' && character <= 'Z' {
			character += 'a' - 'A'
		}
		lowered.WriteByte(character)
	}
	return lowered.String()
}

func (p *HTMLParser) nextTableTokenReconstructsFormatting(parent *types.Node) bool {
	if !p.nextTokenReconstructsFormatting() {
		return false
	}
	if _, _, tag := p.peekTagName(); tag {
		return true
	}
	// The in-table-text rules buffer character tokens before deciding whether
	// foster parenting applies. A buffer containing only HTML whitespace stays
	// in the table and must not reconstruct an off-stack formatting element.
	preview := *p
	previewParent := &types.Node{Type: types.ElementNode, Name: parent.Name, Children: []*types.Node{}}
	text, err := preview.parseTextNode(previewParent, preview.pos, preview.line, preview.col)
	if err != nil || text == nil {
		return true
	}
	return !isHTMLWhitespaceString(text.Value)
}

func (p *HTMLParser) parseTableColgroup(table *types.Node, synthetic bool) (*types.Node, error) {
	var group *types.Node
	if synthetic {
		group = syntheticTableNode("colgroup", table)
	} else {
		var err error
		group, err = p.parseTableOpenElement(table)
		if err != nil {
			return nil, err
		}
	}
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(group) {
			p.finishTableImplicit(group, group.ContentStart)
			return group, nil
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			if _, err := p.consumeDiscardedTableToken(group); err != nil {
				return nil, err
			}
			p.finishTableImplicit(group, group.ContentStart)
			return group, nil
		}
		if tag && closing && name == "colgroup" {
			if synthetic {
				_, _, err := p.parseClosingTag()
				return group, err
			}
			return group, p.closeTableNode(group)
		}
		if tag && closing && name == "col" {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if tag && !closing && name == "col" {
			col, err := p.parseNode(group)
			if err != nil {
				return nil, err
			}
			p.appendTableChild(group, col)
			continue
		}
		if tag && (isTableStructuralName(name) || closing) {
			p.finishTableImplicit(group, group.ContentStart)
			return group, nil
		}
		if p.peek() == '<' && p.startsMarkupToken() && !p.nextTokenIsComment() && !p.nextTokenIsProcessingInstruction() {
			p.finishTableImplicit(group, group.ContentStart)
			return group, nil
		}
		triggerPos, triggerLine, triggerColumn := p.pos, p.line, p.col
		child, err := p.parseNode(group)
		if err != nil {
			return nil, err
		}
		if child != nil && (child.Type == types.CommentNode || child.Type == types.ProcessingInstructionNode || (child.Type == types.TextNode && isHTMLWhitespaceString(child.Value))) {
			p.appendTableChild(group, child)
			continue
		}
		if child != nil {
			// Non-whitespace character tokens leave colgroup mode and are
			// reprocessed by the table. Text cannot be rolled back, so retain
			// the already-tokenized node as fostered content.
			p.fosterTableChild(table, child)
		}
		p.finishElementAt(group, group.TextContent, group.ContentStart, triggerPos, triggerLine, triggerColumn)
		return group, nil
	}
	p.recoverOpenElementsAtEOF = true
	p.finishTableImplicit(group, group.ContentStart)
	return group, nil
}

func syntheticTableNode(name string, parent *types.Node) *types.Node {
	return &types.Node{Type: types.ElementNode, Name: name, NamespaceURI: htmlElementNamespace, Attributes: make(map[string]string), Children: []*types.Node{}, Parent: parent}
}

func (p *HTMLParser) closeTableNode(node *types.Node) error {
	p.markTextDirty(node)
	p.refreshNodeTextContent(node)
	contentEnd := p.pos
	name, emitted, err := p.parseClosingTag()
	if err != nil {
		return err
	}
	if !emitted || name != node.Name {
		return fmt.Errorf("expected closing tag </%s>", node.Name)
	}
	if node.StartLine == 0 {
		return nil
	}
	node.ContentEnd = contentEnd
	node.EndPos = p.pos
	node.EndLine, node.EndColumn = p.line, p.col
	return nil
}

func (p *HTMLParser) finishTableImplicit(node *types.Node, contentStart int) {
	p.markTextDirty(node)
	p.refreshNodeTextContent(node)
	p.finishImplicitElement(node, node.TextContent, contentStart)
}

// markTextDirty invalidates the changed node and every currently attached
// ancestor. A later synchronization only rebuilds nodes that remain dirty.
func (p *HTMLParser) markTextDirty(node *types.Node) {
	for current := node; current != nil; current = current.Parent {
		if _, alreadyDirty := p.dirtyTextNodes[current]; alreadyDirty {
			// The first invalidation of a branch always marks its ancestors, so
			// later append-only changes need not walk the same open-element path.
			break
		}
		p.dirtyTextNodes[current] = struct{}{}
	}
}

// markTextSubtreeDirty is reserved for final document synchronization. It
// covers newly assembled virtual recovery chains whose parents did not exist
// when their children were moved; each node is still rebuilt only once by the
// dirty-aware traversal below.
func (p *HTMLParser) markTextSubtreeDirty(node *types.Node) {
	if node == nil {
		return
	}
	p.dirtyTextNodes[node] = struct{}{}
	for _, child := range node.Children {
		if child.Type == types.ElementNode {
			p.markTextSubtreeDirty(child)
		}
	}
}

// refreshNodeTextContent is a synchronization boundary for local recovery
// edits. It intentionally does no work for a clean tree.
func (p *HTMLParser) refreshNodeTextContent(node *types.Node) {
	p.refreshTreeText(node)
}

func (p *HTMLParser) refreshTreeText(node *types.Node) {
	if node == nil {
		return
	}
	if _, dirty := p.dirtyTextNodes[node]; !dirty {
		return
	}
	for _, child := range node.Children {
		if _, dirty := p.dirtyTextNodes[child]; dirty {
			p.refreshTreeText(child)
		}
	}
	p.refreshVisits++
	p.rebuildTextContent(node)
	delete(p.dirtyTextNodes, node)
}

// finalizeTextContent is the one full-tree synchronization at the end of a
// Parse call. Recovery paths use refreshTreeText instead, so they never pay
// for an unconditional subtree walk while manipulating an open tree.
func (p *HTMLParser) finalizeTextContent(node *types.Node) {
	if node == nil {
		return
	}
	for _, child := range node.Children {
		if child.Type == types.ElementNode {
			p.finalizeTextContent(child)
		}
	}
	p.rebuildTextContent(node)
}

func (p *HTMLParser) rebuildTextContent(node *types.Node) {
	var only string
	contributors := 0
	for _, child := range node.Children {
		switch child.Type {
		case types.TextNode:
			only = child.Value
			contributors++
		case types.ElementNode:
			only = child.TextContent
			contributors++
		}
		if contributors > 1 {
			break
		}
	}
	if contributors <= 1 {
		node.TextContent = only
		return
	}
	var text strings.Builder
	for _, child := range node.Children {
		switch child.Type {
		case types.TextNode:
			text.WriteString(child.Value)
		case types.ElementNode:
			text.WriteString(child.TextContent)
		}
	}
	node.TextContent = text.String()
}

func (p *HTMLParser) appendTableChild(parent, child *types.Node) {
	if child == nil {
		return
	}
	child.Parent = parent
	if child.Type == types.TextNode && len(parent.Children) > 0 && parent.Children[len(parent.Children)-1].Type == types.TextNode {
		mergeTextNodes(parent.Children[len(parent.Children)-1], child)
	} else {
		parent.Children = append(parent.Children, child)
	}
	p.markTextDirty(parent)
	p.appendDeferredTableFoster(parent, child)
}

func (p *HTMLParser) fosterTableChild(table, child *types.Node) {
	if table == nil || table.Parent == nil || child == nil {
		return
	}
	p.deferredTableFoster[table] = append(p.deferredTableFoster[table], child)
}

func (p *HTMLParser) appendDeferredTableFoster(parent, child *types.Node) string {
	if child == nil || child.Name != "table" {
		return ""
	}
	fosteredNodes := p.deferredTableFoster[child]
	if len(fosteredNodes) == 0 {
		return ""
	}
	tableIndex := len(parent.Children) - 1
	if tableIndex < 0 || parent.Children[tableIndex] != child {
		tableIndex = -1
		for index, candidate := range parent.Children {
			if candidate == child {
				tableIndex = index
				break
			}
		}
	}
	if tableIndex < 0 {
		return ""
	}
	suffix := append([]*types.Node(nil), parent.Children[tableIndex+1:]...)
	parent.Children = parent.Children[:tableIndex]
	var fosteredText strings.Builder
	for index := 0; index < len(fosteredNodes); {
		fostered := fosteredNodes[index]
		if fostered.Type != types.TextNode {
			fostered.Parent = parent
			parent.Children = append(parent.Children, fostered)
			// Adoption surgery performed while the element was logically open in
			// table mode may have produced deferred siblings. They belong at the
			// fostered element's DOM position, before the table, rather than in the
			// document-level adoption finalizer after the table.
			p.appendDeferredAdoption(parent, fostered)
			if fostered.Type == types.ElementNode {
				fosteredText.WriteString(fostered.TextContent)
			}
			index++
			continue
		}
		first := fostered
		last := fostered
		var value strings.Builder
		for index < len(fosteredNodes) && fosteredNodes[index].Type == types.TextNode {
			last = fosteredNodes[index]
			value.WriteString(last.Value)
			index++
		}
		combined := value.String()
		fosteredText.WriteString(combined)
		first.Value = combined
		first.TextContent = combined
		first.EndPos = last.EndPos
		first.EndLine = last.EndLine
		first.EndColumn = last.EndColumn
		first.Parent = parent
		if len(parent.Children) > 0 && parent.Children[len(parent.Children)-1].Type == types.TextNode {
			mergeTextNodes(parent.Children[len(parent.Children)-1], first)
		} else {
			parent.Children = append(parent.Children, first)
		}
	}
	parent.Children = append(parent.Children, child)
	parent.Children = append(parent.Children, suffix...)
	delete(p.deferredTableFoster, child)
	return fosteredText.String()
}

func (p *HTMLParser) appendDeferredListComments(parent, child *types.Node) {
	var comments []*types.Node
	switch {
	case parent == p.documentHTML && child == p.documentBody:
		comments = p.deferredHTMLComments
		p.deferredHTMLComments = nil
	case parent == p.documentRoot && child == p.documentHTML:
		comments = p.deferredRootComments
		p.deferredRootComments = nil
	default:
		return
	}
	for _, comment := range comments {
		comment.Parent = parent
		parent.Children = append(parent.Children, comment)
	}
}

func (p *HTMLParser) parseTableLooseContainer(parent *types.Node, expected string, stop map[string]bool) (*types.Node, error) {
	node, err := p.parseTableOpenElement(parent)
	if err != nil {
		return nil, err
	}
	if expected == "caption" {
		previousFormattingFamilies := p.formattingFamilies
		previousFormattingByName := p.formattingByName
		previousFormattingMarkerStart := p.formattingMarkerStart
		p.activeFormatting = append(p.activeFormatting, &formattingEntry{marker: true})
		p.formattingMarkerStart = len(p.activeFormatting)
		p.formattingFamilies = make(map[string][]*formattingEntry)
		p.formattingByName = make(map[string][]*formattingEntry)
		defer func() {
			p.clearActiveFormattingToMarker()
			p.formattingFamilies = previousFormattingFamilies
			p.formattingByName = previousFormattingByName
			p.formattingMarkerStart = previousFormattingMarkerStart
		}()
		defer p.closeFormattingOnElementReturn(node)
	}
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(node) {
			p.finishTableImplicit(node, node.ContentStart)
			return node, nil
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			if _, err := p.consumeDiscardedTableToken(node); err != nil {
				return nil, err
			}
			p.finishTableImplicit(node, node.ContentStart)
			return node, nil
		}
		if consumed, consumeErr := p.consumeForeignIgnoredEnd(node); consumed || consumeErr != nil {
			if consumeErr != nil {
				return nil, consumeErr
			}
			continue
		}
		if expected == "caption" {
			if handled, formErr := p.handleFormToken(node, false); handled || formErr != nil {
				if formErr != nil {
					return nil, formErr
				}
				continue
			}
		}
		if tag && closing && name == expected {
			return node, p.closeTableNode(node)
		}
		if tag && closing && expected == "caption" {
			if name == "table" {
				p.finishTableImplicit(node, node.ContentStart)
				return node, nil
			}
			if isCaptionIgnoredEndTag(name) {
				_, _, err := p.parseClosingTag()
				if err != nil {
					return nil, err
				}
				continue
			}
		}
		if tag && !closing && stop[name] {
			p.finishTableImplicit(node, node.ContentStart)
			return node, nil
		}
		if expected == "caption" {
			if name, closing, tag := p.peekTagName(); tag && closing && isCoreAdoptionSubject(name) && p.activeFormattingEntryByName(name) == nil {
				start := p.pos
				if _, _, closeErr := p.parseClosingTag(); closeErr != nil {
					return nil, closeErr
				}
				p.extendPreviousTextRange(node, start)
				continue
			}
			if name, closing, tag := p.peekTagName(); tag && closing && isCoreAdoptionSubject(name) {
				action, adoptionErr := p.handleCoreAdoption(node, name, node.TextContent, node.ContentStart, true)
				if adoptionErr != nil {
					return nil, adoptionErr
				}
				if action != adoptionNone {
					continue
				}
			}
		}
		if expected == "caption" {
			p.tableCaptionDepth++
		}
		insertionParent := node
		if expected == "caption" {
			insertionParent = p.formattingInsertionParent(node, p.nextTokenReconstructsFormatting())
		}
		child, childErr := p.parseNode(insertionParent)
		if expected == "caption" {
			p.tableCaptionDepth--
		}
		if childErr != nil {
			return nil, childErr
		}
		if child != nil {
			if insertionParent != node {
				p.appendFormattingChild(node, insertionParent, child)
				p.markTextDirty(node)
				p.refreshNodeTextContent(node)
			} else {
				p.appendTableChild(node, child)
			}
			p.appendDeferredAdoption(node, child)
		}
	}
	p.recoverOpenElementsAtEOF = true
	p.finishTableImplicit(node, node.ContentStart)
	return node, nil
}

func mergeTextNodes(target, source *types.Node) {
	target.Value += source.Value
	target.TextContent += source.TextContent
	target.EndPos = source.EndPos
	target.EndLine = source.EndLine
	target.EndColumn = source.EndColumn
}

// consumeVoidElementClosingTag applies browser recovery for end tags belonging
// to HTML void elements. Browsers ignore these tokens except for </br>, which
// creates a br element. Any recovered node retains the original token's range.
func (p *HTMLParser) consumeVoidElementClosingTag(parent *types.Node) (*types.Node, bool, error) {
	tagName, closing, ok := p.peekTagName()
	if !ok || !closing || !p.isSelfClosingTag(tagName) {
		return nil, false, nil
	}

	startPos := p.pos
	startLine := p.line
	startColumn := p.col
	if _, emitted, err := p.parseClosingTag(); err != nil {
		return nil, true, err
	} else if !emitted {
		p.recoverOpenElementsAtEOF = true
		p.extendPreviousTextRange(parent, startPos)
		return nil, true, nil
	}
	if tagName != "br" {
		return nil, true, nil
	}

	recovered := &types.Node{
		Type:           types.ElementNode,
		Name:           "br",
		Attributes:     make(map[string]string),
		AttributeOrder: []string{},
		Children:       []*types.Node{},
		Parent:         parent,
		StartPos:       startPos,
		EndPos:         p.pos,
		ContentStart:   p.pos,
		ContentEnd:     p.pos,
		StartLine:      startLine,
		StartColumn:    startColumn,
		EndLine:        p.line,
		EndColumn:      p.col,
	}
	if p.foreignBRReprocess {
		p.foreignBRReprocess = false
		recovered.StartPos, recovered.EndPos = 0, 0
		recovered.ContentStart, recovered.ContentEnd = 0, 0
		recovered.StartLine, recovered.StartColumn = 0, 0
		recovered.EndLine, recovered.EndColumn = 0, 0
	}
	return recovered, true, nil
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
	if p.tagPreview.position == p.pos {
		return p.tagPreview.name, p.tagPreview.closing, true
	}
	if p.pos >= len(p.content) || p.content[p.pos] != '<' {
		return "", false, false
	}

	position := p.pos + 1
	if position < len(p.content) && p.content[position] == '/' {
		closing = true
		position++
	}
	start := position
	for position < len(p.content) && !isWhitespace(p.content[position]) && p.content[position] != '/' && p.content[position] != '>' {
		position++
	}
	if position == start {
		return "", false, false
	}

	var normalized strings.Builder
	for _, r := range p.content[start:position] {
		if r == 0 {
			r = '\uFFFD'
		} else if isASCIIUpper(r) {
			r += 'a' - 'A'
		}
		normalized.WriteRune(r)
	}
	name = normalized.String()
	p.tagPreview = tagPreview{position: p.pos, name: name, closing: closing}
	return name, closing, true
}

// consumeInitialPreformattedLF applies the in-body one-shot rule for HTML
// pre/listing elements. The check survives malformed input that emits no
// token, but any emitted token (including a NUL later ignored by the tree
// builder) consumes it. A stripped LF is removed before active-formatting
// reconstruction can create an otherwise empty wrapper.
func (p *HTMLParser) consumeInitialPreformattedLF(active *bool) (suppressed, retainSourceStart bool, start, line, column int) {
	if active == nil || !*active || p.pos >= len(p.content) {
		return false, false, 0, 0, 0
	}
	start, line, column = p.pos, p.line, p.col
	if p.peek() == '<' && p.startsMarkupToken() {
		// The missing end-tag-name state for exact </> consumes source input but
		// emits no token, so the next real token still gets the LF check.
		if p.isMissingEndTagName() {
			return false, false, start, line, column
		}
		if _, _, ok := p.peekTagName(); ok {
			if p.currentTagTokenWillEmit() {
				*active = false
			}
			return false, false, start, line, column
		}
		// Comments, doctypes, bogus comments, and processing instructions are
		// emitted tokenizer tokens even when their tree-builder action ignores
		// them. An incomplete token cannot be followed by a later LF anyway.
		*active = false
		return false, false, start, line, column
	}

	// The next token is a character token. It always consumes the one-shot
	// check, even when the data-state NUL rule later omits it from the DOM.
	*active = false
	if p.peek() == 0 {
		return false, false, start, line, column
	}
	preview := *p
	if value, consumed := preview.consumeNumericCharacterReference(); consumed {
		if value != '\n' {
			return false, false, start, line, column
		}
		_, _ = p.consumeNumericCharacterReference()
		suppressed = true
	} else {
		preview = *p
		if value, consumed := preview.consumeNamedCharacterReference(false); consumed {
			if value != "\n" {
				return false, false, start, line, column
			}
			_, _ = p.consumeNamedCharacterReference(false)
			suppressed = true
		} else {
			r, size := p.peekHTMLRune()
			if r != '\n' || size == 0 {
				return false, false, start, line, column
			}
			p.advanceRune(size)
			suppressed = true
		}
	}

	// parse5 retains the stripped token in the source range when the next
	// emitted character is also LF, even though only the latter reaches DOM.
	if p.pos < len(p.content) && (p.peek() != '<' || !p.startsMarkupToken()) && p.peek() != 0 {
		preview = *p
		if value, consumed := preview.consumeNumericCharacterReference(); consumed {
			retainSourceStart = value == '\n'
		} else {
			preview = *p
			if value, consumed := preview.consumeNamedCharacterReference(false); consumed {
				retainSourceStart = strings.HasPrefix(value, "\n")
			} else {
				r, _ := p.peekHTMLRune()
				retainSourceStart = r == '\n'
			}
		}
	}
	return suppressed, retainSourceStart, start, line, column
}

// preformattedRangeStartContinues reports whether a stripped initial LF still
// belongs to the source range of the next emitted LF character token. Exact
// </> consumes input without emitting a token and therefore preserves the
// tentative boundary; every other emitted/non-LF token retires it.
func (p *HTMLParser) preformattedRangeStartContinues() bool {
	if p.pos >= len(p.content) {
		return false
	}
	if p.peek() == '<' && p.startsMarkupToken() {
		return p.isMissingEndTagName()
	}
	if p.peek() == 0 {
		return false
	}
	preview := *p
	if value, consumed := preview.consumeNumericCharacterReference(); consumed {
		return value == '\n'
	}
	preview = *p
	if value, consumed := preview.consumeNamedCharacterReference(false); consumed {
		return strings.HasPrefix(value, "\n")
	}
	r, _ := p.peekHTMLRune()
	return r == '\n'
}

// parseTextNode parses a text node
func (p *HTMLParser) parseTextNode(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	var text strings.Builder

	for p.pos < len(p.content) {
		if p.peek() == '<' && p.startsMarkupToken() {
			break
		}
		if p.content[p.pos:] == "</" {
			p.recoverOpenElementsAtEOF = true
		}
		// The HTML tokenizer ignores NUL in the data state. Other states such
		// as comments, attributes, and raw text replace it with U+FFFD.
		if p.peek() == 0 {
			p.advance()
			if text.Len() == 0 {
				startPos, startLine, startCol = p.pos, p.line, p.col
			}
			continue
		}
		if value, consumed := p.consumeNumericCharacterReference(); consumed {
			text.WriteRune(value)
			continue
		}
		if value, consumed := p.consumeNamedCharacterReference(false); consumed {
			text.WriteString(value)
			continue
		}
		// Handle UTF-8 correctly by reading the full character
		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		text.WriteRune(r)
		p.advanceRune(size)
	}

	if text.Len() == 0 {
		return nil, nil
	}
	textValue := text.String()

	// Preserve all text nodes including whitespace-only text for XPath compatibility
	// Don't skip whitespace-only text nodes as they are significant for XPath expressions

	return &types.Node{
		Type:        types.TextNode,
		Name:        "#text",
		Value:       textValue, // Preserve original text with whitespace
		TextContent: textValue, // Preserve original text with whitespace
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}
