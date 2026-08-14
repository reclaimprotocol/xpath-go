package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryNumericCharacterReferencesInDataUseDecodedValuesAndRawByteRanges(t *testing.T) {
	testCases := []struct {
		name      string
		raw       string
		wantValue string
	}{
		{name: "decimal", raw: `a&#65;b`, wantValue: "aAb"},
		{name: "hexadecimal", raw: `a&#x41;b`, wantValue: "aAb"},
		{name: "uppercase X and supplementary code point", raw: `a&#X1F642;b`, wantValue: "a🙂b"},
		{name: "missing decimal semicolon", raw: `a&#65b`, wantValue: "aAb"},
		{name: "missing hexadecimal semicolon", raw: `a&#x41Z`, wantValue: "aAZ"},
		{name: "no decimal digits", raw: `a&#;b`, wantValue: `a&#;b`},
		{name: "no hexadecimal digits", raw: `a&#xZb`, wantValue: `a&#xZb`},
		{name: "NUL becomes replacement", raw: `a&#0;b`, wantValue: "a\uFFFDb"},
		{name: "outside Unicode range becomes replacement", raw: `a&#x110000;b`, wantValue: "a\uFFFDb"},
		{name: "surrogate becomes replacement", raw: `a&#xD800;b`, wantValue: "a\uFFFDb"},
		{name: "Windows-1252 C1 remapping", raw: `a&#x80;b`, wantValue: "a€b"},
		{name: "noncharacter is emitted", raw: `a&#xFDD0;b`, wantValue: "a\uFDD0b"},
		{name: "control is emitted", raw: `a&#13;b`, wantValue: "a\rb"},
		{name: "adjacent references coalesce", raw: `before&#65;&#x42;after`, wantValue: "beforeABafter"},
		{name: "multibyte prefix", raw: `é&#x1F642;z`, wantValue: "é🙂z"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document := `<div>` + testCase.raw + `</div>`
			results, err := xpath.Query(`//div/text()`, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].NodeName != "#text" || results[0].TextContent != testCase.wantValue {
				t.Fatalf("Expected one decoded text value %q, got %#v", testCase.wantValue, results)
			}
			result := results[0]
			wantStart := len(`<div>`)
			wantEnd := wantStart + len(testCase.raw)
			if result.StartLocation != wantStart || result.EndLocation != wantEnd {
				t.Fatalf("Expected original UTF-8 byte range %d:%d, got %d:%d", wantStart, wantEnd, result.StartLocation, result.EndLocation)
			}
			if source := document[result.StartLocation:result.EndLocation]; source != testCase.raw {
				t.Fatalf("Expected location to retain raw source %q, got %q", testCase.raw, source)
			}
		})
	}
}

func TestQueryNumericCharacterReferencesInAttributesUseDecodedValues(t *testing.T) {
	testCases := []struct {
		name      string
		document  string
		wantValue string
	}{
		{name: "double-quoted decimal", document: `<div data-x="a&#65;b">t</div>`, wantValue: "aAb"},
		{name: "single-quoted uppercase X", document: `<div data-x='a&#X41;b'>t</div>`, wantValue: "aAb"},
		{name: "unquoted missing semicolon", document: `<div data-x=a&#x41z>t</div>`, wantValue: "aAz"},
		{name: "no hexadecimal digits", document: `<div data-x="a&#xZb">t</div>`, wantValue: `a&#xZb`},
		{name: "NUL becomes replacement", document: `<div data-x="a&#0;b">t</div>`, wantValue: "a\uFFFDb"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(`//div`, testCase.document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("Expected one div result, got %#v", results)
			}
			if got := results[0].Attributes["data-x"]; got != testCase.wantValue {
				t.Fatalf("Expected decoded attribute value %q, got %q", testCase.wantValue, got)
			}
			if results[0].StartLocation != 0 || results[0].EndLocation != len(testCase.document) {
				t.Fatalf("Expected full original element range 0:%d, got %#v", len(testCase.document), results[0])
			}
		})
	}
}

func TestQueryLeadingDecodedWhitespaceStillAcceptsInitialDoctype(t *testing.T) {
	const document = `&#32;<!DOCTYPE html><p>x</p>`

	results, err := xpath.Query(`/node()[1]`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].NodeType != 10 || results[0].NodeName != "html" {
		t.Fatalf("Expected initial html doctype after discarded decoded whitespace, got %#v", results)
	}
	if results[0].StartLocation != len(`&#32;`) || results[0].EndLocation != len(`&#32;<!DOCTYPE html>`) {
		t.Fatalf("Expected exact accepted doctype range, got %#v", results[0])
	}
}

func TestQueryDecodedWhitespaceAfterBodyContentIsPreserved(t *testing.T) {
	const document = `<!DOCTYPE html><p>x</p>&#32;`

	results, err := xpath.Query(`//text()[.=' ']`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != " " {
		t.Fatalf("Expected one decoded trailing space, got %#v", results)
	}
	wantStart := len(`<!DOCTYPE html><p>x</p>`)
	if results[0].StartLocation != wantStart || results[0].EndLocation != len(document) {
		t.Fatalf("Expected raw trailing reference range %d:%d, got %#v", wantStart, len(document), results[0])
	}
}

func TestQueryMixedLeadingDecodedWhitespaceStillAcceptsInitialDoctype(t *testing.T) {
	document := `&#32; ` + "\t" + `&#x20;<!DOCTYPE html><p>x</p>`
	prefix := `&#32; ` + "\t" + `&#x20;`

	results, err := xpath.Query(`/node()[1]`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].NodeType != 10 || results[0].NodeName != "html" {
		t.Fatalf("Expected initial html doctype after mixed whitespace, got %#v", results)
	}
	if results[0].StartLocation != len(prefix) || results[0].EndLocation != len(prefix)+len(`<!DOCTYPE html>`) {
		t.Fatalf("Expected exact doctype range after whitespace prefix, got %#v", results[0])
	}
}

func TestQueryLeadingDecodedNonWhitespaceIgnoresFollowingDoctype(t *testing.T) {
	const document = `&#65;<!DOCTYPE html><p>x</p>`

	results, err := xpath.Query(`//text()[.='A']`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "A" {
		t.Fatalf("Expected decoded leading A, got %#v", results)
	}
	if results[0].StartLocation != 0 || results[0].EndLocation != len(`&#65;`) {
		t.Fatalf("Expected raw A reference range 0:%d, got %#v", len(`&#65;`), results[0])
	}
}

func TestQueryMultibyteTextAndDecodedSpaceUseUTF8ByteRange(t *testing.T) {
	const document = `é&#32;<!DOCTYPE html><p>x</p>`

	results, err := xpath.Query(`//text()[.='é ']`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "é " {
		t.Fatalf("Expected decoded multibyte text and space, got %#v", results)
	}
	if results[0].StartLocation != 0 || results[0].EndLocation != len(`é&#32;`) {
		t.Fatalf("Expected UTF-8 byte range 0:%d, got %#v", len(`é&#32;`), results[0])
	}
	if source := document[results[0].StartLocation:results[0].EndLocation]; source != `é&#32;` {
		t.Fatalf("Expected exact original multibyte/reference source, got %q", source)
	}
}
