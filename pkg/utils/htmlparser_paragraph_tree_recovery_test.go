package utils

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func assertNodeIdentity(t *testing.T, node *types.Node, name, text string, start, end int) {
	t.Helper()
	if node == nil || node.Name != name || node.TextContent != text || node.StartPos != start || node.EndPos != end {
		t.Fatalf("Expected <%s> text %q at %d:%d, got %#v", name, text, start, end, node)
	}
}

func TestParseXMPAndPlaintextStartTagsCloseParagraphLikeBrowser(t *testing.T) {
	t.Run("xmp closes p and stray p end creates empty p", func(t *testing.T) {
		const content = `<p><xmp>x</xmp>y</p>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		if len(parsedBodyChildren(document)) != 4 {
			t.Fatalf("Expected empty source p, xmp, y, and synthetic p, got %#v", parsedBodyChildren(document))
		}
		assertNodeIdentity(t, parsedBodyChildren(document)[0], "p", "", 0, len(`<p>`))
		assertNodeIdentity(t, parsedBodyChildren(document)[1], "xmp", "x", len(`<p>`), len(`<p><xmp>x</xmp>`))
		text := parsedBodyChildren(document)[2]
		if text.Type != types.TextNode || text.Value != "y" || text.StartPos != 15 || text.EndPos != 16 {
			t.Fatalf("Expected root y at 15:16, got %#v", text)
		}
		assertNodeIdentity(t, parsedBodyChildren(document)[3], "p", "", 0, 0)
	})

	t.Run("plaintext closes p and consumes EOF", func(t *testing.T) {
		const content = `<p><plaintext>x`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		if len(parsedBodyChildren(document)) != 2 {
			t.Fatalf("Expected empty p followed by plaintext, got %#v", parsedBodyChildren(document))
		}
		assertNodeIdentity(t, parsedBodyChildren(document)[0], "p", "", 0, len(`<p>`))
		assertNodeIdentity(t, parsedBodyChildren(document)[1], "plaintext", "x", len(`<p>`), len(content))
	})
}

func TestParseBlockStartTagsAutoCloseParagraphLikeBrowser(t *testing.T) {
	for _, tag := range []string{"div", "address", "article", "section", "h2", "pre", "ul", "form"} {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			closing := `</` + tag + `>`
			content := `<p>a` + opening + `b` + closing + `c</p>`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 4 {
				t.Fatalf("Expected p, block, c, synthetic p, got %#v", parsedBodyChildren(document))
			}
			assertNodeIdentity(t, parsedBodyChildren(document)[0], "p", "a", 0, len(`<p>a`))
			blockStart := len(`<p>a`)
			blockEnd := blockStart + len(opening+`b`+closing)
			assertNodeIdentity(t, parsedBodyChildren(document)[1], tag, "b", blockStart, blockEnd)
			text := parsedBodyChildren(document)[2]
			if text.Type != types.TextNode || text.Value != "c" || text.StartPos != blockEnd || text.EndPos != blockEnd+1 {
				t.Fatalf("Expected c immediately after block at %d:%d, got %#v", blockEnd, blockEnd+1, text)
			}
			assertNodeIdentity(t, parsedBodyChildren(document)[3], "p", "", 0, 0)
		})
	}
}

func TestParseStrayParagraphEndTagCreatesEmptyParagraph(t *testing.T) {
	testCases := []struct {
		name, content string
		parentName    string
	}{
		{name: "document root", content: `before</p>after`, parentName: "#document"},
		{name: "nested div", content: `<div>before</p>after</div>`, parentName: "div"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			parent := parsedBody(document)
			if testCase.parentName == "div" {
				if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
					t.Fatalf("Expected outer div, got %#v", parsedBodyChildren(document))
				}
				parent = parsedBodyChildren(document)[0]
			}
			if len(parent.Children) != 3 {
				t.Fatalf("Expected before, synthetic p, after, got %#v", parent.Children)
			}
			before, paragraph, after := parent.Children[0], parent.Children[1], parent.Children[2]
			if before.Type != types.TextNode || before.Value != "before" || after.Type != types.TextNode || after.Value != "after" {
				t.Fatalf("Expected before/after text around synthetic p, got %#v", parent.Children)
			}
			assertNodeIdentity(t, paragraph, "p", "", 0, 0)
		})
	}
}

func TestParseParagraphStartAndAncestorCloseImplicitlyCloseOpenParagraph(t *testing.T) {
	t.Run("paragraph start", func(t *testing.T) {
		const content = `<p>one<p>two</p>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		if len(parsedBodyChildren(document)) != 2 {
			t.Fatalf("Expected two sibling paragraphs, got %#v", parsedBodyChildren(document))
		}
		assertNodeIdentity(t, parsedBodyChildren(document)[0], "p", "one", 0, 6)
		assertNodeIdentity(t, parsedBodyChildren(document)[1], "p", "two", 6, len(content))
	})

	t.Run("ancestor div close", func(t *testing.T) {
		const content = `<div><p>x</div>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" || len(parsedBodyChildren(document)[0].Children) != 1 {
			t.Fatalf("Expected outer div with one p, got %#v", parsedBodyChildren(document))
		}
		assertNodeIdentity(t, parsedBodyChildren(document)[0], "div", "x", 0, len(content))
		assertNodeIdentity(t, parsedBodyChildren(document)[0].Children[0], "p", "x", 5, 9)
	})
}

func TestParseParagraphAutoCloseUnwindsInlineElements(t *testing.T) {
	t.Run("block start unwinds span and p", func(t *testing.T) {
		const content = `<p><span>x<div>y</div>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		if len(parsedBodyChildren(document)) != 3 {
			t.Fatalf("Expected p, div, z siblings, got %#v", parsedBodyChildren(document))
		}
		assertNodeIdentity(t, parsedBodyChildren(document)[0], "p", "x", 0, 10)
		assertNodeIdentity(t, parsedBodyChildren(document)[0].Children[0], "span", "x", 3, 10)
		assertNodeIdentity(t, parsedBodyChildren(document)[1], "div", "y", 10, 22)
	})

	t.Run("p end unwinds span", func(t *testing.T) {
		const content = `<p><span>x</p>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		if len(parsedBodyChildren(document)) != 2 {
			t.Fatalf("Expected p and z siblings, got %#v", parsedBodyChildren(document))
		}
		assertNodeIdentity(t, parsedBodyChildren(document)[0], "p", "x", 0, 14)
		assertNodeIdentity(t, parsedBodyChildren(document)[0].Children[0], "span", "x", 3, 10)
	})
}

func TestParseParagraphScopeBoundariesKeepParagraphOpen(t *testing.T) {
	for _, tag := range []string{"button", "object"} {
		t.Run(tag, func(t *testing.T) {
			content := `<p>x<` + tag + `>y<div>z</div>w</` + tag + `>q`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" {
				t.Fatalf("Expected p to remain open across %s scope boundary, got %#v", tag, parsedBodyChildren(document))
			}
			assertNodeIdentity(t, parsedBodyChildren(document)[0], "p", "xyzwq", 0, len(content))
		})
	}
}

func TestParseInitialParagraphEndTagIsIgnoredBeforeBodyContent(t *testing.T) {
	const content = `</p><p>x</p>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 {
		t.Fatalf("Expected initial stray p close to be ignored, got %#v", parsedBodyChildren(document))
	}
	assertNodeIdentity(t, parsedBodyChildren(document)[0], "p", "x", 4, len(content))
}

func TestParseParagraphAutoClosePreservesNestedMultibyteLocations(t *testing.T) {
	const content = `<div><p>é<div>x</div>y</p><span>z</span></div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
		t.Fatalf("Expected outer div, got %#v", parsedBodyChildren(document))
	}
	outer := parsedBodyChildren(document)[0]
	assertNodeIdentity(t, outer, "div", "éxyz", 0, len(content))
	if len(outer.Children) != 5 {
		t.Fatalf("Expected p, div, y, synthetic p, span, got %#v", outer.Children)
	}
	assertNodeIdentity(t, outer.Children[0], "p", "é", 5, 10)
	assertNodeIdentity(t, outer.Children[1], "div", "x", 10, 22)
	if y := outer.Children[2]; y.Type != types.TextNode || y.Value != "y" || y.StartPos != 22 || y.EndPos != 23 {
		t.Fatalf("Expected y at 22:23, got %#v", y)
	}
	assertNodeIdentity(t, outer.Children[3], "p", "", 0, 0)
	assertNodeIdentity(t, outer.Children[4], "span", "z", 27, 41)
	text := outer.Children[0].Children[0]
	if text.StartPos != 8 || text.EndPos != 10 || text.StartLine != 1 || text.StartColumn != 9 || text.EndLine != 1 || text.EndColumn != 10 {
		t.Fatalf("Expected multibyte é at bytes 8:10 and parse5 columns 1:9-1:10, got %#v", text)
	}
}

func TestParseParagraphAutoClosePreservesMultibyteCRLFCoordinates(t *testing.T) {
	const content = "<p>é\r\n<div>y</div>"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 2 {
		t.Fatalf("Expected p and div siblings, got %#v", parsedBodyChildren(document))
	}
	paragraph := parsedBodyChildren(document)[0]
	assertNodeIdentity(t, paragraph, "p", "é\n", 0, 7)
	if len(paragraph.Children) != 1 {
		t.Fatalf("Expected one normalized paragraph text node, got %#v", paragraph.Children)
	}
	text := paragraph.Children[0]
	if text.Value != "é\n" || text.StartPos != 3 || text.EndPos != 7 {
		t.Fatalf("Expected normalized text at raw UTF-8 range 3:7, got %#v", text)
	}
	if text.StartLine != 1 || text.StartColumn != 4 || text.EndLine != 2 || text.EndColumn != 1 {
		t.Fatalf("Expected parse5 coordinates 1:4-2:1, got %d:%d-%d:%d", text.StartLine, text.StartColumn, text.EndLine, text.EndColumn)
	}
	assertNodeIdentity(t, parsedBodyChildren(document)[1], "div", "y", 7, len(content))
}

func TestParseIncompleteTagsAtEOFStayInParagraphSourceRange(t *testing.T) {
	testCases := []struct {
		name, content  string
		paragraphStart int
		textStart      int
		withSpan       bool
	}{
		{name: "incomplete div name", content: `<p>x<div`, paragraphStart: 0, textStart: 3},
		{name: "incomplete div attribute", content: `<p>x<div a="`, paragraphStart: 0, textStart: 3},
		{name: "incomplete p name", content: `<p>x<p`, paragraphStart: 0, textStart: 3},
		{name: "incomplete ancestor close name", content: `<div><p>x</div`, paragraphStart: 5, textStart: 8},
		{name: "incomplete ancestor close attribute", content: `<div><p>x</div `, paragraphStart: 5, textStart: 8},
		{name: "incomplete div after span", content: `<p><span>x<div`, paragraphStart: 0, textStart: 9, withSpan: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			paragraph := findFirstElementByName(document, "p")
			if paragraph == nil || paragraph.TextContent != "x" || paragraph.StartPos != testCase.paragraphStart || paragraph.EndPos != len(testCase.content) {
				t.Fatalf("Expected p x at %d:%d, got %#v", testCase.paragraphStart, len(testCase.content), paragraph)
			}
			textParent := paragraph
			if testCase.withSpan {
				if len(paragraph.Children) != 1 || paragraph.Children[0].Name != "span" {
					t.Fatalf("Expected one span child, got %#v", paragraph.Children)
				}
				textParent = paragraph.Children[0]
				if textParent.EndPos != len(testCase.content) {
					t.Fatalf("Expected span range through EOF %d, got %#v", len(testCase.content), textParent)
				}
			}
			if len(textParent.Children) != 1 {
				t.Fatalf("Expected one text child, got %#v", textParent.Children)
			}
			text := textParent.Children[0]
			if text.Value != "x" || text.StartPos != testCase.textStart || text.EndPos != len(testCase.content) {
				t.Fatalf("Expected x value with raw range %d:%d, got %#v", testCase.textStart, len(testCase.content), text)
			}
		})
	}
}

func TestParseHeadContextStrayParagraphEndTagsAreIgnored(t *testing.T) {
	testCases := []struct {
		name, content string
		wantRealP     bool
	}{
		{name: "after meta", content: `<meta></p>`},
		{name: "after title", content: `<title>x</title></p>`},
		{name: "after script", content: `<script>x</script></p>`},
		{name: "explicit head meta", content: `<html><head><meta></p></head><body><p>x</p></body></html>`, wantRealP: true},
		{name: "explicit head title", content: `<html><head><title>x</title></p></head><body></body></html>`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			paragraphs := findAllElementsByName(document, "p")
			if testCase.wantRealP {
				if len(paragraphs) != 1 || paragraphs[0].TextContent != "x" || paragraphs[0].StartPos != 35 || paragraphs[0].EndPos != 43 {
					t.Fatalf("Expected only the explicit body p at 35:43, got %#v", paragraphs)
				}
			} else if len(paragraphs) != 0 {
				t.Fatalf("Expected stray head-context p end to create no paragraph, got %#v", paragraphs)
			}
		})
	}
}

func TestParseMalformedBlockLikeTagNamesDoNotCloseParagraph(t *testing.T) {
	testCases := []struct {
		name, content, childName string
		childStart, childEnd     int
	}{
		{name: "equals suffix", content: `<p>x<div=y>z</div=y>w</p>`, childName: "div=y", childStart: 4, childEnd: 20},
		{name: "bang suffix", content: `<p>x<div!>z</div!>w</p>`, childName: "div!", childStart: 4, childEnd: 18},
		{name: "non-ASCII suffix", content: `<p>x<divé>z</divé>w</p>`, childName: "divé", childStart: 4, childEnd: 20},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" {
				t.Fatalf("Expected one p root, got %#v", parsedBodyChildren(document))
			}
			paragraph := parsedBodyChildren(document)[0]
			assertNodeIdentity(t, paragraph, "p", "xzw", 0, len(testCase.content))
			if len(paragraph.Children) != 3 {
				t.Fatalf("Expected x, custom child, w inside p, got %#v", paragraph.Children)
			}
			assertNodeIdentity(t, paragraph.Children[1], testCase.childName, "z", testCase.childStart, testCase.childEnd)
		})
	}
}

func TestParseParagraphEndTagContextTransitionsLikeBrowser(t *testing.T) {
	t.Run("inside otherwise empty html", func(t *testing.T) {
		const content = `<html></p></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		if paragraphs := findAllElementsByName(document, "p"); len(paragraphs) != 0 {
			t.Fatalf("Expected html-context p end to be ignored, got %#v", paragraphs)
		}
	})

	t.Run("block start exits head before p end", func(t *testing.T) {
		const content = `<html><head><div>x</div></p></head></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		div := findFirstElementByName(document, "div")
		assertNodeIdentity(t, div, "div", "x", 12, 24)
		paragraphs := findAllElementsByName(document, "p")
		if len(paragraphs) != 1 || paragraphs[0].TextContent != "" || paragraphs[0].StartPos != 0 || paragraphs[0].EndPos != 0 {
			t.Fatalf("Expected one synthetic body-like paragraph after head exit, got %#v", paragraphs)
		}
	})
}

func TestParseScopeBoundaryEndTagsCloseInnerParagraphs(t *testing.T) {
	for _, tag := range []string{"button", "object"} {
		t.Run(tag, func(t *testing.T) {
			content := `<` + tag + `><p>x</` + tag + `>`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != tag || len(parsedBodyChildren(document)[0].Children) != 1 {
				t.Fatalf("Expected %s with one p, got %#v", tag, parsedBodyChildren(document))
			}
			assertNodeIdentity(t, parsedBodyChildren(document)[0], tag, "x", 0, len(content))
			assertNodeIdentity(t, parsedBodyChildren(document)[0].Children[0], "p", "x", len(`<`+tag+`>`), len(`<`+tag+`><p>x`))
		})
	}

	t.Run("html end closes body paragraph", func(t *testing.T) {
		const content = `<html><body><p>x</html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		paragraph := findFirstElementByName(document, "p")
		assertNodeIdentity(t, paragraph, "p", "x", 12, len(content))
	})
}

func TestParseBodyCommittingTextControlsStrayParagraphEndRecovery(t *testing.T) {
	testCases := []struct {
		name, content, wantValue   string
		wantTextStart, wantTextEnd int
		wantSyntheticP             bool
	}{
		{name: "html literal", content: `<html>x</p></html>`, wantValue: "x", wantTextStart: 6, wantTextEnd: 7, wantSyntheticP: true},
		{name: "head literal", content: `<html><head>x</p></head></html>`, wantValue: "x", wantTextStart: 12, wantTextEnd: 13, wantSyntheticP: true},
		{name: "html decoded non-whitespace", content: `<html>&#65;</p></html>`, wantValue: "A", wantTextStart: 6, wantTextEnd: 11, wantSyntheticP: true},
		{name: "head decoded non-whitespace", content: `<html><head>&#65;</p></head></html>`, wantValue: "A", wantTextStart: 12, wantTextEnd: 17, wantSyntheticP: true},
		{name: "html literal whitespace", content: "<html> \n</p></html>"},
		{name: "head literal whitespace", content: "<html><head> \n</p></head></html>"},
		{name: "html decoded whitespace", content: `<html>&#32;</p></html>`},
		{name: "head decoded whitespace", content: `<html><head>&#32;</p></head></html>`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			paragraphs := findAllElementsByName(document, "p")
			if !testCase.wantSyntheticP {
				if len(paragraphs) != 0 {
					t.Fatalf("Expected whitespace-only head/body context to ignore p end, got %#v", paragraphs)
				}
				return
			}
			if len(paragraphs) != 1 || paragraphs[0].TextContent != "" || paragraphs[0].StartPos != 0 || paragraphs[0].EndPos != 0 {
				t.Fatalf("Expected one synthetic 0:0 p, got %#v", paragraphs)
			}
			texts := findAllTextNodes(document)
			if len(texts) != 1 || texts[0].Value != testCase.wantValue || texts[0].StartPos != testCase.wantTextStart || texts[0].EndPos != testCase.wantTextEnd {
				t.Fatalf("Expected committed body text %q at %d:%d, got %#v", testCase.wantValue, testCase.wantTextStart, testCase.wantTextEnd, texts)
			}
		})
	}
}
