package utils

import (
	"reflect"
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseRecoversMalformedStartTagAttributesLikeBrowser(t *testing.T) {
	testCases := []struct {
		name           string
		content        string
		wantName       string
		wantAttributes map[string]string
		wantOrder      []string
		wantText       string
		wantSource     string
	}{
		{
			name:           "unexpected solidus before attribute",
			content:        `<div/ id=target>payload</div>`,
			wantName:       "div",
			wantAttributes: map[string]string{"id": "target"},
			wantOrder:      []string{"id"},
			wantText:       "payload",
			wantSource:     `<div/ id=target>payload</div>`,
		},
		{
			name:           "unexpected solidus starts attribute recovery",
			content:        `<div /foo=bar>payload</div>`,
			wantName:       "div",
			wantAttributes: map[string]string{"foo": "bar"},
			wantOrder:      []string{"foo"},
			wantText:       "payload",
			wantSource:     `<div /foo=bar>payload</div>`,
		},
		{
			name:           "adjacent quoted attributes",
			content:        `<div id="target"class='primary'data-x=ok>payload</div>`,
			wantName:       "div",
			wantAttributes: map[string]string{"id": "target", "class": "primary", "data-x": "ok"},
			wantOrder:      []string{"id", "class", "data-x"},
			wantText:       "payload",
			wantSource:     `<div id="target"class='primary'data-x=ok>payload</div>`,
		},
		{
			name:           "equals before attribute name",
			content:        `<div =foo bar=baz>payload</div>`,
			wantName:       "div",
			wantAttributes: map[string]string{"=foo": "", "bar": "baz"},
			wantOrder:      []string{"=foo", "bar"},
			wantText:       "payload",
			wantSource:     `<div =foo bar=baz>payload</div>`,
		},
		{
			name:           "less than in attribute name",
			content:        `<span <="">broken</span>`,
			wantName:       "span",
			wantAttributes: map[string]string{"<": ""},
			wantOrder:      []string{"<"},
			wantText:       "broken",
			wantSource:     `<span <="">broken</span>`,
		},
		{
			name:           "invalid characters remain in unquoted value",
			content:        "<div data-x=a\"b'c<d=e`f>payload</div>",
			wantName:       "div",
			wantAttributes: map[string]string{"data-x": "a\"b'c<d=e`f"},
			wantOrder:      []string{"data-x"},
			wantText:       "payload",
			wantSource:     "<div data-x=a\"b'c<d=e`f>payload</div>",
		},
		{
			name:           "missing attribute value becomes empty",
			content:        `<div id=>payload</div>`,
			wantName:       "div",
			wantAttributes: map[string]string{"id": ""},
			wantOrder:      []string{"id"},
			wantText:       "payload",
			wantSource:     `<div id=>payload</div>`,
		},
		{
			name:           "duplicate attributes keep first case insensitive occurrence",
			content:        `<div id=first ID=second class=a CLASS=b>payload</div>`,
			wantName:       "div",
			wantAttributes: map[string]string{"id": "first", "class": "a"},
			wantOrder:      []string{"id", "class"},
			wantText:       "payload",
			wantSource:     `<div id=first ID=second class=a CLASS=b>payload</div>`,
		},
		{
			name:           "CRLF and CR normalize in quoted attribute",
			content:        "<div data-x='a\r\nb\rc'>payload</div>",
			wantName:       "div",
			wantAttributes: map[string]string{"data-x": "a\nb\nc"},
			wantOrder:      []string{"data-x"},
			wantText:       "payload",
			wantSource:     "<div data-x='a\r\nb\rc'>payload</div>",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 {
				t.Fatalf("Expected one element, got %#v", parsedBodyChildren(document))
			}

			element := parsedBodyChildren(document)[0]
			if element.Type != types.ElementNode || element.Name != testCase.wantName {
				t.Fatalf("Expected <%s>, got %#v", testCase.wantName, element)
			}
			if !reflect.DeepEqual(element.Attributes, testCase.wantAttributes) {
				t.Fatalf("Expected attributes %#v, got %#v", testCase.wantAttributes, element.Attributes)
			}
			if !reflect.DeepEqual(element.AttributeOrder, testCase.wantOrder) {
				t.Fatalf("Expected attribute order %#v, got %#v", testCase.wantOrder, element.AttributeOrder)
			}
			if element.TextContent != testCase.wantText {
				t.Fatalf("Expected text %q, got %q", testCase.wantText, element.TextContent)
			}
			if source := testCase.content[element.StartPos:element.EndPos]; source != testCase.wantSource {
				t.Fatalf("Expected exact original source %q, got %q", testCase.wantSource, source)
			}
		})
	}
}

func TestParseRecoversAdjacentAttributesAfterLongPrefixAtReportedPosition(t *testing.T) {
	const malformedAttributePosition = 1446
	const openingPrefix = `<div data-prefix="`
	const adjacentAttributes = `id="target"class="primary">payload</div>`

	paddingLength := malformedAttributePosition - len(openingPrefix) - 1 // closing quote
	if paddingLength < 0 {
		t.Fatalf("test setup: prefix is longer than reported position")
	}
	content := openingPrefix + strings.Repeat("p", paddingLength) + `"` + adjacentAttributes
	if got := strings.Index(content, `id="target"class`); got != malformedAttributePosition {
		t.Fatalf("test setup: adjacent attribute starts at %d, want %d", got, malformedAttributePosition)
	}

	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("browser-compatible adjacent-attribute recovery failed: %v", err)
	}
	elements := parsedBodyChildren(document)
	if len(elements) != 1 || elements[0].Name != "div" {
		t.Fatalf("expected one recovered div, got %#v", elements)
	}
	div := elements[0]
	if div.Attributes["id"] != "target" || div.Attributes["class"] != "primary" || div.TextContent != "payload" {
		t.Fatalf("recovered long-prefix attributes/text mismatch: %#v", div)
	}
	if got := div.Attributes["data-prefix"]; got != strings.Repeat("p", paddingLength) {
		t.Fatalf("long-prefix value mismatch: got length %d, want %d", len(got), paddingLength)
	}
	if !reflect.DeepEqual(div.AttributeOrder, []string{"data-prefix", "id", "class"}) {
		t.Fatalf("recovered attribute order mismatch: %#v", div.AttributeOrder)
	}
	if source := content[div.StartPos:div.EndPos]; source != content {
		t.Fatalf("recovery must preserve the original node byte range, got %d:%d", div.StartPos, div.EndPos)
	}
}

func TestParseTreatsWhitespaceAfterLessThanAsTextLikeBrowser(t *testing.T) {
	const content = `<section>before< div id=bad>after<span id=target>tail</span></section>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "section" {
		t.Fatalf("Expected one section, got %#v", parsedBodyChildren(document))
	}

	section := parsedBodyChildren(document)[0]
	if section.TextContent != "before< div id=bad>aftertail" {
		t.Fatalf("Expected less-than sequence as text, got %q", section.TextContent)
	}
	if len(section.Children) != 2 || section.Children[0].Type != types.TextNode || section.Children[1].Name != "span" {
		t.Fatalf("Expected recovered text followed by span, got %#v", section.Children)
	}
	text := section.Children[0]
	if text.Value != "before< div id=bad>after" {
		t.Fatalf("Expected recovered text value, got %q", text.Value)
	}
	if source := content[text.StartPos:text.EndPos]; source != text.Value {
		t.Fatalf("Expected text to reference original bytes, got %q", source)
	}
	span := section.Children[1]
	if source := content[span.StartPos:span.EndPos]; source != `<span id=target>tail</span>` {
		t.Fatalf("Expected exact original span source, got %q", source)
	}
}

func TestParseIgnoresStartTagAtEOFLikeBrowser(t *testing.T) {
	testCases := []string{
		"before<div",
		"before<div id=",
		"before<div id='target",
	}

	for _, content := range testCases {
		t.Run(content, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Type != types.TextNode {
				t.Fatalf("Expected one text node, got %#v", parsedBodyChildren(document))
			}
			text := parsedBodyChildren(document)[0]
			if text.Value != "before" {
				t.Fatalf("Expected emitted text %q, got %q", "before", text.Value)
			}
			// parse5/jsdom associates the ignored incomplete tag's bytes with the
			// preceding text node, so the location spans the entire original input.
			if text.StartPos != 0 || text.EndPos != len(content) {
				t.Fatalf("Expected browser location 0:%d, got %d:%d", len(content), text.StartPos, text.EndPos)
			}
		})
	}
}

func TestParseIgnoresNonVoidSelfClosingFlagLikeBrowser(t *testing.T) {
	testCases := []struct {
		name           string
		content        string
		wantAttributes map[string]string
		wantStartTag   string
	}{
		{
			name:           "flag immediately after tag name",
			content:        `<div/>x</div><span>tail</span>`,
			wantAttributes: map[string]string{},
			wantStartTag:   `<div/>`,
		},
		{
			name:           "flag after attribute and whitespace",
			content:        `<div a=b />x</div><span>tail</span>`,
			wantAttributes: map[string]string{"a": "b"},
			wantStartTag:   `<div a=b />`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0].Name != "div" || parsedBodyChildren(document)[1].Name != "span" {
				t.Fatalf("Expected div followed by span, got %#v", parsedBodyChildren(document))
			}

			div := parsedBodyChildren(document)[0]
			if div.TextContent != "x" || !reflect.DeepEqual(div.Attributes, testCase.wantAttributes) {
				t.Fatalf("Expected non-void div with text and attributes %#v, got %#v", testCase.wantAttributes, div)
			}
			if source := testCase.content[div.StartPos:div.EndPos]; source != testCase.wantStartTag+`x</div>` {
				t.Fatalf("Expected div range through its real closing tag, got %q", source)
			}
			if source := testCase.content[div.ContentStart:div.ContentEnd]; source != "x" {
				t.Fatalf("Expected div content range %q, got %q", "x", source)
			}
			span := parsedBodyChildren(document)[1]
			if source := testCase.content[span.StartPos:span.EndPos]; source != `<span>tail</span>` {
				t.Fatalf("Expected exact following span source, got %q", source)
			}
		})
	}
}

func TestParseKeepsSolidusInUnquotedAttributeValueLikeBrowser(t *testing.T) {
	const content = `<div a=b/>x</div><span>tail</span>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0].Name != "div" || parsedBodyChildren(document)[1].Name != "span" {
		t.Fatalf("Expected div followed by span, got %#v", parsedBodyChildren(document))
	}
	div := parsedBodyChildren(document)[0]
	if div.Attributes["a"] != "b/" || div.TextContent != "x" {
		t.Fatalf("Expected slash in value and closing div to be consumed, got %#v", div)
	}
	if source := content[div.StartPos:div.EndPos]; source != `<div a=b/>x</div>` {
		t.Fatalf("Expected original div source through closing tag, got %q", source)
	}
}

func TestParseFoldsOnlyASCIIAttributeNamesForDuplicates(t *testing.T) {
	const content = `<div ID=first id=second É=one é=two>x</div>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
		t.Fatalf("Expected one div, got %#v", parsedBodyChildren(document))
	}

	div := parsedBodyChildren(document)[0]
	wantAttributes := map[string]string{"id": "first", "É": "one", "é": "two"}
	wantOrder := []string{"id", "É", "é"}
	if !reflect.DeepEqual(div.Attributes, wantAttributes) {
		t.Fatalf("Expected ASCII-only duplicate folding %#v, got %#v", wantAttributes, div.Attributes)
	}
	if !reflect.DeepEqual(div.AttributeOrder, wantOrder) {
		t.Fatalf("Expected attribute order %#v, got %#v", wantOrder, div.AttributeOrder)
	}
	if source := content[div.StartPos:div.EndPos]; source != content {
		t.Fatalf("Expected exact original source, got %q", source)
	}
}

func TestParseUsesSameBrowserTagNameRulesForStartAndEndTags(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		wantName string
		wantNode string
	}{
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
			span := parsedBodyChildren(document)[1]
			if source := testCase.content[span.StartPos:span.EndPos]; source != `<span>tail</span>` {
				t.Fatalf("Expected exact following span source, got %q", source)
			}
		})
	}
}
