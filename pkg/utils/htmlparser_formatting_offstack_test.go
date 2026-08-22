package utils

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseFormattingReconstructsOffStackEntryInsideOpenAncestor(t *testing.T) {
	const ordinary = `<b><p><i>x</p>y</b>`
	document, err := NewHTMLParser().Parse(ordinary)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")[0]
	paragraph := formattingElements(bold, "p")[0]
	italics := formattingElements(bold, "i")
	assertFormattingNode(t, bold, "b", "xy", 0, 19)
	assertFormattingNode(t, paragraph, "p", "x", 3, 14)
	if len(italics) != 2 {
		t.Fatalf("Expected off-stack i entry to reconstruct under still-open b, got %#v", italics)
	}
	assertFormattingNode(t, italics[0], "i", "x", 6, 10)
	assertFormattingNode(t, italics[1], "i", "y", 6, 15)
	if italics[0].Parent != paragraph || italics[1].Parent != bold {
		t.Fatalf("Expected original i in p and reconstructed i in b, got %#v", italics)
	}

	const cell = `<table><tr><td><b><p><i>x</p>y</table>`
	document, err = NewHTMLParser().Parse(cell)
	if err != nil {
		t.Fatal(err)
	}
	table := formattingElements(document, "table")[0]
	td := formattingElements(table, "td")[0]
	bold = formattingElements(td, "b")[0]
	paragraph = formattingElements(bold, "p")[0]
	italics = formattingElements(bold, "i")
	assertFormattingNode(t, table, "table", "xy", 0, 38)
	assertFormattingNode(t, td, "td", "xy", 11, 30)
	assertFormattingNode(t, bold, "b", "xy", 15, 30)
	assertFormattingNode(t, paragraph, "p", "x", 18, 29)
	if len(italics) != 2 {
		t.Fatalf("Expected i reconstruction within the same cell marker, got %#v", italics)
	}
	assertFormattingNode(t, italics[0], "i", "x", 21, 25)
	assertFormattingNode(t, italics[1], "i", "y", 21, 30)
	if italics[1].Parent != bold {
		t.Fatalf("Expected reconstructed i(y) under still-open b in cell, got %#v", italics[1])
	}
}

func TestParseFormattingOffStackCommentsDoNotReconstructOrLeakState(t *testing.T) {
	const content = `<p><b id=0><b id=1><b id=2><b id=3>x<div><!--c--><!--c--><!--c--><!--c--></div>`
	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	div := formattingElements(document, "div")[0]
	bold := formattingElements(document, "b")
	assertFormattingNode(t, paragraph, "p", "x", 0, 36)
	assertFormattingNode(t, div, "div", "", 36, 79)
	if len(bold) != 4 {
		t.Fatalf("Comment-only div must not reconstruct any of four off-stack b entries, got %#v", bold)
	}
	for i, start := range []int{3, 11, 19, 27} {
		assertFormattingNode(t, bold[i], "b", "x", start, 36)
		if bold[i].Attributes["id"] != fmt.Sprint(i) {
			t.Fatalf("Expected distinct b id=%d, got %#v", i, bold[i].Attributes)
		}
	}
	if len(div.Children) != 4 {
		t.Fatalf("Expected four comments and no reconstructed element in div, got %#v", div.Children)
	}
	for i, comment := range div.Children {
		wantStart := 41 + 8*i
		if comment.Type != types.CommentNode || comment.Value != "c" || comment.StartPos != wantStart || comment.EndPos != wantStart+8 {
			t.Fatalf("Expected comment %d at %d:%d, got %#v", i, wantStart, wantStart+8, comment)
		}
	}
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0] != paragraph || parsedBodyChildren(document)[1] != div {
		t.Fatalf("Expected sibling p and comment-only div, got %#v", parsedBodyChildren(document))
	}

	plain, err := parser.Parse(`z`)
	if err != nil || len(parsedBodyChildren(plain)) != 1 || parsedBodyChildren(plain)[0].Type != types.TextNode || parsedBodyChildren(plain)[0].Value != "z" || len(formattingElements(plain, "b")) != 0 {
		t.Fatalf("Parser reuse leaked off-stack active formatting entries: doc=%#v err=%v", plain, err)
	}
}

func TestParseFormattingOffStackCommentScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n*20 + 20)
		b.WriteString(`<p>`)
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, `<b id=%d>`, i)
		}
		b.WriteString(`x<div>`)
		for i := 0; i < n; i++ {
			b.WriteString(`<!--c-->`)
		}
		b.WriteString(`</div>`)
		return b.String()
	}
	measure := func(n int) time.Duration {
		content := build(n)
		_, _ = NewHTMLParser().Parse(content)
		best := time.Duration(1<<63 - 1)
		for i := 0; i < 3; i++ {
			start := time.Now()
			if _, err := NewHTMLParser().Parse(content); err != nil {
				t.Fatal(err)
			}
			if elapsed := time.Since(start); elapsed < best {
				best = elapsed
			}
		}
		return best
	}
	small, large := measure(2000), measure(8000)
	ratio := float64(large) / float64(small)
	t.Logf("off-stack comment scaling 2000=%v 8000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 30*time.Millisecond {
		t.Fatalf("Off-stack nontrigger handling scaled superlinearly: 2000=%v 8000=%v ratio=%.1fx", small, large, ratio)
	}
}

func TestParseFormattingReconstructedCloneTextAndComments(t *testing.T) {
	const content = `<p><b id=0><b id=1><b id=2><b id=3>x<div>y<!--c-->y<!--c-->y<!--c-->y<!--c--></div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	div := formattingElements(document, "div")[0]
	bold := formattingElements(document, "b")
	assertFormattingNode(t, paragraph, "p", "x", 0, 36)
	assertFormattingNode(t, div, "div", "yyyy", 36, 83)
	if len(bold) != 8 {
		t.Fatalf("Expected four original and four reconstructed b nodes, got %#v", bold)
	}
	for i, start := range []int{3, 11, 19, 27} {
		assertFormattingNode(t, bold[i], "b", "x", start, 36)
		assertFormattingNode(t, bold[i+4], "b", "yyyy", start, 77)
	}
	if bold[4].Parent != div || bold[5].Parent != bold[4] || bold[6].Parent != bold[5] || bold[7].Parent != bold[6] {
		t.Fatalf("Expected reconstructed b chain directly in div, got %#v", bold[4:])
	}
	deepest := bold[7]
	if len(deepest.Children) != 8 {
		t.Fatalf("Expected alternating y/comment tokens in deepest reconstructed b, got %#v", deepest.Children)
	}
	for i := 0; i < 4; i++ {
		text := deepest.Children[2*i]
		comment := deepest.Children[2*i+1]
		wantTextStart := 41 + 9*i
		wantCommentStart := 42 + 9*i
		if text.Type != types.TextNode || text.Value != "y" || text.StartPos != wantTextStart || text.EndPos != wantTextStart+1 {
			t.Fatalf("Expected y token %d at %d:%d, got %#v", i, wantTextStart, wantTextStart+1, text)
		}
		if comment.Type != types.CommentNode || comment.Value != "c" || comment.StartPos != wantCommentStart || comment.EndPos != wantCommentStart+8 {
			t.Fatalf("Expected comment %d at %d:%d, got %#v", i, wantCommentStart, wantCommentStart+8, comment)
		}
	}
}

func TestParseFormattingReconstructedCloneTextScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n*22 + 20)
		b.WriteString(`<p>`)
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, `<b id=%d>`, i)
		}
		b.WriteString(`x<div>`)
		for i := 0; i < n; i++ {
			b.WriteString(`y<!--c-->`)
		}
		b.WriteString(`</div>`)
		return b.String()
	}
	measure := func(n int) time.Duration {
		content := build(n)
		_, _ = NewHTMLParser().Parse(content)
		best := time.Duration(1<<63 - 1)
		for i := 0; i < 3; i++ {
			start := time.Now()
			if _, err := NewHTMLParser().Parse(content); err != nil {
				t.Fatal(err)
			}
			if elapsed := time.Since(start); elapsed < best {
				best = elapsed
			}
		}
		return best
	}
	small, large := measure(250), measure(1000)
	ratio := float64(large) / float64(small)
	t.Logf("reconstructed clone text scaling 250=%v 1000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 10*time.Millisecond {
		t.Fatalf("Reconstructed-clone TextContent updates scaled superlinearly: 250=%v 1000=%v ratio=%.1fx", small, large, ratio)
	}
}

func TestParseFormattingReconstructionTriggerMatrix(t *testing.T) {
	for _, testCase := range []struct {
		name, content, trigger         string
		bEnd, triggerStart, triggerEnd int
	}{
		{name: "whitespace character", content: `<p><b>x</p> y`, bEnd: 13},
		{name: "generic start", content: `<p><b>x</p><span>y</span>`, trigger: "span", bEnd: 25, triggerStart: 11, triggerEnd: 25},
		{name: "void img start", content: `<p><b>x</p><img>y`, trigger: "img", bEnd: 17, triggerStart: 11, triggerEnd: 16},
		{name: "xmp start", content: `<p><b>x</p><xmp>y</xmp>`, trigger: "xmp", bEnd: 23, triggerStart: 11, triggerEnd: 23},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatalf("Parse failed for %s: %v", testCase.name, err)
		}
		bold := formattingElements(document, "b")
		if len(bold) != 2 {
			t.Fatalf("Expected reconstruction for %s, got %#v", testCase.name, bold)
		}
		assertFormattingNode(t, bold[1], "b", map[string]string{"whitespace character": " y", "generic start": "y", "void img start": "y", "xmp start": "y"}[testCase.name], 3, testCase.bEnd)
		if testCase.trigger != "" {
			trigger := formattingElements(bold[1], testCase.trigger)[0]
			assertFormattingNode(t, trigger, testCase.trigger, map[string]string{"img": "", "span": "y", "xmp": "y"}[testCase.trigger], testCase.triggerStart, testCase.triggerEnd)
		}
	}

	const comment = `<p><b>x</p><!--c-->y`
	document, err := NewHTMLParser().Parse(comment)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	assertFormattingNode(t, bold[1], "b", "y", 3, 20)
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[1].Type != types.CommentNode || parsedBodyChildren(document)[1].Value != "c" || parsedBodyChildren(document)[1].StartPos != 11 || parsedBodyChildren(document)[1].EndPos != 19 || parsedBodyChildren(document)[2] != bold[1] {
		t.Fatalf("Expected comment before reconstructed b because comments do not trigger reconstruction, got %#v", parsedBodyChildren(document))
	}

	const block = `<p><b>x</p><div>y</div>`
	document, err = NewHTMLParser().Parse(block)
	if err != nil {
		t.Fatal(err)
	}
	div := formattingElements(document, "div")[0]
	bold = formattingElements(document, "b")
	assertFormattingNode(t, div, "div", "y", 11, 23)
	assertFormattingNode(t, bold[1], "b", "y", 3, 17)
	if bold[1].Parent != div {
		t.Fatalf("Block start itself must not reconstruct b; first character must reconstruct inside div, got %#v", bold[1])
	}

	const plaintext = `<p><b>x</p><plaintext>y`
	document, err = NewHTMLParser().Parse(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	plain := formattingElements(document, "plaintext")[0]
	bold = formattingElements(document, "b")
	assertFormattingNode(t, plain, "plaintext", "y", 11, 23)
	assertFormattingNode(t, bold[1], "b", "y", 3, 23)
	if bold[1].Parent != plain {
		t.Fatalf("Plaintext start must not reconstruct; first plaintext character must, got %#v", parsedBodyChildren(plain))
	}
}

func TestParseFormattingCommentAndDoctypeDoNotTriggerReconstruction(t *testing.T) {
	const comment = `<p><b>x</p><!--c-->y`
	document, err := NewHTMLParser().Parse(comment)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	assertFormattingNode(t, bold[1], "b", "y", 3, 20)
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[1].Type != types.CommentNode || parsedBodyChildren(document)[1].Value != "c" || parsedBodyChildren(document)[1].StartPos != 11 || parsedBodyChildren(document)[1].EndPos != 19 || parsedBodyChildren(document)[2] != bold[1] {
		t.Fatalf("Expected comment before reconstructed b because comments do not trigger reconstruction, got %#v", parsedBodyChildren(document))
	}

	const doctype = `<p><b>x</p><!DOCTYPE html>y`
	document, err = NewHTMLParser().Parse(doctype)
	if err != nil {
		t.Fatal(err)
	}
	bold = formattingElements(document, "b")
	assertFormattingNode(t, bold[1], "b", "y", 3, 27)
	if len(formattingElements(document, "html")) != 1 || len(findAllNodesByType(document, types.DocumentTypeNode)) != 0 || bold[1].Children[0].StartPos != 26 || bold[1].Children[0].EndPos != 27 {
		t.Fatalf("Expected ignored doctype not to trigger reconstruction before y, got %#v", parsedBodyChildren(document))
	}
}

func TestParseFormattingReconstructionUsesEmittedCharacterTokens(t *testing.T) {
	for _, testCase := range []struct {
		name, suffix string
		wantCount    int
		wantText     string
	}{
		{name: "literal NUL ignored", suffix: "\x00", wantCount: 1},
		{name: "comment does not trigger", suffix: `<!--c-->`, wantCount: 1},
		{name: "decoded NUL replacement triggers", suffix: `&#0;`, wantCount: 2, wantText: "�"},
		{name: "decoded whitespace triggers", suffix: `&#32;`, wantCount: 2, wantText: " "},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			content := `<p><b>x<div>` + testCase.suffix + `</div>`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			bold := formattingElements(document, "b")
			if len(bold) != testCase.wantCount {
				t.Fatalf("Expected %d b nodes for emitted-token case, got %#v", testCase.wantCount, bold)
			}
			if testCase.wantCount == 2 {
				div := formattingElements(document, "div")[0]
				if bold[1].Parent != div || bold[1].TextContent != testCase.wantText {
					t.Fatalf("Expected emitted character token %q to reconstruct b inside div, got %#v", testCase.wantText, bold[1])
				}
			}
		})
	}
}

func TestParseFormattingCloneMetadataIsIndependent(t *testing.T) {
	const content = `<p><b class=x>one<div>two</div>three`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	if len(bold) != 3 {
		t.Fatalf("Expected three b nodes for clone independence, got %#v", bold)
	}
	bold[1].Attributes["class"] = "mutated"
	bold[1].Attributes["new"] = "value"
	bold[1].AttributeOrder[0] = "new"
	if bold[0].Attributes["class"] != "x" || bold[2].Attributes["class"] != "x" {
		t.Fatalf("Mutating one clone's Attributes changed original/sibling: %#v", bold)
	}
	if _, exists := bold[0].Attributes["new"]; exists {
		t.Fatalf("Mutating one clone added attribute to original: %#v", bold[0].Attributes)
	}
	if bold[0].AttributeOrder[0] != "class" || bold[2].AttributeOrder[0] != "class" {
		t.Fatalf("Mutating one clone's AttributeOrder changed original/sibling: %#v", bold)
	}
}
