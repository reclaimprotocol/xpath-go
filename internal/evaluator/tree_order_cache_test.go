package evaluator

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestPrepareAxisOrderBuildsReusableDocumentIndexes(t *testing.T) {
	document := &types.Node{Type: types.DocumentNode, Name: "#document"}
	first := &types.Node{Type: types.ElementNode, Name: "first", Parent: document}
	second := &types.Node{Type: types.ElementNode, Name: "second", Parent: document}
	document.Children = []*types.Node{first, second}

	e := NewEvaluator()
	e.prepareAxisOrder(document)
	if len(e.axisOrder) != 3 || len(e.axisIndex) != 3 || len(e.treeOrder) != 3 {
		t.Fatalf("document indexes = axisOrder:%d axisIndex:%d treeOrder:%d, want 3 each", len(e.axisOrder), len(e.axisIndex), len(e.treeOrder))
	}
	if e.treeOrder[document] >= e.treeOrder[first] || e.treeOrder[first] >= e.treeOrder[second] {
		t.Fatalf("tree order = document:%d first:%d second:%d, want preorder", e.treeOrder[document], e.treeOrder[first], e.treeOrder[second])
	}

	nodes := []*types.Node{second, document, first}
	e.sortNodePointersByTreeOrder(nodes, document)
	if nodes[0] != document || nodes[1] != first || nodes[2] != second {
		t.Fatalf("tree-order sort = %#v, want document/first/second", nodes)
	}
}
