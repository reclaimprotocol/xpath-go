package xpath_test

import (
	"strings"
	"testing"
	"time"

	xpath "github.com/reclaimprotocol/xpath-go"
)

// Chrome 151/current WHATWG is authoritative for these PI node tests.
// Bundled jsdom represents valid HTML processing instructions as comments, so
// valid PIs intentionally do not enter the shared jsdom comparator corpus.

func requirePIQuery(t *testing.T, content, expression, target, data string, start, end int) xpath.Result {
	t.Helper()
	results, err := xpath.Query(expression, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("XPath %q returned %d results, want one: %#v", expression, len(results), results)
	}
	result := results[0]
	if result.NodeType != 7 || result.NodeName != target || result.TextContent != data || result.StartLocation != start || result.EndLocation != end || result.NamespaceURI != "" {
		t.Fatalf("XPath %q PI mismatch: %#v", expression, result)
	}
	return result
}

func TestQueryProcessingInstructionNodeTestsAndTargetLiteral(t *testing.T) {
	const content = `<div id=h>a<?Foo x?>b<?bar y?>c</div>`
	fooStart := strings.Index(content, `<?Foo x?>`)
	barStart := strings.Index(content, `<?bar y?>`)
	foo := requirePIQuery(t, content, `//processing-instruction('foo')`, "foo", "x", fooStart, fooStart+len(`<?Foo x?>`))
	if foo.Value != `<?Foo x?>` {
		t.Fatalf("full-node PI value must preserve original source: %#v", foo)
	}
	requirePIQuery(t, content, `//processing-instruction("bar")`, "bar", "y", barStart, barStart+len(`<?bar y?>`))
	results, err := xpath.Query(`//processing-instruction()`, content)
	if err != nil || len(results) != 2 || results[0].NodeName != "foo" || results[1].NodeName != "bar" {
		t.Fatalf("generic PI node test mismatch: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//processing-instruction('Foo')`, content)
	if err != nil || len(results) != 0 {
		t.Fatalf("PI target literal must be case-sensitive after HTML target folding: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//processing-instruction('')`, content)
	if err != nil || len(results) != 2 {
		t.Fatalf("Chrome treats an empty PI target literal as the generic node test: %#v err=%v", results, err)
	}
	for _, expression := range []string{
		`//processing-instruction(foo)`,
		`//processing-instruction('foo','bar')`,
		`//processing-instruction('foo'`,
	} {
		if _, err = xpath.Query(expression, content); err == nil {
			t.Fatalf("malformed PI node test %q was accepted", expression)
		}
	}
}

func TestQueryProcessingInstructionNodeOrderAxesUnionAndTextExclusion(t *testing.T) {
	const content = `<div id=h>a<?foo x?>b<?bar y?>c</div>`
	results, err := xpath.Query(`//*[@id='h']/node()`, content)
	if err != nil || len(results) != 5 {
		t.Fatalf("node() must include PI children in tree order: %#v err=%v", results, err)
	}
	wantTypes := []int{3, 7, 3, 7, 3}
	wantNames := []string{"#text", "foo", "#text", "bar", "#text"}
	for i := range results {
		if results[i].NodeType != wantTypes[i] || results[i].NodeName != wantNames[i] {
			t.Fatalf("node() child %d mismatch: %#v", i, results[i])
		}
	}
	results, err = xpath.Query(`//*[@id='h']/text()`, content)
	if err != nil || len(results) != 3 || results[0].TextContent != "a" || results[1].TextContent != "b" || results[2].TextContent != "c" {
		t.Fatalf("text() must exclude processing instructions: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//processing-instruction('foo')/following-sibling::text()[1]`, content)
	if err != nil || len(results) != 1 || results[0].TextContent != "b" {
		t.Fatalf("PI following-sibling axis mismatch: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//processing-instruction('bar')/preceding-sibling::processing-instruction()[1]`, content)
	if err != nil || len(results) != 1 || results[0].NodeName != "foo" {
		t.Fatalf("PI preceding-sibling axis mismatch: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//processing-instruction('bar') | //processing-instruction('foo') | //processing-instruction('foo')`, content)
	if err != nil || len(results) != 2 || results[0].NodeName != "foo" || results[1].NodeName != "bar" {
		t.Fatalf("PI union order/dedup mismatch: %#v err=%v", results, err)
	}
}

func TestQueryProcessingInstructionNameLocalNameAndNamespaceFunctions(t *testing.T) {
	const content = `<div><?foo x?><?bar y?></div>`
	for _, expression := range []string{
		`//processing-instruction()[name()='foo']`,
		`//processing-instruction()[local-name()='foo']`,
		`//processing-instruction()[namespace-uri()='']`,
		`//processing-instruction()[name()='foo' and local-name()='foo' and namespace-uri()='']`,
	} {
		results, err := xpath.Query(expression, content)
		if err != nil {
			t.Fatalf("XPath %q: %v", expression, err)
		}
		want := 1
		if expression == `//processing-instruction()[namespace-uri()='']` {
			want = 2
		}
		if len(results) != want || results[0].NodeName != "foo" {
			t.Fatalf("PI function predicate %q mismatch: %#v", expression, results)
		}
	}
	results, err := xpath.Query(`//text()[local-name()='']`, content)
	if err != nil || len(results) != 0 {
		t.Fatalf("empty document text contrast mismatch: %#v err=%v", results, err)
	}
}

func TestQueryProcessingInstructionDocumentAndContextPlacement(t *testing.T) {
	const content = `<?pre?><!doctype html><?post?><html><head><?head?></head><body><?body?><svg id=s><?svg?></svg><math id=m><mi><?math?></mi></math><table id=t><?table?><caption id=c><?caption?></caption><tbody><tr><td id=d><?cell?>x</td></tr></tbody></table><select id=q><?select?><option>x</option></select></body><?afterbody?></html><?afterhtml?>`
	checks := []struct{ expression, target string }{
		{`/processing-instruction('pre')`, "pre"},
		{`/processing-instruction('post')`, "post"},
		{`//head/processing-instruction('head')`, "head"},
		{`//body/processing-instruction('body')`, "body"},
		{`//*[@id='s']/processing-instruction('svg')`, "svg"},
		{`//mi/processing-instruction('math')`, "math"},
		{`//*[@id='t']/processing-instruction('table')`, "table"},
		{`//*[@id='c']/processing-instruction('caption')`, "caption"},
		{`//*[@id='d']/processing-instruction('cell')`, "cell"},
		{`//*[@id='q']/processing-instruction('select')`, "select"},
		{`//html/processing-instruction('afterbody')`, "afterbody"},
		{`/processing-instruction('afterhtml')`, "afterhtml"},
	}
	for _, check := range checks {
		start := strings.Index(content, `<?`+check.target)
		end := strings.Index(content[start:], `>`) + start + 1
		requirePIQuery(t, content, check.expression, check.target, "", start, end)
	}
}

func TestQueryProcessingInstructionBogusFallbackAndEOF(t *testing.T) {
	const invalid = `<div>a<?xml bad?>b<?1bad?>c<?_bad?>d</div>`
	results, err := xpath.Query(`//processing-instruction()`, invalid)
	if err != nil || len(results) != 1 || results[0].NodeName != "_bad" {
		t.Fatalf("invalid PI target selected as a PI: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//div/node()`, invalid)
	if err != nil || len(results) != 7 || results[1].NodeType != 8 || results[1].TextContent != "?xml bad?" || results[3].TextContent != "?1bad?" || results[5].NodeType != 7 || results[5].NodeName != "_bad" {
		t.Fatalf("bogus PI fallback query mismatch: %#v err=%v", results, err)
	}
	for _, validEOF := range []string{`<div><?`, `<div>x</div><?foo data`} {
		results, err = xpath.Query(`//processing-instruction() | //comment()`, validEOF)
		if err != nil || len(results) != 0 {
			t.Fatalf("valid incomplete PI %q must be discarded: %#v err=%v", validEOF, results, err)
		}
	}
	const invalidEOF = `<div>x</div><?1foo data`
	results, err = xpath.Query(`//comment()[last()]`, invalidEOF)
	if err != nil || len(results) != 1 || results[0].NodeType != 8 || results[0].TextContent != "?1foo data" || results[0].StartLocation != 12 || results[0].EndLocation != len(invalidEOF) {
		t.Fatalf("invalid incomplete PI bogus-comment range mismatch: %#v err=%v", results, err)
	}
}

func TestQueryProcessingInstructionUnicodeLocationsAndContents(t *testing.T) {
	const content = "<div>é\r\n<?MiXeD 😀\r\nx?>z</div>"
	result := requirePIQuery(t, content, `//processing-instruction('mixed')`, "mixed", "😀\nx", 9, 26)
	if result.Value != "<?MiXeD 😀\r\nx?>" {
		t.Fatalf("PI full value lost original CRLF/source spelling: %#v", result)
	}
	const nul = "<div><?foo a\x00b?>x</div>"
	requirePIQuery(t, nul, `//processing-instruction('foo')`, "foo", "a�b", 5, 16)

	const simple = `<div><?foo x?></div>`
	contents, err := xpath.QueryWithOptions(`//processing-instruction('foo')`, simple, xpath.Options{IncludeLocation: true, OutputFormat: "nodes", ContentsOnly: true})
	if err != nil || len(contents) != 1 || contents[0].Value != "x" || contents[0].TextContent != "x" || contents[0].StartLocation != 5 || contents[0].EndLocation != 14 {
		t.Fatalf("PI contents-only result must keep its full non-container source range: %#v err=%v", contents, err)
	}
}

func bestProcessingInstructionQueryDuration(t *testing.T, content string) time.Duration {
	t.Helper()
	best := time.Duration(1<<63 - 1)
	for i := 0; i < 3; i++ {
		start := time.Now()
		results, err := xpath.Query(`//processing-instruction('target')`, content)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) == 0 {
			t.Fatal("PI scaling query returned no results")
		}
		if elapsed := time.Since(start); elapsed < best {
			best = elapsed
		}
	}
	return best
}

func TestQueryProcessingInstructionScaling(t *testing.T) {
	build := func(n int) string {
		var out strings.Builder
		out.Grow(n * 26)
		for i := 0; i < n; i++ {
			out.WriteString(`<div><?target data?></div>`)
		}
		return out.String()
	}
	smallInput, largeInput := build(1000), build(4000)
	_ = bestProcessingInstructionQueryDuration(t, smallInput)
	small := bestProcessingInstructionQueryDuration(t, smallInput)
	large := bestProcessingInstructionQueryDuration(t, largeInput)
	if small > 8*time.Millisecond && large > 12*small+100*time.Millisecond {
		t.Fatalf("processing-instruction XPath scales superlinearly: 1k=%v 4k=%v", small, large)
	}
}
