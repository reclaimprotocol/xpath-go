// In-body list, select, button, and paragraph insertion-mode recovery.
package utils

import (
	"fmt"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func isListStartScanStopper(name string) bool {
	// Current Chromium and parse5 continue the list-item scan through search,
	// despite the Living Standard's newer special-element classification.
	if name == "address" || name == "div" || name == "p" || name == "search" {
		return false
	}
	return isHTMLSpecialElement(name)
}

func isOrdinaryScopeBoundary(name string) bool {
	switch name {
	case "applet", "caption", "html", "marquee", "object", "select", "table", "td", "th", "template":
		return true
	default:
		return false
	}
}

func isListItemScopeBoundary(name string) bool {
	return isOrdinaryScopeBoundary(name) || name == "ol" || name == "ul"
}

func isHTMLSpecialElement(name string) bool {
	switch name {
	case "address", "applet", "area", "article", "aside", "base", "basefont", "bgsound", "blockquote", "body", "br", "button", "caption", "center", "col", "colgroup", "dd", "details", "dir", "div", "dl", "dt", "embed", "fieldset", "figcaption", "figure", "footer", "form", "frame", "frameset", "h1", "h2", "h3", "h4", "h5", "h6", "head", "header", "hgroup", "hr", "html", "iframe", "img", "input", "li", "link", "listing", "main", "marquee", "menu", "meta", "nav", "noembed", "noframes", "noscript", "object", "ol", "p", "param", "plaintext", "pre", "script", "search", "section", "select", "source", "style", "summary", "table", "tbody", "td", "template", "textarea", "tfoot", "th", "thead", "title", "tr", "track", "ul", "wbr", "xmp":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) hasActiveListItem() bool {
	return p.activeLIEnd != nil || p.activeDDEnd != nil || p.activeDTEnd != nil
}

func (p *HTMLParser) currentListDocumentEndTag() bool {
	name, closing, ok := p.peekTagName()
	// Body/html end tags change the document insertion mode without popping
	// still-open list items or active formatting elements.  Keep those virtual
	// open stacks alive so later after-body text can be reprocessed in body and
	// EOF can finish their browser ranges.
	activeDocumentEnd := p.listDocumentEndMode != listDocumentEndNone || p.hasActiveListItem() || len(p.activeFormatting) > 0 ||
		(name == "html" && p.documentBodyBoundary.end.pos > 0)
	return activeDocumentEnd && ok && closing && (name == "body" || name == "html") && p.currentTagTokenWillEmit()
}

func (p *HTMLParser) currentTokenReentersListBody() bool {
	if p.pos >= len(p.content) || p.nextTokenIsComment() {
		return false
	}
	remaining := p.content[p.pos:]
	if len(remaining) >= len("<!DOCTYPE") && strings.EqualFold(remaining[:len("<!DOCTYPE")], "<!DOCTYPE") {
		return false
	}
	if p.peek() == '<' && p.startsMarkupToken() {
		return true
	}
	preview := *p
	previewParent := &types.Node{Type: types.ElementNode, Name: "#preview", Children: []*types.Node{}}
	text, err := preview.parseTextNode(previewParent, preview.pos, preview.line, preview.col)
	if err != nil || text == nil {
		return true
	}
	return !isHTMLWhitespaceString(text.Value)
}

func (p *HTMLParser) nextTokenIsIgnoredDoctype() bool {
	remaining := p.content[p.pos:]
	return len(remaining) >= len("<!DOCTYPE") && strings.EqualFold(remaining[:len("<!DOCTYPE")], "<!DOCTYPE")
}

func (p *HTMLParser) nextTokenIsHTMLStart() bool {
	name, closing, ok := p.peekTagName()
	return ok && !closing && name == "html" && p.currentTagTokenWillEmit()
}

func (p *HTMLParser) mergeDuplicateHTMLStart() error {
	return p.mergeDuplicateStartInto(p.documentHTML, "html")
}

func (p *HTMLParser) mergeDuplicateStartInto(target *types.Node, expected string) error {
	if p.peek() != '<' {
		return nil
	}
	p.advance()
	name := p.parseTagName()
	if name != expected {
		return fmt.Errorf("expected %s start tag at position %d", expected, p.pos)
	}
	node := &types.Node{Type: types.ElementNode, Name: name, Attributes: make(map[string]string), AttributeOrder: []string{}}
	_, emitted := p.parseStartTagTail(node)
	if !emitted {
		return nil
	}
	if target == nil {
		return nil
	}
	if target.AttributeNamespaces == nil {
		target.AttributeNamespaces = make(map[string]string)
	}
	if target.AttributeLocalNames == nil {
		target.AttributeLocalNames = make(map[string]string)
	}
	if target.AttributePrefixes == nil {
		target.AttributePrefixes = make(map[string]string)
	}
	for _, name := range node.AttributeOrder {
		if _, exists := target.Attributes[name]; exists {
			continue
		}
		target.Attributes[name] = node.Attributes[name]
		target.AttributeOrder = append(target.AttributeOrder, name)
		target.AttributeNamespaces[name] = ""
		target.AttributeLocalNames[name] = name
		target.AttributePrefixes[name] = ""
	}
	return nil
}

func isHeadingName(name string) bool {
	return len(name) == 2 && name[0] == 'h' && name[1] >= '1' && name[1] <= '6'
}

func nodeIsDescendantOrSelf(node, ancestor *types.Node) bool {
	for current := node; current != nil; current = current.Parent {
		if current == ancestor {
			return true
		}
	}
	return false
}

func nodeHasAncestorName(node *types.Node, name string) bool {
	for current := node.Parent; current != nil; current = current.Parent {
		if current.Name == name {
			return true
		}
	}
	return false
}

func nearestOpenElementByName(node *types.Node, name string) *types.Node {
	for current := node; current != nil; current = current.Parent {
		if current.Name == name {
			return current
		}
	}
	return nil
}

func (p *HTMLParser) listItemUnwindTarget(current *types.Node) (*types.Node, bool) {
	if p.listUnwindCache.position == p.pos && p.listUnwindCache.target != nil {
		if current == p.listUnwindCache.target {
			name, closing, ok := p.peekTagName()
			if ok && closing && name == current.Name && current != p.activeLIEnd && current != p.activeDDEnd && current != p.activeDTEnd {
				p.clearListUnwindCache()
				return nil, false
			}
		}
		return p.listUnwindCache.target, p.listUnwindCache.explicit
	}
	name, closing, ok := p.peekTagName()
	if !ok || !p.currentTagTokenWillEmit() {
		return nil, false
	}
	if p.activeLIStart == nil && p.activeDefinitionStart == nil && p.activeLIEnd == nil && p.activeDDEnd == nil && p.activeDTEnd == nil {
		return nil, false
	}
	var target *types.Node
	if !closing {
		switch name {
		case "li":
			target = p.activeLIStart
		case "dd", "dt":
			target = p.activeDefinitionStart
		default:
			return nil, false
		}
	} else {
		switch name {
		case "li":
			target = p.activeLIEnd
		case "dd":
			target = p.activeDDEnd
		case "dt":
			target = p.activeDTEnd
		default:
			if name == "body" || name == "html" {
				return nil, false
			}
			nearest := nearestOpenElementByName(current, name)
			if nearest == current {
				return nil, false
			}
			if nearest != nil {
				for _, candidate := range []*types.Node{p.activeLIEnd, p.activeDDEnd, p.activeDTEnd} {
					if candidate != nil && nodeIsDescendantOrSelf(nearest, candidate) {
						return p.cacheListUnwind(nearest, false)
					}
				}
			}
			if isHeadingName(name) {
				for currentNode := current; currentNode != nil; currentNode = currentNode.Parent {
					if !isHeadingName(currentNode.Name) {
						continue
					}
					for _, candidate := range []*types.Node{p.activeLIEnd, p.activeDDEnd, p.activeDTEnd} {
						if candidate != nil && nodeIsDescendantOrSelf(candidate, currentNode) {
							return p.cacheListUnwind(candidate, false)
						}
					}
					break
				}
			}
			if !isParagraphClosingAncestorEndTag(name) {
				return nil, false
			}
			if nearest != nil {
				for _, candidate := range []*types.Node{p.activeLIEnd, p.activeDDEnd, p.activeDTEnd} {
					if candidate == nil || !nodeIsDescendantOrSelf(current, candidate) {
						continue
					}
					if nodeIsDescendantOrSelf(candidate, nearest) && nearest != candidate {
						return p.cacheListUnwind(candidate, false)
					}
				}
			}
			for _, candidate := range []*types.Node{p.activeLIEnd, p.activeDDEnd, p.activeDTEnd} {
				if candidate != nil && nearestOpenElementByName(candidate.Parent, name) != nil && nodeIsDescendantOrSelf(current, candidate) {
					target = candidate
					break
				}
			}
			return p.cacheListUnwind(target, false)
		}
	}
	if target == nil {
		return nil, false
	}
	return p.cacheListUnwind(target, closing)
}

func (p *HTMLParser) cacheListUnwind(target *types.Node, explicit bool) (*types.Node, bool) {
	if target == nil {
		return nil, false
	}
	p.listUnwindCache = listUnwindCache{position: p.pos, target: target, explicit: explicit}
	return target, explicit
}

func (p *HTMLParser) clearListUnwindCache() {
	p.listUnwindCache = listUnwindCache{position: -1}
}

func (p *HTMLParser) selectTreeUnwindTarget(current *types.Node) (*types.Node, selectUnwindAction) {
	if p.selectUnwindCache.position == p.pos && p.selectUnwindCache.target != nil {
		return p.selectUnwindCache.target, p.selectUnwindCache.action
	}
	name, closing, ok := p.peekTagName()
	if !ok || !p.currentTagTokenWillEmit() {
		return nil, selectUnwindImplicit
	}
	var target *types.Node
	action := selectUnwindImplicit
	if closing {
		action = selectUnwindEnd
		switch name {
		case "select":
			target = p.activeSelect
		case "option":
			target = p.activeOptionEnd
		case "optgroup":
			target = p.activeOptgroupEnd
		default:
			if p.activeSelect != nil {
				if name == "li" {
					target = p.activeLIEnd
				} else if name == "dd" {
					target = p.activeDDEnd
				} else if name == "dt" {
					target = p.activeDTEnd
				} else if isHeadingName(name) {
					target = nearestHeadingWithinBoundary(current, p.activeSelect)
					if target != nil && target.Name != name {
						action = selectUnwindImplicit
					}
				} else {
					target = p.activeSelectNames[name]
					if isParagraphClosingAncestorEndTag(name) {
						target = p.activeSelectAllNames[name]
					}
				}
				if target == current {
					return nil, selectUnwindImplicit
				}
			}
		}
	} else {
		switch name {
		case "option":
			if p.activeSelect != nil {
				target = p.activeSelectOptionImplied
				if target == nil {
					target = p.activeOptionStart
				}
			} else if p.activeOptionStart == current {
				target = p.activeOptionStart
			}
		case "optgroup":
			if p.activeSelect != nil {
				target = p.activeSelectImpliedEnd
				if target == nil {
					target = p.activeOptionStart
				}
			} else if p.activeOptionStart == current {
				target = p.activeOptionStart
			}
			if target == nil && p.activeSelect != nil {
				target = p.activeOptgroupStart
			}
		case "select":
			target = p.activeSelect
			action = selectUnwindNestedSelect
		case "input":
			target = p.activeSelect
		case "hr":
			if p.activeSelect != nil {
				target = p.activeSelectImpliedEnd
				if target == nil {
					target = p.activeOptionStart
				}
				if target == nil {
					target = p.activeOptgroupStart
				}
			}
		}
	}
	if target == nil {
		return nil, selectUnwindImplicit
	}
	p.selectUnwindCache = selectUnwindCache{position: p.pos, target: target, action: action}
	return target, action
}

func (p *HTMLParser) clearSelectUnwindCache() {
	p.selectUnwindCache = selectUnwindCache{position: -1}
}

func (p *HTMLParser) preserveIgnoredTextRange(parent *types.Node, start, line, column int) {
	if len(parent.Children) > 0 {
		last := parent.Children[len(parent.Children)-1]
		if last.Type == types.TextNode {
			last.EndPos = p.pos
			last.EndLine = p.line
			last.EndColumn = p.col
			return
		}
	}
}

func (p *HTMLParser) currentIgnoredSelectEndTag(current *types.Node) bool {
	name, closing, ok := p.peekTagName()
	if !ok || !closing || !p.currentTagTokenWillEmit() {
		return false
	}
	if p.isSelfClosingTag(name) {
		return false
	}
	switch name {
	case "select":
		return p.activeSelect == nil
	case "option":
		return p.activeOptionEnd == nil
	case "optgroup":
		return p.activeOptgroupEnd == nil
	default:
		// p has its own in-body orphan-end recovery, including creation of a
		// locationless synthetic paragraph inside the select.
		if name == "p" || p.activeSelect == nil {
			return false
		}
		if isHeadingName(name) {
			return nearestHeadingWithinBoundary(current, p.activeSelect) == nil
		}
		if name == "li" {
			return p.activeLIEnd == nil
		}
		if name == "dd" {
			return p.activeDDEnd == nil
		}
		if name == "dt" {
			return p.activeDTEnd == nil
		}
		if isParagraphClosingAncestorEndTag(name) {
			return p.activeSelectAllNames[name] == nil
		}
		return p.activeSelectNames[name] == nil
	}
}

func nearestHeadingWithinBoundary(current, boundary *types.Node) *types.Node {
	for node := current; node != nil; node = node.Parent {
		if isHeadingName(node.Name) {
			return node
		}
		if node == boundary {
			break
		}
	}
	return nil
}

func (p *HTMLParser) consumeIgnoredStartTag() bool {
	if p.peek() != '<' {
		return false
	}
	p.advance()
	name := p.parseTagName()
	if name == "" {
		return false
	}
	temporary := &types.Node{Attributes: make(map[string]string)}
	_, emitted := p.parseStartTagTail(temporary)
	return emitted
}

func (p *HTMLParser) currentIgnoredListEndTag(current *types.Node) bool {
	name, closing, ok := p.peekTagName()
	if !ok || !closing || !p.currentTagTokenWillEmit() {
		return false
	}
	var target *types.Node
	switch name {
	case "li":
		target = p.activeLIEnd
	case "dd":
		target = p.activeDDEnd
	case "dt":
		target = p.activeDTEnd
	default:
		return false
	}
	return target == nil || !nodeIsDescendantOrSelf(current, target)
}

func (p *HTMLParser) listItemBlocksGenericAncestorEnd(current *types.Node, name string) bool {
	if isParagraphClosingAncestorEndTag(name) {
		return false
	}
	for _, target := range []*types.Node{p.activeLIEnd, p.activeDDEnd, p.activeDTEnd} {
		if target != nil && nodeIsDescendantOrSelf(current, target) && nodeHasAncestorName(target, name) {
			return true
		}
	}
	return false
}

func (p *HTMLParser) listItemIgnoresAbsentAncestorEnd(current *types.Node, name string) bool {
	if !isParagraphClosingAncestorEndTag(name) || !p.hasActiveListItem() {
		return false
	}
	if nearestOpenElementByName(current, name) != nil {
		return false
	}
	return true
}

func isParagraphButtonScopeBoundary(name string) bool {
	switch name {
	case "applet", "button", "caption", "html", "marquee", "object", "select", "table", "td", "th", "template":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) currentNestedButtonStart(current *types.Node) bool {
	if p.activeButton == nil || !nodeIsDescendantOrSelf(current, p.activeButton) {
		return false
	}
	name, closing, ok := p.peekTagName()
	return ok && !closing && name == "button" && p.currentTagTokenWillEmit()
}

func isParagraphAncestorScopeBoundary(name string) bool {
	switch name {
	case "applet", "caption", "html", "marquee", "object", "select", "table", "td", "th", "template":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) paragraphInButtonScope(current *types.Node) *types.Node {
	return p.activeParagraph
}

func isParagraphClosingStartTag(name string) bool {
	switch name {
	case "address", "article", "aside", "blockquote", "center", "details", "dialog", "dir", "div", "dl",
		"fieldset", "figcaption", "figure", "footer", "form", "header", "hgroup", "hr", "listing", "main",
		"menu", "nav", "ol", "p", "plaintext", "pre", "search", "section", "summary", "ul", "xmp",
		"dd", "dt", "li", "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) paragraphClosingStartTag(name string) bool {
	if name == "table" {
		return !p.quirksMode
	}
	return isParagraphClosingStartTag(name)
}

func isParagraphClosingAncestorEndTag(name string) bool {
	switch name {
	case "address", "applet", "article", "aside", "blockquote", "button", "center", "dd", "details", "dialog", "dir", "div", "dl", "dt",
		"fieldset", "figcaption", "figure", "footer", "form", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hgroup", "html", "li", "listing", "main", "marquee", "menu",
		"nav", "object", "ol", "pre", "search", "section", "summary", "ul", "body":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) shouldUnwindForParagraph(current *types.Node) bool {
	paragraph := p.paragraphInButtonScope(current)
	if paragraph == nil {
		return false
	}
	if !p.currentTagTokenWillEmit() {
		return false
	}

	name, closing, ok := p.peekTagName()
	if !ok {
		return false
	}
	if !closing {
		return p.paragraphClosingStartTag(name)
	}
	if name == "p" {
		return current != paragraph
	}
	if !isParagraphClosingAncestorEndTag(name) {
		return false
	}
	_, closesAncestor := p.activeParagraphAncestors[name]
	return closesAncestor
}
