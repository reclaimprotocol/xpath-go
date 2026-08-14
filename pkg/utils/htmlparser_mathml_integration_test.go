package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 18C is limited to MathML text integration points and annotation-xml
// integration dispatch in ordinary whole-document in-body parsing. Table,
// select, template and fragment modes, processing instructions, execution,
// layout validation, and broad form/adoption interactions remain deferred.

func requireMathIntegrationNode(t *testing.T, node *types.Node, name, namespace, text string, start, end, contentStart, contentEnd int) {
	t.Helper()
	if node == nil || node.Name != name || node.NamespaceURI != namespace || node.TextContent != text || node.StartPos != start || node.EndPos != end || node.ContentStart != contentStart || node.ContentEnd != contentEnd {
		t.Fatalf("integration node mismatch: got %#v, want <%s> ns=%q text=%q range=%d:%d content=%d:%d", node, name, namespace, text, start, end, contentStart, contentEnd)
	}
}

func TestParseMathMLTextIntegrationPointOwners(t *testing.T) {
	for _, owner := range []string{"mi", "mo", "mn", "ms", "mtext"} {
		t.Run(owner, func(t *testing.T) {
			content := `<math><mrow><` + owner + ` id=o>a<b id=h>x</b>c</` + owner + `><mn id=n>z</mn></mrow></math>`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			ownerNode := svgNodeByID(t, document, "o")
			htmlBold := svgNodeByID(t, document, "h")
			following := svgNodeByID(t, document, "n")
			ownerEnd := strings.Index(content, `</`+owner+`>`) + len(`</`+owner+`>`)
			ownerContentStart := strings.Index(content, `>a<b`) + 1
			requireMathIntegrationNode(t, ownerNode, owner, mathMLTestNamespaceURI, "axc", 12, ownerEnd, ownerContentStart, ownerEnd-len(`</`+owner+`>`))
			requireMathIntegrationNode(t, htmlBold, "b", htmlNamespaceURI, "x", strings.Index(content, `<b`), strings.Index(content, `</b>`)+4, strings.Index(content, `>x</b>`)+1, strings.Index(content, `</b>`))
			if htmlBold.Parent != ownerNode || following.Parent == nil || following.Parent.Name != "mrow" || following.NamespaceURI != mathMLTestNamespaceURI || following.TextContent != "z" {
				t.Fatalf("text integration ownership/restoration mismatch: owner=%#v bold=%#v next=%#v", ownerNode, htmlBold, following)
			}
		})
	}
}

func TestParseMathMLTextIntegrationExceptionsAndArbitraryHTML(t *testing.T) {
	const exceptions = `<math><mi id=o>a<mglyph id=g /><malignmark id=l />b<span id=h>c</span></mi></math>`
	document, err := NewHTMLParser().Parse(exceptions)
	if err != nil {
		t.Fatal(err)
	}
	owner := svgNodeByID(t, document, "o")
	mglyph := svgNodeByID(t, document, "g")
	malignmark := svgNodeByID(t, document, "l")
	span := svgNodeByID(t, document, "h")
	requireMathIntegrationNode(t, owner, "mi", mathMLTestNamespaceURI, "abc", 6, 75, 15, 70)
	requireMathIntegrationNode(t, mglyph, "mglyph", mathMLTestNamespaceURI, "", 16, 31, 31, 31)
	requireMathIntegrationNode(t, malignmark, "malignmark", mathMLTestNamespaceURI, "", 31, 50, 50, 50)
	requireMathIntegrationNode(t, span, "span", htmlNamespaceURI, "c", 51, 70, 62, 63)
	if mglyph.Parent != owner || malignmark.Parent != owner || span.Parent != owner {
		t.Fatalf("MathML text-point exception parents mismatch: %#v", owner.Children)
	}

	const arbitrary = `<math><mi id=o>a<section id=h><b id=b>x</b></section>c</mi><mn id=n>z</mn></math>`
	document, err = NewHTMLParser().Parse(arbitrary)
	if err != nil {
		t.Fatal(err)
	}
	owner = svgNodeByID(t, document, "o")
	section := svgNodeByID(t, document, "h")
	bold := svgNodeByID(t, document, "b")
	next := svgNodeByID(t, document, "n")
	requireMathIntegrationNode(t, owner, "mi", mathMLTestNamespaceURI, "axc", 6, 59, 15, 54)
	requireMathIntegrationNode(t, section, "section", htmlNamespaceURI, "x", 16, 53, 30, 43)
	if section.Parent != owner || bold.Parent != section || bold.NamespaceURI != htmlNamespaceURI || next.NamespaceURI != mathMLTestNamespaceURI || next.Parent != owner.Parent {
		t.Fatalf("arbitrary HTML start under MathML text integration point mismatch: owner=%#v section=%#v next=%#v", owner, section, next)
	}
}

func TestParseMathMLTextIntegrationLiveHTMLBarrier(t *testing.T) {
	// Chrome 151 is authoritative here. Bundled jsdom closes the owner at the
	// first </mi>, but current WHATWG keeps it open through the live HTML span.
	const content = `<math><mi id=o><span id=s>x</mi>y</span>z</mi><mn id=n>q</mn></math>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	owner := svgNodeByID(t, document, "o")
	span := svgNodeByID(t, document, "s")
	next := svgNodeByID(t, document, "n")
	requireMathIntegrationNode(t, owner, "mi", mathMLTestNamespaceURI, "xyz", 6, 46, 15, 41)
	requireMathIntegrationNode(t, span, "span", htmlNamespaceURI, "xy", 15, 40, 26, 33)
	if len(span.Children) != 1 || span.Children[0].Value != "xy" || span.Children[0].StartPos != 26 || span.Children[0].EndPos != 33 || next.Parent != owner.Parent || next.NamespaceURI != mathMLTestNamespaceURI {
		t.Fatalf("live HTML barrier did not ignore first owner end: owner=%#v span=%#v next=%#v", owner, span, next)
	}
}

func TestParseMathMLTextIntegrationNestedForeignRoots(t *testing.T) {
	const content = `<math id=M><mtext id=t><svg id=s><circle id=c /></svg><math id=i><mn id=n>x</mn></math></mtext><mo id=q>y</mo></math>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	outer := svgNodeByID(t, document, "M")
	text := svgNodeByID(t, document, "t")
	svg := svgNodeByID(t, document, "s")
	circle := svgNodeByID(t, document, "c")
	inner := svgNodeByID(t, document, "i")
	mn := svgNodeByID(t, document, "n")
	mo := svgNodeByID(t, document, "q")
	requireMathIntegrationNode(t, outer, "math", mathMLTestNamespaceURI, "xy", 0, 117, 11, 110)
	requireMathIntegrationNode(t, text, "mtext", mathMLTestNamespaceURI, "x", 11, 95, 23, 87)
	requireMathIntegrationNode(t, svg, "svg", svgNamespaceURI, "", 23, 54, 33, 48)
	requireMathIntegrationNode(t, circle, "circle", svgNamespaceURI, "", 33, 48, 48, 48)
	if svg.Parent != text || inner.Parent != text || inner.NamespaceURI != mathMLTestNamespaceURI || mn.Parent != inner || mn.NamespaceURI != mathMLTestNamespaceURI || mo.Parent != outer || mo.NamespaceURI != mathMLTestNamespaceURI {
		t.Fatalf("nested SVG/MathML integration stack restoration mismatch: outer=%#v text=%#v", outer, text)
	}
}

func TestParseMathMLTextIntegrationCharactersEOFAndReuse(t *testing.T) {
	const characters = "<math><mi id=o>é\r\n&amp;\x00&#0;<!--c-->😀</mi></math>"
	document, err := NewHTMLParser().Parse(characters)
	if err != nil {
		t.Fatal(err)
	}
	owner := svgNodeByID(t, document, "o")
	if owner.TextContent != "é\n&�😀" || len(owner.Children) != 3 || owner.Children[0].Value != "é\n&�" || owner.Children[0].StartPos != 15 || owner.Children[0].EndPos != 29 || owner.Children[1].Type != types.CommentNode || owner.Children[1].Value != "c" || owner.Children[1].StartPos != 29 || owner.Children[1].EndPos != 37 || owner.Children[2].Value != "😀" || owner.Children[2].StartPos != 37 || owner.Children[2].EndPos != 41 {
		t.Fatalf("text integration characters/comment mismatch: %#v", owner)
	}

	const eof = "<div><math><mi id=o>é\r\n😀<b id=b>x"
	parser := NewHTMLParser()
	document, err = parser.Parse(eof)
	if err != nil {
		t.Fatal(err)
	}
	owner = svgNodeByID(t, document, "o")
	bold := svgNodeByID(t, document, "b")
	for _, node := range []*types.Node{svgNodesNamed(document, "div")[0], svgNodesNamed(document, "math")[0], owner, bold} {
		if node.EndPos != 37 || node.EndLine != 2 || node.EndColumn != 12 {
			t.Fatalf("MathML text integration EOF endpoint mismatch on <%s>: %#v", node.Name, node)
		}
	}
	if owner.NamespaceURI != mathMLTestNamespaceURI || bold.NamespaceURI != htmlNamespaceURI || bold.Parent != owner || owner.TextContent != "é\n😀x" {
		t.Fatalf("MathML text integration EOF tree mismatch: owner=%#v bold=%#v", owner, bold)
	}
	if document, parseErr := parser.Parse(`<div>x`); parseErr != nil || len(svgNodesNamed(document, "div")) != 1 || svgNodesNamed(document, "div")[0].EndPos != 6 {
		t.Fatalf("MathML integration implicit-document EOF reuse mismatch: doc=%#v err=%v", document, parseErr)
	}
}

func TestParseMathMLAnnotationXMLQualifiedEncodings(t *testing.T) {
	for _, test := range []struct {
		name, encoding string
		end            int
	}{
		{"html", "text/html", 82},
		{"xhtml", "APPLICATION/XHTML+XML", 94},
	} {
		t.Run(test.name, func(t *testing.T) {
			content := `<math><annotation-xml id=a encoding=` + test.encoding + `>a<div id=h>x</div>c</annotation-xml><mn id=n>z</mn></math>`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			annotation := svgNodeByID(t, document, "a")
			htmlDiv := svgNodeByID(t, document, "h")
			next := svgNodeByID(t, document, "n")
			if annotation.NamespaceURI != mathMLTestNamespaceURI || annotation.TextContent != "axc" || annotation.EndPos != test.end || htmlDiv.NamespaceURI != htmlNamespaceURI || htmlDiv.Parent != annotation || next.NamespaceURI != mathMLTestNamespaceURI || next.Parent != annotation.Parent {
				t.Fatalf("qualifying annotation-xml encoding %q mismatch: annotation=%#v div=%#v next=%#v", test.encoding, annotation, htmlDiv, next)
			}
		})
	}
}

func TestParseMathMLAnnotationXMLEncodingExactness(t *testing.T) {
	const decoded = `<math><annotation-xml id=a encoding=text&#47;html><span id=h>x</span></annotation-xml><mn id=n>z</mn></math>`
	document, err := NewHTMLParser().Parse(decoded)
	if err != nil {
		t.Fatal(err)
	}
	annotation := svgNodeByID(t, document, "a")
	span := svgNodeByID(t, document, "h")
	if annotation.NamespaceURI != mathMLTestNamespaceURI || annotation.Attributes["encoding"] != "text/html" || span.NamespaceURI != htmlNamespaceURI || span.Parent != annotation || svgNodeByID(t, document, "n").NamespaceURI != mathMLTestNamespaceURI {
		t.Fatalf("decoded qualifying annotation encoding mismatch: annotation=%#v span=%#v", annotation, span)
	}

	for _, content := range []string{
		`<math><annotation-xml id=a encoding=' text/html'><span id=h>x</span></annotation-xml><mn id=n>z</mn></math>`,
		`<math><annotation-xml id=a encoding=text/plain ENCODING=text/html><span id=h>x</span></annotation-xml><mn id=n>z</mn></math>`,
	} {
		document, err = NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		annotation = svgNodeByID(t, document, "a")
		span = svgNodeByID(t, document, "h")
		next := svgNodeByID(t, document, "n")
		if annotation.NamespaceURI != mathMLTestNamespaceURI || annotation.EndPos != span.StartPos || annotation.Parent == span.Parent || span.NamespaceURI != htmlNamespaceURI || next.NamespaceURI != htmlNamespaceURI {
			t.Fatalf("nonqualifying annotation encoding did not break out at span: annotation=%#v span=%#v next=%#v", annotation, span, next)
		}
	}

	const noException = `<math><annotation-xml id=a encoding=text/html><mglyph id=g>x</mglyph></annotation-xml></math>`
	document, err = NewHTMLParser().Parse(noException)
	if err != nil {
		t.Fatal(err)
	}
	mglyph := svgNodeByID(t, document, "g")
	if mglyph.NamespaceURI != htmlNamespaceURI || mglyph.Parent != svgNodeByID(t, document, "a") {
		t.Fatalf("annotation-xml incorrectly applied text-point mglyph exception: %#v", mglyph)
	}
}

func TestParseMathMLAnnotationXMLDirectSVGTransition(t *testing.T) {
	const content = `<math><annotation-xml id=a encoding=x><svg id=s><circle id=c /></svg><mrow id=r>x</mrow></annotation-xml></math>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	annotation := svgNodeByID(t, document, "a")
	svg := svgNodeByID(t, document, "s")
	circle := svgNodeByID(t, document, "c")
	mrow := svgNodeByID(t, document, "r")
	requireMathIntegrationNode(t, annotation, "annotation-xml", mathMLTestNamespaceURI, "x", 6, 105, 38, 88)
	requireMathIntegrationNode(t, svg, "svg", svgNamespaceURI, "", 38, 69, 48, 63)
	requireMathIntegrationNode(t, circle, "circle", svgNamespaceURI, "", 48, 63, 63, 63)
	if svg.Parent != annotation || mrow.Parent != annotation || mrow.NamespaceURI != mathMLTestNamespaceURI || mrow.TextContent != "x" {
		t.Fatalf("annotation direct SVG transition/restoration mismatch: annotation=%#v svg=%#v mrow=%#v", annotation, svg, mrow)
	}
}

func TestParseMathMLAnnotationXMLLiveHTMLBarrier(t *testing.T) {
	// Chrome 151 is authoritative; bundled jsdom closes at the first owner end.
	const content = `<math><annotation-xml id=a encoding=text/html><span id=s>x</annotation-xml>y</span>z</annotation-xml><mn id=n>q</mn></math>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	annotation := svgNodeByID(t, document, "a")
	span := svgNodeByID(t, document, "s")
	next := svgNodeByID(t, document, "n")
	firstEnd := strings.Index(content, `</annotation-xml>`)
	spanEnd := strings.Index(content, `</span>`) + len(`</span>`)
	secondEndStart := strings.LastIndex(content, `</annotation-xml>`)
	secondEnd := secondEndStart + len(`</annotation-xml>`)
	requireMathIntegrationNode(t, annotation, "annotation-xml", mathMLTestNamespaceURI, "xyz", 6, secondEnd, strings.Index(content, `<span`), secondEndStart)
	requireMathIntegrationNode(t, span, "span", htmlNamespaceURI, "xy", strings.Index(content, `<span`), spanEnd, strings.Index(content, `>x</annotation`)+1, strings.Index(content, `</span>`))
	if len(span.Children) != 1 || span.Children[0].Value != "xy" || span.Children[0].StartPos >= firstEnd || span.Children[0].EndPos != strings.Index(content, `</span>`) || next.NamespaceURI != mathMLTestNamespaceURI || next.Parent != annotation.Parent {
		t.Fatalf("annotation live-child end barrier mismatch: annotation=%#v span=%#v next=%#v", annotation, span, next)
	}
}

func TestParseMathMLIntegrationIncompleteTokensFormattingAndScaling(t *testing.T) {
	for _, test := range []struct {
		content, owner string
	}{
		{`<math><mi id=o>x<span`, "o"},
		{`<math><annotation-xml id=a encoding=text/html>x<div`, "a"},
	} {
		document, err := NewHTMLParser().Parse(test.content)
		if err != nil {
			t.Fatal(err)
		}
		owner := svgNodeByID(t, document, test.owner)
		if owner.TextContent != "x" || owner.EndPos != len(test.content) || len(owner.Children) != 1 || owner.Children[0].Value != "x" || owner.Children[0].EndPos != len(test.content) {
			t.Fatalf("incomplete integration child start mutated dispatcher state: %#v", owner)
		}
	}

	const formatting = `<b><math><mi><i>x</i></mi></math>y</b>z`
	document, err := NewHTMLParser().Parse(formatting)
	if err != nil {
		t.Fatal(err)
	}
	bold := svgNodesNamed(document, "b")[0]
	if bold.TextContent != "xy" || svgNodesNamed(document, "i")[0].NamespaceURI != htmlNamespaceURI || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].Value != "z" {
		t.Fatalf("MathML integration corrupted active formatting exit: %#v", parsedBodyChildren(document))
	}

	build := func(n int) string {
		return strings.Repeat(`<math><mi>a<b>x</b>c</mi><annotation-xml encoding=text/html><span>y</span></annotation-xml></math>`, n)
	}
	smallInput, largeInput := build(500), build(2000)
	_ = bestMathMLParseDuration(t, smallInput)
	small := bestMathMLParseDuration(t, smallInput)
	large := bestMathMLParseDuration(t, largeInput)
	if large > 10*small && large-small > 100*time.Millisecond {
		t.Fatalf("MathML integration parsing scaled superlinearly: small=%s large=%s", small, large)
	}
}
