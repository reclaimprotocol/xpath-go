package xpath_test

import "testing"

func TestQueryTableAdoptionInsideCell(t *testing.T) {
	const content = `<table><tr><td><b>1<div>2</b>3</div>4</td></tr></table>5`
	assertTableQuery(t, content, `/table`, "1234", 0, 55)
	assertTableQuery(t, content, `/table/tbody`, "1234", 0, 0)
	assertTableQuery(t, content, `/table/tbody/tr`, "1234", 7, 47)
	assertTableQuery(t, content, `/table/tbody/tr/td`, "1234", 11, 42)
	assertTableQuery(t, content, `//td/b`, "1", 15, 29)
	assertTableQuery(t, content, `//td/div`, "23", 19, 36)
	assertTableQuery(t, content, `//td/div/b`, "2", 0, 0)
	assertTableQuery(t, content, `//td/div/b/following-sibling::text()`, "3", 29, 30)
	assertTableQuery(t, content, `//td/div/following-sibling::text()`, "4", 36, 37)
	assertTableQuery(t, content, `/table/following-sibling::text()`, "5", 55, 56)
}

func TestQueryTableAdoptionCanonicalMisnestingInsideCell(t *testing.T) {
	const content = `<table><tr><td><b>1<i>2</b>3</i>4</td></tr></table>5`
	assertTableQuery(t, content, `//td/b`, "12", 15, 27)
	assertTableQuery(t, content, `//td/b/i`, "2", 19, 23)
	assertTableQuery(t, content, `//td/b/following-sibling::i`, "3", 19, 32)
	assertTableQuery(t, content, `//td/i/following-sibling::text()`, "4", 32, 33)
	assertTableQuery(t, content, `/table/following-sibling::text()`, "5", 51, 52)
}

func TestQueryTableCellMarkerIsolatesAdoptionEnd(t *testing.T) {
	const content = `<table><tr><td><b>1<td>2</b>3</table>4`
	assertTableQuery(t, content, `/table`, "123", 0, 37)
	assertTableQuery(t, content, `//td[1]`, "1", 11, 19)
	assertTableQuery(t, content, `//td[1]/b`, "1", 15, 19)
	assertTableQuery(t, content, `//td[2]`, "23", 19, 29)
	assertTableQuery(t, content, `//td[2]/text()`, "23", 23, 29)
	assertFormattingNoQuery(t, content, `//td[2]/b`)
	assertTableQuery(t, content, `/table/following-sibling::text()`, "4", 37, 38)
}

func TestQueryTableAdoptionInsideCaption(t *testing.T) {
	const content = `<table><caption><b>1<div>2</b>3</div>4</caption><tr><td>x</table>5`
	assertTableQuery(t, content, `/table`, "1234x", 0, 65)
	assertTableQuery(t, content, `/table/caption`, "1234", 7, 48)
	assertTableQuery(t, content, `/table/caption/b`, "1", 16, 30)
	assertTableQuery(t, content, `/table/caption/div`, "23", 20, 37)
	assertTableQuery(t, content, `/table/caption/div/b`, "2", 0, 0)
	assertTableQuery(t, content, `/table/caption/div/b/following-sibling::text()`, "3", 30, 31)
	assertTableQuery(t, content, `/table/caption/div/following-sibling::text()`, "4", 37, 38)
	assertTableQuery(t, content, `/table/tbody/tr/td`, "x", 52, 57)
	assertTableQuery(t, content, `/table/following-sibling::text()`, "5", 65, 66)
}

func TestQueryTableAdoptionCanonicalMisnestingInsideCaption(t *testing.T) {
	const content = `<table><caption><b>1<i>2</b>3</i>4</caption><tr><td>x</table>5`
	assertTableQuery(t, content, `//caption/b`, "12", 16, 28)
	assertTableQuery(t, content, `//caption/b/i`, "2", 20, 24)
	assertTableQuery(t, content, `//caption/b/following-sibling::i`, "3", 20, 33)
	assertTableQuery(t, content, `//caption/i/following-sibling::text()`, "4", 33, 34)
	assertTableQuery(t, content, `//td`, "x", 48, 53)
}

func TestQueryTableFosteredAdoptionWithFurthestBlock(t *testing.T) {
	const content = `<table><b>x<div>y</b>z</div><tr><td>q</table>r`
	assertTableQuery(t, content, `/b`, "x", 7, 21)
	assertTableQuery(t, content, `/b/following-sibling::div`, "yz", 11, 28)
	assertTableQuery(t, content, `/div/b`, "y", 0, 0)
	assertTableQuery(t, content, `/div/b/following-sibling::text()`, "z", 21, 22)
	assertTableQuery(t, content, `/table`, "q", 0, 45)
	assertTableQuery(t, content, `/table/following-sibling::text()`, "r", 45, 46)
}

func TestQueryTablePreMarkerFormattingSurvivesWithoutEnteringCell(t *testing.T) {
	const content = `<table><b>A<tr><td>B</td></tr>C</table>D</b>E`
	assertTableQuery(t, content, `/b[1]`, "A", 7, 11)
	assertTableQuery(t, content, `/b[2]`, "C", 7, 31)
	assertTableQuery(t, content, `/table`, "B", 0, 39)
	assertTableQuery(t, content, `//td`, "B", 15, 25)
	assertFormattingNoQuery(t, content, `//td/b`)
	assertTableQuery(t, content, `/b[3]`, "D", 7, 44)
	assertTableQuery(t, content, `/b[3]/following-sibling::text()`, "E", 44, 45)
}

func TestQueryTablePreMarkerAnchorAllowsIndependentCellAnchor(t *testing.T) {
	const content = `<table><a id=old>A<tr><td><a id=new>B</a>C</td></tr>D</table>E`
	assertTableQuery(t, content, `/a[@id='old'][1]`, "A", 7, 18)
	assertTableQuery(t, content, `/a[@id='old'][2]`, "D", 7, 53)
	assertTableQuery(t, content, `//td/a[@id='new']`, "B", 26, 41)
	assertFormattingNoQuery(t, content, `//td/a[@id='new']/a`)
	assertTableQuery(t, content, `/a[@id='old'][3]`, "E", 7, 62)
	assertTableQuery(t, content, `//td/a/following-sibling::text()`, "C", 41, 42)
}

func TestQueryTableCaptionMarkerClearsFormattingBeforeCell(t *testing.T) {
	const content = `<table><caption><b>x<tbody><tr><td>y</b>z</table>q`
	assertTableQuery(t, content, `/table/caption`, "x", 7, 20)
	assertTableQuery(t, content, `/table/caption/b`, "x", 16, 20)
	assertTableQuery(t, content, `//td`, "yz", 31, 41)
	assertTableQuery(t, content, `//td/text()`, "yz", 35, 41)
	assertFormattingNoQuery(t, content, `//td/b`)
	assertTableQuery(t, content, `/table/following-sibling::text()`, "q", 49, 50)
}

func TestQueryNestedTableFosterFormattingStaysInOuterCell(t *testing.T) {
	const content = `<table><tr><td><b>A<table><i>B</i><tr><td>C</td></tr>D</table>E</b></td></tr></table>F`
	assertTableQuery(t, content, `/table/tbody/tr/td/b`, "ABDCE", 15, 67)
	assertTableQuery(t, content, `/table/tbody/tr/td/b/i`, "B", 26, 34)
	assertTableQuery(t, content, `/table/tbody/tr/td/b/table`, "C", 19, 62)
	assertTableQuery(t, content, `/table/tbody/tr/td/b/table/preceding-sibling::text()[last()]`, "A", 18, 19)
	assertTableQuery(t, content, `/table/tbody/tr/td/b/table/preceding-sibling::text()[1]`, "D", 53, 54)
	assertTableQuery(t, content, `/table/tbody/tr/td/b/table/following-sibling::text()[1]`, "E", 62, 63)
	assertTableQuery(t, content, `/table/following-sibling::text()`, "F", 85, 86)
}

func TestQueryTableFormattingWhitespaceDoesNotReconstructOrFoster(t *testing.T) {
	const literal = `<table><b><tr><td>a</td></tr>   </table>x`
	assertTableQuery(t, literal, `/b[1]`, "", 7, 10)
	assertTableQuery(t, literal, `/table/tbody/text()`, "   ", 29, 32)
	assertTableQuery(t, literal, `/table/following-sibling::b`, "x", 7, 41)
	assertFormattingNoQuery(t, literal, `/table/preceding-sibling::b[normalize-space(.)!='']`)

	const numeric = `<table><b><tr><td>a</td></tr>&#32;</table>x`
	assertTableQuery(t, numeric, `/table/tbody/text()`, " ", 29, 34)
	assertTableQuery(t, numeric, `/table/following-sibling::b`, "x", 7, 43)
}

func TestQueryTableFormattingNonspaceReferenceRunReconstructsAndFosters(t *testing.T) {
	const content = `<table><b><tr><td>a</td></tr>&#32;&#65;</table>x`
	assertTableQuery(t, content, `/b[1]`, "", 7, 10)
	assertTableQuery(t, content, `/b[2]`, " A", 7, 39)
	assertTableQuery(t, content, `/b[2]/text()`, " A", 29, 39)
	assertTableQuery(t, content, `/table`, "a", 0, 47)
	assertTableQuery(t, content, `/table/following-sibling::b`, "x", 7, 48)
}

func TestQueryFormattingStartedOutsideTableUsesTableScope(t *testing.T) {
	const content = `<b>1<table>2</b>3<tr><td>x</td></tr></table>4`
	assertTableQuery(t, content, `/b`, "123x4", 0, 45)
	assertTableQuery(t, content, `/b/text()[1]`, "123", 3, 17)
	assertTableQuery(t, content, `/b/table`, "x", 4, 44)
	assertTableQuery(t, content, `/b/table/following-sibling::text()`, "4", 44, 45)
}

func TestQueryAnchorAndNobrStartedOutsideTableUseDistinctSpecialStartRules(t *testing.T) {
	const anchor = `<a id=o>1<table>2<a id=n>3</a>4<tr><td>x</td></tr></table>5</a>6`
	assertTableQuery(t, anchor, `/a[@id='o']`, "1234x", 0, 17)
	assertTableQuery(t, anchor, `/a[@id='o']/a[@id='n']`, "3", 17, 30)
	assertTableQuery(t, anchor, `/a[@id='o']/table`, "x", 9, 58)
	assertTableQuery(t, anchor, `/a/following-sibling::text()`, "56", 58, 64)

	const nobr = `<nobr>1<table>2<nobr>3</nobr>4<tr><td>x</td></tr></table>5</nobr>6`
	assertTableQuery(t, nobr, `/nobr`, "1234x5", 0, 65)
	assertTableQuery(t, nobr, `/nobr/nobr`, "3", 15, 29)
	assertTableQuery(t, nobr, `/nobr/table`, "x", 7, 57)
	assertTableQuery(t, nobr, `/nobr/table/following-sibling::text()`, "5", 57, 58)
	assertTableQuery(t, nobr, `/nobr/following-sibling::text()`, "6", 65, 66)
}

func TestQueryTableFosteredAdoptionMultibyteLocations(t *testing.T) {
	const content = "<table><b>é\r\n<div>😀</b>z</div><tr><td>x</table>w"
	assertTableQuery(t, content, `/b`, "é\n", 7, 27)
	assertTableQuery(t, content, `/div`, "😀z", 14, 34)
	assertTableQuery(t, content, `/div/b`, "😀", 0, 0)
	assertTableQuery(t, content, `/div/b/following-sibling::text()`, "z", 27, 28)
	assertTableQuery(t, content, `/table`, "x", 0, 51)
	assertTableQuery(t, content, `/table/following-sibling::text()`, "w", 51, 52)
}

func TestQueryTableAdoptionIncompleteCellEnd(t *testing.T) {
	const content = `<table><tr><td><b>1<div>2</b`
	assertTableQuery(t, content, `//td/b`, "12", 15, len(content))
	assertTableQuery(t, content, `//td/b/div`, "2", 19, len(content))
	assertFormattingNoQuery(t, content, `//td/div/b`)
}

func TestQueryTableFosteredIncompleteEndFlushesAtEOFWithoutSurgery(t *testing.T) {
	const content = `<table><b>x<div>y</b`
	assertTableQuery(t, content, `/b`, "xy", 7, len(content))
	assertTableQuery(t, content, `/b/div`, "y", 11, len(content))
	assertTableQuery(t, content, `/table`, "", 0, len(content))
	assertFormattingNoQuery(t, content, `/b/div/b`)
}

func TestQueryTableBodyAndRowFosteredAdoption(t *testing.T) {
	const tbody = `<table><tbody><b>x<div>y</b>z</div><tr><td>q</table>r`
	assertTableQuery(t, tbody, `/b`, "x", 14, 28)
	assertTableQuery(t, tbody, `/div`, "yz", 18, 35)
	assertTableQuery(t, tbody, `/div/b`, "y", 0, 0)
	assertTableQuery(t, tbody, `/table`, "q", 0, 52)

	const row = `<table><tr><b>x<div>y</b>z</div><td>q</table>r`
	assertTableQuery(t, row, `/b`, "x", 11, 25)
	assertTableQuery(t, row, `/div`, "yz", 15, 32)
	assertTableQuery(t, row, `/div/b`, "y", 0, 0)
	assertTableQuery(t, row, `/table`, "q", 0, 45)
}

func TestQueryTableCellFontAdoptionPreservesAttributes(t *testing.T) {
	const content = `<table><tr><td><font color=red>1<div>2</font>3</div>4</table>5`
	assertTableQuery(t, content, `//td/font[@color='red']`, "1", 15, 45)
	assertTableQuery(t, content, `//td/div/font[@color='red']`, "2", 0, 0)
	assertTableQuery(t, content, `//td/div`, "23", 32, 52)
	assertTableQuery(t, content, `/table/following-sibling::text()`, "5", 61, 62)
}
