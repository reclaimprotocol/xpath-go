package xpath_test

import (
	"fmt"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func assertFormattingNoQuery(t *testing.T, document, expression string) {
	t.Helper()
	expression = documentBodyExpression(expression)
	results, err := xpath.Query(expression, document)
	if err != nil {
		t.Fatalf("Query %s returned %v", expression, err)
	}
	if len(results) != 0 {
		t.Fatalf("Expected %s not to match, got %#v", expression, results)
	}
}

func TestQueryFormattingReconstructsAcrossBlockParagraphAndList(t *testing.T) {
	const block = `<p><b>one<div>two</div>three`
	assertTableQuery(t, block, `//p/b`, "one", 3, 9)
	assertTableQuery(t, block, `//div/b`, "two", 3, 17)
	assertTableQuery(t, block, `//div/following-sibling::b`, "three", 3, 28)

	const paragraphEnd = `<p><em>a</p>b`
	assertTableQuery(t, paragraphEnd, `//p/em`, "a", 3, 8)
	assertTableQuery(t, paragraphEnd, `//p/following-sibling::em`, "b", 3, 13)

	const list = `<ul><li><b>a<li>b</ul>`
	assertTableQuery(t, list, `//ul/li[1]/b`, "a", 8, 12)
	assertTableQuery(t, list, `//ul/li[2]/b`, "b", 8, 17)
}

func TestQueryFormattingDefinitionItemAndCloneCloseIntegration(t *testing.T) {
	const definition = `<dl><dt><i>x<dd>y</dl>`
	assertTableQuery(t, definition, `//dl/dt/i`, "x", 8, 12)
	assertTableQuery(t, definition, `//dl/dd/i`, "y", 8, 17)

	const cloneClose = `<p><b>x<div>y</b>z`
	assertTableQuery(t, cloneClose, `//p/b`, "x", 3, 7)
	assertTableQuery(t, cloneClose, `//div/b`, "y", 3, 17)
	assertTableQuery(t, cloneClose, `//div/b/following-sibling::text()`, "z", 17, 18)
}

func TestQueryFormattingMarkersAndNoahsArk(t *testing.T) {
	const markers = `<button><b>x</button>y<object><i>z</object>w`
	assertTableQuery(t, markers, `//button/b`, "x", 8, 12)
	assertTableQuery(t, markers, `//button/following-sibling::b`, "yzw", 8, 44)
	assertTableQuery(t, markers, `//object/i`, "z", 30, 34)

	const noah = `<p><b class=x><b class=x><b class=x><b class=x>a</p>b`
	assertTableQuery(t, noah, `//p/following-sibling::b`, "b", 14, 53)
	assertTableQuery(t, noah, `//p/following-sibling::b/b`, "b", 25, 53)
	assertTableQuery(t, noah, `//p/following-sibling::b/b/b`, "b", 36, 53)
}

func TestQuerySafeAnchorMarkersAndNoahAttributeFamilies(t *testing.T) {
	const anchor = `<p><a href=x>a</p>b`
	assertTableQuery(t, anchor, `//p/a`, "a", 3, 14)
	assertTableQuery(t, anchor, `//p/following-sibling::a`, "b", 3, 19)

	const cell = `<table><tr><td><b>x<td>y</table>`
	assertTableQuery(t, cell, `//table/tbody/tr/td[1]/b`, "x", 15, 19)
	assertTableQuery(t, cell, `//table/tbody/tr/td[2]`, "y", 19, 24)

	const caption = `<table><caption><b>x<tr><td>y</table>`
	assertTableQuery(t, caption, `//table/caption/b`, "x", 16, 20)
	assertTableQuery(t, caption, `//table/tbody/tr/td`, "y", 24, 29)

	const reordered = `<p><b a=1 b=2><b b=2 a=1><b a=1 b=2><b b=2 a=1>x</p>y`
	assertTableQuery(t, reordered, `//p/following-sibling::b/b/b`, "y", 36, 53)

	const distinct = `<p><b x=1><b x=2><b y=1><b y=2>x</p>y`
	assertTableQuery(t, distinct, `//p/following-sibling::b/b/b/b`, "y", 24, 37)

	const marker = `<object><b><b><b><b>x</object>y`
	assertTableQuery(t, marker, `//object/b/b/b/b`, "x", 17, 21)
	assertTableQuery(t, marker, `//object/following-sibling::text()`, "y", 30, 31)
}

func TestQueryFormattingTriggerAndCurrentSelectMatrix(t *testing.T) {
	const whitespace = `<p><b>x</p> y`
	assertTableQuery(t, whitespace, `//p/following-sibling::b`, " y", 3, 13)

	const comment = `<p><b>x</p><!--c-->y`
	assertTableQuery(t, comment, `//p/following-sibling::b`, "y", 3, 20)

	const block = `<p><b>x</p><div>y</div>`
	assertTableQuery(t, block, `//div/b`, "y", 3, 17)

	const plaintext = `<p><b>x</p><plaintext>y`
	assertTableQuery(t, plaintext, `//plaintext/b`, "y", 3, 23)

	const currentSelect = `<p><b>x</p><select><span>y</span></select>z`
	assertTableQuery(t, currentSelect, `//p/following-sibling::b`, "yz", 3, 43)
	assertTableQuery(t, currentSelect, `//p/following-sibling::b/select/span`, "y", 19, 33)

	for _, testCase := range []struct {
		name, content string
		end           int
	}{
		{name: "span", content: `<p><span>x</p>y`, end: 10},
		{name: "blink", content: `<p><blink>x</p>y`, end: 11},
	} {
		assertTableQuery(t, testCase.content, `//p/`+testCase.name, "x", 3, testCase.end)
		assertTableQuery(t, testCase.content, `//p/following-sibling::text()`, "y", len(testCase.content)-1, len(testCase.content))
	}
}

func TestQueryFormattingCanonicalTableAndCaptionMarkers(t *testing.T) {
	const canonical = `<table><b><tr><td>aaa</td></tr>bbb</table>ccc`
	assertTableQuery(t, canonical, `/b[1]`, "", 7, 10)
	assertTableQuery(t, canonical, `/b[2]`, "bbb", 7, 34)
	assertTableQuery(t, canonical, `/table`, "aaa", 0, 42)
	assertTableQuery(t, canonical, `/table/tbody/tr/td`, "aaa", 14, 26)
	assertTableQuery(t, canonical, `/b[3]`, "ccc", 7, 45)

	const caption = `<table><caption><b>x</caption>y</table>z`
	assertTableQuery(t, caption, `/table/preceding-sibling::text()`, "y", 30, 31)
	assertTableQuery(t, caption, `/table/caption/b`, "x", 16, 20)
	assertTableQuery(t, caption, `/table/following-sibling::text()`, "z", 39, 40)
}

func TestQueryFormattingNoahsArkEvictionUsesNodeIdentity(t *testing.T) {
	const content = `<b><b><b><b>x</b></b></b></b>y`
	assertTableQuery(t, content, `/b`, "x", 0, 29)
	assertTableQuery(t, content, `/b/b/b/b`, "x", 9, 17)
	assertTableQuery(t, content, `/b/following-sibling::text()`, "y", 29, 30)
}

func TestQueryFormattingMarkerPreservationAndCurrentOpenInsertion(t *testing.T) {
	const noah = `<p><b><b><b>x<object><b>y</object></p>z`
	assertTableQuery(t, noah, `//p/b/b/b/object/b`, "y", 21, 25)
	assertTableQuery(t, noah, `//p/following-sibling::b`, "z", 3, 39)
	assertTableQuery(t, noah, `//p/following-sibling::b/b/b`, "z", 9, 39)

	const cell = `<b><table><tr><td><i>x<td>y</table>z`
	assertTableQuery(t, cell, `/b`, "xyz", 0, 36)
	assertTableQuery(t, cell, `/b/table/tbody/tr/td[1]/i`, "x", 18, 22)
	assertFormattingNoQuery(t, cell, `/b/table/tbody/tr/td[2]/i`)

	const caption = `<b><table><caption><i>x<tr><td>y</table>z`
	assertTableQuery(t, caption, `/b`, "xyz", 0, 41)
	assertTableQuery(t, caption, `/b/table/caption/i`, "x", 19, 23)
	assertFormattingNoQuery(t, caption, `/b/table/tbody/tr/td/i`)

	const current = `<p><b>x</p>y<div>z</div>w`
	assertTableQuery(t, current, `//p/following-sibling::b`, "yzw", 3, 25)
	assertTableQuery(t, current, `//p/following-sibling::b/div`, "z", 12, 24)

	const currentMatrix = `<p><b>x</p>y<!--c--><div>z</div><style>s</style>w`
	assertTableQuery(t, currentMatrix, `//p/following-sibling::b`, "yzsw", 3, 49)
	assertTableQuery(t, currentMatrix, `//p/following-sibling::b/div`, "z", 20, 32)
	assertTableQuery(t, currentMatrix, `//p/following-sibling::b/style`, "s", 32, 48)

	const button = `<p><b>x</p><button>y</button>z`
	assertTableQuery(t, button, `//p/following-sibling::b`, "yz", 3, 30)
	assertTableQuery(t, button, `//p/following-sibling::b/button`, "y", 11, 29)

	const br = `<p><b>x</p></br>y`
	assertTableQuery(t, br, `//p/following-sibling::b`, "y", 3, 17)
	assertTableQuery(t, br, `//p/following-sibling::b/br`, "", 11, 16)
}

func TestQueryFormattingDiscardedIncompleteStartsAndDocumentEndReentry(t *testing.T) {
	for _, content := range []string{`<p><b>x</p><span`, `<p><b>x</p><br`} {
		assertTableQuery(t, content, `//p/b`, "x", 3, 7)
		assertFormattingNoQuery(t, content, `//p/following-sibling::b`)
	}

	for _, testCase := range []struct {
		content string
		bEnd    int
	}{
		{content: `<html><body><p><b>x</p></body>y</html>`, bEnd: 31},
		{content: `<html><body><p><b>x</p></body></html>y`, bEnd: 38},
	} {
		assertTableQuery(t, testCase.content, `//body/p/b`, "x", 15, 19)
		assertTableQuery(t, testCase.content, `//body/b`, "y", 15, testCase.bEnd)
	}
}

func TestQueryFormattingDocumentEndCommentPlacement(t *testing.T) {
	const beforeHTML = `<html><body><p><b>x</p></body><!--c--></html>`
	assertTableQuery(t, beforeHTML, `//body`, "x", 6, 30)
	assertFormattingNoQuery(t, beforeHTML, `//body/b`)
	assertTableQuery(t, beforeHTML, `//html/comment()[.='c']`, "c", 30, 38)

	const afterHTML = `<html><body><p><b>x</p></body></html><!--c-->`
	assertTableQuery(t, afterHTML, `//body`, "x", 6, 30)
	assertFormattingNoQuery(t, afterHTML, `//body/b`)
	assertTableQuery(t, afterHTML, `//html/following-sibling::node()`, "c", 37, 45)
}

func TestQueryFormattingReconstructsOffStackEntryInsideOpenAncestor(t *testing.T) {
	const ordinary = `<b><p><i>x</p>y</b>`
	assertTableQuery(t, ordinary, `/b`, "xy", 0, 19)
	assertTableQuery(t, ordinary, `/b/p/i`, "x", 6, 10)
	assertTableQuery(t, ordinary, `/b/p/following-sibling::i`, "y", 6, 15)

	const cell = `<table><tr><td><b><p><i>x</p>y</table>`
	assertTableQuery(t, cell, `//table/tbody/tr/td/b`, "xy", 15, 30)
	assertTableQuery(t, cell, `//table/tbody/tr/td/b/p/i`, "x", 21, 25)
	assertTableQuery(t, cell, `//table/tbody/tr/td/b/p/following-sibling::i`, "y", 21, 30)
}

func TestQueryFormattingOffStackCommentsDoNotReconstruct(t *testing.T) {
	const content = `<p><b id=0><b id=1><b id=2><b id=3>x<div><!--c--><!--c--><!--c--><!--c--></div>`
	assertTableQuery(t, content, `//p`, "x", 0, 36)
	assertTableQuery(t, content, `//p/b/b/b/b`, "x", 27, 36)
	assertTableQuery(t, content, `//div`, "", 36, 79)
	assertFormattingNoQuery(t, content, `//div/b`)
	for i := 1; i <= 4; i++ {
		assertTableQuery(t, content, `//div/node()[`+fmt.Sprint(i)+`]`, "c", 41+8*(i-1), 49+8*(i-1))
	}
}

func TestQueryFormattingReconstructedCloneTextAndComments(t *testing.T) {
	const content = `<p><b id=0><b id=1><b id=2><b id=3>x<div>y<!--c-->y<!--c-->y<!--c-->y<!--c--></div>`
	assertTableQuery(t, content, `//p`, "x", 0, 36)
	assertTableQuery(t, content, `//div`, "yyyy", 36, 83)
	assertTableQuery(t, content, `//div/b`, "yyyy", 3, 77)
	assertTableQuery(t, content, `//div/b/b/b/b`, "yyyy", 27, 77)
	for i := 1; i <= 4; i++ {
		assertTableQuery(t, content, `//div/b/b/b/b/text()[`+fmt.Sprint(i)+`]`, "y", 41+9*(i-1), 42+9*(i-1))
		assertTableQuery(t, content, `//div/b/b/b/b/node()[`+fmt.Sprint(2*i)+`]`, "c", 42+9*(i-1), 50+9*(i-1))
	}
}

func TestQueryAllFormattingElementsHaveSimpleEndBehavior(t *testing.T) {
	for _, name := range []string{"a", "b", "big", "code", "em", "font", "i", "nobr", "s", "small", "strike", "strong", "tt", "u"} {
		t.Run(name, func(t *testing.T) {
			open, close := "<"+name+">", "</"+name+">"
			content := open + "x" + close + "y"
			assertTableQuery(t, content, `//`+name, "x", 0, len(open)+1+len(close))
			assertTableQuery(t, content, `//`+name+`/following-sibling::text()`, "y", len(content)-1, len(content))
		})
	}
}

func TestQueryFormattingEOF(t *testing.T) {
	const eof = `<section><strong>x`
	assertTableQuery(t, eof, `//section`, "x", 0, len(eof))
	assertTableQuery(t, eof, `//section/strong`, "x", 9, len(eof))
}

func TestQueryFormattingMultibyteLocations(t *testing.T) {
	const multiline = "<p><strong>é\r\n<span>x<div>😀</div>z"
	assertTableQuery(t, multiline, `//p/strong`, "é\nx", 3, 22)
	assertTableQuery(t, multiline, `//div/strong`, "😀", 3, 31)
	assertTableQuery(t, multiline, `//div/following-sibling::strong`, "z", 3, 38)
}
