package xpath_test

import (
	"bytes"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

// charsetQueryOptions describes the contract for response-body decoding:
// input remains an arbitrary byte-preserving Go string, while Charset controls
// the logical Unicode view used by the HTML parser and XPath evaluator.
func charsetQueryOptions(charset string) xpath.Options {
	return xpath.Options{
		IncludeLocation: true,
		OutputFormat:    "nodes",
		Charset:         charset,
	}
}

func TestQueryISO88591DecodesXPathTextAndPreservesRawByteLocations(t *testing.T) {
	// The prefix and target contain Latin-1 bytes. Their decoded Unicode view
	// must be searchable, but all public locations must still index raw bytes.
	prefix := []byte("<p>caf\xe9</p>")
	target := []byte("<div title=\"Se\xf1or\">Jos\xe9</div>")
	document := append(append([]byte{}, prefix...), target...)

	results, err := xpath.QueryWithOptions(
		`//div[contains(text(), 'José') and @title='Señor']`,
		string(document),
		charsetQueryOptions("iso-8859-1"),
	)
	if err != nil {
		t.Fatalf("ISO-8859-1 XPath query returned an error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected one Latin-1 div, got %#v", results)
	}

	result := results[0]
	wantStart := len(prefix)
	wantEnd := len(document)
	wantContentStart := wantStart + bytes.Index(target, []byte(">")) + 1
	wantContentEnd := wantEnd - len("</div>")
	if result.TextContent != "José" {
		t.Fatalf("Expected decoded text %q, got %q", "José", result.TextContent)
	}
	if result.Attributes["title"] != "Señor" {
		t.Fatalf("Expected decoded title %q, got %q", "Señor", result.Attributes["title"])
	}
	if result.StartLocation != wantStart || result.EndLocation != wantEnd ||
		result.ContentStart != wantContentStart || result.ContentEnd != wantContentEnd {
		t.Fatalf("Expected raw locations %d:%d content %d:%d, got %d:%d content %d:%d",
			wantStart, wantEnd, wantContentStart, wantContentEnd,
			result.StartLocation, result.EndLocation, result.ContentStart, result.ContentEnd)
	}
	if result.Value != string(target) {
		t.Fatalf("Expected Value to preserve raw source bytes %q, got %q", string(target), result.Value)
	}
	if got := string(document[result.StartLocation:result.EndLocation]); got != string(target) {
		t.Fatalf("Result range does not slice the target raw bytes: got %q", got)
	}
}

func TestQueryISO88591MatchesRequestedNonASCIIXPathForms(t *testing.T) {
	document := []byte("<table><tr><td>Se\xf1or</td></tr></table><div title=\"Informaci\xf3n\"></div><span>  Jos\xe9  </span>")
	testCases := []struct {
		name       string
		expression string
		wantName   string
	}{
		{name: "contains text", expression: `//td[contains(text(), 'Señor')]`, wantName: "td"},
		{name: "attribute equality", expression: `//div[@title='Información']`, wantName: "div"},
		{name: "normalized text", expression: `//span[normalize-space(text())='José']`, wantName: "span"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.QueryWithOptions(testCase.expression, string(document), charsetQueryOptions("iso-8859-1"))
			if err != nil {
				t.Fatalf("charset-aware query failed: %v", err)
			}
			if len(results) != 1 || results[0].NodeName != testCase.wantName {
				t.Fatalf("%s returned %#v", testCase.expression, results)
			}
		})
	}
}

func TestQueryWindows1252DecodesNonASCIIAttribute(t *testing.T) {
	// 0x93 and 0x94 are smart quotes in Windows-1252 (and control characters
	// in ISO-8859-1), making this case distinguish the two charset mappings.
	document := append([]byte(`<div title="`), 0x93)
	document = append(document, []byte(`Hola`)...)
	document = append(document, 0x94)
	document = append(document, []byte(`">ok</div>`)...)

	results, err := xpath.QueryWithOptions(
		`//div[@title='“Hola”']/text()`,
		string(document),
		charsetQueryOptions("windows-1252"),
	)
	if err != nil {
		t.Fatalf("Windows-1252 XPath query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "ok" {
		t.Fatalf("Expected the Windows-1252 title predicate to select ok, got %#v", results)
	}
	wantStart := bytes.Index(document, []byte("ok"))
	if results[0].StartLocation != wantStart || results[0].EndLocation != wantStart+len("ok") {
		t.Fatalf("Expected text raw range %d:%d, got %d:%d", wantStart, wantStart+len("ok"), results[0].StartLocation, results[0].EndLocation)
	}
}

func TestQueryASCIIStructuralXPathToleratesUnrelatedInvalidUTF8(t *testing.T) {
	// XPath only examines structure and ASCII text here. An invalid UTF-8 byte
	// in an earlier, unrelated text node must not prevent the later match.
	document := []byte("<div>before\xff</div><span>target</span>")
	results, err := xpath.Query(`//span/text()`, string(document))
	if err != nil {
		t.Fatalf("ASCII structural XPath must tolerate unrelated invalid UTF-8: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "target" {
		t.Fatalf("Expected the later ASCII span text, got %#v", results)
	}
	wantStart := bytes.Index(document, []byte("target"))
	if results[0].StartLocation != wantStart || results[0].EndLocation != wantStart+len("target") {
		t.Fatalf("Expected target raw range %d:%d, got %d:%d", wantStart, wantStart+len("target"), results[0].StartLocation, results[0].EndLocation)
	}
}
