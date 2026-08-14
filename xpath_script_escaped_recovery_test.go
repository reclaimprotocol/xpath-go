package xpath_test

import (
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func assertScriptQuery(t *testing.T, document, wantValue string, wantStart, wantEnd int) {
	t.Helper()
	results, err := xpath.Query(`//script/text()`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].NodeName != "#text" || results[0].TextContent != wantValue {
		t.Fatalf("Expected one script text value %q, got %#v", wantValue, results)
	}
	if results[0].StartLocation != wantStart || results[0].EndLocation != wantEnd {
		t.Fatalf("Expected exact raw range %d:%d, got %#v", wantStart, wantEnd, results[0])
	}
}

func TestQueryScriptEscapedAndDoubleEscapedTransitions(t *testing.T) {
	testCases := []struct {
		name, raw string
	}{
		{name: "reviewer double escaped repro", raw: `<!--<script></script>-->`},
		{name: "escaped end returns to data", raw: `<!--a-->b`},
		{name: "similar hyphenated name", raw: `<!--<script-x></script-x>-->`},
		{name: "non-delimited longer name", raw: `<!--<scriptx></scriptx>-->`},
		{name: "ASCII case-insensitive temporary buffer", raw: `<!--<ScRiPt></sCrIpT>-->`},
		{name: "escape start dash", raw: `<!-x-->`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document := `<script>` + testCase.raw + `</script><p>ok</p>`
			start := len(`<script>`)
			assertScriptQuery(t, document, testCase.raw, start, start+len(testCase.raw))
			paragraphs, err := xpath.Query(`//p/text()`, document)
			if err != nil || len(paragraphs) != 1 || paragraphs[0].TextContent != "ok" {
				t.Fatalf("Expected parsing to continue with p, results=%#v err=%v", paragraphs, err)
			}
		})
	}
}

func TestQueryScriptEscapedStatesReplaceNUL(t *testing.T) {
	testCases := []struct {
		name, raw, want string
	}{
		{name: "escaped", raw: "<!--a\x00b-->", want: "<!--a\uFFFDb-->"},
		{name: "double escaped", raw: "<!--<script>a\x00b</script>-->", want: "<!--<script>a\uFFFDb</script>-->"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			document := `<script>` + testCase.raw + `</script>`
			start := len(`<script>`)
			assertScriptQuery(t, document, testCase.want, start, start+len(testCase.raw))
		})
	}
}

func TestQueryScriptEscapedStatesCloseAtEOF(t *testing.T) {
	for _, raw := range []string{`<!--é&amp;`, `<!--<script>é&amp;`} {
		t.Run(raw, func(t *testing.T) {
			document := `<script>` + raw
			assertScriptQuery(t, document, raw, len(`<script>`), len(document))
		})
	}
}

func TestQueryScriptImmediateEscapeEndSequences(t *testing.T) {
	testCases := []struct {
		name, document, wantRaw string
	}{
		{name: "immediate comment end", document: `<script><!--><script></script><p>ok</p>`, wantRaw: `<!--><script>`},
		{name: "immediate dash comment end", document: `<script><!---><script></script><p>ok</p>`, wantRaw: `<!---><script>`},
		{name: "double escaped dash end", document: `<script><!--<script>--></script></script><p>ok</p>`, wantRaw: `<!--<script>-->`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			start := len(`<script>`)
			assertScriptQuery(t, testCase.document, testCase.wantRaw, start, start+len(testCase.wantRaw))
			paragraphs, err := xpath.Query(`//p/text()`, testCase.document)
			if err != nil || len(paragraphs) != 1 || paragraphs[0].TextContent != "ok" {
				t.Fatalf("Expected p after correct script close, results=%#v err=%v", paragraphs, err)
			}
		})
	}
}

func TestQueryScriptEscapedAppropriateEndTagDelimiters(t *testing.T) {
	for _, document := range []string{
		`<script><!--x</SCRIPT   ><p>ok</p>`,
		`<script><!--x</script/><p>ok</p>`,
	} {
		t.Run(document, func(t *testing.T) {
			assertScriptQuery(t, document, `<!--x`, len(`<script>`), len(`<script><!--x`))
			paragraphs, err := xpath.Query(`//p/text()`, document)
			if err != nil || len(paragraphs) != 1 || paragraphs[0].TextContent != "ok" {
				t.Fatalf("Expected p after delimited script close, results=%#v err=%v", paragraphs, err)
			}
		})
	}
}

func TestQueryScriptEscapedIncompleteEndCandidatesAtEOF(t *testing.T) {
	t.Run("escaped appropriate candidate", func(t *testing.T) {
		const document = `<script><!--x</script `
		assertScriptQuery(t, document, `<!--x`, len(`<script>`), len(document))
	})
	t.Run("double escaped candidate", func(t *testing.T) {
		const document = `<script><!--<script>x</script `
		assertScriptQuery(t, document, `<!--<script>x</script `, len(`<script>`), len(document))
	})
}

func TestQueryNestedScriptEscapedEOFClosesAncestor(t *testing.T) {
	const document = `<div><script><!--é`
	assertScriptQuery(t, document, `<!--é`, len(`<div><script>`), len(document))
	divs, err := xpath.Query(`//div`, document)
	if err != nil || len(divs) != 1 || divs[0].TextContent != `<!--é` || divs[0].StartLocation != 0 || divs[0].EndLocation != len(document) {
		t.Fatalf("Expected outer div through EOF, results=%#v err=%v", divs, err)
	}
	scripts, err := xpath.Query(`//script`, document)
	if err != nil || len(scripts) != 1 || scripts[0].TextContent != `<!--é` || scripts[0].StartLocation != len(`<div>`) || scripts[0].EndLocation != len(document) {
		t.Fatalf("Expected nested script through EOF, results=%#v err=%v", scripts, err)
	}
}

func TestQueryScriptDoubleEscapedInnerEndTagAfterDashes(t *testing.T) {
	for dashCount := 1; dashCount <= 3; dashCount++ {
		t.Run(string(rune('0'+dashCount))+" dashes", func(t *testing.T) {
			dashes := strings.Repeat("-", dashCount)
			raw := `<!--<script>` + dashes + `</script>-->`
			document := `<script>` + raw + `</script><p>ok</p>`
			start := len(`<script>`)
			assertScriptQuery(t, document, raw, start, start+len(raw))
			paragraphs, err := xpath.Query(`//p/text()`, document)
			if err != nil || len(paragraphs) != 1 || paragraphs[0].TextContent != "ok" {
				t.Fatalf("Expected p after outer close, results=%#v err=%v", paragraphs, err)
			}
		})
	}
}

func TestQueryIgnoresOrphanScriptEndTags(t *testing.T) {
	testCases := []struct {
		name, document string
		wantStart      int
	}{
		{
			name: "after correctly closed script", document: `<script>x</script></script><p>ok</p>`,
			wantStart: len(`<script>x</script></script><p>`),
		},
		{name: "at document root", document: `</script><p>ok</p>`, wantStart: len(`</script><p>`)},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			paragraphs, err := xpath.Query(`//p/text()`, testCase.document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(paragraphs) != 1 || paragraphs[0].TextContent != "ok" || paragraphs[0].StartLocation != testCase.wantStart || paragraphs[0].EndLocation != testCase.wantStart+2 {
				t.Fatalf("Expected p text ok at %d:%d, got %#v", testCase.wantStart, testCase.wantStart+2, paragraphs)
			}
		})
	}
}

func TestQueryIgnoresNestedOrphanScriptEndTags(t *testing.T) {
	testCases := []struct {
		name, document     string
		wantStart, wantEnd int
	}{
		{name: "inside div", document: `<div></script><p>ok</p></div>`, wantStart: 17, wantEnd: 19},
		{name: "after closed nested script", document: `<div><script>x</script></script><p>ok</p></div>`, wantStart: 35, wantEnd: 37},
		{name: "inside explicit body", document: `<html><body></script><p>ok</p></body></html>`, wantStart: 24, wantEnd: 26},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			paragraphs, err := xpath.Query(`//p/text()`, testCase.document)
			if err != nil {
				t.Fatalf("Query returned an error: %v", err)
			}
			if len(paragraphs) != 1 || paragraphs[0].TextContent != "ok" || paragraphs[0].StartLocation != testCase.wantStart || paragraphs[0].EndLocation != testCase.wantEnd {
				t.Fatalf("Expected p text ok at %d:%d, got %#v", testCase.wantStart, testCase.wantEnd, paragraphs)
			}
		})
	}
}
