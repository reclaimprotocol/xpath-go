package utils

import (
	"reflect"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 24 covers real HTML template contents for whole-document HTML parsing.
// The public XPath API still evaluates from the Document and must not cross a
// template-content DocumentFragment. Direct XPath evaluation with a fragment
// context, declarative shadow DOM, scripting/cloning, and fragment parsing are
// intentionally deferred.

func batch24TemplateField(t *testing.T) reflect.StructField {
	t.Helper()
	field, ok := reflect.TypeOf(types.Node{}).FieldByName("TemplateContent")
	if !ok {
		t.Fatal("types.Node must expose TemplateContent *Node for HTML template contents")
	}
	if field.Type != reflect.TypeOf((*types.Node)(nil)) {
		t.Fatalf("TemplateContent must have type *types.Node, got %v", field.Type)
	}
	if got := field.Tag.Get("json"); got != "template_content,omitempty" {
		t.Fatalf("TemplateContent JSON tag = %q, want template_content,omitempty", got)
	}
	return field
}

func batch24TemplateContent(t *testing.T, template *types.Node) *types.Node {
	t.Helper()
	if template == nil || template.Type != types.ElementNode || template.Name != "template" {
		t.Fatalf("Expected an HTML template element, got %#v", template)
	}
	field := batch24TemplateField(t)
	value := reflect.ValueOf(template).Elem().FieldByIndex(field.Index)
	if value.IsNil() {
		t.Fatalf("template %#v has nil TemplateContent", template)
	}
	fragment, ok := value.Interface().(*types.Node)
	if !ok || fragment == nil {
		t.Fatalf("TemplateContent must expose a non-nil *types.Node, got %#v", value.Interface())
	}
	if fragment.Type != types.DocumentFragmentNode || fragment.Name != "#document-fragment" {
		t.Fatalf("TemplateContent must be a #document-fragment/type 11, got %#v", fragment)
	}
	if fragment.Parent != nil {
		t.Fatalf("TemplateContent is parentless in the DOM model, got parent %#v", fragment.Parent)
	}
	return fragment
}

func batch24ElementByID(node *types.Node, id string) *types.Node {
	if node == nil {
		return nil
	}
	if node.Type == types.ElementNode && node.Attributes["id"] == id {
		return node
	}
	for _, child := range node.Children {
		if found := batch24ElementByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

func batch24RequireElementByID(t *testing.T, node *types.Node, id string) *types.Node {
	t.Helper()
	if found := batch24ElementByID(node, id); found != nil {
		return found
	}
	t.Fatalf("Expected ordinary-tree element #%s below %#v", id, node)
	return nil
}

func batch24AssertNoOrdinaryID(t *testing.T, node *types.Node, ids ...string) {
	t.Helper()
	for _, id := range ids {
		if found := batch24ElementByID(node, id); found != nil {
			t.Fatalf("Template-content element #%s leaked into the ordinary document tree: %#v", id, found)
		}
	}
}

func TestNodeExposesTemplateContentDocumentFragment(t *testing.T) {
	batch24TemplateField(t)
}

func TestParseTemplateContentIsNotOrdinaryChildren(t *testing.T) {
	const content = `<div id=o>before<template id=t><span id=in>inner</span><!--tc-->tail</template>after<span id=out>outer</span></div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	div := batch24RequireElementByID(t, document, "o")
	template := batch24RequireElementByID(t, document, "t")
	if div.TextContent != "beforeafterouter" || template.TextContent != "" || len(template.Children) != 0 {
		t.Fatalf("Template contents must not contribute ordinary children/text: div=%#v template=%#v", div, template)
	}
	if len(div.Children) != 4 || div.Children[1] != template || div.Children[0].Value != "before" || div.Children[2].Value != "after" || div.Children[3].Attributes["id"] != "out" {
		t.Fatalf("Unexpected ordinary sibling order around template: %#v", div.Children)
	}
	batch24AssertNoOrdinaryID(t, document, "in")

	fragment := batch24TemplateContent(t, template)
	if fragment.TextContent != "innertail" || len(fragment.Children) != 3 {
		t.Fatalf("Expected span, comment, and tail in TemplateContent, got %#v", fragment)
	}
	span, comment, tail := fragment.Children[0], fragment.Children[1], fragment.Children[2]
	if span.Name != "span" || span.Attributes["id"] != "in" || span.Parent != fragment || span.TextContent != "inner" ||
		comment.Type != types.CommentNode || comment.Value != "tc" || comment.Parent != fragment ||
		tail.Type != types.TextNode || tail.Value != "tail" || tail.Parent != fragment {
		t.Fatalf("Unexpected TemplateContent children: %#v", fragment.Children)
	}
	if fragment.StartPos != 0 || fragment.EndPos != 0 || fragment.ContentStart != 0 || fragment.ContentEnd != 0 || fragment.StartLine != 0 || fragment.StartColumn != 0 || fragment.EndLine != 0 || fragment.EndColumn != 0 {
		t.Fatalf("Parentless parse5 template-content fragments have no source location: %#v", fragment)
	}
	if template.StartPos != 16 || template.EndPos != 79 || template.ContentStart != 31 || template.ContentEnd != 68 ||
		span.StartPos != 31 || span.EndPos != 55 || comment.StartPos != 55 || comment.EndPos != 64 || tail.StartPos != 64 || tail.EndPos != 68 {
		t.Fatalf("Template and content nodes must retain regular source ranges: template=%#v children=%#v", template, fragment.Children)
	}
}

func TestParseNestedTemplatesOwnIndependentFragments(t *testing.T) {
	const content = `<template id=a>A<template id=b><b id=x>X</b></template>Z</template><i id=y>Y</i>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsedHeadChildren(document)) != 1 || parsedHeadChildren(document)[0].Attributes["id"] != "a" {
		t.Fatalf("Top-level template must be a HEAD child: %#v", parsedHeadChildren(document))
	}
	outer := parsedHeadChildren(document)[0]
	outerContent := batch24TemplateContent(t, outer)
	if outer.TextContent != "" || len(outer.Children) != 0 || outerContent.TextContent != "AZ" || len(outerContent.Children) != 3 {
		t.Fatalf("Nested template must contribute empty string-value to its containing fragment: outer=%#v content=%#v", outer, outerContent)
	}
	inner := outerContent.Children[1]
	if inner.Name != "template" || inner.Attributes["id"] != "b" || inner.Parent != outerContent || inner.TextContent != "" || len(inner.Children) != 0 {
		t.Fatalf("Expected nested template as an ordinary child of the outer fragment: %#v", inner)
	}
	innerContent := batch24TemplateContent(t, inner)
	if innerContent == outerContent || innerContent.TextContent != "X" || len(innerContent.Children) != 1 || innerContent.Children[0].Name != "b" || innerContent.Children[0].Attributes["id"] != "x" || innerContent.Children[0].Parent != innerContent {
		t.Fatalf("Nested template requires its own isolated content fragment: %#v", innerContent)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Attributes["id"] != "y" || parsedBodyChildren(document)[0].TextContent != "Y" {
		t.Fatalf("Parsing did not resume in BODY after the outer template: %#v", parsedBodyChildren(document))
	}
	batch24AssertNoOrdinaryID(t, document, "b", "x")
}

func TestParseTemplatePlacementAndTemplateInsertionModes(t *testing.T) {
	t.Run("head", func(t *testing.T) {
		const content = `<head><template id=t><title id=q>x</title><div id=d>y</div></template></head><body><p id=p>z</p></body>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.Parent == nil || template.Parent.Name != "head" || len(fragment.Children) != 2 || fragment.Children[0].Name != "title" || fragment.Children[1].Name != "div" || fragment.TextContent != "xy" {
			t.Fatalf("HEAD template used the wrong insertion target/mode: template=%#v content=%#v", template, fragment)
		}
		batch24AssertNoOrdinaryID(t, document, "q", "d")
	})

	t.Run("body head-only content", func(t *testing.T) {
		const content = `<body><template id=t><base id=b><link id=l><title id=q>x</title><style id=s>y</style></template><p id=p>z</p>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.Parent == nil || template.Parent.Name != "body" || len(fragment.Children) != 4 || fragment.Children[0].Name != "base" || fragment.Children[1].Name != "link" || fragment.Children[2].Name != "title" || fragment.Children[3].Name != "style" || fragment.TextContent != "xy" {
			t.Fatalf("Head-only tokens in a BODY template must stay in TemplateContent: %#v", fragment)
		}
		batch24AssertNoOrdinaryID(t, document, "b", "l", "q", "s")
	})

	t.Run("select option and ordinary flow", func(t *testing.T) {
		const content = `<select id=s><template id=t><option id=i>x</option><div id=d>z</div></template><option id=o>y</option></select>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		selectNode := batch24RequireElementByID(t, document, "s")
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.Parent != selectNode || len(selectNode.Children) != 2 || selectNode.Children[0] != template || selectNode.Children[1].Attributes["id"] != "o" || selectNode.TextContent != "y" {
			t.Fatalf("SELECT must contain the template leaf and outside option only: %#v", selectNode)
		}
		if len(fragment.Children) != 2 || fragment.Children[0].Name != "option" || fragment.Children[0].Attributes["id"] != "i" || fragment.Children[1].Name != "div" || fragment.Children[1].Attributes["id"] != "d" || fragment.TextContent != "xz" {
			t.Fatalf("Template insertion modes must suspend SELECT restrictions: %#v", fragment)
		}
		batch24AssertNoOrdinaryID(t, document, "i", "d")
	})
}

func TestParseTemplateActiveFormattingMarkersAreScoped(t *testing.T) {
	const outerFormatting = `<b id=o>O<template id=t><i id=i>I</b>A</template>Z`
	document, err := NewHTMLParser().Parse(outerFormatting)
	if err != nil {
		t.Fatal(err)
	}
	bold := batch24RequireElementByID(t, document, "o")
	template := batch24RequireElementByID(t, document, "t")
	fragment := batch24TemplateContent(t, template)
	if bold.TextContent != "OZ" || template.Parent != bold || fragment.TextContent != "IA" || len(fragment.Children) != 1 || fragment.Children[0].Name != "i" || fragment.Children[0].Attributes["id"] != "i" || fragment.Children[0].TextContent != "IA" {
		t.Fatalf("Outer formatting must not reconstruct in content, while inner formatting stays scoped: bold=%#v content=%#v", bold, fragment)
	}
	batch24AssertNoOrdinaryID(t, document, "i")

	const noLeak = `<template id=t><b id=i>X</template>Y<p id=p>Z`
	document, err = NewHTMLParser().Parse(noLeak)
	if err != nil {
		t.Fatal(err)
	}
	template = batch24RequireElementByID(t, document, "t")
	fragment = batch24TemplateContent(t, template)
	if len(fragment.Children) != 1 || fragment.Children[0].Name != "b" || fragment.Children[0].TextContent != "X" || len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0].Value != "Y" || parsedBodyChildren(document)[1].Name != "p" || parsedBodyChildren(document)[1].TextContent != "Z" {
		t.Fatalf("Formatting opened in TemplateContent leaked after </template>: content=%#v body=%#v", fragment, parsedBodyChildren(document))
	}
}

func TestParseTemplateFormPointerIsIsolated(t *testing.T) {
	const outerAndInner = `<form id=o><template id=t><form id=i><input id=q></form></template><form id=x></form><input id=z></form>`
	document, err := NewHTMLParser().Parse(outerAndInner)
	if err != nil {
		t.Fatal(err)
	}
	outer := batch24RequireElementByID(t, document, "o")
	template := batch24RequireElementByID(t, document, "t")
	fragment := batch24TemplateContent(t, template)
	if len(outer.Children) != 1 || outer.Children[0] != template || len(fragment.Children) != 1 || fragment.Children[0].Name != "form" || fragment.Children[0].Attributes["id"] != "i" || len(fragment.Children[0].Children) != 1 || fragment.Children[0].Children[0].Attributes["id"] != "q" {
		t.Fatalf("Template must permit its own form without exposing it in the outer form tree: outer=%#v content=%#v", outer, fragment)
	}
	if len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0] != outer || parsedBodyChildren(document)[1].Attributes["id"] != "z" || batch24ElementByID(document, "x") != nil {
		t.Fatalf("Outer form pointer must ignore the second form start and close before z: %#v", parsedBodyChildren(document))
	}
	batch24AssertNoOrdinaryID(t, document, "i", "q")

	const strayEnd = `<form id=o><template id=t></form><input id=i></template><input id=z></form>`
	document, err = NewHTMLParser().Parse(strayEnd)
	if err != nil {
		t.Fatal(err)
	}
	outer = batch24RequireElementByID(t, document, "o")
	template = batch24RequireElementByID(t, document, "t")
	fragment = batch24TemplateContent(t, template)
	if len(fragment.Children) != 1 || fragment.Children[0].Attributes["id"] != "i" || len(outer.Children) != 2 || outer.Children[0] != template || outer.Children[1].Attributes["id"] != "z" {
		t.Fatalf("A form end without a form in template scope must not clear the outer form pointer: outer=%#v content=%#v", outer, fragment)
	}
}

func TestParseTemplateTableInsertionModeBranches(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		fragmentRoot   string
		fragmentID     string
		outsideRoot    string
		outsideID      string
		outsideWrapper string
	}{
		{name: "row", content: `<table id=table><template id=t><tr id=in><td>x</td></tr></template><tr id=out><td>y</td></tr></table>`, fragmentRoot: "tr", fragmentID: "in", outsideRoot: "tr", outsideID: "out", outsideWrapper: "tbody"},
		{name: "cell", content: `<table id=table><template id=t><td id=in>x</template><tr id=out><td>y</table>`, fragmentRoot: "td", fragmentID: "in", outsideRoot: "tr", outsideID: "out", outsideWrapper: "tbody"},
		{name: "section", content: `<table id=table><template id=t><tbody id=in><tr><td>x</template><tr id=out><td>y</table>`, fragmentRoot: "tbody", fragmentID: "in", outsideRoot: "tr", outsideID: "out", outsideWrapper: "tbody"},
		{name: "column", content: `<table id=table><template id=t><col id=in></template><col id=out></table>`, fragmentRoot: "col", fragmentID: "in", outsideRoot: "col", outsideID: "out", outsideWrapper: "colgroup"},
		{name: "caption", content: `<table id=table><template id=t><caption id=in>x</caption></template><caption id=out>y</caption></table>`, fragmentRoot: "caption", fragmentID: "in", outsideRoot: "caption", outsideID: "out"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			table := batch24RequireElementByID(t, document, "table")
			template := batch24RequireElementByID(t, document, "t")
			fragment := batch24TemplateContent(t, template)
			if template.Parent != table || len(fragment.Children) != 1 || fragment.Children[0].Name != test.fragmentRoot || fragment.Children[0].Attributes["id"] != test.fragmentID {
				t.Fatalf("Template %s branch used wrong insertion target: table=%#v content=%#v", test.name, table.Children, fragment)
			}
			batch24AssertNoOrdinaryID(t, document, test.fragmentID)
			outside := batch24RequireElementByID(t, document, test.outsideID)
			if outside.Name != test.outsideRoot {
				t.Fatalf("Outside node #%s = %s, want %s", test.outsideID, outside.Name, test.outsideRoot)
			}
			if test.outsideWrapper != "" && (outside.Parent == nil || outside.Parent.Name != test.outsideWrapper) {
				t.Fatalf("Outside %s must be wrapped by %s: %#v", test.outsideRoot, test.outsideWrapper, outside)
			}
		})
	}
}

func TestParseTemplateTableFosterParentingOrder(t *testing.T) {
	const textFoster = `<div id=o>A<table id=table><template id=t><div id=in>x</div></template>tail<tr><td>y</table>Z</div>`
	document, err := NewHTMLParser().Parse(textFoster)
	if err != nil {
		t.Fatal(err)
	}
	outer := batch24RequireElementByID(t, document, "o")
	table := batch24RequireElementByID(t, document, "table")
	template := batch24RequireElementByID(t, document, "t")
	fragment := batch24TemplateContent(t, template)
	if len(outer.Children) != 3 || outer.Children[0].Type != types.TextNode || outer.Children[0].Value != "Atail" || outer.Children[1] != table || outer.Children[2].Value != "Z" {
		t.Fatalf("Only text outside TemplateContent should foster before table in token order: %#v", outer.Children)
	}
	if template.Parent != table || len(fragment.Children) != 1 || fragment.Children[0].Name != "div" || fragment.Children[0].Attributes["id"] != "in" || fragment.TextContent != "x" {
		t.Fatalf("Template content was incorrectly foster-parented: table=%#v content=%#v", table.Children, fragment)
	}
	batch24AssertNoOrdinaryID(t, document, "in")

	const elementFoster = `<div id=o><table id=table><template id=t><p id=in>x</template><div id=f>z</div><tr><td>y</table></div>`
	document, err = NewHTMLParser().Parse(elementFoster)
	if err != nil {
		t.Fatal(err)
	}
	outer = batch24RequireElementByID(t, document, "o")
	table = batch24RequireElementByID(t, document, "table")
	template = batch24RequireElementByID(t, document, "t")
	fragment = batch24TemplateContent(t, template)
	if len(outer.Children) != 2 || outer.Children[0].Attributes["id"] != "f" || outer.Children[1] != table || template.Parent != table {
		t.Fatalf("Outside div must foster immediately before table while template stays under table: %#v", outer.Children)
	}
	if len(fragment.Children) != 1 || fragment.Children[0].Name != "p" || fragment.Children[0].Attributes["id"] != "in" {
		t.Fatalf("Template paragraph must stay in fragment: %#v", fragment)
	}
}

func TestParseTemplateEndTagScopeAndEOFRecovery(t *testing.T) {
	t.Run("stray and extra ends", func(t *testing.T) {
		const content = `</template><template id=t>x</template></template><p id=p>y`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		templates := formattingElements(document, "template")
		if len(templates) != 1 || batch24TemplateContent(t, templates[0]).TextContent != "x" || len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Attributes["id"] != "p" {
			t.Fatalf("Stray template ends must be ignored: head=%#v body=%#v", parsedHeadChildren(document), parsedBodyChildren(document))
		}
	})

	t.Run("matching end pops inner stack", func(t *testing.T) {
		const content = `<template id=t><div id=d><span id=s>x</template><p id=p>y`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		fragment := batch24TemplateContent(t, batch24RequireElementByID(t, document, "t"))
		if len(fragment.Children) != 1 || fragment.Children[0].Attributes["id"] != "d" || len(fragment.Children[0].Children) != 1 || fragment.Children[0].Children[0].Attributes["id"] != "s" || fragment.TextContent != "x" || len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Attributes["id"] != "p" {
			t.Fatalf("</template> must pop open content elements and resume outside: content=%#v body=%#v", fragment, parsedBodyChildren(document))
		}
	})

	t.Run("nested closes one at a time", func(t *testing.T) {
		const content = `<template id=a><template id=b><i id=i>x</template><b id=c>y</template><p id=p>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		outer := batch24RequireElementByID(t, document, "a")
		outerContent := batch24TemplateContent(t, outer)
		if len(outerContent.Children) != 2 || outerContent.Children[0].Attributes["id"] != "b" || outerContent.Children[1].Attributes["id"] != "c" || outerContent.TextContent != "y" {
			t.Fatalf("Inner </template> must not close the outer template: %#v", outerContent)
		}
		innerContent := batch24TemplateContent(t, outerContent.Children[0])
		if len(innerContent.Children) != 1 || innerContent.Children[0].Attributes["id"] != "i" || innerContent.TextContent != "x" {
			t.Fatalf("Nested content was not isolated: %#v", innerContent)
		}
	})

	t.Run("self-closing flag ignored", func(t *testing.T) {
		const content = `<template id=t /><b id=b>x</b>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if len(fragment.Children) != 1 || fragment.Children[0].Name != "b" || fragment.Children[0].Attributes["id"] != "b" || fragment.TextContent != "x" || len(parsedBodyChildren(document)) != 0 {
			t.Fatalf("HTML must ignore template's self-closing flag: content=%#v body=%#v", fragment, parsedBodyChildren(document))
		}
		if template.StartPos != 0 || template.EndPos != 26 || template.ContentStart != 17 || template.ContentEnd != 26 {
			t.Fatalf("parse5 closes the EOF-open template at the child's explicit end-tag start: %#v", template)
		}
	})
}

func TestParseTemplateReviewerClosureCases(t *testing.T) {
	t.Run("table end is blocked by template scope", func(t *testing.T) {
		const content = `<table id=o><template id=t></table><p id=in>x</template><tr id=out><td>y</table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		table := batch24RequireElementByID(t, document, "o")
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.Parent != table || len(fragment.Children) != 1 || fragment.Children[0].Name != "p" || fragment.Children[0].Attributes["id"] != "in" || fragment.TextContent != "x" {
			t.Fatalf("An outer </table> inside template scope must be ignored: table=%#v content=%#v", table.Children, fragment)
		}
		outside := batch24RequireElementByID(t, document, "out")
		if outside.Parent == nil || outside.Parent.Name != "tbody" || outside.Parent.Parent != table {
			t.Fatalf("Outer table must remain open for the post-template row: %#v", outside)
		}
		batch24AssertNoOrdinaryID(t, document, "in")
	})

	t.Run("form token in table template mode", func(t *testing.T) {
		const content = `<table id=table><template id=t><form id=in><input id=q></form></template><form id=out></form><tr id=r><td>y</table>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		table := batch24RequireElementByID(t, document, "table")
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if len(fragment.Children) != 1 || fragment.Children[0].Name != "form" || fragment.Children[0].Attributes["id"] != "in" || len(fragment.Children[0].Children) != 1 || fragment.Children[0].Children[0].Attributes["id"] != "q" {
			t.Fatalf("Template table mode must switch to in-body rules for its form: %#v", fragment)
		}
		outsideForm := batch24RequireElementByID(t, document, "out")
		if outsideForm.Parent != table || len(outsideForm.Children) != 0 || table.Children[0] != template || table.Children[1] != outsideForm || table.Children[2].Name != "tbody" {
			t.Fatalf("Outside in-table form must remain an empty table child after template form isolation: %#v", table.Children)
		}
		batch24AssertNoOrdinaryID(t, document, "in", "q")
	})

	t.Run("foreign SVG template stays ordinary", func(t *testing.T) {
		const content = `<svg id=s><template id=t><g id=g>x</g></template><circle id=c /></svg>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		svg := batch24RequireElementByID(t, document, "s")
		template := batch24RequireElementByID(t, document, "t")
		if template.NamespaceURI != "http://www.w3.org/2000/svg" || template.Parent != svg || template.TextContent != "x" || len(template.Children) != 1 || template.Children[0].Name != "g" || template.Children[0].NamespaceURI != "http://www.w3.org/2000/svg" {
			t.Fatalf("Foreign SVG template must use ordinary foreign-content semantics: %#v", template)
		}
		field := batch24TemplateField(t)
		if value := reflect.ValueOf(template).Elem().FieldByIndex(field.Index); !value.IsNil() {
			t.Fatalf("Foreign SVG template must not own HTML TemplateContent: %#v", value.Interface())
		}
	})

	for _, test := range []struct {
		name         string
		content      string
		rootID       string
		rootName     string
		childID      string
		childName    string
		namespace    string
		templateEnd  int
		foreignStart int
		foreignEnd   int
		childStart   int
		contentStart int
	}{
		{name: "template end unwinds open SVG subtree", content: `<template id=t><svg id=s><g id=g>x</template><p id=p>y`, rootID: "s", rootName: "svg", childID: "g", childName: "g", namespace: "http://www.w3.org/2000/svg", templateEnd: 45, foreignStart: 15, foreignEnd: 34, childStart: 25, contentStart: 33},
		{name: "template end unwinds open MathML subtree", content: `<template id=t><math id=m><mrow id=r>x</template><p id=p>y`, rootID: "m", rootName: "math", childID: "r", childName: "mrow", namespace: "http://www.w3.org/1998/Math/MathML", templateEnd: 49, foreignStart: 15, foreignEnd: 38, childStart: 26, contentStart: 37},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			template := batch24RequireElementByID(t, document, "t")
			fragment := batch24TemplateContent(t, template)
			if template.TextContent != "" || len(template.Children) != 0 || template.StartPos != 0 || template.EndPos != test.templateEnd || template.ContentStart != 15 || template.ContentEnd != test.foreignEnd {
				t.Fatalf("Template host did not close at the foreign unwind boundary: %#v", template)
			}
			if len(fragment.Children) != 1 || fragment.TextContent != "x" {
				t.Fatalf("Expected one foreign root in TemplateContent: %#v", fragment)
			}
			foreign := fragment.Children[0]
			if foreign.Name != test.rootName || foreign.Attributes["id"] != test.rootID || foreign.NamespaceURI != test.namespace || foreign.Parent != fragment || foreign.StartPos != test.foreignStart || foreign.EndPos != test.foreignEnd || len(foreign.Children) != 1 {
				t.Fatalf("Open foreign root was not implicitly closed by </template>: %#v", foreign)
			}
			child := foreign.Children[0]
			if child.Name != test.childName || child.Attributes["id"] != test.childID || child.NamespaceURI != test.namespace || child.Parent != foreign || child.StartPos != test.childStart || child.EndPos != test.foreignEnd || child.ContentStart != test.contentStart || child.ContentEnd != test.foreignEnd || child.TextContent != "x" {
				t.Fatalf("Open foreign descendant was not retained/closed in TemplateContent: %#v", child)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" || parsedBodyChildren(document)[0].Attributes["id"] != "p" || parsedBodyChildren(document)[0].TextContent != "y" {
				t.Fatalf("Parser did not resume in BODY after foreign/template unwind: %#v", parsedBodyChildren(document))
			}
			batch24AssertNoOrdinaryID(t, document, test.rootID, test.childID)
		})
	}

	t.Run("comments stay ordered around isolated content comment", func(t *testing.T) {
		const content = `<!--a--><template id=t><!--in--><span>x</span></template><!--b--><p>y</p>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if len(fragment.Children) != 2 || fragment.Children[0].Type != types.CommentNode || fragment.Children[0].Value != "in" || fragment.Children[1].Name != "span" {
			t.Fatalf("Inner comment must stay in TemplateContent: %#v", fragment.Children)
		}
		if len(document.Children) < 2 || document.Children[0].Type != types.CommentNode || document.Children[0].Value != "a" {
			t.Fatalf("Preamble comment must precede the document HTML element: %#v", document.Children)
		}
		head := parsedHeadChildren(document)
		if len(head) != 2 || head[0] != template || head[1].Type != types.CommentNode || head[1].Value != "b" {
			t.Fatalf("Post-template comment must follow the host in HEAD: %#v", head)
		}
	})
}

func TestParseTemplateRegularUTF8CRLFLocations(t *testing.T) {
	const content = "<template id=t>\r\né<!--注--><span>😀</span>\r\n</template>"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	template := batch24RequireElementByID(t, document, "t")
	fragment := batch24TemplateContent(t, template)
	if template.StartPos != 0 || template.EndPos != len(content) || template.ContentStart != 15 || template.ContentEnd != 48 || template.StartLine != 1 || template.StartColumn != 1 || template.EndLine != 3 || template.EndColumn != 12 {
		t.Fatalf("Template range must use UTF-8 bytes and parse5 UTF-16 line/columns: %#v", template)
	}
	if fragment.TextContent != "\né😀\n" || len(fragment.Children) != 4 {
		t.Fatalf("Expected normalized CRLF text, comment, span, and trailing text: %#v", fragment)
	}
	lead, comment, span, trail := fragment.Children[0], fragment.Children[1], fragment.Children[2], fragment.Children[3]
	if lead.Value != "\né" || lead.StartPos != 15 || lead.EndPos != 19 || lead.StartLine != 1 || lead.StartColumn != 16 || lead.EndLine != 2 || lead.EndColumn != 2 ||
		comment.Value != "注" || comment.StartPos != 19 || comment.EndPos != 29 || comment.StartLine != 2 || comment.StartColumn != 2 || comment.EndLine != 2 || comment.EndColumn != 10 ||
		span.StartPos != 29 || span.EndPos != 46 || span.ContentStart != 35 || span.ContentEnd != 39 || span.StartLine != 2 || span.StartColumn != 10 || span.EndLine != 2 || span.EndColumn != 25 ||
		trail.Value != "\n" || trail.StartPos != 46 || trail.EndPos != 48 || trail.StartLine != 2 || trail.StartColumn != 25 || trail.EndLine != 3 || trail.EndColumn != 1 {
		t.Fatalf("Template content locations diverged from parse5: lead=%#v comment=%#v span=%#v trail=%#v", lead, comment, span, trail)
	}
}

func TestParseTemplateStableParse5EOFLocations(t *testing.T) {
	t.Run("ordinary content", func(t *testing.T) {
		const content = `<template id=t><!--c-->a<div id=d>x`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.StartPos != 0 || template.EndPos != 24 || template.ContentStart != 15 || template.ContentEnd != 24 {
			t.Fatalf("parse5 closes an EOF-open template at the next EOF-open child boundary: %#v", template)
		}
		if len(fragment.Children) != 3 || fragment.Children[0].Type != types.CommentNode || fragment.Children[0].StartPos != 15 || fragment.Children[0].EndPos != 23 || fragment.Children[1].Value != "a" || fragment.Children[1].StartPos != 23 || fragment.Children[1].EndPos != 24 {
			t.Fatalf("Unexpected EOF template prefix nodes: %#v", fragment.Children)
		}
		div := fragment.Children[2]
		if div.Name != "div" || div.Attributes["id"] != "d" || div.StartPos != 24 || div.EndPos != 24 || div.ContentStart != 34 || div.ContentEnd != 24 || div.TextContent != "x" || len(div.Children) != 1 || div.Children[0].StartPos != 34 || div.Children[0].EndPos != 35 {
			t.Fatalf("Expected stable parse5 EOF range quirk on the open child: %#v", div)
		}
	})

	t.Run("table content", func(t *testing.T) {
		const content = `<table><template id=t><tr><td>x`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		if template.Parent == nil || template.Parent.Name != "table" || template.StartPos != 7 || template.EndPos != 26 || template.ContentStart != 22 || template.ContentEnd != 26 || len(fragment.Children) != 1 {
			t.Fatalf("Unexpected table-template EOF boundary: template=%#v content=%#v", template, fragment)
		}
		row := fragment.Children[0]
		if row.Name != "tr" || row.StartPos != 22 || row.EndPos != 26 || len(row.Children) != 1 || row.Children[0].Name != "td" || row.Children[0].StartPos != 26 || row.Children[0].EndPos != 26 || row.Children[0].TextContent != "x" {
			t.Fatalf("Table insertion mode EOF locations diverged from parse5: %#v", row)
		}
	})
}
