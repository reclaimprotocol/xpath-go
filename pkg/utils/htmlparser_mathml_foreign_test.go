package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Batch 18B covers closed MathML foreign islands in ordinary in-body document
// parsing. MathML text integration points, annotation-xml HTML integration,
// nested SVG, table/select/template modes, fragments, and processing
// instructions remain deferred.

const mathMLTestNamespaceURI = "http://www.w3.org/1998/Math/MathML"

func TestParseClosedMathMLIslandNamespaceTreeAndExit(t *testing.T) {
	const content = `<div>a<math id=m><mrow id=r><mi id=i>x</mi></mrow></math>b</div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	div := svgNodesNamed(document, "div")[0]
	math := svgNodeByID(t, document, "m")
	mrow := svgNodeByID(t, document, "r")
	mi := svgNodeByID(t, document, "i")
	assertFormattingNode(t, div, "div", "axb", 0, 64)
	assertFormattingNode(t, math, "math", "x", 6, 57)
	assertFormattingNode(t, mrow, "mrow", "x", 17, 50)
	assertFormattingNode(t, mi, "mi", "x", 28, 43)
	if div.ContentStart != 5 || div.ContentEnd != 58 || math.ContentStart != 17 || math.ContentEnd != 50 || mrow.ContentStart != 28 || mrow.ContentEnd != 43 || mi.ContentStart != 37 || mi.ContentEnd != 38 || len(div.Children) != 3 || div.Children[1] != math {
		t.Fatalf("Closed MathML source tree mismatch: div=%#v math=%#v mrow=%#v mi=%#v", div, math, mrow, mi)
	}
	requireNodeNamespace(t, div, htmlNamespaceURI)
	for _, node := range []*types.Node{math, mrow, mi} {
		requireNodeNamespace(t, node, mathMLTestNamespaceURI)
	}
}

func TestParseMathMLSelfClosingAndRootExit(t *testing.T) {
	const content = `<math><mrow id=r /><mi id=i>x</mi></math><p>z</p>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	math := svgNodesNamed(document, "math")[0]
	mrow := svgNodeByID(t, document, "r")
	mi := svgNodeByID(t, document, "i")
	p := svgNodesNamed(document, "p")[0]
	assertFormattingNode(t, math, "math", "x", 0, 41)
	assertFormattingNode(t, mrow, "mrow", "", 6, 19)
	assertFormattingNode(t, mi, "mi", "x", 19, 34)
	assertFormattingNode(t, p, "p", "z", 41, 49)
	if mrow.ContentStart != 19 || mrow.ContentEnd != 19 || mi.ContentStart != 28 || mi.ContentEnd != 29 || p.ContentStart != 44 || p.ContentEnd != 45 {
		t.Fatalf("MathML self-close/HTML exit boundaries mismatch: mrow=%#v mi=%#v p=%#v", mrow, mi, p)
	}
	for _, node := range []*types.Node{math, mrow, mi} {
		requireNodeNamespace(t, node, mathMLTestNamespaceURI)
	}
	requireNodeNamespace(t, p, htmlNamespaceURI)

	const root = `<math id=m />z`
	document, err = NewHTMLParser().Parse(root)
	if err != nil {
		t.Fatal(err)
	}
	math = svgNodeByID(t, document, "m")
	assertFormattingNode(t, math, "math", "", 0, 13)
	if math.ContentStart != 13 || math.ContentEnd != 13 || len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[1].Value != "z" || parsedBodyChildren(document)[1].StartPos != 13 {
		t.Fatalf("Self-closing MathML root did not restore HTML data state: %#v", parsedBodyChildren(document))
	}
}

func TestParseMathMLAdjustsDefinitionURLAndForeignAttributes(t *testing.T) {
	const content = `<math id=m definitionurl=root xlink:href=a><mrow id=n definitionurl=first definitionURL=second xml:lang=en /></math>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	math := svgNodeByID(t, document, "m")
	annotation := svgNodeByID(t, document, "n")
	if len(math.AttributeOrder) != 3 || math.AttributeOrder[1] != "definitionURL" || math.Attributes["definitionURL"] != "root" || math.AttributeLocalNames["definitionURL"] != "definitionURL" || math.AttributeNamespaces["definitionURL"] != "" || math.AttributeNamespaces["xlink:href"] != xlinkNamespaceURI {
		t.Fatalf("MathML root attribute adjustment mismatch: %#v", math)
	}
	if len(annotation.AttributeOrder) != 3 || annotation.AttributeOrder[1] != "definitionURL" || annotation.Attributes["definitionURL"] != "first" || annotation.AttributeLocalNames["definitionURL"] != "definitionURL" || annotation.AttributeNamespaces["xml:lang"] != xmlNamespaceURI {
		t.Fatalf("MathML descendant definitionURL/duplicate/foreign attrs mismatch: %#v", annotation)
	}
	requireNodeNamespace(t, annotation, mathMLTestNamespaceURI)
}

func TestParseMathMLTextReferencesNULCommentsCDATAAndDoctype(t *testing.T) {
	const textContent = "<math>a&amp;b&#32;c\x00<mi>d</mi></math>z"
	document, err := NewHTMLParser().Parse(textContent)
	if err != nil {
		t.Fatal(err)
	}
	math := svgNodesNamed(document, "math")[0]
	mi := svgNodesNamed(document, "mi")[0]
	assertFormattingNode(t, math, "math", "a&b c�d", 0, 37)
	assertFormattingNode(t, mi, "mi", "d", 20, 30)
	if len(math.Children) != 2 || math.Children[0].Value != "a&b c�" || math.Children[0].StartPos != 6 || math.Children[0].EndPos != 20 || parsedBodyChildren(document)[1].Value != "z" {
		t.Fatalf("MathML text/reference/NUL behavior mismatch: %#v", math.Children)
	}

	const markup = `<math><!--c--><![CDATA[a<b>&copy;]]><mi>x</mi></math>z`
	document, err = NewHTMLParser().Parse(markup)
	if err != nil {
		t.Fatal(err)
	}
	math = svgNodesNamed(document, "math")[0]
	if len(math.Children) != 3 || math.Children[0].Type != types.CommentNode || math.Children[0].Value != "c" || math.Children[0].StartPos != 6 || math.Children[0].EndPos != 14 || math.Children[1].Type != types.TextNode || math.Children[1].Value != "a<b>&copy;" || math.Children[1].StartPos != 14 || math.Children[1].EndPos != 36 {
		t.Fatalf("MathML comment/CDATA order or literal value mismatch: %#v", math.Children)
	}

	const doctype = `<math>a<!DOCTYPE x>b<mi /></math>z`
	document, err = NewHTMLParser().Parse(doctype)
	if err != nil {
		t.Fatal(err)
	}
	math = svgNodesNamed(document, "math")[0]
	if math.TextContent != "ab" || len(math.Children) != 2 || math.Children[0].Value != "ab" || math.Children[0].StartPos != 6 || math.Children[0].EndPos != 20 || math.Children[1].Name != "mi" || math.Children[1].StartPos != 20 || math.Children[1].EndPos != 26 {
		t.Fatalf("MathML ignored doctype/coalesced text mismatch: %#v", math)
	}
}

func TestParseMathMLCaseCloseAncestorPopAndMalformedEnds(t *testing.T) {
	const caseClose = `<math><mrow id=r>x</MROW>y</math>z`
	document, err := NewHTMLParser().Parse(caseClose)
	if err != nil {
		t.Fatal(err)
	}
	math := svgNodesNamed(document, "math")[0]
	mrow := svgNodeByID(t, document, "r")
	assertFormattingNode(t, math, "math", "xy", 0, 33)
	assertFormattingNode(t, mrow, "mrow", "x", 6, 25)

	const nonASCII = `<math><AÀ id=x>q</AÀ></math>`
	document, err = NewHTMLParser().Parse(nonASCII)
	if err != nil {
		t.Fatal(err)
	}
	foreign := svgNodeByID(t, document, "x")
	if foreign.Name != "aÀ" || foreign.TextContent != "q" || foreign.NamespaceURI != mathMLTestNamespaceURI {
		t.Fatalf("MathML tokenizer/end matching used Unicode rather than ASCII folding: %#v", foreign)
	}

	const ancestor = `<math><mrow><mi id=i>x</mrow><mn id=n /></math>z`
	document, err = NewHTMLParser().Parse(ancestor)
	if err != nil {
		t.Fatal(err)
	}
	mrow = svgNodesNamed(document, "mrow")[0]
	mi := svgNodeByID(t, document, "i")
	mn := svgNodeByID(t, document, "n")
	assertFormattingNode(t, mrow, "mrow", "x", 6, 29)
	assertFormattingNode(t, mi, "mi", "x", 12, 22)
	assertFormattingNode(t, mn, "mn", "", 29, 40)
	if mi.ContentStart != 21 || mi.ContentEnd != 22 || mn.ContentStart != 40 || mn.ContentEnd != 40 {
		t.Fatalf("MathML ancestor pop/self-close boundary mismatch: mi=%#v mn=%#v", mi, mn)
	}

	for _, test := range []struct {
		content, value string
		start, end     int
	}{
		{`<math>a</>b</math>z`, "ab", 6, 11},
		{`<math>x</foo></math>`, "x", 6, 7},
		{`<math>x</foo>y</math>`, "xy", 6, 14},
	} {
		document, err = NewHTMLParser().Parse(test.content)
		if err != nil {
			t.Fatal(err)
		}
		math = svgNodesNamed(document, "math")[0]
		if len(math.Children) != 1 || math.Children[0].Value != test.value || math.Children[0].StartPos != test.start || math.Children[0].EndPos != test.end {
			t.Fatalf("Malformed/unmatched MathML end mismatch for %q: %#v", test.content, math.Children)
		}
	}

	const bogus = `<math>a</42>b</math>z`
	document, err = NewHTMLParser().Parse(bogus)
	if err != nil {
		t.Fatal(err)
	}
	math = svgNodesNamed(document, "math")[0]
	if len(math.Children) != 3 || math.Children[0].Value != "a" || math.Children[1].Type != types.CommentNode || math.Children[1].Value != "42" || math.Children[1].StartPos != 7 || math.Children[1].EndPos != 12 || math.Children[2].Value != "b" {
		t.Fatalf("Bogus MathML end did not become a comment: %#v", math.Children)
	}
}

func TestParseMathMLEOFIncompleteEndCDATAReuse(t *testing.T) {
	const unicode = "<div><math><mrow>é\r\n😀"
	parser := NewHTMLParser()
	document, err := parser.Parse(unicode)
	if err != nil {
		t.Fatal(err)
	}
	div := svgNodesNamed(document, "div")[0]
	math := svgNodesNamed(document, "math")[0]
	mrow := svgNodesNamed(document, "mrow")[0]
	for _, node := range []*types.Node{div, math, mrow} {
		if node.EndPos != 25 || node.EndLine != 2 || node.EndColumn != 3 {
			t.Fatalf("MathML EOF endpoint mismatch on <%s>: %#v", node.Name, node)
		}
	}
	if len(mrow.Children) != 1 || mrow.Children[0].Value != "é\n😀" || mrow.Children[0].StartPos != 17 || mrow.Children[0].EndPos != 25 {
		t.Fatalf("MathML EOF Unicode text mismatch: %#v", mrow.Children)
	}
	if document, parseErr := parser.Parse(`<div>x`); parseErr != nil || len(svgNodesNamed(document, "div")) != 1 || svgNodesNamed(document, "div")[0].EndPos != 6 {
		t.Fatalf("MathML-to-implicit-document EOF reuse mismatch: doc=%#v err=%v", document, parseErr)
	}

	const incompleteEnd = `<math><mrow>x</mrow`
	document, err = NewHTMLParser().Parse(incompleteEnd)
	if err != nil {
		t.Fatal(err)
	}
	math = svgNodesNamed(document, "math")[0]
	mrow = svgNodesNamed(document, "mrow")[0]
	if math.EndPos != 19 || mrow.EndPos != 19 || len(mrow.Children) != 1 || mrow.Children[0].Value != "x" || mrow.Children[0].StartPos != 12 || mrow.Children[0].EndPos != 19 {
		t.Fatalf("Incomplete MathML end did not extend preceding text through EOF: math=%#v mrow=%#v", math, mrow)
	}

	const incompleteCDATA = `<math><![CDATA[x]]`
	document, err = NewHTMLParser().Parse(incompleteCDATA)
	if err != nil {
		t.Fatal(err)
	}
	math = svgNodesNamed(document, "math")[0]
	if len(math.Children) != 1 || math.Children[0].Value != "x]]" || math.Children[0].StartPos != 6 || math.Children[0].EndPos != 18 || math.EndPos != 18 {
		t.Fatalf("Incomplete MathML CDATA recovery mismatch: %#v", math)
	}

	const incompleteStart = `<div><math><mrow>x<mi a="`
	document, err = NewHTMLParser().Parse(incompleteStart)
	if err != nil {
		t.Fatal(err)
	}
	div = svgNodesNamed(document, "div")[0]
	math = svgNodesNamed(document, "math")[0]
	mrow = svgNodesNamed(document, "mrow")[0]
	if len(svgNodesNamed(document, "mi")) != 0 || div.EndPos != len(incompleteStart) || math.EndPos != len(incompleteStart) || mrow.EndPos != len(incompleteStart) || len(mrow.Children) != 1 || mrow.Children[0].Value != "x" || mrow.Children[0].StartPos != 17 || mrow.Children[0].EndPos != len(incompleteStart) {
		t.Fatalf("Incomplete MathML start emitted a phantom child or lost EOF range: div=%#v math=%#v mrow=%#v", div, math, mrow)
	}
}

func TestParseMathMLForeignBreakoutAndFontContrast(t *testing.T) {
	const block = `<math><mrow>a<div id=h>b</div>c</mrow><mi>d</mi></math>z`
	document, err := NewHTMLParser().Parse(block)
	if err != nil {
		t.Fatal(err)
	}
	math := svgNodesNamed(document, "math")[0]
	mrow := svgNodesNamed(document, "mrow")[0]
	div := svgNodeByID(t, document, "h")
	mi := svgNodesNamed(document, "mi")[0]
	assertFormattingNode(t, math, "math", "a", 0, 13)
	assertFormattingNode(t, mrow, "mrow", "a", 6, 13)
	assertFormattingNode(t, div, "div", "b", 13, 30)
	assertFormattingNode(t, mi, "mi", "d", 38, 48)
	if div.NamespaceURI != htmlNamespaceURI || mi.NamespaceURI != htmlNamespaceURI || len(parsedBodyChildren(document)) != 5 || parsedBodyChildren(document)[0] != math || parsedBodyChildren(document)[1] != div || parsedBodyChildren(document)[2].Value != "c" || parsedBodyChildren(document)[3] != mi || parsedBodyChildren(document)[4].Value != "z" {
		t.Fatalf("MathML breakout did not permanently reprocess in HTML mode: %#v", parsedBodyChildren(document))
	}

	const plainFont = `<math><mrow><font>x</font></mrow></math>z`
	document, err = NewHTMLParser().Parse(plainFont)
	if err != nil {
		t.Fatal(err)
	}
	font := svgNodesNamed(document, "font")[0]
	if font.NamespaceURI != mathMLTestNamespaceURI || font.Parent == nil || font.Parent.Name != "mrow" {
		t.Fatalf("Plain font must remain MathML foreign content: %#v", font)
	}

	const attributedFont = `<math><mrow><font color=red>x</font></mrow></math>z`
	document, err = NewHTMLParser().Parse(attributedFont)
	if err != nil {
		t.Fatal(err)
	}
	math = svgNodesNamed(document, "math")[0]
	mrow = svgNodesNamed(document, "mrow")[0]
	font = svgNodesNamed(document, "font")[0]
	assertFormattingNode(t, math, "math", "", 0, 12)
	assertFormattingNode(t, mrow, "mrow", "", 6, 12)
	assertFormattingNode(t, font, "font", "x", 12, 36)
	if font.NamespaceURI != htmlNamespaceURI || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].Value != "z" {
		t.Fatalf("Attributed font did not break out of MathML: %#v", parsedBodyChildren(document))
	}
}

func TestParseMathMLNonBreakoutHTMLSpecialContrasts(t *testing.T) {
	const content = `<math id=m><mrow id=r><section id=s>x</section><form id=f>y</form><font id=q>z</font></mrow></math>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	math := svgNodeByID(t, document, "m")
	mrow := svgNodeByID(t, document, "r")
	for _, id := range []string{"s", "f", "q"} {
		node := svgNodeByID(t, document, id)
		if node == nil || node.NamespaceURI != mathMLTestNamespaceURI || node.Parent != mrow {
			t.Fatalf("Non-breakout HTML-special spelling #%s escaped MathML: %#v", id, node)
		}
	}
	if math.TextContent != "xyz" || mrow.TextContent != "xyz" {
		t.Fatalf("Non-breakout MathML descendants lost text/structure: math=%#v mrow=%#v", math, mrow)
	}
}

func TestParseMathMLBreakoutTracksNamespaceQualifiedElementIdentity(t *testing.T) {
	const sameName = `<math><mrow><div>x</div><mrow id=h><span id=s>y</mrow>z</span>`
	document, err := NewHTMLParser().Parse(sameName)
	if err != nil {
		t.Fatal(err)
	}
	math := svgNodesNamed(document, "math")[0]
	foreignMRow := svgNodesNamed(math, "mrow")[0]
	htmlMRow := svgNodeByID(t, document, "h")
	span := svgNodeByID(t, document, "s")
	assertFormattingNode(t, math, "math", "", 0, 12)
	assertFormattingNode(t, foreignMRow, "mrow", "", 6, 12)
	assertFormattingNode(t, htmlMRow, "mrow", "y", 24, 54)
	assertFormattingNode(t, span, "span", "y", 35, 47)
	if foreignMRow.NamespaceURI != mathMLTestNamespaceURI || htmlMRow.NamespaceURI != htmlNamespaceURI || span.Parent != htmlMRow || len(parsedBodyChildren(document)) != 4 || parsedBodyChildren(document)[3].Value != "z" || parsedBodyChildren(document)[3].StartPos != 54 || parsedBodyChildren(document)[3].EndPos != 55 {
		t.Fatalf("MathML/HTML same-name identity collided after breakout: %#v", parsedBodyChildren(document))
	}

	const staleOwnerEnd = `<math id=m><mrow id=r><span id=h>x</math><mi id=i></mi>z`
	document, err = NewHTMLParser().Parse(staleOwnerEnd)
	if err != nil {
		t.Fatal(err)
	}
	math = svgNodeByID(t, document, "m")
	foreignMRow = svgNodeByID(t, document, "r")
	span = svgNodeByID(t, document, "h")
	htmlMI := svgNodeByID(t, document, "i")
	assertFormattingNode(t, math, "math", "", 0, 22)
	assertFormattingNode(t, foreignMRow, "mrow", "", 11, 22)
	assertFormattingNode(t, span, "span", "xz", 22, 56)
	assertFormattingNode(t, htmlMI, "mi", "", 41, 55)
	if span.NamespaceURI != htmlNamespaceURI || htmlMI.NamespaceURI != htmlNamespaceURI || htmlMI.Parent != span {
		t.Fatalf("Stale MathML owner end crossed a live HTML breakout child: span=%#v mi=%#v", span, htmlMI)
	}

	if document, parseErr := NewHTMLParser().Parse(`<math><mrow><div>x</div></mrow></math><section>y`); parseErr != nil || len(svgNodesNamed(document, "section")) != 1 || svgNodesNamed(document, "section")[0].EndPos != 48 {
		t.Fatalf("MathML breakout implicit-document EOF mismatch: doc=%#v err=%v", document, parseErr)
	}
}

func TestParseMathMLForeignEndBreakoutAndFormattingExit(t *testing.T) {
	for _, test := range []struct {
		name, content, emitted string
		textStart              int
	}{
		{"p", `<math><mrow>x</p>y</mrow></math>z`, "p", 17},
		{"br", `<math><mrow>x</br>y</mrow></math>z`, "br", 18},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(test.content)
			if err != nil {
				t.Fatal(err)
			}
			math := svgNodesNamed(document, "math")[0]
			mrow := svgNodesNamed(document, "mrow")[0]
			emitted := svgNodesNamed(document, test.emitted)
			assertFormattingNode(t, math, "math", "x", 0, 13)
			assertFormattingNode(t, mrow, "mrow", "x", 6, 13)
			if len(emitted) != 1 || emitted[0].NamespaceURI != htmlNamespaceURI || emitted[0].StartPos != 0 || emitted[0].EndPos != 0 || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].Value != "yz" || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].StartPos != test.textStart {
				t.Fatalf("MathML </%s> breakout mismatch: %#v", test.emitted, parsedBodyChildren(document))
			}
		})
	}

	const formatting = `<b><math><mi /></math>x</b>y`
	document, err := NewHTMLParser().Parse(formatting)
	if err != nil {
		t.Fatal(err)
	}
	bold := svgNodesNamed(document, "b")[0]
	math := svgNodesNamed(document, "math")[0]
	mi := svgNodesNamed(document, "mi")[0]
	assertFormattingNode(t, bold, "b", "x", 0, 27)
	assertFormattingNode(t, math, "math", "", 3, 22)
	assertFormattingNode(t, mi, "mi", "", 9, 15)
	if mi.NamespaceURI != mathMLTestNamespaceURI || bold.Children[len(bold.Children)-1].Value != "x" || parsedBodyChildren(document)[len(parsedBodyChildren(document))-1].Value != "y" {
		t.Fatalf("MathML exit corrupted active formatting: bold=%#v doc=%#v", bold, parsedBodyChildren(document))
	}
}

func TestParseMathMLForeignBreakoutStartAllowlist(t *testing.T) {
	breakout := []string{
		"b", "big", "blockquote", "body", "br", "center", "code", "dd", "div", "dl", "dt", "em", "embed",
		"h1", "h2", "h3", "h4", "h5", "h6", "head", "hr", "i", "img", "li", "listing", "menu", "meta",
		"nobr", "ol", "p", "pre", "ruby", "s", "small", "span", "strong", "strike", "sub", "sup", "tt", "u", "ul", "var",
	}
	for _, name := range breakout {
		t.Run(name, func(t *testing.T) {
			content := `<math><mrow>a<` + name + ` id=h>x`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			math := svgNodesNamed(document, "math")[0]
			mrow := svgNodesNamed(document, "mrow")[0]
			if math.EndPos != 13 || mrow.EndPos != 13 || math.TextContent != "a" || mrow.TextContent != "a" {
				t.Fatalf("<%s> did not pop MathML ancestors at its token start: math=%#v mrow=%#v", name, math, mrow)
			}
			if html := svgNodeByID(t, document, "h"); html != nil && html.NamespaceURI != htmlNamespaceURI {
				t.Fatalf("MathML breakout <%s> was not reprocessed as HTML: %#v", name, html)
			}
		})
	}
}

func bestMathMLParseDuration(t *testing.T, content string) time.Duration {
	t.Helper()
	best := time.Duration(1<<63 - 1)
	for attempt := 0; attempt < 3; attempt++ {
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

func TestParseMathMLForeignIslandScaling(t *testing.T) {
	build := func(n int) string {
		return strings.Repeat(`<math><mrow><mi xlink:href=x /></mrow></math>`, n)
	}
	smallInput, largeInput := build(1000), build(4000)
	_ = bestMathMLParseDuration(t, smallInput)
	small := bestMathMLParseDuration(t, smallInput)
	large := bestMathMLParseDuration(t, largeInput)
	if large > 10*small && large-small > 100*time.Millisecond {
		t.Fatalf("Closed MathML parsing scaled superlinearly: small=%s large=%s", small, large)
	}
}
