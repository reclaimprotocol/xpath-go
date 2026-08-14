package utils

import (
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseNamedCharacterReferencesInDataLikeBrowser(t *testing.T) {
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
			content := `<div>` + testCase.raw + `</div>`
			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
				t.Fatalf("Expected one div root, got %#v", parsedBodyChildren(document))
			}

			div := parsedBodyChildren(document)[0]
			if div.TextContent != testCase.wantValue || len(div.Children) != 1 || div.Children[0].Type != types.TextNode {
				t.Fatalf("Expected exactly one decoded text node %q, got %#v", testCase.wantValue, div)
			}
			text := div.Children[0]
			if text.Value != testCase.wantValue || text.TextContent != testCase.wantValue {
				t.Fatalf("Expected decoded value %q, got %#v", testCase.wantValue, text)
			}
			wantStart := len(`<div>`)
			wantEnd := wantStart + len(testCase.raw)
			if text.StartPos != wantStart || text.EndPos != wantEnd {
				t.Fatalf("Expected raw source byte range %d:%d, got %d:%d", wantStart, wantEnd, text.StartPos, text.EndPos)
			}
			if source := content[text.StartPos:text.EndPos]; source != testCase.raw {
				t.Fatalf("Expected raw reference source %q, got %q", testCase.raw, source)
			}
		})
	}
}

func TestParseSemicolonlessNamedCharacterReferenceAtEOF(t *testing.T) {
	const content = `a&amp`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Type != types.TextNode {
		t.Fatalf("Expected one EOF text node, got %#v", parsedBodyChildren(document))
	}
	text := parsedBodyChildren(document)[0]
	if text.Value != "a&" || text.StartPos != 0 || text.EndPos != len(content) {
		t.Fatalf("Expected decoded ampersand over full EOF range, got %#v", text)
	}
}

func TestParseNamedCharacterReferencesInAttributesLikeBrowser(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		wantValue string
	}{
		{name: "quoted semicolon", content: `<div data-x="a&copy;b">x</div>`, wantValue: "a©b"},
		{name: "quoted ambiguous alphanumeric stays literal", content: `<div data-x="a&copycat">x</div>`, wantValue: `a&copycat`},
		{name: "quoted ambiguous equals stays literal", content: `<div data-x="a&copy=1">x</div>`, wantValue: `a&copy=1`},
		{name: "quoted legacy before space decodes", content: `<div data-x="a&copy z">x</div>`, wantValue: "a© z"},
		{name: "quoted semicolon before alphanumeric decodes", content: `<div data-x="a&copy;cat">x</div>`, wantValue: "a©cat"},
		{name: "unquoted semicolon", content: `<div data-x=a&amp;b>x</div>`, wantValue: `a&b`},
		{name: "unquoted ambiguous alphanumeric stays literal", content: `<div data-x=a&ampb>x</div>`, wantValue: `a&ampb`},
		{name: "unquoted ambiguous equals stays literal", content: `<div data-x=a&amp=1>x</div>`, wantValue: `a&amp=1`},
		{name: "unknown remains literal", content: `<div data-x="a&wat;b">x</div>`, wantValue: `a&wat;b`},
		{name: "multi-codepoint reference", content: `<div data-x="a&NotEqualTilde;b">x</div>`, wantValue: "a≂̸b"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
				t.Fatalf("Expected one div root, got %#v", parsedBodyChildren(document))
			}
			div := parsedBodyChildren(document)[0]
			if got := div.Attributes["data-x"]; got != testCase.wantValue {
				t.Fatalf("Expected attribute value %q, got %q", testCase.wantValue, got)
			}
			if div.StartPos != 0 || div.EndPos != len(testCase.content) {
				t.Fatalf("Expected exact original element range 0:%d, got %d:%d", len(testCase.content), div.StartPos, div.EndPos)
			}
		})
	}
}

func TestParseUnknownLongerNamedReferencesUsesLegacyPrefixOnlyInData(t *testing.T) {
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
			content := `<div>` + testCase.raw + `</div>`
			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" || len(parsedBodyChildren(document)[0].Children) != 1 {
				t.Fatalf("Expected one div with one text node, got %#v", parsedBodyChildren(document))
			}
			text := parsedBodyChildren(document)[0].Children[0]
			if text.Type != types.TextNode || text.Value != testCase.wantValue {
				t.Fatalf("Expected legacy-prefix data value %q, got %#v", testCase.wantValue, text)
			}
			wantStart := len(`<div>`)
			wantEnd := wantStart + len(testCase.raw)
			if text.StartPos != wantStart || text.EndPos != wantEnd {
				t.Fatalf("Expected exact raw data range %d:%d, got %d:%d", wantStart, wantEnd, text.StartPos, text.EndPos)
			}
			if source := content[text.StartPos:text.EndPos]; source != testCase.raw {
				t.Fatalf("Expected raw source %q, got %q", testCase.raw, source)
			}
		})
	}
}

func TestParseUnknownLongerNamedReferencesRemainLiteralInAttributes(t *testing.T) {
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
			content := `<div data-x="` + testCase.raw + `">x</div>`
			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
				t.Fatalf("Expected one div, got %#v", parsedBodyChildren(document))
			}
			div := parsedBodyChildren(document)[0]
			if got := div.Attributes["data-x"]; got != testCase.wantValue {
				t.Fatalf("Expected attribute value %q, got %q", testCase.wantValue, got)
			}
			if div.StartPos != 0 || div.EndPos != len(content) {
				t.Fatalf("Expected exact original element range 0:%d, got %d:%d", len(content), div.StartPos, div.EndPos)
			}
		})
	}
}

func TestParseLongUnknownNamedReferenceRunContinuesAtFollowingKnownReference(t *testing.T) {
	const runLength = 4096
	unknownRun := strings.Repeat("a", runLength)
	raw := `&` + unknownRun + `;&copy;z`
	content := `<div>` + raw + `</div>`
	wantValue := `&` + unknownRun + `;©z`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
		t.Fatalf("Expected one div root, got %#v", parsedBodyChildren(document))
	}
	div := parsedBodyChildren(document)[0]
	if div.TextContent != wantValue || len(div.Children) != 1 || div.Children[0].Type != types.TextNode {
		t.Fatalf("Expected one text node with literal unknown run and decoded copy suffix; children=%d text length=%d", len(div.Children), len(div.TextContent))
	}
	text := div.Children[0]
	wantStart := len(`<div>`)
	wantEnd := wantStart + len(raw)
	if text.StartPos != wantStart || text.EndPos != wantEnd {
		t.Fatalf("Expected complete raw source range %d:%d, got %d:%d", wantStart, wantEnd, text.StartPos, text.EndPos)
	}
	if source := content[text.StartPos:text.EndPos]; source != raw {
		t.Fatalf("Expected exact raw inner source of length %d, got length %d", len(raw), len(source))
	}
}

func TestConsumeNamedCharacterReferenceHasBoundedWorkForLongUnknownRun(t *testing.T) {
	const runLength = 32768
	content := `&` + strings.Repeat("a", runLength) + `;&copy;`
	parser := NewHTMLParser()
	parser.content = content

	// An implementation capped near the longest WHATWG entity name performs
	// bounded lookups here. Scanning and allocating every growing prefix is an
	// obvious quadratic regression, independent of machine timing.
	allocations := testing.AllocsPerRun(1, func() {
		parser.pos = 0
		parser.line = 1
		parser.col = 1
		value, matched := parser.consumeNamedCharacterReference(false)
		if matched || value != "" || parser.pos != 0 {
			panic("unknown run must remain unconsumed")
		}
	})
	if allocations > 256 {
		t.Fatalf("Expected bounded named-reference lookup allocations, got %.0f for a %d-byte run", allocations, runLength)
	}
}
