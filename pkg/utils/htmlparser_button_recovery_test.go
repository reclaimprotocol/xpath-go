package utils

import (
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func batch25ElementByID(t *testing.T, node *types.Node, id string) *types.Node {
	t.Helper()
	var found *types.Node
	var visit func(*types.Node)
	visit = func(current *types.Node) {
		if current.Type == types.ElementNode && current.Attributes["id"] == id {
			if found != nil {
				t.Fatalf("More than one element has id %q", id)
			}
			found = current
		}
		for _, child := range current.Children {
			visit(child)
		}
	}
	visit(node)
	if found == nil {
		t.Fatalf("No element has id %q", id)
	}
	return found
}

func batch25AssertButtonSiblings(t *testing.T, parent *types.Node, content string) {
	t.Helper()
	if len(parent.Children) != 3 {
		t.Fatalf("Expected button a, button b, and trailing text as siblings, got %#v", parent.Children)
	}
	first, second, tail := parent.Children[0], parent.Children[1], parent.Children[2]
	secondStart := strings.Index(content, `<button id=b>`)
	secondEnd := strings.Index(content, `</button>`) + len(`</button>`)
	if first.Name != "button" || first.Attributes["id"] != "a" || first.TextContent != "x" || first.Parent != parent || first.StartPos != strings.Index(content, `<button id=a>`) || first.EndPos != secondStart || first.ContentStart != secondStart-len("x") || first.ContentEnd != secondStart {
		t.Fatalf("Implicitly closed first button range/tree mismatch: %#v", first)
	}
	if second.Name != "button" || second.Attributes["id"] != "b" || second.TextContent != "y" || second.Parent != parent || second.StartPos != secondStart || second.EndPos != secondEnd || second.ContentStart != secondStart+len(`<button id=b>`) || second.ContentEnd != secondEnd-len(`</button>`) {
		t.Fatalf("Explicitly closed second button range/tree mismatch: %#v", second)
	}
	if tail.Type != types.TextNode || tail.Value != "z" || tail.Parent != parent || tail.StartPos != secondEnd || tail.EndPos != secondEnd+1 {
		t.Fatalf("Trailing text after second button mismatch: %#v", tail)
	}
}

func TestParseNestedButtonStartClosesButtonInScopeLikeBrowser(t *testing.T) {
	const content = `<button id=a>x<button id=b>y</button>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Nested button start must recover: %v", err)
	}
	_, _, body := requireImplicitDocumentSkeleton(t, document)
	batch25AssertButtonSiblings(t, body, content)
}

func TestParseNestedButtonGeneratesImpliedEndsBeforeClosingButton(t *testing.T) {
	const content = `<button id=a><p>x<button id=b>y</button>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	_, _, body := requireImplicitDocumentSkeleton(t, document)
	if len(body.Children) != 3 {
		t.Fatalf("Expected two button siblings and trailing text, got %#v", body.Children)
	}
	first, second := body.Children[0], body.Children[1]
	if first.Name != "button" || first.Attributes["id"] != "a" || first.StartPos != 0 || first.EndPos != 17 || len(first.Children) != 1 {
		t.Fatalf("First button implied-end boundary mismatch: %#v", first)
	}
	paragraph := first.Children[0]
	if paragraph.Name != "p" || paragraph.TextContent != "x" || paragraph.StartPos != 13 || paragraph.EndPos != 17 || paragraph.ContentStart != 16 || paragraph.ContentEnd != 17 {
		t.Fatalf("Paragraph was not implied-ended with the first button: %#v", paragraph)
	}
	if second.Name != "button" || second.Attributes["id"] != "b" || second.TextContent != "y" || second.StartPos != 17 || second.EndPos != 40 || body.Children[2].Value != "z" || body.Children[2].StartPos != 40 || body.Children[2].EndPos != 41 {
		t.Fatalf("Second button/tail mismatch after implied ends: %#v", body.Children)
	}
}

func TestParseNestedButtonReconstructsFormattingFromInsideOldButton(t *testing.T) {
	const content = `<button id=a><b>x<button id=b>y</button>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	_, _, body := requireImplicitDocumentSkeleton(t, document)
	buttons := findAllElementsByName(document, "button")
	bold := findAllElementsByName(document, "b")
	if len(buttons) != 2 || len(bold) != 2 || len(body.Children) != 2 || body.Children[0] != buttons[0] || body.Children[1] != bold[1] {
		t.Fatalf("Expected old button then reconstructed b at body level: body=%#v buttons=%#v bold=%#v", body.Children, buttons, bold)
	}
	if buttons[0].Attributes["id"] != "a" || buttons[0].TextContent != "x" || buttons[0].StartPos != 0 || buttons[0].EndPos != 17 || len(buttons[0].Children) != 1 || buttons[0].Children[0] != bold[0] {
		t.Fatalf("Old button/source formatting mismatch: %#v", buttons[0])
	}
	if bold[0].TextContent != "x" || bold[0].StartPos != 13 || bold[0].EndPos != 17 || bold[0].ContentStart != 16 || bold[0].ContentEnd != 17 {
		t.Fatalf("Source formatting range mismatch: %#v", bold[0])
	}
	if bold[1].TextContent != "yz" || bold[1].StartPos != 13 || bold[1].EndPos != 41 || bold[1].ContentStart != 16 || bold[1].ContentEnd != 41 || len(bold[1].Children) != 2 || bold[1].Children[0] != buttons[1] || bold[1].Children[1].Value != "z" {
		t.Fatalf("Reconstructed formatting range/tree mismatch: %#v", bold[1])
	}
	if buttons[1].Attributes["id"] != "b" || buttons[1].TextContent != "y" || buttons[1].StartPos != 17 || buttons[1].EndPos != 40 || buttons[1].Parent != bold[1] {
		t.Fatalf("Second button was not inserted in reconstructed formatting: %#v", buttons[1])
	}
}

func TestParseNestedButtonRecoveryAcrossBoundedContexts(t *testing.T) {
	testCases := []struct {
		name, content, parentID string
	}{
		{name: "formatting", content: `<b id=p><button id=a>x<button id=b>y</button>z</b>`, parentID: "p"},
		{name: "anchor", content: `<a id=p><button id=a>x<button id=b>y</button>z</a>`, parentID: "p"},
		{name: "list", content: `<ul><li id=p><button id=a>x<button id=b>y</button>z</li></ul>`, parentID: "p"},
		{name: "form", content: `<form id=p><button id=a>x<button id=b>y</button>z</form>`, parentID: "p"},
		{name: "table cell", content: `<table><tr><td id=p><button id=a>x<button id=b>y</button>z</td></tr></table>`, parentID: "p"},
		{name: "foreign HTML integration point", content: `<svg><foreignObject id=p><button id=a>x<button id=b>y</button>z</foreignObject></svg>`, parentID: "p"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Nested button context must recover: %v", err)
			}
			batch25AssertButtonSiblings(t, batch25ElementByID(t, document, testCase.parentID), testCase.content)
		})
	}
}

func TestParseNestedButtonSelectAndTemplateBoundaries(t *testing.T) {
	t.Run("customizable select contains recovered sibling buttons", func(t *testing.T) {
		const content = `<select id=s><button id=a>x<button id=b>y</button>z</select>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		selectNode := batch25ElementByID(t, document, "s")
		batch25AssertButtonSiblings(t, selectNode, content)
	})

	t.Run("select is a button-scope barrier", func(t *testing.T) {
		const content = `<button id=a>x<select><button id=b>y</button></select>z</button>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		buttons := findAllElementsByName(document, "button")
		selects := findAllElementsByName(document, "select")
		if len(buttons) != 2 || len(selects) != 1 || buttons[0].Attributes["id"] != "a" || buttons[1].Attributes["id"] != "b" || buttons[0].StartPos != 0 || buttons[0].EndPos != len(content) || buttons[0].TextContent != "xyz" || selects[0].Parent != buttons[0] || buttons[1].Parent != selects[0] || buttons[1].StartPos != 22 || buttons[1].EndPos != 45 {
			t.Fatalf("Select scope barrier incorrectly closed outer button: buttons=%#v select=%#v", buttons, selects)
		}
	})

	t.Run("template isolates recovered siblings", func(t *testing.T) {
		const content = `<template id=t><button id=a>x<button id=b>y</button>z</template>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		template := batch24RequireElementByID(t, document, "t")
		fragment := batch24TemplateContent(t, template)
		batch25AssertButtonSiblings(t, fragment, content)
		if len(template.Children) != 0 || len(findAllElementsByName(document, "button")) != 0 {
			t.Fatalf("Recovered template buttons escaped TemplateContent: template=%#v body=%#v", template, parsedBodyChildren(document))
		}
	})
}

func TestParseIncompleteNestedButtonStartDoesNotCloseActiveButton(t *testing.T) {
	const content = `<button id=a>x<button id=b`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	buttons := findAllElementsByName(document, "button")
	if len(buttons) != 1 || buttons[0].Attributes["id"] != "a" || buttons[0].TextContent != "x" || buttons[0].StartPos != 0 || buttons[0].EndPos != len(content) || buttons[0].ContentStart != 13 || buttons[0].ContentEnd != len(content) {
		t.Fatalf("Incomplete second start must be discarded before button-scope recovery: %#v", buttons)
	}
	if len(buttons[0].Children) != 1 || buttons[0].Children[0].Value != "x" || buttons[0].Children[0].StartPos != 13 || buttons[0].Children[0].EndPos != len(content) {
		t.Fatalf("Discarded incomplete button must remain in prior text raw range: %#v", buttons[0].Children)
	}
}

func TestParseNestedButtonRecoveryReuseAndScaling(t *testing.T) {
	parser := NewHTMLParser()
	if _, err := parser.Parse(`<button id=a>x<button id=b>y</button>z`); err != nil {
		t.Fatal(err)
	}
	clean, err := parser.Parse(`<button id=clean>ok</button>`)
	if err != nil {
		t.Fatal(err)
	}
	buttons := findAllElementsByName(clean, "button")
	if len(buttons) != 1 || buttons[0].Attributes["id"] != "clean" || buttons[0].TextContent != "ok" || buttons[0].StartPos != 0 || buttons[0].EndPos != len(`<button id=clean>ok</button>`) {
		t.Fatalf("Button-scope state leaked across parser reuse: %#v", buttons)
	}

	if testing.Short() {
		return
	}
	build := func(n int) string {
		return strings.Repeat(`<button>x<button>y</button>z`, n)
	}
	_ = productionRecoveryDuration(t, build(25))
	small := productionRecoveryDuration(t, build(100))
	large := productionRecoveryDuration(t, build(400))
	ratio := float64(large) / float64(small)
	t.Logf("nested button recovery scaling 100=%v 400=%v ratio=%.1fx", small, large, ratio)
	if ratio > 14 && large-small > 150_000_000 {
		t.Fatalf("Nested button recovery scaled quadratically: %.1fx (%v -> %v)", ratio, small, large)
	}
}
