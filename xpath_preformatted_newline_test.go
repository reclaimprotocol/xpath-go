package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryPreAndListingInitialLineFeedUsesBrowserTextAndRange(t *testing.T) {
	testCases := []struct {
		name, tag, raw, wantValue, wantSource string
		wantStart, wantEnd                    int
	}{
		{name: "pre LF", tag: "pre", raw: "\nabc", wantValue: "abc", wantSource: "abc", wantStart: 6, wantEnd: 9},
		{name: "listing LF", tag: "listing", raw: "\nabc", wantValue: "abc", wantSource: "abc", wantStart: 10, wantEnd: 13},
		{name: "CRLF", tag: "pre", raw: "\r\nabc", wantValue: "abc", wantSource: "abc", wantStart: 7, wantEnd: 10},
		{name: "lone CR", tag: "pre", raw: "\rabc", wantValue: "abc", wantSource: "abc", wantStart: 6, wantEnd: 9},
		{name: "double LF", tag: "pre", raw: "\n\nabc", wantValue: "\nabc", wantSource: "\n\nabc", wantStart: 5, wantEnd: 10},
		{name: "numeric LF", tag: "pre", raw: "&#10;abc", wantValue: "abc", wantSource: "abc", wantStart: 10, wantEnd: 13},
		{name: "named LF", tag: "pre", raw: "&NewLine;abc", wantValue: "abc", wantSource: "abc", wantStart: 14, wantEnd: 17},
		{name: "UTF-8 and UTF-16", tag: "pre", raw: "\né😀", wantValue: "é😀", wantSource: "é😀", wantStart: 6, wantEnd: 12},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document := `<` + testCase.tag + `>` + testCase.raw + `</` + testCase.tag + `>`
			results, err := xpath.Query(`//`+testCase.tag+`/text()`, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].NodeName != "#text" || results[0].TextContent != testCase.wantValue || results[0].StartLocation != testCase.wantStart || results[0].EndLocation != testCase.wantEnd {
				t.Fatalf("Expected text %q at byte range %d:%d, got %#v", testCase.wantValue, testCase.wantStart, testCase.wantEnd, results)
			}
			if source := document[results[0].StartLocation:results[0].EndLocation]; source != testCase.wantSource {
				t.Fatalf("Expected exact browser source %q, got %q", testCase.wantSource, source)
			}
		})
	}
}

func TestQueryPreAndListingSoleInitialLineFeedHasNoTextNode(t *testing.T) {
	for _, tag := range []string{"pre", "listing"} {
		t.Run(tag, func(t *testing.T) {
			document := `<` + tag + ">\n</" + tag + `>`
			texts, err := xpath.Query(`//`+tag+`/text()`, document)
			if err != nil || len(texts) != 0 {
				t.Fatalf("Expected stripped sole LF to emit no text node, results=%#v err=%v", texts, err)
			}
			elements, err := xpath.Query(`//`+tag, document)
			if err != nil || len(elements) != 1 || elements[0].TextContent != "" || elements[0].StartLocation != 0 || elements[0].EndLocation != len(document) {
				t.Fatalf("Expected one empty source-backed <%s>, results=%#v err=%v", tag, elements, err)
			}
		})
	}
}

func TestQueryPreInitialLineFeedBoundaryTokensRemainVisible(t *testing.T) {
	testCases := []struct {
		name, document, expression, wantValue, wantSource string
		wantStart, wantEnd                                int
	}{
		{name: "comment", document: "<pre><!--x-->\nabc</pre>", expression: "//pre/text()", wantValue: "\nabc", wantSource: "\nabc", wantStart: 13, wantEnd: 17},
		{name: "child", document: "<pre><span>\nabc</span></pre>", expression: "//pre/span/text()", wantValue: "\nabc", wantSource: "\nabc", wantStart: 11, wantEnd: 15},
		{name: "space", document: "<pre> \nabc</pre>", expression: "//pre/text()", wantValue: " \nabc", wantSource: " \nabc", wantStart: 5, wantEnd: 10},
		{name: "xmp", document: "<xmp>\nabc</xmp>", expression: "//xmp/text()", wantValue: "\nabc", wantSource: "\nabc", wantStart: 5, wantEnd: 9},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(testCase.expression, testCase.document)
			if err != nil || len(results) != 1 || results[0].TextContent != testCase.wantValue || results[0].StartLocation != testCase.wantStart || results[0].EndLocation != testCase.wantEnd {
				t.Fatalf("Expected preserved text %q at %d:%d, results=%#v err=%v", testCase.wantValue, testCase.wantStart, testCase.wantEnd, results, err)
			}
			if source := testCase.document[results[0].StartLocation:results[0].EndLocation]; source != testCase.wantSource {
				t.Fatalf("Expected exact preserved source %q, got %q", testCase.wantSource, source)
			}
		})
	}
}

func TestQueryPreInitialLineFeedSuppressionIsConsumedByIgnoredToken(t *testing.T) {
	testCases := []struct {
		name, document, wantValue, wantSource string
		wantStart, wantEnd                    int
	}{
		{name: "NUL character token", document: "<pre>\x00\nx</pre>", wantValue: "\nx", wantSource: "\nx", wantStart: 6, wantEnd: 8},
		{name: "doctype token", document: "<pre><!doctype html>\nx</pre>", wantValue: "\nx", wantSource: "\nx", wantStart: 20, wantEnd: 22},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query("//pre/text()", testCase.document)
			if err != nil || len(results) != 1 || results[0].TextContent != testCase.wantValue || results[0].StartLocation != testCase.wantStart || results[0].EndLocation != testCase.wantEnd {
				t.Fatalf("Ignored first token must consume LF suppression; expected %q at %d:%d, results=%#v err=%v", testCase.wantValue, testCase.wantStart, testCase.wantEnd, results, err)
			}
			if source := testCase.document[results[0].StartLocation:results[0].EndLocation]; source != testCase.wantSource {
				t.Fatalf("Expected exact parse5 source %q, got %q", testCase.wantSource, source)
			}
		})
	}
}

func TestQueryPreInitialLineFeedSuppressionPersistsAcrossNonEmittingSyntax(t *testing.T) {
	const document = "<pre></>\nx</pre>"
	results, err := xpath.Query("//pre/text()", document)
	if err != nil || len(results) != 1 || results[0].TextContent != "x" || results[0].StartLocation != 9 || results[0].EndLocation != 10 {
		t.Fatalf("Non-emitting </> syntax must leave LF suppression armed, results=%#v err=%v", results, err)
	}
}

func TestQueryStrippedPreLineFeedRangeExtendsAcrossFollowingNonEmittingSyntax(t *testing.T) {
	const document = "<pre>\n</>\nx</pre>"
	results, err := xpath.Query("//pre/text()", document)
	if err != nil || len(results) != 1 || results[0].TextContent != "\nx" || results[0].StartLocation != 5 || results[0].EndLocation != 11 {
		t.Fatalf("Expected newline-x with source range 5:11, results=%#v err=%v", results, err)
	}
	if source := document[results[0].StartLocation:results[0].EndLocation]; source != "\n</>\nx" {
		t.Fatalf("Expected source range to retain stripped LF and non-emitting syntax, got %q", source)
	}
}

func TestQueryEmittedCommentSeparatesStrippedPreLineFeedFromFollowingTextRange(t *testing.T) {
	const document = "<pre>\n<!--c-->\nx</pre>"
	comments, err := xpath.Query("//pre/comment()", document)
	if err != nil || len(comments) != 1 || comments[0].TextContent != "c" || comments[0].StartLocation != 6 || comments[0].EndLocation != 14 {
		t.Fatalf("Expected comment c at 6:14, results=%#v err=%v", comments, err)
	}
	texts, err := xpath.Query("//pre/text()", document)
	if err != nil || len(texts) != 1 || texts[0].TextContent != "\nx" || texts[0].StartLocation != 14 || texts[0].EndLocation != 16 {
		t.Fatalf("Expected only following newline-x at 14:16, results=%#v err=%v", texts, err)
	}
}

func TestQueryPreInitialLineFeedAcrossInsertionContexts(t *testing.T) {
	t.Run("fostered from table", func(t *testing.T) {
		const document = "<table><pre>\nx</pre></table>"
		text, err := xpath.Query("/html/body/pre/text()", document)
		if err != nil || len(text) != 1 || text[0].TextContent != "x" || text[0].StartLocation != 13 || text[0].EndLocation != 14 {
			t.Fatalf("Expected fostered pre text x at 13:14, results=%#v err=%v", text, err)
		}
		parents, err := xpath.Query("/html/body/pre/parent::body", document)
		if err != nil || len(parents) != 1 {
			t.Fatalf("Expected pre foster-parented into body, results=%#v err=%v", parents, err)
		}
	})

	t.Run("template content remains outside whole-document XPath", func(t *testing.T) {
		const document = "<template><pre>\nx</pre></template>"
		templates, err := xpath.Query("//template", document)
		if err != nil || len(templates) != 1 || templates[0].TextContent != "" || templates[0].StartLocation != 0 || templates[0].EndLocation != 34 || templates[0].ContentStart != 10 || templates[0].ContentEnd != 23 {
			t.Fatalf("Expected empty source-backed template host, results=%#v err=%v", templates, err)
		}
		for _, expression := range []string{"//template/pre", "//template//text()", "//pre"} {
			results, err := xpath.Query(expression, document)
			if err != nil || len(results) != 0 {
				t.Fatalf("TemplateContent leaked through %q, results=%#v err=%v", expression, results, err)
			}
		}
	})

	t.Run("SVG foreignObject integration point", func(t *testing.T) {
		const document = "<svg><foreignObject><pre>\nx</pre></foreignObject></svg>"
		results, err := xpath.Query("//foreignObject/pre/text()", document)
		if err != nil || len(results) != 1 || results[0].TextContent != "x" || results[0].StartLocation != 26 || results[0].EndLocation != 27 {
			t.Fatalf("Expected integration-point pre text x at 26:27, results=%#v err=%v", results, err)
		}
	})
}

func TestQuerySoleStrippedPreLineFeedHasNoFormattingPhantom(t *testing.T) {
	const document = "<p><b id=b><pre id=p>\n</pre>y"
	pre, err := xpath.Query("//*[@id='p']", document)
	if err != nil || len(pre) != 1 || pre[0].TextContent != "" || pre[0].StartLocation != 11 || pre[0].EndLocation != 28 || pre[0].ContentStart != 21 || pre[0].ContentEnd != 22 {
		t.Fatalf("Expected empty source-backed pre, results=%#v err=%v", pre, err)
	}
	for _, expression := range []string{"//*[@id='p']/node()", "//*[@id='p']//b"} {
		results, err := xpath.Query(expression, document)
		if err != nil || len(results) != 0 {
			t.Fatalf("Stripped LF created a phantom pre child through %q: results=%#v err=%v", expression, results, err)
		}
	}
}
