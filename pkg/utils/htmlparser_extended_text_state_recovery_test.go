package utils

import (
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

var extendedRawTextTags = []string{"xmp", "iframe", "noembed", "noframes"}

func assertExtendedTextElement(t *testing.T, content, tag, wantValue string, wantTextStart, wantTextEnd int) *types.Node {
	t.Helper()
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	elements := listElements(document, tag)
	if len(elements) != 1 {
		t.Fatalf("Expected one <%s>, got %#v", tag, elements)
	}
	element := elements[0]
	if element.TextContent != wantValue || len(element.Children) != 1 || element.Children[0].Type != types.TextNode {
		t.Fatalf("Expected one <%s> text node %q, got %#v", tag, wantValue, element)
	}
	text := element.Children[0]
	if text.Value != wantValue || text.StartPos != wantTextStart || text.EndPos != wantTextEnd {
		t.Fatalf("Expected <%s> text %q at %d:%d, got %#v", tag, wantValue, wantTextStart, wantTextEnd, text)
	}
	return element
}

func TestParseRemainingRawTextElementsKeepMarkupAndReferencesLiteral(t *testing.T) {
	const raw = `é<b>&amp;&#65;`
	for _, tag := range extendedRawTextTags {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			closing := `</` + strings.ToUpper(tag) + `>`
			content := opening + raw + closing + `<p>after</p>`
			textStart := len(opening)
			textEnd := textStart + len(raw)
			element := assertExtendedTextElement(t, content, tag, raw, textStart, textEnd)
			if source := content[textStart:textEnd]; source != raw {
				t.Fatalf("Expected exact raw UTF-8 source %q, got %q", raw, source)
			}
			if element.StartPos != 0 || element.EndPos != textEnd+len(closing) {
				t.Fatalf("Expected exact <%s> range 0:%d, got %d:%d", tag, textEnd+len(closing), element.StartPos, element.EndPos)
			}
			root := element
			for root.Parent != nil {
				root = root.Parent
			}
			paragraphs := listElements(root, "p")
			if len(paragraphs) != 1 || paragraphs[0].TextContent != "after" {
				t.Fatalf("Expected parsing to continue with p after <%s>, got %#v", tag, paragraphs)
			}
			text := element.Children[0]
			wantStartColumn := len(opening) + 1
			wantEndColumn := wantStartColumn + len([]rune(raw))
			if text.StartLine != 1 || text.StartColumn != wantStartColumn || text.EndLine != 1 || text.EndColumn != wantEndColumn {
				t.Fatalf("Expected parse5 coordinates 1:%d-1:%d, got %d:%d-%d:%d", wantStartColumn, wantEndColumn, text.StartLine, text.StartColumn, text.EndLine, text.EndColumn)
			}
		})
	}
}

func TestParseRemainingRawTextElementsKeepNonmatchingEndTagsLiteral(t *testing.T) {
	for _, tag := range extendedRawTextTags {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			raw := `x</` + tag + `-x>y`
			content := opening + raw + `</` + tag + `>`
			assertExtendedTextElement(t, content, tag, raw, len(opening), len(opening)+len(raw))
		})
	}
}

func TestParseRemainingRawTextElementsReplaceNUL(t *testing.T) {
	for _, tag := range extendedRawTextTags {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			content := opening + "a\x00b</" + tag + `>`
			assertExtendedTextElement(t, content, tag, "a\uFFFDb", len(opening), len(opening)+3)
		})
	}
}

func TestParseRemainingRawTextElementsCloseAtEOF(t *testing.T) {
	const raw = `x&amp;<b>`
	for _, tag := range extendedRawTextTags {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			content := opening + raw
			element := assertExtendedTextElement(t, content, tag, raw, len(opening), len(content))
			if element.EndPos != len(content) {
				t.Fatalf("Expected <%s> to close at EOF %d, got %d", tag, len(content), element.EndPos)
			}
		})
	}
}

func TestParsePlaintextConsumesEverythingThroughEOF(t *testing.T) {
	const opening = `<plaintext>`
	const raw = "é\n<b>x</b>&amp;&#65;</plaintext><p>after</p>"
	content := opening + raw
	element := assertExtendedTextElement(t, content, "plaintext", raw, len(opening), len(content))
	if element.EndPos != len(content) || len(element.Parent.Children) != 1 {
		t.Fatalf("Expected plaintext to consume the apparent closing tag and following p through EOF, got %#v", element.Parent.Children)
	}
	text := element.Children[0]
	if source := content[text.StartPos:text.EndPos]; source != raw {
		t.Fatalf("Expected exact multibyte plaintext source %q, got %q", raw, source)
	}
	if text.StartLine != 1 || text.StartColumn != 12 || text.EndLine != 2 || text.EndColumn != 43 {
		t.Fatalf("Expected parse5 plaintext coordinates 1:12-2:43, got %d:%d-%d:%d", text.StartLine, text.StartColumn, text.EndLine, text.EndColumn)
	}
}

func TestParsePlaintextReplacesNULThroughEOF(t *testing.T) {
	const content = "<plaintext>a\x00b"
	element := assertExtendedTextElement(t, content, "plaintext", "a\uFFFDb", len(`<plaintext>`), len(content))
	if element.EndPos != len(content) {
		t.Fatalf("Expected plaintext to end at EOF %d, got %d", len(content), element.EndPos)
	}
}

func TestParseNestedExtendedTextStatesCloseAllAncestorsAtEOF(t *testing.T) {
	tags := append(append([]string{}, extendedRawTextTags...), "plaintext")
	for _, tag := range tags {
		t.Run(tag, func(t *testing.T) {
			content := `<div><` + tag + `>x`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
				t.Fatalf("Expected one outer div, got %#v", parsedBodyChildren(document))
			}
			div := parsedBodyChildren(document)[0]
			if div.StartPos != 0 || div.EndPos != len(content) || div.TextContent != "x" || len(div.Children) != 1 {
				t.Fatalf("Expected outer div x at 0:%d, got %#v", len(content), div)
			}
			inner := div.Children[0]
			wantInnerStart := len(`<div>`)
			wantTextStart := wantInnerStart + len(`<`+tag+`>`)
			if inner.Name != tag || inner.StartPos != wantInnerStart || inner.EndPos != len(content) || inner.TextContent != "x" || len(inner.Children) != 1 {
				t.Fatalf("Expected nested <%s> x at %d:%d, got %#v", tag, wantInnerStart, len(content), inner)
			}
			text := inner.Children[0]
			if text.Type != types.TextNode || text.Value != "x" || text.StartPos != wantTextStart || text.EndPos != len(content) {
				t.Fatalf("Expected exact nested text range %d:%d, got %#v", wantTextStart, len(content), text)
			}
		})
	}
}

func TestParseEmptyExtendedTextStatesLikeBrowser(t *testing.T) {
	testCases := []struct {
		tag, content string
	}{
		{tag: "xmp", content: `<xmp></xmp>`},
		{tag: "plaintext", content: `<plaintext>`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.tag, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 {
				t.Fatalf("Expected one empty <%s>, got %#v", testCase.tag, parsedBodyChildren(document))
			}
			element := parsedBodyChildren(document)[0]
			if element.Name != testCase.tag || element.TextContent != "" || len(element.Children) != 0 || element.StartPos != 0 || element.EndPos != len(testCase.content) {
				t.Fatalf("Expected empty <%s> at 0:%d, got %#v", testCase.tag, len(testCase.content), element)
			}
		})
	}
}

func TestParsePlaintextNormalizesCRLFAndTracksSupplementaryCoordinates(t *testing.T) {
	const content = "<plaintext>🙂\r\nz"
	element := assertExtendedTextElement(t, content, "plaintext", "🙂\nz", len(`<plaintext>`), len(content))
	text := element.Children[0]
	if text.StartLine != 1 || text.StartColumn != 12 || text.EndLine != 2 || text.EndColumn != 2 {
		t.Fatalf("Expected parse5 plaintext coordinates 1:12-2:2, got %d:%d-%d:%d", text.StartLine, text.StartColumn, text.EndLine, text.EndColumn)
	}
	if source := content[text.StartPos:text.EndPos]; source != "🙂\r\nz" {
		t.Fatalf("Expected exact supplementary/CRLF source, got %q", source)
	}
}

func TestParseExtendedRawTextEndTagDelimiterAndIncompleteEOF(t *testing.T) {
	t.Run("whitespace delimiter closes and ignores attributes", func(t *testing.T) {
		const content = `<iframe>x</IFRAME foo=bar><p>y</p>`
		const endTagStart = len(`<iframe>x`)
		element := assertExtendedTextElement(t, content, "iframe", "x", len(`<iframe>`), endTagStart)
		if element.EndPos != len(`<iframe>x</IFRAME foo=bar>`) || len(element.Parent.Children) != 2 || element.Parent.Children[1].Name != "p" {
			t.Fatalf("Expected iframe delimiter recovery and following p, got %#v", element.Parent.Children)
		}
	})

	t.Run("incomplete appropriate end tag is discarded at EOF", func(t *testing.T) {
		const content = `<iframe>x</iframe `
		element := assertExtendedTextElement(t, content, "iframe", "x", len(`<iframe>`), len(content))
		if element.EndPos != len(content) {
			t.Fatalf("Expected iframe to close at EOF %d, got %d", len(content), element.EndPos)
		}
	})
}

func TestHTMLParserReuseAfterExtendedTextStateEOF(t *testing.T) {
	parser := NewHTMLParser()
	first, err := parser.Parse(`<plaintext>x`)
	if err != nil {
		t.Fatalf("Expected first plaintext parse, got error: %v", err)
	}
	if len(parsedBodyChildren(first)) != 1 || parsedBodyChildren(first)[0].Name != "plaintext" || parsedBodyChildren(first)[0].TextContent != "x" {
		t.Fatalf("Expected first plaintext parse, document=%#v err=%v", first, err)
	}
	second, err := parser.Parse(`<p>ok</p>`)
	if err != nil {
		t.Fatalf("Parser reuse returned an error: %v", err)
	}
	if len(parsedBodyChildren(second)) != 1 || parsedBodyChildren(second)[0].Name != "p" || parsedBodyChildren(second)[0].TextContent != "ok" {
		t.Fatalf("Expected parser state to reset for an implicit-body p, got %#v", parsedBodyChildren(second))
	}
}
