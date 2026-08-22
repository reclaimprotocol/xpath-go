package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseIgnoredStructuralEndsKeepFosterElementOpen(t *testing.T) {
	for _, tc := range []struct {
		content                     string
		divEnd, rowStart, cellStart int
	}{
		{`<table><div>a</td>b<tr><td>x</table>`, 19, 19, 23},
		{`<table><div>a</tfoot>b<tr><td>x</table>`, 22, 22, 26},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatal(err)
		}
		if len(parsedBodyChildren(document)) != 2 {
			t.Fatalf("Expected foster div and table siblings, got %#v", parsedBodyChildren(document))
		}
		assertTableNode(t, parsedBodyChildren(document)[0], "div", "ab", 7, tc.divEnd)
		assertTableNode(t, parsedBodyChildren(document)[1], "table", "x", 0, len(tc.content))
		assertTableNode(t, tableElements(document, "tr")[0], "tr", "x", tc.rowStart, tc.rowStart+9)
		assertTableNode(t, tableElements(document, "td")[0], "td", "x", tc.cellStart, tc.cellStart+5)
	}
}

func TestParseIgnoredCellEndsKeepOpenDescendantWhileMatchingCellEndUnwinds(t *testing.T) {
	for _, tc := range []struct {
		end             string
		divEnd, cellEnd int
	}{{"th", 27, 27}, {"tfoot", 30, 30}, {"caption", 32, 32}, {"html", 29, 29}} {
		content := `<table><tr><td><div>a</` + tc.end + `>b</table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		assertTableNode(t, tableElements(document, "div")[0], "div", "ab", 15, tc.divEnd)
		assertTableNode(t, tableElements(document, "td")[0], "td", "ab", 11, tc.cellEnd)
	}
	const matching = `<table><tr><td><div>a</td>b</table>`
	document, err := NewHTMLParser().Parse(matching)
	if err != nil {
		t.Fatal(err)
	}
	assertTableNode(t, tableElements(document, "div")[0], "div", "a", 15, 21)
	assertTableNode(t, tableElements(document, "td")[0], "td", "a", 11, 26)
}

func TestParseCaptionDescendantsRespectCaptionInsertionMode(t *testing.T) {
	for _, tc := range []struct {
		content, divText   string
		divEnd, captionEnd int
	}{
		{`<table><caption><div>a</tr>b</caption></table>`, "ab", 28, 38},
		{`<table><caption><div>a</table>b`, "a", 22, 22},
		{`<table><caption><div>a<tr><td>x</table>`, "a", 22, 22},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatal(err)
		}
		assertTableNode(t, tableElements(document, "div")[0], "div", tc.divText, 16, tc.divEnd)
		caption := tableElements(document, "caption")[0]
		assertTableNode(t, caption, "caption", tc.divText, 7, tc.captionEnd)
	}
	const rowStart = `<table><caption><div>a<tr><td>x</table>`
	document, err := NewHTMLParser().Parse(rowStart)
	if err != nil {
		t.Fatal(err)
	}
	assertTableNode(t, tableElements(document, "tr")[0], "tr", "x", 22, 31)
	assertTableNode(t, tableElements(document, "td")[0], "td", "x", 26, 31)
}

func TestParseNestedTableStartClosesOuterFromSectionOrRow(t *testing.T) {
	for _, tc := range []struct {
		content                        string
		outerEnd, innerStart, innerEnd int
		wantRow                        bool
	}{
		{`<table><tbody><table><tr><td>x</table>z`, 14, 14, 38, false},
		{`<table><tbody><tr><table><tr><td>x</table>z`, 18, 18, 42, true},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatal(err)
		}
		tables := tableElements(document, "table")
		if len(tables) != 2 || tables[0].Parent != parsedBody(document) || tables[1].Parent != parsedBody(document) {
			t.Fatalf("Expected sibling tables, got %#v", tables)
		}
		assertTableNode(t, tables[0], "table", "", 0, tc.outerEnd)
		assertTableNode(t, tables[1], "table", "x", tc.innerStart, tc.innerEnd)
		if tc.wantRow {
			assertTableNode(t, tableElements(tables[0], "tr")[0], "tr", "", 14, 18)
		}
	}
}

func TestParseMatchingTableStructuralEndsUnwindFosteredOpenElement(t *testing.T) {
	for _, tc := range []struct {
		content, firstBodyText                                              string
		divStart, divEnd                                                    int
		bodyCount, firstBodyStart, firstBodyEnd, firstRowStart, firstRowEnd int
	}{
		{`<table><tbody><div>a</tbody><tr><td>x</table>`, "", 14, 20, 2, 7, 28, 0, 0},
		{`<table><tbody><tr><div>a</tr><tr><td>x</table>`, "x", 18, 24, 1, 7, 38, 14, 29},
		{`<table><tbody><tr><div>a</tbody><tbody><tr><td>x</table>`, "", 18, 24, 2, 7, 32, 14, 24},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatal(err)
		}
		if len(parsedBodyChildren(document)) != 2 {
			t.Fatalf("Expected foster div then table, got %#v", parsedBodyChildren(document))
		}
		assertTableNode(t, parsedBodyChildren(document)[0], "div", "a", tc.divStart, tc.divEnd)
		table := parsedBodyChildren(document)[1]
		assertTableNode(t, table, "table", "x", 0, len(tc.content))
		bodies := tableElements(table, "tbody")
		if len(bodies) != tc.bodyCount {
			t.Fatalf("Expected %d tbody nodes, got %#v", tc.bodyCount, bodies)
		}
		assertTableNode(t, bodies[0], "tbody", tc.firstBodyText, tc.firstBodyStart, tc.firstBodyEnd)
		if tc.firstRowStart != 0 {
			assertTableNode(t, tableElements(bodies[0], "tr")[0], "tr", "", tc.firstRowStart, tc.firstRowEnd)
		}
	}
}

func TestParseIncompleteCaptionAndColgroupEndTokensCloseAtEOF(t *testing.T) {
	captionDoc, err := NewHTMLParser().Parse(`<table><caption>x</caption`)
	if err != nil {
		t.Fatal(err)
	}
	caption := tableElements(captionDoc, "caption")[0]
	assertTableNode(t, caption, "caption", "x", 7, 26)
	text := findAllNodesByType(caption, types.TextNode)[0]
	if text.StartPos != 16 || text.EndPos != 26 || text.Value != "x" {
		t.Fatalf("Expected caption text range through discarded token, got %#v", text)
	}
	colDoc, err := NewHTMLParser().Parse(`<table><colgroup><col></colgroup`)
	if err != nil {
		t.Fatal(err)
	}
	group := tableElements(colDoc, "colgroup")[0]
	assertTableNode(t, group, "colgroup", "", 7, 32)
	assertTableNode(t, tableElements(group, "col")[0], "col", "", 17, 22)
}

func TestParseNestedTableFlushesFosterTextWithinOuterContext(t *testing.T) {
	const cellContent = `<div><table><tr><td><table>x<tr><td>y</table>z</table></div>`
	cellDoc, err := NewHTMLParser().Parse(cellContent)
	if err != nil {
		t.Fatal(err)
	}
	tables := tableElements(cellDoc, "table")
	if len(tables) != 2 {
		t.Fatalf("Expected two nested tables, got %#v", tables)
	}
	assertTableNode(t, tableElements(cellDoc, "div")[0], "div", "xyz", 0, 60)
	assertTableNode(t, tables[0], "table", "xyz", 5, 54)
	assertTableNode(t, tables[1], "table", "y", 20, 45)
	outerCell := tableElements(tables[0], "td")[0]
	assertTableNode(t, outerCell, "td", "xyz", 16, 46)
	if len(outerCell.Children) != 3 || outerCell.Children[0].Value != "x" || outerCell.Children[0].StartPos != 27 || outerCell.Children[0].EndPos != 28 || outerCell.Children[1] != tables[1] || outerCell.Children[2].Value != "z" || outerCell.Children[2].StartPos != 45 || outerCell.Children[2].EndPos != 46 {
		t.Fatalf("Expected x, inner table, z in outer cell, got %#v", outerCell.Children)
	}
	const captionContent = `<table><caption>a<table>x<tr><td>y</table>z</caption></table>`
	captionDoc, err := NewHTMLParser().Parse(captionContent)
	if err != nil {
		t.Fatal(err)
	}
	caption := tableElements(captionDoc, "caption")[0]
	assertTableNode(t, caption, "caption", "axyz", 7, 53)
	inner := tableElements(caption, "table")[0]
	assertTableNode(t, inner, "table", "y", 17, 42)
}

func TestParseColgroupNonspaceTextClosesThenFosters(t *testing.T) {
	const content = `<table><colgroup>x<tr><td>y</table>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0].Type != types.TextNode || parsedBodyChildren(document)[0].Value != "x" || parsedBodyChildren(document)[0].StartPos != 17 || parsedBodyChildren(document)[0].EndPos != 18 {
		t.Fatalf("Expected fostered x before table, got %#v", parsedBodyChildren(document))
	}
	assertTableNode(t, parsedBodyChildren(document)[1], "table", "y", 0, 35)
	assertTableNode(t, tableElements(document, "colgroup")[0], "colgroup", "", 7, 17)
}

func TestParseTableRowsAndCommentSeparatedFosterTextScaleBoundedly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping table scaling regression")
	}
	builders := map[string]func(int) string{
		"foster comments and rows": func(n int) string {
			var b strings.Builder
			b.WriteString(`<div><table>`)
			for i := 0; i < n; i++ {
				b.WriteString(`x<!--c--><tr><td>y</td></tr>`)
			}
			b.WriteString(`</table></div>`)
			return b.String()
		},
		"rows": func(n int) string {
			var b strings.Builder
			b.WriteString(`<table>`)
			for i := 0; i < n; i++ {
				b.WriteString(`<tr><td>x</td></tr>`)
			}
			b.WriteString(`</table>`)
			return b.String()
		},
		"sibling foster tables": func(n int) string {
			var b strings.Builder
			b.WriteString(`<div>`)
			for i := 0; i < n; i++ {
				b.WriteString(`<table>x<tr><td>y</td></tr></table>`)
			}
			b.WriteString(`</div>`)
			return b.String()
		},
	}
	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			measure := func(n int) time.Duration {
				content := build(n)
				if _, err := NewHTMLParser().Parse(content); err != nil {
					t.Fatal(err)
				}
				best := time.Duration(1<<63 - 1)
				for i := 0; i < 5; i++ {
					start := time.Now()
					if _, err := NewHTMLParser().Parse(content); err != nil {
						t.Fatal(err)
					}
					if d := time.Since(start); d < best {
						best = d
					}
				}
				return best
			}
			small, large := measure(250), measure(1000)
			ratio := float64(large) / float64(small)
			if large > small*17/2 && large-small > 5*time.Millisecond {
				t.Fatalf("%s scaled superlinearly: 250=%v 1000=%v ratio=%.1fx", name, small, large, ratio)
			}
			t.Logf("%s scaling: 250=%v 1000=%v ratio=%.1fx", name, small, large, ratio)
		})
	}
}
