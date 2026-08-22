package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseNestedSelectRestoresStandaloneOptionAndOptgroupIdentity(t *testing.T) {
	const optionCase = `<option>a<select><option>b</select>c`
	document, err := NewHTMLParser().Parse(optionCase)
	if err != nil {
		t.Fatal(err)
	}
	options := selectElements(document, "option")
	selectNode := selectElements(document, "select")[0]
	assertSelectNode(t, options[0], "option", "abc", 0, len(optionCase))
	assertSelectNode(t, selectNode, "select", "b", 9, 35)
	assertSelectNode(t, options[1], "option", "b", 17, 26)
	if selectNode.Parent != options[0] || options[1].Parent != selectNode || len(options[0].Children) != 3 || options[0].Children[2].Type != types.TextNode || options[0].Children[2].Value != "c" || options[0].Children[2].StartPos != 35 || options[0].Children[2].EndPos != 36 {
		t.Fatalf("Expected nested select option state restored to outer option, got outer=%#v inner=%#v", options[0], options[1])
	}

	const groupCase = `<optgroup>a<select><optgroup>b</select>c`
	document, err = NewHTMLParser().Parse(groupCase)
	if err != nil {
		t.Fatal(err)
	}
	groups := selectElements(document, "optgroup")
	selectNode = selectElements(document, "select")[0]
	assertSelectNode(t, groups[0], "optgroup", "abc", 0, len(groupCase))
	assertSelectNode(t, selectNode, "select", "b", 11, 39)
	assertSelectNode(t, groups[1], "optgroup", "b", 19, 30)
	if selectNode.Parent != groups[0] || groups[1].Parent != selectNode || len(groups[0].Children) != 3 || groups[0].Children[2].Value != "c" || groups[0].Children[2].StartPos != 39 || groups[0].Children[2].EndPos != 40 {
		t.Fatalf("Expected nested select optgroup state restored to outer optgroup, got outer=%#v inner=%#v", groups[0], groups[1])
	}
}

func TestParseSelectDeepDescendantEndScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	builders := map[string]func(int) string{
		"option end": func(depth int) string {
			var b strings.Builder
			b.Grow(depth*6 + 40)
			b.WriteString(`<select><option>`)
			for i := 0; i < depth; i++ {
				b.WriteString(`<span>`)
			}
			b.WriteString(`x</option>y</select>`)
			return b.String()
		},
		"optgroup end": func(depth int) string {
			var b strings.Builder
			b.Grow(depth*6 + 44)
			b.WriteString(`<select><optgroup>`)
			for i := 0; i < depth; i++ {
				b.WriteString(`<span>`)
			}
			b.WriteString(`x</optgroup>y</select>`)
			return b.String()
		},
	}
	measure := func(build func(int) string, depth int) time.Duration {
		content := build(depth)
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
			t.Logf("select descendant unwind scaling 500=%v 2000=%v ratio=%.1fx", small, large, ratio)
			if large > small*10 && large-small > 10*time.Millisecond {
				t.Fatalf("Select descendant unwind scaled superlinearly: 500=%v 2000=%v ratio=%.1fx", small, large, ratio)
			}
		})
	}
}

func TestParseSelectDeepIgnoredEndScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	builders := map[string]func(int) string{
		"absent end": func(n int) string {
			var b strings.Builder
			b.Grow(n*13 + 24)
			b.WriteString(`<select>`)
			for i := 0; i < n; i++ {
				b.WriteString(`<span>`)
			}
			b.WriteByte('x')
			for i := 0; i < n; i++ {
				b.WriteString(`</foo>`)
			}
			b.WriteString(`</select>`)
			return b.String()
		},
		"outside ancestor end": func(n int) string {
			var b strings.Builder
			b.Grow(n*13 + 35)
			b.WriteString(`<div><select>`)
			for i := 0; i < n; i++ {
				b.WriteString(`<span>`)
			}
			b.WriteByte('x')
			for i := 0; i < n; i++ {
				b.WriteString(`</div>`)
			}
			b.WriteString(`</select></div>`)
			return b.String()
		},
	}
	measure := func(build func(int) string, n int) time.Duration {
		content := build(n)
		if _, err := NewHTMLParser().Parse(content); err != nil {
			t.Fatalf("warmup parse failed: %v", err)
		}
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
			t.Logf("select deep ignored-end scaling 500=%v 2000=%v ratio=%.1fx", small, large, ratio)
			if large > small*10 && large-small > 10*time.Millisecond {
				t.Fatalf("Select ignored-end recovery scaled superlinearly: 500=%v 2000=%v ratio=%.1fx", small, large, ratio)
			}
		})
	}
}

func TestParseSearchDescendantDoesNotBlockExplicitSelectItemEnd(t *testing.T) {
	for _, testCase := range []struct {
		content, item                              string
		itemEnd, searchStart, searchEnd, tailStart int
	}{
		{content: `<select><option><search>x</option>y</select>`, item: "option", itemEnd: 34, searchStart: 16, searchEnd: 25, tailStart: 34},
		{content: `<select><optgroup><search>x</optgroup>y</select>`, item: "optgroup", itemEnd: 38, searchStart: 18, searchEnd: 27, tailStart: 38},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		selectNode := selectElements(document, "select")[0]
		item := selectElements(selectNode, testCase.item)[0]
		search := selectElements(item, "search")[0]
		assertSelectNode(t, item, testCase.item, "x", 8, testCase.itemEnd)
		assertSelectNode(t, search, "search", "x", testCase.searchStart, testCase.searchEnd)
		if len(selectNode.Children) != 2 || selectNode.Children[0] != item || selectNode.Children[1].Type != types.TextNode || selectNode.Children[1].Value != "y" || selectNode.Children[1].StartPos != testCase.tailStart || selectNode.Children[1].EndPos != testCase.tailStart+1 {
			t.Fatalf("Expected search unwind and y directly under select, got %#v", selectNode.Children)
		}
	}
}

func TestParseGenericEndsInsideSelectIgnoreAbsentAndCloseMatchingDescendant(t *testing.T) {
	const absent = `<select><option>a</foo>b</select>`
	document, err := NewHTMLParser().Parse(absent)
	if err != nil {
		t.Fatal(err)
	}
	option := selectElements(document, "option")[0]
	assertSelectNode(t, option, "option", "ab", 8, 24)
	if len(option.Children) != 1 || option.Children[0].Value != "ab" || option.Children[0].StartPos != 16 || option.Children[0].EndPos != 24 {
		t.Fatalf("Expected absent custom end ignored inside coalesced option text, got %#v", option.Children)
	}

	const matching = `<select><div><span>x</div>y</select>`
	document, err = NewHTMLParser().Parse(matching)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	div := selectElements(selectNode, "div")[0]
	span := selectElements(div, "span")[0]
	assertSelectNode(t, div, "div", "x", 8, 26)
	assertSelectNode(t, span, "span", "x", 13, 20)
	if len(selectNode.Children) != 2 || selectNode.Children[0] != div || selectNode.Children[1].Type != types.TextNode || selectNode.Children[1].Value != "y" || selectNode.Children[1].StartPos != 26 || selectNode.Children[1].EndPos != 27 {
		t.Fatalf("Expected matching div end to unwind span and put y directly under select, got %#v", selectNode.Children)
	}
}

func TestParseSelectMatchingEndUsesNearestOpenElementIdentity(t *testing.T) {
	const duplicateSpan = `<select><span>x</span>y</span>z</select>`
	document, err := NewHTMLParser().Parse(duplicateSpan)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	spans := selectElements(selectNode, "span")
	assertSelectNode(t, selectNode, "select", "xyz", 0, len(duplicateSpan))
	if len(spans) != 1 {
		t.Fatalf("Expected exactly one explicitly closed span, got %#v", spans)
	}
	assertSelectNode(t, spans[0], "span", "x", 8, 22)
	if len(selectNode.Children) != 2 || selectNode.Children[1].Type != types.TextNode || selectNode.Children[1].Value != "yz" || selectNode.Children[1].StartPos != 22 || selectNode.Children[1].EndPos != 31 {
		t.Fatalf("Expected second span end ignored within coalesced yz source range, got %#v", selectNode.Children)
	}

	const duplicateDiv = `<select><div><div>x</div>y</div>z</div>w</select>`
	document, err = NewHTMLParser().Parse(duplicateDiv)
	if err != nil {
		t.Fatal(err)
	}
	selectNode = selectElements(document, "select")[0]
	divs := selectElements(selectNode, "div")
	assertSelectNode(t, selectNode, "select", "xyzw", 0, len(duplicateDiv))
	if len(divs) != 2 || divs[1].Parent != divs[0] {
		t.Fatalf("Expected exactly two nested divs, got %#v", divs)
	}
	assertSelectNode(t, divs[0], "div", "xy", 8, 32)
	assertSelectNode(t, divs[1], "div", "x", 13, 25)
	if len(selectNode.Children) != 2 || selectNode.Children[1].Value != "zw" || selectNode.Children[1].StartPos != 32 || selectNode.Children[1].EndPos != 40 {
		t.Fatalf("Expected third div end ignored after nearest two divs close, got %#v", selectNode.Children)
	}

	parser := NewHTMLParser()
	for _, content := range []string{duplicateDiv, duplicateSpan, duplicateDiv} {
		reused, parseErr := parser.Parse(content)
		if parseErr != nil {
			t.Fatalf("Reused parser failed for %q: %v", content, parseErr)
		}
		if got := selectElements(reused, "select")[0].TextContent; got != map[string]string{duplicateSpan: "xyz", duplicateDiv: "xyzw"}[content] {
			t.Fatalf("Reused parser leaked element identity for %q: text=%q", content, got)
		}
	}
}

func TestParseSelectGenericEndCannotCrossSpecialElementBarrier(t *testing.T) {
	for _, testCase := range []struct {
		content                          string
		outerName, barrierName, wantText string
		outerStart, barrierStart, end    int
		textStart                        int
	}{
		{content: `<select><span><div>x</span>y</select>`, outerName: "span", barrierName: "div", wantText: "xy", outerStart: 8, barrierStart: 14, end: 28, textStart: 19},
		{content: `<select><foo><section>x</foo>y</select>`, outerName: "foo", barrierName: "section", wantText: "xy", outerStart: 8, barrierStart: 13, end: 30, textStart: 22},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", testCase.content, err)
		}
		selectNode := selectElements(document, "select")[0]
		outer := selectElements(selectNode, testCase.outerName)[0]
		barrier := selectElements(outer, testCase.barrierName)[0]
		assertSelectNode(t, outer, testCase.outerName, testCase.wantText, testCase.outerStart, testCase.end)
		assertSelectNode(t, barrier, testCase.barrierName, testCase.wantText, testCase.barrierStart, testCase.end)
		if len(barrier.Children) != 1 || barrier.Children[0].Type != types.TextNode || barrier.Children[0].Value != testCase.wantText || barrier.Children[0].StartPos != testCase.textStart || barrier.Children[0].EndPos != testCase.end {
			t.Fatalf("Expected ignored outer end coalesced inside special barrier for %q, got %#v", testCase.content, barrier.Children)
		}
	}
}

func TestParseSelectDedicatedSpecialEndUnwindsButGenericEndStopsAtBarrier(t *testing.T) {
	const dedicated = `<select><div><section>x</div>y`
	document, err := NewHTMLParser().Parse(dedicated)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	div := selectElements(selectNode, "div")[0]
	section := selectElements(div, "section")[0]
	assertSelectNode(t, selectNode, "select", "xy", 0, len(dedicated))
	assertSelectNode(t, div, "div", "x", 8, 29)
	assertSelectNode(t, section, "section", "x", 13, 23)
	if len(selectNode.Children) != 2 || selectNode.Children[0] != div || selectNode.Children[1].Type != types.TextNode || selectNode.Children[1].Value != "y" || selectNode.Children[1].StartPos != 29 || selectNode.Children[1].EndPos != 30 {
		t.Fatalf("Expected dedicated div end to unwind section and put y under select, got %#v", selectNode.Children)
	}

	const generic = `<select><foo><section>x</foo>y`
	document, err = NewHTMLParser().Parse(generic)
	if err != nil {
		t.Fatal(err)
	}
	selectNode = selectElements(document, "select")[0]
	foo := selectElements(selectNode, "foo")[0]
	section = selectElements(foo, "section")[0]
	assertSelectNode(t, selectNode, "select", "xy", 0, len(generic))
	assertSelectNode(t, foo, "foo", "xy", 8, len(generic))
	assertSelectNode(t, section, "section", "xy", 13, len(generic))
	if len(section.Children) != 1 || section.Children[0].Value != "xy" || section.Children[0].StartPos != 22 || section.Children[0].EndPos != len(generic) {
		t.Fatalf("Expected generic foo end ignored behind section barrier, got %#v", section.Children)
	}
}
