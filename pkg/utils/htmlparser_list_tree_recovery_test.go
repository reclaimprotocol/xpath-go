package utils

import (
	"strings"
	"testing"
	"time"

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

func TestParseListItemsCloseWithMatchingAncestorEnds(t *testing.T) {
	for _, testCase := range []struct {
		content, itemName, itemText, ancestorName      string
		itemStart, itemEnd, ancestorStart, ancestorEnd int
	}{
		{content: `<div><li>x</div>y`, itemName: "li", itemText: "x", ancestorName: "div", itemStart: 5, itemEnd: 10, ancestorStart: 0, ancestorEnd: 16},
		{content: `<section><dd>x</section>y`, itemName: "dd", itemText: "x", ancestorName: "section", itemStart: 9, itemEnd: 14, ancestorStart: 0, ancestorEnd: 24},
		{content: `<ul><li>a<div>b</ul>y`, itemName: "li", itemText: "ab", ancestorName: "ul", itemStart: 4, itemEnd: 15, ancestorStart: 0, ancestorEnd: 20},
	} {
		t.Run(testCase.itemName+testCase.ancestorName, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			assertListNode(t, listElements(document, testCase.itemName)[0], testCase.itemName, testCase.itemText, testCase.itemStart, testCase.itemEnd)
			ancestor := listElements(document, testCase.ancestorName)[0]
			if ancestor.StartPos != testCase.ancestorStart || ancestor.EndPos != testCase.ancestorEnd {
				t.Fatalf("Expected <%s> at %d:%d, got %#v", testCase.ancestorName, testCase.ancestorStart, testCase.ancestorEnd, ancestor)
			}
		})
	}

	const blocked = `<span><li>x</span>y`
	document, err := NewHTMLParser().Parse(blocked)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "li")[0], "li", "xy", 6, len(blocked))
	assertListNode(t, listElements(document, "span")[0], "span", "xy", 0, len(blocked))
}

func TestParseListItemEOFPropagationAndRootStrayEnds(t *testing.T) {
	for _, testCase := range []struct {
		content, itemName string
		itemStart         int
	}{
		{content: `<li>x`, itemName: "li", itemStart: 0},
		{content: `<dd>x`, itemName: "dd", itemStart: 0},
		{content: `<dt>x`, itemName: "dt", itemStart: 0},
		{content: `<div><ul><li>x`, itemName: "li", itemStart: 9},
		{content: `<section><dl><dt>x`, itemName: "dt", itemStart: 13},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		assertListNode(t, listElements(document, testCase.itemName)[0], testCase.itemName, "x", testCase.itemStart, len(testCase.content))
	}

	for _, content := range []string{`</li>`, `</dd>`, `</dt>`} {
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		if len(listElements(document, content[2:4])) != 0 {
			t.Fatalf("Expected stray %s to create no synthetic item", content)
		}
	}
	if document, err := NewHTMLParser().Parse(`<div>x`); err != nil || len(listElements(document, "div")) != 1 || listElements(document, "div")[0].EndPos != 6 {
		t.Fatalf("List-to-implicit-document EOF recovery mismatch: doc=%#v err=%v", document, err)
	}
}

func TestParseListRecoveryParserReuseAndScaling(t *testing.T) {
	parser := NewHTMLParser()
	for _, testCase := range []struct {
		content, first, second string
	}{
		{content: `<ul><li>a<li>b</ul>`, first: "li", second: "li"},
		{content: `<dl><dt>a<dd>b</dl>`, first: "dt", second: "dd"},
	} {
		document, err := parser.Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		first := listElements(document, testCase.first)
		second := listElements(document, testCase.second)
		secondIndex := 0
		if testCase.first == testCase.second {
			secondIndex = 1
		}
		if len(first) == 0 || len(second) <= secondIndex || first[0].TextContent != "a" || second[secondIndex].TextContent != "b" {
			t.Fatalf("Expected parser reuse items a and b, got first=%#v second=%#v", first, second)
		}
	}
	if testing.Short() {
		return
	}
	measure := func(build func(int) string, n int) time.Duration {
		content := build(n)
		_, _ = NewHTMLParser().Parse(content)
		best := time.Duration(1<<63 - 1)
		for i := 0; i < 5; i++ {
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
	builders := map[string]func(int) string{
		"many li": func(n int) string {
			var b strings.Builder
			b.WriteString(`<ul>`)
			for i := 0; i < n; i++ {
				b.WriteString(`<li>x`)
			}
			b.WriteString(`</ul>`)
			return b.String()
		},
		"inline and div scan with alternating dd dt": func(n int) string {
			var b strings.Builder
			b.WriteString(`<dl>`)
			for i := 0; i < n; i++ {
				if i%2 == 0 {
					b.WriteString(`<dt><div><span>x`)
				} else {
					b.WriteString(`<dd><div><span>x`)
				}
			}
			b.WriteString(`</dl>`)
			return b.String()
		},
	}
	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			small, large := measure(build, 250), measure(build, 1000)
			if large > small*17/2 && large-small > 5*time.Millisecond {
				t.Fatalf("Recovery scaled superlinearly: 250=%v 1000=%v ratio=%.1fx", small, large, float64(large)/float64(small))
			}
			t.Logf("scaling 250=%v 1000=%v ratio=%.1fx", small, large, float64(large)/float64(small))
		})
	}
}

func TestParseListRecoveryUsesAncestorIdentityForDuplicateNames(t *testing.T) {
	const content = `<div><li><div>x</div>y</li>z</div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	divs := listElements(document, "div")
	if len(divs) != 2 {
		t.Fatalf("Expected outer and inner div, got %#v", divs)
	}
	assertListNode(t, divs[0], "div", "xyz", 0, 34)
	assertListNode(t, listElements(document, "li")[0], "li", "xy", 5, 27)
	assertListNode(t, divs[1], "div", "x", 9, 21)
	if divs[1].Parent.Name != "li" || divs[0].Children[len(divs[0].Children)-1].Value != "z" {
		t.Fatalf("Expected inner div in li and z directly under outer div, got outer=%#v inner=%#v", divs[0], divs[1])
	}
}

func TestParseListRecoveryClosesValidDescendantBlocks(t *testing.T) {
	for _, testCase := range []struct {
		name, content, itemName string
	}{
		{name: "li", content: `<ul><li><section><span>x</section>y</li></ul>`, itemName: "li"},
		{name: "dd", content: `<dl><dd><section><span>x</section>y</dd></dl>`, itemName: "dd"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			item := listElements(document, testCase.itemName)[0]
			section := listElements(document, "section")[0]
			span := listElements(document, "span")[0]
			assertListNode(t, item, testCase.itemName, "xy", 4, 40)
			assertListNode(t, section, "section", "x", 8, 34)
			assertListNode(t, span, "span", "x", 17, 24)
			if len(item.Children) != 2 || item.Children[0] != section || item.Children[1].Type != types.TextNode || item.Children[1].Value != "y" || item.Children[1].StartPos != 34 || item.Children[1].EndPos != 35 {
				t.Fatalf("Expected y directly under item after section close, got %#v", item.Children)
			}
		})
	}
}

func TestParseExplicitListEndDoesNotLeakEOFRecovery(t *testing.T) {
	for _, content := range []string{
		`<ul><li>x</li></ul><div>y`,
		`<dl><dd>x</dd></dl><section>y`,
	} {
		if document, err := NewHTMLParser().Parse(content); err != nil || len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].EndPos != len(content) {
			t.Fatalf("Expected implicit-document EOF recovery after explicit list close for %q: doc=%#v err=%v", content, document, err)
		}
	}

	for _, content := range []string{`<div><ul><li>x`, `<section><dl><dt>x`} {
		if _, err := NewHTMLParser().Parse(content); err != nil {
			t.Fatalf("Expected active list-item EOF recovery to remain valid for %q: %v", content, err)
		}
	}
}

func TestParseSearchDoesNotStopListItemStartScanLikeBrowsers(t *testing.T) {
	t.Run("li", func(t *testing.T) {
		const content = `<ul><li>a<search>b<li>c</search>d</ul>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		items := listElements(document, "li")
		if len(items) != 2 || items[0].Parent != items[1].Parent || items[0].Parent.Name != "ul" {
			t.Fatalf("Expected two sibling li nodes under ul, got %#v", items)
		}
		assertListNode(t, items[0], "li", "ab", 4, 18)
		assertListNode(t, listElements(items[0], "search")[0], "search", "b", 9, 18)
		assertListNode(t, items[1], "li", "cd", 18, 33)
	})

	t.Run("definition item", func(t *testing.T) {
		const content = `<dl><dt>a<search>b<dd>c</search>d</dl>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		dt := listElements(document, "dt")[0]
		dd := listElements(document, "dd")[0]
		if dt.Parent != dd.Parent || dt.Parent.Name != "dl" {
			t.Fatalf("Expected sibling dt and dd under dl, got dt=%#v dd=%#v", dt, dd)
		}
		assertListNode(t, dt, "dt", "ab", 4, 18)
		assertListNode(t, listElements(dt, "search")[0], "search", "b", 9, 18)
		assertListNode(t, dd, "dd", "cd", 18, 33)
	})
}

func TestParsePendingListItemsSurviveBodyAndHTMLEnds(t *testing.T) {
	for _, testCase := range []struct {
		name, content, container, item                  string
		htmlEnd, bodyEnd, itemStart, textStart, textEnd int
	}{
		{name: "ul li", content: `<html><body><ul><li>x</body>y</html>`, container: "ul", item: "li", htmlEnd: 36, bodyEnd: 28, itemStart: 16, textStart: 20, textEnd: 29},
		{name: "dl dd", content: `<html><body><dl><dd>x</html>y`, container: "dl", item: "dd", htmlEnd: 28, bodyEnd: 21, itemStart: 16, textStart: 20, textEnd: 29},
		{name: "bare li", content: `<html><body><li>x</body>y</html>`, item: "li", htmlEnd: 32, bodyEnd: 24, itemStart: 12, textStart: 16, textEnd: 25},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			html := listElements(document, "html")[0]
			body := listElements(document, "body")[0]
			item := listElements(document, testCase.item)[0]
			assertListNode(t, html, "html", "xy", 0, testCase.htmlEnd)
			assertListNode(t, body, "body", "xy", 6, testCase.bodyEnd)
			assertListNode(t, item, testCase.item, "xy", testCase.itemStart, len(testCase.content))
			if len(item.Children) != 1 || item.Children[0].Type != types.TextNode || item.Children[0].Value != "xy" || item.Children[0].StartPos != testCase.textStart || item.Children[0].EndPos != testCase.textEnd {
				t.Fatalf("Expected body/html end tokens to stay in merged item text range, got %#v", item.Children)
			}
			if testCase.container == "" {
				if item.Parent != body {
					t.Fatalf("Expected bare item to remain under body, got %#v", item.Parent)
				}
			} else {
				container := listElements(document, testCase.container)[0]
				assertListNode(t, container, testCase.container, "xy", 12, len(testCase.content))
				if item.Parent != container || container.Parent != body {
					t.Fatalf("Expected item/container/body ancestry, got item=%#v container=%#v", item.Parent, container.Parent)
				}
			}
		})
	}
}

func TestParseMismatchedHeadingEndClosesCurrentHeadingAndItem(t *testing.T) {
	for _, testCase := range []struct {
		content, heading, item string
	}{
		{content: `<h1><li>x</h2>y`, heading: "h1", item: "li"},
		{content: `<h2><dd>x</h6>y`, heading: "h2", item: "dd"},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		heading := listElements(document, testCase.heading)[0]
		item := listElements(document, testCase.item)[0]
		assertListNode(t, heading, testCase.heading, "x", 0, 9)
		assertListNode(t, item, testCase.item, "x", 4, 9)
		if item.Parent != heading || len(parsedBodyChildren(document)) < 2 || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].Type != types.TextNode || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].Value != "y" || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].StartPos != 14 || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].EndPos != 15 {
			t.Fatalf("Expected y outside closed heading/item, got document=%#v itemParent=%#v", parsedBodyChildren(document), item.Parent)
		}
	}
}

func TestParseGenericDescendantEndInsideListItemVersusAncestorEnd(t *testing.T) {
	const descendant = `<ul><li><span><em>x</span>y</li></ul>`
	document, err := NewHTMLParser().Parse(descendant)
	if err != nil {
		t.Fatal(err)
	}
	li := listElements(document, "li")[0]
	span := listElements(document, "span")[0]
	em := listElements(document, "em")[0]
	assertListNode(t, li, "li", "xy", 4, 32)
	assertListNode(t, span, "span", "x", 8, 26)
	assertListNode(t, em, "em", "x", 14, 19)
	if span.Parent != li || em.Parent != span {
		t.Fatalf("Expected matching descendant span and em inside li, got span=%#v em=%#v", span.Parent, em.Parent)
	}

	const ancestor = `<span><li>x</span>y`
	document, err = NewHTMLParser().Parse(ancestor)
	if err != nil {
		t.Fatal(err)
	}
	assertListNode(t, listElements(document, "span")[0], "span", "xy", 0, len(ancestor))
	assertListNode(t, listElements(document, "li")[0], "li", "xy", 6, len(ancestor))
}

func TestParseListRecoveryDeepChainScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(depth int) string {
		var b strings.Builder
		b.Grow(depth*3 + 24)
		b.WriteString(`<ul><li>x`)
		for i := 0; i < depth; i++ {
			b.WriteString(`<x>`)
		}
		b.WriteString(`<li>y</ul>`)
		return b.String()
	}
	measure := func(depth int) time.Duration {
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
	small, large := measure(1000), measure(4000)
	ratio := float64(large) / float64(small)
	t.Logf("deep list scan scaling 1000=%v 4000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 15*time.Millisecond {
		t.Fatalf("Deep list-item ancestor scan scaled superlinearly: 1000=%v 4000=%v ratio=%.1fx", small, large, ratio)
	}
}

func TestParsePendingListItemCommentPlacementAtDocumentEnd(t *testing.T) {
	t.Run("comment under html after body end", func(t *testing.T) {
		const content = `<html><body><li>x</body><!--c-->y</html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html := listElements(document, "html")[0]
		body := listElements(document, "body")[0]
		li := listElements(document, "li")[0]
		comments := findAllNodesByType(document, types.CommentNode)
		assertListNode(t, html, "html", "xy", 0, 40)
		assertListNode(t, body, "body", "xy", 6, 24)
		assertListNode(t, li, "li", "xy", 12, 40)
		if len(comments) != 1 || comments[0].Value != "c" || comments[0].StartPos != 24 || comments[0].EndPos != 32 || comments[0].Parent != html {
			t.Fatalf("Expected comment c at 24:32 directly under html, got %#v", comments)
		}
		if len(li.Children) != 1 || li.Children[0].Value != "xy" || li.Children[0].StartPos != 16 || li.Children[0].EndPos != 33 {
			t.Fatalf("Expected li text xy spanning body end and html-level comment at 16:33, got %#v", li.Children)
		}
	})

	t.Run("comment under document after html end", func(t *testing.T) {
		const content = `<html><body><dl><dd>x</html><!--c-->y`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html := listElements(document, "html")[0]
		body := listElements(document, "body")[0]
		dl := listElements(document, "dl")[0]
		dd := listElements(document, "dd")[0]
		comments := findAllNodesByType(document, types.CommentNode)
		assertListNode(t, html, "html", "xy", 0, 28)
		assertListNode(t, body, "body", "xy", 6, 21)
		assertListNode(t, dl, "dl", "xy", 12, 37)
		assertListNode(t, dd, "dd", "xy", 16, 37)
		if len(comments) != 1 || comments[0].Value != "c" || comments[0].StartPos != 28 || comments[0].EndPos != 36 || comments[0].Parent != document {
			t.Fatalf("Expected comment c at 28:36 directly under Document, got %#v", comments)
		}
		if len(dd.Children) != 1 || dd.Children[0].Value != "xy" || dd.Children[0].StartPos != 20 || dd.Children[0].EndPos != 37 {
			t.Fatalf("Expected dd text xy spanning html end and document comment at 20:37, got %#v", dd.Children)
		}
	})
}

func TestParseListRecoveryDeepMatchingAncestorEndScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	builders := map[string]func(int) string{
		"li closed with ancestor div": func(depth int) string {
			var b strings.Builder
			b.Grow(depth*3 + 24)
			b.WriteString(`<div><li>x`)
			for i := 0; i < depth; i++ {
				b.WriteString(`<x>`)
			}
			b.WriteString(`</div>y`)
			return b.String()
		},
		"dd closed with ancestor section": func(depth int) string {
			var b strings.Builder
			b.Grow(depth*3 + 32)
			b.WriteString(`<section><dd>x`)
			for i := 0; i < depth; i++ {
				b.WriteString(`<x>`)
			}
			b.WriteString(`</section>y`)
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
			small, large := measure(build, 1000), measure(build, 4000)
			ratio := float64(large) / float64(small)
			t.Logf("deep matching ancestor scaling 1000=%v 4000=%v ratio=%.1fx", small, large, ratio)
			if large > small*10 && large-small > 5*time.Millisecond {
				t.Fatalf("Deep matching ancestor end scaled superlinearly: 1000=%v 4000=%v ratio=%.1fx", small, large, ratio)
			}
		})
	}
}

func TestParsePendingListReentryClassifiesDecodedCharacterTokens(t *testing.T) {
	for _, testCase := range []struct {
		name, rawReference, decoded string
		commentStart, commentEnd    int
	}{
		{name: "decimal space", rawReference: `&#32;`, decoded: " ", commentStart: 29, commentEnd: 37},
		{name: "decimal tab", rawReference: `&#9;`, decoded: "\t", commentStart: 28, commentEnd: 36},
		{name: "hex space", rawReference: `&#x20;`, decoded: " ", commentStart: 30, commentEnd: 38},
		{name: "named tab", rawReference: `&Tab;`, decoded: "\t", commentStart: 29, commentEnd: 37},
		{name: "named newline", rawReference: `&NewLine;`, decoded: "\n", commentStart: 33, commentEnd: 41},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			content := `<html><body><li>x</body>` + testCase.rawReference + `<!--c--></html>`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			html := listElements(document, "html")[0]
			body := listElements(document, "body")[0]
			li := listElements(document, "li")[0]
			comments := findAllNodesByType(document, types.CommentNode)
			assertListNode(t, html, "html", "x"+testCase.decoded, 0, len(content))
			assertListNode(t, body, "body", "x"+testCase.decoded, 6, 24)
			assertListNode(t, li, "li", "x"+testCase.decoded, 12, len(content))
			if len(comments) != 1 || comments[0].StartPos != testCase.commentStart || comments[0].EndPos != testCase.commentEnd || comments[0].Parent != html {
				t.Fatalf("Expected decoded whitespace to stay after-body with comment under html, got %#v", comments)
			}
			if len(li.Children) != 1 || li.Children[0].Value != "x"+testCase.decoded || li.Children[0].StartPos != 16 || li.Children[0].EndPos != testCase.commentStart {
				t.Fatalf("Expected decoded whitespace in pending li raw text range 16:%d, got %#v", testCase.commentStart, li.Children)
			}
		})
	}

	t.Run("decoded non-whitespace reenters body", func(t *testing.T) {
		const content = `<html><body><li>x</body>&#65;<!--c--></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		li := listElements(document, "li")[0]
		comments := findAllNodesByType(document, types.CommentNode)
		assertListNode(t, li, "li", "xA", 12, len(content))
		if len(li.Children) != 2 || li.Children[0].Type != types.TextNode || li.Children[0].Value != "xA" || li.Children[0].StartPos != 16 || li.Children[0].EndPos != 29 || li.Children[1] != comments[0] {
			t.Fatalf("Expected numeric A to reenter body and following comment to remain in li, got %#v", li.Children)
		}
		if len(comments) != 1 || comments[0].StartPos != 29 || comments[0].EndPos != 37 || comments[0].Parent != li {
			t.Fatalf("Expected comment under reentered li at 29:37, got %#v", comments)
		}
	})
}

func TestParsePendingListClassifierIgnoresDoctypeAndMergesDuplicateHTML(t *testing.T) {
	for _, testCase := range []struct {
		name, content       string
		htmlEnd, bodyEnd    int
		commentStart, liEnd int
		commentUnderHTML    bool
		wantLang            string
	}{
		{name: "doctype after body", content: `<html><body><li>x</body><!DOCTYPE svg><!--c--></html>`, htmlEnd: 53, bodyEnd: 24, commentStart: 38, liEnd: 53, commentUnderHTML: true},
		{name: "doctype after html", content: `<html><body><li>x</html><!DOCTYPE svg><!--c-->`, htmlEnd: 24, bodyEnd: 17, commentStart: 38, liEnd: 46},
		{name: "duplicate html after body", content: `<html><body><li>x</body><html lang=z><!--c--></html>`, htmlEnd: 52, bodyEnd: 24, commentStart: 37, liEnd: 52, commentUnderHTML: true, wantLang: "z"},
		{name: "duplicate html after html", content: `<html><body><li>x</html><html lang=z><!--c-->`, htmlEnd: 24, bodyEnd: 17, commentStart: 37, liEnd: 45, wantLang: "z"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			html := listElements(document, "html")[0]
			body := listElements(document, "body")[0]
			li := listElements(document, "li")[0]
			comments := findAllNodesByType(document, types.CommentNode)
			assertListNode(t, html, "html", "x", 0, testCase.htmlEnd)
			assertListNode(t, body, "body", "x", 6, testCase.bodyEnd)
			assertListNode(t, li, "li", "x", 12, testCase.liEnd)
			if len(comments) != 1 || comments[0].StartPos != testCase.commentStart || comments[0].EndPos != testCase.commentStart+8 {
				t.Fatalf("Expected comment c at %d:%d, got %#v", testCase.commentStart, testCase.commentStart+8, comments)
			}
			wantParent := document
			if testCase.commentUnderHTML {
				wantParent = html
			}
			if comments[0].Parent != wantParent {
				t.Fatalf("Expected comment parent %#v, got %#v", wantParent, comments[0].Parent)
			}
			if testCase.wantLang != "" && html.Attributes["lang"] != testCase.wantLang {
				t.Fatalf("Expected duplicate html token to merge lang=%q, got %#v", testCase.wantLang, html.Attributes)
			}
			if strings.Contains(testCase.content, "DOCTYPE") && len(findAllNodesByType(document, types.DocumentTypeNode)) != 0 {
				t.Fatalf("Expected post-body/html doctype to be ignored")
			}
		})
	}
}

func TestParsePendingListDecodedPreviewParserReuse(t *testing.T) {
	parser := NewHTMLParser()
	for _, testCase := range []struct {
		content, wantText string
		commentParentHTML bool
	}{
		{content: `<html><body><li>x</body>&#65;<!--c--></html>`, wantText: "xA"},
		{content: `<html><body><li>x</body>&Tab;<!--c--></html>`, wantText: "x\t", commentParentHTML: true},
		{content: `<html><body><li>x</body>&NewLine;<!--c--></html>`, wantText: "x\n", commentParentHTML: true},
	} {
		document, err := parser.Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		li := listElements(document, "li")[0]
		comments := findAllNodesByType(document, types.CommentNode)
		if li.TextContent != testCase.wantText || len(comments) != 1 {
			t.Fatalf("Expected reused parser item text %q and one comment, got li=%#v comments=%#v", testCase.wantText, li, comments)
		}
		if testCase.commentParentHTML != (comments[0].Parent.Name == "html") {
			t.Fatalf("Unexpected comment parent after parser reuse: %#v", comments[0].Parent)
		}
	}
}

func TestParsePendingListWhitespaceCommentScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n*16 + 48)
		b.WriteString(`<html><body><li>x</body>`)
		for i := 0; i < n; i++ {
			b.WriteString(`&#32;<!--c-->`)
		}
		b.WriteString(`<!DOCTYPE svg></html>`)
		return b.String()
	}
	measure := func(n int) time.Duration {
		content := build(n)
		_, _ = NewHTMLParser().Parse(content)
		best := time.Duration(1<<63 - 1)
		for i := 0; i < 3; i++ {
			start := time.Now()
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(findAllNodesByType(document, types.CommentNode)); got != n {
				t.Fatalf("Expected %d after-body comments, got %d", n, got)
			}
			if got := len(findAllNodesByType(document, types.DocumentTypeNode)); got != 0 {
				t.Fatalf("Expected suffix doctype to remain ignored, got %d", got)
			}
			if elapsed := time.Since(start); elapsed < best {
				best = elapsed
			}
		}
		return best
	}
	small, large := measure(250), measure(1000)
	ratio := float64(large) / float64(small)
	t.Logf("post-body whitespace/comment scaling 250=%v 1000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 10*time.Millisecond {
		t.Fatalf("Post-body decoded whitespace/comment classification scaled superlinearly: 250=%v 1000=%v ratio=%.1fx", small, large, ratio)
	}
}
