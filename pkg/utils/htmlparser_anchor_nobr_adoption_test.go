package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 14C covers current-WHATWG non-table in-body special starts and
// adoption ends for a/nobr. Table/foster/cell markers, customizable select,
// templates, foreign content, fragments, and other adoption subjects remain
// explicitly deferred.

func TestParseNestedAnchorStartsForceCloseExactPriorAnchor(t *testing.T) {
	const basic = `<p><a href=one>1<a href=two>2</a>3</a>4`
	document, err := NewHTMLParser().Parse(basic)
	if err != nil {
		t.Fatal(err)
	}
	p := formattingElements(document, "p")[0]
	a := formattingElements(p, "a")
	assertFormattingNode(t, p, "p", "1234", 0, 39)
	assertFormattingNode(t, a[0], "a", "1", 3, 16)
	assertFormattingNode(t, a[1], "a", "2", 16, 33)
	if len(a) != 2 || a[0].Attributes["href"] != "one" || a[1].Attributes["href"] != "two" || len(p.Children) != 3 || p.Children[2].Value != "34" || p.Children[2].StartPos != 33 || p.Children[2].EndPos != 39 {
		t.Fatalf("Unexpected forced anchor close tree: %#v", p.Children)
	}

	const formatting = `<p><a id=a><b>1<a id=b>2</a>3</b>4`
	document, err = NewHTMLParser().Parse(formatting)
	if err != nil {
		t.Fatal(err)
	}
	p = formattingElements(document, "p")[0]
	a = formattingElements(p, "a")
	bold := formattingElements(p, "b")
	assertFormattingNode(t, a[0], "a", "1", 3, 15)
	assertFormattingNode(t, bold[0], "b", "1", 11, 15)
	assertFormattingNode(t, bold[1], "b", "23", 11, 33)
	assertFormattingNode(t, a[1], "a", "2", 15, 28)
	if bold[1].Parent != p || a[1].Parent != bold[1] || p.Children[2].Value != "4" {
		t.Fatalf("Expected reconstructed b around new anchor and 3, got %#v", p.Children)
	}
}

func TestParseNestedAnchorAcrossBlockMarkerAndButton(t *testing.T) {
	for _, testCase := range []struct {
		name, content, container                                 string
		outerEnd, containerStart, containerEnd, newStart, newEnd int
	}{
		{name: "block", content: `<a id=a>1<div>2<a id=b>3</a>4</div>5`, container: "div", outerEnd: 15, containerStart: 9, containerEnd: 35, newStart: 15, newEnd: 28},
		{name: "button", content: `<a id=a>1<button>2<a id=b>3</a>4</button>5`, container: "button", outerEnd: 18, containerStart: 9, containerEnd: 41, newStart: 18, newEnd: 31},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		a := formattingElements(document, "a")
		container := formattingElements(document, testCase.container)[0]
		if len(a) != 3 {
			t.Fatalf("Expected source, reconstructed, and new anchors for %s, got %#v", testCase.name, a)
		}
		assertFormattingNode(t, a[0], "a", "1", 0, testCase.outerEnd)
		assertFormattingNode(t, container, testCase.container, "234", testCase.containerStart, testCase.containerEnd)
		assertFormattingNode(t, a[1], "a", "2", 0, 0)
		assertFormattingNode(t, a[2], "a", "3", testCase.newStart, testCase.newEnd)
		if a[1].Parent != container || a[2].Parent != container || container.Children[2].Value != "4" {
			t.Fatalf("Wrong nested anchor hierarchy for %s: %#v", testCase.name, container.Children)
		}
	}

	const marker = `<a id=a><object>1<a id=b>2</a>3</object>4`
	document, err := NewHTMLParser().Parse(marker)
	if err != nil {
		t.Fatal(err)
	}
	a := formattingElements(document, "a")
	object := formattingElements(document, "object")[0]
	assertFormattingNode(t, a[0], "a", "1234", 0, 41)
	assertFormattingNode(t, object, "object", "123", 8, 40)
	assertFormattingNode(t, a[1], "a", "2", 17, 30)
	if a[1].Parent != object || object.Parent != a[0] {
		t.Fatalf("Object marker must isolate nested anchor identity, got %#v", a)
	}
}

func TestParseNestedNobrStartsForceCloseAndReconstruct(t *testing.T) {
	const basic = `<p><nobr>1<nobr>2</nobr>3</nobr>4`
	document, err := NewHTMLParser().Parse(basic)
	if err != nil {
		t.Fatal(err)
	}
	p := formattingElements(document, "p")[0]
	nobr := formattingElements(p, "nobr")
	assertFormattingNode(t, nobr[0], "nobr", "1", 3, 10)
	assertFormattingNode(t, nobr[1], "nobr", "2", 10, 24)
	if len(nobr) != 2 || p.Children[2].Value != "34" || p.Children[2].StartPos != 24 || p.Children[2].EndPos != 33 {
		t.Fatalf("Unexpected nested nobr close tree: %#v", p.Children)
	}

	const formatting = `<p><nobr><b>1<nobr>2</nobr>3</b>4`
	document, err = NewHTMLParser().Parse(formatting)
	if err != nil {
		t.Fatal(err)
	}
	p = formattingElements(document, "p")[0]
	nobr = formattingElements(p, "nobr")
	bold := formattingElements(p, "b")
	assertFormattingNode(t, nobr[0], "nobr", "1", 3, 13)
	assertFormattingNode(t, bold[0], "b", "1", 9, 13)
	assertFormattingNode(t, bold[1], "b", "23", 9, 32)
	assertFormattingNode(t, nobr[1], "nobr", "2", 13, 27)
	if nobr[1].Parent != bold[1] || p.Children[2].Value != "4" {
		t.Fatalf("Expected b reconstruction around new nobr and 3, got %#v", p.Children)
	}
}

func TestParseNestedNobrAcrossBlockAndMarker(t *testing.T) {
	const block = `<nobr>1<div>2<nobr>3</nobr>4</div>5`
	document, err := NewHTMLParser().Parse(block)
	if err != nil {
		t.Fatal(err)
	}
	nobr := formattingElements(document, "nobr")
	div := formattingElements(document, "div")[0]
	assertFormattingNode(t, nobr[0], "nobr", "1", 0, 13)
	assertFormattingNode(t, div, "div", "234", 7, 34)
	assertFormattingNode(t, nobr[1], "nobr", "2", 0, 0)
	assertFormattingNode(t, nobr[2], "nobr", "3", 13, 27)
	if nobr[1].Parent != div || nobr[2].Parent != div || div.Children[2].Value != "4" {
		t.Fatalf("Wrong block nobr hierarchy: %#v", div.Children)
	}

	const marker = `<nobr><object>1<nobr>2</nobr>3</object>4`
	document, err = NewHTMLParser().Parse(marker)
	if err != nil {
		t.Fatal(err)
	}
	nobr = formattingElements(document, "nobr")
	object := formattingElements(document, "object")[0]
	assertFormattingNode(t, nobr[0], "nobr", "1234", 0, 40)
	assertFormattingNode(t, object, "object", "123", 6, 39)
	assertFormattingNode(t, nobr[1], "nobr", "2", 15, 29)
}

func TestParseAnchorNobrSpecialStartGating(t *testing.T) {
	for _, testCase := range []struct {
		name, incomplete, self string
		secondStart            int
	}{
		{name: "a", incomplete: "<a id=a>é\r\n<a id=b", self: `<p><a>1<a/>2`, secondStart: 7},
		{name: "nobr", incomplete: "<nobr>é\r\n<nobr", self: `<p><nobr>1<nobr/>2`, secondStart: 10},
	} {
		document, err := NewHTMLParser().Parse(testCase.incomplete)
		if err != nil {
			t.Fatal(err)
		}
		nodes := formattingElements(document, testCase.name)
		if len(nodes) != 1 || nodes[0].TextContent != "é\n" || nodes[0].StartPos != 0 || nodes[0].EndPos != len([]byte(testCase.incomplete)) {
			t.Fatalf("Incomplete second <%s> must be discarded without fake end, got %#v", testCase.name, nodes)
		}
		document, err = NewHTMLParser().Parse(testCase.self)
		if err != nil {
			t.Fatal(err)
		}
		nodes = formattingElements(document, testCase.name)
		if len(nodes) != 2 || nodes[0].TextContent != "1" || nodes[0].EndPos != testCase.secondStart || nodes[1].TextContent != "2" || nodes[1].StartPos != testCase.secondStart || nodes[1].EndPos != len(testCase.self) {
			t.Fatalf("Self-closing slash on nonvoid <%s> must still run special start, got %#v", testCase.name, nodes)
		}
	}
}

func TestParseAnchorNobrAdoptionEnds(t *testing.T) {
	const anchor = `<p><a href=x>1<i>2</a>3</i>4`
	document, err := NewHTMLParser().Parse(anchor)
	if err != nil {
		t.Fatal(err)
	}
	a := formattingElements(document, "a")[0]
	italics := formattingElements(document, "i")
	assertFormattingNode(t, a, "a", "12", 3, 22)
	assertFormattingNode(t, italics[0], "i", "2", 14, 18)
	assertFormattingNode(t, italics[1], "i", "3", 14, 27)

	const nobrBlock = `<nobr>1<div>2</nobr>3</div>4`
	document, err = NewHTMLParser().Parse(nobrBlock)
	if err != nil {
		t.Fatal(err)
	}
	nobr := formattingElements(document, "nobr")
	div := formattingElements(document, "div")[0]
	assertFormattingNode(t, nobr[0], "nobr", "1", 0, 20)
	assertFormattingNode(t, div, "div", "23", 7, 27)
	assertFormattingNode(t, nobr[1], "nobr", "2", 0, 0)
}

func TestParseAnchorNobrOutOfScopeAbsentAndIncompleteEnds(t *testing.T) {
	for _, testCase := range []struct{ name, content string }{
		{name: "a", content: `<a><object>x</a>y</object>z`},
		{name: "nobr", content: `<nobr><object>x</nobr>y</object>z`},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		outer := formattingElements(document, testCase.name)[0]
		object := formattingElements(document, "object")[0]
		assertFormattingNode(t, outer, testCase.name, "xyz", 0, len(testCase.content))
		if object.Parent != outer || object.TextContent != "xy" || outer.Children[1].Value != "z" {
			t.Fatalf("Out-of-scope %s end must be ignored behind object, got %#v", testCase.name, outer.Children)
		}
	}

	const absent = `<p>x</a>y</nobr>z`
	document, err := NewHTMLParser().Parse(absent)
	if err != nil {
		t.Fatal(err)
	}
	p := formattingElements(document, "p")[0]
	assertFormattingNode(t, p, "p", "xyz", 0, 17)
	if len(p.Children) != 1 || p.Children[0].Value != "xyz" || p.Children[0].StartPos != 3 || p.Children[0].EndPos != 17 {
		t.Fatalf("Absent a/nobr ends must be ignored and text coalesced, got %#v", p.Children)
	}

	for _, testCase := range []struct{ name, content string }{{"a", `<a>1<div>2</a`}, {"nobr", `<nobr>1<div>2</nobr`}} {
		document, err = NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		outer := formattingElements(document, testCase.name)
		div := formattingElements(document, "div")
		if len(outer) != 1 || len(div) != 1 || div[0].Parent != outer[0] || outer[0].TextContent != "12" || outer[0].EndPos != len(testCase.content) || div[0].EndPos != len(testCase.content) {
			t.Fatalf("Incomplete %s end must be discarded without adoption, outer=%#v div=%#v", testCase.name, outer, div)
		}
	}
}

func TestParseAnchorNobrMultibyteLocationsAndReuse(t *testing.T) {
	for _, testCase := range []struct {
		name, content, attr, value         string
		start, end, iStart, iEnd, cloneEnd int
	}{
		{name: "a", content: "<p><a href=x>é\r\n<i>😀</a>z</i>w", attr: "href", value: "x", start: 3, end: 28, iStart: 17, iEnd: 24, cloneEnd: 33},
		{name: "nobr", content: "<p><nobr class=x>é\r\n<i>😀</nobr>z</i>w", attr: "class", value: "x", start: 3, end: 35, iStart: 21, iEnd: 28, cloneEnd: 40},
	} {
		parser := NewHTMLParser()
		document, err := parser.Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		outer := formattingElements(document, testCase.name)[0]
		italics := formattingElements(document, "i")
		assertFormattingNode(t, outer, testCase.name, "é\n😀", testCase.start, testCase.end)
		assertFormattingNode(t, italics[0], "i", "😀", testCase.iStart, testCase.iEnd)
		assertFormattingNode(t, italics[1], "i", "z", testCase.iStart, testCase.cloneEnd)
		if outer.Attributes[testCase.attr] != testCase.value || outer.StartLine != 1 || outer.StartColumn != 4 || italics[0].StartLine != 2 || italics[0].StartColumn != 1 || italics[0].EndLine != 2 || italics[0].EndColumn != 6 {
			t.Fatalf("Wrong metadata/UTF-16 coordinates for %s: outer=%#v i=%#v", testCase.name, outer, italics)
		}
		plain, err := parser.Parse(`z`)
		if err != nil || len(parsedBodyChildren(plain)) != 1 || parsedBodyChildren(plain)[0].Type != types.TextNode || parsedBodyChildren(plain)[0].Value != "z" || len(formattingElements(plain, testCase.name)) != 0 {
			t.Fatalf("Parser reuse leaked %s state: doc=%#v err=%v", testCase.name, plain, err)
		}
	}
}

func TestParseAnchorNobrStartScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	for _, name := range []string{"a", "nobr"} {
		t.Run(name, func(t *testing.T) {
			build := func(n int) string {
				var b strings.Builder
				b.Grow(n * 18)
				b.WriteString(`<p>`)
				for i := 0; i < n; i++ {
					b.WriteByte('<')
					b.WriteString(name)
					b.WriteString(`><b>`)
				}
				b.WriteByte('x')
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
					if d := time.Since(start); d < best {
						best = d
					}
				}
				return best
			}
			small, large := measure(1000), measure(4000)
			ratio := float64(large) / float64(small)
			t.Logf("%s nested-start scaling 1000=%v 4000=%v ratio=%.1fx", name, small, large, ratio)
			if large > small*10 && large-small > 20*time.Millisecond {
				t.Fatalf("%s special-start recovery scaled superlinearly: %.1fx", name, ratio)
			}
		})
	}
}

func TestParseAnchorNobrOffStackStartLoopIdentity(t *testing.T) {
	const anchor = `<p><a id=a>1<div>2</div><a id=b>3`
	document, err := NewHTMLParser().Parse(anchor)
	if err != nil {
		t.Fatal(err)
	}
	p := formattingElements(document, "p")[0]
	div := formattingElements(document, "div")[0]
	a := formattingElements(document, "a")
	if len(a) != 3 {
		t.Fatalf("Expected source, div reconstruction, and new root anchor only, got %#v", a)
	}
	assertFormattingNode(t, p, "p", "1", 0, 12)
	assertFormattingNode(t, a[0], "a", "1", 3, 12)
	assertFormattingNode(t, div, "div", "2", 12, 24)
	assertFormattingNode(t, a[1], "a", "2", 3, 18)
	assertFormattingNode(t, a[2], "a", "3", 24, 33)
	if a[0].Parent != p || a[1].Parent != div || a[2].Parent != parsedBody(document) || a[2].Attributes["id"] != "b" {
		t.Fatalf("Off-stack prior anchor cleanup produced wrong hierarchy: %#v", a)
	}

	// Current Chrome 151 and jsdom agree that nobr's required
	// reconstruct/fake-end/reconstruct ordering creates an empty root clone of
	// the old nobr before the new token is inserted. This intentionally differs
	// from anchor's forced exact-identity cleanup above.
	const nobrContent = `<p><nobr id=a>1<div>2</div><nobr id=b>3`
	document, err = NewHTMLParser().Parse(nobrContent)
	if err != nil {
		t.Fatal(err)
	}
	p = formattingElements(document, "p")[0]
	div = formattingElements(document, "div")[0]
	nobr := formattingElements(document, "nobr")
	if len(nobr) != 4 {
		t.Fatalf("Chrome expects source, div clone, empty root clone, and new root nobr, got %#v", nobr)
	}
	assertFormattingNode(t, p, "p", "1", 0, 15)
	assertFormattingNode(t, nobr[0], "nobr", "1", 3, 15)
	assertFormattingNode(t, div, "div", "2", 15, 27)
	assertFormattingNode(t, nobr[1], "nobr", "2", 3, 21)
	assertFormattingNode(t, nobr[2], "nobr", "", 3, 27)
	assertFormattingNode(t, nobr[3], "nobr", "3", 27, 39)
	if nobr[1].Parent != div || nobr[2].Parent != parsedBody(document) || nobr[3].Parent != parsedBody(document) || nobr[3].Attributes["id"] != "b" {
		t.Fatalf("Off-stack nobr start loop produced wrong hierarchy: %#v", nobr)
	}
}

func TestParseAnchorNobrSectionVirtualScopeStartLoop(t *testing.T) {
	const anchor = `<section><p><a id=a>1<div>2</div><a id=b>3</section>`
	document, err := NewHTMLParser().Parse(anchor)
	if err != nil {
		t.Fatal(err)
	}
	section := formattingElements(document, "section")[0]
	p := formattingElements(section, "p")[0]
	div := formattingElements(section, "div")[0]
	a := formattingElements(section, "a")
	if len(a) != 3 {
		t.Fatalf("Expected exactly three section-scoped anchors, got %#v", a)
	}
	assertFormattingNode(t, section, "section", "123", 0, 52)
	assertFormattingNode(t, p, "p", "1", 9, 21)
	assertFormattingNode(t, a[0], "a", "1", 12, 21)
	assertFormattingNode(t, div, "div", "2", 21, 33)
	assertFormattingNode(t, a[1], "a", "2", 12, 27)
	assertFormattingNode(t, a[2], "a", "3", 33, 42)
	if a[0].Parent != p || a[1].Parent != div || a[2].Parent != section || a[2].Attributes["id"] != "b" {
		t.Fatalf("Section virtual-scope anchor cleanup produced wrong hierarchy: %#v", a)
	}

	for _, testCase := range []struct {
		name, content        string
		hasText              bool
		newStart, sectionEnd int
	}{
		{name: "empty reconstruction", content: `<section><p><nobr id=a>1<div>2</div><nobr id=b>3</section>`, newStart: 36, sectionEnd: 58},
		{name: "text reconstruction", content: `<section><p><nobr id=a>1<div>2</div>x<nobr id=b>3</section>`, hasText: true, newStart: 37, sectionEnd: 59},
	} {
		document, err = NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		section = formattingElements(document, "section")[0]
		p = formattingElements(section, "p")[0]
		div = formattingElements(section, "div")[0]
		nobr := formattingElements(section, "nobr")
		if len(nobr) != 4 {
			t.Fatalf("Expected source, div clone, section clone, and new nobr for %s, got %#v", testCase.name, nobr)
		}
		assertFormattingNode(t, section, "section", map[bool]string{false: "123", true: "12x3"}[testCase.hasText], 0, testCase.sectionEnd)
		assertFormattingNode(t, p, "p", "1", 9, 24)
		assertFormattingNode(t, nobr[0], "nobr", "1", 12, 24)
		assertFormattingNode(t, div, "div", "2", 24, 36)
		assertFormattingNode(t, nobr[1], "nobr", "2", 12, 30)
		assertFormattingNode(t, nobr[2], "nobr", map[bool]string{false: "", true: "x"}[testCase.hasText], 12, testCase.newStart)
		assertFormattingNode(t, nobr[3], "nobr", "3", testCase.newStart, testCase.newStart+12)
		if nobr[1].Parent != div || nobr[2].Parent != section || nobr[3].Parent != section || nobr[3].Attributes["id"] != "b" {
			t.Fatalf("Section virtual-scope nobr handling produced wrong hierarchy for %s: %#v", testCase.name, nobr)
		}
	}
}

func TestParseAnchorNobrOffStackStartPreservesSuffixFormatting(t *testing.T) {
	for _, testCase := range []struct {
		name, subject, content                 string
		sourceSubjectStart, sourceEnd          int
		divStart, divEnd, cloneEnd             int
		suffixEnd, newStart, newEnd, finalBEnd int
	}{
		{name: "root anchor", subject: "a", content: `<p><a id=a><b>1<div>2</div>x<a id=b>3`, sourceSubjectStart: 3, sourceEnd: 15, divStart: 15, divEnd: 27, cloneEnd: 21, suffixEnd: 28, newStart: 28, newEnd: 37, finalBEnd: 37},
		{name: "section anchor", subject: "a", content: `<section><p><a id=a><b>1<div>2</div>x<a id=b>3</section>`, sourceSubjectStart: 12, sourceEnd: 24, divStart: 24, divEnd: 36, cloneEnd: 30, suffixEnd: 37, newStart: 37, newEnd: 46, finalBEnd: 46},
		{name: "explicit body anchor", subject: "a", content: `<html><body><p><a id=a><b>1<div>2</div>x<a id=b>3</body></html>`, sourceSubjectStart: 15, sourceEnd: 27, divStart: 27, divEnd: 39, cloneEnd: 33, suffixEnd: 40, newStart: 40, newEnd: 63, finalBEnd: 63},
		{name: "root nobr", subject: "nobr", content: `<p><nobr id=a><b>1<div>2</div>x<nobr id=b>3`, sourceSubjectStart: 3, sourceEnd: 18, divStart: 18, divEnd: 30, cloneEnd: 24, suffixEnd: 31, newStart: 31, newEnd: 43, finalBEnd: 43},
		{name: "section nobr", subject: "nobr", content: `<section><p><nobr id=a><b>1<div>2</div>x<nobr id=b>3</section>`, sourceSubjectStart: 12, sourceEnd: 27, divStart: 27, divEnd: 39, cloneEnd: 33, suffixEnd: 40, newStart: 40, newEnd: 52, finalBEnd: 52},
		{name: "explicit body nobr", subject: "nobr", content: `<html><body><p><nobr id=a><b>1<div>2</div>x<nobr id=b>3</body></html>`, sourceSubjectStart: 15, sourceEnd: 30, divStart: 30, divEnd: 42, cloneEnd: 36, suffixEnd: 43, newStart: 43, newEnd: 69, finalBEnd: 69},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		subjects := formattingElements(document, testCase.subject)
		bold := formattingElements(document, "b")
		div := formattingElements(document, "div")[0]
		if len(subjects) != 4 || len(bold) != 4 {
			t.Fatalf("Expected source/div/suffix/new subjects and four b instances for %s, subject=%#v b=%#v", testCase.name, subjects, bold)
		}
		assertFormattingNode(t, subjects[0], testCase.subject, "1", testCase.sourceSubjectStart, testCase.sourceEnd)
		assertFormattingNode(t, bold[0], "b", "1", testCase.sourceSubjectStart+map[string]int{"a": 8, "nobr": 11}[testCase.subject], testCase.sourceEnd)
		assertFormattingNode(t, div, "div", "2", testCase.divStart, testCase.divEnd)
		assertFormattingNode(t, subjects[1], testCase.subject, "2", testCase.sourceSubjectStart, testCase.cloneEnd)
		assertFormattingNode(t, subjects[2], testCase.subject, "x", testCase.sourceSubjectStart, testCase.suffixEnd)
		assertFormattingNode(t, subjects[3], testCase.subject, "3", testCase.newStart, testCase.newEnd)
		assertFormattingNode(t, bold[3], "b", "3", bold[0].StartPos, testCase.finalBEnd)
		if subjects[1].Parent != div || bold[1].Parent != subjects[1] || bold[2].Parent != subjects[2] || subjects[3].Parent != bold[3] || bold[3].Parent == subjects[2] {
			t.Fatalf("New %s must be inside reconstructed b but outside old suffix %s: subject=%#v b=%#v", testCase.subject, testCase.subject, subjects, bold)
		}
	}
}

func TestParseNobrSpecialStartAfterBodyAndHTML(t *testing.T) {
	for _, testCase := range []struct {
		name, content, cloneText   string
		cloneEnd, newStart, newEnd int
		commentStart, commentEnd   int
		commentParent              string
	}{
		{name: "after body", content: `<html><body><p><nobr id=a>1<div>2</div></body><nobr id=b>3</html>`, cloneEnd: 46, newStart: 46, newEnd: 65},
		{name: "after body whitespace", content: `<html><body><p><nobr id=a>1<div>2</div></body> <nobr id=b>3</html>`, cloneText: " ", cloneEnd: 47, newStart: 47, newEnd: 66},
		{name: "after body comment", content: `<html><body><p><nobr id=a>1<div>2</div></body><!--c--><nobr id=b>3</html>`, cloneEnd: 54, newStart: 54, newEnd: 73, commentStart: 46, commentEnd: 54, commentParent: "html"},
		{name: "after body comment whitespace", content: `<html><body><p><nobr id=a>1<div>2</div></body><!--c--> <nobr id=b>3</html>`, cloneText: " ", cloneEnd: 55, newStart: 55, newEnd: 74, commentStart: 46, commentEnd: 54, commentParent: "html"},
		{name: "after html", content: `<html><body><p><nobr id=a>1<div>2</div></body></html><nobr id=b>3`, cloneEnd: 53, newStart: 53, newEnd: 65},
		{name: "after html whitespace", content: `<html><body><p><nobr id=a>1<div>2</div></body></html> <nobr id=b>3`, cloneText: " ", cloneEnd: 54, newStart: 54, newEnd: 66},
		{name: "after html comment", content: `<html><body><p><nobr id=a>1<div>2</div></body></html><!--c--><nobr id=b>3`, cloneEnd: 61, newStart: 61, newEnd: 73, commentStart: 53, commentEnd: 61, commentParent: "#document"},
		{name: "after html comment whitespace", content: `<html><body><p><nobr id=a>1<div>2</div></body></html><!--c--> <nobr id=b>3`, cloneText: " ", cloneEnd: 62, newStart: 62, newEnd: 74, commentStart: 53, commentEnd: 61, commentParent: "#document"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			html := formattingElements(document, "html")[0]
			body := formattingElements(html, "body")[0]
			nobr := formattingElements(document, "nobr")
			if len(nobr) != 4 {
				t.Fatalf("Expected source, div clone, after-body clone, and new nobr, got %#v", nobr)
			}
			assertFormattingNode(t, nobr[0], "nobr", "1", 15, 27)
			assertFormattingNode(t, nobr[1], "nobr", "2", 15, 33)
			assertFormattingNode(t, nobr[2], "nobr", testCase.cloneText, 15, testCase.cloneEnd)
			assertFormattingNode(t, nobr[3], "nobr", "3", testCase.newStart, testCase.newEnd)
			if nobr[2].Parent != body || nobr[3].Parent != body || nobr[3].Attributes["id"] != "b" || nobr[3].Parent == nobr[2] {
				t.Fatalf("Post-document nobr start must produce sibling old/new nobr in body, got %#v", nobr)
			}
			if testCase.cloneText == " " {
				if len(nobr[2].Children) != 1 || nobr[2].Children[0].Type != types.TextNode || nobr[2].Children[0].Value != " " || nobr[2].Children[0].StartPos != testCase.cloneEnd-1 || nobr[2].Children[0].EndPos != testCase.cloneEnd {
					t.Fatalf("Expected separator whitespace inside reconstructed old nobr, got %#v", nobr[2].Children)
				}
			}
			comments := findAllNodesByType(document, types.CommentNode)
			if testCase.commentStart == 0 {
				if len(comments) != 0 {
					t.Fatalf("Unexpected comment nodes: %#v", comments)
				}
			} else if len(comments) != 1 || comments[0].Value != "c" || comments[0].StartPos != testCase.commentStart || comments[0].EndPos != testCase.commentEnd || comments[0].Parent == nil || comments[0].Parent.Name != testCase.commentParent {
				t.Fatalf("Expected comment at %d:%d under %s, got %#v", testCase.commentStart, testCase.commentEnd, testCase.commentParent, comments)
			}
		})
	}
}

func TestParseAnchorNobrExplicitDocumentRecoveryClosesEOFDescendantsWithoutLegacyLeak(t *testing.T) {
	for _, testCase := range []struct {
		name, subject, valid, invalid string
	}{
		{
			name:    "anchor",
			subject: "a",
			valid:   `<html><body><p><a>x</p></body>y</a>`,
			invalid: `<html><body><p><a>x</p></body>y</a><foo>x`,
		},
		{
			name:    "nobr",
			subject: "nobr",
			valid:   `<html><body><p><nobr>x</p></body>y</nobr>`,
			invalid: `<html><body><p><nobr>x</p></body>y</nobr><foo>x`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			valid, err := parser.Parse(testCase.valid)
			if err != nil {
				t.Fatalf("Valid document-end reconstruction failed: %v", err)
			}
			subject := formattingElements(valid, testCase.subject)
			if len(subject) != 2 || subject[0].TextContent != "x" || subject[1].TextContent != "y" {
				t.Fatalf("Expected source %s(x) plus reconstructed and explicitly closed %s(y), got %#v", testCase.subject, testCase.subject, subject)
			}

			recovered, err := parser.Parse(testCase.invalid)
			if err != nil {
				t.Fatalf("Qualifying explicit document must EOF-close later foo after %s recovery: %v", testCase.name, err)
			}
			foo := formattingElements(recovered, "foo")
			if len(foo) != 1 || foo[0].TextContent != "x" || foo[0].EndPos != len(testCase.invalid) {
				t.Fatalf("Explicit document foo EOF recovery mismatch after %s: %#v", testCase.name, foo)
			}

			plain, reuseErr := parser.Parse(`z`)
			if reuseErr != nil || len(parsedBodyChildren(plain)) != 1 || parsedBodyChildren(plain)[0].Type != types.TextNode || parsedBodyChildren(plain)[0].Value != "z" || len(formattingElements(plain, testCase.subject)) != 0 || len(formattingElements(plain, "foo")) != 0 {
				t.Fatalf("Parser reuse after explicit EOF recovery leaked %s/foo state: doc=%#v err=%v", testCase.name, plain, reuseErr)
			}
		})
	}
}
