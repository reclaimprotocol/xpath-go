package utils

import (
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

type productionFormattingMismatchNode struct {
	name                     string
	text                     string
	parent                   string
	start, end               int
	contentStart, contentEnd int
}

func productionFormattingMismatchElements(document *types.Node) []*types.Node {
	var elements []*types.Node
	var visit func(*types.Node)
	visit = func(node *types.Node) {
		if node.Type == types.ElementNode && node.Name != "html" && node.Name != "head" && node.Name != "body" {
			elements = append(elements, node)
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(document)
	return elements
}

func TestParseProductionFormattingMismatchFamilies(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		want    []productionFormattingMismatchNode
	}{
		{
			name:    "b/span",
			content: `<span><b>x</span>y</b>`,
			want: []productionFormattingMismatchNode{
				{name: "span", text: "x", parent: "body", start: 0, end: 17, contentStart: 6, contentEnd: 10},
				{name: "b", text: "x", parent: "span", start: 6, end: 10, contentStart: 9, contentEnd: 10},
				{name: "b", text: "y", parent: "body", start: 6, end: 22, contentStart: 9, contentEnd: 18},
			},
		},
		{
			name:    "a/li",
			content: `<ul><li><a>x</li>y</a></ul>`,
			want: []productionFormattingMismatchNode{
				{name: "ul", text: "xy", parent: "body", start: 0, end: 27, contentStart: 4, contentEnd: 22},
				{name: "li", text: "x", parent: "ul", start: 4, end: 17, contentStart: 8, contentEnd: 12},
				{name: "a", text: "x", parent: "li", start: 8, end: 12, contentStart: 11, contentEnd: 12},
				{name: "a", text: "y", parent: "ul", start: 8, end: 22, contentStart: 11, contentEnd: 18},
			},
		},
		{
			name:    "button/a",
			content: `<a><button>x</a>y</button>`,
			want: []productionFormattingMismatchNode{
				{name: "a", text: "", parent: "body", start: 0, end: 16, contentStart: 3, contentEnd: 12},
				{name: "button", text: "xy", parent: "body", start: 3, end: 26, contentStart: 11, contentEnd: 17},
				{name: "a", text: "x", parent: "button", start: 0, end: 0, contentStart: 0, contentEnd: 0},
			},
		},
		{
			name:    "span/small",
			content: `<small><span>x</small>y</span>`,
			want: []productionFormattingMismatchNode{
				{name: "small", text: "x", parent: "body", start: 0, end: 22, contentStart: 7, contentEnd: 14},
				{name: "span", text: "x", parent: "small", start: 7, end: 14, contentStart: 13, contentEnd: 14},
			},
		},
		{
			name:    "div/a",
			content: `<a><div>x</a>y</div>`,
			want: []productionFormattingMismatchNode{
				{name: "a", text: "", parent: "body", start: 0, end: 13, contentStart: 3, contentEnd: 9},
				{name: "div", text: "xy", parent: "body", start: 3, end: 20, contentStart: 8, contentEnd: 14},
				{name: "a", text: "x", parent: "div", start: 0, end: 0, contentStart: 0, contentEnd: 0},
			},
		},
		{
			name:    "font/b",
			content: `<b><font>x</b>y</font>`,
			want: []productionFormattingMismatchNode{
				{name: "b", text: "x", parent: "body", start: 0, end: 14, contentStart: 3, contentEnd: 10},
				{name: "font", text: "x", parent: "b", start: 3, end: 10, contentStart: 9, contentEnd: 10},
				{name: "font", text: "y", parent: "body", start: 3, end: 22, contentStart: 9, contentEnd: 15},
			},
		},
	}

	parser := NewHTMLParser()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Production mismatch %s must recover: %v", testCase.name, err)
			}
			elements := productionFormattingMismatchElements(document)
			if len(elements) != len(testCase.want) {
				t.Fatalf("Element count mismatch: got %#v", elements)
			}
			for index, want := range testCase.want {
				got := elements[index]
				parent := ""
				if got.Parent != nil {
					parent = got.Parent.Name
				}
				if got.Name != want.name || got.TextContent != want.text || parent != want.parent || got.StartPos != want.start || got.EndPos != want.end || got.ContentStart != want.contentStart || got.ContentEnd != want.contentEnd {
					t.Fatalf("Node %d: want %#v, got <%s> text %q parent %s range %d:%d content %d:%d", index, want, got.Name, got.TextContent, parent, got.StartPos, got.EndPos, got.ContentStart, got.ContentEnd)
				}
			}
		})
	}

	clean, err := parser.Parse(`<div id=clean>clean</div>`)
	if err != nil {
		t.Fatalf("Clean reused parse failed: %v", err)
	}
	elements := productionFormattingMismatchElements(clean)
	if len(elements) != 1 || elements[0].Name != "div" || elements[0].Attributes["id"] != "clean" || elements[0].TextContent != "clean" {
		t.Fatalf("Formatting mismatch state leaked through parser reuse: %#v", elements)
	}
}

func TestParseProductionFormattingMismatchScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping production formatting mismatch scaling regression in short mode")
	}
	build := func(n int) string {
		return strings.Repeat(`<span><b>x</span>y</b><b><font>x</b>y</font>`, n)
	}
	_ = productionRecoveryDuration(t, build(50))
	small := productionRecoveryDuration(t, build(250))
	large := productionRecoveryDuration(t, build(1000))
	ratio := float64(large) / float64(small)
	t.Logf("production formatting mismatch scaling 250=%v 1000=%v ratio=%.1fx", small, large, ratio)
	if ratio > 14 && large-small > 150_000_000 {
		t.Fatalf("Production formatting mismatch recovery scaled quadratically: %.1fx (%v -> %v)", ratio, small, large)
	}
}
