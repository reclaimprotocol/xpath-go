package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 14E covers adoption-agency integration with table tree construction.
// Current Chrome is authoritative for foster ordering. Template, select,
// foreign content, fragments, and scripting remain deferred.

func requireTableAdoptionElements(t *testing.T, node *types.Node, name string, count int) []*types.Node {
	t.Helper()
	elements := formattingElements(node, name)
	if len(elements) != count {
		t.Fatalf("Expected %d <%s> elements, got %d in %#v", count, name, len(elements), node)
	}
	return elements
}

func TestParseTableAdoptionInsideCell(t *testing.T) {
	const content = `<table><tr><td><b>1<div>2</b>3</div>4</td></tr></table>5`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	tbody := requireTableAdoptionElements(t, table, "tbody", 1)[0]
	row := requireTableAdoptionElements(t, table, "tr", 1)[0]
	cell := requireTableAdoptionElements(t, table, "td", 1)[0]
	bold := requireTableAdoptionElements(t, table, "b", 2)
	div := requireTableAdoptionElements(t, table, "div", 1)[0]
	assertFormattingNode(t, table, "table", "1234", 0, 55)
	assertFormattingNode(t, tbody, "tbody", "1234", 0, 0)
	assertFormattingNode(t, row, "tr", "1234", 7, 47)
	assertFormattingNode(t, cell, "td", "1234", 11, 42)
	assertFormattingNode(t, bold[0], "b", "1", 15, 29)
	assertFormattingNode(t, div, "div", "23", 19, 36)
	assertFormattingNode(t, bold[1], "b", "2", 0, 0)
	if len(bold) != 2 || bold[0].Parent != cell || bold[1].Parent != div || div.Parent != cell {
		t.Fatalf("Cell adoption produced wrong parents: b=%#v div=%#v", bold, div)
	}
	if div.Children[1].Value != "3" || div.Children[1].StartPos != 29 || div.Children[1].EndPos != 30 || cell.Children[2].Value != "4" || cell.Children[2].StartPos != 36 || cell.Children[2].EndPos != 37 || parsedBodyChildren(document)[1].Value != "5" || parsedBodyChildren(document)[1].StartPos != 55 || parsedBodyChildren(document)[1].EndPos != 56 {
		t.Fatalf("Wrong cell adoption continuation ranges: cell=%#v div=%#v doc=%#v", cell.Children, div.Children, parsedBodyChildren(document))
	}
}

func TestParseTableAdoptionCanonicalMisnestingInsideCell(t *testing.T) {
	const content = `<table><tr><td><b>1<i>2</b>3</i>4</td></tr></table>5`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	cell := requireTableAdoptionElements(t, table, "td", 1)[0]
	bold := requireTableAdoptionElements(t, cell, "b", 1)[0]
	italics := requireTableAdoptionElements(t, cell, "i", 2)
	assertFormattingNode(t, table, "table", "1234", 0, 51)
	assertFormattingNode(t, cell, "td", "1234", 11, 38)
	assertFormattingNode(t, bold, "b", "12", 15, 27)
	assertFormattingNode(t, italics[0], "i", "2", 19, 23)
	assertFormattingNode(t, italics[1], "i", "3", 19, 32)
	if italics[0].Parent != bold || italics[1].Parent != cell || cell.Children[2].Value != "4" || cell.Children[2].StartPos != 32 || cell.Children[2].EndPos != 33 || parsedBodyChildren(document)[1].Value != "5" || parsedBodyChildren(document)[1].StartPos != 51 || parsedBodyChildren(document)[1].EndPos != 52 {
		t.Fatalf("Wrong canonical cell adoption tree: cell=%#v doc=%#v", cell.Children, parsedBodyChildren(document))
	}
}

func TestParseTableCellMarkerIsolatesAdoptionEnd(t *testing.T) {
	const content = `<table><tr><td><b>1<td>2</b>3</table>4`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	tbody := requireTableAdoptionElements(t, table, "tbody", 1)[0]
	row := requireTableAdoptionElements(t, table, "tr", 1)[0]
	cells := requireTableAdoptionElements(t, table, "td", 2)
	bold := requireTableAdoptionElements(t, table, "b", 1)
	assertFormattingNode(t, table, "table", "123", 0, 37)
	assertFormattingNode(t, tbody, "tbody", "123", 0, 0)
	assertFormattingNode(t, row, "tr", "123", 7, 29)
	assertFormattingNode(t, cells[0], "td", "1", 11, 19)
	assertFormattingNode(t, bold[0], "b", "1", 15, 19)
	assertFormattingNode(t, cells[1], "td", "23", 19, 29)
	if len(cells) != 2 || len(bold) != 1 || bold[0].Parent != cells[0] || len(cells[1].Children) != 1 || cells[1].Children[0].Type != types.TextNode || cells[1].Children[0].Value != "23" || cells[1].Children[0].StartPos != 23 || cells[1].Children[0].EndPos != 29 {
		t.Fatalf("Cell marker did not isolate absent b end: cells=%#v b=%#v", cells, bold)
	}
	if parsedBodyChildren(document)[1].Value != "4" || parsedBodyChildren(document)[1].StartPos != 37 || parsedBodyChildren(document)[1].EndPos != 38 {
		t.Fatalf("Wrong continuation after marker-isolated end: %#v", parsedBodyChildren(document))
	}
}

func TestParseTableAdoptionInsideCaption(t *testing.T) {
	const content = `<table><caption><b>1<div>2</b>3</div>4</caption><tr><td>x</table>5`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	caption := requireTableAdoptionElements(t, table, "caption", 1)[0]
	tbody := requireTableAdoptionElements(t, table, "tbody", 1)[0]
	row := requireTableAdoptionElements(t, table, "tr", 1)[0]
	cell := requireTableAdoptionElements(t, table, "td", 1)[0]
	bold := requireTableAdoptionElements(t, table, "b", 2)
	div := requireTableAdoptionElements(t, table, "div", 1)[0]
	assertFormattingNode(t, table, "table", "1234x", 0, 65)
	assertFormattingNode(t, caption, "caption", "1234", 7, 48)
	assertFormattingNode(t, bold[0], "b", "1", 16, 30)
	assertFormattingNode(t, div, "div", "23", 20, 37)
	assertFormattingNode(t, bold[1], "b", "2", 0, 0)
	assertFormattingNode(t, tbody, "tbody", "x", 0, 0)
	assertFormattingNode(t, row, "tr", "x", 48, 57)
	assertFormattingNode(t, cell, "td", "x", 52, 57)
	if len(bold) != 2 || bold[0].Parent != caption || bold[1].Parent != div || div.Parent != caption || tbody.Parent != table {
		t.Fatalf("Caption adoption produced wrong parents: b=%#v div=%#v tbody=%#v", bold, div, tbody)
	}
	if div.Children[1].Value != "3" || div.Children[1].StartPos != 30 || div.Children[1].EndPos != 31 || caption.Children[2].Value != "4" || caption.Children[2].StartPos != 37 || caption.Children[2].EndPos != 38 || parsedBodyChildren(document)[1].Value != "5" || parsedBodyChildren(document)[1].StartPos != 65 || parsedBodyChildren(document)[1].EndPos != 66 {
		t.Fatalf("Wrong caption adoption continuation ranges: caption=%#v div=%#v doc=%#v", caption.Children, div.Children, parsedBodyChildren(document))
	}
}

func TestParseTableAdoptionCanonicalMisnestingInsideCaption(t *testing.T) {
	const content = `<table><caption><b>1<i>2</b>3</i>4</caption><tr><td>x</table>5`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	caption := requireTableAdoptionElements(t, table, "caption", 1)[0]
	bold := requireTableAdoptionElements(t, caption, "b", 1)[0]
	italics := requireTableAdoptionElements(t, caption, "i", 2)
	cell := requireTableAdoptionElements(t, table, "td", 1)[0]
	assertFormattingNode(t, table, "table", "1234x", 0, 61)
	assertFormattingNode(t, caption, "caption", "1234", 7, 44)
	assertFormattingNode(t, bold, "b", "12", 16, 28)
	assertFormattingNode(t, italics[0], "i", "2", 20, 24)
	assertFormattingNode(t, italics[1], "i", "3", 20, 33)
	assertFormattingNode(t, cell, "td", "x", 48, 53)
	if italics[1].Parent != caption || caption.Children[2].Value != "4" || parsedBodyChildren(document)[1].Value != "5" {
		t.Fatalf("Wrong canonical caption adoption tree: caption=%#v doc=%#v", caption.Children, parsedBodyChildren(document))
	}
}

func TestParseTableFosteredAdoptionWithFurthestBlock(t *testing.T) {
	const content = `<table><b>x<div>y</b>z</div><tr><td>q</table>r`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := requireTableAdoptionElements(t, document, "b", 2)
	div := requireTableAdoptionElements(t, document, "div", 1)[0]
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	cell := requireTableAdoptionElements(t, table, "td", 1)[0]
	if len(parsedBodyChildren(document)) != 4 || parsedBodyChildren(document)[0] != bold[0] || parsedBodyChildren(document)[1] != div || parsedBodyChildren(document)[2] != table {
		t.Fatalf("Fostered adoption order must be b, div, table, tail: %#v", parsedBodyChildren(document))
	}
	assertFormattingNode(t, bold[0], "b", "x", 7, 21)
	assertFormattingNode(t, div, "div", "yz", 11, 28)
	assertFormattingNode(t, bold[1], "b", "y", 0, 0)
	assertFormattingNode(t, table, "table", "q", 0, 45)
	assertFormattingNode(t, cell, "td", "q", 32, 37)
	if bold[1].Parent != div || div.Children[1].Value != "z" || div.Children[1].StartPos != 21 || div.Children[1].EndPos != 22 || parsedBodyChildren(document)[3].Value != "r" || parsedBodyChildren(document)[3].StartPos != 45 || parsedBodyChildren(document)[3].EndPos != 46 {
		t.Fatalf("Wrong fostered adoption ownership/ranges: div=%#v doc=%#v", div.Children, parsedBodyChildren(document))
	}
}

func TestParseTablePreMarkerFormattingSurvivesWithoutEnteringCell(t *testing.T) {
	const content = `<table><b>A<tr><td>B</td></tr>C</table>D</b>E`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := requireTableAdoptionElements(t, document, "b", 3)
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	cell := requireTableAdoptionElements(t, table, "td", 1)[0]
	if len(parsedBodyChildren(document)) != 5 || parsedBodyChildren(document)[0] != bold[0] || parsedBodyChildren(document)[1] != bold[1] || parsedBodyChildren(document)[2] != table || parsedBodyChildren(document)[3] != bold[2] {
		t.Fatalf("Pre-marker b foster order must be b(A), b(C), table, b(D), E: %#v", parsedBodyChildren(document))
	}
	assertFormattingNode(t, bold[0], "b", "A", 7, 11)
	assertFormattingNode(t, bold[1], "b", "C", 7, 31)
	assertFormattingNode(t, table, "table", "B", 0, 39)
	assertFormattingNode(t, cell, "td", "B", 15, 25)
	assertFormattingNode(t, bold[2], "b", "D", 7, 44)
	if len(formattingElements(cell, "b")) != 0 || parsedBodyChildren(document)[4].Value != "E" || parsedBodyChildren(document)[4].StartPos != 44 || parsedBodyChildren(document)[4].EndPos != 45 {
		t.Fatalf("Cell marker leaked preexisting b into cell or continuation: cell=%#v doc=%#v", cell.Children, parsedBodyChildren(document))
	}
}

func TestParseTablePreMarkerAnchorAllowsIndependentCellAnchor(t *testing.T) {
	const content = `<table><a id=old>A<tr><td><a id=new>B</a>C</td></tr>D</table>E`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	anchors := requireTableAdoptionElements(t, document, "a", 4)
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	cell := requireTableAdoptionElements(t, table, "td", 1)[0]
	if len(parsedBodyChildren(document)) != 4 || parsedBodyChildren(document)[0] != anchors[0] || parsedBodyChildren(document)[1] != anchors[1] || parsedBodyChildren(document)[2] != table || parsedBodyChildren(document)[3] != anchors[3] {
		t.Fatalf("Pre-marker anchor foster order mismatch: %#v", parsedBodyChildren(document))
	}
	assertFormattingNode(t, anchors[0], "a", "A", 7, 18)
	assertFormattingNode(t, anchors[1], "a", "D", 7, 53)
	assertFormattingNode(t, anchors[2], "a", "B", 26, 41)
	assertFormattingNode(t, anchors[3], "a", "E", 7, 62)
	if anchors[0].Attributes["id"] != "old" || anchors[1].Attributes["id"] != "old" || anchors[2].Attributes["id"] != "new" || anchors[3].Attributes["id"] != "old" || anchors[2].Parent != cell || len(formattingElements(anchors[2], "a")) != 1 {
		t.Fatalf("Cell anchor start confused pre-marker old anchor identity: %#v", anchors)
	}
	if cell.TextContent != "BC" || cell.Children[1].Value != "C" || cell.Children[1].StartPos != 41 || cell.Children[1].EndPos != 42 {
		t.Fatalf("Wrong independent cell anchor continuation: %#v", cell.Children)
	}
}

func TestParseTableCaptionMarkerClearsFormattingBeforeCell(t *testing.T) {
	const content = `<table><caption><b>x<tbody><tr><td>y</b>z</table>q`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	caption := requireTableAdoptionElements(t, table, "caption", 1)[0]
	bold := requireTableAdoptionElements(t, table, "b", 1)[0]
	tbody := requireTableAdoptionElements(t, table, "tbody", 1)[0]
	cell := requireTableAdoptionElements(t, table, "td", 1)[0]
	assertFormattingNode(t, table, "table", "xyz", 0, 49)
	assertFormattingNode(t, caption, "caption", "x", 7, 20)
	assertFormattingNode(t, bold, "b", "x", 16, 20)
	assertFormattingNode(t, tbody, "tbody", "yz", 20, 41)
	assertFormattingNode(t, cell, "td", "yz", 31, 41)
	if bold.Parent != caption || len(formattingElements(cell, "b")) != 0 || len(cell.Children) != 1 || cell.Children[0].Value != "yz" || cell.Children[0].StartPos != 35 || cell.Children[0].EndPos != 41 || parsedBodyChildren(document)[1].Value != "q" {
		t.Fatalf("Caption marker leaked b into cell or continuation: caption=%#v cell=%#v doc=%#v", caption.Children, cell.Children, parsedBodyChildren(document))
	}
}

func TestParseNestedTableFosterFormattingStaysInOuterCell(t *testing.T) {
	const content = `<table><tr><td><b>A<table><i>B</i><tr><td>C</td></tr>D</table>E</b></td></tr></table>F`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	tables := requireTableAdoptionElements(t, document, "table", 2)
	outerCells := formattingElements(tables[0], "td")
	if len(outerCells) != 2 {
		t.Fatalf("Expected outer and nested td in table walk, got %#v", outerCells)
	}
	outerCell := outerCells[0]
	innerCell := outerCells[1]
	bold := requireTableAdoptionElements(t, outerCell, "b", 1)[0]
	italics := requireTableAdoptionElements(t, bold, "i", 1)[0]
	assertFormattingNode(t, tables[0], "table", "ABDCE", 0, 85)
	assertFormattingNode(t, outerCell, "td", "ABDCE", 11, 72)
	assertFormattingNode(t, bold, "b", "ABDCE", 15, 67)
	assertFormattingNode(t, italics, "i", "B", 26, 34)
	assertFormattingNode(t, tables[1], "table", "C", 19, 62)
	assertFormattingNode(t, innerCell, "td", "C", 38, 48)
	if len(bold.Children) != 5 || bold.Children[0].Value != "A" || bold.Children[1] != italics || bold.Children[2].Value != "D" || bold.Children[3] != tables[1] || bold.Children[4].Value != "E" {
		t.Fatalf("Inner-table foster content must stay in outer b/cell around inner table: %#v", bold.Children)
	}
	if bold.Children[2].StartPos != 53 || bold.Children[2].EndPos != 54 || bold.Children[4].StartPos != 62 || bold.Children[4].EndPos != 63 || parsedBodyChildren(document)[1].Value != "F" || parsedBodyChildren(document)[1].StartPos != 85 || parsedBodyChildren(document)[1].EndPos != 86 {
		t.Fatalf("Wrong nested foster source ranges: b=%#v doc=%#v", bold.Children, parsedBodyChildren(document))
	}
}

func TestParseTableFormattingWhitespaceDoesNotReconstructOrFoster(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		spaceEnd    int
		tableEnd    int
		documentEnd int
	}{
		{"literal", `<table><b><tr><td>a</td></tr>   </table>x`, 32, 40, 41},
		{"numeric-reference", `<table><b><tr><td>a</td></tr>&#32;</table>x`, 34, 42, 43},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			bold := requireTableAdoptionElements(t, document, "b", 2)
			table := requireTableAdoptionElements(t, document, "table", 1)[0]
			tbody := requireTableAdoptionElements(t, table, "tbody", 1)[0]
			assertFormattingNode(t, bold[0], "b", "", 7, 10)
			expectedText := "a "
			if test.name == "literal" {
				expectedText = "a   "
			}
			assertFormattingNode(t, table, "table", expectedText, 0, test.tableEnd)
			assertFormattingNode(t, bold[1], "b", "x", 7, test.documentEnd)
			if bold[0].Parent != parsedBody(document) || bold[1].Parent != parsedBody(document) || len(tbody.Children) != 2 || tbody.Children[1].Type != types.TextNode || tbody.Children[1].StartPos != 29 || tbody.Children[1].EndPos != test.spaceEnd {
				t.Fatalf("Table whitespace must stay after row without reconstruction/fostering: tbody=%#v b=%#v", tbody.Children, bold)
			}
		})
	}
}

func TestParseTableFormattingNonspaceReferenceRunReconstructsAndFosters(t *testing.T) {
	const content = `<table><b><tr><td>a</td></tr>&#32;&#65;</table>x`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := requireTableAdoptionElements(t, document, "b", 3)
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	if len(parsedBodyChildren(document)) != 4 || parsedBodyChildren(document)[0] != bold[0] || parsedBodyChildren(document)[1] != bold[1] || parsedBodyChildren(document)[2] != table || parsedBodyChildren(document)[3] != bold[2] {
		t.Fatalf("Decoded nonspace table run must foster before table in reconstructed b: %#v", parsedBodyChildren(document))
	}
	assertFormattingNode(t, bold[0], "b", "", 7, 10)
	assertFormattingNode(t, bold[1], "b", " A", 7, 39)
	assertFormattingNode(t, table, "table", "a", 0, 47)
	assertFormattingNode(t, bold[2], "b", "x", 7, 48)
	if len(bold[1].Children) != 1 || bold[1].Children[0].Value != " A" || bold[1].Children[0].StartPos != 29 || bold[1].Children[0].EndPos != 39 {
		t.Fatalf("Wrong decoded fostered text range: %#v", bold[1].Children)
	}
}

func TestParseFormattingStartedOutsideTableUsesTableScope(t *testing.T) {
	const content = `<b>1<table>2</b>3<tr><td>x</td></tr></table>4`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := requireTableAdoptionElements(t, document, "b", 1)[0]
	table := requireTableAdoptionElements(t, bold, "table", 1)[0]
	cell := requireTableAdoptionElements(t, table, "td", 1)[0]
	assertFormattingNode(t, bold, "b", "123x4", 0, 45)
	assertFormattingNode(t, table, "table", "x", 4, 44)
	assertFormattingNode(t, cell, "td", "x", 21, 31)
	if bold.Children[0].Value != "123" || bold.Children[0].StartPos != 3 || bold.Children[0].EndPos != 17 || bold.Children[1] != table || bold.Children[2].Value != "4" || bold.Children[2].StartPos != 44 || bold.Children[2].EndPos != 45 {
		t.Fatalf("Out-of-table b end must be ignored behind table scope while foster text stays inside b: %#v", bold.Children)
	}
}

func TestParseAnchorAndNobrStartedOutsideTableUseDistinctSpecialStartRules(t *testing.T) {
	const anchorContent = `<a id=o>1<table>2<a id=n>3</a>4<tr><td>x</td></tr></table>5</a>6`
	document, err := NewHTMLParser().Parse(anchorContent)
	if err != nil {
		t.Fatal(err)
	}
	anchors := requireTableAdoptionElements(t, document, "a", 2)
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	assertFormattingNode(t, anchors[0], "a", "1234x", 0, 17)
	assertFormattingNode(t, anchors[1], "a", "3", 17, 30)
	assertFormattingNode(t, table, "table", "x", 9, 58)
	if anchors[0].Attributes["id"] != "o" || anchors[1].Attributes["id"] != "n" || anchors[1].Parent != anchors[0] || table.Parent != anchors[0] || len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].Type != types.TextNode || parsedBodyChildren(document)[1].Value != "56" || parsedBodyChildren(document)[1].StartPos != 58 || parsedBodyChildren(document)[1].EndPos != 64 {
		t.Fatalf("Anchor start behind table scope must forcibly remove old AFE while retaining DOM ancestry: a=%#v doc=%#v", anchors, parsedBodyChildren(document))
	}

	const nobrContent = `<nobr>1<table>2<nobr>3</nobr>4<tr><td>x</td></tr></table>5</nobr>6`
	document, err = NewHTMLParser().Parse(nobrContent)
	if err != nil {
		t.Fatal(err)
	}
	nobrs := requireTableAdoptionElements(t, document, "nobr", 2)
	table = requireTableAdoptionElements(t, document, "table", 1)[0]
	assertFormattingNode(t, nobrs[0], "nobr", "1234x5", 0, 65)
	assertFormattingNode(t, nobrs[1], "nobr", "3", 15, 29)
	assertFormattingNode(t, table, "table", "x", 7, 57)
	if nobrs[1].Parent != nobrs[0] || table.Parent != nobrs[0] || nobrs[0].Children[4].Value != "5" || parsedBodyChildren(document)[1].Value != "6" || parsedBodyChildren(document)[1].StartPos != 65 || parsedBodyChildren(document)[1].EndPos != 66 {
		t.Fatalf("Nobr table-scope start/end behavior mismatch: nobr=%#v doc=%#v", nobrs, parsedBodyChildren(document))
	}
}

func TestParseTableFosteredAdoptionMultibyteLocations(t *testing.T) {
	const content = "<table><b>é\r\n<div>😀</b>z</div><tr><td>x</table>w"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := requireTableAdoptionElements(t, document, "b", 2)
	div := requireTableAdoptionElements(t, document, "div", 1)[0]
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	assertFormattingNode(t, bold[0], "b", "é\n", 7, 27)
	assertFormattingNode(t, bold[1], "b", "😀", 0, 0)
	assertFormattingNode(t, div, "div", "😀z", 14, 34)
	assertFormattingNode(t, table, "table", "x", 0, 51)
	if bold[0].StartLine != 1 || bold[0].StartColumn != 8 || bold[0].EndLine != 2 || bold[0].EndColumn != 12 || div.StartLine != 2 || div.StartColumn != 1 || div.EndLine != 2 || div.EndColumn != 19 || table.EndLine != 2 || table.EndColumn != 36 {
		t.Fatalf("Wrong fostered UTF-8 byte / UTF-16 coordinates: b=%#v div=%#v table=%#v", bold[0], div, table)
	}
	if div.Children[1].Value != "z" || div.Children[1].StartPos != 27 || div.Children[1].EndPos != 28 || parsedBodyChildren(document)[3].Value != "w" || parsedBodyChildren(document)[3].StartPos != 51 || parsedBodyChildren(document)[3].EndPos != 52 {
		t.Fatalf("Wrong multibyte foster continuation byte ranges: div=%#v doc=%#v", div.Children, parsedBodyChildren(document))
	}
}

func TestParseTableAdoptionScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n * 64)
		for i := 0; i < n; i++ {
			b.WriteString(`<table><b>x<div>y</b>z</div><tr><td>q</table>r`)
		}
		return b.String()
	}
	measure := func(n int) time.Duration {
		content := build(n)
		_, _ = NewHTMLParser().Parse(content)
		best := time.Duration(1<<63 - 1)
		for i := 0; i < 3; i++ {
			start := time.Now()
			if _, err := NewHTMLParser().Parse(content); err != nil {
				t.Fatal(err)
			}
			if elapsed := time.Since(start); elapsed < best {
				best = elapsed
			}
		}
		return best
	}
	small, large := measure(300), measure(1200)
	ratio := float64(large) / float64(small)
	t.Logf("table adoption scaling 300=%v 1200=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 30*time.Millisecond {
		t.Fatalf("Table foster/adoption recovery scaled superlinearly: %.1fx", ratio)
	}
}

func TestParseTableAdoptionIncompleteEndAndReuse(t *testing.T) {
	parser := NewHTMLParser()
	const content = `<table><tr><td><b>1<div>2</b`
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	div := formattingElements(document, "div")
	if len(bold) != 1 || len(div) != 1 || div[0].Parent != bold[0] || bold[0].TextContent != "12" || bold[0].EndPos != len(content) || div[0].EndPos != len(content) {
		t.Fatalf("Incomplete cell adoption end must not run surgery: b=%#v div=%#v", bold, div)
	}
	plain, err := parser.Parse(`z`)
	if err != nil || len(parsedBodyChildren(plain)) != 1 || parsedBodyChildren(plain)[0].Type != types.TextNode || parsedBodyChildren(plain)[0].Value != "z" || len(formattingElements(plain, "b")) != 0 {
		t.Fatalf("Parser reuse leaked table adoption state: doc=%#v err=%v", plain, err)
	}
}

func TestParseTableFosteredIncompleteEndFlushesAtEOFWithoutSurgery(t *testing.T) {
	const content = `<table><b>x<div>y</b`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	bold := requireTableAdoptionElements(t, document, "b", 1)[0]
	div := requireTableAdoptionElements(t, document, "div", 1)[0]
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0] != bold || parsedBodyChildren(document)[1] != table || div.Parent != bold {
		t.Fatalf("EOF foster flush must place original b/div before empty table: %#v", parsedBodyChildren(document))
	}
	assertFormattingNode(t, bold, "b", "xy", 7, len(content))
	assertFormattingNode(t, div, "div", "y", 11, len(content))
	assertFormattingNode(t, table, "table", "", 0, len(content))
	if len(formattingElements(div, "b")) != 0 {
		t.Fatalf("Incomplete end must not run adoption surgery: %#v", div.Children)
	}
}

func TestParseTableBodyAndRowFosteredAdoption(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		boldStart int
		boldEnd   int
		divStart  int
		divEnd    int
		tableEnd  int
	}{
		{"tbody", `<table><tbody><b>x<div>y</b>z</div><tr><td>q</table>r`, 14, 28, 18, 35, 52},
		{"row", `<table><tr><b>x<div>y</b>z</div><td>q</table>r`, 11, 25, 15, 32, 45},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			bold := requireTableAdoptionElements(t, document, "b", 2)
			div := requireTableAdoptionElements(t, document, "div", 1)[0]
			table := requireTableAdoptionElements(t, document, "table", 1)[0]
			if len(parsedBodyChildren(document)) != 4 || parsedBodyChildren(document)[0] != bold[0] || parsedBodyChildren(document)[1] != div || parsedBodyChildren(document)[2] != table {
				t.Fatalf("%s foster order must be b, div, table, tail: %#v", test.name, parsedBodyChildren(document))
			}
			assertFormattingNode(t, bold[0], "b", "x", test.boldStart, test.boldEnd)
			assertFormattingNode(t, bold[1], "b", "y", 0, 0)
			assertFormattingNode(t, div, "div", "yz", test.divStart, test.divEnd)
			assertFormattingNode(t, table, "table", "q", 0, test.tableEnd)
			if bold[1].Parent != div || parsedBodyChildren(document)[3].Value != "r" || parsedBodyChildren(document)[3].StartPos != test.tableEnd || parsedBodyChildren(document)[3].EndPos != test.tableEnd+1 {
				t.Fatalf("Wrong %s foster ownership/continuation: div=%#v doc=%#v", test.name, div.Children, parsedBodyChildren(document))
			}
		})
	}
}

func TestParseTableCellFontAdoptionPreservesAttributes(t *testing.T) {
	const content = `<table><tr><td><font color=red>1<div>2</font>3</div>4</table>5`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	cell := requireTableAdoptionElements(t, table, "td", 1)[0]
	fonts := requireTableAdoptionElements(t, cell, "font", 2)
	div := requireTableAdoptionElements(t, cell, "div", 1)[0]
	assertFormattingNode(t, fonts[0], "font", "1", 15, 45)
	assertFormattingNode(t, fonts[1], "font", "2", 0, 0)
	assertFormattingNode(t, div, "div", "23", 32, 52)
	if fonts[0].Attributes["color"] != "red" || fonts[1].Attributes["color"] != "red" || fonts[1].Parent != div || cell.TextContent != "1234" {
		t.Fatalf("Cell font adoption lost attributes or ownership: font=%#v cell=%#v", fonts, cell)
	}
}
