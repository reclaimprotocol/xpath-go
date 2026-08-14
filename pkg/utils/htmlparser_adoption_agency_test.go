package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 14B begins with the core adoption-agency algorithm for misnested
// b/i/em/strong ends. Anchors, nobr, table integration, formatting markers,
// and the broader active-formatting surgery matrix are intentionally deferred.

func TestParseAdoptionAgencyCanonicalMisnestedFormatting(t *testing.T) {
	const content = `<p><b>1<i>2</b>3</i>4`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	bold := formattingElements(paragraph, "b")
	italics := formattingElements(paragraph, "i")
	assertFormattingNode(t, paragraph, "p", "1234", 0, 21)
	if len(bold) != 1 || len(italics) != 2 {
		t.Fatalf("Expected b(1,i(2)), reconstructed i(3), and plain 4, b=%#v i=%#v", bold, italics)
	}
	assertFormattingNode(t, bold[0], "b", "12", 3, 15)
	assertFormattingNode(t, italics[0], "i", "2", 7, 11)
	assertFormattingNode(t, italics[1], "i", "3", 7, 20)
	if italics[0].Parent != bold[0] || italics[1].Parent != paragraph || len(paragraph.Children) != 3 || paragraph.Children[2].Type != types.TextNode || paragraph.Children[2].Value != "4" || paragraph.Children[2].StartPos != 20 || paragraph.Children[2].EndPos != 21 {
		t.Fatalf("Unexpected canonical adoption tree: %#v", paragraph.Children)
	}
}

func TestParseAdoptionAgencyFurthestBlockReparentsFormatting(t *testing.T) {
	const content = `<b>1<div>2</b>3</div>4`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	div := formattingElements(document, "div")[0]
	if len(bold) != 2 || len(parsedBodyChildren(document)) != 3 {
		t.Fatalf("Expected source b, div containing synthetic b, and trailing 4, got b=%#v children=%#v", bold, parsedBodyChildren(document))
	}
	assertFormattingNode(t, bold[0], "b", "1", 0, 14)
	assertFormattingNode(t, div, "div", "23", 4, 21)
	assertFormattingNode(t, bold[1], "b", "2", 0, 0)
	if bold[1].Parent != div || len(div.Children) != 2 || div.Children[0] != bold[1] || div.Children[1].Type != types.TextNode || div.Children[1].Value != "3" || div.Children[1].StartPos != 14 || div.Children[1].EndPos != 15 {
		t.Fatalf("Expected furthest block to adopt b(2) then plain 3, got %#v", div.Children)
	}
	if parsedBodyChildren(document)[0] != bold[0] || parsedBodyChildren(document)[1] != div || parsedBodyChildren(document)[2].Type != types.TextNode || parsedBodyChildren(document)[2].Value != "4" || parsedBodyChildren(document)[2].StartPos != 21 || parsedBodyChildren(document)[2].EndPos != 22 {
		t.Fatalf("Unexpected furthest-block sibling order: %#v", parsedBodyChildren(document))
	}
}

func TestParseAdoptionAgencyAbsentAndOutOfScopeEnds(t *testing.T) {
	const absent = `<p>x</b>y`
	document, err := NewHTMLParser().Parse(absent)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	assertFormattingNode(t, paragraph, "p", "xy", 0, 9)
	if len(formattingElements(document, "b")) != 0 || len(paragraph.Children) != 1 || paragraph.Children[0].Type != types.TextNode || paragraph.Children[0].Value != "xy" || paragraph.Children[0].StartPos != 3 || paragraph.Children[0].EndPos != 9 {
		t.Fatalf("Absent formatting end must be ignored and adjacent text coalesced across its raw range, got %#v", paragraph.Children)
	}

	const outOfScope = `<b><button>x</b>y</button>z`
	document, err = NewHTMLParser().Parse(outOfScope)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	button := formattingElements(document, "button")[0]
	if len(bold) != 2 || len(parsedBodyChildren(document)) != 3 {
		t.Fatalf("Expected empty source b, button with reconstructed b(x)+y, and z, got b=%#v children=%#v", bold, parsedBodyChildren(document))
	}
	assertFormattingNode(t, bold[0], "b", "", 0, 16)
	assertFormattingNode(t, button, "button", "xy", 3, 26)
	assertFormattingNode(t, bold[1], "b", "x", 0, 0)
	if bold[1].Parent != button || len(button.Children) != 2 || button.Children[1].Type != types.TextNode || button.Children[1].Value != "y" || button.Children[1].StartPos != 16 || button.Children[1].EndPos != 17 {
		t.Fatalf("Out-of-scope b end must be ignored behind button scope, got %#v", button.Children)
	}
	if parsedBodyChildren(document)[2].Type != types.TextNode || parsedBodyChildren(document)[2].Value != "z" || parsedBodyChildren(document)[2].StartPos != 26 || parsedBodyChildren(document)[2].EndPos != 27 {
		t.Fatalf("Expected z after button, got %#v", parsedBodyChildren(document))
	}
}

func TestParseAdoptionAgencyEOFAndParserReuse(t *testing.T) {
	const content = `<p><strong>a<em>b</strong>c`
	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	strong := formattingElements(paragraph, "strong")
	emphasis := formattingElements(paragraph, "em")
	assertFormattingNode(t, paragraph, "p", "abc", 0, 27)
	assertFormattingNode(t, strong[0], "strong", "ab", 3, 26)
	assertFormattingNode(t, emphasis[0], "em", "b", 12, 17)
	assertFormattingNode(t, emphasis[1], "em", "c", 12, 27)
	if emphasis[0].Parent != strong[0] || emphasis[1].Parent != paragraph {
		t.Fatalf("Expected em reconstruction after misnested strong end, got %#v", emphasis)
	}

	plain, err := parser.Parse(`z`)
	if err != nil || len(parsedBodyChildren(plain)) != 1 || parsedBodyChildren(plain)[0].Type != types.TextNode || parsedBodyChildren(plain)[0].Value != "z" || len(formattingElements(plain, "b")) != 0 || len(formattingElements(plain, "i")) != 0 || len(formattingElements(plain, "em")) != 0 || len(formattingElements(plain, "strong")) != 0 {
		t.Fatalf("Parser reuse leaked adoption-agency state: doc=%#v err=%v", plain, err)
	}
}

func TestParseAdoptionAgencyMultibyteLocations(t *testing.T) {
	const content = "<p><b>é\r\n<i>😀</b>z</i>w"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	bold := formattingElements(paragraph, "b")[0]
	italics := formattingElements(paragraph, "i")
	assertFormattingNode(t, paragraph, "p", "é\n😀zw", 0, 27)
	assertFormattingNode(t, bold, "b", "é\n😀", 3, 21)
	assertFormattingNode(t, italics[0], "i", "😀", 10, 17)
	assertFormattingNode(t, italics[1], "i", "z", 10, 26)
	if bold.StartLine != 1 || bold.StartColumn != 4 || bold.EndLine != 2 || bold.EndColumn != 10 || italics[0].StartLine != 2 || italics[0].StartColumn != 1 || italics[0].EndLine != 2 || italics[0].EndColumn != 6 || italics[1].EndLine != 2 || italics[1].EndColumn != 15 {
		t.Fatalf("Expected original UTF-8 bytes with parse5 UTF-16 line/columns, b=%#v i=%#v", bold, italics)
	}
}

func TestParseAdoptionAgencyScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n * 24)
		for i := 0; i < n; i++ {
			b.WriteString(`<p><b>1<i>2</b>3</i>4`)
		}
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
	small, large := measure(1000), measure(4000)
	ratio := float64(large) / float64(small)
	t.Logf("adoption-agency scaling 1000=%v 4000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 20*time.Millisecond {
		t.Fatalf("Adoption-agency recovery scaled superlinearly: 1000=%v 4000=%v ratio=%.1fx", small, large, ratio)
	}
}

func TestParseAdoptionAgencySurvivingFormattingDescendant(t *testing.T) {
	const content = `<p><b>1<div><i>2</b>3</i>4</div>5`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	div := formattingElements(document, "div")[0]
	bold := formattingElements(document, "b")
	italics := formattingElements(document, "i")
	assertFormattingNode(t, paragraph, "p", "1", 0, 7)
	assertFormattingNode(t, bold[0], "b", "1", 3, 7)
	assertFormattingNode(t, div, "div", "234", 7, 32)
	assertFormattingNode(t, bold[1], "b", "2", 3, 20)
	assertFormattingNode(t, italics[0], "i", "2", 12, 16)
	assertFormattingNode(t, italics[1], "i", "3", 12, 25)
	if bold[1].Parent != div || italics[0].Parent != bold[1] || italics[1].Parent != div || len(div.Children) != 3 || div.Children[2].Value != "4" || div.Children[2].StartPos != 25 || div.Children[2].EndPos != 26 {
		t.Fatalf("Expected surviving i entry to reconstruct after b adoption, got %#v", div.Children)
	}
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[2].Value != "5" || parsedBodyChildren(document)[2].StartPos != 32 || parsedBodyChildren(document)[2].EndPos != 33 {
		t.Fatalf("Expected trailing 5 outside furthest block, got %#v", parsedBodyChildren(document))
	}
}

func TestParseAdoptionAgencyOrdinarySpanIsNotFurthestBlock(t *testing.T) {
	const content = `<p><b>1<span>2</b>3</span>4`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	bold := formattingElements(paragraph, "b")[0]
	span := formattingElements(bold, "span")[0]
	assertFormattingNode(t, paragraph, "p", "1234", 0, 27)
	assertFormattingNode(t, bold, "b", "12", 3, 18)
	assertFormattingNode(t, span, "span", "2", 7, 14)
	if len(formattingElements(document, "span")) != 1 || len(paragraph.Children) != 2 || paragraph.Children[1].Type != types.TextNode || paragraph.Children[1].Value != "34" || paragraph.Children[1].StartPos != 18 || paragraph.Children[1].EndPos != 27 {
		t.Fatalf("Ordinary span must be popped with b rather than treated as furthest block, got %#v", paragraph.Children)
	}
}

func TestParseAdoptionAgencyDeepBookmarkAndNestedBlocks(t *testing.T) {
	const deep = `<p><b><em><i>1<div>2</b>3</i>4</em>5</div>6`
	document, err := NewHTMLParser().Parse(deep)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	div := formattingElements(document, "div")[0]
	bold := formattingElements(document, "b")
	emphasis := formattingElements(document, "em")
	italics := formattingElements(document, "i")
	assertFormattingNode(t, paragraph, "p", "1", 0, 14)
	assertFormattingNode(t, bold[0], "b", "1", 3, 14)
	assertFormattingNode(t, emphasis[0], "em", "1", 6, 14)
	assertFormattingNode(t, italics[0], "i", "1", 10, 14)
	assertFormattingNode(t, div, "div", "2345", 14, 42)
	assertFormattingNode(t, bold[1], "b", "2", 3, 24)
	assertFormattingNode(t, emphasis[1], "em", "2", 6, 20)
	assertFormattingNode(t, italics[1], "i", "2", 10, 20)
	assertFormattingNode(t, emphasis[2], "em", "34", 6, 35)
	assertFormattingNode(t, italics[2], "i", "3", 10, 29)
	if bold[1].Parent != div || emphasis[1].Parent != bold[1] || emphasis[2].Parent != div || italics[2].Parent != emphasis[2] || div.Children[2].Value != "5" {
		t.Fatalf("Deep bookmark replacement produced wrong hierarchy: %#v", div.Children)
	}
	if parsedBodyChildren(document)[2].Value != "6" || parsedBodyChildren(document)[2].StartPos != 42 || parsedBodyChildren(document)[2].EndPos != 43 {
		t.Fatalf("Expected 6 after furthest block, got %#v", parsedBodyChildren(document))
	}

	const nested = `<b>1<div>2<div>3</b>4</div>5</div>`
	document, err = NewHTMLParser().Parse(nested)
	if err != nil {
		t.Fatal(err)
	}
	bold = formattingElements(document, "b")
	divs := formattingElements(document, "div")
	if len(bold) != 3 || len(divs) != 2 {
		t.Fatalf("Expected source b plus one replacement in each nested block, b=%#v div=%#v", bold, divs)
	}
	assertFormattingNode(t, bold[0], "b", "1", 0, 20)
	assertFormattingNode(t, divs[0], "div", "2345", 4, 34)
	assertFormattingNode(t, bold[1], "b", "2", 0, 0)
	assertFormattingNode(t, divs[1], "div", "34", 10, 27)
	assertFormattingNode(t, bold[2], "b", "3", 0, 0)
	if bold[1].Parent != divs[0] || divs[1].Parent != divs[0] || bold[2].Parent != divs[1] {
		t.Fatalf("Nested furthest blocks produced wrong hierarchy: %#v", divs)
	}
}

func TestParseAdoptionAgencyAbsentRepeatedEndsAndComments(t *testing.T) {
	const absent = `<p><b>x</strong>y</b>z</b>w`
	document, err := NewHTMLParser().Parse(absent)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	bold := formattingElements(paragraph, "b")[0]
	assertFormattingNode(t, paragraph, "p", "xyzw", 0, 27)
	assertFormattingNode(t, bold, "b", "xy", 3, 21)
	if len(paragraph.Children) != 2 || paragraph.Children[0] != bold || paragraph.Children[1].Type != types.TextNode || paragraph.Children[1].Value != "zw" || paragraph.Children[1].StartPos != 21 || paragraph.Children[1].EndPos != 27 {
		t.Fatalf("Absent strong and repeated b ends must be ignored without synthetic formatting, got %#v", paragraph.Children)
	}

	const comments = `<b>1<div><!--a-->2</b><!--b-->3</div>4`
	document, err = NewHTMLParser().Parse(comments)
	if err != nil {
		t.Fatal(err)
	}
	bolds := formattingElements(document, "b")
	div := formattingElements(document, "div")[0]
	assertFormattingNode(t, bolds[0], "b", "1", 0, 22)
	assertFormattingNode(t, div, "div", "23", 4, 37)
	assertFormattingNode(t, bolds[1], "b", "2", 0, 0)
	if len(bolds[1].Children) != 2 || bolds[1].Children[0].Type != types.CommentNode || bolds[1].Children[0].Value != "a" || bolds[1].Children[0].StartPos != 9 || bolds[1].Children[0].EndPos != 17 || bolds[1].Children[1].Value != "2" {
		t.Fatalf("Expected comment a and text 2 to move with replacement b, got %#v", bolds[1].Children)
	}
	if len(div.Children) != 3 || div.Children[1].Type != types.CommentNode || div.Children[1].Value != "b" || div.Children[1].StartPos != 22 || div.Children[1].EndPos != 30 || div.Children[2].Value != "3" {
		t.Fatalf("Expected comment b and text 3 outside replacement b, got %#v", div.Children)
	}
}

func TestParseAdoptionAgencyAttributeCloneMultibyteAndIncompleteEnd(t *testing.T) {
	const multibyte = "<p><b title=x>é\r\n<div>😀</b>z</div>w"
	document, err := NewHTMLParser().Parse(multibyte)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	div := formattingElements(document, "div")[0]
	bold := formattingElements(document, "b")
	assertFormattingNode(t, paragraph, "p", "é\n", 0, 18)
	assertFormattingNode(t, bold[0], "b", "é\n", 3, 18)
	assertFormattingNode(t, div, "div", "😀z", 18, 38)
	assertFormattingNode(t, bold[1], "b", "😀", 3, 31)
	if bold[1].Attributes["title"] != "x" || bold[1].ContentStart != 14 || len(bold[1].AttributeOrder) != 1 || bold[1].AttributeOrder[0] != "title" || bold[1].Parent != div {
		t.Fatalf("Expected reconstructed attribute clone under div, got %#v", bold[1])
	}
	if bold[1].StartLine != 1 || bold[1].StartColumn != 4 || bold[1].EndLine != 2 || bold[1].EndColumn != 12 || div.StartLine != 2 || div.StartColumn != 1 || div.EndLine != 2 || div.EndColumn != 19 {
		t.Fatalf("Expected raw UTF-8 offsets and parse5 UTF-16 coordinates, b=%#v div=%#v", bold[1], div)
	}

	const incomplete = `<p><b>x</b`
	document, err = NewHTMLParser().Parse(incomplete)
	if err != nil {
		t.Fatal(err)
	}
	paragraph = formattingElements(document, "p")[0]
	bold = formattingElements(document, "b")
	assertFormattingNode(t, paragraph, "p", "x", 0, 10)
	assertFormattingNode(t, bold[0], "b", "x", 3, 10)
	if len(bold) != 1 || len(bold[0].Children) != 1 || bold[0].Children[0].Value != "x" || bold[0].Children[0].StartPos != 6 || bold[0].Children[0].EndPos != 10 {
		t.Fatalf("Incomplete formatting end must be discarded without running adoption, got %#v", bold)
	}
}

func TestParseAdoptionAgencyCurrentInnerLoopLimit(t *testing.T) {
	const content = `<b><em><foo><foo><foo><aside></b></em>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	outerBold := formattingElements(document, "b")[0]
	emphasis := formattingElements(outerBold, "em")[0]
	foos := formattingElements(emphasis, "foo")
	aside := formattingElements(document, "aside")[0]
	asideBold := formattingElements(aside, "b")
	assertFormattingNode(t, outerBold, "b", "", 0, 33)
	assertFormattingNode(t, emphasis, "em", "", 3, 29)
	assertFormattingNode(t, aside, "aside", "", 22, 38)
	if emphasis.EndLine != 1 || emphasis.EndColumn != 30 {
		t.Fatalf("Emphasis beyond the third inner-loop node must end at adoption token start 29 (line 1 col 30), got %#v", emphasis)
	}
	if len(foos) != 3 || len(asideBold) != 1 || asideBold[0].StartPos != 0 || asideBold[0].EndPos != 0 || asideBold[0].Parent != aside || len(formattingElements(aside, "em")) != 0 {
		t.Fatalf("Current inner-loop rule must leave aside with empty b, not em>b, foo=%#v aside=%#v", foos, aside.Children)
	}
}

func TestParseAdoptionAgencyImpliedOrdinarySpanCoordinates(t *testing.T) {
	const content = "<b><span>1\r\n<div>2</b>3</span>4</div>5"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	span := formattingElements(document, "span")[0]
	div := formattingElements(document, "div")[0]
	assertFormattingNode(t, bold[0], "b", "1\n", 0, 22)
	assertFormattingNode(t, span, "span", "1\n", 3, 18)
	assertFormattingNode(t, div, "div", "234", 12, 37)
	if span.EndLine != 2 || span.EndColumn != 7 || bold[0].EndLine != 2 || bold[0].EndColumn != 11 {
		t.Fatalf("Implicitly popped span must end at </b> token start (line 2 col 7) while explicitly closed b ends after token (col 11), b=%#v span=%#v", bold[0], span)
	}
}

func TestParseAdoptionAgencyIncompleteFurthestBlockEndsAreDiscarded(t *testing.T) {
	for _, testCase := range []struct {
		content string
		hasI    bool
	}{
		{content: `<b>1<div>2</b`},
		{content: `<b>1<div><i>2</b`, hasI: true},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatalf("Incomplete subject end must be discarded for %q: %v", testCase.content, err)
		}
		bold := formattingElements(document, "b")
		div := formattingElements(document, "div")
		if len(bold) != 1 || len(div) != 1 {
			t.Fatalf("Discarded token must not create adoption clones for %q, b=%#v div=%#v", testCase.content, bold, div)
		}
		assertFormattingNode(t, bold[0], "b", "12", 0, len(testCase.content))
		assertFormattingNode(t, div[0], "div", "2", 4, len(testCase.content))
		if div[0].Parent != bold[0] {
			t.Fatalf("Original div must remain under original b for %q, got %#v", testCase.content, div[0].Parent)
		}
		if testCase.hasI {
			italics := formattingElements(document, "i")
			if len(italics) != 1 {
				t.Fatalf("Expected only source i for %q, got %#v", testCase.content, italics)
			}
			assertFormattingNode(t, italics[0], "i", "2", 9, len(testCase.content))
		} else if len(div[0].Children) != 1 || div[0].Children[0].Type != types.TextNode || div[0].Children[0].Value != "2" || div[0].Children[0].StartPos != 9 || div[0].Children[0].EndPos != len(testCase.content) {
			t.Fatalf("Prior text raw range must extend across discarded EOF token, got %#v", div[0].Children)
		}
	}
}

func TestParseAdoptionAgencyNestedSpecialImplicitClose(t *testing.T) {
	const content = `<b>1<div>2<section>3</b>4</div>5`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	div := formattingElements(document, "div")[0]
	section := formattingElements(document, "section")[0]
	if len(bold) != 3 {
		t.Fatalf("Expected source b plus replacements in div and section, got %#v", bold)
	}
	assertFormattingNode(t, bold[0], "b", "1", 0, 24)
	assertFormattingNode(t, div, "div", "234", 4, 31)
	assertFormattingNode(t, bold[1], "b", "2", 0, 0)
	assertFormattingNode(t, section, "section", "34", 10, 25)
	assertFormattingNode(t, bold[2], "b", "3", 0, 0)
	if bold[1].Parent != div || section.Parent != div || bold[2].Parent != section || len(section.Children) != 2 || section.Children[1].Value != "4" || section.Children[1].StartPos != 24 || section.Children[1].EndPos != 25 {
		t.Fatalf("Missing section end must be implied by </div> after adoption, got %#v", section.Children)
	}
	if parsedBodyChildren(document)[2].Value != "5" || parsedBodyChildren(document)[2].StartPos != 31 || parsedBodyChildren(document)[2].EndPos != 32 {
		t.Fatalf("Expected 5 after div, got %#v", parsedBodyChildren(document))
	}
}

func TestParseAdoptionAgencyDeepChainScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	builders := map[string]func(int) string{
		"furthest block through ordinary and formatting wrappers": func(n int) string {
			var b strings.Builder
			b.Grow(n*18 + 32)
			b.WriteString(`<b>`)
			for i := 0; i < n; i++ {
				b.WriteString(`<span><i>`) // i is active formatting; span is ordinary.
			}
			b.WriteString(`x<div>y</b>z</div>`)
			return b.String()
		},
		"no furthest block through ordinary descendants": func(n int) string {
			var b strings.Builder
			b.Grow(n*3 + 12)
			b.WriteString(`<b>`)
			for i := 0; i < n; i++ {
				b.WriteString(`<x>`)
			}
			b.WriteString(`y</b>z`)
			return b.String()
		},
	}
	measure := func(build func(int) string, n int) time.Duration {
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
	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			small, large := measure(build, 500), measure(build, 2000)
			ratio := float64(large) / float64(small)
			t.Logf("deep adoption scaling 500=%v 2000=%v ratio=%.1fx", small, large, ratio)
			if large > small*10 && large-small > 10*time.Millisecond {
				t.Fatalf("Deep adoption recovery scaled superlinearly: 500=%v 2000=%v ratio=%.1fx", small, large, ratio)
			}
		})
	}
}

func TestParseAdoptionAgencyOuterLoopCapsAtEight(t *testing.T) {
	const content = `<b><div>1<div>2<div>3<div>4<div>5<div>6<div>7<div>8<div>9</b>X`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	divs := formattingElements(document, "div")
	if len(bold) != 9 || len(divs) != 9 {
		t.Fatalf("Expected source b plus exactly eight adoption clones across nine divs, b=%d div=%d", len(bold), len(divs))
	}
	assertFormattingNode(t, bold[0], "b", "", 0, 61)
	for i, div := range divs {
		wantStart := 3 + 6*i
		wantText := "123456789X"[i:]
		assertFormattingNode(t, div, "div", wantText, wantStart, 62)
	}
	for i := 0; i < 7; i++ {
		clone := bold[i+1]
		assertFormattingNode(t, clone, "b", string(rune('1'+i)), 0, 0)
		if clone.Parent != divs[i] || len(clone.Children) != 1 || clone.Children[0].Type != types.TextNode {
			t.Fatalf("Expected adoption clone %d to wrap only its digit in div %d, got %#v", i+1, i+1, clone)
		}
	}
	assertFormattingNode(t, bold[8], "b", "89X", 0, 0)
	if bold[8].Parent != divs[7] || divs[8].Parent != bold[8] {
		t.Fatalf("Eighth clone must contain ninth div when outer loop stops, clone=%#v div9=%#v", bold[8], divs[8])
	}
	if len(formattingElements(divs[8], "b")) != 0 || len(divs[8].Children) != 1 || divs[8].Children[0].Type != types.TextNode || divs[8].Children[0].Value != "9X" || divs[8].Children[0].StartPos != 56 || divs[8].Children[0].EndPos != 62 {
		t.Fatalf("Ninth div must remain plain 9X inside the eighth b clone, got %#v", divs[8].Children)
	}
}

func TestParseAdoptionAgencyOffStackFormattingEndRemovesEntry(t *testing.T) {
	const content = `<p><b>1<div>2</div></b>3`
	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	div := formattingElements(document, "div")[0]
	bold := formattingElements(document, "b")
	if len(bold) != 2 || len(parsedBodyChildren(document)) != 3 {
		t.Fatalf("Expected source b, reconstructed off-stack b, and plain 3, got b=%#v children=%#v", bold, parsedBodyChildren(document))
	}
	assertFormattingNode(t, paragraph, "p", "1", 0, 7)
	assertFormattingNode(t, bold[0], "b", "1", 3, 7)
	assertFormattingNode(t, div, "div", "2", 7, 19)
	assertFormattingNode(t, bold[1], "b", "2", 3, 13)
	if bold[1].Parent != div || len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[2].Type != types.TextNode || parsedBodyChildren(document)[2].Value != "3" || parsedBodyChildren(document)[2].StartPos != 23 || parsedBodyChildren(document)[2].EndPos != 24 {
		t.Fatalf("Off-stack b end must remove AFE without reconstructing around 3, got %#v", parsedBodyChildren(document))
	}

	plain, err := parser.Parse(`z`)
	if err != nil || len(parsedBodyChildren(plain)) != 1 || parsedBodyChildren(plain)[0].Type != types.TextNode || parsedBodyChildren(plain)[0].Value != "z" || len(formattingElements(plain, "b")) != 0 {
		t.Fatalf("Parser reuse leaked off-stack formatting entry: doc=%#v err=%v", plain, err)
	}
}

func TestParseAdoptionAgencyInnerLoopRemovesOrdinarySpan(t *testing.T) {
	const content = `<b><span>1<div>2</b>3</span>4</div>5`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	span := formattingElements(document, "span")[0]
	div := formattingElements(document, "div")[0]
	if len(bold) != 2 || len(parsedBodyChildren(document)) != 3 {
		t.Fatalf("Expected source b/span, div with replacement b, and trailing 5, b=%#v children=%#v", bold, parsedBodyChildren(document))
	}
	assertFormattingNode(t, bold[0], "b", "1", 0, 20)
	assertFormattingNode(t, span, "span", "1", 3, 16)
	assertFormattingNode(t, div, "div", "234", 10, 35)
	assertFormattingNode(t, bold[1], "b", "2", 0, 0)
	if span.Parent != bold[0] || bold[1].Parent != div || len(div.Children) != 2 || div.Children[0] != bold[1] || div.Children[1].Type != types.TextNode || div.Children[1].Value != "34" || div.Children[1].StartPos != 20 || div.Children[1].EndPos != 29 {
		t.Fatalf("Expected removed span's later end ignored with plain 34 in div, got %#v", div.Children)
	}
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[2].Type != types.TextNode || parsedBodyChildren(document)[2].Value != "5" || parsedBodyChildren(document)[2].StartPos != 35 || parsedBodyChildren(document)[2].EndPos != 36 {
		t.Fatalf("Expected trailing 5 after div, got %#v", parsedBodyChildren(document))
	}
}

func TestParseAdoptionAgencyNoahEvictedEntryUsesGenericEndFallback(t *testing.T) {
	const content = `<b><b><b><b>x</b></b></b><span>y</b>z</span>w`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	span := formattingElements(document, "span")[0]
	if len(bold) != 4 || len(parsedBodyChildren(document)) != 2 {
		t.Fatalf("Expected four source b nodes followed by one plain text node, b=%#v children=%#v", bold, parsedBodyChildren(document))
	}
	starts := []int{0, 3, 6, 9}
	ends := []int{36, 25, 21, 17}
	texts := []string{"xy", "x", "x", "x"}
	for i := range bold {
		assertFormattingNode(t, bold[i], "b", texts[i], starts[i], ends[i])
	}
	assertFormattingNode(t, span, "span", "y", 25, 32)
	if span.Parent != bold[0] || len(bold[0].Children) != 2 || bold[0].Children[1] != span {
		t.Fatalf("Expected span(y) to remain in outer Noah-evicted b until generic fallback pops both, got %#v", bold[0].Children)
	}
	trailing := parsedBodyChildren(document)[1]
	if trailing.Type != types.TextNode || trailing.Value != "zw" || trailing.StartPos != 36 || trailing.EndPos != 45 {
		t.Fatalf("Expected ignored stale span end to coalesce plain zw outside b, got %#v", trailing)
	}
}

func TestParseAdoptionAgencyNoahGenericFallbackStopsAtSpecialBoundary(t *testing.T) {
	const content = `<b><b><b><b>x</b></b></b><div>y</b>z</div>w`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	div := formattingElements(document, "div")[0]
	if len(bold) != 4 || len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0] != bold[0] {
		t.Fatalf("Expected evicted outer b to remain the sole body child through EOF, b=%#v children=%#v", bold, parsedBodyChildren(document))
	}
	starts := []int{0, 3, 6, 9}
	ends := []int{43, 25, 21, 17}
	texts := []string{"xyzw", "x", "x", "x"}
	for i := range bold {
		assertFormattingNode(t, bold[i], "b", texts[i], starts[i], ends[i])
	}
	assertFormattingNode(t, div, "div", "yz", 25, 42)
	if div.Parent != bold[0] || len(div.Children) != 1 || div.Children[0].Type != types.TextNode || div.Children[0].Value != "yz" || div.Children[0].StartPos != 30 || div.Children[0].EndPos != 36 {
		t.Fatalf("Special div must block generic b scan while ignored end coalesces yz, got %#v", div.Children)
	}
	if len(bold[0].Children) != 3 || bold[0].Children[1] != div || bold[0].Children[2].Type != types.TextNode || bold[0].Children[2].Value != "w" || bold[0].Children[2].StartPos != 42 || bold[0].Children[2].EndPos != 43 {
		t.Fatalf("Expected div(yz) and trailing w to remain inside outer b, got %#v", bold[0].Children)
	}
}
