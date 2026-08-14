package utils

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseCommentUnicodeNULCRLFCoordinatesForXPath(t *testing.T) {
	const content = "<div>é\r\n<!--a\x00b\r\n😀-->z</div>"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	comments := findAllNodesByType(document, types.CommentNode)
	if len(comments) != 1 {
		t.Fatalf("expected one comment, got %#v", comments)
	}
	comment := comments[0]
	if comment.Value != "a�b\n😀" || comment.TextContent != "a�b\n😀" || comment.StartPos != 9 || comment.EndPos != 25 || comment.StartLine != 2 || comment.StartColumn != 1 || comment.EndLine != 3 || comment.EndColumn != 6 {
		t.Fatalf("comment byte/UTF-16 coordinates mismatch: %#v", comment)
	}
}
