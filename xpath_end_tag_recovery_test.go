package xpath_test

import (
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryMissingEndTagNamePreservesMergedBrowserTextLocation(t *testing.T) {
	const document = `<div>before</>after<span>tail</span></div>`

	results, err := xpath.Query(`//div/text()[1]`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "beforeafter" {
		t.Fatalf("Expected one merged text result, got %#v", results)
	}
	result := results[0]
	wantStart := strings.Index(document, "before")
	wantEnd := strings.Index(document, `<span>`)
	if result.StartLocation != wantStart || result.EndLocation != wantEnd {
		t.Fatalf("Expected merged text range %d:%d, got %d:%d", wantStart, wantEnd, result.StartLocation, result.EndLocation)
	}
	if source := document[result.StartLocation:result.EndLocation]; source != `before</>after` {
		t.Fatalf("Expected original range through ignored token, got %q", source)
	}
}

func TestQueryInvalidEndTagOpenReturnsBrowserBogusComment(t *testing.T) {
	testCases := []struct {
		token     string
		wantValue string
	}{
		{token: `</42>`, wantValue: "42"},
		{token: `</$foo>`, wantValue: "$foo"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.token, func(t *testing.T) {
			document := `<div>before` + testCase.token + `after<span>tail</span></div>`
			results, err := xpath.Query(`//div/node()[2]`, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("Expected one comment result, got %d", len(results))
			}
			result := results[0]
			if result.NodeName != "#comment" || result.TextContent != testCase.wantValue {
				t.Fatalf("Expected bogus comment %q, got %#v", testCase.wantValue, result)
			}
			wantStart := strings.Index(document, testCase.token)
			if result.StartLocation != wantStart || result.EndLocation != wantStart+len(testCase.token) {
				t.Fatalf("Expected exact comment range %d:%d, got %d:%d", wantStart, wantStart+len(testCase.token), result.StartLocation, result.EndLocation)
			}
			if source := document[result.StartLocation:result.EndLocation]; source != testCase.token {
				t.Fatalf("Expected exact original comment source, got %q", source)
			}
		})
	}
}

func TestQueryEndTagEOFRecoveryUsesBrowserLocations(t *testing.T) {
	testCases := []struct {
		name       string
		document   string
		expression string
		wantText   string
		wantStart  int
	}{
		{name: "EOF after opener", document: `before</`, expression: `//text()`, wantText: `before</`, wantStart: 0},
		{name: "EOF inside tag", document: `<div>before</div`, expression: `//div/text()`, wantText: "before", wantStart: len(`<div>`)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(testCase.expression, testCase.document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].TextContent != testCase.wantText {
				t.Fatalf("Expected text %q, got %#v", testCase.wantText, results)
			}
			result := results[0]
			if result.StartLocation != testCase.wantStart || result.EndLocation != len(testCase.document) {
				t.Fatalf("Expected browser range %d:%d, got %d:%d", testCase.wantStart, len(testCase.document), result.StartLocation, result.EndLocation)
			}
		})
	}
}

func TestQueryEndTagJunkStillClosesMatchingElementWithExactLocation(t *testing.T) {
	closingTags := []string{
		`</div foo=bar>`,
		`</div foo='bar' baz>`,
		`</div   >`,
		`</div/>`,
	}

	for _, closingTag := range closingTags {
		t.Run(closingTag, func(t *testing.T) {
			document := `<div>payload` + closingTag + `<span>tail</span>`
			results, err := xpath.Query(`//div`, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].TextContent != "payload" {
				t.Fatalf("Expected one closed div, got %#v", results)
			}
			result := results[0]
			wantEnd := len(`<div>payload`) + len(closingTag)
			if result.StartLocation != 0 || result.EndLocation != wantEnd {
				t.Fatalf("Expected exact div range 0:%d, got %d:%d", wantEnd, result.StartLocation, result.EndLocation)
			}
			if source := document[result.StartLocation:result.EndLocation]; source != `<div>payload`+closingTag {
				t.Fatalf("Expected exact original div source, got %q", source)
			}

			following, err := xpath.Query(`//span/text()`, document)
			if err != nil || len(following) != 1 || following[0].TextContent != "tail" {
				t.Fatalf("Expected parsing to continue with span: results=%#v err=%v", following, err)
			}
			wantTailStart := strings.Index(document, "tail")
			if following[0].StartLocation != wantTailStart || following[0].EndLocation != wantTailStart+len("tail") {
				t.Fatalf("Expected exact following text range, got %#v", following[0])
			}
		})
	}
}

func TestQueryEndTagNameTokenizationMatchesStartTags(t *testing.T) {
	testCases := []struct {
		name     string
		document string
		wantName string
		wantNode string
	}{
		{name: "ASCII case fold", document: `<DiV>x</dIv><span>tail</span>`, wantName: "div", wantNode: `<DiV>x</dIv>`},
		{name: "bang punctuation", document: `<a!>x</a!><span>tail</span>`, wantName: "a!", wantNode: `<a!>x</a!>`},
		{name: "at punctuation", document: `<a@>x</a@><span>tail</span>`, wantName: "a@", wantNode: `<a@>x</a@>`},
		{name: "Unicode", document: `<aé>x</aé><span>tail</span>`, wantName: "aé", wantNode: `<aé>x</aé>`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(`//*[text()='x']`, testCase.document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].NodeName != testCase.wantName {
				t.Fatalf("Expected one <%s> result, got %#v", testCase.wantName, results)
			}
			result := results[0]
			if result.StartLocation != 0 || result.EndLocation != len(testCase.wantNode) {
				t.Fatalf("Expected exact byte range 0:%d, got %d:%d", len(testCase.wantNode), result.StartLocation, result.EndLocation)
			}
			if source := testCase.document[result.StartLocation:result.EndLocation]; source != testCase.wantNode {
				t.Fatalf("Expected exact original source %q, got %q", testCase.wantNode, source)
			}
		})
	}
}

func TestQueryDocumentLevelIncompleteEndTagsUseBrowserTextRanges(t *testing.T) {
	testCases := []struct {
		name      string
		document  string
		wantValue string
	}{
		{name: "missing name", document: `before</>after`, wantValue: "beforeafter"},
		{name: "incomplete named tag", document: `before</div`, wantValue: "before"},
		{name: "incomplete br closer", document: `before</br`, wantValue: "before"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(`//text()`, testCase.document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].TextContent != testCase.wantValue {
				t.Fatalf("Expected one text value %q, got %#v", testCase.wantValue, results)
			}
			result := results[0]
			if result.StartLocation != 0 || result.EndLocation != len(testCase.document) {
				t.Fatalf("Expected full browser range 0:%d, got %d:%d", len(testCase.document), result.StartLocation, result.EndLocation)
			}
			if source := testCase.document[result.StartLocation:result.EndLocation]; source != testCase.document {
				t.Fatalf("Expected exact original range including ignored token, got %q", source)
			}
		})
	}
}

func TestQueryIncompleteEndTagAtEOFClosesNestedAncestorsWithBrowserRanges(t *testing.T) {
	const document = `<div><span>x</div`

	testCases := []struct {
		expression string
		wantName   string
		wantText   string
		wantStart  int
	}{
		{expression: `//div`, wantName: "div", wantText: "x", wantStart: 0},
		{expression: `//span`, wantName: "span", wantText: "x", wantStart: len(`<div>`)},
		{expression: `//span/text()`, wantName: "#text", wantText: "x", wantStart: len(`<div><span>`)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.expression, func(t *testing.T) {
			results, err := xpath.Query(testCase.expression, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].NodeName != testCase.wantName || results[0].TextContent != testCase.wantText {
				t.Fatalf("Expected one %s result with text %q, got %#v", testCase.wantName, testCase.wantText, results)
			}
			result := results[0]
			if result.StartLocation != testCase.wantStart || result.EndLocation != len(document) {
				t.Fatalf("Expected browser EOF range %d:%d, got %d:%d", testCase.wantStart, len(document), result.StartLocation, result.EndLocation)
			}
		})
	}
}

func TestQueryExactEndTagOpenerAtEOFBecomesTextInsideDiv(t *testing.T) {
	const document = `<div>before</`

	results, err := xpath.Query(`//div/text()`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != `before</` {
		t.Fatalf("Expected literal end opener in text, got %#v", results)
	}
	result := results[0]
	if result.StartLocation != len(`<div>`) || result.EndLocation != len(document) {
		t.Fatalf("Expected exact text range %d:%d, got %d:%d", len(`<div>`), len(document), result.StartLocation, result.EndLocation)
	}
	if source := document[result.StartLocation:result.EndLocation]; source != `before</` {
		t.Fatalf("Expected exact original source, got %q", source)
	}
}

func TestQueryFinalEndTagEOFEdgesUseBrowserValuesAndSourceRanges(t *testing.T) {
	testCases := []struct {
		name      string
		document  string
		wantValue string
	}{
		{
			name:      "missing name retains following whitespace",
			document:  `before</> after`,
			wantValue: "before after",
		},
		{
			name:      "unterminated quoted end-tag attribute is discarded",
			document:  `before</div foo='>'`,
			wantValue: "before",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(`//text()`, testCase.document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].NodeName != "#text" || results[0].TextContent != testCase.wantValue {
				t.Fatalf("Expected one text value %q, got %#v", testCase.wantValue, results)
			}

			result := results[0]
			if result.StartLocation != 0 || result.EndLocation != len(testCase.document) {
				t.Fatalf("Expected full original range 0:%d, got %d:%d", len(testCase.document), result.StartLocation, result.EndLocation)
			}
			if source := testCase.document[result.StartLocation:result.EndLocation]; source != testCase.document {
				t.Fatalf("Expected range to retain the discarded token, got %q", source)
			}
		})
	}
}

func TestQueryBogusEndTagAtEOFUsesExactOriginalRanges(t *testing.T) {
	testCases := []struct {
		name       string
		document   string
		expression string
		wantValue  string
		wantSource string
		wantStart  int
	}{
		{
			name:       "numeric bogus comment inside div",
			document:   `<div>before</42`,
			expression: `//div/node()[2]`,
			wantValue:  "42",
			wantSource: `</42`,
			wantStart:  len(`<div>before`),
		},
		{
			name:       "nested bogus comment",
			document:   `<div><span>x</$foo`,
			expression: `//span/node()[2]`,
			wantValue:  "$foo",
			wantSource: `</$foo`,
			wantStart:  len(`<div><span>x`),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(testCase.expression, testCase.document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].NodeName != "#comment" || results[0].TextContent != testCase.wantValue {
				t.Fatalf("Expected one bogus comment value %q, got %#v", testCase.wantValue, results)
			}

			result := results[0]
			if result.StartLocation != testCase.wantStart || result.EndLocation != len(testCase.document) {
				t.Fatalf("Expected exact EOF range %d:%d, got %d:%d", testCase.wantStart, len(testCase.document), result.StartLocation, result.EndLocation)
			}
			if source := testCase.document[result.StartLocation:result.EndLocation]; source != testCase.wantSource {
				t.Fatalf("Expected exact original comment source %q, got %q", testCase.wantSource, source)
			}
		})
	}
}

func TestQueryBogusEndTagAtEOFAfterMultibyteTextUsesByteRange(t *testing.T) {
	const document = `<div>é</42`

	results, err := xpath.Query(`//div/node()[2]`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].NodeName != "#comment" || results[0].TextContent != "42" {
		t.Fatalf("Expected one EOF bogus comment, got %#v", results)
	}

	result := results[0]
	wantStart := len(`<div>é`)
	if result.StartLocation != wantStart || result.EndLocation != len(document) {
		t.Fatalf("Expected original byte range %d:%d, got %d:%d", wantStart, len(document), result.StartLocation, result.EndLocation)
	}
	if source := document[result.StartLocation:result.EndLocation]; source != `</42` {
		t.Fatalf("Expected exact original comment source, got %q", source)
	}
}
