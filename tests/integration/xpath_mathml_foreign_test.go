package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

const mathMLResultNamespaceURI = "http://www.w3.org/1998/Math/MathML"

func assertMathMLQuery(t *testing.T, content, expression, name, value string, start, end, contentStart, contentEnd int) xpath.Result {
	t.Helper()
	result := assertSVGQuery(t, content, expression, name, value, start, end, contentStart, contentEnd)
	requireResultNamespace(t, result, mathMLResultNamespaceURI)
	return result
}

func TestQueryClosedMathMLIslandNamespaceAndExit(t *testing.T) {
	const content = `<div>a<math id=m><mrow id=r><mi id=i>x</mi></mrow></math>b</div>`
	assertMathMLQuery(t, content, `//*[@id='m']`, "math", "x", 6, 57, 17, 50)
	assertMathMLQuery(t, content, `//*[@id='r']`, "mrow", "x", 17, 50, 28, 43)
	assertMathMLQuery(t, content, `//*[@id='i']`, "mi", "x", 28, 43, 37, 38)
	assertTableQuery(t, content, `//*[@id='m']/following-sibling::text()`, "b", 57, 58)
}

func TestQueryMathMLSelfCloseAndAdjustedAttributes(t *testing.T) {
	const selfClose = `<math><mrow id=r /><mi id=i>x</mi></math><p>z</p>`
	assertMathMLQuery(t, selfClose, `//*[@id='r']`, "mrow", "", 6, 19, 19, 19)
	assertMathMLQuery(t, selfClose, `//*[@id='i']`, "mi", "x", 19, 34, 28, 29)
	assertTableQuery(t, selfClose, `/math/following-sibling::p`, "z", 41, 49)

	const attributes = `<math id=m definitionurl=root xlink:href=a><mrow id=n definitionurl=first definitionURL=second xml:lang=en /></math>`
	results, err := xpath.Query(`//*[@id='m']/@* | //*[@id='n']/@*`, attributes)
	if err != nil {
		t.Fatal(err)
	}
	definitionCount, xlinkCount, xmlCount := 0, 0, 0
	for _, result := range results {
		switch result.NodeName {
		case "definitionURL":
			definitionCount++
			if result.LocalName != "definitionURL" || result.NamespaceURI != "" {
				t.Fatalf("Adjusted definitionURL metadata mismatch: %#v", result)
			}
		case "xlink:href":
			xlinkCount++
			if result.LocalName != "href" || result.Prefix != "xlink" || result.NamespaceURI != "http://www.w3.org/1999/xlink" {
				t.Fatalf("MathML XLink attr metadata mismatch: %#v", result)
			}
		case "xml:lang":
			xmlCount++
			if result.LocalName != "lang" || result.Prefix != "xml" || result.NamespaceURI != "http://www.w3.org/XML/1998/namespace" {
				t.Fatalf("MathML XML attr metadata mismatch: %#v", result)
			}
		}
	}
	if definitionCount != 2 || xlinkCount != 1 || xmlCount != 1 {
		t.Fatalf("Adjusted MathML attr results incomplete: %#v", results)
	}
}

func TestQueryMathMLTextCDATACommentAndDoctype(t *testing.T) {
	const textContent = "<math>a&amp;b&#32;c\x00<mi>d</mi></math>z"
	assertTableQuery(t, textContent, `/math/text()`, "a&b c�", 6, 20)
	assertMathMLQuery(t, textContent, `/math/mi`, "mi", "d", 20, 30, 24, 25)

	const markup = `<math><!--c--><![CDATA[a<b>&copy;]]><mi>x</mi></math>z`
	assertTableQuery(t, markup, `/math/node()[1]`, "c", 6, 14)
	assertTableQuery(t, markup, `/math/node()[2]`, "a<b>&copy;", 14, 36)
	assertMathMLQuery(t, markup, `/math/mi`, "mi", "x", 36, 46, 40, 41)

	const doctype = `<math>a<!DOCTYPE x>b<mi /></math>z`
	assertTableQuery(t, doctype, `/math/text()`, "ab", 6, 20)
	assertMathMLQuery(t, doctype, `/math/mi`, "mi", "", 20, 26, 26, 26)
}

func TestQueryMathMLCloseRecoveryAndMalformedEnds(t *testing.T) {
	const caseClose = `<math><mrow id=r>x</MROW>y</math>z`
	assertMathMLQuery(t, caseClose, `//*[@id='r']`, "mrow", "x", 6, 25, 17, 18)
	assertTableQuery(t, caseClose, `/math/mrow/following-sibling::text()`, "y", 25, 26)

	const ancestor = `<math><mrow><mi id=i>x</mrow><mn id=n /></math>z`
	assertMathMLQuery(t, ancestor, `//*[@id='i']`, "mi", "x", 12, 22, 21, 22)
	assertMathMLQuery(t, ancestor, `//*[@id='n']`, "mn", "", 29, 40, 40, 40)

	const emptyEnd = `<math>a</>b</math>z`
	assertTableQuery(t, emptyEnd, `/math/text()`, "ab", 6, 11)

	const terminal = `<math>x</foo></math>`
	assertTableQuery(t, terminal, `/math/text()`, "x", 6, 7)

	const coalesced = `<math>x</foo>y</math>`
	assertTableQuery(t, coalesced, `/math/text()`, "xy", 6, 14)

	const bogus = `<math>a</42>b</math>z`
	assertTableQuery(t, bogus, `/math/node()[2]`, "42", 7, 12)

	const nonASCII = `<math><AÀ id=x>q</AÀ></math>`
	assertMathMLQuery(t, nonASCII, `//*[@id='x']`, "aÀ", "q", 6, 23, 16, 17)
}

func TestQueryMathMLBreakoutEndReprocessingAndFormattingExit(t *testing.T) {
	const block = `<math><mrow>a<div id=h>b</div>c</mrow><mi>d</mi></math>z`
	assertMathMLQuery(t, block, `/math`, "math", "a", 0, 13, 6, 13)
	assertMathMLQuery(t, block, `/math/mrow`, "mrow", "a", 6, 13, 12, 13)
	div := assertSVGQuery(t, block, `//*[@id='h']`, "div", "b", 13, 30, 23, 24)
	requireResultNamespace(t, div, "http://www.w3.org/1999/xhtml")
	mi := assertSVGQuery(t, block, `/mi`, "mi", "d", 38, 48, 42, 43)
	requireResultNamespace(t, mi, "http://www.w3.org/1999/xhtml")

	const endP = `<math><mrow>x</p>y</mrow></math>z`
	assertMathMLQuery(t, endP, `/math`, "math", "x", 0, 13, 6, 13)
	assertSVGQuery(t, endP, `/p`, "p", "", 0, 0, 0, 0)
	assertTableQuery(t, endP, `/p/following-sibling::text()`, "yz", 17, 33)

	const endBR = `<math><mrow>x</br>y</mrow></math>z`
	assertSVGQuery(t, endBR, `/br`, "br", "", 0, 0, 0, 0)
	assertTableQuery(t, endBR, `/br/following-sibling::text()`, "yz", 18, 34)

	const formatting = `<b><math><mi /></math>x</b>y`
	assertMathMLQuery(t, formatting, `/b/math`, "math", "", 3, 22, 9, 15)
	assertMathMLQuery(t, formatting, `/b/math/mi`, "mi", "", 9, 15, 15, 15)
	assertTableQuery(t, formatting, `/b/math/following-sibling::text()`, "x", 22, 23)
}

func TestQueryMathMLEOFIncompleteAndReuse(t *testing.T) {
	const unicode = "<div><math><mrow>é\r\n😀"
	assertMathMLQuery(t, unicode, `/div/math`, "math", "é\n😀", 5, 25, 11, 25)
	assertMathMLQuery(t, unicode, `/div/math/mrow`, "mrow", "é\n😀", 11, 25, 17, 25)
	assertTableQuery(t, unicode, `/div/math/mrow/text()`, "é\n😀", 17, 25)

	const incompleteEnd = `<math><mrow>x</mrow`
	assertMathMLQuery(t, incompleteEnd, `/math/mrow`, "mrow", "x", 6, 19, 12, 19)
	assertTableQuery(t, incompleteEnd, `/math/mrow/text()`, "x", 12, 19)

	const incompleteCDATA = `<math><![CDATA[x]]`
	assertTableQuery(t, incompleteCDATA, `/math/text()`, "x]]", 6, 18)

	const incompleteStart = `<div><math><mrow>x<mi a="`
	assertFormattingNoQuery(t, incompleteStart, `//mi`)
	assertTableQuery(t, incompleteStart, `/div/math/mrow/text()`, "x", 17, len(incompleteStart))

	if results, err := xpath.Query(`/html/body/div`, `<div>x`); err != nil || len(results) != 1 || results[0].EndLocation != 6 {
		t.Fatalf("Implicit document EOF recovery after MathML parse mismatch: %#v err=%v", results, err)
	}
}

func TestQueryMathMLNonBreakoutHTMLSpecialContrasts(t *testing.T) {
	const content = `<math id=m><mrow id=r><section id=s>x</section><form id=f>y</form><font id=q>z</font></mrow></math>`
	for _, id := range []string{"s", "f", "q"} {
		result := assertSVGQuery(t, content, `//*[@id='`+id+`']`, map[string]string{"s": "section", "f": "form", "q": "font"}[id], map[string]string{"s": "x", "f": "y", "q": "z"}[id], map[string]int{"s": 22, "f": 47, "q": 66}[id], map[string]int{"s": 47, "f": 66, "q": 85}[id], map[string]int{"s": 36, "f": 58, "q": 77}[id], map[string]int{"s": 37, "f": 59, "q": 78}[id])
		requireResultNamespace(t, result, mathMLResultNamespaceURI)
	}
}

func TestQueryMathMLBreakoutNamespaceQualifiedIdentityAndBarrier(t *testing.T) {
	const sameName = `<math><mrow><div>x</div><mrow id=h><span id=s>y</mrow>z</span>`
	foreign := assertMathMLQuery(t, sameName, `/math/mrow`, "mrow", "", 6, 12, 12, 12)
	html := assertSVGQuery(t, sameName, `//*[@id='h']`, "mrow", "y", 24, 54, 35, 47)
	span := assertSVGQuery(t, sameName, `//*[@id='s']`, "span", "y", 35, 47, 46, 47)
	requireResultNamespace(t, foreign, mathMLResultNamespaceURI)
	requireResultNamespace(t, html, "http://www.w3.org/1999/xhtml")
	requireResultNamespace(t, span, "http://www.w3.org/1999/xhtml")
	assertTableQuery(t, sameName, `//*[@id='h']/following-sibling::text()`, "z", 54, 55)

	const staleOwnerEnd = `<math id=m><mrow id=r><span id=h>x</math><mi id=i></mi>z`
	assertMathMLQuery(t, staleOwnerEnd, `//*[@id='m']`, "math", "", 0, 22, 11, 22)
	assertSVGQuery(t, staleOwnerEnd, `//*[@id='h']`, "span", "xz", 22, 56, 33, 56)
	mi := assertSVGQuery(t, staleOwnerEnd, `//*[@id='i']`, "mi", "", 41, 55, 50, 50)
	requireResultNamespace(t, mi, "http://www.w3.org/1999/xhtml")

	if results, err := xpath.Query(`/html/body/section`, `<math><mrow><div>x</div></mrow></math><section>y`); err != nil || len(results) != 1 || results[0].EndLocation != 48 {
		t.Fatalf("MathML breakout document EOF recovery mismatch: %#v err=%v", results, err)
	}
}
