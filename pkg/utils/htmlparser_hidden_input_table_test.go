package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 15 covers the hidden-input special case in HTML table modes. Form
// pointer/association, templates, foreign and fragment parsing, select/table
// interaction, and scripting are intentionally deferred.

func requireInputByID(t *testing.T, node *types.Node, id string) *types.Node {
	t.Helper()
	for _, input := range formattingElements(node, "input") {
		if input.Attributes["id"] == id {
			return input
		}
	}
	t.Fatalf("Expected input#%s in %#v", id, node)
	return nil
}

func TestParseHiddenInputPlacementAcrossTableModes(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		parentName string
		start      int
		end        int
	}{
		{"table", `<table><input id=h type=hidden><tr><td>x</table>z`, "table", 7, 31},
		{"tbody-case-insensitive", `<table><tbody><input id=h type=HIDDEN><tr><td>x</table>z`, "tbody", 14, 38},
		{"row-decoded-reference", `<table><tr><input id=h type=hidd&#101;n><td>x</table>z`, "tr", 11, 40},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			table := requireTableAdoptionElements(t, document, "table", 1)[0]
			input := requireTableAdoptionElements(t, document, "input", 1)[0]
			cell := requireTableAdoptionElements(t, table, "td", 1)[0]
			assertFormattingNode(t, input, "input", "", test.start, test.end)
			if input.Parent == nil || input.Parent.Name != test.parentName || input.Attributes["id"] != "h" || strings.ToLower(input.Attributes["type"]) != "hidden" {
				t.Fatalf("Hidden input was not inserted into current <%s>: %#v", test.parentName, input)
			}
			if table.TextContent != "x" || cell.TextContent != "x" || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].Value != "z" {
				t.Fatalf("Hidden input changed table continuation: table=%#v doc=%#v", table, parsedBodyChildren(document))
			}
		})
	}
}

func TestParseHiddenAndOrdinaryInputCombinedTableOrder(t *testing.T) {
	const content = `<div id=p></div><table id=t><input type=hidden id=h><input id=o><tr><td>x</table>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	div := requireTableAdoptionElements(t, document, "div", 1)[0]
	if len(formattingElements(document, "input")) != 2 {
		t.Fatalf("Expected two inputs: %#v", document)
	}
	hidden := requireInputByID(t, document, "h")
	ordinary := requireInputByID(t, document, "o")
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[0] != div || parsedBodyChildren(document)[1] != ordinary || parsedBodyChildren(document)[2] != table {
		t.Fatalf("Expected div, fostered ordinary input, then table: %#v", parsedBodyChildren(document))
	}
	assertFormattingNode(t, div, "div", "", 0, 16)
	assertFormattingNode(t, hidden, "input", "", 28, 52)
	assertFormattingNode(t, ordinary, "input", "", 52, 64)
	assertFormattingNode(t, table, "table", "x", 16, 81)
	if hidden.Parent != table || ordinary.Parent != parsedBody(document) || table.Children[0] != hidden {
		t.Fatalf("Hidden/ordinary combined placement mismatch: inputs=%#v table=%#v", formattingElements(document, "input"), table.Children)
	}
}

func TestParseHiddenInputAfterExplicitCellCloseStaysInRow(t *testing.T) {
	const content = `<table><tr><td>x</td><input type=hidden id=h><td>y</table>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	row := requireTableAdoptionElements(t, document, "tr", 1)[0]
	cells := requireTableAdoptionElements(t, row, "td", 2)
	input := requireTableAdoptionElements(t, row, "input", 1)[0]
	assertFormattingNode(t, row, "tr", "xy", 7, 50)
	assertFormattingNode(t, cells[0], "td", "x", 11, 21)
	assertFormattingNode(t, input, "input", "", 21, 45)
	assertFormattingNode(t, cells[1], "td", "y", 45, 50)
	if len(row.Children) != 3 || row.Children[0] != cells[0] || row.Children[1] != input || row.Children[2] != cells[1] {
		t.Fatalf("Hidden input must stay between explicitly separated cells: %#v", row.Children)
	}
}

func TestParseHiddenInputDoesNotReconstructOffStackFormatting(t *testing.T) {
	const content = `<table><b><tr><td>x</td></tr><input type=hidden id=h><!--c--><input id=o></table>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := requireTableAdoptionElements(t, document, "b", 3)
	if len(formattingElements(document, "input")) != 2 {
		t.Fatalf("Expected two inputs: %#v", document)
	}
	hidden := requireInputByID(t, document, "h")
	ordinary := requireInputByID(t, document, "o")
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	tbody := requireTableAdoptionElements(t, table, "tbody", 1)[0]
	if len(parsedBodyChildren(document)) != 4 || parsedBodyChildren(document)[0] != bold[0] || parsedBodyChildren(document)[1] != bold[1] || parsedBodyChildren(document)[2] != table || parsedBodyChildren(document)[3] != bold[2] {
		t.Fatalf("Expected empty b, b(ordinary input), table, b(z): %#v", parsedBodyChildren(document))
	}
	assertFormattingNode(t, bold[0], "b", "", 7, 10)
	assertFormattingNode(t, bold[1], "b", "", 7, 73)
	assertFormattingNode(t, hidden, "input", "", 29, 53)
	assertFormattingNode(t, ordinary, "input", "", 61, 73)
	assertFormattingNode(t, bold[2], "b", "z", 7, 82)
	if hidden.Parent != tbody || ordinary.Parent != bold[1] || len(tbody.Children) != 3 || tbody.Children[2].Type != types.CommentNode || tbody.Children[2].Value != "c" {
		t.Fatalf("Hidden input must not reconstruct/foster while ordinary input does: inputs=%#v tbody=%#v", formattingElements(document, "input"), tbody.Children)
	}
}

func TestParseHiddenInputInOpenFosterFormattingUsesCurrentNode(t *testing.T) {
	const content = `<table><b><input type=hidden id=h><input id=o><tr><td>x</table>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := requireTableAdoptionElements(t, document, "b", 2)
	inputs := requireTableAdoptionElements(t, document, "input", 2)
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[0] != bold[0] || parsedBodyChildren(document)[1] != table || parsedBodyChildren(document)[2] != bold[1] {
		t.Fatalf("Expected b(inputs), table, b(z): %#v", parsedBodyChildren(document))
	}
	assertFormattingNode(t, bold[0], "b", "", 7, 46)
	assertFormattingNode(t, inputs[0], "input", "", 10, 34)
	assertFormattingNode(t, inputs[1], "input", "", 34, 46)
	if inputs[0].Parent != bold[0] || inputs[1].Parent != bold[0] || len(bold[0].Children) != 2 || bold[1].TextContent != "z" {
		t.Fatalf("Generic foster loop did not insert both inputs at current b: b=%#v inputs=%#v", bold, inputs)
	}
}

func TestParseHiddenInputClosesColgroupAndStaysInTable(t *testing.T) {
	const content = `<table><colgroup id=c><input type=hidden id=h><col id=k></table>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	colgroups := requireTableAdoptionElements(t, table, "colgroup", 2)
	input := requireTableAdoptionElements(t, table, "input", 1)[0]
	col := requireTableAdoptionElements(t, table, "col", 1)[0]
	assertFormattingNode(t, colgroups[0], "colgroup", "", 7, 22)
	assertFormattingNode(t, input, "input", "", 22, 46)
	assertFormattingNode(t, colgroups[1], "colgroup", "", 0, 0)
	assertFormattingNode(t, col, "col", "", 46, 56)
	if len(table.Children) != 3 || table.Children[0] != colgroups[0] || table.Children[1] != input || table.Children[2] != colgroups[1] || col.Parent != colgroups[1] {
		t.Fatalf("Hidden input must close explicit colgroup, stay in table, then allow synthetic colgroup: %#v", table.Children)
	}
}

func TestParseNestedTableHiddenAndOrdinaryInputPlacement(t *testing.T) {
	const content = `<table><tr><td><table><input type=hidden id=h><input id=o><tr><td>x</table>y</table>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	tables := requireTableAdoptionElements(t, document, "table", 2)
	if len(formattingElements(document, "input")) != 2 {
		t.Fatalf("Expected two inputs: %#v", document)
	}
	hidden := requireInputByID(t, document, "h")
	ordinary := requireInputByID(t, document, "o")
	outerCells := formattingElements(tables[0], "td")
	outerCell := outerCells[0]
	assertFormattingNode(t, tables[0], "table", "xy", 0, 84)
	assertFormattingNode(t, tables[1], "table", "x", 15, 75)
	assertFormattingNode(t, hidden, "input", "", 22, 46)
	assertFormattingNode(t, ordinary, "input", "", 46, 58)
	if hidden.Parent != tables[1] || ordinary.Parent != outerCell || len(outerCell.Children) != 3 || outerCell.Children[0] != ordinary || outerCell.Children[1] != tables[1] || outerCell.Children[2].Value != "y" {
		t.Fatalf("Nested table hidden/ordinary placement mismatch: input=%#v outerCell=%#v", formattingElements(document, "input"), outerCell.Children)
	}
}

func TestParseOrdinaryInputsAreFosteredBeforeTable(t *testing.T) {
	tests := []struct {
		name     string
		startTag string
		typeVal  string
	}{
		{"text", `<input id=t type=text>`, "text"},
		{"missing", `<input id=t>`, ""},
		{"boolean-hidden-no-type", `<input hidden id=t>`, ""},
		{"surrounding-space", `<input id=t type=' hidden '>`, " hidden "},
		{"crlf-in-type", "<input id=t type='hid\r\nden'>", "hid\nden"},
		{"nul", "<input id=t type=hid\x00den>", "hid�den"},
		{"non-ascii", `<input id=t type=hıdden>`, "hıdden"},
		{"decoded-trailing-space", `<input id=t type=hidden&#32;>`, "hidden "},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content := `<table>` + test.startTag + `<tr><td>x</table>z`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			table := requireTableAdoptionElements(t, document, "table", 1)[0]
			input := requireTableAdoptionElements(t, document, "input", 1)[0]
			if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[0] != input || parsedBodyChildren(document)[1] != table || input.Parent != parsedBody(document) {
				t.Fatalf("Ordinary input must be fostered immediately before table: %#v", parsedBodyChildren(document))
			}
			if input.Attributes["type"] != test.typeVal || input.StartPos != 7 || input.EndPos != 7+len([]byte(test.startTag)) {
				t.Fatalf("Wrong ordinary input value/range: %#v", input)
			}
		})
	}
}

func TestParseHiddenInputSectionPlacement(t *testing.T) {
	const content = `<table><tbody id=a><tr><td>x</td></tr><input type=hidden id=h></tbody><input type=hidden id=i><tfoot id=f><input type=hidden id=j><tr><td>y</table>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	tbody := requireTableAdoptionElements(t, table, "tbody", 1)[0]
	tfoot := requireTableAdoptionElements(t, table, "tfoot", 1)[0]
	h := requireInputByID(t, table, "h")
	i := requireInputByID(t, table, "i")
	j := requireInputByID(t, table, "j")
	assertFormattingNode(t, tbody, "tbody", "x", 7, 70)
	assertFormattingNode(t, h, "input", "", 38, 62)
	assertFormattingNode(t, i, "input", "", 70, 94)
	assertFormattingNode(t, tfoot, "tfoot", "y", 94, 139)
	assertFormattingNode(t, j, "input", "", 106, 130)
	if h.Parent != tbody || i.Parent != table || j.Parent != tfoot || tbody.Attributes["id"] != "a" || tfoot.Attributes["id"] != "f" {
		t.Fatalf("Hidden input section placement mismatch: h=%#v i=%#v j=%#v", h, i, j)
	}
}

func TestParseHiddenInputInOpenFosterDivUsesCurrentNode(t *testing.T) {
	const content = `<table><div id=d><input type=hidden id=h><input id=o><tr><td>x</table>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	div := requireTableAdoptionElements(t, document, "div", 1)[0]
	h := requireInputByID(t, document, "h")
	o := requireInputByID(t, document, "o")
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	assertFormattingNode(t, div, "div", "", 7, 53)
	assertFormattingNode(t, h, "input", "", 17, 41)
	assertFormattingNode(t, o, "input", "", 41, 53)
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0] != div || parsedBodyChildren(document)[1] != table || h.Parent != div || o.Parent != div || len(div.Children) != 2 {
		t.Fatalf("Open foster div must own both hidden and ordinary input: div=%#v doc=%#v", div.Children, parsedBodyChildren(document))
	}
}

func TestParseHiddenAndOrdinaryInputInsideCellFormattingUseInBodyMode(t *testing.T) {
	const content = `<table><tr><td><b><input type=hidden id=h><input id=o>x</table>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	cell := requireTableAdoptionElements(t, document, "td", 1)[0]
	bold := requireTableAdoptionElements(t, cell, "b", 1)[0]
	h := requireInputByID(t, bold, "h")
	o := requireInputByID(t, bold, "o")
	assertFormattingNode(t, bold, "b", "x", 15, 55)
	assertFormattingNode(t, h, "input", "", 18, 42)
	assertFormattingNode(t, o, "input", "", 42, 54)
	if h.Parent != bold || o.Parent != bold || len(bold.Children) != 3 || bold.Children[2].Value != "x" {
		t.Fatalf("Cell in-body mode must keep both inputs under b: %#v", bold.Children)
	}
}

func TestParseHiddenInputUnexpectedSolidusAttributeRemainsHidden(t *testing.T) {
	const content = `<table><input type=hidden /foo=bar><tr><td>x</table>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	input := requireTableAdoptionElements(t, document, "input", 1)[0]
	assertFormattingNode(t, input, "input", "", 7, 35)
	if input.Parent.Name != "table" || input.Attributes["type"] != "hidden" || input.Attributes["foo"] != "bar" {
		t.Fatalf("Unexpected solidus recovery must retain hidden type and following foo attr: %#v", input)
	}
}

func TestParseHiddenInputDuplicateAttributesUseFirstValue(t *testing.T) {
	tests := []struct {
		name       string
		startTag   string
		inside     bool
		expectedTy string
	}{
		{"hidden-first", `<input id=h type=hidden type=text>`, true, "hidden"},
		{"text-first", `<input id=t type=text type=hidden>`, false, "text"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content := `<table>` + test.startTag + `<tr><td>x</table>z`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			table := requireTableAdoptionElements(t, document, "table", 1)[0]
			input := requireTableAdoptionElements(t, document, "input", 1)[0]
			if input.Attributes["type"] != test.expectedTy || len(input.AttributeOrder) != 2 {
				t.Fatalf("Duplicate attributes must keep the first value: %#v", input)
			}
			if test.inside != (input.Parent == table) {
				t.Fatalf("First duplicate type must control hidden placement: input=%#v table=%#v", input, table)
			}
		})
	}
}

func TestParseHiddenInputWithActiveFormattingDoesNotFosterItself(t *testing.T) {
	const content = `<table><b>a<input id=h type=hidden>b<tr><td>x</table>c`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := requireTableAdoptionElements(t, document, "b", 2)
	input := requireTableAdoptionElements(t, document, "input", 1)[0]
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[0] != bold[0] || parsedBodyChildren(document)[1] != table || parsedBodyChildren(document)[2] != bold[1] {
		t.Fatalf("Expected b(ab), table, b(c): %#v", parsedBodyChildren(document))
	}
	assertFormattingNode(t, bold[0], "b", "ab", 7, 36)
	assertFormattingNode(t, input, "input", "", 11, 35)
	assertFormattingNode(t, bold[1], "b", "c", 7, 54)
	if input.Parent != bold[0] || bold[0].Children[1] != input || table.TextContent != "x" {
		t.Fatalf("Hidden input must remain in the active fostered b without contributing text: b=%#v input=%#v", bold[0], input)
	}
}

func TestParseHiddenInputInCellAndCaptionUsesNormalInBodyPlacement(t *testing.T) {
	const cellContent = `<table><tr><td><input id=h type=hidden>x</table>z`
	document, err := NewHTMLParser().Parse(cellContent)
	if err != nil {
		t.Fatal(err)
	}
	cell := requireTableAdoptionElements(t, document, "td", 1)[0]
	input := requireTableAdoptionElements(t, document, "input", 1)[0]
	assertFormattingNode(t, input, "input", "", 15, 39)
	if input.Parent != cell || cell.TextContent != "x" || cell.Children[1].Value != "x" {
		t.Fatalf("Cell input must use ordinary in-body insertion: %#v", cell.Children)
	}

	const captionContent = `<table><caption><input id=h type=hidden>x</caption><tr><td>y</table>z`
	document, err = NewHTMLParser().Parse(captionContent)
	if err != nil {
		t.Fatal(err)
	}
	caption := requireTableAdoptionElements(t, document, "caption", 1)[0]
	input = requireTableAdoptionElements(t, document, "input", 1)[0]
	assertFormattingNode(t, input, "input", "", 16, 40)
	if input.Parent != caption || caption.TextContent != "x" || requireTableAdoptionElements(t, document, "td", 1)[0].TextContent != "y" {
		t.Fatalf("Caption input must use ordinary in-body insertion: %#v", caption.Children)
	}
}

func TestParseHiddenInputVoidSyntaxIgnoredEndAndEOF(t *testing.T) {
	const separatedSlash = `<table><input type=hidden /><tr><td>x</table>z`
	document, err := NewHTMLParser().Parse(separatedSlash)
	if err != nil {
		t.Fatal(err)
	}
	input := requireTableAdoptionElements(t, document, "input", 1)[0]
	if input.Parent.Name != "table" || input.Attributes["type"] != "hidden" {
		t.Fatalf("Separated self-closing slash changed hidden input placement: %#v", input)
	}

	const attachedSlash = `<table><input type=hidden/><tr><td>x</table>z`
	document, err = NewHTMLParser().Parse(attachedSlash)
	if err != nil {
		t.Fatal(err)
	}
	input = requireTableAdoptionElements(t, document, "input", 1)[0]
	if input.Parent != parsedBody(document) || input.Attributes["type"] != "hidden/" {
		t.Fatalf("Attached slash must remain in unquoted value and prevent hidden matching: %#v", input)
	}

	const ignoredEnd = `<table><input type=hidden></input><tr><td>x</table>z`
	document, err = NewHTMLParser().Parse(ignoredEnd)
	if err != nil {
		t.Fatal(err)
	}
	input = requireTableAdoptionElements(t, document, "input", 1)[0]
	if input.Parent.Name != "table" || input.EndPos != 26 || requireTableAdoptionElements(t, document, "tr", 1)[0].StartPos != 34 {
		t.Fatalf("Ignored input end must not change void node range or continuation: %#v", input)
	}

	const incomplete = `<table><input type=hidden`
	document, err = NewHTMLParser().Parse(incomplete)
	if err != nil {
		t.Fatal(err)
	}
	if len(formattingElements(document, "input")) != 0 {
		t.Fatalf("Incomplete input token at EOF must be discarded: %#v", document)
	}
}

func TestParseHiddenInputMultibyteLocationsAndReuse(t *testing.T) {
	const content = "<table>\r\n<input id=h type=hidden data-x=é😀><tr><td>x</table>z"
	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	input := requireTableAdoptionElements(t, document, "input", 1)[0]
	assertFormattingNode(t, input, "input", "", 9, 47)
	if input.Parent.Name != "table" || input.Attributes["data-x"] != "é😀" || input.StartLine != 2 || input.StartColumn != 1 || input.EndLine != 2 || input.EndColumn != 36 {
		t.Fatalf("Wrong hidden-input UTF-8 byte / UTF-16 coordinates: %#v", input)
	}
	plain, err := parser.Parse(`z`)
	if err != nil || len(parsedBodyChildren(plain)) != 1 || parsedBodyChildren(plain)[0].Type != types.TextNode || parsedBodyChildren(plain)[0].Value != "z" || len(formattingElements(plain, "input")) != 0 {
		t.Fatalf("Parser reuse leaked hidden-input table state: doc=%#v err=%v", plain, err)
	}
}

func TestParseHiddenInputTableScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n * 96)
		for i := 0; i < n; i++ {
			if i%2 == 0 {
				b.WriteString(`<table><input type=hidden><tr><td>x</table>z`)
			} else {
				b.WriteString(`<table><b><tr><td>x</td></tr><input type=hidden><!--c--><input></table>z`)
			}
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
	small, large := measure(1000), measure(4000)
	ratio := float64(large) / float64(small)
	t.Logf("hidden input table scaling 1000=%v 4000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 30*time.Millisecond {
		t.Fatalf("Hidden-input table recovery scaled superlinearly: %.1fx", ratio)
	}
}
