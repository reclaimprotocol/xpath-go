package xpath_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	xpath "github.com/reclaimprotocol/xpath-go"
)

const commentNodeFixture = `<!--pre--><div>a<!--one-->b<!--two-->c</div><!--post-->`

// Deliberate next-batch gap: Chrome places the trailing comment in the
// implicit BODY, while xpath-go's established source-root model leaves omitted
// html/head/body wrappers absent. Batch 21 does not invent a virtual evaluator
// parent; parser and XPath trees must acquire implicit skeletons together.

func requireCommentQuery(t *testing.T, content, expression, value string, start, end int) xpath.Result {
	t.Helper()
	results, err := xpath.Query(expression, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("XPath %q returned %d results, want one: %#v", expression, len(results), results)
	}
	result := results[0]
	if result.NodeType != 8 || result.NodeName != "#comment" || result.TextContent != value || result.StartLocation != start || result.EndLocation != end || result.NamespaceURI != "" {
		t.Fatalf("XPath %q comment mismatch: %#v", expression, result)
	}
	return result
}

func TestQueryCommentNodeTestTreeOrderAndPositions(t *testing.T) {
	results, err := xpath.Query(`//comment()`, commentNodeFixture)
	if err != nil {
		t.Fatal(err)
	}
	wants := []struct {
		value      string
		start, end int
	}{{"pre", 0, 10}, {"one", 16, 26}, {"two", 27, 37}, {"post", 44, 55}}
	if len(results) != len(wants) {
		t.Fatalf("comment() returned %d results, want %d: %#v", len(results), len(wants), results)
	}
	for i, want := range wants {
		got := results[i]
		if got.NodeType != 8 || got.NodeName != "#comment" || got.TextContent != want.value || got.StartLocation != want.start || got.EndLocation != want.end {
			t.Fatalf("comment() result %d mismatch: %#v", i, got)
		}
	}
	requireCommentQuery(t, commentNodeFixture, `//div/comment()[1]`, "one", 16, 26)
	requireCommentQuery(t, commentNodeFixture, `//div/comment()[last()]`, "two", 27, 37)

	// Use explicit wrappers for browser per-parent semantics; xpath-go's public
	// source-root contract intentionally does not synthesize omitted html/body.
	const explicit = `<html><head></head><body><!--body--><div><!--one--><!--two--></div><!--post--></body></html>`
	results, err = xpath.Query(`//comment()[1]`, explicit)
	if err != nil || len(results) != 2 || results[0].TextContent != "body" || results[1].TextContent != "one" {
		t.Fatalf("comment()[1] must apply per explicit parent under abbreviated //: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//comment()[last()]`, explicit)
	if err != nil || len(results) != 2 || results[0].TextContent != "two" || results[1].TextContent != "post" {
		t.Fatalf("comment()[last()] must apply per explicit parent under abbreviated //: %#v err=%v", results, err)
	}
}

func TestQueryCommentNodeNodeAndWildcardContrast(t *testing.T) {
	results, err := xpath.Query(`//div/node()`, commentNodeFixture)
	if err != nil || len(results) != 5 {
		t.Fatalf("node() must include text and comment children: %#v err=%v", results, err)
	}
	wantTypes := []int{3, 8, 3, 8, 3}
	for i, want := range wantTypes {
		if results[i].NodeType != want {
			t.Fatalf("node() result %d type=%d, want %d: %#v", i, results[i].NodeType, want, results[i])
		}
	}
	results, err = xpath.Query(`//div/*`, commentNodeFixture)
	if err != nil || len(results) != 0 {
		t.Fatalf("wildcard must select only element children, not comments: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//div/text()`, commentNodeFixture)
	if err != nil || len(results) != 3 || results[0].TextContent != "a" || results[1].TextContent != "b" || results[2].TextContent != "c" {
		t.Fatalf("text() must exclude comments: %#v err=%v", results, err)
	}
}

func TestQueryCommentNodeParentAndSiblingAxes(t *testing.T) {
	results, err := xpath.Query(`//comment()[.='one']/parent::div`, commentNodeFixture)
	if err != nil || len(results) != 1 || results[0].NodeName != "div" || results[0].TextContent != "abc" {
		t.Fatalf("comment parent axis mismatch: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//comment()[.='one']/following-sibling::text()[1]`, commentNodeFixture)
	if err != nil || len(results) != 1 || results[0].TextContent != "b" || results[0].StartLocation != 26 || results[0].EndLocation != 27 {
		t.Fatalf("comment following-sibling text mismatch: %#v err=%v", results, err)
	}
	requireCommentQuery(t, commentNodeFixture, `//comment()[.='one']/following-sibling::comment()[1]`, "two", 27, 37)
	requireCommentQuery(t, commentNodeFixture, `//comment()[.='two']/preceding-sibling::comment()[1]`, "one", 16, 26)

	// Explicit body keeps this browser-parent assertion within xpath-go's
	// established no-synthetic-wrapper source contract.
	const explicit = `<html><head></head><body><div>a<!--one-->b</div><!--post--></body></html>`
	results, err = xpath.Query(`//comment()[last()]/parent::body`, explicit)
	if err != nil || len(results) != 1 || results[0].NodeName != "body" {
		t.Fatalf("trailing comment browser parent mismatch: %#v err=%v", results, err)
	}
}

func TestQueryCommentNodePredicatesFunctionsAndUnion(t *testing.T) {
	const content = `<div><!-- one --><!--  spaced   data  --><!--two--></div>`
	requireCommentQuery(t, content, `//comment()[.=' one ']`, " one ", 5, 17)
	requireCommentQuery(t, content, `//comment()[normalize-space(.)='spaced data']`, "  spaced   data  ", 17, 41)
	results, err := xpath.Query(`//comment()[name()='' and local-name()='' and namespace-uri()='']`, content)
	if err != nil || len(results) != 3 {
		t.Fatalf("comment name/local-name/namespace predicates mismatch: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//comment()[.='two'] | //comment()[normalize-space(.)='one'] | //comment()[normalize-space(.)='one']`, content)
	if err != nil || len(results) != 2 || results[0].TextContent != " one " || results[1].TextContent != "two" {
		t.Fatalf("comment union order/dedup mismatch: %#v err=%v", results, err)
	}
}

func TestQueryCommentNodeExistencePredicateAndWhitespaceGrammar(t *testing.T) {
	const content = `<section><div id=a><!--x--></div><div id=b>y</div></section>`
	results, err := xpath.Query(`//div[comment()]`, content)
	if err != nil || len(results) != 1 || results[0].Attributes["id"] != "a" {
		t.Fatalf("relative comment() existence predicate mismatch: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//comment ( )`, content)
	if err != nil || len(results) != 1 || results[0].NodeType != 8 || results[0].TextContent != "x" {
		t.Fatalf("whitespace-separated comment ( ) node test mismatch: %#v err=%v", results, err)
	}
	// count(comment()) depends on general typed node-set function composition,
	// a pre-existing evaluator gap intentionally deferred from this batch.
}

func TestQueryRecoveredBogusAndAbruptComments(t *testing.T) {
	const content = `<div>a<!foo>b<?xml bad?>c<?>d</$bad>e<!-->f<!--->g<!--ok--!>h</div>`
	results, err := xpath.Query(`//comment()`, content)
	if err != nil {
		t.Fatal(err)
	}
	wants := []string{"foo", "?xml bad?", "?", "$bad", "", "", "ok"}
	if len(results) != len(wants) {
		t.Fatalf("recovered comment count mismatch: %#v", results)
	}
	for i, want := range wants {
		if results[i].NodeType != 8 || results[i].TextContent != want {
			t.Fatalf("recovered comment %d mismatch: %#v", i, results[i])
		}
	}
}

func TestQueryCommentPlacementAcrossDocumentForeignAndTableContexts(t *testing.T) {
	const content = `<!--doc--><!doctype html><html><head><!--head--></head><body><!--body--><svg id=s><!--svg--></svg><math id=m><!--math--></math><table id=t><!--table--><caption id=c><!--caption--></caption><tbody id=b><!--tbody--><tr id=r><!--row--><td id=d><!--cell-->x</td></tr></tbody></table></body><!--afterbody--></html><!--afterhtml-->`
	checks := []struct {
		expression, value string
	}{
		{`/comment()[1]`, "doc"},
		{`//head/comment()`, "head"},
		{`//body/comment()[1]`, "body"},
		{`//*[@id='s']/comment()`, "svg"},
		{`//*[@id='m']/comment()`, "math"},
		{`//*[@id='t']/comment()`, "table"},
		{`//*[@id='c']/comment()`, "caption"},
		{`//*[@id='b']/comment()`, "tbody"},
		{`//*[@id='r']/comment()`, "row"},
		{`//*[@id='d']/comment()`, "cell"},
		{`//html/comment()`, "afterbody"},
		{`/comment()[last()]`, "afterhtml"},
	}
	search := 0
	for _, check := range checks {
		raw := `<!--` + check.value + `-->`
		start := strings.Index(content[search:], raw) + search
		requireCommentQuery(t, content, check.expression, check.value, start, start+len(raw))
		search = start + len(raw)
	}
}

func TestQueryCommentUnicodeNULCRLFAndContentsOnly(t *testing.T) {
	const content = "<div>é\r\n<!--a\x00b\r\n😀-->z</div>"
	result := requireCommentQuery(t, content, `//comment()`, "a�b\n😀", 9, 25)
	if result.Value != "<!--a\x00b\r\n😀-->" {
		t.Fatalf("comment full-node value lost original source: %#v", result)
	}
	results, err := xpath.QueryWithOptions(`//comment()`, content, xpath.Options{IncludeLocation: true, OutputFormat: "nodes", ContentsOnly: true})
	if err != nil || len(results) != 1 || results[0].Value != "a�b\n😀" || results[0].StartLocation != 9 || results[0].EndLocation != 25 {
		t.Fatalf("contents-only comment must retain full-token source range: %#v err=%v", results, err)
	}
}

func TestQueryCommentMalformedNodeTestsAreRejected(t *testing.T) {
	for _, expression := range []string{
		`//comment('x')`,
		`//comment("x")`,
		`//comment(1)`,
		`//comment(foo)`,
		`//comment ( 'x' )`,
		`//comment ( 1 )`,
		`//comment(`,
		`//comment())`,
		`//comment()foo`,
		`//comment() comment()`,
	} {
		if results, err := xpath.Query(expression, `<!--x-->`); err == nil {
			t.Fatalf("malformed comment node test %q unexpectedly succeeded: %#v", expression, results)
		}
	}
	results, err := xpath.Query(`//comment`, `<comment id=x></comment><!--y-->`)
	if err != nil || len(results) != 1 || results[0].NodeType != 1 || results[0].NodeName != "comment" {
		t.Fatalf("element-name comment must remain distinct from comment(): %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//comment() | //div/comment()`, `<div><!--x--></div><!--y-->`)
	if err != nil || len(results) != 2 || results[0].TextContent != "x" || results[1].TextContent != "y" {
		t.Fatalf("valid comment node-test union contrast mismatch: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//div/comment()/parent::div`, `<div><!--x--></div>`)
	if err != nil || len(results) != 1 || results[0].NodeName != "div" {
		t.Fatalf("valid chained comment node-test contrast mismatch: %#v err=%v", results, err)
	}
}

func TestCompiledCommentNodeQueryReuse(t *testing.T) {
	compiled, err := xpath.Compile(`//comment()`)
	if err != nil {
		t.Fatal(err)
	}
	for i, content := range []string{`<!--a--><div><!--b--></div>`, `<p>x</p><!--c-->`} {
		results, err := compiled.Evaluate(content)
		if err != nil {
			t.Fatalf("compiled comment query reuse %d: %v", i, err)
		}
		want := 2 - i
		if len(results) != want {
			t.Fatalf("compiled comment query reuse %d returned %#v", i, results)
		}
	}
}

func bestCommentQueryDuration(t *testing.T, content string) time.Duration {
	t.Helper()
	best := time.Duration(1<<63 - 1)
	for i := 0; i < 3; i++ {
		start := time.Now()
		results, err := xpath.Query(`//comment()`, content)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) == 0 {
			t.Fatal("comment scaling query returned no results")
		}
		if elapsed := time.Since(start); elapsed < best {
			best = elapsed
		}
	}
	return best
}

func TestQueryCommentNodeScaling(t *testing.T) {
	build := func(n int) string {
		var out strings.Builder
		out.Grow(n * 28)
		for i := 0; i < n; i++ {
			fmt.Fprintf(&out, `<div><!--comment-%d--></div>`, i)
		}
		return out.String()
	}
	smallInput, largeInput := build(1000), build(4000)
	_ = bestCommentQueryDuration(t, smallInput)
	small := bestCommentQueryDuration(t, smallInput)
	large := bestCommentQueryDuration(t, largeInput)
	if small > 8*time.Millisecond && large > 12*small+100*time.Millisecond {
		t.Fatalf("comment() XPath scales superlinearly: 1k=%v 4k=%v", small, large)
	}
}
