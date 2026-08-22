package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseSVGIntegrationCommentsAndContinuation(t *testing.T) {
	const content = `<svg><desc><!--c--><span>x</span>y</desc>z</svg>q`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodesNamed(document, "svg")[0]
	desc := svgNodesNamed(document, "desc")[0]
	span := svgNodesNamed(document, "span")[0]
	assertFormattingNode(t, svg, "svg", "xyz", 0, 48)
	assertFormattingNode(t, desc, "desc", "xy", 5, 41)
	assertFormattingNode(t, span, "span", "x", 19, 33)
	if len(desc.Children) != 3 || desc.Children[0].Type != types.CommentNode || desc.Children[0].Value != "c" || desc.Children[0].StartPos != 11 || desc.Children[0].EndPos != 19 || desc.Children[2].Value != "y" || desc.Children[2].StartPos != 33 || len(svg.Children) != 2 || svg.Children[1].Value != "z" || parsedBodyChildren(document)[1].Value != "q" {
		t.Fatalf("Integration comment/text ordering mismatch: desc=%#v svg=%#v doc=%#v", desc.Children, svg.Children, parsedBodyChildren(document))
	}
}

func bestSVGIntegrationDuration(t *testing.T, content string) time.Duration {
	t.Helper()
	best := time.Duration(1<<63 - 1)
	for i := 0; i < 3; i++ {
		start := time.Now()
		if _, err := NewHTMLParser().Parse(content); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed < best {
			best = elapsed
		}
	}
	return best
}

func TestParseSVGIntegrationAndBreakoutScaling(t *testing.T) {
	builders := map[string]func(int) string{
		"integration": func(n int) string { return strings.Repeat(`<svg><foreignObject><div>x</div></foreignObject></svg>`, n) },
		"breakout":    func(n int) string { return strings.Repeat(`<svg><g><div>x</div>`, n) },
	}
	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			smallInput, largeInput := build(500), build(2000)
			_ = bestSVGIntegrationDuration(t, smallInput)
			small := bestSVGIntegrationDuration(t, smallInput)
			large := bestSVGIntegrationDuration(t, largeInput)
			if large > 10*small && large-small > 100*time.Millisecond {
				t.Fatalf("SVG %s parsing scaled superlinearly: small=%s large=%s", name, small, large)
			}
		})
	}
}
