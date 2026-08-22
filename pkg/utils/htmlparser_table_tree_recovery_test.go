package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func assertTableNode(t *testing.T, node *types.Node, name, text string, start, end int) {
	t.Helper()
	if node == nil || node.Name != name || node.TextContent != text || node.StartPos != start || node.EndPos != end {
		t.Fatalf("Expected <%s> text %q at %d:%d, got %#v", name, text, start, end, node)
	}
}

func tableElements(node *types.Node, name string) []*types.Node {
	var matches []*types.Node
	if node == nil {
		return matches
	}
	if node.Type == types.ElementNode && node.Name == name {
		matches = append(matches, node)
	}
	for _, child := range node.Children {
		matches = append(matches, tableElements(child, name)...)
	}
	return matches
}

func TestParseTableCreatesImplicitSectionsAndRows(t *testing.T) {
	t.Run("tbody", func(t *testing.T) {
		const content = `<table><tr><td>x</td></tr></table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		table := tableElements(document, "table")[0]
		assertTableNode(t, table, "table", "x", 0, 34)
		if len(table.Children) != 1 {
			t.Fatalf("Expected one implicit tbody, got %#v", table.Children)
		}
		tbody := table.Children[0]
		assertTableNode(t, tbody, "tbody", "x", 0, 0)
		assertTableNode(t, tbody.Children[0], "tr", "x", 7, 26)
		assertTableNode(t, tbody.Children[0].Children[0], "td", "x", 11, 21)
	})

	t.Run("row", func(t *testing.T) {
		const content = `<table><td>x</td><th>y</th></table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		table := tableElements(document, "table")[0]
		if len(table.Children) != 1 || len(table.Children[0].Children) != 1 {
			t.Fatalf("Expected implicit tbody/tr, got %#v", table.Children)
		}
		tbody, row := table.Children[0], table.Children[0].Children[0]
		assertTableNode(t, tbody, "tbody", "xy", 0, 0)
		assertTableNode(t, row, "tr", "xy", 0, 0)
		assertTableNode(t, row.Children[0], "td", "x", 7, 17)
		assertTableNode(t, row.Children[1], "th", "y", 17, 27)
	})

	t.Run("row inside explicit tbody", func(t *testing.T) {
		const content = `<table><tbody><td>x</tbody></table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		tbody := tableElements(document, "tbody")[0]
		assertTableNode(t, tbody, "tbody", "x", 7, 27)
		if len(tbody.Children) != 1 {
			t.Fatalf("Expected one implicit tr, got %#v", tbody.Children)
		}
		assertTableNode(t, tbody.Children[0], "tr", "x", 0, 0)
		assertTableNode(t, tbody.Children[0].Children[0], "td", "x", 14, 19)
	})
}

func TestParseTableAutoClosesRowsCellsAndTransitionsSections(t *testing.T) {
	t.Run("rows and cells", func(t *testing.T) {
		const content = `<table><tr><td>a<td>b<tr><th>c</table>tail`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		table := tableElements(document, "table")[0]
		assertTableNode(t, table, "table", "abc", 0, 38)
		rows := tableElements(table, "tr")
		if len(rows) != 2 {
			t.Fatalf("Expected two rows, got %#v", rows)
		}
		assertTableNode(t, rows[0], "tr", "ab", 7, 21)
		assertTableNode(t, rows[1], "tr", "c", 21, 30)
		cells := append(tableElements(rows[0], "td"), tableElements(rows[1], "th")...)
		for i, want := range []struct {
			name, text string
			start, end int
		}{{"td", "a", 11, 16}, {"td", "b", 16, 21}, {"th", "c", 25, 30}} {
			assertTableNode(t, cells[i], want.name, want.text, want.start, want.end)
		}
		if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].Type != types.TextNode || parsedBodyChildren(document)[1].Value != "tail" || parsedBodyChildren(document)[1].StartPos != 38 || parsedBodyChildren(document)[1].EndPos != 42 {
			t.Fatalf("Expected tail after table at 38:42, got %#v", parsedBodyChildren(document))
		}
	})

	t.Run("explicit sections", func(t *testing.T) {
		const content = `<table><thead><tr><td>h<tbody><tr><td>b<tfoot><tr><td>f</table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		table := tableElements(document, "table")[0]
		if len(table.Children) != 3 {
			t.Fatalf("Expected thead/tbody/tfoot siblings, got %#v", table.Children)
		}
		assertTableNode(t, table.Children[0], "thead", "h", 7, 23)
		assertTableNode(t, table.Children[1], "tbody", "b", 23, 39)
		assertTableNode(t, table.Children[2], "tfoot", "f", 39, 55)
	})

	t.Run("implicit section after explicit close", func(t *testing.T) {
		const content = `<table><tbody><tr><td>a</tbody><tr><td>b</table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		table := tableElements(document, "table")[0]
		if len(table.Children) != 2 {
			t.Fatalf("Expected explicit and implicit tbody siblings, got %#v", table.Children)
		}
		assertTableNode(t, table.Children[0], "tbody", "a", 7, 31)
		assertTableNode(t, table.Children[1], "tbody", "b", 0, 0)
		assertTableNode(t, table.Children[1].Children[0], "tr", "b", 31, 40)
	})
}

func TestParseTableFosterParentsTextAndElements(t *testing.T) {
	t.Run("mixed table text buffer fosters as one unit", func(t *testing.T) {
		const content = "<div><table> \nA\t<tr><td>x</td></tr></table></div>"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		div := tableElements(document, "div")[0]
		if len(div.Children) != 2 || div.Children[0].Type != types.TextNode || div.Children[0].Value != " \nA\t" || div.Children[0].StartPos != 12 || div.Children[0].EndPos != 16 || div.Children[1].Name != "table" {
			t.Fatalf("Expected the complete mixed whitespace/nonspace buffer fostered as one node, got %#v", div.Children)
		}
	})

	t.Run("text coalesces across table source", func(t *testing.T) {
		const content = `before<table>alpha<tr><td>x</td></tr>omega</table>after`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		if len(parsedBodyChildren(document)) != 3 {
			t.Fatalf("Expected fostered text, table, after, got %#v", parsedBodyChildren(document))
		}
		before := parsedBodyChildren(document)[0]
		if before.Type != types.TextNode || before.Value != "beforealphaomega" || before.StartPos != 0 || before.EndPos != 42 {
			t.Fatalf("Expected fostered coalesced text at 0:42, got %#v", before)
		}
		assertTableNode(t, parsedBodyChildren(document)[1], "table", "x", 6, 50)
		if parsedBodyChildren(document)[2].Value != "after" || parsedBodyChildren(document)[2].StartPos != 50 || parsedBodyChildren(document)[2].EndPos != 55 {
			t.Fatalf("Expected after at 50:55, got %#v", parsedBodyChildren(document)[2])
		}
	})

	t.Run("elements preserve source order before table", func(t *testing.T) {
		const content = `<div><table><span>a</span><tr><td>x</td></tr><b>b</b></table>z</div>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		div := tableElements(document, "div")[0]
		if len(div.Children) != 4 {
			t.Fatalf("Expected span, b, table, z, got %#v", div.Children)
		}
		assertTableNode(t, div.Children[0], "span", "a", 12, 26)
		assertTableNode(t, div.Children[1], "b", "b", 45, 53)
		assertTableNode(t, div.Children[2], "table", "x", 5, 61)
		if div.Children[3].Value != "z" || div.Children[3].StartPos != 61 || div.Children[3].EndPos != 62 {
			t.Fatalf("Expected z after table, got %#v", div.Children[3])
		}
	})
}

func TestParseTableTextTokenizationAndFosterOrder(t *testing.T) {
	t.Run("first child literal and reference foster before table", func(t *testing.T) {
		for _, tc := range []struct {
			content, value                                      string
			tableStart, tableEnd, textStart, textEnd, tailStart int
		}{
			{`<table>x<tr><td>y</table>z`, "x", 0, 25, 7, 8, 25},
			{`<div><table>x<tr><td>y</table>z</div>`, "x", 5, 30, 12, 13, 30},
			{`<table>&amp;<tr><td>y</table>z`, "&", 0, 29, 7, 12, 29},
		} {
			document, err := NewHTMLParser().Parse(tc.content)
			if err != nil {
				t.Fatal(err)
			}
			table := tableElements(document, "table")[0]
			parent := table.Parent
			if len(parent.Children) < 3 || parent.Children[0].Type != types.TextNode || parent.Children[0].Value != tc.value || parent.Children[0].StartPos != tc.textStart || parent.Children[0].EndPos != tc.textEnd || parent.Children[1] != table {
				t.Fatalf("Expected fostered %q immediately before first-child table, got %#v", tc.value, parent.Children)
			}
			assertTableNode(t, table, "table", "y", tc.tableStart, tc.tableEnd)
			tail := parent.Children[2]
			if tail.Value != "z" || tail.StartPos != tc.tailStart || tail.EndPos != tc.tailStart+1 {
				t.Fatalf("Expected z after table, got %#v", tail)
			}
		}
	})
	t.Run("decoded whitespace stays and NUL disappears", func(t *testing.T) {
		spaceDoc, err := NewHTMLParser().Parse(`<table>&#32;<tr><td>y</table>`)
		if err != nil {
			t.Fatal(err)
		}
		spaceTable := tableElements(spaceDoc, "table")[0]
		if len(spaceTable.Children) != 2 || spaceTable.Children[0].Type != types.TextNode || spaceTable.Children[0].Value != " " || spaceTable.Children[0].StartPos != 7 || spaceTable.Children[0].EndPos != 12 {
			t.Fatalf("Expected decoded table whitespace at 7:12, got %#v", spaceTable.Children)
		}
		nulDoc, err := NewHTMLParser().Parse("<table>\x00<tr><td>y</table>")
		if err != nil {
			t.Fatal(err)
		}
		nulTable := tableElements(nulDoc, "table")[0]
		if nulTable.TextContent != "y" || len(nulTable.Children) != 1 || nulTable.Children[0].Name != "tbody" {
			t.Fatalf("Expected table-context NUL ignored, got %#v", nulTable)
		}
	})
}

func TestParseTableFosteredContentContributesToAncestorText(t *testing.T) {
	for _, tc := range []struct {
		content string
		element bool
	}{
		{`<div>p<table>x<tr><td>y</table>z</div>`, false},
		{`<div>p<table><span>x</span><tr><td>y</table>z</div>`, true},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatal(err)
		}
		div := tableElements(document, "div")[0]
		assertTableNode(t, div, "div", "pxyz", 0, len(tc.content))
		if tc.element {
			assertTableNode(t, tableElements(div, "span")[0], "span", "x", 13, 27)
		}
		if tableElements(div, "table")[0].TextContent != "y" {
			t.Fatalf("Expected table text to exclude fostered x, got %#v", div)
		}
	}
}

func TestParseTableColCaptionAndInvalidEndRecovery(t *testing.T) {
	t.Run("explicit colgroup", func(t *testing.T) {
		const content = `<table><colgroup><col></colgroup><tr><td>x</table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		assertTableNode(t, tableElements(document, "colgroup")[0], "colgroup", "", 7, 33)
		assertTableNode(t, tableElements(document, "col")[0], "col", "", 17, 22)
	})
	for _, tc := range []struct {
		content, name string
		start, end    int
	}{
		{`<table><tbody><tr><td>x<caption>c</caption><tr><td>y</table>`, "caption", 23, 43},
		{`<table><tbody><tr><td>x<col><tr><td>y</table>`, "col", 23, 28},
		{`<table><tbody><tr><td>x<colgroup><col><tr><td>y</table>`, "colgroup", 23, 38},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatal(err)
		}
		node := tableElements(document, tc.name)[0]
		assertTableNode(t, node, tc.name, map[string]string{"caption": "c", "col": "", "colgroup": ""}[tc.name], tc.start, tc.end)
		cells := tableElements(document, "td")
		assertTableNode(t, cells[0], "td", "x", 18, 23)
	}
	for _, tc := range []struct {
		content, xpathName string
		start, end         int
	}{
		{`<table><tbody><tr><td>a</tfoot>b</table>`, "td", 18, 32},
		{`<table><tbody></tr><tr><td>x</table>`, "tr", 19, 28},
		{`<table></tbody><tr><td>x</table>`, "tr", 15, 24},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatal(err)
		}
		node := tableElements(document, tc.xpathName)[0]
		assertTableNode(t, node, tc.xpathName, map[string]string{"td": "ab", "tr": "x"}[tc.xpathName], tc.start, tc.end)
	}
}

func TestParseUnclosedFosterElementReentersTableMode(t *testing.T) {
	const content = `<table><div>a<tr><td>x</table>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsedBodyChildren(document)) != 3 {
		t.Fatalf("Expected div, table, z siblings, got %#v", parsedBodyChildren(document))
	}
	assertTableNode(t, parsedBodyChildren(document)[0], "div", "a", 7, 13)
	assertTableNode(t, parsedBodyChildren(document)[1], "table", "x", 0, 30)
	if parsedBodyChildren(document)[2].Value != "z" || parsedBodyChildren(document)[2].StartPos != 30 || parsedBodyChildren(document)[2].EndPos != 31 {
		t.Fatalf("Expected z at 30:31, got %#v", parsedBodyChildren(document)[2])
	}
}

func TestParseTableWhitespaceCommentsMultibyteAndNestedBoundaries(t *testing.T) {
	t.Run("whitespace and comment stay in table", func(t *testing.T) {
		const content = "<div><table> \n<!--c--><tr><td>x</td></tr>\t</table></div>"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		table := tableElements(document, "table")[0]
		if len(table.Children) != 3 || table.Children[0].Type != types.TextNode || table.Children[0].Value != " \n" || table.Children[0].StartPos != 12 || table.Children[0].EndPos != 14 || table.Children[1].Type != types.CommentNode || table.Children[1].Value != "c" || table.Children[1].StartPos != 14 || table.Children[1].EndPos != 22 {
			t.Fatalf("Expected table whitespace/comment before tbody, got %#v", table.Children)
		}
		tbody := table.Children[2]
		assertTableNode(t, tbody, "tbody", "x\t", 0, 0)
		if len(tbody.Children) != 2 || tbody.Children[1].Value != "\t" || tbody.Children[1].StartPos != 41 || tbody.Children[1].EndPos != 42 {
			t.Fatalf("Expected trailing table whitespace inside tbody, got %#v", tbody.Children)
		}
	})

	t.Run("multibyte foster positions", func(t *testing.T) {
		const content = `<div>é<table>β<tr><td>γ</td></tr>δ</table>ω</div>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		div := tableElements(document, "div")[0]
		if len(div.Children) != 3 || div.Children[0].Value != "éβδ" || div.Children[0].StartPos != 5 || div.Children[0].EndPos != 38 {
			t.Fatalf("Expected fostered multibyte text at byte range 5:38, got %#v", div.Children)
		}
		assertTableNode(t, div.Children[1], "table", "γ", 7, 46)
		cell := tableElements(div.Children[1], "td")[0]
		assertTableNode(t, cell, "td", "γ", 20, 31)
		if div.Children[2].Value != "ω" || div.Children[2].StartPos != 46 || div.Children[2].EndPos != 48 {
			t.Fatalf("Expected omega at byte range 46:48, got %#v", div.Children[2])
		}
	})

	t.Run("CRLF and supplementary foster positions", func(t *testing.T) {
		const content = "<div>A\r\n<table> 🧪B<tr><td>x</td></tr>C</table>Z</div>"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		div := tableElements(document, "div")[0]
		if len(div.Children) != 3 {
			t.Fatalf("Expected fostered text, table, Z, got %#v", div.Children)
		}
		text := div.Children[0]
		if text.Type != types.TextNode || text.Value != "A\n 🧪BC" || text.StartPos != 5 || text.EndPos != 41 || text.StartLine != 1 || text.StartColumn != 6 || text.EndLine != 2 || text.EndColumn != 32 {
			t.Fatalf("Expected normalized foster text at bytes 5:41 and 1:6-2:32, got %#v", text)
		}
		assertTableNode(t, div.Children[1], "table", "x", 8, 49)
		if div.Children[2].Value != "Z" || div.Children[2].StartPos != 49 || div.Children[2].EndPos != 50 {
			t.Fatalf("Expected Z at bytes 49:50, got %#v", div.Children[2])
		}
	})

	t.Run("nested table boundary", func(t *testing.T) {
		const content = `<table><tr><td>a<table><tr><td>b</table>c</table>tail`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		tables := tableElements(document, "table")
		if len(tables) != 2 {
			t.Fatalf("Expected nested tables, got %#v", tables)
		}
		assertTableNode(t, tables[0], "table", "abc", 0, 49)
		assertTableNode(t, tables[1], "table", "b", 16, 40)
		outerCell := tableElements(tables[0], "td")[0]
		assertTableNode(t, outerCell, "td", "abc", 11, 41)
		if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].Value != "tail" || parsedBodyChildren(document)[1].StartPos != 49 || parsedBodyChildren(document)[1].EndPos != 53 {
			t.Fatalf("Expected tail after outer table, got %#v", parsedBodyChildren(document))
		}
	})
}

func TestParseTableEOFRecoveryAndParserReuse(t *testing.T) {
	parser := NewHTMLParser()
	for i := 0; i < 2; i++ {
		content := `<table><tr><td>x`
		if i == 1 {
			content = `<div><table><span>a</span><tr><td>x</td></tr></table></div>`
		}
		document, err := parser.Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		table := tableElements(document, "table")[0]
		if i == 1 {
			div := tableElements(document, "div")[0]
			if len(div.Children) != 2 || div.Children[0].Name != "span" || div.Children[1] != table {
				t.Fatalf("Expected fostered span then table on parser reuse, got %#v", div.Children)
			}
			continue
		}
		assertTableNode(t, table, "table", "x", 0, 16)
		assertTableNode(t, tableElements(table, "tbody")[0], "tbody", "x", 0, 0)
		assertTableNode(t, tableElements(table, "tr")[0], "tr", "x", 7, 16)
		assertTableNode(t, tableElements(table, "td")[0], "td", "x", 11, 16)
	}
}

func TestParseTableAdditionalInsertionModeBoundaries(t *testing.T) {
	t.Run("mismatched structural ends ignored in cell", func(t *testing.T) {
		for _, content := range []string{`<table><tr><td>a</th>b</table>`, `<table><tbody><tr><td>a</thead>b</table>`} {
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			cell := tableElements(document, "td")[0]
			if cell.TextContent != "ab" {
				t.Fatalf("Expected mismatched end ignored inside td, got %#v", cell)
			}
			if cell.EndPos != len(content)-len(`</table>`) {
				t.Fatalf("Expected td to end at table token start, got %#v", cell)
			}
		}
	})
	t.Run("direct nested table becomes sibling", func(t *testing.T) {
		const content = `<table><table><tr><td>x</table>tail`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		tables := tableElements(document, "table")
		if len(tables) != 2 || tables[0].Parent != parsedBody(document) || tables[1].Parent != parsedBody(document) {
			t.Fatalf("Expected sibling tables, got %#v", tables)
		}
		assertTableNode(t, tables[0], "table", "", 0, 7)
		assertTableNode(t, tables[1], "table", "x", 7, 31)
	})
	t.Run("caption and colgroup transitions", func(t *testing.T) {
		captionDoc, err := NewHTMLParser().Parse(`<table><caption>c<tr><td>x</table>`)
		if err != nil {
			t.Fatal(err)
		}
		caption := tableElements(captionDoc, "caption")[0]
		assertTableNode(t, caption, "caption", "c", 7, 17)
		colDoc, err := NewHTMLParser().Parse(`<table><col><tr><td>x</table>`)
		if err != nil {
			t.Fatal(err)
		}
		colgroup := tableElements(colDoc, "colgroup")[0]
		assertTableNode(t, colgroup, "colgroup", "", 0, 0)
		assertTableNode(t, tableElements(colgroup, "col")[0], "col", "", 7, 12)
	})
	t.Run("explicit and nested EOF propagation", func(t *testing.T) {
		for _, tc := range []struct {
			content string
			starts  []int
			names   []string
		}{
			{`<div><table><tbody><tr><td>x`, []int{0, 5, 12, 19, 23}, []string{"div", "table", "tbody", "tr", "td"}},
			{`<section><div><table><tbody><tr><td>x`, []int{0, 9, 14, 21, 28, 32}, []string{"section", "div", "table", "tbody", "tr", "td"}},
		} {
			document, err := NewHTMLParser().Parse(tc.content)
			if err != nil {
				t.Fatal(err)
			}
			for i, name := range tc.names {
				assertTableNode(t, tableElements(document, name)[0], name, "x", tc.starts[i], len(tc.content))
			}
		}
	})
}

func TestParseTableFosterParentingScalingIsBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping table scaling regression in short mode")
	}
	build := func(count int) string {
		return `<div><table>` + strings.Repeat(`<span>x</span>`, count) + `<tr><td>y</td></tr></table></div>`
	}
	measure := func(count int) time.Duration {
		content := build(count)
		best := time.Duration(1<<63 - 1)
		for i := 0; i < 3; i++ {
			started := time.Now()
			if _, err := NewHTMLParser().Parse(content); err != nil {
				t.Fatalf("Parse %d foster tokens: %v", count, err)
			}
			if elapsed := time.Since(started); elapsed < best {
				best = elapsed
			}
		}
		return best
	}
	small, large := measure(300), measure(1200)
	if small > 0 && large > small*14 {
		t.Fatalf("Table foster-parenting scaled superlinearly: 300=%v 1200=%v ratio=%.1fx", small, large, float64(large)/float64(small))
	}
}
