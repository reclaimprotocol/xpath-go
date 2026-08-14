package utils

import (
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseDoctypeEOFAndNameStatesLikeBrowser(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		wantDocName string
	}{
		{name: "case-insensitive keyword and name", content: `<!dOcTyPe HTML>`, wantDocName: "html"},
		{name: "missing name", content: `<!DOCTYPE>`, wantDocName: ""},
		{name: "EOF immediately after keyword", content: `<!DOCTYPE`, wantDocName: ""},
		{name: "EOF after keyword whitespace", content: `<!DOCTYPE `, wantDocName: ""},
		{name: "EOF in name", content: `<!DOCTYPE HTML`, wantDocName: "html"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(document.Children) != 2 || document.Children[0].Type != types.DocumentTypeNode || document.Children[1].Name != "html" {
				t.Fatalf("Expected exactly one document type, got %#v", document.Children)
			}

			doctype := document.Children[0]
			if doctype.Name != testCase.wantDocName {
				t.Fatalf("Expected browser document type name %q, got %q", testCase.wantDocName, doctype.Name)
			}
			if doctype.Value != "" || doctype.TextContent != "" {
				t.Fatalf("Expected DocumentType nodeValue/textContent to be empty, got value=%q text=%q", doctype.Value, doctype.TextContent)
			}
			if doctype.StartPos != 0 || doctype.EndPos != len(testCase.content) {
				t.Fatalf("Expected exact original range 0:%d, got %d:%d", len(testCase.content), doctype.StartPos, doctype.EndPos)
			}
		})
	}
}

func TestParseMalformedDoctypeIdentifiersContinueLikeBrowser(t *testing.T) {
	testCases := []struct {
		name        string
		declaration string
	}{
		{name: "abrupt public keyword", declaration: `<!DOCTYPE html PUBLIC>`},
		{name: "unquoted public identifier", declaration: `<!DOCTYPE html PUBLIC id>`},
		{name: "abrupt system keyword", declaration: `<!DOCTYPE html SYSTEM>`},
		{name: "unquoted system identifier", declaration: `<!DOCTYPE html SYSTEM id>`},
		{name: "invalid public keyword", declaration: `<!DOCTYPE html PUBLIK "id">`},
		{name: "invalid sequence after name", declaration: `<!DOCTYPE html FOO>`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := testCase.declaration + `<p>tail</p>`
			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(document.Children) != 2 || document.Children[0].Type != types.DocumentTypeNode || document.Children[1].Name != "html" || len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" {
				t.Fatalf("Expected document type followed by an implicit-body p, got document=%#v body=%#v", document.Children, parsedBodyChildren(document))
			}

			doctype := document.Children[0]
			if doctype.Name != "html" || doctype.StartPos != 0 || doctype.EndPos != len(testCase.declaration) {
				t.Fatalf("Expected recovered html document type over 0:%d, got %#v", len(testCase.declaration), doctype)
			}
			paragraph := parsedBodyChildren(document)[0]
			if paragraph.TextContent != "tail" || paragraph.StartPos != len(testCase.declaration) || paragraph.EndPos != len(content) {
				t.Fatalf("Expected exact continuation paragraph range, got %#v", paragraph)
			}
			if source := content[paragraph.StartPos:paragraph.EndPos]; source != `<p>tail</p>` {
				t.Fatalf("Expected exact continuation source, got %q", source)
			}
		})
	}
}

func TestParseGreaterThanAbruptlyClosesQuotedDoctypeIdentifierLikeBrowser(t *testing.T) {
	const content = `<!DOCTYPE html PUBLIC "a>b"><p>tail</p>`
	const emittedDoctype = `<!DOCTYPE html PUBLIC "a>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 2 || document.Children[0].Type != types.DocumentTypeNode || document.Children[1].Name != "html" || len(parsedBodyChildren(document)) != 2 || parsedBodyChildren(document)[0].Type != types.TextNode || parsedBodyChildren(document)[1].Name != "p" {
		t.Fatalf("Expected document type plus body remainder text and p, got document=%#v body=%#v", document.Children, parsedBodyChildren(document))
	}

	doctype := document.Children[0]
	if doctype.Name != "html" || doctype.StartPos != 0 || doctype.EndPos != len(emittedDoctype) {
		t.Fatalf("Expected document type to end at abrupt greater-than, got %#v", doctype)
	}
	remainder := parsedBodyChildren(document)[0]
	if remainder.Value != `b">` || remainder.StartPos != len(emittedDoctype) || remainder.EndPos != strings.Index(content, `<p>`) {
		t.Fatalf("Expected exact remainder text after emitted document type, got %#v", remainder)
	}
	if source := content[remainder.StartPos:remainder.EndPos]; source != `b">` {
		t.Fatalf("Expected exact remainder source, got %q", source)
	}
}

func TestParseDoctypeQuotedIdentifierAtEOFLikeBrowser(t *testing.T) {
	const content = `<!DOCTYPE html PUBLIC "id`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 2 || document.Children[0].Type != types.DocumentTypeNode || document.Children[1].Name != "html" {
		t.Fatalf("Expected one EOF-emitted document type, got %#v", document.Children)
	}
	doctype := document.Children[0]
	if doctype.Name != "html" || doctype.StartPos != 0 || doctype.EndPos != len(content) {
		t.Fatalf("Expected html document type over exact EOF range, got %#v", doctype)
	}
}

func TestParseCDATAInHTMLUsesBogusCommentRecovery(t *testing.T) {
	testCases := []struct {
		name      string
		token     string
		wantValue string
	}{
		{name: "complete CDATA-like declaration", token: `<![CDATA[payload]]>`, wantValue: `[CDATA[payload]]`},
		{name: "first greater-than ends bogus comment", token: `<![CDATA[foo>`, wantValue: `[CDATA[foo`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := `<div>before` + testCase.token + `after<span>tail</span></div>`
			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
				t.Fatalf("Expected one body div, got %#v", parsedBodyChildren(document))
			}

			div := parsedBodyChildren(document)[0]
			if len(div.Children) != 4 || div.Children[1].Type != types.CommentNode || div.Children[3].Name != "span" {
				t.Fatalf("Expected text, comment, text, and continuation span, got %#v", div.Children)
			}
			comment := div.Children[1]
			wantStart := strings.Index(content, testCase.token)
			if comment.Value != testCase.wantValue || comment.StartPos != wantStart || comment.EndPos != wantStart+len(testCase.token) {
				t.Fatalf("Expected recovered comment %q at %d:%d, got %#v", testCase.wantValue, wantStart, wantStart+len(testCase.token), comment)
			}
			if source := content[comment.StartPos:comment.EndPos]; source != testCase.token {
				t.Fatalf("Expected exact declaration source %q, got %q", testCase.token, source)
			}
			span := div.Children[3]
			if span.TextContent != "tail" || content[span.StartPos:span.EndPos] != `<span>tail</span>` {
				t.Fatalf("Expected parsing to continue at exact span source, got %#v", span)
			}
		})
	}
}

func TestParseCDATAInHTMLAtEOFBecomesCommentAndClosesAncestors(t *testing.T) {
	const content = `<div>before<![CDATA[payload`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
		t.Fatalf("Expected one body div, got %#v", parsedBodyChildren(document))
	}
	div := parsedBodyChildren(document)[0]
	if div.StartPos != 0 || div.EndPos != len(content) || len(div.Children) != 2 {
		t.Fatalf("Expected div to close at EOF around text and comment, got %#v", div)
	}
	comment := div.Children[1]
	wantStart := strings.Index(content, `<![CDATA[`)
	if comment.Type != types.CommentNode || comment.Value != `[CDATA[payload` || comment.StartPos != wantStart || comment.EndPos != len(content) {
		t.Fatalf("Expected EOF CDATA-like comment over %d:%d, got %#v", wantStart, len(content), comment)
	}
	if source := content[comment.StartPos:comment.EndPos]; source != `<![CDATA[payload` {
		t.Fatalf("Expected exact EOF declaration source, got %q", source)
	}
}

func TestParseIgnoresDoctypeInsideBodyAndCoalescesTextLikeBrowser(t *testing.T) {
	const content = `<div>x<!DOCTYPE html>y</div>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
		t.Fatalf("Expected ignored body doctype and one body div, got %#v", parsedBodyChildren(document))
	}

	div := parsedBodyChildren(document)[0]
	if div.TextContent != "xy" || len(div.Children) != 1 || div.Children[0].Type != types.TextNode {
		t.Fatalf("Expected one coalesced xy text node, got %#v", div)
	}
	text := div.Children[0]
	wantStart := len(`<div>`)
	wantEnd := strings.Index(content, `</div>`)
	if text.Value != "xy" || text.StartPos != wantStart || text.EndPos != wantEnd {
		t.Fatalf("Expected coalesced text range %d:%d, got %#v", wantStart, wantEnd, text)
	}
	if source := content[text.StartPos:text.EndPos]; source != `x<!DOCTYPE html>y` {
		t.Fatalf("Expected original range to span ignored doctype, got %q", source)
	}
}

func TestParseKeepsOnlyFirstInitialDoctypeLikeBrowser(t *testing.T) {
	const content = `<!DOCTYPE html><!DOCTYPE svg><p>x</p>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 2 || document.Children[0].Type != types.DocumentTypeNode || document.Children[1].Name != "html" || len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" {
		t.Fatalf("Expected first doctype and one body p only, got document=%#v body=%#v", document.Children, parsedBodyChildren(document))
	}

	doctype := document.Children[0]
	if doctype.Name != "html" || doctype.StartPos != 0 || doctype.EndPos != len(`<!DOCTYPE html>`) {
		t.Fatalf("Expected only the first html doctype at its exact range, got %#v", doctype)
	}
	paragraph := parsedBodyChildren(document)[0]
	wantParagraphStart := strings.Index(content, `<p>`)
	if paragraph.TextContent != "x" || paragraph.StartPos != wantParagraphStart || paragraph.EndPos != len(content) {
		t.Fatalf("Expected parsing to continue with exact paragraph range, got %#v", paragraph)
	}
}

func TestParseIgnoresPostContentDoctypeLikeBrowser(t *testing.T) {
	const content = `<p>x</p><!DOCTYPE html>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" {
		t.Fatalf("Expected post-content doctype to be ignored, got %#v", parsedBodyChildren(document))
	}
	paragraph := parsedBodyChildren(document)[0]
	if paragraph.StartPos != 0 || paragraph.EndPos != len(`<p>x</p>`) || paragraph.TextContent != "x" {
		t.Fatalf("Expected paragraph location to exclude ignored trailing doctype, got %#v", paragraph)
	}
	if len(paragraph.Children) != 1 || paragraph.Children[0].StartPos != len(`<p>`) || paragraph.Children[0].EndPos != len(`<p>x`) {
		t.Fatalf("Expected exact paragraph text location, got %#v", paragraph.Children)
	}
}

func TestParseIgnoresEOFDoctypeInsideBodyAndClosesAncestorLikeBrowser(t *testing.T) {
	const content = `<div>x<!DOCTYPE`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
		t.Fatalf("Expected one implicitly closed body div, got %#v", parsedBodyChildren(document))
	}
	div := parsedBodyChildren(document)[0]
	if div.TextContent != "x" || div.StartPos != 0 || div.EndPos != len(content) {
		t.Fatalf("Expected div to extend through ignored EOF doctype, got %#v", div)
	}
	if len(div.Children) != 1 || div.Children[0].Type != types.TextNode {
		t.Fatalf("Expected exactly one text child, got %#v", div.Children)
	}
	text := div.Children[0]
	if text.Value != "x" || text.StartPos != len(`<div>`) || text.EndPos != len(`<div>x`) {
		t.Fatalf("Expected text location to remain on the emitted x only, got %#v", text)
	}
}

func TestParseAllowsCommentBeforeInitialDoctypeLikeBrowser(t *testing.T) {
	const content = `<!--lead--><!DOCTYPE html><p>x</p>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 3 || document.Children[0].Type != types.CommentNode || document.Children[1].Type != types.DocumentTypeNode || document.Children[2].Name != "html" || len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" {
		t.Fatalf("Expected comment, accepted doctype, and implicit-body p, got document=%#v body=%#v", document.Children, parsedBodyChildren(document))
	}
	comment := document.Children[0]
	if comment.Value != "lead" || comment.StartPos != 0 || comment.EndPos != len(`<!--lead-->`) {
		t.Fatalf("Expected exact leading comment, got %#v", comment)
	}
	doctype := document.Children[1]
	wantStart := len(`<!--lead-->`)
	wantEnd := wantStart + len(`<!DOCTYPE html>`)
	if doctype.Name != "html" || doctype.StartPos != wantStart || doctype.EndPos != wantEnd {
		t.Fatalf("Expected accepted initial doctype range %d:%d, got %#v", wantStart, wantEnd, doctype)
	}
}

func TestParseIgnoredBodyDoctypePreservesFollowingWhitespaceInCoalescedText(t *testing.T) {
	const content = `x<!DOCTYPE html> y`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Type != types.TextNode {
		t.Fatalf("Expected one coalesced body text node, got %#v", parsedBodyChildren(document))
	}
	text := parsedBodyChildren(document)[0]
	if text.Value != "x y" || text.TextContent != "x y" {
		t.Fatalf("Expected browser text value %q, got %#v", "x y", text)
	}
	if text.StartPos != 0 || text.EndPos != len(content) {
		t.Fatalf("Expected full original range 0:%d, got %d:%d", len(content), text.StartPos, text.EndPos)
	}
	if source := content[text.StartPos:text.EndPos]; source != content {
		t.Fatalf("Expected source range to span ignored doctype, got %q", source)
	}
}

func TestParseIgnoredDuplicateDoctypeDoesNotCreateLeadingWhitespaceText(t *testing.T) {
	const content = `<!DOCTYPE html><!DOCTYPE svg> <p>x</p>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 2 || document.Children[0].Type != types.DocumentTypeNode || document.Children[1].Name != "html" || len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" {
		t.Fatalf("Expected first doctype and body p without a whitespace node, got document=%#v body=%#v", document.Children, parsedBodyChildren(document))
	}
	doctype := document.Children[0]
	if doctype.Name != "html" || doctype.StartPos != 0 || doctype.EndPos != len(`<!DOCTYPE html>`) {
		t.Fatalf("Expected exact retained first doctype, got %#v", doctype)
	}
	paragraph := parsedBodyChildren(document)[0]
	wantStart := strings.Index(content, `<p>`)
	if paragraph.TextContent != "x" || paragraph.StartPos != wantStart || paragraph.EndPos != len(content) {
		t.Fatalf("Expected p to start after discarded pre-body whitespace at %d, got %#v", wantStart, paragraph)
	}
}

func TestParseIgnoredMissingEndTagBeforeDoctypeDoesNotCreateLeadingWhitespaceText(t *testing.T) {
	const content = `</> <!DOCTYPE html><p>x</p>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 2 || document.Children[0].Type != types.DocumentTypeNode || document.Children[1].Name != "html" || len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" {
		t.Fatalf("Expected accepted doctype and body p without leading whitespace text, got document=%#v body=%#v", document.Children, parsedBodyChildren(document))
	}
	doctype := document.Children[0]
	wantStart := strings.Index(content, `<!DOCTYPE`)
	wantEnd := wantStart + len(`<!DOCTYPE html>`)
	if doctype.Name != "html" || doctype.StartPos != wantStart || doctype.EndPos != wantEnd {
		t.Fatalf("Expected exact retained doctype range %d:%d, got %#v", wantStart, wantEnd, doctype)
	}
}
