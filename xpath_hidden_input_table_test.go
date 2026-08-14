package xpath_test

import "testing"

func TestQueryHiddenInputPlacementAcrossTableModes(t *testing.T) {
	const direct = `<table><input id=h type=hidden><tr><td>x</table>z`
	assertTableQuery(t, direct, `/table/input[@id='h' and @type='hidden']`, "", 7, 31)
	assertTableQuery(t, direct, `/table/tbody/tr/td`, "x", 35, 40)

	const tbody = `<table><tbody><input id=h type=HIDDEN><tr><td>x</table>z`
	assertTableQuery(t, tbody, `/table/tbody/input[@id='h']`, "", 14, 38)

	const row = `<table><tr><input id=h type=hidd&#101;n><td>x</table>z`
	assertTableQuery(t, row, `/table/tbody/tr/input[@id='h' and @type='hidden']`, "", 11, 40)
}

func TestQueryHiddenAndOrdinaryInputCombinedTableOrder(t *testing.T) {
	const content = `<div id=p></div><table id=t><input type=hidden id=h><input id=o><tr><td>x</table>`
	assertTableQuery(t, content, `/div`, "", 0, 16)
	assertTableQuery(t, content, `/div/following-sibling::input[@id='o']`, "", 52, 64)
	assertTableQuery(t, content, `/table[@id='t']/input[@id='h']`, "", 28, 52)
	assertFormattingNoQuery(t, content, `/table/input[@id='o']`)
}

func TestQueryHiddenInputAfterExplicitCellCloseStaysInRow(t *testing.T) {
	const content = `<table><tr><td>x</td><input type=hidden id=h><td>y</table>z`
	assertTableQuery(t, content, `//tr/td[1]`, "x", 11, 21)
	assertTableQuery(t, content, `//tr/input[@id='h']`, "", 21, 45)
	assertTableQuery(t, content, `//tr/td[2]`, "y", 45, 50)
}

func TestQueryHiddenInputDoesNotReconstructOffStackFormatting(t *testing.T) {
	const content = `<table><b><tr><td>x</td></tr><input type=hidden id=h><!--c--><input id=o></table>z`
	assertTableQuery(t, content, `/b[1]`, "", 7, 10)
	assertTableQuery(t, content, `/b[2]/input[@id='o']`, "", 61, 73)
	assertTableQuery(t, content, `/table/tbody/input[@id='h']`, "", 29, 53)
	assertTableQuery(t, content, `/table/tbody/node()[last()]`, "c", 53, 61)
	assertTableQuery(t, content, `/table/following-sibling::b`, "z", 7, 82)
}

func TestQueryHiddenInputInOpenFosterFormattingUsesCurrentNode(t *testing.T) {
	const content = `<table><b><input type=hidden id=h><input id=o><tr><td>x</table>z`
	assertTableQuery(t, content, `/b[1]/input[@id='h']`, "", 10, 34)
	assertTableQuery(t, content, `/b[1]/input[@id='o']`, "", 34, 46)
	assertTableQuery(t, content, `/table`, "x", 0, 63)
	assertTableQuery(t, content, `/table/following-sibling::b`, "z", 7, 64)
}

func TestQueryHiddenInputClosesColgroupAndStaysInTable(t *testing.T) {
	const content = `<table><colgroup id=c><input type=hidden id=h><col id=k></table>z`
	assertTableQuery(t, content, `/table/colgroup[@id='c']`, "", 7, 22)
	assertTableQuery(t, content, `/table/input[@id='h']`, "", 22, 46)
	assertTableQuery(t, content, `/table/colgroup[not(@id)]`, "", 0, 0)
	assertTableQuery(t, content, `/table/colgroup/col[@id='k']`, "", 46, 56)
}

func TestQueryNestedTableHiddenAndOrdinaryInputPlacement(t *testing.T) {
	const content = `<table><tr><td><table><input type=hidden id=h><input id=o><tr><td>x</table>y</table>z`
	assertTableQuery(t, content, `/table/tbody/tr/td/input[@id='o']`, "", 46, 58)
	assertTableQuery(t, content, `/table/tbody/tr/td/table/input[@id='h']`, "", 22, 46)
	assertTableQuery(t, content, `/table/tbody/tr/td/table`, "x", 15, 75)
	assertTableQuery(t, content, `/table/tbody/tr/td/table/following-sibling::text()`, "y", 75, 76)
}

func TestQueryOrdinaryInputsAreFosteredBeforeTable(t *testing.T) {
	tests := []struct {
		name     string
		startTag string
		typeVal  string
	}{
		{"text", `<input id=t type=text>`, "text"},
		{"missing", `<input id=t>`, ""},
		{"boolean-hidden", `<input hidden id=t>`, ""},
		{"spaces", `<input id=t type=' hidden '>`, " hidden "},
		{"crlf", "<input id=t type='hid\r\nden'>", "hid\nden"},
		{"nul", "<input id=t type=hid\x00den>", "hid�den"},
		{"non-ascii", `<input id=t type=hıdden>`, "hıdden"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content := `<table>` + test.startTag + `<tr><td>x</table>z`
			expression := `/input[@id='t']`
			if test.typeVal != "" {
				expression = `/input[@id='t' and @type="` + test.typeVal + `"]`
			}
			assertTableQuery(t, content, expression, "", 7, 7+len([]byte(test.startTag)))
			assertTableQuery(t, content, `/input/following-sibling::table`, "x", 0, len(content)-1)
		})
	}
}

func TestQueryHiddenInputSectionPlacement(t *testing.T) {
	const content = `<table><tbody id=a><tr><td>x</td></tr><input type=hidden id=h></tbody><input type=hidden id=i><tfoot id=f><input type=hidden id=j><tr><td>y</table>`
	assertTableQuery(t, content, `/table/tbody/input[@id='h']`, "", 38, 62)
	assertTableQuery(t, content, `/table/input[@id='i']`, "", 70, 94)
	assertTableQuery(t, content, `/table/tfoot/input[@id='j']`, "", 106, 130)
}

func TestQueryHiddenInputInOpenFosterDivUsesCurrentNode(t *testing.T) {
	const content = `<table><div id=d><input type=hidden id=h><input id=o><tr><td>x</table>`
	assertTableQuery(t, content, `/div[@id='d']`, "", 7, 53)
	assertTableQuery(t, content, `/div/input[@id='h']`, "", 17, 41)
	assertTableQuery(t, content, `/div/input[@id='o']`, "", 41, 53)
	assertTableQuery(t, content, `/div/following-sibling::table`, "x", 0, 70)
}

func TestQueryHiddenAndOrdinaryInputInsideCellFormattingUseInBodyMode(t *testing.T) {
	const content = `<table><tr><td><b><input type=hidden id=h><input id=o>x</table>`
	assertTableQuery(t, content, `//td/b`, "x", 15, 55)
	assertTableQuery(t, content, `//td/b/input[@id='h']`, "", 18, 42)
	assertTableQuery(t, content, `//td/b/input[@id='o']`, "", 42, 54)
}

func TestQueryHiddenInputUnexpectedSolidusAttributeRemainsHidden(t *testing.T) {
	const content = `<table><input type=hidden /foo=bar><tr><td>x</table>`
	assertTableQuery(t, content, `/table/input[@type='hidden' and @foo='bar']`, "", 7, 35)
}

func TestQueryHiddenInputDuplicateAttributesUseFirstValue(t *testing.T) {
	const hiddenFirst = `<table><input id=h type=hidden type=text><tr><td>x</table>z`
	assertTableQuery(t, hiddenFirst, `/table/input[@type='hidden']`, "", 7, 41)
	assertFormattingNoQuery(t, hiddenFirst, `/input[@type='text']`)

	const textFirst = `<table><input id=t type=text type=hidden><tr><td>x</table>z`
	assertTableQuery(t, textFirst, `/input[@type='text']`, "", 7, 41)
	assertFormattingNoQuery(t, textFirst, `/table/input`)
}

func TestQueryHiddenInputWithActiveFormatting(t *testing.T) {
	const content = `<table><b>a<input id=h type=hidden>b<tr><td>x</table>c`
	assertTableQuery(t, content, `/b[1]`, "ab", 7, 36)
	assertTableQuery(t, content, `/b[1]/input`, "", 11, 35)
	assertTableQuery(t, content, `/table`, "x", 0, 53)
	assertTableQuery(t, content, `/table/following-sibling::b`, "c", 7, 54)
}

func TestQueryHiddenInputInCellAndCaption(t *testing.T) {
	const cell = `<table><tr><td><input id=h type=hidden>x</table>z`
	assertTableQuery(t, cell, `//td/input`, "", 15, 39)
	assertTableQuery(t, cell, `//td/input/following-sibling::text()`, "x", 39, 40)

	const caption = `<table><caption><input id=h type=hidden>x</caption><tr><td>y</table>z`
	assertTableQuery(t, caption, `//caption/input`, "", 16, 40)
	assertTableQuery(t, caption, `//caption/input/following-sibling::text()`, "x", 40, 41)
	assertTableQuery(t, caption, `//td`, "y", 55, 60)
}

func TestQueryHiddenInputVoidSyntaxIgnoredEndAndEOF(t *testing.T) {
	const separated = `<table><input type=hidden /><tr><td>x</table>z`
	assertTableQuery(t, separated, `/table/input[@type='hidden']`, "", 7, 28)

	const attached = `<table><input type=hidden/><tr><td>x</table>z`
	assertTableQuery(t, attached, `/input[@type='hidden/']`, "", 7, 27)
	assertFormattingNoQuery(t, attached, `/table/input`)

	const ignoredEnd = `<table><input type=hidden></input><tr><td>x</table>z`
	assertTableQuery(t, ignoredEnd, `/table/input`, "", 7, 26)
	assertTableQuery(t, ignoredEnd, `/table/tbody/tr`, "x", 34, 43)

	const incomplete = `<table><input type=hidden`
	assertTableQuery(t, incomplete, `/table`, "", 0, len(incomplete))
	assertFormattingNoQuery(t, incomplete, `//input`)
}

func TestQueryHiddenInputMultibyteLocations(t *testing.T) {
	const content = "<table>\r\n<input id=h type=hidden data-x=é😀><tr><td>x</table>z"
	assertTableQuery(t, content, `/table/input[@data-x='é😀']`, "", 9, 47)
	assertTableQuery(t, content, `/table/tbody/tr`, "x", 47, 56)
	assertTableQuery(t, content, `/table/following-sibling::text()`, "z", 64, 65)
}
