package utils

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseCurrentSelectOptionEndScopeAndInBodyBlocks(t *testing.T) {
	const blockedEnd = `<select><option><div>x</option>y</select>`
	document, err := NewHTMLParser().Parse(blockedEnd)
	if err != nil {
		t.Fatal(err)
	}
	option := selectElements(document, "option")[0]
	div := selectElements(document, "div")[0]
	assertSelectNode(t, option, "option", "xy", 8, 32)
	assertSelectNode(t, div, "div", "xy", 16, 32)
	if len(div.Children) != 1 || div.Children[0].Value != "xy" || div.Children[0].StartPos != 21 || div.Children[0].EndPos != 32 {
		t.Fatalf("Expected block-scoped ignored option end inside merged div text, got %#v", div.Children)
	}

	const blocks = `<select><option>a<p>b<li>c<button>d<option>e</select>`
	document, err = NewHTMLParser().Parse(blocks)
	if err != nil {
		t.Fatal(err)
	}
	options := selectElements(document, "option")
	assertSelectNode(t, options[0], "option", "abcde", 8, 44)
	assertSelectNode(t, selectElements(document, "p")[0], "p", "b", 17, 21)
	assertSelectNode(t, selectElements(document, "li")[0], "li", "cde", 21, 44)
	assertSelectNode(t, selectElements(document, "button")[0], "button", "de", 26, 44)
	assertSelectNode(t, options[1], "option", "e", 35, 44)
}

func TestParseSelectRootASCIIHRCharacterTokensAndIncompleteEOF(t *testing.T) {
	for _, content := range []string{`<option>a<option>b`, `<OPTION>a<OPTION>b`} {
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		options := selectElements(document, "option")
		assertSelectNode(t, options[0], "option", "a", 0, 9)
		assertSelectNode(t, options[1], "option", "b", 9, 18)
	}

	const hr = `<select><option>a<hr><option>b</select>`
	document, err := NewHTMLParser().Parse(hr)
	if err != nil {
		t.Fatal(err)
	}
	options := selectElements(document, "option")
	assertSelectNode(t, options[0], "option", "a", 8, 17)
	if hrNode := selectElements(document, "hr")[0]; hrNode.Parent != selectElements(document, "select")[0] || hrNode.StartPos != 17 || hrNode.EndPos != 21 {
		t.Fatalf("Expected hr to close option and remain under select, got %#v", hrNode)
	}
	assertSelectNode(t, options[1], "option", "b", 21, 30)

	const characters = "<select>a&amp;&#65;\x00<option>b</select>"
	document, err = NewHTMLParser().Parse(characters)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	assertSelectNode(t, selectNode, "select", "a&Ab", 0, 38)
	if len(selectNode.Children) != 2 || selectNode.Children[0].Type != types.TextNode || selectNode.Children[0].Value != "a&A" || selectNode.Children[0].StartPos != 8 || selectNode.Children[0].EndPos != 20 {
		t.Fatalf("Expected references decoded and NUL ignored in select text, got %#v", selectNode.Children)
	}

	const incomplete = `<select><option>x<option`
	document, err = NewHTMLParser().Parse(incomplete)
	if err != nil {
		t.Fatal(err)
	}
	options = selectElements(document, "option")
	if len(options) != 1 {
		t.Fatalf("Expected incomplete option start discarded, got %#v", options)
	}
	assertSelectNode(t, options[0], "option", "x", 8, len(incomplete))
	if options[0].Children[0].StartPos != 16 || options[0].Children[0].EndPos != len(incomplete) {
		t.Fatalf("Expected prior text range to extend through incomplete token, got %#v", options[0].Children[0])
	}
}

func TestParseSelectScopeBoundaryProtectsOuterListAndDefinitionItems(t *testing.T) {
	for _, testCase := range []struct {
		content, container, outerItem, innerItem string
	}{
		{content: `<ul><li>a<select></li><li>b</select>c</li></ul>`, container: "ul", outerItem: "li", innerItem: "li"},
		{content: `<dl><dd>a<select></dd><dt>b</select>c</dd></dl>`, container: "dl", outerItem: "dd", innerItem: "dt"},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		container := selectElements(document, testCase.container)[0]
		outer := selectElements(container, testCase.outerItem)[0]
		selectNode := selectElements(outer, "select")[0]
		inner := selectElements(selectNode, testCase.innerItem)[0]
		assertSelectNode(t, outer, testCase.outerItem, "abc", 4, 42)
		assertSelectNode(t, selectNode, "select", "b", 9, 36)
		assertSelectNode(t, inner, testCase.innerItem, "b", 22, 27)
		if selectNode.Parent != outer || inner.Parent != selectNode {
			t.Fatalf("Expected select boundary to preserve outer item and contain new inner item")
		}
	}
}

func TestParseSelectScopeBoundaryProtectsOuterParagraphButtonAndGenericAncestor(t *testing.T) {
	const paragraph = `<p>a<select><div>b</div></select>c</p>`
	document, err := NewHTMLParser().Parse(paragraph)
	if err != nil {
		t.Fatal(err)
	}
	p := selectElements(document, "p")[0]
	assertSelectNode(t, p, "p", "abc", 0, 38)
	assertSelectNode(t, selectElements(p, "select")[0], "select", "b", 4, 33)
	assertSelectNode(t, selectElements(p, "div")[0], "div", "b", 12, 24)

	const button = `<button>a<select><button>b</button></select>c</button>`
	document, err = NewHTMLParser().Parse(button)
	if err != nil {
		t.Fatal(err)
	}
	buttons := selectElements(document, "button")
	assertSelectNode(t, buttons[0], "button", "abc", 0, 54)
	assertSelectNode(t, selectElements(buttons[0], "select")[0], "select", "b", 9, 44)
	assertSelectNode(t, buttons[1], "button", "b", 17, 35)
	if buttons[1].Parent.Name != "select" {
		t.Fatalf("Expected nested button retained behind select button-scope boundary")
	}

	const generic = `<div>a<select></div>b</select>c</div>`
	document, err = NewHTMLParser().Parse(generic)
	if err != nil {
		t.Fatal(err)
	}
	div := selectElements(document, "div")[0]
	selectNode := selectElements(div, "select")[0]
	assertSelectNode(t, div, "div", "abc", 0, 37)
	assertSelectNode(t, selectNode, "select", "b", 6, 30)
	if len(selectNode.Children) != 1 || selectNode.Children[0].Value != "b" || selectNode.Children[0].StartPos != 20 || selectNode.Children[0].EndPos != 21 {
		t.Fatalf("Expected ignored outer div end in select text raw range, got %#v", selectNode.Children)
	}
}

func TestParseSelectIgnoresBodyHTMLEndsAndSynthesizesStrayParagraph(t *testing.T) {
	const bodyHTML = `<html><body><select>a</body>b</html>c</select><p>d</p>`
	document, err := NewHTMLParser().Parse(bodyHTML)
	if err != nil {
		t.Fatal(err)
	}
	html := selectElements(document, "html")[0]
	body := selectElements(document, "body")[0]
	selectNode := selectElements(document, "select")[0]
	assertSelectNode(t, html, "html", "abcd", 0, len(bodyHTML))
	assertSelectNode(t, body, "body", "abcd", 6, len(bodyHTML))
	assertSelectNode(t, selectNode, "select", "abc", 12, 46)
	if len(selectNode.Children) != 1 || selectNode.Children[0].Value != "abc" || selectNode.Children[0].StartPos != 20 || selectNode.Children[0].EndPos != 37 {
		t.Fatalf("Expected ignored body/html ends in coalesced select text, got %#v", selectNode.Children)
	}
	assertSelectNode(t, selectElements(body, "p")[0], "p", "d", 46, 54)

	const strayP = `<p>a<select></p>b</select>c</p>`
	document, err = NewHTMLParser().Parse(strayP)
	if err != nil {
		t.Fatal(err)
	}
	paragraphs := selectElements(document, "p")
	if len(paragraphs) != 2 {
		t.Fatalf("Expected outer and synthetic paragraphs, got %#v", paragraphs)
	}
	assertSelectNode(t, paragraphs[0], "p", "abc", 0, 31)
	assertSelectNode(t, selectElements(paragraphs[0], "select")[0], "select", "b", 4, 26)
	assertSelectNode(t, paragraphs[1], "p", "", 0, 0)
	if paragraphs[1].Parent.Name != "select" {
		t.Fatalf("Expected stray p end to synthesize empty p inside select")
	}
}

func TestParseSelectStartsOnlyGenerateImpliedEnds(t *testing.T) {
	const optionSpan = `<select><option><span>x<option>y</select>`
	document, err := NewHTMLParser().Parse(optionSpan)
	if err != nil {
		t.Fatal(err)
	}
	options := selectElements(document, "option")
	span := selectElements(document, "span")[0]
	assertSelectNode(t, options[0], "option", "xy", 8, 32)
	assertSelectNode(t, span, "span", "xy", 16, 32)
	assertSelectNode(t, options[1], "option", "y", 23, 32)
	if options[1].Parent != span {
		t.Fatalf("Expected span to prevent direct option auto-close")
	}

	for _, testCase := range []struct {
		content, implied string
		trigger          int
	}{
		{content: `<select><option><p>x<option>y</select>`, implied: "p", trigger: 20},
		{content: `<select><option><li>x<option>y</select>`, implied: "li", trigger: 21},
	} {
		document, err = NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		options = selectElements(document, "option")
		assertSelectNode(t, options[0], "option", "x", 8, testCase.trigger)
		assertSelectNode(t, selectElements(options[0], testCase.implied)[0], testCase.implied, "x", 16, testCase.trigger)
		assertSelectNode(t, options[1], "option", "y", testCase.trigger, len(testCase.content)-9)
		if options[0].Parent != options[1].Parent {
			t.Fatalf("Expected implied-end descendant to permit sibling options")
		}
	}

	const groupSpan = `<select><optgroup><span>x<optgroup>y</select>`
	document, err = NewHTMLParser().Parse(groupSpan)
	if err != nil {
		t.Fatal(err)
	}
	groups := selectElements(document, "optgroup")
	span = selectElements(document, "span")[0]
	assertSelectNode(t, groups[0], "optgroup", "xy", 8, 36)
	assertSelectNode(t, span, "span", "xy", 18, 36)
	assertSelectNode(t, groups[1], "optgroup", "y", 25, 36)
	if groups[1].Parent != span {
		t.Fatalf("Expected span to prevent direct optgroup auto-close")
	}

	const groupP = `<select><optgroup><p>x<optgroup>y</select>`
	document, err = NewHTMLParser().Parse(groupP)
	if err != nil {
		t.Fatal(err)
	}
	groups = selectElements(document, "optgroup")
	assertSelectNode(t, groups[0], "optgroup", "x", 8, 22)
	assertSelectNode(t, selectElements(groups[0], "p")[0], "p", "x", 18, 22)
	assertSelectNode(t, groups[1], "optgroup", "y", 22, 33)

	const hrInsideDiv = `<select><option><div>x<hr>y</select>`
	document, err = NewHTMLParser().Parse(hrInsideDiv)
	if err != nil {
		t.Fatal(err)
	}
	option := selectElements(document, "option")[0]
	div := selectElements(document, "div")[0]
	assertSelectNode(t, option, "option", "xy", 8, 27)
	assertSelectNode(t, div, "div", "xy", 16, 27)
	if hr := selectElements(div, "hr")[0]; hr.StartPos != 22 || hr.EndPos != 26 {
		t.Fatalf("Expected hr retained inside div/option, got %#v", hr)
	}
}

func TestParseStandaloneOptionAndOptgroupStartsOnlyPopCurrentNode(t *testing.T) {
	for _, testCase := range []struct {
		name, content, descendant, nested string
		descendantStart, nestedStart      int
	}{
		{name: "paragraph between options", content: `<option><p>x<option>y`, descendant: "p", nested: "option", descendantStart: 8, nestedStart: 12},
		{name: "list item between options", content: `<option><li>x<option>y`, descendant: "li", nested: "option", descendantStart: 8, nestedStart: 13},
		{name: "paragraph between optgroups", content: `<optgroup><p>x<optgroup>y`, descendant: "p", nested: "optgroup", descendantStart: 10, nestedStart: 14},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			outer := selectElements(document, testCase.nested)[0]
			descendant := selectElements(outer, testCase.descendant)[0]
			nestedNodes := selectElements(descendant, testCase.nested)
			if len(nestedNodes) != 1 {
				t.Fatalf("Expected nested <%s> behind current <%s>, got %#v", testCase.nested, testCase.descendant, nestedNodes)
			}
			nested := nestedNodes[0]
			assertSelectNode(t, outer, testCase.nested, "xy", 0, len(testCase.content))
			assertSelectNode(t, descendant, testCase.descendant, "xy", testCase.descendantStart, len(testCase.content))
			assertSelectNode(t, nested, testCase.nested, "y", testCase.nestedStart, len(testCase.content))
			if nested.Parent != descendant {
				t.Fatalf("Expected second start nested because outer item was not current")
			}
		})
	}
}

func TestParseSelectStartsGenerateImpliedEndsWithoutPriorItem(t *testing.T) {
	for _, testCase := range []struct {
		name, content, implied, inserted, container string
		impliedStart, trigger                       int
	}{
		{name: "p before option", content: `<select><p>x<option>y`, implied: "p", inserted: "option", impliedStart: 8, trigger: 12},
		{name: "li before option", content: `<select><li>x<option>y`, implied: "li", inserted: "option", impliedStart: 8, trigger: 13},
		{name: "p before option in optgroup", content: `<select><optgroup><p>x<option>y`, implied: "p", inserted: "option", container: "optgroup", impliedStart: 18, trigger: 22},
		{name: "p before optgroup", content: `<select><p>x<optgroup>y`, implied: "p", inserted: "optgroup", impliedStart: 8, trigger: 12},
		{name: "li before optgroup", content: `<select><li>x<optgroup>y`, implied: "li", inserted: "optgroup", impliedStart: 8, trigger: 13},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			selectNode := selectElements(document, "select")[0]
			parent := selectNode
			if testCase.container != "" {
				parent = selectElements(selectNode, testCase.container)[0]
				assertSelectNode(t, parent, testCase.container, "xy", 8, len(testCase.content))
			}
			implied := selectElements(parent, testCase.implied)[0]
			inserted := selectElements(parent, testCase.inserted)[0]
			assertSelectNode(t, implied, testCase.implied, "x", testCase.impliedStart, testCase.trigger)
			assertSelectNode(t, inserted, testCase.inserted, "y", testCase.trigger, len(testCase.content))
			if implied.Parent != parent || inserted.Parent != parent {
				t.Fatalf("Expected implied node and inserted item to be siblings under %#v", parent)
			}
		})
	}

	for _, itemName := range []string{"li", "dd"} {
		content := `<select><` + itemName + `>x<hr>y`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		selectNode := selectElements(document, "select")[0]
		item := selectElements(selectNode, itemName)[0]
		hr := selectElements(selectNode, "hr")[0]
		assertSelectNode(t, item, itemName, "x", 8, 13)
		if hr.StartPos != 13 || hr.EndPos != 17 || hr.Parent != selectNode {
			t.Fatalf("Expected hr to follow implied-end item under select, got %#v", hr)
		}
		if len(selectNode.Children) != 3 || selectNode.Children[2].Type != types.TextNode || selectNode.Children[2].Value != "y" || selectNode.Children[2].StartPos != 17 || selectNode.Children[2].EndPos != 18 {
			t.Fatalf("Expected y after hr under select, got %#v", selectNode.Children)
		}
	}
}
