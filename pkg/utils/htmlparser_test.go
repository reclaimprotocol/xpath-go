package utils

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
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
		"invalid declaration":                 `<div><!x></div>`,
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
