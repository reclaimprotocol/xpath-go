package xpath_test

import (
	"reflect"
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryMalformedStartTagRecoveryPreservesOriginalLocations(t *testing.T) {
	testCases := []struct {
		name       string
		document   string
		expression string
		want       string
	}{
		{
			name:       "whitespace after less-than remains text and parsing continues",
			document:   `<section>before< div id=bad>after<span id=target>tail</span></section>`,
			expression: `//span[@id='target']/text()`,
			want:       "tail",
		},
		{
			name:       "unexpected solidus before attribute",
			document:   `<div/ id=target>payload</div>`,
			expression: `//div[@id='target']/text()`,
			want:       "payload",
		},
		{
			name:       "unexpected solidus starts attribute recovery",
			document:   `<div /foo=bar>payload</div>`,
			expression: `//div[@foo='bar']/text()`,
			want:       "payload",
		},
		{
			name:       "adjacent quoted attributes",
			document:   `<div id="target"class='primary'data-x=ok>payload</div>`,
			expression: `//div[@id='target'][@class='primary'][@data-x='ok']/text()`,
			want:       "payload",
		},
		{
			name:       "equals before attribute name",
			document:   `<div =foo bar=baz>payload</div>`,
			expression: `//div[@bar='baz']/text()`,
			want:       "payload",
		},
		{
			name:       "less than in attribute name",
			document:   `<span <="">broken</span>`,
			expression: `//span/text()`,
			want:       "broken",
		},
		{
			name:       "invalid characters in unquoted value",
			document:   "<div data-x=a\"b'c<d=e`f>payload</div>",
			expression: `//div[@data-x]/text()`,
			want:       "payload",
		},
		{
			name:       "missing value becomes empty",
			document:   `<div id=>payload</div>`,
			expression: `//div[@id='']/text()`,
			want:       "payload",
		},
		{
			name:       "duplicate attributes keep first occurrence",
			document:   `<div id=first ID=second class=a CLASS=b>payload</div>`,
			expression: `//div[@id='first'][@class='a']/text()`,
			want:       "payload",
		},
		{
			name:       "CRLF and CR normalize in quoted value",
			document:   "<div data-x='a\r\nb\rc'>payload</div>",
			expression: `//div[@data-x]/text()`,
			want:       "payload",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(testCase.expression, testCase.document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("Expected one result, got %d", len(results))
			}
			result := results[0]
			if result.TextContent != testCase.want {
				t.Fatalf("Expected %q, got %q", testCase.want, result.TextContent)
			}
			wantStart := strings.Index(testCase.document, testCase.want)
			if wantStart < 0 {
				t.Fatalf("Test setup: expected source to contain %q", testCase.want)
			}
			if result.StartLocation != wantStart || result.EndLocation != wantStart+len(testCase.want) {
				t.Fatalf("Expected original range %d:%d, got %d:%d", wantStart, wantStart+len(testCase.want), result.StartLocation, result.EndLocation)
			}
			if source := testCase.document[result.StartLocation:result.EndLocation]; source != testCase.want {
				t.Fatalf("Expected exact original source %q, got %q", testCase.want, source)
			}
		})
	}
}

func TestQueryIgnoredStartTagAtEOFUsesBrowserLocation(t *testing.T) {
	for _, document := range []string{"before<div", "before<div id=", "before<div id='target"} {
		t.Run(document, func(t *testing.T) {
			results, err := xpath.Query(`//text()[. = 'before']`, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("Expected one text result, got %d", len(results))
			}
			result := results[0]
			if result.TextContent != "before" || result.StartLocation != 0 || result.EndLocation != len(document) {
				t.Fatalf("Expected browser text/value location for ignored tag, got %#v", result)
			}
		})
	}
}

func TestQueryNonVoidSelfClosingFlagPreservesBrowserStructureAndLocations(t *testing.T) {
	testCases := []struct {
		name       string
		document   string
		expression string
	}{
		{name: "flag after name", document: `<div/>x</div><span>tail</span>`, expression: `//div/text()`},
		{name: "flag after attribute", document: `<div a=b />x</div><span>tail</span>`, expression: `//div[@a='b']/text()`},
		{name: "slash belongs to unquoted value", document: `<div a=b/>x</div><span>tail</span>`, expression: `//div[@a='b/']/text()`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(testCase.expression, testCase.document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].TextContent != "x" {
				t.Fatalf("Expected one div text result x, got %#v", results)
			}
			wantStart := strings.Index(testCase.document, "x")
			result := results[0]
			if result.StartLocation != wantStart || result.EndLocation != wantStart+1 {
				t.Fatalf("Expected exact original x range %d:%d, got %d:%d", wantStart, wantStart+1, result.StartLocation, result.EndLocation)
			}
		})
	}
}

func TestQueryAttributeDuplicateFoldingIsASCIIOnly(t *testing.T) {
	const document = `<div ID=first id=second É=one é=two>x</div>`

	results, err := xpath.Query(`//div[@id='first']`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected one result, got %d", len(results))
	}
	result := results[0]
	wantAttributes := map[string]string{"id": "first", "É": "one", "é": "two"}
	if !reflect.DeepEqual(result.Attributes, wantAttributes) {
		t.Fatalf("Expected ASCII-only duplicate folding %#v, got %#v", wantAttributes, result.Attributes)
	}
	if result.StartLocation != 0 || result.EndLocation != len(document) {
		t.Fatalf("Expected exact original byte range 0:%d, got %d:%d", len(document), result.StartLocation, result.EndLocation)
	}
	if source := document[result.StartLocation:result.EndLocation]; source != document {
		t.Fatalf("Expected exact original source, got %q", source)
	}
}

func TestQueryPunctuationAndUnicodeTagNamesPreserveOriginalByteLocations(t *testing.T) {
	testCases := []struct {
		name     string
		document string
		wantName string
		wantNode string
	}{
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
			if len(results) != 1 {
				t.Fatalf("Expected one <%s> result, got %#v", testCase.wantName, results)
			}
			result := results[0]
			if result.NodeName != testCase.wantName || result.TextContent != "x" {
				t.Fatalf("Expected <%s> with text x, got %#v", testCase.wantName, result)
			}
			if result.StartLocation != 0 || result.EndLocation != len(testCase.wantNode) {
				t.Fatalf("Expected byte range 0:%d, got %d:%d", len(testCase.wantNode), result.StartLocation, result.EndLocation)
			}
			if source := testCase.document[result.StartLocation:result.EndLocation]; source != testCase.wantNode {
				t.Fatalf("Expected exact original source %q, got %q", testCase.wantNode, source)
			}
		})
	}
}
