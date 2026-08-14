package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParsePreAndListingStripInitialLineFeedLikeBrowser(t *testing.T) {
	testCases := []struct {
		name, tag, raw, wantValue, wantSource string
		wantStart, wantEnd                    int
		wantStartLine, wantStartColumn        int
		wantEndLine, wantEndColumn            int
	}{
		{name: "pre LF", tag: "pre", raw: "\nabc", wantValue: "abc", wantSource: "abc", wantStart: 6, wantEnd: 9, wantStartLine: 2, wantStartColumn: 1, wantEndLine: 2, wantEndColumn: 4},
		{name: "listing LF", tag: "listing", raw: "\nabc", wantValue: "abc", wantSource: "abc", wantStart: 10, wantEnd: 13, wantStartLine: 2, wantStartColumn: 1, wantEndLine: 2, wantEndColumn: 4},
		{name: "CRLF is preprocessed then stripped", tag: "pre", raw: "\r\nabc", wantValue: "abc", wantSource: "abc", wantStart: 7, wantEnd: 10, wantStartLine: 2, wantStartColumn: 1, wantEndLine: 2, wantEndColumn: 4},
		{name: "lone CR is preprocessed then stripped", tag: "pre", raw: "\rabc", wantValue: "abc", wantSource: "abc", wantStart: 6, wantEnd: 9, wantStartLine: 2, wantStartColumn: 1, wantEndLine: 2, wantEndColumn: 4},
		{name: "double LF retains one DOM newline and both source bytes", tag: "pre", raw: "\n\nabc", wantValue: "\nabc", wantSource: "\n\nabc", wantStart: 5, wantEnd: 10, wantStartLine: 1, wantStartColumn: 6, wantEndLine: 3, wantEndColumn: 4},
		{name: "decoded numeric LF is stripped", tag: "pre", raw: "&#10;abc", wantValue: "abc", wantSource: "abc", wantStart: 10, wantEnd: 13, wantStartLine: 1, wantStartColumn: 11, wantEndLine: 1, wantEndColumn: 14},
		{name: "decoded named LF is stripped", tag: "pre", raw: "&NewLine;abc", wantValue: "abc", wantSource: "abc", wantStart: 14, wantEnd: 17, wantStartLine: 1, wantStartColumn: 15, wantEndLine: 1, wantEndColumn: 18},
		{name: "UTF-8 bytes and UTF-16 columns remain distinct", tag: "pre", raw: "\né😀", wantValue: "é😀", wantSource: "é😀", wantStart: 6, wantEnd: 12, wantStartLine: 2, wantStartColumn: 1, wantEndLine: 2, wantEndColumn: 4},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			opening := `<` + testCase.tag + `>`
			content := opening + testCase.raw + `</` + testCase.tag + `>`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			elements := listElements(document, testCase.tag)
			if len(elements) != 1 {
				t.Fatalf("Expected one <%s>, got %#v", testCase.tag, elements)
			}
			element := elements[0]
			if element.TextContent != testCase.wantValue || len(element.Children) != 1 || element.Children[0].Type != types.TextNode {
				t.Fatalf("Expected one <%s> text node %q, got %#v", testCase.tag, testCase.wantValue, element)
			}
			text := element.Children[0]
			if text.Value != testCase.wantValue || text.StartPos != testCase.wantStart || text.EndPos != testCase.wantEnd {
				t.Fatalf("Expected text %q at byte range %d:%d, got %#v", testCase.wantValue, testCase.wantStart, testCase.wantEnd, text)
			}
			if source := content[text.StartPos:text.EndPos]; source != testCase.wantSource {
				t.Fatalf("Expected exact parse5 source %q, got %q", testCase.wantSource, source)
			}
			if text.StartLine != testCase.wantStartLine || text.StartColumn != testCase.wantStartColumn || text.EndLine != testCase.wantEndLine || text.EndColumn != testCase.wantEndColumn {
				t.Fatalf(
					"Expected parse5 coordinates %d:%d-%d:%d, got %d:%d-%d:%d",
					testCase.wantStartLine, testCase.wantStartColumn, testCase.wantEndLine, testCase.wantEndColumn,
					text.StartLine, text.StartColumn, text.EndLine, text.EndColumn,
				)
			}
			contentEnd := len(opening) + len(testCase.raw)
			if element.ContentStart != len(opening) || element.ContentEnd != contentEnd || element.EndPos != len(content) {
				t.Fatalf("Initial LF suppression must not change the element source range: %#v", element)
			}
		})
	}
}

func TestParsePreAndListingDropSoleInitialLineFeedNode(t *testing.T) {
	for _, tag := range []string{"pre", "listing"} {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			content := opening + "\n</" + tag + `>`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			elements := listElements(document, tag)
			if len(elements) != 1 || elements[0].TextContent != "" || len(elements[0].Children) != 0 {
				t.Fatalf("Expected stripped sole LF to emit no text node, got %#v", elements)
			}
			if elements[0].ContentStart != len(opening) || elements[0].ContentEnd != len(opening)+1 || elements[0].EndPos != len(content) {
				t.Fatalf("Expected source-backed empty <%s> range, got %#v", tag, elements[0])
			}
		})
	}
}

func TestParsePreInitialLineFeedSuppressionStopsAfterAnotherToken(t *testing.T) {
	testCases := []struct {
		name, content, owner, wantValue, wantSource string
		wantStart, wantEnd                          int
	}{
		{name: "comment token", content: "<pre><!--x-->\nabc</pre>", owner: "pre", wantValue: "\nabc", wantSource: "\nabc", wantStart: 13, wantEnd: 17},
		{name: "child start token", content: "<pre><span>\nabc</span></pre>", owner: "span", wantValue: "\nabc", wantSource: "\nabc", wantStart: 11, wantEnd: 15},
		{name: "preceding space character", content: "<pre> \nabc</pre>", owner: "pre", wantValue: " \nabc", wantSource: " \nabc", wantStart: 5, wantEnd: 10},
		{name: "xmp has no preformatted LF rule", content: "<xmp>\nabc</xmp>", owner: "xmp", wantValue: "\nabc", wantSource: "\nabc", wantStart: 5, wantEnd: 9},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			owners := listElements(document, testCase.owner)
			if len(owners) != 1 {
				t.Fatalf("Expected one <%s>, got %#v", testCase.owner, owners)
			}
			var text *types.Node
			for _, child := range owners[0].Children {
				if child.Type == types.TextNode {
					text = child
					break
				}
			}
			if text == nil || text.Value != testCase.wantValue || text.StartPos != testCase.wantStart || text.EndPos != testCase.wantEnd {
				t.Fatalf("Expected preserved text %q at %d:%d, got %#v", testCase.wantValue, testCase.wantStart, testCase.wantEnd, text)
			}
			if source := testCase.content[text.StartPos:text.EndPos]; source != testCase.wantSource {
				t.Fatalf("Expected exact preserved source %q, got %q", testCase.wantSource, source)
			}
		})
	}
}

func TestParsePreInitialLineFeedSuppressionIsConsumedByIgnoredToken(t *testing.T) {
	testCases := []struct {
		name, content, wantValue, wantSource string
		wantStart, wantEnd                   int
		wantStartLine, wantStartColumn       int
		wantEndLine, wantEndColumn           int
	}{
		{
			name: "ignored NUL character token", content: "<pre>\x00\nx</pre>",
			wantValue: "\nx", wantSource: "\nx", wantStart: 6, wantEnd: 8,
			wantStartLine: 1, wantStartColumn: 7, wantEndLine: 2, wantEndColumn: 2,
		},
		{
			name: "ignored doctype token", content: "<pre><!doctype html>\nx</pre>",
			wantValue: "\nx", wantSource: "\nx", wantStart: 20, wantEnd: 22,
			wantStartLine: 1, wantStartColumn: 21, wantEndLine: 2, wantEndColumn: 2,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			pre := listElements(document, "pre")
			if len(pre) != 1 || pre[0].TextContent != testCase.wantValue || len(pre[0].Children) != 1 || pre[0].Children[0].Type != types.TextNode {
				t.Fatalf("Ignored first token must consume the one-shot LF suppression, got %#v", pre)
			}
			text := pre[0].Children[0]
			if text.Value != testCase.wantValue || text.StartPos != testCase.wantStart || text.EndPos != testCase.wantEnd {
				t.Fatalf("Expected preserved text %q at %d:%d, got %#v", testCase.wantValue, testCase.wantStart, testCase.wantEnd, text)
			}
			if source := testCase.content[text.StartPos:text.EndPos]; source != testCase.wantSource {
				t.Fatalf("Expected exact parse5 source %q, got %q", testCase.wantSource, source)
			}
			if text.StartLine != testCase.wantStartLine || text.StartColumn != testCase.wantStartColumn || text.EndLine != testCase.wantEndLine || text.EndColumn != testCase.wantEndColumn {
				t.Fatalf(
					"Expected parse5 coordinates %d:%d-%d:%d, got %d:%d-%d:%d",
					testCase.wantStartLine, testCase.wantStartColumn, testCase.wantEndLine, testCase.wantEndColumn,
					text.StartLine, text.StartColumn, text.EndLine, text.EndColumn,
				)
			}
		})
	}
}

func TestParsePreInitialLineFeedSuppressionPersistsAcrossNonEmittingSyntax(t *testing.T) {
	const content = "<pre></>\nx</pre>"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	pre := listElements(document, "pre")
	if len(pre) != 1 || pre[0].TextContent != "x" || len(pre[0].Children) != 1 || pre[0].Children[0].Type != types.TextNode {
		t.Fatalf("Non-emitting </> syntax must leave one-shot LF suppression armed, got %#v", pre)
	}
	text := pre[0].Children[0]
	if text.Value != "x" || text.StartPos != 9 || text.EndPos != 10 || text.StartLine != 2 || text.StartColumn != 1 || text.EndLine != 2 || text.EndColumn != 2 {
		t.Fatalf("Expected browser text x at 9:10 and 2:1-2:2, got %#v", text)
	}
}

func TestParseStrippedPreLineFeedRangeExtendsAcrossFollowingNonEmittingSyntax(t *testing.T) {
	const content = "<pre>\n</>\nx</pre>"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	pre := listElements(document, "pre")
	if len(pre) != 1 || pre[0].TextContent != "\nx" || len(pre[0].Children) != 1 || pre[0].Children[0].Type != types.TextNode {
		t.Fatalf("Expected one surviving newline-x text node, got %#v", pre)
	}
	text := pre[0].Children[0]
	if text.Value != "\nx" || text.StartPos != 5 || text.EndPos != 11 || text.StartLine != 1 || text.StartColumn != 6 || text.EndLine != 3 || text.EndColumn != 2 {
		t.Fatalf("Expected value newline-x with parse5 range 5:11 and 1:6-3:2, got %#v", text)
	}
	if source := content[text.StartPos:text.EndPos]; source != "\n</>\nx" {
		t.Fatalf("Expected surviving text range to retain stripped LF and non-emitting syntax, got %q", source)
	}
}

func TestParseEmittedCommentSeparatesStrippedPreLineFeedFromFollowingTextRange(t *testing.T) {
	const content = "<pre>\n<!--c-->\nx</pre>"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	pre := listElements(document, "pre")
	if len(pre) != 1 || pre[0].TextContent != "\nx" || len(pre[0].Children) != 2 {
		t.Fatalf("Expected comment and following newline-x only, got %#v", pre)
	}
	comment, text := pre[0].Children[0], pre[0].Children[1]
	if comment.Type != types.CommentNode || comment.Value != "c" || comment.StartPos != 6 || comment.EndPos != 14 || comment.StartLine != 2 || comment.StartColumn != 1 || comment.EndLine != 2 || comment.EndColumn != 9 {
		t.Fatalf("Expected emitted comment c at 6:14 and 2:1-2:9, got %#v", comment)
	}
	if text.Type != types.TextNode || text.Value != "\nx" || text.StartPos != 14 || text.EndPos != 16 || text.StartLine != 2 || text.StartColumn != 9 || text.EndLine != 3 || text.EndColumn != 2 {
		t.Fatalf("Expected following newline-x at 14:16 and 2:9-3:2, got %#v", text)
	}
}

func TestParsePreInitialLineFeedInTableTemplateAndForeignIntegration(t *testing.T) {
	t.Run("fostered from table", func(t *testing.T) {
		const content = "<table><pre>\nx</pre></table>"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		body := parsedBodyChildren(document)
		if len(body) != 2 || body[0].Name != "pre" || body[1].Name != "table" {
			t.Fatalf("Expected fostered pre before table, got %#v", body)
		}
		pre, table := body[0], body[1]
		if pre.Parent != table.Parent || pre.TextContent != "x" || len(pre.Children) != 1 || pre.StartPos != 7 || pre.EndPos != 20 || table.StartPos != 0 || table.EndPos != 28 {
			t.Fatalf("Unexpected fostered pre/table tree or range: pre=%#v table=%#v", pre, table)
		}
		text := pre.Children[0]
		if text.Value != "x" || text.StartPos != 13 || text.EndPos != 14 || text.StartLine != 2 || text.StartColumn != 1 || text.EndLine != 2 || text.EndColumn != 2 {
			t.Fatalf("Expected fostered pre text x at 13:14 and 2:1-2:2, got %#v", text)
		}
	})

	t.Run("template content fragment", func(t *testing.T) {
		const content = "<template><pre>\nx</pre></template>"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		templates := listElements(document, "template")
		if len(templates) != 1 || templates[0].TextContent != "" || len(templates[0].Children) != 0 || templates[0].StartPos != 0 || templates[0].EndPos != 34 || templates[0].ContentStart != 10 || templates[0].ContentEnd != 23 {
			t.Fatalf("Unexpected template host: %#v", templates)
		}
		fragment := batch24TemplateContent(t, templates[0])
		if fragment.TextContent != "x" || len(fragment.Children) != 1 || fragment.Children[0].Name != "pre" {
			t.Fatalf("Expected one pre in TemplateContent, got %#v", fragment)
		}
		pre := fragment.Children[0]
		if pre.Parent != fragment || pre.TextContent != "x" || len(pre.Children) != 1 || pre.StartPos != 10 || pre.EndPos != 23 {
			t.Fatalf("Unexpected template pre tree or range: %#v", pre)
		}
		text := pre.Children[0]
		if text.Value != "x" || text.StartPos != 16 || text.EndPos != 17 || text.StartLine != 2 || text.StartColumn != 1 || text.EndLine != 2 || text.EndColumn != 2 {
			t.Fatalf("Expected template pre text x at 16:17 and 2:1-2:2, got %#v", text)
		}
	})

	t.Run("SVG foreignObject integration point", func(t *testing.T) {
		const content = "<svg><foreignObject><pre>\nx</pre></foreignObject></svg>"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		pres := listElements(document, "pre")
		foreignObjects := listElements(document, "foreignObject")
		if len(pres) != 1 || len(foreignObjects) != 1 {
			t.Fatalf("Expected one foreignObject and HTML pre, pre=%#v foreignObject=%#v", pres, foreignObjects)
		}
		pre, foreignObject := pres[0], foreignObjects[0]
		if pre.Parent != foreignObject || pre.NamespaceURI != htmlElementNamespace || foreignObject.NamespaceURI != svgElementNamespace || pre.TextContent != "x" || len(pre.Children) != 1 || pre.StartPos != 20 || pre.EndPos != 33 {
			t.Fatalf("Unexpected foreign integration tree or range: pre=%#v foreignObject=%#v", pre, foreignObject)
		}
		text := pre.Children[0]
		if text.Value != "x" || text.StartPos != 26 || text.EndPos != 27 || text.StartLine != 2 || text.StartColumn != 1 || text.EndLine != 2 || text.EndColumn != 2 {
			t.Fatalf("Expected integration-point pre text x at 26:27 and 2:1-2:2, got %#v", text)
		}
	})
}

func TestParseSoleStrippedPreLineFeedDoesNotCreateFormattingPhantom(t *testing.T) {
	const content = "<p><b id=b><pre id=p>\n</pre>y"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	pres := listElements(document, "pre")
	if len(pres) != 1 {
		t.Fatalf("Expected one pre, got %#v", pres)
	}
	pre := pres[0]
	if pre.TextContent != "" || len(pre.Children) != 0 || pre.StartPos != 11 || pre.EndPos != 28 || pre.ContentStart != 21 || pre.ContentEnd != 22 {
		t.Fatalf("Sole stripped LF must not reconstruct a formatting child inside pre: %#v", pre)
	}
	bold := listElements(document, "b")
	if len(bold) != 2 || bold[0].Parent == pre || bold[1].Parent == pre || bold[0].TextContent != "" || bold[1].TextContent != "y" {
		t.Fatalf("Expected empty source b and reconstructed trailing-y b outside pre, got %#v", bold)
	}
}

func TestParsePreInitialLineFeedStateDoesNotLeakAcrossReuse(t *testing.T) {
	parser := NewHTMLParser()
	first, err := parser.Parse("<pre>\nfirst</pre>")
	if err != nil {
		t.Fatalf("First Parse returned an error: %v", err)
	}
	firstPre := listElements(first, "pre")
	if len(firstPre) != 1 || firstPre[0].TextContent != "first" || firstPre[0].Children[0].StartPos != 6 {
		t.Fatalf("Expected first parse to strip its initial LF, got %#v", firstPre)
	}

	second, err := parser.Parse("<pre> \nsecond</pre><listing>\nthird</listing>")
	if err != nil {
		t.Fatalf("Second Parse returned an error: %v", err)
	}
	pre := listElements(second, "pre")
	listing := listElements(second, "listing")
	if len(pre) != 1 || pre[0].TextContent != " \nsecond" || len(listing) != 1 || listing[0].TextContent != "third" || listing[0].Children[0].StartPos != 29 {
		t.Fatalf("Preformatted initial-LF state leaked across parser reuse: pre=%#v listing=%#v", pre, listing)
	}
}

func TestParseManyPreformattedInitialLineFeedsScalesNearLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping parser scaling regression in short mode")
	}

	measure := func(count int) time.Duration {
		content := strings.Repeat("<pre>\nx</pre>", count)
		started := time.Now()
		document, err := NewHTMLParser().Parse(content)
		elapsed := time.Since(started)
		if err != nil {
			t.Fatalf("Parse(%d pre elements) failed: %v", count, err)
		}
		if got := len(listElements(document, "pre")); got != count {
			t.Fatalf("Parse(%d pre elements) returned %d", count, got)
		}
		return elapsed
	}

	_ = measure(64)
	minOfTwo := func(count int) time.Duration {
		first, second := measure(count), measure(count)
		if first < second {
			return first
		}
		return second
	}
	small := minOfTwo(256)
	large := minOfTwo(1024)
	t.Logf("parsed 256 pre elements in %s and 1024 in %s", small, large)
	if large > 10*small && large-small > 50*time.Millisecond {
		t.Fatalf("Initial-LF handling scales quadratically: 4x input took %.1fx longer (%s -> %s)", float64(large)/float64(small), small, large)
	}
}
