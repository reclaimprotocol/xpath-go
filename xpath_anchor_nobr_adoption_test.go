package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryNestedAnchorAndNobrSpecialStarts(t *testing.T) {
	const a = `<p><a href=one>1<a href=two>2</a>3</a>4`
	assertTableQuery(t, a, `//p/a[1]`, "1", 3, 16)
	assertTableQuery(t, a, `//p/a[2]`, "2", 16, 33)
	assertTableQuery(t, a, `//p/a[2]/following-sibling::text()`, "34", 33, 39)

	const af = `<p><a id=a><b>1<a id=b>2</a>3</b>4`
	assertTableQuery(t, af, `//p/a[@id='a']/b`, "1", 11, 15)
	assertTableQuery(t, af, `//p/a/following-sibling::b`, "23", 11, 33)
	assertTableQuery(t, af, `//p/a/following-sibling::b/a`, "2", 15, 28)

	const n = `<p><nobr>1<nobr>2</nobr>3</nobr>4`
	assertTableQuery(t, n, `//p/nobr[1]`, "1", 3, 10)
	assertTableQuery(t, n, `//p/nobr[2]`, "2", 10, 24)
	assertTableQuery(t, n, `//p/nobr[2]/following-sibling::text()`, "34", 24, 33)

	const nf = `<p><nobr><b>1<nobr>2</nobr>3</b>4`
	assertTableQuery(t, nf, `//p/nobr/b`, "1", 9, 13)
	assertTableQuery(t, nf, `//p/nobr/following-sibling::b`, "23", 9, 32)
	assertTableQuery(t, nf, `//p/nobr/following-sibling::b/nobr`, "2", 13, 27)
}

func TestQueryAnchorNobrBlockMarkerAndButtonStarts(t *testing.T) {
	const ab = `<a id=a>1<div>2<a id=b>3</a>4</div>5`
	assertTableQuery(t, ab, `/a`, "1", 0, 15)
	assertTableQuery(t, ab, `/div/a[@id='a']`, "2", 0, 0)
	assertTableQuery(t, ab, `/div/a[@id='b']`, "3", 15, 28)

	const button = `<a id=a>1<button>2<a id=b>3</a>4</button>5`
	assertTableQuery(t, button, `/button/a[@id='a']`, "2", 0, 0)
	assertTableQuery(t, button, `/button/a[@id='b']`, "3", 18, 31)

	const marker = `<a id=a><object>1<a id=b>2</a>3</object>4`
	assertTableQuery(t, marker, `/a`, "1234", 0, 41)
	assertTableQuery(t, marker, `/a/object/a`, "2", 17, 30)

	const nb = `<nobr>1<div>2<nobr>3</nobr>4</div>5`
	assertTableQuery(t, nb, `/div/nobr[1]`, "2", 0, 0)
	assertTableQuery(t, nb, `/div/nobr[2]`, "3", 13, 27)

	const nm = `<nobr><object>1<nobr>2</nobr>3</object>4`
	assertTableQuery(t, nm, `/nobr`, "1234", 0, 40)
	assertTableQuery(t, nm, `/nobr/object/nobr`, "2", 15, 29)
}

func TestQueryAnchorNobrAdoptionEndsAndGating(t *testing.T) {
	const aEnd = `<p><a href=x>1<i>2</a>3</i>4`
	assertTableQuery(t, aEnd, `//p/a`, "12", 3, 22)
	assertTableQuery(t, aEnd, `//p/a/following-sibling::i`, "3", 14, 27)

	const nEnd = `<nobr>1<div>2</nobr>3</div>4`
	assertTableQuery(t, nEnd, `/nobr`, "1", 0, 20)
	assertTableQuery(t, nEnd, `/div/nobr`, "2", 0, 0)

	const absent = `<p>x</a>y</nobr>z`
	assertTableQuery(t, absent, `//p/text()`, "xyz", 3, 17)

	const incompleteA = `<a>1<div>2</a`
	assertTableQuery(t, incompleteA, `/a/div`, "2", 4, 13)
	const incompleteN = `<nobr>1<div>2</nobr`
	assertTableQuery(t, incompleteN, `/nobr/div`, "2", 7, 19)

	const selfA = `<p><a>1<a/>2`
	assertTableQuery(t, selfA, `//p/a[1]`, "1", 3, 7)
	assertTableQuery(t, selfA, `//p/a[2]`, "2", 7, 12)
	const selfN = `<p><nobr>1<nobr/>2`
	assertTableQuery(t, selfN, `//p/nobr[1]`, "1", 3, 10)
	assertTableQuery(t, selfN, `//p/nobr[2]`, "2", 10, 18)
}

func TestQueryAnchorNobrOutOfScopeAndMultibyte(t *testing.T) {
	const ao = `<a><object>x</a>y</object>z`
	assertTableQuery(t, ao, `/a`, "xyz", 0, 27)
	assertTableQuery(t, ao, `/a/object/text()`, "xy", 11, 17)
	const no = `<nobr><object>x</nobr>y</object>z`
	assertTableQuery(t, no, `/nobr`, "xyz", 0, 33)
	assertTableQuery(t, no, `/nobr/object/text()`, "xy", 14, 23)

	const am = "<p><a href=x>é\r\n<i>😀</a>z</i>w"
	assertTableQuery(t, am, `//p/a`, "é\n😀", 3, 28)
	assertTableQuery(t, am, `//p/a/following-sibling::i`, "z", 17, 33)
	const nm = "<p><nobr class=x>é\r\n<i>😀</nobr>z</i>w"
	assertTableQuery(t, nm, `//p/nobr`, "é\n😀", 3, 35)
	assertTableQuery(t, nm, `//p/nobr/following-sibling::i`, "z", 21, 40)
}

func TestQueryAnchorNobrOffStackStartLoopIdentity(t *testing.T) {
	const anchor = `<p><a id=a>1<div>2</div><a id=b>3`
	assertTableQuery(t, anchor, `/p/a`, "1", 3, 12)
	assertTableQuery(t, anchor, `/div/a`, "2", 3, 18)
	assertTableQuery(t, anchor, `/a[@id='b']`, "3", 24, 33)
	assertFormattingNoQuery(t, anchor, `/a[@id='a']`)

	const nobr = `<p><nobr id=a>1<div>2</div><nobr id=b>3`
	assertTableQuery(t, nobr, `/p/nobr`, "1", 3, 15)
	assertTableQuery(t, nobr, `/div/nobr`, "2", 3, 21)
	assertTableQuery(t, nobr, `/nobr[@id='a']`, "", 3, 27)
	assertTableQuery(t, nobr, `/nobr[@id='b']`, "3", 27, 39)
}

func TestQueryAnchorNobrSectionVirtualScopeStartLoop(t *testing.T) {
	const anchor = `<section><p><a id=a>1<div>2</div><a id=b>3</section>`
	assertTableQuery(t, anchor, `//section/p/a`, "1", 12, 21)
	assertTableQuery(t, anchor, `//section/div/a`, "2", 12, 27)
	assertTableQuery(t, anchor, `//section/a[@id='b']`, "3", 33, 42)
	assertFormattingNoQuery(t, anchor, `//section/a[@id='a'][a[@id='b']]`)

	const emptyNobr = `<section><p><nobr id=a>1<div>2</div><nobr id=b>3</section>`
	assertTableQuery(t, emptyNobr, `//section/p/nobr`, "1", 12, 24)
	assertTableQuery(t, emptyNobr, `//section/div/nobr`, "2", 12, 30)
	assertTableQuery(t, emptyNobr, `//section/nobr[@id='a']`, "", 12, 36)
	assertTableQuery(t, emptyNobr, `//section/nobr[@id='b']`, "3", 36, 48)

	const textNobr = `<section><p><nobr id=a>1<div>2</div>x<nobr id=b>3</section>`
	assertTableQuery(t, textNobr, `//section/nobr[@id='a']`, "x", 12, 37)
	assertTableQuery(t, textNobr, `//section/nobr[@id='b']`, "3", 37, 49)
	assertFormattingNoQuery(t, textNobr, `//section/nobr[@id='a']/nobr[@id='b']`)
}

func TestQueryAnchorNobrOffStackStartPreservesSuffixFormatting(t *testing.T) {
	for _, testCase := range []struct {
		name, subject, content, scope            string
		suffixStart, suffixEnd, newStart, newEnd int
	}{
		{name: "root anchor", subject: "a", content: `<p><a id=a><b>1<div>2</div>x<a id=b>3`, scope: ``, suffixStart: 3, suffixEnd: 28, newStart: 28, newEnd: 37},
		{name: "section anchor", subject: "a", content: `<section><p><a id=a><b>1<div>2</div>x<a id=b>3</section>`, scope: `//section`, suffixStart: 12, suffixEnd: 37, newStart: 37, newEnd: 46},
		{name: "explicit body anchor", subject: "a", content: `<html><body><p><a id=a><b>1<div>2</div>x<a id=b>3</body></html>`, scope: `//body`, suffixStart: 15, suffixEnd: 40, newStart: 40, newEnd: 63},
		{name: "root nobr", subject: "nobr", content: `<p><nobr id=a><b>1<div>2</div>x<nobr id=b>3`, scope: ``, suffixStart: 3, suffixEnd: 31, newStart: 31, newEnd: 43},
		{name: "section nobr", subject: "nobr", content: `<section><p><nobr id=a><b>1<div>2</div>x<nobr id=b>3</section>`, scope: `//section`, suffixStart: 12, suffixEnd: 40, newStart: 40, newEnd: 52},
		{name: "explicit body nobr", subject: "nobr", content: `<html><body><p><nobr id=a><b>1<div>2</div>x<nobr id=b>3</body></html>`, scope: `//body`, suffixStart: 15, suffixEnd: 43, newStart: 43, newEnd: 69},
	} {
		assertTableQuery(t, testCase.content, testCase.scope+`/`+testCase.subject+`[@id='a'][normalize-space(.)='x']`, "x", testCase.suffixStart, testCase.suffixEnd)
		assertTableQuery(t, testCase.content, testCase.scope+`/b/`+testCase.subject+`[@id='b']`, "3", testCase.newStart, testCase.newEnd)
		assertFormattingNoQuery(t, testCase.content, testCase.scope+`/`+testCase.subject+`[@id='a']/b/`+testCase.subject+`[@id='b']`)
	}
}

func TestQueryNobrSpecialStartAfterBodyAndHTML(t *testing.T) {
	for _, testCase := range []struct {
		name, content, cloneText   string
		cloneEnd, newStart, newEnd int
	}{
		{name: "after body", content: `<html><body><p><nobr id=a>1<div>2</div></body><nobr id=b>3</html>`, cloneEnd: 46, newStart: 46, newEnd: 65},
		{name: "after body whitespace", content: `<html><body><p><nobr id=a>1<div>2</div></body> <nobr id=b>3</html>`, cloneText: " ", cloneEnd: 47, newStart: 47, newEnd: 66},
		{name: "after html", content: `<html><body><p><nobr id=a>1<div>2</div></body></html><nobr id=b>3`, cloneEnd: 53, newStart: 53, newEnd: 65},
		{name: "after html whitespace", content: `<html><body><p><nobr id=a>1<div>2</div></body></html> <nobr id=b>3`, cloneText: " ", cloneEnd: 54, newStart: 54, newEnd: 66},
	} {
		assertTableQuery(t, testCase.content, `//body/nobr[@id='a']`, testCase.cloneText, 15, testCase.cloneEnd)
		assertTableQuery(t, testCase.content, `//body/nobr[@id='b' and not(ancestor::nobr)]`, "3", testCase.newStart, testCase.newEnd)
	}

	const bodyComment = `<html><body><p><nobr id=a>1<div>2</div></body><!--c--><nobr id=b>3</html>`
	assertTableQuery(t, bodyComment, `//html/node()[last()]`, "c", 46, 54)
	assertTableQuery(t, bodyComment, `//body/nobr[@id='a']`, "", 15, 54)
	assertTableQuery(t, bodyComment, `//body/nobr[@id='b']`, "3", 54, 73)

	const htmlComment = `<html><body><p><nobr id=a>1<div>2</div></body></html><!--c--><nobr id=b>3`
	assertTableQuery(t, htmlComment, `//html/following-sibling::node()`, "c", 53, 61)
	assertTableQuery(t, htmlComment, `//body/nobr[@id='a']`, "", 15, 61)
	assertTableQuery(t, htmlComment, `//body/nobr[@id='b']`, "3", 61, 73)
}

func TestQueryAnchorNobrExplicitDocumentRecoveryClosesEOFDescendants(t *testing.T) {
	for _, document := range []string{
		`<html><body><p><a>x</p></body>y</a><foo>x`,
		`<html><body><p><nobr>x</p></body>y</nobr><foo>x`,
	} {
		results, err := xpath.Query(`//foo`, document)
		if err != nil || len(results) != 1 || results[0].TextContent != "x" || results[0].EndLocation != len(document) {
			t.Fatalf("Expected explicit document to EOF-close foo for %q, got %#v err=%v", document, results, err)
		}
	}
}
