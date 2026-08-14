package xpath_test

import "testing"

func TestQueryFormPointerIgnoresNestedStart(t *testing.T) {
	const content = `<form id=a><div>1<form id=b>2</form>3</div>4</form>5`
	assertTableQuery(t, content, `/form[@id='a']`, "123", 0, 36)
	assertTableQuery(t, content, `/form/div`, "123", 11, 43)
	assertFormattingNoQuery(t, content, `//form[@id='b']`)
	assertTableQuery(t, content, `/form/following-sibling::text()`, "45", 43, 52)
}

func TestQueryFormStartClosesParagraph(t *testing.T) {
	const content = `<p>x<form id=f>y</form>z`
	assertTableQuery(t, content, `/p`, "x", 0, 4)
	assertTableQuery(t, content, `/form[@id='f']`, "y", 4, 23)
	assertTableQuery(t, content, `/form/following-sibling::text()`, "z", 23, 24)
}

func TestQueryFormEndRemovesExactNonCurrentForm(t *testing.T) {
	const content = `<form id=a><div>x</form>y</div>z`
	assertTableQuery(t, content, `/form`, "xy", 0, 24)
	assertTableQuery(t, content, `/form/div`, "xy", 11, 31)
	assertTableQuery(t, content, `/form/div/text()`, "xy", 16, 25)
	assertTableQuery(t, content, `/form/following-sibling::text()`, "z", 31, 32)
}

func TestQueryFormPointerScopeContrasts(t *testing.T) {
	const object = `<form id=a><object><form id=b>x</form>y</object>z</form>`
	assertTableQuery(t, object, `/form[@id='a']`, "xyz", 0, len(object))
	assertTableQuery(t, object, `/form/object`, "xy", 11, 48)
	assertFormattingNoQuery(t, object, `//form[@id='b']`)

	const button = `<form id=a><button><form id=b>x</form>y</button>z</form>`
	assertTableQuery(t, button, `/form[@id='a']`, "xy", 0, 38)
	assertTableQuery(t, button, `/form/button`, "xy", 11, 48)
	assertFormattingNoQuery(t, button, `//form[@id='b']`)
	assertTableQuery(t, button, `/form/following-sibling::text()`, "z", 48, 49)
}

func TestQueryAbsentFormEndsAreIgnored(t *testing.T) {
	const content = `<div>a</form>b<form id=f>c</form>d</form>e</div>`
	assertTableQuery(t, content, `/div`, "abcde", 0, 48)
	assertTableQuery(t, content, `/div/text()[1]`, "ab", 5, 14)
	assertTableQuery(t, content, `/div/form`, "c", 14, 33)
	assertTableQuery(t, content, `/div/text()[2]`, "de", 33, 42)
}

func TestQueryInTableFormStartInsertsThenPops(t *testing.T) {
	const content = `<table><form id=f><tr><td>x</form>y</table>z`
	assertTableQuery(t, content, `/table/form[@id='f']`, "", 7, 7)
	assertTableQuery(t, content, `/table/tbody/tr/td`, "xy", 22, 35)
	assertTableQuery(t, content, `/table/following-sibling::text()`, "z", 43, 44)
}

func TestQueryTableFormPointerIgnoresDuplicateStart(t *testing.T) {
	const content = `<table><form id=a><form id=b><tr><td>x</form>y</table>z`
	assertTableQuery(t, content, `/table/form[@id='a']`, "", 7, 7)
	assertFormattingNoQuery(t, content, `//form[@id='b']`)
	assertTableQuery(t, content, `//td`, "xy", 33, 46)
}

func TestQueryTableFormPointerPersistsAndClears(t *testing.T) {
	const persists = `<table><form id=a><tr><td>x</table><form id=b>y</form>z`
	assertTableQuery(t, persists, `/table/form[@id='a']`, "", 7, 7)
	assertFormattingNoQuery(t, persists, `//form[@id='b']`)
	assertTableQuery(t, persists, `/table/following-sibling::text()`, "yz", 46, 55)

	const clears = `<table><form id=a><tr><td>x</table></form><form id=b>y</form>z`
	assertTableQuery(t, clears, `/table/form[@id='a']`, "", 7, 7)
	assertTableQuery(t, clears, `/table/following-sibling::form[@id='b']`, "y", 42, 61)
}

func TestQuerySequentialInTableFormsAfterPointerClear(t *testing.T) {
	const content = `<table><form id=a></form><form id=b><tr><td>x</table>z`
	assertTableQuery(t, content, `/table/form[@id='a']`, "", 7, 7)
	assertTableQuery(t, content, `/table/form[@id='b']`, "", 25, 25)
	assertTableQuery(t, content, `//td`, "x", 40, 45)
}

func TestQueryInTableFormAcrossSectionRowAndColgroupModes(t *testing.T) {
	const tbody = `<table><tbody><form id=f><tr><td>x</form>y</table>z`
	assertTableQuery(t, tbody, `/table/tbody/form`, "", 14, 14)
	assertTableQuery(t, tbody, `//td`, "xy", 29, 42)

	const row = `<table><tr><form id=f><td>x</form>y</table>z`
	assertTableQuery(t, row, `/table/tbody/tr/form`, "", 11, 11)
	assertTableQuery(t, row, `//td`, "xy", 22, 35)

	const colgroup = `<table><colgroup id=c><form id=f><col id=k></table>z`
	assertTableQuery(t, colgroup, `/table/colgroup[@id='c']`, "", 7, 22)
	assertTableQuery(t, colgroup, `/table/form`, "", 22, 22)
	assertTableQuery(t, colgroup, `/table/colgroup[not(@id)]`, "", 0, 0)
}

func TestQueryFormPointerInCellAndCaptionUsesInBodyRules(t *testing.T) {
	const cell = `<table><tr><td><form id=f>x<form id=g>y</form>z</table>w`
	assertTableQuery(t, cell, `//td/form[@id='f']`, "xy", 15, 46)
	assertFormattingNoQuery(t, cell, `//form[@id='g']`)
	assertTableQuery(t, cell, `//td/form/following-sibling::text()`, "z", 46, 47)

	const caption = `<table><caption><form id=f>x<form id=g>y</form>z</caption><tr><td>q</table>w`
	assertTableQuery(t, caption, `//caption/form[@id='f']`, "xy", 16, 47)
	assertFormattingNoQuery(t, caption, `//form[@id='g']`)
	assertTableQuery(t, caption, `//caption/form/following-sibling::text()`, "z", 47, 48)
}

func TestQueryInTableFormSyntaxAndIncompleteStart(t *testing.T) {
	const selfClosing = `<table><FORM ID=f /><tr><td>x</table>z`
	assertTableQuery(t, selfClosing, `/table/form[@id='f']`, "", 7, 7)

	const incomplete = `<table><form id=f`
	assertTableQuery(t, incomplete, `/table`, "", 0, len(incomplete))
	assertFormattingNoQuery(t, incomplete, `//form`)
}

func TestQueryOuterFormPointerBlocksInTableForm(t *testing.T) {
	const content = `<form id=o><table><form id=t><tr><td>x</table>y</form>z`
	assertTableQuery(t, content, `/form[@id='o']`, "xy", 0, 54)
	assertTableQuery(t, content, `/form/table`, "x", 11, 46)
	assertFormattingNoQuery(t, content, `//form[@id='t']`)
	assertTableQuery(t, content, `/form/following-sibling::text()`, "z", 54, 55)
}

func TestQueryFormPointerSyntaxEOFAndMultibyte(t *testing.T) {
	const syntax = `<FORM ID=a>x<form id=b>y</FORM>z`
	assertTableQuery(t, syntax, `/form[@id='a']`, "xy", 0, 31)
	assertFormattingNoQuery(t, syntax, `//form[@id='b']`)

	const selfClosing = `<form id=a/>x<form id=b>y</form>z`
	assertTableQuery(t, selfClosing, `/form[@id='a/']`, "xy", 0, 32)
	assertFormattingNoQuery(t, selfClosing, `//form[@id='b']`)

	const incomplete = `<form id=a`
	assertFormattingNoQuery(t, incomplete, `//form`)

	const eof = `<form id=a>x<!--c-->`
	assertTableQuery(t, eof, `/form`, "x", 0, len(eof))
	assertTableQuery(t, eof, `/form/node()[2]`, "c", 12, 20)

	const multibyte = "<form id=a>é\r\n<div>😀</form>z</div>w"
	assertTableQuery(t, multibyte, `/form`, "é\n😀z", 0, 31)
	assertTableQuery(t, multibyte, `/form/div`, "😀z", 15, 38)
	assertTableQuery(t, multibyte, `/form/following-sibling::text()`, "w", 38, 39)
}

func TestQueryFormPointerEOFBoundaryMatrix(t *testing.T) {
	const open = `<form><div>x`
	assertTableQuery(t, open, `/form`, "x", 0, 12)
	assertTableQuery(t, open, `/form/div`, "x", 6, 12)

	const ended = `<form><div>x</form>`
	assertTableQuery(t, ended, `/form`, "x", 0, 19)
	assertTableQuery(t, ended, `/form/div`, "x", 6, 19)

	const trailingText = `<form><div>x</form>y`
	assertTableQuery(t, trailingText, `/form`, "xy", 0, 19)
	assertTableQuery(t, trailingText, `/form/div`, "xy", 6, 20)

	const trailingSpan = `<form><div>x</form><span>y`
	assertTableQuery(t, trailingSpan, `/form`, "xy", 0, 19)
	assertTableQuery(t, trailingSpan, `/form/div`, "xy", 6, 26)
	assertTableQuery(t, trailingSpan, `/form/div/span`, "y", 19, 26)
}

func TestQueryFormPointerEOFBoundarySurvivesLaterForm(t *testing.T) {
	const content = `<form id=a><div>x</form><form id=b>y</form>z`
	assertTableQuery(t, content, `/form[@id='a']`, "xyz", 0, 24)
	assertTableQuery(t, content, `/form[@id='a']/div`, "xyz", 11, 44)
	assertTableQuery(t, content, `/form[@id='a']/div/form[@id='b']`, "y", 24, 43)
	assertTableQuery(t, content, `/form/div/form/following-sibling::text()`, "z", 43, 44)
}

func TestQueryFormPointerAdoptionRecovery(t *testing.T) {
	const content = `<form id=f><b>x<div>y</form>z</b>q</div>w`
	assertTableQuery(t, content, `/form[@id='f']`, "x", 0, 28)
	assertTableQuery(t, content, `/form/b`, "x", 11, 33)
	assertTableQuery(t, content, `/div`, "yzq", 15, 40)
	assertTableQuery(t, content, `/div/b`, "yz", 0, 0)
	assertTableQuery(t, content, `/div/b/following-sibling::text()`, "q", 33, 34)
	assertTableQuery(t, content, `/div/following-sibling::text()`, "w", 40, 41)
}
