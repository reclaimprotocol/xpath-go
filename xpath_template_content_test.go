package xpath_test

import (
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

// These tests intentionally exercise only whole-document XPath evaluation.
// Chrome rejects DocumentFragment as an XPath context while bundled jsdom
// accepts it, so direct TemplateContent-context XPath is not a portable oracle
// and remains outside Batch 24.

func batch24Query(t *testing.T, content, expression string) []xpath.Result {
	t.Helper()
	results, err := xpath.Query(expression, content)
	if err != nil {
		t.Fatalf("Query %q failed for %q: %v", expression, content, err)
	}
	return results
}

func batch24AssertNoQuery(t *testing.T, content, expression string) {
	t.Helper()
	if results := batch24Query(t, content, expression); len(results) != 0 {
		t.Fatalf("Query %q exposed template content: %#v", expression, results)
	}
}

func batch24ResultIDs(results []xpath.Result) []string {
	ids := make([]string, len(results))
	for i := range results {
		ids[i] = results[i].Attributes["id"]
	}
	return ids
}

func TestQueryTemplateIsLeafInWholeDocumentXPath(t *testing.T) {
	const content = `<div id=o>before<template id=t><span id=in>inner</span><!--tc-->tail</template>after<span id=out>outer</span></div>`
	template := batch24Query(t, content, `//template`)
	if len(template) != 1 || template[0].NodeName != "template" || template[0].NodeType != 1 || template[0].TextContent != "" || template[0].StartLocation != 16 || template[0].EndLocation != 79 || template[0].ContentStart != 31 || template[0].ContentEnd != 68 {
		t.Fatalf("Template element must be a source-backed empty XPath leaf: %#v", template)
	}
	for _, expression := range []string{
		`//template/*`,
		`//template/node()`,
		`//*[@id='in']`,
		`//text()[.='inner']`,
		`//text()[.='tail']`,
		`//comment()[.='tc']`,
	} {
		batch24AssertNoQuery(t, content, expression)
	}
	ids := batch24Query(t, content, `//*[@id]`)
	if got := strings.Join(batch24ResultIDs(ids), ","); got != "o,t,out" {
		t.Fatalf("Whole-document IDs = %q, want o,t,out; results=%#v", got, ids)
	}
	if ids[0].TextContent != "beforeafterouter" || ids[1].TextContent != "" || ids[2].TextContent != "outer" {
		t.Fatalf("Template content contaminated ancestor/string values: %#v", ids)
	}
	texts := batch24Query(t, content, `//text()`)
	if len(texts) != 3 || texts[0].TextContent != "before" || texts[1].TextContent != "after" || texts[2].TextContent != "outer" {
		t.Fatalf("Whole-document text axis crossed TemplateContent: %#v", texts)
	}
	preceding := batch24Query(t, content, `//*[@id='out']/preceding::*[1]`)
	following := batch24Query(t, content, `//template/following::*[1]`)
	if len(preceding) != 1 || preceding[0].NodeName != "template" || len(following) != 1 || following[0].Attributes["id"] != "out" {
		t.Fatalf("Following/preceding axes must treat template as a leaf: preceding=%#v following=%#v", preceding, following)
	}

	contentsOnly, err := xpath.QueryWithOptions(`//template`, content, xpath.Options{IncludeLocation: true, OutputFormat: "nodes", ContentsOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(contentsOnly) != 1 || contentsOnly[0].Value != "" || contentsOnly[0].TextContent != "" || contentsOnly[0].StartLocation != 31 || contentsOnly[0].EndLocation != 68 {
		t.Fatalf("ContentsOnly keeps the source inner range but uses DOM template textContent: %#v", contentsOnly)
	}
}

func TestQueryNestedTemplateContentsStayOutsideDocumentXPath(t *testing.T) {
	const content = `<template id=a>A<template id=b><b id=x>X</b></template>Z</template><i id=y>Y</i>`
	ids := batch24Query(t, content, `//*[@id]`)
	if got := strings.Join(batch24ResultIDs(ids), ","); got != "a,y" {
		t.Fatalf("Document XPath must see outer template and outside i only, got %q (%#v)", got, ids)
	}
	templates := batch24Query(t, content, `//template`)
	if len(templates) != 1 || templates[0].Attributes["id"] != "a" || templates[0].TextContent != "" || templates[0].StartLocation != 0 || templates[0].EndLocation != 67 {
		t.Fatalf("Nested template element leaked out of outer TemplateContent: %#v", templates)
	}
	if head := batch24Query(t, content, `/html/head/template`); len(head) != 1 || head[0].Attributes["id"] != "a" {
		t.Fatalf("Top-level template must be placed in HEAD: %#v", head)
	}
	if body := batch24Query(t, content, `/html/body/i`); len(body) != 1 || body[0].Attributes["id"] != "y" || body[0].TextContent != "Y" {
		t.Fatalf("Parsing must resume in BODY after outer template: %#v", body)
	}
	texts := batch24Query(t, content, `//text()`)
	if len(texts) != 1 || texts[0].TextContent != "Y" {
		t.Fatalf("Nested fragments leaked text nodes into document XPath: %#v", texts)
	}
}

func TestQueryTemplatePlacementAndScopedFormatting(t *testing.T) {
	const placement = `<head><template id=h><title id=t>x</title><div id=d>y</div></template></head><body><template id=b><meta id=m><p id=p>z</p></template><select id=s><template id=q><option id=i>a</option><div id=v>b</div></template><option id=o>c</option></select></body>`
	ids := batch24Query(t, placement, `//*[@id]`)
	if got := strings.Join(batch24ResultIDs(ids), ","); got != "h,b,s,q,o" {
		t.Fatalf("Head/body/select template content leaked into document XPath: %q (%#v)", got, ids)
	}
	if templates := batch24Query(t, placement, `//template`); len(templates) != 3 || templates[0].Attributes["id"] != "h" || templates[1].Attributes["id"] != "b" || templates[2].Attributes["id"] != "q" {
		t.Fatalf("Template hosts are not in document order: %#v", templates)
	}
	if option := batch24Query(t, placement, `//select/option`); len(option) != 1 || option[0].Attributes["id"] != "o" || option[0].TextContent != "c" {
		t.Fatalf("SELECT document children crossed template fragment: %#v", option)
	}

	const formatting = `<b id=o>O<template id=t><i id=i>I</b>A</template>Z`
	visible := batch24Query(t, formatting, `//*[@id]`)
	if got := strings.Join(batch24ResultIDs(visible), ","); got != "o,t" || visible[0].TextContent != "OZ" || visible[1].TextContent != "" {
		t.Fatalf("Formatting/template marker scope diverged from browser DOM: %q (%#v)", got, visible)
	}
	batch24AssertNoQuery(t, formatting, `//*[@id='i']`)

	const noLeak = `<template id=t><b id=i>X</template>Y<p id=p>Z`
	visible = batch24Query(t, noLeak, `//*[@id]`)
	if got := strings.Join(batch24ResultIDs(visible), ","); got != "t,p" || visible[1].TextContent != "Z" {
		t.Fatalf("Formatting opened inside TemplateContent leaked outside: %q (%#v)", got, visible)
	}
	texts := batch24Query(t, noLeak, `/html/body/text() | /html/body/p/text()`)
	if len(texts) != 2 || texts[0].TextContent != "Y" || texts[1].TextContent != "Z" {
		t.Fatalf("Unexpected post-template formatting/text tree: %#v", texts)
	}
}

func TestQueryTemplateFormPointerIsolation(t *testing.T) {
	const content = `<form id=o><template id=t><form id=i><input id=q></form></template><form id=x></form><input id=z></form>`
	ids := batch24Query(t, content, `//*[@id]`)
	if got := strings.Join(batch24ResultIDs(ids), ","); got != "o,t,z" {
		t.Fatalf("Template form-pointer isolation IDs = %q, want o,t,z; %#v", got, ids)
	}
	if outer := batch24Query(t, content, `/html/body/form[@id='o']`); len(outer) != 1 || outer[0].TextContent != "" || outer[0].StartLocation != 0 || outer[0].EndLocation != 85 {
		t.Fatalf("Outer form must close at the ignored second form end: %#v", outer)
	}
	if z := batch24Query(t, content, `/html/body/input[@id='z']`); len(z) != 1 || z[0].StartLocation != 85 || z[0].EndLocation != 97 {
		t.Fatalf("z must be a BODY sibling after the outer form: %#v", z)
	}
	for _, id := range []string{"i", "q", "x"} {
		batch24AssertNoQuery(t, content, `//*[@id='`+id+`']`)
	}

	const strayEnd = `<form id=o><template id=t></form><input id=i></template><input id=z></form>`
	ids = batch24Query(t, strayEnd, `//*[@id]`)
	if got := strings.Join(batch24ResultIDs(ids), ","); got != "o,t,z" {
		t.Fatalf("Out-of-template-scope form end cleared outer pointer: %q (%#v)", got, ids)
	}
}

func TestQueryTemplateTableInsertionModesAndFosterOrder(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		visibleIDs string
		hiddenID   string
	}{
		{name: "row", content: `<table id=table><template id=t><tr id=in><td>x</td></tr></template><tr id=out><td>y</td></tr></table>`, visibleIDs: "table,t,out", hiddenID: "in"},
		{name: "cell", content: `<table id=table><template id=t><td id=in>x</template><tr id=out><td>y</table>`, visibleIDs: "table,t,out", hiddenID: "in"},
		{name: "section", content: `<table id=table><template id=t><tbody id=in><tr><td>x</template><tr id=out><td>y</table>`, visibleIDs: "table,t,out", hiddenID: "in"},
		{name: "column", content: `<table id=table><template id=t><col id=in></template><col id=out></table>`, visibleIDs: "table,t,out", hiddenID: "in"},
		{name: "caption", content: `<table id=table><template id=t><caption id=in>x</caption></template><caption id=out>y</caption></table>`, visibleIDs: "table,t,out", hiddenID: "in"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ids := batch24Query(t, test.content, `//*[@id]`)
			if got := strings.Join(batch24ResultIDs(ids), ","); got != test.visibleIDs {
				t.Fatalf("Visible IDs = %q, want %q; %#v", got, test.visibleIDs, ids)
			}
			batch24AssertNoQuery(t, test.content, `//*[@id='`+test.hiddenID+`']`)
			if template := batch24Query(t, test.content, `//table/template`); len(template) != 1 || template[0].TextContent != "" {
				t.Fatalf("Template must stay a leaf child of table: %#v", template)
			}
		})
	}

	const textFoster = `<div id=o>A<table id=table><template id=t><div id=in>x</div></template>tail<tr><td>y</table>Z</div>`
	children := batch24Query(t, textFoster, `/html/body/div[@id='o']/node()`)
	if len(children) != 3 || children[0].NodeType != 3 || children[0].TextContent != "Atail" || children[1].NodeName != "table" || children[2].TextContent != "Z" {
		t.Fatalf("Outside table text must foster while TemplateContent remains isolated: %#v", children)
	}
	batch24AssertNoQuery(t, textFoster, `//*[@id='in']`)

	const elementFoster = `<div id=o><table id=table><template id=t><p id=in>x</template><div id=f>z</div><tr><td>y</table></div>`
	children = batch24Query(t, elementFoster, `/html/body/div[@id='o']/*`)
	if len(children) != 2 || children[0].Attributes["id"] != "f" || children[1].Attributes["id"] != "table" {
		t.Fatalf("Fostered div must precede table in DOM order: %#v", children)
	}
	if tableChildren := batch24Query(t, elementFoster, `//*[@id='table']/*`); len(tableChildren) != 2 || tableChildren[0].NodeName != "template" || tableChildren[1].NodeName != "tbody" {
		t.Fatalf("Template must remain the first table child: %#v", tableChildren)
	}
	batch24AssertNoQuery(t, elementFoster, `//*[@id='in']`)
}

func TestQueryTemplateEndScopeSelfClosingAndEOF(t *testing.T) {
	const ends = `</template><template id=t><div id=d><span id=s>x</template></template><p id=p>y`
	ids := batch24Query(t, ends, `//*[@id]`)
	if got := strings.Join(batch24ResultIDs(ids), ","); got != "t,p" {
		t.Fatalf("Stray ends or matching end leaked content: %q (%#v)", got, ids)
	}
	if body := batch24Query(t, ends, `/html/body/p`); len(body) != 1 || body[0].TextContent != "y" {
		t.Fatalf("Parser did not resume outside template: %#v", body)
	}

	const nested = `<template id=a><template id=b><i id=i>x</template><b id=c>y</template><p id=p>z`
	ids = batch24Query(t, nested, `//*[@id]`)
	if got := strings.Join(batch24ResultIDs(ids), ","); got != "a,p" {
		t.Fatalf("Nested template end scope leaked fragment IDs: %q (%#v)", got, ids)
	}

	const selfClosing = `<template id=t /><b id=b>x</b>`
	ids = batch24Query(t, selfClosing, `//*[@id]`)
	if got := strings.Join(batch24ResultIDs(ids), ","); got != "t" || ids[0].TextContent != "" {
		t.Fatalf("Template self-closing flag must be ignored: %q (%#v)", got, ids)
	}

	const eof = `<template id=t><!--c-->a<div id=d>x`
	template := batch24Query(t, eof, `//template | //*[@id='d']`)
	if len(template) != 1 || template[0].NodeName != "template" || template[0].TextContent != "" || template[0].StartLocation != 0 || template[0].EndLocation != 24 || template[0].ContentStart != 15 || template[0].ContentEnd != 24 {
		t.Fatalf("Stable parse5 EOF template range/isolation mismatch: %#v", template)
	}
	batch24AssertNoQuery(t, eof, `//comment() | //text()`)

	const tableEOF = `<table><template id=t><tr><td>x`
	template = batch24Query(t, tableEOF, `//template`)
	if len(template) != 1 || template[0].StartLocation != 7 || template[0].EndLocation != 26 || template[0].TextContent != "" {
		t.Fatalf("Table-template EOF boundary mismatch: %#v", template)
	}
	batch24AssertNoQuery(t, tableEOF, `//tr | //td | //text()`)
}

func TestQueryTemplateReviewerClosureCases(t *testing.T) {
	t.Run("table end blocked by template scope", func(t *testing.T) {
		const content = `<table id=o><template id=t></table><p id=in>x</template><tr id=out><td>y</table>`
		ids := batch24Query(t, content, `//*[@id]`)
		if got := strings.Join(batch24ResultIDs(ids), ","); got != "o,t,out" {
			t.Fatalf("Inner </table> escaped template scope: %q (%#v)", got, ids)
		}
		if row := batch24Query(t, content, `//*[@id='o']/tbody/tr`); len(row) != 1 || row[0].Attributes["id"] != "out" || row[0].TextContent != "y" {
			t.Fatalf("Outer table did not remain open for outside row: %#v", row)
		}
	})

	t.Run("form token in table template mode", func(t *testing.T) {
		const content = `<table id=table><template id=t><form id=in><input id=q></form></template><form id=out></form><tr id=r><td>y</table>`
		ids := batch24Query(t, content, `//*[@id]`)
		if got := strings.Join(batch24ResultIDs(ids), ","); got != "table,t,out,r" {
			t.Fatalf("Table-template form pointer/isolation IDs = %q: %#v", got, ids)
		}
		if children := batch24Query(t, content, `//*[@id='table']/*`); len(children) != 3 || children[0].NodeName != "template" || children[1].NodeName != "form" || children[1].Attributes["id"] != "out" || children[2].NodeName != "tbody" {
			t.Fatalf("Outside in-table form order diverged from browser: %#v", children)
		}
	})

	t.Run("foreign SVG template remains ordinary", func(t *testing.T) {
		const content = `<svg id=s><template id=t><g id=g>x</g></template><circle id=c /></svg>`
		ids := batch24Query(t, content, `//*[@id]`)
		if got := strings.Join(batch24ResultIDs(ids), ","); got != "s,t,g,c" {
			t.Fatalf("Foreign template incorrectly acquired HTML fragment semantics: %q (%#v)", got, ids)
		}
		template := batch24Query(t, content, `//*[local-name()='template']`)
		if len(template) != 1 || template[0].NamespaceURI != "http://www.w3.org/2000/svg" || template[0].TextContent != "x" {
			t.Fatalf("Expected ordinary SVG template with visible g/text: %#v", template)
		}
	})

	for _, test := range []struct {
		name        string
		content     string
		foreignExpr string
		end         int
	}{
		{name: "template end unwinds SVG content", content: `<template id=t><svg id=s><g id=g>x</template><p id=p>y`, foreignExpr: `//svg | //g`, end: 45},
		{name: "template end unwinds MathML content", content: `<template id=t><math id=m><mrow id=r>x</template><p id=p>y`, foreignExpr: `//math | //mrow`, end: 49},
	} {
		t.Run(test.name, func(t *testing.T) {
			ids := batch24Query(t, test.content, `//*[@id]`)
			if got := strings.Join(batch24ResultIDs(ids), ","); got != "t,p" || ids[0].TextContent != "" || ids[0].StartLocation != 0 || ids[0].EndLocation != test.end || ids[1].TextContent != "y" {
				t.Fatalf("Foreign TemplateContent leaked through document XPath or swallowed outside p: %q (%#v)", got, ids)
			}
			batch24AssertNoQuery(t, test.content, test.foreignExpr+` | //text()[.='x']`)
			texts := batch24Query(t, test.content, `//text()`)
			if len(texts) != 1 || texts[0].TextContent != "y" {
				t.Fatalf("Only outside p text must be visible after foreign/template unwind: %#v", texts)
			}
		})
	}

	t.Run("comments ordered around isolated comment", func(t *testing.T) {
		const content = `<!--a--><template id=t><!--in--><span>x</span></template><!--b--><p>y</p>`
		comments := batch24Query(t, content, `//template/preceding::comment()[1] | //template/following::comment()[1]`)
		if len(comments) != 2 || comments[0].TextContent != "a" || comments[1].TextContent != "b" {
			t.Fatalf("Outside comments must bracket template in document order: %#v", comments)
		}
		batch24AssertNoQuery(t, content, `//comment()[.='in']`)
	})
}

func TestQueryTemplateUTF8CRLFLocationsAndScaling(t *testing.T) {
	const locations = "<template id=t>\r\né<!--注--><span id=in>😀</span>\r\n</template><p id=out>終</p>"
	results := batch24Query(t, locations, `//*[@id]`)
	if got := strings.Join(batch24ResultIDs(results), ","); got != "t,out" {
		t.Fatalf("Template content ID leaked in UTF-8 document: %q (%#v)", got, results)
	}
	if results[0].StartLocation != 0 || results[0].EndLocation != 65 || results[0].ContentStart != 15 || results[0].ContentEnd != 54 || results[0].TextContent != "" || results[1].TextContent != "終" {
		t.Fatalf("UTF-8 template/public locations mismatch: %#v", results)
	}

	const siblings = 1000
	var content strings.Builder
	content.WriteString(`<body>`)
	for i := 0; i < siblings; i++ {
		content.WriteString(`<template><span>x</span></template>`)
	}
	templates := batch24Query(t, content.String(), `//template`)
	if len(templates) != siblings {
		t.Fatalf("Expected %d sibling template leaves, got %d", siblings, len(templates))
	}
	for i, template := range templates {
		if template.TextContent != "" {
			t.Fatalf("Template %d leaked fragment text %q", i, template.TextContent)
		}
	}
	batch24AssertNoQuery(t, content.String(), `//span | //text()`)
}

func TestQueryTemplatePersistentModesAndNestedLocalForms(t *testing.T) {
	t.Run("caption then row remains isolated", func(t *testing.T) {
		const content = `<template id=t><caption id=c>x</caption><tr id=r><td>y`
		results := batch24Query(t, content, `//*[@id] | //caption | //tbody | //tr | //td | //text()`)
		if len(results) != 1 || results[0].NodeName != "template" || results[0].Attributes["id"] != "t" || results[0].TextContent != "" || results[0].StartLocation != 0 || results[0].EndLocation != 49 || results[0].ContentStart != 15 || results[0].ContentEnd != 49 {
			t.Fatalf("Persistent in-table content leaked into document XPath: %#v", results)
		}
	})

	t.Run("div switches persistent mode to in-body", func(t *testing.T) {
		const content = `<template id=t><div id=d></div><tr><td>x`
		results := batch24Query(t, content, `//*[@id] | //div | //tr | //td | //text()`)
		if len(results) != 1 || results[0].NodeName != "template" || results[0].Attributes["id"] != "t" || results[0].TextContent != "" || results[0].StartLocation != 0 || results[0].EndLocation != 35 || results[0].ContentStart != 15 || results[0].ContentEnd != 35 {
			t.Fatalf("Persistent in-body content leaked into document XPath: %#v", results)
		}
	})

	t.Run("template-local forms can nest", func(t *testing.T) {
		const content = `<template id=t><form id=a><form id=b>x</form></form></template><p id=p>y`
		ids := batch24Query(t, content, `//*[@id]`)
		if got := strings.Join(batch24ResultIDs(ids), ","); got != "t,p" || ids[0].TextContent != "" || ids[1].TextContent != "y" {
			t.Fatalf("Nested template-local forms leaked into document XPath: %q (%#v)", got, ids)
		}
		batch24AssertNoQuery(t, content, `//form | //text()[.='x']`)
		texts := batch24Query(t, content, `//text()`)
		if len(texts) != 1 || texts[0].TextContent != "y" {
			t.Fatalf("Only outside p text must remain visible: %#v", texts)
		}
	})
}

func TestQueryTemplateInitialPseudoModesRemainIsolated(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		end        int
		contentEnd int
	}{
		{name: "html wrapper", content: `<template id=t><html id=h><tr id=r><td id=d>x</template><p id=o>y`, end: 56, contentEnd: 45},
		{name: "head wrapper", content: `<template id=t><head id=h><tr id=r><td id=d>x</template><p id=o>y`, end: 56, contentEnd: 45},
		{name: "body wrapper", content: `<template id=t><body id=b><tr id=r><td id=d>x</template><p id=o>y`, end: 56, contentEnd: 45},
		{name: "column group", content: `<template id=t><col id=c1><col id=c2><tr id=r><td id=d>x</template><p id=o>y`, end: 67, contentEnd: 56},
		{name: "row mode", content: `<template id=t><tr id=r1><td>a</td></tr><caption id=c>x</caption><tr id=r2><td>b</template><p id=o>y`, end: 91, contentEnd: 80},
		{name: "cell to row", content: `<template id=t><td id=d1>a</td><tr id=r2><td id=d2>b</template><p id=o>y`, end: 63, contentEnd: 52},
	}
	const query = `//*[@id] | //html[@id] | //head[@id] | //body[@id] | //col | //caption | //tr | //td | //text()[.='x' or .='a' or .='b']`
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := batch24Query(t, test.content, query)
			if got := strings.Join(batch24ResultIDs(results), ","); got != "t,o" || len(results) != 2 {
				t.Fatalf("Pseudo-mode content leaked into whole-document XPath: %q (%#v)", got, results)
			}
			if results[0].NodeName != "template" || results[0].TextContent != "" || results[0].StartLocation != 0 || results[0].EndLocation != test.end || results[0].ContentStart != 15 || results[0].ContentEnd != test.contentEnd || results[1].NodeName != "p" || results[1].TextContent != "y" {
				t.Fatalf("Unexpected isolated pseudo-mode host/outside node: %#v", results)
			}
		})
	}
}

func TestQueryTemplateColumnModeNestedTemplateRemainsIsolated(t *testing.T) {
	const content = `<template id=o><col><template id=i><span>x</template><tr id=r><td>y</template><p id=p>z`
	results := batch24Query(t, content, `//*[@id] | //col | //template//template | //span | //tr | //td | //text()[.='x' or .='y']`)
	if got := strings.Join(batch24ResultIDs(results), ","); got != "o,p" || len(results) != 2 {
		t.Fatalf("Nested column-mode template content leaked into whole-document XPath: %q (%#v)", got, results)
	}
	if results[0].NodeName != "template" || results[0].StartLocation != 0 || results[0].EndLocation != 78 || results[0].ContentStart != 15 || results[0].ContentEnd != 67 || results[0].TextContent != "" || results[1].NodeName != "p" || results[1].TextContent != "z" {
		t.Fatalf("Unexpected outer host/outside node after nested column-mode template: %#v", results)
	}
}

func TestQueryTemplateTableStartAndSelectRecoveryRemainIsolated(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		end        int
		contentEnd int
	}{
		{name: "table after caption", content: `<template id=t><caption>c</caption><table id=q><tr id=r><td>y</template><p id=p>z`, end: 72, contentEnd: 61},
		{name: "table after row", content: `<template id=t><tr id=a><td>x</td></tr><table id=q><tr id=b><td>y</template><p id=p>z`, end: 76, contentEnd: 65},
		{name: "select closed by tr", content: `<template id=t><caption>c</caption><select id=s><option>a<tr id=r><td>b</template><p id=p>z`, end: 82, contentEnd: 71},
		{name: "select closed by td", content: `<template id=t><caption>c</caption><select id=s><option>a<td id=d>b</template><p id=p>z`, end: 78, contentEnd: 67},
		{name: "in-body select control", content: `<template id=t><div></div><select id=s><option>x<tr id=r><td>y</template><p id=p>z`, end: 73, contentEnd: 62},
	}
	const query = `//*[@id] | //table | //caption | //select | //option | //tbody | //tr | //td | //text()[.='a' or .='b' or .='c' or .='x' or .='y']`
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := batch24Query(t, test.content, query)
			if got := strings.Join(batch24ResultIDs(results), ","); got != "t,p" || len(results) != 2 {
				t.Fatalf("Table/select recovery content leaked into document XPath: %q (%#v)", got, results)
			}
			if results[0].EndLocation != test.end || results[0].ContentStart != 15 || results[0].ContentEnd != test.contentEnd || results[0].TextContent != "" || results[1].NodeName != "p" || results[1].TextContent != "z" {
				t.Fatalf("Unexpected isolated table/select host: %#v", results)
			}
		})
	}
}

func TestQueryTemplateGenericFrameTransitions(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		end        int
		contentEnd int
		wantIDs    string
	}{
		{name: "caption div row", content: `<template id=t><caption>c</caption><div id=d>x<tr id=r><td>y</template><p id=o>z`, end: 71, contentEnd: 60, wantIDs: "t,o"},
		{name: "caption formatting row", content: `<template id=t><caption>c</caption><b id=d>x<tr id=r><td>y</template><p id=o>z`, end: 69, contentEnd: 58, wantIDs: "t,o"},
		{name: "caption paragraph section", content: `<template id=t><caption>c</caption><p id=d>x<tbody id=s><tr id=r><td>y</template><p id=o>z`, end: 81, contentEnd: 70, wantIDs: "t,o"},
		{name: "row div row", content: `<template id=t><tr id=a><td>x</td></tr><div id=d>m<tr id=b><td>y</template><p id=o>z`, end: 75, contentEnd: 64, wantIDs: "t,o"},
		{name: "cell div cell", content: `<template id=t><td id=a>x</td><div id=d>m<td id=b>y</template><p id=o>z`, end: 62, contentEnd: 51, wantIDs: "t,o"},
	}
	const query = `//*[@id] | //caption | //tbody | //tr | //td | //text()[.='c' or .='x' or .='m' or .='y']`
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := batch24Query(t, test.content, query)
			if got := strings.Join(batch24ResultIDs(results), ","); got != test.wantIDs {
				t.Fatalf("Generic-frame recovery document IDs = %q, want %q (%#v)", got, test.wantIDs, results)
			}
			if results[0].NodeName != "template" || results[0].EndLocation != test.end || results[0].ContentStart != 15 || results[0].ContentEnd != test.contentEnd || results[0].TextContent != "" {
				t.Fatalf("Unexpected generic-frame template host: %#v", results)
			}
		})
	}
}

func TestQueryTemplateForeignFrameAndAFETransitionsRemainIsolated(t *testing.T) {
	tests := []struct {
		name, content   string
		end, contentEnd int
	}{
		{name: "caption MathML row", content: `<template id=t><caption>c</caption><math id=m><mtext id=x>q<tr id=r><td>y</template><p id=o>z`, end: 84, contentEnd: 73},
		{name: "direct row MathML", content: `<template id=t><tr id=a><td>x</td></tr><math id=m><mtext id=n>q<tr id=b><td>y</template><p id=o>z`, end: 88, contentEnd: 77},
		{name: "direct cell MathML", content: `<template id=t><td id=a>x</td><math id=m><mtext id=n>q<td id=b>y</template><p id=o>z`, end: 75, contentEnd: 64},
		{name: "initial frame", content: `<template id=t><frame id=f>x</template><p id=o>z`, end: 39, contentEnd: 28},
		{name: "initial frameset", content: `<template id=t><frameset id=f>x</template><p id=o>z`, end: 42, contentEnd: 31},
		{name: "table frame", content: `<template id=t><caption>c</caption><frame id=f><tr id=r><td>y</template><p id=o>z`, end: 72, contentEnd: 61},
		{name: "table frameset", content: `<template id=t><caption>c</caption><frameset id=f><tr id=r><td>y</template><p id=o>z`, end: 75, contentEnd: 64},
		{name: "AFE b reconstruction", content: `<template id=t><caption>c</caption><b id=b>x<tr id=r><td>y</tr>z</template><p id=o>w`, end: 75, contentEnd: 64},
		{name: "AFE nested reconstruction", content: `<template id=t><caption>c</caption><b id=b><i id=i>x<tr id=r><td>y</tr>z</template><p id=o>w`, end: 83, contentEnd: 72},
	}
	const query = `//*[@id] | //math | //mtext | //frame | //frameset | //caption | //tbody | //tr | //td | //text()[.='c' or .='q' or .='x' or .='y']`
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := batch24Query(t, test.content, query)
			if got := strings.Join(batch24ResultIDs(results), ","); got != "t,o" || len(results) != 2 {
				t.Fatalf("Foreign/frame/AFE TemplateContent leaked into XPath: %q (%#v)", got, results)
			}
			if results[0].EndLocation != test.end || results[0].ContentStart != 15 || results[0].ContentEnd != test.contentEnd || results[0].TextContent != "" {
				t.Fatalf("Unexpected isolated transition host: %#v", results)
			}
		})
	}
}

func TestQueryTemplateFinalScopeRecoveryRemainsIsolated(t *testing.T) {
	tests := []struct {
		name, content   string
		end, contentEnd int
	}{
		{name: "stale math end", content: `<template id=t><caption>c</caption><math id=m><mtext>q<tr id=a><td>x</tr></math><tr id=b><td>y</template><p id=o>z`, end: 105, contentEnd: 94},
		{name: "stale svg end", content: `<template id=t><caption>c</caption><svg id=s><foreignObject>q<tr id=a><td>x</tr></svg><tr id=b><td>y</template><p id=o>z`, end: 111, contentEnd: 100},
		{name: "absent div end", content: `<template id=t><td id=a>x</div><td id=b>y</template><p id=o>z`, end: 52, contentEnd: 41},
		{name: "absent svg end", content: `<template id=t><td id=a>x</svg><td id=b>y</template><p id=o>z`, end: 52, contentEnd: 41},
		{name: "cell ignores row end", content: `<template id=t><td id=a>x</tr><td id=b>y</template><p id=o>z`, end: 51, contentEnd: 40},
		{name: "cell ignores table end", content: `<template id=t><td id=a>x</table><td id=b>y</template><p id=o>z`, end: 54, contentEnd: 43},
		{name: "actual row closes", content: `<template id=t><tr id=a><td>x</tr><tr id=b><td>y</template><p id=o>z`, end: 59, contentEnd: 48},
		{name: "recursive caption form", content: `<template id=t><caption>c</caption><div id=d>a<form id=f>x</form><tr id=r><td>y</template><p id=o>z`, end: 90, contentEnd: 79},
		{name: "recursive row form", content: `<template id=t><tr id=a><td>q</td></tr><div id=d>a<form id=f>x</form><tr id=b><td>y</template><p id=o>z`, end: 94, contentEnd: 83},
		{name: "recursive cell form", content: `<template id=t><td id=a>q</td><div id=d>a<form id=f>x</form><td id=b>y</template><p id=o>z`, end: 81, contentEnd: 70},
		{name: "recursive frame", content: `<template id=t><caption>c</caption><div id=d>a<frame id=f>x<tr id=r><td>y</template><p id=o>z`, end: 84, contentEnd: 73},
		{name: "recursive frameset", content: `<template id=t><caption>c</caption><div id=d>a<frameset id=f>x<tr id=r><td>y</template><p id=o>z`, end: 87, contentEnd: 76},
		{name: "ordinary svg control", content: `<template id=t><caption>c</caption><svg id=s><g id=g>q<tr id=r><td>y</template><p id=o>z`, end: 79, contentEnd: 68},
		{name: "ordinary math control", content: `<template id=t><caption>c</caption><math id=m><mrow id=n>q<tr id=r><td>y</template><p id=o>z`, end: 83, contentEnd: 72},
	}
	const query = `//*[@id] | //math | //mrow | //mtext | //svg | //g | //foreignObject | //frame | //frameset | //form | //caption | //tbody | //tr | //td | //text()[.='q' or .='x' or .='y']`
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := batch24Query(t, test.content, query)
			if got := strings.Join(batch24ResultIDs(results), ","); got != "t,o" || len(results) != 2 {
				t.Fatalf("Final scope recovery leaked TemplateContent into XPath: %q (%#v)", got, results)
			}
			if results[0].EndLocation != test.end || results[0].ContentStart != 15 || results[0].ContentEnd != test.contentEnd || results[0].TextContent != "" {
				t.Fatalf("Unexpected final scope-recovery host: %#v", results)
			}
		})
	}
}

func TestQueryTemplateStableEOFHostRangeMatrix(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		end        int
		contentEnd int
	}{
		{name: "text", content: `<template>x`, end: 0, contentEnd: 0},
		{name: "comment then text", content: `<template><!--c-->x`, end: 0, contentEnd: 0},
		{name: "br then text", content: `<template><br>x`, end: 10, contentEnd: 10},
		{name: "closed div then tail", content: `<template><div>x</div>tail`, end: 16, contentEnd: 16},
		{name: "empty div then text", content: `<template><div></div>x`, end: 15, contentEnd: 15},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := batch24Query(t, test.content, `//template | //comment() | //br | //div | //text()`)
			if len(results) != 1 || results[0].NodeName != "template" || results[0].TextContent != "" || results[0].StartLocation != 0 || results[0].EndLocation != test.end || results[0].ContentStart != 10 || results[0].ContentEnd != test.contentEnd {
				t.Fatalf("EOF fragment nodes leaked or host range diverged from parse5: %#v", results)
			}
		})
	}
}
