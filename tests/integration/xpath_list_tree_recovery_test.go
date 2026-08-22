package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func assertListQueryNode(t *testing.T, document, expression, wantText string, wantStart, wantEnd int) {
	t.Helper()
	expression = documentBodyExpression(expression)
	results, err := xpath.Query(expression, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != wantText || results[0].StartLocation != wantStart || results[0].EndLocation != wantEnd {
		t.Fatalf("Expected %s text %q at %d:%d, got %#v", expression, wantText, wantStart, wantEnd, results)
	}
}

func TestQueryLIStartExplicitAndAbsentRecovery(t *testing.T) {
	const starts = `<ul><li>a<li>b</ul>tail`
	assertListQueryNode(t, starts, `//ul/li[1]`, "a", 4, 9)
	assertListQueryNode(t, starts, `//ul/li[2]`, "b", 9, 14)
	assertListQueryNode(t, starts, `//ul/following-sibling::text()`, "tail", 19, 23)

	const inline = `<ul><li><span>a<li>b</ul>`
	assertListQueryNode(t, inline, `//ul/li[1]`, "a", 4, 15)
	assertListQueryNode(t, inline, `//ul/li[1]/span`, "a", 8, 15)
	assertListQueryNode(t, inline, `//ul/li[2]`, "b", 15, 20)

	const explicit = `<ul><li>a</li>b</ul>`
	assertListQueryNode(t, explicit, `//li`, "a", 4, 14)
	assertListQueryNode(t, explicit, `//li/following-sibling::text()`, "b", 14, 15)

	const absent = `<ul><div>a</li>b</div><li>x</ul>`
	assertListQueryNode(t, absent, `//ul/div`, "ab", 4, 22)
	assertListQueryNode(t, absent, `//ul/div/text()`, "ab", 9, 16)
	assertListQueryNode(t, absent, `//ul/li`, "x", 22, 27)
}

func TestQueryLINestingScopeAncestorAndEOFRecovery(t *testing.T) {
	const nested = `<ul><li>a<ul><li>b<li>c</ul>d</ul>`
	assertListQueryNode(t, nested, `/ul/li`, "abcd", 4, 29)
	assertListQueryNode(t, nested, `/ul/li/ul`, "bc", 9, 28)
	assertListQueryNode(t, nested, `/ul/li/ul/li[1]`, "b", 13, 18)
	assertListQueryNode(t, nested, `/ul/li/ul/li[2]`, "c", 18, 23)

	const unwind = `<ul><li>a<div><li>b</div>c</ul>`
	assertListQueryNode(t, unwind, `//ul/li[1]`, "a", 4, 14)
	assertListQueryNode(t, unwind, `//ul/li[1]/div`, "", 9, 14)
	assertListQueryNode(t, unwind, `//ul/li[2]`, "bc", 14, 26)

	const boundary = `<ul><li>a<section><li>b</section>c</ul>`
	assertListQueryNode(t, boundary, `//ul/li[1]`, "abc", 4, 34)
	assertListQueryNode(t, boundary, `//ul/li[1]/section/li`, "b", 18, 23)

	const ancestor = `<div><ul><li>a</ul>tail</div>`
	assertListQueryNode(t, ancestor, `//div/ul/li`, "a", 9, 14)
	assertListQueryNode(t, ancestor, `//div/ul/following-sibling::text()`, "tail", 19, 23)

	const eof = `<div><ul><li>a`
	assertListQueryNode(t, eof, `//div/ul/li`, "a", 9, len(eof))
}

func TestQueryDefinitionItemRecovery(t *testing.T) {
	const starts = `<dl><dt>a<dd>b<dt>c</dl>tail`
	assertListQueryNode(t, starts, `//dl/dt[1]`, "a", 4, 9)
	assertListQueryNode(t, starts, `//dl/dd`, "b", 9, 14)
	assertListQueryNode(t, starts, `//dl/dt[2]`, "c", 14, 19)
	assertListQueryNode(t, starts, `//dl/following-sibling::text()`, "tail", 24, 28)

	const inline = `<dl><dt><span>a<dd><em>b</dl>`
	assertListQueryNode(t, inline, `//dl/dt`, "a", 4, 15)
	assertListQueryNode(t, inline, `//dl/dt/span`, "a", 8, 15)
	assertListQueryNode(t, inline, `//dl/dd`, "b", 15, 24)
	assertListQueryNode(t, inline, `//dl/dd/em`, "b", 19, 24)

	const explicit = `<dl><dt>a</dt><dd>b</dd>z</dl>`
	assertListQueryNode(t, explicit, `//dl/dt`, "a", 4, 14)
	assertListQueryNode(t, explicit, `//dl/dd`, "b", 14, 24)
	assertListQueryNode(t, explicit, `//dl/dd/following-sibling::text()`, "z", 24, 25)

	const absent = `<dl><div>a</dd></dt>b</div><dt>x</dl>`
	assertListQueryNode(t, absent, `//dl/div`, "ab", 4, 27)
	assertListQueryNode(t, absent, `//dl/div/text()`, "ab", 9, 21)
	assertListQueryNode(t, absent, `//dl/dt`, "x", 27, 32)
}

func TestQueryDefinitionScopeAncestorAndEOFRecovery(t *testing.T) {
	const unwind = `<dl><dt>a<div><dd>b</div>c</dl>`
	assertListQueryNode(t, unwind, `//dl/dt`, "a", 4, 14)
	assertListQueryNode(t, unwind, `//dl/dt/div`, "", 9, 14)
	assertListQueryNode(t, unwind, `//dl/dd`, "bc", 14, 26)

	const boundary = `<dl><dt>a<section><dd>b</section>c</dl>`
	assertListQueryNode(t, boundary, `//dl/dt`, "abc", 4, 34)
	assertListQueryNode(t, boundary, `//dl/dt/section/dd`, "b", 18, 23)

	const ancestor = `<section><dl><dd>a</dl>tail</section>`
	assertListQueryNode(t, ancestor, `//section/dl/dd`, "a", 13, 18)
	assertListQueryNode(t, ancestor, `//section/dl/following-sibling::text()`, "tail", 23, 27)

	const eof = `<section><dl><dt>a`
	assertListQueryNode(t, eof, `//section/dl/dt`, "a", 13, len(eof))
}

func TestQueryListRecoveryPreservesMultibyteCRLFLocations(t *testing.T) {
	const document = "<ul>\r\n<li>é\r\n<span>x<li>y</ul>"
	assertListQueryNode(t, document, `//ul/li[1]`, "é\nx", 6, 21)
	assertListQueryNode(t, document, `//ul/li[1]/text()`, "é\n", 10, 14)
	assertListQueryNode(t, document, `//ul/li[1]/span`, "x", 14, 21)
	assertListQueryNode(t, document, `//ul/li[2]`, "y", 21, 26)
}

func TestQueryListExplicitEndsAndScopeRecovery(t *testing.T) {
	const implied = `<ul><li><dd><p>x</li>y</ul>`
	assertListQueryNode(t, implied, `//ul/li`, "x", 4, 21)
	assertListQueryNode(t, implied, `//ul/li/dd`, "x", 8, 16)
	assertListQueryNode(t, implied, `//ul/li/dd/p`, "x", 12, 16)
	assertListQueryNode(t, implied, `//ul/li/following-sibling::text()`, "y", 21, 22)

	const liScope = `<ul><li>a<ul></li>b</ul>`
	assertListQueryNode(t, liScope, `/ul/li`, "ab", 4, len(liScope))
	assertListQueryNode(t, liScope, `/ul/li/ul`, "b", 9, len(liScope))

	const ddScope = `<dl><dd>a<ul>b</dd>c</dl>`
	assertListQueryNode(t, ddScope, `//dl/dd`, "ab", 4, 19)
	assertListQueryNode(t, ddScope, `//dl/dd/ul`, "b", 9, 14)
	assertListQueryNode(t, ddScope, `//dl/dd/following-sibling::text()`, "c", 19, 20)

	const stopped = `<dl><dd>a<object>b</dd>c</object>d</dl>`
	assertListQueryNode(t, stopped, `//dl/dd`, "abcd", 4, 34)
	assertListQueryNode(t, stopped, `//dl/dd/object/text()`, "bc", 17, 24)
}

func TestQueryListStartScanExceptionsAndCrossFamilies(t *testing.T) {
	const div = `<ul><li>a<div>b<li>c</ul>`
	assertListQueryNode(t, div, `//ul/li[1]`, "ab", 4, 15)
	assertListQueryNode(t, div, `//ul/li[1]/div`, "b", 9, 15)
	assertListQueryNode(t, div, `//ul/li[2]`, "c", 15, 20)

	const section = `<ul><li>a<section>b<li>c</section>d</ul>`
	assertListQueryNode(t, section, `//ul/li`, "abcd", 4, 35)
	assertListQueryNode(t, section, `//ul/li/section/li`, "c", 19, 24)

	const definition = `<dl><dd>a<dt>b<dd>c</dl>`
	assertListQueryNode(t, definition, `//dl/dd[1]`, "a", 4, 9)
	assertListQueryNode(t, definition, `//dl/dt`, "b", 9, 14)
	assertListQueryNode(t, definition, `//dl/dd[2]`, "c", 14, 19)

	const liStoppedByDD = `<ul><li>a<dd>b<li>c</ul>`
	assertListQueryNode(t, liStoppedByDD, `//ul/li[1]`, "abc", 4, 19)
	assertListQueryNode(t, liStoppedByDD, `//ul/li/dd/li`, "c", 14, 19)

	const dtStoppedByLI = `<dl><dd>a<li>b<dt>c</dl>`
	assertListQueryNode(t, dtStoppedByLI, `//dl/dd`, "abc", 4, 19)
	assertListQueryNode(t, dtStoppedByLI, `//dl/dd/li/dt`, "c", 14, 19)
}

func TestQueryListParagraphButtonScopeAndAncestorEnds(t *testing.T) {
	const paragraph = `<p>a<li>b`
	assertListQueryNode(t, paragraph, `//p`, "a", 0, 4)
	assertListQueryNode(t, paragraph, `//p/following-sibling::li`, "b", 4, 9)

	const outsideButtonScope = `<p>a<button>b<li>c</button>d`
	assertListQueryNode(t, outsideButtonScope, `//p`, "abcd", 0, len(outsideButtonScope))
	assertListQueryNode(t, outsideButtonScope, `//p/button/li`, "c", 13, 18)

	for _, testCase := range []struct {
		document, expression, text string
		start, end                 int
	}{
		{document: `<div><li>x</div>y`, expression: `//div/li`, text: "x", start: 5, end: 10},
		{document: `<section><dd>x</section>y`, expression: `//section/dd`, text: "x", start: 9, end: 14},
		{document: `<ul><li>a<div>b</ul>y`, expression: `//ul/li`, text: "ab", start: 4, end: 15},
	} {
		assertListQueryNode(t, testCase.document, testCase.expression, testCase.text, testCase.start, testCase.end)
	}

	const blocked = `<span><li>x</span>y`
	assertListQueryNode(t, blocked, `//span/li`, "xy", 6, len(blocked))
}

func TestQueryListTokenizerEOFCommentsAndSupplementaryLocations(t *testing.T) {
	for _, document := range []string{`<UL><LI>a<LI>b</UL>`, `<ul><li/>a<li>b</ul>`, `<ul><li>a</li x/><li>b</ul>`} {
		results, err := xpath.Query(`//li`, document)
		if err != nil || len(results) != 2 || results[0].TextContent != "a" || results[1].TextContent != "b" {
			t.Fatalf("Expected tokenizer form to produce sibling li nodes, got %#v err=%v", results, err)
		}
	}

	for _, document := range []string{`<ul><li>a<li`, `<ul><li>a</li`} {
		assertListQueryNode(t, document, `//li`, "a", 4, len(document))
		assertListQueryNode(t, document, `//li/text()`, "a", 8, len(document))
	}

	const comment = `<ul><li>a<!--c--><li>b</ul>`
	assertListQueryNode(t, comment, `//ul/li[1]`, "a", 4, 17)
	assertListQueryNode(t, comment, `//ul/li[2]`, "b", 17, 22)

	const multiline = "<ul>\r\n<li>😀é\r\n<span>x<li>z</ul>"
	assertListQueryNode(t, multiline, `//ul/li[1]`, "😀é\nx", 6, 25)
	assertListQueryNode(t, multiline, `//ul/li[1]/text()`, "😀é\n", 10, 18)
	assertListQueryNode(t, multiline, `//ul/li[1]/span`, "x", 18, 25)
	assertListQueryNode(t, multiline, `//ul/li[2]`, "z", 25, 30)
}

func TestQueryListDuplicateAncestorAndDescendantBlockClosures(t *testing.T) {
	const duplicate = `<div><li><div>x</div>y</li>z</div>`
	assertListQueryNode(t, duplicate, `/div`, "xyz", 0, 34)
	assertListQueryNode(t, duplicate, `/div/li`, "xy", 5, 27)
	assertListQueryNode(t, duplicate, `/div/li/div`, "x", 9, 21)
	assertListQueryNode(t, duplicate, `/div/li/following-sibling::text()`, "z", 27, 28)

	for _, testCase := range []struct {
		document, root, item string
	}{
		{document: `<ul><li><section><span>x</section>y</li></ul>`, root: "ul", item: "li"},
		{document: `<dl><dd><section><span>x</section>y</dd></dl>`, root: "dl", item: "dd"},
	} {
		base := `/` + testCase.root + `/` + testCase.item
		assertListQueryNode(t, testCase.document, base, "xy", 4, 40)
		assertListQueryNode(t, testCase.document, base+`/section`, "x", 8, 34)
		assertListQueryNode(t, testCase.document, base+`/section/span`, "x", 17, 24)
		assertListQueryNode(t, testCase.document, base+`/section/following-sibling::text()`, "y", 34, 35)
	}
}

func TestQueryExplicitListEndDoesNotLeakEOFRecovery(t *testing.T) {
	for _, document := range []string{
		`<ul><li>x</li></ul><div>y`,
		`<dl><dd>x</dd></dl><section>y`,
	} {
		results, err := xpath.Query(`/html/body/*[last()]`, document)
		if err != nil || len(results) != 1 || results[0].EndLocation != len(document) {
			t.Fatalf("Expected document-mode EOF recovery after explicit list close for %q: %#v err=%v", document, results, err)
		}
	}
}

func TestQuerySearchDoesNotStopListItemStartScanLikeBrowsers(t *testing.T) {
	const list = `<ul><li>a<search>b<li>c</search>d</ul>`
	assertListQueryNode(t, list, `/ul/li[1]`, "ab", 4, 18)
	assertListQueryNode(t, list, `/ul/li[1]/search`, "b", 9, 18)
	assertListQueryNode(t, list, `/ul/li[2]`, "cd", 18, 33)

	const definition = `<dl><dt>a<search>b<dd>c</search>d</dl>`
	assertListQueryNode(t, definition, `/dl/dt`, "ab", 4, 18)
	assertListQueryNode(t, definition, `/dl/dt/search`, "b", 9, 18)
	assertListQueryNode(t, definition, `/dl/dd`, "cd", 18, 33)
}

func TestQueryPendingListItemsSurviveBodyAndHTMLEnds(t *testing.T) {
	for _, testCase := range []struct {
		document, expression                                     string
		htmlEnd, bodyEnd, itemStart, itemEnd, textStart, textEnd int
	}{
		{document: `<html><body><ul><li>x</body>y</html>`, expression: `//body/ul/li`, htmlEnd: 36, bodyEnd: 28, itemStart: 16, itemEnd: 36, textStart: 20, textEnd: 29},
		{document: `<html><body><dl><dd>x</html>y`, expression: `//body/dl/dd`, htmlEnd: 28, bodyEnd: 21, itemStart: 16, itemEnd: 29, textStart: 20, textEnd: 29},
		{document: `<html><body><li>x</body>y</html>`, expression: `//body/li`, htmlEnd: 32, bodyEnd: 24, itemStart: 12, itemEnd: 32, textStart: 16, textEnd: 25},
	} {
		assertListQueryNode(t, testCase.document, `//html`, "xy", 0, testCase.htmlEnd)
		assertListQueryNode(t, testCase.document, `//body`, "xy", 6, testCase.bodyEnd)
		assertListQueryNode(t, testCase.document, testCase.expression, "xy", testCase.itemStart, testCase.itemEnd)
		assertListQueryNode(t, testCase.document, testCase.expression+`/text()`, "xy", testCase.textStart, testCase.textEnd)
	}
}

func TestQueryMismatchedHeadingEndClosesCurrentHeadingAndItem(t *testing.T) {
	for _, testCase := range []struct {
		document, heading, item string
	}{
		{document: `<h1><li>x</h2>y`, heading: "h1", item: "li"},
		{document: `<h2><dd>x</h6>y`, heading: "h2", item: "dd"},
	} {
		assertListQueryNode(t, testCase.document, `//`+testCase.heading, "x", 0, 9)
		assertListQueryNode(t, testCase.document, `//`+testCase.heading+`/`+testCase.item, "x", 4, 9)
		assertListQueryNode(t, testCase.document, `//`+testCase.heading+`/following-sibling::text()`, "y", 14, 15)
	}
}

func TestQueryGenericDescendantEndInsideListItemVersusAncestorEnd(t *testing.T) {
	const descendant = `<ul><li><span><em>x</span>y</li></ul>`
	assertListQueryNode(t, descendant, `//ul/li`, "xy", 4, 32)
	assertListQueryNode(t, descendant, `//ul/li/span`, "x", 8, 26)
	assertListQueryNode(t, descendant, `//ul/li/span/em`, "x", 14, 19)

	const ancestor = `<span><li>x</span>y`
	assertListQueryNode(t, ancestor, `//span`, "xy", 0, len(ancestor))
	assertListQueryNode(t, ancestor, `//span/li`, "xy", 6, len(ancestor))
}

func TestQueryPendingListItemCommentPlacementAtDocumentEnd(t *testing.T) {
	const list = `<html><body><li>x</body><!--c-->y</html>`
	assertListQueryNode(t, list, `//body/li`, "xy", 12, 40)
	assertListQueryNode(t, list, `//body/li/text()`, "xy", 16, 33)
	assertListQueryNode(t, list, `//html/node()[last()]`, "c", 24, 32)

	const definition = `<html><body><dl><dd>x</html><!--c-->y`
	assertListQueryNode(t, definition, `//body/dl/dd`, "xy", 16, 37)
	assertListQueryNode(t, definition, `//body/dl/dd/text()`, "xy", 20, 37)
	assertListQueryNode(t, definition, `/node()[last()]`, "c", 28, 36)
}

func TestQueryPendingListReentryClassifiesDecodedCharacterTokens(t *testing.T) {
	for _, testCase := range []struct {
		reference, decoded string
		commentStart       int
	}{
		{reference: `&#32;`, decoded: " ", commentStart: 29},
		{reference: `&#9;`, decoded: "\t", commentStart: 28},
		{reference: `&#x20;`, decoded: " ", commentStart: 30},
		{reference: `&Tab;`, decoded: "\t", commentStart: 29},
		{reference: `&NewLine;`, decoded: "\n", commentStart: 33},
	} {
		document := `<html><body><li>x</body>` + testCase.reference + `<!--c--></html>`
		assertListQueryNode(t, document, `//body/li`, "x"+testCase.decoded, 12, len(document))
		assertListQueryNode(t, document, `//body/li/text()`, "x"+testCase.decoded, 16, testCase.commentStart)
		assertListQueryNode(t, document, `//html/node()[last()]`, "c", testCase.commentStart, testCase.commentStart+8)
	}

	const letter = `<html><body><li>x</body>&#65;<!--c--></html>`
	assertListQueryNode(t, letter, `//body/li`, "xA", 12, len(letter))
	assertListQueryNode(t, letter, `//body/li/text()`, "xA", 16, 29)
	assertListQueryNode(t, letter, `//body/li/node()[last()]`, "c", 29, 37)
}

func TestQueryPendingListClassifierIgnoresDoctypeAndMergesDuplicateHTML(t *testing.T) {
	for _, document := range []string{
		`<html><body><li>x</body><!DOCTYPE svg><!--c--></html>`,
		`<html><body><li>x</body><html lang=z><!--c--></html>`,
	} {
		assertListQueryNode(t, document, `//html/node()[last()]`, "c", len(document)-15, len(document)-7)
	}
	for _, document := range []string{
		`<html><body><li>x</html><!DOCTYPE svg><!--c-->`,
		`<html><body><li>x</html><html lang=z><!--c-->`,
	} {
		assertListQueryNode(t, document, `/node()[last()]`, "c", len(document)-8, len(document))
	}
	for _, document := range []string{
		`<html><body><li>x</body><html lang=z><!--c--></html>`,
		`<html><body><li>x</html><html lang=z><!--c-->`,
	} {
		results, err := xpath.Query(`//html`, document)
		if err != nil || len(results) != 1 || results[0].Attributes["lang"] != "z" {
			t.Fatalf("Expected duplicate html token to merge lang=z, got %#v err=%v", results, err)
		}
	}
}
