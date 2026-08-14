package utils

import (
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func scriptUTF16Length(value string) int {
	return len(utf16.Encode([]rune(value)))
}

func assertScriptText(t *testing.T, content, wantValue string, wantTextStart, wantTextEnd, wantElementEnd int) *types.Node {
	t.Helper()
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	scripts := listElements(document, "script")
	if len(scripts) != 1 {
		t.Fatalf("Expected one script, got %#v", scripts)
	}
	script := scripts[0]
	if script.TextContent != wantValue || len(script.Children) != 1 || script.Children[0].Type != types.TextNode {
		t.Fatalf("Expected one script text node %q, got %#v", wantValue, script)
	}
	text := script.Children[0]
	if text.Value != wantValue || text.StartPos != wantTextStart || text.EndPos != wantTextEnd {
		t.Fatalf("Expected script text %q at %d:%d, got %#v", wantValue, wantTextStart, wantTextEnd, text)
	}
	if script.StartPos != 0 || script.EndPos != wantElementEnd {
		t.Fatalf("Expected script range 0:%d, got %d:%d", wantElementEnd, script.StartPos, script.EndPos)
	}
	return script
}

func requireParagraphAfterScript(t *testing.T, script *types.Node) {
	t.Helper()
	root := script
	for root.Parent != nil {
		root = root.Parent
	}
	paragraphs := listElements(root, "p")
	if len(paragraphs) != 1 || paragraphs[0].TextContent != "ok" || paragraphs[0].StartPos < script.EndPos {
		t.Fatalf("Expected parsing to continue with p after script, got %#v", paragraphs)
	}
}

func TestParseScriptEscapedAndDoubleEscapedTransitionsLikeBrowser(t *testing.T) {
	testCases := []struct {
		name, raw string
	}{
		{name: "reviewer double escaped repro", raw: `<!--<script></script>-->`},
		{name: "escaped end returns to script data", raw: `<!--a-->b`},
		{name: "similar hyphenated name remains escaped text", raw: `<!--<script-x></script-x>-->`},
		{name: "non-delimited longer name remains escaped text", raw: `<!--<scriptx></scriptx>-->`},
		{name: "temporary buffer compares ASCII case-insensitively", raw: `<!--<ScRiPt></sCrIpT>-->`},
		{name: "escape start single dash returns to script data", raw: `<!-x-->`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := `<script>` + testCase.raw + `</script><p>ok</p>`
			textStart := len(`<script>`)
			textEnd := textStart + len(testCase.raw)
			script := assertScriptText(t, content, testCase.raw, textStart, textEnd, textEnd+len(`</script>`))
			if source := content[textStart:textEnd]; source != testCase.raw {
				t.Fatalf("Expected exact raw script source %q, got %q", testCase.raw, source)
			}
			requireParagraphAfterScript(t, script)
			text := script.Children[0]
			if text.StartLine != 1 || text.StartColumn != 9 || text.EndLine != 1 || text.EndColumn != 9+scriptUTF16Length(testCase.raw) {
				t.Fatalf("Expected parse5 script coordinates 1:9-1:%d, got %d:%d-%d:%d", 9+scriptUTF16Length(testCase.raw), text.StartLine, text.StartColumn, text.EndLine, text.EndColumn)
			}
		})
	}
}

func TestParseScriptEscapedStatesReplaceNUL(t *testing.T) {
	testCases := []struct {
		name, raw, want string
	}{
		{name: "escaped", raw: "<!--a\x00b-->", want: "<!--a\uFFFDb-->"},
		{name: "double escaped", raw: "<!--<script>a\x00b</script>-->", want: "<!--<script>a\uFFFDb</script>-->"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := `<script>` + testCase.raw + `</script>`
			textStart := len(`<script>`)
			textEnd := textStart + len(testCase.raw)
			script := assertScriptText(t, content, testCase.want, textStart, textEnd, len(content))
			if source := content[script.Children[0].StartPos:script.Children[0].EndPos]; source != testCase.raw {
				t.Fatalf("Expected exact NUL-bearing source %q, got %q", testCase.raw, source)
			}
		})
	}
}

func TestParseScriptEscapedStatesCloseAtEOFWithExactLocations(t *testing.T) {
	testCases := []struct {
		name, raw string
	}{
		{name: "escaped", raw: `<!--é&amp;`},
		{name: "double escaped", raw: `<!--<script>é&amp;`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			content := `<script>` + testCase.raw
			script := assertScriptText(t, content, testCase.raw, len(`<script>`), len(content), len(content))
			text := script.Children[0]
			if source := content[text.StartPos:text.EndPos]; source != testCase.raw {
				t.Fatalf("Expected exact multibyte EOF source %q, got %q", testCase.raw, source)
			}
			wantEndColumn := 9 + scriptUTF16Length(testCase.raw)
			if text.StartLine != 1 || text.StartColumn != 9 || text.EndLine != 1 || text.EndColumn != wantEndColumn {
				t.Fatalf("Expected parse5 EOF coordinates 1:9-1:%d, got %d:%d-%d:%d", wantEndColumn, text.StartLine, text.StartColumn, text.EndLine, text.EndColumn)
			}
		})
	}
}

func TestParseScriptImmediateEscapeEndSequencesLikeBrowser(t *testing.T) {
	testCases := []struct {
		name, content, wantRaw string
		wantElementEnd         int
	}{
		{
			name:    "immediate comment end returns to data",
			content: `<script><!--><script></script><p>ok</p>`, wantRaw: `<!--><script>`, wantElementEnd: 30,
		},
		{
			name:    "immediate dash comment end returns to data",
			content: `<script><!---><script></script><p>ok</p>`, wantRaw: `<!---><script>`, wantElementEnd: 31,
		},
		{
			name:    "double escaped dash end returns to data and closes",
			content: `<script><!--<script>--></script></script><p>ok</p>`, wantRaw: `<!--<script>-->`, wantElementEnd: 32,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			textStart := len(`<script>`)
			textEnd := textStart + len(testCase.wantRaw)
			script := assertScriptText(t, testCase.content, testCase.wantRaw, textStart, textEnd, testCase.wantElementEnd)
			if source := testCase.content[textStart:textEnd]; source != testCase.wantRaw {
				t.Fatalf("Expected exact immediate escape source %q, got %q", testCase.wantRaw, source)
			}
			requireParagraphAfterScript(t, script)
			text := script.Children[0]
			if text.StartLine != 1 || text.StartColumn != 9 || text.EndLine != 1 || text.EndColumn != 9+scriptUTF16Length(testCase.wantRaw) {
				t.Fatalf("Expected parse5 coordinates 1:9-1:%d, got %d:%d-%d:%d", 9+scriptUTF16Length(testCase.wantRaw), text.StartLine, text.StartColumn, text.EndLine, text.EndColumn)
			}
		})
	}
}

func TestParseScriptEscapedAppropriateEndTagDelimiters(t *testing.T) {
	testCases := []struct {
		name, content  string
		wantElementEnd int
	}{
		{name: "whitespace and attributes", content: `<script><!--x</SCRIPT   ><p>ok</p>`, wantElementEnd: 25},
		{name: "trailing slash", content: `<script><!--x</script/><p>ok</p>`, wantElementEnd: 23},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			script := assertScriptText(t, testCase.content, `<!--x`, len(`<script>`), len(`<script><!--x`), testCase.wantElementEnd)
			requireParagraphAfterScript(t, script)
		})
	}
}

func TestParseScriptEscapedIncompleteEndCandidatesAtEOF(t *testing.T) {
	t.Run("escaped appropriate candidate is discarded", func(t *testing.T) {
		const content = `<script><!--x</script `
		script := assertScriptText(t, content, `<!--x`, len(`<script>`), len(content), len(content))
		if source := content[script.Children[0].StartPos:script.Children[0].EndPos]; source != `<!--x</script ` {
			t.Fatalf("Expected text range through discarded end-tag source, got %q", source)
		}
	})

	t.Run("double escaped candidate remains text", func(t *testing.T) {
		const content = `<script><!--<script>x</script `
		const raw = `<!--<script>x</script `
		assertScriptText(t, content, raw, len(`<script>`), len(content), len(content))
	})
}

func TestParseNestedScriptEscapedEOFClosesAncestorWithExactLocations(t *testing.T) {
	const content = `<div><script><!--é`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}
	if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "div" {
		t.Fatalf("Expected one outer div, got %#v", parsedBodyChildren(document))
	}
	div := parsedBodyChildren(document)[0]
	if div.StartPos != 0 || div.EndPos != len(content) || div.TextContent != `<!--é` || len(div.Children) != 1 {
		t.Fatalf("Expected div through EOF 0:%d, got %#v", len(content), div)
	}
	script := div.Children[0]
	if script.Name != "script" || script.StartPos != len(`<div>`) || script.EndPos != len(content) || script.TextContent != `<!--é` || len(script.Children) != 1 {
		t.Fatalf("Expected nested script through EOF, got %#v", script)
	}
	text := script.Children[0]
	if text.StartPos != len(`<div><script>`) || text.EndPos != len(content) || text.Value != `<!--é` {
		t.Fatalf("Expected exact nested multibyte text range %d:%d, got %#v", len(`<div><script>`), len(content), text)
	}
	if text.StartLine != 1 || text.StartColumn != 14 || text.EndLine != 1 || text.EndColumn != 19 {
		t.Fatalf("Expected parse5 coordinates 1:14-1:19, got %d:%d-%d:%d", text.StartLine, text.StartColumn, text.EndLine, text.EndColumn)
	}
}

func TestHTMLParserReuseAfterDoubleEscapedScriptEOF(t *testing.T) {
	parser := NewHTMLParser()
	first, err := parser.Parse(`<script><!--<script>é`)
	if err != nil {
		t.Fatalf("First script parse returned an error: %v", err)
	}
	firstScripts := listElements(first, "script")
	if len(firstScripts) != 1 || firstScripts[0].TextContent != `<!--<script>é` {
		t.Fatalf("Expected first double-escaped EOF script, got %#v", firstScripts)
	}
	second, err := parser.Parse(`<p>ok</p>`)
	if err != nil {
		t.Fatalf("Parser reuse returned an error: %v", err)
	}
	paragraphs := listElements(second, "p")
	if len(paragraphs) != 1 || paragraphs[0].TextContent != "ok" {
		t.Fatalf("Expected parser state reset for p, got %#v", paragraphs)
	}
}

func TestParseScriptDoubleEscapedInnerEndTagAfterDashesLikeBrowser(t *testing.T) {
	for dashCount := 1; dashCount <= 3; dashCount++ {
		t.Run(string(rune('0'+dashCount))+" dashes", func(t *testing.T) {
			dashes := strings.Repeat("-", dashCount)
			raw := `<!--<script>` + dashes + `</script>-->`
			content := `<script>` + raw + `</script><p>ok</p>`
			textStart := len(`<script>`)
			textEnd := textStart + len(raw)
			script := assertScriptText(t, content, raw, textStart, textEnd, textEnd+len(`</script>`))
			if source := content[textStart:textEnd]; source != raw {
				t.Fatalf("Expected exact %d-dash raw source %q, got %q", dashCount, raw, source)
			}
			requireParagraphAfterScript(t, script)
			text := script.Children[0]
			if text.StartLine != 1 || text.StartColumn != 9 || text.EndLine != 1 || text.EndColumn != 9+len(raw) {
				t.Fatalf("Expected exact ASCII coordinates 1:9-1:%d, got %d:%d-%d:%d", 9+len(raw), text.StartLine, text.StartColumn, text.EndLine, text.EndColumn)
			}
		})
	}
}

func TestParseIgnoresOrphanScriptEndTagsLikeBrowser(t *testing.T) {
	t.Run("after correctly closed script", func(t *testing.T) {
		const content = `<script>x</script></script><p>ok</p>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		scripts := listElements(document, "script")
		paragraphs := listElements(document, "p")
		if len(scripts) != 1 || len(paragraphs) != 1 {
			t.Fatalf("Expected script and p with orphan close ignored, script=%#v p=%#v", scripts, paragraphs)
		}
		script, paragraph := scripts[0], paragraphs[0]
		if script.TextContent != "x" || script.StartPos != 0 || script.EndPos != len(`<script>x</script>`) {
			t.Fatalf("Expected exact script range 0:%d, got %#v", len(`<script>x</script>`), script)
		}
		if paragraph.TextContent != "ok" || paragraph.StartPos != len(`<script>x</script></script>`) || paragraph.EndPos != len(content) {
			t.Fatalf("Expected p at %d:%d, got %#v", len(`<script>x</script></script>`), len(content), paragraph)
		}
	})

	t.Run("at document root", func(t *testing.T) {
		const content = `</script><p>ok</p>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse returned an error: %v", err)
		}
		if len(parsedBodyChildren(document)) != 1 || parsedBodyChildren(document)[0].Name != "p" {
			t.Fatalf("Expected only p after ignored root script close, got %#v", parsedBodyChildren(document))
		}
		paragraph := parsedBodyChildren(document)[0]
		if paragraph.TextContent != "ok" || paragraph.StartPos != len(`</script>`) || paragraph.EndPos != len(content) {
			t.Fatalf("Expected p at %d:%d, got %#v", len(`</script>`), len(content), paragraph)
		}
	})
}

func TestParseIgnoresNestedOrphanScriptEndTagsLikeBrowser(t *testing.T) {
	testCases := []struct {
		name, content                        string
		wantParagraphStart, wantParagraphEnd int
		wantScript                           bool
	}{
		{
			name: "inside div", content: `<div></script><p>ok</p></div>`,
			wantParagraphStart: 14, wantParagraphEnd: 23,
		},
		{
			name: "after closed script inside div", content: `<div><script>x</script></script><p>ok</p></div>`,
			wantParagraphStart: 32, wantParagraphEnd: 41, wantScript: true,
		},
		{
			name: "inside explicit body", content: `<html><body></script><p>ok</p></body></html>`,
			wantParagraphStart: 21, wantParagraphEnd: 30,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document, err := NewHTMLParser().Parse(testCase.content)
			if err != nil {
				t.Fatalf("Parse returned an error: %v", err)
			}
			paragraph := findFirstElementByName(document, "p")
			if paragraph == nil || paragraph.TextContent != "ok" || paragraph.StartPos != testCase.wantParagraphStart || paragraph.EndPos != testCase.wantParagraphEnd {
				t.Fatalf("Expected p ok at %d:%d, got %#v", testCase.wantParagraphStart, testCase.wantParagraphEnd, paragraph)
			}
			script := findFirstElementByName(document, "script")
			if testCase.wantScript {
				if script == nil || script.TextContent != "x" || script.StartPos != 5 || script.EndPos != 23 {
					t.Fatalf("Expected original nested script x at 5:23, got %#v", script)
				}
			} else if script != nil {
				t.Fatalf("Expected no script element for orphan end tag, got %#v", script)
			}
		})
	}
}

func findFirstElementByName(node *types.Node, name string) *types.Node {
	if node == nil {
		return nil
	}
	if node.Type == types.ElementNode && node.Name == name {
		return node
	}
	for _, child := range node.Children {
		if match := findFirstElementByName(child, name); match != nil {
			return match
		}
	}
	return nil
}
