package xpath_test

import (
	"reflect"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

const svgResultNamespaceURI = "http://www.w3.org/2000/svg"

func requireResultNamespace(t *testing.T, result xpath.Result, expected string) {
	t.Helper()
	field := reflect.ValueOf(result).FieldByName("NamespaceURI")
	if !field.IsValid() {
		t.Fatalf("XPath Result must expose NamespaceURI=%q: %#v", expected, result)
	}
	if field.Kind() != reflect.String || field.String() != expected {
		t.Fatalf("Expected <%s> result namespace %q, got %v", result.NodeName, expected, field.Interface())
	}
}

func assertSVGQuery(t *testing.T, document, expression, name, text string, start, end, contentStart, contentEnd int) xpath.Result {
	t.Helper()
	expression = documentBodyExpression(expression)
	results, err := xpath.Query(expression, document)
	if err != nil {
		t.Fatalf("Query %s returned %v", expression, err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected one result for %s, got %#v", expression, results)
	}
	result := results[0]
	if result.NodeName != name || result.TextContent != text || result.StartLocation != start || result.EndLocation != end || result.ContentStart != contentStart || result.ContentEnd != contentEnd {
		t.Fatalf("Expected %s <%s> text %q at %d:%d content %d:%d, got %#v", expression, name, text, start, end, contentStart, contentEnd, result)
	}
	return result
}

func TestQueryClosedSVGIslandNamespaceAndHTMLExit(t *testing.T) {
	const content = `<div>a<svg id=s><circle id=c>t</circle></svg>b</div>`
	svg := assertSVGQuery(t, content, `//*[@id='s']`, "svg", "t", 6, 45, 16, 39)
	circle := assertSVGQuery(t, content, `//*[@id='c']`, "circle", "t", 16, 39, 29, 30)
	div := assertSVGQuery(t, content, `/div`, "div", "atb", 0, 52, 5, 46)
	assertTableQuery(t, content, `/div/svg/following-sibling::text()`, "b", 45, 46)
	requireResultNamespace(t, svg, svgResultNamespaceURI)
	requireResultNamespace(t, circle, svgResultNamespaceURI)
	requireResultNamespace(t, div, "http://www.w3.org/1999/xhtml")
}

func TestQueryForeignSelfClosingAndAdjustedNames(t *testing.T) {
	const selfClosing = `<svg><g id=a /><path id=p /></svg><p>z</p>`
	assertSVGQuery(t, selfClosing, `//*[@id='a']`, "g", "", 5, 15, 15, 15)
	assertSVGQuery(t, selfClosing, `//*[@id='p']`, "path", "", 15, 28, 28, 28)
	assertSVGQuery(t, selfClosing, `/svg`, "svg", "", 0, 34, 5, 28)
	assertTableQuery(t, selfClosing, `/svg/following-sibling::p`, "z", 34, 42)

	const names = `<svg><lineargradient id=g><clippath id=c><feblend id=f /></clippath></lineargradient></svg>`
	assertSVGQuery(t, names, `//*[@id='g']`, "linearGradient", "", 5, 85, 26, 68)
	assertSVGQuery(t, names, `//*[@id='c']`, "clipPath", "", 26, 68, 41, 57)
	assertSVGQuery(t, names, `//*[@id='f']`, "feBlend", "", 41, 57, 57, 57)
}

func TestQuerySVGAdjustedAttributes(t *testing.T) {
	const content = `<svg id=s viewbox="0 0 1 1" preserveaspectratio=x><lineargradient id=g gradientunits=u attributename=fill /></svg>`
	svg := assertSVGQuery(t, content, `//*[@id='s']`, "svg", "", 0, 114, 50, 108)
	linear := assertSVGQuery(t, content, `//*[@id='g']`, "linearGradient", "", 50, 108, 108, 108)
	if svg.Attributes["viewBox"] != "0 0 1 1" || svg.Attributes["preserveAspectRatio"] != "x" || linear.Attributes["gradientUnits"] != "u" || linear.Attributes["attributeName"] != "fill" {
		t.Fatalf("Adjusted SVG result attributes mismatch: svg=%#v linear=%#v", svg.Attributes, linear.Attributes)
	}
	results, err := xpath.Query(`//*[@id='s']/@* | //*[@id='g']/@*`, content)
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]string, len(results))
	for _, result := range results {
		got[result.NodeName] = result.Value
	}
	for name, value := range map[string]string{"id": "g", "viewBox": "0 0 1 1", "preserveAspectRatio": "x", "gradientUnits": "u", "attributeName": "fill"} {
		if got[name] != value {
			t.Fatalf("Expected adjusted attribute %s=%q in %#v", name, value, results)
		}
	}
}

func TestQuerySVGCaseCloseAndAncestorPop(t *testing.T) {
	const caseClose = `<svg><linearGradient id=g>x</LINEARGRADIENT>z</svg>q`
	assertSVGQuery(t, caseClose, `//*[@id='g']`, "linearGradient", "x", 5, 44, 26, 27)
	assertSVGQuery(t, caseClose, `/svg`, "svg", "xz", 0, 51, 5, 45)
	assertTableQuery(t, caseClose, `/svg/text()[last()]`, "z", 44, 45)
	assertTableQuery(t, caseClose, `/svg/following-sibling::text()`, "q", 51, 52)

	const ancestor = `<svg><g><path id=p></g><circle id=c></svg>z`
	assertSVGQuery(t, ancestor, `//*[@id='p']`, "path", "", 8, 19, 19, 19)
	assertSVGQuery(t, ancestor, `//*[@id='c']`, "circle", "", 23, 36, 36, 36)
	assertSVGQuery(t, ancestor, `/svg/g`, "g", "", 5, 23, 8, 19)
	assertTableQuery(t, ancestor, `/svg/following-sibling::text()`, "z", 42, 43)
}

func TestQuerySVGCDATAReferencesNULDoctypeAndOrder(t *testing.T) {
	const tokens = "<svg><![CDATA[a<b>&copy;]]><text>c&amp;d\x00e</text></svg>z"
	assertSVGQuery(t, tokens, `/svg`, "svg", "a<b>&copy;c&d�e", 0, 55, 5, 49)
	assertSVGQuery(t, tokens, `/svg/text`, "text", "c&d�e", 27, 49, 33, 42)
	assertTableQuery(t, tokens, `/svg/text()[1]`, "a<b>&copy;", 5, 27)
	assertTableQuery(t, tokens, `/svg/text/text()`, "c&d�e", 33, 42)

	const doctype = `<svg>a<!DOCTYPE x>b<circle /></svg>z`
	assertSVGQuery(t, doctype, `/svg`, "svg", "ab", 0, 35, 5, 29)
	assertTableQuery(t, doctype, `/svg/text()`, "ab", 5, 19)
	assertSVGQuery(t, doctype, `/svg/circle`, "circle", "", 19, 29, 29, 29)

	const ordered = `<svg><!--c--><text>x</text><![CDATA[y]]></svg>`
	assertTableQuery(t, ordered, `/svg/node()[1]`, "c", 5, 13)
	assertSVGQuery(t, ordered, `/svg/text`, "text", "x", 13, 27, 19, 20)
	assertTableQuery(t, ordered, `/svg/node()[3]`, "y", 27, 40)
}

func TestQuerySVGEOFAndIncompleteTokens(t *testing.T) {
	const unicode = "<div><svg><g>é\r\n😀"
	assertSVGQuery(t, unicode, `/div/svg`, "svg", "é\n😀", 5, 21, 10, 21)
	assertSVGQuery(t, unicode, `/div/svg/g`, "g", "é\n😀", 10, 21, 13, 21)
	assertTableQuery(t, unicode, `/div/svg/g/text()`, "é\n😀", 13, 21)

	const incompleteEnd = `<svg><g>x</g`
	assertSVGQuery(t, incompleteEnd, `/svg/g`, "g", "x", 5, 12, 8, 12)
	assertTableQuery(t, incompleteEnd, `/svg/g/text()`, "x", 8, 12)

	const incompleteCDATA = `<svg><![CDATA[x]]`
	assertSVGQuery(t, incompleteCDATA, `/svg`, "svg", "x]]", 0, 17, 5, 17)
	assertTableQuery(t, incompleteCDATA, `/svg/text()`, "x]]", 5, 17)
}

func TestQuerySVGFormattingExitAndRootSelfClose(t *testing.T) {
	const formatting = `<b><svg><circle /></svg>x</b>y`
	assertSVGQuery(t, formatting, `/b/svg`, "svg", "", 3, 24, 8, 18)
	assertSVGQuery(t, formatting, `/b/svg/circle`, "circle", "", 8, 18, 18, 18)
	assertTableQuery(t, formatting, `/b/svg/following-sibling::text()`, "x", 24, 25)
	assertTableQuery(t, formatting, `/b/following-sibling::text()`, "y", 29, 30)

	const root = `<svg id=s />z`
	assertSVGQuery(t, root, `//*[@id='s']`, "svg", "", 0, 12, 12, 12)
	assertTableQuery(t, root, `/svg/following-sibling::text()`, "z", 12, 13)
}

func TestQuerySVGForeignTokenizerProgressBogusCommentAndNames(t *testing.T) {
	const emptyEnd = `<svg>a</>b</svg>z`
	assertSVGQuery(t, emptyEnd, `/svg`, "svg", "ab", 0, 16, 5, 10)
	assertTableQuery(t, emptyEnd, `/svg/text()`, "ab", 5, 10)

	const bogus = `<svg>a<!foo>b</svg>z`
	assertTableQuery(t, bogus, `/svg/node()[2]`, "foo", 6, 12)
	assertTableQuery(t, bogus, `/svg/text()[2]`, "b", 12, 13)

	const adjusted = `<svg><fedropshadow id=f /></svg>`
	assertSVGQuery(t, adjusted, `//*[@id='f']`, "feDropShadow", "", 5, 26, 26, 26)

	const unicode = `<svg><aÀ id=x>y</AÀ></svg>`
	assertSVGQuery(t, unicode, `//*[@id='x']`, "aÀ", "y", 5, 22, 15, 16)
}

func TestQuerySVGMalformedEndTagsAdvance(t *testing.T) {
	assertTableQuery(t, `<svg>a</>b</svg>z`, `/svg/text()`, "ab", 5, 10)
	assertTableQuery(t, `<svg>a</!x>b</svg>z`, `/svg/node()[2]`, "!x", 6, 11)
	assertTableQuery(t, `<svg>a<!foo>b</svg>z`, `/svg/node()[2]`, "foo", 6, 12)
}

func TestQuerySyntheticHTMLNamespace(t *testing.T) {
	results, err := xpath.Query(`//tbody`, `<table><tr><td>x</table>`)
	if err != nil || len(results) != 1 {
		t.Fatalf("Expected synthetic tbody: results=%#v err=%v", results, err)
	}
	requireResultNamespace(t, results[0], "http://www.w3.org/1999/xhtml")
}

func TestQuerySVGUnknownNamesUseASCIIFoldingOnly(t *testing.T) {
	assertSVGQuery(t, `<svg><AÀ id=x>y</aÀ></svg>`, `//*[@id='x']`, "aÀ", "y", 5, 22, 15, 16)
	results, err := xpath.Query(`//*[@id='x']`, `<svg><aÀ id=x>y</aà>z</svg>`)
	if err != nil || len(results) != 1 || results[0].NodeName != "aÀ" || results[0].TextContent != "yz" {
		t.Fatalf("Foreign close must not Unicode-fold: results=%#v err=%v", results, err)
	}
}

func TestQueryAttributeUnionUsesOwnerAttributeOrder(t *testing.T) {
	results, err := xpath.Query(`//@b | //@a`, `<div a=1 b=2></div>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].NodeName != "a" || results[1].NodeName != "b" {
		t.Fatalf("Attribute union must use DOM attribute order: %#v", results)
	}
	results, err = xpath.Query(`//@a | //div`, `<div a=1><span></span></div>`)
	if err != nil || len(results) != 2 || results[0].NodeName != "div" || results[1].NodeName != "a" {
		t.Fatalf("Owner element must precede its attribute: results=%#v err=%v", results, err)
	}
	results, err = xpath.Query(`//span | //@a`, `<div a=1><span></span></div>`)
	if err != nil || len(results) != 2 || results[0].NodeName != "a" || results[1].NodeName != "span" {
		t.Fatalf("Owner attribute must precede descendants: results=%#v err=%v", results, err)
	}
	results, err = xpath.Query(`//@a | //@a`, `<div a=1></div>`)
	if err != nil || len(results) != 1 || results[0].NodeName != "a" {
		t.Fatalf("Union must deduplicate the same attribute identity: results=%#v err=%v", results, err)
	}
}
