package utils

import (
	"reflect"
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 26 covers the WHATWG parser scripting flag only for whole-document
// <noscript> parsing. Fragment parsing and script execution remain out of scope.

func batch26SetScriptingEnabled(t *testing.T, parser *HTMLParser, enabled bool) {
	t.Helper()
	method := reflect.ValueOf(parser).MethodByName("SetScriptingEnabled")
	if !method.IsValid() {
		t.Fatal("HTMLParser must expose SetScriptingEnabled(bool)")
	}
	methodType := method.Type()
	if methodType.NumIn() != 1 || methodType.In(0).Kind() != reflect.Bool {
		t.Fatalf("SetScriptingEnabled must accept one bool, got %s", methodType)
	}
	method.Call([]reflect.Value{reflect.ValueOf(enabled)})
}

func batch26ElementByID(node *types.Node, id string) *types.Node {
	if node.Type == types.ElementNode && node.Attributes["id"] == id {
		return node
	}
	for _, child := range node.Children {
		if found := batch26ElementByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

func batch26RequireElementByID(t *testing.T, document *types.Node, id string) *types.Node {
	t.Helper()
	if found := batch26ElementByID(document, id); found != nil {
		return found
	}
	t.Fatalf("expected element id=%q", id)
	return nil
}

func TestParseNoscriptDefaultsToScriptingDisabledLikeJSDOM(t *testing.T) {
	const content = `<html><head></head><body><noscript id=n><b id=b>x&amp;y</b></noscript><p id=p>z</p></body></html>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}

	noscript := batch26RequireElementByID(t, document, "n")
	bold := batch26RequireElementByID(t, document, "b")
	paragraph := batch26RequireElementByID(t, document, "p")
	if noscript.Parent == nil || noscript.Parent.Name != "body" || len(noscript.Children) != 1 || noscript.Children[0] != bold {
		t.Fatalf("disabled body noscript must parse transparent markup: %#v", noscript)
	}
	if bold.TextContent != "x&y" || bold.StartPos != strings.Index(content, `<b id=b>`) || bold.EndPos != strings.Index(content, `</b>`)+len(`</b>`) {
		t.Fatalf("disabled body noscript child/source mismatch: %#v", bold)
	}
	if paragraph.Parent != noscript.Parent || paragraph.TextContent != "z" {
		t.Fatalf("parsing did not resume after disabled noscript: %#v", paragraph)
	}
}

func TestParseHeadNoscriptDisabledUsesInHeadNoscriptRules(t *testing.T) {
	const content = `<html><head><noscript id=n><link id=l href=x><style id=s>.a{}</style></noscript></head><body><p id=p>z</p></body></html>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}

	noscript := batch26RequireElementByID(t, document, "n")
	link := batch26RequireElementByID(t, document, "l")
	style := batch26RequireElementByID(t, document, "s")
	if noscript.Parent == nil || noscript.Parent.Name != "head" || len(noscript.Children) != 2 || noscript.Children[0] != link || noscript.Children[1] != style {
		t.Fatalf("disabled head noscript allow-list tree mismatch: %#v", noscript)
	}
	if link.Parent != noscript || style.Parent != noscript || style.TextContent != ".a{}" {
		t.Fatalf("disabled head noscript children mismatch: link=%#v style=%#v", link, style)
	}
	if paragraph := batch26RequireElementByID(t, document, "p"); paragraph.Parent == nil || paragraph.Parent.Name != "body" {
		t.Fatalf("body placement changed after disabled head noscript: %#v", paragraph)
	}
}

func TestParseHeadNoscriptDisabledRecoversUnsupportedContent(t *testing.T) {
	const content = `<html><head><noscript id=n><meta id=m><div id=d>x</div></noscript><title id=t>y</title></head><body><p id=p>z</p></body></html>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}

	noscript := batch26RequireElementByID(t, document, "n")
	meta := batch26RequireElementByID(t, document, "m")
	if noscript.Parent == nil || noscript.Parent.Name != "head" || len(noscript.Children) != 1 || noscript.Children[0] != meta {
		t.Fatalf("in-head-noscript must retain only accepted prefix content: %#v", noscript)
	}
	for _, id := range []string{"d", "t", "p"} {
		node := batch26RequireElementByID(t, document, id)
		if node.Parent == nil || node.Parent.Name != "body" {
			t.Fatalf("unsupported head-noscript recovery must reprocess id=%q in body: %#v", id, node)
		}
	}
}

func TestParseNoscriptWithScriptingEnabledUsesRawText(t *testing.T) {
	const content = `<html><head></head><body><noscript id=n><b id=b>x&amp;y</b></noscript><p id=p>z</p></body></html>`
	parser := NewHTMLParser()
	batch26SetScriptingEnabled(t, parser, true)
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatal(err)
	}

	noscript := batch26RequireElementByID(t, document, "n")
	if batch26ElementByID(document, "b") != nil {
		t.Fatal("enabled body noscript must not emit descendant markup")
	}
	rawStart := strings.Index(content, `<b id=b>`)
	rawEnd := strings.Index(content, `</noscript>`)
	wantRaw := content[rawStart:rawEnd]
	if noscript.Parent == nil || noscript.Parent.Name != "body" || noscript.TextContent != wantRaw || len(noscript.Children) != 1 {
		t.Fatalf("enabled body noscript must contain one literal text node: %#v", noscript)
	}
	text := noscript.Children[0]
	if text.Type != types.TextNode || text.Value != wantRaw || text.StartPos != rawStart || text.EndPos != rawEnd {
		t.Fatalf("enabled body noscript raw text/source mismatch: %#v", text)
	}
	if paragraph := batch26RequireElementByID(t, document, "p"); paragraph.Parent != noscript.Parent || paragraph.TextContent != "z" {
		t.Fatalf("parsing did not resume after enabled noscript: %#v", paragraph)
	}
}

func TestParseHeadNoscriptWithScriptingEnabledUsesRawText(t *testing.T) {
	const content = `<html><head><noscript id=n><link id=l href=x><style id=s>.a{}</style></noscript></head><body><p id=p>z</p></body></html>`
	parser := NewHTMLParser()
	batch26SetScriptingEnabled(t, parser, true)
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatal(err)
	}

	noscript := batch26RequireElementByID(t, document, "n")
	if noscript.Parent == nil || noscript.Parent.Name != "head" || batch26ElementByID(document, "l") != nil || batch26ElementByID(document, "s") != nil {
		t.Fatalf("enabled head noscript must remain raw text in head: %#v", noscript)
	}
	rawStart := strings.Index(content, `<link`)
	rawEnd := strings.Index(content, `</noscript>`)
	if len(noscript.Children) != 1 || noscript.Children[0].Type != types.TextNode || noscript.Children[0].Value != content[rawStart:rawEnd] || noscript.Children[0].StartPos != rawStart || noscript.Children[0].EndPos != rawEnd {
		t.Fatalf("enabled head noscript raw text/source mismatch: %#v", noscript.Children)
	}
}

func TestHTMLParserNoscriptScriptingModePersistsAcrossReuse(t *testing.T) {
	parser := NewHTMLParser()
	batch26SetScriptingEnabled(t, parser, true)
	first, err := parser.Parse(`<html><head></head><body><noscript><b id=first>x</b></noscript></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	if batch26ElementByID(first, "first") != nil {
		t.Fatal("enabled mode was not retained for the first parse")
	}

	batch26SetScriptingEnabled(t, parser, false)
	second, err := parser.Parse(`<html><head></head><body><noscript><b id=second>y</b></noscript></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	if bold := batch26ElementByID(second, "second"); bold == nil || bold.TextContent != "y" {
		t.Fatalf("disabled mode was not applied on parser reuse: %#v", bold)
	}
}
