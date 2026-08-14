package xpath_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func followingPrecedingIDs(results []xpath.Result) []string {
	ids := make([]string, 0, len(results))
	for _, result := range results {
		ids = append(ids, result.Attributes["id"])
	}
	return ids
}

func requireFollowingPrecedingIDs(t *testing.T, content, expression string, want ...string) []xpath.Result {
	t.Helper()
	results, err := xpath.Query(expression, content)
	if err != nil {
		t.Fatalf("XPath %q: %v", expression, err)
	}
	got := followingPrecedingIDs(results)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("XPath %q IDs = %q, want %q; results=%#v", expression, got, want, results)
	}
	return results
}

func TestQueryFollowingAndPrecedingReviewerFixture(t *testing.T) {
	const content = `<main><a id=a><span id=desc>x</span></a><b id=b></b><aside id=near></aside><c id=c></c></main>`
	requireFollowingPrecedingIDs(t, content, `//*[@id='a']/following::*`, "b", "near", "c")
	requireFollowingPrecedingIDs(t, content, `//*[@id='c']/preceding::*[1]`, "near")
}

func TestQueryFollowingAndPrecedingNodeKinds(t *testing.T) {
	const content = `<main><a id=a><span>x</span>tail</a>α<!--mid--><b id=b>β</b></main>`
	results, err := xpath.Query(`//*[@id='a']/following::node()`, content)
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{"#text", "#comment", "b", "#text"}
	wantText := []string{"α", "mid", "β", "β"}
	if len(results) != len(wantNames) {
		t.Fatalf("following::node() returned %#v", results)
	}
	for i := range results {
		if results[i].NodeName != wantNames[i] || results[i].TextContent != wantText[i] {
			t.Fatalf("following::node() result %d = %#v", i, results[i])
		}
	}
	results, err = xpath.Query(`//*[@id='b']/preceding::node()[1]`, content)
	if err != nil || len(results) != 1 || results[0].NodeType != 8 || results[0].TextContent != "mid" {
		t.Fatalf("nearest preceding node must be the comment: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//*[@id='b']/preceding::text()[1]`, content)
	if err != nil || len(results) != 1 || results[0].TextContent != "α" {
		t.Fatalf("nearest preceding text mismatch: %#v err=%v", results, err)
	}
}

func TestQueryFollowingAndPrecedingPositionAndLast(t *testing.T) {
	const content = `<main><a id=a><span></span></a><b id=b></b><i id=i1></i><i id=i2></i><c id=c></c></main>`
	requireFollowingPrecedingIDs(t, content, `//*[@id='a']/following::*[1]`, "b")
	requireFollowingPrecedingIDs(t, content, `//*[@id='a']/following::*[last()]`, "c")
	requireFollowingPrecedingIDs(t, content, `//*[@id='c']/preceding::i[1]`, "i2")
	requireFollowingPrecedingIDs(t, content, `//*[@id='c']/preceding::i[last()]`, "i1")
}

func TestQueryFollowingAndPrecedingMultiContextDedupAndUnion(t *testing.T) {
	const following = `<main><i class=ctx id=a></i><em id=x></em><i class=ctx id=b></i><em id=y></em><em id=z></em></main>`
	requireFollowingPrecedingIDs(t, following, `//*[@class='ctx']/following::em`, "x", "y", "z")
	requireFollowingPrecedingIDs(t, following, `//*[@class='ctx']/following::em[1]`, "x", "y")
	requireFollowingPrecedingIDs(t, following, `//*[@id='b']/preceding::em | //*[@id='a']/following::em | //*[@id='a']/following::em`, "x", "y", "z")

	const preceding = `<main><em id=x></em><i class=ctx id=a></i><em id=y></em><i class=ctx id=b></i></main>`
	requireFollowingPrecedingIDs(t, preceding, `//*[@class='ctx']/preceding::em`, "x", "y")
	requireFollowingPrecedingIDs(t, preceding, `//*[@class='ctx']/preceding::em[1]`, "x", "y")
}

func TestQueryFollowingAndPrecedingFromAttributeContext(t *testing.T) {
	const content = `<section><p id=p>lead<span id=s>inside</span></p><aside id=a>after</aside></section>`
	results, err := xpath.Query(`//*[@id='p']/@id/following::node()[1]`, content)
	if err != nil || len(results) != 1 || results[0].NodeType != 3 || results[0].TextContent != "lead" {
		t.Fatalf("an attribute's following axis must begin with its owner's first child: %#v err=%v", results, err)
	}
	requireFollowingPrecedingIDs(t, content, `//*[@id='p']/@id/following::*[1]`, "s")
	requireFollowingPrecedingIDs(t, content, `//*[@id='a']/@id/preceding::*[1]`, "s")
}

func TestQueryFollowingAndPrecedingSyntheticAndRecoveredTreeOrder(t *testing.T) {
	const fragment = `<a id=a></a>gap<b id=b></b>`
	results, err := xpath.Query(`/html/head/following::*[1] | //*[@id='a']/preceding::*[1] | //*[@id='a']/following::*[1]`, fragment)
	if err != nil || len(results) != 3 || results[0].NodeName != "head" || results[1].NodeName != "body" || results[2].Attributes["id"] != "b" {
		t.Fatalf("synthetic wrapper axis order mismatch: %#v err=%v", results, err)
	}

	// The fostered div occurs after the table start in source, but before the
	// table in the DOM. Axes must follow tree order, not source offsets.
	const foster = `<table id=t><div id=f>f</div><tr><td id=c>cell</table><p id=p>after</p>`
	requireFollowingPrecedingIDs(t, foster, `//*[@id='f']/following::*[1] | //*[@id='t']/preceding::*[1]`, "f", "t")

	// Adoption creates a locationless cloned b after the source-backed div.
	// A union must still expose div then clone in DOM order.
	const adoption = `<b id=o>1<div id=d>2</b>3</div><p id=p>4</p>`
	results, err = xpath.Query(`//*[@id='d'] | //*[@id='p']/preceding::*[1]`, adoption)
	if err != nil || len(results) != 2 || results[0].NodeName != "div" || results[0].Attributes["id"] != "d" || results[1].NodeName != "b" || results[1].Attributes["id"] != "o" || results[1].StartLocation != 0 || results[1].EndLocation != 0 {
		t.Fatalf("adoption tree-order union mismatch: %#v err=%v", results, err)
	}
}

func TestQueryFollowingAndPrecedingUTF8Locations(t *testing.T) {
	const content = `<main><a id=a>éU0001f600</a>λ<!--注--><b id=b>終</b></main>`
	results, err := xpath.Query(`//*[@id='a']/following::text()[1] | //*[@id='b']/preceding::comment()[1]`, content)
	if err != nil || len(results) != 2 {
		t.Fatalf("UTF-8 following/preceding result mismatch: %#v err=%v", results, err)
	}
	lambdaStart := strings.Index(content, "λ")
	commentStart := strings.Index(content, `<!--注-->`)
	if results[0].TextContent != "λ" || results[0].StartLocation != lambdaStart || results[0].EndLocation != lambdaStart+len("λ") {
		t.Fatalf("following UTF-8 text location mismatch: %#v", results[0])
	}
	if results[1].NodeType != 8 || results[1].TextContent != "注" || results[1].StartLocation != commentStart || results[1].EndLocation != commentStart+len(`<!--注-->`) {
		t.Fatalf("preceding UTF-8 comment location mismatch: %#v", results[1])
	}
}

func TestQueryFollowingAndPrecedingProcessingInstructionAndDoctype(t *testing.T) {
	// Current Chrome HTML parsing exposes these valid processing instructions;
	// bundled jsdom represents them as comments, so PI cases stay Go-only.
	const content = `<?pre?><!doctype html><html><head></head><body><a id=a>x</a><?middle data?><b id=b>y</b></body></html><?post?>`
	results, err := xpath.Query(`//*[@id='a']/following::processing-instruction()[1] | //*[@id='a']/following::processing-instruction()[last()]`, content)
	if err != nil || len(results) != 2 || results[0].NodeName != "middle" || results[1].NodeName != "post" {
		t.Fatalf("following processing-instruction axis mismatch: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`//*[@id='b']/preceding::processing-instruction()[1] | //*[@id='b']/preceding::processing-instruction()[last()]`, content)
	if err != nil || len(results) != 2 || results[0].NodeName != "pre" || results[1].NodeName != "middle" {
		t.Fatalf("preceding processing-instruction reverse-axis mismatch: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`/processing-instruction('pre')/following::node()[1]`, content)
	doctypeStart := strings.Index(content, `<!doctype html>`)
	if err != nil || len(results) != 1 || results[0].NodeType != 10 || results[0].NodeName != "html" || results[0].StartLocation != doctypeStart || results[0].EndLocation != doctypeStart+len(`<!doctype html>`) {
		t.Fatalf("following axis did not expose the doctype: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`/html/preceding::node()[1]`, content)
	if err != nil || len(results) != 1 || results[0].NodeType != 10 {
		t.Fatalf("doctype must be the nearest node preceding html: %#v err=%v", results, err)
	}
}

func TestCompiledFollowingAndPrecedingReuse(t *testing.T) {
	compiled, err := xpath.Compile(`//*[@id='mark']/following::*[1] | //*[@id='mark']/preceding::*[1]`)
	if err != nil {
		t.Fatal(err)
	}
	for i, testCase := range []struct {
		content string
		want    []string
	}{
		{`<main><i id=before></i><b id=mark></b><em id=after></em></main>`, []string{"before", "after"}},
		{`<section><p id=left></p><a id=mark></a><aside id=right></aside></section>`, []string{"left", "right"}},
	} {
		results, evalErr := compiled.Evaluate(testCase.content)
		if evalErr != nil || strings.Join(followingPrecedingIDs(results), ",") != strings.Join(testCase.want, ",") {
			t.Fatalf("compiled axis reuse %d: results=%#v err=%v", i, results, evalErr)
		}
	}
}

func bestFollowingPrecedingDuration(t *testing.T, content, expression string) time.Duration {
	t.Helper()
	best := time.Duration(1<<63 - 1)
	for i := 0; i < 3; i++ {
		start := time.Now()
		results, err := xpath.Query(expression, content)
		if err != nil || len(results) != 1 {
			t.Fatalf("axis scaling query %q: results=%#v err=%v", expression, results, err)
		}
		if elapsed := time.Since(start); elapsed < best {
			best = elapsed
		}
	}
	return best
}

func TestQueryFollowingAndPrecedingScaling(t *testing.T) {
	build := func(n int) string {
		var out strings.Builder
		out.Grow(n * 24)
		out.WriteString(`<main><i id=first></i>`)
		for i := 0; i < n; i++ {
			fmt.Fprintf(&out, `<b id=b%d></b>`, i)
		}
		out.WriteString(`<i id=last></i></main>`)
		return out.String()
	}
	expressions := []string{
		`//*[@id='first']/following::b[last()]`,
		`//*[@id='last']/preceding::b[last()]`,
	}
	for _, expression := range expressions {
		smallInput, largeInput := build(1000), build(4000)
		_ = bestFollowingPrecedingDuration(t, smallInput, expression)
		small := bestFollowingPrecedingDuration(t, smallInput, expression)
		large := bestFollowingPrecedingDuration(t, largeInput, expression)
		if small > 8*time.Millisecond && large > 12*small+100*time.Millisecond {
			t.Fatalf("axis query %q scales superlinearly: 1k=%v 4k=%v", expression, small, large)
		}
	}
}
