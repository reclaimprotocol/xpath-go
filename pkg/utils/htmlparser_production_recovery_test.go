package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func requireProductionRecoveryElement(t *testing.T, document *types.Node, name string, index int) *types.Node {
	t.Helper()
	elements := findAllElementsByName(document, name)
	if len(elements) <= index {
		t.Fatalf("Expected %s element %d, got %#v", name, index, elements)
	}
	return elements[index]
}

func assertProductionRecoveryNode(t *testing.T, node *types.Node, name, text string, start, end, contentStart, contentEnd int) {
	t.Helper()
	if node.Name != name || node.TextContent != text || node.StartPos != start || node.EndPos != end || node.ContentStart != contentStart || node.ContentEnd != contentEnd {
		t.Fatalf("Expected <%s> text %q range %d:%d content %d:%d, got %#v", name, text, start, end, contentStart, contentEnd, node)
	}
}

func TestParseProductionRecoveryClosesBlockThroughFooterAndAnchor(t *testing.T) {
	const content = `<html><body><div><footer><a>x</div>y</a></footer></body></html>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Browser-recognized mismatched block tree must recover: %v", err)
	}

	div := requireProductionRecoveryElement(t, document, "div", 0)
	footer := requireProductionRecoveryElement(t, document, "footer", 0)
	anchors := findAllElementsByName(document, "a")
	if len(anchors) != 2 {
		t.Fatalf("Expected source and reconstructed anchors, got %#v", anchors)
	}
	assertProductionRecoveryNode(t, div, "div", "x", 12, 35, 17, 29)
	assertProductionRecoveryNode(t, footer, "footer", "x", 17, 29, 25, 29)
	assertProductionRecoveryNode(t, anchors[0], "a", "x", 25, 29, 28, 29)
	assertProductionRecoveryNode(t, anchors[1], "a", "y", 25, 40, 28, 36)
	if footer.Parent != div || anchors[0].Parent != footer || anchors[1].Parent == nil || anchors[1].Parent.Name != "body" {
		t.Fatalf("Recovered block/anchor parentage mismatch: div=%#v footer=%#v anchors=%#v", div.Parent, footer.Parent, anchors)
	}
	if len(anchors[0].Children) != 1 || anchors[0].Children[0].Value != "x" || anchors[0].Children[0].StartPos != 28 || anchors[0].Children[0].EndPos != 29 || len(anchors[1].Children) != 1 || anchors[1].Children[0].Value != "y" || anchors[1].Children[0].StartPos != 35 || anchors[1].Children[0].EndPos != 36 {
		t.Fatalf("Recovered anchor text/source ranges mismatch: %#v %#v", anchors[0].Children, anchors[1].Children)
	}
}

func TestParseProductionRecoveryClosesListThroughNestedNav(t *testing.T) {
	const content = `<ul><li>x<nav><li>y</ul>z</nav>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Browser-recognized nested nav/list tree must recover: %v", err)
	}

	ul := requireProductionRecoveryElement(t, document, "ul", 0)
	items := findAllElementsByName(document, "li")
	nav := requireProductionRecoveryElement(t, document, "nav", 0)
	if len(items) != 2 {
		t.Fatalf("Expected outer and nested list items, got %#v", items)
	}
	assertProductionRecoveryNode(t, ul, "ul", "xy", 0, 24, 4, 19)
	assertProductionRecoveryNode(t, items[0], "li", "xy", 4, 19, 8, 19)
	assertProductionRecoveryNode(t, nav, "nav", "y", 9, 19, 14, 19)
	assertProductionRecoveryNode(t, items[1], "li", "y", 14, 19, 18, 19)
	if items[0].Parent != ul || nav.Parent != items[0] || items[1].Parent != nav {
		t.Fatalf("Recovered nested list parentage mismatch: ul=%#v items=%#v nav=%#v", ul, items, nav)
	}
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0] != ul || parsedBodyChildren(document)[1].Type != types.TextNode || parsedBodyChildren(document)[1].Value != "z" || parsedBodyChildren(document)[1].StartPos != 24 || parsedBodyChildren(document)[1].EndPos != 25 {
		t.Fatalf("Expected stale nav end ignored after root tail text, got %#v", parsedBodyChildren(document))
	}
}

func TestParseProductionRecoveryKeepsLessThanInTagName(t *testing.T) {
	const content = `<div><span<em>x</span>y</div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Browser-recognized malformed tag name must recover: %v", err)
	}

	div := requireProductionRecoveryElement(t, document, "div", 0)
	custom := requireProductionRecoveryElement(t, document, "span<em", 0)
	assertProductionRecoveryNode(t, div, "div", "xy", 0, 29, 5, 23)
	assertProductionRecoveryNode(t, custom, "span<em", "xy", 5, 23, 14, 23)
	if custom.Parent != div || len(custom.Children) != 1 || custom.Children[0].Type != types.TextNode || custom.Children[0].Value != "xy" || custom.Children[0].StartPos != 14 || custom.Children[0].EndPos != 23 {
		t.Fatalf("Malformed tag-name recovery structure/range mismatch: %#v", custom)
	}
}

func TestParseProductionRecoveryHandlesGenericEndTagsLikeBrowser(t *testing.T) {
	t.Run("matching ancestor implicitly pops descendant", func(t *testing.T) {
		const content = `<div><span></div>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Matching ancestor end must implicitly pop span: %v", err)
		}
		div := requireProductionRecoveryElement(t, document, "div", 0)
		span := requireProductionRecoveryElement(t, document, "span", 0)
		assertProductionRecoveryNode(t, div, "div", "", 0, 17, 5, 11)
		assertProductionRecoveryNode(t, span, "span", "", 5, 11, 11, 11)
		if span.Parent != div {
			t.Fatalf("Expected implicitly popped span under div, got %#v", span.Parent)
		}
	})

	t.Run("absent ordinary end is ignored", func(t *testing.T) {
		const content = `<div></span></div>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Absent ordinary end tag must be ignored: %v", err)
		}
		div := requireProductionRecoveryElement(t, document, "div", 0)
		assertProductionRecoveryNode(t, div, "div", "", 0, 18, 5, 12)
		if spans := findAllElementsByName(document, "span"); len(spans) != 0 {
			t.Fatalf("Ignored absent end created span nodes: %#v", spans)
		}
	})
}

func TestParseProductionRecoveryReconstructsFormattingAfterGenericPop(t *testing.T) {
	const content = `<b><span><small>x</span>y</b>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Generic span end must preserve active small formatting: %v", err)
	}

	bold := requireProductionRecoveryElement(t, document, "b", 0)
	span := requireProductionRecoveryElement(t, document, "span", 0)
	smalls := findAllElementsByName(document, "small")
	if len(smalls) != 2 {
		t.Fatalf("Expected source and reconstructed small nodes, got %#v", smalls)
	}
	assertProductionRecoveryNode(t, bold, "b", "xy", 0, 29, 3, 25)
	assertProductionRecoveryNode(t, span, "span", "x", 3, 24, 9, 17)
	assertProductionRecoveryNode(t, smalls[0], "small", "x", 9, 17, 16, 17)
	assertProductionRecoveryNode(t, smalls[1], "small", "y", 9, 25, 16, 25)
	if smalls[0].Parent != span || smalls[1].Parent != bold {
		t.Fatalf("Formatting reconstruction parentage mismatch: %#v", smalls)
	}
}

func TestParseProductionRecoveryPreservesFormPointerAcrossBlockUnwind(t *testing.T) {
	const content = `<div><form id=a>x</div><form id=b>y</form><form id=c>z</form>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Block unwind must preserve form pointer semantics: %v", err)
	}

	div := requireProductionRecoveryElement(t, document, "div", 0)
	forms := findAllElementsByName(document, "form")
	if len(forms) != 2 || forms[0].Attributes["id"] != "a" || forms[1].Attributes["id"] != "c" {
		t.Fatalf("Expected ignored form b and admitted forms a/c, got %#v", forms)
	}
	assertProductionRecoveryNode(t, div, "div", "x", 0, 23, 5, 17)
	assertProductionRecoveryNode(t, forms[0], "form", "x", 5, 17, 16, 17)
	assertProductionRecoveryNode(t, forms[1], "form", "z", 42, 61, 53, 54)
	if forms[0].Parent != div || len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[0] != div || parsedBodyChildren(document)[1].Type != types.TextNode || parsedBodyChildren(document)[1].Value != "y" || parsedBodyChildren(document)[1].StartPos != 34 || parsedBodyChildren(document)[1].EndPos != 35 || parsedBodyChildren(document)[2] != forms[1] {
		t.Fatalf("Recovered form-pointer tree mismatch: %#v", parsedBodyChildren(document))
	}
}

func TestParseProductionRecoverySwallowsMissingGreaterThanIntoAttributes(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		wantEnd     int
		wantStart   int
		wantContent int
		wantAttrs   []string
		wantValues  map[string]string
	}{
		{
			name:        "unquoted value",
			content:     `<div class=x<span>y</span></div>`,
			wantEnd:     32,
			wantStart:   18,
			wantContent: 26,
			wantAttrs:   []string{"class"},
			wantValues:  map[string]string{"class": "x<span"},
		},
		{
			name:        "quoted value followed by malformed attribute",
			content:     `<div class="x"<span>y</span></div>`,
			wantEnd:     34,
			wantStart:   20,
			wantContent: 28,
			wantAttrs:   []string{"class", "<span"},
			wantValues:  map[string]string{"class": "x", "<span": ""},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Missing > attribute recovery failed: %v", err)
			}
			div := requireProductionRecoveryElement(t, document, "div", 0)
			assertProductionRecoveryNode(t, div, "div", "y", 0, testCase.wantEnd, testCase.wantStart, testCase.wantContent)
			if len(div.AttributeOrder) != len(testCase.wantAttrs) {
				t.Fatalf("Attribute order mismatch: %#v", div.AttributeOrder)
			}
			for index, name := range testCase.wantAttrs {
				if div.AttributeOrder[index] != name || div.Attributes[name] != testCase.wantValues[name] {
					t.Fatalf("Attribute %d mismatch: order=%#v values=%#v", index, div.AttributeOrder, div.Attributes)
				}
			}
			if len(findAllElementsByName(document, "span")) != 0 || len(div.Children) != 1 || div.Children[0].Value != "y" || div.Children[0].StartPos != testCase.wantStart || div.Children[0].EndPos != testCase.wantStart+1 {
				t.Fatalf("Swallowed span token emitted a node or shifted text: %#v", div)
			}
		})
	}
}

func TestParseProductionRecoveryParserReuseClearsUnwindState(t *testing.T) {
	parser := NewHTMLParser()
	for _, content := range []string{
		`<html><body><div><footer><a>x</div>y</a></footer></body></html>`,
		`<ul><li>x<nav><li>y</ul>z</nav>`,
		`<div><span<em>x</span>y</div>`,
		`<div><form id=a>x</div><form id=b>y</form><form id=c>z</form>`,
	} {
		if _, err := parser.Parse(content); err != nil {
			t.Fatalf("Recovery parse failed for %q: %v", content, err)
		}
	}
	document, err := parser.Parse(`<div id=clean>clean</div>`)
	if err != nil {
		t.Fatalf("Clean reused parse failed: %v", err)
	}
	divs := findAllElementsByName(document, "div")
	if len(divs) != 1 || divs[0].Attributes["id"] != "clean" || divs[0].TextContent != "clean" || len(findAllElementsByName(document, "a")) != 0 || len(findAllElementsByName(document, "form")) != 0 {
		t.Fatalf("Generic recovery state leaked into reused parse: %#v", document)
	}
}

func TestParseProductionRecoveryIgnoredEndAndDocumentEOF(t *testing.T) {
	const recovered = `<div></span></div><foo>x`
	document, err := NewHTMLParser().Parse(recovered)
	if err != nil || len(findAllElementsByName(document, "foo")) != 1 || findAllElementsByName(document, "foo")[0].EndPos != len(recovered) {
		t.Fatalf("Ignored end followed by implicit-document EOF recovery mismatch: doc=%#v err=%v", document, err)
	}

	parser := NewHTMLParser()
	if _, err := parser.Parse(`<div><span></div>`); err != nil {
		t.Fatalf("Expected recovered matching ancestor contrast: %v", err)
	}
	if document, err = parser.Parse(recovered); err != nil || len(findAllElementsByName(document, "foo")) != 1 || findAllElementsByName(document, "foo")[0].EndPos != len(recovered) {
		t.Fatalf("Reused generic recovery implicit-document EOF mismatch: doc=%#v err=%v", document, err)
	}
	clean, err := parser.Parse(`<foo>x</foo>`)
	if err != nil || len(findAllElementsByName(clean, "foo")) != 1 || findAllElementsByName(clean, "foo")[0].TextContent != "x" {
		t.Fatalf("Parser did not recover cleanly after document EOF recovery: document=%#v err=%v", clean, err)
	}
}

func productionRecoveryDuration(t *testing.T, content string) time.Duration {
	t.Helper()
	best := time.Duration(1<<63 - 1)
	for attempt := 0; attempt < 2; attempt++ {
		started := time.Now()
		if _, err := NewHTMLParser().Parse(content); err != nil {
			t.Fatalf("Production recovery scaling parse failed: %v", err)
		}
		if elapsed := time.Since(started); elapsed < best {
			best = elapsed
		}
	}
	return best
}

func TestParseProductionRecoveryScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping production recovery scaling regression in short mode")
	}

	builders := map[string]func(int) string{
		"shallow absent ends": func(n int) string {
			return `<div>` + strings.Repeat(`x</span>`, n) + `</div>`
		},
		"deep one-shot ancestor unwind": func(n int) string {
			return `<div><footer><a>x` + strings.Repeat(`<span>`, n) + `</div>y</a>`
		},
	}
	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			_ = productionRecoveryDuration(t, build(100))
			small := productionRecoveryDuration(t, build(500))
			large := productionRecoveryDuration(t, build(2000))
			ratio := float64(large) / float64(small)
			t.Logf("production recovery scaling 500=%v 2000=%v ratio=%.1fx", small, large, ratio)
			if ratio > 14 && large-small > 150*time.Millisecond {
				t.Fatalf("Production recovery scaled quadratically: %.1fx (%v -> %v)", ratio, small, large)
			}
		})
	}
}
