package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 16 covers the whole-document, non-template HTML form pointer and the
// in-table form rule. Template-specific pointer behavior, fragment-context
// seeding, form association/activation and control semantics, foreign content,
// and scripting are intentionally deferred.

func formByID(t *testing.T, node *types.Node, id string) *types.Node {
	t.Helper()
	for _, form := range formattingElements(node, "form") {
		if form.Attributes["id"] == id {
			return form
		}
	}
	t.Fatalf("Expected form#%s in %#v", id, node)
	return nil
}

func TestParseFormPointerIgnoresNestedStart(t *testing.T) {
	const content = `<form id=a><div>1<form id=b>2</form>3</div>4</form>5`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	forms := formattingElements(document, "form")
	if len(forms) != 1 {
		t.Fatalf("Nested form start must be ignored while pointer is set: %#v", forms)
	}
	form := forms[0]
	div := requireTableAdoptionElements(t, form, "div", 1)[0]
	assertFormattingNode(t, form, "form", "123", 0, 36)
	assertFormattingNode(t, div, "div", "123", 11, 43)
	if form.Attributes["id"] != "a" || len(div.Children) != 1 || div.Children[0].Value != "123" || div.Children[0].StartPos != 16 || div.Children[0].EndPos != 37 || parsedBodyChildren(document)[1].Value != "45" || parsedBodyChildren(document)[1].StartPos != 43 || parsedBodyChildren(document)[1].EndPos != 52 {
		t.Fatalf("Nested ignored form token changed tree/ranges: form=%#v div=%#v doc=%#v", form, div.Children, parsedBodyChildren(document))
	}
}

func TestParseFormStartClosesParagraphInButtonScope(t *testing.T) {
	const content = `<p>x<form id=f>y</form>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	p := requireTableAdoptionElements(t, document, "p", 1)[0]
	form := formByID(t, document, "f")
	assertFormattingNode(t, p, "p", "x", 0, 4)
	assertFormattingNode(t, form, "form", "y", 4, 23)
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[0] != p || parsedBodyChildren(document)[1] != form || parsedBodyChildren(document)[2].Value != "z" {
		t.Fatalf("Form start must close p before insertion: %#v", parsedBodyChildren(document))
	}
}

func TestParseFormEndRemovesExactNonCurrentForm(t *testing.T) {
	const content = `<form id=a><div>x</form>y</div>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	form := formByID(t, document, "a")
	div := requireTableAdoptionElements(t, form, "div", 1)[0]
	assertFormattingNode(t, form, "form", "xy", 0, 24)
	assertFormattingNode(t, div, "div", "xy", 11, 31)
	if div.Parent != form || len(div.Children) != 1 || div.Children[0].Value != "xy" || div.Children[0].StartPos != 16 || div.Children[0].EndPos != 25 || parsedBodyChildren(document)[1].Value != "z" {
		t.Fatalf("Form end must remove pointed form without popping descendant div: form=%#v div=%#v doc=%#v", form, div.Children, parsedBodyChildren(document))
	}
}

func TestParseFormPointerScopeContrasts(t *testing.T) {
	const objectContent = `<form id=a><object><form id=b>x</form>y</object>z</form>`
	document, err := NewHTMLParser().Parse(objectContent)
	if err != nil {
		t.Fatal(err)
	}
	forms := formattingElements(document, "form")
	object := requireTableAdoptionElements(t, document, "object", 1)[0]
	if len(forms) != 1 || object.TextContent != "xy" || forms[0].TextContent != "xyz" {
		t.Fatalf("Object scope must block pointed form end after nested start is ignored: form=%#v object=%#v", forms, object)
	}
	assertFormattingNode(t, forms[0], "form", "xyz", 0, len(objectContent))

	const buttonContent = `<form id=a><button><form id=b>x</form>y</button>z</form>`
	document, err = NewHTMLParser().Parse(buttonContent)
	if err != nil {
		t.Fatal(err)
	}
	forms = formattingElements(document, "form")
	button := requireTableAdoptionElements(t, document, "button", 1)[0]
	if len(forms) != 1 || button.TextContent != "xy" || forms[0].TextContent != "xy" || parsedBodyChildren(document)[1].Value != "z" {
		t.Fatalf("Button is not an ordinary form-scope barrier: form=%#v button=%#v doc=%#v", forms, button, parsedBodyChildren(document))
	}
	assertFormattingNode(t, forms[0], "form", "xy", 0, 38)
}

func TestParseAbsentFormEndsAreIgnoredAndTextCoalesces(t *testing.T) {
	const content = `<div>a</form>b<form id=f>c</form>d</form>e</div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	div := requireTableAdoptionElements(t, document, "div", 1)[0]
	form := formByID(t, document, "f")
	assertFormattingNode(t, div, "div", "abcde", 0, 48)
	assertFormattingNode(t, form, "form", "c", 14, 33)
	if len(div.Children) != 3 || div.Children[0].Value != "ab" || div.Children[0].StartPos != 5 || div.Children[0].EndPos != 14 || div.Children[2].Value != "de" || div.Children[2].StartPos != 33 || div.Children[2].EndPos != 42 {
		t.Fatalf("Absent form ends must be ignored while adjacent text coalesces: %#v", div.Children)
	}
}

func TestParseInTableFormStartInsertsCurrentNodeThenPops(t *testing.T) {
	const content = `<table><form id=f><tr><td>x</form>y</table>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	form := formByID(t, table, "f")
	cell := requireTableAdoptionElements(t, table, "td", 1)[0]
	assertFormattingNode(t, form, "form", "", 7, 7)
	assertFormattingNode(t, cell, "td", "xy", 22, 35)
	if form.Parent != table || table.Children[0] != form || cell.Children[0].Value != "xy" || parsedBodyChildren(document)[1].Value != "z" {
		t.Fatalf("In-table form must remain empty table child while its end clears pointer out of scope: table=%#v cell=%#v", table.Children, cell.Children)
	}
}

func TestParseTableFormPointerIgnoresDuplicateStart(t *testing.T) {
	const content = `<table><form id=a><form id=b><tr><td>x</form>y</table>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	forms := formattingElements(document, "form")
	if len(forms) != 1 || forms[0].Attributes["id"] != "a" || forms[0].Parent.Name != "table" {
		t.Fatalf("Duplicate in-table form start must be ignored while pointer is set: %#v", forms)
	}
	assertFormattingNode(t, forms[0], "form", "", 7, 7)
	if requireTableAdoptionElements(t, document, "td", 1)[0].TextContent != "xy" {
		t.Fatalf("Ignored table form start/end changed cell continuation: %#v", document)
	}
}

func TestParseTableFormPointerPersistsAndClears(t *testing.T) {
	const persists = `<table><form id=a><tr><td>x</table><form id=b>y</form>z`
	document, err := NewHTMLParser().Parse(persists)
	if err != nil {
		t.Fatal(err)
	}
	forms := formattingElements(document, "form")
	if len(forms) != 1 || forms[0].Attributes["id"] != "a" || parsedBodyChildren(document)[1].Value != "yz" || parsedBodyChildren(document)[1].StartPos != 46 || parsedBodyChildren(document)[1].EndPos != 55 {
		t.Fatalf("Popped table form pointer must block later form until its end clears it: form=%#v doc=%#v", forms, parsedBodyChildren(document))
	}

	const clears = `<table><form id=a><tr><td>x</table></form><form id=b>y</form>z`
	document, err = NewHTMLParser().Parse(clears)
	if err != nil {
		t.Fatal(err)
	}
	forms = formattingElements(document, "form")
	if len(forms) != 2 {
		t.Fatalf("End after table must clear stale pointer and permit second form: %#v", forms)
	}
	assertFormattingNode(t, formByID(t, document, "a"), "form", "", 7, 7)
	assertFormattingNode(t, formByID(t, document, "b"), "form", "y", 42, 61)
}

func TestParseSequentialInTableFormsAfterPointerClear(t *testing.T) {
	const content = `<table><form id=a></form><form id=b><tr><td>x</table>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	forms := formattingElements(table, "form")
	if len(forms) != 2 || len(table.Children) != 3 || table.Children[0] != forms[0] || table.Children[1] != forms[1] {
		t.Fatalf("Cleared pointer must allow second empty table-child form: table=%#v forms=%#v", table.Children, forms)
	}
	assertFormattingNode(t, forms[0], "form", "", 7, 7)
	assertFormattingNode(t, forms[1], "form", "", 25, 25)
}

func TestParseInTableFormStartAcrossSectionRowAndColgroupModes(t *testing.T) {
	tests := []struct {
		name, content, parent string
		start, tableEnd       int
	}{
		{"tbody", `<table><tbody><form id=f><tr><td>x</form>y</table>z`, "tbody", 14, 50},
		{"row", `<table><tr><form id=f><td>x</form>y</table>z`, "tr", 11, 43},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			table := requireTableAdoptionElements(t, document, "table", 1)[0]
			form := formByID(t, table, "f")
			assertFormattingNode(t, form, "form", "", test.start, test.start)
			if form.Parent.Name != test.parent || requireTableAdoptionElements(t, table, "td", 1)[0].TextContent != "xy" || table.EndPos != test.tableEnd {
				t.Fatalf("In-%s form insertion/reprocess mismatch: form=%#v table=%#v", test.name, form, table)
			}
		})
	}

	const colgroupContent = `<table><colgroup id=c><form id=f><col id=k></table>z`
	document, err := NewHTMLParser().Parse(colgroupContent)
	if err != nil {
		t.Fatal(err)
	}
	table := requireTableAdoptionElements(t, document, "table", 1)[0]
	form := formByID(t, table, "f")
	colgroups := requireTableAdoptionElements(t, table, "colgroup", 2)
	assertFormattingNode(t, colgroups[0], "colgroup", "", 7, 22)
	assertFormattingNode(t, form, "form", "", 22, 22)
	assertFormattingNode(t, colgroups[1], "colgroup", "", 0, 0)
	if len(table.Children) != 3 || table.Children[0] != colgroups[0] || table.Children[1] != form || table.Children[2] != colgroups[1] {
		t.Fatalf("Colgroup must close/reprocess form at table, then synthesize colgroup: %#v", table.Children)
	}
}

func TestParseFormPointerInCellAndCaptionUsesInBodyRules(t *testing.T) {
	const cellContent = `<table><tr><td><form id=f>x<form id=g>y</form>z</table>w`
	document, err := NewHTMLParser().Parse(cellContent)
	if err != nil {
		t.Fatal(err)
	}
	forms := formattingElements(document, "form")
	cell := requireTableAdoptionElements(t, document, "td", 1)[0]
	if len(forms) != 1 || forms[0].Attributes["id"] != "f" || forms[0].Parent != cell || forms[0].TextContent != "xy" || cell.TextContent != "xyz" {
		t.Fatalf("Cell must use in-body pointer rules and ignore nested form start: forms=%#v cell=%#v", forms, cell)
	}
	assertFormattingNode(t, forms[0], "form", "xy", 15, 46)

	const captionContent = `<table><caption><form id=f>x<form id=g>y</form>z</caption><tr><td>q</table>w`
	document, err = NewHTMLParser().Parse(captionContent)
	if err != nil {
		t.Fatal(err)
	}
	forms = formattingElements(document, "form")
	caption := requireTableAdoptionElements(t, document, "caption", 1)[0]
	if len(forms) != 1 || forms[0].Parent != caption || forms[0].TextContent != "xy" || caption.TextContent != "xyz" {
		t.Fatalf("Caption must use in-body pointer rules: forms=%#v caption=%#v", forms, caption)
	}
	assertFormattingNode(t, forms[0], "form", "xy", 16, 47)
}

func TestParseInTableFormSyntaxAndIncompleteStart(t *testing.T) {
	const selfClosing = `<table><FORM ID=f /><tr><td>x</table>z`
	document, err := NewHTMLParser().Parse(selfClosing)
	if err != nil {
		t.Fatal(err)
	}
	form := formByID(t, document, "f")
	assertFormattingNode(t, form, "form", "", 7, 7)
	if form.Parent.Name != "table" {
		t.Fatalf("Nonvoid self-closing flag must not bypass in-table form rule: %#v", form)
	}

	const incomplete = `<table><form id=f`
	document, err = NewHTMLParser().Parse(incomplete)
	if err != nil {
		t.Fatal(err)
	}
	if len(formattingElements(document, "form")) != 0 || len(formattingElements(document, "table")) != 1 {
		t.Fatalf("Incomplete in-table form start must be discarded: %#v", document)
	}
}

func TestParseOuterFormPointerBlocksInTableFormStart(t *testing.T) {
	const content = `<form id=o><table><form id=t><tr><td>x</table>y</form>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	forms := formattingElements(document, "form")
	if len(forms) != 1 || forms[0].Attributes["id"] != "o" || forms[0].TextContent != "xy" || len(formattingElements(forms[0], "form")) != 1 {
		t.Fatalf("Outer form pointer must block table form start: %#v", forms)
	}
	assertFormattingNode(t, forms[0], "form", "xy", 0, 54)
}

func TestParseFormPointerSyntaxEOFMultibyteAndReuse(t *testing.T) {
	const syntax = `<FORM ID=a>x<form id=b>y</FORM>z`
	document, err := NewHTMLParser().Parse(syntax)
	if err != nil {
		t.Fatal(err)
	}
	forms := formattingElements(document, "form")
	if len(forms) != 1 || forms[0].Attributes["id"] != "a" || forms[0].TextContent != "xy" {
		t.Fatalf("ASCII-case form tokens or ignored nested start failed: %#v", forms)
	}
	assertFormattingNode(t, forms[0], "form", "xy", 0, 31)

	const selfClosing = `<form id=a/>x<form id=b>y</form>z`
	document, err = NewHTMLParser().Parse(selfClosing)
	if err != nil {
		t.Fatal(err)
	}
	forms = formattingElements(document, "form")
	if len(forms) != 1 || forms[0].Attributes["id"] != "a/" || forms[0].TextContent != "xy" {
		t.Fatalf("Attached slash must stay in id while nonvoid form remains open: %#v", forms)
	}

	const incomplete = `<form id=a`
	document, err = NewHTMLParser().Parse(incomplete)
	if err != nil || len(formattingElements(document, "form")) != 0 {
		t.Fatalf("Incomplete form start must be discarded: doc=%#v err=%v", document, err)
	}

	const eof = `<form id=a>x<!--c-->`
	parser := NewHTMLParser()
	document, err = parser.Parse(eof)
	if err != nil {
		t.Fatal(err)
	}
	form := formByID(t, document, "a")
	assertFormattingNode(t, form, "form", "x", 0, len(eof))
	if len(form.Children) != 2 || form.Children[1].Type != types.CommentNode || form.Children[1].Value != "c" || form.Children[1].StartPos != 12 || form.Children[1].EndPos != 20 {
		t.Fatalf("EOF form/comment recovery mismatch: %#v", form.Children)
	}

	const multibyte = "<form id=a>é\r\n<div>😀</form>z</div>w"
	document, err = parser.Parse(multibyte)
	if err != nil {
		t.Fatal(err)
	}
	form = formByID(t, document, "a")
	div := requireTableAdoptionElements(t, form, "div", 1)[0]
	assertFormattingNode(t, form, "form", "é\n😀z", 0, 31)
	assertFormattingNode(t, div, "div", "😀z", 15, 38)
	if form.EndLine != 2 || form.EndColumn != 15 || div.EndLine != 2 || div.EndColumn != 22 || parsedBodyChildren(document)[1].Value != "w" || parsedBodyChildren(document)[1].StartPos != 38 || parsedBodyChildren(document)[1].EndPos != 39 {
		t.Fatalf("Wrong form UTF-8 bytes / UTF-16 coordinates: form=%#v div=%#v doc=%#v", form, div, parsedBodyChildren(document))
	}

	plain, err := parser.Parse(`z`)
	if err != nil || len(parsedBodyChildren(plain)) != 1 || parsedBodyChildren(plain)[0].Value != "z" || len(formattingElements(plain, "form")) != 0 {
		t.Fatalf("Parser reuse leaked form pointer: doc=%#v err=%v", plain, err)
	}
}

func TestParseFormPointerScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n * 110)
		for i := 0; i < n; i++ {
			b.WriteString(`<form><div>x<form>y</form>z</div></form>`)
			b.WriteString(`<table><form></form><form><tr><td>x</table>z`)
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
	t.Logf("form pointer scaling 1000=%v 4000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 30*time.Millisecond {
		t.Fatalf("Form pointer recovery scaled superlinearly: %.1fx", ratio)
	}
}

func TestParseFormPointerEOFBoundaryMatrix(t *testing.T) {
	tests := []struct {
		name               string
		content            string
		formEnd, divEnd    int
		formText, divText  string
		spanStart, spanEnd int
	}{
		{name: "open form and div", content: `<form><div>x`, formEnd: 12, divEnd: 12, formText: "x", divText: "x"},
		{name: "form end with open div", content: `<form><div>x</form>`, formEnd: 19, divEnd: 19, formText: "x", divText: "x"},
		{name: "text after form end", content: `<form><div>x</form>y`, formEnd: 19, divEnd: 20, formText: "xy", divText: "xy"},
		{name: "unclosed span after form end", content: `<form><div>x</form><span>y`, formEnd: 19, divEnd: 26, formText: "xy", divText: "xy", spanStart: 19, spanEnd: 26},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			form := requireTableAdoptionElements(t, document, "form", 1)[0]
			div := requireTableAdoptionElements(t, form, "div", 1)[0]
			assertFormattingNode(t, form, "form", test.formText, 0, test.formEnd)
			assertFormattingNode(t, div, "div", test.divText, 6, test.divEnd)
			if test.spanStart != 0 {
				span := requireTableAdoptionElements(t, div, "span", 1)[0]
				assertFormattingNode(t, span, "span", "y", test.spanStart, test.spanEnd)
			}
		})
	}

	if document, err := NewHTMLParser().Parse(`<div>x`); err != nil || len(formattingElements(document, "div")) != 1 || formattingElements(document, "div")[0].EndPos != 6 {
		t.Fatalf("Form-to-implicit-document EOF recovery mismatch: doc=%#v err=%v", document, err)
	}
}

func TestParseFormPointerEOFBoundarySurvivesLaterForm(t *testing.T) {
	const content = `<form id=a><div>x</form><form id=b>y</form>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	forms := formattingElements(document, "form")
	if len(forms) != 2 {
		t.Fatalf("Expected original and later form, got %#v", forms)
	}
	first := formByID(t, document, "a")
	second := formByID(t, document, "b")
	div := requireTableAdoptionElements(t, first, "div", 1)[0]
	assertFormattingNode(t, first, "form", "xyz", 0, 24)
	assertFormattingNode(t, div, "div", "xyz", 11, 44)
	assertFormattingNode(t, second, "form", "y", 24, 43)
	if second.Parent != div || div.Children[2].Value != "z" || div.Children[2].StartPos != 43 || div.Children[2].EndPos != 44 {
		t.Fatalf("Later form overwrote pending first-form EOF boundary: first=%#v second=%#v div=%#v", first, second, div.Children)
	}
}

func TestParseFormPointerAdoptionRecovery(t *testing.T) {
	const content = `<form id=f><b>x<div>y</form>z</b>q</div>w`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	form := formByID(t, document, "f")
	bold := formattingElements(document, "b")
	div := requireTableAdoptionElements(t, document, "div", 1)[0]
	if len(bold) != 2 || len(parsedBodyChildren(document)) != 3 {
		t.Fatalf("Expected source b, sibling div with synthetic b, and trailing w: b=%#v doc=%#v", bold, parsedBodyChildren(document))
	}
	assertFormattingNode(t, form, "form", "x", 0, 28)
	assertFormattingNode(t, bold[0], "b", "x", 11, 33)
	assertFormattingNode(t, div, "div", "yzq", 15, 40)
	assertFormattingNode(t, bold[1], "b", "yz", 0, 0)
	if bold[0].Parent != form || div.Parent != parsedBody(document) || bold[1].Parent != div || div.Children[1].Value != "q" || div.Children[1].StartPos != 33 || div.Children[1].EndPos != 34 || parsedBodyChildren(document)[2].Value != "w" || parsedBodyChildren(document)[2].StartPos != 40 || parsedBodyChildren(document)[2].EndPos != 41 {
		t.Fatalf("Form pointer/adoption integration mismatch: form=%#v b=%#v div=%#v doc=%#v", form, bold, div.Children, parsedBodyChildren(document))
	}
}
