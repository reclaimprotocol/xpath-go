package xpath_test

import (
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryDocumentTypeNodesMatchBrowserNamesAndRanges(t *testing.T) {
	testCases := []struct {
		name         string
		document     string
		wantNodeName string
	}{
		{name: "case-insensitive keyword and name", document: `<!dOcTyPe HTML>`, wantNodeName: "html"},
		{name: "missing name", document: `<!DOCTYPE>`, wantNodeName: ""},
		{name: "EOF after keyword", document: `<!DOCTYPE`, wantNodeName: ""},
		{name: "EOF in name", document: `<!DOCTYPE HTML`, wantNodeName: "html"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(`/node()[1]`, testCase.document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].NodeType != 10 || results[0].NodeName != testCase.wantNodeName {
				t.Fatalf("Expected one browser DocumentType named %q, got %#v", testCase.wantNodeName, results)
			}
			result := results[0]
			if result.Value != "" || result.TextContent != "" {
				t.Fatalf("Expected empty DocumentType value/textContent, got %#v", result)
			}
			if result.StartLocation != 0 || result.EndLocation != len(testCase.document) {
				t.Fatalf("Expected exact original byte range 0:%d, got %d:%d", len(testCase.document), result.StartLocation, result.EndLocation)
			}
		})
	}
}

func TestQueryContinuesAfterMalformedDoctypeIdentifiers(t *testing.T) {
	declarations := []string{
		`<!DOCTYPE html PUBLIC>`,
		`<!DOCTYPE html PUBLIC id>`,
		`<!DOCTYPE html SYSTEM>`,
		`<!DOCTYPE html SYSTEM id>`,
		`<!DOCTYPE html PUBLIK "id">`,
		`<!DOCTYPE html FOO>`,
	}

	for _, declaration := range declarations {
		t.Run(declaration, func(t *testing.T) {
			document := declaration + `<p>tail</p>`
			results, err := xpath.Query(`//p/text()`, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].TextContent != "tail" {
				t.Fatalf("Expected continuation paragraph text, got %#v", results)
			}
			wantStart := len(declaration) + len(`<p>`)
			if results[0].StartLocation != wantStart || results[0].EndLocation != wantStart+len("tail") {
				t.Fatalf("Expected exact continuation text range %d:%d, got %#v", wantStart, wantStart+len("tail"), results[0])
			}
		})
	}
}

func TestQueryGreaterThanAbruptlyClosesQuotedDoctypeIdentifier(t *testing.T) {
	const document = `<!DOCTYPE html PUBLIC "a>b"><p>tail</p>`

	results, err := xpath.Query(`//text()[.='b">']`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != `b">` {
		t.Fatalf("Expected remainder text after abrupt greater-than, got %#v", results)
	}
	wantStart := len(`<!DOCTYPE html PUBLIC "a>`)
	wantEnd := strings.Index(document, `<p>`)
	if results[0].StartLocation != wantStart || results[0].EndLocation != wantEnd {
		t.Fatalf("Expected exact remainder range %d:%d, got %#v", wantStart, wantEnd, results[0])
	}
}

func TestQueryDoctypeQuotedIdentifierAtEOFDoesNotError(t *testing.T) {
	const document = `<!DOCTYPE html PUBLIC "id`

	results, err := xpath.Query(`/node()[1]`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].NodeType != 10 || results[0].NodeName != "html" {
		t.Fatalf("Expected EOF-emitted html DocumentType, got %#v", results)
	}
	if results[0].StartLocation != 0 || results[0].EndLocation != len(document) {
		t.Fatalf("Expected exact EOF range 0:%d, got %#v", len(document), results[0])
	}
}

func TestQueryCDATAInHTMLReturnsBogusCommentsWithExactRanges(t *testing.T) {
	testCases := []struct {
		name      string
		token     string
		wantValue string
	}{
		{name: "complete", token: `<![CDATA[payload]]>`, wantValue: `[CDATA[payload]]`},
		{name: "first greater-than terminates comment", token: `<![CDATA[foo>`, wantValue: `[CDATA[foo`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document := `<div>before` + testCase.token + `after<span>tail</span></div>`
			results, err := xpath.Query(`//div/node()[2]`, document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(results) != 1 || results[0].NodeName != "#comment" || results[0].TextContent != testCase.wantValue {
				t.Fatalf("Expected recovered comment %q, got %#v", testCase.wantValue, results)
			}
			wantStart := strings.Index(document, testCase.token)
			if results[0].StartLocation != wantStart || results[0].EndLocation != wantStart+len(testCase.token) {
				t.Fatalf("Expected exact comment range %d:%d, got %#v", wantStart, wantStart+len(testCase.token), results[0])
			}
			if source := document[results[0].StartLocation:results[0].EndLocation]; source != testCase.token {
				t.Fatalf("Expected exact original declaration source, got %q", source)
			}

			continuation, err := xpath.Query(`//span/text()`, document)
			if err != nil || len(continuation) != 1 || continuation[0].TextContent != "tail" {
				t.Fatalf("Expected parsing continuation: results=%#v err=%v", continuation, err)
			}
		})
	}
}

func TestQueryCDATAInHTMLAtEOFReturnsCommentWithSourceByteRange(t *testing.T) {
	const document = `<div>before<![CDATA[payload`

	results, err := xpath.Query(`//div/node()[2]`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].NodeName != "#comment" || results[0].TextContent != `[CDATA[payload` {
		t.Fatalf("Expected EOF CDATA-like bogus comment, got %#v", results)
	}
	wantStart := strings.Index(document, `<![CDATA[`)
	if results[0].StartLocation != wantStart || results[0].EndLocation != len(document) {
		t.Fatalf("Expected exact EOF range %d:%d, got %#v", wantStart, len(document), results[0])
	}
	if source := document[results[0].StartLocation:results[0].EndLocation]; source != `<![CDATA[payload` {
		t.Fatalf("Expected exact original EOF declaration, got %q", source)
	}
}

func TestQueryIgnoredBodyDoctypeCoalescesTextWithOriginalRange(t *testing.T) {
	const document = `<div>x<!DOCTYPE html>y</div>`

	results, err := xpath.Query(`//div/text()`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "xy" {
		t.Fatalf("Expected one coalesced xy text node, got %#v", results)
	}
	wantStart := len(`<div>`)
	wantEnd := strings.Index(document, `</div>`)
	if results[0].StartLocation != wantStart || results[0].EndLocation != wantEnd {
		t.Fatalf("Expected source range %d:%d spanning ignored doctype, got %#v", wantStart, wantEnd, results[0])
	}
	if source := document[results[0].StartLocation:results[0].EndLocation]; source != `x<!DOCTYPE html>y` {
		t.Fatalf("Expected exact original range through ignored token, got %q", source)
	}
}

func TestQueryKeepsFirstDoctypeAndContinuesPastIgnoredSecond(t *testing.T) {
	const document = `<!DOCTYPE html><!DOCTYPE svg><p>x</p>`

	doctypes, err := xpath.Query(`/node()[1]`, document)
	if err != nil {
		t.Fatalf("DocumentType query returned an error: %v", err)
	}
	if len(doctypes) != 1 || doctypes[0].NodeType != 10 || doctypes[0].NodeName != "html" {
		t.Fatalf("Expected first html DocumentType, got %#v", doctypes)
	}
	if doctypes[0].StartLocation != 0 || doctypes[0].EndLocation != len(`<!DOCTYPE html>`) {
		t.Fatalf("Expected exact first doctype range, got %#v", doctypes[0])
	}

	paragraph, err := xpath.Query(`//p/text()`, document)
	if err != nil || len(paragraph) != 1 || paragraph[0].TextContent != "x" {
		t.Fatalf("Expected paragraph continuation: results=%#v err=%v", paragraph, err)
	}
	wantTextStart := strings.Index(document, `<p>`) + len(`<p>`)
	if paragraph[0].StartLocation != wantTextStart || paragraph[0].EndLocation != wantTextStart+1 {
		t.Fatalf("Expected exact paragraph text range, got %#v", paragraph[0])
	}

	ignored, err := xpath.Query(`/node()[2][not(self::*)]`, document)
	if err != nil {
		t.Fatalf("Ignored second doctype query returned an error: %v", err)
	}
	if len(ignored) != 0 {
		t.Fatalf("Expected second svg doctype to be absent, got %#v", ignored)
	}
}

func TestQueryPostContentDoctypeDoesNotAffectParagraphLocations(t *testing.T) {
	const document = `<p>x</p><!DOCTYPE html>`

	results, err := xpath.Query(`//p/text()`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "x" {
		t.Fatalf("Expected one paragraph text result, got %#v", results)
	}
	if results[0].StartLocation != len(`<p>`) || results[0].EndLocation != len(`<p>x`) {
		t.Fatalf("Expected paragraph text range to exclude trailing doctype, got %#v", results[0])
	}

	ignored, err := xpath.Query(`/node()[2]`, document)
	if err != nil {
		t.Fatalf("Ignored trailing doctype query returned an error: %v", err)
	}
	if len(ignored) != 0 {
		t.Fatalf("Expected no document node for the trailing doctype, got %#v", ignored)
	}
}

func TestQueryEOFDoctypeInsideBodyExtendsElementButNotTextRange(t *testing.T) {
	const document = `<div>x<!DOCTYPE`

	divs, err := xpath.Query(`//div`, document)
	if err != nil {
		t.Fatalf("Div query returned an error: %v", err)
	}
	if len(divs) != 1 || divs[0].TextContent != "x" || divs[0].StartLocation != 0 || divs[0].EndLocation != len(document) {
		t.Fatalf("Expected div range through ignored EOF doctype, got %#v", divs)
	}

	texts, err := xpath.Query(`//div/text()`, document)
	if err != nil {
		t.Fatalf("Text query returned an error: %v", err)
	}
	if len(texts) != 1 || texts[0].TextContent != "x" || texts[0].StartLocation != len(`<div>`) || texts[0].EndLocation != len(`<div>x`) {
		t.Fatalf("Expected emitted text range to remain on x, got %#v", texts)
	}
}

func TestQueryCommentBeforeInitialDoctypeStillReturnsAcceptedDoctype(t *testing.T) {
	const document = `<!--lead--><!DOCTYPE html><p>x</p>`

	results, err := xpath.Query(`/node()[2]`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].NodeType != 10 || results[0].NodeName != "html" {
		t.Fatalf("Expected accepted html DocumentType after comment, got %#v", results)
	}
	wantStart := len(`<!--lead-->`)
	wantEnd := wantStart + len(`<!DOCTYPE html>`)
	if results[0].StartLocation != wantStart || results[0].EndLocation != wantEnd {
		t.Fatalf("Expected exact doctype range %d:%d, got %#v", wantStart, wantEnd, results[0])
	}
}

func TestQueryIgnoredBodyDoctypePreservesWhitespaceInCoalescedText(t *testing.T) {
	const document = `x<!DOCTYPE html> y`

	results, err := xpath.Query(`//text()`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].NodeName != "#text" || results[0].TextContent != "x y" {
		t.Fatalf("Expected one coalesced text value %q, got %#v", "x y", results)
	}
	if results[0].StartLocation != 0 || results[0].EndLocation != len(document) {
		t.Fatalf("Expected full source range 0:%d, got %#v", len(document), results[0])
	}
	if source := document[results[0].StartLocation:results[0].EndLocation]; source != document {
		t.Fatalf("Expected exact original source through ignored doctype, got %q", source)
	}
}

func TestQueryIgnoredDuplicateDoctypeDoesNotExposePreBodyWhitespace(t *testing.T) {
	const document = `<!DOCTYPE html><!DOCTYPE svg> <p>x</p>`

	whitespace, err := xpath.Query(`/text()`, document)
	if err != nil {
		t.Fatalf("Document text query returned an error: %v", err)
	}
	if len(whitespace) != 0 {
		t.Fatalf("Expected ignored duplicate doctype and pre-body whitespace to emit no document text, got %#v", whitespace)
	}

	paragraph, err := xpath.Query(`//p/text()`, document)
	if err != nil || len(paragraph) != 1 || paragraph[0].TextContent != "x" {
		t.Fatalf("Expected paragraph continuation: results=%#v err=%v", paragraph, err)
	}
	wantStart := strings.Index(document, `<p>`) + len(`<p>`)
	if paragraph[0].StartLocation != wantStart || paragraph[0].EndLocation != wantStart+1 {
		t.Fatalf("Expected exact paragraph text range, got %#v", paragraph[0])
	}
}

func TestQueryIgnoredMissingEndTagBeforeDoctypeKeepsDoctypeInitial(t *testing.T) {
	const document = `</> <!DOCTYPE html><p>x</p>`

	whitespace, err := xpath.Query(`/text()`, document)
	if err != nil {
		t.Fatalf("Document text query returned an error: %v", err)
	}
	if len(whitespace) != 0 {
		t.Fatalf("Expected ignored </> and pre-doctype whitespace to expose no text, got %#v", whitespace)
	}

	nodes, err := xpath.Query(`/node()[1]`, document)
	if err != nil || len(nodes) != 1 || nodes[0].NodeName != "html" || nodes[0].NodeType != 10 {
		t.Fatalf("Expected html DocumentType to remain the first document node: results=%#v err=%v", nodes, err)
	}
	wantStart := strings.Index(document, `<!DOCTYPE`)
	wantEnd := wantStart + len(`<!DOCTYPE html>`)
	if nodes[0].StartLocation != wantStart || nodes[0].EndLocation != wantEnd {
		t.Fatalf("Expected exact doctype range %d:%d, got %#v", wantStart, wantEnd, nodes[0])
	}
}
