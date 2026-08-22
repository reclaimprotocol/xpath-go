package xpath_test

import (
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryManyTextStateElementsPreservesLastSourceLocation(t *testing.T) {
	const count = 256
	const token = "<title>x</title>"
	document := strings.Repeat(token, count)
	results, err := xpath.Query(`//title/text()`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != count {
		t.Fatalf("Expected %d title text nodes, got %d", count, len(results))
	}

	last := results[len(results)-1]
	wantStart := (count-1)*len(token) + len("<title>")
	if last.NodeName != "#text" || last.TextContent != "x" || last.StartLocation != wantStart || last.EndLocation != wantStart+1 {
		t.Fatalf("Expected final title text x at %d:%d, got %#v", wantStart, wantStart+1, last)
	}
	if source := document[last.StartLocation:last.EndLocation]; source != "x" {
		t.Fatalf("Expected final result to slice the original x, got %q", source)
	}
}
