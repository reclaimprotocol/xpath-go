package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryImplicitDocumentSkeletonForEmptyPreambleAndBodyContent(t *testing.T) {
	requireDocumentSkeletonQuery(t, "", `/html`, "html", "", 0, 0)
	requireDocumentSkeletonQuery(t, "", `/html/head`, "head", "", 0, 0)
	requireDocumentSkeletonQuery(t, "", `/html/body`, "body", "", 0, 0)

	const comment = `<!--c-->`
	commentResults, err := xpath.Query(`/comment()`, comment)
	if err != nil || len(commentResults) != 1 || commentResults[0].NodeName != "#comment" || commentResults[0].TextContent != "c" || commentResults[0].StartLocation != 0 || commentResults[0].EndLocation != 8 {
		t.Fatalf("Implicit comment preamble XPath mismatch: %#v err=%v", commentResults, err)
	}
	requireDocumentSkeletonQuery(t, comment, `/html/body`, "body", "", 0, 0)

	requireDocumentSkeletonQuery(t, `<!doctype html>`, `/html`, "html", "", 0, 0)

	requireDocumentSkeletonQuery(t, "x", `/html/body`, "body", "x", 0, 0)
	textResults, err := xpath.Query(`/html/body/text()`, "x")
	if err != nil || len(textResults) != 1 || textResults[0].TextContent != "x" || textResults[0].StartLocation != 0 || textResults[0].EndLocation != 1 {
		t.Fatalf("Implicit body text XPath mismatch: %#v err=%v", textResults, err)
	}

	const div = `<div id=d>x</div>`
	requireDocumentSkeletonQuery(t, div, `/html/body/div`, "div", "x", 0, 17)
}

func TestQueryImplicitDocumentFramesetAndStrayTableStructureStarts(t *testing.T) {
	const frameset = `<frameset>x<frame><!--c--></frameset>tail`
	requireDocumentSkeletonQuery(t, frameset, `/html/frameset`, "frameset", "", 0, 37)
	requireDocumentSkeletonQuery(t, frameset, `/html/frameset/frame`, "frame", "", 11, 18)
	comments, err := xpath.Query(`/html/frameset/comment()`, frameset)
	if err != nil || len(comments) != 1 || comments[0].TextContent != "c" || comments[0].StartLocation != 18 || comments[0].EndLocation != 26 {
		t.Fatalf("Implicit frameset comment XPath mismatch: %#v err=%v", comments, err)
	}
	if results, err := xpath.Query(`/html/body | /html/frameset/text()`, frameset); err != nil || len(results) != 0 {
		t.Fatalf("Frameset must have no body or ignored text nodes: %#v err=%v", results, err)
	}

	for _, testCase := range []struct {
		content   string
		textStart int
	}{
		{content: `<td>x`, textStart: 4},
		{content: `<tr><td>x`, textStart: 8},
		{content: `<tbody><tr><td>x`, textStart: 15},
		{content: `<caption>x`, textStart: 9},
	} {
		results, err := xpath.Query(`/html/body/text()`, testCase.content)
		if err != nil || len(results) != 1 || results[0].TextContent != "x" || results[0].StartLocation != testCase.textStart || results[0].EndLocation != len(testCase.content) {
			t.Fatalf("Stray table structure XPath mismatch for %q: %#v err=%v", testCase.content, results, err)
		}
		if ignored, err := xpath.Query(`/html/body/table | /html/body/tbody | /html/body/tr | /html/body/td | /html/body/caption | /html/body/col`, testCase.content); err != nil || len(ignored) != 0 {
			t.Fatalf("Stray table structure emitted nodes for %q: %#v err=%v", testCase.content, ignored, err)
		}
	}
	requireDocumentSkeletonQuery(t, `<col><p>x`, `/html/body/p`, "p", "x", 5, 9)
	if ignored, err := xpath.Query(`/html/body/col`, `<col><p>x`); err != nil || len(ignored) != 0 {
		t.Fatalf("Stray col emitted an element: %#v err=%v", ignored, err)
	}
}

func TestQueryImplicitDocumentSkeletonWhitespaceDecodedTextAndCompiledEmpty(t *testing.T) {
	const whitespace = " \t&#32;"
	requireDocumentSkeletonQuery(t, whitespace, `/html/body`, "body", "", 0, 0)

	const decoded = " \t&#32;&#65;x"
	requireDocumentSkeletonQuery(t, decoded, `/html/body`, "body", "Ax", 0, 0)
	results, err := xpath.Query(`/html/body/text()`, decoded)
	if err != nil || len(results) != 1 || results[0].TextContent != "Ax" || results[0].StartLocation != 11 || results[0].EndLocation != 13 {
		t.Fatalf("Implicit decoded body XPath mismatch: %#v err=%v", results, err)
	}

	compiled, err := xpath.Compile(`/html/head | /html/body`)
	if err != nil {
		t.Fatal(err)
	}
	results, err = compiled.Evaluate("")
	if err != nil || len(results) != 2 || results[0].NodeName != "head" || results[1].NodeName != "body" || results[0].StartLocation != 0 || results[0].EndLocation != 0 || results[1].StartLocation != 0 || results[1].EndLocation != 0 {
		t.Fatalf("Compiled empty-document skeleton mismatch: %#v err=%v", results, err)
	}
	results, err = compiled.Evaluate(`<div>x</div>`)
	if err != nil || len(results) != 2 || results[1].TextContent != "x" {
		t.Fatalf("Compiled implicit-skeleton reuse mismatch: %#v err=%v", results, err)
	}
}

func TestQueryImplicitDocumentSkeletonRetainsSourceHeadAndBody(t *testing.T) {
	const head = `<head id=h><title id=t>x</title></head>`
	requireDocumentSkeletonQuery(t, head, `/html`, "html", "x", 0, 0)
	requireDocumentSkeletonQuery(t, head, `/html/head`, "head", "x", 0, 39)
	requireDocumentSkeletonQuery(t, head, `/html/head/title`, "title", "x", 11, 32)
	requireDocumentSkeletonQuery(t, head, `/html/body`, "body", "", 0, 0)

	const body = `<body id=b><div id=d>x</div></body>`
	requireDocumentSkeletonQuery(t, body, `/html/head`, "head", "", 0, 0)
	requireDocumentSkeletonQuery(t, body, `/html/body`, "body", "x", 0, 35)
	requireDocumentSkeletonQuery(t, body, `/html/body/div`, "div", "x", 11, 28)
}

func TestQueryImplicitDocumentSkeletonForSourceTitleAndPartialHead(t *testing.T) {
	const title = `<title>x</title>`
	requireDocumentSkeletonQuery(t, title, `/html/head`, "head", "x", 0, 0)
	requireDocumentSkeletonQuery(t, title, `/html/head/title`, "title", "x", 0, 16)
	requireDocumentSkeletonQuery(t, title, `/html/body`, "body", "", 0, 0)

	const partial = `<head></head><div>x</div>`
	requireDocumentSkeletonQuery(t, partial, `/html/head`, "head", "", 0, 13)
	requireDocumentSkeletonQuery(t, partial, `/html/body/div`, "div", "x", 13, 25)
}

func TestQueryImplicitDocumentRoutesHeadOnlyTokensBeforeBody(t *testing.T) {
	const content = `<title id=t>x</title><meta id=m><style id=s>y</style><script id=c>z</script><base id=a href=x><link id=l rel=x><div id=d>q</div>`
	results, err := xpath.Query(`/html/head/* | /html/body/*`, content)
	if err != nil || len(results) != 7 {
		t.Fatalf("Implicit head-only routing XPath mismatch: %#v err=%v", results, err)
	}
	wantNames := []string{"title", "meta", "style", "script", "base", "link", "div"}
	for index, want := range wantNames {
		if results[index].NodeName != want {
			t.Fatalf("Implicit head/body result %d = %#v, want %s", index, results[index], want)
		}
	}
	if results[0].StartLocation != 0 || results[0].EndLocation != 21 || results[5].StartLocation != 94 || results[5].EndLocation != 111 || results[6].StartLocation != 111 || results[6].EndLocation != 128 {
		t.Fatalf("Implicit head/body exact locations mismatch: %#v", results)
	}
}

func TestQueryImplicitDocumentCommentsAndPIsFollowInsertionPhases(t *testing.T) {
	const content = `<?pre?><!--pre--><title id=t>x</title><?head?><!--head--><div id=d>y</div><?body?><!--body-->`
	requirePIQuery(t, content, `/processing-instruction('pre')`, "pre", "", 0, 7)
	requirePIQuery(t, content, `/html/head/processing-instruction('head')`, "head", "", 38, 46)
	requirePIQuery(t, content, `/html/body/processing-instruction('body')`, "body", "", 74, 82)
	for _, testCase := range []struct {
		expression string
		value      string
		start      int
		end        int
	}{
		{`/comment()[.='pre']`, "pre", 7, 17},
		{`/html/head/comment()[.='head']`, "head", 46, 57},
		{`/html/body/comment()[.='body']`, "body", 82, 93},
	} {
		results, err := xpath.Query(testCase.expression, content)
		if err != nil || len(results) != 1 || results[0].TextContent != testCase.value || results[0].StartLocation != testCase.start || results[0].EndLocation != testCase.end {
			t.Fatalf("Implicit phase XPath %s mismatch: %#v err=%v", testCase.expression, results, err)
		}
	}
}

func TestQueryImplicitDocumentPreambleAndLateHeadTokenPlacement(t *testing.T) {
	const comments = `<!--pre--><div>x</div><!--post-->`
	for _, testCase := range []struct {
		expression string
		value      string
		start      int
		end        int
	}{
		{`/comment()[.='pre']`, "pre", 0, 10},
		{`/html/body/div`, "x", 10, 22},
		{`/html/body/comment()[.='post']`, "post", 22, 33},
	} {
		results, err := xpath.Query(testCase.expression, comments)
		if err != nil || len(results) != 1 || results[0].TextContent != testCase.value || results[0].StartLocation != testCase.start || results[0].EndLocation != testCase.end {
			t.Fatalf("Implicit pre/body query %s mismatch: %#v err=%v", testCase.expression, results, err)
		}
	}

	const meta = `<meta id=m><div>x</div><meta id=late>`
	requireDocumentSkeletonQuery(t, meta, `/html/head/meta[@id='m']`, "meta", "", 0, 11)
	requireDocumentSkeletonQuery(t, meta, `/html/body/meta[@id='late']`, "meta", "", 23, 37)

	const phases = `<head id=h><title>x</title></head><!--ah--><meta id=m><body id=b>y</body><!--ab--></html><!--aa-->`
	requireDocumentSkeletonQuery(t, phases, `/html/head`, "head", "x", 0, 43)
	requireDocumentSkeletonQuery(t, phases, `/html/head/meta`, "meta", "", 43, 54)
	requireDocumentSkeletonQuery(t, phases, `/html/body`, "body", "y", 54, 73)
	for _, testCase := range []struct {
		expression string
		value      string
		start      int
		end        int
	}{
		{`/html/comment()[.='ah']`, "ah", 34, 43},
		{`/html/comment()[.='ab']`, "ab", 73, 82},
		{`/comment()[.='aa']`, "aa", 89, 98},
	} {
		results, err := xpath.Query(testCase.expression, phases)
		if err != nil || len(results) != 1 || results[0].TextContent != testCase.value || results[0].StartLocation != testCase.start || results[0].EndLocation != testCase.end {
			t.Fatalf("Implicit retained-head query %s mismatch: %#v err=%v", testCase.expression, results, err)
		}
	}
}

func TestQueryImplicitDocumentAfterBodyAndHTMLPlacement(t *testing.T) {
	const content = `<body id=b>x</body><!--ab--></html><!--aa--><div id=d>y</div><!--tail-->`
	requireDocumentSkeletonQuery(t, content, `/html/body`, "body", "xy", 0, 19)
	requireDocumentSkeletonQuery(t, content, `/html/body/div`, "div", "y", 44, 61)
	for _, testCase := range []struct {
		expression string
		value      string
		start      int
		end        int
	}{
		{`/html/comment()[.='ab']`, "ab", 19, 28},
		{`/comment()[.='aa']`, "aa", 35, 44},
		{`/html/body/comment()[.='tail']`, "tail", 61, 72},
	} {
		results, err := xpath.Query(testCase.expression, content)
		if err != nil || len(results) != 1 || results[0].TextContent != testCase.value || results[0].StartLocation != testCase.start || results[0].EndLocation != testCase.end {
			t.Fatalf("Implicit after-body XPath %s mismatch: %#v err=%v", testCase.expression, results, err)
		}
	}
}

func TestQueryImplicitDocumentEOFClosesSourceHeadAndBodyDescendants(t *testing.T) {
	const standalone = `<div>x`
	requireDocumentSkeletonQuery(t, standalone, `/html/head`, "head", "", 0, 0)
	requireDocumentSkeletonQuery(t, standalone, `/html/body`, "body", "x", 0, 0)
	requireDocumentSkeletonQuery(t, standalone, `/html/body/div`, "div", "x", 0, len(standalone))

	const head = `<head><title>x</title>`
	requireDocumentSkeletonQuery(t, head, `/html/head`, "head", "x", 0, 14)
	requireDocumentSkeletonQuery(t, head, `/html/head/title`, "title", "x", 6, 22)
	requireDocumentSkeletonQuery(t, head, `/html/body`, "body", "", 0, 0)

	const body = `<body><div>x`
	requireDocumentSkeletonQuery(t, body, `/html/head`, "head", "", 0, 0)
	requireDocumentSkeletonQuery(t, body, `/html/body`, "body", "x", 0, 6)
	requireDocumentSkeletonQuery(t, body, `/html/body/div`, "div", "x", 6, len(body))
}

func TestQueryImplicitDocumentUnicodeIncompleteAndDuplicateWrappers(t *testing.T) {
	const plain = "é\r\n😀"
	results, err := xpath.Query(`/html/body/text()`, plain)
	if err != nil || len(results) != 1 || results[0].TextContent != "é\n😀" || results[0].StartLocation != 0 || results[0].EndLocation != len(plain) {
		t.Fatalf("Implicit Unicode body XPath mismatch: %#v err=%v", results, err)
	}

	const head = "<head><title>é\r\n😀</title>"
	requireDocumentSkeletonQuery(t, head, `/html/head`, "head", "é\n😀", 0, 21)
	requireDocumentSkeletonQuery(t, head, `/html/head/title`, "title", "é\n😀", 6, 29)

	const body = "<body><div>é\r\n😀"
	requireDocumentSkeletonQuery(t, body, `/html/body`, "body", "é\n😀", 0, 6)
	requireDocumentSkeletonQuery(t, body, `/html/body/div`, "div", "é\n😀", 6, len(body))

	for _, content := range []string{`<html`, `<head`, `<body`} {
		results, err = xpath.Query(`/html | /html/head | /html/body`, content)
		if err != nil || len(results) != 3 {
			t.Fatalf("Incomplete wrapper %q implicit skeleton mismatch: %#v err=%v", content, results, err)
		}
		for _, result := range results {
			if result.StartLocation != 0 || result.EndLocation != 0 {
				t.Fatalf("Incomplete wrapper %q emitted source location: %#v", content, result)
			}
		}
	}

	const duplicate = `x<html lang=z>y<body id=b>z<head id=h>`
	html := requireDocumentSkeletonQuery(t, duplicate, `/html`, "html", "xyz", 0, 0)
	implicitBody := requireDocumentSkeletonQuery(t, duplicate, `/html/body`, "body", "xyz", 0, 0)
	if html.Attributes["lang"] != "z" || implicitBody.Attributes["id"] != "b" {
		t.Fatalf("Duplicate implicit wrapper attributes mismatch: html=%#v body=%#v", html.Attributes, implicitBody.Attributes)
	}
}

func TestQueryImplicitDocumentBodyAttachmentIntegrationGuards(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		content    string
		expression string
	}{
		{"formatting", `<b>x</b>`, `/html/body/b`},
		{"list", `<ul><li>x</ul>`, `/html/body/ul`},
		{"form", `<form>x</form>`, `/html/body/form`},
		{"select", `<select><option>x</select>`, `/html/body/select`},
		{"svg", `<svg><circle /></svg>`, `/html/body/svg`},
		{"math", `<math><mi>x</mi></math>`, `/html/body/math`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(testCase.expression, testCase.content)
			if err != nil || len(results) != 1 {
				t.Fatalf("Root integration guard %s mismatch: %#v err=%v", testCase.name, results, err)
			}
		})
	}

	const table = `<table>x<tr><td>y</table>`
	results, err := xpath.Query(`/html/body/node()`, table)
	if err != nil || len(results) != 2 || results[0].NodeName != "#text" || results[0].TextContent != "x" || results[1].NodeName != "table" || results[1].TextContent != "y" {
		t.Fatalf("Root table foster/body XPath mismatch: %#v err=%v", results, err)
	}
}

func TestQueryImplicitDocumentFramesetSkeleton(t *testing.T) {
	const frameset = `<frameset><frame src=x></frameset>`
	requireDocumentSkeletonQuery(t, frameset, `/html`, "html", "", 0, 0)
	requireDocumentSkeletonQuery(t, frameset, `/html/head`, "head", "", 0, 0)
	requireDocumentSkeletonQuery(t, frameset, `/html/frameset`, "frameset", "", 0, 34)
	requireDocumentSkeletonQuery(t, frameset, `/html/frameset/frame`, "frame", "", 10, 23)
	results, err := xpath.Query(`/html/body`, frameset)
	if err != nil || len(results) != 0 {
		t.Fatalf("Frameset document emitted a body result: %#v err=%v", results, err)
	}

	const disabled = `x<frameset><frame src=x></frameset>`
	requireDocumentSkeletonQuery(t, disabled, `/html/body`, "body", "x", 0, 0)
	results, err = xpath.Query(`//frameset | //frame`, disabled)
	if err != nil || len(results) != 0 {
		t.Fatalf("Body-committed frameset tokens were not ignored: %#v err=%v", results, err)
	}
}

func TestQueryTextPredicateUsesImmediateTextChildren(t *testing.T) {
	const content = "<a\x00b>payload</a\x00b>"
	results, err := xpath.Query(`//*[text()='payload']`, content)
	if err != nil || len(results) != 1 || results[0].NodeName != "a�b" {
		t.Fatalf("Direct text predicate matched aggregate ancestor text: %#v err=%v", results, err)
	}
}

func TestQuerySourceBodyInSourceHTMLClosesAtEOF(t *testing.T) {
	const content = `<html><head></head><body><div>x`
	body := requireDocumentSkeletonQuery(t, content, `/html/body`, "body", "x", 19, len(content))
	if body.ContentStart != 25 || body.ContentEnd != len(content) {
		t.Fatalf("Source body EOF content range mismatch: %#v", body)
	}
	requireDocumentSkeletonQuery(t, content, `/html/body/div`, "div", "x", 25, len(content))
}
