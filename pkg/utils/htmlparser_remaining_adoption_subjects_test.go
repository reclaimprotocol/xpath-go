package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

var remainingAdoptionSubjects = []string{"big", "code", "font", "s", "small", "strike", "tt", "u"}

// Batch 14D covers non-table in-body adoption for the remaining formatting
// subjects. Table/foster and markers, select, templates, foreign content,
// fragments, and other insertion modes remain deferred.

func TestParseRemainingAdoptionSubjectsCanonicalMisnesting(t *testing.T) {
	for _, subject := range remainingAdoptionSubjects {
		t.Run(subject, func(t *testing.T) {
			content := `<p><` + subject + `>1<i>2</` + subject + `>3</i>4`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			p := formattingElements(document, "p")[0]
			subjects := formattingElements(p, subject)
			italics := formattingElements(p, "i")
			open := `<` + subject + `>`
			iStart := 3 + len(open) + 1
			subjectEndStart := iStart + len(`<i>2`)
			subjectEnd := subjectEndStart + len(`</`+subject+`>`)
			iEnd := subjectEnd + 1 + len(`</i>`)
			assertFormattingNode(t, p, "p", "1234", 0, len(content))
			assertFormattingNode(t, subjects[0], subject, "12", 3, subjectEnd)
			assertFormattingNode(t, italics[0], "i", "2", iStart, subjectEndStart)
			assertFormattingNode(t, italics[1], "i", "3", iStart, iEnd)
			if len(subjects) != 1 || len(italics) != 2 || italics[0].Parent != subjects[0] || italics[1].Parent != p || p.Children[2].Value != "4" {
				t.Fatalf("Unexpected canonical <%s> adoption tree: %#v", subject, p.Children)
			}
		})
	}
}

func TestParseRemainingAdoptionSubjectsFurthestBlock(t *testing.T) {
	for _, subject := range remainingAdoptionSubjects {
		t.Run(subject, func(t *testing.T) {
			content := `<` + subject + `>1<div>2</` + subject + `>3</div>4`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			subjects := formattingElements(document, subject)
			div := formattingElements(document, "div")[0]
			openLen := len(subject) + 2
			endStart := openLen + len(`1<div>2`)
			end := endStart + len(`</`+subject+`>`)
			assertFormattingNode(t, subjects[0], subject, "1", 0, end)
			assertFormattingNode(t, div, "div", "23", openLen+1, len(content)-1)
			assertFormattingNode(t, subjects[1], subject, "2", 0, 0)
			if len(subjects) != 2 || subjects[1].Parent != div || len(div.Children) != 2 || div.Children[1].Value != "3" || div.Children[1].StartPos != end || div.Children[1].EndPos != end+1 || parsedBodyChildren(document)[2].Value != "4" {
				t.Fatalf("Unexpected furthest-block <%s> adoption tree: %#v", subject, parsedBodyChildren(document))
			}
		})
	}
}

func TestParseRemainingAdoptionSubjectsAbsentAndOutOfScopeEnds(t *testing.T) {
	for _, subject := range remainingAdoptionSubjects {
		content := `<p>x</` + subject + `>y`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		p := formattingElements(document, "p")[0]
		assertFormattingNode(t, p, "p", "xy", 0, len(content))
		if len(formattingElements(document, subject)) != 0 || len(p.Children) != 1 || p.Children[0].Type != types.TextNode || p.Children[0].Value != "xy" || p.Children[0].StartPos != 3 || p.Children[0].EndPos != len(content) {
			t.Fatalf("Absent </%s> must be ignored and raw text coalesced, got %#v", subject, p.Children)
		}
	}

	const aggregate = `<p>a</big>b</code>c</font>d</s>e</small>f</strike>g</tt>h</u>i`
	document, err := NewHTMLParser().Parse(aggregate)
	if err != nil {
		t.Fatal(err)
	}
	p := formattingElements(document, "p")[0]
	if p.TextContent != "abcdefghi" || len(p.Children) != 1 || p.Children[0].StartPos != 3 || p.Children[0].EndPos != len(aggregate) {
		t.Fatalf("Absent adoption ends must coalesce one text node across all raw tokens: %#v", p.Children)
	}

	for _, subject := range remainingAdoptionSubjects {
		content := `<` + subject + `><object>x</` + subject + `>y</object>z`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		outer := formattingElements(document, subject)[0]
		object := formattingElements(document, "object")[0]
		assertFormattingNode(t, outer, subject, "xyz", 0, len(content))
		if object.Parent != outer || object.TextContent != "xy" || outer.Children[1].Value != "z" {
			t.Fatalf("Out-of-scope </%s> must be ignored behind object, got %#v", subject, outer.Children)
		}
	}
}

func TestParseFontAdoptionClonePreservesAttributes(t *testing.T) {
	const content = `<font color=red face=x>1<div>2</font>3</div>4`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	fonts := formattingElements(document, "font")
	div := formattingElements(document, "div")[0]
	if len(fonts) != 2 || fonts[0].Parent != parsedBody(document) || fonts[1].Parent != div {
		t.Fatalf("Expected source and replacement font, got %#v", fonts)
	}
	assertFormattingNode(t, fonts[0], "font", "1", 0, 37)
	assertFormattingNode(t, fonts[1], "font", "2", 0, 0)
	assertFormattingNode(t, div, "div", "23", 24, 44)
	for _, font := range fonts {
		if font.Attributes["color"] != "red" || font.Attributes["face"] != "x" || len(font.AttributeOrder) != 2 || font.AttributeOrder[0] != "color" || font.AttributeOrder[1] != "face" {
			t.Fatalf("Font adoption clone lost source attributes/order: %#v", font)
		}
	}
	if &fonts[0].AttributeOrder[0] == &fonts[1].AttributeOrder[0] {
		t.Fatal("Font adoption clone must own an independent attribute-order slice")
	}
}

func TestParseRemainingAdoptionSubjectsIncompleteEndsAndReuse(t *testing.T) {
	parser := NewHTMLParser()
	for _, subject := range remainingAdoptionSubjects {
		content := `<` + subject + `>1<div>2</` + subject
		document, err := parser.Parse(content)
		if err != nil {
			t.Fatalf("Incomplete </%s must be discarded: %v", subject, err)
		}
		subjects := formattingElements(document, subject)
		div := formattingElements(document, "div")
		if len(subjects) != 1 || len(div) != 1 || div[0].Parent != subjects[0] || subjects[0].TextContent != "12" || subjects[0].EndPos != len(content) || div[0].EndPos != len(content) {
			t.Fatalf("Incomplete </%s must not run adoption surgery, subject=%#v div=%#v", subject, subjects, div)
		}
	}
	plain, err := parser.Parse(`z`)
	if err != nil || len(parsedBodyChildren(plain)) != 1 || parsedBodyChildren(plain)[0].Type != types.TextNode || parsedBodyChildren(plain)[0].Value != "z" {
		t.Fatalf("Parser reuse after remaining-subject recovery failed: doc=%#v err=%v", plain, err)
	}
	for _, subject := range remainingAdoptionSubjects {
		if len(formattingElements(plain, subject)) != 0 {
			t.Fatalf("Parser reuse leaked <%s> state", subject)
		}
	}
}

func TestParseRemainingAdoptionSubjectsOffStackEnds(t *testing.T) {
	for _, subject := range remainingAdoptionSubjects {
		t.Run(subject, func(t *testing.T) {
			content := `<p><` + subject + `>1<div>2</div></` + subject + `>3`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatal(err)
			}
			subjects := formattingElements(document, subject)
			div := formattingElements(document, "div")[0]
			endStart := len(content) - len(`</`+subject+`>3`)
			end := endStart + len(`</`+subject+`>`)
			if len(subjects) != 2 || subjects[0].Parent.Name != "p" || subjects[1].Parent != div || subjects[0].TextContent != "1" || subjects[1].TextContent != "2" {
				t.Fatalf("Off-stack </%s> did not preserve reconstructed tree: %#v", subject, parsedBodyChildren(document))
			}
			if len(parsedBodyChildren(document)) != 3 || parsedBodyChildren(document)[2].Type != types.TextNode || parsedBodyChildren(document)[2].Value != "3" || parsedBodyChildren(document)[2].StartPos != end || parsedBodyChildren(document)[2].EndPos != len(content) {
				t.Fatalf("Off-stack </%s> must remove its formatting entry before trailing text: %#v", subject, parsedBodyChildren(document))
			}
		})
	}
}

func TestParseRemainingAdoptionSubjectsCrossIdentityAndOrdinaryInnerNode(t *testing.T) {
	const cross = `<big><code>1</big>2</code>3`
	document, err := NewHTMLParser().Parse(cross)
	if err != nil {
		t.Fatal(err)
	}
	big := formattingElements(document, "big")[0]
	codes := formattingElements(document, "code")
	assertFormattingNode(t, big, "big", "1", 0, 18)
	assertFormattingNode(t, codes[0], "code", "1", 5, 12)
	assertFormattingNode(t, codes[1], "code", "2", 5, 26)
	if len(codes) != 2 || codes[0].Parent != big || codes[1].Parent != parsedBody(document) || parsedBodyChildren(document)[2].Value != "3" {
		t.Fatalf("Cross-subject adoption used the wrong formatting identity: %#v", parsedBodyChildren(document))
	}

	const ordinary = `<tt><span>1<div>2</tt>3</span>4</div>5`
	document, err = NewHTMLParser().Parse(ordinary)
	if err != nil {
		t.Fatal(err)
	}
	tts := formattingElements(document, "tt")
	spans := formattingElements(document, "span")
	div := formattingElements(document, "div")[0]
	if len(tts) != 2 || len(spans) != 1 || tts[0].TextContent != "1" || tts[1].TextContent != "2" || div.TextContent != "234" || parsedBodyChildren(document)[2].Value != "5" {
		t.Fatalf("Ordinary inner node was not removed during tt adoption: %#v", parsedBodyChildren(document))
	}
	if spans[0].EndPos != 17 || tts[1].StartPos != 0 || tts[1].EndPos != 0 {
		t.Fatalf("Wrong ordinary-inner adoption locations: span=%#v clone=%#v", spans[0], tts[1])
	}
}

func TestParseRemainingAdoptionSubjectSurvivingFormattingDescendant(t *testing.T) {
	const content = `<s><b>1<div>2</s>3</b>4</div>5`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	strikes := formattingElements(document, "s")
	bold := formattingElements(document, "b")
	div := formattingElements(document, "div")[0]
	if len(strikes) != 2 || len(bold) != 3 || len(parsedBodyChildren(document)) != 4 {
		t.Fatalf("Expected source s>b, empty b, recovered div, and trailing text; s=%#v b=%#v children=%#v", strikes, bold, parsedBodyChildren(document))
	}
	assertFormattingNode(t, strikes[0], "s", "1", 0, 17)
	assertFormattingNode(t, bold[0], "b", "1", 3, 6)
	assertFormattingNode(t, bold[1], "b", "", 0, 0)
	assertFormattingNode(t, div, "div", "234", 7, 29)
	assertFormattingNode(t, bold[2], "b", "23", 0, 0)
	assertFormattingNode(t, strikes[1], "s", "2", 0, 0)
	if bold[0].Parent != strikes[0] || bold[1].Parent != parsedBody(document) || bold[2].Parent != div || strikes[1].Parent != bold[2] {
		t.Fatalf("Wrong surviving-formatting parents: s=%#v b=%#v", strikes, bold)
	}
	if len(bold[2].Children) != 2 || bold[2].Children[1].Value != "3" || bold[2].Children[1].StartPos != 17 || bold[2].Children[1].EndPos != 18 || div.Children[1].Value != "4" || div.Children[1].StartPos != 22 || div.Children[1].EndPos != 23 || parsedBodyChildren(document)[3].Value != "5" || parsedBodyChildren(document)[3].StartPos != 29 || parsedBodyChildren(document)[3].EndPos != 30 {
		t.Fatalf("Wrong surviving-formatting continuation ranges: div=%#v document=%#v", div.Children, parsedBodyChildren(document))
	}
	if strikes[0].StartLine != 1 || strikes[0].StartColumn != 1 || strikes[0].EndLine != 1 || strikes[0].EndColumn != 18 || bold[0].StartColumn != 4 || bold[0].EndColumn != 7 || div.StartColumn != 8 || div.EndColumn != 30 {
		t.Fatalf("Wrong source coordinates for mixed adoption: s=%#v b=%#v div=%#v", strikes[0], bold[0], div)
	}
}

func TestParseRemainingAdoptionSubjectDeepSurvivingFormattingDescendants(t *testing.T) {
	const content = `<small><b><i>1<div>2</small>3</i>4</b>5</div>6`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	smalls := formattingElements(document, "small")
	bold := formattingElements(document, "b")
	italics := formattingElements(document, "i")
	div := formattingElements(document, "div")[0]
	if len(smalls) != 2 || len(bold) != 3 || len(italics) != 3 || len(parsedBodyChildren(document)) != 4 {
		t.Fatalf("Expected source small>b>i, empty b>i, recovered div, and trailing text; small=%#v b=%#v i=%#v children=%#v", smalls, bold, italics, parsedBodyChildren(document))
	}
	assertFormattingNode(t, smalls[0], "small", "1", 0, 28)
	assertFormattingNode(t, bold[0], "b", "1", 7, 10)
	assertFormattingNode(t, italics[0], "i", "1", 10, 13)
	assertFormattingNode(t, bold[1], "b", "", 0, 0)
	assertFormattingNode(t, italics[1], "i", "", 0, 0)
	assertFormattingNode(t, div, "div", "2345", 14, 45)
	assertFormattingNode(t, bold[2], "b", "234", 0, 0)
	assertFormattingNode(t, italics[2], "i", "23", 0, 0)
	assertFormattingNode(t, smalls[1], "small", "2", 0, 0)
	if bold[0].Parent != smalls[0] || italics[0].Parent != bold[0] || bold[1].Parent != parsedBody(document) || italics[1].Parent != bold[1] || bold[2].Parent != div || italics[2].Parent != bold[2] || smalls[1].Parent != italics[2] {
		t.Fatalf("Wrong deep surviving-formatting parents: small=%#v b=%#v i=%#v", smalls, bold, italics)
	}
	if italics[2].Children[1].Value != "3" || italics[2].Children[1].StartPos != 28 || italics[2].Children[1].EndPos != 29 || bold[2].Children[1].Value != "4" || bold[2].Children[1].StartPos != 33 || bold[2].Children[1].EndPos != 34 || div.Children[1].Value != "5" || div.Children[1].StartPos != 38 || div.Children[1].EndPos != 39 || parsedBodyChildren(document)[3].Value != "6" || parsedBodyChildren(document)[3].StartPos != 45 || parsedBodyChildren(document)[3].EndPos != 46 {
		t.Fatalf("Wrong deep surviving-formatting continuation ranges: div=%#v document=%#v", div.Children, parsedBodyChildren(document))
	}
	if smalls[0].EndColumn != 29 || bold[0].StartColumn != 8 || bold[0].EndColumn != 11 || italics[0].StartColumn != 11 || italics[0].EndColumn != 14 || div.StartColumn != 15 || div.EndColumn != 46 {
		t.Fatalf("Wrong deep source coordinates: small=%#v b=%#v i=%#v div=%#v", smalls[0], bold[0], italics[0], div)
	}
}

func TestParseRemainingAdoptionSubjectReverseCloseOrderKeepsSurvivingWrapper(t *testing.T) {
	const content = `<u><b>1<div>2</u>3</div>4</b>5`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	underlines := formattingElements(document, "u")
	bold := formattingElements(document, "b")
	div := formattingElements(document, "div")[0]
	if len(underlines) != 2 || len(bold) != 2 || len(parsedBodyChildren(document)) != 3 {
		t.Fatalf("Expected source u>b, surviving b wrapper, and trailing text; u=%#v b=%#v children=%#v", underlines, bold, parsedBodyChildren(document))
	}
	assertFormattingNode(t, underlines[0], "u", "1", 0, 17)
	assertFormattingNode(t, bold[0], "b", "1", 3, 6)
	assertFormattingNode(t, bold[1], "b", "234", 0, 0)
	assertFormattingNode(t, div, "div", "23", 7, 24)
	assertFormattingNode(t, underlines[1], "u", "2", 0, 0)
	if bold[0].Parent != underlines[0] || bold[1].Parent != parsedBody(document) || div.Parent != bold[1] || underlines[1].Parent != div {
		t.Fatalf("Reverse close order lost surviving b ownership: u=%#v b=%#v div=%#v", underlines, bold, div)
	}
	if div.Children[1].Value != "3" || div.Children[1].StartPos != 17 || div.Children[1].EndPos != 18 || bold[1].Children[1].Value != "4" || bold[1].Children[1].StartPos != 24 || bold[1].Children[1].EndPos != 25 || parsedBodyChildren(document)[2].Value != "5" || parsedBodyChildren(document)[2].StartPos != 29 || parsedBodyChildren(document)[2].EndPos != 30 {
		t.Fatalf("Wrong reverse-close continuation ranges: b=%#v div=%#v document=%#v", bold[1].Children, div.Children, parsedBodyChildren(document))
	}
	if underlines[0].EndColumn != 18 || bold[0].StartColumn != 4 || bold[0].EndColumn != 7 || div.StartColumn != 8 || div.EndColumn != 25 {
		t.Fatalf("Wrong reverse-close source coordinates: u=%#v b=%#v div=%#v", underlines[0], bold[0], div)
	}
}

func TestParseRemainingAdoptionSubjectSequentialNestedAdoptionRefreshesText(t *testing.T) {
	const content = `<u><b>1<div>2</b>3</u>4</div>5`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	underlines := formattingElements(document, "u")
	bold := formattingElements(document, "b")
	div := formattingElements(document, "div")[0]
	if len(underlines) != 2 || len(bold) != 2 || len(parsedBodyChildren(document)) != 3 {
		t.Fatalf("Expected source u>b, recovered div, and trailing text; u=%#v b=%#v children=%#v", underlines, bold, parsedBodyChildren(document))
	}
	assertFormattingNode(t, underlines[0], "u", "1", 0, 22)
	assertFormattingNode(t, bold[0], "b", "1", 3, 17)
	assertFormattingNode(t, div, "div", "234", 7, 29)
	assertFormattingNode(t, underlines[1], "u", "23", 0, 0)
	assertFormattingNode(t, bold[1], "b", "2", 0, 0)
	if bold[0].Parent != underlines[0] || underlines[1].Parent != div || bold[1].Parent != underlines[1] {
		t.Fatalf("Sequential adoption produced wrong parents: u=%#v b=%#v", underlines, bold)
	}
	if underlines[1].Children[1].Value != "3" || underlines[1].Children[1].StartPos != 17 || underlines[1].Children[1].EndPos != 18 || div.Children[1].Value != "4" || div.Children[1].StartPos != 22 || div.Children[1].EndPos != 23 || parsedBodyChildren(document)[2].Value != "5" {
		t.Fatalf("Sequential adoption continuation/range mismatch: div=%#v document=%#v", div.Children, parsedBodyChildren(document))
	}
}

func TestParseRemainingAdoptionSubjectDoesNotLeakIgnoredEndByName(t *testing.T) {
	const content = `<u><span>1<div>2</u>3<span>x</span>y</div>z`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	underlines := formattingElements(document, "u")
	spans := formattingElements(document, "span")
	div := formattingElements(document, "div")[0]
	if len(underlines) != 2 || len(spans) != 2 || len(parsedBodyChildren(document)) != 3 {
		t.Fatalf("Expected source u>span, recovered div with a new span, and trailing z; u=%#v span=%#v children=%#v", underlines, spans, parsedBodyChildren(document))
	}
	assertFormattingNode(t, underlines[0], "u", "1", 0, 20)
	assertFormattingNode(t, spans[0], "span", "1", 3, 16)
	assertFormattingNode(t, div, "div", "23xy", 10, 42)
	assertFormattingNode(t, underlines[1], "u", "2", 0, 0)
	assertFormattingNode(t, spans[1], "span", "x", 21, 35)
	if spans[0].Parent != underlines[0] || underlines[1].Parent != div || spans[1].Parent != div {
		t.Fatalf("Stale ignored-end state changed span ownership: u=%#v span=%#v", underlines, spans)
	}
	if div.Children[1].Value != "3" || div.Children[1].StartPos != 20 || div.Children[1].EndPos != 21 || div.Children[3].Value != "y" || div.Children[3].StartPos != 35 || div.Children[3].EndPos != 36 || parsedBodyChildren(document)[2].Value != "z" || parsedBodyChildren(document)[2].StartPos != 42 || parsedBodyChildren(document)[2].EndPos != 43 {
		t.Fatalf("Wrong ignored-end identity continuation ranges: div=%#v document=%#v", div.Children, parsedBodyChildren(document))
	}
}

func TestParseRemainingAdoptionSubjectSyntaxEdges(t *testing.T) {
	const uppercase = `<p><font>x</FONT>y`
	document, err := NewHTMLParser().Parse(uppercase)
	if err != nil {
		t.Fatal(err)
	}
	font := formattingElements(document, "font")[0]
	assertFormattingNode(t, font, "font", "x", 3, 17)
	if formattingElements(document, "p")[0].TextContent != "xy" {
		t.Fatalf("Uppercase font end did not use adoption: %#v", document)
	}

	const attrs = `<font>x</font ignored=yes/>y`
	document, err = NewHTMLParser().Parse(attrs)
	if err != nil {
		t.Fatal(err)
	}
	font = formattingElements(document, "font")[0]
	assertFormattingNode(t, font, "font", "x", 0, 27)
	if parsedBodyChildren(document)[1].Value != "y" {
		t.Fatalf("End-tag attributes/solidus changed adoption semantics: %#v", parsedBodyChildren(document))
	}

	const selfClosing = `<u/>x</u>y`
	document, err = NewHTMLParser().Parse(selfClosing)
	if err != nil {
		t.Fatal(err)
	}
	u := formattingElements(document, "u")[0]
	assertFormattingNode(t, u, "u", "x", 0, 9)
	if parsedBodyChildren(document)[1].Value != "y" {
		t.Fatalf("Self-closing flag on non-void u was not ignored: %#v", parsedBodyChildren(document))
	}
}

func TestParseFontAttributeMultibyteAdoptionLocations(t *testing.T) {
	const content = "<font color=red face=x>é\r\n<i>😀</font>z</i>w"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	font := formattingElements(document, "font")[0]
	italics := formattingElements(document, "i")
	assertFormattingNode(t, font, "font", "é\n😀", 0, 41)
	assertFormattingNode(t, italics[0], "i", "😀", 27, 34)
	assertFormattingNode(t, italics[1], "i", "z", 27, 46)
	if font.Attributes["color"] != "red" || font.Attributes["face"] != "x" || font.ContentStart != 23 || font.StartLine != 1 || font.StartColumn != 1 || font.EndLine != 2 || font.EndColumn != 13 || italics[0].StartLine != 2 || italics[0].StartColumn != 1 || italics[0].EndLine != 2 || italics[0].EndColumn != 6 || italics[1].EndLine != 2 || italics[1].EndColumn != 18 {
		t.Fatalf("Wrong font attribute/UTF-16 location metadata: font=%#v i=%#v", font, italics)
	}
	if italics[1].Attributes == nil || font.AttributeOrder[0] != "color" || font.AttributeOrder[1] != "face" {
		t.Fatalf("Formatting source metadata was not preserved: font=%#v clone=%#v", font, italics[1])
	}
}

func TestParseRemainingAdoptionSubjectsExplicitDocumentClosesEOFDescendants(t *testing.T) {
	for _, subject := range remainingAdoptionSubjects {
		t.Run(subject, func(t *testing.T) {
			valid := `<html><body><p><` + subject + `>x</p></body>y</` + subject + `></html>`
			if _, err := NewHTMLParser().Parse(valid); err != nil {
				t.Fatalf("Valid document-end </%s> recovery failed: %v", subject, err)
			}
			content := valid + `<foo>x`
			document, err := NewHTMLParser().Parse(content)
			if err != nil {
				t.Fatalf("Explicit document-end </%s> must EOF-close later foo: %v", subject, err)
			}
			foo := formattingElements(document, "foo")
			if len(foo) != 1 || foo[0].TextContent != "x" || foo[0].EndPos != len(content) {
				t.Fatalf("Explicit document-end </%s> foo recovery mismatch: %#v", subject, foo)
			}
		})
	}
}

func TestParseRemainingAdoptionSubjectsScaling(t *testing.T) {
	if testing.Short() {
		return
	}
	build := func(n int) string {
		var b strings.Builder
		b.Grow(n * 28)
		for i := 0; i < n; i++ {
			subject := remainingAdoptionSubjects[i%len(remainingAdoptionSubjects)]
			b.WriteString(`<p><` + subject + `>1<i>2</` + subject + `>3</i>4`)
		}
		return b.String()
	}
	measure := func(n int) time.Duration {
		content := build(n)
		_, _ = NewHTMLParser().Parse(content)
		best := time.Duration(1<<63 - 1)
		for i := 0; i < 3; i++ {
			start := time.Now()
			if _, err := NewHTMLParser().Parse(content); err != nil {
				t.Fatal(err)
			}
			if d := time.Since(start); d < best {
				best = d
			}
		}
		return best
	}
	small, large := measure(1000), measure(4000)
	ratio := float64(large) / float64(small)
	t.Logf("remaining adoption scaling 1000=%v 4000=%v ratio=%.1fx", small, large, ratio)
	if large > small*10 && large-small > 20*time.Millisecond {
		t.Fatalf("Remaining adoption subjects scaled superlinearly: %.1fx", ratio)
	}
}
