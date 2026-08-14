package utils

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseNumericCharacterReferencesInDataLikeBrowser(t *testing.T) {
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
				t.Fatalf("Expected one decoded text value %q, got %#v", testCase.wantValue, div)
			}
			text := div.Children[0]
			if text.Value != testCase.wantValue || text.TextContent != testCase.wantValue {
				t.Fatalf("Expected decoded node value %q, got %#v", testCase.wantValue, text)
			}
			wantStart := len(`<div>`)
			wantEnd := wantStart + len(testCase.raw)
			if text.StartPos != wantStart || text.EndPos != wantEnd {
				t.Fatalf("Expected original byte range %d:%d, got %d:%d", wantStart, wantEnd, text.StartPos, text.EndPos)
			}
			if source := content[text.StartPos:text.EndPos]; source != testCase.raw {
				t.Fatalf("Expected decoded text range to retain raw source %q, got %q", testCase.raw, source)
			}
		})
	}
}

func TestParseNumericCharacterReferencesInAttributesLikeBrowser(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		wantValue string
	}{
		{name: "double-quoted decimal", content: `<div data-x="a&#65;b">t</div>`, wantValue: "aAb"},
		{name: "single-quoted uppercase X", content: `<div data-x='a&#X41;b'>t</div>`, wantValue: "aAb"},
		{name: "unquoted missing semicolon", content: `<div data-x=a&#x41z>t</div>`, wantValue: "aAz"},
		{name: "no hexadecimal digits", content: `<div data-x="a&#xZb">t</div>`, wantValue: `a&#xZb`},
		{name: "NUL becomes replacement", content: `<div data-x="a&#0;b">t</div>`, wantValue: "a\uFFFDb"},
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
				t.Fatalf("Expected decoded attribute value %q, got %q", testCase.wantValue, got)
			}
			if div.StartPos != 0 || div.EndPos != len(testCase.content) {
				t.Fatalf("Expected full original element range 0:%d, got %d:%d", len(testCase.content), div.StartPos, div.EndPos)
			}
		})
	}
}

func TestParseLeadingDecodedWhitespaceDoesNotPreventInitialDoctype(t *testing.T) {
	const content = `&#32;<!DOCTYPE html><p>x</p>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 2 || document.Children[0].Type != types.DocumentTypeNode || len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" {
		t.Fatalf("Expected leading decoded whitespace to be discarded before doctype and implicit document: document=%#v body=%#v", document.Children, parsedBodyChildren(document))
	}
	doctype := document.Children[0]
	if doctype.Name != "html" || doctype.StartPos != len(`&#32;`) || doctype.EndPos != len(`&#32;<!DOCTYPE html>`) {
		t.Fatalf("Expected exact accepted doctype range, got %#v", doctype)
	}
	paragraph := parsedBodyChildren(document)[0]
	if paragraph.StartPos != len(`&#32;<!DOCTYPE html>`) || paragraph.EndPos != len(content) || paragraph.TextContent != "x" {
		t.Fatalf("Expected exact continuation paragraph, got %#v", paragraph)
	}
}

func TestParseDecodedWhitespaceAfterBodyContentIsPreserved(t *testing.T) {
	const content = `<!DOCTYPE html><p>x</p>&#32;`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 2 || document.Children[0].Type != types.DocumentTypeNode || len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0].Name != "p" || parsedBodyChildren(document)[1].Type != types.TextNode {
		t.Fatalf("Expected document doctype plus body p and trailing decoded whitespace, got document=%#v body=%#v", document.Children, parsedBodyChildren(document))
	}
	text := parsedBodyChildren(document)[1]
	wantStart := len(`<!DOCTYPE html><p>x</p>`)
	if text.Value != " " || text.StartPos != wantStart || text.EndPos != len(content) {
		t.Fatalf("Expected decoded trailing space with raw range %d:%d, got %#v", wantStart, len(content), text)
	}
	if source := content[text.StartPos:text.EndPos]; source != `&#32;` {
		t.Fatalf("Expected exact raw reference source, got %q", source)
	}
}

func TestParseMixedLeadingDecodedWhitespaceBeforeDoctypeIsDiscarded(t *testing.T) {
	content := `&#32; ` + "\t" + `&#x20;<!DOCTYPE html><p>x</p>`
	prefix := `&#32; ` + "\t" + `&#x20;`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 2 || document.Children[0].Type != types.DocumentTypeNode || document.Children[1].Name != "html" || len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" {
		t.Fatalf("Expected all leading decoded/literal whitespace discarded before a document doctype and body p, got document=%#v body=%#v", document.Children, parsedBodyChildren(document))
	}
	doctype := document.Children[0]
	if doctype.Name != "html" || doctype.StartPos != len(prefix) || doctype.EndPos != len(prefix)+len(`<!DOCTYPE html>`) {
		t.Fatalf("Expected accepted doctype after exact whitespace prefix, got %#v", doctype)
	}
}

func TestParseLeadingDecodedNonWhitespaceCommitsBodyAndIgnoresDoctype(t *testing.T) {
	const content = `&#65;<!DOCTYPE html><p>x</p>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0].Type != types.TextNode || parsedBodyChildren(document)[1].Name != "p" {
		t.Fatalf("Expected decoded A and p with doctype ignored, got %#v", parsedBodyChildren(document))
	}
	text := parsedBodyChildren(document)[0]
	if text.Value != "A" || text.StartPos != 0 || text.EndPos != len(`&#65;`) {
		t.Fatalf("Expected decoded A at exact raw range, got %#v", text)
	}
	paragraph := parsedBodyChildren(document)[1]
	if paragraph.StartPos != len(`&#65;<!DOCTYPE html>`) || paragraph.EndPos != len(content) {
		t.Fatalf("Expected paragraph after ignored doctype at exact range, got %#v", paragraph)
	}
}

func TestParseMultibyteBodyTextAndDecodedSpaceUseOriginalByteRange(t *testing.T) {
	const content = `é&#32;<!DOCTYPE html><p>x</p>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0].Type != types.TextNode || parsedBodyChildren(document)[1].Name != "p" {
		t.Fatalf("Expected multibyte body text and p with doctype ignored, got %#v", parsedBodyChildren(document))
	}
	text := parsedBodyChildren(document)[0]
	if text.Value != "é " || text.StartPos != 0 || text.EndPos != len(`é&#32;`) {
		t.Fatalf("Expected decoded multibyte text at byte range 0:%d, got %#v", len(`é&#32;`), text)
	}
	if source := content[text.StartPos:text.EndPos]; source != `é&#32;` {
		t.Fatalf("Expected exact original multibyte/reference source, got %q", source)
	}
}
