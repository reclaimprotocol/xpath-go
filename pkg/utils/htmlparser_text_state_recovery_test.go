package utils

import (
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func assertTextStateElement(t *testing.T, content, tag, wantValue string, wantTextStart, wantTextEnd int) *types.Node {
	t.Helper()
	parser := NewHTMLParser()
	document, err := parser.Parse(content)
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
		t.Fatalf("Expected text %q at %d:%d, got %#v", wantValue, wantTextStart, wantTextEnd, text)
	}
	return element
}

func TestParseRCDATAAndRawTextCharacterReferencesLikeBrowser(t *testing.T) {
	testCases := []struct {
		name      string
		tag       string
		raw       string
		wantValue string
	}{
		{name: "title decodes named numeric and less-than", tag: "title", raw: `a&amp;&#65;&lt;b&gt;`, wantValue: `a&A<b>`},
		{name: "textarea decodes named numeric and less-than", tag: "textarea", raw: `a&amp;&#65;&lt;b&gt;`, wantValue: `a&A<b>`},
		{name: "style keeps references literal", tag: "style", raw: `a&amp;&#65;&lt;`, wantValue: `a&amp;&#65;&lt;`},
		{name: "script keeps references literal", tag: "script", raw: `a&amp;&#65;&lt;`, wantValue: `a&amp;&#65;&lt;`},
		{name: "multibyte RCDATA preserves UTF-8 locations", tag: "title", raw: `é&amp;&#x1F642;z`, wantValue: "é&🙂z"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			opening := `<` + testCase.tag + `>`
			content := opening + testCase.raw + `</` + testCase.tag + `>`
			start := len(opening)
			end := start + len(testCase.raw)
			element := assertTextStateElement(t, content, testCase.tag, testCase.wantValue, start, end)
			if source := content[start:end]; source != testCase.raw {
				t.Fatalf("Expected exact raw source %q, got %q", testCase.raw, source)
			}
			if testCase.name == "multibyte RCDATA preserves UTF-8 locations" {
				text := element.Children[0]
				if text.StartLine != 1 || text.StartColumn != 8 || text.EndLine != 1 || text.EndColumn != 24 {
					t.Fatalf("Expected parse5 text coordinates 1:8-1:24, got %d:%d-%d:%d", text.StartLine, text.StartColumn, text.EndLine, text.EndColumn)
				}
			}
		})
	}
}

func TestParseTextStateAppropriateEndTagCandidatesLikeBrowser(t *testing.T) {
	testCases := []struct {
		name          string
		content       string
		tag           string
		wantValue     string
		wantElement   string
		wantFollowing bool
	}{
		{name: "ASCII case-insensitive close", content: `<title>before</TITLE><p>after</p>`, tag: "title", wantValue: "before", wantElement: `<title>before</TITLE>`, wantFollowing: true},
		{name: "nonmatching similar name stays text", content: `<title>a</title-x>b</title>`, tag: "title", wantValue: `a</title-x>b`, wantElement: `<title>a</title-x>b</title>`},
		{name: "malformed candidate stays text", content: `<title>a</title!>b</title>`, tag: "title", wantValue: `a</title!>b`, wantElement: `<title>a</title!>b</title>`},
		{name: "whitespace end tag closes", content: `<title>x</title   ><p>y</p>`, tag: "title", wantValue: "x", wantElement: `<title>x</title   >`, wantFollowing: true},
		{name: "end tag attributes ignored", content: `<textarea>x</textarea foo=bar><p>y</p>`, tag: "textarea", wantValue: "x", wantElement: `<textarea>x</textarea foo=bar>`, wantFollowing: true},
		{name: "trailing slash ignored", content: `<style>x</style/><p>y</p>`, tag: "style", wantValue: "x", wantElement: `<style>x</style/>`, wantFollowing: true},
		{name: "less-than start opener stays text", content: `<title>a<b>c</title>`, tag: "title", wantValue: `a<b>c`, wantElement: `<title>a<b>c</title>`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			openingEnd := strings.Index(testCase.content, ">") + 1
			closingStart := len(testCase.wantElement) - len(`</`+testCase.tag+`>`)
			if strings.Contains(testCase.wantElement, `</`+testCase.tag+` `) || strings.Contains(testCase.wantElement, `</`+testCase.tag+`/`) {
				closingStart = strings.LastIndex(testCase.wantElement, `</`)
			}
			if testCase.name == "ASCII case-insensitive close" {
				closingStart = strings.LastIndex(testCase.wantElement, `</`)
			}
			element := assertTextStateElement(t, testCase.content, testCase.tag, testCase.wantValue, openingEnd, closingStart)
			if element.StartPos != 0 || element.EndPos != len(testCase.wantElement) {
				t.Fatalf("Expected exact element range 0:%d, got %d:%d", len(testCase.wantElement), element.StartPos, element.EndPos)
			}
			if source := testCase.content[element.StartPos:element.EndPos]; source != testCase.wantElement {
				t.Fatalf("Expected exact element source %q, got %q", testCase.wantElement, source)
			}
			if testCase.wantFollowing {
				document := element
				for document.Parent != nil {
					document = document.Parent
				}
				paragraphs := listElements(document, "p")
				if len(paragraphs) != 1 || paragraphs[0].TextContent == "" {
					t.Fatalf("Expected parsing to continue with one body p, got %#v", paragraphs)
				}
			}
		})
	}
}

func TestParseTextStatesReplaceNULLikeBrowser(t *testing.T) {
	for _, tag := range []string{"title", "textarea", "style", "script"} {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			content := opening + "a\x00b</" + tag + `>`
			assertTextStateElement(t, content, tag, "a\uFFFDb", len(opening), len(opening)+3)
		})
	}
}

func TestParseTextStatesCloseAtEOFLIkeBrowser(t *testing.T) {
	testCases := []struct {
		tag       string
		raw       string
		wantValue string
	}{
		{tag: "title", raw: `x&amp;`, wantValue: `x&`},
		{tag: "textarea", raw: `x&#65;`, wantValue: `xA`},
		{tag: "style", raw: `x&amp;`, wantValue: `x&amp;`},
		{tag: "script", raw: `x&amp;`, wantValue: `x&amp;`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.tag, func(t *testing.T) {
			opening := `<` + testCase.tag + `>`
			content := opening + testCase.raw
			element := assertTextStateElement(t, content, testCase.tag, testCase.wantValue, len(opening), len(content))
			if element.EndPos != len(content) {
				t.Fatalf("Expected <%s> to close at EOF %d, got %d", testCase.tag, len(content), element.EndPos)
			}
		})
	}
}

func TestParseTextareaStripsOnlyInitialNewlineFromValueAndLocation(t *testing.T) {
	const content = "<textarea>\nabc</textarea>"
	element := assertTextStateElement(t, content, "textarea", "abc", len("<textarea>\n"), len("<textarea>\nabc"))
	if source := content[element.Children[0].StartPos:element.Children[0].EndPos]; source != "abc" {
		t.Fatalf("Expected stripped newline to be excluded from text location, got %q", source)
	}
}

func TestParseTextareaInitialNewlineRecoveryLikeBrowser(t *testing.T) {
	testCases := []struct {
		name                           string
		raw                            string
		wantValue, wantSource          string
		wantStart, wantEnd             int
		wantStartLine, wantStartColumn int
		wantEndLine, wantEndColumn     int
	}{
		{
			name: "CRLF is normalized then stripped", raw: "\r\nabc", wantValue: "abc", wantSource: "abc",
			wantStart: 12, wantEnd: 15, wantStartLine: 2, wantStartColumn: 1, wantEndLine: 2, wantEndColumn: 4,
		},
		{
			name: "lone CR is normalized then stripped", raw: "\rabc", wantValue: "abc", wantSource: "abc",
			wantStart: 11, wantEnd: 14, wantStartLine: 2, wantStartColumn: 1, wantEndLine: 2, wantEndColumn: 4,
		},
		{
			name: "double LF retains one DOM newline and both raw bytes", raw: "\n\nabc", wantValue: "\nabc", wantSource: "\n\nabc",
			wantStart: 10, wantEnd: 15, wantStartLine: 1, wantStartColumn: 11, wantEndLine: 3, wantEndColumn: 4,
		},
		{
			name: "decoded numeric LF is stripped", raw: "&#10;abc", wantValue: "abc", wantSource: "abc",
			wantStart: 15, wantEnd: 18, wantStartLine: 1, wantStartColumn: 16, wantEndLine: 1, wantEndColumn: 19,
		},
		{
			name: "decoded named LF is stripped", raw: "&NewLine;abc", wantValue: "abc", wantSource: "abc",
			wantStart: 19, wantEnd: 22, wantStartLine: 1, wantStartColumn: 20, wantEndLine: 1, wantEndColumn: 23,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := "<textarea>" + testCase.raw + "</textarea>"
			element := assertTextStateElement(t, content, "textarea", testCase.wantValue, testCase.wantStart, testCase.wantEnd)
			text := element.Children[0]
			if source := content[text.StartPos:text.EndPos]; source != testCase.wantSource {
				t.Fatalf("Expected browser raw source %q, got %q", testCase.wantSource, source)
			}
			if text.StartLine != testCase.wantStartLine || text.StartColumn != testCase.wantStartColumn || text.EndLine != testCase.wantEndLine || text.EndColumn != testCase.wantEndColumn {
				t.Fatalf(
					"Expected parse5 text coordinates %d:%d-%d:%d, got %d:%d-%d:%d",
					testCase.wantStartLine, testCase.wantStartColumn, testCase.wantEndLine, testCase.wantEndColumn,
					text.StartLine, text.StartColumn, text.EndLine, text.EndColumn,
				)
			}
		})
	}
}

func TestParseTitleIncompleteEndTagCandidatesAtEOFLikeBrowser(t *testing.T) {
	testCases := []struct {
		name, content, wantValue string
	}{
		{name: "name without delimiter remains RCDATA", content: "<title>x</title", wantValue: "x</title"},
		{name: "whitespace discards incomplete end tag", content: "<title>x</title ", wantValue: "x"},
		{name: "unterminated end tag attribute is discarded", content: `<title>x</title a="`, wantValue: "x"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			element := assertTextStateElement(t, testCase.content, "title", testCase.wantValue, len("<title>"), len(testCase.content))
			text := element.Children[0]
			if source := testCase.content[text.StartPos:text.EndPos]; source != testCase.content[len("<title>"):] {
				t.Fatalf("Expected text range to cover the exact remaining source, got %q", source)
			}
			if element.EndPos != len(testCase.content) {
				t.Fatalf("Expected title to close at EOF %d, got %d", len(testCase.content), element.EndPos)
			}
		})
	}
}
