package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQuerySVGHTMLBreakoutAndPermanentReprocessing(t *testing.T) {
	const paragraph = `<div>a<svg><g><p>x</p></g><circle /></svg>z</div>`
	assertSVGQuery(t, paragraph, `/div/svg`, "svg", "", 6, 14, 11, 14)
	assertSVGQuery(t, paragraph, `/div/p`, "p", "x", 14, 22, 17, 18)
	circle := assertSVGQuery(t, paragraph, `/div/circle`, "circle", "z", 26, 43, 34, 43)
	requireResultNamespace(t, circle, "http://www.w3.org/1999/xhtml")

	const permanent = `<svg><g>a<div>b</div>c</g><circle></circle></svg>d`
	assertSVGQuery(t, permanent, `/svg`, "svg", "a", 0, 9, 5, 9)
	assertSVGQuery(t, permanent, `/div`, "div", "b", 9, 21, 14, 15)
	assertSVGQuery(t, permanent, `/circle`, "circle", "", 26, 43, 34, 34)
	assertTableQuery(t, permanent, `/div/following-sibling::text()[1]`, "c", 21, 22)
}

func TestQuerySVGFontAndSpanBreakoutContrasts(t *testing.T) {
	const plain = `<svg><g><font>x</font><circle /></g></svg>z`
	font := assertSVGQuery(t, plain, `/svg/g/font`, "font", "x", 8, 22, 14, 15)
	requireResultNamespace(t, font, svgResultNamespaceURI)

	const attributed = `<svg><g><font color=red>x</font><circle /></g></svg>z`
	font = assertSVGQuery(t, attributed, `/font`, "font", "x", 8, 32, 24, 25)
	requireResultNamespace(t, font, "http://www.w3.org/1999/xhtml")
	assertSVGQuery(t, attributed, `/circle`, "circle", "z", 32, 53, 40, 53)

	const anchor = `<svg><g><a id=a>x</a><circle /></g></svg>z`
	a := assertSVGQuery(t, anchor, `//*[@id='a']`, "a", "x", 8, 21, 16, 17)
	requireResultNamespace(t, a, svgResultNamespaceURI)

	const span = `<svg><g><span id=s>x</span><a id=a>y</a></g></svg>z`
	assertSVGQuery(t, span, `//*[@id='s']`, "span", "x", 8, 27, 19, 20)
	a = assertSVGQuery(t, span, `//*[@id='a']`, "a", "y", 27, 40, 35, 36)
	requireResultNamespace(t, a, "http://www.w3.org/1999/xhtml")
}

func TestQuerySVGHTMLIntegrationPointNamespaces(t *testing.T) {
	const foreignObject = `<svg><foreignObject><div id=d><b>x</b></div></foreignObject><circle /></svg>z`
	fo := assertSVGQuery(t, foreignObject, `/svg/foreignObject`, "foreignObject", "x", 5, 60, 20, 44)
	div := assertSVGQuery(t, foreignObject, `//*[@id='d']`, "div", "x", 20, 44, 30, 38)
	circle := assertSVGQuery(t, foreignObject, `/svg/circle`, "circle", "", 60, 70, 70, 70)
	requireResultNamespace(t, fo, svgResultNamespaceURI)
	requireResultNamespace(t, div, "http://www.w3.org/1999/xhtml")
	requireResultNamespace(t, circle, svgResultNamespaceURI)

	const desc = `<svg><desc><p id=p>x</p></desc><circle /></svg>z`
	assertSVGQuery(t, desc, `//*[@id='p']`, "p", "x", 11, 24, 19, 20)

	const title = `<svg><title><span id=s>x</span></title><circle /></svg>z`
	assertSVGQuery(t, title, `//*[@id='s']`, "span", "x", 12, 31, 23, 24)
}

func TestQuerySVGMixedCaseForeignObjectAdjustmentAndIntegration(t *testing.T) {
	const content = `<svg><FOREIGNOBJECT><DIV>x</DIV></FOREIGNOBJECT><rect /></svg>`
	foreignObject := assertSVGQuery(t, content, `/svg/foreignObject`, "foreignObject", "x", 5, 48, 20, 32)
	div := assertSVGQuery(t, content, `/svg/foreignObject/div`, "div", "x", 20, 32, 25, 26)
	requireResultNamespace(t, foreignObject, svgResultNamespaceURI)
	requireResultNamespace(t, div, "http://www.w3.org/1999/xhtml")
}

func TestQuerySVGIntegrationEndAndNestedBreakoutBoundaries(t *testing.T) {
	const endP = `<svg id=s><foreignObject id=f>x</p>y</foreignObject><rect id=r></rect></svg>z`
	assertSVGQuery(t, endP, `//*[@id='f']`, "foreignObject", "xy", 10, 52, 30, 36)
	assertSVGQuery(t, endP, `//*[@id='f']/p`, "p", "", 0, 0, 0, 0)
	assertTableQuery(t, endP, `//*[@id='f']/p/following-sibling::text()`, "y", 35, 36)

	const endBR = `<svg id=s><desc id=f>x</br>y</desc><rect id=r></rect></svg>z`
	assertSVGQuery(t, endBR, `//*[@id='f']/br`, "br", "", 0, 0, 0, 0)
	assertTableQuery(t, endBR, `//*[@id='f']/br/following-sibling::text()`, "y", 27, 28)

	const nested = `<svg id=o><foreignObject id=f><svg id=i><g id=g>a<div id=h>b</div>c</foreignObject><rect id=r></rect></svg>q`
	assertSVGQuery(t, nested, `//*[@id='i']`, "svg", "a", 30, 49, 40, 49)
	assertSVGQuery(t, nested, `//*[@id='g']`, "g", "a", 40, 49, 48, 49)
	div := assertSVGQuery(t, nested, `//*[@id='h']`, "div", "b", 49, 66, 59, 60)
	requireResultNamespace(t, div, "http://www.w3.org/1999/xhtml")
	assertTableQuery(t, nested, `//*[@id='h']/following-sibling::text()`, "c", 66, 67)
}

func TestQuerySVGIntegrationReferenceDecoding(t *testing.T) {
	const content = `<svg><foreignObject id=f>a&amp;b&#32;c</foreignObject></svg>`
	assertSVGQuery(t, content, `//*[@id='f']`, "foreignObject", "a&b c", 5, 54, 25, 38)
	assertTableQuery(t, content, `//*[@id='f']/text()`, "a&b c", 25, 38)
}

func TestQuerySVGIntegrationNestedSVGEOFAndMalformedTokens(t *testing.T) {
	const nested = `<svg><foreignObject><svg id=i><circle /></svg><div>x</div></foreignObject></svg>z`
	inner := assertSVGQuery(t, nested, `//*[@id='i']`, "svg", "", 20, 46, 30, 40)
	assertSVGQuery(t, nested, `//*[@id='i']/circle`, "circle", "", 30, 40, 40, 40)
	assertSVGQuery(t, nested, `/svg/foreignObject/div`, "div", "x", 46, 58, 51, 52)
	requireResultNamespace(t, inner, svgResultNamespaceURI)

	const unicode = "<svg><foreignObject><div>é\r\n<b>😀"
	assertSVGQuery(t, unicode, `/svg/foreignObject/div`, "div", "é\n😀", 20, 36, 25, 36)
	assertSVGQuery(t, unicode, `/svg/foreignObject/div/b`, "b", "😀", 29, 36, 32, 36)

	const incompleteStart = `<svg><foreignObject><div x`
	assertSVGQuery(t, incompleteStart, `/svg/foreignObject`, "foreignObject", "", 5, 26, 20, 26)
	assertFormattingNoQuery(t, incompleteStart, `/svg/foreignObject/div`)

	const incompleteEnd = `<svg><foreignObject><div>x</foreignObject`
	assertSVGQuery(t, incompleteEnd, `/svg/foreignObject/div`, "div", "x", 20, 41, 25, 41)
}

func TestQuerySVGIncompleteBreakoutTokensDoNotMutateForeignState(t *testing.T) {
	for _, content := range []string{
		`<div><svg id=s><g id=g>a<font color=red`,
		`<div><svg id=s><g id=g>a</p`,
	} {
		g := assertSVGQuery(t, content, `//*[@id='g']`, "g", "a", 15, len(content), 23, len(content))
		requireResultNamespace(t, g, svgResultNamespaceURI)
		assertTableQuery(t, content, `//*[@id='g']/text()`, "a", 23, len(content))
	}
}

func TestQuerySVGIntegrationBarrierEndBreakoutFormattingAndComments(t *testing.T) {
	const barrier = `<svg><foreignObject><p>x</foreignObject>y</svg>z`
	assertSVGQuery(t, barrier, `/svg/foreignObject/p`, "p", "xyz", 20, 48, 23, 48)

	const formatting = `<b><svg><g><p>x</p></g></svg>y</b>z`
	assertSVGQuery(t, formatting, `/b/svg`, "svg", "", 3, 11, 8, 11)
	assertSVGQuery(t, formatting, `/b/p`, "p", "x", 11, 19, 14, 15)
	assertTableQuery(t, formatting, `/b/p/following-sibling::text()`, "y", 29, 30)

	const comments = `<svg><desc><!--c--><span>x</span>y</desc>z</svg>q`
	assertTableQuery(t, comments, `/svg/desc/node()[1]`, "c", 11, 19)
	assertSVGQuery(t, comments, `/svg/desc/span`, "span", "x", 19, 33, 25, 26)
	assertTableQuery(t, comments, `/svg/desc/span/following-sibling::text()`, "y", 33, 34)

	const endP = `<svg><g>x</p>y</g></svg>z`
	assertSVGQuery(t, endP, `/svg`, "svg", "x", 0, 9, 5, 9)
	assertSVGQuery(t, endP, `/p`, "p", "", 0, 0, 0, 0)
	assertTableQuery(t, endP, `/p/following-sibling::text()`, "yz", 13, 25)

	const endBR = `<svg><g>x</br>y</g></svg>z`
	assertSVGQuery(t, endBR, `/br`, "br", "", 0, 0, 0, 0)
	assertTableQuery(t, endBR, `/br/following-sibling::text()`, "yz", 14, 26)
}

func TestQuerySVGForeignEndDoesNotCrossHTMLIntegrationBarrier(t *testing.T) {
	const content = `<svg id=o><foreignObject id=f><div id=h><svg id=i><g id=g>x</foreignObject><circle id=c></circle></svg>z</div></foreignObject><rect id=r></rect></svg>q`
	assertSVGQuery(t, content, `//*[@id='o']`, "svg", "xz", 0, 150, 10, 144)
	assertSVGQuery(t, content, `//*[@id='f']`, "foreignObject", "xz", 10, 126, 30, 110)
	div := assertSVGQuery(t, content, `//*[@id='h']`, "div", "xz", 30, 110, 40, 104)
	assertSVGQuery(t, content, `//*[@id='i']`, "svg", "x", 40, 103, 50, 97)
	assertSVGQuery(t, content, `//*[@id='g']`, "g", "x", 50, 97, 58, 97)
	assertSVGQuery(t, content, `//*[@id='c']`, "circle", "", 75, 97, 88, 88)
	assertSVGQuery(t, content, `//*[@id='r']`, "rect", "", 126, 144, 137, 137)
	requireResultNamespace(t, div, "http://www.w3.org/1999/xhtml")
	assertTableQuery(t, content, `//*[@id='i']/following-sibling::text()`, "z", 103, 104)
}

func TestQuerySVGForeignOwnerEndIgnoredAcrossLiveHTMLChild(t *testing.T) {
	tests := []struct {
		name, element, content string
		length, contentStart   int
		textStart, rectStart   int
	}{
		{"span", "span", `<svg id=o><foreignObject id=f><span id=h>x</foreignObject><rect id=r></rect></svg>z`, 83, 41, 41, 58},
		{"b", "b", `<svg id=o><foreignObject id=f><b id=h>x</foreignObject><rect id=r></rect></svg>z`, 80, 38, 38, 55},
		{"i", "i", `<svg id=o><foreignObject id=f><i id=h>x</foreignObject><rect id=r></rect></svg>z`, 80, 38, 38, 55},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			html := assertSVGQuery(t, test.content, `//*[@id='h']`, test.element, "xz", 30, test.length, test.contentStart, test.length)
			rect := assertSVGQuery(t, test.content, `//*[@id='r']`, "rect", "", test.rectStart, test.rectStart+18, test.rectStart+11, test.rectStart+11)
			requireResultNamespace(t, html, "http://www.w3.org/1999/xhtml")
			requireResultNamespace(t, rect, "http://www.w3.org/1999/xhtml")
			assertTableQuery(t, test.content, `//*[@id='h']/text()[1]`, "x", test.textStart, test.textStart+1)
			assertTableQuery(t, test.content, `//*[@id='h']/text()[2]`, "z", test.length-1, test.length)
		})
	}
}

func TestQuerySVGBreakoutStrictEOFAndStaleNameIdentity(t *testing.T) {
	const recovered = `<svg><g><div>x</div></g></svg><section>y`
	assertSVGQuery(t, recovered, `/section`, "section", "y", 30, 40, 39, 40)

	const stale = `<svg><g><div>x</div><g id=h><span id=s>y</g>z</span>`
	htmlG := assertSVGQuery(t, stale, `//*[@id='h']`, "g", "y", 20, 44, 28, 40)
	span := assertSVGQuery(t, stale, `//*[@id='s']`, "span", "y", 28, 40, 39, 40)
	requireResultNamespace(t, htmlG, "http://www.w3.org/1999/xhtml")
	requireResultNamespace(t, span, "http://www.w3.org/1999/xhtml")
	assertTableQuery(t, stale, `//*[@id='h']/following-sibling::text()`, "z", 44, 45)
}

func TestQuerySVGBreakoutDocumentElementStartDispatch(t *testing.T) {
	const bodyStart = `<html><head></head><body id=b><div id=o>a<svg><g><body class=x>c</div></body></html>`
	assertSVGQuery(t, bodyStart, `//html`, "html", "ac", 0, 84, 6, 77)
	assertSVGQuery(t, bodyStart, `//body[@id='b' and @class='x']`, "body", "ac", 19, 77, 30, 70)
	assertSVGQuery(t, bodyStart, `//*[@id='o']`, "div", "ac", 30, 70, 40, 64)
	assertTableQuery(t, bodyStart, `//*[@id='o']/svg/following-sibling::text()`, "c", 63, 64)
	results, err := xpath.Query(`//body`, bodyStart)
	if err != nil || len(results) != 1 {
		t.Fatalf("Breakout body start created nested body nodes: results=%#v err=%v", results, err)
	}

	const headStart = `<html><head></head><body id=b><div id=o>a<svg><g><head id=h>c</div></body></html>`
	assertFormattingNoQuery(t, headStart, `//head[@id='h']`)
	results, err = xpath.Query(`//head`, headStart)
	if err != nil || len(results) != 1 {
		t.Fatalf("Breakout head start created a nested head: results=%#v err=%v", results, err)
	}
	assertTableQuery(t, headStart, `//*[@id='o']/svg/following-sibling::text()`, "c", 60, 61)
}

func TestQuerySVGIntegrationDocumentElementStartDispatch(t *testing.T) {
	const content = `<html id=o><head></head><body id=b><svg><foreignObject id=f><html class=x><br><body class=y><br><head id=h><br></foreignObject></svg></body></html>`
	assertSVGQuery(t, content, `//html[@id='o' and @class='x']`, "html", "", 0, 147, 11, 140)
	assertSVGQuery(t, content, `//body[@id='b' and @class='y']`, "body", "", 24, 140, 35, 133)
	assertFormattingNoQuery(t, content, `//head[@id='h']`)
	results, err := xpath.Query(`//html | //body | //head`, content)
	if err != nil || len(results) != 3 {
		t.Fatalf("Integration document starts created nested document elements: results=%#v err=%v", results, err)
	}
	results, err = xpath.Query(`//*[@id='f']/br`, content)
	if err != nil || len(results) != 3 {
		t.Fatalf("Expected three HTML br children under foreignObject, got results=%#v err=%v", results, err)
	}
	for i, expected := range [][2]int{{74, 78}, {92, 96}, {107, 111}} {
		if results[i].NodeName != "br" || results[i].StartLocation != expected[0] || results[i].EndLocation != expected[1] {
			t.Fatalf("Unexpected integration br %d: %#v", i, results[i])
		}
		requireResultNamespace(t, results[i], "http://www.w3.org/1999/xhtml")
	}
}

func TestQuerySVGIgnoredForeignEndSourceRangeContrasts(t *testing.T) {
	const terminal = `<svg>x</foo></svg>`
	assertTableQuery(t, terminal, `/svg/text()`, "x", 5, 6)

	const between = `<svg>x</foo>y</svg>`
	assertTableQuery(t, between, `/svg/text()`, "xy", 5, 13)

	const blocked = `<svg><foreignObject><p>x</foreignObject></p></svg>`
	assertTableQuery(t, blocked, `/svg/foreignObject/p/text()`, "x", 23, 24)

	const staleIncomplete = `<svg><g><div>x</div>y</g`
	assertTableQuery(t, staleIncomplete, `/div/following-sibling::text()`, "y", 20, 24)
}

func TestQuerySVGIntegrationIgnoredHTMLEndFallbacks(t *testing.T) {
	const terminal = `<svg><foreignObject><div>x</foo></div></foreignObject></svg>`
	assertSVGQuery(t, terminal, `/svg/foreignObject/div`, "div", "x", 20, 38, 25, 32)
	assertTableQuery(t, terminal, `/svg/foreignObject/div/text()`, "x", 25, 26)

	const between = `<svg><foreignObject><div>x</foo>y</div></foreignObject></svg>`
	assertSVGQuery(t, between, `/svg/foreignObject/div`, "div", "xy", 20, 39, 25, 33)
	assertTableQuery(t, between, `/svg/foreignObject/div/text()`, "xy", 25, 33)

	const specialAbsent = `<svg><foreignObject><span>x</div>y</span></foreignObject></svg>`
	assertSVGQuery(t, specialAbsent, `/svg/foreignObject/span`, "span", "xy", 20, 41, 26, 34)
	assertTableQuery(t, specialAbsent, `/svg/foreignObject/span/text()`, "xy", 26, 34)

	const barrier = `<div><svg><foreignObject><span>x</div>y</span></foreignObject></svg></div>`
	assertSVGQuery(t, barrier, `/div`, "div", "xy", 0, 74, 5, 68)
	assertSVGQuery(t, barrier, `/div/svg`, "svg", "xy", 5, 68, 10, 62)
	assertSVGQuery(t, barrier, `/div/svg/foreignObject/span`, "span", "xy", 25, 46, 31, 39)
}
