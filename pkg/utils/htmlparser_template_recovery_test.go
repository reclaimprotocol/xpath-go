package utils

import (
	"strings"
	"testing"
)

func TestParseTemplateStaleForeignAndAbsentEndTags(t *testing.T) {
	t.Run("stale foreign ends preserve table mode", func(t *testing.T) {
		tests := []struct {
			name, content, foreign string
			end, contentEnd        int
		}{
			{name: "math", foreign: "math", content: `<template id=t><caption>c</caption><math id=m><mtext>q<tr id=a><td>x</tr></math><tr id=b><td>y</template><p id=o>z`, end: 105, contentEnd: 94},
			{name: "svg", foreign: "svg", content: `<template id=t><caption>c</caption><svg id=s><foreignObject>q<tr id=a><td>x</tr></svg><tr id=b><td>y</template><p id=o>z`, end: 111, contentEnd: 100},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				document, err := NewHTMLParser().Parse(test.content)
				if err != nil {
					t.Fatal(err)
				}
				template := batch24RequireElementByID(t, document, "t")
				fragment := batch24TemplateContent(t, template)
				if template.EndPos != test.end || template.ContentEnd != test.contentEnd || fragment.TextContent != "cqxy" || len(fragment.Children) != 3 || fragment.Children[0].Name != "caption" || fragment.Children[1].Name != test.foreign || fragment.Children[2].Name != "tbody" || len(fragment.Children[2].Children) != 2 {
					t.Fatalf("Stale foreign end disturbed retained table mode: template=%#v content=%#v", template, fragment)
				}
				rows := fragment.Children[2].Children
				if rows[0].Attributes["id"] != "a" || rows[0].TextContent != "x" || rows[1].Attributes["id"] != "b" || rows[1].TextContent != "y" {
					t.Fatalf("Expected both rows after stale foreign end recovery: %#v", rows)
				}
			})
		}
	})

	t.Run("absent generic ends are ignored in row mode", func(t *testing.T) {
		tests := []struct {
			name, content   string
			end, contentEnd int
		}{
			{name: "div", content: `<template id=t><td id=a>x</div><td id=b>y</template><p id=o>z`, end: 52, contentEnd: 41},
			{name: "svg", content: `<template id=t><td id=a>x</svg><td id=b>y</template><p id=o>z`, end: 52, contentEnd: 41},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				document, err := NewHTMLParser().Parse(test.content)
				if err != nil {
					t.Fatal(err)
				}
				template := batch24RequireElementByID(t, document, "t")
				fragment := batch24TemplateContent(t, template)
				if template.EndPos != test.end || template.ContentEnd != test.contentEnd || fragment.TextContent != "xy" || len(fragment.Children) != 2 || fragment.Children[0].Name != "td" || fragment.Children[0].Attributes["id"] != "a" || fragment.Children[1].Name != "td" || fragment.Children[1].Attributes["id"] != "b" {
					t.Fatalf("Absent generic end disturbed pseudo row mode: template=%#v content=%#v", template, fragment)
				}
			})
		}
	})

	t.Run("row and table ends require actual scope", func(t *testing.T) {
		tests := []struct {
			name, content, childName string
			end, contentEnd          int
		}{
			{name: "direct cell ignores row end", childName: "td", content: `<template id=t><td id=a>x</tr><td id=b>y</template><p id=o>z`, end: 51, contentEnd: 40},
			{name: "direct cell ignores table end", childName: "td", content: `<template id=t><td id=a>x</table><td id=b>y</template><p id=o>z`, end: 54, contentEnd: 43},
			{name: "direct row closes on row end", childName: "tr", content: `<template id=t><tr id=a><td>x</tr><tr id=b><td>y</template><p id=o>z`, end: 59, contentEnd: 48},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				document, err := NewHTMLParser().Parse(test.content)
				if err != nil {
					t.Fatal(err)
				}
				template := batch24RequireElementByID(t, document, "t")
				fragment := batch24TemplateContent(t, template)
				if template.EndPos != test.end || template.ContentEnd != test.contentEnd || fragment.TextContent != "xy" || len(fragment.Children) != 2 || fragment.Children[0].Name != test.childName || fragment.Children[0].Attributes["id"] != "a" || fragment.Children[1].Name != test.childName || fragment.Children[1].Attributes["id"] != "b" {
					t.Fatalf("Scope-sensitive row/table end recovery diverged: template=%#v content=%#v", template, fragment)
				}
			})
		}
	})
}

func TestParseTemplateRecursiveSuppressionAndForeignControls(t *testing.T) {
	t.Run("forms inside generic frames stay table-mode suppressed", func(t *testing.T) {
		tests := []struct {
			name, content, first, last string
			end, contentEnd            int
		}{
			{name: "caption", first: "caption", last: "tbody", content: `<template id=t><caption>c</caption><div id=d>a<form id=f>x</form><tr id=r><td>y</template><p id=o>z`, end: 90, contentEnd: 79},
			{name: "row", first: "tr", last: "tr", content: `<template id=t><tr id=a><td>q</td></tr><div id=d>a<form id=f>x</form><tr id=b><td>y</template><p id=o>z`, end: 94, contentEnd: 83},
			{name: "cell", first: "td", last: "td", content: `<template id=t><td id=a>q</td><div id=d>a<form id=f>x</form><td id=b>y</template><p id=o>z`, end: 81, contentEnd: 70},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				document, err := NewHTMLParser().Parse(test.content)
				if err != nil {
					t.Fatal(err)
				}
				template := batch24RequireElementByID(t, document, "t")
				fragment := batch24TemplateContent(t, template)
				if template.EndPos != test.end || template.ContentEnd != test.contentEnd || len(fragment.Children) != 3 || fragment.Children[0].Name != test.first || fragment.Children[1].Name != "div" || fragment.Children[1].Attributes["id"] != "d" || fragment.Children[1].TextContent != "ax" || fragment.Children[2].Name != test.last || len(formattingElements(fragment, "form")) != 0 {
					t.Fatalf("Recursive table-mode form suppression diverged: template=%#v content=%#v", template, fragment)
				}
			})
		}
	})

	t.Run("frame starts inside generic table frame are ignored", func(t *testing.T) {
		for _, tag := range []string{"frame", "frameset"} {
			t.Run(tag, func(t *testing.T) {
				content := `<template id=t><caption>c</caption><div id=d>a<` + tag + ` id=f>x<tr id=r><td>y</template><p id=o>z`
				document, err := NewHTMLParser().Parse(content)
				if err != nil {
					t.Fatal(err)
				}
				fragment := batch24TemplateContent(t, batch24RequireElementByID(t, document, "t"))
				if fragment.TextContent != "caxy" || len(fragment.Children) != 3 || fragment.Children[1].Name != "div" || fragment.Children[1].TextContent != "ax" || fragment.Children[2].Name != "tbody" || len(formattingElements(fragment, tag)) != 0 {
					t.Fatalf("Recursive %s start was not suppressed: %#v", tag, fragment)
				}
			})
		}
	})

	t.Run("ordinary foreign descendants retain table-looking tokens", func(t *testing.T) {
		tests := []struct {
			name, content, root, child, namespace string
			end, contentEnd                       int
		}{
			{name: "svg g", root: "svg", child: "g", namespace: "http://www.w3.org/2000/svg", content: `<template id=t><caption>c</caption><svg id=s><g id=g>q<tr id=r><td>y</template><p id=o>z`, end: 79, contentEnd: 68},
			{name: "math mrow", root: "math", child: "mrow", namespace: "http://www.w3.org/1998/Math/MathML", content: `<template id=t><caption>c</caption><math id=m><mrow id=n>q<tr id=r><td>y</template><p id=o>z`, end: 83, contentEnd: 72},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				document, err := NewHTMLParser().Parse(test.content)
				if err != nil {
					t.Fatal(err)
				}
				template := batch24RequireElementByID(t, document, "t")
				fragment := batch24TemplateContent(t, template)
				if template.EndPos != test.end || template.ContentEnd != test.contentEnd || len(fragment.Children) != 2 || fragment.Children[1].Name != test.root || fragment.Children[1].NamespaceURI != test.namespace || len(fragment.Children[1].Children) != 1 || fragment.Children[1].Children[0].Name != test.child {
					t.Fatalf("Unexpected ordinary foreign control: template=%#v content=%#v", template, fragment)
				}
				foreignRow := formattingElements(fragment.Children[1], "tr")
				foreignCell := formattingElements(fragment.Children[1], "td")
				if len(foreignRow) != 1 || len(foreignCell) != 1 || foreignRow[0].NamespaceURI != test.namespace || foreignCell[0].NamespaceURI != test.namespace || foreignCell[0].TextContent != "y" {
					t.Fatalf("Ordinary foreign owner must retain tr/td in foreign namespace: row=%#v cell=%#v", foreignRow, foreignCell)
				}
			})
		}
	})
}

func TestParseTemplateLocalFormsCanNest(t *testing.T) {
	const content = `<template id=t><form id=a><form id=b>x</form></form></template><p id=p>y`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	template := batch24RequireElementByID(t, document, "t")
	fragment := batch24TemplateContent(t, template)
	if template.StartPos != 0 || template.EndPos != 63 || template.ContentStart != 15 || template.ContentEnd != 52 || fragment.TextContent != "x" || len(fragment.Children) != 1 {
		t.Fatalf("Unexpected nested template-form host/content: template=%#v content=%#v", template, fragment)
	}
	outer := fragment.Children[0]
	if outer.Name != "form" || outer.Attributes["id"] != "a" || outer.Parent != fragment || outer.TextContent != "x" || len(outer.Children) != 1 {
		t.Fatalf("Expected form#a as TemplateContent root: %#v", outer)
	}
	inner := outer.Children[0]
	if inner.Name != "form" || inner.Attributes["id"] != "b" || inner.Parent != outer || inner.TextContent != "x" || len(inner.Children) != 1 || inner.Children[0].Value != "x" {
		t.Fatalf("Template-local form rules must preserve form#a > form#b: %#v", inner)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Attributes["id"] != "p" || parsedBodyChildren(document)[0].TextContent != "y" {
		t.Fatalf("Parsing did not resume outside nested template-local forms: %#v", parsedBodyChildren(document))
	}
	batch24AssertNoOrdinaryID(t, document, "a", "b")
}

func TestParseTemplateStableEOFHostRangeMatrix(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		end          int
		contentEnd   int
		fragmentText string
		childKinds   []string
		childStarts  []int
		childEnds    []int
	}{
		{name: "text", content: `<template>x`, end: 0, contentEnd: 0, fragmentText: "x", childKinds: []string{"#text"}, childStarts: []int{10}, childEnds: []int{11}},
		{name: "comment then text", content: `<template><!--c-->x`, end: 0, contentEnd: 0, fragmentText: "x", childKinds: []string{"#comment", "#text"}, childStarts: []int{10, 18}, childEnds: []int{18, 19}},
		{name: "br then text", content: `<template><br>x`, end: 10, contentEnd: 10, fragmentText: "x", childKinds: []string{"br", "#text"}, childStarts: []int{10, 14}, childEnds: []int{14, 15}},
		{name: "closed div then tail", content: `<template><div>x</div>tail`, end: 16, contentEnd: 16, fragmentText: "xtail", childKinds: []string{"div", "#text"}, childStarts: []int{10, 22}, childEnds: []int{22, 26}},
		{name: "empty div then text", content: `<template><div></div>x`, end: 15, contentEnd: 15, fragmentText: "x", childKinds: []string{"div", "#text"}, childStarts: []int{10, 21}, childEnds: []int{21, 22}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			if len(parsedHeadChildren(document)) != 1 || parsedHeadChildren(document)[0].Name != "template" {
				t.Fatalf("Expected one EOF-open template in HEAD: %#v", parsedHeadChildren(document))
			}
			template := parsedHeadChildren(document)[0]
			fragment := batch24TemplateContent(t, template)
			if template.StartPos != 0 || template.EndPos != test.end || template.ContentStart != 10 || template.ContentEnd != test.contentEnd || template.TextContent != "" || len(template.Children) != 0 {
				t.Fatalf("EOF host range diverged from parse5: %#v", template)
			}
			if fragment.TextContent != test.fragmentText || len(fragment.Children) != len(test.childKinds) {
				t.Fatalf("Unexpected EOF TemplateContent: %#v", fragment)
			}
			for i, child := range fragment.Children {
				if child.Name != test.childKinds[i] || child.StartPos != test.childStarts[i] || child.EndPos != test.childEnds[i] || child.Parent != fragment {
					t.Fatalf("EOF child %d = %#v, want %s %d:%d parent fragment", i, child, test.childKinds[i], test.childStarts[i], test.childEnds[i])
				}
			}
		})
	}
}

func TestParseTemplateParserReuseResetsAllTemplateState(t *testing.T) {
	parser := NewHTMLParser()
	first, err := parser.Parse(`<template id=t><b id=in>x`)
	if err != nil {
		t.Fatal(err)
	}
	if fragment := batch24TemplateContent(t, batch24RequireElementByID(t, first, "t")); len(fragment.Children) != 1 || fragment.Children[0].Attributes["id"] != "in" {
		t.Fatalf("Unexpected first parse template content: %#v", fragment)
	}
	second, err := parser.Parse(`<form id=f><table id=t><tr><td>clean</table><p id=p>tail`)
	if err != nil {
		t.Fatal(err)
	}
	if len(formattingElements(second, "template")) != 0 || batch24RequireElementByID(t, second, "f").TextContent != "cleantail" || batch24RequireElementByID(t, second, "p").TextContent != "tail" {
		t.Fatalf("Template stack/insertion mode leaked across parser reuse: %#v", second)
	}
}

func TestParseTemplateNestedAndSiblingScaling(t *testing.T) {
	const depth = 128
	var nested strings.Builder
	for i := 0; i < depth; i++ {
		nested.WriteString(`<template>`)
	}
	nested.WriteString(`<span>x</span>`)
	for i := 0; i < depth; i++ {
		nested.WriteString(`</template>`)
	}
	document, err := NewHTMLParser().Parse(nested.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(parsedHeadChildren(document)) != 1 || parsedHeadChildren(document)[0].Name != "template" {
		t.Fatalf("Nested templates must expose only the outer template in the document tree: %#v", parsedHeadChildren(document))
	}
	template := parsedHeadChildren(document)[0]
	for level := 0; level < depth; level++ {
		fragment := batch24TemplateContent(t, template)
		if len(fragment.Children) != 1 {
			t.Fatalf("Nested level %d has %d content children", level, len(fragment.Children))
		}
		if level == depth-1 {
			if fragment.Children[0].Name != "span" || fragment.Children[0].TextContent != "x" {
				t.Fatalf("Deepest template lost its span: %#v", fragment)
			}
			break
		}
		if fragment.Children[0].Name != "template" {
			t.Fatalf("Nested level %d contains %#v, want template", level, fragment.Children[0])
		}
		template = fragment.Children[0]
	}

	const siblings = 1000
	var many strings.Builder
	many.WriteString(`<body>`)
	for i := 0; i < siblings; i++ {
		many.WriteString(`<template><span>x</span></template>`)
	}
	document, err = NewHTMLParser().Parse(many.String())
	if err != nil {
		t.Fatal(err)
	}
	body := parsedBodyChildren(document)
	if len(body) != siblings {
		t.Fatalf("Expected %d sibling templates, got %d", siblings, len(body))
	}
	for i, node := range body {
		if node.Name != "template" || len(node.Children) != 0 || node.TextContent != "" {
			t.Fatalf("Sibling %d is not an isolated template leaf: %#v", i, node)
		}
		fragment := batch24TemplateContent(t, node)
		if len(fragment.Children) != 1 || fragment.Children[0].Name != "span" || fragment.TextContent != "x" {
			t.Fatalf("Sibling %d lost content: %#v", i, fragment)
		}
	}
}
