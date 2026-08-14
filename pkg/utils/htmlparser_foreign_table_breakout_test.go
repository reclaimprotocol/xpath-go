package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 19 covers a table start token breaking out of ordinary SVG/MathML in
// whole-document body, cell, caption, and current customizable-select
// contexts. Templates, fragments, foreign integration reached from legacy
// select modes, hidden/form-pointer quirks, and foreign processing
// instructions remain deferred.

func requireForeignTableNode(t *testing.T, node *types.Node, name, namespace, text string, start, end int) {
	t.Helper()
	if node == nil || node.Name != name || node.NamespaceURI != namespace || node.TextContent != text || node.StartPos != start || node.EndPos != end {
		t.Fatalf("foreign-table node mismatch: got %#v, want <%s> ns=%q text=%q range=%d:%d", node, name, namespace, text, start, end)
	}
}

func requireHTMLTableShape(t *testing.T, table *types.Node, parent *types.Node, text string, start, end int) {
	t.Helper()
	requireForeignTableNode(t, table, "table", htmlNamespaceURI, text, start, end)
	if table.Parent != parent || len(table.Children) != 1 || table.Children[0].Name != "tbody" || table.Children[0].NamespaceURI != htmlNamespaceURI || len(table.Children[0].Children) != 1 || table.Children[0].Children[0].Name != "tr" || len(table.Children[0].Children[0].Children) != 1 || table.Children[0].Children[0].Children[0].Name != "td" || table.Children[0].Children[0].Children[0].TextContent != text {
		t.Fatalf("reprocessed HTML table shape/parent mismatch: %#v", table)
	}
}

func TestParseForeignTableBreakoutInBody(t *testing.T) {
	for _, test := range []struct {
		name, content, rootID, innerID, rootName, innerName, namespace string
		rootStart, innerStart, trigger, tableEnd, tailEnd              int
	}{
		{"svg", `<div id=o><svg id=s><g id=g>a<table id=t><tr><td>b</table>c</g></svg>d</div>`, "s", "g", "svg", "g", svgNamespaceURI, 10, 20, 29, 58, 70},
		{"math", `<div id=o><math id=m><mrow id=r>a<table id=t><tr><td>b</table>c</mrow></math>d</div>`, "m", "r", "math", "mrow", mathMLTestNamespaceURI, 10, 21, 33, 62, 78},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			div := svgNodeByID(t, document, "o")
			foreign := svgNodeByID(t, document, test.rootID)
			inner := svgNodeByID(t, document, test.innerID)
			table := svgNodeByID(t, document, "t")
			requireForeignTableNode(t, foreign, test.rootName, test.namespace, "a", test.rootStart, test.trigger)
			requireForeignTableNode(t, inner, test.innerName, test.namespace, "a", test.innerStart, test.trigger)
			requireHTMLTableShape(t, table, div, "b", test.trigger, test.tableEnd)
			if len(div.Children) != 3 || div.Children[0] != foreign || div.Children[1] != table || div.Children[2].Value != "cd" || div.Children[2].StartPos != test.tableEnd || div.Children[2].EndPos != test.tailEnd || div.TextContent != "abcd" {
				t.Fatalf("foreign table body breakout order/ranges mismatch: %#v", div.Children)
			}
		})
	}
}

func TestParseForeignTableBreakoutInCell(t *testing.T) {
	for _, test := range []struct {
		name, content, rootID, innerID, rootName, innerName, namespace string
		cellEnd, rootStart, innerStart, trigger, tableEnd              int
	}{
		{"svg", `<table id=o><tr><td id=d><svg id=s><g id=g>a<table id=t><tr><td>b</table>c</g></svg>d</table>`, "s", "g", "svg", "g", svgNamespaceURI, 85, 25, 35, 44, 73},
		{"math", `<table id=o><tr><td id=d><math id=m><mrow id=r>a<table id=t><tr><td>b</table>c</mrow></math>d</table>`, "m", "r", "math", "mrow", mathMLTestNamespaceURI, 93, 25, 36, 48, 77},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			cell := svgNodeByID(t, document, "d")
			foreign := svgNodeByID(t, document, test.rootID)
			inner := svgNodeByID(t, document, test.innerID)
			table := svgNodeByID(t, document, "t")
			requireForeignTableNode(t, foreign, test.rootName, test.namespace, "a", test.rootStart, test.trigger)
			requireForeignTableNode(t, inner, test.innerName, test.namespace, "a", test.innerStart, test.trigger)
			requireHTMLTableShape(t, table, cell, "b", test.trigger, test.tableEnd)
			if cell.StartPos != 16 || cell.EndPos != test.cellEnd || cell.TextContent != "abcd" || len(cell.Children) != 3 || cell.Children[0] != foreign || cell.Children[1] != table || cell.Children[2].Value != "cd" || cell.Children[2].StartPos != test.tableEnd || cell.Children[2].EndPos != test.cellEnd {
				t.Fatalf("foreign table cell breakout order/ranges mismatch: %#v", cell)
			}
		})
	}
}

func TestParseForeignTableBreakoutInCaption(t *testing.T) {
	for _, test := range []struct {
		name, content, rootID, rootName, namespace string
		captionEnd, rootStart, trigger, tableEnd   int
	}{
		{"svg", `<table id=o><caption id=p><svg id=s><g>a<table id=t><tr><td>b</table>c</g></svg>d</caption><tr><td>e</table>`, "s", "svg", svgNamespaceURI, 91, 26, 40, 69},
		{"math", `<table id=o><caption id=p><math id=m><mrow>a<table id=t><tr><td>b</table>c</mrow></math>d</caption><tr><td>e</table>`, "m", "math", mathMLTestNamespaceURI, 99, 26, 44, 73},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			caption := svgNodeByID(t, document, "p")
			foreign := svgNodeByID(t, document, test.rootID)
			table := svgNodeByID(t, document, "t")
			requireForeignTableNode(t, foreign, test.rootName, test.namespace, "a", test.rootStart, test.trigger)
			requireHTMLTableShape(t, table, caption, "b", test.trigger, test.tableEnd)
			if caption.StartPos != 12 || caption.EndPos != test.captionEnd || caption.TextContent != "abcd" || len(caption.Children) != 3 || caption.Children[0] != foreign || caption.Children[1] != table || caption.Children[2].Value != "cd" || caption.Children[2].StartPos != test.tableEnd || caption.Children[2].EndPos != test.captionEnd-10 {
				t.Fatalf("foreign table caption breakout order/ranges mismatch: %#v", caption)
			}
		})
	}
}

func TestParseForeignTableBreakoutInCustomizableSelect(t *testing.T) {
	// Chrome 151/current WHATWG is authoritative. Bundled jsdom still applies
	// the legacy in-select tree builder and drops these retained foreign nodes.
	for _, test := range []struct {
		name, content, rootID, rootName, namespace string
		rootStart, innerStart, trigger, tableEnd   int
	}{
		{"svg", `<select id=q><svg id=s><g>a<table id=t><tr><td>b</table>c</g></svg>d</select>`, "s", "svg", svgNamespaceURI, 13, 23, 27, 56},
		{"math", `<select id=q><math id=m><mrow>a<table id=t><tr><td>b</table>c</mrow></math>d</select>`, "m", "math", mathMLTestNamespaceURI, 13, 24, 31, 60},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			selectNode := svgNodeByID(t, document, "q")
			foreign := svgNodeByID(t, document, test.rootID)
			table := svgNodeByID(t, document, "t")
			requireForeignTableNode(t, foreign, test.rootName, test.namespace, "a", test.rootStart, test.trigger)
			requireHTMLTableShape(t, table, selectNode, "b", test.trigger, test.tableEnd)
			if selectNode.TextContent != "abcd" || len(selectNode.Children) != 3 || selectNode.Children[0] != foreign || selectNode.Children[1] != table || selectNode.Children[2].Value != "cd" || selectNode.Children[2].StartPos != test.tableEnd || selectNode.Children[2].EndPos != len(test.content)-len(`</select>`) {
				t.Fatalf("customizable-select foreign table breakout mismatch: %#v", selectNode)
			}
		})
	}
}

func TestParseForeignTablePlacementGuards(t *testing.T) {
	const direct = `<div id=o><table id=t><svg id=s><circle /></svg><tr><td>x</table>z</div>`
	document, err := NewHTMLParser().Parse(direct)
	if err != nil {
		t.Fatal(err)
	}
	div := svgNodeByID(t, document, "o")
	table := svgNodeByID(t, document, "t")
	svg := svgNodeByID(t, document, "s")
	if len(div.Children) != 3 || div.Children[0] != svg || div.Children[1] != table || div.Children[2].Value != "z" || svg.Parent != div || svg.NamespaceURI != svgNamespaceURI || svg.StartPos != 22 || svg.EndPos != 48 || table.StartPos != 10 || table.EndPos != 65 || table.TextContent != "x" {
		t.Fatalf("direct in-table SVG fostering guard mismatch: %#v", div.Children)
	}

	const foreignObject = `<svg id=s><foreignObject id=f><table id=t><tr><td>x</table></foreignObject><circle /></svg>`
	document, err = NewHTMLParser().Parse(foreignObject)
	if err != nil {
		t.Fatal(err)
	}
	fo := svgNodeByID(t, document, "f")
	table = svgNodeByID(t, document, "t")
	if fo.NamespaceURI != svgNamespaceURI || table.NamespaceURI != htmlNamespaceURI || table.Parent != fo || table.StartPos != 30 || table.EndPos != 59 || len(table.Children) != 1 || table.Children[0].Name != "tbody" {
		t.Fatalf("foreignObject HTML table guard mismatch: fo=%#v table=%#v", fo, table)
	}

	for _, content := range []string{
		`<svg id=s><g><select id=q><option>x</option></select></g></svg>`,
		`<math id=m><mrow><select id=q><option>x</option></select></mrow></math>`,
	} {
		document, err = NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		selectNode := svgNodeByID(t, document, "q")
		owner := selectNode.Parent
		if selectNode.NamespaceURI != owner.NamespaceURI || selectNode.NamespaceURI == htmlNamespaceURI {
			t.Fatalf("generic select spelling incorrectly broke foreign content: %#v", selectNode)
		}
	}
}

func TestParseForeignTableBreakoutClosesStandardsParagraph(t *testing.T) {
	const paragraph = `<!doctype html><p id=p>a<svg id=s><g>b<table id=t><tr><td>x</table>c</g></svg>d`
	document, err := NewHTMLParser().Parse(paragraph)
	if err != nil {
		t.Fatal(err)
	}
	p := svgNodeByID(t, document, "p")
	svg := svgNodeByID(t, document, "s")
	table := svgNodeByID(t, document, "t")
	requireForeignTableNode(t, svg, "svg", svgNamespaceURI, "b", 24, 38)
	if p.EndPos != 38 || p.TextContent != "ab" || svg.Parent != p || table.Parent == p || table.NamespaceURI != htmlNamespaceURI || table.TextContent != "x" || table.StartPos != 38 || table.EndPos != 67 || table.Parent == nil || len(table.Parent.Children) != 3 || table.Parent.Children[0] != p || table.Parent.Children[1] != table || table.Parent.Children[2].Value != "cd" {
		t.Fatalf("foreign table reprocess did not apply in-body paragraph rule: p=%#v table=%#v", p, table)
	}
}

func TestParseForeignTableBreakoutParagraphDoctypeModes(t *testing.T) {
	const core = `<p id=p>a<svg id=s><g>b<table id=t><tr><td>x</table>c</g></svg>d`
	for _, test := range []struct {
		name, doctype string
		closesP       bool
		pStart        int
		trigger       int
		tableEnd      int
	}{
		{"no-doctype-quirks", "", false, 0, 23, 52},
		{"canonical-standards", `<!doctype html>`, true, 15, 38, 67},
		{"tab-standards", "<!DOCTYPE\thtml>", true, 15, 38, 67},
		{"xhtml-limited-quirks", `<!DOCTYPE HTML PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">`, true, 121, 144, 173},
		{"html4-limited-quirks", `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">`, true, 102, 125, 154},
		{"legacy-force-quirks", `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN">`, false, 63, 86, 115},
		{"missing-name-force-quirks", `<!DOCTYPE>`, false, 10, 33, 62},
	} {
		t.Run(test.name, func(t *testing.T) {
			content := test.doctype + core
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			p := svgNodeByID(t, document, "p")
			table := svgNodeByID(t, document, "t")
			if p.StartPos != test.pStart || table.StartPos != test.trigger || table.EndPos != test.tableEnd || table.NamespaceURI != htmlNamespaceURI {
				t.Fatalf("doctype mode source ranges mismatch: p=%#v table=%#v", p, table)
			}
			if test.closesP {
				if p.EndPos != test.trigger || p.TextContent != "ab" || table.Parent == p || table.Parent != p.Parent || table.Parent == nil || table.Parent.Children[len(table.Parent.Children)-1].Value != "cd" {
					t.Fatalf("standards/limited-quirks table did not close p: p=%#v table=%#v", p, table)
				}
			} else if p.EndPos != len(content) || p.TextContent != "abxcd" || table.Parent != p || p.Children[len(p.Children)-1].Value != "cd" {
				t.Fatalf("full-quirks table incorrectly closed p: p=%#v table=%#v", p, table)
			}
		})
	}

	parser := NewHTMLParser()
	for _, content := range []string{`<!doctype html>` + core, core, core, `<!doctype html>` + core} {
		document, err := parser.Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		p := svgNodeByID(t, document, "p")
		standards := strings.HasPrefix(content, `<!doctype html>`)
		if (p.TextContent == "ab") != standards {
			t.Fatalf("doctype/quirks mode leaked across parser reuse: standards=%v p=%#v", standards, p)
		}
	}
}

func TestParseForeignTableBreakoutAfterBodyModeContrast(t *testing.T) {
	const core = `<html><body><p id=p>a<svg id=s><g>b<table id=t><tr><td>x</table>c</g></svg>d</body><table id=u><tr><td>e</table></html>`
	for _, test := range []struct {
		name, prefix string
		pStart, pEnd int
		pText        string
		tableParentP bool
	}{
		{"quirks", "", 12, 119, "abxcde", true},
		{"standards", `<!doctype html>`, 27, 50, "ab", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.prefix + core)
			if err != nil {
				t.Fatal(err)
			}
			p := svgNodeByID(t, document, "p")
			first := svgNodeByID(t, document, "t")
			afterBody := svgNodeByID(t, document, "u")
			if p.StartPos != test.pStart || p.EndPos != test.pEnd || p.TextContent != test.pText || (first.Parent == p) != test.tableParentP || (afterBody.Parent == p) != test.tableParentP {
				t.Fatalf("after-body table/paragraph mode contrast mismatch: p=%#v first=%#v after=%#v", p, first, afterBody)
			}
		})
	}
}

func TestParseForeignTableBreakoutStopsAtSVGIntegrationOwner(t *testing.T) {
	for _, test := range []struct {
		owner                                      string
		outerEnd, ownerEnd                         int
		innerStart, trigger, tableEnd, circleStart int
	}{
		{"foreignObject", 116, 95, 30, 49, 78, 95},
		{"desc", 98, 77, 21, 40, 69, 77},
		{"title", 100, 79, 22, 41, 70, 79},
	} {
		t.Run(test.owner, func(t *testing.T) {
			content := `<svg id=o><` + test.owner + ` id=f><svg id=i><g id=g>a<table id=t><tr><td>b</table>c</` + test.owner + `><circle id=q /></svg>`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			outer := svgNodeByID(t, document, "o")
			owner := svgNodeByID(t, document, "f")
			inner := svgNodeByID(t, document, "i")
			g := svgNodeByID(t, document, "g")
			table := svgNodeByID(t, document, "t")
			circle := svgNodeByID(t, document, "q")
			if outer.NamespaceURI != svgNamespaceURI || outer.EndPos != test.outerEnd || owner.NamespaceURI != svgNamespaceURI || owner.Parent != outer || owner.EndPos != test.ownerEnd || owner.TextContent != "abc" || inner.Parent != owner || inner.StartPos != test.innerStart || inner.EndPos != test.trigger || inner.TextContent != "a" || g.EndPos != test.trigger || table.Parent != owner || table.NamespaceURI != htmlNamespaceURI || table.StartPos != test.trigger || table.EndPos != test.tableEnd || owner.Children[len(owner.Children)-1].Value != "c" || circle.Parent != outer || circle.NamespaceURI != svgNamespaceURI || circle.StartPos != test.circleStart {
				t.Fatalf("SVG integration-owner table breakout/restoration mismatch: outer=%#v owner=%#v inner=%#v table=%#v circle=%#v", outer, owner, inner, table, circle)
			}
		})
	}

}

func TestParseForeignTableBreakoutConsumesExactStaleForeignEnds(t *testing.T) {
	const content = `<svg id=o><foreignObject id=f><svg id=i><g id=g>a<table id=t><tr><td>b</table>b</g></svg>c</foreignObject><circle id=q /></svg>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	outer := svgNodeByID(t, document, "o")
	owner := svgNodeByID(t, document, "f")
	inner := svgNodeByID(t, document, "i")
	g := svgNodeByID(t, document, "g")
	table := svgNodeByID(t, document, "t")
	circle := svgNodeByID(t, document, "q")
	if outer.NamespaceURI != svgNamespaceURI || outer.EndPos != 89 || outer.TextContent != "abb" || owner.Parent != outer || owner.NamespaceURI != svgNamespaceURI || owner.EndPos != 83 || owner.TextContent != "abb" || inner.Parent != owner || inner.EndPos != 49 || g.EndPos != 49 || table.Parent != owner || table.NamespaceURI != htmlNamespaceURI || table.StartPos != 49 || table.EndPos != 78 || owner.Children[len(owner.Children)-1].Value != "b" || owner.Children[len(owner.Children)-1].StartPos != 78 || owner.Children[len(owner.Children)-1].EndPos != 79 || circle.NamespaceURI != htmlNamespaceURI || circle.Parent == outer {
		t.Fatalf("stale </g></svg> debt after integration-owner table breakout mismatch: outer=%#v owner=%#v inner=%#v table=%#v circle=%#v", outer, owner, inner, table, circle)
	}
	if len(parsedBodyChildren(document)) < 3 || parsedBodyChildren(document)[1].Value != "c" || parsedBodyChildren(document)[1].StartPos != 89 || parsedBodyChildren(document)[1].EndPos != 90 {
		t.Fatalf("stale inner </svg> did not close the live outer SVG at its exact token: %#v", parsedBodyChildren(document))
	}
}

func TestParseForeignTableBreakoutFormattingAndNestedBarrier(t *testing.T) {
	const formatting = `<b id=b><svg id=s><g>a<table id=t><tr><td>x</table>c</g></svg>d</b>e`
	document, err := NewHTMLParser().Parse(formatting)
	if err != nil {
		t.Fatal(err)
	}
	bold := svgNodeByID(t, document, "b")
	svg := svgNodeByID(t, document, "s")
	table := svgNodeByID(t, document, "t")
	if bold.TextContent != "axcd" || bold.StartPos != 0 || bold.EndPos != 67 || len(bold.Children) != 3 || bold.Children[0] != svg || bold.Children[1] != table || bold.Children[2].Value != "cd" || table.Parent != bold || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].Value != "e" {
		t.Fatalf("foreign table breakout corrupted active formatting: bold=%#v doc=%#v", bold, parsedBodyChildren(document))
	}

	const nested = `<div id=o><svg id=s><g><math id=m><mrow>a<table id=t><tr><td>x</table>b</mrow></math>c</g></svg>d</div>`
	document, err = NewHTMLParser().Parse(nested)
	if err != nil {
		t.Fatal(err)
	}
	div := svgNodeByID(t, document, "o")
	svg = svgNodeByID(t, document, "s")
	mathSpelling := svgNodeByID(t, document, "m")
	table = svgNodeByID(t, document, "t")
	if svg.NamespaceURI != svgNamespaceURI || mathSpelling.NamespaceURI != svgNamespaceURI || svg.EndPos != 41 || mathSpelling.EndPos != 41 || mathSpelling.TextContent != "a" || table.Parent != div || table.StartPos != 41 || table.EndPos != 70 || div.Children[len(div.Children)-1].Value != "bcd" {
		t.Fatalf("nested foreign namespace barrier/table breakout mismatch: div=%#v svg=%#v math=%#v", div, svg, mathSpelling)
	}
}

func TestParseForeignTableBreakoutTokenVariantsAndCoordinates(t *testing.T) {
	for _, start := range []string{`<TaBlE id=t>`, `<table id=t />`, `<table id=t data-x=y>`, `<table id=t data-x=y/>`} {
		content := `<svg id=s><g>a` + start + `<tr><td>x</table>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		svg := svgNodeByID(t, document, "s")
		table := svgNodeByID(t, document, "t")
		trigger := strings.Index(content, start)
		bodies := listElements(document, "body")
		if svg.EndPos != trigger || svg.TextContent != "a" || table.NamespaceURI != htmlNamespaceURI || table.StartPos != trigger || len(bodies) != 1 || table.Parent != bodies[0] || table.TextContent != "x" {
			t.Fatalf("emitted foreign table variant %q mismatch: svg=%#v table=%#v", start, svg, table)
		}
	}

	const unicode = "<div>é\r\n<svg id=s><g id=g>😀<table id=t><tr><td>x</table>z</div>"
	document, err := NewHTMLParser().Parse(unicode)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodeByID(t, document, "s")
	g := svgNodeByID(t, document, "g")
	table := svgNodeByID(t, document, "t")
	for _, node := range []*types.Node{svg, g} {
		if node.EndPos != 31 || node.EndLine != 2 || node.EndColumn != 21 {
			t.Fatalf("multibyte foreign ancestor trigger endpoint mismatch: %#v", node)
		}
	}
	if table.StartPos != 31 || table.EndPos != 60 || table.StartLine != 2 || table.StartColumn != 21 || table.EndLine != 2 || table.EndColumn != 50 {
		t.Fatalf("multibyte table raw range/coordinates mismatch: %#v", table)
	}
}

func TestParseForeignTableIntegrationPointAndEndTagGuards(t *testing.T) {
	for _, test := range []struct {
		content, ownerID string
	}{
		{`<math><mi id=i><table id=t><tr><td>x</table>y</mi><mn>z</mn></math>`, "i"},
		{`<math><mtext id=i><table id=t><tr><td>x</table>y</mtext><mn>z</mn></math>`, "i"},
		{`<math><annotation-xml id=i encoding=text/html><table id=t><tr><td>x</table>y</annotation-xml></math>`, "i"},
	} {
		document, err := NewHTMLParser().Parse(test.content)
		if err != nil {
			t.Fatal(err)
		}
		owner := svgNodeByID(t, document, test.ownerID)
		table := svgNodeByID(t, document, "t")
		if owner.NamespaceURI != mathMLTestNamespaceURI || table.NamespaceURI != htmlNamespaceURI || table.Parent != owner || owner.TextContent != "xy" {
			t.Fatalf("HTML/MathML integration point incorrectly applied foreign table pop: owner=%#v table=%#v", owner, table)
		}
	}

	const endTag = `<svg id=s><g id=g>x</table>y</g></svg>z`
	document, err := NewHTMLParser().Parse(endTag)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodeByID(t, document, "s")
	g := svgNodeByID(t, document, "g")
	if svg.EndPos != 38 || g.EndPos != 32 || g.TextContent != "xy" || len(g.Children) != 1 || g.Children[0].Value != "xy" || g.Children[0].StartPos != 18 || g.Children[0].EndPos != 28 {
		t.Fatalf("foreign </table> must stay an ignored foreign end: svg=%#v g=%#v", svg, g)
	}
}

func TestParseForeignTableIncompleteEOFReuseAndScaling(t *testing.T) {
	for _, test := range []struct {
		content, rootID, innerID string
	}{
		{`<svg id=s><g id=g>a<table`, "s", "g"},
		{`<math id=m><mrow id=r>a<table`, "m", "r"},
	} {
		document, err := NewHTMLParser().Parse(test.content)
		if err != nil {
			t.Fatal(err)
		}
		root := svgNodeByID(t, document, test.rootID)
		inner := svgNodeByID(t, document, test.innerID)
		if len(svgNodesNamed(document, "table")) != 0 || root.EndPos != len(test.content) || inner.EndPos != len(test.content) || inner.TextContent != "a" || len(inner.Children) != 1 || inner.Children[0].Value != "a" || inner.Children[0].EndPos != len(test.content) {
			t.Fatalf("incomplete foreign table start mutated state: root=%#v inner=%#v", root, inner)
		}
	}
	parser := NewHTMLParser()
	if _, err := parser.Parse(`<svg><g>a<table><tr><td>b</table>c`); err != nil {
		t.Fatal(err)
	}
	if document, err := parser.Parse(`<div>x`); err != nil || len(svgNodesNamed(document, "div")) != 1 || svgNodesNamed(document, "div")[0].EndPos != 6 {
		t.Fatalf("foreign-table to implicit-document EOF reuse mismatch: doc=%#v err=%v", document, err)
	}

	build := func(n int) string {
		return strings.Repeat(`<div><svg><g>a<table><tr><td>b</table>c</g></svg>d</div>`, n)
	}
	smallInput, largeInput := build(500), build(2000)
	_ = bestMathMLParseDuration(t, smallInput)
	small := bestMathMLParseDuration(t, smallInput)
	large := bestMathMLParseDuration(t, largeInput)
	if large > 10*small && large-small > 100*time.Millisecond {
		t.Fatalf("foreign table breakout scaled superlinearly: small=%s large=%s", small, large)
	}
}
