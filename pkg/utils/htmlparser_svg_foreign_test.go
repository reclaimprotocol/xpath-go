package utils

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 17A covers closed SVG foreign islands parsed from an HTML document.
// Chrome is authoritative for namespace, adjusted-name, and source-location
// behavior. Integration points/breakout, MathML, XLink/XML attributes,
// templates, table/select modes, adoption, fragments, and foreign scripting
// are intentionally deferred.

const svgNamespaceURI = "http://www.w3.org/2000/svg"
const htmlNamespaceURI = "http://www.w3.org/1999/xhtml"

func svgNodeByID(t *testing.T, node *types.Node, id string) *types.Node {
	t.Helper()
	if node != nil && node.Attributes["id"] == id {
		return node
	}
	if node != nil {
		for _, child := range node.Children {
			if found := svgNodeByID(t, child, id); found != nil {
				return found
			}
		}
	}
	return nil
}

func svgNodesNamed(node *types.Node, name string) []*types.Node {
	if node == nil {
		return nil
	}
	var result []*types.Node
	if node.Type == types.ElementNode && node.Name == name {
		result = append(result, node)
	}
	for _, child := range node.Children {
		result = append(result, svgNodesNamed(child, name)...)
	}
	return result
}

func requireNodeNamespace(t *testing.T, node *types.Node, expected string) {
	t.Helper()
	if node == nil {
		t.Fatal("Expected node for namespace assertion")
	}
	field := reflect.ValueOf(node).Elem().FieldByName("NamespaceURI")
	if !field.IsValid() {
		t.Fatalf("Node must expose NamespaceURI=%q: %#v", expected, node)
	}
	if field.Kind() != reflect.String || field.String() != expected {
		t.Fatalf("Expected <%s> NamespaceURI %q, got %v", node.Name, expected, field.Interface())
	}
}

func TestParseClosedSVGIslandNamespaceTreeAndExit(t *testing.T) {
	const content = `<div>a<svg id=s><circle id=c>t</circle></svg>b</div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	div := svgNodesNamed(document, "div")[0]
	svg := svgNodeByID(t, document, "s")
	circle := svgNodeByID(t, document, "c")
	assertFormattingNode(t, div, "div", "atb", 0, 52)
	assertFormattingNode(t, svg, "svg", "t", 6, 45)
	assertFormattingNode(t, circle, "circle", "t", 16, 39)
	if div.ContentStart != 5 || div.ContentEnd != 46 || svg.ContentStart != 16 || svg.ContentEnd != 39 || circle.ContentStart != 29 || circle.ContentEnd != 30 {
		t.Fatalf("SVG content boundaries differ from Chrome: div=%#v svg=%#v circle=%#v", div, svg, circle)
	}
	if len(div.Children) != 3 || div.Children[0].Value != "a" || div.Children[0].StartPos != 5 || div.Children[1] != svg || div.Children[2].Value != "b" || div.Children[2].StartPos != 45 {
		t.Fatalf("SVG island did not return to the HTML parent: %#v", div.Children)
	}
	requireNodeNamespace(t, div, htmlNamespaceURI)
	requireNodeNamespace(t, svg, svgNamespaceURI)
	requireNodeNamespace(t, circle, svgNamespaceURI)
	requireNodeNamespace(t, circle.Children[0], "")
}

func TestParseForeignSelfClosingElementsAndRootSVG(t *testing.T) {
	const content = `<svg><g id=a /><path id=p /></svg><p>z</p>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	g := svgNodeByID(t, document, "a")
	path := svgNodeByID(t, document, "p")
	paragraph := svgNodesNamed(document, "p")[0]
	assertFormattingNode(t, svg, "svg", "", 0, 34)
	assertFormattingNode(t, g, "g", "", 5, 15)
	assertFormattingNode(t, path, "path", "", 15, 28)
	assertFormattingNode(t, paragraph, "p", "z", 34, 42)
	for _, node := range []*types.Node{g, path} {
		if node.ContentStart != node.EndPos || node.ContentEnd != node.EndPos {
			t.Fatalf("Foreign self-closing node must have empty content at its source end: %#v", node)
		}
		requireNodeNamespace(t, node, svgNamespaceURI)
	}

	const root = `<svg id=s />z`
	document, err = NewHTMLParser().Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	svg = svgNodeByID(t, document, "s")
	assertFormattingNode(t, svg, "svg", "", 0, 12)
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0] != svg || parsedBodyChildren(document)[1].Value != "z" || parsedBodyChildren(document)[1].StartPos != 12 || parsedBodyChildren(document)[1].EndPos != 13 {
		t.Fatalf("Root self-closing SVG did not return to HTML data state: %#v", parsedBodyChildren(document))
	}
}

func TestParseSVGAdjustsElementAndAttributeNames(t *testing.T) {
	const names = `<svg><lineargradient id=g><clippath id=c><feblend id=f /></clippath></lineargradient></svg>`
	document, err := NewHTMLParser().Parse(names)
	if err != nil {
		t.Fatal(err)
	}
	linear := svgNodeByID(t, document, "g")
	clip := svgNodeByID(t, document, "c")
	blend := svgNodeByID(t, document, "f")
	assertFormattingNode(t, linear, "linearGradient", "", 5, 85)
	assertFormattingNode(t, clip, "clipPath", "", 26, 68)
	assertFormattingNode(t, blend, "feBlend", "", 41, 57)
	if linear.Parent == nil || clip.Parent != linear || blend.Parent != clip || blend.ContentStart != 57 || blend.ContentEnd != 57 {
		t.Fatalf("Adjusted SVG chain mismatch: linear=%#v clip=%#v blend=%#v", linear, clip, blend)
	}

	const attrs = `<svg id=s viewbox="0 0 1 1" preserveaspectratio=x><lineargradient id=g gradientunits=u attributename=fill /></svg>`
	document, err = NewHTMLParser().Parse(attrs)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodeByID(t, document, "s")
	linear = svgNodeByID(t, document, "g")
	assertFormattingNode(t, svg, "svg", "", 0, 114)
	assertFormattingNode(t, linear, "linearGradient", "", 50, 108)
	if svg.Attributes["viewBox"] != "0 0 1 1" || svg.Attributes["preserveAspectRatio"] != "x" || linear.Attributes["gradientUnits"] != "u" || linear.Attributes["attributeName"] != "fill" {
		t.Fatalf("SVG attributes were not adjusted like Chrome: svg=%#v linear=%#v", svg.Attributes, linear.Attributes)
	}
	for _, oldName := range []string{"viewbox", "preserveaspectratio", "gradientunits", "attributename"} {
		if _, exists := svg.Attributes[oldName]; exists {
			t.Fatalf("Unadjusted SVG attribute %q remained on svg: %#v", oldName, svg.Attributes)
		}
		if _, exists := linear.Attributes[oldName]; exists {
			t.Fatalf("Unadjusted SVG attribute %q remained on linearGradient: %#v", oldName, linear.Attributes)
		}
	}
}

func TestParseSVGCaseInsensitiveCloseAndAncestorPop(t *testing.T) {
	const caseClose = `<svg><linearGradient id=g>x</LINEARGRADIENT>z</svg>q`
	document, err := NewHTMLParser().Parse(caseClose)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	linear := svgNodeByID(t, document, "g")
	assertFormattingNode(t, svg, "svg", "xz", 0, 51)
	assertFormattingNode(t, linear, "linearGradient", "x", 5, 44)
	if len(svg.Children) != 2 || svg.Children[1].Value != "z" || svg.Children[1].StartPos != 44 || parsedBodyChildren(document)[1].Value != "q" || parsedBodyChildren(document)[1].StartPos != 51 {
		t.Fatalf("Foreign close/HTML continuation mismatch: svg=%#v doc=%#v", svg.Children, parsedBodyChildren(document))
	}

	const ancestor = `<svg><g><path id=p></g><circle id=c></svg>z`
	document, err = NewHTMLParser().Parse(ancestor)
	if err != nil {
		t.Fatal(err)
	}
	svg = svgNodesNamed(document, "svg")[0]
	g := svgNodesNamed(document, "g")[0]
	path := svgNodeByID(t, document, "p")
	circle := svgNodeByID(t, document, "c")
	assertFormattingNode(t, svg, "svg", "", 0, 42)
	assertFormattingNode(t, g, "g", "", 5, 23)
	assertFormattingNode(t, path, "path", "", 8, 19)
	assertFormattingNode(t, circle, "circle", "", 23, 36)
	if path.ContentStart != 19 || path.ContentEnd != 19 || circle.ContentStart != 36 || circle.ContentEnd != 36 || parsedBodyChildren(document)[1].Value != "z" {
		t.Fatalf("Foreign ancestor-pop ranges mismatch: path=%#v circle=%#v doc=%#v", path, circle, parsedBodyChildren(document))
	}
}

func TestParseSVGCDATAReferencesNULAndComments(t *testing.T) {
	const tokens = "<svg><![CDATA[a<b>&copy;]]><text>c&amp;d\x00e</text></svg>z"
	document, err := NewHTMLParser().Parse(tokens)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	textElement := svgNodesNamed(document, "text")[0]
	assertFormattingNode(t, svg, "svg", "a<b>&copy;c&d�e", 0, 55)
	assertFormattingNode(t, textElement, "text", "c&d�e", 27, 49)
	if len(svg.Children) != 2 || svg.Children[0].Type != types.TextNode || svg.Children[0].Value != "a<b>&copy;" || svg.Children[0].StartPos != 5 || svg.Children[0].EndPos != 27 {
		t.Fatalf("Foreign CDATA must emit literal text over its raw declaration: %#v", svg.Children)
	}
	if len(textElement.Children) != 1 || textElement.Children[0].Value != "c&d�e" || textElement.Children[0].StartPos != 33 || textElement.Children[0].EndPos != 42 {
		t.Fatalf("Foreign references/NUL mismatch: %#v", textElement.Children)
	}

	const ordered = `<svg><!--c--><text>x</text><![CDATA[y]]></svg>`
	document, err = NewHTMLParser().Parse(ordered)
	if err != nil {
		t.Fatal(err)
	}
	svg = svgNodesNamed(document, "svg")[0]
	if svg.TextContent != "xy" || len(svg.Children) != 3 || svg.Children[0].Type != types.CommentNode || svg.Children[0].Value != "c" || svg.Children[0].StartPos != 5 || svg.Children[0].EndPos != 13 || svg.Children[1].Name != "text" || svg.Children[2].Type != types.TextNode || svg.Children[2].Value != "y" || svg.Children[2].StartPos != 27 || svg.Children[2].EndPos != 40 {
		t.Fatalf("Comment/CDATA order differs from Chrome: %#v", svg.Children)
	}
	requireNodeNamespace(t, svg.Children[0], "")
	requireNodeNamespace(t, svg.Children[2], "")
}

func TestParseSVGIgnoresDoctypeAndExtendsCoalescedTextRange(t *testing.T) {
	const content = `<svg>a<!DOCTYPE x>b<circle /></svg>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	circle := svgNodesNamed(document, "circle")[0]
	assertFormattingNode(t, svg, "svg", "ab", 0, 35)
	assertFormattingNode(t, circle, "circle", "", 19, 29)
	if len(svg.Children) != 2 || svg.Children[0].Type != types.TextNode || svg.Children[0].Value != "ab" || svg.Children[0].StartPos != 5 || svg.Children[0].EndPos != 19 {
		t.Fatalf("Ignored foreign doctype must coalesce adjacent text across its source: %#v", svg.Children)
	}
	if parsedBodyChildren(document)[1].Value != "z" || parsedBodyChildren(document)[1].StartPos != 35 || parsedBodyChildren(document)[1].EndPos != 36 {
		t.Fatalf("Foreign doctype changed post-SVG continuation: %#v", parsedBodyChildren(document))
	}
}

func TestParseSVGEOFUnicodeIncompleteEndAndCDATA(t *testing.T) {
	const unicode = "<div><svg><g>é\r\n😀"
	parser := NewHTMLParser()
	document, err := parser.Parse(unicode)
	if err != nil {
		t.Fatal(err)
	}
	div := svgNodesNamed(document, "div")[0]
	svg := svgNodesNamed(document, "svg")[0]
	g := svgNodesNamed(document, "g")[0]
	assertFormattingNode(t, div, "div", "é\n😀", 0, 21)
	assertFormattingNode(t, svg, "svg", "é\n😀", 5, 21)
	assertFormattingNode(t, g, "g", "é\n😀", 10, 21)
	if len(g.Children) != 1 || g.Children[0].Value != "é\n😀" || g.Children[0].StartPos != 13 || g.Children[0].EndPos != 21 || g.Children[0].StartLine != 1 || g.Children[0].StartColumn != 14 || g.Children[0].EndLine != 2 || g.Children[0].EndColumn != 3 {
		t.Fatalf("Foreign UTF-8/UTF-16 source coordinates mismatch: %#v", g.Children)
	}
	for _, node := range []*types.Node{div, svg, g} {
		if node.EndLine != 2 || node.EndColumn != 3 {
			t.Fatalf("EOF did not propagate browser endpoint through <%s>: %#v", node.Name, node)
		}
	}
	document, err = parser.Parse(`<div>x`)
	if err != nil {
		t.Fatalf("Universal document EOF recovery failed after SVG parser reuse: %v", err)
	}
	reusedDiv := svgNodesNamed(document, "div")
	if len(reusedDiv) != 1 || reusedDiv[0].StartPos != 0 || reusedDiv[0].EndPos != 6 || reusedDiv[0].TextContent != "x" || reusedDiv[0].Parent != parsedBody(document) {
		t.Fatalf("SVG parser reuse leaked state into universal document parse: %#v", reusedDiv)
	}

	const incompleteEnd = `<svg><g>x</g`
	document, err = NewHTMLParser().Parse(incompleteEnd)
	if err != nil {
		t.Fatal(err)
	}
	svg = svgNodesNamed(document, "svg")[0]
	g = svgNodesNamed(document, "g")[0]
	if svg.EndPos != 12 || g.EndPos != 12 || len(g.Children) != 1 || g.Children[0].Value != "x" || g.Children[0].StartPos != 8 || g.Children[0].EndPos != 12 || g.Children[0].EndColumn != 13 {
		t.Fatalf("Discarded foreign end tag did not extend preceding text through EOF: svg=%#v g=%#v", svg, g)
	}

	const incompleteCDATA = `<svg><![CDATA[x]]`
	document, err = NewHTMLParser().Parse(incompleteCDATA)
	if err != nil {
		t.Fatal(err)
	}
	svg = svgNodesNamed(document, "svg")[0]
	if svg.TextContent != "x]]" || svg.EndPos != 17 || len(svg.Children) != 1 || svg.Children[0].Type != types.TextNode || svg.Children[0].Value != "x]]" || svg.Children[0].StartPos != 5 || svg.Children[0].EndPos != 17 {
		t.Fatalf("Incomplete foreign CDATA must become literal text through EOF: %#v", svg)
	}
}

func TestParseSVGReturnsToFormattingState(t *testing.T) {
	const content = `<b><svg><circle /></svg>x</b>y`
	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := svgNodesNamed(document, "b")[0]
	svg := svgNodesNamed(document, "svg")[0]
	circle := svgNodesNamed(document, "circle")[0]
	assertFormattingNode(t, bold, "b", "x", 0, 29)
	assertFormattingNode(t, svg, "svg", "", 3, 24)
	assertFormattingNode(t, circle, "circle", "", 8, 18)
	if len(bold.Children) != 2 || bold.Children[0] != svg || bold.Children[1].Value != "x" || bold.Children[1].StartPos != 24 || len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].Value != "y" {
		t.Fatalf("SVG exit corrupted active-formatting state: bold=%#v doc=%#v", bold.Children, parsedBodyChildren(document))
	}
	document, err = parser.Parse(`<p>z</p>`)
	if err != nil || len(svgNodesNamed(document, "svg")) != 0 || len(svgNodesNamed(document, "b")) != 0 || svgNodesNamed(document, "p")[0].TextContent != "z" {
		t.Fatalf("SVG/formatting state leaked across parser reuse: doc=%#v err=%v", document, err)
	}
}

func TestParseSVGForeignTokenizerMakesProgressAndHandlesBogusComments(t *testing.T) {
	const emptyEnd = `<svg>a</>b</svg>z`
	document, err := NewHTMLParser().Parse(emptyEnd)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	assertFormattingNode(t, svg, "svg", "ab", 0, 16)
	if svg.ContentStart != 5 || svg.ContentEnd != 10 || len(svg.Children) != 1 || svg.Children[0].Type != types.TextNode || svg.Children[0].Value != "ab" || svg.Children[0].StartPos != 5 || svg.Children[0].EndPos != 10 {
		t.Fatalf("Ignored empty foreign end tag must preserve progress and coalesce text: %#v", svg)
	}
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].Value != "z" || parsedBodyChildren(document)[1].StartPos != 16 || parsedBodyChildren(document)[1].EndPos != 17 {
		t.Fatalf("Empty foreign end tag changed continuation: %#v", parsedBodyChildren(document))
	}

	const bogus = `<svg>a<!foo>b</svg>z`
	document, err = NewHTMLParser().Parse(bogus)
	if err != nil {
		t.Fatal(err)
	}
	svg = svgNodesNamed(document, "svg")[0]
	assertFormattingNode(t, svg, "svg", "ab", 0, 19)
	if len(svg.Children) != 3 || svg.Children[0].Value != "a" || svg.Children[0].StartPos != 5 || svg.Children[0].EndPos != 6 || svg.Children[1].Type != types.CommentNode || svg.Children[1].Value != "foo" || svg.Children[1].StartPos != 6 || svg.Children[1].EndPos != 12 || svg.Children[2].Value != "b" || svg.Children[2].StartPos != 12 || svg.Children[2].EndPos != 13 {
		t.Fatalf("Foreign bogus declaration must become a comment between text nodes: %#v", svg.Children)
	}
}

func TestParseSVGAdditionalAdjustedNameAndASCIIFolding(t *testing.T) {
	const adjusted = `<svg><fedropshadow id=f /></svg>`
	document, err := NewHTMLParser().Parse(adjusted)
	if err != nil {
		t.Fatal(err)
	}
	drop := svgNodeByID(t, document, "f")
	assertFormattingNode(t, drop, "feDropShadow", "", 5, 26)
	requireNodeNamespace(t, drop, svgNamespaceURI)

	const unicode = `<svg><aÀ id=x>y</AÀ></svg>`
	document, err = NewHTMLParser().Parse(unicode)
	if err != nil {
		t.Fatal(err)
	}
	node := svgNodeByID(t, document, "x")
	assertFormattingNode(t, node, "aÀ", "y", 5, 22)
	if node.EndLine != 1 || node.EndColumn != 21 {
		t.Fatalf("Foreign tag folding/coordinates must use ASCII, not Unicode: %#v", node)
	}

	if node.ContentStart != 15 || node.ContentEnd != 16 || node.Parent == nil || node.Parent.Name != "svg" {
		t.Fatalf("Foreign non-ASCII start/end tag names were not symmetric: %#v", node)
	}
}

func TestParseSVGMalformedEndTagsAlwaysAdvance(t *testing.T) {
	const missing = `<svg>a</>b</svg>z`
	document, err := NewHTMLParser().Parse(missing)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	assertFormattingNode(t, svg, "svg", "ab", 0, 16)
	if len(svg.Children) != 1 || svg.Children[0].Value != "ab" || svg.Children[0].StartPos != 5 || svg.Children[0].EndPos != 10 || parsedBodyChildren(document)[1].Value != "z" {
		t.Fatalf("Nameless foreign end recovery mismatch: svg=%#v document=%#v", svg.Children, parsedBodyChildren(document))
	}

	const bogus = `<svg>a</!x>b</svg>z`
	document, err = NewHTMLParser().Parse(bogus)
	if err != nil {
		t.Fatal(err)
	}
	svg = svgNodesNamed(document, "svg")[0]
	if svg.TextContent != "ab" || len(svg.Children) != 3 || svg.Children[1].Type != types.CommentNode || svg.Children[1].Value != "!x" || parsedBodyChildren(document)[1].Value != "z" {
		t.Fatalf("Bogus foreign end recovery mismatch: svg=%#v document=%#v", svg.Children, parsedBodyChildren(document))
	}
}

func TestParseNamespaceIncludesSyntheticHTMLNodes(t *testing.T) {
	document, err := NewHTMLParser().Parse(`<table><tr><td>x</table>`)
	if err != nil {
		t.Fatal(err)
	}
	tbody := svgNodesNamed(document, "tbody")[0]
	requireNodeNamespace(t, tbody, htmlNamespaceURI)

	document, err = NewHTMLParser().Parse(`<p><b>x</p>y`)
	if err != nil {
		t.Fatal(err)
	}
	bolds := svgNodesNamed(document, "b")
	if len(bolds) != 2 {
		t.Fatalf("Expected source and reconstructed b, got %#v", bolds)
	}
	for _, bold := range bolds {
		requireNodeNamespace(t, bold, htmlNamespaceURI)
	}
}

func TestParseSVGUnknownNamesUseASCIIFoldingOnly(t *testing.T) {
	document, err := NewHTMLParser().Parse(`<svg><AÀ id=x>y</aÀ></svg>`)
	if err != nil {
		t.Fatal(err)
	}
	foreign := svgNodeByID(t, document, "x")
	assertFormattingNode(t, foreign, "aÀ", "y", 5, 22)
	requireNodeNamespace(t, foreign, svgNamespaceURI)
}

func TestParseSVGAdjustsFeDropShadow(t *testing.T) {
	document, err := NewHTMLParser().Parse(`<svg><fedropshadow id=f /></svg>`)
	if err != nil {
		t.Fatal(err)
	}
	shadow := svgNodeByID(t, document, "f")
	assertFormattingNode(t, shadow, "feDropShadow", "", 5, 26)
}

func TestParseSVGForeignCloseDoesNotUnicodeFold(t *testing.T) {
	document, err := NewHTMLParser().Parse(`<svg><aÀ id=x>y</aà>z</svg>`)
	if err != nil {
		t.Fatal(err)
	}
	foreign := svgNodeByID(t, document, "x")
	if foreign.Name != "aÀ" || foreign.TextContent != "yz" {
		t.Fatalf("Foreign close must use ASCII-only matching: %#v", foreign)
	}
}

func TestParseSVGBogusCommentOrder(t *testing.T) {
	document, err := NewHTMLParser().Parse(`<svg>a<!foo>b</svg>z`)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	if svg.TextContent != "ab" || len(svg.Children) != 3 || svg.Children[1].Type != types.CommentNode || svg.Children[1].Value != "foo" || svg.Children[1].StartPos != 6 || svg.Children[1].EndPos != 12 || svg.Children[2].Value != "b" || svg.Children[2].StartPos != 12 {
		t.Fatalf("Foreign bogus-comment recovery mismatch: %#v", svg.Children)
	}
}

func bestSVGParseDuration(t *testing.T, content string) time.Duration {
	t.Helper()
	best := time.Duration(1<<63 - 1)
	for i := 0; i < 3; i++ {
		started := time.Now()
		if _, err := NewHTMLParser().Parse(content); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(started); elapsed < best {
			best = elapsed
		}
	}
	return best
}

func TestParseSVGForeignIslandScaling(t *testing.T) {
	buildWide := func(n int) string { return `<svg>` + strings.Repeat(`<g /><path />`, n) + `</svg>` }
	buildPops := func(n int) string { return `<svg>` + strings.Repeat(`<g><path></g>`, n) + `</svg>` }
	for name, build := range map[string]func(int) string{"wide-self-close": buildWide, "ancestor-pop": buildPops} {
		t.Run(name, func(t *testing.T) {
			smallInput, largeInput := build(500), build(2000)
			_ = bestSVGParseDuration(t, smallInput)
			small := bestSVGParseDuration(t, smallInput)
			large := bestSVGParseDuration(t, largeInput)
			if large > 10*small && large-small > 100*time.Millisecond {
				t.Fatalf("SVG foreign parsing scaled superlinearly from 500 to 2000 tokens: small=%s large=%s", small, large)
			}
		})
	}
}
