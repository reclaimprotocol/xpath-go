package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func requireDocumentSkeletonQuery(t *testing.T, content, expression, name, text string, start, end int) xpath.Result {
	t.Helper()
	results, err := xpath.Query(expression, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("XPath %q returned %d results, want one: %#v", expression, len(results), results)
	}
	result := results[0]
	if result.NodeType != 1 || result.NodeName != name || result.NamespaceURI != "http://www.w3.org/1999/xhtml" || result.TextContent != text || result.StartLocation != start || result.EndLocation != end {
		t.Fatalf("XPath %q document-skeleton result mismatch: %#v", expression, result)
	}
	return result
}

func TestQueryExplicitDocumentSyntheticSkeletonAbsolutePaths(t *testing.T) {
	const content = `<!doctype html><html id=h><title id=t>x</title><div id=d>y</div></html>`
	results, err := xpath.Query(`/html/head | /html/head/title | /html/body | /html/body/div`, content)
	if err != nil {
		t.Fatal(err)
	}
	wants := []struct {
		name, text string
		start, end int
	}{{"head", "x", 0, 0}, {"title", "x", 26, 47}, {"body", "y", 0, 0}, {"div", "y", 47, 64}}
	if len(results) != len(wants) {
		t.Fatalf("absolute skeleton query returned %d results, want %d: %#v", len(results), len(wants), results)
	}
	for i, want := range wants {
		got := results[i]
		if got.NodeName != want.name || got.TextContent != want.text || got.StartLocation != want.start || got.EndLocation != want.end || got.NamespaceURI != "http://www.w3.org/1999/xhtml" {
			t.Fatalf("absolute skeleton result %d mismatch: %#v", i, got)
		}
	}
	contents, err := xpath.QueryWithOptions(`/html/head | /html/body`, content, xpath.Options{IncludeLocation: true, OutputFormat: "nodes", ContentsOnly: true})
	if err != nil || len(contents) != 2 || contents[0].ContentStart != 0 || contents[0].ContentEnd != 0 || contents[1].ContentStart != 0 || contents[1].ContentEnd != 0 {
		t.Fatalf("locationless synthetic wrapper contents-only mismatch: %#v err=%v", contents, err)
	}
}

func TestQueryExplicitDocumentSyntheticParentsAndBodyCommit(t *testing.T) {
	const omitted = `<!doctype html><html id=h><title id=t>x</title><div id=d>y</div></html>`
	requireDocumentSkeletonQuery(t, omitted, `//*[@id='t']/parent::head`, "head", "x", 0, 0)
	requireDocumentSkeletonQuery(t, omitted, `//*[@id='d']/ancestor::body`, "body", "y", 0, 0)
	requireDocumentSkeletonQuery(t, omitted, `//*[@id='d']/ancestor::html`, "html", "xy", 15, 71)

	const committed = `<!doctype html><html id=h>x<head id=e><title id=t>y</title></head><meta id=m><p id=p>z</p></html>`
	requireDocumentSkeletonQuery(t, committed, `//*[@id='t']/parent::body`, "body", "xyz", 0, 0)
	results, err := xpath.Query(`/html/head/*`, committed)
	if err != nil || len(results) != 0 {
		t.Fatalf("ignored late head start populated synthetic head: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`/html/body/node()`, committed)
	if err != nil || len(results) != 4 || results[0].NodeType != 3 || results[0].TextContent != "x" || results[1].NodeName != "title" || results[2].NodeName != "meta" || results[3].NodeName != "p" {
		t.Fatalf("body-commit child order mismatch: %#v err=%v", results, err)
	}
}

func TestQueryExplicitDocumentRetainedHeadAndPostBodyReentry(t *testing.T) {
	const retained = `<html><!--bh--><head id=h><title>T</title></head><!--ah--><meta id=late><body id=b>x</body><!--ab--></html><!--aa-->`
	requireDocumentSkeletonQuery(t, retained, `//*[@id='late']/parent::head`, "head", "T", 15, 58)
	for _, testCase := range []struct {
		expression, value string
	}{{`/html/comment()[.='bh']`, "bh"}, {`/html/comment()[.='ah']`, "ah"}, {`/html/comment()[.='ab']`, "ab"}, {`/comment()[.='aa']`, "aa"}} {
		results, err := xpath.Query(testCase.expression, retained)
		if err != nil || len(results) != 1 || results[0].NodeType != 8 || results[0].TextContent != testCase.value {
			t.Fatalf("phase comment XPath %q mismatch: %#v err=%v", testCase.expression, results, err)
		}
	}

	const afterBody = `<html><body id=b>x</body><meta id=m><title id=t>L</title><p>y</p></html>`
	requireDocumentSkeletonQuery(t, afterBody, `//*[@id='m']/parent::body`, "body", "xLy", 6, 25)
	requireDocumentSkeletonQuery(t, afterBody, `//*[@id='t']/parent::body`, "body", "xLy", 6, 25)
	results, err := xpath.Query(`/html/body/node()`, afterBody)
	if err != nil || len(results) != 4 || results[0].TextContent != "x" || results[1].NodeName != "meta" || results[2].NodeName != "title" || results[3].NodeName != "p" {
		t.Fatalf("post-body XPath reentry mismatch: %#v err=%v", results, err)
	}
}

func TestQueryExplicitDocumentHeadExitAndDuplicateStarts(t *testing.T) {
	const headExit = `<!doctype html><html id=h><head id=e><meta id=m><div id=d>x</div><title id=t>y</title></head><body id=b>z</body></html>`
	requireDocumentSkeletonQuery(t, headExit, `//*[@id='m']/parent::head`, "head", "", 26, 48)
	requireDocumentSkeletonQuery(t, headExit, `//*[@id='d']/parent::body`, "body", "xyz", 0, 0)
	requireDocumentSkeletonQuery(t, headExit, `//*[@id='t']/parent::body`, "body", "xyz", 0, 0)

	const duplicate = `<html id=a><html id=b class=x><head id=h><head class=q></head><body id=b><body class=y id=z>x</body></html>`
	results, err := xpath.Query(`/html[@id='a' and @class='x']/body[@id='b' and @class='y']`, duplicate)
	if err != nil || len(results) != 1 || results[0].TextContent != "x" || results[0].StartLocation != 62 || results[0].EndLocation != 100 {
		t.Fatalf("Chrome first-wins duplicate wrapper XPath mismatch: %#v err=%v", results, err)
	}
	results, err = xpath.Query(`/html/head[@id='h' and not(@class)]`, duplicate)
	if err != nil || len(results) != 1 {
		t.Fatalf("duplicate head start must be ignored: %#v err=%v", results, err)
	}
}

func TestQueryExplicitDocumentPIsFollowInsertionPhases(t *testing.T) {
	const content = `<?pre?><!doctype html><?beforehtml?><html id=h><?beforehead?><head id=e><?head?></head><?afterhead?><body id=b><?body?></body><?afterbody?></html><?afterhtml?>`
	tests := []struct {
		expression, target string
	}{
		{`/processing-instruction('pre')`, "pre"},
		{`/processing-instruction('beforehtml')`, "beforehtml"},
		{`/html/processing-instruction('beforehead')`, "beforehead"},
		{`/html/head/processing-instruction('head')`, "head"},
		{`/html/processing-instruction('afterhead')`, "afterhead"},
		{`/html/body/processing-instruction('body')`, "body"},
		{`/html/processing-instruction('afterbody')`, "afterbody"},
		{`/processing-instruction('afterhtml')`, "afterhtml"},
	}
	for _, testCase := range tests {
		results, err := xpath.Query(testCase.expression, content)
		if err != nil || len(results) != 1 || results[0].NodeType != 7 || results[0].NodeName != testCase.target {
			t.Fatalf("PI phase XPath %q mismatch: %#v err=%v", testCase.expression, results, err)
		}
	}
}

func TestQueryExplicitDocumentUTF8AndDecodedInsertionBoundaries(t *testing.T) {
	const unicode = "<!doctype html><html id=h>\r\n<title>é</title>\r\n<div>😀</div></html>"
	requireDocumentSkeletonQuery(t, unicode, `/html/head/title`, "title", "é", 28, 45)
	requireDocumentSkeletonQuery(t, unicode, `/html/body/div`, "div", "😀", 47, 62)

	const decoded = "<html><head></head> \t&#32;&#65;x"
	results, err := xpath.Query(`/html/text() | /html/body/text()`, decoded)
	if err != nil || len(results) != 2 || results[0].TextContent != " \t " || results[0].StartLocation != 19 || results[0].EndLocation != 30 || results[1].TextContent != "Ax" || results[1].StartLocation != 30 || results[1].EndLocation != 32 {
		t.Fatalf("decoded insertion-boundary XPath mismatch: %#v err=%v", results, err)
	}
}

func TestQueryExplicitDocumentEOFAndLegacyReuse(t *testing.T) {
	compiled, err := xpath.Compile(`/html/head | /html/body | /html/body/div`)
	if err != nil {
		t.Fatal(err)
	}
	const explicitEOF = `<html><head></head><body><div>x`
	results, err := compiled.Evaluate(explicitEOF)
	if err != nil || len(results) != 3 || results[0].NodeName != "head" || results[1].NodeName != "body" || results[2].NodeName != "div" || results[0].EndLocation != 19 || results[1].EndLocation != len(explicitEOF) || results[2].EndLocation != len(explicitEOF) {
		t.Fatalf("explicit document EOF XPath mismatch: %#v err=%v", results, err)
	}
	results, err = compiled.Evaluate(`<div>x`)
	if err != nil || len(results) != 3 || results[0].NodeName != "head" || results[1].NodeName != "body" || results[2].NodeName != "div" || results[2].EndLocation != 6 {
		t.Fatalf("compiled explicit-to-implicit document reuse mismatch: %#v err=%v", results, err)
	}
	results, err = compiled.Evaluate(`<html><body>y</body></html>`)
	if err != nil || len(results) != 2 || results[0].NodeName != "head" || results[1].TextContent != "y" {
		t.Fatalf("compiled explicit->legacy->explicit reuse mismatch: %#v err=%v", results, err)
	}
}
