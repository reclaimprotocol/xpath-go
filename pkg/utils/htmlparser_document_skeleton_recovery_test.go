package utils

import (
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

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
