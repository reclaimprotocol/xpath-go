package utils

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 13 targets current WHATWG/Chrome customizable-select behavior. Legacy
// parse5 "in select" handling is not the oracle for divergent descendant cases.
// Templates, foreign content, fragments, adoption-agency behavior, and the
// table/select interaction are deferred.

func selectElements(node *types.Node, name string) []*types.Node {
	var matches []*types.Node
	if node == nil {
		return matches
	}
	if node.Type == types.ElementNode && node.Name == name {
		matches = append(matches, node)
	}
	for _, child := range node.Children {
		matches = append(matches, selectElements(child, name)...)
	}
	return matches
}

func assertSelectNode(t *testing.T, node *types.Node, name, text string, start, end int) {
	t.Helper()
	if node == nil || node.Name != name || node.TextContent != text || node.StartPos != start || node.EndPos != end {
		t.Fatalf("Expected <%s> text %q at %d:%d, got %#v", name, text, start, end, node)
	}
}

func TestParseOptionAndOptgroupStartsAutoCloseDirectItems(t *testing.T) {
	const options = `<select><option>a<option>b</select>tail`
	document, err := NewHTMLParser().Parse(options)
	if err != nil {
		t.Fatal(err)
	}
	optionNodes := selectElements(document, "option")
	assertSelectNode(t, optionNodes[0], "option", "a", 8, 17)
	assertSelectNode(t, optionNodes[1], "option", "b", 17, 26)
	assertSelectNode(t, selectElements(document, "select")[0], "select", "ab", 0, 35)
	if optionNodes[0].Parent != optionNodes[1].Parent {
		t.Fatalf("Expected direct option starts to create siblings")
	}

	const groups = `<select><optgroup label=a><option>x<optgroup label=b><option>y</select>tail`
	document, err = NewHTMLParser().Parse(groups)
	if err != nil {
		t.Fatal(err)
	}
	groupNodes := selectElements(document, "optgroup")
	optionNodes = selectElements(document, "option")
	assertSelectNode(t, groupNodes[0], "optgroup", "x", 8, 35)
	assertSelectNode(t, optionNodes[0], "option", "x", 26, 35)
	assertSelectNode(t, groupNodes[1], "optgroup", "y", 35, 62)
	assertSelectNode(t, optionNodes[1], "option", "y", 53, 62)
	if groupNodes[0].Parent != groupNodes[1].Parent || optionNodes[0].Parent != groupNodes[0] || optionNodes[1].Parent != groupNodes[1] {
		t.Fatalf("Expected sibling optgroups with option children")
	}
}

func TestParseOptionAndOptgroupExplicitAndMismatchedEnds(t *testing.T) {
	const explicit = `<select><option>a</option><optgroup><option>b</option></optgroup>z</select>`
	document, err := NewHTMLParser().Parse(explicit)
	if err != nil {
		t.Fatal(err)
	}
	optionNodes := selectElements(document, "option")
	assertSelectNode(t, optionNodes[0], "option", "a", 8, 26)
	assertSelectNode(t, optionNodes[1], "option", "b", 36, 54)
	assertSelectNode(t, selectElements(document, "optgroup")[0], "optgroup", "b", 26, 65)

	const ignored = `<select>a</option>b</optgroup>c<option>d</select>`
	document, err = NewHTMLParser().Parse(ignored)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	assertSelectNode(t, selectNode, "select", "abcd", 0, 49)
	if len(selectNode.Children) != 2 || selectNode.Children[0].Type != types.TextNode || selectNode.Children[0].Value != "abc" || selectNode.Children[0].StartPos != 8 || selectNode.Children[0].EndPos != 31 {
		t.Fatalf("Expected absent ends to be ignored inside merged text, got %#v", selectNode.Children)
	}

	const mismatch = `<select><option>a</optgroup>b</option>c</select>tail`
	document, err = NewHTMLParser().Parse(mismatch)
	if err != nil {
		t.Fatal(err)
	}
	option := selectElements(document, "option")[0]
	assertSelectNode(t, option, "option", "ab", 8, 38)
	if len(option.Children) != 1 || option.Children[0].Value != "ab" || option.Children[0].StartPos != 16 || option.Children[0].EndPos != 29 {
		t.Fatalf("Expected mismatched optgroup end inside option text range, got %#v", option.Children)
	}

	const groupEnd = `<select><optgroup><option>a</optgroup>b<option>c</select>`
	document, err = NewHTMLParser().Parse(groupEnd)
	if err != nil {
		t.Fatal(err)
	}
	group := selectElements(document, "optgroup")[0]
	optionNodes = selectElements(document, "option")
	assertSelectNode(t, optionNodes[0], "option", "a", 18, 27)
	assertSelectNode(t, group, "optgroup", "a", 8, 38)
	assertSelectNode(t, optionNodes[1], "option", "c", 39, 48)
}

func TestParseCurrentSelectKeepsOrdinaryDescendants(t *testing.T) {
	const content = `<select><option>a<div>x<option>b</select>tail`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	options := selectElements(document, "option")
	div := selectElements(document, "div")[0]
	assertSelectNode(t, selectNode, "select", "axb", 0, 41)
	assertSelectNode(t, options[0], "option", "axb", 8, 32)
	assertSelectNode(t, div, "div", "xb", 17, 32)
	assertSelectNode(t, options[1], "option", "b", 23, 32)
	if div.Parent != options[0] || options[1].Parent != div {
		t.Fatalf("Expected current Chrome nested descendant structure, got div=%#v option=%#v", div.Parent, options[1].Parent)
	}

	const ignoredLegacy = `<select>a<div>b</div>c<option>d<span>e</span>f</select>tail`
	document, err = NewHTMLParser().Parse(ignoredLegacy)
	if err != nil {
		t.Fatal(err)
	}
	assertSelectNode(t, selectElements(document, "div")[0], "div", "b", 9, 21)
	assertSelectNode(t, selectElements(document, "span")[0], "span", "e", 31, 45)
	assertSelectNode(t, selectElements(document, "option")[0], "option", "def", 22, 46)

	const nestedGroup = `<select><optgroup><option>a<div>x<optgroup><option>b</select>tail`
	document, err = NewHTMLParser().Parse(nestedGroup)
	if err != nil {
		t.Fatal(err)
	}
	groups := selectElements(document, "optgroup")
	options = selectElements(document, "option")
	div = selectElements(document, "div")[0]
	assertSelectNode(t, groups[0], "optgroup", "axb", 8, 52)
	assertSelectNode(t, options[0], "option", "axb", 18, 52)
	assertSelectNode(t, div, "div", "xb", 27, 52)
	assertSelectNode(t, groups[1], "optgroup", "b", 33, 52)
	assertSelectNode(t, options[1], "option", "b", 43, 52)
}

func TestParseOptionAndOptgroupEndScopeWithDescendants(t *testing.T) {
	const optionEnd = `<select><option><span>x</option>y</select>`
	document, err := NewHTMLParser().Parse(optionEnd)
	if err != nil {
		t.Fatal(err)
	}
	option := selectElements(document, "option")[0]
	span := selectElements(document, "span")[0]
	assertSelectNode(t, option, "option", "x", 8, 32)
	assertSelectNode(t, span, "span", "x", 16, 23)
	if span.Parent != option {
		t.Fatalf("Expected descendant span inside explicitly closed option")
	}

	const groupEndBlocked = `<select><optgroup><div><option>x</optgroup>y</select>`
	document, err = NewHTMLParser().Parse(groupEndBlocked)
	if err != nil {
		t.Fatal(err)
	}
	group := selectElements(document, "optgroup")[0]
	div := selectElements(document, "div")[0]
	option = selectElements(document, "option")[0]
	assertSelectNode(t, group, "optgroup", "xy", 8, 44)
	assertSelectNode(t, div, "div", "xy", 18, 44)
	assertSelectNode(t, option, "option", "xy", 23, 44)
	if len(option.Children) != 1 || option.Children[0].Value != "xy" || option.Children[0].StartPos != 31 || option.Children[0].EndPos != 44 {
		t.Fatalf("Expected blocked optgroup end to remain in option raw text range, got %#v", option.Children)
	}
}

func TestParseCurrentSelectKeepsTextareaAndKeygenButInputEndsSelect(t *testing.T) {
	const textarea = `<select><option>a<textarea>b&amp;c</textarea><option>d</select>`
	document, err := NewHTMLParser().Parse(textarea)
	if err != nil {
		t.Fatal(err)
	}
	options := selectElements(document, "option")
	assertSelectNode(t, options[0], "option", "ab&c", 8, 45)
	assertSelectNode(t, selectElements(document, "textarea")[0], "textarea", "b&c", 17, 45)
	assertSelectNode(t, options[1], "option", "d", 45, 54)

	const keygen = `<select><option>a<keygen name=k><option>b</select>`
	document, err = NewHTMLParser().Parse(keygen)
	if err != nil {
		t.Fatal(err)
	}
	options = selectElements(document, "option")
	assertSelectNode(t, options[0], "option", "a", 8, 32)
	if len(selectElements(options[0], "keygen")) != 1 || selectElements(options[0], "keygen")[0].StartPos != 17 || selectElements(options[0], "keygen")[0].EndPos != 32 {
		t.Fatalf("Expected keygen retained in first option")
	}
	assertSelectNode(t, options[1], "option", "b", 32, 41)

	const input = `<select><option>a<input value=x><option>b</select>tail`
	document, err = NewHTMLParser().Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	inputNode := selectElements(document, "input")[0]
	options = selectElements(document, "option")
	assertSelectNode(t, selectNode, "select", "a", 0, 17)
	assertSelectNode(t, options[0], "option", "a", 8, 17)
	if inputNode.Parent != parsedBody(document) || inputNode.StartPos != 17 || inputNode.EndPos != 32 {
		t.Fatalf("Expected input to end select and reprocess outside it, got %#v", inputNode)
	}
	assertSelectNode(t, options[1], "option", "btail", 32, 54)
}

func TestParseNestedSelectStartAndSelectEndRecovery(t *testing.T) {
	const nested = `<select><option>a<select><option>b</select>tail`
	document, err := NewHTMLParser().Parse(nested)
	if err != nil {
		t.Fatal(err)
	}
	selectNodes := selectElements(document, "select")
	optionNodes := selectElements(document, "option")
	if len(selectNodes) != 1 || len(optionNodes) != 2 {
		t.Fatalf("Expected nested select start to close existing select, got select=%#v option=%#v", selectNodes, optionNodes)
	}
	assertSelectNode(t, selectNodes[0], "select", "a", 0, 17)
	assertSelectNode(t, optionNodes[0], "option", "a", 8, 17)
	assertSelectNode(t, optionNodes[1], "option", "btail", 25, 47)
	if optionNodes[1].Parent != parsedBody(document) {
		t.Fatalf("Expected later option outside closed select")
	}

	const explicit = `<div><select><option>a</select><p>b</p></div>`
	document, err = NewHTMLParser().Parse(explicit)
	if err != nil {
		t.Fatal(err)
	}
	assertSelectNode(t, selectElements(document, "select")[0], "select", "a", 5, 31)
	assertSelectNode(t, selectElements(document, "option")[0], "option", "a", 13, 22)
	assertSelectNode(t, selectElements(document, "p")[0], "p", "b", 31, 39)
}

func TestParseSelectAllowsCommentsAndScript(t *testing.T) {
	const content = `<select>a<!--c--><script>x<y</script><option>b</select>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	assertSelectNode(t, selectNode, "select", "ax<yb", 0, 55)
	assertSelectNode(t, selectElements(document, "script")[0], "script", "x<y", 17, 37)
	comments := findAllNodesByType(document, types.CommentNode)
	if len(comments) != 1 || comments[0].Value != "c" || comments[0].StartPos != 9 || comments[0].EndPos != 17 || comments[0].Parent != selectNode {
		t.Fatalf("Expected comment retained directly in select, got %#v", comments)
	}
}

func TestParseSelectEOFAndMultibyteCRLFLocations(t *testing.T) {
	const eof = `<div><select><optgroup><option>x`
	document, err := NewHTMLParser().Parse(eof)
	if err != nil {
		t.Fatal(err)
	}
	assertSelectNode(t, selectElements(document, "div")[0], "div", "x", 0, len(eof))
	assertSelectNode(t, selectElements(document, "select")[0], "select", "x", 5, len(eof))
	assertSelectNode(t, selectElements(document, "optgroup")[0], "optgroup", "x", 13, len(eof))
	assertSelectNode(t, selectElements(document, "option")[0], "option", "x", 23, len(eof))

	const multiline = "<select>\r\n<option>😀é\r\n<span>x<option>z</select>"
	document, err = NewHTMLParser().Parse(multiline)
	if err != nil {
		t.Fatal(err)
	}
	options := selectElements(document, "option")
	span := selectElements(document, "span")[0]
	assertSelectNode(t, options[0], "option", "😀é\nxz", 10, 42)
	assertSelectNode(t, span, "span", "xz", 26, 42)
	assertSelectNode(t, options[1], "option", "z", 33, 42)
	if options[0].StartLine != 2 || options[0].StartColumn != 1 || options[0].EndLine != 3 || options[0].EndColumn != 17 || span.StartLine != 3 || span.StartColumn != 1 || span.EndColumn != 17 {
		t.Fatalf("Expected UTF-16 coordinates 2:1-3:17 and span 3:1-3:17, got option=%d:%d-%d:%d span=%d:%d-%d:%d", options[0].StartLine, options[0].StartColumn, options[0].EndLine, options[0].EndColumn, span.StartLine, span.StartColumn, span.EndLine, span.EndColumn)
	}
}

func TestParseSelectSelfClosingAndEndTagAttributes(t *testing.T) {
	const selfClosing = `<select><option/>a<option>b</select>`
	document, err := NewHTMLParser().Parse(selfClosing)
	if err != nil {
		t.Fatal(err)
	}
	options := selectElements(document, "option")
	assertSelectNode(t, options[0], "option", "a", 8, 18)
	assertSelectNode(t, options[1], "option", "b", 18, 27)

	const endAttributes = `<select><option>a</option x/><optgroup><option>b</optgroup y/></select>`
	document, err = NewHTMLParser().Parse(endAttributes)
	if err != nil {
		t.Fatal(err)
	}
	options = selectElements(document, "option")
	assertSelectNode(t, options[0], "option", "a", 8, 29)
	assertSelectNode(t, options[1], "option", "b", 39, 48)
	assertSelectNode(t, selectElements(document, "optgroup")[0], "optgroup", "b", 29, 62)
}

func TestParseSelectRecoveryParserReuseAndScaling(t *testing.T) {
	parser := NewHTMLParser()
	for _, content := range []string{`<select><option>a<option>b</select>`, `<select><optgroup><option>x<optgroup><option>y</select>`} {
		document, err := parser.Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		options := selectElements(document, "option")
		if len(options) != 2 || options[0].TextContent != "a" && options[0].TextContent != "x" || options[1].TextContent != "b" && options[1].TextContent != "y" {
			t.Fatalf("Unexpected parser reuse option structure: %#v", options)
		}
	}
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n*10 + 18)
		b.WriteString(`<select>`)
		for i := 0; i < n; i++ {
			b.WriteString(`<option>x`)
		}
		b.WriteString(`</select>`)
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
	small, large := measure(500), measure(2000)
	ratio := float64(large) / float64(small)
	t.Logf("select option scaling 500=%v 2000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 10*time.Millisecond {
		t.Fatalf("Select option recovery scaled superlinearly: 500=%v 2000=%v ratio=%.1fx", small, large, ratio)
	}
}

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

func TestParseSelectItemEndScopesRespectListBoundary(t *testing.T) {
	const listItem = `<select><li><ul>x</li>y</select>`
	document, err := NewHTMLParser().Parse(listItem)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	li := selectElements(selectNode, "li")[0]
	ul := selectElements(li, "ul")[0]
	assertSelectNode(t, li, "li", "xy", 8, 23)
	assertSelectNode(t, ul, "ul", "xy", 12, 23)
	if len(ul.Children) != 1 || ul.Children[0].Value != "xy" || ul.Children[0].StartPos != 16 || ul.Children[0].EndPos != 23 {
		t.Fatalf("Expected li end ignored behind inner ul list-item-scope boundary, got %#v", ul.Children)
	}

	for _, itemName := range []string{"dd", "dt"} {
		content := `<select><` + itemName + `><ul>x</` + itemName + `>y</select>`
		document, err = NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse failed for %s ordinary-scope contrast: %v", itemName, err)
		}
		selectNode = selectElements(document, "select")[0]
		item := selectElements(selectNode, itemName)[0]
		ul = selectElements(item, "ul")[0]
		assertSelectNode(t, item, itemName, "x", 8, 22)
		assertSelectNode(t, ul, "ul", "x", 12, 17)
		if len(selectNode.Children) != 2 || selectNode.Children[0] != item || selectNode.Children[1].Value != "y" || selectNode.Children[1].StartPos != 22 || selectNode.Children[1].EndPos != 23 {
			t.Fatalf("Expected %s end to cross ul in ordinary scope and put y under select, got %#v", itemName, selectNode.Children)
		}
	}
}

func TestParseSelectSpecialStartsMergeHTMLAndIgnoreBodyHead(t *testing.T) {
	const htmlStart = `<html id=a><body><select><option>x<html lang=z>y</select></body></html>`
	document, err := NewHTMLParser().Parse(htmlStart)
	if err != nil {
		t.Fatal(err)
	}
	htmlNodes := selectElements(document, "html")
	if len(htmlNodes) != 1 || htmlNodes[0].Attributes["id"] != "a" || htmlNodes[0].Attributes["lang"] != "z" {
		t.Fatalf("Expected duplicate html start to merge lang=z into the one html node, got %#v", htmlNodes)
	}
	assertSelectNode(t, selectElements(document, "option")[0], "option", "xy", 25, 48)
	if text := selectElements(document, "option")[0].Children[0]; text.Value != "xy" || text.StartPos != 33 || text.EndPos != 48 {
		t.Fatalf("Expected ignored duplicate html token within coalesced option text, got %#v", text)
	}

	const bodyStart = `<html><body id=a><select><option>x<body class=z>y</select></body></html>`
	document, err = NewHTMLParser().Parse(bodyStart)
	if err != nil {
		t.Fatal(err)
	}
	bodyNodes := selectElements(document, "body")
	if len(bodyNodes) != 1 || bodyNodes[0].Attributes["id"] != "a" {
		t.Fatalf("Expected one original body, got %#v", bodyNodes)
	}
	if _, exists := bodyNodes[0].Attributes["class"]; exists {
		t.Fatalf("Chrome ignores duplicate body attributes behind select scope, got %#v", bodyNodes[0].Attributes)
	}
	assertSelectNode(t, selectElements(document, "option")[0], "option", "xy", 25, 49)

	const headStart = `<html><body><select><option>x<head>y</select></body></html>`
	document, err = NewHTMLParser().Parse(headStart)
	if err != nil {
		t.Fatal(err)
	}
	headNodes := selectElements(document, "head")
	if len(headNodes) != 1 || headNodes[0].StartPos != 0 || headNodes[0].EndPos != 0 || len(headNodes[0].Children) != 0 {
		t.Fatalf("Explicit html/body must retain one locationless synthetic head while ignoring the select-scoped head start: %#v", headNodes)
	}
	assertSelectNode(t, selectElements(document, "option")[0], "option", "xy", 20, 36)
}

func TestParseRootSelectSpecialStartsTerminateWithoutDocumentTargets(t *testing.T) {
	if helperCase := os.Getenv("XPATH_SELECT_SPECIAL_HELPER"); helperCase != "" {
		content := `<select>a<html lang=z>b</select>`
		if helperCase == "body" {
			content = `<select>a<body class=z>b</select>`
		}
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		selectNode := selectElements(document, "select")[0]
		assertSelectNode(t, selectNode, "select", "ab", 0, len(content))
		if len(selectNode.Children) != 1 || selectNode.Children[0].Value != "ab" || selectNode.Children[0].StartPos != 8 || selectNode.Children[0].EndPos != len(content)-9 {
			t.Fatalf("Expected special start ignored inside coalesced root select text, got %#v", selectNode.Children)
		}
		if helperCase == "html" {
			htmlNodes := selectElements(document, "html")
			if len(htmlNodes) > 1 {
				t.Fatalf("Expected at most one html target, got %#v", htmlNodes)
			}
		}
		return
	}

	for _, helperCase := range []string{"html", "body"} {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestParseRootSelectSpecialStartsTerminateWithoutDocumentTargets$")
		command.Env = append(os.Environ(), "XPATH_SELECT_SPECIAL_HELPER="+helperCase)
		output, err := command.CombinedOutput()
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("Root select %s start hung without a document target", helperCase)
		}
		if err != nil {
			t.Fatalf("Root select %s helper failed: %v\n%s", helperCase, err, output)
		}
	}

	parser := NewHTMLParser()
	for _, content := range []string{`<select>a<html lang=z>b</select>`, `<select>a<body class=z>b</select>`} {
		document, err := parser.Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		if selectElements(document, "select")[0].TextContent != "ab" {
			t.Fatalf("Expected parser reuse text ab for %q", content)
		}
	}
}

func TestParseMismatchedHeadingGroupEndInsideSelect(t *testing.T) {
	const plain = `<select><h1>x</h2>y</select>`
	document, err := NewHTMLParser().Parse(plain)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	heading := selectElements(selectNode, "h1")[0]
	assertSelectNode(t, heading, "h1", "x", 8, 13)
	if len(selectNode.Children) != 2 || selectNode.Children[1].Value != "y" || selectNode.Children[1].StartPos != 18 || selectNode.Children[1].EndPos != 19 {
		t.Fatalf("Expected y after mismatched heading-group end, got %#v", selectNode.Children)
	}

	const spanCase = `<select><h1><span>x</h2>y</select>`
	document, err = NewHTMLParser().Parse(spanCase)
	if err != nil {
		t.Fatal(err)
	}
	selectNode = selectElements(document, "select")[0]
	heading = selectElements(selectNode, "h1")[0]
	span := selectElements(heading, "span")[0]
	assertSelectNode(t, heading, "h1", "x", 8, 19)
	assertSelectNode(t, span, "span", "x", 12, 19)
	if selectNode.Children[1].Value != "y" || selectNode.Children[1].StartPos != 24 || selectNode.Children[1].EndPos != 25 {
		t.Fatalf("Expected y after heading/span unwind, got %#v", selectNode.Children)
	}

	const optionCase = `<select><option><h1><span>x</h2>y</option></select>`
	document, err = NewHTMLParser().Parse(optionCase)
	if err != nil {
		t.Fatal(err)
	}
	option := selectElements(document, "option")[0]
	heading = selectElements(option, "h1")[0]
	span = selectElements(heading, "span")[0]
	assertSelectNode(t, option, "option", "xy", 8, 42)
	assertSelectNode(t, heading, "h1", "x", 16, 27)
	assertSelectNode(t, span, "span", "x", 20, 27)
	if option.Children[1].Value != "y" || option.Children[1].StartPos != 32 || option.Children[1].EndPos != 33 {
		t.Fatalf("Expected y to remain in option after heading close, got %#v", option.Children)
	}
}

func TestParseLegacyImageAndBrEndBecomeVoidElementsInsideSelect(t *testing.T) {
	const selectImage = `<select>a<image src=x>b</select>`
	document, err := NewHTMLParser().Parse(selectImage)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	images := selectElements(selectNode, "img")
	if len(images) != 1 || images[0].StartPos != 9 || images[0].EndPos != 22 || images[0].Attributes["src"] != "x" || len(images[0].Children) != 0 {
		t.Fatalf("Expected source image token renamed to void img at 9:22, got %#v", images)
	}
	if len(selectNode.Children) != 3 || selectNode.Children[0].Value != "a" || selectNode.Children[0].StartPos != 8 || selectNode.Children[0].EndPos != 9 || selectNode.Children[2].Value != "b" || selectNode.Children[2].StartPos != 22 || selectNode.Children[2].EndPos != 23 {
		t.Fatalf("Expected text to continue around void img, got %#v", selectNode.Children)
	}

	const optionImage = `<select><option>a<image src=x>b</option></select>`
	document, err = NewHTMLParser().Parse(optionImage)
	if err != nil {
		t.Fatal(err)
	}
	option := selectElements(document, "option")[0]
	images = selectElements(option, "img")
	assertSelectNode(t, option, "option", "ab", 8, 40)
	if len(images) != 1 || images[0].StartPos != 17 || images[0].EndPos != 30 || images[0].Parent != option || option.Children[2].Value != "b" || option.Children[2].StartPos != 30 || option.Children[2].EndPos != 31 {
		t.Fatalf("Expected void img inside option with following b, got option=%#v image=%#v", option.Children, images)
	}

	const brEnd = `<select><option>a</br>b</option></select>`
	document, err = NewHTMLParser().Parse(brEnd)
	if err != nil {
		t.Fatal(err)
	}
	option = selectElements(document, "option")[0]
	brNodes := selectElements(option, "br")
	if len(brNodes) != 1 {
		t.Fatalf("Expected br end reprocessed as one br start, got %#v", brNodes)
	}
	br := brNodes[0]
	assertSelectNode(t, option, "option", "ab", 8, 32)
	if br.StartPos != 17 || br.EndPos != 22 || br.Parent != option || option.Children[2].Value != "b" || option.Children[2].StartPos != 22 || option.Children[2].EndPos != 23 {
		t.Fatalf("Expected br end reprocessed as void br start, got br=%#v option=%#v", br, option.Children)
	}
}

func TestParseIgnoredSelectEndTokenRangeDoesNotLeakAcrossElements(t *testing.T) {
	const divCase = `<select><div></foo></div>x</select>`
	document, err := NewHTMLParser().Parse(divCase)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	div := selectElements(selectNode, "div")[0]
	assertSelectNode(t, div, "div", "", 8, 25)
	if len(selectNode.Children) != 2 || selectNode.Children[1].Value != "x" || selectNode.Children[1].StartPos != 25 || selectNode.Children[1].EndPos != 26 {
		t.Fatalf("Expected ignored token range confined before outer x, got %#v", selectNode.Children)
	}

	const spanCase = `<select><option><span></foo></span>x</option></select>`
	document, err = NewHTMLParser().Parse(spanCase)
	if err != nil {
		t.Fatal(err)
	}
	option := selectElements(document, "option")[0]
	span := selectElements(option, "span")[0]
	assertSelectNode(t, span, "span", "", 16, 35)
	if len(option.Children) != 2 || option.Children[1].Value != "x" || option.Children[1].StartPos != 35 || option.Children[1].EndPos != 36 {
		t.Fatalf("Expected ignored token range confined before option x, got %#v", option.Children)
	}

	const innerElement = `<select><div></foo><span>a</span></div>x</select>`
	document, err = NewHTMLParser().Parse(innerElement)
	if err != nil {
		t.Fatal(err)
	}
	selectNode = selectElements(document, "select")[0]
	div = selectElements(selectNode, "div")[0]
	span = selectElements(div, "span")[0]
	assertSelectNode(t, div, "div", "a", 8, 39)
	assertSelectNode(t, span, "span", "a", 19, 33)
	if span.Children[0].StartPos != 25 || span.Children[0].EndPos != 26 || selectNode.Children[1].Value != "x" || selectNode.Children[1].StartPos != 39 || selectNode.Children[1].EndPos != 40 {
		t.Fatalf("Expected ignored token not to leak into later inner/outer text, got span=%#v select=%#v", span.Children, selectNode.Children)
	}
}

func TestParseIgnoredBodyHTMLEndsInsideSelectDoNotLeakEOFRecovery(t *testing.T) {
	const explicit = `<html><body><select>a</body>b</html>c</select></body></html><foo>x`
	document, err := NewHTMLParser().Parse(explicit)
	if err != nil {
		t.Fatalf("Qualifying explicit document must EOF-close later foo: %v", err)
	}
	foo := selectElements(document, "foo")
	if len(foo) != 1 || foo[0].TextContent != "x" || foo[0].EndPos != len(explicit) {
		t.Fatalf("Explicit document foo EOF recovery mismatch: %#v", foo)
	}

	const legacy = `<select>a</body>b</html>c</select><foo>x`
	document, err = NewHTMLParser().Parse(legacy)
	foo = selectElements(document, "foo")
	if err != nil || len(foo) != 1 || foo[0].EndPos != len(legacy) {
		t.Fatalf("Implicit-document select/foo EOF mismatch for %q: %#v err=%v", legacy, foo, err)
	}

	for _, content := range []string{
		`<html><body><select>a</body>b`,
		`<select>a</body>b`,
	} {
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Expected unresolved select EOF recovery for %q: %v", content, err)
		}
		selectNode := selectElements(document, "select")[0]
		assertSelectNode(t, selectNode, "select", "ab", strings.Index(content, `<select>`), len(content))
	}

	parser := NewHTMLParser()
	if _, err := parser.Parse(explicit); err != nil {
		t.Fatalf("Expected reused parser explicit EOF recovery for %q: %v", explicit, err)
	}
	if document, err := parser.Parse(legacy); err != nil || len(selectElements(document, "foo")) != 1 || selectElements(document, "foo")[0].EndPos != len(legacy) {
		t.Fatalf("Expected explicit-to-implicit document reuse for %q: doc=%#v err=%v", legacy, document, err)
	}
	for _, content := range []string{`<select>a</body>b`, `<select>x`} {
		if _, err := parser.Parse(content); err != nil {
			t.Fatalf("Expected reused parser unresolved select recovery for %q: %v", content, err)
		}
	}
}
