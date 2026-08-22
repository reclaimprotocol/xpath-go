package evaluator

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestAttributeNodesAreStableWithinEvaluation(t *testing.T) {
	e := NewEvaluator()
	owner := &types.Node{
		Type:           types.ElementNode,
		Name:           "item",
		Attributes:     map[string]string{"id": "one", "class": "keep"},
		AttributeOrder: []string{"id", "class"},
	}

	first := e.getAttributeNodes(owner)
	second := e.getAttributeNodes(owner)
	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("attribute nodes = %d and %d, want two", len(first), len(second))
	}
	if first[0] != second[0] || first[1] != second[1] {
		t.Fatal("repeated attribute-axis access rematerialized nodes")
	}
	if first[0].Parent != owner || first[0].Value != "one" {
		t.Fatalf("cached attribute = %#v, want owner/value preserved", first[0])
	}
}

func TestAttributeNodeCacheDoesNotCrossDocuments(t *testing.T) {
	e := NewEvaluator()
	firstOwner := &types.Node{Type: types.ElementNode, Name: "item", Attributes: map[string]string{"id": "one"}}
	first := e.getAttributeNodes(firstOwner)

	// A fresh document gets a fresh cache entry and must not retain nodes from
	// the previous document. EvaluateProgramWithDocument resets this cache in
	// production; this test exercises the same boundary directly.
	e.attributeNodes = make(map[*types.Node][]*types.Node)
	secondOwner := &types.Node{Type: types.ElementNode, Name: "item", Attributes: map[string]string{"id": "two"}}
	second := e.getAttributeNodes(secondOwner)
	if first[0] == second[0] || second[0].Value != "two" {
		t.Fatalf("attribute cache crossed documents: first=%#v second=%#v", first[0], second[0])
	}
}

func BenchmarkAttributeNodeMaterialization(b *testing.B) {
	owner := &types.Node{
		Type:           types.ElementNode,
		Name:           "item",
		Attributes:     map[string]string{"id": "one", "class": "keep", "data-x": "value"},
		AttributeOrder: []string{"id", "class", "data-x"},
	}
	b.Run("cached", func(b *testing.B) {
		e := NewEvaluator()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if got := e.getAttributeNodes(owner); len(got) != 3 {
				b.Fatalf("got %d attributes, want 3", len(got))
			}
		}
	})
	b.Run("fresh", func(b *testing.B) {
		e := NewEvaluator()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			e.attributeNodes = nil
			if got := e.getAttributeNodes(owner); len(got) != 3 {
				b.Fatalf("got %d attributes, want 3", len(got))
			}
		}
	})
}
