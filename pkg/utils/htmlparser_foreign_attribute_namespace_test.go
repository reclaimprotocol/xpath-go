package utils

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Batch 18A covers the fixed WHATWG foreign-attribute adjustment table for
// SVG owners. Namespace-prefix resolution in XPath names, MathML parsing,
// dynamic xmlns bindings, templates, and fragments remain deferred.

const (
	xlinkNamespaceURI = "http://www.w3.org/1999/xlink"
	xmlNamespaceURI   = "http://www.w3.org/XML/1998/namespace"
	xmlnsNamespaceURI = "http://www.w3.org/2000/xmlns/"
)

type expectedForeignAttribute struct {
	name, local, prefix, namespace, value string
}

func assertForeignAttributeMetadata(t *testing.T, nodeName string, order []string, values, namespaces, locals, prefixes map[string]string, expected []expectedForeignAttribute) {
	t.Helper()
	if len(order) != len(expected) {
		t.Fatalf("<%s> attribute order length=%d, want %d: %#v", nodeName, len(order), len(expected), order)
	}
	for index, want := range expected {
		name := order[index]
		if name != want.name || values[name] != want.value || namespaces[name] != want.namespace || locals[name] != want.local || prefixes[name] != want.prefix {
			t.Fatalf("<%s> attr[%d] mismatch: name=%q value=%q ns=%q local=%q prefix=%q, want %#v", nodeName, index, name, values[name], namespaces[name], locals[name], prefixes[name], want)
		}
	}
}

func TestParseSVGFixedForeignAttributeNamespaces(t *testing.T) {
	const content = `<svg id=s xlink:actuate=a xlink:arcrole=b xlink:href=c xlink:role=d xlink:show=e xlink:title=f xlink:type=g xml:lang=h xml:space=i xmlns=j xmlns:xlink=k />`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodeByID(t, document, "s")
	expected := []expectedForeignAttribute{{"id", "id", "", "", "s"}}
	for _, item := range []struct{ name, value string }{
		{"actuate", "a"}, {"arcrole", "b"}, {"href", "c"}, {"role", "d"}, {"show", "e"}, {"title", "f"}, {"type", "g"},
	} {
		expected = append(expected, expectedForeignAttribute{"xlink:" + item.name, item.name, "xlink", xlinkNamespaceURI, item.value})
	}
	expected = append(expected,
		expectedForeignAttribute{"xml:lang", "lang", "xml", xmlNamespaceURI, "h"},
		expectedForeignAttribute{"xml:space", "space", "xml", xmlNamespaceURI, "i"},
		expectedForeignAttribute{"xmlns", "xmlns", "", xmlnsNamespaceURI, "j"},
		expectedForeignAttribute{"xmlns:xlink", "xlink", "xmlns", xmlnsNamespaceURI, "k"},
	)
	assertForeignAttributeMetadata(t, svg.Name, svg.AttributeOrder, svg.Attributes, svg.AttributeNamespaces, svg.AttributeLocalNames, svg.AttributePrefixes, expected)
}

func TestParseSVGForeignAttributeAdjustmentIsASCIIFoldedAndDeclarationIndependent(t *testing.T) {
	const content = `<svg><g id=g XLINK:HREF=x XML:LANG=y XMLNS:XLINK=z /></svg>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	g := svgNodeByID(t, document, "g")
	assertForeignAttributeMetadata(t, g.Name, g.AttributeOrder, g.Attributes, g.AttributeNamespaces, g.AttributeLocalNames, g.AttributePrefixes, []expectedForeignAttribute{
		{"id", "id", "", "", "g"},
		{"xlink:href", "href", "xlink", xlinkNamespaceURI, "x"},
		{"xml:lang", "lang", "xml", xmlNamespaceURI, "y"},
		{"xmlns:xlink", "xlink", "xmlns", xmlnsNamespaceURI, "z"},
	})
}

func TestParseSVGUnknownQualifiedAttributesRemainUnnamespaced(t *testing.T) {
	const content = `<svg id=s xml:base=a xlink:bogus=b foo:bar=c xmlns:foo=d xmlns=e xmlns:xlink=f />`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodeByID(t, document, "s")
	assertForeignAttributeMetadata(t, svg.Name, svg.AttributeOrder, svg.Attributes, svg.AttributeNamespaces, svg.AttributeLocalNames, svg.AttributePrefixes, []expectedForeignAttribute{
		{"id", "id", "", "", "s"},
		{"xml:base", "xml:base", "", "", "a"},
		{"xlink:bogus", "xlink:bogus", "", "", "b"},
		{"foo:bar", "foo:bar", "", "", "c"},
		{"xmlns:foo", "xmlns:foo", "", "", "d"},
		{"xmlns", "xmlns", "", xmlnsNamespaceURI, "e"},
		{"xmlns:xlink", "xlink", "xmlns", xmlnsNamespaceURI, "f"},
	})
}

func TestParseSVGXMLNSValuesDoNotDynamicallyBindForeignNames(t *testing.T) {
	const content = `<svg id=s xmlns:foo=http://example.test/ns foo:bar=v><foo:node id=n foo:bar=w /></svg>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodeByID(t, document, "s")
	child := svgNodeByID(t, document, "n")
	if svg.AttributeNamespaces["xmlns:foo"] != "" || svg.AttributeLocalNames["xmlns:foo"] != "xmlns:foo" || child.NamespaceURI != svgNamespaceURI || child.Name != "foo:node" || child.AttributeNamespaces["foo:bar"] != "" || child.AttributeLocalNames["foo:bar"] != "foo:bar" {
		t.Fatalf("Arbitrary xmlns value dynamically rebound a foreign name: svg=%#v child=%#v", svg, child)
	}
}

func TestParseSVGForeignAttributeAdjustmentDependsOnOwnerNamespace(t *testing.T) {
	const content = `<div id=h xlink:href=a xml:lang=b xmlns:xlink=c></div><svg id=s xlink:href=d><foreignObject id=f xlink:href=e><div id=i xlink:href=g xml:lang=h xmlns:xlink=j></div></foreignObject></svg>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"h", "i"} {
		node := svgNodeByID(t, document, id)
		for _, name := range []string{"xlink:href", "xml:lang", "xmlns:xlink"} {
			if node.AttributeNamespaces[name] != "" || node.AttributeLocalNames[name] != name || node.AttributePrefixes[name] != "" {
				t.Fatalf("HTML-owned %s unexpectedly adjusted on #%s: %#v", name, id, node)
			}
		}
	}
	for _, id := range []string{"s", "f"} {
		node := svgNodeByID(t, document, id)
		if node.AttributeNamespaces["xlink:href"] != xlinkNamespaceURI || node.AttributeLocalNames["xlink:href"] != "href" || node.AttributePrefixes["xlink:href"] != "xlink" {
			t.Fatalf("SVG-owned xlink:href was not adjusted on #%s: %#v", id, node)
		}
	}
}

func TestParseSVGForeignAttributeDuplicatesAndAdjustedOrder(t *testing.T) {
	const duplicates = `<svg id=s XLINK:HREF=first xlink:href=second XML:LANG=a xml:lang=b />`
	document, err := NewHTMLParser().Parse(duplicates)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodeByID(t, document, "s")
	assertForeignAttributeMetadata(t, svg.Name, svg.AttributeOrder, svg.Attributes, svg.AttributeNamespaces, svg.AttributeLocalNames, svg.AttributePrefixes, []expectedForeignAttribute{
		{"id", "id", "", "", "s"},
		{"xlink:href", "href", "xlink", xlinkNamespaceURI, "first"},
		{"xml:lang", "lang", "xml", xmlNamespaceURI, "a"},
	})

	const adjusted = `<svg id=s viewbox="0 0 1 1" xlink:href=x preserveaspectratio=y />`
	document, err = NewHTMLParser().Parse(adjusted)
	if err != nil {
		t.Fatal(err)
	}
	svg = svgNodeByID(t, document, "s")
	if fmt.Sprint(svg.AttributeOrder) != "[id viewBox xlink:href preserveAspectRatio]" || svg.AttributeNamespaces["xlink:href"] != xlinkNamespaceURI || svg.AttributeNamespaces["viewBox"] != "" || svg.AttributeLocalNames["viewBox"] != "viewBox" {
		t.Fatalf("Namespace adjustment changed SVG camel-case order/metadata: %#v", svg)
	}
}

func TestParseSVGForeignAttributeIncompleteTokenReuseAndUnicodeLocation(t *testing.T) {
	parser := NewHTMLParser()
	if document, err := parser.Parse(`<svg xlink:href=x`); err != nil || len(svgNodesNamed(document, "svg")) != 0 {
		t.Fatalf("Incomplete SVG start emitted attribute metadata: doc=%#v err=%v", document, err)
	}

	const content = "<div>é\r\n<svg id=s xlink:title=\"😀\" xml:lang=é /></div>"
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	svg := svgNodeByID(t, document, "s")
	if svg.StartPos != 9 || svg.EndPos != len(content)-6 || svg.StartLine != 2 || svg.StartColumn != 1 || svg.EndLine != 2 || svg.Attributes["xlink:title"] != "😀" || svg.AttributeNamespaces["xlink:title"] != xlinkNamespaceURI || svg.Attributes["xml:lang"] != "é" || svg.AttributeNamespaces["xml:lang"] != xmlNamespaceURI {
		t.Fatalf("Unicode foreign attrs/location mismatch: %#v", svg)
	}

	document, err = parser.Parse(`<svg id=r href=x />`)
	if err != nil {
		t.Fatal(err)
	}
	reused := svgNodeByID(t, document, "r")
	if reused.AttributeNamespaces["href"] != "" || reused.AttributeLocalNames["href"] != "href" || reused.AttributePrefixes["href"] != "" {
		t.Fatalf("Foreign attribute state leaked across parser reuse: %#v", reused)
	}
}

func bestForeignAttributeDuration(t *testing.T, content string) time.Duration {
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

func TestParseSVGForeignAttributeScaling(t *testing.T) {
	build := func(n int) string {
		return strings.Repeat(`<svg xlink:actuate=a xlink:arcrole=b xlink:href=c xlink:role=d xlink:show=e xlink:title=f xlink:type=g xml:lang=h xml:space=i xmlns=j xmlns:xlink=k />`, n)
	}
	smallInput, largeInput := build(1000), build(4000)
	_ = bestForeignAttributeDuration(t, smallInput)
	small := bestForeignAttributeDuration(t, smallInput)
	large := bestForeignAttributeDuration(t, largeInput)
	if large > 10*small && large-small > 100*time.Millisecond {
		t.Fatalf("Foreign attribute adjustment scaled superlinearly: small=%s large=%s", small, large)
	}
}
