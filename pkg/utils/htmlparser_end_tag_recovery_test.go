package utils

import (
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseIgnoresMissingEndTagNameLikeBrowser(t *testing.T) {
	const content = `<div>before</>after<span>tail</span></div>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
		t.Fatalf("Expected one div, got %#v", parsedBodyChildren(document))
	}

	div := parsedBodyChildren(document)[0]
	if div.TextContent != "beforeaftertail" {
		t.Fatalf("Expected ignored </> to contribute no text, got %q", div.TextContent)
	}
	if len(div.Children) != 2 || div.Children[0].Type != types.TextNode || div.Children[1].Name != "span" {
		t.Fatalf("Expected merged text followed by span, got %#v", div.Children)
	}

	text := div.Children[0]
	if text.Value != "beforeafter" {
		t.Fatalf("Expected adjacent browser text to merge, got %q", text.Value)
	}
	wantStart := strings.Index(content, "before")
	wantEnd := strings.Index(content, `<span>`)
	if text.StartPos != wantStart || text.EndPos != wantEnd {
		t.Fatalf("Expected text range %d:%d spanning ignored token, got %d:%d", wantStart, wantEnd, text.StartPos, text.EndPos)
	}
	if source := content[text.StartPos:text.EndPos]; source != `before</>after` {
		t.Fatalf("Expected original range to retain ignored token, got %q", source)
	}
}

func TestParseRecoversInvalidEndTagOpenAsBogusCommentLikeBrowser(t *testing.T) {
	testCases := []struct {
		name      string
		token     string
		wantValue string
	}{
		{name: "numeric first character", token: `</42>`, wantValue: "42"},
		{name: "dollar first character", token: `</$foo>`, wantValue: "$foo"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := `<div>before` + testCase.token + `after<span>tail</span></div>`
			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
				t.Fatalf("Expected one div, got %#v", parsedBodyChildren(document))
			}

			div := parsedBodyChildren(document)[0]
			if div.TextContent != "beforeaftertail" || len(div.Children) != 4 {
				t.Fatalf("Expected text, comment, text, and span continuation, got %#v", div.Children)
			}
			comment := div.Children[1]
			if comment.Type != types.CommentNode || comment.Value != testCase.wantValue {
				t.Fatalf("Expected bogus comment %q, got %#v", testCase.wantValue, comment)
			}
			if source := content[comment.StartPos:comment.EndPos]; source != testCase.token {
				t.Fatalf("Expected exact bogus end-tag source %q, got %q", testCase.token, source)
			}
			span := div.Children[3]
			if span.Name != "span" || content[span.StartPos:span.EndPos] != `<span>tail</span>` {
				t.Fatalf("Expected parsing to continue at exact span source, got %#v", span)
			}
		})
	}
}

func TestParseRecoversEndTagEOFStatesLikeBrowser(t *testing.T) {
	t.Run("EOF after end tag opener is text", func(t *testing.T) {
		const content = `before</`
		parser := NewHTMLParser()
		document, err := parser.Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Type != types.TextNode {
			t.Fatalf("Expected one text node, got %#v", parsedBodyChildren(document))
		}
		text := parsedBodyChildren(document)[0]
		if text.Value != content || text.StartPos != 0 || text.EndPos != len(content) {
			t.Fatalf("Expected literal end-tag opener with full range, got %#v", text)
		}
	})

	t.Run("EOF inside end tag ignores token but extends locations", func(t *testing.T) {
		const content = `<div>before</div`
		parser := NewHTMLParser()
		document, err := parser.Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
			t.Fatalf("Expected one implicitly closed div, got %#v", parsedBodyChildren(document))
		}

		div := parsedBodyChildren(document)[0]
		if div.TextContent != "before" || div.StartPos != 0 || div.EndPos != len(content) {
			t.Fatalf("Expected div location through ignored end tag, got %#v", div)
		}
		if len(div.Children) != 1 || div.Children[0].Type != types.TextNode {
			t.Fatalf("Expected one text child, got %#v", div.Children)
		}
		text := div.Children[0]
		if text.Value != "before" || text.StartPos != len(`<div>`) || text.EndPos != len(content) {
			t.Fatalf("Expected text location through ignored EOF token, got %#v", text)
		}
		if source := content[text.StartPos:text.EndPos]; source != `before</div` {
			t.Fatalf("Expected exact original text range including ignored token, got %q", source)
		}
	})
}

func TestParseIgnoresEndTagAttributesAndTrailingSolidusLikeBrowser(t *testing.T) {
	testCases := []struct {
		name       string
		closingTag string
	}{
		{name: "unquoted attribute", closingTag: `</div foo=bar>`},
		{name: "quoted and boolean attributes", closingTag: `</div foo='bar' baz>`},
		{name: "whitespace before close", closingTag: `</div   >`},
		{name: "trailing solidus", closingTag: `</div/>`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := `<div>payload` + testCase.closingTag + `<span>tail</span>`
			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0].Name != "div" || parsedBodyChildren(document)[1].Name != "span" {
				t.Fatalf("Expected closed div followed by span, got %#v", parsedBodyChildren(document))
			}

			div := parsedBodyChildren(document)[0]
			if div.TextContent != "payload" || len(div.Attributes) != 0 {
				t.Fatalf("Expected end-tag junk to be ignored, got %#v", div)
			}
			wantDiv := `<div>payload` + testCase.closingTag
			if source := content[div.StartPos:div.EndPos]; source != wantDiv {
				t.Fatalf("Expected exact div source %q, got %q", wantDiv, source)
			}
			span := parsedBodyChildren(document)[1]
			if source := content[span.StartPos:span.EndPos]; source != `<span>tail</span>` {
				t.Fatalf("Expected exact following span source, got %q", source)
			}
		})
	}
}

func TestParseUsesHTMLTagNameRulesForEndTags(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		wantName string
		wantNode string
	}{
		{name: "ASCII case fold", content: `<DiV>x</dIv><span>tail</span>`, wantName: "div", wantNode: `<DiV>x</dIv>`},
		{name: "bang punctuation", content: `<a!>x</a!><span>tail</span>`, wantName: "a!", wantNode: `<a!>x</a!>`},
		{name: "at punctuation", content: `<a@>x</a@><span>tail</span>`, wantName: "a@", wantNode: `<a@>x</a@>`},
		{name: "Unicode", content: `<aé>x</aé><span>tail</span>`, wantName: "aé", wantNode: `<aé>x</aé>`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].Name != "span" {
				t.Fatalf("Expected recovered element followed by span, got %#v", parsedBodyChildren(document))
			}
			element := parsedBodyChildren(document)[0]
			if element.Name != testCase.wantName || element.TextContent != "x" {
				t.Fatalf("Expected <%s> with text x, got %#v", testCase.wantName, element)
			}
			if source := testCase.content[element.StartPos:element.EndPos]; source != testCase.wantNode {
				t.Fatalf("Expected exact original node source %q, got %q", testCase.wantNode, source)
			}
		})
	}
}

func TestParseDocumentLevelIncompleteEndTagsUseBrowserTextRanges(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		wantValue string
	}{
		{name: "missing end tag name coalesces surrounding text", content: `before</>after`, wantValue: "beforeafter"},
		{name: "incomplete named end tag is ignored", content: `before</div`, wantValue: "before"},
		{name: "incomplete void closer is ignored without emitting br", content: `before</br`, wantValue: "before"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Type != types.TextNode {
				t.Fatalf("Expected exactly one browser text node, got %#v", parsedBodyChildren(document))
			}
			text := parsedBodyChildren(document)[0]
			if text.Value != testCase.wantValue {
				t.Fatalf("Expected text value %q, got %q", testCase.wantValue, text.Value)
			}
			if text.StartPos != 0 || text.EndPos != len(testCase.content) {
				t.Fatalf("Expected full original range 0:%d, got %d:%d", len(testCase.content), text.StartPos, text.EndPos)
			}
			if source := testCase.content[text.StartPos:text.EndPos]; source != testCase.content {
				t.Fatalf("Expected original range to retain ignored token, got %q", source)
			}
		})
	}
}

func TestParseIncompleteEndTagAtEOFClosesAllOpenAncestorsLikeBrowser(t *testing.T) {
	const content = `<div><span>x</div`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
		t.Fatalf("Expected one div root, got %#v", parsedBodyChildren(document))
	}

	div := parsedBodyChildren(document)[0]
	if div.StartPos != 0 || div.EndPos != len(content) || div.TextContent != "x" {
		t.Fatalf("Expected div to close at EOF over full range, got %#v", div)
	}
	if len(div.Children) != 1 || div.Children[0].Name != "span" {
		t.Fatalf("Expected span inside div, got %#v", div.Children)
	}
	span := div.Children[0]
	if span.StartPos != len(`<div>`) || span.EndPos != len(content) || span.TextContent != "x" {
		t.Fatalf("Expected span to close at EOF through ignored token, got %#v", span)
	}
	if len(span.Children) != 1 || span.Children[0].Type != types.TextNode {
		t.Fatalf("Expected one span text child, got %#v", span.Children)
	}
	text := span.Children[0]
	wantTextStart := len(`<div><span>`)
	if text.Value != "x" || text.StartPos != wantTextStart || text.EndPos != len(content) {
		t.Fatalf("Expected x text range %d:%d through ignored token, got %#v", wantTextStart, len(content), text)
	}
	if source := content[text.StartPos:text.EndPos]; source != `x</div` {
		t.Fatalf("Expected text source to retain ignored token, got %q", source)
	}
}

func TestParseExactEndTagOpenerAtEOFBecomesTextInsideOpenElement(t *testing.T) {
	const content = `<div>before</`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
		t.Fatalf("Expected one div, got %#v", parsedBodyChildren(document))
	}

	div := parsedBodyChildren(document)[0]
	if div.TextContent != `before</` || div.StartPos != 0 || div.EndPos != len(content) {
		t.Fatalf("Expected div text and range through EOF, got %#v", div)
	}
	if len(div.Children) != 1 || div.Children[0].Type != types.TextNode {
		t.Fatalf("Expected one text child, got %#v", div.Children)
	}
	text := div.Children[0]
	if text.Value != `before</` || text.StartPos != len(`<div>`) || text.EndPos != len(content) {
		t.Fatalf("Expected literal end-tag opener in text with exact range, got %#v", text)
	}
	if source := content[text.StartPos:text.EndPos]; source != `before</` {
		t.Fatalf("Expected exact original text source, got %q", source)
	}
}

func TestParseFinalEndTagEOFEdgesLikeBrowser(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		wantValue string
	}{
		{
			name:      "ignored missing name retains following whitespace",
			content:   `before</> after`,
			wantValue: "before after",
		},
		{
			name:      "unterminated quoted end-tag attribute is discarded",
			content:   `before</div foo='>'`,
			wantValue: "before",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Type != types.TextNode {
				t.Fatalf("Expected exactly one browser text node, got %#v", parsedBodyChildren(document))
			}

			text := parsedBodyChildren(document)[0]
			if text.Value != testCase.wantValue {
				t.Fatalf("Expected text value %q, got %q", testCase.wantValue, text.Value)
			}
			if text.StartPos != 0 || text.EndPos != len(testCase.content) {
				t.Fatalf("Expected full original range 0:%d, got %d:%d", len(testCase.content), text.StartPos, text.EndPos)
			}
			if source := testCase.content[text.StartPos:text.EndPos]; source != testCase.content {
				t.Fatalf("Expected range to retain the discarded token, got %q", source)
			}
		})
	}
}

func TestParseBogusEndTagAtEOFClosesOpenElementsLikeBrowser(t *testing.T) {
	t.Run("numeric bogus comment inside div", func(t *testing.T) {
		const content = `<div>before</42`

		parser := NewHTMLParser()
		document, err := parser.Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
			t.Fatalf("Expected one div root, got %#v", parsedBodyChildren(document))
		}

		div := parsedBodyChildren(document)[0]
		if div.TextContent != "before" || div.StartPos != 0 || div.EndPos != len(content) {
			t.Fatalf("Expected div to close at EOF over the full source, got %#v", div)
		}
		if len(div.Children) != 2 || div.Children[0].Type != types.TextNode || div.Children[1].Type != types.CommentNode {
			t.Fatalf("Expected text followed by bogus comment, got %#v", div.Children)
		}

		text := div.Children[0]
		if text.Value != "before" || text.StartPos != len(`<div>`) || text.EndPos != len(`<div>before`) {
			t.Fatalf("Expected exact preceding text range, got %#v", text)
		}
		comment := div.Children[1]
		wantCommentStart := len(`<div>before`)
		if comment.Value != "42" || comment.StartPos != wantCommentStart || comment.EndPos != len(content) {
			t.Fatalf("Expected EOF bogus comment range %d:%d, got %#v", wantCommentStart, len(content), comment)
		}
		if source := content[comment.StartPos:comment.EndPos]; source != `</42` {
			t.Fatalf("Expected exact original bogus-comment source, got %q", source)
		}
	})

	t.Run("nested bogus comment closes all ancestors", func(t *testing.T) {
		const content = `<div><span>x</$foo`

		parser := NewHTMLParser()
		document, err := parser.Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
			t.Fatalf("Expected one div root, got %#v", parsedBodyChildren(document))
		}

		div := parsedBodyChildren(document)[0]
		if div.StartPos != 0 || div.EndPos != len(content) || len(div.Children) != 1 {
			t.Fatalf("Expected div to close at EOF, got %#v", div)
		}
		span := div.Children[0]
		if span.Name != "span" || span.StartPos != len(`<div>`) || span.EndPos != len(content) {
			t.Fatalf("Expected nested span to close at EOF, got %#v", span)
		}
		if len(span.Children) != 2 || span.Children[0].Type != types.TextNode || span.Children[1].Type != types.CommentNode {
			t.Fatalf("Expected x followed by nested bogus comment, got %#v", span.Children)
		}
		text := span.Children[0]
		if text.Value != "x" || text.StartPos != len(`<div><span>`) || text.EndPos != len(`<div><span>x`) {
			t.Fatalf("Expected exact nested text range, got %#v", text)
		}
		comment := span.Children[1]
		wantCommentStart := len(`<div><span>x`)
		if comment.Value != "$foo" || comment.StartPos != wantCommentStart || comment.EndPos != len(content) {
			t.Fatalf("Expected nested EOF comment range %d:%d, got %#v", wantCommentStart, len(content), comment)
		}
		if source := content[comment.StartPos:comment.EndPos]; source != `</$foo` {
			t.Fatalf("Expected exact nested bogus-comment source, got %q", source)
		}
	})
}
