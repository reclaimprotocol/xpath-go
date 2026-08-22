package xpath_test

import "testing"

var remainingAdoptionXPathSubjects = []string{"big", "code", "font", "s", "small", "strike", "tt", "u"}

func TestQueryRemainingAdoptionSubjectsCanonicalMisnesting(t *testing.T) {
	for _, subject := range remainingAdoptionXPathSubjects {
		t.Run(subject, func(t *testing.T) {
			content := `<p><` + subject + `>1<i>2</` + subject + `>3</i>4`
			openLen := len(subject) + 2
			iStart := 3 + openLen + 1
			subjectEndStart := iStart + len(`<i>2`)
			subjectEnd := subjectEndStart + len(`</`+subject+`>`)
			iEnd := subjectEnd + 1 + len(`</i>`)
			assertTableQuery(t, content, `//p/`+subject, "12", 3, subjectEnd)
			assertTableQuery(t, content, `//p/`+subject+`/i`, "2", iStart, subjectEndStart)
			assertTableQuery(t, content, `//p/`+subject+`/following-sibling::i`, "3", iStart, iEnd)
			assertTableQuery(t, content, `//p/i/following-sibling::text()`, "4", iEnd, len(content))
		})
	}
}

func TestQueryRemainingAdoptionSubjectsFurthestBlock(t *testing.T) {
	for _, subject := range remainingAdoptionXPathSubjects {
		t.Run(subject, func(t *testing.T) {
			content := `<` + subject + `>1<div>2</` + subject + `>3</div>4`
			openLen := len(subject) + 2
			endStart := openLen + len(`1<div>2`)
			end := endStart + len(`</`+subject+`>`)
			assertTableQuery(t, content, `/`+subject, "1", 0, end)
			assertTableQuery(t, content, `/div`, "23", openLen+1, len(content)-1)
			assertTableQuery(t, content, `/div/`+subject, "2", 0, 0)
			assertTableQuery(t, content, `/div/`+subject+`/following-sibling::text()`, "3", end, end+1)
		})
	}
}

func TestQueryRemainingAdoptionSubjectsOffStackAndIncompleteEnds(t *testing.T) {
	for _, subject := range remainingAdoptionXPathSubjects {
		t.Run(subject, func(t *testing.T) {
			offStack := `<p><` + subject + `>1<div>2</div></` + subject + `>3`
			end := len(offStack) - 1
			assertTableQuery(t, offStack, `/p/`+subject, "1", 3, 3+len(subject)+3)
			assertTableQuery(t, offStack, `/div/`+subject, "2", 3, 3+len(subject)+9)
			assertTableQuery(t, offStack, `/div/following-sibling::text()`, "3", end, len(offStack))
			assertFormattingNoQuery(t, offStack, `/div/following-sibling::`+subject)

			incomplete := `<` + subject + `>1<div>2</` + subject
			assertTableQuery(t, incomplete, `/`+subject, "12", 0, len(incomplete))
			assertTableQuery(t, incomplete, `/`+subject+`/div`, "2", len(subject)+3, len(incomplete))
			assertFormattingNoQuery(t, incomplete, `/div/`+subject)
		})
	}
}

func TestQueryRemainingAdoptionSubjectsCrossIdentityAndOrdinaryInner(t *testing.T) {
	const cross = `<big><code>1</big>2</code>3`
	assertTableQuery(t, cross, `/big`, "1", 0, 18)
	assertTableQuery(t, cross, `/big/code`, "1", 5, 12)
	assertTableQuery(t, cross, `/big/following-sibling::code`, "2", 5, 26)
	assertTableQuery(t, cross, `/code/following-sibling::text()`, "3", 26, 27)

	const ordinary = `<tt><span>1<div>2</tt>3</span>4</div>5`
	assertTableQuery(t, ordinary, `/tt/span`, "1", 4, 17)
	assertTableQuery(t, ordinary, `/div/tt`, "2", 0, 0)
	assertTableQuery(t, ordinary, `/div/tt/following-sibling::text()`, "34", 22, 31)
	assertTableQuery(t, ordinary, `/div/following-sibling::text()`, "5", 37, 38)
}

func TestQueryRemainingAdoptionSubjectSurvivingFormattingDescendant(t *testing.T) {
	const content = `<s><b>1<div>2</s>3</b>4</div>5`
	assertTableQuery(t, content, `/s`, "1", 0, 17)
	assertTableQuery(t, content, `/s/b`, "1", 3, 6)
	assertTableQuery(t, content, `/s/following-sibling::b[1]`, "", 0, 0)
	assertTableQuery(t, content, `/div`, "234", 7, 29)
	assertTableQuery(t, content, `/div/b`, "23", 0, 0)
	assertTableQuery(t, content, `/div/b/s`, "2", 0, 0)
	assertTableQuery(t, content, `/div/b/s/following-sibling::text()`, "3", 17, 18)
	assertTableQuery(t, content, `/div/b/following-sibling::text()`, "4", 22, 23)
	assertTableQuery(t, content, `/div/following-sibling::text()`, "5", 29, 30)
}

func TestQueryRemainingAdoptionSubjectDeepSurvivingFormattingDescendants(t *testing.T) {
	const content = `<small><b><i>1<div>2</small>3</i>4</b>5</div>6`
	assertTableQuery(t, content, `/small`, "1", 0, 28)
	assertTableQuery(t, content, `/small/b`, "1", 7, 10)
	assertTableQuery(t, content, `/small/b/i`, "1", 10, 13)
	assertTableQuery(t, content, `/small/following-sibling::b[1]`, "", 0, 0)
	assertTableQuery(t, content, `/small/following-sibling::b[1]/i`, "", 0, 0)
	assertTableQuery(t, content, `/div`, "2345", 14, 45)
	assertTableQuery(t, content, `/div/b/i/small`, "2", 0, 0)
	assertTableQuery(t, content, `/div/b/i/small/following-sibling::text()`, "3", 28, 29)
	assertTableQuery(t, content, `/div/b/i/following-sibling::text()`, "4", 33, 34)
	assertTableQuery(t, content, `/div/b/following-sibling::text()`, "5", 38, 39)
	assertTableQuery(t, content, `/div/following-sibling::text()`, "6", 45, 46)
}

func TestQueryRemainingAdoptionSubjectReverseCloseOrderKeepsSurvivingWrapper(t *testing.T) {
	const content = `<u><b>1<div>2</u>3</div>4</b>5`
	assertTableQuery(t, content, `/u`, "1", 0, 17)
	assertTableQuery(t, content, `/u/b`, "1", 3, 6)
	assertTableQuery(t, content, `/u/following-sibling::b`, "234", 0, 0)
	assertTableQuery(t, content, `/u/following-sibling::b/div`, "23", 7, 24)
	assertTableQuery(t, content, `/u/following-sibling::b/div/u`, "2", 0, 0)
	assertTableQuery(t, content, `/u/following-sibling::b/div/u/following-sibling::text()`, "3", 17, 18)
	assertTableQuery(t, content, `/u/following-sibling::b/div/following-sibling::text()`, "4", 24, 25)
	assertTableQuery(t, content, `/u/following-sibling::b/following-sibling::text()`, "5", 29, 30)
	assertFormattingNoQuery(t, content, `/div/following-sibling::b`)
}

func TestQueryRemainingAdoptionSubjectSequentialNestedAdoptionRefreshesText(t *testing.T) {
	const content = `<u><b>1<div>2</b>3</u>4</div>5`
	assertTableQuery(t, content, `/u`, "1", 0, 22)
	assertTableQuery(t, content, `/u/b`, "1", 3, 17)
	assertTableQuery(t, content, `/div`, "234", 7, 29)
	assertTableQuery(t, content, `/div/u`, "23", 0, 0)
	assertTableQuery(t, content, `/div/u/b`, "2", 0, 0)
	assertTableQuery(t, content, `/div/u/b/following-sibling::text()`, "3", 17, 18)
	assertTableQuery(t, content, `/div/u/following-sibling::text()`, "4", 22, 23)
	assertTableQuery(t, content, `/div/following-sibling::text()`, "5", 29, 30)
}

func TestQueryRemainingAdoptionSubjectDoesNotLeakIgnoredEndByName(t *testing.T) {
	const content = `<u><span>1<div>2</u>3<span>x</span>y</div>z`
	assertTableQuery(t, content, `/u`, "1", 0, 20)
	assertTableQuery(t, content, `/u/span`, "1", 3, 16)
	assertTableQuery(t, content, `/div`, "23xy", 10, 42)
	assertTableQuery(t, content, `/div/u`, "2", 0, 0)
	assertTableQuery(t, content, `/div/u/following-sibling::span`, "x", 21, 35)
	assertTableQuery(t, content, `/div/span/following-sibling::text()`, "y", 35, 36)
	assertTableQuery(t, content, `/div/following-sibling::text()`, "z", 42, 43)
	assertFormattingNoQuery(t, content, `/div/span/text()[contains(.,'y')]`)
}

func TestQueryFontAttributeMultibyteAdoptionLocations(t *testing.T) {
	const content = "<font color=red face=x>é\r\n<i>😀</font>z</i>w"
	assertTableQuery(t, content, `/font[@color='red' and @face='x']`, "é\n😀", 0, 41)
	assertTableQuery(t, content, `/font/i`, "😀", 27, 34)
	assertTableQuery(t, content, `/font/following-sibling::i`, "z", 27, 46)
	assertTableQuery(t, content, `/i/following-sibling::text()`, "w", 46, 47)
}

func TestQueryFontAdoptionClonePreservesAttributes(t *testing.T) {
	const content = `<font color=red face=x>1<div>2</font>3</div>4`
	assertTableQuery(t, content, `/font[@color='red' and @face='x']`, "1", 0, 37)
	assertTableQuery(t, content, `/div/font[@color='red' and @face='x']`, "2", 0, 0)
	assertTableQuery(t, content, `/div/font/following-sibling::text()`, "3", 37, 38)
	assertTableQuery(t, content, `/div/following-sibling::text()`, "4", 44, 45)
}

func TestQueryRemainingAdoptionSubjectsAbsentEndsAggregate(t *testing.T) {
	const content = `<p>a</big>b</code>c</font>d</s>e</small>f</strike>g</tt>h</u>i`
	assertTableQuery(t, content, `//p`, "abcdefghi", 0, len(content))
	assertTableQuery(t, content, `//p/text()`, "abcdefghi", 3, len(content))
}

func TestQueryRemainingAdoptionSubjectSyntaxEdges(t *testing.T) {
	const uppercase = `<p><font>x</FONT>y`
	assertTableQuery(t, uppercase, `//p/font`, "x", 3, 17)
	assertTableQuery(t, uppercase, `//p/font/following-sibling::text()`, "y", 17, 18)

	const endAttrs = `<font>x</font ignored=yes/>y`
	assertTableQuery(t, endAttrs, `/font`, "x", 0, 27)
	assertTableQuery(t, endAttrs, `/font/following-sibling::text()`, "y", 27, 28)

	const selfClosing = `<u/>x</u>y`
	assertTableQuery(t, selfClosing, `/u`, "x", 0, 9)
	assertTableQuery(t, selfClosing, `/u/following-sibling::text()`, "y", 9, 10)
}
