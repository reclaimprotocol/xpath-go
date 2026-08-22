package utils

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseFormattingCanonicalTableReconstructionAndMarkers(t *testing.T) {
	const canonical = `<table><b><tr><td>aaa</td></tr>bbb</table>ccc`
	document, err := NewHTMLParser().Parse(canonical)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	table := tableElements(document, "table")[0]
	if len(bold) != 3 || len(parsedBodyChildren(document)) != 4 {
		t.Fatalf("Expected empty b, bbb b, table, ccc b, got b=%#v children=%#v", bold, parsedBodyChildren(document))
	}
	assertFormattingNode(t, bold[0], "b", "", 7, 10)
	assertFormattingNode(t, bold[1], "b", "bbb", 7, 34)
	assertFormattingNode(t, table, "table", "aaa", 0, 42)
	assertFormattingNode(t, tableElements(table, "tbody")[0], "tbody", "aaa", 0, 0)
	assertFormattingNode(t, tableElements(table, "tr")[0], "tr", "aaa", 10, 31)
	assertFormattingNode(t, tableElements(table, "td")[0], "td", "aaa", 14, 26)
	assertFormattingNode(t, bold[2], "b", "ccc", 7, 45)
	if parsedBodyChildren(document)[0] != bold[0] || parsedBodyChildren(document)[1] != bold[1] || parsedBodyChildren(document)[2] != table || parsedBodyChildren(document)[3] != bold[2] {
		t.Fatalf("Expected canonical formatting/table foster order, got %#v", parsedBodyChildren(document))
	}

	const caption = `<table><caption><b>x</caption>y</table>z`
	document, err = NewHTMLParser().Parse(caption)
	if err != nil {
		t.Fatal(err)
	}
	table = tableElements(document, "table")[0]
	captionNode := tableElements(table, "caption")[0]
	bold = formattingElements(document, "b")
	assertFormattingNode(t, table, "table", "x", 0, 39)
	assertFormattingNode(t, captionNode, "caption", "x", 7, 30)
	assertFormattingNode(t, bold[0], "b", "x", 16, 20)
	if len(bold) != 1 || len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[0].Value != "y" || parsedBodyChildren(document)[0].StartPos != 30 || parsedBodyChildren(document)[0].EndPos != 31 || parsedBodyChildren(document)[1] != table || parsedBodyChildren(document)[2].Value != "z" || parsedBodyChildren(document)[2].StartPos != 39 || parsedBodyChildren(document)[2].EndPos != 40 {
		t.Fatalf("Expected Chrome caption marker order y, table, z with no b leakage, got %#v", parsedBodyChildren(document))
	}
}

func TestParseFormattingNonTriggerAndCurrentSelectMatrix(t *testing.T) {
	for _, testCase := range []struct {
		name, content    string
		start, end, bEnd int
	}{
		{name: "textarea", content: `<p><b>x</p><textarea>t</textarea>y`, start: 11, end: 33, bEnd: 34},
		{name: "iframe", content: `<p><b>x</p><iframe>t</iframe>y`, start: 11, end: 29, bEnd: 30},
		{name: "param", content: `<p><b>x</p><param>y`, start: 11, end: 18, bEnd: 19},
		{name: "source", content: `<p><b>x</p><source>y`, start: 11, end: 19, bEnd: 20},
		{name: "track", content: `<p><b>x</p><track>y`, start: 11, end: 18, bEnd: 19},
		{name: "hr", content: `<p><b>x</p><hr>y`, start: 11, end: 15, bEnd: 16},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatalf("Parse failed for non-trigger %s: %v", testCase.name, err)
		}
		node := formattingElements(document, testCase.name)[0]
		bold := formattingElements(document, "b")
		assertFormattingNode(t, node, testCase.name, map[string]string{"textarea": "t", "iframe": "t"}[testCase.name], testCase.start, testCase.end)
		assertFormattingNode(t, bold[1], "b", "y", 3, testCase.bEnd)
		if node.Parent != parsedBody(document) || bold[1].Parent != parsedBody(document) {
			t.Fatalf("Expected %s not to trigger b until following y, got node=%#v b=%#v", testCase.name, node, bold[1])
		}
	}

	const currentSelect = `<p><b>x</p><select><span>y</span></select>z`
	document, err := NewHTMLParser().Parse(currentSelect)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	selectNode := formattingElements(document, "select")[0]
	span := formattingElements(selectNode, "span")[0]
	assertFormattingNode(t, bold[1], "b", "yz", 3, 43)
	assertFormattingNode(t, selectNode, "select", "y", 11, 42)
	assertFormattingNode(t, span, "span", "y", 19, 33)
	if selectNode.Parent != bold[1] {
		t.Fatalf("Current Chrome customizable select must participate in in-body reconstruction, got %#v", selectNode.Parent)
	}
}

func TestParseBlinkAndSpanAreNotActiveFormatting(t *testing.T) {
	for _, testCase := range []struct {
		name, content string
		end           int
	}{
		{name: "span", content: `<p><span>x</p>y`, end: 10},
		{name: "blink", content: `<p><blink>x</p>y`, end: 11},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatalf("Parse failed for ordinary %s: %v", testCase.name, err)
		}
		nodes := formattingElements(document, testCase.name)
		if len(nodes) != 1 {
			t.Fatalf("Expected no reconstructed %s, got %#v", testCase.name, nodes)
		}
		assertFormattingNode(t, nodes[0], testCase.name, "x", 3, testCase.end)
		if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].Value != "y" || parsedBodyChildren(document)[1].StartPos != len(testCase.content)-1 || parsedBodyChildren(document)[1].EndPos != len(testCase.content) {
			t.Fatalf("Expected y outside ordinary %s with no reconstruction, got %#v", testCase.name, parsedBodyChildren(document))
		}
	}
}

func TestParseAllFormattingElementsHaveSimpleEndBehavior(t *testing.T) {
	for _, name := range formattingElementNames {
		t.Run(name, func(t *testing.T) {
			open := "<" + name + ">"
			close := "</" + name + ">"
			content := open + "x" + close + "y"
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			nodes := formattingElements(document, name)
			if len(nodes) != 1 {
				t.Fatalf("Expected one simple <%s>, got %#v", name, nodes)
			}
			assertFormattingNode(t, nodes[0], name, "x", 0, len(open)+1+len(close))
			if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0] != nodes[0] || parsedBodyChildren(document)[1].Type != types.TextNode || parsedBodyChildren(document)[1].Value != "y" || parsedBodyChildren(document)[1].StartPos != len(content)-1 || parsedBodyChildren(document)[1].EndPos != len(content) {
				t.Fatalf("Expected y after explicitly closed <%s>, got %#v", name, parsedBodyChildren(document))
			}
		})
	}
}

func TestParseFormattingEOFAndParserReuse(t *testing.T) {
	const eof = `<section><strong>x`
	document, err := NewHTMLParser().Parse(eof)
	if err != nil {
		t.Fatal(err)
	}
	assertFormattingNode(t, formattingElements(document, "section")[0], "section", "x", 0, len(eof))
	assertFormattingNode(t, formattingElements(document, "strong")[0], "strong", "x", 9, len(eof))

	parser := NewHTMLParser()
	for _, content := range []string{`<p><b>one<div>two</div>three`, `<ul><li><em>a<li>b</ul>`, eof, `<u>x</u>y`} {
		if _, parseErr := parser.Parse(content); parseErr != nil {
			t.Fatalf("Reused parser failed for %q: %v", content, parseErr)
		}
	}
	if document, parseErr := parser.Parse(`<b>x</b><div>y`); parseErr != nil || len(formattingElements(document, "div")) != 1 || formattingElements(document, "div")[0].EndPos != len(`<b>x</b><div>y`) {
		t.Fatalf("Formatting-to-implicit-document EOF recovery mismatch: doc=%#v err=%v", document, parseErr)
	}
	plain, parseErr := parser.Parse(`z`)
	if parseErr != nil || len(parsedBodyChildren(plain)) != 1 || parsedBodyChildren(plain)[0].Type != types.TextNode || parsedBodyChildren(plain)[0].Value != "z" || len(formattingElements(plain, "b")) != 0 || len(formattingElements(plain, "em")) != 0 || len(formattingElements(plain, "strong")) != 0 {
		t.Fatalf("Parser reuse leaked active formatting state into plain z: doc=%#v err=%v", plain, parseErr)
	}
}

func TestParseFormattingMultibyteLocations(t *testing.T) {
	const multiline = "<p><strong>é\r\n<span>x<div>😀</div>z"
	document, err := NewHTMLParser().Parse(multiline)
	if err != nil {
		t.Fatal(err)
	}
	strong := formattingElements(document, "strong")
	if len(strong) != 3 {
		t.Fatalf("Expected strong reconstruction across multibyte block transition, got %#v", strong)
	}
	assertFormattingNode(t, formattingElements(document, "p")[0], "p", "é\nx", 0, 22)
	assertFormattingNode(t, strong[0], "strong", "é\nx", 3, 22)
	assertFormattingNode(t, formattingElements(document, "span")[0], "span", "x", 15, 22)
	assertFormattingNode(t, formattingElements(document, "div")[0], "div", "😀", 22, 37)
	assertFormattingNode(t, strong[1], "strong", "😀", 3, 31)
	assertFormattingNode(t, strong[2], "strong", "z", 3, 38)
	if strong[0].StartLine != 1 || strong[0].StartColumn != 4 || strong[0].EndLine != 2 || strong[0].EndColumn != 8 || strong[1].EndLine != 2 || strong[1].EndColumn != 15 || strong[2].EndLine != 2 || strong[2].EndColumn != 22 {
		t.Fatalf("Expected parse5 UTF-16 coordinates on reconstructed strong nodes, got %#v", strong)
	}
}

func TestParseFormattingReconstructionScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	type scalingCase struct {
		build        func(int) string
		small, large int
	}
	builders := map[string]scalingCase{
		"fixed reconstruction cycles": {build: func(n int) string {
			var b strings.Builder
			b.Grow(n * 27)
			for i := 0; i < n; i++ {
				b.WriteString(`<p><b>x<div>y</div>z</b>`)
			}
			return b.String()
		}, small: 250, large: 1000},
		"distinct nested formatting": {build: func(n int) string {
			var b strings.Builder
			b.Grow(n * 20)
			for i := 0; i < n; i++ {
				fmt.Fprintf(&b, `<b id=%d>`, i)
			}
			b.WriteByte('x')
			for i := 0; i < n; i++ {
				b.WriteString(`</b>`)
			}
			return b.String()
		}, small: 250, large: 1000},
		"distinct b then repeated i": {build: func(n int) string {
			var b strings.Builder
			b.Grow(n * 24)
			for i := 0; i < n; i++ {
				fmt.Fprintf(&b, `<b id=%d>`, i)
			}
			for i := 0; i < n; i++ {
				b.WriteString(`<i>`)
			}
			b.WriteByte('x')
			for i := 0; i < n; i++ {
				b.WriteString(`</i>`)
			}
			for i := 0; i < n; i++ {
				b.WriteString(`</b>`)
			}
			return b.String()
		}, small: 500, large: 2000},
	}
	measure := func(t *testing.T, build func(int) string, n int) time.Duration {
		t.Helper()
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
	for name, testCase := range builders {
		t.Run(name, func(t *testing.T) {
			small := measure(t, testCase.build, testCase.small)
			large := measure(t, testCase.build, testCase.large)
			ratio := float64(large) / float64(small)
			t.Logf("formatting scaling %d=%v %d=%v ratio=%.1fx", testCase.small, small, testCase.large, large, ratio)
			if large > small*10 && large-small > 20*time.Millisecond {
				t.Fatalf("Formatting recovery scaled superlinearly: %d=%v %d=%v ratio=%.1fx", testCase.small, small, testCase.large, large, ratio)
			}
		})
	}
}
