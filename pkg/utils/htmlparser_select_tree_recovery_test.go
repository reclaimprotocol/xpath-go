package utils

import (
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
