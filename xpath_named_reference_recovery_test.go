package xpath_test

import (
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryNamedCharacterReferencesInDataUseDecodedValuesAndRawByteRanges(t *testing.T) {
	testCases := []struct {
		name      string
		raw       string
		wantValue string
	}{
		{name: "basic references", raw: `&amp;&lt;&gt;&quot;&apos;`, wantValue: `&<>"'`},
		{name: "non-basic copy", raw: `a&copy;b`, wantValue: "a©b"},
		{name: "longest match", raw: `a&notin;b`, wantValue: "a∉b"},
		{name: "semicolonless longest legacy prefix", raw: `a&notinX`, wantValue: "a¬inX"},
		{name: "legacy semicolonless copy", raw: `a&copy b`, wantValue: "a© b"},
		{name: "unknown reference remains literal", raw: `a&doesnotexist;b`, wantValue: `a&doesnotexist;b`},
		{name: "multi-codepoint reference", raw: `a&NotEqualTilde;b`, wantValue: "a≂̸b"},
		{name: "adjacent references coalesce", raw: `before&amp;&copy;&gt;after`, wantValue: "before&©>after"},
		{name: "decoded markup remains text", raw: `&lt;b&gt;x&lt;/b&gt;`, wantValue: `<b>x</b>`},
		{name: "multibyte prefix", raw: `é&copy;z`, wantValue: "é©z"},
		{name: "bounded entity corpus", raw: `&AElig;&frac12;&trade;&euro;&Alpha;&rarr;&hearts;`, wantValue: "Æ½™€Α→♥"},
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
				t.Fatalf("Expected raw UTF-8 byte range %d:%d, got %d:%d", wantStart, wantEnd, result.StartLocation, result.EndLocation)
			}
			if source := document[result.StartLocation:result.EndLocation]; source != testCase.raw {
				t.Fatalf("Expected exact raw reference source %q, got %q", testCase.raw, source)
			}
		})
	}
}

func TestQuerySemicolonlessNamedCharacterReferenceAtEOF(t *testing.T) {
	const document = `a&amp`

	results, err := xpath.Query(`//text()`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "a&" || results[0].StartLocation != 0 || results[0].EndLocation != len(document) {
		t.Fatalf("Expected decoded EOF ampersand over full raw range, got %#v", results)
	}
}

func TestQueryNamedCharacterReferencesInAttributesUseBrowserAmbiguityRules(t *testing.T) {
	testCases := []struct {
		name      string
		document  string
		wantValue string
	}{
		{name: "quoted semicolon", document: `<div data-x="a&copy;b">x</div>`, wantValue: "a©b"},
		{name: "quoted ambiguous alphanumeric stays literal", document: `<div data-x="a&copycat">x</div>`, wantValue: `a&copycat`},
		{name: "quoted ambiguous equals stays literal", document: `<div data-x="a&copy=1">x</div>`, wantValue: `a&copy=1`},
		{name: "quoted legacy before space decodes", document: `<div data-x="a&copy z">x</div>`, wantValue: "a© z"},
		{name: "quoted semicolon before alphanumeric decodes", document: `<div data-x="a&copy;cat">x</div>`, wantValue: "a©cat"},
		{name: "unquoted semicolon", document: `<div data-x=a&amp;b>x</div>`, wantValue: `a&b`},
		{name: "unquoted ambiguous alphanumeric stays literal", document: `<div data-x=a&ampb>x</div>`, wantValue: `a&ampb`},
		{name: "unquoted ambiguous equals stays literal", document: `<div data-x=a&amp=1>x</div>`, wantValue: `a&amp=1`},
		{name: "unknown remains literal", document: `<div data-x="a&wat;b">x</div>`, wantValue: `a&wat;b`},
		{name: "multi-codepoint reference", document: `<div data-x="a&NotEqualTilde;b">x</div>`, wantValue: "a≂̸b"},
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
				t.Fatalf("Expected attribute value %q, got %q", testCase.wantValue, got)
			}
			if results[0].StartLocation != 0 || results[0].EndLocation != len(testCase.document) {
				t.Fatalf("Expected exact original element range 0:%d, got %#v", len(testCase.document), results[0])
			}
		})
	}
}

func TestQueryUnknownLongerNamedReferencesUsesLegacyPrefixOnlyInData(t *testing.T) {
	testCases := []struct {
		raw       string
		wantValue string
	}{
		{raw: `&notinX;`, wantValue: "¬inX;"},
		{raw: `&copycat;`, wantValue: "©cat;"},
		{raw: `&ampx;`, wantValue: `&x;`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.raw, func(t *testing.T) {
			document := `<div>` + testCase.raw + `</div>`
			results, err := xpath.Query(`//div/text()`, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].TextContent != testCase.wantValue {
				t.Fatalf("Expected data value %q, got %#v", testCase.wantValue, results)
			}
			wantStart := len(`<div>`)
			wantEnd := wantStart + len(testCase.raw)
			if results[0].StartLocation != wantStart || results[0].EndLocation != wantEnd {
				t.Fatalf("Expected exact raw data range %d:%d, got %#v", wantStart, wantEnd, results[0])
			}
			if source := document[results[0].StartLocation:results[0].EndLocation]; source != testCase.raw {
				t.Fatalf("Expected exact raw source %q, got %q", testCase.raw, source)
			}
		})
	}
}

func TestQueryUnknownLongerNamedReferencesRemainLiteralInAttributes(t *testing.T) {
	testCases := []struct {
		raw       string
		wantValue string
	}{
		{raw: `&notinX;`, wantValue: `&notinX;`},
		{raw: `&copycat;`, wantValue: `&copycat;`},
		{raw: `&ampx;`, wantValue: `&ampx;`},
		{raw: `&notin;`, wantValue: "∉"},
		{raw: `&copy;`, wantValue: "©"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.raw, func(t *testing.T) {
			document := `<div data-x="` + testCase.raw + `">x</div>`
			results, err := xpath.Query(`//div`, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].Attributes["data-x"] != testCase.wantValue {
				t.Fatalf("Expected attribute value %q, got %#v", testCase.wantValue, results)
			}
			if results[0].StartLocation != 0 || results[0].EndLocation != len(document) {
				t.Fatalf("Expected exact original element range 0:%d, got %#v", len(document), results[0])
			}
		})
	}
}

func TestQueryLongUnknownNamedReferenceRunContinuesAtFollowingKnownReference(t *testing.T) {
	const runLength = 4096
	unknownRun := strings.Repeat("a", runLength)
	raw := `&` + unknownRun + `;&copy;z`
	document := `<div>` + raw + `</div>`
	wantValue := `&` + unknownRun + `;©z`

	results, err := xpath.Query(`//div/text()`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].NodeName != "#text" || results[0].TextContent != wantValue {
		t.Fatalf("Expected one text node with literal unknown run and decoded known suffix; results=%d", len(results))
	}
	wantStart := len(`<div>`)
	wantEnd := wantStart + len(raw)
	if results[0].StartLocation != wantStart || results[0].EndLocation != wantEnd {
		t.Fatalf("Expected raw source range %d:%d, got %#v", wantStart, wantEnd, results[0])
	}
	if source := document[results[0].StartLocation:results[0].EndLocation]; source != raw {
		t.Fatalf("Expected exact raw inner source of length %d, got length %d", len(raw), len(source))
	}
}
