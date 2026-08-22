package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func assertSelectQueryNode(t *testing.T, document, expression, wantText string, wantStart, wantEnd int) {
	t.Helper()
	expression = documentBodyExpression(expression)
	results, err := xpath.Query(expression, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != wantText || results[0].StartLocation != wantStart || results[0].EndLocation != wantEnd {
		t.Fatalf("Expected %s text %q at %d:%d, got %#v", expression, wantText, wantStart, wantEnd, results)
	}
}

func TestQueryOptionAndOptgroupStartAndEndRecovery(t *testing.T) {
	const options = `<select><option>a<option>b</select>tail`
	assertSelectQueryNode(t, options, `//select/option[1]`, "a", 8, 17)
	assertSelectQueryNode(t, options, `//select/option[2]`, "b", 17, 26)
	assertSelectQueryNode(t, options, `//select/following-sibling::text()`, "tail", 35, 39)

	const groups = `<select><optgroup label=a><option>x<optgroup label=b><option>y</select>tail`
	assertSelectQueryNode(t, groups, `//select/optgroup[1]`, "x", 8, 35)
	assertSelectQueryNode(t, groups, `//select/optgroup[1]/option`, "x", 26, 35)
	assertSelectQueryNode(t, groups, `//select/optgroup[2]`, "y", 35, 62)
	assertSelectQueryNode(t, groups, `//select/optgroup[2]/option`, "y", 53, 62)

	const absent = `<select>a</option>b</optgroup>c<option>d</select>`
	assertSelectQueryNode(t, absent, `//select/text()`, "abc", 8, 31)
	assertSelectQueryNode(t, absent, `//select/option`, "d", 31, 40)

	const explicit = `<select><option>a</option><optgroup><option>b</option></optgroup>z</select>`
	assertSelectQueryNode(t, explicit, `//select/option`, "a", 8, 26)
	assertSelectQueryNode(t, explicit, `//select/optgroup/option`, "b", 36, 54)
	assertSelectQueryNode(t, explicit, `//select/optgroup/following-sibling::text()`, "z", 65, 66)
}

func TestQueryCurrentSelectKeepsOrdinaryDescendants(t *testing.T) {
	const nested = `<select><option>a<div>x<option>b</select>tail`
	assertSelectQueryNode(t, nested, `//select/option[1]`, "axb", 8, 32)
	assertSelectQueryNode(t, nested, `//select/option/div`, "xb", 17, 32)
	assertSelectQueryNode(t, nested, `//select/option/div/option`, "b", 23, 32)
	assertSelectQueryNode(t, nested, `//select/following-sibling::text()`, "tail", 41, 45)

	const ordinary = `<select>a<div>b</div>c<option>d<span>e</span>f</select>tail`
	assertSelectQueryNode(t, ordinary, `//select/div`, "b", 9, 21)
	assertSelectQueryNode(t, ordinary, `//select/option`, "def", 22, 46)
	assertSelectQueryNode(t, ordinary, `//select/option/span`, "e", 31, 45)

	const nestedGroup = `<select><optgroup><option>a<div>x<optgroup><option>b</select>tail`
	assertSelectQueryNode(t, nestedGroup, `//select/optgroup[1]`, "axb", 8, 52)
	assertSelectQueryNode(t, nestedGroup, `//select/optgroup[1]/option/div/optgroup`, "b", 33, 52)
	assertSelectQueryNode(t, nestedGroup, `//select/optgroup[1]/option/div/optgroup/option`, "b", 43, 52)
}

func TestQueryOptionAndOptgroupEndScopeWithDescendants(t *testing.T) {
	const optionEnd = `<select><option><span>x</option>y</select>`
	assertSelectQueryNode(t, optionEnd, `//select/option`, "x", 8, 32)
	assertSelectQueryNode(t, optionEnd, `//select/option/span`, "x", 16, 23)
	assertSelectQueryNode(t, optionEnd, `//select/option/following-sibling::text()`, "y", 32, 33)

	const groupEndBlocked = `<select><optgroup><div><option>x</optgroup>y</select>`
	assertSelectQueryNode(t, groupEndBlocked, `//select/optgroup`, "xy", 8, 44)
	assertSelectQueryNode(t, groupEndBlocked, `//select/optgroup/div/option`, "xy", 23, 44)
	assertSelectQueryNode(t, groupEndBlocked, `//select/optgroup/div/option/text()`, "xy", 31, 44)
}

func TestQueryCurrentSelectTextareaKeygenAndInputBehavior(t *testing.T) {
	const textarea = `<select><option>a<textarea>b&amp;c</textarea><option>d</select>`
	assertSelectQueryNode(t, textarea, `//select/option[1]`, "ab&c", 8, 45)
	assertSelectQueryNode(t, textarea, `//select/option[1]/textarea`, "b&c", 17, 45)
	assertSelectQueryNode(t, textarea, `//select/option[2]`, "d", 45, 54)

	const keygen = `<select><option>a<keygen name=k><option>b</select>`
	assertSelectQueryNode(t, keygen, `//select/option[1]`, "a", 8, 32)
	assertSelectQueryNode(t, keygen, `//select/option[1]/keygen`, "", 17, 32)
	assertSelectQueryNode(t, keygen, `//select/option[2]`, "b", 32, 41)

	const input = `<select><option>a<input value=x><option>b</select>tail`
	assertSelectQueryNode(t, input, `//select`, "a", 0, 17)
	assertSelectQueryNode(t, input, `//select/following-sibling::input`, "", 17, 32)
	assertSelectQueryNode(t, input, `//select/following-sibling::option`, "btail", 32, 54)
}

func TestQueryNestedSelectSelectEndCommentsAndScript(t *testing.T) {
	const nested = `<select><option>a<select><option>b</select>tail`
	assertSelectQueryNode(t, nested, `//select`, "a", 0, 17)
	assertSelectQueryNode(t, nested, `//select/option`, "a", 8, 17)
	assertSelectQueryNode(t, nested, `/option`, "btail", 25, 47)

	const explicit = `<div><select><option>a</select><p>b</p></div>`
	assertSelectQueryNode(t, explicit, `//div/select`, "a", 5, 31)
	assertSelectQueryNode(t, explicit, `//div/select/option`, "a", 13, 22)
	assertSelectQueryNode(t, explicit, `//div/select/following-sibling::p`, "b", 31, 39)

	const allowed = `<select>a<!--c--><script>x<y</script><option>b</select>`
	assertSelectQueryNode(t, allowed, `//select/script`, "x<y", 17, 37)
	assertSelectQueryNode(t, allowed, `//select/option`, "b", 37, 46)
}

func TestQuerySelectEOFAndMultibyteCRLFLocations(t *testing.T) {
	const eof = `<div><select><optgroup><option>x`
	assertSelectQueryNode(t, eof, `//div/select`, "x", 5, len(eof))
	assertSelectQueryNode(t, eof, `//div/select/optgroup`, "x", 13, len(eof))
	assertSelectQueryNode(t, eof, `//div/select/optgroup/option`, "x", 23, len(eof))

	const multiline = "<select>\r\n<option>😀é\r\n<span>x<option>z</select>"
	assertSelectQueryNode(t, multiline, `//select/option`, "😀é\nxz", 10, 42)
	assertSelectQueryNode(t, multiline, `//select/option/span`, "xz", 26, 42)
	assertSelectQueryNode(t, multiline, `//select/option/span/option`, "z", 33, 42)
}

func TestQuerySelectSelfClosingAndEndTagAttributes(t *testing.T) {
	const selfClosing = `<select><option/>a<option>b</select>`
	assertSelectQueryNode(t, selfClosing, `//select/option[1]`, "a", 8, 18)
	assertSelectQueryNode(t, selfClosing, `//select/option[2]`, "b", 18, 27)

	const endAttributes = `<select><option>a</option x/><optgroup><option>b</optgroup y/></select>`
	assertSelectQueryNode(t, endAttributes, `//select/option`, "a", 8, 29)
	assertSelectQueryNode(t, endAttributes, `//select/optgroup`, "b", 29, 62)
	assertSelectQueryNode(t, endAttributes, `//select/optgroup/option`, "b", 39, 48)
}

func TestQueryCurrentSelectOptionEndScopeAndInBodyBlocks(t *testing.T) {
	const blockedEnd = `<select><option><div>x</option>y</select>`
	assertSelectQueryNode(t, blockedEnd, `//select/option`, "xy", 8, 32)
	assertSelectQueryNode(t, blockedEnd, `//select/option/div`, "xy", 16, 32)
	assertSelectQueryNode(t, blockedEnd, `//select/option/div/text()`, "xy", 21, 32)

	const blocks = `<select><option>a<p>b<li>c<button>d<option>e</select>`
	assertSelectQueryNode(t, blocks, `//select/option[1]`, "abcde", 8, 44)
	assertSelectQueryNode(t, blocks, `//select/option/p`, "b", 17, 21)
	assertSelectQueryNode(t, blocks, `//select/option/li/button/option`, "e", 35, 44)
}

func TestQuerySelectRootASCIIHRCharacterTokensAndIncompleteEOF(t *testing.T) {
	const root = `<option>a<option>b`
	assertSelectQueryNode(t, root, `/option[1]`, "a", 0, 9)
	assertSelectQueryNode(t, root, `/option[2]`, "b", 9, 18)

	const uppercase = `<SELECT><OPTION>a<OPTION>b</SELECT>`
	assertSelectQueryNode(t, uppercase, `//select/option[1]`, "a", 8, 17)
	assertSelectQueryNode(t, uppercase, `//select/option[2]`, "b", 17, 26)

	const hr = `<select><option>a<hr><option>b</select>`
	assertSelectQueryNode(t, hr, `//select/option[1]`, "a", 8, 17)
	assertSelectQueryNode(t, hr, `//select/hr`, "", 17, 21)
	assertSelectQueryNode(t, hr, `//select/option[2]`, "b", 21, 30)

	const characters = "<select>a&amp;&#65;\x00<option>b</select>"
	assertSelectQueryNode(t, characters, `//select/text()`, "a&A", 8, 20)

	const incomplete = `<select><option>x<option`
	assertSelectQueryNode(t, incomplete, `//select/option`, "x", 8, len(incomplete))
	assertSelectQueryNode(t, incomplete, `//select/option/text()`, "x", 16, len(incomplete))
}

func TestQuerySelectScopeBoundaryProtectsOuterRecoveryState(t *testing.T) {
	const list = `<ul><li>a<select></li><li>b</select>c</li></ul>`
	assertSelectQueryNode(t, list, `//ul/li`, "abc", 4, 42)
	assertSelectQueryNode(t, list, `//ul/li/select`, "b", 9, 36)
	assertSelectQueryNode(t, list, `//ul/li/select/li`, "b", 22, 27)

	const definition = `<dl><dd>a<select></dd><dt>b</select>c</dd></dl>`
	assertSelectQueryNode(t, definition, `//dl/dd`, "abc", 4, 42)
	assertSelectQueryNode(t, definition, `//dl/dd/select/dt`, "b", 22, 27)

	const paragraph = `<p>a<select><div>b</div></select>c</p>`
	assertSelectQueryNode(t, paragraph, `//p`, "abc", 0, 38)
	assertSelectQueryNode(t, paragraph, `//p/select/div`, "b", 12, 24)

	const button = `<button>a<select><button>b</button></select>c</button>`
	assertSelectQueryNode(t, button, `/button`, "abc", 0, 54)
	assertSelectQueryNode(t, button, `/button/select/button`, "b", 17, 35)

	const generic = `<div>a<select></div>b</select>c</div>`
	assertSelectQueryNode(t, generic, `//div`, "abc", 0, 37)
	assertSelectQueryNode(t, generic, `//div/select/text()`, "b", 20, 21)
}

func TestQuerySelectIgnoresBodyHTMLEndsAndSynthesizesStrayParagraph(t *testing.T) {
	const bodyHTML = `<html><body><select>a</body>b</html>c</select><p>d</p>`
	assertSelectQueryNode(t, bodyHTML, `//body/select`, "abc", 12, 46)
	assertSelectQueryNode(t, bodyHTML, `//body/select/text()`, "abc", 20, 37)
	assertSelectQueryNode(t, bodyHTML, `//body/select/following-sibling::p`, "d", 46, 54)

	const strayP = `<p>a<select></p>b</select>c</p>`
	assertSelectQueryNode(t, strayP, `/p`, "abc", 0, 31)
	assertSelectQueryNode(t, strayP, `/p/select/p`, "", 0, 0)
	assertSelectQueryNode(t, strayP, `/p/select/p/following-sibling::text()`, "b", 16, 17)
}

func TestQuerySelectStartsOnlyGenerateImpliedEnds(t *testing.T) {
	const optionSpan = `<select><option><span>x<option>y</select>`
	assertSelectQueryNode(t, optionSpan, `//select/option`, "xy", 8, 32)
	assertSelectQueryNode(t, optionSpan, `//select/option/span/option`, "y", 23, 32)

	const optionP = `<select><option><p>x<option>y</select>`
	assertSelectQueryNode(t, optionP, `//select/option[1]`, "x", 8, 20)
	assertSelectQueryNode(t, optionP, `//select/option[1]/p`, "x", 16, 20)
	assertSelectQueryNode(t, optionP, `//select/option[2]`, "y", 20, 29)

	const optionLI = `<select><option><li>x<option>y</select>`
	assertSelectQueryNode(t, optionLI, `//select/option[1]`, "x", 8, 21)
	assertSelectQueryNode(t, optionLI, `//select/option[2]`, "y", 21, 30)

	const groupSpan = `<select><optgroup><span>x<optgroup>y</select>`
	assertSelectQueryNode(t, groupSpan, `//select/optgroup`, "xy", 8, 36)
	assertSelectQueryNode(t, groupSpan, `//select/optgroup/span/optgroup`, "y", 25, 36)

	const groupP = `<select><optgroup><p>x<optgroup>y</select>`
	assertSelectQueryNode(t, groupP, `//select/optgroup[1]`, "x", 8, 22)
	assertSelectQueryNode(t, groupP, `//select/optgroup[2]`, "y", 22, 33)

	const hrInsideDiv = `<select><option><div>x<hr>y</select>`
	assertSelectQueryNode(t, hrInsideDiv, `//select/option/div`, "xy", 16, 27)
	assertSelectQueryNode(t, hrInsideDiv, `//select/option/div/hr`, "", 22, 26)
}

func TestQueryStandaloneOptionAndOptgroupStartsOnlyPopCurrentNode(t *testing.T) {
	const paragraph = `<option><p>x<option>y`
	assertSelectQueryNode(t, paragraph, `/option`, "xy", 0, len(paragraph))
	assertSelectQueryNode(t, paragraph, `/option/p`, "xy", 8, len(paragraph))
	assertSelectQueryNode(t, paragraph, `/option/p/option`, "y", 12, len(paragraph))

	const listItem = `<option><li>x<option>y`
	assertSelectQueryNode(t, listItem, `/option/li`, "xy", 8, len(listItem))
	assertSelectQueryNode(t, listItem, `/option/li/option`, "y", 13, len(listItem))

	const groups = `<optgroup><p>x<optgroup>y`
	assertSelectQueryNode(t, groups, `/optgroup`, "xy", 0, len(groups))
	assertSelectQueryNode(t, groups, `/optgroup/p`, "xy", 10, len(groups))
	assertSelectQueryNode(t, groups, `/optgroup/p/optgroup`, "y", 14, len(groups))
}

func TestQuerySelectStartsGenerateImpliedEndsWithoutPriorItem(t *testing.T) {
	const paragraphOption = `<select><p>x<option>y`
	assertSelectQueryNode(t, paragraphOption, `//select/p`, "x", 8, 12)
	assertSelectQueryNode(t, paragraphOption, `//select/option`, "y", 12, 21)

	const listOption = `<select><li>x<option>y`
	assertSelectQueryNode(t, listOption, `//select/li`, "x", 8, 13)
	assertSelectQueryNode(t, listOption, `//select/option`, "y", 13, 22)

	const groupedOption = `<select><optgroup><p>x<option>y`
	assertSelectQueryNode(t, groupedOption, `//select/optgroup`, "xy", 8, 31)
	assertSelectQueryNode(t, groupedOption, `//select/optgroup/p`, "x", 18, 22)
	assertSelectQueryNode(t, groupedOption, `//select/optgroup/option`, "y", 22, 31)

	const paragraphGroup = `<select><p>x<optgroup>y`
	assertSelectQueryNode(t, paragraphGroup, `//select/p`, "x", 8, 12)
	assertSelectQueryNode(t, paragraphGroup, `//select/optgroup`, "y", 12, 23)

	const listGroup = `<select><li>x<optgroup>y`
	assertSelectQueryNode(t, listGroup, `//select/li`, "x", 8, 13)
	assertSelectQueryNode(t, listGroup, `//select/optgroup`, "y", 13, 24)

	for _, itemName := range []string{"li", "dd"} {
		document := `<select><` + itemName + `>x<hr>y`
		assertSelectQueryNode(t, document, `//select/`+itemName, "x", 8, 13)
		assertSelectQueryNode(t, document, `//select/hr`, "", 13, 17)
		assertSelectQueryNode(t, document, `//select/hr/following-sibling::text()`, "y", 17, 18)
	}
}

func TestQueryNestedSelectRestoresStandaloneOptionAndOptgroupIdentity(t *testing.T) {
	const optionCase = `<option>a<select><option>b</select>c`
	assertSelectQueryNode(t, optionCase, `/option`, "abc", 0, len(optionCase))
	assertSelectQueryNode(t, optionCase, `/option/select`, "b", 9, 35)
	assertSelectQueryNode(t, optionCase, `/option/select/option`, "b", 17, 26)
	assertSelectQueryNode(t, optionCase, `/option/select/following-sibling::text()`, "c", 35, 36)

	const groupCase = `<optgroup>a<select><optgroup>b</select>c`
	assertSelectQueryNode(t, groupCase, `/optgroup`, "abc", 0, len(groupCase))
	assertSelectQueryNode(t, groupCase, `/optgroup/select`, "b", 11, 39)
	assertSelectQueryNode(t, groupCase, `/optgroup/select/optgroup`, "b", 19, 30)
	assertSelectQueryNode(t, groupCase, `/optgroup/select/following-sibling::text()`, "c", 39, 40)
}

func TestQuerySearchDescendantDoesNotBlockExplicitSelectItemEnd(t *testing.T) {
	const optionCase = `<select><option><search>x</option>y</select>`
	assertSelectQueryNode(t, optionCase, `//select/option`, "x", 8, 34)
	assertSelectQueryNode(t, optionCase, `//select/option/search`, "x", 16, 25)
	assertSelectQueryNode(t, optionCase, `//select/option/following-sibling::text()`, "y", 34, 35)

	const groupCase = `<select><optgroup><search>x</optgroup>y</select>`
	assertSelectQueryNode(t, groupCase, `//select/optgroup`, "x", 8, 38)
	assertSelectQueryNode(t, groupCase, `//select/optgroup/search`, "x", 18, 27)
	assertSelectQueryNode(t, groupCase, `//select/optgroup/following-sibling::text()`, "y", 38, 39)
}

func TestQueryGenericEndsInsideSelectIgnoreAbsentAndCloseMatchingDescendant(t *testing.T) {
	const absent = `<select><option>a</foo>b</select>`
	assertSelectQueryNode(t, absent, `//select/option`, "ab", 8, 24)
	assertSelectQueryNode(t, absent, `//select/option/text()`, "ab", 16, 24)

	const matching = `<select><div><span>x</div>y</select>`
	assertSelectQueryNode(t, matching, `//select/div`, "x", 8, 26)
	assertSelectQueryNode(t, matching, `//select/div/span`, "x", 13, 20)
	assertSelectQueryNode(t, matching, `//select/div/following-sibling::text()`, "y", 26, 27)
}

func TestQuerySelectMatchingEndUsesNearestOpenElementIdentity(t *testing.T) {
	const duplicateSpan = `<select><span>x</span>y</span>z</select>`
	assertSelectQueryNode(t, duplicateSpan, `//select`, "xyz", 0, len(duplicateSpan))
	assertSelectQueryNode(t, duplicateSpan, `//select/span`, "x", 8, 22)
	assertSelectQueryNode(t, duplicateSpan, `//select/span/following-sibling::text()`, "yz", 22, 31)

	const duplicateDiv = `<select><div><div>x</div>y</div>z</div>w</select>`
	assertSelectQueryNode(t, duplicateDiv, `//select`, "xyzw", 0, len(duplicateDiv))
	assertSelectQueryNode(t, duplicateDiv, `//select/div`, "xy", 8, 32)
	assertSelectQueryNode(t, duplicateDiv, `//select/div/div`, "x", 13, 25)
	assertSelectQueryNode(t, duplicateDiv, `//select/div/following-sibling::text()`, "zw", 32, 40)
}

func TestQuerySelectGenericEndCannotCrossSpecialElementBarrier(t *testing.T) {
	const spanBarrier = `<select><span><div>x</span>y</select>`
	assertSelectQueryNode(t, spanBarrier, `//select/span`, "xy", 8, 28)
	assertSelectQueryNode(t, spanBarrier, `//select/span/div`, "xy", 14, 28)
	assertSelectQueryNode(t, spanBarrier, `//select/span/div/text()`, "xy", 19, 28)

	const customBarrier = `<select><foo><section>x</foo>y</select>`
	assertSelectQueryNode(t, customBarrier, `//select/foo`, "xy", 8, 30)
	assertSelectQueryNode(t, customBarrier, `//select/foo/section`, "xy", 13, 30)
	assertSelectQueryNode(t, customBarrier, `//select/foo/section/text()`, "xy", 22, 30)
}

func TestQuerySelectDedicatedSpecialEndUnwindsButGenericEndStopsAtBarrier(t *testing.T) {
	const dedicated = `<select><div><section>x</div>y`
	assertSelectQueryNode(t, dedicated, `//select`, "xy", 0, len(dedicated))
	assertSelectQueryNode(t, dedicated, `//select/div`, "x", 8, 29)
	assertSelectQueryNode(t, dedicated, `//select/div/section`, "x", 13, 23)
	assertSelectQueryNode(t, dedicated, `//select/div/following-sibling::text()`, "y", 29, 30)

	const generic = `<select><foo><section>x</foo>y`
	assertSelectQueryNode(t, generic, `//select`, "xy", 0, len(generic))
	assertSelectQueryNode(t, generic, `//select/foo`, "xy", 8, len(generic))
	assertSelectQueryNode(t, generic, `//select/foo/section`, "xy", 13, len(generic))
	assertSelectQueryNode(t, generic, `//select/foo/section/text()`, "xy", 22, len(generic))
}

func TestQuerySelectItemEndScopesRespectListBoundary(t *testing.T) {
	const listItem = `<select><li><ul>x</li>y</select>`
	assertSelectQueryNode(t, listItem, `//select/li`, "xy", 8, 23)
	assertSelectQueryNode(t, listItem, `//select/li/ul`, "xy", 12, 23)
	assertSelectQueryNode(t, listItem, `//select/li/ul/text()`, "xy", 16, 23)

	const definition = `<select><dd><ul>x</dd>y</select>`
	assertSelectQueryNode(t, definition, `//select/dd`, "x", 8, 22)
	assertSelectQueryNode(t, definition, `//select/dd/ul`, "x", 12, 17)
	assertSelectQueryNode(t, definition, `//select/dd/following-sibling::text()`, "y", 22, 23)

	const term = `<select><dt><ul>x</dt>y</select>`
	assertSelectQueryNode(t, term, `//select/dt`, "x", 8, 22)
	assertSelectQueryNode(t, term, `//select/dt/ul`, "x", 12, 17)
	assertSelectQueryNode(t, term, `//select/dt/following-sibling::text()`, "y", 22, 23)
}

func TestQuerySelectSpecialStartsMergeHTMLAndIgnoreBodyHead(t *testing.T) {
	const htmlStart = `<html id=a><body><select><option>x<html lang=z>y</select></body></html>`
	results, err := xpath.Query(`//html`, htmlStart)
	if err != nil || len(results) != 1 || results[0].Attributes["id"] != "a" || results[0].Attributes["lang"] != "z" {
		t.Fatalf("Expected one html with merged id/lang attributes, got %#v err=%v", results, err)
	}
	assertSelectQueryNode(t, htmlStart, `//select/option`, "xy", 25, 48)
	assertSelectQueryNode(t, htmlStart, `//select/option/text()`, "xy", 33, 48)

	const bodyStart = `<html><body id=a><select><option>x<body class=z>y</select></body></html>`
	results, err = xpath.Query(`//body`, bodyStart)
	if err != nil || len(results) != 1 || results[0].Attributes["id"] != "a" {
		t.Fatalf("Expected one original body, got %#v err=%v", results, err)
	}
	if _, exists := results[0].Attributes["class"]; exists {
		t.Fatalf("Expected Chrome-compatible ignored body attributes behind select, got %#v", results[0].Attributes)
	}
	assertSelectQueryNode(t, bodyStart, `//select/option`, "xy", 25, 49)

	const headStart = `<html><body><select><option>x<head>y</select></body></html>`
	assertSelectQueryNode(t, headStart, `//select/option`, "xy", 20, 36)
}

func TestQueryMismatchedHeadingGroupEndInsideSelect(t *testing.T) {
	const plain = `<select><h1>x</h2>y</select>`
	assertSelectQueryNode(t, plain, `//select/h1`, "x", 8, 13)
	assertSelectQueryNode(t, plain, `//select/h1/following-sibling::text()`, "y", 18, 19)

	const spanCase = `<select><h1><span>x</h2>y</select>`
	assertSelectQueryNode(t, spanCase, `//select/h1`, "x", 8, 19)
	assertSelectQueryNode(t, spanCase, `//select/h1/span`, "x", 12, 19)
	assertSelectQueryNode(t, spanCase, `//select/h1/following-sibling::text()`, "y", 24, 25)

	const optionCase = `<select><option><h1><span>x</h2>y</option></select>`
	assertSelectQueryNode(t, optionCase, `//select/option`, "xy", 8, 42)
	assertSelectQueryNode(t, optionCase, `//select/option/h1`, "x", 16, 27)
	assertSelectQueryNode(t, optionCase, `//select/option/h1/span`, "x", 20, 27)
	assertSelectQueryNode(t, optionCase, `//select/option/h1/following-sibling::text()`, "y", 32, 33)
}

func TestQueryLegacyImageAndBrEndBecomeVoidElementsInsideSelect(t *testing.T) {
	const selectImage = `<select>a<image src=x>b</select>`
	assertSelectQueryNode(t, selectImage, `//select/img`, "", 9, 22)
	assertSelectQueryNode(t, selectImage, `//select/img/following-sibling::text()`, "b", 22, 23)

	const optionImage = `<select><option>a<image src=x>b</option></select>`
	assertSelectQueryNode(t, optionImage, `//select/option/img`, "", 17, 30)
	assertSelectQueryNode(t, optionImage, `//select/option/img/following-sibling::text()`, "b", 30, 31)

	const brEnd = `<select><option>a</br>b</option></select>`
	assertSelectQueryNode(t, brEnd, `//select/option/br`, "", 17, 22)
	assertSelectQueryNode(t, brEnd, `//select/option/br/following-sibling::text()`, "b", 22, 23)
}

func TestQueryIgnoredSelectEndTokenRangeDoesNotLeakAcrossElements(t *testing.T) {
	const divCase = `<select><div></foo></div>x</select>`
	assertSelectQueryNode(t, divCase, `//select/div`, "", 8, 25)
	assertSelectQueryNode(t, divCase, `//select/div/following-sibling::text()`, "x", 25, 26)

	const spanCase = `<select><option><span></foo></span>x</option></select>`
	assertSelectQueryNode(t, spanCase, `//select/option/span`, "", 16, 35)
	assertSelectQueryNode(t, spanCase, `//select/option/span/following-sibling::text()`, "x", 35, 36)

	const innerElement = `<select><div></foo><span>a</span></div>x</select>`
	assertSelectQueryNode(t, innerElement, `//select/div/span/text()`, "a", 25, 26)
	assertSelectQueryNode(t, innerElement, `//select/div/following-sibling::text()`, "x", 39, 40)
}

func TestQueryIgnoredBodyHTMLEndsInsideSelectDoNotLeakEOFRecovery(t *testing.T) {
	const explicit = `<html><body><select>a</body>b</html>c</select></body></html><foo>x`
	assertSelectQueryNode(t, explicit, `//foo`, "x", 60, len(explicit))

	const legacy = `<select>a</body>b</html>c</select><foo>x`
	assertSelectQueryNode(t, legacy, `//foo`, "x", 34, len(legacy))

	const unresolved = `<select>a</body>b`
	assertSelectQueryNode(t, unresolved, `//select`, "ab", 0, len(unresolved))
	assertSelectQueryNode(t, unresolved, `//select/text()`, "ab", 8, len(unresolved))
}
