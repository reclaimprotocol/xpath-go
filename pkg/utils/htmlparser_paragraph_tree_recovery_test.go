package utils

import (
	"strings"
	"testing"
	"time"

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

func TestParseAncestorEndTagsUnwindParagraphAndButton(t *testing.T) {
	testCases := []struct {
		name, content, outerName         string
		outerEnd, buttonStart, buttonEnd int
		paragraphStart, paragraphEnd     int
		wantTail                         bool
	}{
		{
			name: "div", content: `<div><button><p>x</div>tail`, outerName: "div", outerEnd: 23,
			buttonStart: 5, buttonEnd: 17, paragraphStart: 13, paragraphEnd: 17, wantTail: true,
		},
		{
			name: "section", content: `<section><button><p>x</section>`, outerName: "section", outerEnd: 31,
			buttonStart: 9, buttonEnd: 21, paragraphStart: 17, paragraphEnd: 21,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) == 0 || parsedBodyChildren(document)[0].Name != testCase.outerName {
				t.Fatalf("Expected outer %s, got %#v", testCase.outerName, parsedBodyChildren(document))
			}
			outer := parsedBodyChildren(document)[0]
			assertNodeIdentity(t, outer, testCase.outerName, "x", 0, testCase.outerEnd)
			button := findFirstElementByName(outer, "button")
			assertNodeIdentity(t, button, "button", "x", testCase.buttonStart, testCase.buttonEnd)
			paragraph := findFirstElementByName(button, "p")
			assertNodeIdentity(t, paragraph, "p", "x", testCase.paragraphStart, testCase.paragraphEnd)
			if testCase.wantTail {
				if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].Type != types.TextNode || parsedBodyChildren(document)[1].Value != "tail" || parsedBodyChildren(document)[1].StartPos != 23 || parsedBodyChildren(document)[1].EndPos != 27 {
					t.Fatalf("Expected tail sibling at 23:27, got %#v", parsedBodyChildren(document))
				}
			}
		})
	}
}

func TestParseHTMLCloseExtendsAllPendingParagraphsToEOF(t *testing.T) {
	const content = `<html><body><p>a<button><p>b</html>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	paragraphs := findAllElementsByName(document, "p")
	if len(paragraphs) != 2 {
		t.Fatalf("Expected outer and inner paragraphs, got %#v", paragraphs)
	}
	assertNodeIdentity(t, paragraphs[0], "p", "ab", 12, 35)
	assertNodeIdentity(t, paragraphs[1], "p", "b", 24, 35)
	button := findFirstElementByName(document, "button")
	assertNodeIdentity(t, button, "button", "b", 16, 35)
	for _, paragraph := range paragraphs {
		if paragraph.EndLine != 1 || paragraph.EndColumn != 36 {
			t.Fatalf("Expected pending paragraph to end at browser EOF coordinate 1:36, got %#v", paragraph)
		}
	}
}

func TestParseBodyAndHTMLCloseKeepParagraphLocationPendingToEOF(t *testing.T) {
	testCases := []struct {
		name, content                  string
		htmlEnd, bodyEnd, paragraphEnd int
	}{
		{name: "body and html close", content: `<html><body><p>x</body></html>`, htmlEnd: 30, bodyEnd: 23, paragraphEnd: 30},
		{name: "comment after body and html", content: `<html><body><p>x</body></html><!--c-->`, htmlEnd: 30, bodyEnd: 23, paragraphEnd: 38},
		{name: "comment after html", content: `<html><body><p>x</html><!--c-->`, htmlEnd: 23, bodyEnd: 16, paragraphEnd: 31},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			html := findFirstElementByName(document, "html")
			body := findFirstElementByName(document, "body")
			paragraph := findFirstElementByName(document, "p")
			assertNodeIdentity(t, html, "html", "x", 0, testCase.htmlEnd)
			assertNodeIdentity(t, body, "body", "x", 6, testCase.bodyEnd)
			assertNodeIdentity(t, paragraph, "p", "x", 12, testCase.paragraphEnd)
			if paragraph.EndLine != 1 || paragraph.EndColumn != testCase.paragraphEnd+1 {
				t.Fatalf("Expected paragraph EOF coordinate 1:%d, got %#v", testCase.paragraphEnd+1, paragraph)
			}
			texts := findAllTextNodes(paragraph)
			if len(texts) != 1 || texts[0].Value != "x" || texts[0].StartPos != 15 || texts[0].EndPos != 16 {
				t.Fatalf("Expected x text at 15:16, got %#v", texts)
			}
		})
	}
}

func TestParseNULInAncestorTagNameStillClosesInnerParagraph(t *testing.T) {
	const content = "<div\x00><p>x</div\x00>"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 {
		t.Fatalf("Expected one recovered div ancestor, got %#v", parsedBodyChildren(document))
	}
	outer := parsedBodyChildren(document)[0]
	assertNodeIdentity(t, outer, "div�", "x", 0, 17)
	if len(outer.Children) != 1 {
		t.Fatalf("Expected one inner p, got %#v", outer.Children)
	}
	assertNodeIdentity(t, outer.Children[0], "p", "x", 6, 17)
}

func TestParseGenericAncestorEndTagIsIgnoredWhileParagraphIsOpen(t *testing.T) {
	testCases := []struct {
		name, content, ancestor   string
		paragraphStart, textStart int
	}{
		{name: "custom element", content: `<foo><p>x</foo>tail`, ancestor: "foo", paragraphStart: 5, textStart: 8},
		{name: "inline span", content: `<span><p>x</span>tail`, ancestor: "span", paragraphStart: 6, textStart: 9},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 {
				t.Fatalf("Expected one ancestor containing all continuation text, got %#v", parsedBodyChildren(document))
			}
			ancestor := parsedBodyChildren(document)[0]
			assertNodeIdentity(t, ancestor, testCase.ancestor, "xtail", 0, len(testCase.content))
			if len(ancestor.Children) != 1 {
				t.Fatalf("Expected ancestor to retain its inner p, got %#v", ancestor.Children)
			}
			paragraph := ancestor.Children[0]
			assertNodeIdentity(t, paragraph, "p", "xtail", testCase.paragraphStart, len(testCase.content))
			if len(paragraph.Children) != 1 {
				t.Fatalf("Expected ignored end token and tail to coalesce into one text node, got %#v", paragraph.Children)
			}
			text := paragraph.Children[0]
			if text.Type != types.TextNode || text.Value != "xtail" || text.StartPos != testCase.textStart || text.EndPos != len(testCase.content) {
				t.Fatalf("Expected coalesced xtail text at %d:%d, got %#v", testCase.textStart, len(testCase.content), text)
			}
		})
	}
}

func TestParsePendingParagraphResolvesOnFollowingBlockOrEOFText(t *testing.T) {
	t.Run("block after html close", func(t *testing.T) {
		const content = `<html><body><p>x</html><div>y</div>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		html := findFirstElementByName(document, "html")
		body := findFirstElementByName(document, "body")
		paragraph := findFirstElementByName(document, "p")
		div := findFirstElementByName(document, "div")
		assertNodeIdentity(t, paragraph, "p", "x", 12, 23)
		assertNodeIdentity(t, div, "div", "y", 23, 35)
		assertNodeIdentity(t, html, "html", "xy", 0, 23)
		assertNodeIdentity(t, body, "body", "xy", 6, 16)
		if paragraph.Parent != div.Parent {
			t.Fatalf("Expected p and following div to be body siblings, got p parent %#v and div parent %#v", paragraph.Parent, div.Parent)
		}
	})

	t.Run("block after body close", func(t *testing.T) {
		const content = `<html><body><p>x</body><div>y</div></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		html := findFirstElementByName(document, "html")
		body := findFirstElementByName(document, "body")
		paragraph := findFirstElementByName(document, "p")
		div := findFirstElementByName(document, "div")
		assertNodeIdentity(t, paragraph, "p", "x", 12, 23)
		assertNodeIdentity(t, div, "div", "y", 23, 35)
		assertNodeIdentity(t, html, "html", "xy", 0, 42)
		assertNodeIdentity(t, body, "body", "xy", 6, 23)
		if paragraph.Parent != div.Parent {
			t.Fatalf("Expected p and following div to be body siblings, got p parent %#v and div parent %#v", paragraph.Parent, div.Parent)
		}
	})

	t.Run("text after html close", func(t *testing.T) {
		const content = `<html><body><p>x</html>tail`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		html := findFirstElementByName(document, "html")
		body := findFirstElementByName(document, "body")
		paragraph := findFirstElementByName(document, "p")
		assertNodeIdentity(t, paragraph, "p", "xtail", 12, 27)
		if len(paragraph.Children) != 1 || paragraph.Children[0].Type != types.TextNode || paragraph.Children[0].Value != "xtail" || paragraph.Children[0].StartPos != 15 || paragraph.Children[0].EndPos != 27 {
			t.Fatalf("Expected one coalesced xtail text node at 15:27, got %#v", paragraph.Children)
		}
		assertNodeIdentity(t, html, "html", "xtail", 0, 23)
		assertNodeIdentity(t, body, "body", "xtail", 6, 16)
	})

	t.Run("whitespace after html close", func(t *testing.T) {
		const content = `<html><body><p>x</html>   `
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		html := findFirstElementByName(document, "html")
		body := findFirstElementByName(document, "body")
		paragraph := findFirstElementByName(document, "p")
		assertNodeIdentity(t, paragraph, "p", "x   ", 12, 26)
		if len(paragraph.Children) != 1 || paragraph.Children[0].Type != types.TextNode || paragraph.Children[0].Value != "x   " || paragraph.Children[0].StartPos != 15 || paragraph.Children[0].EndPos != 26 {
			t.Fatalf("Expected one coalesced whitespace continuation text node at 15:26, got %#v", paragraph.Children)
		}
		assertNodeIdentity(t, html, "html", "x   ", 0, 23)
		assertNodeIdentity(t, body, "body", "x   ", 6, 16)
	})
}

func TestParsePendingParagraphCommentAndStackResolution(t *testing.T) {
	t.Run("comments and processing instructions defer until block", func(t *testing.T) {
		for _, tc := range []struct {
			declaration string
			nodeType    types.NodeType
			name, value string
		}{{`<!foo>`, types.CommentNode, "#comment", "foo"}, {`<?foo>`, types.ProcessingInstructionNode, "foo", ""}, {`</$foo>`, types.CommentNode, "#comment", "$foo"}} {
			content := `<html><body><p>x</html>` + tc.declaration + `<div>y</div>`
			blockStart := 23 + len(tc.declaration)
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatalf("Parse %q: %v", tc.declaration, err)
			}
			assertNodeIdentity(t, findFirstElementByName(document, "p"), "p", "x", 12, blockStart)
			assertNodeIdentity(t, findFirstElementByName(document, "div"), "div", "y", blockStart, len(content))
			nodes := findAllNodesByType(document, tc.nodeType)
			if len(nodes) != 1 || nodes[0].Name != tc.name || nodes[0].Value != tc.value || nodes[0].StartPos != 23 || nodes[0].EndPos != blockStart || nodes[0].Parent != document {
				t.Fatalf("Expected document node %s=%q at 23:%d, got %#v", tc.name, tc.value, blockStart, nodes)
			}
		}
	})
	t.Run("explicit p end", func(t *testing.T) {
		for _, content := range []string{`<html><body><p>x</html></p>tail`, `<html><body><p>x</body></p>tail</html>`} {
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			ps := findAllElementsByName(document, "p")
			if len(ps) != 1 {
				t.Fatalf("Expected one p, got %#v", ps)
			}
			assertNodeIdentity(t, ps[0], "p", "x", 12, 27)
			texts := findAllTextNodes(document)
			if len(texts) != 2 || texts[1].Value != "tail" || texts[1].StartPos != 27 || texts[1].EndPos != 31 || texts[1].Parent != ps[0].Parent {
				t.Fatalf("Expected body tail at 27:31, got %#v", texts)
			}
		}
	})
	t.Run("ordered pending stack", func(t *testing.T) {
		const base = `<html><body><p>a<button><p>b</html>`
		for _, tc := range []struct {
			name, suffix, outerText, buttonText string
			outerEnd, buttonEnd, innerEnd       int
		}{
			{"block", `<div>y</div>`, "aby", "by", 47, 47, 35},
			{"p end", `</p>tail`, "abtail", "btail", 43, 43, 39},
			{"button end", `</button>tail`, "abtail", "b", 48, 44, 35},
		} {
			document, err := NewHTMLParser().Parse(base + tc.suffix)
			if err != nil {
				t.Fatal(err)
			}
			ps := findAllElementsByName(document, "p")
			if len(ps) != 2 {
				t.Fatalf("%s: expected two p nodes, got %#v", tc.name, ps)
			}
			assertNodeIdentity(t, ps[0], "p", tc.outerText, 12, tc.outerEnd)
			assertNodeIdentity(t, ps[1], "p", "b", 24, tc.innerEnd)
			button := findFirstElementByName(document, "button")
			assertNodeIdentity(t, button, "button", tc.buttonText, 16, tc.buttonEnd)
			if tc.name == "block" {
				div := findFirstElementByName(document, "div")
				assertNodeIdentity(t, div, "div", "y", 35, 47)
				if div.Parent != button {
					t.Fatalf("Expected div inside button")
				}
			}
		}
	})
	t.Run("inline descendants stay pending", func(t *testing.T) {
		for _, tc := range []struct {
			content, value string
			end            int
		}{{`<html><body><p><span>x</html>tail`, "xtail", 33}, {`<html><body><p><span>x</html><!--c-->`, "x", 37}} {
			document, err := NewHTMLParser().Parse(tc.content)
			if err != nil {
				t.Fatal(err)
			}
			assertNodeIdentity(t, findFirstElementByName(document, "p"), "p", tc.value, 12, tc.end)
			assertNodeIdentity(t, findFirstElementByName(document, "span"), "span", tc.value, 15, tc.end)
		}
	})
}

func TestParsePendingParagraphContentRangesAndCommentReentry(t *testing.T) {
	t.Run("tail content ranges", func(t *testing.T) {
		const content = `<html><body><p><span>x</html>tail`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		p, span := findFirstElementByName(document, "p"), findFirstElementByName(document, "span")
		if p.ContentStart != 15 || p.ContentEnd != 33 || span.ContentStart != 21 || span.ContentEnd != 33 {
			t.Fatalf("Expected p content 15:33 and span 21:33, got p %#v span %#v", p, span)
		}
	})
	t.Run("comment EOF content ranges", func(t *testing.T) {
		const content = `<html><body><p><span>x</html><!--c-->`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		p, span := findFirstElementByName(document, "p"), findFirstElementByName(document, "span")
		if p.ContentEnd != 37 || span.ContentEnd != 37 {
			t.Fatalf("Expected pending content ranges through comment EOF, got p %#v span %#v", p, span)
		}
	})
	t.Run("multiline explicit close", func(t *testing.T) {
		const content = "<html>\n<body><p><span>x</html>\n</p>tail"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		p, span := findFirstElementByName(document, "p"), findFirstElementByName(document, "span")
		assertNodeIdentity(t, p, "p", "x\n", 13, 35)
		assertNodeIdentity(t, span, "span", "x\n", 16, 31)
		if p.ContentStart != 16 || p.ContentEnd != 31 || span.ContentStart != 22 || span.ContentEnd != 31 || span.EndLine != 3 || span.EndColumn != 1 || p.EndLine != 3 || p.EndColumn != 5 {
			t.Fatalf("Expected implied span endpoint 3:1 and matched p endpoint 3:5, got p %#v span %#v", p, span)
		}
	})
	t.Run("explicit close retains whitespace", func(t *testing.T) {
		const content = `<html><body><p>x</html></p>   `
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		p := findFirstElementByName(document, "p")
		assertNodeIdentity(t, p, "p", "x", 12, 27)
		texts := findAllTextNodes(document)
		if len(texts) != 2 || texts[1].Value != "   " || texts[1].StartPos != 27 || texts[1].EndPos != 30 || texts[1].Parent != p.Parent {
			t.Fatalf("Expected body whitespace at 27:30, got %#v", texts)
		}
	})
	t.Run("comments reenter body after resolution", func(t *testing.T) {
		for _, tc := range []struct {
			content, wantText string
			pEnd, secondStart int
		}{
			{`<html><body><p>x</html><!--a--></p><!--b-->`, "x", 35, 35},
			{`<html><body><p>x</html><!--a--><div>y</div><!--b-->`, "xy", 31, 43},
		} {
			document, err := NewHTMLParser().Parse(tc.content)
			if err != nil {
				t.Fatal(err)
			}
			p := findFirstElementByName(document, "p")
			assertNodeIdentity(t, p, "p", "x", 12, tc.pEnd)
			comments := findAllNodesByType(document, types.CommentNode)
			var first, second *types.Node
			for _, comment := range comments {
				if comment.Value == "a" {
					first = comment
				}
				if comment.Value == "b" {
					second = comment
				}
			}
			if len(comments) != 2 || first == nil || first.StartPos != 23 || first.EndPos != 31 || first.Parent != document || second == nil || second.StartPos != tc.secondStart || second.EndPos != tc.secondStart+8 || second.Parent != p.Parent {
				t.Fatalf("Expected document a then body b comments, got %#v", comments)
			}
			html, body := findFirstElementByName(document, "html"), findFirstElementByName(document, "body")
			if html.TextContent != tc.wantText || body.TextContent != tc.wantText {
				t.Fatalf("Expected comments excluded from html/body text %q, got html=%q body=%q", tc.wantText, html.TextContent, body.TextContent)
			}
		}
	})
}

func TestParseSpecialAncestorEndTagsUnwindOpenParagraph(t *testing.T) {
	for _, tag := range []string{"form", "li", "dd", "dt", "h1", "h2", "h3", "h4", "h5", "h6", "applet", "marquee"} {
		t.Run(tag, func(t *testing.T) {
			opening, closing := `<`+tag+`>`, `</`+tag+`>`
			content := opening + `<p>x` + closing + `tail`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			if len(parsedBodyChildren(document)) != 2 {
				t.Fatalf("Expected closed ancestor plus tail, got %#v", parsedBodyChildren(document))
			}
			ancestor := parsedBodyChildren(document)[0]
			assertNodeIdentity(t, ancestor, tag, "x", 0, len(content)-4)
			p := findFirstElementByName(ancestor, "p")
			assertNodeIdentity(t, p, "p", "x", len(opening), len(opening)+4)
			tail := parsedBodyChildren(document)[1]
			if tail.Type != types.TextNode || tail.Value != "tail" || tail.StartPos != len(content)-4 || tail.EndPos != len(content) {
				t.Fatalf("Expected tail outside %s at %d:%d, got %#v", tag, len(content)-4, len(content), tail)
			}
		})
	}
}

func findAllTextNodes(node *types.Node) []*types.Node {
	var matches []*types.Node
	if node == nil {
		return matches
	}
	if node.Type == types.TextNode {
		matches = append(matches, node)
	}
	for _, child := range node.Children {
		matches = append(matches, findAllTextNodes(child)...)
	}
	return matches
}

func findAllNodesByType(node *types.Node, nodeType types.NodeType) []*types.Node {
	var matches []*types.Node
	if node == nil {
		return matches
	}
	if node.Type == nodeType {
		matches = append(matches, node)
	}
	for _, child := range node.Children {
		matches = append(matches, findAllNodesByType(child, nodeType)...)
	}
	return matches
}

func findAllElementsByName(node *types.Node, name string) []*types.Node {
	var matches []*types.Node
	if node == nil {
		return matches
	}
	if node.Type == types.ElementNode && node.Name == name {
		matches = append(matches, node)
	}
	for _, child := range node.Children {
		matches = append(matches, findAllElementsByName(child, name)...)
	}
	return matches
}

func TestParseDeepInlineParagraphLookupScalesNearLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping parser scaling regression in short mode")
	}
	build := func(depth int, paragraph bool) string {
		var content strings.Builder
		if paragraph {
			content.WriteString(`<p>`)
		}
		content.WriteString(strings.Repeat(`<span>`, depth))
		content.WriteString(`x`)
		content.WriteString(strings.Repeat(`</span>`, depth))
		if paragraph {
			content.WriteString(`</p>`)
		}
		return content.String()
	}
	measure := func(depth int, paragraph bool) time.Duration {
		content := build(depth, paragraph)
		started := time.Now()
		document, err := NewHTMLParser().Parse(content)
		elapsed := time.Since(started)
		if err != nil || document.TextContent != "" {
			// Document nodes do not aggregate TextContent; only the error matters.
			if err != nil {
				t.Fatalf("Parse depth %d paragraph=%v failed: %v", depth, paragraph, err)
			}
		}
		return elapsed
	}
	minOfTwo := func(depth int, paragraph bool) time.Duration {
		first, second := measure(depth, paragraph), measure(depth, paragraph)
		if first < second {
			return first
		}
		return second
	}

	_ = measure(64, true)
	for _, paragraph := range []bool{false, true} {
		paragraph := paragraph
		name := "without paragraph"
		if paragraph {
			name = "with paragraph"
		}
		t.Run(name, func(t *testing.T) {
			small := minOfTwo(600, paragraph)
			large := minOfTwo(2400, paragraph)
			t.Logf("depth 600 in %s, depth 2400 in %s", small, large)
			if large > 10*small && large-small > 20*time.Millisecond {
				t.Fatalf("Paragraph scope lookup scales quadratically: 4x depth took %.1fx longer", float64(large)/float64(small))
			}
		})
	}
}

func TestParseDeepInlineLongTagInspectionHasBoundedScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping adversarial parser scaling regression in short mode")
	}
	build := func(depth, attributeLength int) string {
		return `<p>` + strings.Repeat(`<span>`, depth) + `x<div data-x="` + strings.Repeat("a", attributeLength) + `">y</div>`
	}
	measure := func(depth, attributeLength int) time.Duration {
		content := build(depth, attributeLength)
		started := time.Now()
		_, err := NewHTMLParser().Parse(content)
		elapsed := time.Since(started)
		if err != nil {
			t.Fatalf("Parse depth=%d attr=%d failed: %v", depth, attributeLength, err)
		}
		return elapsed
	}
	minOfTwo := func(depth, attributeLength int) time.Duration {
		first, second := measure(depth, attributeLength), measure(depth, attributeLength)
		if first < second {
			return first
		}
		return second
	}
	_ = measure(32, 128)
	small := minOfTwo(250, 12000)
	large := minOfTwo(1000, 48000)
	t.Logf("depth/attribute 250/12k in %s and 1000/48k in %s", small, large)
	if large > 12*small && large-small > 30*time.Millisecond {
		t.Fatalf("Deep paragraph tag inspection scales superlinearly: 4x depth and attribute took %.1fx longer", float64(large)/float64(small))
	}
}
