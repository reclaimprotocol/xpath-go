package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func assertParagraphQueryNode(t *testing.T, document, expression, wantText string, wantStart, wantEnd int) {
	t.Helper()
	results, err := xpath.Query(expression, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != wantText || results[0].StartLocation != wantStart || results[0].EndLocation != wantEnd {
		t.Fatalf("Expected %s text %q at %d:%d, got %#v", expression, wantText, wantStart, wantEnd, results)
	}
}

func TestQueryPendingParagraphCommentAndStackResolution(t *testing.T) {
	for _, declaration := range []string{`<!foo>`, `<?foo>`, `</$foo>`} {
		document := `<html><body><p>x</html>` + declaration + `<div>y</div>`
		blockStart := 23 + len(declaration)
		assertParagraphQueryNode(t, document, `//body/p`, "x", 12, blockStart)
		assertParagraphQueryNode(t, document, `//body/p/text()`, "x", 15, 16)
		assertParagraphQueryNode(t, document, `//body/div`, "y", blockStart, len(document))
	}
	for _, document := range []string{`<html><body><p>x</html></p>tail`, `<html><body><p>x</body></p>tail</html>`} {
		assertParagraphQueryNode(t, document, `//p`, "x", 12, 27)
		assertParagraphQueryNode(t, document, `//p/following-sibling::text()`, "tail", 27, 31)
	}
	const base = `<html><body><p>a<button><p>b</html>`
	document := base + `<div>y</div>`
	assertParagraphQueryNode(t, document, `//body/p`, "aby", 12, 47)
	assertParagraphQueryNode(t, document, `//button/p`, "b", 24, 35)
	assertParagraphQueryNode(t, document, `//button/div`, "y", 35, 47)
	document = base + `</p>tail`
	assertParagraphQueryNode(t, document, `//body/p`, "abtail", 12, 43)
	assertParagraphQueryNode(t, document, `//button/p`, "b", 24, 39)
	assertParagraphQueryNode(t, document, `//button/p/following-sibling::text()`, "tail", 39, 43)
	document = base + `</button>tail`
	assertParagraphQueryNode(t, document, `//body/p`, "abtail", 12, 48)
	assertParagraphQueryNode(t, document, `//button`, "b", 16, 44)
	assertParagraphQueryNode(t, document, `//button/following-sibling::text()`, "tail", 44, 48)
	document = `<html><body><p><span>x</html>tail`
	assertParagraphQueryNode(t, document, `//p`, "xtail", 12, 33)
	assertParagraphQueryNode(t, document, `//span`, "xtail", 15, 33)
	assertParagraphQueryNode(t, document, `//span/text()`, "xtail", 21, 33)
	document = `<html><body><p><span>x</html><!--c-->`
	assertParagraphQueryNode(t, document, `//p`, "x", 12, 37)
	assertParagraphQueryNode(t, document, `//span`, "x", 15, 37)
}

func TestQueryPendingParagraphContentRangesAndWhitespace(t *testing.T) {
	for _, tc := range []struct {
		document, expression, value string
		start, end                  int
	}{
		{`<html><body><p><span>x</html>tail`, `//p`, "xtail", 15, 33},
		{`<html><body><p><span>x</html>tail`, `//span`, "xtail", 21, 33},
		{`<html><body><p><span>x</html><!--c-->`, `//p`, "x", 15, 37},
		{`<html><body><p><span>x</html><!--c-->`, `//span`, "x", 21, 37},
	} {
		results, err := xpath.QueryWithOptions(tc.expression, tc.document, xpath.Options{IncludeLocation: true, OutputFormat: "nodes", ContentsOnly: true})
		if err != nil || len(results) != 1 || results[0].Value != tc.value || results[0].StartLocation != tc.start || results[0].EndLocation != tc.end {
			t.Fatalf("Expected content-only %s %q at %d:%d, got %#v err=%v", tc.expression, tc.value, tc.start, tc.end, results, err)
		}
	}
	const multiline = "<html>\n<body><p><span>x</html>\n</p>tail"
	assertParagraphQueryNode(t, multiline, `//p`, "x\n", 13, 35)
	assertParagraphQueryNode(t, multiline, `//span`, "x\n", 16, 31)
	const whitespace = `<html><body><p>x</html></p>   `
	assertParagraphQueryNode(t, whitespace, `//p`, "x", 12, 27)
	assertParagraphQueryNode(t, whitespace, `//p/following-sibling::text()`, "   ", 27, 30)
	for _, tc := range []struct{ document, wantText string }{{`<html><body><p>x</html><!--a--></p><!--b-->`, "x"}, {`<html><body><p>x</html><!--a--><div>y</div><!--b-->`, "xy"}} {
		document := tc.document
		results, err := xpath.Query(`//p`, document)
		if err != nil || len(results) != 1 || results[0].TextContent != "x" {
			t.Fatalf("Expected unpolluted p text for comment reentry, got %#v err=%v", results, err)
		}
		for _, expression := range []string{`//html`, `//body`} {
			results, err = xpath.Query(expression, document)
			if err != nil || len(results) != 1 || results[0].TextContent != tc.wantText {
				t.Fatalf("Expected %s text %q excluding comment data, got %#v err=%v", expression, tc.wantText, results, err)
			}
		}
	}
}

func TestQuerySpecialAncestorEndTagsUnwindOpenParagraph(t *testing.T) {
	for _, tag := range []string{"form", "li", "dd", "dt", "h1", "h2", "h3", "h4", "h5", "h6", "applet", "marquee"} {
		t.Run(tag, func(t *testing.T) {
			opening, closing := `<`+tag+`>`, `</`+tag+`>`
			document := opening + `<p>x` + closing + `tail`
			assertParagraphQueryNode(t, document, `//`+tag+`/p`, "x", len(opening), len(opening)+4)
			assertParagraphQueryNode(t, document, `//`+tag+`/following-sibling::text()`, "tail", len(document)-4, len(document))
		})
	}
}

func TestQueryXMPAndPlaintextAreSiblingsAfterParagraphAutoClose(t *testing.T) {
	t.Run("xmp", func(t *testing.T) {
		const document = `<p><xmp>x</xmp>y</p>`
		assertParagraphQueryNode(t, document, `//p[1]`, "", 0, 3)
		assertParagraphQueryNode(t, document, `//xmp/text()`, "x", 8, 9)
		assertParagraphQueryNode(t, document, `//xmp/following-sibling::text()[1]`, "y", 15, 16)
		assertParagraphQueryNode(t, document, `//p[2]`, "", 0, 0)
	})

	t.Run("plaintext", func(t *testing.T) {
		const document = `<p><plaintext>x`
		assertParagraphQueryNode(t, document, `//p`, "", 0, 3)
		assertParagraphQueryNode(t, document, `//plaintext/text()`, "x", 14, 15)
	})
}

func TestQueryRepresentativeBlockStartsAutoCloseParagraph(t *testing.T) {
	for _, tag := range []string{"div", "address", "article", "section", "h2", "pre", "ul", "form"} {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			closing := `</` + tag + `>`
			document := `<p>a` + opening + `b` + closing + `c</p>`
			assertParagraphQueryNode(t, document, `//p[1]/text()`, "a", 3, 4)
			blockStart := 4
			assertParagraphQueryNode(t, document, `//`+tag+`/text()`, "b", blockStart+len(opening), blockStart+len(opening)+1)
			assertParagraphQueryNode(t, document, `//`+tag+`/following-sibling::text()[1]`, "c", blockStart+len(opening+`b`+closing), blockStart+len(opening+`b`+closing)+1)
			assertParagraphQueryNode(t, document, `//p[2]`, "", 0, 0)
		})
	}
}

func TestQueryStrayParagraphEndTagCreatesEmptyParagraph(t *testing.T) {
	for _, document := range []string{`before</p>after`, `<div>before</p>after</div>`} {
		t.Run(document, func(t *testing.T) {
			assertParagraphQueryNode(t, document, `//p`, "", 0, 0)
		})
	}
}

func TestQueryParagraphStartAndAncestorCloseImplicitlyCloseOpenParagraph(t *testing.T) {
	t.Run("paragraph start", func(t *testing.T) {
		const document = `<p>one<p>two</p>`
		assertParagraphQueryNode(t, document, `//p[1]`, "one", 0, 6)
		assertParagraphQueryNode(t, document, `//p[2]`, "two", 6, len(document))
	})
	t.Run("ancestor div close", func(t *testing.T) {
		const document = `<div><p>x</div>`
		assertParagraphQueryNode(t, document, `//div/p`, "x", 5, 9)
		assertParagraphQueryNode(t, document, `//div`, "x", 0, len(document))
	})
}

func TestQueryParagraphScopeBoundariesKeepParagraphOpen(t *testing.T) {
	for _, tag := range []string{"button", "object"} {
		t.Run(tag, func(t *testing.T) {
			document := `<p>x<` + tag + `>y<div>z</div>w</` + tag + `>q`
			assertParagraphQueryNode(t, document, `//p`, "xyzwq", 0, len(document))
		})
	}
}

func TestQueryInitialParagraphEndTagIsIgnoredBeforeBodyContent(t *testing.T) {
	const document = `</p><p>x</p>`
	assertParagraphQueryNode(t, document, `//p`, "x", 4, len(document))
}

func TestQueryParagraphAutoClosePreservesNestedMultibyteLocations(t *testing.T) {
	const document = `<div><p>é<div>x</div>y</p><span>z</span></div>`
	assertParagraphQueryNode(t, document, `//p[1]`, "é", 5, 10)
	assertParagraphQueryNode(t, document, `//p[1]/text()`, "é", 8, 10)
	assertParagraphQueryNode(t, document, `//p[1]/following-sibling::div/text()`, "x", 15, 16)
	assertParagraphQueryNode(t, document, `//p[1]/following-sibling::text()[1]`, "y", 22, 23)
	assertParagraphQueryNode(t, document, `//p[2]`, "", 0, 0)
	assertParagraphQueryNode(t, document, `//span/text()`, "z", 33, 34)
}

func TestQueryParagraphAutoClosePreservesMultibyteCRLFLocation(t *testing.T) {
	const document = "<p>é\r\n<div>y</div>"
	assertParagraphQueryNode(t, document, `//p`, "é\n", 0, 7)
	assertParagraphQueryNode(t, document, `//p/text()`, "é\n", 3, 7)
	assertParagraphQueryNode(t, document, `//p/following-sibling::div/text()`, "y", 12, 13)
}

func TestQueryIncompleteTagsAtEOFStayInParagraphSourceRange(t *testing.T) {
	testCases := []struct {
		name, document, expression string
		wantStart                  int
	}{
		{name: "div name", document: `<p>x<div`, expression: `//p/text()`, wantStart: 3},
		{name: "div attribute", document: `<p>x<div a="`, expression: `//p/text()`, wantStart: 3},
		{name: "p name", document: `<p>x<p`, expression: `//p/text()`, wantStart: 3},
		{name: "ancestor close name", document: `<div><p>x</div`, expression: `//p/text()`, wantStart: 8},
		{name: "ancestor close attribute", document: `<div><p>x</div `, expression: `//p/text()`, wantStart: 8},
		{name: "span div", document: `<p><span>x<div`, expression: `//p/span/text()`, wantStart: 9},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assertParagraphQueryNode(t, testCase.document, testCase.expression, "x", testCase.wantStart, len(testCase.document))
		})
	}
}

func TestQueryHeadContextStrayParagraphEndTagsAreIgnored(t *testing.T) {
	testCases := []struct {
		name, document string
		wantCount      int
	}{
		{name: "after meta", document: `<meta></p>`},
		{name: "after title", document: `<title>x</title></p>`},
		{name: "after script", document: `<script>x</script></p>`},
		{name: "explicit head meta", document: `<html><head><meta></p></head><body><p>x</p></body></html>`, wantCount: 1},
		{name: "explicit head title", document: `<html><head><title>x</title></p></head><body></body></html>`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			paragraphs, err := xpath.Query(`//p`, testCase.document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(paragraphs) != testCase.wantCount {
				t.Fatalf("Expected %d paragraphs, got %#v", testCase.wantCount, paragraphs)
			}
			if testCase.wantCount == 1 && (paragraphs[0].TextContent != "x" || paragraphs[0].StartLocation != 35 || paragraphs[0].EndLocation != 43) {
				t.Fatalf("Expected only explicit body p at 35:43, got %#v", paragraphs[0])
			}
		})
	}
}

func TestQueryMalformedBlockLikeTagNamesStayInsideParagraph(t *testing.T) {
	testCases := []struct {
		name, document, childName string
		childStart, childEnd      int
	}{
		{name: "equals", document: `<p>x<div=y>z</div=y>w</p>`, childName: "div=y", childStart: 4, childEnd: 20},
		{name: "bang", document: `<p>x<div!>z</div!>w</p>`, childName: "div!", childStart: 4, childEnd: 18},
		{name: "unicode", document: `<p>x<divé>z</divé>w</p>`, childName: "divé", childStart: 4, childEnd: 20},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assertParagraphQueryNode(t, testCase.document, `//p`, "xzw", 0, len(testCase.document))
			assertParagraphQueryNode(t, testCase.document, `//p/*`, "z", testCase.childStart, testCase.childEnd)
		})
	}
}

func TestQueryParagraphEndTagContextTransitions(t *testing.T) {
	t.Run("html context ignores p close", func(t *testing.T) {
		const document = `<html></p></html>`
		paragraphs, err := xpath.Query(`//p`, document)
		if err != nil || len(paragraphs) != 0 {
			t.Fatalf("Expected no p, results=%#v err=%v", paragraphs, err)
		}
	})
	t.Run("head exits to body-like content", func(t *testing.T) {
		const document = `<html><head><div>x</div></p></head></html>`
		assertParagraphQueryNode(t, document, `//div`, "x", 12, 24)
		assertParagraphQueryNode(t, document, `//p`, "", 0, 0)
	})
}

func TestQueryScopeBoundaryEndTagsCloseInnerParagraphs(t *testing.T) {
	for _, tag := range []string{"button", "object"} {
		t.Run(tag, func(t *testing.T) {
			document := `<` + tag + `><p>x</` + tag + `>`
			assertParagraphQueryNode(t, document, `//`+tag+`/p`, "x", len(`<`+tag+`>`), len(`<`+tag+`><p>x`))
		})
	}
	t.Run("html", func(t *testing.T) {
		const document = `<html><body><p>x</html>`
		assertParagraphQueryNode(t, document, `//p`, "x", 12, len(document))
	})
}

func TestQueryBodyCommittingTextControlsStrayParagraphEndRecovery(t *testing.T) {
	testCases := []struct {
		name, document, wantValue string
		wantStart, wantEnd        int
		wantPCount                int
	}{
		{name: "html literal", document: `<html>x</p></html>`, wantValue: "x", wantStart: 6, wantEnd: 7, wantPCount: 1},
		{name: "head literal", document: `<html><head>x</p></head></html>`, wantValue: "x", wantStart: 12, wantEnd: 13, wantPCount: 1},
		{name: "html decoded", document: `<html>&#65;</p></html>`, wantValue: "A", wantStart: 6, wantEnd: 11, wantPCount: 1},
		{name: "head decoded", document: `<html><head>&#65;</p></head></html>`, wantValue: "A", wantStart: 12, wantEnd: 17, wantPCount: 1},
		{name: "html whitespace", document: "<html> \n</p></html>"},
		{name: "head whitespace", document: "<html><head> \n</p></head></html>"},
		{name: "html decoded whitespace", document: `<html>&#32;</p></html>`},
		{name: "head decoded whitespace", document: `<html><head>&#32;</p></head></html>`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			paragraphs, err := xpath.Query(`//p`, testCase.document)
			if err != nil {
				t.Fatalf("Query p returned an error: %v", err)
			}
			if len(paragraphs) != testCase.wantPCount {
				t.Fatalf("Expected %d p nodes, got %#v", testCase.wantPCount, paragraphs)
			}
			if testCase.wantPCount == 0 {
				return
			}
			if paragraphs[0].StartLocation != 0 || paragraphs[0].EndLocation != 0 || paragraphs[0].TextContent != "" {
				t.Fatalf("Expected synthetic p at 0:0, got %#v", paragraphs[0])
			}
			assertParagraphQueryNode(t, testCase.document, `//text()`, testCase.wantValue, testCase.wantStart, testCase.wantEnd)
		})
	}
}

func TestQueryAncestorEndTagsUnwindParagraphAndButton(t *testing.T) {
	t.Run("div", func(t *testing.T) {
		const document = `<div><button><p>x</div>tail`
		assertParagraphQueryNode(t, document, `//div`, "x", 0, 23)
		assertParagraphQueryNode(t, document, `//div/button`, "x", 5, 17)
		assertParagraphQueryNode(t, document, `//div/button/p`, "x", 13, 17)
		assertParagraphQueryNode(t, document, `//div/following-sibling::text()`, "tail", 23, 27)
	})
	t.Run("section", func(t *testing.T) {
		const document = `<section><button><p>x</section>`
		assertParagraphQueryNode(t, document, `//section`, "x", 0, 31)
		assertParagraphQueryNode(t, document, `//section/button`, "x", 9, 21)
		assertParagraphQueryNode(t, document, `//section/button/p`, "x", 17, 21)
	})
}

func TestQueryHTMLCloseExtendsAllPendingParagraphsToEOF(t *testing.T) {
	const document = `<html><body><p>a<button><p>b</html>`
	assertParagraphQueryNode(t, document, `//body/p`, "ab", 12, 35)
	assertParagraphQueryNode(t, document, `//button`, "b", 16, 35)
	assertParagraphQueryNode(t, document, `//button/p`, "b", 24, 35)
}

func TestQueryBodyAndHTMLCloseKeepParagraphLocationPendingToEOF(t *testing.T) {
	testCases := []struct {
		name, document                 string
		htmlEnd, bodyEnd, paragraphEnd int
	}{
		{name: "body and html close", document: `<html><body><p>x</body></html>`, htmlEnd: 30, bodyEnd: 23, paragraphEnd: 30},
		{name: "comment after body and html", document: `<html><body><p>x</body></html><!--c-->`, htmlEnd: 30, bodyEnd: 23, paragraphEnd: 38},
		{name: "comment after html", document: `<html><body><p>x</html><!--c-->`, htmlEnd: 23, bodyEnd: 16, paragraphEnd: 31},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assertParagraphQueryNode(t, testCase.document, `//html`, "x", 0, testCase.htmlEnd)
			assertParagraphQueryNode(t, testCase.document, `//body`, "x", 6, testCase.bodyEnd)
			assertParagraphQueryNode(t, testCase.document, `//p`, "x", 12, testCase.paragraphEnd)
			assertParagraphQueryNode(t, testCase.document, `//p/text()`, "x", 15, 16)
		})
	}
}

func TestQueryNULInAncestorTagNameStillClosesInnerParagraph(t *testing.T) {
	const document = "<div\x00><p>x</div\x00>"
	assertParagraphQueryNode(t, document, `//p`, "x", 6, 17)
}

func TestQueryGenericAncestorEndTagIsIgnoredWhileParagraphIsOpen(t *testing.T) {
	testCases := []struct {
		document, ancestor        string
		paragraphStart, textStart int
	}{
		{document: `<foo><p>x</foo>tail`, ancestor: "foo", paragraphStart: 5, textStart: 8},
		{document: `<span><p>x</span>tail`, ancestor: "span", paragraphStart: 6, textStart: 9},
	}
	for _, testCase := range testCases {
		t.Run(testCase.ancestor, func(t *testing.T) {
			assertParagraphQueryNode(t, testCase.document, `//`+testCase.ancestor, "xtail", 0, len(testCase.document))
			assertParagraphQueryNode(t, testCase.document, `//`+testCase.ancestor+`/p`, "xtail", testCase.paragraphStart, len(testCase.document))
			assertParagraphQueryNode(t, testCase.document, `//`+testCase.ancestor+`/p/text()`, "xtail", testCase.textStart, len(testCase.document))
		})
	}
}

func TestQueryPendingParagraphResolvesOnFollowingBlockOrEOFText(t *testing.T) {
	t.Run("block after html close", func(t *testing.T) {
		const document = `<html><body><p>x</html><div>y</div>`
		assertParagraphQueryNode(t, document, `//body/p`, "x", 12, 23)
		assertParagraphQueryNode(t, document, `//body/div`, "y", 23, 35)
		assertParagraphQueryNode(t, document, `//html`, "xy", 0, 23)
		assertParagraphQueryNode(t, document, `//body`, "xy", 6, 16)
	})
	t.Run("block after body close", func(t *testing.T) {
		const document = `<html><body><p>x</body><div>y</div></html>`
		assertParagraphQueryNode(t, document, `//body/p`, "x", 12, 23)
		assertParagraphQueryNode(t, document, `//body/div`, "y", 23, 35)
		assertParagraphQueryNode(t, document, `//html`, "xy", 0, 42)
		assertParagraphQueryNode(t, document, `//body`, "xy", 6, 23)
	})
	t.Run("text after html close", func(t *testing.T) {
		const document = `<html><body><p>x</html>tail`
		assertParagraphQueryNode(t, document, `//p`, "xtail", 12, 27)
		assertParagraphQueryNode(t, document, `//p/text()`, "xtail", 15, 27)
		assertParagraphQueryNode(t, document, `//html`, "xtail", 0, 23)
		assertParagraphQueryNode(t, document, `//body`, "xtail", 6, 16)
	})
	t.Run("whitespace after html close", func(t *testing.T) {
		const document = `<html><body><p>x</html>   `
		assertParagraphQueryNode(t, document, `//p`, "x   ", 12, 26)
		assertParagraphQueryNode(t, document, `//p/text()`, "x   ", 15, 26)
		assertParagraphQueryNode(t, document, `//html`, "x   ", 0, 23)
		assertParagraphQueryNode(t, document, `//body`, "x   ", 6, 16)
	})
}
