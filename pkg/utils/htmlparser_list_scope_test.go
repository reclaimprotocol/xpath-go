package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

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
