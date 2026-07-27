package xpath_test

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryRejectsGzipInput(t *testing.T) {
	const document = `<html><body><table><tr><td>Student Name</td><td><div>Jane Doe</div></td></tr></table></body></html>`
	const expression = `//td[normalize-space()='Student Name']/following-sibling::td[1]/div/text()`

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte(document)); err != nil {
		t.Fatalf("Could not create gzip input: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Could not finish gzip input: %v", err)
	}

	results, err := xpath.Query(expression, compressed.String())
	if err == nil {
		t.Fatalf("Expected binary input error, got %d results", len(results))
	}
	if !strings.Contains(err.Error(), "binary input") {
		t.Fatalf("Expected descriptive binary input error, got %q", err)
	}
}

func TestQueryRejectsMalformedHTML(t *testing.T) {
	testCases := map[string]string{
		"invalid nested attributes": `<div><span <="">broken</span></div>`,
		"invalid declaration":       `<div><!x></div>`,
		"malformed sibling":         `<ul><li>a</li><li <=""></li><li>c</li></ul>`,
	}

	for name, document := range testCases {
		t.Run(name, func(t *testing.T) {
			results, err := xpath.Query("//*", document)
			if err == nil {
				t.Fatalf("Expected malformed HTML error, got %d results", len(results))
			}
			if !strings.Contains(err.Error(), "HTML parsing failed") {
				t.Fatalf("Expected wrapped HTML parsing error, got %q", err)
			}
		})
	}
}
