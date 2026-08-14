package utils

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func parsedHeadChildren(document *types.Node) []*types.Node {
	for _, node := range document.Children {
		if node.Type != types.ElementNode || node.Name != "html" {
			continue
		}
		for _, child := range node.Children {
			if child.Type == types.ElementNode && child.Name == "head" {
				return child.Children
			}
		}
	}
	return nil
}

func TestParseRejectsBinaryInput(t *testing.T) {
	testCases := map[string]string{
		"control bytes":  "\x01\x02text",
		"invalid UTF-8":  "\xff\xfe",
		"gzip signature": "\x1f\x8bnot-a-complete-stream",
	}

	for name, content := range testCases {
		t.Run(name, func(t *testing.T) {
			parser := NewHTMLParser()
			node, err := parser.Parse(content)
			if err == nil {
				t.Fatalf("Expected binary input error, got node %#v", node)
			}
			if !strings.Contains(err.Error(), "binary input") {
				t.Fatalf("Expected descriptive binary input error, got %q", err)
			}
		})
	}
}

func TestParseRejectsGzipInput(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte("<html><body>content</body></html>")); err != nil {
		t.Fatalf("Could not create gzip input: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Could not finish gzip input: %v", err)
	}

	parser := NewHTMLParser()
	node, err := parser.Parse(compressed.String())
	if err == nil {
		t.Fatalf("Expected binary input error, got node %#v", node)
	}
	if !strings.Contains(err.Error(), "binary input") {
		t.Fatalf("Expected descriptive binary input error, got %q", err)
	}
}

func TestParseClosesStandaloneElementAtEOFLikeBrowser(t *testing.T) {
	const content = `<div>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Whole-document parsing must close an open element at EOF: %v", err)
	}
	children := parsedBodyChildren(document)
	if len(children) != 1 || children[0].Name != "div" || children[0].StartPos != 0 || children[0].EndPos != len(content) {
		t.Fatalf("Expected one EOF-closed div in the implicit body, got %#v", children)
	}
}

func TestParseIgnoresRootOrphanClosingTagLikeBrowser(t *testing.T) {
	const content = `</orphan>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Root orphan end tag must be ignored: %v", err)
	}
	if len(parsedBodyChildren(document)) != 0 {
		t.Fatalf("Root orphan end tag emitted body nodes: %#v", parsedBodyChildren(document))
	}
}

func TestParseRecoversBogusCommentsLikeBrowser(t *testing.T) {
	testCases := []struct {
		name        string
		declaration string
		wantValue   string
	}{
		{name: "unknown declaration", declaration: `<!x>`, wantValue: "x"},
		{name: "CDATA in HTML", declaration: `<![CDATA[payload]]>`, wantValue: "[CDATA[payload]]"},
		{name: "entity declaration", declaration: `<!ENTITY example "value">`, wantValue: `ENTITY example "value"`},
		{name: "empty declaration", declaration: `<!>`, wantValue: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := `<div>before` + testCase.declaration + `after<span>tail</span></div>`

			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 {
				t.Fatalf("Expected one body element, got %#v", parsedBodyChildren(document))
			}

			div := parsedBodyChildren(document)[0]
			if div.TextContent != "beforeaftertail" {
				t.Fatalf("Expected comments to be excluded from text content, got %q", div.TextContent)
			}
			if len(div.Children) != 4 || div.Children[1].Type != types.CommentNode {
				t.Fatalf("Expected text, comment, text, and span children, got %#v", div.Children)
			}

			comment := div.Children[1]
			if comment.Value != testCase.wantValue {
				t.Fatalf("Expected comment value %q, got %q", testCase.wantValue, comment.Value)
			}
			if source := content[comment.StartPos:comment.EndPos]; source != testCase.declaration {
				t.Fatalf("Expected original declaration source %q, got %q", testCase.declaration, source)
			}
			if div.Children[3].Name != "span" {
				t.Fatalf("Expected parsing to continue with span, got %q", div.Children[3].Name)
			}
		})
	}
}

func TestParseRecoversBogusCommentAtEOF(t *testing.T) {
	const content = `<!unfinished`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(document.Children) != 2 || document.Children[0].Type != types.CommentNode || document.Children[1].Name != "html" {
		t.Fatalf("Expected one comment node, got %#v", document.Children)
	}
	comment := document.Children[0]
	if comment.Value != "unfinished" || comment.StartPos != 0 || comment.EndPos != len(content) {
		t.Fatalf("Expected EOF-terminated comment with original range, got %#v", comment)
	}
}

func TestParseRecoversCommentTokensLikeBrowser(t *testing.T) {
	testCases := []struct {
		name      string
		token     string
		wantValue string
	}{
		{name: "empty comment", token: `<!---->`, wantValue: ""},
		{name: "abrupt empty comment", token: `<!-->`, wantValue: ""},
		{name: "abrupt empty comment with dash", token: `<!--->`, wantValue: ""},
		{name: "incorrectly closed comment", token: `<!--foo--!>`, wantValue: "foo"},
		{name: "nested comment opener", token: `<!--foo<!--bar-->`, wantValue: `foo<!--bar`},
		{name: "double dash inside comment", token: `<!--foo--bar-->`, wantValue: `foo--bar`},
		{name: "triple dash close", token: `<!--foo--->`, wantValue: `foo-`},
		{name: "invalid question mark target becomes bogus comment", token: `<?1xml version="1.0"?>`, wantValue: `?1xml version="1.0"?`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Chromium and jsdom both produce text, comment, text, and span nodes
			// for these inputs, and parsing continues after the recovered token.
			content := `<div>before` + testCase.token + `after<span>tail</span></div>`

			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 {
				t.Fatalf("Expected one body element, got %#v", parsedBodyChildren(document))
			}

			div := parsedBodyChildren(document)[0]
			if div.TextContent != "beforeaftertail" {
				t.Fatalf("Expected comment to be excluded from text content, got %q", div.TextContent)
			}
			if len(div.Children) != 4 {
				t.Fatalf("Expected text, comment, text, and span children, got %#v", div.Children)
			}

			comment := div.Children[1]
			if comment.Type != types.CommentNode || comment.Name != "#comment" {
				t.Fatalf("Expected a comment node, got %#v", comment)
			}
			if comment.Value != testCase.wantValue {
				t.Fatalf("Expected comment value %q, got %q", testCase.wantValue, comment.Value)
			}
			if source := content[comment.StartPos:comment.EndPos]; source != testCase.token {
				t.Fatalf("Expected original token source %q, got %q", testCase.token, source)
			}
			if span := div.Children[3]; span.Name != "span" || span.TextContent != "tail" {
				t.Fatalf("Expected parsing to continue with span, got %#v", span)
			}
		})
	}
}

func TestParseRecoversCommentAtEOFLikeBrowser(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		wantValue string
	}{
		{name: "comment start at EOF", content: `<!--`, wantValue: ""},
		{name: "comment start dash at EOF", content: `<!---`, wantValue: ""},
		{name: "comment data at EOF", content: `<!--unfinished`, wantValue: "unfinished"},
		{name: "comment end dash at EOF", content: `<!--foo-`, wantValue: "foo"},
		{name: "comment end at EOF", content: `<!--foo--`, wantValue: "foo"},
		{name: "comment end bang at EOF", content: `<!--foo--!`, wantValue: "foo"},
		{name: "comment less than bang at EOF", content: `<!--foo<!`, wantValue: "foo<!"},
		{name: "nested opener at EOF", content: `<!--foo<!--`, wantValue: `foo<!`},
		{name: "invalid question mark target bogus comment at EOF", content: `<?1unfinished`, wantValue: `?1unfinished`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(document.Children) != 2 || document.Children[0].Type != types.CommentNode || document.Children[1].Name != "html" {
				t.Fatalf("Expected one comment node, got %#v", document.Children)
			}

			comment := document.Children[0]
			if comment.Value != testCase.wantValue {
				t.Fatalf("Expected comment value %q, got %q", testCase.wantValue, comment.Value)
			}
			if comment.StartPos != 0 || comment.EndPos != len(testCase.content) {
				t.Fatalf("Expected original range 0:%d, got %d:%d", len(testCase.content), comment.StartPos, comment.EndPos)
			}
			if source := testCase.content[comment.StartPos:comment.EndPos]; source != testCase.content {
				t.Fatalf("Expected original source %q, got %q", testCase.content, source)
			}
		})
	}
}

func TestParseNormalizesCommentInputLikeBrowser(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		wantValue string
	}{
		{name: "NUL in comment", content: "<!--a\x00b-->", wantValue: "a\uFFFDb"},
		{name: "NUL in declaration", content: "<!a\x00b>", wantValue: "a\uFFFDb"},
		{name: "NUL after question mark", content: "<?a\x00b>", wantValue: "?a\uFFFDb"},
		{name: "CR in comment", content: "<!--a\rb-->", wantValue: "a\nb"},
		{name: "CRLF in comment", content: "<!--a\r\nb-->", wantValue: "a\nb"},
		{name: "CR in declaration", content: "<!a\rb>", wantValue: "a\nb"},
		{name: "UTF-8 in comment", content: "<!--a🙂b-->", wantValue: "a🙂b"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(document.Children) != 2 || document.Children[0].Type != types.CommentNode || document.Children[1].Name != "html" {
				t.Fatalf("Expected one comment node, got %#v", document.Children)
			}

			comment := document.Children[0]
			if comment.Value != testCase.wantValue {
				t.Fatalf("Expected comment value %q, got %q", testCase.wantValue, comment.Value)
			}
			if comment.StartPos != 0 || comment.EndPos != len(testCase.content) {
				t.Fatalf("Expected original byte range 0:%d, got %d:%d", len(testCase.content), comment.StartPos, comment.EndPos)
			}
			if source := testCase.content[comment.StartPos:comment.EndPos]; source != testCase.content {
				t.Fatalf("Expected original source %q, got %q", testCase.content, source)
			}
		})
	}
}

func TestParseHandlesNULByTokenizerStateLikeBrowser(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		assertion func(t *testing.T, document *types.Node)
	}{
		{
			name:    "ignored in data text",
			content: "<div>a\x00b</div>",
			assertion: func(t *testing.T, document *types.Node) {
				t.Helper()
				if got := parsedBodyChildren(document)[0].TextContent; got != "ab" {
					t.Fatalf("Expected data-state NUL to be ignored, got %q", got)
				}
			},
		},
		{
			name:    "replaced in quoted attribute",
			content: "<div title=\"a\x00b\">payload</div>",
			assertion: func(t *testing.T, document *types.Node) {
				t.Helper()
				if got := parsedBodyChildren(document)[0].Attributes["title"]; got != "a\uFFFDb" {
					t.Fatalf("Expected quoted-attribute NUL replacement, got %q", got)
				}
			},
		},
		{
			name:    "replaced in unquoted attribute",
			content: "<div title=a\x00b>payload</div>",
			assertion: func(t *testing.T, document *types.Node) {
				t.Helper()
				if got := parsedBodyChildren(document)[0].Attributes["title"]; got != "a\uFFFDb" {
					t.Fatalf("Expected unquoted-attribute NUL replacement, got %q", got)
				}
			},
		},
		{
			name:    "replaced in raw text",
			content: "<textarea>a\x00b</textarea>",
			assertion: func(t *testing.T, document *types.Node) {
				t.Helper()
				if got := parsedBodyChildren(document)[0].TextContent; got != "a\uFFFDb" {
					t.Fatalf("Expected raw-text NUL replacement, got %q", got)
				}
			},
		},
		{
			name:    "replaced in tag name",
			content: "<a\x00b>payload</a\x00b>",
			assertion: func(t *testing.T, document *types.Node) {
				t.Helper()
				if got := parsedBodyChildren(document)[0].Name; got != "a\uFFFDb" {
					t.Fatalf("Expected tag-name NUL replacement, got %q", got)
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			testCase.assertion(t, document)
		})
	}
}

func TestParseTracksCRLFAsOneLogicalNewline(t *testing.T) {
	testCases := []struct {
		name      string
		separator string
	}{
		{name: "CR", separator: "\r"},
		{name: "CRLF", separator: "\r\n"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := "<!--a" + testCase.separator + "b--><span>tail</span>"
			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(document.Children) != 2 || document.Children[0].Type != types.CommentNode || len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "span" {
				t.Fatalf("Expected a document comment and body span, got document=%#v body=%#v", document.Children, parsedBodyChildren(document))
			}

			span := parsedBodyChildren(document)[0]
			if span.StartLine != 2 || span.StartColumn != 5 {
				t.Fatalf("Expected span at logical position 2:5, got %d:%d", span.StartLine, span.StartColumn)
			}
			if source := content[span.StartPos:span.EndPos]; source != `<span>tail</span>` {
				t.Fatalf("Expected exact original span source, got %q", source)
			}
		})
	}
}

func TestParseAllowsAdjacentHTMLAttributes(t *testing.T) {
	const content = `<div id="target"class='primary'disabled data-value="42">payload</div>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 {
		t.Fatalf("Expected one body element, got %d", len(parsedBodyChildren(document)))
	}

	element := parsedBodyChildren(document)[0]
	wantAttributes := map[string]string{
		"id":         "target",
		"class":      "primary",
		"disabled":   "",
		"data-value": "42",
	}
	for name, wantValue := range wantAttributes {
		if value, ok := element.Attributes[name]; !ok || value != wantValue {
			t.Errorf("Attribute %q: expected %q, got %q (present=%t)", name, wantValue, value, ok)
		}
	}

	if source := content[element.StartPos:element.EndPos]; source != content {
		t.Fatalf("Expected original source %q, got %q", content, source)
	}
}

func TestParseIgnoresMetaClosingTagLikeBrowser(t *testing.T) {
	const content = `<html><head><meta name="description"content="sample"></meta><title>Page</title></head><body><div id="target">payload</div></body></html>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}

	if len(document.Children) != 1 || document.Children[0].Name != "html" {
		t.Fatalf("Expected one html root, got %#v", document.Children)
	}
	html := document.Children[0]
	if len(html.Children) != 2 || html.Children[0].Name != "head" || html.Children[1].Name != "body" {
		t.Fatalf("Expected head and body children, got %#v", html.Children)
	}

	head := html.Children[0]
	if len(head.Children) != 2 || head.Children[0].Name != "meta" || head.Children[1].Name != "title" {
		t.Fatalf("Expected meta and title children, got %#v", head.Children)
	}
	meta := head.Children[0]
	if source := content[meta.StartPos:meta.EndPos]; source != `<meta name="description"content="sample">` {
		t.Fatalf("Expected original meta source, got %q", source)
	}

	target := html.Children[1].Children[0]
	if source := content[target.StartPos:target.EndPos]; source != `<div id="target">payload</div>` {
		t.Fatalf("Expected original target source, got %q", source)
	}
}

func TestParseIgnoresVoidElementClosingTags(t *testing.T) {
	voidElements := []string{
		"area", "base", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr",
	}

	for _, tagName := range voidElements {
		t.Run(tagName, func(t *testing.T) {
			content := `<div>before</` + tagName + `><span>after</span></div>`

			parser := NewHTMLParser()
			document, err := parser.Parse(content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].TextContent != "beforeafter" {
				t.Fatalf("Expected void closer to be ignored, got %#v", parsedBodyChildren(document))
			}

			span := parsedBodyChildren(document)[0].Children[1]
			if span.Name != "span" {
				t.Fatalf("Expected span after ignored closer, got %q", span.Name)
			}
			if source := content[span.StartPos:span.EndPos]; source != `<span>after</span>` {
				t.Fatalf("Expected original span source, got %q", source)
			}
		})
	}
}

func TestParseTreatsBrClosingTagAsStartTagLikeBrowser(t *testing.T) {
	const content = `<div>before</br><span>after</span></div>`

	parser := NewHTMLParser()
	document, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 {
		t.Fatalf("Expected one body element, got %#v", parsedBodyChildren(document))
	}

	div := parsedBodyChildren(document)[0]
	if len(div.Children) != 3 || div.Children[1].Name != "br" || div.Children[2].Name != "span" {
		t.Fatalf("Expected text, recovered br, and span children, got %#v", div.Children)
	}
	br := div.Children[1]
	if source := content[br.StartPos:br.EndPos]; source != `</br>` {
		t.Fatalf("Expected recovered br location to reference original token, got %q", source)
	}
	if br.ContentStart != br.EndPos || br.ContentEnd != br.EndPos {
		t.Fatalf("Expected recovered br to have empty content at %d, got %d:%d", br.EndPos, br.ContentStart, br.ContentEnd)
	}
}

func TestParseImplicitlyClosesTableRows(t *testing.T) {
	testCases := []struct {
		name           string
		content        string
		wantRowTexts   []string
		wantRowSources []string
	}{
		{
			name:           "new row closes current row",
			content:        `<table><tr><td>first</td></tr><tr><tr><td>third</td></tr></table>`,
			wantRowTexts:   []string{"first", "", "third"},
			wantRowSources: []string{`<tr><td>first</td></tr>`, `<tr>`, `<tr><td>third</td></tr>`},
		},
		{
			name:           "table end closes current row",
			content:        `<table><tr><td>only</td></table>`,
			wantRowTexts:   []string{"only"},
			wantRowSources: []string{`<tr><td>only</td>`},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parser := NewHTMLParser()
			document, err := parser.Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}

			if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "table" {
				t.Fatalf("Expected one body table, got %#v", parsedBodyChildren(document))
			}

			table := parsedBodyChildren(document)[0]
			if len(table.Children) != 1 || table.Children[0].Name != "tbody" || table.Children[0].StartPos != 0 || table.Children[0].EndPos != 0 {
				t.Fatalf("Expected one synthetic 0:0 tbody, got %#v", table.Children)
			}
			rows := table.Children[0].Children
			if len(rows) != len(testCase.wantRowTexts) {
				t.Fatalf("Expected %d rows, got %d", len(testCase.wantRowTexts), len(rows))
			}
			for index, wantText := range testCase.wantRowTexts {
				row := rows[index]
				if row.Name != "tr" {
					t.Fatalf("Child %d: expected tr, got %q", index, row.Name)
				}
				if row.TextContent != wantText {
					t.Fatalf("Row %d: expected text %q, got %q", index, wantText, row.TextContent)
				}
				if row.StartPos < 0 || row.EndPos < row.StartPos || row.EndPos > len(testCase.content) {
					t.Fatalf("Row %d: invalid original-source range %d:%d", index, row.StartPos, row.EndPos)
				}
				if source := testCase.content[row.StartPos:row.EndPos]; source != testCase.wantRowSources[index] {
					t.Fatalf("Row %d: expected original source %q, got %q", index, testCase.wantRowSources[index], source)
				}
			}
		})
	}
}
