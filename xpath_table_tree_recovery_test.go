package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func assertTableQuery(t *testing.T, document, expression, text string, start, end int) {
	t.Helper()
	expression = documentBodyExpression(expression)
	results, err := xpath.Query(expression, document)
	if err != nil {
		t.Fatalf("Query %s returned %v", expression, err)
	}
	if len(results) != 1 || results[0].TextContent != text || results[0].StartLocation != start || results[0].EndLocation != end {
		t.Fatalf("Expected %s text %q at %d:%d, got %#v", expression, text, start, end, results)
	}
}

func TestQueryTableImplicitSectionsRowsAndCells(t *testing.T) {
	const tbodyDocument = `<table><tr><td>x</td></tr></table>`
	assertTableQuery(t, tbodyDocument, `//table/tbody`, "x", 0, 0)
	assertTableQuery(t, tbodyDocument, `//table/tbody/tr`, "x", 7, 26)
	assertTableQuery(t, tbodyDocument, `//table/tbody/tr/td`, "x", 11, 21)

	const rowDocument = `<table><td>x</td><th>y</th></table>`
	assertTableQuery(t, rowDocument, `//table/tbody`, "xy", 0, 0)
	assertTableQuery(t, rowDocument, `//table/tbody/tr`, "xy", 0, 0)
	assertTableQuery(t, rowDocument, `//table/tbody/tr/td`, "x", 7, 17)
	assertTableQuery(t, rowDocument, `//table/tbody/tr/th`, "y", 17, 27)

	const explicitBody = `<table><tbody><td>x</tbody></table>`
	assertTableQuery(t, explicitBody, `//table/tbody`, "x", 7, 27)
	assertTableQuery(t, explicitBody, `//table/tbody/tr`, "x", 0, 0)
	assertTableQuery(t, explicitBody, `//table/tbody/tr/td`, "x", 14, 19)
}

func TestQueryTableAutoCloseAndSectionTransitions(t *testing.T) {
	const autoClose = `<table><tr><td>a<td>b<tr><th>c</table>tail`
	assertTableQuery(t, autoClose, `//table`, "abc", 0, 38)
	assertTableQuery(t, autoClose, `//table/tbody/tr[1]`, "ab", 7, 21)
	assertTableQuery(t, autoClose, `//table/tbody/tr[1]/td[1]`, "a", 11, 16)
	assertTableQuery(t, autoClose, `//table/tbody/tr[1]/td[2]`, "b", 16, 21)
	assertTableQuery(t, autoClose, `//table/tbody/tr[2]/th`, "c", 25, 30)
	assertTableQuery(t, autoClose, `//table/following-sibling::text()`, "tail", 38, 42)

	const sections = `<table><thead><tr><td>h<tbody><tr><td>b<tfoot><tr><td>f</table>`
	assertTableQuery(t, sections, `//table/thead`, "h", 7, 23)
	assertTableQuery(t, sections, `//table/tbody`, "b", 23, 39)
	assertTableQuery(t, sections, `//table/tfoot`, "f", 39, 55)

	const transition = `<table><tbody><tr><td>a</tbody><tr><td>b</table>`
	assertTableQuery(t, transition, `//table/tbody[1]`, "a", 7, 31)
	assertTableQuery(t, transition, `//table/tbody[2]`, "b", 0, 0)
	assertTableQuery(t, transition, `//table/tbody[2]/tr`, "b", 31, 40)
}

func TestQueryTableFosterParentingAndSourceRanges(t *testing.T) {
	const textDocument = `before<table>alpha<tr><td>x</td></tr>omega</table>after`
	assertTableQuery(t, textDocument, `//table/preceding-sibling::text()`, "beforealphaomega", 0, 42)
	assertTableQuery(t, textDocument, `//table`, "x", 6, 50)
	assertTableQuery(t, textDocument, `//table/following-sibling::text()`, "after", 50, 55)

	const elementDocument = `<div><table><span>a</span><tr><td>x</td></tr><b>b</b></table>z</div>`
	assertTableQuery(t, elementDocument, `//div/span`, "a", 12, 26)
	assertTableQuery(t, elementDocument, `//div/b`, "b", 45, 53)
	assertTableQuery(t, elementDocument, `//div/table`, "x", 5, 61)
	assertTableQuery(t, elementDocument, `//div/table/following-sibling::text()`, "z", 61, 62)

	const multibyte = `<div>é<table>β<tr><td>γ</td></tr>δ</table>ω</div>`
	assertTableQuery(t, multibyte, `//div/table/preceding-sibling::text()`, "éβδ", 5, 38)
	assertTableQuery(t, multibyte, `//div/table`, "γ", 7, 46)
	assertTableQuery(t, multibyte, `//div/table//td`, "γ", 20, 31)
	assertTableQuery(t, multibyte, `//div/table/following-sibling::text()`, "ω", 46, 48)

	const crlf = "<div>A\r\n<table> 🧪B<tr><td>x</td></tr>C</table>Z</div>"
	assertTableQuery(t, crlf, `//div/table/preceding-sibling::text()`, "A\n 🧪BC", 5, 41)
	assertTableQuery(t, crlf, `//div/table`, "x", 8, 49)
	assertTableQuery(t, crlf, `//div/table/following-sibling::text()`, "Z", 49, 50)
}

func TestQueryTableWhitespaceCommentsNestedAndEOF(t *testing.T) {
	const whitespace = "<div><table> \n<!--c--><tr><td>x</td></tr>\t</table></div>"
	assertTableQuery(t, whitespace, `//table/text()[1]`, " \n", 12, 14)
	assertTableQuery(t, whitespace, `//table/tbody`, "x\t", 0, 0)
	assertTableQuery(t, whitespace, `//table/tbody/text()`, "\t", 41, 42)

	const nested = `<table><tr><td>a<table><tr><td>b</table>c</table>tail`
	assertTableQuery(t, nested, `/table`, "abc", 0, 49)
	assertTableQuery(t, nested, `/table//td/table`, "b", 16, 40)
	assertTableQuery(t, nested, `/table/following-sibling::text()`, "tail", 49, 53)

	const eof = `<table><tr><td>x`
	assertTableQuery(t, eof, `//table/tbody`, "x", 0, 0)
	assertTableQuery(t, eof, `//table/tbody/tr`, "x", 7, 16)
	assertTableQuery(t, eof, `//table/tbody/tr/td`, "x", 11, 16)
}

func TestQueryTableAdditionalInsertionModeBoundaries(t *testing.T) {
	assertTableQuery(t, `<table><tr><td>a</th>b</table>`, `//td`, "ab", 11, 22)
	assertTableQuery(t, `<table><tbody><tr><td>a</thead>b</table>`, `//td`, "ab", 18, 32)
	const direct = `<table><table><tr><td>x</table>tail`
	assertTableQuery(t, direct, `//table[1]`, "", 0, 7)
	assertTableQuery(t, direct, `//table[2]`, "x", 7, 31)
	assertTableQuery(t, direct, `//table[2]/following-sibling::text()`, "tail", 31, 35)
	const caption = `<table><caption>c<tr><td>x</table>`
	assertTableQuery(t, caption, `//table/caption`, "c", 7, 17)
	const col = `<table><col><tr><td>x</table>`
	assertTableQuery(t, col, `//table/colgroup`, "", 0, 0)
	assertTableQuery(t, col, `//table/colgroup/col`, "", 7, 12)
	const eof = `<section><div><table><tbody><tr><td>x`
	assertTableQuery(t, eof, `//section`, "x", 0, 37)
	assertTableQuery(t, eof, `//section/div/table/tbody/tr/td`, "x", 32, 37)
}

func TestQueryTableTextTokenizationAndFosterOrder(t *testing.T) {
	for _, tc := range []struct {
		document, value                                     string
		tableStart, tableEnd, textStart, textEnd, tailStart int
	}{
		{`<table>x<tr><td>y</table>z`, "x", 0, 25, 7, 8, 25},
		{`<div><table>x<tr><td>y</table>z</div>`, "x", 5, 30, 12, 13, 30},
		{`<table>&amp;<tr><td>y</table>z`, "&", 0, 29, 7, 12, 29},
	} {
		assertTableQuery(t, tc.document, `//table/preceding-sibling::text()`, tc.value, tc.textStart, tc.textEnd)
		assertTableQuery(t, tc.document, `//table`, "y", tc.tableStart, tc.tableEnd)
		assertTableQuery(t, tc.document, `//table/following-sibling::text()`, "z", tc.tailStart, tc.tailStart+1)
	}
	const space = `<table>&#32;<tr><td>y</table>`
	assertTableQuery(t, space, `//table/text()`, " ", 7, 12)
	assertTableQuery(t, space, `//table`, " y", 0, 29)
	const nul = "<table>\x00<tr><td>y</table>"
	assertTableQuery(t, nul, `//table`, "y", 0, 25)
	results, err := xpath.Query(`//table/text()`, nul)
	if err != nil || len(results) != 0 {
		t.Fatalf("Expected table-context NUL to produce no text node, got %#v err=%v", results, err)
	}
}

func TestQueryTableFosteredContentContributesToAncestorText(t *testing.T) {
	const textDocument = `<div>p<table>x<tr><td>y</table>z</div>`
	assertTableQuery(t, textDocument, `//div`, "pxyz", 0, 38)
	assertTableQuery(t, textDocument, `//div/table`, "y", 6, 31)
	const elementDocument = `<div>p<table><span>x</span><tr><td>y</table>z</div>`
	assertTableQuery(t, elementDocument, `//div`, "pxyz", 0, 51)
	assertTableQuery(t, elementDocument, `//div/span`, "x", 13, 27)
	assertTableQuery(t, elementDocument, `//div/table`, "y", 6, 44)
}

func TestQueryTableColCaptionAndInvalidEndRecovery(t *testing.T) {
	const explicit = `<table><colgroup><col></colgroup><tr><td>x</table>`
	assertTableQuery(t, explicit, `//table/colgroup`, "", 7, 33)
	assertTableQuery(t, explicit, `//table/colgroup/col`, "", 17, 22)
	assertTableQuery(t, `<table><tbody><tr><td>x<caption>c</caption><tr><td>y</table>`, `//table/caption`, "c", 23, 43)
	assertTableQuery(t, `<table><tbody><tr><td>x<col><tr><td>y</table>`, `//table/colgroup/col`, "", 23, 28)
	assertTableQuery(t, `<table><tbody><tr><td>x<colgroup><col><tr><td>y</table>`, `//table/colgroup`, "", 23, 38)
	assertTableQuery(t, `<table><tbody><tr><td>a</tfoot>b</table>`, `//td`, "ab", 18, 32)
	assertTableQuery(t, `<table><tbody></tr><tr><td>x</table>`, `//tr`, "x", 19, 28)
	assertTableQuery(t, `<table></tbody><tr><td>x</table>`, `//tr`, "x", 15, 24)
}

func TestQueryUnclosedFosterElementReentersTableMode(t *testing.T) {
	const document = `<table><div>a<tr><td>x</table>z`
	assertTableQuery(t, document, `//div`, "a", 7, 13)
	assertTableQuery(t, document, `//div/following-sibling::table`, "x", 0, 30)
	assertTableQuery(t, document, `//table/following-sibling::text()`, "z", 30, 31)
}

func TestQueryTableIncompleteEOFTokensAreDiscarded(t *testing.T) {
	for _, tc := range []struct {
		document, expression string
		start, end           int
	}{
		{`<table><tbody x`, `//table`, 0, 15}, {`<table><tbody><tr x`, `//tbody`, 7, 19}, {`<table><tbody><tr><td x`, `//tr`, 14, 23}, {`<table><caption x`, `//table`, 0, 17}, {`<table><col x`, `//table`, 0, 13},
	} {
		assertTableQuery(t, tc.document, tc.expression, "", tc.start, tc.end)
	}
	for _, document := range []string{`<table><tbody><tr><td>x</td`, `<table><tbody><tr><td>x</tr`, `<table><tbody><tr><td>x</table`} {
		assertTableQuery(t, document, `//td`, "x", 18, len(document))
		assertTableQuery(t, document, `//td/text()`, "x", 22, len(document))
	}
	results, err := xpath.Query(`//colgroup`, `<table><col x`)
	if err != nil || len(results) != 0 {
		t.Fatalf("Expected no phantom colgroup, got %#v err=%v", results, err)
	}
}

func TestQueryDirectIncompleteTagsAndSyntheticWrapperLocations(t *testing.T) {
	for _, document := range []string{`<table><tr x`, `<table><td x`} {
		assertTableQuery(t, document, `//table`, "", 0, 12)
		for _, expression := range []string{`//tbody`, `//tr`} {
			results, err := xpath.Query(expression, document)
			if err != nil || len(results) != 0 {
				t.Fatalf("Expected no phantom wrappers for %q %s, got %#v err=%v", document, expression, results, err)
			}
		}
	}
	assertTableQuery(t, `<table><tr><td>x</tbody></table>`, `//tbody`, "x", 0, 0)
	assertTableQuery(t, `<table><td>x</tr></table>`, `//tbody/tr`, "x", 0, 0)
}

func TestQueryDiscardedTableEOFTokensExtendPriorTextRawRange(t *testing.T) {
	for _, tc := range []struct {
		document, expression, value string
		start                       int
	}{
		{`<table> </td`, `//table/text()`, ` `, 7}, {`<table><tbody> </tr`, `//tbody/text()`, ` `, 14}, {`<table><tr> </td`, `//tr/text()`, ` `, 11}, {`<table><tr><td>x</caption`, `//td/text()`, `x`, 15}, {`<table> <tr x`, `//table/text()`, ` `, 7},
	} {
		assertTableQuery(t, tc.document, tc.expression, tc.value, tc.start, len(tc.document))
	}
	for _, expression := range []string{`//tbody`, `//tr`} {
		results, err := xpath.Query(expression, `<table> <tr x`)
		if err != nil || len(results) != 0 {
			t.Fatalf("Expected no phantom %s, got %#v err=%v", expression, results, err)
		}
	}
}

func TestQueryStyleAndScriptRemainAllowedTableChildren(t *testing.T) {
	assertTableQuery(t, `<table><style>a{}</style><tr><td>x</table>`, `//table/style`, "a{}", 7, 25)
	assertTableQuery(t, `<table><script>if(a<b)c()</script><tr><td>x</table>`, `//table/script`, "if(a<b)c()", 7, 34)
}

func TestQueryTableInvalidEndsCaptionColsAndDescendantUnwind(t *testing.T) {
	for _, tc := range []struct {
		document, expression, text string
		start, end                 int
	}{
		{`<table></tr></td></tbody><tr><td>x</table>`, `//tr`, "x", 25, 34},
		{`<table><tbody></tr><tr><td>x</table>`, `//tr`, "x", 19, 28},
		{`<table><tbody><tr></td><td>x</table>`, `//tr`, "x", 14, 28},
		{`<table><caption>a</tr>b</caption></table>`, `//caption`, "ab", 7, 33},
		{`<table><tr><td>a</caption></colgroup>b</table>`, `//td`, "ab", 11, 38},
		{`<table><caption>x</table>`, `//caption`, "x", 7, 17},
		{`<table><tr><td><div>x</td><td>y</table>`, `//td[1]/div`, "x", 15, 21},
		{`<table><tr><td><p>x</tr><tr><td>y</table>`, `//tr[1]/td/p`, "x", 15, 19},
		{`<table><tr><td><div>x</table>z`, `//td/div`, "x", 15, 21},
	} {
		assertTableQuery(t, tc.document, tc.expression, tc.text, tc.start, tc.end)
	}
	const cols = "<table><col> \n<!--c--><col><tr><td>x</table>"
	assertTableQuery(t, cols, `//colgroup`, " \n", 0, 0)
	results, err := xpath.Query(`//colgroup/col`, cols)
	if err != nil || len(results) != 2 {
		t.Fatalf("Expected two cols in one group, got %#v err=%v", results, err)
	}
	const foster = `<table><span>a<tr><td>x</table>z`
	assertTableQuery(t, foster, `//span`, "a", 7, 14)
	assertTableQuery(t, foster, `//span/following-sibling::table`, "x", 0, 31)
}

func TestQueryColgroupIgnoresColEndAndKeepsFollowingCol(t *testing.T) {
	const document = `<table><colgroup><col></col> <col><tr><td>x</table>`
	assertTableQuery(t, document, `//colgroup`, " ", 7, 34)
	results, err := xpath.Query(`//colgroup/col`, document)
	if err != nil || len(results) != 2 || results[0].StartLocation != 17 || results[1].StartLocation != 29 {
		t.Fatalf("Expected two cols in one explicit group, got %#v err=%v", results, err)
	}
	assertTableQuery(t, document, `//colgroup/text()`, " ", 28, 29)
}

func TestQueryNestedTableStartClosesOuterFromSectionOrRow(t *testing.T) {
	for _, tc := range []struct {
		document                       string
		outerEnd, innerStart, innerEnd int
	}{
		{`<table><tbody><table><tr><td>x</table>z`, 14, 14, 38},
		{`<table><tbody><tr><table><tr><td>x</table>z`, 18, 18, 42},
	} {
		assertTableQuery(t, tc.document, `//table[1]`, "", 0, tc.outerEnd)
		assertTableQuery(t, tc.document, `//table[2]`, "x", tc.innerStart, tc.innerEnd)
	}
}

func TestQueryMatchingTableStructuralEndsUnwindFosteredOpenElement(t *testing.T) {
	for _, tc := range []struct {
		document, bodyText        string
		divStart, divEnd, bodyEnd int
	}{
		{`<table><tbody><div>a</tbody><tr><td>x</table>`, "", 14, 20, 28},
		{`<table><tbody><tr><div>a</tr><tr><td>x</table>`, "x", 18, 24, 38},
		{`<table><tbody><tr><div>a</tbody><tbody><tr><td>x</table>`, "", 18, 24, 32},
	} {
		assertTableQuery(t, tc.document, `//div`, "a", tc.divStart, tc.divEnd)
		assertTableQuery(t, tc.document, `//table/tbody[1]`, tc.bodyText, 7, tc.bodyEnd)
		assertTableQuery(t, tc.document, `//div/following-sibling::table`, "x", 0, len(tc.document))
	}
}

func TestQueryIncompleteCaptionAndColgroupEndTokensCloseAtEOF(t *testing.T) {
	assertTableQuery(t, `<table><caption>x</caption`, `//caption`, "x", 7, 26)
	assertTableQuery(t, `<table><caption>x</caption`, `//caption/text()`, "x", 16, 26)
	assertTableQuery(t, `<table><colgroup><col></colgroup`, `//colgroup`, "", 7, 32)
	assertTableQuery(t, `<table><colgroup><col></colgroup`, `//colgroup/col`, "", 17, 22)
}

func TestQueryNestedTableFlushesFosterTextWithinOuterContext(t *testing.T) {
	const cell = `<div><table><tr><td><table>x<tr><td>y</table>z</table></div>`
	assertTableQuery(t, cell, `//div`, "xyz", 0, 60)
	assertTableQuery(t, cell, `//table[1]//td/table`, "y", 20, 45)
	assertTableQuery(t, cell, `//table[1]//td/table/preceding-sibling::text()`, "x", 27, 28)
	assertTableQuery(t, cell, `//table[1]//td/table/following-sibling::text()`, "z", 45, 46)
	const caption = `<table><caption>a<table>x<tr><td>y</table>z</caption></table>`
	assertTableQuery(t, caption, `//caption`, "axyz", 7, 53)
	assertTableQuery(t, caption, `//caption/table`, "y", 17, 42)
}

func TestQueryColgroupNonspaceTextClosesThenFosters(t *testing.T) {
	const document = `<table><colgroup>x<tr><td>y</table>`
	assertTableQuery(t, document, `//table/preceding-sibling::text()`, "x", 17, 18)
	assertTableQuery(t, document, `//table/colgroup`, "", 7, 17)
	assertTableQuery(t, document, `//table`, "y", 0, 35)
}

func TestQueryIgnoredStructuralEndsKeepFosterElementOpen(t *testing.T) {
	for _, tc := range []struct {
		document         string
		divEnd, rowStart int
	}{
		{`<table><div>a</td>b<tr><td>x</table>`, 19, 19},
		{`<table><div>a</tfoot>b<tr><td>x</table>`, 22, 22},
	} {
		assertTableQuery(t, tc.document, `//div`, "ab", 7, tc.divEnd)
		assertTableQuery(t, tc.document, `//div/following-sibling::table/tbody/tr`, "x", tc.rowStart, tc.rowStart+9)
	}
}

func TestQueryIgnoredCellEndsKeepOpenDescendantWhileMatchingCellEndUnwinds(t *testing.T) {
	for _, tc := range []struct {
		end    string
		divEnd int
	}{{"th", 27}, {"tfoot", 30}, {"caption", 32}, {"html", 29}} {
		document := `<table><tr><td><div>a</` + tc.end + `>b</table>`
		assertTableQuery(t, document, `//td/div`, "ab", 15, tc.divEnd)
	}
	assertTableQuery(t, `<table><tr><td><div>a</td>b</table>`, `//td/div`, "a", 15, 21)
}

func TestQueryCaptionDescendantsRespectCaptionInsertionMode(t *testing.T) {
	assertTableQuery(t, `<table><caption><div>a</tr>b</caption></table>`, `//caption/div`, "ab", 16, 28)
	assertTableQuery(t, `<table><caption><div>a</table>b`, `//caption/div`, "a", 16, 22)
	const row = `<table><caption><div>a<tr><td>x</table>`
	assertTableQuery(t, row, `//caption/div`, "a", 16, 22)
	assertTableQuery(t, row, `//table/tbody/tr`, "x", 22, 31)
}

func TestQueryTableInvalidEndTagIgnoreListsByInsertionMode(t *testing.T) {
	groups := [][]string{{"body", "html"}, {"body", "caption", "col", "colgroup", "html", "td", "th", "tr"}, {"body", "caption", "col", "colgroup", "html", "td", "th"}, {"body", "caption", "col", "colgroup", "html"}}
	for mode, ends := range groups {
		for _, end := range ends {
			var document, expression, want string
			switch mode {
			case 0:
				document = `<table></` + end + `><tr><td>x</table>`
				expression = `//td`
				want = "x"
			case 1:
				document = `<table><tbody></` + end + `><tr><td>x</table>`
				expression = `//tbody`
				want = "x"
			case 2:
				document = `<table><tbody><tr></` + end + `><td>x</table>`
				expression = `//tr`
				want = "x"
			case 3:
				document = `<table><tbody><tr><td>a</` + end + `>b</table>`
				expression = `//td`
				want = "ab"
			}
			results, err := xpath.Query(expression, document)
			if err != nil || len(results) != 1 || results[0].TextContent != want {
				t.Fatalf("mode %d </%s>: expected %s text %q, got %#v err=%v", mode, end, expression, want, results, err)
			}
		}
	}
	for _, end := range []string{"body", "col", "colgroup", "html", "tbody", "td", "tfoot", "th", "thead", "tr"} {
		document := `<table><caption>a</` + end + `>b</caption></table>`
		assertTableQuery(t, document, `//caption`, "ab", 7, len(document)-8)
	}
}

func TestQueryColgroupNonspaceReprocessesThroughTableMode(t *testing.T) {
	const document = `<table><colgroup><div>x</div><tr><td>y</table>`
	assertTableQuery(t, document, `//div`, "x", 17, 29)
	assertTableQuery(t, document, `//div/following-sibling::table`, "y", 0, 46)
	assertTableQuery(t, document, `//table/colgroup`, "", 7, 17)
}
