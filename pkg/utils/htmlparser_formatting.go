// Active-formatting elements and adoption-agency recovery.
package utils

import (
	"slices"
	"sort"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func isImpliedEndTagName(name string) bool {
	switch name {
	case "dd", "dt", "li", "option", "optgroup", "p", "rb", "rp", "rt", "rtc":
		return true
	default:
		return false
	}
}

func isFormattingElement(name string) bool {
	switch name {
	case "a", "b", "big", "code", "em", "font", "i", "nobr", "s", "small", "strike", "strong", "tt", "u":
		return true
	default:
		return false
	}
}

func isCoreAdoptionSubject(name string) bool {
	switch name {
	case "a", "b", "big", "code", "em", "font", "i", "nobr", "s", "small", "strike", "strong", "tt", "u":
		return true
	default:
		return false
	}
}

type adoptionAction uint8

const (
	adoptionNone adoptionAction = iota
	adoptionContinue
	adoptionReturn
)

func (p *HTMLParser) consumeAdoptionEnd(consume bool) (bool, error) {
	if !consume {
		return true, nil
	}
	_, emitted, err := p.parseClosingTag()
	return emitted, err
}

func (p *HTMLParser) resetAdoptionUnwindCache() {
	p.adoptionUnwindPos = -1
	p.adoptionUnwindFormatting = nil
	p.adoptionUnwindBlock = nil
	p.adoptionUnwindInScope = false
}

func (p *HTMLParser) handleSpecialFormattingStart(current *types.Node, text string, contentStart int) (adoptionAction, error) {
	name, closing, ok := p.peekTagName()
	if !ok || closing || (name != "a" && name != "nobr") || !p.currentTagTokenWillEmit() {
		return adoptionNone, nil
	}
	firstPass := p.specialFormattingStartPos != p.pos || p.specialFormattingStartName != name
	if firstPass {
		p.specialFormattingStartPos = p.pos
		p.specialFormattingStartName = name
	}
	if name == "nobr" && firstPass {
		p.formattingInsertionParent(current, true)
	}
	entry := p.activeFormattingEntryByName(name)
	if entry == nil {
		return adoptionNone, nil
	}
	// A formatting entry can remain active after its source element has been
	// popped by a block.  An anchor start must still remove that exact old
	// entry before the normal reconstruction step.  For nobr the mandatory
	// reconstruction above gives the entry a current virtual-open clone.
	if entry.current == nil {
		if name == "a" {
			p.removeActiveFormattingEntry(entry)
			p.resetAdoptionUnwindCache()
		}
		return adoptionNone, nil
	}
	// Recursive source parents are not always the parser's effective open
	// formatting stack.  After block unwinding, reconstructed formatting
	// elements are DOM children of current while remaining virtually open.
	// Fake adoption must pop through the subject on that virtual stack, retain
	// later AFE entries for reconstruction, and remove only the old subject.
	virtualContainer := current
	if p.documentBody != nil && (current == p.documentHTML || current == p.documentRoot) {
		virtualContainer = p.documentBody
	}
	if p.openFormatting[entry.current] && p.formattingContainer[entry.current] == virtualContainer {
		p.closeVirtualFormattingThrough(entry, virtualContainer)
		return adoptionNone, nil
	}
	if name == "nobr" && !formattingInOrdinaryScope(entry.current, current) {
		return adoptionNone, nil
	}
	old := entry
	p.resetAdoptionUnwindCache()
	action, err := p.handleCoreAdoption(current, name, text, contentStart, false)
	if err != nil {
		return adoptionNone, err
	}
	if name == "a" && action != adoptionReturn {
		if slices.Contains(p.activeFormatting, old) {
			if old.current != nil && old.current.StartLine != 0 && !formattingInOrdinaryScope(old.current, current) {
				old.current.ContentEnd = p.pos
				old.current.EndPos = p.pos
				old.current.EndLine = p.line
				old.current.EndColumn = p.col
				p.setAdoptionFlag(old.current, adoptionReturnAtSplit)
			}
			p.removeActiveFormattingEntry(old)
		}
	}
	p.resetAdoptionUnwindCache()
	return action, nil
}

func (p *HTMLParser) closeVirtualFormattingThrough(subject *formattingEntry, container *types.Node) {
	if subject == nil || subject.current == nil {
		return
	}
	subjectNode := subject.current
	for index := len(p.formattingOpenStack) - 1; index >= 0; index-- {
		node := p.formattingOpenStack[index]
		if !p.openFormatting[node] || p.formattingContainer[node] != container {
			continue
		}
		p.refreshFormattingText(node)
		if node.StartLine != 0 {
			node.ContentEnd = p.pos
			node.EndPos = p.pos
			node.EndLine = p.line
			node.EndColumn = p.col
		}
		p.setFormattingOpen(node, false)
		if node == subjectNode {
			break
		}
		if entry := p.formattingByNode[node]; entry != nil {
			delete(p.formattingByNode, node)
			entry.current = nil
		}
	}
	p.removeActiveFormattingEntry(subject)
	p.resetAdoptionUnwindCache()
}

func (p *HTMLParser) handleCoreAdoption(current *types.Node, name, text string, contentStart int, consume bool) (adoptionAction, error) {
	entry := p.activeFormattingEntryByName(name)
	if entry == nil || entry.current == nil {
		if entry != nil {
			p.removeActiveFormattingEntry(entry)
		}
		if current.Name == name {
			return adoptionNone, nil
		}
		if target := adoptionGenericEndTarget(current, name); target != nil {
			closeStart := p.pos
			closeLine, closeColumn := p.line, p.col
			if emitted, err := p.consumeAdoptionEnd(consume); err != nil {
				return adoptionNone, err
			} else if !emitted {
				p.extendPreviousTextRange(current, closeStart)
				return adoptionContinue, nil
			}
			for node := current; node != nil && node != target; node = node.Parent {
				node.ContentEnd = closeStart
				node.EndPos = closeStart
				node.EndLine = closeLine
				node.EndColumn = closeColumn
				p.setAdoptionFlag(node, adoptionClosed)
				p.adoptionIgnoredEnds[node.Name]++
			}
			p.markTextDirty(target)
			p.refreshTreeText(target)
			target.ContentEnd = closeStart
			target.EndPos = p.pos
			target.EndLine = p.line
			target.EndColumn = p.col
			p.setAdoptionFlag(target, adoptionClosed)
			return adoptionReturn, nil
		}
		start := p.pos
		if _, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		}
		p.extendPreviousTextRange(current, start)
		if nearestOpenElementByName(current.Parent, name) != nil {
			p.recoverOpenElementsAtEOF = true
		}
		return adoptionContinue, nil
	}
	formatting := entry.current
	if !p.openFormatting[formatting] {
		start := p.pos
		if _, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		}
		p.extendPreviousTextRange(current, start)
		p.removeActiveFormatting(formatting)
		return adoptionContinue, nil
	}
	if p.formattingContainer[formatting] == current && p.lastOpenFormattingNode() == formatting {
		documentEndClose := p.formattingDocumentEnd
		contentEnd := p.pos
		if emitted, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		} else if !emitted {
			p.extendPreviousTextRange(formatting, contentEnd)
		}
		p.refreshFormattingChain(formatting)
		if formatting.StartLine != 0 {
			formatting.ContentEnd = contentEnd
			formatting.EndPos = p.pos
			formatting.EndLine = p.line
			formatting.EndColumn = p.col
		}
		p.removeActiveFormatting(formatting)
		if formatting.StartLine == 0 {
			formatting.StartPos, formatting.ContentStart, formatting.ContentEnd, formatting.EndPos = 0, 0, 0, 0
			formatting.StartColumn, formatting.EndLine, formatting.EndColumn = 0, 0, 0
		}
		if !documentEndClose {
			p.recoverOpenElementsAtEOF = true
			p.adoptionEOFBoundary = current
		}
		return adoptionContinue, nil
	}
	if p.formattingContainer[formatting] == current {
		documentEndClose := p.formattingDocumentEnd
		closeStart := p.pos
		if emitted, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		} else if !emitted {
			p.extendPreviousTextRange(formatting, closeStart)
		}
		for node := p.lastOpenFormattingNode(); node != nil && node != formatting; node = node.Parent {
			entry := p.formattingByNode[node]
			if entry == nil {
				continue
			}
			p.refreshFormattingText(node)
			node.ContentEnd = closeStart
			node.EndPos = closeStart
			node.EndLine = p.line
			node.EndColumn = p.col
			p.setFormattingOpen(node, false)
			delete(p.formattingByNode, node)
			entry.current = nil
		}
		p.markTextDirty(formatting)
		p.refreshTreeText(formatting)
		if formatting.StartLine != 0 {
			formatting.ContentEnd = closeStart
			formatting.EndPos = p.pos
			formatting.EndLine = p.line
			formatting.EndColumn = p.col
		}
		p.removeActiveFormatting(formatting)
		if !documentEndClose {
			p.recoverOpenElementsAtEOF = true
		}
		return adoptionContinue, nil
	}
	if !p.cachedAdoptionInScope(formatting, current) {
		start := p.pos
		if _, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		}
		p.extendPreviousTextRange(current, start)
		return adoptionContinue, nil
	}
	if current == formatting {
		contentEnd := p.pos
		if emitted, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		} else if !emitted {
			p.extendPreviousTextRange(current, contentEnd)
		}
		current.TextContent = text
		if current.StartLine != 0 {
			current.ContentStart = contentStart
			current.ContentEnd = contentEnd
			current.EndPos = p.pos
			current.EndLine = p.line
			current.EndColumn = p.col
		}
		p.removeActiveFormatting(current)
		return adoptionReturn, nil
	}
	if entry.current != entry.template && nodeWithin(entry.current, current) {
		contentEnd := p.pos
		if emitted, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		} else if !emitted {
			p.extendPreviousTextRange(entry.current, contentEnd)
		}
		p.refreshFormattingChain(entry.current)
		if entry.current.StartLine != 0 {
			entry.current.ContentEnd = contentEnd
			entry.current.EndPos = p.pos
			entry.current.EndLine = p.line
			entry.current.EndColumn = p.col
		}
		p.removeActiveFormatting(entry.current)
		return adoptionContinue, nil
	}

	block := p.cachedAdoptionFurthestBlock(formatting, current)
	if block == nil {
		if current != formatting {
			p.adoptionIgnoredEnds[current.Name]++
		}
		p.finishImplicitElement(current, text, contentStart)
		return adoptionReturn, nil
	}
	if block != current {
		p.finishImplicitElement(current, text, contentStart)
		return adoptionReturn, nil
	}

	closeStart := p.pos
	closeLine, closeColumn := p.line, p.col
	if emitted, err := p.consumeAdoptionEnd(consume); err != nil {
		return adoptionNone, err
	} else if !emitted {
		p.extendIncompleteAdoptionAtEOF(block, closeStart)
		p.recoverOpenElementsAtEOF = true
		p.finishImplicitElement(block, block.TextContent, block.ContentStart)
		return adoptionReturn, nil
	}
	wrappers := p.adoptionInnerFormatting(formatting, block, closeStart, closeLine, closeColumn)
	oldAncestors := make([]*types.Node, 0, 4)
	for current := block.Parent; current != nil; current = current.Parent {
		oldAncestors = append(oldAncestors, current)
		if current == formatting {
			break
		}
	}
	p.detachDeferredAdoption(block)
	p.detachNodeShallow(block)
	p.adoptionPendingEnds[block] = adoptionSpecialDescendants(block)
	p.adoptionWrapSpecialContents(block, formatting, nil, closeStart, closeLine, closeColumn, 8)
	for _, current := range oldAncestors {
		delete(p.formattingText, current)
		p.markTextDirty(current)
	}
	for current := formatting; current != nil; current = current.Parent {
		if current.Type == types.DocumentNode {
			break
		}
		delete(p.formattingText, current)
		p.markTextDirty(current)
	}
	p.markTextDirty(formatting)
	for node := block.Parent; node != nil && node != formatting; node = node.Parent {
		p.markTextDirty(node)
	}
	p.markTextDirty(block)
	if formatting.StartLine != 0 {
		formatting.ContentEnd = closeStart
		formatting.EndPos = p.pos
		formatting.EndLine = p.line
		formatting.EndColumn = p.col
	}
	p.removeActiveFormatting(formatting)
	if formatting.StartLine != 0 {
		p.setAdoptionFlag(formatting, adoptionReturnAtSplit)
	}
	splitNode := block
	if len(wrappers) > 0 {
		var outer *types.Node
		var container *types.Node
		clones := make([]*types.Node, 0, len(wrappers))
		for _, template := range wrappers {
			clone := cloneFormattingNode(template, container)
			clone.StartPos, clone.StartLine, clone.StartColumn = 0, 0, 0
			clone.ContentStart = 0
			if outer == nil {
				outer = clone
			}
			if container != nil {
				container.Children = append(container.Children, clone)
				p.markTextDirty(container)
			}
			container = clone
			clones = append(clones, clone)
			if entry := p.activeFormattingEntryByName(template.Name); entry != nil && entry.current == nil {
				entry.current = clone
				p.formattingByNode[clone] = entry
			}
		}
		block.Parent = container
		container.Children = append(container.Children, block)
		p.markTextDirty(container)
		// The outer clone is a virtual child of the formatting element's
		// original parent until the deferred split is materialized.  Record
		// that relationship before registering the clones as open so their
		// insertion containers are derived from the final chain.
		outer.Parent = formatting.Parent
		for _, clone := range clones {
			p.setFormattingOpen(clone, true)
		}
		splitNode = outer
	} else {
		block.Parent = formatting.Parent
	}
	p.adoptionSplit[formatting] = append(p.adoptionSplit[formatting], splitNode)
	p.setAdoptionFlag(block, adoptionDetached)
	p.recoverOpenElementsAtEOF = true
	return adoptionContinue, nil
}

func (p *HTMLParser) extendIncompleteAdoptionAtEOF(node *types.Node, tokenStart int) {
	if node == nil {
		return
	}
	if len(node.Children) > 0 {
		last := node.Children[len(node.Children)-1]
		if last.Type == types.ElementNode {
			p.extendIncompleteAdoptionAtEOF(last, tokenStart)
		} else if last.Type == types.TextNode && last.EndPos == tokenStart {
			last.EndPos = p.pos
			last.EndLine = p.line
			last.EndColumn = p.col
		}
	}
	if node.EndPos == 0 || node.EndPos == tokenStart {
		node.ContentEnd = p.pos
		node.EndPos = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
	}
	p.markTextDirty(node)
	p.refreshNodeTextContent(node)
}

func (p *HTMLParser) cachedAdoptionFurthestBlock(formatting, current *types.Node) *types.Node {
	if p.adoptionUnwindPos == p.pos && p.adoptionUnwindFormatting == formatting {
		return p.adoptionUnwindBlock
	}
	block := adoptionFurthestBlock(formatting, current)
	p.adoptionUnwindPos = p.pos
	p.adoptionUnwindFormatting = formatting
	p.adoptionUnwindBlock = block
	return block
}

func (p *HTMLParser) cachedAdoptionInScope(formatting, current *types.Node) bool {
	if p.adoptionUnwindPos == p.pos && p.adoptionUnwindFormatting == formatting {
		return p.adoptionUnwindInScope
	}
	inScope := formattingInOrdinaryScope(formatting, current)
	p.adoptionUnwindPos = p.pos
	p.adoptionUnwindFormatting = formatting
	p.adoptionUnwindBlock = adoptionFurthestBlock(formatting, current)
	p.adoptionUnwindInScope = inScope
	return inScope
}

func adoptionGenericEndTarget(current *types.Node, name string) *types.Node {
	for node := current; node != nil; node = node.Parent {
		if node.Name == name {
			return node
		}
		if isHTMLSpecialElement(node.Name) {
			return nil
		}
	}
	return nil
}

func (p *HTMLParser) cachedGenericEndTarget(current *types.Node, name string) *types.Node {
	if p.genericEndPos == p.pos && p.genericEndName == name {
		return p.genericEndTarget
	}
	var target *types.Node
	if isParagraphClosingAncestorEndTag(name) {
		target = nearestOpenElementByName(current, name)
		if target != nil && !formattingInOrdinaryScope(target, current) {
			target = nil
		}
	} else {
		target = adoptionGenericEndTarget(current, name)
	}
	p.genericEndPos = p.pos
	p.genericEndName = name
	p.genericEndTarget = target
	return target
}

func (p *HTMLParser) removeActiveFormattingEntry(entry *formattingEntry) {
	if entry == nil {
		return
	}
	p.setFormattingOpen(entry.current, false)
	p.removeFormattingFamilyEntry(entry)
	for index := len(p.activeFormatting) - 1; index >= 0; index-- {
		if p.activeFormatting[index] == entry {
			p.activeFormatting = append(p.activeFormatting[:index], p.activeFormatting[index+1:]...)
			p.finishFormattingDocumentEndRecovery()
			return
		}
	}
}

func (p *HTMLParser) adoptionInnerFormatting(formatting, block *types.Node, closeStart, closeLine, closeColumn int) []*types.Node {
	var insideOut []*types.Node
	distance := 0
	for node := block.Parent; node != nil && node != formatting; node = node.Parent {
		distance++
		entry := p.formattingByNode[node]
		if entry == nil {
			node.ContentEnd = closeStart
			node.EndPos = closeStart
			node.EndLine = closeLine
			node.EndColumn = closeColumn
			p.setAdoptionFlag(node, adoptionClosed)
			p.adoptionIgnoredEnds[node.Name]++
			p.setFormattingOpen(node, false)
			continue
		}
		if distance > 3 {
			p.refreshFormattingText(node)
			node.ContentEnd = closeStart
			node.EndPos = closeStart
			node.EndLine = closeLine
			node.EndColumn = closeColumn
			p.setAdoptionFlag(node, adoptionClosed)
			p.adoptionIgnoredEnds[node.Name]++
			p.removeActiveFormatting(node)
			continue
		}
		insideOut = append(insideOut, node)
		// The source recursive frame is removed from the open-elements stack by
		// the adoption inner loop. Its active-formatting entry survives with a
		// replacement identity, but the stale source frame must return when the
		// furthest block finishes instead of waiting for an end tag that will be
		// consumed by the reconstructed clone.
		p.setFormattingOpen(node, false)
		p.setAdoptionFlag(node, adoptionReturnAtSplit)
		end := node.ContentStart
		if end == 0 && len(node.Children) > 0 {
			end = node.Children[0].StartPos
		}
		node.ContentEnd = end
		node.EndPos = end
		if contentStart := p.elementContentStart[node]; contentStart.line != 0 {
			node.EndLine = contentStart.line
			node.EndColumn = contentStart.column
		} else if len(node.Children) > 0 {
			node.EndLine = node.Children[0].StartLine
			node.EndColumn = node.Children[0].StartColumn
		}
		delete(p.formattingText, node)
		delete(p.formattingByNode, node)
		entry.current = nil
	}
	wrappers := make([]*types.Node, len(insideOut))
	for index := range insideOut {
		wrappers[len(insideOut)-1-index] = insideOut[index]
	}
	for node := block.Parent; node != nil && node != formatting; node = node.Parent {
		if node.Type != types.ElementNode || isFormattingElement(node.Name) {
			continue
		}
		node.ContentEnd = closeStart
		node.EndPos = closeStart
		node.EndLine = closeLine
		node.EndColumn = closeColumn
		p.setAdoptionFlag(node, adoptionClosed)
	}
	return wrappers
}

func adoptionSpecialDescendants(node *types.Node) []*types.Node {
	var descendants []*types.Node
	var visit func(*types.Node)
	visit = func(parent *types.Node) {
		for _, child := range parent.Children {
			if child.Type != types.ElementNode {
				continue
			}
			visit(child)
			if isHTMLSpecialElement(child.Name) {
				descendants = append(descendants, child)
			}
		}
	}
	visit(node)
	return descendants
}

func (p *HTMLParser) consumePendingAdoptionEnd(node *types.Node) (bool, error) {
	pending := p.adoptionPendingEnds[node]
	if len(pending) == 0 {
		return false, nil
	}
	name, closing, ok := p.peekTagName()
	if !ok || !closing {
		return false, nil
	}
	match := -1
	for index, candidate := range pending {
		if name == candidate.Name {
			match = index
			break
		}
	}
	if match < 0 {
		return false, nil
	}
	tokenStart := p.pos
	closed := pending[match]
	_, emitted, err := p.parseClosingTag()
	if err != nil {
		return true, err
	}
	if !emitted {
		p.extendPreviousTextRange(closed, tokenStart)
		return true, nil
	}
	closed.ContentEnd = tokenStart
	closed.EndPos = p.pos
	closed.EndLine = p.line
	closed.EndColumn = p.col
	p.markTextDirty(closed)
	p.refreshNodeTextContent(closed)
	p.adoptionPendingEnds[node] = pending[match+1:]
	return true, nil
}

func (p *HTMLParser) finishPendingAdoptionAtEOF(node *types.Node) {
	pending := p.adoptionPendingEnds[node]
	if len(pending) == 0 || p.pos < len(p.content) {
		return
	}
	for _, current := range pending {
		current.ContentEnd = p.pos
		current.EndPos = p.pos
		current.EndLine = p.line
		current.EndColumn = p.col
	}
	delete(p.adoptionPendingEnds, node)
	p.markTextDirty(node)
	p.refreshTreeText(node)
}

func (p *HTMLParser) adoptionWrapSpecialContents(block, formatting *types.Node, wrappers []*types.Node, closeEnd, closeLine, closeColumn, remaining int) {
	original := block.Children
	block.Children = nil
	p.markTextDirty(block)
	segment := make([]*types.Node, 0, len(original))
	emitEmpty := len(original) == 0
	flush := func() {
		if len(segment) == 0 && !emitEmpty {
			return
		}
		emitEmpty = false
		formatClone := cloneFormattingNode(formatting, block)
		formatClone.StartPos, formatClone.StartLine, formatClone.StartColumn = 0, 0, 0
		formatClone.ContentStart = 0
		formatClone.Children = segment
		var parent = formatClone
		for _, wrapper := range wrappers {
			clone := cloneFormattingNode(wrapper, parent)
			clone.ContentEnd = closeEnd
			clone.EndPos = closeEnd
			clone.EndLine = closeLine
			clone.EndColumn = closeColumn
			clone.Children = parent.Children
			for _, child := range clone.Children {
				child.Parent = clone
			}
			parent.Children = []*types.Node{clone}
			parent = clone
		}
		for _, child := range formatClone.Children {
			child.Parent = formatClone
		}
		p.markTextDirty(formatClone)
		p.refreshTreeText(formatClone)
		if len(wrappers) > 0 {
			formatClone.StartPos = formatting.StartPos
			formatClone.StartLine = formatting.StartLine
			formatClone.StartColumn = formatting.StartColumn
			formatClone.ContentStart = formatting.ContentStart
			formatClone.EndPos = closeEnd
			formatClone.EndLine = closeLine
			formatClone.EndColumn = closeColumn
		}
		block.Children = append(block.Children, formatClone)
		p.markTextDirty(block)
		segment = nil
	}
	for _, child := range original {
		if remaining > 1 && child.Type == types.ElementNode && isHTMLSpecialElement(child.Name) {
			flush()
			p.adoptionWrapSpecialContents(child, formatting, nil, closeEnd, closeLine, closeColumn, remaining-1)
			child.Parent = block
			block.Children = append(block.Children, child)
			p.markTextDirty(block)
			continue
		}
		segment = append(segment, child)
	}
	flush()
}

func (p *HTMLParser) detachNodeShallow(node *types.Node) {
	if node == nil || node.Parent == nil {
		return
	}
	parent := node.Parent
	for index, child := range parent.Children {
		if child == node {
			parent.Children = append(parent.Children[:index], parent.Children[index+1:]...)
			break
		}
	}
	node.Parent = nil
	p.markTextDirty(parent)
}

func formattingInOrdinaryScope(formatting, current *types.Node) bool {
	for node := current; node != nil; node = node.Parent {
		if node == formatting {
			return true
		}
		if isOrdinaryScopeBoundary(node.Name) {
			return false
		}
	}
	return false
}

func adoptionFurthestBlock(formatting, current *types.Node) *types.Node {
	var block *types.Node
	for node := current; node != nil && node != formatting; node = node.Parent {
		if isHTMLSpecialElement(node.Name) {
			block = node
		}
	}
	return block
}

func (p *HTMLParser) appendDeferredAdoption(parent, child *types.Node) {
	siblings := p.adoptionSplit[child]
	if len(siblings) == 0 {
		return
	}
	delete(p.adoptionSplit, child)
	if p.closedForms[parent] {
		p.deferredFormSiblings[parent] = append(p.deferredFormSiblings[parent], siblings...)
		return
	}
	for _, sibling := range siblings {
		if sibling.Parent != nil && sibling.Parent.StartLine == 0 {
			if split := p.adoptionSplit[sibling.Parent]; len(split) > 0 {
				delete(p.adoptionSplit, sibling.Parent)
				for _, nested := range split {
					p.appendUniqueChild(parent, nested)
					p.appendNestedDeferredAdoption(parent, nested)
				}
				continue
			}
		}
		container := parent
		for _, entry := range p.activeFormatting[p.formattingMarkerStart:] {
			if entry.marker || entry.current == nil || entry.current == sibling || !p.openFormatting[entry.current] || p.formattingContainer[entry.current] != parent {
				continue
			}
			container = entry.current
		}
		p.appendUniqueChild(container, sibling)
		p.appendDeferredAdoption(parent, sibling)
		p.appendNestedDeferredAdoption(parent, sibling)
	}
}

func (p *HTMLParser) appendDeferredFormSiblings(parent, form *types.Node) {
	siblings := p.deferredFormSiblings[form]
	if len(siblings) == 0 {
		return
	}
	delete(p.deferredFormSiblings, form)
	delete(p.closedForms, form)
	for _, sibling := range siblings {
		if sibling.Parent != nil {
			p.detachNodeShallow(sibling)
		}
		p.appendUniqueChild(parent, sibling)
		p.appendNestedDeferredAdoption(parent, sibling)
	}
	p.markTextDirty(form)
	p.refreshNodeTextContent(form)
}

func (p *HTMLParser) appendAllDeferredAdoption(parent *types.Node) {
	keys := make([]*types.Node, 0, len(p.adoptionSplit))
	for key := range p.adoptionSplit {
		keys = append(keys, key)
	}
	for _, key := range keys {
		siblings, present := p.adoptionSplit[key]
		if !present {
			continue
		}
		if len(siblings) == 0 {
			continue
		}
		delete(p.adoptionSplit, key)
		for _, sibling := range siblings {
			if sibling.Parent != nil {
				p.detachNodeShallow(sibling)
			}
			p.appendUniqueChild(parent, sibling)
			p.appendNestedDeferredAdoption(parent, sibling)
		}
	}
}

func (p *HTMLParser) appendUniqueChild(parent, child *types.Node) {
	if child.Parent != parent {
		child.Parent = parent
		parent.Children = append(parent.Children, child)
		p.markTextDirty(parent)
		return
	}
	if slices.Contains(parent.Children, child) {
		child.Parent = parent
		return
	}
	child.Parent = parent
	parent.Children = append(parent.Children, child)
	p.markTextDirty(parent)
}

func (p *HTMLParser) detachDeferredAdoption(node *types.Node) {
	for owner, siblings := range p.adoptionSplit {
		for index, sibling := range siblings {
			if sibling == node {
				p.adoptionSplit[owner] = append(siblings[:index], siblings[index+1:]...)
				if len(p.adoptionSplit[owner]) == 0 {
					delete(p.adoptionSplit, owner)
				}
				return
			}
			// If the block lives inside an already queued synthetic wrapper,
			// keep that wrapper queued. Detaching the block below naturally
			// leaves the empty wrapper chain in the old output slot; the new
			// split for the adopted formatting element will emit the block
			// immediately afterwards.
			if nodeWithin(node, sibling) {
				return
			}
		}
	}
}

func (p *HTMLParser) appendNestedDeferredAdoption(parent, node *types.Node) {
	for _, child := range append([]*types.Node(nil), node.Children...) {
		if len(p.adoptionSplit[child]) > 0 {
			p.appendDeferredAdoption(parent, child)
			continue
		}
		p.appendNestedDeferredAdoption(parent, child)
	}
}

func isFormattingMarkerElement(name string) bool {
	switch name {
	case "applet", "object", "marquee", "template", "td", "th", "caption":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) addActiveFormatting(node *types.Node) {
	key := formattingFamilyKey(node)
	family := p.formattingFamilies[key]
	if len(family) >= 3 {
		evicted := family[0]
		family = family[1:]
		for index := len(p.activeFormatting) - 1; index >= 0; index-- {
			if p.activeFormatting[index] == evicted {
				p.activeFormatting = append(p.activeFormatting[:index], p.activeFormatting[index+1:]...)
				break
			}
		}
		p.removeFormattingNameAndNodeEntry(evicted)
	}
	entry := &formattingEntry{template: node, current: node, family: key, index: len(p.activeFormatting)}
	p.activeFormatting = append(p.activeFormatting, entry)
	p.formattingFamilies[key] = append(family, entry)
	p.formattingByName[node.Name] = append(p.formattingByName[node.Name], entry)
	p.formattingByNode[node] = entry
}

func formattingFamilyKey(node *types.Node) string {
	if node == nil {
		return ""
	}
	if len(node.Attributes) == 0 {
		return node.Name
	}
	if len(node.Attributes) == 1 {
		for name, value := range node.Attributes {
			return node.Name + "\x00" + name + "=" + value
		}
	}
	names := make([]string, 0, len(node.Attributes))
	for name := range node.Attributes {
		names = append(names, name)
	}
	sort.Strings(names)
	var key strings.Builder
	key.WriteString(node.Name)
	for _, name := range names {
		key.WriteByte(0)
		key.WriteString(name)
		key.WriteByte('=')
		key.WriteString(node.Attributes[name])
	}
	return key.String()
}

func (p *HTMLParser) clearActiveFormattingToMarker() {
	for len(p.activeFormatting) > 0 {
		last := len(p.activeFormatting) - 1
		marker := p.activeFormatting[last].marker
		if !marker {
			p.removeFormattingFamilyEntry(p.activeFormatting[last])
		}
		p.activeFormatting = p.activeFormatting[:last]
		if marker {
			p.finishFormattingDocumentEndRecovery()
			return
		}
	}
	p.finishFormattingDocumentEndRecovery()
}

func (p *HTMLParser) removeActiveFormatting(node *types.Node) {
	entry := p.formattingByNode[node]
	if entry == nil {
		return
	}
	p.setFormattingOpen(entry.current, false)
	p.removeFormattingFamilyEntry(entry)
	if last := len(p.activeFormatting) - 1; last >= 0 && p.activeFormatting[last] == entry {
		p.activeFormatting = p.activeFormatting[:last]
		p.finishFormattingDocumentEndRecovery()
		return
	}
	for index := len(p.activeFormatting) - 1; index >= 0; index-- {
		if p.activeFormatting[index] == entry {
			p.activeFormatting = append(p.activeFormatting[:index], p.activeFormatting[index+1:]...)
			p.finishFormattingDocumentEndRecovery()
			return
		}
	}
}

func (p *HTMLParser) finishFormattingDocumentEndRecovery() {
	if !p.formattingDocumentEnd || len(p.activeFormatting) > 0 {
		return
	}
	p.formattingDocumentEnd = false
	p.recoverOpenElementsAtEOF = p.formattingDocumentEndPrior
	p.formattingDocumentEndPrior = false
	if p.hasActiveListItem() || p.hasActiveSelectTree() || p.adoptionEOFBoundary != nil {
		p.recoverOpenElementsAtEOF = true
	}
}

func (p *HTMLParser) removeFormattingFamilyEntry(entry *formattingEntry) {
	if entry == nil || entry.family == "" {
		return
	}
	family := p.formattingFamilies[entry.family]
	for index, candidate := range family {
		if candidate == entry {
			family = append(family[:index], family[index+1:]...)
			break
		}
	}
	if len(family) == 0 {
		delete(p.formattingFamilies, entry.family)
	} else {
		p.formattingFamilies[entry.family] = family
	}
	p.removeFormattingNameAndNodeEntry(entry)
}

func (p *HTMLParser) removeFormattingNameAndNodeEntry(entry *formattingEntry) {
	name := entry.template.Name
	byName := p.formattingByName[name]
	if last := len(byName) - 1; last >= 0 && byName[last] == entry {
		byName = byName[:last]
	} else {
		for index, candidate := range byName {
			if candidate == entry {
				byName = append(byName[:index], byName[index+1:]...)
				break
			}
		}
	}
	if len(byName) == 0 {
		delete(p.formattingByName, name)
	} else {
		p.formattingByName[name] = byName
	}
	delete(p.formattingByNode, entry.template)
	if entry.current != nil {
		delete(p.formattingByNode, entry.current)
	}
}

func cloneFormattingNode(template, parent *types.Node) *types.Node {
	attributes := make(map[string]string, len(template.Attributes))
	for name, value := range template.Attributes {
		attributes[name] = value
	}
	attributeNamespaces := cloneStringMap(template.AttributeNamespaces)
	attributeLocalNames := cloneStringMap(template.AttributeLocalNames)
	attributePrefixes := cloneStringMap(template.AttributePrefixes)
	return &types.Node{
		Type: types.ElementNode, Name: template.Name, NamespaceURI: template.NamespaceURI, Attributes: attributes,
		AttributeNamespaces: attributeNamespaces, AttributeLocalNames: attributeLocalNames, AttributePrefixes: attributePrefixes,
		AttributeOrder: append([]string(nil), template.AttributeOrder...), Children: []*types.Node{}, Parent: parent,
		StartPos: template.StartPos, StartLine: template.StartLine, StartColumn: template.StartColumn,
		ContentStart: template.ContentStart,
	}
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func nodeWithin(node, ancestor *types.Node) bool {
	for current := node; current != nil; current = current.Parent {
		if current == ancestor {
			return true
		}
	}
	return false
}

func (p *HTMLParser) formattingInsertionParent(container *types.Node, reconstruct bool) *types.Node {
	if p.documentBody != nil && (container == p.documentHTML || container == p.documentRoot) && (reconstruct || p.hasOpenFormatting(p.documentBody)) {
		container = p.documentBody
	}
	if !reconstruct {
		if current := p.lastOpenFormattingNode(); current != nil && p.formattingContainer[current] == container {
			return current
		}
		return container
	}
	parent := container
	start := p.formattingMarkerStart
	lastOpen := start - 1
	if openNode := p.lastOpenFormattingNode(); openNode != nil {
		if p.formattingContainer[openNode] == container {
			parent = openNode
		}
		if entry := p.formattingByNode[openNode]; entry != nil {
			if entry.index >= start && entry.index < len(p.activeFormatting) && p.activeFormatting[entry.index] == entry {
				lastOpen = entry.index
			} else {
				for index := len(p.activeFormatting) - 1; index >= start; index-- {
					if p.activeFormatting[index] == entry {
						entry.index = index
						lastOpen = index
						break
					}
				}
			}
		}
	}
	for index := lastOpen + 1; index < len(p.activeFormatting); index++ {
		entry := p.activeFormatting[index]
		clone := cloneFormattingNode(entry.template, parent)
		parent.Children = append(parent.Children, clone)
		if entry.current != nil {
			delete(p.formattingByNode, entry.current)
		}
		entry.current = clone
		p.formattingByNode[clone] = entry
		p.setFormattingOpen(clone, true)
		parent = clone
	}
	return parent
}

func (p *HTMLParser) nextTokenReconstructsFormatting() bool {
	if p.pos >= len(p.content) || p.nextTokenIsComment() || p.nextTokenIsIgnoredDoctype() {
		return false
	}
	if p.peek() != '<' || !p.startsMarkupToken() {
		preview := *p
		node, _ := preview.parseTextNode(nil, preview.pos, preview.line, preview.col)
		return node != nil
	}
	name, closing, ok := p.peekTagName()
	if !ok || !p.currentTagTokenWillEmit() {
		return false
	}
	if closing {
		return name == "br"
	}
	// Batch 14A reconstructs on ordinary/generic and formatting starts. The
	// special block starts that only change the open-element stack wait for a
	// later character token, matching the in-body reconstruction step.
	if isFormattingElement(name) || !isHTMLSpecialElement(name) {
		return true
	}
	switch name {
	case "applet", "object", "marquee", "button", "area", "br", "embed", "img", "image", "input", "keygen", "wbr", "xmp", "select", "option", "optgroup", "math", "svg":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) hasOpenFormatting(container *types.Node) bool {
	current := p.lastOpenFormattingNode()
	return current != nil && p.formattingContainer[current] == container
}

func (p *HTMLParser) setFormattingOpen(node *types.Node, open bool) {
	if node == nil {
		return
	}
	if open {
		if !p.openFormatting[node] {
			p.openFormatting[node] = true
			if parent := node.Parent; parent != nil && p.openFormatting[parent] {
				p.formattingContainer[node] = p.formattingContainer[parent]
			} else {
				p.formattingContainer[node] = node.Parent
			}
			p.formattingOpenStack = append(p.formattingOpenStack, node)
			p.formattingTable[node] = p.activeTable
		}
		return
	}
	delete(p.openFormatting, node)
	delete(p.formattingContainer, node)
	delete(p.formattingTable, node)
}

func (p *HTMLParser) lastOpenFormattingNode() *types.Node {
	for len(p.formattingOpenStack) > 0 {
		last := len(p.formattingOpenStack) - 1
		node := p.formattingOpenStack[last]
		if p.openFormatting[node] {
			return node
		}
		p.formattingOpenStack = p.formattingOpenStack[:last]
	}
	return nil
}

func (p *HTMLParser) refreshOpenFormattingText() {
	if node := p.lastOpenFormattingNode(); node != nil {
		p.refreshFormattingChain(node)
	}
}

func (p *HTMLParser) refreshFormattingChain(parent *types.Node) {
	path := make([]*types.Node, 0, 8)
	for node := parent; node != nil; node = node.Parent {
		if !p.openFormatting[node] {
			break
		}
		path = append(path, node)
	}
	for _, node := range path {
		p.refreshFormattingText(node)
	}
}

func (p *HTMLParser) refreshFormattingText(node *types.Node) {
	if builder := p.formattingText[node]; builder != nil {
		node.TextContent = builder.String()
		delete(p.formattingText, node)
		delete(p.dirtyTextNodes, node)
	} else {
		p.refreshNodeTextContent(node)
	}
	if len(node.Children) > 0 {
		last := node.Children[len(node.Children)-1]
		if node.StartLine != 0 && last.EndPos > node.EndPos {
			node.ContentEnd = last.EndPos
			node.EndPos = last.EndPos
			node.EndLine = last.EndLine
			node.EndColumn = last.EndColumn
		}
	}
}

func (p *HTMLParser) appendFormattingChild(container, parent, child *types.Node) {
	if parent == container {
		return
	}
	if child.Type == types.TextNode && len(parent.Children) > 0 && parent.Children[len(parent.Children)-1].Type == types.TextNode {
		mergeTextNodes(parent.Children[len(parent.Children)-1], child)
	} else {
		parent.Children = append(parent.Children, child)
	}
	p.markTextDirty(parent)
	value := child.TextContent
	switch child.Type {
	case types.TextNode:
		value = child.Value
	case types.CommentNode, types.ProcessingInstructionNode:
		value = ""
	}
	if value != "" {
		builder := p.formattingText[parent]
		if builder == nil {
			builder = &strings.Builder{}
			builder.WriteString(parent.TextContent)
			p.formattingText[parent] = builder
		}
		builder.WriteString(value)
	}
	if parent.StartLine != 0 {
		parent.ContentEnd = child.EndPos
		parent.EndPos = child.EndPos
		parent.EndLine = child.EndLine
		parent.EndColumn = child.EndColumn
	}
}

func (p *HTMLParser) finishFormattingWithin(container *types.Node, end, line, column int) {
	start := p.formattingOpenIndexWithin(container)
	for index := len(p.formattingOpenStack) - 1; index >= start; index-- {
		current := p.formattingOpenStack[index]
		if current == container || !p.openFormatting[current] {
			continue
		}
		p.refreshFormattingText(current)
		current.ContentEnd = end
		current.EndPos = end
		current.EndLine = line
		current.EndColumn = column
	}
}

func (p *HTMLParser) closeFormattingOnElementReturn(container *types.Node) {
	end := container.ContentEnd
	if end == 0 {
		end = container.EndPos
	}
	start := p.formattingOpenIndexWithin(container)
	for index := len(p.formattingOpenStack) - 1; index >= start; index-- {
		current := p.formattingOpenStack[index]
		if !p.openFormatting[current] {
			continue
		}
		entry := p.formattingByNode[current]
		if entry == nil {
			continue
		}
		// Adoption may leave a locationless formatting clone logically open
		// around the special element that is returning.  The clone must keep
		// owning later siblings until its own formatting end tag is handled.
		if current.StartLine == 0 && current != container && nodeWithin(container, current) {
			continue
		}
		if current != container {
			p.refreshFormattingText(current)
			if current.EndPos != 0 {
				entry.current = nil
				p.setFormattingOpen(current, false)
				continue
			}
			current.ContentEnd = end
			current.EndPos = end
			current.EndLine = container.EndLine
			current.EndColumn = container.EndColumn
		}
		entry.current = nil
		p.setFormattingOpen(current, false)
	}
}

func (p *HTMLParser) formattingOpenIndexWithin(container *types.Node) int {
	if p.lastOpenFormattingNode() == nil {
		return len(p.formattingOpenStack)
	}

	// Logically open formatting elements form a suffix of the open-formatting
	// stack for their nearest non-formatting container. Walk that live suffix
	// backwards and stop at the first open entry owned outside the returning
	// container. This avoids both a whole-stack scan and a walk through a deeply
	// reconstructed formatting DOM chain on every table row/cell return.
	start := len(p.formattingOpenStack)
	for index := len(p.formattingOpenStack) - 1; index >= 0; index-- {
		current := p.formattingOpenStack[index]
		if !p.openFormatting[current] {
			continue
		}
		base := p.formattingContainer[current]
		ownerTable := p.formattingTable[current]
		within := current == container || base == container
		if !within && ownerTable != nil && container == ownerTable {
			within = true
		} else if !within && ownerTable != nil && nodeWithin(container, ownerTable) {
			// Both nodes belong to the same table construction scope. Stop at
			// that table rather than following fostered/reconstructed ancestors
			// into arbitrarily deep prior document content.
			within = nodeWithinBefore(base, container, ownerTable)
		} else if !within {
			within = nodeWithin(base, container)
		}
		if !within {
			break
		}
		start = index
	}
	return start
}

func nodeWithinBefore(node, ancestor, boundary *types.Node) bool {
	for current := node; current != nil; current = current.Parent {
		if current == ancestor {
			return true
		}
		if current == boundary {
			return false
		}
	}
	return false
}

func (p *HTMLParser) formattingEntryForNode(node *types.Node) *formattingEntry {
	return p.formattingByNode[node]
}

func (p *HTMLParser) activeFormattingEntryByName(name string) *formattingEntry {
	entries := p.formattingByName[name]
	if len(entries) > 0 {
		return entries[len(entries)-1]
	}
	return nil
}
