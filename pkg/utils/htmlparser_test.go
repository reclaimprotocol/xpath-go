package utils

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseRejectsBinaryInput(t *testing.T) {
	testCases := map[string]string{
		"NUL byte":       "binary\x00input",
		"control bytes":  "\x01\x02text",
		"invalid UTF-8":  "\xff\xfe",
		"gzip signature": "\x1f\x8bnot-a-complete-stream",
	}

	for name, content := range testCases {
		t.Run(name, func(t *testing.T) {
			parser := NewHTMLParser()
			node, err := parser.Parse(content)
			if err == nil {
				t.Fatalf("Expected binary input error, got node %#v", node)
			}
			if !strings.Contains(err.Error(), "binary input") {
				t.Fatalf("Expected descriptive binary input error, got %q", err)
			}
		})
	}
}

func TestParseRejectsGzipInput(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte("<html><body>content</body></html>")); err != nil {
		t.Fatalf("Could not create gzip input: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Could not finish gzip input: %v", err)
	}

	parser := NewHTMLParser()
	node, err := parser.Parse(compressed.String())
	if err == nil {
		t.Fatalf("Expected binary input error, got node %#v", node)
	}
	if !strings.Contains(err.Error(), "binary input") {
		t.Fatalf("Expected descriptive binary input error, got %q", err)
	}
}

func TestParseRejectsMalformedInput(t *testing.T) {
	testCases := map[string]string{
		"invalid nested attributes":           `<div><span <="">broken</span></div>`,
		"malformed sibling":                   `<ul><li>a</li><li <=""></li><li>c</li></ul>`,
		"unterminated element":                `<div>`,
		"mismatched closing tag":              `<div><span></div>`,
		"unexpected nested closing tag":       `<div></span></div>`,
		"orphan closing tag":                  `</orphan>`,
		"unterminated raw text":               `<script>alert(1)`,
		"unterminated comment":                `<!--comment`,
		"unterminated doctype":                `<!DOCTYPE html`,
		"unterminated processing instruction": `<?xml version="1.0"`,
		"missing attribute value":             `<div a=>`,
		"invalid unquoted attribute value":    `<div a==b></div>`,
	}

	for name, content := range testCases {
		t.Run(name, func(t *testing.T) {
			parser := NewHTMLParser()
			node, err := parser.Parse(content)
			if err == nil {
				t.Fatalf("Expected malformed input error, got node %#v", node)
			}
		})
	}
}

func TestParseRecoversBogusCommentsLikeBrowser(t *testing.T) {
	testCases := []struct {
		name        string
		declaration string
		wantValue   string
	}{
		{name: "unknown declaration", declaration: `<!x>`, wantValue: "x"},
		{name: "CDATA in HTML", declaration: `<![CDATA[payload]]>`, wantValue: "[CDATA[payload]]"},
		{name: "entity declaration", declaration: `<!ENTITY example "value">`, wantValue: `ENTITY example "value"`},
		{name: "empty declaration", declaration: `<!>`, wantValue: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := `<div>before` + testCase.declaration + `after<span>tail</span></div>`

			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(document.Children) != 1 {
				t.Fatalf("Expected one root element, got %#v", document.Children)
			}

			div := document.Children[0]
			if div.TextContent != "beforeaftertail" {
				t.Fatalf("Expected comments to be excluded from text content, got %q", div.TextContent)
			}
			if len(div.Children) != 4 || div.Children[1].Type != types.CommentNode {
				t.Fatalf("Expected text, comment, text, and span children, got %#v", div.Children)
			}

			comment := div.Children[1]
			if comment.Value != testCase.wantValue {
				t.Fatalf("Expected comment value %q, got %q", testCase.wantValue, comment.Value)
			}
			if source := content[comment.StartPos:comment.EndPos]; source != testCase.declaration {
				t.Fatalf("Expected original declaration source %q, got %q", testCase.declaration, source)
			}
			if div.Children[3].Name != "span" {
				t.Fatalf("Expected parsing to continue with span, got %q", div.Children[3].Name)
			}
		})
	}
}

func TestParseRecoversBogusCommentAtEOF(t *testing.T) {
	const content = `<!unfinished`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 1 || document.Children[0].Type != types.CommentNode {
		t.Fatalf("Expected one comment node, got %#v", document.Children)
	}
	comment := document.Children[0]
	if comment.Value != "unfinished" || comment.StartPos != 0 || comment.EndPos != len(content) {
		t.Fatalf("Expected EOF-terminated comment with original range, got %#v", comment)
	}
}

func TestParseAllowsAdjacentHTMLAttributes(t *testing.T) {
	const content = `<div id="target"class='primary'disabled data-value="42">payload</div>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 1 {
		t.Fatalf("Expected one root element, got %d", len(document.Children))
	}

	element := document.Children[0]
	wantAttributes := map[string]string{
		"id":         "target",
		"class":      "primary",
		"disabled":   "",
		"data-value": "42",
	}
	for name, wantValue := range wantAttributes {
		if value, ok := element.Attributes[name]; !ok || value != wantValue {
			t.Errorf("Attribute %q: expected %q, got %q (present=%t)", name, wantValue, value, ok)
		}
	}

	if source := content[element.StartPos:element.EndPos]; source != content {
		t.Fatalf("Expected original source %q, got %q", content, source)
	}
}

func TestParseIgnoresMetaClosingTagLikeBrowser(t *testing.T) {
	const content = `<html><head><meta name="description"content="sample"></meta><title>Page</title></head><body><div id="target">payload</div></body></html>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}

	if len(document.Children) != 1 || document.Children[0].Name != "html" {
		t.Fatalf("Expected one html root, got %#v", document.Children)
	}
	html := document.Children[0]
	if len(html.Children) != 2 || html.Children[0].Name != "head" || html.Children[1].Name != "body" {
		t.Fatalf("Expected head and body children, got %#v", html.Children)
	}

	head := html.Children[0]
	if len(head.Children) != 2 || head.Children[0].Name != "meta" || head.Children[1].Name != "title" {
		t.Fatalf("Expected meta and title children, got %#v", head.Children)
	}
	meta := head.Children[0]
	if source := content[meta.StartPos:meta.EndPos]; source != `<meta name="description"content="sample">` {
		t.Fatalf("Expected original meta source, got %q", source)
	}

	target := html.Children[1].Children[0]
	if source := content[target.StartPos:target.EndPos]; source != `<div id="target">payload</div>` {
		t.Fatalf("Expected original target source, got %q", source)
	}
}

func TestParseIgnoresVoidElementClosingTags(t *testing.T) {
	voidElements := []string{
		"area", "base", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr",
	}

	for _, tagName := range voidElements {
		t.Run(tagName, func(t *testing.T) {
			content := `<div>before</` + tagName + `><span>after</span></div>`

			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(document.Children) != 1 || document.Children[0].TextContent != "beforeafter" {
				t.Fatalf("Expected void closer to be ignored, got %#v", document.Children)
			}

			span := document.Children[0].Children[1]
			if span.Name != "span" {
				t.Fatalf("Expected span after ignored closer, got %q", span.Name)
			}
			if source := content[span.StartPos:span.EndPos]; source != `<span>after</span>` {
				t.Fatalf("Expected original span source, got %q", source)
			}
		})
	}
}

func TestParseTreatsBrClosingTagAsStartTagLikeBrowser(t *testing.T) {
	const content = `<div>before</br><span>after</span></div>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 1 {
		t.Fatalf("Expected one root element, got %#v", document.Children)
	}

	div := document.Children[0]
	if len(div.Children) != 3 || div.Children[1].Name != "br" || div.Children[2].Name != "span" {
		t.Fatalf("Expected text, recovered br, and span children, got %#v", div.Children)
	}
	br := div.Children[1]
	if source := content[br.StartPos:br.EndPos]; source != `</br>` {
		t.Fatalf("Expected recovered br location to reference original token, got %q", source)
	}
	if br.ContentStart != br.EndPos || br.ContentEnd != br.EndPos {
		t.Fatalf("Expected recovered br to have empty content at %d, got %d:%d", br.EndPos, br.ContentStart, br.ContentEnd)
	}
}

func TestParseImplicitlyClosesTableRows(t *testing.T) {
	testCases := []struct {
		name           string
		content        string
		wantRowTexts   []string
		wantRowSources []string
	}{
		{
			name:           "new row closes current row",
			content:        `<table><tr><td>first</td></tr><tr><tr><td>third</td></tr></table>`,
			wantRowTexts:   []string{"first", "", "third"},
			wantRowSources: []string{`<tr><td>first</td></tr>`, `<tr>`, `<tr><td>third</td></tr>`},
		},
		{
			name:           "table end closes current row",
			content:        `<table><tr><td>only</td></table>`,
			wantRowTexts:   []string{"only"},
			wantRowSources: []string{`<tr><td>only</td>`},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}

			if len(document.Children) != 1 || document.Children[0].Name != "table" {
				t.Fatalf("Expected one table root, got %#v", document.Children)
			}

			table := document.Children[0]
			if len(table.Children) != len(testCase.wantRowTexts) {
				t.Fatalf("Expected %d rows, got %d", len(testCase.wantRowTexts), len(table.Children))
			}
			for index, wantText := range testCase.wantRowTexts {
				row := table.Children[index]
				if row.Name != "tr" {
					t.Fatalf("Child %d: expected tr, got %q", index, row.Name)
				}
				if row.TextContent != wantText {
					t.Fatalf("Row %d: expected text %q, got %q", index, wantText, row.TextContent)
				}
				if row.StartPos < 0 || row.EndPos < row.StartPos || row.EndPos > len(testCase.content) {
					t.Fatalf("Row %d: invalid original-source range %d:%d", index, row.StartPos, row.EndPos)
				}
				if source := testCase.content[row.StartPos:row.EndPos]; source != testCase.wantRowSources[index] {
					t.Fatalf("Row %d: expected original source %q, got %q", index, testCase.wantRowSources[index], source)
				}
			}
		})
	}
}
