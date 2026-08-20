package xpath

import (
	"bytes"
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
	"github.com/reclaimprotocol/xpath-go/pkg/utils"
)

func TestSourceMapperHandlesInvalidUTF8AndMultibyteEncodings(t *testing.T) {
	t.Run("coalesces ascii source runs", func(t *testing.T) {
		decoded, mapper, err := decodeForXPath([]byte("prefix-and-suffix"), "")
		if err != nil || decoded != "prefix-and-suffix" {
			t.Fatalf("decoded=%q err=%v", decoded, err)
		}
		if len(mapper.segments) != 1 {
			t.Fatalf("ASCII source should use one affine map segment, got %#v", mapper.segments)
		}
		if got := mapper.rawOffset(len("prefix")); got != len("prefix") {
			t.Fatalf("ASCII offset mapped to raw %d", got)
		}
	})

	t.Run("tolerant utf-8", func(t *testing.T) {
		decoded, mapper, err := decodeForXPath([]byte{'a', 0xff, 'b'}, "")
		if err != nil || decoded != "a\uFFFDb" {
			t.Fatalf("decoded=%q err=%v", decoded, err)
		}
		if got := mapper.rawOffset(len("a\uFFFD")); got != 2 {
			t.Fatalf("replacement boundary mapped to raw %d, want 2", got)
		}
	})

	t.Run("shift jis", func(t *testing.T) {
		decoded, mapper, err := decodeForXPath([]byte{0x82, 0xa0, 'x'}, "shift_jis")
		if err != nil || decoded != "あx" {
			t.Fatalf("decoded=%q err=%v", decoded, err)
		}
		if got := mapper.rawOffset(len("あ")); got != 2 {
			t.Fatalf("Shift_JIS boundary mapped to raw %d, want 2", got)
		}
	})

	t.Run("utf-16 little endian", func(t *testing.T) {
		raw := []byte{'<', 0, 'p', 0, '>', 0, 0xe9, 0, '<', 0, '/', 0, 'p', 0, '>', 0}
		decoded, mapper, err := decodeForXPath(raw, "utf-16le")
		if err != nil || decoded != "<p>é</p>" {
			t.Fatalf("decoded=%q err=%v", decoded, err)
		}
		if got := mapper.rawOffset(len("<p>é")); got != 8 {
			t.Fatalf("UTF-16 boundary mapped to raw %d, want 8", got)
		}
	})

	t.Run("iso-2022-jp state transitions", func(t *testing.T) {
		raw := []byte{'<', 'p', '>', 0x1b, '$', 'B', 0x24, 0x22, 0x1b, '(', 'B', '<', '/', 'p', '>'}
		decoded, mapper, err := decodeForXPath(raw, "iso-2022-jp")
		if err != nil || decoded != "<p>あ</p>" {
			t.Fatalf("decoded=%q err=%v", decoded, err)
		}
		// The shift-back escape emits no Unicode bytes, so the decoded
		// boundary before </p> maps to the next output byte after that escape.
		if got := mapper.rawOffset(len("<p>あ")); got != 11 {
			t.Fatalf("ISO-2022-JP output boundary mapped to raw %d, want 11", got)
		}
	})
}

func TestSourceMapperDoesNotPreallocateFromRawInputLength(t *testing.T) {
	tests := []struct {
		name    string
		charset string
	}{
		{name: "utf-8"},
		{name: "windows-1252", charset: "windows-1252"},
	}

	raw := bytes.Repeat([]byte("a"), 1<<20)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoded, mapper, err := decodeForXPath(raw, test.charset)
			if err != nil || len(decoded) != len(raw) {
				t.Fatalf("decoded length=%d err=%v", len(decoded), err)
			}
			if len(mapper.segments) != 1 {
				t.Fatalf("ASCII source should coalesce to one segment, got %d", len(mapper.segments))
			}
			if capacity := cap(mapper.segments); capacity > 64 {
				t.Fatalf("segment capacity=%d for one affine run; capacity must not scale with the %d-byte input", capacity, len(raw))
			}
		})
	}
}

func TestISO2022JPStateBytesUseBoundarySpecificOffsets(t *testing.T) {
	t.Run("between sibling elements", func(t *testing.T) {
		first := []byte(`<p>a</p>`)
		stateOnly := []byte{0x1b, '$', 'B', 0x1b, '(', 'B'}
		second := []byte(`<div>b</div>`)
		raw := append(append(append([]byte{}, first...), stateOnly...), second...)
		opts := Options{IncludeLocation: true, OutputFormat: "nodes", Charset: "iso-2022-jp"}

		paragraphs, err := QueryBytesWithOptions(`//p`, raw, opts)
		if err != nil || len(paragraphs) != 1 {
			t.Fatalf("paragraph query: results=%#v err=%v", paragraphs, err)
		}
		if paragraphs[0].StartLocation != 0 || paragraphs[0].EndLocation != len(first) {
			t.Fatalf("paragraph range=%d:%d, want 0:%d", paragraphs[0].StartLocation, paragraphs[0].EndLocation, len(first))
		}

		divs, err := QueryBytesWithOptions(`//div`, raw, opts)
		if err != nil || len(divs) != 1 {
			t.Fatalf("div query: results=%#v err=%v", divs, err)
		}
		wantDivStart := len(first) + len(stateOnly)
		if divs[0].StartLocation != wantDivStart || divs[0].EndLocation != len(raw) {
			t.Fatalf("div range=%d:%d, want %d:%d", divs[0].StartLocation, divs[0].EndLocation, wantDivStart, len(raw))
		}
	})

	t.Run("inside element content", func(t *testing.T) {
		shiftToJIS := []byte{0x1b, '$', 'B'}
		encodedText := []byte{0x24, 0x22}
		shiftToASCII := []byte{0x1b, '(', 'B'}
		raw := append([]byte(`<p>`), shiftToJIS...)
		raw = append(raw, encodedText...)
		raw = append(raw, shiftToASCII...)
		raw = append(raw, []byte(`</p>`)...)
		opts := Options{IncludeLocation: true, OutputFormat: "nodes", Charset: "iso-2022-jp"}

		paragraphs, err := QueryBytesWithOptions(`//p`, raw, opts)
		if err != nil || len(paragraphs) != 1 {
			t.Fatalf("paragraph query: results=%#v err=%v", paragraphs, err)
		}
		wantContentStart := len(`<p>`)
		wantContentEnd := len(raw) - len(`</p>`)
		if paragraphs[0].ContentStart != wantContentStart || paragraphs[0].ContentEnd != wantContentEnd {
			t.Fatalf("content range=%d:%d, want %d:%d", paragraphs[0].ContentStart, paragraphs[0].ContentEnd, wantContentStart, wantContentEnd)
		}
		if got := raw[paragraphs[0].ContentStart:paragraphs[0].ContentEnd]; !bytes.Equal(got, append(append(append([]byte{}, shiftToJIS...), encodedText...), shiftToASCII...)) {
			t.Fatalf("content slice=%x does not contain the complete encoded element content", got)
		}

		text, err := QueryBytesWithOptions(`//p/text()`, raw, opts)
		if err != nil || len(text) != 1 {
			t.Fatalf("text query: results=%#v err=%v", text, err)
		}
		wantTextStart := len(`<p>`) + len(shiftToJIS)
		wantTextEnd := wantTextStart + len(encodedText)
		if text[0].StartLocation != wantTextStart || text[0].EndLocation != wantTextEnd {
			t.Fatalf("text range=%d:%d, want direct character bytes %d:%d", text[0].StartLocation, text[0].EndLocation, wantTextStart, wantTextEnd)
		}
	})
}

func TestCharsetRemappingDescendsTemplateContentWithoutLocatingFragment(t *testing.T) {
	raw := []byte(`<template><p>Jos`)
	raw = append(raw, 0xe9)
	raw = append(raw, []byte(`</p></template>`)...)
	decoded, mapper, err := decodeForXPath(raw, "iso-8859-1")
	if err != nil {
		t.Fatal(err)
	}
	document, err := utils.NewHTMLParser().Parse(decoded)
	if err != nil {
		t.Fatal(err)
	}
	mapper.remapDocumentLocations(document)

	template := findElement(document, "template")
	if template == nil || template.TemplateContent == nil || len(template.TemplateContent.Children) != 1 {
		t.Fatalf("template content missing after parse: %#v", template)
	}
	fragment := template.TemplateContent
	paragraph := fragment.Children[0]
	wantStart := bytes.Index(raw, []byte(`<p>`))
	if paragraph.StartPos != wantStart || paragraph.EndPos != bytes.Index(raw, []byte(`</template>`)) {
		t.Fatalf("template child did not retain raw locations: %#v", paragraph)
	}
	if fragment.StartPos != 0 || fragment.EndPos != 0 || fragment.StartLine != 0 {
		t.Fatalf("synthetic template fragment acquired locations: %#v", fragment)
	}
	if document.SourceLength != len(raw) {
		t.Fatalf("document SourceLength=%d, want raw length %d", document.SourceLength, len(raw))
	}
}

func TestPublicCharsetBytesAndUnknownLabel(t *testing.T) {
	document := []byte(`<p>`)
	document = append(document, 0x82, 0xa0)
	document = append(document, []byte(`</p>`)...)

	results, err := QueryBytesWithOptions(`//p[text()='あ']`, document, Options{
		IncludeLocation: true,
		OutputFormat:    "nodes",
		Charset:         "shift_jis",
	})
	if err != nil || len(results) != 1 {
		t.Fatalf("byte API results=%#v err=%v", results, err)
	}
	if results[0].TextContent != "あ" || results[0].Value != string(document) {
		t.Fatalf("byte API did not preserve raw source and decoded text: %#v", results[0])
	}

	compiled, err := Compile(`//p/text()`)
	if err != nil {
		t.Fatal(err)
	}
	results, err = compiled.EvaluateBytesWithOptions(document, Options{Charset: "shift_jis"})
	if err != nil || len(results) != 1 || results[0].TextContent != "あ" {
		t.Fatalf("compiled byte API results=%#v err=%v", results, err)
	}

	_, err = QueryBytesWithOptions("//p", []byte("<p>x</p>"), Options{Charset: "made-up-charset"})
	if err == nil || !strings.Contains(err.Error(), "charset") || !strings.Contains(err.Error(), "made-up-charset") {
		t.Fatalf("unknown charset error must be descriptive, got %v", err)
	}

	// WHATWG aliases ISO-8859-1 labels to Windows-1252, including C1 bytes.
	results, err = QueryBytesWithOptions(`//p[@title='€']`, []byte{'<', 'p', ' ', 't', 'i', 't', 'l', 'e', '=', '\'', 0x80, '\'', '>', 'x', '<', '/', 'p', '>'}, Options{Charset: "ISO-8859-1"})
	if err != nil || len(results) != 1 {
		t.Fatalf("ISO-8859-1 must use Windows-1252 mapping: %#v err=%v", results, err)
	}

	for _, raw := range [][]byte{
		{0x1f, 0x8b, 'x'},
		{'<', 'p', '>', 0x01, '<', '/', 'p', '>'},
	} {
		if _, err := QueryBytes("//p", raw); err == nil || !strings.Contains(err.Error(), "binary input") {
			t.Fatalf("public byte path must reject binary payload %q, err=%v", raw, err)
		}
	}
}

func TestUTF16CodeUnitBytesAreValidatedAfterDecoding(t *testing.T) {
	// U+0101 encodes as the raw bytes 0x01 0x01 in UTF-16LE. Those bytes look
	// like ASCII controls in isolation but decode to ordinary Unicode text.
	raw := []byte{'<', 0, 'p', 0, '>', 0, 0x01, 0x01, '<', 0, '/', 0, 'p', 0, '>', 0}
	results, err := QueryBytesWithOptions(`//p[text()='ā']`, raw, Options{Charset: "utf-16le"})
	if err != nil || len(results) != 1 || results[0].TextContent != "ā" {
		t.Fatalf("UTF-16 code-unit bytes were rejected before decoding: results=%#v err=%v", results, err)
	}
}

func TestMalformedShiftJISPreservesMarkupAndTextRawBoundaries(t *testing.T) {
	// 0x82 is an incomplete Shift_JIS lead byte. The following '<' is emitted
	// by x/text in the same Transform call as U+FFFD, so the mapper must still
	// distinguish the raw byte before the tag from the tag's own boundary.
	raw := append([]byte{'<', 'p', '>', 0x82}, []byte(`<span>target</span></p>`)...)
	spanStart := bytes.Index(raw, []byte(`<span>`))
	spanEnd := bytes.Index(raw, []byte(`</p>`))
	targetStart := bytes.Index(raw, []byte(`target`))
	targetEnd := targetStart + len(`target`)
	opts := Options{IncludeLocation: true, OutputFormat: "nodes", Charset: "shift_jis"}

	spans, err := QueryBytesWithOptions(`//span`, raw, opts)
	if err != nil || len(spans) != 1 {
		t.Fatalf("Shift_JIS malformed-boundary span query: results=%#v err=%v", spans, err)
	}
	if spans[0].StartLocation != spanStart || spans[0].EndLocation != spanEnd || spans[0].Value != string(raw[spanStart:spanEnd]) {
		t.Fatalf("span must retain exact raw boundaries: %#v, want %d:%d", spans[0], spanStart, spanEnd)
	}

	text, err := QueryBytesWithOptions(`//span/text()`, raw, opts)
	if err != nil || len(text) != 1 {
		t.Fatalf("Shift_JIS malformed-boundary text query: results=%#v err=%v", text, err)
	}
	if text[0].StartLocation != targetStart || text[0].EndLocation != targetEnd || text[0].Value != "target" {
		t.Fatalf("text must retain exact raw boundaries: %#v, want %d:%d", text[0], targetStart, targetEnd)
	}
}

func TestCharsetBOMIsStrippedBeforeXPath(t *testing.T) {
	t.Run("utf-8", func(t *testing.T) {
		raw := append([]byte{0xef, 0xbb, 0xbf}, []byte(`<p>text</p>`)...)
		results, err := QueryBytesWithOptions(`//text()`, raw, Options{OutputFormat: "nodes", Charset: "utf-8"})
		if err != nil || len(results) != 1 || results[0].TextContent != "text" {
			t.Fatalf("UTF-8 BOM must not become a text node: results=%#v err=%v", results, err)
		}
		if results[0].StartLocation != 6 || results[0].EndLocation != len(raw)-len(`</p>`) {
			t.Fatalf("UTF-8 BOM text range=%d:%d, want %d:%d", results[0].StartLocation, results[0].EndLocation, 6, len(raw)-len(`</p>`))
		}
	})

	t.Run("utf-16le", func(t *testing.T) {
		raw := []byte{0xff, 0xfe}
		for _, r := range `<p>text</p>` {
			raw = append(raw, byte(r), byte(r>>8))
		}
		results, err := QueryBytesWithOptions(`//text()`, raw, Options{OutputFormat: "nodes", Charset: "utf-16le"})
		if err != nil || len(results) != 1 || results[0].TextContent != "text" {
			t.Fatalf("UTF-16LE BOM must not become a text node: results=%#v err=%v", results, err)
		}
		wantStart := 2 + len(`<p>`)*2
		wantEnd := len(raw) - len(`</p>`)*2
		if results[0].StartLocation != wantStart || results[0].EndLocation != wantEnd {
			t.Fatalf("UTF-16LE BOM text range=%d:%d, want %d:%d", results[0].StartLocation, results[0].EndLocation, wantStart, wantEnd)
		}
	})
}

func findElement(node *types.Node, name string) *types.Node {
	if node == nil {
		return nil
	}
	if node.Type == types.ElementNode && node.Name == name {
		return node
	}
	for _, child := range node.Children {
		if found := findElement(child, name); found != nil {
			return found
		}
	}
	return findElement(node.TemplateContent, name)
}
