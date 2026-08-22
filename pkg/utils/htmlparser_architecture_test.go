package utils

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestHTMLParserArchitectureSeparatesResponsibilities(t *testing.T) {
	// These modules are a deliberate package-private boundary. Keeping the
	// parser package-local preserves the recursive tree-builder API while
	// preventing the monolith from growing again.
	required := map[string]string{
		"htmlparser_state.go":                   "type htmlParserState struct",
		"htmlparser_tokenizer.go":               "consumeNamedCharacterReference",
		"htmlparser_foreign.go":                 "parseSVGElement",
		"htmlparser_formatting.go":              "handleCoreAdoption",
		"htmlparser_template.go":                "parseTemplateElement",
		"htmlparser_treebuilder_document.go":    "parseExplicitDocumentHTML",
		"htmlparser_treebuilder_list_select.go": "selectTreeUnwindTarget",
		"htmlparser_treebuilder_table.go":       "parseTableElement",
	}
	for filename, symbol := range required {
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("read %s: %v", filename, err)
		}
		if !strings.Contains(string(content), symbol) {
			t.Fatalf("%s no longer owns %q", filename, symbol)
		}
	}

	mainSource, err := os.ReadFile("htmlparser.go")
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(string(mainSource), "\n") + 1; lines > 2100 {
		t.Fatalf("parser facade grew to %d lines; extract a responsibility-specific module", lines)
	}
	for _, extracted := range []string{
		"consumeNamedCharacterReference", "parseSVGElement", "handleCoreAdoption",
		"parseTemplateElement", "parseExplicitDocumentHTML", "parseTableElement",
	} {
		if strings.Contains(string(mainSource), "func (p *HTMLParser) "+extracted) {
			t.Fatalf("htmlparser.go reclaimed extracted algorithm %s", extracted)
		}
	}
}

func TestHTMLParserFacadeHasOnlyConfigurationAndReplaceableState(t *testing.T) {
	file, err := goparser.ParseFile(token.NewFileSet(), "htmlparser_state.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, spec := range general.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "HTMLParser" {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				t.Fatal("HTMLParser must remain a struct facade")
			}
			if len(structType.Fields.List) != 2 {
				t.Fatalf("HTMLParser has %d fields; expected configuration plus replaceable state", len(structType.Fields.List))
			}
			if got := structType.Fields.List[0].Names[0].Name; got != "scriptingEnabled" {
				t.Fatalf("first facade field = %q, want scriptingEnabled", got)
			}
			if embedded, ok := structType.Fields.List[1].Type.(*ast.Ident); !ok || embedded.Name != "htmlParserState" {
				t.Fatalf("second facade field must embed htmlParserState: %#v", structType.Fields.List[1])
			}
			return
		}
	}
	t.Fatal("HTMLParser declaration not found")
}

func TestTableAndDocumentEndModesAreTyped(t *testing.T) {
	if tableInsertionInactive.active() {
		t.Fatal("inactive table insertion depth reported active")
	}
	if !tableInsertionDepth(1).active() {
		t.Fatal("non-zero table insertion depth reported inactive")
	}
	if listDocumentEndNone == listDocumentEndBody || listDocumentEndBody == listDocumentEndHTML {
		t.Fatalf("document-end modes are not distinct: none=%d body=%d html=%d", listDocumentEndNone, listDocumentEndBody, listDocumentEndHTML)
	}
}

func TestDirtyTextRefreshSkipsCleanSubtreesAndRebuildsChangedAncestors(t *testing.T) {
	parser := NewHTMLParser()
	parser.htmlParserState = newHTMLParserState("")
	root := &types.Node{Type: types.ElementNode, Name: "root"}
	child := &types.Node{Type: types.ElementNode, Name: "child", Parent: root}
	child.Children = []*types.Node{{Type: types.TextNode, Name: "#text", Value: "one", TextContent: "one", Parent: child}}
	root.Children = []*types.Node{child}

	parser.markTextSubtreeDirty(root)
	parser.refreshTreeText(root)
	if root.TextContent != "one" || parser.refreshVisits != 2 {
		t.Fatalf("initial dirty refresh = text %q visits %d, want one and 2", root.TextContent, parser.refreshVisits)
	}
	parser.refreshVisits = 0
	parser.refreshTreeText(root)
	if parser.refreshVisits != 0 {
		t.Fatalf("clean refresh visited %d nodes", parser.refreshVisits)
	}

	child.Children[0].Value = "two"
	child.Children[0].TextContent = "two"
	parser.markTextDirty(child)
	parser.refreshTreeText(root)
	if root.TextContent != "two" || parser.refreshVisits != 2 {
		t.Fatalf("dirty ancestor refresh = text %q visits %d, want two and 2", root.TextContent, parser.refreshVisits)
	}
}

func TestHTMLParserOwnsOpenElementStackAndModeBoundary(t *testing.T) {
	p := NewHTMLParser()
	document, err := p.Parse(`<html><body><div><span>x</span></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	if document == nil || len(p.openElements) != 0 {
		t.Fatalf("parse must leave no live open-element frames: document=%#v stack=%#v", document, p.openElements)
	}
	if p.documentMode != explicitAfterBody {
		t.Fatalf("document mode after a complete explicit document = %v, want after-body", p.documentMode)
	}

	// The stack is a parse boundary, not reusable parser state. A second parse
	// must start empty and end empty even when the first tree used nested frames.
	if _, err := p.Parse(`<section>y</section>`); err != nil {
		t.Fatal(err)
	}
	if len(p.openElements) != 0 {
		t.Fatalf("open-element stack leaked across Parse: %#v", p.openElements)
	}
}

func TestAppendChildIncrementalMaintainsTextAggregate(t *testing.T) {
	parent := &types.Node{Type: types.ElementNode, Name: "div"}
	appendChildIncremental(parent, &types.Node{Type: types.TextNode, Name: "#text", Value: "a", TextContent: "a"})
	appendChildIncremental(parent, &types.Node{Type: types.ElementNode, Name: "span", TextContent: "b"})
	if parent.TextContent != "ab" || len(parent.Children) != 2 {
		t.Fatalf("incremental append lost aggregate: text=%q children=%d", parent.TextContent, len(parent.Children))
	}
	appendChildIncremental(parent, &types.Node{Type: types.TextNode, Name: "#text", Value: "c", TextContent: "c"})
	if parent.TextContent != "abc" || len(parent.Children) != 3 {
		t.Fatalf("incremental append after element mismatch: text=%q children=%d", parent.TextContent, len(parent.Children))
	}
}
