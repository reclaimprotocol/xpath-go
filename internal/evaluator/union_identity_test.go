package evaluator

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestTypedUnionIdentityKeepsDistinctOriginlessRecoveredNodes(t *testing.T) {
	// Recovery can create two structurally similar nodes with no meaningful
	// source locations. A source-coordinate fallback would incorrectly collapse
	// them in a union.
	results := []types.Node{
		{Type: types.ElementNode, Name: "b", StartPos: 0, EndPos: 0},
		{Type: types.ElementNode, Name: "b", StartPos: 0, EndPos: 0},
	}
	if nodeSetIdentityFor(&results[0]) == nodeSetIdentityFor(&results[1]) {
		t.Fatal("distinct origin-less recovered nodes share a union identity")
	}
}
