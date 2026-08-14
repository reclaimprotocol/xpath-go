package utils

import (
	"strings"
	"testing"
	"time"

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

func TestParseSVGIncompleteBreakoutTokensDoNotMutateForeignState(t *testing.T) {
	for _, content := range []string{
		`<div><svg id=s><g id=g>a<font color=red`,
		`<div><svg id=s><g id=g>a</p`,
	} {
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		svg := svgNodeByID(t, document, "s")
		g := svgNodeByID(t, document, "g")
		if svg.NamespaceURI != svgNamespaceURI || g.NamespaceURI != svgNamespaceURI || len(g.Children) != 1 || g.Children[0].Type != types.TextNode || g.Children[0].Value != "a" || g.Children[0].StartPos != 23 || g.Children[0].EndPos != len(content) || svg.EndPos != len(content) || g.EndPos != len(content) {
			t.Fatalf("Incomplete breakout token mutated foreign state for %q: svg=%#v g=%#v", content, svg, g)
		}
	}
}

func TestParseSVGIntegrationScopeBarrierAndFormattingBreakout(t *testing.T) {
	const barrier = `<svg><foreignObject><p>x</foreignObject>y</svg>z`
	document, err := NewHTMLParser().Parse(barrier)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	foreignObject := svgNodesNamed(document, "foreignObject")[0]
	p := svgNodesNamed(document, "p")[0]
	assertFormattingNode(t, svg, "svg", "xyz", 0, 48)
	assertFormattingNode(t, foreignObject, "foreignObject", "xyz", 5, 48)
	assertFormattingNode(t, p, "p", "xyz", 20, 48)
	if p.Parent != foreignObject || len(p.Children) != 1 || p.Children[0].Value != "xyz" || p.Children[0].StartPos != 23 || p.Children[0].EndPos != 48 {
		t.Fatalf("Special HTML p must block foreign ancestor ends and retain suffix text: %#v", p)
	}

	const formatting = `<b><svg><g><p>x</p></g></svg>y</b>z`
	document, err = NewHTMLParser().Parse(formatting)
	if err != nil {
		t.Fatal(err)
	}
	bold := svgNodesNamed(document, "b")[0]
	svg = svgNodesNamed(document, "svg")[0]
	p = svgNodesNamed(document, "p")[0]
	assertFormattingNode(t, bold, "b", "xy", 0, 34)
	assertFormattingNode(t, svg, "svg", "", 3, 11)
	assertFormattingNode(t, p, "p", "x", 11, 19)
	if p.Parent != bold || p.NamespaceURI != htmlNamespaceURI || bold.Children[len(bold.Children)-1].Value != "y" || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].Value != "z" {
		t.Fatalf("Foreign breakout corrupted active-formatting placement: bold=%#v doc=%#v", bold.Children, parsedBodyChildren(document))
	}
}

func TestParseSVGForeignEndDoesNotCrossHTMLIntegrationBarrier(t *testing.T) {
	const content = `<svg id=o><foreignObject id=f><div id=h><svg id=i><g id=g>x</foreignObject><circle id=c></circle></svg>z</div></foreignObject><rect id=r></rect></svg>q`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	outer := svgNodeByID(t, document, "o")
	foreignObject := svgNodeByID(t, document, "f")
	div := svgNodeByID(t, document, "h")
	inner := svgNodeByID(t, document, "i")
	g := svgNodeByID(t, document, "g")
	circle := svgNodeByID(t, document, "c")
	rect := svgNodeByID(t, document, "r")
	assertFormattingNode(t, outer, "svg", "xz", 0, 150)
	assertFormattingNode(t, foreignObject, "foreignObject", "xz", 10, 126)
	assertFormattingNode(t, div, "div", "xz", 30, 110)
	assertFormattingNode(t, inner, "svg", "x", 40, 103)
	assertFormattingNode(t, g, "g", "x", 50, 97)
	assertFormattingNode(t, circle, "circle", "", 75, 97)
	assertFormattingNode(t, rect, "rect", "", 126, 144)
	if foreignObject.Parent != outer || div.Parent != foreignObject || inner.Parent != div || g.Parent != inner || circle.Parent != g || rect.Parent != outer || div.NamespaceURI != htmlNamespaceURI || inner.NamespaceURI != svgNamespaceURI || circle.NamespaceURI != svgNamespaceURI || len(div.Children) != 2 || div.Children[1].Value != "z" || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].Value != "q" {
		t.Fatalf("Foreign end crossed an intervening HTML integration barrier: outer=%#v fo=%#v div=%#v inner=%#v g=%#v", outer, foreignObject, div, inner, g)
	}
}

func TestParseSVGForeignOwnerEndIgnoredAcrossLiveHTMLChild(t *testing.T) {
	tests := []struct {
		name, element, content string
		length, childStart     int
	}{
		{"span", "span", `<svg id=o><foreignObject id=f><span id=h>x</foreignObject><rect id=r></rect></svg>z`, 83, 41},
		{"b", "b", `<svg id=o><foreignObject id=f><b id=h>x</foreignObject><rect id=r></rect></svg>z`, 80, 38},
		{"i", "i", `<svg id=o><foreignObject id=f><i id=h>x</foreignObject><rect id=r></rect></svg>z`, 80, 38},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			outer := svgNodeByID(t, document, "o")
			foreignObject := svgNodeByID(t, document, "f")
			html := svgNodeByID(t, document, "h")
			rect := svgNodeByID(t, document, "r")
			assertFormattingNode(t, outer, "svg", "xz", 0, test.length)
			assertFormattingNode(t, foreignObject, "foreignObject", "xz", 10, test.length)
			assertFormattingNode(t, html, test.element, "xz", 30, test.length)
			if html.NamespaceURI != htmlNamespaceURI || rect.NamespaceURI != htmlNamespaceURI || html.Parent != foreignObject || rect.Parent != html || len(html.Children) != 3 || html.Children[0].Value != "x" || html.Children[0].StartPos != test.childStart || html.Children[1] != rect || html.Children[2].Value != "z" || html.Children[2].EndPos != test.length {
				t.Fatalf("Foreign owner end crossed live HTML <%s>: outer=%#v fo=%#v html=%#v", test.element, outer, foreignObject, html)
			}
		})
	}
}

func TestParseSVGBreakoutUsesDocumentEOFRecovery(t *testing.T) {
	const content = `<svg><g><div>x</div></g></svg><section>y`
	parser := NewHTMLParser()
	if document, err := parser.Parse(content); err != nil || len(svgNodesNamed(document, "section")) != 1 || svgNodesNamed(document, "section")[0].EndPos != len(content) {
		t.Fatalf("SVG breakout implicit-document EOF mismatch: doc=%#v err=%v", document, err)
	}
	document, err := parser.Parse(`<svg><g><circle /></g></svg>z`)
	if err != nil || len(svgNodesNamed(document, "svg")) != 1 || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].Value != "z" {
		t.Fatalf("Strict breakout failure leaked into parser reuse: doc=%#v err=%v", document, err)
	}
}

func TestParseSVGBreakoutStaleForeignNameDoesNotConsumeHTMLEnd(t *testing.T) {
	const content = `<svg><g><div>x</div><g id=h><span id=s>y</g>z</span>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	div := svgNodesNamed(document, "div")[0]
	htmlG := svgNodeByID(t, document, "h")
	span := svgNodeByID(t, document, "s")
	assertFormattingNode(t, svg, "svg", "", 0, 8)
	assertFormattingNode(t, div, "div", "x", 8, 20)
	assertFormattingNode(t, htmlG, "g", "y", 20, 44)
	assertFormattingNode(t, span, "span", "y", 28, 40)
	if htmlG.NamespaceURI != htmlNamespaceURI || span.NamespaceURI != htmlNamespaceURI || span.Parent != htmlG || len(parsedBodyChildren(document)) != 4 || parsedBodyChildren(document)[2] != htmlG || parsedBodyChildren(document)[3].Value != "z" || parsedBodyChildren(document)[3].StartPos != 44 || parsedBodyChildren(document)[3].EndPos != 45 {
		t.Fatalf("Stale foreign-name debt consumed the real HTML g end: doc=%#v g=%#v span=%#v", parsedBodyChildren(document), htmlG, span)
	}
}

func TestParseSVGBreakoutReprocessesDocumentElementStarts(t *testing.T) {
	tests := []struct {
		name, content      string
		length, tokenStart int
		bodyClass          string
	}{
		{
			"body merges attributes",
			`<html><head></head><body id=b><div id=o>a<svg><g><body class=x>c</div></body></html>`,
			84,
			49,
			"x",
		},
		{
			"head is ignored",
			`<html><head></head><body id=b><div id=o>a<svg><g><head id=h>c</div></body></html>`,
			81,
			49,
			"",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			htmlNodes := svgNodesNamed(document, "html")
			headNodes := svgNodesNamed(document, "head")
			bodyNodes := svgNodesNamed(document, "body")
			if len(htmlNodes) != 1 || len(headNodes) != 1 || len(bodyNodes) != 1 {
				t.Fatalf("Document-element start created nested document nodes: html=%#v head=%#v body=%#v", htmlNodes, headNodes, bodyNodes)
			}
			html, head, body := htmlNodes[0], headNodes[0], bodyNodes[0]
			div := svgNodeByID(t, document, "o")
			svg := svgNodesNamed(document, "svg")[0]
			g := svgNodesNamed(document, "g")[0]
			assertFormattingNode(t, html, "html", "ac", 0, test.length)
			assertFormattingNode(t, head, "head", "", 6, 19)
			bodyEnd, divEnd, contentEnd := test.length-7, test.length-14, test.length-20
			assertFormattingNode(t, body, "body", "ac", 19, bodyEnd)
			assertFormattingNode(t, div, "div", "ac", 30, divEnd)
			assertFormattingNode(t, svg, "svg", "", 41, test.tokenStart)
			assertFormattingNode(t, g, "g", "", 46, test.tokenStart)
			if html.ContentStart != 6 || html.ContentEnd != test.length-7 || head.ContentStart != 12 || head.ContentEnd != 12 || body.ContentStart != 30 || body.ContentEnd != test.length-14 || div.ContentStart != 40 || div.ContentEnd != contentEnd {
				t.Fatalf("Document breakout source boundaries differ from Chrome: html=%#v head=%#v body=%#v div=%#v", html, head, body, div)
			}
			if body.Attributes["id"] != "b" || body.Attributes["class"] != test.bodyClass || head.Attributes["id"] != "" || len(div.Children) != 3 || div.Children[0].Value != "a" || div.Children[1] != svg || div.Children[2].Value != "c" || div.Children[2].StartPos != contentEnd-1 || div.Children[2].EndPos != contentEnd {
				t.Fatalf("Document start was not reprocessed with HTML document-token semantics: html=%#v head=%#v body=%#v div=%#v", html, head, body, div)
			}
		})
	}
}

func TestParseSVGIntegrationReprocessesDocumentElementStarts(t *testing.T) {
	const content = `<html id=o><head></head><body id=b><svg><foreignObject id=f><html class=x><br><body class=y><br><head id=h><br></foreignObject></svg></body></html>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	htmlNodes := svgNodesNamed(document, "html")
	headNodes := svgNodesNamed(document, "head")
	bodyNodes := svgNodesNamed(document, "body")
	brNodes := svgNodesNamed(document, "br")
	if len(htmlNodes) != 1 || len(headNodes) != 1 || len(bodyNodes) != 1 || len(brNodes) != 3 {
		t.Fatalf("Integration-point document starts created nested nodes: html=%#v head=%#v body=%#v br=%#v", htmlNodes, headNodes, bodyNodes, brNodes)
	}
	html, head, body := htmlNodes[0], headNodes[0], bodyNodes[0]
	foreignObject := svgNodeByID(t, document, "f")
	assertFormattingNode(t, html, "html", "", 0, 147)
	assertFormattingNode(t, head, "head", "", 11, 24)
	assertFormattingNode(t, body, "body", "", 24, 140)
	assertFormattingNode(t, foreignObject, "foreignObject", "", 40, 127)
	if html.Attributes["id"] != "o" || html.Attributes["class"] != "x" || body.Attributes["id"] != "b" || body.Attributes["class"] != "y" || head.Attributes["id"] != "" {
		t.Fatalf("Integration document attributes were not merged/ignored correctly: html=%#v head=%#v body=%#v", html.Attributes, head.Attributes, body.Attributes)
	}
	for _, test := range []struct {
		node       *types.Node
		attributes []string
	}{
		{html, []string{"id", "class"}},
		{body, []string{"id", "class"}},
	} {
		if len(test.node.AttributeOrder) != 2 || test.node.AttributeOrder[0] != "id" || test.node.AttributeOrder[1] != "class" {
			t.Fatalf("Duplicate <%s> start did not preserve merged attribute order [id class]: %#v", test.node.Name, test.node.AttributeOrder)
		}
		for _, name := range test.attributes {
			if test.node.AttributeNamespaces[name] != "" || test.node.AttributeLocalNames[name] != name || test.node.AttributePrefixes[name] != "" {
				t.Fatalf("Duplicate document-start merge omitted metadata for <%s> @%s: ns=%q local=%q prefix=%q", test.node.Name, name, test.node.AttributeNamespaces[name], test.node.AttributeLocalNames[name], test.node.AttributePrefixes[name])
			}
		}
	}
	expectedRanges := [][2]int{{74, 78}, {92, 96}, {107, 111}}
	if len(foreignObject.Children) != 3 {
		t.Fatalf("Expected exactly three br children under foreignObject, got %#v", foreignObject.Children)
	}
	for i, br := range brNodes {
		if br.Parent != foreignObject || br.NamespaceURI != htmlNamespaceURI || foreignObject.Children[i] != br || br.StartPos != expectedRanges[i][0] || br.EndPos != expectedRanges[i][1] || br.ContentStart != expectedRanges[i][1] || br.ContentEnd != expectedRanges[i][1] {
			t.Fatalf("Integration document token %d was not reprocessed as an HTML br child: %#v", i, br)
		}
	}
}

func TestParseSVGIgnoredForeignEndSourceRangeContrasts(t *testing.T) {
	tests := []struct {
		name, content, value string
		textStart, textEnd   int
	}{
		{"terminal complete end", `<svg>x</foo></svg>`, "x", 5, 6},
		{"complete end between text", `<svg>x</foo>y</svg>`, "xy", 5, 13},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			svg := svgNodesNamed(document, "svg")[0]
			if len(svg.Children) != 1 || svg.Children[0].Type != types.TextNode || svg.Children[0].Value != test.value || svg.Children[0].StartPos != test.textStart || svg.Children[0].EndPos != test.textEnd {
				t.Fatalf("Complete ignored foreign end leaked into text range: svg=%#v", svg)
			}
		})
	}

	const blocked = `<svg><foreignObject><p>x</foreignObject></p></svg>`
	document, err := NewHTMLParser().Parse(blocked)
	if err != nil {
		t.Fatal(err)
	}
	p := svgNodesNamed(document, "p")[0]
	if len(p.Children) != 1 || p.Children[0].Value != "x" || p.Children[0].StartPos != 23 || p.Children[0].EndPos != 24 {
		t.Fatalf("Blocked complete foreign owner end leaked into terminal p text range: %#v", p)
	}

	const staleIncomplete = `<svg><g><div>x</div>y</g`
	document, err = NewHTMLParser().Parse(staleIncomplete)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[2].Type != types.TextNode || parsedBodyChildren(document)[2].Value != "y" || parsedBodyChildren(document)[2].StartPos != 20 || parsedBodyChildren(document)[2].EndPos != 24 {
		t.Fatalf("Discarded incomplete stale foreign end did not extend preceding text through EOF: %#v", parsedBodyChildren(document))
	}
}

func TestParseSVGIntegrationIgnoredHTMLEndFallbacks(t *testing.T) {
	tests := []struct {
		name, content, element, value string
		elementStart, elementEnd      int
		textStart, textEnd            int
	}{
		{
			"terminal absent generic end",
			`<svg><foreignObject><div>x</foo></div></foreignObject></svg>`,
			"div", "x", 20, 38, 25, 26,
		},
		{
			"absent generic end between text",
			`<svg><foreignObject><div>x</foo>y</div></foreignObject></svg>`,
			"div", "xy", 20, 39, 25, 33,
		},
		{
			"special absent end under live span",
			`<svg><foreignObject><span>x</div>y</span></foreignObject></svg>`,
			"span", "xy", 20, 41, 26, 34,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			elements := svgNodesNamed(document, test.element)
			if len(elements) != 1 {
				t.Fatalf("Expected one <%s>, got %#v", test.element, elements)
			}
			element := elements[0]
			assertFormattingNode(t, element, test.element, test.value, test.elementStart, test.elementEnd)
			if element.NamespaceURI != htmlNamespaceURI || len(element.Children) != 1 || element.Children[0].Type != types.TextNode || element.Children[0].Value != test.value || element.Children[0].StartPos != test.textStart || element.Children[0].EndPos != test.textEnd {
				t.Fatalf("Ignored HTML end fallback leaked into terminal/adjacent text: %#v", element)
			}
		})
	}

	const barrier = `<div><svg><foreignObject><span>x</div>y</span></foreignObject></svg></div>`
	document, err := NewHTMLParser().Parse(barrier)
	if err != nil {
		t.Fatal(err)
	}
	div := svgNodesNamed(document, "div")[0]
	svg := svgNodesNamed(document, "svg")[0]
	foreignObject := svgNodesNamed(document, "foreignObject")[0]
	span := svgNodesNamed(document, "span")[0]
	assertFormattingNode(t, div, "div", "xy", 0, 74)
	assertFormattingNode(t, svg, "svg", "xy", 5, 68)
	assertFormattingNode(t, foreignObject, "foreignObject", "xy", 10, 62)
	assertFormattingNode(t, span, "span", "xy", 25, 46)
	if span.Parent != foreignObject || foreignObject.Parent != svg || svg.Parent != div || len(span.Children) != 1 || span.Children[0].Value != "xy" || span.Children[0].StartPos != 31 || span.Children[0].EndPos != 39 {
		t.Fatalf("Outer HTML end crossed the SVG/integration barrier: div=%#v svg=%#v foreignObject=%#v span=%#v", div, svg, foreignObject, span)
	}
}

func TestParseSVGIntegrationCommentsAndContinuation(t *testing.T) {
	const content = `<svg><desc><!--c--><span>x</span>y</desc>z</svg>q`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	desc := svgNodesNamed(document, "desc")[0]
	span := svgNodesNamed(document, "span")[0]
	assertFormattingNode(t, svg, "svg", "xyz", 0, 48)
	assertFormattingNode(t, desc, "desc", "xy", 5, 41)
	assertFormattingNode(t, span, "span", "x", 19, 33)
	if len(desc.Children) != 3 || desc.Children[0].Type != types.CommentNode || desc.Children[0].Value != "c" || desc.Children[0].StartPos != 11 || desc.Children[0].EndPos != 19 || desc.Children[2].Value != "y" || desc.Children[2].StartPos != 33 || len(svg.Children) != 2 || svg.Children[1].Value != "z" || parsedBodyChildren(document)[1].Value != "q" {
		t.Fatalf("Integration comment/text ordering mismatch: desc=%#v svg=%#v doc=%#v", desc.Children, svg.Children, parsedBodyChildren(document))
	}
}

func bestSVGIntegrationDuration(t *testing.T, content string) time.Duration {
	t.Helper()
	best := time.Duration(1<<63 - 1)
	for i := 0; i < 3; i++ {
		start := time.Now()
		if _, err := NewHTMLParser().Parse(content); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed < best {
			best = elapsed
		}
	}
	return best
}

func TestParseSVGIntegrationAndBreakoutScaling(t *testing.T) {
	builders := map[string]func(int) string{
		"integration": func(n int) string { return strings.Repeat(`<svg><foreignObject><div>x</div></foreignObject></svg>`, n) },
		"breakout":    func(n int) string { return strings.Repeat(`<svg><g><div>x</div>`, n) },
	}
	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			smallInput, largeInput := build(500), build(2000)
			_ = bestSVGIntegrationDuration(t, smallInput)
			small := bestSVGIntegrationDuration(t, smallInput)
			large := bestSVGIntegrationDuration(t, largeInput)
			if large > 10*small && large-small > 100*time.Millisecond {
				t.Fatalf("SVG %s parsing scaled superlinearly: small=%s large=%s", name, small, large)
			}
		})
	}
}
