package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

const repeatedTitleToken = "<title>x</title>"

func TestParseManyTextStateElementsPreservesLastSourceLocation(t *testing.T) {
	const count = 256
	content := strings.Repeat(repeatedTitleToken, count)
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedHeadChildren(document)) != count {
		t.Fatalf("Expected %d title elements in the implicit head, got %d", count, len(parsedHeadChildren(document)))
	}

	last := parsedHeadChildren(document)[count-1]
	if last.Type != types.ElementNode || last.Name != "title" || last.TextContent != "x" || len(last.Children) != 1 {
		t.Fatalf("Expected final title with one x text node, got %#v", last)
	}
	wantElementStart := (count - 1) * len(repeatedTitleToken)
	wantTextStart := wantElementStart + len("<title>")
	wantTextEnd := wantTextStart + 1
	text := last.Children[0]
	if last.StartPos != wantElementStart || last.EndPos != len(content) {
		t.Fatalf("Expected final title at %d:%d, got %d:%d", wantElementStart, len(content), last.StartPos, last.EndPos)
	}
	if text.StartPos != wantTextStart || text.EndPos != wantTextEnd || text.Value != "x" {
		t.Fatalf("Expected final x at %d:%d, got %#v", wantTextStart, wantTextEnd, text)
	}
	if text.StartLine != 1 || text.StartColumn != wantTextStart+1 || text.EndLine != 1 || text.EndColumn != wantTextEnd+1 {
		t.Fatalf("Expected final x coordinates 1:%d-1:%d, got %d:%d-%d:%d", wantTextStart+1, wantTextEnd+1, text.StartLine, text.StartColumn, text.EndLine, text.EndColumn)
	}
}

func TestParseManyTextStateElementsScalesNearLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping parser scaling regression in short mode")
	}

	measure := func(count int) time.Duration {
		content := strings.Repeat(repeatedTitleToken, count)
		started := time.Now()
		document, err := NewHTMLParser().Parse(content)
		elapsed := time.Since(started)
		if err != nil {
			t.Fatalf("Parse(%d titles) failed: %v", count, err)
		}
		if len(parsedHeadChildren(document)) != count {
			t.Fatalf("Parse(%d titles) returned %d head children", count, len(parsedHeadChildren(document)))
		}
		return elapsed
	}

	// Warm caches and the Go runtime before comparing the two workloads.
	_ = measure(128)
	const smallCount = 1000
	const largeCount = 4000
	minOfTwo := func(count int) time.Duration {
		first := measure(count)
		second := measure(count)
		if first < second {
			return first
		}
		return second
	}
	small := minOfTwo(smallCount)
	large := minOfTwo(largeCount)
	t.Logf("parsed %d titles in %s and %d titles in %s", smallCount, small, largeCount, large)

	// The input grows by 4x. Allow substantially more than 4x for noisy CI,
	// but reject the roughly 16x curve caused by rescanning the source from
	// byte zero for every emitted text boundary.
	if large > 10*small && large-small > 50*time.Millisecond {
		t.Fatalf("Text-state parsing scales quadratically: 4x input took %.1fx longer (%s -> %s)", float64(large)/float64(small), small, large)
	}
}

func TestParseLongPlainTextPreservesContentAndLocations(t *testing.T) {
	const repeats = 4096
	const unit = "plain-text-🙂 " // 14 UTF-16 code units.
	content := strings.Repeat(unit, repeats)
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	children := parsedBodyChildren(document)
	if len(children) != 1 || children[0].Type != types.TextNode {
		t.Fatalf("Expected one body text node, got %#v", children)
	}
	text := children[0]
	if text.Value != content || text.TextContent != content {
		t.Fatalf("Long plain-text content was not preserved")
	}
	if text.StartPos != 0 || text.EndPos != len(content) || text.StartLine != 1 || text.StartColumn != 1 || text.EndLine != 1 || text.EndColumn != 1+14*repeats {
		t.Fatalf("Unexpected long text range: %#v", text)
	}
}

func TestParseLongPlainTextScalesNearLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long plain-text scaling regression in short mode")
	}

	measure := func(repeats int) time.Duration {
		content := strings.Repeat("plain-text-0123456789 ", repeats)
		started := time.Now()
		document, err := NewHTMLParser().Parse(content)
		elapsed := time.Since(started)
		if err != nil {
			t.Fatalf("Parse(%d repeats) returned an error: %v", repeats, err)
		}
		children := parsedBodyChildren(document)
		if len(children) != 1 || children[0].Value != content {
			t.Fatalf("Parse(%d repeats) did not preserve its text node", repeats)
		}
		return elapsed
	}

	// The previous append-by-concatenation loop copied the accumulated string
	// for every rune. A 4x input should stay well below that quadratic curve.
	_ = measure(256)
	minOfTwo := func(repeats int) time.Duration {
		first, second := measure(repeats), measure(repeats)
		if first < second {
			return first
		}
		return second
	}
	small, large := minOfTwo(1024), minOfTwo(4096)
	t.Logf("parsed long plain text 1024=%v 4096=%v ratio=%.1fx", small, large, float64(large)/float64(small))
	if large > 10*small && large-small > 50*time.Millisecond {
		t.Fatalf("Long plain-text parsing scaled quadratically: 4x input took %.1fx longer (%s -> %s)", float64(large)/float64(small), small, large)
	}
}
