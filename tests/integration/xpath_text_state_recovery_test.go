package xpath_test

import (
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func assertTextStateQuery(t *testing.T, document, expression, wantValue string, wantStart, wantEnd int) {
	t.Helper()
	results, err := xpath.Query(expression, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].NodeName != "#text" || results[0].TextContent != wantValue {
		t.Fatalf("Expected one text value %q, got %#v", wantValue, results)
	}
	if results[0].StartLocation != wantStart || results[0].EndLocation != wantEnd {
		t.Fatalf("Expected exact raw range %d:%d, got %#v", wantStart, wantEnd, results[0])
	}
}

func TestQueryRCDATAAndRawTextCharacterReferenceStates(t *testing.T) {
	testCases := []struct {
		tag       string
		raw       string
		wantValue string
	}{
		{tag: "title", raw: `a&amp;&#65;&lt;b&gt;`, wantValue: `a&A<b>`},
		{tag: "textarea", raw: `a&amp;&#65;&lt;b&gt;`, wantValue: `a&A<b>`},
		{tag: "style", raw: `a&amp;&#65;&lt;`, wantValue: `a&amp;&#65;&lt;`},
		{tag: "script", raw: `a&amp;&#65;&lt;`, wantValue: `a&amp;&#65;&lt;`},
		{tag: "title", raw: `é&amp;&#x1F642;z`, wantValue: "é&🙂z"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.tag+testCase.raw, func(t *testing.T) {
			opening := `<` + testCase.tag + `>`
			document := opening + testCase.raw + `</` + testCase.tag + `>`
			assertTextStateQuery(t, document, `//`+testCase.tag+`/text()`, testCase.wantValue, len(opening), len(opening)+len(testCase.raw))
		})
	}
}

func TestQueryTextStateEndTagCandidateRecovery(t *testing.T) {
	testCases := []struct {
		name      string
		document  string
		tag       string
		wantValue string
	}{
		{name: "case fold", document: `<title>before</TITLE><p>after</p>`, tag: "title", wantValue: "before"},
		{name: "nonmatch", document: `<title>a</title-x>b</title>`, tag: "title", wantValue: `a</title-x>b`},
		{name: "malformed", document: `<title>a</title!>b</title>`, tag: "title", wantValue: `a</title!>b`},
		{name: "whitespace close", document: `<title>x</title   ><p>y</p>`, tag: "title", wantValue: "x"},
		{name: "attributes close", document: `<textarea>x</textarea foo=bar><p>y</p>`, tag: "textarea", wantValue: "x"},
		{name: "slash close", document: `<style>x</style/><p>y</p>`, tag: "style", wantValue: "x"},
		{name: "less opener", document: `<title>a<b>c</title>`, tag: "title", wantValue: `a<b>c`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			start := strings.Index(testCase.document, ">") + 1
			lowerDocument := strings.ToLower(testCase.document)
			end := strings.LastIndex(lowerDocument, `</`+testCase.tag)
			if strings.Contains(testCase.document, `<p>`) {
				end = strings.Index(lowerDocument, `</`+testCase.tag)
			}
			assertTextStateQuery(t, testCase.document, `//`+testCase.tag+`/text()`, testCase.wantValue, start, end)
		})
	}
}

func TestQueryTextStateNULReplacement(t *testing.T) {
	for _, tag := range []string{"title", "textarea", "style", "script"} {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			document := opening + "a\x00b</" + tag + `>`
			assertTextStateQuery(t, document, `//`+tag+`/text()`, "a\uFFFDb", len(opening), len(opening)+3)
		})
	}
}

func TestQueryTextStatesCloseAtEOF(t *testing.T) {
	testCases := []struct{ tag, raw, wantValue string }{
		{tag: "title", raw: `x&amp;`, wantValue: `x&`},
		{tag: "textarea", raw: `x&#65;`, wantValue: `xA`},
		{tag: "style", raw: `x&amp;`, wantValue: `x&amp;`},
		{tag: "script", raw: `x&amp;`, wantValue: `x&amp;`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.tag, func(t *testing.T) {
			opening := `<` + testCase.tag + `>`
			document := opening + testCase.raw
			assertTextStateQuery(t, document, `//`+testCase.tag+`/text()`, testCase.wantValue, len(opening), len(document))
		})
	}
}

func TestQueryTextareaInitialNewlineStrippingUsesBrowserLocation(t *testing.T) {
	const document = "<textarea>\nabc</textarea>"
	assertTextStateQuery(t, document, `//textarea/text()`, "abc", len("<textarea>\n"), len("<textarea>\nabc"))
}

func TestQueryTextareaInitialNewlineRecoveryUsesBrowserLocations(t *testing.T) {
	testCases := []struct {
		name, raw, wantValue, wantSource string
		wantStart, wantEnd               int
	}{
		{name: "CRLF", raw: "\r\nabc", wantValue: "abc", wantSource: "abc", wantStart: 12, wantEnd: 15},
		{name: "lone CR", raw: "\rabc", wantValue: "abc", wantSource: "abc", wantStart: 11, wantEnd: 14},
		{name: "double LF", raw: "\n\nabc", wantValue: "\nabc", wantSource: "\n\nabc", wantStart: 10, wantEnd: 15},
		{name: "decoded numeric LF", raw: "&#10;abc", wantValue: "abc", wantSource: "abc", wantStart: 15, wantEnd: 18},
		{name: "decoded named LF", raw: "&NewLine;abc", wantValue: "abc", wantSource: "abc", wantStart: 19, wantEnd: 22},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document := "<textarea>" + testCase.raw + "</textarea>"
			assertTextStateQuery(t, document, `//textarea/text()`, testCase.wantValue, testCase.wantStart, testCase.wantEnd)
			if source := document[testCase.wantStart:testCase.wantEnd]; source != testCase.wantSource {
				t.Fatalf("Expected browser raw source %q, got %q", testCase.wantSource, source)
			}
		})
	}
}

func TestQueryTitleIncompleteEndTagCandidatesAtEOF(t *testing.T) {
	testCases := []struct {
		name, document, wantValue string
	}{
		{name: "name without delimiter remains RCDATA", document: "<title>x</title", wantValue: "x</title"},
		{name: "whitespace discards incomplete end tag", document: "<title>x</title ", wantValue: "x"},
		{name: "unterminated end tag attribute is discarded", document: `<title>x</title a="`, wantValue: "x"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			wantStart := len("<title>")
			assertTextStateQuery(t, testCase.document, `//title/text()`, testCase.wantValue, wantStart, len(testCase.document))
		})
	}
}
