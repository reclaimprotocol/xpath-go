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

func TestParseTableIncompleteEOFTokensAreDiscarded(t *testing.T) {
	for _, tc := range []struct {
		content string
		present map[string][2]int
		absent  []string
	}{
		{`<table><tbody x`, map[string][2]int{"table": {0, 15}}, []string{"tbody"}},
		{`<table><tbody><tr x`, map[string][2]int{"table": {0, 19}, "tbody": {7, 19}}, []string{"tr"}},
		{`<table><tbody><tr><td x`, map[string][2]int{"table": {0, 23}, "tbody": {7, 23}, "tr": {14, 23}}, []string{"td"}},
		{`<table><caption x`, map[string][2]int{"table": {0, 17}}, []string{"caption"}},
		{`<table><col x`, map[string][2]int{"table": {0, 13}}, []string{"col", "colgroup"}},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatalf("Parse %q: %v", tc.content, err)
		}
		for name, bounds := range tc.present {
			assertTableNode(t, tableElements(document, name)[0], name, "", bounds[0], bounds[1])
		}
		for _, name := range tc.absent {
			if len(tableElements(document, name)) != 0 {
				t.Fatalf("Expected no %s for discarded EOF token in %q", name, tc.content)
			}
		}
	}
	for _, tc := range []struct {
		content string
		end     int
	}{
		{`<table><tbody><tr><td>x</td`, 27}, {`<table><tbody><tr><td>x</tr`, 27}, {`<table><tbody><tr><td>x</table`, 30},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatalf("Parse %q: %v", tc.content, err)
		}
		for _, name := range []string{"table", "tbody", "tr", "td"} {
			node := tableElements(document, name)[0]
			if node.EndPos != tc.end || node.TextContent != "x" {
				t.Fatalf("Expected %s through EOF %d, got %#v", name, tc.end, node)
			}
		}
		text := tableElements(document, "td")[0].Children[0]
		if text.Value != "x" || text.StartPos != 22 || text.EndPos != tc.end {
			t.Fatalf("Expected text raw range to span discarded EOF token, got %#v", text)
		}
	}
}

func TestParseDirectIncompleteRowOrCellCreatesNoSyntheticWrapper(t *testing.T) {
	for _, content := range []string{`<table><tr x`, `<table><td x`} {
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		table := tableElements(document, "table")[0]
		assertTableNode(t, table, "table", "", 0, 12)
		if len(table.Children) != 0 || len(tableElements(document, "tbody")) != 0 || len(tableElements(document, "tr")) != 0 {
			t.Fatalf("Expected no phantom wrappers for %q, got %#v", content, table.Children)
		}
	}
}

func TestParseDiscardedTableEOFTokensExtendPriorTextRawRange(t *testing.T) {
	for _, tc := range []struct {
		content, parent, value string
		textStart              int
	}{
		{`<table> </td`, "table", " ", 7}, {`<table><tbody> </tr`, "tbody", " ", 14}, {`<table><tr> </td`, "tr", " ", 11}, {`<table><tr><td>x</caption`, "td", "x", 15}, {`<table> <tr x`, "table", " ", 7},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatalf("Parse %q: %v", tc.content, err)
		}
		parent := tableElements(document, tc.parent)[0]
		texts := findAllNodesByType(parent, types.TextNode)
		if len(texts) != 1 || texts[0].Value != tc.value || texts[0].StartPos != tc.textStart || texts[0].EndPos != len(tc.content) {
			t.Fatalf("Expected prior %q text raw range %d:%d, got %#v", tc.value, tc.textStart, len(tc.content), texts)
		}
		if tc.content == `<table> <tr x` && (len(tableElements(document, "tbody")) != 0 || len(tableElements(document, "tr")) != 0) {
			t.Fatalf("Expected no phantom wrappers, got %#v", document)
		}
	}
}

func TestParseSyntheticTableWrappersRemainLocationlessWhenStructurallyClosed(t *testing.T) {
	for _, tc := range []struct {
		content                              string
		rowStart, rowEnd, cellStart, cellEnd int
	}{
		{`<table><tr><td>x</tbody></table>`, 7, 16, 11, 16},
		{`<table><td>x</tr></table>`, 0, 0, 7, 12},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatal(err)
		}
		tbody := tableElements(document, "tbody")[0]
		row := tableElements(document, "tr")[0]
		assertTableNode(t, tbody, "tbody", "x", 0, 0)
		assertTableNode(t, row, "tr", "x", tc.rowStart, tc.rowEnd)
		assertTableNode(t, tableElements(document, "td")[0], "td", "x", tc.cellStart, tc.cellEnd)
	}
}

func TestParseStyleAndScriptRemainAllowedTableChildren(t *testing.T) {
	for _, tc := range []struct {
		content, name, text string
		end, rowStart       int
	}{
		{`<table><style>a{}</style><tr><td>x</table>`, "style", "a{}", 25, 25},
		{`<table><script>if(a<b)c()</script><tr><td>x</table>`, "script", "if(a<b)c()", 34, 34},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatal(err)
		}
		table := tableElements(document, "table")[0]
		assertTableNode(t, tableElements(document, tc.name)[0], tc.name, tc.text, 7, tc.end)
		if table.TextContent != tc.text+"x" || len(table.Children) != 2 || table.Children[0].Name != tc.name || table.Children[1].Name != "tbody" {
			t.Fatalf("Expected %s then tbody as table children, got %#v", tc.name, table.Children)
		}
		assertTableNode(t, tableElements(table, "tr")[0], "tr", "x", tc.rowStart, tc.rowStart+9)
	}
}

func TestParseTableInvalidStructuralEndTagsAreIgnoredByMode(t *testing.T) {
	for _, tc := range []struct {
		content, name, text string
		start, end          int
	}{
		{`<table></tr></td></tbody><tr><td>x</table>`, "tr", "x", 25, 34},
		{`<table><tbody></tr><tr><td>x</table>`, "tr", "x", 19, 28},
		{`<table><tbody><tr></td><td>x</table>`, "tr", "x", 14, 28},
		{`<table><caption>a</tr>b</caption></table>`, "caption", "ab", 7, 33},
		{`<table><tr><td>a</caption></colgroup>b</table>`, "td", "ab", 11, 38},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatalf("Parse %q: %v", tc.content, err)
		}
		assertTableNode(t, tableElements(document, tc.name)[0], tc.name, tc.text, tc.start, tc.end)
	}
}

func TestParseTableInvalidEndTagIgnoreListsByInsertionMode(t *testing.T) {
	for _, end := range []string{"body", "html"} {
		content := `<table></` + end + `><tr><td>x</table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("in table </%s>: %v", end, err)
		}
		if tableElements(document, "td")[0].TextContent != "x" {
			t.Fatalf("in table </%s> disturbed cell", end)
		}
	}
	for _, end := range []string{"body", "caption", "col", "colgroup", "html", "td", "th", "tr"} {
		content := `<table><tbody></` + end + `><tr><td>x</table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("in tbody </%s>: %v", end, err)
		}
		if tableElements(document, "tbody")[0].TextContent != "x" {
			t.Fatalf("in tbody </%s> disturbed section", end)
		}
	}
	for _, end := range []string{"body", "caption", "col", "colgroup", "html", "td", "th"} {
		content := `<table><tbody><tr></` + end + `><td>x</table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("in row </%s>: %v", end, err)
		}
		if tableElements(document, "tr")[0].TextContent != "x" {
			t.Fatalf("in row </%s> disturbed row", end)
		}
	}
	for _, end := range []string{"body", "caption", "col", "colgroup", "html"} {
		content := `<table><tbody><tr><td>a</` + end + `>b</table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("in cell </%s>: %v", end, err)
		}
		if tableElements(document, "td")[0].TextContent != "ab" {
			t.Fatalf("in cell </%s> did not retain ab", end)
		}
	}
	for _, end := range []string{"body", "col", "colgroup", "html", "tbody", "td", "tfoot", "th", "thead", "tr"} {
		content := `<table><caption>a</` + end + `>b</caption></table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("in caption </%s>: %v", end, err)
		}
		assertTableNode(t, tableElements(document, "caption")[0], "caption", "ab", 7, len(content)-8)
	}
}

func TestParseColgroupNonspaceReprocessesThroughTableMode(t *testing.T) {
	const content = `<table><colgroup><div>x</div><tr><td>y</table>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsedBodyChildren(document)) != 2 {
		t.Fatalf("Expected fostered div then table, got %#v", parsedBodyChildren(document))
	}
	assertTableNode(t, parsedBodyChildren(document)[0], "div", "x", 17, 29)
	assertTableNode(t, parsedBodyChildren(document)[1], "table", "y", 0, 46)
	assertTableNode(t, tableElements(document, "colgroup")[0], "colgroup", "", 7, 17)
	assertTableNode(t, tableElements(document, "tr")[0], "tr", "y", 29, 38)
}

func TestParseTableCaptionAndSyntheticColgroupClosure(t *testing.T) {
	const captionContent = `<table><caption>x</table>`
	captionDoc, err := NewHTMLParser().Parse(captionContent)
	if err != nil {
		t.Fatal(err)
	}
	assertTableNode(t, tableElements(captionDoc, "caption")[0], "caption", "x", 7, 17)
	const cols = "<table><col> \n<!--c--><col><tr><td>x</table>"
	colDoc, err := NewHTMLParser().Parse(cols)
	if err != nil {
		t.Fatal(err)
	}
	groups := tableElements(colDoc, "colgroup")
	if len(groups) != 1 {
		t.Fatalf("Expected one synthetic colgroup, got %#v", groups)
	}
	group := groups[0]
	assertTableNode(t, group, "colgroup", " \n", 0, 0)
	if len(tableElements(group, "col")) != 2 || len(group.Children) != 4 || group.Children[1].Type != types.TextNode || group.Children[1].Value != " \n" || group.Children[1].StartPos != 12 || group.Children[1].EndPos != 14 || group.Children[2].Type != types.CommentNode || group.Children[2].Value != "c" || group.Children[2].StartPos != 14 || group.Children[2].EndPos != 22 {
		t.Fatalf("Expected cols, whitespace and comment in one group, got %#v", group.Children)
	}
}

func TestParseColgroupIgnoresColEndAndKeepsFollowingCol(t *testing.T) {
	const content = `<table><colgroup><col></col> <col><tr><td>x</table>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	group := tableElements(document, "colgroup")[0]
	assertTableNode(t, group, "colgroup", " ", 7, 34)
	cols := tableElements(group, "col")
	if len(cols) != 2 {
		t.Fatalf("Expected two cols in one group, got %#v", cols)
	}
	assertTableNode(t, cols[0], "col", "", 17, 22)
	assertTableNode(t, cols[1], "col", "", 29, 34)
	if len(group.Children) != 3 || group.Children[1].Type != types.TextNode || group.Children[1].Value != " " || group.Children[1].StartPos != 28 || group.Children[1].EndPos != 29 {
		t.Fatalf("Expected whitespace between cols, got %#v", group.Children)
	}
}

func TestParseTableStructuralCloseUnwindsCellDescendants(t *testing.T) {
	for _, tc := range []struct {
		content, desc                          string
		descStart, descEnd, cellStart, cellEnd int
	}{
		{`<table><tr><td><div>x</td><td>y</table>`, "div", 15, 21, 11, 26},
		{`<table><tr><td><p>x</tr><tr><td>y</table>`, "p", 15, 19, 11, 19},
		{`<table><tr><td><div>x</table>z`, "div", 15, 21, 11, 21},
	} {
		document, err := NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatalf("Parse %q: %v", tc.content, err)
		}
		assertTableNode(t, tableElements(document, tc.desc)[0], tc.desc, "x", tc.descStart, tc.descEnd)
		assertTableNode(t, tableElements(document, "td")[0], "td", "x", tc.cellStart, tc.cellEnd)
	}
	const foster = `<table><span>a<tr><td>x</table>z`
	document, err := NewHTMLParser().Parse(foster)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsedBodyChildren(document)) != 3 {
		t.Fatalf("Expected span, table, z, got %#v", parsedBodyChildren(document))
	}
	assertTableNode(t, parsedBodyChildren(document)[0], "span", "a", 7, 14)
	assertTableNode(t, parsedBodyChildren(document)[1], "table", "x", 0, 31)
}

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
