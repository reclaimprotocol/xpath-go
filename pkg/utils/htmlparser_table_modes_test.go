package utils

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

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
