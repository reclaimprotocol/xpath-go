package xpath_test

import (
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryForeignTableBreakoutBodyCellAndCaption(t *testing.T) {
	const svgBody = `<div id=o><svg id=s><g id=g>a<table id=t><tr><td>b</table>c</g></svg>d</div>`
	assertSVGQuery(t, svgBody, `//*[@id='s']`, "svg", "a", 10, 29, 20, 29)
	assertSVGQuery(t, svgBody, `//*[@id='g']`, "g", "a", 20, 29, 28, 29)
	table := assertSVGQuery(t, svgBody, `//*[@id='t']`, "table", "b", 29, 58, 41, 50)
	requireResultNamespace(t, table, "http://www.w3.org/1999/xhtml")
	assertTableQuery(t, svgBody, `//*[@id='t']/following-sibling::text()`, "cd", 58, 70)

	const mathBody = `<div id=o><math id=m><mrow id=r>a<table id=t><tr><td>b</table>c</mrow></math>d</div>`
	assertMathMLQuery(t, mathBody, `//*[@id='m']`, "math", "a", 10, 33, 21, 33)
	assertMathMLQuery(t, mathBody, `//*[@id='r']`, "mrow", "a", 21, 33, 32, 33)
	table = assertSVGQuery(t, mathBody, `//*[@id='t']`, "table", "b", 33, 62, 45, 54)
	requireResultNamespace(t, table, "http://www.w3.org/1999/xhtml")
	assertTableQuery(t, mathBody, `//*[@id='t']/following-sibling::text()`, "cd", 62, 78)

	const svgCell = `<table id=o><tr><td id=d><svg id=s><g id=g>a<table id=t><tr><td>b</table>c</g></svg>d</table>`
	assertSVGQuery(t, svgCell, `//*[@id='s']`, "svg", "a", 25, 44, 35, 44)
	assertSVGQuery(t, svgCell, `//*[@id='t']`, "table", "b", 44, 73, 56, 65)
	assertTableQuery(t, svgCell, `//*[@id='t']/following-sibling::text()`, "cd", 73, 85)

	const mathCell = `<table id=o><tr><td id=d><math id=m><mrow id=r>a<table id=t><tr><td>b</table>c</mrow></math>d</table>`
	assertMathMLQuery(t, mathCell, `//*[@id='m']`, "math", "a", 25, 48, 36, 48)
	assertSVGQuery(t, mathCell, `//*[@id='t']`, "table", "b", 48, 77, 60, 69)
	assertTableQuery(t, mathCell, `//*[@id='t']/following-sibling::text()`, "cd", 77, 93)

	const svgCaption = `<table id=o><caption id=p><svg id=s><g>a<table id=t><tr><td>b</table>c</g></svg>d</caption><tr><td>e</table>`
	assertSVGQuery(t, svgCaption, `//*[@id='s']`, "svg", "a", 26, 40, 36, 40)
	assertSVGQuery(t, svgCaption, `//*[@id='t']`, "table", "b", 40, 69, 52, 61)
	assertTableQuery(t, svgCaption, `//*[@id='t']/following-sibling::text()`, "cd", 69, 81)

	const mathCaption = `<table id=o><caption id=p><math id=m><mrow>a<table id=t><tr><td>b</table>c</mrow></math>d</caption><tr><td>e</table>`
	assertMathMLQuery(t, mathCaption, `//*[@id='m']`, "math", "a", 26, 44, 37, 44)
	assertSVGQuery(t, mathCaption, `//*[@id='t']`, "table", "b", 44, 73, 56, 65)
	assertTableQuery(t, mathCaption, `//*[@id='t']/following-sibling::text()`, "cd", 73, 89)
}

func TestQueryForeignTableBreakoutCustomizableSelect(t *testing.T) {
	const svgContent = `<select id=q><svg id=s><g>a<table id=t><tr><td>b</table>c</g></svg>d</select>`
	assertSVGQuery(t, svgContent, `//*[@id='s']`, "svg", "a", 13, 27, 23, 27)
	table := assertSVGQuery(t, svgContent, `//*[@id='t']`, "table", "b", 27, 56, 39, 48)
	requireResultNamespace(t, table, "http://www.w3.org/1999/xhtml")
	assertTableQuery(t, svgContent, `//*[@id='t']/following-sibling::text()`, "cd", 56, 68)

	const mathContent = `<select id=q><math id=m><mrow>a<table id=t><tr><td>b</table>c</mrow></math>d</select>`
	assertMathMLQuery(t, mathContent, `//*[@id='m']`, "math", "a", 13, 31, 24, 31)
	table = assertSVGQuery(t, mathContent, `//*[@id='t']`, "table", "b", 31, 60, 43, 52)
	requireResultNamespace(t, table, "http://www.w3.org/1999/xhtml")
	assertTableQuery(t, mathContent, `//*[@id='t']/following-sibling::text()`, "cd", 60, 76)
}

func TestQueryForeignTableBreakoutClosesStandardsParagraph(t *testing.T) {
	const paragraph = `<!doctype html><p id=p>a<svg id=s><g>b<table id=t><tr><td>x</table>c</g></svg>d`
	assertSVGQuery(t, paragraph, `//*[@id='p']`, "p", "ab", 15, 38, 23, 38)
	assertSVGQuery(t, paragraph, `//*[@id='t']`, "table", "x", 38, 67, 50, 59)
	assertTableQuery(t, paragraph, `//*[@id='t']/following-sibling::text()`, "cd", 67, 79)
}

func TestQueryForeignTableBreakoutParagraphDoctypeModes(t *testing.T) {
	const core = `<p id=p>a<svg id=s><g>b<table id=t><tr><td>x</table>c</g></svg>d`
	for _, test := range []struct {
		name, doctype string
		pText         string
		pStart, pEnd  int
		tableStart    int
	}{
		{"no-doctype-quirks", "", "abxcd", 0, 64, 23},
		{"canonical-standards", `<!doctype html>`, "ab", 15, 38, 38},
		{"tab-standards", "<!DOCTYPE\thtml>", "ab", 15, 38, 38},
		{"xhtml-limited-quirks", `<!DOCTYPE HTML PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">`, "ab", 121, 144, 144},
		{"html4-limited-quirks", `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">`, "ab", 102, 125, 125},
		{"legacy-force-quirks", `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN">`, "abxcd", 63, 127, 86},
		{"missing-name-force-quirks", `<!DOCTYPE>`, "abxcd", 10, 74, 33},
	} {
		t.Run(test.name, func(t *testing.T) {
			content := test.doctype + core
			p := assertSVGQuery(t, content, `//*[@id='p']`, "p", test.pText, test.pStart, test.pEnd, test.pStart+8, map[bool]int{true: test.pEnd, false: len(content)}[test.pText == "ab"])
			requireResultNamespace(t, p, "http://www.w3.org/1999/xhtml")
			table := firstMathIntegrationQuery(t, content, `//*[@id='t']`)
			if table.StartLocation != test.tableStart || table.NamespaceURI != "http://www.w3.org/1999/xhtml" {
				t.Fatalf("doctype mode table mismatch: %#v", table)
			}
		})
	}

}

func TestQueryForeignTableBreakoutAfterBodyAndSVGIntegrationOwners(t *testing.T) {
	const afterBody = `<html><body><p id=p>a<svg id=s><g>b<table id=t><tr><td>x</table>c</g></svg>d</body><table id=u><tr><td>e</table></html>`
	assertSVGQuery(t, afterBody, `//*[@id='p']`, "p", "abxcde", 12, 119, 20, 119)
	assertSVGQuery(t, `<!doctype html>`+afterBody, `//*[@id='p']`, "p", "ab", 27, 50, 35, 50)

	for _, test := range []struct {
		owner                                      string
		outerEnd, ownerEnd                         int
		innerStart, trigger, tableEnd, circleStart int
	}{
		{"foreignObject", 116, 95, 30, 49, 78, 95},
		{"desc", 98, 77, 21, 40, 69, 77},
		{"title", 100, 79, 22, 41, 70, 79},
	} {
		t.Run(test.owner, func(t *testing.T) {
			content := `<svg id=o><` + test.owner + ` id=f><svg id=i><g id=g>a<table id=t><tr><td>b</table>c</` + test.owner + `><circle id=q /></svg>`
			assertSVGQuery(t, content, `//*[@id='o']`, "svg", "abc", 0, test.outerEnd, 10, test.outerEnd-6)
			owner := assertSVGQuery(t, content, `//*[@id='f']`, test.owner, "abc", 10, test.ownerEnd, test.innerStart, test.ownerEnd-len(`</`+test.owner+`>`))
			requireResultNamespace(t, owner, "http://www.w3.org/2000/svg")
			assertSVGQuery(t, content, `//*[@id='i']`, "svg", "a", test.innerStart, test.trigger, test.innerStart+10, test.trigger)
			table := assertSVGQuery(t, content, `//*[@id='t']`, "table", "b", test.trigger, test.tableEnd, test.trigger+12, test.tableEnd-8)
			requireResultNamespace(t, table, "http://www.w3.org/1999/xhtml")
			circle := firstMathIntegrationQuery(t, content, `//*[@id='q']`)
			requireResultNamespace(t, circle, "http://www.w3.org/2000/svg")
			if circle.StartLocation != test.circleStart {
				t.Fatalf("outer SVG state did not resume after %s: %#v", test.owner, circle)
			}
		})
	}
}

func TestQueryForeignTableBreakoutExactStaleForeignEnds(t *testing.T) {
	const content = `<svg id=o><foreignObject id=f><svg id=i><g id=g>a<table id=t><tr><td>b</table>b</g></svg>c</foreignObject><circle id=q /></svg>`
	assertSVGQuery(t, content, `//*[@id='o']`, "svg", "abb", 0, 89, 10, 83)
	assertSVGQuery(t, content, `//*[@id='f']`, "foreignObject", "abb", 10, 83, 30, 83)
	assertSVGQuery(t, content, `//*[@id='i']`, "svg", "a", 30, 49, 40, 49)
	table := assertSVGQuery(t, content, `//*[@id='t']`, "table", "b", 49, 78, 61, 70)
	requireResultNamespace(t, table, "http://www.w3.org/1999/xhtml")
	assertTableQuery(t, content, `//*[@id='t']/following-sibling::text()`, "b", 78, 79)
	assertTableQuery(t, content, `//*[@id='o']/following-sibling::text()`, "c", 89, 90)
	circle := firstMathIntegrationQuery(t, content, `//*[@id='q']`)
	requireResultNamespace(t, circle, "http://www.w3.org/1999/xhtml")
}

func TestQueryForeignTableBreakoutFormattingNestedAndCoordinates(t *testing.T) {
	const formatting = `<b id=b><svg id=s><g>a<table id=t><tr><td>x</table>c</g></svg>d</b>e`
	assertSVGQuery(t, formatting, `//*[@id='b']`, "b", "axcd", 0, 67, 8, 63)
	assertSVGQuery(t, formatting, `//*[@id='t']`, "table", "x", 22, 51, 34, 43)
	assertTableQuery(t, formatting, `//*[@id='t']/following-sibling::text()`, "cd", 51, 63)

	const nested = `<div id=o><svg id=s><g><math id=m><mrow>a<table id=t><tr><td>x</table>b</mrow></math>c</g></svg>d</div>`
	assertSVGQuery(t, nested, `//*[@id='s']`, "svg", "a", 10, 41, 20, 41)
	mathSpelling := assertSVGQuery(t, nested, `//*[@id='m']`, "math", "a", 23, 41, 34, 41)
	requireResultNamespace(t, mathSpelling, "http://www.w3.org/2000/svg")
	assertSVGQuery(t, nested, `//*[@id='t']`, "table", "x", 41, 70, 53, 62)
	assertTableQuery(t, nested, `//*[@id='t']/following-sibling::text()`, "bcd", 70, 97)

	const unicode = "<div>é\r\n<svg id=s><g id=g>😀<table id=t><tr><td>x</table>z</div>"
	assertSVGQuery(t, unicode, `//*[@id='s']`, "svg", "😀", 9, 31, 19, 31)
	assertSVGQuery(t, unicode, `//*[@id='g']`, "g", "😀", 19, 31, 27, 31)
	assertSVGQuery(t, unicode, `//*[@id='t']`, "table", "x", 31, 60, 43, 52)
}

func TestQueryForeignTableBreakoutGuardsIncompleteAndReuse(t *testing.T) {
	const direct = `<div id=o><table id=t><svg id=s><circle /></svg><tr><td>x</table>z</div>`
	assertSVGQuery(t, direct, `//*[@id='s']`, "svg", "", 22, 48, 32, 42)
	assertSVGQuery(t, direct, `//*[@id='t']`, "table", "x", 10, 65, 22, 57)

	const foreignObject = `<svg id=s><foreignObject id=f><table id=t><tr><td>x</table></foreignObject><circle /></svg>`
	assertSVGQuery(t, foreignObject, `//*[@id='t']`, "table", "x", 30, 59, 42, 51)

	const integration = `<math><mi id=i><table id=t><tr><td>x</table>y</mi><mn>z</mn></math>`
	assertMathMLQuery(t, integration, `//*[@id='i']`, "mi", "xy", 6, 50, 15, 45)
	table := assertSVGQuery(t, integration, `//*[@id='t']`, "table", "x", 15, 44, 27, 36)
	requireResultNamespace(t, table, "http://www.w3.org/1999/xhtml")

	const endTag = `<svg id=s><g id=g>x</table>y</g></svg>z`
	assertSVGQuery(t, endTag, `//*[@id='g']`, "g", "xy", 10, 32, 18, 28)
	assertTableQuery(t, endTag, `//*[@id='g']/text()`, "xy", 18, 28)

	for _, incomplete := range []string{`<svg id=s><g id=g>a<table`, `<math id=m><mrow id=r>a<table`} {
		if results, err := xpath.Query(`//table`, incomplete); err != nil || len(results) != 0 {
			t.Fatalf("incomplete foreign table start emitted a table for %q: results=%#v err=%v", incomplete, results, err)
		}
	}
	if results, err := xpath.Query(`/html/body/div`, `<div>x`); err != nil || len(results) != 1 || results[0].EndLocation != 6 {
		t.Fatalf("Implicit document EOF recovery after foreign-table parsing mismatch: %#v err=%v", results, err)
	}

	for _, start := range []string{`<TaBlE id=t>`, `<table id=t />`, `<table id=t data-x=y>`, `<table id=t data-x=y/>`} {
		content := `<svg id=s><g>a` + start + `<tr><td>x</table>z`
		result := firstMathIntegrationQuery(t, content, `//*[@id='t']`)
		requireResultNamespace(t, result, "http://www.w3.org/1999/xhtml")
		if result.StartLocation != strings.Index(content, start) || result.TextContent != "x" {
			t.Fatalf("foreign table emitted variant mismatch for %q: %#v", start, result)
		}
	}
}
