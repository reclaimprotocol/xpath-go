package utils

import (
	"fmt"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func adjustSVGTagName(name string) string {
	if adjusted := svgTagNameAdjustments[name]; adjusted != "" {
		return adjusted
	}
	// parseTagName has already applied the tokenizer's ASCII-only folding.
	// Preserve non-ASCII code points exactly; Unicode lowercasing would change
	// foreign element identity in a way browsers do not.
	return name
}

func adjustSVGAttributes(node *types.Node) {
	if node == nil || len(node.AttributeOrder) == 0 {
		return
	}
	for index, oldName := range node.AttributeOrder {
		adjusted := svgAttributeAdjustments[oldName]
		if adjusted == "" || adjusted == oldName {
			continue
		}
		value := node.Attributes[oldName]
		delete(node.Attributes, oldName)
		node.Attributes[adjusted] = value
		node.AttributeOrder[index] = adjusted
	}
}

func adjustMathMLAttributes(node *types.Node) {
	if node == nil || len(node.AttributeOrder) == 0 {
		return
	}
	for index, oldName := range node.AttributeOrder {
		if oldName != "definitionurl" {
			continue
		}
		value := node.Attributes[oldName]
		delete(node.Attributes, oldName)
		node.Attributes["definitionURL"] = value
		node.AttributeOrder[index] = "definitionURL"
	}
}

const (
	xlinkAttributeNamespace = "http://www.w3.org/1999/xlink"
	xmlAttributeNamespace   = "http://www.w3.org/XML/1998/namespace"
	xmlnsAttributeNamespace = "http://www.w3.org/2000/xmlns/"
)

func initializeAttributeMetadata(node *types.Node, foreign bool) {
	if node == nil {
		return
	}
	node.AttributeNamespaces = make(map[string]string, len(node.AttributeOrder))
	node.AttributeLocalNames = make(map[string]string, len(node.AttributeOrder))
	node.AttributePrefixes = make(map[string]string, len(node.AttributeOrder))
	for _, name := range node.AttributeOrder {
		local, prefix, namespace := name, "", ""
		if foreign {
			switch name {
			case "xlink:actuate", "xlink:arcrole", "xlink:href", "xlink:role", "xlink:show", "xlink:title", "xlink:type":
				local, prefix, namespace = strings.TrimPrefix(name, "xlink:"), "xlink", xlinkAttributeNamespace
			case "xml:lang", "xml:space":
				local, prefix, namespace = strings.TrimPrefix(name, "xml:"), "xml", xmlAttributeNamespace
			case "xmlns":
				local, namespace = "xmlns", xmlnsAttributeNamespace
			case "xmlns:xlink":
				local, prefix, namespace = "xlink", "xmlns", xmlnsAttributeNamespace
			}
		}
		node.AttributeNamespaces[name] = namespace
		node.AttributeLocalNames[name] = local
		node.AttributePrefixes[name] = prefix
	}
}

func (p *HTMLParser) finishForeignSelfClosing(node *types.Node) {
	node.ContentStart = p.pos
	node.ContentEnd = p.pos
	node.EndPos = p.pos
	node.EndLine = p.line
	node.EndColumn = p.col
}

func (p *HTMLParser) parseSVGElement(node *types.Node, contentStart int) (*types.Node, error) {
	key := asciiLower(node.Name)
	previous, hadPrevious := p.activeSVGNames[key]
	p.activeSVGNames[key] = node
	defer func() {
		if hadPrevious {
			p.activeSVGNames[key] = previous
		} else {
			delete(p.activeSVGNames, key)
		}
	}()

	var text strings.Builder
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(node) {
			p.finishImplicitElement(node, text.String(), contentStart)
			return node, nil
		}
		if p.templateForeignTransition || (isForeignIntegrationOwner(node) && p.currentTemplateTableTransitionWillEmit(node)) {
			p.templateForeignTransition = true
			p.finishImplicitElement(node, text.String(), contentStart)
			return node, nil
		}
		if isForeignIntegrationOwner(node) {
			handled, err := p.processForeignIntegrationToken(node, &text)
			if err != nil {
				return nil, err
			}
			if handled {
				continue
			}
		}
		if p.content[p.pos] == '<' && p.pos+1 < len(p.content) && p.content[p.pos+1] == '/' && p.startsMarkupToken() {
			tokenStart, tokenLine, tokenColumn := p.pos, p.line, p.col
			if p.isMissingEndTagName() {
				p.advanceRune(3)
				p.extendPreviousTextRange(node, tokenStart)
				continue
			}
			if p.isBogusEndTagOpen() {
				comment, err := p.parseBogusEndTagComment(node, tokenStart, tokenLine, tokenColumn)
				if err != nil {
					return nil, err
				}
				p.appendSVGChild(node, comment, &text)
				continue
			}
			name, _, ok := p.peekTagName()
			if !ok {
				child, err := p.parseSVGText(node)
				if err != nil {
					return nil, err
				}
				p.appendSVGChild(node, child, &text)
				continue
			}
			if (name == "p" || name == "br") && p.currentEndTagTokenWillEmit() {
				if name == "br" {
					p.foreignBRReprocess = true
				}
				p.addForeignIgnoredEnd(key)
				p.finishElementAt(node, text.String(), contentStart, tokenStart, tokenLine, tokenColumn)
				return node, nil
			}
			foreignTarget := p.activeSVGNames[asciiLower(name)]
			if foreignTarget != nil && foreignTarget != node {
				p.addForeignIgnoredEnd(key)
				p.finishElementAt(node, text.String(), contentStart, tokenStart, tokenLine, tokenColumn)
				return node, nil
			}
			if p.foreignIgnoredEnds[name] > 0 && foreignTarget == nil && !asciiEqualFold(name, key) {
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
			closingName, emitted, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if !emitted {
				p.extendPreviousTextRange(node, tokenStart)
				p.recoverOpenElementsAtEOF = true
				p.finishImplicitElement(node, text.String(), contentStart)
				return node, nil
			}
			if asciiEqualFold(closingName, key) {
				node.TextContent = text.String()
				node.ContentStart = contentStart
				node.ContentEnd = tokenStart
				node.EndPos = p.pos
				node.EndLine = p.line
				node.EndColumn = p.col
				return node, nil
			}
			// Foreign end tags without a matching SVG ancestor are parse errors
			// that do not alter the tree. A complete ignored token does not extend
			// a terminal text node; if more text follows, normal coalescing spans
			// the ignored source gap.
			continue
		}

		startPos, startLine, startColumn := p.pos, p.line, p.col
		if p.svgStartTokenBreaksOut() {
			p.addForeignIgnoredEnd(key)
			p.finishElementAt(node, text.String(), contentStart, startPos, startLine, startColumn)
			return node, nil
		}
		switch {
		case strings.HasPrefix(p.content[p.pos:], "<!--"):
			comment, err := p.parseComment(node, startPos, startLine, startColumn)
			if err != nil {
				return nil, err
			}
			p.appendSVGChild(node, comment, &text)
		case strings.HasPrefix(p.content[p.pos:], "<![CDATA["):
			cdata := p.parseSVGCDATA(node, startPos, startLine, startColumn)
			p.appendSVGChild(node, cdata, &text)
		case hasASCIIPrefixFold(p.content[p.pos:], "<!DOCTYPE"):
			if _, err := p.parseDoctype(node, startPos, startLine, startColumn); err != nil {
				return nil, err
			}
			p.extendPreviousTextRange(node, startPos)
		case strings.HasPrefix(p.content[p.pos:], "<?"):
			instruction, err := p.parseProcessingInstruction(node, startPos, startLine, startColumn)
			if err != nil {
				return nil, err
			}
			p.appendSVGChild(node, instruction, &text)
		case strings.HasPrefix(p.content[p.pos:], "<!"):
			comment, err := p.parseBogusComment(node, startPos, startLine, startColumn)
			if err != nil {
				return nil, err
			}
			p.appendSVGChild(node, comment, &text)
		case p.peek() == '<' && p.startsMarkupToken():
			child, emitted, err := p.parseSVGStartElement(node, startPos, startLine, startColumn)
			if err != nil {
				return nil, err
			}
			if !emitted {
				p.extendPreviousTextRange(node, startPos)
				p.recoverOpenElementsAtEOF = true
				p.finishImplicitElement(node, text.String(), contentStart)
				return node, nil
			}
			p.appendSVGChild(node, child, &text)
		default:
			child, err := p.parseSVGText(node)
			if err != nil {
				return nil, err
			}
			p.appendSVGChild(node, child, &text)
		}
	}

	p.recoverOpenElementsAtEOF = true
	p.finishImplicitElement(node, text.String(), contentStart)
	return node, nil
}

func isSVGHTMLIntegrationPoint(node *types.Node) bool {
	if node == nil || node.NamespaceURI != svgElementNamespace {
		return false
	}
	switch node.Name {
	case "foreignObject", "desc", "title":
		return true
	default:
		return false
	}
}

func isMathMLTextIntegrationPoint(node *types.Node) bool {
	if node == nil || node.NamespaceURI != mathElementNamespace {
		return false
	}
	switch node.Name {
	case "mi", "mo", "mn", "ms", "mtext":
		return true
	default:
		return false
	}
}

func isMathMLAnnotationXML(node *types.Node) bool {
	return node != nil && node.NamespaceURI == mathElementNamespace && node.Name == "annotation-xml"
}

func isMathMLAnnotationXMLHTMLIntegrationPoint(node *types.Node) bool {
	if !isMathMLAnnotationXML(node) {
		return false
	}
	encoding, ok := node.Attributes["encoding"]
	return ok && (asciiEqualFold(encoding, "text/html") || asciiEqualFold(encoding, "application/xhtml+xml"))
}

func isForeignIntegrationOwner(node *types.Node) bool {
	return isSVGHTMLIntegrationPoint(node) || isMathMLTextIntegrationPoint(node) || isMathMLAnnotationXML(node)
}

func (p *HTMLParser) processForeignIntegrationToken(parent *types.Node, text *strings.Builder) (bool, error) {
	if p.pos >= len(p.content) {
		return false, nil
	}
	textIntegration := isMathMLTextIntegrationPoint(parent)
	annotationIntegration := isMathMLAnnotationXMLHTMLIntegrationPoint(parent)
	htmlIntegration := isSVGHTMLIntegrationPoint(parent) || textIntegration || annotationIntegration
	if htmlIntegration && (p.peek() != '<' || !p.startsMarkupToken()) {
		startPos, startLine, startColumn := p.pos, p.line, p.col
		child, err := p.parseTextNode(parent, startPos, startLine, startColumn)
		if err != nil {
			return true, err
		}
		p.appendSVGChild(parent, child, text)
		return true, nil
	}
	name, closing, tag := p.peekTagName()
	if htmlIntegration && closing && (name == "p" || name == "br") && p.currentEndTagTokenWillEmit() {
		if name == "br" {
			p.foreignBRReprocess = true
		}
		child, err := p.parseNode(parent)
		if err != nil {
			return true, err
		}
		p.appendSVGChild(parent, child, text)
		return true, nil
	}
	if !tag || closing {
		return false, nil
	}
	if textIntegration && (name == "mglyph" || name == "malignmark") {
		return false, nil
	}
	if isMathMLAnnotationXML(parent) && !annotationIntegration && name != "svg" {
		return false, nil
	}
	if handled, err := p.handleDocumentStartToken(); handled || err != nil {
		return handled, err
	}

	previousNames := p.activeSVGNames
	if name == "svg" || name == "math" {
		// An HTML integration point is a foreign-scope barrier. A nested SVG
		// or MathML island must not be able to match an end tag against the outer
		// stack through the intervening HTML insertion context.
		p.activeSVGNames = make(map[string]*types.Node)
		defer func() { p.activeSVGNames = previousNames }()
	}
	p.foreignHTMLDepth++
	child, err := p.parseNode(parent)
	p.foreignHTMLDepth--
	if err != nil {
		return true, err
	}
	p.appendSVGChild(parent, child, text)
	return true, nil
}

func (p *HTMLParser) currentEndTagTokenWillEmit() bool {
	preview := *p
	_, emitted, err := preview.parseClosingTag()
	return err == nil && emitted
}

func (p *HTMLParser) addForeignIgnoredEnd(name string) {
	p.foreignIgnoredEnds[name]++
	p.foreignIgnoredEndCount++
	p.foreignBreakoutRecovery = true
}

func (p *HTMLParser) retireForeignIgnoredEnd(name string) {
	if p.foreignIgnoredEnds[name] == 0 {
		return
	}
	p.foreignIgnoredEnds[name]--
	if p.foreignIgnoredEndCount > 0 {
		p.foreignIgnoredEndCount--
	}
	if p.foreignIgnoredEndCount == 0 && p.foreignBreakoutBoundary == nil {
		p.foreignBreakoutRecovery = false
	}
}

func (p *HTMLParser) consumeForeignIgnoredEnd(parent *types.Node) (bool, error) {
	name, closing, ok := p.peekTagName()
	if !ok || !closing || p.foreignIgnoredEnds[name] == 0 || nearestOpenElementByName(parent, name) != nil {
		return false, nil
	}
	start := p.pos
	_, emitted, err := p.parseClosingTag()
	if err != nil {
		return true, err
	}
	if !emitted {
		p.extendPreviousTextRange(parent, start)
		p.recoverOpenElementsAtEOF = true
	}
	p.retireForeignIgnoredEnd(name)
	return true, nil
}

func (p *HTMLParser) handleForeignDocumentStart() (bool, error) {
	if p.foreignHTMLDepth == 0 && p.foreignIgnoredEndCount == 0 && !p.foreignBreakoutRecovery {
		return false, nil
	}
	return p.handleDocumentStartToken()
}

func (p *HTMLParser) handleDocumentStartToken() (bool, error) {
	name, closing, ok := p.peekTagName()
	if !ok || closing || (name != "html" && name != "body" && name != "head") || !p.currentTagTokenWillEmit() {
		return false, nil
	}
	switch name {
	case "html":
		return true, p.mergeDuplicateStartInto(p.documentHTML, "html")
	case "body":
		return true, p.mergeDuplicateStartInto(p.documentBody, "body")
	default:
		p.consumeIgnoredStartTag()
		return true, nil
	}
}

func (p *HTMLParser) svgStartTokenBreaksOut() bool {
	name, closing, ok := p.peekTagName()
	if !ok || closing || !p.currentTagTokenWillEmit() {
		return false
	}
	switch name {
	case "b", "big", "blockquote", "body", "br", "center", "code", "dd", "div", "dl", "dt", "em", "embed",
		"h1", "h2", "h3", "h4", "h5", "h6", "head", "hr", "i", "img", "li", "listing", "menu", "meta",
		"nobr", "ol", "p", "pre", "ruby", "s", "small", "span", "strong", "strike", "sub", "sup", "table", "tt", "u", "ul", "var":
		return true
	case "font":
		preview := *p
		preview.advance()
		if preview.parseTagName() != "font" {
			return false
		}
		attributes := &types.Node{Attributes: make(map[string]string)}
		if _, emitted := preview.parseStartTagTail(attributes); !emitted {
			return false
		}
		for _, attribute := range []string{"color", "face", "size"} {
			if _, present := attributes.Attributes[attribute]; present {
				return true
			}
		}
	}
	return false
}

func (p *HTMLParser) matchingForeignAncestorWithinIntegration(node *types.Node, closingName string) (*types.Node, bool) {
	if p.foreignHTMLDepth == 0 || p.activeSVGNames[asciiLower(closingName)] == nil {
		return nil, false
	}
	blocked := false
	for current := node; current != nil; current = current.Parent {
		if current.NamespaceURI == svgElementNamespace || current.NamespaceURI == mathElementNamespace {
			if asciiEqualFold(asciiLower(current.Name), closingName) {
				return current, blocked
			}
			continue
		}
		// An end tag processed in the HTML insertion mode cannot reach back
		// through a live HTML child to close its foreign integration owner.
		// The owner itself is the special-element barrier, even when the live
		// HTML child is ordinary phrasing content such as span, b, or i.
		if current.Type == types.ElementNode {
			blocked = true
		}
	}
	return nil, false
}

func foreignRecoveryGenericEndTarget(node *types.Node, closingName string) *types.Node {
	for current := node; current != nil && current.NamespaceURI == htmlElementNamespace; current = current.Parent {
		if current.Name == closingName {
			return current
		}
		if isHTMLSpecialElement(current.Name) {
			return nil
		}
	}
	return nil
}

func hasASCIIPrefixFold(value, prefix string) bool {
	return len(value) >= len(prefix) && asciiEqualFold(value[:len(prefix)], strings.ToLower(prefix))
}

func (p *HTMLParser) parseSVGStartElement(parent *types.Node, startPos, startLine, startColumn int) (*types.Node, bool, error) {
	p.advance()
	name := p.parseTagName()
	if name == "" {
		return nil, false, fmt.Errorf("expected foreign tag name at position %d", startPos)
	}
	namespace := parent.NamespaceURI
	adjustedName := name
	if namespace == svgElementNamespace {
		adjustedName = adjustSVGTagName(name)
	}
	node := &types.Node{
		Type: types.ElementNode, Name: adjustedName, NamespaceURI: namespace,
		Attributes: make(map[string]string), AttributeOrder: []string{}, Children: []*types.Node{}, Parent: parent,
		StartPos: startPos, StartLine: startLine, StartColumn: startColumn,
	}
	if owner := p.templateOwner[parent]; owner != nil {
		p.templateOwner[node] = owner
	}
	selfClosing, emitted := p.parseStartTagTail(node)
	if !emitted {
		return nil, false, nil
	}
	switch namespace {
	case svgElementNamespace:
		adjustSVGAttributes(node)
	case mathElementNamespace:
		adjustMathMLAttributes(node)
	}
	initializeAttributeMetadata(node, true)
	node.ContentStart = p.pos
	p.elementContentStart[node] = sourcePoint{line: p.line, column: p.col}
	if selfClosing {
		p.finishForeignSelfClosing(node)
		return node, true, nil
	}
	parsed, err := p.parseSVGElement(node, p.pos)
	return parsed, true, err
}

func (p *HTMLParser) parseSVGText(parent *types.Node) (*types.Node, error) {
	startPos, startLine, startColumn := p.pos, p.line, p.col
	var value strings.Builder
	for p.pos < len(p.content) {
		if p.peek() == '<' && p.startsMarkupToken() {
			break
		}
		if reference, consumed := p.consumeNumericCharacterReference(); consumed {
			value.WriteRune(reference)
			continue
		}
		if reference, consumed := p.consumeNamedCharacterReference(false); consumed {
			value.WriteString(reference)
			continue
		}
		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		value.WriteRune(r)
		p.advanceRune(size)
	}
	if value.Len() == 0 {
		return nil, nil
	}
	text := value.String()
	return &types.Node{
		Type: types.TextNode, Name: "#text", Value: text, TextContent: text, Parent: parent,
		StartPos: startPos, EndPos: p.pos, StartLine: startLine, StartColumn: startColumn,
		EndLine: p.line, EndColumn: p.col,
	}, nil
}

func (p *HTMLParser) parseSVGCDATA(parent *types.Node, startPos, startLine, startColumn int) *types.Node {
	const prefix = "<![CDATA["
	p.advanceRune(len(prefix))
	var value strings.Builder
	for p.pos < len(p.content) && !strings.HasPrefix(p.content[p.pos:], "]]>") {
		r, size := p.peekHTMLRune()
		value.WriteRune(r)
		p.advanceRune(size)
	}
	if strings.HasPrefix(p.content[p.pos:], "]]>") {
		p.advanceRune(3)
	} else {
		p.recoverOpenElementsAtEOF = true
	}
	text := value.String()
	return &types.Node{
		Type: types.TextNode, Name: "#text", Value: text, TextContent: text, Parent: parent,
		StartPos: startPos, EndPos: p.pos, StartLine: startLine, StartColumn: startColumn,
		EndLine: p.line, EndColumn: p.col,
	}
}

func (p *HTMLParser) appendSVGChild(parent, child *types.Node, text *strings.Builder) {
	if child == nil {
		return
	}
	child.Parent = parent
	if child.Type == types.TextNode && len(parent.Children) > 0 && parent.Children[len(parent.Children)-1].Type == types.TextNode {
		mergeTextNodes(parent.Children[len(parent.Children)-1], child)
	} else {
		parent.Children = append(parent.Children, child)
	}
	if child.Type == types.TextNode || child.Type == types.ElementNode {
		text.WriteString(child.TextContent)
	}
}

// consumeNumericCharacterReference implements the numeric branch of HTML's
// character-reference tokenizer. It consumes only references containing at
// least one digit; malformed "&#" and "&#x" prefixes remain ordinary text.
// Advancing across the original ASCII source keeps decoded DOM values and
// original byte positions independent.

const (
	htmlElementNamespace = "http://www.w3.org/1999/xhtml"
	svgElementNamespace  = "http://www.w3.org/2000/svg"
	mathElementNamespace = "http://www.w3.org/1998/Math/MathML"
)

var svgTagNameAdjustments = map[string]string{
	"altglyph": "altGlyph", "altglyphdef": "altGlyphDef", "altglyphitem": "altGlyphItem",
	"animatecolor": "animateColor", "animatemotion": "animateMotion", "animatetransform": "animateTransform",
	"clippath": "clipPath", "feblend": "feBlend", "fecolormatrix": "feColorMatrix",
	"fecomponenttransfer": "feComponentTransfer", "fecomposite": "feComposite",
	"feconvolvematrix": "feConvolveMatrix", "fediffuselighting": "feDiffuseLighting",
	"fedisplacementmap": "feDisplacementMap", "fedistantlight": "feDistantLight", "feflood": "feFlood",
	"fedropshadow": "feDropShadow",
	"fefunca":      "feFuncA", "fefuncb": "feFuncB", "fefuncg": "feFuncG", "fefuncr": "feFuncR",
	"fegaussianblur": "feGaussianBlur", "feimage": "feImage", "femerge": "feMerge",
	"femergenode": "feMergeNode", "femorphology": "feMorphology", "feoffset": "feOffset",
	"fepointlight": "fePointLight", "fespecularlighting": "feSpecularLighting", "fespotlight": "feSpotLight",
	"fetile": "feTile", "feturbulence": "feTurbulence", "foreignobject": "foreignObject",
	"glyphref": "glyphRef", "lineargradient": "linearGradient", "radialgradient": "radialGradient",
	"textpath": "textPath",
}

var svgAttributeAdjustments = func() map[string]string {
	names := []string{
		"attributeName", "attributeType", "baseFrequency", "baseProfile", "calcMode", "clipPathUnits",
		"diffuseConstant", "edgeMode", "filterUnits", "glyphRef", "gradientTransform", "gradientUnits",
		"kernelMatrix", "kernelUnitLength", "keyPoints", "keySplines", "keyTimes", "lengthAdjust",
		"limitingConeAngle", "markerHeight", "markerUnits", "markerWidth", "maskContentUnits", "maskUnits",
		"numOctaves", "pathLength", "patternContentUnits", "patternTransform", "patternUnits", "pointsAtX",
		"pointsAtY", "pointsAtZ", "preserveAlpha", "preserveAspectRatio", "primitiveUnits", "refX", "refY",
		"repeatCount", "repeatDur", "requiredExtensions", "requiredFeatures", "specularConstant",
		"specularExponent", "spreadMethod", "startOffset", "stdDeviation", "stitchTiles", "surfaceScale",
		"systemLanguage", "tableValues", "targetX", "targetY", "textLength", "viewBox", "viewTarget",
		"xChannelSelector", "yChannelSelector", "zoomAndPan",
	}
	adjustments := make(map[string]string, len(names))
	for _, name := range names {
		adjustments[strings.ToLower(name)] = name
	}
	return adjustments
}()
