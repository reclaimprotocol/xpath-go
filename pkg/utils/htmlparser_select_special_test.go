package utils

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestParseSelectItemEndScopesRespectListBoundary(t *testing.T) {
	const listItem = `<select><li><ul>x</li>y</select>`
	document, err := NewHTMLParser().Parse(listItem)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	li := selectElements(selectNode, "li")[0]
	ul := selectElements(li, "ul")[0]
	assertSelectNode(t, li, "li", "xy", 8, 23)
	assertSelectNode(t, ul, "ul", "xy", 12, 23)
	if len(ul.Children) != 1 || ul.Children[0].Value != "xy" || ul.Children[0].StartPos != 16 || ul.Children[0].EndPos != 23 {
		t.Fatalf("Expected li end ignored behind inner ul list-item-scope boundary, got %#v", ul.Children)
	}

	for _, itemName := range []string{"dd", "dt"} {
		content := `<select><` + itemName + `><ul>x</` + itemName + `>y</select>`
		document, err = NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Parse failed for %s ordinary-scope contrast: %v", itemName, err)
		}
		selectNode = selectElements(document, "select")[0]
		item := selectElements(selectNode, itemName)[0]
		ul = selectElements(item, "ul")[0]
		assertSelectNode(t, item, itemName, "x", 8, 22)
		assertSelectNode(t, ul, "ul", "x", 12, 17)
		if len(selectNode.Children) != 2 || selectNode.Children[0] != item || selectNode.Children[1].Value != "y" || selectNode.Children[1].StartPos != 22 || selectNode.Children[1].EndPos != 23 {
			t.Fatalf("Expected %s end to cross ul in ordinary scope and put y under select, got %#v", itemName, selectNode.Children)
		}
	}
}

func TestParseSelectSpecialStartsMergeHTMLAndIgnoreBodyHead(t *testing.T) {
	const htmlStart = `<html id=a><body><select><option>x<html lang=z>y</select></body></html>`
	document, err := NewHTMLParser().Parse(htmlStart)
	if err != nil {
		t.Fatal(err)
	}
	htmlNodes := selectElements(document, "html")
	if len(htmlNodes) != 1 || htmlNodes[0].Attributes["id"] != "a" || htmlNodes[0].Attributes["lang"] != "z" {
		t.Fatalf("Expected duplicate html start to merge lang=z into the one html node, got %#v", htmlNodes)
	}
	assertSelectNode(t, selectElements(document, "option")[0], "option", "xy", 25, 48)
	if text := selectElements(document, "option")[0].Children[0]; text.Value != "xy" || text.StartPos != 33 || text.EndPos != 48 {
		t.Fatalf("Expected ignored duplicate html token within coalesced option text, got %#v", text)
	}

	const bodyStart = `<html><body id=a><select><option>x<body class=z>y</select></body></html>`
	document, err = NewHTMLParser().Parse(bodyStart)
	if err != nil {
		t.Fatal(err)
	}
	bodyNodes := selectElements(document, "body")
	if len(bodyNodes) != 1 || bodyNodes[0].Attributes["id"] != "a" {
		t.Fatalf("Expected one original body, got %#v", bodyNodes)
	}
	if _, exists := bodyNodes[0].Attributes["class"]; exists {
		t.Fatalf("Chrome ignores duplicate body attributes behind select scope, got %#v", bodyNodes[0].Attributes)
	}
	assertSelectNode(t, selectElements(document, "option")[0], "option", "xy", 25, 49)

	const headStart = `<html><body><select><option>x<head>y</select></body></html>`
	document, err = NewHTMLParser().Parse(headStart)
	if err != nil {
		t.Fatal(err)
	}
	headNodes := selectElements(document, "head")
	if len(headNodes) != 1 || headNodes[0].StartPos != 0 || headNodes[0].EndPos != 0 || len(headNodes[0].Children) != 0 {
		t.Fatalf("Explicit html/body must retain one locationless synthetic head while ignoring the select-scoped head start: %#v", headNodes)
	}
	assertSelectNode(t, selectElements(document, "option")[0], "option", "xy", 20, 36)
}

func TestParseRootSelectSpecialStartsTerminateWithoutDocumentTargets(t *testing.T) {
	if helperCase := os.Getenv("XPATH_SELECT_SPECIAL_HELPER"); helperCase != "" {
		content := `<select>a<html lang=z>b</select>`
		if helperCase == "body" {
			content = `<select>a<body class=z>b</select>`
		}
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		selectNode := selectElements(document, "select")[0]
		assertSelectNode(t, selectNode, "select", "ab", 0, len(content))
		if len(selectNode.Children) != 1 || selectNode.Children[0].Value != "ab" || selectNode.Children[0].StartPos != 8 || selectNode.Children[0].EndPos != len(content)-9 {
			t.Fatalf("Expected special start ignored inside coalesced root select text, got %#v", selectNode.Children)
		}
		if helperCase == "html" {
			htmlNodes := selectElements(document, "html")
			if len(htmlNodes) > 1 {
				t.Fatalf("Expected at most one html target, got %#v", htmlNodes)
			}
		}
		return
	}

	for _, helperCase := range []string{"html", "body"} {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestParseRootSelectSpecialStartsTerminateWithoutDocumentTargets$")
		command.Env = append(os.Environ(), "XPATH_SELECT_SPECIAL_HELPER="+helperCase)
		output, err := command.CombinedOutput()
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("Root select %s start hung without a document target", helperCase)
		}
		if err != nil {
			t.Fatalf("Root select %s helper failed: %v\n%s", helperCase, err, output)
		}
	}

	parser := NewHTMLParser()
	for _, content := range []string{`<select>a<html lang=z>b</select>`, `<select>a<body class=z>b</select>`} {
		document, err := parser.Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		if selectElements(document, "select")[0].TextContent != "ab" {
			t.Fatalf("Expected parser reuse text ab for %q", content)
		}
	}
}

func TestParseMismatchedHeadingGroupEndInsideSelect(t *testing.T) {
	const plain = `<select><h1>x</h2>y</select>`
	document, err := NewHTMLParser().Parse(plain)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	heading := selectElements(selectNode, "h1")[0]
	assertSelectNode(t, heading, "h1", "x", 8, 13)
	if len(selectNode.Children) != 2 || selectNode.Children[1].Value != "y" || selectNode.Children[1].StartPos != 18 || selectNode.Children[1].EndPos != 19 {
		t.Fatalf("Expected y after mismatched heading-group end, got %#v", selectNode.Children)
	}

	const spanCase = `<select><h1><span>x</h2>y</select>`
	document, err = NewHTMLParser().Parse(spanCase)
	if err != nil {
		t.Fatal(err)
	}
	selectNode = selectElements(document, "select")[0]
	heading = selectElements(selectNode, "h1")[0]
	span := selectElements(heading, "span")[0]
	assertSelectNode(t, heading, "h1", "x", 8, 19)
	assertSelectNode(t, span, "span", "x", 12, 19)
	if selectNode.Children[1].Value != "y" || selectNode.Children[1].StartPos != 24 || selectNode.Children[1].EndPos != 25 {
		t.Fatalf("Expected y after heading/span unwind, got %#v", selectNode.Children)
	}

	const optionCase = `<select><option><h1><span>x</h2>y</option></select>`
	document, err = NewHTMLParser().Parse(optionCase)
	if err != nil {
		t.Fatal(err)
	}
	option := selectElements(document, "option")[0]
	heading = selectElements(option, "h1")[0]
	span = selectElements(heading, "span")[0]
	assertSelectNode(t, option, "option", "xy", 8, 42)
	assertSelectNode(t, heading, "h1", "x", 16, 27)
	assertSelectNode(t, span, "span", "x", 20, 27)
	if option.Children[1].Value != "y" || option.Children[1].StartPos != 32 || option.Children[1].EndPos != 33 {
		t.Fatalf("Expected y to remain in option after heading close, got %#v", option.Children)
	}
}

func TestParseLegacyImageAndBrEndBecomeVoidElementsInsideSelect(t *testing.T) {
	const selectImage = `<select>a<image src=x>b</select>`
	document, err := NewHTMLParser().Parse(selectImage)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	images := selectElements(selectNode, "img")
	if len(images) != 1 || images[0].StartPos != 9 || images[0].EndPos != 22 || images[0].Attributes["src"] != "x" || len(images[0].Children) != 0 {
		t.Fatalf("Expected source image token renamed to void img at 9:22, got %#v", images)
	}
	if len(selectNode.Children) != 3 || selectNode.Children[0].Value != "a" || selectNode.Children[0].StartPos != 8 || selectNode.Children[0].EndPos != 9 || selectNode.Children[2].Value != "b" || selectNode.Children[2].StartPos != 22 || selectNode.Children[2].EndPos != 23 {
		t.Fatalf("Expected text to continue around void img, got %#v", selectNode.Children)
	}

	const optionImage = `<select><option>a<image src=x>b</option></select>`
	document, err = NewHTMLParser().Parse(optionImage)
	if err != nil {
		t.Fatal(err)
	}
	option := selectElements(document, "option")[0]
	images = selectElements(option, "img")
	assertSelectNode(t, option, "option", "ab", 8, 40)
	if len(images) != 1 || images[0].StartPos != 17 || images[0].EndPos != 30 || images[0].Parent != option || option.Children[2].Value != "b" || option.Children[2].StartPos != 30 || option.Children[2].EndPos != 31 {
		t.Fatalf("Expected void img inside option with following b, got option=%#v image=%#v", option.Children, images)
	}

	const brEnd = `<select><option>a</br>b</option></select>`
	document, err = NewHTMLParser().Parse(brEnd)
	if err != nil {
		t.Fatal(err)
	}
	option = selectElements(document, "option")[0]
	brNodes := selectElements(option, "br")
	if len(brNodes) != 1 {
		t.Fatalf("Expected br end reprocessed as one br start, got %#v", brNodes)
	}
	br := brNodes[0]
	assertSelectNode(t, option, "option", "ab", 8, 32)
	if br.StartPos != 17 || br.EndPos != 22 || br.Parent != option || option.Children[2].Value != "b" || option.Children[2].StartPos != 22 || option.Children[2].EndPos != 23 {
		t.Fatalf("Expected br end reprocessed as void br start, got br=%#v option=%#v", br, option.Children)
	}
}

func TestParseIgnoredSelectEndTokenRangeDoesNotLeakAcrossElements(t *testing.T) {
	const divCase = `<select><div></foo></div>x</select>`
	document, err := NewHTMLParser().Parse(divCase)
	if err != nil {
		t.Fatal(err)
	}
	selectNode := selectElements(document, "select")[0]
	div := selectElements(selectNode, "div")[0]
	assertSelectNode(t, div, "div", "", 8, 25)
	if len(selectNode.Children) != 2 || selectNode.Children[1].Value != "x" || selectNode.Children[1].StartPos != 25 || selectNode.Children[1].EndPos != 26 {
		t.Fatalf("Expected ignored token range confined before outer x, got %#v", selectNode.Children)
	}

	const spanCase = `<select><option><span></foo></span>x</option></select>`
	document, err = NewHTMLParser().Parse(spanCase)
	if err != nil {
		t.Fatal(err)
	}
	option := selectElements(document, "option")[0]
	span := selectElements(option, "span")[0]
	assertSelectNode(t, span, "span", "", 16, 35)
	if len(option.Children) != 2 || option.Children[1].Value != "x" || option.Children[1].StartPos != 35 || option.Children[1].EndPos != 36 {
		t.Fatalf("Expected ignored token range confined before option x, got %#v", option.Children)
	}

	const innerElement = `<select><div></foo><span>a</span></div>x</select>`
	document, err = NewHTMLParser().Parse(innerElement)
	if err != nil {
		t.Fatal(err)
	}
	selectNode = selectElements(document, "select")[0]
	div = selectElements(selectNode, "div")[0]
	span = selectElements(div, "span")[0]
	assertSelectNode(t, div, "div", "a", 8, 39)
	assertSelectNode(t, span, "span", "a", 19, 33)
	if span.Children[0].StartPos != 25 || span.Children[0].EndPos != 26 || selectNode.Children[1].Value != "x" || selectNode.Children[1].StartPos != 39 || selectNode.Children[1].EndPos != 40 {
		t.Fatalf("Expected ignored token not to leak into later inner/outer text, got span=%#v select=%#v", span.Children, selectNode.Children)
	}
}

func TestParseIgnoredBodyHTMLEndsInsideSelectDoNotLeakEOFRecovery(t *testing.T) {
	const explicit = `<html><body><select>a</body>b</html>c</select></body></html><foo>x`
	document, err := NewHTMLParser().Parse(explicit)
	if err != nil {
		t.Fatalf("Qualifying explicit document must EOF-close later foo: %v", err)
	}
	foo := selectElements(document, "foo")
	if len(foo) != 1 || foo[0].TextContent != "x" || foo[0].EndPos != len(explicit) {
		t.Fatalf("Explicit document foo EOF recovery mismatch: %#v", foo)
	}

	const legacy = `<select>a</body>b</html>c</select><foo>x`
	document, err = NewHTMLParser().Parse(legacy)
	foo = selectElements(document, "foo")
	if err != nil || len(foo) != 1 || foo[0].EndPos != len(legacy) {
		t.Fatalf("Implicit-document select/foo EOF mismatch for %q: %#v err=%v", legacy, foo, err)
	}

	for _, content := range []string{
		`<html><body><select>a</body>b`,
		`<select>a</body>b`,
	} {
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatalf("Expected unresolved select EOF recovery for %q: %v", content, err)
		}
		selectNode := selectElements(document, "select")[0]
		assertSelectNode(t, selectNode, "select", "ab", strings.Index(content, `<select>`), len(content))
	}

	parser := NewHTMLParser()
	if _, err := parser.Parse(explicit); err != nil {
		t.Fatalf("Expected reused parser explicit EOF recovery for %q: %v", explicit, err)
	}
	if document, err := parser.Parse(legacy); err != nil || len(selectElements(document, "foo")) != 1 || selectElements(document, "foo")[0].EndPos != len(legacy) {
		t.Fatalf("Expected explicit-to-implicit document reuse for %q: doc=%#v err=%v", legacy, document, err)
	}
	for _, content := range []string{`<select>a</body>b`, `<select>x`} {
		if _, err := parser.Parse(content); err != nil {
			t.Fatalf("Expected reused parser unresolved select recovery for %q: %v", content, err)
		}
	}
}
