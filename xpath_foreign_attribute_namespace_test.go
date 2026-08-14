package xpath_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQuerySVGForeignAttributePublicMetadata(t *testing.T) {
	const content = `<svg id=s xlink:href=x xml:lang=en xml:base=b xmlns=u xmlns:xlink=v href=y />`
	results, err := xpath.Query(`//svg/@*`, content)
	if err != nil {
		t.Fatal(err)
	}
	expected := []struct {
		name, local, prefix, namespace, value string
	}{
		{"id", "id", "", "", "s"},
		{"xlink:href", "href", "xlink", "http://www.w3.org/1999/xlink", "x"},
		{"xml:lang", "lang", "xml", "http://www.w3.org/XML/1998/namespace", "en"},
		{"xml:base", "xml:base", "", "", "b"},
		{"xmlns", "xmlns", "", "http://www.w3.org/2000/xmlns/", "u"},
		{"xmlns:xlink", "xlink", "xmlns", "http://www.w3.org/2000/xmlns/", "v"},
		{"href", "href", "", "", "y"},
	}
	if len(results) != len(expected) {
		t.Fatalf("Expected %d attr results, got %#v", len(expected), results)
	}
	for index, want := range expected {
		got := results[index]
		if got.NodeName != want.name || got.LocalName != want.local || got.Prefix != want.prefix || got.NamespaceURI != want.namespace || got.TextContent != want.value || got.Value != want.value || got.StartLocation != 0 || got.EndLocation != 0 {
			t.Fatalf("Attr result[%d] mismatch: got=%#v want=%#v", index, got, want)
		}
	}
}

func TestQuerySVGForeignAttributeLocalNameAndNamespacePredicates(t *testing.T) {
	const content = `<svg id=s xlink:href=x xml:lang=en xml:base=b xmlns:xlink=v href=y />`
	tests := []struct {
		expression, name, value string
	}{
		{`//svg/@*[local-name()='href' and namespace-uri()='http://www.w3.org/1999/xlink']`, "xlink:href", "x"},
		{`//svg/@*[local-name()='lang' and namespace-uri()='http://www.w3.org/XML/1998/namespace']`, "xml:lang", "en"},
		{`//svg/@*[local-name()='xml:base' and namespace-uri()='']`, "xml:base", "b"},
		{`//svg/@*[local-name()='href' and namespace-uri()='']`, "href", "y"},
	}
	for _, test := range tests {
		results, err := xpath.Query(test.expression, content)
		if err != nil {
			t.Fatalf("Query %s failed: %v", test.expression, err)
		}
		if len(results) != 1 || results[0].NodeName != test.name || results[0].TextContent != test.value {
			t.Fatalf("Query %s returned %#v", test.expression, results)
		}
	}
	assertFormattingNoQuery(t, content, `//svg/@href[namespace-uri()='http://www.w3.org/1999/xlink']`)
}

func TestQuerySVGForeignAttributeNameTestsAndUnionIdentity(t *testing.T) {
	const content = `<svg id=a xlink:href=x href=y><g id=b xlink:href=z /></svg>`
	results, err := xpath.Query(`//svg/@href`, content)
	if err != nil || len(results) != 1 || results[0].NodeName != "href" || results[0].TextContent != "y" || results[0].NamespaceURI != "" {
		t.Fatalf("Null-namespace @href confused with xlink:href: results=%#v err=%v", results, err)
	}

	results, err = xpath.Query(`//svg/@* | //svg/@*`, content)
	if err != nil || len(results) != 3 {
		t.Fatalf("Union failed to deduplicate identical namespaced attr identities: results=%#v err=%v", results, err)
	}

	results, err = xpath.Query(`//*[@id='a']/@* | //*[@id='b']/@*`, content)
	if err != nil || len(results) != 5 {
		t.Fatalf("Same qualified attrs on different owners lost identity: results=%#v err=%v", results, err)
	}
	xlinkCount := 0
	for _, result := range results {
		if result.NodeName == "xlink:href" && result.NamespaceURI == "http://www.w3.org/1999/xlink" {
			xlinkCount++
		}
	}
	if xlinkCount != 2 {
		t.Fatalf("Expected two owner-distinct XLink href attrs, got %#v", results)
	}
}

func TestQuerySVGForeignAttributeHTMLIntegrationContrastAndReuse(t *testing.T) {
	const content = `<svg id=s xlink:href=x><foreignObject><div id=h xlink:href=y /></foreignObject></svg>`
	results, err := xpath.Query(`//*[@id='s']/@* | //*[@id='h']/@*`, content)
	if err != nil || len(results) != 4 {
		t.Fatalf("Expected SVG/HTML qualified attr contrast, got %#v err=%v", results, err)
	}
	var svgAttr, htmlAttr *xpath.Result
	for i := range results {
		if results[i].NodeName != "xlink:href" {
			continue
		}
		switch results[i].TextContent {
		case "x":
			svgAttr = &results[i]
		case "y":
			htmlAttr = &results[i]
		}
	}
	if svgAttr == nil || htmlAttr == nil || svgAttr.NamespaceURI != "http://www.w3.org/1999/xlink" || svgAttr.LocalName != "href" || svgAttr.Prefix != "xlink" || htmlAttr.NamespaceURI != "" || htmlAttr.LocalName != "xlink:href" || htmlAttr.Prefix != "" {
		t.Fatalf("Owner namespace did not control foreign attribute adjustment: %#v", results)
	}

	if results, err = xpath.Query(`//svg/@*`, `<svg xlink:href=x`); err != nil || len(results) != 0 {
		t.Fatalf("Incomplete SVG start emitted public attr metadata: results=%#v err=%v", results, err)
	}
	results, err = xpath.Query(`//svg/@href`, `<svg href=z />`)
	if err != nil || len(results) != 1 || results[0].NamespaceURI != "" || results[0].LocalName != "href" || results[0].Prefix != "" {
		t.Fatalf("Foreign attr metadata leaked across query reuse: results=%#v err=%v", results, err)
	}
}

func TestQueryLocalNameOfTextNodeIsEmpty(t *testing.T) {
	const content = `<div>a<span>x</span>b</div>`
	results, err := xpath.Query(`//div/text()[local-name()='']`, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].NodeName != "#text" || results[0].TextContent != "a" || results[1].NodeName != "#text" || results[1].TextContent != "b" {
		t.Fatalf("local-name() did not return the empty string for text nodes: %#v", results)
	}
	assertFormattingNoQuery(t, content, `//div/text()[local-name()='#text']`)
}

func buildHighAttributeSVG(count int) string {
	var builder strings.Builder
	builder.Grow(count*16 + 10)
	builder.WriteString(`<svg`)
	for index := 0; index < count; index++ {
		fmt.Fprintf(&builder, " data-%d=x", index)
	}
	builder.WriteString(` />`)
	return builder.String()
}

func bestHighAttributeQueryDuration(t *testing.T, content string, expected int) time.Duration {
	t.Helper()
	best := time.Duration(1<<63 - 1)
	for attempt := 0; attempt < 3; attempt++ {
		start := time.Now()
		results, err := xpath.Query(`//svg/@*`, content)
		elapsed := time.Since(start)
		if err != nil || len(results) != expected {
			t.Fatalf("High-attribute query returned %d results, want %d: err=%v", len(results), expected, err)
		}
		if elapsed < best {
			best = elapsed
		}
	}
	return best
}

func TestQuerySVGHighAttributeSelectionScaling(t *testing.T) {
	smallInput, largeInput := buildHighAttributeSVG(1000), buildHighAttributeSVG(4000)
	_ = bestHighAttributeQueryDuration(t, smallInput, 1000)
	small := bestHighAttributeQueryDuration(t, smallInput, 1000)
	large := bestHighAttributeQueryDuration(t, largeInput, 4000)
	if large > 8*small && large-small > 75*time.Millisecond {
		t.Fatalf("Selecting attributes from one SVG owner scaled superlinearly: small=%s large=%s", small, large)
	}
}
