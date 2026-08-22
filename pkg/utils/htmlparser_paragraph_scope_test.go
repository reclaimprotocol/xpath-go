package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseSpecialAncestorEndTagsUnwindOpenParagraph(t *testing.T) {
	for _, tag := range []string{"form", "li", "dd", "dt", "h1", "h2", "h3", "h4", "h5", "h6", "applet", "marquee"} {
		t.Run(tag, func(t *testing.T) {
			opening, closing := `<`+tag+`>`, `</`+tag+`>`
			content := opening + `<p>x` + closing + `tail`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			if len(parsedBodyChildren(document)) != 2 {
				t.Fatalf("Expected closed ancestor plus tail, got %#v", parsedBodyChildren(document))
			}
			ancestor := parsedBodyChildren(document)[0]
			assertNodeIdentity(t, ancestor, tag, "x", 0, len(content)-4)
			p := findFirstElementByName(ancestor, "p")
			assertNodeIdentity(t, p, "p", "x", len(opening), len(opening)+4)
			tail := parsedBodyChildren(document)[1]
			if tail.Type != types.TextNode || tail.Value != "tail" || tail.StartPos != len(content)-4 || tail.EndPos != len(content) {
				t.Fatalf("Expected tail outside %s at %d:%d, got %#v", tag, len(content)-4, len(content), tail)
			}
		})
	}
}

func findAllTextNodes(node *types.Node) []*types.Node {
	var matches []*types.Node
	if node == nil {
		return matches
	}
	if node.Type == types.TextNode {
		matches = append(matches, node)
	}
	for _, child := range node.Children {
		matches = append(matches, findAllTextNodes(child)...)
	}
	return matches
}

func findAllNodesByType(node *types.Node, nodeType types.NodeType) []*types.Node {
	var matches []*types.Node
	if node == nil {
		return matches
	}
	if node.Type == nodeType {
		matches = append(matches, node)
	}
	for _, child := range node.Children {
		matches = append(matches, findAllNodesByType(child, nodeType)...)
	}
	return matches
}

func findAllElementsByName(node *types.Node, name string) []*types.Node {
	var matches []*types.Node
	if node == nil {
		return matches
	}
	if node.Type == types.ElementNode && node.Name == name {
		matches = append(matches, node)
	}
	for _, child := range node.Children {
		matches = append(matches, findAllElementsByName(child, name)...)
	}
	return matches
}

func TestParseDeepInlineParagraphLookupScalesNearLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping parser scaling regression in short mode")
	}
	build := func(depth int, paragraph bool) string {
		var content strings.Builder
		if paragraph {
			content.WriteString(`<p>`)
		}
		content.WriteString(strings.Repeat(`<span>`, depth))
		content.WriteString(`x`)
		content.WriteString(strings.Repeat(`</span>`, depth))
		if paragraph {
			content.WriteString(`</p>`)
		}
		return content.String()
	}
	measure := func(depth int, paragraph bool) time.Duration {
		content := build(depth, paragraph)
		started := time.Now()
		document, err := NewHTMLParser().Parse(content)
		elapsed := time.Since(started)
		if err != nil || document.TextContent != "" {
			// Document nodes do not aggregate TextContent; only the error matters.
			if err != nil {
				t.Fatalf("Parse depth %d paragraph=%v failed: %v", depth, paragraph, err)
			}
		}
		return elapsed
	}
	minOfTwo := func(depth int, paragraph bool) time.Duration {
		first, second := measure(depth, paragraph), measure(depth, paragraph)
		if first < second {
			return first
		}
		return second
	}

	_ = measure(64, true)
	for _, paragraph := range []bool{false, true} {
		paragraph := paragraph
		name := "without paragraph"
		if paragraph {
			name = "with paragraph"
		}
		t.Run(name, func(t *testing.T) {
			small := minOfTwo(600, paragraph)
			large := minOfTwo(2400, paragraph)
			t.Logf("depth 600 in %s, depth 2400 in %s", small, large)
			if large > 10*small && large-small > 20*time.Millisecond {
				t.Fatalf("Paragraph scope lookup scales quadratically: 4x depth took %.1fx longer", float64(large)/float64(small))
			}
		})
	}
}

func TestParseDeepInlineLongTagInspectionHasBoundedScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping adversarial parser scaling regression in short mode")
	}
	build := func(depth, attributeLength int) string {
		return `<p>` + strings.Repeat(`<span>`, depth) + `x<div data-x="` + strings.Repeat("a", attributeLength) + `">y</div>`
	}
	measure := func(depth, attributeLength int) time.Duration {
		content := build(depth, attributeLength)
		started := time.Now()
		_, err := NewHTMLParser().Parse(content)
		elapsed := time.Since(started)
		if err != nil {
			t.Fatalf("Parse depth=%d attr=%d failed: %v", depth, attributeLength, err)
		}
		return elapsed
	}
	minOfTwo := func(depth, attributeLength int) time.Duration {
		first, second := measure(depth, attributeLength), measure(depth, attributeLength)
		if first < second {
			return first
		}
		return second
	}
	_ = measure(32, 128)
	small := minOfTwo(250, 12000)
	large := minOfTwo(1000, 48000)
	t.Logf("depth/attribute 250/12k in %s and 1000/48k in %s", small, large)
	if large > 12*small && large-small > 30*time.Millisecond {
		t.Fatalf("Deep paragraph tag inspection scales superlinearly: 4x depth and attribute took %.1fx longer", float64(large)/float64(small))
	}
}
