package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func batch26ParseWithScripting(t *testing.T, content string, enabled bool) *types.Node {
	t.Helper()
	parser := NewHTMLParser()
	parser.SetScriptingEnabled(enabled)
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func TestParseEnabledNoscriptAppropriateEndTagRecovery(t *testing.T) {
	t.Run("mixed case closes", func(t *testing.T) {
		const content = `<html><head></head><body><noscript id=n>x</NoScRiPt><p id=p>y</p></body></html>`
		document := batch26ParseWithScripting(t, content, true)
		noscript := batch26RequireElementByID(t, document, "n")
		if noscript.TextContent != "x" || noscript.EndPos != strings.Index(content, `<p`) {
			t.Fatalf("mixed-case appropriate end tag did not close noscript: %#v", noscript)
		}
		if paragraph := batch26RequireElementByID(t, document, "p"); paragraph.Parent != noscript.Parent || paragraph.TextContent != "y" {
			t.Fatalf("parsing did not resume after mixed-case end tag: %#v", paragraph)
		}
	})

	t.Run("nonmatching name remains literal", func(t *testing.T) {
		const content = `<html><head></head><body><noscript id=n>a</noscript-x>b</noscript><p id=p>c</p></body></html>`
		document := batch26ParseWithScripting(t, content, true)
		noscript := batch26RequireElementByID(t, document, "n")
		if noscript.TextContent != `a</noscript-x>b` || len(noscript.Children) != 1 || noscript.Children[0].Value != `a</noscript-x>b` {
			t.Fatalf("nonmatching noscript end candidate was not literal RAWTEXT: %#v", noscript)
		}
	})
}

func TestParseEnabledNoscriptRawTextLiteralNormalizationAndCoordinates(t *testing.T) {
	const content = "<html><head></head><body><noscript id=n>é🙂\r\n&amp;&#65;\x00z</NOSCRIPT><p id=p>y</p></body></html>"
	document := batch26ParseWithScripting(t, content, true)
	noscript := batch26RequireElementByID(t, document, "n")
	rawStart := strings.Index(content, "é")
	rawEnd := strings.Index(content, `</NOSCRIPT>`)
	const want = "é🙂\n&amp;&#65;\uFFFDz"
	if noscript.TextContent != want || len(noscript.Children) != 1 {
		t.Fatalf("noscript RAWTEXT normalization/literal references mismatch: %#v", noscript)
	}
	text := noscript.Children[0]
	if text.Type != types.TextNode || text.Value != want || text.StartPos != rawStart || text.EndPos != rawEnd {
		t.Fatalf("noscript RAWTEXT raw byte range mismatch: %#v", text)
	}
	if text.StartLine != 1 || text.StartColumn != rawStart+1 || text.EndLine != 2 || text.EndColumn != 13 {
		t.Fatalf("noscript RAWTEXT UTF-16/CRLF coordinates mismatch: %#v", text)
	}
	if source := content[text.StartPos:text.EndPos]; source != "é🙂\r\n&amp;&#65;\x00z" {
		t.Fatalf("noscript raw source changed: %q", source)
	}
}

func TestParseEnabledNoscriptEOFAndIncompleteEndCandidates(t *testing.T) {
	tests := []struct {
		name, suffix, want string
	}{
		{name: "plain EOF", suffix: `x`, want: `x`},
		{name: "name without delimiter stays literal", suffix: `x</noscript`, want: `x</noscript`},
		{name: "incomplete delimited end is discarded", suffix: `x</noscript `, want: `x`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content := `<html><head></head><body><noscript id=n>` + test.suffix
			document := batch26ParseWithScripting(t, content, true)
			noscript := batch26RequireElementByID(t, document, "n")
			if noscript.TextContent != test.want || noscript.EndPos != len(content) || noscript.ContentEnd != len(content) {
				t.Fatalf("enabled noscript EOF recovery mismatch: %#v", noscript)
			}
			if len(noscript.Children) != 1 || noscript.Children[0].Value != test.want || noscript.Children[0].EndPos != len(content) {
				t.Fatalf("enabled noscript EOF text/source mismatch: %#v", noscript.Children)
			}
		})
	}
}

func TestParseRootNoscriptUsesImplicitHeadLikeExplicitHead(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		mode := "disabled"
		if enabled {
			mode = "enabled"
		}
		for _, test := range []struct {
			name, content string
		}{
			{name: "implicit root", content: `<noscript id=n><meta id=m></noscript><p id=p>x</p>`},
			{name: "explicit head", content: `<html><head><noscript id=n><meta id=m></noscript></head><body><p id=p>x</p></body></html>`},
		} {
			t.Run(mode+"/"+test.name, func(t *testing.T) {
				document := batch26ParseWithScripting(t, test.content, enabled)
				noscript := batch26RequireElementByID(t, document, "n")
				if noscript.Parent == nil || noscript.Parent.Name != "head" {
					t.Fatalf("document-start noscript must be routed to head: %#v", noscript)
				}
				meta := batch26ElementByID(document, "m")
				if enabled {
					if meta != nil || len(noscript.Children) != 1 || noscript.Children[0].Type != types.TextNode || noscript.Children[0].Value != `<meta id=m>` {
						t.Fatalf("enabled head noscript must retain meta source as RAWTEXT: %#v", noscript)
					}
				} else if meta == nil || meta.Parent != noscript {
					t.Fatalf("disabled head noscript must parse meta: noscript=%#v meta=%#v", noscript, meta)
				}
				if paragraph := batch26RequireElementByID(t, document, "p"); paragraph.Parent == nil || paragraph.Parent.Name != "body" {
					t.Fatalf("following content must start body: %#v", paragraph)
				}
			})
		}
	}
}

func TestParseDisabledHeadNoscriptMergesHTMLStartIntoDocumentRoot(t *testing.T) {
	const content = `<html id=root><head><noscript id=n><html lang=en><meta id=m></noscript><title id=t>x</title></head><body><p id=p>y</p></body></html>`
	document := batch26ParseWithScripting(t, content, false)
	html := requireOneDocumentSkeletonElement(t, document, "html")
	noscript := batch26RequireElementByID(t, document, "n")
	meta := batch26RequireElementByID(t, document, "m")
	title := batch26RequireElementByID(t, document, "t")
	if html.Attributes["id"] != "root" || html.Attributes["lang"] != "en" {
		t.Fatalf("html start inside disabled head noscript must merge missing root attributes: %#v", html.Attributes)
	}
	if noscript.Parent == nil || noscript.Parent.Name != "head" || meta.Parent != noscript {
		t.Fatalf("html merge must not disturb disabled noscript insertion mode: noscript=%#v meta=%#v", noscript, meta)
	}
	if title.Parent != noscript.Parent || title.TextContent != "x" {
		t.Fatalf("title after disabled noscript must remain in head: %#v", title)
	}
}

func TestParseNoscriptTableModeContrastsByScriptingFlag(t *testing.T) {
	const content = `<html><head></head><body><div id=before></div><table id=t><noscript id=n><tr id=r><td id=c>x</td></tr></noscript><tr id=q><td>y</td></tr></table></body></html>`

	t.Run("disabled", func(t *testing.T) {
		disabled := batch26ParseWithScripting(t, content, false)
		disabledNoscript := batch26RequireElementByID(t, disabled, "n")
		disabledTable := batch26RequireElementByID(t, disabled, "t")
		disabledRow := batch26RequireElementByID(t, disabled, "r")
		if disabledNoscript.Parent != disabledTable.Parent || disabledNoscript.TextContent != "" || disabledRow.Parent == nil || disabledRow.Parent.Name != "tbody" || disabledRow.Parent.Parent != disabledTable {
			t.Fatalf("disabled table noscript foster/tree recovery mismatch: noscript=%#v row=%#v", disabledNoscript, disabledRow)
		}
	})

	t.Run("enabled", func(t *testing.T) {
		enabled := batch26ParseWithScripting(t, content, true)
		enabledNoscript := batch26RequireElementByID(t, enabled, "n")
		enabledTable := batch26RequireElementByID(t, enabled, "t")
		if enabledNoscript.Parent != enabledTable.Parent || enabledNoscript.TextContent != `<tr id=r><td id=c>x</td></tr>` || batch26ElementByID(enabled, "r") != nil || batch26ElementByID(enabled, "c") != nil {
			t.Fatalf("enabled table noscript must foster one RAWTEXT element: %#v", enabledNoscript)
		}
		if row := batch26RequireElementByID(t, enabled, "q"); row.Parent == nil || row.Parent.Name != "tbody" || row.Parent.Parent != enabledTable {
			t.Fatalf("table parsing did not resume after enabled noscript: %#v", row)
		}
	})
}

func TestParseNoscriptTokensAreIgnoredBySelectMode(t *testing.T) {
	const content = `<html><head></head><body><select id=s><noscript id=n><option id=o>x</option></noscript><option id=p>y</option></select></body></html>`
	for _, enabled := range []bool{false, true} {
		document := batch26ParseWithScripting(t, content, enabled)
		selectNode := batch26RequireElementByID(t, document, "s")
		noscript := batch26ElementByID(document, "n")
		if noscript != nil || len(selectNode.Children) != 2 || selectNode.Children[0].Attributes["id"] != "o" || selectNode.Children[1].Attributes["id"] != "p" || selectNode.TextContent != "xy" {
			t.Fatalf("select mode must ignore noscript tags in scripting=%v: select=%#v noscript=%#v", enabled, selectNode, noscript)
		}
	}
}

func TestParseTemplateNoscriptContrastsByScriptingFlag(t *testing.T) {
	const content = `<template id=t><noscript id=n><b id=b>x&amp;y</b></noscript><i id=i>z</i></template><p id=p>q</p>`
	for _, enabled := range []bool{false, true} {
		document := batch26ParseWithScripting(t, content, enabled)
		template := batch26RequireElementByID(t, document, "t")
		fragment := template.TemplateContent
		if fragment == nil || len(fragment.Children) != 2 {
			t.Fatalf("template content missing in scripting=%v: %#v", enabled, template)
		}
		noscript := batch26RequireElementByID(t, fragment, "n")
		if enabled {
			if batch26ElementByID(fragment, "b") != nil || len(noscript.Children) != 1 || noscript.Children[0].Type != types.TextNode || noscript.Children[0].Value != `<b id=b>x&amp;y</b>` {
				t.Fatalf("enabled template noscript RAWTEXT mismatch: %#v", noscript)
			}
		} else if bold := batch26ElementByID(fragment, "b"); bold == nil || bold.Parent != noscript || bold.TextContent != "x&y" {
			t.Fatalf("disabled template noscript markup mismatch: noscript=%#v bold=%#v", noscript, bold)
		}
		if italic := batch26RequireElementByID(t, fragment, "i"); italic.Parent != fragment || italic.TextContent != "z" {
			t.Fatalf("template parsing did not resume after noscript: %#v", italic)
		}
	}
}

func TestParseForeignNoscriptIsUnaffectedByHTMLScriptingFlag(t *testing.T) {
	const content = `<html><head></head><body><svg id=s><noscript id=n><g id=g><text id=t>x</text></g></noscript></svg><p id=p>y</p></body></html>`
	const svgNamespace = "http://www.w3.org/2000/svg"
	for _, enabled := range []bool{false, true} {
		document := batch26ParseWithScripting(t, content, enabled)
		svg := batch26RequireElementByID(t, document, "s")
		noscript := batch26RequireElementByID(t, document, "n")
		group := batch26RequireElementByID(t, document, "g")
		text := batch26RequireElementByID(t, document, "t")
		if svg.NamespaceURI != svgNamespace || noscript.NamespaceURI != svgNamespace || group.NamespaceURI != svgNamespace || text.NamespaceURI != svgNamespace || noscript.Parent != svg || group.Parent != noscript || text.Parent != group || noscript.TextContent != "x" {
			t.Fatalf("foreign noscript changed in scripting=%v: svg=%#v noscript=%#v", enabled, svg, noscript)
		}
	}
}

func TestParseManyEnabledNoscriptsPreservesLastSourceLocation(t *testing.T) {
	const count = 256
	const token = `<noscript>x</noscript>`
	const prefix = `<html><head></head><body>`
	const suffix = `</body></html>`
	content := prefix + strings.Repeat(token, count) + suffix
	document := batch26ParseWithScripting(t, content, true)
	body := requireOneDocumentSkeletonElement(t, document, "body")
	if len(body.Children) != count {
		t.Fatalf("expected %d noscripts, got %d", count, len(body.Children))
	}
	last := body.Children[count-1]
	wantStart := len(prefix) + (count-1)*len(token)
	wantTextStart := wantStart + len(`<noscript>`)
	if last.Name != "noscript" || last.StartPos != wantStart || last.EndPos != wantStart+len(token) || last.TextContent != "x" || len(last.Children) != 1 || last.Children[0].StartPos != wantTextStart || last.Children[0].EndPos != wantTextStart+1 {
		t.Fatalf("last enabled noscript/source location mismatch: %#v", last)
	}
}

func TestParseManyEnabledNoscriptsScalesNearLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping parser scaling regression in short mode")
	}
	measure := func(count int) time.Duration {
		content := `<html><head></head><body>` + strings.Repeat(`<noscript>x</noscript>`, count) + `</body></html>`
		started := time.Now()
		document := batch26ParseWithScripting(t, content, true)
		elapsed := time.Since(started)
		body := requireOneDocumentSkeletonElement(t, document, "body")
		if len(body.Children) != count {
			t.Fatalf("Parse(%d noscripts) returned %d body children", count, len(body.Children))
		}
		return elapsed
	}
	_ = measure(128)
	minOfTwo := func(count int) time.Duration {
		first, second := measure(count), measure(count)
		if first < second {
			return first
		}
		return second
	}
	small, large := minOfTwo(1000), minOfTwo(4000)
	t.Logf("parsed 1000 enabled noscripts in %s and 4000 in %s", small, large)
	if large > 10*small && large-small > 50*time.Millisecond {
		t.Fatalf("enabled noscript parsing scales quadratically: 4x input took %.1fx longer (%s -> %s)", float64(large)/float64(small), small, large)
	}
}
