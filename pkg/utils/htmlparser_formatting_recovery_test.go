package utils

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 14A covers active-formatting reconstruction and simple, correctly
// nested formatting ends. Misnested formatting that requires adoption-agency
// reparenting, nested anchors/nobr, and full formatting tree surgery are
// intentionally deferred to Batch 14B.

var formattingElementNames = []string{
	"a", "b", "big", "code", "em", "font", "i", "nobr", "s", "small",
	"strike", "strong", "tt", "u",
}

func formattingElements(node *types.Node, name string) []*types.Node {
	var matches []*types.Node
	if node == nil {
		return matches
	}
	if node.Type == types.ElementNode && node.Name == name {
		matches = append(matches, node)
	}
	for _, child := range node.Children {
		matches = append(matches, formattingElements(child, name)...)
	}
	return matches
}

func assertFormattingNode(t *testing.T, node *types.Node, name, text string, start, end int) {
	t.Helper()
	if node == nil || node.Name != name || node.TextContent != text || node.StartPos != start || node.EndPos != end {
		t.Fatalf("Expected <%s> text %q at %d:%d, got %#v", name, text, start, end, node)
	}
}

func TestParseFormattingReconstructsAcrossBlockAndParagraphBoundaries(t *testing.T) {
	const block = `<p><b>one<div>two</div>three`
	document, err := NewHTMLParser().Parse(block)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	div := formattingElements(document, "div")[0]
	bold := formattingElements(document, "b")
	if len(bold) != 3 {
		t.Fatalf("Expected original plus two reconstructed b elements, got %#v", bold)
	}
	assertFormattingNode(t, paragraph, "p", "one", 0, 9)
	assertFormattingNode(t, bold[0], "b", "one", 3, 9)
	assertFormattingNode(t, div, "div", "two", 9, 23)
	assertFormattingNode(t, bold[1], "b", "two", 3, 17)
	assertFormattingNode(t, bold[2], "b", "three", 3, 28)
	for _, reconstructed := range bold[1:] {
		if reconstructed.ContentStart != 6 || reconstructed.Attributes == nil || len(reconstructed.AttributeOrder) != 0 {
			t.Fatalf("Expected reconstructed b clone to retain original ContentStart=6 and empty attributes, got %#v", reconstructed)
		}
	}
	if bold[0].Parent != paragraph || bold[1].Parent != div || bold[2].Parent != parsedBody(document) {
		t.Fatalf("Expected reconstructed b parents p/div/document, got %#v", bold)
	}

	const paragraphEnd = `<p><em>a</p>b`
	document, err = NewHTMLParser().Parse(paragraphEnd)
	if err != nil {
		t.Fatal(err)
	}
	ems := formattingElements(document, "em")
	if len(ems) != 2 {
		t.Fatalf("Expected em reconstruction after explicit p end, got %#v", ems)
	}
	assertFormattingNode(t, formattingElements(document, "p")[0], "p", "a", 0, 12)
	assertFormattingNode(t, ems[0], "em", "a", 3, 8)
	assertFormattingNode(t, ems[1], "em", "b", 3, 13)
}

func TestParseSafeAnchorFormattingAndReconstruction(t *testing.T) {
	const content = `<p><a href=x>a</p>b`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	anchors := formattingElements(document, "a")
	if len(anchors) != 2 {
		t.Fatalf("Expected one source and one reconstructed safe anchor, got %#v", anchors)
	}
	assertFormattingNode(t, formattingElements(document, "p")[0], "p", "a", 0, 18)
	assertFormattingNode(t, anchors[0], "a", "a", 3, 14)
	assertFormattingNode(t, anchors[1], "a", "b", 3, 19)
	for _, anchor := range anchors {
		if anchor.Attributes["href"] != "x" || anchor.ContentStart != 13 || len(anchor.AttributeOrder) != 1 || anchor.AttributeOrder[0] != "href" {
			t.Fatalf("Expected cloned href=x anchor metadata and ContentStart=13, got %#v", anchor)
		}
	}
}

func TestParseFormattingReconstructsAcrossListItemBoundary(t *testing.T) {
	const content = `<ul><li><b>a<li>b</ul>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	ul := formattingElements(document, "ul")[0]
	items := formattingElements(ul, "li")
	bold := formattingElements(ul, "b")
	assertFormattingNode(t, ul, "ul", "ab", 0, 22)
	if len(items) != 2 || len(bold) != 2 {
		t.Fatalf("Expected two sibling list items with one b each, got items=%#v bold=%#v", items, bold)
	}
	assertFormattingNode(t, items[0], "li", "a", 4, 12)
	assertFormattingNode(t, bold[0], "b", "a", 8, 12)
	assertFormattingNode(t, items[1], "li", "b", 12, 17)
	assertFormattingNode(t, bold[1], "b", "b", 8, 17)
	if bold[0].Parent != items[0] || bold[1].Parent != items[1] {
		t.Fatalf("Expected reconstructed b in second sibling li, got %#v", bold)
	}
}

func TestParseFormattingReconstructsAcrossDefinitionItemBoundary(t *testing.T) {
	const content = `<dl><dt><i>x<dd>y</dl>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	dl := formattingElements(document, "dl")[0]
	dt := formattingElements(dl, "dt")[0]
	dd := formattingElements(dl, "dd")[0]
	italics := formattingElements(dl, "i")
	assertFormattingNode(t, dl, "dl", "xy", 0, 22)
	assertFormattingNode(t, dt, "dt", "x", 4, 12)
	assertFormattingNode(t, dd, "dd", "y", 12, 17)
	if len(italics) != 2 {
		t.Fatalf("Expected i reconstruction in dd after dt closes, got %#v", italics)
	}
	assertFormattingNode(t, italics[0], "i", "x", 8, 12)
	assertFormattingNode(t, italics[1], "i", "y", 8, 17)
	if italics[0].Parent != dt || italics[1].Parent != dd {
		t.Fatalf("Expected source/reconstructed i under dt/dd siblings, got %#v", italics)
	}
}

func TestParseFormattingCloseOfReconstructedCloneStopsFurtherReconstruction(t *testing.T) {
	const content = `<p><b>x<div>y</b>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	div := formattingElements(document, "div")[0]
	bold := formattingElements(document, "b")
	assertFormattingNode(t, paragraph, "p", "x", 0, 7)
	assertFormattingNode(t, div, "div", "yz", 7, 18)
	if len(bold) != 2 {
		t.Fatalf("Expected source b plus one reconstructed clone, got %#v", bold)
	}
	assertFormattingNode(t, bold[0], "b", "x", 3, 7)
	assertFormattingNode(t, bold[1], "b", "y", 3, 17)
	if len(div.Children) != 2 || div.Children[0] != bold[1] || div.Children[1].Type != types.TextNode || div.Children[1].Value != "z" || div.Children[1].StartPos != 17 || div.Children[1].EndPos != 18 {
		t.Fatalf("Expected clone close to remove replacement entry so z stays plain, got %#v", div.Children)
	}
}

func TestParseFormattingMarkersLimitReconstruction(t *testing.T) {
	const content = `<button><b>x</button>y<object><i>z</object>w`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	button := formattingElements(document, "button")[0]
	object := formattingElements(document, "object")[0]
	bold := formattingElements(document, "b")
	italics := formattingElements(document, "i")
	if len(bold) != 2 || len(italics) != 1 {
		t.Fatalf("Expected b reconstruction across button but object marker to suppress i reconstruction, got b=%#v i=%#v", bold, italics)
	}
	assertFormattingNode(t, button, "button", "x", 0, 21)
	assertFormattingNode(t, bold[0], "b", "x", 8, 12)
	assertFormattingNode(t, bold[1], "b", "yzw", 8, 44)
	assertFormattingNode(t, object, "object", "z", 22, 43)
	assertFormattingNode(t, italics[0], "i", "z", 30, 34)
	if bold[1].Parent != parsedBody(document) || object.Parent != bold[1] || italics[0].Parent != object {
		t.Fatalf("Unexpected marker/reconstruction hierarchy: b=%#v object=%#v i=%#v", bold[1], object, italics[0])
	}
}

func TestParseFormattingMarkerMatrixAndNegativeMarkers(t *testing.T) {
	const cell = `<table><tr><td><b>x<td>y</table>`
	document, err := NewHTMLParser().Parse(cell)
	if err != nil {
		t.Fatal(err)
	}
	table := tableElements(document, "table")[0]
	row := tableElements(table, "tr")[0]
	cells := tableElements(row, "td")
	bold := formattingElements(table, "b")
	assertFormattingNode(t, table, "table", "xy", 0, 32)
	assertFormattingNode(t, tableElements(table, "tbody")[0], "tbody", "xy", 0, 0)
	assertFormattingNode(t, row, "tr", "xy", 7, 24)
	assertFormattingNode(t, cells[0], "td", "x", 11, 19)
	assertFormattingNode(t, bold[0], "b", "x", 15, 19)
	assertFormattingNode(t, cells[1], "td", "y", 19, 24)
	if len(bold) != 1 {
		t.Fatalf("Expected cell marker to prevent b reconstruction in second cell, got %#v", bold)
	}

	const caption = `<table><caption><b>x<tr><td>y</table>`
	document, err = NewHTMLParser().Parse(caption)
	if err != nil {
		t.Fatal(err)
	}
	table = tableElements(document, "table")[0]
	captionNode := tableElements(table, "caption")[0]
	bold = formattingElements(table, "b")
	assertFormattingNode(t, table, "table", "xy", 0, 37)
	assertFormattingNode(t, captionNode, "caption", "x", 7, 20)
	assertFormattingNode(t, bold[0], "b", "x", 16, 20)
	assertFormattingNode(t, tableElements(table, "tr")[0], "tr", "y", 20, 29)
	if len(bold) != 1 {
		t.Fatalf("Expected caption marker to prevent b reconstruction in row, got %#v", bold)
	}

	for _, testCase := range []struct {
		name                 string
		content              string
		containerEnd, bStart int
	}{
		{name: "marquee", content: `<marquee><b>x</marquee>y`, containerEnd: 23, bStart: 9},
		{name: "applet", content: `<applet><b>x</applet>y`, containerEnd: 21, bStart: 8},
		{name: "object", content: `<object><b>x</object>y`, containerEnd: 21, bStart: 8},
	} {
		document, err = NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatalf("Parse failed for %s marker: %v", testCase.name, err)
		}
		containers := formattingElements(document, testCase.name)
		bold = formattingElements(document, "b")
		assertFormattingNode(t, containers[0], testCase.name, "x", 0, testCase.containerEnd)
		assertFormattingNode(t, bold[0], "b", "x", testCase.bStart, testCase.containerEnd-len("</"+testCase.name+">"))
		if len(bold) != 1 || len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].Value != "y" || parsedBodyChildren(document)[1].StartPos != testCase.containerEnd || parsedBodyChildren(document)[1].EndPos != len(testCase.content) {
			t.Fatalf("Expected %s marker to clear b before y, got %#v", testCase.name, parsedBodyChildren(document))
		}
	}

	for _, testCase := range []struct {
		name, content string
	}{
		{name: "button", content: `<button><b>x</button>y`},
		{name: "select", content: `<select><b>x</select>y`},
	} {
		document, err = NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatalf("Parse failed for negative marker %s: %v", testCase.name, err)
		}
		bold = formattingElements(document, "b")
		if len(bold) != 2 || bold[0].TextContent != "x" || bold[1].TextContent != "y" || bold[1].Parent != parsedBody(document) || bold[1].StartPos != len("<"+testCase.name+">") || bold[1].EndPos != len(testCase.content) {
			t.Fatalf("Expected %s not to clear active b entry, got %#v", testCase.name, bold)
		}
	}
}

func TestParseFormattingNoahsArkCapsEquivalentEntries(t *testing.T) {
	const content = `<p><b class=x><b class=x><b class=x><b class=x>a</p>b`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	bold := formattingElements(document, "b")
	if len(bold) != 7 {
		t.Fatalf("Expected four source b nodes and three reconstructed Noah's Ark entries, got %d: %#v", len(bold), bold)
	}
	assertFormattingNode(t, paragraph, "p", "a", 0, 52)
	for i, start := range []int{3, 14, 25, 36} {
		assertFormattingNode(t, bold[i], "b", "a", start, 48)
		if bold[i].Attributes["class"] != "x" {
			t.Fatalf("Expected class=x on original b %d, got %#v", i, bold[i].Attributes)
		}
	}
	for i, start := range []int{14, 25, 36} {
		assertFormattingNode(t, bold[i+4], "b", "b", start, 53)
		if bold[i+4].Attributes["class"] != "x" {
			t.Fatalf("Expected class=x on reconstructed b %d, got %#v", i, bold[i+4].Attributes)
		}
	}
	if bold[4].Parent != parsedBody(document) || bold[5].Parent != bold[4] || bold[6].Parent != bold[5] {
		t.Fatalf("Expected exactly three nested reconstructed b nodes, got %#v", bold[4:])
	}
}

func TestParseFormattingNoahsArkAttributeEquivalenceAndMarkerReset(t *testing.T) {
	const reordered = `<p><b a=1 b=2><b b=2 a=1><b a=1 b=2><b b=2 a=1>x</p>y`
	document, err := NewHTMLParser().Parse(reordered)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	if len(bold) != 7 {
		t.Fatalf("Attribute order must not change Noah equivalence; expected 4+3 b nodes, got %#v", bold)
	}
	for i, start := range []int{14, 25, 36} {
		reconstructed := bold[i+4]
		assertFormattingNode(t, reconstructed, "b", "y", start, 53)
		if reconstructed.ContentStart != []int{25, 36, 47}[i] || reconstructed.Attributes["a"] != "1" || reconstructed.Attributes["b"] != "2" {
			t.Fatalf("Expected reconstructed attribute clone %d, got %#v", i, reconstructed)
		}
	}
	if got := bold[4].AttributeOrder; len(got) != 2 || got[0] != "b" || got[1] != "a" {
		t.Fatalf("Expected reconstructed clone to preserve source attribute order, got %#v", got)
	}

	const distinct = `<p><b x=1><b x=2><b y=1><b y=2>x</p>y`
	document, err = NewHTMLParser().Parse(distinct)
	if err != nil {
		t.Fatal(err)
	}
	bold = formattingElements(document, "b")
	if len(bold) != 8 {
		t.Fatalf("Distinct attribute names/values must preserve all four entries, got %#v", bold)
	}
	for i, start := range []int{3, 10, 17, 24} {
		assertFormattingNode(t, bold[i+4], "b", "y", start, 37)
	}

	const marker = `<object><b><b><b><b>x</object>y`
	document, err = NewHTMLParser().Parse(marker)
	if err != nil {
		t.Fatal(err)
	}
	bold = formattingElements(document, "b")
	if len(bold) != 4 {
		t.Fatalf("Object marker must discard all equivalent entries rather than reconstruct after close, got %#v", bold)
	}
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].Value != "y" || parsedBodyChildren(document)[1].StartPos != 30 || parsedBodyChildren(document)[1].EndPos != 31 {
		t.Fatalf("Expected y outside object with no reconstructed b, got %#v", parsedBodyChildren(document))
	}
}

func TestParseFormattingNoahsArkEvictionUsesNodeIdentity(t *testing.T) {
	const content = `<b><b><b><b>x</b></b></b></b>y`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	if len(bold) != 4 {
		t.Fatalf("Expected four nested source b nodes, got %#v", bold)
	}
	starts := []int{0, 3, 6, 9}
	ends := []int{29, 25, 21, 17}
	for i := range bold {
		assertFormattingNode(t, bold[i], "b", "x", starts[i], ends[i])
	}
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0] != bold[0] || parsedBodyChildren(document)[1].Type != types.TextNode || parsedBodyChildren(document)[1].Value != "y" || parsedBodyChildren(document)[1].StartPos != 29 || parsedBodyChildren(document)[1].EndPos != 30 {
		t.Fatalf("Expected outer evicted b close to pop by node identity and leave y plain, got %#v", parsedBodyChildren(document))
	}
}

func TestParseFormattingMarkersPreserveEntriesBeforeMarker(t *testing.T) {
	const noah = `<p><b><b><b>x<object><b>y</object></p>z`
	document, err := NewHTMLParser().Parse(noah)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	bold := formattingElements(document, "b")
	if len(bold) != 7 {
		t.Fatalf("Expected three entries before the object marker to reconstruct, with the inner entry cleared, got %#v", bold)
	}
	assertFormattingNode(t, paragraph, "p", "xy", 0, 38)
	for i, start := range []int{3, 6, 9} {
		assertFormattingNode(t, bold[i], "b", "xy", start, 34)
	}
	assertFormattingNode(t, bold[3], "b", "y", 21, 25)
	for i, start := range []int{3, 6, 9} {
		assertFormattingNode(t, bold[i+4], "b", "z", start, 39)
	}
	if bold[4].Parent != parsedBody(document) || bold[5].Parent != bold[4] || bold[6].Parent != bold[5] {
		t.Fatalf("Expected exactly the three pre-marker b entries around z, got %#v", bold[4:])
	}

	for _, testCase := range []struct {
		name, content                string
		tableEnd, itemStart, itemEnd int
	}{
		{name: "cell", content: `<b><table><tr><td><i>x<td>y</table>z`, tableEnd: 35, itemStart: 18, itemEnd: 22},
		{name: "caption", content: `<b><table><caption><i>x<tr><td>y</table>z`, tableEnd: 40, itemStart: 19, itemEnd: 23},
	} {
		document, err = NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatalf("Parse failed for %s marker loop: %v", testCase.name, err)
		}
		bold = formattingElements(document, "b")
		italics := formattingElements(document, "i")
		if len(bold) != 1 || len(italics) != 1 {
			t.Fatalf("Expected outer b to survive and inner i to be cleared by %s marker, b=%#v i=%#v", testCase.name, bold, italics)
		}
		assertFormattingNode(t, bold[0], "b", "xyz", 0, len(testCase.content))
		assertFormattingNode(t, italics[0], "i", "x", testCase.itemStart, testCase.itemEnd)
		table := formattingElements(document, "table")[0]
		assertFormattingNode(t, table, "table", "xy", 3, testCase.tableEnd)
		if table.Parent != bold[0] || bold[0].Children[len(bold[0].Children)-1].Value != "z" {
			t.Fatalf("Expected table then trailing z inside the surviving outer b for %s, got %#v", testCase.name, bold[0].Children)
		}
	}
}

func TestParseFormattingCurrentOpenAndRecoveredBRTriggers(t *testing.T) {
	const current = `<p><b>x</p>y<div>z</div>w`
	document, err := NewHTMLParser().Parse(current)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	div := formattingElements(document, "div")[0]
	if len(bold) != 2 {
		t.Fatalf("Expected one reconstructed b to remain current across div insertion, got %#v", bold)
	}
	assertFormattingNode(t, bold[1], "b", "yzw", 3, 25)
	assertFormattingNode(t, div, "div", "z", 12, 24)
	if div.Parent != bold[1] || len(bold[1].Children) != 3 || bold[1].Children[0].Value != "y" || bold[1].Children[2].Value != "w" {
		t.Fatalf("A current reconstructed b must wrap y, div, and w without another reconstruction, got %#v", bold[1].Children)
	}

	const currentMatrix = `<p><b>x</p>y<!--c--><div>z</div><style>s</style>w`
	document, err = NewHTMLParser().Parse(currentMatrix)
	if err != nil {
		t.Fatal(err)
	}
	bold = formattingElements(document, "b")
	if len(bold) != 2 {
		t.Fatalf("Comment, special, and raw-text tokens must not duplicate a current reconstructed b, got %#v", bold)
	}
	assertFormattingNode(t, bold[1], "b", "yzsw", 3, 49)
	assertFormattingNode(t, formattingElements(bold[1], "div")[0], "div", "z", 20, 32)
	assertFormattingNode(t, formattingElements(bold[1], "style")[0], "style", "s", 32, 48)
	if len(bold[1].Children) != 5 || bold[1].Children[1].Type != types.CommentNode || bold[1].Children[1].Value != "c" || bold[1].Children[1].StartPos != 12 || bold[1].Children[1].EndPos != 20 {
		t.Fatalf("Expected y/comment/div/style/w under the single current b, got %#v", bold[1].Children)
	}

	for _, testCase := range []struct {
		name, content, child string
		childStart, childEnd int
	}{
		{name: "button", content: `<p><b>x</p><button>y</button>z`, child: "button", childStart: 11, childEnd: 29},
		{name: "recovered br end", content: `<p><b>x</p></br>y`, child: "br", childStart: 11, childEnd: 16},
	} {
		document, err = NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatalf("Parse failed for %s: %v", testCase.name, err)
		}
		bold = formattingElements(document, "b")
		if len(bold) != 2 {
			t.Fatalf("Expected %s to trigger b reconstruction, got %#v", testCase.name, bold)
		}
		assertFormattingNode(t, bold[1], "b", map[string]string{"button": "yz", "recovered br end": "y"}[testCase.name], 3, len(testCase.content))
		child := formattingElements(bold[1], testCase.child)[0]
		assertFormattingNode(t, child, testCase.child, map[string]string{"button": "y", "recovered br end": ""}[testCase.name], testCase.childStart, testCase.childEnd)
	}
}

func TestParseFormattingDiscardedIncompleteStartsDoNotReconstruct(t *testing.T) {
	for _, content := range []string{`<p><b>x</p><span`, `<p><b>x</p><br`} {
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Incomplete start must be discarded without error for %q: %v", content, err)
		}
		bold := formattingElements(document, "b")
		if len(bold) != 1 {
			t.Fatalf("Discarded incomplete start must not reconstruct b for %q, got %#v", content, bold)
		}
		assertFormattingNode(t, bold[0], "b", "x", 3, 7)
		if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" {
			t.Fatalf("Expected only the explicitly closed p for %q, got %#v", content, parsedBodyChildren(document))
		}
	}
}

func TestParseFormattingDocumentEndReentryKeepsCloneInBody(t *testing.T) {
	for _, testCase := range []struct {
		content string
		bEnd    int
	}{
		{content: `<html><body><p><b>x</p></body>y</html>`, bEnd: 31},
		{content: `<html><body><p><b>x</p></body></html>y`, bEnd: 38},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatal(err)
		}
		body := formattingElements(document, "body")[0]
		bold := formattingElements(document, "b")
		if len(bold) != 2 {
			t.Fatalf("Expected reentered y to reconstruct b, got %#v", bold)
		}
		assertFormattingNode(t, bold[0], "b", "x", 15, 19)
		assertFormattingNode(t, bold[1], "b", "y", 15, testCase.bEnd)
		if bold[1].Parent != body || body.TextContent != "xy" {
			t.Fatalf("Document-end reentry must put reconstructed b(y) back in body, body=%#v b=%#v", body, bold[1])
		}
	}
}

func TestParseFormattingDocumentEndCommentsKeepAfterBodyPlacement(t *testing.T) {
	for _, testCase := range []struct {
		name, content string
		commentStart  int
		commentParent string
		htmlEnd       int
	}{
		{
			name:          "before html end",
			content:       `<html><body><p><b>x</p></body><!--c--></html>`,
			commentStart:  30,
			commentParent: "html",
			htmlEnd:       45,
		},
		{
			name:          "after html end",
			content:       `<html><body><p><b>x</p></body></html><!--c-->`,
			commentStart:  37,
			commentParent: "#document",
			htmlEnd:       37,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatal(err)
			}
			html := formattingElements(document, "html")[0]
			body := formattingElements(html, "body")[0]
			bold := formattingElements(body, "b")
			assertFormattingNode(t, html, "html", "x", 0, testCase.htmlEnd)
			assertFormattingNode(t, body, "body", "x", 6, 30)
			if body.ContentStart != 12 || body.ContentEnd != 23 || len(bold) != 1 {
				t.Fatalf("Comment after body must not stretch or reconstruct formatting in body, body=%#v b=%#v", body, bold)
			}
			comments := findAllNodesByType(document, types.CommentNode)
			if len(comments) != 1 || comments[0].Value != "c" || comments[0].StartPos != testCase.commentStart || comments[0].EndPos != testCase.commentStart+8 || comments[0].Parent == nil || comments[0].Parent.Name != testCase.commentParent {
				t.Fatalf("Expected one comment c at %d:%d under %s, got %#v", testCase.commentStart, testCase.commentStart+8, testCase.commentParent, comments)
			}
			comment := comments[0]
			if testCase.commentParent == "html" {
				if len(html.Children) != 3 || html.Children[0].Name != "head" || html.Children[0].StartPos != 0 || html.Children[0].EndPos != 0 || html.Children[1] != body || html.Children[2] != comment || len(parsedBodyChildren(document)) != 1 {
					t.Fatalf("Expected synthetic head, body, then comment under html, got document=%#v html=%#v", parsedBodyChildren(document), html.Children)
				}
			} else if len(document.Children) != 2 || document.Children[0] != html || document.Children[1] != comment || len(html.Children) != 2 || html.Children[0].Name != "head" || html.Children[0].StartPos != 0 || html.Children[0].EndPos != 0 || html.Children[1] != body {
				t.Fatalf("Expected html then comment under document, got document=%#v html=%#v", document.Children, html.Children)
			}
		})
	}
}

func TestParseFormattingReconstructsOffStackEntryInsideOpenAncestor(t *testing.T) {
	const ordinary = `<b><p><i>x</p>y</b>`
	document, err := NewHTMLParser().Parse(ordinary)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")[0]
	paragraph := formattingElements(bold, "p")[0]
	italics := formattingElements(bold, "i")
	assertFormattingNode(t, bold, "b", "xy", 0, 19)
	assertFormattingNode(t, paragraph, "p", "x", 3, 14)
	if len(italics) != 2 {
		t.Fatalf("Expected off-stack i entry to reconstruct under still-open b, got %#v", italics)
	}
	assertFormattingNode(t, italics[0], "i", "x", 6, 10)
	assertFormattingNode(t, italics[1], "i", "y", 6, 15)
	if italics[0].Parent != paragraph || italics[1].Parent != bold {
		t.Fatalf("Expected original i in p and reconstructed i in b, got %#v", italics)
	}

	const cell = `<table><tr><td><b><p><i>x</p>y</table>`
	document, err = NewHTMLParser().Parse(cell)
	if err != nil {
		t.Fatal(err)
	}
	table := formattingElements(document, "table")[0]
	td := formattingElements(table, "td")[0]
	bold = formattingElements(td, "b")[0]
	paragraph = formattingElements(bold, "p")[0]
	italics = formattingElements(bold, "i")
	assertFormattingNode(t, table, "table", "xy", 0, 38)
	assertFormattingNode(t, td, "td", "xy", 11, 30)
	assertFormattingNode(t, bold, "b", "xy", 15, 30)
	assertFormattingNode(t, paragraph, "p", "x", 18, 29)
	if len(italics) != 2 {
		t.Fatalf("Expected i reconstruction within the same cell marker, got %#v", italics)
	}
	assertFormattingNode(t, italics[0], "i", "x", 21, 25)
	assertFormattingNode(t, italics[1], "i", "y", 21, 30)
	if italics[1].Parent != bold {
		t.Fatalf("Expected reconstructed i(y) under still-open b in cell, got %#v", italics[1])
	}
}

func TestParseFormattingOffStackCommentsDoNotReconstructOrLeakState(t *testing.T) {
	const content = `<p><b id=0><b id=1><b id=2><b id=3>x<div><!--c--><!--c--><!--c--><!--c--></div>`
	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	div := formattingElements(document, "div")[0]
	bold := formattingElements(document, "b")
	assertFormattingNode(t, paragraph, "p", "x", 0, 36)
	assertFormattingNode(t, div, "div", "", 36, 79)
	if len(bold) != 4 {
		t.Fatalf("Comment-only div must not reconstruct any of four off-stack b entries, got %#v", bold)
	}
	for i, start := range []int{3, 11, 19, 27} {
		assertFormattingNode(t, bold[i], "b", "x", start, 36)
		if bold[i].Attributes["id"] != fmt.Sprint(i) {
			t.Fatalf("Expected distinct b id=%d, got %#v", i, bold[i].Attributes)
		}
	}
	if len(div.Children) != 4 {
		t.Fatalf("Expected four comments and no reconstructed element in div, got %#v", div.Children)
	}
	for i, comment := range div.Children {
		wantStart := 41 + 8*i
		if comment.Type != types.CommentNode || comment.Value != "c" || comment.StartPos != wantStart || comment.EndPos != wantStart+8 {
			t.Fatalf("Expected comment %d at %d:%d, got %#v", i, wantStart, wantStart+8, comment)
		}
	}
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0] != paragraph || parsedBodyChildren(document)[1] != div {
		t.Fatalf("Expected sibling p and comment-only div, got %#v", parsedBodyChildren(document))
	}

	plain, err := parser.Parse(`z`)
	if err != nil || len(parsedBodyChildren(plain)) != 1 || parsedBodyChildren(plain)[0].Type != types.TextNode || parsedBodyChildren(plain)[0].Value != "z" || len(formattingElements(plain, "b")) != 0 {
		t.Fatalf("Parser reuse leaked off-stack active formatting entries: doc=%#v err=%v", plain, err)
	}
}

func TestParseFormattingOffStackCommentScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n*20 + 20)
		b.WriteString(`<p>`)
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, `<b id=%d>`, i)
		}
		b.WriteString(`x<div>`)
		for i := 0; i < n; i++ {
			b.WriteString(`<!--c-->`)
		}
		b.WriteString(`</div>`)
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
	small, large := measure(2000), measure(8000)
	ratio := float64(large) / float64(small)
	t.Logf("off-stack comment scaling 2000=%v 8000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 30*time.Millisecond {
		t.Fatalf("Off-stack nontrigger handling scaled superlinearly: 2000=%v 8000=%v ratio=%.1fx", small, large, ratio)
	}
}

func TestParseFormattingReconstructedCloneTextAndComments(t *testing.T) {
	const content = `<p><b id=0><b id=1><b id=2><b id=3>x<div>y<!--c-->y<!--c-->y<!--c-->y<!--c--></div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := formattingElements(document, "p")[0]
	div := formattingElements(document, "div")[0]
	bold := formattingElements(document, "b")
	assertFormattingNode(t, paragraph, "p", "x", 0, 36)
	assertFormattingNode(t, div, "div", "yyyy", 36, 83)
	if len(bold) != 8 {
		t.Fatalf("Expected four original and four reconstructed b nodes, got %#v", bold)
	}
	for i, start := range []int{3, 11, 19, 27} {
		assertFormattingNode(t, bold[i], "b", "x", start, 36)
		assertFormattingNode(t, bold[i+4], "b", "yyyy", start, 77)
	}
	if bold[4].Parent != div || bold[5].Parent != bold[4] || bold[6].Parent != bold[5] || bold[7].Parent != bold[6] {
		t.Fatalf("Expected reconstructed b chain directly in div, got %#v", bold[4:])
	}
	deepest := bold[7]
	if len(deepest.Children) != 8 {
		t.Fatalf("Expected alternating y/comment tokens in deepest reconstructed b, got %#v", deepest.Children)
	}
	for i := 0; i < 4; i++ {
		text := deepest.Children[2*i]
		comment := deepest.Children[2*i+1]
		wantTextStart := 41 + 9*i
		wantCommentStart := 42 + 9*i
		if text.Type != types.TextNode || text.Value != "y" || text.StartPos != wantTextStart || text.EndPos != wantTextStart+1 {
			t.Fatalf("Expected y token %d at %d:%d, got %#v", i, wantTextStart, wantTextStart+1, text)
		}
		if comment.Type != types.CommentNode || comment.Value != "c" || comment.StartPos != wantCommentStart || comment.EndPos != wantCommentStart+8 {
			t.Fatalf("Expected comment %d at %d:%d, got %#v", i, wantCommentStart, wantCommentStart+8, comment)
		}
	}
}

func TestParseFormattingReconstructedCloneTextScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n*22 + 20)
		b.WriteString(`<p>`)
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, `<b id=%d>`, i)
		}
		b.WriteString(`x<div>`)
		for i := 0; i < n; i++ {
			b.WriteString(`y<!--c-->`)
		}
		b.WriteString(`</div>`)
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
	small, large := measure(250), measure(1000)
	ratio := float64(large) / float64(small)
	t.Logf("reconstructed clone text scaling 250=%v 1000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 10*time.Millisecond {
		t.Fatalf("Reconstructed-clone TextContent updates scaled superlinearly: 250=%v 1000=%v ratio=%.1fx", small, large, ratio)
	}
}

func TestParseFormattingReconstructionTriggerMatrix(t *testing.T) {
	for _, testCase := range []struct {
		name, content, trigger         string
		bEnd, triggerStart, triggerEnd int
	}{
		{name: "whitespace character", content: `<p><b>x</p> y`, bEnd: 13},
		{name: "generic start", content: `<p><b>x</p><span>y</span>`, trigger: "span", bEnd: 25, triggerStart: 11, triggerEnd: 25},
		{name: "void img start", content: `<p><b>x</p><img>y`, trigger: "img", bEnd: 17, triggerStart: 11, triggerEnd: 16},
		{name: "xmp start", content: `<p><b>x</p><xmp>y</xmp>`, trigger: "xmp", bEnd: 23, triggerStart: 11, triggerEnd: 23},
	} {
		document, err := NewHTMLParser().Parse(testCase.content)
		if err != nil {
			t.Fatalf("Parse failed for %s: %v", testCase.name, err)
		}
		bold := formattingElements(document, "b")
		if len(bold) != 2 {
			t.Fatalf("Expected reconstruction for %s, got %#v", testCase.name, bold)
		}
		assertFormattingNode(t, bold[1], "b", map[string]string{"whitespace character": " y", "generic start": "y", "void img start": "y", "xmp start": "y"}[testCase.name], 3, testCase.bEnd)
		if testCase.trigger != "" {
			trigger := formattingElements(bold[1], testCase.trigger)[0]
			assertFormattingNode(t, trigger, testCase.trigger, map[string]string{"img": "", "span": "y", "xmp": "y"}[testCase.trigger], testCase.triggerStart, testCase.triggerEnd)
		}
	}

	const comment = `<p><b>x</p><!--c-->y`
	document, err := NewHTMLParser().Parse(comment)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	assertFormattingNode(t, bold[1], "b", "y", 3, 20)
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[1].Type != types.CommentNode || parsedBodyChildren(document)[1].Value != "c" || parsedBodyChildren(document)[1].StartPos != 11 || parsedBodyChildren(document)[1].EndPos != 19 || parsedBodyChildren(document)[2] != bold[1] {
		t.Fatalf("Expected comment before reconstructed b because comments do not trigger reconstruction, got %#v", parsedBodyChildren(document))
	}

	const block = `<p><b>x</p><div>y</div>`
	document, err = NewHTMLParser().Parse(block)
	if err != nil {
		t.Fatal(err)
	}
	div := formattingElements(document, "div")[0]
	bold = formattingElements(document, "b")
	assertFormattingNode(t, div, "div", "y", 11, 23)
	assertFormattingNode(t, bold[1], "b", "y", 3, 17)
	if bold[1].Parent != div {
		t.Fatalf("Block start itself must not reconstruct b; first character must reconstruct inside div, got %#v", bold[1])
	}

	const plaintext = `<p><b>x</p><plaintext>y`
	document, err = NewHTMLParser().Parse(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	plain := formattingElements(document, "plaintext")[0]
	bold = formattingElements(document, "b")
	assertFormattingNode(t, plain, "plaintext", "y", 11, 23)
	assertFormattingNode(t, bold[1], "b", "y", 3, 23)
	if bold[1].Parent != plain {
		t.Fatalf("Plaintext start must not reconstruct; first plaintext character must, got %#v", parsedBodyChildren(plain))
	}
}

func TestParseFormattingCommentAndDoctypeDoNotTriggerReconstruction(t *testing.T) {
	const comment = `<p><b>x</p><!--c-->y`
	document, err := NewHTMLParser().Parse(comment)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	assertFormattingNode(t, bold[1], "b", "y", 3, 20)
	if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[1].Type != types.CommentNode || parsedBodyChildren(document)[1].Value != "c" || parsedBodyChildren(document)[1].StartPos != 11 || parsedBodyChildren(document)[1].EndPos != 19 || parsedBodyChildren(document)[2] != bold[1] {
		t.Fatalf("Expected comment before reconstructed b because comments do not trigger reconstruction, got %#v", parsedBodyChildren(document))
	}

	const doctype = `<p><b>x</p><!DOCTYPE html>y`
	document, err = NewHTMLParser().Parse(doctype)
	if err != nil {
		t.Fatal(err)
	}
	bold = formattingElements(document, "b")
	assertFormattingNode(t, bold[1], "b", "y", 3, 27)
	if len(formattingElements(document, "html")) != 1 || len(findAllNodesByType(document, types.DocumentTypeNode)) != 0 || bold[1].Children[0].StartPos != 26 || bold[1].Children[0].EndPos != 27 {
		t.Fatalf("Expected ignored doctype not to trigger reconstruction before y, got %#v", parsedBodyChildren(document))
	}
}

func TestParseFormattingReconstructionUsesEmittedCharacterTokens(t *testing.T) {
	for _, testCase := range []struct {
		name, suffix string
		wantCount    int
		wantText     string
	}{
		{name: "literal NUL ignored", suffix: "\x00", wantCount: 1},
		{name: "comment does not trigger", suffix: `<!--c-->`, wantCount: 1},
		{name: "decoded NUL replacement triggers", suffix: `&#0;`, wantCount: 2, wantText: "�"},
		{name: "decoded whitespace triggers", suffix: `&#32;`, wantCount: 2, wantText: " "},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			content := `<p><b>x<div>` + testCase.suffix + `</div>`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			bold := formattingElements(document, "b")
			if len(bold) != testCase.wantCount {
				t.Fatalf("Expected %d b nodes for emitted-token case, got %#v", testCase.wantCount, bold)
			}
			if testCase.wantCount == 2 {
				div := formattingElements(document, "div")[0]
				if bold[1].Parent != div || bold[1].TextContent != testCase.wantText {
					t.Fatalf("Expected emitted character token %q to reconstruct b inside div, got %#v", testCase.wantText, bold[1])
				}
			}
		})
	}
}

func TestParseFormattingCloneMetadataIsIndependent(t *testing.T) {
	const content = `<p><b class=x>one<div>two</div>three`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(document, "b")
	if len(bold) != 3 {
		t.Fatalf("Expected three b nodes for clone independence, got %#v", bold)
	}
	bold[1].Attributes["class"] = "mutated"
	bold[1].Attributes["new"] = "value"
	bold[1].AttributeOrder[0] = "new"
	if bold[0].Attributes["class"] != "x" || bold[2].Attributes["class"] != "x" {
		t.Fatalf("Mutating one clone's Attributes changed original/sibling: %#v", bold)
	}
	if _, exists := bold[0].Attributes["new"]; exists {
		t.Fatalf("Mutating one clone added attribute to original: %#v", bold[0].Attributes)
	}
	if bold[0].AttributeOrder[0] != "class" || bold[2].AttributeOrder[0] != "class" {
		t.Fatalf("Mutating one clone's AttributeOrder changed original/sibling: %#v", bold)
	}
}

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
