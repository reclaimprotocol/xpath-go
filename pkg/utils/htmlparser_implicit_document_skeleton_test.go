package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 22B extends document-mode tree construction to every whole-document
// parse. Fragment parsing and context-specific fragment roots remain deferred.

func requireImplicitDocumentSkeleton(t *testing.T, document *types.Node) (html, head, body *types.Node) {
	t.Helper()
	html, head, body = requireDirectSkeleton(t, document)
	requireSyntheticDocumentElement(t, html, "html", document)
	return html, head, body
}

func parsedBodyChildren(document *types.Node) []*types.Node {
	body := parsedBody(document)
	if body == nil {
		return nil
	}
	return body.Children
}

func parsedBody(document *types.Node) *types.Node {
	bodies := listElements(document, "body")
	if len(bodies) != 1 {
		return nil
	}
	return bodies[0]
}

func TestParseImplicitDocumentSkeletonForEmptyPreambleAndBodyContent(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		document, err := NewHTMLParser().Parse("")
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		if len(document.Children) != 1 || document.Children[0] != html || len(head.Children) != 0 || len(body.Children) != 0 {
			t.Fatalf("Empty document skeleton mismatch: %#v", document.Children)
		}
	})

	t.Run("comment preamble", func(t *testing.T) {
		const content = `<!--c-->`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		if len(document.Children) != 2 || document.Children[0].Type != types.CommentNode || document.Children[0].Value != "c" || document.Children[0].StartPos != 0 || document.Children[0].EndPos != len(content) || document.Children[0].Parent != document || document.Children[1] != html {
			t.Fatalf("Comment preamble/document order mismatch: %#v", document.Children)
		}
	})

	t.Run("doctype preamble", func(t *testing.T) {
		const content = `<!doctype html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		if len(document.Children) != 2 || document.Children[0].Type != types.DocumentTypeNode || document.Children[0].Name != "html" || document.Children[0].StartPos != 0 || document.Children[0].EndPos != len(content) || document.Children[0].Parent != document || document.Children[1] != html {
			t.Fatalf("Doctype preamble/document order mismatch: %#v", document.Children)
		}
	})

	t.Run("text commits body", func(t *testing.T) {
		document, err := NewHTMLParser().Parse("x")
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		if len(body.Children) != 1 || body.Children[0].Type != types.TextNode || body.Children[0].Value != "x" || body.Children[0].StartPos != 0 || body.Children[0].EndPos != 1 || body.TextContent != "x" || html.TextContent != "x" {
			t.Fatalf("Implicit body text mismatch: %#v", body)
		}
	})

	t.Run("whitespace only stays outside the tree", func(t *testing.T) {
		const content = " \t&#32;"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		if len(body.Children) != 0 || body.TextContent != "" {
			t.Fatalf("Before-html whitespace must not enter the implicit body: %#v", body)
		}
	})

	t.Run("decoded nonspace commits body after ignored whitespace", func(t *testing.T) {
		const content = " \t&#32;&#65;x"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		_, _, body := requireImplicitDocumentSkeleton(t, document)
		if len(body.Children) != 1 || body.Children[0].Value != "Ax" || body.Children[0].StartPos != 11 || body.Children[0].EndPos != 13 {
			t.Fatalf("Decoded implicit body commit mismatch: %#v", body.Children)
		}
	})

	t.Run("element commits body", func(t *testing.T) {
		const content = `<div id=d>x</div>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		div := requireOneDocumentSkeletonElement(t, document, "div")
		if div.Parent != body || div.StartPos != 0 || div.EndPos != 17 || div.ContentStart != 10 || div.ContentEnd != 11 || div.TextContent != "x" {
			t.Fatalf("Implicit body element mismatch: %#v", div)
		}
	})
}

func TestParseImplicitDocumentFramesetAndStrayTableStructureStarts(t *testing.T) {
	t.Run("frameset ignores non-whitespace and retains comments", func(t *testing.T) {
		const content = `<frameset>x<frame><!--c--></frameset>tail`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		htmls := listElements(document, "html")
		heads := listElements(document, "head")
		bodies := listElements(document, "body")
		framesets := listElements(document, "frameset")
		frames := listElements(document, "frame")
		if len(htmls) != 1 || len(heads) != 1 || len(bodies) != 0 || len(framesets) != 1 || len(frames) != 1 {
			t.Fatalf("Implicit frameset skeleton mismatch: html=%d head=%d body=%d frameset=%d frame=%d", len(htmls), len(heads), len(bodies), len(framesets), len(frames))
		}
		frameset := framesets[0]
		if frameset.Parent != htmls[0] || frameset.StartPos != 0 || frameset.EndPos != 37 || frameset.ContentStart != 10 || frameset.ContentEnd != 26 || frameset.TextContent != "" || frames[0].Parent != frameset || frames[0].StartPos != 11 || frames[0].EndPos != 18 {
			t.Fatalf("Frameset source structure mismatch: frameset=%#v frame=%#v", frameset, frames[0])
		}
		if len(frameset.Children) != 2 || frameset.Children[1].Type != types.CommentNode || frameset.Children[1].Value != "c" || frameset.Children[1].StartPos != 18 || frameset.Children[1].EndPos != 26 {
			t.Fatalf("Frameset comment placement mismatch: %#v", frameset.Children)
		}
	})

	for _, testCase := range []struct {
		name, content string
		textStart     int
		paragraph     bool
	}{
		{name: "cell", content: `<td>x`, textStart: 4},
		{name: "row and cell", content: `<tr><td>x`, textStart: 8},
		{name: "section row and cell", content: `<tbody><tr><td>x`, textStart: 15},
		{name: "caption", content: `<caption>x`, textStart: 9},
		{name: "col then paragraph", content: `<col><p>x`, textStart: 8, paragraph: true},
	} {
		t.Run("stray "+testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			_, _, body := requireImplicitDocumentSkeleton(t, document)
			for _, ignored := range []string{"table", "tbody", "tr", "td", "caption", "col"} {
				if elements := listElements(document, ignored); len(elements) != 0 {
					t.Fatalf("Stray <%s> emitted an element for %q: %#v", ignored, testCase.content, elements)
				}
			}
			if body.TextContent != "x" {
				t.Fatalf("Stray table structure lost body text for %q: %#v", testCase.content, body)
			}
			texts := findAllTextNodes(body)
			if len(texts) != 1 || texts[0].Value != "x" || texts[0].StartPos != testCase.textStart || texts[0].EndPos != len(testCase.content) {
				t.Fatalf("Stray table structure text range mismatch for %q: %#v", testCase.content, texts)
			}
			paragraphs := listElements(document, "p")
			if testCase.paragraph {
				if len(paragraphs) != 1 || paragraphs[0].Parent != body || paragraphs[0].StartPos != 5 || paragraphs[0].EndPos != len(testCase.content) {
					t.Fatalf("Paragraph after stray col mismatch: %#v", paragraphs)
				}
			} else if len(paragraphs) != 0 {
				t.Fatalf("Unexpected paragraph for %q: %#v", testCase.content, paragraphs)
			}
		})
	}
}

func TestParseImplicitDocumentSkeletonForSourceTitleAndPartialHead(t *testing.T) {
	t.Run("title creates implicit head", func(t *testing.T) {
		const content = `<title>x</title>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		title := requireOneDocumentSkeletonElement(t, document, "title")
		if title.Parent != head || title.StartPos != 0 || title.EndPos != 16 || title.ContentStart != 7 || title.ContentEnd != 8 || title.TextContent != "x" {
			t.Fatalf("Implicit title/head mismatch: %#v", title)
		}
	})

	t.Run("closed source head then body element", func(t *testing.T) {
		const content = `<head></head><div>x</div>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, body, "body", html)
		div := requireOneDocumentSkeletonElement(t, document, "div")
		if head.StartPos != 0 || head.EndPos != 13 || div.Parent != body || div.StartPos != 13 || div.EndPos != 25 || div.TextContent != "x" {
			t.Fatalf("Partial head/body transition mismatch: head=%#v div=%#v", head, div)
		}
	})
}

func TestParseImplicitDocumentSkeletonRetainsSourceHeadAndBody(t *testing.T) {
	t.Run("source head", func(t *testing.T) {
		const content = `<head id=h><title id=t>x</title></head>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, body, "body", html)
		title := requireOneDocumentSkeletonElement(t, document, "title")
		if head.Attributes["id"] != "h" || head.StartPos != 0 || head.EndPos != 39 || head.ContentStart != 11 || head.ContentEnd != 32 || title.Parent != head || title.StartPos != 11 || title.EndPos != 32 || title.TextContent != "x" {
			t.Fatalf("Source head/ranges mismatch: head=%#v title=%#v", head, title)
		}
	})

	t.Run("source body", func(t *testing.T) {
		const content = `<body id=b><div id=d>x</div></body>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		div := requireOneDocumentSkeletonElement(t, document, "div")
		if body.Attributes["id"] != "b" || body.StartPos != 0 || body.EndPos != 35 || body.ContentStart != 11 || body.ContentEnd != 28 || div.Parent != body || div.StartPos != 11 || div.EndPos != 28 || div.TextContent != "x" {
			t.Fatalf("Source body/ranges mismatch: body=%#v div=%#v", body, div)
		}
	})
}

func TestParseImplicitDocumentRoutesHeadOnlyTokensBeforeBody(t *testing.T) {
	const content = `<title id=t>x</title><meta id=m><style id=s>y</style><script id=c>z</script><base id=a href=x><link id=l rel=x><div id=d>q</div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	html, head, body := requireImplicitDocumentSkeleton(t, document)
	requireSyntheticDocumentElement(t, head, "head", html)
	requireSyntheticDocumentElement(t, body, "body", html)
	wantHead := []string{"title", "meta", "style", "script", "base", "link"}
	if len(head.Children) != len(wantHead) {
		t.Fatalf("Implicit head children mismatch: %#v", head.Children)
	}
	for index, name := range wantHead {
		if head.Children[index].Name != name || head.Children[index].Parent != head {
			t.Fatalf("Implicit head child %d mismatch: %#v", index, head.Children[index])
		}
	}
	if head.Children[0].StartPos != 0 || head.Children[0].EndPos != 21 || head.Children[5].StartPos != 94 || head.Children[5].EndPos != 111 || len(body.Children) != 1 || body.Children[0].Name != "div" || body.Children[0].StartPos != 111 || body.Children[0].EndPos != 128 || body.TextContent != "q" {
		t.Fatalf("Head-only routing/source ranges mismatch: head=%#v body=%#v", head, body)
	}
}

func TestParseImplicitDocumentCommentsAndPIsFollowInsertionPhases(t *testing.T) {
	const content = `<?pre?><!--pre--><title id=t>x</title><?head?><!--head--><div id=d>y</div><?body?><!--body-->`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	html, head, body := requireImplicitDocumentSkeleton(t, document)
	requireSyntheticDocumentElement(t, head, "head", html)
	requireSyntheticDocumentElement(t, body, "body", html)
	pis := processingInstructions(document)
	comments := processingComments(document)
	if len(pis) != 3 || len(comments) != 3 {
		t.Fatalf("Implicit phase PI/comment counts mismatch: pis=%#v comments=%#v", pis, comments)
	}
	requireProcessingInstruction(t, pis[0], "pre", "", 0, 7)
	requireProcessingInstruction(t, pis[1], "head", "", 38, 46)
	requireProcessingInstruction(t, pis[2], "body", "", 74, 82)
	if pis[0].Parent != document || comments[0].Value != "pre" || comments[0].StartPos != 7 || comments[0].EndPos != 17 || comments[0].Parent != document || pis[1].Parent != head || comments[1].Value != "head" || comments[1].StartPos != 46 || comments[1].EndPos != 57 || comments[1].Parent != head || pis[2].Parent != body || comments[2].Value != "body" || comments[2].StartPos != 82 || comments[2].EndPos != 93 || comments[2].Parent != body {
		t.Fatalf("Implicit phase PI/comment placement mismatch: pis=%#v comments=%#v", pis, comments)
	}
}

func TestParseImplicitDocumentPreambleAndLateHeadTokenPlacement(t *testing.T) {
	t.Run("preamble and body comments", func(t *testing.T) {
		const content = `<!--pre--><div>x</div><!--post-->`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, _, body := requireImplicitDocumentSkeleton(t, document)
		comments := processingComments(document)
		div := requireOneDocumentSkeletonElement(t, document, "div")
		if len(comments) != 2 || comments[0].Value != "pre" || comments[0].Parent != document || comments[0].StartPos != 0 || comments[0].EndPos != 10 || div.Parent != body || div.StartPos != 10 || div.EndPos != 22 || comments[1].Value != "post" || comments[1].Parent != body || comments[1].StartPos != 22 || comments[1].EndPos != 33 || html.TextContent != "x" {
			t.Fatalf("Implicit pre/body comment placement mismatch: comments=%#v div=%#v", comments, div)
		}
	})

	t.Run("late meta stays in body", func(t *testing.T) {
		const content = `<meta id=m><div>x</div><meta id=late>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		_, head, body := requireImplicitDocumentSkeleton(t, document)
		metas := documentSkeletonElements(document, "meta")
		if len(metas) != 2 || metas[0].Attributes["id"] != "m" || metas[0].Parent != head || metas[0].StartPos != 0 || metas[0].EndPos != 11 || metas[1].Attributes["id"] != "late" || metas[1].Parent != body || metas[1].StartPos != 23 || metas[1].EndPos != 37 {
			t.Fatalf("Implicit early/late meta placement mismatch: %#v", metas)
		}
	})

	t.Run("retained head and document phase comments", func(t *testing.T) {
		const content = `<head id=h><title>x</title></head><!--ah--><meta id=m><body id=b>y</body><!--ab--></html><!--aa-->`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		meta := requireOneDocumentSkeletonElement(t, document, "meta")
		comments := processingComments(document)
		if head.StartPos != 0 || head.EndPos != 43 || meta.Parent != head || meta.StartPos != 43 || meta.EndPos != 54 || body.StartPos != 54 || body.EndPos != 73 || body.TextContent != "y" {
			t.Fatalf("Retained implicit-document head/body range mismatch: head=%#v meta=%#v body=%#v", head, meta, body)
		}
		if len(comments) != 3 || comments[0].Value != "ah" || comments[0].Parent != html || comments[0].StartPos != 34 || comments[0].EndPos != 43 || comments[1].Value != "ab" || comments[1].Parent != html || comments[1].StartPos != 73 || comments[1].EndPos != 82 || comments[2].Value != "aa" || comments[2].Parent != document || comments[2].StartPos != 89 || comments[2].EndPos != 98 {
			t.Fatalf("Retained implicit-document comment phases mismatch: %#v", comments)
		}
	})
}

func TestParseImplicitDocumentAfterBodyAndHTMLPlacement(t *testing.T) {
	const content = `<body id=b>x</body><!--ab--></html><!--aa--><div id=d>y</div><!--tail-->`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	html, head, body := requireImplicitDocumentSkeleton(t, document)
	requireSyntheticDocumentElement(t, head, "head", html)
	if body.StartPos != 0 || body.EndPos != 19 || body.ContentStart != 11 || body.ContentEnd != 12 || body.TextContent != "xy" {
		t.Fatalf("Source body post-end range/text mismatch: %#v", body)
	}
	comments := processingComments(document)
	commentsByValue := make(map[string]*types.Node, len(comments))
	for _, comment := range comments {
		commentsByValue[comment.Value] = comment
	}
	if len(comments) != 3 || commentsByValue["ab"] == nil || commentsByValue["ab"].Parent != html || commentsByValue["ab"].StartPos != 19 || commentsByValue["ab"].EndPos != 28 || commentsByValue["aa"] == nil || commentsByValue["aa"].Parent != document || commentsByValue["aa"].StartPos != 35 || commentsByValue["aa"].EndPos != 44 || commentsByValue["tail"] == nil || commentsByValue["tail"].Parent != body || commentsByValue["tail"].StartPos != 61 || commentsByValue["tail"].EndPos != 72 {
		t.Fatalf("After-body/html comment placement mismatch: %#v", comments)
	}
	div := requireOneDocumentSkeletonElement(t, document, "div")
	if div.Parent != body || div.StartPos != 44 || div.EndPos != 61 || div.TextContent != "y" {
		t.Fatalf("After-html body reentry mismatch: %#v", div)
	}
}

func TestParseImplicitDocumentEOFClosesSourceHeadAndBodyDescendants(t *testing.T) {
	t.Run("standalone body element", func(t *testing.T) {
		const content = `<div>x`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Standalone implicit-body EOF recovery failed: %v", err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		div := requireOneDocumentSkeletonElement(t, document, "div")
		if div.Parent != body || div.StartPos != 0 || div.EndPos != len(content) || div.ContentStart != 5 || div.ContentEnd != len(content) || div.TextContent != "x" {
			t.Fatalf("Standalone implicit-body EOF ranges mismatch: %#v", div)
		}
	})

	t.Run("head and complete title", func(t *testing.T) {
		const content = `<head><title>x</title>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Implicit head EOF recovery failed: %v", err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, body, "body", html)
		title := requireOneDocumentSkeletonElement(t, document, "title")
		if head.StartPos != 0 || head.EndPos != 14 || head.ContentStart != 6 || head.ContentEnd != 14 || title.StartPos != 6 || title.EndPos != 22 || title.ContentStart != 13 || title.ContentEnd != 14 || title.TextContent != "x" {
			t.Fatalf("Implicit head/title EOF ranges mismatch: head=%#v title=%#v", head, title)
		}
	})

	t.Run("body and open div", func(t *testing.T) {
		const content = `<body><div>x`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Implicit body EOF recovery failed: %v", err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		div := requireOneDocumentSkeletonElement(t, document, "div")
		if body.StartPos != 0 || body.EndPos != 6 || body.ContentStart != 6 || body.ContentEnd != 6 || div.StartPos != 6 || div.EndPos != len(content) || div.ContentStart != 11 || div.ContentEnd != len(content) || div.TextContent != "x" {
			t.Fatalf("Implicit body/div EOF ranges mismatch: body=%#v div=%#v", body, div)
		}
	})
}

func TestParseImplicitDocumentUnicodeCoordinatesAndIncompleteWrappers(t *testing.T) {
	t.Run("plain Unicode body text", func(t *testing.T) {
		const content = "é\r\n😀"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		_, _, body := requireImplicitDocumentSkeleton(t, document)
		if len(body.Children) != 1 || body.Children[0].Value != "é\n😀" || body.Children[0].StartPos != 0 || body.Children[0].EndPos != len(content) || body.Children[0].StartLine != 1 || body.Children[0].StartColumn != 1 || body.Children[0].EndLine != 2 || body.Children[0].EndColumn != 3 {
			t.Fatalf("Implicit Unicode body coordinates mismatch: %#v", body.Children)
		}
	})

	t.Run("Unicode head and title", func(t *testing.T) {
		const content = "<head><title>é\r\n😀</title>"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		_, head, _ := requireImplicitDocumentSkeleton(t, document)
		title := requireOneDocumentSkeletonElement(t, document, "title")
		if head.StartPos != 0 || head.EndPos != 21 || head.EndLine != 2 || head.EndColumn != 3 || title.StartPos != 6 || title.EndPos != 29 || title.ContentStart != 13 || title.ContentEnd != 21 || title.EndLine != 2 || title.EndColumn != 11 || title.TextContent != "é\n😀" {
			t.Fatalf("Implicit Unicode head/title coordinates mismatch: head=%#v title=%#v", head, title)
		}
	})

	t.Run("Unicode body descendant EOF", func(t *testing.T) {
		const content = "<body><div>é\r\n😀"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		_, _, body := requireImplicitDocumentSkeleton(t, document)
		div := requireOneDocumentSkeletonElement(t, document, "div")
		if body.StartPos != 0 || body.EndPos != 6 || div.StartPos != 6 || div.EndPos != len(content) || div.ContentStart != 11 || div.ContentEnd != len(content) || div.EndLine != 2 || div.EndColumn != 3 || div.TextContent != "é\n😀" {
			t.Fatalf("Implicit Unicode body EOF mismatch: body=%#v div=%#v", body, div)
		}
	})

	for _, content := range []string{`<html`, `<head`, `<body`} {
		t.Run(content, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatalf("Incomplete wrapper token must be discarded: %v", err)
			}
			html, head, body := requireImplicitDocumentSkeleton(t, document)
			requireSyntheticDocumentElement(t, html, "html", document)
			requireSyntheticDocumentElement(t, head, "head", html)
			requireSyntheticDocumentElement(t, body, "body", html)
			if len(documentSkeletonElements(document, strings.TrimPrefix(content, "<"))) != 1 {
				t.Fatalf("Discarded wrapper created an extra source node: %#v", document)
			}
		})
	}
}

func TestParseImplicitDocumentMergesDuplicateWrapperStarts(t *testing.T) {
	const content = `x<html lang=z>y<body id=b>z<head id=h>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	html, head, body := requireImplicitDocumentSkeleton(t, document)
	if html.Attributes["lang"] != "z" || html.StartPos != 0 || html.EndPos != 0 || body.Attributes["id"] != "b" || body.StartPos != 0 || body.EndPos != 0 || head.Attributes["id"] != "" || body.TextContent != "xyz" {
		t.Fatalf("Duplicate implicit wrapper merge/ignore mismatch: html=%#v head=%#v body=%#v", html, head, body)
	}
}

func TestParseImplicitDocumentBodyAttachmentIntegrationGuards(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		owner   string
	}{
		{"formatting", `<b>x</b>`, "b"},
		{"list", `<ul><li>x</ul>`, "ul"},
		{"form", `<form>x</form>`, "form"},
		{"select", `<select><option>x</select>`, "select"},
		{"svg", `<svg><circle /></svg>`, "svg"},
		{"math", `<math><mi>x</mi></math>`, "math"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			_, _, body := requireImplicitDocumentSkeleton(t, document)
			owner := requireOneDocumentSkeletonElement(t, document, testCase.owner)
			if owner.Parent != body {
				t.Fatalf("Root <%s> did not attach to implicit body: %#v", testCase.owner, owner.Parent)
			}
		})
	}

	t.Run("table foster", func(t *testing.T) {
		const content = `<table>x<tr><td>y</table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		_, _, body := requireImplicitDocumentSkeleton(t, document)
		table := requireOneDocumentSkeletonElement(t, document, "table")
		if len(body.Children) != 2 || body.Children[0].Type != types.TextNode || body.Children[0].Value != "x" || body.Children[1] != table || table.TextContent != "y" {
			t.Fatalf("Root table foster/body attachment mismatch: %#v", body.Children)
		}
	})
}

func TestParseImplicitDocumentFramesetSkeleton(t *testing.T) {
	t.Run("frameset document has no body", func(t *testing.T) {
		const content = `<frameset><frame src=x></frameset>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html := requireOneDocumentSkeletonElement(t, document, "html")
		requireSyntheticDocumentElement(t, html, "html", document)
		head := requireOneDocumentSkeletonElement(t, document, "head")
		requireSyntheticDocumentElement(t, head, "head", html)
		if bodies := documentSkeletonElements(document, "body"); len(bodies) != 0 {
			t.Fatalf("Frameset document emitted a body: %#v", bodies)
		}
		frameset := requireOneDocumentSkeletonElement(t, document, "frameset")
		frame := requireOneDocumentSkeletonElement(t, document, "frame")
		if frameset.Parent != html || frameset.StartPos != 0 || frameset.EndPos != 34 || frame.Parent != frameset || frame.StartPos != 10 || frame.EndPos != 23 {
			t.Fatalf("Implicit frameset tree/ranges mismatch: frameset=%#v frame=%#v", frameset, frame)
		}
	})

	t.Run("body text disables later frameset", func(t *testing.T) {
		const content = `x<frameset><frame src=x></frameset>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		_, _, body := requireImplicitDocumentSkeleton(t, document)
		if body.TextContent != "x" || len(documentSkeletonElements(document, "frameset")) != 0 || len(documentSkeletonElements(document, "frame")) != 0 {
			t.Fatalf("Body-committed frameset tokens were not ignored: %#v", document)
		}
	})
}

func TestParseImplicitDocumentSkeletonParserReuse(t *testing.T) {
	parser := NewHTMLParser()
	for _, content := range []string{"", `<!--c-->`, `<div>x</div>`, `<head><title>x</title>`, `<body><div>y</div></body>`, `<html><body>z</body></html>`} {
		document, err := parser.Parse(content)
		if err != nil {
			t.Fatalf("Implicit skeleton reuse parse failed for %q: %v", content, err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		if html.Parent != document || head.Parent != html || body.Parent != html || len(documentSkeletonElements(document, "html")) != 1 || len(documentSkeletonElements(document, "head")) != 1 || len(documentSkeletonElements(document, "body")) != 1 {
			t.Fatalf("Implicit skeleton reuse state leaked for %q: %#v", content, document)
		}
	}
}

func TestParseImplicitDocumentSkeletonScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping implicit document skeleton scaling regression in short mode")
	}
	build := func(n int) string {
		return strings.Repeat(`<meta>`, n) + `<div>` + strings.Repeat(`<span>x</span>`, n) + `</div>`
	}
	measure := func(n int) time.Duration {
		content := build(n)
		best := time.Duration(1<<63 - 1)
		for attempt := 0; attempt < 2; attempt++ {
			started := time.Now()
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatalf("Implicit skeleton scaling parse failed: %v", err)
			}
			elapsed := time.Since(started)
			if elapsed < best {
				best = elapsed
			}
			_, head, body := requireDirectSkeleton(t, document)
			if len(documentSkeletonElements(head, "meta")) != n || len(documentSkeletonElements(body, "span")) != n {
				t.Fatalf("Implicit skeleton scaling shape mismatch for %d tokens", n)
			}
		}
		return best
	}
	_ = measure(50)
	small := measure(500)
	large := measure(2000)
	ratio := float64(large) / float64(small)
	t.Logf("implicit skeleton scaling 500=%v 2000=%v ratio=%.1fx", small, large, ratio)
	if ratio > 12 && large-small > 150*time.Millisecond {
		t.Fatalf("Implicit skeleton processing scaled superlinearly: %.1fx", ratio)
	}
}
