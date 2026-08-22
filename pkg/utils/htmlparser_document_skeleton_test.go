package utils

import (
	"testing"

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
