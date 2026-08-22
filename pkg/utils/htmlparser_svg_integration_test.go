package utils

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 17B covers SVG breakout/reprocessing plus the SVG foreignObject,
// desc, and title HTML integration points in ordinary in-body parsing.
// MathML, SVG/MathML integration attributes, table/select/template modes,
// adoption across foreign boundaries, fragments, foreign scripting, and
// XML/XLink attributes remain deferred.

func TestParseSVGHTMLBreakoutReprocessesInBody(t *testing.T) {
	const paragraph = `<div>a<svg><g><p>x</p></g><circle /></svg>z</div>`
	document, err := NewHTMLParser().Parse(paragraph)
	if err != nil {
		t.Fatal(err)
	}
	div := svgNodesNamed(document, "div")[0]
	svg := svgNodesNamed(document, "svg")[0]
	g := svgNodesNamed(document, "g")[0]
	p := svgNodesNamed(document, "p")[0]
	circle := svgNodesNamed(document, "circle")[0]
	assertFormattingNode(t, div, "div", "axz", 0, 49)
	assertFormattingNode(t, svg, "svg", "", 6, 14)
	assertFormattingNode(t, g, "g", "", 11, 14)
	assertFormattingNode(t, p, "p", "x", 14, 22)
	assertFormattingNode(t, circle, "circle", "z", 26, 43)
	if len(div.Children) != 4 || div.Children[1] != svg || div.Children[2] != p || div.Children[3] != circle || circle.NamespaceURI != htmlNamespaceURI {
		t.Fatalf("Foreign p breakout was not reprocessed at the HTML parent: %#v", div.Children)
	}
	if svg.ContentStart != 11 || svg.ContentEnd != 14 || g.ContentStart != 14 || g.ContentEnd != 14 || p.ContentStart != 17 || p.ContentEnd != 18 || circle.ContentStart != 34 || circle.ContentEnd != 43 {
		t.Fatalf("Breakout source boundaries differ from Chrome: svg=%#v g=%#v p=%#v circle=%#v", svg, g, p, circle)
	}

	const block = `<svg><g><div>x</div>y</g></svg>z`
	document, err = NewHTMLParser().Parse(block)
	if err != nil {
		t.Fatal(err)
	}
	svg = svgNodesNamed(document, "svg")[0]
	g = svgNodesNamed(document, "g")[0]
	div = svgNodesNamed(document, "div")[0]
	assertFormattingNode(t, svg, "svg", "", 0, 8)
	assertFormattingNode(t, g, "g", "", 5, 8)
	assertFormattingNode(t, div, "div", "x", 8, 20)
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[0] != svg || parsedBodyChildren(document)[1] != div || parsedBodyChildren(document)[2].Value != "yz" || parsedBodyChildren(document)[2].StartPos != 20 || parsedBodyChildren(document)[2].EndPos != 32 {
		t.Fatalf("Breakout did not ignore stale foreign ends and coalesce HTML text: %#v", parsedBodyChildren(document))
	}
}

func TestParseSVGFontBreakoutRequiresSpecialAttributes(t *testing.T) {
	const plain = `<svg><g><font>x</font><circle /></g></svg>z`
	document, err := NewHTMLParser().Parse(plain)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	font := svgNodesNamed(document, "font")[0]
	circle := svgNodesNamed(document, "circle")[0]
	assertFormattingNode(t, svg, "svg", "x", 0, 42)
	assertFormattingNode(t, font, "font", "x", 8, 22)
	assertFormattingNode(t, circle, "circle", "", 22, 32)
	if font.NamespaceURI != svgNamespaceURI || circle.NamespaceURI != svgNamespaceURI {
		t.Fatalf("Plain foreign font must remain SVG: font=%#v circle=%#v", font, circle)
	}

	const attributed = `<svg><g><font color=red>x</font><circle /></g></svg>z`
	document, err = NewHTMLParser().Parse(attributed)
	if err != nil {
		t.Fatal(err)
	}
	svg = svgNodesNamed(document, "svg")[0]
	font = svgNodesNamed(document, "font")[0]
	circle = svgNodesNamed(document, "circle")[0]
	assertFormattingNode(t, svg, "svg", "", 0, 8)
	assertFormattingNode(t, font, "font", "x", 8, 32)
	assertFormattingNode(t, circle, "circle", "z", 32, 53)
	if font.NamespaceURI != htmlNamespaceURI || font.Attributes["color"] != "red" || circle.NamespaceURI != htmlNamespaceURI {
		t.Fatalf("Attributed font must break out and reprocess following tokens as HTML: font=%#v circle=%#v", font, circle)
	}
}

func TestParseSVGSpanBreaksOutWhileAnchorRemainsForeign(t *testing.T) {
	const anchor = `<svg><g><a id=a>x</a><circle /></g></svg>z`
	document, err := NewHTMLParser().Parse(anchor)
	if err != nil {
		t.Fatal(err)
	}
	a := svgNodeByID(t, document, "a")
	if a.Name != "a" || a.NamespaceURI != svgNamespaceURI || a.Parent == nil || a.Parent.Name != "g" || a.TextContent != "x" {
		t.Fatalf("SVG anchor must remain foreign: %#v", a)
	}

	const span = `<svg><g><span id=s>x</span><a id=a>y</a></g></svg>z`
	document, err = NewHTMLParser().Parse(span)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	g := svgNodesNamed(document, "g")[0]
	spanNode := svgNodeByID(t, document, "s")
	a = svgNodeByID(t, document, "a")
	assertFormattingNode(t, svg, "svg", "", 0, 8)
	assertFormattingNode(t, g, "g", "", 5, 8)
	assertFormattingNode(t, spanNode, "span", "x", 8, 27)
	assertFormattingNode(t, a, "a", "y", 27, 40)
	if spanNode.NamespaceURI != htmlNamespaceURI || a.NamespaceURI != htmlNamespaceURI || len(parsedBodyChildren(document)) != 4 || parsedBodyChildren(document)[0] != svg || parsedBodyChildren(document)[1] != spanNode || parsedBodyChildren(document)[2] != a || parsedBodyChildren(document)[3].Value != "z" {
		t.Fatalf("Span breakout did not permanently restore HTML mode: %#v", parsedBodyChildren(document))
	}
}

func TestParseSVGForeignEndBreakoutForParagraphAndBR(t *testing.T) {
	tests := []struct {
		name, content, element string
		textStart              int
	}{
		{"p", `<svg><g>x</p>y</g></svg>z`, "p", 13},
		{"br", `<svg><g>x</br>y</g></svg>z`, "br", 14},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			svg := svgNodesNamed(document, "svg")[0]
			g := svgNodesNamed(document, "g")[0]
			htmlNodes := svgNodesNamed(document, test.element)
			if len(htmlNodes) != 1 {
				t.Fatalf("Expected one synthetic HTML <%s> after foreign end breakout, got %#v", test.element, htmlNodes)
			}
			html := htmlNodes[0]
			assertFormattingNode(t, svg, "svg", "x", 0, 9)
			assertFormattingNode(t, g, "g", "x", 5, 9)
			if html.NamespaceURI != htmlNamespaceURI || html.StartPos != 0 || html.EndPos != 0 || len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[1] != html || parsedBodyChildren(document)[2].Value != "yz" || parsedBodyChildren(document)[2].StartPos != test.textStart || parsedBodyChildren(document)[2].EndPos != len(test.content) {
				t.Fatalf("Foreign </%s> breakout mismatch: html=%#v doc=%#v", test.element, html, parsedBodyChildren(document))
			}
		})
	}
}

func TestParseSVGPermanentBreakoutIgnoresStaleForeignTokens(t *testing.T) {
	const content = `<svg><g>a<div>b</div>c</g><circle></circle></svg>d`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	g := svgNodesNamed(document, "g")[0]
	div := svgNodesNamed(document, "div")[0]
	circle := svgNodesNamed(document, "circle")[0]
	assertFormattingNode(t, svg, "svg", "a", 0, 9)
	assertFormattingNode(t, g, "g", "a", 5, 9)
	assertFormattingNode(t, div, "div", "b", 9, 21)
	assertFormattingNode(t, circle, "circle", "", 26, 43)
	if circle.NamespaceURI != htmlNamespaceURI || len(parsedBodyChildren(document)) != 5 || parsedBodyChildren(document)[0] != svg || parsedBodyChildren(document)[1] != div || parsedBodyChildren(document)[2].Value != "c" || parsedBodyChildren(document)[2].StartPos != 21 || parsedBodyChildren(document)[3] != circle || parsedBodyChildren(document)[4].Value != "d" {
		t.Fatalf("Foreign breakout did not permanently reprocess following tokens as HTML: %#v", parsedBodyChildren(document))
	}
}

func TestParseSVGForeignBreakoutStartAllowlist(t *testing.T) {
	breakout := []string{
		"b", "big", "blockquote", "body", "br", "center", "code", "dd", "div", "dl", "dt", "em", "embed",
		"h1", "h2", "h3", "h4", "h5", "h6", "head", "hr", "i", "img", "li", "listing", "menu", "meta",
		"nobr", "ol", "p", "pre", "ruby", "s", "small", "span", "strong", "strike", "sub", "sup", "tt", "u", "ul", "var",
	}
	for _, name := range breakout {
		t.Run(name, func(t *testing.T) {
			content := `<svg><g>a<` + name + ` id=h>x`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			svg := svgNodesNamed(document, "svg")[0]
			g := svgNodesNamed(document, "g")[0]
			if svg.EndPos != 9 || g.EndPos != 9 || svg.ContentEnd != 9 || g.ContentEnd != 9 || svg.TextContent != "a" || g.TextContent != "a" {
				t.Fatalf("<%s> did not pop foreign ancestors at its token start: svg=%#v g=%#v", name, svg, g)
			}
			if html := svgNodeByID(t, document, "h"); html != nil && html.NamespaceURI != htmlNamespaceURI {
				t.Fatalf("Breakout <%s> was reprocessed in the SVG namespace: %#v", name, html)
			}
		})
	}

	for _, name := range []string{"a", "circle", "font"} {
		t.Run("nonmember-"+name, func(t *testing.T) {
			content := `<svg><g>a<` + name + ` id=h>x</` + name + `></g></svg>`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			node := svgNodeByID(t, document, "h")
			if node == nil || node.NamespaceURI != svgNamespaceURI || node.Parent == nil || node.Parent.Name != "g" {
				t.Fatalf("Non-breakout <%s> did not remain SVG: %#v", name, node)
			}
		})
	}
}

func TestParseSVGHTMLIntegrationPointsSwitchNamespaces(t *testing.T) {
	tests := []struct {
		name, content, integrationName, htmlName, id string
		integrationStart, integrationEnd             int
		htmlStart, htmlEnd                           int
		circleStart, circleEnd                       int
	}{
		{"foreignObject", `<svg><foreignObject><div id=d><b>x</b></div></foreignObject><circle /></svg>z`, "foreignObject", "div", "d", 5, 60, 20, 44, 60, 70},
		{"desc", `<svg><desc><p id=p>x</p></desc><circle /></svg>z`, "desc", "p", "p", 5, 31, 11, 24, 31, 41},
		{"title", `<svg><title><span id=s>x</span></title><circle /></svg>z`, "title", "span", "s", 5, 39, 12, 31, 39, 49},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			svg := svgNodesNamed(document, "svg")[0]
			integration := svgNodesNamed(document, test.integrationName)[0]
			html := svgNodeByID(t, document, test.id)
			circle := svgNodesNamed(document, "circle")[0]
			assertFormattingNode(t, integration, test.integrationName, "x", test.integrationStart, test.integrationEnd)
			assertFormattingNode(t, html, test.htmlName, "x", test.htmlStart, test.htmlEnd)
			assertFormattingNode(t, circle, "circle", "", test.circleStart, test.circleEnd)
			if integration.NamespaceURI != svgNamespaceURI || html.NamespaceURI != htmlNamespaceURI || circle.NamespaceURI != svgNamespaceURI || integration.Parent != svg || html.Parent != integration || circle.Parent != svg {
				t.Fatalf("Integration-point namespace/parent transition mismatch: integration=%#v html=%#v circle=%#v", integration, html, circle)
			}
			if parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].Value != "z" {
				t.Fatalf("Integration point failed to restore HTML continuation: %#v", parsedBodyChildren(document))
			}
		})
	}
}

func TestParseSVGMixedCaseForeignObjectAdjustmentAndIntegration(t *testing.T) {
	const content = `<svg><FOREIGNOBJECT><DIV>x</DIV></FOREIGNOBJECT><rect /></svg>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	foreignObjects := svgNodesNamed(document, "foreignObject")
	divs := svgNodesNamed(document, "div")
	rects := svgNodesNamed(document, "rect")
	if len(foreignObjects) != 1 || len(divs) != 1 || len(rects) != 1 {
		t.Fatalf("Expected adjusted foreignObject, HTML div, and SVG rect, got fo=%#v div=%#v rect=%#v", foreignObjects, divs, rects)
	}
	foreignObject, div, rect := foreignObjects[0], divs[0], rects[0]
	if foreignObject.Name != "foreignObject" || foreignObject.NamespaceURI != svgNamespaceURI || div.NamespaceURI != htmlNamespaceURI || div.Parent != foreignObject || div.TextContent != "x" || rect.NamespaceURI != svgNamespaceURI || rect.Parent != foreignObject.Parent {
		t.Fatalf("Mixed-case foreignObject did not adjust/dispatch as integration point: fo=%#v div=%#v rect=%#v", foreignObject, div, rect)
	}
}

func TestParseSVGIntegrationPointEndBreakoutStaysWithinIntegration(t *testing.T) {
	tests := []struct {
		name, content, integration, emitted string
		integrationStart, integrationEnd    int
		textStart                           int
	}{
		{"p", `<svg id=s><foreignObject id=f>x</p>y</foreignObject><rect id=r></rect></svg>z`, "foreignObject", "p", 10, 52, 35},
		{"br", `<svg id=s><desc id=f>x</br>y</desc><rect id=r></rect></svg>z`, "desc", "br", 10, 35, 27},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			integrations := svgNodesNamed(document, test.integration)
			emittedNodes := svgNodesNamed(document, test.emitted)
			if len(integrations) != 1 || len(emittedNodes) != 1 {
				t.Fatalf("Expected one <%s> integration parent and one synthetic <%s>, got integration=%#v emitted=%#v", test.integration, test.emitted, integrations, emittedNodes)
			}
			integration, emitted := integrations[0], emittedNodes[0]
			assertFormattingNode(t, integration, test.integration, "xy", test.integrationStart, test.integrationEnd)
			if emitted.Parent != integration || emitted.NamespaceURI != htmlNamespaceURI || emitted.StartPos != 0 || emitted.EndPos != 0 || len(integration.Children) != 3 || integration.Children[0].Value != "x" || integration.Children[1] != emitted || integration.Children[2].Value != "y" || integration.Children[2].StartPos != test.textStart {
				t.Fatalf("Integration-point </%s> escaped its integration parent: %#v", test.emitted, integration.Children)
			}
		})
	}
}

func TestParseSVGNestedBreakoutStopsAtIntegrationPoint(t *testing.T) {
	const content = `<svg id=o><foreignObject id=f><svg id=i><g id=g>a<div id=h>b</div>c</foreignObject><rect id=r></rect></svg>q`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	outer := svgNodeByID(t, document, "o")
	foreignObject := svgNodeByID(t, document, "f")
	inner := svgNodeByID(t, document, "i")
	g := svgNodeByID(t, document, "g")
	div := svgNodeByID(t, document, "h")
	rect := svgNodeByID(t, document, "r")
	assertFormattingNode(t, outer, "svg", "abc", 0, 107)
	assertFormattingNode(t, foreignObject, "foreignObject", "abc", 10, 83)
	assertFormattingNode(t, inner, "svg", "a", 30, 49)
	assertFormattingNode(t, g, "g", "a", 40, 49)
	assertFormattingNode(t, div, "div", "b", 49, 66)
	assertFormattingNode(t, rect, "rect", "", 83, 101)
	if inner.Parent != foreignObject || div.Parent != foreignObject || rect.Parent != outer || div.NamespaceURI != htmlNamespaceURI || len(foreignObject.Children) != 3 || foreignObject.Children[2].Value != "c" || parsedBodyChildren(document)[1].Value != "q" {
		t.Fatalf("Nested breakout crossed integration boundary: outer=%#v fo=%#v", outer.Children, foreignObject.Children)
	}
}

func TestParseSVGIntegrationPointDecodesReferencesAsHTMLCharacterTokens(t *testing.T) {
	const content = `<svg><foreignObject id=f>a&amp;b&#32;c</foreignObject></svg>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	foreignObject := svgNodeByID(t, document, "f")
	assertFormattingNode(t, foreignObject, "foreignObject", "a&b c", 5, 54)
	if len(foreignObject.Children) != 1 || foreignObject.Children[0].Type != types.TextNode || foreignObject.Children[0].Value != "a&b c" || foreignObject.Children[0].StartPos != 25 || foreignObject.Children[0].EndPos != 38 {
		t.Fatalf("Integration-point references did not coalesce into one HTML character token: %#v", foreignObject.Children)
	}
}

func TestParseSVGIntegrationPointCanEnterNestedSVG(t *testing.T) {
	const content = `<svg><foreignObject><svg id=i><circle /></svg><div>x</div></foreignObject></svg>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	svgs := svgNodesNamed(document, "svg")
	if len(svgs) != 2 {
		t.Fatalf("Expected outer and nested SVG, got %#v", svgs)
	}
	outer, inner := svgs[0], svgNodeByID(t, document, "i")
	foreignObject := svgNodesNamed(document, "foreignObject")[0]
	circle := svgNodesNamed(document, "circle")[0]
	div := svgNodesNamed(document, "div")[0]
	assertFormattingNode(t, outer, "svg", "x", 0, 80)
	assertFormattingNode(t, foreignObject, "foreignObject", "x", 5, 74)
	assertFormattingNode(t, inner, "svg", "", 20, 46)
	assertFormattingNode(t, circle, "circle", "", 30, 40)
	assertFormattingNode(t, div, "div", "x", 46, 58)
	if inner.NamespaceURI != svgNamespaceURI || circle.NamespaceURI != svgNamespaceURI || div.NamespaceURI != htmlNamespaceURI || inner.Parent != foreignObject || div.Parent != foreignObject {
		t.Fatalf("Nested SVG/integration namespace restoration mismatch: inner=%#v circle=%#v div=%#v", inner, circle, div)
	}
}

func TestParseSVGIntegrationEOFMalformedTokensAndReuse(t *testing.T) {
	const unicode = "<svg><foreignObject><div>é\r\n<b>😀"
	parser := NewHTMLParser()
	document, err := parser.Parse(unicode)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	foreignObject := svgNodesNamed(document, "foreignObject")[0]
	div := svgNodesNamed(document, "div")[0]
	bold := svgNodesNamed(document, "b")[0]
	for _, node := range []*types.Node{svg, foreignObject, div, bold} {
		if node.EndPos != 36 || node.EndLine != 2 || node.EndColumn != 6 {
			t.Fatalf("Integration EOF endpoint mismatch on <%s>: %#v", node.Name, node)
		}
	}
	if svg.StartPos != 0 || foreignObject.StartPos != 5 || div.StartPos != 20 || bold.StartPos != 29 || bold.ContentStart != 32 || bold.ContentEnd != 36 || bold.TextContent != "😀" || div.TextContent != "é\n😀" {
		t.Fatalf("Integration EOF tree/ranges mismatch: svg=%#v foreignObject=%#v div=%#v b=%#v", svg, foreignObject, div, bold)
	}
	if document, parseErr := parser.Parse(`<div>x`); parseErr != nil || len(svgNodesNamed(document, "div")) != 1 || svgNodesNamed(document, "div")[0].EndPos != 6 {
		t.Fatalf("SVG integration implicit-document EOF reuse mismatch: doc=%#v err=%v", document, parseErr)
	}

	const incompleteStart = `<svg><foreignObject><div x`
	document, err = NewHTMLParser().Parse(incompleteStart)
	if err != nil {
		t.Fatal(err)
	}
	svg = svgNodesNamed(document, "svg")[0]
	foreignObject = svgNodesNamed(document, "foreignObject")[0]
	if len(svgNodesNamed(document, "div")) != 0 || svg.EndPos != 26 || foreignObject.EndPos != 26 || foreignObject.ContentStart != 20 || foreignObject.ContentEnd != 26 {
		t.Fatalf("Discarded integration-point start created a phantom node: svg=%#v foreignObject=%#v", svg, foreignObject)
	}

	const incompleteEnd = `<svg><foreignObject><div>x</foreignObject`
	document, err = NewHTMLParser().Parse(incompleteEnd)
	if err != nil {
		t.Fatal(err)
	}
	div = svgNodesNamed(document, "div")[0]
	if div.EndPos != 41 || div.ContentStart != 25 || div.ContentEnd != 41 || len(div.Children) != 1 || div.Children[0].Value != "x" || div.Children[0].StartPos != 25 || div.Children[0].EndPos != 41 {
		t.Fatalf("Discarded integration end must extend preceding text through EOF: %#v", div)
	}
}
