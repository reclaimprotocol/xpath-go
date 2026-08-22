package utils

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

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
