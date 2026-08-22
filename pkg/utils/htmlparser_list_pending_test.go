package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParsePendingListReentryClassifiesDecodedCharacterTokens(t *testing.T) {
	for _, testCase := range []struct {
		name, rawReference, decoded string
		commentStart, commentEnd    int
	}{
		{name: "decimal space", rawReference: `&#32;`, decoded: " ", commentStart: 29, commentEnd: 37},
		{name: "decimal tab", rawReference: `&#9;`, decoded: "\t", commentStart: 28, commentEnd: 36},
		{name: "hex space", rawReference: `&#x20;`, decoded: " ", commentStart: 30, commentEnd: 38},
		{name: "named tab", rawReference: `&Tab;`, decoded: "\t", commentStart: 29, commentEnd: 37},
		{name: "named newline", rawReference: `&NewLine;`, decoded: "\n", commentStart: 33, commentEnd: 41},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			content := `<html><body><li>x</body>` + testCase.rawReference + `<!--c--></html>`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			html := listElements(document, "html")[0]
			body := listElements(document, "body")[0]
			li := listElements(document, "li")[0]
			comments := findAllNodesByType(document, types.CommentNode)
			assertListNode(t, html, "html", "x"+testCase.decoded, 0, len(content))
			assertListNode(t, body, "body", "x"+testCase.decoded, 6, 24)
			assertListNode(t, li, "li", "x"+testCase.decoded, 12, len(content))
			if len(comments) != 1 || comments[0].StartPos != testCase.commentStart || comments[0].EndPos != testCase.commentEnd || comments[0].Parent != html {
				t.Fatalf("Expected decoded whitespace to stay after-body with comment under html, got %#v", comments)
			}
			if len(li.Children) != 1 || li.Children[0].Value != "x"+testCase.decoded || li.Children[0].StartPos != 16 || li.Children[0].EndPos != testCase.commentStart {
				t.Fatalf("Expected decoded whitespace in pending li raw text range 16:%d, got %#v", testCase.commentStart, li.Children)
			}
		})
	}

	t.Run("decoded non-whitespace reenters body", func(t *testing.T) {
		const content = `<html><body><li>x</body>&#65;<!--c--></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		li := listElements(document, "li")[0]
		comments := findAllNodesByType(document, types.CommentNode)
		assertListNode(t, li, "li", "xA", 12, len(content))
		if len(li.Children) != 2 || li.Children[0].Type != types.TextNode || li.Children[0].Value != "xA" || li.Children[0].StartPos != 16 || li.Children[0].EndPos != 29 || li.Children[1] != comments[0] {
			t.Fatalf("Expected numeric A to reenter body and following comment to remain in li, got %#v", li.Children)
		}
		if len(comments) != 1 || comments[0].StartPos != 29 || comments[0].EndPos != 37 || comments[0].Parent != li {
			t.Fatalf("Expected comment under reentered li at 29:37, got %#v", comments)
		}
	})
}

func TestParsePendingListClassifierIgnoresDoctypeAndMergesDuplicateHTML(t *testing.T) {
	for _, testCase := range []struct {
		name, content       string
		htmlEnd, bodyEnd    int
		commentStart, liEnd int
		commentUnderHTML    bool
		wantLang            string
	}{
		{name: "doctype after body", content: `<html><body><li>x</body><!DOCTYPE svg><!--c--></html>`, htmlEnd: 53, bodyEnd: 24, commentStart: 38, liEnd: 53, commentUnderHTML: true},
		{name: "doctype after html", content: `<html><body><li>x</html><!DOCTYPE svg><!--c-->`, htmlEnd: 24, bodyEnd: 17, commentStart: 38, liEnd: 46},
		{name: "duplicate html after body", content: `<html><body><li>x</body><html lang=z><!--c--></html>`, htmlEnd: 52, bodyEnd: 24, commentStart: 37, liEnd: 52, commentUnderHTML: true, wantLang: "z"},
		{name: "duplicate html after html", content: `<html><body><li>x</html><html lang=z><!--c-->`, htmlEnd: 24, bodyEnd: 17, commentStart: 37, liEnd: 45, wantLang: "z"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			html := listElements(document, "html")[0]
			body := listElements(document, "body")[0]
			li := listElements(document, "li")[0]
			comments := findAllNodesByType(document, types.CommentNode)
			assertListNode(t, html, "html", "x", 0, testCase.htmlEnd)
			assertListNode(t, body, "body", "x", 6, testCase.bodyEnd)
			assertListNode(t, li, "li", "x", 12, testCase.liEnd)
			if len(comments) != 1 || comments[0].StartPos != testCase.commentStart || comments[0].EndPos != testCase.commentStart+8 {
				t.Fatalf("Expected comment c at %d:%d, got %#v", testCase.commentStart, testCase.commentStart+8, comments)
			}
			wantParent := document
			if testCase.commentUnderHTML {
				wantParent = html
			}
			if comments[0].Parent != wantParent {
				t.Fatalf("Expected comment parent %#v, got %#v", wantParent, comments[0].Parent)
			}
			if testCase.wantLang != "" && html.Attributes["lang"] != testCase.wantLang {
				t.Fatalf("Expected duplicate html token to merge lang=%q, got %#v", testCase.wantLang, html.Attributes)
			}
			if strings.Contains(testCase.content, "DOCTYPE") && len(findAllNodesByType(document, types.DocumentTypeNode)) != 0 {
				t.Fatalf("Expected post-body/html doctype to be ignored")
			}
		})
	}
}

func TestParsePendingListDecodedPreviewParserReuse(t *testing.T) {
	parser := NewHTMLParser()
	for _, testCase := range []struct {
		content, wantText string
		commentParentHTML bool
	}{
		{content: `<html><body><li>x</body>&#65;<!--c--></html>`, wantText: "xA"},
		{content: `<html><body><li>x</body>&Tab;<!--c--></html>`, wantText: "x\t", commentParentHTML: true},
		{content: `<html><body><li>x</body>&NewLine;<!--c--></html>`, wantText: "x\n", commentParentHTML: true},
	} {
		document, err := parser.Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		li := listElements(document, "li")[0]
		comments := findAllNodesByType(document, types.CommentNode)
		if li.TextContent != testCase.wantText || len(comments) != 1 {
			t.Fatalf("Expected reused parser item text %q and one comment, got li=%#v comments=%#v", testCase.wantText, li, comments)
		}
		if testCase.commentParentHTML != (comments[0].Parent.Name == "html") {
			t.Fatalf("Unexpected comment parent after parser reuse: %#v", comments[0].Parent)
		}
	}
}

func TestParsePendingListWhitespaceCommentScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n*16 + 48)
		b.WriteString(`<html><body><li>x</body>`)
		for i := 0; i < n; i++ {
			b.WriteString(`&#32;<!--c-->`)
		}
		b.WriteString(`<!DOCTYPE svg></html>`)
		return b.String()
	}
	measure := func(n int) time.Duration {
		content := build(n)
		_, _ = NewHTMLParser().Parse(content)
		best := time.Duration(1<<63 - 1)
		for i := 0; i < 3; i++ {
			start := time.Now()
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(findAllNodesByType(document, types.CommentNode)); got != n {
				t.Fatalf("Expected %d after-body comments, got %d", n, got)
			}
			if got := len(findAllNodesByType(document, types.DocumentTypeNode)); got != 0 {
				t.Fatalf("Expected suffix doctype to remain ignored, got %d", got)
			}
			if elapsed := time.Since(start); elapsed < best {
				best = elapsed
			}
		}
		return best
	}
	small, large := measure(250), measure(1000)
	ratio := float64(large) / float64(small)
	t.Logf("post-body whitespace/comment scaling 250=%v 1000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 10*time.Millisecond {
		t.Fatalf("Post-body decoded whitespace/comment classification scaled superlinearly: 250=%v 1000=%v ratio=%.1fx", small, large, ratio)
	}
}
