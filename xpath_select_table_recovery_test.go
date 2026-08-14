package xpath_test

import "testing"

func TestQuerySelectFosteredFromTableAndRow(t *testing.T) {
	const direct = `<table><select><option>x</select><tr><td>y</table>`
	assertTableQuery(t, direct, `//select`, "x", 7, 33)
	assertTableQuery(t, direct, `//select/option`, "x", 15, 24)
	assertTableQuery(t, direct, `//select/following-sibling::table`, "y", 0, 50)
	assertTableQuery(t, direct, `//table/tbody`, "y", 0, 0)
	assertTableQuery(t, direct, `//table/tbody/tr/td`, "y", 37, 42)

	const row = `<table><tr><select><option>x</select><td>y</table>`
	assertTableQuery(t, row, `//select`, "x", 11, 37)
	assertTableQuery(t, row, `//select/option`, "x", 19, 28)
	assertTableQuery(t, row, `//select/following-sibling::table/tbody/tr`, "y", 7, 42)
}

func TestQueryNestedSelectStartClosesFosteredOuterAndPreservesOrder(t *testing.T) {
	const content = `<div><table><select id=a>x<select id=b>y</select>z<tr><td>c</table></div>`
	assertTableQuery(t, content, `//div/select`, "x", 12, 26)
	assertTableQuery(t, content, `//div/select/following-sibling::text()`, "yz", 39, 50)
	assertTableQuery(t, content, `//div/select/following-sibling::table`, "c", 5, 67)
	assertTableQuery(t, content, `//div/table/tbody/tr/td`, "c", 54, 59)
}

func TestQuerySelectTableFosterOrderAndImplicitTransitions(t *testing.T) {
	const ordered = `<div>a<table>b<select>c</select>d<tr><td>e</table>f</div>`
	assertTableQuery(t, ordered, `//div/select`, "c", 14, 32)
	assertTableQuery(t, ordered, `//div/select/following-sibling::text()[1]`, "d", 32, 33)
	assertTableQuery(t, ordered, `//div/table`, "e", 6, 50)
	assertTableQuery(t, ordered, `//div/table/following-sibling::text()`, "f", 50, 51)

	const direct = `<table><select><option>x<tr><td>y</table>`
	assertTableQuery(t, direct, `//select`, "x", 7, 24)
	assertTableQuery(t, direct, `//select/option`, "x", 15, 24)
	assertTableQuery(t, direct, `//select/following-sibling::table/tbody/tr/td`, "y", 28, 33)

	const row = `<table><tr><select><option>x<td>y</table>`
	assertTableQuery(t, row, `//select`, "x", 11, 28)
	assertTableQuery(t, row, `//select/following-sibling::table/tbody/tr/td`, "y", 28, 33)
}

func TestQueryDirectTableInputAndNestedSelectFosterTransitions(t *testing.T) {
	const input = `<table><select><option>x<input>y<tr><td>z</table>`
	assertTableQuery(t, input, `//select`, "x", 7, 24)
	assertTableQuery(t, input, `//select/following-sibling::input`, "", 24, 31)
	assertTableQuery(t, input, `//input/following-sibling::text()`, "y", 31, 32)
	assertTableQuery(t, input, `//input/following-sibling::table`, "z", 0, 49)

	const nested = `<table><select><option>x<select>y<tr><td>z</table>`
	assertTableQuery(t, nested, `//select`, "x", 7, 24)
	assertTableQuery(t, nested, `//select/following-sibling::text()`, "y", 32, 33)
	assertTableQuery(t, nested, `//select/following-sibling::table`, "z", 0, 50)
}

func TestQuerySelectInsideCellAndTableTransitions(t *testing.T) {
	const normal = `<table><tr><td><select><option>x</select>y</table>`
	assertTableQuery(t, normal, `//table/tbody/tr/td/select`, "x", 15, 41)
	assertTableQuery(t, normal, `//table/tbody/tr/td/select/option`, "x", 23, 32)
	assertTableQuery(t, normal, `//table/tbody/tr/td/select/following-sibling::text()`, "y", 41, 42)

	const nextCell = `<table><tr><td><select><option>x<td>y</table>`
	assertTableQuery(t, nextCell, `//table/tbody/tr/td[1]/select`, "x", 15, 32)
	assertTableQuery(t, nextCell, `//table/tbody/tr/td[1]/select/option`, "x", 23, 32)
	assertTableQuery(t, nextCell, `//table/tbody/tr/td[2]`, "y", 32, 37)

	const nextRow = `<table><tr><td><select><option>x<tr><td>y</table>`
	assertTableQuery(t, nextRow, `//table/tbody/tr[1]/td/select`, "x", 15, 32)
	assertTableQuery(t, nextRow, `//table/tbody/tr[2]/td`, "y", 36, 41)

	const tableEnd = `<table><tr><td><select><option>x</table>tail`
	assertTableQuery(t, tableEnd, `//table/tbody/tr/td/select`, "x", 15, 32)
	assertTableQuery(t, tableEnd, `//table/following-sibling::text()`, "tail", 40, 44)
}

func TestQuerySelectCellExplicitAndSectionTransitions(t *testing.T) {
	const cellEnd = `<table><tr><td><select><option>x</td><td>y</table>`
	assertTableQuery(t, cellEnd, `//table/tbody/tr/td[1]/select`, "x", 15, 32)
	assertTableQuery(t, cellEnd, `//table/tbody/tr/td[1]`, "x", 11, 37)
	assertTableQuery(t, cellEnd, `//table/tbody/tr/td[2]`, "y", 37, 42)

	const rowEnd = `<table><tr><td><select><option>x</tr><tr><td>y</table>`
	assertTableQuery(t, rowEnd, `//table/tbody/tr[1]/td/select`, "x", 15, 32)
	assertTableQuery(t, rowEnd, `//table/tbody/tr[1]`, "x", 7, 37)
	assertTableQuery(t, rowEnd, `//table/tbody/tr[2]/td`, "y", 41, 46)

	const section = `<table><tbody><tr><td><select><option>x<tfoot><tr><td>y</table>`
	assertTableQuery(t, section, `//table/tbody//select`, "x", 22, 39)
	assertTableQuery(t, section, `//table/tbody`, "x", 7, 39)
	assertTableQuery(t, section, `//table/tfoot`, "y", 39, 55)
	assertTableQuery(t, section, `//table/tfoot/tr/td`, "y", 50, 55)
}

func TestQuerySelectInColgroupFostersAndCaptionRetains(t *testing.T) {
	const colgroup = `<table><colgroup><select><option>x</select><tr><td>y</table>`
	assertTableQuery(t, colgroup, `//select`, "x", 17, 43)
	assertTableQuery(t, colgroup, `//select/following-sibling::table/colgroup`, "", 7, 17)
	assertTableQuery(t, colgroup, `//select/following-sibling::table/tbody/tr/td`, "y", 47, 52)

	const caption = `<table><caption><select><option>x</select>y</caption><tr><td>z</table>`
	assertTableQuery(t, caption, `//table/caption`, "xy", 7, 53)
	assertTableQuery(t, caption, `//table/caption/select`, "x", 16, 42)
	assertTableQuery(t, caption, `//table/tbody/tr/td`, "z", 57, 62)
}

func TestQueryTableRetainedInsideCurrentSelectAndBlocksOuterSelectRules(t *testing.T) {
	const nested = `<select><option>x<table><tr><td>y</table>z</select>`
	assertTableQuery(t, nested, `//select`, "xyz", 0, 51)
	assertTableQuery(t, nested, `//select/option`, "xyz", 8, 42)
	assertTableQuery(t, nested, `//select/option/table`, "y", 17, 41)
	assertTableQuery(t, nested, `//select/option/table/tbody`, "y", 0, 0)
	assertTableQuery(t, nested, `//select/option/table/tbody/tr/td`, "y", 28, 33)

	const blocked = `<select><option>x<table><tr><td>y</select>z<input>w</table>q</select>`
	assertTableQuery(t, blocked, `//select`, "xyzwq", 0, 69)
	assertTableQuery(t, blocked, `//select/option`, "xyzwq", 8, 60)
	assertTableQuery(t, blocked, `//select/option/table`, "yzw", 17, 59)
	assertTableQuery(t, blocked, `//select/option/table//td`, "yzw", 28, 51)
	assertTableQuery(t, blocked, `//select/option/table//td/input`, "", 43, 50)

	const inputBlocked = `<select><option>x<table><tr><td>y<input>z</table>q</select>`
	assertTableQuery(t, inputBlocked, `/select`, "xyzq", 0, 59)
	assertTableQuery(t, inputBlocked, `/select/option/table`, "yz", 17, 49)
	assertTableQuery(t, inputBlocked, `/select/option/table//td/input`, "", 33, 40)
}

func TestQueryNestedSelectAllowedBehindNestedTableScope(t *testing.T) {
	const nestedSelect = `<select><option>x<table><tr><td>y<select><option>z</select>w</table>q</select>`
	assertTableQuery(t, nestedSelect, `/select`, "xyzwq", 0, 78)
	assertTableQuery(t, nestedSelect, `/select/option/table//td/select`, "z", 33, 59)
	assertTableQuery(t, nestedSelect, `/select/option/table//td/select/option`, "z", 41, 50)
}

func TestQueryInputDoesNotCloseOuterSelectBehindNestedTableScope(t *testing.T) {
	const content = `<select><option>x<table><tr><td>y<input>z</table>q</select>`
	assertTableQuery(t, content, `/select`, "xyzq", 0, 59)
	assertTableQuery(t, content, `/select/option/table`, "yz", 17, 49)
	assertTableQuery(t, content, `/select/option/table//td/input`, "", 33, 40)
}

func TestQuerySelectScopeRestoredAfterNestedTableCloses(t *testing.T) {
	const blockedThenRestored = `<table><tr><td><select id=o>a<table></select><tr><td>b</table>c</select></table>`
	assertTableQuery(t, blockedThenRestored, `/table`, "abc", 0, 80)
	assertTableQuery(t, blockedThenRestored, `/table/tbody/tr/td/select`, "abc", 15, 72)
	assertTableQuery(t, blockedThenRestored, `/table/tbody/tr/td/select/table`, "b", 29, 62)
	assertTableQuery(t, blockedThenRestored, `/table/tbody/tr/td/select/table/tbody/tr/td`, "b", 49, 54)

	const normalAfterInner = `<table><tr><td><select>a<table></table>b</select>c</table>`
	assertTableQuery(t, normalAfterInner, `/table/tbody/tr/td/select`, "ab", 15, 49)
	assertTableQuery(t, normalAfterInner, `/table/tbody/tr/td/select/table`, "", 24, 39)
	assertTableQuery(t, normalAfterInner, `/table/tbody/tr/td/select/following-sibling::text()`, "c", 49, 50)
}

func TestQueryNestedTableClosesDirectFosteredSelectButIsRetainedInCell(t *testing.T) {
	const direct = `<div><table><select>a<table><tr><td>b</table>c</select>d</div>`
	assertTableQuery(t, direct, `//div/select`, "a", 12, 21)
	assertTableQuery(t, direct, `//div/table[1]`, "", 5, 21)
	assertTableQuery(t, direct, `//div/table[2]`, "b", 21, 45)
	assertTableQuery(t, direct, `//div/table[2]/tbody/tr/td`, "b", 32, 37)
	assertTableQuery(t, direct, `//div/table[2]/following-sibling::text()`, "cd", 45, 56)
}

func TestQuerySelectTableMultibyteCRLFFosterLocations(t *testing.T) {
	const content = "<div>é\r\n<table>β<select>😀</select>γ<tr><td>δ</table>ω</div>"
	assertTableQuery(t, content, `//div/select`, "😀", 18, 39)
	assertTableQuery(t, content, `//div/select/preceding-sibling::text()`, "é\nβ", 5, 18)
	assertTableQuery(t, content, `//div/select/following-sibling::text()[1]`, "γ", 39, 41)
	assertTableQuery(t, content, `//div/table`, "δ", 9, 59)
	assertTableQuery(t, content, `//div/table/following-sibling::text()`, "ω", 59, 61)
}

func TestQuerySelectTableEOF(t *testing.T) {
	const eof = `<div><table><tr><td><select><option>x`
	assertTableQuery(t, eof, `//div/table`, "x", 5, len(eof))
	assertTableQuery(t, eof, `//div/table/tbody`, "x", 0, 0)
	assertTableQuery(t, eof, `//div/table/tbody/tr/td/select`, "x", 20, len(eof))
	assertTableQuery(t, eof, `//div/table/tbody/tr/td/select/option`, "x", 28, len(eof))
}
