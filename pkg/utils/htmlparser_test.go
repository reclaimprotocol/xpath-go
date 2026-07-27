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
		"invalid nested attributes": `<div><span <="">broken</span></div>`,
		"invalid declaration":       `<div><!x></div>`,
		"malformed sibling":         `<ul><li>a</li><li <=""></li><li>c</li></ul>`,
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
