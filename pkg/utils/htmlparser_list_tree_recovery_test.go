package utils

import (
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// This batch is intentionally limited to in-body li/dd/dt recovery. Table and
// select insertion modes, template scope, adoption-agency formatting, foreign
// namespaces, fragment contexts, and scripting/document.write remain deferred.

func listElements(node *types.Node, name string) []*types.Node {
	var out []*types.Node
	if node == nil {
		return out
	}
	if node.Type == types.ElementNode && node.Name == name {
		out = append(out, node)
	}
	for _, child := range node.Children {
		out = append(out, listElements(child, name)...)
	}
	return out
}

func assertListNode(t *testing.T, node *types.Node, name, text string, start, end int) {
	t.Helper()
	if node == nil || node.Name != name || node.TextContent != text || node.StartPos != start || node.EndPos != end {
		t.Fatalf("Expected <%s> text %q at %d:%d, got %#v", name, text, start, end, node)
	}
}

func TestParseLIStartClosesPriorItemAndInlineDescendants(t *testing.T) {
	const simple = `<ul><li>a<li>b</ul>tail`
	document, err := NewHTMLParser().Parse(simple)
	if err != nil {
		t.Fatal(err)
	}
	items := listElements(document, "li")
	if len(items) != 2 {
		t.Fatalf("Expected two li siblings, got %#v", items)
	}
	assertListNode(t, items[0], "li", "a", 4, 9)
	assertListNode(t, items[1], "li", "b", 9, 14)
	if items[0].Parent != items[1].Parent {
		t.Fatalf("Expected li siblings")
	}
	assertListNode(t, listElements(document, "ul")[0], "ul", "ab", 0, 19)
	const inline = `<ul><li><span>a<li>b</ul>`
	document, err = NewHTMLParser().Parse(inline)
	if err != nil {
		t.Fatal(err)
	}
	items = listElements(document, "li")
	assertListNode(t, items[0], "li", "a", 4, 15)
	assertListNode(t, listElements(items[0], "span")[0], "span", "a", 8, 15)
	assertListNode(t, items[1], "li", "b", 15, 20)
}

func TestParseLIExplicitAbsentAncestorAndEOFRecovery(t *testing.T) {
	const explicit = `<ul><li>a</li>b</ul>`
	document, err := NewHTMLParser().Parse(explicit)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "li")[0], "li", "a", 4, 14)
	ul := listElements(document, "ul")[0]
	if len(ul.Children) != 2 || ul.Children[1].Type != types.TextNode || ul.Children[1].Value != "b" || ul.Children[1].StartPos != 14 || ul.Children[1].EndPos != 15 {
		t.Fatalf("Expected b after explicit li, got %#v", ul.Children)
	}
	const absent = `<ul><div>a</li>b</div><li>x</ul>`
	document, err = NewHTMLParser().Parse(absent)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "div")[0], "div", "ab", 4, 22)
	assertListNode(t, listElements(document, "li")[0], "li", "x", 22, 27)
	for _, tc := range []struct {
		content          string
		ulStart, liStart int
	}{{`<div><ul><li>a</ul>tail</div>`, 5, 9}, {`<div><ul><li>a`, 5, 9}} {
		document, err = NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatal(err)
		}
		end := len(tc.content)
		if strings.Contains(tc.content, "</ul>") {
			end = 19
		}
		assertListNode(t, listElements(document, "ul")[0], "ul", "a", tc.ulStart, end)
		liEnd := end
		if strings.Contains(tc.content, "</ul>") {
			liEnd = 14
		}
		assertListNode(t, listElements(document, "li")[0], "li", "a", tc.liStart, liEnd)
	}
}

func TestParseLINestedListsAndScopeBoundaries(t *testing.T) {
	const nested = `<ul><li>a<ul><li>b<li>c</ul>d</ul>`
	document, err := NewHTMLParser().Parse(nested)
	if err != nil {
		t.Fatal(err)
	}
	uls := listElements(document, "ul")
	items := listElements(document, "li")
	assertListNode(t, items[0], "li", "abcd", 4, 29)
	assertListNode(t, uls[1], "ul", "bc", 9, 28)
	assertListNode(t, items[1], "li", "b", 13, 18)
	assertListNode(t, items[2], "li", "c", 18, 23)
	const unwind = `<ul><li>a<div><li>b</div>c</ul>`
	document, err = NewHTMLParser().Parse(unwind)
	if err != nil {
		t.Fatal(err)
	}
	items = listElements(document, "li")
	assertListNode(t, items[0], "li", "a", 4, 14)
	assertListNode(t, listElements(items[0], "div")[0], "div", "", 9, 14)
	assertListNode(t, items[1], "li", "bc", 14, 26)
	const boundary = `<ul><li>a<section><li>b</section>c</ul>`
	document, err = NewHTMLParser().Parse(boundary)
	if err != nil {
		t.Fatal(err)
	}
	items = listElements(document, "li")
	assertListNode(t, items[0], "li", "abc", 4, 34)
	section := listElements(document, "section")[0]
	assertListNode(t, section, "section", "b", 9, 33)
	if items[1].Parent != section {
		t.Fatalf("Expected scope-boundary li to remain inside section")
	}
}

func TestParseDefinitionItemsStartEndScopeAncestorAndEOF(t *testing.T) {
	const starts = `<dl><dt>a<dd>b<dt>c</dl>tail`
	document, err := NewHTMLParser().Parse(starts)
	if err != nil {
		t.Fatal(err)
	}
	dt := listElements(document, "dt")
	dd := listElements(document, "dd")
	assertListNode(t, dt[0], "dt", "a", 4, 9)
	assertListNode(t, dd[0], "dd", "b", 9, 14)
	assertListNode(t, dt[1], "dt", "c", 14, 19)
	assertListNode(t, listElements(document, "dl")[0], "dl", "abc", 0, 24)
	const inline = `<dl><dt><span>a<dd><em>b</dl>`
	document, err = NewHTMLParser().Parse(inline)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "dt")[0], "dt", "a", 4, 15)
	assertListNode(t, listElements(document, "span")[0], "span", "a", 8, 15)
	assertListNode(t, listElements(document, "dd")[0], "dd", "b", 15, 24)
	assertListNode(t, listElements(document, "em")[0], "em", "b", 19, 24)
	const explicit = `<dl><dt>a</dt><dd>b</dd>z</dl>`
	document, err = NewHTMLParser().Parse(explicit)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "dt")[0], "dt", "a", 4, 14)
	assertListNode(t, listElements(document, "dd")[0], "dd", "b", 14, 24)
	const absent = `<dl><div>a</dd></dt>b</div><dt>x</dl>`
	document, err = NewHTMLParser().Parse(absent)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "div")[0], "div", "ab", 4, 27)
	assertListNode(t, listElements(document, "dt")[0], "dt", "x", 27, 32)
	for _, tc := range []struct {
		content, name                  string
		outerStart, itemStart, itemEnd int
	}{{`<section><dl><dd>a</dl>tail</section>`, "dd", 9, 13, 18}, {`<section><dl><dt>a`, "dt", 9, 13, 18}} {
		document, err = NewHTMLParser().Parse(tc.content)
		if err != nil {
			t.Fatal(err)
		}
		outerEnd := len(tc.content)
		if tc.name == "dd" {
			outerEnd = 23
		}
		assertListNode(t, listElements(document, "dl")[0], "dl", "a", tc.outerStart, outerEnd)
		assertListNode(t, listElements(document, tc.name)[0], tc.name, "a", tc.itemStart, tc.itemEnd)
	}
}

func TestParseDefinitionItemBlockUnwindAndScopeBoundary(t *testing.T) {
	const unwind = `<dl><dt>a<div><dd>b</div>c</dl>`
	document, err := NewHTMLParser().Parse(unwind)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "dt")[0], "dt", "a", 4, 14)
	assertListNode(t, listElements(document, "div")[0], "div", "", 9, 14)
	assertListNode(t, listElements(document, "dd")[0], "dd", "bc", 14, 26)
	const boundary = `<dl><dt>a<section><dd>b</section>c</dl>`
	document, err = NewHTMLParser().Parse(boundary)
	if err != nil {
		t.Fatal(err)
	}
	dt := listElements(document, "dt")[0]
	assertListNode(t, dt, "dt", "abc", 4, 34)
	section := listElements(document, "section")[0]
	assertListNode(t, section, "section", "b", 9, 33)
	if listElements(document, "dd")[0].Parent != section {
		t.Fatalf("Expected dd to remain in scope-boundary section")
	}
}

func TestParseListRecoveryPreservesMultibyteCRLFLocations(t *testing.T) {
	const content = "<ul>\r\n<li>é\r\n<span>x<li>y</ul>"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	items := listElements(document, "li")
	assertListNode(t, items[0], "li", "é\nx", 6, 21)
	assertListNode(t, listElements(items[0], "span")[0], "span", "x", 14, 21)
	assertListNode(t, items[1], "li", "y", 21, 26)
	if items[0].StartLine != 2 || items[0].StartColumn != 1 || items[0].EndLine != 3 || items[0].EndColumn != 8 {
		t.Fatalf("Expected browser UTF-16 coords 2:1-3:8, got %d:%d-%d:%d", items[0].StartLine, items[0].StartColumn, items[0].EndLine, items[0].EndColumn)
	}
}

func TestParseListExplicitEndsGenerateImpliedEndTags(t *testing.T) {
	t.Run("li", func(t *testing.T) {
		const content = `<ul><li><dd><p>x</li>y</ul>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		li := listElements(document, "li")[0]
		dd := listElements(document, "dd")[0]
		p := listElements(document, "p")[0]
		assertListNode(t, li, "li", "x", 4, 21)
		assertListNode(t, dd, "dd", "x", 8, 16)
		assertListNode(t, p, "p", "x", 12, 16)
		if li.ContentEnd != 16 || dd.ContentEnd != 16 || p.ContentEnd != 16 {
			t.Fatalf("Expected descendants to end at li end-tag start, got li=%d dd=%d p=%d", li.ContentEnd, dd.ContentEnd, p.ContentEnd)
		}
	})

	t.Run("dd", func(t *testing.T) {
		const content = `<dl><dd><li><p>x</dd>y</dl>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		dd := listElements(document, "dd")[0]
		li := listElements(document, "li")[0]
		p := listElements(document, "p")[0]
		assertListNode(t, dd, "dd", "x", 4, 21)
		assertListNode(t, li, "li", "x", 8, 16)
		assertListNode(t, p, "p", "x", 12, 16)
		if dd.ContentEnd != 16 || li.ContentEnd != 16 || p.ContentEnd != 16 {
			t.Fatalf("Expected descendants to end at dd end-tag start, got dd=%d li=%d p=%d", dd.ContentEnd, li.ContentEnd, p.ContentEnd)
		}
	})
}

func TestParseListItemStartScanExceptionsAndSpecialBoundaries(t *testing.T) {
	for _, testCase := range []struct {
		name, content, child string
		trigger              int
	}{
		{name: "div", content: `<ul><li>a<div>b<li>c</ul>`, child: "div", trigger: 15},
		{name: "custom", content: `<ul><li>a<x-box>b<li>c</ul>`, child: "x-box", trigger: 17},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			items := listElements(document, "li")
			if len(items) != 2 || items[0].Parent != items[1].Parent {
				t.Fatalf("Expected two sibling li nodes, got %#v", items)
			}
			if items[0].EndPos != testCase.trigger || listElements(items[0], testCase.child)[0].EndPos != testCase.trigger {
				t.Fatalf("Expected li and child to unwind at %d", testCase.trigger)
			}
		})
	}

	const special = `<ul><li>a<section>b<li>c</section>d</ul>`
	document, err := NewHTMLParser().Parse(special)
	if err != nil {
		t.Fatal(err)
	}
	items := listElements(document, "li")
	assertListNode(t, items[0], "li", "abcd", 4, 35)
	assertListNode(t, items[1], "li", "c", 19, 24)
	if items[1].Parent.Name != "section" {
		t.Fatalf("Expected special section boundary to keep nested li, got %#v", items[1].Parent)
	}
}

func TestParseDefinitionItemStartCrossNameAndBoundaries(t *testing.T) {
	const crossName = `<dl><dd>a<dt>b<dd>c</dl>`
	document, err := NewHTMLParser().Parse(crossName)
	if err != nil {
		t.Fatal(err)
	}
	dd := listElements(document, "dd")
	dt := listElements(document, "dt")
	assertListNode(t, dd[0], "dd", "a", 4, 9)
	assertListNode(t, dt[0], "dt", "b", 9, 14)
	assertListNode(t, dd[1], "dd", "c", 14, 19)

	const exception = `<dl><dt>a<div>b<dd>c</dl>`
	document, err = NewHTMLParser().Parse(exception)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "dt")[0], "dt", "ab", 4, 15)
	assertListNode(t, listElements(document, "div")[0], "div", "b", 9, 15)
	assertListNode(t, listElements(document, "dd")[0], "dd", "c", 15, 20)

	const boundary = `<dl><dt>a<section>b<dd>c</section>d</dl>`
	document, err = NewHTMLParser().Parse(boundary)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "dt")[0], "dt", "abcd", 4, 35)
	assertListNode(t, listElements(document, "dd")[0], "dd", "c", 19, 24)
}

func TestParseListAndDefinitionStartsDoNotCrossSpecialFamilies(t *testing.T) {
	const list = `<ul><li>a<dd>b<li>c</ul>`
	document, err := NewHTMLParser().Parse(list)
	if err != nil {
		t.Fatal(err)
	}
	items := listElements(document, "li")
	assertListNode(t, items[0], "li", "abc", 4, 19)
	assertListNode(t, listElements(document, "dd")[0], "dd", "bc", 9, 19)
	if items[1].Parent.Name != "dd" {
		t.Fatalf("Expected dd to stop li ancestor scan")
	}

	const definition = `<dl><dd>a<li>b<dt>c</dl>`
	document, err = NewHTMLParser().Parse(definition)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "dd")[0], "dd", "abc", 4, 19)
	assertListNode(t, listElements(document, "li")[0], "li", "bc", 9, 19)
	if listElements(document, "dt")[0].Parent.Name != "li" {
		t.Fatalf("Expected li to stop definition-item ancestor scan")
	}
}

func TestParseListEndTagScopeAndIgnoredTokenRanges(t *testing.T) {
	t.Run("li stopped by nested ul", func(t *testing.T) {
		const content = `<ul><li>a<ul></li>b</ul>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		uls := listElements(document, "ul")
		li := listElements(document, "li")[0]
		assertListNode(t, li, "li", "ab", 4, len(content))
		assertListNode(t, uls[1], "ul", "b", 9, len(content))
	})

	t.Run("dd crosses nested ul", func(t *testing.T) {
		const content = `<dl><dd>a<ul>b</dd>c</dl>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		assertListNode(t, listElements(document, "dd")[0], "dd", "ab", 4, 19)
		assertListNode(t, listElements(document, "ul")[0], "ul", "b", 9, 14)
	})

	t.Run("dd stopped by object", func(t *testing.T) {
		const content = `<dl><dd>a<object>b</dd>c</object>d</dl>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		assertListNode(t, listElements(document, "dd")[0], "dd", "abcd", 4, 34)
		assertListNode(t, listElements(document, "object")[0], "object", "bc", 9, 33)
	})

	t.Run("wrong name merges adjacent text", func(t *testing.T) {
		const content = `<dl><dt>a</dd>b</dt>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		dt := listElements(document, "dt")[0]
		assertListNode(t, dt, "dt", "ab", 4, 20)
		if len(dt.Children) != 1 || dt.Children[0].Value != "ab" || dt.Children[0].StartPos != 8 || dt.Children[0].EndPos != 15 {
			t.Fatalf("Expected ignored token inside merged raw text range, got %#v", dt.Children)
		}
	})

	t.Run("comment prevents merge across ignored end", func(t *testing.T) {
		const content = `<div>a</li><!--c-->b</div>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		div := listElements(document, "div")[0]
		if len(div.Children) != 3 || div.Children[0].Value != "a" || div.Children[0].StartPos != 5 || div.Children[0].EndPos != 6 || div.Children[2].Value != "b" || div.Children[2].StartPos != 19 || div.Children[2].EndPos != 20 {
			t.Fatalf("Expected comment to separate text around ignored li end, got %#v", div.Children)
		}
	})
}

func TestParseListStartClosesParagraphOnlyInButtonScope(t *testing.T) {
	for _, testCase := range []struct {
		content        string
		pStart, pEnd   int
		liStart, liEnd int
	}{
		{content: `<p>a<li>b`, pStart: 0, pEnd: 4, liStart: 4, liEnd: 9},
		{content: `<ul><li><p>a<li>b</ul>`, pStart: 8, pEnd: 12, liStart: 12, liEnd: 17},
		{content: `<button><p>a<li>b`, pStart: 8, pEnd: 12, liStart: 12, liEnd: 17},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		assertListNode(t, listElements(document, "p")[0], "p", "a", testCase.pStart, testCase.pEnd)
		items := listElements(document, "li")
		assertListNode(t, items[len(items)-1], "li", "b", testCase.liStart, testCase.liEnd)
	}

	const outside = `<p>a<button>b<li>c</button>d`
	document, err := NewHTMLParser().Parse(outside)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "p")[0], "p", "abcd", 0, len(outside))
	assertListNode(t, listElements(document, "li")[0], "li", "c", 13, 18)
}

func TestParseListTokenizerFormsAndIncompleteEOF(t *testing.T) {
	for _, testCase := range []struct {
		name, content string
		firstEnd      int
	}{
		{name: "uppercase", content: `<UL><LI>a<LI>b</UL>`, firstEnd: 9},
		{name: "self closing flag ignored", content: `<ul><li/>a<li>b</ul>`, firstEnd: 10},
		{name: "end attributes and slash", content: `<ul><li>a</li x/><li>b</ul>`, firstEnd: 17},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			items := listElements(document, "li")
			assertListNode(t, items[0], "li", "a", 4, testCase.firstEnd)
			if len(items) != 2 || items[1].TextContent != "b" {
				t.Fatalf("Expected second li b, got %#v", items)
			}
		})
	}

	for _, content := range []string{`<ul><li>a<li`, `<ul><li>a</li`} {
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		items := listElements(document, "li")
		if len(items) != 1 {
			t.Fatalf("Expected discarded incomplete token, got %#v", items)
		}
		assertListNode(t, items[0], "li", "a", 4, len(content))
		if items[0].Children[0].StartPos != 8 || items[0].Children[0].EndPos != len(content) {
			t.Fatalf("Expected prior text raw range to extend through discarded token, got %#v", items[0].Children[0])
		}
	}
}

func TestParseListRecoveryRootCommentsAndSupplementaryCRLFLocations(t *testing.T) {
	for _, testCase := range []struct {
		content, first, second string
		firstStart, firstEnd   int
	}{
		{content: `<li>a<li>b`, first: "li", second: "li", firstStart: 0, firstEnd: 5},
		{content: `<dd>a<dt>b`, first: "dd", second: "dt", firstStart: 0, firstEnd: 5},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		assertListNode(t, listElements(document, testCase.first)[0], testCase.first, "a", testCase.firstStart, testCase.firstEnd)
		secondIndex := 0
		if testCase.first == testCase.second {
			secondIndex = 1
		}
		assertListNode(t, listElements(document, testCase.second)[secondIndex], testCase.second, "b", 5, 10)
	}

	const comment = `<ul><li>a<!--c--><li>b</ul>`
	document, err := NewHTMLParser().Parse(comment)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "li")[0], "li", "a", 4, 17)

	const multiline = "<ul>\r\n<li>😀é\r\n<span>x<li>z</ul>"
	document, err = NewHTMLParser().Parse(multiline)
	if err != nil {
		t.Fatal(err)
	}
	items := listElements(document, "li")
	assertListNode(t, items[0], "li", "😀é\nx", 6, 25)
	assertListNode(t, listElements(items[0], "span")[0], "span", "x", 18, 25)
	assertListNode(t, items[1], "li", "z", 25, 30)
	if items[0].StartLine != 2 || items[0].StartColumn != 1 || items[0].EndLine != 3 || items[0].EndColumn != 8 {
		t.Fatalf("Expected parse5 UTF-16 coords 2:1-3:8, got %d:%d-%d:%d", items[0].StartLine, items[0].StartColumn, items[0].EndLine, items[0].EndColumn)
	}
}
