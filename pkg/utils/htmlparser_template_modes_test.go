package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseTemplateInsertionModePersistsAcrossSiblings(t *testing.T) {
	t.Run("caption then row stays in table mode", func(t *testing.T) {
		const content = `<template id=t><caption id=c>x</caption><tr id=r><td>y`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.StartPos != 0 || template.EndPos != 49 || template.ContentStart != 15 || template.ContentEnd != 49 || template.TextContent != "" || len(template.Children) != 0 {
			t.Fatalf("Unexpected EOF host boundary for persistent in-table mode: %#v", template)
		}
		if fragment.TextContent != "xy" || len(fragment.Children) != 2 || fragment.Children[0].Name != "caption" || fragment.Children[0].Attributes["id"] != "c" || fragment.Children[0].TextContent != "x" || fragment.Children[1].Name != "tbody" || fragment.Children[1].StartPos != 0 || fragment.Children[1].EndPos != 0 {
			t.Fatalf("Caption must be followed by a synthetic tbody in TemplateContent: %#v", fragment)
		}
		tbody := fragment.Children[1]
		if tbody.Parent != fragment || len(tbody.Children) != 1 || tbody.Children[0].Name != "tr" || tbody.Children[0].Attributes["id"] != "r" || tbody.Children[0].Parent != tbody || tbody.Children[0].TextContent != "y" {
			t.Fatalf("Persistent table mode lost the row under synthetic tbody: %#v", tbody)
		}
		batch24AssertNoOrdinaryID(t, document, "c", "r")
	})

	t.Run("in-body mode ignores later table wrappers", func(t *testing.T) {
		const content = `<template id=t><div id=d></div><tr><td>x`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.StartPos != 0 || template.EndPos != 35 || template.ContentStart != 15 || template.ContentEnd != 35 || fragment.TextContent != "x" || len(fragment.Children) != 2 {
			t.Fatalf("Unexpected persistent in-body template result: template=%#v content=%#v", template, fragment)
		}
		if fragment.Children[0].Name != "div" || fragment.Children[0].Attributes["id"] != "d" || fragment.Children[0].TextContent != "" || fragment.Children[1].Type != types.TextNode || fragment.Children[1].Value != "x" || fragment.Children[1].StartPos != 39 || fragment.Children[1].EndPos != 40 {
			t.Fatalf("Later tr/td tokens must be ignored while their text is retained: %#v", fragment.Children)
		}
		if len(formattingElements(fragment, "tr")) != 0 || len(formattingElements(fragment, "td")) != 0 {
			t.Fatalf("Persistent in-body mode emitted table wrappers: %#v", fragment)
		}
		batch24AssertNoOrdinaryID(t, document, "d")
	})
}

func TestParseTemplateInitialPseudoModesPersist(t *testing.T) {
	t.Run("document wrappers switch to in-body", func(t *testing.T) {
		tests := []struct {
			name    string
			content string
		}{
			{name: "html", content: `<template id=t><html id=h><tr id=r><td id=d>x</template><p id=o>y`},
			{name: "head", content: `<template id=t><head id=h><tr id=r><td id=d>x</template><p id=o>y`},
			{name: "body", content: `<template id=t><body id=b><tr id=r><td id=d>x</template><p id=o>y`},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				document, err := NewHTMLParser().Parse(test.content)
				if err != nil {
					t.Fatal(err)
				}
				template := batch24RequireElementByID(t, document, "t")
				fragment := batch24TemplateContent(t, template)
				if template.StartPos != 0 || template.EndPos != 56 || template.ContentStart != 15 || template.ContentEnd != 45 || fragment.TextContent != "x" || len(fragment.Children) != 1 {
					t.Fatalf("Unexpected wrapper pseudo-mode result: template=%#v content=%#v", template, fragment)
				}
				if text := fragment.Children[0]; text.Type != types.TextNode || text.Value != "x" || text.StartPos != 44 || text.EndPos != 45 || text.Parent != fragment {
					t.Fatalf("html/head/body and later row wrappers must be ignored, retaining x: %#v", text)
				}
				if body := parsedBodyChildren(document); len(body) != 1 || body[0].Name != "p" || body[0].Attributes["id"] != "o" || body[0].TextContent != "y" {
					t.Fatalf("Parsing did not resume after the template: %#v", body)
				}
				batch24AssertNoOrdinaryID(t, document, "h", "b", "r", "d")
			})
		}
	})

	t.Run("column group mode ignores non-column tokens", func(t *testing.T) {
		const content = `<template id=t><col id=c1><col id=c2><tr id=r><td id=d>x</template><p id=o>y`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.EndPos != 67 || template.ContentStart != 15 || template.ContentEnd != 56 || fragment.TextContent != "" || len(fragment.Children) != 2 {
			t.Fatalf("Unexpected persistent column-group pseudo-mode: template=%#v content=%#v", template, fragment)
		}
		for i, want := range []struct {
			id, name string
			start    int
			end      int
		}{{"c1", "col", 15, 26}, {"c2", "col", 26, 37}} {
			col := fragment.Children[i]
			if col.Name != want.name || col.Attributes["id"] != want.id || col.StartPos != want.start || col.EndPos != want.end || col.Parent != fragment {
				t.Fatalf("Column %d diverged from browser/parse5: %#v", i, col)
			}
		}
		batch24AssertNoOrdinaryID(t, document, "c1", "c2", "r", "d")
	})

	t.Run("row mode ignores caption wrapper and retains rows", func(t *testing.T) {
		const content = `<template id=t><tr id=r1><td>a</td></tr><caption id=c>x</caption><tr id=r2><td>b</template><p id=o>y`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.EndPos != 91 || template.ContentEnd != 80 || fragment.TextContent != "axb" || len(fragment.Children) != 3 {
			t.Fatalf("Unexpected persistent row pseudo-mode: template=%#v content=%#v", template, fragment)
		}
		first, middle, second := fragment.Children[0], fragment.Children[1], fragment.Children[2]
		if first.Name != "tr" || first.Attributes["id"] != "r1" || first.StartPos != 15 || first.EndPos != 40 || first.TextContent != "a" || middle.Type != types.TextNode || middle.Value != "x" || middle.StartPos != 54 || middle.EndPos != 55 || second.Name != "tr" || second.Attributes["id"] != "r2" || second.StartPos != 65 || second.EndPos != 80 || second.TextContent != "b" {
			t.Fatalf("Caption wrapper must be ignored between direct rows: %#v", fragment.Children)
		}
		if len(formattingElements(fragment, "caption")) != 0 {
			t.Fatalf("Persistent row mode emitted a caption wrapper: %#v", fragment)
		}
		batch24AssertNoOrdinaryID(t, document, "r1", "c", "r2")
	})

	t.Run("cell close restores row mode", func(t *testing.T) {
		const content = `<template id=t><td id=d1>a</td><tr id=r2><td id=d2>b</template><p id=o>y`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.EndPos != 63 || template.ContentEnd != 52 || fragment.TextContent != "ab" || len(fragment.Children) != 2 {
			t.Fatalf("Unexpected cell-to-row pseudo-mode transition: template=%#v content=%#v", template, fragment)
		}
		first, second := fragment.Children[0], fragment.Children[1]
		if first.Name != "td" || first.Attributes["id"] != "d1" || first.StartPos != 15 || first.EndPos != 31 || first.TextContent != "a" || second.Name != "td" || second.Attributes["id"] != "d2" || second.StartPos != 41 || second.EndPos != 52 || second.TextContent != "b" {
			t.Fatalf("Intervening row token must be ignored after the first cell closes: %#v", fragment.Children)
		}
		if len(formattingElements(fragment, "tr")) != 0 {
			t.Fatalf("Cell-to-row transition emitted a row wrapper: %#v", fragment)
		}
		batch24AssertNoOrdinaryID(t, document, "d1", "r2", "d2")
	})
}

func TestParseTemplateColumnModeNestedTemplateDoesNotHang(t *testing.T) {
	const content = `<template id=o><col><template id=i><span>x</template><tr id=r><td>y</template><p id=p>z`
	type parseResult struct {
		document *types.Node
		err      error
	}
	parsed := make(chan parseResult, 1)
	go func() {
		document, err := NewHTMLParser().Parse(content)
		parsed <- parseResult{document: document, err: err}
	}()

	var document *types.Node
	select {
	case result := <-parsed:
		if result.err != nil {
			t.Fatal(result.err)
		}
		document = result.document
	case <-time.After(2 * time.Second):
		t.Fatal("column-group pseudo mode hung while entering a nested template")
	}

	outer := batch24RequireElementByID(t, document, "o")
	contentFragment := batch24TemplateContent(t, outer)
	if outer.EndPos != 78 || outer.ContentStart != 15 || outer.ContentEnd != 67 || contentFragment.TextContent != "" || len(contentFragment.Children) != 2 {
		t.Fatalf("Unexpected outer column-mode template: template=%#v content=%#v", outer, contentFragment)
	}
	col, inner := contentFragment.Children[0], contentFragment.Children[1]
	if col.Name != "col" || col.StartPos != 15 || col.EndPos != 20 || col.Parent != contentFragment {
		t.Fatalf("Expected the source-backed col before the nested template: %#v", col)
	}
	if inner.Name != "template" || inner.Attributes["id"] != "i" || inner.StartPos != 20 || inner.EndPos != 53 || inner.ContentStart != 35 || inner.ContentEnd != 42 || inner.Parent != contentFragment {
		t.Fatalf("Unexpected nested template in column-group pseudo mode: %#v", inner)
	}
	innerFragment := batch24TemplateContent(t, inner)
	if innerFragment.TextContent != "x" || len(innerFragment.Children) != 1 || innerFragment.Children[0].Name != "span" || innerFragment.Children[0].StartPos != 35 || innerFragment.Children[0].EndPos != 42 || innerFragment.Children[0].TextContent != "x" {
		t.Fatalf("Nested template must retain only span>x: %#v", innerFragment)
	}
	if len(formattingElements(contentFragment, "tr")) != 0 || len(formattingElements(contentFragment, "td")) != 0 || strings.Contains(contentFragment.TextContent, "y") {
		t.Fatalf("Outer row/cell tokens after the nested template must be ignored: %#v", contentFragment)
	}
	if body := parsedBodyChildren(document); len(body) != 1 || body[0].Attributes["id"] != "p" || body[0].TextContent != "z" {
		t.Fatalf("Parsing did not resume after the outer template: %#v", body)
	}
	batch24AssertNoOrdinaryID(t, document, "i", "r")
}

func TestParseTemplateTableModeFormAfterCaptionIsIgnored(t *testing.T) {
	// Current WHATWG behavior, as implemented by parse5, omits the form and
	// retains its text directly. Chrome 151 instead retains an empty form before
	// the text, so this intentionally remains a direct parser regression rather
	// than a browser comparator case.
	const content = `<template id=t><caption>c</caption><form id=f>x</form></template>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	template := batch24RequireElementByID(t, document, "t")
	fragment := batch24TemplateContent(t, template)
	if template.EndPos != 65 || template.ContentStart != 15 || template.ContentEnd != 54 || fragment.TextContent != "cx" || len(fragment.Children) != 2 {
		t.Fatalf("Unexpected table-mode form recovery: template=%#v content=%#v", template, fragment)
	}
	caption, text := fragment.Children[0], fragment.Children[1]
	if caption.Name != "caption" || caption.StartPos != 15 || caption.EndPos != 35 || caption.TextContent != "c" || text.Type != types.TextNode || text.Value != "x" || text.StartPos != 46 || text.EndPos != 47 {
		t.Fatalf("Expected caption followed by direct form text: %#v", fragment.Children)
	}
	if len(formattingElements(fragment, "form")) != 0 {
		t.Fatalf("parse5/WHATWG table mode must omit the form element: %#v", fragment)
	}
}

func TestParseTemplateTableStartWithoutScopeIsIgnored(t *testing.T) {
	t.Run("after caption", func(t *testing.T) {
		const content = `<template id=t><caption>c</caption><table id=q><tr id=r><td>y</template><p id=p>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.EndPos != 72 || template.ContentEnd != 61 || fragment.TextContent != "cy" || len(fragment.Children) != 2 {
			t.Fatalf("Unexpected ignored table start after caption: template=%#v content=%#v", template, fragment)
		}
		caption, tbody := fragment.Children[0], fragment.Children[1]
		if caption.Name != "caption" || caption.StartPos != 15 || caption.EndPos != 35 || caption.TextContent != "c" || tbody.Name != "tbody" || tbody.StartPos != 0 || tbody.EndPos != 0 || len(tbody.Children) != 1 {
			t.Fatalf("Expected caption followed by synthetic tbody: %#v", fragment.Children)
		}
		row := tbody.Children[0]
		if row.Name != "tr" || row.Attributes["id"] != "r" || row.StartPos != 47 || row.EndPos != 61 || row.TextContent != "y" {
			t.Fatalf("Table contents were not reprocessed in retained table mode: %#v", row)
		}
		if len(formattingElements(fragment, "table")) != 0 {
			t.Fatalf("A table start without a table in scope must be ignored: %#v", fragment)
		}
	})

	t.Run("after initial row", func(t *testing.T) {
		const content = `<template id=t><tr id=a><td>x</td></tr><table id=q><tr id=b><td>y</template><p id=p>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.EndPos != 76 || template.ContentEnd != 65 || fragment.TextContent != "xy" || len(fragment.Children) != 2 {
			t.Fatalf("Unexpected ignored table start after row: template=%#v content=%#v", template, fragment)
		}
		for i, want := range []struct {
			id, text   string
			start, end int
		}{{"a", "x", 15, 39}, {"b", "y", 51, 65}} {
			row := fragment.Children[i]
			if row.Name != "tr" || row.Attributes["id"] != want.id || row.StartPos != want.start || row.EndPos != want.end || row.TextContent != want.text {
				t.Fatalf("Retained row %d diverged after ignored table start: %#v", i, row)
			}
		}
		if len(formattingElements(fragment, "table")) != 0 {
			t.Fatalf("A table start without scope must not wrap retained rows: %#v", fragment)
		}
	})
}

func TestParseTemplateTableDerivedSelectClosesOnStructure(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		end        int
		contentEnd int
		cellID     string
		rowID      string
	}{
		{name: "tr token", content: `<template id=t><caption>c</caption><select id=s><option>a<tr id=r><td>b</template><p id=p>z`, end: 82, contentEnd: 71, rowID: "r"},
		{name: "td token", content: `<template id=t><caption>c</caption><select id=s><option>a<td id=d>b</template><p id=p>z`, end: 78, contentEnd: 67, cellID: "d"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			template := batch24RequireElementByID(t, document, "t")
			fragment := batch24TemplateContent(t, template)
			if template.EndPos != test.end || template.ContentEnd != test.contentEnd || fragment.TextContent != "cab" || len(fragment.Children) != 3 {
				t.Fatalf("Unexpected table-derived select recovery: template=%#v content=%#v", template, fragment)
			}
			caption, selectNode, tbody := fragment.Children[0], fragment.Children[1], fragment.Children[2]
			if caption.Name != "caption" || caption.TextContent != "c" || selectNode.Name != "select" || selectNode.Attributes["id"] != "s" || selectNode.StartPos != 35 || selectNode.EndPos != 57 || selectNode.TextContent != "a" || tbody.Name != "tbody" || len(tbody.Children) != 1 {
				t.Fatalf("Structural token must close select before a synthetic tbody: %#v", fragment.Children)
			}
			row := tbody.Children[0]
			if row.Name != "tr" || row.Attributes["id"] != test.rowID || len(row.Children) != 1 || row.Children[0].Name != "td" || row.Children[0].Attributes["id"] != test.cellID || row.TextContent != "b" {
				t.Fatalf("Structural token was not reprocessed outside select: %#v", row)
			}
		})
	}

	t.Run("in-body control keeps structural tokens in select", func(t *testing.T) {
		const content = `<template id=t><div></div><select id=s><option>x<tr id=r><td>y</template><p id=p>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.EndPos != 73 || template.ContentEnd != 62 || fragment.TextContent != "xy" || len(fragment.Children) != 2 {
			t.Fatalf("Unexpected in-body select control: template=%#v content=%#v", template, fragment)
		}
		selectNode := fragment.Children[1]
		rows, cells := formattingElements(fragment, "tr"), formattingElements(fragment, "td")
		if fragment.Children[0].Name != "div" || selectNode.Name != "select" || selectNode.Attributes["id"] != "s" || selectNode.StartPos != 26 || selectNode.EndPos != 62 || selectNode.TextContent != "xy" || len(rows) != 0 || len(cells) != 0 {
			t.Fatalf("In-body select must ignore row wrappers and retain their text in option: select=%#v rows=%#v cells=%#v", selectNode, rows, cells)
		}
	})
}

func TestParseTemplateGenericFramesReturnToPersistentMode(t *testing.T) {
	t.Run("caption mode closes div on row", func(t *testing.T) {
		const content = `<template id=t><caption>c</caption><div id=d>x<tr id=r><td>y</template><p id=o>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.EndPos != 71 || template.ContentEnd != 60 || fragment.TextContent != "cxy" || len(fragment.Children) != 3 || fragment.Children[0].Name != "caption" || fragment.Children[1].Name != "div" || fragment.Children[1].Attributes["id"] != "d" || fragment.Children[1].StartPos != 35 || fragment.Children[1].EndPos != 46 || fragment.Children[1].TextContent != "x" || fragment.Children[2].Name != "tbody" {
			t.Fatalf("Row token did not close div and return to retained table mode: template=%#v content=%#v", template, fragment)
		}
		row := fragment.Children[2].Children[0]
		if row.Name != "tr" || row.Attributes["id"] != "r" || row.StartPos != 46 || row.EndPos != 60 || row.TextContent != "y" {
			t.Fatalf("Row was not reprocessed beneath synthetic tbody: %#v", row)
		}
	})

	t.Run("caption mode closes formatting frame on row", func(t *testing.T) {
		const content = `<template id=t><caption>c</caption><b id=d>x<tr id=r><td>y</template><p id=o>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.EndPos != 69 || template.ContentEnd != 58 || fragment.TextContent != "cxy" || len(fragment.Children) != 3 || fragment.Children[1].Name != "b" || fragment.Children[1].Attributes["id"] != "d" || fragment.Children[1].StartPos != 35 || fragment.Children[1].EndPos != 44 || fragment.Children[1].TextContent != "x" || fragment.Children[2].Name != "tbody" {
			t.Fatalf("Row token did not close formatting frame and return to table mode: template=%#v content=%#v", template, fragment)
		}
		body := parsedBodyChildren(document)
		if len(body) != 1 || body[0].Name != "p" || body[0].Attributes["id"] != "o" || body[0].TextContent != "z" || len(body[0].Children) != 1 || body[0].Children[0].Type != types.TextNode || body[0].Children[0].Value != "z" {
			t.Fatalf("Template AFE marker must prevent formatting reconstruction outside content: %#v", body)
		}
	})

	t.Run("caption mode closes paragraph on section", func(t *testing.T) {
		const content = `<template id=t><caption>c</caption><p id=d>x<tbody id=s><tr id=r><td>y</template><p id=o>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.EndPos != 81 || template.ContentEnd != 70 || fragment.TextContent != "cxy" || len(fragment.Children) != 3 || fragment.Children[1].Name != "p" || fragment.Children[1].Attributes["id"] != "d" || fragment.Children[1].StartPos != 35 || fragment.Children[1].EndPos != 44 || fragment.Children[1].TextContent != "x" || fragment.Children[2].Name != "tbody" || fragment.Children[2].Attributes["id"] != "s" || fragment.Children[2].StartPos != 44 || fragment.Children[2].EndPos != 70 {
			t.Fatalf("Section token did not close paragraph and return to table mode: template=%#v content=%#v", template, fragment)
		}
	})

	t.Run("row mode closes div and reprocesses row", func(t *testing.T) {
		const content = `<template id=t><tr id=a><td>x</td></tr><div id=d>m<tr id=b><td>y</template><p id=o>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.EndPos != 75 || template.ContentEnd != 64 || fragment.TextContent != "xmy" || len(fragment.Children) != 3 {
			t.Fatalf("Unexpected generic frame in retained row mode: template=%#v content=%#v", template, fragment)
		}
		first, div, second := fragment.Children[0], fragment.Children[1], fragment.Children[2]
		if first.Name != "tr" || first.Attributes["id"] != "a" || first.StartPos != 15 || first.EndPos != 39 || div.Name != "div" || div.Attributes["id"] != "d" || div.StartPos != 39 || div.EndPos != 50 || div.TextContent != "m" || second.Name != "tr" || second.Attributes["id"] != "b" || second.StartPos != 50 || second.EndPos != 64 {
			t.Fatalf("Expected row, closed div, row siblings: %#v", fragment.Children)
		}
	})

	t.Run("cell mode closes div and reprocesses cell", func(t *testing.T) {
		const content = `<template id=t><td id=a>x</td><div id=d>m<td id=b>y</template><p id=o>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.EndPos != 62 || template.ContentEnd != 51 || fragment.TextContent != "xmy" || len(fragment.Children) != 3 {
			t.Fatalf("Unexpected generic frame in retained cell mode: template=%#v content=%#v", template, fragment)
		}
		first, div, second := fragment.Children[0], fragment.Children[1], fragment.Children[2]
		if first.Name != "td" || first.Attributes["id"] != "a" || first.StartPos != 15 || first.EndPos != 30 || div.Name != "div" || div.Attributes["id"] != "d" || div.StartPos != 30 || div.EndPos != 41 || div.TextContent != "m" || second.Name != "td" || second.Attributes["id"] != "b" || second.StartPos != 41 || second.EndPos != 51 {
			t.Fatalf("Expected cell, closed div, cell siblings: %#v", fragment.Children)
		}
	})
}

func TestParseTemplateMathMLIntegrationReturnsToPersistentMode(t *testing.T) {
	tests := []struct {
		name, content        string
		end, contentEnd      int
		mathIndex, tailIndex int
		tailName, tailID     string
	}{
		{name: "caption row", content: `<template id=t><caption>c</caption><math id=m><mtext id=x>q<tr id=r><td>y</template><p id=o>z`, end: 84, contentEnd: 73, mathIndex: 1, tailIndex: 2, tailName: "tbody", tailID: ""},
		{name: "direct row", content: `<template id=t><tr id=a><td>x</td></tr><math id=m><mtext id=n>q<tr id=b><td>y</template><p id=o>z`, end: 88, contentEnd: 77, mathIndex: 1, tailIndex: 2, tailName: "tr", tailID: "b"},
		{name: "direct cell", content: `<template id=t><td id=a>x</td><math id=m><mtext id=n>q<td id=b>y</template><p id=o>z`, end: 75, contentEnd: 64, mathIndex: 1, tailIndex: 2, tailName: "td", tailID: "b"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			template := batch24RequireElementByID(t, document, "t")
			fragment := batch24TemplateContent(t, template)
			if template.EndPos != test.end || template.ContentEnd != test.contentEnd || len(fragment.Children) != 3 {
				t.Fatalf("Unexpected MathML integration transition: template=%#v content=%#v", template, fragment)
			}
			math := fragment.Children[test.mathIndex]
			if math.Name != "math" || math.NamespaceURI != "http://www.w3.org/1998/Math/MathML" || len(math.Children) != 1 || math.Children[0].Name != "mtext" || math.Children[0].NamespaceURI != "http://www.w3.org/1998/Math/MathML" || math.Children[0].TextContent != "q" {
				t.Fatalf("Expected closed MathML mtext integration subtree: %#v", math)
			}
			tail := fragment.Children[test.tailIndex]
			if tail.Name != test.tailName || tail.Attributes["id"] != test.tailID || tail.TextContent != "y" {
				t.Fatalf("Structural token was not reprocessed after MathML unwind: %#v", tail)
			}
		})
	}
}

func TestParseTemplateFrameStartsAreIgnored(t *testing.T) {
	tests := []struct {
		name, content   string
		end, contentEnd int
		textStart       int
		tableDerived    bool
	}{
		{name: "initial frame", content: `<template id=t><frame id=f>x</template><p id=o>z`, end: 39, contentEnd: 28, textStart: 27},
		{name: "initial frameset", content: `<template id=t><frameset id=f>x</template><p id=o>z`, end: 42, contentEnd: 31, textStart: 30},
		{name: "table frame", content: `<template id=t><caption>c</caption><frame id=f><tr id=r><td>y</template><p id=o>z`, end: 72, contentEnd: 61, tableDerived: true},
		{name: "table frameset", content: `<template id=t><caption>c</caption><frameset id=f><tr id=r><td>y</template><p id=o>z`, end: 75, contentEnd: 64, tableDerived: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			template := batch24RequireElementByID(t, document, "t")
			fragment := batch24TemplateContent(t, template)
			if template.EndPos != test.end || template.ContentEnd != test.contentEnd || len(formattingElements(fragment, "frame")) != 0 || len(formattingElements(fragment, "frameset")) != 0 {
				t.Fatalf("frame/frameset start was not ignored: template=%#v content=%#v", template, fragment)
			}
			if !test.tableDerived {
				if fragment.TextContent != "x" || len(fragment.Children) != 1 || fragment.Children[0].Type != types.TextNode || fragment.Children[0].StartPos != test.textStart || fragment.Children[0].EndPos != test.textStart+1 {
					t.Fatalf("Initial frame token must leave direct x text: %#v", fragment)
				}
			} else if fragment.TextContent != "cy" || len(fragment.Children) != 2 || fragment.Children[0].Name != "caption" || fragment.Children[1].Name != "tbody" || fragment.Children[1].TextContent != "y" {
				t.Fatalf("Table-derived frame token disturbed retained table mode: %#v", fragment)
			}
		})
	}
}

func TestParseTemplateAFEReconstructsInsideAfterTableTransition(t *testing.T) {
	tests := []struct {
		name, content   string
		end, contentEnd int
		nested          bool
	}{
		{name: "b", content: `<template id=t><caption>c</caption><b id=b>x<tr id=r><td>y</tr>z</template><p id=o>w`, end: 75, contentEnd: 64},
		{name: "nested b i", content: `<template id=t><caption>c</caption><b id=b><i id=i>x<tr id=r><td>y</tr>z</template><p id=o>w`, end: 83, contentEnd: 72, nested: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			template := batch24RequireElementByID(t, document, "t")
			fragment := batch24TemplateContent(t, template)
			if template.EndPos != test.end || template.ContentEnd != test.contentEnd || fragment.TextContent != "cxyz" || len(fragment.Children) != 4 || fragment.Children[0].Name != "caption" || fragment.Children[1].Name != "b" || fragment.Children[1].TextContent != "x" || fragment.Children[2].Name != "tbody" || fragment.Children[2].TextContent != "y" || fragment.Children[3].Name != "b" || fragment.Children[3].TextContent != "z" {
				t.Fatalf("Unexpected AFE reconstruction within TemplateContent: template=%#v content=%#v", template, fragment)
			}
			if test.nested {
				if len(fragment.Children[1].Children) != 1 || fragment.Children[1].Children[0].Name != "i" || fragment.Children[1].Children[0].TextContent != "x" || len(fragment.Children[3].Children) != 1 || fragment.Children[3].Children[0].Name != "i" || fragment.Children[3].Children[0].TextContent != "z" {
					t.Fatalf("Nested b/i formatting entries were not reconstructed together: %#v", fragment.Children)
				}
			}
			body := parsedBodyChildren(document)
			if len(body) != 1 || body[0].Attributes["id"] != "o" || body[0].TextContent != "w" || len(body[0].Children) != 1 || body[0].Children[0].Type != types.TextNode {
				t.Fatalf("AFE reconstruction escaped TemplateContent: %#v", body)
			}
		})
	}
}
