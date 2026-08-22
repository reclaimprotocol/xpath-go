package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func firstMathIntegrationQuery(t *testing.T, content, expression string) xpath.Result {
	t.Helper()
	results, err := xpath.Query(expression, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("XPath %q returned %d results, want 1: %#v", expression, len(results), results)
	}
	return results[0]
}

func TestQueryMathMLTextIntegrationOwnersAndExceptions(t *testing.T) {
	for _, owner := range []string{"mi", "mo", "mn", "ms", "mtext"} {
		content := `<math><mrow><` + owner + ` id=o>a<b id=h>x</b>c</` + owner + `><mn id=n>z</mn></mrow></math>`
		result := firstMathIntegrationQuery(t, content, `//*[@id='o']`)
		requireResultNamespace(t, result, mathMLResultNamespaceURI)
		if result.NodeName != owner || result.TextContent != "axc" {
			t.Fatalf("text integration owner <%s> mismatch: %#v", owner, result)
		}
		bold := firstMathIntegrationQuery(t, content, `//*[@id='h']`)
		requireResultNamespace(t, bold, "http://www.w3.org/1999/xhtml")
		if bold.NodeName != "b" || bold.TextContent != "x" {
			t.Fatalf("HTML child under MathML <%s> mismatch: %#v", owner, bold)
		}
		next := firstMathIntegrationQuery(t, content, `//*[@id='n']`)
		requireResultNamespace(t, next, mathMLResultNamespaceURI)
	}
}

func TestQueryMathMLTextIntegrationCoreRanges(t *testing.T) {
	const content = `<math><mrow><mi id=o>a<b id=h>x</b>c</mi><mn id=n>z</mn></mrow></math>`
	assertMathMLQuery(t, content, `//*[@id='o']`, "mi", "axc", 12, 41, 21, 36)
	bold := assertSVGQuery(t, content, `//*[@id='h']`, "b", "x", 22, 35, 30, 31)
	requireResultNamespace(t, bold, "http://www.w3.org/1999/xhtml")
	assertMathMLQuery(t, content, `//*[@id='n']`, "mn", "z", 41, 56, 50, 51)

	const exceptions = `<math><mi id=o>a<mglyph id=g /><malignmark id=l />b<span id=h>c</span></mi></math>`
	assertMathMLQuery(t, exceptions, `//*[@id='o']`, "mi", "abc", 6, 75, 15, 70)
	assertMathMLQuery(t, exceptions, `//*[@id='g']`, "mglyph", "", 16, 31, 31, 31)
	assertMathMLQuery(t, exceptions, `//*[@id='l']`, "malignmark", "", 31, 50, 50, 50)
	span := assertSVGQuery(t, exceptions, `//*[@id='h']`, "span", "c", 51, 70, 62, 63)
	requireResultNamespace(t, span, "http://www.w3.org/1999/xhtml")
}

func TestQueryMathMLTextIntegrationArbitraryHTMLAndBarrier(t *testing.T) {
	const arbitrary = `<math><mi id=o>a<section id=h><b id=b>x</b></section>c</mi><mn id=n>z</mn></math>`
	assertMathMLQuery(t, arbitrary, `//*[@id='o']`, "mi", "axc", 6, 59, 15, 54)
	section := assertSVGQuery(t, arbitrary, `//*[@id='h']`, "section", "x", 16, 53, 30, 43)
	requireResultNamespace(t, section, "http://www.w3.org/1999/xhtml")
	assertMathMLQuery(t, arbitrary, `//*[@id='n']`, "mn", "z", 59, 74, 68, 69)

	// Chrome-only current-WHATWG oracle; bundled jsdom closes at the first end.
	const barrier = `<math><mi id=o><span id=s>x</mi>y</span>z</mi><mn id=n>q</mn></math>`
	assertMathMLQuery(t, barrier, `//*[@id='o']`, "mi", "xyz", 6, 46, 15, 41)
	span := assertSVGQuery(t, barrier, `//*[@id='s']`, "span", "xy", 15, 40, 26, 33)
	requireResultNamespace(t, span, "http://www.w3.org/1999/xhtml")
	assertMathMLQuery(t, barrier, `//*[@id='n']`, "mn", "q", 46, 61, 55, 56)
}

func TestQueryMathMLTextIntegrationNestedRootsCharactersAndEOF(t *testing.T) {
	const nested = `<math id=M><mtext id=t><svg id=s><circle id=c /></svg><math id=i><mn id=n>x</mn></math></mtext><mo id=q>y</mo></math>`
	assertMathMLQuery(t, nested, `//*[@id='M']`, "math", "xy", 0, 117, 11, 110)
	assertMathMLQuery(t, nested, `//*[@id='t']`, "mtext", "x", 11, 95, 23, 87)
	svg := assertSVGQuery(t, nested, `//*[@id='s']`, "svg", "", 23, 54, 33, 48)
	requireResultNamespace(t, svg, "http://www.w3.org/2000/svg")
	assertMathMLQuery(t, nested, `//*[@id='i']`, "math", "x", 54, 87, 65, 80)
	assertMathMLQuery(t, nested, `//*[@id='q']`, "mo", "y", 95, 110, 104, 105)

	const characters = "<math><mi id=o>é\r\n&amp;\x00&#0;<!--c-->😀</mi></math>"
	assertMathMLQuery(t, characters, `//*[@id='o']`, "mi", "é\n&�😀", 6, 46, 15, 41)
	assertTableQuery(t, characters, `//*[@id='o']/text()[1]`, "é\n&�", 15, 29)
	assertTableQuery(t, characters, `//*[@id='o']/node()[2]`, "c", 29, 37)

	const eof = "<div><math><mi id=o>é\r\n😀<b id=b>x"
	assertMathMLQuery(t, eof, `//*[@id='o']`, "mi", "é\n😀x", 11, 37, 20, 37)
	bold := assertSVGQuery(t, eof, `//*[@id='b']`, "b", "x", 28, 37, 36, 37)
	requireResultNamespace(t, bold, "http://www.w3.org/1999/xhtml")
}

func TestQueryMathMLAnnotationXMLQualifiedAndExactEncodings(t *testing.T) {
	const qualified = `<math><annotation-xml id=a encoding=text/html>a<div id=h>x</div>c</annotation-xml><mn id=n>z</mn></math>`
	assertMathMLQuery(t, qualified, `//*[@id='a']`, "annotation-xml", "axc", 6, 82, 46, 65)
	div := assertSVGQuery(t, qualified, `//*[@id='h']`, "div", "x", 47, 64, 57, 58)
	requireResultNamespace(t, div, "http://www.w3.org/1999/xhtml")
	assertMathMLQuery(t, qualified, `//*[@id='n']`, "mn", "z", 82, 97, 91, 92)

	const decoded = `<math><annotation-xml id=a encoding=text&#47;html><span id=h>x</span></annotation-xml><mn id=n>z</mn></math>`
	span := assertSVGQuery(t, decoded, `//*[@id='h']`, "span", "x", 50, 69, 61, 62)
	requireResultNamespace(t, span, "http://www.w3.org/1999/xhtml")

	for _, content := range []string{
		`<math><annotation-xml id=a encoding=' text/html'><span id=h>x</span></annotation-xml><mn id=n>z</mn></math>`,
		`<math><annotation-xml id=a encoding=text/plain ENCODING=text/html><span id=h>x</span></annotation-xml><mn id=n>z</mn></math>`,
	} {
		span = firstMathIntegrationQuery(t, content, `//*[@id='h']`)
		requireResultNamespace(t, span, "http://www.w3.org/1999/xhtml")
		next := firstMathIntegrationQuery(t, content, `//*[@id='n']`)
		requireResultNamespace(t, next, "http://www.w3.org/1999/xhtml")
	}

	const noException = `<math><annotation-xml id=a encoding=text/html><mglyph id=g>x</mglyph></annotation-xml></math>`
	mglyph := assertSVGQuery(t, noException, `//*[@id='g']`, "mglyph", "x", 46, 69, 59, 60)
	requireResultNamespace(t, mglyph, "http://www.w3.org/1999/xhtml")
}

func TestQueryMathMLAnnotationXMLSVGAndLiveBarrier(t *testing.T) {
	const svgContent = `<math><annotation-xml id=a encoding=x><svg id=s><circle id=c /></svg><mrow id=r>x</mrow></annotation-xml></math>`
	assertMathMLQuery(t, svgContent, `//*[@id='a']`, "annotation-xml", "x", 6, 105, 38, 88)
	svg := assertSVGQuery(t, svgContent, `//*[@id='s']`, "svg", "", 38, 69, 48, 63)
	requireResultNamespace(t, svg, "http://www.w3.org/2000/svg")
	assertMathMLQuery(t, svgContent, `//*[@id='r']`, "mrow", "x", 69, 88, 80, 81)

	// Chrome-only current-WHATWG oracle.
	const barrier = `<math><annotation-xml id=a encoding=text/html><span id=s>x</annotation-xml>y</span>z</annotation-xml><mn id=n>q</mn></math>`
	annotation := firstMathIntegrationQuery(t, barrier, `//*[@id='a']`)
	requireResultNamespace(t, annotation, mathMLResultNamespaceURI)
	if annotation.TextContent != "xyz" {
		t.Fatalf("annotation live barrier text mismatch: %#v", annotation)
	}
	span := firstMathIntegrationQuery(t, barrier, `//*[@id='s']`)
	requireResultNamespace(t, span, "http://www.w3.org/1999/xhtml")
	if span.TextContent != "xy" {
		t.Fatalf("annotation live barrier span mismatch: %#v", span)
	}
	assertMathMLQuery(t, barrier, `//*[@id='n']`, "mn", "q", 101, 116, 110, 111)
}

func TestQueryMathMLIntegrationIncompleteReuseAndFormatting(t *testing.T) {
	const incompleteText = `<math><mi id=o>x<span`
	assertMathMLQuery(t, incompleteText, `//*[@id='o']`, "mi", "x", 6, 21, 15, 21)
	assertFormattingNoQuery(t, incompleteText, `//span`)

	const incompleteAnnotation = `<math><annotation-xml id=a encoding=text/html>x<div`
	annotation := firstMathIntegrationQuery(t, incompleteAnnotation, `//*[@id='a']`)
	requireResultNamespace(t, annotation, mathMLResultNamespaceURI)
	if annotation.TextContent != "x" || annotation.EndLocation != len(incompleteAnnotation) {
		t.Fatalf("incomplete annotation child start mismatch: %#v", annotation)
	}

	const formatting = `<b><math><mi><i>x</i></mi></math>y</b>z`
	assertTableQuery(t, formatting, `/b/math/following-sibling::text()`, "y", 33, 34)
	if results, err := xpath.Query(`/html/body/div`, `<div>x`); err != nil || len(results) != 1 || results[0].EndLocation != 6 {
		t.Fatalf("Implicit document EOF recovery after MathML integration mismatch: %#v err=%v", results, err)
	}
}
