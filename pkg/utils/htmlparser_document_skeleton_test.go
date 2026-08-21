package utils

import (
	"strings"
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func documentSkeletonElements(node *types.Node, name string) []*types.Node {
	var result []*types.Node
	var visit func(*types.Node)
	visit = func(current *types.Node) {
		if current.Type == types.ElementNode && current.Name == name {
			result = append(result, current)
		}
		for _, child := range current.Children {
			visit(child)
		}
	}
	visit(node)
	return result
}

func requireOneDocumentSkeletonElement(t *testing.T, document *types.Node, name string) *types.Node {
	t.Helper()
	elements := documentSkeletonElements(document, name)
	if len(elements) != 1 {
		t.Fatalf("expected one <%s>, got %d: %#v", name, len(elements), elements)
	}
	return elements[0]
}

func requireSyntheticDocumentElement(t *testing.T, node *types.Node, name string, parent *types.Node) {
	t.Helper()
	if node.Type != types.ElementNode || node.Name != name || node.NamespaceURI != htmlNamespaceURI || node.Parent != parent {
		t.Fatalf("synthetic <%s> identity mismatch: %#v", name, node)
	}
	if node.StartPos != 0 || node.EndPos != 0 || node.ContentStart != 0 || node.ContentEnd != 0 || node.StartLine != 0 || node.StartColumn != 0 || node.EndLine != 0 || node.EndColumn != 0 {
		t.Fatalf("synthetic <%s> must remain locationless, got %#v", name, node)
	}
}

func requireDirectSkeleton(t *testing.T, document *types.Node) (html, head, body *types.Node) {
	t.Helper()
	html = requireOneDocumentSkeletonElement(t, document, "html")
	headIndex, bodyIndex := -1, -1
	for i, child := range html.Children {
		if child.Type != types.ElementNode {
			continue
		}
		switch child.Name {
		case "head":
			if head != nil {
				t.Fatalf("expected one direct head child, got duplicate at index %d", i)
			}
			head, headIndex = child, i
		case "body":
			if body != nil {
				t.Fatalf("expected one direct body child, got duplicate at index %d", i)
			}
			body, bodyIndex = child, i
		}
	}
	if head == nil || body == nil || headIndex > bodyIndex {
		names := make([]string, len(html.Children))
		for i, child := range html.Children {
			names[i] = child.Name
		}
		t.Fatalf("expected one ordered direct head and body, got %v", names)
	}
	if head.Parent != html || body.Parent != html || head.NamespaceURI != htmlNamespaceURI || body.NamespaceURI != htmlNamespaceURI {
		t.Fatalf("head/body parent or namespace mismatch: head=%#v body=%#v", head, body)
	}
	return html, head, body
}

func TestParseExplicitDocumentQualificationGate(t *testing.T) {
	t.Run("preamble comment then emitted html opts in", func(t *testing.T) {
		const content = `<!--p--><html><p>x</p></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		if len(document.Children) != 2 || document.Children[0].Type != types.CommentNode || document.Children[0].Value != "p" {
			t.Fatalf("preamble comment/document order mismatch: %#v", document.Children)
		}
		html, head, body := requireDirectSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		if len(body.Children) != 1 || body.Children[0].Name != "p" || body.TextContent != "x" {
			t.Fatalf("qualifying html body mismatch: %#v", body)
		}
	})

	t.Run("decoded whitespace comment and ignored end remain before html", func(t *testing.T) {
		const content = `&#32;<!--p--></foo><html><p>x</p></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		if len(document.Children) != 2 || document.Children[0].Type != types.CommentNode || document.Children[0].Value != "p" || len(body.Children) != 1 || body.Children[0].Name != "p" {
			t.Fatalf("before-html whitespace/comment/end gate mismatch: %#v", document.Children)
		}
	})

	t.Run("decoded nonspace prevents later html opt in", func(t *testing.T) {
		const content = `&#32;&#65;<html><p>x</p></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		if html.Attributes != nil && html.Attributes["lang"] != "" || len(documentSkeletonElements(document, "p")) != 1 || head.Parent != html || body.Parent != html || body.TextContent != "Ax" {
			t.Fatalf("decoded-nonspace implicit-document tree mismatch: html=%#v body=%#v", html, body)
		}
	})

	t.Run("late html does not retroactively wrap source roots", func(t *testing.T) {
		const content = `<title>x</title><html lang=z></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		title := requireOneDocumentSkeletonElement(t, document, "title")
		if html.Attributes["lang"] != "z" || title.Parent != head || body.Parent != html || len(body.Children) != 0 {
			t.Fatalf("late html implicit-document merge mismatch: html=%#v head=%#v body=%#v", html, head, body)
		}
	})
}

func TestParseExplicitDocumentSynthesizesMissingHeadAndBody(t *testing.T) {
	t.Run("both omitted", func(t *testing.T) {
		const content = `<!doctype html><html id=h><title id=t>x</title><div id=d>y</div></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		title := requireOneDocumentSkeletonElement(t, document, "title")
		div := requireOneDocumentSkeletonElement(t, document, "div")
		if title.Parent != head || div.Parent != body || html.TextContent != "xy" || head.TextContent != "x" || body.TextContent != "y" {
			t.Fatalf("omitted-wrapper tree mismatch: html=%#v head=%#v body=%#v", html, head, body)
		}
		if html.StartPos != 15 || html.EndPos != 71 || title.StartPos != 26 || title.EndPos != 47 || div.StartPos != 47 || div.EndPos != 64 {
			t.Fatalf("source-backed ranges changed around synthetic wrappers: html=%#v title=%#v div=%#v", html, title, div)
		}
	})

	t.Run("head omitted", func(t *testing.T) {
		const content = `<!doctype html><html id=h><body id=b>x</body></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		if body.StartPos != 26 || body.EndPos != 45 || body.Attributes["id"] != "b" || body.TextContent != "x" {
			t.Fatalf("explicit body mismatch: %#v", body)
		}
	})

	t.Run("body omitted", func(t *testing.T) {
		const content = `<!doctype html><html id=h><head id=e><title>x</title></head>y</html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		requireSyntheticDocumentElement(t, body, "body", html)
		if head.StartPos != 26 || head.EndPos != 60 || head.Attributes["id"] != "e" || head.TextContent != "x" || body.TextContent != "y" || len(body.Children) != 1 || body.Children[0].StartPos != 60 || body.Children[0].EndPos != 61 {
			t.Fatalf("explicit head/synthetic body mismatch: head=%#v body=%#v", head, body)
		}
	})
}

func TestParseExplicitDocumentRoutesHeadOnlyElementsBeforeBody(t *testing.T) {
	const content = `<!doctype html><html id=h><title id=t>x</title><meta id=m><style id=s>y</style><script id=c type=application/json>z</script><base id=a><link id=l><div id=d>q</div></html>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	html, head, body := requireDirectSkeleton(t, document)
	requireSyntheticDocumentElement(t, head, "head", html)
	requireSyntheticDocumentElement(t, body, "body", html)
	wantHead := []string{"title", "meta", "style", "script", "base", "link"}
	if len(head.Children) != len(wantHead) {
		t.Fatalf("head children mismatch: %#v", head.Children)
	}
	for i, name := range wantHead {
		if head.Children[i].Name != name || head.Children[i].Parent != head {
			t.Fatalf("head child %d mismatch: %#v", i, head.Children[i])
		}
	}
	if len(body.Children) != 1 || body.Children[0].Name != "div" || body.Children[0].Parent != body || html.TextContent != "xyzq" || head.TextContent != "xyz" || body.TextContent != "q" {
		t.Fatalf("head/body routing mismatch: html=%#v head=%#v body=%#v", html, head, body)
	}
	if head.Children[0].StartPos != 26 || head.Children[0].EndPos != 47 || head.Children[5].StartPos != 135 || head.Children[5].EndPos != 146 || body.Children[0].StartPos != 146 || body.Children[0].EndPos != 163 {
		t.Fatalf("head/body routed source ranges mismatch: head=%#v body=%#v", head.Children, body.Children)
	}
}

func TestParseExplicitDocumentBodyCommitKeepsLaterHeadOnlyTokensInBody(t *testing.T) {
	const content = `<!doctype html><html id=h>x<head id=e><title id=t>y</title></head><meta id=m><p id=p>z</p></html>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	html, head, body := requireDirectSkeleton(t, document)
	requireSyntheticDocumentElement(t, head, "head", html)
	requireSyntheticDocumentElement(t, body, "body", html)
	if len(head.Children) != 0 || len(head.Attributes) != 0 {
		t.Fatalf("ignored late head start mutated synthetic head: %#v", head)
	}
	want := []struct {
		name  string
		value string
		start int
		end   int
	}{{"#text", "x", 26, 27}, {"title", "y", 38, 59}, {"meta", "", 66, 77}, {"p", "z", 77, 90}}
	if len(body.Children) != len(want) {
		t.Fatalf("body children mismatch: %#v", body.Children)
	}
	for i, expected := range want {
		child := body.Children[i]
		if child.Name != expected.name || child.TextContent != expected.value || child.StartPos != expected.start || child.EndPos != expected.end || child.Parent != body {
			t.Fatalf("body child %d mismatch: %#v", i, child)
		}
	}
}

func TestParseExplicitDocumentAfterHeadUsesRetainedHeadAndAfterBodyReentry(t *testing.T) {
	t.Run("late meta before body uses retained head", func(t *testing.T) {
		const content = `<html><!--bh--><head id=h><title>T</title></head><!--ah--><meta id=late><body id=b>x</body><!--ab--></html><!--aa-->`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		if len(head.Children) != 2 || head.Children[0].Name != "title" || head.Children[1].Name != "meta" || head.Children[1].Attributes["id"] != "late" || head.Children[1].Parent != head {
			t.Fatalf("after-head meta did not use retained head: %#v", head.Children)
		}
		wantParents := map[string]*types.Node{"bh": html, "ah": html, "ab": html, "aa": document}
		seen := map[string]bool{}
		var visit func(*types.Node)
		visit = func(node *types.Node) {
			if node.Type == types.CommentNode {
				if node.Parent != wantParents[node.Value] {
					t.Fatalf("phase comment %q parent mismatch: %#v", node.Value, node.Parent)
				}
				seen[node.Value] = true
			}
			for _, child := range node.Children {
				visit(child)
			}
		}
		visit(document)
		if len(seen) != 4 || body.TextContent != "x" {
			t.Fatalf("phase comment/body mismatch: seen=%#v body=%#v", seen, body)
		}
	})

	t.Run("post body tokens reenter body", func(t *testing.T) {
		const content = `<html><body id=b>x</body><meta id=m><title id=t>L</title><p>y</p></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		if body.StartPos != 6 || body.EndPos != 25 || body.ContentStart != 17 || body.ContentEnd != 18 || body.TextContent != "xLy" {
			t.Fatalf("explicit body range/text after reentry mismatch: %#v", body)
		}
		want := []string{"#text", "meta", "title", "p"}
		if len(body.Children) != len(want) {
			t.Fatalf("post-body children mismatch: %#v", body.Children)
		}
		for i, name := range want {
			if body.Children[i].Name != name || body.Children[i].Parent != body {
				t.Fatalf("post-body child %d mismatch: %#v", i, body.Children[i])
			}
		}
	})

	t.Run("post html ordinary tokens reenter body but comments keep phase parent", func(t *testing.T) {
		const content = `<html><head></head><body>x</body><!--ab--></html><!--aa--><p>y</p><!--tail-->`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, _, body := requireDirectSkeleton(t, document)
		if body.TextContent != "xy" || len(documentSkeletonElements(body, "p")) != 1 || documentSkeletonElements(body, "p")[0].TextContent != "y" {
			t.Fatalf("post-html ordinary-token body reentry mismatch: %#v", body)
		}
		parents := map[string]*types.Node{}
		var visit func(*types.Node)
		visit = func(node *types.Node) {
			if node.Type == types.CommentNode {
				parents[node.Value] = node.Parent
			}
			for _, child := range node.Children {
				visit(child)
			}
		}
		visit(document)
		if parents["ab"] != html || parents["aa"] != document || parents["tail"] != body {
			t.Fatalf("post-body/post-html comment parents mismatch: %#v", parents)
		}
	})
}

func TestParseExplicitEmptyHTMLSynthesizesBothWrappers(t *testing.T) {
	const content = `<html id=h></html>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	html, head, body := requireDirectSkeleton(t, document)
	requireSyntheticDocumentElement(t, head, "head", html)
	requireSyntheticDocumentElement(t, body, "body", html)
	if html.StartPos != 0 || html.EndPos != len(content) || html.TextContent != "" {
		t.Fatalf("empty explicit html mismatch: %#v", html)
	}
}

func TestParseExplicitDocumentSyntheticWrappersPreserveUTF8AndUTF16Locations(t *testing.T) {
	const content = "<!doctype html><html id=h>\r\n<title>é</title>\r\n<div>😀</div></html>"
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	html, head, body := requireDirectSkeleton(t, document)
	requireSyntheticDocumentElement(t, head, "head", html)
	requireSyntheticDocumentElement(t, body, "body", html)
	title := requireOneDocumentSkeletonElement(t, document, "title")
	div := requireOneDocumentSkeletonElement(t, document, "div")
	if title.StartPos != 28 || title.EndPos != 45 || title.StartLine != 2 || title.StartColumn != 1 || title.EndLine != 2 || title.EndColumn != 17 {
		t.Fatalf("multibyte title source/coordinate mismatch: %#v", title)
	}
	if len(title.Children) != 1 || title.Children[0].StartPos != 35 || title.Children[0].EndPos != 37 || title.Children[0].StartColumn != 8 || title.Children[0].EndColumn != 9 {
		t.Fatalf("multibyte title text mismatch: %#v", title.Children)
	}
	if div.StartPos != 47 || div.EndPos != 62 || div.StartLine != 3 || div.StartColumn != 1 || div.EndLine != 3 || div.EndColumn != 14 || len(div.Children) != 1 || div.Children[0].StartPos != 52 || div.Children[0].EndPos != 56 || div.Children[0].StartColumn != 6 || div.Children[0].EndColumn != 8 {
		t.Fatalf("supplementary div source/coordinate mismatch: %#v", div)
	}
}

func TestParseExplicitDocumentDecodedTextCommitsInsertionPhase(t *testing.T) {
	t.Run("before head whitespace ignored until decoded nonspace", func(t *testing.T) {
		const content = "<html> \t&#32;&#65;x"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		if len(body.Children) != 1 || body.Children[0].Value != "Ax" || body.Children[0].StartPos != 17 || body.Children[0].EndPos != 19 {
			t.Fatalf("decoded before-head commit mismatch: %#v", body.Children)
		}
	})

	t.Run("after head whitespace stays under html until decoded nonspace", func(t *testing.T) {
		const content = "<html><head></head> \t&#32;&#65;x"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, _, body := requireDirectSkeleton(t, document)
		if len(html.Children) != 3 || html.Children[1].Type != types.TextNode || html.Children[1].Value != " \t " || html.Children[1].StartPos != 19 || html.Children[1].EndPos != 30 || html.Children[2] != body {
			t.Fatalf("after-head whitespace placement mismatch: %#v", html.Children)
		}
		if len(body.Children) != 1 || body.Children[0].Value != "Ax" || body.Children[0].StartPos != 30 || body.Children[0].EndPos != 32 {
			t.Fatalf("decoded after-head commit mismatch: %#v", body.Children)
		}
	})

	t.Run("after body text reenters and coalesces", func(t *testing.T) {
		const content = "<html><head></head><body>x</body> \t&#32;&#65;y</html>"
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		_, _, body := requireDirectSkeleton(t, document)
		if body.TextContent != "x \t Ay" || len(body.Children) != 1 || body.Children[0].Value != "x \t Ay" || body.Children[0].StartPos != 25 || body.Children[0].EndPos != 46 {
			t.Fatalf("after-body decoded text reentry mismatch: %#v", body)
		}
	})
}

func TestParseExplicitDocumentHeadExitReprocessesBodyContent(t *testing.T) {
	const content = `<!doctype html><html id=h><head id=e><meta id=m><div id=d>x</div><title id=t>y</title></head><body id=b>z</body></html>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	html, head, body := requireDirectSkeleton(t, document)
	if head.StartPos != 26 || head.EndPos != 48 || head.ContentStart != 37 || head.ContentEnd != 48 || len(head.Children) != 1 || head.Children[0].Name != "meta" {
		t.Fatalf("head must end at body-content trigger: %#v", head)
	}
	requireSyntheticDocumentElement(t, body, "body", html)
	if body.Attributes["id"] != "b" {
		t.Fatalf("later body start must merge attributes into synthetic body: %#v", body.Attributes)
	}
	if len(body.Children) != 3 || body.Children[0].Name != "div" || body.Children[0].StartPos != 48 || body.Children[0].EndPos != 65 || body.Children[1].Name != "title" || body.Children[1].StartPos != 65 || body.Children[1].EndPos != 86 || body.Children[2].Value != "z" || body.Children[2].StartPos != 104 || body.Children[2].EndPos != 105 {
		t.Fatalf("body reprocessing mismatch: %#v", body.Children)
	}
}

func TestParseExplicitDocumentStrayHeadEndAndBodyStartRecovery(t *testing.T) {
	t.Run("stray head end before head", func(t *testing.T) {
		const content = `<!doctype html><html id=h></head><title id=t>x</title><body id=b>y</body></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		if len(head.Children) != 1 || head.Children[0].Name != "title" || head.Children[0].StartPos != 33 || head.Children[0].EndPos != 54 || body.StartPos != 54 || body.EndPos != 73 {
			t.Fatalf("stray head end recovery mismatch: head=%#v body=%#v", head, body)
		}
	})

	t.Run("body start closes head", func(t *testing.T) {
		const content = `<!doctype html><html id=h><head id=e><meta id=m><body id=b>x</body></html>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		_, head, body := requireDirectSkeleton(t, document)
		if head.StartPos != 26 || head.EndPos != 48 || len(head.Children) != 1 || head.Children[0].Name != "meta" || body.StartPos != 48 || body.EndPos != 67 || body.TextContent != "x" {
			t.Fatalf("body-start head transition mismatch: head=%#v body=%#v", head, body)
		}
	})
}

func TestParseExplicitDocumentDuplicateWrapperStartsUseChromeFirstWins(t *testing.T) {
	// Chrome 151 keeps the first id and adds only absent attributes from a
	// duplicate html/body start. Bundled jsdom 23 instead replaces both ids,
	// so this browser-backed case intentionally stays out of the shared corpus.
	const content = `<html id=a><html id=b class=x><head id=h><head class=q></head><body id=b><body class=y id=z>x</body></html>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	html, head, body := requireDirectSkeleton(t, document)
	if html.Attributes["id"] != "a" || html.Attributes["class"] != "x" || len(html.Attributes) != 2 {
		t.Fatalf("duplicate html attributes must keep first values and add absent values: %#v", html.Attributes)
	}
	if head.Attributes["id"] != "h" || len(head.Attributes) != 1 {
		t.Fatalf("duplicate head start must be ignored without merging class: %#v", head.Attributes)
	}
	if body.Attributes["id"] != "b" || body.Attributes["class"] != "y" || len(body.Attributes) != 2 || body.TextContent != "x" {
		t.Fatalf("duplicate body attributes/text mismatch: %#v", body)
	}
	if html.StartPos != 0 || html.EndPos != 107 || head.StartPos != 30 || head.EndPos != 62 || body.StartPos != 62 || body.EndPos != 100 {
		t.Fatalf("duplicate wrapper source ranges mismatch: html=%#v head=%#v body=%#v", html, head, body)
	}
}

func TestParseExplicitDocumentPhaseCommentsPIsAndDoctypes(t *testing.T) {
	t.Run("comments", func(t *testing.T) {
		const content = `<!--pre--><!doctype html><!--doctype--><html id=h><!--html--><head id=e><!--head--></head><!--afterhead--><body id=b><!--body--><div id=d>x</div></body><!--afterbody--></html><!--afterhtml-->`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		wantParents := map[string]*types.Node{"pre": document, "doctype": document, "html": html, "head": head, "afterhead": html, "body": body, "afterbody": html, "afterhtml": document}
		seen := map[string]bool{}
		var visit func(*types.Node)
		visit = func(node *types.Node) {
			if node.Type == types.CommentNode {
				parent, ok := wantParents[node.Value]
				if !ok || node.Parent != parent {
					t.Fatalf("comment %q parent mismatch: %#v", node.Value, node.Parent)
				}
				seen[node.Value] = true
			}
			for _, child := range node.Children {
				visit(child)
			}
		}
		visit(document)
		if len(seen) != len(wantParents) {
			t.Fatalf("comment phase coverage mismatch: %#v", seen)
		}
	})

	t.Run("processing instructions", func(t *testing.T) {
		const content = `<?pre?><!doctype html><?beforehtml?><html id=h><?beforehead?><head id=e><?head?></head><?afterhead?><body id=b><?body?></body><?afterbody?></html><?afterhtml?>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		wantParents := map[string]*types.Node{"pre": document, "beforehtml": document, "beforehead": html, "head": head, "afterhead": html, "body": body, "afterbody": html, "afterhtml": document}
		seen := map[string]bool{}
		var visit func(*types.Node)
		visit = func(node *types.Node) {
			if node.Type == types.ProcessingInstructionNode {
				parent, ok := wantParents[node.Name]
				start := strings.Index(content, "<?"+node.Name+"?>")
				if !ok || node.Parent != parent || node.StartPos != start || node.EndPos != start+len(node.Name)+4 {
					t.Fatalf("processing instruction %q phase/range mismatch: %#v", node.Name, node)
				}
				seen[node.Name] = true
			}
			for _, child := range node.Children {
				visit(child)
			}
		}
		visit(document)
		if len(seen) != len(wantParents) {
			t.Fatalf("processing-instruction phase coverage mismatch: %#v", seen)
		}
	})

	t.Run("later doctypes ignored", func(t *testing.T) {
		const content = `<!doctype html><html id=h><head id=e><!doctype x></head><body id=b><!doctype y>x</body></html><!doctype z>`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		var visit func(*types.Node)
		visit = func(node *types.Node) {
			if node.Type == types.DocumentTypeNode {
				count++
				if node.Parent != document || node.StartPos != 0 || node.EndPos != 15 {
					t.Fatalf("surviving doctype mismatch: %#v", node)
				}
			}
			for _, child := range node.Children {
				visit(child)
			}
		}
		visit(document)
		if count != 1 || requireOneDocumentSkeletonElement(t, document, "body").TextContent != "x" {
			t.Fatalf("later doctypes must be ignored, count=%d tree=%#v", count, document.Children)
		}
	})
}

func TestParseExplicitDocumentIncompleteTokensAndEOFRecovery(t *testing.T) {
	t.Run("incomplete head start", func(t *testing.T) {
		const content = `<!doctype html><html id=h><head id=e`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		requireSyntheticDocumentElement(t, body, "body", html)
		if html.EndPos != len(content) || len(documentSkeletonElements(document, "meta")) != 0 {
			t.Fatalf("incomplete head start emitted state: %#v", html)
		}
	})

	t.Run("incomplete meta start", func(t *testing.T) {
		const content = `<!doctype html><html id=h><head id=e><meta id=m`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		_, head, body := requireDirectSkeleton(t, document)
		if head.Attributes["id"] != "e" || len(documentSkeletonElements(document, "meta")) != 0 {
			t.Fatalf("incomplete meta start emitted a node or lost head: head=%#v", head)
		}
		requireSyntheticDocumentElement(t, body, "body", head.Parent)
	})

	t.Run("incomplete wrapper ends discarded", func(t *testing.T) {
		const headContent = `<!doctype html><html id=h><head id=e><title>x</title></head`
		document, err := NewHTMLParser().Parse(headContent)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		if html.EndPos != len(headContent) || head.StartPos != 26 || head.EndPos != 45 || head.TextContent != "x" {
			t.Fatalf("incomplete head end recovery mismatch: html=%#v head=%#v", html, head)
		}
		requireSyntheticDocumentElement(t, body, "body", html)

		const bodyContent = `<!doctype html><html id=h><head></head><body id=b><div>x</div></body`
		document, err = NewHTMLParser().Parse(bodyContent)
		if err != nil {
			t.Fatal(err)
		}
		html, _, body = requireDirectSkeleton(t, document)
		if html.EndPos != len(bodyContent) || body.StartPos != 39 || body.EndPos != len(bodyContent) || body.TextContent != "x" {
			t.Fatalf("incomplete body end recovery mismatch: html=%#v body=%#v", html, body)
		}
	})

	t.Run("explicit document EOF closes descendants", func(t *testing.T) {
		const content = `<!doctype html><html id=h><head id=e><title>x</title></head><body id=b><div id=d>y`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, _, body := requireDirectSkeleton(t, document)
		div := requireOneDocumentSkeletonElement(t, document, "div")
		if html.EndPos != len(content) || body.EndPos != len(content) || div.EndPos != len(content) || div.TextContent != "y" {
			t.Fatalf("explicit document EOF propagation mismatch: html=%#v body=%#v div=%#v", html, body, div)
		}
	})

	t.Run("head and body EOF recovery", func(t *testing.T) {
		const headContent = `<html><head><title>T</title>`
		document, err := NewHTMLParser().Parse(headContent)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireDirectSkeleton(t, document)
		requireSyntheticDocumentElement(t, body, "body", html)
		title := requireOneDocumentSkeletonElement(t, document, "title")
		if html.EndPos != len(headContent) || head.StartPos != 6 || head.EndPos != 20 || title.StartPos != 12 || title.EndPos != len(headContent) || title.TextContent != "T" {
			t.Fatalf("head EOF range mismatch: html=%#v head=%#v title=%#v", html, head, title)
		}

		const bodyContent = `<html><body>x`
		document, err = NewHTMLParser().Parse(bodyContent)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body = requireDirectSkeleton(t, document)
		requireSyntheticDocumentElement(t, head, "head", html)
		if html.EndPos != len(bodyContent) || body.StartPos != 6 || body.EndPos != len(bodyContent) || body.TextContent != "x" {
			t.Fatalf("body EOF range mismatch: html=%#v body=%#v", html, body)
		}
	})

	t.Run("discarded incomplete body trigger creates empty synthetic body", func(t *testing.T) {
		const content = `<html><head></head><div`
		document, err := NewHTMLParser().Parse(content)
		if err != nil {
			t.Fatal(err)
		}
		html, _, body := requireDirectSkeleton(t, document)
		requireSyntheticDocumentElement(t, body, "body", html)
		if len(documentSkeletonElements(document, "div")) != 0 || html.EndPos != len(content) {
			t.Fatalf("discarded incomplete body trigger mutated tree: %#v", html)
		}
	})

	t.Run("implicit-document EOF recovery and reuse", func(t *testing.T) {
		parser := NewHTMLParser()
		if _, err := parser.Parse(`<!doctype html><html><head></head><body>x</body></html>`); err != nil {
			t.Fatal(err)
		}
		document, err := parser.Parse(`<div>x`)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body := requireImplicitDocumentSkeleton(t, document)
		div := requireOneDocumentSkeletonElement(t, document, "div")
		if div.Parent != body || div.EndPos != 6 || head.Parent != html {
			t.Fatalf("Explicit-to-implicit EOF reuse mismatch: html=%#v body=%#v div=%#v", html, body, div)
		}
		document, err = parser.Parse(`<div>x</div>`)
		if err != nil {
			t.Fatal(err)
		}
		html, head, body = requireImplicitDocumentSkeleton(t, document)
		div = requireOneDocumentSkeletonElement(t, document, "div")
		if div.Parent != body || head.Parent != html {
			t.Fatalf("Implicit document reuse mismatch: html=%#v body=%#v div=%#v", html, body, div)
		}
	})
}

func TestParseExplicitDocumentSkeletonScalesNearLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping document-skeleton scaling regression in short mode")
	}
	measure := func(count int) time.Duration {
		content := `<html><head></head>` + strings.Repeat(`<!--late--><meta>`, count) + `<body>` + strings.Repeat(`<span>x</span>`, count) + `</body></html>`
		started := time.Now()
		document, err := NewHTMLParser().Parse(content)
		elapsed := time.Since(started)
		if err != nil {
			t.Fatalf("Parse(%d phase tokens) failed: %v", count, err)
		}
		_, head, body := requireDirectSkeleton(t, document)
		if len(head.Children) != count || len(body.Children) != count {
			t.Fatalf("Parse(%d) routed head/body counts %d/%d", count, len(head.Children), len(body.Children))
		}
		return elapsed
	}
	best := func(count int) time.Duration {
		bestDuration := time.Duration(1<<63 - 1)
		for attempt := 0; attempt < 2; attempt++ {
			if elapsed := measure(count); elapsed < bestDuration {
				bestDuration = elapsed
			}
		}
		return bestDuration
	}
	_ = measure(100)
	small, large := best(1000), best(4000)
	t.Logf("parsed late-head/body document phases at 1000=%s 4000=%s", small, large)
	if large > 12*small && large-small > 50*time.Millisecond {
		t.Fatalf("explicit document phase handling scales superlinearly: 4x input took %.1fx (%s -> %s)", float64(large)/float64(small), small, large)
	}
}
