package utils

import (
	"reflect"
	"strings"
	"testing"
	"time"

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
