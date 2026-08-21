package xpath_test

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryRejectsGzipInput(t *testing.T) {
	const document = `<html><body><table><tr><td>Student Name</td><td><div>Jane Doe</div></td></tr></table></body></html>`
	const expression = `//td[normalize-space()='Student Name']/following-sibling::td[1]/div/text()`

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte(document)); err != nil {
		t.Fatalf("Could not create gzip input: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Could not finish gzip input: %v", err)
	}

	results, err := xpath.Query(expression, compressed.String())
	if err == nil {
		t.Fatalf("Expected binary input error, got %d results", len(results))
	}
	if !strings.Contains(err.Error(), "binary input") {
		t.Fatalf("Expected descriptive binary input error, got %q", err)
	}
}

func TestQueryRecoversUnterminatedElementInImplicitDocument(t *testing.T) {
	results, err := xpath.Query(`/html/body/div`, `<div>`)
	if err != nil || len(results) != 1 || results[0].StartLocation != 0 || results[0].EndLocation != 5 {
		t.Fatalf("Implicit document element EOF recovery mismatch: %#v err=%v", results, err)
	}
}

func TestQueryIgnoresRootOrphanClosingTagLikeBrowser(t *testing.T) {
	results, err := xpath.Query("//*", `</orphan>`)
	if err != nil {
		t.Fatalf("Root orphan end tag must be ignored: %v", err)
	}
	if len(results) != 3 || results[0].NodeName != "html" || results[1].NodeName != "head" || results[2].NodeName != "body" {
		t.Fatalf("Root orphan end tag should emit only the implicit document skeleton: %#v", results)
	}
}

func TestQueryRecoversBogusDeclarationAtReportedOffset(t *testing.T) {
	const declaration = `<!ENTITY example "value">`
	document := `<div>` + strings.Repeat("a", 14733) + declaration + `<span id="target">payload</span></div>`

	results, err := xpath.Query(`//span[@id='target']/text()`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected one result, got %d", len(results))
	}

	result := results[0]
	if result.TextContent != "payload" {
		t.Fatalf("Expected payload, got %q", result.TextContent)
	}
	if source := document[result.StartLocation:result.EndLocation]; source != "payload" {
		t.Fatalf("Expected original source %q, got %q", "payload", source)
	}
	if declarationOffset := strings.Index(document, declaration); declarationOffset != 14738 {
		t.Fatalf("Expected declaration at reported offset 14738, got %d", declarationOffset)
	}
}

func TestQueryContinuesAfterRecoveredCommentsWithOriginalLocations(t *testing.T) {
	tokens := []string{
		`<!-->`,
		`<!--->`,
		`<!--foo--!>`,
		`<!--foo<!--bar-->`,
		`<?xml version="1.0"?>`,
	}

	for _, token := range tokens {
		t.Run(token, func(t *testing.T) {
			document := `<div>before` + token + `after<span id="target">payload</span></div>`
			results, err := xpath.Query(`//span[@id='target']/text()`, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("Expected one result, got %d", len(results))
			}

			result := results[0]
			if result.TextContent != "payload" {
				t.Fatalf("Expected payload, got %q", result.TextContent)
			}
			if source := document[result.StartLocation:result.EndLocation]; source != "payload" {
				t.Fatalf("Expected original source %q, got %q", "payload", source)
			}
			wantStart := strings.Index(document, "payload")
			if result.StartLocation != wantStart || result.EndLocation != wantStart+len("payload") {
				t.Fatalf("Expected original range %d:%d, got %d:%d", wantStart, wantStart+len("payload"), result.StartLocation, result.EndLocation)
			}
		})
	}
}

func TestQueryAdjacentHTMLAttributesPreservesOriginalLocations(t *testing.T) {
	const document = `<section><div id="target"class='primary'data-value="42">payload</div></section>`
	const expression = `//div[@id='target'][@class='primary'][@data-value='42']/text()`

	results, err := xpath.Query(expression, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected one result, got %d", len(results))
	}

	result := results[0]
	if result.TextContent != "payload" {
		t.Fatalf("Expected payload, got %q", result.TextContent)
	}
	if source := document[result.StartLocation:result.EndLocation]; source != "payload" {
		t.Fatalf("Expected original source %q, got %q", "payload", source)
	}
}

func TestQueryIgnoresMetaClosingTagAndPreservesOriginalLocations(t *testing.T) {
	const document = `<html><head><meta name="description"content="sample"></meta><title>Page</title></head><body><div id="target">payload</div></body></html>`
	const expression = `//div[@id='target']/text()`

	results, err := xpath.Query(expression, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected one result, got %d", len(results))
	}

	result := results[0]
	if result.TextContent != "payload" {
		t.Fatalf("Expected payload, got %q", result.TextContent)
	}
	if source := document[result.StartLocation:result.EndLocation]; source != "payload" {
		t.Fatalf("Expected original source %q, got %q", "payload", source)
	}
}

func TestQueryRecoversBrClosingTagWithOriginalLocation(t *testing.T) {
	const document = `<div>before</br><span>after</span></div>`

	results, err := xpath.Query(`//br`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected one result, got %d", len(results))
	}

	result := results[0]
	if result.NodeName != "br" || result.Value != `</br>` {
		t.Fatalf("Expected recovered br backed by original source, got %#v", result)
	}
	if source := document[result.StartLocation:result.EndLocation]; source != `</br>` {
		t.Fatalf("Expected original source %q, got %q", `</br>`, source)
	}
}

func TestQueryRecoversConsecutiveTableRowsLikeJavaScript(t *testing.T) {
	const document = `<table><tr><th>Full Name</th><td>AKASH</td></tr><tr><tr><th>Date of Birth</th><td>2003-07-12</td></tr></table>`

	testCases := []struct {
		name       string
		expression string
		want       string
		wantSource string
	}{
		{
			name:       "user XPath selects first matching cell sibling",
			expression: `//th[normalize-space(text())='Full Name']/following-sibling::td[1]/text()`,
			want:       "AKASH",
			wantSource: "AKASH",
		},
		{
			name:       "implicit empty row keeps JavaScript position",
			expression: `//table//tr[2]`,
			want:       "",
			wantSource: `<tr>`,
		},
		{
			name:       "row after implicit close has third position",
			expression: `//table//tr[3]/th/text()`,
			want:       "Date of Birth",
			wantSource: "Date of Birth",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(testCase.expression, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("Expected one result, got %d", len(results))
			}
			if results[0].TextContent != testCase.want {
				t.Fatalf("Expected text %q, got %q", testCase.want, results[0].TextContent)
			}
			start, end := results[0].StartLocation, results[0].EndLocation
			if start < 0 || end < start || end > len(document) {
				t.Fatalf("Invalid original-source range %d:%d", start, end)
			}
			if source := document[start:end]; source != testCase.wantSource {
				t.Fatalf("Expected original source %q, got %q", testCase.wantSource, source)
			}
		})
	}
}

func TestSiblingAxisPositionsMatchJavaScript(t *testing.T) {
	const document = `<table><tr><td>first</td><th>middle</th><td>nearest</td><th>anchor</th><td>next</td><th>skip</th><td>last</td></tr></table>`

	testCases := []struct {
		name       string
		expression string
		want       string
	}{
		{
			name:       "following sibling position uses first matching node",
			expression: `//th[normalize-space(text())='anchor']/following-sibling::td[1]/text()`,
			want:       "next",
		},
		{
			name:       "preceding sibling position uses reverse axis order",
			expression: `//th[normalize-space(text())='anchor']/preceding-sibling::td[1]/text()`,
			want:       "nearest",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(testCase.expression, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("Expected one result, got %d", len(results))
			}
			if results[0].TextContent != testCase.want {
				t.Fatalf("Expected text %q, got %q", testCase.want, results[0].TextContent)
			}
		})
	}
}
