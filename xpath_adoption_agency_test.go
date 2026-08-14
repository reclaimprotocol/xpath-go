package xpath_test

import "testing"

func TestQueryAdoptionAgencyCanonicalMisnestedFormatting(t *testing.T) {
	const content = `<p><b>1<i>2</b>3</i>4`
	assertTableQuery(t, content, `//p`, "1234", 0, 21)
	assertTableQuery(t, content, `//p/b`, "12", 3, 15)
	assertTableQuery(t, content, `//p/b/i`, "2", 7, 11)
	assertTableQuery(t, content, `//p/b/following-sibling::i`, "3", 7, 20)
	assertTableQuery(t, content, `//p/i/following-sibling::text()`, "4", 20, 21)
}

func TestQueryAdoptionAgencyFurthestBlock(t *testing.T) {
	const content = `<b>1<div>2</b>3</div>4`
	assertTableQuery(t, content, `/b`, "1", 0, 14)
	assertTableQuery(t, content, `/div`, "23", 4, 21)
	assertTableQuery(t, content, `/div/b`, "2", 0, 0)
	assertTableQuery(t, content, `/div/b/following-sibling::text()`, "3", 14, 15)
	assertTableQuery(t, content, `/div/following-sibling::text()`, "4", 21, 22)
}

func TestQueryAdoptionAgencyAbsentAndOutOfScopeEnds(t *testing.T) {
	const absent = `<p>x</b>y`
	assertTableQuery(t, absent, `//p`, "xy", 0, 9)
	assertTableQuery(t, absent, `//p/text()`, "xy", 3, 9)

	const outOfScope = `<b><button>x</b>y</button>z`
	assertTableQuery(t, outOfScope, `/b`, "", 0, 16)
	assertTableQuery(t, outOfScope, `/button`, "xy", 3, 26)
	assertTableQuery(t, outOfScope, `/button/b`, "x", 0, 0)
	assertTableQuery(t, outOfScope, `/button/b/following-sibling::text()`, "y", 16, 17)
	assertTableQuery(t, outOfScope, `/button/following-sibling::text()`, "z", 26, 27)
}

func TestQueryAdoptionAgencyEOFAndMultibyteLocations(t *testing.T) {
	const eof = `<p><strong>a<em>b</strong>c`
	assertTableQuery(t, eof, `//p/strong`, "ab", 3, 26)
	assertTableQuery(t, eof, `//p/strong/em`, "b", 12, 17)
	assertTableQuery(t, eof, `//p/strong/following-sibling::em`, "c", 12, 27)

	const multibyte = "<p><b>é\r\n<i>😀</b>z</i>w"
	assertTableQuery(t, multibyte, `//p/b`, "é\n😀", 3, 21)
	assertTableQuery(t, multibyte, `//p/b/i`, "😀", 10, 17)
	assertTableQuery(t, multibyte, `//p/b/following-sibling::i`, "z", 10, 26)
	assertTableQuery(t, multibyte, `//p/i/following-sibling::text()`, "w", 26, 27)
}

func TestQueryAdoptionAgencySurvivingDescendantAndOrdinarySpan(t *testing.T) {
	const surviving = `<p><b>1<div><i>2</b>3</i>4</div>5`
	assertTableQuery(t, surviving, `//p/b`, "1", 3, 7)
	assertTableQuery(t, surviving, `//div/b/i`, "2", 12, 16)
	assertTableQuery(t, surviving, `//div/b/following-sibling::i`, "3", 12, 25)
	assertTableQuery(t, surviving, `//div/i/following-sibling::text()`, "4", 25, 26)
	assertTableQuery(t, surviving, `//div/following-sibling::text()`, "5", 32, 33)

	const span = `<p><b>1<span>2</b>3</span>4`
	assertTableQuery(t, span, `//p/b`, "12", 3, 18)
	assertTableQuery(t, span, `//p/b/span`, "2", 7, 14)
	assertTableQuery(t, span, `//p/b/following-sibling::text()`, "34", 18, 27)
}

func TestQueryAdoptionAgencyDeepBookmarkAndNestedBlocks(t *testing.T) {
	const deep = `<p><b><em><i>1<div>2</b>3</i>4</em>5</div>6`
	assertTableQuery(t, deep, `//p/b/em/i`, "1", 10, 14)
	assertTableQuery(t, deep, `//div/b/em/i`, "2", 10, 20)
	assertTableQuery(t, deep, `//div/b/following-sibling::em`, "34", 6, 35)
	assertTableQuery(t, deep, `//div/b/following-sibling::em/i`, "3", 10, 29)
	assertTableQuery(t, deep, `//div/following-sibling::text()`, "6", 42, 43)

	const nested = `<b>1<div>2<div>3</b>4</div>5</div>`
	assertTableQuery(t, nested, `/div`, "2345", 4, 34)
	assertTableQuery(t, nested, `/div/b`, "2", 0, 0)
	assertTableQuery(t, nested, `/div/div`, "34", 10, 27)
	assertTableQuery(t, nested, `/div/div/b`, "3", 0, 0)
}

func TestQueryAdoptionAgencyAbsentRepeatedEndsAndComments(t *testing.T) {
	const absent = `<p><b>x</strong>y</b>z</b>w`
	assertTableQuery(t, absent, `//p/b`, "xy", 3, 21)
	assertTableQuery(t, absent, `//p/b/text()`, "xy", 6, 17)
	assertTableQuery(t, absent, `//p/b/following-sibling::text()`, "zw", 21, 27)

	const comments = `<b>1<div><!--a-->2</b><!--b-->3</div>4`
	assertTableQuery(t, comments, `/div/b`, "2", 0, 0)
	assertTableQuery(t, comments, `/div/b/node()[1]`, "a", 9, 17)
	assertTableQuery(t, comments, `/div/b/text()`, "2", 17, 18)
	assertTableQuery(t, comments, `/div/node()[2]`, "b", 22, 30)
	assertTableQuery(t, comments, `/div/text()`, "3", 30, 31)
}

func TestQueryAdoptionAgencyAttributeCloneAndIncompleteEnd(t *testing.T) {
	const multibyte = "<p><b title=x>é\r\n<div>😀</b>z</div>w"
	assertTableQuery(t, multibyte, `//p/b`, "é\n", 3, 18)
	assertTableQuery(t, multibyte, `//div/b[@title='x']`, "😀", 3, 31)
	assertTableQuery(t, multibyte, `//div/b/following-sibling::text()`, "z", 31, 32)
	assertTableQuery(t, multibyte, `//div/following-sibling::text()`, "w", 38, 39)

	const incomplete = `<p><b>x</b`
	assertTableQuery(t, incomplete, `//p/b`, "x", 3, 10)
	assertTableQuery(t, incomplete, `//p/b/text()`, "x", 6, 10)
}

func TestQueryAdoptionAgencyCurrentInnerLoopLimit(t *testing.T) {
	const content = `<b><em><foo><foo><foo><aside></b></em>`
	assertTableQuery(t, content, `/b/em/foo/foo/foo`, "", 17, 29)
	assertTableQuery(t, content, `/aside`, "", 22, 38)
	assertTableQuery(t, content, `/aside/b`, "", 0, 0)
}

func TestQueryAdoptionAgencyOuterLoopCapsAtEight(t *testing.T) {
	const content = `<b><div>1<div>2<div>3<div>4<div>5<div>6<div>7<div>8<div>9</b>X`
	assertTableQuery(t, content, `/b`, "", 0, 61)
	assertTableQuery(t, content, `/div/b`, "1", 0, 0)
	assertTableQuery(t, content, `/div/div/div/div/div/div/div/div/b`, "89X", 0, 0)
	assertTableQuery(t, content, `/div/div/div/div/div/div/div/div/b/div`, "9X", 51, 62)
	assertTableQuery(t, content, `/div/div/div/div/div/div/div/div/b/div/text()`, "9X", 56, 62)
	assertFormattingNoQuery(t, content, `/div/div/div/div/div/div/div/div/b/div/b`)
}

func TestQueryAdoptionAgencyOffStackFormattingEndRemovesEntry(t *testing.T) {
	const content = `<p><b>1<div>2</div></b>3`
	assertTableQuery(t, content, `/p/b`, "1", 3, 7)
	assertTableQuery(t, content, `/div/b`, "2", 3, 13)
	assertTableQuery(t, content, `/div/following-sibling::text()`, "3", 23, 24)
	assertFormattingNoQuery(t, content, `/div/following-sibling::b`)
}

func TestQueryAdoptionAgencyInnerLoopRemovesOrdinarySpan(t *testing.T) {
	const content = `<b><span>1<div>2</b>3</span>4</div>5`
	assertTableQuery(t, content, `/b`, "1", 0, 20)
	assertTableQuery(t, content, `/b/span`, "1", 3, 16)
	assertTableQuery(t, content, `/div`, "234", 10, 35)
	assertTableQuery(t, content, `/div/b`, "2", 0, 0)
	assertTableQuery(t, content, `/div/b/following-sibling::text()`, "34", 20, 29)
	assertTableQuery(t, content, `/div/following-sibling::text()`, "5", 35, 36)
}

func TestQueryAdoptionAgencyNoahEvictedEntryUsesGenericEndFallback(t *testing.T) {
	const content = `<b><b><b><b>x</b></b></b><span>y</b>z</span>w`
	assertTableQuery(t, content, `/b`, "xy", 0, 36)
	assertTableQuery(t, content, `/b/b/b/b`, "x", 9, 17)
	assertTableQuery(t, content, `/b/span`, "y", 25, 32)
	assertTableQuery(t, content, `/b/following-sibling::text()`, "zw", 36, 45)
}

func TestQueryAdoptionAgencyNoahGenericFallbackStopsAtSpecialBoundary(t *testing.T) {
	const content = `<b><b><b><b>x</b></b></b><div>y</b>z</div>w`
	assertTableQuery(t, content, `/b`, "xyzw", 0, 43)
	assertTableQuery(t, content, `/b/div`, "yz", 25, 42)
	assertTableQuery(t, content, `/b/div/text()`, "yz", 30, 36)
	assertTableQuery(t, content, `/b/div/following-sibling::text()`, "w", 42, 43)
}

func TestQueryAdoptionAgencyIncompleteFurthestBlockEnds(t *testing.T) {
	const incompleteBlock = `<b>1<div>2</b`
	assertTableQuery(t, incompleteBlock, `/b`, "12", 0, 13)
	assertTableQuery(t, incompleteBlock, `/b/div`, "2", 4, 13)
	assertTableQuery(t, incompleteBlock, `/b/div/text()`, "2", 9, 13)
	assertFormattingNoQuery(t, incompleteBlock, `/div/b`)

	const incompleteDescendant = `<b>1<div><i>2</b`
	assertTableQuery(t, incompleteDescendant, `/b`, "12", 0, 16)
	assertTableQuery(t, incompleteDescendant, `/b/div`, "2", 4, 16)
	assertTableQuery(t, incompleteDescendant, `/b/div/i`, "2", 9, 16)
	assertFormattingNoQuery(t, incompleteDescendant, `/b/div/b`)
}

func TestQueryAdoptionAgencyNestedSpecialImplicitClose(t *testing.T) {
	const content = `<b>1<div>2<section>3</b>4</div>5`
	assertTableQuery(t, content, `/b`, "1", 0, 24)
	assertTableQuery(t, content, `/div`, "234", 4, 31)
	assertTableQuery(t, content, `/div/b`, "2", 0, 0)
	assertTableQuery(t, content, `/div/section`, "34", 10, 25)
	assertTableQuery(t, content, `/div/section/b`, "3", 0, 0)
	assertTableQuery(t, content, `/div/section/b/following-sibling::text()`, "4", 24, 25)
	assertTableQuery(t, content, `/div/following-sibling::text()`, "5", 31, 32)
}
