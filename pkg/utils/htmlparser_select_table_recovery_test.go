package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 13B covers the interaction between current customizable-select tree
// recovery and the table insertion modes. Chrome is authoritative where the
// legacy parse5/jsdom select mode differs. The in-table hidden-input special
// case is intentionally deferred.

func TestParseSelectFosteredFromTableAndRow(t *testing.T) {
	for _, testCase := range []struct {
		content                               string
		selectStart, selectEnd, optionStart   int
		rowStart, rowEnd, cellStart, tableEnd int
	}{
		{content: `<table><select><option>x</select><tr><td>y</table>`, selectStart: 7, selectEnd: 33, optionStart: 15, rowStart: 33, rowEnd: 42, cellStart: 37, tableEnd: 50},
		{content: `<table><tr><select><option>x</select><td>y</table>`, selectStart: 11, selectEnd: 37, optionStart: 19, rowStart: 7, rowEnd: 42, cellStart: 37, tableEnd: 50},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", testCase.content, err)
		}
		if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0].Name != "select" || parsedBodyChildren(document)[1].Name != "table" {
			t.Fatalf("Expected fostered select immediately before table, got %#v", parsedBodyChildren(document))
		}
		selectNode, table := parsedBodyChildren(document)[0], parsedBodyChildren(document)[1]
		option := tableElements(selectNode, "option")[0]
		tbody := tableElements(table, "tbody")[0]
		row := tableElements(tbody, "tr")[0]
		cell := tableElements(row, "td")[0]
		assertTableNode(t, selectNode, "select", "x", testCase.selectStart, testCase.selectEnd)
		assertTableNode(t, option, "option", "x", testCase.optionStart, testCase.selectEnd-9)
		assertTableNode(t, table, "table", "y", 0, testCase.tableEnd)
		assertTableNode(t, tbody, "tbody", "y", 0, 0)
		assertTableNode(t, row, "tr", "y", testCase.rowStart, testCase.rowEnd)
		assertTableNode(t, cell, "td", "y", testCase.cellStart, testCase.rowEnd)
	}
}

func TestParseNestedSelectStartClosesFosteredOuterAndPreservesOrder(t *testing.T) {
	const content = `<div><table><select id=a>x<select id=b>y</select>z<tr><td>c</table></div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	div := tableElements(document, "div")[0]
	selects := tableElements(div, "select")
	tables := tableElements(div, "table")
	assertTableNode(t, div, "div", "xyzc", 0, 73)
	if len(selects) != 1 || selects[0].Attributes["id"] != "a" {
		t.Fatalf("Expected only fostered outer select a after nested start is ignored, got %#v", selects)
	}
	assertTableNode(t, selects[0], "select", "x", 12, 26)
	if len(tables) != 1 {
		t.Fatalf("Expected one table after fostered select/text, got %#v", tables)
	}
	assertTableNode(t, tables[0], "table", "c", 5, 67)
	assertTableNode(t, tableElements(tables[0], "tbody")[0], "tbody", "c", 0, 0)
	assertTableNode(t, tableElements(tables[0], "tr")[0], "tr", "c", 50, 59)
	assertTableNode(t, tableElements(tables[0], "td")[0], "td", "c", 54, 59)
	if len(div.Children) != 3 || div.Children[0] != selects[0] || div.Children[1].Type != types.TextNode || div.Children[1].Value != "yz" || div.Children[1].StartPos != 39 || div.Children[1].EndPos != 50 || div.Children[2] != tables[0] {
		t.Fatalf("Expected select, fostered yz, table ordering, got %#v", div.Children)
	}
}

func TestParseSelectTableFosterOrderAndImplicitTransitions(t *testing.T) {
	const ordered = `<div>a<table>b<select>c</select>d<tr><td>e</table>f</div>`
	document, err := NewHTMLParser().Parse(ordered)
	if err != nil {
		t.Fatal(err)
	}
	div := tableElements(document, "div")[0]
	selectNode := tableElements(div, "select")[0]
	table := tableElements(div, "table")[0]
	assertTableNode(t, div, "div", "abcdef", 0, 57)
	assertTableNode(t, selectNode, "select", "c", 14, 32)
	assertTableNode(t, table, "table", "e", 6, 50)
	if len(div.Children) != 5 || div.Children[0].Value != "ab" || div.Children[0].StartPos != 5 || div.Children[0].EndPos != 14 || div.Children[1] != selectNode || div.Children[2].Value != "d" || div.Children[2].StartPos != 32 || div.Children[2].EndPos != 33 || div.Children[3] != table || div.Children[4].Value != "f" || div.Children[4].StartPos != 50 || div.Children[4].EndPos != 51 {
		t.Fatalf("Expected Chrome foster order ab, select, d, table, f, got %#v", div.Children)
	}
	assertTableNode(t, tableElements(table, "tbody")[0], "tbody", "e", 0, 0)
	assertTableNode(t, tableElements(table, "tr")[0], "tr", "e", 33, 42)
	assertTableNode(t, tableElements(table, "td")[0], "td", "e", 37, 42)

	for _, testCase := range []struct {
		content                                   string
		selectStart, trigger, rowStart, cellStart int
	}{
		{content: `<table><select><option>x<tr><td>y</table>`, selectStart: 7, trigger: 24, rowStart: 24, cellStart: 28},
		{content: `<table><tr><select><option>x<td>y</table>`, selectStart: 11, trigger: 28, rowStart: 7, cellStart: 28},
	} {
		document, err = NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", testCase.content, err)
		}
		selectNode = tableElements(document, "select")[0]
		table = tableElements(document, "table")[0]
		assertTableNode(t, selectNode, "select", "x", testCase.selectStart, testCase.trigger)
		assertTableNode(t, tableElements(selectNode, "option")[0], "option", "x", testCase.selectStart+8, testCase.trigger)
		assertTableNode(t, table, "table", "y", 0, len(testCase.content))
		assertTableNode(t, tableElements(table, "tbody")[0], "tbody", "y", 0, 0)
		assertTableNode(t, tableElements(table, "tr")[0], "tr", "y", testCase.rowStart, 33)
		assertTableNode(t, tableElements(table, "td")[0], "td", "y", testCase.cellStart, 33)
	}
}

func TestParseDirectTableInputAndNestedSelectFosterTransitions(t *testing.T) {
	const input = `<table><select><option>x<input>y<tr><td>z</table>`
	document, err := NewHTMLParser().Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := tableElements(document, "select")[0]
	inputs := tableElements(document, "input")
	table := tableElements(document, "table")[0]
	assertTableNode(t, selectNode, "select", "x", 7, 24)
	assertTableNode(t, tableElements(selectNode, "option")[0], "option", "x", 15, 24)
	if len(inputs) != 1 || inputs[0].StartPos != 24 || inputs[0].EndPos != 31 {
		t.Fatalf("Expected input fostered before table after closing select, got %#v", inputs)
	}
	if len(parsedBodyChildren(document)) != 4 || parsedBodyChildren(document)[0] != selectNode || parsedBodyChildren(document)[1] != inputs[0] || parsedBodyChildren(document)[2].Value != "y" || parsedBodyChildren(document)[2].StartPos != 31 || parsedBodyChildren(document)[2].EndPos != 32 || parsedBodyChildren(document)[3] != table {
		t.Fatalf("Expected select, input, y, table foster order, got %#v", parsedBodyChildren(document))
	}
	assertTableNode(t, table, "table", "z", 0, 49)

	const nested = `<table><select><option>x<select>y<tr><td>z</table>`
	document, err = NewHTMLParser().Parse(nested)
	if err != nil {
		t.Fatal(err)
	}
	selects := tableElements(document, "select")
	if len(selects) != 1 {
		t.Fatalf("Expected nested select start ignored after closing outer, got %#v", selects)
	}
	assertTableNode(t, selects[0], "select", "x", 7, 24)
	table = tableElements(document, "table")[0]
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[0] != selects[0] || parsedBodyChildren(document)[1].Value != "y" || parsedBodyChildren(document)[1].StartPos != 32 || parsedBodyChildren(document)[1].EndPos != 33 || parsedBodyChildren(document)[2] != table {
		t.Fatalf("Expected select, fostered y, table after nested select trigger, got %#v", parsedBodyChildren(document))
	}
	assertTableNode(t, table, "table", "z", 0, 50)
}

func TestParseSelectInsideCellAndTableTransitions(t *testing.T) {
	const normal = `<table><tr><td><select><option>x</select>y</table>`
	document, err := NewHTMLParser().Parse(normal)
	if err != nil {
		t.Fatal(err)
	}
	table := tableElements(document, "table")[0]
	row := tableElements(table, "tr")[0]
	cell := tableElements(row, "td")[0]
	selectNode := tableElements(cell, "select")[0]
	option := tableElements(selectNode, "option")[0]
	assertTableNode(t, table, "table", "xy", 0, 50)
	assertTableNode(t, tableElements(table, "tbody")[0], "tbody", "xy", 0, 0)
	assertTableNode(t, row, "tr", "xy", 7, 42)
	assertTableNode(t, cell, "td", "xy", 11, 42)
	assertTableNode(t, selectNode, "select", "x", 15, 41)
	assertTableNode(t, option, "option", "x", 23, 32)
	if len(cell.Children) != 2 || cell.Children[1].Value != "y" || cell.Children[1].StartPos != 41 || cell.Children[1].EndPos != 42 {
		t.Fatalf("Expected y after normally closed select in cell, got %#v", cell.Children)
	}

	const nextCell = `<table><tr><td><select><option>x<td>y</table>`
	document, err = NewHTMLParser().Parse(nextCell)
	if err != nil {
		t.Fatal(err)
	}
	table = tableElements(document, "table")[0]
	row = tableElements(table, "tr")[0]
	cells := tableElements(row, "td")
	assertTableNode(t, table, "table", "xy", 0, 45)
	assertTableNode(t, row, "tr", "xy", 7, 37)
	assertTableNode(t, cells[0], "td", "x", 11, 32)
	assertTableNode(t, tableElements(cells[0], "select")[0], "select", "x", 15, 32)
	assertTableNode(t, tableElements(cells[0], "option")[0], "option", "x", 23, 32)
	assertTableNode(t, cells[1], "td", "y", 32, 37)

	const nextRow = `<table><tr><td><select><option>x<tr><td>y</table>`
	document, err = NewHTMLParser().Parse(nextRow)
	if err != nil {
		t.Fatal(err)
	}
	rows := tableElements(document, "tr")
	assertTableNode(t, tableElements(document, "table")[0], "table", "xy", 0, 49)
	assertTableNode(t, rows[0], "tr", "x", 7, 32)
	assertTableNode(t, tableElements(rows[0], "td")[0], "td", "x", 11, 32)
	assertTableNode(t, tableElements(rows[0], "select")[0], "select", "x", 15, 32)
	assertTableNode(t, tableElements(rows[0], "option")[0], "option", "x", 23, 32)
	assertTableNode(t, rows[1], "tr", "y", 32, 41)
	assertTableNode(t, tableElements(rows[1], "td")[0], "td", "y", 36, 41)

	const tableEnd = `<table><tr><td><select><option>x</table>tail`
	document, err = NewHTMLParser().Parse(tableEnd)
	if err != nil {
		t.Fatal(err)
	}
	table = tableElements(document, "table")[0]
	assertTableNode(t, table, "table", "x", 0, 40)
	assertTableNode(t, tableElements(table, "tr")[0], "tr", "x", 7, 32)
	assertTableNode(t, tableElements(table, "td")[0], "td", "x", 11, 32)
	assertTableNode(t, tableElements(table, "select")[0], "select", "x", 15, 32)
	assertTableNode(t, tableElements(table, "option")[0], "option", "x", 23, 32)
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].Type != types.TextNode || parsedBodyChildren(document)[1].Value != "tail" || parsedBodyChildren(document)[1].StartPos != 40 || parsedBodyChildren(document)[1].EndPos != 44 {
		t.Fatalf("Expected tail after table transition, got %#v", parsedBodyChildren(document))
	}
}

func TestParseSelectCellExplicitAndSectionTransitions(t *testing.T) {
	const cellEnd = `<table><tr><td><select><option>x</td><td>y</table>`
	document, err := NewHTMLParser().Parse(cellEnd)
	if err != nil {
		t.Fatal(err)
	}
	table := tableElements(document, "table")[0]
	row := tableElements(table, "tr")[0]
	cells := tableElements(row, "td")
	assertTableNode(t, table, "table", "xy", 0, 50)
	assertTableNode(t, row, "tr", "xy", 7, 42)
	assertTableNode(t, cells[0], "td", "x", 11, 37)
	assertTableNode(t, tableElements(cells[0], "select")[0], "select", "x", 15, 32)
	assertTableNode(t, tableElements(cells[0], "option")[0], "option", "x", 23, 32)
	assertTableNode(t, cells[1], "td", "y", 37, 42)

	const rowEnd = `<table><tr><td><select><option>x</tr><tr><td>y</table>`
	document, err = NewHTMLParser().Parse(rowEnd)
	if err != nil {
		t.Fatal(err)
	}
	table = tableElements(document, "table")[0]
	rows := tableElements(table, "tr")
	assertTableNode(t, table, "table", "xy", 0, 54)
	assertTableNode(t, rows[0], "tr", "x", 7, 37)
	assertTableNode(t, tableElements(rows[0], "td")[0], "td", "x", 11, 32)
	assertTableNode(t, tableElements(rows[0], "select")[0], "select", "x", 15, 32)
	assertTableNode(t, rows[1], "tr", "y", 37, 46)
	assertTableNode(t, tableElements(rows[1], "td")[0], "td", "y", 41, 46)

	const section = `<table><tbody><tr><td><select><option>x<tfoot><tr><td>y</table>`
	document, err = NewHTMLParser().Parse(section)
	if err != nil {
		t.Fatal(err)
	}
	table = tableElements(document, "table")[0]
	tbody := tableElements(table, "tbody")[0]
	tfoot := tableElements(table, "tfoot")[0]
	assertTableNode(t, table, "table", "xy", 0, 63)
	assertTableNode(t, tbody, "tbody", "x", 7, 39)
	assertTableNode(t, tableElements(tbody, "tr")[0], "tr", "x", 14, 39)
	assertTableNode(t, tableElements(tbody, "td")[0], "td", "x", 18, 39)
	assertTableNode(t, tableElements(tbody, "select")[0], "select", "x", 22, 39)
	assertTableNode(t, tableElements(tbody, "option")[0], "option", "x", 30, 39)
	assertTableNode(t, tfoot, "tfoot", "y", 39, 55)
	assertTableNode(t, tableElements(tfoot, "tr")[0], "tr", "y", 46, 55)
	assertTableNode(t, tableElements(tfoot, "td")[0], "td", "y", 50, 55)
}

func TestParseSelectInColgroupFostersAndCaptionRetains(t *testing.T) {
	const colgroup = `<table><colgroup><select><option>x</select><tr><td>y</table>`
	document, err := NewHTMLParser().Parse(colgroup)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := tableElements(document, "select")[0]
	table := tableElements(document, "table")[0]
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0] != selectNode || parsedBodyChildren(document)[1] != table {
		t.Fatalf("Expected select fostered before table from colgroup mode, got %#v", parsedBodyChildren(document))
	}
	assertTableNode(t, selectNode, "select", "x", 17, 43)
	assertTableNode(t, tableElements(selectNode, "option")[0], "option", "x", 25, 34)
	assertTableNode(t, table, "table", "y", 0, 60)
	assertTableNode(t, tableElements(table, "colgroup")[0], "colgroup", "", 7, 17)
	assertTableNode(t, tableElements(table, "tbody")[0], "tbody", "y", 0, 0)
	assertTableNode(t, tableElements(table, "tr")[0], "tr", "y", 43, 52)

	const caption = `<table><caption><select><option>x</select>y</caption><tr><td>z</table>`
	document, err = NewHTMLParser().Parse(caption)
	if err != nil {
		t.Fatal(err)
	}
	table = tableElements(document, "table")[0]
	captionNode := tableElements(table, "caption")[0]
	selectNode = tableElements(captionNode, "select")[0]
	assertTableNode(t, table, "table", "xyz", 0, 70)
	assertTableNode(t, captionNode, "caption", "xy", 7, 53)
	assertTableNode(t, selectNode, "select", "x", 16, 42)
	assertTableNode(t, tableElements(selectNode, "option")[0], "option", "x", 24, 33)
	assertTableNode(t, tableElements(table, "tbody")[0], "tbody", "z", 0, 0)
	assertTableNode(t, tableElements(table, "tr")[0], "tr", "z", 53, 62)
	assertTableNode(t, tableElements(table, "td")[0], "td", "z", 57, 62)
}

func TestParseTableRetainedInsideCurrentSelectAndBlocksOuterSelectRules(t *testing.T) {
	const nested = `<select><option>x<table><tr><td>y</table>z</select>`
	document, err := NewHTMLParser().Parse(nested)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := tableElements(document, "select")[0]
	option := tableElements(selectNode, "option")[0]
	table := tableElements(option, "table")[0]
	assertTableNode(t, selectNode, "select", "xyz", 0, 51)
	assertTableNode(t, option, "option", "xyz", 8, 42)
	assertTableNode(t, table, "table", "y", 17, 41)
	assertTableNode(t, tableElements(table, "tbody")[0], "tbody", "y", 0, 0)
	assertTableNode(t, tableElements(table, "tr")[0], "tr", "y", 24, 33)
	assertTableNode(t, tableElements(table, "td")[0], "td", "y", 28, 33)

	const blocked = `<select><option>x<table><tr><td>y</select>z<input>w</table>q</select>`
	document, err = NewHTMLParser().Parse(blocked)
	if err != nil {
		t.Fatal(err)
	}
	selectNode = tableElements(document, "select")[0]
	option = tableElements(selectNode, "option")[0]
	table = tableElements(option, "table")[0]
	cell := tableElements(table, "td")[0]
	inputs := tableElements(cell, "input")
	assertTableNode(t, selectNode, "select", "xyzwq", 0, 69)
	assertTableNode(t, option, "option", "xyzwq", 8, 60)
	assertTableNode(t, table, "table", "yzw", 17, 59)
	assertTableNode(t, cell, "td", "yzw", 28, 51)
	if len(inputs) != 1 || inputs[0].StartPos != 43 || inputs[0].EndPos != 50 || inputs[0].Parent != cell {
		t.Fatalf("Expected input retained in cell because table blocks outer select scope, got %#v", inputs)
	}

	const inputBlocked = `<select><option>x<table><tr><td>y<input>z</table>q</select>`
	document, err = NewHTMLParser().Parse(inputBlocked)
	if err != nil {
		t.Fatal(err)
	}
	selectNode = tableElements(document, "select")[0]
	option = tableElements(selectNode, "option")[0]
	table = tableElements(option, "table")[0]
	cell = tableElements(table, "td")[0]
	inputs = tableElements(cell, "input")
	assertTableNode(t, selectNode, "select", "xyzq", 0, 59)
	assertTableNode(t, option, "option", "xyzq", 8, 50)
	assertTableNode(t, table, "table", "yz", 17, 49)
	assertTableNode(t, cell, "td", "yz", 28, 41)
	if len(inputs) != 1 || inputs[0].StartPos != 33 || inputs[0].EndPos != 40 || inputs[0].Parent != cell {
		t.Fatalf("Expected input to remain in cell behind table scope, got %#v", inputs)
	}
}

func TestParseNestedSelectAllowedBehindNestedTableScope(t *testing.T) {
	const nestedSelect = `<select><option>x<table><tr><td>y<select><option>z</select>w</table>q</select>`
	document, err := NewHTMLParser().Parse(nestedSelect)
	if err != nil {
		t.Fatal(err)
	}
	selects := tableElements(document, "select")
	if len(selects) != 2 {
		t.Fatalf("Expected a nested select inside table scope, got %#v", selects)
	}
	outer, inner := selects[0], selects[1]
	outerOption := tableElements(outer, "option")[0]
	table := tableElements(outerOption, "table")[0]
	cell := tableElements(table, "td")[0]
	assertTableNode(t, outer, "select", "xyzwq", 0, 78)
	assertTableNode(t, outerOption, "option", "xyzwq", 8, 69)
	assertTableNode(t, table, "table", "yzw", 17, 68)
	assertTableNode(t, cell, "td", "yzw", 28, 60)
	assertTableNode(t, inner, "select", "z", 33, 59)
	assertTableNode(t, tableElements(inner, "option")[0], "option", "z", 41, 50)
	if inner.Parent != cell {
		t.Fatalf("Expected inner select parent to be td, got %#v", inner.Parent)
	}
}

func TestParseInputDoesNotCloseOuterSelectBehindNestedTableScope(t *testing.T) {
	const content = `<select><option>x<table><tr><td>y<input>z</table>q</select>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := tableElements(document, "select")[0]
	option := tableElements(selectNode, "option")[0]
	table := tableElements(option, "table")[0]
	cell := tableElements(table, "td")[0]
	inputs := tableElements(cell, "input")
	assertTableNode(t, selectNode, "select", "xyzq", 0, 59)
	assertTableNode(t, option, "option", "xyzq", 8, 50)
	assertTableNode(t, table, "table", "yz", 17, 49)
	assertTableNode(t, cell, "td", "yz", 28, 41)
	if len(inputs) != 1 || inputs[0].StartPos != 33 || inputs[0].EndPos != 40 || inputs[0].Parent != cell {
		t.Fatalf("Expected input to remain in cell behind table scope, got %#v", inputs)
	}
}

func TestParseSelectScopeRestoredAfterNestedTableCloses(t *testing.T) {
	const blockedThenRestored = `<table><tr><td><select id=o>a<table></select><tr><td>b</table>c</select></table>`
	document, err := NewHTMLParser().Parse(blockedThenRestored)
	if err != nil {
		t.Fatal(err)
	}
	tables := tableElements(document, "table")
	selects := tableElements(document, "select")
	if len(tables) != 2 || len(selects) != 1 || selects[0].Attributes["id"] != "o" {
		t.Fatalf("Expected one outer select retaining one nested table, got tables=%#v selects=%#v", tables, selects)
	}
	outerTable, innerTable, selectNode := tables[0], tables[1], selects[0]
	assertTableNode(t, outerTable, "table", "abc", 0, 80)
	assertTableNode(t, selectNode, "select", "abc", 15, 72)
	assertTableNode(t, innerTable, "table", "b", 29, 62)
	assertTableNode(t, tableElements(innerTable, "tbody")[0], "tbody", "b", 0, 0)
	assertTableNode(t, tableElements(innerTable, "tr")[0], "tr", "b", 45, 54)
	assertTableNode(t, tableElements(innerTable, "td")[0], "td", "b", 49, 54)
	outerRow := tableElements(outerTable, "tr")[0]
	outerCell := tableElements(outerRow, "td")[0]
	assertTableNode(t, outerRow, "tr", "abc", 7, 72)
	assertTableNode(t, outerCell, "td", "abc", 11, 72)

	const normalAfterInner = `<table><tr><td><select>a<table></table>b</select>c</table>`
	document, err = NewHTMLParser().Parse(normalAfterInner)
	if err != nil {
		t.Fatal(err)
	}
	tables = tableElements(document, "table")
	selectNode = tableElements(document, "select")[0]
	outerTable, innerTable = tables[0], tables[1]
	outerRow = tableElements(outerTable, "tr")[0]
	outerCell = tableElements(outerRow, "td")[0]
	assertTableNode(t, outerTable, "table", "abc", 0, 58)
	assertTableNode(t, outerRow, "tr", "abc", 7, 50)
	assertTableNode(t, outerCell, "td", "abc", 11, 50)
	assertTableNode(t, selectNode, "select", "ab", 15, 49)
	assertTableNode(t, innerTable, "table", "", 24, 39)
	if len(outerCell.Children) != 2 || outerCell.Children[0] != selectNode || outerCell.Children[1].Value != "c" || outerCell.Children[1].StartPos != 49 || outerCell.Children[1].EndPos != 50 {
		t.Fatalf("Expected c outside explicitly closed select after inner table restores scope, got %#v", outerCell.Children)
	}
}

func TestParseNestedTableClosesDirectFosteredSelectButIsRetainedInCell(t *testing.T) {
	const direct = `<div><table><select>a<table><tr><td>b</table>c</select>d</div>`
	document, err := NewHTMLParser().Parse(direct)
	if err != nil {
		t.Fatal(err)
	}
	div := tableElements(document, "div")[0]
	tables := tableElements(div, "table")
	selectNode := tableElements(div, "select")[0]
	assertTableNode(t, div, "div", "abcd", 0, 62)
	assertTableNode(t, selectNode, "select", "a", 12, 21)
	if len(tables) != 2 || len(tableElements(selectNode, "table")) != 0 {
		t.Fatalf("Expected nested table start to close direct fostered select and make sibling tables, got %#v", div.Children)
	}
	assertTableNode(t, tables[0], "table", "", 5, 21)
	assertTableNode(t, tables[1], "table", "b", 21, 45)
	assertTableNode(t, tableElements(tables[1], "tbody")[0], "tbody", "b", 0, 0)
	assertTableNode(t, tableElements(tables[1], "tr")[0], "tr", "b", 28, 37)
	assertTableNode(t, tableElements(tables[1], "td")[0], "td", "b", 32, 37)
	if len(div.Children) != 4 || div.Children[0] != selectNode || div.Children[1] != tables[0] || div.Children[2] != tables[1] || div.Children[3].Value != "cd" || div.Children[3].StartPos != 45 || div.Children[3].EndPos != 56 {
		t.Fatalf("Expected select, empty table, b table, trailing cd order, got %#v", div.Children)
	}
}

func TestParseSelectTableMultibyteCRLFFosterLocations(t *testing.T) {
	const content = "<div>é\r\n<table>β<select>😀</select>γ<tr><td>δ</table>ω</div>"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	div := tableElements(document, "div")[0]
	selectNode := tableElements(div, "select")[0]
	table := tableElements(div, "table")[0]
	assertTableNode(t, div, "div", "é\nβ😀γδω", 0, 67)
	assertTableNode(t, selectNode, "select", "😀", 18, 39)
	assertTableNode(t, table, "table", "δ", 9, 59)
	assertTableNode(t, tableElements(table, "tbody")[0], "tbody", "δ", 0, 0)
	assertTableNode(t, tableElements(table, "tr")[0], "tr", "δ", 41, 51)
	assertTableNode(t, tableElements(table, "td")[0], "td", "δ", 45, 51)
	if len(div.Children) != 5 || div.Children[0].Value != "é\nβ" || div.Children[0].StartPos != 5 || div.Children[0].EndPos != 18 || div.Children[1] != selectNode || div.Children[2].Value != "γ" || div.Children[2].StartPos != 39 || div.Children[2].EndPos != 41 || div.Children[3] != table || div.Children[4].Value != "ω" || div.Children[4].StartPos != 59 || div.Children[4].EndPos != 61 {
		t.Fatalf("Expected multibyte foster order and exact UTF-8 byte locations, got %#v", div.Children)
	}
	text := div.Children[0]
	if text.StartLine != 1 || text.StartColumn != 6 || text.EndLine != 2 || text.EndColumn != 9 || selectNode.StartLine != 2 || selectNode.StartColumn != 9 || selectNode.EndLine != 2 || selectNode.EndColumn != 28 {
		t.Fatalf("Expected parse5 UTF-16 coordinates across CRLF/multibyte content, got text=%#v select=%#v", text, selectNode)
	}
}

func TestParseSelectTableEOFReuseAndScaling(t *testing.T) {
	const eof = `<div><table><tr><td><select><option>x`
	document, err := NewHTMLParser().Parse(eof)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"div", "table", "tr", "td", "select", "option"} {
		node := tableElements(document, name)[0]
		if node.EndPos != len(eof) || node.TextContent != "x" {
			t.Fatalf("Expected %s to close at EOF %d with x, got %#v", name, len(eof), node)
		}
	}
	if tbody := tableElements(document, "tbody")[0]; tbody.StartPos != 0 || tbody.EndPos != 0 {
		t.Fatalf("Expected EOF synthetic tbody to remain locationless, got %#v", tbody)
	}

	parser := NewHTMLParser()
	for _, content := range []string{
		`<table><select><option>x</select><tr><td>y</table>`,
		`<select><option>x<table><tr><td>y</table>z</select>`,
		eof,
	} {
		if _, parseErr := parser.Parse(content); parseErr != nil {
			t.Fatalf("Reused parser failed for %q: %v", content, parseErr)
		}
	}

	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n*45 + 15)
		b.WriteString(`<table>`)
		for i := 0; i < n; i++ {
			b.WriteString(`<tr><td><select><option>x</select>`)
		}
		b.WriteString(`</table>`)
		return b.String()
	}
	measure := func(n int) time.Duration {
		content := build(n)
		_, _ = NewHTMLParser().Parse(content)
		best := time.Duration(1<<63 - 1)
		for i := 0; i < 3; i++ {
			start := time.Now()
			if _, parseErr := NewHTMLParser().Parse(content); parseErr != nil {
				t.Fatal(parseErr)
			}
			if elapsed := time.Since(start); elapsed < best {
				best = elapsed
			}
		}
		return best
	}
	small, large := measure(250), measure(1000)
	ratio := float64(large) / float64(small)
	t.Logf("select/table rows scaling 250=%v 1000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 10*time.Millisecond {
		t.Fatalf("Select/table row recovery scaled superlinearly: 250=%v 1000=%v ratio=%.1fx", small, large, ratio)
	}
}

func TestParseSelectBlockedByNestedTableScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n*15 + 70)
		b.WriteString(`<select><option>x<table><tr><td>y`)
		for i := 0; i < n; i++ {
			b.WriteString(`<span>`)
		}
		b.WriteByte('z')
		for i := 0; i < n; i++ {
			b.WriteString(`</select>`)
		}
		b.WriteString(`</table>q</select>`)
		return b.String()
	}
	measure := func(n int) time.Duration {
		content := build(n)
		if _, err := NewHTMLParser().Parse(content); err != nil {
			t.Fatalf("Warmup parse failed for %d blockers: %v", n, err)
		}
		best := time.Duration(1<<63 - 1)
		for i := 0; i < 3; i++ {
			start := time.Now()
			if _, err := NewHTMLParser().Parse(content); err != nil {
				t.Fatalf("Timed parse failed for %d blockers: %v", n, err)
			}
			if elapsed := time.Since(start); elapsed < best {
				best = elapsed
			}
		}
		return best
	}
	small, large := measure(2000), measure(8000)
	ratio := float64(large) / float64(small)
	t.Logf("select blocked-by-table scaling 2000=%v 8000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 50*time.Millisecond {
		t.Fatalf("Select table-boundary checks scaled superlinearly: 2000=%v 8000=%v ratio=%.1fx", small, large, ratio)
	}
}
