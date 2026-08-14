package utils

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 20 covers current-WHATWG processing instructions in whole HTML
// documents. XML parsing, template/fragment parsing, execution or external
// entity semantics, and legacy jsdom's comment representation are deferred.

func processingInstructions(node *types.Node) []*types.Node {
	if node == nil {
		return nil
	}
	var result []*types.Node
	if node.Type == types.ProcessingInstructionNode {
		result = append(result, node)
	}
	for _, child := range node.Children {
		result = append(result, processingInstructions(child)...)
	}
	return result
}

func processingComments(node *types.Node) []*types.Node {
	if node == nil {
		return nil
	}
	var result []*types.Node
	if node.Type == types.CommentNode {
		result = append(result, node)
	}
	for _, child := range node.Children {
		result = append(result, processingComments(child)...)
	}
	return result
}

func requireProcessingInstruction(t *testing.T, node *types.Node, target, data string, start, end int) {
	t.Helper()
	if node == nil || node.Type != types.ProcessingInstructionNode || node.Name != target || node.Value != data || node.TextContent != data || node.StartPos != start || node.EndPos != end || node.NamespaceURI != "" {
		t.Fatalf("processing instruction %q=%q at %d:%d mismatch: %#v", target, data, start, end, node)
	}
}

func TestParseProcessingInstructionTargetsDataAndQuestionStates(t *testing.T) {
	const content = `<div id=h>a<?Foo?>b<?bar data?>c<?baz data with ? &amp;?>d<?bare data>e<?questions ?x??>f<?tight?x?>g<?trail x   ?>h<?A-B hyphen?>i<?_under data?>j</div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	div := svgNodeByID(t, document, "h")
	if div == nil || div.TextContent != "abcdefghij" {
		t.Fatalf("PI data polluted element textContent: %#v", div)
	}
	wants := []struct{ target, data, raw string }{
		{"foo", "", `<?Foo?>`},
		{"bar", "data", `<?bar data?>`},
		{"baz", "data with ? &amp;", `<?baz data with ? &amp;?>`},
		{"bare", "data", `<?bare data>`},
		{"questions", "?x?", `<?questions ?x??>`},
		{"tight", "?x", `<?tight?x?>`},
		{"trail", "x   ", `<?trail x   ?>`},
		{"a-b", "hyphen", `<?A-B hyphen?>`},
		{"_under", "data", `<?_under data?>`},
	}
	got := processingInstructions(div)
	if len(got) != len(wants) {
		t.Fatalf("got %d processing instructions, want %d: %#v", len(got), len(wants), got)
	}
	search := 0
	for i, want := range wants {
		start := strings.Index(content[search:], want.raw) + search
		requireProcessingInstruction(t, got[i], want.target, want.data, start, start+len(want.raw))
		search = start + len(want.raw)
		if got[i].Parent != div {
			t.Fatalf("PI %q parent mismatch: %#v", want.target, got[i].Parent)
		}
	}
}

func TestParseInvalidProcessingInstructionTargetsBecomeBogusComments(t *testing.T) {
	const content = `<div id=h>a<?xml bad?>b<?XML-STYLESHEET x?>c<?1bad?>d<?>e<?a.b?>f<?a:b?>g</div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	div := svgNodeByID(t, document, "h")
	if got := processingInstructions(div); len(got) != 0 {
		t.Fatalf("invalid/disallowed targets emitted PIs: %#v", got)
	}
	wants := []struct{ raw, value string }{
		{`<?xml bad?>`, `?xml bad?`},
		{`<?XML-STYLESHEET x?>`, `?XML-STYLESHEET x?`},
		{`<?1bad?>`, `?1bad?`},
		{`<?>`, `?`},
		{`<?a.b?>`, `?a.b?`},
		{`<?a:b?>`, `?a:b?`},
	}
	comments := processingComments(div)
	if len(comments) != len(wants) || div.TextContent != "abcdefg" {
		t.Fatalf("bogus PI fallback tree mismatch: comments=%#v div=%#v", comments, div)
	}
	search := 0
	for i, want := range wants {
		start := strings.Index(content[search:], want.raw) + search
		comment := comments[i]
		if comment.Name != "#comment" || comment.Value != want.value || comment.TextContent != want.value || comment.StartPos != start || comment.EndPos != start+len(want.raw) || comment.Parent != div {
			t.Fatalf("bogus PI comment %d mismatch: %#v", i, comment)
		}
		search = start + len(want.raw)
	}
}

func TestParseProcessingInstructionEOFAndParserReuse(t *testing.T) {
	parser := NewHTMLParser()
	for _, content := range []string{`<?`, `<div><?`, `<div>x</div><?foo`, `<div>x</div><?foo data`} {
		document, err := parser.Parse(content)
		if err != nil {
			t.Fatalf("valid incomplete PI at EOF should be discarded: %v", err)
		}
		if len(processingInstructions(document)) != 0 || len(processingComments(document)) != 0 {
			t.Fatalf("valid EOF PI emitted a node: %#v", document)
		}
		if strings.Contains(content, `<div>x`) && svgNodesNamed(document, "div")[0].TextContent != "x" {
			t.Fatalf("discarded EOF PI changed preceding content: %#v", document)
		}
	}
	const invalid = `<div>x</div><?1foo data`
	document, err := parser.Parse(invalid)
	if err != nil {
		t.Fatal(err)
	}
	comments := processingComments(document)
	if len(comments) != 1 || comments[0].Value != `?1foo data` || comments[0].StartPos != 12 || comments[0].EndPos != len(invalid) {
		t.Fatalf("invalid EOF PI must emit a bogus comment through EOF: %#v", comments)
	}
	if document, parseErr := parser.Parse(`<?foo?><section>x`); parseErr != nil || len(listElements(document, "section")) != 1 || listElements(document, "section")[0].EndPos != len(`<?foo?><section>x`) {
		t.Fatalf("PI-to-implicit-document EOF reuse mismatch: doc=%#v err=%v", document, parseErr)
	}
	document, err = parser.Parse(`<div><?ok data?></div>`)
	if err != nil || len(processingInstructions(document)) != 1 || processingInstructions(document)[0].Name != "ok" {
		t.Fatalf("parser reuse after PI EOF recovery was not clean: document=%#v err=%v", document, err)
	}
}

func TestParseProcessingInstructionUnicodeNULAndCoordinates(t *testing.T) {
	const content = "<div>é\r\n<?MiXeD 😀\r\nx?>z</div>"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	instructions := processingInstructions(document)
	if len(instructions) != 1 {
		t.Fatalf("unicode PI missing: %#v", document)
	}
	pi := instructions[0]
	requireProcessingInstruction(t, pi, "mixed", "😀\nx", 9, 26)
	if pi.StartLine != 2 || pi.StartColumn != 1 || pi.EndLine != 3 || pi.EndColumn != 4 {
		t.Fatalf("PI byte/UTF-16 line-column mismatch: %#v", pi)
	}

	const nul = "<div><?foo a\x00b?>x</div>"
	document, err = NewHTMLParser().Parse(nul)
	if err != nil {
		t.Fatal(err)
	}
	pi = processingInstructions(document)[0]
	requireProcessingInstruction(t, pi, "foo", "a�b", 5, 16)
	if svgNodesNamed(document, "div")[0].TextContent != "x" {
		t.Fatalf("PI NUL data polluted parent text: %#v", document)
	}
}

func TestParseProcessingInstructionDocumentPlacement(t *testing.T) {
	const content = `<?pre?><!doctype html><?post?><html><head><?head data?></head><body><?body?><div id=h><?child?></div></body><?afterbody?></html><?afterhtml?>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	instructions := processingInstructions(document)
	wants := []struct{ target, parent string }{
		{"pre", "#document"}, {"post", "#document"}, {"head", "head"}, {"body", "body"}, {"child", "div"}, {"afterbody", "html"}, {"afterhtml", "#document"},
	}
	if len(instructions) != len(wants) {
		t.Fatalf("document PI placement count mismatch: %#v", instructions)
	}
	for i, want := range wants {
		pi := instructions[i]
		if pi.Name != want.target || pi.Parent == nil || pi.Parent.Name != want.parent {
			t.Fatalf("PI %q placement mismatch: %#v", want.target, pi)
		}
		raw := `<?` + want.target
		start := strings.Index(content, raw)
		end := strings.Index(content[start:], `>`) + start + 1
		if pi.StartPos != start || pi.EndPos != end {
			t.Fatalf("PI %q source range mismatch: %#v", want.target, pi)
		}
	}
}

func TestParseProcessingInstructionsInForeignAndSpecializedContexts(t *testing.T) {
	const content = `<svg id=s><?svg x?></svg><math id=m><mi id=i><?math y?></mi></math><table id=t><?table?><caption id=c><?caption?></caption><colgroup id=g><?group?><col></colgroup><tbody id=b><?tbody?><tr id=r><?row?><td id=d><?cell?>x</td></tr></tbody></table><select id=q><?select?><option>x</option></select>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	wants := []struct{ target, parent, namespace string }{
		{"svg", "svg", svgNamespaceURI}, {"math", "mi", mathMLTestNamespaceURI},
		{"table", "table", htmlNamespaceURI}, {"caption", "caption", htmlNamespaceURI}, {"group", "colgroup", htmlNamespaceURI}, {"tbody", "tbody", htmlNamespaceURI}, {"row", "tr", htmlNamespaceURI}, {"cell", "td", htmlNamespaceURI}, {"select", "select", htmlNamespaceURI},
	}
	instructions := processingInstructions(document)
	if len(instructions) != len(wants) {
		t.Fatalf("specialized PI count mismatch: %#v", instructions)
	}
	for i, want := range wants {
		pi := instructions[i]
		if pi.Name != want.target || pi.Parent == nil || pi.Parent.Name != want.parent || pi.Parent.NamespaceURI != want.namespace {
			t.Fatalf("specialized PI %q placement mismatch: %#v", want.target, pi)
		}
	}
}

func TestParseProcessingInstructionDoesNotTriggerFormattingReconstruction(t *testing.T) {
	const content = `<p><b>x<div id=d><?hold?>y</div>z</b>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	div := svgNodeByID(t, document, "d")
	instructions := processingInstructions(div)
	if len(instructions) != 1 || instructions[0].Parent != div || instructions[0].Name != "hold" {
		t.Fatalf("formatting PI placement mismatch: %#v", div)
	}
	if len(div.Children) != 2 || div.Children[0] != instructions[0] || div.Children[1].Name != "b" || div.Children[1].TextContent != "y" {
		t.Fatalf("PI itself reconstructed formatting or following text did not: %#v", div.Children)
	}
	if div.TextContent != "y" {
		t.Fatalf("PI data polluted formatting aggregate: %#v", div)
	}
}

func bestProcessingInstructionDuration(t *testing.T, content string) time.Duration {
	t.Helper()
	best := time.Duration(1<<63 - 1)
	for i := 0; i < 3; i++ {
		start := time.Now()
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		if document == nil {
			t.Fatal("nil PI scaling document")
		}
		if elapsed := time.Since(start); elapsed < best {
			best = elapsed
		}
	}
	return best
}

func TestParseProcessingInstructionScaling(t *testing.T) {
	build := func(n int) string {
		var out strings.Builder
		out.Grow(n * 28)
		for i := 0; i < n; i++ {
			fmt.Fprintf(&out, `<div><?target data-%d?></div>`, i)
		}
		return out.String()
	}
	smallInput, largeInput := build(1000), build(4000)
	_ = bestProcessingInstructionDuration(t, smallInput)
	small := bestProcessingInstructionDuration(t, smallInput)
	large := bestProcessingInstructionDuration(t, largeInput)
	if small > 5*time.Millisecond && large > 12*small+75*time.Millisecond {
		t.Fatalf("processing-instruction parsing scales superlinearly: 1k=%v 4k=%v", small, large)
	}
}
